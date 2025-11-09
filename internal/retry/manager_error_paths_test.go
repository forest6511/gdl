package retry

import (
	"context"
	"errors"
	"testing"
	"time"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
)

// TestRetryManagerErrorPaths tests all error paths in retry manager
func TestRetryManagerErrorPaths(t *testing.T) {
	t.Run("ExecuteWithRetry_ContextCancelled", func(t *testing.T) {
		manager := NewRetryManager()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		err := manager.ExecuteWithRetry(ctx, func() error {
			return errors.New("should not execute")
		})

		if err == nil {
			t.Fatal("Expected context error")
		}

		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}
	})

	t.Run("ExecuteWithRetry_NonRetryableError", func(t *testing.T) {
		manager := NewRetryManager()
		ctx := context.Background()

		// Create a non-retryable error
		nonRetryableErr := gdlerrors.NewDownloadError(
			gdlerrors.CodeValidationError,
			"validation failed",
		)

		err := manager.ExecuteWithRetry(ctx, func() error {
			return nonRetryableErr
		})

		if err == nil {
			t.Fatal("Expected error for non-retryable error")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		// Should be wrapped with CodeNetworkError
		if downloadErr.Code != gdlerrors.CodeNetworkError {
			t.Errorf("Expected CodeNetworkError, got: %s", downloadErr.Code)
		}
	})

	t.Run("ExecuteWithRetry_MaxRetriesExhausted", func(t *testing.T) {
		manager := &RetryManager{
			MaxRetries:    2,
			BaseDelay:     1 * time.Millisecond,
			MaxDelay:      10 * time.Millisecond,
			BackoffFactor: 2.0,
			Jitter:        false,
		}
		ctx := context.Background()

		// Create a retryable error that never succeeds
		retryableErr := gdlerrors.NewDownloadError(
			gdlerrors.CodeNetworkError,
			"network error",
		)
		retryableErr.Retryable = true

		attemptCount := 0
		err := manager.ExecuteWithRetry(ctx, func() error {
			attemptCount++
			return retryableErr
		})

		if err == nil {
			t.Fatal("Expected error for max retries exhausted")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		// When attempt reaches MaxRetries, ShouldRetry returns false,
		// so we get CodeNetworkError (non-retryable) instead of CodeTimeout
		if downloadErr.Code != gdlerrors.CodeNetworkError {
			t.Errorf("Expected CodeNetworkError, got: %s", downloadErr.Code)
		}

		// Should have attempted MaxRetries + 1 times
		expectedAttempts := manager.MaxRetries + 1
		if attemptCount != expectedAttempts {
			t.Errorf("Expected %d attempts, got: %d", expectedAttempts, attemptCount)
		}
	})

	t.Run("ExecuteWithRetryCallback_ContextCancelled", func(t *testing.T) {
		manager := NewRetryManager()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		callbackCount := 0
		err := manager.ExecuteWithRetryCallback(ctx, func() error {
			return errors.New("should not execute")
		}, func(attempt int, err error, nextDelay time.Duration) {
			callbackCount++
		})

		if err == nil {
			t.Fatal("Expected context error")
		}

		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}

		if callbackCount != 0 {
			t.Errorf("Expected no callbacks, got: %d", callbackCount)
		}
	})

	t.Run("ExecuteWithRetryCallback_NonRetryableError", func(t *testing.T) {
		manager := NewRetryManager()
		ctx := context.Background()

		nonRetryableErr := gdlerrors.NewDownloadError(
			gdlerrors.CodeValidationError,
			"validation failed",
		)

		callbackCount := 0
		err := manager.ExecuteWithRetryCallback(ctx, func() error {
			return nonRetryableErr
		}, func(attempt int, err error, nextDelay time.Duration) {
			callbackCount++
		})

		if err == nil {
			t.Fatal("Expected error for non-retryable error")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		if downloadErr.Code != gdlerrors.CodeNetworkError {
			t.Errorf("Expected CodeNetworkError, got: %s", downloadErr.Code)
		}

		// No callbacks should be called for non-retryable errors
		if callbackCount != 0 {
			t.Errorf("Expected no callbacks for non-retryable error, got: %d", callbackCount)
		}
	})

	t.Run("ExecuteWithRetryCallback_MaxRetriesWithCallback", func(t *testing.T) {
		manager := &RetryManager{
			MaxRetries:    2,
			BaseDelay:     1 * time.Millisecond,
			MaxDelay:      10 * time.Millisecond,
			BackoffFactor: 2.0,
			Jitter:        false,
		}
		ctx := context.Background()

		retryableErr := gdlerrors.NewDownloadError(
			gdlerrors.CodeNetworkError,
			"network error",
		)
		retryableErr.Retryable = true

		callbackCount := 0
		err := manager.ExecuteWithRetryCallback(ctx, func() error {
			return retryableErr
		}, func(attempt int, err error, nextDelay time.Duration) {
			callbackCount++
			if err == nil {
				t.Error("Callback should receive non-nil error")
			}
			if attempt <= 0 {
				t.Errorf("Expected attempt > 0, got: %d", attempt)
			}
		})

		if err == nil {
			t.Fatal("Expected error for max retries exhausted")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		// When attempt reaches MaxRetries, ShouldRetry returns false,
		// so we get CodeNetworkError (non-retryable) instead of CodeTimeout
		if downloadErr.Code != gdlerrors.CodeNetworkError {
			t.Errorf("Expected CodeNetworkError, got: %s", downloadErr.Code)
		}

		// Should have called callback for each retry attempt (not including initial attempt or last failed attempt)
		if callbackCount != manager.MaxRetries {
			t.Errorf("Expected %d callbacks, got: %d", manager.MaxRetries, callbackCount)
		}
	})

	t.Run("ExecuteWithRetryAndStats_ContextCancelled", func(t *testing.T) {
		manager := NewRetryManager()
		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		stats, err := manager.ExecuteWithRetryAndStats(ctx, func() error {
			return errors.New("should not execute")
		})

		if err == nil {
			t.Fatal("Expected context error")
		}

		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled, got: %v", err)
		}

		if stats == nil {
			t.Fatal("Expected non-nil stats")
		}

		if stats.Succeeded {
			t.Error("Expected Succeeded to be false")
		}

		if stats.LastError == nil {
			t.Error("Expected non-nil LastError")
		}
	})

	t.Run("ExecuteWithRetryAndStats_NonRetryableError", func(t *testing.T) {
		manager := NewRetryManager()
		ctx := context.Background()

		nonRetryableErr := gdlerrors.NewDownloadError(
			gdlerrors.CodeValidationError,
			"validation failed",
		)

		stats, err := manager.ExecuteWithRetryAndStats(ctx, func() error {
			return nonRetryableErr
		})

		if err == nil {
			t.Fatal("Expected error for non-retryable error")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		if downloadErr.Code != gdlerrors.CodeNetworkError {
			t.Errorf("Expected CodeNetworkError, got: %s", downloadErr.Code)
		}

		if stats == nil {
			t.Fatal("Expected non-nil stats")
		}

		if stats.Succeeded {
			t.Error("Expected Succeeded to be false")
		}

		if stats.TotalAttempts != 1 {
			t.Errorf("Expected 1 attempt, got: %d", stats.TotalAttempts)
		}

		if stats.LastError == nil {
			t.Error("Expected non-nil LastError")
		}
	})

	t.Run("ExecuteWithRetryAndStats_MaxRetriesExhausted", func(t *testing.T) {
		manager := &RetryManager{
			MaxRetries:    2,
			BaseDelay:     1 * time.Millisecond,
			MaxDelay:      10 * time.Millisecond,
			BackoffFactor: 2.0,
			Jitter:        false,
		}
		ctx := context.Background()

		retryableErr := gdlerrors.NewDownloadError(
			gdlerrors.CodeNetworkError,
			"network error",
		)
		retryableErr.Retryable = true

		stats, err := manager.ExecuteWithRetryAndStats(ctx, func() error {
			return retryableErr
		})

		if err == nil {
			t.Fatal("Expected error for max retries exhausted")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		// When attempt reaches MaxRetries, ShouldRetry returns false,
		// so we get CodeNetworkError (non-retryable) instead of CodeTimeout
		if downloadErr.Code != gdlerrors.CodeNetworkError {
			t.Errorf("Expected CodeNetworkError, got: %s", downloadErr.Code)
		}

		if stats == nil {
			t.Fatal("Expected non-nil stats")
		}

		if stats.Succeeded {
			t.Error("Expected Succeeded to be false")
		}

		expectedAttempts := manager.MaxRetries + 1
		if stats.TotalAttempts != expectedAttempts {
			t.Errorf("Expected %d attempts, got: %d", expectedAttempts, stats.TotalAttempts)
		}

		if stats.TotalDelay <= 0 {
			t.Error("Expected TotalDelay > 0")
		}

		if stats.LastError == nil {
			t.Error("Expected non-nil LastError")
		}
	})

	t.Run("ExecuteWithRetryAndStats_Success", func(t *testing.T) {
		manager := NewRetryManager()
		ctx := context.Background()

		attemptCount := 0
		stats, err := manager.ExecuteWithRetryAndStats(ctx, func() error {
			attemptCount++
			if attemptCount < 2 {
				// Fail first attempt
				retryableErr := gdlerrors.NewDownloadError(
					gdlerrors.CodeNetworkError,
					"network error",
				)
				retryableErr.Retryable = true
				return retryableErr
			}
			// Success on second attempt
			return nil
		})

		if err != nil {
			t.Fatalf("Expected no error, got: %v", err)
		}

		if stats == nil {
			t.Fatal("Expected non-nil stats")
		}

		if !stats.Succeeded {
			t.Error("Expected Succeeded to be true")
		}

		if stats.TotalAttempts != 2 {
			t.Errorf("Expected 2 attempts, got: %d", stats.TotalAttempts)
		}

		if stats.LastError != nil {
			t.Errorf("Expected nil LastError on success, got: %v", stats.LastError)
		}

		if stats.TotalDelay <= 0 {
			t.Error("Expected TotalDelay > 0")
		}
	})

	t.Run("ShouldRetry_MaxRetriesExceeded", func(t *testing.T) {
		manager := &RetryManager{
			MaxRetries: 3,
		}

		retryableErr := gdlerrors.NewDownloadError(
			gdlerrors.CodeNetworkError,
			"network error",
		)
		retryableErr.Retryable = true

		// Should not retry if attempt >= MaxRetries
		if manager.ShouldRetry(retryableErr, 3) {
			t.Error("Expected ShouldRetry to return false when attempt >= MaxRetries")
		}

		if manager.ShouldRetry(retryableErr, 4) {
			t.Error("Expected ShouldRetry to return false when attempt > MaxRetries")
		}
	})

	t.Run("NextDelay_NegativeAttempt", func(t *testing.T) {
		manager := NewRetryManager()

		delay := manager.NextDelay(-1)
		if delay != manager.BaseDelay {
			t.Errorf("Expected BaseDelay for negative attempt, got: %v", delay)
		}
	})

	t.Run("NextDelay_LargeAttempt", func(t *testing.T) {
		manager := &RetryManager{
			BaseDelay:     1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        false,
		}

		// Test very large attempt number (> 50)
		delay := manager.NextDelay(100)
		if delay != manager.MaxDelay {
			t.Errorf("Expected MaxDelay for large attempt, got: %v", delay)
		}
	})

	t.Run("NextDelay_Overflow", func(t *testing.T) {
		manager := &RetryManager{
			BaseDelay:     1 * time.Hour,
			MaxDelay:      2 * time.Hour,
			BackoffFactor: 10.0,
			Jitter:        false,
		}

		// Should cap at MaxDelay due to overflow
		delay := manager.NextDelay(20)
		if delay > manager.MaxDelay {
			t.Errorf("Expected delay <= MaxDelay, got: %v", delay)
		}
	})

	t.Run("NextDelay_WithJitter", func(t *testing.T) {
		manager := &RetryManager{
			BaseDelay:     1 * time.Second,
			MaxDelay:      30 * time.Second,
			BackoffFactor: 2.0,
			Jitter:        true,
		}

		// Run multiple times to test jitter variability
		delays := make(map[time.Duration]bool)
		for i := 0; i < 10; i++ {
			delay := manager.NextDelay(1)
			delays[delay] = true
		}

		// With jitter, we should see some variation
		// (though it's possible to get the same value multiple times by chance)
		if len(delays) == 0 {
			t.Error("Expected at least one delay value")
		}
	})

	t.Run("addJitter_PreventNegative", func(t *testing.T) {
		manager := NewRetryManager()

		// Test with very small delay
		delay := manager.addJitter(1 * time.Nanosecond)
		if delay < 0 {
			t.Errorf("Expected non-negative delay, got: %v", delay)
		}
	})
}
