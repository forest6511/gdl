package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig() should not return nil")
	}

	if config.Version != "1.0" {
		t.Errorf("Expected version '1.0', got %s", config.Version)
	}

	if config.RetryPolicy.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", config.RetryPolicy.MaxRetries)
	}

	if config.RetryPolicy.BaseDelay != 1*time.Second {
		t.Errorf("Expected BaseDelay 1s, got %v", config.RetryPolicy.BaseDelay)
	}

	if config.RetryPolicy.Strategy != "exponential" {
		t.Errorf("Expected Strategy 'exponential', got %s", config.RetryPolicy.Strategy)
	}

	if !config.RetryPolicy.Jitter {
		t.Error("Expected Jitter to be true")
	}

	if config.ErrorHandling.ErrorFormat != "text" {
		t.Errorf("Expected ErrorFormat 'text', got %s", config.ErrorHandling.ErrorFormat)
	}

	if !config.ErrorHandling.RecoveryEnabled {
		t.Error("Expected RecoveryEnabled to be true")
	}

	if config.OutputFormat.Format != "text" {
		t.Errorf("Expected Format 'text', got %s", config.OutputFormat.Format)
	}

	if !config.OutputFormat.Color {
		t.Error("Expected Color to be true")
	}

	if config.Timeouts.ConnectTimeout != 10*time.Second {
		t.Errorf("Expected ConnectTimeout 10s, got %v", config.Timeouts.ConnectTimeout)
	}

	if config.Network.UserAgent != "godl/1.0" {
		t.Errorf("Expected UserAgent 'godl/1.0', got %s", config.Network.UserAgent)
	}

	if config.Network.MaxConcurrentDownloads != 4 {
		t.Errorf("Expected MaxConcurrentDownloads 4, got %d", config.Network.MaxConcurrentDownloads)
	}

	if config.Storage.MinFreeSpace != 100*1024*1024 {
		t.Errorf("Expected MinFreeSpace 100MB, got %d", config.Storage.MinFreeSpace)
	}
}

func TestNewConfigLoader(t *testing.T) {
	configPath := "/tmp/test_config.json"
	loader := NewConfigLoader(configPath)

	if loader == nil {
		t.Fatal("NewConfigLoader() should not return nil")
	}

	if loader.configPath != configPath {
		t.Errorf("Expected configPath %s, got %s", configPath, loader.configPath)
	}
}

func TestDefaultConfigPath(t *testing.T) {
	path, err := DefaultConfigPath()
	if err != nil {
		t.Fatalf("DefaultConfigPath() failed: %v", err)
	}

	if path == "" {
		t.Error("DefaultConfigPath() should not return empty string")
	}

	if !filepath.IsAbs(path) {
		t.Error("DefaultConfigPath() should return absolute path")
	}
}

func TestConfigLoader_LoadNonExistent(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "nonexistent.json")
	loader := NewConfigLoader(configPath)

	// Should return default config when file doesn't exist
	config, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() should not fail for non-existent file: %v", err)
	}

	if config == nil {
		t.Fatal("Load() should return default config")
	}

	if config.Version != "1.0" {
		t.Error("Should return default config values")
	}
}

func TestConfigLoader_SaveAndLoad(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "test_config.json")
	loader := NewConfigLoader(configPath)

	// Create test config
	originalConfig := DefaultConfig()
	originalConfig.RetryPolicy.MaxRetries = 10
	originalConfig.ErrorHandling.VerboseErrors = true

	// Save config
	err = loader.Save(originalConfig)
	if err != nil {
		t.Fatalf("Save() failed: %v", err)
	}

	// Load config
	loadedConfig, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	if loadedConfig.RetryPolicy.MaxRetries != 10 {
		t.Errorf("Expected MaxRetries 10, got %d", loadedConfig.RetryPolicy.MaxRetries)
	}

	if !loadedConfig.ErrorHandling.VerboseErrors {
		t.Error("Expected VerboseErrors to be true")
	}
}

func TestConfig_Validate(t *testing.T) {
	config := DefaultConfig()

	// Valid config should pass
	err := config.Validate()
	if err != nil {
		t.Errorf("Valid config should pass validation: %v", err)
	}

	// Test invalid retry policy
	config.RetryPolicy.MaxRetries = -1

	err = config.Validate()
	if err == nil {
		t.Error("Should fail validation with negative MaxRetries")
	}

	// Reset and test invalid base delay
	config = DefaultConfig()
	config.RetryPolicy.BaseDelay = -1

	err = config.Validate()
	if err == nil {
		t.Error("Should fail validation with negative BaseDelay")
	}

	// Test invalid strategy
	config = DefaultConfig()
	config.RetryPolicy.Strategy = "invalid"

	err = config.Validate()
	if err == nil {
		t.Error("Should fail validation with invalid strategy")
	}

	// Test invalid error format
	config = DefaultConfig()
	config.ErrorHandling.ErrorFormat = "invalid"

	err = config.Validate()
	if err == nil {
		t.Error("Should fail validation with invalid error format")
	}

	// Test invalid output format
	config = DefaultConfig()
	config.OutputFormat.Format = "invalid"

	err = config.Validate()
	if err == nil {
		t.Error("Should fail validation with invalid output format")
	}

	// Test invalid log level
	config = DefaultConfig()
	config.OutputFormat.LogLevel = "invalid"

	err = config.Validate()
	if err == nil {
		t.Error("Should fail validation with invalid log level")
	}

	// Test invalid timeout
	config = DefaultConfig()
	config.Timeouts.ConnectTimeout = -1

	err = config.Validate()
	if err == nil {
		t.Error("Should fail validation with negative ConnectTimeout")
	}

	// Test invalid network settings
	config = DefaultConfig()
	config.Network.MaxConcurrentDownloads = -1

	err = config.Validate()
	if err == nil {
		t.Error("Should fail validation with negative MaxConcurrentDownloads")
	}
}

func TestConfig_Clone(t *testing.T) {
	original := DefaultConfig()
	original.RetryPolicy.MaxRetries = 5

	clone := original.Clone()

	if clone == original {
		t.Error("Clone should return a different instance")
	}

	if clone.RetryPolicy.MaxRetries != 5 {
		t.Error("Clone should have same values")
	}

	// Modify clone to ensure independence
	clone.RetryPolicy.MaxRetries = 10
	if original.RetryPolicy.MaxRetries == 10 {
		t.Error("Modifying clone should not affect original")
	}
}

func TestConfig_Merge(t *testing.T) {
	base := DefaultConfig()
	base.RetryPolicy.MaxRetries = 3
	base.ErrorHandling.VerboseErrors = false

	override := DefaultConfig()
	override.RetryPolicy.MaxRetries = 10
	override.ErrorHandling.VerboseErrors = true
	override.OutputFormat.Color = false

	base.Merge(override)

	if base.RetryPolicy.MaxRetries != 10 {
		t.Errorf("Expected merged MaxRetries 10, got %d", base.RetryPolicy.MaxRetries)
	}

	if !base.ErrorHandling.VerboseErrors {
		t.Error("Expected merged VerboseErrors to be true")
	}

	if base.OutputFormat.Color {
		t.Error("Expected merged Color to be false")
	}
}

func TestConfigManager_NewConfigManager(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_manager_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "test_config.json")

	manager, err := NewConfigManager(configPath)
	if err != nil {
		t.Fatalf("NewConfigManager() failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewConfigManager() should not return nil")
	}

	config := manager.GetConfig()
	if config == nil {
		t.Error("GetConfig() should not return nil")
	}
}

func TestConfigManager_UpdateConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_manager_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "test_config.json")

	manager, err := NewConfigManager(configPath)
	if err != nil {
		t.Fatalf("NewConfigManager() failed: %v", err)
	}

	// Update config
	newConfig := DefaultConfig()
	newConfig.RetryPolicy.MaxRetries = 15

	err = manager.UpdateConfig(newConfig)
	if err != nil {
		t.Fatalf("UpdateConfig() failed: %v", err)
	}

	// Verify update
	currentConfig := manager.GetConfig()
	if currentConfig.RetryPolicy.MaxRetries != 15 {
		t.Errorf("Expected MaxRetries 15, got %d", currentConfig.RetryPolicy.MaxRetries)
	}
}

func TestConfigManager_InvalidConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_manager_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "test_config.json")

	manager, err := NewConfigManager(configPath)
	if err != nil {
		t.Fatalf("NewConfigManager() failed: %v", err)
	}

	// Try to update with invalid config
	invalidConfig := DefaultConfig()
	invalidConfig.RetryPolicy.MaxRetries = -5

	err = manager.UpdateConfig(invalidConfig)
	if err == nil {
		t.Error("UpdateConfig() should fail with invalid config")
	}
}

func TestConfigManager_SaveConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_manager_save_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "test_config.json")

	manager, err := NewConfigManager(configPath)
	if err != nil {
		t.Fatalf("NewConfigManager() failed: %v", err)
	}

	// Update config
	newConfig := DefaultConfig()
	newConfig.RetryPolicy.MaxRetries = 20
	newConfig.ErrorHandling.VerboseErrors = true

	err = manager.UpdateConfig(newConfig)
	if err != nil {
		t.Fatalf("UpdateConfig() failed: %v", err)
	}

	// Save config
	err = manager.SaveConfig()
	if err != nil {
		t.Fatalf("SaveConfig() failed: %v", err)
	}

	// Verify file was written
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Error("Config file was not saved")
	}

	// Create a new manager and verify config was persisted
	newManager, err := NewConfigManager(configPath)
	if err != nil {
		t.Fatalf("NewConfigManager() failed: %v", err)
	}

	loadedConfig := newManager.GetConfig()
	if loadedConfig.RetryPolicy.MaxRetries != 20 {
		t.Errorf("Expected MaxRetries 20, got %d", loadedConfig.RetryPolicy.MaxRetries)
	}

	if !loadedConfig.ErrorHandling.VerboseErrors {
		t.Error("Expected VerboseErrors to be true")
	}
}

func TestConfigManager_ReloadConfig(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "config_manager_reload_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "test_config.json")

	// Create initial config
	initialConfig := DefaultConfig()
	initialConfig.RetryPolicy.MaxRetries = 5
	loader := NewConfigLoader(configPath)

	err = loader.Save(initialConfig)
	if err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Create manager
	manager, err := NewConfigManager(configPath)
	if err != nil {
		t.Fatalf("NewConfigManager() failed: %v", err)
	}

	// Verify initial config
	if manager.GetConfig().RetryPolicy.MaxRetries != 5 {
		t.Errorf(
			"Expected initial MaxRetries 5, got %d",
			manager.GetConfig().RetryPolicy.MaxRetries,
		)
	}

	// Modify config file externally
	externalConfig := DefaultConfig()
	externalConfig.RetryPolicy.MaxRetries = 25
	externalConfig.OutputFormat.Color = false

	err = loader.Save(externalConfig)
	if err != nil {
		t.Fatalf("Failed to save external config: %v", err)
	}

	// Reload config
	err = manager.ReloadConfig()
	if err != nil {
		t.Fatalf("ReloadConfig() failed: %v", err)
	}

	// Verify reloaded config
	reloadedConfig := manager.GetConfig()
	if reloadedConfig.RetryPolicy.MaxRetries != 25 {
		t.Errorf("Expected reloaded MaxRetries 25, got %d", reloadedConfig.RetryPolicy.MaxRetries)
	}

	if reloadedConfig.OutputFormat.Color {
		t.Error("Expected Color to be false after reload")
	}
}

// Test NewDefaultConfigManager function (0% coverage)
func TestNewDefaultConfigManager(t *testing.T) {
	manager, err := NewDefaultConfigManager()
	if err != nil {
		t.Fatalf("NewDefaultConfigManager() failed: %v", err)
	}

	if manager == nil {
		t.Fatal("NewDefaultConfigManager() should not return nil")
	}

	config := manager.GetConfig()
	if config == nil {
		t.Error("GetConfig() should not return nil")
		return
	}

	// Verify it uses default values
	if config.Version != "1.0" {
		t.Errorf("Expected version '1.0', got %s", config.Version)
	}
}

// Test apply defaults functions (50% coverage)
func TestConfigLoader_ApplyDefaults(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "apply_defaults_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "partial_config.json")
	loader := NewConfigLoader(configPath)

	// Create a partial config with some fields missing
	partialConfig := &Config{
		RetryPolicy: RetryPolicyConfig{
			MaxRetries: 0,  // Will be filled by defaults
			Strategy:   "", // Will be filled by defaults
		},
		ErrorHandling: ErrorHandlingConfig{
			ErrorFormat: "", // Will be filled by defaults
		},
		OutputFormat: OutputFormatConfig{
			Format:          "", // Will be filled by defaults
			LogLevel:        "", // Will be filled by defaults
			TimestampFormat: "", // Will be filled by defaults
		},
		Timeouts: TimeoutConfig{
			ConnectTimeout: 0, // Will be filled by defaults
		},
		Network: NetworkConfig{
			UserAgent: "", // Will be filled by defaults
		},
		Storage: StorageConfig{
			DefaultDownloadDir: "", // Will be filled by defaults
		},
	}

	// Save partial config
	err = loader.Save(partialConfig)
	if err != nil {
		t.Fatalf("Failed to save partial config: %v", err)
	}

	// Load config (should apply defaults)
	loadedConfig, err := loader.Load()
	if err != nil {
		t.Fatalf("Load() failed: %v", err)
	}

	// Verify defaults were applied
	if loadedConfig.RetryPolicy.MaxRetries == 0 {
		t.Error("Expected MaxRetries to be filled with default value")
	}

	if loadedConfig.RetryPolicy.Strategy == "" {
		t.Error("Expected Strategy to be filled with default value")
	}

	if loadedConfig.ErrorHandling.ErrorFormat == "" {
		t.Error("Expected ErrorFormat to be filled with default value")
	}

	if loadedConfig.OutputFormat.Format == "" {
		t.Error("Expected Format to be filled with default value")
	}

	if loadedConfig.OutputFormat.LogLevel == "" {
		t.Error("Expected LogLevel to be filled with default value")
	}

	if loadedConfig.Timeouts.ConnectTimeout == 0 {
		t.Error("Expected ConnectTimeout to be filled with default value")
	}

	if loadedConfig.Network.UserAgent == "" {
		t.Error("Expected UserAgent to be filled with default value")
	}

	if loadedConfig.Storage.DefaultDownloadDir == "" {
		t.Error("Expected DefaultDownloadDir to be filled with default value")
	}

	// Verify version was applied
	if loadedConfig.Version == "" {
		t.Error("Expected Version to be filled with default value")
	}
}

// Test error scenarios
func TestConfigLoader_LoadInvalidJSON(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "invalid_json_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "invalid.json")

	// Write invalid JSON
	err = os.WriteFile(configPath, []byte("{ invalid json }"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid JSON: %v", err)
	}

	loader := NewConfigLoader(configPath)

	// Should return error for invalid JSON
	_, err = loader.Load()
	if err == nil {
		t.Error("Load() should fail with invalid JSON")
	}
}

func TestConfigLoader_SaveDirectoryPermissionError(t *testing.T) {
	// Try to save to a directory that doesn't exist and can't be created
	configPath := "/root/nonexistent/config.json"
	loader := NewConfigLoader(configPath)

	config := DefaultConfig()

	err := loader.Save(config)
	if err == nil {
		t.Error("Save() should fail when directory can't be created")
	}
}

func TestDefaultConfigPath_Error(t *testing.T) {
	// Test by temporarily unsetting HOME
	oldHome := os.Getenv("HOME")
	defer func() { _ = os.Setenv("HOME", oldHome) }()

	// Unset HOME environment variable
	_ = os.Unsetenv("HOME")

	_, err := DefaultConfigPath()
	if err == nil {
		t.Error("DefaultConfigPath() should fail when HOME is not set")
	}
}

// Test validate functions with more edge cases
func TestConfig_ValidateEdgeCases(t *testing.T) {
	// Test timeout edge cases
	config := DefaultConfig()
	config.Timeouts.ReadTimeout = -1

	err := config.Validate()
	if err == nil {
		t.Error("Should fail validation with negative ReadTimeout")
	}

	// Test network edge cases
	config = DefaultConfig()
	config.Network.ChunkSize = -1

	err = config.Validate()
	if err == nil {
		t.Error("Should fail validation with negative ChunkSize")
	}

	// Test storage edge cases
	config = DefaultConfig()
	config.Storage.MinFreeSpace = -1

	err = config.Validate()
	if err == nil {
		t.Error("Should fail validation with negative MinFreeSpace")
	}
}

// Test ReloadConfig error scenarios
func TestConfigManager_ReloadConfigErrors(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "reload_error_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer func() { _ = os.RemoveAll(tempDir) }()

	configPath := filepath.Join(tempDir, "test_config.json")

	manager, err := NewConfigManager(configPath)
	if err != nil {
		t.Fatalf("NewConfigManager() failed: %v", err)
	}

	// Remove the config file to simulate file not found
	_ = os.Remove(configPath)

	// Write invalid config
	invalidConfigJSON := `{"retry_policy": {"max_retries": -1}}`
	err = os.WriteFile(configPath, []byte(invalidConfigJSON), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid config: %v", err)
	}

	// ReloadConfig should fail validation
	err = manager.ReloadConfig()
	if err == nil {
		t.Error("ReloadConfig() should fail with invalid config")
	}
}
