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
