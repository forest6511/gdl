# API Reference

Complete API documentation for the godl library.

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
import "github.com/forest6511/godl"
```

### Simple Download

```go
err := godl.Download(context.Background(), url, filename)
if err != nil {
    log.Fatal(err)
}
```

### Download with Options

```go
options := &godl.Options{
    MaxConcurrency:    8,
    ChunkSize:         1024 * 1024, // 1MB chunks
    MaxRate:           2 * 1024 * 1024, // 2MB/s rate limit
    EnableResume:      true,
    OverwriteExisting: false,
    ProgressCallback: func(p godl.Progress) {
        fmt.Printf("Progress: %.1f%%\n", p.Percentage)
    },
}

err := godl.DownloadWithOptions(context.Background(), url, filename, options)
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
stats, err := godl.Download(ctx, url, filename, options)
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

stats, err := godl.Download(ctx, url, filename, options)
```

### Custom HTTP Client

```go
import "github.com/forest6511/godl/internal/core"

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
        
        stats, err := godl.Download(context.Background(), 
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
stats, err := godl.DownloadToWriter(ctx, url, &buf, options)
if err != nil {
    log.Fatal(err)
}

data := buf.Bytes()
fmt.Printf("Downloaded %d bytes to memory\n", len(data))
```

### Resume with Manual Control

```go
// Check if partial file exists
info, err := os.Stat(filename)
resumeOffset := int64(0)
if err == nil {
    resumeOffset = info.Size()
}

options := &types.DownloadOptions{
    Resume: resumeOffset > 0,
}

stats, err := godl.Download(ctx, url, filename, options)
if stats.Resumed {
    fmt.Printf("Resumed from byte %d\n", resumeOffset)
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
downloader := godl.NewDownloader()

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