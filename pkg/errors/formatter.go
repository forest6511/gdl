// Package errors provides formatting utilities for displaying errors to users.
package errors

import (
	"errors"
	"fmt"
	"strings"
)

// FormatForCLI formats an error for command-line interface display.
// It returns a user-friendly error message suitable for terminal output.
// If the error is not a DownloadError, it returns the error's Error() string.
func FormatForCLI(err error) string {
	if err == nil {
		return ""
	}

	var downloadErr *DownloadError
	if !AsDownloadError(err, &downloadErr) {
		return err.Error()
	}

	var parts []string

	// Add error type
	parts = append(parts, fmt.Sprintf("Error [%s]:", downloadErr.Code.String()))

	// Add main message
	if downloadErr.Message != "" {
		parts = append(parts, downloadErr.Message)
	}

	// Add URL if present
	if downloadErr.URL != "" {
		parts = append(parts, fmt.Sprintf("URL: %s", downloadErr.URL))
	}

	// Add filename if present
	if downloadErr.Filename != "" {
		parts = append(parts, fmt.Sprintf("File: %s", downloadErr.Filename))
	}

	// Add retryability info
	if downloadErr.Retryable {
		parts = append(parts, "(retryable)")
	}

	return strings.Join(parts, " ")
}

// GetUserMessage extracts a user-friendly message from any error.
// For DownloadError, it returns the Message field.
// For other errors, it returns the Error() string.
func GetUserMessage(err error) string {
	if err == nil {
		return ""
	}

	var downloadErr *DownloadError
	if AsDownloadError(err, &downloadErr) && downloadErr.Message != "" {
		return downloadErr.Message
	}

	return err.Error()
}

// FormatWithDetails formats an error with technical details for debugging.
// This includes the error message, details, underlying error, and other metadata.
func FormatWithDetails(err error) string {
	if err == nil {
		return ""
	}

	var downloadErr *DownloadError
	if !AsDownloadError(err, &downloadErr) {
		return err.Error()
	}

	var builder strings.Builder

	// Error header
	builder.WriteString(fmt.Sprintf("=== DownloadError [%s] ===\n", downloadErr.Code.String()))

	// Message
	if downloadErr.Message != "" {
		builder.WriteString(fmt.Sprintf("Message: %s\n", downloadErr.Message))
	}

	// Details
	if downloadErr.Details != "" {
		builder.WriteString(fmt.Sprintf("Details: %s\n", downloadErr.Details))
	}

	// URL
	if downloadErr.URL != "" {
		builder.WriteString(fmt.Sprintf("URL: %s\n", downloadErr.URL))
	}

	// Filename
	if downloadErr.Filename != "" {
		builder.WriteString(fmt.Sprintf("Filename: %s\n", downloadErr.Filename))
	}

	// HTTP Status
	if downloadErr.HTTPStatusCode > 0 {
		builder.WriteString(fmt.Sprintf("HTTP Status: %d\n", downloadErr.HTTPStatusCode))
	}

	// Bytes Transferred
	if downloadErr.BytesTransferred > 0 {
		builder.WriteString(fmt.Sprintf("Bytes Transferred: %d\n", downloadErr.BytesTransferred))
	}

	// Retryable
	builder.WriteString(fmt.Sprintf("Retryable: %v\n", downloadErr.Retryable))

	// Underlying error
	if downloadErr.Underlying != nil {
		builder.WriteString(fmt.Sprintf("Underlying: %v\n", downloadErr.Underlying))
	}

	return builder.String()
}

// AsDownloadError is a convenience wrapper around errors.As for DownloadError.
// It returns true if the error is or wraps a DownloadError.
// This properly handles errors.Join and other complex error wrapping patterns.
func AsDownloadError(err error, target **DownloadError) bool {
	if err == nil {
		return false
	}
	return errors.As(err, target)
}

// IsDownloadError checks if an error is or contains a DownloadError.
func IsDownloadError(err error) bool {
	if err == nil {
		return false
	}
	var downloadErr *DownloadError
	return AsDownloadError(err, &downloadErr)
}

// SuggestAction returns a suggested action for the user based on the error code.
func SuggestAction(err error) string {
	if err == nil {
		return ""
	}

	code := GetErrorCode(err)

	switch code {
	case CodeInvalidURL:
		return "Please check the URL format and try again."
	case CodeFileExists:
		return "Use --overwrite flag to replace the existing file."
	case CodeInsufficientSpace:
		return "Free up disk space and try again."
	case CodeNetworkError, CodeTimeout:
		return "Check your network connection and try again."
	case CodePermissionDenied:
		return "Check file permissions or run with appropriate privileges."
	case CodeFileNotFound:
		return "Verify the URL or file path is correct."
	case CodeAuthenticationFailed:
		return "Check your credentials and authentication settings."
	case CodeServerError:
		return "The server is experiencing issues. Try again later."
	case CodeClientError:
		return "Check your request parameters and try again."
	case CodeCancelled:
		return "Operation was cancelled."
	case CodeCorruptedData:
		return "The downloaded data may be corrupted. Try downloading again."
	case CodeInvalidPath:
		return "Check the file path and ensure it is valid."
	case CodePluginError:
		return "Check plugin configuration and compatibility."
	case CodeConfigError:
		return "Review your configuration settings."
	case CodeValidationError:
		return "Check your input values and try again."
	case CodeStorageError:
		return "Check storage configuration and availability."
	default:
		return "Please try again or contact support."
	}
}
