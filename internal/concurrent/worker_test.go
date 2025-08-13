package concurrent

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewWorker(t *testing.T) {
	worker := NewWorker(1, "https://example.com/file.zip")

	if worker.ID != 1 {
		t.Errorf("ID = %d, want 1", worker.ID)
	}

	if worker.URL != "https://example.com/file.zip" {
		t.Errorf("URL = %s, want https://example.com/file.zip", worker.URL)
	}

	if worker.Client == nil {
		t.Fatal("Client should not be nil")
	}

	if worker.Client.Timeout != 30*time.Second {
		t.Errorf("Client timeout = %v, want 30s", worker.Client.Timeout)
	}
}

func TestWorkerDownload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		serverHandler  http.HandlerFunc
		chunk          *ChunkInfo
		wantErr        bool
		expectedErrMsg string
	}{
		{
			name: "Successful download",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				rangeHeader := r.Header.Get("Range")
				if rangeHeader != "bytes=0-999" {
					t.Errorf("unexpected Range header: %s", rangeHeader)
				}

				w.Header().Set("Content-Range", "bytes 0-999/1000")
				w.WriteHeader(http.StatusPartialContent)

				// Write exactly 1000 bytes
				data := make([]byte, 1000)
				for i := range data {
					data[i] = byte(i % 256)
				}
				_, _ = w.Write(data)
			},
			chunk: &ChunkInfo{
				Index:      0,
				Start:      0,
				End:        999,
				Downloaded: 0,
				Complete:   false,
			},
			wantErr: false,
		},
		{
			name: "Resume download",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				rangeHeader := r.Header.Get("Range")
				if rangeHeader != "bytes=500-999" {
					t.Errorf("unexpected Range header for resume: %s", rangeHeader)
				}

				w.Header().Set("Content-Range", "bytes 500-999/1000")
				w.WriteHeader(http.StatusPartialContent)

				// Write remaining 500 bytes
				data := make([]byte, 500)
				_, _ = w.Write(data)
			},
			chunk: &ChunkInfo{
				Index:      0,
				Start:      0,
				End:        999,
				Downloaded: 500, // Already downloaded 500 bytes
				Complete:   false,
			},
			wantErr: false,
		},
		{
			name: "Server error",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusInternalServerError)
			},
			chunk: &ChunkInfo{
				Index:      0,
				Start:      0,
				End:        999,
				Downloaded: 0,
				Complete:   false,
			},
			wantErr:        true,
			expectedErrMsg: "unexpected status code: 500",
		},
		{
			name: "No chunk assigned",
			serverHandler: func(w http.ResponseWriter, r *http.Request) {
				// Should not be called
				t.Error("server should not be called when no chunk assigned")
			},
			chunk:          nil,
			wantErr:        true,
			expectedErrMsg: "no chunk assigned",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(tt.serverHandler)
			defer server.Close()

			progressChan := make(chan Progress, 10)
			errorChan := make(chan error, 10)

			worker := NewWorker(1, server.URL)
			worker.ChunkInfo = tt.chunk
			worker.Progress = progressChan
			worker.Error = errorChan

			ctx := context.Background()
			err := worker.Download(ctx)

			if tt.wantErr {
				if err == nil {
					t.Error("expected error but got nil")
				} else if tt.expectedErrMsg != "" && !containsString(err.Error(), tt.expectedErrMsg) {
					t.Errorf("error = %v, want to contain %s", err, tt.expectedErrMsg)
				}
			} else {
				if err != nil {
					t.Errorf("unexpected error: %v", err)
				}

				if tt.chunk != nil && !tt.chunk.Complete {
					t.Error("chunk should be marked as complete")
				}
			}

			close(progressChan)
			close(errorChan)
		})
	}
}

func TestDownloadChunk(t *testing.T) {
	testData := make([]byte, 5000)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rangeHeader := r.Header.Get("Range")

		// Parse range header
		var start, end int64
		_, _ = fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)

		if start < 0 || end >= int64(len(testData)) || start > end {
			w.WriteHeader(http.StatusRequestedRangeNotSatisfiable)
			return
		}

		w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(testData)))
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(testData[start : end+1])
	}))
	defer server.Close()

	tests := []struct {
		name          string
		chunk         *ChunkInfo
		expectedBytes int64
	}{
		{
			name: "Download first chunk",
			chunk: &ChunkInfo{
				Index:      0,
				Start:      0,
				End:        999,
				Downloaded: 0,
				Complete:   false,
			},
			expectedBytes: 1000,
		},
		{
			name: "Download middle chunk",
			chunk: &ChunkInfo{
				Index:      1,
				Start:      1000,
				End:        2999,
				Downloaded: 0,
				Complete:   false,
			},
			expectedBytes: 2000,
		},
		{
			name: "Download last chunk",
			chunk: &ChunkInfo{
				Index:      2,
				Start:      3000,
				End:        4999,
				Downloaded: 0,
				Complete:   false,
			},
			expectedBytes: 2000,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			progressChan := make(chan Progress, 100)

			worker := NewWorker(1, server.URL)
			worker.ChunkInfo = tt.chunk
			worker.Progress = progressChan

			err := worker.downloadChunk()
			if err != nil {
				t.Fatalf("downloadChunk() error = %v", err)
			}

			if worker.ChunkInfo.Downloaded != tt.expectedBytes {
				t.Errorf("Downloaded = %d, want %d", worker.ChunkInfo.Downloaded, tt.expectedBytes)
			}

			close(progressChan)

			// Verify progress updates were sent
			progressCount := 0
			for range progressChan {
				progressCount++
			}

			if progressCount == 0 {
				t.Error("no progress updates were sent")
			}
		})
	}
}

func TestDownloadChunkWithRetry(t *testing.T) {
	t.Parallel()

	attemptCount := 0

	tests := []struct {
		name          string
		failTimes     int
		expectSuccess bool
		expectedCalls int
	}{
		{
			name:          "Success on first attempt",
			failTimes:     0,
			expectSuccess: true,
			expectedCalls: 1,
		},
		{
			name:          "Success after 2 retries",
			failTimes:     2,
			expectSuccess: true,
			expectedCalls: 3,
		},
		{
			name:          "Fail after max retries",
			failTimes:     5,
			expectSuccess: false,
			expectedCalls: 4, // initial + 3 retries
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			attemptCount = 0
			failCount := 0

			server := httptest.NewServer(
				http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					attemptCount++

					if failCount < tt.failTimes {
						failCount++

						w.WriteHeader(http.StatusInternalServerError)

						return
					}

					w.Header().Set("Content-Range", "bytes 0-99/100")
					w.WriteHeader(http.StatusPartialContent)
					_, _ = w.Write(make([]byte, 100))
				}),
			)
			defer server.Close()

			worker := NewWorker(1, server.URL)
			worker.ChunkInfo = &ChunkInfo{
				Index:      0,
				Start:      0,
				End:        99,
				Downloaded: 0,
				Complete:   false,
			}

			errorChan := make(chan error, 10)
			worker.Error = errorChan

			err := worker.downloadChunk()

			if tt.expectSuccess {
				if err != nil {
					t.Errorf("expected success but got error: %v", err)
				}
			} else {
				if err == nil {
					t.Error("expected error but got success")
				}
			}

			if attemptCount != tt.expectedCalls {
				t.Errorf("attempt count = %d, want %d", attemptCount, tt.expectedCalls)
			}

			close(errorChan)
		})
	}
}

func TestWorkerWithNetworkFailure(t *testing.T) {
	t.Parallel()

	callCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount <= 2 {
			// Simulate network failure by closing connection
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				_ = conn.Close()
			}

			return
		}

		// Success on third attempt
		w.Header().Set("Content-Range", "bytes 0-99/100")
		w.WriteHeader(http.StatusPartialContent)
		_, _ = w.Write(make([]byte, 100))
	}))
	defer server.Close()

	worker := NewWorker(1, server.URL)
	worker.ChunkInfo = &ChunkInfo{
		Index:      0,
		Start:      0,
		End:        99,
		Downloaded: 0,
		Complete:   false,
	}

	ctx := context.Background()

	err := worker.Download(ctx)
	if err != nil {
		t.Errorf("expected success after retries, got error: %v", err)
	}

	if !worker.ChunkInfo.Complete {
		t.Error("chunk should be marked as complete")
	}

	if callCount != 3 {
		t.Errorf("expected 3 calls (initial + 2 retries), got %d", callCount)
	}
}

// Helper function.
func containsString(s, substr string) bool {
	return len(substr) > 0 && len(s) >= len(substr) &&
		(s == substr || s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			len(s) > len(substr) && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}

	return false
}
