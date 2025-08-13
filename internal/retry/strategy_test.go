package retry

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"
	"time"

	"github.com/forest6511/godl/pkg/errors"
)

func TestExponentialStrategy(t *testing.T) {
	strategy := NewExponentialStrategy()

	tests := []struct {
		name            string
		attempt         int
		expectedDelay   time.Duration
		shouldBeAtLeast time.Duration
		shouldBeAtMost  time.Duration
	}{
		{
			name:            "First retry",
			attempt:         0,
			expectedDelay:   1 * time.Second,
			shouldBeAtLeast: 800 * time.Millisecond,
			shouldBeAtMost:  1200 * time.Millisecond,
		},
		{
			name:            "Second retry",
			attempt:         1,
			expectedDelay:   2 * time.Second,
			shouldBeAtLeast: 1600 * time.Millisecond,
			shouldBeAtMost:  2400 * time.Millisecond,
		},
		{
			name:            "Third retry",
			attempt:         2,
			expectedDelay:   4 * time.Second,
			shouldBeAtLeast: 3200 * time.Millisecond,
			shouldBeAtMost:  4800 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := strategy.NextDelay(tt.attempt)

			if delay < tt.shouldBeAtLeast || delay > tt.shouldBeAtMost {
				t.Errorf("NextDelay(%d) = %v, want between %v and %v",
					tt.attempt, delay, tt.shouldBeAtLeast, tt.shouldBeAtMost)
			}
		})
	}
}

func TestExponentialStrategy_NoJitter(t *testing.T) {
	strategy := NewExponentialStrategy().WithJitter(false, 0)

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"First retry", 0, 1 * time.Second},
		{"Second retry", 1, 2 * time.Second},
		{"Third retry", 2, 4 * time.Second},
		{"Fourth retry", 3, 8 * time.Second},
		{"Fifth retry", 4, 16 * time.Second},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := strategy.NextDelay(tt.attempt)
			if delay != tt.expected {
				t.Errorf("NextDelay(%d) = %v, want %v", tt.attempt, delay, tt.expected)
			}
		})
	}
}

func TestExponentialStrategy_MaxDelay(t *testing.T) {
	strategy := NewExponentialStrategy().
		WithMaxDelay(5*time.Second).
		WithJitter(false, 0)

	// High attempt number should be capped at MaxDelay
	delay := strategy.NextDelay(10)
	if delay != 5*time.Second {
		t.Errorf("NextDelay(10) = %v, want %v (max delay)", delay, 5*time.Second)
	}
}

func TestLinearStrategy(t *testing.T) {
	strategy := NewLinearStrategy().
		WithBaseDelay(1*time.Second).
		WithIncrement(2*time.Second).
		WithJitter(false, 0)

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"First retry", 0, 1 * time.Second},
		{"Second retry", 1, 3 * time.Second}, // 1 + 2*1
		{"Third retry", 2, 5 * time.Second},  // 1 + 2*2
		{"Fourth retry", 3, 7 * time.Second}, // 1 + 2*3
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := strategy.NextDelay(tt.attempt)
			if delay != tt.expected {
				t.Errorf("NextDelay(%d) = %v, want %v", tt.attempt, delay, tt.expected)
			}
		})
	}
}

func TestConstantStrategy(t *testing.T) {
	delay := 3 * time.Second
	strategy := NewConstantStrategy().
		WithDelay(delay).
		WithJitter(false, 0)

	for attempt := 0; attempt < 5; attempt++ {
		actualDelay := strategy.NextDelay(attempt)
		if actualDelay != delay {
			t.Errorf("NextDelay(%d) = %v, want %v", attempt, actualDelay, delay)
		}
	}
}

func TestCustomStrategy(t *testing.T) {
	intervals := []time.Duration{
		1 * time.Second,
		3 * time.Second,
		5 * time.Second,
		10 * time.Second,
	}

	strategy := NewCustomStrategy(intervals)

	for i, expectedDelay := range intervals {
		actualDelay := strategy.NextDelay(i)
		if actualDelay != expectedDelay {
			t.Errorf("NextDelay(%d) = %v, want %v", i, actualDelay, expectedDelay)
		}
	}

	// Test beyond defined intervals - should return last interval
	lastDelay := strategy.NextDelay(len(intervals))
	if lastDelay != intervals[len(intervals)-1] {
		t.Errorf("NextDelay(%d) = %v, want %v (last interval)",
			len(intervals), lastDelay, intervals[len(intervals)-1])
	}
}

func TestStrategy_ShouldRetry(t *testing.T) {
	strategy := NewExponentialStrategy().WithMaxRetries(3)

	retryableErr := stderrors.New("temporary network error")
	nonRetryableErr := stderrors.New("permanent error")

	// Mock the retry checker to make first error retryable and second non-retryable
	strategy = strategy.WithRetryChecker(func(err error) bool {
		return err.Error() == "temporary network error"
	})

	tests := []struct {
		name        string
		err         error
		attempt     int
		shouldRetry bool
	}{
		{"Retryable error, first attempt", retryableErr, 0, true},
		{"Retryable error, second attempt", retryableErr, 1, true},
		{"Retryable error, third attempt", retryableErr, 2, true},
		{"Retryable error, max attempts reached", retryableErr, 3, false},
		{"Non-retryable error, first attempt", nonRetryableErr, 0, false},
		{"Non-retryable error, second attempt", nonRetryableErr, 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := strategy.ShouldRetry(tt.err, tt.attempt)
			if result != tt.shouldRetry {
				t.Errorf("ShouldRetry(%v, %d) = %v, want %v",
					tt.err, tt.attempt, result, tt.shouldRetry)
			}
		})
	}
}

func TestStrategyBuilder(t *testing.T) {
	strategy := NewStrategyBuilder().
		WithBackoffType(ExponentialBackoff).
		WithMaxRetries(5).
		WithBaseDelay(500*time.Millisecond).
		WithMaxDelay(10*time.Second).
		WithBackoffFactor(1.5).
		WithJitter(false, 0).
		Build()

	expStrategy, ok := strategy.(*ExponentialStrategy)
	if !ok {
		t.Fatal("Expected ExponentialStrategy")
	}

	if expStrategy.MaxRetries != 5 {
		t.Errorf("MaxRetries = %d, want 5", expStrategy.MaxRetries)
	}

	if expStrategy.BaseDelay != 500*time.Millisecond {
		t.Errorf("BaseDelay = %v, want %v", expStrategy.BaseDelay, 500*time.Millisecond)
	}

	if expStrategy.MaxDelay != 10*time.Second {
		t.Errorf("MaxDelay = %v, want %v", expStrategy.MaxDelay, 10*time.Second)
	}

	if expStrategy.BackoffFactor != 1.5 {
		t.Errorf("BackoffFactor = %f, want 1.5", expStrategy.BackoffFactor)
	}

	if expStrategy.Jitter != false {
		t.Errorf("Jitter = %v, want false", expStrategy.Jitter)
	}
}

func TestPredefinedStrategies(t *testing.T) {
	strategies := map[string]Strategy{
		"Default":      DefaultStrategy(),
		"Aggressive":   AggressiveStrategy(),
		"Conservative": ConservativeStrategy(),
		"Network":      NetworkStrategy(),
		"FileSystem":   FileSystemStrategy(),
	}

	for name, strategy := range strategies {
		t.Run(name, func(t *testing.T) {
			if strategy == nil {
				t.Fatal("Strategy should not be nil")
			}

			maxAttempts := strategy.MaxAttempts()
			if maxAttempts <= 0 {
				t.Errorf("MaxAttempts() = %d, want > 0", maxAttempts)
			}

			delay := strategy.NextDelay(0)
			if delay <= 0 {
				t.Errorf("NextDelay(0) = %v, want > 0", delay)
			}
		})
	}
}

func TestExecuteWithRetry(t *testing.T) {
	t.Run("Success on first try", func(t *testing.T) {
		strategy := NewExponentialStrategy().WithMaxRetries(3)

		callCount := 0
		operation := func() error {
			callCount++
			return nil // Success
		}

		ctx := context.Background()

		err := ExecuteWithRetry(ctx, strategy, operation)
		if err != nil {
			t.Errorf("ExecuteWithRetry() = %v, want nil", err)
		}

		if callCount != 1 {
			t.Errorf("Operation called %d times, want 1", callCount)
		}
	})

	t.Run("Success after retries", func(t *testing.T) {
		strategy := NewExponentialStrategy().WithMaxRetries(3).WithBaseDelay(1 * time.Millisecond)

		callCount := 0
		operation := func() error {
			callCount++
			if callCount < 3 {
				return stderrors.New("temporary error") // Will be treated as retryable
			}

			return nil // Success on third try
		}

		// Mock retry checker to make all errors retryable
		strategy = strategy.WithRetryChecker(func(error) bool { return true })

		ctx := context.Background()

		err := ExecuteWithRetry(ctx, strategy, operation)
		if err != nil {
			t.Errorf("ExecuteWithRetry() = %v, want nil", err)
		}

		if callCount != 3 {
			t.Errorf("Operation called %d times, want 3", callCount)
		}
	})

	t.Run("Failure after max retries", func(t *testing.T) {
		strategy := NewExponentialStrategy().WithMaxRetries(2).WithBaseDelay(1 * time.Millisecond)

		callCount := 0
		operation := func() error {
			callCount++
			return stderrors.New("persistent error")
		}

		// Mock retry checker to make all errors retryable
		strategy = strategy.WithRetryChecker(func(error) bool { return true })

		ctx := context.Background()

		err := ExecuteWithRetry(ctx, strategy, operation)
		if err == nil {
			t.Error("ExecuteWithRetry() = nil, want error")
		}

		if callCount != 3 { // Initial attempt + 2 retries
			t.Errorf("Operation called %d times, want 3", callCount)
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		strategy := NewExponentialStrategy().WithMaxRetries(5).WithBaseDelay(100 * time.Millisecond)

		callCount := 0
		operation := func() error {
			callCount++
			return stderrors.New("error")
		}

		// Mock retry checker to make all errors retryable
		strategy = strategy.WithRetryChecker(func(error) bool { return true })

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()

		err := ExecuteWithRetry(ctx, strategy, operation)

		if !stderrors.Is(err, context.DeadlineExceeded) {
			t.Errorf("ExecuteWithRetry() = %v, want %v", err, context.DeadlineExceeded)
		}

		// Should have been called at least once but not all 6 times due to timeout
		if callCount == 0 || callCount > 6 {
			t.Errorf("Operation called %d times, want between 1 and 6", callCount)
		}
	})
}

func TestStrategyFluentInterface(t *testing.T) {
	// Test that all fluent methods return new instances (don't modify original)
	original := NewExponentialStrategy()

	modified := original.
		WithMaxRetries(10).
		WithBaseDelay(500*time.Millisecond).
		WithMaxDelay(1*time.Minute).
		WithBackoffFactor(3.0).
		WithJitter(false, 0)

	// Original should be unchanged
	if original.MaxRetries == 10 {
		t.Error("Original strategy was modified by fluent methods")
	}

	// Modified should have new values
	if modified.MaxRetries != 10 {
		t.Errorf("Modified strategy MaxRetries = %d, want 10", modified.MaxRetries)
	}

	if modified.BaseDelay != 500*time.Millisecond {
		t.Errorf(
			"Modified strategy BaseDelay = %v, want %v",
			modified.BaseDelay,
			500*time.Millisecond,
		)
	}
}

func TestLinearStrategy_Comprehensive(t *testing.T) {
	// Test LinearStrategy with retry checker
	strategy := NewLinearStrategy().
		WithMaxRetries(3).
		WithBaseDelay(100*time.Millisecond).
		WithMaxDelay(1*time.Second).
		WithIncrement(50*time.Millisecond).
		WithJitter(true, 0.1).
		WithRetryChecker(func(err error) bool {
			return err != nil && err.Error() != "permanent"
		})

	// Test ShouldRetry with custom checker
	temporaryErr := fmt.Errorf("temporary error")
	permanentErr := fmt.Errorf("permanent")

	if !strategy.ShouldRetry(temporaryErr, 1) {
		t.Error("Expected to retry temporary error")
	}

	if strategy.ShouldRetry(permanentErr, 1) {
		t.Error("Expected not to retry permanent error")
	}

	if strategy.ShouldRetry(temporaryErr, 4) {
		t.Error("Expected not to retry after max attempts")
	}

	// Test NextDelay with various scenarios
	delay1 := strategy.NextDelay(1)
	if delay1 < 100*time.Millisecond || delay1 > 200*time.Millisecond {
		t.Errorf("Expected delay around 100ms, got %v", delay1)
	}

	delay2 := strategy.NextDelay(2)
	if delay2 < 150*time.Millisecond || delay2 > 250*time.Millisecond {
		t.Errorf("Expected delay around 150ms, got %v", delay2)
	}

	// Test max delay constraint
	delay4 := strategy.NextDelay(4)
	if delay4 > 1*time.Second+100*time.Millisecond { // Allow for jitter
		t.Errorf("Expected delay not to exceed max delay, got %v", delay4)
	}
}

func TestConstantStrategy_Comprehensive(t *testing.T) {
	// Test ConstantStrategy with all features
	strategy := NewConstantStrategy().
		WithMaxRetries(5).
		WithDelay(200*time.Millisecond).
		WithJitter(true, 0.1).
		WithRetryChecker(func(err error) bool {
			return err != nil && err.Error() == "retryable"
		})

	// Test ShouldRetry with custom checker
	retryableErr := fmt.Errorf("retryable")
	nonRetryableErr := fmt.Errorf("non-retryable")

	if !strategy.ShouldRetry(retryableErr, 3) {
		t.Error("Expected to retry retryable error")
	}

	if strategy.ShouldRetry(nonRetryableErr, 1) {
		t.Error("Expected not to retry non-retryable error")
	}

	if strategy.ShouldRetry(retryableErr, 6) {
		t.Error("Expected not to retry after max attempts")
	}

	// Test NextDelay - should be constant with jitter
	delay1 := strategy.NextDelay(1)
	delay2 := strategy.NextDelay(2)
	delay3 := strategy.NextDelay(3)

	// All delays should be around 200ms ± jitter
	delays := []time.Duration{delay1, delay2, delay3}
	for i, delay := range delays {
		if delay < 150*time.Millisecond || delay > 250*time.Millisecond {
			t.Errorf("Expected delay %d around 200ms, got %v", i+1, delay)
		}
	}

	// Test MaxAttempts
	if strategy.MaxAttempts() != 5 {
		t.Errorf("Expected MaxAttempts 5, got %d", strategy.MaxAttempts())
	}
}

func TestCustomStrategy_Comprehensive(t *testing.T) {
	intervals := []time.Duration{
		100 * time.Millisecond,
		200 * time.Millisecond,
		500 * time.Millisecond,
		1 * time.Second,
	}

	// Test CustomStrategy with retry checker
	strategy := NewCustomStrategy(intervals).
		WithRetryChecker(func(err error) bool {
			return err != nil && err.Error() != "fatal"
		})

	// Test ShouldRetry
	retryableErr := fmt.Errorf("network timeout")
	fatalErr := fmt.Errorf("fatal")

	if !strategy.ShouldRetry(retryableErr, 2) {
		t.Error("Expected to retry retryable error")
	}

	if strategy.ShouldRetry(fatalErr, 1) {
		t.Error("Expected not to retry fatal error")
	}

	// Test NextDelay follows custom intervals
	delay1 := strategy.NextDelay(1)
	if delay1 != 200*time.Millisecond {
		t.Errorf("Expected first delay 200ms, got %v", delay1)
	}

	delay2 := strategy.NextDelay(2)
	if delay2 != 500*time.Millisecond {
		t.Errorf("Expected second delay 500ms, got %v", delay2)
	}

	delay4 := strategy.NextDelay(4)
	if delay4 != 1*time.Second {
		t.Errorf("Expected fourth delay 1s, got %v", delay4)
	}

	// Test delay beyond intervals (should use last interval)
	delay5 := strategy.NextDelay(5)
	if delay5 != 1*time.Second {
		t.Errorf("Expected fifth delay to use last interval 1s, got %v", delay5)
	}

	// Test MaxAttempts
	if strategy.MaxAttempts() != len(intervals) {
		t.Errorf("Expected MaxAttempts %d, got %d", len(intervals), strategy.MaxAttempts())
	}
}

func TestStrategyBuilder_Comprehensive(t *testing.T) {
	builder := NewStrategyBuilder()

	// Test building exponential strategy
	expStrategy := builder.
		WithBackoffType(ExponentialBackoff).
		WithMaxRetries(5).
		WithBaseDelay(100*time.Millisecond).
		WithMaxDelay(5*time.Second).
		WithBackoffFactor(2.0).
		WithJitter(true, 0.1).
		Build()

	if expStrategy == nil {
		t.Fatal("Expected non-nil exponential strategy")
	}

	// Test building linear strategy
	builder2 := NewStrategyBuilder()
	linStrategy := builder2.
		WithBackoffType(LinearBackoff).
		WithMaxRetries(3).
		WithBaseDelay(200*time.Millisecond).
		WithMaxDelay(2*time.Second).
		WithIncrement(100*time.Millisecond).
		WithJitter(false, 0.0).
		Build()

	if linStrategy == nil {
		t.Fatal("Expected non-nil linear strategy")
	}

	// Test building constant strategy
	builder3 := NewStrategyBuilder()
	constStrategy := builder3.
		WithBackoffType(ConstantBackoff).
		WithMaxRetries(10).
		WithBaseDelay(500*time.Millisecond).
		WithJitter(true, 0.2).
		Build()

	if constStrategy == nil {
		t.Fatal("Expected non-nil constant strategy")
	}

	// Test building custom strategy
	builder4 := NewStrategyBuilder()
	intervals := []time.Duration{50 * time.Millisecond, 100 * time.Millisecond}
	customStrategy := builder4.
		WithBackoffType(CustomBackoff).
		WithCustomIntervals(intervals).
		Build()

	if customStrategy == nil {
		t.Fatal("Expected non-nil custom strategy")
	}

	// Test WithRetryChecker
	builder7 := NewStrategyBuilder()
	strategy := builder7.
		WithBackoffType(ExponentialBackoff).
		WithRetryChecker(func(error) bool { return true }).
		Build()

	if strategy == nil {
		t.Fatal("Expected non-nil strategy")
	}
}

func TestRetryManager_ExecuteWithRetryAndStats_EdgeCases(t *testing.T) {
	ctx := context.Background()
	manager := NewRetryManager()

	// Test with operation that always succeeds
	successCount := 0
	successOp := func() error {
		successCount++
		return nil
	}

	stats, err := manager.ExecuteWithRetryAndStats(ctx, successOp)
	if err != nil {
		t.Errorf("Expected no error for successful operation: %v", err)
	}

	if successCount != 1 {
		t.Errorf("Expected operation to be called once, got %d", successCount)
	}

	if stats.TotalAttempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", stats.TotalAttempts)
	}

	if stats.TotalDelay != 0 {
		t.Errorf("Expected no delay for first success, got %v", stats.TotalDelay)
	}

	// Test with operation that fails then succeeds (using retryable errors)
	callCount := 0
	partialFailOp := func() error {
		callCount++
		if callCount < 3 {
			return &errors.DownloadError{
				Code:      errors.CodeNetworkError,
				Message:   "temporary failure",
				Retryable: true,
			}
		}

		return nil
	}

	stats2, err2 := manager.ExecuteWithRetryAndStats(ctx, partialFailOp)
	if err2 != nil {
		t.Errorf("Expected no error for eventually successful operation: %v", err2)
	}

	if stats2.TotalAttempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", stats2.TotalAttempts)
	}

	if stats2.TotalDelay == 0 {
		t.Error("Expected some delay for retried operation")
	}

	if !stats2.Succeeded {
		t.Error("Expected operation to eventually succeed")
	}
}

func TestRetryManager_ExecuteWithRetryCallback_EdgeCases(t *testing.T) {
	ctx := context.Background()
	manager := NewRetryManager().WithMaxRetries(2)

	callbackCount := 0
	callback := func(attempt int, err error, delay time.Duration) {
		callbackCount++

		if attempt < 1 {
			t.Errorf("Expected attempt >= 1, got %d", attempt)
		}

		if err == nil {
			t.Error("Expected error in callback")
		}

		if delay < 0 {
			t.Error("Expected non-negative delay")
		}
	}

	failCount := 0
	alwaysFailOp := func() error {
		failCount++

		return &errors.DownloadError{
			Code:      errors.CodeNetworkError,
			Message:   fmt.Sprintf("failure %d", failCount),
			Retryable: true,
		}
	}

	err := manager.ExecuteWithRetryCallback(ctx, alwaysFailOp, callback)
	if err == nil {
		t.Error("Expected error for always failing operation")
	}

	// Should have 3 total attempts (1 initial + 2 retries)
	if failCount != 3 {
		t.Errorf("Expected 3 attempts, got %d", failCount)
	}

	// Should have 2 callback calls (one for each retry)
	if callbackCount != 2 {
		t.Errorf("Expected 2 callback calls, got %d", callbackCount)
	}
}
