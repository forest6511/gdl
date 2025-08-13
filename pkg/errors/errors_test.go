package errors

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
	"time"
)

func TestErrorCode_String(t *testing.T) {
	tests := []struct {
		name     string
		code     ErrorCode
		expected string
	}{
		{"CodeUnknown", CodeUnknown, "unknown"},
		{"CodeInvalidURL", CodeInvalidURL, "invalid_url"},
		{"CodeFileExists", CodeFileExists, "file_exists"},
		{"CodeInsufficientSpace", CodeInsufficientSpace, "insufficient_space"},
		{"CodeNetworkError", CodeNetworkError, "network_error"},
		{"CodeTimeout", CodeTimeout, "timeout"},
		{"CodePermissionDenied", CodePermissionDenied, "permission_denied"},
		{"CodeFileNotFound", CodeFileNotFound, "file_not_found"},
		{"CodeAuthenticationFailed", CodeAuthenticationFailed, "authentication_failed"},
		{"CodeServerError", CodeServerError, "server_error"},
		{"CodeClientError", CodeClientError, "client_error"},
		{"CodeCancelled", CodeCancelled, "cancelled"},
		{"CodeCorruptedData", CodeCorruptedData, "corrupted_data"},
		{"Invalid code", ErrorCode(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.code.String(); got != tt.expected {
				t.Errorf("ErrorCode.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDownloadError_Error(t *testing.T) {
	tests := []struct {
		name     string
		err      *DownloadError
		expected string
	}{
		{
			name: "with message",
			err: &DownloadError{
				Code:    CodeInvalidURL,
				Message: "Invalid URL provided",
			},
			expected: "Invalid URL provided",
		},
		{
			name: "with underlying error, no message",
			err: &DownloadError{
				Code:       CodeNetworkError,
				Underlying: errors.New("connection refused"),
			},
			expected: "connection refused",
		},
		{
			name: "no message, no underlying error",
			err: &DownloadError{
				Code: CodeUnknown,
			},
			expected: "download error occurred",
		},
		{
			name: "message takes precedence over underlying",
			err: &DownloadError{
				Code:       CodeTimeout,
				Message:    "Request timeout",
				Underlying: errors.New("context deadline exceeded"),
			},
			expected: "Request timeout",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Error(); got != tt.expected {
				t.Errorf("DownloadError.Error() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDownloadError_Unwrap(t *testing.T) {
	underlyingErr := errors.New("underlying error")

	tests := []struct {
		name     string
		err      *DownloadError
		expected error
	}{
		{
			name: "with underlying error",
			err: &DownloadError{
				Underlying: underlyingErr,
			},
			expected: underlyingErr,
		},
		{
			name: "without underlying error",
			err: &DownloadError{
				Code: CodeFileExists,
			},
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Unwrap(); !errors.Is(got, tt.expected) {
				t.Errorf("DownloadError.Unwrap() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestDownloadError_Is(t *testing.T) {
	tests := []struct {
		name     string
		err      *DownloadError
		target   error
		expected bool
	}{
		{
			name: "matches sentinel error by code",
			err: &DownloadError{
				Code: CodeInvalidURL,
			},
			target:   ErrInvalidURL,
			expected: true,
		},
		{
			name: "matches underlying error",
			err: &DownloadError{
				Code:       CodeNetworkError,
				Underlying: ErrNetworkError,
			},
			target:   ErrNetworkError,
			expected: true,
		},
		{
			name: "no match",
			err: &DownloadError{
				Code: CodeFileExists,
			},
			target:   ErrInvalidURL,
			expected: false,
		},
		{
			name: "matches file exists error",
			err: &DownloadError{
				Code: CodeFileExists,
			},
			target:   ErrFileExists,
			expected: true,
		},
		{
			name: "matches insufficient space error",
			err: &DownloadError{
				Code: CodeInsufficientSpace,
			},
			target:   ErrInsufficientSpace,
			expected: true,
		},
		{
			name: "matches network error",
			err: &DownloadError{
				Code: CodeNetworkError,
			},
			target:   ErrNetworkError,
			expected: true,
		},
		{
			name: "unknown code doesn't match",
			err: &DownloadError{
				Code: CodeUnknown,
			},
			target:   ErrInvalidURL,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.err.Is(tt.target); got != tt.expected {
				t.Errorf("DownloadError.Is() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewDownloadError(t *testing.T) {
	tests := []struct {
		name      string
		code      ErrorCode
		message   string
		retryable bool
	}{
		{
			name:      "retryable error",
			code:      CodeNetworkError,
			message:   "Network error occurred",
			retryable: true,
		},
		{
			name:      "non-retryable error",
			code:      CodeInvalidURL,
			message:   "Invalid URL",
			retryable: false,
		},
		{
			name:      "server error is retryable",
			code:      CodeServerError,
			message:   "Server error",
			retryable: true,
		},
		{
			name:      "client error is not retryable",
			code:      CodeClientError,
			message:   "Client error",
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := NewDownloadError(tt.code, tt.message)

			if err.Code != tt.code {
				t.Errorf("Expected code %v, got %v", tt.code, err.Code)
			}

			if err.Message != tt.message {
				t.Errorf("Expected message %v, got %v", tt.message, err.Message)
			}

			if err.Retryable != tt.retryable {
				t.Errorf("Expected retryable %v, got %v", tt.retryable, err.Retryable)
			}
		})
	}
}

func TestNewDownloadErrorWithDetails(t *testing.T) {
	code := CodePermissionDenied
	message := "Permission denied"
	details := "Cannot write to /root/file.txt"

	err := NewDownloadErrorWithDetails(code, message, details)

	if err.Code != code {
		t.Errorf("Expected code %v, got %v", code, err.Code)
	}

	if err.Message != message {
		t.Errorf("Expected message %v, got %v", message, err.Message)
	}

	if err.Details != details {
		t.Errorf("Expected details %v, got %v", details, err.Details)
	}

	if err.Retryable != false {
		t.Errorf("Permission denied should not be retryable")
	}
}

func TestWrapError(t *testing.T) {
	underlyingErr := errors.New("original error")
	code := CodeTimeout
	message := "Operation timed out"

	err := WrapError(underlyingErr, code, message)

	if err.Code != code {
		t.Errorf("Expected code %v, got %v", code, err.Code)
	}

	if err.Message != message {
		t.Errorf("Expected message %v, got %v", message, err.Message)
	}

	if !errors.Is(err.Underlying, underlyingErr) {
		t.Errorf("Expected underlying error %v, got %v", underlyingErr, err.Underlying)
	}

	if !err.Retryable {
		t.Errorf("Timeout errors should be retryable")
	}
}

func TestWrapErrorWithURL(t *testing.T) {
	underlyingErr := errors.New("connection refused")
	code := CodeNetworkError
	message := "Network error"
	testURL := "https://example.com/file.txt"

	err := WrapErrorWithURL(underlyingErr, code, message, testURL)

	if err.URL != testURL {
		t.Errorf("Expected URL %v, got %v", testURL, err.URL)
	}

	if err.Code != code {
		t.Errorf("Expected code %v, got %v", code, err.Code)
	}

	if err.Message != message {
		t.Errorf("Expected message %v, got %v", message, err.Message)
	}

	if !errors.Is(err.Underlying, underlyingErr) {
		t.Errorf("Expected underlying error %v, got %v", underlyingErr, err.Underlying)
	}
}

func TestFromHTTPStatus(t *testing.T) {
	tests := []struct {
		name         string
		statusCode   int
		expectedCode ErrorCode
		retryable    bool
	}{
		{
			name:         "500 server error",
			statusCode:   500,
			expectedCode: CodeServerError,
			retryable:    true,
		},
		{
			name:         "502 bad gateway",
			statusCode:   502,
			expectedCode: CodeServerError,
			retryable:    true,
		},
		{
			name:         "404 not found",
			statusCode:   404,
			expectedCode: CodeFileNotFound,
			retryable:    false,
		},
		{
			name:         "401 unauthorized",
			statusCode:   401,
			expectedCode: CodeAuthenticationFailed,
			retryable:    false,
		},
		{
			name:         "403 forbidden",
			statusCode:   403,
			expectedCode: CodeAuthenticationFailed,
			retryable:    false,
		},
		{
			name:         "400 bad request",
			statusCode:   400,
			expectedCode: CodeClientError,
			retryable:    false,
		},
		{
			name:         "418 teapot",
			statusCode:   418,
			expectedCode: CodeClientError,
			retryable:    false,
		},
		{
			name:         "200 ok (unexpected)",
			statusCode:   200,
			expectedCode: CodeUnknown,
			retryable:    false,
		},
	}

	testURL := "https://example.com/file.txt"

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FromHTTPStatus(tt.statusCode, testURL)

			if err.Code != tt.expectedCode {
				t.Errorf("Expected code %v, got %v", tt.expectedCode, err.Code)
			}

			if err.Retryable != tt.retryable {
				t.Errorf("Expected retryable %v, got %v", tt.retryable, err.Retryable)
			}

			if err.HTTPStatusCode != tt.statusCode {
				t.Errorf("Expected status code %v, got %v", tt.statusCode, err.HTTPStatusCode)
			}

			if err.URL != testURL {
				t.Errorf("Expected URL %v, got %v", testURL, err.URL)
			}
		})
	}
}

func TestIsRetryableByCode(t *testing.T) {
	tests := []struct {
		name      string
		code      ErrorCode
		retryable bool
	}{
		{"CodeNetworkError", CodeNetworkError, true},
		{"CodeTimeout", CodeTimeout, true},
		{"CodeServerError", CodeServerError, true},
		{"CodeInvalidURL", CodeInvalidURL, false},
		{"CodeFileExists", CodeFileExists, false},
		{"CodePermissionDenied", CodePermissionDenied, false},
		{"CodeFileNotFound", CodeFileNotFound, false},
		{"CodeAuthenticationFailed", CodeAuthenticationFailed, false},
		{"CodeClientError", CodeClientError, false},
		{"CodeCancelled", CodeCancelled, false},
		{"CodeCorruptedData", CodeCorruptedData, false},
		{"CodeInsufficientSpace", CodeInsufficientSpace, false},
		{"CodeUnknown", CodeUnknown, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableByCode(tt.code); got != tt.retryable {
				t.Errorf("isRetryableByCode(%v) = %v, want %v", tt.code, got, tt.retryable)
			}
		})
	}
}

func TestIsRetryableError(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "nil error",
			err:       nil,
			retryable: false,
		},
		{
			name:      "temporary network error",
			err:       &mockNetError{temporary: true, timeout: false},
			retryable: true,
		},
		{
			name:      "timeout network error",
			err:       &mockNetError{temporary: false, timeout: true},
			retryable: true,
		},
		{
			name:      "permanent network error",
			err:       &mockNetError{temporary: false, timeout: false},
			retryable: false,
		},
		{
			name:      "context canceled",
			err:       context.Canceled,
			retryable: false,
		},
		{
			name:      "context deadline exceeded",
			err:       context.DeadlineExceeded,
			retryable: false,
		},
		{
			name:      "generic error",
			err:       errors.New("generic error"),
			retryable: false,
		},
		{
			name:      "url error with retryable underlying",
			err:       &url.Error{Err: &mockNetError{temporary: true}},
			retryable: true,
		},
		{
			name:      "url error with non-retryable underlying",
			err:       &url.Error{Err: errors.New("parse error")},
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableError(tt.err); got != tt.retryable {
				t.Errorf("isRetryableError() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

func TestIsRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name: "DownloadError with retryable code",
			err: &DownloadError{
				Code:      CodeNetworkError,
				Retryable: true,
			},
			retryable: true,
		},
		{
			name: "DownloadError with non-retryable code",
			err: &DownloadError{
				Code:      CodeFileExists,
				Retryable: false,
			},
			retryable: false,
		},
		{
			name:      "temporary network error",
			err:       &mockNetError{temporary: true},
			retryable: true,
		},
		{
			name:      "generic error",
			err:       errors.New("generic error"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.retryable {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

func TestGetErrorCode(t *testing.T) {
	tests := []struct {
		name         string
		err          error
		expectedCode ErrorCode
	}{
		{
			name: "DownloadError",
			err: &DownloadError{
				Code: CodeTimeout,
			},
			expectedCode: CodeTimeout,
		},
		{
			name:         "generic error",
			err:          errors.New("generic error"),
			expectedCode: CodeUnknown,
		},
		{
			name:         "wrapped DownloadError",
			err:          WrapError(errors.New("underlying"), CodeNetworkError, "Network error"),
			expectedCode: CodeNetworkError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetErrorCode(tt.err); got != tt.expectedCode {
				t.Errorf("GetErrorCode() = %v, want %v", got, tt.expectedCode)
			}
		})
	}
}

func TestSentinelErrors(t *testing.T) {
	// Test that sentinel errors are not nil and have meaningful messages
	sentinelErrors := []struct {
		name string
		err  error
	}{
		{"ErrInvalidURL", ErrInvalidURL},
		{"ErrFileExists", ErrFileExists},
		{"ErrInsufficientSpace", ErrInsufficientSpace},
		{"ErrNetworkError", ErrNetworkError},
	}

	for _, tt := range sentinelErrors {
		t.Run(tt.name, func(t *testing.T) {
			if tt.err == nil {
				t.Errorf("%s should not be nil", tt.name)
			}

			if tt.err.Error() == "" {
				t.Errorf("%s should have a non-empty error message", tt.name)
			}
		})
	}
}

// mockNetError implements net.Error for testing.
type mockNetError struct {
	temporary bool
	timeout   bool
}

func (e *mockNetError) Error() string {
	return "mock network error"
}

func (e *mockNetError) Temporary() bool {
	return e.temporary
}

func (e *mockNetError) Timeout() bool {
	return e.timeout
}

func TestDownloadError_ErrorChaining(t *testing.T) {
	// Test error chain with multiple levels
	rootErr := errors.New("root cause")
	urlErr := &url.Error{Op: "Get", URL: "https://example.com", Err: rootErr}
	downloadErr := WrapError(urlErr, CodeNetworkError, "Network error occurred")

	// Test that we can unwrap to the root cause
	if !errors.Is(downloadErr, rootErr) {
		t.Error("Should be able to detect root cause error")
	}

	// Test that we can detect URL error
	var urlErrCheck *url.Error
	if !errors.As(downloadErr, &urlErrCheck) {
		t.Error("Should be able to extract URL error")
	}

	// Test that we can detect DownloadError
	var downloadErrCheck *DownloadError
	if !errors.As(downloadErr, &downloadErrCheck) {
		t.Error("Should be able to extract DownloadError")
	}
}

func TestIsRetryableError_URLErrorEdgeCases(t *testing.T) {
	// Test the uncovered path in isRetryableError for URL errors
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name: "nested url error with retryable net error",
			err: &url.Error{
				Op:  "Get",
				URL: "https://example.com",
				Err: &url.Error{
					Op:  "Dial",
					URL: "example.com:443",
					Err: &mockNetError{temporary: true, timeout: false},
				},
			},
			retryable: true,
		},
		{
			name: "nested url error with non-retryable error",
			err: &url.Error{
				Op:  "Get",
				URL: "https://example.com",
				Err: &url.Error{
					Op:  "Parse",
					URL: "invalid://url",
					Err: errors.New("invalid URL scheme"),
				},
			},
			retryable: false,
		},
		{
			name: "url error with context canceled",
			err: &url.Error{
				Op:  "Get",
				URL: "https://example.com",
				Err: context.Canceled,
			},
			retryable: false,
		},
		{
			name: "url error with context deadline exceeded",
			err: &url.Error{
				Op:  "Get",
				URL: "https://example.com",
				Err: context.DeadlineExceeded,
			},
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isRetryableError(tt.err); got != tt.retryable {
				t.Errorf("isRetryableError() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

func TestIsRetryableError_DeepRecursion(t *testing.T) {
	// Test the specific uncovered path: url error that wraps another url error
	// which wraps a non-retryable error (not net.Error)
	deeplyNestedErr := &url.Error{
		Op:  "Get",
		URL: "https://example.com",
		Err: &url.Error{
			Op:  "Dial",
			URL: "example.com:443",
			Err: &url.Error{
				Op:  "Connect",
				URL: "192.168.1.1:443",
				Err: errors.New("connection refused - final error"),
			},
		},
	}

	// This should trigger the recursive path in isRetryableError
	// where urlErr.Err is another *url.Error, causing recursive call
	result := isRetryableError(deeplyNestedErr)

	if result {
		t.Error("Expected non-retryable for deeply nested error ending in non-net.Error")
	}
}

func TestIsRetryableError_CompleteCodeCoverage(t *testing.T) {
	// Test to ensure we hit the uncovered line in isRetryableError
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name: "url error wrapping url error with generic error",
			err: &url.Error{
				Op:  "Get",
				URL: "https://test.com",
				Err: &url.Error{
					Op:  "Parse",
					URL: "invalid",
					Err: errors.New("generic parse error"),
				},
			},
			retryable: false,
		},
		{
			name: "url error wrapping url error with net error (timeout)",
			err: &url.Error{
				Op:  "Get",
				URL: "https://test.com",
				Err: &url.Error{
					Op:  "Dial",
					URL: "test.com:443",
					Err: &mockNetError{timeout: true},
				},
			},
			retryable: true,
		},
		{
			name: "url error wrapping url error with net error (temporary)",
			err: &url.Error{
				Op:  "Get",
				URL: "https://test.com",
				Err: &url.Error{
					Op:  "Dial",
					URL: "test.com:443",
					Err: &mockNetError{temporary: true},
				},
			},
			retryable: true,
		},
		{
			name: "triple nested url error with context.Canceled at end",
			err: &url.Error{
				Op:  "Get",
				URL: "https://test.com",
				Err: &url.Error{
					Op:  "Dial",
					URL: "test.com:443",
					Err: &url.Error{
						Op:  "Connect",
						URL: "192.168.1.1",
						Err: context.Canceled,
					},
				},
			},
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRetryableError(tt.err)
			if result != tt.retryable {
				t.Errorf("isRetryableError() = %v, want %v", result, tt.retryable)
			}
		})
	}
}

func TestIsRetryableError_URLErrorRecursivePath(t *testing.T) {
	// The uncovered path is specifically when we have a URL error that is NOT caught
	// by the net.Error check first. This means the URL error contains a non-net.Error.

	// Test 1: URL error with a simple error (not net.Error) - should hit the URL error path
	urlErrWithSimpleError := &url.Error{
		Op:  "Parse",
		URL: "invalid-url",
		Err: errors.New("parse error"), // This is NOT a net.Error
	}

	result1 := isRetryableError(urlErrWithSimpleError)
	if result1 {
		t.Error("Expected false for URL error with simple parse error")
	}

	// Test 2: URL error with context.Canceled - should hit URL error path and then context check
	urlErrWithContext := &url.Error{
		Op:  "Get",
		URL: "https://example.com",
		Err: context.Canceled, // This is NOT a net.Error
	}

	result2 := isRetryableError(urlErrWithContext)
	if result2 {
		t.Error("Expected false for URL error with context.Canceled")
	}
}

func TestIsNetworkRetryable(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		retryable bool
	}{
		{
			name:      "connection refused",
			err:       errors.New("connection refused"),
			retryable: true,
		},
		{
			name:      "connection reset by peer",
			err:       errors.New("connection reset by peer"),
			retryable: true,
		},
		{
			name:      "connection timeout",
			err:       errors.New("connection timeout"),
			retryable: true,
		},
		{
			name:      "i/o timeout",
			err:       errors.New("i/o timeout"),
			retryable: true,
		},
		{
			name:      "network is unreachable",
			err:       errors.New("network is unreachable"),
			retryable: true,
		},
		{
			name:      "no route to host",
			err:       errors.New("no route to host"),
			retryable: true,
		},
		{
			name:      "broken pipe",
			err:       errors.New("broken pipe"),
			retryable: true,
		},
		{
			name:      "connection aborted",
			err:       errors.New("connection aborted"),
			retryable: true,
		},
		{
			name:      "Connection Refused (case insensitive)",
			err:       errors.New("Connection Refused"),
			retryable: true,
		},
		{
			name:      "CONNECTION RESET (case insensitive)",
			err:       errors.New("CONNECTION RESET"),
			retryable: true,
		},
		{
			name:      "non-retryable error",
			err:       errors.New("invalid request format"),
			retryable: false,
		},
		{
			name:      "parse error",
			err:       errors.New("parse error"),
			retryable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isNetworkRetryable(tt.err); got != tt.retryable {
				t.Errorf("isNetworkRetryable() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

func TestErrorCodeRegistry(t *testing.T) {
	registry := NewErrorCodeRegistry()

	// Test GetNetworkCodeMessage
	tests := []struct {
		name        string
		code        NetworkErrorCode
		expectedMsg string
	}{
		{"DNS not found", NetworkDNSNotFound, "Domain name could not be resolved"},
		{"DNS timeout", NetworkDNSTimeout, "DNS resolution timed out"},
		{"Connection refused", NetworkConnectionRefused, "Connection was refused by the server"},
		{"Connection timeout", NetworkConnectionTimeout, "Connection attempt timed out"},
		{"Connection reset", NetworkConnectionReset, "Connection was reset by the server"},
		{"TLS handshake failure", NetworkTLSHandshakeFailure, "TLS/SSL handshake failed"},
		{"Invalid network code", NetworkErrorCode("invalid_network_code"), "Unknown network error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := registry.GetNetworkCodeMessage(tt.code)
			if msg != tt.expectedMsg {
				t.Errorf("GetNetworkCodeMessage() = %v, want %v", msg, tt.expectedMsg)
			}
		})
	}

	// Test GetHTTPCodeMessage
	httpTests := []struct {
		name        string
		code        HTTPErrorCode
		expectedMsg string
	}{
		{"Bad request", HTTPBadRequest, "Bad request - invalid syntax"},
		{"Not found", HTTPNotFound, "Resource not found"},
		{"Internal server error", HTTPInternalServerError, "Internal server error"},
		{"Service unavailable", HTTPServiceUnavailable, "Service temporarily unavailable"},
		{"Invalid HTTP code", HTTPErrorCode("invalid_http_code"), "Unknown HTTP error"},
	}

	for _, tt := range httpTests {
		t.Run(tt.name, func(t *testing.T) {
			msg := registry.GetHTTPCodeMessage(tt.code)
			if msg != tt.expectedMsg {
				t.Errorf("GetHTTPCodeMessage() = %v, want %v", msg, tt.expectedMsg)
			}
		})
	}

	// Test GetFileSystemCodeMessage
	fsTests := []struct {
		name        string
		code        FileSystemErrorCode
		expectedMsg string
	}{
		{"Permission denied", FSPermissionDenied, "Permission denied"},
		{"File not found", FSFileNotFound, "File not found"},
		{"Insufficient space", FSInsufficientSpace, "Insufficient disk space"},
		{"File exists", FSFileExists, "File already exists"},
		{"Invalid FS code", FileSystemErrorCode("invalid_fs_code"), "Unknown file system error"},
	}

	for _, tt := range fsTests {
		t.Run(tt.name, func(t *testing.T) {
			msg := registry.GetFileSystemCodeMessage(tt.code)
			if msg != tt.expectedMsg {
				t.Errorf("GetFileSystemCodeMessage() = %v, want %v", msg, tt.expectedMsg)
			}
		})
	}

	// Test GetValidationCodeMessage
	validationTests := []struct {
		name        string
		code        ValidationErrorCode
		expectedMsg string
	}{
		{"Invalid URL", ValidationInvalidURL, "Invalid URL format"},
		{"Unsupported scheme", ValidationUnsupportedScheme, "Unsupported URL scheme"},
		{"Invalid parameter", ValidationInvalidParameter, "Invalid parameter"},
		{"Missing parameter", ValidationMissingParameter, "Missing required parameter"},
		{
			"Invalid validation code",
			ValidationErrorCode("invalid_validation_code"),
			"Unknown validation error",
		},
	}

	for _, tt := range validationTests {
		t.Run(tt.name, func(t *testing.T) {
			msg := registry.GetValidationCodeMessage(tt.code)
			if msg != tt.expectedMsg {
				t.Errorf("GetValidationCodeMessage() = %v, want %v", msg, tt.expectedMsg)
			}
		})
	}
}

func TestErrorCodeRegistry_GetHTTPCodeFromStatus(t *testing.T) {
	registry := NewErrorCodeRegistry()

	tests := []struct {
		name         string
		statusCode   int
		expectedCode HTTPErrorCode
	}{
		{"400 Bad Request", 400, HTTPBadRequest},
		{"401 Unauthorized", 401, HTTPUnauthorized},
		{"404 Not Found", 404, HTTPNotFound},
		{"500 Internal Server Error", 500, HTTPInternalServerError},
		{"502 Bad Gateway", 502, HTTPBadGateway},
		{"503 Service Unavailable", 503, HTTPServiceUnavailable},
		{"999 Invalid status", 999, HTTPErrorCode("HTTP_999_UNKNOWN")},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			code := registry.GetHTTPCodeFromStatus(tt.statusCode)
			if code != tt.expectedCode {
				t.Errorf("GetHTTPCodeFromStatus() code = %v, want %v", code, tt.expectedCode)
			}
		})
	}
}

func TestErrorCodeRegistry_IsRetryableNetworkCode(t *testing.T) {
	registry := NewErrorCodeRegistry()

	tests := []struct {
		name      string
		code      NetworkErrorCode
		retryable bool
	}{
		{"DNS timeout (retryable)", NetworkDNSTimeout, true},
		{"Connection timeout (retryable)", NetworkConnectionTimeout, true},
		{"Connection refused (retryable)", NetworkConnectionRefused, true},
		{"Connection reset (retryable)", NetworkConnectionReset, true},
		{"Host unreachable (retryable)", NetworkHostUnreachable, true},
		{"Network unreachable (retryable)", NetworkNetworkUnreachable, true},
		{"Read timeout (retryable)", NetworkReadTimeout, true},
		{"Write timeout (retryable)", NetworkWriteTimeout, true},
		{"Request timeout (retryable)", NetworkRequestTimeout, true},
		{"DNS not found (not retryable)", NetworkDNSNotFound, false},
		{"DNS failure (not retryable)", NetworkDNSFailure, false},
		{"TLS handshake failure (not retryable)", NetworkTLSHandshakeFailure, false},
		{"Protocol error (not retryable)", NetworkProtocolError, false},
		{"Proxy error (not retryable)", NetworkProxyError, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := registry.IsRetryableNetworkCode(tt.code); got != tt.retryable {
				t.Errorf("IsRetryableNetworkCode() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

func TestErrorCodeRegistry_IsRetryableHTTPCode(t *testing.T) {
	registry := NewErrorCodeRegistry()

	tests := []struct {
		name      string
		code      HTTPErrorCode
		retryable bool
	}{
		{"Request timeout (retryable)", HTTPRequestTimeout, true},
		{"Too many requests (retryable)", HTTPTooManyRequests, true},
		{"Internal server error (retryable)", HTTPInternalServerError, true},
		{"Bad gateway (retryable)", HTTPBadGateway, true},
		{"Service unavailable (retryable)", HTTPServiceUnavailable, true},
		{"Gateway timeout (retryable)", HTTPGatewayTimeout, true},
		{"Insufficient storage (retryable)", HTTPInsufficientStorage, true},
		{"Bad request (not retryable)", HTTPBadRequest, false},
		{"Unauthorized (not retryable)", HTTPUnauthorized, false},
		{"Forbidden (not retryable)", HTTPForbidden, false},
		{"Not found (not retryable)", HTTPNotFound, false},
		{"Method not allowed (not retryable)", HTTPMethodNotAllowed, false},
		{"Conflict (not retryable)", HTTPConflict, false},
		{"Gone (not retryable)", HTTPGone, false},
		{"Range not satisfiable (not retryable)", HTTPRangeNotSatisfiable, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := registry.IsRetryableHTTPCode(tt.code); got != tt.retryable {
				t.Errorf("IsRetryableHTTPCode() = %v, want %v", got, tt.retryable)
			}
		})
	}
}

func TestErrorHandler(t *testing.T) {
	handler := NewErrorHandler(true)

	// Test HandleError
	err := NewDownloadError(CodeNetworkError, "Connection failed")

	handled := handler.HandleError(err, "test context")
	if handled == nil {
		t.Error("HandleError should return non-nil error")
	}

	// Test FormatError
	formatted := handler.FormatError(err)
	if formatted == "" {
		t.Error("FormatError should return non-empty string")
	}

	// Test with different error types
	tests := []struct {
		name string
		err  error
	}{
		{"Download error", NewDownloadError(CodeTimeout, "Request timeout")},
		{"Generic error", errors.New("generic error")},
		{
			"Network error with details",
			NewDownloadErrorWithDetails(
				CodeNetworkError,
				"Connection failed",
				"Port 443 unreachable",
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			formatted := handler.FormatError(tt.err)
			if formatted == "" {
				t.Error("FormatError should return non-empty string")
			}
		})
	}
}

func TestErrorHandler_AggregateErrors(t *testing.T) {
	handler := NewErrorHandler(true)

	// Test with nil slice
	result := handler.AggregateErrors(nil)
	if result != nil {
		t.Error("AggregateErrors with nil should return nil")
	}

	// Test with empty slice
	result = handler.AggregateErrors([]error{})
	if result != nil {
		t.Error("AggregateErrors with empty slice should return nil")
	}

	// Test with single error
	singleErr := errors.New("single error")

	result = handler.AggregateErrors([]error{singleErr})
	if !errors.Is(result, singleErr) {
		t.Error("AggregateErrors with single error should return that error")
	}

	// Test with multiple errors
	err1 := NewDownloadError(CodeNetworkError, "Network error")
	err2 := NewDownloadError(CodeTimeout, "Timeout error")
	err3 := errors.New("Generic error")

	result = handler.AggregateErrors([]error{err1, err2, err3})
	if result == nil {
		t.Error("AggregateErrors with multiple errors should return non-nil")
	}

	// Should contain all error messages
	resultStr := result.Error()
	if !contains(resultStr, "Network error") || !contains(resultStr, "Timeout error") ||
		!contains(resultStr, "Generic error") {
		t.Error("Aggregated error should contain all individual error messages")
	}
}

func TestRetryStrategy(t *testing.T) {
	// Test DefaultRetryStrategy
	defaultStrategy := DefaultRetryStrategy()
	if defaultStrategy.MaxAttempts < 1 {
		t.Error("Default strategy should have MaxAttempts > 0")
	}

	if defaultStrategy.InitialDelay <= 0 {
		t.Error("Default strategy should have InitialDelay > 0")
	}

	// Test ExponentialBackoffStrategy
	expStrategy := ExponentialBackoffStrategy(5)
	if expStrategy.MaxAttempts != 5 {
		t.Error("ExponentialBackoffStrategy should set MaxAttempts correctly")
	}

	if expStrategy.InitialDelay <= 0 {
		t.Error("ExponentialBackoffStrategy should have InitialDelay > 0")
	}

	if expStrategy.Multiplier != 2.0 {
		t.Error("ExponentialBackoffStrategy should set Multiplier correctly")
	}
}

func TestRetryManager(t *testing.T) {
	strategy := &RetryStrategy{
		MaxAttempts:    3,
		InitialDelay:   10 * time.Millisecond,
		MaxDelay:       100 * time.Millisecond,
		Multiplier:     2.0,
		Jitter:         false,
		RetryCondition: IsRetryable,
	}

	manager := NewRetryManager(strategy)

	// Test successful execution (no retries needed)
	var callCount int

	err := manager.Execute(context.Background(), func() error {
		callCount++
		return nil
	})
	if err != nil {
		t.Errorf("Execute should succeed on first try, got error: %v", err)
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call, got %d", callCount)
	}

	// Test retryable error
	callCount = 0

	err = manager.Execute(context.Background(), func() error {
		callCount++
		if callCount < 3 {
			return NewDownloadError(CodeNetworkError, "Temporary network error")
		}

		return nil
	})
	if err != nil {
		t.Errorf("Execute should succeed after retries, got error: %v", err)
	}

	if callCount != 3 {
		t.Errorf("Expected 3 calls, got %d", callCount)
	}

	// Test non-retryable error
	callCount = 0

	err = manager.Execute(context.Background(), func() error {
		callCount++
		return NewDownloadError(CodeInvalidURL, "Invalid URL")
	})
	if err == nil {
		t.Error("Execute should fail for non-retryable error")
	}

	if callCount != 1 {
		t.Errorf("Expected 1 call for non-retryable error, got %d", callCount)
	}

	// Test max retries exceeded
	callCount = 0

	err = manager.Execute(context.Background(), func() error {
		callCount++
		return NewDownloadError(CodeNetworkError, "Persistent network error")
	})
	if err == nil {
		t.Error("Execute should fail when max retries exceeded")
	}

	if callCount != strategy.MaxAttempts {
		t.Errorf("Expected %d calls, got %d", strategy.MaxAttempts, callCount)
	}
}

// Helper function to check if string contains substring.
func contains(s, substr string) bool {
	return strings.Contains(s, substr)
}
