package errors

import (
	"errors"
	"strings"
	"testing"
)

// TestNewInvalidPathError tests the NewInvalidPathError helper function
func TestNewInvalidPathError(t *testing.T) {
	t.Run("WithoutUnderlyingError", func(t *testing.T) {
		path := "/invalid/path/../../../etc/passwd"

		err := NewInvalidPathError(path, nil)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Code != CodeInvalidPath {
			t.Errorf("Expected CodeInvalidPath, got: %s", err.Code)
		}

		if !strings.Contains(err.Message, "invalid file path") {
			t.Errorf("Expected message to contain 'invalid file path', got: %s", err.Message)
		}

		if !strings.Contains(err.Message, path) {
			t.Errorf("Expected message to contain path, got: %s", err.Message)
		}

		if err.Details != path {
			t.Errorf("Expected details to be path, got: %s", err.Details)
		}

		if err.Retryable {
			t.Error("Expected Retryable to be false for path errors")
		}

		if err.Underlying != nil {
			t.Error("Expected no underlying error")
		}
	})

	t.Run("WithUnderlyingError", func(t *testing.T) {
		path := "/invalid/path"
		underlyingErr := errors.New("permission denied")

		err := NewInvalidPathError(path, underlyingErr)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Code != CodeInvalidPath {
			t.Errorf("Expected CodeInvalidPath, got: %s", err.Code)
		}

		if err.Underlying != underlyingErr {
			t.Errorf("Expected underlying error to be preserved, got: %v", err.Underlying)
		}

		if err.Retryable {
			t.Error("Expected Retryable to be false even with underlying error")
		}
	})
}

// TestNewPluginError tests the NewPluginError helper function
func TestNewPluginError(t *testing.T) {
	t.Run("WithoutUnderlyingError", func(t *testing.T) {
		pluginName := "test-plugin"
		details := "plugin initialization failed"

		err := NewPluginError(pluginName, nil, details)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Code != CodePluginError {
			t.Errorf("Expected CodePluginError, got: %s", err.Code)
		}

		if !strings.Contains(err.Message, "plugin error") {
			t.Errorf("Expected message to contain 'plugin error', got: %s", err.Message)
		}

		if !strings.Contains(err.Message, pluginName) {
			t.Errorf("Expected message to contain plugin name, got: %s", err.Message)
		}

		if err.Details != details {
			t.Errorf("Expected details: %s, got: %s", details, err.Details)
		}

		if err.Retryable {
			t.Error("Expected Retryable to be false without underlying error")
		}
	})

	t.Run("WithNonRetryableUnderlyingError", func(t *testing.T) {
		pluginName := "test-plugin"
		details := "plugin load failed"
		underlyingErr := errors.New("file not found")

		err := NewPluginError(pluginName, underlyingErr, details)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Code != CodePluginError {
			t.Errorf("Expected CodePluginError, got: %s", err.Code)
		}

		if err.Underlying != underlyingErr {
			t.Errorf("Expected underlying error to be preserved")
		}

		if err.Retryable {
			t.Error("Expected Retryable to be false for non-retryable underlying error")
		}
	})

	t.Run("RetryabilityNotPropagatedFromDownloadError", func(t *testing.T) {
		pluginName := "network-plugin"
		details := "plugin connection failed"
		// Create a DownloadError with Retryable=true
		underlyingErr := NewDownloadError(CodeNetworkError, "network timeout")
		underlyingErr.Retryable = true

		err := NewPluginError(pluginName, underlyingErr, details)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Code != CodePluginError {
			t.Errorf("Expected CodePluginError, got: %s", err.Code)
		}

		// isRetryableError() doesn't check DownloadError.Retryable field,
		// so retryability is NOT propagated from wrapped DownloadErrors
		if err.Retryable {
			t.Error("Expected Retryable to be false (not propagated from DownloadError)")
		}
	})
}

// TestNewConfigError tests the NewConfigError helper function
func TestNewConfigError(t *testing.T) {
	t.Run("WithoutUnderlyingError", func(t *testing.T) {
		message := "invalid configuration value"
		details := "max_retries must be positive"

		err := NewConfigError(message, nil, details)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Code != CodeConfigError {
			t.Errorf("Expected CodeConfigError, got: %s", err.Code)
		}

		if err.Message != message {
			t.Errorf("Expected message: %s, got: %s", message, err.Message)
		}

		if err.Details != details {
			t.Errorf("Expected details: %s, got: %s", details, err.Details)
		}

		if err.Retryable {
			t.Error("Expected Retryable to be false without underlying error")
		}

		if err.Underlying != nil {
			t.Error("Expected no underlying error")
		}
	})

	t.Run("WithNonRetryableUnderlyingError", func(t *testing.T) {
		message := "failed to parse config"
		details := "invalid JSON syntax"
		underlyingErr := errors.New("unexpected token")

		err := NewConfigError(message, underlyingErr, details)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Code != CodeConfigError {
			t.Errorf("Expected CodeConfigError, got: %s", err.Code)
		}

		if err.Underlying != underlyingErr {
			t.Errorf("Expected underlying error to be preserved")
		}

		if err.Retryable {
			t.Error("Expected Retryable to be false for non-retryable underlying error")
		}
	})

	t.Run("RetryabilityNotPropagatedFromDownloadError", func(t *testing.T) {
		message := "failed to load config"
		details := "network error reading config file"
		// Create a DownloadError with Retryable=true
		underlyingErr := NewDownloadError(CodeNetworkError, "connection timeout")
		underlyingErr.Retryable = true

		err := NewConfigError(message, underlyingErr, details)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Code != CodeConfigError {
			t.Errorf("Expected CodeConfigError, got: %s", err.Code)
		}

		// isRetryableError() doesn't check DownloadError.Retryable field,
		// so retryability is NOT propagated from wrapped DownloadErrors
		if err.Retryable {
			t.Error("Expected Retryable to be false (not propagated from DownloadError)")
		}
	})
}

// TestNewValidationError tests the NewValidationError helper function
func TestNewValidationError(t *testing.T) {
	t.Run("SimpleValidation", func(t *testing.T) {
		field := "url"
		reason := "URL must include scheme (http:// or https://)"

		err := NewValidationError(field, reason)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Code != CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", err.Code)
		}

		if !strings.Contains(err.Message, "validation failed") {
			t.Errorf("Expected message to contain 'validation failed', got: %s", err.Message)
		}

		if !strings.Contains(err.Message, field) {
			t.Errorf("Expected message to contain field name, got: %s", err.Message)
		}

		if !strings.Contains(err.Message, reason) {
			t.Errorf("Expected message to contain reason, got: %s", err.Message)
		}

		if !strings.Contains(err.Details, field) {
			t.Errorf("Expected details to contain field, got: %s", err.Details)
		}

		if !strings.Contains(err.Details, reason) {
			t.Errorf("Expected details to contain reason, got: %s", err.Details)
		}

		if err.Retryable {
			t.Error("Expected Retryable to be false for validation errors")
		}

		if err.Underlying != nil {
			t.Error("Expected no underlying error for validation errors")
		}
	})

	t.Run("NumericValidation", func(t *testing.T) {
		field := "timeout"
		reason := "timeout cannot be negative"

		err := NewValidationError(field, reason)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		expectedMsg := "validation failed for timeout: timeout cannot be negative"
		if err.Message != expectedMsg {
			t.Errorf("Expected message: %s, got: %s", expectedMsg, err.Message)
		}

		if !strings.Contains(err.Details, "field=timeout") {
			t.Errorf("Expected details to contain 'field=timeout', got: %s", err.Details)
		}

		if !strings.Contains(err.Details, "reason="+reason) {
			t.Errorf("Expected details to contain reason, got: %s", err.Details)
		}
	})

	t.Run("ComplexValidation", func(t *testing.T) {
		field := "chunk_size"
		reason := "chunk size is too small, minimum is 1KB"

		err := NewValidationError(field, reason)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Code != CodeValidationError {
			t.Errorf("Expected CodeValidationError, got: %s", err.Code)
		}
	})
}

// TestNewStorageError tests the NewStorageError helper function
func TestNewStorageError(t *testing.T) {
	t.Run("WithoutUnderlyingError", func(t *testing.T) {
		operation := "cache write"
		details := "cache directory not writable"

		err := NewStorageError(operation, nil, details)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Code != CodeStorageError {
			t.Errorf("Expected CodeStorageError, got: %s", err.Code)
		}

		if !strings.Contains(err.Message, "storage error") {
			t.Errorf("Expected message to contain 'storage error', got: %s", err.Message)
		}

		if !strings.Contains(err.Message, operation) {
			t.Errorf("Expected message to contain operation, got: %s", err.Message)
		}

		if err.Details != details {
			t.Errorf("Expected details: %s, got: %s", details, err.Details)
		}

		if err.Retryable {
			t.Error("Expected Retryable to be false without underlying error")
		}

		if err.Underlying != nil {
			t.Error("Expected no underlying error")
		}
	})

	t.Run("WithNonRetryableUnderlyingError", func(t *testing.T) {
		operation := "database query"
		details := "failed to execute SELECT statement"
		underlyingErr := errors.New("syntax error in SQL")

		err := NewStorageError(operation, underlyingErr, details)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Code != CodeStorageError {
			t.Errorf("Expected CodeStorageError, got: %s", err.Code)
		}

		if err.Underlying != underlyingErr {
			t.Errorf("Expected underlying error to be preserved")
		}

		if err.Retryable {
			t.Error("Expected Retryable to be false for non-retryable underlying error")
		}
	})

	t.Run("RetryabilityNotPropagatedFromDownloadError", func(t *testing.T) {
		operation := "cache read"
		details := "network timeout reading from distributed cache"
		// Create a DownloadError with Retryable=true
		underlyingErr := NewDownloadError(CodeNetworkError, "connection timeout")
		underlyingErr.Retryable = true

		err := NewStorageError(operation, underlyingErr, details)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		if err.Code != CodeStorageError {
			t.Errorf("Expected CodeStorageError, got: %s", err.Code)
		}

		// isRetryableError() doesn't check DownloadError.Retryable field,
		// so retryability is NOT propagated from wrapped DownloadErrors
		if err.Retryable {
			t.Error("Expected Retryable to be false (not propagated from DownloadError)")
		}
	})

	t.Run("FileOperationError", func(t *testing.T) {
		operation := "file write"
		details := "insufficient disk space"

		err := NewStorageError(operation, nil, details)

		if err == nil {
			t.Fatal("Expected error, got nil")
		}

		expectedMsg := "storage error during file write"
		if err.Message != expectedMsg {
			t.Errorf("Expected message: %s, got: %s", expectedMsg, err.Message)
		}
	})
}

// TestHelpersIntegration tests the helpers working together
func TestHelpersIntegration(t *testing.T) {
	t.Run("ValidationErrorCanBeWrapped", func(t *testing.T) {
		validationErr := NewValidationError("url", "malformed URL")

		// Wrap in a config error
		configErr := NewConfigError("configuration validation failed", validationErr, "url field")

		if configErr.Code != CodeConfigError {
			t.Errorf("Expected CodeConfigError, got: %s", configErr.Code)
		}

		// Check that we can unwrap to the validation error
		var downloadErr *DownloadError
		if !AsDownloadError(configErr.Underlying, &downloadErr) {
			t.Error("Expected to be able to unwrap to DownloadError")
		}

		if downloadErr.Code != CodeValidationError {
			t.Errorf("Expected underlying to be CodeValidationError, got: %s", downloadErr.Code)
		}
	})

	t.Run("StorageErrorWithPluginError", func(t *testing.T) {
		pluginErr := NewPluginError("storage-plugin", errors.New("plugin crashed"), "segfault")
		storageErr := NewStorageError("plugin initialization", pluginErr, "failed to load storage plugin")

		if storageErr.Code != CodeStorageError {
			t.Errorf("Expected CodeStorageError, got: %s", storageErr.Code)
		}

		var downloadErr *DownloadError
		if !AsDownloadError(storageErr.Underlying, &downloadErr) {
			t.Error("Expected to be able to unwrap to DownloadError")
		}

		if downloadErr.Code != CodePluginError {
			t.Errorf("Expected underlying to be CodePluginError, got: %s", downloadErr.Code)
		}
	})
}
