package events

import (
	"context"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// TestEventEmitter tests the event emitter functionality
func TestEventEmitter(t *testing.T) {
	t.Run("NewEventEmitter", func(t *testing.T) {
		emitter := NewEventEmitter()

		if emitter == nil {
			t.Fatal("Expected emitter to be created")
		}

		if emitter.listeners == nil {
			t.Error("Expected listeners map to be initialized")
		}

		if len(emitter.listeners) != 0 {
			t.Error("Expected empty listeners map initially")
		}

		if emitter.IsClosed() {
			t.Error("New emitter should not be closed")
		}
	})
}

// TestEventListenerOperations tests adding and removing listeners
func TestEventListenerOperations(t *testing.T) {
	emitter := NewEventEmitter()

	t.Run("OnListener", func(t *testing.T) {
		executed := false
		listener := func(event Event) {
			executed = true
		}

		emitter.On(EventDownloadStarted, listener)

		// Verify listener was added
		count := emitter.ListenerCount(EventDownloadStarted)
		if count != 1 {
			t.Errorf("Expected 1 listener, got: %d", count)
		}

		// Test emitting to verify listener works
		event := Event{
			Type: EventDownloadStarted,
			Data: map[string]interface{}{"test": true},
		}
		emitter.EmitSync(event)

		if !executed {
			t.Error("Listener should have been executed")
		}
	})

	t.Run("OnceListener", func(t *testing.T) {
		executionCount := 0
		listener := func(event Event) {
			executionCount++
		}

		emitter.Once(EventDownloadCompleted, listener)

		// Verify listener was added
		count := emitter.ListenerCount(EventDownloadCompleted)
		if count != 1 {
			t.Errorf("Expected 1 listener, got: %d", count)
		}

		// Emit event twice
		event := Event{
			Type: EventDownloadCompleted,
			Data: map[string]interface{}{"status": "success"},
		}
		emitter.EmitSync(event)
		emitter.EmitSync(event)

		// Should only execute once
		if executionCount != 1 {
			t.Errorf("Once listener should execute only once, got: %d executions", executionCount)
		}

		// Listener should be removed after first execution
		count = emitter.ListenerCount(EventDownloadCompleted)
		if count != 0 {
			t.Errorf("Expected 0 listeners after once execution, got: %d", count)
		}
	})

	t.Run("OffListener", func(t *testing.T) {
		executed := false
		listener := func(event Event) {
			executed = true
		}

		// Add listener
		emitter.On(EventDownloadFailed, listener)

		// Verify it was added
		if emitter.ListenerCount(EventDownloadFailed) != 1 {
			t.Error("Expected listener to be added")
		}

		// Remove listener
		emitter.Off(EventDownloadFailed, listener)

		// Verify it was removed
		if emitter.ListenerCount(EventDownloadFailed) != 0 {
			t.Error("Expected listener to be removed")
		}

		// Emit event and verify listener doesn't execute
		event := Event{
			Type: EventDownloadFailed,
			Data: map[string]interface{}{"error": "test error"},
		}
		emitter.EmitSync(event)

		if executed {
			t.Error("Listener should not have been executed after removal")
		}
	})

	t.Run("RemoveAllListeners", func(t *testing.T) {
		emitter := NewEventEmitter()

		// Add multiple listeners for different event types
		for i := 0; i < 3; i++ {
			listener := func(event Event) {}
			emitter.On(EventDownloadStarted, listener)
			emitter.On(EventDownloadCompleted, listener)
		}

		// Verify listeners were added
		if emitter.ListenerCount(EventDownloadStarted) != 3 {
			t.Error("Expected 3 EventDownloadStarted listeners")
		}
		if emitter.ListenerCount(EventDownloadCompleted) != 3 {
			t.Error("Expected 3 EventDownloadCompleted listeners")
		}

		// Remove all listeners for one event type
		emitter.RemoveAllListeners(EventDownloadStarted)

		// Verify only EventDownloadStarted listeners were removed
		if emitter.ListenerCount(EventDownloadStarted) != 0 {
			t.Error("Expected all EventDownloadStarted listeners to be removed")
		}
		if emitter.ListenerCount(EventDownloadCompleted) != 3 {
			t.Error("Expected EventDownloadCompleted listeners to remain")
		}
	})

	t.Run("RemoveAllListenersGlobal", func(t *testing.T) {
		emitter := NewEventEmitter()

		// Add listeners for multiple event types
		listener := func(event Event) {}
		emitter.On(EventDownloadStarted, listener)
		emitter.On(EventDownloadCompleted, listener)
		emitter.On(EventDownloadFailed, listener)

		// Verify listeners were added
		eventTypes := emitter.EventTypes()
		if len(eventTypes) != 3 {
			t.Errorf("Expected 3 event types with listeners, got: %d", len(eventTypes))
		}

		// Remove all listeners
		emitter.RemoveAllListeners()

		// Verify all listeners were removed
		eventTypesAfter := emitter.EventTypes()
		if len(eventTypesAfter) != 0 {
			t.Errorf("Expected 0 event types after removing all, got: %d", len(eventTypesAfter))
		}
	})
}

// TestEventEmission tests event emission functionality
func TestEventEmission(t *testing.T) {
	t.Run("EmitSyncToSingleListener", func(t *testing.T) {
		emitter := NewEventEmitter()

		var receivedEvent Event
		listener := func(event Event) {
			receivedEvent = event
		}

		emitter.On(EventStorageSaved, listener)

		testEvent := Event{
			Type: EventStorageSaved,
			Data: map[string]interface{}{
				"filename": "test.txt",
				"size":     1024,
			},
			Source: "test-source",
		}

		emitter.EmitSync(testEvent)

		// Verify event was received correctly
		if receivedEvent.Type != EventStorageSaved {
			t.Errorf("Expected event type %s, got: %s", EventStorageSaved, receivedEvent.Type)
		}

		data := receivedEvent.Data.(map[string]interface{})
		if data["filename"] != "test.txt" {
			t.Errorf("Expected filename 'test.txt', got: %v", data["filename"])
		}

		if data["size"] != 1024 {
			t.Errorf("Expected size 1024, got: %v", data["size"])
		}

		if receivedEvent.Source != "test-source" {
			t.Errorf("Expected source 'test-source', got: %s", receivedEvent.Source)
		}

		// Verify timestamp was set
		if receivedEvent.Timestamp.IsZero() {
			t.Error("Event timestamp should be set")
		}
	})

	t.Run("EmitAsyncToMultipleListeners", func(t *testing.T) {
		emitter := NewEventEmitter()

		executed := make([]bool, 3)
		var mu sync.Mutex

		for i := 0; i < 3; i++ {
			idx := i // Capture loop variable
			listener := func(event Event) {
				mu.Lock()
				executed[idx] = true
				mu.Unlock()
			}
			emitter.On(EventPluginLoaded, listener)
		}

		// Verify all listeners were added
		count := emitter.ListenerCount(EventPluginLoaded)
		if count != 3 {
			t.Errorf("Expected 3 listeners, got: %d", count)
		}

		// Emit event asynchronously
		event := Event{
			Type: EventPluginLoaded,
			Data: map[string]interface{}{"plugin": "test"},
		}
		emitter.Emit(event)

		// Wait for async execution
		time.Sleep(50 * time.Millisecond)

		// Verify all listeners executed
		mu.Lock()
		for i, exec := range executed {
			if !exec {
				t.Errorf("Listener %d should have been executed", i)
			}
		}
		mu.Unlock()
	})

	t.Run("EmitWithNoListeners", func(t *testing.T) {
		emitter := NewEventEmitter()

		// Emitting to an event type with no listeners should not error or panic
		event := Event{
			Type: EventAuthSuccess,
			Data: map[string]interface{}{"user": "test"},
		}

		// Should not panic
		emitter.Emit(event)
		emitter.EmitSync(event)
	})

	t.Run("EmitWithContext", func(t *testing.T) {
		emitter := NewEventEmitter()

		var executed int32
		listener := func(event Event) {
			atomic.StoreInt32(&executed, 1)
		}

		emitter.On(EventSystemStarted, listener)

		event := Event{
			Type: EventSystemStarted,
			Data: map[string]interface{}{"version": "1.0.0"},
		}

		ctx := context.Background()
		err := emitter.EmitWithContext(ctx, event)
		if err != nil {
			t.Errorf("Failed to emit event with context: %v", err)
		}

		// Give time for async execution
		time.Sleep(10 * time.Millisecond)

		if atomic.LoadInt32(&executed) == 0 {
			t.Error("Listener should have been executed")
		}
	})

	t.Run("EmitWithCancelledContext", func(t *testing.T) {
		emitter := NewEventEmitter()

		ctx, cancel := context.WithCancel(context.Background())
		cancel() // Cancel immediately

		event := Event{
			Type: EventSystemError,
			Data: map[string]interface{}{"error": "test"},
		}

		err := emitter.EmitWithContext(ctx, event)
		if err == nil {
			t.Error("Expected error when emitting with cancelled context")
		}

		if err != context.Canceled {
			t.Errorf("Expected context.Canceled error, got: %v", err)
		}
	})
}

// TestWaitForEvent tests waiting for events
func TestWaitForEvent(t *testing.T) {
	t.Run("WaitForEventSuccess", func(t *testing.T) {
		emitter := NewEventEmitter()

		// Start waiting for event in goroutine
		ctx := context.Background()
		eventChan := make(chan Event, 1)
		errChan := make(chan error, 1)

		go func() {
			event, err := emitter.WaitForEvent(ctx, EventCacheHit)
			if err != nil {
				errChan <- err
			} else {
				eventChan <- event
			}
		}()

		// Give a moment for the wait to be set up
		time.Sleep(10 * time.Millisecond)

		// Emit the expected event
		testEvent := Event{
			Type: EventCacheHit,
			Data: map[string]interface{}{"key": "test-key"},
		}
		emitter.Emit(testEvent)

		// Wait for result
		select {
		case receivedEvent := <-eventChan:
			if receivedEvent.Type != EventCacheHit {
				t.Errorf("Expected event type %s, got: %s", EventCacheHit, receivedEvent.Type)
			}
		case err := <-errChan:
			t.Errorf("Unexpected error waiting for event: %v", err)
		case <-time.After(100 * time.Millisecond):
			t.Error("Timed out waiting for event")
		}
	})

	t.Run("WaitForEventTimeout", func(t *testing.T) {
		emitter := NewEventEmitter()

		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
		defer cancel()

		_, err := emitter.WaitForEvent(ctx, EventCacheMiss)
		if err == nil {
			t.Error("Expected timeout error")
		}

		if err != context.DeadlineExceeded {
			t.Errorf("Expected context.DeadlineExceeded, got: %v", err)
		}
	})
}

// TestCreateEvent tests event creation helpers
func TestCreateEvent(t *testing.T) {
	t.Run("CreateEvent", func(t *testing.T) {
		data := map[string]interface{}{"test": "value"}
		source := "test-source"

		event := CreateEvent(EventDownloadProgress, data, source)

		if event.Type != EventDownloadProgress {
			t.Errorf("Expected type %s, got: %s", EventDownloadProgress, event.Type)
		}

		eventData, ok := event.Data.(map[string]interface{})
		if !ok {
			t.Error("Event data should be map[string]interface{}")
		} else if eventData["test"] != "value" {
			t.Error("Event data should match input")
		}

		if event.Source != source {
			t.Errorf("Expected source '%s', got: '%s'", source, event.Source)
		}

		if event.Timestamp.IsZero() {
			t.Error("Event timestamp should be set")
		}

		if event.Metadata == nil {
			t.Error("Event metadata should be initialized")
		}

		if event.ID != "" {
			t.Error("Event ID should be empty when not specified")
		}
	})

	t.Run("CreateEventWithID", func(t *testing.T) {
		data := "test data"
		source := "test-source"
		id := "test-id-123"

		event := CreateEventWithID(EventAuthFailure, data, source, id)

		if event.Type != EventAuthFailure {
			t.Errorf("Expected type %s, got: %s", EventAuthFailure, event.Type)
		}

		if event.ID != id {
			t.Errorf("Expected ID '%s', got: '%s'", id, event.ID)
		}

		if event.Data != data {
			t.Error("Event data should match input")
		}

		if event.Source != source {
			t.Errorf("Expected source '%s', got: '%s'", source, event.Source)
		}
	})
}

// TestFilteredEventEmitter tests filtered event emission
func TestFilteredEventEmitter(t *testing.T) {
	t.Run("FilteredEmission", func(t *testing.T) {
		baseEmitter := NewEventEmitter()

		// Create filter that only allows download events
		filter := func(event Event) bool {
			return strings.HasPrefix(string(event.Type), "download")
		}

		filteredEmitter := NewFilteredEventEmitter(baseEmitter, filter)

		var executed int32
		listener := func(event Event) {
			atomic.StoreInt32(&executed, 1)
		}

		filteredEmitter.On(EventDownloadStarted, listener)

		// This should pass the filter
		downloadEvent := Event{
			Type: EventDownloadStarted,
			Data: "test",
		}
		filteredEmitter.Emit(downloadEvent)

		time.Sleep(10 * time.Millisecond)

		if atomic.LoadInt32(&executed) == 0 {
			t.Error("Download event should have passed filter and executed listener")
		}

		// Reset
		atomic.StoreInt32(&executed, 0)

		// This should be filtered out
		authEvent := Event{
			Type: EventAuthSuccess,
			Data: "test",
		}
		filteredEmitter.Emit(authEvent)

		time.Sleep(10 * time.Millisecond)

		if atomic.LoadInt32(&executed) != 0 {
			t.Error("Auth event should have been filtered out")
		}
	})
}

// TestEmitterClose tests closing the emitter
func TestEmitterClose(t *testing.T) {
	t.Run("CloseEmitter", func(t *testing.T) {
		emitter := NewEventEmitter()

		// Add a listener
		listener := func(event Event) {}
		emitter.On(EventSystemShutdown, listener)

		// Verify listener was added
		if emitter.ListenerCount(EventSystemShutdown) != 1 {
			t.Error("Expected listener to be added")
		}

		if emitter.IsClosed() {
			t.Error("Emitter should not be closed initially")
		}

		// Close emitter
		emitter.Close()

		// Verify emitter is closed
		if !emitter.IsClosed() {
			t.Error("Emitter should be closed")
		}

		// Verify all listeners were removed
		if emitter.ListenerCount(EventSystemShutdown) != 0 {
			t.Error("Expected all listeners to be removed on close")
		}

		// Adding listeners after close should be ignored
		emitter.On(EventSystemStarted, listener)
		if emitter.ListenerCount(EventSystemStarted) != 0 {
			t.Error("Adding listeners after close should be ignored")
		}

		// Emitting after close should not cause issues
		event := Event{Type: EventSystemError}
		emitter.Emit(event)
		emitter.EmitSync(event)
	})
}

// TestConcurrentOperations tests thread safety
func TestConcurrentOperations(t *testing.T) {
	t.Run("ConcurrentAddListeners", func(t *testing.T) {
		emitter := NewEventEmitter()

		var wg sync.WaitGroup
		numGoroutines := 10

		// Concurrently add listeners
		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				listener := func(event Event) {}
				emitter.On(EventDownloadStarted, listener)
			}(i)
		}

		wg.Wait()

		// Should have added all listeners without race conditions
		count := emitter.ListenerCount(EventDownloadStarted)
		if count != numGoroutines {
			t.Errorf("Expected %d listeners, got: %d", numGoroutines, count)
		}
	})

	t.Run("ConcurrentEmissions", func(t *testing.T) {
		emitter := NewEventEmitter()

		executionCount := int64(0)
		var mu sync.Mutex

		listener := func(event Event) {
			mu.Lock()
			executionCount++
			mu.Unlock()
		}

		emitter.On(EventProtocolConnected, listener)

		var wg sync.WaitGroup
		numEmissions := 50

		for i := 0; i < numEmissions; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				event := Event{
					Type: EventProtocolConnected,
					Data: map[string]interface{}{"id": id},
				}
				emitter.EmitSync(event)
			}(i)
		}

		wg.Wait()

		mu.Lock()
		finalCount := executionCount
		mu.Unlock()

		if finalCount != int64(numEmissions) {
			t.Errorf("Expected %d executions, got: %d", numEmissions, finalCount)
		}
	})
}

// TestEventTypes tests all event type constants
func TestEventTypes(t *testing.T) {
	expectedEventTypes := []EventType{
		EventDownloadStarted,
		EventDownloadProgress,
		EventDownloadCompleted,
		EventDownloadFailed,
		EventDownloadPaused,
		EventDownloadResumed,
		EventDownloadCancelled,
		EventChunkStarted,
		EventChunkCompleted,
		EventChunkFailed,
		EventStorageSaved,
		EventStorageLoaded,
		EventStorageDeleted,
		EventStorageFailed,
		EventProtocolConnected,
		EventProtocolDisconnected,
		EventProtocolError,
		EventPluginLoaded,
		EventPluginUnloaded,
		EventPluginError,
		EventSystemStarted,
		EventSystemShutdown,
		EventSystemError,
		EventCacheHit,
		EventCacheMiss,
		EventCacheEvict,
		EventAuthSuccess,
		EventAuthFailure,
		EventAuthExpired,
	}

	t.Run("EventTypeConstants", func(t *testing.T) {
		emitter := NewEventEmitter()

		// Test that we can add listeners for all event types
		for _, eventType := range expectedEventTypes {
			listener := func(event Event) {}
			emitter.On(eventType, listener)

			// Verify listener was added
			if emitter.ListenerCount(eventType) != 1 {
				t.Errorf("Failed to add listener for event type: %s", eventType)
			}
		}
	})

	t.Run("EventTypeStrings", func(t *testing.T) {
		// Verify event types have meaningful string representations
		for _, eventType := range expectedEventTypes {
			str := string(eventType)
			if len(str) == 0 {
				t.Errorf("Event type %v should have non-empty string representation", eventType)
			}
		}
	})
}

// TestListenerPanic tests panic handling in listeners
func TestListenerPanic(t *testing.T) {
	t.Run("ListenerPanicHandling", func(t *testing.T) {
		emitter := NewEventEmitter()

		// Add a listener that panics
		panicListener := func(event Event) {
			panic("test panic")
		}

		// Add a normal listener
		var executed int32
		normalListener := func(event Event) {
			atomic.StoreInt32(&executed, 1)
		}

		emitter.On(EventSystemError, panicListener)
		emitter.On(EventSystemError, normalListener)

		// Emitting should not cause the test to fail due to panic
		event := Event{Type: EventSystemError}
		emitter.Emit(event)

		// Give time for async execution
		time.Sleep(20 * time.Millisecond)

		// Normal listener should still execute despite panic in other listener
		if atomic.LoadInt32(&executed) == 0 {
			t.Error("Normal listener should execute despite panic in other listener")
		}
	})
}

// Benchmark tests for performance
func BenchmarkEventEmitter(b *testing.B) {
	emitter := NewEventEmitter()

	listener := func(event Event) {
		// Simple no-op listener
	}

	emitter.On(EventDownloadProgress, listener)

	b.Run("Emit", func(b *testing.B) {
		event := Event{
			Type: EventDownloadProgress,
			Data: map[string]interface{}{"progress": 50},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			emitter.Emit(event)
		}
	})

	b.Run("EmitSync", func(b *testing.B) {
		event := Event{
			Type: EventDownloadProgress,
			Data: map[string]interface{}{"progress": 50},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			emitter.EmitSync(event)
		}
	})

	b.Run("On", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			listener := func(event Event) {}
			emitter.On(EventDownloadCompleted, listener)
		}
	})
}
