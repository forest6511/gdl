package plugin

import (
	"testing"
)

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    *Version
		wantErr bool
	}{
		{
			name:  "simple version",
			input: "1.2.3",
			want: &Version{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			wantErr: false,
		},
		{
			name:  "version with v prefix",
			input: "v2.0.0",
			want: &Version{
				Major: 2,
				Minor: 0,
				Patch: 0,
			},
			wantErr: false,
		},
		{
			name:  "version with pre-release",
			input: "1.0.0-alpha.1",
			want: &Version{
				Major:      1,
				Minor:      0,
				Patch:      0,
				PreRelease: "alpha.1",
			},
			wantErr: false,
		},
		{
			name:  "version with build metadata",
			input: "1.0.0+build.123",
			want: &Version{
				Major:         1,
				Minor:         0,
				Patch:         0,
				BuildMetadata: "build.123",
			},
			wantErr: false,
		},
		{
			name:  "version with pre-release and build",
			input: "2.1.0-rc.1+build.456",
			want: &Version{
				Major:         2,
				Minor:         1,
				Patch:         0,
				PreRelease:    "rc.1",
				BuildMetadata: "build.456",
			},
			wantErr: false,
		},
		{
			name:    "invalid version",
			input:   "invalid",
			want:    nil,
			wantErr: true,
		},
		{
			name:    "incomplete version",
			input:   "1.2",
			want:    nil,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseVersion(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseVersion() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Major != tt.want.Major || got.Minor != tt.want.Minor || got.Patch != tt.want.Patch {
					t.Errorf("ParseVersion() = %v, want %v", got, tt.want)
				}
				if got.PreRelease != tt.want.PreRelease {
					t.Errorf("PreRelease = %v, want %v", got.PreRelease, tt.want.PreRelease)
				}
				if got.BuildMetadata != tt.want.BuildMetadata {
					t.Errorf("BuildMetadata = %v, want %v", got.BuildMetadata, tt.want.BuildMetadata)
				}
			}
		})
	}
}

func TestVersion_String(t *testing.T) {
	tests := []struct {
		name    string
		version *Version
		want    string
	}{
		{
			name: "simple version",
			version: &Version{
				Major: 1,
				Minor: 2,
				Patch: 3,
			},
			want: "1.2.3",
		},
		{
			name: "with pre-release",
			version: &Version{
				Major:      1,
				Minor:      0,
				Patch:      0,
				PreRelease: "alpha",
			},
			want: "1.0.0-alpha",
		},
		{
			name: "with build metadata",
			version: &Version{
				Major:         2,
				Minor:         0,
				Patch:         0,
				BuildMetadata: "build.123",
			},
			want: "2.0.0+build.123",
		},
		{
			name: "with both",
			version: &Version{
				Major:         1,
				Minor:         0,
				Patch:         0,
				PreRelease:    "beta.1",
				BuildMetadata: "sha.abc123",
			},
			want: "1.0.0-beta.1+sha.abc123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.version.String(); got != tt.want {
				t.Errorf("String() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_Compare(t *testing.T) {
	tests := []struct {
		name string
		v1   *Version
		v2   *Version
		want int
	}{
		{
			name: "equal versions",
			v1:   &Version{Major: 1, Minor: 0, Patch: 0},
			v2:   &Version{Major: 1, Minor: 0, Patch: 0},
			want: 0,
		},
		{
			name: "major version difference",
			v1:   &Version{Major: 2, Minor: 0, Patch: 0},
			v2:   &Version{Major: 1, Minor: 0, Patch: 0},
			want: 1,
		},
		{
			name: "minor version difference",
			v1:   &Version{Major: 1, Minor: 1, Patch: 0},
			v2:   &Version{Major: 1, Minor: 0, Patch: 0},
			want: 1,
		},
		{
			name: "patch version difference",
			v1:   &Version{Major: 1, Minor: 0, Patch: 1},
			v2:   &Version{Major: 1, Minor: 0, Patch: 0},
			want: 1,
		},
		{
			name: "pre-release vs normal",
			v1:   &Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha"},
			v2:   &Version{Major: 1, Minor: 0, Patch: 0},
			want: -1,
		},
		{
			name: "normal vs pre-release",
			v1:   &Version{Major: 1, Minor: 0, Patch: 0},
			v2:   &Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha"},
			want: 1,
		},
		{
			name: "pre-release comparison",
			v1:   &Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "alpha"},
			v2:   &Version{Major: 1, Minor: 0, Patch: 0, PreRelease: "beta"},
			want: -1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.v1.Compare(tt.v2); got != tt.want {
				t.Errorf("Compare() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersion_IsCompatible(t *testing.T) {
	tests := []struct {
		name        string
		version     *Version
		requirement string
		want        bool
		wantErr     bool
	}{
		{
			name:        "caret - same major compatible",
			version:     &Version{Major: 1, Minor: 2, Patch: 3},
			requirement: "^1.0.0",
			want:        true,
		},
		{
			name:        "caret - different major incompatible",
			version:     &Version{Major: 2, Minor: 0, Patch: 0},
			requirement: "^1.0.0",
			want:        false,
		},
		{
			name:        "caret - lower version incompatible",
			version:     &Version{Major: 1, Minor: 0, Patch: 0},
			requirement: "^1.2.0",
			want:        false,
		},
		{
			name:        "tilde - same minor compatible",
			version:     &Version{Major: 1, Minor: 2, Patch: 5},
			requirement: "~1.2.0",
			want:        true,
		},
		{
			name:        "tilde - different minor incompatible",
			version:     &Version{Major: 1, Minor: 3, Patch: 0},
			requirement: "~1.2.0",
			want:        false,
		},
		{
			name:        "greater than or equal",
			version:     &Version{Major: 2, Minor: 0, Patch: 0},
			requirement: ">=1.5.0",
			want:        true,
		},
		{
			name:        "greater than",
			version:     &Version{Major: 2, Minor: 0, Patch: 0},
			requirement: ">1.9.9",
			want:        true,
		},
		{
			name:        "less than or equal",
			version:     &Version{Major: 1, Minor: 5, Patch: 0},
			requirement: "<=2.0.0",
			want:        true,
		},
		{
			name:        "less than",
			version:     &Version{Major: 1, Minor: 9, Patch: 9},
			requirement: "<2.0.0",
			want:        true,
		},
		{
			name:        "exact match with =",
			version:     &Version{Major: 1, Minor: 2, Patch: 3},
			requirement: "=1.2.3",
			want:        true,
		},
		{
			name:        "exact match without operator",
			version:     &Version{Major: 1, Minor: 2, Patch: 3},
			requirement: "1.2.3",
			want:        true,
		},
		{
			name:        "invalid requirement",
			version:     &Version{Major: 1, Minor: 0, Patch: 0},
			requirement: "invalid",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.version.IsCompatible(tt.requirement)
			if (err != nil) != tt.wantErr {
				t.Errorf("IsCompatible() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsCompatible() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestVersionedPlugin(t *testing.T) {
	mockPlugin := &MockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
	}

	vp, err := NewVersionedPlugin(mockPlugin, "1.2.3")
	if err != nil {
		t.Fatalf("NewVersionedPlugin() error = %v", err)
	}

	if vp.GetVersion().String() != "1.2.3" {
		t.Errorf("GetVersion() = %v, want %v", vp.GetVersion().String(), "1.2.3")
	}

	// Test invalid version
	_, err = NewVersionedPlugin(mockPlugin, "invalid")
	if err == nil {
		t.Error("Expected error for invalid version")
	}
}

func TestVersionedPlugin_Dependencies(t *testing.T) {
	mockPlugin := &MockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
	}

	vp, _ := NewVersionedPlugin(mockPlugin, "1.0.0")

	// Add dependencies
	vp.AddDependency("dep1", "^1.0.0")
	vp.AddDependency("dep2", "~2.0.0")

	deps := vp.GetDependencies()
	if len(deps) != 2 {
		t.Errorf("GetDependencies() length = %v, want %v", len(deps), 2)
	}

	if deps["dep1"] != "^1.0.0" {
		t.Errorf("deps[dep1] = %v, want %v", deps["dep1"], "^1.0.0")
	}
}

func TestVersionedPlugin_CheckDependencies(t *testing.T) {
	mockPlugin := &MockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
	}

	vp, _ := NewVersionedPlugin(mockPlugin, "1.0.0")
	vp.AddDependency("dep1", "^1.0.0")
	vp.AddDependency("dep2", ">=2.0.0")

	availablePlugins := map[string]*Version{
		"dep1": {Major: 1, Minor: 5, Patch: 0},
		"dep2": {Major: 2, Minor: 1, Patch: 0},
	}

	// Should succeed with compatible versions
	err := vp.CheckDependencies(availablePlugins)
	if err != nil {
		t.Errorf("CheckDependencies() error = %v", err)
	}

	// Test missing dependency
	delete(availablePlugins, "dep2")
	err = vp.CheckDependencies(availablePlugins)
	if err == nil {
		t.Error("Expected error for missing dependency")
	}

	// Test incompatible version
	availablePlugins["dep2"] = &Version{Major: 1, Minor: 9, Patch: 0}
	err = vp.CheckDependencies(availablePlugins)
	if err == nil {
		t.Error("Expected error for incompatible version")
	}
}

func TestVersionedPlugin_ErrorCases(t *testing.T) {
	plugin := &MockPlugin{name: "test", version: "1.0.0"}
	vp, err := NewVersionedPlugin(plugin, "1.0.0")
	if err != nil {
		t.Fatalf("NewVersionedPlugin failed: %v", err)
	}

	t.Run("CheckDependencies_MissingDependency", func(t *testing.T) {
		vp.AddDependency("missing-plugin", "^1.0.0")

		availableVersions := map[string]*Version{
			"other-plugin": {Major: 1, Minor: 0, Patch: 0},
		}

		err := vp.CheckDependencies(availableVersions)
		if err == nil {
			t.Error("Expected error for missing dependency")
		}
	})

	t.Run("CheckDependencies_IncompatibleVersion", func(t *testing.T) {
		vp2, _ := NewVersionedPlugin(plugin, "2.0.0")
		vp2.AddDependency("old-plugin", "^3.0.0")

		availableVersions := map[string]*Version{
			"old-plugin": {Major: 1, Minor: 0, Patch: 0}, // Version 1.0.0, but needs ^3.0.0
		}

		err := vp2.CheckDependencies(availableVersions)
		if err == nil {
			t.Error("Expected error for incompatible version")
		}
	})

	t.Run("GetDependencies_Empty", func(t *testing.T) {
		vp3, _ := NewVersionedPlugin(plugin, "1.0.0")
		deps := vp3.GetDependencies()

		if len(deps) != 0 {
			t.Errorf("Expected empty dependencies, got %d", len(deps))
		}
	})

	t.Run("GetDependencies_Multiple", func(t *testing.T) {
		vp4, _ := NewVersionedPlugin(plugin, "1.0.0")
		vp4.AddDependency("dep1", "^1.0.0")
		vp4.AddDependency("dep2", "~2.0.0")
		vp4.AddDependency("dep3", ">=3.0.0")

		deps := vp4.GetDependencies()
		if len(deps) != 3 {
			t.Errorf("Expected 3 dependencies, got %d", len(deps))
		}

		// Check that all dependencies are present
		expectedDeps := map[string]string{
			"dep1": "^1.0.0",
			"dep2": "~2.0.0",
			"dep3": ">=3.0.0",
		}

		for name, requirement := range expectedDeps {
			if deps[name] != requirement {
				t.Errorf("Expected %s: %s, got %s", name, requirement, deps[name])
			}
		}
	})
}
