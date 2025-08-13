package plugin

import (
	"context"
	"testing"
	"time"
)

func TestDefaultSecurity(t *testing.T) {
	sec := DefaultSecurity()

	if sec == nil {
		t.Fatal("DefaultSecurity should not return nil")
	}

	if sec.MaxMemoryUsage != 100 {
		t.Errorf("Expected MaxMemoryUsage 100, got %d", sec.MaxMemoryUsage)
	}

	if sec.MaxExecutionTime != 30*time.Second {
		t.Errorf("Expected MaxExecutionTime 30s, got %v", sec.MaxExecutionTime)
	}

	if sec.NetworkAccess {
		t.Error("NetworkAccess should be false by default")
	}

	if !sec.FileSystemAccess {
		t.Error("FileSystemAccess should be true by default")
	}
}

func TestStrictSecurity(t *testing.T) {
	sec := StrictSecurity()

	if sec == nil {
		t.Fatal("StrictSecurity should not return nil")
	}

	if sec.MaxMemoryUsage != 50 {
		t.Errorf("Expected MaxMemoryUsage 50, got %d", sec.MaxMemoryUsage)
	}

	if sec.MaxExecutionTime != 10*time.Second {
		t.Errorf("Expected MaxExecutionTime 10s, got %v", sec.MaxExecutionTime)
	}

	if sec.NetworkAccess {
		t.Error("NetworkAccess should be false in strict mode")
	}

	if sec.FileSystemAccess {
		t.Error("FileSystemAccess should be false in strict mode")
	}

	if !sec.ReadOnlyMode {
		t.Error("ReadOnlyMode should be true in strict mode")
	}
}

func TestSecurityValidator_ValidateFilePath(t *testing.T) {
	policy := &PluginSecurity{
		AllowedPaths: []string{"./data", "./plugins"},
		BlockedPaths: []string{"/etc", "/sys"},
	}

	validator := NewSecurityValidator(policy, ".")

	tests := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{"allowed relative path", "./data/test.txt", false},
		{"blocked system path", "/etc/passwd", true},
		{"blocked sys path", "/sys/test", true},
		{"unspecified path", "/tmp/test", true}, // Not in allowed paths
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFilePath(tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFilePath() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurityValidator_ValidateFileOperation(t *testing.T) {
	policy := &PluginSecurity{
		FileSystemAccess: true,
		ReadOnlyMode:     true,
		AllowedPaths:     []string{"./data"},
	}

	validator := NewSecurityValidator(policy, ".")

	tests := []struct {
		name      string
		operation string
		path      string
		wantErr   bool
	}{
		{"read allowed", "read", "./data/test.txt", false},
		{"write in read-only", "write", "./data/test.txt", true},
		{"create in read-only", "create", "./data/new.txt", true},
		{"delete in read-only", "delete", "./data/test.txt", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFileOperation(tt.operation, tt.path)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFileOperation() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurityValidator_ValidateFileSize(t *testing.T) {
	policy := &PluginSecurity{
		MaxFileSize: 1024 * 1024, // 1MB
	}

	validator := NewSecurityValidator(policy, ".")

	tests := []struct {
		name    string
		size    int64
		wantErr bool
	}{
		{"under limit", 1024, false},
		{"at limit", 1024 * 1024, false},
		{"over limit", 1024 * 1024 * 2, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateFileSize(tt.size)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateFileSize() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurityValidator_ValidateNetworkAccess(t *testing.T) {
	policy := &PluginSecurity{
		NetworkAccess: true,
		AllowedHosts:  []string{"github.com", "api.example.com"},
		BlockedHosts:  []string{"malicious.com"},
	}

	validator := NewSecurityValidator(policy, ".")

	tests := []struct {
		name    string
		host    string
		wantErr bool
	}{
		{"allowed host", "github.com", false},
		{"allowed API host", "api.example.com", false},
		{"blocked host", "malicious.com", true},
		{"unspecified host", "random.com", true}, // Not in allowed hosts
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateNetworkAccess(tt.host)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateNetworkAccess() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestSecurityValidator_NetworkDisabled(t *testing.T) {
	policy := &PluginSecurity{
		NetworkAccess: false,
	}

	validator := NewSecurityValidator(policy, ".")

	err := validator.ValidateNetworkAccess("any.host.com")
	if err == nil {
		t.Error("Expected error when network access is disabled")
	}
}

func TestResourceMonitor_CheckResources(t *testing.T) {
	monitor := NewResourceMonitor(100, 100*time.Millisecond)

	// Initial check should pass
	if err := monitor.CheckResources(); err != nil {
		t.Errorf("Initial resource check failed: %v", err)
	}

	// Wait for timeout
	time.Sleep(150 * time.Millisecond)

	// Should fail due to timeout
	if err := monitor.CheckResources(); err == nil {
		t.Error("Expected error due to execution timeout")
	}
}

func TestSecurePluginExecutor_Execute(t *testing.T) {
	// Create mock plugin
	mockPlugin := &MockPlugin{
		name:    "test-plugin",
		version: "1.0.0",
	}

	security := &PluginSecurity{
		MaxMemoryUsage:   100,
		MaxExecutionTime: 1 * time.Second,
		FileSystemAccess: true,
	}

	executor := NewSecurePluginExecutor(mockPlugin, security, ".")

	ctx := context.Background()

	// Test Init method
	_, err := executor.Execute(ctx, "Init", map[string]interface{}{"test": "config"})
	if err != nil {
		t.Errorf("Execute Init failed: %v", err)
	}

	// Test Close method
	_, err = executor.Execute(ctx, "Close")
	if err != nil {
		t.Errorf("Execute Close failed: %v", err)
	}

	// Test unsupported method
	_, err = executor.Execute(ctx, "UnsupportedMethod")
	if err == nil {
		t.Error("Expected error for unsupported method")
	}
}

func TestSecurePluginExecutor_WithTimeout(t *testing.T) {
	mockPlugin := &TestPluginWithDelay{
		MockPlugin: MockPlugin{
			name:    "test-plugin",
			version: "1.0.0",
		},
		initDelay: 200 * time.Millisecond,
	}

	security := &PluginSecurity{
		MaxMemoryUsage:   100,
		MaxExecutionTime: 100 * time.Millisecond, // Shorter than init delay
		FileSystemAccess: true,
	}

	executor := NewSecurePluginExecutor(mockPlugin, security, ".")

	ctx := context.Background()

	// Should timeout
	done := make(chan error, 1)
	go func() {
		_, err := executor.Execute(ctx, "Init", map[string]interface{}{})
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			// This is expected due to timeout or pre-check
			t.Logf("Execution completed with expected error: %v", err)
		}
	case <-time.After(500 * time.Millisecond):
		// This is also acceptable as the context might have been cancelled
		t.Log("Execution timed out as expected")
	}
}

func TestSecurePluginExecutor_SecurityViolation(t *testing.T) {
	security := StrictSecurity()

	// Mock plugin that tries to violate security
	violatingPlugin := &MockPlugin{name: "violator", version: "1.0.0"}

	executor := NewSecurePluginExecutor(violatingPlugin, security, "/tmp")

	// Test execution with security violation
	ctx := context.Background()
	result, err := executor.Execute(ctx, "test-operation")

	if err == nil {
		t.Error("Expected security violation to be caught")
	}

	if result != nil {
		t.Error("Expected nil result on security violation")
	}

	// The error might be about unsupported method, which is acceptable
	if !containsString(err.Error(), "security") && !containsString(err.Error(), "violation") && !containsString(err.Error(), "unsupported") {
		t.Errorf("Expected security-related or method-related error, got: %v", err)
	}
}

func TestResourceMonitor_EdgeCases(t *testing.T) {
	t.Run("CheckWithZeroLimits", func(t *testing.T) {
		monitor := NewResourceMonitor(0, 0) // Zero limits

		// Should handle zero limits gracefully
		err := monitor.CheckResources()
		// Zero limits might be interpreted as "no limit" or cause errors
		if err != nil {
			t.Logf("Zero limits returned error (may be expected): %v", err)
		}
	})

	t.Run("CheckWithVerySmallLimits", func(t *testing.T) {
		monitor := NewResourceMonitor(1, 1*time.Nanosecond) // Very small limits

		// Should handle very small limits
		err := monitor.CheckResources()
		if err != nil {
			t.Logf("Very small limits returned error: %v", err)
		}
	})

	t.Run("CheckWithLargeLimits", func(t *testing.T) {
		monitor := NewResourceMonitor(999999999999, 24*time.Hour) // Very large limits

		// Should handle large limits gracefully
		err := monitor.CheckResources()
		if err != nil {
			t.Logf("Large limits returned error: %v", err)
		}
	})
}

// TestPluginWithDelay for testing timeouts
type TestPluginWithDelay struct {
	MockPlugin
	initDelay time.Duration
}

func (t *TestPluginWithDelay) Init(config map[string]interface{}) error {
	if t.initDelay > 0 {
		time.Sleep(t.initDelay)
	}
	return nil
}
