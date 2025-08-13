// Package main demonstrates basic download functionality using the godl library.
//
// This example shows:
// - Simple download using the basic Download function
// - Download with custom destination
// - Basic error handling
// - File verification after download
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
	fmt.Println("=== Basic Download Examples ===")
	fmt.Println("Demonstrating simple file downloads using the godl library")
	fmt.Println()

	// Create a context with timeout for all downloads
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	// Create examples directory
	examplesDir := "downloads"
	if err := os.MkdirAll(examplesDir, 0o750); err != nil {
		log.Fatalf("Failed to create examples directory: %v", err)
	}
	defer cleanup(examplesDir)

	// Example 1: Simple download using default filename
	fmt.Println("📥 Example 1: Simple Download")
	fmt.Println("URL: https://httpbin.org/json")
	fmt.Println("Destination: auto-detected filename")

	// Use a temporary file for this example
	tempFile := filepath.Join(examplesDir, "httpbin_response.json")

	stats, err := godl.Download(ctx, "https://httpbin.org/json", tempFile)
	if err != nil {
		log.Printf("❌ Example 1 failed: %v", err)
	} else {
		// Verify the download
		if info, err := os.Stat(tempFile); err == nil {
			fmt.Printf("✅ Download completed: %s (%d bytes in %v)\n", tempFile, info.Size(), stats.Duration)

			// Show file content preview
			// #nosec G304 - This is a demo file with controlled input
			if content, err := os.ReadFile(tempFile); err == nil && len(content) > 0 {
				preview := string(content)
				if len(preview) > 100 {
					preview = preview[:100] + "..."
				}

				fmt.Printf("📄 Content preview: %s\n", preview)
			}
		} else {
			fmt.Printf("⚠️ Download may have failed: %v\n", err)
		}
	}

	fmt.Println()

	// Example 2: Download with custom destination
	fmt.Println("📥 Example 2: Custom Destination Download")
	fmt.Println("URL: https://httpbin.org/bytes/1024")
	fmt.Println("Destination: custom filename")

	customFile := filepath.Join(examplesDir, "sample_1kb.bin")

	stats, err = godl.Download(ctx, "https://httpbin.org/bytes/1024", customFile)
	if err != nil {
		log.Printf("❌ Example 2 failed: %v", err)
	} else {
		if info, err := os.Stat(customFile); err == nil {
			fmt.Printf("✅ Download completed: %s (%d bytes, avg speed: %.2f KB/s)\n",
				customFile, info.Size(), float64(stats.AverageSpeed)/1024)
		} else {
			fmt.Printf("⚠️ Download may have failed: %v\n", err)
		}
	}

	fmt.Println()

	// Example 3: Download to memory and save
	fmt.Println("💾 Example 3: Download to Memory")
	fmt.Println("URL: https://httpbin.org/uuid")
	fmt.Println("Method: Download to memory, then save to file")

	data, stats, err := godl.DownloadToMemory(ctx, "https://httpbin.org/uuid")
	if err != nil {
		log.Printf("❌ Example 3 failed: %v", err)
	} else {
		fmt.Printf("✅ Downloaded %d bytes to memory in %v\n", len(data), stats.Duration)
		fmt.Printf("📄 Content: %s\n", string(data))

		// Save to file
		memoryFile := filepath.Join(examplesDir, "uuid_from_memory.json")
		if err := os.WriteFile(memoryFile, data, 0o600); err != nil {
			log.Printf("⚠️ Failed to save memory data: %v", err)
		} else {
			fmt.Printf("💾 Saved to file: %s\n", memoryFile)
		}
	}

	fmt.Println()

	// Example 4: Get file information without downloading
	fmt.Println("ℹ️ Example 4: File Information")
	fmt.Println("URL: https://httpbin.org/bytes/2048")
	fmt.Println("Method: Get file info without downloading")

	info, err := godl.GetFileInfo(ctx, "https://httpbin.org/bytes/2048")
	if err != nil {
		log.Printf("❌ Example 4 failed: %v", err)
	} else {
		fmt.Printf("✅ File information retrieved:\n")
		fmt.Printf("  📏 Size: %d bytes\n", info.Size)
		fmt.Printf("  📄 Content-Type: %s\n", info.ContentType)
		fmt.Printf("  🔗 Supports Ranges: %v\n", info.SupportsRanges)

		if !info.LastModified.IsZero() {
			fmt.Printf("  📅 Last Modified: %s\n", info.LastModified.Format(time.RFC3339))
		}
	}

	fmt.Println()

	// Example 5: Error handling demonstration
	fmt.Println("🚨 Example 5: Error Handling")
	fmt.Println("URL: http://nonexistent-domain-12345.invalid/file.txt")
	fmt.Println("Purpose: Demonstrate error handling")

	_, err = godl.Download(ctx, "http://nonexistent-domain-12345.invalid/file.txt",
		filepath.Join(examplesDir, "should_fail.txt"))
	if err != nil {
		fmt.Printf("✅ Error handled correctly: %v\n", err)
	} else {
		fmt.Println("⚠️ Expected an error but download succeeded")
	}

	fmt.Println()

	fmt.Println("🎉 Basic download examples completed!")
	fmt.Printf("📁 Check the '%s' directory for downloaded files.\n", examplesDir)
	fmt.Println("🧹 Files will be cleaned up automatically.")
}

// cleanup removes the examples directory and its contents.
func cleanup(dir string) {
	fmt.Printf("\n🧹 Cleaning up examples directory: %s\n", dir)

	if err := os.RemoveAll(dir); err != nil {
		log.Printf("Warning: Failed to clean up directory %s: %v", dir, err)
	}
}
