package plugin

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestParseVersion_ConfigMigration(t *testing.T) {
	migrator := NewConfigMigrator(3, "")

	tests := []struct {
		name        string
		config      map[string]interface{}
		expected    int
		expectError bool
	}{
		{
			name:     "missing version defaults to 0",
			config:   map[string]interface{}{},
			expected: 0,
		},
		{
			name: "integer version",
			config: map[string]interface{}{
				"version": 2,
			},
			expected: 2,
		},
		{
			name: "float version",
			config: map[string]interface{}{
				"version": 1.0,
			},
			expected: 1,
		},
		{
			name: "string version",
			config: map[string]interface{}{
				"version": "3",
			},
			expected: 3,
		},
		{
			name: "invalid string version",
			config: map[string]interface{}{
				"version": "invalid",
			},
			expectError: true,
		},
		{
			name: "unsupported version type",
			config: map[string]interface{}{
				"version": []string{"1", "2"},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := migrator.getConfigVersion(tt.config)
			if tt.expectError {
				if err == nil {
					t.Error("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if version != tt.expected {
				t.Errorf("Expected version %d, got %d", tt.expected, version)
			}
		})
	}
}

func TestConfigMigration_RegisterAndMigrate(t *testing.T) {
	migrator := NewConfigMigrator(2, "")

	// Register migrations
	migration1 := ConfigMigration{
		FromVersion: 0,
		ToVersion:   1,
		Description: "Add default settings",
		Migrate: func(config map[string]interface{}) (map[string]interface{}, error) {
			config["settings"] = map[string]interface{}{
				"enabled": true,
			}
			return config, nil
		},
	}

	migration2 := ConfigMigration{
		FromVersion: 1,
		ToVersion:   2,
		Description: "Add security settings",
		Migrate: func(config map[string]interface{}) (map[string]interface{}, error) {
			config["security"] = map[string]interface{}{
				"sandbox": true,
			}
			return config, nil
		},
	}

	migrator.RegisterMigration(migration1)
	migrator.RegisterMigration(migration2)

	// Test migration from version 0 to 2
	originalConfig := map[string]interface{}{
		"name": "test-plugin",
	}

	migratedConfig, err := migrator.MigrateConfig(originalConfig)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Check version was updated
	if migratedConfig["version"] != 2 {
		t.Errorf("Expected version 2, got %v", migratedConfig["version"])
	}

	// Check migration 1 was applied
	settings, ok := migratedConfig["settings"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected settings to be added")
	}

	if settings["enabled"] != true {
		t.Error("Expected settings.enabled to be true")
	}

	// Check migration 2 was applied
	security, ok := migratedConfig["security"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected security to be added")
	}

	if security["sandbox"] != true {
		t.Error("Expected security.sandbox to be true")
	}
}

func TestConfigMigration_NoMigrationNeeded(t *testing.T) {
	migrator := NewConfigMigrator(2, "")

	config := map[string]interface{}{
		"version": 2,
		"name":    "test",
	}

	result, err := migrator.MigrateConfig(config)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	// Should return original config unchanged
	if !reflect.DeepEqual(result, config) {
		t.Error("Config should remain unchanged when no migration is needed")
	}
}

func TestConfigMigration_FutureVersion(t *testing.T) {
	migrator := NewConfigMigrator(2, "")

	config := map[string]interface{}{
		"version": 5, // Future version
		"name":    "test",
	}

	_, err := migrator.MigrateConfig(config)
	if err == nil {
		t.Error("Expected error for future version")
	}
}

func TestConfigMigration_MissingMigrationPath(t *testing.T) {
	migrator := NewConfigMigrator(3, "")

	// Only register migration from 0 to 1, but current version is 3
	migration := ConfigMigration{
		FromVersion: 0,
		ToVersion:   1,
		Migrate: func(config map[string]interface{}) (map[string]interface{}, error) {
			return config, nil
		},
	}
	migrator.RegisterMigration(migration)

	config := map[string]interface{}{
		"version": 0,
		"name":    "test",
	}

	_, err := migrator.MigrateConfig(config)
	if err == nil {
		t.Error("Expected error for missing migration path")
	}
}

func TestConfigMigration_FileOperations(t *testing.T) {
	tempDir := t.TempDir()
	migrator := NewConfigMigrator(1, tempDir)

	// Register a simple migration
	migration := ConfigMigration{
		FromVersion: 0,
		ToVersion:   1,
		Migrate: func(config map[string]interface{}) (map[string]interface{}, error) {
			config["migrated"] = true
			return config, nil
		},
	}
	migrator.RegisterMigration(migration)

	configPath := filepath.Join(tempDir, "test-config.json")

	// Create config file with version 0 manually (bypass SaveConfig which sets currentVersion)
	originalConfig := map[string]interface{}{
		"name":    "test",
		"version": 0, // Explicit version 0 to trigger migration
	}

	// Manually write the JSON file with version 0
	data, err := json.MarshalIndent(originalConfig, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal config: %v", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	// Load and migrate config
	migratedConfig, err := migrator.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Check migration was applied
	if migratedConfig["migrated"] != true {
		t.Error("Expected migration to be applied")
	}

	if migratedConfig["version"] != 1 {
		t.Errorf("Expected version 1, got %v", migratedConfig["version"])
	}
}

func TestConfigMigration_BackupCreation(t *testing.T) {
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	migrator := NewConfigMigrator(1, backupDir)

	// Register migration
	migration := ConfigMigration{
		FromVersion: 0,
		ToVersion:   1,
		Migrate: func(config map[string]interface{}) (map[string]interface{}, error) {
			config["new_field"] = "added"
			return config, nil
		},
	}
	migrator.RegisterMigration(migration)

	config := map[string]interface{}{
		"version": 0,
		"name":    "test",
	}

	_, err := migrator.MigrateConfig(config)
	if err != nil {
		t.Fatalf("Migration failed: %v", err)
	}

	// Check backup was created
	backupFiles, err := os.ReadDir(backupDir)
	if err != nil {
		t.Fatalf("Failed to read backup directory: %v", err)
	}

	if len(backupFiles) != 1 {
		t.Errorf("Expected 1 backup file, got %d", len(backupFiles))
	}
}

func TestConfigMigration_BackupCreation_ErrorCases(t *testing.T) {
	// Test backup creation with various error cases
	tempDir := t.TempDir()
	backupDir := filepath.Join(tempDir, "backups")
	migrator := NewConfigMigrator(1, backupDir)

	// Test with invalid backup directory (read-only parent)
	readOnlyDir := filepath.Join(tempDir, "readonly")
	err := os.Mkdir(readOnlyDir, 0444) // read-only
	if err != nil {
		t.Fatalf("Failed to create read-only dir: %v", err)
	}

	invalidBackupDir := filepath.Join(readOnlyDir, "backups")
	migrator.backupDir = invalidBackupDir

	config := map[string]interface{}{
		"version": 0,
		"name":    "test",
	}

	// This should try to create backup but may fail due to permissions
	// The exact behavior depends on the OS, so we just ensure it doesn't panic
	result, err := migrator.MigrateConfig(config)
	// Either succeeds or fails gracefully
	if err != nil {
		t.Logf("Expected permission error during backup creation: %v", err)
	}
	if result != nil {
		t.Logf("Migration completed despite backup issues")
	}
}

func TestConfigMigration_SaveConfig_ErrorCases(t *testing.T) {
	tempDir := t.TempDir()
	migrator := NewConfigMigrator(1, tempDir)

	// Test saving to invalid path
	var invalidPath string
	if runtime.GOOS == "windows" {
		// Use an invalid path with illegal characters on Windows
		invalidPath = filepath.Join(tempDir, "test<>:|*.json")
	} else {
		invalidPath = "/proc/version" // Should be read-only on Unix systems
	}
	config := map[string]interface{}{
		"name": "test",
	}

	err := migrator.SaveConfig(config, invalidPath)
	if err == nil {
		t.Error("Expected error when saving to invalid path")
	}

	// Test saving with invalid JSON content (this shouldn't happen with map[string]interface{} but test anyway)
	validPath := filepath.Join(tempDir, "test.json")
	invalidConfig := map[string]interface{}{
		"invalid": make(chan int), // channels cannot be marshaled to JSON
	}

	err = migrator.SaveConfig(invalidConfig, validPath)
	if err == nil {
		t.Error("Expected error when marshaling invalid content")
	}
}

func TestConfigMigration_ValidateConfig(t *testing.T) {
	migrator := NewConfigMigrator(2, "")

	tests := []struct {
		name    string
		config  map[string]interface{}
		wantErr bool
	}{
		{
			name: "valid config",
			config: map[string]interface{}{
				"version": 2,
				"name":    "test",
			},
			wantErr: false,
		},
		{
			name: "wrong version",
			config: map[string]interface{}{
				"version": 1,
				"name":    "test",
			},
			wantErr: true,
		},
		{
			name: "missing version",
			config: map[string]interface{}{
				"name": "test",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := migrator.ValidateConfig(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestDefaultMigrations(t *testing.T) {
	migrations := DefaultMigrations()

	if len(migrations) == 0 {
		t.Error("Expected default migrations to be provided")
	}

	// Test that default migrations can be applied
	migrator := NewConfigMigrator(3, "")
	for _, migration := range migrations {
		migrator.RegisterMigration(migration)
	}

	// Test migration from version 0
	config := map[string]interface{}{
		"name": "test-plugin",
	}

	result, err := migrator.MigrateConfig(config)
	if err != nil {
		t.Fatalf("Default migrations failed: %v", err)
	}

	if result["version"] != 3 {
		t.Errorf("Expected version 3 after default migrations, got %v", result["version"])
	}

	// Check specific migration results
	plugins, ok := result["plugins"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected plugins configuration to be added")
	}

	system, ok := plugins["system"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected plugins.system configuration to be restructured")
	}

	if system["enabled"] != true {
		t.Error("Expected plugins.system.enabled to be true")
	}
}

func TestPluginConfigManager(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewPluginConfigManager(tempDir, 2)

	t.Run("LoadNonExistentConfig", func(t *testing.T) {
		config, err := manager.LoadPluginConfig("non-existent")
		if err != nil {
			t.Errorf("Loading non-existent config should return default: %v", err)
		}

		if config["name"] != "non-existent" {
			t.Error("Default config should have correct name")
		}

		if config["enabled"] != true {
			t.Error("Default config should be enabled")
		}
	})

	t.Run("SaveAndLoadConfig", func(t *testing.T) {
		config := map[string]interface{}{
			"name":    "test-plugin",
			"version": 2,
			"enabled": true,
			"settings": map[string]interface{}{
				"timeout": 30,
			},
		}

		err := manager.SavePluginConfig("test-plugin", config)
		if err != nil {
			t.Fatalf("Failed to save config: %v", err)
		}

		loadedConfig, err := manager.LoadPluginConfig("test-plugin")
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		if loadedConfig["name"] != "test-plugin" {
			t.Error("Loaded config should match saved config")
		}

		settings, ok := loadedConfig["settings"].(map[string]interface{})
		if !ok {
			t.Fatal("Expected settings in loaded config")
		}

		if settings["timeout"] != 30 {
			t.Error("Settings should be preserved")
		}
	})

	t.Run("ReloadAllConfigs", func(t *testing.T) {
		// Save multiple configs
		for i := 0; i < 3; i++ {
			config := map[string]interface{}{
				"name":    fmt.Sprintf("plugin-%d", i),
				"version": 2,
				"enabled": true,
			}
			_ = manager.SavePluginConfig(fmt.Sprintf("plugin-%d", i), config)
		}

		// Clear cache by creating new manager
		manager2 := NewPluginConfigManager(tempDir, 2)

		err := manager2.ReloadAllConfigs()
		if err != nil {
			t.Fatalf("Failed to reload configs: %v", err)
		}

		// Check that configs are loaded
		for i := 0; i < 3; i++ {
			config, err := manager2.LoadPluginConfig(fmt.Sprintf("plugin-%d", i))
			if err != nil {
				t.Errorf("Failed to load plugin-%d config: %v", i, err)
			}

			if config["name"] != fmt.Sprintf("plugin-%d", i) {
				t.Errorf("Unexpected config name for plugin-%d", i)
			}
		}
	})
}

func TestPluginConfigManager_ReloadAllConfigs_ErrorCases(t *testing.T) {
	tempDir := t.TempDir()
	manager := NewPluginConfigManager(tempDir, 2)

	// Create invalid JSON files to test error handling during reload
	invalidFiles := []string{"invalid1.json", "invalid2.json"}
	for _, filename := range invalidFiles {
		path := filepath.Join(tempDir, filename)
		// Write invalid JSON content
		err := os.WriteFile(path, []byte("{ invalid json content"), 0644)
		if err != nil {
			t.Fatalf("Failed to create invalid JSON file: %v", err)
		}
	}

	// Create valid but unmarshalable files
	weirdPath := filepath.Join(tempDir, "weird.json")
	err := os.WriteFile(weirdPath, []byte("\"just a string, not an object\""), 0644)
	if err != nil {
		t.Fatalf("Failed to create weird JSON file: %v", err)
	}

	// ReloadAllConfigs should handle errors gracefully
	err = manager.ReloadAllConfigs()
	// Should complete without crashing, but may log errors
	if err != nil {
		t.Logf("ReloadAllConfigs reported errors (expected): %v", err)
	}
}

func TestPluginConfigManager_Migration(t *testing.T) {
	tempDir := t.TempDir()

	// Create config file with old version
	configPath := filepath.Join(tempDir, "legacy-plugin.json")
	oldConfig := `{
		"version": 1,
		"name": "legacy-plugin",
		"enabled": true,
		"plugins": {
			"enabled": true,
			"directory": "./plugins",
			"auto_load": true
		}
	}`

	err := os.WriteFile(configPath, []byte(oldConfig), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	// Load with config manager (should auto-migrate)
	manager := NewPluginConfigManager(tempDir, 3)
	config, err := manager.LoadPluginConfig("legacy-plugin")
	if err != nil {
		t.Fatalf("Failed to load and migrate config: %v", err)
	}

	// Check migration was applied
	if config["version"] != 3 {
		t.Errorf("Expected version 3 after migration, got %v", config["version"])
	}

	// Check restructured plugins configuration
	plugins, ok := config["plugins"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected plugins configuration")
	}

	system, ok := plugins["system"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected restructured plugins.system configuration")
	}

	if system["enabled"] != true {
		t.Error("Expected plugins.system.enabled to be migrated")
	}
}
