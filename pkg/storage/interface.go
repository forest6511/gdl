package storage

import (
	"context"
	"io"
)

// StorageBackend defines the interface for different storage backends
// This allows for flexible storage options like local filesystem, cloud storage, etc.
type StorageBackend interface {
	// Init initializes the storage backend with configuration
	Init(config map[string]interface{}) error

	// Save stores data to the storage backend at the specified key/path
	Save(ctx context.Context, key string, data io.Reader) error

	// Load retrieves data from the storage backend for the given key/path
	Load(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes data from the storage backend for the given key/path
	Delete(ctx context.Context, key string) error

	// Exists checks if data exists at the given key/path in the storage backend
	Exists(ctx context.Context, key string) (bool, error)

	// List returns a list of keys with the given prefix
	List(ctx context.Context, prefix string) ([]string, error)

	// Close cleans up any resources used by the storage backend
	Close() error
}

// StorageConfig holds common configuration options for storage backends
type StorageConfig struct {
	// Type specifies the storage backend type (filesystem, s3, redis, memory)
	Type string `json:"type" yaml:"type"`

	// Config holds backend-specific configuration
	Config map[string]interface{} `json:"config" yaml:"config"`
}

// StorageManager manages multiple storage backends
type StorageManager struct {
	backends    map[string]StorageBackend
	defaultName string
}

// NewStorageManager creates a new storage manager
func NewStorageManager() *StorageManager {
	return &StorageManager{
		backends: make(map[string]StorageBackend),
	}
}

// Register registers a storage backend with a given name
func (sm *StorageManager) Register(name string, backend StorageBackend) error {
	sm.backends[name] = backend

	// Set as default if it's the first one registered
	if sm.defaultName == "" {
		sm.defaultName = name
	}

	return nil
}

// SetDefault sets the default storage backend
func (sm *StorageManager) SetDefault(name string) error {
	if _, exists := sm.backends[name]; !exists {
		return ErrBackendNotFound
	}
	sm.defaultName = name
	return nil
}

// GetBackend returns a storage backend by name
func (sm *StorageManager) GetBackend(name string) (StorageBackend, error) {
	backend, exists := sm.backends[name]
	if !exists {
		return nil, ErrBackendNotFound
	}
	return backend, nil
}

// GetDefault returns the default storage backend
func (sm *StorageManager) GetDefault() (StorageBackend, error) {
	if sm.defaultName == "" {
		return nil, ErrNoDefaultBackend
	}
	return sm.GetBackend(sm.defaultName)
}

// Save saves data using the default backend
func (sm *StorageManager) Save(ctx context.Context, key string, data io.Reader) error {
	backend, err := sm.GetDefault()
	if err != nil {
		return err
	}
	return backend.Save(ctx, key, data)
}

// Load loads data using the default backend
func (sm *StorageManager) Load(ctx context.Context, key string) (io.ReadCloser, error) {
	backend, err := sm.GetDefault()
	if err != nil {
		return nil, err
	}
	return backend.Load(ctx, key)
}

// Delete deletes data using the default backend
func (sm *StorageManager) Delete(ctx context.Context, key string) error {
	backend, err := sm.GetDefault()
	if err != nil {
		return err
	}
	return backend.Delete(ctx, key)
}

// Exists checks if data exists using the default backend
func (sm *StorageManager) Exists(ctx context.Context, key string) (bool, error) {
	backend, err := sm.GetDefault()
	if err != nil {
		return false, err
	}
	return backend.Exists(ctx, key)
}

// List lists keys using the default backend
func (sm *StorageManager) List(ctx context.Context, prefix string) ([]string, error) {
	backend, err := sm.GetDefault()
	if err != nil {
		return nil, err
	}
	return backend.List(ctx, prefix)
}

// Close closes all registered backends
func (sm *StorageManager) Close() error {
	var lastErr error
	for _, backend := range sm.backends {
		if err := backend.Close(); err != nil {
			lastErr = err
		}
	}
	return lastErr
}
