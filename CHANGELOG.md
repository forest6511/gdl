# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- **Resume Functionality**: Complete implementation of download resume with HTTP Range requests (#32)
  - Automatic resume state persistence in `~/.gdl/resume/`
  - ETag and Last-Modified validation for safe resume
  - File integrity verification using SHA256 checksums
  - Graceful fallback when server doesn't support Range requests
  - Automatic cleanup of resume files on successful completion
  - Progress saving on interruption (Ctrl+C, network failure)
  - Resume offset tracking with partial content (206) support

## [1.3.1] - 2025-09-30

### Changed
- **Build System**: Removed Docker support to simplify maintenance
  - Removed Docker-related files and workflows
  - Cleaned up GoReleaser configuration
  - Simplified CI/CD pipelines

### Fixed
- **Test Coverage**: Improved CLI plugin system test coverage to 85%
  - Added comprehensive tests for plugin registry operations
  - Enhanced error handling coverage
  - Fixed Windows-specific test compatibility issues

### Removed
- **Docker Support**: Removed all Docker-related functionality
  - Deleted Dockerfile and Docker workflows
  - Removed Docker configurations from GoReleaser
  - Updated documentation to reflect Docker removal 


## [1.3.0] - 2025-09-23

### Added
- **Cross-Platform Optimizations**: Comprehensive platform-specific performance enhancements
  - Zero-copy download support using sendfile on Linux/macOS for large files
  - Platform-specific buffer sizes and concurrency settings
  - ARM architecture optimizations for both server and embedded devices
  - Adaptive performance tuning based on system capabilities (CPU, memory)
- **Advanced Buffer Pool Management**: Smart memory management with size-based allocation
  - Tiered buffer pool system (8KB, 64KB, 512KB)
  - Automatic buffer sizing based on download requirements
  - Reduced memory allocations and improved GC performance
- **Network Optimizations**: Enhanced connection handling and transport settings
  - HTTP/2 support with optimized settings
  - Connection pooling with platform-specific limits
  - Optimized TCP socket options for each platform
- **Lightweight Download Mode**: Efficient single-threaded download for small files
  - Automatic mode selection based on file size
  - Reduced overhead for files under 100KB
  - Lower memory footprint for embedded systems
- **ARM Platform Support**: Full support for ARM and ARM64 architectures
  - Dedicated CI/CD pipeline for ARM testing
  - QEMU-based testing for cross-compilation validation
  - Optimized settings for ARM server and mobile/embedded devices
- **Performance Regression Testing**: Automated performance monitoring
  - Benchmark comparisons between base and PR branches
  - Real-world performance tests against curl
  - Memory profiling and leak detection

### Changed
- **Go Version**: Updated to Go 1.24 for latest performance improvements
- **Test Coverage**: Improved internal/core package coverage from 81.2% to 87.8%
- **CI/CD Workflows**: Restructured for better performance and reliability
  - Added ARM platform testing workflow
  - Optimized performance regression workflow (3x faster)
  - Enhanced cross-platform compatibility testing

### Fixed
- **Windows Compatibility**: Resolved path handling issues in tests
- **Linux Zero-copy**: Fixed syscall.EAGAIN handling in sendfile operations
- **Error Handling**: Added proper error checking for all Close() operations
- **Branch Protection**: Aligned CI job names with GitHub protection rules

### Security
- **Enhanced Error Checking**: All file operations now properly handle errors
- **Resource Cleanup**: Improved cleanup of file descriptors and network connections
- **Test Isolation**: Better test isolation to prevent resource leaks


## [0.9.2] - 2025-08-19

### Added
- **Bandwidth Throttling System**: Comprehensive rate limiting with token bucket algorithm
  - New `--max-rate` CLI flag supporting human-readable formats (e.g., `1MB/s`, `500k`)
  - `MaxRate` option in library API (bytes/sec, 0=unlimited)  
  - Seamless integration with concurrent downloads and all worker threads
  - ParseRate and FormatRate utilities for flexible rate specifications
- **Enhanced Test Coverage**: Significantly improved test coverage across core packages
  - pkg/ratelimit: 93.0% coverage with comprehensive edge case testing
  - internal/concurrent: 91.9% coverage for manager and worker components
  - internal/core: 85.8% coverage for downloader functionality
  - Overall project coverage improved to 65.5%
- **Storage Backend Testing**: Comprehensive filesystem backend testing
  - Security tests for directory traversal protection
  - Cross-platform path handling validation
  - Home directory expansion (`~/`) testing

### Changed
- **API Enhancement**: Extended Options struct with MaxRate field for bandwidth control
- **Documentation Updates**: 
  - Updated README.md with bandwidth control features
  - Enhanced CLI and API reference documentation
  - Added rate limiting examples and troubleshooting section

### Fixed
- **Windows Compatibility**: Improved cross-platform file path operations
- **Lint Issues**: Resolved errcheck and staticcheck violations
- **Race Conditions**: Enhanced signal handler test stability
- **Test Infrastructure**: Fixed cross-platform test failures and CI issues

### Security
- **Path Validation**: Enhanced directory traversal protection in filesystem backend
- **Rate Limiting**: Prevents resource abuse with configurable bandwidth limits

### Performance
- **Token Bucket Algorithm**: Efficient rate limiting with minimal overhead
- **Concurrent Downloads**: Rate limiting seamlessly integrated across all worker threads

## [0.9.0] - 2025-01-13

### Added
- Fast, concurrent file download capability with configurable worker pools
- Resume support for interrupted downloads with automatic chunk management
- Progress tracking with visual indicators and ETA calculation
- Configurable retry mechanism with exponential backoff
- Cross-platform support (Linux, macOS, Windows)
- Comprehensive test coverage including unit and integration tests
- CI/CD pipeline setup with GitHub Actions
- Basic error handling and recovery mechanisms
- Command-line interface with intuitive flags and options
- Timeout configuration for download operations

### Changed
- Initial release with core functionality

### Fixed
- macOS timeout command compatibility in test scripts
- CI job naming conflicts
- CI environment test issues

[Unreleased]: https://github.com/forest6511/gdl/compare/v1.3.1...HEAD
[0.9.2]: https://github.com/forest6511/gdl/compare/v0.9.0...v0.9.2
[0.9.0]: https://github.com/forest6511/gdl/releases/tag/v0.9.0
[1.3.0]: https://github.com/forest6511/gdl/compare/v0.9.2...v1.3.0
[1.3.1]: https://github.com/forest6511/gdl/compare/v1.3.0...v1.3.1
