// Package main demonstrates comprehensive error handling capabilities.
//
// This example shows:
// - Different types of download errors and how to handle them
// - Retry strategies and automatic error recovery
// - Custom error handling and user-friendly error messages
// - Network error scenarios and troubleshooting
//
// Usage: go run main.go
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/forest6511/gdl"
)

func main() {
	fmt.Println("=== Error Handling Examples ===")
	fmt.Println("Demonstrating comprehensive error handling and recovery")
	fmt.Println()

	// Create context with timeout for error demos
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// Create examples directory
	examplesDir := "error_examples"
	if err := os.MkdirAll(examplesDir, 0o750); err != nil {
		log.Fatalf("Failed to create examples directory: %v", err)
	}
	defer cleanup(examplesDir)

	// Example 1: Handling network errors
	fmt.Println("🌐 Example 1: Network Error Handling")
	fmt.Println("Demonstrating handling of various network-related errors")
	fmt.Println()

	networkErrorCases := []struct {
		name        string
		url         string
		description string
		expectError bool
	}{
		{
			name:        "dns_failure",
			url:         "https://nonexistent-domain-12345.invalid/file.zip",
			description: "DNS resolution failure",
			expectError: true,
		},
		{
			name:        "connection_refused",
			url:         "http://127.0.0.1:99999/file.zip",
			description: "Connection refused (invalid port)",
			expectError: true,
		},
		{
			name:        "timeout_simulation",
			url:         "https://httpbin.org/delay/60", // Will timeout
			description: "Request timeout simulation",
			expectError: true,
		},
		{
			name:        "valid_url",
			url:         "https://httpbin.org/json",
			description: "Valid URL for comparison",
			expectError: false,
		},
	}

	for i, testCase := range networkErrorCases {
		fmt.Printf("%d. Testing %s: %s\n", i+1, testCase.name, testCase.description)
		fmt.Printf("   URL: %s\n", testCase.url)

		destPath := filepath.Join(examplesDir, testCase.name+".dat")

		// Use short timeout for timeout test
		testCtx := ctx

		if strings.Contains(testCase.name, "timeout") {
			var timeoutCancel context.CancelFunc

			testCtx, timeoutCancel = context.WithTimeout(ctx, 5*time.Second)
			defer timeoutCancel()
		}

		start := time.Now()
		_, err := gdl.Download(testCtx, testCase.url, destPath)
		elapsed := time.Since(start)

		if testCase.expectError {
			if err != nil {
				fmt.Printf("   ✅ Expected error handled correctly: %v\n", err)
				fmt.Printf("   ⏱️  Failed after: %v\n", elapsed)
				analyzeError(err)
			} else {
				fmt.Printf("   ⚠️  Expected error but download succeeded\n")
			}
		} else {
			if err != nil {
				fmt.Printf("   ❌ Unexpected error: %v\n", err)
			} else {
				fmt.Printf("   ✅ Valid download succeeded in %v\n", elapsed)
			}
		}

		fmt.Println()
	}

	// Example 2: HTTP status code error handling
	fmt.Println("🔧 Example 2: HTTP Status Code Error Handling")
	fmt.Println("Handling various HTTP error responses")
	fmt.Println()

	httpErrorCases := []struct {
		name        string
		url         string
		description string
		statusCode  int
	}{
		{
			name:        "not_found",
			url:         "https://httpbin.org/status/404",
			description: "404 Not Found",
			statusCode:  404,
		},
		{
			name:        "unauthorized",
			url:         "https://httpbin.org/status/401",
			description: "401 Unauthorized",
			statusCode:  401,
		},
		{
			name:        "forbidden",
			url:         "https://httpbin.org/status/403",
			description: "403 Forbidden",
			statusCode:  403,
		},
		{
			name:        "server_error",
			url:         "https://httpbin.org/status/500",
			description: "500 Internal Server Error",
			statusCode:  500,
		},
		{
			name:        "service_unavailable",
			url:         "https://httpbin.org/status/503",
			description: "503 Service Unavailable",
			statusCode:  503,
		},
	}

	for i, testCase := range httpErrorCases {
		fmt.Printf("%d. Testing %s: %s\n", i+1, testCase.name, testCase.description)
		fmt.Printf("   URL: %s\n", testCase.url)

		destPath := filepath.Join(examplesDir, testCase.name+".dat")

		start := time.Now()
		_, err := gdl.Download(ctx, testCase.url, destPath)
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("   ✅ HTTP error handled: %v\n", err)
			fmt.Printf("   ⏱️  Failed after: %v\n", elapsed)

			// Provide user-friendly suggestions
			provideSuggestions(testCase.statusCode, err)
		} else {
			fmt.Printf("   ⚠️  Expected HTTP error but download succeeded\n")
		}

		fmt.Println()
	}

	// Example 3: Retry strategies and error recovery
	fmt.Println("🔄 Example 3: Retry Strategies and Error Recovery")
	fmt.Println("Demonstrating automatic retry and error recovery")
	fmt.Println()

	// Simulate an unreliable server using httpbin's random status endpoint
	retryTestCases := []struct {
		name        string
		url         string
		description string
		retryCount  int
		retryDelay  time.Duration
	}{
		{
			name:        "basic_retry",
			url:         "https://httpbin.org/status/200,500", // 50% chance of success
			description: "Basic retry with 50% success rate",
			retryCount:  3,
			retryDelay:  1 * time.Second,
		},
		{
			name:        "aggressive_retry",
			url:         "https://httpbin.org/status/200,500,502,503", // 25% chance of success
			description: "Aggressive retry with 25% success rate",
			retryCount:  8,
			retryDelay:  500 * time.Millisecond,
		},
	}

	for i, testCase := range retryTestCases {
		fmt.Printf("%d. Testing %s: %s\n", i+1, testCase.name, testCase.description)
		fmt.Printf("   URL: %s\n", testCase.url)
		fmt.Printf(
			"   Retry config: %d attempts, %v delay\n",
			testCase.retryCount,
			testCase.retryDelay,
		)

		destPath := filepath.Join(examplesDir, testCase.name+".json")

		var attempts int

		opts := &gdl.Options{
			RetryAttempts: testCase.retryCount,
			Timeout:       30 * time.Second,
			ProgressCallback: func(p gdl.Progress) {
				// Track retry attempts (simplified)
				if p.BytesDownloaded == 0 && p.TotalSize > 0 {
					attempts++
					if attempts > 1 {
						fmt.Printf("   🔄 Retry attempt %d...\n", attempts-1)
					}
				}
			},
		}

		start := time.Now()
		_, err := gdl.DownloadWithOptions(ctx, testCase.url, destPath, opts)
		elapsed := time.Since(start)

		if err != nil {
			fmt.Printf("   ❌ Failed after %d attempts: %v\n", attempts, err)
			fmt.Printf("   ⏱️  Total time: %v\n", elapsed)
		} else {
			fmt.Printf("   ✅ Succeeded after %d attempts in %v\n", attempts, elapsed)
		}

		fmt.Println()
	}

	// Example 4: File system error handling
	fmt.Println("📁 Example 4: File System Error Handling")
	fmt.Println("Handling file system related errors")
	fmt.Println()

	fsErrorCases := []struct {
		name        string
		destPath    string
		description string
		setup       func(string) error
		expectError bool
	}{
		{
			name:        "readonly_directory",
			destPath:    filepath.Join(examplesDir, "readonly", "file.dat"),
			description: "Read-only directory",
			setup: func(path string) error {
				dir := filepath.Dir(path)
				if err := os.MkdirAll(dir, 0o750); err != nil {
					return err
				}
				// #nosec G302 - Intentionally setting read-only for error testing
				return os.Chmod(dir, 0o444) // Read-only
			},
			expectError: true,
		},
		{
			name:        "insufficient_space_simulation",
			destPath:    filepath.Join(examplesDir, "large_file.dat"),
			description: "Simulated insufficient disk space",
			setup:       nil,   // No setup needed
			expectError: false, // We'll simulate this differently
		},
		{
			name:        "invalid_filename",
			destPath:    filepath.Join(examplesDir, "invalid\x00name.dat"), // Null character
			description: "Invalid filename characters",
			setup:       nil,
			expectError: true,
		},
	}

	for i, testCase := range fsErrorCases[:2] { // Skip invalid filename on some systems
		fmt.Printf("%d. Testing %s: %s\n", i+1, testCase.name, testCase.description)
		fmt.Printf("   Destination: %s\n", testCase.destPath)

		// Setup error condition
		if testCase.setup != nil {
			if err := testCase.setup(testCase.destPath); err != nil {
				fmt.Printf("   ⚠️ Setup failed: %v\n", err)
				continue
			}
		}

		start := time.Now()
		_, err := gdl.Download(ctx, "https://httpbin.org/json", testCase.destPath)
		elapsed := time.Since(start)

		if testCase.expectError {
			if err != nil {
				fmt.Printf("   ✅ File system error handled: %v\n", err)
				fmt.Printf("   ⏱️  Failed after: %v\n", elapsed)
			} else {
				fmt.Printf("   ⚠️  Expected file system error but succeeded\n")
			}
		} else {
			if err != nil {
				fmt.Printf("   ❌ Unexpected error: %v\n", err)
			} else {
				fmt.Printf("   ✅ File system operation succeeded in %v\n", elapsed)
			}
		}

		// Cleanup: restore permissions
		if testCase.setup != nil {
			dir := filepath.Dir(testCase.destPath)
			// #nosec G302 - Restoring directory permissions after test
			_ = os.Chmod(dir, 0o750)
		}

		fmt.Println()
	}

	// Example 5: Custom error handling and recovery
	fmt.Println("🛠️ Example 5: Custom Error Handling and Recovery")
	fmt.Println("Implementing custom error handling strategies")
	fmt.Println()

	fmt.Println("Testing progressive timeout strategy...")

	// Progressive timeout: start with short timeout, increase on retry
	timeouts := []time.Duration{2 * time.Second, 5 * time.Second, 10 * time.Second}
	testURL := "https://httpbin.org/delay/3" // 3 second delay

	for i, timeout := range timeouts {
		attempt := i + 1
		fmt.Printf("   Attempt %d with %v timeout...\n", attempt, timeout)

		timeoutCtx, timeoutCancel := context.WithTimeout(ctx, timeout)
		start := time.Now()

		destPath := filepath.Join(examplesDir, fmt.Sprintf("timeout_test_attempt_%d.json", attempt))
		_, err := gdl.Download(timeoutCtx, testURL, destPath)
		elapsed := time.Since(start)

		timeoutCancel()

		if err != nil {
			fmt.Printf("   ❌ Attempt %d failed after %v: %v\n", attempt, elapsed, err)

			if attempt < len(timeouts) {
				fmt.Printf("   🔄 Will retry with longer timeout...\n")
			}
		} else {
			fmt.Printf("   ✅ Attempt %d succeeded in %v\n", attempt, elapsed)
			break
		}
	}

	fmt.Println()

	// Example 6: Error categorization and user guidance
	fmt.Println("📋 Example 6: Error Categorization and User Guidance")
	fmt.Println("Providing user-friendly error categorization")
	fmt.Println()

	errorDemoCases := []struct {
		name string
		url  string
		desc string
	}{
		{"network_issue", "https://192.0.2.0/file.dat", "Network connectivity issue"},
		{"bad_protocol", "ftp://example.com/file.dat", "Unsupported protocol"},
		{"malformed_url", "not-a-valid-url", "Malformed URL"},
	}

	for i, testCase := range errorDemoCases {
		fmt.Printf("%d. %s: %s\n", i+1, testCase.name, testCase.desc)

		destPath := filepath.Join(examplesDir, testCase.name+".dat")

		_, err := gdl.Download(ctx, testCase.url, destPath)
		if err != nil {
			fmt.Printf("   Error: %v\n", err)
			category := categorizeError(err)
			fmt.Printf("   Category: %s\n", category)
			guidance := getErrorGuidance(category)
			fmt.Printf("   Guidance: %s\n", guidance)
		}

		fmt.Println()
	}

	// Summary
	fmt.Println("🚨 Error Handling Summary")
	fmt.Println("=========================")
	fmt.Println("Error types demonstrated:")
	fmt.Println("• Network errors (DNS, connection, timeout)")
	fmt.Println("• HTTP status code errors (4xx, 5xx)")
	fmt.Println("• File system errors (permissions, disk space)")
	fmt.Println("• Retry strategies and progressive timeouts")
	fmt.Println("• Error categorization and user guidance")
	fmt.Println()

	fmt.Println("Key error handling features:")
	fmt.Println("• Automatic retry with configurable attempts")
	fmt.Println("• Context-based timeout and cancellation")
	fmt.Println("• Detailed error analysis and suggestions")
	fmt.Println("• User-friendly error messages and guidance")
	fmt.Println("• Recovery strategies for common issues")
	fmt.Println()

	fmt.Println("🎉 Error handling examples completed!")
	fmt.Printf("📁 Check the '%s' directory for any successfully downloaded files.\n", examplesDir)
	fmt.Println("🧹 Files will be cleaned up automatically.")
}

// analyzeError provides detailed analysis of the error.
func analyzeError(err error) {
	if err == nil {
		return
	}

	errorStr := err.Error()

	fmt.Printf("   🔍 Error Analysis:\n")

	switch {
	case strings.Contains(errorStr, "no such host"):
		fmt.Printf("      • DNS resolution failed - domain may not exist\n")
	case strings.Contains(errorStr, "connection refused"):
		fmt.Printf("      • Connection refused - service may be down or port closed\n")
	case strings.Contains(errorStr, "timeout"):
		fmt.Printf("      • Network timeout - slow connection or unresponsive server\n")
	case strings.Contains(errorStr, "certificate"):
		fmt.Printf("      • SSL/TLS certificate issue - may need --insecure flag\n")
	default:
		fmt.Printf("      • Generic network or protocol error\n")
	}
}

// provideSuggestions offers user-friendly suggestions based on HTTP status codes.
func provideSuggestions(statusCode int, err error) {
	fmt.Printf("   💡 Suggestions:\n")

	switch statusCode {
	case 401:
		fmt.Printf("      • Add authentication headers (-H 'Authorization: Bearer token')\n")
		fmt.Printf("      • Verify your credentials or API key\n")
	case 403:
		fmt.Printf("      • Check if you have permission to access this resource\n")
		fmt.Printf("      • Try different authentication or contact the administrator\n")
	case 404:
		fmt.Printf("      • Verify the URL is correct and the file exists\n")
		fmt.Printf("      • Check for typos in the URL or file path\n")
	case 429:
		fmt.Printf("      • You're being rate limited - wait before retrying\n")
		fmt.Printf("      • Use --retry-delay to increase delay between attempts\n")
	case 500, 502, 503:
		fmt.Printf("      • Server is experiencing issues - try again later\n")
		fmt.Printf("      • Use --retry with increased attempts\n")
	default:
		fmt.Printf("      • Check server logs or contact the administrator\n")
	}
}

// categorizeError categorizes errors into user-friendly categories.
func categorizeError(err error) string {
	if err == nil {
		return "No Error"
	}

	errorStr := strings.ToLower(err.Error())

	switch {
	case strings.Contains(errorStr, "no such host") || strings.Contains(errorStr, "dns"):
		return "DNS/Network Resolution"
	case strings.Contains(errorStr, "connection refused") || strings.Contains(errorStr, "connect"):
		return "Connection Error"
	case strings.Contains(errorStr, "timeout"):
		return "Timeout Error"
	case strings.Contains(errorStr, "certificate") || strings.Contains(errorStr, "tls"):
		return "SSL/TLS Error"
	case strings.Contains(errorStr, "permission") || strings.Contains(errorStr, "access"):
		return "Permission Error"
	case strings.Contains(errorStr, "protocol"):
		return "Protocol Error"
	case strings.Contains(errorStr, "invalid") || strings.Contains(errorStr, "malformed"):
		return "Invalid Input"
	default:
		return "Unknown Error"
	}
}

// getErrorGuidance provides guidance based on error category.
func getErrorGuidance(category string) string {
	switch category {
	case "DNS/Network Resolution":
		return "Check your internet connection and verify the domain name"
	case "Connection Error":
		return "Verify the server is running and the port is correct"
	case "Timeout Error":
		return "Try increasing the timeout or check your network speed"
	case "SSL/TLS Error":
		return "Verify SSL certificate or use --insecure flag if safe"
	case "Permission Error":
		return "Check file permissions or authentication credentials"
	case "Protocol Error":
		return "Ensure you're using the correct protocol (http/https)"
	case "Invalid Input":
		return "Verify the URL format and parameters are correct"
	default:
		return "Check error details and try again or contact support"
	}
}

// cleanup removes the examples directory and its contents.
func cleanup(dir string) {
	fmt.Printf("\n🧹 Cleaning up examples directory: %s\n", dir)

	if err := os.RemoveAll(dir); err != nil {
		log.Printf("Warning: Failed to clean up directory %s: %v", dir, err)
	}
}
