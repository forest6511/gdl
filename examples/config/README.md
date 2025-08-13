# Configuration Examples

This directory contains example configuration files for godl (Go Downloader) showing different usage patterns and environments.

## Configuration Files

### `basic-config.yaml`
A minimal configuration file with essential settings:
- Simple retry policy
- Basic error handling
- Default storage settings
- Single authentication plugin
- Essential middleware

**Use case**: Quick start, simple downloads, learning the system.

### `full-config.yaml`
A comprehensive configuration showcasing all available features:
- Advanced retry policies with jitter
- Complete error handling with recovery
- Multiple plugins (auth, transform, storage)
- Full middleware chain
- Event hooks for all stages
- Security and networking options

**Use case**: Understanding all available options, complex enterprise setups.

### `development.yaml`
Optimized for development and debugging:
- Verbose logging and debug output
- Short timeouts for quick feedback
- Development-friendly error messages
- Mock authentication
- Local storage paths

**Use case**: Local development, testing, debugging plugins.

### `production.yaml`
Production-ready configuration with security and reliability focus:
- Conservative timeouts and retry policies
- JSON logging for log aggregation
- Rate limiting and security validations
- Environment variable usage for secrets
- Comprehensive monitoring hooks

**Use case**: Production deployments, enterprise environments.

## Configuration Structure

### Core Sections

#### `version`
Configuration schema version (e.g., "1.0")

#### `retry_policy`
Defines how failed operations are retried:
- `max_retries`: Maximum retry attempts
- `base_delay`: Initial delay between retries
- `strategy`: Retry strategy (exponential, linear, constant)
- `jitter`: Adds randomness to prevent thundering herd

#### `error_handling`
Controls error reporting and recovery:
- `verbose_errors`: Detailed error messages
- `log_errors`: Enable error logging
- `recovery_enabled`: Intelligent recovery suggestions

#### `output_format`
Controls output formatting:
- `format`: Output format (text, json, yaml)
- `show_progress`: Enable progress bars
- `color`: Colored output
- `log_level`: Minimum log level

#### `network`
Network-related settings:
- `max_concurrent_connections`: Connection pool size
- `proxy_url`: HTTP proxy configuration
- `user_agent`: Custom User-Agent header

#### `storage`
Storage configuration:
- `default_download_dir`: Default download directory
- `resume_support`: Enable resume functionality
- `create_dirs`: Auto-create directories

### Plugin System

#### `plugins`
Array of plugin configurations:
- `name`: Plugin identifier
- `type`: Plugin type (auth, transform, storage, protocol)
- `enabled`: Enable/disable the plugin
- `priority`: Execution order
- `settings`: Plugin-specific configuration

#### `middleware`
Middleware chain configuration:
- Executes in priority order
- Each middleware can modify requests/responses
- Common middleware: logging, auth, rate limiting, metrics

#### `hooks`
Event hooks for different stages:
- `pre_download`: Before download starts
- `post_download`: After download completes
- `on_error`: When errors occur
- `on_retry`: During retry attempts

## Environment Variables

Configuration files support environment variable substitution using `${VARIABLE_NAME}` syntax:

```yaml
plugins:
  - name: "oauth2"
    settings:
      client_id: "${OAUTH_CLIENT_ID}"
      client_secret: "${OAUTH_CLIENT_SECRET}"
```

## Security Best Practices

### 1. Secrets Management
- Never commit secrets to version control
- Use environment variables for sensitive data
- Consider using secret management services
- Rotate credentials regularly

### 2. Network Security
- Configure appropriate proxy settings
- Use HTTPS where possible
- Validate SSL certificates in production
- Implement proper authentication

### 3. File System Security
- Use appropriate file permissions
- Validate download paths
- Implement virus scanning
- Monitor disk usage

## Plugin Development

See the `examples/plugins/` directory for plugin implementation examples:
- `auth/oauth2/`: OAuth2 authentication plugin
- `transform/image-optimizer/`: Image optimization plugin
- `storage/gcs/`: Google Cloud Storage plugin

## Usage Examples

### Basic Usage
```bash
godl --config examples/config/basic-config.yaml <url>
```

### Development Mode
```bash
godl --config examples/config/development.yaml --debug <url>
```

### Production Deployment
```bash
export OAUTH_CLIENT_ID="your-client-id"
export OAUTH_CLIENT_SECRET="your-client-secret"
export S3_BUCKET="your-bucket"
godl --config /etc/godl/production.yaml <url>
```

### Environment-Specific Overrides
```bash
# Override log level for debugging
LOG_LEVEL=debug godl --config production.yaml <url>

# Use different storage location
DOWNLOAD_DIR=/tmp/downloads godl --config basic-config.yaml <url>
```

## Configuration Validation

The configuration system includes validation for:
- Required fields
- Valid enumeration values
- Proper data types
- Plugin compatibility
- Security settings

Invalid configurations will result in startup errors with detailed messages indicating what needs to be fixed.

## Extending Configuration

To add new configuration options:
1. Update the configuration structures in `pkg/config/`
2. Add validation logic
3. Update default values
4. Add example usage to configuration files
5. Update this documentation

## Troubleshooting

### Common Issues

#### "Plugin not found"
- Check plugin path is correct
- Ensure plugin file exists and is executable
- Verify plugin implements required interfaces

#### "Authentication failed"
- Verify environment variables are set
- Check credentials are valid
- Ensure auth plugin is enabled

#### "Permission denied"
- Check file system permissions
- Verify download directory is writable
- Ensure user has appropriate access

#### "Network timeout"
- Increase timeout values in configuration
- Check network connectivity
- Verify proxy settings if applicable

For more troubleshooting information, see the main documentation.