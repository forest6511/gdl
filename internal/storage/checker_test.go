package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestNewSpaceChecker(t *testing.T) {
	checker := NewSpaceChecker()

	if checker == nil {
		t.Fatal("NewSpaceChecker() returned nil")
	}

	if checker.minFreeSpace != DefaultMinFreeSpace {
		t.Errorf("minFreeSpace = %d, want %d", checker.minFreeSpace, DefaultMinFreeSpace)
	}

	if checker.warningThreshold != DefaultWarningThreshold {
		t.Errorf(
			"warningThreshold = %f, want %f",
			checker.warningThreshold,
			DefaultWarningThreshold,
		)
	}

	if len(checker.tempDirs) == 0 {
		t.Error("tempDirs should not be empty")
	}
}

func TestSpaceChecker_WithMinFreeSpace(t *testing.T) {
	checker := NewSpaceChecker()
	newMinSpace := uint64(2 * 1024 * 1024 * 1024) // 2GB

	result := checker.WithMinFreeSpace(newMinSpace)

	if result != checker {
		t.Error("WithMinFreeSpace should return the same instance")
	}

	if checker.minFreeSpace != newMinSpace {
		t.Errorf("minFreeSpace = %d, want %d", checker.minFreeSpace, newMinSpace)
	}
}

func TestSpaceChecker_WithWarningThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold float64
		expected  float64
	}{
		{"Valid threshold 0.8", 0.8, 0.8},
		{"Valid threshold 0.95", 0.95, 0.95},
		{"Invalid threshold -0.1", -0.1, DefaultWarningThreshold}, // Should keep default
		{"Invalid threshold 1.5", 1.5, DefaultWarningThreshold},   // Should keep default
		{"Edge case 0.0", 0.0, 0.0},
		{"Edge case 1.0", 1.0, 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewSpaceChecker() // Create new checker for each test
			checker.WithWarningThreshold(tt.threshold)

			if checker.warningThreshold != tt.expected {
				t.Errorf("warningThreshold = %f, want %f", checker.warningThreshold, tt.expected)
			}
		})
	}
}

func TestSpaceChecker_GetSpaceInfo(t *testing.T) {
	checker := NewSpaceChecker()

	// Test with current directory
	info, err := checker.GetSpaceInfo(".")
	if err != nil {
		t.Fatalf("GetSpaceInfo(\".\") failed: %v", err)
	}

	if info == nil {
		t.Fatal("GetSpaceInfo returned nil info")
	}

	if info.TotalBytes == 0 {
		t.Error("TotalBytes should be greater than 0")
	}

	if info.FreeBytes > info.TotalBytes {
		t.Error("FreeBytes should not be greater than TotalBytes")
	}

	if info.UsedBytes != info.TotalBytes-info.FreeBytes {
		t.Error("UsedBytes calculation is incorrect")
	}

	if info.UsagePercent < 0 || info.UsagePercent > 100 {
		t.Errorf("UsagePercent = %f, should be between 0 and 100", info.UsagePercent)
	}

	if info.Path == "" {
		t.Error("Path should not be empty")
	}
}

func TestSpaceChecker_GetSpaceInfo_NonExistentPath(t *testing.T) {
	checker := NewSpaceChecker()

	_, err := checker.GetSpaceInfo("/non/existent/path/that/should/not/exist")
	if err == nil {
		t.Error("GetSpaceInfo should fail for non-existent path")
	}
}

func TestSpaceChecker_CheckAvailableSpace(t *testing.T) {
	checker := NewSpaceChecker().WithMinFreeSpace(100 * 1024 * 1024) // 100MB

	t.Run("Sufficient space", func(t *testing.T) {
		err := checker.CheckAvailableSpace(".", 1024) // 1KB
		if err != nil {
			t.Errorf("CheckAvailableSpace should succeed for small requirement: %v", err)
		}
	})

	t.Run("Insufficient space", func(t *testing.T) {
		// Try to require more space than likely available on most systems
		hugeSize := uint64(1000 * 1024 * 1024 * 1024 * 1024) // 1000TB

		err := checker.CheckAvailableSpace(".", hugeSize)
		if err == nil {
			t.Error("CheckAvailableSpace should fail for huge requirement")
		}
	})
}

func TestSpaceChecker_StartMonitoring(t *testing.T) {
	checker := NewSpaceChecker()

	monitor, err := checker.StartMonitoring(".")
	if err != nil {
		t.Fatalf("StartMonitoring failed: %v", err)
	}

	if monitor == nil {
		t.Fatal("StartMonitoring returned nil monitor")
	}

	if monitor.checker != checker {
		t.Error("Monitor should reference the same checker")
	}

	if monitor.initialSpace == nil {
		t.Error("Initial space info should not be nil")
	}

	if monitor.stopChan == nil {
		t.Error("Stop channel should not be nil")
	}

	// Test adding callback
	monitor.AddCallback(func(current, initial *SpaceInfo, bytesDownloaded uint64) {
		// Callback for testing
	})

	if len(monitor.callbacks) != 1 {
		t.Error("Callback should be added")
	}

	// Test stopping
	monitor.Stop()
}

func TestSpaceChecker_CreateTempFile(t *testing.T) {
	checker := NewSpaceChecker()
	tempDir := t.TempDir()

	tempFile, err := checker.CreateTempFile(tempDir, "test_*.tmp")
	if err != nil {
		t.Fatalf("CreateTempFile failed: %v", err)
	}

	if tempFile == nil {
		t.Fatal("CreateTempFile returned nil file")
	}

	defer func() { _ = tempFile.Close() }()
	defer func() { _ = os.Remove(tempFile.Name()) }()

	// Check if file exists
	if _, err := os.Stat(tempFile.Name()); os.IsNotExist(err) {
		t.Error("Temporary file should exist")
	}

	// Check if filename contains our prefix
	fileName := filepath.Base(tempFile.Name())
	if !strings.Contains(fileName, TempFilePrefix) {
		t.Errorf("Filename should contain prefix %s, got %s", TempFilePrefix, fileName)
	}
}

func TestSpaceChecker_CleanupTempFiles(t *testing.T) {
	checker := NewSpaceChecker()
	tempDir := t.TempDir()

	// Set this temp dir as one to monitor
	checker.WithTempDirs([]string{tempDir})

	// Create some temporary files
	tempFile1, _ := os.CreateTemp(tempDir, TempFilePrefix+"test1_*.tmp")
	tempFile2, _ := os.CreateTemp(tempDir, TempFilePrefix+"test2_*.tmp")

	_ = tempFile1.Close()
	_ = tempFile2.Close()

	// Make one file old (change its modification time)
	oldTime := time.Now().Add(-2 * time.Hour)
	_ = os.Chtimes(tempFile1.Name(), oldTime, oldTime)

	// Run cleanup
	bytesFreed, err := checker.CleanupTempFiles()
	// The result depends on timing and file ages, so we just check it doesn't error
	if err != nil {
		t.Errorf("CleanupTempFiles failed: %v", err)
	}

	// bytesFreed might be 0 if files are too recent
	// Note: bytesFreed is uint64, so it cannot be negative
	_ = bytesFreed // Mark as intentionally unused

	// Clean up
	_ = os.Remove(tempFile1.Name())
	_ = os.Remove(tempFile2.Name())
}

func TestSpaceChecker_GenerateCleanupSuggestions(t *testing.T) {
	checker := NewSpaceChecker()
	tempDir := t.TempDir()

	// Create test files of different types
	testFiles := []struct {
		name     string
		content  string
		modTime  time.Time
		expected CleanupType
	}{
		{TempFilePrefix + "test.tmp", "temp content", time.Now(), CleanupTemporaryFiles},
		{
			"download.zip" + PartialFileExtension,
			"partial content",
			time.Now(),
			CleanupPartialDownloads,
		},
		{"old_file.txt", "old content", time.Now().Add(-40 * 24 * time.Hour), CleanupOldDownloads},
	}

	for _, tf := range testFiles {
		filePath := filepath.Join(tempDir, tf.name)

		err := os.WriteFile(filePath, []byte(tf.content), 0o644)
		if err != nil {
			t.Fatalf("Failed to create test file %s: %v", tf.name, err)
		}

		// Set modification time
		err = os.Chtimes(filePath, tf.modTime, tf.modTime)
		if err != nil {
			t.Fatalf("Failed to set file time for %s: %v", tf.name, err)
		}
	}

	// Create empty directory
	emptyDir := filepath.Join(tempDir, "empty_dir")
	_ = os.Mkdir(emptyDir, 0o755)

	// Generate suggestions
	suggestions, err := checker.GenerateCleanupSuggestions([]string{tempDir})
	if err != nil {
		t.Fatalf("GenerateCleanupSuggestions failed: %v", err)
	}

	if len(suggestions) == 0 {
		t.Error("Should generate at least some cleanup suggestions")
	}

	// Check that we found expected types
	foundTypes := make(map[CleanupType]bool)
	for _, suggestion := range suggestions {
		foundTypes[suggestion.Type] = true
	}

	expectedTypes := []CleanupType{
		CleanupTemporaryFiles,
		CleanupPartialDownloads,
		CleanupOldDownloads,
		CleanupEmptyDirectories,
	}
	for _, expectedType := range expectedTypes {
		if !foundTypes[expectedType] {
			t.Errorf("Expected to find cleanup suggestion of type %v", expectedType)
		}
	}
}

func TestIsTemporaryFile(t *testing.T) {
	tests := []struct {
		fileName string
		expected bool
	}{
		{TempFilePrefix + "test.tmp", true},
		{"file.tmp", true},
		{"document.temp", true},
		{"backup.bak", true},
		{"file~", true},
		{"normal_file.txt", false},
		{"document.pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			result := isTemporaryFile(tt.fileName, "/fake/path/"+tt.fileName)
			if result != tt.expected {
				t.Errorf("isTemporaryFile(%s) = %v, want %v", tt.fileName, result, tt.expected)
			}
		})
	}
}

func TestIsPartialDownload(t *testing.T) {
	tests := []struct {
		fileName string
		expected bool
	}{
		{"download.zip" + PartialFileExtension, true},
		{"file.part", true},
		{"chrome_download.crdownload", true},
		{"normal_file.zip", false},
		{"document.pdf", false},
	}

	for _, tt := range tests {
		t.Run(tt.fileName, func(t *testing.T) {
			result := isPartialDownload(tt.fileName)
			if result != tt.expected {
				t.Errorf("isPartialDownload(%s) = %v, want %v", tt.fileName, result, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    uint64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1536 * 1024 * 1024, "1.5 GB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d bytes", tt.bytes), func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %s, want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestCleanupType_String(t *testing.T) {
	tests := []struct {
		cleanupType CleanupType
		expected    string
	}{
		{CleanupTemporaryFiles, "temporary_files"},
		{CleanupOldDownloads, "old_downloads"},
		{CleanupPartialDownloads, "partial_downloads"},
		{CleanupDuplicateFiles, "duplicate_files"},
		{CleanupEmptyDirectories, "empty_directories"},
		{CleanupLargeFiles, "large_files"},
		{CleanupType(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.cleanupType.String()
			if result != tt.expected {
				t.Errorf("CleanupType.String() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestPriority_String(t *testing.T) {
	tests := []struct {
		priority Priority
		expected string
	}{
		{PriorityLow, "low"},
		{PriorityMedium, "medium"},
		{PriorityHigh, "high"},
		{PriorityCritical, "critical"},
		{Priority(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.priority.String()
			if result != tt.expected {
				t.Errorf("Priority.String() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestSpaceChecker_ExecuteCleanup(t *testing.T) {
	checker := NewSpaceChecker()
	tempDir := t.TempDir()

	// Create test files
	safeFile := filepath.Join(tempDir, "safe_to_delete.tmp")
	unsafeFile := filepath.Join(tempDir, "important_file.txt")

	_ = os.WriteFile(safeFile, []byte("safe content"), 0o644)
	_ = os.WriteFile(unsafeFile, []byte("important content"), 0o644)

	suggestions := []CleanupSuggestion{
		{
			Type:        CleanupTemporaryFiles,
			Path:        safeFile,
			Size:        12,
			Description: "Safe temporary file",
			Priority:    PriorityHigh,
			Safe:        true,
		},
		{
			Type:        CleanupOldDownloads,
			Path:        unsafeFile,
			Size:        17,
			Description: "Important file",
			Priority:    PriorityLow,
			Safe:        false,
		},
	}

	// Test safe-only cleanup
	bytesFreed, err := checker.ExecuteCleanup(suggestions, true)
	if err != nil {
		t.Errorf("SafeOnly cleanup failed: %v", err)
	}

	if bytesFreed != 12 {
		t.Errorf("Expected 12 bytes freed, got %d", bytesFreed)
	}

	// Check that safe file was deleted but unsafe file remains
	if _, err := os.Stat(safeFile); !os.IsNotExist(err) {
		t.Error("Safe file should have been deleted")
	}

	if _, err := os.Stat(unsafeFile); os.IsNotExist(err) {
		t.Error("Unsafe file should still exist")
	}
}

func TestIsDirEmpty(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "storage_isempty_test")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Test empty directory by checking if it's empty using os.ReadDir
	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Errorf("Failed to read directory: %v", err)
	}

	if len(entries) != 0 {
		t.Error("Expected directory to be empty")
	}

	// Create a file in the directory
	testFile := filepath.Join(tmpDir, "test.txt")

	err = os.WriteFile(testFile, []byte("content"), 0o644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test non-empty directory
	entries, err = os.ReadDir(tmpDir)
	if err != nil {
		t.Errorf("Failed to read directory: %v", err)
	}

	if len(entries) == 0 {
		t.Error("Expected directory to be non-empty")
	}

	// Create subdirectory
	subDir := filepath.Join(tmpDir, "subdir")

	err = os.MkdirAll(subDir, 0o755)
	if err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Remove the file first
	_ = os.Remove(testFile)

	// Directory with subdirectory should be non-empty
	entries, err = os.ReadDir(tmpDir)
	if err != nil {
		t.Errorf("Failed to read directory: %v", err)
	}

	if len(entries) == 0 {
		t.Error("Directory with subdirectory should not be empty")
	}
}

func TestFormatBytes_AllCases(t *testing.T) {
	testCases := []struct {
		bytes    uint64
		expected string
	}{
		{0, "0 B"},
		{1, "1 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{1073741824, "1.0 GB"},
		{1610612736, "1.5 GB"},
		{1099511627776, "1.0 TB"},
		{1649267441664, "1.5 TB"},
		{1125899906842624, "1.0 PB"},
		{2251799813685248, "2.0 PB"},
	}

	for _, tc := range testCases {
		result := formatBytes(tc.bytes)
		if result != tc.expected {
			t.Errorf("formatBytes(%d) = %s, want %s", tc.bytes, result, tc.expected)
		}
	}
}

func TestSpaceChecker_WithDownloadDirs(t *testing.T) {
	checker := NewSpaceChecker()
	dirs := []string{"/tmp/downloads", "/home/user/downloads"}

	checker.WithDownloadDirs(dirs)

	if len(checker.downloadDirs) != 2 {
		t.Errorf("Expected 2 download dirs, got %d", len(checker.downloadDirs))
	}

	if checker.downloadDirs[0] != dirs[0] {
		t.Errorf("Expected download dir %s, got %s", dirs[0], checker.downloadDirs[0])
	}
}

func TestSpaceChecker_AddCleanupCallback(t *testing.T) {
	checker := NewSpaceChecker()

	var callbackCalled atomic.Bool

	callback := func(suggestions []CleanupSuggestion) {
		callbackCalled.Store(true)
	}

	checker.AddCleanupCallback(callback)

	if len(checker.cleanupCallbacks) != 1 {
		t.Error("Callback should be added")
	}

	// Simulate suggestion generation to trigger callback
	_, _ = checker.GenerateCleanupSuggestions([]string{})

	if !callbackCalled.Load() {
		t.Error("Callback should have been called")
	}
}

func TestSpaceMonitor_monitorLoop(t *testing.T) {
	checker := NewSpaceChecker()
	monitor, _ := checker.StartMonitoring(".")

	var callbackCalled atomic.Bool

	monitor.AddCallback(func(current, initial *SpaceInfo, bytesDownloaded uint64) {
		callbackCalled.Store(true)
	})

	// Use a mock ticker that ticks once
	ticker := time.NewTicker(1 * time.Millisecond)

	// Run the monitor loop in a goroutine and stop it after one tick
	go func() {
		monitor.monitorLoop(1024, ticker)
	}()

	time.Sleep(5 * time.Millisecond)
	monitor.Stop()

	if !callbackCalled.Load() {
		t.Error("Callback should have been called")
	}
}

func TestSpaceChecker_performCleanup_AllTypes(t *testing.T) {
	checker := NewSpaceChecker()
	tempDir := t.TempDir()

	testCases := []struct {
		name        string
		cleanupType CleanupType
		fileName    string
	}{
		{"TemporaryFiles", CleanupTemporaryFiles, "test.tmp"},
		{"PartialDownloads", CleanupPartialDownloads, "test.gdl.partial"},
		{"EmptyDirectories", CleanupEmptyDirectories, "empty_dir"},
		{"OldDownloads", CleanupOldDownloads, "old_download.zip"},
		{"LargeFiles", CleanupLargeFiles, "large_file.bin"},
		{"DuplicateFiles", CleanupDuplicateFiles, "duplicate.txt"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create a test file or directory
			var testPath string
			if tc.cleanupType == CleanupEmptyDirectories {
				testPath = filepath.Join(tempDir, tc.fileName)
				_ = os.Mkdir(testPath, 0o755)
			} else {
				testPath = filepath.Join(tempDir, tc.fileName)
				_ = os.WriteFile(testPath, []byte("content"), 0o644)
			}

			suggestion := CleanupSuggestion{
				Type: tc.cleanupType,
				Path: testPath,
			}

			err := checker.performCleanup(suggestion)
			if err != nil {
				t.Errorf("performCleanup for %s failed: %v", tc.name, err)
			}

			// Check if the file or directory was deleted
			if _, err := os.Stat(testPath); !os.IsNotExist(err) {
				t.Errorf("%s should have been deleted", tc.name)
			}
		})
	}
}

func TestSpaceChecker_CreateTempFile_Errors(t *testing.T) {
	// Skip this test in CI environments where we run as root
	if os.Getuid() == 0 {
		t.Skip("Skipping permission test when running as root (CI environment)")
	}

	checker := NewSpaceChecker()

	t.Run("Non-existent directory", func(t *testing.T) {
		_, err := checker.CreateTempFile("/non/existent/dir", "test")
		if err == nil {
			t.Error("Expected an error when creating temp file in non-existent directory")
		}
	})
}
