package middleware

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/forest6511/godl/pkg/types"
)

func TestNewMiddlewareChain(t *testing.T) {
	chain := NewMiddlewareChain()
	if chain == nil {
		t.Error("Expected middleware chain to be created, got nil")
	}

	if chain.Count() != 0 {
		t.Errorf("Expected empty chain count 0, got %d", chain.Count())
	}
}

func TestMiddlewareChainUse(t *testing.T) {
	chain := NewMiddlewareChain()

	// Test middleware that sets a flag
	executed := false
	middleware := func(next Handler) Handler {
		return func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			executed = true
			return next(ctx, req)
		}
	}

	chain.Use(middleware)

	if chain.Count() != 1 {
		t.Errorf("Expected chain count 1, got %d", chain.Count())
	}

	// Test execution
	handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
		return &DownloadResponse{}, nil
	}

	finalHandler := chain.Then(handler)
	req := &DownloadRequest{URL: "test"}

	_, err := finalHandler(context.Background(), req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if !executed {
		t.Error("Expected middleware to be executed")
	}
}

func TestMiddlewareChainThen(t *testing.T) {
	chain := NewMiddlewareChain()

	// Add middleware that modifies request
	chain.Use(func(next Handler) Handler {
		return func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			req.URL = "modified-" + req.URL
			return next(ctx, req)
		}
	})

	// Test handler
	finalURL := ""
	handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
		finalURL = req.URL
		return &DownloadResponse{}, nil
	}

	// Execute chain
	finalHandler := chain.Then(handler)
	req := &DownloadRequest{URL: "original"}

	_, err := finalHandler(context.Background(), req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if finalURL != "modified-original" {
		t.Errorf("Expected 'modified-original', got %q", finalURL)
	}
}

func TestMiddlewareChainClear(t *testing.T) {
	chain := NewMiddlewareChain()

	// Add some middleware
	chain.Use(func(next Handler) Handler {
		return func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return next(ctx, req)
		}
	})

	if chain.Count() != 1 {
		t.Errorf("Expected chain count 1, got %d", chain.Count())
	}

	chain.Clear()

	if chain.Count() != 0 {
		t.Errorf("Expected chain count 0 after clear, got %d", chain.Count())
	}
}

func TestTimeoutMiddleware(t *testing.T) {
	middleware := TimeoutMiddleware(10 * time.Millisecond)

	handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
		// Check if context is cancelled
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(50 * time.Millisecond): // Longer than timeout
			return &DownloadResponse{}, nil
		}
	}

	wrappedHandler := middleware(handler)
	req := &DownloadRequest{URL: "http://example.com/test"}

	_, err := wrappedHandler(context.Background(), req)
	if err == nil {
		t.Error("Expected timeout error, got nil")
	}
}

func TestDefaultLogger(t *testing.T) {
	logger := NewDefaultLogger()
	if logger == nil {
		t.Error("Expected logger to be created, got nil")
	}

	// Test methods don't panic
	logger.Info("test info")
	logger.Error("test error")
	logger.Debug("test debug")
}

func TestMemoryCache(t *testing.T) {
	cache := NewMemoryCache()

	// Test basic operations
	key := "test-key"
	data := []byte("test data")

	// Initially should not exist
	_, found := cache.Get(key)
	if found {
		t.Error("Expected key not to exist initially")
	}

	// Set and get
	_ = cache.Set(key, data, 5*time.Minute)

	retrieved, found := cache.Get(key)
	if !found {
		t.Error("Expected key to exist after setting")
	}

	if string(retrieved) != string(data) {
		t.Errorf("Expected %q, got %q", string(data), string(retrieved))
	}

	// Delete
	_ = cache.Delete(key)
	_, found = cache.Get(key)
	if found {
		t.Error("Expected key not to exist after deletion")
	}

	// Clear
	_ = cache.Set("key1", []byte("data1"), 5*time.Minute)
	_ = cache.Set("key2", []byte("data2"), 5*time.Minute)
	_ = cache.Clear()

	_, found1 := cache.Get("key1")
	_, found2 := cache.Get("key2")
	if found1 || found2 {
		t.Error("Expected all keys to be cleared")
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test generateCacheKey
	req := &DownloadRequest{URL: "http://example.com/test"}
	key := generateCacheKey(req)
	if key == "" {
		t.Error("Expected non-empty cache key")
	}

	// Test getScheme
	scheme := getScheme("http://example.com/test")
	if scheme != "http" {
		t.Errorf("Expected 'http', got %q", scheme)
	}

	scheme = getScheme("https://example.com/test")
	if scheme != "https" {
		t.Errorf("Expected 'https', got %q", scheme)
	}
}

func TestCompressedWriter(t *testing.T) {
	var buf bytes.Buffer
	writer, err := NewCompressedWriter(&buf, 1)
	if err != nil {
		t.Errorf("Expected no error creating compressed writer, got %v", err)
	}
	if writer == nil {
		t.Error("Expected compressed writer to be created, got nil")
	}

	// Test basic write
	data := []byte("test data")
	n, err := writer.Write(data)
	if err != nil {
		t.Errorf("Expected no error writing, got %v", err)
	}

	if n != len(data) {
		t.Errorf("Expected to write %d bytes, wrote %d", len(data), n)
	}

	// Test close
	err = writer.Close()
	if err != nil {
		t.Errorf("Expected no error closing, got %v", err)
	}
}

func TestMiddlewareIntegration(t *testing.T) {
	chain := NewMiddlewareChain()

	// Add multiple middleware
	chain.Use(LoggingMiddleware(NewDefaultLogger()))
	chain.Use(func(next Handler) Handler {
		return func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			resp, err := next(ctx, req)
			if resp != nil {
				resp.Cached = true
			}
			return resp, err
		}
	})

	handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
		return &DownloadResponse{
			Stats: &types.DownloadStats{Success: true},
		}, nil
	}

	finalHandler := chain.Then(handler)
	req := &DownloadRequest{URL: "http://example.com/test"}

	resp, err := finalHandler(context.Background(), req)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}

	if resp == nil {
		t.Error("Expected response, got nil")
		return
	}

	if !resp.Cached {
		t.Error("Expected response to be marked as cached")
	}
}

func TestUtilityFunctions(t *testing.T) {
	// Test generateCacheKey function
	req := &DownloadRequest{URL: "http://example.com/test"}
	key := generateCacheKey(req)
	if key == "" {
		t.Error("Expected non-empty cache key")
	}

	// Test getScheme function
	scheme := getScheme("http://example.com/test")
	if scheme != "http" {
		t.Errorf("Expected 'http', got %q", scheme)
	}

	scheme = getScheme("https://example.com/test")
	if scheme != "https" {
		t.Errorf("Expected 'https', got %q", scheme)
	}

	// Test with invalid URL
	scheme = getScheme("invalid-url")
	if scheme == "" {
		// This is expected for invalid URLs - no action needed
		t.Logf("Got empty scheme for invalid URL as expected")
	}
}

func TestMemoryCacheAdvanced(t *testing.T) {
	cache := NewMemoryCache()

	// Test cache expiration
	key := "test-key"
	data := []byte("test data")

	// Set with very short TTL
	_ = cache.Set(key, data, 1*time.Millisecond)

	// Should exist immediately
	retrieved, found := cache.Get(key)
	if !found {
		t.Error("Expected key to exist immediately after setting")
	}

	if string(retrieved) != string(data) {
		t.Errorf("Expected %q, got %q", string(data), string(retrieved))
	}

	// Wait for expiration
	time.Sleep(2 * time.Millisecond)

	// Should not exist after expiration
	_, found = cache.Get(key)
	if found {
		t.Error("Expected key to be expired")
	}
}

func TestCompressedWriterError(t *testing.T) {
	var buf bytes.Buffer

	// Test with invalid compression level
	_, err := NewCompressedWriter(&buf, 100) // Invalid level
	if err == nil {
		t.Error("Expected error for invalid compression level")
	}
}

func TestDefaultLoggerMethods(t *testing.T) {
	logger := NewDefaultLogger()

	// These methods should not panic
	logger.Info("test info message")
	logger.Error("test error message")
	logger.Debug("test debug message")

	// Test with empty messages
	logger.Info("")
	logger.Error("")
	logger.Debug("")
}
