package gdl

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/forest6511/gdl/pkg/events"
	"github.com/forest6511/gdl/pkg/middleware"
	"github.com/forest6511/gdl/pkg/types"
	"github.com/forest6511/gdl/pkg/validation"
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

func TestDownloaderMethods(t *testing.T) {
	downloader := NewDownloader()

	t.Run("UsePlugin", func(t *testing.T) {
		// Create a mock plugin
		mockPlugin := &mockTestPlugin{
			name:    "test-plugin",
			version: "1.0.0",
		}

		err := downloader.UsePlugin(mockPlugin)
		if err != nil {
			t.Errorf("UsePlugin() error = %v, wantErr false", err)
		}
	})

	t.Run("RegisterProtocol", func(t *testing.T) {
		mockProtocol := &mockTestProtocol{
			scheme: "test",
		}

		_ = downloader.RegisterProtocol(mockProtocol)
		// No error expected from RegisterProtocol
	})

	t.Run("SetStorageBackend", func(t *testing.T) {
		mockBackend := &mockTestStorageBackend{
			name: "test-storage",
		}

		err := downloader.SetStorageBackend("test", mockBackend)
		if err != nil {
			t.Errorf("SetStorageBackend() error = %v, wantErr false", err)
		}
	})
}

// Mock plugin for testing
type mockTestPlugin struct {
	name    string
	version string
}

func (m *mockTestPlugin) Name() string                                           { return m.name }
func (m *mockTestPlugin) Version() string                                        { return m.version }
func (m *mockTestPlugin) Init(config map[string]interface{}) error               { return nil }
func (m *mockTestPlugin) Close() error                                           { return nil }
func (m *mockTestPlugin) ValidateAccess(operation string, resource string) error { return nil }

// Mock protocol for testing
type mockTestProtocol struct {
	scheme string
}

func (m *mockTestProtocol) Scheme() string            { return m.scheme }
func (m *mockTestProtocol) CanHandle(url string) bool { return true }
func (m *mockTestProtocol) Download(ctx context.Context, url string, options *types.DownloadOptions) (*types.DownloadStats, error) {
	return &types.DownloadStats{Success: true}, nil
}

// Mock storage backend for testing
type mockTestStorageBackend struct {
	name string
}

func (m *mockTestStorageBackend) Init(config map[string]interface{}) error { return nil }
func (m *mockTestStorageBackend) Save(ctx context.Context, key string, data io.Reader) error {
	return nil
}
func (m *mockTestStorageBackend) Load(ctx context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(strings.NewReader("")), nil
}
func (m *mockTestStorageBackend) Delete(ctx context.Context, key string) error { return nil }
func (m *mockTestStorageBackend) Exists(ctx context.Context, key string) (bool, error) {
	return false, nil
}
func (m *mockTestStorageBackend) List(ctx context.Context, prefix string) ([]string, error) {
	return nil, nil
}
func (m *mockTestStorageBackend) Close() error { return nil }

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

func TestDownloadWithMaxRate(t *testing.T) {
	// Create a server that serves data in small chunks over time
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "4096") // 4KB
		w.WriteHeader(http.StatusOK)

		// Write data in chunks with small delays to simulate network latency
		chunkSize := 1024 // 1KB chunks
		data := make([]byte, chunkSize)
		for i := range data {
			data[i] = byte(i % 256)
		}

		// Write 4 chunks of 1KB each
		for i := 0; i < 4; i++ {
			_, _ = w.Write(data)
			if flusher, ok := w.(http.Flusher); ok {
				flusher.Flush()
			}
			time.Sleep(10 * time.Millisecond) // Small delay between chunks
		}
	}))
	defer server.Close()

	// Test with conservative rate limiting
	opts := &Options{
		MaxRate: 2048, // 2KB/s - should allow download but with some throttling
	}

	tmpDir, _ := os.MkdirTemp("", "gdl_test")
	defer func() { _ = os.RemoveAll(tmpDir) }()
	dest := filepath.Join(tmpDir, "maxrate_test.txt")

	start := time.Now()
	stats, err := DownloadWithOptions(context.Background(), server.URL, dest, opts)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("DownloadWithOptions() with MaxRate error = %v", err)
	}

	if stats == nil {
		t.Error("Expected stats to be returned, got nil")
	}

	// Verify file was downloaded correctly
	fileInfo, err := os.Stat(dest)
	if err != nil {
		t.Errorf("Failed to stat downloaded file: %v", err)
	}

	if fileInfo.Size() != 4096 {
		t.Errorf("Expected file size 4096, got %d", fileInfo.Size())
	}

	t.Logf("Download completed in %v with rate limiting applied (file size: %d bytes)", duration, fileInfo.Size())
}

func TestDownloadWithUnlimitedRate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("unlimited rate test"))
	}))
	defer server.Close()

	// Test with unlimited rate (0)
	opts := &Options{
		MaxRate: 0, // Unlimited
	}

	tmpDir, _ := os.MkdirTemp("", "gdl_test")
	defer func() { _ = os.RemoveAll(tmpDir) }()
	dest := filepath.Join(tmpDir, "unlimited_test.txt")

	stats, err := DownloadWithOptions(context.Background(), server.URL, dest, opts)
	if err != nil {
		t.Errorf("DownloadWithOptions() with unlimited rate error = %v", err)
	}

	if stats == nil {
		t.Error("Expected stats to be returned, got nil")
	}

	content, err := os.ReadFile(dest)
	if err != nil {
		t.Fatalf("Failed to read downloaded file: %v", err)
	}

	if string(content) != "unlimited rate test" {
		t.Errorf("Expected content \"unlimited rate test\", got %q", string(content))
	}
}

func TestDownloadToWriterWithMaxRate(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("writer rate test"))
	}))
	defer server.Close()

	opts := &Options{
		MaxRate: 1024, // 1KB/s
	}

	var buf bytes.Buffer
	stats, err := DownloadToWriterWithOptions(context.Background(), server.URL, &buf, opts)
	if err != nil {
		t.Errorf("DownloadToWriterWithOptions() with MaxRate error = %v", err)
	}

	if stats == nil {
		t.Error("Expected stats to be returned, got nil")
	}

	if buf.String() != "writer rate test" {
		t.Errorf("Expected content \"writer rate test\", got %q", buf.String())
	}
}

func TestDownloadWithOptionsComprehensive(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "20")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("comprehensive test  ")) // 20 bytes
	}))
	defer server.Close()

	tempDir := t.TempDir()
	dest := filepath.Join(tempDir, "comprehensive_test.txt")

	t.Run("with all options set", func(t *testing.T) {
		var progressCalls []Progress
		opts := &Options{
			Timeout:           10 * time.Second,
			UserAgent:         "test-agent",
			MaxConcurrency:    2,
			ChunkSize:         1024,
			EnableResume:      true,
			CreateDirs:        true,
			OverwriteExisting: true,
			MaxRate:           1024, // 1KB/s
			Headers:           map[string]string{"X-Test": "value"},
			ProgressCallback: func(p Progress) {
				progressCalls = append(progressCalls, p)
			},
		}

		stats, err := DownloadWithOptions(context.Background(), server.URL, dest, opts)
		if err != nil {
			t.Errorf("DownloadWithOptions() with comprehensive options error = %v", err)
		}

		if stats == nil {
			t.Error("Expected stats to be returned, got nil")
		}

		// Verify file was downloaded
		content, err := os.ReadFile(dest)
		if err != nil {
			t.Fatalf("Failed to read downloaded file: %v", err)
		}

		if string(content) != "comprehensive test  " {
			t.Errorf("Expected content 'comprehensive test  ', got %q", string(content))
		}

		// Verify progress callback was called
		if len(progressCalls) == 0 {
			t.Error("Expected progress callback to be called, but it wasn't")
		}

		// Check that progress includes percentage calculation
		if len(progressCalls) > 0 {
			finalProgress := progressCalls[len(progressCalls)-1]
			if finalProgress.TotalSize != 20 {
				t.Errorf("Expected total size 20, got %d", finalProgress.TotalSize)
			}
			if finalProgress.Percentage <= 0 {
				t.Errorf("Expected percentage > 0, got %f", finalProgress.Percentage)
			}
		}
	})

	t.Run("with progress callback but zero total size", func(t *testing.T) {
		// Create server that doesn't send Content-Length
		serverNoLength := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("no length test"))
		}))
		defer serverNoLength.Close()

		destNoLength := filepath.Join(tempDir, "no_length_test.txt")
		var progressCalls []Progress

		opts := &Options{
			ProgressCallback: func(p Progress) {
				progressCalls = append(progressCalls, p)
			},
		}

		stats, err := DownloadWithOptions(context.Background(), serverNoLength.URL, destNoLength, opts)
		if err != nil {
			t.Errorf("DownloadWithOptions() with no content-length error = %v", err)
		}

		if stats == nil {
			t.Error("Expected stats to be returned, got nil")
		}

		// Verify progress callback was called even with unknown total size
		if len(progressCalls) == 0 {
			t.Error("Expected progress callback to be called, but it wasn't")
		}

		// When total size is 0 or unknown, percentage should be 0
		if len(progressCalls) > 0 {
			for _, progress := range progressCalls {
				if progress.TotalSize == 0 && progress.Percentage != 0 {
					t.Errorf("Expected percentage 0 when total size is 0, got %f", progress.Percentage)
				}
			}
		}
	})

	t.Run("with nil options", func(t *testing.T) {
		destNil := filepath.Join(tempDir, "nil_options_test.txt")

		stats, err := DownloadWithOptions(context.Background(), server.URL, destNil, nil)
		if err != nil {
			t.Errorf("DownloadWithOptions() with nil options error = %v", err)
		}

		if stats == nil {
			t.Error("Expected stats to be returned, got nil")
		}

		// Verify file was downloaded
		content, err := os.ReadFile(destNil)
		if err != nil {
			t.Fatalf("Failed to read downloaded file: %v", err)
		}

		if string(content) != "comprehensive test  " {
			t.Errorf("Expected content 'comprehensive test  ', got %q", string(content))
		}
	})
}

// Helper function for DownloadToWriterWithOptions
func DownloadToWriterWithOptions(ctx context.Context, url string, w io.Writer, opts *Options) (*DownloadStats, error) {
	downloader := NewDownloader()
	return downloader.DownloadToWriter(ctx, url, w, opts)
}

// TestUseMiddleware tests the UseMiddleware method
func TestUseMiddleware(t *testing.T) {
	downloader := NewDownloader()

	// Create a simple test middleware
	testMiddleware := func(next middleware.Handler) middleware.Handler {
		return func(ctx context.Context, req *middleware.DownloadRequest) (*middleware.DownloadResponse, error) {
			return next(ctx, req)
		}
	}

	// Register middleware
	downloader.UseMiddleware(testMiddleware)

	// Verify the method doesn't panic
	if downloader.middleware == nil {
		t.Error("Expected middleware to be initialized")
	}
}

// TestOn tests the On event listener registration
func TestOn(t *testing.T) {
	downloader := NewDownloader()

	// Create a simple test handler
	handler := func(event events.Event) {
		// Handler logic would go here
	}

	// Register event listener
	downloader.On(events.EventDownloadStarted, handler)

	// Verify the method doesn't panic
	if downloader.eventEmitter == nil {
		t.Error("Expected eventEmitter to be initialized")
	}

	// Test multiple event types
	downloader.On(events.EventDownloadCompleted, handler)
	downloader.On(events.EventDownloadFailed, handler)
	downloader.On(events.EventDownloadProgress, handler)
}

// TestDownloaderMethodChaining tests that methods can be chained
func TestDownloaderMethodChaining(t *testing.T) {
	downloader := NewDownloader()

	// Test chaining UseMiddleware calls
	middleware1 := func(next middleware.Handler) middleware.Handler {
		return next
	}
	middleware2 := func(next middleware.Handler) middleware.Handler {
		return next
	}

	downloader.UseMiddleware(middleware1)
	downloader.UseMiddleware(middleware2)

	// Test chaining On calls
	handler := func(event events.Event) {}

	downloader.On(events.EventDownloadStarted, handler)
	downloader.On(events.EventDownloadCompleted, handler)

	// Verify downloader is still functional
	if downloader.eventEmitter == nil {
		t.Error("Expected eventEmitter to remain initialized after chaining")
	}
	if downloader.middleware == nil {
		t.Error("Expected middleware to remain initialized after chaining")
	}
}
