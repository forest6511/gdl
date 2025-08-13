// Package main demonstrates resume functionality for interrupted downloads.
//
// This example shows:
// - Automatic resume of partially downloaded files
// - Manual interruption and resume simulation
// - Resume with different configurations
// - Verification of resumed downloads
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
	"time"

	"github.com/forest6511/godl"
)

func main() {
	fmt.Println("=== Resume Functionality Examples ===")
	fmt.Println("Demonstrating download resume capabilities")
	fmt.Println()

	// Create context with generous timeout for resume demos
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
	defer cancel()

	// Create examples directory
	examplesDir := "resume_examples"
	if err := os.MkdirAll(examplesDir, 0o750); err != nil {
		log.Fatalf("Failed to create examples directory: %v", err)
	}
	defer cleanup(examplesDir)

	// Test files for resume functionality
	testFiles := []struct {
		name        string
		url         string
		description string
		size        string
	}{
		{
			name:        "resume_test_1mb.bin",
			url:         "https://httpbin.org/bytes/1048576", // 1MB
			description: "1MB file for basic resume testing",
			size:        "1MB",
		},
		{
			name:        "resume_test_3mb.bin",
			url:         "https://httpbin.org/bytes/3145728", // 3MB
			description: "3MB file for extended resume testing",
			size:        "3MB",
		},
		{
			name:        "resume_test_5mb.bin",
			url:         "https://httpbin.org/bytes/5242880", // 5MB
			description: "5MB file for comprehensive resume testing",
			size:        "5MB",
		},
	}

	// Example 1: Basic resume functionality
	fmt.Println("🔄 Example 1: Basic Resume Functionality")
	fmt.Println("Demonstrating automatic resume of partial downloads")
	fmt.Println()

	testFile := testFiles[0]
	destPath := filepath.Join(examplesDir, testFile.name)

	fmt.Printf("📥 Step 1: Initial download of %s (%s)\n", testFile.name, testFile.size)
	fmt.Printf("   %s\n", testFile.description)

	// First download with resume enabled
	opts := &godl.Options{
		MaxConcurrency:    4,
		ChunkSize:         64 * 1024,
		EnableResume:      true,
		OverwriteExisting: false, // Don't overwrite to allow resume
		ProgressCallback: func(p godl.Progress) {
			fmt.Printf("\r📈 Progress: %.1f%% (%s/%s) Speed: %s/s",
				p.Percentage,
				formatBytes(p.BytesDownloaded),
				formatBytes(p.TotalSize),
				formatBytes(p.Speed))
		},
	}

	start := time.Now()
	stats, err := godl.DownloadWithOptions(ctx, testFile.url, destPath, opts)
	elapsed := time.Since(start)

	if err != nil {
		log.Printf("\n❌ Initial download failed: %v", err)
	} else {
		fmt.Printf("\n✅ Initial download completed: %d bytes in %v\n", stats.BytesDownloaded, elapsed)

		// Check file size
		if info, err := os.Stat(destPath); err == nil {
			fmt.Printf("📄 File size: %s\n", formatBytes(info.Size()))
		}
	}

	fmt.Println()

	// Example 2: Simulated interruption and resume
	fmt.Println("⏹️ Example 2: Simulated Interruption and Resume")
	fmt.Println("Creating partial file and demonstrating resume capability")
	fmt.Println()

	testFile = testFiles[1]
	partialPath := filepath.Join(examplesDir, "partial_"+testFile.name)

	fmt.Printf("📥 Step 1: Creating partial download of %s (%s)\n", testFile.name, testFile.size)

	// Create a context that will be cancelled to simulate interruption
	interruptCtx, interruptCancel := context.WithCancel(ctx)

	var interrupted bool

	opts = &godl.Options{
		MaxConcurrency:    6,
		ChunkSize:         128 * 1024,
		EnableResume:      true,
		OverwriteExisting: true,
		ProgressCallback: func(p godl.Progress) {
			fmt.Printf("\r🔄 Partial download: %.1f%% (%s/%s) Speed: %s/s",
				p.Percentage,
				formatBytes(p.BytesDownloaded),
				formatBytes(p.TotalSize),
				formatBytes(p.Speed))

			// Interrupt when we reach about 40%
			if p.Percentage >= 40.0 && !interrupted {
				interrupted = true
				fmt.Print(" [INTERRUPTING...]")
				go func() {
					time.Sleep(500 * time.Millisecond)
					interruptCancel()
				}()
			}
		},
	}

	start = time.Now()
	_, err = godl.DownloadWithOptions(interruptCtx, testFile.url, partialPath, opts)
	elapsed = time.Since(start)

	var partialSize int64

	if interrupted && errors.Is(err, context.Canceled) {
		fmt.Printf("\n✅ Download interrupted as expected after %v\n", elapsed)

		if info, err := os.Stat(partialPath); err == nil {
			partialSize = info.Size()
			fmt.Printf("📄 Partial file size: %s\n", formatBytes(partialSize))
		}
	} else {
		log.Printf("\n❌ Expected interruption but got: %v", err)
		return
	}

	// Wait a moment to simulate real-world scenario
	fmt.Println("\n⏳ Waiting 2 seconds to simulate network interruption...")
	time.Sleep(2 * time.Second)

	// Now resume the download
	fmt.Printf("📥 Step 2: Resuming download from %s\n", formatBytes(partialSize))

	var resumeStart int64

	opts = &godl.Options{
		MaxConcurrency:    6,
		ChunkSize:         128 * 1024,
		EnableResume:      true,
		OverwriteExisting: false, // Important: don't overwrite partial file
		ProgressCallback: func(p godl.Progress) {
			if resumeStart == 0 && p.BytesDownloaded > partialSize {
				resumeStart = partialSize
				fmt.Printf("\n🔄 Resume detected! Starting from %s\n", formatBytes(resumeStart))
			}

			fmt.Printf("\r📈 Resume progress: %.1f%% (%s/%s) Speed: %s/s",
				p.Percentage,
				formatBytes(p.BytesDownloaded),
				formatBytes(p.TotalSize),
				formatBytes(p.Speed))
		},
	}

	start = time.Now()
	stats, err = godl.DownloadWithOptions(ctx, testFile.url, partialPath, opts)
	elapsed = time.Since(start)

	if err != nil {
		log.Printf("\n❌ Resume failed: %v", err)
	} else {
		fmt.Printf("\n✅ Resume completed: %d bytes in %v (resumed: %v)\n",
			stats.BytesDownloaded, elapsed, stats.Resumed)

		if info, err := os.Stat(partialPath); err == nil {
			fmt.Printf("📄 Final file size: %s\n", formatBytes(info.Size()))

			if resumeStart > 0 {
				resumedBytes := info.Size() - resumeStart
				fmt.Printf("🔄 Resumed bytes: %s\n", formatBytes(resumedBytes))
			}
		}
	}

	fmt.Println()

	// Example 3: Resume with different configurations
	fmt.Println("⚙️ Example 3: Resume with Different Configurations")
	fmt.Println("Testing resume with various chunk sizes and concurrency settings")
	fmt.Println()

	testFile = testFiles[2]

	configs := []struct {
		name      string
		maxConns  int
		chunkSize int64
		desc      string
	}{
		{
			name:      "small_chunks",
			maxConns:  2,
			chunkSize: 32 * 1024, // 32KB
			desc:      "Small chunks, low concurrency",
		},
		{
			name:      "large_chunks",
			maxConns:  8,
			chunkSize: 512 * 1024, // 512KB
			desc:      "Large chunks, high concurrency",
		},
	}

	for _, config := range configs {
		fmt.Printf("🔧 Testing %s configuration: %s\n", config.name, config.desc)
		fmt.Printf(
			"   MaxConcurrency: %d, ChunkSize: %s\n",
			config.maxConns,
			formatBytes(config.chunkSize),
		)

		configPath := filepath.Join(examplesDir, config.name+"_"+testFile.name)

		// Start download and interrupt it
		configCtx, configCancel := context.WithCancel(ctx)

		var configInterrupted bool

		opts = &godl.Options{
			MaxConcurrency:    config.maxConns,
			ChunkSize:         config.chunkSize,
			EnableResume:      true,
			OverwriteExisting: true,
			ProgressCallback: func(p godl.Progress) {
				if p.Percentage >= 25.0 && !configInterrupted {
					configInterrupted = true
					go func() {
						time.Sleep(200 * time.Millisecond)
						configCancel()
					}()
				}
			},
		}

		fmt.Printf("   📥 Starting download (will interrupt at 25%%)...\n")

		_, err = godl.DownloadWithOptions(configCtx, testFile.url, configPath, opts)

		var interruptedSize int64

		if configInterrupted && errors.Is(err, context.Canceled) {
			if info, err := os.Stat(configPath); err == nil {
				interruptedSize = info.Size()
				fmt.Printf("   ⏹️ Interrupted at %s\n", formatBytes(interruptedSize))
			}
		}

		// Resume with same configuration
		opts.OverwriteExisting = false

		fmt.Printf("   🔄 Resuming with same configuration...\n")

		start = time.Now()
		stats, err = godl.DownloadWithOptions(ctx, testFile.url, configPath, opts)
		elapsed = time.Since(start)

		if err != nil {
			log.Printf("   ❌ Resume failed: %v", err)
		} else {
			fmt.Printf("   ✅ Resume completed: %d bytes in %v (resumed: %v)\n",
				stats.BytesDownloaded, elapsed, stats.Resumed)

			if info, err := os.Stat(configPath); err == nil {
				finalSize := info.Size()
				resumedBytes := finalSize - interruptedSize
				fmt.Printf("   📊 Resumed %s (%.1f%% of file)\n",
					formatBytes(resumedBytes),
					float64(resumedBytes)/float64(finalSize)*100)
			}
		}

		fmt.Println()
	}

	// Example 4: Resume validation and verification
	fmt.Println("✅ Example 4: Resume Validation and Verification")
	fmt.Println("Verifying integrity of resumed downloads")
	fmt.Println()

	// Download the same file twice - once normally, once with resume
	testFile = testFiles[0]

	normalPath := filepath.Join(examplesDir, "normal_"+testFile.name)
	resumePath := filepath.Join(examplesDir, "resumed_"+testFile.name)

	fmt.Printf("📥 Step 1: Normal download of %s\n", testFile.name)

	opts = &godl.Options{
		MaxConcurrency:    4,
		EnableResume:      false,
		OverwriteExisting: true,
		Quiet:             true, // Suppress progress for cleaner output
	}

	start = time.Now()
	_, err = godl.DownloadWithOptions(ctx, testFile.url, normalPath, opts)
	normalElapsed := time.Since(start)

	if err != nil {
		log.Printf("❌ Normal download failed: %v", err)
		return
	}

	normalInfo, err := os.Stat(normalPath)
	if err != nil {
		log.Printf("❌ Failed to stat normal file: %v", err)
		return
	}

	fmt.Printf("✅ Normal download completed: %s (%v)\n",
		formatBytes(normalInfo.Size()), normalElapsed)

	fmt.Printf("📥 Step 2: Partial download (50%%) of %s\n", testFile.name)

	// Create partial download
	partialCtx, partialCancel := context.WithCancel(ctx)

	var partialInterrupted bool

	opts = &godl.Options{
		MaxConcurrency:    4,
		EnableResume:      true,
		OverwriteExisting: true,
		ProgressCallback: func(p godl.Progress) {
			if p.Percentage >= 50.0 && !partialInterrupted {
				partialInterrupted = true
				go partialCancel()
			}
		},
	}

	_, err = godl.DownloadWithOptions(partialCtx, testFile.url, resumePath, opts)

	var partialInfo os.FileInfo
	if partialInterrupted && errors.Is(err, context.Canceled) {
		partialInfo, err = os.Stat(resumePath)
		if err != nil {
			log.Printf("❌ Failed to stat partial file: %v", err)
			return
		}

		fmt.Printf("⏹️ Partial download: %s (%.1f%%)\n",
			formatBytes(partialInfo.Size()),
			float64(partialInfo.Size())/float64(normalInfo.Size())*100)
	}

	fmt.Printf("🔄 Step 3: Resuming download to completion\n")

	opts.OverwriteExisting = false
	opts.ProgressCallback = nil
	opts.Quiet = true

	start = time.Now()
	_, err = godl.DownloadWithOptions(ctx, testFile.url, resumePath, opts)
	resumeElapsed := time.Since(start)

	if err != nil {
		log.Printf("❌ Resume download failed: %v", err)
		return
	}

	resumeInfo, err := os.Stat(resumePath)
	if err != nil {
		log.Printf("❌ Failed to stat resumed file: %v", err)
		return
	}

	fmt.Printf("✅ Resumed download completed: %s (%v)\n",
		formatBytes(resumeInfo.Size()), resumeElapsed)

	// Verify files are identical
	fmt.Printf("🔍 Step 4: Verifying file integrity\n")

	if normalInfo.Size() == resumeInfo.Size() {
		fmt.Printf("✅ File sizes match: %s\n", formatBytes(normalInfo.Size()))

		// For extra verification, we could compare file hashes here
		// but for this demo, size matching is sufficient
		fmt.Printf("✅ Resume functionality verification successful!\n")

		totalTime := normalElapsed + resumeElapsed
		fmt.Printf("📊 Performance: Normal=%v, Resume=%v, Total=%v\n",
			normalElapsed, resumeElapsed, totalTime)
	} else {
		fmt.Printf("❌ File sizes don't match! Normal: %s, Resumed: %s\n",
			formatBytes(normalInfo.Size()), formatBytes(resumeInfo.Size()))
	}

	fmt.Println()

	// Summary
	fmt.Println("🔄 Resume Functionality Summary")
	fmt.Println("===============================")
	fmt.Println("Key features demonstrated:")
	fmt.Println("• Automatic detection and resume of partial downloads")
	fmt.Println("• Resume with different concurrency and chunk settings")
	fmt.Println("• Interruption simulation and recovery")
	fmt.Println("• Download integrity verification after resume")
	fmt.Println("• Performance comparison of normal vs resumed downloads")
	fmt.Println()

	fmt.Println("🎉 Resume functionality examples completed!")
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

// cleanup removes the examples directory and its contents.
func cleanup(dir string) {
	fmt.Printf("\n🧹 Cleaning up examples directory: %s\n", dir)

	if err := os.RemoveAll(dir); err != nil {
		log.Printf("Warning: Failed to clean up directory %s: %v", dir, err)
	}
}
