# Plugin Development Guide

This guide provides comprehensive information for developing plugins for the godl download tool.

## Table of Contents

1. [Overview](#overview)
2. [Plugin Architecture](#plugin-architecture)
3. [Plugin Interfaces](#plugin-interfaces)
4. [Development Setup](#development-setup)
5. [Creating Your First Plugin](#creating-your-first-plugin)
6. [Plugin Types](#plugin-types)
7. [Best Practices](#best-practices)
8. [Testing](#testing)
9. [Distribution](#distribution)
10. [Examples](#examples)

## Overview

godl's plugin system allows developers to extend the downloader's functionality through well-defined interfaces. Plugins can provide:

- **Authentication mechanisms** (OAuth2, API keys, custom schemes)
- **Protocol support** (FTP, S3, custom protocols)
- **Storage backends** (cloud storage, databases, custom storage)
- **Content transformation** (compression, encryption, format conversion)
- **Download hooks** (pre/post processing, logging, analytics)

## Plugin Architecture

```
┌─────────────────────────────────────────────────────────────┐
│                      godl Core                              │
├─────────────────────────────────────────────────────────────┤
│                 Plugin Manager                              │
├─────────────────────────────────────────────────────────────┤
│  Auth Plugins  │ Protocol Plugins │ Storage Plugins         │
├────────────────┼──────────────────┼─────────────────────────┤
│ Transform      │ Hook Plugins     │ Custom Plugins          │
│ Plugins        │                  │                         │
└─────────────────────────────────────────────────────────────┘
```

### Core Components

- **Plugin Interface**: Base interface all plugins must implement
- **Plugin Manager**: Handles plugin lifecycle and execution
- **Dependency Manager**: Manages plugin dependencies and load order
- **Security Validator**: Ensures plugin security and isolation
- **Hot Reload Manager**: Supports dynamic plugin loading/unloading

## Plugin Interfaces

### Base Plugin Interface

All plugins must implement the base `Plugin` interface:

```go
type Plugin interface {
    // Basic plugin information
    Name() string
    Version() string
    
    // Plugin lifecycle
    Init(config map[string]interface{}) error
    Close() error
    
    // Security validation
    ValidateAccess(operation string, resource string) error
}
```

### Specialized Plugin Types

#### Authentication Plugin

```go
type AuthPlugin interface {
    Plugin
    
    // Authentication methods
    Authenticate(ctx context.Context, request *AuthRequest) (*AuthResponse, error)
    RefreshToken(ctx context.Context, token string) (*AuthResponse, error)
    ValidateToken(ctx context.Context, token string) error
}
```

#### Protocol Plugin

```go
type ProtocolPlugin interface {
    Plugin
    
    // Protocol handling
    SupportsScheme(scheme string) bool
    CreateHandler(ctx context.Context, url *url.URL) (ProtocolHandler, error)
}

type ProtocolHandler interface {
    Download(ctx context.Context, destination io.Writer) error
    GetInfo(ctx context.Context) (*FileInfo, error)
    SupportsResume() bool
}
```

#### Storage Plugin

```go
type StoragePlugin interface {
    Plugin
    
    // Storage operations
    Store(ctx context.Context, key string, data io.Reader) error
    Retrieve(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
    Exists(ctx context.Context, key string) (bool, error)
    List(ctx context.Context, prefix string) ([]StorageItem, error)
}
```

#### Transform Plugin

```go
type TransformPlugin interface {
    Plugin
    
    // Data transformation
    Transform(ctx context.Context, input io.Reader, output io.Writer) error
    GetTransformInfo() TransformInfo
}
```

## Development Setup

### Prerequisites

- Go 1.21 or later
- godl development environment

### Project Structure

```
your-plugin/
├── go.mod
├── go.sum
├── main.go              # Plugin entry point
├── plugin.go            # Plugin implementation
├── config/
│   └── config.go       # Configuration handling
├── internal/           # Internal implementation
└── examples/
    └── usage.go        # Usage examples
```

### Initialize Your Plugin

```bash
# Create plugin directory
mkdir my-godl-plugin
cd my-godl-plugin

# Initialize Go module
go mod init github.com/youruser/my-godl-plugin

# Add godl dependency
go get github.com/forest6511/godl
```

## Creating Your First Plugin

### Example: Simple Authentication Plugin

```go
// plugin.go
package main

import (
    "context"
    "fmt"
    "github.com/forest6511/godl/pkg/plugin"
)

// SimpleAuthPlugin implements a basic API key authentication
type SimpleAuthPlugin struct {
    apiKey    string
    headerKey string
    enabled   bool
}

// Plugin interface implementation
func (p *SimpleAuthPlugin) Name() string {
    return "simple-auth"
}

func (p *SimpleAuthPlugin) Version() string {
    return "1.0.0"
}

func (p *SimpleAuthPlugin) Init(config map[string]interface{}) error {
    if apiKey, ok := config["api_key"].(string); ok {
        p.apiKey = apiKey
    } else {
        return fmt.Errorf("api_key is required in configuration")
    }
    
    p.headerKey = "X-API-Key" // default
    if headerKey, ok := config["header_key"].(string); ok {
        p.headerKey = headerKey
    }
    
    p.enabled = true
    return nil
}

func (p *SimpleAuthPlugin) Close() error {
    p.enabled = false
    return nil
}

func (p *SimpleAuthPlugin) ValidateAccess(operation string, resource string) error {
    if !p.enabled {
        return fmt.Errorf("plugin is not enabled")
    }
    return nil
}

// AuthPlugin interface implementation
func (p *SimpleAuthPlugin) Authenticate(ctx context.Context, request *plugin.AuthRequest) (*plugin.AuthResponse, error) {
    if request.Headers == nil {
        request.Headers = make(map[string]string)
    }
    
    request.Headers[p.headerKey] = p.apiKey
    
    return &plugin.AuthResponse{
        Success: true,
        Headers: request.Headers,
    }, nil
}

func (p *SimpleAuthPlugin) RefreshToken(ctx context.Context, token string) (*plugin.AuthResponse, error) {
    // Simple auth doesn't need token refresh
    return &plugin.AuthResponse{Success: true}, nil
}

func (p *SimpleAuthPlugin) ValidateToken(ctx context.Context, token string) error {
    return nil // Always valid for simple auth
}

// Plugin export symbol
var Plugin plugin.AuthPlugin = &SimpleAuthPlugin{}
```

### Build Plugin

```bash
# Build as shared library
go build -buildmode=plugin -o simple-auth.so .
```

### Plugin Configuration

Create a configuration file for your plugin:

```json
{
    "name": "simple-auth",
    "version": "1.0.0",
    "type": "auth",
    "enabled": true,
    "config": {
        "api_key": "your-api-key-here",
        "header_key": "X-API-Key"
    }
}
```

## Plugin Types

### 1. Authentication Plugins

Handle various authentication mechanisms:

```go
// OAuth2 Authentication Plugin
type OAuth2Plugin struct {
    clientID     string
    clientSecret string
    tokenURL     string
    currentToken *oauth2.Token
}

func (p *OAuth2Plugin) Authenticate(ctx context.Context, request *plugin.AuthRequest) (*plugin.AuthResponse, error) {
    token, err := p.getValidToken(ctx)
    if err != nil {
        return nil, err
    }
    
    request.Headers["Authorization"] = "Bearer " + token.AccessToken
    return &plugin.AuthResponse{
        Success: true,
        Headers: request.Headers,
        Token:   token.AccessToken,
    }, nil
}
```

### 2. Protocol Plugins

Support custom protocols:

```go
// S3 Protocol Plugin
type S3Plugin struct {
    client *s3.Client
}

func (p *S3Plugin) SupportsScheme(scheme string) bool {
    return scheme == "s3"
}

func (p *S3Plugin) CreateHandler(ctx context.Context, url *url.URL) (plugin.ProtocolHandler, error) {
    return &S3Handler{
        client: p.client,
        bucket: url.Host,
        key:    url.Path,
    }, nil
}
```

### 3. Storage Plugins

Implement custom storage backends:

```go
// Database Storage Plugin
type DatabaseStoragePlugin struct {
    db *sql.DB
}

func (p *DatabaseStoragePlugin) Store(ctx context.Context, key string, data io.Reader) error {
    content, err := io.ReadAll(data)
    if err != nil {
        return err
    }
    
    _, err = p.db.ExecContext(ctx, 
        "INSERT INTO downloads (key, content, created_at) VALUES (?, ?, ?)",
        key, content, time.Now())
    return err
}
```

### 4. Transform Plugins

Process downloaded content:

```go
// Compression Plugin
type CompressionPlugin struct {
    algorithm string
}

func (p *CompressionPlugin) Transform(ctx context.Context, input io.Reader, output io.Writer) error {
    switch p.algorithm {
    case "gzip":
        return p.gzipCompress(input, output)
    case "zstd":
        return p.zstdCompress(input, output)
    default:
        return fmt.Errorf("unsupported compression algorithm: %s", p.algorithm)
    }
}
```

## Best Practices

### 1. Plugin Design

```go
// ✅ Good: Clear, focused interface
type LoggingPlugin struct {
    logger *log.Logger
    config LoggingConfig
}

// ❌ Bad: Too many responsibilities
type SuperPlugin struct {
    logger    *log.Logger
    auth      AuthHandler
    storage   StorageHandler
    transform TransformHandler
}
```

### 2. Error Handling

```go
// ✅ Good: Structured error handling
func (p *MyPlugin) ProcessData(data []byte) error {
    if len(data) == 0 {
        return plugin.NewPluginError(
            plugin.ErrInvalidInput,
            "empty data provided",
        ).WithSuggestions("ensure data is not empty before processing")
    }
    
    // Process data...
    return nil
}

// ❌ Bad: Generic errors
func (p *MyPlugin) ProcessData(data []byte) error {
    if len(data) == 0 {
        return errors.New("bad data")
    }
    return nil
}
```

### 3. Configuration Management

```go
type PluginConfig struct {
    Timeout    time.Duration `json:"timeout" validate:"required,min=1s"`
    MaxRetries int           `json:"max_retries" validate:"min=0,max=10"`
    APIKey     string        `json:"api_key" validate:"required"`
}

func (p *MyPlugin) Init(config map[string]interface{}) error {
    var cfg PluginConfig
    if err := mapstructure.Decode(config, &cfg); err != nil {
        return fmt.Errorf("invalid configuration: %w", err)
    }
    
    if err := validator.Validate(cfg); err != nil {
        return fmt.Errorf("configuration validation failed: %w", err)
    }
    
    p.config = cfg
    return nil
}
```

### 4. Resource Management

```go
func (p *MyPlugin) Init(config map[string]interface{}) error {
    // Initialize resources
    p.client = &http.Client{
        Timeout: p.config.Timeout,
    }
    
    // Setup cleanup
    runtime.SetFinalizer(p, (*MyPlugin).cleanup)
    return nil
}

func (p *MyPlugin) Close() error {
    p.cleanup()
    runtime.SetFinalizer(p, nil)
    return nil
}

func (p *MyPlugin) cleanup() {
    if p.client != nil {
        p.client.CloseIdleConnections()
    }
}
```

### 5. Thread Safety

```go
type ThreadSafePlugin struct {
    mu    sync.RWMutex
    cache map[string]interface{}
}

func (p *ThreadSafePlugin) Get(key string) (interface{}, bool) {
    p.mu.RLock()
    defer p.mu.RUnlock()
    
    value, exists := p.cache[key]
    return value, exists
}

func (p *ThreadSafePlugin) Set(key string, value interface{}) {
    p.mu.Lock()
    defer p.mu.Unlock()
    
    p.cache[key] = value
}
```

## Testing

### Unit Testing

```go
func TestSimpleAuthPlugin_Authenticate(t *testing.T) {
    plugin := &SimpleAuthPlugin{}
    config := map[string]interface{}{
        "api_key":    "test-key",
        "header_key": "X-API-Key",
    }
    
    err := plugin.Init(config)
    require.NoError(t, err)
    
    request := &plugin.AuthRequest{
        URL:     "https://api.example.com/data",
        Headers: make(map[string]string),
    }
    
    response, err := plugin.Authenticate(context.Background(), request)
    require.NoError(t, err)
    require.True(t, response.Success)
    require.Equal(t, "test-key", response.Headers["X-API-Key"])
}
```

### Integration Testing

```go
func TestPlugin_Integration(t *testing.T) {
    // Setup plugin manager
    manager := plugin.NewPluginManager()
    pluginInstance := &SimpleAuthPlugin{}
    
    // Register plugin
    err := manager.Register(pluginInstance)
    require.NoError(t, err)
    
    // Test with godl downloader
    downloader := godl.NewDownloader()
    err = downloader.UsePlugin(pluginInstance)
    require.NoError(t, err)
    
    // Perform test download
    ctx := context.Background()
    err = downloader.Download(ctx, "https://api.example.com/test", "/tmp/test", nil)
    require.NoError(t, err)
}
```

### Performance Testing

```go
func BenchmarkPlugin_Authenticate(b *testing.B) {
    plugin := setupTestPlugin()
    request := &plugin.AuthRequest{
        URL:     "https://api.example.com/data",
        Headers: make(map[string]string),
    }
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := plugin.Authenticate(context.Background(), request)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

## Distribution

### 1. Plugin Metadata

Create a `plugin.json` metadata file:

```json
{
    "name": "simple-auth",
    "version": "1.0.0",
    "description": "Simple API key authentication plugin for godl",
    "author": "Your Name",
    "license": "MIT",
    "type": "auth",
    "godl_version": ">=1.0.0",
    "dependencies": {},
    "configuration_schema": {
        "type": "object",
        "properties": {
            "api_key": {
                "type": "string",
                "description": "API key for authentication"
            },
            "header_key": {
                "type": "string",
                "default": "X-API-Key",
                "description": "HTTP header name for the API key"
            }
        },
        "required": ["api_key"]
    }
}
```

### 2. Plugin Registry

Upload to a plugin registry or distribute via Git:

```bash
# Tag release
git tag v1.0.0
git push origin v1.0.0

# Install from Git
godl plugin install github.com/youruser/my-godl-plugin simple-auth
```

## Examples

### Complete OAuth2 Plugin

See `examples/plugins/auth/oauth2/` for a complete OAuth2 authentication plugin implementation.

### S3 Storage Plugin

See `examples/plugins/storage/s3/` for a complete S3 storage backend plugin.

### Image Optimization Plugin

See `examples/plugins/transform/image-optimizer/` for a content transformation plugin.

## Troubleshooting

### Common Issues

1. **Plugin Not Loading**
   - Check plugin is in correct directory
   - Verify shared library build
   - Check plugin exports `Plugin` symbol

2. **Configuration Errors**
   - Validate JSON configuration syntax
   - Ensure required fields are present
   - Check data types match expected format

3. **Runtime Errors**
   - Check plugin implements all required interfaces
   - Verify resource cleanup in Close() method
   - Check thread safety for concurrent access

### Debug Mode

Enable debug logging:

```bash
GODL_DEBUG=1 godl --plugin your-plugin https://example.com/file
```

### Plugin Validation

Use built-in validation:

```bash
godl plugin validate ./your-plugin.so
```

## Resources

- [Plugin API Reference](./API_REFERENCE.md)
- [Example Plugins](../examples/plugins/)
- [Testing Guide](./TESTING.md)
- [Security Guidelines](./SECURITY.md)

For more advanced topics and API details, see the [Extending Guide](./EXTENDING.md).