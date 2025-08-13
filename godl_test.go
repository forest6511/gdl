package godl

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/forest6511/godl/pkg/validation"
)

func init() {
	// Enable localhost for testing
	validation.SetConfig(validation.TestConfig())
}

func TestDownload(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test content"))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "test.txt")

	stats, err := Download(context.Background(), server.URL, dest)
	if err != nil {
		t.Errorf("Download() error = %v, wantErr %v", err, false)
	}

	if stats == nil {
		t.Error("Expected stats to be returned, got nil")
	}

	if stats != nil && stats.Success != true {
		t.Errorf("Expected download to be successful, got success=%v", stats.Success)
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != "test content" {
		t.Errorf("Expected content 'test content', got %q", string(content))
	}
}

func TestDownloadWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test content"))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "test.txt")

	opts := &Options{
		Timeout: 10 * time.Second,
	}

	stats, err := DownloadWithOptions(context.Background(), server.URL, dest, opts)
	if err != nil {
		t.Errorf("DownloadWithOptions() error = %v, wantErr %v", err, false)
	}

	if stats == nil {
		t.Error("Expected stats to be returned, got nil")
	}
}

func TestDownloadToWriter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test content"))
	}))
	defer server.Close()

	var buf bytes.Buffer

	stats, err := DownloadToWriter(context.Background(), server.URL, &buf)
	if err != nil {
		t.Errorf("DownloadToWriter() error = %v, wantErr %v", err, false)
	}

	if stats == nil {
		t.Error("Expected stats to be returned, got nil")
	}

	if buf.String() != "test content" {
		t.Errorf("Expected content 'test content', got %q", buf.String())
	}
}

func TestDownloadToMemory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("test content"))
	}))
	defer server.Close()

	data, stats, err := DownloadToMemory(context.Background(), server.URL)
	if err != nil {
		t.Errorf("DownloadToMemory() error = %v, wantErr %v", err, false)
	}

	if stats == nil {
		t.Error("Expected stats to be returned, got nil")
	}

	if string(data) != "test content" {
		t.Errorf("Expected content 'test content', got %q", string(data))
	}
}

func TestDownloadWithResume(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Simulate a server that supports range requests
		if r.Header.Get("Range") != "" {
			w.Header().Set("Content-Range", "bytes 5-11/12")
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write([]byte("content"))
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("test content"))
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "resume_test.txt")

	// Create a partial file
	err := os.WriteFile(dest, []byte("test "), 0644)
	if err != nil {
		t.Fatalf("Failed to create partial file: %v", err)
	}

	stats, err := DownloadWithResume(context.Background(), server.URL, dest)
	if err != nil {
		t.Errorf("DownloadWithResume() error = %v, wantErr %v", err, false)
	}

	if stats == nil {
		t.Error("Expected stats to be returned, got nil")
	}

	// Check if file was completed
	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("Failed to read resumed file: %v", err)
	}

	expectedContent := "test content"
	if string(content) != expectedContent {
		t.Errorf("Expected content '%s', got %q", expectedContent, string(content))
	}
}

func TestGetFileInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "12")
		w.Header().Set("Content-Type", "text/plain")
		w.Header().Set("Last-Modified", "Wed, 21 Oct 2015 07:28:00 GMT")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	info, err := GetFileInfo(context.Background(), server.URL)
	if err != nil {
		t.Errorf("GetFileInfo() error = %v, wantErr %v", err, false)
	}

	if info == nil {
		t.Error("Expected info to be returned, got nil")
	}

	if info != nil {
		if info.Size != 12 {
			t.Errorf("Expected size 12, got %d", info.Size)
		}

		if info.ContentType != "text/plain" {
			t.Errorf("Expected content type 'text/plain', got %q", info.ContentType)
		}
	}
}

func TestNewDownloader(t *testing.T) {
	downloader := NewDownloader()
	if downloader == nil {
		t.Error("Expected downloader to be created, got nil")
	}
}

func TestDownloaderBasicFunctionality(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("downloader test"))
	}))
	defer server.Close()

	downloader := NewDownloader()
	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "downloader_test.txt")

	stats, err := downloader.Download(context.Background(), server.URL, dest, nil)
	if err != nil {
		t.Errorf("Downloader.Download() error = %v, wantErr %v", err, false)
	}

	if stats == nil {
		t.Error("Expected stats to be returned, got nil")
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != "downloader test" {
		t.Errorf("Expected content 'downloader test', got %q", string(content))
	}
}

func TestDownloaderToWriter(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("writer test"))
	}))
	defer server.Close()

	downloader := NewDownloader()
	var buf bytes.Buffer

	stats, err := downloader.DownloadToWriter(context.Background(), server.URL, &buf, nil)
	if err != nil {
		t.Errorf("Downloader.DownloadToWriter() error = %v, wantErr %v", err, false)
	}

	if stats == nil {
		t.Error("Expected stats to be returned, got nil")
	}

	if buf.String() != "writer test" {
		t.Errorf("Expected content 'writer test', got %q", buf.String())
	}
}

func TestDownloaderGetFileInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "11")
		w.Header().Set("Content-Type", "text/html")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	downloader := NewDownloader()
	info, err := downloader.GetFileInfo(context.Background(), server.URL)
	if err != nil {
		t.Errorf("Downloader.GetFileInfo() error = %v, wantErr %v", err, false)
	}

	if info == nil {
		t.Error("Expected info to be returned, got nil")
	}

	if info != nil {
		if info.Size != 11 {
			t.Errorf("Expected size 11, got %d", info.Size)
		}

		if info.ContentType != "text/html" {
			t.Errorf("Expected content type 'text/html', got %q", info.ContentType)
		}
	}
}

func TestDownloadErrorHandling(t *testing.T) {
	// Test with an invalid URL
	_, err := Download(context.Background(), "invalid://url", "/tmp/test.txt")
	if err == nil {
		t.Error("Expected error for invalid URL, got nil")
	}

	// Test with context cancellation
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err = Download(ctx, "http://example.com/test.txt", "/tmp/test.txt")
	if err == nil {
		t.Error("Expected error for canceled context, got nil")
	}
}

func TestDownloadOptionsValidation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check User-Agent header
		if r.Header.Get("User-Agent") == "test-agent" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("custom agent"))
		} else {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("default agent"))
		}
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "options_test.txt")

	// Test with custom options
	opts := &Options{
		UserAgent:         "test-agent",
		Timeout:           5 * time.Second,
		OverwriteExisting: true,
		CreateDirs:        true,
	}

	stats, err := DownloadWithOptions(context.Background(), server.URL, dest, opts)
	if err != nil {
		t.Errorf("DownloadWithOptions() error = %v, wantErr %v", err, false)
	}

	if stats == nil {
		t.Error("Expected stats to be returned, got nil")
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != "custom agent" {
		t.Errorf("Expected content 'custom agent', got %q", string(content))
	}
}

func TestProgressCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "13")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("progress test"))
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "progress_test.txt")

	var progressCalls []Progress
	callback := func(p Progress) {
		progressCalls = append(progressCalls, p)
	}

	opts := &Options{
		ProgressCallback:  callback,
		OverwriteExisting: true,
	}

	stats, err := DownloadWithOptions(context.Background(), server.URL, dest, opts)
	if err != nil {
		t.Errorf("DownloadWithOptions() error = %v, wantErr %v", err, false)
	}

	if stats == nil {
		t.Error("Expected stats to be returned, got nil")
	}

	if len(progressCalls) == 0 {
		t.Error("Expected progress callback to be called, but it wasn't")
	}

	// Check that final progress shows completion
	if len(progressCalls) > 0 {
		finalProgress := progressCalls[len(progressCalls)-1]
		if finalProgress.BytesDownloaded == 0 {
			t.Error("Expected final progress to show bytes downloaded")
		}
	}
}
