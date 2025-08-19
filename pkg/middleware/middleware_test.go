package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/forest6511/gdl/pkg/types"
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

// Additional tests for improving coverage

// Mock MetricsCollector for testing
type mockMetricsCollector struct {
	counters  map[string]int
	gauges    map[string]float64
	durations map[string]time.Duration
}

func newMockMetricsCollector() *mockMetricsCollector {
	return &mockMetricsCollector{
		counters:  make(map[string]int),
		gauges:    make(map[string]float64),
		durations: make(map[string]time.Duration),
	}
}

func (m *mockMetricsCollector) IncrementCounter(name string, tags map[string]string) {
	m.counters[name]++
}

func (m *mockMetricsCollector) RecordGauge(name string, value float64, tags map[string]string) {
	m.gauges[name] = value
}

func (m *mockMetricsCollector) RecordDuration(name string, duration time.Duration, tags map[string]string) {
	m.durations[name] = duration
}

func TestMetricsMiddleware(t *testing.T) {
	collector := newMockMetricsCollector()
	middleware := MetricsMiddleware(collector)

	t.Run("SuccessfulDownload", func(t *testing.T) {
		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return &DownloadResponse{
				Stats: &types.DownloadStats{
					BytesDownloaded: 1024,
					Success:         true,
				},
				Cached: false,
			}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test"}

		resp, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}

		// Check metrics were recorded
		if collector.counters["download_requests_total"] != 1 {
			t.Errorf("Expected request counter to be 1, got %d", collector.counters["download_requests_total"])
		}

		if collector.counters["download_success_total"] != 1 {
			t.Errorf("Expected success counter to be 1, got %d", collector.counters["download_success_total"])
		}

		if collector.gauges["download_bytes"] != 1024 {
			t.Errorf("Expected bytes gauge to be 1024, got %f", collector.gauges["download_bytes"])
		}
	})

	t.Run("FailedDownload", func(t *testing.T) {
		collector := newMockMetricsCollector()
		middleware := MetricsMiddleware(collector)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return nil, context.DeadlineExceeded // Use a standard error
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "https://example.com/test"}

		_, err := wrappedHandler(context.Background(), req)
		if err == nil {
			t.Error("Expected error, got nil")
		}

		// Check error metrics were recorded
		if collector.counters["download_errors_total"] != 1 {
			t.Errorf("Expected error counter to be 1, got %d", collector.counters["download_errors_total"])
		}
	})
}

func TestRetryMiddleware(t *testing.T) {
	t.Run("SuccessOnFirstAttempt", func(t *testing.T) {
		middleware := RetryMiddleware(3, 10*time.Millisecond)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return &DownloadResponse{
				Stats: &types.DownloadStats{Success: true},
			}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test"}

		resp, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}

		// Check retry attempts metadata
		if attempts, ok := resp.Metadata["retry_attempts"].(int); ok {
			if attempts != 0 {
				t.Errorf("Expected 0 retry attempts, got %d", attempts)
			}
		}
	})

	t.Run("SuccessAfterRetry", func(t *testing.T) {
		attempts := 0
		middleware := RetryMiddleware(3, 10*time.Millisecond)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			attempts++
			if attempts < 2 {
				// Use an error that contains "timeout" which is retryable
				return nil, fmt.Errorf("connection timeout")
			}
			return &DownloadResponse{
				Stats: &types.DownloadStats{Success: true},
			}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test"}

		resp, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}

		// Check retry attempts metadata
		if retryAttempts, ok := resp.Metadata["retry_attempts"].(int); ok {
			if retryAttempts != 1 {
				t.Errorf("Expected 1 retry attempt, got %d", retryAttempts)
			}
		}
	})

	t.Run("NonRetryableError", func(t *testing.T) {
		middleware := RetryMiddleware(3, 10*time.Millisecond)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return nil, context.Canceled // Non-retryable error
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test"}

		_, err := wrappedHandler(context.Background(), req)
		if err == nil {
			t.Error("Expected error, got nil")
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		middleware := RetryMiddleware(3, 100*time.Millisecond)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return nil, context.DeadlineExceeded
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test"}

		ctx, cancel := context.WithCancel(context.Background())
		go func() {
			time.Sleep(50 * time.Millisecond)
			cancel()
		}()

		_, err := wrappedHandler(ctx, req)
		if err == nil {
			t.Error("Expected context cancellation error")
		}
	})
}

func TestCacheMiddleware(t *testing.T) {
	cache := NewMemoryCache()
	middleware := CacheMiddleware(cache, 5*time.Minute)

	t.Run("CacheMiss", func(t *testing.T) {
		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return &DownloadResponse{
				Stats: &types.DownloadStats{Success: true},
			}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test"}

		resp, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if resp == nil {
			t.Fatal("Expected response, got nil")
		}

		if resp.Cached {
			t.Error("Expected response not to be cached on first request")
		}
	})

	t.Run("CacheSetOnSuccess", func(t *testing.T) {
		cache := NewMemoryCache()
		middleware := CacheMiddleware(cache, 5*time.Minute)

		callCount := 0
		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			callCount++
			return &DownloadResponse{
				Stats: &types.DownloadStats{Success: true},
			}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/cached"}

		// First request - should cache
		resp1, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if resp1 == nil {
			t.Fatal("Expected response, got nil")
		}

		// Verify cache was set
		cacheKey := generateCacheKey(req)
		if _, found := cache.Get(cacheKey); !found {
			t.Error("Expected cache to be set after successful response")
		}

		// Note: Current implementation doesn't actually use cached responses,
		// so second request will still call handler
		resp2, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if resp2 == nil {
			t.Fatal("Expected response, got nil")
		}

		// Both requests should call handler since cache isn't fully implemented
		if callCount != 2 {
			t.Errorf("Expected handler to be called twice, got %d", callCount)
		}
	})
}

// TestAuthenticationMiddleware is skipped as it requires AuthPlugin interface
// which is not easily mockable in current test setup

// TestRateLimitMiddleware is skipped as it requires RateLimiter interface

// TestCompressionMiddleware is skipped as DownloadResponse doesn't have Data field

func TestAdditionalHelperFunctions(t *testing.T) {
	t.Run("isRetryableError", func(t *testing.T) {
		// Test with context errors
		if isRetryableError(context.DeadlineExceeded) {
			// Deadline exceeded may be retryable in some implementations
			t.Log("DeadlineExceeded is considered retryable")
		}

		if isRetryableError(context.Canceled) {
			t.Error("Canceled context should not be retryable")
		}
	})

	t.Run("contains", func(t *testing.T) {
		str := "apple banana orange"

		if !contains(str, "banana") {
			t.Error("Expected 'banana' to be in string")
		}

		if contains(str, "grape") {
			t.Error("Expected 'grape' not to be in string")
		}
	})

	t.Run("containsSubstring", func(t *testing.T) {
		str := "hello world foo bar test string"

		if !containsSubstring(str, "world") {
			t.Error("Expected to find 'world' substring")
		}

		if containsSubstring(str, "missing") {
			t.Error("Expected not to find 'missing' substring")
		}
	})
}

// Mock RateLimiter for testing
type mockRateLimiter struct {
	allowResult bool
	waitError   error
}

func (m *mockRateLimiter) Allow() bool {
	return m.allowResult
}

func (m *mockRateLimiter) Wait(ctx context.Context) error {
	return m.waitError
}

func TestRateLimitMiddleware(t *testing.T) {
	t.Run("AllowedRequest", func(t *testing.T) {
		limiter := &mockRateLimiter{allowResult: true}
		middleware := RateLimitMiddleware(limiter)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return &DownloadResponse{}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test"}

		resp, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if resp == nil {
			t.Error("Expected response, got nil")
		}
	})

	t.Run("BlockedRequestWithWait", func(t *testing.T) {
		limiter := &mockRateLimiter{allowResult: false, waitError: nil}
		middleware := RateLimitMiddleware(limiter)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return &DownloadResponse{}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test"}

		resp, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error after wait, got %v", err)
		}
		if resp == nil {
			t.Error("Expected response, got nil")
		}
	})

	t.Run("BlockedRequestWithWaitError", func(t *testing.T) {
		limiter := &mockRateLimiter{allowResult: false, waitError: fmt.Errorf("wait timeout")}
		middleware := RateLimitMiddleware(limiter)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return &DownloadResponse{}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test"}

		_, err := wrappedHandler(context.Background(), req)
		if err == nil {
			t.Error("Expected error when wait fails")
		}
	})
}

// Mock AuthPlugin for testing
type mockAuthPlugin struct {
	authError error
}

func (m *mockAuthPlugin) Name() string {
	return "mock-auth-plugin"
}

func (m *mockAuthPlugin) Version() string {
	return "1.0.0"
}

func (m *mockAuthPlugin) Init(config map[string]interface{}) error {
	return nil
}

func (m *mockAuthPlugin) Close() error {
	return nil
}

func (m *mockAuthPlugin) ValidateAccess(operation string, resource string) error {
	return nil
}

func (m *mockAuthPlugin) Authenticate(ctx context.Context, req *http.Request) error {
	if m.authError != nil {
		return m.authError
	}
	// Simulate adding authentication header
	req.Header.Set("Authorization", "Bearer test-token")
	return nil
}

func TestAuthenticationMiddleware(t *testing.T) {
	t.Run("SuccessfulAuthentication", func(t *testing.T) {
		auth := &mockAuthPlugin{}
		middleware := AuthenticationMiddleware(auth)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			// Check that auth header was added
			if req.Headers["Authorization"] != "Bearer test-token" {
				t.Error("Expected Authorization header to be set")
			}
			return &DownloadResponse{}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{
			URL:     "http://example.com/test",
			Headers: make(map[string]string),
		}

		resp, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if resp == nil {
			t.Error("Expected response, got nil")
		}
	})

	t.Run("AuthenticationFailure", func(t *testing.T) {
		auth := &mockAuthPlugin{authError: fmt.Errorf("auth failed")}
		middleware := AuthenticationMiddleware(auth)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return &DownloadResponse{}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{
			URL:     "http://example.com/test",
			Headers: make(map[string]string),
		}

		_, err := wrappedHandler(context.Background(), req)
		if err == nil {
			t.Error("Expected authentication error")
		}
	})

	t.Run("InvalidURL", func(t *testing.T) {
		auth := &mockAuthPlugin{}
		middleware := AuthenticationMiddleware(auth)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return &DownloadResponse{}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{
			URL: "://invalid-url",
		}

		_, err := wrappedHandler(context.Background(), req)
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})
}

func TestCompressionMiddleware(t *testing.T) {
	t.Run("AddCompressionHeaders", func(t *testing.T) {
		middleware := CompressionMiddleware(6)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			// Check compression headers were added
			if req.Headers["Accept-Encoding"] != "gzip, deflate" {
				t.Error("Expected Accept-Encoding header to be set")
			}
			return &DownloadResponse{}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test"}

		resp, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if resp == nil {
			t.Error("Expected response, got nil")
		}
	})

	t.Run("PreserveExistingHeaders", func(t *testing.T) {
		middleware := CompressionMiddleware(6)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			// Check existing headers are preserved
			if req.Headers["Accept-Encoding"] != "custom-encoding" {
				t.Error("Expected existing Accept-Encoding header to be preserved")
			}
			return &DownloadResponse{}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{
			URL: "http://example.com/test",
			Headers: map[string]string{
				"Accept-Encoding": "custom-encoding",
			},
		}

		_, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
	})

	t.Run("DetectCompressedResponse", func(t *testing.T) {
		middleware := CompressionMiddleware(6)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return &DownloadResponse{
				Headers: map[string][]string{
					"Content-Encoding": {"gzip"},
				},
			}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test"}

		resp, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if resp == nil {
			t.Fatal("Expected response, got nil")
		}

		// Check metadata was set
		if resp.Metadata["compressed"] != true {
			t.Error("Expected compressed metadata to be true")
		}
		if resp.Metadata["compression_type"] != "gzip" {
			t.Error("Expected compression_type to be gzip")
		}
	})

	t.Run("HandlerError", func(t *testing.T) {
		middleware := CompressionMiddleware(6)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return nil, fmt.Errorf("handler error")
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test"}

		_, err := wrappedHandler(context.Background(), req)
		if err == nil {
			t.Error("Expected handler error to be propagated")
		}
	})
}

func TestGenerateCacheKeyAdditional(t *testing.T) {
	// Test with different request parameters
	req1 := &DownloadRequest{URL: "http://example.com/file1.txt"}
	req2 := &DownloadRequest{URL: "http://example.com/file2.txt"}

	key1 := generateCacheKey(req1)
	key2 := generateCacheKey(req2)

	if key1 == key2 {
		t.Error("Expected different cache keys for different URLs")
	}

	// Test with headers
	req3 := &DownloadRequest{
		URL: "http://example.com/file.txt",
		Headers: map[string]string{
			"Authorization": "Bearer token",
		},
	}

	key3 := generateCacheKey(req3)
	if key3 == "" {
		t.Error("Expected non-empty cache key with headers")
	}
}

func TestGetSchemeAdditional(t *testing.T) {
	// Test edge cases
	scheme := getScheme("")
	if scheme != "unknown" {
		t.Errorf("Expected 'unknown' scheme for empty URL, got %q", scheme)
	}

	scheme = getScheme("ftp://example.com/file.txt")
	if scheme != "ftp" {
		t.Errorf("Expected 'ftp', got %q", scheme)
	}

	scheme = getScheme("file:///path/to/file")
	if scheme != "unknown" {
		t.Errorf("Expected 'unknown' for file scheme (not implemented), got %q", scheme)
	}

	// Test malformed URL
	scheme = getScheme("not-a-url")
	if scheme != "unknown" {
		t.Errorf("Expected 'unknown' for malformed URL, got %q", scheme)
	}
}
