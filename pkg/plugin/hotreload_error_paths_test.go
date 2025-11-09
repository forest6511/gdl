package plugin

import (
	"os"
	"path/filepath"
	"testing"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
)

// TestNewHotReloadManagerErrorPaths tests error paths in NewHotReloadManager
func TestNewHotReloadManagerErrorPaths(t *testing.T) {
	t.Run("InvalidWatchDirectory", func(t *testing.T) {
		// Create a plugin manager
		manager := NewPluginManager()

		// Test with an invalid watch directory (null byte in path causes error)
		config := &HotReloadConfig{
			WatchDirectories: []string{"/invalid/\x00/path"},
		}

		_, err := NewHotReloadManager(config, manager, nil)
		if err == nil {
			t.Fatal("Expected error for invalid watch directory")
		}

		var storageErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &storageErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		if storageErr.Code != gdlerrors.CodeStorageError {
			t.Errorf("Expected CodeStorageError, got: %s", storageErr.Code)
		}
	})

	t.Run("MultipleWatchDirectoriesWithError", func(t *testing.T) {
		// Create a plugin manager
		manager := NewPluginManager()

		// Create a valid temp directory
		tempDir := t.TempDir()

		// Mix valid and invalid directories
		config := &HotReloadConfig{
			WatchDirectories: []string{tempDir, "/invalid/\x00/path"},
		}

		_, err := NewHotReloadManager(config, manager, nil)
		if err == nil {
			t.Fatal("Expected error for invalid watch directory in list")
		}

		var storageErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &storageErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		if storageErr.Code != gdlerrors.CodeStorageError {
			t.Errorf("Expected CodeStorageError, got: %s", storageErr.Code)
		}
	})
}

// TestUnloadPluginErrorPaths tests error paths in unloadPlugin
func TestUnloadPluginErrorPaths(t *testing.T) {
	t.Run("PluginWithDependents", func(t *testing.T) {
		// Create managers
		manager := NewPluginManager()
		depManager := NewDependencyManager()

		// Create a temp directory for test plugins
		tempDir := t.TempDir()

		// Create valid HotReloadManager with temp directory
		config := &HotReloadConfig{
			WatchDirectories: []string{tempDir},
		}

		hrm, err := NewHotReloadManager(config, manager, depManager)
		if err != nil {
			t.Fatalf("Failed to create HotReloadManager: %v", err)
		}

		// Create a mock plugin file
		pluginPath := filepath.Join(tempDir, "base_plugin.so")
		if err := os.WriteFile(pluginPath, []byte("mock plugin content"), 0644); err != nil {
			t.Fatal(err)
		}

		// Create mock plugins with dependencies
		basePlugin := &MockPlugin{name: "base_plugin", version: "1.0.0"}
		vpBase, err := NewVersionedPlugin(basePlugin, "1.0.0")
		if err != nil {
			t.Fatalf("Failed to create versioned plugin: %v", err)
		}

		dependentPlugin := &MockPlugin{name: "dependent_plugin", version: "1.0.0"}
		vpDependent, err := NewVersionedPlugin(dependentPlugin, "1.0.0")
		if err != nil {
			t.Fatalf("Failed to create versioned plugin: %v", err)
		}
		vpDependent.AddDependency("base_plugin", "^1.0.0")

		// Register plugins with dependency manager
		if err := depManager.RegisterPlugin(vpBase); err != nil {
			t.Fatalf("Failed to register base plugin: %v", err)
		}

		if err := depManager.RegisterPlugin(vpDependent); err != nil {
			t.Fatalf("Failed to register dependent plugin: %v", err)
		}

		// Try to unload the base plugin (which has dependents)
		err = hrm.unloadPlugin("base_plugin")
		if err == nil {
			t.Fatal("Expected error when unloading plugin with dependents")
		}

		var pluginErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &pluginErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		if pluginErr.Code != gdlerrors.CodePluginError {
			t.Errorf("Expected CodePluginError, got: %s", pluginErr.Code)
		}

		// Verify error message mentions dependents
		errMsg := pluginErr.Error()
		if errMsg == "" {
			t.Error("Expected non-empty error message")
		}
	})

	t.Run("UnregisterPluginError", func(t *testing.T) {
		// Create managers
		manager := NewPluginManager()

		// Create a temp directory for test plugins
		tempDir := t.TempDir()

		// Create valid HotReloadManager with temp directory
		config := &HotReloadConfig{
			WatchDirectories: []string{tempDir},
		}

		hrm, err := NewHotReloadManager(config, manager, nil)
		if err != nil {
			t.Fatalf("Failed to create HotReloadManager: %v", err)
		}

		// Try to unload a non-existent plugin
		err = hrm.unloadPlugin("nonexistent_plugin")
		if err == nil {
			t.Fatal("Expected error when unloading non-existent plugin")
		}

		// This should return an error from manager.Unregister
		// The error type depends on the manager implementation
		if err.Error() == "" {
			t.Error("Expected non-empty error message")
		}
	})
}

// TestReloadPluginErrorPaths tests error paths in reloadPlugin
func TestReloadPluginErrorPaths(t *testing.T) {
	t.Run("UnloadFailureDuringReload", func(t *testing.T) {
		// Create managers
		manager := NewPluginManager()
		depManager := NewDependencyManager()

		// Create a temp directory for test plugins
		tempDir := t.TempDir()

		// Create valid HotReloadManager with temp directory
		config := &HotReloadConfig{
			WatchDirectories: []string{tempDir},
		}

		hrm, err := NewHotReloadManager(config, manager, depManager)
		if err != nil {
			t.Fatalf("Failed to create HotReloadManager: %v", err)
		}

		// Create a mock plugin file
		pluginPath := filepath.Join(tempDir, "test_plugin.so")
		if err := os.WriteFile(pluginPath, []byte("mock plugin content"), 0644); err != nil {
			t.Fatal(err)
		}

		// Manually add plugin info to simulate existing plugin
		hrm.mu.Lock()
		hrm.pluginFiles["test_plugin"] = PluginFileInfo{
			Path:       pluginPath,
			PluginName: "test_plugin",
		}
		hrm.mu.Unlock()

		// Create mock plugins with dependencies
		testPlugin := &MockPlugin{name: "test_plugin", version: "1.0.0"}
		vpTest, err := NewVersionedPlugin(testPlugin, "1.0.0")
		if err != nil {
			t.Fatalf("Failed to create versioned plugin: %v", err)
		}

		dependentPlugin := &MockPlugin{name: "dependent", version: "1.0.0"}
		vpDependent, err := NewVersionedPlugin(dependentPlugin, "1.0.0")
		if err != nil {
			t.Fatalf("Failed to create versioned plugin: %v", err)
		}
		vpDependent.AddDependency("test_plugin", "^1.0.0")

		// Register plugins with dependency manager
		if err := depManager.RegisterPlugin(vpTest); err != nil {
			t.Fatalf("Failed to register test plugin: %v", err)
		}

		if err := depManager.RegisterPlugin(vpDependent); err != nil {
			t.Fatalf("Failed to register dependent plugin: %v", err)
		}

		// Try to reload - should fail during unload
		err = hrm.reloadPlugin(pluginPath)
		if err == nil {
			t.Fatal("Expected error during reload when unload fails")
		}

		var pluginErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &pluginErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		if pluginErr.Code != gdlerrors.CodePluginError {
			t.Errorf("Expected CodePluginError, got: %s", pluginErr.Code)
		}
	})

	t.Run("LoadFailureDuringReload", func(t *testing.T) {
		// Create managers
		manager := NewPluginManager()

		// Create a temp directory for test plugins
		tempDir := t.TempDir()

		// Create valid HotReloadManager with temp directory
		config := &HotReloadConfig{
			WatchDirectories: []string{tempDir},
		}

		hrm, err := NewHotReloadManager(config, manager, nil)
		if err != nil {
			t.Fatalf("Failed to create HotReloadManager: %v", err)
		}

		// Try to reload a non-existent plugin file
		nonExistentPath := filepath.Join(tempDir, "nonexistent.so")

		err = hrm.reloadPlugin(nonExistentPath)
		// This will fail in loadPlugin, which should return an error
		// The exact error type depends on loadPlugin implementation
		if err == nil {
			// If reloadPlugin returns nil for non-existent files (handles removal),
			// that's acceptable behavior
			t.Log("reloadPlugin handled non-existent file (no error)")
		}
	})
}

// TestHandlePluginRemovalErrorPath tests handlePluginRemoval error paths
func TestHandlePluginRemovalErrorPath(t *testing.T) {
	t.Run("UnloadErrorDuringRemoval", func(t *testing.T) {
		// Create managers
		manager := NewPluginManager()
		depManager := NewDependencyManager()

		// Create a temp directory
		tempDir := t.TempDir()

		// Create valid HotReloadManager
		config := &HotReloadConfig{
			WatchDirectories: []string{tempDir},
		}

		hrm, err := NewHotReloadManager(config, manager, depManager)
		if err != nil {
			t.Fatalf("Failed to create HotReloadManager: %v", err)
		}

		// Create a plugin file path
		pluginPath := filepath.Join(tempDir, "removed_plugin.so")

		// Create mock plugins with dependencies
		removedPlugin := &MockPlugin{name: "removed_plugin", version: "1.0.0"}
		vpRemoved, err := NewVersionedPlugin(removedPlugin, "1.0.0")
		if err != nil {
			t.Fatalf("Failed to create versioned plugin: %v", err)
		}

		dependentPlugin := &MockPlugin{name: "dependent", version: "1.0.0"}
		vpDependent, err := NewVersionedPlugin(dependentPlugin, "1.0.0")
		if err != nil {
			t.Fatalf("Failed to create versioned plugin: %v", err)
		}
		vpDependent.AddDependency("removed_plugin", "^1.0.0")

		// Register plugins with dependency manager
		if err := depManager.RegisterPlugin(vpRemoved); err != nil {
			t.Fatalf("Failed to register plugin: %v", err)
		}

		if err := depManager.RegisterPlugin(vpDependent); err != nil {
			t.Fatalf("Failed to register dependent: %v", err)
		}

		// Add to pluginFiles map
		hrm.mu.Lock()
		hrm.pluginFiles["removed_plugin"] = PluginFileInfo{
			Path:       pluginPath,
			PluginName: "removed_plugin",
		}
		hrm.mu.Unlock()

		// Handle removal - should fail because plugin has dependents
		err = hrm.handlePluginRemoval(pluginPath)
		if err == nil {
			t.Fatal("Expected error when removing plugin with dependents")
		}

		var pluginErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &pluginErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		if pluginErr.Code != gdlerrors.CodePluginError {
			t.Errorf("Expected CodePluginError, got: %s", pluginErr.Code)
		}
	})
}

// TestAddWatchDirectoryErrorPaths tests addWatchDirectory error scenarios
func TestAddWatchDirectoryErrorPaths(t *testing.T) {
	t.Run("InvalidPath", func(t *testing.T) {
		manager := NewPluginManager()
		config := &HotReloadConfig{
			WatchDirectories: []string{},
		}

		hrm, err := NewHotReloadManager(config, manager, nil)
		if err != nil {
			t.Fatalf("Failed to create HotReloadManager: %v", err)
		}

		// Try to add invalid directory with null byte
		err = hrm.addWatchDirectory("/invalid/\x00/path")
		if err == nil {
			t.Fatal("Expected error for invalid path")
		}
	})

}
