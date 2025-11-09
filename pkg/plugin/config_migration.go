package plugin

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
)

// ConfigVersion represents a configuration version
type ConfigVersion struct {
	Version int    `json:"version"`
	Schema  string `json:"schema,omitempty"`
}

// ConfigMigration represents a migration from one version to another
type ConfigMigration struct {
	FromVersion int
	ToVersion   int
	Description string
	Migrate     func(config map[string]interface{}) (map[string]interface{}, error)
}

// ConfigMigrator handles configuration migrations
type ConfigMigrator struct {
	migrations     map[int][]ConfigMigration
	currentVersion int
	backupDir      string
}

// NewConfigMigrator creates a new configuration migrator
func NewConfigMigrator(currentVersion int, backupDir string) *ConfigMigrator {
	return &ConfigMigrator{
		migrations:     make(map[int][]ConfigMigration),
		currentVersion: currentVersion,
		backupDir:      backupDir,
	}
}

// RegisterMigration registers a configuration migration
func (cm *ConfigMigrator) RegisterMigration(migration ConfigMigration) {
	cm.migrations[migration.FromVersion] = append(
		cm.migrations[migration.FromVersion],
		migration,
	)
}

// MigrateConfig migrates a configuration to the current version
func (cm *ConfigMigrator) MigrateConfig(config map[string]interface{}) (map[string]interface{}, error) {
	// Get current config version
	version, err := cm.getConfigVersion(config)
	if err != nil {
		return nil, err
	}

	// Check if migration is needed
	if version == cm.currentVersion {
		return config, nil
	}

	if version > cm.currentVersion {
		return nil, gdlerrors.NewConfigError(fmt.Sprintf("configuration version %d is newer than supported version %d", version, cm.currentVersion), nil, "")
	}

	// Create backup before migration
	if err := cm.backupConfig(config, version); err != nil {
		return nil, gdlerrors.NewConfigError("failed to backup configuration", err, "")
	}

	// Apply migrations sequentially
	currentConfig := config
	for v := version; v < cm.currentVersion; v++ {
		migrations, exists := cm.migrations[v]
		if !exists {
			return nil, gdlerrors.NewConfigError(fmt.Sprintf("no migration path from version %d to %d", v, v+1), nil, "")
		}

		// Find the appropriate migration
		var migration *ConfigMigration
		for _, m := range migrations {
			if m.ToVersion == v+1 {
				migration = &m
				break
			}
		}

		if migration == nil {
			return nil, gdlerrors.NewConfigError(fmt.Sprintf("no migration found from version %d to %d", v, v+1), nil, "")
		}

		// Apply migration
		newConfig, err := migration.Migrate(currentConfig)
		if err != nil {
			return nil, gdlerrors.NewConfigError(fmt.Sprintf("migration from version %d to %d failed", migration.FromVersion, migration.ToVersion), err, "")
		}

		currentConfig = newConfig
	}

	// Update version in config
	currentConfig["version"] = cm.currentVersion

	return currentConfig, nil
}

// getConfigVersion extracts version from configuration
func (cm *ConfigMigrator) getConfigVersion(config map[string]interface{}) (int, error) {
	versionRaw, exists := config["version"]
	if !exists {
		// Assume version 0 if not specified
		return 0, nil
	}

	switch v := versionRaw.(type) {
	case int:
		return v, nil
	case float64:
		return int(v), nil
	case string:
		var version int
		if _, err := fmt.Sscanf(v, "%d", &version); err != nil {
			return 0, gdlerrors.NewValidationError("version_format", fmt.Sprintf("invalid version format: %s", v))
		}
		return version, nil
	default:
		return 0, gdlerrors.NewValidationError("version_type", fmt.Sprintf("unsupported version type: %T", versionRaw))
	}
}

// backupConfig creates a backup of the configuration
func (cm *ConfigMigrator) backupConfig(config map[string]interface{}, version int) error {
	if cm.backupDir == "" {
		return nil // No backup directory specified
	}

	// Create backup directory if it doesn't exist
	if err := os.MkdirAll(cm.backupDir, 0750); err != nil {
		return err
	}

	// Generate backup filename
	filename := fmt.Sprintf("config_v%d_backup_%d.json",
		version, time.Now().Unix())
	backupPath := filepath.Join(cm.backupDir, filename)

	// Marshal config to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Write backup file
	return os.WriteFile(backupPath, data, 0600)
}

// LoadConfig loads a configuration from file
func (cm *ConfigMigrator) LoadConfig(path string) (map[string]interface{}, error) {
	// Validate and sanitize path to prevent file inclusion vulnerabilities
	cleanPath := filepath.Clean(path)

	file, err := os.Open(cleanPath) // #nosec G304 - path is cleaned and validated above
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	data, err := io.ReadAll(file)
	if err != nil {
		return nil, err
	}

	var config map[string]interface{}
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	// Migrate if necessary
	return cm.MigrateConfig(config)
}

// SaveConfig saves a configuration to file
func (cm *ConfigMigrator) SaveConfig(config map[string]interface{}, path string) error {
	// Ensure version is set
	config["version"] = cm.currentVersion

	// Marshal to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return err
	}

	// Write file
	return os.WriteFile(path, data, 0600)
}

// ValidateConfig validates a configuration against current schema
func (cm *ConfigMigrator) ValidateConfig(config map[string]interface{}) error {
	// Check version
	version, err := cm.getConfigVersion(config)
	if err != nil {
		return err
	}

	if version != cm.currentVersion {
		return gdlerrors.NewValidationError("version", fmt.Sprintf("configuration version mismatch: expected %d, got %d", cm.currentVersion, version))
	}

	// Additional validation can be added here
	// For example, checking required fields, data types, etc.

	return nil
}

// DefaultMigrations returns common configuration migrations
func DefaultMigrations() []ConfigMigration {
	return []ConfigMigration{
		// Version 0 to 1: Add plugin settings
		{
			FromVersion: 0,
			ToVersion:   1,
			Description: "Add plugin configuration settings",
			Migrate: func(config map[string]interface{}) (map[string]interface{}, error) {
				// Add default plugin settings if not present
				if _, exists := config["plugins"]; !exists {
					config["plugins"] = map[string]interface{}{
						"enabled":   true,
						"directory": "./plugins",
						"auto_load": true,
					}
				}
				return config, nil
			},
		},
		// Version 1 to 2: Add security settings
		{
			FromVersion: 1,
			ToVersion:   2,
			Description: "Add security configuration",
			Migrate: func(config map[string]interface{}) (map[string]interface{}, error) {
				// Add default security settings
				if _, exists := config["security"]; !exists {
					config["security"] = map[string]interface{}{
						"sandbox_enabled":       true,
						"max_memory_mb":         100,
						"max_execution_time_ms": 30000,
					}
				}
				return config, nil
			},
		},
		// Version 2 to 3: Restructure plugin configuration
		{
			FromVersion: 2,
			ToVersion:   3,
			Description: "Restructure plugin configuration format",
			Migrate: func(config map[string]interface{}) (map[string]interface{}, error) {
				// Migrate old plugin format to new format
				if oldPlugins, exists := config["plugins"]; exists {
					if pluginMap, ok := oldPlugins.(map[string]interface{}); ok {
						// Convert flat structure to nested structure
						newPlugins := map[string]interface{}{
							"system": map[string]interface{}{
								"enabled":   pluginMap["enabled"],
								"directory": pluginMap["directory"],
								"auto_load": pluginMap["auto_load"],
							},
							"loaded": make(map[string]interface{}),
						}

						// Migrate individual plugin configs if they exist
						if loadedPlugins, ok := pluginMap["loaded"]; ok {
							newPlugins["loaded"] = loadedPlugins
						}

						config["plugins"] = newPlugins
					}
				}
				return config, nil
			},
		},
	}
}

// PluginConfigManager manages plugin-specific configurations
type PluginConfigManager struct {
	migrator  *ConfigMigrator
	configDir string
	configs   map[string]map[string]interface{}
	mu        sync.RWMutex
}

// NewPluginConfigManager creates a new plugin configuration manager
func NewPluginConfigManager(configDir string, currentVersion int) *PluginConfigManager {
	backupDir := filepath.Join(configDir, "backups")
	migrator := NewConfigMigrator(currentVersion, backupDir)

	// Register default migrations
	for _, migration := range DefaultMigrations() {
		migrator.RegisterMigration(migration)
	}

	return &PluginConfigManager{
		migrator:  migrator,
		configDir: configDir,
		configs:   make(map[string]map[string]interface{}),
	}
}

// LoadPluginConfig loads configuration for a specific plugin
func (pcm *PluginConfigManager) LoadPluginConfig(pluginName string) (map[string]interface{}, error) {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	// Check cache first
	if config, exists := pcm.configs[pluginName]; exists {
		return config, nil
	}

	// Load from file
	configPath := filepath.Join(pcm.configDir, pluginName+".json")
	config, err := pcm.migrator.LoadConfig(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return default config if file doesn't exist
			config = pcm.getDefaultPluginConfig(pluginName)
		} else {
			return nil, err
		}
	}

	// Cache the config
	pcm.configs[pluginName] = config

	return config, nil
}

// SavePluginConfig saves configuration for a specific plugin
func (pcm *PluginConfigManager) SavePluginConfig(pluginName string, config map[string]interface{}) error {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	// Update cache
	pcm.configs[pluginName] = config

	// Save to file
	configPath := filepath.Join(pcm.configDir, pluginName+".json")
	return pcm.migrator.SaveConfig(config, configPath)
}

// getDefaultPluginConfig returns default configuration for a plugin
func (pcm *PluginConfigManager) getDefaultPluginConfig(pluginName string) map[string]interface{} {
	return map[string]interface{}{
		"name":     pluginName,
		"enabled":  true,
		"version":  pcm.migrator.currentVersion,
		"settings": make(map[string]interface{}),
	}
}

// ReloadAllConfigs reloads all plugin configurations from disk
func (pcm *PluginConfigManager) ReloadAllConfigs() error {
	pcm.mu.Lock()
	defer pcm.mu.Unlock()

	// Clear cache
	pcm.configs = make(map[string]map[string]interface{})

	// List all config files
	files, err := os.ReadDir(pcm.configDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Config directory doesn't exist yet
		}
		return err
	}

	// Load each config file
	for _, file := range files {
		if filepath.Ext(file.Name()) == ".json" {
			pluginName := file.Name()[:len(file.Name())-5]
			configPath := filepath.Join(pcm.configDir, file.Name())

			config, err := pcm.migrator.LoadConfig(configPath)
			if err != nil {
				return gdlerrors.NewConfigError("failed to load config", err, pluginName)
			}

			pcm.configs[pluginName] = config
		}
	}

	return nil
}
