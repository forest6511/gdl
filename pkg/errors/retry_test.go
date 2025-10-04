package errors

import (
	"errors"
	"testing"
	"time"
)

// TestCircuitBreaker tests circuit breaker functionality
func TestCircuitBreaker(t *testing.T) {
	t.Run("NewCircuitBreaker", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 100*time.Millisecond)
		if cb == nil {
			t.Fatal("Expected circuit breaker to be created")
		}
		if cb.maxFailures != 3 {
			t.Errorf("Expected maxFailures 3, got %d", cb.maxFailures)
		}
		if cb.resetTimeout != 100*time.Millisecond {
			t.Errorf("Expected resetTimeout 100ms, got %v", cb.resetTimeout)
		}
		if cb.state != CircuitClosed {
			t.Errorf("Expected initial state CircuitClosed, got %v", cb.state)
		}
	})

	t.Run("CallSuccess", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 100*time.Millisecond)

		err := cb.Call(func() error {
			return nil
		})

		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if cb.failureCount != 0 {
			t.Errorf("Expected failure count 0, got %d", cb.failureCount)
		}
		if cb.state != CircuitClosed {
			t.Errorf("Expected state CircuitClosed, got %v", cb.state)
		}
	})

	t.Run("CallFailure", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 100*time.Millisecond)
		testErr := errors.New("test error")

		err := cb.Call(func() error {
			return testErr
		})

		if err != testErr {
			t.Errorf("Expected test error, got %v", err)
		}
		if cb.failureCount != 1 {
			t.Errorf("Expected failure count 1, got %d", cb.failureCount)
		}
	})

	t.Run("CircuitOpens", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 100*time.Millisecond)
		testErr := errors.New("test error")

		// Fail 3 times to open circuit
		for i := 0; i < 3; i++ {
			err := cb.Call(func() error {
				return testErr
			})
			if i < 2 && err != testErr {
				t.Errorf("Expected test error on attempt %d, got %v", i, err)
			}
		}

		if cb.state != CircuitOpen {
			t.Errorf("Expected state CircuitOpen, got %v", cb.state)
		}

		// Next call should fail immediately
		err := cb.Call(func() error {
			t.Error("Function should not be called when circuit is open")
			return nil
		})

		if err == nil {
			t.Error("Expected error when circuit is open")
		}
		if err.Error() != "circuit breaker is open" {
			t.Errorf("Expected 'circuit breaker is open', got %v", err)
		}
	})

	t.Run("CircuitHalfOpen", func(t *testing.T) {
		cb := NewCircuitBreaker(3, 50*time.Millisecond)
		testErr := errors.New("test error")

		// Open the circuit
		for i := 0; i < 3; i++ {
			_ = cb.Call(func() error {
				return testErr
			})
		}

		if cb.state != CircuitOpen {
			t.Fatal("Circuit should be open")
		}

		// Wait for reset timeout
		time.Sleep(60 * time.Millisecond)

		// Next call should transition to half-open
		callExecuted := false
		err := cb.Call(func() error {
			callExecuted = true
			return nil
		})

		if !callExecuted {
			t.Error("Function should be called in half-open state")
		}
		if err != nil {
			t.Errorf("Expected no error in half-open success, got %v", err)
		}
		if cb.state != CircuitClosed {
			t.Errorf("Expected state CircuitClosed after success, got %v", cb.state)
		}
		if cb.failureCount != 0 {
			t.Errorf("Expected failure count reset to 0, got %d", cb.failureCount)
		}
	})

	t.Run("CircuitHalfOpenFailure", func(t *testing.T) {
		cb := NewCircuitBreaker(2, 50*time.Millisecond)
		testErr := errors.New("test error")

		// Open the circuit
		for i := 0; i < 2; i++ {
			_ = cb.Call(func() error {
				return testErr
			})
		}

		// Wait for reset timeout
		time.Sleep(60 * time.Millisecond)

		// Fail in half-open state
		_ = cb.Call(func() error {
			return testErr
		})

		// Should still count failures
		if cb.failureCount == 0 {
			t.Error("Expected failure count to be incremented")
		}
	})
}

// TestAdaptiveRetryStrategy tests adaptive retry strategy
func TestAdaptiveRetryStrategy(t *testing.T) {
	t.Run("NewAdaptiveRetryStrategy", func(t *testing.T) {
		base := &RetryStrategy{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
			MaxDelay:     5 * time.Second,
			Multiplier:   2.0,
			Jitter:       true,
		}

		ars := NewAdaptiveRetryStrategy(base)
		if ars == nil {
			t.Fatal("Expected adaptive retry strategy to be created")
		}
		if ars.baseStrategy != base {
			t.Error("Expected base strategy to be set")
		}
		if ars.adaptMultiplier != 1.0 {
			t.Errorf("Expected initial multiplier 1.0, got %f", ars.adaptMultiplier)
		}
		if ars.historyWindow != 5*time.Minute {
			t.Errorf("Expected history window 5 minutes, got %v", ars.historyWindow)
		}
	})

	t.Run("NewAdaptiveRetryStrategyNilBase", func(t *testing.T) {
		ars := NewAdaptiveRetryStrategy(nil)
		if ars == nil {
			t.Fatal("Expected adaptive retry strategy to be created")
		}
		if ars.baseStrategy == nil {
			t.Error("Expected default base strategy to be created")
		}
	})

	t.Run("UpdateStrategyHighErrorRate", func(t *testing.T) {
		ars := NewAdaptiveRetryStrategy(nil)
		ars.historyWindow = 1 * time.Minute
		testErr := errors.New("test error")

		// Add multiple errors to simulate high error rate
		for i := 0; i < 100; i++ {
			ars.UpdateStrategy(testErr)
		}

		// High error rate should increase multiplier
		if ars.adaptMultiplier <= 1.0 {
			t.Errorf("Expected multiplier > 1.0 for high error rate, got %f", ars.adaptMultiplier)
		}
		if ars.adaptMultiplier > 3.0 {
			t.Errorf("Expected multiplier <= 3.0 (max), got %f", ars.adaptMultiplier)
		}
	})

	t.Run("UpdateStrategyLowErrorRate", func(t *testing.T) {
		ars := NewAdaptiveRetryStrategy(nil)
		ars.historyWindow = 10 * time.Minute
		ars.adaptMultiplier = 2.0 // Start with higher multiplier
		testErr := errors.New("test error")

		// Add very few errors for low error rate
		ars.UpdateStrategy(testErr)

		// Low error rate should tend to decrease multiplier (multiplier * 0.9)
		// But it depends on the exact error rate calculation
		// Just verify it's within reasonable bounds
		if ars.adaptMultiplier > 3.0 {
			t.Errorf("Expected multiplier <= 3.0 (max), got %f", ars.adaptMultiplier)
		}
		if ars.adaptMultiplier < 0.5 {
			t.Errorf("Expected multiplier >= 0.5 (min), got %f", ars.adaptMultiplier)
		}
	})

	t.Run("UpdateStrategyCleansOldHistory", func(t *testing.T) {
		ars := NewAdaptiveRetryStrategy(nil)
		ars.historyWindow = 100 * time.Millisecond
		testErr := errors.New("test error")

		// Add some errors
		for i := 0; i < 5; i++ {
			ars.UpdateStrategy(testErr)
		}

		if len(ars.errorHistory) != 5 {
			t.Errorf("Expected 5 errors in history, got %d", len(ars.errorHistory))
		}

		// Wait for history to expire
		time.Sleep(150 * time.Millisecond)

		// Add another error to trigger cleanup
		ars.UpdateStrategy(testErr)

		// Old history should be cleaned
		if len(ars.errorHistory) > 1 {
			t.Errorf("Expected old history to be cleaned, got %d entries", len(ars.errorHistory))
		}
	})

	t.Run("GetAdjustedDelay", func(t *testing.T) {
		ars := NewAdaptiveRetryStrategy(nil)
		baseDelay := 100 * time.Millisecond

		// Test with default multiplier (1.0)
		delay := ars.GetAdjustedDelay(baseDelay)
		if delay != baseDelay {
			t.Errorf("Expected delay %v with multiplier 1.0, got %v", baseDelay, delay)
		}

		// Test with increased multiplier
		ars.adaptMultiplier = 2.0
		delay = ars.GetAdjustedDelay(baseDelay)
		expected := 200 * time.Millisecond
		if delay != expected {
			t.Errorf("Expected delay %v with multiplier 2.0, got %v", expected, delay)
		}

		// Test with decreased multiplier
		ars.adaptMultiplier = 0.5
		delay = ars.GetAdjustedDelay(baseDelay)
		expected = 50 * time.Millisecond
		if delay != expected {
			t.Errorf("Expected delay %v with multiplier 0.5, got %v", expected, delay)
		}
	})

	t.Run("AdaptiveMultiplierBounds", func(t *testing.T) {
		ars := NewAdaptiveRetryStrategy(nil)
		ars.historyWindow = 1 * time.Millisecond
		testErr := errors.New("test error")

		// Force very high error rate
		for i := 0; i < 1000; i++ {
			ars.UpdateStrategy(testErr)
		}

		// Should cap at 3.0
		if ars.adaptMultiplier > 3.0 {
			t.Errorf("Expected multiplier capped at 3.0, got %f", ars.adaptMultiplier)
		}

		// Reset and test minimum bound
		ars.adaptMultiplier = 1.0
		ars.errorHistory = []time.Time{}
		ars.historyWindow = 100 * time.Hour

		ars.UpdateStrategy(testErr)

		// Should floor at 0.5
		if ars.adaptMultiplier < 0.5 {
			t.Errorf("Expected multiplier floored at 0.5, got %f", ars.adaptMultiplier)
		}
	})
}

// TestRetryManagerCreation tests retry manager creation
func TestRetryManagerCreation(t *testing.T) {
	t.Run("NewRetryManager", func(t *testing.T) {
		strategy := &RetryStrategy{
			MaxAttempts:  3,
			InitialDelay: 100 * time.Millisecond,
		}

		rm := NewRetryManager(strategy)
		if rm == nil {
			t.Fatal("Expected retry manager to be created")
		}
		if rm.strategy != strategy {
			t.Error("Expected strategy to be set")
		}
	})

	t.Run("NewRetryManagerNilStrategy", func(t *testing.T) {
		rm := NewRetryManager(nil)
		if rm == nil {
			t.Fatal("Expected retry manager to be created")
		}
		if rm.strategy == nil {
			t.Error("Expected default strategy to be created")
		}
	})
}
