package plugin

import (
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"plugin"
	"strings"
	"sync"
	"time"
)

// PluginInfo contains metadata about a loaded plugin
type PluginInfo struct {
	Path         string         `json:"path"`
	Name         string         `json:"name"`
	Version      string         `json:"version"`
	Type         string         `json:"type"`
	LoadTime     time.Time      `json:"load_time"`
	Size         int64          `json:"size"`
	Checksum     string         `json:"checksum"`
	Plugin       Plugin         `json:"-"`
	NativePlugin *plugin.Plugin `json:"-"`
}

// PluginLoader handles dynamic loading of Go plugins
type PluginLoader struct {
	searchPaths    []string
	loadedPlugins  map[string]*PluginInfo
	allowedPaths   []string
	blockedPaths   []string
	verifyChecksum bool
	maxPluginSize  int64
	mu             sync.RWMutex
}

// LoaderConfig contains configuration for the plugin loader
type LoaderConfig struct {
	SearchPaths    []string `json:"search_paths"`
	AllowedPaths   []string `json:"allowed_paths,omitempty"`
	BlockedPaths   []string `json:"blocked_paths,omitempty"`
	VerifyChecksum bool     `json:"verify_checksum"`
	MaxPluginSize  int64    `json:"max_plugin_size"` // in bytes
}

// NewPluginLoader creates a new plugin loader with the given configuration
func NewPluginLoader(config *LoaderConfig) *PluginLoader {
	if config == nil {
		config = &LoaderConfig{
			SearchPaths:    []string{"./plugins", "/usr/local/lib/godl/plugins"},
			VerifyChecksum: false,
			MaxPluginSize:  100 * 1024 * 1024, // 100MB default
		}
	}

	return &PluginLoader{
		searchPaths:    config.SearchPaths,
		allowedPaths:   config.AllowedPaths,
		blockedPaths:   config.BlockedPaths,
		verifyChecksum: config.VerifyChecksum,
		maxPluginSize:  config.MaxPluginSize,
		loadedPlugins:  make(map[string]*PluginInfo),
	}
}

// Load loads a plugin from the specified path
func (pl *PluginLoader) Load(path string) (Plugin, error) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	// Check if plugin is already loaded
	if info, exists := pl.loadedPlugins[path]; exists {
		return info.Plugin, nil
	}

	// Validate the plugin path
	if err := pl.validatePath(path); err != nil {
		return nil, fmt.Errorf("invalid plugin path: %w", err)
	}

	// Get file information
	fileInfo, err := os.Stat(path)
	if err != nil {
		return nil, fmt.Errorf("failed to stat plugin file: %w", err)
	}

	// Check file size
	if pl.maxPluginSize > 0 && fileInfo.Size() > pl.maxPluginSize {
		return nil, fmt.Errorf("plugin file too large: %d bytes (max: %d)", fileInfo.Size(), pl.maxPluginSize)
	}

	// Calculate checksum if verification is enabled
	var checksum string
	if pl.verifyChecksum {
		checksum, err = pl.calculateChecksum(path)
		if err != nil {
			return nil, fmt.Errorf("failed to calculate checksum: %w", err)
		}
	}

	// Load the plugin
	nativePlugin, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open plugin %s: %w", path, err)
	}

	// Look for the Plugin symbol
	symbol, err := nativePlugin.Lookup("Plugin")
	if err != nil {
		return nil, fmt.Errorf("plugin %s does not export 'Plugin' symbol: %w", path, err)
	}

	// Validate that the symbol implements the Plugin interface
	pluginInstance, ok := symbol.(Plugin)
	if !ok {
		return nil, fmt.Errorf("plugin %s does not implement Plugin interface", path)
	}

	// Create plugin info
	info := &PluginInfo{
		Path:         path,
		Name:         pluginInstance.Name(),
		Version:      pluginInstance.Version(),
		LoadTime:     time.Now(),
		Size:         fileInfo.Size(),
		Checksum:     checksum,
		Plugin:       pluginInstance,
		NativePlugin: nativePlugin,
	}

	// Determine plugin type
	info.Type = pl.determinePluginType(pluginInstance)

	// Store the loaded plugin
	pl.loadedPlugins[path] = info

	return pluginInstance, nil
}

// LoadFromSearchPath loads a plugin by searching through configured search paths
func (pl *PluginLoader) LoadFromSearchPath(filename string) (Plugin, error) {
	for _, searchPath := range pl.searchPaths {
		pluginPath := filepath.Join(searchPath, filename)

		// Check if file exists
		if _, err := os.Stat(pluginPath); os.IsNotExist(err) {
			continue
		}

		// Try to load the plugin
		pluginInstance, err := pl.Load(pluginPath)
		if err == nil {
			return pluginInstance, nil
		}

		// Log the error but continue searching
		// In a production system, you might want to use a proper logger here
		fmt.Printf("Failed to load plugin from %s: %v\n", pluginPath, err)
	}

	return nil, fmt.Errorf("plugin %s not found in search paths: %v", filename, pl.searchPaths)
}

// DiscoverPlugins discovers all plugins in the search paths
func (pl *PluginLoader) DiscoverPlugins() ([]string, error) {
	var discovered []string

	for _, searchPath := range pl.searchPaths {
		// Check if search path exists
		if _, err := os.Stat(searchPath); os.IsNotExist(err) {
			continue
		}

		// Walk through the search path
		err := filepath.Walk(searchPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Check if it's a .so file (Go plugin)
			if strings.HasSuffix(path, ".so") {
				discovered = append(discovered, path)
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to discover plugins in %s: %w", searchPath, err)
		}
	}

	return discovered, nil
}

// LoadAll loads all discovered plugins
func (pl *PluginLoader) LoadAll() ([]Plugin, []error) {
	discovered, err := pl.DiscoverPlugins()
	if err != nil {
		return nil, []error{fmt.Errorf("failed to discover plugins: %w", err)}
	}

	var plugins []Plugin
	var errors []error

	for _, path := range discovered {
		pluginInstance, err := pl.Load(path)
		if err != nil {
			errors = append(errors, fmt.Errorf("failed to load %s: %w", path, err))
			continue
		}
		plugins = append(plugins, pluginInstance)
	}

	return plugins, errors
}

// GetLoadedPlugins returns information about all loaded plugins
func (pl *PluginLoader) GetLoadedPlugins() map[string]*PluginInfo {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	// Return a copy to prevent external modification
	result := make(map[string]*PluginInfo)
	for path, info := range pl.loadedPlugins {
		result[path] = info
	}

	return result
}

// UnloadPlugin unloads a specific plugin
func (pl *PluginLoader) UnloadPlugin(path string) error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	info, exists := pl.loadedPlugins[path]
	if !exists {
		return fmt.Errorf("plugin %s is not loaded", path)
	}

	// Close the plugin if it implements the interface
	if err := info.Plugin.Close(); err != nil {
		return fmt.Errorf("failed to close plugin %s: %w", path, err)
	}

	// Remove from loaded plugins map
	delete(pl.loadedPlugins, path)

	return nil
}

// UnloadAll unloads all loaded plugins
func (pl *PluginLoader) UnloadAll() []error {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	var errors []error

	for path, info := range pl.loadedPlugins {
		if err := info.Plugin.Close(); err != nil {
			errors = append(errors, fmt.Errorf("failed to close plugin %s: %w", path, err))
		}
	}

	// Clear the loaded plugins map
	pl.loadedPlugins = make(map[string]*PluginInfo)

	return errors
}

// AddSearchPath adds a new search path
func (pl *PluginLoader) AddSearchPath(path string) error {
	// Validate the path
	if err := pl.validatePath(path); err != nil {
		return fmt.Errorf("invalid search path: %w", err)
	}

	pl.mu.Lock()
	defer pl.mu.Unlock()

	// Check if path already exists
	for _, existing := range pl.searchPaths {
		if existing == path {
			return nil // Already exists
		}
	}

	pl.searchPaths = append(pl.searchPaths, path)
	return nil
}

// RemoveSearchPath removes a search path
func (pl *PluginLoader) RemoveSearchPath(path string) {
	pl.mu.Lock()
	defer pl.mu.Unlock()

	for i, existing := range pl.searchPaths {
		if existing == path {
			pl.searchPaths = append(pl.searchPaths[:i], pl.searchPaths[i+1:]...)
			return
		}
	}
}

// GetSearchPaths returns the current search paths
func (pl *PluginLoader) GetSearchPaths() []string {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	// Return a copy
	paths := make([]string, len(pl.searchPaths))
	copy(paths, pl.searchPaths)
	return paths
}

// validatePath validates a plugin path against allowed and blocked paths
func (pl *PluginLoader) validatePath(path string) error {
	// Convert to absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	// Check blocked paths first
	for _, blocked := range pl.blockedPaths {
		if strings.HasPrefix(absPath, blocked) {
			return fmt.Errorf("path is blocked: %s", absPath)
		}
	}

	// If allowed paths are specified, check them
	if len(pl.allowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range pl.allowedPaths {
			if strings.HasPrefix(absPath, allowedPath) {
				allowed = true
				break
			}
		}
		if !allowed {
			return fmt.Errorf("path not in allowed paths: %s", absPath)
		}
	}

	return nil
}

// calculateChecksum calculates SHA256 checksum of a file
func (pl *PluginLoader) calculateChecksum(path string) (string, error) {
	// Validate and sanitize the path to prevent file inclusion vulnerabilities
	cleanPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanPath) {
		return "", fmt.Errorf("path must be absolute: %s", path)
	}

	file, err := os.Open(cleanPath) // #nosec G304 - path is validated and sanitized above
	if err != nil {
		return "", err
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// determinePluginType determines the type of plugin based on interfaces it implements
func (pl *PluginLoader) determinePluginType(pluginInstance Plugin) string {
	// Check different plugin types
	if _, ok := pluginInstance.(AuthPlugin); ok {
		return "auth"
	}
	if _, ok := pluginInstance.(TransformPlugin); ok {
		return "transform"
	}
	if _, ok := pluginInstance.(StoragePlugin); ok {
		return "storage"
	}
	if _, ok := pluginInstance.(ProtocolPlugin); ok {
		return "protocol"
	}
	if _, ok := pluginInstance.(DownloadPlugin); ok {
		return "download"
	}

	return "unknown"
}

// VerifyPlugin verifies a plugin's integrity and compatibility
func (pl *PluginLoader) VerifyPlugin(path string) error {
	// Validate and sanitize the path to prevent file inclusion vulnerabilities
	cleanPath := filepath.Clean(path)
	if !filepath.IsAbs(cleanPath) {
		return fmt.Errorf("path must be absolute: %s", path)
	}

	// Check if file exists and is readable
	file, err := os.Open(cleanPath) // #nosec G304 - path is validated and sanitized above
	if err != nil {
		return fmt.Errorf("cannot open plugin file: %w", err)
	}
	defer func() { _ = file.Close() }()

	// Get file info
	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("cannot stat plugin file: %w", err)
	}

	// Check file size
	if pl.maxPluginSize > 0 && info.Size() > pl.maxPluginSize {
		return fmt.Errorf("plugin file too large: %d bytes", info.Size())
	}

	// Check file permissions
	mode := info.Mode()
	if mode&0111 == 0 {
		return fmt.Errorf("plugin file is not executable")
	}

	// Additional security checks could be added here
	// For example: code signing verification, sandboxing checks, etc.

	return nil
}

// ReloadPlugin reloads a specific plugin
func (pl *PluginLoader) ReloadPlugin(path string) (Plugin, error) {
	// Unload the existing plugin
	if err := pl.UnloadPlugin(path); err != nil && !strings.Contains(err.Error(), "is not loaded") {
		return nil, fmt.Errorf("failed to unload plugin for reload: %w", err)
	}

	// Load the plugin again
	return pl.Load(path)
}

// GetPluginByName finds a loaded plugin by name
func (pl *PluginLoader) GetPluginByName(name string) (Plugin, error) {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	for _, info := range pl.loadedPlugins {
		if info.Name == name {
			return info.Plugin, nil
		}
	}

	return nil, fmt.Errorf("plugin with name %s not found", name)
}

// GetPluginsByType returns all loaded plugins of a specific type
func (pl *PluginLoader) GetPluginsByType(pluginType string) []Plugin {
	pl.mu.RLock()
	defer pl.mu.RUnlock()

	var plugins []Plugin
	for _, info := range pl.loadedPlugins {
		if info.Type == pluginType {
			plugins = append(plugins, info.Plugin)
		}
	}

	return plugins
}
