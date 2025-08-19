// Package main demonstrates the complete gdl library API usage
// with practical examples of all public functions and features.
//
// Usage:
//   go run main.go
//
// This example shows:
// - Basic download operations
// - Progress tracking with callbacks
// - Download to memory and custom writers
// - Resume functionality
// - File information retrieval
// - Comprehensive error handling

package main

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/forest6511/gdl"
)

func main() {
	fmt.Println("=== Godl Library Usage Examples ===")

	// Example 1: Basic download
	example1BasicDownload()

	// Example 2: Download with progress tracking
	example2ProgressTracking()

	// Example 3: Download with options
	example3DownloadWithOptions()

	// Example 4: Download to memory
	example4DownloadToMemory()

	// Example 5: Error handling
	example5ErrorHandling()

	fmt.Println()
	fmt.Println("=== All examples completed successfully ===")
}

func example1BasicDownload() {
	fmt.Println("1. Basic Download Example")
	fmt.Println(strings.Repeat("-", 30))

	url := "https://httpbin.org/bytes/1024"
	filename := "basic_download.bin"

	stats, err := gdl.Download(context.Background(), url, filename)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Downloaded %s successfully: %d bytes in %v\n", filename, stats.BytesDownloaded, stats.Duration)
	fmt.Println()
}

func example2ProgressTracking() {
	fmt.Println("2. Progress Tracking Example")
	fmt.Println(strings.Repeat("-", 30))

	url := "https://httpbin.org/bytes/10240"
	filename := "progress_download.bin"

	// Create progress callback
	progressCallback := func(p gdl.Progress) {
		fmt.Printf("\rProgress: %.1f%% (%d/%d bytes) Speed: %.2f KB/s",
			p.Percentage, p.BytesDownloaded, p.TotalSize, float64(p.Speed)/1024)
	}

	options := &gdl.Options{
		ProgressCallback:  progressCallback,
		OverwriteExisting: true,
	}

	stats, err := gdl.DownloadWithOptions(context.Background(), url, filename, options)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("\nCompleted download of %s: %d bytes (avg speed: %.2f KB/s)\n",
		filename, stats.BytesDownloaded, float64(stats.AverageSpeed)/1024)
	fmt.Println()
}

func example3DownloadWithOptions() {
	fmt.Println("3. Download with Options Example")
	fmt.Println(strings.Repeat("-", 30))

	url := "https://httpbin.org/bytes/51200"
	filename := "options_download.bin"

	options := &gdl.Options{
		MaxConcurrency:    4,
		ChunkSize:         12800, // 12.5KB chunks
		EnableResume:      true,
		OverwriteExisting: true,
		Headers: map[string]string{
			"User-Agent": "gdl-examples/1.0",
		},
		Timeout: 30 * time.Second,
	}

	start := time.Now()
	stats, err := gdl.DownloadWithOptions(context.Background(), url, filename, options)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Downloaded with options: %d bytes in %v (resumed: %v)\n",
		stats.BytesDownloaded, time.Since(start), stats.Resumed)
	fmt.Println()
}

func example4DownloadToMemory() {
	fmt.Println("4. Download to Memory Example")
	fmt.Println(strings.Repeat("-", 30))

	url := "https://httpbin.org/bytes/2048"

	data, stats, err := gdl.DownloadToMemory(context.Background(), url)
	if err != nil {
		log.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Downloaded %d bytes to memory in %v (success: %v)\n",
		len(data), stats.Duration, stats.Success)
	fmt.Println()
}

func example5ErrorHandling() {
	fmt.Println("5. Error Handling Example")
	fmt.Println(strings.Repeat("-", 30))

	// Try to download from an invalid URL
	url := "https://invalid-domain-that-does-not-exist.com/file.bin"
	filename := "error_test.bin"

	options := &gdl.Options{
		RetryAttempts: 2,
		Timeout:       5 * time.Second,
	}

	fmt.Println("Attempting download with retry...")
	stats, err := gdl.DownloadWithOptions(context.Background(), url, filename, options)
	if err != nil {
		fmt.Printf("Download failed as expected: %v (stats: %v)\n",
			err, stats != nil)
	} else {
		fmt.Printf("Unexpected success: %d bytes downloaded\n", stats.BytesDownloaded)
	}
	fmt.Println()
}
