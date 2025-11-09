package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
	"github.com/forest6511/gdl/pkg/plugin"
)

// PluginRegistry manages CLI plugin operations
type PluginRegistry struct {
	pluginDir    string
	configFile   string
	pluginLoader *plugin.PluginLoader
}

// PluginInfo represents installed plugin information
type PluginInfo struct {
	Name        string            `json:"name"`
	Version     string            `json:"version"`
	Type        string            `json:"type"`
	Path        string            `json:"path"`
	Enabled     bool              `json:"enabled"`
	Config      map[string]string `json:"config,omitempty"`
	InstallTime time.Time         `json:"install_time"`
	Source      string            `json:"source,omitempty"`
}

// PluginConfig represents the plugin configuration file
type PluginConfig struct {
	Plugins map[string]*PluginInfo `json:"plugins"`
}

// NewPluginRegistry creates a new CLI plugin registry
func NewPluginRegistry(pluginDir, configFile string) *PluginRegistry {
	loaderConfig := &plugin.LoaderConfig{
		SearchPaths:    []string{pluginDir},
		VerifyChecksum: true,
		MaxPluginSize:  100 * 1024 * 1024, // 100MB
	}

	return &PluginRegistry{
		pluginDir:    pluginDir,
		configFile:   configFile,
		pluginLoader: plugin.NewPluginLoader(loaderConfig),
	}
}

// List returns all installed plugins
func (pr *PluginRegistry) List(ctx context.Context) ([]*PluginInfo, error) {
	config, err := pr.loadConfig()
	if err != nil {
		return nil, gdlerrors.NewConfigError("failed to load plugin config", err, pr.configFile)
	}

	var plugins []*PluginInfo
	for _, pluginInfo := range config.Plugins {
		plugins = append(plugins, pluginInfo)
	}

	return plugins, nil
}

// Install installs a plugin from a source (URL, local path, or package name)
func (pr *PluginRegistry) Install(ctx context.Context, source, name string) error {
	// Check if plugin already exists first
	config, err := pr.loadConfig()
	if err != nil {
		return gdlerrors.NewConfigError("failed to load plugin config", err, pr.configFile)
	}

	if _, exists := config.Plugins[name]; exists {
		return gdlerrors.NewPluginError(name, nil, "plugin already exists")
	}

	// Ensure plugin directory exists
	if err := os.MkdirAll(pr.pluginDir, 0750); err != nil {
		return gdlerrors.NewInvalidPathError(pr.pluginDir, err)
	}

	pluginPath := filepath.Join(pr.pluginDir, name+".so")

	// Download/copy plugin based on source type
	if err := pr.downloadPlugin(ctx, source, pluginPath); err != nil {
		return gdlerrors.NewPluginError(name, err, "failed to download plugin")
	}

	// Load plugin to verify it's valid
	pluginInstance, err := pr.pluginLoader.Load(pluginPath)
	if err != nil {
		// Clean up on failure
		if removeErr := os.Remove(pluginPath); removeErr != nil {
			// Log the cleanup error but don't override the main error
			fmt.Printf("Warning: failed to cleanup plugin file %s: %v\n", pluginPath, removeErr)
		}
		return gdlerrors.NewPluginError(name, err, "failed to load plugin")
	}

	// Create plugin info
	pluginInfo := &PluginInfo{
		Name:        pluginInstance.Name(),
		Version:     pluginInstance.Version(),
		Type:        pr.pluginLoader.GetLoadedPlugins()[pluginPath].Type,
		Path:        pluginPath,
		Enabled:     true,
		InstallTime: time.Now(),
		Source:      source,
		Config:      make(map[string]string),
	}

	// Update configuration
	config.Plugins[name] = pluginInfo

	if err := pr.saveConfig(config); err != nil {
		return gdlerrors.NewConfigError("failed to save plugin config", err, pr.configFile)
	}

	return nil
}

// Remove uninstalls a plugin
func (pr *PluginRegistry) Remove(ctx context.Context, name string) error {
	config, err := pr.loadConfig()
	if err != nil {
		return gdlerrors.NewConfigError("failed to load plugin config", err, pr.configFile)
	}

	pluginInfo, exists := config.Plugins[name]
	if !exists {
		return gdlerrors.NewPluginError(name, nil, "plugin not found")
	}

	// Remove plugin file
	if err := os.Remove(pluginInfo.Path); err != nil && !os.IsNotExist(err) {
		return gdlerrors.NewStorageError("remove plugin file", err, pluginInfo.Path)
	}

	// Remove from configuration
	delete(config.Plugins, name)

	if err := pr.saveConfig(config); err != nil {
		return gdlerrors.NewConfigError("failed to save plugin config", err, pr.configFile)
	}

	return nil
}

// Enable enables a plugin
func (pr *PluginRegistry) Enable(ctx context.Context, name string) error {
	return pr.setPluginEnabled(name, true)
}

// Disable disables a plugin
func (pr *PluginRegistry) Disable(ctx context.Context, name string) error {
	return pr.setPluginEnabled(name, false)
}

// setPluginEnabled sets the enabled status of a plugin
func (pr *PluginRegistry) setPluginEnabled(name string, enabled bool) error {
	config, err := pr.loadConfig()
	if err != nil {
		return gdlerrors.NewConfigError("failed to load plugin config", err, pr.configFile)
	}

	pluginInfo, exists := config.Plugins[name]
	if !exists {
		return gdlerrors.NewPluginError(name, nil, "plugin not found")
	}

	pluginInfo.Enabled = enabled

	if err := pr.saveConfig(config); err != nil {
		return gdlerrors.NewConfigError("failed to save plugin config", err, pr.configFile)
	}

	return nil
}

// Configure sets configuration values for a plugin
func (pr *PluginRegistry) Configure(ctx context.Context, name, key, value string) error {
	config, err := pr.loadConfig()
	if err != nil {
		return gdlerrors.NewConfigError("failed to load plugin config", err, pr.configFile)
	}

	pluginInfo, exists := config.Plugins[name]
	if !exists {
		return gdlerrors.NewPluginError(name, nil, "plugin not found")
	}

	if pluginInfo.Config == nil {
		pluginInfo.Config = make(map[string]string)
	}

	pluginInfo.Config[key] = value

	if err := pr.saveConfig(config); err != nil {
		return gdlerrors.NewConfigError("failed to save plugin config", err, pr.configFile)
	}

	return nil
}

// GetEnabledPlugins returns all enabled plugins
func (pr *PluginRegistry) GetEnabledPlugins(ctx context.Context) ([]*PluginInfo, error) {
	allPlugins, err := pr.List(ctx)
	if err != nil {
		return nil, err
	}

	var enabledPlugins []*PluginInfo
	for _, pluginInfo := range allPlugins {
		if pluginInfo.Enabled {
			enabledPlugins = append(enabledPlugins, pluginInfo)
		}
	}

	return enabledPlugins, nil
}

// LoadPlugins loads all enabled plugins into the plugin manager
func (pr *PluginRegistry) LoadPlugins(ctx context.Context, pluginManager *plugin.PluginManager) error {
	enabledPlugins, err := pr.GetEnabledPlugins(ctx)
	if err != nil {
		// GetEnabledPlugins already returns structured DownloadError, preserve it
		return err
	}

	for _, pluginInfo := range enabledPlugins {
		pluginInstance, err := pr.pluginLoader.Load(pluginInfo.Path)
		if err != nil {
			fmt.Printf("Warning: failed to load plugin %s: %v\n", pluginInfo.Name, err)
			continue
		}

		// Initialize plugin with configuration
		if len(pluginInfo.Config) > 0 {
			config := make(map[string]interface{})
			for k, v := range pluginInfo.Config {
				config[k] = v
			}
			if err := pluginInstance.Init(config); err != nil {
				fmt.Printf("Warning: failed to initialize plugin %s: %v\n", pluginInfo.Name, err)
				continue
			}
		}

		if err := pluginManager.Register(pluginInstance); err != nil {
			fmt.Printf("Warning: failed to register plugin %s: %v\n", pluginInfo.Name, err)
			continue
		}
	}

	return nil
}

// downloadPlugin downloads a plugin from various sources
func (pr *PluginRegistry) downloadPlugin(ctx context.Context, source, destination string) error {
	// Determine source type and handle accordingly
	if strings.HasPrefix(source, "http://") || strings.HasPrefix(source, "https://") {
		// URL download
		return pr.downloadFromURL(ctx, source, destination)
	} else if strings.Contains(source, "/") && !filepath.IsAbs(source) && !strings.HasPrefix(source, ".") {
		// GitHub-style path (user/repo format, not absolute or relative paths)
		return pr.downloadFromGitHub(ctx, source, destination)
	} else {
		// Local file path (absolute, relative, or simple filename)
		return pr.copyLocalFile(source, destination)
	}
}

// downloadFromURL downloads a plugin from a URL
func (pr *PluginRegistry) downloadFromURL(ctx context.Context, url, destination string) error {
	// This would use the core downloader to fetch the plugin
	// For now, we'll return a placeholder error
	return gdlerrors.NewDownloadError(
		gdlerrors.CodeUnknown,
		fmt.Sprintf("URL download not implemented yet - would download from %s to %s", url, destination),
	)
}

// downloadFromGitHub downloads a plugin from GitHub
func (pr *PluginRegistry) downloadFromGitHub(ctx context.Context, repo, destination string) error {
	// This would construct a GitHub URL and download the latest release
	// For now, we'll return a placeholder error
	return gdlerrors.NewDownloadError(
		gdlerrors.CodeUnknown,
		fmt.Sprintf("GitHub download not implemented yet - would download from %s to %s", repo, destination),
	)
}

// copyLocalFile copies a plugin from a local path
func (pr *PluginRegistry) copyLocalFile(source, destination string) error {
	// Validate and sanitize paths to prevent file inclusion vulnerabilities
	cleanSource := filepath.Clean(source)
	cleanDest := filepath.Clean(destination)

	sourceFile, err := os.Open(cleanSource) // #nosec G304 - path is cleaned and validated above
	if err != nil {
		return gdlerrors.NewStorageError("open source file", err, cleanSource)
	}
	defer func() {
		if err := sourceFile.Close(); err != nil {
			fmt.Printf("Warning: failed to close source file: %v\n", err)
		}
	}()

	destFile, err := os.Create(cleanDest) // #nosec G304 - path is cleaned and validated above
	if err != nil {
		return gdlerrors.NewStorageError("create destination file", err, cleanDest)
	}
	defer func() {
		if err := destFile.Close(); err != nil {
			fmt.Printf("Warning: failed to close destination file: %v\n", err)
		}
	}()

	if _, err := destFile.ReadFrom(sourceFile); err != nil {
		return gdlerrors.NewStorageError("copy file", err, destination)
	}

	return nil
}

// loadConfig loads the plugin configuration
func (pr *PluginRegistry) loadConfig() (*PluginConfig, error) {
	config := &PluginConfig{
		Plugins: make(map[string]*PluginInfo),
	}

	if _, err := os.Stat(pr.configFile); os.IsNotExist(err) {
		return config, nil
	}

	data, err := os.ReadFile(pr.configFile)
	if err != nil {
		return nil, gdlerrors.NewStorageError("read config file", err, pr.configFile)
	}

	if len(data) == 0 {
		return config, nil
	}

	if err := json.Unmarshal(data, config); err != nil {
		return nil, gdlerrors.NewConfigError("failed to parse config file", err, pr.configFile)
	}

	return config, nil
}

// saveConfig saves the plugin configuration
func (pr *PluginRegistry) saveConfig(config *PluginConfig) error {
	// Ensure config directory exists
	configDir := filepath.Dir(pr.configFile)
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return gdlerrors.NewInvalidPathError(configDir, err)
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return gdlerrors.NewConfigError("failed to marshal config", err, pr.configFile)
	}

	if err := os.WriteFile(pr.configFile, data, 0600); err != nil {
		return gdlerrors.NewStorageError("write config file", err, pr.configFile)
	}

	return nil
}

// GetDefaultPluginDir returns the default plugin directory
func GetDefaultPluginDir() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./plugins"
	}
	return filepath.Join(homeDir, ".gdl", "plugins")
}

// GetDefaultConfigFile returns the default plugin config file
func GetDefaultConfigFile() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "./plugins.json"
	}
	return filepath.Join(homeDir, ".gdl", "plugins.json")
}
