# API Reference

Complete API documentation for the gdl library.

## Table of Contents

- [Quick Start](#quick-start)
- [Main Functions](#main-functions)
- [Types](#types)
- [Options](#options)
- [Progress Callbacks](#progress-callbacks)
- [Error Handling](#error-handling)
- [Plugin System](#plugin-system)
- [Extension Points](#extension-points)
- [Advanced Usage](#advanced-usage)

## Quick Start

```go
import "github.com/forest6511/gdl"
```

### Simple Download

```go
err := gdl.Download(context.Background(), url, filename)
if err != nil {
    log.Fatal(err)
}
```

### Download with Options

```go
options := &gdl.Options{
    MaxConcurrency:    8,
    ChunkSize:         1024 * 1024, // 1MB chunks
    MaxRate:           2 * 1024 * 1024, // 2MB/s rate limit
    EnableResume:      true,
    OverwriteExisting: false,
    ProgressCallback: func(p gdl.Progress) {
        fmt.Printf("Progress: %.1f%%\n", p.Percentage)
    },
}

err := gdl.DownloadWithOptions(context.Background(), url, filename, options)
```

## Main Functions

### Download

Downloads a file from a URL to a local file.

```go
func Download(ctx context.Context, url, filename string, options *types.DownloadOptions) (*types.DownloadStats, error)
```

**Parameters:**
- `ctx`: Context for cancellation and timeout control
- `url`: Source URL to download from
- `filename`: Destination filename
- `options`: Optional download configuration

**Returns:**
- `*types.DownloadStats`: Download statistics
- `error`: Error if download failed

### DownloadToWriter

Downloads content directly to an io.Writer.

```go
func DownloadToWriter(ctx context.Context, url string, writer io.Writer, options *types.DownloadOptions) (*types.DownloadStats, error)
```

**Parameters:**
- `ctx`: Context for cancellation
- `url`: Source URL
- `writer`: Destination writer
- `options`: Optional configuration

### GetFileInfo

Retrieves file metadata without downloading.

```go
func GetFileInfo(ctx context.Context, url string) (*types.FileInfo, error)
```

**Returns:**
- `*types.FileInfo`: File metadata including size, MIME type, and headers

## Types

### DownloadOptions

Configuration for download operations.

```go
type DownloadOptions struct {
    // Concurrent connections
    Concurrent    int
    NoConcurrent  bool
    ChunkSize     int64
    
    // Bandwidth control
    MaxRate       int64  // Maximum download rate in bytes per second (0 = unlimited)
    
    // Resume and overwrite
    Resume            bool
    Overwrite         bool
    OverwriteExisting bool
    
    // Headers and authentication
    Headers    map[string]string
    UserAgent  string
    
    // Retry configuration
    Retry      int
    RetryDelay time.Duration
    
    // Network settings
    Timeout      time.Duration
    MaxRedirects int
    Insecure     bool
    Proxy        string
    
    // Progress tracking
    Progress         ProgressInterface
    ProgressCallback func(downloaded, total int64, speed float64)
    
    // Output control
    Quiet    bool
    Verbose  bool
    NoColor  bool
    
    // Pre-download checks
    CheckConnectivity bool
    CheckSpace        bool
    
    // Other options
    CreateDirs       bool
    ContinuePartial  bool
    Interactive      bool
}
```

### DownloadStats

Statistics about a completed download.

```go
type DownloadStats struct {
    URL             string
    Filename        string
    TotalSize       int64
    BytesDownloaded int64
    StartTime       time.Time
    EndTime         time.Time
    Duration        time.Duration
    AverageSpeed    int64
    Success         bool
    Resumed         bool
    Error           error
}
```

### FileInfo

File metadata from server.

```go
type FileInfo struct {
    Size           int64
    LastModified   time.Time
    ContentType    string
    SupportsRanges bool
    Headers        map[string][]string
    ETag           string
    Filename       string
}
```

### DownloadError

Custom error type with detailed information.

```go
type DownloadError struct {
    Code       ErrorCode
    Message    string
    URL        string
    Underlying error
    RetryAfter time.Duration
    Suggestion string
}
```

## Platform Optimization

gdl automatically detects and applies platform-specific optimizations for maximum performance.

### PlatformInfo

Platform detection information.

```go
type PlatformInfo struct {
    OS           string  // Operating system (linux, darwin, windows, etc.)
    Arch         string  // Architecture (amd64, arm64, arm, etc.)
    NumCPU       int     // Number of CPUs
    IsARM        bool    // Whether running on ARM architecture
    IsServerGrade bool   // Whether server-grade hardware (≥8 CPUs)
    Optimizations PlatformOptimizationSet
}
```

### PlatformOptimizationSet

Platform-specific optimization settings.

```go
type PlatformOptimizationSet struct {
    BufferSize       int  // Optimal buffer size for platform
    Concurrency      int  // Optimal concurrent connections
    MaxConnections   int  // Maximum connection pool size
    UseSendfile      bool // Whether sendfile is supported
    UseZeroCopy      bool // Whether zero-copy I/O is available
    EnableHTTP2      bool // Whether HTTP/2 is enabled
    ConnectionReuse  bool // Whether to reuse connections
}
```

### Platform Detection Functions

```go
// Get current platform information
info := gdl.GetPlatformInfo()

// Get platform description string
platformStr := gdl.GetPlatformString()
// Returns: "linux/amd64 (16 CPUs) [zero-copy, sendfile, server-grade]"

// Check if zero-copy should be used for file size
shouldUseZeroCopy := gdl.ShouldUseZeroCopyPlatform(fileSize)

// Get optimal chunk size for file size
chunkSize := gdl.GetOptimalChunkSizePlatform(fileSize)
```

### Platform-Specific Behaviors

The library automatically applies optimizations based on detected platform:

#### Linux
- Buffer Size: 512KB
- TCP optimizations: TCP_NODELAY, TCP_QUICKACK, TCP_CORK
- Sendfile and zero-copy I/O enabled
- Automatic ulimit adjustment

#### macOS (Darwin)
- Intel: 256KB buffers
- Apple Silicon: 128KB buffers (optimized for unified memory)
- SO_REUSEPORT enabled
- Sendfile support

#### Windows
- Buffer Size: 128KB
- Windows auto-tuning integration
- Conservative connection settings

#### ARM Architecture
- ARM32: 32KB buffers for embedded devices
- ARM64: 128KB buffers with server detection
- Optimized for power efficiency on mobile

## Progress Callbacks

### Simple Callback

```go
options := &types.DownloadOptions{
    ProgressCallback: func(downloaded, total int64, speed float64) {
        percent := float64(downloaded) / float64(total) * 100
        fmt.Printf("Progress: %.1f%% Speed: %.2f MB/s\n", 
            percent, speed/1024/1024)
    },
}
```

### Progress Interface

Implement the ProgressInterface for advanced progress tracking:

```go
type ProgressInterface interface {
    Start(filename string, totalSize int64)
    Update(filename string, downloaded int64, speed float64)
    Finish(filename string, stats *DownloadStats)
    Error(filename string, err error)
}
```

## Error Handling

### Error Codes

```go
const (
    CodeNetworkError      ErrorCode = "NETWORK_ERROR"
    CodeTimeout           ErrorCode = "TIMEOUT"
    CodeInvalidURL        ErrorCode = "INVALID_URL"
    CodeFileNotFound      ErrorCode = "FILE_NOT_FOUND"
    CodePermissionDenied  ErrorCode = "PERMISSION_DENIED"
    CodeInsufficientSpace ErrorCode = "INSUFFICIENT_SPACE"
    CodeHTTPError         ErrorCode = "HTTP_ERROR"
)
```

### Error Checking

```go
stats, err := gdl.Download(ctx, url, filename, options)
if err != nil {
    if dlErr, ok := err.(*types.DownloadError); ok {
        switch dlErr.Code {
        case types.CodeNetworkError:
            // Handle network error
        case types.CodeTimeout:
            // Handle timeout
        case types.CodeInsufficientSpace:
            // Handle disk space issue
        default:
            // Handle other errors
        }
    }
}
```

## Advanced Usage

### Context with Timeout

```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()

stats, err := gdl.Download(ctx, url, filename, options)
```

### Custom HTTP Client

```go
import "github.com/forest6511/gdl/internal/core"

client := &http.Client{
    Timeout: 60 * time.Second,
    Transport: &http.Transport{
        MaxIdleConns:    100,
        IdleConnTimeout: 90 * time.Second,
    },
}

downloader := core.NewDownloaderWithClient(client)
stats, err := downloader.Download(ctx, url, filename, options)
```

### Concurrent Downloads of Multiple Files

```go
type downloadJob struct {
    URL      string
    Filename string
}

jobs := []downloadJob{
    {"https://example.com/file1.zip", "file1.zip"},
    {"https://example.com/file2.zip", "file2.zip"},
    {"https://example.com/file3.zip", "file3.zip"},
}

var wg sync.WaitGroup
results := make(chan *types.DownloadStats, len(jobs))

for _, job := range jobs {
    wg.Add(1)
    go func(j downloadJob) {
        defer wg.Done()
        
        stats, err := gdl.Download(context.Background(), 
            j.URL, j.Filename, nil)
        if err != nil {
            log.Printf("Failed to download %s: %v", j.URL, err)
            return
        }
        results <- stats
    }(job)
}

wg.Wait()
close(results)

for stats := range results {
    fmt.Printf("Downloaded %s: %d bytes\n", 
        stats.Filename, stats.BytesDownloaded)
}
```

### Download to Memory

```go
var buf bytes.Buffer
stats, err := gdl.DownloadToWriter(ctx, url, &buf, options)
if err != nil {
    log.Fatal(err)
}

data := buf.Bytes()
fmt.Printf("Downloaded %d bytes to memory\n", len(data))
```

### Resume Support

gdl provides automatic resume functionality with intelligent validation and state management.

#### Automatic Resume (Recommended)

```go
// Enable automatic resume with state persistence
options := &types.DownloadOptions{
    Resume: true,
}

stats, err := gdl.Download(ctx, url, filename, options)
if err != nil {
    log.Fatal(err)
}

// Check if download was resumed
if stats.Resumed {
    fmt.Printf("Download resumed from previous state\n")
    fmt.Printf("Downloaded: %d bytes, Total: %d bytes\n",
        stats.BytesDownloaded, stats.TotalSize)
}
```

**How Automatic Resume Works**:
1. Resume state saved to `~/.gdl/resume/<filename>.json` during download
2. On interruption (Ctrl+C, network failure, timeout), progress is persisted
3. On restart, gdl automatically:
   - Loads previous resume state
   - Validates ETag and Last-Modified headers
   - Verifies partial file integrity with SHA256 checksum
   - Sends HTTP Range request to resume from offset
   - Falls back to full download if server doesn't support Range requests
4. On completion, resume state file is automatically cleaned up

**Resume State Information**:
```go
// Resume state stored in ~/.gdl/resume/
type ResumeInfo struct {
    URL             string    // Original download URL
    FilePath        string    // Destination file path
    DownloadedBytes int64     // Bytes downloaded so far
    TotalBytes      int64     // Total file size
    ETag            string    // Server ETag for validation
    LastModified    time.Time // Last-Modified timestamp
    Checksum        string    // SHA256 checksum of partial file
    AcceptRanges    bool      // Server supports Range requests
}
```

#### Resume Validation

The library automatically validates resume safety before continuing:

```go
// Validation performed automatically:
// 1. URL must match original download
// 2. ETag must match (if available)
// 3. Last-Modified must match (if ETag not available)
// 4. Partial file integrity verified with SHA256
// 5. Server must support Range requests

// If validation fails, gdl starts fresh download
```

#### Manual Resume Control

For advanced use cases, you can manually control resume behavior:

```go
import "github.com/forest6511/gdl/internal/resume"

// Create resume manager
homeDir, _ := os.UserHomeDir()
resumeDir := filepath.Join(homeDir, ".gdl", "resume")
manager := resume.NewManager(resumeDir)

// Load existing resume state
resumeInfo, err := manager.Load(filename)
if err == nil && resumeInfo != nil {
    // Check if resume is safe
    if manager.CanResume(resumeInfo) {
        // Validate partial file integrity
        valid, err := manager.ValidatePartialFile(resumeInfo)
        if err == nil && valid {
            fmt.Printf("Can safely resume from byte %d\n",
                resumeInfo.DownloadedBytes)
        }
    }
}

// Perform download with resume
options := &types.DownloadOptions{
    Resume: true,
}
stats, err := gdl.Download(ctx, url, filename, options)

// Clean up resume state on success
if stats.Success {
    manager.Delete(filename)
}
```

#### Resume Examples

**Example 1: Large File Download with Auto-Resume**
```go
// Download large ISO file with automatic resume
options := &types.DownloadOptions{
    Resume:       true,
    Concurrent:   8,
    ChunkSize:    1024 * 1024, // 1MB chunks
    Retry:        3,
    RetryDelay:   2 * time.Second,
}

stats, err := gdl.Download(ctx,
    "https://example.com/large-file.iso",
    "large-file.iso",
    options)

if err != nil {
    // Safe to retry - will resume from last position
    log.Printf("Download failed: %v (can retry with same command)", err)
} else if stats.Resumed {
    log.Printf("Successfully resumed download from %d bytes",
        stats.BytesDownloaded)
}
```

**Example 2: Resume with Progress Tracking**
```go
options := &types.DownloadOptions{
    Resume: true,
    ProgressCallback: func(downloaded, total int64, speed float64) {
        percent := float64(downloaded) / float64(total) * 100
        fmt.Printf("\rProgress: %.1f%% (%.2f MB/s)",
            percent, speed/1024/1024)
    },
}

stats, err := gdl.Download(ctx, url, filename, options)
if stats.Resumed {
    fmt.Printf("\n✓ Download resumed successfully\n")
}
```

**Example 3: Handling Resume Failures**
```go
stats, err := gdl.Download(ctx, url, filename, &types.DownloadOptions{
    Resume: true,
})

if err != nil {
    if dlErr, ok := err.(*types.DownloadError); ok {
        switch dlErr.Code {
        case types.CodeResumeValidationFailed:
            log.Printf("Cannot resume: %v (starting fresh)", err)
        case types.CodeRangeNotSupported:
            log.Printf("Server doesn't support resume (starting fresh)")
        default:
            log.Printf("Download error: %v", err)
        }
    }
}
```

## Monitoring System

The monitoring package provides comprehensive metrics collection and aggregation for production deployments.

### MetricsCollector

```go
import "github.com/forest6511/gdl/pkg/monitoring"

// Create a new metrics collector
mc := monitoring.NewMetricsCollector()

// Enable/disable metrics collection
mc.Enable()
mc.Disable()

// Set metrics retention duration
mc.SetRetentionDuration(24 * time.Hour)

// Record download lifecycle
mc.RecordDownloadStart("download-123", "https://example.com/file.zip")
mc.RecordDownloadProgress("download-123", 5120, 10240, 1024) // bytes downloaded, total, speed
mc.RecordDownloadComplete("download-123", &types.DownloadStats{
    Success:      true,
    TotalSize:    10240,
    AverageSpeed: 1024,
    Retries:      1,
})

// Get individual download metrics
metrics, err := mc.GetDownloadMetrics("download-123")
if err == nil {
    fmt.Printf("Downloaded %d bytes in %v\n", metrics.BytesDownloaded, metrics.Duration)
}

// Get aggregated metrics
aggregated := mc.GetAggregatedMetrics()
fmt.Printf("Success rate: %.2f%%\n", aggregated.SuccessRate * 100)
fmt.Printf("Average speed: %.2f MB/s\n", aggregated.AverageSpeed / (1024*1024))

// Export metrics for external systems
exported := mc.ExportMetrics()
// exported contains structured data ready for JSON marshaling

// Get real-time performance snapshot
snapshot := mc.GetPerformanceSnapshot()
fmt.Printf("Active downloads: %d\n", snapshot.ActiveDownloads)
```

### Production Usage with Monitoring

```go
// Production downloader with monitoring
import (
    "github.com/forest6511/gdl"
    "github.com/forest6511/gdl/pkg/monitoring"
)

type ProductionDownloader struct {
    downloader *gdl.Downloader
    metrics    *monitoring.MetricsCollector
}

func NewProductionDownloader() *ProductionDownloader {
    return &ProductionDownloader{
        downloader: gdl.NewDownloader(),
        metrics:    monitoring.NewMetricsCollector(),
    }
}

func (pd *ProductionDownloader) DownloadWithMonitoring(ctx context.Context, id, url, dest string) error {
    // Record download start
    pd.metrics.RecordDownloadStart(id, url)

    // Configure options with progress tracking
    options := &gdl.Options{
        ProgressCallback: func(progress gdl.Progress) {
            pd.metrics.RecordDownloadProgress(id,
                progress.BytesDownloaded,
                progress.TotalSize,
                progress.Speed)
        },
    }

    // Perform download
    stats, err := pd.downloader.DownloadWithOptions(ctx, url, dest, options)

    // Record completion (success or failure)
    pd.metrics.RecordDownloadComplete(id, stats)

    return err
}
```

### Metrics Data Types

```go
type DownloadMetrics struct {
    ID              string
    URL             string
    StartTime       time.Time
    EndTime         time.Time
    Duration        time.Duration
    TotalBytes      int64
    BytesDownloaded int64
    AverageSpeed    int64
    MaxSpeed        int64
    MinSpeed        int64
    RetryCount      int
    ChunksUsed      int
    Success         bool
    ErrorType       string
    ErrorMessage    string
    Protocol        string
}

type AggregatedMetrics struct {
    TotalDownloads      int64
    SuccessfulDownloads int64
    FailedDownloads     int64
    TotalBytes          int64
    AverageSpeed        float64
    ThroughputMBps      float64
    SuccessRate         float64
    ProtocolBreakdown   map[string]int64
    ErrorBreakdown      map[string]int64
    LastUpdated         time.Time
}
```

## Plugin System

### Extensible Downloader

The `Downloader` type provides plugin support and extensibility:

```go
type Downloader struct {
    pluginManager     *plugin.PluginManager
    eventEmitter      *events.EventEmitter
    middleware        *middleware.MiddlewareChain
    protocolRegistry  *protocols.ProtocolRegistry
    storageManager    *storage.StorageManager
}

// Create a new extensible downloader
downloader := gdl.NewDownloader()

// Register plugins
err := downloader.UsePlugin(oauthPlugin)

// Add middleware
downloader.UseMiddleware(rateLimitingMiddleware)

// Register event listeners
downloader.On(events.EventDownloadStarted, func(event events.Event) {
    log.Printf("Started downloading: %s", event.Data["url"])
})

// Download with plugins and middleware
err = downloader.Download(ctx, url, dest, options)
```

### Plugin Types

#### Authentication Plugins

```go
type AuthPlugin interface {
    Plugin
    
    Authenticate(ctx context.Context, request *AuthRequest) (*AuthResponse, error)
    RefreshToken(ctx context.Context, token string) (*AuthResponse, error)
    ValidateToken(ctx context.Context, token string) error
}
```

#### Protocol Plugins

```go
type ProtocolPlugin interface {
    Plugin
    
    SupportsScheme(scheme string) bool
    CreateHandler(ctx context.Context, url *url.URL) (ProtocolHandler, error)
}
```

#### Storage Plugins

```go
type StoragePlugin interface {
    Plugin
    
    Store(ctx context.Context, key string, data io.Reader) error
    Retrieve(ctx context.Context, key string) (io.ReadCloser, error)
    Delete(ctx context.Context, key string) error
}
```

### Plugin Management

```go
// Load plugin from file
plug, err := plugin.LoadPlugin("/path/to/plugin.so")

// Register plugin
err = downloader.UsePlugin(plug)

// List active plugins
for _, p := range downloader.GetActivePlugins() {
    fmt.Printf("Plugin: %s v%s\n", p.Name(), p.Version())
}

// Unload plugin
err = downloader.UnloadPlugin("plugin-name")
```

## Extension Points

### Event System

Subscribe to download lifecycle events:

```go
// Event types
const (
    EventDownloadStarted   EventType = "download.started"
    EventDownloadProgress  EventType = "download.progress"
    EventDownloadCompleted EventType = "download.completed"
    EventDownloadFailed    EventType = "download.failed"
)

// Register event handlers
downloader.On(events.EventDownloadProgress, func(event events.Event) {
    progress := event.Data["progress"].(int64)
    total := event.Data["total"].(int64)
    fmt.Printf("Progress: %d/%d bytes\n", progress, total)
})
```

### Middleware Chain

Add request/response processing middleware:

```go
type CustomMiddleware struct{}

func (m *CustomMiddleware) ProcessRequest(ctx context.Context, req *http.Request) (*http.Request, error) {
    // Add custom headers or modify request
    req.Header.Set("X-Custom-Header", "value")
    return req, nil
}

func (m *CustomMiddleware) ProcessResponse(ctx context.Context, resp *http.Response) (*http.Response, error) {
    // Process response
    return resp, nil
}

downloader.UseMiddleware(&CustomMiddleware{})
```

### Protocol Registry

Register custom protocol handlers:

```go
type CustomProtocolHandler struct{}

func (h *CustomProtocolHandler) Scheme() string {
    return "custom"
}

func (h *CustomProtocolHandler) Download(ctx context.Context, url *url.URL, dest io.Writer, options *DownloadOptions) error {
    // Custom download logic
    return nil
}

err := downloader.RegisterProtocol(&CustomProtocolHandler{})
```

### Storage Backends

Implement custom storage backends:

```go
type S3StorageBackend struct{
    client *s3.Client
}

func (s *S3StorageBackend) Store(ctx context.Context, key string, data io.Reader, metadata *StorageMetadata) error {
    // Store to S3
    return nil
}

func (s *S3StorageBackend) Retrieve(ctx context.Context, key string) (io.ReadCloser, *StorageMetadata, error) {
    // Retrieve from S3
    return nil, nil, nil
}

err := downloader.SetStorageBackend("s3", &S3StorageBackend{})
```

## Best Practices

1. **Always use context**: Pass appropriate context for cancellation control
2. **Handle errors properly**: Check error types for appropriate recovery
3. **Configure timeouts**: Set reasonable timeouts for your use case
4. **Use progress callbacks**: Provide user feedback for long downloads
5. **Enable resume**: For large files, always enable resume support
6. **Check disk space**: Enable space checking for large downloads
7. **Validate URLs**: Ensure URLs are valid before attempting download
8. **Clean up resources**: Properly close files and cancel contexts
9. **Use plugins wisely**: Only load plugins you need to reduce overhead
10. **Monitor events**: Use event system for logging and monitoring
11. **Test plugins**: Thoroughly test custom plugins in isolation
12. **Handle plugin errors**: Plugin failures shouldn't crash the main application

## Related Documentation

- [Plugin Development Guide](./PLUGIN_DEVELOPMENT.md)
- [Extension Guide](./EXTENDING.md)
- [CLI Reference](./CLI_REFERENCE.md)