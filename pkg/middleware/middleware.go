package middleware

import (
	"compress/gzip"
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/forest6511/gdl/pkg/plugin"
	"github.com/forest6511/gdl/pkg/types"
)

// Handler represents a download handler function
type Handler func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error)

// Middleware represents a middleware function that wraps a Handler
type Middleware func(next Handler) Handler

// DownloadRequest contains the download request information
type DownloadRequest struct {
	URL         string
	Destination string
	Options     *types.DownloadOptions
	Headers     map[string]string
	UserAgent   string
	Metadata    map[string]interface{}
}

// DownloadResponse contains the download response information
type DownloadResponse struct {
	Stats    *types.DownloadStats
	Headers  map[string][]string
	Metadata map[string]interface{}
	Cached   bool
}

// MiddlewareChain manages a chain of middleware
type MiddlewareChain struct {
	middlewares []Middleware
	mu          sync.RWMutex
}

// NewMiddlewareChain creates a new middleware chain
func NewMiddlewareChain() *MiddlewareChain {
	return &MiddlewareChain{
		middlewares: make([]Middleware, 0),
	}
}

// Use adds a middleware to the chain
func (mc *MiddlewareChain) Use(m Middleware) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.middlewares = append(mc.middlewares, m)
}

// Then applies all middleware to the final handler and returns the wrapped handler
func (mc *MiddlewareChain) Then(handler Handler) Handler {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Apply middleware in reverse order so the first added runs first
	result := handler
	for i := len(mc.middlewares) - 1; i >= 0; i-- {
		result = mc.middlewares[i](result)
	}
	return result
}

// Clear removes all middleware from the chain
func (mc *MiddlewareChain) Clear() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.middlewares = mc.middlewares[:0]
}

// Count returns the number of middleware in the chain
func (mc *MiddlewareChain) Count() int {
	mc.mu.RLock()
	defer mc.mu.RUnlock()
	return len(mc.middlewares)
}

// Logger interface for logging middleware
type Logger interface {
	Info(msg string, args ...interface{})
	Error(msg string, args ...interface{})
	Debug(msg string, args ...interface{})
}

// MetricsCollector interface for metrics middleware
type MetricsCollector interface {
	IncrementCounter(name string, tags map[string]string)
	RecordDuration(name string, duration time.Duration, tags map[string]string)
	RecordGauge(name string, value float64, tags map[string]string)
}

// CacheBackend interface for cache middleware
type CacheBackend interface {
	Get(key string) ([]byte, bool)
	Set(key string, value []byte, ttl time.Duration) error
	Delete(key string) error
	Clear() error
}

// RateLimiter interface for rate limiting
type RateLimiter interface {
	Allow() bool
	Wait(ctx context.Context) error
}

// Built-in Middleware Functions

// RateLimitMiddleware creates a rate limiting middleware
func RateLimitMiddleware(limiter RateLimiter) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			// Check if request is allowed
			if !limiter.Allow() {
				// If not allowed, wait for permission
				if err := limiter.Wait(ctx); err != nil {
					return nil, fmt.Errorf("rate limit exceeded: %w", err)
				}
			}

			return next(ctx, req)
		}
	}
}

// AuthenticationMiddleware creates an authentication middleware
func AuthenticationMiddleware(auth plugin.AuthPlugin) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			// Create HTTP request for authentication
			httpReq, err := http.NewRequestWithContext(ctx, "GET", req.URL, nil)
			if err != nil {
				return nil, fmt.Errorf("failed to create request for authentication: %w", err)
			}

			// Add headers from download request
			for key, value := range req.Headers {
				httpReq.Header.Set(key, value)
			}

			// Authenticate the request
			if err := auth.Authenticate(ctx, httpReq); err != nil {
				return nil, fmt.Errorf("authentication failed: %w", err)
			}

			// Update headers from authenticated request
			for key, values := range httpReq.Header {
				if len(values) > 0 {
					req.Headers[key] = values[0]
				}
			}

			return next(ctx, req)
		}
	}
}

// LoggingMiddleware creates a logging middleware
func LoggingMiddleware(logger Logger) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			start := time.Now()

			logger.Info("Download started",
				"url", req.URL,
				"destination", req.Destination,
				"user_agent", req.UserAgent,
			)

			resp, err := next(ctx, req)

			duration := time.Since(start)

			if err != nil {
				logger.Error("Download failed",
					"url", req.URL,
					"duration", duration,
					"error", err,
				)
			} else {
				logger.Info("Download completed",
					"url", req.URL,
					"duration", duration,
					"bytes_downloaded", resp.Stats.BytesDownloaded,
					"success", resp.Stats.Success,
					"cached", resp.Cached,
				)
			}

			return resp, err
		}
	}
}

// MetricsMiddleware creates a metrics collection middleware
func MetricsMiddleware(collector MetricsCollector) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			start := time.Now()

			tags := map[string]string{
				"url_scheme": getScheme(req.URL),
			}

			// Increment request counter
			collector.IncrementCounter("download_requests_total", tags)

			resp, err := next(ctx, req)

			duration := time.Since(start)

			// Record duration
			collector.RecordDuration("download_duration", duration, tags)

			if err != nil {
				tags["status"] = "error"
				collector.IncrementCounter("download_errors_total", tags)
			} else {
				tags["status"] = "success"
				tags["cached"] = fmt.Sprintf("%t", resp.Cached)
				collector.IncrementCounter("download_success_total", tags)
				collector.RecordGauge("download_bytes", float64(resp.Stats.BytesDownloaded), tags)
			}

			return resp, err
		}
	}
}

// CacheMiddleware creates a cache middleware
func CacheMiddleware(cache CacheBackend, ttl time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			// Generate cache key
			cacheKey := generateCacheKey(req)

			// Try to get from cache first
			if cached, found := cache.Get(cacheKey); found {
				// TODO: Deserialize cached response
				// For now, we'll skip cache hit and proceed with download
				// In a real implementation, you'd deserialize the cached DownloadResponse
				_ = cached
			}

			// Execute the download
			resp, err := next(ctx, req)
			if err != nil {
				return resp, err
			}

			// Cache the successful response
			if resp.Stats.Success {
				// TODO: Serialize response for caching
				// For now, we'll just mark it as potentially cacheable
				// In a real implementation, you'd serialize the DownloadResponse
				cachedData := []byte(fmt.Sprintf("cached_%s", req.URL))
				if err := cache.Set(cacheKey, cachedData, ttl); err != nil {
					fmt.Printf("Warning: failed to cache response: %v\n", err)
				}
			}

			return resp, err
		}
	}
}

// CompressionMiddleware creates a compression middleware
func CompressionMiddleware(level int) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			// Add compression headers if not already present
			if req.Headers == nil {
				req.Headers = make(map[string]string)
			}

			// Request compressed response if supported
			if _, exists := req.Headers["Accept-Encoding"]; !exists {
				req.Headers["Accept-Encoding"] = "gzip, deflate"
			}

			resp, err := next(ctx, req)
			if err != nil {
				return resp, err
			}

			// Check if response was compressed
			if resp.Headers != nil {
				if encoding, exists := resp.Headers["Content-Encoding"]; exists {
					for _, enc := range encoding {
						if enc == "gzip" {
							// Response is already compressed
							if resp.Metadata == nil {
								resp.Metadata = make(map[string]interface{})
							}
							resp.Metadata["compressed"] = true
							resp.Metadata["compression_type"] = "gzip"
							break
						}
					}
				}
			}

			return resp, err
		}
	}
}

// RetryMiddleware creates a retry middleware
func RetryMiddleware(maxRetries int, delay time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			var lastErr error

			for attempt := 0; attempt <= maxRetries; attempt++ {
				if attempt > 0 {
					// Wait before retrying
					select {
					case <-time.After(delay):
					case <-ctx.Done():
						return nil, ctx.Err()
					}
				}

				resp, err := next(ctx, req)
				if err == nil {
					// Success
					if resp.Metadata == nil {
						resp.Metadata = make(map[string]interface{})
					}
					resp.Metadata["retry_attempts"] = attempt
					return resp, nil
				}

				lastErr = err

				// Check if error is retryable
				if !isRetryableError(err) {
					break
				}
			}

			return nil, fmt.Errorf("download failed after %d retries: %w", maxRetries, lastErr)
		}
	}
}

// TimeoutMiddleware creates a timeout middleware
func TimeoutMiddleware(timeout time.Duration) Middleware {
	return func(next Handler) Handler {
		return func(ctx context.Context, req *DownloadRequest) (*DownloadResponse, error) {
			// Create a context with timeout
			timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
			defer cancel()

			// Execute with timeout context
			return next(timeoutCtx, req)
		}
	}
}

// Helper functions

func generateCacheKey(req *DownloadRequest) string {
	h := sha256.New()
	h.Write([]byte(req.URL))
	if req.UserAgent != "" {
		h.Write([]byte(req.UserAgent))
	}
	for key, value := range req.Headers {
		h.Write([]byte(key + ":" + value))
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func getScheme(url string) string {
	if len(url) > 8 && url[:7] == "http://" {
		return "http"
	}
	if len(url) > 9 && url[:8] == "https://" {
		return "https"
	}
	if len(url) > 7 && url[:6] == "ftp://" {
		return "ftp"
	}
	if len(url) > 6 && url[:5] == "s3://" {
		return "s3"
	}
	return "unknown"
}

func isRetryableError(err error) bool {
	// TODO: Implement proper error type checking
	// For now, consider network-related errors as retryable
	errStr := err.Error()
	retryableKeywords := []string{
		"timeout",
		"connection refused",
		"connection reset",
		"network",
		"temporary",
	}

	for _, keyword := range retryableKeywords {
		if contains(errStr, keyword) {
			return true
		}
	}

	return false
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr ||
		(len(s) > len(substr) &&
			(s[:len(substr)] == substr ||
				s[len(s)-len(substr):] == substr ||
				containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// DefaultLogger provides a simple logger implementation
type DefaultLogger struct {
	logger *log.Logger
}

func NewDefaultLogger() *DefaultLogger {
	return &DefaultLogger{
		logger: log.New(log.Writer(), "", log.LstdFlags),
	}
}

func (dl *DefaultLogger) Info(msg string, args ...interface{}) {
	dl.logger.Printf("INFO: "+msg, args...)
}

func (dl *DefaultLogger) Error(msg string, args ...interface{}) {
	dl.logger.Printf("ERROR: "+msg, args...)
}

func (dl *DefaultLogger) Debug(msg string, args ...interface{}) {
	dl.logger.Printf("DEBUG: "+msg, args...)
}

// MemoryCache provides a simple in-memory cache implementation
type MemoryCache struct {
	data map[string]cacheEntry
	mu   sync.RWMutex
}

type cacheEntry struct {
	value  []byte
	expiry time.Time
}

func NewMemoryCache() *MemoryCache {
	return &MemoryCache{
		data: make(map[string]cacheEntry),
	}
}

func (mc *MemoryCache) Get(key string) ([]byte, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	entry, exists := mc.data[key]
	if !exists {
		return nil, false
	}

	if time.Now().After(entry.expiry) {
		// Entry expired
		delete(mc.data, key)
		return nil, false
	}

	return entry.value, true
}

func (mc *MemoryCache) Set(key string, value []byte, ttl time.Duration) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.data[key] = cacheEntry{
		value:  value,
		expiry: time.Now().Add(ttl),
	}

	return nil
}

func (mc *MemoryCache) Delete(key string) error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	delete(mc.data, key)
	return nil
}

func (mc *MemoryCache) Clear() error {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	mc.data = make(map[string]cacheEntry)
	return nil
}

// CompressedWriter wraps an io.Writer with gzip compression
type CompressedWriter struct {
	writer     io.Writer
	gzipWriter *gzip.Writer
}

func NewCompressedWriter(w io.Writer, level int) (*CompressedWriter, error) {
	gzipWriter, err := gzip.NewWriterLevel(w, level)
	if err != nil {
		return nil, err
	}

	return &CompressedWriter{
		writer:     w,
		gzipWriter: gzipWriter,
	}, nil
}

func (cw *CompressedWriter) Write(p []byte) (n int, err error) {
	return cw.gzipWriter.Write(p)
}

func (cw *CompressedWriter) Close() error {
	return cw.gzipWriter.Close()
}
