// Package storage provides disk space management and monitoring capabilities.
//go:build !windows

package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/forest6511/gdl/pkg/errors"
)

// SpaceInfo represents disk space information.
type SpaceInfo struct {
	TotalBytes     uint64  `json:"total_bytes"`
	FreeBytes      uint64  `json:"free_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	AvailableBytes uint64  `json:"available_bytes"` // Available to non-privileged users
	UsagePercent   float64 `json:"usage_percent"`
	Path           string  `json:"path"`
}

// CleanupSuggestion represents a cleanup recommendation.
type CleanupSuggestion struct {
	Type        CleanupType `json:"type"`
	Path        string      `json:"path"`
	Size        uint64      `json:"size"`
	Description string      `json:"description"`
	Priority    Priority    `json:"priority"`
	Safe        bool        `json:"safe"` // Whether it's safe to delete automatically
}

// CleanupType represents the type of cleanup operation.
const (
	unknownValue = "unknown"
)

type CleanupType int

const (
	// CleanupTemporaryFiles represents temporary files that can be safely deleted.
	CleanupTemporaryFiles CleanupType = iota

	// CleanupOldDownloads represents old download files.
	CleanupOldDownloads

	// CleanupPartialDownloads represents incomplete/corrupted downloads.
	CleanupPartialDownloads

	// CleanupDuplicateFiles represents duplicate files.
	CleanupDuplicateFiles

	// CleanupEmptyDirectories represents empty directories.
	CleanupEmptyDirectories

	// CleanupLargeFiles represents unusually large files.
	CleanupLargeFiles
)

// Priority represents the cleanup priority.
type Priority int

const (
	// PriorityLow for low priority cleanup items.
	PriorityLow Priority = iota

	// PriorityMedium for medium priority cleanup items.
	PriorityMedium

	// PriorityHigh for high priority cleanup items.
	PriorityHigh

	// PriorityCritical for critical cleanup items.
	PriorityCritical
)

// String returns the string representation of CleanupType.
func (ct CleanupType) String() string {
	switch ct {
	case CleanupTemporaryFiles:
		return "temporary_files"
	case CleanupOldDownloads:
		return "old_downloads"
	case CleanupPartialDownloads:
		return "partial_downloads"
	case CleanupDuplicateFiles:
		return "duplicate_files"
	case CleanupEmptyDirectories:
		return "empty_directories"
	case CleanupLargeFiles:
		return "large_files"
	default:
		return unknownValue
	}
}

// String returns the string representation of Priority.
func (p Priority) String() string {
	switch p {
	case PriorityLow:
		return "low"
	case PriorityMedium:
		return "medium"
	case PriorityHigh:
		return "high"
	case PriorityCritical:
		return "critical"
	default:
		return unknownValue
	}
}

// SpaceChecker provides disk space monitoring and management functionality.
type SpaceChecker struct {
	minFreeSpace     uint64            // Minimum required free space in bytes
	warningThreshold float64           // Warning threshold as percentage (0.0-1.0)
	tempDirs         []string          // List of temporary directories to monitor
	downloadDirs     []string          // List of download directories to monitor
	cleanupCallbacks []CleanupCallback // Callbacks for cleanup suggestions
}

// CleanupCallback is called when cleanup suggestions are generated.
type CleanupCallback func(suggestions []CleanupSuggestion)

// SpaceMonitor tracks space usage during downloads.
type SpaceMonitor struct {
	checker      *SpaceChecker
	monitorPath  string
	initialSpace *SpaceInfo
	stopChan     chan bool
	callbacks    []SpaceCallback
}

// SpaceCallback is called during space monitoring.
type SpaceCallback func(current *SpaceInfo, initial *SpaceInfo, bytesDownloaded uint64)

// Constants for space calculations.
const (
	// DefaultMinFreeSpace is the default minimum free space (1GB).
	DefaultMinFreeSpace = 1 * 1024 * 1024 * 1024

	// DefaultWarningThreshold is the default warning threshold (90%).
	DefaultWarningThreshold = 0.9

	// LargeFileThreshold defines what constitutes a large file (100MB).
	LargeFileThreshold = 100 * 1024 * 1024

	// OldFileAge defines what constitutes an old file (30 days).
	OldFileAge = 30 * 24 * time.Hour

	// PartialFileExtension is the extension used for partial downloads.
	PartialFileExtension = ".gdl.partial"

	// TempFilePrefix is the prefix used for temporary files.
	TempFilePrefix = "gdl_temp_"
)

// NewSpaceChecker creates a new disk space checker.
func NewSpaceChecker() *SpaceChecker {
	return &SpaceChecker{
		minFreeSpace:     DefaultMinFreeSpace,
		warningThreshold: DefaultWarningThreshold,
		tempDirs:         getDefaultTempDirs(),
		downloadDirs:     []string{},
		cleanupCallbacks: []CleanupCallback{},
	}
}

// WithMinFreeSpace sets the minimum required free space.
func (sc *SpaceChecker) WithMinFreeSpace(bytes uint64) *SpaceChecker {
	sc.minFreeSpace = bytes
	return sc
}

// WithWarningThreshold sets the warning threshold (0.0-1.0).
func (sc *SpaceChecker) WithWarningThreshold(threshold float64) *SpaceChecker {
	if threshold >= 0.0 && threshold <= 1.0 {
		sc.warningThreshold = threshold
	}

	return sc
}

// WithTempDirs sets the temporary directories to monitor.
func (sc *SpaceChecker) WithTempDirs(dirs []string) *SpaceChecker {
	sc.tempDirs = dirs
	return sc
}

// WithDownloadDirs sets the download directories to monitor.
func (sc *SpaceChecker) WithDownloadDirs(dirs []string) *SpaceChecker {
	sc.downloadDirs = dirs
	return sc
}

// AddCleanupCallback adds a callback for cleanup suggestions.
func (sc *SpaceChecker) AddCleanupCallback(callback CleanupCallback) {
	sc.cleanupCallbacks = append(sc.cleanupCallbacks, callback)
}

// GetSpaceInfo retrieves disk space information for a given path.
func (sc *SpaceChecker) GetSpaceInfo(path string) (*SpaceInfo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.WrapError(err, errors.CodeFileNotFound,
			"Failed to resolve absolute path")
	}

	// Ensure the directory exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, errors.NewDownloadErrorWithDetails(errors.CodeFileNotFound,
			"Path does not exist", fmt.Sprintf("Path: %s", absPath))
	}

	var stat syscall.Statfs_t
	if err := syscall.Statfs(absPath, &stat); err != nil {
		return nil, errors.WrapError(err, errors.CodePermissionDenied,
			"Failed to get filesystem statistics")
	}

	// Calculate space information
	// #nosec G115 -- stat.Blocks, stat.Bfree, stat.Bavail are system-provided disk stats within uint64 range
	totalBytes := uint64(stat.Blocks) * uint64(stat.Bsize)
	// #nosec G115 -- stat.Bfree and stat.Bsize are system filesystem values, safe for uint64 arithmetic
	freeBytes := uint64(stat.Bfree) * uint64(stat.Bsize)
	// #nosec G115 -- stat.Bavail and stat.Bsize are system filesystem values, safe for uint64 arithmetic
	availableBytes := uint64(stat.Bavail) * uint64(stat.Bsize)
	usedBytes := totalBytes - freeBytes
	usagePercent := float64(usedBytes) / float64(totalBytes) * 100.0

	return &SpaceInfo{
		TotalBytes:     totalBytes,
		FreeBytes:      freeBytes,
		UsedBytes:      usedBytes,
		AvailableBytes: availableBytes,
		UsagePercent:   usagePercent,
		Path:           absPath,
	}, nil
}

// CheckAvailableSpace validates if there's enough space for a download.
func (sc *SpaceChecker) CheckAvailableSpace(path string, requiredBytes uint64) error {
	spaceInfo, err := sc.GetSpaceInfo(path)
	if err != nil {
		return err
	}

	// Check if there's enough available space
	if spaceInfo.AvailableBytes < requiredBytes {
		return errors.NewDownloadErrorWithDetails(errors.CodeInsufficientSpace,
			"Insufficient disk space",
			fmt.Sprintf("Required: %s, Available: %s",
				formatBytes(requiredBytes),
				formatBytes(spaceInfo.AvailableBytes)))
	}

	// Check if free space would drop below minimum threshold
	spaceAfterDownload := spaceInfo.AvailableBytes - requiredBytes
	if spaceAfterDownload < sc.minFreeSpace {
		return errors.NewDownloadErrorWithDetails(errors.CodeInsufficientSpace,
			"Download would leave insufficient free space",
			fmt.Sprintf("Space after download: %s, Minimum required: %s",
				formatBytes(spaceAfterDownload),
				formatBytes(sc.minFreeSpace)))
	}

	// Check if usage would exceed warning threshold
	newUsagePercent := float64(spaceInfo.UsedBytes+requiredBytes) / float64(spaceInfo.TotalBytes)
	if newUsagePercent > sc.warningThreshold {
		// This is a warning, not an error
		fmt.Printf("Warning: Download will increase disk usage to %.1f%%\n", newUsagePercent*100)
	}

	return nil
}

// StartMonitoring begins monitoring disk space during a download.
func (sc *SpaceChecker) StartMonitoring(path string) (*SpaceMonitor, error) {
	initialSpace, err := sc.GetSpaceInfo(path)
	if err != nil {
		return nil, err
	}

	monitor := &SpaceMonitor{
		checker:      sc,
		monitorPath:  path,
		initialSpace: initialSpace,
		stopChan:     make(chan bool, 1),
		callbacks:    []SpaceCallback{},
	}

	return monitor, nil
}

// AddCallback adds a space monitoring callback.
func (sm *SpaceMonitor) AddCallback(callback SpaceCallback) {
	sm.callbacks = append(sm.callbacks, callback)
}

// Monitor continuously monitors disk space usage.
func (sm *SpaceMonitor) Monitor(bytesDownloaded uint64) {
	go sm.monitorLoop(bytesDownloaded, time.NewTicker(5*time.Second))
}

func (sm *SpaceMonitor) monitorLoop(bytesDownloaded uint64, ticker *time.Ticker) {
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopChan:
			return
		case <-ticker.C:
			currentSpace, err := sm.checker.GetSpaceInfo(sm.monitorPath)
			if err != nil {
				continue // Skip this check if we can't get space info
			}

			// Call callbacks with current space information
			for _, callback := range sm.callbacks {
				callback(currentSpace, sm.initialSpace, bytesDownloaded)
			}

			// Check for critical space conditions
			if currentSpace.AvailableBytes < sm.checker.minFreeSpace {
				fmt.Printf("Critical: Available disk space is below minimum threshold: %s\n",
					formatBytes(currentSpace.AvailableBytes))
			}
		}
	}
}

// Stop stops the space monitoring.
func (sm *SpaceMonitor) Stop() {
	select {
	case sm.stopChan <- true:
	default:
	}
}

// GenerateCleanupSuggestions analyzes directories and generates cleanup recommendations.
func (sc *SpaceChecker) GenerateCleanupSuggestions(paths []string) ([]CleanupSuggestion, error) {
	var suggestions []CleanupSuggestion

	// If no paths provided, use configured directories
	if len(paths) == 0 {
		paths = append(paths, sc.tempDirs...)
		paths = append(paths, sc.downloadDirs...)
	}

	for _, path := range paths {
		pathSuggestions, err := sc.analyzePath(path)
		if err != nil {
			continue // Skip paths we can't analyze
		}

		suggestions = append(suggestions, pathSuggestions...)
	}

	// Sort suggestions by priority and size
	sort.Slice(suggestions, func(i, j int) bool {
		if suggestions[i].Priority != suggestions[j].Priority {
			return suggestions[i].Priority > suggestions[j].Priority // Higher priority first
		}

		return suggestions[i].Size > suggestions[j].Size // Larger files first within same priority
	})

	// Notify callbacks
	for _, callback := range sc.cleanupCallbacks {
		callback(suggestions)
	}

	return suggestions, nil
}

// analyzePath analyzes a single path for cleanup opportunities.
func (sc *SpaceChecker) analyzePath(path string) ([]CleanupSuggestion, error) {
	var suggestions []CleanupSuggestion

	err := filepath.Walk(path, func(filePath string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking even if we encounter errors
		}

		if info.IsDir() {
			// Check for empty directories
			if isEmpty, _ := isDirEmpty(filePath); isEmpty && filePath != path {
				suggestions = append(suggestions, CleanupSuggestion{
					Type:        CleanupEmptyDirectories,
					Path:        filePath,
					Size:        0,
					Description: "Empty directory",
					Priority:    PriorityLow,
					Safe:        true,
				})
			}

			return nil
		}

		// Analyze files
		// #nosec G115 -- info.Size() returns int64 which is safe to convert to uint64 for file sizes
		fileSize := uint64(info.Size())
		fileName := info.Name()

		// Check for temporary files
		if isTemporaryFile(fileName, filePath) {
			suggestions = append(suggestions, CleanupSuggestion{
				Type:        CleanupTemporaryFiles,
				Path:        filePath,
				Size:        fileSize,
				Description: "Temporary file",
				Priority:    PriorityHigh,
				Safe:        true,
			})
		}

		// Check for partial download files
		if isPartialDownload(fileName) {
			priority := PriorityMedium
			if time.Since(info.ModTime()) > 24*time.Hour {
				priority = PriorityHigh // Old partial files get higher priority
			}

			suggestions = append(suggestions, CleanupSuggestion{
				Type:        CleanupPartialDownloads,
				Path:        filePath,
				Size:        fileSize,
				Description: "Incomplete download file",
				Priority:    priority,
				Safe:        true,
			})
		}

		// Check for old files
		if time.Since(info.ModTime()) > OldFileAge {
			suggestions = append(suggestions, CleanupSuggestion{
				Type: CleanupOldDownloads,
				Path: filePath,
				Size: fileSize,
				Description: fmt.Sprintf(
					"Old file (%.0f days)",
					time.Since(info.ModTime()).Hours()/24,
				),
				Priority: PriorityLow,
				Safe:     false, // Old files might still be needed
			})
		}

		// Check for large files
		if fileSize > LargeFileThreshold {
			suggestions = append(suggestions, CleanupSuggestion{
				Type:        CleanupLargeFiles,
				Path:        filePath,
				Size:        fileSize,
				Description: fmt.Sprintf("Large file (%s)", formatBytes(fileSize)),
				Priority:    PriorityLow,
				Safe:        false, // Large files should be manually reviewed
			})
		}

		return nil
	})

	return suggestions, err
}

// ExecuteCleanup performs cleanup based on suggestions.
func (sc *SpaceChecker) ExecuteCleanup(
	suggestions []CleanupSuggestion,
	safeOnly bool,
) (uint64, error) {
	var (
		totalFreed uint64
		errorsList []error
	)

	for _, suggestion := range suggestions {
		if safeOnly && !suggestion.Safe {
			continue // Skip unsafe cleanups if safeOnly is true
		}

		err := sc.performCleanup(suggestion)
		if err != nil {
			errorsList = append(errorsList, err)
			continue
		}

		totalFreed += suggestion.Size
	}

	if len(errorsList) > 0 {
		return totalFreed, errors.NewDownloadError(errors.CodeStorageError,
			fmt.Sprintf("cleanup completed with %d errors", len(errorsList)))
	}

	return totalFreed, nil
}

// performCleanup executes a single cleanup operation.
func (sc *SpaceChecker) performCleanup(suggestion CleanupSuggestion) error {
	switch suggestion.Type {
	case CleanupTemporaryFiles, CleanupPartialDownloads:
		return os.Remove(suggestion.Path)

	case CleanupEmptyDirectories:
		return os.Remove(suggestion.Path)

	case CleanupOldDownloads, CleanupLargeFiles:
		// For these types, we might want to move to trash instead of delete
		return os.Remove(suggestion.Path)

	case CleanupDuplicateFiles:
		// This would require more sophisticated logic to identify duplicates
		return os.Remove(suggestion.Path)

	default:
		return errors.NewDownloadError(errors.CodeStorageError,
			fmt.Sprintf("unknown cleanup type: %v", suggestion.Type))
	}
}

// CreateTempFile creates a temporary file for download operations.
func (sc *SpaceChecker) CreateTempFile(dir, pattern string) (*os.File, error) {
	if dir == "" {
		dir = os.TempDir()
	}

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return nil, errors.WrapError(err, errors.CodePermissionDenied,
			"Failed to create temporary directory")
	}

	// Check space before creating temp file
	spaceInfo, err := sc.GetSpaceInfo(dir)
	if err != nil {
		return nil, err
	}

	if spaceInfo.AvailableBytes < sc.minFreeSpace {
		return nil, errors.NewDownloadErrorWithDetails(errors.CodeInsufficientSpace,
			"Insufficient space for temporary file",
			fmt.Sprintf("Available: %s, Required minimum: %s",
				formatBytes(spaceInfo.AvailableBytes),
				formatBytes(sc.minFreeSpace)))
	}

	// Create temporary file with our prefix
	fullPattern := TempFilePrefix + pattern

	tempFile, err := os.CreateTemp(dir, fullPattern)
	if err != nil {
		return nil, errors.WrapError(err, errors.CodePermissionDenied,
			"Failed to create temporary file")
	}

	return tempFile, nil
}

// CleanupTempFiles removes temporary files created by gdl.
func (sc *SpaceChecker) CleanupTempFiles() (uint64, error) {
	var totalFreed uint64

	for _, tempDir := range sc.tempDirs {
		freed, err := sc.cleanupTempFilesInDir(tempDir)
		if err != nil {
			continue // Continue with other directories
		}

		totalFreed += freed
	}

	return totalFreed, nil
}

// cleanupTempFilesInDir removes temporary files in a specific directory.
func (sc *SpaceChecker) cleanupTempFilesInDir(dir string) (uint64, error) {
	var totalFreed uint64

	entries, err := os.ReadDir(dir)
	if err != nil {
		return 0, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		fileName := entry.Name()
		if !isTemporaryFile(fileName, filepath.Join(dir, fileName)) {
			continue
		}

		filePath := filepath.Join(dir, fileName)

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Remove old temporary files (older than 1 hour)
		if time.Since(info.ModTime()) > time.Hour {
			if err := os.Remove(filePath); err == nil {
				// #nosec G115 -- info.Size() returns int64 which is safe to convert to uint64 for file sizes
				totalFreed += uint64(info.Size())
			}
		}
	}

	return totalFreed, nil
}

// Helper functions

// getDefaultTempDirs returns the default temporary directories to monitor.
func getDefaultTempDirs() []string {
	return []string{
		os.TempDir(),
		"/tmp",
		"/var/tmp",
	}
}

// isTemporaryFile checks if a file is a temporary file.
func isTemporaryFile(fileName, filePath string) bool {
	// Check for common temporary file patterns
	tempPatterns := []string{
		TempFilePrefix,
		".tmp",
		".temp",
		"~",
		".bak",
		".cache",
	}

	for _, pattern := range tempPatterns {
		if strings.Contains(fileName, pattern) {
			return true
		}
	}

	return false
}

// isPartialDownload checks if a file is a partial download.
func isPartialDownload(fileName string) bool {
	return strings.HasSuffix(fileName, PartialFileExtension) ||
		strings.HasSuffix(fileName, ".part") ||
		strings.HasSuffix(fileName, ".crdownload")
}

// isDirEmpty checks if a directory is empty.
func isDirEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	return len(entries) == 0, nil
}

// formatBytes formats byte counts in human-readable format.
func formatBytes(bytes uint64) string {
	if bytes < 1024 {
		return fmt.Sprintf("%d B", bytes)
	}

	units := []string{"KB", "MB", "GB", "TB", "PB"}
	value := float64(bytes)

	for _, unit := range units {
		value /= 1024
		if value < 1024 {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}

	return fmt.Sprintf("%.1f EB", value/1024)
}
