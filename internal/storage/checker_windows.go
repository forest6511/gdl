//go:build windows
// +build windows

// Package storage provides disk space management and monitoring capabilities.
package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/forest6511/gdl/pkg/errors"
	"golang.org/x/sys/windows"
)

// SpaceInfo represents disk space information.
type SpaceInfo struct {
	TotalBytes     uint64  `json:"total_bytes"`
	AvailableBytes uint64  `json:"available_bytes"`
	UsedBytes      uint64  `json:"used_bytes"`
	FreeBytes      uint64  `json:"free_bytes"`
	UsagePercent   float64 `json:"usage_percent"`
	Path           string  `json:"path"`
}

// SpaceChecker provides functionality to check available disk space.
type SpaceChecker struct {
	mu               sync.RWMutex
	thresholdPercent float64
	callbacks        []SpaceCallback
	minFreeSpace     uint64            // Minimum required free space in bytes
	warningThreshold float64           // Warning threshold as percentage (0.0-1.0)
	tempDirs         []string          // List of temporary directories to monitor
	downloadDirs     []string          // List of download directories to monitor
	cleanupCallbacks []CleanupCallback // Callbacks for cleanup suggestions
}

// SpaceCallback is called when space information changes.
type SpaceCallback func(current, initial *SpaceInfo, bytesDownloaded uint64)

// CleanupCallback is called when cleanup suggestions are generated.
type CleanupCallback func(suggestions []CleanupSuggestion)

// Constants for space calculations.
const (
	// DefaultMinFreeSpace is the default minimum free space (1GB).
	DefaultMinFreeSpace = 1 * 1024 * 1024 * 1024

	// DefaultWarningThreshold is the default warning threshold (90%).
	DefaultWarningThreshold = 0.9

	// TempFilePrefix is the prefix used for temporary files.
	TempFilePrefix = "gdl_temp_"

	// PartialFileExtension is the extension used for partial downloads.
	PartialFileExtension = ".gdl.partial"
	// LargeFileThreshold defines what constitutes a large file (100MB).
	LargeFileThreshold = 100 * 1024 * 1024

	// OldFileAge defines what constitutes an old file (30 days).
	OldFileAge = 30 * 24 * time.Hour
)

// CleanupType represents the type of cleanup operation.
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
		return "unknown"
	}
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
		return "unknown"
	}
}

// NewSpaceChecker creates a new SpaceChecker with default settings.
func NewSpaceChecker() *SpaceChecker {
	return &SpaceChecker{
		thresholdPercent: DefaultWarningThreshold * 100, // Convert to percentage
		minFreeSpace:     DefaultMinFreeSpace,
		warningThreshold: DefaultWarningThreshold,
		tempDirs:         getDefaultTempDirs(),
		downloadDirs:     []string{},
		cleanupCallbacks: []CleanupCallback{},
	}
}

// New creates a new SpaceChecker with default settings.
func New() *SpaceChecker {
	return &SpaceChecker{
		thresholdPercent: 90.0, // 90% usage threshold
	}
}

// WithThreshold sets the space usage threshold percentage.
func (sc *SpaceChecker) WithThreshold(percent float64) *SpaceChecker {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.thresholdPercent = percent
	return sc
}

// WithMinFreeSpace sets the minimum required free space.
func (sc *SpaceChecker) WithMinFreeSpace(bytes uint64) *SpaceChecker {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.minFreeSpace = bytes
	return sc
}

// WithWarningThreshold sets the warning threshold (0.0-1.0).
func (sc *SpaceChecker) WithWarningThreshold(threshold float64) *SpaceChecker {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	if threshold >= 0.0 && threshold <= 1.0 {
		sc.warningThreshold = threshold
		sc.thresholdPercent = threshold * 100 // Keep both in sync
	}
	return sc
}

// WithTempDirs sets the temporary directories to monitor.
func (sc *SpaceChecker) WithTempDirs(dirs []string) *SpaceChecker {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.tempDirs = dirs
	return sc
}

// WithDownloadDirs sets the download directories to monitor.
func (sc *SpaceChecker) WithDownloadDirs(dirs []string) *SpaceChecker {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.downloadDirs = dirs
	return sc
}

// AddCleanupCallback adds a callback for cleanup suggestions.
func (sc *SpaceChecker) AddCleanupCallback(callback CleanupCallback) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.cleanupCallbacks = append(sc.cleanupCallbacks, callback)
}

// AddCallback adds a callback to be invoked when space information changes.
func (sc *SpaceChecker) AddCallback(callback SpaceCallback) {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.callbacks = append(sc.callbacks, callback)
}

// GetSpaceInfo retrieves disk space information for the given path on Windows.
func (sc *SpaceChecker) GetSpaceInfo(path string) (*SpaceInfo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, errors.WrapErrorWithURL(err, errors.CodeUnknown,
			"Failed to get absolute path", fmt.Sprintf("Path: %s", path))
	}

	// Ensure the path exists (to match Unix behavior)
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, errors.NewDownloadErrorWithDetails(errors.CodeFileNotFound,
			"Path does not exist", fmt.Sprintf("Path: %s", absPath))
	}

	// Get the root of the path (drive)
	root := filepath.VolumeName(absPath)
	if root == "" {
		root = absPath[:3] // Fallback to first 3 characters (e.g., "C:\")
	}

	var freeBytesAvailable, totalNumberOfBytes, totalNumberOfFreeBytes uint64

	rootPtr, err := windows.UTF16PtrFromString(root)
	if err != nil {
		return nil, errors.WrapErrorWithURL(err, errors.CodeUnknown,
			"Failed to convert path to UTF16", fmt.Sprintf("Path: %s", root))
	}

	err = windows.GetDiskFreeSpaceEx(
		rootPtr,
		&freeBytesAvailable,
		&totalNumberOfBytes,
		&totalNumberOfFreeBytes,
	)
	if err != nil {
		return nil, errors.WrapError(err, errors.CodePermissionDenied,
			"Failed to get disk space information")
	}

	usedBytes := totalNumberOfBytes - totalNumberOfFreeBytes
	usagePercent := float64(usedBytes) / float64(totalNumberOfBytes) * 100.0

	return &SpaceInfo{
		TotalBytes:     totalNumberOfBytes,
		AvailableBytes: freeBytesAvailable,
		UsedBytes:      usedBytes,
		FreeBytes:      totalNumberOfFreeBytes,
		UsagePercent:   usagePercent,
		Path:           absPath,
	}, nil
}

// CheckSpace verifies if there's enough space at the target path.
func (sc *SpaceChecker) CheckSpace(targetPath string, requiredBytes uint64) error {
	spaceInfo, err := sc.GetSpaceInfo(targetPath)
	if err != nil {
		return err
	}

	if spaceInfo.AvailableBytes < requiredBytes {
		return errors.NewDownloadErrorWithDetails(errors.CodeInsufficientSpace,
			"Insufficient disk space",
			fmt.Sprintf("Required: %d bytes, Available: %d bytes",
				requiredBytes, spaceInfo.AvailableBytes))
	}

	// Check if usage would exceed threshold
	projectedUsed := spaceInfo.UsedBytes + requiredBytes
	usagePercent := float64(projectedUsed) / float64(spaceInfo.TotalBytes) * 100

	sc.mu.RLock()
	threshold := sc.thresholdPercent
	sc.mu.RUnlock()

	if usagePercent > threshold {
		return errors.NewDownloadErrorWithDetails(errors.CodeInsufficientSpace,
			"Disk space usage would exceed threshold",
			fmt.Sprintf("Projected usage: %.1f%%, Threshold: %.1f%%",
				usagePercent, threshold))
	}

	return nil
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
			fmt.Sprintf("Required: %d bytes, Available: %d bytes",
				requiredBytes, spaceInfo.AvailableBytes))
	}

	// Check if free space would drop below minimum threshold
	sc.mu.RLock()
	minFreeSpace := sc.minFreeSpace
	sc.mu.RUnlock()

	spaceAfterDownload := spaceInfo.AvailableBytes - requiredBytes
	if spaceAfterDownload < minFreeSpace {
		return errors.NewDownloadErrorWithDetails(errors.CodeInsufficientSpace,
			"Download would leave insufficient free space",
			fmt.Sprintf("Space after download: %d bytes, Minimum required: %d bytes",
				spaceAfterDownload, minFreeSpace))
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
		checker:       sc,
		initialSpace:  initialSpace,
		targetPath:    path,
		stopCh:        make(chan struct{}),
		stopChan:      make(chan bool, 1), // For test compatibility
		checkInterval: 5 * time.Second,
	}

	return monitor, nil
}

// CreateTempFile creates a temporary file for download operations.
func (sc *SpaceChecker) CreateTempFile(dir, pattern string) (*os.File, error) {
	if dir == "" {
		dir = os.TempDir()
	}

	// Check if parent directory exists before creating subdirectories
	parentDir := filepath.Dir(dir)
	if parentDir != "." && parentDir != "/" {
		if _, err := os.Stat(parentDir); os.IsNotExist(err) {
			return nil, errors.WrapError(err, errors.CodeFileNotFound,
				"Parent directory does not exist")
		}
	}

	// Ensure directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, errors.WrapError(err, errors.CodePermissionDenied,
			"Failed to create temporary directory")
	}

	// Check space before creating temp file
	spaceInfo, err := sc.GetSpaceInfo(dir)
	if err != nil {
		return nil, err
	}

	sc.mu.RLock()
	minFreeSpace := sc.minFreeSpace
	sc.mu.RUnlock()

	if spaceInfo.AvailableBytes < minFreeSpace {
		return nil, errors.NewDownloadErrorWithDetails(errors.CodeInsufficientSpace,
			"Insufficient space for temporary file",
			fmt.Sprintf("Available: %d bytes, Required minimum: %d bytes",
				spaceInfo.AvailableBytes, minFreeSpace))
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

	sc.mu.RLock()
	tempDirs := make([]string, len(sc.tempDirs))
	copy(tempDirs, sc.tempDirs)
	sc.mu.RUnlock()

	for _, tempDir := range tempDirs {
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
		filePath := filepath.Join(dir, fileName)
		if !isTemporaryFile(fileName, filePath) {
			continue
		}

		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Remove old temporary files (older than 1 hour)
		if time.Since(info.ModTime()) > time.Hour {
			if err := os.Remove(filePath); err == nil {
				totalFreed += uint64(info.Size())
			}
		}
	}

	return totalFreed, nil
}

// GenerateCleanupSuggestions analyzes directories and generates cleanup recommendations.
func (sc *SpaceChecker) GenerateCleanupSuggestions(paths []string) ([]CleanupSuggestion, error) {
	var suggestions []CleanupSuggestion

	// If no paths provided, use configured directories
	if len(paths) == 0 {
		sc.mu.RLock()
		paths = append(paths, sc.tempDirs...)
		paths = append(paths, sc.downloadDirs...)
		sc.mu.RUnlock()
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
	sc.mu.RLock()
	callbacks := make([]CleanupCallback, len(sc.cleanupCallbacks))
	copy(callbacks, sc.cleanupCallbacks)
	sc.mu.RUnlock()

	for _, callback := range callbacks {
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
func (sc *SpaceChecker) ExecuteCleanup(suggestions []CleanupSuggestion, safeOnly bool) (uint64, error) {
	var totalFreed uint64
	for _, suggestion := range suggestions {
		if safeOnly && !suggestion.Safe {
			continue // Skip unsafe cleanups if safeOnly is true
		}

		err := sc.performCleanup(suggestion)
		if err != nil {
			continue
		}
		totalFreed += suggestion.Size
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
		return os.Remove(suggestion.Path)
	case CleanupDuplicateFiles:
		return os.Remove(suggestion.Path)
	default:
		return fmt.Errorf("unknown cleanup type: %v", suggestion.Type)
	}
}

// SpaceMonitor monitors disk space changes during download operations.
type SpaceMonitor struct {
	checker       *SpaceChecker
	initialSpace  *SpaceInfo
	targetPath    string
	callbacks     []SpaceCallback
	mu            sync.RWMutex
	stopCh        chan struct{}
	stopChan      chan bool // For compatibility with tests
	stopped       bool
	checkInterval time.Duration
}

// NewSpaceMonitor creates a new space monitor.
func NewSpaceMonitor(targetPath string) (*SpaceMonitor, error) {
	checker := New()
	initialSpace, err := checker.GetSpaceInfo(targetPath)
	if err != nil {
		return nil, err
	}

	return &SpaceMonitor{
		checker:       checker,
		initialSpace:  initialSpace,
		targetPath:    targetPath,
		stopCh:        make(chan struct{}),
		stopChan:      make(chan bool, 1), // For test compatibility
		checkInterval: 5 * time.Second,
	}, nil
}

// AddCallback adds a callback to be invoked when space changes are detected.
func (sm *SpaceMonitor) AddCallback(callback SpaceCallback) {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	sm.callbacks = append(sm.callbacks, callback)
}

// Start begins monitoring disk space changes.
func (sm *SpaceMonitor) Start() {
	ticker := time.NewTicker(sm.checkInterval)
	go sm.monitorLoop(0, ticker)
}

// Stop stops the space monitoring.
func (sm *SpaceMonitor) Stop() {
	sm.mu.Lock()
	defer sm.mu.Unlock()
	if !sm.stopped {
		sm.stopped = true
		close(sm.stopCh)
	}
}

// GetInitialSpace returns the initial space information.
func (sm *SpaceMonitor) GetInitialSpace() *SpaceInfo {
	return sm.initialSpace
}

// GetCurrentSpace returns current space information.
func (sm *SpaceMonitor) GetCurrentSpace() (*SpaceInfo, error) {
	return sm.checker.GetSpaceInfo(sm.targetPath)
}

// monitorLoop runs the monitoring loop.
func (sm *SpaceMonitor) monitorLoop(bytesDownloaded uint64, ticker *time.Ticker) {
	defer ticker.Stop()

	for {
		select {
		case <-sm.stopCh:
			return
		case <-ticker.C:
			sm.checkSpace()
		}
	}
}

// checkSpace checks current space and notifies callbacks if needed.
func (sm *SpaceMonitor) checkSpace() {
	currentSpace, err := sm.checker.GetSpaceInfo(sm.targetPath)
	if err != nil {
		return
	}

	// Calculate bytes downloaded based on space difference
	bytesDownloaded := uint64(0)
	if sm.initialSpace.AvailableBytes > currentSpace.AvailableBytes {
		bytesDownloaded = sm.initialSpace.AvailableBytes - currentSpace.AvailableBytes
	}

	sm.mu.RLock()
	callbacks := make([]SpaceCallback, len(sm.callbacks))
	copy(callbacks, sm.callbacks)
	sm.mu.RUnlock()

	// Notify callbacks
	for _, callback := range callbacks {
		callback(currentSpace, sm.initialSpace, bytesDownloaded)
	}
}

// ValidateWritePermissions checks if we can write to the target directory.
func ValidateWritePermissions(targetPath string) error {
	dir := filepath.Dir(targetPath)

	// Check if directory exists
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return errors.WrapErrorWithURL(err, errors.CodeFileNotFound,
			"Target directory does not exist", fmt.Sprintf("Directory: %s", dir))
	}

	// Try creating a temporary file to test write permissions
	tempFile, err := os.CreateTemp(dir, "gdl_write_test_")
	if err != nil {
		return errors.WrapError(err, errors.CodePermissionDenied,
			"Cannot write to target directory")
	}

	// Clean up
	tempFile.Close()
	os.Remove(tempFile.Name())

	return nil
}

// CreateDirectoryIfNotExists creates the directory if it doesn't exist.
func CreateDirectoryIfNotExists(path string) error {
	dir := filepath.Dir(path)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return errors.WrapError(err, errors.CodePermissionDenied,
				"Failed to create directory")
		}
	}
	return nil
}

// getDefaultTempDirs returns the default temporary directories to monitor on Windows.
func getDefaultTempDirs() []string {
	tempDir := os.TempDir()
	if tempDir == "" {
		tempDir = "C:\\Windows\\Temp"
	}
	return []string{tempDir}
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

// isDirEmpty checks if a directory is empty.
func isDirEmpty(path string) (bool, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return false, err
	}

	return len(entries) == 0, nil
}
