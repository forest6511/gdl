# IoT Delta Updates

Bandwidth-efficient firmware and software updates for IoT devices using gdl's resume and rate limiting capabilities.

## Use Case

- Firmware updates for embedded devices
- Software updates over limited bandwidth connections
- OTA (Over-The-Air) update systems
- Edge device management

## Key Challenges

IoT environments often face:
- Limited bandwidth (cellular, satellite, low-bandwidth networks)
- Unreliable connections (dropouts, intermittent connectivity)
- Resource constraints (limited storage, memory)
- Large device fleets requiring staggered updates

## Quick Start

```bash
# Bandwidth-limited download with resume
gdl --max-rate 100k --resume -o firmware.bin https://updates.example.com/firmware-v2.0.bin

# Download with retry for unreliable connections
gdl --retry 10 --max-rate 50k --resume -o firmware.bin https://updates.example.com/firmware.bin
```

## Implementation Patterns

### Basic OTA Update Script

```bash
#!/bin/bash
# ota-update.sh - IoT firmware update script

FIRMWARE_URL="https://updates.example.com/device/${DEVICE_ID}/firmware.bin"
FIRMWARE_PATH="/tmp/firmware.bin"
CHECKSUM_URL="${FIRMWARE_URL}.sha256"

# Bandwidth limit based on connection type
case "$CONNECTION_TYPE" in
    cellular) RATE="50k" ;;
    wifi)     RATE="1MB/s" ;;
    ethernet) RATE="10MB/s" ;;
    *)        RATE="100k" ;;
esac

echo "Starting firmware update (max rate: ${RATE})..."

# Download checksum first (small file)
gdl --quiet -o "${FIRMWARE_PATH}.sha256" "$CHECKSUM_URL"

# Download firmware with rate limiting and resume
if gdl --max-rate "$RATE" --resume --retry 5 \
       -o "$FIRMWARE_PATH" "$FIRMWARE_URL"; then

    # Verify checksum
    EXPECTED=$(cat "${FIRMWARE_PATH}.sha256" | cut -d' ' -f1)
    ACTUAL=$(sha256sum "$FIRMWARE_PATH" | cut -d' ' -f1)

    if [ "$EXPECTED" = "$ACTUAL" ]; then
        echo "Firmware verified, applying update..."
        # Apply firmware update here
    else
        echo "Checksum mismatch, retrying..."
        rm -f "$FIRMWARE_PATH"
        exit 1
    fi
else
    echo "Download failed, will retry later"
    exit 1
fi
```

### Go Library for OTA System

```go
package ota

import (
    "context"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "io"
    "os"
    "time"

    "github.com/forest6511/gdl"
)

type UpdateConfig struct {
    FirmwareURL     string
    DestinationPath string
    ExpectedHash    string
    MaxBandwidth    int64  // bytes per second
    MaxRetries      int
}

type UpdateResult struct {
    Success       bool
    BytesDownloaded int64
    Duration      time.Duration
    Resumed       bool
}

func DownloadFirmware(ctx context.Context, cfg UpdateConfig) (*UpdateResult, error) {
    opts := &gdl.Options{
        MaxRate:        cfg.MaxBandwidth,
        Resume:         true,
        MaxRetries:     cfg.MaxRetries,
        ProgressCallback: func(p gdl.Progress) {
            // Log progress for monitoring
            fmt.Printf("Update progress: %.1f%% (%.2f KB/s)\n",
                p.Percentage, float64(p.Speed)/1024)
        },
    }

    stats, err := gdl.DownloadWithOptions(ctx, cfg.FirmwareURL, cfg.DestinationPath, opts)
    if err != nil {
        return nil, fmt.Errorf("download failed: %w", err)
    }

    // Verify checksum
    if cfg.ExpectedHash != "" {
        hash, err := calculateFileHash(cfg.DestinationPath)
        if err != nil {
            return nil, fmt.Errorf("hash calculation failed: %w", err)
        }
        if hash != cfg.ExpectedHash {
            os.Remove(cfg.DestinationPath)
            return nil, fmt.Errorf("checksum mismatch: expected %s, got %s",
                cfg.ExpectedHash, hash)
        }
    }

    return &UpdateResult{
        Success:         true,
        BytesDownloaded: stats.BytesDownloaded,
        Duration:        stats.Duration,
        Resumed:         stats.Resumed,
    }, nil
}

func calculateFileHash(path string) (string, error) {
    f, err := os.Open(path)
    if err != nil {
        return "", err
    }
    defer f.Close()

    h := sha256.New()
    if _, err := io.Copy(h, f); err != nil {
        return "", err
    }

    return hex.EncodeToString(h.Sum(nil)), nil
}
```

### Staggered Fleet Updates

```go
package fleet

import (
    "context"
    "math/rand"
    "time"

    "github.com/forest6511/gdl"
)

type Device struct {
    ID       string
    Priority int
}

type FleetUpdater struct {
    devices      []Device
    baseURL      string
    maxConcurrent int
}

func (f *FleetUpdater) RolloutUpdate(ctx context.Context, version string) error {
    // Stagger updates to avoid overwhelming servers and network
    sem := make(chan struct{}, f.maxConcurrent)

    for _, device := range f.devices {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case sem <- struct{}{}:
        }

        go func(d Device) {
            defer func() { <-sem }()

            // Random delay to spread load
            delay := time.Duration(rand.Intn(60)) * time.Second
            time.Sleep(delay)

            url := fmt.Sprintf("%s/%s/firmware-%s.bin", f.baseURL, d.ID, version)
            dest := fmt.Sprintf("/tmp/firmware-%s.bin", d.ID)

            opts := &gdl.Options{
                MaxRate:    50 * 1024, // 50KB/s per device
                Resume:     true,
                MaxRetries: 5,
            }

            gdl.DownloadWithOptions(ctx, url, dest, opts)
        }(device)
    }

    return nil
}
```

## Bandwidth Optimization

### Adaptive Rate Limiting

```bash
#!/bin/bash
# Adapt bandwidth based on network conditions

# Check current network load
PING_TIME=$(ping -c 3 8.8.8.8 | tail -1 | awk -F'/' '{print $5}')

if (( $(echo "$PING_TIME < 50" | bc -l) )); then
    RATE="1MB/s"  # Good connection
elif (( $(echo "$PING_TIME < 200" | bc -l) )); then
    RATE="100k"   # Moderate connection
else
    RATE="50k"    # Poor connection
fi

gdl --max-rate "$RATE" --resume URL
```

### Time-Based Scheduling

```bash
#!/bin/bash
# Download during off-peak hours

HOUR=$(date +%H)

if [ "$HOUR" -ge 2 ] && [ "$HOUR" -lt 6 ]; then
    # Off-peak: full bandwidth
    gdl --resume URL
else
    # Peak hours: limited bandwidth
    gdl --max-rate 100k --resume URL
fi
```

## Monitoring and Reporting

### Progress Reporting to Server

```go
func downloadWithReporting(ctx context.Context, url, dest, reportURL string) error {
    opts := &gdl.Options{
        Resume: true,
        ProgressCallback: func(p gdl.Progress) {
            // Report progress to monitoring server every 10%
            if int(p.Percentage)%10 == 0 {
                reportProgress(reportURL, p.Percentage, p.Speed)
            }
        },
    }

    _, err := gdl.DownloadWithOptions(ctx, url, dest, opts)
    return err
}

func reportProgress(url string, percentage float64, speed int64) {
    // Send progress to monitoring server
    data := fmt.Sprintf(`{"progress": %.1f, "speed": %d}`, percentage, speed)
    http.Post(url, "application/json", strings.NewReader(data))
}
```

## Error Recovery

### Automatic Retry with Backoff

gdl handles retry automatically, but for IoT scenarios you may want additional logic:

```bash
#!/bin/bash
MAX_ATTEMPTS=10
ATTEMPT=0
BACKOFF=60  # Initial backoff in seconds

while [ $ATTEMPT -lt $MAX_ATTEMPTS ]; do
    if gdl --retry 3 --max-rate 100k --resume -o firmware.bin URL; then
        echo "Download successful"
        exit 0
    fi

    ATTEMPT=$((ATTEMPT + 1))
    echo "Attempt $ATTEMPT failed, waiting ${BACKOFF}s..."
    sleep $BACKOFF
    BACKOFF=$((BACKOFF * 2))  # Exponential backoff
done

echo "All attempts failed"
exit 1
```

## Related

- [CDN Large File Delivery](cdn-delivery.md) - Optimize large downloads
- [CI/CD Artifact Fetching](ci-artifacts.md) - Automated downloads
