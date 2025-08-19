package concurrent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/forest6511/gdl/pkg/progress"
	"github.com/forest6511/gdl/pkg/ratelimit"
	"github.com/forest6511/gdl/pkg/types"
)

type ConcurrentDownloadManager struct {
	workers     []*Worker
	chunker     *Chunker
	progressMgr *progress.Manager
	wg          sync.WaitGroup
	rateLimiter ratelimit.Limiter
}

// NewConcurrentDownloadManager creates a new concurrent download manager.
func NewConcurrentDownloadManager() *ConcurrentDownloadManager {
	return &ConcurrentDownloadManager{
		progressMgr: progress.NewManager(),
	}
}

// NewConcurrentDownloadManagerWithOptions creates a new concurrent download manager with options.
func NewConcurrentDownloadManagerWithOptions(options *types.DownloadOptions) *ConcurrentDownloadManager {
	manager := &ConcurrentDownloadManager{
		progressMgr: progress.NewManager(),
	}

	// Create rate limiter if MaxRate is specified
	if options != nil && options.MaxRate > 0 {
		manager.rateLimiter = ratelimit.NewBandwidthLimiter(options.MaxRate)
	}

	return manager
}

// Download performs concurrent download of the file.
func (m *ConcurrentDownloadManager) Download(ctx context.Context, url, dest string) error {
	// Get file size first
	fileSize, err := m.getFileSize(url)
	if err != nil {
		return fmt.Errorf("getting file size: %w", err)
	}

	// Check if server supports range requests
	supportsRange, err := m.checkRangeSupport(url)
	if err != nil {
		return fmt.Errorf("checking range support: %w", err)
	}

	if !supportsRange || fileSize <= 0 {
		// Fall back to single-threaded download
		return m.singleDownload(ctx, url, dest)
	}

	// Initialize chunker
	m.chunker = NewChunker(fileSize)
	chunks := m.chunker.GetChunks()

	// Create temporary directory for chunks
	tempDir := dest + ".chunks"
	if err := os.MkdirAll(tempDir, 0o750); err != nil {
		return fmt.Errorf("creating temp directory: %w", err)
	}
	defer m.cleanup(tempDir)

	// Start progress manager
	m.progressMgr.Start()
	defer m.progressMgr.Stop()

	// Create channels for worker communication
	progressChan := make(chan Progress, len(chunks))
	errorChan := make(chan error, len(chunks))

	// Create workers
	m.workers = make([]*Worker, len(chunks))
	for i, chunk := range chunks {
		m.workers[i] = NewWorker(i, url)
		m.workers[i].ChunkInfo = chunk
		m.workers[i].Progress = progressChan
		m.workers[i].Error = errorChan
		m.workers[i].RateLimiter = m.rateLimiter // Share the same rate limiter across all workers
	}

	// Start workers
	m.startWorkers(ctx, tempDir)

	// Monitor progress and errors
	done := make(chan bool)
	go m.monitorProgress(progressChan, errorChan, done, fileSize)

	// Wait for all workers to complete
	m.wg.Wait()
	close(progressChan)
	close(errorChan)
	<-done

	// Check if all chunks completed
	for _, chunk := range chunks {
		if !chunk.Complete {
			return fmt.Errorf("chunk %d incomplete", chunk.Index)
		}
	}

	// Merge chunks into final file
	if err := m.mergeChunks(tempDir, dest, chunks); err != nil {
		return fmt.Errorf("merging chunks: %w", err)
	}

	return nil
}

// startWorkers launches all workers concurrently.
func (m *ConcurrentDownloadManager) startWorkers(ctx context.Context, tempDir string) {
	for i, worker := range m.workers {
		m.wg.Add(1)

		go func(w *Worker, chunkFile string) {
			defer m.wg.Done()

			// Create chunk file
			// #nosec G304 -- chunkFile is constructed internally from validated paths
			file, err := os.Create(chunkFile)
			if err != nil {
				if w.Error != nil {
					w.Error <- fmt.Errorf("creating chunk file: %w", err)
				}

				return
			}
			defer func() { _ = file.Close() }()

			// Wrap the original download to write to file
			originalChunk := w.ChunkInfo

			downloadErr := w.downloadChunkToFile(ctx, file)
			if downloadErr != nil {
				w.ChunkInfo = originalChunk // Restore chunk info
				if w.Error != nil {
					w.Error <- downloadErr
				}
			}
		}(worker, filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i)))
	}
}

// downloadChunkToFile downloads a chunk and writes it to a file.
func (w *Worker) downloadChunkToFile(ctx context.Context, file *os.File) error {
	// Create range request
	req, err := http.NewRequestWithContext(ctx, "GET", w.URL, nil)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}

	// Set range header
	rangeStart := w.ChunkInfo.Start + w.ChunkInfo.Downloaded
	rangeEnd := w.ChunkInfo.End
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd))

	// Execute request
	resp, err := w.Client.Do(req)
	if err != nil {
		return fmt.Errorf("executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Download and write to file
	buffer := make([]byte, 32*1024)
	for {
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			// Apply rate limiting if a limiter is set
			if w.RateLimiter != nil {
				if rateLimiterErr := w.RateLimiter.Wait(ctx, n); rateLimiterErr != nil {
					return fmt.Errorf("rate limiting error: %w", rateLimiterErr)
				}
			}

			if _, writeErr := file.Write(buffer[:n]); writeErr != nil {
				return fmt.Errorf("writing to file: %w", writeErr)
			}

			w.ChunkInfo.Downloaded += int64(n)

			if w.Progress != nil {
				w.Progress <- Progress{
					WorkerID:   w.ID,
					ChunkIndex: w.ChunkInfo.Index,
					Downloaded: w.ChunkInfo.Downloaded,
					Total:      w.ChunkInfo.End - w.ChunkInfo.Start + 1,
					Complete:   false,
				}
			}
		}

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("reading response: %w", err)
		}
	}

	w.ChunkInfo.Complete = true
	if w.Progress != nil {
		w.Progress <- Progress{
			WorkerID:   w.ID,
			ChunkIndex: w.ChunkInfo.Index,
			Downloaded: w.ChunkInfo.Downloaded,
			Total:      w.ChunkInfo.End - w.ChunkInfo.Start + 1,
			Complete:   true,
		}
	}

	return nil
}

// mergeChunks combines all chunk files into the final destination file.
func (m *ConcurrentDownloadManager) mergeChunks(tempDir, dest string, chunks []*ChunkInfo) error {
	// Create destination file
	// #nosec G304 -- dest validated by ValidateDestination() in public API functions
	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("creating destination file: %w", err)
	}
	defer func() { _ = destFile.Close() }()

	// Merge chunks in order
	for i := range chunks {
		chunkPath := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))

		// #nosec G304 -- chunkPath is constructed internally from validated tempDir
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			return fmt.Errorf("opening chunk %d: %w", i, err)
		}

		if _, err := io.Copy(destFile, chunkFile); err != nil {
			_ = chunkFile.Close()
			return fmt.Errorf("copying chunk %d: %w", i, err)
		}

		_ = chunkFile.Close()
	}

	return nil
}

// cleanup removes temporary chunk files.
func (m *ConcurrentDownloadManager) cleanup(tempDir string) {
	if tempDir != "" {
		_ = os.RemoveAll(tempDir)
	}
}

// monitorProgress monitors download progress from all workers.
func (m *ConcurrentDownloadManager) monitorProgress(
	progressChan <-chan Progress,
	errorChan <-chan error,
	done chan<- bool,
	totalSize int64,
) {
	var totalDownloaded int64

	chunkProgress := make(map[int]int64)

	for {
		select {
		case prog, ok := <-progressChan:
			if !ok {
				done <- true
				return
			}

			// Update chunk progress
			oldProgress := chunkProgress[prog.ChunkIndex]
			chunkProgress[prog.ChunkIndex] = prog.Downloaded

			// Calculate total downloaded
			totalDownloaded += (prog.Downloaded - oldProgress)

			// Update progress manager
			m.progressMgr.Update(totalDownloaded, totalSize)

		case err, ok := <-errorChan:
			if ok && err != nil {
				// Log error but continue monitoring
				fmt.Printf("Worker error: %v\n", err)
			}
		}
	}
}

// getFileSize retrieves the size of the file from the server.
func (m *ConcurrentDownloadManager) getFileSize(url string) (int64, error) {
	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Head(url)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("status code: %d", resp.StatusCode)
	}

	return resp.ContentLength, nil
}

// checkRangeSupport checks if the server supports range requests.
func (m *ConcurrentDownloadManager) checkRangeSupport(url string) (bool, error) {
	req, err := http.NewRequest("HEAD", url, nil)
	if err != nil {
		return false, err
	}

	client := &http.Client{Timeout: 5 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return false, err
	}
	defer func() { _ = resp.Body.Close() }()

	acceptRanges := resp.Header.Get("Accept-Ranges")

	return acceptRanges == "bytes", nil
}

// singleDownload performs a simple single-threaded download.
func (m *ConcurrentDownloadManager) singleDownload(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return err
	}

	client := &http.Client{Timeout: 30 * time.Second}

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("status code: %d", resp.StatusCode)
	}

	// #nosec G304 -- dest validated by ValidateDestination() in public API functions
	file, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()

	// Copy with rate limiting if enabled
	if m.rateLimiter != nil {
		buffer := make([]byte, 32*1024)
		for {
			n, readErr := resp.Body.Read(buffer)
			if n > 0 {
				// Apply rate limiting
				if rateLimiterErr := m.rateLimiter.Wait(ctx, n); rateLimiterErr != nil {
					return fmt.Errorf("rate limiting error: %w", rateLimiterErr)
				}

				if _, writeErr := file.Write(buffer[:n]); writeErr != nil {
					return fmt.Errorf("writing to file: %w", writeErr)
				}
			}

			if readErr == io.EOF {
				break
			}
			if readErr != nil {
				return fmt.Errorf("reading response: %w", readErr)
			}
		}
		return nil
	}

	_, err = io.Copy(file, resp.Body)
	return err
}
