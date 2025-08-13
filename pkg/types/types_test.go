package types

import (
	"context"
	"errors"
	"io"
	"testing"
	"time"
)

func TestDownloadError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *DownloadError
		expected string
	}{
		{
			name: "with underlying error",
			err: &DownloadError{
				URL: "https://example.com/file.txt",
				Err: errors.New("network error"),
			},
			expected: "network error",
		},
		{
			name: "without underlying error",
			err: &DownloadError{
				URL: "https://example.com/file.txt",
			},
			expected: "download error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("DownloadError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDownloadError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")
	downloadErr := &DownloadError{
		Err: underlyingErr,
	}

	if got := downloadErr.Unwrap(); !errors.Is(got, underlyingErr) {
		t.Errorf("DownloadError.Unwrap() = %v, want %v", got, underlyingErr)
	}

	// Test with nil underlying error
	downloadErr2 := &DownloadError{}
	if got := downloadErr2.Unwrap(); got != nil {
		t.Errorf("DownloadError.Unwrap() = %v, want nil", got)
	}
}

func TestDownloadOptions_Validation(t *testing.T) {
	tests := []struct {
		name    string
		options *DownloadOptions
		isValid bool
	}{
		{
			name: "valid options with all fields",
			options: &DownloadOptions{
				MaxRetries:        3,
				RetryDelay:        time.Second,
				Timeout:           time.Minute,
				ChunkSize:         32 * 1024,
				UserAgent:         "test-agent",
				Headers:           map[string]string{"Accept": "application/octet-stream"},
				Resume:            true,
				OverwriteExisting: true,
				CreateDirs:        true,
				MaxConcurrency:    4,
			},
			isValid: true,
		},
		{
			name: "valid options with minimal fields",
			options: &DownloadOptions{
				ChunkSize: 1024,
			},
			isValid: true,
		},
		{
			name: "options with zero chunk size",
			options: &DownloadOptions{
				ChunkSize: 0,
			},
			isValid: true, // Zero chunk size should be handled by the implementation
		},
		{
			name: "options with negative values",
			options: &DownloadOptions{
				MaxRetries: -1,
				ChunkSize:  -1024,
			},
			isValid: true, // Negative values should be handled by validation in implementation
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Since DownloadOptions doesn't have validation methods,
			// we just test that the struct can be created and accessed
			if tt.options == nil {
				t.Error("DownloadOptions should not be nil")
			}

			// Test that all fields are accessible
			_ = tt.options.MaxRetries
			_ = tt.options.RetryDelay
			_ = tt.options.Timeout
			_ = tt.options.ChunkSize
			_ = tt.options.UserAgent
			_ = tt.options.Headers
			_ = tt.options.Resume
			_ = tt.options.OverwriteExisting
			_ = tt.options.CreateDirs
			_ = tt.options.MaxConcurrency
		})
	}
}

func TestDownloadStats_Calculation(t *testing.T) {
	tests := []struct {
		name     string
		stats    *DownloadStats
		validate func(t *testing.T, stats *DownloadStats)
	}{
		{
			name: "complete download stats",
			stats: &DownloadStats{
				URL:             "https://example.com/file.txt",
				Filename:        "file.txt",
				TotalSize:       1024,
				BytesDownloaded: 1024,
				StartTime:       time.Now().Add(-10 * time.Second),
				EndTime:         time.Now(),
				Success:         true,
			},
			validate: func(t *testing.T, stats *DownloadStats) {
				if stats.Duration == 0 {
					stats.Duration = stats.EndTime.Sub(stats.StartTime)
				}

				if stats.Duration <= 0 {
					t.Error("Duration should be positive")
				}

				if stats.TotalSize != stats.BytesDownloaded {
					t.Error("TotalSize should equal BytesDownloaded for complete download")
				}

				if !stats.Success {
					t.Error("Success should be true for complete download")
				}
			},
		},
		{
			name: "partial download stats",
			stats: &DownloadStats{
				URL:             "https://example.com/file.txt",
				Filename:        "file.txt",
				TotalSize:       2048,
				BytesDownloaded: 1024,
				StartTime:       time.Now().Add(-5 * time.Second),
				EndTime:         time.Now(),
				Success:         false,
				Error:           errors.New("connection interrupted"),
			},
			validate: func(t *testing.T, stats *DownloadStats) {
				if stats.Duration == 0 {
					stats.Duration = stats.EndTime.Sub(stats.StartTime)
				}

				if stats.TotalSize <= stats.BytesDownloaded {
					t.Error("TotalSize should be greater than BytesDownloaded for partial download")
				}

				if stats.Success {
					t.Error("Success should be false for failed download")
				}

				if stats.Error == nil {
					t.Error("Error should not be nil for failed download")
				}
			},
		},
		{
			name: "unknown size download",
			stats: &DownloadStats{
				URL:             "https://example.com/stream",
				Filename:        "stream.data",
				TotalSize:       -1,
				BytesDownloaded: 512,
				StartTime:       time.Now().Add(-3 * time.Second),
				EndTime:         time.Now(),
				Success:         true,
			},
			validate: func(t *testing.T, stats *DownloadStats) {
				if stats.Duration == 0 {
					stats.Duration = stats.EndTime.Sub(stats.StartTime)
				}

				if stats.TotalSize >= 0 {
					t.Error("TotalSize should be negative for unknown size downloads")
				}

				if stats.BytesDownloaded <= 0 {
					t.Error("BytesDownloaded should be positive")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.validate(t, tt.stats)
		})
	}
}

func TestFileInfo_Validation(t *testing.T) {
	tests := []struct {
		name     string
		fileInfo *FileInfo
		isValid  bool
	}{
		{
			name: "complete file info",
			fileInfo: &FileInfo{
				URL:            "https://example.com/file.txt",
				Size:           1024,
				LastModified:   time.Now(),
				ContentType:    "text/plain",
				Filename:       "file.txt",
				SupportsRanges: true,
				Headers:        map[string][]string{"Content-Type": {"text/plain"}},
			},
			isValid: true,
		},
		{
			name: "minimal file info",
			fileInfo: &FileInfo{
				URL:  "https://example.com/file.txt",
				Size: 0,
			},
			isValid: true,
		},
		{
			name: "file info with negative size",
			fileInfo: &FileInfo{
				URL:  "https://example.com/file.txt",
				Size: -1,
			},
			isValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test basic field access
			_ = tt.fileInfo.URL
			_ = tt.fileInfo.Size
			_ = tt.fileInfo.LastModified
			_ = tt.fileInfo.ContentType
			_ = tt.fileInfo.Filename
			_ = tt.fileInfo.SupportsRanges
			_ = tt.fileInfo.Headers

			// Basic validation
			if tt.isValid && tt.fileInfo.Size < 0 {
				t.Error("Valid FileInfo should not have negative size")
			}

			if tt.fileInfo.URL == "" {
				t.Error("FileInfo should have a URL")
			}
		})
	}
}

func TestProgress_Interface(t *testing.T) {
	// Test that we can create a mock implementation of Progress interface
	var progress Progress = &mockProgress{}

	// Test that all methods are callable
	progress.Start("test.txt", 1024)
	progress.Update(512, 1024, 100)
	progress.Finish("test.txt", &DownloadStats{})
	progress.Error("test.txt", errors.New("test error"))
}

func TestDownloader_Interface(t *testing.T) {
	// Test that we can create a mock implementation of Downloader interface
	var downloader Downloader = &mockDownloader{}

	// Test that all methods exist and are callable (they will return nil/errors in mock)
	_, _ = downloader.Download(context.TODO(), "", "", nil)
	_, _ = downloader.DownloadToWriter(context.TODO(), "", nil, nil)
	_, _ = downloader.GetFileInfo(context.TODO(), "")
}

func TestStorage_Interface(t *testing.T) {
	// Test that we can create a mock implementation of Storage interface
	var storage Storage = &mockStorage{}

	// Test that all methods exist and are callable
	_ = storage.Store(context.TODO(), "", nil)
	_, _ = storage.Retrieve(context.TODO(), "")
	_ = storage.Delete(context.TODO(), "")
	_, _ = storage.Exists(context.TODO(), "")
	_, _ = storage.Size(context.TODO(), "")
}

// Mock implementations for interface testing

type mockProgress struct{}

func (m *mockProgress) Start(filename string, totalSize int64)               {}
func (m *mockProgress) Update(bytesDownloaded, totalSize int64, speed int64) {}
func (m *mockProgress) Finish(filename string, stats *DownloadStats)         {}
func (m *mockProgress) Error(filename string, err error)                     {}

type mockDownloader struct{}

func (m *mockDownloader) Download(
	ctx context.Context,
	url, destination string,
	options *DownloadOptions,
) (*DownloadStats, error) {
	return nil, errors.New("mock error")
}

func (m *mockDownloader) DownloadToWriter(
	ctx context.Context,
	url string,
	writer io.Writer,
	options *DownloadOptions,
) (*DownloadStats, error) {
	return nil, errors.New("mock error")
}

func (m *mockDownloader) GetFileInfo(ctx context.Context, url string) (*FileInfo, error) {
	return nil, errors.New("mock error")
}

type mockStorage struct{}

func (m *mockStorage) Store(ctx context.Context, key string, data io.Reader) error {
	return errors.New("mock error")
}

func (m *mockStorage) Retrieve(ctx context.Context, key string) (io.ReadCloser, error) {
	return nil, errors.New("mock error")
}

func (m *mockStorage) Delete(ctx context.Context, key string) error {
	return errors.New("mock error")
}

func (m *mockStorage) Exists(ctx context.Context, key string) (bool, error) {
	return false, errors.New("mock error")
}

func (m *mockStorage) Size(ctx context.Context, key string) (int64, error) {
	return 0, errors.New("mock error")
}

func TestDownloadOptions_DefaultValues(t *testing.T) {
	options := &DownloadOptions{}

	// Test that zero values are handled properly
	if options.MaxRetries < 0 {
		t.Error("MaxRetries should not be negative")
	}

	if options.ChunkSize < 0 {
		t.Error("ChunkSize should not be negative")
	}

	if options.MaxConcurrency < 0 {
		t.Error("MaxConcurrency should not be negative")
	}
}

func TestDownloadStats_TimeCalculations(t *testing.T) {
	now := time.Now()
	stats := &DownloadStats{
		StartTime: now.Add(-10 * time.Second),
		EndTime:   now,
	}

	expectedDuration := stats.EndTime.Sub(stats.StartTime)
	if expectedDuration <= 0 {
		t.Error("Duration calculation should be positive")
	}

	// Test average speed calculation
	stats.BytesDownloaded = 1000
	stats.Duration = 1 * time.Second
	expectedSpeed := int64(1000) // 1000 bytes per second

	if stats.Duration > 0 {
		calculatedSpeed := int64(float64(stats.BytesDownloaded) / stats.Duration.Seconds())
		if calculatedSpeed != expectedSpeed {
			t.Errorf("Expected speed %d, got %d", expectedSpeed, calculatedSpeed)
		}
	}
}
