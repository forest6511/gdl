package errors

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"
)

// RetryStrategy defines the retry behavior.
type RetryStrategy struct {
	MaxAttempts    int
	InitialDelay   time.Duration
	MaxDelay       time.Duration
	Multiplier     float64
	Jitter         bool
	RetryCondition func(error) bool
	OnRetry        func(attempt int, err error, nextDelay time.Duration)
}

// DefaultRetryStrategy returns a default retry strategy.
func DefaultRetryStrategy() *RetryStrategy {
	return &RetryStrategy{
		MaxAttempts:    3,
		InitialDelay:   1 * time.Second,
		MaxDelay:       30 * time.Second,
		Multiplier:     2.0,
		Jitter:         true,
		RetryCondition: IsRetryable, // Use existing IsRetryable function
		OnRetry:        nil,
	}
}

// ExponentialBackoffStrategy returns an exponential backoff strategy.
func ExponentialBackoffStrategy(maxAttempts int) *RetryStrategy {
	return &RetryStrategy{
		MaxAttempts:    maxAttempts,
		InitialDelay:   500 * time.Millisecond,
		MaxDelay:       60 * time.Second,
		Multiplier:     2.0,
		Jitter:         true,
		RetryCondition: IsRetryable,
		OnRetry:        nil,
	}
}

// RetryManager handles retry logic.
type RetryManager struct {
	strategy *RetryStrategy
}

// NewRetryManager creates a new retry manager.
func NewRetryManager(strategy *RetryStrategy) *RetryManager {
	if strategy == nil {
		strategy = DefaultRetryStrategy()
	}

	return &RetryManager{
		strategy: strategy,
	}
}

// Execute runs the given function with retry logic.
func (rm *RetryManager) Execute(ctx context.Context, operation func() error) error {
	var lastErr error

	delay := rm.strategy.InitialDelay

	for attempt := 1; attempt <= rm.strategy.MaxAttempts; attempt++ {
		// Check context
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute operation
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Check if we should retry
		if !rm.shouldRetry(err) {
			return err
		}

		// Check if this was the last attempt
		if attempt >= rm.strategy.MaxAttempts {
			break
		}

		// Calculate next delay
		if attempt > 1 {
			delay = rm.calculateDelay(delay)
		}

		// Call retry callback if provided
		if rm.strategy.OnRetry != nil {
			rm.strategy.OnRetry(attempt, err, delay)
		}

		// Wait before next attempt
		timer := time.NewTimer(delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
		}
	}

	// Wrap the last error with retry exhausted information
	return fmt.Errorf("failed after %d attempts: %w", rm.strategy.MaxAttempts, lastErr)
}

// shouldRetry determines if an error should be retried.
func (rm *RetryManager) shouldRetry(err error) bool {
	// Use custom condition if provided
	if rm.strategy.RetryCondition != nil {
		return rm.strategy.RetryCondition(err)
	}

	// Default: use the existing IsRetryable function
	return IsRetryable(err)
}

// calculateDelay calculates the next retry delay.
func (rm *RetryManager) calculateDelay(currentDelay time.Duration) time.Duration {
	// Apply multiplier
	nextDelay := time.Duration(float64(currentDelay) * rm.strategy.Multiplier)

	// Apply max delay cap
	if nextDelay > rm.strategy.MaxDelay {
		nextDelay = rm.strategy.MaxDelay
	}

	// Apply jitter if enabled
	if rm.strategy.Jitter {
		// #nosec G404 -- Jitter for retry delays doesn't require cryptographic randomness
		jitter := time.Duration(rand.Float64() * float64(nextDelay) * 0.1)
		// #nosec G404 -- Random choice for jitter direction is not security-sensitive
		if rand.Intn(2) == 0 {
			nextDelay += jitter
		} else {
			nextDelay -= jitter
		}
	}

	return nextDelay
}

// CircuitBreaker implements a circuit breaker pattern for retry logic.
type CircuitBreaker struct {
	maxFailures     int
	resetTimeout    time.Duration
	failureCount    int
	lastFailureTime time.Time
	state           CircuitState
}

// CircuitState represents the state of a circuit breaker.
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker.
func NewCircuitBreaker(maxFailures int, resetTimeout time.Duration) *CircuitBreaker {
	return &CircuitBreaker{
		maxFailures:  maxFailures,
		resetTimeout: resetTimeout,
		state:        CircuitClosed,
	}
}

// Call executes a function through the circuit breaker.
func (cb *CircuitBreaker) Call(fn func() error) error {
	if cb.state == CircuitOpen {
		if time.Since(cb.lastFailureTime) > cb.resetTimeout {
			cb.state = CircuitHalfOpen
			cb.failureCount = 0
		} else {
			return fmt.Errorf("circuit breaker is open")
		}
	}

	err := fn()
	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = time.Now()

		if cb.failureCount >= cb.maxFailures {
			cb.state = CircuitOpen
			return fmt.Errorf("circuit breaker opened: %w", err)
		}

		return err
	}

	// Success - reset the circuit
	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
	}

	cb.failureCount = 0

	return nil
}

// AdaptiveRetryStrategy implements adaptive retry with backoff based on error patterns.
type AdaptiveRetryStrategy struct {
	baseStrategy    *RetryStrategy
	errorHistory    []time.Time
	historyWindow   time.Duration
	adaptMultiplier float64
}

// NewAdaptiveRetryStrategy creates a new adaptive retry strategy.
func NewAdaptiveRetryStrategy(base *RetryStrategy) *AdaptiveRetryStrategy {
	if base == nil {
		base = DefaultRetryStrategy()
	}

	return &AdaptiveRetryStrategy{
		baseStrategy:    base,
		errorHistory:    make([]time.Time, 0),
		historyWindow:   5 * time.Minute,
		adaptMultiplier: 1.0,
	}
}

// UpdateStrategy updates the strategy based on error patterns.
func (ars *AdaptiveRetryStrategy) UpdateStrategy(err error) {
	now := time.Now()
	ars.errorHistory = append(ars.errorHistory, now)

	// Clean old history
	cutoff := now.Add(-ars.historyWindow)
	newHistory := make([]time.Time, 0)

	for _, t := range ars.errorHistory {
		if t.After(cutoff) {
			newHistory = append(newHistory, t)
		}
	}

	ars.errorHistory = newHistory

	// Adapt multiplier based on error frequency
	errorRate := float64(len(ars.errorHistory)) / ars.historyWindow.Minutes()
	if errorRate > 1.0 {
		// High error rate - increase backoff
		ars.adaptMultiplier = math.Min(ars.adaptMultiplier*1.2, 3.0)
	} else if errorRate < 0.1 {
		// Low error rate - decrease backoff
		ars.adaptMultiplier = math.Max(ars.adaptMultiplier*0.9, 0.5)
	}
}

// GetAdjustedDelay returns the adjusted delay based on adaptive multiplier.
func (ars *AdaptiveRetryStrategy) GetAdjustedDelay(baseDelay time.Duration) time.Duration {
	return time.Duration(float64(baseDelay) * ars.adaptMultiplier)
}
