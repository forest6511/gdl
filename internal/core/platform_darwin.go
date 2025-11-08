//go:build darwin

package core

import (
	"net/http"
	"syscall"
	"time"
)

// PlatformOptimizations returns macOS-specific optimizations
func PlatformOptimizations() *http.Transport {
	return &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        150, // macOS handles more connections efficiently
		MaxIdleConnsPerHost: 50,
		MaxConnsPerHost:     50,
		IdleConnTimeout:     120 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableKeepAlives:   false,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}
}

// SetPlatformSocketOptions sets macOS-specific socket options
func SetPlatformSocketOptions(fd uintptr) error {
	// Enable SO_REUSEADDR
	err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		return err
	}

	// Enable SO_REUSEPORT on macOS for better load distribution
	err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEPORT, 1)
	if err != nil {
		return err
	}

	// Set send buffer size (macOS default is often small)
	err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 4*1024*1024)
	if err != nil {
		return err
	}

	// Set receive buffer size
	err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 4*1024*1024)
	if err != nil {
		return err
	}

	return nil
}

// GetOptimalBufferSize returns the optimal buffer size for macOS
func GetOptimalBufferSize() int {
	// macOS performs well with 256KB buffers
	return 256 * 1024
}

// GetOptimalConcurrency returns the optimal concurrency for macOS
func GetOptimalConcurrency() int {
	// macOS handles concurrency well due to efficient kqueue
	return 32
}

// UseSendfile checks if sendfile should be used on macOS
func UseSendfile() bool {
	// macOS supports sendfile but it's not as efficient as Linux
	// Use for files > 50MB
	return true
}

// GetPlatformName returns the platform name
func GetPlatformName() string {
	return "darwin"
}

// PlatformSpecificInit performs any platform-specific initialization
func PlatformSpecificInit() {
	// macOS-specific initialization
	// Increase ulimit if possible
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err == nil {
		rLimit.Cur = rLimit.Max
		_ = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	}
}
