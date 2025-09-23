//go:build windows
// +build windows

package core

import (
	"net/http"
	"time"
)

// PlatformOptimizations returns Windows-specific optimizations
func PlatformOptimizations() *http.Transport {
	return &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        100, // Windows has different connection limits
		MaxIdleConnsPerHost: 30,
		MaxConnsPerHost:     30,
		IdleConnTimeout:     120 * time.Second,
		TLSHandshakeTimeout: 15 * time.Second, // Windows TLS can be slower
		DisableKeepAlives:   false,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}
}

// SetPlatformSocketOptions sets Windows-specific socket options
func SetPlatformSocketOptions(fd uintptr) error {
	// Windows socket options are different from Unix
	// Most optimizations are handled by the Windows TCP/IP stack
	// We rely on Windows auto-tuning for buffer sizes
	return nil
}

// GetOptimalBufferSize returns the optimal buffer size for Windows
func GetOptimalBufferSize() int {
	// Windows performs well with 128KB buffers
	return 128 * 1024
}

// GetOptimalConcurrency returns the optimal concurrency for Windows
func GetOptimalConcurrency() int {
	// Windows IOCP is efficient but has different characteristics
	return 16
}

// UseSendfile checks if sendfile should be used on Windows
func UseSendfile() bool {
	// Windows has TransmitFile but it's not exposed in Go stdlib
	// Use regular copy operations
	return false
}

// GetPlatformName returns the platform name
func GetPlatformName() string {
	return "windows"
}

// PlatformSpecificInit performs any platform-specific initialization
func PlatformSpecificInit() {
	// Windows-specific initialization
	// Windows handles most TCP optimizations automatically
	// via its auto-tuning features introduced in Vista/Server 2008

	// Key Windows optimizations are set at system level:
	// - TCP Window Auto-Tuning
	// - Receive Side Scaling (RSS)
	// - TCP Chimney Offload
	// These require admin privileges to modify
}
