package plugin

import (
	"fmt"
	"sort"
)

// DependencyGraph represents plugin dependencies
type DependencyGraph struct {
	nodes     map[string]*DependencyNode
	resolved  []string
	resolving map[string]bool
}

// DependencyNode represents a plugin and its dependencies
type DependencyNode struct {
	Name         string
	Version      *Version
	Dependencies []string
	Plugin       Plugin
}

// NewDependencyGraph creates a new dependency graph
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes:     make(map[string]*DependencyNode),
		resolving: make(map[string]bool),
	}
}

// AddNode adds a plugin node to the graph
func (dg *DependencyGraph) AddNode(name string, version *Version, plugin Plugin, dependencies []string) {
	dg.nodes[name] = &DependencyNode{
		Name:         name,
		Version:      version,
		Dependencies: dependencies,
		Plugin:       plugin,
	}
}

// Resolve returns the plugins in the order they should be loaded
func (dg *DependencyGraph) Resolve() ([]string, error) {
	dg.resolved = []string{}
	dg.resolving = make(map[string]bool)

	// Get all nodes and sort them for deterministic ordering
	nodeNames := make([]string, 0, len(dg.nodes))
	for name := range dg.nodes {
		nodeNames = append(nodeNames, name)
	}
	sort.Strings(nodeNames)

	// Resolve each node
	for _, name := range nodeNames {
		if err := dg.resolveNode(name); err != nil {
			return nil, err
		}
	}

	return dg.resolved, nil
}

// resolveNode recursively resolves a node and its dependencies
func (dg *DependencyGraph) resolveNode(name string) error {
	// Already resolved
	if contains(dg.resolved, name) {
		return nil
	}

	// Circular dependency detection
	if dg.resolving[name] {
		return NewPluginError(ErrCircularDependency,
			fmt.Sprintf("circular dependency detected for plugin: %s", name)).
			WithPlugin(name, "").
			WithSuggestions(
				"Review plugin dependencies",
				"Remove circular references",
				"Consider restructuring plugin architecture",
			)
	}

	node, exists := dg.nodes[name]
	if !exists {
		return NewPluginError(ErrDependencyNotFound,
			fmt.Sprintf("dependency not found: %s", name)).
			WithSuggestions(
				fmt.Sprintf("Install plugin %s", name),
				"Check plugin name spelling",
				"Verify plugin is available",
			)
	}

	// Mark as resolving
	dg.resolving[name] = true

	// Resolve dependencies first
	for _, dep := range node.Dependencies {
		if err := dg.resolveNode(dep); err != nil {
			return err
		}
	}

	// Mark as resolved
	delete(dg.resolving, name)
	dg.resolved = append(dg.resolved, name)

	return nil
}

// GetLoadOrder returns the order in which plugins should be loaded
func (dg *DependencyGraph) GetLoadOrder() []string {
	return dg.resolved
}

// ValidateDependencies checks if all dependencies can be satisfied
func (dg *DependencyGraph) ValidateDependencies() error {
	for nodeName, node := range dg.nodes {
		for _, depName := range node.Dependencies {
			depNode, exists := dg.nodes[depName]
			if !exists {
				return NewPluginError(ErrDependencyNotFound,
					fmt.Sprintf("plugin %s requires %s which is not available",
						nodeName, depName)).
					WithPlugin(nodeName, "").
					WithDetails(map[string]interface{}{
						"missing_dependency": depName,
					})
			}

			// Check version compatibility if both plugins are versioned
			if node.Version != nil && depNode.Version != nil {
				// This is a simplified check - in practice you'd want to
				// store version requirements and check against them
				if node.Version.Major < depNode.Version.Major {
					return NewPluginError(ErrIncompatibleVersion,
						fmt.Sprintf("plugin %s requires newer version of %s",
							nodeName, depName)).
						WithPlugin(nodeName, "").
						WithDetails(map[string]interface{}{
							"required_plugin":  depName,
							"required_version": depNode.Version.String(),
							"current_version":  node.Version.String(),
						})
				}
			}
		}
	}

	return nil
}

// DependencyManager manages plugin dependencies
type DependencyManager struct {
	graph    *DependencyGraph
	plugins  map[string]*VersionedPlugin
	resolved bool
}

// NewDependencyManager creates a new dependency manager
func NewDependencyManager() *DependencyManager {
	return &DependencyManager{
		graph:   NewDependencyGraph(),
		plugins: make(map[string]*VersionedPlugin),
	}
}

// RegisterPlugin registers a versioned plugin with its dependencies
func (dm *DependencyManager) RegisterPlugin(plugin *VersionedPlugin) error {
	name := plugin.Name()

	// Check if already registered
	if _, exists := dm.plugins[name]; exists {
		return ErrPluginAlreadyExists(name)
	}

	// Extract dependency names
	depNames := make([]string, 0, len(plugin.dependencies))
	for depName := range plugin.dependencies {
		depNames = append(depNames, depName)
	}

	// Add to graph
	dm.graph.AddNode(name, plugin.version, plugin, depNames)
	dm.plugins[name] = plugin
	dm.resolved = false

	return nil
}

// ResolveDependencies resolves all plugin dependencies
func (dm *DependencyManager) ResolveDependencies() ([]string, error) {
	// Validate all dependencies exist
	if err := dm.graph.ValidateDependencies(); err != nil {
		return nil, err
	}

	// Check version compatibility for all plugins
	availableVersions := make(map[string]*Version)
	for name, plugin := range dm.plugins {
		availableVersions[name] = plugin.version
	}

	for _, plugin := range dm.plugins {
		if err := plugin.CheckDependencies(availableVersions); err != nil {
			return nil, err
		}
	}

	// Resolve load order
	loadOrder, err := dm.graph.Resolve()
	if err != nil {
		return nil, err
	}

	dm.resolved = true
	return loadOrder, nil
}

// GetPlugin returns a registered plugin by name
func (dm *DependencyManager) GetPlugin(name string) (*VersionedPlugin, error) {
	plugin, exists := dm.plugins[name]
	if !exists {
		return nil, ErrPluginNotFoundError(name)
	}
	return plugin, nil
}

// GetLoadOrder returns the order in which plugins should be loaded
func (dm *DependencyManager) GetLoadOrder() ([]string, error) {
	if !dm.resolved {
		return dm.ResolveDependencies()
	}
	return dm.graph.GetLoadOrder(), nil
}

// GetDependents returns plugins that depend on the given plugin
func (dm *DependencyManager) GetDependents(pluginName string) []string {
	var dependents []string

	for name, node := range dm.graph.nodes {
		if name == pluginName {
			continue
		}

		for _, dep := range node.Dependencies {
			if dep == pluginName {
				dependents = append(dependents, name)
				break
			}
		}
	}

	return dependents
}

// CanUnload checks if a plugin can be safely unloaded
func (dm *DependencyManager) CanUnload(pluginName string) (bool, []string) {
	dependents := dm.GetDependents(pluginName)
	return len(dependents) == 0, dependents
}

// UnregisterPlugin removes a plugin from the dependency manager
func (dm *DependencyManager) UnregisterPlugin(name string) error {
	// Check if plugin exists
	_, exists := dm.plugins[name]
	if !exists {
		return NewPluginError(ErrPluginNotFound,
			fmt.Sprintf("plugin %s not found", name)).
			WithPlugin(name, "")
	}

	// Check if plugin can be unloaded
	canUnload, dependents := dm.CanUnload(name)
	if !canUnload {
		return NewPluginError(ErrPluginInUse,
			fmt.Sprintf("plugin %s is required by other plugins", name)).
			WithPlugin(name, "").
			WithDetails(map[string]interface{}{
				"dependents": dependents,
			}).
			WithSuggestions(
				"Unload dependent plugins first",
				fmt.Sprintf("Dependent plugins: %v", dependents),
			)
	}

	// Remove from graph and plugins map
	delete(dm.graph.nodes, name)
	delete(dm.plugins, name)
	dm.resolved = false

	return nil
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// PluginInUse error
const ErrPluginInUse PluginErrorCode = "PLUGIN_IN_USE"
