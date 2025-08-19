// Package main demonstrates advanced progress tracking functionality.
//
// This example shows:
// - Real-time progress callbacks with detailed metrics
// - Custom progress display formats (percentage, speed, ETA)
// - Progress tracking for different download scenarios
// - Interactive progress updates and cancellation
//
// Usage: go run main.go
package main

import (
	"context"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/forest6511/gdl"
)

func main() {
	fmt.Println("=== Progress Tracking Examples ===")
	fmt.Println("Demonstrating advanced progress tracking and monitoring")
	fmt.Println()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Create examples directory
	examplesDir := "progress_examples"
	if err := os.MkdirAll(examplesDir, 0o750); err != nil {
		log.Fatalf("Failed to create examples directory: %v", err)
	}
	defer cleanup(examplesDir)

	// Test files for different progress tracking scenarios
	testFiles := []struct {
		name        string
		url         string
		description string
		size        string
	}{
		{
			name:        "small_progress.bin",
			url:         "https://httpbin.org/bytes/102400", // 100KB
			description: "Small file - rapid progress updates",
			size:        "100KB",
		},
		{
			name:        "medium_progress.bin",
			url:         "https://httpbin.org/bytes/1048576", // 1MB
			description: "Medium file - detailed progress tracking",
			size:        "1MB",
		},
		{
			name:        "large_progress.bin",
			url:         "https://httpbin.org/bytes/5242880", // 5MB
			description: "Large file - comprehensive progress metrics",
			size:        "5MB",
		},
	}

	// Example 1: Basic progress tracking
	fmt.Println("📊 Example 1: Basic Progress Tracking")
	fmt.Println("Simple progress callbacks with essential metrics")
	fmt.Println()

	testFile := testFiles[0]
	fmt.Printf("📥 Downloading %s (%s)...\n", testFile.name, testFile.size)
	fmt.Printf("   %s\n", testFile.description)

	destPath := filepath.Join(examplesDir, "basic_"+testFile.name)

	opts := &gdl.Options{
		MaxConcurrency:    4,
		ChunkSize:         64 * 1024, // 64KB
		OverwriteExisting: true,
		ProgressCallback: func(p gdl.Progress) {
			fmt.Printf("\r📈 Progress: %.1f%% (%s/%s) Speed: %s/s ETA: %s",
				p.Percentage,
				formatBytes(p.BytesDownloaded),
				formatBytes(p.TotalSize),
				formatBytes(p.Speed),
				formatDuration(p.TimeRemaining))
		},
	}

	start := time.Now()
	stats, err := gdl.DownloadWithOptions(ctx, testFile.url, destPath, opts)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("\n❌ Example 1 failed: %v", err)
	} else {
		fmt.Printf("\n✅ Downloaded %d bytes in %v (avg speed: %.2f KB/s)\n",
			stats.BytesDownloaded, elapsed, float64(stats.AverageSpeed)/1024)
	}

	fmt.Println()

	// Example 2: Detailed progress tracking with statistics
	fmt.Println("📊 Example 2: Detailed Progress Statistics")
	fmt.Println("Comprehensive progress metrics and statistics")
	fmt.Println()

	testFile = testFiles[1]
	fmt.Printf("📥 Downloading %s (%s)...\n", testFile.name, testFile.size)

	var (
		progressUpdates int64
		maxSpeed        int64
	)

	minSpeed := int64(^uint64(0) >> 1) // max int64

	var totalSpeed int64

	destPath = filepath.Join(examplesDir, "detailed_"+testFile.name)

	opts = &gdl.Options{
		MaxConcurrency:    6,
		ChunkSize:         128 * 1024, // 128KB
		OverwriteExisting: true,
		ProgressCallback: func(p gdl.Progress) {
			updates := atomic.AddInt64(&progressUpdates, 1)

			// Track speed statistics
			if p.Speed > atomic.LoadInt64(&maxSpeed) {
				atomic.StoreInt64(&maxSpeed, p.Speed)
			}
			if p.Speed < atomic.LoadInt64(&minSpeed) && p.Speed > 0 {
				atomic.StoreInt64(&minSpeed, p.Speed)
			}
			atomic.AddInt64(&totalSpeed, p.Speed)

			// Display progress with statistics
			avgSpeed := atomic.LoadInt64(&totalSpeed) / updates

			fmt.Printf(
				"\r📊 %.1f%% | %s/%s | Speed: %s/s (avg: %s/s, max: %s/s) | ETA: %s | Updates: %d",
				p.Percentage,
				formatBytes(p.BytesDownloaded),
				formatBytes(p.TotalSize),
				formatBytes(p.Speed),
				formatBytes(avgSpeed),
				formatBytes(atomic.LoadInt64(&maxSpeed)),
				formatDuration(p.TimeRemaining),
				updates,
			)
		},
	}

	start = time.Now()
	stats, err = gdl.DownloadWithOptions(ctx, testFile.url, destPath, opts)
	elapsed = time.Since(start)

	if err != nil {
		log.Printf("\n❌ Example 2 failed: %v", err)
	} else {
		fmt.Printf("\n✅ Download completed: %d bytes in %v\n", stats.BytesDownloaded, elapsed)
		fmt.Printf("📈 Final Statistics: %d updates, reported avg speed: %s/s, tracked max speed: %s/s\n",
			progressUpdates,
			formatBytes(stats.AverageSpeed),
			formatBytes(maxSpeed))
	}

	fmt.Println()

	// Example 3: Progress tracking with visual indicators
	fmt.Println("🎨 Example 3: Visual Progress Indicators")
	fmt.Println("Progress tracking with visual progress bar and indicators")
	fmt.Println()

	testFile = testFiles[2]
	fmt.Printf("📥 Downloading %s (%s)...\n", testFile.name, testFile.size)

	destPath = filepath.Join(examplesDir, "visual_"+testFile.name)

	opts = &gdl.Options{
		MaxConcurrency:    8,
		ChunkSize:         256 * 1024, // 256KB
		OverwriteExisting: true,
		ProgressCallback: func(p gdl.Progress) {
			progressBar := createProgressBar(p.Percentage, 40)

			fmt.Printf("\r%s %.1f%% [%s] %s/%s (%s/s) ETA:%s",
				getSpinner(p.BytesDownloaded),
				p.Percentage,
				progressBar,
				formatBytes(p.BytesDownloaded),
				formatBytes(p.TotalSize),
				formatBytes(p.Speed),
				formatDuration(p.TimeRemaining))
		},
	}

	start = time.Now()
	stats, err = gdl.DownloadWithOptions(ctx, testFile.url, destPath, opts)
	elapsed = time.Since(start)

	if err != nil {
		log.Printf("\n❌ Example 3 failed: %v", err)
	} else {
		fmt.Printf("\n✅ Download completed: %d bytes in %v with visual progress tracking\n",
			stats.BytesDownloaded, elapsed)
	}

	fmt.Println()

	// Example 4: Progress tracking with cancellation
	fmt.Println("⏹️ Example 4: Progress with Cancellation Demo")
	fmt.Println("Demonstrating progress tracking with controlled cancellation")
	fmt.Println()

	testFile = testFiles[2]
	fmt.Printf("📥 Starting %s download (will cancel at 30%%)...\n", testFile.name)

	cancelCtx, cancelFunc := context.WithCancel(ctx)
	destPath = filepath.Join(examplesDir, "cancelled_"+testFile.name)

	var cancelled bool

	opts = &gdl.Options{
		MaxConcurrency:    4,
		ChunkSize:         128 * 1024,
		OverwriteExisting: true,
		ProgressCallback: func(p gdl.Progress) {
			fmt.Printf("\r🔄 Cancellation demo: %.1f%% (%s/%s) Speed: %s/s",
				p.Percentage,
				formatBytes(p.BytesDownloaded),
				formatBytes(p.TotalSize),
				formatBytes(p.Speed))

			// Cancel when we reach 30%
			if p.Percentage >= 30.0 && !cancelled {
				cancelled = true
				fmt.Print(" [CANCELLING...]")
				go func() {
					time.Sleep(500 * time.Millisecond)
					cancelFunc()
				}()
			}
		},
	}

	start = time.Now()
	stats, err = gdl.DownloadWithOptions(cancelCtx, testFile.url, destPath, opts)
	elapsed = time.Since(start)

	if err != nil {
		if cancelled && errors.Is(err, context.Canceled) {
			fmt.Printf("\n✅ Download cancelled successfully after %v (partial: %d bytes)\n",
				elapsed, func() int64 {
					if stats != nil {
						return stats.BytesDownloaded
					} else {
						return 0
					}
				}())
		} else {
			log.Printf("\n❌ Example 4 failed: %v", err)
		}
	} else {
		fmt.Printf("\n⚠️ Expected cancellation but download completed\n")
	}

	fmt.Println()

	// Example 5: Multi-stage progress tracking
	fmt.Println("🎯 Example 5: Multi-Stage Progress Tracking")
	fmt.Println("Progress tracking across multiple download stages")
	fmt.Println()

	for i, testFile := range testFiles[:2] {
		stage := i + 1
		fmt.Printf("📥 Stage %d/%d: %s (%s)\n", stage, 2, testFile.name, testFile.size)

		destPath = filepath.Join(examplesDir, fmt.Sprintf("stage%d_%s", stage, testFile.name))

		stageStart := time.Now()

		opts = &gdl.Options{
			MaxConcurrency:    4,
			ChunkSize:         64 * 1024,
			OverwriteExisting: true,
			ProgressCallback: func(p gdl.Progress) {
				stageElapsed := time.Since(stageStart)

				fmt.Printf("\r🎯 Stage %d: %.1f%% | %s/%s | %s/s | Elapsed: %v | ETA: %v",
					stage,
					p.Percentage,
					formatBytes(p.BytesDownloaded),
					formatBytes(p.TotalSize),
					formatBytes(p.Speed),
					stageElapsed.Round(time.Second),
					p.TimeRemaining.Round(time.Second))
			},
		}

		stats, err := gdl.DownloadWithOptions(ctx, testFile.url, destPath, opts)
		if err != nil {
			log.Printf("\n❌ Stage %d failed: %v", stage, err)
			break
		} else {
			stageElapsed := time.Since(stageStart)
			fmt.Printf("\n✅ Stage %d completed: %d bytes in %v\n", stage, stats.BytesDownloaded, stageElapsed)
		}
	}

	fmt.Println()

	// Summary
	fmt.Println("📊 Progress Tracking Summary")
	fmt.Println("============================")
	fmt.Println("Key features demonstrated:")
	fmt.Println("• Real-time progress callbacks with detailed metrics")
	fmt.Println("• Speed tracking (current, average, maximum)")
	fmt.Println("• Visual progress indicators and progress bars")
	fmt.Println("• ETA calculation and time tracking")
	fmt.Println("• Progress-based cancellation control")
	fmt.Println("• Multi-stage progress monitoring")
	fmt.Println()

	fmt.Println("🎉 Progress tracking examples completed!")
	fmt.Printf("📁 Check the '%s' directory for all downloaded files.\n", examplesDir)
	fmt.Println("🧹 Files will be cleaned up automatically.")
}

// formatBytes formats a byte count in human-readable format.
func formatBytes(bytes int64) string {
	if bytes == 0 {
		return "0 B"
	}

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

// formatDuration formats a duration in human-readable format.
func formatDuration(d time.Duration) string {
	if d <= 0 {
		return "0s"
	}

	d = d.Round(time.Second)

	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	} else if d < time.Hour {
		min := int(d.Minutes())

		sec := int(d.Seconds()) % 60
		if sec == 0 {
			return fmt.Sprintf("%dm", min)
		}

		return fmt.Sprintf("%dm%ds", min, sec)
	} else {
		hour := int(d.Hours())

		min := int(d.Minutes()) % 60
		if min == 0 {
			return fmt.Sprintf("%dh", hour)
		}

		return fmt.Sprintf("%dh%dm", hour, min)
	}
}

// createProgressBar creates a visual progress bar.
func createProgressBar(percentage float64, width int) string {
	if width <= 0 {
		width = 20
	}

	filled := int(percentage * float64(width) / 100.0)
	if filled > width {
		filled = width
	}

	bar := strings.Repeat("█", filled) + strings.Repeat("░", width-filled)

	return bar
}

// getSpinner returns a spinning character based on the current progress.
func getSpinner(bytes int64) string {
	spinners := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
	index := int(bytes/10240) % len(spinners) // Change every 10KB

	return spinners[index]
}

// cleanup removes the examples directory and its contents.
func cleanup(dir string) {
	fmt.Printf("\n🧹 Cleaning up examples directory: %s\n", dir)

	if err := os.RemoveAll(dir); err != nil {
		log.Printf("Warning: Failed to clean up directory %s: %v", dir, err)
	}
}
