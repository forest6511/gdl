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

// Additional tests to improve coverage

func TestStorageManagerAdvanced(t *testing.T) {
	t.Run("Multiple backends", func(t *testing.T) {
		manager := NewStorageManager()

		// Register multiple backends
		backend1 := newMockStorage("backend1")
		backend2 := newMockStorage("backend2")

		err := manager.Register("backend1", backend1)
		if err != nil {
			t.Fatalf("Failed to register backend1: %v", err)
		}

		err = manager.Register("backend2", backend2)
		if err != nil {
			t.Fatalf("Failed to register backend2: %v", err)
		}

		// Test getting specific backends
		b1, err := manager.GetBackend("backend1")
		if err != nil {
			t.Errorf("Failed to get backend1: %v", err)
		}
		if b1 != backend1 {
			t.Error("Got wrong backend1 instance")
		}

		b2, err := manager.GetBackend("backend2")
		if err != nil {
			t.Errorf("Failed to get backend2: %v", err)
		}
		if b2 != backend2 {
			t.Error("Got wrong backend2 instance")
		}
	})

	t.Run("Default backend operations", func(t *testing.T) {
		manager := NewStorageManager()

		// Test getting default when none set
		_, err := manager.GetDefault()
		if err == nil {
			t.Error("Expected error when no default backend set")
		}

		// Register and set default
		backend := newMockStorage("default")
		err = manager.Register("default", backend)
		if err != nil {
			t.Fatalf("Failed to register backend: %v", err)
		}

		err = manager.SetDefault("default")
		if err != nil {
			t.Fatalf("Failed to set default: %v", err)
		}

		// Test all operations use default
		ctx := context.Background()

		// Save
		data := strings.NewReader("test data")
		err = manager.Save(ctx, "test.txt", data)
		if err != nil {
			t.Errorf("Save failed: %v", err)
		}

		// Check data was saved
		if _, exists := backend.files["test.txt"]; !exists {
			t.Error("Data was not saved to backend")
		}

		// Load
		reader, err := manager.Load(ctx, "test.txt")
		if err != nil {
			t.Errorf("Load failed: %v", err)
		}
		defer func() { _ = reader.Close() }()

		loadedData, err := io.ReadAll(reader)
		if err != nil {
			t.Errorf("Failed to read loaded data: %v", err)
		}

		if string(loadedData) != "test data" {
			t.Errorf("Expected 'test data', got '%s'", string(loadedData))
		}

		// Exists
		exists, err := manager.Exists(ctx, "test.txt")
		if err != nil {
			t.Errorf("Exists failed: %v", err)
		}
		if !exists {
			t.Error("File should exist")
		}

		// List
		files, err := manager.List(ctx, "")
		if err != nil {
			t.Errorf("List failed: %v", err)
		}
		if len(files) != 1 || files[0] != "test.txt" {
			t.Errorf("Expected ['test.txt'], got %v", files)
		}

		// Delete
		err = manager.Delete(ctx, "test.txt")
		if err != nil {
			t.Errorf("Delete failed: %v", err)
		}

		// Verify deleted
		if _, exists := backend.files["test.txt"]; exists {
			t.Error("File should have been deleted")
		}
	})
}

func TestMockStorageBackendEdgeCases(t *testing.T) {
	backend := newMockStorage("test")
	ctx := context.Background()

	t.Run("Load non-existent file", func(t *testing.T) {
		_, err := backend.Load(ctx, "non-existent.txt")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})

	t.Run("Delete non-existent file", func(t *testing.T) {
		// Delete should not error for non-existent files in our mock
		err := backend.Delete(ctx, "non-existent.txt")
		if err != nil {
			t.Errorf("Delete non-existent should not error, got: %v", err)
		}
	})

	t.Run("Exists for non-existent file", func(t *testing.T) {
		exists, err := backend.Exists(ctx, "non-existent.txt")
		if err != nil {
			t.Errorf("Exists check should not error, got: %v", err)
		}
		if exists {
			t.Error("Non-existent file should not exist")
		}
	})

	t.Run("List with no files", func(t *testing.T) {
		files, err := backend.List(ctx, "prefix")
		if err != nil {
			t.Errorf("List should not error, got: %v", err)
		}
		if len(files) != 0 {
			t.Errorf("Expected empty list, got %v", files)
		}
	})

	t.Run("List with prefix matching", func(t *testing.T) {
		// Add some files
		_ = backend.Save(ctx, "prefix_file1.txt", strings.NewReader("data1"))
		_ = backend.Save(ctx, "prefix_file2.txt", strings.NewReader("data2"))
		_ = backend.Save(ctx, "other_file.txt", strings.NewReader("data3"))

		files, err := backend.List(ctx, "prefix_")
		if err != nil {
			t.Errorf("List should not error, got: %v", err)
		}

		if len(files) != 2 {
			t.Errorf("Expected 2 files with prefix, got %d", len(files))
		}

		// Check that the right files are returned
		expectedFiles := map[string]bool{
			"prefix_file1.txt": false,
			"prefix_file2.txt": false,
		}

		for _, file := range files {
			if _, ok := expectedFiles[file]; ok {
				expectedFiles[file] = true
			}
		}

		for file, found := range expectedFiles {
			if !found {
				t.Errorf("Expected file %s not found in list", file)
			}
		}
	})
}

func TestStorageManagerErrorHandling(t *testing.T) {
	t.Run("Backend save error", func(t *testing.T) {
		manager := NewStorageManager()
		backend := newMockStorage("failing")
		backend.failSave = true

		err := manager.Register("failing", backend)
		if err != nil {
			t.Fatalf("Failed to register backend: %v", err)
		}

		err = manager.SetDefault("failing")
		if err != nil {
			t.Fatalf("Failed to set default: %v", err)
		}

		ctx := context.Background()
		data := strings.NewReader("test")
		err = manager.Save(ctx, "test.txt", data)
		if err == nil {
			t.Error("Expected save error from failing backend")
		}
	})

	t.Run("Backend load error", func(t *testing.T) {
		manager := NewStorageManager()
		backend := newMockStorage("failing")
		backend.failLoad = true

		err := manager.Register("failing", backend)
		if err != nil {
			t.Fatalf("Failed to register backend: %v", err)
		}

		err = manager.SetDefault("failing")
		if err != nil {
			t.Fatalf("Failed to set default: %v", err)
		}

		ctx := context.Background()
		_, err = manager.Load(ctx, "test.txt")
		if err == nil {
			t.Error("Expected load error from failing backend")
		}
	})

	t.Run("Close with multiple backends", func(t *testing.T) {
		manager := NewStorageManager()

		// Register multiple backends
		backend1 := newMockStorage("backend1")
		backend2 := newMockStorage("backend2")

		err := manager.Register("backend1", backend1)
		if err != nil {
			t.Fatalf("Failed to register backend1: %v", err)
		}

		err = manager.Register("backend2", backend2)
		if err != nil {
			t.Fatalf("Failed to register backend2: %v", err)
		}

		// Close should close all backends
		err = manager.Close()
		if err != nil {
			t.Errorf("Close should not error, got: %v", err)
		}
	})
}

func TestStorageInitialization(t *testing.T) {
	t.Run("Backend initialization", func(t *testing.T) {
		backend := newMockStorage("test")

		config := map[string]interface{}{
			"setting1": "value1",
			"setting2": 123,
		}

		err := backend.Init(config)
		if err != nil {
			t.Errorf("Init should not error, got: %v", err)
		}
	})

	t.Run("Backend close", func(t *testing.T) {
		backend := newMockStorage("test")

		err := backend.Close()
		if err != nil {
			t.Errorf("Close should not error, got: %v", err)
		}
	})
}

func TestStorageErrors(t *testing.T) {
	// Test that all error constants are defined and non-nil
	errors := []error{
		ErrBackendNotFound,
		ErrNoDefaultBackend,
		ErrKeyNotFound,
		ErrInvalidConfig,
		ErrBackendNotReady,
		ErrUnsupportedOp,
	}

	for i, err := range errors {
		if err == nil {
			t.Errorf("Error %d should not be nil", i)
		}
		if err.Error() == "" {
			t.Errorf("Error %d should have a message", i)
		}
	}

	// Test specific error messages
	if ErrBackendNotFound.Error() != "storage backend not found" {
		t.Errorf("ErrBackendNotFound has wrong message: %s", ErrBackendNotFound.Error())
	}

	if ErrNoDefaultBackend.Error() != "no default storage backend configured" {
		t.Errorf("ErrNoDefaultBackend has wrong message: %s", ErrNoDefaultBackend.Error())
	}

	if ErrKeyNotFound.Error() != "key not found in storage" {
		t.Errorf("ErrKeyNotFound has wrong message: %s", ErrKeyNotFound.Error())
	}
}
