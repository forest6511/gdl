// Package progress provides progress tracking functionality for downloads.
package progress

import (
	"fmt"
	"io"
	"sync"
	"time"
)

// ProgressCallback is a function type for receiving progress updates.
const (
	unknownValue = "unknown"
)

type ProgressCallback func(bytesDownloaded, totalBytes int64, speed int64)

// Progress represents the current state of a download operation.
type Progress struct {
	// TotalBytes is the total size of the download in bytes.
	// Set to -1 if unknown.
	TotalBytes int64

	// DownloadedBytes is the number of bytes downloaded so far.
	DownloadedBytes int64

	// Speed is the current download speed in bytes per second.
	Speed int64

	// StartTime is when the download started.
	StartTime time.Time

	// LastUpdate is when the progress was last updated.
	LastUpdate time.Time

	// callback is the function called when progress is updated.
	callback ProgressCallback

	// rateLimiter controls how often progress updates are sent.
	rateLimiter *RateLimiter

	// mutex protects concurrent access to progress data.
	mutex sync.RWMutex
}

// RateLimiter controls the frequency of progress updates to avoid overwhelming
// the callback with too many updates.
type RateLimiter struct {
	// interval is the minimum time between updates.
	interval time.Duration

	// lastUpdate is the time of the last update.
	lastUpdate time.Time

	// mutex protects concurrent access.
	mutex sync.Mutex
}

// NewRateLimiter creates a new rate limiter with the specified minimum interval.
func NewRateLimiter(interval time.Duration) *RateLimiter {
	return &RateLimiter{
		interval: interval,
	}
}

// Allow returns true if enough time has passed since the last update.
func (rl *RateLimiter) Allow() bool {
	rl.mutex.Lock()
	defer rl.mutex.Unlock()

	now := time.Now()
	if now.Sub(rl.lastUpdate) >= rl.interval {
		rl.lastUpdate = now
		return true
	}

	return false
}

// NewProgress creates a new Progress instance with the specified parameters.
func NewProgress(totalBytes int64, callback ProgressCallback) *Progress {
	return &Progress{
		TotalBytes:  totalBytes,
		StartTime:   time.Now(),
		LastUpdate:  time.Now(),
		callback:    callback,
		rateLimiter: NewRateLimiter(500 * time.Millisecond), // Update every 500ms
	}
}

// Update updates the progress with new downloaded bytes count.
// It calculates speed and calls the callback if rate limiting allows it.
func (p *Progress) Update(downloadedBytes int64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.DownloadedBytes = downloadedBytes
	now := time.Now()

	// Calculate speed based on elapsed time
	elapsed := now.Sub(p.StartTime)
	if elapsed > 0 {
		p.Speed = int64(float64(downloadedBytes) / elapsed.Seconds())
	}

	p.LastUpdate = now

	// Only call callback if rate limiter allows it
	if p.callback != nil && p.rateLimiter.Allow() {
		p.callback(p.DownloadedBytes, p.TotalBytes, p.Speed)
	}
}

// ForceUpdate forces an immediate progress update, bypassing rate limiting.
func (p *Progress) ForceUpdate(downloadedBytes int64) {
	p.mutex.Lock()
	defer p.mutex.Unlock()

	p.DownloadedBytes = downloadedBytes
	now := time.Now()

	// Calculate speed based on elapsed time
	elapsed := now.Sub(p.StartTime)
	if elapsed > 0 {
		p.Speed = int64(float64(downloadedBytes) / elapsed.Seconds())
	}

	p.LastUpdate = now

	// Always call callback for forced updates
	if p.callback != nil {
		p.callback(p.DownloadedBytes, p.TotalBytes, p.Speed)
	}
}

// GetSnapshot returns a thread-safe snapshot of the current progress state.
func (p *Progress) GetSnapshot() ProgressSnapshot {
	p.mutex.RLock()
	defer p.mutex.RUnlock()

	return ProgressSnapshot{
		TotalBytes:      p.TotalBytes,
		DownloadedBytes: p.DownloadedBytes,
		Speed:           p.Speed,
		StartTime:       p.StartTime,
		LastUpdate:      p.LastUpdate,
	}
}

// ProgressSnapshot represents a point-in-time snapshot of progress data.
type ProgressSnapshot struct {
	TotalBytes      int64
	DownloadedBytes int64
	Speed           int64
	StartTime       time.Time
	LastUpdate      time.Time
}

// Percentage returns the completion percentage (0-100).
// Returns -1 if total size is unknown.
func (ps ProgressSnapshot) Percentage() float64 {
	if ps.TotalBytes <= 0 {
		return -1
	}

	return float64(ps.DownloadedBytes) / float64(ps.TotalBytes) * 100
}

// Elapsed returns the time elapsed since the download started.
func (ps ProgressSnapshot) Elapsed() time.Duration {
	return ps.LastUpdate.Sub(ps.StartTime)
}

// ETA returns the estimated time to completion.
// Returns -1 if total size is unknown or speed is zero.
func (ps ProgressSnapshot) ETA() time.Duration {
	if ps.TotalBytes <= 0 || ps.Speed <= 0 {
		return -1
	}

	remaining := ps.TotalBytes - ps.DownloadedBytes
	if remaining <= 0 {
		return 0
	}

	return time.Duration(float64(remaining)/float64(ps.Speed)) * time.Second
}

// String returns a human-readable representation of the progress.
func (ps ProgressSnapshot) String() string {
	if ps.TotalBytes <= 0 {
		return fmt.Sprintf("%s downloaded at %s/s",
			formatBytes(ps.DownloadedBytes),
			formatBytes(ps.Speed))
	}

	percentage := ps.Percentage()
	eta := ps.ETA()

	etaStr := "unknown"
	if eta >= 0 {
		etaStr = formatDuration(eta)
	}

	return fmt.Sprintf("%.1f%% (%s/%s) at %s/s, ETA: %s",
		percentage,
		formatBytes(ps.DownloadedBytes),
		formatBytes(ps.TotalBytes),
		formatBytes(ps.Speed),
		etaStr)
}

// ProgressReader wraps an io.Reader to track progress as data is read.
type ProgressReader struct {
	reader   io.Reader
	progress *Progress
	total    int64
}

// NewProgressReader creates a new ProgressReader that wraps the given reader.
func NewProgressReader(
	reader io.Reader,
	totalBytes int64,
	callback ProgressCallback,
) *ProgressReader {
	return &ProgressReader{
		reader:   reader,
		progress: NewProgress(totalBytes, callback),
		total:    0,
	}
}

// Read reads data from the underlying reader and updates progress.
func (pr *ProgressReader) Read(p []byte) (n int, err error) {
	n, err = pr.reader.Read(p)
	if n > 0 {
		pr.total += int64(n)
		pr.progress.Update(pr.total)
	}

	return n, err
}

// GetProgress returns the underlying progress tracker.
func (pr *ProgressReader) GetProgress() *Progress {
	return pr.progress
}

// formatBytes formats a byte count as a human-readable string.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// formatDuration formats a duration as a human-readable string.
func formatDuration(d time.Duration) string {
	if d < 0 {
		return unknownValue
	}

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}

	if d < time.Hour {
		minutes := int(d.Minutes())
		seconds := int(d.Seconds()) % 60

		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}

	hours := int(d.Hours())
	minutes := int(d.Minutes()) % 60

	return fmt.Sprintf("%dh%dm", hours, minutes)
}
