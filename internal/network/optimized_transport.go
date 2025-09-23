package network

import (
	"net"
	"net/http"
	"time"
)

// CreateOptimizedTransport creates an HTTP transport optimized for file downloads
// with improved connection pooling, timeouts, and HTTP/2 support
func CreateOptimizedTransport() *http.Transport {
	return &http.Transport{
		// Connection pooling settings for better reuse
		MaxIdleConns:        100, // Increase from default 100
		MaxIdleConnsPerHost: 20,  // Increase from default 2
		MaxConnsPerHost:     30,  // Limit concurrent connections per host
		IdleConnTimeout:     90 * time.Second,

		// Timeout settings optimized for downloads
		TLSHandshakeTimeout:   10 * time.Second, // Default is 10s
		ExpectContinueTimeout: 1 * time.Second,  // Default is 1s
		ResponseHeaderTimeout: 10 * time.Second, // Timeout for response headers

		// Connection settings
		DisableKeepAlives:  false, // Keep connections alive for reuse
		DisableCompression: false, // Allow compression for headers (body usually not compressed)
		ForceAttemptHTTP2:  true,  // Try HTTP/2 first for better performance

		// Dialer settings for connection establishment
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second, // Connection timeout
			KeepAlive: 30 * time.Second, // Keep-alive probe interval
			DualStack: true,             // Enable both IPv4 and IPv6
		}).DialContext,

		// Buffer sizes for better throughput
		WriteBufferSize: 64 * 1024, // 64KB write buffer
		ReadBufferSize:  64 * 1024, // 64KB read buffer
	}
}

// CreateOptimizedClient creates an HTTP client with optimized transport settings
func CreateOptimizedClient(timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = 30 * time.Minute // Default timeout for large downloads
	}

	return &http.Client{
		Transport: CreateOptimizedTransport(),
		Timeout:   timeout,
		// CheckRedirect is default (follows up to 10 redirects)
	}
}

// CreateLightweightClient creates a minimal HTTP client for small files
// with reduced overhead and connection pooling
func CreateLightweightClient() *http.Client {
	return &http.Client{
		Transport: &http.Transport{
			MaxIdleConns:          10,
			MaxIdleConnsPerHost:   2,
			MaxConnsPerHost:       5,
			IdleConnTimeout:       30 * time.Second,
			TLSHandshakeTimeout:   5 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
			DisableKeepAlives:     false,
			DisableCompression:    false,
			ForceAttemptHTTP2:     true,

			DialContext: (&net.Dialer{
				Timeout:   10 * time.Second,
				KeepAlive: 30 * time.Second,
				DualStack: true,
			}).DialContext,

			// Smaller buffers for small files
			WriteBufferSize: 16 * 1024, // 16KB
			ReadBufferSize:  16 * 1024, // 16KB
		},
		Timeout: 5 * time.Minute, // Shorter timeout for small files
	}
}
