package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
)

// TestValidateURLErrors tests error handling in ValidateURL
func TestValidateURLErrors(t *testing.T) {
	t.Run("EmptyURL", func(t *testing.T) {
		err := ValidateURL("")

		if err == nil {
			t.Fatal("Expected error for empty URL, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "URL cannot be empty") {
			t.Errorf("Expected 'URL cannot be empty' in message, got: %s", downloadErr.Message)
		}
	})

	t.Run("MalformedURL", func(t *testing.T) {
		err := ValidateURL("ht!tp://inv@lid:url")

		if err == nil {
			t.Fatal("Expected error for malformed URL, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeInvalidURL {
			t.Errorf("Expected CodeInvalidURL, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "malformed URL") {
			t.Errorf("Expected 'malformed URL' in message, got: %s", downloadErr.Message)
		}
	})

	t.Run("MissingScheme", func(t *testing.T) {
		err := ValidateURL("example.com/file.txt")

		if err == nil {
			t.Fatal("Expected error for missing scheme, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "must include scheme") {
			t.Errorf("Expected 'must include scheme' in message, got: %s", downloadErr.Message)
		}
	})

	t.Run("UnsupportedScheme", func(t *testing.T) {
		err := ValidateURL("ftp://example.com/file.txt")

		if err == nil {
			t.Fatal("Expected error for unsupported scheme, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "unsupported URL scheme") {
			t.Errorf("Expected 'unsupported URL scheme' in message, got: %s", downloadErr.Message)
		}
		if !strings.Contains(downloadErr.Message, "ftp") {
			t.Errorf("Expected 'ftp' in message, got: %s", downloadErr.Message)
		}
	})

	t.Run("MissingHost", func(t *testing.T) {
		err := ValidateURL("http:///path/to/file")

		if err == nil {
			t.Fatal("Expected error for missing host, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "must include a valid host") {
			t.Errorf("Expected 'must include a valid host' in message, got: %s", downloadErr.Message)
		}
	})

	t.Run("LocalhostNotAllowed", func(t *testing.T) {
		// Save original config
		originalAllowLocalhost := globalConfig.AllowLocalhost
		defer func() { globalConfig.AllowLocalhost = originalAllowLocalhost }()

		// Set to disallow localhost
		globalConfig.AllowLocalhost = false

		testCases := []string{
			"http://localhost/file.txt",
			"http://127.0.0.1/file.txt",
			"https://localhost:8080/download",
		}

		for _, url := range testCases {
			err := ValidateURL(url)

			if err == nil {
				t.Errorf("Expected error for localhost URL: %s, got nil", url)
				continue
			}

			var downloadErr *gdlerrors.DownloadError
			if !gdlerrors.AsDownloadError(err, &downloadErr) {
				t.Errorf("Expected DownloadError for %s", url)
				continue
			}
			if downloadErr.Code != gdlerrors.CodeValidationError {
				t.Errorf("Expected CodeValidationError for %s, got: %s", url, downloadErr.Code)
			}
			if !strings.Contains(downloadErr.Message, "localhost URLs are not allowed") {
				t.Errorf("Expected 'localhost URLs are not allowed' in message for %s, got: %s", url, downloadErr.Message)
			}
		}
	})
}

// TestValidateDestinationErrors tests error handling in ValidateDestination
func TestValidateDestinationErrors(t *testing.T) {
	t.Run("EmptyDestination", func(t *testing.T) {
		err := ValidateDestination("")

		if err == nil {
			t.Fatal("Expected error for empty destination, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "destination path cannot be empty") {
			t.Errorf("Expected 'destination path cannot be empty' in message, got: %s", downloadErr.Message)
		}
	})

	t.Run("DirectoryTraversal", func(t *testing.T) {
		testCases := []string{
			"../../../etc/passwd",
			"./../../sensitive/file",
			"subdir/../../../file.txt",
		}

		for _, dest := range testCases {
			err := ValidateDestination(dest)

			if err == nil {
				t.Errorf("Expected error for directory traversal: %s, got nil", dest)
				continue
			}

			var downloadErr *gdlerrors.DownloadError
			if !gdlerrors.AsDownloadError(err, &downloadErr) {
				t.Errorf("Expected DownloadError for %s", dest)
				continue
			}
			if downloadErr.Code != gdlerrors.CodeValidationError {
				t.Errorf("Expected CodeValidationError for %s, got: %s", dest, downloadErr.Code)
			}
			if !strings.Contains(downloadErr.Message, "directory traversal") {
				t.Errorf("Expected 'directory traversal' in message for %s, got: %s", dest, downloadErr.Message)
			}
		}
	})

	t.Run("ParentNotDirectory", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a file that will be mistaken for a parent directory
		parentFile := filepath.Join(tempDir, "parent.txt")
		if err := os.WriteFile(parentFile, []byte("content"), 0o600); err != nil {
			t.Fatalf("Failed to create parent file: %v", err)
		}

		// Try to create a destination where parent is actually a file
		dest := filepath.Join(parentFile, "child.txt")
		err := ValidateDestination(dest)

		if err == nil {
			t.Fatal("Expected error when parent path is not a directory, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "is not a directory") {
			t.Errorf("Expected 'is not a directory' in message, got: %s", downloadErr.Message)
		}
	})

	t.Run("DestinationIsDirectory", func(t *testing.T) {
		tempDir := t.TempDir()

		// Create a directory as the destination
		dirDest := filepath.Join(tempDir, "existing_dir")
		if err := os.Mkdir(dirDest, 0o755); err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		err := ValidateDestination(dirDest)

		if err == nil {
			t.Fatal("Expected error when destination is a directory, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "is a directory") {
			t.Errorf("Expected 'is a directory' in message, got: %s", downloadErr.Message)
		}
	})

	t.Run("ParentDirectoryAccessFailure", func(t *testing.T) {
		if os.Getuid() == 0 {
			t.Skip("Skipping test when running as root (permissions tests won't work)")
		}

		tempDir := t.TempDir()

		// Create a read-only directory
		readOnlyDir := filepath.Join(tempDir, "readonly")
		if err := os.Mkdir(readOnlyDir, 0o400); err != nil {
			t.Fatalf("Failed to create read-only directory: %v", err)
		}
		defer func() {
			_ = os.Chmod(readOnlyDir, 0o700)
		}()

		// Try to create a file in a subdirectory of the read-only directory
		dest := filepath.Join(readOnlyDir, "subdir", "file.txt")
		err := ValidateDestination(dest)

		if err == nil {
			t.Fatal("Expected error when parent directory cannot be accessed, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if gdlerrors.AsDownloadError(err, &downloadErr) {
			if downloadErr.Code != gdlerrors.CodePermissionDenied {
				t.Errorf("Expected CodePermissionDenied, got: %s", downloadErr.Code)
			}
			// The error can be either "cannot create" or "cannot access" depending on the exact path
			if !strings.Contains(downloadErr.Message, "cannot") {
				t.Errorf("Expected permission error message, got: %s", downloadErr.Message)
			}
		}
	})
}

// TestValidateContentLengthErrors tests error handling in ValidateContentLength
func TestValidateContentLengthErrors(t *testing.T) {
	t.Run("InvalidFormat", func(t *testing.T) {
		testCases := []string{
			"abc",
			"12.34",
			"1a2b3c",
			"  ",
		}

		for _, input := range testCases {
			_, err := ValidateContentLength(input)

			if err == nil {
				t.Errorf("Expected error for invalid format: %q, got nil", input)
				continue
			}

			var downloadErr *gdlerrors.DownloadError
			if !gdlerrors.AsDownloadError(err, &downloadErr) {
				t.Errorf("Expected DownloadError for %q", input)
				continue
			}
			if downloadErr.Code != gdlerrors.CodeValidationError {
				t.Errorf("Expected CodeValidationError for %q, got: %s", input, downloadErr.Code)
			}
		}
	})

	t.Run("NegativeValue", func(t *testing.T) {
		_, err := ValidateContentLength("-100")

		if err == nil {
			t.Fatal("Expected error for negative value, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "invalid Content-Length header") {
			t.Errorf("Expected 'invalid Content-Length header' in message, got: %s", downloadErr.Message)
		}
	})
}

// TestParseIntValueErrors tests error handling in parseIntValue
func TestParseIntValueErrors(t *testing.T) {
	t.Run("InvalidCharacters", func(t *testing.T) {
		var result int64
		_, err := parseIntValue("12a34", &result)

		if err == nil {
			t.Fatal("Expected error for invalid characters, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "invalid integer format") {
			t.Errorf("Expected 'invalid integer format' in message, got: %s", downloadErr.Message)
		}
	})

	t.Run("NoDigitsFound", func(t *testing.T) {
		var result int64
		_, err := parseIntValue("", &result)

		if err == nil {
			t.Fatal("Expected error for no digits, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "no digits found") {
			t.Errorf("Expected 'no digits found' in message, got: %s", downloadErr.Message)
		}
	})

	t.Run("SpecialCharacters", func(t *testing.T) {
		testCases := []string{
			"123!",
			"@456",
			"78#90",
		}

		for _, input := range testCases {
			var result int64
			_, err := parseIntValue(input, &result)

			if err == nil {
				t.Errorf("Expected error for special characters in %q, got nil", input)
				continue
			}

			var downloadErr *gdlerrors.DownloadError
			if !gdlerrors.AsDownloadError(err, &downloadErr) {
				t.Errorf("Expected DownloadError for %q", input)
				continue
			}
			if downloadErr.Code != gdlerrors.CodeValidationError {
				t.Errorf("Expected CodeValidationError for %q, got: %s", input, downloadErr.Code)
			}
		}
	})
}

// TestValidateFileSizeErrors tests error handling in ValidateFileSize
func TestValidateFileSizeErrors(t *testing.T) {
	t.Run("NegativeSize", func(t *testing.T) {
		err := ValidateFileSize(-100)

		if err == nil {
			t.Fatal("Expected error for negative size, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "file size cannot be negative") {
			t.Errorf("Expected 'file size cannot be negative' in message, got: %s", downloadErr.Message)
		}
	})
}

// TestValidateTimeoutErrors tests error handling in ValidateTimeout
func TestValidateTimeoutErrors(t *testing.T) {
	t.Run("NegativeTimeout", func(t *testing.T) {
		err := ValidateTimeout(-5)

		if err == nil {
			t.Fatal("Expected error for negative timeout, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "timeout cannot be negative") {
			t.Errorf("Expected 'timeout cannot be negative' in message, got: %s", downloadErr.Message)
		}
	})
}

// TestValidateChunkSizeErrors tests error handling in ValidateChunkSize
func TestValidateChunkSizeErrors(t *testing.T) {
	t.Run("TooSmall", func(t *testing.T) {
		err := ValidateChunkSize(512) // Less than 1KB (1024)

		if err == nil {
			t.Fatal("Expected error for chunk size too small, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "chunk size is too small") {
			t.Errorf("Expected 'chunk size is too small' in message, got: %s", downloadErr.Message)
		}
	})

	t.Run("TooLarge", func(t *testing.T) {
		err := ValidateChunkSize(101 * 1024 * 1024) // More than 100MB

		if err == nil {
			t.Fatal("Expected error for chunk size too large, got nil")
		}

		var downloadErr *gdlerrors.DownloadError
		if !gdlerrors.AsDownloadError(err, &downloadErr) {
			t.Fatal("Expected DownloadError")
		}
		if downloadErr.Code != gdlerrors.CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", downloadErr.Code)
		}
		if !strings.Contains(downloadErr.Message, "chunk size is too large") {
			t.Errorf("Expected 'chunk size is too large' in message, got: %s", downloadErr.Message)
		}
	})
}
