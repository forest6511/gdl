package concurrent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// benchmarkData creates test data of specified size.
func benchmarkData(size int) []byte {
	data := make([]byte, size)
	for i := range data {
		data[i] = byte(i % 256)
	}

	return data
}

// createBenchmarkServer creates a test server that supports range requests.
func createBenchmarkServer(data []byte, delay time.Duration) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Add artificial delay to simulate network latency
		if delay > 0 {
			time.Sleep(delay)
		}

		switch r.Method {
		case "HEAD":
			w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
			w.Header().Set("Accept-Ranges", "bytes")
			w.WriteHeader(http.StatusOK)

		case "GET":
			rangeHeader := r.Header.Get("Range")
			if rangeHeader == "" {
				// Full file download
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(data)
			} else {
				// Parse range header
				var start, end int64
				_, _ = fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)

				if start < 0 || end >= int64(len(data)) || start > end {
					w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
					return
				}

				w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
				w.Header().Set("Content-Length", fmt.Sprintf("%d", end-start+1))
				w.WriteHeader(http.StatusPartialContent)
				_, _ = w.Write(data[start : end+1])
			}
		}
	}))
}

func BenchmarkSingleDownload(b *testing.B) {
	// Test data sizes: 1MB, 10MB, 100MB
	sizes := []struct {
		name string
		size int
	}{
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			data := benchmarkData(size.size)

			server := createBenchmarkServer(data, 5*time.Millisecond) // 5ms latency
			defer server.Close()

			b.ResetTimer()
			b.SetBytes(int64(size.size))

			for i := 0; i < b.N; i++ {
				tempDir := b.TempDir()
				destFile := filepath.Join(tempDir, fmt.Sprintf("download_%d.dat", i))

				manager := NewConcurrentDownloadManager()
				ctx := context.Background()

				// Force single download by creating server without range support
				singleServer := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						time.Sleep(5 * time.Millisecond) // 5ms latency

						switch r.Method {
						case "HEAD":
							w.Header().Set("Content-Length", fmt.Sprintf("%d", len(data)))
							// No Accept-Ranges header to force single download
							w.WriteHeader(http.StatusOK)
						case "GET":
							w.WriteHeader(http.StatusOK)
							_, _ = w.Write(data)
						}
					}),
				)

				err := manager.Download(ctx, singleServer.URL, destFile)
				singleServer.Close()

				if err != nil {
					b.Fatalf("Download failed: %v", err)
				}

				// Verify file size
				stat, err := os.Stat(destFile)
				if err != nil {
					b.Fatalf("Failed to stat file: %v", err)
				}

				if stat.Size() != int64(size.size) {
					b.Fatalf("File size mismatch: got %d, want %d", stat.Size(), size.size)
				}
			}
		})
	}
}

func BenchmarkConcurrentDownload4(b *testing.B) {
	// Test data sizes with 4 concurrent connections
	sizes := []struct {
		name string
		size int
	}{
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			data := benchmarkData(size.size)

			server := createBenchmarkServer(data, 5*time.Millisecond) // 5ms latency
			defer server.Close()

			b.ResetTimer()
			b.SetBytes(int64(size.size))

			for i := 0; i < b.N; i++ {
				tempDir := b.TempDir()
				destFile := filepath.Join(tempDir, fmt.Sprintf("download_%d.dat", i))

				manager := NewConcurrentDownloadManager()
				ctx := context.Background()

				// Force 4 chunks by creating a custom chunker
				manager.chunker = &Chunker{
					fileSize:   int64(size.size),
					chunkCount: 4,
				}
				manager.chunker.SplitIntoChunks()

				err := manager.Download(ctx, server.URL, destFile)
				if err != nil {
					b.Fatalf("Download failed: %v", err)
				}

				// Verify file size
				stat, err := os.Stat(destFile)
				if err != nil {
					b.Fatalf("Failed to stat file: %v", err)
				}

				if stat.Size() != int64(size.size) {
					b.Fatalf("File size mismatch: got %d, want %d", stat.Size(), size.size)
				}
			}
		})
	}
}

func BenchmarkConcurrentDownload8(b *testing.B) {
	// Test data sizes with 8 concurrent connections
	sizes := []struct {
		name string
		size int
	}{
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			data := benchmarkData(size.size)

			server := createBenchmarkServer(data, 5*time.Millisecond) // 5ms latency
			defer server.Close()

			b.ResetTimer()
			b.SetBytes(int64(size.size))

			for i := 0; i < b.N; i++ {
				tempDir := b.TempDir()
				destFile := filepath.Join(tempDir, fmt.Sprintf("download_%d.dat", i))

				manager := NewConcurrentDownloadManager()
				ctx := context.Background()

				// Force 8 chunks by creating a custom chunker
				manager.chunker = &Chunker{
					fileSize:   int64(size.size),
					chunkCount: 8,
				}
				manager.chunker.SplitIntoChunks()

				err := manager.Download(ctx, server.URL, destFile)
				if err != nil {
					b.Fatalf("Download failed: %v", err)
				}

				// Verify file size
				stat, err := os.Stat(destFile)
				if err != nil {
					b.Fatalf("Failed to stat file: %v", err)
				}

				if stat.Size() != int64(size.size) {
					b.Fatalf("File size mismatch: got %d, want %d", stat.Size(), size.size)
				}
			}
		})
	}
}

// BenchmarkConcurrentDownload16 tests with 16 concurrent connections.
func BenchmarkConcurrentDownload16(b *testing.B) {
	// Test data sizes with 16 concurrent connections
	sizes := []struct {
		name string
		size int
	}{
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"100MB", 100 * 1024 * 1024},
	}

	for _, size := range sizes {
		b.Run(size.name, func(b *testing.B) {
			data := benchmarkData(size.size)

			server := createBenchmarkServer(data, 5*time.Millisecond) // 5ms latency
			defer server.Close()

			b.ResetTimer()
			b.SetBytes(int64(size.size))

			for i := 0; i < b.N; i++ {
				tempDir := b.TempDir()
				destFile := filepath.Join(tempDir, fmt.Sprintf("download_%d.dat", i))

				manager := NewConcurrentDownloadManager()
				ctx := context.Background()

				// Force 16 chunks by creating a custom chunker
				manager.chunker = &Chunker{
					fileSize:   int64(size.size),
					chunkCount: 16,
				}
				manager.chunker.SplitIntoChunks()

				err := manager.Download(ctx, server.URL, destFile)
				if err != nil {
					b.Fatalf("Download failed: %v", err)
				}

				// Verify file size
				stat, err := os.Stat(destFile)
				if err != nil {
					b.Fatalf("Failed to stat file: %v", err)
				}

				if stat.Size() != int64(size.size) {
					b.Fatalf("File size mismatch: got %d, want %d", stat.Size(), size.size)
				}
			}
		})
	}
}

// BenchmarkOptimalChunkCalculation benchmarks the chunk calculation algorithm.
func BenchmarkOptimalChunkCalculation(b *testing.B) {
	fileSizes := []int64{
		1024 * 1024,        // 1MB
		10 * 1024 * 1024,   // 10MB
		100 * 1024 * 1024,  // 100MB
		1024 * 1024 * 1024, // 1GB
	}

	for _, size := range fileSizes {
		b.Run(fmt.Sprintf("%dMB", size/(1024*1024)), func(b *testing.B) {
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				chunker := NewChunker(size)
				_ = chunker.GetChunks()
			}
		})
	}
}

// BenchmarkChunkMerging benchmarks the chunk merging process.
func BenchmarkChunkMerging(b *testing.B) {
	chunkCounts := []int{4, 8, 16, 32}
	chunkSize := 1024 * 1024 // 1MB per chunk

	for _, count := range chunkCounts {
		b.Run(fmt.Sprintf("%dChunks", count), func(b *testing.B) {
			// Prepare test data
			tempDir := b.TempDir()
			chunks := make([]*ChunkInfo, count)
			totalSize := int64(count * chunkSize)

			// Create chunk files
			chunkData := benchmarkData(chunkSize)
			for i := 0; i < count; i++ {
				chunks[i] = &ChunkInfo{
					Index: i,
					Start: int64(i * chunkSize),
					End:   int64((i+1)*chunkSize - 1),
				}

				chunkFile := filepath.Join(tempDir, fmt.Sprintf("chunk_%d", i))
				if err := os.WriteFile(chunkFile, chunkData, 0o644); err != nil {
					b.Fatalf("Failed to create chunk file: %v", err)
				}
			}

			manager := NewConcurrentDownloadManager()

			b.ResetTimer()
			b.SetBytes(totalSize)

			for i := 0; i < b.N; i++ {
				destFile := filepath.Join(tempDir, fmt.Sprintf("merged_%d.dat", i))

				err := manager.mergeChunks(tempDir, destFile, chunks)
				if err != nil {
					b.Fatalf("Merge failed: %v", err)
				}

				// Cleanup for next iteration
				_ = os.Remove(destFile)
			}
		})
	}
}

// BenchmarkProgressTracking benchmarks the progress tracking overhead.
func BenchmarkProgressTracking(b *testing.B) {
	manager := NewConcurrentDownloadManager()

	manager.progressMgr.Start()
	defer manager.progressMgr.Stop()

	totalSize := int64(100 * 1024 * 1024) // 100MB

	b.ResetTimer()
	b.SetBytes(totalSize)

	for i := 0; i < b.N; i++ {
		// Simulate progress updates throughout download
		for downloaded := int64(0); downloaded < totalSize; downloaded += 1024 * 1024 {
			manager.progressMgr.Update(downloaded, totalSize)
			_ = manager.progressMgr.GetProgress()
		}
	}
}

// BenchmarkWorkerCreation benchmarks worker creation overhead.
func BenchmarkWorkerCreation(b *testing.B) {
	url := "https://example.com/file.zip"

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		worker := NewWorker(i, url)
		_ = worker
	}
}

// BenchmarkNetworkLatencyComparison compares performance under different network conditions.
func BenchmarkNetworkLatencyComparison(b *testing.B) {
	latencies := []struct {
		name  string
		delay time.Duration
	}{
		{"NoLatency", 0},
		{"1ms", 1 * time.Millisecond},
		{"5ms", 5 * time.Millisecond},
		{"10ms", 10 * time.Millisecond},
		{"50ms", 50 * time.Millisecond},
	}

	fileSize := 10 * 1024 * 1024 // 10MB
	data := benchmarkData(fileSize)

	for _, latency := range latencies {
		b.Run(fmt.Sprintf("Single_%s", latency.name), func(b *testing.B) {
			b.ResetTimer()
			b.SetBytes(int64(fileSize))

			for i := 0; i < b.N; i++ {
				server := httptest.NewServer(
					http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
						time.Sleep(latency.delay)
						w.WriteHeader(http.StatusOK)
						_, _ = w.Write(data)
					}),
				)

				tempDir := b.TempDir()
				destFile := filepath.Join(tempDir, "download.dat")

				manager := NewConcurrentDownloadManager()
				ctx := context.Background()

				err := manager.singleDownload(ctx, server.URL, destFile)
				server.Close()

				if err != nil {
					b.Fatalf("Download failed: %v", err)
				}
			}
		})

		b.Run(fmt.Sprintf("Concurrent8_%s", latency.name), func(b *testing.B) {
			server := createBenchmarkServer(data, latency.delay)
			defer server.Close()

			b.ResetTimer()
			b.SetBytes(int64(fileSize))

			for i := 0; i < b.N; i++ {
				tempDir := b.TempDir()
				destFile := filepath.Join(tempDir, "download.dat")

				manager := NewConcurrentDownloadManager()
				ctx := context.Background()

				// Force 8 chunks
				manager.chunker = &Chunker{
					fileSize:   int64(fileSize),
					chunkCount: 8,
				}
				manager.chunker.SplitIntoChunks()

				err := manager.Download(ctx, server.URL, destFile)
				if err != nil {
					b.Fatalf("Download failed: %v", err)
				}
			}
		})
	}
}
