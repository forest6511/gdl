// Package main demonstrates concurrent download functionality.
//
// This example shows:
// - High-performance concurrent downloads with multiple connections
// - Custom chunk size configuration
// - Comparison between single-threaded and multi-threaded downloads
// - Performance measurement and optimization
//
// Usage: go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/forest6511/godl"
)

func main() {
	fmt.Println("=== Concurrent Download Examples ===")
	fmt.Println("Demonstrating high-performance concurrent downloads")
	fmt.Println()

	// Create context with generous timeout for large downloads
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Create examples directory
	examplesDir := "concurrent_downloads"
	if err := os.MkdirAll(examplesDir, 0o750); err != nil {
		log.Fatalf("Failed to create examples directory: %v", err)
	}
	defer cleanup(examplesDir)

	// Test URLs with different sizes for performance comparison
	testFiles := []struct {
		name        string
		url         string
		description string
	}{
		{
			name:        "small_file.bin",
			url:         "https://httpbin.org/bytes/51200", // 50KB
			description: "Small file (50KB) - minimal benefit from concurrency",
		},
		{
			name:        "medium_file.bin",
			url:         "https://httpbin.org/bytes/512000", // 500KB
			description: "Medium file (500KB) - moderate concurrency benefit",
		},
		{
			name:        "large_file.bin",
			url:         "https://httpbin.org/bytes/2097152", // 2MB
			description: "Large file (2MB) - significant concurrency benefit",
		},
	}

	// Example 1: Single-threaded downloads (baseline)
	fmt.Println("🔄 Example 1: Single-threaded Downloads (Baseline)")
	fmt.Println("Configuration: MaxConcurrency = 1")
	fmt.Println()

	singleThreadTimes := make(map[string]time.Duration)

	for _, testFile := range testFiles {
		fmt.Printf("📥 Downloading %s...\n", testFile.name)
		fmt.Printf("   %s\n", testFile.description)

		start := time.Now()

		opts := &godl.Options{
			MaxConcurrency:    1,         // Force single-threaded
			ChunkSize:         64 * 1024, // 64KB chunks
			OverwriteExisting: true,
		}

		destPath := filepath.Join(examplesDir, "single_"+testFile.name)
		_, err := godl.DownloadWithOptions(ctx, testFile.url, destPath, opts)

		elapsed := time.Since(start)
		singleThreadTimes[testFile.name] = elapsed

		if err != nil {
			log.Printf("❌ Failed to download %s: %v", testFile.name, err)
			continue
		}

		if info, err := os.Stat(destPath); err == nil {
			speed := float64(info.Size()) / elapsed.Seconds()
			fmt.Printf("✅ Downloaded in %v (%s, %.1f KB/s)\n",
				elapsed, formatBytes(info.Size()), speed/1024)
		}

		fmt.Println()
	}

	// Example 2: Multi-threaded downloads with different configurations
	fmt.Println("🚀 Example 2: Multi-threaded Downloads (Optimized)")
	fmt.Println()

	concurrentConfigs := []struct {
		name        string
		maxConns    int
		chunkSize   int64
		maxRate     int64
		description string
	}{
		{
			name:        "balanced",
			maxConns:    4,
			chunkSize:   128 * 1024, // 128KB
			maxRate:     0,          // Unlimited
			description: "Balanced configuration (4 connections, 128KB chunks)",
		},
		{
			name:        "rate_limited",
			maxConns:    4,
			chunkSize:   128 * 1024, // 128KB
			maxRate:     512 * 1024, // 512KB/s
			description: "Rate limited configuration (4 connections, 512KB/s limit)",
		},
		{
			name:        "aggressive",
			maxConns:    8,
			chunkSize:   256 * 1024, // 256KB
			maxRate:     0,          // Unlimited
			description: "Aggressive configuration (8 connections, 256KB chunks)",
		},
		{
			name:        "max_performance",
			maxConns:    16,
			chunkSize:   512 * 1024, // 512KB
			maxRate:     0,          // Unlimited
			description: "Maximum performance (16 connections, 512KB chunks)",
		},
	}

	for _, config := range concurrentConfigs {
		fmt.Printf("🔧 Configuration: %s\n", config.description)
		if config.maxRate > 0 {
			fmt.Printf("   MaxConcurrency: %d, ChunkSize: %s, MaxRate: %s/s\n",
				config.maxConns, formatBytes(config.chunkSize), formatBytes(config.maxRate))
		} else {
			fmt.Printf("   MaxConcurrency: %d, ChunkSize: %s, MaxRate: unlimited\n",
				config.maxConns, formatBytes(config.chunkSize))
		}
		fmt.Println()

		for _, testFile := range testFiles {
			fmt.Printf("📥 Downloading %s...\n", testFile.name)

			start := time.Now()

			opts := &godl.Options{
				MaxConcurrency:    config.maxConns,
				ChunkSize:         config.chunkSize,
				MaxRate:           config.maxRate,
				OverwriteExisting: true,
				UserAgent:         "godl-concurrent-example/1.0",
				Headers: map[string]string{
					"X-Download-Method": "concurrent",
				},
			}

			destPath := filepath.Join(examplesDir,
				fmt.Sprintf("%s_%s", config.name, testFile.name))

			_, err := godl.DownloadWithOptions(ctx, testFile.url, destPath, opts)
			elapsed := time.Since(start)

			if err != nil {
				log.Printf("❌ Failed to download %s with %s: %v",
					testFile.name, config.name, err)

				continue
			}

			if info, err := os.Stat(destPath); err == nil {
				speed := float64(info.Size()) / elapsed.Seconds()

				// Calculate improvement over single-threaded
				var improvement string

				if baseline, exists := singleThreadTimes[testFile.name]; exists && baseline > 0 {
					ratio := baseline.Seconds() / elapsed.Seconds()
					improvement = fmt.Sprintf(" (%.1fx faster)", ratio)
				}

				fmt.Printf("✅ Downloaded in %v (%s, %.1f KB/s)%s\n",
					elapsed, formatBytes(info.Size()), speed/1024, improvement)
			}
		}

		fmt.Println()
	}

	// Example 3: Custom configuration with optimal settings
	fmt.Println("⚙️ Example 3: Adaptive Configuration")
	fmt.Println("Automatically adjusting settings based on file size")
	fmt.Println()

	for _, testFile := range testFiles {
		fmt.Printf("📥 Downloading %s with adaptive settings...\n", testFile.name)

		// Get file info first to determine optimal settings
		fileInfo, err := godl.GetFileInfo(ctx, testFile.url)
		if err != nil {
			log.Printf("❌ Failed to get file info for %s: %v", testFile.name, err)
			continue
		}

		// Adaptive configuration based on file size
		var (
			maxConns  int
			chunkSize int64
		)

		switch {
		case fileInfo.Size < 100*1024: // < 100KB
			maxConns = 2
			chunkSize = 32 * 1024 // 32KB
		case fileInfo.Size < 1024*1024: // < 1MB
			maxConns = 4
			chunkSize = 64 * 1024 // 64KB
		default: // >= 1MB
			maxConns = 8
			chunkSize = 256 * 1024 // 256KB
		}

		fmt.Printf("   📊 File size: %s\n", formatBytes(fileInfo.Size))
		fmt.Printf("   🔧 Adaptive config: %d connections, %s chunks\n",
			maxConns, formatBytes(chunkSize))

		start := time.Now()

		opts := &godl.Options{
			MaxConcurrency:    maxConns,
			ChunkSize:         chunkSize,
			OverwriteExisting: true,
			EnableResume:      true, // Enable resume for robustness
			RetryAttempts:     3,    // Retry on failures
			UserAgent:         "godl-adaptive-example/1.0",
		}

		destPath := filepath.Join(examplesDir, "adaptive_"+testFile.name)
		_, err = godl.DownloadWithOptions(ctx, testFile.url, destPath, opts)
		elapsed := time.Since(start)

		if err != nil {
			log.Printf("❌ Failed to download %s with adaptive config: %v",
				testFile.name, err)

			continue
		}

		if info, err := os.Stat(destPath); err == nil {
			speed := float64(info.Size()) / elapsed.Seconds()
			fmt.Printf("✅ Downloaded in %v (%.1f KB/s)\n", elapsed, speed/1024)
		}

		fmt.Println()
	}

	// Performance Summary
	fmt.Println("📊 Performance Summary")
	fmt.Println("=====================")
	fmt.Println("Key takeaways:")
	fmt.Println("• Concurrent downloads provide significant speed improvements for larger files")
	fmt.Println("• Optimal configuration depends on file size and network conditions")
	fmt.Println("• Too many connections can sometimes hurt performance due to overhead")
	fmt.Println("• Adaptive configuration provides good balance across different scenarios")
	fmt.Println()

	fmt.Println("🎉 Concurrent download examples completed!")
	fmt.Printf("📁 Check the '%s' directory for all downloaded files.\n", examplesDir)
	fmt.Println("🧹 Files will be cleaned up automatically.")
}

// formatBytes formats a byte count in human-readable format.
func formatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}

	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}

	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// cleanup removes the examples directory and its contents.
func cleanup(dir string) {
	fmt.Printf("\n🧹 Cleaning up examples directory: %s\n", dir)

	if err := os.RemoveAll(dir); err != nil {
		log.Printf("Warning: Failed to clean up directory %s: %v", dir, err)
	}
}
