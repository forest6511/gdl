package plugin

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// HotReloadConfig configures hot reload behavior
type HotReloadConfig struct {
	Enabled          bool          `json:"enabled"`
	WatchDirectories []string      `json:"watch_directories"`
	CheckInterval    time.Duration `json:"check_interval"`
	Debounce         time.Duration `json:"debounce"`
	AutoRestart      bool          `json:"auto_restart"`
	MaxRetries       int           `json:"max_retries"`
}

// DefaultHotReloadConfig returns default hot reload configuration
func DefaultHotReloadConfig() *HotReloadConfig {
	return &HotReloadConfig{
		Enabled:          false,
		WatchDirectories: []string{"./plugins"},
		CheckInterval:    time.Second * 5,
		Debounce:         time.Millisecond * 500,
		AutoRestart:      true,
		MaxRetries:       3,
	}
}

// HotReloadManager manages plugin hot reloading
type HotReloadManager struct {
	config      *HotReloadConfig
	manager     *PluginManager
	depManager  *DependencyManager
	watcher     interface{}               // File watcher interface
	pluginFiles map[string]PluginFileInfo // plugin name -> file info
	reloadQueue chan string
	stopCh      chan struct{}
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
}

// PluginFileInfo stores information about a plugin file
type PluginFileInfo struct {
	Path         string
	ModTime      time.Time
	Size         int64
	Hash         string
	PluginName   string
	Version      string
	Dependencies []string
}

// NewHotReloadManager creates a new hot reload manager
func NewHotReloadManager(config *HotReloadConfig, manager *PluginManager, depManager *DependencyManager) (*HotReloadManager, error) {
	if config == nil {
		config = DefaultHotReloadConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	hrm := &HotReloadManager{
		config:      config,
		manager:     manager,
		depManager:  depManager,
		watcher:     nil, // Would be initialized with actual file watcher
		pluginFiles: make(map[string]PluginFileInfo),
		reloadQueue: make(chan string, 100),
		stopCh:      make(chan struct{}),
		ctx:         ctx,
		cancel:      cancel,
	}

	// Add watch directories
	for _, dir := range config.WatchDirectories {
		if err := hrm.addWatchDirectory(dir); err != nil {
			return nil, fmt.Errorf("failed to watch directory %s: %w", dir, err)
		}
	}

	return hrm, nil
}

// Start begins hot reload monitoring
func (hrm *HotReloadManager) Start() error {
	if !hrm.config.Enabled {
		return nil
	}

	// Start file watcher
	go hrm.watchFiles()

	// Start reload processor
	go hrm.processReloads()

	// Start periodic check for new plugins
	if hrm.config.CheckInterval > 0 {
		go hrm.periodicCheck()
	}

	return nil
}

// Stop stops hot reload monitoring
func (hrm *HotReloadManager) Stop() error {
	hrm.cancel()
	close(hrm.stopCh)
	// If using a real file watcher, close it here
	return nil
}

// addWatchDirectory adds a directory to watch for changes
func (hrm *HotReloadManager) addWatchDirectory(dir string) error {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(absDir, 0750); err != nil {
		return err
	}

	// In a real implementation, add to file watcher here
	// For now, just scan the directory

	// Scan for existing plugins
	return hrm.scanDirectory(absDir)
}

// scanDirectory scans a directory for plugin files
func (hrm *HotReloadManager) scanDirectory(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && filepath.Ext(path) == ".so" {
			hrm.registerPluginFile(path, info)
		}

		return nil
	})
}

// registerPluginFile registers a plugin file for tracking
func (hrm *HotReloadManager) registerPluginFile(path string, info os.FileInfo) {
	hrm.mu.Lock()
	defer hrm.mu.Unlock()

	// Extract plugin name from file
	pluginName := extractPluginName(path)

	hrm.pluginFiles[pluginName] = PluginFileInfo{
		Path:       path,
		ModTime:    info.ModTime(),
		Size:       info.Size(),
		PluginName: pluginName,
	}
}

// watchFiles watches for file system events
func (hrm *HotReloadManager) watchFiles() {
	// Simplified implementation without fsnotify
	// In production, you would use a proper file watcher library

	ticker := time.NewTicker(hrm.config.CheckInterval)
	defer ticker.Stop()

	knownFiles := make(map[string]time.Time)

	for {
		select {
		case <-ticker.C:
			// Scan directories for changes
			for _, dir := range hrm.config.WatchDirectories {
				_ = filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
					if err != nil || info.IsDir() || filepath.Ext(path) != ".so" {
						return nil
					}

					// Check if file has changed
					if lastMod, exists := knownFiles[path]; exists {
						if info.ModTime().After(lastMod) {
							hrm.queueReload(path)
						}
					} else {
						// New file discovered
						hrm.queueReload(path)
					}

					knownFiles[path] = info.ModTime()
					return nil
				})
			}

		case <-hrm.stopCh:
			return
		}
	}
}

// queueReload queues a plugin for reloading
func (hrm *HotReloadManager) queueReload(path string) {
	select {
	case hrm.reloadQueue <- path:
	default:
		// Queue is full, skip this reload
		fmt.Printf("Reload queue full, skipping reload for: %s\n", path)
	}
}

// processReloads processes queued plugin reloads
func (hrm *HotReloadManager) processReloads() {
	for {
		select {
		case path := <-hrm.reloadQueue:
			if err := hrm.reloadPlugin(path); err != nil {
				fmt.Printf("Failed to reload plugin %s: %v\n", path, err)

				// Retry if configured
				if hrm.config.AutoRestart && hrm.config.MaxRetries > 0 {
					go hrm.retryReload(path, hrm.config.MaxRetries)
				}
			}

		case <-hrm.stopCh:
			return
		}
	}
}

// reloadPlugin reloads a single plugin
func (hrm *HotReloadManager) reloadPlugin(path string) error {
	pluginName := extractPluginName(path)

	hrm.mu.Lock()
	oldInfo, exists := hrm.pluginFiles[pluginName]
	hrm.mu.Unlock()

	// Check if file still exists
	fileInfo, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return hrm.handlePluginRemoval(path)
		}
		return err
	}

	// Check if file has actually changed
	if exists && oldInfo.ModTime.Equal(fileInfo.ModTime()) && oldInfo.Size == fileInfo.Size() {
		return nil // No changes
	}

	// Get dependents before unloading
	dependents := []string{}
	if hrm.depManager != nil {
		dependents = hrm.depManager.GetDependents(pluginName)
	}

	// Unload existing plugin if loaded
	if exists {
		if err := hrm.unloadPlugin(pluginName); err != nil {
			return fmt.Errorf("failed to unload plugin: %w", err)
		}
	}

	// Load new version
	if err := hrm.loadPlugin(path); err != nil {
		return fmt.Errorf("failed to load plugin: %w", err)
	}

	// Reload dependents if necessary
	for _, dependent := range dependents {
		if depInfo, ok := hrm.pluginFiles[dependent]; ok {
			if err := hrm.reloadPlugin(depInfo.Path); err != nil {
				fmt.Printf("Failed to reload dependent plugin %s: %v\n", dependent, err)
			}
		}
	}

	// Update file info
	hrm.mu.Lock()
	hrm.pluginFiles[pluginName] = PluginFileInfo{
		Path:       path,
		ModTime:    fileInfo.ModTime(),
		Size:       fileInfo.Size(),
		PluginName: pluginName,
	}
	hrm.mu.Unlock()

	fmt.Printf("Successfully reloaded plugin: %s\n", pluginName)
	return nil
}

// unloadPlugin unloads a plugin
func (hrm *HotReloadManager) unloadPlugin(name string) error {
	// Check dependencies
	if hrm.depManager != nil {
		canUnload, dependents := hrm.depManager.CanUnload(name)
		if !canUnload {
			return fmt.Errorf("cannot unload plugin %s, required by: %v", name, dependents)
		}

		// Unregister from dependency manager
		if err := hrm.depManager.UnregisterPlugin(name); err != nil {
			return err
		}
	}

	// Unregister from plugin manager
	return hrm.manager.Unregister(name)
}

// loadPlugin loads a plugin from file
func (hrm *HotReloadManager) loadPlugin(path string) error {
	// This would use the plugin loader to load the plugin
	// For now, we'll return a placeholder
	return hrm.manager.loadPlugin(path)
}

// handlePluginRemoval handles when a plugin file is removed
func (hrm *HotReloadManager) handlePluginRemoval(path string) error {
	pluginName := extractPluginName(path)

	hrm.mu.Lock()
	delete(hrm.pluginFiles, pluginName)
	hrm.mu.Unlock()

	return hrm.unloadPlugin(pluginName)
}

// retryReload retries loading a plugin
func (hrm *HotReloadManager) retryReload(path string, retries int) {
	for i := 0; i < retries; i++ {
		time.Sleep(time.Second * time.Duration(i+1)) // Exponential backoff

		if err := hrm.reloadPlugin(path); err == nil {
			return // Success
		}
	}

	fmt.Printf("Failed to reload plugin %s after %d retries\n", path, retries)
}

// periodicCheck periodically checks for new plugins
func (hrm *HotReloadManager) periodicCheck() {
	ticker := time.NewTicker(hrm.config.CheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			for _, dir := range hrm.config.WatchDirectories {
				if err := hrm.scanDirectory(dir); err != nil {
					fmt.Printf("Periodic scan error for %s: %v\n", dir, err)
				}
			}

		case <-hrm.stopCh:
			return
		}
	}
}

// GetPluginInfo returns information about a tracked plugin file
func (hrm *HotReloadManager) GetPluginInfo(name string) (PluginFileInfo, bool) {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()

	info, exists := hrm.pluginFiles[name]
	return info, exists
}

// GetTrackedPlugins returns all tracked plugin names
func (hrm *HotReloadManager) GetTrackedPlugins() []string {
	hrm.mu.RLock()
	defer hrm.mu.RUnlock()

	names := make([]string, 0, len(hrm.pluginFiles))
	for name := range hrm.pluginFiles {
		names = append(names, name)
	}

	return names
}

// extractPluginName extracts plugin name from file path
func extractPluginName(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return base[:len(base)-len(ext)]
}
