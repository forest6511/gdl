package plugin

import (
	"context"
	"io"
	"net/http"

	"github.com/forest6511/gdl/pkg/types"
)

// DownloadRequest represents a download request for plugins
type DownloadRequest struct {
	URL     string
	Options *types.DownloadOptions
}

// DownloadResponse represents a download response for plugins
type DownloadResponse struct {
	Stats *types.DownloadStats
	Error error
}

// Core plugin interfaces
type Plugin interface {
	Name() string
	Version() string
	Init(config map[string]interface{}) error
	Close() error

	// Security validation method
	ValidateAccess(operation string, resource string) error
}

type DownloadPlugin interface {
	Plugin
	PreDownload(ctx context.Context, req *DownloadRequest) error
	PostDownload(ctx context.Context, resp *DownloadResponse) error
}

type ProtocolPlugin interface {
	Plugin
	SupportedSchemes() []string
	Download(ctx context.Context, url string, writer io.Writer) error
}

type StoragePlugin interface {
	Plugin
	Store(ctx context.Context, data []byte, key string) error
	Retrieve(ctx context.Context, key string) ([]byte, error)
}

type AuthPlugin interface {
	Plugin
	Authenticate(ctx context.Context, req *http.Request) error
}

type TransformPlugin interface {
	Plugin
	Transform(data []byte) ([]byte, error)
}

// SecurePlugin wraps a plugin with security constraints
type SecurePlugin struct {
	Plugin
	security  *PluginSecurity
	validator *SecurityValidator
	monitor   *ResourceMonitor
}

// NewSecurePlugin creates a new secure plugin wrapper
func NewSecurePlugin(plugin Plugin, security *PluginSecurity, basePath string) *SecurePlugin {
	return &SecurePlugin{
		Plugin:    plugin,
		security:  security,
		validator: NewSecurityValidator(security, basePath),
		monitor:   NewResourceMonitor(security.MaxMemoryUsage, security.MaxExecutionTime),
	}
}

// ValidateAccess implements security validation for the wrapped plugin
func (sp *SecurePlugin) ValidateAccess(operation string, resource string) error {
	// Check if operation requires file system access
	if operation == "read" || operation == "write" || operation == "create" || operation == "delete" {
		return sp.validator.ValidateFileOperation(operation, resource)
	}

	// Check if operation requires network access
	if operation == "network" {
		return sp.validator.ValidateNetworkAccess(resource)
	}

	// Check resource limits
	if err := sp.monitor.CheckResources(); err != nil {
		return ErrResourceLimitError("execution", sp.security.MaxExecutionTime, sp.monitor.maxExecTime)
	}

	return nil
}

// Init wraps the plugin's Init method with security checks
func (sp *SecurePlugin) Init(config map[string]interface{}) error {
	if err := sp.ValidateAccess("init", "config"); err != nil {
		return ErrSecurityViolationError("init", map[string]interface{}{
			"operation": "init",
			"config":    config,
		})
	}

	return sp.Plugin.Init(config)
}

// Close wraps the plugin's Close method with security checks
func (sp *SecurePlugin) Close() error {
	if err := sp.ValidateAccess("close", "cleanup"); err != nil {
		return ErrSecurityViolationError("close", map[string]interface{}{
			"operation": "close",
		})
	}

	return sp.Plugin.Close()
}
