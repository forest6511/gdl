package errors

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorHandler provides utilities for error handling and formatting.
type ErrorHandler struct {
	verboseMode bool
}

// NewErrorHandler creates a new error handler.
func NewErrorHandler(verbose bool) *ErrorHandler {
	return &ErrorHandler{
		verboseMode: verbose,
	}
}

// HandleError processes an error and returns a DownloadError.
func (h *ErrorHandler) HandleError(err error, context string) *DownloadError {
	if err == nil {
		return nil
	}

	// Check if it's already a DownloadError
	downloadErr := &DownloadError{}
	if errors.As(err, &downloadErr) {
		return downloadErr
	}

	// Wrap the error as a DownloadError
	return WrapError(err, CodeUnknown, context)
}

// FormatError formats an error for display to the user.
func (h *ErrorHandler) FormatError(err error) string {
	if err == nil {
		return ""
	}

	downloadErr := &DownloadError{}
	if errors.As(err, &downloadErr) {
		return h.formatDownloadError(downloadErr)
	}

	return err.Error()
}

// formatDownloadError formats a DownloadError for display.
func (h *ErrorHandler) formatDownloadError(err *DownloadError) string {
	var sb strings.Builder

	// Error type and message
	sb.WriteString(fmt.Sprintf("❌ [%s Error] %s\n", err.Code.String(), err.Message))

	// Additional details if verbose
	if h.verboseMode {
		if err.Details != "" {
			sb.WriteString(fmt.Sprintf("   Details: %s\n", err.Details))
		}

		if err.URL != "" {
			sb.WriteString(fmt.Sprintf("   URL: %s\n", err.URL))
		}

		if err.Filename != "" {
			sb.WriteString(fmt.Sprintf("   File: %s\n", err.Filename))
		}

		if err.HTTPStatusCode > 0 {
			sb.WriteString(fmt.Sprintf("   HTTP Status: %d\n", err.HTTPStatusCode))
		}

		if err.Underlying != nil {
			sb.WriteString(fmt.Sprintf("   Cause: %v\n", err.Underlying))
		}
	}

	// Suggestion based on error code
	suggestion := h.getSuggestion(err.Code)
	if suggestion != "" {
		sb.WriteString(fmt.Sprintf("\n💡 Suggestion: %s\n", suggestion))
	}

	// Retry information
	if err.Retryable {
		sb.WriteString("\n🔄 This error may be temporary. Retrying might help.\n")
	}

	return sb.String()
}

// getSuggestion returns a helpful suggestion based on error code.
func (h *ErrorHandler) getSuggestion(code ErrorCode) string {
	switch code {
	case CodeInvalidURL:
		return "Check that the URL is correct and properly formatted"
	case CodeFileExists:
		return "Use -f or --force to overwrite existing files"
	case CodeInsufficientSpace:
		return "Free up disk space and try again"
	case CodeNetworkError:
		return "Check your internet connection and try again"
	case CodeTimeout:
		return "Try increasing the timeout with --timeout flag"
	case CodePermissionDenied:
		return "Check file permissions or try a different output location"
	case CodeFileNotFound:
		return "Ensure the parent directory exists"
	case CodeAuthenticationFailed:
		return "Check your credentials or authentication"
	case CodeServerError:
		return "The server is experiencing issues. Try again later"
	case CodeClientError:
		return "Check your request parameters"
	case CodeCorruptedData:
		return "The download may be corrupted. Try downloading again"
	default:
		return ""
	}
}

// AggregateErrors combines multiple errors into a single error message.
func (h *ErrorHandler) AggregateErrors(errors []error) error {
	if len(errors) == 0 {
		return nil
	}

	if len(errors) == 1 {
		return errors[0]
	}

	var messages []string
	for i, err := range errors {
		messages = append(messages, fmt.Sprintf("%d. %s", i+1, h.FormatError(err)))
	}

	return fmt.Errorf("multiple errors occurred:\n%s", strings.Join(messages, "\n"))
}
