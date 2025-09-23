package core

import (
	"testing"

	"github.com/forest6511/gdl/pkg/types"
)

func TestGetOptimalConcurrency(t *testing.T) {
	tests := []struct {
		name        string
		size        int64
		expected    int
		description string
	}{
		{
			name:        "Very small file (1KB)",
			size:        1024,
			expected:    1,
			description: "Small files should use single connection",
		},
		{
			name:        "Small file (50KB)",
			size:        50 * 1024,
			expected:    1,
			description: "Files under 100KB should use single connection",
		},
		{
			name:        "Small-medium file (500KB)",
			size:        500 * 1024,
			expected:    2,
			description: "Files 100KB-10MB should use 2 connections",
		},
		{
			name:        "Medium file (5MB)",
			size:        5 * 1024 * 1024,
			expected:    2,
			description: "Files under 10MB should use 2 connections",
		},
		{
			name:        "Larger file (50MB)",
			size:        50 * 1024 * 1024,
			expected:    4,
			description: "Files 10-100MB should use 4 connections",
		},
		{
			name:        "Large file (500MB)",
			size:        500 * 1024 * 1024,
			expected:    8,
			description: "Files 100MB-1GB should use 8 connections",
		},
		{
			name:        "Very large file (2GB)",
			size:        2 * 1024 * 1024 * 1024,
			expected:    16,
			description: "Files over 1GB should use maximum concurrency",
		},
		{
			name:        "Edge case (exactly 100KB)",
			size:        100 * 1024,
			expected:    2,
			description: "Files at 100KB boundary should use 2 connections",
		},
		{
			name:        "Edge case (exactly 10MB)",
			size:        10 * 1024 * 1024,
			expected:    4,
			description: "Files at 10MB boundary should use 4 connections",
		},
		{
			name:        "Edge case (exactly 100MB)",
			size:        100 * 1024 * 1024,
			expected:    8,
			description: "Files at 100MB boundary should use 8 connections",
		},
		{
			name:        "Edge case (exactly 1GB)",
			size:        1024 * 1024 * 1024,
			expected:    16,
			description: "Files at 1GB boundary should use maximum concurrency",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getOptimalConcurrency(tt.size)
			if result != tt.expected {
				t.Errorf("getOptimalConcurrency(%d) = %d, want %d (%s)",
					tt.size, result, tt.expected, tt.description)
			}
		})
	}
}

func TestShouldUseLightweightMode(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected bool
	}{
		{
			name:     "Very small file (1KB)",
			size:     1024,
			expected: true,
		},
		{
			name:     "Small file (512KB)",
			size:     512 * 1024,
			expected: true,
		},
		{
			name:     "Medium file (1MB)",
			size:     1024 * 1024,
			expected: false,
		},
		{
			name:     "Large file (10MB)",
			size:     10 * 1024 * 1024,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shouldUseLightweightMode(tt.size)
			if result != tt.expected {
				t.Errorf("shouldUseLightweightMode(%d) = %t, want %t",
					tt.size, result, tt.expected)
			}
		})
	}
}

func TestGetOptimalChunkSize(t *testing.T) {
	tests := []struct {
		name     string
		size     int64
		expected int64
	}{
		{
			name:     "Very small file (1KB)",
			size:     1024,
			expected: 32 * 1024, // 32KB
		},
		{
			name:     "Small file (512KB)",
			size:     512 * 1024,
			expected: 32 * 1024, // 32KB
		},
		{
			name:     "Medium file (50MB)",
			size:     50 * 1024 * 1024,
			expected: 128 * 1024, // 128KB
		},
		{
			name:     "Large file (500MB)",
			size:     500 * 1024 * 1024,
			expected: 1024 * 1024, // 1MB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getOptimalChunkSize(tt.size)
			if result != tt.expected {
				t.Errorf("getOptimalChunkSize(%d) = %d, want %d",
					tt.size, result, tt.expected)
			}
		})
	}
}

func TestOptimizeOptionsForContentLength(t *testing.T) {
	tests := []struct {
		name                   string
		initialOptions         *types.DownloadOptions
		contentLength          int64
		expectedMaxConcurrency int
		expectedChunkSize      int64
		shouldOptimize         bool
	}{
		{
			name: "Default options with small file",
			initialOptions: &types.DownloadOptions{
				MaxConcurrency: 4,         // Default value
				ChunkSize:      32 * 1024, // Default value
			},
			contentLength:          1 * 1024 * 1024,                                     // 1MB
			expectedMaxConcurrency: 2,                                                   // Optimized for small-medium file
			expectedChunkSize:      int64(GetOptimalChunkSizePlatform(1 * 1024 * 1024)), // Platform-aware optimization
			shouldOptimize:         true,
		},
		{
			name: "Default options with large file",
			initialOptions: &types.DownloadOptions{
				MaxConcurrency: 4,         // Default value
				ChunkSize:      32 * 1024, // Default value
			},
			contentLength:          500 * 1024 * 1024,                                     // 500MB
			expectedMaxConcurrency: 8,                                                     // Optimized for large file
			expectedChunkSize:      int64(GetOptimalChunkSizePlatform(500 * 1024 * 1024)), // Platform-aware optimization
			shouldOptimize:         true,
		},
		{
			name: "User-specified options should not be optimized",
			initialOptions: &types.DownloadOptions{
				MaxConcurrency: 2,         // User-specified value
				ChunkSize:      64 * 1024, // User-specified value
			},
			contentLength:          10 * 1024 * 1024, // 10MB
			expectedMaxConcurrency: 2,                // Should remain unchanged
			expectedChunkSize:      64 * 1024,        // Should remain unchanged
			shouldOptimize:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Make a copy to avoid modifying the test case
			options := &types.DownloadOptions{
				MaxConcurrency: tt.initialOptions.MaxConcurrency,
				ChunkSize:      tt.initialOptions.ChunkSize,
			}

			optimizeOptionsForContentLength(options, tt.contentLength)

			if options.MaxConcurrency != tt.expectedMaxConcurrency {
				t.Errorf("MaxConcurrency = %d, want %d",
					options.MaxConcurrency, tt.expectedMaxConcurrency)
			}

			if options.ChunkSize != tt.expectedChunkSize {
				t.Errorf("ChunkSize = %d, want %d",
					options.ChunkSize, tt.expectedChunkSize)
			}
		})
	}
}

// Benchmark tests to verify performance of optimization functions
func BenchmarkGetOptimalConcurrency(b *testing.B) {
	sizes := []int64{
		1024,               // 1KB
		10 * 1024 * 1024,   // 10MB
		100 * 1024 * 1024,  // 100MB
		1024 * 1024 * 1024, // 1GB
	}

	for _, size := range sizes {
		b.Run("Size_"+formatSize(size), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				getOptimalConcurrency(size)
			}
		})
	}
}

func formatSize(size int64) string {
	switch {
	case size < 1024:
		return "1KB"
	case size < 1024*1024:
		return "1KB"
	case size < 1024*1024*1024:
		return "1MB"
	default:
		return "1GB"
	}
}
