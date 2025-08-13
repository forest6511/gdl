package retry

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/forest6511/godl/pkg/errors"
)

// RetryManager manages retry attempts with configurable backoff strategies.
type RetryManager struct {
	MaxRetries    int           // Maximum number of retry attempts
	BaseDelay     time.Duration // Base delay for the first retry
	MaxDelay      time.Duration // Maximum delay between retries
	BackoffFactor float64       // Multiplier for exponential backoff
	Jitter        bool          // Whether to add jitter to delays
}

// NewRetryManager creates a new RetryManager with default settings.
func NewRetryManager() *RetryManager {
	return &RetryManager{
		MaxRetries:    3,
		BaseDelay:     1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// NewRetryManagerWithConfig creates a new RetryManager with the specified configuration.
func NewRetryManagerWithConfig(
	maxRetries int,
	baseDelay, maxDelay time.Duration,
	backoffFactor float64,
	jitter bool,
) *RetryManager {
	return &RetryManager{
		MaxRetries:    maxRetries,
		BaseDelay:     baseDelay,
		MaxDelay:      maxDelay,
		BackoffFactor: backoffFactor,
		Jitter:        jitter,
	}
}

// ShouldRetry determines whether an error should be retried based on the error type and attempt number.
func (rm *RetryManager) ShouldRetry(err error, attempt int) bool {
	// Don't retry if we've exceeded the maximum number of retries
	if attempt >= rm.MaxRetries {
		return false
	}

	// Check if the error is retryable using the error package's logic
	return errors.IsRetryable(err)
}

// NextDelay calculates the delay for the next retry attempt using exponential backoff.
func (rm *RetryManager) NextDelay(attempt int) time.Duration {
	if attempt < 0 {
		return rm.BaseDelay
	}

	// For very large attempt numbers, avoid overflow by returning MaxDelay early
	if attempt > 50 {
		delay := rm.MaxDelay
		if rm.Jitter {
			delay = rm.addJitter(delay)
		}
		return delay
	}

	// Calculate exponential backoff: baseDelay * (backoffFactor ^ attempt)
	power := math.Pow(rm.BackoffFactor, float64(attempt))

	// Check for potential overflow before converting to Duration
	if power > float64(rm.MaxDelay)/float64(rm.BaseDelay) {
		delay := rm.MaxDelay
		if rm.Jitter {
			delay = rm.addJitter(delay)
		}
		return delay
	}

	delay := time.Duration(float64(rm.BaseDelay) * power)

	// Apply maximum delay cap
	if delay > rm.MaxDelay || delay < 0 { // Check for negative values (overflow)
		delay = rm.MaxDelay
	}

	// Apply jitter if enabled
	if rm.Jitter {
		delay = rm.addJitter(delay)
	}

	return delay
}

// addJitter adds randomness to the delay to prevent thundering herd problems.
func (rm *RetryManager) addJitter(delay time.Duration) time.Duration {
	// Add up to 10% jitter (±5%)
	jitterRange := 0.1
	// #nosec G404 -- Jitter for retry delays doesn't require cryptographic randomness
	jitter := time.Duration(float64(delay) * jitterRange * (rand.Float64()*2 - 1))
	jitteredDelay := delay + jitter

	// Ensure the delay doesn't become negative
	if jitteredDelay < 0 {
		jitteredDelay = delay
	}

	return jitteredDelay
}

// ExecuteWithRetry executes an operation with retry logic using the manager's configuration.
func (rm *RetryManager) ExecuteWithRetry(ctx context.Context, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= rm.MaxRetries; attempt++ {
		// Check if context is cancelled before attempting
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the operation
		err := operation()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry this error and if we have attempts left
		if !rm.ShouldRetry(err, attempt) {
			return fmt.Errorf(
				"operation failed after %d attempt(s) (non-retryable error): %w",
				attempt+1,
				err,
			)
		}

		// Check if this was the last attempt
		if attempt >= rm.MaxRetries {
			break
		}

		// Calculate delay and wait before next attempt
		delay := rm.NextDelay(attempt)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retries exhausted
	return fmt.Errorf("operation failed after %d attempt(s): %w", rm.MaxRetries+1, lastErr)
}

// ExecuteWithRetryCallback executes an operation with retry logic and calls a callback on each retry.
func (rm *RetryManager) ExecuteWithRetryCallback(
	ctx context.Context,
	operation func() error,
	onRetry func(attempt int, err error, nextDelay time.Duration),
) error {
	var lastErr error

	for attempt := 0; attempt <= rm.MaxRetries; attempt++ {
		// Check if context is cancelled before attempting
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		// Execute the operation
		err := operation()
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Check if we should retry this error and if we have attempts left
		if !rm.ShouldRetry(err, attempt) {
			return fmt.Errorf(
				"operation failed after %d attempt(s) (non-retryable error): %w",
				attempt+1,
				err,
			)
		}

		// Check if this was the last attempt
		if attempt >= rm.MaxRetries {
			break
		}

		// Calculate delay for next attempt
		delay := rm.NextDelay(attempt)

		// Call the retry callback if provided
		if onRetry != nil {
			onRetry(attempt+1, err, delay)
		}

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retries exhausted
	return fmt.Errorf("operation failed after %d attempt(s): %w", rm.MaxRetries+1, lastErr)
}

// WithMaxRetries returns a new RetryManager with the specified maximum retries.
func (rm *RetryManager) WithMaxRetries(maxRetries int) *RetryManager {
	newManager := *rm
	newManager.MaxRetries = maxRetries

	return &newManager
}

// WithBaseDelay returns a new RetryManager with the specified base delay.
func (rm *RetryManager) WithBaseDelay(baseDelay time.Duration) *RetryManager {
	newManager := *rm
	newManager.BaseDelay = baseDelay

	return &newManager
}

// WithMaxDelay returns a new RetryManager with the specified maximum delay.
func (rm *RetryManager) WithMaxDelay(maxDelay time.Duration) *RetryManager {
	newManager := *rm
	newManager.MaxDelay = maxDelay

	return &newManager
}

// WithBackoffFactor returns a new RetryManager with the specified backoff factor.
func (rm *RetryManager) WithBackoffFactor(factor float64) *RetryManager {
	newManager := *rm
	newManager.BackoffFactor = factor

	return &newManager
}

// WithJitter returns a new RetryManager with jitter enabled or disabled.
func (rm *RetryManager) WithJitter(enabled bool) *RetryManager {
	newManager := *rm
	newManager.Jitter = enabled

	return &newManager
}

// Stats holds statistics about retry operations.
type Stats struct {
	TotalAttempts int           // Total number of attempts made
	TotalDelay    time.Duration // Total time spent waiting between retries
	LastError     error         // The last error encountered
	Succeeded     bool          // Whether the operation ultimately succeeded
}

// ExecuteWithRetryAndStats executes an operation with retry logic and returns detailed statistics.
func (rm *RetryManager) ExecuteWithRetryAndStats(
	ctx context.Context,
	operation func() error,
) (*Stats, error) {
	stats := &Stats{}

	var (
		totalDelay time.Duration
		lastErr    error
	)

	for attempt := 0; attempt <= rm.MaxRetries; attempt++ {
		stats.TotalAttempts++

		// Check if context is cancelled before attempting
		select {
		case <-ctx.Done():
			stats.LastError = ctx.Err()
			return stats, ctx.Err()
		default:
		}

		// Execute the operation
		err := operation()
		if err == nil {
			stats.Succeeded = true
			stats.TotalDelay = totalDelay
			stats.LastError = nil // Clear error on success

			return stats, nil // Success
		}

		lastErr = err
		stats.LastError = err

		// Check if we should retry this error and if we have attempts left
		if !rm.ShouldRetry(err, attempt) {
			stats.TotalDelay = totalDelay
			return stats, fmt.Errorf(
				"operation failed after %d attempt(s) (non-retryable error): %w",
				attempt+1,
				err,
			)
		}

		// Check if this was the last attempt
		if attempt >= rm.MaxRetries {
			break
		}

		// Calculate delay and wait before next attempt
		delay := rm.NextDelay(attempt)
		totalDelay += delay

		select {
		case <-ctx.Done():
			stats.LastError = ctx.Err()
			stats.TotalDelay = totalDelay

			return stats, ctx.Err()
		case <-time.After(delay):
			// Continue to next attempt
		}
	}

	// All retries exhausted
	stats.TotalDelay = totalDelay

	return stats, fmt.Errorf("operation failed after %d attempt(s): %w", rm.MaxRetries+1, lastErr)
}

// RetryableOperation wraps an operation to make it retryable with the manager.
type RetryableOperation struct {
	manager   *RetryManager
	operation func() error
}

// NewRetryableOperation creates a new retryable operation.
func (rm *RetryManager) NewRetryableOperation(operation func() error) *RetryableOperation {
	return &RetryableOperation{
		manager:   rm,
		operation: operation,
	}
}

// Execute runs the retryable operation.
func (ro *RetryableOperation) Execute(ctx context.Context) error {
	return ro.manager.ExecuteWithRetry(ctx, ro.operation)
}

// ExecuteWithCallback runs the retryable operation with a retry callback.
func (ro *RetryableOperation) ExecuteWithCallback(
	ctx context.Context,
	onRetry func(attempt int, err error, nextDelay time.Duration),
) error {
	return ro.manager.ExecuteWithRetryCallback(ctx, ro.operation, onRetry)
}

// ExecuteWithStats runs the retryable operation and returns statistics.
func (ro *RetryableOperation) ExecuteWithStats(ctx context.Context) (*Stats, error) {
	return ro.manager.ExecuteWithRetryAndStats(ctx, ro.operation)
}

// Predefined retry managers for common scenarios

// DefaultRetryManager returns a retry manager with sensible defaults for general use.
func DefaultRetryManager() *RetryManager {
	return NewRetryManager()
}

// NetworkRetryManager returns a retry manager optimized for network operations.
func NetworkRetryManager() *RetryManager {
	return &RetryManager{
		MaxRetries:    4,
		BaseDelay:     1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
	}
}

// FileSystemRetryManager returns a retry manager optimized for file system operations.
func FileSystemRetryManager() *RetryManager {
	return &RetryManager{
		MaxRetries:    3,
		BaseDelay:     100 * time.Millisecond,
		MaxDelay:      2 * time.Second,
		BackoffFactor: 1.5,
		Jitter:        false,
	}
}

// AggressiveRetryManager returns a retry manager with more aggressive retry attempts.
func AggressiveRetryManager() *RetryManager {
	return &RetryManager{
		MaxRetries:    6,
		BaseDelay:     500 * time.Millisecond,
		MaxDelay:      15 * time.Second,
		BackoffFactor: 1.5,
		Jitter:        true,
	}
}

// ConservativeRetryManager returns a retry manager with fewer, more spaced-out retry attempts.
func ConservativeRetryManager() *RetryManager {
	return &RetryManager{
		MaxRetries:    2,
		BaseDelay:     500 * time.Millisecond,
		MaxDelay:      2 * time.Minute,
		BackoffFactor: 3.0,
		Jitter:        true,
	}
}
