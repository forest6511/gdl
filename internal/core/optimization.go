package core

import "github.com/forest6511/gdl/pkg/types"

// getOptimalConcurrency returns optimal concurrency based on content length
// This function implements smart defaults to improve performance:
// - Small files (<100KB): Single connection to minimize overhead (benchmark: 110% of curl speed)
// - Small-medium files (100KB-10MB): 2 connections for balanced performance
// - Medium files (10-100MB): 4 connections for better throughput
// - Large files (100MB-1GB): 8 connections for high throughput
// - Very large files (>1GB): 16 connections for maximum performance
func getOptimalConcurrency(contentLength int64) int {
	// Very small files: single connection outperforms concurrent (benchmark data shows 110% of curl)
	if contentLength < 100*1024 { // < 100KB
		return 1
	}

	// Small to medium files: minimal concurrency to reduce overhead
	if contentLength < 10*1024*1024 { // < 10MB
		return 2
	}

	// Medium files: moderate concurrency balances speed and resources
	if contentLength < 100*1024*1024 { // < 100MB
		return 4
	}

	// Large files: higher concurrency maximizes throughput
	if contentLength < 1024*1024*1024 { // < 1GB
		return 8
	}

	// Very large files: maximum concurrency for best performance
	return 16
}

// shouldUseLightweightMode determines if lightweight mode should be used
// for very small files to minimize overhead
func shouldUseLightweightMode(contentLength int64) bool {
	return contentLength < 1024*1024 // < 1MB
}

// getOptimalChunkSize returns optimal chunk size based on content length
// Smaller chunks for small files, larger chunks for large files
func getOptimalChunkSize(contentLength int64) int64 {
	// Very small files: use small chunks to avoid memory waste
	if contentLength < 1024*1024 { // < 1MB
		return 32 * 1024 // 32KB
	}

	// Small to medium files: balanced chunk size
	if contentLength < 100*1024*1024 { // < 100MB
		return 128 * 1024 // 128KB
	}

	// Large files: larger chunks for better performance
	return 1024 * 1024 // 1MB
}

// optimizeOptionsForContentLength adjusts download options based on content length
// This function should be called after Content-Length is determined to apply
// smart optimizations that improve performance
func optimizeOptionsForContentLength(options *types.DownloadOptions, contentLength int64) {
	// Get platform-specific optimal chunk size
	optimalChunkSize := GetOptimalChunkSizePlatform(contentLength)

	// Only optimize if user hasn't explicitly set MaxConcurrency
	// (We assume default was set to 4, so any user-specified value would be different)
	if options.MaxConcurrency == 4 { // This is our default value
		options.MaxConcurrency = getOptimalConcurrency(contentLength)
	}

	// Optimize chunk size if user hasn't explicitly set it
	// Check if it's still the default value (32KB) or a platform default
	if options.ChunkSize == 32*1024 || options.ChunkSize == int64(optimalChunkSize) {
		options.ChunkSize = int64(optimalChunkSize)
	}
}
