package core

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/forest6511/gdl/pkg/types"
)

// BenchmarkDownloadWithOptimization tests download performance with smart defaults
func BenchmarkDownloadWithOptimization(b *testing.B) {
	sizes := []struct {
		name string
		size int64
	}{
		{"1KB", 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"50MB", 50 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			// Create test content
			content := make([]byte, tc.size)
			for i := range content {
				content[i] = byte(i % 256)
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
				w.Header().Set("Accept-Ranges", "bytes")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(content)
			}))
			defer server.Close()

			tempDir := b.TempDir()
			downloader := NewDownloader()
			ctx := context.Background()

			b.ResetTimer()
			b.SetBytes(tc.size)

			for i := 0; i < b.N; i++ {
				tempFile := filepath.Join(tempDir, fmt.Sprintf("bench_%d.bin", i))

				options := &types.DownloadOptions{
					MaxConcurrency: 4, // Will be optimized based on size
					ChunkSize:      32 * 1024,
				}

				_, err := downloader.Download(ctx, server.URL, tempFile, options)
				if err != nil {
					b.Fatalf("Download failed: %v", err)
				}

				_ = os.Remove(tempFile)
			}
		})
	}
}

// BenchmarkDownloadWithoutOptimization tests download performance without smart defaults
func BenchmarkDownloadWithoutOptimization(b *testing.B) {
	sizes := []struct {
		name string
		size int64
	}{
		{"1KB", 1024},
		{"100KB", 100 * 1024},
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
		{"50MB", 50 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			// Create test content
			content := make([]byte, tc.size)
			for i := range content {
				content[i] = byte(i % 256)
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
				w.Header().Set("Accept-Ranges", "bytes")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write(content)
			}))
			defer server.Close()

			tempDir := b.TempDir()
			downloader := NewDownloader()
			ctx := context.Background()

			b.ResetTimer()
			b.SetBytes(tc.size)

			for i := 0; i < b.N; i++ {
				tempFile := filepath.Join(tempDir, fmt.Sprintf("bench_%d.bin", i))

				// Force fixed concurrency (no optimization)
				options := &types.DownloadOptions{
					MaxConcurrency: 8,          // Always use 8 connections
					ChunkSize:      512 * 1024, // Always use 512KB chunks
				}

				_, err := downloader.Download(ctx, server.URL, tempFile, options)
				if err != nil {
					b.Fatalf("Download failed: %v", err)
				}

				_ = os.Remove(tempFile)
			}
		})
	}
}

// BenchmarkHTTPClientOptimization compares optimized vs standard HTTP client
func BenchmarkHTTPClientOptimization(b *testing.B) {
	content := make([]byte, 10*1024*1024) // 10MB
	for i := range content {
		content[i] = byte(i % 256)
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
		_, _ = w.Write(content)
	}))
	defer server.Close()

	b.Run("OptimizedClient", func(b *testing.B) {
		downloader := NewDownloader() // Uses optimized client
		ctx := context.Background()
		tempDir := b.TempDir()

		b.SetBytes(int64(len(content)))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			tempFile := filepath.Join(tempDir, fmt.Sprintf("opt_%d.bin", i))
			_, err := downloader.Download(ctx, server.URL, tempFile, nil)
			if err != nil {
				b.Fatalf("Download failed: %v", err)
			}
			_ = os.Remove(tempFile)
		}
	})

	b.Run("StandardClient", func(b *testing.B) {
		// Create downloader with standard HTTP client
		standardClient := &http.Client{
			Timeout: 30 * time.Second,
		}
		downloader := NewDownloaderWithClient(standardClient)
		ctx := context.Background()
		tempDir := b.TempDir()

		b.SetBytes(int64(len(content)))
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			tempFile := filepath.Join(tempDir, fmt.Sprintf("std_%d.bin", i))
			_, err := downloader.Download(ctx, server.URL, tempFile, nil)
			if err != nil {
				b.Fatalf("Download failed: %v", err)
			}
			_ = os.Remove(tempFile)
		}
	})
}

// BenchmarkDownloadToWriter tests direct writer performance
func BenchmarkDownloadToWriter(b *testing.B) {
	sizes := []struct {
		name string
		size int64
	}{
		{"1MB", 1024 * 1024},
		{"10MB", 10 * 1024 * 1024},
	}

	for _, tc := range sizes {
		b.Run(tc.name, func(b *testing.B) {
			content := make([]byte, tc.size)
			for i := range content {
				content[i] = byte(i % 256)
			}

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Length", fmt.Sprintf("%d", len(content)))
				_, _ = w.Write(content)
			}))
			defer server.Close()

			downloader := NewDownloader()
			ctx := context.Background()

			b.SetBytes(tc.size)
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				_, err := downloader.DownloadToWriter(ctx, server.URL, io.Discard, nil)
				if err != nil {
					b.Fatalf("Download failed: %v", err)
				}
			}
		})
	}
}
