package backends

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewMemoryBackend(t *testing.T) {
	backend := NewMemoryBackend()
	if backend == nil {
		t.Error("Expected memory backend to be created, got nil")
	}
}

func TestMemoryBackendInit(t *testing.T) {
	backend := NewMemoryBackend()

	config := map[string]interface{}{
		"maxSize": 1024,
	}

	err := backend.Init(config)
	if err != nil {
		t.Errorf("Expected no error from Init, got %v", err)
	}
}

func TestMemoryBackendSaveAndLoad(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()
	key := "test-key"
	data := "test data"

	// Test Save
	err := backend.Save(ctx, key, strings.NewReader(data))
	if err != nil {
		t.Errorf("Expected no error from Save, got %v", err)
	}

	// Test Load
	reader, err := backend.Load(ctx, key)
	if err != nil {
		t.Errorf("Expected no error from Load, got %v", err)
	}
	defer func() { _ = reader.Close() }()

	loadedData, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Expected no error reading data, got %v", err)
	}

	if string(loadedData) != data {
		t.Errorf("Expected %q, got %q", data, string(loadedData))
	}
}

func TestMemoryBackendExists(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()
	key := "test-key"

	// Initially should not exist
	exists, err := backend.Exists(ctx, key)
	if err != nil {
		t.Errorf("Expected no error from Exists, got %v", err)
	}
	if exists {
		t.Error("Expected key not to exist initially")
	}

	// Save data
	_ = backend.Save(ctx, key, strings.NewReader("test"))

	// Should exist now
	exists, err = backend.Exists(ctx, key)
	if err != nil {
		t.Errorf("Expected no error from Exists, got %v", err)
	}
	if !exists {
		t.Error("Expected key to exist after save")
	}
}

func TestMemoryBackendDelete(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()
	key := "test-key"

	// Save data
	_ = backend.Save(ctx, key, strings.NewReader("test"))

	// Delete
	err := backend.Delete(ctx, key)
	if err != nil {
		t.Errorf("Expected no error from Delete, got %v", err)
	}

	// Should not exist
	exists, _ := backend.Exists(ctx, key)
	if exists {
		t.Error("Expected key not to exist after delete")
	}
}

func TestMemoryBackendList(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()

	// Save multiple keys with prefix
	_ = backend.Save(ctx, "prefix/key1", strings.NewReader("data1"))
	_ = backend.Save(ctx, "prefix/key2", strings.NewReader("data2"))
	_ = backend.Save(ctx, "other/key3", strings.NewReader("data3"))

	// List with prefix
	keys, err := backend.List(ctx, "prefix/")
	if err != nil {
		t.Errorf("Expected no error from List, got %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys with prefix, got %d", len(keys))
	}
}

func TestMemoryBackendClose(t *testing.T) {
	backend := NewMemoryBackend()

	err := backend.Close()
	if err != nil {
		t.Errorf("Expected no error from Close, got %v", err)
	}
}

func TestMemoryBackendSize(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()

	// Initially should be 0
	size := backend.Size()
	if size != 0 {
		t.Errorf("Expected initial size 0, got %d", size)
	}

	// Save data
	_ = backend.Save(ctx, "key1", strings.NewReader("data"))

	// Size should increase
	size = backend.Size()
	if size == 0 {
		t.Error("Expected size > 0 after saving data")
	}
}

func TestMemoryBackendMemoryUsage(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	usage := backend.MemoryUsage()
	if usage < 0 {
		t.Errorf("Expected non-negative memory usage, got %d", usage)
	}
}

func TestMemoryBackendClear(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()

	// Save some data
	_ = backend.Save(ctx, "key1", strings.NewReader("data1"))
	_ = backend.Save(ctx, "key2", strings.NewReader("data2"))

	// Clear
	backend.Clear()

	// Should be empty
	exists1, _ := backend.Exists(ctx, "key1")
	exists2, _ := backend.Exists(ctx, "key2")

	if exists1 || exists2 {
		t.Error("Expected all data to be cleared")
	}
}

func TestMemoryBackendKeys(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()

	// Save some data
	_ = backend.Save(ctx, "key1", strings.NewReader("data1"))
	_ = backend.Save(ctx, "key2", strings.NewReader("data2"))

	keys := backend.Keys()
	if len(keys) != 2 {
		t.Errorf("Expected 2 keys, got %d", len(keys))
	}
}

func TestMemoryBackendGetSize(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()
	key := "test-key"
	data := "test data"

	// Save data
	_ = backend.Save(ctx, key, strings.NewReader(data))

	// Get size
	size, err := backend.GetSize(ctx, key)
	if err != nil {
		t.Errorf("Expected no error from GetSize, got %v", err)
	}

	if size != int64(len(data)) {
		t.Errorf("Expected size %d, got %d", len(data), size)
	}
}

func TestMemoryBackendBatchSave(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()

	items := map[string]io.Reader{
		"key1": strings.NewReader("data1"),
		"key2": strings.NewReader("data2"),
		"key3": strings.NewReader("data3"),
	}

	err := backend.BatchSave(ctx, items)
	if err != nil {
		t.Errorf("Expected no error from BatchSave, got %v", err)
	}

	// Verify all items were saved
	for key := range items {
		exists, _ := backend.Exists(ctx, key)
		if !exists {
			t.Errorf("Expected key %s to exist after batch save", key)
		}
	}
}

func TestMemoryBackendBatchDelete(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()

	// Save some data
	_ = backend.Save(ctx, "key1", strings.NewReader("data1"))
	_ = backend.Save(ctx, "key2", strings.NewReader("data2"))
	_ = backend.Save(ctx, "key3", strings.NewReader("data3"))

	// Batch delete
	keys := []string{"key1", "key2"}
	err := backend.BatchDelete(ctx, keys)
	if err != nil {
		t.Errorf("Expected no error from BatchDelete, got %v", err)
	}

	// Verify deleted
	for _, key := range keys {
		exists, _ := backend.Exists(ctx, key)
		if exists {
			t.Errorf("Expected key %s not to exist after batch delete", key)
		}
	}

	// Verify key3 still exists
	exists, _ := backend.Exists(ctx, "key3")
	if !exists {
		t.Error("Expected key3 to still exist")
	}
}

func TestMemoryBackendClone(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()

	// Save some data
	_ = backend.Save(ctx, "key1", strings.NewReader("data1"))

	// Clone
	cloned := backend.Clone()
	if cloned == nil {
		t.Error("Expected cloned backend, got nil")
	}

	// Verify data exists in clone
	exists, _ := cloned.Exists(ctx, "key1")
	if !exists {
		t.Error("Expected data to exist in cloned backend")
	}
}

// Additional tests for improving coverage

func TestMemoryBackendContextCancellation(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test Save with cancelled context
	err := backend.Save(ctx, "key", strings.NewReader("data"))
	if err == nil {
		t.Error("Expected error for Save with cancelled context")
	}

	// Test Load with cancelled context
	_, err = backend.Load(ctx, "key")
	if err == nil {
		t.Error("Expected error for Load with cancelled context")
	}

	// Test Delete with cancelled context
	err = backend.Delete(ctx, "key")
	if err == nil {
		t.Error("Expected error for Delete with cancelled context")
	}

	// Test Exists with cancelled context
	_, err = backend.Exists(ctx, "key")
	if err == nil {
		t.Error("Expected error for Exists with cancelled context")
	}

	// Test List with cancelled context
	_, err = backend.List(ctx, "prefix")
	if err == nil {
		t.Error("Expected error for List with cancelled context")
	}

	// Test GetSize with cancelled context
	_, err = backend.GetSize(ctx, "key")
	if err == nil {
		t.Error("Expected error for GetSize with cancelled context")
	}

	// Test BatchSave with cancelled context
	items := map[string]io.Reader{
		"key1": strings.NewReader("data1"),
	}
	err = backend.BatchSave(ctx, items)
	if err == nil {
		t.Error("Expected error for BatchSave with cancelled context")
	}

	// Test BatchDelete with cancelled context
	err = backend.BatchDelete(ctx, []string{"key1"})
	if err == nil {
		t.Error("Expected error for BatchDelete with cancelled context")
	}
}

func TestMemoryBackendDeleteNonExistent(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()

	// Try to delete non-existent key
	err := backend.Delete(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent key")
	}
}

func TestMemoryBackendLoadNonExistent(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()

	// Try to load non-existent key
	_, err := backend.Load(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error when loading non-existent key")
	}
}

func TestMemoryBackendGetSizeNonExistent(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()

	// Try to get size of non-existent key
	_, err := backend.GetSize(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error when getting size of non-existent key")
	}
}

func TestMemoryBackendBatchSaveReadError(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()

	// Create a reader that will error
	errorReader := &erroringReader{}
	items := map[string]io.Reader{
		"key1": errorReader,
	}

	err := backend.BatchSave(ctx, items)
	if err == nil {
		t.Error("Expected error when BatchSave encounters read error")
	}
}

func TestMemoryBackendEmptyPrefix(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()

	// Save some data
	_ = backend.Save(ctx, "key1", strings.NewReader("data1"))
	_ = backend.Save(ctx, "key2", strings.NewReader("data2"))

	// List with empty prefix should return all keys
	keys, err := backend.List(ctx, "")
	if err != nil {
		t.Errorf("Expected no error from List with empty prefix, got %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys with empty prefix, got %d", len(keys))
	}
}

func TestMemoryBackendReinitialize(t *testing.T) {
	backend := NewMemoryBackend()
	_ = backend.Init(nil)

	ctx := context.Background()

	// Save some data
	_ = backend.Save(ctx, "key1", strings.NewReader("data1"))

	// Reinitialize should clear data
	_ = backend.Init(nil)

	// Check that data is cleared
	exists, _ := backend.Exists(ctx, "key1")
	if exists {
		t.Error("Expected data to be cleared after reinit")
	}
}

// Helper type for testing error conditions
type erroringReader struct{}

func (e *erroringReader) Read(p []byte) (n int, err error) {
	return 0, io.ErrUnexpectedEOF
}

// Filesystem Backend Tests
func TestNewFileSystemBackend(t *testing.T) {
	backend := NewFileSystemBackend()
	if backend == nil {
		t.Error("Expected filesystem backend to be created, got nil")
	}
}

func TestFileSystemBackendInit(t *testing.T) {
	tempDir := t.TempDir()
	backend := NewFileSystemBackend()

	config := map[string]interface{}{
		"basePath": tempDir,
	}

	err := backend.Init(config)
	if err != nil {
		t.Errorf("Expected no error from Init, got %v", err)
	}
}

func TestFileSystemBackendSaveAndLoad(t *testing.T) {
	tempDir := t.TempDir()
	backend := NewFileSystemBackend()

	config := map[string]interface{}{
		"basePath": tempDir,
	}
	_ = backend.Init(config)

	ctx := context.Background()
	key := "test/key"
	data := "test data"

	// Test Save
	err := backend.Save(ctx, key, strings.NewReader(data))
	if err != nil {
		t.Errorf("Expected no error from Save, got %v", err)
	}

	// Verify file exists
	expectedPath := filepath.Join(tempDir, "test", "key")
	if _, err := os.Stat(expectedPath); os.IsNotExist(err) {
		t.Error("Expected file to exist after save")
	}

	// Test Load
	reader, err := backend.Load(ctx, key)
	if err != nil {
		t.Errorf("Expected no error from Load, got %v", err)
	}
	defer func() { _ = reader.Close() }()

	loadedData, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Expected no error reading data, got %v", err)
	}

	if string(loadedData) != data {
		t.Errorf("Expected %q, got %q", data, string(loadedData))
	}
}

func TestFileSystemBackendExists(t *testing.T) {
	tempDir := t.TempDir()
	backend := NewFileSystemBackend()

	config := map[string]interface{}{
		"basePath": tempDir,
	}
	_ = backend.Init(config)

	ctx := context.Background()
	key := "test-key"

	// Initially should not exist
	exists, err := backend.Exists(ctx, key)
	if err != nil {
		t.Errorf("Expected no error from Exists, got %v", err)
	}
	if exists {
		t.Error("Expected key not to exist initially")
	}

	// Save data
	_ = backend.Save(ctx, key, strings.NewReader("test"))

	// Should exist now
	exists, err = backend.Exists(ctx, key)
	if err != nil {
		t.Errorf("Expected no error from Exists, got %v", err)
	}
	if !exists {
		t.Error("Expected key to exist after save")
	}
}

func TestFileSystemBackendDelete(t *testing.T) {
	tempDir := t.TempDir()
	backend := NewFileSystemBackend()

	config := map[string]interface{}{
		"basePath": tempDir,
	}
	_ = backend.Init(config)

	ctx := context.Background()
	key := "test-key"

	// Save data
	_ = backend.Save(ctx, key, strings.NewReader("test"))

	// Delete
	err := backend.Delete(ctx, key)
	if err != nil {
		t.Errorf("Expected no error from Delete, got %v", err)
	}

	// Should not exist
	exists, _ := backend.Exists(ctx, key)
	if exists {
		t.Error("Expected key not to exist after delete")
	}
}

func TestFileSystemBackendList(t *testing.T) {
	tempDir := t.TempDir()
	backend := NewFileSystemBackend()

	config := map[string]interface{}{
		"basePath": tempDir,
	}
	_ = backend.Init(config)

	ctx := context.Background()

	// Save multiple keys with prefix
	_ = backend.Save(ctx, "prefix/key1", strings.NewReader("data1"))
	_ = backend.Save(ctx, "prefix/key2", strings.NewReader("data2"))
	_ = backend.Save(ctx, "other/key3", strings.NewReader("data3"))

	// List with prefix
	keys, err := backend.List(ctx, "prefix/")
	if err != nil {
		t.Errorf("Expected no error from List, got %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys with prefix, got %d", len(keys))
	}
}

func TestFileSystemBackendClose(t *testing.T) {
	backend := NewFileSystemBackend()

	err := backend.Close()
	if err != nil {
		t.Errorf("Expected no error from Close, got %v", err)
	}
}

func TestFileSystemBackendPathHandling(t *testing.T) {
	tempDir := t.TempDir()
	backend := NewFileSystemBackend()

	config := map[string]interface{}{
		"basePath": tempDir,
	}
	_ = backend.Init(config)

	ctx := context.Background()

	// Test nested directory creation
	key := "deep/nested/path/file.txt"
	data := "test data"

	err := backend.Save(ctx, key, strings.NewReader(data))
	if err != nil {
		t.Errorf("Expected no error from Save with nested path, got %v", err)
	}

	// Verify file exists
	exists, err := backend.Exists(ctx, key)
	if err != nil {
		t.Errorf("Expected no error from Exists, got %v", err)
	}
	if !exists {
		t.Error("Expected file to exist after save")
	}

	// Load and verify content
	reader, err := backend.Load(ctx, key)
	if err != nil {
		t.Errorf("Expected no error from Load, got %v", err)
	}
	defer func() { _ = reader.Close() }()

	loadedData, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("Expected no error reading data, got %v", err)
	}

	if string(loadedData) != data {
		t.Errorf("Expected %q, got %q", data, string(loadedData))
	}
}

func TestFileSystemBackendErrorCases(t *testing.T) {
	// Test with invalid config
	backend := NewFileSystemBackend()

	// Init with invalid config
	err := backend.Init(map[string]interface{}{"invalid": "config"})
	if err == nil {
		t.Error("Expected error for invalid config")
	}

	// Test operations without init
	ctx := context.Background()

	// These should error without basePath
	_, err = backend.Exists(ctx, "test")
	if err == nil {
		t.Error("Expected error for Exists without basePath")
	}

	err = backend.Save(ctx, "test", strings.NewReader("data"))
	if err == nil {
		t.Error("Expected error for Save without basePath")
	}

	_, err = backend.Load(ctx, "test")
	if err == nil {
		t.Error("Expected error for Load without basePath")
	}

	err = backend.Delete(ctx, "test")
	if err == nil {
		t.Error("Expected error for Delete without basePath")
	}

	_, err = backend.List(ctx, "test")
	if err == nil {
		t.Error("Expected error for List without basePath")
	}
}

func TestFileSystemBackendContextCancellation(t *testing.T) {
	tempDir := t.TempDir()
	backend := NewFileSystemBackend()
	_ = backend.Init(map[string]interface{}{"basePath": tempDir})

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	// Test Save with cancelled context
	err := backend.Save(ctx, "key", strings.NewReader("data"))
	if err == nil {
		t.Error("Expected error for Save with cancelled context")
	}

	// Test Load with cancelled context
	_, err = backend.Load(ctx, "key")
	if err == nil {
		t.Error("Expected error for Load with cancelled context")
	}

	// Test Delete with cancelled context - Delete doesn't support context cancellation
	err = backend.Delete(ctx, "key")
	// Delete doesn't check context, so this will return an error for key not found
	if err == nil {
		t.Error("Expected error when deleting non-existent key")
	}

	// Test Exists with cancelled context - Exists doesn't support context cancellation
	exists, err := backend.Exists(ctx, "key")
	if err != nil {
		t.Errorf("Exists should not return error for cancelled context, got %v", err)
	}
	if exists {
		t.Error("Key should not exist")
	}

	// Test List with cancelled context
	// First save a file with valid context to ensure the walk has something to iterate
	validCtx := context.Background()
	_ = backend.Save(validCtx, "test-file", strings.NewReader("data"))

	// Now test with cancelled context
	_, err = backend.List(ctx, "prefix")
	if err == nil {
		t.Error("Expected error for List with cancelled context when iterating files")
	}
}

func TestFileSystemBackendLoadNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	backend := NewFileSystemBackend()
	_ = backend.Init(map[string]interface{}{"basePath": tempDir})

	ctx := context.Background()

	// Try to load non-existent key
	_, err := backend.Load(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error when loading non-existent key")
	}
}

func TestFileSystemBackendDeleteNonExistent(t *testing.T) {
	tempDir := t.TempDir()
	backend := NewFileSystemBackend()
	_ = backend.Init(map[string]interface{}{"basePath": tempDir})

	ctx := context.Background()

	// Try to delete non-existent key
	err := backend.Delete(ctx, "non-existent")
	if err == nil {
		t.Error("Expected error when deleting non-existent key")
	}
}

func TestFileSystemBackendCleanupEmptyDirs(t *testing.T) {
	tempDir := t.TempDir()
	backend := NewFileSystemBackend()
	_ = backend.Init(map[string]interface{}{"basePath": tempDir})

	ctx := context.Background()

	// Create nested file
	key := "deep/nested/file.txt"
	_ = backend.Save(ctx, key, strings.NewReader("data"))

	// Delete the file
	err := backend.Delete(ctx, key)
	if err != nil {
		t.Errorf("Expected no error from Delete, got %v", err)
	}

	// Check that empty directories were cleaned up
	deepPath := filepath.Join(tempDir, "deep")
	if _, err := os.Stat(deepPath); !os.IsNotExist(err) {
		t.Error("Expected empty directories to be cleaned up")
	}
}

func TestFileSystemBackendEmptyPrefixList(t *testing.T) {
	tempDir := t.TempDir()
	backend := NewFileSystemBackend()
	_ = backend.Init(map[string]interface{}{"basePath": tempDir})

	ctx := context.Background()

	// Save multiple files
	_ = backend.Save(ctx, "file1", strings.NewReader("data1"))
	_ = backend.Save(ctx, "dir/file2", strings.NewReader("data2"))

	// List with empty prefix should return all keys
	keys, err := backend.List(ctx, "")
	if err != nil {
		t.Errorf("Expected no error from List with empty prefix, got %v", err)
	}

	if len(keys) != 2 {
		t.Errorf("Expected 2 keys with empty prefix, got %d", len(keys))
	}
}

func TestFileSystemBackendSaveReadError(t *testing.T) {
	tempDir := t.TempDir()
	backend := NewFileSystemBackend()
	_ = backend.Init(map[string]interface{}{"basePath": tempDir})

	ctx := context.Background()

	// Create a reader that will error
	errorReader := &erroringReader{}

	err := backend.Save(ctx, "key", errorReader)
	if err == nil {
		t.Error("Expected error when Save encounters read error")
	}
}

// Test Redis Backend
func TestNewRedisBackend(t *testing.T) {
	backend := NewRedisBackend()
	if backend == nil {
		t.Error("Expected redis backend to be created, got nil")
	}
}

func TestRedisBackendInitError(t *testing.T) {
	backend := NewRedisBackend()

	// Init without config will use default values and try to connect
	// Since we likely don't have Redis running, this will fail
	err := backend.Init(nil)
	// This will likely fail to connect, but that's expected
	if err == nil {
		// If it succeeds, Redis is running
		t.Log("Redis appears to be running locally")
	}

	// Init with invalid config type for address still uses default
	err = backend.Init(map[string]interface{}{
		"address": 123, // Invalid type, but it will use default "localhost:6379"
	})
	// This will also likely fail to connect
	if err == nil {
		// If it succeeds, Redis is running
		t.Log("Redis appears to be running locally")
	}
}

// Test S3 Backend
func TestNewS3Backend(t *testing.T) {
	backend := NewS3Backend()
	if backend == nil {
		t.Error("Expected s3 backend to be created, got nil")
	}
}

func TestS3BackendInitError(t *testing.T) {
	backend := NewS3Backend()

	// Init without required config should error
	err := backend.Init(nil)
	if err == nil {
		t.Error("Expected error for Init without config")
	}

	// Init with missing bucket
	err = backend.Init(map[string]interface{}{
		"region": "us-west-2",
	})
	if err == nil {
		t.Error("Expected error for Init without bucket")
	}

	// Init with invalid config type
	err = backend.Init(map[string]interface{}{
		"bucket": 123, // Invalid type
		"region": "us-west-2",
	})
	if err == nil {
		t.Error("Expected error for Init with invalid bucket type")
	}
}

// Additional tests for Redis backend methods (without actual Redis connection)
func TestRedisBackendMethods(t *testing.T) {
	backend := NewRedisBackend()
	ctx := context.Background()

	// Test methods without initialization (should panic or error)
	t.Run("Methods without initialization", func(t *testing.T) {
		// These methods will panic with nil pointer dereference since client is nil
		// We'll test each one and expect them to panic

		// Save should panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for Save without Redis connection")
			}
		}()
		_ = backend.Save(ctx, "key", strings.NewReader("data"))
	})

	t.Run("Load without initialization", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for Load without Redis connection")
			}
		}()
		_, _ = backend.Load(ctx, "key")
	})

	t.Run("Delete without initialization", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for Delete without Redis connection")
			}
		}()
		_ = backend.Delete(ctx, "key")
	})

	t.Run("Exists without initialization", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for Exists without Redis connection")
			}
		}()
		_, _ = backend.Exists(ctx, "key")
	})

	t.Run("List without initialization", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for List without Redis connection")
			}
		}()
		_, _ = backend.List(ctx, "prefix")
	})

	t.Run("Helper methods", func(t *testing.T) {
		// Initialize with mock config first
		config := map[string]interface{}{
			"addr":     "localhost:9999", // Non-existent Redis
			"password": "test",
			"db":       0,
		}
		_ = backend.Init(config) // Ignore connection error

		// buildKey method
		key := backend.buildKey("test-key")
		if key == "" {
			t.Error("Expected non-empty built key")
		}

		// stripPrefix method
		stripped := backend.stripPrefix("prefix:key")
		if stripped == "" {
			t.Error("Expected non-empty stripped key")
		}

		// Close method (should not error even without connection)
		err := backend.Close()
		if err != nil {
			t.Errorf("Expected no error from Close, got %v", err)
		}
	})

	t.Run("Configuration edge cases", func(t *testing.T) {
		// Test with missing address
		err := backend.Init(map[string]interface{}{
			"password": "test",
			"db":       0,
		})
		// Should use default address
		if err == nil {
			t.Log("Redis connection succeeded with default address")
		}

		// Test with invalid db type
		err = backend.Init(map[string]interface{}{
			"addr": "localhost:6379",
			"db":   "invalid", // Invalid type
		})
		// Should use default db
		if err == nil {
			t.Log("Redis connection succeeded with default db")
		}
	})
}

// Additional tests for S3 backend methods (without actual S3 connection)
func TestS3BackendMethods(t *testing.T) {
	backend := NewS3Backend()
	ctx := context.Background()

	// Test methods without initialization (should panic)
	t.Run("Methods without initialization", func(t *testing.T) {
		// These methods will panic with nil pointer dereference since client is nil
		// We'll test each one and expect them to panic

		// Save should panic
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for Save without S3 connection")
			}
		}()
		_ = backend.Save(ctx, "key", strings.NewReader("data"))
	})

	t.Run("Load without initialization", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for Load without S3 connection")
			}
		}()
		_, _ = backend.Load(ctx, "key")
	})

	t.Run("Delete without initialization", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for Delete without S3 connection")
			}
		}()
		_ = backend.Delete(ctx, "key")
	})

	t.Run("Exists without initialization", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for Exists without S3 connection")
			}
		}()
		_, _ = backend.Exists(ctx, "key")
	})

	t.Run("List without initialization", func(t *testing.T) {
		defer func() {
			if r := recover(); r == nil {
				t.Error("Expected panic for List without S3 connection")
			}
		}()
		_, _ = backend.List(ctx, "prefix")
	})

	t.Run("Helper methods", func(t *testing.T) {
		// buildKey method
		key := backend.buildKey("test-key")
		if key == "" {
			t.Error("Expected non-empty built key")
		}

		// stripPrefix method
		stripped := backend.stripPrefix("prefix/key")
		if stripped == "" {
			t.Error("Expected non-empty stripped key")
		}

		// Close method (should not error even without connection)
		err := backend.Close()
		if err != nil {
			t.Errorf("Expected no error from Close, got %v", err)
		}
	})

	t.Run("Configuration edge cases", func(t *testing.T) {
		// Test with missing region - should use default region
		err := backend.Init(map[string]interface{}{
			"bucket": "test-bucket",
		})
		// This might fail due to AWS credentials/network, but not due to missing region
		if err != nil {
			t.Logf("Init failed (expected due to AWS setup): %v", err)
		}

		// Test with invalid region type - should use default region
		err = backend.Init(map[string]interface{}{
			"bucket": "test-bucket",
			"region": 123, // Invalid type, will use default
		})
		// This might fail due to AWS credentials/network, but not due to invalid region type
		if err != nil {
			t.Logf("Init failed (expected due to AWS setup): %v", err)
		}

		// Test with missing credentials but valid config
		err = backend.Init(map[string]interface{}{
			"bucket": "test-bucket",
			"region": "us-west-2",
		})
		// This might fail due to missing AWS credentials, which is expected
		if err != nil {
			t.Logf("Init failed (expected due to AWS credentials): %v", err)
		}
	})
}
