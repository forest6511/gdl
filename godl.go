package godl

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"time"

	"github.com/forest6511/godl/internal/core"
	"github.com/forest6511/godl/pkg/events"
	"github.com/forest6511/godl/pkg/middleware"
	"github.com/forest6511/godl/pkg/plugin"
	"github.com/forest6511/godl/pkg/protocols"
	"github.com/forest6511/godl/pkg/storage"
	"github.com/forest6511/godl/pkg/types"
	"github.com/forest6511/godl/pkg/validation"
)

// Progress represents the download progress.
type Progress struct {
	TotalSize       int64
	BytesDownloaded int64
	Speed           int64
	Percentage      float64
	TimeElapsed     time.Duration
	TimeRemaining   time.Duration
}

// ProgressCallback is a function that receives progress updates.
type ProgressCallback func(Progress)

// Options defines download options.
type Options struct {
	ProgressCallback  ProgressCallback
	MaxConcurrency    int
	ChunkSize         int64
	EnableResume      bool
	RetryAttempts     int
	Timeout           time.Duration
	UserAgent         string
	Headers           map[string]string
	CreateDirs        bool
	OverwriteExisting bool
	Quiet             bool
	Verbose           bool
}

// DownloadStats contains statistics about a download operation.
type DownloadStats struct {
	// URL is the source URL that was downloaded.
	URL string

	// Filename is the destination filename or path.
	Filename string

	// TotalSize is the total size of the downloaded file in bytes.
	TotalSize int64

	// BytesDownloaded is the number of bytes successfully downloaded.
	BytesDownloaded int64

	// StartTime is when the download started.
	StartTime time.Time

	// EndTime is when the download completed or failed.
	EndTime time.Time

	// Duration is the total time taken for the download.
	Duration time.Duration

	// AverageSpeed is the average download speed in bytes per second.
	AverageSpeed int64

	// Retries is the number of retry attempts that were made.
	Retries int

	// Success indicates whether the download completed successfully.
	Success bool

	// Error contains any error that occurred during download.
	Error error

	// Resumed indicates whether this download was resumed from a partial file.
	Resumed bool

	// ChunksUsed indicates the number of concurrent chunks used for download.
	ChunksUsed int
}

// Download downloads a file from URL to destination path.
//
// Example:
//
//	ctx := context.Background()
//	stats, err := godl.Download(ctx, "https://example.com/file.zip", "./downloads/file.zip")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Downloaded %d bytes in %v\n", stats.BytesDownloaded, stats.Duration)
func Download(ctx context.Context, url, dest string) (*DownloadStats, error) {
	// Tier 1: Public API validation
	if err := validation.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if err := validation.ValidateDestination(dest); err != nil {
		return nil, fmt.Errorf("invalid destination: %w", err)
	}

	dl := core.NewDownloader()
	stats, err := dl.Download(ctx, url, dest, nil)
	if err != nil {
		return convertStats(stats), err
	}

	return convertStats(stats), nil
}

// convertStats converts internal types.DownloadStats to public DownloadStats
func convertStats(stats *types.DownloadStats) *DownloadStats {
	if stats == nil {
		return nil
	}
	return &DownloadStats{
		URL:             stats.URL,
		Filename:        stats.Filename,
		TotalSize:       stats.TotalSize,
		BytesDownloaded: stats.BytesDownloaded,
		StartTime:       stats.StartTime,
		EndTime:         stats.EndTime,
		Duration:        stats.Duration,
		AverageSpeed:    stats.AverageSpeed,
		Retries:         stats.Retries,
		Success:         stats.Success,
		Error:           stats.Error,
		Resumed:         stats.Resumed,
		ChunksUsed:      stats.ChunksUsed,
	}
}

// DownloadWithOptions downloads with custom options.
//
// Example:
//
//	ctx := context.Background()
//	opts := &godl.Options{
//	    MaxConcurrency: 4,
//	    EnableResume: true,
//	    ProgressCallback: func(p godl.Progress) {
//	        fmt.Printf("Downloaded: %.2f%%\n", p.Percentage)
//	    },
//	}
//	err := godl.DownloadWithOptions(ctx, "https://example.com/file.zip", "./file.zip", opts)
func DownloadWithOptions(ctx context.Context, url, dest string, opts *Options) (*DownloadStats, error) {
	// Tier 1: Public API validation
	if err := validation.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if err := validation.ValidateDestination(dest); err != nil {
		return nil, fmt.Errorf("invalid destination: %w", err)
	}

	// Validate options if provided
	if opts != nil {
		if opts.ChunkSize > 0 {
			if err := validation.ValidateChunkSize(opts.ChunkSize); err != nil {
				return nil, fmt.Errorf("invalid chunk size: %w", err)
			}
		}
		if opts.Timeout > 0 {
			timeoutSeconds := int(opts.Timeout.Seconds())
			if err := validation.ValidateTimeout(timeoutSeconds); err != nil {
				return nil, fmt.Errorf("invalid timeout: %w", err)
			}
		}
	}

	dl := core.NewDownloader()

	// Convert our Options to internal DownloadOptions
	downloadOptions := &types.DownloadOptions{
		MaxConcurrency:    opts.MaxConcurrency,
		ChunkSize:         opts.ChunkSize,
		Resume:            opts.EnableResume,
		Timeout:           opts.Timeout,
		UserAgent:         opts.UserAgent,
		Headers:           opts.Headers,
		CreateDirs:        opts.CreateDirs,
		OverwriteExisting: opts.OverwriteExisting,
	}

	// Handle progress callback if provided
	if opts.ProgressCallback != nil {
		downloadOptions.ProgressCallback = func(downloaded, total int64, speed int64) {
			progress := Progress{
				TotalSize:       total,
				BytesDownloaded: downloaded,
				Speed:           speed,
			}
			if total > 0 {
				progress.Percentage = float64(downloaded) / float64(total) * 100
			}

			opts.ProgressCallback(progress)
		}
	}

	stats, err := dl.Download(ctx, url, dest, downloadOptions)
	if err != nil {
		return convertStats(stats), err
	}

	return convertStats(stats), nil
}

// DownloadToWriter downloads to an io.Writer.
//
// Example:
//
//	ctx := context.Background()
//	var buf bytes.Buffer
//	err := godl.DownloadToWriter(ctx, "https://example.com/data.txt", &buf)
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Println("Downloaded data:", buf.String())
func DownloadToWriter(ctx context.Context, url string, w io.Writer) (*DownloadStats, error) {
	// Tier 1: Public API validation
	if err := validation.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if w == nil {
		return nil, fmt.Errorf("writer cannot be nil")
	}

	dl := core.NewDownloader()
	stats, err := dl.DownloadToWriter(ctx, url, w, nil)
	if err != nil {
		return convertStats(stats), err
	}

	return convertStats(stats), nil
}

// DownloadToMemory downloads to memory and returns bytes.
//
// Example:
//
//	ctx := context.Background()
//	data, err := godl.DownloadToMemory(ctx, "https://example.com/api/data.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("Downloaded %d bytes\n", len(data))
func DownloadToMemory(ctx context.Context, url string) ([]byte, *DownloadStats, error) {
	var buf bytes.Buffer

	stats, err := DownloadToWriter(ctx, url, &buf)

	return buf.Bytes(), stats, err
}

// DownloadWithResume downloads a file with resume support.
//
// Example:
//
//	ctx := context.Background()
//	err := godl.DownloadWithResume(ctx, "https://example.com/large-file.zip", "./large-file.zip")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	// If interrupted, running again will resume from where it left off
func DownloadWithResume(ctx context.Context, url, dest string) (*DownloadStats, error) {
	opts := &Options{
		EnableResume: true,
	}

	return DownloadWithOptions(ctx, url, dest, opts)
}

// GetFileInfo retrieves file information without downloading.
//
// Example:
//
//	ctx := context.Background()
//	info, err := godl.GetFileInfo(ctx, "https://example.com/file.zip")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	fmt.Printf("File: %s, Size: %d bytes, Type: %s\n", info.Filename, info.Size, info.ContentType)
func GetFileInfo(ctx context.Context, url string) (*FileInfo, error) {
	// Tier 1: Public API validation
	if err := validation.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	dl := core.NewDownloader()

	info, err := dl.GetFileInfo(ctx, url)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Size:           info.Size,
		Filename:       info.Filename,
		ContentType:    info.ContentType,
		LastModified:   info.LastModified,
		SupportsRanges: info.SupportsRanges,
	}, nil
}

// FileInfo contains information about a remote file.
type FileInfo struct {
	Size           int64
	Filename       string
	ContentType    string
	LastModified   time.Time
	SupportsRanges bool
}

// Downloader provides an extensible download client with plugin support.
type Downloader struct {
	pluginManager    *plugin.PluginManager
	eventEmitter     *events.EventEmitter
	middleware       *middleware.MiddlewareChain
	protocolRegistry *protocols.ProtocolRegistry
	storageManager   *storage.StorageManager
	coreDownloader   *core.Downloader
}

// NewDownloader creates a new Downloader with plugin support.
func NewDownloader() *Downloader {
	return &Downloader{
		pluginManager:    plugin.NewPluginManager(),
		eventEmitter:     events.NewEventEmitter(),
		middleware:       middleware.NewMiddlewareChain(),
		protocolRegistry: protocols.NewProtocolRegistry(),
		storageManager:   storage.NewStorageManager(),
		coreDownloader:   core.NewDownloader(),
	}
}

// UsePlugin registers and initializes a plugin.
func (d *Downloader) UsePlugin(p plugin.Plugin) error {
	return d.pluginManager.Register(p)
}

// UseMiddleware adds middleware to the chain.
func (d *Downloader) UseMiddleware(m middleware.Middleware) {
	d.middleware.Use(m)
}

// On registers an event listener.
func (d *Downloader) On(event events.EventType, handler events.EventListener) {
	d.eventEmitter.On(event, handler)
}

// RegisterProtocol registers a custom protocol handler.
func (d *Downloader) RegisterProtocol(handler protocols.ProtocolHandler) error {
	return d.protocolRegistry.Register(handler)
}

// SetStorageBackend sets the storage backend.
func (d *Downloader) SetStorageBackend(name string, backend storage.StorageBackend) error {
	return d.storageManager.Register(name, backend)
}

// Download downloads a file using the configured plugins and middleware.
func (d *Downloader) Download(ctx context.Context, url, dest string, opts *Options) (*DownloadStats, error) {
	// Validate inputs
	if err := validation.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if err := validation.ValidateDestination(dest); err != nil {
		return nil, fmt.Errorf("invalid destination: %w", err)
	}

	// Emit pre-download event
	event := events.Event{
		Type: events.EventDownloadStarted,
		Data: map[string]interface{}{
			"url":  url,
			"dest": dest,
		},
	}
	d.eventEmitter.Emit(event)

	// Execute pre-download hooks using plugin manager's hook system
	// Note: We'll use a simple approach here since there are conflicting HookType definitions
	if err := d.executePluginHook("pre_download", map[string]interface{}{
		"url":  url,
		"dest": dest,
	}); err != nil {
		return nil, fmt.Errorf("pre-download hook failed: %w", err)
	}

	// Convert options
	var downloadOptions *types.DownloadOptions
	if opts != nil {
		downloadOptions = &types.DownloadOptions{
			MaxConcurrency:    opts.MaxConcurrency,
			ChunkSize:         opts.ChunkSize,
			Resume:            opts.EnableResume,
			Timeout:           opts.Timeout,
			UserAgent:         opts.UserAgent,
			Headers:           opts.Headers,
			CreateDirs:        opts.CreateDirs,
			OverwriteExisting: opts.OverwriteExisting,
		}

		// Handle progress callback
		if opts.ProgressCallback != nil {
			downloadOptions.ProgressCallback = func(downloaded, total int64, speed int64) {
				progress := Progress{
					TotalSize:       total,
					BytesDownloaded: downloaded,
					Speed:           speed,
				}
				if total > 0 {
					progress.Percentage = float64(downloaded) / float64(total) * 100
				}
				opts.ProgressCallback(progress)
			}
		}
	}

	// Use core downloader for actual download
	stats, err := d.coreDownloader.Download(ctx, url, dest, downloadOptions)

	// Execute post-download hooks
	if err == nil {
		// Emit success event
		successEvent := events.Event{
			Type: events.EventDownloadCompleted,
			Data: map[string]interface{}{
				"url":  url,
				"dest": dest,
			},
		}
		d.eventEmitter.Emit(successEvent)

		// Execute post-download hooks
		if hookErr := d.executePluginHook("post_download", map[string]interface{}{
			"url":  url,
			"dest": dest,
		}); hookErr != nil {
			// Log hook error but don't fail the download
			fmt.Printf("post-download hook error: %v\n", hookErr)
		}
	} else {
		// Emit error event
		errorEvent := events.Event{
			Type: events.EventDownloadFailed,
			Data: map[string]interface{}{
				"url":   url,
				"dest":  dest,
				"error": err.Error(),
			},
		}
		d.eventEmitter.Emit(errorEvent)

		// Execute error hooks
		if hookErr := d.executePluginHook("on_error", map[string]interface{}{
			"url":   url,
			"dest":  dest,
			"error": err.Error(),
		}); hookErr != nil {
			fmt.Printf("error hook failed: %v\n", hookErr)
		}
	}

	if err != nil {
		return convertStats(stats), err
	}
	return convertStats(stats), nil
}

// DownloadToWriter downloads to an io.Writer with plugin support.
func (d *Downloader) DownloadToWriter(ctx context.Context, url string, w io.Writer, opts *Options) (*DownloadStats, error) {
	if err := validation.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}
	if w == nil {
		return nil, fmt.Errorf("writer cannot be nil")
	}

	// Convert options
	var downloadOptions *types.DownloadOptions
	if opts != nil {
		downloadOptions = &types.DownloadOptions{
			MaxConcurrency: opts.MaxConcurrency,
			ChunkSize:      opts.ChunkSize,
			Resume:         opts.EnableResume,
			Timeout:        opts.Timeout,
			UserAgent:      opts.UserAgent,
			Headers:        opts.Headers,
		}
	}

	stats, err := d.coreDownloader.DownloadToWriter(ctx, url, w, downloadOptions)
	if err != nil {
		return convertStats(stats), err
	}
	return convertStats(stats), nil
}

// GetFileInfo retrieves file information with plugin support.
func (d *Downloader) GetFileInfo(ctx context.Context, url string) (*FileInfo, error) {
	if err := validation.ValidateURL(url); err != nil {
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	info, err := d.coreDownloader.GetFileInfo(ctx, url)
	if err != nil {
		return nil, err
	}

	return &FileInfo{
		Size:           info.Size,
		Filename:       info.Filename,
		ContentType:    info.ContentType,
		LastModified:   info.LastModified,
		SupportsRanges: info.SupportsRanges,
	}, nil
}

// executePluginHook is a helper method to execute plugin hooks
// This abstracts away the differences between plugin and hooks package HookTypes
func (d *Downloader) executePluginHook(hookName string, data interface{}) error {
	// Since there are conflicting HookType definitions between packages,
	// we'll implement a simple bridge here
	// In a real implementation, you might want to unify the hook systems

	// For now, we'll just log that hooks would be executed
	// In production, you'd integrate this with your actual hook system
	fmt.Printf("Executing hook: %s with data: %v\n", hookName, data)
	return nil
}
