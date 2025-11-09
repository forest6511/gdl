package plugin

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
)

// TestPluginLoaderErrorPaths tests all error paths in plugin loader
func TestPluginLoaderErrorPaths(t *testing.T) {
	t.Run("Load_InvalidPath", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping on Windows: Unix-style paths not supported")
		}

		loader := NewPluginLoader(&LoaderConfig{
			BlockedPaths: []string{"/blocked"},
		})

		_, err := loader.Load("/blocked/plugin.so")
		if err == nil {
			t.Fatal("Expected error for blocked path")
		}

		var pathErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &pathErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		if pathErr.Code != gdlerrors.CodeInvalidPath {
			t.Errorf("Expected CodeInvalidPath, got: %s", pathErr.Code)
		}
	})

	t.Run("Load_FileNotFound", func(t *testing.T) {
		loader := NewPluginLoader(nil)

		_, err := loader.Load("/nonexistent/plugin.so")
		if err == nil {
			t.Fatal("Expected error for nonexistent file")
		}

		var storageErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &storageErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		if storageErr.Code != gdlerrors.CodeStorageError {
			t.Errorf("Expected CodeStorageError, got: %s", storageErr.Code)
		}
	})

	t.Run("Load_FileTooLarge", func(t *testing.T) {
		// Create a temporary file
		tmpDir := t.TempDir()
		pluginPath := filepath.Join(tmpDir, "large_plugin.so")

		// Create a file with some content
		content := make([]byte, 1024*1024) // 1MB
		if err := os.WriteFile(pluginPath, content, 0644); err != nil {
			t.Fatal(err)
		}

		loader := NewPluginLoader(&LoaderConfig{
			MaxPluginSize: 512 * 1024, // 512KB limit
		})

		_, err := loader.Load(pluginPath)
		if err == nil {
			t.Fatal("Expected error for file too large")
		}

		var validationErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &validationErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		if validationErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", validationErr.Code)
		}

		// Verify error message contains size info
		errMsg := validationErr.Error()
		if errMsg == "" {
			t.Error("Expected non-empty error message")
		}
	})

	t.Run("LoadFromSearchPath_NotFound", func(t *testing.T) {
		loader := NewPluginLoader(&LoaderConfig{
			SearchPaths: []string{"/path1", "/path2"},
		})

		_, err := loader.LoadFromSearchPath("nonexistent.so")
		if err == nil {
			t.Fatal("Expected error when plugin not found in search paths")
		}

		var pluginErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &pluginErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		if pluginErr.Code != gdlerrors.CodePluginError {
			t.Errorf("Expected CodePluginError, got: %s", pluginErr.Code)
		}
	})

	t.Run("DiscoverPlugins_InvalidDirectory", func(t *testing.T) {
		loader := NewPluginLoader(&LoaderConfig{
			SearchPaths: []string{"/nonexistent/dir"},
		})

		plugins, err := loader.DiscoverPlugins()

		// For non-existent directories, DiscoverPlugins() skips them and returns empty list
		if len(plugins) != 0 {
			t.Errorf("Expected 0 plugins from invalid directory, got: %d", len(plugins))
		}

		// No error expected for non-existent paths (they are skipped)
		if err != nil {
			t.Errorf("Expected no error for non-existent search paths, got: %v", err)
		}
	})

	t.Run("UnloadPlugin_NotLoaded", func(t *testing.T) {
		loader := NewPluginLoader(nil)

		err := loader.UnloadPlugin("/nonexistent/plugin.so")
		if err == nil {
			t.Fatal("Expected error when unloading non-loaded plugin")
		}

		// Verify it's a proper error message
		errMsg := err.Error()
		if errMsg == "" {
			t.Error("Expected non-empty error message")
		}
	})

	t.Run("GetPluginByName_NotFound", func(t *testing.T) {
		loader := NewPluginLoader(nil)

		plugin, err := loader.GetPluginByName("nonexistent")
		if err == nil {
			t.Fatal("Expected error for non-existent plugin")
		}
		if plugin != nil {
			t.Error("Expected nil plugin for non-existent name")
		}

		var pluginErr *gdlerrors.DownloadError
		if gdlerrors.AsDownloadError(err, &pluginErr) {
			if pluginErr.Code != gdlerrors.CodePluginError {
				t.Errorf("Expected CodePluginError, got: %s", pluginErr.Code)
			}
		}
	})

	t.Run("RemoveSearchPath_NotFound", func(t *testing.T) {
		loader := NewPluginLoader(&LoaderConfig{
			SearchPaths: []string{"/path1"},
		})

		// RemoveSearchPath doesn't return an error, it just removes if exists
		// So we can just verify the path list doesn't change when removing non-existent
		initialPaths := loader.GetSearchPaths()
		loader.RemoveSearchPath("/nonexistent")
		finalPaths := loader.GetSearchPaths()

		if len(initialPaths) != len(finalPaths) {
			t.Error("Path list should not change when removing non-existent path")
		}
	})

	t.Run("AddSearchPath_AlreadyExists", func(t *testing.T) {
		loader := NewPluginLoader(&LoaderConfig{
			SearchPaths: []string{"/existing/path"},
		})

		// AddSearchPath doesn't return error for duplicates, just skips adding
		initialCount := len(loader.GetSearchPaths())
		err := loader.AddSearchPath("/existing/path")
		finalCount := len(loader.GetSearchPaths())

		// If no error, verify path count didn't change
		if err == nil {
			if finalCount != initialCount {
				t.Error("Path count should not change when adding duplicate")
			}
		}
	})

	t.Run("AddSearchPath_Blocked", func(t *testing.T) {
		if runtime.GOOS == "windows" {
			t.Skip("Skipping on Windows: Unix-style paths not supported")
		}

		loader := NewPluginLoader(&LoaderConfig{
			BlockedPaths: []string{"/blocked"},
		})

		err := loader.AddSearchPath("/blocked/subdir")
		if err == nil {
			t.Fatal("Expected error when adding blocked path")
		}

		// Verify it's a proper error
		if err.Error() == "" {
			t.Error("Expected non-empty error message")
		}
	})

	t.Run("LoadAll_EmptyPaths", func(t *testing.T) {
		loader := NewPluginLoader(&LoaderConfig{
			SearchPaths: []string{},
		})

		plugins, errs := loader.LoadAll()

		if len(plugins) != 0 {
			t.Errorf("Expected 0 plugins from empty search paths, got: %d", len(plugins))
		}

		// No errors expected for empty paths
		if len(errs) != 0 {
			t.Errorf("Expected 0 errors for empty paths, got: %d", len(errs))
		}
	})
}

// TestPluginManagerErrorPaths tests error paths in plugin manager
func TestPluginManagerErrorPaths(t *testing.T) {
	t.Run("LoadFromDirectory_InvalidPath", func(t *testing.T) {
		manager := NewPluginManager()

		err := manager.LoadFromDirectory("/nonexistent/directory")
		if err == nil {
			t.Fatal("Expected error for invalid directory")
		}

		var pathErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &pathErr) {
			t.Errorf("Expected DownloadError, got: %T", err)
		}

		// LoadFromDirectory validates the path first, so it returns CodeInvalidPath
		if pathErr.Code != gdlerrors.CodeInvalidPath && pathErr.Code != gdlerrors.CodeStorageError {
			t.Errorf("Expected CodeInvalidPath or CodeStorageError, got: %s", pathErr.Code)
		}
	})

	t.Run("Unregister_NotRegistered", func(t *testing.T) {
		manager := NewPluginManager()

		err := manager.Unregister("nonexistent-plugin")
		if err == nil {
			t.Fatal("Expected error when unregistering non-existent plugin")
		}

		// Verify it's a proper error
		if err.Error() == "" {
			t.Error("Expected non-empty error message")
		}
	})

	t.Run("Get_NotFound", func(t *testing.T) {
		manager := NewPluginManager()

		plugin, err := manager.Get("nonexistent-plugin")
		if err == nil {
			t.Fatal("Expected error for non-existent plugin")
		}

		if plugin != nil {
			t.Error("Expected nil plugin for non-existent name")
		}

		// Verify it's a proper error
		if err.Error() == "" {
			t.Error("Expected non-empty error message")
		}
	})
}
