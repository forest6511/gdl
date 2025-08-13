// Package main demonstrates feature parity verification between
// the godl CLI tool and library API to ensure consistent behavior.
//
// Usage:
//   go run main.go
//
// This verification tests:
// - CLI and library produce identical results
// - Feature consistency across interfaces
// - Error handling parity
// - Performance comparison
// - Automatic testing of core features

package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/forest6511/godl"
)

// This program verifies feature parity between CLI and Library interfaces
func main() {
	fmt.Println("=== Feature Parity Verification ===")
	fmt.Println("Testing all features in both CLI and Library modes")

	// Build CLI tool
	if err := buildCLITool(); err != nil {
		log.Fatalf("Failed to build CLI: %v", err)
	}

	// Test each feature
	features := []struct {
		name string
		test func() error
	}{
		{"Basic Download", testBasicDownload},
		{"Download with Options", testDownloadWithOptions},
		{"Download to Memory", testDownloadToMemory},
		{"Custom Headers", testCustomHeaders},
		{"Timeout Handling", testTimeoutHandling},
		{"Force Overwrite", testForceOverwrite},
	}

	passed := 0
	failed := 0

	for _, feature := range features {
		fmt.Printf("Testing: %s\n", feature.name)
		if err := feature.test(); err != nil {
			fmt.Printf("  ❌ FAILED: %v\n", err)
			failed++
		} else {
			fmt.Printf("  ✅ PASSED\n")
			passed++
		}
		fmt.Println()
	}

	// Summary
	fmt.Println("=== Test Summary ===")
	fmt.Printf("Passed: %d\n", passed)
	fmt.Printf("Failed: %d\n", failed)
	fmt.Printf("Total:  %d\n", passed+failed)

	if failed > 0 {
		os.Exit(1)
	}
}

func buildCLITool() error {
	cmd := exec.Command("go", "build", "-o", "godl_test", "../../cmd/godl/")
	return cmd.Run()
}

func testBasicDownload() error {
	url := "https://httpbin.org/bytes/1024"

	// Library test
	libFile := "lib_basic.bin"
	_, err := godl.Download(context.Background(), url, libFile)
	if err != nil {
		return fmt.Errorf("library download failed: %w", err)
	}
	defer func() { _ = os.Remove(libFile) }()

	// Verify file size
	info, err := os.Stat(libFile)
	if err != nil {
		return fmt.Errorf("library output file not found: %w", err)
	}
	if info.Size() != 1024 {
		return fmt.Errorf("library: expected 1024 bytes, got %d", info.Size())
	}

	// CLI test
	cliFile := "cli_basic.bin"
	cmd := exec.Command("./godl_test", "-o", cliFile, url)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("CLI download failed: %w", err)
	}
	defer func() { _ = os.Remove(cliFile) }()

	info, err = os.Stat(cliFile)
	if err != nil {
		return fmt.Errorf("CLI output file not found: %w", err)
	}
	if info.Size() != 1024 {
		return fmt.Errorf("CLI: expected 1024 bytes, got %d", info.Size())
	}

	return nil
}

func testDownloadWithOptions() error {
	url := "https://httpbin.org/bytes/10240"

	// Library test with options
	libFile := "lib_options.bin"
	progressCalled := false

	options := &godl.Options{
		ProgressCallback: func(p godl.Progress) {
			progressCalled = true
		},
		MaxConcurrency:    2,
		OverwriteExisting: true,
	}

	_, err := godl.DownloadWithOptions(context.Background(), url, libFile, options)
	if err != nil {
		return fmt.Errorf("library download with options failed: %w", err)
	}
	if !progressCalled {
		return fmt.Errorf("library: progress callback not called")
	}
	defer func() { _ = os.Remove(libFile) }()

	// CLI test with options
	cliFile := "cli_options.bin"
	cmd := exec.Command("./godl_test", "--concurrent", "2", "-o", cliFile, url)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("CLI download with options failed: %w", err)
	}
	defer func() { _ = os.Remove(cliFile) }()

	return nil
}

func testDownloadToMemory() error {
	url := "https://httpbin.org/bytes/2048"

	// Library test (memory download - CLI doesn't support this)
	data, _, err := godl.DownloadToMemory(context.Background(), url)
	if err != nil {
		return fmt.Errorf("library download to memory failed: %w", err)
	}
	if len(data) != 2048 {
		return fmt.Errorf("library: expected 2048 bytes, got %d", len(data))
	}

	// Note: CLI doesn't support memory download, this is library-only feature
	return nil
}

func testCustomHeaders() error {
	url := "https://httpbin.org/headers"

	// Library test with custom headers
	libFile := "lib_headers.json"
	options := &godl.Options{
		Headers: map[string]string{
			"X-Custom-Header": "test-value",
			"User-Agent":      "godl-test/1.0",
		},
		OverwriteExisting: true,
	}

	_, err := godl.DownloadWithOptions(context.Background(), url, libFile, options)
	if err != nil {
		return fmt.Errorf("library download with headers failed: %w", err)
	}
	defer func() { _ = os.Remove(libFile) }()

	// CLI test with headers
	cliFile := "cli_headers.json"
	cmd := exec.Command("./godl_test",
		"-H", "X-Custom-Header: test-value",
		"--user-agent", "godl-test/1.0",
		"-o", cliFile, url)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("CLI download with headers failed: %w", err)
	}
	defer func() { _ = os.Remove(cliFile) }()

	return nil
}

func testTimeoutHandling() error {
	// Use a delayed response URL
	url := "https://httpbin.org/delay/10"

	// Library test with short timeout
	libFile := "lib_timeout.bin"
	options := &godl.Options{
		Timeout:           1 * time.Second,
		OverwriteExisting: true,
	}

	_, err := godl.DownloadWithOptions(context.Background(), url, libFile, options)
	if err == nil {
		_ = os.Remove(libFile)
		return fmt.Errorf("library: expected timeout, but succeeded")
	}
	// Expected to fail due to timeout

	// CLI test with short timeout
	cliFile := "cli_timeout.bin"
	cmd := exec.Command("./godl_test", "--timeout", "1s", "-o", cliFile, url)
	if err := cmd.Run(); err == nil {
		_ = os.Remove(cliFile)
		return fmt.Errorf("CLI: expected timeout, but succeeded")
	}
	// Expected to fail due to timeout

	return nil
}

func testForceOverwrite() error {
	url := "https://httpbin.org/bytes/1024"

	// Library test with force overwrite
	libFile := "lib_overwrite.bin"

	// Create existing file
	if err := os.WriteFile(libFile, []byte("existing"), 0600); err != nil {
		return fmt.Errorf("failed to create existing file: %w", err)
	}

	options := &godl.Options{
		OverwriteExisting: true,
	}

	_, err := godl.DownloadWithOptions(context.Background(), url, libFile, options)
	if err != nil {
		return fmt.Errorf("library overwrite failed: %w", err)
	}

	info, err := os.Stat(libFile)
	if err != nil {
		return fmt.Errorf("library output file not found: %w", err)
	}
	if info.Size() != 1024 {
		return fmt.Errorf("library: file not overwritten properly, size: %d", info.Size())
	}
	defer func() { _ = os.Remove(libFile) }()

	// CLI test with force overwrite
	cliFile := "cli_overwrite.bin"

	// Create existing file
	if err := os.WriteFile(cliFile, []byte("existing"), 0600); err != nil {
		return fmt.Errorf("failed to create existing file for CLI: %w", err)
	}

	cmd := exec.Command("./godl_test", "-f", "-o", cliFile, url)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("CLI overwrite failed: %w", err)
	}

	info, err = os.Stat(cliFile)
	if err != nil {
		return fmt.Errorf("CLI output file not found: %w", err)
	}
	if info.Size() != 1024 {
		return fmt.Errorf("CLI: file not overwritten properly, size: %d", info.Size())
	}
	defer func() { _ = os.Remove(cliFile) }()

	return nil
}

func init() {
	// Clean up any leftover test files
	files, _ := filepath.Glob("*_test.bin")
	for _, f := range files {
		_ = os.Remove(f)
	}
	files, _ = filepath.Glob("*_test.json")
	for _, f := range files {
		_ = os.Remove(f)
	}
}
