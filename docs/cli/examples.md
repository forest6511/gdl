# GODL CLI Examples and Usage Guide

This guide provides comprehensive examples for using the GODL command-line tool in various scenarios.

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
godl download "https://example.com/file.zip"
```

### Download to Specific Location
Specify the output file path:
```bash
godl download "https://example.com/file.zip" -o "/downloads/myfile.zip"
godl download "https://example.com/file.zip" --output "/downloads/myfile.zip"
```

### Download to Directory
Download to a specific directory (filename auto-detected):
```bash
godl download "https://example.com/file.zip" --output-dir "/downloads"
```

### Create Parent Directories
Automatically create parent directories if they don't exist:
```bash
godl download "https://example.com/file.zip" -o "/new/path/file.zip" --create-dirs
```

## Advanced Downloads

### Resume Interrupted Downloads
Resume a partially downloaded file:
```bash
godl download "https://example.com/largefile.zip" --resume
```

### Concurrent Downloads
Use multiple connections for faster downloads:
```bash
# Use 4 concurrent connections
godl download "https://example.com/file.zip" --concurrent 4

# Set global concurrent limit
godl config set network.max_concurrent_downloads 6
```

### Custom Timeouts
Configure timeout values for slow connections:
```bash
# Set overall timeout to 10 minutes
godl download "https://example.com/file.zip" --timeout 600s

# Configure specific timeouts
godl download "https://example.com/file.zip" \
  --connect-timeout 30s \
  --read-timeout 120s
```

### Retry Configuration
Configure retry behavior for unreliable connections:
```bash
# Retry up to 10 times with exponential backoff
godl download "https://example.com/file.zip" --retry 10 --retry-delay 2s

# Custom retry strategy
godl download "https://example.com/file.zip" \
  --retry 5 \
  --retry-strategy exponential \
  --retry-max-delay 60s
```

### Overwrite Existing Files
Overwrite files that already exist:
```bash
godl download "https://example.com/file.zip" --overwrite
```

## Authentication

### Basic Authentication
Use username and password:
```bash
# URL-embedded credentials
godl download "https://username:password@example.com/file.zip"

# Using headers
godl download "https://example.com/file.zip" \
  --header "Authorization: Basic $(echo -n 'username:password' | base64)"
```

### Bearer Token Authentication
Use API tokens:
```bash
# Bearer token
godl download "https://api.example.com/file.zip" \
  --header "Authorization: Bearer your_token_here"

# Token from file
godl download "https://api.example.com/file.zip" \
  --header "Authorization: Bearer $(cat ~/.tokens/api_token)"
```

### API Key Authentication
Use API keys:
```bash
# API key in header
godl download "https://api.example.com/file.zip" \
  --header "X-API-Key: your_api_key"

# Multiple headers
godl download "https://api.example.com/file.zip" \
  --header "X-API-Key: your_api_key" \
  --header "X-Client-Version: 1.0"
```

### Custom User Agent
Set a custom User-Agent string:
```bash
godl download "https://example.com/file.zip" \
  --user-agent "Mozilla/5.0 (compatible; MyBot/1.0)"
```

## Batch Operations

### Download Multiple Files
Download multiple URLs:
```bash
# Multiple URLs as arguments
godl download \
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
godl download -f urls.txt --output-dir "/downloads"

# With concurrency limit
godl download -f urls.txt --concurrent 3 --output-dir "/downloads"
```

### Batch Download with Individual Settings
Different settings per URL (using config files):
```bash
# Create individual configs
echo '{"retry_policy": {"max_retries": 10}}' > config1.json
echo '{"timeouts": {"download_timeout": "1h"}}' > config2.json

# Download with specific configs
godl download "https://unreliable.com/file.zip" --config config1.json
godl download "https://slow.com/largefile.zip" --config config2.json
```

## Error Handling

### Verbose Output and Logging
Enable detailed output for debugging:
```bash
# Verbose output to console
godl download "https://example.com/file.zip" --verbose

# Save logs to file
godl download "https://example.com/file.zip" --log-file download.log

# Both verbose and logging
godl download "https://example.com/file.zip" --verbose --log-file debug.log
```

### Structured Error Output
Get machine-readable error information:
```bash
# JSON output format
godl download "https://example.com/file.zip" --format json

# Include error details
godl download "https://example.com/file.zip" --format json --include-errors
```

### Error Recovery
Handle specific error scenarios:
```bash
# Continue on non-fatal errors
godl download -f urls.txt --continue-on-error

# Fail fast (stop on first error)
godl download -f urls.txt --fail-fast

# Custom error handling
godl download "https://example.com/file.zip" \
  --on-error "retry:3,delay:5s" \
  --on-fatal-error "log,exit"
```

## Configuration

### View Current Configuration
```bash
# Show all configuration
godl config show

# Show specific section
godl config show retry_policy
godl config show timeouts

# Show in different formats
godl config show --format json
godl config show --format yaml
```

### Modify Configuration
```bash
# Set retry policy
godl config set retry_policy.max_retries 10
godl config set retry_policy.strategy "exponential"

# Set timeout values
godl config set timeouts.download_timeout "30m"
godl config set timeouts.connect_timeout "15s"

# Set network options
godl config set network.max_concurrent_downloads 8
godl config set network.chunk_size 65536
```

### Configuration Profiles
Use different configurations for different scenarios:
```bash
# Create configuration profiles
godl config create-profile "slow-network" \
  --set retry_policy.max_retries=10 \
  --set network.chunk_size=8192 \
  --set timeouts.read_timeout="300s"

godl config create-profile "fast-network" \
  --set network.max_concurrent_downloads=12 \
  --set network.chunk_size=131072

# Use specific profile
godl download "https://example.com/file.zip" --profile "slow-network"
```

### Environment-Based Configuration
Use environment variables to override settings:
```bash
# Set environment variables
export GODL_MAX_RETRIES=5
export GODL_CHUNK_SIZE=32768
export GODL_OUTPUT_DIR="/downloads"

# These will override config file settings
godl download "https://example.com/file.zip"
```

## Troubleshooting

### Network Diagnostics
Test and diagnose network issues:
```bash
# Test network connectivity
godl --check-network

# Test specific URL
godl head "https://example.com/file.zip"

# Network diagnostics with download
godl download "https://example.com/file.zip" --diagnose
```

### Connection Testing
Test different connection methods:
```bash
# Test with IPv4 only
godl download "https://example.com/file.zip" --ipv4-only

# Test with different DNS servers
godl download "https://example.com/file.zip" --dns "8.8.8.8,8.8.4.4"

# Test with proxy
godl download "https://example.com/file.zip" --proxy "http://proxy:8080"
```

### SSL/TLS Issues
Handle SSL certificate problems:
```bash
# Ignore SSL certificate errors (for testing only)
godl download "https://self-signed.badssl.com/file.zip" --insecure

# Use specific CA bundle
godl download "https://example.com/file.zip" --ca-bundle "/path/to/ca-bundle.pem"

# Show SSL certificate information
godl head "https://example.com/file.zip" --show-cert
```

### Performance Debugging
Analyze download performance:
```bash
# Show transfer statistics
godl download "https://example.com/file.zip" --stats

# Benchmark download speed
godl --speed-test "https://example.com/file.zip"

# Profile download performance
godl download "https://example.com/file.zip" --profile-performance
```

### Server Compatibility
Handle different server behaviors:
```bash
# Disable range requests (for servers that don't support them)
godl download "https://example.com/file.zip" --no-ranges

# Limit redirects
godl download "https://example.com/file.zip" --max-redirects 3

# Handle non-standard headers
godl download "https://example.com/file.zip" \
  --header "Accept: */*" \
  --header "Connection: close"
```

## Performance Optimization

### High-Speed Connections (100+ Mbps)
Optimize for fast networks:
```bash
# High-performance configuration
godl config set network.max_concurrent_downloads 12
godl config set network.chunk_size 131072
godl config set network.buffer_size 32768

# Download with optimized settings
godl download "https://example.com/file.zip" \
  --concurrent 8 \
  --chunk-size 131072
```

### Slow Connections (<10 Mbps)
Optimize for slow or unreliable networks:
```bash
# Slow network configuration
godl config set network.max_concurrent_downloads 2
godl config set network.chunk_size 16384
godl config set retry_policy.max_retries 10
godl config set timeouts.read_timeout "300s"

# Download with conservative settings
godl download "https://example.com/file.zip" \
  --concurrent 1 \
  --chunk-size 8192 \
  --retry 15
```

### Memory-Constrained Systems
Optimize for low memory usage:
```bash
# Memory-efficient configuration
godl config set network.buffer_size 4096
godl config set network.chunk_size 16384
godl config set network.max_concurrent_downloads 2

# Download with minimal memory usage
godl download "https://example.com/largefile.zip" \
  --chunk-size 8192 \
  --concurrent 1
```

### Large Files (>1GB)
Optimize for very large downloads:
```bash
# Large file configuration
godl config set timeouts.download_timeout "4h"
godl config set storage.resume_support true
godl config set retry_policy.max_retries 20

# Download large file with resume and extended timeouts
godl download "https://example.com/10gb-file.zip" \
  --resume \
  --timeout 14400s \
  --retry 20 \
  --chunk-size 65536
```

## Real-World Examples

### Downloading Software Releases
```bash
# Download latest release from GitHub
godl download "https://github.com/user/repo/releases/latest/download/app.zip" \
  -o "~/Downloads/app-latest.zip" \
  --resume

# Download specific version with authentication
godl download "https://github.com/private/repo/releases/download/v1.0.0/app.zip" \
  --header "Authorization: token ghp_your_token" \
  -o "~/Downloads/app-v1.0.0.zip"
```

### Media Files
```bash
# Download video with resume support
godl download "https://example.com/video.mp4" \
  --resume \
  --timeout 1800s \
  -o "~/Videos/video.mp4"

# Download playlist/multiple media files
cat > media_urls.txt << 'EOF'
https://example.com/video1.mp4
https://example.com/video2.mp4
https://example.com/audio.mp3
EOF

godl download -f media_urls.txt \
  --output-dir "~/Media" \
  --concurrent 2 \
  --resume
```

### Dataset Downloads
```bash
# Download research dataset
godl download "https://data.example.com/dataset.tar.gz" \
  -o "~/Research/dataset.tar.gz" \
  --timeout 7200s \
  --resume \
  --verify-checksum "sha256:abc123..."

# Download and extract automatically
godl download "https://data.example.com/dataset.tar.gz" \
  -o "~/Research/dataset.tar.gz" \
  --resume \
  --extract \
  --extract-to "~/Research/data/"
```

### Corporate/Enterprise Usage
```bash
# Download through corporate proxy
godl download "https://vendor.com/software.zip" \
  --proxy "http://proxy.company.com:8080" \
  --header "User-Agent: Company-Downloader/1.0" \
  --ca-bundle "/etc/ssl/corporate-ca.pem" \
  -o "/opt/software/software.zip"

# Batch download with logging for audit
godl download -f software_urls.txt \
  --output-dir "/opt/software" \
  --log-file "/var/log/godl/downloads.log" \
  --format json \
  --concurrent 3
```

## Getting Help

### Built-in Help
```bash
# General help
godl --help
godl help

# Command-specific help
godl help download
godl download --help

# Configuration help
godl help config
godl config --help
```

### Context-Sensitive Help
```bash
# Get help for specific error
godl help error network_error

# Troubleshooting guides
godl help troubleshoot slow-downloads
godl help troubleshoot connection-issues

# Example commands for scenarios
godl examples batch-download
godl examples authentication
```

### Diagnostic Commands
```bash
# System information
godl --version
godl --system-info

# Configuration validation
godl config validate
godl config doctor

# Network diagnostics
godl --check-network
godl --test-dns
```

For more detailed information, see:
- [Error Reference Guide](../errors/README.md)
- [Configuration Guide](../config/README.md)
- [Troubleshooting Guide](../troubleshooting/README.md)