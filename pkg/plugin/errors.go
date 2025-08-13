package plugin

import (
	"fmt"
	"runtime"
	"time"
)

// PluginErrorCode represents different types of plugin errors
type PluginErrorCode string

const (
	// Registration errors
	ErrPluginAlreadyRegistered PluginErrorCode = "PLUGIN_ALREADY_REGISTERED"
	ErrPluginNotFound          PluginErrorCode = "PLUGIN_NOT_FOUND"
	ErrInvalidPlugin           PluginErrorCode = "INVALID_PLUGIN"

	// Loading errors
	ErrPluginLoadFailed       PluginErrorCode = "PLUGIN_LOAD_FAILED"
	ErrPluginSymbolNotFound   PluginErrorCode = "PLUGIN_SYMBOL_NOT_FOUND"
	ErrPluginInvalidInterface PluginErrorCode = "PLUGIN_INVALID_INTERFACE"

	// Execution errors
	ErrPluginInitFailed      PluginErrorCode = "PLUGIN_INIT_FAILED"
	ErrPluginExecutionFailed PluginErrorCode = "PLUGIN_EXECUTION_FAILED"
	ErrPluginCloseFailed     PluginErrorCode = "PLUGIN_CLOSE_FAILED"

	// Security errors
	ErrSecurityViolation     PluginErrorCode = "SECURITY_VIOLATION"
	ErrResourceLimitExceeded PluginErrorCode = "RESOURCE_LIMIT_EXCEEDED"
	ErrPermissionDenied      PluginErrorCode = "PERMISSION_DENIED"

	// Hook errors
	ErrHookExecutionFailed PluginErrorCode = "HOOK_EXECUTION_FAILED"
	ErrHookNotFound        PluginErrorCode = "HOOK_NOT_FOUND"

	// Configuration errors
	ErrInvalidConfiguration PluginErrorCode = "INVALID_CONFIGURATION"
	ErrConfigurationMissing PluginErrorCode = "CONFIGURATION_MISSING"

	// Dependency errors
	ErrDependencyNotFound  PluginErrorCode = "DEPENDENCY_NOT_FOUND"
	ErrCircularDependency  PluginErrorCode = "CIRCULAR_DEPENDENCY"
	ErrIncompatibleVersion PluginErrorCode = "INCOMPATIBLE_VERSION"
)

// PluginError represents a detailed plugin error
type PluginError struct {
	Code        PluginErrorCode        `json:"code"`
	Message     string                 `json:"message"`
	PluginName  string                 `json:"plugin_name,omitempty"`
	PluginPath  string                 `json:"plugin_path,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Cause       error                  `json:"cause,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	StackTrace  []string               `json:"stack_trace,omitempty"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// Error implements the error interface
func (pe *PluginError) Error() string {
	if pe.PluginName != "" {
		return fmt.Sprintf("[%s] Plugin '%s': %s", pe.Code, pe.PluginName, pe.Message)
	}
	return fmt.Sprintf("[%s] %s", pe.Code, pe.Message)
}

// Unwrap returns the underlying error
func (pe *PluginError) Unwrap() error {
	return pe.Cause
}

// NewPluginError creates a new plugin error
func NewPluginError(code PluginErrorCode, message string) *PluginError {
	return &PluginError{
		Code:      code,
		Message:   message,
		Timestamp: time.Now(),
	}
}

// NewPluginErrorWithCause creates a new plugin error with a cause
func NewPluginErrorWithCause(code PluginErrorCode, message string, cause error) *PluginError {
	return &PluginError{
		Code:      code,
		Message:   message,
		Cause:     cause,
		Timestamp: time.Now(),
	}
}

// WithPlugin adds plugin context to the error
func (pe *PluginError) WithPlugin(name, path string) *PluginError {
	pe.PluginName = name
	pe.PluginPath = path
	return pe
}

// WithDetails adds additional details to the error
func (pe *PluginError) WithDetails(details map[string]interface{}) *PluginError {
	if pe.Details == nil {
		pe.Details = make(map[string]interface{})
	}
	for k, v := range details {
		pe.Details[k] = v
	}
	return pe
}

// WithStackTrace captures the current stack trace
func (pe *PluginError) WithStackTrace() *PluginError {
	const maxDepth = 32
	var pcs [maxDepth]uintptr
	n := runtime.Callers(2, pcs[:])

	pe.StackTrace = make([]string, 0, n)
	frames := runtime.CallersFrames(pcs[:n])

	for {
		frame, more := frames.Next()
		pe.StackTrace = append(pe.StackTrace, fmt.Sprintf("%s:%d %s", frame.File, frame.Line, frame.Function))
		if !more {
			break
		}
	}

	return pe
}

// WithSuggestions adds troubleshooting suggestions
func (pe *PluginError) WithSuggestions(suggestions ...string) *PluginError {
	pe.Suggestions = append(pe.Suggestions, suggestions...)
	return pe
}

// IsRetryable returns true if the error might be resolved by retrying
func (pe *PluginError) IsRetryable() bool {
	switch pe.Code {
	case ErrPluginLoadFailed, ErrPluginExecutionFailed, ErrResourceLimitExceeded:
		return true
	case ErrPluginAlreadyRegistered, ErrPluginNotFound, ErrInvalidPlugin,
		ErrPluginSymbolNotFound, ErrPluginInvalidInterface, ErrSecurityViolation,
		ErrPermissionDenied, ErrInvalidConfiguration, ErrCircularDependency:
		return false
	default:
		return false
	}
}

// GetSeverity returns the error severity level
func (pe *PluginError) GetSeverity() string {
	switch pe.Code {
	case ErrSecurityViolation, ErrPermissionDenied, ErrCircularDependency:
		return "CRITICAL"
	case ErrPluginLoadFailed, ErrPluginInitFailed, ErrDependencyNotFound, ErrIncompatibleVersion:
		return "HIGH"
	case ErrPluginExecutionFailed, ErrHookExecutionFailed, ErrResourceLimitExceeded:
		return "MEDIUM"
	case ErrPluginNotFound, ErrHookNotFound, ErrConfigurationMissing:
		return "LOW"
	default:
		return "MEDIUM"
	}
}

// Common error constructors for frequently used errors

func ErrPluginAlreadyExists(name string) *PluginError {
	return NewPluginError(ErrPluginAlreadyRegistered, "plugin already registered").
		WithPlugin(name, "").
		WithSuggestions(
			"Check if plugin is already loaded",
			"Use a different plugin name",
			"Unregister the existing plugin first",
		)
}

func ErrPluginNotFoundError(name string) *PluginError {
	return NewPluginError(ErrPluginNotFound, "plugin not found").
		WithPlugin(name, "").
		WithSuggestions(
			"Check if plugin name is spelled correctly",
			"Ensure plugin is registered",
			"Verify plugin is installed",
		)
}

func ErrPluginLoadError(path string, cause error) *PluginError {
	return NewPluginErrorWithCause(ErrPluginLoadFailed, "failed to load plugin", cause).
		WithPlugin("", path).
		WithStackTrace().
		WithSuggestions(
			"Check if plugin file exists and is readable",
			"Verify plugin file is a valid Go plugin (.so)",
			"Check plugin dependencies are available",
			"Ensure plugin is compiled for correct architecture",
		)
}

func ErrPluginInitError(name string, cause error) *PluginError {
	return NewPluginErrorWithCause(ErrPluginInitFailed, "plugin initialization failed", cause).
		WithPlugin(name, "").
		WithStackTrace().
		WithSuggestions(
			"Check plugin configuration is valid",
			"Verify all required dependencies are available",
			"Review plugin initialization logs",
		)
}

func ErrSecurityViolationError(operation string, details map[string]interface{}) *PluginError {
	return NewPluginError(ErrSecurityViolation, fmt.Sprintf("security violation: %s", operation)).
		WithDetails(details).
		WithStackTrace().
		WithSuggestions(
			"Review plugin security policy",
			"Check if operation is allowed by current security settings",
			"Consider adjusting security policy if operation is legitimate",
		)
}

func ErrResourceLimitError(resource string, limit interface{}, actual interface{}) *PluginError {
	details := map[string]interface{}{
		"resource": resource,
		"limit":    limit,
		"actual":   actual,
	}

	return NewPluginError(ErrResourceLimitExceeded, fmt.Sprintf("%s limit exceeded", resource)).
		WithDetails(details).
		WithSuggestions(
			"Optimize plugin resource usage",
			"Increase resource limits if appropriate",
			"Check for resource leaks in plugin",
		)
}

// ErrorCollector collects multiple plugin errors
type ErrorCollector struct {
	errors []*PluginError
}

// NewErrorCollector creates a new error collector
func NewErrorCollector() *ErrorCollector {
	return &ErrorCollector{
		errors: make([]*PluginError, 0),
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err *PluginError) {
	if err != nil {
		ec.errors = append(ec.errors, err)
	}
}

// AddError adds a generic error by converting it to PluginError
func (ec *ErrorCollector) AddError(code PluginErrorCode, err error) {
	if err != nil {
		pluginErr := NewPluginErrorWithCause(code, err.Error(), err)
		ec.errors = append(ec.errors, pluginErr)
	}
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	return len(ec.errors) > 0
}

// GetErrors returns all collected errors
func (ec *ErrorCollector) GetErrors() []*PluginError {
	return ec.errors
}

// GetCriticalErrors returns only critical errors
func (ec *ErrorCollector) GetCriticalErrors() []*PluginError {
	var critical []*PluginError
	for _, err := range ec.errors {
		if err.GetSeverity() == "CRITICAL" {
			critical = append(critical, err)
		}
	}
	return critical
}

// Error implements the error interface for ErrorCollector
func (ec *ErrorCollector) Error() string {
	if len(ec.errors) == 0 {
		return "no errors"
	}

	if len(ec.errors) == 1 {
		return ec.errors[0].Error()
	}

	return fmt.Sprintf("multiple plugin errors (%d total): %s", len(ec.errors), ec.errors[0].Error())
}

// Summary returns a summary of all errors
func (ec *ErrorCollector) Summary() map[string]int {
	summary := make(map[string]int)
	for _, err := range ec.errors {
		severity := err.GetSeverity()
		summary[severity]++
	}
	return summary
}
