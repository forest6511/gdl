package hooks

import (
	"context"
	"fmt"
	"sync"
)

// DefaultHookExecutor is the default implementation of HookExecutor
type DefaultHookExecutor struct {
	mu       sync.RWMutex
	handlers map[HookType][]HookHandler
	enabled  bool
}

// NewHookExecutor creates a new DefaultHookExecutor
func NewHookExecutor() *DefaultHookExecutor {
	return &DefaultHookExecutor{
		handlers: make(map[HookType][]HookHandler),
		enabled:  true,
	}
}

// Execute runs all registered handlers for the given hook type
func (e *DefaultHookExecutor) Execute(ctx context.Context, hook *HookContext) error {
	if hook == nil {
		return fmt.Errorf("hook context cannot be nil")
	}

	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.enabled {
		return nil
	}

	handlers, exists := e.handlers[hook.Type]
	if !exists || len(handlers) == 0 {
		return nil
	}

	for i, handler := range handlers {
		if handler == nil {
			continue
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := handler(ctx, hook); err != nil {
				return fmt.Errorf("hook handler %d failed: %w", i, err)
			}
		}
	}

	return nil
}

// Register adds a new handler for the specified hook type
func (e *DefaultHookExecutor) Register(hookType HookType, handler HookHandler) error {
	if handler == nil {
		return fmt.Errorf("handler cannot be nil")
	}

	e.mu.Lock()
	defer e.mu.Unlock()

	e.handlers[hookType] = append(e.handlers[hookType], handler)
	return nil
}

// Unregister removes all handlers for the specified hook type
func (e *DefaultHookExecutor) Unregister(hookType HookType) {
	e.mu.Lock()
	defer e.mu.Unlock()

	delete(e.handlers, hookType)
}

// Clear removes all registered handlers
func (e *DefaultHookExecutor) Clear() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.handlers = make(map[HookType][]HookHandler)
}

// Enable enables hook execution
func (e *DefaultHookExecutor) Enable() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.enabled = true
}

// Disable disables hook execution
func (e *DefaultHookExecutor) Disable() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.enabled = false
}

// IsEnabled returns whether hook execution is enabled
func (e *DefaultHookExecutor) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.enabled
}

// GetHandlerCount returns the number of handlers registered for a hook type
func (e *DefaultHookExecutor) GetHandlerCount(hookType HookType) int {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return len(e.handlers[hookType])
}

// HasHandlers returns whether any handlers are registered for a hook type
func (e *DefaultHookExecutor) HasHandlers(hookType HookType) bool {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return len(e.handlers[hookType]) > 0
}
