//go:build linux

package core

import (
	"net/http"
	"syscall"
	"time"
)

// PlatformOptimizations returns Linux-specific optimizations
func PlatformOptimizations() *http.Transport {
	return &http.Transport{
		Proxy:               http.ProxyFromEnvironment,
		MaxIdleConns:        200, // Linux handles many connections efficiently
		MaxIdleConnsPerHost: 100,
		MaxConnsPerHost:     100,
		IdleConnTimeout:     90 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
		DisableKeepAlives:   false,
		DisableCompression:  false,
		ForceAttemptHTTP2:   true,
	}
}

// SetPlatformSocketOptions sets Linux-specific socket options
func SetPlatformSocketOptions(fd uintptr) error {
	// Enable SO_REUSEADDR
	err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_REUSEADDR, 1)
	if err != nil {
		return err
	}

	// Enable TCP_NODELAY for lower latency
	err = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1)
	if err != nil {
		return err
	}

	// Enable TCP_QUICKACK for faster ACKs
	err = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, 0x0C, 1) // TCP_QUICKACK = 0x0C
	if err != nil {
		return err
	}

	// Set send buffer size (Linux auto-tunes but we can hint)
	err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, 6*1024*1024)
	if err != nil {
		return err
	}

	// Set receive buffer size
	err = syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, 6*1024*1024)
	if err != nil {
		return err
	}

	// Enable TCP_CORK for better packet batching
	_ = syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, 0x03, 1) // TCP_CORK = 0x03

	return nil
}

// GetOptimalBufferSize returns the optimal buffer size for Linux
func GetOptimalBufferSize() int {
	// Linux performs well with 512KB-1MB buffers
	return 512 * 1024
}

// GetOptimalConcurrency returns the optimal concurrency for Linux
func GetOptimalConcurrency() int {
	// Linux with epoll can handle many connections efficiently
	return 64
}

// UseSendfile checks if sendfile should be used on Linux
func UseSendfile() bool {
	// Linux has excellent sendfile support
	return true
}

// GetPlatformName returns the platform name
func GetPlatformName() string {
	return "linux"
}

// PlatformSpecificInit performs any platform-specific initialization
func PlatformSpecificInit() {
	// Linux-specific initialization
	// Increase ulimit if possible
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err == nil {
		// Try to set to a reasonable high value
		rLimit.Cur = 65536
		if rLimit.Cur > rLimit.Max {
			rLimit.Cur = rLimit.Max
		}
		_ = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	}

	// Set TCP settings via sysctl (would need root)
	// In production, these would be set at system level:
	// net.core.rmem_max = 134217728
	// net.core.wmem_max = 134217728
	// net.ipv4.tcp_rmem = 4096 87380 134217728
	// net.ipv4.tcp_wmem = 4096 65536 134217728
}
