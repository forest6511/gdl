package errors

import (
	"errors"
	"strings"
	"testing"
)

func TestFormatForCLI(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected []string // Parts that should be in the output
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: []string{""},
		},
		{
			name:     "non-DownloadError",
			err:      errors.New("generic error"),
			expected: []string{"generic error"},
		},
		{
			name: "DownloadError with all fields",
			err: &DownloadError{
				Code:      CodeNetworkError,
				Message:   "connection failed",
				URL:       "https://example.com/file.zip",
				Filename:  "file.zip",
				Retryable: true,
			},
			expected: []string{"Error [network_error]:", "connection failed", "URL: https://example.com/file.zip", "File: file.zip", "(retryable)"},
		},
		{
			name: "DownloadError without URL and filename",
			err: &DownloadError{
				Code:      CodeStorageError,
				Message:   "disk full",
				Retryable: false,
			},
			expected: []string{"Error [storage_error]:", "disk full"},
		},
		{
			name: "DownloadError with empty message",
			err: &DownloadError{
				Code: CodeTimeout,
			},
			expected: []string{"Error [timeout]:"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatForCLI(tt.err)
			for _, part := range tt.expected {
				if !strings.Contains(result, part) {
					t.Errorf("FormatForCLI() result missing expected part\nGot: %s\nMissing: %s", result, part)
				}
			}
		})
	}
}

func TestGetUserMessage(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: "",
		},
		{
			name:     "non-DownloadError",
			err:      errors.New("generic error"),
			expected: "generic error",
		},
		{
			name: "DownloadError with message",
			err: &DownloadError{
				Code:    CodeNetworkError,
				Message: "connection failed",
			},
			expected: "connection failed",
		},
		{
			name: "DownloadError without message",
			err: &DownloadError{
				Code: CodeTimeout,
			},
			expected: "download error occurred",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetUserMessage(tt.err)
			if result != tt.expected {
				t.Errorf("GetUserMessage() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestFormatWithDetails(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected []string
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: []string{""},
		},
		{
			name:     "non-DownloadError",
			err:      errors.New("generic error"),
			expected: []string{"generic error"},
		},
		{
			name: "DownloadError with all metadata",
			err: &DownloadError{
				Code:             CodeNetworkError,
				Message:          "connection failed",
				Details:          "TCP timeout",
				URL:              "https://example.com/file.zip",
				Filename:         "file.zip",
				HTTPStatusCode:   503,
				BytesTransferred: 1024,
				Retryable:        true,
				Underlying:       errors.New("underlying error"),
			},
			expected: []string{
				"=== DownloadError [network_error] ===",
				"Message: connection failed",
				"Details: TCP timeout",
				"URL: https://example.com/file.zip",
				"Filename: file.zip",
				"HTTP Status: 503",
				"Bytes Transferred: 1024",
				"Retryable: true",
				"Underlying: underlying error",
			},
		},
		{
			name: "DownloadError with minimal fields",
			err: &DownloadError{
				Code:    CodeStorageError,
				Message: "disk error",
			},
			expected: []string{
				"=== DownloadError [storage_error] ===",
				"Message: disk error",
				"Retryable: false",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := FormatWithDetails(tt.err)
			for _, part := range tt.expected {
				if !strings.Contains(result, part) {
					t.Errorf("FormatWithDetails() result missing expected part\nGot: %s\nMissing: %s", result, part)
				}
			}
		})
	}
}

func TestAsDownloadError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-DownloadError",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name: "DownloadError",
			err: &DownloadError{
				Code:    CodeNetworkError,
				Message: "connection failed",
			},
			expected: true,
		},
		{
			name: "wrapped DownloadError",
			err: WrapError(
				&DownloadError{Code: CodeNetworkError, Message: "inner"},
				CodeStorageError,
				"outer",
			),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var downloadErr *DownloadError
			result := AsDownloadError(tt.err, &downloadErr)
			if result != tt.expected {
				t.Errorf("AsDownloadError() = %v, want %v", result, tt.expected)
			}
			if tt.expected && downloadErr == nil {
				t.Error("AsDownloadError() returned true but target is nil")
			}
		})
	}
}

func TestIsDownloadError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{
			name:     "nil error",
			err:      nil,
			expected: false,
		},
		{
			name:     "non-DownloadError",
			err:      errors.New("generic error"),
			expected: false,
		},
		{
			name: "DownloadError",
			err: &DownloadError{
				Code:    CodeNetworkError,
				Message: "connection failed",
			},
			expected: true,
		},
		{
			name: "wrapped DownloadError",
			err: WrapError(
				&DownloadError{Code: CodeNetworkError, Message: "inner"},
				CodeStorageError,
				"outer",
			),
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsDownloadError(tt.err)
			if result != tt.expected {
				t.Errorf("IsDownloadError() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestSuggestAction(t *testing.T) {
	tests := []struct {
		name          string
		err           error
		expectedParts []string
	}{
		{
			name:          "nil error",
			err:           nil,
			expectedParts: []string{""},
		},
		{
			name: "CodeInvalidURL",
			err: &DownloadError{
				Code: CodeInvalidURL,
			},
			expectedParts: []string{"check", "URL"},
		},
		{
			name: "CodeFileExists",
			err: &DownloadError{
				Code: CodeFileExists,
			},
			expectedParts: []string{"--overwrite"},
		},
		{
			name: "CodeInsufficientSpace",
			err: &DownloadError{
				Code: CodeInsufficientSpace,
			},
			expectedParts: []string{"disk space"},
		},
		{
			name: "CodeNetworkError",
			err: &DownloadError{
				Code: CodeNetworkError,
			},
			expectedParts: []string{"network"},
		},
		{
			name: "CodeTimeout",
			err: &DownloadError{
				Code: CodeTimeout,
			},
			expectedParts: []string{"network"},
		},
		{
			name: "CodePermissionDenied",
			err: &DownloadError{
				Code: CodePermissionDenied,
			},
			expectedParts: []string{"permissions"},
		},
		{
			name: "CodeFileNotFound",
			err: &DownloadError{
				Code: CodeFileNotFound,
			},
			expectedParts: []string{"URL", "path"},
		},
		{
			name: "CodeAuthenticationFailed",
			err: &DownloadError{
				Code: CodeAuthenticationFailed,
			},
			expectedParts: []string{"credentials"},
		},
		{
			name: "CodeServerError",
			err: &DownloadError{
				Code: CodeServerError,
			},
			expectedParts: []string{"server"},
		},
		{
			name: "CodeClientError",
			err: &DownloadError{
				Code: CodeClientError,
			},
			expectedParts: []string{"parameters"},
		},
		{
			name: "CodeCancelled",
			err: &DownloadError{
				Code: CodeCancelled,
			},
			expectedParts: []string{"cancelled"},
		},
		{
			name: "CodeCorruptedData",
			err: &DownloadError{
				Code: CodeCorruptedData,
			},
			expectedParts: []string{"corrupted"},
		},
		{
			name: "CodeInvalidPath",
			err: &DownloadError{
				Code: CodeInvalidPath,
			},
			expectedParts: []string{"path"},
		},
		{
			name: "CodePluginError",
			err: &DownloadError{
				Code: CodePluginError,
			},
			expectedParts: []string{"plugin"},
		},
		{
			name: "CodeConfigError",
			err: &DownloadError{
				Code: CodeConfigError,
			},
			expectedParts: []string{"configuration"},
		},
		{
			name: "CodeValidationError",
			err: &DownloadError{
				Code: CodeValidationError,
			},
			expectedParts: []string{"input"},
		},
		{
			name: "CodeStorageError",
			err: &DownloadError{
				Code: CodeStorageError,
			},
			expectedParts: []string{"storage"},
		},
		{
			name: "CodeUnknown",
			err: &DownloadError{
				Code: CodeUnknown,
			},
			expectedParts: []string{"try again"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SuggestAction(tt.err)
			for _, part := range tt.expectedParts {
				if !strings.Contains(strings.ToLower(result), strings.ToLower(part)) {
					t.Errorf("SuggestAction() result missing expected part\nGot: %s\nMissing: %s", result, part)
				}
			}
		})
	}
}
