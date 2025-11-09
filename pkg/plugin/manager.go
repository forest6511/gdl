package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"plugin"
	"sync"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
	"github.com/forest6511/gdl/pkg/types"
)

// HookType represents different types of plugin hooks
type HookType = types.HookType

const (
	PreDownloadHook  = types.PreDownloadHook
	PostDownloadHook = types.PostDownloadHook
	PreStoreHook     = types.PreStoreHook
	PostStoreHook    = types.PostStoreHook
	AuthHook         = types.AuthHook
	TransformHook    = types.TransformHook
)

// PluginHook represents a hook function
type PluginHook func(data interface{}) error

// PluginManager manages all registered plugins
type PluginManager struct {
	plugins        map[string]Plugin
	hooks          map[HookType][]PluginHook
	security       *PluginSecurity
	errorCollector *ErrorCollector
	mu             sync.RWMutex
}

// NewPluginManager creates a new plugin manager instance
func NewPluginManager() *PluginManager {
	return &PluginManager{
		plugins:        make(map[string]Plugin),
		hooks:          make(map[HookType][]PluginHook),
		security:       DefaultSecurity(),
		errorCollector: NewErrorCollector(),
	}
}

// Register adds a plugin to the manager
func (pm *PluginManager) Register(plugin Plugin) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	name := plugin.Name()
	if _, exists := pm.plugins[name]; exists {
		err := ErrPluginAlreadyExists(name)
		pm.errorCollector.Add(err)
		return err
	}

	// Wrap plugin with security if not already wrapped
	securePlugin := plugin
	if _, isSecure := plugin.(*SecurePlugin); !isSecure {
		securePlugin = NewSecurePlugin(plugin, pm.security, ".")
	}

	pm.plugins[name] = securePlugin
	return nil
}

// Unregister removes a plugin from the manager
func (pm *PluginManager) Unregister(name string) error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		err := ErrPluginNotFoundError(name)
		pm.errorCollector.Add(err)
		return err
	}

	// Close the plugin before removing it
	if err := plugin.Close(); err != nil {
		pluginErr := ErrPluginInitError(name, err).WithSuggestions(
			"Check if plugin is still in use",
			"Review plugin cleanup procedures",
		)
		pm.errorCollector.Add(pluginErr)
		return pluginErr
	}

	delete(pm.plugins, name)
	return nil
}

// Get retrieves a plugin by name
func (pm *PluginManager) Get(name string) (Plugin, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	plugin, exists := pm.plugins[name]
	if !exists {
		return nil, ErrPluginNotFoundError(name)
	}

	return plugin, nil
}

// ExecuteHook executes all hooks of a given type
func (pm *PluginManager) ExecuteHook(hook HookType, data interface{}) error {
	pm.mu.RLock()
	hooks := pm.hooks[hook]
	pm.mu.RUnlock()

	collector := NewErrorCollector()

	for i, hookFunc := range hooks {
		if err := hookFunc(data); err != nil {
			hookErr := NewPluginErrorWithCause(ErrHookExecutionFailed,
				fmt.Sprintf("hook %d execution failed", i), err).
				WithDetails(map[string]interface{}{
					"hook_type":  string(hook),
					"hook_index": i,
					"data_type":  fmt.Sprintf("%T", data),
				}).
				WithSuggestions(
					"Check hook implementation for errors",
					"Verify hook data is valid",
					"Review hook error handling",
				)
			collector.Add(hookErr)
			pm.errorCollector.Add(hookErr)

			// Continue executing other hooks unless it's a critical error
			if hookErr.GetSeverity() == "CRITICAL" {
				return hookErr
			}
		}
	}

	if collector.HasErrors() {
		return collector
	}

	return nil
}

// LoadFromDirectory loads all plugins from a directory
func (pm *PluginManager) LoadFromDirectory(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return gdlerrors.NewInvalidPathError(dir, nil)
	}

	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories and non-.so files
		if info.IsDir() || filepath.Ext(path) != ".so" {
			return nil
		}

		return pm.loadPlugin(path)
	})
}

// loadPlugin loads a single plugin from a file
func (pm *PluginManager) loadPlugin(path string) error {
	p, err := plugin.Open(path)
	if err != nil {
		return gdlerrors.NewPluginError(path, err, "failed to open plugin")
	}

	// Look for the plugin instance
	sym, err := p.Lookup("Plugin")
	if err != nil {
		return gdlerrors.NewPluginError(path, err, "does not export 'Plugin' symbol")
	}

	pluginInstance, ok := sym.(Plugin)
	if !ok {
		return gdlerrors.NewPluginError(path, nil, "does not implement Plugin interface")
	}

	return pm.Register(pluginInstance)
}

// AddHook adds a hook function for a specific hook type
func (pm *PluginManager) AddHook(hookType HookType, hookFunc PluginHook) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.hooks[hookType] = append(pm.hooks[hookType], hookFunc)
}

// RemoveHooks removes all hooks for a specific hook type
func (pm *PluginManager) RemoveHooks(hookType HookType) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	delete(pm.hooks, hookType)
}

// ListPlugins returns a list of all registered plugin names
func (pm *PluginManager) ListPlugins() []string {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	names := make([]string, 0, len(pm.plugins))
	for name := range pm.plugins {
		names = append(names, name)
	}

	return names
}

// Close closes all registered plugins
func (pm *PluginManager) Close() error {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	var errors []error
	for name, plugin := range pm.plugins {
		if err := plugin.Close(); err != nil {
			errors = append(errors, gdlerrors.NewPluginError(name, err, "failed to close plugin"))
		}
	}

	if len(errors) > 0 {
		// Return first error to preserve its error code and context
		// TODO: Consider using errors.Join() in Go 1.20+ for multiple errors
		return errors[0]
	}

	return nil
}

// SetSecurity updates the security policy for the plugin manager
func (pm *PluginManager) SetSecurity(security *PluginSecurity) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.security = security
}

// GetErrorCollector returns the error collector for diagnostics
func (pm *PluginManager) GetErrorCollector() *ErrorCollector {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	return pm.errorCollector
}

// ClearErrors clears all collected errors
func (pm *PluginManager) ClearErrors() {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	pm.errorCollector = NewErrorCollector()
}

// GetPluginStats returns statistics about loaded plugins
func (pm *PluginManager) GetPluginStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["total_plugins"] = len(pm.plugins)
	stats["total_hooks"] = len(pm.hooks)
	stats["error_count"] = len(pm.errorCollector.GetErrors())
	stats["critical_errors"] = len(pm.errorCollector.GetCriticalErrors())

	// Count hooks by type
	hookCounts := make(map[string]int)
	for hookType, hooks := range pm.hooks {
		hookCounts[string(hookType)] = len(hooks)
	}
	stats["hooks_by_type"] = hookCounts

	return stats
}
