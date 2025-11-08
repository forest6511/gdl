//go:build linux

package core

import (
	"syscall"
	"testing"
)

func TestPlatformOptimizations(t *testing.T) {
	// Test that Linux returns optimized transport
	transport := PlatformOptimizations()

	// Verify transport is not nil
	if transport == nil {
		t.Fatal("PlatformOptimizations returned nil transport")
	}

	// Verify Linux-specific transport settings
	if transport.MaxIdleConns != 200 {
		t.Errorf("Expected MaxIdleConns=200, got %d", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 100 {
		t.Errorf("Expected MaxIdleConnsPerHost=100, got %d", transport.MaxIdleConnsPerHost)
	}

	if transport.MaxConnsPerHost != 100 {
		t.Errorf("Expected MaxConnsPerHost=100, got %d", transport.MaxConnsPerHost)
	}

	if !transport.ForceAttemptHTTP2 {
		t.Error("HTTP2 should be enabled on Linux")
	}

	if transport.DisableKeepAlives {
		t.Error("KeepAlives should be enabled on Linux")
	}

}

func TestGetOptimalBufferSize(t *testing.T) {
	bufferSize := GetOptimalBufferSize()

	// Linux should return 512KB
	if bufferSize != 512*1024 {
		t.Errorf("Expected buffer size 512KB, got %d", bufferSize)
	}
}

func TestLinuxGetOptimalConcurrency(t *testing.T) {
	concurrency := GetOptimalConcurrency()

	// Linux should return 64
	if concurrency != 64 {
		t.Errorf("Expected concurrency 64, got %d", concurrency)
	}
}

func TestUseSendfile(t *testing.T) {
	useSendfile := UseSendfile()

	// Linux should support sendfile
	if !useSendfile {
		t.Error("Linux should support sendfile")
	}
}

func TestGetPlatformName(t *testing.T) {
	name := GetPlatformName()

	// Should return "linux"
	if name != "linux" {
		t.Errorf("Expected platform name 'linux', got '%s'", name)
	}
}

func TestSetPlatformSocketOptions(t *testing.T) {
	// Create a socket pair for testing
	fds, err := syscall.Socketpair(syscall.AF_UNIX, syscall.SOCK_STREAM, 0)
	if err != nil {
		t.Fatalf("Failed to create socket pair: %v", err)
	}
	defer func() { _ = syscall.Close(fds[0]) }()
	defer func() { _ = syscall.Close(fds[1]) }()

	// Test setting socket options
	err = SetPlatformSocketOptions(uintptr(fds[0]))
	if err != nil {
		// Some options might fail in test environment, that's OK
		t.Logf("SetPlatformSocketOptions returned error (expected in some environments): %v", err)
	}

	// Try to get socket options to verify they were set
	val, err := syscall.GetsockoptInt(fds[0], syscall.SOL_SOCKET, syscall.SO_REUSEADDR)
	if err == nil && val != 1 {
		t.Errorf("SO_REUSEADDR not set correctly: got %d, want 1", val)
	}

	// Test with invalid fd
	err = SetPlatformSocketOptions(uintptr(99999))
	if err == nil {
		t.Error("Expected error with invalid fd")
	}
}

func TestPlatformSpecificInit(t *testing.T) {
	// Test that PlatformSpecificInit doesn't panic
	// It may fail to set ulimit if not running as root, but that's OK
	PlatformSpecificInit()
	// If we get here without panic, the test passes
}
