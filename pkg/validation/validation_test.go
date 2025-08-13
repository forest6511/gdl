package validation

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{
			name:    "valid https URL",
			url:     "https://example.com/file.zip",
			wantErr: false,
		},
		{
			name:    "valid http URL",
			url:     "http://example.com/file.zip",
			wantErr: false,
		},
		{
			name:    "empty URL",
			url:     "",
			wantErr: true,
		},
		{
			name:    "malformed URL",
			url:     "not-a-url",
			wantErr: true,
		},
		{
			name:    "unsupported scheme",
			url:     "ftp://example.com/file.zip",
			wantErr: true,
		},
		{
			name:    "localhost URL",
			url:     "http://localhost:8080/file.zip",
			wantErr: true,
		},
		{
			name:    "127.0.0.1 URL",
			url:     "http://127.0.0.1:8080/file.zip",
			wantErr: true,
		},
		{
			name:    "URL without scheme",
			url:     "example.com/file.zip",
			wantErr: true,
		},
		{
			name:    "URL without host",
			url:     "https:///file.zip",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateURL(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateURL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateDestination(t *testing.T) {
	// Create temporary directory for testing
	tempDir := t.TempDir()

	tests := []struct {
		name    string
		dest    string
		wantErr bool
	}{
		{
			name:    "valid absolute path",
			dest:    filepath.Join(tempDir, "test.zip"),
			wantErr: false,
		},
		{
			name:    "valid relative path",
			dest:    "test.zip",
			wantErr: false,
		},
		{
			name:    "empty destination",
			dest:    "",
			wantErr: true,
		},
		{
			name:    "directory traversal attempt",
			dest:    "../../../etc/passwd",
			wantErr: true,
		},
		{
			name:    "path with directory traversal",
			dest:    "safe/../../dangerous.zip",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateDestination(tt.dest)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateDestination() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateFileSize(t *testing.T) {
	tests := []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{
			name:    "valid small size",
			size:    1024,
			wantErr: false,
		},
		{
			name:    "valid large size",
			size:    1024 * 1024 * 1024, // 1GB
			wantErr: false,
		},
		{
			name:    "negative size",
			size:    -1,
			wantErr: true,
		},
		{
			name:    "zero size",
			size:    0,
			wantErr: false,
		},
		{
			name:    "excessively large size",
			size:    200 * 1024 * 1024 * 1024, // 200GB
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileSize(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFileSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateContentLength(t *testing.T) {
	tests := []struct {
		name          string
		contentLength string
		expectedSize  int64
		wantErr       bool
	}{
		{
			name:          "valid content length",
			contentLength: "1024",
			expectedSize:  1024,
			wantErr:       false,
		},
		{
			name:          "empty content length",
			contentLength: "",
			expectedSize:  -1,
			wantErr:       false,
		},
		{
			name:          "invalid content length",
			contentLength: "not-a-number",
			expectedSize:  0,
			wantErr:       true,
		},
		{
			name:          "negative content length",
			contentLength: "-100",
			expectedSize:  0,
			wantErr:       true,
		},
		{
			name:          "excessively large content length",
			contentLength: "999999999999999999999", // Much larger than 100GB
			expectedSize:  0,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			size, err := ValidateContentLength(tt.contentLength)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateContentLength() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if size != tt.expectedSize {
				t.Errorf("ValidateContentLength() size = %v, expected %v", size, tt.expectedSize)
			}
		})
	}
}

func TestValidateChunkSize(t *testing.T) {
	tests := []struct {
		name      string
		chunkSize int64
		wantErr   bool
	}{
		{
			name:      "valid chunk size",
			chunkSize: 64 * 1024, // 64KB
			wantErr:   false,
		},
		{
			name:      "minimum valid chunk size",
			chunkSize: 1024, // 1KB
			wantErr:   false,
		},
		{
			name:      "maximum valid chunk size",
			chunkSize: 100 * 1024 * 1024, // 100MB
			wantErr:   false,
		},
		{
			name:      "zero chunk size",
			chunkSize: 0,
			wantErr:   true,
		},
		{
			name:      "negative chunk size",
			chunkSize: -1024,
			wantErr:   true,
		},
		{
			name:      "too small chunk size",
			chunkSize: 512, // Less than 1KB
			wantErr:   true,
		},
		{
			name:      "too large chunk size",
			chunkSize: 200 * 1024 * 1024, // 200MB
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateChunkSize(tt.chunkSize)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateChunkSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestValidateTimeout(t *testing.T) {
	tests := []struct {
		name           string
		timeoutSeconds int
		wantErr        bool
	}{
		{
			name:           "valid timeout",
			timeoutSeconds: 30,
			wantErr:        false,
		},
		{
			name:           "zero timeout",
			timeoutSeconds: 0,
			wantErr:        false,
		},
		{
			name:           "maximum valid timeout",
			timeoutSeconds: 24 * 60 * 60, // 24 hours
			wantErr:        false,
		},
		{
			name:           "negative timeout",
			timeoutSeconds: -10,
			wantErr:        true,
		},
		{
			name:           "excessively long timeout",
			timeoutSeconds: 48 * 60 * 60, // 48 hours
			wantErr:        true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTimeout(tt.timeoutSeconds)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateTimeout() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		expected string
	}{
		{
			name:     "clean filename",
			filename: "document.pdf",
			expected: "document.pdf",
		},
		{
			name:     "filename with dangerous characters",
			filename: "my/file\\name:with*dangerous?chars",
			expected: "my_file_name_with_dangerous_chars",
		},
		{
			name:     "empty filename",
			filename: "",
			expected: "download",
		},
		{
			name:     "filename with only whitespace",
			filename: "   ",
			expected: "download",
		},
		{
			name:     "filename with leading/trailing dots",
			filename: "..hidden.file..",
			expected: "hidden.file",
		},
		{
			name:     "very long filename",
			filename: strings.Repeat("a", 300) + ".txt",
			expected: strings.Repeat("a", 255-4) + ".txt", // 255 total chars with .txt
		},
		{
			name:     "filename becomes empty after sanitization",
			filename: "///\\\\\\",
			expected: "download",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SanitizeFilename(tt.filename)
			if result != tt.expected {
				t.Errorf("SanitizeFilename() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestValidateDestination_DirectoryCreation(t *testing.T) {
	// Test that the function can create parent directories
	tempDir := t.TempDir()
	newPath := filepath.Join(tempDir, "new", "nested", "directory", "file.txt")

	err := ValidateDestination(newPath)
	if err != nil {
		t.Errorf("ValidateDestination() failed to create parent directories: %v", err)
	}

	// Check that the parent directory was actually created
	parentDir := filepath.Dir(newPath)
	if _, err := os.Stat(parentDir); os.IsNotExist(err) {
		t.Errorf("Parent directory was not created: %s", parentDir)
	}
}

func TestValidateDestination_ExistingDirectory(t *testing.T) {
	// Test that the function rejects existing directories as destinations
	tempDir := t.TempDir()

	err := ValidateDestination(tempDir)
	if err == nil {
		t.Error("ValidateDestination() should reject existing directory as destination")
	}
}
