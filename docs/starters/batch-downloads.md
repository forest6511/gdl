# Batch Downloads

Download multiple files efficiently using gdl with parallel execution and proper error handling.

## Use Case

- Download multiple files from a list
- Mirror directory contents
- Batch asset downloads
- Multi-file synchronization

## Quick Start

```bash
# Download multiple files in parallel using shell
gdl -o file1.zip https://example.com/file1.zip &
gdl -o file2.zip https://example.com/file2.zip &
gdl -o file3.zip https://example.com/file3.zip &
wait
```

## Batch Download Methods

### Method 1: Shell Script with URL List

```bash
#!/bin/bash
# batch-download.sh

URL_FILE="urls.txt"
OUTPUT_DIR="./downloads"
MAX_PARALLEL=4

mkdir -p "$OUTPUT_DIR"

# Read URLs and download in parallel
cat "$URL_FILE" | xargs -P "$MAX_PARALLEL" -I {} bash -c '
    URL="{}"
    FILENAME=$(basename "$URL")
    echo "Downloading: $FILENAME"
    gdl --quiet --retry 3 -o "'"$OUTPUT_DIR"'/$FILENAME" "$URL"
'

echo "All downloads completed"
```

**urls.txt:**
```
https://example.com/file1.zip
https://example.com/file2.zip
https://example.com/file3.zip
```

### Method 2: Go Library Batch Download

```go
package main

import (
    "context"
    "fmt"
    "sync"

    "github.com/forest6511/gdl"
)

type DownloadTask struct {
    URL         string
    Destination string
}

type BatchResult struct {
    Task    DownloadTask
    Success bool
    Error   error
    Stats   *gdl.DownloadStats
}

func BatchDownload(ctx context.Context, tasks []DownloadTask, maxConcurrent int) []BatchResult {
    results := make([]BatchResult, len(tasks))
    sem := make(chan struct{}, maxConcurrent)
    var wg sync.WaitGroup

    for i, task := range tasks {
        wg.Add(1)
        go func(idx int, t DownloadTask) {
            defer wg.Done()

            sem <- struct{}{}
            defer func() { <-sem }()

            opts := &gdl.Options{
                Resume:     true,
                MaxRetries: 3,
            }

            stats, err := gdl.DownloadWithOptions(ctx, t.URL, t.Destination, opts)

            results[idx] = BatchResult{
                Task:    t,
                Success: err == nil,
                Error:   err,
                Stats:   stats,
            }
        }(i, task)
    }

    wg.Wait()
    return results
}

func main() {
    ctx := context.Background()

    tasks := []DownloadTask{
        {"https://example.com/file1.zip", "downloads/file1.zip"},
        {"https://example.com/file2.zip", "downloads/file2.zip"},
        {"https://example.com/file3.zip", "downloads/file3.zip"},
    }

    results := BatchDownload(ctx, tasks, 4)

    for _, r := range results {
        if r.Success {
            fmt.Printf("OK: %s (%d bytes)\n", r.Task.Destination, r.Stats.BytesDownloaded)
        } else {
            fmt.Printf("FAILED: %s - %v\n", r.Task.URL, r.Error)
        }
    }
}
```

### Method 3: JSON Manifest Download

```bash
#!/bin/bash
# Download files from JSON manifest

MANIFEST="manifest.json"
OUTPUT_DIR="./assets"

mkdir -p "$OUTPUT_DIR"

# Parse JSON and download each file
jq -r '.files[] | "\(.url) \(.name)"' "$MANIFEST" | while read URL NAME; do
    echo "Downloading: $NAME"
    gdl --quiet --retry 3 -o "$OUTPUT_DIR/$NAME" "$URL"
done
```

**manifest.json:**
```json
{
  "version": "1.0",
  "files": [
    {"name": "asset1.png", "url": "https://cdn.example.com/assets/asset1.png"},
    {"name": "asset2.png", "url": "https://cdn.example.com/assets/asset2.png"},
    {"name": "config.json", "url": "https://cdn.example.com/config/config.json"}
  ]
}
```

## Advanced Patterns

### Download with Progress Tracking

```go
package main

import (
    "context"
    "fmt"
    "sync"
    "sync/atomic"

    "github.com/forest6511/gdl"
)

type BatchDownloader struct {
    totalFiles     int
    completedFiles int64
    totalBytes     int64
    downloadedBytes int64
    mu             sync.Mutex
}

func (b *BatchDownloader) Download(ctx context.Context, tasks []DownloadTask) {
    b.totalFiles = len(tasks)
    var wg sync.WaitGroup

    for _, task := range tasks {
        wg.Add(1)
        go func(t DownloadTask) {
            defer wg.Done()

            opts := &gdl.Options{
                Resume: true,
                ProgressCallback: func(p gdl.Progress) {
                    // Update global progress
                    atomic.AddInt64(&b.downloadedBytes, int64(p.Speed))
                },
            }

            _, err := gdl.DownloadWithOptions(ctx, t.URL, t.Destination, opts)
            if err == nil {
                atomic.AddInt64(&b.completedFiles, 1)
            }

            // Print progress
            completed := atomic.LoadInt64(&b.completedFiles)
            fmt.Printf("\rProgress: %d/%d files completed", completed, b.totalFiles)
        }(task)
    }

    wg.Wait()
    fmt.Println("\nAll downloads completed")
}
```

### Retry Failed Downloads

```bash
#!/bin/bash
# Download with retry tracking

URL_FILE="urls.txt"
FAILED_FILE="failed.txt"
MAX_RETRIES=3

> "$FAILED_FILE"  # Clear failed file

download_url() {
    local url=$1
    local dest=$2
    local retries=0

    while [ $retries -lt $MAX_RETRIES ]; do
        if gdl --quiet --retry 2 -o "$dest" "$url"; then
            return 0
        fi
        retries=$((retries + 1))
        sleep 5
    done

    echo "$url" >> "$FAILED_FILE"
    return 1
}

# Download all URLs
while read url; do
    filename=$(basename "$url")
    download_url "$url" "downloads/$filename"
done < "$URL_FILE"

# Report results
if [ -s "$FAILED_FILE" ]; then
    echo "Some downloads failed. See: $FAILED_FILE"
    exit 1
fi

echo "All downloads completed successfully"
```

### Bandwidth-Limited Batch Download

```bash
#!/bin/bash
# Limit total bandwidth across all downloads

# Total bandwidth in KB/s (numeric value for calculation)
TOTAL_BANDWIDTH_KB=10240  # 10MB/s = 10240KB/s
MAX_PARALLEL=4

# Calculate per-download bandwidth in KB/s
PER_DOWNLOAD_KB=$((TOTAL_BANDWIDTH_KB / MAX_PARALLEL))

cat urls.txt | xargs -P $MAX_PARALLEL -I {} bash -c '
    gdl --max-rate '"${PER_DOWNLOAD_KB}k"' --quiet -o downloads/$(basename {}) {}
'
```

## Error Handling

### Comprehensive Error Handling Script

```bash
#!/bin/bash
set -e

download_with_validation() {
    local url=$1
    local dest=$2
    local expected_size=$3

    # Download file
    if ! gdl --retry 3 --quiet -o "$dest" "$url"; then
        echo "ERROR: Failed to download $url"
        return 1
    fi

    # Validate file exists
    if [ ! -f "$dest" ]; then
        echo "ERROR: File not created: $dest"
        return 1
    fi

    # Validate file size (if expected size provided)
    if [ -n "$expected_size" ]; then
        actual_size=$(stat -f%z "$dest" 2>/dev/null || stat --printf="%s" "$dest")
        if [ "$actual_size" -ne "$expected_size" ]; then
            echo "ERROR: Size mismatch for $dest (expected: $expected_size, got: $actual_size)"
            rm -f "$dest"
            return 1
        fi
    fi

    echo "OK: $dest"
    return 0
}

# Example usage
download_with_validation "https://example.com/file.zip" "file.zip" 1048576
```

## Performance Tips

### 1. Limit Concurrent Downloads

Too many concurrent downloads can overwhelm the server:

```bash
# Recommended: 4-8 concurrent downloads
cat urls.txt | xargs -P 4 -I {} gdl --quiet -o downloads/$(basename {}) {}
```

### 2. Use Resume for Large Batches

```bash
# Resume-enabled batch download
for url in $(cat urls.txt); do
    gdl --resume --quiet -o "downloads/$(basename $url)" "$url"
done
```

### 3. Monitor Overall Progress

```bash
#!/bin/bash
TOTAL=$(wc -l < urls.txt)
COUNT=0

while read url; do
    COUNT=$((COUNT + 1))
    echo "[$COUNT/$TOTAL] Downloading: $(basename $url)"
    gdl --quiet -o "downloads/$(basename $url)" "$url"
done < urls.txt
```

## Related

- [CDN Large File Delivery](cdn-delivery.md) - Optimize large downloads
- [CI/CD Artifact Fetching](ci-artifacts.md) - Automated downloads
- [API Integration](api-integration.md) - Library usage
