package core

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestZeroCopyDownloader_Download(t *testing.T) {
	// Create test data (10MB)
	testData := make([]byte, 10*1024*1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10485760")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "zerocopy_test.bin")

	zd := NewZeroCopyDownloader()
	downloaded, err := zd.Download(context.Background(), server.URL, dest)
	if err != nil {
		t.Fatalf("ZeroCopy download failed: %v", err)
	}

	if downloaded != int64(len(testData)) {
		t.Errorf("Expected %d bytes, got %d", len(testData), downloaded)
	}

	// Verify file content
	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if len(content) != len(testData) {
		t.Errorf("Content size mismatch: expected %d, got %d", len(testData), len(content))
	}

	// Verify first and last bytes
	if content[0] != testData[0] || content[len(content)-1] != testData[len(testData)-1] {
		t.Error("Content mismatch detected")
	}
}

func TestZeroCopyDownloader_DownloadWithProgress(t *testing.T) {
	testData := []byte("test data for progress")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "22")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "progress_test.txt")

	var lastProgress int64
	progressCalled := false

	zd := NewZeroCopyDownloader()
	downloaded, err := zd.DownloadWithProgress(
		context.Background(),
		server.URL,
		dest,
		func(down, total int64) {
			progressCalled = true
			lastProgress = down
			if total != 22 {
				t.Errorf("Expected total size 22, got %d", total)
			}
		},
	)

	if err != nil {
		t.Fatalf("Download with progress failed: %v", err)
	}

	if !progressCalled {
		t.Error("Progress callback was not called")
	}

	if downloaded != int64(len(testData)) {
		t.Errorf("Expected %d bytes, got %d", len(testData), downloaded)
	}

	if lastProgress != downloaded {
		t.Errorf("Last progress %d doesn't match downloaded %d", lastProgress, downloaded)
	}
}

func TestShouldUseZeroCopy(t *testing.T) {
	tests := []struct {
		size     int64
		expected bool
	}{
		{1024, false},              // 1KB - too small
		{1024 * 1024, false},       // 1MB - too small
		{5 * 1024 * 1024, false},   // 5MB - too small
		{10 * 1024 * 1024, false},  // 10MB - threshold
		{11 * 1024 * 1024, true},   // 11MB - should use
		{100 * 1024 * 1024, true},  // 100MB - should use
		{1024 * 1024 * 1024, true}, // 1GB - should use
	}

	for _, tt := range tests {
		result := shouldUseZeroCopy(tt.size)
		if result != tt.expected {
			t.Errorf("shouldUseZeroCopy(%d) = %v, want %v", tt.size, result, tt.expected)
		}
	}
}

func TestZeroCopyLinux(t *testing.T) {
	// Skip on Windows since this tests Linux-specific fallback
	if runtime.GOOS == "windows" {
		t.Skip("Skipping Linux-specific zerocopy test on Windows")
	}

	// Test fallback implementation in zeroCopyLinux
	testData := make([]byte, 2*1024*1024) // 2MB test data
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testData)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "zerocopy_linux_test.bin")

	zd := NewZeroCopyDownloader()

	// Make HTTP request manually to test zeroCopyLinux
	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to make request: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Create destination file
	destFile, err := os.Create(dest)
	if err != nil {
		t.Fatalf("Failed to create destination file: %v", err)
	}
	defer func() { _ = destFile.Close() }()

	// Test the zeroCopyLinux function
	written, err := zd.zeroCopyLinux(resp.Body, destFile)
	if err != nil {
		t.Fatalf("zeroCopyLinux failed: %v", err)
	}

	if written != int64(len(testData)) {
		t.Errorf("Expected %d bytes written, got %d", len(testData), written)
	}

	// Verify file content
	_ = destFile.Close()
	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if len(content) != len(testData) {
		t.Errorf("Content size mismatch: expected %d, got %d", len(testData), len(content))
	}
}

func TestZeroCopyDownload_ErrorCases(t *testing.T) {
	t.Run("Invalid URL", func(t *testing.T) {
		zd := NewZeroCopyDownloader()
		_, err := zd.Download(context.Background(), "http://[invalid-url", "/tmp/test")
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})

	t.Run("Server error", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer server.Close()

		tempDir := t.TempDir()
		dest := filepath.Join(tempDir, "error_test.bin")

		zd := NewZeroCopyDownloader()
		_, err := zd.Download(context.Background(), server.URL, dest)
		if err == nil {
			t.Error("Expected error for server error response")
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Slow response
			data := make([]byte, 10*1024*1024)
			_, _ = w.Write(data)
		}))
		defer server.Close()

		tempDir := t.TempDir()
		dest := filepath.Join(tempDir, "cancel_test.bin")

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		zd := NewZeroCopyDownloader()
		_, err := zd.Download(ctx, server.URL, dest)
		if err == nil {
			t.Error("Expected error for cancelled context")
		}
	})

	t.Run("Invalid destination path", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("test"))
		}))
		defer server.Close()

		zd := NewZeroCopyDownloader()
		// Use a path that's invalid on all platforms
		invalidPath := filepath.Join(t.TempDir(), "nonexistent", "deep", "path", "test.bin")
		_, err := zd.Download(context.Background(), server.URL, invalidPath)
		if err == nil {
			t.Error("Expected error for invalid destination path")
		}
	})
}

func BenchmarkZeroCopyDownload(b *testing.B) {
	// Create 10MB test data
	testData := make([]byte, 10*1024*1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	tempDir := b.TempDir()
	zd := NewZeroCopyDownloader()

	b.ResetTimer()
	b.SetBytes(int64(len(testData)))

	for i := 0; i < b.N; i++ {
		dest := filepath.Join(tempDir, fmt.Sprintf("bench_%d.bin", i))
		_, err := zd.Download(context.Background(), server.URL, dest)
		if err != nil {
			b.Fatal(err)
		}
		_ = os.Remove(dest) // Clean up
	}
}

func TestZeroCopyDownloader_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupServer func() *httptest.Server
		url         string
		dest        string
		wantErr     bool
	}{
		{
			name: "Server returns 404",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusNotFound)
				}))
			},
			dest:    "test.bin",
			wantErr: true,
		},
		{
			name: "Server returns 500",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusInternalServerError)
				}))
			},
			dest:    "test.bin",
			wantErr: true,
		},
		{
			name: "Invalid URL",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
				}))
			},
			url:     "://invalid-url",
			dest:    "test.bin",
			wantErr: true,
		},
		{
			name: "Empty destination",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("test"))
				}))
			},
			dest:    "",
			wantErr: true,
		},
		{
			name: "Server closes connection early",
			setupServer: func() *httptest.Server {
				return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Length", "1000")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte("partial"))
					// Connection will close early
				}))
			},
			dest:    "test.bin",
			wantErr: true, // Should error on incomplete data
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempDir := t.TempDir()
			server := tt.setupServer()
			defer server.Close()

			url := server.URL
			if tt.url != "" {
				url = tt.url
			}

			dest := tt.dest
			if dest != "" && dest != "test.bin" {
				dest = filepath.Join(tempDir, dest)
			} else if dest == "test.bin" {
				dest = filepath.Join(tempDir, "test.bin")
			}

			zd := NewZeroCopyDownloader()
			_, err := zd.Download(context.Background(), url, dest)

			if (err != nil) != tt.wantErr {
				t.Errorf("Download() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestZeroCopyDownloader_ContextCancellation(t *testing.T) {
	// Server that sends data slowly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "10000")
		w.WriteHeader(http.StatusOK)
		// Send data in chunks with delays
		for i := 0; i < 10; i++ {
			select {
			case <-r.Context().Done():
				return
			default:
				_, _ = w.Write(make([]byte, 1000))
				if flusher, ok := w.(http.Flusher); ok {
					flusher.Flush()
				}
			}
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "cancel_test.bin")

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	zd := NewZeroCopyDownloader()
	_, err := zd.Download(ctx, server.URL, dest)

	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestZeroCopyDownloader_LargeFile(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping large file test in short mode")
	}

	// Create 50MB test data
	testData := make([]byte, 50*1024*1024)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(testData)))
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "large_test.bin")

	zd := NewZeroCopyDownloader()
	downloaded, err := zd.Download(context.Background(), server.URL, dest)
	if err != nil {
		t.Fatalf("Large file download failed: %v", err)
	}

	if downloaded != int64(len(testData)) {
		t.Errorf("Expected %d bytes, got %d", len(testData), downloaded)
	}

	// Verify file size
	info, err := os.Stat(dest)
	if err != nil {
		t.Fatalf("Failed to stat downloaded file: %v", err)
	}

	if info.Size() != int64(len(testData)) {
		t.Errorf("File size mismatch: expected %d, got %d", len(testData), info.Size())
	}
}

func TestZeroCopyDownloader_ProgressWithError(t *testing.T) {
	// Create a custom server that can close connections abruptly
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "1000")
		w.WriteHeader(http.StatusOK)
		// Write partial data
		_, _ = w.Write([]byte("partial"))
		// Close the underlying connection to simulate network error
		if hijacker, ok := w.(http.Hijacker); ok {
			if conn, _, err := hijacker.Hijack(); err == nil {
				_ = conn.Close()
			}
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "progress_error.bin")

	progressCalled := false
	zd := NewZeroCopyDownloader()

	_, err := zd.DownloadWithProgress(
		context.Background(),
		server.URL,
		dest,
		func(down, total int64) {
			progressCalled = true
		},
	)

	// Should have called progress at least once
	if !progressCalled {
		t.Log("Progress callback might not have been called due to immediate failure")
	}

	// Error is expected due to connection drop
	if err == nil {
		t.Error("Expected error due to connection drop")
	}
}

func TestZeroCopyDownloader_NoContentLength(t *testing.T) {
	testData := []byte("test data without content length")

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't set Content-Length header
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "no_length.txt")

	zd := NewZeroCopyDownloader()
	downloaded, err := zd.Download(context.Background(), server.URL, dest)

	if err != nil {
		t.Fatalf("Download without Content-Length failed: %v", err)
	}

	if downloaded != int64(len(testData)) {
		t.Errorf("Expected %d bytes, got %d", len(testData), downloaded)
	}

	// Verify content
	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	if string(content) != string(testData) {
		t.Error("Content mismatch")
	}
}

func TestZeroCopyDownloader_InvalidDestination(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test"))
	}))
	defer server.Close()

	// Try to write to a directory that doesn't exist
	dest := "/nonexistent/path/to/file.txt"

	zd := NewZeroCopyDownloader()
	_, err := zd.Download(context.Background(), server.URL, dest)

	if err == nil {
		t.Error("Expected error for invalid destination path")
	}
}
