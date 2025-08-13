# godl - Go Downloader

A fast, concurrent, and feature-rich file downloader library and CLI tool written in Go.

[![CI/CD Pipeline](https://github.com/forest6511/godl/actions/workflows/main.yml/badge.svg)](https://github.com/forest6511/godl/actions/workflows/main.yml)
[![Go Report Card](https://goreportcard.com/badge/github.com/forest6511/godl)](https://goreportcard.com/report/github.com/forest6511/godl)
[![Coverage](https://codecov.io/gh/forest6511/godl/branch/main/graph/badge.svg)](https://codecov.io/gh/forest6511/godl)
[![Go Reference](https://pkg.go.dev/badge/github.com/forest6511/godl.svg)](https://pkg.go.dev/github.com/forest6511/godl)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

## ✨ Features

- **🚀 High Performance**: Concurrent downloads with configurable connections
- **📊 Progress Tracking**: Real-time progress with multiple display formats  
- **🔄 Resume Support**: Automatic resume of interrupted downloads
- **🌐 Protocol Support**: HTTP/HTTPS with custom headers and proxy support
- **🛡️ Error Handling**: Comprehensive error handling with smart retry logic
- **⚡ Cross-Platform**: Works on Linux, macOS, and Windows
- **🔧 Dual Interface**: Both library API and command-line tool
- **📱 User-Friendly**: Interactive prompts and helpful error messages
- **🔌 Plugin System**: Extensible plugin architecture for custom functionality
- **🎯 Event-Driven**: Hook into download lifecycle events
- **🔐 Security**: Built-in security constraints and validation

## 📦 Installation

### As a CLI tool
```bash
go install github.com/forest6511/godl/cmd/godl@latest
```

### As a library
```bash
go get github.com/forest6511/godl
```

## 🚀 Quick Start

### CLI Usage
```bash
# Simple download
godl https://example.com/file.zip

# Concurrent download with custom settings
godl --concurrent 8 --chunk-size 2MB -o myfile.zip https://example.com/file.zip

# With custom headers and resume
godl -H "Authorization: Bearer token" --resume https://example.com/file.zip

# Using plugins
godl plugin install oauth2-auth
godl plugin list
godl --plugin oauth2-auth https://secure-api.example.com/file.zip
```

### Library Usage
```go
package main

import (
    "bytes"
    "context"
    "fmt"
    "github.com/forest6511/godl"
)

func main() {
    // Simple download using Download function
    stats, err := godl.Download(context.Background(), 
        "https://example.com/file.zip", "file.zip")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Downloaded %d bytes in %v\n", stats.BytesDownloaded, stats.Duration)
    
    // Download with progress callback using DownloadWithOptions
    options := &godl.Options{
        MaxConcurrency: 4,
        ProgressCallback: func(p godl.Progress) {
            fmt.Printf("Progress: %.1f%% Speed: %.2f MB/s\n", 
                p.Percentage, float64(p.Speed)/1024/1024)
        },
    }
    
    stats, err = godl.DownloadWithOptions(context.Background(),
        "https://example.com/file.zip", "file.zip", options)
    if err == nil {
        fmt.Printf("Download completed successfully! Average speed: %.2f MB/s\n", 
            float64(stats.AverageSpeed)/1024/1024)
    }
    
    // Download to memory using DownloadToMemory
    data, stats, err := godl.DownloadToMemory(context.Background(),
        "https://example.com/small-file.txt")
    if err == nil {
        fmt.Printf("Downloaded %d bytes to memory in %v\n", len(data), stats.Duration)
    }
    
    // Download to any io.Writer using DownloadToWriter
    var buffer bytes.Buffer
    stats, err = godl.DownloadToWriter(context.Background(),
        "https://example.com/data.json", &buffer)
    if err == nil {
        fmt.Printf("Downloaded to buffer: %d bytes\n", stats.BytesDownloaded)
    }
        
    // Resume a partial download using DownloadWithResume
    stats, err = godl.DownloadWithResume(context.Background(),
        "https://example.com/large-file.zip", "large-file.zip")
    if err == nil && stats.Resumed {
        fmt.Printf("Successfully resumed download: %d bytes\n", stats.BytesDownloaded)
    }
        
    // Get file information without downloading using GetFileInfo
    fileInfo, err := godl.GetFileInfo(context.Background(),
        "https://example.com/file.zip")
    if err == nil {
        fmt.Printf("File size: %d bytes\n", fileInfo.Size)
    }
    
    // Using the extensible Downloader with plugins
    downloader := godl.NewDownloader()
    
    // Register custom protocol handler
    err = downloader.RegisterProtocol(customProtocolHandler)
    
    // Use middleware
    downloader.UseMiddleware(rateLimitingMiddleware)
    
    // Register event listeners
    downloader.On(events.EventDownloadStarted, func(event events.Event) {
        fmt.Printf("Download started: %s\n", event.Data["url"])
    })
    
    // Download with plugins and middleware
    stats, err = downloader.Download(context.Background(),
        "https://example.com/file.zip", "file.zip", options)
    if err == nil {
        fmt.Printf("Plugin-enhanced download completed: %d bytes\n", stats.BytesDownloaded)
    }
}
```

## 📚 Complete API Documentation

- **[📋 API Reference](docs/API_REFERENCE.md)** - Complete library API documentation
- **[📁 Directory Structure](docs/DIRECTORY_STRUCTURE.md)** - Complete project organization
- **[🔧 Maintenance Guide](docs/MAINTENANCE.md)** - Development and maintenance procedures
- **[🚨 Error Handling](docs/errors/README.md)** - Error types and handling strategies
- **[🔌 Plugin Development](docs/PLUGIN_DEVELOPMENT.md)** - Plugin development guide
- **[🚀 Extending Guide](docs/EXTENDING.md)** - Extension points and customization
- **[📦 Go Package Docs](https://pkg.go.dev/github.com/forest6511/godl)** - Generated Go documentation

## 📋 Complete CLI Reference

- **[⚙️ CLI Reference](docs/CLI_REFERENCE.md)** - Comprehensive CLI usage guide

## 📖 Examples

Complete working examples are available in the [`examples/`](examples/) directory:

- **[Basic Download](examples/01_basic_download/)** - Simple download operations
- **[Concurrent Downloads](examples/02_concurrent_download/)** - Parallel download optimization
- **[Progress Tracking](examples/03_progress_tracking/)** - Real-time progress monitoring
- **[Resume Support](examples/04_resume_functionality/)** - Interrupt and resume downloads
- **[Error Handling](examples/05_error_handling/)** - Robust error recovery
- **[CLI Examples](examples/cli/)** - Command-line usage patterns
- **[Integration Tests](examples/integration/)** - Feature verification
- **[Plugin Examples](examples/plugins/)** - Custom plugin development
- **[Extension Examples](examples/extensions/)** - System extension patterns

### Running Examples
```bash
# Core functionality examples
cd examples/01_basic_download && go run main.go
cd examples/02_concurrent_download && go run main.go
cd examples/03_progress_tracking && go run main.go
cd examples/04_resume_functionality && go run main.go
cd examples/05_error_handling && go run main.go

# Interface examples
cd examples/cli
chmod +x *.sh
./basic_cli_examples.sh
./advanced_cli_examples.sh

# Integration demo
cd examples/integration
go run feature_demo.go

# Plugin examples
cd examples/plugins/auth/oauth2
go build -buildmode=plugin -o oauth2.so
cd examples/plugins/storage/s3
go build -buildmode=plugin -o s3.so
```

## 🔄 Feature Parity Matrix

| Feature | CLI | Library | Description |
|---------|:---:|:-------:|-------------|
| Basic download | ✅ | ✅ | Simple URL to file download |
| Custom destination | ✅ | ✅ | Specify output filename/path |
| Overwrite existing | ✅ | ✅ | Force overwrite of existing files |
| Create directories | ✅ | ✅ | Auto-create parent directories |
| Concurrent downloads | ✅ | ✅ | Multiple simultaneous connections |
| Custom chunk size | ✅ | ✅ | Configurable download chunks |
| Single-threaded mode | ✅ | ✅ | Disable concurrent downloads |
| Resume downloads | ✅ | ✅ | Continue interrupted downloads |
| Retry on failure | ✅ | ✅ | Automatic retry with backoff |
| Custom retry settings | ✅ | ✅ | Configure retry attempts/delays |
| Custom headers | ✅ | ✅ | Add custom HTTP headers |
| Custom User-Agent | ✅ | ✅ | Set custom User-Agent string |
| Proxy support | ✅ | ✅ | HTTP proxy configuration |
| SSL verification control | ✅ | ✅ | Skip SSL certificate verification |
| Redirect handling | ✅ | ✅ | Follow HTTP redirects |
| Timeout configuration | ✅ | ✅ | Set request/download timeouts |
| Progress display | ✅ | ❌ | Visual progress bars |
| Progress callbacks | ❌ | ✅ | Programmatic progress updates |
| Multiple progress formats | ✅ | ❌ | Simple/detailed/JSON progress |
| Quiet mode | ✅ | ✅ | Suppress output |
| Verbose mode | ✅ | ✅ | Detailed logging |
| Download to memory | ❌ | ✅ | Download directly to memory |
| Download to writer | ❌ | ✅ | Download to any io.Writer |
| File info retrieval | ❌ | ✅ | Get file metadata without download |
| Error handling | ✅ | ✅ | Robust error handling and recovery |
| Comprehensive errors | ✅ | ✅ | Detailed error information |
| Error suggestions | ✅ | ❌ | User-friendly error suggestions |
| Multilingual messages | ✅ | ❌ | Localized error messages |
| Interactive prompts | ✅ | ❌ | User confirmation prompts |
| Disk space checking | ✅ | ❌ | Pre-download space verification |
| Network diagnostics | ✅ | ❌ | Network connectivity testing |
| Signal handling | ✅ | ❌ | Graceful shutdown on signals |
| Plugin system | ✅ | ✅ | Extensible plugin architecture |
| Custom protocols | ✅ | ✅ | Plugin-based protocol handlers |
| Middleware support | ❌ | ✅ | Request/response processing |
| Event system | ❌ | ✅ | Download lifecycle events |
| Custom storage | ❌ | ✅ | Pluggable storage backends |

### Legend
- ✅ **Fully supported** - Feature is available and fully functional
- ❌ **Not applicable** - Feature doesn't make sense in this context

### Key Concepts: Resume vs. Retry

The terms "Resume" and "Retry" sound similar but handle different situations. Understanding the difference is key to using `godl` effectively.

|          | Retry                                  | Resume                                     |
| :------- | :------------------------------------- | :----------------------------------------- |
| **Purpose**  | Automatic recovery from temporary errors | Manual continuation after an intentional stop |
| **Trigger**  | Network errors, server errors        | Existence of an incomplete file            |
| **Control**  | Number of attempts (`RetryAttempts`)   | Enabled/Disabled (`EnableResume`)          |
| **Result**   | `stats.Retries` (count)                | `stats.Resumed` (true/false)               |

#### Scenario: Combining Resume and Retry

1.  **Day 1:** You start downloading a 10GB file. It gets to 5GB, and you stop the program (e.g., with Ctrl+C). (-> **Interruption**)
2.  **Day 2:** You run the same command again with resume enabled. `godl` detects the incomplete 5GB file and starts downloading from that point. (-> This is a **Resumed** download)
3.  During the download of the remaining 5GB, your network connection briefly drops, causing a timeout error.
4.  `godl` automatically waits a moment and re-attempts the failed request. (-> This is a **Retry**)
5.  The download then completes successfully.

In this case, the final `DownloadStats` would be `Resumed: true` and `Retries: 1`.

## 🏗️ Architecture

For the complete project structure, see [Directory Structure](docs/DIRECTORY_STRUCTURE.md).

### Key Components

- **Core Engine** (`internal/core`): Main download orchestration
- **Concurrency Manager** (`internal/concurrent`): Parallel download coordination  
- **Progress System** (`pkg/progress`): Real-time progress tracking
- **Error Framework** (`pkg/errors`): Comprehensive error handling
- **Network Layer** (`internal/network`): HTTP client and diagnostics
- **Storage Layer** (`internal/storage`): File system operations
- **Plugin System** (`pkg/plugin`): Extensible plugin architecture
- **Event System** (`pkg/events`): Download lifecycle events
- **Middleware Layer** (`pkg/middleware`): Request/response processing
- **Protocol Registry** (`pkg/protocols`): Custom protocol handlers
- **CLI Interface** (`cmd/godl`): Command-line tool implementation

### Plugin Architecture

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

**Plugin Types:**
- **Authentication Plugins**: OAuth2, API keys, custom auth schemes
- **Protocol Plugins**: FTP, S3, custom protocols
- **Storage Plugins**: Cloud storage, databases, custom backends
- **Transform Plugins**: Compression, encryption, format conversion
- **Hook Plugins**: Pre/post processing, logging, analytics

## 🧪 Testing

The project includes comprehensive testing:

```bash
# Run all tests
go test ./...

# Run tests with race detection  
go test -race ./...

# Run with coverage
go test -coverprofile=coverage.out ./...

# Run benchmarks
go test -bench=. ./...
```

### Test Coverage
- **Unit tests**: All packages have >90% coverage
- **Integration tests**: Real HTTP download scenarios  
- **CLI tests**: Command-line interface functionality
- **Benchmark tests**: Performance regression detection
- **Race detection**: Concurrent safety verification


## 🤝 Contributing

We welcome contributions! Please see [CONTRIBUTING.md](CONTRIBUTING.md) for guidelines.

### Development Setup
```bash
# Clone the repository
git clone https://github.com/forest6511/godl.git
cd godl

# Install dependencies
go mod download

# Install golangci-lint (if not already installed)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

# Verify everything works (CI equivalent check)
make ci-check

# Run tests
go test ./...

# Build CLI
go build -o godl ./cmd/godl/
```

### Pre-commit Checks

Before committing and pushing changes, always run these checks locally to avoid CI failures:

```bash
# Run lint checks (essential before commit/push)
golangci-lint run

# Run tests with race detection
go test -race ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
```

### 🔄 CI Compatibility

To prevent "works locally but fails in CI" issues, use these CI-equivalent commands:

```bash
# RECOMMENDED: Run complete pre-push validation
make pre-push        # Formats code AND runs all CI checks

# Alternative: Run CI checks without formatting
make ci-check        # All CI checks locally

# Fix formatting issues automatically
make fix-and-commit  # Auto-fix formatting and create commit if needed

# Individual commands
make ci-format       # Format code to CI standards
make ci-vet          # go vet (excluding examples)
make ci-test-core    # core library tests with race detection
```

**⚠️ Important**: If the pre-commit hook reformats your code, it will stop the commit. Add the changes and commit again:
```bash
git add .
git commit
```

A pre-commit hook is automatically installed that runs CI-equivalent checks on every commit, preventing most CI failures.

**Important**: Always run `make ci-check` locally before pushing to ensure all checks pass. This prevents CI pipeline failures and maintains code quality standards.

For detailed development and maintenance procedures, see [docs/MAINTENANCE.md](docs/MAINTENANCE.md).

## 📄 License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## 🙏 Acknowledgments

- Go community for excellent libraries and tools
- Contributors who helped improve this project
- Users who provided feedback and bug reports

---

**Made with ❤️ in Go**