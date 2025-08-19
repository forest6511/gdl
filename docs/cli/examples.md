# GDL CLI Examples and Usage Guide

This guide provides comprehensive examples for using the GDL command-line tool in various scenarios.

## Table of Contents

1. [Basic Usage](#basic-usage)
2. [Advanced Downloads](#advanced-downloads)
3. [Authentication](#authentication)
4. [Batch Operations](#batch-operations)
5. [Error Handling](#error-handling)
6. [Configuration](#configuration)
7. [Troubleshooting](#troubleshooting)
8. [Performance Optimization](#performance-optimization)

## Basic Usage

### Simple Download
Download a file to the current directory:
```bash
gdl download "https://example.com/file.zip"
```

### Download to Specific Location
Specify the output file path:
```bash
gdl download "https://example.com/file.zip" -o "/downloads/myfile.zip"
gdl download "https://example.com/file.zip" --output "/downloads/myfile.zip"
```

### Download to Directory
Download to a specific directory (filename auto-detected):
```bash
gdl download "https://example.com/file.zip" --output-dir "/downloads"
```

### Create Parent Directories
Automatically create parent directories if they don't exist:
```bash
gdl download "https://example.com/file.zip" -o "/new/path/file.zip" --create-dirs
```

## Advanced Downloads

### Resume Interrupted Downloads
Resume a partially downloaded file:
```bash
gdl download "https://example.com/largefile.zip" --resume
```

### Concurrent Downloads
Use multiple connections for faster downloads:
```bash
# Use 4 concurrent connections
gdl download "https://example.com/file.zip" --concurrent 4

# Set global concurrent limit
gdl config set network.max_concurrent_downloads 6
```

### Custom Timeouts
Configure timeout values for slow connections:
```bash
# Set overall timeout to 10 minutes
gdl download "https://example.com/file.zip" --timeout 600s

# Configure specific timeouts
gdl download "https://example.com/file.zip" \
  --connect-timeout 30s \
  --read-timeout 120s
```

### Retry Configuration
Configure retry behavior for unreliable connections:
```bash
# Retry up to 10 times with exponential backoff
gdl download "https://example.com/file.zip" --retry 10 --retry-delay 2s

# Custom retry strategy
gdl download "https://example.com/file.zip" \
  --retry 5 \
  --retry-strategy exponential \
  --retry-max-delay 60s
```

### Overwrite Existing Files
Overwrite files that already exist:
```bash
gdl download "https://example.com/file.zip" --overwrite
```

## Authentication

### Basic Authentication
Use username and password:
```bash
# URL-embedded credentials
gdl download "https://username:password@example.com/file.zip"

# Using headers
gdl download "https://example.com/file.zip" \
  --header "Authorization: Basic $(echo -n 'username:password' | base64)"
```

### Bearer Token Authentication
Use API tokens:
```bash
# Bearer token
gdl download "https://api.example.com/file.zip" \
  --header "Authorization: Bearer your_token_here"

# Token from file
gdl download "https://api.example.com/file.zip" \
  --header "Authorization: Bearer $(cat ~/.tokens/api_token)"
```

### API Key Authentication
Use API keys:
```bash
# API key in header
gdl download "https://api.example.com/file.zip" \
  --header "X-API-Key: your_api_key"

# Multiple headers
gdl download "https://api.example.com/file.zip" \
  --header "X-API-Key: your_api_key" \
  --header "X-Client-Version: 1.0"
```

### Custom User Agent
Set a custom User-Agent string:
```bash
gdl download "https://example.com/file.zip" \
  --user-agent "Mozilla/5.0 (compatible; MyBot/1.0)"
```

## Batch Operations

### Download Multiple Files
Download multiple URLs:
```bash
# Multiple URLs as arguments
gdl download \
  "https://example.com/file1.zip" \
  "https://example.com/file2.zip" \
  "https://example.com/file3.zip"
```

### Download from File List
Download URLs from a text file:
```bash
# Create URL list file
cat > urls.txt << 'EOF'
https://example.com/file1.zip
https://example.com/file2.zip
https://example.com/file3.zip
EOF

# Download all URLs
gdl download -f urls.txt --output-dir "/downloads"

# With concurrency limit
gdl download -f urls.txt --concurrent 3 --output-dir "/downloads"
```

### Batch Download with Individual Settings
Different settings per URL (using config files):
```bash
# Create individual configs
echo '{"retry_policy": {"max_retries": 10}}' > config1.json
echo '{"timeouts": {"download_timeout": "1h"}}' > config2.json

# Download with specific configs
gdl download "https://unreliable.com/file.zip" --config config1.json
gdl download "https://slow.com/largefile.zip" --config config2.json
```

## Error Handling

### Verbose Output and Logging
Enable detailed output for debugging:
```bash
# Verbose output to console
gdl download "https://example.com/file.zip" --verbose

# Save logs to file
gdl download "https://example.com/file.zip" --log-file download.log

# Both verbose and logging
gdl download "https://example.com/file.zip" --verbose --log-file debug.log
```

### Structured Error Output
Get machine-readable error information:
```bash
# JSON output format
gdl download "https://example.com/file.zip" --format json

# Include error details
gdl download "https://example.com/file.zip" --format json --include-errors
```

### Error Recovery
Handle specific error scenarios:
```bash
# Continue on non-fatal errors
gdl download -f urls.txt --continue-on-error

# Fail fast (stop on first error)
gdl download -f urls.txt --fail-fast

# Custom error handling
gdl download "https://example.com/file.zip" \
  --on-error "retry:3,delay:5s" \
  --on-fatal-error "log,exit"
```

## Configuration

### View Current Configuration
```bash
# Show all configuration
gdl config show

# Show specific section
gdl config show retry_policy
gdl config show timeouts

# Show in different formats
gdl config show --format json
gdl config show --format yaml
```

### Modify Configuration
```bash
# Set retry policy
gdl config set retry_policy.max_retries 10
gdl config set retry_policy.strategy "exponential"

# Set timeout values
gdl config set timeouts.download_timeout "30m"
gdl config set timeouts.connect_timeout "15s"

# Set network options
gdl config set network.max_concurrent_downloads 8
gdl config set network.chunk_size 65536
```

### Configuration Profiles
Use different configurations for different scenarios:
```bash
# Create configuration profiles
gdl config create-profile "slow-network" \
  --set retry_policy.max_retries=10 \
  --set network.chunk_size=8192 \
  --set timeouts.read_timeout="300s"

gdl config create-profile "fast-network" \
  --set network.max_concurrent_downloads=12 \
  --set network.chunk_size=131072

# Use specific profile
gdl download "https://example.com/file.zip" --profile "slow-network"
```

### Environment-Based Configuration
Use environment variables to override settings:
```bash
# Set environment variables
export GDL_MAX_RETRIES=5
export GDL_CHUNK_SIZE=32768
export GDL_OUTPUT_DIR="/downloads"

# These will override config file settings
gdl download "https://example.com/file.zip"
```

## Troubleshooting

### Network Diagnostics
Test and diagnose network issues:
```bash
# Test network connectivity
gdl --check-network

# Test specific URL
gdl head "https://example.com/file.zip"

# Network diagnostics with download
gdl download "https://example.com/file.zip" --diagnose
```

### Connection Testing
Test different connection methods:
```bash
# Test with IPv4 only
gdl download "https://example.com/file.zip" --ipv4-only

# Test with different DNS servers
gdl download "https://example.com/file.zip" --dns "8.8.8.8,8.8.4.4"

# Test with proxy
gdl download "https://example.com/file.zip" --proxy "http://proxy:8080"
```

### SSL/TLS Issues
Handle SSL certificate problems:
```bash
# Ignore SSL certificate errors (for testing only)
gdl download "https://self-signed.badssl.com/file.zip" --insecure

# Use specific CA bundle
gdl download "https://example.com/file.zip" --ca-bundle "/path/to/ca-bundle.pem"

# Show SSL certificate information
gdl head "https://example.com/file.zip" --show-cert
```

### Performance Debugging
Analyze download performance:
```bash
# Show transfer statistics
gdl download "https://example.com/file.zip" --stats

# Benchmark download speed
gdl --speed-test "https://example.com/file.zip"

# Profile download performance
gdl download "https://example.com/file.zip" --profile-performance
```

### Server Compatibility
Handle different server behaviors:
```bash
# Disable range requests (for servers that don't support them)
gdl download "https://example.com/file.zip" --no-ranges

# Limit redirects
gdl download "https://example.com/file.zip" --max-redirects 3

# Handle non-standard headers
gdl download "https://example.com/file.zip" \
  --header "Accept: */*" \
  --header "Connection: close"
```

## Performance Optimization

### High-Speed Connections (100+ Mbps)
Optimize for fast networks:
```bash
# High-performance configuration
gdl config set network.max_concurrent_downloads 12
gdl config set network.chunk_size 131072
gdl config set network.buffer_size 32768

# Download with optimized settings
gdl download "https://example.com/file.zip" \
  --concurrent 8 \
  --chunk-size 131072
```

### Slow Connections (<10 Mbps)
Optimize for slow or unreliable networks:
```bash
# Slow network configuration
gdl config set network.max_concurrent_downloads 2
gdl config set network.chunk_size 16384
gdl config set retry_policy.max_retries 10
gdl config set timeouts.read_timeout "300s"

# Download with conservative settings
gdl download "https://example.com/file.zip" \
  --concurrent 1 \
  --chunk-size 8192 \
  --retry 15
```

### Memory-Constrained Systems
Optimize for low memory usage:
```bash
# Memory-efficient configuration
gdl config set network.buffer_size 4096
gdl config set network.chunk_size 16384
gdl config set network.max_concurrent_downloads 2

# Download with minimal memory usage
gdl download "https://example.com/largefile.zip" \
  --chunk-size 8192 \
  --concurrent 1
```

### Large Files (>1GB)
Optimize for very large downloads:
```bash
# Large file configuration
gdl config set timeouts.download_timeout "4h"
gdl config set storage.resume_support true
gdl config set retry_policy.max_retries 20

# Download large file with resume and extended timeouts
gdl download "https://example.com/10gb-file.zip" \
  --resume \
  --timeout 14400s \
  --retry 20 \
  --chunk-size 65536
```

## Real-World Examples

### Downloading Software Releases
```bash
# Download latest release from GitHub
gdl download "https://github.com/user/repo/releases/latest/download/app.zip" \
  -o "~/Downloads/app-latest.zip" \
  --resume

# Download specific version with authentication
gdl download "https://github.com/private/repo/releases/download/v1.0.0/app.zip" \
  --header "Authorization: token ghp_your_token" \
  -o "~/Downloads/app-v1.0.0.zip"
```

### Media Files
```bash
# Download video with resume support
gdl download "https://example.com/video.mp4" \
  --resume \
  --timeout 1800s \
  -o "~/Videos/video.mp4"

# Download playlist/multiple media files
cat > media_urls.txt << 'EOF'
https://example.com/video1.mp4
https://example.com/video2.mp4
https://example.com/audio.mp3
EOF

gdl download -f media_urls.txt \
  --output-dir "~/Media" \
  --concurrent 2 \
  --resume
```

### Dataset Downloads
```bash
# Download research dataset
gdl download "https://data.example.com/dataset.tar.gz" \
  -o "~/Research/dataset.tar.gz" \
  --timeout 7200s \
  --resume \
  --verify-checksum "sha256:abc123..."

# Download and extract automatically
gdl download "https://data.example.com/dataset.tar.gz" \
  -o "~/Research/dataset.tar.gz" \
  --resume \
  --extract \
  --extract-to "~/Research/data/"
```

### Corporate/Enterprise Usage
```bash
# Download through corporate proxy
gdl download "https://vendor.com/software.zip" \
  --proxy "http://proxy.company.com:8080" \
  --header "User-Agent: Company-Downloader/1.0" \
  --ca-bundle "/etc/ssl/corporate-ca.pem" \
  -o "/opt/software/software.zip"

# Batch download with logging for audit
gdl download -f software_urls.txt \
  --output-dir "/opt/software" \
  --log-file "/var/log/gdl/downloads.log" \
  --format json \
  --concurrent 3
```

## Getting Help

### Built-in Help
```bash
# General help
gdl --help
gdl help

# Command-specific help
gdl help download
gdl download --help

# Configuration help
gdl help config
gdl config --help
```

### Context-Sensitive Help
```bash
# Get help for specific error
gdl help error network_error

# Troubleshooting guides
gdl help troubleshoot slow-downloads
gdl help troubleshoot connection-issues

# Example commands for scenarios
gdl examples batch-download
gdl examples authentication
```

### Diagnostic Commands
```bash
# System information
gdl --version
gdl --system-info

# Configuration validation
gdl config validate
gdl config doctor

# Network diagnostics
gdl --check-network
gdl --test-dns
```

For more detailed information, see:
- [Error Reference Guide](../errors/README.md)
- [Configuration Guide](../config/README.md)
- [Troubleshooting Guide](../troubleshooting/README.md)