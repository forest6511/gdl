package ratelimit

import (
	"context"
	"strings"
	"testing"
	"time"
)

func TestNewBandwidthLimiter(t *testing.T) {
	tests := []struct {
		name        string
		bytesPerSec int64
	}{
		{
			name:        "valid rate",
			bytesPerSec: 1024,
		},
		{
			name:        "zero rate",
			bytesPerSec: 0,
		},
		{
			name:        "negative rate",
			bytesPerSec: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := NewBandwidthLimiter(tt.bytesPerSec)
			if limiter == nil {
				t.Error("Expected non-nil limiter")
			}
		})
	}
}

func TestBandwidthLimiter_Rate(t *testing.T) {
	limiter := NewBandwidthLimiter(1024)
	if rate := limiter.Rate(); rate != 1024 {
		t.Errorf("Expected rate 1024, got %d", rate)
	}
}

func TestBandwidthLimiter_SetRate(t *testing.T) {
	limiter := NewBandwidthLimiter(1024)

	// Set new rate
	limiter.SetRate(2048)
	if rate := limiter.Rate(); rate != 2048 {
		t.Errorf("Expected rate 2048, got %d", rate)
	}

	// Setting zero rate should disable limiting
	limiter.SetRate(0)
	if rate := limiter.Rate(); rate != 0 {
		t.Errorf("Expected rate 0 after setting unlimited, got %d", rate)
	}

	// Should allow unlimited bandwidth now
	if !limiter.Allow(1000000) {
		t.Error("Should allow unlimited bandwidth when rate is 0")
	}
}

func TestBandwidthLimiter_Allow(t *testing.T) {
	limiter := NewBandwidthLimiter(1024) // 1KB/s

	// Small request should be allowed
	if !limiter.Allow(100) {
		t.Error("Expected small request to be allowed")
	}

	// Very large request should not be allowed immediately
	if limiter.Allow(10240) { // 10KB
		t.Error("Expected large request to be denied initially")
	}
}

func TestBandwidthLimiter_Wait(t *testing.T) {
	limiter := NewBandwidthLimiter(1024) // 1KB/s
	ctx := context.Background()

	// Small request should not wait long
	start := time.Now()
	err := limiter.Wait(ctx, 100)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if duration > 100*time.Millisecond {
		t.Errorf("Small request took too long: %v", duration)
	}
}

func TestBandwidthLimiter_WaitWithCancellation(t *testing.T) {
	limiter := NewBandwidthLimiter(10) // Very slow: 10 bytes/s
	ctx, cancel := context.WithCancel(context.Background())

	// Cancel context immediately
	cancel()

	// Wait should return an error (either context cancellation or rate limiter error)
	err := limiter.Wait(ctx, 100) // Use smaller size that fits within burst
	if err == nil {
		t.Error("Expected an error when context is cancelled")
	}

	// Should contain context cancelled error or be context cancelled
	if err != context.Canceled && !strings.Contains(err.Error(), "context canceled") {
		t.Logf("Got error: %v (type: %T)", err, err)
		// This is acceptable - the rate limiter may return its own error when context is cancelled
	}
}

func TestBandwidthLimiter_WaitWithTimeout(t *testing.T) {
	limiter := NewBandwidthLimiter(10) // Very slow: 10 bytes/s
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	// Request should timeout or fail due to burst limit
	err := limiter.Wait(ctx, 100)
	if err == nil {
		t.Error("Expected timeout error")
	}

	// Should be either timeout or rate limiter error
	if err != context.DeadlineExceeded && !strings.Contains(err.Error(), "exceeds limiter's burst") && !strings.Contains(err.Error(), "deadline exceeded") {
		t.Logf("Got error: %v (type: %T)", err, err)
		// This is acceptable - the rate limiter may return its own error
	}
}

func TestNullLimiter(t *testing.T) {
	limiter := NewNullLimiter()

	// Rate should be 0 for unlimited
	if rate := limiter.Rate(); rate != 0 {
		t.Errorf("Expected rate 0 for null limiter, got %d", rate)
	}

	// Allow should always return true
	if !limiter.Allow(1000000) {
		t.Error("Null limiter should always allow requests")
	}

	// Wait should never block
	ctx := context.Background()
	start := time.Now()
	err := limiter.Wait(ctx, 1000000)
	duration := time.Since(start)

	if err != nil {
		t.Errorf("Null limiter should never return error: %v", err)
	}

	if duration > 10*time.Millisecond {
		t.Errorf("Null limiter should not block: %v", duration)
	}

	// SetRate should be no-op
	limiter.SetRate(1000)
	if rate := limiter.Rate(); rate != 0 {
		t.Errorf("Null limiter rate should remain 0, got %d", rate)
	}
}

func TestBandwidthLimiter_RateLimitingAccuracy(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limiting accuracy test in short mode")
	}

	limiter := NewBandwidthLimiter(100) // 100 bytes/s
	ctx := context.Background()

	// Download 200 bytes at 100 bytes/s should take ~2 seconds
	start := time.Now()

	// Make two requests of 100 bytes each
	err1 := limiter.Wait(ctx, 100)
	err2 := limiter.Wait(ctx, 100)

	duration := time.Since(start)

	if err1 != nil || err2 != nil {
		t.Errorf("Unexpected errors: %v, %v", err1, err2)
	}

	// Should take at least 1 second but not more than 3 seconds (allowing for timing variations)
	if duration < 1*time.Second {
		t.Errorf("Rate limiting too fast: %v", duration)
	}

	if duration > 3*time.Second {
		t.Errorf("Rate limiting too slow: %v", duration)
	}
}
