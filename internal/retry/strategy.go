// Package retry provides retry strategies and mechanisms for handling transient failures.
package retry

import (
	"context"
	"math"
	"math/rand"
	"time"

	"github.com/forest6511/gdl/pkg/errors"
)

// Strategy defines the interface for retry strategies.
type Strategy interface {
	// ShouldRetry determines if an operation should be retried based on the error and attempt number
	ShouldRetry(err error, attempt int) bool

	// NextDelay calculates the delay before the next retry attempt
	NextDelay(attempt int) time.Duration

	// MaxAttempts returns the maximum number of retry attempts allowed
	MaxAttempts() int
}

// BackoffType represents the type of backoff strategy.
type BackoffType int

const (
	// ExponentialBackoff increases delay exponentially with each attempt.
	ExponentialBackoff BackoffType = iota

	// LinearBackoff increases delay linearly with each attempt.
	LinearBackoff

	// ConstantBackoff uses a constant delay between attempts.
	ConstantBackoff

	// CustomBackoff uses custom intervals provided by the user.
	CustomBackoff
)

// ExponentialStrategy implements exponential backoff with optional jitter.
type ExponentialStrategy struct {
	MaxRetries    int
	BaseDelay     time.Duration
	MaxDelay      time.Duration
	BackoffFactor float64
	Jitter        bool
	JitterFactor  float64 // How much randomness to add (0.0 to 1.0)
	RetryChecker  func(error) bool
}

// NewExponentialStrategy creates a new exponential backoff strategy with sensible defaults.
func NewExponentialStrategy() *ExponentialStrategy {
	return &ExponentialStrategy{
		MaxRetries:    3,
		BaseDelay:     1 * time.Second,
		MaxDelay:      30 * time.Second,
		BackoffFactor: 2.0,
		Jitter:        true,
		JitterFactor:  0.1,
		RetryChecker:  errors.IsRetryable,
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func (s *ExponentialStrategy) WithMaxRetries(maxRetries int) *ExponentialStrategy {
	newStrategy := *s
	newStrategy.MaxRetries = maxRetries

	return &newStrategy
}

// WithBaseDelay sets the base delay for the first retry.
func (s *ExponentialStrategy) WithBaseDelay(baseDelay time.Duration) *ExponentialStrategy {
	newStrategy := *s
	newStrategy.BaseDelay = baseDelay

	return &newStrategy
}

// WithMaxDelay sets the maximum delay between retries.
func (s *ExponentialStrategy) WithMaxDelay(maxDelay time.Duration) *ExponentialStrategy {
	newStrategy := *s
	newStrategy.MaxDelay = maxDelay

	return &newStrategy
}

// WithBackoffFactor sets the multiplier for exponential backoff.
func (s *ExponentialStrategy) WithBackoffFactor(factor float64) *ExponentialStrategy {
	newStrategy := *s
	newStrategy.BackoffFactor = factor

	return &newStrategy
}

// WithJitter enables or disables jitter and sets the jitter factor.
func (s *ExponentialStrategy) WithJitter(enabled bool, factor float64) *ExponentialStrategy {
	newStrategy := *s
	newStrategy.Jitter = enabled
	newStrategy.JitterFactor = factor

	return &newStrategy
}

// WithRetryChecker sets a custom function to determine if an error should be retried.
func (s *ExponentialStrategy) WithRetryChecker(checker func(error) bool) *ExponentialStrategy {
	newStrategy := *s
	newStrategy.RetryChecker = checker

	return &newStrategy
}

// ShouldRetry determines if an operation should be retried.
func (s *ExponentialStrategy) ShouldRetry(err error, attempt int) bool {
	if attempt >= s.MaxRetries {
		return false
	}

	if s.RetryChecker != nil {
		return s.RetryChecker(err)
	}

	return errors.IsRetryable(err)
}

// NextDelay calculates the delay for the next retry attempt using exponential backoff.
func (s *ExponentialStrategy) NextDelay(attempt int) time.Duration {
	if attempt <= 0 {
		return s.BaseDelay
	}

	// For very large attempt numbers, avoid overflow by returning MaxDelay early
	if attempt > 50 {
		delay := s.MaxDelay
		if s.Jitter {
			delay = s.addJitter(delay)
		}
		return delay
	}

	// Calculate exponential delay: baseDelay * (backoffFactor ^ attempt)
	power := math.Pow(s.BackoffFactor, float64(attempt))

	// Check for potential overflow before converting to Duration
	if power > float64(s.MaxDelay)/float64(s.BaseDelay) {
		delay := s.MaxDelay
		if s.Jitter {
			delay = s.addJitter(delay)
		}
		return delay
	}

	delay := time.Duration(float64(s.BaseDelay) * power)

	// Apply maximum delay cap
	if delay > s.MaxDelay || delay < 0 { // Check for negative values (overflow)
		delay = s.MaxDelay
	}

	// Apply jitter if enabled
	if s.Jitter {
		delay = s.addJitter(delay)
	}

	return delay
}

// MaxAttempts returns the maximum number of retry attempts.
func (s *ExponentialStrategy) MaxAttempts() int {
	return s.MaxRetries
}

// addJitter adds randomness to the delay to prevent thundering herd problems.
func (s *ExponentialStrategy) addJitter(delay time.Duration) time.Duration {
	// #nosec G404 - Jitter for retry delays doesn't require cryptographic randomness
	jitter := time.Duration(float64(delay) * s.JitterFactor * (rand.Float64()*2 - 1))
	jitteredDelay := delay + jitter

	// Ensure the delay doesn't become negative or exceed the original delay by too much
	if jitteredDelay < delay/2 {
		jitteredDelay = delay / 2
	}

	if jitteredDelay > delay*2 {
		jitteredDelay = delay * 2
	}

	return jitteredDelay
}

// LinearStrategy implements linear backoff.
type LinearStrategy struct {
	MaxRetries   int
	BaseDelay    time.Duration
	MaxDelay     time.Duration
	Increment    time.Duration
	Jitter       bool
	JitterFactor float64
	RetryChecker func(error) bool
}

// NewLinearStrategy creates a new linear backoff strategy.
func NewLinearStrategy() *LinearStrategy {
	return &LinearStrategy{
		MaxRetries:   5,
		BaseDelay:    1 * time.Second,
		MaxDelay:     60 * time.Second,
		Increment:    2 * time.Second,
		Jitter:       false,
		JitterFactor: 0.1,
		RetryChecker: errors.IsRetryable,
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func (s *LinearStrategy) WithMaxRetries(maxRetries int) *LinearStrategy {
	s.MaxRetries = maxRetries
	return s
}

// WithBaseDelay sets the base delay for the first retry.
func (s *LinearStrategy) WithBaseDelay(baseDelay time.Duration) *LinearStrategy {
	s.BaseDelay = baseDelay
	return s
}

// WithMaxDelay sets the maximum delay between retries.
func (s *LinearStrategy) WithMaxDelay(maxDelay time.Duration) *LinearStrategy {
	s.MaxDelay = maxDelay
	return s
}

// WithIncrement sets the linear increment for each retry.
func (s *LinearStrategy) WithIncrement(increment time.Duration) *LinearStrategy {
	s.Increment = increment
	return s
}

// WithJitter enables or disables jitter.
func (s *LinearStrategy) WithJitter(enabled bool, factor float64) *LinearStrategy {
	s.Jitter = enabled
	s.JitterFactor = factor

	return s
}

// WithRetryChecker sets a custom function to determine if an error should be retried.
func (s *LinearStrategy) WithRetryChecker(checker func(error) bool) *LinearStrategy {
	s.RetryChecker = checker
	return s
}

// ShouldRetry determines if an operation should be retried.
func (s *LinearStrategy) ShouldRetry(err error, attempt int) bool {
	if attempt >= s.MaxRetries {
		return false
	}

	if s.RetryChecker != nil {
		return s.RetryChecker(err)
	}

	return errors.IsRetryable(err)
}

// NextDelay calculates the delay for the next retry attempt using linear backoff.
func (s *LinearStrategy) NextDelay(attempt int) time.Duration {
	// Linear increase: baseDelay + (increment * attempt)
	delay := s.BaseDelay + time.Duration(attempt)*s.Increment

	// Apply maximum delay cap
	if delay > s.MaxDelay {
		delay = s.MaxDelay
	}

	// Apply jitter if enabled
	if s.Jitter {
		// #nosec G404 -- Jitter for retry delays doesn't require cryptographic randomness
		jitter := time.Duration(float64(delay) * s.JitterFactor * (rand.Float64()*2 - 1))
		delay += jitter

		// Ensure delay doesn't become negative
		if delay < 0 {
			delay = s.BaseDelay
		}
	}

	return delay
}

// MaxAttempts returns the maximum number of retry attempts.
func (s *LinearStrategy) MaxAttempts() int {
	return s.MaxRetries
}

// ConstantStrategy implements constant delay backoff.
type ConstantStrategy struct {
	MaxRetries   int
	Delay        time.Duration
	Jitter       bool
	JitterFactor float64
	RetryChecker func(error) bool
}

// NewConstantStrategy creates a new constant delay strategy.
func NewConstantStrategy() *ConstantStrategy {
	return &ConstantStrategy{
		MaxRetries:   3,
		Delay:        5 * time.Second,
		Jitter:       false,
		JitterFactor: 0.1,
		RetryChecker: errors.IsRetryable,
	}
}

// WithMaxRetries sets the maximum number of retry attempts.
func (s *ConstantStrategy) WithMaxRetries(maxRetries int) *ConstantStrategy {
	s.MaxRetries = maxRetries
	return s
}

// WithDelay sets the constant delay between retries.
func (s *ConstantStrategy) WithDelay(delay time.Duration) *ConstantStrategy {
	s.Delay = delay
	return s
}

// WithJitter enables or disables jitter.
func (s *ConstantStrategy) WithJitter(enabled bool, factor float64) *ConstantStrategy {
	s.Jitter = enabled
	s.JitterFactor = factor

	return s
}

// WithRetryChecker sets a custom function to determine if an error should be retried.
func (s *ConstantStrategy) WithRetryChecker(checker func(error) bool) *ConstantStrategy {
	s.RetryChecker = checker
	return s
}

// ShouldRetry determines if an operation should be retried.
func (s *ConstantStrategy) ShouldRetry(err error, attempt int) bool {
	if attempt >= s.MaxRetries {
		return false
	}

	if s.RetryChecker != nil {
		return s.RetryChecker(err)
	}

	return errors.IsRetryable(err)
}

// NextDelay returns the constant delay.
func (s *ConstantStrategy) NextDelay(attempt int) time.Duration {
	delay := s.Delay

	// Apply jitter if enabled
	if s.Jitter {
		// #nosec G404 -- Jitter for retry delays doesn't require cryptographic randomness
		jitter := time.Duration(float64(delay) * s.JitterFactor * (rand.Float64()*2 - 1))
		delay += jitter

		// Ensure delay doesn't become negative
		if delay < 0 {
			delay = s.Delay
		}
	}

	return delay
}

// MaxAttempts returns the maximum number of retry attempts.
func (s *ConstantStrategy) MaxAttempts() int {
	return s.MaxRetries
}

// CustomStrategy implements custom retry intervals.
type CustomStrategy struct {
	Intervals    []time.Duration
	RetryChecker func(error) bool
}

// NewCustomStrategy creates a new custom interval strategy.
func NewCustomStrategy(intervals []time.Duration) *CustomStrategy {
	if len(intervals) == 0 {
		intervals = []time.Duration{1 * time.Second, 2 * time.Second, 5 * time.Second}
	}

	return &CustomStrategy{
		Intervals:    intervals,
		RetryChecker: errors.IsRetryable,
	}
}

// WithRetryChecker sets a custom function to determine if an error should be retried.
func (s *CustomStrategy) WithRetryChecker(checker func(error) bool) *CustomStrategy {
	s.RetryChecker = checker
	return s
}

// ShouldRetry determines if an operation should be retried.
func (s *CustomStrategy) ShouldRetry(err error, attempt int) bool {
	if attempt >= len(s.Intervals) {
		return false
	}

	if s.RetryChecker != nil {
		return s.RetryChecker(err)
	}

	return errors.IsRetryable(err)
}

// NextDelay returns the delay for the specified attempt.
func (s *CustomStrategy) NextDelay(attempt int) time.Duration {
	if attempt < len(s.Intervals) {
		return s.Intervals[attempt]
	}

	// Return the last interval if we've exceeded the defined intervals
	if len(s.Intervals) > 0 {
		return s.Intervals[len(s.Intervals)-1]
	}

	return 1 * time.Second // Fallback
}

// MaxAttempts returns the maximum number of retry attempts.
func (s *CustomStrategy) MaxAttempts() int {
	return len(s.Intervals)
}

// StrategyBuilder provides a fluent interface for building retry strategies.
type StrategyBuilder struct {
	backoffType   BackoffType
	maxRetries    int
	baseDelay     time.Duration
	maxDelay      time.Duration
	backoffFactor float64
	increment     time.Duration
	jitter        bool
	jitterFactor  float64
	intervals     []time.Duration
	retryChecker  func(error) bool
}

// NewStrategyBuilder creates a new strategy builder.
func NewStrategyBuilder() *StrategyBuilder {
	return &StrategyBuilder{
		backoffType:   ExponentialBackoff,
		maxRetries:    3,
		baseDelay:     1 * time.Second,
		maxDelay:      30 * time.Second,
		backoffFactor: 2.0,
		increment:     2 * time.Second,
		jitter:        true,
		jitterFactor:  0.1,
		retryChecker:  errors.IsRetryable,
	}
}

// WithBackoffType sets the backoff type.
func (b *StrategyBuilder) WithBackoffType(backoffType BackoffType) *StrategyBuilder {
	b.backoffType = backoffType
	return b
}

// WithMaxRetries sets the maximum number of retries.
func (b *StrategyBuilder) WithMaxRetries(maxRetries int) *StrategyBuilder {
	b.maxRetries = maxRetries
	return b
}

// WithBaseDelay sets the base delay.
func (b *StrategyBuilder) WithBaseDelay(baseDelay time.Duration) *StrategyBuilder {
	b.baseDelay = baseDelay
	return b
}

// WithMaxDelay sets the maximum delay.
func (b *StrategyBuilder) WithMaxDelay(maxDelay time.Duration) *StrategyBuilder {
	b.maxDelay = maxDelay
	return b
}

// WithBackoffFactor sets the backoff factor for exponential backoff.
func (b *StrategyBuilder) WithBackoffFactor(factor float64) *StrategyBuilder {
	b.backoffFactor = factor
	return b
}

// WithIncrement sets the increment for linear backoff.
func (b *StrategyBuilder) WithIncrement(increment time.Duration) *StrategyBuilder {
	b.increment = increment
	return b
}

// WithJitter enables jitter with the specified factor.
func (b *StrategyBuilder) WithJitter(enabled bool, factor float64) *StrategyBuilder {
	b.jitter = enabled
	b.jitterFactor = factor

	return b
}

// WithCustomIntervals sets custom intervals for custom backoff.
func (b *StrategyBuilder) WithCustomIntervals(intervals []time.Duration) *StrategyBuilder {
	b.intervals = intervals
	return b
}

// WithRetryChecker sets a custom retry checker function.
func (b *StrategyBuilder) WithRetryChecker(checker func(error) bool) *StrategyBuilder {
	b.retryChecker = checker
	return b
}

// Build creates the configured retry strategy.
func (b *StrategyBuilder) Build() Strategy {
	switch b.backoffType {
	case ExponentialBackoff:
		return &ExponentialStrategy{
			MaxRetries:    b.maxRetries,
			BaseDelay:     b.baseDelay,
			MaxDelay:      b.maxDelay,
			BackoffFactor: b.backoffFactor,
			Jitter:        b.jitter,
			JitterFactor:  b.jitterFactor,
			RetryChecker:  b.retryChecker,
		}

	case LinearBackoff:
		return &LinearStrategy{
			MaxRetries:   b.maxRetries,
			BaseDelay:    b.baseDelay,
			MaxDelay:     b.maxDelay,
			Increment:    b.increment,
			Jitter:       b.jitter,
			JitterFactor: b.jitterFactor,
			RetryChecker: b.retryChecker,
		}

	case ConstantBackoff:
		return &ConstantStrategy{
			MaxRetries:   b.maxRetries,
			Delay:        b.baseDelay,
			Jitter:       b.jitter,
			JitterFactor: b.jitterFactor,
			RetryChecker: b.retryChecker,
		}

	case CustomBackoff:
		intervals := b.intervals
		if len(intervals) == 0 {
			intervals = []time.Duration{b.baseDelay}
		}

		return &CustomStrategy{
			Intervals:    intervals,
			RetryChecker: b.retryChecker,
		}

	default:
		return NewExponentialStrategy()
	}
}

// Predefined strategies for common use cases

// DefaultStrategy returns a default exponential backoff strategy suitable for most use cases.
func DefaultStrategy() Strategy {
	return NewExponentialStrategy()
}

// AggressiveStrategy returns a strategy with shorter delays for time-sensitive operations.
func AggressiveStrategy() Strategy {
	return NewExponentialStrategy().
		WithMaxRetries(5).
		WithBaseDelay(500 * time.Millisecond).
		WithMaxDelay(10 * time.Second).
		WithBackoffFactor(1.5)
}

// ConservativeStrategy returns a strategy with longer delays for less time-sensitive operations.
func ConservativeStrategy() Strategy {
	return NewExponentialStrategy().
		WithMaxRetries(3).
		WithBaseDelay(2 * time.Second).
		WithMaxDelay(2 * time.Minute).
		WithBackoffFactor(3.0)
}

// NetworkStrategy returns a strategy optimized for network operations.
func NetworkStrategy() Strategy {
	return NewExponentialStrategy().
		WithMaxRetries(4).
		WithBaseDelay(1*time.Second).
		WithMaxDelay(30*time.Second).
		WithBackoffFactor(2.0).
		WithJitter(true, 0.25)
}

// FileSystemStrategy returns a strategy optimized for file system operations.
func FileSystemStrategy() Strategy {
	return NewLinearStrategy().
		WithMaxRetries(3).
		WithBaseDelay(100 * time.Millisecond).
		WithIncrement(200 * time.Millisecond).
		WithMaxDelay(2 * time.Second)
}

// ExecuteWithRetry is a convenience function that executes an operation with the specified strategy.
func ExecuteWithRetry(ctx context.Context, strategy Strategy, operation func() error) error {
	var lastErr error

	for attempt := 0; attempt <= strategy.MaxAttempts(); attempt++ {
		// Check context before attempting
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

		// Check if we should retry
		if !strategy.ShouldRetry(err, attempt) {
			break
		}

		// Check if this was the last attempt
		if attempt >= strategy.MaxAttempts() {
			break
		}

		// Calculate delay and wait
		delay := strategy.NextDelay(attempt)
		timer := time.NewTimer(delay)

		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// Continue to next attempt
		}
	}

	return lastErr
}
