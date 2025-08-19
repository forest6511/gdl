// Package ratelimit provides bandwidth throttling functionality for downloads.
package ratelimit

import (
	"context"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

// Limiter interface defines the contract for bandwidth limiting.
type Limiter interface {
	// Wait blocks until the limiter allows n bytes to be processed.
	// Returns an error if the context is cancelled.
	Wait(ctx context.Context, n int) error

	// Allow reports whether n bytes can be processed immediately.
	Allow(n int) bool

	// Rate returns the current rate limit in bytes per second.
	Rate() int64

	// SetRate updates the rate limit. A value of 0 means unlimited.
	SetRate(bytesPerSec int64)
}

// BandwidthLimiter implements thread-safe bandwidth limiting using a token bucket algorithm.
type BandwidthLimiter struct {
	mu      sync.RWMutex
	limiter *rate.Limiter
	maxRate int64 // bytes per second, 0 means unlimited
}

// NewBandwidthLimiter creates a new bandwidth limiter.
// maxRate is in bytes per second. A value of 0 means unlimited.
func NewBandwidthLimiter(maxRate int64) *BandwidthLimiter {
	bl := &BandwidthLimiter{
		maxRate: maxRate,
	}

	if maxRate > 0 {
		// Create rate limiter with the specified rate and a burst of 1 second worth of data
		// This allows for some burstiness while maintaining the overall rate
		bl.limiter = rate.NewLimiter(rate.Limit(maxRate), int(maxRate))
	}

	return bl
}

// Wait blocks until the limiter allows n bytes to be processed.
func (bl *BandwidthLimiter) Wait(ctx context.Context, n int) error {
	bl.mu.RLock()
	limiter := bl.limiter
	bl.mu.RUnlock()

	// If no rate limit is set, allow immediately
	if limiter == nil {
		return nil
	}

	// Wait for the required tokens
	return limiter.WaitN(ctx, n)
}

// Allow reports whether n bytes can be processed immediately.
func (bl *BandwidthLimiter) Allow(n int) bool {
	bl.mu.RLock()
	limiter := bl.limiter
	bl.mu.RUnlock()

	// If no rate limit is set, allow immediately
	if limiter == nil {
		return true
	}

	return limiter.AllowN(time.Now(), n)
}

// Rate returns the current rate limit in bytes per second.
func (bl *BandwidthLimiter) Rate() int64 {
	bl.mu.RLock()
	defer bl.mu.RUnlock()
	return bl.maxRate
}

// SetRate updates the rate limit. A value of 0 means unlimited.
func (bl *BandwidthLimiter) SetRate(bytesPerSec int64) {
	bl.mu.Lock()
	defer bl.mu.Unlock()

	bl.maxRate = bytesPerSec

	if bytesPerSec <= 0 {
		bl.limiter = nil
	} else {
		// Update the existing limiter or create a new one
		bl.limiter = rate.NewLimiter(rate.Limit(bytesPerSec), int(bytesPerSec))
	}
}

// NullLimiter is a no-op limiter that allows unlimited bandwidth.
type NullLimiter struct{}

// NewNullLimiter creates a limiter that imposes no limits.
func NewNullLimiter() *NullLimiter {
	return &NullLimiter{}
}

// Wait always returns immediately without blocking.
func (nl *NullLimiter) Wait(ctx context.Context, n int) error {
	return nil
}

// Allow always returns true.
func (nl *NullLimiter) Allow(n int) bool {
	return true
}

// Rate always returns 0 (unlimited).
func (nl *NullLimiter) Rate() int64 {
	return 0
}

// SetRate is a no-op for the null limiter.
func (nl *NullLimiter) SetRate(bytesPerSec int64) {
	// No-op
}
