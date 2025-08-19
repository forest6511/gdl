// Package errors defines custom error types and sentinel errors for the gdl download library.
package errors

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"strings"
)

// Sentinel errors for common download scenarios.
// These can be used with errors.Is() for error comparison.
var (
	// ErrInvalidURL is returned when a provided URL is malformed or invalid.
	ErrInvalidURL = errors.New("invalid URL provided")

	// ErrFileExists is returned when attempting to download to a file that already exists
	// and overwrite is not enabled.
	ErrFileExists = errors.New("file already exists")

	// ErrInsufficientSpace is returned when there is not enough disk space to complete
	// the download operation.
	ErrInsufficientSpace = errors.New("insufficient disk space")

	// ErrNetworkError is returned for general network-related errors during download.
	ErrNetworkError = errors.New("network error occurred")
)

// ErrorCode represents different types of errors that can occur during downloads.
const (
	unknownValue = "unknown"
)

type ErrorCode int

const (
	// CodeUnknown represents an unknown or unclassified error.
	CodeUnknown ErrorCode = iota

	// CodeInvalidURL represents errors related to malformed or invalid URLs.
	CodeInvalidURL

	// CodeFileExists represents errors when target file already exists.
	CodeFileExists

	// CodeInsufficientSpace represents errors due to lack of disk space.
	CodeInsufficientSpace

	// CodeNetworkError represents network-related errors.
	CodeNetworkError

	// CodeTimeout represents timeout errors during download.
	CodeTimeout

	// CodePermissionDenied represents permission-related errors.
	CodePermissionDenied

	// CodeFileNotFound represents errors when source file is not found.
	CodeFileNotFound

	// CodeAuthenticationFailed represents authentication or authorization errors.
	CodeAuthenticationFailed

	// CodeServerError represents server-side errors (5xx HTTP status codes).
	CodeServerError

	// CodeClientError represents client-side errors (4xx HTTP status codes).
	CodeClientError

	// CodeCancelled represents errors when download is cancelled by user.
	CodeCancelled

	// CodeCorruptedData represents errors when downloaded data is corrupted.
	CodeCorruptedData
)

// String returns a string representation of the error code.
func (c ErrorCode) String() string {
	switch c {
	case CodeUnknown:
		return "unknown"
	case CodeInvalidURL:
		return "invalid_url"
	case CodeFileExists:
		return "file_exists"
	case CodeInsufficientSpace:
		return "insufficient_space"
	case CodeNetworkError:
		return "network_error"
	case CodeTimeout:
		return "timeout"
	case CodePermissionDenied:
		return "permission_denied"
	case CodeFileNotFound:
		return "file_not_found"
	case CodeAuthenticationFailed:
		return "authentication_failed"
	case CodeServerError:
		return "server_error"
	case CodeClientError:
		return "client_error"
	case CodeCancelled:
		return "cancelled"
	case CodeCorruptedData:
		return "corrupted_data"
	default:
		return unknownValue
	}
}

// DownloadError represents a structured error that occurs during download operations.
// It provides detailed information about what went wrong, including user-friendly messages
// and technical details for debugging.
type DownloadError struct {
	// Code represents the type of error that occurred.
	Code ErrorCode

	// Message is a user-friendly error message that can be displayed to end users.
	Message string

	// Details contains technical details about the error for debugging purposes.
	Details string

	// URL is the source URL that caused the error, if applicable.
	URL string

	// Filename is the target filename that was being written to, if applicable.
	Filename string

	// Underlying is the original error that caused this download error.
	Underlying error

	// Retryable indicates whether this error condition might succeed if retried.
	Retryable bool

	// HTTPStatusCode contains the HTTP status code if the error is HTTP-related.
	HTTPStatusCode int

	// BytesTransferred indicates how many bytes were successfully transferred
	// before the error occurred.
	BytesTransferred int64
}

// Error implements the error interface for DownloadError.
func (e *DownloadError) Error() string {
	if e.Message != "" {
		return e.Message
	}

	if e.Underlying != nil {
		return e.Underlying.Error()
	}

	return "download error occurred"
}

// Unwrap returns the underlying error for error unwrapping support.
// This allows the use of errors.Is() and errors.As() with DownloadError.
func (e *DownloadError) Unwrap() error {
	return e.Underlying
}

// Is implements error comparison for DownloadError.
// This allows checking if a DownloadError wraps a specific sentinel error.
func (e *DownloadError) Is(target error) bool {
	if e.Underlying != nil && errors.Is(e.Underlying, target) {
		return true
	}

	// Check against sentinel errors based on error code
	switch e.Code {
	case CodeInvalidURL:
		return errors.Is(target, ErrInvalidURL)
	case CodeFileExists:
		return errors.Is(target, ErrFileExists)
	case CodeInsufficientSpace:
		return errors.Is(target, ErrInsufficientSpace)
	case CodeNetworkError:
		return errors.Is(target, ErrNetworkError)
	}

	return false
}

// NewDownloadError creates a new DownloadError with the specified code and message.
func NewDownloadError(code ErrorCode, message string) *DownloadError {
	return &DownloadError{
		Code:      code,
		Message:   message,
		Retryable: isRetryableByCode(code),
	}
}

// NewDownloadErrorWithDetails creates a new DownloadError with code, message, and technical details.
func NewDownloadErrorWithDetails(code ErrorCode, message, details string) *DownloadError {
	return &DownloadError{
		Code:      code,
		Message:   message,
		Details:   details,
		Retryable: isRetryableByCode(code),
	}
}

// WrapError wraps an existing error as a DownloadError with additional context.
func WrapError(underlying error, code ErrorCode, message string) *DownloadError {
	return &DownloadError{
		Code:       code,
		Message:    message,
		Underlying: underlying,
		Retryable:  isRetryableByCode(code) || isRetryableError(underlying),
	}
}

// WrapErrorWithURL wraps an existing error as a DownloadError with URL context.
func WrapErrorWithURL(underlying error, code ErrorCode, message, url string) *DownloadError {
	return &DownloadError{
		Code:       code,
		Message:    message,
		URL:        url,
		Underlying: underlying,
		Retryable:  isRetryableByCode(code) || isRetryableError(underlying),
	}
}

// FromHTTPStatus creates a DownloadError based on an HTTP status code.
func FromHTTPStatus(statusCode int, url string) *DownloadError {
	var (
		code      ErrorCode
		message   string
		retryable bool
	)

	switch {
	case statusCode >= 500:
		code = CodeServerError
		message = fmt.Sprintf("Server error (HTTP %d)", statusCode)
		retryable = true
	case statusCode == 404:
		code = CodeFileNotFound
		message = "File not found on server"
		retryable = false
	case statusCode == 401 || statusCode == 403:
		code = CodeAuthenticationFailed
		message = "Authentication or authorization failed"
		retryable = false
	case statusCode >= 400:
		code = CodeClientError
		message = fmt.Sprintf("Client error (HTTP %d)", statusCode)
		retryable = false
	default:
		code = CodeUnknown
		message = fmt.Sprintf("Unexpected HTTP status: %d", statusCode)
		retryable = false
	}

	return &DownloadError{
		Code:           code,
		Message:        message,
		URL:            url,
		Retryable:      retryable,
		HTTPStatusCode: statusCode,
	}
}

// isRetryableByCode determines if an error code represents a retryable condition.
func isRetryableByCode(code ErrorCode) bool {
	switch code {
	case CodeNetworkError, CodeTimeout, CodeServerError:
		return true
	case CodeInvalidURL, CodeFileExists, CodePermissionDenied,
		CodeFileNotFound, CodeAuthenticationFailed, CodeClientError,
		CodeCancelled, CodeCorruptedData:
		return false
	case CodeInsufficientSpace:
		return false // Usually not retryable without user intervention
	default:
		return false
	}
}

// isRetryableError checks if a standard Go error represents a retryable condition.
// isNetworkRetryable determines if a network error is retryable based on error patterns.
func isNetworkRetryable(err error) bool {
	errStr := err.Error()
	// Common retryable network error patterns
	retryablePatterns := []string{
		"connection refused",
		"connection reset",
		"connection timeout",
		"i/o timeout",
		"network is unreachable",
		"no route to host",
		"broken pipe",
		"connection aborted",
	}

	for _, pattern := range retryablePatterns {
		if strings.Contains(strings.ToLower(errStr), pattern) {
			return true
		}
	}

	return false
}

func isRetryableError(err error) bool {
	if err == nil {
		return false
	}

	// Context errors are not retryable - they represent user cancellation or timeout
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}

	// Network errors are generally retryable
	var netErr net.Error
	if errors.As(err, &netErr) {
		// Check timeout first
		if netErr.Timeout() {
			return true
		}
		// For testing compatibility, check if it has a Temporary method and call it
		// This handles both real network errors and test mocks
		type temporaryError interface {
			Temporary() bool
		}
		if tempErr, ok := netErr.(temporaryError); ok {
			return tempErr.Temporary()
		}
		// Fallback to pattern-based detection for real network errors
		return isNetworkRetryable(err)
	}

	// URL parse errors are not retryable
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return isRetryableError(urlErr.Err)
	}

	return false
}

// IsRetryable is a convenience function to check if any error is retryable.
func IsRetryable(err error) bool {
	var downloadErr *DownloadError
	if errors.As(err, &downloadErr) {
		return downloadErr.Retryable
	}

	return isRetryableError(err)
}

// GetErrorCode extracts the error code from any error, returning CodeUnknown
// if the error is not a DownloadError.
func GetErrorCode(err error) ErrorCode {
	var downloadErr *DownloadError
	if errors.As(err, &downloadErr) {
		return downloadErr.Code
	}

	return CodeUnknown
}
