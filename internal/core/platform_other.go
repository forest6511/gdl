//go:build !linux && !darwin && !windows
// +build !linux,!darwin,!windows

package core

import (
	"net/http"
	"time"
)

// PlatformOptimizations returns generic optimizations for other platforms
func PlatformOptimizations() *http.Transport {
	return &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        100,
		MaxIdleConnsPerHost: 20,
		MaxConnsPerHost:     20,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableKeepAlives:   false,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}
}

// SetPlatformSocketOptions sets generic socket options
func SetPlatformSocketOptions(fd uintptr) error {
	// Generic platforms use default socket options
	return nil
}

// GetOptimalBufferSize returns the optimal buffer size for generic platforms
func GetOptimalBufferSize() int {
	// Use conservative 64KB buffers for unknown platforms
	return 64 * 1024
}

// GetOptimalConcurrency returns the optimal concurrency for generic platforms
func GetOptimalConcurrency() int {
	// Conservative concurrency for unknown platforms
	return 8
}

// UseSendfile checks if sendfile should be used
func UseSendfile() bool {
	// Disable sendfile on unknown platforms
	return false
}

// GetPlatformName returns the platform name
func GetPlatformName() string {
	return "other"
}

// PlatformSpecificInit performs any platform-specific initialization
func PlatformSpecificInit() {
	// No specific initialization for generic platforms
}
