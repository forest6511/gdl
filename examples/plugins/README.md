# Plugin Examples

This directory contains example plugin implementations for the gdl download tool. These examples demonstrate how to create different types of plugins and integrate them with the gdl plugin system.

## Available Plugin Examples

### Authentication Plugins

#### Simple Auth Plugin
- **Location**: `auth/simple-auth/`
- **Description**: Basic API key authentication plugin
- **Features**: HTTP header injection, configurable key names
- **Build**: `go build -buildmode=plugin -o simple-auth.so`

#### OAuth2 Plugin  
- **Location**: `auth/oauth2/`
- **Description**: OAuth2 authentication with token refresh
- **Features**: Authorization code flow, token refresh, secure token storage
- **Build**: `go build -buildmode=plugin -o oauth2.so`

#### JWT Plugin
- **Location**: `auth/jwt/`
- **Description**: JWT token-based authentication
- **Features**: Token validation, expiry handling, custom claims
- **Build**: `go build -buildmode=plugin -o jwt.so`

### Protocol Plugins

#### S3 Protocol Plugin
- **Location**: `protocol/s3/`
- **Description**: AWS S3 protocol support for `s3://` URLs
- **Features**: AWS SDK integration, credential management, multipart downloads
- **Build**: `go build -buildmode=plugin -o s3.so`

#### FTP Protocol Plugin
- **Location**: `protocol/ftp/`
- **Description**: FTP/FTPS protocol support
- **Features**: Active/passive modes, TLS support, resume capability
- **Build**: `go build -buildmode=plugin -o ftp.so`

#### Custom API Plugin
- **Location**: `protocol/custom-api/`
- **Description**: Custom REST API protocol handler
- **Features**: Pagination handling, rate limiting, custom authentication
- **Build**: `go build -buildmode=plugin -o custom-api.so`

### Storage Plugins

#### Database Storage Plugin
- **Location**: `storage/database/`
- **Description**: Store downloads in database (PostgreSQL, MySQL)
- **Features**: BLOB storage, metadata indexing, compression
- **Build**: `go build -buildmode=plugin -o database-storage.so`

#### Redis Cache Plugin
- **Location**: `storage/redis/`
- **Description**: Redis-based caching storage backend
- **Features**: TTL support, compression, clustering
- **Build**: `go build -buildmode=plugin -o redis-storage.so`

#### S3 Storage Plugin
- **Location**: `storage/s3-storage/`
- **Description**: Amazon S3 storage backend
- **Features**: Server-side encryption, versioning, lifecycle management
- **Build**: `go build -buildmode=plugin -o s3-storage.so`

### Transform Plugins

#### Compression Plugin
- **Location**: `transform/compression/`
- **Description**: Real-time compression during download
- **Features**: Multiple algorithms (gzip, zstd, lz4), streaming compression
- **Build**: `go build -buildmode=plugin -o compression.so`

#### Encryption Plugin
- **Location**: `transform/encryption/`
- **Description**: Encrypt downloads with AES-256
- **Features**: Key derivation, secure random IVs, authenticated encryption
- **Build**: `go build -buildmode=plugin -o encryption.so`

#### Image Optimization Plugin
- **Location**: `transform/image-optimizer/`
- **Description**: Optimize images during download
- **Features**: Format conversion, quality adjustment, resizing
- **Build**: `go build -buildmode=plugin -o image-optimizer.so`

### Hook Plugins

#### Logging Plugin
- **Location**: `hooks/logging/`
- **Description**: Advanced logging with multiple outputs
- **Features**: Structured logging, log rotation, remote endpoints
- **Build**: `go build -buildmode=plugin -o logging.so`

#### Webhook Plugin
- **Location**: `hooks/webhook/`
- **Description**: Send webhooks on download events
- **Features**: Event filtering, retry logic, custom payloads
- **Build**: `go build -buildmode=plugin -o webhook.so`

#### Metrics Plugin
- **Location**: `hooks/metrics/`
- **Description**: Collect and export metrics
- **Features**: Prometheus integration, custom metrics, aggregation
- **Build**: `go build -buildmode=plugin -o metrics.so`

## Plugin Development Workflow

### 1. Create Plugin Structure

```bash
mkdir -p examples/plugins/auth/my-plugin
cd examples/plugins/auth/my-plugin
```

### 2. Implement Plugin Interface

```go
package main

import (
    \"context\"
    \"github.com/forest6511/gdl/pkg/plugin\"
)

type MyPlugin struct {
    config map[string]interface{}
}

func (p *MyPlugin) Name() string {
    return \"my-plugin\"
}

func (p *MyPlugin) Version() string {
    return \"1.0.0\"
}

func (p *MyPlugin) Init(config map[string]interface{}) error {
    p.config = config
    return nil
}

func (p *MyPlugin) Close() error {
    return nil
}

func (p *MyPlugin) ValidateAccess(operation string, resource string) error {
    return nil
}

// Export plugin symbol
var Plugin plugin.AuthPlugin = &MyPlugin{}
```

### 3. Build Plugin

```bash
go build -buildmode=plugin -o my-plugin.so
```

### 4. Create Plugin Configuration

```json
{
    \"name\": \"my-plugin\",
    \"version\": \"1.0.0\",
    \"type\": \"auth\",
    \"enabled\": true,
    \"config\": {
        \"setting1\": \"value1\",
        \"setting2\": \"value2\"
    }
}
```

### 5. Test Plugin

```bash
go test ./...
```

### 6. Integration Test

```bash
# Install plugin
gdl plugin install ./my-plugin.so my-plugin

# Use plugin
gdl --plugin my-plugin https://example.com/file.zip
```

## Plugin Configuration

### Global Plugin Directory

Plugins are typically installed in:
- Linux/macOS: `~/.gdl/plugins/`
- Windows: `%APPDATA%/gdl/plugins/`

### Configuration Files

Each plugin requires a JSON configuration file:

```json
{
    \"name\": \"plugin-name\",
    \"version\": \"1.0.0\",
    \"description\": \"Plugin description\",
    \"type\": \"auth|protocol|storage|transform|hook\",
    \"enabled\": true,
    \"config\": {
        \"key\": \"value\"
    },
    \"dependencies\": [
        \"other-plugin-name\"
    ],
    \"permissions\": [
        \"network\",
        \"filesystem\"
    ]
}
```

## Testing Plugins

### Unit Testing

```bash
cd examples/plugins/auth/oauth2
go test ./...
```

### Integration Testing

```bash
# Build plugin
go build -buildmode=plugin -o oauth2.so

# Test with gdl
echo \"test config\" > config.json
gdl plugin load ./oauth2.so --config config.json
gdl --plugin oauth2 https://secure-api.example.com/file.zip
```

### Benchmark Testing

```bash
go test -bench=. ./...
```

## Plugin Management Commands

### List Available Plugins

```bash
gdl plugin list
```

### Install Plugin

```bash
gdl plugin install /path/to/plugin.so plugin-name
gdl plugin install github.com/user/plugin plugin-name
```

### Enable/Disable Plugin

```bash
gdl plugin enable plugin-name
gdl plugin disable plugin-name
```

### Plugin Information

```bash
gdl plugin info plugin-name
```

### Remove Plugin

```bash
gdl plugin remove plugin-name
```

## Best Practices

### Plugin Design
1. **Single Responsibility**: Each plugin should have one clear purpose
2. **Minimal Dependencies**: Reduce external dependencies to avoid conflicts
3. **Configuration Validation**: Validate configuration thoroughly
4. **Error Handling**: Provide clear, actionable error messages
5. **Resource Cleanup**: Always clean up resources in Close() method

### Security
1. **Input Validation**: Validate all inputs and configurations
2. **Permission Checks**: Use ValidateAccess() to check permissions
3. **Secure Defaults**: Use secure default configurations
4. **Credential Handling**: Never log or expose credentials
5. **Isolation**: Plugins should not interfere with each other

### Performance
1. **Lazy Loading**: Initialize resources only when needed
2. **Connection Pooling**: Reuse connections where possible
3. **Caching**: Cache expensive operations
4. **Memory Management**: Avoid memory leaks
5. **Concurrency**: Handle concurrent access safely

### Documentation
1. **Clear README**: Include usage examples and configuration
2. **API Documentation**: Document all public interfaces
3. **Configuration Schema**: Provide JSON schema for configuration
4. **Examples**: Include working examples
5. **Troubleshooting**: Common issues and solutions

## Contributing

To contribute a new plugin example:

1. Create the plugin in the appropriate category directory
2. Include complete implementation with tests
3. Add configuration examples
4. Update this README
5. Follow existing code style and patterns

## Resources

- [Plugin Development Guide](../../docs/PLUGIN_DEVELOPMENT.md)
- [Extension Guide](../../docs/EXTENDING.md)
- [API Reference](../../docs/API_REFERENCE.md)
- [Security Guidelines](../../docs/SECURITY.md)