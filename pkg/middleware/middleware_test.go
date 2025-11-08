package middleware

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
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
				Stats: &types.DownloadStats{
					URL:             req.URL,
					Filename:        "test.txt",
					TotalSize:       1024,
					BytesDownloaded: 1024,
					StartTime:       time.Now(),
					EndTime:         time.Now(),
					Duration:        time.Second,
					Success:         true,
				},
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

		// Second request should use cached response
		resp2, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}

		if resp2 == nil {
			t.Fatal("Expected response, got nil")
		}

		// Handler should only be called once (second request uses cache)
		if callCount != 1 {
			t.Errorf("Expected handler to be called once (cache hit on second request), got %d", callCount)
		}

		// Verify second response is marked as cached
		if !resp2.Cached {
			t.Error("Expected second response to be marked as cached")
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

// Test cache serialization and deserialization
func TestSerializeDeserializeResponse(t *testing.T) {
	t.Run("RoundTripSuccessfulResponse", func(t *testing.T) {
		startTime := time.Now().Add(-5 * time.Minute)
		endTime := time.Now()

		originalResp := &DownloadResponse{
			Stats: &types.DownloadStats{
				URL:             "http://example.com/file.txt",
				Filename:        "file.txt",
				TotalSize:       1024,
				BytesDownloaded: 1024,
				StartTime:       startTime,
				EndTime:         endTime,
				Duration:        endTime.Sub(startTime),
				Success:         true,
				Error:           nil,
			},
			Headers: map[string][]string{
				"Content-Type": {"text/plain"},
				"ETag":         {"abc123"},
			},
			Metadata: map[string]interface{}{
				"source":  "test",
				"retries": 2,
			},
			Cached: false,
		}

		// Serialize
		data, err := serializeResponse(originalResp)
		if err != nil {
			t.Fatalf("Failed to serialize response: %v", err)
		}

		if len(data) == 0 {
			t.Error("Expected non-empty serialized data")
		}

		// Deserialize
		deserializedResp, err := deserializeResponse(data)
		if err != nil {
			t.Fatalf("Failed to deserialize response: %v", err)
		}

		// Verify stats
		if deserializedResp.Stats.URL != originalResp.Stats.URL {
			t.Errorf("URL mismatch: expected %q, got %q", originalResp.Stats.URL, deserializedResp.Stats.URL)
		}
		if deserializedResp.Stats.Filename != originalResp.Stats.Filename {
			t.Errorf("Filename mismatch: expected %q, got %q", originalResp.Stats.Filename, deserializedResp.Stats.Filename)
		}
		if deserializedResp.Stats.TotalSize != originalResp.Stats.TotalSize {
			t.Errorf("TotalSize mismatch: expected %d, got %d", originalResp.Stats.TotalSize, deserializedResp.Stats.TotalSize)
		}
		if deserializedResp.Stats.BytesDownloaded != originalResp.Stats.BytesDownloaded {
			t.Errorf("BytesDownloaded mismatch: expected %d, got %d", originalResp.Stats.BytesDownloaded, deserializedResp.Stats.BytesDownloaded)
		}
		if deserializedResp.Stats.Success != originalResp.Stats.Success {
			t.Errorf("Success mismatch: expected %v, got %v", originalResp.Stats.Success, deserializedResp.Stats.Success)
		}

		// Verify times (within 1 second tolerance due to RFC3339 format)
		timeDiff := deserializedResp.Stats.StartTime.Sub(originalResp.Stats.StartTime)
		if timeDiff > time.Second || timeDiff < -time.Second {
			t.Errorf("StartTime mismatch: diff %v", timeDiff)
		}

		// Verify duration
		if deserializedResp.Stats.Duration != originalResp.Stats.Duration {
			t.Errorf("Duration mismatch: expected %v, got %v", originalResp.Stats.Duration, deserializedResp.Stats.Duration)
		}

		// Verify headers
		if len(deserializedResp.Headers) != len(originalResp.Headers) {
			t.Errorf("Headers count mismatch: expected %d, got %d", len(originalResp.Headers), len(deserializedResp.Headers))
		}

		// Verify cached flag is set
		if !deserializedResp.Cached {
			t.Error("Expected Cached flag to be true after deserialization")
		}
	})

	t.Run("ResponseWithError", func(t *testing.T) {
		originalResp := &DownloadResponse{
			Stats: &types.DownloadStats{
				URL:             "http://example.com/error.txt",
				Filename:        "error.txt",
				TotalSize:       0,
				BytesDownloaded: 0,
				StartTime:       time.Now(),
				EndTime:         time.Now(),
				Duration:        0,
				Success:         false,
				Error:           fmt.Errorf("download failed: network timeout"),
			},
			Headers:  make(map[string][]string),
			Metadata: make(map[string]interface{}),
		}

		// Serialize
		data, err := serializeResponse(originalResp)
		if err != nil {
			t.Fatalf("Failed to serialize error response: %v", err)
		}

		// Deserialize
		deserializedResp, err := deserializeResponse(data)
		if err != nil {
			t.Fatalf("Failed to deserialize error response: %v", err)
		}

		// Verify error was preserved
		if deserializedResp.Stats.Error == nil {
			t.Error("Expected error to be preserved")
		} else {
			if deserializedResp.Stats.Error.Error() != originalResp.Stats.Error.Error() {
				t.Errorf("Error message mismatch: expected %q, got %q",
					originalResp.Stats.Error.Error(), deserializedResp.Stats.Error.Error())
			}
		}
	})

	t.Run("SerializeNilResponse", func(t *testing.T) {
		_, err := serializeResponse(nil)
		if err == nil {
			t.Error("Expected error when serializing nil response")
		}
	})

	t.Run("SerializeNilStats", func(t *testing.T) {
		resp := &DownloadResponse{
			Stats: nil,
		}
		_, err := serializeResponse(resp)
		if err == nil {
			t.Error("Expected error when serializing response with nil stats")
		}
	})

	t.Run("DeserializeInvalidJSON", func(t *testing.T) {
		invalidData := []byte("{invalid json}")
		_, err := deserializeResponse(invalidData)
		if err == nil {
			t.Error("Expected error when deserializing invalid JSON")
		}
	})

	t.Run("DeserializeInvalidTime", func(t *testing.T) {
		invalidTimeData := []byte(`{
			"stats": {
				"start_time": "not-a-valid-time",
				"end_time": "2024-01-01T00:00:00Z"
			}
		}`)
		_, err := deserializeResponse(invalidTimeData)
		if err == nil {
			t.Error("Expected error when deserializing invalid start time")
		}
	})
}

// Test cache middleware with actual caching implementation
func TestCacheMiddlewareImplementation(t *testing.T) {
	t.Run("CacheHit", func(t *testing.T) {
		cache := NewMemoryCache()
		middleware := CacheMiddleware(cache, 5*time.Minute)

		callCount := 0
		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			callCount++
			return &DownloadResponse{
				Stats: &types.DownloadStats{
					URL:             req.URL,
					Filename:        "test.txt",
					TotalSize:       1024,
					BytesDownloaded: 1024,
					StartTime:       time.Now(),
					EndTime:         time.Now(),
					Duration:        time.Second,
					Success:         true,
				},
				Headers:  make(map[string][]string),
				Metadata: make(map[string]interface{}),
			}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/cached-file.txt"}

		// First request - should call handler and cache result
		resp1, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Fatalf("Expected no error on first request, got %v", err)
		}
		if resp1 == nil {
			t.Fatal("Expected response on first request")
		}
		if resp1.Cached {
			t.Error("Expected first response not to be marked as cached")
		}
		if callCount != 1 {
			t.Errorf("Expected handler to be called once, got %d", callCount)
		}

		// Second request - should use cache without calling handler
		resp2, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Fatalf("Expected no error on second request, got %v", err)
		}
		if resp2 == nil {
			t.Fatal("Expected response on second request")
		}
		if !resp2.Cached {
			t.Error("Expected second response to be marked as cached")
		}
		if callCount != 1 {
			t.Errorf("Expected handler to be called only once (cache hit), got %d", callCount)
		}

		// Verify cached data matches
		if resp2.Stats.URL != resp1.Stats.URL {
			t.Error("Cached response URL doesn't match original")
		}
		if resp2.Stats.BytesDownloaded != resp1.Stats.BytesDownloaded {
			t.Error("Cached response bytes downloaded doesn't match original")
		}
	})

	t.Run("CacheMissOnFailure", func(t *testing.T) {
		cache := NewMemoryCache()
		middleware := CacheMiddleware(cache, 5*time.Minute)

		callCount := 0
		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			callCount++
			return &DownloadResponse{
				Stats: &types.DownloadStats{
					URL:     req.URL,
					Success: false,
					Error:   fmt.Errorf("download failed"),
				},
			}, fmt.Errorf("download failed")
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/failing-file.txt"}

		// First request - fails, should not cache
		_, err := wrappedHandler(context.Background(), req)
		if err == nil {
			t.Error("Expected error on failed download")
		}

		// Second request - should call handler again (no cache)
		_, err = wrappedHandler(context.Background(), req)
		if err == nil {
			t.Error("Expected error on second failed download")
		}

		if callCount != 2 {
			t.Errorf("Expected handler to be called twice (no caching on failure), got %d", callCount)
		}
	})

	t.Run("CacheExpiration", func(t *testing.T) {
		cache := NewMemoryCache()
		// Very short TTL for testing
		middleware := CacheMiddleware(cache, 1*time.Millisecond)

		callCount := 0
		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			callCount++
			return &DownloadResponse{
				Stats: &types.DownloadStats{
					URL:             req.URL,
					Filename:        "test.txt",
					TotalSize:       1024,
					BytesDownloaded: 1024,
					StartTime:       time.Now(),
					EndTime:         time.Now(),
					Duration:        time.Second,
					Success:         true,
				},
			}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/expiring-file.txt"}

		// First request
		_, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		// Wait for cache to expire
		time.Sleep(5 * time.Millisecond)

		// Second request after expiration - should call handler again
		_, err = wrappedHandler(context.Background(), req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}

		if callCount != 2 {
			t.Errorf("Expected handler to be called twice (after expiration), got %d", callCount)
		}
	})
}

// Test isRetryableError with DownloadError types
func TestIsRetryableErrorWithDownloadError(t *testing.T) {
	t.Run("RetryableDownloadError", func(t *testing.T) {
		err := &gdlerrors.DownloadError{
			Code:      gdlerrors.CodeNetworkError,
			Message:   "network timeout",
			Retryable: true,
		}

		if !isRetryableError(err) {
			t.Error("Expected retryable DownloadError to be retryable")
		}
	})

	t.Run("NonRetryableDownloadError", func(t *testing.T) {
		err := &gdlerrors.DownloadError{
			Code:      gdlerrors.CodeInvalidURL,
			Message:   "invalid URL",
			Retryable: false,
		}

		if isRetryableError(err) {
			t.Error("Expected non-retryable DownloadError to not be retryable")
		}
	})

	t.Run("WrappedRetryableDownloadError", func(t *testing.T) {
		innerErr := &gdlerrors.DownloadError{
			Code:      gdlerrors.CodeNetworkError,
			Message:   "connection timeout",
			Retryable: true,
		}
		wrappedErr := fmt.Errorf("wrapped error: %w", innerErr)

		if !isRetryableError(wrappedErr) {
			t.Error("Expected wrapped retryable DownloadError to be retryable")
		}
	})

	t.Run("FallbackToKeywordMatching", func(t *testing.T) {
		// Generic error with retryable keyword
		err := fmt.Errorf("connection timeout occurred")

		if !isRetryableError(err) {
			t.Error("Expected error with 'timeout' keyword to be retryable")
		}

		// Generic error without retryable keyword
		err = fmt.Errorf("file not found")
		if isRetryableError(err) {
			t.Error("Expected error without retryable keywords to not be retryable")
		}
	})

	t.Run("NilError", func(t *testing.T) {
		if isRetryableError(nil) {
			t.Error("Expected nil error to not be retryable")
		}
	})

	t.Run("ContextErrors", func(t *testing.T) {
		// Context canceled should not be retryable
		if isRetryableError(context.Canceled) {
			t.Error("Expected context.Canceled to not be retryable")
		}

		// Context deadline exceeded might be retryable depending on implementation
		// Test the actual behavior
		result := isRetryableError(context.DeadlineExceeded)
		t.Logf("context.DeadlineExceeded retryable: %v", result)
	})
}

// Test coverage improvements for CacheMiddleware error paths
func TestCacheMiddlewareErrorCases(t *testing.T) {
	t.Run("DeserializationFailureFallsBackToDownload", func(t *testing.T) {
		cache := NewMemoryCache()
		middleware := CacheMiddleware(cache, 5*time.Minute)

		// Manually insert invalid cached data
		cacheKey := generateCacheKey(&DownloadRequest{URL: "http://example.com/test.txt"})
		if err := cache.Set(cacheKey, []byte("invalid json data"), 5*time.Minute); err != nil {
			t.Fatalf("Failed to set cache: %v", err)
		}

		callCount := 0
		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			callCount++
			return &DownloadResponse{
				Stats: &types.DownloadStats{
					URL:             req.URL,
					Filename:        "test.txt",
					TotalSize:       1024,
					BytesDownloaded: 1024,
					StartTime:       time.Now(),
					EndTime:         time.Now(),
					Duration:        time.Second,
					Success:         true,
				},
				Headers:  make(map[string][]string),
				Metadata: make(map[string]interface{}),
			}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test.txt"}

		// Should fall back to handler when deserialization fails
		resp, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if resp == nil {
			t.Fatal("Expected response")
		}
		if callCount != 1 {
			t.Errorf("Expected handler to be called once (deserialization failed), got %d", callCount)
		}
	})

	t.Run("SerializationFailureLogsWarning", func(t *testing.T) {
		// Create a mock cache that tracks Set calls
		setCalled := false
		mockCache := &mockCacheBackend{
			data: make(map[string]cacheEntry),
			setFunc: func(key string, data []byte, ttl time.Duration) error {
				setCalled = true
				return nil
			},
		}

		middleware := CacheMiddleware(mockCache, 5*time.Minute)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			// Return response with nil Stats to trigger serialization error
			return &DownloadResponse{
				Stats:    nil, // This will cause serializeResponse to fail
				Headers:  make(map[string][]string),
				Metadata: make(map[string]interface{}),
			}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test.txt"}

		resp, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Fatalf("Expected no error, got %v", err)
		}
		if resp == nil {
			t.Fatal("Expected response")
		}

		// Cache.Set should not be called because serialization failed
		if setCalled {
			t.Error("Expected cache.Set not to be called when serialization fails")
		}
	})

	t.Run("CacheSetFailureLogsWarning", func(t *testing.T) {
		// Create a mock cache that fails on Set
		mockCache := &mockCacheBackend{
			data: make(map[string]cacheEntry),
			setFunc: func(key string, data []byte, ttl time.Duration) error {
				return fmt.Errorf("cache set failed")
			},
		}

		middleware := CacheMiddleware(mockCache, 5*time.Minute)

		handler := func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			return &DownloadResponse{
				Stats: &types.DownloadStats{
					URL:             req.URL,
					Filename:        "test.txt",
					TotalSize:       1024,
					BytesDownloaded: 1024,
					StartTime:       time.Now(),
					EndTime:         time.Now(),
					Duration:        time.Second,
					Success:         true,
				},
				Headers:  make(map[string][]string),
				Metadata: make(map[string]interface{}),
			}, nil
		}

		wrappedHandler := middleware(handler)
		req := &DownloadRequest{URL: "http://example.com/test.txt"}

		// Should complete successfully even though cache.Set failed
		resp, err := wrappedHandler(context.Background(), req)
		if err != nil {
			t.Fatalf("Expected no error even when cache.Set fails, got %v", err)
		}
		if resp == nil {
			t.Fatal("Expected response")
		}
	})
}

// Mock cache backend for testing error cases
type mockCacheBackend struct {
	data    map[string]cacheEntry
	setFunc func(key string, data []byte, ttl time.Duration) error
}

func (m *mockCacheBackend) Get(key string) ([]byte, bool) {
	entry, exists := m.data[key]
	if !exists || time.Now().After(entry.expiry) {
		return nil, false
	}
	return entry.value, true
}

func (m *mockCacheBackend) Set(key string, data []byte, ttl time.Duration) error {
	if m.setFunc != nil {
		return m.setFunc(key, data, ttl)
	}
	m.data[key] = cacheEntry{
		value:  data,
		expiry: time.Now().Add(ttl),
	}
	return nil
}

func (m *mockCacheBackend) Delete(key string) error {
	delete(m.data, key)
	return nil
}

func (m *mockCacheBackend) Clear() error {
	m.data = make(map[string]cacheEntry)
	return nil
}

// Test generateCacheKey with UserAgent
func TestGenerateCacheKeyWithUserAgent(t *testing.T) {
	req1 := &DownloadRequest{
		URL:       "http://example.com/test.txt",
		UserAgent: "TestAgent/1.0",
	}
	req2 := &DownloadRequest{
		URL:       "http://example.com/test.txt",
		UserAgent: "TestAgent/2.0",
	}
	req3 := &DownloadRequest{
		URL: "http://example.com/test.txt",
		// No UserAgent
	}

	key1 := generateCacheKey(req1)
	key2 := generateCacheKey(req2)
	key3 := generateCacheKey(req3)

	// Different UserAgents should produce different keys
	if key1 == key2 {
		t.Error("Expected different cache keys for different UserAgents")
	}

	// UserAgent vs no UserAgent should produce different keys
	if key1 == key3 {
		t.Error("Expected different cache keys when UserAgent is present vs absent")
	}
	if key2 == key3 {
		t.Error("Expected different cache keys when UserAgent is present vs absent")
	}
}

// Test getScheme with s3:// URLs
func TestGetSchemeWithS3(t *testing.T) {
	// Test full s3 URL
	scheme := getScheme("s3://bucket/key")
	if scheme != "s3" {
		t.Errorf("Expected 's3', got %q", scheme)
	}

	// Test minimum valid s3 URL (len > 6, so at least 7 chars)
	scheme = getScheme("s3://bu")
	if scheme != "s3" {
		t.Errorf("Expected 's3' for minimum valid s3 URL, got %q", scheme)
	}

	// Test edge cases that are too short (len <= 6)
	// These should return "unknown" based on the implementation
	scheme = getScheme("s3://b")
	if scheme != "unknown" {
		t.Errorf("Expected 'unknown' for too-short URL (len=6), got %q", scheme)
	}

	scheme = getScheme("s3://")
	if scheme != "unknown" {
		t.Errorf("Expected 'unknown' for bare s3:// (len=5), got %q", scheme)
	}
}

// Test deserializeResponse with endTime parsing error
func TestDeserializeResponseWithEndTimeError(t *testing.T) {
	invalidData := []byte(`{
		"stats": {
			"start_time": "2024-01-01T00:00:00Z",
			"end_time": "invalid-time-format"
		}
	}`)

	_, err := deserializeResponse(invalidData)
	if err == nil {
		t.Error("Expected error when deserializing invalid end time")
	}
	if err != nil && !contains(err.Error(), "failed to parse end time") {
		t.Errorf("Expected 'failed to parse end time' error, got: %v", err)
	}
}
