package hooks

import (
	"context"
	"github.com/forest6511/godl/pkg/types"
	"testing"
)

func TestHookTypes(t *testing.T) {
	// Test that hook types are properly assigned from types package
	tests := []struct {
		name     string
		hookType HookType
		expected types.HookType
	}{
		{"PreDownload", HookPreDownload, types.PreDownloadHook},
		{"PostDownload", HookPostDownload, types.PostDownloadHook},
		{"OnError", HookOnError, types.ErrorHook},
		{"OnComplete", HookOnComplete, types.PostDownloadHook},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.hookType != tt.expected {
				t.Errorf("HookType %s = %v, want %v", tt.name, tt.hookType, tt.expected)
			}
		})
	}
}

func TestHookContext(t *testing.T) {
	ctx := &HookContext{
		Type: HookPreDownload,
		Data: map[string]interface{}{
			"url":  "https://example.com/file.zip",
			"size": 1024,
		},
		Metadata: map[string]interface{}{
			"retry_count": 0,
		},
	}

	if ctx.Type != HookPreDownload {
		t.Errorf("Type = %v, want %v", ctx.Type, HookPreDownload)
	}

	if url, ok := ctx.Data.(map[string]interface{})["url"]; !ok || url != "https://example.com/file.zip" {
		t.Errorf("Data[url] = %v, want %v", url, "https://example.com/file.zip")
	}

	if retryCount, ok := ctx.Metadata["retry_count"]; !ok || retryCount != 0 {
		t.Errorf("Metadata[retry_count] = %v, want %v", retryCount, 0)
	}
}

// MockHookExecutor implements HookExecutor for testing
type MockHookExecutor struct {
	handlers      map[HookType][]HookHandler
	executeCalled bool
	lastHook      *HookContext
}

func NewMockHookExecutor() *MockHookExecutor {
	return &MockHookExecutor{
		handlers: make(map[HookType][]HookHandler),
	}
}

func (m *MockHookExecutor) Execute(ctx context.Context, hook *HookContext) error {
	m.executeCalled = true
	m.lastHook = hook

	handlers, exists := m.handlers[hook.Type]
	if !exists {
		return nil
	}

	for _, handler := range handlers {
		if err := handler(ctx, hook); err != nil {
			return err
		}
	}

	return nil
}

func (m *MockHookExecutor) Register(hookType HookType, handler HookHandler) error {
	m.handlers[hookType] = append(m.handlers[hookType], handler)
	return nil
}

func TestMockHookExecutor(t *testing.T) {
	executor := NewMockHookExecutor()

	executed := false
	handler := func(ctx context.Context, hookCtx *HookContext) error {
		executed = true
		return nil
	}

	// Register handler
	err := executor.Register(HookPreDownload, handler)
	if err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	// Execute hook
	hookCtx := &HookContext{
		Type: HookPreDownload,
		Data: "test data",
	}

	ctx := context.Background()
	err = executor.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if !executed {
		t.Error("Handler should have been executed")
	}

	if !executor.executeCalled {
		t.Error("Execute should have been called")
	}

	if executor.lastHook != hookCtx {
		t.Error("Last hook context should match")
	}
}

func TestHookExecutorWithCancel(t *testing.T) {
	executor := NewMockHookExecutor()

	// Create a context with cancel
	ctx, cancel := context.WithCancel(context.Background())

	hookCtx := &HookContext{
		Type:   HookOnError,
		Data:   "error data",
		Cancel: cancel,
	}

	// Register a handler that cancels the context
	handler := func(ctx context.Context, hc *HookContext) error {
		if hc.Cancel != nil {
			hc.Cancel()
		}
		return nil
	}

	_ = executor.Register(HookOnError, handler)

	// Execute hook
	err := executor.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// Context should be cancelled
	select {
	case <-ctx.Done():
		// Expected
	default:
		t.Error("Context should be cancelled")
	}
}

func TestMultipleHandlers(t *testing.T) {
	executor := NewMockHookExecutor()

	order := []int{}

	handler1 := func(ctx context.Context, hc *HookContext) error {
		order = append(order, 1)
		return nil
	}

	handler2 := func(ctx context.Context, hc *HookContext) error {
		order = append(order, 2)
		return nil
	}

	handler3 := func(ctx context.Context, hc *HookContext) error {
		order = append(order, 3)
		return nil
	}

	// Register multiple handlers for same hook type
	_ = executor.Register(HookPostDownload, handler1)
	_ = executor.Register(HookPostDownload, handler2)
	_ = executor.Register(HookPostDownload, handler3)

	// Execute hook
	hookCtx := &HookContext{
		Type: HookPostDownload,
		Data: "test",
	}

	ctx := context.Background()
	err := executor.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	// All handlers should execute in order
	if len(order) != 3 {
		t.Errorf("Expected 3 handlers executed, got %d", len(order))
	}

	for i, val := range order {
		if val != i+1 {
			t.Errorf("Handler executed out of order: position %d has value %d", i, val)
		}
	}
}

func TestHookWithMetadata(t *testing.T) {
	executor := NewMockHookExecutor()

	var capturedMetadata map[string]interface{}

	handler := func(ctx context.Context, hc *HookContext) error {
		capturedMetadata = hc.Metadata
		return nil
	}

	_ = executor.Register(HookOnProgress, handler)

	// Create hook with metadata
	hookCtx := &HookContext{
		Type: HookOnProgress,
		Data: map[string]interface{}{
			"bytes_downloaded": 1024,
			"total_bytes":      10240,
		},
		Metadata: map[string]interface{}{
			"start_time": "2024-01-01T00:00:00Z",
			"chunk_id":   5,
		},
	}

	ctx := context.Background()
	err := executor.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}

	if capturedMetadata == nil {
		t.Fatal("Metadata should have been captured")
	}

	if capturedMetadata["chunk_id"] != 5 {
		t.Errorf("Metadata[chunk_id] = %v, want %v", capturedMetadata["chunk_id"], 5)
	}
}

func TestHookContext_EdgeCases(t *testing.T) {
	t.Run("NilData", func(t *testing.T) {
		ctx := &HookContext{
			Type: HookPreDownload,
			Data: nil,
		}

		if ctx.Type != HookPreDownload {
			t.Errorf("Type = %v, want %v", ctx.Type, HookPreDownload)
		}

		if ctx.Data != nil {
			t.Errorf("Expected nil data, got %v", ctx.Data)
		}
	})

	t.Run("EmptyMetadata", func(t *testing.T) {
		ctx := &HookContext{
			Type:     HookPostDownload,
			Data:     "test",
			Metadata: map[string]interface{}{},
		}

		if len(ctx.Metadata) != 0 {
			t.Errorf("Expected empty metadata, got %v", ctx.Metadata)
		}
	})

	t.Run("ComplexData", func(t *testing.T) {
		complexData := map[string]interface{}{
			"nested": map[string]interface{}{
				"level1": map[string]interface{}{
					"level2": []interface{}{1, 2, 3, "test"},
				},
			},
		}

		ctx := &HookContext{
			Type: HookOnProgress,
			Data: complexData,
		}

		if ctx.Data == nil {
			t.Error("Expected complex data to be preserved")
		}

		// Verify complex data structure is preserved
		data, ok := ctx.Data.(map[string]interface{})
		if !ok {
			t.Fatal("Expected data to be map[string]interface{}")
		}

		nested, ok := data["nested"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected nested data structure")
		}

		if nested["level1"] == nil {
			t.Error("Expected nested level1 to exist")
		}
	})
}

func TestEmptyHookExecution(t *testing.T) {
	executor := NewMockHookExecutor()

	// Execute hook with no registered handlers
	hookCtx := &HookContext{
		Type: HookPreChunk,
		Data: "test",
	}

	ctx := context.Background()
	err := executor.Execute(ctx, hookCtx)
	if err != nil {
		t.Fatalf("Execute() error = %v for empty hook", err)
	}

	if !executor.executeCalled {
		t.Error("Execute should have been called even with no handlers")
	}
}
