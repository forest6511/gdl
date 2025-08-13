# GODL Error Reference and Troubleshooting Guide

This comprehensive guide provides detailed information about error handling, troubleshooting procedures, and best practices for the GODL download tool.

## Table of Contents

1. [Error Code Reference](#error-code-reference)
2. [Troubleshooting Guide](#troubleshooting-guide)
3. [Recovery Procedures](#recovery-procedures)
4. [Best Practices](#best-practices)
5. [Configuration for Error Handling](#configuration-for-error-handling)
6. [CLI Help and Examples](#cli-help-and-examples)

## Error Code Reference

### Overview

GODL uses a structured error system with specific error codes to help identify and resolve issues quickly. Each error includes:

- **Error Code**: A unique identifier for the error type
- **Error Message**: Human-readable description
- **Context Information**: URL, filename, bytes transferred, etc.
- **Retryable Flag**: Whether the error can be automatically retried
- **Recovery Suggestions**: Recommended actions to resolve the issue

### Error Codes

#### Network Errors

##### `network_error` - Network Connection Issues
- **Description**: General network connectivity problems
- **Common Causes**:
  - Internet connection down
  - DNS resolution failures
  - Firewall blocking connections
  - Proxy configuration issues
- **Retryable**: Yes
- **Example**:
  ```
  Error: Network connection failed
  Code: network_error
  URL: https://example.com/file.zip
  Retryable: true
  ```

##### `timeout` - Request Timeouts
- **Description**: Operation exceeded configured timeout limits
- **Common Causes**:
  - Slow network connections
  - Server overload
  - Large file downloads
  - Restrictive timeout settings
- **Retryable**: Yes
- **Timeout Types**:
  - Connect timeout (establishing connection)
  - Read timeout (receiving data)
  - Request timeout (complete operation)

#### Server Errors

##### `server_error` - HTTP 5xx Server Errors
- **Description**: Server-side errors (HTTP 500-599)
- **Common Status Codes**:
  - `500 Internal Server Error`
  - `502 Bad Gateway`
  - `503 Service Unavailable`
  - `504 Gateway Timeout`
- **Retryable**: Yes
- **Recovery**: Usually resolves automatically with retry

##### `file_not_found` - HTTP 404 File Not Found
- **Description**: Requested file does not exist on server
- **HTTP Status**: 404
- **Retryable**: No
- **Recovery**: Check URL spelling and file availability

##### `authentication_failed` - HTTP 401/403 Authentication Issues
- **Description**: Authentication or authorization failures
- **HTTP Status Codes**: 401, 403
- **Retryable**: No
- **Recovery**: Check credentials or permissions

#### Client Errors

##### `invalid_url` - Invalid URL Format
- **Description**: Malformed or unsupported URL
- **Common Issues**:
  - Missing protocol (http/https)
  - Invalid characters
  - Unsupported schemes (ftp, file, etc.)
- **Retryable**: No
- **Recovery**: Correct URL format

##### `client_error` - HTTP 4xx Client Errors
- **Description**: Client-side errors (HTTP 400-499, excluding 401, 403, 404)
- **Common Status Codes**:
  - `400 Bad Request`
  - `429 Too Many Requests`
- **Retryable**: No (except 429)

#### Storage Errors

##### `file_exists` - File Already Exists
- **Description**: Destination file exists and overwrite not enabled
- **Retryable**: No
- **Recovery Options**:
  - Use `--overwrite` flag
  - Choose different destination
  - Enable resume mode

##### `permission_denied` - File System Permission Error
- **Description**: Insufficient permissions for file operations
- **Common Causes**:
  - Read-only file system
  - Protected directories
  - Insufficient user privileges
- **Retryable**: No
- **Recovery**: Check permissions and directory access

##### `insufficient_space` - Disk Space Error
- **Description**: Insufficient disk space for download
- **Retryable**: No
- **Recovery**: Free up disk space or change destination

#### System Errors

##### `cancelled` - Operation Cancelled
- **Description**: User cancelled the operation (Ctrl+C)
- **Retryable**: No
- **Recovery**: Restart download if needed

##### `corrupted_data` - Data Integrity Error
- **Description**: Downloaded data failed integrity checks
- **Retryable**: Yes
- **Recovery**: Automatic retry with fresh download

##### `unknown` - Unknown Error
- **Description**: Unclassified or unexpected errors
- **Retryable**: No
- **Recovery**: Check logs for details

## Troubleshooting Guide

### Quick Diagnostics

#### Step 1: Check Network Connectivity
```bash
# Test basic connectivity
godl --check-network

# Test specific URL
godl --test-url "https://example.com/file.zip"

# Run with network diagnostics
godl download "https://example.com/file.zip" --diagnose
```

#### Step 2: Enable Verbose Logging
```bash
# Enable detailed error information
godl download "https://example.com/file.zip" --verbose --log-errors

# Save logs to file
godl download "https://example.com/file.zip" --log-file download.log
```

#### Step 3: Check Configuration
```bash
# Display current configuration
godl config show

# Validate configuration
godl config validate

# Reset to defaults
godl config reset
```

### Common Issues and Solutions

#### Issue: "Connection Refused" Errors
```bash
Error: Network connection failed
Code: network_error
Details: connection refused
```

**Possible Causes:**
1. Server is down
2. Port blocked by firewall
3. Incorrect URL/port

**Solutions:**
```bash
# Test with different URL
godl download "https://httpbin.org/get" 

# Check firewall settings
# Configure proxy if needed
godl config set network.proxy "http://proxy.example.com:8080"

# Try with different timeout
godl download "https://example.com/file.zip" --timeout 60s
```

#### Issue: "Timeout" Errors
```bash
Error: Download timed out
Code: timeout
Details: Request timed out after 30 seconds
```

**Solutions:**
```bash
# Increase timeout
godl download "https://example.com/file.zip" --timeout 300s

# Use smaller chunk size for slow connections
godl download "https://example.com/file.zip" --chunk-size 8192

# Enable resume for large files
godl download "https://example.com/file.zip" --resume
```

#### Issue: "File Already Exists" Errors
```bash
Error: File already exists
Code: file_exists
Filename: /downloads/file.zip
```

**Solutions:**
```bash
# Overwrite existing file
godl download "https://example.com/file.zip" --overwrite

# Resume incomplete download
godl download "https://example.com/file.zip" --resume

# Download to different location
godl download "https://example.com/file.zip" -o "/downloads/file_new.zip"
```

#### Issue: "Insufficient Disk Space" Errors
```bash
Error: Insufficient disk space
Code: insufficient_space
Available: 45MB, Required: 100MB
```

**Solutions:**
```bash
# Check available space
df -h /downloads

# Change download location
godl download "https://example.com/file.zip" -o "/tmp/file.zip"

# Clean up space and retry
rm -rf /downloads/old_files/*
```

#### Issue: "Authentication Failed" Errors
```bash
Error: Authentication or authorization failed
Code: authentication_failed
HTTP Status: 401
```

**Solutions:**
```bash
# Add authentication headers
godl download "https://example.com/file.zip" \
  --header "Authorization: Bearer your_token"

# Use basic auth
godl download "https://user:pass@example.com/file.zip"

# Check if credentials are required
curl -I "https://example.com/file.zip"
```

### Advanced Troubleshooting

#### Network Issues
```bash
# Test with different DNS servers
godl config set network.dns_servers "8.8.8.8,8.8.4.4"

# Disable TLS verification for testing
godl download "https://example.com/file.zip" --insecure

# Use different User-Agent
godl download "https://example.com/file.zip" \
  --user-agent "Mozilla/5.0 (compatible; GODL/1.0)"
```

#### Performance Issues
```bash
# Reduce concurrent downloads
godl config set network.max_concurrent_downloads 2

# Optimize buffer sizes
godl config set network.chunk_size 65536
godl config set network.buffer_size 16384

# Enable compression
godl download "https://example.com/file.zip" \
  --header "Accept-Encoding: gzip, deflate"
```

## Recovery Procedures

### Automatic Recovery

GODL includes intelligent recovery mechanisms that automatically handle many error scenarios:

#### 1. Retry with Exponential Backoff
- Automatically retries transient errors
- Uses exponential backoff to avoid overwhelming servers
- Configurable retry limits and delays

#### 2. Network Diagnostics
- Automatically detects network issues
- Provides specific recommendations
- Tests alternative connection methods

#### 3. Resume Support
- Automatically resumes interrupted downloads
- Validates partial files before resuming
- Handles server-side resume limitations

### Manual Recovery Procedures

#### For Network Errors:
1. **Check Internet Connection**
   ```bash
   ping google.com
   curl -I https://httpbin.org/get
   ```

2. **Test Alternative URLs**
   ```bash
   # Try direct IP if DNS fails
   nslookup example.com
   godl download "http://192.168.1.1/file.zip"
   ```

3. **Configure Proxy/VPN**
   ```bash
   # Set HTTP proxy
   export HTTP_PROXY=http://proxy.example.com:8080
   export HTTPS_PROXY=http://proxy.example.com:8080
   ```

#### For Server Errors:
1. **Wait and Retry**
   ```bash
   # Server might be temporarily down
   sleep 300  # Wait 5 minutes
   godl download "https://example.com/file.zip"
   ```

2. **Try Alternative Servers**
   ```bash
   # Use mirror or CDN
   godl download "https://mirror.example.com/file.zip"
   ```

#### For Storage Errors:
1. **Free Disk Space**
   ```bash
   # Check disk usage
   du -sh /downloads/*
   
   # Clean temporary files
   rm -rf /tmp/godl_*
   
   # Use different disk
   godl download "https://example.com/file.zip" -o "/mnt/external/file.zip"
   ```

2. **Fix Permissions**
   ```bash
   # Create writable directory
   mkdir -p ~/downloads
   chmod 755 ~/downloads
   
   # Download to user directory
   godl download "https://example.com/file.zip" -o "~/downloads/file.zip"
   ```

### Recovery Configuration

Configure automatic recovery behavior:

```bash
# Enable aggressive retry
godl config set retry_policy.max_retries 10
godl config set retry_policy.strategy "exponential"

# Enable recovery suggestions
godl config set error_handling.recovery_enabled true

# Enable network diagnostics
godl config set error_handling.network_diagnostics true

# Configure resume support
godl config set storage.resume_support true
```

## Best Practices

### Error Prevention

#### 1. URL Validation
```bash
# Always test URLs first
godl --test-url "https://example.com/file.zip"

# Use head request to check availability
godl head "https://example.com/file.zip"
```

#### 2. Resource Management
```bash
# Check available space before large downloads
df -h /downloads

# Set appropriate timeouts
godl config set timeouts.download_timeout "1h"

# Limit concurrent downloads
godl config set network.max_concurrent_downloads 3
```

#### 3. Network Optimization
```bash
# Use appropriate chunk size
godl config set network.chunk_size 32768  # 32KB for normal connections
godl config set network.chunk_size 8192   # 8KB for slow connections

# Configure reasonable timeouts
godl config set timeouts.connect_timeout "10s"
godl config set timeouts.read_timeout "30s"
```

### Error Handling Configuration

#### 1. Logging Setup
```bash
# Enable comprehensive logging
godl config set error_handling.log_errors true
godl config set error_handling.verbose_errors true
godl config set error_handling.log_file "~/.godl/errors.log"
```

#### 2. Output Configuration
```bash
# Configure structured error output
godl config set error_handling.error_format "structured"
godl config set output_format.format "json"

# Enable colored output for better readability
godl config set output_format.color true
```

#### 3. Retry Policy Setup
```bash
# Configure retry behavior
godl config set retry_policy.max_retries 5
godl config set retry_policy.base_delay "1s"
godl config set retry_policy.max_delay "60s"
godl config set retry_policy.backoff_factor 2.0
godl config set retry_policy.jitter true
```

### Monitoring and Maintenance

#### 1. Log Rotation
```bash
# Set up log rotation
# Add to crontab: 0 0 * * * logrotate ~/.godl/logrotate.conf

# logrotate.conf:
~/.godl/errors.log {
    daily
    rotate 7
    compress
    missingok
    notifempty
}
```

#### 2. Health Checks
```bash
# Regular connectivity tests
godl --health-check

# Configuration validation
godl config validate

# Clean temporary files
godl --cleanup-temp
```

### Performance Optimization

#### 1. Concurrent Downloads
```bash
# Optimize for your connection
# Fast connection (100+ Mbps):
godl config set network.max_concurrent_downloads 8
godl config set network.chunk_size 65536

# Slow connection (<10 Mbps):
godl config set network.max_concurrent_downloads 2
godl config set network.chunk_size 16384
```

#### 2. Memory Usage
```bash
# Optimize for low memory systems
godl config set network.buffer_size 4096
godl config set network.chunk_size 16384

# Optimize for high memory systems
godl config set network.buffer_size 32768
godl config set network.chunk_size 131072
```

## Configuration for Error Handling

### Complete Error Handling Configuration Example

```json
{
  "version": "1.0",
  "retry_policy": {
    "max_retries": 5,
    "base_delay": "2s",
    "max_delay": "120s",
    "backoff_factor": 2.0,
    "jitter": true,
    "strategy": "exponential",
    "retryable_errors": [
      "network_error",
      "timeout",
      "server_error",
      "corrupted_data"
    ],
    "non_retryable_errors": [
      "invalid_url",
      "file_exists",
      "permission_denied",
      "authentication_failed",
      "insufficient_space"
    ]
  },
  "error_handling": {
    "verbose_errors": true,
    "show_stack_trace": false,
    "log_errors": true,
    "log_file": "~/.godl/errors.log",
    "fail_fast": false,
    "error_format": "structured",
    "recovery_enabled": true,
    "network_diagnostics": true
  },
  "output_format": {
    "format": "text",
    "pretty": true,
    "color": true,
    "show_progress": true,
    "quiet": false,
    "verbose": true,
    "timestamp_format": "2006-01-02 15:04:05",
    "log_level": "info"
  }
}
```

### Environment Variables

Override configuration with environment variables:

```bash
# Retry configuration
export GODL_MAX_RETRIES=10
export GODL_RETRY_STRATEGY=exponential

# Error handling
export GODL_VERBOSE_ERRORS=true
export GODL_LOG_ERRORS=true
export GODL_ERROR_LOG_FILE=/var/log/godl.log

# Output formatting
export GODL_OUTPUT_FORMAT=json
export GODL_COLOR=true
export GODL_VERBOSE=true

# Timeouts
export GODL_CONNECT_TIMEOUT=15s
export GODL_REQUEST_TIMEOUT=300s
export GODL_DOWNLOAD_TIMEOUT=3600s
```

## CLI Help and Examples

See [CLI Help Guide](../cli/help.md) for detailed command-line help and examples.

## Getting Help

If you encounter issues not covered in this guide:

1. **Enable verbose logging** and check error details
2. **Search existing issues** on GitHub
3. **Create a new issue** with:
   - Error message and code
   - Command used
   - Configuration file
   - Log output
   - System information

For more information, see:
- [Configuration Guide](../config/README.md)
- [API Reference](../api/README.md)
- [Examples](../examples/README.md)