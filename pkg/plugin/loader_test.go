package plugin

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestNewPluginLoader tests the plugin loader creation
func TestNewPluginLoader(t *testing.T) {
	t.Run("WithNilConfig", func(t *testing.T) {
		loader := NewPluginLoader(nil)

		if loader == nil {
			t.Fatal("Expected loader to be created with default config")
		}

		if len(loader.searchPaths) == 0 {
			t.Error("Expected default search paths to be set")
		}

		if loader.maxPluginSize != 100*1024*1024 {
			t.Errorf("Expected default max plugin size 100MB, got: %d", loader.maxPluginSize)
		}

		if loader.verifyChecksum {
			t.Error("Expected default checksum verification to be false")
		}

		if loader.loadedPlugins == nil {
			t.Error("Expected loadedPlugins map to be initialized")
		}
	})

	t.Run("WithCustomConfig", func(t *testing.T) {
		config := &LoaderConfig{
			SearchPaths:    []string{"/custom/path1", "/custom/path2"},
			AllowedPaths:   []string{"/allowed/path"},
			BlockedPaths:   []string{"/blocked/path"},
			VerifyChecksum: true,
			MaxPluginSize:  50 * 1024 * 1024, // 50MB
		}

		loader := NewPluginLoader(config)

		if len(loader.searchPaths) != 2 {
			t.Errorf("Expected 2 search paths, got: %d", len(loader.searchPaths))
		}

		if loader.searchPaths[0] != "/custom/path1" {
			t.Errorf("Expected first search path '/custom/path1', got: %s", loader.searchPaths[0])
		}

		if len(loader.allowedPaths) != 1 {
			t.Errorf("Expected 1 allowed path, got: %d", len(loader.allowedPaths))
		}

		if len(loader.blockedPaths) != 1 {
			t.Errorf("Expected 1 blocked path, got: %d", len(loader.blockedPaths))
		}

		if !loader.verifyChecksum {
			t.Error("Expected checksum verification to be enabled")
		}

		if loader.maxPluginSize != 50*1024*1024 {
			t.Errorf("Expected max plugin size 50MB, got: %d", loader.maxPluginSize)
		}
	})
}

// TestSearchPathOperations tests search path management
func TestSearchPathOperations(t *testing.T) {
	loader := NewPluginLoader(&LoaderConfig{
		SearchPaths: []string{"/initial/path"},
	})

	t.Run("GetSearchPaths", func(t *testing.T) {
		paths := loader.GetSearchPaths()

		if len(paths) != 1 {
			t.Errorf("Expected 1 search path, got: %d", len(paths))
		}

		if paths[0] != "/initial/path" {
			t.Errorf("Expected '/initial/path', got: %s", paths[0])
		}

		// Verify it's a copy (modifying returned slice shouldn't affect internal)
		paths[0] = "/modified"
		internalPaths := loader.GetSearchPaths()
		if internalPaths[0] == "/modified" {
			t.Error("Returned search paths should be a copy")
		}
	})

	t.Run("AddSearchPath", func(t *testing.T) {
		// Create temporary directory for testing
		tempDir := t.TempDir()

		err := loader.AddSearchPath(tempDir)
		if err != nil {
			t.Fatalf("Failed to add search path: %v", err)
		}

		paths := loader.GetSearchPaths()
		if len(paths) != 2 {
			t.Errorf("Expected 2 search paths after adding, got: %d", len(paths))
		}

		found := false
		for _, path := range paths {
			if strings.Contains(path, tempDir) {
				found = true
				break
			}
		}
		if !found {
			t.Error("Added search path not found in search paths")
		}

		// Test adding duplicate path
		err = loader.AddSearchPath(tempDir)
		if err != nil {
			t.Errorf("Adding duplicate path should not error: %v", err)
		}

		pathsAfterDuplicate := loader.GetSearchPaths()
		if len(pathsAfterDuplicate) != 2 {
			t.Errorf("Expected 2 search paths after adding duplicate, got: %d", len(pathsAfterDuplicate))
		}
	})

	t.Run("RemoveSearchPath", func(t *testing.T) {
		tempDir := t.TempDir()
		_ = loader.AddSearchPath(tempDir)

		initialCount := len(loader.GetSearchPaths())

		loader.RemoveSearchPath(tempDir)

		paths := loader.GetSearchPaths()
		if len(paths) != initialCount-1 {
			t.Errorf("Expected %d search paths after removal, got: %d", initialCount-1, len(paths))
		}

		// Verify the path was actually removed
		for _, path := range paths {
			if strings.Contains(path, tempDir) {
				t.Error("Removed path still exists in search paths")
			}
		}

		// Test removing non-existent path (should not error)
		loader.RemoveSearchPath("/non/existent/path")
	})
}

// TestValidatePath tests path validation functionality
func TestValidatePath(t *testing.T) {
	// Create temporary directories for testing
	allowedDir := t.TempDir()
	blockedDir := t.TempDir()

	loader := NewPluginLoader(&LoaderConfig{
		AllowedPaths: []string{allowedDir},
		BlockedPaths: []string{blockedDir},
	})

	t.Run("AllowedPath", func(t *testing.T) {
		testPath := filepath.Join(allowedDir, "test.so")
		err := loader.validatePath(testPath)
		if err != nil {
			t.Errorf("Expected allowed path to be valid, got error: %v", err)
		}
	})

	t.Run("BlockedPath", func(t *testing.T) {
		testPath := filepath.Join(blockedDir, "test.so")
		err := loader.validatePath(testPath)
		if err == nil {
			t.Error("Expected blocked path to be invalid")
		}

		if !strings.Contains(err.Error(), "path is blocked") {
			t.Errorf("Expected 'path is blocked' error, got: %v", err)
		}
	})

	t.Run("PathNotInAllowedList", func(t *testing.T) {
		otherDir := t.TempDir()
		testPath := filepath.Join(otherDir, "test.so")
		err := loader.validatePath(testPath)
		if err == nil {
			t.Error("Expected path not in allowed list to be invalid")
		}

		if !strings.Contains(err.Error(), "path not in allowed paths") {
			t.Errorf("Expected 'path not in allowed paths' error, got: %v", err)
		}
	})

	t.Run("NoRestrictionsApplied", func(t *testing.T) {
		// Loader with no path restrictions
		unrestricted := NewPluginLoader(&LoaderConfig{})

		tempDir := t.TempDir()
		testPath := filepath.Join(tempDir, "test.so")
		err := unrestricted.validatePath(testPath)
		if err != nil {
			t.Errorf("Expected unrestricted path to be valid, got error: %v", err)
		}
	})
}

// TestCalculateChecksum tests checksum calculation
func TestCalculateChecksum(t *testing.T) {
	loader := NewPluginLoader(&LoaderConfig{})

	t.Run("ValidFile", func(t *testing.T) {
		// Create a temporary file
		tempFile, err := os.CreateTemp("", "checksum_test_*.txt")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer func() { _ = os.Remove(tempFile.Name()) }()
		defer func() { _ = tempFile.Close() }()

		// Write some test data
		testData := "test checksum data"
		if _, err := tempFile.WriteString(testData); err != nil {
			t.Fatalf("Failed to write test data: %v", err)
		}
		_ = tempFile.Close()

		checksum, err := loader.calculateChecksum(tempFile.Name())
		if err != nil {
			t.Fatalf("Failed to calculate checksum: %v", err)
		}

		if len(checksum) != 64 { // SHA256 hex string length
			t.Errorf("Expected checksum length 64, got: %d", len(checksum))
		}

		// Calculate the same checksum again to verify consistency
		checksum2, err := loader.calculateChecksum(tempFile.Name())
		if err != nil {
			t.Fatalf("Failed to calculate checksum second time: %v", err)
		}

		if checksum != checksum2 {
			t.Error("Checksums should be consistent")
		}
	})

	t.Run("NonExistentFile", func(t *testing.T) {
		_, err := loader.calculateChecksum("/non/existent/file")
		if err == nil {
			t.Error("Expected error for non-existent file")
		}
	})
}

// MockLoaderAuthPlugin implements AuthPlugin interface matching the actual interface
type MockLoaderAuthPlugin struct {
	*MockPlugin
}

func NewMockLoaderAuthPlugin(name, version string) *MockLoaderAuthPlugin {
	return &MockLoaderAuthPlugin{
		MockPlugin: NewMockPlugin(name, version),
	}
}

func (m *MockLoaderAuthPlugin) Authenticate(ctx context.Context, req *http.Request) error {
	return nil
}

// MockLoaderTransformPlugin implements TransformPlugin interface matching the actual interface
type MockLoaderTransformPlugin struct {
	*MockPlugin
}

func NewMockLoaderTransformPlugin(name, version string) *MockLoaderTransformPlugin {
	return &MockLoaderTransformPlugin{
		MockPlugin: NewMockPlugin(name, version),
	}
}

func (m *MockLoaderTransformPlugin) Transform(data []byte) ([]byte, error) {
	return data, nil
}

// TestDeterminePluginType tests plugin type detection
func TestDeterminePluginType(t *testing.T) {
	loader := NewPluginLoader(&LoaderConfig{})

	t.Run("AuthPlugin", func(t *testing.T) {
		authPlugin := NewMockLoaderAuthPlugin("auth-test", "1.0.0")
		pluginType := loader.determinePluginType(authPlugin)
		if pluginType != "auth" {
			t.Errorf("Expected 'auth', got: %s", pluginType)
		}
	})

	t.Run("TransformPlugin", func(t *testing.T) {
		transformPlugin := NewMockLoaderTransformPlugin("transform-test", "1.0.0")
		pluginType := loader.determinePluginType(transformPlugin)
		if pluginType != "transform" {
			t.Errorf("Expected 'transform', got: %s", pluginType)
		}
	})

	t.Run("UnknownPlugin", func(t *testing.T) {
		basicPlugin := NewMockPlugin("basic-test", "1.0.0")
		pluginType := loader.determinePluginType(basicPlugin)
		if pluginType != "unknown" {
			t.Errorf("Expected 'unknown', got: %s", pluginType)
		}
	})
}

// TestGetLoadedPlugins tests getting loaded plugin information
func TestGetLoadedPlugins(t *testing.T) {
	loader := NewPluginLoader(&LoaderConfig{})

	// Initially should be empty
	loaded := loader.GetLoadedPlugins()
	if len(loaded) != 0 {
		t.Errorf("Expected 0 loaded plugins initially, got: %d", len(loaded))
	}

	// Manually add a mock plugin to the loaded plugins map for testing
	mockPlugin := NewMockPlugin("test-plugin", "1.0.0")
	loader.loadedPlugins["/test/path"] = &PluginInfo{
		Path:     "/test/path",
		Name:     mockPlugin.Name(),
		Version:  mockPlugin.Version(),
		Type:     "unknown",
		LoadTime: time.Now(),
		Plugin:   mockPlugin,
	}

	loaded = loader.GetLoadedPlugins()
	if len(loaded) != 1 {
		t.Errorf("Expected 1 loaded plugin, got: %d", len(loaded))
	}

	// Verify it's a copy (modifying returned map shouldn't affect internal)
	delete(loaded, "/test/path")
	internalLoaded := loader.GetLoadedPlugins()
	if len(internalLoaded) == 0 {
		t.Error("Returned loaded plugins should be a copy")
	}
}

// TestVerifyPlugin tests plugin verification
func TestVerifyPlugin(t *testing.T) {
	loader := NewPluginLoader(&LoaderConfig{
		MaxPluginSize: 1024, // 1KB for testing
	})

	t.Run("ValidPlugin", func(t *testing.T) {
		// Skip this test on Windows as plugins are not supported
		if runtime.GOOS == "windows" {
			t.Skip("Skipping plugin test on Windows - plugins not supported")
		}

		// Create a small executable temporary file
		tempFile, err := os.CreateTemp("", "verify_test_*.so")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer func() { _ = os.Remove(tempFile.Name()) }()
		defer func() { _ = tempFile.Close() }()

		// Write some test data (less than 1KB)
		testData := "small plugin file"
		if _, err := tempFile.WriteString(testData); err != nil {
			t.Fatalf("Failed to write test data: %v", err)
		}
		_ = tempFile.Close()

		// Make file executable
		if err := os.Chmod(tempFile.Name(), 0755); err != nil {
			t.Fatalf("Failed to make file executable: %v", err)
		}

		err = loader.VerifyPlugin(tempFile.Name())
		if err != nil {
			t.Errorf("Expected valid plugin verification to pass, got: %v", err)
		}
	})

	t.Run("TooLargePlugin", func(t *testing.T) {
		// Create a large temporary file
		tempFile, err := os.CreateTemp("", "large_verify_test_*.so")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer func() { _ = os.Remove(tempFile.Name()) }()
		defer func() { _ = tempFile.Close() }()

		// Write data larger than maxPluginSize (1KB)
		largeData := strings.Repeat("x", 2048) // 2KB
		if _, err := tempFile.WriteString(largeData); err != nil {
			t.Fatalf("Failed to write test data: %v", err)
		}
		_ = tempFile.Close()

		// Make file executable
		if err := os.Chmod(tempFile.Name(), 0755); err != nil {
			t.Fatalf("Failed to make file executable: %v", err)
		}

		err = loader.VerifyPlugin(tempFile.Name())
		if err == nil {
			t.Error("Expected error for plugin file too large")
		}

		if !strings.Contains(err.Error(), "plugin file too large") &&
			!strings.Contains(err.Error(), "file too large") {
			t.Errorf("Expected 'plugin file too large' error, got: %v", err)
		}
	})

	t.Run("NonExecutablePlugin", func(t *testing.T) {
		// Create a non-executable temporary file
		tempFile, err := os.CreateTemp("", "non_exec_test_*.so")
		if err != nil {
			t.Fatalf("Failed to create temp file: %v", err)
		}
		defer func() { _ = os.Remove(tempFile.Name()) }()
		defer func() { _ = tempFile.Close() }()

		// Write some test data
		testData := "non-executable plugin"
		if _, err := tempFile.WriteString(testData); err != nil {
			t.Fatalf("Failed to write test data: %v", err)
		}
		_ = tempFile.Close()

		// Make file non-executable
		if err := os.Chmod(tempFile.Name(), 0644); err != nil {
			t.Fatalf("Failed to set file permissions: %v", err)
		}

		err = loader.VerifyPlugin(tempFile.Name())
		if err == nil {
			t.Error("Expected error for non-executable plugin file")
		}

		if !strings.Contains(err.Error(), "plugin file is not executable") {
			t.Errorf("Expected 'plugin file is not executable' error, got: %v", err)
		}
	})

	t.Run("NonExistentPlugin", func(t *testing.T) {
		// Use a proper absolute path that doesn't exist
		nonExistentPath := filepath.Join(os.TempDir(), "non_existent_plugin_test_12345.so")
		err := loader.VerifyPlugin(nonExistentPath)
		if err == nil {
			t.Error("Expected error for non-existent plugin file")
		}

		// Check for error about file not existing (message may vary by platform)
		if !strings.Contains(err.Error(), "cannot open plugin file") &&
			!strings.Contains(err.Error(), "no such file") &&
			!strings.Contains(err.Error(), "cannot find") &&
			!strings.Contains(err.Error(), "storage error") {
			t.Errorf("Expected error about non-existent file, got: %v", err)
		}
	})
}

// TestDiscoverPlugins tests plugin discovery functionality
func TestDiscoverPlugins(t *testing.T) {
	// Create temporary directory structure for testing
	testDir := t.TempDir()

	// Create subdirectory
	subDir := filepath.Join(testDir, "subdir")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	// Create test plugin files
	pluginFiles := []string{
		filepath.Join(testDir, "plugin1.so"),
		filepath.Join(testDir, "plugin2.so"),
		filepath.Join(subDir, "plugin3.so"),
		filepath.Join(testDir, "notaplugin.txt"), // Should be ignored
	}

	for _, file := range pluginFiles {
		if err := os.WriteFile(file, []byte("test"), 0644); err != nil {
			t.Fatalf("Failed to create test file %s: %v", file, err)
		}
	}

	loader := NewPluginLoader(&LoaderConfig{
		SearchPaths: []string{testDir},
	})

	t.Run("DiscoverPlugins", func(t *testing.T) {
		discovered, err := loader.DiscoverPlugins()
		if err != nil {
			t.Fatalf("Failed to discover plugins: %v", err)
		}

		// Should find 3 .so files
		if len(discovered) != 3 {
			t.Errorf("Expected 3 discovered plugins, got: %d", len(discovered))
		}

		// Verify all discovered files end with .so
		for _, path := range discovered {
			if !strings.HasSuffix(path, ".so") {
				t.Errorf("Discovered non-plugin file: %s", path)
			}
		}
	})

	t.Run("DiscoverPluginsNonExistentPath", func(t *testing.T) {
		loader := NewPluginLoader(&LoaderConfig{
			SearchPaths: []string{"/non/existent/path", testDir},
		})

		// Should still work and find plugins in the existing path
		discovered, err := loader.DiscoverPlugins()
		if err != nil {
			t.Fatalf("Failed to discover plugins with non-existent path: %v", err)
		}

		if len(discovered) != 3 {
			t.Errorf("Expected 3 discovered plugins even with non-existent path, got: %d", len(discovered))
		}
	})
}

// TestLoadFromSearchPath tests loading plugins from search paths
func TestLoadFromSearchPath(t *testing.T) {
	loader := NewPluginLoader(&LoaderConfig{
		SearchPaths: []string{"/non/existent/path1", "/non/existent/path2"},
	})

	t.Run("PluginNotFound", func(t *testing.T) {
		_, err := loader.LoadFromSearchPath("nonexistent.so")
		if err == nil {
			t.Error("Expected error when plugin not found in search paths")
		}

		if !strings.Contains(err.Error(), "not found in search paths") &&
			!strings.Contains(err.Error(), "plugin error") {
			t.Errorf("Expected 'not found in search paths' error, got: %v", err)
		}
	})
}

// TestLoadAllPlugins tests loading all discovered plugins
func TestLoadAllPlugins(t *testing.T) {
	// Create a loader with non-existent search paths to test error handling
	loader := NewPluginLoader(&LoaderConfig{
		SearchPaths: []string{"/non/existent/path"},
	})

	t.Run("NoPluginsToLoad", func(t *testing.T) {
		plugins, errors := loader.LoadAll()

		// Should return empty results, no errors for empty discovery
		if len(plugins) != 0 {
			t.Errorf("Expected 0 plugins loaded, got: %d", len(plugins))
		}

		if len(errors) != 0 {
			t.Errorf("Expected 0 errors, got: %d", len(errors))
		}
	})
}

func TestLoadAllPlugins_WithDiscoveredPlugins(t *testing.T) {
	tempDir := t.TempDir()
	loader := NewPluginLoader(nil)
	_ = loader.AddSearchPath(tempDir)

	// Create mock plugin files
	pluginFiles := []string{"auth.so", "transform.so", "invalid.txt"}
	for _, filename := range pluginFiles {
		path := filepath.Join(tempDir, filename)
		err := os.WriteFile(path, []byte("mock plugin content"), 0755)
		if err != nil {
			t.Fatalf("Failed to create mock plugin file: %v", err)
		}
	}

	// LoadAll will discover the .so files but fail to load them (mock files)
	plugins, errors := loader.LoadAll()

	// Should have found some files but failed to load them
	if len(errors) == 0 {
		t.Error("Expected errors when loading mock plugin files")
	}

	if len(plugins) != 0 {
		t.Errorf("Expected 0 plugins loaded, got %d", len(plugins))
	}

	// Check that errors are properly formatted
	for _, err := range errors {
		if !containsString(err.Error(), "failed to load") &&
			!containsString(err.Error(), "plugin error") {
			t.Errorf("Expected 'failed to load' in error message, got: %v", err)
		}
	}
}

// TestUnloadOperations tests plugin unloading functionality
func TestUnloadOperations(t *testing.T) {
	loader := NewPluginLoader(&LoaderConfig{})

	// Manually add a mock plugin to test unloading
	mockPlugin := NewMockPlugin("unload-test", "1.0.0")
	loader.loadedPlugins["/test/unload"] = &PluginInfo{
		Path:     "/test/unload",
		Name:     mockPlugin.Name(),
		Version:  mockPlugin.Version(),
		Type:     "unknown",
		LoadTime: time.Now(),
		Plugin:   mockPlugin,
	}

	t.Run("UnloadPlugin", func(t *testing.T) {
		err := loader.UnloadPlugin("/test/unload")
		if err != nil {
			t.Errorf("Failed to unload plugin: %v", err)
		}

		// Verify plugin was removed
		if len(loader.loadedPlugins) != 0 {
			t.Error("Plugin should have been removed from loaded plugins")
		}

		// Verify plugin was closed
		if !mockPlugin.IsClosed() {
			t.Error("Plugin should have been closed")
		}
	})

	t.Run("UnloadNonExistentPlugin", func(t *testing.T) {
		err := loader.UnloadPlugin("/non/existent/plugin")
		if err == nil {
			t.Error("Expected error when unloading non-existent plugin")
		}

		if !strings.Contains(err.Error(), "is not loaded") &&
			!strings.Contains(err.Error(), "plugin error") {
			t.Errorf("Expected 'is not loaded' error, got: %v", err)
		}
	})

	t.Run("UnloadAll", func(t *testing.T) {
		// Add multiple mock plugins
		plugin1 := NewMockPlugin("unload-all-1", "1.0.0")
		plugin2 := NewMockPlugin("unload-all-2", "1.0.0")
		plugin3 := NewMockPlugin("unload-all-3", "1.0.0")
		plugin3.SetCloseError(fmt.Errorf("close error"))

		loader.loadedPlugins["/test/1"] = &PluginInfo{Plugin: plugin1}
		loader.loadedPlugins["/test/2"] = &PluginInfo{Plugin: plugin2}
		loader.loadedPlugins["/test/3"] = &PluginInfo{Plugin: plugin3}

		errors := loader.UnloadAll()

		// Should have one error from plugin3
		if len(errors) != 1 {
			t.Errorf("Expected 1 error, got: %d", len(errors))
		}

		// All plugins should be removed from map
		if len(loader.loadedPlugins) != 0 {
			t.Errorf("Expected all plugins to be unloaded, got: %d remaining", len(loader.loadedPlugins))
		}

		// Verify plugins were closed (even the one that errored)
		if !plugin1.IsClosed() {
			t.Error("Plugin1 should have been closed")
		}
		if !plugin2.IsClosed() {
			t.Error("Plugin2 should have been closed")
		}
		// plugin3 won't be closed due to error, which is expected
	})
}

// TestGetPluginByName tests finding plugins by name
func TestGetPluginByName(t *testing.T) {
	loader := NewPluginLoader(&LoaderConfig{})

	// Add mock plugins
	plugin1 := NewMockPlugin("findable-plugin", "1.0.0")
	plugin2 := NewMockPlugin("another-plugin", "2.0.0")

	loader.loadedPlugins["/test/1"] = &PluginInfo{
		Name:   plugin1.Name(),
		Plugin: plugin1,
	}
	loader.loadedPlugins["/test/2"] = &PluginInfo{
		Name:   plugin2.Name(),
		Plugin: plugin2,
	}

	t.Run("FindExistingPlugin", func(t *testing.T) {
		found, err := loader.GetPluginByName("findable-plugin")
		if err != nil {
			t.Errorf("Failed to find existing plugin: %v", err)
		}

		if found.Name() != "findable-plugin" {
			t.Errorf("Expected plugin name 'findable-plugin', got: %s", found.Name())
		}
	})

	t.Run("FindNonExistentPlugin", func(t *testing.T) {
		_, err := loader.GetPluginByName("non-existent")
		if err == nil {
			t.Error("Expected error when finding non-existent plugin")
		}

		if !strings.Contains(err.Error(), "not found") &&
			!strings.Contains(err.Error(), "plugin error") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})
}

// TestGetPluginsByType tests finding plugins by type
func TestGetPluginsByType(t *testing.T) {
	loader := NewPluginLoader(&LoaderConfig{})

	// Add mock plugins of different types
	authPlugin := NewMockAuthPlugin("auth-plugin", "1.0.0")
	transformPlugin := NewMockTransformPlugin("transform-plugin", "1.0.0")
	basicPlugin := NewMockPlugin("basic-plugin", "1.0.0")

	loader.loadedPlugins["/test/auth"] = &PluginInfo{
		Type:   "auth",
		Plugin: authPlugin,
	}
	loader.loadedPlugins["/test/transform"] = &PluginInfo{
		Type:   "transform",
		Plugin: transformPlugin,
	}
	loader.loadedPlugins["/test/basic"] = &PluginInfo{
		Type:   "unknown",
		Plugin: basicPlugin,
	}

	t.Run("FindAuthPlugins", func(t *testing.T) {
		authPlugins := loader.GetPluginsByType("auth")
		if len(authPlugins) != 1 {
			t.Errorf("Expected 1 auth plugin, got: %d", len(authPlugins))
		}

		if authPlugins[0].Name() != "auth-plugin" {
			t.Errorf("Expected auth plugin name 'auth-plugin', got: %s", authPlugins[0].Name())
		}
	})

	t.Run("FindTransformPlugins", func(t *testing.T) {
		transformPlugins := loader.GetPluginsByType("transform")
		if len(transformPlugins) != 1 {
			t.Errorf("Expected 1 transform plugin, got: %d", len(transformPlugins))
		}
	})

	t.Run("FindNonExistentType", func(t *testing.T) {
		noPlugins := loader.GetPluginsByType("non-existent-type")
		if len(noPlugins) != 0 {
			t.Errorf("Expected 0 plugins for non-existent type, got: %d", len(noPlugins))
		}
	})
}

// TestReloadPlugin tests plugin reloading functionality
func TestReloadPlugin(t *testing.T) {
	loader := NewPluginLoader(&LoaderConfig{})

	t.Run("ReloadNonExistentPlugin", func(t *testing.T) {
		// This will try to unload a non-existent plugin and then load it
		// Since the plugin file doesn't exist, it should fail at the load step
		_, err := loader.ReloadPlugin("/non/existent/plugin.so")
		if err == nil {
			t.Error("Expected error when reloading non-existent plugin")
		}

		// The error should be from the Load operation, not the Unload
		if strings.Contains(err.Error(), "failed to unload plugin for reload") {
			t.Errorf("Expected load error, got unload error: %v", err)
		}
	})
}

// Benchmark tests for performance verification
func BenchmarkPluginLoaderOperations(b *testing.B) {
	loader := NewPluginLoader(&LoaderConfig{})

	b.Run("GetSearchPaths", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_ = loader.GetSearchPaths()
		}
	})

	b.Run("GetLoadedPlugins", func(b *testing.B) {
		// Add some plugins for more realistic benchmarking
		for i := 0; i < 100; i++ {
			plugin := NewMockPlugin(fmt.Sprintf("bench-plugin-%d", i), "1.0.0")
			loader.loadedPlugins[fmt.Sprintf("/test/%d", i)] = &PluginInfo{
				Name:   plugin.Name(),
				Plugin: plugin,
				Type:   "unknown",
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = loader.GetLoadedPlugins()
		}
	})

	b.Run("GetPluginsByType", func(b *testing.B) {
		// Use the plugins added in previous benchmark
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = loader.GetPluginsByType("unknown")
		}
	})
}
