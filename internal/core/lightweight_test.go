package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestLightweightDownloader_Download(t *testing.T) {
	tests := []struct {
		name        string
		fileSize    int
		expectedErr bool
		statusCode  int
	}{
		{
			name:        "Tiny file (512B)",
			fileSize:    512,
			expectedErr: false,
			statusCode:  http.StatusOK,
		},
		{
			name:        "Small file (1KB)",
			fileSize:    1024,
			expectedErr: false,
			statusCode:  http.StatusOK,
		},
		{
			name:        "Medium file (100KB)",
			fileSize:    100 * 1024,
			expectedErr: false,
			statusCode:  http.StatusOK,
		},
		{
			name:        "Near threshold (900KB)",
			fileSize:    900 * 1024,
			expectedErr: false,
			statusCode:  http.StatusOK,
		},
		{
			name:        "Server error",
			fileSize:    1024,
			expectedErr: true,
			statusCode:  http.StatusInternalServerError,
		},
		{
			name:        "Not found",
			fileSize:    1024,
			expectedErr: true,
			statusCode:  http.StatusNotFound,
		},
		{
			name:        "Forbidden",
			fileSize:    1024,
			expectedErr: true,
			statusCode:  http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test data
			testData := make([]byte, tt.fileSize)
			for i := range testData {
				testData[i] = byte(i % 256)
			}

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
				if tt.statusCode == http.StatusOK {
					_, _ = w.Write(testData)
				}
			}))
			defer server.Close()

			// Create lightweight downloader
			ld := NewLightweightDownloader()

			// Download to buffer
			var buf bytes.Buffer
			downloaded, err := ld.Download(context.Background(), server.URL, &buf)

			// Check error
			if tt.expectedErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if downloaded != int64(tt.fileSize) {
					t.Errorf("Downloaded size mismatch: got %d, want %d", downloaded, tt.fileSize)
				}
				if !bytes.Equal(buf.Bytes(), testData) {
					t.Error("Downloaded data mismatch")
				}
			}
		})
	}
}

func TestLightweightDownloader_DownloadWithProgress(t *testing.T) {
	fileSize := 100 * 1024 // 100KB
	testData := make([]byte, fileSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	ld := NewLightweightDownloader()

	var progressCalls int
	var lastDownloaded, lastTotal int64

	var buf bytes.Buffer
	downloaded, err := ld.DownloadWithProgress(
		context.Background(),
		server.URL,
		&buf,
		func(downloaded, total int64) {
			progressCalls++
			lastDownloaded = downloaded
			lastTotal = total
		},
	)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if downloaded != int64(fileSize) {
		t.Errorf("Downloaded size mismatch: got %d, want %d", downloaded, fileSize)
	}

	if progressCalls == 0 {
		t.Error("Progress callback was never called")
	}

	if lastDownloaded != int64(fileSize) {
		t.Errorf("Final progress downloaded mismatch: got %d, want %d", lastDownloaded, fileSize)
	}

	if lastTotal != int64(fileSize) {
		t.Errorf("Progress total mismatch: got %d, want %d", lastTotal, fileSize)
	}
}

func TestLightweightDownloader_ContextCancellation(t *testing.T) {
	// Test context cancellation
	fileSize := 1024 * 1024 // 1MB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))
		// Slow write to allow cancellation
		for i := 0; i < fileSize; i += 1024 {
			_, _ = w.Write(make([]byte, 1024))
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	ld := NewLightweightDownloader()
	var buf bytes.Buffer
	_, err := ld.Download(ctx, server.URL, &buf)

	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func TestLightweightDownloader_NoContentLength(t *testing.T) {
	// Test download without Content-Length header
	testData := []byte("test data without content length")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Don't set Content-Length
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	ld := NewLightweightDownloader()
	var buf bytes.Buffer
	downloaded, err := ld.Download(context.Background(), server.URL, &buf)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if downloaded != int64(len(testData)) {
		t.Errorf("Downloaded size mismatch: got %d, want %d", downloaded, len(testData))
	}

	if !bytes.Equal(buf.Bytes(), testData) {
		t.Error("Downloaded data mismatch")
	}
}

func TestShouldUseLightweight(t *testing.T) {
	tests := []struct {
		name          string
		contentLength int64
		expected      bool
	}{
		{
			name:          "Small file (1KB)",
			contentLength: 1024,
			expected:      true,
		},
		{
			name:          "Medium file (500KB)",
			contentLength: 500 * 1024,
			expected:      true,
		},
		{
			name:          "Threshold (1MB)",
			contentLength: 1024 * 1024,
			expected:      false,
		},
		{
			name:          "Large file (10MB)",
			contentLength: 10 * 1024 * 1024,
			expected:      false,
		},
		{
			name:          "Unknown size",
			contentLength: -1,
			expected:      false,
		},
		{
			name:          "Zero size",
			contentLength: 0,
			expected:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldUseLightweight(tt.contentLength)
			if result != tt.expected {
				t.Errorf("shouldUseLightweight(%d) = %v, want %v",
					tt.contentLength, result, tt.expected)
			}
		})
	}
}

// Benchmark lightweight vs standard downloader
func BenchmarkLightweightDownloader(b *testing.B) {
	sizes := []int{
		1024,       // 1KB
		10 * 1024,  // 10KB
		100 * 1024, // 100KB
		500 * 1024, // 500KB
	}

	for _, size := range sizes {
		testData := make([]byte, size)
		for i := range testData {
			testData[i] = byte(i % 256)
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write(testData)
		}))
		defer server.Close()

		b.Run(fmt.Sprintf("Lightweight_%dKB", size/1024), func(b *testing.B) {
			ld := NewLightweightDownloader()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				var buf bytes.Buffer
				_, _ = ld.Download(context.Background(), server.URL, &buf)
			}
		})
	}
}

// Benchmark memory allocation
func BenchmarkLightweightMemoryAllocation(b *testing.B) {
	testData := make([]byte, 100*1024) // 100KB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	b.Run("Lightweight", func(b *testing.B) {
		ld := NewLightweightDownloader()
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			_, _ = ld.Download(context.Background(), server.URL, &buf)
		}
	})

	b.Run("Standard", func(b *testing.B) {
		client := &http.Client{
			Timeout: 30 * time.Second,
		}
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			resp, _ := client.Get(server.URL)
			if resp != nil {
				_, _ = io.Copy(&buf, resp.Body)
				_ = resp.Body.Close()
			}
		}
	})
}
