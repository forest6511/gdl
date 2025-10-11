// Package types defines the core types and interfaces for the gdl download library.
package types

import (
	"context"
	"io"
	"time"
)

// Downloader defines the interface for downloading files from various sources.
// Implementations should handle different protocols and provide progress tracking.
type Downloader interface {
	// Download downloads a file from the given URL to the specified destination.
	// It returns download statistics and any error that occurred during the process.
	Download(
		ctx context.Context,
		url, destination string,
		options *DownloadOptions,
	) (*DownloadStats, error)

	// DownloadToWriter downloads a file from the given URL and writes it to the provided writer.
	// This method is useful when you want to process the downloaded content without saving to disk.
	DownloadToWriter(
		ctx context.Context,
		url string,
		writer io.Writer,
		options *DownloadOptions,
	) (*DownloadStats, error)

	// GetFileInfo retrieves information about a file without downloading it.
	// This can be used to check file size, modification time, and other metadata.
	GetFileInfo(ctx context.Context, url string) (*FileInfo, error)
}

// Storage defines the interface for different storage backends.
// This allows for flexible storage options like local filesystem, cloud storage, etc.
type Storage interface {
	// Store saves data to the storage backend at the specified key/path.
	Store(ctx context.Context, key string, data io.Reader) error

	// Retrieve retrieves data from the storage backend for the given key/path.
	Retrieve(ctx context.Context, key string) (io.ReadCloser, error)

	// Delete removes data from the storage backend for the given key/path.
	Delete(ctx context.Context, key string) error

	// Exists checks if data exists at the given key/path in the storage backend.
	Exists(ctx context.Context, key string) (bool, error)

	// Size returns the size of the data at the given key/path.
	Size(ctx context.Context, key string) (int64, error)
}

// Progress defines the interface for tracking download progress.
// Implementations can provide various forms of progress reporting like console output,
// GUI progress bars, or logging.
type Progress interface {
	// Start is called when a download begins.
	Start(filename string, totalSize int64)

	// Update is called periodically during download to report progress.
	Update(bytesDownloaded, totalSize int64, speed int64)

	// Finish is called when a download completes successfully.
	Finish(filename string, stats *DownloadStats)

	// Error is called when a download fails.
	Error(filename string, err error)
}

// DownloadOptions contains configuration options for downloads.
type DownloadOptions struct {
	// Destination specifies the destination file path for the download.
	// If empty, a filename will be extracted from the URL.
	Destination string

	// MaxRetries specifies the maximum number of retry attempts for failed downloads.
	MaxRetries int

	// RetryDelay specifies the delay between retry attempts.
	RetryDelay time.Duration

	// Timeout specifies the maximum time to wait for a download to complete.
	Timeout time.Duration

	// ChunkSize specifies the size of chunks to download at a time.
	// A larger chunk size may improve performance but uses more memory.
	ChunkSize int64

	// UserAgent specifies the User-Agent header to use for HTTP requests.
	UserAgent string

	// Headers contains additional HTTP headers to send with requests.
	Headers map[string]string

	// Progress specifies the progress tracker to use for this download.
	Progress Progress

	// Resume indicates whether to resume partial downloads if supported.
	Resume bool

	// OverwriteExisting indicates whether to overwrite existing files.
	OverwriteExisting bool

	// CreateDirs indicates whether to create parent directories if they don't exist.
	CreateDirs bool

	// MaxConcurrency specifies the maximum number of concurrent download chunks.
	// Only applicable for downloads that support parallel downloading.
	MaxConcurrency int

	// ProgressCallback is called periodically during download to report progress.
	// If set, this takes precedence over the Progress interface.
	ProgressCallback func(bytesDownloaded, totalBytes int64, speed int64)

	// MaxRedirects specifies the maximum number of HTTP redirects to follow.
	MaxRedirects int

	// InsecureSkipVerify skips TLS certificate verification when true.
	InsecureSkipVerify bool

	// ProxyURL specifies the HTTP proxy URL to use for requests.
	ProxyURL string

	// MaxRate specifies the maximum download rate in bytes per second.
	// A value of 0 means unlimited bandwidth.
	MaxRate int64
}

// DownloadStats contains statistics about a completed or failed download.
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

// DownloadError represents errors that can occur during downloads.
// It provides structured error information with additional context.
type DownloadError struct {
	// URL is the source URL that caused the error.
	URL string

	// Operation describes what operation was being performed when the error occurred.
	Operation string

	// Err is the underlying error that caused this download error.
	Err error

	// StatusCode is the HTTP status code if applicable.
	StatusCode int

	// Retryable indicates whether this error can be retried.
	Retryable bool

	// Temporary indicates whether this is a temporary error.
	Temporary bool
}

// Error implements the error interface for DownloadError.
func (e *DownloadError) Error() string {
	if e.Err != nil {
		return e.Err.Error()
	}

	return "download error"
}

// Unwrap returns the underlying error for error unwrapping.
func (e *DownloadError) Unwrap() error {
	return e.Err
}

// FileInfo contains metadata about a file available for download.
type FileInfo struct {
	// URL is the source URL of the file.
	URL string

	// Size is the size of the file in bytes.
	Size int64

	// LastModified is the last modification time of the file.
	LastModified time.Time

	// ContentType is the MIME type of the file.
	ContentType string

	// Filename is the suggested filename for the file.
	Filename string

	// SupportsRanges indicates whether the server supports range requests.
	SupportsRanges bool

	// Headers contains the response headers from the server.
	Headers map[string][]string
}
