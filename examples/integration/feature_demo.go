// Package main provides a comprehensive demonstration of all gdl features
// This program shows both library and CLI integration working together
package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/forest6511/gdl"
)

func main() {
	fmt.Println("=== Godl Comprehensive Feature Demonstration ===")

	// Set up test server for demonstrations
	server := setupTestServer()
	defer server.Close()

	fmt.Printf("Test server running at: %s\n\n", server.URL)

	// Clean up any existing test files
	cleanup()
	defer cleanup()

	// Run library demonstrations
	fmt.Println("🔧 LIBRARY API DEMONSTRATIONS")
	fmt.Println("=" + strings.Repeat("=", 50))
	runLibraryDemos(server.URL)

	// Build CLI for demonstrations
	fmt.Println("\n🖥️  CLI DEMONSTRATIONS")
	fmt.Println("=" + strings.Repeat("=", 50))
	buildCLI()
	runCLIDemos(server.URL)

	// Feature integration tests
	fmt.Println("\n🔄 FEATURE INTEGRATION TESTS")
	fmt.Println("=" + strings.Repeat("=", 50))
	runIntegrationTests(server.URL)

	fmt.Println("\n✅ All demonstrations completed successfully!")
	fmt.Println("Check the generated files to see the results of each feature.")
}

// setupTestServer creates a test HTTP server with various endpoints.
func setupTestServer() *httptest.Server {
	mux := http.NewServeMux()

	// Basic file endpoint
	mux.HandleFunc("/file", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Accept-Ranges", "bytes")

		data := strings.Repeat("Hello, World! ", 100) // ~1.3KB file

		// Handle range requests
		if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
			w.Header().Set("Content-Range", fmt.Sprintf("bytes 0-%d/%d", len(data)-1, len(data)))
			w.WriteHeader(http.StatusPartialContent)
		}

		_, _ = w.Write([]byte(data))
	})

	// JSON endpoint
	mux.HandleFunc("/json", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = fmt.Fprintf(
			w,
			`{"message": "Hello from test server", "timestamp": "%s"}`,
			time.Now().Format(time.RFC3339),
		)
	})

	// Large file endpoint for concurrent testing
	mux.HandleFunc("/largefile", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Accept-Ranges", "bytes")

		size := 50 * 1024 // 50KB

		data := make([]byte, size)
		for i := range data {
			data[i] = byte(i % 256)
		}

		// Handle range requests for concurrent downloads
		if rangeHeader := r.Header.Get("Range"); rangeHeader != "" {
			var start, end int
			_, _ = fmt.Sscanf(rangeHeader, "bytes=%d-%d", &start, &end)

			if end >= len(data) {
				end = len(data) - 1
			}

			w.Header().Set("Content-Range", fmt.Sprintf("bytes %d-%d/%d", start, end, len(data)))
			w.WriteHeader(http.StatusPartialContent)
			_, _ = w.Write(data[start : end+1])

			return
		}

		_, _ = w.Write(data)
	})

	// Redirect endpoint
	mux.HandleFunc("/redirect", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/json", http.StatusMovedPermanently)
	})

	// Slow endpoint for timeout testing
	mux.HandleFunc("/slow", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(2 * time.Second)

		_, _ = w.Write([]byte("Slow response"))
	})

	return httptest.NewServer(mux)
}

// runLibraryDemos demonstrates all library API features.
func runLibraryDemos(baseURL string) {
	ctx := context.Background()

	fmt.Println("📚 Library API Features:")

	// 1. Basic download
	fmt.Println("  1. Basic Download...")

	_, err := gdl.Download(ctx, baseURL+"/file", "demo_basic.txt")
	checkResult("Basic Download", err)

	// 2. Download with options
	fmt.Println("  2. Download with Advanced Options...")

	opts := &gdl.Options{
		MaxConcurrency: 4,
		ChunkSize:      1024,
		Timeout:        30 * time.Second,
		UserAgent:      "GodlDemo/1.0",
		Headers: map[string]string{
			"X-Demo": "LibraryAPI",
		},
		EnableResume:      true,
		OverwriteExisting: true,
		CreateDirs:        true,
		ProgressCallback: func(p gdl.Progress) {
			if p.TotalSize > 0 && p.Percentage > 0 {
				fmt.Printf("    Progress: %.1f%% (%d/%d bytes)\n",
					p.Percentage, p.BytesDownloaded, p.TotalSize)
			}
		},
	}

	_ = os.MkdirAll("demo_downloads", 0o750)

	_, err = gdl.DownloadWithOptions(ctx, baseURL+"/largefile", "demo_downloads/advanced.bin", opts)
	checkResult("Advanced Options Download", err)

	// 3. Download to memory
	fmt.Println("  3. Download to Memory...")

	data, _, err := gdl.DownloadToMemory(ctx, baseURL+"/json")
	if err == nil {
		fmt.Printf("    Downloaded %d bytes to memory\n", len(data))
	}

	checkResult("Memory Download", err)

	// 4. Download to writer
	fmt.Println("  4. Download to Writer...")

	file, err := os.Create("demo_writer.json")
	if err == nil {
		defer func() { _ = file.Close() }()

		_, err = gdl.DownloadToWriter(ctx, baseURL+"/json", file)
	}

	checkResult("Writer Download", err)

	// 5. Resume download
	fmt.Println("  5. Resume Download...")

	_, err = gdl.DownloadWithResume(ctx, baseURL+"/largefile", "demo_resume.bin")
	checkResult("Resume Download", err)

	// 6. Get file info
	fmt.Println("  6. Get File Information...")

	info, err := gdl.GetFileInfo(ctx, baseURL+"/largefile")
	if err == nil && info != nil {
		fmt.Printf("    File size: %d bytes, Content-Type: %s, Ranges: %v\n",
			info.Size, info.ContentType, info.SupportsRanges)
	}

	checkResult("Get File Info", err)
}

// buildCLI builds the gdl CLI binary.
func buildCLI() {
	fmt.Println("🔨 Building CLI binary...")

	cmd := exec.Command("go", "build", "-o", "gdl", "./cmd/gdl")

	cmd.Dir = "."
	if err := cmd.Run(); err != nil {
		log.Fatalf("Failed to build CLI: %v", err)
	}

	fmt.Println("  ✓ CLI binary built successfully")
}

// runCLIDemos demonstrates CLI features.
func runCLIDemos(baseURL string) {
	fmt.Println("💻 CLI Features:")

	// Basic CLI usage
	fmt.Println("  1. Basic CLI Download...")
	runCLICommand("./gdl", "-o", "cli_basic.txt", baseURL+"/file")

	// Advanced headers
	fmt.Println("  2. CLI with Custom Headers...")
	runCLICommand("./gdl",
		"-H", "X-Demo: CLI",
		"-H", "Authorization: Bearer demo123",
		"-o", "cli_headers.json",
		baseURL+"/json")

	// Concurrent download
	fmt.Println("  3. CLI Concurrent Download...")
	runCLICommand("./gdl",
		"--concurrent", "4",
		"--chunk-size", "2KB",
		"--verbose",
		"-o", "cli_concurrent.bin",
		baseURL+"/largefile")

	// Progress bar variations
	fmt.Println("  4. CLI Progress Bar Types...")
	runCLICommand(
		"./gdl",
		"--progress-bar",
		"simple",
		"-o",
		"cli_progress_simple.bin",
		baseURL+"/file",
	)
	runCLICommand(
		"./gdl",
		"--progress-bar",
		"detailed",
		"-o",
		"cli_progress_detailed.bin",
		baseURL+"/file",
	)
	runCLICommand(
		"./gdl",
		"--progress-bar",
		"json",
		"-o",
		"cli_progress_json.bin",
		baseURL+"/file",
	)

	// Advanced features
	fmt.Println("  5. CLI Advanced Features...")
	runCLICommand("./gdl",
		"--max-redirects", "5",
		"--retry", "3",
		"--retry-delay", "1s",
		"--create-dirs",
		"--force",
		"--resume",
		"-o", "cli_advanced/test.json",
		baseURL+"/redirect")

	// Quiet and verbose modes
	fmt.Println("  6. CLI Output Modes...")
	runCLICommand("./gdl", "--quiet", "-o", "cli_quiet.txt", baseURL+"/file")
	runCLICommand("./gdl", "--verbose", "-o", "cli_verbose.json", baseURL+"/json")
}

// runIntegrationTests runs comprehensive integration tests.
func runIntegrationTests(baseURL string) {
	fmt.Println("🧪 Integration Tests:")

	// Test 1: Library + CLI consistency
	fmt.Println("  1. Library vs CLI Consistency Test...")

	ctx := context.Background()

	// Download same file with library and CLI
	_, err1 := gdl.Download(ctx, baseURL+"/file", "integration_lib.txt")
	runCLICommand("./gdl", "-o", "integration_cli.txt", baseURL+"/file")

	// Compare files
	lib_data, _ := os.ReadFile("integration_lib.txt")
	cli_data, _ := os.ReadFile("integration_cli.txt")

	if string(lib_data) == string(cli_data) {
		fmt.Println("    ✓ Library and CLI produce identical results")
	} else {
		fmt.Println("    ✗ Library and CLI results differ")
	}

	// Test 2: Feature compatibility
	fmt.Println("  2. Feature Compatibility Test...")

	// Library with all features
	opts := &gdl.Options{
		MaxConcurrency:    2,
		ChunkSize:         1024,
		UserAgent:         "Integration/1.0",
		Headers:           map[string]string{"X-Test": "Integration"},
		EnableResume:      true,
		OverwriteExisting: true,
		CreateDirs:        true,
	}

	_ = os.MkdirAll("integration_test", 0o750)

	_, err2 := gdl.DownloadWithOptions(
		ctx,
		baseURL+"/largefile",
		"integration_test/lib_full.bin",
		opts,
	)

	// CLI with equivalent features
	runCLICommand("./gdl",
		"--concurrent", "2",
		"--chunk-size", "1KB",
		"--user-agent", "Integration/1.0",
		"-H", "X-Test: Integration",
		"--resume",
		"--force",
		"--create-dirs",
		"-o", "integration_test/cli_full.bin",
		baseURL+"/largefile")

	// Verify both succeeded
	if err1 == nil && err2 == nil {
		if _, err := os.Stat("integration_test/cli_full.bin"); err == nil {
			fmt.Println("    ✓ All features work in both library and CLI")
		}
	}

	// Test 3: Error handling consistency
	fmt.Println("  3. Error Handling Test...")

	// Test with invalid URL
	_, err_lib := gdl.Download(ctx, "http://invalid-url-test.local/file", "error_test.txt")
	cmd_err := exec.Command(
		"./gdl",
		"-o",
		"error_cli_test.txt",
		"http://invalid-url-test.local/file",
	)
	err_cli := cmd_err.Run()

	if err_lib != nil && err_cli != nil {
		fmt.Println("    ✓ Both library and CLI handle errors appropriately")
	}
}

// runCLICommand runs a CLI command and handles errors.
func runCLICommand(args ...string) {
	// #nosec G204 - This is a demo with controlled command arguments
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = "."

	output, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("    Command failed: %s\n", strings.Join(args, " "))
		fmt.Printf("    Error: %v\n", err)

		if len(output) > 0 {
			fmt.Printf("    Output: %s\n", string(output))
		}
	} else {
		fmt.Printf("    ✓ Command succeeded: %s\n", filepath.Base(args[len(args)-1]))
	}
}

// checkResult checks and reports the result of an operation.
func checkResult(operation string, err error) {
	if err != nil {
		fmt.Printf("    ✗ %s failed: %v\n", operation, err)
	} else {
		fmt.Printf("    ✓ %s succeeded\n", operation)
	}
}

// cleanup removes test files.
func cleanup() {
	patterns := []string{
		"demo_*.txt", "demo_*.json", "demo_*.bin",
		"cli_*.txt", "cli_*.json", "cli_*.bin",
		"integration_*.txt", "integration_*.bin",
		"error_*.txt",
		"gdl", // CLI binary
	}

	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			_ = os.Remove(match)
		}
	}

	// Remove directories
	_ = os.RemoveAll("demo_downloads")
	_ = os.RemoveAll("cli_advanced")
	_ = os.RemoveAll("integration_test")
}
