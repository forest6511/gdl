# API Integration

Use gdl as a library in your Go application for programmatic file downloads with full control over concurrency, progress, and error handling.

## Use Case

- Embed download functionality in Go applications
- Build custom download managers
- Create automated data pipelines
- Integrate with existing systems

## Quick Start

```go
package main

import (
    "context"
    "fmt"
    "github.com/forest6511/gdl"
)

func main() {
    // Simplest download
    stats, err := gdl.Download(context.Background(),
        "https://example.com/file.zip", "file.zip")
    if err != nil {
        panic(err)
    }
    fmt.Printf("Downloaded %d bytes\n", stats.BytesDownloaded)
}
```

## Installation

```bash
go get github.com/forest6511/gdl
```

## API Overview

### Simple Downloads

```go
// Download to file
stats, err := gdl.Download(ctx, url, destination)

// Download with options
stats, err := gdl.DownloadWithOptions(ctx, url, destination, options)

// Download to memory
data, stats, err := gdl.DownloadToMemory(ctx, url)

// Download to io.Writer
stats, err := gdl.DownloadToWriter(ctx, url, writer)

// Resume partial download
stats, err := gdl.DownloadWithResume(ctx, url, destination)

// Get file info without downloading
info, err := gdl.GetFileInfo(ctx, url)
```

### Download Options

```go
options := &gdl.Options{
    // Concurrency
    MaxConcurrency: 8,           // Parallel connections (0 = auto)

    // Bandwidth
    MaxRate: 1024 * 1024,        // Max bytes/second (0 = unlimited)

    // Resume
    Resume: true,                // Enable resume for partial downloads

    // Retry
    MaxRetries: 3,               // Retry attempts on failure

    // Progress
    ProgressCallback: func(p gdl.Progress) {
        fmt.Printf("%.1f%% - %d bytes/s\n", p.Percentage, p.Speed)
    },

    // Headers
    Headers: map[string]string{
        "Authorization": "Bearer token",
        "User-Agent":    "MyApp/1.0",
    },
}
```

## Common Patterns

### Progress Tracking

```go
package main

import (
    "context"
    "fmt"
    "github.com/forest6511/gdl"
)

func main() {
    ctx := context.Background()

    options := &gdl.Options{
        ProgressCallback: func(p gdl.Progress) {
            bar := progressBar(p.Percentage, 40)
            fmt.Printf("\r%s %.1f%% | %.2f MB/s | ETA: %s",
                bar,
                p.Percentage,
                float64(p.Speed)/1024/1024,
                formatDuration(p.ETA))
        },
    }

    stats, err := gdl.DownloadWithOptions(ctx,
        "https://example.com/large-file.zip",
        "large-file.zip",
        options)

    if err != nil {
        fmt.Printf("\nError: %v\n", err)
        return
    }

    fmt.Printf("\nCompleted: %d bytes in %v\n",
        stats.BytesDownloaded, stats.Duration)
}

func progressBar(percent float64, width int) string {
    filled := int(percent / 100 * float64(width))
    bar := ""
    for i := 0; i < width; i++ {
        if i < filled {
            bar += "█"
        } else {
            bar += "░"
        }
    }
    return "[" + bar + "]"
}

func formatDuration(d time.Duration) string {
    if d < time.Minute {
        return fmt.Sprintf("%ds", int(d.Seconds()))
    }
    return fmt.Sprintf("%dm%ds", int(d.Minutes()), int(d.Seconds())%60)
}
```

### Context Cancellation

```go
package main

import (
    "context"
    "fmt"
    "os"
    "os/signal"
    "syscall"

    "github.com/forest6511/gdl"
)

func main() {
    // Create cancellable context
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle interrupt signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

    go func() {
        <-sigChan
        fmt.Println("\nCancelling download...")
        cancel()
    }()

    options := &gdl.Options{
        Resume: true, // Enable resume so we can continue later
    }

    stats, err := gdl.DownloadWithOptions(ctx,
        "https://example.com/large-file.zip",
        "large-file.zip",
        options)

    if err != nil {
        if ctx.Err() == context.Canceled {
            fmt.Println("Download cancelled. Run again to resume.")
        } else {
            fmt.Printf("Error: %v\n", err)
        }
        return
    }

    fmt.Printf("Downloaded %d bytes\n", stats.BytesDownloaded)
}
```

### Download with Timeout

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/forest6511/gdl"
)

func main() {
    // 5 minute timeout
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
    defer cancel()

    stats, err := gdl.Download(ctx,
        "https://example.com/file.zip",
        "file.zip")

    if err != nil {
        if ctx.Err() == context.DeadlineExceeded {
            fmt.Println("Download timed out")
        } else {
            fmt.Printf("Error: %v\n", err)
        }
        return
    }

    fmt.Printf("Downloaded %d bytes\n", stats.BytesDownloaded)
}
```

### Download to Memory

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/forest6511/gdl"
)

type Config struct {
    Version string `json:"version"`
    Name    string `json:"name"`
}

func main() {
    ctx := context.Background()

    // Download JSON config to memory
    data, stats, err := gdl.DownloadToMemory(ctx,
        "https://api.example.com/config.json")

    if err != nil {
        panic(err)
    }

    fmt.Printf("Downloaded %d bytes in %v\n", len(data), stats.Duration)

    // Parse JSON
    var config Config
    if err := json.Unmarshal(data, &config); err != nil {
        panic(err)
    }

    fmt.Printf("Config: %+v\n", config)
}
```

### Download to Custom Writer

```go
package main

import (
    "bytes"
    "context"
    "crypto/sha256"
    "fmt"
    "io"

    "github.com/forest6511/gdl"
)

func main() {
    ctx := context.Background()

    // Download and calculate hash simultaneously
    var buffer bytes.Buffer
    hash := sha256.New()
    writer := io.MultiWriter(&buffer, hash)

    stats, err := gdl.DownloadToWriter(ctx,
        "https://example.com/data.bin",
        writer)

    if err != nil {
        panic(err)
    }

    fmt.Printf("Downloaded %d bytes\n", stats.BytesDownloaded)
    fmt.Printf("SHA256: %x\n", hash.Sum(nil))
    fmt.Printf("Buffer size: %d bytes\n", buffer.Len())
}
```

### Using the Downloader Object

```go
package main

import (
    "context"
    "fmt"

    "github.com/forest6511/gdl"
    "github.com/forest6511/gdl/pkg/events"
)

func main() {
    ctx := context.Background()

    // Create downloader instance
    downloader := gdl.NewDownloader()

    // Register event listeners
    downloader.On(events.EventDownloadStarted, func(e events.Event) {
        fmt.Printf("Started: %s\n", e.Data["url"])
    })

    downloader.On(events.EventDownloadProgress, func(e events.Event) {
        fmt.Printf("Progress: %.1f%%\n", e.Data["percentage"])
    })

    downloader.On(events.EventDownloadCompleted, func(e events.Event) {
        fmt.Printf("Completed: %d bytes\n", e.Data["bytes"])
    })

    downloader.On(events.EventDownloadFailed, func(e events.Event) {
        fmt.Printf("Failed: %v\n", e.Data["error"])
    })

    // Download
    options := &gdl.Options{MaxConcurrency: 4}
    stats, err := downloader.Download(ctx,
        "https://example.com/file.zip",
        "file.zip",
        options)

    if err != nil {
        fmt.Printf("Error: %v\n", err)
        return
    }

    fmt.Printf("Final stats: %+v\n", stats)
}
```

### Middleware Usage

```go
package main

import (
    "context"
    "fmt"
    "time"

    "github.com/forest6511/gdl"
    "github.com/forest6511/gdl/pkg/middleware"
)

func main() {
    ctx := context.Background()

    downloader := gdl.NewDownloader()

    // Add logging middleware
    downloader.UseMiddleware(middleware.Logging(func(msg string) {
        fmt.Printf("[%s] %s\n", time.Now().Format("15:04:05"), msg)
    }))

    // Add custom middleware
    downloader.UseMiddleware(func(next middleware.Handler) middleware.Handler {
        return func(ctx context.Context, req *middleware.Request) (*middleware.Response, error) {
            fmt.Printf("Starting download: %s\n", req.URL)
            start := time.Now()

            resp, err := next(ctx, req)

            fmt.Printf("Finished in %v\n", time.Since(start))
            return resp, err
        }
    })

    stats, _ := downloader.Download(ctx,
        "https://example.com/file.zip",
        "file.zip",
        nil)

    fmt.Printf("Downloaded %d bytes\n", stats.BytesDownloaded)
}
```

## Error Handling

```go
package main

import (
    "context"
    "errors"
    "fmt"

    "github.com/forest6511/gdl"
    gdlerrors "github.com/forest6511/gdl/pkg/errors"
)

func main() {
    ctx := context.Background()

    _, err := gdl.Download(ctx,
        "https://example.com/file.zip",
        "file.zip")

    if err != nil {
        // Check for specific error types
        var httpErr *gdlerrors.HTTPError
        if errors.As(err, &httpErr) {
            fmt.Printf("HTTP error %d: %s\n", httpErr.StatusCode, httpErr.Message)
            return
        }

        var netErr *gdlerrors.NetworkError
        if errors.As(err, &netErr) {
            fmt.Printf("Network error: %s\n", netErr.Message)
            if netErr.Retryable {
                fmt.Println("This error is retryable")
            }
            return
        }

        // Generic error
        fmt.Printf("Download failed: %v\n", err)
    }
}
```

## Related

- [API Reference](../API_REFERENCE.md) - Complete API documentation
- [Plugin Development](../PLUGIN_DEVELOPMENT.md) - Extend gdl functionality
- [Batch Downloads](batch-downloads.md) - Download multiple files
