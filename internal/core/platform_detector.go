package core

import (
	"fmt"
	"net/http"
	"runtime"
	"strings"
)

// PlatformInfo contains information about the current platform
type PlatformInfo struct {
	OS            string
	Arch          string
	NumCPU        int
	IsARM         bool
	IsServerGrade bool
	Optimizations PlatformOptimizationSet
}

// PlatformOptimizationSet contains platform-specific optimizations
type PlatformOptimizationSet struct {
	BufferSize      int
	Concurrency     int
	MaxConnections  int
	UseSendfile     bool
	UseZeroCopy     bool
	EnableHTTP2     bool
	ConnectionReuse bool
}

// DetectPlatform detects the current platform and returns optimization settings
func DetectPlatform() *PlatformInfo {
	info := &PlatformInfo{
		OS:     runtime.GOOS,
		Arch:   runtime.GOARCH,
		NumCPU: runtime.NumCPU(),
	}

	// Check if ARM
	info.IsARM = strings.HasPrefix(runtime.GOARCH, "arm")

	// Determine if server-grade hardware
	info.IsServerGrade = info.NumCPU >= 8

	// Set optimizations based on platform
	info.Optimizations = getOptimizationsForPlatform(info)

	return info
}

// getOptimizationsForPlatform returns optimizations for the detected platform
func getOptimizationsForPlatform(info *PlatformInfo) PlatformOptimizationSet {
	opts := PlatformOptimizationSet{
		EnableHTTP2:     true,
		ConnectionReuse: true,
	}

	// OS-specific optimizations
	switch info.OS {
	case "linux":
		opts.BufferSize = 512 * 1024 // 512KB
		opts.Concurrency = info.NumCPU * 8
		opts.MaxConnections = 200
		opts.UseSendfile = true
		opts.UseZeroCopy = true

	case "darwin": // macOS
		opts.BufferSize = 256 * 1024 // 256KB
		opts.Concurrency = info.NumCPU * 4
		opts.MaxConnections = 150
		opts.UseSendfile = true
		opts.UseZeroCopy = true

	case "windows":
		opts.BufferSize = 128 * 1024 // 128KB
		opts.Concurrency = info.NumCPU * 2
		opts.MaxConnections = 100
		opts.UseSendfile = false // Windows TransmitFile not exposed
		opts.UseZeroCopy = false

	default:
		opts.BufferSize = 64 * 1024 // 64KB
		opts.Concurrency = info.NumCPU * 2
		opts.MaxConnections = 50
		opts.UseSendfile = false
		opts.UseZeroCopy = false
	}

	// Architecture-specific adjustments
	if info.IsARM {
		if info.Arch == "arm64" {
			opts.BufferSize = 128 * 1024
			if info.IsServerGrade {
				// Server ARM (Graviton, Ampere)
				opts.Concurrency = info.NumCPU * 6
				opts.MaxConnections = 150
			} else {
				// Mobile/embedded ARM
				opts.Concurrency = info.NumCPU * 2
				opts.MaxConnections = 50
			}
		} else {
			// 32-bit ARM
			opts.BufferSize = 32 * 1024
			opts.Concurrency = info.NumCPU
			opts.MaxConnections = 30
			opts.UseZeroCopy = false // Limited memory on 32-bit ARM
		}
	}

	// Adjust for low-end hardware
	if info.NumCPU <= 2 {
		opts.Concurrency = 4
		opts.MaxConnections = 20
		opts.BufferSize = 32 * 1024
	}

	return opts
}

// ApplyPlatformOptimizations applies platform-specific optimizations to HTTP transport
func ApplyPlatformOptimizations(transport *http.Transport) {
	info := DetectPlatform()
	opts := info.Optimizations

	transport.MaxIdleConns = opts.MaxConnections
	transport.MaxIdleConnsPerHost = opts.MaxConnections / 4
	transport.MaxConnsPerHost = opts.MaxConnections / 4
	transport.ForceAttemptHTTP2 = opts.EnableHTTP2
	transport.DisableKeepAlives = !opts.ConnectionReuse

	// Platform-specific initialization
	PlatformSpecificInit()
}

// GetPlatformString returns a string describing the platform
func GetPlatformString() string {
	info := DetectPlatform()

	var features []string
	if info.Optimizations.UseZeroCopy {
		features = append(features, "zero-copy")
	}
	if info.Optimizations.UseSendfile {
		features = append(features, "sendfile")
	}
	if info.IsServerGrade {
		features = append(features, "server-grade")
	}

	featureStr := ""
	if len(features) > 0 {
		featureStr = " [" + strings.Join(features, ", ") + "]"
	}

	return fmt.Sprintf("%s/%s (%d CPUs)%s", info.OS, info.Arch, info.NumCPU, featureStr)
}

// ShouldUseZeroCopyPlatform checks if zero-copy should be used on current platform
func ShouldUseZeroCopyPlatform(fileSize int64) bool {
	info := DetectPlatform()

	if !info.Optimizations.UseZeroCopy {
		return false
	}

	// Platform-specific thresholds
	switch info.OS {
	case "linux":
		return fileSize > 1*1024*1024 // 1MB on Linux
	case "darwin":
		return fileSize > 5*1024*1024 // 5MB on macOS
	default:
		return false
	}
}

// GetOptimalChunkSizePlatform returns optimal chunk size for current platform
func GetOptimalChunkSizePlatform(fileSize int64) int {
	info := DetectPlatform()
	baseSize := info.Optimizations.BufferSize

	// Adjust based on file size
	if fileSize < 1*1024*1024 { // < 1MB
		return baseSize / 4
	} else if fileSize < 10*1024*1024 { // < 10MB
		return baseSize / 2
	} else if fileSize < 100*1024*1024 { // < 100MB
		return baseSize
	} else { // >= 100MB
		return baseSize * 2
	}
}
