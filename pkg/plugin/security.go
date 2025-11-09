package plugin

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
)

// PluginSecurity defines security constraints for plugins
type PluginSecurity struct {
	// File system access restrictions
	AllowedPaths []string `json:"allowed_paths"`
	BlockedPaths []string `json:"blocked_paths"`
	ReadOnlyMode bool     `json:"read_only_mode"`

	// Resource limitations
	MaxMemoryUsage   int64         `json:"max_memory_mb"`      // in MB
	MaxExecutionTime time.Duration `json:"max_execution_time"` // per operation
	MaxFileSize      int64         `json:"max_file_size"`      // in bytes

	// Network access
	NetworkAccess bool     `json:"network_access"`
	AllowedHosts  []string `json:"allowed_hosts,omitempty"`
	BlockedHosts  []string `json:"blocked_hosts,omitempty"`

	// System access
	FileSystemAccess  bool `json:"file_system_access"`
	SystemCalls       bool `json:"system_calls"`
	EnvironmentAccess bool `json:"environment_access"`

	// Plugin-specific
	AllowNativeLibs  bool `json:"allow_native_libs"`
	TrustedSignature bool `json:"trusted_signature"`
	CodeSigning      bool `json:"code_signing"`
}

// DefaultSecurity returns a conservative security configuration
func DefaultSecurity() *PluginSecurity {
	return &PluginSecurity{
		AllowedPaths:      []string{"./data", "./plugins", "./temp"},
		BlockedPaths:      []string{"/etc", "/sys", "/proc", "/root", "/home"},
		ReadOnlyMode:      false,
		MaxMemoryUsage:    100, // 100MB
		MaxExecutionTime:  30 * time.Second,
		MaxFileSize:       100 * 1024 * 1024, // 100MB
		NetworkAccess:     false,
		FileSystemAccess:  true,
		SystemCalls:       false,
		EnvironmentAccess: false,
		AllowNativeLibs:   false,
		TrustedSignature:  true,
		CodeSigning:       false,
	}
}

// StrictSecurity returns a highly restrictive security configuration
func StrictSecurity() *PluginSecurity {
	return &PluginSecurity{
		AllowedPaths:      []string{"./temp"},
		BlockedPaths:      []string{"/", "/etc", "/sys", "/proc", "/root", "/home", "/usr", "/bin", "/sbin"},
		ReadOnlyMode:      true,
		MaxMemoryUsage:    50, // 50MB
		MaxExecutionTime:  10 * time.Second,
		MaxFileSize:       10 * 1024 * 1024, // 10MB
		NetworkAccess:     false,
		FileSystemAccess:  false,
		SystemCalls:       false,
		EnvironmentAccess: false,
		AllowNativeLibs:   false,
		TrustedSignature:  true,
		CodeSigning:       true,
	}
}

// SecurityValidator validates plugin operations against security policies
type SecurityValidator struct {
	policy   *PluginSecurity
	basePath string
}

// NewSecurityValidator creates a new security validator
func NewSecurityValidator(policy *PluginSecurity, basePath string) *SecurityValidator {
	if policy == nil {
		policy = DefaultSecurity()
	}

	return &SecurityValidator{
		policy:   policy,
		basePath: basePath,
	}
}

// ValidateFilePath validates if a file path is allowed
func (sv *SecurityValidator) ValidateFilePath(path string) error {
	// Resolve absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return gdlerrors.NewInvalidPathError(path, err)
	}

	// Normalize path separators
	absPath = filepath.Clean(absPath)

	// Check blocked paths first
	for _, blocked := range sv.policy.BlockedPaths {
		blockedAbs, _ := filepath.Abs(blocked)
		if strings.HasPrefix(absPath, blockedAbs) {
			return gdlerrors.WrapError(nil, gdlerrors.CodePermissionDenied, fmt.Sprintf("path %s is blocked by security policy", absPath))
		}
	}

	// Check allowed paths
	if len(sv.policy.AllowedPaths) > 0 {
		allowed := false
		for _, allowedPath := range sv.policy.AllowedPaths {
			allowedAbs, _ := filepath.Abs(allowedPath)
			if strings.HasPrefix(absPath, allowedAbs) {
				allowed = true
				break
			}
		}
		if !allowed {
			return gdlerrors.WrapError(nil, gdlerrors.CodePermissionDenied, fmt.Sprintf("path %s is not in allowed paths", absPath))
		}
	}

	return nil
}

// ValidateFileOperation validates file operations
func (sv *SecurityValidator) ValidateFileOperation(operation string, path string) error {
	if !sv.policy.FileSystemAccess {
		return gdlerrors.WrapError(nil, gdlerrors.CodePermissionDenied, "file system access is disabled")
	}

	if err := sv.ValidateFilePath(path); err != nil {
		return err
	}

	// Check read-only mode
	if sv.policy.ReadOnlyMode && (operation == "write" || operation == "create" || operation == "delete") {
		return gdlerrors.WrapError(nil, gdlerrors.CodePermissionDenied, fmt.Sprintf("write operation %s is not allowed in read-only mode", operation))
	}

	return nil
}

// ValidateFileSize validates file size against policy
func (sv *SecurityValidator) ValidateFileSize(size int64) error {
	if sv.policy.MaxFileSize > 0 && size > sv.policy.MaxFileSize {
		return gdlerrors.NewValidationError("file_size", fmt.Sprintf("file size %d bytes exceeds maximum allowed size %d bytes", size, sv.policy.MaxFileSize))
	}
	return nil
}

// ValidateNetworkAccess validates network operations
func (sv *SecurityValidator) ValidateNetworkAccess(host string) error {
	if !sv.policy.NetworkAccess {
		return gdlerrors.WrapError(nil, gdlerrors.CodePermissionDenied, "network access is disabled")
	}

	// Check blocked hosts
	for _, blocked := range sv.policy.BlockedHosts {
		if strings.Contains(host, blocked) {
			return gdlerrors.WrapError(nil, gdlerrors.CodePermissionDenied, fmt.Sprintf("host %s is blocked by security policy", host))
		}
	}

	// Check allowed hosts if specified
	if len(sv.policy.AllowedHosts) > 0 {
		allowed := false
		for _, allowedHost := range sv.policy.AllowedHosts {
			if strings.Contains(host, allowedHost) {
				allowed = true
				break
			}
		}
		if !allowed {
			return gdlerrors.WrapError(nil, gdlerrors.CodePermissionDenied, fmt.Sprintf("host %s is not in allowed hosts", host))
		}
	}

	return nil
}

// ResourceMonitor monitors plugin resource usage
type ResourceMonitor struct {
	maxMemory   int64
	maxExecTime time.Duration
	startTime   time.Time
	initialMem  runtime.MemStats
}

// NewResourceMonitor creates a new resource monitor
func NewResourceMonitor(maxMemoryMB int64, maxExecTime time.Duration) *ResourceMonitor {
	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	return &ResourceMonitor{
		maxMemory:   maxMemoryMB * 1024 * 1024, // Convert MB to bytes
		maxExecTime: maxExecTime,
		startTime:   time.Now(),
		initialMem:  memStats,
	}
}

// CheckResources validates current resource usage
func (rm *ResourceMonitor) CheckResources() error {
	// Check execution time
	if rm.maxExecTime > 0 && time.Since(rm.startTime) > rm.maxExecTime {
		return gdlerrors.WrapError(nil, gdlerrors.CodeTimeout, fmt.Sprintf("execution time exceeded maximum allowed time %v", rm.maxExecTime))
	}

	// Check memory usage
	if rm.maxMemory > 0 {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		// #nosec G115 -- Safe conversion: memory stats are controlled values
		memoryDiff := int64(memStats.Alloc - rm.initialMem.Alloc)
		if memoryDiff > rm.maxMemory {
			return gdlerrors.WrapError(nil, gdlerrors.CodePermissionDenied, fmt.Sprintf("memory usage %d bytes exceeds maximum allowed %d bytes", memoryDiff, rm.maxMemory))
		}
	}

	return nil
}

// SecurePluginExecutor wraps plugin execution with security checks
type SecurePluginExecutor struct {
	plugin    Plugin
	validator *SecurityValidator
	monitor   *ResourceMonitor
}

// NewSecurePluginExecutor creates a secure plugin executor
func NewSecurePluginExecutor(plugin Plugin, security *PluginSecurity, basePath string) *SecurePluginExecutor {
	return &SecurePluginExecutor{
		plugin:    plugin,
		validator: NewSecurityValidator(security, basePath),
		monitor:   NewResourceMonitor(security.MaxMemoryUsage, security.MaxExecutionTime),
	}
}

// Execute executes a plugin method with security checks
func (spe *SecurePluginExecutor) Execute(ctx context.Context, method string, args ...interface{}) (interface{}, error) {
	// Pre-execution security check
	if err := spe.monitor.CheckResources(); err != nil {
		return nil, gdlerrors.WrapError(err, gdlerrors.CodePermissionDenied, "pre-execution security check failed")
	}

	// Create timeout context
	if spe.monitor.maxExecTime > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, spe.monitor.maxExecTime)
		defer cancel()
	}

	// Execute the actual plugin method (this would need to be implemented based on reflection or interface methods)
	result, err := spe.executeMethod(ctx, method, args...)

	// Post-execution security check
	if resErr := spe.monitor.CheckResources(); resErr != nil {
		return nil, gdlerrors.WrapError(resErr, gdlerrors.CodePermissionDenied, "post-execution security check failed")
	}

	return result, err
}

// executeMethod is a placeholder for actual method execution
// In a real implementation, this would use reflection or predefined interface methods
func (spe *SecurePluginExecutor) executeMethod(ctx context.Context, method string, args ...interface{}) (interface{}, error) {
	// This is a simplified implementation - in practice, you'd need to use reflection
	// or have a switch statement for different plugin interface methods
	switch method {
	case "Init":
		if len(args) > 0 {
			if config, ok := args[0].(map[string]interface{}); ok {
				return nil, spe.plugin.Init(config)
			}
		}
		return nil, gdlerrors.NewValidationError("init_args", "invalid arguments for Init method")
	case "Close":
		return nil, spe.plugin.Close()
	default:
		return nil, gdlerrors.NewValidationError("method", fmt.Sprintf("unsupported method: %s", method))
	}
}

// GetValidator returns the security validator
func (spe *SecurePluginExecutor) GetValidator() *SecurityValidator {
	return spe.validator
}

// GetMonitor returns the resource monitor
func (spe *SecurePluginExecutor) GetMonitor() *ResourceMonitor {
	return spe.monitor
}
