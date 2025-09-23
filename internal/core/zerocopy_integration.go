package core

import (
	"context"
	"time"

	"github.com/forest6511/gdl/pkg/types"
)

// performZeroCopyDownload performs a download using zero-copy optimization
func (d *Downloader) performZeroCopyDownload(
	ctx context.Context,
	url, destination string,
	options *types.DownloadOptions,
) (*types.DownloadStats, error) {
	startTime := time.Now()
	stats := &types.DownloadStats{
		URL:       url,
		Filename:  destination,
		StartTime: startTime,
	}

	var downloaded int64
	var err error

	if options.ProgressCallback != nil {
		downloaded, err = d.zeroCopy.DownloadWithProgress(
			ctx, url, destination,
			func(down, total int64) {
				elapsed := time.Since(startTime).Seconds()
				speed := int64(0)
				if elapsed > 0 {
					speed = int64(float64(down) / elapsed)
				}
				options.ProgressCallback(down, total, speed)
			},
		)
	} else {
		downloaded, err = d.zeroCopy.Download(ctx, url, destination)
	}

	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)
	stats.BytesDownloaded = downloaded
	stats.TotalSize = downloaded

	if err != nil {
		stats.Error = err
		stats.Success = false
		return stats, err
	}

	stats.Success = true

	// Calculate average speed
	if stats.Duration > 0 {
		stats.AverageSpeed = int64(float64(downloaded) / stats.Duration.Seconds())
	}

	d.logInfo("zerocopy_download_complete", "Zero-copy download completed successfully", map[string]interface{}{
		"url":         url,
		"destination": destination,
		"size":        downloaded,
		"duration":    stats.Duration.String(),
	})

	return stats, nil
}
