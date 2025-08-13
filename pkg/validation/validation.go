// Package validation provides input validation functions for public APIs.
// This package implements Tier 1 security validation for OSS distribution.
package validation

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"
)

// Config holds validation configuration
type Config struct {
	AllowLocalhost bool // Allow localhost URLs (for testing)
}

// DefaultConfig returns the default validation configuration for production use
func DefaultConfig() *Config {
	return &Config{
		AllowLocalhost: false,
	}
}

// TestConfig returns a validation configuration suitable for testing
func TestConfig() *Config {
	return &Config{
		AllowLocalhost: true,
	}
}

var globalConfig = DefaultConfig()

// ValidateURL validates a download URL for security and correctness.
// Returns an error if the URL is malformed, uses unsupported scheme, or poses security risks.
func ValidateURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("URL cannot be empty")
	}

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("malformed URL: %w", err)
	}

	// Check for supported schemes
	switch strings.ToLower(parsedURL.Scheme) {
	case "http", "https":
		// Allowed schemes
	case "":
		return fmt.Errorf("URL must include scheme (http:// or https://)")
	default:
		return fmt.Errorf("unsupported URL scheme: %s (only http and https are supported)", parsedURL.Scheme)
	}

	// Check for valid host
	if parsedURL.Host == "" {
		return fmt.Errorf("URL must include a valid host")
	}

	// Security checks
	if !globalConfig.AllowLocalhost && (strings.Contains(parsedURL.Host, "localhost") || strings.Contains(parsedURL.Host, "127.0.0.1")) {
		return fmt.Errorf("localhost URLs are not allowed for security reasons")
	}

	return nil
}

// ValidateDestination validates a file destination path for security and usability.
// Returns an error if the path is unsafe, invalid, or poses security risks.
func ValidateDestination(dest string) error {
	if dest == "" {
		return fmt.Errorf("destination path cannot be empty")
	}

	// Clean the path to resolve any ./ or ../ components
	cleanPath := filepath.Clean(dest)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return fmt.Errorf("destination path contains directory traversal: %s", dest)
	}

	// Convert to absolute path for further validation
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return fmt.Errorf("invalid destination path: %w", err)
	}

	// Check if the parent directory exists or can be created
	parentDir := filepath.Dir(absPath)
	if parentDir != "" {
		if info, err := os.Stat(parentDir); err != nil {
			if os.IsNotExist(err) {
				// Parent directory doesn't exist, which is fine - we can create it
				// But let's check if we have permission by attempting to create it
				if err := os.MkdirAll(parentDir, 0o750); err != nil {
					return fmt.Errorf("cannot create parent directory %s: %w", parentDir, err)
				}
			} else {
				return fmt.Errorf("cannot access parent directory %s: %w", parentDir, err)
			}
		} else if !info.IsDir() {
			return fmt.Errorf("parent path %s is not a directory", parentDir)
		}
	}

	// Check if destination already exists and is a directory
	if info, err := os.Stat(absPath); err == nil {
		if info.IsDir() {
			return fmt.Errorf("destination %s is a directory, expected file path", dest)
		}
	}

	return nil
}

// ValidateFileSize validates that a file size is within reasonable bounds.
// Returns an error if the size is negative or exceeds system limits.
func ValidateFileSize(size int64) error {
	if size < 0 {
		return fmt.Errorf("file size cannot be negative: %d", size)
	}

	// Check against reasonable maximum file size (100GB)
	const maxFileSize = 100 * 1024 * 1024 * 1024 // 100GB
	if size > maxFileSize {
		return fmt.Errorf("file size %d bytes exceeds maximum allowed size of %d bytes", size, maxFileSize)
	}

	return nil
}

// ValidateContentLength validates HTTP Content-Length header value.
// Returns an error if the value is invalid or poses security risks.
func ValidateContentLength(contentLength string) (int64, error) {
	if contentLength == "" {
		// Missing Content-Length is acceptable for chunked transfer
		return -1, nil
	}

	// Parse as integer
	var size int64
	n, err := fmt.Sscanf(contentLength, "%d", &size)
	if err != nil || n != 1 {
		return 0, fmt.Errorf("invalid Content-Length header: %s", contentLength)
	}

	// Validate the parsed size
	if err := ValidateFileSize(size); err != nil {
		return 0, fmt.Errorf("invalid Content-Length value: %w", err)
	}

	return size, nil
}

// ValidateChunkSize validates that a download chunk size is reasonable.
// Returns an error if the chunk size is too small, too large, or invalid.
func ValidateChunkSize(chunkSize int64) error {
	if chunkSize <= 0 {
		return fmt.Errorf("chunk size must be positive: %d", chunkSize)
	}

	const minChunkSize = 1024              // 1KB minimum
	const maxChunkSize = 100 * 1024 * 1024 // 100MB maximum

	if chunkSize < minChunkSize {
		return fmt.Errorf("chunk size %d is too small, minimum is %d bytes", chunkSize, minChunkSize)
	}

	if chunkSize > maxChunkSize {
		return fmt.Errorf("chunk size %d is too large, maximum is %d bytes", chunkSize, maxChunkSize)
	}

	return nil
}

// ValidateTimeout validates that a timeout duration is reasonable.
// Returns an error if the timeout is negative or excessively long.
func ValidateTimeout(timeoutSeconds int) error {
	if timeoutSeconds < 0 {
		return fmt.Errorf("timeout cannot be negative: %d", timeoutSeconds)
	}

	const maxTimeout = 24 * 60 * 60 // 24 hours in seconds
	if timeoutSeconds > maxTimeout {
		return fmt.Errorf("timeout %d seconds is too long, maximum is %d seconds (24 hours)", timeoutSeconds, maxTimeout)
	}

	return nil
}

// SanitizeFilename removes or replaces potentially dangerous characters in a filename.
// Returns a safe filename that can be used across different operating systems.
func SanitizeFilename(filename string) string {
	if filename == "" {
		return "download"
	}

	// Remove or replace dangerous characters
	dangerous := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	sanitized := filename

	for _, char := range dangerous {
		sanitized = strings.ReplaceAll(sanitized, char, "_")
	}

	// Trim whitespace and dots to prevent hidden files or path issues
	sanitized = strings.Trim(sanitized, " .")

	// Ensure the filename is not empty or just underscores after sanitization
	if sanitized == "" || strings.Trim(sanitized, "_") == "" {
		return "download"
	}

	// Limit filename length to prevent filesystem issues
	const maxFilenameLength = 255
	if len(sanitized) > maxFilenameLength {
		// Keep the extension if possible
		ext := filepath.Ext(sanitized)
		base := sanitized[:maxFilenameLength-len(ext)]
		sanitized = base + ext
	}

	return sanitized
}

// SetConfig sets the global validation configuration.
// This should only be used for testing purposes.
func SetConfig(config *Config) {
	globalConfig = config
}

// GetConfig returns the current validation configuration.
func GetConfig() *Config {
	return globalConfig
}
