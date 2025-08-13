package plugin

import (
	"testing"
)

func TestDependencyGraph_AddNode(t *testing.T) {
	graph := NewDependencyGraph()

	plugin := &MockPlugin{name: "test", version: "1.0.0"}
	version := &Version{Major: 1, Minor: 0, Patch: 0}
	deps := []string{"dep1", "dep2"}

	graph.AddNode("test", version, plugin, deps)

	if _, exists := graph.nodes["test"]; !exists {
		t.Error("Node should be added to graph")
	}

	node := graph.nodes["test"]
	if node.Name != "test" {
		t.Errorf("Node name = %v, want %v", node.Name, "test")
	}

	if len(node.Dependencies) != 2 {
		t.Errorf("Dependencies length = %v, want %v", len(node.Dependencies), 2)
	}
}

func TestDependencyGraph_Resolve(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(*DependencyGraph)
		want    []string
		wantErr bool
	}{
		{
			name: "simple dependency chain",
			setup: func(dg *DependencyGraph) {
				dg.AddNode("a", nil, &MockPlugin{name: "a"}, []string{"b"})
				dg.AddNode("b", nil, &MockPlugin{name: "b"}, []string{"c"})
				dg.AddNode("c", nil, &MockPlugin{name: "c"}, []string{})
			},
			want:    []string{"c", "b", "a"},
			wantErr: false,
		},
		{
			name: "multiple dependencies",
			setup: func(dg *DependencyGraph) {
				dg.AddNode("a", nil, &MockPlugin{name: "a"}, []string{"b", "c"})
				dg.AddNode("b", nil, &MockPlugin{name: "b"}, []string{})
				dg.AddNode("c", nil, &MockPlugin{name: "c"}, []string{})
			},
			want:    []string{"b", "c", "a"},
			wantErr: false,
		},
		{
			name: "circular dependency",
			setup: func(dg *DependencyGraph) {
				dg.AddNode("a", nil, &MockPlugin{name: "a"}, []string{"b"})
				dg.AddNode("b", nil, &MockPlugin{name: "b"}, []string{"c"})
				dg.AddNode("c", nil, &MockPlugin{name: "c"}, []string{"a"})
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "missing dependency",
			setup: func(dg *DependencyGraph) {
				dg.AddNode("a", nil, &MockPlugin{name: "a"}, []string{"missing"})
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "independent nodes",
			setup: func(dg *DependencyGraph) {
				dg.AddNode("a", nil, &MockPlugin{name: "a"}, []string{})
				dg.AddNode("b", nil, &MockPlugin{name: "b"}, []string{})
				dg.AddNode("c", nil, &MockPlugin{name: "c"}, []string{})
			},
			want:    []string{"a", "b", "c"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			graph := NewDependencyGraph()
			tt.setup(graph)

			got, err := graph.Resolve()
			if (err != nil) != tt.wantErr {
				t.Errorf("Resolve() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if len(got) != len(tt.want) {
					t.Errorf("Resolve() length = %v, want %v", len(got), len(tt.want))
				}

				// For independent nodes, order doesn't matter as much
				// Just check all expected nodes are present
				for _, expected := range tt.want {
					found := false
					for _, actual := range got {
						if actual == expected {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("Expected node %s not found in result", expected)
					}
				}
			}
		})
	}
}

func TestDependencyGraph_ValidateDependencies(t *testing.T) {
	graph := NewDependencyGraph()

	// Add nodes with compatible versions
	v2 := &Version{Major: 2, Minor: 0, Patch: 0} // plugin1 has higher version
	v1 := &Version{Major: 1, Minor: 0, Patch: 0} // plugin2 has lower version

	graph.AddNode("plugin1", v2, &MockPlugin{name: "plugin1"}, []string{"plugin2"})
	graph.AddNode("plugin2", v1, &MockPlugin{name: "plugin2"}, []string{})

	// Should validate successfully (plugin1 v2.0.0 can depend on plugin2 v1.0.0)
	err := graph.ValidateDependencies()
	if err != nil {
		t.Errorf("ValidateDependencies() error = %v", err)
	}

	// Add node with missing dependency
	graph.AddNode("plugin3", v1, &MockPlugin{name: "plugin3"}, []string{"missing"})

	err = graph.ValidateDependencies()
	if err == nil {
		t.Error("Expected error for missing dependency")
	}
}

func TestDependencyManager_RegisterPlugin(t *testing.T) {
	dm := NewDependencyManager()

	plugin := &MockPlugin{name: "test", version: "1.0.0"}
	vp, _ := NewVersionedPlugin(plugin, "1.0.0")

	err := dm.RegisterPlugin(vp)
	if err != nil {
		t.Errorf("RegisterPlugin() error = %v", err)
	}

	// Try to register same plugin again
	err = dm.RegisterPlugin(vp)
	if err == nil {
		t.Error("Expected error for duplicate registration")
	}
}

func TestDependencyManager_ResolveDependencies(t *testing.T) {
	dm := NewDependencyManager()

	// Create plugins with dependencies
	plugin1 := &MockPlugin{name: "plugin1", version: "1.0.0"}
	vp1, _ := NewVersionedPlugin(plugin1, "1.0.0")
	vp1.AddDependency("plugin2", "^1.0.0")

	plugin2 := &MockPlugin{name: "plugin2", version: "1.5.0"}
	vp2, _ := NewVersionedPlugin(plugin2, "1.5.0")

	plugin3 := &MockPlugin{name: "plugin3", version: "2.0.0"}
	vp3, _ := NewVersionedPlugin(plugin3, "2.0.0")
	vp3.AddDependency("plugin2", ">=1.0.0")

	// Register plugins
	_ = dm.RegisterPlugin(vp2) // Register dependency first
	_ = dm.RegisterPlugin(vp1)
	_ = dm.RegisterPlugin(vp3)

	// Resolve dependencies
	loadOrder, err := dm.ResolveDependencies()
	if err != nil {
		t.Errorf("ResolveDependencies() error = %v", err)
	}

	// plugin2 should come before plugin1 and plugin3
	plugin2Index := -1
	plugin1Index := -1
	plugin3Index := -1

	for i, name := range loadOrder {
		switch name {
		case "plugin1":
			plugin1Index = i
		case "plugin2":
			plugin2Index = i
		case "plugin3":
			plugin3Index = i
		}
	}

	if plugin2Index >= plugin1Index {
		t.Error("plugin2 should be loaded before plugin1")
	}

	if plugin2Index >= plugin3Index {
		t.Error("plugin2 should be loaded before plugin3")
	}
}

func TestDependencyManager_GetPlugin(t *testing.T) {
	dm := NewDependencyManager()

	plugin := &MockPlugin{name: "test", version: "1.0.0"}
	vp, _ := NewVersionedPlugin(plugin, "1.0.0")

	_ = dm.RegisterPlugin(vp)

	// Get existing plugin
	retrieved, err := dm.GetPlugin("test")
	if err != nil {
		t.Errorf("GetPlugin() error = %v", err)
	}

	if retrieved.Name() != "test" {
		t.Errorf("Plugin name = %v, want %v", retrieved.Name(), "test")
	}

	// Get non-existent plugin
	_, err = dm.GetPlugin("non-existent")
	if err == nil {
		t.Error("Expected error for non-existent plugin")
	}
}

func TestDependencyManager_GetDependents(t *testing.T) {
	dm := NewDependencyManager()

	// Create plugins with dependencies
	plugin1 := &MockPlugin{name: "plugin1", version: "1.0.0"}
	vp1, _ := NewVersionedPlugin(plugin1, "1.0.0")
	vp1.AddDependency("base", "^1.0.0")

	plugin2 := &MockPlugin{name: "plugin2", version: "1.0.0"}
	vp2, _ := NewVersionedPlugin(plugin2, "1.0.0")
	vp2.AddDependency("base", "^1.0.0")

	base := &MockPlugin{name: "base", version: "1.0.0"}
	vpBase, _ := NewVersionedPlugin(base, "1.0.0")

	_ = dm.RegisterPlugin(vpBase)
	_ = dm.RegisterPlugin(vp1)
	_ = dm.RegisterPlugin(vp2)

	dependents := dm.GetDependents("base")
	if len(dependents) != 2 {
		t.Errorf("GetDependents() length = %v, want %v", len(dependents), 2)
	}

	// Check both dependents are found
	hasPlugin1 := false
	hasPlugin2 := false
	for _, dep := range dependents {
		if dep == "plugin1" {
			hasPlugin1 = true
		}
		if dep == "plugin2" {
			hasPlugin2 = true
		}
	}

	if !hasPlugin1 || !hasPlugin2 {
		t.Error("Expected both plugin1 and plugin2 as dependents")
	}
}

func TestDependencyManager_CanUnload(t *testing.T) {
	dm := NewDependencyManager()

	// Create plugins with dependencies
	plugin1 := &MockPlugin{name: "plugin1", version: "1.0.0"}
	vp1, _ := NewVersionedPlugin(plugin1, "1.0.0")
	vp1.AddDependency("base", "^1.0.0")

	base := &MockPlugin{name: "base", version: "1.0.0"}
	vpBase, _ := NewVersionedPlugin(base, "1.0.0")

	_ = dm.RegisterPlugin(vpBase)
	_ = dm.RegisterPlugin(vp1)

	// Cannot unload base because plugin1 depends on it
	canUnload, dependents := dm.CanUnload("base")
	if canUnload {
		t.Error("Should not be able to unload base with dependents")
	}

	if len(dependents) != 1 || dependents[0] != "plugin1" {
		t.Errorf("Expected plugin1 as dependent, got: %v", dependents)
	}

	// Can unload plugin1 because nothing depends on it
	canUnload, dependents = dm.CanUnload("plugin1")
	if !canUnload {
		t.Error("Should be able to unload plugin1")
	}

	if len(dependents) != 0 {
		t.Errorf("Expected no dependents, got: %v", dependents)
	}
}

func TestDependencyManager_UnregisterPlugin(t *testing.T) {
	dm := NewDependencyManager()

	plugin := &MockPlugin{name: "test", version: "1.0.0"}
	vp, _ := NewVersionedPlugin(plugin, "1.0.0")

	_ = dm.RegisterPlugin(vp)

	// Unregister plugin
	err := dm.UnregisterPlugin("test")
	if err != nil {
		t.Errorf("UnregisterPlugin() error = %v", err)
	}

	// Plugin should not be found
	_, err = dm.GetPlugin("test")
	if err == nil {
		t.Error("Plugin should not exist after unregistration")
	}

	// Try to unregister again
	err = dm.UnregisterPlugin("test")
	if err == nil {
		t.Error("Expected error when unregistering non-existent plugin")
	}
}

func TestDependencyManager_UnregisterWithDependents(t *testing.T) {
	dm := NewDependencyManager()

	// Create plugins with dependencies
	plugin1 := &MockPlugin{name: "plugin1", version: "1.0.0"}
	vp1, _ := NewVersionedPlugin(plugin1, "1.0.0")
	vp1.AddDependency("base", "^1.0.0")

	base := &MockPlugin{name: "base", version: "1.0.0"}
	vpBase, _ := NewVersionedPlugin(base, "1.0.0")

	_ = dm.RegisterPlugin(vpBase)
	_ = dm.RegisterPlugin(vp1)

	// Try to unregister base (should fail)
	err := dm.UnregisterPlugin("base")
	if err == nil {
		t.Error("Expected error when unregistering plugin with dependents")
	}

	pluginErr, ok := err.(*PluginError)
	if !ok {
		t.Error("Expected PluginError type")
	} else if pluginErr.Code != ErrPluginInUse {
		t.Errorf("Expected ErrPluginInUse, got: %v", pluginErr.Code)
	}
}

func TestContains(t *testing.T) {
	slice := []string{"a", "b", "c"}

	if !contains(slice, "b") {
		t.Error("Should contain 'b'")
	}

	if contains(slice, "d") {
		t.Error("Should not contain 'd'")
	}

	if contains([]string{}, "a") {
		t.Error("Empty slice should not contain anything")
	}
}

func TestDependencyManager_GetLoadOrder(t *testing.T) {
	dm := NewDependencyManager()

	// Create simple dependency chain
	plugin1 := &MockPlugin{name: "plugin1", version: "1.0.0"}
	vp1, _ := NewVersionedPlugin(plugin1, "1.0.0")
	vp1.AddDependency("plugin2", "^1.0.0")

	plugin2 := &MockPlugin{name: "plugin2", version: "1.0.0"}
	vp2, _ := NewVersionedPlugin(plugin2, "1.0.0")

	_ = dm.RegisterPlugin(vp2)
	_ = dm.RegisterPlugin(vp1)

	// Get load order without explicit resolution
	loadOrder, err := dm.GetLoadOrder()
	if err != nil {
		t.Errorf("GetLoadOrder() error = %v", err)
	}

	if len(loadOrder) != 2 {
		t.Errorf("LoadOrder length = %v, want %v", len(loadOrder), 2)
	}

	// plugin2 should come before plugin1
	if loadOrder[0] != "plugin2" || loadOrder[1] != "plugin1" {
		t.Errorf("LoadOrder = %v, want [plugin2, plugin1]", loadOrder)
	}

	// Call again (should use cached result)
	loadOrder2, err := dm.GetLoadOrder()
	if err != nil {
		t.Errorf("GetLoadOrder() (cached) error = %v", err)
	}

	if len(loadOrder2) != len(loadOrder) {
		t.Error("Cached load order should match original")
	}
}
