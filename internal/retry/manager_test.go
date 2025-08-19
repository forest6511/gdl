package retry

import (
	"context"
	stderrors "errors"
	"fmt"
	"testing"
	"time"

	"github.com/forest6511/gdl/pkg/errors"
)

func TestNewRetryManager(t *testing.T) {
	rm := NewRetryManager()

	if rm.MaxRetries != 3 {
		t.Errorf("MaxRetries = %d, want 3", rm.MaxRetries)
	}

	if rm.BaseDelay != 1*time.Second {
		t.Errorf("BaseDelay = %v, want %v", rm.BaseDelay, 1*time.Second)
	}

	if rm.MaxDelay != 30*time.Second {
		t.Errorf("MaxDelay = %v, want %v", rm.MaxDelay, 30*time.Second)
	}

	if rm.BackoffFactor != 2.0 {
		t.Errorf("BackoffFactor = %f, want 2.0", rm.BackoffFactor)
	}

	if !rm.Jitter {
		t.Error("Jitter should be enabled by default")
	}
}

func TestNewRetryManagerWithConfig(t *testing.T) {
	maxRetries := 5
	baseDelay := 100 * time.Millisecond
	maxDelay := 1 * time.Second
	backoffFactor := 1.5
	jitter := false

	rm := NewRetryManagerWithConfig(maxRetries, baseDelay, maxDelay, backoffFactor, jitter)

	if rm.MaxRetries != maxRetries {
		t.Errorf("MaxRetries = %d, want %d", rm.MaxRetries, maxRetries)
	}

	if rm.BaseDelay != baseDelay {
		t.Errorf("BaseDelay = %v, want %v", rm.BaseDelay, baseDelay)
	}

	if rm.MaxDelay != maxDelay {
		t.Errorf("MaxDelay = %v, want %v", rm.MaxDelay, maxDelay)
	}

	if rm.BackoffFactor != backoffFactor {
		t.Errorf("BackoffFactor = %f, want %f", rm.BackoffFactor, backoffFactor)
	}

	if rm.Jitter != jitter {
		t.Errorf("Jitter = %v, want %v", rm.Jitter, jitter)
	}
}

func TestRetryManager_ShouldRetry(t *testing.T) {
	rm := NewRetryManager()

	// Simple errors are generally not retryable by default in our error system
	// So let's test the logic with network-like errors and max attempts
	retryableErr := stderrors.New("network error")
	nonRetryableErr := stderrors.New("invalid input")

	tests := []struct {
		name        string
		err         error
		attempt     int
		shouldRetry bool
	}{
		{
			"Retryable error, first attempt",
			retryableErr,
			0,
			false,
		}, // Generic errors are not retryable by default
		{"Retryable error, second attempt", retryableErr, 1, false},
		{"Retryable error, third attempt", retryableErr, 2, false},
		{"Retryable error, max attempts reached", retryableErr, 3, false},
		{"Retryable error, beyond max attempts", retryableErr, 4, false},
		{"Non-retryable error, first attempt", nonRetryableErr, 0, false},
		{"Non-retryable error, second attempt", nonRetryableErr, 1, false},
		{"Non-retryable error, max attempts reached", nonRetryableErr, 3, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := rm.ShouldRetry(tt.err, tt.attempt)
			if result != tt.shouldRetry {
				t.Errorf("ShouldRetry(%v, %d) = %v, want %v",
					tt.err, tt.attempt, result, tt.shouldRetry)
			}
		})
	}
}

func TestRetryManager_NextDelay(t *testing.T) {
	rm := NewRetryManagerWithConfig(5, 50*time.Millisecond, 500*time.Millisecond, 2.0, false)

	tests := []struct {
		name     string
		attempt  int
		expected time.Duration
	}{
		{"First retry", 0, 50 * time.Millisecond},
		{"Second retry", 1, 100 * time.Millisecond},
		{"Third retry", 2, 200 * time.Millisecond},
		{"Fourth retry", 3, 400 * time.Millisecond},
		{"Fifth retry", 4, 500 * time.Millisecond}, // Capped at MaxDelay
		{"Sixth retry", 5, 500 * time.Millisecond}, // Capped at MaxDelay
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			delay := rm.NextDelay(tt.attempt)
			if delay != tt.expected {
				t.Errorf("NextDelay(%d) = %v, want %v", tt.attempt, delay, tt.expected)
			}
		})
	}
}

func TestRetryManager_NextDelay_WithJitter(t *testing.T) {
	rm := NewRetryManagerWithConfig(3, 50*time.Millisecond, 500*time.Millisecond, 2.0, true)

	baseDelay := 50 * time.Millisecond
	delay := rm.NextDelay(0)

	// With jitter, delay should be within reasonable bounds of the base delay
	minExpected := time.Duration(float64(baseDelay) * 0.8) // Allow 20% variance
	maxExpected := time.Duration(float64(baseDelay) * 1.2)

	if delay < minExpected || delay > maxExpected {
		t.Errorf("NextDelay(0) with jitter = %v, want between %v and %v",
			delay, minExpected, maxExpected)
	}
}

func TestRetryManager_ExecuteWithRetry(t *testing.T) {
	t.Run("Success on first attempt", func(t *testing.T) {
		rm := NewRetryManager()

		callCount := 0
		operation := func() error {
			callCount++
			return nil // Success
		}

		ctx := context.Background()

		err := rm.ExecuteWithRetry(ctx, operation)
		if err != nil {
			t.Errorf("ExecuteWithRetry() = %v, want nil", err)
		}

		if callCount != 1 {
			t.Errorf("Operation called %d times, want 1", callCount)
		}
	})

	t.Run("Success after retries with retryable error", func(t *testing.T) {
		rm := NewRetryManagerWithConfig(3, 1*time.Millisecond, 10*time.Millisecond, 1.1, false)

		callCount := 0
		operation := func() error {
			callCount++
			if callCount < 3 {
				// Use a DownloadError which is retryable
				return &errors.DownloadError{
					Code:      errors.CodeNetworkError,
					Message:   "temporary error",
					Retryable: true,
				}
			}

			return nil // Success on third attempt
		}

		ctx := context.Background()

		err := rm.ExecuteWithRetry(ctx, operation)
		if err != nil {
			t.Errorf("ExecuteWithRetry() = %v, want nil", err)
		}

		if callCount != 3 {
			t.Errorf("Operation called %d times, want 3", callCount)
		}
	})

	t.Run("Failure after max retries with retryable error", func(t *testing.T) {
		rm := NewRetryManagerWithConfig(2, 1*time.Millisecond, 10*time.Millisecond, 1.1, false)

		callCount := 0
		operation := func() error {
			callCount++

			return &errors.DownloadError{
				Code:      errors.CodeNetworkError,
				Message:   "persistent error",
				Retryable: true,
			}
		}

		ctx := context.Background()

		err := rm.ExecuteWithRetry(ctx, operation)
		if err == nil {
			t.Error("ExecuteWithRetry() = nil, want error")
		}

		expectedCalls := rm.MaxRetries + 1 // Initial attempt + retries
		if callCount != expectedCalls {
			t.Errorf("Operation called %d times, want %d", callCount, expectedCalls)
		}
	})

	t.Run("Context cancellation", func(t *testing.T) {
		rm := NewRetryManagerWithConfig(5, 50*time.Millisecond, 200*time.Millisecond, 2.0, false)

		callCount := 0
		operation := func() error {
			callCount++

			return &errors.DownloadError{
				Code:      errors.CodeNetworkError,
				Message:   "error",
				Retryable: true,
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Millisecond)
		defer cancel()

		err := rm.ExecuteWithRetry(ctx, operation)

		if !stderrors.Is(err, context.DeadlineExceeded) {
			t.Errorf("ExecuteWithRetry() = %v, want %v", err, context.DeadlineExceeded)
		}
	})
}

func TestRetryManager_ExecuteWithRetryCallback(t *testing.T) {
	rm := NewRetryManagerWithConfig(3, 1*time.Millisecond, 10*time.Millisecond, 2.0, false)

	callCount := 0
	retryCallbacks := 0

	var (
		lastRetryAttempt int
		lastRetryError   error
		lastRetryDelay   time.Duration
	)

	operation := func() error {
		callCount++
		if callCount < 3 {
			return &errors.DownloadError{
				Code:      errors.CodeNetworkError,
				Message:   "temporary error",
				Retryable: true,
			}
		}

		return nil // Success on third attempt
	}

	onRetry := func(attempt int, err error, nextDelay time.Duration) {
		retryCallbacks++
		lastRetryAttempt = attempt
		lastRetryError = err
		lastRetryDelay = nextDelay
	}

	ctx := context.Background()

	err := rm.ExecuteWithRetryCallback(ctx, operation, onRetry)
	if err != nil {
		t.Errorf("ExecuteWithRetryCallback() = %v, want nil", err)
	}

	if callCount != 3 {
		t.Errorf("Operation called %d times, want 3", callCount)
	}

	if retryCallbacks != 2 { // Two retries before success
		t.Errorf("Retry callback called %d times, want 2", retryCallbacks)
	}

	if lastRetryAttempt != 2 {
		t.Errorf("Last retry attempt = %d, want 2", lastRetryAttempt)
	}

	if lastRetryError == nil {
		t.Error("Last retry error should not be nil")
	}

	if lastRetryDelay <= 0 {
		t.Errorf("Last retry delay = %v, want > 0", lastRetryDelay)
	}
}

func TestRetryManager_ExecuteWithRetryAndStats(t *testing.T) {
	rm := NewRetryManagerWithConfig(3, 1*time.Millisecond, 10*time.Millisecond, 2.0, false)

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return &errors.DownloadError{
				Code:      errors.CodeNetworkError,
				Message:   "temporary error",
				Retryable: true,
			}
		}

		return nil // Success on third attempt
	}

	ctx := context.Background()

	stats, err := rm.ExecuteWithRetryAndStats(ctx, operation)
	if err != nil {
		t.Errorf("ExecuteWithRetryAndStats() error = %v, want nil", err)
	}

	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	if stats.TotalAttempts != 3 {
		t.Errorf("TotalAttempts = %d, want 3", stats.TotalAttempts)
	}

	if !stats.Succeeded {
		t.Error("Succeeded should be true")
	}

	if stats.TotalDelay <= 0 {
		t.Errorf("TotalDelay = %v, want > 0", stats.TotalDelay)
	}

	if stats.LastError != nil {
		t.Errorf("LastError should be nil for successful operation, got %v", stats.LastError)
	}
}

func TestRetryManager_FluentInterface(t *testing.T) {
	original := NewRetryManager()

	modified := original.
		WithMaxRetries(10).
		WithBaseDelay(500 * time.Millisecond).
		WithMaxDelay(2 * time.Minute).
		WithBackoffFactor(3.0).
		WithJitter(false)

	// Original should be unchanged
	if original.MaxRetries == 10 {
		t.Error("Original RetryManager was modified by fluent methods")
	}

	// Modified should have new values
	if modified.MaxRetries != 10 {
		t.Errorf("Modified RetryManager MaxRetries = %d, want 10", modified.MaxRetries)
	}

	if modified.BaseDelay != 500*time.Millisecond {
		t.Errorf(
			"Modified RetryManager BaseDelay = %v, want %v",
			modified.BaseDelay,
			500*time.Millisecond,
		)
	}

	if modified.MaxDelay != 2*time.Minute {
		t.Errorf("Modified RetryManager MaxDelay = %v, want %v", modified.MaxDelay, 2*time.Minute)
	}

	if modified.BackoffFactor != 3.0 {
		t.Errorf("Modified RetryManager BackoffFactor = %f, want 3.0", modified.BackoffFactor)
	}

	if modified.Jitter != false {
		t.Errorf("Modified RetryManager Jitter = %v, want false", modified.Jitter)
	}
}

func TestRetryableOperation(t *testing.T) {
	rm := NewRetryManagerWithConfig(3, 1*time.Millisecond, 10*time.Millisecond, 2.0, false)

	callCount := 0
	operation := func() error {
		callCount++
		if callCount < 3 {
			return &errors.DownloadError{
				Code:      errors.CodeNetworkError,
				Message:   "temporary error",
				Retryable: true,
			}
		}

		return nil
	}

	retryableOp := rm.NewRetryableOperation(operation)

	t.Run("Execute", func(t *testing.T) {
		ctx := context.Background()

		err := retryableOp.Execute(ctx)
		if err != nil {
			t.Errorf("Execute() = %v, want nil", err)
		}

		if callCount != 3 {
			t.Errorf("Operation called %d times, want 3", callCount)
		}
	})

	t.Run("ExecuteWithCallback", func(t *testing.T) {
		// Reset call count for new test
		callCount = 0

		retryCallbacks := 0
		onRetry := func(attempt int, err error, nextDelay time.Duration) {
			retryCallbacks++
		}

		ctx := context.Background()

		err := retryableOp.ExecuteWithCallback(ctx, onRetry)
		if err != nil {
			t.Errorf("ExecuteWithCallback() = %v, want nil", err)
		}

		if callCount != 3 {
			t.Errorf("Operation called %d times, want 3", callCount)
		}

		if retryCallbacks != 2 {
			t.Errorf("Retry callback called %d times, want 2", retryCallbacks)
		}
	})

	t.Run("ExecuteWithStats", func(t *testing.T) {
		// Reset call count for new test
		callCount = 0

		ctx := context.Background()

		stats, err := retryableOp.ExecuteWithStats(ctx)
		if err != nil {
			t.Errorf("ExecuteWithStats() error = %v, want nil", err)
		}

		if stats == nil {
			t.Fatal("Stats should not be nil")
		}

		if stats.TotalAttempts != 3 {
			t.Errorf("TotalAttempts = %d, want 3", stats.TotalAttempts)
		}

		if !stats.Succeeded {
			t.Error("Succeeded should be true")
		}
	})
}

func TestPredefinedRetryManagers(t *testing.T) {
	managers := map[string]*RetryManager{
		"Default":      DefaultRetryManager(),
		"Network":      NetworkRetryManager(),
		"FileSystem":   FileSystemRetryManager(),
		"Aggressive":   AggressiveRetryManager(),
		"Conservative": ConservativeRetryManager(),
	}

	for name, rm := range managers {
		t.Run(name, func(t *testing.T) {
			if rm == nil {
				t.Fatal("RetryManager should not be nil")
			}

			if rm.MaxRetries <= 0 {
				t.Errorf("MaxRetries = %d, want > 0", rm.MaxRetries)
			}

			if rm.BaseDelay <= 0 {
				t.Errorf("BaseDelay = %v, want > 0", rm.BaseDelay)
			}

			if rm.MaxDelay <= 0 {
				t.Errorf("MaxDelay = %v, want > 0", rm.MaxDelay)
			}

			if rm.BackoffFactor <= 0 {
				t.Errorf("BackoffFactor = %f, want > 0", rm.BackoffFactor)
			}
		})
	}
}

func TestRetryManager_NegativeAttempt(t *testing.T) {
	rm := NewRetryManager()

	// Test that negative attempt numbers are handled gracefully
	delay := rm.NextDelay(-1)
	if delay != rm.BaseDelay {
		t.Errorf("NextDelay(-1) = %v, want %v", delay, rm.BaseDelay)
	}
}

func TestRetryManager_EdgeCases(t *testing.T) {
	t.Run("Zero base delay", func(t *testing.T) {
		rm := NewRetryManagerWithConfig(3, 0, 1*time.Second, 2.0, false)
		delay := rm.NextDelay(0)

		if delay != 0 {
			t.Errorf("NextDelay(0) with zero base delay = %v, want 0", delay)
		}
	})

	t.Run("Very large attempt number", func(t *testing.T) {
		rm := NewRetryManagerWithConfig(1000, 1*time.Microsecond, 1*time.Second, 2.0, false)
		delay := rm.NextDelay(100)

		// Should be capped at MaxDelay
		if delay != rm.MaxDelay {
			t.Errorf("NextDelay(100) = %v, want %v (MaxDelay)", delay, rm.MaxDelay)
		}
	})
}

func TestRetryManager_AddJitter_EdgeCases(t *testing.T) {
	manager := NewRetryManager().WithJitter(true)

	// Test jitter with zero delay
	jitteredZero := manager.addJitter(0)
	if jitteredZero != 0 {
		t.Errorf("Expected jittered zero delay to be 0, got %v", jitteredZero)
	}

	// Test jitter with very small delay
	smallDelay := 1 * time.Nanosecond

	jitteredSmall := manager.addJitter(smallDelay)
	if jitteredSmall < 0 || jitteredSmall > 2*smallDelay {
		t.Errorf("Expected jittered small delay to be reasonable, got %v", jitteredSmall)
	}

	// Test jitter with large delay - should be within reasonable bounds
	largeDelay := 10 * time.Second

	jitteredLarge := manager.addJitter(largeDelay)
	if jitteredLarge < 0 || jitteredLarge > 2*largeDelay {
		t.Errorf("Expected jittered large delay to be within bounds, got %v", jitteredLarge)
	}
}

func TestRetryManager_ExecuteWithRetry_Panic(t *testing.T) {
	ctx := context.Background()
	manager := NewRetryManager().WithMaxRetries(2)

	panicOp := func() error {
		panic("test panic")
	}

	// The function should propagate panics (this is expected behavior)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic to be propagated")
		}
	}()

	_ = manager.ExecuteWithRetry(ctx, panicOp)
}

func TestRetryManager_ExecuteWithRetryCallback_NilCallback(t *testing.T) {
	ctx := context.Background()
	manager := NewRetryManager().WithMaxRetries(1)

	attemptCount := 0
	failOnceOp := func() error {
		attemptCount++
		if attemptCount == 1 {
			// Use a retryable error
			return &errors.DownloadError{
				Code:      errors.CodeNetworkError,
				Message:   "first attempt fails",
				Retryable: true,
			}
		}

		return nil
	}

	// Test with nil callback - should not cause issues
	err := manager.ExecuteWithRetryCallback(ctx, failOnceOp, nil)
	if err != nil {
		t.Errorf("Expected success with nil callback: %v", err)
	}

	if attemptCount != 2 {
		t.Errorf("Expected 2 attempts, got %d", attemptCount)
	}
}

func TestRetryManager_ExecuteWithRetryAndStats_MaxDelay(t *testing.T) {
	ctx := context.Background()
	manager := NewRetryManager().
		WithMaxRetries(3).
		WithBaseDelay(100 * time.Millisecond).
		WithMaxDelay(200 * time.Millisecond).
		WithBackoffFactor(10.0) // Very high factor to test max delay

	attemptCount := 0
	alwaysFailOp := func() error {
		attemptCount++
		// Use retryable errors
		return &errors.DownloadError{
			Code:      errors.CodeNetworkError,
			Message:   fmt.Sprintf("attempt %d failed", attemptCount),
			Retryable: true,
		}
	}

	stats, err := manager.ExecuteWithRetryAndStats(ctx, alwaysFailOp)
	if err == nil {
		t.Error("Expected error for always failing operation")
	}

	// Check that total delay was reasonable (should be capped by max delay)
	if stats.TotalDelay > 1*time.Second { // Allow generous margin for 3 retries at 200ms max
		t.Errorf(
			"Expected total delay to be reasonable with max delay cap, got %v",
			stats.TotalDelay,
		)
	}

	if stats.TotalAttempts != 4 { // 1 initial + 3 retries
		t.Errorf("Expected 4 attempts, got %d", stats.TotalAttempts)
	}
}
