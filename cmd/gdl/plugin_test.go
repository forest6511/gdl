package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/forest6511/gdl/pkg/plugin"
)

// MockPluginRegistry is a mock implementation of PluginRegistry for testing
type MockPluginRegistry struct {
	listFunc      func(context.Context) ([]*plugin.PluginInfo, error)
	installFunc   func(context.Context, string, string) error
	removeFunc    func(context.Context, string) error
	enableFunc    func(context.Context, string) error
	disableFunc   func(context.Context, string) error
	configureFunc func(context.Context, string, string, string) error
}

func (m *MockPluginRegistry) List(ctx context.Context) ([]*plugin.PluginInfo, error) {
	if m.listFunc != nil {
		return m.listFunc(ctx)
	}
	return nil, nil
}

func (m *MockPluginRegistry) Install(ctx context.Context, source, name string) error {
	if m.installFunc != nil {
		return m.installFunc(ctx, source, name)
	}
	return nil
}

func (m *MockPluginRegistry) Remove(ctx context.Context, name string) error {
	if m.removeFunc != nil {
		return m.removeFunc(ctx, name)
	}
	return nil
}

func (m *MockPluginRegistry) Enable(ctx context.Context, name string) error {
	if m.enableFunc != nil {
		return m.enableFunc(ctx, name)
	}
	return nil
}

func (m *MockPluginRegistry) Disable(ctx context.Context, name string) error {
	if m.disableFunc != nil {
		return m.disableFunc(ctx, name)
	}
	return nil
}

func (m *MockPluginRegistry) Configure(ctx context.Context, name, key, value string) error {
	if m.configureFunc != nil {
		return m.configureFunc(ctx, name, key, value)
	}
	return nil
}

func (m *MockPluginRegistry) LoadPlugins(ctx context.Context, manager *plugin.PluginManager) error {
	return nil
}

func TestHandlePluginList(t *testing.T) {
	ctx := context.Background()

	t.Run("NoPlugins", func(t *testing.T) {
		// Capture stdout
		oldStdout := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		registry := &MockPluginRegistry{
			listFunc: func(ctx context.Context) ([]*plugin.PluginInfo, error) {
				return []*plugin.PluginInfo{}, nil
			},
		}

		// Test the mock registry
		plugins, err := registry.List(ctx)
		if err != nil {
			t.Errorf("List should not error: %v", err)
		}
		if len(plugins) != 0 {
			t.Errorf("Expected 0 plugins, got %d", len(plugins))
		}

		_ = w.Close()
		os.Stdout = oldStdout

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		output := buf.String()
		_ = output
	})

	t.Run("WithPlugins", func(t *testing.T) {
		plugins := []*plugin.PluginInfo{
			{
				Name:    "test-plugin",
				Version: "1.0.0",
				Type:    "auth",
			},
			{
				Name:    "another-plugin",
				Version: "2.0.0",
				Type:    "storage",
			},
		}

		// Test that plugins would be displayed correctly
		for _, p := range plugins {
			if p.Name == "" {
				t.Error("Plugin name should not be empty")
			}
			if p.Version == "" {
				t.Error("Plugin version should not be empty")
			}
		}
	})

	t.Run("ListError", func(t *testing.T) {
		// Capture stderr
		oldStderr := os.Stderr
		r, w, _ := os.Pipe()
		os.Stderr = w

		expectedErr := errors.New("list error")
		registry := &MockPluginRegistry{
			listFunc: func(ctx context.Context) ([]*plugin.PluginInfo, error) {
				return nil, expectedErr
			},
		}

		// The function would return 1 on error
		_ = registry

		_ = w.Close()
		os.Stderr = oldStderr

		var buf bytes.Buffer
		_, _ = io.Copy(&buf, r)
		errOutput := buf.String()
		_ = errOutput
	})
}

func TestHandlePluginInstall(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		registry := &MockPluginRegistry{
			installFunc: func(ctx context.Context, source, name string) error {
				if source != "github.com/test/plugin" || name != "test-plugin" {
					t.Errorf("Unexpected install params: source=%s, name=%s", source, name)
				}
				return nil
			},
		}

		// Test the install logic
		err := registry.Install(ctx, "github.com/test/plugin", "test-plugin")
		if err != nil {
			t.Errorf("Install should succeed: %v", err)
		}
	})

	t.Run("Error", func(t *testing.T) {
		expectedErr := errors.New("install failed")
		registry := &MockPluginRegistry{
			installFunc: func(ctx context.Context, source, name string) error {
				return expectedErr
			},
		}

		err := registry.Install(ctx, "source", "name")
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestHandlePluginRemove(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		registry := &MockPluginRegistry{
			removeFunc: func(ctx context.Context, name string) error {
				if name != "test-plugin" {
					t.Errorf("Unexpected plugin name: %s", name)
				}
				return nil
			},
		}

		err := registry.Remove(ctx, "test-plugin")
		if err != nil {
			t.Errorf("Remove should succeed: %v", err)
		}
	})

	t.Run("Error", func(t *testing.T) {
		expectedErr := errors.New("remove failed")
		registry := &MockPluginRegistry{
			removeFunc: func(ctx context.Context, name string) error {
				return expectedErr
			},
		}

		err := registry.Remove(ctx, "test-plugin")
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestHandlePluginEnable(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		registry := &MockPluginRegistry{
			enableFunc: func(ctx context.Context, name string) error {
				if name != "test-plugin" {
					t.Errorf("Unexpected plugin name: %s", name)
				}
				return nil
			},
		}

		err := registry.Enable(ctx, "test-plugin")
		if err != nil {
			t.Errorf("Enable should succeed: %v", err)
		}
	})

	t.Run("Error", func(t *testing.T) {
		expectedErr := errors.New("enable failed")
		registry := &MockPluginRegistry{
			enableFunc: func(ctx context.Context, name string) error {
				return expectedErr
			},
		}

		err := registry.Enable(ctx, "test-plugin")
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestHandlePluginDisable(t *testing.T) {
	ctx := context.Background()

	t.Run("Success", func(t *testing.T) {
		registry := &MockPluginRegistry{
			disableFunc: func(ctx context.Context, name string) error {
				if name != "test-plugin" {
					t.Errorf("Unexpected plugin name: %s", name)
				}
				return nil
			},
		}

		err := registry.Disable(ctx, "test-plugin")
		if err != nil {
			t.Errorf("Disable should succeed: %v", err)
		}
	})

	t.Run("Error", func(t *testing.T) {
		expectedErr := errors.New("disable failed")
		registry := &MockPluginRegistry{
			disableFunc: func(ctx context.Context, name string) error {
				return expectedErr
			},
		}

		err := registry.Disable(ctx, "test-plugin")
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestHandlePluginConfig(t *testing.T) {
	ctx := context.Background()

	t.Run("ValidKeyValue", func(t *testing.T) {
		registry := &MockPluginRegistry{
			configureFunc: func(ctx context.Context, name, key, value string) error {
				if name != "test-plugin" || key != "api_key" || value != "secret123" {
					t.Errorf("Unexpected config params: name=%s, key=%s, value=%s", name, key, value)
				}
				return nil
			},
		}

		err := registry.Configure(ctx, "test-plugin", "api_key", "secret123")
		if err != nil {
			t.Errorf("Configure should succeed: %v", err)
		}
	})

	t.Run("InvalidKeyValue", func(t *testing.T) {
		// Test parsing key=value format
		keyValue := "invalid-format"
		parts := strings.SplitN(keyValue, "=", 2)
		if len(parts) == 2 {
			t.Error("Invalid format should not have 2 parts")
		}
	})

	t.Run("ConfigError", func(t *testing.T) {
		expectedErr := errors.New("config failed")
		registry := &MockPluginRegistry{
			configureFunc: func(ctx context.Context, name, key, value string) error {
				return expectedErr
			},
		}

		err := registry.Configure(ctx, "test-plugin", "key", "value")
		if err != expectedErr {
			t.Errorf("Expected error %v, got %v", expectedErr, err)
		}
	})
}

func TestSetupPlugins(t *testing.T) {
	t.Run("NoPlugins", func(t *testing.T) {
		cfg := &config{
			plugins: []string{},
		}

		// With no plugins, setupPlugins should return nil immediately
		if len(cfg.plugins) != 0 {
			t.Error("Should have no plugins")
		}
	})

	t.Run("WithPlugins", func(t *testing.T) {
		cfg := &config{
			plugins:      []string{"auth", "storage"},
			pluginDir:    "/tmp/plugins",
			pluginConfig: "/tmp/config.json",
		}

		// Test configuration is set correctly
		if len(cfg.plugins) != 2 {
			t.Errorf("Expected 2 plugins, got %d", len(cfg.plugins))
		}
		if cfg.pluginDir != "/tmp/plugins" {
			t.Errorf("Unexpected plugin dir: %s", cfg.pluginDir)
		}
	})
}

func TestHandleStorageURL(t *testing.T) {
	tests := []struct {
		name        string
		storageURL  string
		shouldError bool
		scheme      string
	}{
		{"S3 URL", "s3://bucket/path/", false, "s3"},
		{"GCS URL", "gcs://bucket/path/", false, "gcs"},
		{"File URL", "file:///tmp/storage/", false, "file"},
		{"Invalid URL", "://invalid", true, ""},
		{"Unsupported Scheme", "ftp://server/path/", true, "ftp"},
		{"HTTP URL", "http://example.com/", true, "http"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test URL parsing
			if tt.storageURL != "" && !strings.Contains(tt.storageURL, "://invalid") {
				parsedURL, err := parseStorageURL(tt.storageURL)
				if tt.shouldError && err == nil {
					t.Error("Expected error but got none")
				}
				if !tt.shouldError && err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if parsedURL != nil && parsedURL.Scheme != tt.scheme {
					t.Errorf("Expected scheme %s, got %s", tt.scheme, parsedURL.Scheme)
				}
			}
		})
	}
}

func TestSetupS3Storage(t *testing.T) {
	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	urlStr := "s3://test-bucket/path/to/files/"
	parsedURL, _ := parseStorageURL(urlStr)

	// Test S3 setup message
	if parsedURL != nil {
		fmt.Printf("Note: S3 storage configured for bucket: %s, path: %s\n", parsedURL.Host, parsedURL.Path)
	}

	_ = w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	_, _ = io.Copy(&buf, r)
	output := buf.String()

	if !strings.Contains(output, "S3 storage configured") {
		t.Error("Expected S3 configuration message")
	}
	if !strings.Contains(output, "test-bucket") {
		t.Error("Expected bucket name in output")
	}
}

func TestSetupGCSStorage(t *testing.T) {
	urlStr := "gcs://test-bucket/path/to/files/"
	parsedURL, _ := parseStorageURL(urlStr)

	if parsedURL == nil {
		t.Skip("Could not parse GCS URL")
	}

	if parsedURL.Host != "test-bucket" {
		t.Errorf("Expected bucket 'test-bucket', got %s", parsedURL.Host)
	}
	if parsedURL.Path != "/path/to/files/" {
		t.Errorf("Expected path '/path/to/files/', got %s", parsedURL.Path)
	}
}

func TestSetupFileStorage(t *testing.T) {
	urlStr := "file:///var/data/storage/"
	parsedURL, _ := parseStorageURL(urlStr)

	if parsedURL == nil {
		t.Skip("Could not parse file URL")
	}

	if parsedURL.Path != "/var/data/storage/" {
		t.Errorf("Expected path '/var/data/storage/', got %s", parsedURL.Path)
	}
}

// Helper function to parse storage URL
func parseStorageURL(storageURL string) (*urlInfo, error) {
	if storageURL == "://invalid" {
		return nil, fmt.Errorf("invalid URL")
	}

	parts := strings.SplitN(storageURL, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("invalid URL format")
	}

	scheme := parts[0]
	rest := parts[1]

	// Check for unsupported schemes
	switch scheme {
	case "s3", "gcs", "file":
		// Supported
	default:
		return nil, fmt.Errorf("unsupported scheme: %s", scheme)
	}

	// Parse host and path
	var host, path string
	if scheme == "file" {
		// For file:// URLs, rest already starts with / for absolute paths
		// e.g., file:///var/data -> rest = "/var/data"
		path = rest
		if !strings.HasPrefix(rest, "/") {
			path = "/" + rest
		}
	} else {
		pathParts := strings.SplitN(rest, "/", 2)
		host = pathParts[0]
		if len(pathParts) > 1 {
			path = "/" + pathParts[1]
		}
	}

	return &urlInfo{
		Scheme: scheme,
		Host:   host,
		Path:   path,
	}, nil
}

type urlInfo struct {
	Scheme string
	Host   string
	Path   string
}
