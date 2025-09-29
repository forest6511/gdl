package hooks

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

func TestNewHookExecutor(t *testing.T) {
	executor := NewHookExecutor()

	if executor == nil {
		t.Fatal("NewHookExecutor should return non-nil executor")
	}

	if !executor.IsEnabled() {
		t.Error("New executor should be enabled by default")
	}

	if executor.handlers == nil {
		t.Error("Handlers map should be initialized")
	}
}

func TestDefaultHookExecutor_Execute(t *testing.T) {
	t.Run("NilHookContext", func(t *testing.T) {
		executor := NewHookExecutor()
		err := executor.Execute(context.Background(), nil)
		if err == nil {
			t.Error("Execute with nil hook context should return error")
		}
	})

	t.Run("NoHandlers", func(t *testing.T) {
		executor := NewHookExecutor()
		hook := &HookContext{
			Type: HookPreDownload,
			Data: "test",
		}
		err := executor.Execute(context.Background(), hook)
		if err != nil {
			t.Errorf("Execute with no handlers should not error: %v", err)
		}
	})

	t.Run("SingleHandler", func(t *testing.T) {
		executor := NewHookExecutor()
		executed := false

		handler := func(ctx context.Context, hc *HookContext) error {
			executed = true
			return nil
		}

		_ = executor.Register(HookPreDownload, handler)

		hook := &HookContext{
			Type: HookPreDownload,
			Data: "test",
		}

		err := executor.Execute(context.Background(), hook)
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}

		if !executed {
			t.Error("Handler should have been executed")
		}
	})

	t.Run("MultipleHandlers", func(t *testing.T) {
		executor := NewHookExecutor()
		var executionOrder []int

		for i := 1; i <= 3; i++ {
			idx := i
			handler := func(ctx context.Context, hc *HookContext) error {
				executionOrder = append(executionOrder, idx)
				return nil
			}
			_ = executor.Register(HookPostDownload, handler)
		}

		hook := &HookContext{
			Type: HookPostDownload,
			Data: "test",
		}

		err := executor.Execute(context.Background(), hook)
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}

		if len(executionOrder) != 3 {
			t.Errorf("Expected 3 handlers executed, got %d", len(executionOrder))
		}

		for i, val := range executionOrder {
			if val != i+1 {
				t.Errorf("Handler executed out of order: position %d has value %d", i, val)
			}
		}
	})

	t.Run("HandlerError", func(t *testing.T) {
		executor := NewHookExecutor()
		testErr := errors.New("handler error")

		handler := func(ctx context.Context, hc *HookContext) error {
			return testErr
		}

		_ = executor.Register(HookOnError, handler)

		hook := &HookContext{
			Type: HookOnError,
			Data: "error",
		}

		err := executor.Execute(context.Background(), hook)
		if err == nil {
			t.Error("Execute should return error when handler fails")
		}

		if !errors.Is(err, testErr) {
			t.Errorf("Expected error to wrap %v, got %v", testErr, err)
		}
	})

	t.Run("ContextCancellation", func(t *testing.T) {
		executor := NewHookExecutor()
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		handler := func(ctx context.Context, hc *HookContext) error {
			t.Error("Handler should not be executed after context cancellation")
			return nil
		}

		_ = executor.Register(HookPreDownload, handler)

		hook := &HookContext{
			Type: HookPreDownload,
			Data: "test",
		}

		err := executor.Execute(ctx, hook)
		if err == nil {
			t.Error("Execute should return error when context is cancelled")
		}

		if !errors.Is(err, context.Canceled) {
			t.Errorf("Expected context.Canceled, got %v", err)
		}
	})

	t.Run("DisabledExecutor", func(t *testing.T) {
		executor := NewHookExecutor()
		executed := false

		handler := func(ctx context.Context, hc *HookContext) error {
			executed = true
			return nil
		}

		_ = executor.Register(HookPreDownload, handler)
		executor.Disable()

		hook := &HookContext{
			Type: HookPreDownload,
			Data: "test",
		}

		err := executor.Execute(context.Background(), hook)
		if err != nil {
			t.Errorf("Execute should not error when disabled: %v", err)
		}

		if executed {
			t.Error("Handler should not be executed when executor is disabled")
		}
	})

	t.Run("NilHandler", func(t *testing.T) {
		executor := NewHookExecutor()

		// Manually add nil handler to test robustness
		executor.mu.Lock()
		executor.handlers[HookPreDownload] = []HookHandler{nil}
		executor.mu.Unlock()

		hook := &HookContext{
			Type: HookPreDownload,
			Data: "test",
		}

		err := executor.Execute(context.Background(), hook)
		if err != nil {
			t.Errorf("Execute should handle nil handlers gracefully: %v", err)
		}
	})
}

func TestDefaultHookExecutor_Register(t *testing.T) {
	t.Run("RegisterNilHandler", func(t *testing.T) {
		executor := NewHookExecutor()
		err := executor.Register(HookPreDownload, nil)
		if err == nil {
			t.Error("Register should return error for nil handler")
		}
	})

	t.Run("RegisterMultiple", func(t *testing.T) {
		executor := NewHookExecutor()

		handler1 := func(ctx context.Context, hc *HookContext) error { return nil }
		handler2 := func(ctx context.Context, hc *HookContext) error { return nil }

		err := executor.Register(HookPreDownload, handler1)
		if err != nil {
			t.Errorf("First registration failed: %v", err)
		}

		err = executor.Register(HookPreDownload, handler2)
		if err != nil {
			t.Errorf("Second registration failed: %v", err)
		}

		count := executor.GetHandlerCount(HookPreDownload)
		if count != 2 {
			t.Errorf("Expected 2 handlers, got %d", count)
		}
	})

	t.Run("RegisterDifferentTypes", func(t *testing.T) {
		executor := NewHookExecutor()

		handler := func(ctx context.Context, hc *HookContext) error { return nil }

		_ = executor.Register(HookPreDownload, handler)
		_ = executor.Register(HookPostDownload, handler)
		_ = executor.Register(HookOnError, handler)

		if !executor.HasHandlers(HookPreDownload) {
			t.Error("Should have PreDownload handlers")
		}
		if !executor.HasHandlers(HookPostDownload) {
			t.Error("Should have PostDownload handlers")
		}
		if !executor.HasHandlers(HookOnError) {
			t.Error("Should have OnError handlers")
		}
	})
}

func TestDefaultHookExecutor_Unregister(t *testing.T) {
	executor := NewHookExecutor()

	handler := func(ctx context.Context, hc *HookContext) error { return nil }

	// Register handlers
	_ = executor.Register(HookPreDownload, handler)
	_ = executor.Register(HookPreDownload, handler)

	if executor.GetHandlerCount(HookPreDownload) != 2 {
		t.Error("Should have 2 handlers before unregister")
	}

	// Unregister
	executor.Unregister(HookPreDownload)

	if executor.HasHandlers(HookPreDownload) {
		t.Error("Should have no handlers after unregister")
	}

	// Unregister non-existent should not panic
	executor.Unregister(HookPostDownload)
}

func TestDefaultHookExecutor_Clear(t *testing.T) {
	executor := NewHookExecutor()

	handler := func(ctx context.Context, hc *HookContext) error { return nil }

	// Register multiple handlers for different types
	_ = executor.Register(HookPreDownload, handler)
	_ = executor.Register(HookPostDownload, handler)
	_ = executor.Register(HookOnError, handler)

	// Clear all
	executor.Clear()

	if executor.HasHandlers(HookPreDownload) {
		t.Error("Should have no PreDownload handlers after clear")
	}
	if executor.HasHandlers(HookPostDownload) {
		t.Error("Should have no PostDownload handlers after clear")
	}
	if executor.HasHandlers(HookOnError) {
		t.Error("Should have no OnError handlers after clear")
	}
}

func TestDefaultHookExecutor_EnableDisable(t *testing.T) {
	executor := NewHookExecutor()

	if !executor.IsEnabled() {
		t.Error("Should be enabled by default")
	}

	executor.Disable()
	if executor.IsEnabled() {
		t.Error("Should be disabled after Disable()")
	}

	executor.Enable()
	if !executor.IsEnabled() {
		t.Error("Should be enabled after Enable()")
	}
}

func TestDefaultHookExecutor_GetHandlerCount(t *testing.T) {
	executor := NewHookExecutor()

	if executor.GetHandlerCount(HookPreDownload) != 0 {
		t.Error("Should have 0 handlers initially")
	}

	handler := func(ctx context.Context, hc *HookContext) error { return nil }

	for i := 0; i < 5; i++ {
		_ = executor.Register(HookPreDownload, handler)
	}

	if executor.GetHandlerCount(HookPreDownload) != 5 {
		t.Error("Should have 5 handlers after registration")
	}
}

func TestDefaultHookExecutor_Concurrency(t *testing.T) {
	executor := NewHookExecutor()
	const goroutines = 100

	var wg sync.WaitGroup
	wg.Add(goroutines * 3) // 3 operations per goroutine

	// Concurrent registrations
	hookTypes := []HookType{HookPreDownload, HookPostDownload, HookOnError}
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			handler := func(ctx context.Context, hc *HookContext) error {
				return nil
			}
			_ = executor.Register(hookTypes[idx%3], handler)
		}(i)
	}

	// Concurrent executions
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			hook := &HookContext{
				Type: hookTypes[idx%3],
				Data: idx,
			}
			_ = executor.Execute(context.Background(), hook)
		}(i)
	}

	// Concurrent enable/disable
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			if idx%2 == 0 {
				executor.Enable()
			} else {
				executor.Disable()
			}
		}(i)
	}

	wg.Wait()

	// Verify no panic occurred and executor is still functional
	hook := &HookContext{
		Type: HookPreDownload,
		Data: "test",
	}
	err := executor.Execute(context.Background(), hook)
	if err != nil {
		t.Errorf("Executor should still be functional after concurrent operations: %v", err)
	}
}

func TestDefaultHookExecutor_ComplexScenarios(t *testing.T) {
	t.Run("ChainedHandlers", func(t *testing.T) {
		executor := NewHookExecutor()
		result := 0

		// Handler 1: multiply by 2
		handler1 := func(ctx context.Context, hc *HookContext) error {
			if val, ok := hc.Data.(int); ok {
				hc.Data = val * 2
			}
			return nil
		}

		// Handler 2: add 10
		handler2 := func(ctx context.Context, hc *HookContext) error {
			if val, ok := hc.Data.(int); ok {
				hc.Data = val + 10
			}
			return nil
		}

		// Handler 3: capture result
		handler3 := func(ctx context.Context, hc *HookContext) error {
			if val, ok := hc.Data.(int); ok {
				result = val
			}
			return nil
		}

		_ = executor.Register(HookPreDownload, handler1)
		_ = executor.Register(HookPreDownload, handler2)
		_ = executor.Register(HookPreDownload, handler3)

		hook := &HookContext{
			Type: HookPreDownload,
			Data: 5,
		}

		err := executor.Execute(context.Background(), hook)
		if err != nil {
			t.Errorf("Execute failed: %v", err)
		}

		// 5 * 2 + 10 = 20
		if result != 20 {
			t.Errorf("Expected result 20, got %d", result)
		}
	})

	t.Run("ContextTimeout", func(t *testing.T) {
		executor := NewHookExecutor()

		// Handler that takes time
		handler := func(ctx context.Context, hc *HookContext) error {
			select {
			case <-time.After(100 * time.Millisecond):
				return nil
			case <-ctx.Done():
				return ctx.Err()
			}
		}

		_ = executor.Register(HookPreDownload, handler)

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		hook := &HookContext{
			Type: HookPreDownload,
			Data: "test",
		}

		err := executor.Execute(ctx, hook)
		if err == nil {
			t.Error("Execute should timeout")
		}

		if !errors.Is(err, context.DeadlineExceeded) {
			t.Errorf("Expected DeadlineExceeded, got %v", err)
		}
	})

	t.Run("HandlerPanic", func(t *testing.T) {
		executor := NewHookExecutor()

		// Handler that panics
		handler := func(ctx context.Context, hc *HookContext) error {
			panic("test panic")
		}

		_ = executor.Register(HookOnError, handler)

		hook := &HookContext{
			Type: HookOnError,
			Data: "test",
		}

		// Should recover from panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Should have panicked")
			}
		}()

		_ = executor.Execute(context.Background(), hook)
	})
}

func TestHookContext_WithCancel(t *testing.T) {
	executor := NewHookExecutor()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	hook := &HookContext{
		Type:   HookOnError,
		Data:   "error",
		Cancel: cancel,
	}

	handler := func(ctx context.Context, hc *HookContext) error {
		if hc.Cancel != nil {
			hc.Cancel()
		}
		return nil
	}

	_ = executor.Register(HookOnError, handler)
	_ = executor.Execute(ctx, hook)

	select {
	case <-ctx.Done():
		// Expected - context should be cancelled
	default:
		t.Error("Context should be cancelled")
	}
}

func TestHookContext_Metadata(t *testing.T) {
	executor := NewHookExecutor()

	expectedMetadata := map[string]interface{}{
		"key1": "value1",
		"key2": 42,
		"key3": true,
	}

	var capturedMetadata map[string]interface{}

	handler := func(ctx context.Context, hc *HookContext) error {
		capturedMetadata = hc.Metadata
		return nil
	}

	_ = executor.Register(HookOnProgress, handler)

	hook := &HookContext{
		Type:     HookOnProgress,
		Data:     "progress",
		Metadata: expectedMetadata,
	}

	_ = executor.Execute(context.Background(), hook)

	if len(capturedMetadata) != len(expectedMetadata) {
		t.Errorf("Metadata mismatch: expected %d items, got %d", len(expectedMetadata), len(capturedMetadata))
	}

	for k, v := range expectedMetadata {
		if capturedMetadata[k] != v {
			t.Errorf("Metadata[%s] = %v, want %v", k, capturedMetadata[k], v)
		}
	}
}

func BenchmarkDefaultHookExecutor_Execute(b *testing.B) {
	executor := NewHookExecutor()

	handler := func(ctx context.Context, hc *HookContext) error {
		return nil
	}

	_ = executor.Register(HookPreDownload, handler)

	hook := &HookContext{
		Type: HookPreDownload,
		Data: "test",
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = executor.Execute(ctx, hook)
	}
}

func BenchmarkDefaultHookExecutor_ConcurrentExecute(b *testing.B) {
	executor := NewHookExecutor()

	handler := func(ctx context.Context, hc *HookContext) error {
		return nil
	}

	for i := 0; i < 10; i++ {
		_ = executor.Register(HookPreDownload, handler)
	}

	hook := &HookContext{
		Type: HookPreDownload,
		Data: "test",
	}

	ctx := context.Background()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_ = executor.Execute(ctx, hook)
		}
	})
}
