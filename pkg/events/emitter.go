package events

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// EventType represents different types of events in the system
type EventType string

const (
	// Download events
	EventDownloadStarted   EventType = "download_started"
	EventDownloadProgress  EventType = "download_progress"
	EventDownloadCompleted EventType = "download_completed"
	EventDownloadFailed    EventType = "download_failed"
	EventDownloadPaused    EventType = "download_paused"
	EventDownloadResumed   EventType = "download_resumed"
	EventDownloadCancelled EventType = "download_cancelled"

	// Chunk events
	EventChunkStarted   EventType = "chunk_started"
	EventChunkCompleted EventType = "chunk_completed"
	EventChunkFailed    EventType = "chunk_failed"

	// Storage events
	EventStorageSaved   EventType = "storage_saved"
	EventStorageLoaded  EventType = "storage_loaded"
	EventStorageDeleted EventType = "storage_deleted"
	EventStorageFailed  EventType = "storage_failed"

	// Protocol events
	EventProtocolConnected    EventType = "protocol_connected"
	EventProtocolDisconnected EventType = "protocol_disconnected"
	EventProtocolError        EventType = "protocol_error"

	// Plugin events
	EventPluginLoaded   EventType = "plugin_loaded"
	EventPluginUnloaded EventType = "plugin_unloaded"
	EventPluginError    EventType = "plugin_error"

	// System events
	EventSystemStarted  EventType = "system_started"
	EventSystemShutdown EventType = "system_shutdown"
	EventSystemError    EventType = "system_error"

	// Cache events
	EventCacheHit   EventType = "cache_hit"
	EventCacheMiss  EventType = "cache_miss"
	EventCacheEvict EventType = "cache_evict"

	// Authentication events
	EventAuthSuccess EventType = "auth_success"
	EventAuthFailure EventType = "auth_failure"
	EventAuthExpired EventType = "auth_expired"
)

// Event represents an event that occurs in the system
type Event struct {
	Type      EventType              `json:"type"`
	Timestamp time.Time              `json:"timestamp"`
	Data      interface{}            `json:"data"`
	Source    string                 `json:"source"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
	ID        string                 `json:"id,omitempty"`
}

// EventListener is a function that handles events
type EventListener func(event Event)

// EventEmitter manages event listeners and emits events
type EventEmitter struct {
	listeners map[EventType][]listenerEntry
	mu        sync.RWMutex
	closed    bool
}

// listenerEntry holds a listener function and whether it should only run once
type listenerEntry struct {
	listener EventListener
	once     bool
}

// NewEventEmitter creates a new event emitter
func NewEventEmitter() *EventEmitter {
	return &EventEmitter{
		listeners: make(map[EventType][]listenerEntry),
	}
}

// On adds an event listener for the specified event type
func (ee *EventEmitter) On(eventType EventType, listener EventListener) {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	if ee.closed {
		return
	}

	ee.listeners[eventType] = append(ee.listeners[eventType], listenerEntry{
		listener: listener,
		once:     false,
	})
}

// Off removes the most recently added listener for the specified event type
// Note: Due to Go's limitations with function comparison, this removes the last added listener
func (ee *EventEmitter) Off(eventType EventType, listener EventListener) {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	if ee.closed {
		return
	}

	listeners := ee.listeners[eventType]
	if len(listeners) > 0 {
		// Remove the last listener (most recently added)
		ee.listeners[eventType] = listeners[:len(listeners)-1]
	}

	// Clean up empty slices
	if len(ee.listeners[eventType]) == 0 {
		delete(ee.listeners, eventType)
	}
}

// Once adds an event listener that will only be called once
func (ee *EventEmitter) Once(eventType EventType, listener EventListener) {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	if ee.closed {
		return
	}

	ee.listeners[eventType] = append(ee.listeners[eventType], listenerEntry{
		listener: listener,
		once:     true,
	})
}

// Emit emits an event to all registered listeners
func (ee *EventEmitter) Emit(event Event) {
	ee.mu.RLock()
	if ee.closed {
		ee.mu.RUnlock()
		return
	}

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Get listeners for this event type
	listeners := make([]listenerEntry, len(ee.listeners[event.Type]))
	copy(listeners, ee.listeners[event.Type])
	ee.mu.RUnlock()

	// Track which listeners should be removed (once listeners)
	var toRemove []int

	// Use a worker pool to prevent excessive goroutine creation
	const maxWorkers = 10
	if len(listeners) > maxWorkers {
		ee.emitWithWorkerPool(event, listeners, &toRemove)
	} else {
		// For small number of listeners, use direct goroutines
		for i, entry := range listeners {
			go func(l EventListener, e Event) {
				defer func() {
					if r := recover(); r != nil {
						// Handle panics in listeners gracefully
						fmt.Printf("Event listener panic: %v\n", r)
					}
				}()
				l(e)
			}(entry.listener, event)

			// Mark once listeners for removal
			if entry.once {
				toRemove = append(toRemove, i)
			}
		}
	}

	// Remove once listeners
	if len(toRemove) > 0 {
		ee.removeOnceListeners(event.Type, toRemove)
	}
}

// emitWithWorkerPool uses a worker pool for efficient event emission
func (ee *EventEmitter) emitWithWorkerPool(event Event, listeners []listenerEntry, toRemove *[]int) {
	workerCount := 10
	if len(listeners) < workerCount {
		workerCount = len(listeners)
	}

	type job struct {
		index int
		entry listenerEntry
	}

	jobs := make(chan job, len(listeners))
	var wg sync.WaitGroup

	// Start workers
	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := range jobs {
				func() {
					defer func() {
						if r := recover(); r != nil {
							fmt.Printf("Event listener panic: %v\n", r)
						}
					}()
					j.entry.listener(event)
				}()

				if j.entry.once {
					ee.mu.Lock()
					*toRemove = append(*toRemove, j.index)
					ee.mu.Unlock()
				}
			}
		}()
	}

	// Send jobs
	for i, entry := range listeners {
		jobs <- job{index: i, entry: entry}
	}
	close(jobs)

	wg.Wait()
}

// removeOnceListeners removes listeners marked as once
func (ee *EventEmitter) removeOnceListeners(eventType EventType, toRemove []int) {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	// Remove in reverse order to maintain indices
	for i := len(toRemove) - 1; i >= 0; i-- {
		idx := toRemove[i]
		if idx < len(ee.listeners[eventType]) {
			ee.listeners[eventType] = append(
				ee.listeners[eventType][:idx],
				ee.listeners[eventType][idx+1:]...,
			)
		}
	}

	// Clean up empty slices
	if len(ee.listeners[eventType]) == 0 {
		delete(ee.listeners, eventType)
	}
}

// EmitSync emits an event synchronously (waits for all listeners to complete)
func (ee *EventEmitter) EmitSync(event Event) {
	ee.mu.RLock()
	if ee.closed {
		ee.mu.RUnlock()
		return
	}

	// Set timestamp if not already set
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}

	// Get listeners for this event type
	listeners := make([]listenerEntry, len(ee.listeners[event.Type]))
	copy(listeners, ee.listeners[event.Type])
	ee.mu.RUnlock()

	// Track which listeners should be removed (once listeners)
	var toRemove []int

	// Call each listener synchronously
	for i, entry := range listeners {
		func(l EventListener, e Event) {
			defer func() {
				if r := recover(); r != nil {
					// Handle panics in listeners gracefully
					fmt.Printf("Event listener panic: %v\n", r)
				}
			}()
			l(e)
		}(entry.listener, event)

		// Mark once listeners for removal
		if entry.once {
			toRemove = append(toRemove, i)
		}
	}

	// Remove once listeners
	if len(toRemove) > 0 {
		ee.mu.Lock()
		// Remove in reverse order to maintain indices
		for i := len(toRemove) - 1; i >= 0; i-- {
			idx := toRemove[i]
			if idx < len(ee.listeners[event.Type]) {
				ee.listeners[event.Type] = append(
					ee.listeners[event.Type][:idx],
					ee.listeners[event.Type][idx+1:]...,
				)
			}
		}

		// Clean up empty slices
		if len(ee.listeners[event.Type]) == 0 {
			delete(ee.listeners, event.Type)
		}
		ee.mu.Unlock()
	}
}

// ListenerCount returns the number of listeners for a specific event type
func (ee *EventEmitter) ListenerCount(eventType EventType) int {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	return len(ee.listeners[eventType])
}

// EventTypes returns all event types that have listeners
func (ee *EventEmitter) EventTypes() []EventType {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	types := make([]EventType, 0, len(ee.listeners))
	for eventType := range ee.listeners {
		types = append(types, eventType)
	}

	return types
}

// RemoveAllListeners removes all listeners for a specific event type,
// or all listeners if no event type is specified
func (ee *EventEmitter) RemoveAllListeners(eventType ...EventType) {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	if len(eventType) == 0 {
		// Remove all listeners
		ee.listeners = make(map[EventType][]listenerEntry)
	} else {
		// Remove listeners for specific event types
		for _, et := range eventType {
			delete(ee.listeners, et)
		}
	}
}

// Close closes the event emitter and prevents new listeners from being added
func (ee *EventEmitter) Close() {
	ee.mu.Lock()
	defer ee.mu.Unlock()

	ee.closed = true
	ee.listeners = make(map[EventType][]listenerEntry)
}

// IsClosed returns whether the event emitter is closed
func (ee *EventEmitter) IsClosed() bool {
	ee.mu.RLock()
	defer ee.mu.RUnlock()

	return ee.closed
}

// EmitWithContext emits an event with a context for cancellation
func (ee *EventEmitter) EmitWithContext(ctx context.Context, event Event) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	ee.Emit(event)
	return nil
}

// WaitForEvent waits for a specific event type to be emitted
func (ee *EventEmitter) WaitForEvent(ctx context.Context, eventType EventType) (Event, error) {
	eventChan := make(chan Event, 1)

	// Add a once listener
	listener := func(event Event) {
		select {
		case eventChan <- event:
		default:
			// Channel is full, ignore
		}
	}

	ee.Once(eventType, listener)

	select {
	case event := <-eventChan:
		return event, nil
	case <-ctx.Done():
		return Event{}, ctx.Err()
	}
}

// CreateEvent is a helper function to create a new event
func CreateEvent(eventType EventType, data interface{}, source string) Event {
	return Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
		Source:    source,
		Metadata:  make(map[string]interface{}),
	}
}

// CreateEventWithID creates a new event with a specific ID
func CreateEventWithID(eventType EventType, data interface{}, source, id string) Event {
	return Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Data:      data,
		Source:    source,
		ID:        id,
		Metadata:  make(map[string]interface{}),
	}
}

// EventFilter represents a function that can filter events
type EventFilter func(event Event) bool

// FilteredEventEmitter wraps an EventEmitter with filtering capabilities
type FilteredEventEmitter struct {
	emitter *EventEmitter
	filter  EventFilter
}

// NewFilteredEventEmitter creates a new filtered event emitter
func NewFilteredEventEmitter(emitter *EventEmitter, filter EventFilter) *FilteredEventEmitter {
	return &FilteredEventEmitter{
		emitter: emitter,
		filter:  filter,
	}
}

// Emit emits an event only if it passes the filter
func (fee *FilteredEventEmitter) Emit(event Event) {
	if fee.filter(event) {
		fee.emitter.Emit(event)
	}
}

// On adds a listener to the underlying emitter
func (fee *FilteredEventEmitter) On(eventType EventType, listener EventListener) {
	fee.emitter.On(eventType, listener)
}

// Off removes a listener from the underlying emitter
func (fee *FilteredEventEmitter) Off(eventType EventType, listener EventListener) {
	fee.emitter.Off(eventType, listener)
}

// Once adds a once listener to the underlying emitter
func (fee *FilteredEventEmitter) Once(eventType EventType, listener EventListener) {
	fee.emitter.Once(eventType, listener)
}
