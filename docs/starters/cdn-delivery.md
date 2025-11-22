# CDN Large File Delivery

Download large files from CDNs efficiently using gdl's optimized concurrent download capabilities.

## Use Case

You need to download large files (100MB+) from CDN providers like:
- Software releases and updates
- ISO images
- Video files
- Large datasets

## Quick Start

```bash
# Basic large file download - gdl auto-optimizes concurrency
gdl https://releases.ubuntu.com/22.04/ubuntu-22.04-desktop-amd64.iso

# With resume support (recommended for large files)
gdl --resume https://releases.ubuntu.com/22.04/ubuntu-22.04-desktop-amd64.iso

# With bandwidth limit (if sharing connection)
gdl --max-rate 10MB/s --resume https://example-cdn.com/large-file.zip
```

## Optimal Settings by File Size

gdl automatically selects optimal settings, but you can override:

| File Size | Recommended Concurrency | Command |
|-----------|------------------------|---------|
| < 1MB | 1 (automatic) | `gdl URL` |
| 1-10MB | 2 | `gdl --concurrent 2 URL` |
| 10-100MB | 4 | `gdl --concurrent 4 URL` |
| 100MB-1GB | 8 | `gdl --concurrent 8 URL` |
| > 1GB | 16 | `gdl --concurrent 16 URL` |

## Real-World Examples

### Download Ubuntu ISO

```bash
# Download with progress and auto-resume
gdl --resume -o ubuntu-22.04.iso \
    https://releases.ubuntu.com/22.04/ubuntu-22.04-desktop-amd64.iso

# Verify download integrity
sha256sum ubuntu-22.04.iso
```

### Download from AWS CloudFront

```bash
# CloudFront distribution with custom headers
gdl -H "X-Custom-Header: value" \
    https://d1234567890.cloudfront.net/files/large-dataset.tar.gz
```

### Download from GitHub Releases

```bash
# Download specific release asset
gdl --resume \
    https://github.com/kubernetes/kubernetes/releases/download/v1.28.0/kubernetes.tar.gz
```

## Performance Tips

### 1. Enable Resume for Large Files

Always use `--resume` for files over 100MB:

```bash
gdl --resume https://cdn.example.com/large-file.zip
```

If interrupted, simply run the same command to continue.

### 2. Use Appropriate Chunk Size

For very large files (>1GB), larger chunks can improve throughput:

```bash
gdl --chunk-size 4MB --concurrent 16 https://cdn.example.com/huge-file.zip
```

### 3. Monitor Download Progress

Use verbose mode to see detailed progress:

```bash
gdl --progress detailed https://cdn.example.com/file.zip
```

### 4. Handle CDN Rate Limits

If the CDN enforces rate limits:

```bash
gdl --max-rate 50MB/s --retry 5 https://cdn.example.com/file.zip
```

## Troubleshooting

### Download Keeps Failing

```bash
# Increase retry attempts and timeout
gdl --retry 10 --timeout 300s https://cdn.example.com/file.zip
```

### Resume Not Working

Some CDNs don't support range requests. Check if resume is supported:

```bash
# Check server capabilities
curl -I https://cdn.example.com/file.zip | grep Accept-Ranges
```

### Slow Download Speed

Try adjusting concurrency:

```bash
# Increase concurrent connections
gdl --concurrent 16 https://cdn.example.com/file.zip

# Or reduce if server limits connections
gdl --concurrent 2 https://cdn.example.com/file.zip
```

## Library Usage

For programmatic downloads in Go:

```go
package main

import (
    "context"
    "fmt"
    "github.com/forest6511/gdl"
)

func main() {
    ctx := context.Background()

    opts := &gdl.Options{
        MaxConcurrency: 8,
        Resume:         true,
        ProgressCallback: func(p gdl.Progress) {
            fmt.Printf("\r%.1f%% - %.2f MB/s", p.Percentage, float64(p.Speed)/1024/1024)
        },
    }

    stats, err := gdl.DownloadWithOptions(ctx,
        "https://cdn.example.com/large-file.zip",
        "large-file.zip",
        opts)

    if err != nil {
        panic(err)
    }

    fmt.Printf("\nDownloaded %d bytes in %v\n", stats.BytesDownloaded, stats.Duration)
}
```

## Related

- [Batch Downloads](batch-downloads.md) - Download multiple files
- [CI/CD Artifact Fetching](ci-artifacts.md) - Automate downloads in CI
