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
