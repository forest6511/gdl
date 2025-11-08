//go:build arm || arm64

package core

import (
	"runtime"
)

// ARMOptimizations provides ARM-specific optimizations
type ARMOptimizations struct {
	// ARM-specific settings
	UseNEON          bool // Use NEON instructions if available
	CacheLineSize    int  // ARM cache line size
	PrefetchDistance int  // Prefetch distance for ARM
}

// GetARMOptimizations returns ARM-specific optimizations
func GetARMOptimizations() *ARMOptimizations {
	opts := &ARMOptimizations{
		CacheLineSize:    64, // Most ARM processors use 64-byte cache lines
		PrefetchDistance: 256,
	}

	// Check for ARM64 which typically has NEON
	if runtime.GOARCH == "arm64" {
		opts.UseNEON = true
	}

	return opts
}

// GetOptimalBufferSizeARM returns the optimal buffer size for ARM
func GetOptimalBufferSizeARM() int {
	// ARM processors, especially mobile ones, benefit from smaller buffers
	// to reduce memory pressure
	if runtime.GOARCH == "arm64" {
		// ARM64 can handle larger buffers
		return 128 * 1024 // 128KB
	}
	// ARM32 should use smaller buffers
	return 32 * 1024 // 32KB
}

// GetOptimalConcurrencyARM returns the optimal concurrency for ARM
func GetOptimalConcurrencyARM() int {
	numCPU := runtime.NumCPU()

	// ARM processors vary widely in core count
	// Mobile ARM: typically 4-8 cores
	// Server ARM: can have 32-128 cores

	if numCPU <= 4 {
		// Mobile or embedded ARM
		return numCPU * 2
	} else if numCPU <= 8 {
		// High-end mobile or low-end server
		return numCPU * 3
	} else {
		// Server-grade ARM (Graviton, Ampere, etc.)
		return numCPU * 4
	}
}

// IsLowPowerMode detects if running on a low-power ARM device
func IsLowPowerMode() bool {
	// Check for Raspberry Pi, mobile devices, etc.
	// This is a simplified check - in production, would check more indicators
	numCPU := runtime.NumCPU()
	return numCPU <= 4
}

// OptimizeForARM applies ARM-specific optimizations
func OptimizeForARM() {
	if IsLowPowerMode() {
		// On low-power ARM devices:
		// - Use smaller buffers to reduce memory usage
		// - Reduce concurrency to save battery
		// - Disable aggressive prefetching
		SetMaxMemoryUsage(256 * 1024 * 1024) // 256MB max
	} else {
		// On server-grade ARM:
		// - Can use larger buffers
		// - Higher concurrency
		// - More aggressive optimizations
		SetMaxMemoryUsage(1024 * 1024 * 1024) // 1GB max
	}
}

// SetMaxMemoryUsage sets a soft memory limit for ARM devices
func SetMaxMemoryUsage(bytes int64) {
	// This would integrate with the buffer pool to limit memory usage
	// Implementation would track total allocated buffers
}

// GetCacheAlignedSize aligns buffer sizes to ARM cache lines
func GetCacheAlignedSize(size int) int {
	cacheLineSize := 64 // ARM typically uses 64-byte cache lines
	if size%cacheLineSize == 0 {
		return size
	}
	return ((size / cacheLineSize) + 1) * cacheLineSize
}
