package storage

import (
	"context"
	"errors"
	"io"
	"strings"
	"sync"
	"testing"
)

// Mock storage backend for testing
type mockStorageBackend struct {
	name     string
	files    map[string][]byte
	failSave bool
	failLoad bool
	mu       sync.RWMutex
}

func newMockStorage(name string) *mockStorageBackend {
	return &mockStorageBackend{
		name:  name,
		files: make(map[string][]byte),
	}
}

func (m *mockStorageBackend) Init(config map[string]interface{}) error {
	return nil
}

func (m *mockStorageBackend) Save(ctx context.Context, path string, reader io.Reader) error {
	if m.failSave {
		return errors.New("mock save error")
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.files[path] = data
	return nil
}

func (m *mockStorageBackend) Load(ctx context.Context, path string) (io.ReadCloser, error) {
	if m.failLoad {
		return nil, errors.New("mock load error")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()
	data, exists := m.files[path]
	if !exists {
		return nil, errors.New("file not found")
	}

	return io.NopCloser(strings.NewReader(string(data))), nil
}

func (m *mockStorageBackend) Delete(ctx context.Context, path string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.files, path)
	return nil
}

func (m *mockStorageBackend) Exists(ctx context.Context, path string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, exists := m.files[path]
	return exists, nil
}

func (m *mockStorageBackend) List(ctx context.Context, prefix string) ([]string, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var files []string
	for path := range m.files {
		if strings.HasPrefix(path, prefix) {
			files = append(files, path)
		}
	}
	return files, nil
}

func (m *mockStorageBackend) Close() error {
	return nil
}

func TestStorageManager(t *testing.T) {
	t.Run("NewStorageManager", func(t *testing.T) {
		manager := NewStorageManager()
		if manager == nil {
			t.Fatal("Expected storage manager to be created")
		}
	})

	t.Run("RegisterBackend", func(t *testing.T) {
		manager := NewStorageManager()
		mockBackend := newMockStorage("test")

		err := manager.Register("test", mockBackend)
		if err != nil {
			t.Errorf("Failed to register backend: %v", err)
		}

		// Test duplicate registration (might be allowed in this implementation)
		err = manager.Register("test", mockBackend)
		if err != nil {
			t.Logf("Duplicate registration returned error: %v", err)
		}
	})

	t.Run("SetDefaultBackend", func(t *testing.T) {
		manager := NewStorageManager()
		mockBackend := newMockStorage("default")

		err := manager.Register("default", mockBackend)
		if err != nil {
			t.Fatalf("Failed to register backend: %v", err)
		}

		err = manager.SetDefault("default")
		if err != nil {
			t.Errorf("Failed to set default backend: %v", err)
		}

		// Test setting non-existent backend as default
		err = manager.SetDefault("nonexistent")
		if err == nil {
			t.Error("Expected error when setting non-existent backend as default")
		}
	})

	t.Run("GetBackend", func(t *testing.T) {
		manager := NewStorageManager()
		mockBackend := newMockStorage("get-test")

		err := manager.Register("get-test", mockBackend)
		if err != nil {
			t.Fatalf("Failed to register backend: %v", err)
		}

		backend, err := manager.GetBackend("get-test")
		if err != nil {
			t.Errorf("Failed to get backend: %v", err)
		}
		if backend == nil {
			t.Error("Expected to get registered backend")
		}

		// Test getting non-existent backend
		backend, err = manager.GetBackend("nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent backend")
		}
		if backend != nil {
			t.Error("Expected nil for non-existent backend")
		}
	})

	t.Run("GetDefaultBackend", func(t *testing.T) {
		manager := NewStorageManager()
		mockBackend := newMockStorage("default-test")

		err := manager.Register("default-test", mockBackend)
		if err != nil {
			t.Fatalf("Failed to register backend: %v", err)
		}

		err = manager.SetDefault("default-test")
		if err != nil {
			t.Fatalf("Failed to set default backend: %v", err)
		}

		backend, err := manager.GetDefault()
		if err != nil {
			t.Errorf("Failed to get default backend: %v", err)
		}
		if backend == nil {
			t.Error("Expected to get default backend")
		}
	})
}

func TestStorageOperations(t *testing.T) {
	manager := NewStorageManager()
	mockBackend := newMockStorage("ops-test")

	err := manager.Register("ops-test", mockBackend)
	if err != nil {
		t.Fatalf("Failed to register backend: %v", err)
	}

	err = manager.SetDefault("ops-test")
	if err != nil {
		t.Fatalf("Failed to set default backend: %v", err)
	}

	ctx := context.Background()

	t.Run("Save", func(t *testing.T) {
		data := strings.NewReader("test data")
		err := manager.Save(ctx, "test/file.txt", data)
		if err != nil {
			t.Errorf("Failed to save file: %v", err)
		}
	})

	t.Run("Exists", func(t *testing.T) {
		exists, err := manager.Exists(ctx, "test/file.txt")
		if err != nil {
			t.Errorf("Failed to check existence: %v", err)
		}

		if !exists {
			t.Error("Expected file to exist")
		}

		// Check non-existent file
		exists, err = manager.Exists(ctx, "nonexistent.txt")
		if err != nil {
			t.Errorf("Failed to check non-existent file: %v", err)
		}

		if exists {
			t.Error("Expected file to not exist")
		}
	})

	t.Run("Load", func(t *testing.T) {
		reader, err := manager.Load(ctx, "test/file.txt")
		if err != nil {
			t.Errorf("Failed to load file: %v", err)
		}
		defer func() { _ = reader.Close() }()

		data, err := io.ReadAll(reader)
		if err != nil {
			t.Errorf("Failed to read loaded data: %v", err)
		}

		if string(data) != "test data" {
			t.Errorf("Expected 'test data', got '%s'", string(data))
		}
	})

	t.Run("List", func(t *testing.T) {
		// Save another file for listing test
		data := strings.NewReader("another file")
		err := manager.Save(ctx, "test/another.txt", data)
		if err != nil {
			t.Errorf("Failed to save second file: %v", err)
		}

		files, err := manager.List(ctx, "test/")
		if err != nil {
			t.Errorf("Failed to list files: %v", err)
		}

		if len(files) < 2 {
			t.Errorf("Expected at least 2 files, got %d", len(files))
		}

		// Check that our files are in the list
		foundFiles := make(map[string]bool)
		for _, file := range files {
			foundFiles[file] = true
		}

		if !foundFiles["test/file.txt"] {
			t.Error("Expected 'test/file.txt' in file list")
		}

		if !foundFiles["test/another.txt"] {
			t.Error("Expected 'test/another.txt' in file list")
		}
	})

	t.Run("Delete", func(t *testing.T) {
		err := manager.Delete(ctx, "test/file.txt")
		if err != nil {
			t.Errorf("Failed to delete file: %v", err)
		}

		// Verify file is deleted
		exists, err := manager.Exists(ctx, "test/file.txt")
		if err != nil {
			t.Errorf("Failed to check existence after delete: %v", err)
		}

		if exists {
			t.Error("Expected file to be deleted")
		}
	})

	t.Run("Close", func(t *testing.T) {
		err := manager.Close()
		if err != nil {
			t.Errorf("Failed to close manager: %v", err)
		}
	})
}

func TestStorageErrorHandling(t *testing.T) {
	manager := NewStorageManager()
	ctx := context.Background()

	t.Run("OperationsWithoutDefaultBackend", func(t *testing.T) {
		// Test operations without setting a default backend
		data := strings.NewReader("test")
		err := manager.Save(ctx, "test.txt", data)
		if err == nil {
			t.Error("Expected error when saving without default backend")
		}

		_, err = manager.Load(ctx, "test.txt")
		if err == nil {
			t.Error("Expected error when loading without default backend")
		}

		err = manager.Delete(ctx, "test.txt")
		if err == nil {
			t.Error("Expected error when deleting without default backend")
		}

		_, err = manager.Exists(ctx, "test.txt")
		if err == nil {
			t.Error("Expected error when checking existence without default backend")
		}

		_, err = manager.List(ctx, "")
		if err == nil {
			t.Error("Expected error when listing without default backend")
		}
	})

	t.Run("BackendErrors", func(t *testing.T) {
		failingBackend := newMockStorage("failing")
		failingBackend.failSave = true
		failingBackend.failLoad = true

		err := manager.Register("failing", failingBackend)
		if err != nil {
			t.Fatalf("Failed to register failing backend: %v", err)
		}

		err = manager.SetDefault("failing")
		if err != nil {
			t.Fatalf("Failed to set failing backend as default: %v", err)
		}

		// Test save error
		data := strings.NewReader("test")
		err = manager.Save(ctx, "test.txt", data)
		if err == nil {
			t.Error("Expected save error from failing backend")
		}

		// Test load error
		_, err = manager.Load(ctx, "test.txt")
		if err == nil {
			t.Error("Expected load error from failing backend")
		}
	})
}

func TestStorageConfig(t *testing.T) {
	t.Run("ConfigureBackend", func(t *testing.T) {
		manager := NewStorageManager()
		mockBackend := newMockStorage("config-test")

		err := manager.Register("config-test", mockBackend)
		if err != nil {
			t.Fatalf("Failed to register backend: %v", err)
		}

		// Test configuration (mock backend accepts any config)
		config := map[string]interface{}{"test": "value"}

		backend, err := manager.GetBackend("config-test")
		if err != nil {
			t.Fatalf("Failed to get backend: %v", err)
		}
		if backend == nil {
			t.Fatal("Failed to get backend")
		}

		err = backend.Init(config)
		if err != nil {
			t.Errorf("Failed to initialize backend: %v", err)
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	manager := NewStorageManager()
	mockBackend := newMockStorage("concurrent")

	err := manager.Register("concurrent", mockBackend)
	if err != nil {
		t.Fatalf("Failed to register backend: %v", err)
	}

	err = manager.SetDefault("concurrent")
	if err != nil {
		t.Fatalf("Failed to set default backend: %v", err)
	}

	ctx := context.Background()
	done := make(chan bool, 10)

	// Concurrent saves
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			data := strings.NewReader("concurrent data")
			path := strings.Replace("file{id}.txt", "{id}", string(rune(id+'0')), 1)
			err := manager.Save(ctx, path, data)
			if err != nil {
				t.Errorf("Concurrent save failed: %v", err)
			}
		}(i)
	}

	// Concurrent loads
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			path := strings.Replace("file{id}.txt", "{id}", string(rune(id+'0')), 1)
			// Try to load (might fail if save hasn't completed yet, but shouldn't crash)
			_, err := manager.Load(ctx, path)
			if err != nil {
				// This is acceptable in concurrent scenario
				t.Logf("Concurrent load failed (expected): %v", err)
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestStorageBackendInterface(t *testing.T) {
	// Test that our mock backend implements the interface correctly
	var backend StorageBackend = newMockStorage("interface-test")

	ctx := context.Background()

	// Test all interface methods
	config := map[string]interface{}{"type": "test"}
	err := backend.Init(config)
	if err != nil {
		t.Errorf("Init failed: %v", err)
	}

	data := strings.NewReader("interface test")
	err = backend.Save(ctx, "interface.txt", data)
	if err != nil {
		t.Errorf("Save failed: %v", err)
	}

	exists, err := backend.Exists(ctx, "interface.txt")
	if err != nil {
		t.Errorf("Exists failed: %v", err)
	}

	if !exists {
		t.Error("Expected file to exist")
	}

	reader, err := backend.Load(ctx, "interface.txt")
	if err != nil {
		t.Errorf("Load failed: %v", err)
	}
	defer func() { _ = reader.Close() }()

	files, err := backend.List(ctx, "")
	if err != nil {
		t.Errorf("List failed: %v", err)
	}

	if len(files) == 0 {
		t.Error("Expected at least one file in list")
	}

	err = backend.Delete(ctx, "interface.txt")
	if err != nil {
		t.Errorf("Delete failed: %v", err)
	}

	err = backend.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}
