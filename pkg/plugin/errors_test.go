package plugin

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func TestPluginError_Error(t *testing.T) {
	tests := []struct {
		name       string
		err        *PluginError
		wantPrefix string
	}{
		{
			name: "with plugin name",
			err: &PluginError{
				Code:       ErrPluginNotFound,
				Message:    "plugin not found",
				PluginName: "test-plugin",
			},
			wantPrefix: "[PLUGIN_NOT_FOUND] Plugin 'test-plugin':",
		},
		{
			name: "without plugin name",
			err: &PluginError{
				Code:    ErrPluginLoadFailed,
				Message: "load failed",
			},
			wantPrefix: "[PLUGIN_LOAD_FAILED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			errStr := tt.err.Error()
			if !strings.HasPrefix(errStr, tt.wantPrefix) {
				t.Errorf("Error() = %v, want prefix %v", errStr, tt.wantPrefix)
			}
		})
	}
}

func TestPluginError_Unwrap(t *testing.T) {
	cause := errors.New("underlying error")
	err := &PluginError{
		Code:    ErrPluginInitFailed,
		Message: "init failed",
		Cause:   cause,
	}

	unwrapped := err.Unwrap()
	if unwrapped != cause {
		t.Errorf("Unwrap() = %v, want %v", unwrapped, cause)
	}
}

func TestNewPluginError(t *testing.T) {
	err := NewPluginError(ErrPluginAlreadyRegistered, "test message")

	if err == nil {
		t.Fatal("NewPluginError should not return nil")
	}

	if err.Code != ErrPluginAlreadyRegistered {
		t.Errorf("Code = %v, want %v", err.Code, ErrPluginAlreadyRegistered)
	}

	if err.Message != "test message" {
		t.Errorf("Message = %v, want %v", err.Message, "test message")
	}

	if err.Timestamp.IsZero() {
		t.Error("Timestamp should be set")
	}
}

func TestNewPluginErrorWithCause(t *testing.T) {
	cause := errors.New("root cause")
	err := NewPluginErrorWithCause(ErrPluginLoadFailed, "load error", cause)

	if err == nil {
		t.Fatal("NewPluginErrorWithCause should not return nil")
	}

	if err.Cause != cause {
		t.Errorf("Cause = %v, want %v", err.Cause, cause)
	}
}

func TestPluginError_WithPlugin(t *testing.T) {
	err := NewPluginError(ErrPluginNotFound, "not found")
	_ = err.WithPlugin("test-plugin", "/path/to/plugin")

	if err.PluginName != "test-plugin" {
		t.Errorf("PluginName = %v, want %v", err.PluginName, "test-plugin")
	}

	if err.PluginPath != "/path/to/plugin" {
		t.Errorf("PluginPath = %v, want %v", err.PluginPath, "/path/to/plugin")
	}
}

func TestPluginError_WithDetails(t *testing.T) {
	err := NewPluginError(ErrInvalidConfiguration, "config error")
	details := map[string]interface{}{
		"field":    "timeout",
		"expected": 100,
		"actual":   200,
	}

	_ = err.WithDetails(details)

	if err.Details == nil {
		t.Fatal("Details should not be nil")
	}

	if err.Details["field"] != "timeout" {
		t.Errorf("Details[field] = %v, want %v", err.Details["field"], "timeout")
	}
}

func TestPluginError_WithStackTrace(t *testing.T) {
	err := NewPluginError(ErrPluginExecutionFailed, "execution error")
	_ = err.WithStackTrace()

	if len(err.StackTrace) == 0 {
		t.Error("StackTrace should not be empty")
	}

	// Check that stack trace contains function names
	foundTestFunc := false
	for _, frame := range err.StackTrace {
		if strings.Contains(frame, "TestPluginError_WithStackTrace") {
			foundTestFunc = true
			break
		}
	}

	if !foundTestFunc {
		t.Error("StackTrace should contain test function name")
	}
}

func TestPluginError_WithSuggestions(t *testing.T) {
	err := NewPluginError(ErrDependencyNotFound, "dependency error")
	suggestions := []string{
		"Install missing dependency",
		"Check version compatibility",
	}

	_ = err.WithSuggestions(suggestions...)

	if len(err.Suggestions) != 2 {
		t.Errorf("Suggestions length = %v, want %v", len(err.Suggestions), 2)
	}

	if err.Suggestions[0] != suggestions[0] {
		t.Errorf("Suggestions[0] = %v, want %v", err.Suggestions[0], suggestions[0])
	}
}

func TestPluginError_IsRetryable(t *testing.T) {
	tests := []struct {
		code     PluginErrorCode
		expected bool
	}{
		{ErrPluginLoadFailed, true},
		{ErrPluginExecutionFailed, true},
		{ErrResourceLimitExceeded, true},
		{ErrPluginAlreadyRegistered, false},
		{ErrPluginNotFound, false},
		{ErrSecurityViolation, false},
		{ErrCircularDependency, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := &PluginError{Code: tt.code}
			if got := err.IsRetryable(); got != tt.expected {
				t.Errorf("IsRetryable() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestPluginError_GetSeverity(t *testing.T) {
	tests := []struct {
		code     PluginErrorCode
		expected string
	}{
		{ErrSecurityViolation, "CRITICAL"},
		{ErrPermissionDenied, "CRITICAL"},
		{ErrCircularDependency, "CRITICAL"},
		{ErrPluginLoadFailed, "HIGH"},
		{ErrPluginInitFailed, "HIGH"},
		{ErrDependencyNotFound, "HIGH"},
		{ErrPluginExecutionFailed, "MEDIUM"},
		{ErrResourceLimitExceeded, "MEDIUM"},
		{ErrPluginNotFound, "LOW"},
		{ErrConfigurationMissing, "LOW"},
	}

	for _, tt := range tests {
		t.Run(string(tt.code), func(t *testing.T) {
			err := &PluginError{Code: tt.code}
			if got := err.GetSeverity(); got != tt.expected {
				t.Errorf("GetSeverity() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestErrorConstructors(t *testing.T) {
	t.Run("ErrPluginAlreadyExists", func(t *testing.T) {
		err := ErrPluginAlreadyExists("test-plugin")
		if err.Code != ErrPluginAlreadyRegistered {
			t.Errorf("Code = %v, want %v", err.Code, ErrPluginAlreadyRegistered)
		}
		if len(err.Suggestions) == 0 {
			t.Error("Should have suggestions")
		}
	})

	t.Run("ErrPluginNotFoundError", func(t *testing.T) {
		err := ErrPluginNotFoundError("test-plugin")
		if err.Code != ErrPluginNotFound {
			t.Errorf("Code = %v, want %v", err.Code, ErrPluginNotFound)
		}
		if err.PluginName != "test-plugin" {
			t.Errorf("PluginName = %v, want %v", err.PluginName, "test-plugin")
		}
	})

	t.Run("ErrPluginLoadError", func(t *testing.T) {
		cause := errors.New("file not found")
		err := ErrPluginLoadError("/path/to/plugin.so", cause)
		if err.Code != ErrPluginLoadFailed {
			t.Errorf("Code = %v, want %v", err.Code, ErrPluginLoadFailed)
		}
		if err.Cause != cause {
			t.Errorf("Cause = %v, want %v", err.Cause, cause)
		}
		if len(err.StackTrace) == 0 {
			t.Error("Should have stack trace")
		}
	})

	t.Run("ErrSecurityViolationError", func(t *testing.T) {
		details := map[string]interface{}{"path": "/etc/passwd"}
		err := ErrSecurityViolationError("file access", details)
		if err.Code != ErrSecurityViolation {
			t.Errorf("Code = %v, want %v", err.Code, ErrSecurityViolation)
		}
		if err.Details["path"] != "/etc/passwd" {
			t.Error("Details not set correctly")
		}
	})

	t.Run("ErrResourceLimitError", func(t *testing.T) {
		err := ErrResourceLimitError("memory", 100, 200)
		if err.Code != ErrResourceLimitExceeded {
			t.Errorf("Code = %v, want %v", err.Code, ErrResourceLimitExceeded)
		}
		if err.Details["limit"] != 100 {
			t.Error("Limit not set in details")
		}
		if err.Details["actual"] != 200 {
			t.Error("Actual not set in details")
		}
	})
}

func TestErrorCollector(t *testing.T) {
	collector := NewErrorCollector()

	if collector == nil {
		t.Fatal("NewErrorCollector should not return nil")
	}

	// Initially should have no errors
	if collector.HasErrors() {
		t.Error("New collector should not have errors")
	}

	// Add an error
	err1 := NewPluginError(ErrPluginNotFound, "not found")
	collector.Add(err1)

	if !collector.HasErrors() {
		t.Error("Collector should have errors after adding")
	}

	if len(collector.GetErrors()) != 1 {
		t.Errorf("GetErrors() length = %v, want %v", len(collector.GetErrors()), 1)
	}

	// Add nil error (should be ignored)
	collector.Add(nil)
	if len(collector.GetErrors()) != 1 {
		t.Error("Nil errors should not be added")
	}

	// Add generic error with AddError
	genericErr := errors.New("generic error")
	collector.AddError(ErrPluginLoadFailed, genericErr)

	if len(collector.GetErrors()) != 2 {
		t.Errorf("GetErrors() length = %v, want %v", len(collector.GetErrors()), 2)
	}
}

func TestErrorCollector_GetCriticalErrors(t *testing.T) {
	collector := NewErrorCollector()

	// Add various severity errors
	collector.Add(NewPluginError(ErrSecurityViolation, "security"))  // CRITICAL
	collector.Add(NewPluginError(ErrPluginLoadFailed, "load"))       // HIGH
	collector.Add(NewPluginError(ErrPluginExecutionFailed, "exec"))  // MEDIUM
	collector.Add(NewPluginError(ErrCircularDependency, "circular")) // CRITICAL

	critical := collector.GetCriticalErrors()
	if len(critical) != 2 {
		t.Errorf("GetCriticalErrors() length = %v, want %v", len(critical), 2)
	}
}

func TestErrorCollector_Error(t *testing.T) {
	collector := NewErrorCollector()

	// Empty collector
	if collector.Error() != "no errors" {
		t.Errorf("Empty collector Error() = %v, want %v", collector.Error(), "no errors")
	}

	// Single error
	collector.Add(NewPluginError(ErrPluginNotFound, "test error"))
	errStr := collector.Error()
	if !strings.Contains(errStr, "test error") {
		t.Error("Single error message should contain error text")
	}

	// Multiple errors
	collector.Add(NewPluginError(ErrPluginLoadFailed, "second error"))
	errStr = collector.Error()
	if !strings.Contains(errStr, "multiple plugin errors") {
		t.Error("Multiple errors message should indicate multiple errors")
	}
}

func TestErrorCollector_Summary(t *testing.T) {
	collector := NewErrorCollector()

	// Add errors of different severities
	collector.Add(NewPluginError(ErrSecurityViolation, "critical1"))   // CRITICAL
	collector.Add(NewPluginError(ErrPluginLoadFailed, "high1"))        // HIGH
	collector.Add(NewPluginError(ErrPluginInitFailed, "high2"))        // HIGH
	collector.Add(NewPluginError(ErrPluginExecutionFailed, "medium1")) // MEDIUM

	summary := collector.Summary()

	if summary["CRITICAL"] != 1 {
		t.Errorf("Summary[CRITICAL] = %v, want %v", summary["CRITICAL"], 1)
	}

	if summary["HIGH"] != 2 {
		t.Errorf("Summary[HIGH] = %v, want %v", summary["HIGH"], 2)
	}

	if summary["MEDIUM"] != 1 {
		t.Errorf("Summary[MEDIUM] = %v, want %v", summary["MEDIUM"], 1)
	}
}

func TestPluginError_TimestampSet(t *testing.T) {
	before := time.Now()
	err := NewPluginError(ErrPluginNotFound, "test")
	after := time.Now()

	if err.Timestamp.Before(before) || err.Timestamp.After(after) {
		t.Error("Timestamp should be set to current time")
	}
}
