# CLI Reference

Complete command-line interface documentation for gdl.

## Table of Contents

- [Installation](#installation)
- [Basic Usage](#basic-usage)
- [Command Options](#command-options)
- [Examples](#examples)
- [Configuration](#configuration)
- [Exit Codes](#exit-codes)

## Installation

### Using go install

```bash
go install github.com/forest6511/gdl/cmd/gdl@latest
```

### Building from source

```bash
git clone https://github.com/forest6511/gdl.git
cd gdl
go build -o gdl ./cmd/gdl/
sudo mv gdl /usr/local/bin/
```

## Basic Usage

```bash
gdl [OPTIONS] URL
```

### Simple download

```bash
gdl https://example.com/file.zip
```

### Specify output filename

```bash
gdl -o myfile.zip https://example.com/file.zip
```

### Show help

```bash
gdl --help
```

## Command Options

### Output Options

| Flag | Long Form | Description | Default |
|------|-----------|-------------|---------|
| `-o` | `--output` | Output filename | Extract from URL |
| `-f` | `--force` | Overwrite existing files | false |
| | `--create-dirs` | Create parent directories if needed | false |

### Connection Options

| Flag | Long Form | Description | Default |
|------|-----------|-------------|---------|
| `-c` | `--concurrent` | Number of concurrent connections | 4 |
| | `--chunk-size` | Chunk size for concurrent downloads | auto |
| | `--max-rate` | Maximum download rate (e.g., 1MB/s, 500k) | unlimited |
| | `--no-concurrent` | Force single-threaded download | false |
| | `--resume` | Resume partial downloads if supported | false |
| | `--no-resume` | Disable resume functionality | false |
| | `--continue-partial` | Continue partial downloads | false |

### Network Options

| Flag | Long Form | Description | Default |
|------|-----------|-------------|---------|
| | `--timeout` | Download timeout | 30m |
| | `--retry` | Number of retry attempts | 3 |
| | `--retry-delay` | Delay between retries | 1s |
| | `--max-redirects` | Maximum number of redirects | 10 |
| `-k` | `--insecure` | Skip SSL certificate verification | false |
| | `--proxy` | HTTP proxy URL | none |
| | `--user-agent` | Custom User-Agent string | gdl/version |

### Header Options

| Flag | Long Form | Description | Default |
|------|-----------|-------------|---------|
| `-H` | `--header` | Add custom header (repeatable) | none |

### Display Options

| Flag | Long Form | Description | Default |
|------|-----------|-------------|---------|
| `-q` | `--quiet` | Quiet mode (no progress output) | false |
| `-v` | `--verbose` | Verbose output | false |
| | `--no-color` | Disable colored output | false |
| | `--progress-bar` | Progress bar type (simple/detailed/json) | detailed |
| | `--output-format` | Output format (auto/json/yaml) | auto |

### Check Options

| Flag | Long Form | Description | Default |
|------|-----------|-------------|---------|
| | `--check-connectivity` | Check network before download | false |
| | `--check-space` | Check disk space before download | true |

### Other Options

| Flag | Long Form | Description | Default |
|------|-----------|-------------|---------|
| `-h` | `--help` | Show help information | - |
| | `--version` | Show version information | - |
| | `--interactive` | Enable interactive prompts | auto |
| | `--language` | Language for messages (en/ja/es/fr) | en |

## Examples

### Basic Examples

```bash
# Simple download
gdl https://example.com/file.zip

# Save with specific name
gdl -o archive.zip https://example.com/file.zip

# Download to specific directory
gdl -o /downloads/file.zip https://example.com/file.zip

# Create directories if needed
gdl --create-dirs -o path/to/file.zip https://example.com/file.zip
```

### Concurrent Downloads

```bash
# Use 8 concurrent connections
gdl -c 8 https://example.com/large-file.iso

# Specify chunk size
gdl --concurrent 4 --chunk-size 2MB https://example.com/file.zip

# Limit download rate
gdl --max-rate 1MB/s https://example.com/large-file.zip
gdl --max-rate 500k --concurrent 2 https://example.com/file.zip

# Disable concurrent download
gdl --no-concurrent https://example.com/file.zip
```

### Resume Downloads

```bash
# Enable resume support
gdl --resume https://example.com/large-file.iso

# Continue partial download
gdl --continue-partial -o partial.zip https://example.com/file.zip
```

### Custom Headers

```bash
# Single header
gdl -H "Authorization: Bearer token123" https://api.example.com/file

# Multiple headers
gdl -H "Authorization: Bearer token123" \
     -H "X-Custom-Header: value" \
     https://api.example.com/file

# Custom User-Agent
gdl --user-agent "MyApp/1.0" https://example.com/file.zip
```

### Network Configuration

```bash
# Set timeout
gdl --timeout 5m https://example.com/large-file.iso

# Configure retries
gdl --retry 5 --retry-delay 2s https://unreliable.com/file.zip

# Use proxy
gdl --proxy http://proxy.example.com:8080 https://example.com/file.zip

# Skip SSL verification (not recommended)
gdl -k https://self-signed.example.com/file.zip
```

### Progress Display

```bash
# Quiet mode (no output)
gdl -q https://example.com/file.zip

# Verbose mode (detailed output)
gdl -v https://example.com/file.zip

# Simple progress bar
gdl --progress-bar simple https://example.com/file.zip

# JSON progress output
gdl --progress-bar json https://example.com/file.zip

# No colors
gdl --no-color https://example.com/file.zip
```

### Pre-download Checks

```bash
# Check network connectivity
gdl --check-connectivity https://example.com/file.zip

# Disable disk space check
gdl --check-space=false https://example.com/large-file.iso
```

### Force Overwrite

```bash
# Overwrite existing file
gdl -f -o existing.zip https://example.com/file.zip

# Interactive mode (will prompt)
gdl --interactive -o existing.zip https://example.com/file.zip
```

### Complex Example

```bash
gdl \
  --concurrent 8 \
  --chunk-size 4MB \
  --max-rate 2MB/s \
  --resume \
  --retry 5 \
  --retry-delay 3s \
  --timeout 10m \
  -H "Authorization: Bearer $TOKEN" \
  -H "X-Request-ID: $(uuidgen)" \
  --user-agent "MyDownloader/2.0" \
  --create-dirs \
  --check-connectivity \
  --verbose \
  -o downloads/$(date +%Y%m%d)/large-file.iso \
  https://example.com/releases/latest.iso
```

## Configuration

### Environment Variables

```bash
# Set default concurrent connections
export GDL_CONCURRENT=8

# Set default chunk size
export GDL_CHUNK_SIZE=4MB

# Set default timeout
export GDL_TIMEOUT=10m
```

### Config File (Future)

```yaml
# ~/.gdl/config.yaml
concurrent: 8
chunk-size: 4MB
timeout: 10m
retry: 5
retry-delay: 2s
user-agent: "MyApp/1.0"
headers:
  X-Custom-Header: value
```

## Exit Codes

| Code | Description |
|------|-------------|
| 0 | Success |
| 1 | General error |
| 2 | Network error |
| 3 | File system error |
| 4 | Invalid arguments |
| 5 | Timeout |
| 6 | User cancelled |
| 7 | Insufficient disk space |
| 8 | Permission denied |

## Shell Integration

### Bash Completion

```bash
# Add to ~/.bashrc
complete -W "--help --version --output --force --concurrent --chunk-size \
  --max-rate --resume --retry --timeout --quiet --verbose" gdl
```

### Aliases

```bash
# Add to ~/.bashrc or ~/.zshrc
alias dl='gdl'
alias dlr='gdl --resume'
alias dlf='gdl --force'
alias dlv='gdl --verbose'
```

### Function Wrapper

```bash
# Smart download function
download() {
    local url="$1"
    local filename="${2:-$(basename "$url")}"
    
    gdl \
        --resume \
        --concurrent 8 \
        --max-rate 5MB/s \
        --retry 3 \
        --create-dirs \
        -o "$filename" \
        "$url"
}
```

## Troubleshooting

### Common Issues

**Problem**: Download fails with "connection refused"
```bash
# Check connectivity first
gdl --check-connectivity --verbose https://example.com/file.zip
```

**Problem**: "Insufficient disk space" error
```bash
# Check available space
df -h .
# Download to different location
gdl -o /path/with/space/file.zip https://example.com/file.zip
```

**Problem**: SSL certificate error
```bash
# For self-signed certificates only (not recommended for production)
gdl --insecure https://self-signed.example.com/file.zip
```

**Problem**: Slow download speed
```bash
# Increase concurrent connections
gdl --concurrent 16 --chunk-size 8MB https://example.com/large-file.iso
```

### Debug Mode

```bash
# Maximum verbosity for debugging
gdl --verbose --progress-bar detailed https://example.com/file.zip 2>&1 | tee download.log
```