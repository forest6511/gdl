package core

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/forest6511/gdl/pkg/types"
)

func TestPerformZeroCopyDownload_Integration(t *testing.T) {
	// Skip on Windows as it doesn't support zero-copy
	if runtime.GOOS == "windows" {
		t.Skip("Zero-copy not supported on Windows")
	}

	// Test various file sizes
	testCases := []struct {
		name     string
		fileSize int
	}{
		{"Small file (10MB)", 10 * 1024 * 1024},
		{"Medium file (50MB)", 50 * 1024 * 1024},
		{"Large file (100MB)", 100 * 1024 * 1024},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Skip large tests in short mode
			if testing.Short() && tc.fileSize > 10*1024*1024 {
				t.Skip("Skipping large file test in short mode")
			}

			// Create test data
			testData := make([]byte, tc.fileSize)
			for i := range testData {
				testData[i] = byte(i % 256)
			}

			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", tc.fileSize))
				w.Header().Set("Accept-Ranges", "bytes")
				_, _ = w.Write(testData)
			}))
			defer server.Close()

			// Create downloader
			d := NewDownloader()

			// Test zero-copy download
			tempDir := t.TempDir()
			dest := filepath.Join(tempDir, "test_zerocopy.bin")

			options := &types.DownloadOptions{
				ChunkSize: 1024 * 1024, // 1MB chunks
			}

			stats, err := d.performZeroCopyDownload(
				context.Background(),
				server.URL,
				dest,
				options,
			)

			if err != nil {
				t.Fatalf("Zero-copy download failed: %v", err)
			}

			if stats.BytesDownloaded != int64(tc.fileSize) {
				t.Errorf("Downloaded size mismatch: got %d, want %d",
					stats.BytesDownloaded, tc.fileSize)
			}

			// Verify file contents
			downloaded, err := os.ReadFile(dest)
			if err != nil {
				t.Fatalf("Failed to read downloaded file: %v", err)
			}

			if !bytes.Equal(downloaded, testData) {
				t.Error("Downloaded data mismatch")
			}
		})
	}
}

func TestZeroCopyWithContext(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Zero-copy not supported on Windows")
	}

	// Test context cancellation
	fileSize := 100 * 1024 * 1024 // 100MB
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))
		// Slow response to allow cancellation
		for i := 0; i < fileSize; i += 1024 * 1024 {
			_, _ = w.Write(make([]byte, 1024*1024))
			// Add delay to simulate slow transfer
			time.Sleep(10 * time.Millisecond)
		}
	}))
	defer server.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	d := NewDownloader()
	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "test_cancel.bin")

	options := &types.DownloadOptions{
		ChunkSize: 1024 * 1024,
	}

	_, err := d.performZeroCopyDownload(ctx, server.URL, dest, options)

	if err == nil {
		t.Error("Expected context cancellation error")
	}
}

func BenchmarkZeroCopy_vs_StandardCopy(b *testing.B) {
	if runtime.GOOS == "windows" {
		b.Skip("Zero-copy not supported on Windows")
	}

	fileSize := 10 * 1024 * 1024 // 10MB
	testData := make([]byte, fileSize)
	for i := range testData {
		testData[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write(testData)
	}))
	defer server.Close()

	b.Run("ZeroCopy", func(b *testing.B) {
		d := NewDownloader()
		tempDir := b.TempDir()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dest := filepath.Join(tempDir, fmt.Sprintf("zerocopy_%d.bin", i))
			options := &types.DownloadOptions{
				ChunkSize: 1024 * 1024,
			}
			_, _ = d.performZeroCopyDownload(
				context.Background(),
				server.URL,
				dest,
				options,
			)
			_ = os.Remove(dest)
		}
	})

	b.Run("StandardCopy", func(b *testing.B) {
		tempDir := b.TempDir()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			dest := filepath.Join(tempDir, fmt.Sprintf("standard_%d.bin", i))

			resp, err := http.Get(server.URL)
			if err != nil {
				continue
			}

			file, err := os.Create(dest)
			if err != nil {
				_ = resp.Body.Close()
				continue
			}

			_, _ = io.Copy(file, resp.Body)
			_ = file.Close()
			_ = resp.Body.Close()
			_ = os.Remove(dest)
		}
	})
}
