package core

import (
	"runtime"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	info := DetectPlatform()

	// Basic validation
	if info == nil {
		t.Fatal("DetectPlatform returned nil")
	}

	// OS should be current runtime OS
	if info.OS != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, info.OS)
	}

	// Arch should be current runtime arch
	if info.Arch != runtime.GOARCH {
		t.Errorf("Expected Arch %s, got %s", runtime.GOARCH, info.Arch)
	}

	// NumCPU should be positive
	if info.NumCPU <= 0 {
		t.Errorf("NumCPU should be positive, got %d", info.NumCPU)
	}

	// Check ARM detection
	expectedARM := runtime.GOARCH == "arm" || runtime.GOARCH == "arm64"
	if info.IsARM != expectedARM {
		t.Errorf("IsARM should be %v for %s, got %v", expectedARM, runtime.GOARCH, info.IsARM)
	}

	// Optimizations should be set
	if info.Optimizations.BufferSize <= 0 {
		t.Errorf("BufferSize should be positive, got %d", info.Optimizations.BufferSize)
	}

	if info.Optimizations.Concurrency <= 0 {
		t.Errorf("Concurrency should be positive, got %d", info.Optimizations.Concurrency)
	}

	if info.Optimizations.MaxConnections <= 0 {
		t.Errorf("MaxConnections should be positive, got %d", info.Optimizations.MaxConnections)
	}

	// Test platform-specific features
	switch runtime.GOOS {
	case "linux":
		// Linux should support zero-copy
		if !info.Optimizations.UseZeroCopy {
			t.Error("Linux should support zero-copy")
		}
	case "darwin":
		// macOS should support zero-copy
		if !info.Optimizations.UseZeroCopy {
			t.Error("macOS should support zero-copy")
		}
	case "windows":
		// Windows doesn't support sendfile
		if info.Optimizations.UseZeroCopy {
			t.Error("Windows should not support zero-copy")
		}
	}

	// Test that optimizations are adjusted based on system
	if info.NumCPU <= 2 {
		// Low CPU count should have limited concurrency
		if info.Optimizations.Concurrency > 4 {
			t.Errorf("Low CPU count should limit concurrency, got %d", info.Optimizations.Concurrency)
		}
	}

	// Test server-grade detection
	if info.NumCPU >= 8 {
		if !info.IsServerGrade {
			t.Error("System with 8+ CPUs should be detected as server-grade")
		}
	}

	// Verify optimizations are reasonable
	if info.Optimizations.BufferSize < 8*1024 || info.Optimizations.BufferSize > 8*1024*1024 {
		t.Errorf("BufferSize out of reasonable range: %d", info.Optimizations.BufferSize)
	}

	// Test HTTP2 and connection reuse should be enabled
	if !info.Optimizations.EnableHTTP2 {
		t.Error("HTTP2 should be enabled by default")
	}
	if !info.Optimizations.ConnectionReuse {
		t.Error("Connection reuse should be enabled by default")
	}
}

func TestGetOptimizationsForPlatform(t *testing.T) {
	tests := []struct {
		name     string
		info     *PlatformInfo
		expected func(*PlatformOptimizationSet) bool
	}{
		{
			name: "Linux with 4 CPUs",
			info: &PlatformInfo{
				OS:            "linux",
				Arch:          "amd64",
				NumCPU:        4,
				IsARM:         false,
				IsServerGrade: false,
			},
			expected: func(opts *PlatformOptimizationSet) bool {
				return opts.BufferSize == 512*1024 &&
					opts.Concurrency == 32 &&
					opts.MaxConnections == 200 &&
					opts.UseSendfile == true &&
					opts.UseZeroCopy == true
			},
		},
		{
			name: "Darwin (macOS) with 8 CPUs",
			info: &PlatformInfo{
				OS:            "darwin",
				Arch:          "amd64",
				NumCPU:        8,
				IsARM:         false,
				IsServerGrade: true,
			},
			expected: func(opts *PlatformOptimizationSet) bool {
				return opts.BufferSize == 256*1024 &&
					opts.Concurrency == 32 &&
					opts.MaxConnections == 150 &&
					opts.UseSendfile == true &&
					opts.UseZeroCopy == true
			},
		},
		{
			name: "Windows with 4 CPUs",
			info: &PlatformInfo{
				OS:            "windows",
				Arch:          "amd64",
				NumCPU:        4,
				IsARM:         false,
				IsServerGrade: false,
			},
			expected: func(opts *PlatformOptimizationSet) bool {
				return opts.BufferSize == 128*1024 &&
					opts.Concurrency == 8 &&
					opts.MaxConnections == 100 &&
					opts.UseSendfile == false &&
					opts.UseZeroCopy == false
			},
		},
		{
			name: "ARM64 server grade",
			info: &PlatformInfo{
				OS:            "linux",
				Arch:          "arm64",
				NumCPU:        16,
				IsARM:         true,
				IsServerGrade: true,
			},
			expected: func(opts *PlatformOptimizationSet) bool {
				return opts.BufferSize == 128*1024 &&
					opts.Concurrency == 96 &&
					opts.MaxConnections == 150
			},
		},
		{
			name: "ARM64 mobile/embedded",
			info: &PlatformInfo{
				OS:            "linux",
				Arch:          "arm64",
				NumCPU:        4,
				IsARM:         true,
				IsServerGrade: false,
			},
			expected: func(opts *PlatformOptimizationSet) bool {
				return opts.BufferSize == 128*1024 &&
					opts.Concurrency == 8 &&
					opts.MaxConnections == 50
			},
		},
		{
			name: "32-bit ARM",
			info: &PlatformInfo{
				OS:            "linux",
				Arch:          "arm",
				NumCPU:        2,
				IsARM:         true,
				IsServerGrade: false,
			},
			expected: func(opts *PlatformOptimizationSet) bool {
				return opts.BufferSize == 32*1024 &&
					opts.Concurrency == 4 && // Will be overridden by low CPU count
					opts.MaxConnections == 20 && // Will be overridden by low CPU count
					opts.UseZeroCopy == false
			},
		},
		{
			name: "Low-end hardware (2 CPUs)",
			info: &PlatformInfo{
				OS:            "linux",
				Arch:          "amd64",
				NumCPU:        2,
				IsARM:         false,
				IsServerGrade: false,
			},
			expected: func(opts *PlatformOptimizationSet) bool {
				return opts.Concurrency == 4 &&
					opts.MaxConnections == 20 &&
					opts.BufferSize == 32*1024
			},
		},
		{
			name: "Unknown OS",
			info: &PlatformInfo{
				OS:            "freebsd",
				Arch:          "amd64",
				NumCPU:        4,
				IsARM:         false,
				IsServerGrade: false,
			},
			expected: func(opts *PlatformOptimizationSet) bool {
				return opts.BufferSize == 64*1024 &&
					opts.Concurrency == 8 &&
					opts.MaxConnections == 50 &&
					opts.UseSendfile == false &&
					opts.UseZeroCopy == false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := getOptimizationsForPlatform(tt.info)

			// Check basic optimizations that should always be enabled
			if !opts.EnableHTTP2 {
				t.Error("EnableHTTP2 should be true")
			}
			if !opts.ConnectionReuse {
				t.Error("ConnectionReuse should be true")
			}

			// Check platform-specific optimizations
			if !tt.expected(&opts) {
				t.Errorf("Optimizations don't match expected for %s", tt.name)
			}
		})
	}
}

func TestGetPlatformString(t *testing.T) {
	str := GetPlatformString()

	// Should contain OS and architecture
	if str == "" {
		t.Error("GetPlatformString returned empty string")
	}

	// Should contain runtime info
	if runtime.GOOS == "linux" && !containsAny(str, "linux") {
		t.Errorf("Expected linux in platform string, got: %s", str)
	}

	if runtime.GOOS == "darwin" && !containsAny(str, "darwin") {
		t.Errorf("Expected darwin in platform string, got: %s", str)
	}

	if runtime.GOOS == "windows" && !containsAny(str, "windows") {
		t.Errorf("Expected windows in platform string, got: %s", str)
	}
}

func TestShouldUseZeroCopyPlatform(t *testing.T) {
	tests := []struct {
		name     string
		fileSize int64
		os       string
	}{
		{
			name:     "Small file",
			fileSize: 100 * 1024, // 100KB
			os:       runtime.GOOS,
		},
		{
			name:     "Medium file",
			fileSize: 10 * 1024 * 1024, // 10MB
			os:       runtime.GOOS,
		},
		{
			name:     "Large file",
			fileSize: 100 * 1024 * 1024, // 100MB
			os:       runtime.GOOS,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ShouldUseZeroCopyPlatform(tt.fileSize)

			// On Linux and macOS, zero-copy should be considered for large files
			switch tt.os {
			case "linux", "darwin":
				// For small files, zero-copy might not be beneficial
				// Platform can decide based on file size
			case "windows":
				// Windows doesn't support sendfile currently
				if result {
					t.Errorf("Windows should not use zero-copy, but got true for %d bytes", tt.fileSize)
				}
			}
		})
	}
}

func TestGetOptimalChunkSizePlatform(t *testing.T) {
	tests := []struct {
		name     string
		fileSize int64
		minChunk int
		maxChunk int
	}{
		{
			name:     "Tiny file",
			fileSize: 10 * 1024,  // 10KB
			minChunk: 8 * 1024,   // Minimum 8KB
			maxChunk: 128 * 1024, // Maximum 128KB for small files
		},
		{
			name:     "Small file",
			fileSize: 1 * 1024 * 1024, // 1MB
			minChunk: 16 * 1024,       // Minimum 16KB
			maxChunk: 256 * 1024,      // Maximum 256KB
		},
		{
			name:     "Medium file",
			fileSize: 50 * 1024 * 1024, // 50MB
			minChunk: 32 * 1024,        // Minimum 32KB
			maxChunk: 1024 * 1024,      // Maximum 1MB
		},
		{
			name:     "Large file",
			fileSize: 500 * 1024 * 1024, // 500MB
			minChunk: 64 * 1024,         // Minimum 64KB
			maxChunk: 4 * 1024 * 1024,   // Maximum 4MB
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			chunkSize := GetOptimalChunkSizePlatform(tt.fileSize)

			if chunkSize < tt.minChunk {
				t.Errorf("Chunk size %d is below minimum %d for file size %d",
					chunkSize, tt.minChunk, tt.fileSize)
			}

			if chunkSize > tt.maxChunk {
				t.Errorf("Chunk size %d exceeds maximum %d for file size %d",
					chunkSize, tt.maxChunk, tt.fileSize)
			}
		})
	}
}

func TestPlatformOptimizationConsistency(t *testing.T) {
	info := DetectPlatform()
	opts := info.Optimizations

	// Check consistency of settings
	if opts.UseZeroCopy && !opts.UseSendfile {
		// Zero-copy typically requires sendfile
		if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
			t.Error("UseZeroCopy is true but UseSendfile is false, which is inconsistent")
		}
	}

	// Check HTTP/2 is enabled on modern platforms
	if !opts.EnableHTTP2 {
		t.Log("Warning: HTTP/2 is disabled, which may impact performance")
	}

	// Check connection reuse is enabled
	if !opts.ConnectionReuse {
		t.Error("Connection reuse should be enabled for performance")
	}

	// Buffer size should be reasonable
	if opts.BufferSize < 16*1024 { // Less than 16KB
		t.Errorf("Buffer size %d is too small for efficient I/O", opts.BufferSize)
	}

	if opts.BufferSize > 10*1024*1024 { // More than 10MB
		t.Errorf("Buffer size %d is too large and may waste memory", opts.BufferSize)
	}
}

func TestPlatformSpecificSettings(t *testing.T) {
	info := DetectPlatform()

	// Skip buffer size checks if this is a low-end system (≤2 CPUs)
	// as those get overridden
	isLowEnd := info.NumCPU <= 2

	switch info.OS {
	case "linux":
		// Linux should have specific optimizations
		if !isLowEnd && info.Optimizations.BufferSize != 512*1024 {
			t.Errorf("Linux should use 512KB buffers, got %d", info.Optimizations.BufferSize)
		}
		if !info.Optimizations.UseSendfile {
			t.Error("Linux should support sendfile")
		}
		if !info.Optimizations.UseZeroCopy {
			t.Error("Linux should support zero-copy")
		}

	case "darwin":
		// macOS should have specific optimizations
		// Note: Apple Silicon Macs are ARM64, which have different buffer sizes
		if !isLowEnd && !info.IsARM && info.Optimizations.BufferSize != 256*1024 {
			t.Errorf("Intel macOS should use 256KB buffers, got %d", info.Optimizations.BufferSize)
		}
		if !isLowEnd && info.IsARM && info.Optimizations.BufferSize != 128*1024 {
			t.Errorf("ARM64 macOS should use 128KB buffers, got %d", info.Optimizations.BufferSize)
		}
		if !info.Optimizations.UseSendfile {
			t.Error("macOS should support sendfile")
		}

	case "windows":
		// Windows should have specific optimizations
		if !isLowEnd && info.Optimizations.BufferSize != 128*1024 {
			t.Errorf("Windows should use 128KB buffers, got %d", info.Optimizations.BufferSize)
		}
		if info.Optimizations.UseSendfile {
			t.Error("Windows should not report sendfile support (not exposed in Go)")
		}
	}
}

func TestLowEndHardwareOptimization(t *testing.T) {
	// This test would need to mock NumCPU, which isn't easily doable
	// But we can test the logic by checking current settings
	info := DetectPlatform()

	// If this is a low-end system (2 or fewer CPUs), check optimizations
	if info.NumCPU <= 2 {
		if info.Optimizations.Concurrency > 4 {
			t.Errorf("Low-end system should have limited concurrency, got %d", info.Optimizations.Concurrency)
		}
		if info.Optimizations.MaxConnections > 20 {
			t.Errorf("Low-end system should have limited connections, got %d", info.Optimizations.MaxConnections)
		}
		if info.Optimizations.BufferSize > 32*1024 {
			t.Errorf("Low-end system should use smaller buffers, got %d", info.Optimizations.BufferSize)
		}
	}
}

// Helper function
func containsAny(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && (s == substr || containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	if len(substr) > len(s) {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
