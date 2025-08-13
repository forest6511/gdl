package backends

import (
	"bytes"
	"context"
	"io"
	"strings"
	"sync"

	"github.com/forest6511/godl/pkg/storage"
)

// MemoryBackend implements in-memory storage for testing and temporary use
type MemoryBackend struct {
	data map[string][]byte
	mu   sync.RWMutex
}

// NewMemoryBackend creates a new in-memory storage backend
func NewMemoryBackend() *MemoryBackend {
	return &MemoryBackend{
		data: make(map[string][]byte),
	}
}

// Init initializes the memory backend (no-op, but maintains interface compatibility)
func (m *MemoryBackend) Init(config map[string]interface{}) error {
	// Memory backend doesn't require any configuration
	// Reset data if this is a re-initialization
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string][]byte)
	return nil
}

// Save stores data in memory at the specified key
func (m *MemoryBackend) Save(ctx context.Context, key string, data io.Reader) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Read all data into memory
	dataBytes, err := io.ReadAll(data)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Make a copy to avoid any potential issues with shared slices
	dataCopy := make([]byte, len(dataBytes))
	copy(dataCopy, dataBytes)
	m.data[key] = dataCopy

	return nil
}

// Load retrieves data from memory for the given key
func (m *MemoryBackend) Load(ctx context.Context, key string) (io.ReadCloser, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	data, exists := m.data[key]
	m.mu.RUnlock()

	if !exists {
		return nil, storage.ErrKeyNotFound
	}

	// Return a copy to avoid modification of stored data
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	return io.NopCloser(bytes.NewReader(dataCopy)), nil
}

// Delete removes data from memory for the given key
func (m *MemoryBackend) Delete(ctx context.Context, key string) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.data[key]; !exists {
		return storage.ErrKeyNotFound
	}

	delete(m.data, key)
	return nil
}

// Exists checks if data exists at the given key in memory
func (m *MemoryBackend) Exists(ctx context.Context, key string) (bool, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return false, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	_, exists := m.data[key]
	return exists, nil
}

// List returns a list of keys with the given prefix
func (m *MemoryBackend) List(ctx context.Context, prefix string) ([]string, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	var keys []string
	for key := range m.data {
		if prefix == "" || strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// Close cleans up resources (clears memory)
func (m *MemoryBackend) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Clear all data
	m.data = make(map[string][]byte)
	return nil
}

// Size returns the number of items stored
func (m *MemoryBackend) Size() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.data)
}

// MemoryUsage returns the approximate memory usage in bytes
func (m *MemoryBackend) MemoryUsage() int64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var total int64
	for key, value := range m.data {
		total += int64(len(key) + len(value))
	}
	return total
}

// Clear removes all data from the backend
func (m *MemoryBackend) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.data = make(map[string][]byte)
}

// Keys returns all keys currently stored
func (m *MemoryBackend) Keys() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]string, 0, len(m.data))
	for key := range m.data {
		keys = append(keys, key)
	}
	return keys
}

// GetSize returns the size of data stored at the given key
func (m *MemoryBackend) GetSize(ctx context.Context, key string) (int64, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return 0, ctx.Err()
	default:
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.data[key]
	if !exists {
		return 0, storage.ErrKeyNotFound
	}

	return int64(len(data)), nil
}

// BatchSave saves multiple key-value pairs in a single operation
func (m *MemoryBackend) BatchSave(ctx context.Context, items map[string]io.Reader) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	// Read all data first to avoid partial updates on error
	dataMap := make(map[string][]byte, len(items))
	for key, reader := range items {
		data, err := io.ReadAll(reader)
		if err != nil {
			return err
		}
		dataMap[key] = data
	}

	// Now update in memory atomically
	m.mu.Lock()
	defer m.mu.Unlock()

	for key, data := range dataMap {
		// Make a copy to avoid any potential issues with shared slices
		dataCopy := make([]byte, len(data))
		copy(dataCopy, data)
		m.data[key] = dataCopy
	}

	return nil
}

// BatchDelete removes multiple keys in a single operation
func (m *MemoryBackend) BatchDelete(ctx context.Context, keys []string) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, key := range keys {
		delete(m.data, key)
	}

	return nil
}

// Clone creates a deep copy of the memory backend
func (m *MemoryBackend) Clone() *MemoryBackend {
	m.mu.RLock()
	defer m.mu.RUnlock()

	clone := NewMemoryBackend()
	for key, value := range m.data {
		dataCopy := make([]byte, len(value))
		copy(dataCopy, value)
		clone.data[key] = dataCopy
	}

	return clone
}
