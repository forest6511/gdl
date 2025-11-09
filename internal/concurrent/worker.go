package concurrent

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
	"github.com/forest6511/gdl/pkg/ratelimit"
)

type Progress struct {
	WorkerID   int
	ChunkIndex int
	Downloaded int64
	Total      int64
	Complete   bool
}

type Worker struct {
	ID          int
	Client      *http.Client
	ChunkInfo   *ChunkInfo
	URL         string
	Progress    chan<- Progress
	Error       chan<- error
	RateLimiter ratelimit.Limiter // Shared rate limiter across all workers
}

// NewWorker creates a new download worker.
func NewWorker(id int, url string) *Worker {
	return &Worker{
		ID:  id,
		URL: url,
		Client: &http.Client{
			Timeout: 30 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:       10,
				IdleConnTimeout:    30 * time.Second,
				DisableCompression: true,
				DisableKeepAlives:  false,
			},
		},
	}
}

// Download starts downloading the assigned chunk.
func (w *Worker) Download(ctx context.Context) error {
	if w.ChunkInfo == nil {
		return gdlerrors.NewDownloadError(
			gdlerrors.CodeValidationError,
			fmt.Sprintf("worker %d: no chunk assigned", w.ID),
		)
	}

	// Try download with retry logic
	err := w.downloadChunk(ctx)
	if err != nil {
		if w.Error != nil {
			// Don't re-wrap DownloadError instances
			if gdlerrors.GetErrorCode(err) != gdlerrors.CodeUnknown {
				w.Error <- err
			} else {
				w.Error <- gdlerrors.WrapError(
					err,
					gdlerrors.CodeNetworkError,
					fmt.Sprintf("worker %d failed chunk %d", w.ID, w.ChunkInfo.Index),
				)
			}
		}

		return err
	}

	// Mark chunk as complete
	w.ChunkInfo.Complete = true

	// Send final progress update
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

// downloadChunk performs the actual chunk download with retry logic.
func (w *Worker) downloadChunk(ctx context.Context) error {
	maxRetries := 3
	baseDelay := 100 * time.Millisecond

	for attempt := 0; attempt <= maxRetries; attempt++ {
		if attempt > 0 {
			// Calculate backoff delay
			// #nosec G115 -- attempt-1 is bounded by maxRetries (3), so uint conversion is safe
			delay := baseDelay * time.Duration(1<<uint(attempt-1))
			if delay > 1*time.Second {
				delay = 1 * time.Second
			}

			time.Sleep(delay)
		}

		// Attempt download
		err := w.performDownload(ctx)
		if err == nil {
			return nil
		}

		// Log retry attempt
		if attempt < maxRetries {
			// Send error notification about retry
			if w.Error != nil {
				w.Error <- gdlerrors.WrapError(
					err,
					gdlerrors.CodeNetworkError,
					fmt.Sprintf("worker %d: chunk %d attempt %d failed, retrying", w.ID, w.ChunkInfo.Index, attempt+1),
				)
			}
		} else {
			return gdlerrors.WrapError(
				err,
				gdlerrors.CodeNetworkError,
				fmt.Sprintf("failed after %d attempts", maxRetries+1),
			)
		}
	}

	return nil
}

// performDownload performs a single download attempt.
func (w *Worker) performDownload(ctx context.Context) error {
	// Create range request
	req, err := http.NewRequest("GET", w.URL, nil)
	if err != nil {
		return gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeNetworkError, "creating request", w.URL)
	}

	// Set range header for partial content
	rangeStart := w.ChunkInfo.Start + w.ChunkInfo.Downloaded
	rangeEnd := w.ChunkInfo.End
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-%d", rangeStart, rangeEnd))

	// Execute request
	resp, err := w.Client.Do(req)
	if err != nil {
		return gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeNetworkError, "executing request", w.URL)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return gdlerrors.FromHTTPStatus(resp.StatusCode, w.URL)
	}

	// Create buffer for reading
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		// Read from response body
		n, err := resp.Body.Read(buffer)
		if n > 0 {
			// Apply rate limiting if a limiter is set
			if w.RateLimiter != nil {
				if rateLimiterErr := w.RateLimiter.Wait(ctx, n); rateLimiterErr != nil {
					if ctx.Err() != nil {
						return gdlerrors.WrapError(rateLimiterErr, gdlerrors.CodeCancelled, "rate limiting cancelled")
					}
					return gdlerrors.WrapError(rateLimiterErr, gdlerrors.CodeTimeout, "rate limiting timeout")
				}
			}

			// Update downloaded bytes
			w.ChunkInfo.Downloaded += int64(n)

			// Send progress update
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
			return gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeNetworkError, "reading response", w.URL)
		}
	}

	// Verify we downloaded the expected amount
	expectedSize := w.ChunkInfo.End - w.ChunkInfo.Start + 1
	if w.ChunkInfo.Downloaded != expectedSize {
		return gdlerrors.NewDownloadError(
			gdlerrors.CodeCorruptedData,
			fmt.Sprintf("size mismatch: downloaded %d, expected %d", w.ChunkInfo.Downloaded, expectedSize),
		)
	}

	return nil
}
