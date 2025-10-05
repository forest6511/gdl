// Package core provides the core implementation of the gdl download functionality.
package core

import (
	"context"
	stdErrors "errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/forest6511/gdl/internal/network"
	"github.com/forest6511/gdl/internal/recovery"
	"github.com/forest6511/gdl/internal/resume"
	"github.com/forest6511/gdl/internal/retry"
	"github.com/forest6511/gdl/internal/storage"
	"github.com/forest6511/gdl/pkg/errors"
	"github.com/forest6511/gdl/pkg/progress"
	"github.com/forest6511/gdl/pkg/ratelimit"
	"github.com/forest6511/gdl/pkg/types"
)

// DefaultChunkSize is the default size for reading chunks during download.
const DefaultChunkSize = 32 * 1024 // 32KB

// DefaultTimeout is the default timeout for download operations.
const DefaultTimeout = 30 * time.Minute

// DefaultUserAgent is the default User-Agent string used for HTTP requests.
const DefaultUserAgent = "gdl/1.0"

// Downloader implements the types.Downloader interface and provides
// comprehensive HTTP/HTTPS download functionality with error handling,
// retry mechanisms, disk space management, and recovery capabilities.
const (
	defaultFilename = "download"
)

type Downloader struct {
	client          *http.Client
	retryManager    *retry.RetryManager
	spaceChecker    *storage.SpaceChecker
	networkDiag     *network.Diagnostics
	recoveryAdvisor *recovery.RecoveryAdvisor
	logger          *log.Logger
	enableLogging   bool
	lightweight     *LightweightDownloader
	zeroCopy        *ZeroCopyDownloader
	bufferPool      *BufferPool
	connectionPool  *network.ConnectionPool
	platformInfo    *PlatformInfo
	resumeManager   *resume.Manager
}

// NewDownloader creates a new Downloader instance with default settings.
func NewDownloader() *Downloader {
	// Detect platform and apply optimizations
	platformInfo := DetectPlatform()

	// Create optimized HTTP client for platform
	client := network.CreateOptimizedClient(DefaultTimeout)

	// Apply platform optimizations to the HTTP transport
	if transport, ok := client.Transport.(*http.Transport); ok {
		ApplyPlatformOptimizations(transport)
	}

	retryManager := retry.NewRetryManager().
		WithMaxRetries(3).
		WithBaseDelay(100 * time.Millisecond).
		WithMaxDelay(2 * time.Second).
		WithBackoffFactor(2.0).
		WithJitter(true)

	// Get user's home directory for resume files
	homeDir, err := os.UserHomeDir()
	resumeDir := "."
	if err == nil {
		resumeDir = filepath.Join(homeDir, ".gdl", "resume")
	}

	return &Downloader{
		client:          client,
		retryManager:    retryManager,
		spaceChecker:    storage.NewSpaceChecker(),
		networkDiag:     network.NewDiagnostics(),
		recoveryAdvisor: recovery.NewRecoveryAdvisor(),
		logger:          log.New(os.Stderr, "[GODL] ", log.LstdFlags),
		enableLogging:   false, // Disabled by default, can be enabled via WithLogging
		lightweight:     NewLightweightDownloader(),
		zeroCopy:        NewZeroCopyDownloader(),
		bufferPool:      GlobalBufferPool,
		connectionPool: network.NewConnectionPool(
			platformInfo.Optimizations.MaxConnections,
			platformInfo.Optimizations.MaxConnections,
		),
		platformInfo:  platformInfo,
		resumeManager: resume.NewManager(resumeDir),
	}
}

// NewDownloaderWithClient creates a new Downloader instance with a custom HTTP client.
func NewDownloaderWithClient(client *http.Client) *Downloader {
	downloader := NewDownloader()
	downloader.client = client

	return downloader
}

// WithLogging enables or disables enhanced logging with error context.
func (d *Downloader) WithLogging(enabled bool) *Downloader {
	d.enableLogging = enabled
	return d
}

// WithRetryStrategy configures the retry strategy for downloads.
func (d *Downloader) WithRetryStrategy(manager *retry.RetryManager) *Downloader {
	d.retryManager = manager
	return d
}

// WithSpaceChecker configures the disk space checker.
func (d *Downloader) WithSpaceChecker(checker *storage.SpaceChecker) *Downloader {
	d.spaceChecker = checker
	return d
}

// logError logs error messages with enhanced context when logging is enabled.
func (d *Downloader) logError(operation string, err error, context map[string]interface{}) {
	if !d.enableLogging || d.logger == nil {
		return
	}

	logMsg := fmt.Sprintf("Operation: %s, Error: %v", operation, err)
	if len(context) > 0 {
		logMsg += ", Context: "
		for key, value := range context {
			logMsg += fmt.Sprintf("%s=%v ", key, value)
		}
	}

	d.logger.Println(logMsg)
}

// logInfo logs informational messages when logging is enabled.
func (d *Downloader) logInfo(operation string, message string, context map[string]interface{}) {
	if !d.enableLogging || d.logger == nil {
		return
	}

	logMsg := fmt.Sprintf("Operation: %s, Info: %s", operation, message)
	if len(context) > 0 {
		logMsg += ", Context: "
		for key, value := range context {
			logMsg += fmt.Sprintf("%s=%v ", key, value)
		}
	}

	d.logger.Println(logMsg)
}

// Download downloads a file from the given URL to the specified destination.
// It implements the types.Downloader interface with comprehensive error handling,
// retry logic, disk space management, and recovery capabilities.
func (d *Downloader) initializeDownloadStats(url, destination string, startTime time.Time) *types.DownloadStats {
	stats := &types.DownloadStats{
		URL:       url,
		Filename:  destination,
		StartTime: startTime,
	}

	d.logInfo("download_start", "Starting download", map[string]interface{}{
		"url":         url,
		"destination": destination,
	})

	return stats
}

func (d *Downloader) validateDownloadRequest(url, destination string, stats *types.DownloadStats) error {
	if err := d.validateURL(url); err != nil {
		downloadErr := d.wrapDownloadError(err, url, destination, 0, 0)
		stats.Error = downloadErr
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)

		d.logError("url_validation", err, map[string]interface{}{
			"url": url,
		})

		return downloadErr
	}
	return nil
}

func (d *Downloader) performPreDownloadChecks(destination string, options *types.DownloadOptions, stats *types.DownloadStats) error {
	if options.CreateDirs {
		// For CreateDirs case, check space in existing parent directory
		diskCheckPath, err := d.findExistingParentDir(destination)
		if err != nil {
			return d.handleDiskSpaceError(err, stats, destination, diskCheckPath)
		}

		if err := d.checkDiskSpaceForPath(diskCheckPath, 0); err != nil {
			return d.handleDiskSpaceError(err, stats, destination, diskCheckPath)
		}
	} else {
		// For non-CreateDirs case, check if the parent directory exists first
		if err := d.validateParentDirectory(destination); err != nil {
			return d.handleParentDirError(err, stats, destination)
		}

		// Parent exists, check disk space
		if err := d.checkDiskSpace(destination, 0); err != nil {
			return d.handleDiskSpaceError(err, stats, destination, destination)
		}
	}

	return nil
}

func (d *Downloader) findExistingParentDir(destination string) (string, error) {
	diskCheckPath := destination
	for {
		parent := filepath.Dir(diskCheckPath)
		// Check for root directory on all platforms
		if parent == diskCheckPath {
			return ".", nil
		}

		// On Windows, check for drive root (e.g., "C:\")
		// On Unix-like systems, check for root "/"
		if parent == "." || parent == "/" || filepath.VolumeName(parent) == parent {
			return ".", nil
		}

		if _, err := os.Stat(parent); err == nil {
			return parent, nil
		}

		diskCheckPath = parent
	}
}

func (d *Downloader) validateParentDirectory(destination string) error {
	parentDir := filepath.Dir(destination)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		return errors.NewDownloadErrorWithDetails(
			errors.CodePermissionDenied,
			"Parent directory does not exist",
			fmt.Sprintf("Cannot create file in non-existent directory: %s", parentDir),
		)
	}
	return nil
}

func (d *Downloader) handleDiskSpaceError(err error, stats *types.DownloadStats, destination, checkPath string) error {
	downloadErr := d.wrapDownloadError(err, stats.URL, destination, 0, 0)
	stats.Error = downloadErr
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	d.logError("disk_space_check", err, map[string]interface{}{
		"destination": destination,
		"check_path":  checkPath,
	})

	return downloadErr
}

func (d *Downloader) handleParentDirError(err error, stats *types.DownloadStats, destination string) error {
	downloadErr := err.(*errors.DownloadError)
	downloadErr.URL = stats.URL
	downloadErr.Filename = destination
	stats.Error = downloadErr
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	d.logError("parent_directory_check", downloadErr, map[string]interface{}{
		"destination": destination,
		"parent_dir":  filepath.Dir(destination),
	})

	return downloadErr
}

func (d *Downloader) executeDownloadWithRetries(
	ctx context.Context,
	url, destination string,
	options *types.DownloadOptions,
	stats *types.DownloadStats,
) (*types.DownloadStats, error) {
	var (
		attemptCount    int
		previousActions []recovery.ActionType
		lastErr         error
	)

	for attemptCount = 1; attemptCount <= d.retryManager.MaxRetries+1; attemptCount++ {
		d.logInfo(
			"download_attempt",
			fmt.Sprintf("Attempt %d", attemptCount),
			map[string]interface{}{
				"attempt": attemptCount,
				"url":     url,
			},
		)

		downloadStats, err := d.performDownloadAttempt(ctx, url, destination, options, attemptCount)
		lastErr = err

		if err == nil {
			d.logInfo("download_success", "Download completed successfully", map[string]interface{}{
				"url":              url,
				"bytes_downloaded": downloadStats.BytesDownloaded,
				"attempts":         attemptCount,
			})
			return downloadStats, nil
		}

		// Handle failure and check for retry
		if shouldStop := d.handleDownloadFailure(ctx, err, url, destination, attemptCount, downloadStats, &previousActions); shouldStop {
			break
		}

		if attemptCount >= d.retryManager.MaxRetries+1 || !errors.IsRetryable(err) {
			break
		}

		// Wait before retry
		if err := d.waitForRetry(ctx, attemptCount); err != nil {
			return stats, err
		}
	}

	// All attempts failed
	return d.handleFinalFailure(stats, lastErr, url, destination, attemptCount)
}

func (d *Downloader) handleDownloadFailure(
	ctx context.Context,
	err error,
	url, destination string,
	attemptCount int,
	downloadStats *types.DownloadStats,
	previousActions *[]recovery.ActionType,
) bool {
	d.logError("download_attempt_failed", err, map[string]interface{}{
		"attempt":     attemptCount,
		"url":         url,
		"destination": destination,
	})

	// Use recovery advisor for failure analysis
	var bytesDownloaded, totalSize int64
	if downloadStats != nil {
		bytesDownloaded = downloadStats.BytesDownloaded
		totalSize = downloadStats.TotalSize
	}

	attemptStart := time.Now()
	analysis, analysisErr := d.recoveryAdvisor.AnalyzeFailure(
		ctx,
		err,
		url,
		attemptCount,
		bytesDownloaded,
		totalSize,
		time.Since(attemptStart),
		*previousActions,
	)

	if analysisErr != nil {
		d.logError("failure_analysis", analysisErr, map[string]interface{}{
			"original_error": err.Error(),
		})
		return false
	}

	// Generate recovery recommendation
	recommendation := d.recoveryAdvisor.GenerateRecoveryRecommendation(ctx, analysis)
	d.logInfo("recovery_analysis", "Generated recovery recommendation", map[string]interface{}{
		"failure_type":           analysis.FailureType.String(),
		"recommended_actions":    len(recommendation.RecommendedActions),
		"estimated_success_rate": recommendation.EstimatedSuccessRate,
	})

	// Update previous actions for next attempt
	if len(recommendation.RecommendedActions) > 0 {
		*previousActions = append(*previousActions, recommendation.RecommendedActions[0].Type)
	}

	return false
}

func (d *Downloader) waitForRetry(ctx context.Context, attemptCount int) error {
	delay := d.retryManager.NextDelay(attemptCount - 1)
	d.logInfo(
		"retry_delay",
		fmt.Sprintf("Waiting %v before retry", delay),
		map[string]interface{}{
			"delay":   delay.String(),
			"attempt": attemptCount,
		},
	)

	select {
	case <-ctx.Done():
		return errors.WrapError(
			ctx.Err(),
			errors.CodeCancelled,
			"Download cancelled during retry delay",
		)
	case <-time.After(delay):
		return nil
	}
}

func (d *Downloader) handleFinalFailure(
	stats *types.DownloadStats,
	lastErr error,
	url, destination string,
	attemptCount int,
) (*types.DownloadStats, error) {
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)
	stats.Retries = attemptCount - 1

	finalErr := d.createFinalError(lastErr, url, destination, attemptCount)
	stats.Error = finalErr

	d.logError("download_final_failure", finalErr, map[string]interface{}{
		"total_attempts": attemptCount - 1,
		"url":            url,
		"destination":    destination,
	})

	return stats, finalErr
}

func (d *Downloader) createFinalError(lastErr error, url, destination string, attemptCount int) *errors.DownloadError {
	if lastErr != nil && !errors.IsRetryable(lastErr) {
		downloadErr := &errors.DownloadError{}
		if stdErrors.As(lastErr, &downloadErr) {
			downloadErr.URL = url
			downloadErr.Filename = destination
			return downloadErr
		}
	}

	// Handle retryable errors that exhausted all retries
	if lastErr != nil {
		downloadErr := &errors.DownloadError{}
		if stdErrors.As(lastErr, &downloadErr) {
			return &errors.DownloadError{
				Code:    downloadErr.Code,
				Message: downloadErr.Message,
				Details: fmt.Sprintf(
					"Download failed after %d attempts. Last error: %s",
					attemptCount-1,
					downloadErr.Details,
				),
				URL:              url,
				Filename:         destination,
				Underlying:       downloadErr.Underlying,
				HTTPStatusCode:   downloadErr.HTTPStatusCode,
				BytesTransferred: downloadErr.BytesTransferred,
				Retryable:        false,
			}
		}

		// For non-DownloadError, wrap it
		return &errors.DownloadError{
			Code:       errors.CodeUnknown,
			Message:    fmt.Sprintf("Download failed after %d attempts: %s", attemptCount-1, lastErr.Error()),
			Details:    fmt.Sprintf("All retry attempts exhausted for URL: %s", url),
			URL:        url,
			Filename:   destination,
			Underlying: lastErr,
			Retryable:  false,
		}
	}

	// Fallback if no last error
	finalErr := errors.NewDownloadErrorWithDetails(
		errors.CodeUnknown,
		fmt.Sprintf("Download failed after %d attempts", attemptCount-1),
		fmt.Sprintf("All retry attempts exhausted for URL: %s", url),
	)
	finalErr.URL = url
	finalErr.Filename = destination
	finalErr.Retryable = false
	return finalErr
}

func (d *Downloader) Download(
	ctx context.Context,
	url, destination string,
	options *types.DownloadOptions,
) (*types.DownloadStats, error) {
	startTime := time.Now()
	stats := d.initializeDownloadStats(url, destination, startTime)

	// Set default options and validate inputs
	if options == nil {
		options = &types.DownloadOptions{}
	}
	d.setDefaultOptions(options)

	if err := d.validateDownloadRequest(url, destination, stats); err != nil {
		return stats, err
	}

	// Check disk space before starting
	if err := d.performPreDownloadChecks(destination, options, stats); err != nil {
		return stats, err
	}

	// Main download loop with retry logic
	return d.executeDownloadWithRetries(ctx, url, destination, options, stats)
}

// checkDiskSpace validates available disk space for the download.
func (d *Downloader) checkDiskSpace(destination string, estimatedSize uint64) error {
	dir := filepath.Dir(destination)
	return d.checkDiskSpaceForPath(dir, estimatedSize)
}

// checkDiskSpaceForPath validates available disk space for a specific path.
func (d *Downloader) checkDiskSpaceForPath(path string, estimatedSize uint64) error {
	if d.spaceChecker == nil {
		return nil // Skip check if no space checker configured
	}

	// If we have an estimated size, check if we have enough space
	if estimatedSize > 0 {
		return d.spaceChecker.CheckAvailableSpace(path, estimatedSize)
	}

	// Otherwise, just check general space availability
	spaceInfo, err := d.spaceChecker.GetSpaceInfo(path)
	if err != nil {
		return errors.WrapError(err, errors.CodeInsufficientSpace, "Failed to check disk space")
	}

	// Check if we have at least 100MB free space as a safety buffer
	minSpace := uint64(100 * 1024 * 1024) // 100MB
	if spaceInfo.AvailableBytes < minSpace {
		return errors.NewDownloadErrorWithDetails(
			errors.CodeInsufficientSpace,
			"Insufficient disk space",
			fmt.Sprintf(
				"Available: %d bytes, minimum required: %d bytes",
				spaceInfo.AvailableBytes,
				minSpace,
			),
		)
	}

	return nil
}

// wrapDownloadError wraps any error as a DownloadError with context.
func (d *Downloader) wrapDownloadError(
	err error,
	url, filename string,
	bytesTransferred, totalSize int64,
) *errors.DownloadError {
	// If it's already a DownloadError, enhance it with additional context
	downloadErr := &errors.DownloadError{}
	if stdErrors.As(err, &downloadErr) {
		if downloadErr.URL == "" {
			downloadErr.URL = url
		}

		if downloadErr.Filename == "" {
			downloadErr.Filename = filename
		}

		if downloadErr.BytesTransferred == 0 {
			downloadErr.BytesTransferred = bytesTransferred
		}

		return downloadErr
	}

	// Wrap as new DownloadError
	downloadErr = errors.WrapErrorWithURL(
		err,
		errors.CodeUnknown,
		"Download operation failed",
		url,
	)
	downloadErr.Filename = filename
	downloadErr.BytesTransferred = bytesTransferred

	return downloadErr
}

// performDownloadAttempt performs a single download attempt.
func (d *Downloader) performDownloadAttempt(
	ctx context.Context,
	url, destination string,
	options *types.DownloadOptions,
	attemptCount int,
) (*types.DownloadStats, error) {
	d.logInfo("attempt_start", "Starting download attempt", map[string]interface{}{
		"attempt": attemptCount,
		"url":     url,
	})

	// Check if file exists and handle accordingly (only if not resuming)
	if !options.Resume {
		if err := d.handleExistingFile(destination, options); err != nil {
			return nil, d.wrapDownloadError(err, url, destination, 0, 0)
		}
	}

	// Create parent directories if needed
	if options.CreateDirs {
		if err := d.createParentDirs(destination); err != nil {
			return nil, errors.WrapErrorWithURL(err, errors.CodePermissionDenied,
				"Failed to create parent directories", url)
		}
	}

	// Get file info to check server capabilities and file size with retry
	fileInfo, err := d.GetFileInfo(ctx, url)
	if err != nil {
		// Fall back to simple download if HEAD request fails
		d.logInfo(
			"head_request_failed",
			"HEAD request failed, falling back to simple download",
			map[string]interface{}{
				"error": err.Error(),
			},
		)

		return d.performSimpleDownload(ctx, url, destination, options)
	}

	// Check if we should use lightweight mode for small files
	// Only use lightweight mode when resume is not enabled
	if !options.Resume && shouldUseLightweight(fileInfo.Size) {
		d.logInfo("using_lightweight_mode", "Using lightweight mode for small file", map[string]interface{}{
			"size": fileInfo.Size,
		})
		return d.performLightweightDownload(ctx, url, destination, options)
	}

	// Check if we should use zero-copy mode for large files (platform-aware)
	if !options.Resume && d.platformInfo.Optimizations.UseZeroCopy && ShouldUseZeroCopyPlatform(fileInfo.Size) {
		d.logInfo("using_zerocopy_mode", "Using zero-copy mode for large file", map[string]interface{}{
			"size":     fileInfo.Size,
			"platform": GetPlatformString(),
		})
		return d.performZeroCopyDownload(ctx, url, destination, options)
	}

	// Check disk space with actual file size
	if fileInfo.Size > 0 {
		if err := d.checkDiskSpace(destination, uint64(fileInfo.Size)); err != nil {
			return nil, d.wrapDownloadError(err, url, destination, 0, fileInfo.Size)
		}
	}

	// Handle resume logic
	if options.Resume {
		return d.performResumeDownload(ctx, url, destination, options, fileInfo)
	}

	// Determine download strategy based on conditions
	shouldUseConcurrent := options.MaxConcurrency > 1 &&
		fileInfo.SupportsRanges &&
		fileInfo.Size > 10*1024*1024 && // 10MB threshold
		!options.Resume

	if shouldUseConcurrent {
		// Concurrent download not yet implemented, fall back to single download
		d.logInfo(
			"concurrent_fallback",
			"Concurrent download not implemented, using single download",
			nil,
		)
	}

	// Perform single download with resume support
	return d.performSingleDownload(ctx, url, destination, options, fileInfo)
}

// performResumeDownload handles resume download logic with Range headers.
func (d *Downloader) checkExistingFileForResume(destination string, fileInfo *types.FileInfo, stats *types.DownloadStats) (int64, error) {
	fileInfoStat, err := os.Stat(destination)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil // No existing file
		}
		return 0, errors.WrapErrorWithURL(err, errors.CodePermissionDenied, "Failed to check existing file for resume", stats.URL)
	}

	// Check if it's a directory, which is an error
	if fileInfoStat.IsDir() {
		return 0, errors.WrapErrorWithURL(
			stdErrors.New("destination is a directory"),
			errors.CodePermissionDenied,
			"Cannot download to a directory path",
			stats.URL,
		)
	}

	return fileInfoStat.Size(), nil
}

func (d *Downloader) isFileComplete(resumeOffset int64, fileInfo *types.FileInfo, stats *types.DownloadStats) bool {
	if fileInfo.Size > 0 && resumeOffset >= fileInfo.Size {
		stats.BytesDownloaded = resumeOffset
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)
		stats.Success = true
		return true
	}
	return false
}

func (d *Downloader) createResumeRequest(ctx context.Context, url string, resumeOffset int64, fileInfo *types.FileInfo) (*http.Request, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, errors.WrapErrorWithURL(
			err,
			errors.CodeInvalidURL,
			"Failed to create HTTP request for resume",
			url,
		)
	}

	// Set Range header to resume from where we left off
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeOffset))

	// Set If-Range header if we have ETag or Last-Modified (for validation)
	headers := http.Header(fileInfo.Headers)
	if etag := headers.Get("ETag"); etag != "" {
		req.Header.Set("If-Range", etag)
	} else if lastModified := headers.Get("Last-Modified"); lastModified != "" {
		req.Header.Set("If-Range", lastModified)
	}

	return req, nil
}

func (d *Downloader) handleResumeResponse(
	ctx context.Context,
	req *http.Request,
	destination string,
	options *types.DownloadOptions,
	stats *types.DownloadStats,
	resumeOffset int64,
	fileInfo *types.FileInfo,
) (*types.DownloadStats, error) {
	// Perform the HTTP request
	resp, err := d.client.Do(req)
	if err != nil {
		downloadErr := d.handleHTTPError(err, stats.URL)
		stats.Error = downloadErr
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)
		return stats, downloadErr
	}
	defer func() { _ = resp.Body.Close() }()

	// Handle different response status codes
	switch resp.StatusCode {
	case http.StatusPartialContent:
		return d.handlePartialContentResponse(ctx, resp, destination, options, stats, resumeOffset, fileInfo)
	case http.StatusOK:
		// Server returned full content, fall back to normal download
		stats.Resumed = false
		return d.performSingleDownload(ctx, stats.URL, destination, options, fileInfo)
	case http.StatusRequestedRangeNotSatisfiable:
		// Range not satisfiable, fall back to full download
		stats.Resumed = false
		return d.performSingleDownload(ctx, stats.URL, destination, options, fileInfo)
	default:
		// Some other error
		downloadErr := errors.FromHTTPStatus(resp.StatusCode, stats.URL)
		stats.Error = downloadErr
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)
		return stats, downloadErr
	}
}

func (d *Downloader) handlePartialContentResponse(
	ctx context.Context,
	resp *http.Response,
	destination string,
	options *types.DownloadOptions,
	stats *types.DownloadStats,
	resumeOffset int64,
	fileInfo *types.FileInfo,
) (*types.DownloadStats, error) {
	// Successfully resuming, open file in append mode
	// #nosec G304 -- destination validated by ValidateDestination() in public API Download functions
	file, err := os.OpenFile(destination, os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		downloadErr := errors.WrapErrorWithURL(
			err,
			errors.CodePermissionDenied,
			"Failed to open file for resume",
			stats.URL,
		)
		stats.Error = downloadErr
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)
		return stats, downloadErr
	}
	defer func() { _ = file.Close() }()

	// Start progress tracking if available
	if options.Progress != nil {
		options.Progress.Start(stats.Filename, fileInfo.Size)
	}

	// Download the remaining content
	bytesDownloaded, err := d.downloadContent(ctx, resp.Body, file, options, stats)
	stats.BytesDownloaded = resumeOffset + bytesDownloaded // Include already downloaded bytes
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	if err != nil {
		stats.Error = err
		stats.Success = false
		if options.Progress != nil {
			options.Progress.Error(stats.Filename, err)
		}
		return stats, err
	}

	// Calculate final statistics
	stats.Success = true
	if stats.Duration > 0 {
		stats.AverageSpeed = int64(float64(bytesDownloaded) / stats.Duration.Seconds())
	}

	// Notify progress completion
	if options.Progress != nil {
		options.Progress.Finish(stats.Filename, stats)
	}

	return stats, nil
}

func (d *Downloader) performResumeDownload(
	ctx context.Context,
	url, destination string,
	options *types.DownloadOptions,
	fileInfo *types.FileInfo,
) (*types.DownloadStats, error) {
	// Initialize stats
	stats := &types.DownloadStats{
		URL:       url,
		Filename:  destination,
		TotalSize: fileInfo.Size,
		StartTime: time.Now(),
		Resumed:   true,
	}

	// Check existing file and determine resume offset
	resumeOffset, err := d.checkExistingFileForResume(destination, fileInfo, stats)
	if err != nil {
		return stats, err
	}

	// If file is complete, return success
	if d.isFileComplete(resumeOffset, fileInfo, stats) {
		return stats, nil
	}

	// If no existing file or server doesn't support ranges, do normal download
	if resumeOffset == 0 || !fileInfo.SupportsRanges {
		return d.performSingleDownload(ctx, url, destination, options, fileInfo)
	}

	// Create resume request
	req, err := d.createResumeRequest(ctx, url, resumeOffset, fileInfo)
	if err != nil {
		stats.Error = err
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)
		return stats, err
	}

	// Set other headers
	d.setRequestHeaders(req, options)

	// Perform the HTTP request and handle response
	return d.handleResumeResponse(ctx, req, destination, options, stats, resumeOffset, fileInfo)
}

// performSimpleDownload performs a simple download without file info.
func (d *Downloader) performSimpleDownload(
	ctx context.Context,
	url, destination string,
	options *types.DownloadOptions,
) (*types.DownloadStats, error) {
	// #nosec G304 -- destination validated by ValidateDestination() in public API functions
	file, err := os.Create(destination)
	if err != nil {
		return nil, errors.WrapErrorWithURL(err, errors.CodePermissionDenied,
			"Failed to create destination file", url)
	}
	defer func() { _ = file.Close() }()

	stats, err := d.DownloadToWriter(ctx, url, file, options)
	if stats != nil {
		stats.Filename = destination
	}

	return stats, err
}

// performSingleDownload performs a single-threaded download with file info.
func (d *Downloader) performSingleDownload(
	ctx context.Context,
	url, destination string,
	options *types.DownloadOptions,
	fileInfo *types.FileInfo,
) (*types.DownloadStats, error) {
	// #nosec G304 -- destination validated by ValidateDestination() in public API functions
	file, err := os.Create(destination)
	if err != nil {
		return nil, errors.WrapErrorWithURL(err, errors.CodePermissionDenied,
			"Failed to create destination file", url)
	}
	defer func() { _ = file.Close() }()

	stats, err := d.DownloadToWriter(ctx, url, file, options)
	if stats != nil {
		stats.Filename = destination
	}

	return stats, err
}

// fallbackToSimpleDownload performs a simple download when HEAD request fails.
func (d *Downloader) fallbackToSimpleDownload(
	ctx context.Context,
	url, destination string,
	options *types.DownloadOptions,
	stats *types.DownloadStats,
) (*types.DownloadStats, error) {
	// Open file for writing
	// #nosec G304 -- destination validated by ValidateDestination() in public API functions
	file, err := os.Create(destination)
	if err != nil {
		downloadErr := errors.WrapErrorWithURL(err, errors.CodePermissionDenied,
			"Failed to create destination file", url)
		stats.Error = downloadErr
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)

		return stats, downloadErr
	}
	defer func() { _ = file.Close() }()

	// Use the existing DownloadToWriter method for simple download
	return d.DownloadToWriter(ctx, url, file, options)
}

// downloadWithResumeSupport handles traditional download with resume capabilities.
func (d *Downloader) downloadWithResumeSupport(
	ctx context.Context,
	url, destination string,
	options *types.DownloadOptions,
	stats *types.DownloadStats,
	fileInfo *types.FileInfo,
) (*types.DownloadStats, error) {
	// Check if we should attempt to resume
	var resumeOffset int64

	if options.Resume && fileInfo.SupportsRanges {
		// Try to load existing resume information
		existingInfo, err := d.resumeManager.Load(destination)
		if err == nil && existingInfo != nil {
			// Validate resume conditions
			if d.canResumeDownload(existingInfo, fileInfo, url) {
				resumeOffset = existingInfo.DownloadedBytes
				stats.Resumed = true
				d.logInfo("Resume", url, map[string]interface{}{
					"offset": resumeOffset,
				})
			} else {
				// Resume not valid, clean up old resume file
				_ = d.resumeManager.Delete(destination)
				d.logInfo("Resume invalid", url, nil)
			}
		}
	}

	// Open file for writing or appending
	var file *os.File
	var err error
	if resumeOffset > 0 {
		// Open for appending (resume)
		// #nosec G304 -- destination validated by ValidateDestination() in public API functions
		file, err = os.OpenFile(destination, os.O_WRONLY|os.O_APPEND, 0o600)
	} else {
		// Create new file with secure permissions (0600)
		// #nosec G304 -- destination validated by ValidateDestination() in public API functions
		file, err = os.OpenFile(destination, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o600)
	}

	if err != nil {
		downloadErr := errors.WrapErrorWithURL(err, errors.CodePermissionDenied,
			"Failed to create/open destination file", url)
		stats.Error = downloadErr
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)

		return stats, downloadErr
	}
	defer func() { _ = file.Close() }()

	// If resuming, perform resume download; otherwise, regular download
	if resumeOffset > 0 {
		return d.downloadWithResume(ctx, url, file, options, resumeOffset)
	}

	// Save initial resume info for new downloads
	if options.Resume && fileInfo.SupportsRanges {
		initialInfo := &resume.ResumeInfo{
			URL:             url,
			FilePath:        destination,
			DownloadedBytes: 0,
			TotalBytes:      fileInfo.Size,
			ETag:            d.getETagFromHeaders(fileInfo.Headers),
			LastModified:    fileInfo.LastModified,
			ContentLength:   fileInfo.Size,
			AcceptRanges:    fileInfo.SupportsRanges,
		}
		_ = d.resumeManager.Save(initialInfo)
	}

	// Use the existing DownloadToWriter method
	result, err := d.DownloadToWriter(ctx, url, file, options)

	// Clean up resume file on successful download
	if err == nil && options.Resume {
		_ = d.resumeManager.Delete(destination)
	}

	return result, err
}

// DownloadToWriter downloads a file from the given URL and writes it to the provided writer.
// It implements the types.Downloader interface.
func (d *Downloader) DownloadToWriter(
	ctx context.Context,
	url string,
	writer io.Writer,
	options *types.DownloadOptions,
) (*types.DownloadStats, error) {
	// Validate inputs
	if err := d.validateURL(url); err != nil {
		return nil, err
	}

	// Set default options if not provided
	if options == nil {
		options = &types.DownloadOptions{}
	}

	d.setDefaultOptions(options)

	// Initialize download stats
	stats := &types.DownloadStats{
		URL:       url,
		StartTime: time.Now(),
	}

	// Create HTTP request with context
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		downloadErr := errors.WrapErrorWithURL(err, errors.CodeInvalidURL,
			"Failed to create HTTP request", url)
		stats.Error = downloadErr
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)

		return stats, downloadErr
	}

	// Set request headers
	d.setRequestHeaders(req, options)

	// Get client from connection pool for better performance
	parsedURL, parseErr := parseURL(url)
	var client *http.Client
	if parseErr == nil && parsedURL != nil && d.connectionPool != nil {
		client = d.connectionPool.GetClient(parsedURL.Host, DefaultTimeout)
	} else {
		client = d.client
	}

	// Perform the HTTP request
	resp, err := client.Do(req)
	if err != nil {
		downloadErr := d.handleHTTPError(err, url)
		stats.Error = downloadErr
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)

		return stats, downloadErr
	}
	defer func() { _ = resp.Body.Close() }()

	// Check HTTP status code
	if resp.StatusCode != http.StatusOK {
		downloadErr := errors.FromHTTPStatus(resp.StatusCode, url)
		stats.Error = downloadErr
		stats.EndTime = time.Now()
		stats.Duration = stats.EndTime.Sub(stats.StartTime)

		return stats, downloadErr
	}

	// Get content length for progress tracking
	contentLength := resp.ContentLength
	if contentLength > 0 {
		stats.TotalSize = contentLength

		// Optimize download options based on actual content length
		optimizeOptionsForContentLength(options, contentLength)
	}

	// Create progress reader if callback is available
	var progressReader io.Reader = resp.Body
	if options.ProgressCallback != nil {
		progressReader = progress.NewProgressReader(
			resp.Body,
			contentLength,
			options.ProgressCallback,
		)
	}

	// Start progress tracking if available
	if options.Progress != nil {
		filename := d.extractFilename(url, resp)
		options.Progress.Start(filename, contentLength)
	}

	// Download the content
	bytesDownloaded, err := d.downloadContent(ctx, progressReader, writer, options, stats)
	stats.BytesDownloaded = bytesDownloaded
	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	if err != nil {
		stats.Error = err

		stats.Success = false
		if options.Progress != nil {
			options.Progress.Error(stats.Filename, err)
		}

		return stats, err
	}

	// Calculate final statistics
	stats.Success = true
	if stats.Duration > 0 {
		stats.AverageSpeed = int64(float64(bytesDownloaded) / stats.Duration.Seconds())
	}

	// Notify progress completion
	if options.Progress != nil {
		options.Progress.Finish(stats.Filename, stats)
	}

	return stats, nil
}

// GetFileInfo retrieves information about a file without downloading it.
// It implements the types.Downloader interface.
func (d *Downloader) GetFileInfo(ctx context.Context, url string) (*types.FileInfo, error) {
	// Validate URL
	if err := d.validateURL(url); err != nil {
		return nil, err
	}

	// Create HEAD request
	req, err := http.NewRequestWithContext(ctx, "HEAD", url, nil)
	if err != nil {
		return nil, errors.WrapErrorWithURL(err, errors.CodeInvalidURL,
			"Failed to create HTTP request", url)
	}

	// Set default User-Agent
	req.Header.Set("User-Agent", DefaultUserAgent)

	// Perform the request
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, d.handleHTTPError(err, url)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check status code
	if resp.StatusCode != http.StatusOK {
		return nil, errors.FromHTTPStatus(resp.StatusCode, url)
	}

	// Extract file information
	fileInfo := &types.FileInfo{
		URL:     url,
		Headers: resp.Header,
	}

	// Extract content length
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		if size, err := strconv.ParseInt(contentLength, 10, 64); err == nil {
			fileInfo.Size = size
		}
	}

	// Extract content type
	fileInfo.ContentType = resp.Header.Get("Content-Type")

	// Extract last modified
	if lastModified := resp.Header.Get("Last-Modified"); lastModified != "" {
		if t, err := http.ParseTime(lastModified); err == nil {
			fileInfo.LastModified = t
		}
	}

	// Extract filename
	fileInfo.Filename = d.extractFilename(url, resp)

	// Check if server supports range requests
	fileInfo.SupportsRanges = resp.Header.Get("Accept-Ranges") == "bytes"

	return fileInfo, nil
}

// validateURL validates that the provided URL is valid and supported.
func (d *Downloader) validateURL(rawURL string) error {
	if rawURL == "" {
		return errors.NewDownloadError(errors.CodeInvalidURL, "URL cannot be empty")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return errors.WrapError(err, errors.CodeInvalidURL, "Invalid URL format")
	}

	// Check if scheme is supported
	scheme := strings.ToLower(parsedURL.Scheme)
	if scheme != "http" && scheme != "https" {
		return errors.NewDownloadErrorWithDetails(
			errors.CodeInvalidURL,
			"Unsupported URL scheme",
			fmt.Sprintf("Only HTTP and HTTPS are supported, got: %s", scheme),
		)
	}

	return nil
}

// setDefaultOptions sets default values for download options.
func (d *Downloader) setDefaultOptions(options *types.DownloadOptions) {
	if options.ChunkSize <= 0 {
		// Use platform-optimized buffer size
		options.ChunkSize = int64(d.platformInfo.Optimizations.BufferSize)
		if options.ChunkSize <= 0 {
			options.ChunkSize = DefaultChunkSize
		}
	}

	if options.UserAgent == "" {
		options.UserAgent = DefaultUserAgent
	}

	if options.Timeout <= 0 {
		options.Timeout = DefaultTimeout
	}

	if options.Headers == nil {
		options.Headers = make(map[string]string)
	}

	// Set default MaxConcurrency based on platform
	// This will be further optimized when Content-Length is known
	if options.MaxConcurrency == 0 {
		options.MaxConcurrency = d.platformInfo.Optimizations.Concurrency
		if options.MaxConcurrency <= 0 {
			options.MaxConcurrency = 4 // Conservative fallback
		}
	}
}

// handleExistingFile checks if the destination file exists and handles it according to options.
func (d *Downloader) handleExistingFile(destination string, options *types.DownloadOptions) error {
	if _, err := os.Stat(destination); err == nil {
		// File exists
		if !options.OverwriteExisting {
			return errors.NewDownloadErrorWithDetails(errors.CodeFileExists,
				"File already exists", fmt.Sprintf("File exists at: %s", destination))
		}
	} else if !os.IsNotExist(err) {
		// Some other error occurred
		return errors.WrapError(err, errors.CodePermissionDenied, "Failed to check file existence")
	}

	return nil
}

// createParentDirs creates parent directories for the destination file if they don't exist.
func (d *Downloader) createParentDirs(destination string) error {
	dir := filepath.Dir(destination)
	if dir != "." && dir != "/" {
		return os.MkdirAll(dir, 0o750)
	}

	return nil
}

// setRequestHeaders sets HTTP headers for the request based on options.
func (d *Downloader) setRequestHeaders(req *http.Request, options *types.DownloadOptions) {
	// Set User-Agent
	req.Header.Set("User-Agent", options.UserAgent)

	// Set custom headers
	for key, value := range options.Headers {
		req.Header.Set(key, value)
	}
}

// handleHTTPError converts HTTP client errors to DownloadError.
func (d *Downloader) handleHTTPError(err error, rawURL string) *errors.DownloadError {
	if stdErrors.Is(err, context.Canceled) {
		return errors.WrapErrorWithURL(err, errors.CodeCancelled, "Download was cancelled", rawURL)
	}

	if stdErrors.Is(err, context.DeadlineExceeded) {
		return errors.WrapErrorWithURL(err, errors.CodeTimeout, "Download timed out", rawURL)
	}

	// Check if it's a network error
	urlErr := &url.Error{}
	if stdErrors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return errors.WrapErrorWithURL(
				err,
				errors.CodeTimeout,
				"Network timeout occurred",
				rawURL,
			)
		}

		if urlErr.Temporary() {
			return errors.WrapErrorWithURL(
				err,
				errors.CodeNetworkError,
				"Temporary network error",
				rawURL,
			)
		}
	}

	return errors.WrapErrorWithURL(err, errors.CodeNetworkError, "Network error occurred", rawURL)
}

// downloadContent downloads the content from the response body to the writer.
func (d *Downloader) downloadContent(
	ctx context.Context,
	src io.Reader,
	dst io.Writer,
	options *types.DownloadOptions,
	stats *types.DownloadStats,
) (int64, error) {
	buffer := make([]byte, options.ChunkSize)

	// Create rate limiter if max rate is specified
	var rateLimiter ratelimit.Limiter
	if options.MaxRate > 0 {
		rateLimiter = ratelimit.NewBandwidthLimiter(options.MaxRate)
	} else {
		rateLimiter = ratelimit.NewNullLimiter()
	}

	var totalBytes int64

	lastProgressUpdate := time.Now()
	progressUpdateInterval := time.Second // Update progress every second

	for {
		select {
		case <-ctx.Done():
			return totalBytes, errors.WrapError(
				ctx.Err(),
				errors.CodeCancelled,
				"Download cancelled",
			)
		default:
		}

		// Read chunk
		n, err := src.Read(buffer)
		if n > 0 {
			// Apply rate limiting before writing
			if rateLimiterErr := rateLimiter.Wait(ctx, n); rateLimiterErr != nil {
				return totalBytes, errors.WrapError(
					rateLimiterErr,
					errors.CodeCancelled,
					"Download cancelled during rate limiting",
				)
			}

			// Write chunk
			written, writeErr := dst.Write(buffer[:n])
			if writeErr != nil {
				return totalBytes, errors.WrapError(
					writeErr,
					errors.CodePermissionDenied,
					"Failed to write data",
				)
			}

			totalBytes += int64(written)

			// Update progress if enough time has passed
			now := time.Now()
			if options.Progress != nil && now.Sub(lastProgressUpdate) >= progressUpdateInterval {
				elapsed := now.Sub(stats.StartTime)

				var speed int64
				if elapsed > 0 {
					speed = int64(float64(totalBytes) / elapsed.Seconds())
				}

				options.Progress.Update(totalBytes, stats.TotalSize, speed)

				lastProgressUpdate = now
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}

			return totalBytes, errors.WrapError(err, errors.CodeNetworkError, "Failed to read data")
		}
	}

	return totalBytes, nil
}

// extractFilename extracts a filename from the URL or HTTP response headers.
func (d *Downloader) extractFilename(rawURL string, resp *http.Response) string {
	// Try to get filename from Content-Disposition header
	if contentDisposition := resp.Header.Get("Content-Disposition"); contentDisposition != "" {
		if filename := d.parseContentDisposition(contentDisposition); filename != "" {
			return filename
		}
	}

	// Fall back to extracting from URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return defaultFilename
	}

	// Use URL path handling, not OS-specific filepath
	// URLs always use forward slashes, regardless of OS
	urlPath := parsedURL.Path
	if urlPath == "" || urlPath == "/" {
		return defaultFilename
	}

	// Extract the last segment after the last forward slash
	segments := strings.Split(strings.Trim(urlPath, "/"), "/")
	if len(segments) == 0 {
		return defaultFilename
	}

	filename := segments[len(segments)-1]
	if filename == "" || filename == "." {
		return defaultFilename
	}

	return filename
}

// parseContentDisposition parses the Content-Disposition header to extract filename.
func (d *Downloader) parseContentDisposition(header string) string {
	// Simple parsing for filename parameter
	parts := strings.Split(header, ";")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "filename=") {
			filename := strings.TrimPrefix(part, "filename=")
			filename = strings.Trim(filename, `"`)

			return filename
		}
	}

	return ""
}

// downloadWithResume downloads a file with resume support using Range requests.
func (d *Downloader) downloadWithResume(
	ctx context.Context,
	url string,
	file *os.File,
	options *types.DownloadOptions,
	resumeOffset int64,
) (*types.DownloadStats, error) {
	startTime := time.Now()

	// Create HTTP request with Range header
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, errors.WrapErrorWithURL(err, errors.CodeInvalidURL,
			"Failed to create HTTP request", url)
	}

	// Set Range header to resume from offset
	req.Header.Set("Range", fmt.Sprintf("bytes=%d-", resumeOffset))
	d.setRequestHeaders(req, options)

	// Perform the request
	resp, err := d.client.Do(req)
	if err != nil {
		return nil, errors.WrapErrorWithURL(err, errors.CodeNetworkError,
			"Failed to perform resume request", url)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check for proper resume response (206 Partial Content) or full content (200 OK)
	if resp.StatusCode != http.StatusPartialContent && resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("server returned unexpected status: %d", resp.StatusCode)
	}

	// If server doesn't support range requests, it returns 200 OK
	// In this case, we need to skip the already downloaded bytes
	actualResumeOffset := int64(0)
	if resp.StatusCode == http.StatusPartialContent {
		actualResumeOffset = resumeOffset
	}

	// Initialize download statistics
	stats := &types.DownloadStats{
		URL:             url,
		Filename:        file.Name(),
		StartTime:       startTime,
		BytesDownloaded: actualResumeOffset,
		Resumed:         actualResumeOffset > 0,
	}

	// Get content length from response
	contentLength := resp.ContentLength
	if contentLength > 0 {
		if actualResumeOffset > 0 {
			stats.TotalSize = actualResumeOffset + contentLength
		} else {
			stats.TotalSize = contentLength
		}
	}

	// If server returned full content but we were resuming, skip already downloaded bytes
	if resp.StatusCode == http.StatusOK && resumeOffset > 0 {
		d.logInfo("Server doesn't support resume, discarding offset", url, map[string]interface{}{
			"offset": resumeOffset,
		})
		_, _ = io.CopyN(io.Discard, resp.Body, resumeOffset)
	}

	// Copy data to file with progress tracking
	buf := d.bufferPool.Get(DefaultChunkSize)
	defer d.bufferPool.Put(buf)

	var written int64
	var lastProgressUpdate time.Time
	progressInterval := 100 * time.Millisecond

	for {
		select {
		case <-ctx.Done():
			// Save resume info before canceling
			_ = d.saveResumeProgress(url, file.Name(), stats.BytesDownloaded, stats.TotalSize)
			stats.EndTime = time.Now()
			stats.Duration = stats.EndTime.Sub(startTime)
			return stats, ctx.Err()
		default:
		}

		n, err := resp.Body.Read(buf)
		if n > 0 {
			nw, werr := file.Write(buf[:n])
			if werr != nil {
				_ = d.saveResumeProgress(url, file.Name(), stats.BytesDownloaded, stats.TotalSize)
				return stats, errors.WrapErrorWithURL(werr, errors.CodePermissionDenied,
					"Failed to write to file", url)
			}
			written += int64(nw)
			stats.BytesDownloaded += int64(nw)

			// Update progress callback if set
			if options.ProgressCallback != nil && time.Since(lastProgressUpdate) > progressInterval {
				speed := d.calculateDownloadSpeed(written, time.Since(startTime))
				options.ProgressCallback(stats.BytesDownloaded, stats.TotalSize, speed)
				lastProgressUpdate = time.Now()
			}
		}

		if err != nil {
			if err == io.EOF {
				break
			}
			_ = d.saveResumeProgress(url, file.Name(), stats.BytesDownloaded, stats.TotalSize)
			return stats, errors.WrapErrorWithURL(err, errors.CodeNetworkError,
				"Failed to read response body", url)
		}
	}

	stats.EndTime = time.Now()
	stats.Duration = stats.EndTime.Sub(startTime)
	stats.AverageSpeed = d.calculateDownloadSpeed(written, stats.Duration)
	stats.Success = true

	// Clean up resume file on successful download
	_ = d.resumeManager.Delete(file.Name())

	return stats, nil
}

// performLightweightDownload performs a download using the lightweight mode
// optimized for small files with minimal overhead.
func (d *Downloader) performLightweightDownload(
	ctx context.Context,
	url, destination string,
	options *types.DownloadOptions,
) (*types.DownloadStats, error) {
	// Create or open destination file
	// #nosec G304 -- destination validated by ValidateDestination() in public API functions
	file, err := os.Create(destination)
	if err != nil {
		return nil, errors.WrapErrorWithURL(err, errors.CodePermissionDenied,
			"Failed to create destination file", url)
	}
	defer func() { _ = file.Close() }()

	startTime := time.Now()
	stats := &types.DownloadStats{
		URL:       url,
		Filename:  destination,
		StartTime: startTime,
	}

	// Use lightweight downloader with progress if callback is provided
	var downloaded int64
	userAgent := ""
	if options.UserAgent != "" {
		userAgent = options.UserAgent
	}

	if options.ProgressCallback != nil {
		downloaded, err = d.lightweight.DownloadWithProgressAndOptions(
			ctx, url, file,
			func(down, total int64) {
				// Calculate speed
				elapsed := time.Since(startTime).Seconds()
				speed := int64(0)
				if elapsed > 0 {
					speed = int64(float64(down) / elapsed)
				}
				options.ProgressCallback(down, total, speed)
			},
			userAgent,
		)
	} else {
		downloaded, err = d.lightweight.DownloadWithOptions(ctx, url, file, userAgent)
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

	d.logInfo("lightweight_download_complete", "Lightweight download completed successfully", map[string]interface{}{
		"url":             url,
		"destination":     destination,
		"bytesDownloaded": downloaded,
		"duration":        stats.Duration,
	})

	return stats, nil
}

// parseURL is a helper function to parse URL string
func parseURL(urlStr string) (*url.URL, error) {
	return url.Parse(urlStr)
}

// canResumeDownload checks if a download can be resumed based on resume info and current file info.
func (d *Downloader) canResumeDownload(resumeInfo *resume.ResumeInfo, fileInfo *types.FileInfo, url string) bool {
	// Basic validation using resume manager
	if !d.resumeManager.CanResume(resumeInfo) {
		return false
	}

	// Check URL matches
	if resumeInfo.URL != url {
		return false
	}

	// Check ETag if available (for cache validation)
	currentETag := d.getETagFromHeaders(fileInfo.Headers)
	if resumeInfo.ETag != "" && currentETag != "" && resumeInfo.ETag != currentETag {
		d.logInfo("ETag mismatch", url, map[string]interface{}{
			"stored":  resumeInfo.ETag,
			"current": currentETag,
		})
		return false
	}

	// Check Last-Modified if ETag not available
	if resumeInfo.ETag == "" && !resumeInfo.LastModified.IsZero() && !fileInfo.LastModified.IsZero() {
		if !resumeInfo.LastModified.Equal(fileInfo.LastModified) {
			d.logInfo("Last-Modified mismatch", url, nil)
			return false
		}
	}

	return true
}

// getETagFromHeaders extracts ETag from HTTP headers.
func (d *Downloader) getETagFromHeaders(headers map[string][]string) string {
	if headers == nil {
		return ""
	}

	if etag, ok := headers["Etag"]; ok && len(etag) > 0 {
		return etag[0]
	}

	if etag, ok := headers["ETag"]; ok && len(etag) > 0 {
		return etag[0]
	}

	return ""
}

// saveResumeProgress saves current download progress to resume file.
func (d *Downloader) saveResumeProgress(url, filePath string, downloaded, total int64) error {
	if d.resumeManager == nil {
		return nil
	}

	// Load existing or create new resume info
	resumeInfo, err := d.resumeManager.Load(filePath)
	if err != nil || resumeInfo == nil {
		resumeInfo = &resume.ResumeInfo{
			URL:          url,
			FilePath:     filePath,
			AcceptRanges: true,
		}
	}

	// Update progress
	resumeInfo.DownloadedBytes = downloaded
	resumeInfo.TotalBytes = total

	// Calculate and set checksum
	if err := d.resumeManager.CalculateAndSetChecksum(resumeInfo); err != nil {
		d.logInfo("Failed to calculate checksum", url, map[string]interface{}{
			"error": err.Error(),
		})
	}

	return d.resumeManager.Save(resumeInfo)
}

// calculateDownloadSpeed calculates download speed in bytes per second.
func (d *Downloader) calculateDownloadSpeed(bytes int64, duration time.Duration) int64 {
	if duration == 0 {
		return 0
	}
	return int64(float64(bytes) / duration.Seconds())
}
