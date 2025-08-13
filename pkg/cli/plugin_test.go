package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/forest6511/godl/pkg/plugin"
)

// TestPluginRegistry tests the CLI plugin registry functionality
func TestPluginRegistry(t *testing.T) {
	// Create temporary directories for testing
	pluginDir := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "plugins.json")

	registry := NewPluginRegistry(pluginDir, configFile)

	t.Run("NewPluginRegistry", func(t *testing.T) {
		if registry == nil {
			t.Fatal("Expected registry to be created")
		}

		if registry.pluginDir != pluginDir {
			t.Errorf("Expected plugin dir %s, got: %s", pluginDir, registry.pluginDir)
		}

		if registry.configFile != configFile {
			t.Errorf("Expected config file %s, got: %s", configFile, registry.configFile)
		}

		if registry.pluginLoader == nil {
			t.Error("Expected plugin loader to be initialized")
		}
	})

	t.Run("ListPluginsEmpty", func(t *testing.T) {
		ctx := context.Background()
		plugins, err := registry.List(ctx)
		if err != nil {
			t.Errorf("Failed to list empty plugins: %v", err)
		}

		if len(plugins) != 0 {
			t.Errorf("Expected 0 plugins, got: %d", len(plugins))
		}
	})

	t.Run("GetEnabledPluginsEmpty", func(t *testing.T) {
		ctx := context.Background()
		plugins, err := registry.GetEnabledPlugins(ctx)
		if err != nil {
			t.Errorf("Failed to get empty enabled plugins: %v", err)
		}

		if len(plugins) != 0 {
			t.Errorf("Expected 0 enabled plugins, got: %d", len(plugins))
		}
	})
}

// TestPluginRegistryWithData tests registry operations with actual data
func TestPluginRegistryWithData(t *testing.T) {
	pluginDir := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "plugins.json")

	registry := NewPluginRegistry(pluginDir, configFile)

	// Create a mock plugin info entry by directly manipulating the config
	config := &PluginConfig{
		Plugins: map[string]*PluginInfo{
			"test-plugin": {
				Name:        "test-plugin",
				Version:     "1.0.0",
				Type:        "transform",
				Path:        "/test/path/plugin.so",
				Enabled:     true,
				Config:      map[string]string{"key": "value"},
				InstallTime: time.Now(),
				Source:      "test-source",
			},
			"disabled-plugin": {
				Name:        "disabled-plugin",
				Version:     "2.0.0",
				Type:        "auth",
				Path:        "/test/path/disabled.so",
				Enabled:     false,
				Config:      map[string]string{},
				InstallTime: time.Now(),
				Source:      "test-source",
			},
		},
	}

	// Save the config to test loading
	err := registry.saveConfig(config)
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	t.Run("ListPluginsWithData", func(t *testing.T) {
		ctx := context.Background()
		plugins, err := registry.List(ctx)
		if err != nil {
			t.Errorf("Failed to list plugins: %v", err)
		}

		if len(plugins) != 2 {
			t.Errorf("Expected 2 plugins, got: %d", len(plugins))
		}

		// Verify plugin names
		names := make(map[string]bool)
		for _, p := range plugins {
			names[p.Name] = true
		}

		if !names["test-plugin"] {
			t.Error("Expected to find test-plugin")
		}
		if !names["disabled-plugin"] {
			t.Error("Expected to find disabled-plugin")
		}
	})

	t.Run("GetEnabledPluginsWithData", func(t *testing.T) {
		ctx := context.Background()
		plugins, err := registry.GetEnabledPlugins(ctx)
		if err != nil {
			t.Errorf("Failed to get enabled plugins: %v", err)
		}

		if len(plugins) != 1 {
			t.Errorf("Expected 1 enabled plugin, got: %d", len(plugins))
		}

		if plugins[0].Name != "test-plugin" {
			t.Errorf("Expected enabled plugin to be test-plugin, got: %s", plugins[0].Name)
		}

		if !plugins[0].Enabled {
			t.Error("Plugin should be enabled")
		}
	})

	t.Run("EnablePlugin", func(t *testing.T) {
		ctx := context.Background()

		// Enable the disabled plugin
		err := registry.Enable(ctx, "disabled-plugin")
		if err != nil {
			t.Errorf("Failed to enable plugin: %v", err)
		}

		// Verify it's now enabled
		plugins, err := registry.GetEnabledPlugins(ctx)
		if err != nil {
			t.Errorf("Failed to get enabled plugins: %v", err)
		}

		if len(plugins) != 2 {
			t.Errorf("Expected 2 enabled plugins, got: %d", len(plugins))
		}
	})

	t.Run("DisablePlugin", func(t *testing.T) {
		ctx := context.Background()

		// Disable the test plugin
		err := registry.Disable(ctx, "test-plugin")
		if err != nil {
			t.Errorf("Failed to disable plugin: %v", err)
		}

		// Verify it's now disabled
		plugins, err := registry.GetEnabledPlugins(ctx)
		if err != nil {
			t.Errorf("Failed to get enabled plugins: %v", err)
		}

		// Should still have disabled-plugin enabled from previous test
		if len(plugins) != 1 {
			t.Errorf("Expected 1 enabled plugin, got: %d", len(plugins))
		}

		if plugins[0].Name != "disabled-plugin" {
			t.Errorf("Expected enabled plugin to be disabled-plugin, got: %s", plugins[0].Name)
		}
	})

	t.Run("ConfigurePlugin", func(t *testing.T) {
		ctx := context.Background()

		err := registry.Configure(ctx, "test-plugin", "new-key", "new-value")
		if err != nil {
			t.Errorf("Failed to configure plugin: %v", err)
		}

		// Verify configuration was saved
		plugins, err := registry.List(ctx)
		if err != nil {
			t.Errorf("Failed to list plugins: %v", err)
		}

		var testPlugin *PluginInfo
		for _, p := range plugins {
			if p.Name == "test-plugin" {
				testPlugin = p
				break
			}
		}

		if testPlugin == nil {
			t.Fatal("test-plugin not found")
		}

		if testPlugin.Config["new-key"] != "new-value" {
			t.Errorf("Expected config value 'new-value', got: %s", testPlugin.Config["new-key"])
		}

		// Original config should still be there
		if testPlugin.Config["key"] != "value" {
			t.Errorf("Expected original config value 'value', got: %s", testPlugin.Config["key"])
		}
	})

	t.Run("RemovePlugin", func(t *testing.T) {
		ctx := context.Background()

		// Remove the test plugin
		err := registry.Remove(ctx, "test-plugin")
		if err != nil {
			// This will fail because the plugin file doesn't exist, but that's expected
			// The plugin should still be removed from config
			if !strings.Contains(err.Error(), "no such file or directory") &&
				!strings.Contains(err.Error(), "cannot find the file") &&
				!strings.Contains(err.Error(), "system cannot find the file") {
				t.Errorf("Unexpected error removing plugin: %v", err)
			}
		}

		// Verify plugin was removed from config regardless of file deletion error
		plugins, err := registry.List(ctx)
		if err != nil {
			t.Errorf("Failed to list plugins: %v", err)
		}

		// Should only have disabled-plugin left
		if len(plugins) != 1 {
			t.Errorf("Expected 1 plugin after removal, got: %d", len(plugins))
		}

		if plugins[0].Name != "disabled-plugin" {
			t.Errorf("Expected remaining plugin to be disabled-plugin, got: %s", plugins[0].Name)
		}
	})
}

// TestPluginRegistryErrorCases tests error handling
func TestPluginRegistryErrorCases(t *testing.T) {
	pluginDir := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "plugins.json")

	registry := NewPluginRegistry(pluginDir, configFile)
	ctx := context.Background()

	t.Run("EnableNonExistentPlugin", func(t *testing.T) {
		err := registry.Enable(ctx, "non-existent-plugin")
		if err == nil {
			t.Error("Expected error when enabling non-existent plugin")
		}

		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	t.Run("DisableNonExistentPlugin", func(t *testing.T) {
		err := registry.Disable(ctx, "non-existent-plugin")
		if err == nil {
			t.Error("Expected error when disabling non-existent plugin")
		}

		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	t.Run("ConfigureNonExistentPlugin", func(t *testing.T) {
		err := registry.Configure(ctx, "non-existent-plugin", "key", "value")
		if err == nil {
			t.Error("Expected error when configuring non-existent plugin")
		}

		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})

	t.Run("RemoveNonExistentPlugin", func(t *testing.T) {
		err := registry.Remove(ctx, "non-existent-plugin")
		if err == nil {
			t.Error("Expected error when removing non-existent plugin")
		}

		if !strings.Contains(err.Error(), "not found") {
			t.Errorf("Expected 'not found' error, got: %v", err)
		}
	})
}

// TestInstallPlugin tests plugin installation functionality
func TestInstallPlugin(t *testing.T) {
	pluginDir := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "plugins.json")

	registry := NewPluginRegistry(pluginDir, configFile)
	ctx := context.Background()

	t.Run("InstallFromNonExistentLocalFile", func(t *testing.T) {
		err := registry.Install(ctx, "nonexistent.so", "test-plugin")
		if err == nil {
			t.Error("Expected error when installing from non-existent local file")
		}

		if !strings.Contains(err.Error(), "failed to open source file") {
			t.Errorf("Expected 'failed to open source file' error, got: %v", err)
		}
	})

	t.Run("InstallFromValidLocalFile", func(t *testing.T) {
		// Create a temporary source file
		sourceFile := filepath.Join(t.TempDir(), "source.so")
		err := os.WriteFile(sourceFile, []byte("mock plugin content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create source file: %v", err)
		}

		// This will fail at the plugin loading stage since it's not a real plugin
		err = registry.Install(ctx, sourceFile, "test-plugin")
		if err == nil {
			t.Error("Expected error when installing invalid plugin file")
		}

		// Should get a plugin loading error, not a file copy error
		if strings.Contains(err.Error(), "failed to open source file") {
			t.Errorf("Should not get source file error, got: %v", err)
		}
	})

	t.Run("InstallFromURL", func(t *testing.T) {
		err := registry.Install(ctx, "https://example.com/plugin.so", "url-plugin")
		if err == nil {
			t.Error("Expected error for URL download (not implemented)")
		}

		if !strings.Contains(err.Error(), "URL download not implemented yet") {
			t.Errorf("Expected URL download error, got: %v", err)
		}
	})

	t.Run("InstallFromGitHub", func(t *testing.T) {
		err := registry.Install(ctx, "user/repo", "github-plugin")
		if err == nil {
			t.Error("Expected error for GitHub download (not implemented)")
		}

		if !strings.Contains(err.Error(), "GitHub download not implemented yet") {
			t.Errorf("Expected GitHub download error, got: %v", err)
		}
	})
}

// TestLoadPlugins tests plugin loading into manager
func TestLoadPlugins(t *testing.T) {
	pluginDir := t.TempDir()
	configFile := filepath.Join(t.TempDir(), "plugins.json")

	registry := NewPluginRegistry(pluginDir, configFile)

	// Create a config with some plugins (non-existent files for testing)
	config := &PluginConfig{
		Plugins: map[string]*PluginInfo{
			"enabled-plugin": {
				Name:    "enabled-plugin",
				Version: "1.0.0",
				Path:    "/non/existent/enabled.so",
				Enabled: true,
				Config:  map[string]string{"test": "value"},
			},
			"disabled-plugin": {
				Name:    "disabled-plugin",
				Version: "1.0.0",
				Path:    "/non/existent/disabled.so",
				Enabled: false,
			},
		},
	}

	err := registry.saveConfig(config)
	if err != nil {
		t.Fatalf("Failed to save test config: %v", err)
	}

	t.Run("LoadPluginsIntoManager", func(t *testing.T) {
		ctx := context.Background()
		pluginManager := plugin.NewPluginManager()

		// This will attempt to load plugins but fail since they don't exist
		// However, it should not return an error - it should just print warnings
		err := registry.LoadPlugins(ctx, pluginManager)
		if err != nil {
			t.Errorf("LoadPlugins should not return error for missing files: %v", err)
		}

		// No plugins should be loaded since files don't exist
		loadedPlugins := pluginManager.ListPlugins()
		if len(loadedPlugins) != 0 {
			t.Errorf("Expected 0 loaded plugins, got: %d", len(loadedPlugins))
		}
	})
}

// TestConfigFileOperations tests config file loading and saving
func TestConfigFileOperations(t *testing.T) {
	configDir := t.TempDir()
	configFile := filepath.Join(configDir, "test-plugins.json")

	registry := NewPluginRegistry(t.TempDir(), configFile)

	t.Run("LoadNonExistentConfig", func(t *testing.T) {
		config, err := registry.loadConfig()
		if err != nil {
			t.Errorf("Loading non-existent config should not error: %v", err)
		}

		if config == nil {
			t.Error("Config should not be nil")
			return
		}

		if len(config.Plugins) != 0 {
			t.Errorf("Expected empty plugins map, got: %d plugins", len(config.Plugins))
		}
	})

	t.Run("SaveAndLoadConfig", func(t *testing.T) {
		originalConfig := &PluginConfig{
			Plugins: map[string]*PluginInfo{
				"test": {
					Name:        "test",
					Version:     "1.0.0",
					Type:        "transform",
					Path:        "/test/path",
					Enabled:     true,
					InstallTime: time.Now().Truncate(time.Second), // Truncate for comparison
				},
			},
		}

		err := registry.saveConfig(originalConfig)
		if err != nil {
			t.Errorf("Failed to save config: %v", err)
		}

		// Verify file was created
		if _, err := os.Stat(configFile); os.IsNotExist(err) {
			t.Error("Config file should have been created")
		}

		// Load and verify
		loadedConfig, err := registry.loadConfig()
		if err != nil {
			t.Errorf("Failed to load config: %v", err)
		}

		if len(loadedConfig.Plugins) != 1 {
			t.Errorf("Expected 1 plugin in loaded config, got: %d", len(loadedConfig.Plugins))
		}

		testPlugin := loadedConfig.Plugins["test"]
		if testPlugin == nil {
			t.Fatal("test plugin not found in loaded config")
		}

		if testPlugin.Name != "test" {
			t.Errorf("Expected name 'test', got: %s", testPlugin.Name)
		}

		if testPlugin.Version != "1.0.0" {
			t.Errorf("Expected version '1.0.0', got: %s", testPlugin.Version)
		}
	})

	t.Run("LoadEmptyConfig", func(t *testing.T) {
		// Create an empty file
		emptyConfigFile := filepath.Join(configDir, "empty-config.json")
		err := os.WriteFile(emptyConfigFile, []byte(""), 0644)
		if err != nil {
			t.Fatalf("Failed to create empty config file: %v", err)
		}

		emptyRegistry := NewPluginRegistry(t.TempDir(), emptyConfigFile)
		config, err := emptyRegistry.loadConfig()
		if err != nil {
			t.Errorf("Loading empty config should not error: %v", err)
		}

		if config == nil || config.Plugins == nil {
			t.Error("Config and plugins map should be initialized")
		}

		if len(config.Plugins) != 0 {
			t.Errorf("Expected empty plugins map, got: %d plugins", len(config.Plugins))
		}
	})
}

// TestDefaultPaths tests default path generation
func TestDefaultPaths(t *testing.T) {
	t.Run("GetDefaultPluginDir", func(t *testing.T) {
		dir := GetDefaultPluginDir()
		if len(dir) == 0 {
			t.Error("Default plugin dir should not be empty")
		}

		// Should contain either home directory path or fallback
		if !strings.Contains(dir, ".godl") && dir != "./plugins" {
			t.Errorf("Expected default plugin dir to contain .godl or be ./plugins, got: %s", dir)
		}
	})

	t.Run("GetDefaultConfigFile", func(t *testing.T) {
		file := GetDefaultConfigFile()
		if len(file) == 0 {
			t.Error("Default config file should not be empty")
		}

		// Should contain either home directory path or fallback
		if !strings.Contains(file, ".godl") && file != "./plugins.json" {
			t.Errorf("Expected default config file to contain .godl or be ./plugins.json, got: %s", file)
		}
	})
}

// Benchmark tests for performance
func BenchmarkPluginRegistryOperations(b *testing.B) {
	pluginDir := b.TempDir()
	configFile := filepath.Join(b.TempDir(), "bench-plugins.json")
	registry := NewPluginRegistry(pluginDir, configFile)

	// Create test config with many plugins
	config := &PluginConfig{
		Plugins: make(map[string]*PluginInfo),
	}

	for i := 0; i < 1000; i++ {
		pluginName := fmt.Sprintf("plugin-%d", i)
		config.Plugins[pluginName] = &PluginInfo{
			Name:        pluginName,
			Version:     "1.0.0",
			Type:        "transform",
			Path:        fmt.Sprintf("/test/%d.so", i),
			Enabled:     i%2 == 0, // Half enabled, half disabled
			InstallTime: time.Now(),
		}
	}

	_ = registry.saveConfig(config)
	ctx := context.Background()

	b.Run("List", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := registry.List(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("GetEnabledPlugins", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			_, err := registry.GetEnabledPlugins(ctx)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
