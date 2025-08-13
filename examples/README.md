# Godl Examples

This directory contains comprehensive examples demonstrating all features of the godl download library and CLI tool.

## Overview

The examples are organized into categories:

- **Core Examples** (`01-05_*`) - Step-by-step examples of core features
- **Library Examples** (`library/`) - Go programs showing how to use the godl library API
- **CLI Examples** (`cli/`) - Shell scripts demonstrating command-line usage
- **Integration Examples** (`integration/`) - Comprehensive feature demonstrations and tests

## Quick Start

Run any example directly:

```bash
# Core functionality examples
go run 01_basic_download/main.go
go run 02_concurrent_download/main.go
go run 03_progress_tracking/main.go
go run 04_resume_functionality/main.go
go run 05_error_handling/main.go

# Integration examples
go run integration/feature_demo.go
go run library_api/main.go
go run parity_verification/main.go
bash integration/cli_usage.sh
```

## Library Examples

### Basic Usage (`library/basic_usage.go`)

Demonstrates the core library API:

```bash
cd examples/library
go run basic_usage.go
```

**Features demonstrated:**
- Simple download with `godl.Download()`
- Download to memory with `godl.DownloadToMemory()`
- Progress callbacks with `godl.DownloadWithOptions()`
- Resume downloads with `godl.DownloadWithResume()`
- File information with `godl.GetFileInfo()`

### Advanced Usage (`library/advanced_usage.go`)

Shows advanced library features:

```bash
cd examples/library  
go run advanced_usage.go
```

**Features demonstrated:**
- Custom HTTP headers
- High-performance concurrent downloads
- Timeout and error handling
- Streaming to custom writers
- Comprehensive configuration with all options

## CLI Examples

### Basic CLI Usage (`cli/basic_cli_examples.sh`)

Demonstrates basic command-line features:

```bash
cd examples/cli
chmod +x basic_cli_examples.sh
./basic_cli_examples.sh
```

**Features demonstrated:**
- Simple downloads
- Custom output filenames (`-o`, `--output`)
- Force overwrite (`--force`)
- Directory creation (`--create-dirs`)
- Quiet mode (`--quiet`)
- Verbose mode (`--verbose`)
- Custom user agents (`--user-agent`)
- Timeouts (`--timeout`)
- Resume functionality (`--resume`)

### Advanced CLI Usage (`cli/advanced_cli_examples.sh`)

Shows advanced command-line features:

```bash
cd examples/cli
chmod +x advanced_cli_examples.sh
./advanced_cli_examples.sh
```

**Features demonstrated:**
- Custom headers (`-H`, `--header`)
- Concurrent downloads (`--concurrent`)
- Chunk size control (`--chunk-size`)
- Single-threaded mode (`--no-concurrent`)
- Retry configuration (`--retry`, `--retry-delay`)
- Redirect limits (`--max-redirects`)
- Progress bar types (`--progress-bar`)
- Language settings (`--language`)
- Color control (`--no-color`)
- SSL verification (`--insecure`)
- Pre-flight checks (`--check-connectivity`, `--check-space`)
- Output formatting (`--output-format`)
- Partial downloads (`--continue-partial`)

## Integration Examples

### Feature Demo (`integration/feature_demo.go`)

Comprehensive demonstration of all features:

```bash
cd examples/integration
go run feature_demo.go
```

This program:
1. **Sets up a test HTTP server** with various endpoints
2. **Demonstrates library API** with all major features
3. **Builds and tests CLI** with all command-line options
4. **Runs integration tests** to verify consistency between library and CLI
5. **Tests error handling** to ensure robust behavior

## CLI Flag Reference

### Basic Flags
- `-o, --output FILE` - Output filename
- `-f, --force` - Overwrite existing files
- `--create-dirs` - Create parent directories
- `-q, --quiet` - Quiet mode (no progress output)
- `-v, --verbose` - Verbose output
- `--version` - Show version information
- `-h, --help` - Show help message

### Download Control
- `--user-agent STRING` - Custom User-Agent header
- `--timeout DURATION` - Download timeout (e.g., 30s, 5m, 1h)
- `--resume` - Resume partial downloads
- `--no-resume` - Disable resume functionality

### Concurrent Downloads  
- `--concurrent N` - Number of concurrent connections (1-32)
- `-c N` - Shorthand for --concurrent
- `--chunk-size SIZE` - Chunk size (e.g., 1MB, 512KB, 2GB)
- `--no-concurrent` - Force single-threaded download

### HTTP Configuration
- `-H, --header 'Key: Value'` - Add custom headers (multiple allowed)
- `--max-redirects N` - Maximum redirects to follow
- `--insecure, -k` - Skip SSL certificate verification  
- `--proxy URL` - HTTP proxy URL

### Retry and Reliability
- `--retry N` - Number of retry attempts (default: 3)
- `--retry-delay DURATION` - Delay between retries (default: 1s)
- `--check-connectivity` - Check network before download
- `--check-space` - Check disk space before download (default: true)

### Output and Display
- `--progress-bar TYPE` - Progress bar type (simple|detailed|json)
- `--no-color` - Disable colored output
- `--language LANG` - Language for messages (en, ja, es, fr)
- `--interactive` - Enable interactive prompts
- `--output-format FORMAT` - Output format (auto|json|yaml)
- `--continue-partial` - Continue partial downloads

## Library API Reference

### Basic Functions

```go
// Simple download
func Download(ctx context.Context, url, dest string) error

// Download with custom options
func DownloadWithOptions(ctx context.Context, url, dest string, opts *Options) error

// Download to io.Writer
func DownloadToWriter(ctx context.Context, url string, w io.Writer) error

// Download to memory
func DownloadToMemory(ctx context.Context, url string) ([]byte, error)

// Resume download
func DownloadWithResume(ctx context.Context, url, dest string) error

// Get file information
func GetFileInfo(ctx context.Context, url string) (*FileInfo, error)
```

### Options Structure

```go
type Options struct {
    ProgressCallback  ProgressCallback       // Progress updates
    MaxConcurrency    int                   // Concurrent connections
    ChunkSize         int64                 // Chunk size in bytes
    EnableResume      bool                  // Enable resume support
    RetryAttempts     int                   // Number of retry attempts
    Timeout           time.Duration         // Request timeout
    UserAgent         string                // Custom User-Agent
    Headers           map[string]string     // Custom HTTP headers
    CreateDirs        bool                  // Create parent directories
    OverwriteExisting bool                  // Overwrite existing files
    Quiet             bool                  // Suppress progress output
    Verbose           bool                  // Enable verbose logging
}
```

## Running Examples

To run all examples:

```bash
# Library examples
cd examples/library
go run basic_usage.go
go run advanced_usage.go

# CLI examples  
cd examples/cli
chmod +x *.sh
./basic_cli_examples.sh
./advanced_cli_examples.sh

# Integration demo
cd examples/integration
go run feature_demo.go
```

## Example Output

The examples will create various test files demonstrating different download scenarios. All examples include proper error handling and cleanup.

## Notes

- Examples use `httpbin.org` for testing HTTP features
- The integration demo creates a local test server
- All examples include proper cleanup of generated files
- CLI examples require building the `godl` binary first (done automatically)

## Advanced Usage Patterns

### Progress Tracking
```go
opts := &godl.Options{
    ProgressCallback: func(p godl.Progress) {
        fmt.Printf("Progress: %.1f%% (%d/%d bytes) @ %.1f KB/s\n",
            p.Percentage, p.BytesDownloaded, p.TotalSize, float64(p.Speed)/1024)
    },
}
```

### Custom Headers
```go
opts := &godl.Options{
    Headers: map[string]string{
        "Authorization": "Bearer your-token",
        "User-Agent":    "MyApp/1.0",
        "Accept":        "application/json",
    },
}
```

### High-Performance Downloads
```go
opts := &godl.Options{
    MaxConcurrency: 8,
    ChunkSize:      1024 * 1024, // 1MB chunks
    EnableResume:   true,
    RetryAttempts:  5,
}
```

These examples provide complete coverage of all godl features and serve as both documentation and test cases for the library.