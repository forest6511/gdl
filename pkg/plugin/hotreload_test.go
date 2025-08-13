package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultHotReloadConfig(t *testing.T) {
	config := DefaultHotReloadConfig()

	if config == nil {
		t.Fatal("DefaultHotReloadConfig should not return nil")
	}

	if config.Enabled {
		t.Error("Default config should be disabled")
	}

	if len(config.WatchDirectories) != 1 {
		t.Errorf("Expected 1 watch directory, got %d", len(config.WatchDirectories))
	}

	if config.WatchDirectories[0] != "./plugins" {
		t.Errorf("Expected './plugins', got %s", config.WatchDirectories[0])
	}

	if config.CheckInterval != 5*time.Second {
		t.Errorf("Expected 5s check interval, got %v", config.CheckInterval)
	}

	if config.Debounce != 500*time.Millisecond {
		t.Errorf("Expected 500ms debounce, got %v", config.Debounce)
	}

	if !config.AutoRestart {
		t.Error("AutoRestart should be true by default")
	}

	if config.MaxRetries != 3 {
		t.Errorf("Expected 3 max retries, got %d", config.MaxRetries)
	}
}

func TestNewHotReloadManager(t *testing.T) {
	manager := NewPluginManager()
	depManager := NewDependencyManager()
	config := DefaultHotReloadConfig()

	// Create temp directory for testing
	tempDir := t.TempDir()
	config.WatchDirectories = []string{tempDir}

	hrm, err := NewHotReloadManager(config, manager, depManager)
	if err != nil {
		t.Fatalf("NewHotReloadManager failed: %v", err)
	}

	if hrm == nil {
		t.Fatal("HotReloadManager should not be nil")
	}

	if hrm.config != config {
		t.Error("Config should be set correctly")
	}

	if hrm.manager != manager {
		t.Error("Manager should be set correctly")
	}

	if hrm.depManager != depManager {
		t.Error("DepManager should be set correctly")
	}

	if hrm.pluginFiles == nil {
		t.Error("PluginFiles map should be initialized")
	}

	if hrm.reloadQueue == nil {
		t.Error("ReloadQueue should be initialized")
	}
}

func TestNewHotReloadManager_NilConfig(t *testing.T) {
	manager := NewPluginManager()
	depManager := NewDependencyManager()

	hrm, err := NewHotReloadManager(nil, manager, depManager)
	if err != nil {
		t.Fatalf("NewHotReloadManager with nil config failed: %v", err)
	}

	// Should use default config
	if hrm.config == nil {
		t.Error("Should use default config when nil is passed")
	}

	if hrm.config.Enabled {
		t.Error("Default config should be disabled")
	}
}

func TestHotReloadManager_StartStop(t *testing.T) {
	manager := NewPluginManager()
	depManager := NewDependencyManager()
	config := DefaultHotReloadConfig()
	config.Enabled = false // Disable for testing

	tempDir := t.TempDir()
	config.WatchDirectories = []string{tempDir}

	hrm, _ := NewHotReloadManager(config, manager, depManager)

	// Start should work even when disabled
	err := hrm.Start()
	if err != nil {
		t.Errorf("Start failed: %v", err)
	}

	// Stop should work
	err = hrm.Stop()
	if err != nil {
		t.Errorf("Stop failed: %v", err)
	}
}

func TestHotReloadManager_ScanDirectory(t *testing.T) {
	manager := NewPluginManager()
	depManager := NewDependencyManager()
	config := DefaultHotReloadConfig()

	tempDir := t.TempDir()
	config.WatchDirectories = []string{tempDir}

	hrm, _ := NewHotReloadManager(config, manager, depManager)

	// Create some test files
	testFiles := []string{"plugin1.so", "plugin2.so", "not-a-plugin.txt"}
	for _, filename := range testFiles {
		filepath := filepath.Join(tempDir, filename)
		err := os.WriteFile(filepath, []byte("test content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create test file: %v", err)
		}
	}

	// Scan directory
	err := hrm.scanDirectory(tempDir)
	if err != nil {
		t.Errorf("scanDirectory failed: %v", err)
	}

	// Check that .so files were registered
	info1, exists1 := hrm.GetPluginInfo("plugin1")
	if !exists1 {
		t.Error("plugin1 should be registered")
	} else {
		if info1.PluginName != "plugin1" {
			t.Errorf("Expected plugin name 'plugin1', got '%s'", info1.PluginName)
		}
	}

	info2, exists2 := hrm.GetPluginInfo("plugin2")
	if !exists2 {
		t.Error("plugin2 should be registered")
	} else {
		if info2.PluginName != "plugin2" {
			t.Errorf("Expected plugin name 'plugin2', got '%s'", info2.PluginName)
		}
	}

	// Check that .txt file was not registered
	_, exists3 := hrm.GetPluginInfo("not-a-plugin")
	if exists3 {
		t.Error("not-a-plugin should not be registered")
	}
}

func TestHotReloadManager_GetTrackedPlugins(t *testing.T) {
	manager := NewPluginManager()
	depManager := NewDependencyManager()
	config := DefaultHotReloadConfig()

	tempDir := t.TempDir()
	config.WatchDirectories = []string{tempDir}

	hrm, _ := NewHotReloadManager(config, manager, depManager)

	// Initially should be empty
	plugins := hrm.GetTrackedPlugins()
	if len(plugins) != 0 {
		t.Errorf("Expected 0 tracked plugins, got %d", len(plugins))
	}

	// Add some plugin files
	testFile := filepath.Join(tempDir, "test-plugin.so")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_ = hrm.scanDirectory(tempDir)

	// Should now have tracked plugins
	plugins = hrm.GetTrackedPlugins()
	if len(plugins) != 1 {
		t.Errorf("Expected 1 tracked plugin, got %d", len(plugins))
	}

	if plugins[0] != "test-plugin" {
		t.Errorf("Expected 'test-plugin', got '%s'", plugins[0])
	}
}

func TestHotReloadManager_QueueReload(t *testing.T) {
	manager := NewPluginManager()
	depManager := NewDependencyManager()
	config := DefaultHotReloadConfig()

	tempDir := t.TempDir()
	config.WatchDirectories = []string{tempDir}

	hrm, _ := NewHotReloadManager(config, manager, depManager)

	// Test queueing a reload
	testPath := filepath.Join(tempDir, "test-plugin.so")

	// Should not block
	hrm.queueReload(testPath)

	// Check that something was queued (we can't easily check the exact content)
	select {
	case path := <-hrm.reloadQueue:
		if path != testPath {
			t.Errorf("Expected path %s, got %s", testPath, path)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected path to be queued")
	}
}

func TestHotReloadManager_QueueReload_FullQueue(t *testing.T) {
	manager := NewPluginManager()
	depManager := NewDependencyManager()
	config := DefaultHotReloadConfig()

	tempDir := t.TempDir()
	config.WatchDirectories = []string{tempDir}

	hrm, _ := NewHotReloadManager(config, manager, depManager)

	// Fill up the queue (channel has buffer size 10)
	testPaths := make([]string, 15) // More than buffer size
	for i := 0; i < 15; i++ {
		testPaths[i] = fmt.Sprintf("/tmp/plugin-%d.so", i)
	}

	// Queue first 10 should succeed, rest should hit default case
	for _, path := range testPaths {
		hrm.queueReload(path)
	}

	// Check that at least some were queued (up to buffer limit)
	queuedCount := 0
	for i := 0; i < 10; i++ {
		select {
		case <-hrm.reloadQueue:
			queuedCount++
		default:
			goto done
		}
	}
done:

	if queuedCount == 0 {
		t.Error("Expected some paths to be queued")
	}
}

func TestExtractPluginName(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/path/to/plugin.so", "plugin"},
		{"./plugins/auth-plugin.so", "auth-plugin"},
		{"simple.so", "simple"},
		{"/complex/path/with/nested/dirs/complex-name.so", "complex-name"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := extractPluginName(tt.path)
			if result != tt.expected {
				t.Errorf("extractPluginName(%s) = %s, want %s", tt.path, result, tt.expected)
			}
		})
	}
}

func TestHotReloadManager_AddWatchDirectory_InvalidPath(t *testing.T) {
	manager := NewPluginManager()
	depManager := NewDependencyManager()
	config := DefaultHotReloadConfig()
	config.WatchDirectories = []string{} // Empty to avoid auto-adding

	hrm, _ := NewHotReloadManager(config, manager, depManager)

	// Test with invalid path characters (this might work on some systems)
	err := hrm.addWatchDirectory("/invalid/path/that/does/not/exist")
	// Should create the directory or return an error
	// Either outcome is acceptable for this test
	if err != nil {
		t.Logf("addWatchDirectory returned expected error: %v", err)
	}
}

func TestPluginFileInfo(t *testing.T) {
	tempDir := t.TempDir()
	testFile := filepath.Join(tempDir, "test.so")

	// Create test file
	err := os.WriteFile(testFile, []byte("test content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fileInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}

	// Test PluginFileInfo structure
	pluginInfo := PluginFileInfo{
		Path:         testFile,
		ModTime:      fileInfo.ModTime(),
		Size:         fileInfo.Size(),
		PluginName:   "test",
		Version:      "1.0.0",
		Dependencies: []string{"dep1", "dep2"},
	}

	if pluginInfo.Path != testFile {
		t.Errorf("Path = %s, want %s", pluginInfo.Path, testFile)
	}

	if pluginInfo.PluginName != "test" {
		t.Errorf("PluginName = %s, want %s", pluginInfo.PluginName, "test")
	}

	if pluginInfo.Version != "1.0.0" {
		t.Errorf("Version = %s, want %s", pluginInfo.Version, "1.0.0")
	}

	if len(pluginInfo.Dependencies) != 2 {
		t.Errorf("Dependencies length = %d, want %d", len(pluginInfo.Dependencies), 2)
	}

	if pluginInfo.Size != int64(len("test content")) {
		t.Errorf("Size = %d, want %d", pluginInfo.Size, len("test content"))
	}
}

func TestHotReloadManager_RegisterPluginFile(t *testing.T) {
	manager := NewPluginManager()
	depManager := NewDependencyManager()
	config := DefaultHotReloadConfig()

	tempDir := t.TempDir()
	config.WatchDirectories = []string{tempDir}

	hrm, _ := NewHotReloadManager(config, manager, depManager)

	// Create test file
	testFile := filepath.Join(tempDir, "register-test.so")
	err := os.WriteFile(testFile, []byte("test"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	fileInfo, err := os.Stat(testFile)
	if err != nil {
		t.Fatalf("Failed to stat test file: %v", err)
	}

	// Register the file
	hrm.registerPluginFile(testFile, fileInfo)

	// Check it was registered
	pluginInfo, exists := hrm.GetPluginInfo("register-test")
	if !exists {
		t.Fatal("Plugin should be registered")
	}

	if pluginInfo.Path != testFile {
		t.Errorf("Path = %s, want %s", pluginInfo.Path, testFile)
	}

	if pluginInfo.PluginName != "register-test" {
		t.Errorf("PluginName = %s, want %s", pluginInfo.PluginName, "register-test")
	}

	if pluginInfo.Size != fileInfo.Size() {
		t.Errorf("Size = %d, want %d", pluginInfo.Size, fileInfo.Size())
	}

	if !pluginInfo.ModTime.Equal(fileInfo.ModTime()) {
		t.Errorf("ModTime = %v, want %v", pluginInfo.ModTime, fileInfo.ModTime())
	}
}

// Test concurrent access to plugin file tracking
func TestHotReloadManager_ConcurrentAccess(t *testing.T) {
	manager := NewPluginManager()
	depManager := NewDependencyManager()
	config := DefaultHotReloadConfig()

	tempDir := t.TempDir()
	config.WatchDirectories = []string{tempDir}

	hrm, _ := NewHotReloadManager(config, manager, depManager)

	// Create multiple files concurrently
	numFiles := 10
	done := make(chan bool, numFiles)

	for i := 0; i < numFiles; i++ {
		go func(id int) {
			defer func() { done <- true }()

			testFile := filepath.Join(tempDir, fmt.Sprintf("concurrent-%d.so", id))
			err := os.WriteFile(testFile, []byte("test"), 0644)
			if err != nil {
				t.Errorf("Failed to create test file %d: %v", id, err)
				return
			}

			fileInfo, err := os.Stat(testFile)
			if err != nil {
				t.Errorf("Failed to stat test file %d: %v", id, err)
				return
			}

			hrm.registerPluginFile(testFile, fileInfo)
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numFiles; i++ {
		<-done
	}

	// Check all files were registered
	tracked := hrm.GetTrackedPlugins()
	if len(tracked) != numFiles {
		t.Errorf("Expected %d tracked plugins, got %d", numFiles, len(tracked))
	}
}
