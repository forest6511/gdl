package plugin

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Version represents a semantic version for plugins
type Version struct {
	Major         int    `json:"major"`
	Minor         int    `json:"minor"`
	Patch         int    `json:"patch"`
	PreRelease    string `json:"pre_release,omitempty"`
	BuildMetadata string `json:"build_metadata,omitempty"`
}

// ParseVersion parses a semantic version string
func ParseVersion(versionStr string) (*Version, error) {
	// Semantic version regex pattern
	pattern := `^v?(\d+)\.(\d+)\.(\d+)(?:-([a-zA-Z0-9.-]+))?(?:\+([a-zA-Z0-9.-]+))?$`
	re := regexp.MustCompile(pattern)

	matches := re.FindStringSubmatch(versionStr)
	if matches == nil {
		return nil, fmt.Errorf("invalid version format: %s", versionStr)
	}

	major, _ := strconv.Atoi(matches[1])
	minor, _ := strconv.Atoi(matches[2])
	patch, _ := strconv.Atoi(matches[3])

	return &Version{
		Major:         major,
		Minor:         minor,
		Patch:         patch,
		PreRelease:    matches[4],
		BuildMetadata: matches[5],
	}, nil
}

// String returns the string representation of the version
func (v *Version) String() string {
	base := fmt.Sprintf("%d.%d.%d", v.Major, v.Minor, v.Patch)
	if v.PreRelease != "" {
		base += "-" + v.PreRelease
	}
	if v.BuildMetadata != "" {
		base += "+" + v.BuildMetadata
	}
	return base
}

// Compare compares two versions
// Returns: -1 if v < other, 0 if v == other, 1 if v > other
func (v *Version) Compare(other *Version) int {
	if v.Major != other.Major {
		if v.Major < other.Major {
			return -1
		}
		return 1
	}

	if v.Minor != other.Minor {
		if v.Minor < other.Minor {
			return -1
		}
		return 1
	}

	if v.Patch != other.Patch {
		if v.Patch < other.Patch {
			return -1
		}
		return 1
	}

	// Handle pre-release versions
	if v.PreRelease != "" && other.PreRelease == "" {
		return -1 // Pre-release version is less than normal version
	}
	if v.PreRelease == "" && other.PreRelease != "" {
		return 1
	}
	if v.PreRelease != "" && other.PreRelease != "" {
		return strings.Compare(v.PreRelease, other.PreRelease)
	}

	return 0
}

// IsCompatible checks if this version is compatible with a requirement
func (v *Version) parseRequirementOperator(requirement string) (string, string) {
	operators := []string{">=", "<=", "^", "~", ">", "<", "="}

	for _, op := range operators {
		if strings.HasPrefix(requirement, op) {
			return op, requirement[len(op):]
		}
	}

	// Exact version match (no operator)
	return "=", requirement
}

func (v *Version) checkCompatibilityByOperator(operator string, reqVersion *Version) bool {
	switch operator {
	case "^":
		// Caret: compatible with version (same major)
		return v.Major == reqVersion.Major && v.Compare(reqVersion) >= 0
	case "~":
		// Tilde: approximately equivalent (same major.minor)
		return v.Major == reqVersion.Major &&
			v.Minor == reqVersion.Minor &&
			v.Compare(reqVersion) >= 0
	case ">=":
		return v.Compare(reqVersion) >= 0
	case ">":
		return v.Compare(reqVersion) > 0
	case "<=":
		return v.Compare(reqVersion) <= 0
	case "<":
		return v.Compare(reqVersion) < 0
	case "=":
		return v.Compare(reqVersion) == 0
	default:
		// Exact version match (fallback)
		return v.Compare(reqVersion) == 0
	}
}

func (v *Version) IsCompatible(requirement string) (bool, error) {
	// Parse requirement patterns like "^1.2.3", "~1.2.3", ">=1.2.3", etc.
	operator, versionStr := v.parseRequirementOperator(requirement)

	reqVersion, err := ParseVersion(versionStr)
	if err != nil {
		return false, err
	}

	return v.checkCompatibilityByOperator(operator, reqVersion), nil
}

// VersionedPlugin extends the Plugin interface with version management
type VersionedPlugin struct {
	Plugin
	version      *Version
	dependencies map[string]string // plugin name -> version requirement
}

// NewVersionedPlugin creates a new versioned plugin wrapper
func NewVersionedPlugin(plugin Plugin, version string) (*VersionedPlugin, error) {
	v, err := ParseVersion(version)
	if err != nil {
		return nil, fmt.Errorf("invalid plugin version: %w", err)
	}

	return &VersionedPlugin{
		Plugin:       plugin,
		version:      v,
		dependencies: make(map[string]string),
	}, nil
}

// GetVersion returns the plugin version
func (vp *VersionedPlugin) GetVersion() *Version {
	return vp.version
}

// AddDependency adds a dependency requirement
func (vp *VersionedPlugin) AddDependency(pluginName string, versionRequirement string) {
	vp.dependencies[pluginName] = versionRequirement
}

// GetDependencies returns all dependencies
func (vp *VersionedPlugin) GetDependencies() map[string]string {
	return vp.dependencies
}

// CheckDependencies verifies all dependencies are satisfied
func (vp *VersionedPlugin) CheckDependencies(availablePlugins map[string]*Version) error {
	for pluginName, requirement := range vp.dependencies {
		availableVersion, exists := availablePlugins[pluginName]
		if !exists {
			return NewPluginError(ErrDependencyNotFound,
				fmt.Sprintf("required plugin %s not found", pluginName)).
				WithPlugin(vp.Name(), "").
				WithDetails(map[string]interface{}{
					"required_plugin": pluginName,
					"requirement":     requirement,
				})
		}

		compatible, err := availableVersion.IsCompatible(requirement)
		if err != nil {
			return NewPluginErrorWithCause(ErrInvalidConfiguration,
				fmt.Sprintf("invalid version requirement: %s", requirement), err)
		}

		if !compatible {
			return NewPluginError(ErrIncompatibleVersion,
				fmt.Sprintf("plugin %s version %s does not satisfy requirement %s",
					pluginName, availableVersion.String(), requirement)).
				WithPlugin(vp.Name(), "").
				WithDetails(map[string]interface{}{
					"required_plugin":   pluginName,
					"available_version": availableVersion.String(),
					"requirement":       requirement,
				})
		}
	}

	return nil
}
