// Package validation provides input validation functions for public APIs.
// This package implements Tier 1 security validation for OSS distribution.
package validation

import (
	"net/url"
	"os"
	"path/filepath"
	"strings"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
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
		return gdlerrors.NewValidationError("url", "URL cannot be empty")
	}

	// Parse the URL
	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeInvalidURL, "malformed URL", rawURL)
	}

	// Check for supported schemes
	switch strings.ToLower(parsedURL.Scheme) {
	case "http", "https":
		// Allowed schemes
	case "":
		return gdlerrors.NewValidationError("url", "URL must include scheme (http:// or https://)")
	default:
		return gdlerrors.NewValidationError("url", "unsupported URL scheme: "+parsedURL.Scheme+" (only http and https are supported)")
	}

	// Check for valid host
	if parsedURL.Host == "" {
		return gdlerrors.NewValidationError("url", "URL must include a valid host")
	}

	// Security checks
	if !globalConfig.AllowLocalhost && (strings.Contains(parsedURL.Host, "localhost") || strings.Contains(parsedURL.Host, "127.0.0.1")) {
		return gdlerrors.NewValidationError("url", "localhost URLs are not allowed for security reasons")
	}

	return nil
}

// ValidateDestination validates a file destination path for security and usability.
// Returns an error if the path is unsafe, invalid, or poses security risks.
func ValidateDestination(dest string) error {
	if dest == "" {
		return gdlerrors.NewValidationError("destination", "destination path cannot be empty")
	}

	// Clean the path to resolve any ./ or ../ components
	cleanPath := filepath.Clean(dest)

	// Check for directory traversal attempts
	if strings.Contains(cleanPath, "..") {
		return gdlerrors.NewValidationError("destination", "destination path contains directory traversal: "+dest)
	}

	// Convert to absolute path for further validation
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return gdlerrors.NewInvalidPathError(dest, err)
	}

	// Check if the parent directory exists or can be created
	parentDir := filepath.Dir(absPath)
	if parentDir != "" {
		if info, err := os.Stat(parentDir); err != nil {
			if os.IsNotExist(err) {
				// Parent directory doesn't exist, which is fine - we can create it
				// But let's check if we have permission by attempting to create it
				if err := os.MkdirAll(parentDir, 0o750); err != nil {
					return gdlerrors.WrapError(err, gdlerrors.CodePermissionDenied, "cannot create parent directory "+parentDir)
				}
			} else {
				return gdlerrors.WrapError(err, gdlerrors.CodePermissionDenied, "cannot access parent directory "+parentDir)
			}
		} else if !info.IsDir() {
			return gdlerrors.NewValidationError("destination", "parent path "+parentDir+" is not a directory")
		}
	}

	// Check if destination already exists and is a directory
	if info, err := os.Stat(absPath); err == nil {
		if info.IsDir() {
			return gdlerrors.NewValidationError("destination", "destination "+dest+" is a directory, expected file path")
		}
	}

	return nil
}

// ValidateFileSize validates that a file size is within reasonable bounds.
// Returns an error if the size is negative or exceeds system limits.
func ValidateFileSize(size int64) error {
	if size < 0 {
		return gdlerrors.NewValidationError("file_size", "file size cannot be negative")
	}

	// Check against reasonable maximum file size (100GB)
	const maxFileSize int64 = 100 * 1024 * 1024 * 1024 // 100GB
	if size > maxFileSize {
		return gdlerrors.NewValidationError("file_size", "file size exceeds maximum allowed size of 100GB")
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
	_, err := parseContentLength(contentLength, &size)
	if err != nil {
		return 0, gdlerrors.NewValidationError("content_length", "invalid Content-Length header: "+contentLength)
	}

	// Validate the parsed size
	if err := ValidateFileSize(size); err != nil {
		return 0, gdlerrors.WrapError(err, gdlerrors.CodeValidationError, "invalid Content-Length value")
	}

	return size, nil
}

// parseContentLength is a helper function to parse content length
func parseContentLength(contentLength string, size *int64) (int, error) {
	return parseIntValue(contentLength, size)
}

// parseIntValue parses an integer value from a string
func parseIntValue(s string, v *int64) (int, error) {
	var temp int64
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			temp = temp*10 + int64(c-'0')
			n++
		} else {
			return 0, gdlerrors.NewValidationError("parse", "invalid integer format")
		}
	}
	if n == 0 {
		return 0, gdlerrors.NewValidationError("parse", "no digits found")
	}
	*v = temp
	return 1, nil
}

// ValidateChunkSize validates that a download chunk size is reasonable.
// Returns an error if the chunk size is too small, too large, or invalid.
func ValidateChunkSize(chunkSize int64) error {
	if chunkSize <= 0 {
		return gdlerrors.NewValidationError("chunk_size", "chunk size must be positive")
	}

	const minChunkSize = 1024              // 1KB minimum
	const maxChunkSize = 100 * 1024 * 1024 // 100MB maximum

	if chunkSize < minChunkSize {
		return gdlerrors.NewValidationError("chunk_size", "chunk size is too small, minimum is 1KB")
	}

	if chunkSize > maxChunkSize {
		return gdlerrors.NewValidationError("chunk_size", "chunk size is too large, maximum is 100MB")
	}

	return nil
}

// ValidateTimeout validates that a timeout duration is reasonable.
// Returns an error if the timeout is negative or excessively long.
func ValidateTimeout(timeoutSeconds int) error {
	if timeoutSeconds < 0 {
		return gdlerrors.NewValidationError("timeout", "timeout cannot be negative")
	}

	const maxTimeout = 24 * 60 * 60 // 24 hours in seconds
	if timeoutSeconds > maxTimeout {
		return gdlerrors.NewValidationError("timeout", "timeout is too long, maximum is 24 hours")
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
