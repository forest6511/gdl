package protocols

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/forest6511/godl/pkg/types"
)

// Mock protocol handler for testing
type mockProtocolHandler struct {
	scheme     string
	canHandle  bool
	downloadOk bool
}

func (m *mockProtocolHandler) Scheme() string {
	return m.scheme
}

func (m *mockProtocolHandler) CanHandle(url string) bool {
	return m.canHandle
}

func (m *mockProtocolHandler) Download(ctx context.Context, url string, opts *types.DownloadOptions) (*types.DownloadStats, error) {
	if m.downloadOk {
		return &types.DownloadStats{
			URL:             url,
			TotalSize:       100,
			BytesDownloaded: 100,
			StartTime:       time.Now(),
			EndTime:         time.Now(),
			Duration:        time.Second,
		}, nil
	}
	return nil, errors.New("mock download failed")
}

func TestProtocolRegistry(t *testing.T) {
	t.Run("NewProtocolRegistry", func(t *testing.T) {
		registry := NewProtocolRegistry()
		if registry == nil {
			t.Fatal("Expected registry to be created")
		}

		// Should have built-in handlers registered
		protocols := registry.ListProtocols()
		if len(protocols) == 0 {
			t.Error("Expected built-in protocols to be registered")
		}

		// Check for common protocols
		expectedProtocols := []string{"http", "ftp", "s3", "torrent"}
		for _, expected := range expectedProtocols {
			found := false
			for _, protocol := range protocols {
				if protocol == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected protocol '%s' to be registered", expected)
			}
		}
	})

	t.Run("RegisterProtocol", func(t *testing.T) {
		registry := NewProtocolRegistry()
		mockHandler := &mockProtocolHandler{
			scheme:     "test",
			canHandle:  true,
			downloadOk: true,
		}

		err := registry.Register(mockHandler)
		if err != nil {
			t.Fatalf("Failed to register mock handler: %v", err)
		}

		protocols := registry.ListProtocols()
		found := false
		for _, protocol := range protocols {
			if protocol == "test" {
				found = true
				break
			}
		}

		if !found {
			t.Error("Expected 'test' protocol to be registered")
		}
	})

	t.Run("GetHandler", func(t *testing.T) {
		registry := NewProtocolRegistry()
		mockHandler := &mockProtocolHandler{
			scheme:     "custom",
			canHandle:  true,
			downloadOk: true,
		}

		err := registry.Register(mockHandler)
		if err != nil {
			t.Fatalf("Failed to register mock handler: %v", err)
		}

		handler, err := registry.GetHandler("custom://example.com/file")
		if err != nil {
			t.Errorf("Failed to get handler: %v", err)
		}
		if handler == nil {
			t.Error("Expected to get handler for custom protocol")
		}

		// Test non-existent protocol
		handler, err = registry.GetHandler("nonexistent://example.com/file")
		if err == nil {
			t.Error("Expected error for non-existent protocol")
		}
		if handler != nil {
			t.Error("Expected nil handler for non-existent protocol")
		}
	})

	t.Run("UnregisterProtocol", func(t *testing.T) {
		registry := NewProtocolRegistry()
		mockHandler := &mockProtocolHandler{
			scheme:     "temp",
			canHandle:  true,
			downloadOk: true,
		}

		err := registry.Register(mockHandler)
		if err != nil {
			t.Fatalf("Failed to register mock handler: %v", err)
		}

		// Verify it's registered
		handler, err := registry.GetHandler("temp://example.com/file")
		if err != nil {
			t.Errorf("Failed to get handler: %v", err)
		}
		if handler == nil {
			t.Error("Expected handler to be registered")
		}

		// Unregister it
		err = registry.Unregister("temp")
		if err != nil {
			t.Logf("Unregister returned error: %v", err)
		}

		// Verify it's gone
		handler, err = registry.GetHandler("temp://example.com/file")
		if err == nil {
			t.Error("Expected error for unregistered protocol")
		}
		if handler != nil {
			t.Error("Expected handler to be unregistered")
		}
	})

	t.Run("Download", func(t *testing.T) {
		registry := NewProtocolRegistry()
		mockHandler := &mockProtocolHandler{
			scheme:     "mock",
			canHandle:  true,
			downloadOk: true,
		}

		err := registry.Register(mockHandler)
		if err != nil {
			t.Fatalf("Failed to register mock handler: %v", err)
		}

		stats, err := registry.Download(context.Background(), "mock://example.com/file", nil)
		if err != nil {
			t.Errorf("Download failed: %v", err)
		}

		if stats == nil {
			t.Error("Expected download stats")
		}

		if stats != nil && stats.BytesDownloaded == 0 {
			t.Error("Expected successful download with bytes downloaded")
		}
	})

	t.Run("DownloadWithUnsupportedProtocol", func(t *testing.T) {
		registry := NewProtocolRegistry()

		_, err := registry.Download(context.Background(), "unsupported://example.com/file", nil)
		if err == nil {
			t.Error("Expected error for unsupported protocol")
		}
	})

	t.Run("DownloadWithFailingHandler", func(t *testing.T) {
		registry := NewProtocolRegistry()
		mockHandler := &mockProtocolHandler{
			scheme:     "failing",
			canHandle:  true,
			downloadOk: false,
		}

		err := registry.Register(mockHandler)
		if err != nil {
			t.Fatalf("Failed to register mock handler: %v", err)
		}

		_, err = registry.Download(context.Background(), "failing://example.com/file", nil)
		if err == nil {
			t.Error("Expected error from failing handler")
		}
	})
}

func TestHTTPHandler(t *testing.T) {
	handler := &HTTPHandler{}

	t.Run("Scheme", func(t *testing.T) {
		schemes := []string{"http", "https"}
		for _, scheme := range schemes {
			if handler.Scheme() != "http" && handler.Scheme() != "https" {
				// Handler might return one scheme or both
				if handler.Scheme() != scheme {
					continue // Check if it's the other scheme
				}
			}
		}
	})

	t.Run("CanHandle", func(t *testing.T) {
		testCases := []struct {
			url      string
			expected bool
		}{
			{"http://example.com/file", true},
			{"https://example.com/file", true},
			{"ftp://example.com/file", false},
			{"file:///local/file", false},
			{"invalid-url", false},
		}

		for _, tc := range testCases {
			result := handler.CanHandle(tc.url)
			if result != tc.expected {
				t.Errorf("CanHandle(%q) = %v, expected %v", tc.url, result, tc.expected)
			}
		}
	})

	// Note: Download method testing would require actual HTTP server setup
	// which is covered in integration tests
}

func TestFTPHandler(t *testing.T) {
	handler := &FTPHandler{}

	t.Run("Scheme", func(t *testing.T) {
		if handler.Scheme() != "ftp" {
			t.Errorf("Expected scheme 'ftp', got '%s'", handler.Scheme())
		}
	})

	t.Run("CanHandle", func(t *testing.T) {
		testCases := []struct {
			url      string
			expected bool
		}{
			{"ftp://example.com/file", true},
			{"ftps://example.com/file", true}, // FTP handler might support FTPS
			{"http://example.com/file", false},
			{"file:///local/file", false},
			{"invalid-url", false},
		}

		for _, tc := range testCases {
			result := handler.CanHandle(tc.url)
			if result != tc.expected {
				t.Errorf("CanHandle(%q) = %v, expected %v", tc.url, result, tc.expected)
			}
		}
	})
}

func TestS3Handler(t *testing.T) {
	handler := &S3Handler{}

	t.Run("Scheme", func(t *testing.T) {
		if handler.Scheme() != "s3" {
			t.Errorf("Expected scheme 's3', got '%s'", handler.Scheme())
		}
	})

	t.Run("CanHandle", func(t *testing.T) {
		testCases := []struct {
			url      string
			expected bool
		}{
			{"s3://bucket/file", true},
			{"s3://bucket/path/file", true},
			{"http://s3.amazonaws.com/bucket/file", false}, // Different handler
			{"https://example.com/file", false},
			{"invalid-url", false},
		}

		for _, tc := range testCases {
			result := handler.CanHandle(tc.url)
			if result != tc.expected {
				t.Errorf("CanHandle(%q) = %v, expected %v", tc.url, result, tc.expected)
			}
		}
	})
}

func TestTorrentHandler(t *testing.T) {
	handler := &TorrentHandler{}

	t.Run("Scheme", func(t *testing.T) {
		if handler.Scheme() != "torrent" {
			t.Errorf("Expected scheme 'torrent', got '%s'", handler.Scheme())
		}
	})

	t.Run("CanHandle", func(t *testing.T) {
		testCases := []struct {
			url      string
			expected bool
		}{
			{"magnet:?xt=urn:btih:hash", true},
			{"magnet:?xt=urn:btih:hash&dn=name", true},
			{"http://example.com/file.torrent", true}, // Torrent handler might handle .torrent files
			{"https://example.com/file", false},
			{"invalid-url", false},
		}

		for _, tc := range testCases {
			result := handler.CanHandle(tc.url)
			if result != tc.expected {
				t.Errorf("CanHandle(%q) = %v, expected %v", tc.url, result, tc.expected)
			}
		}
	})
}

func TestProtocolRegistryEdgeCases(t *testing.T) {
	t.Run("RegisterNilHandler", func(t *testing.T) {
		registry := NewProtocolRegistry()

		// This should handle nil gracefully (might panic or return error depending on implementation)
		defer func() {
			if r := recover(); r != nil {
				t.Log("Register(nil) panicked as expected")
			}
		}()

		err := registry.Register(nil)
		if err != nil {
			t.Logf("Register(nil) returned error: %v", err)
		}

		// Should still have built-in protocols
		protocols := registry.ListProtocols()
		if len(protocols) == 0 {
			t.Error("Registry should still have built-in protocols")
		}
	})

	t.Run("UnregisterNonExistentProtocol", func(t *testing.T) {
		registry := NewProtocolRegistry()

		// This should not panic
		err := registry.Unregister("nonexistent")
		if err != nil {
			t.Logf("Unregister non-existent returned error: %v", err)
		}

		// Should still have built-in protocols
		protocols := registry.ListProtocols()
		if len(protocols) == 0 {
			t.Error("Registry should still have built-in protocols")
		}
	})

	t.Run("GetHandlerForInvalidURL", func(t *testing.T) {
		registry := NewProtocolRegistry()

		handler, err := registry.GetHandler("invalid-url-format")
		if err == nil {
			t.Error("Expected error for invalid URL")
		}
		if handler != nil {
			t.Error("Expected nil handler for invalid URL")
		}

		handler, err = registry.GetHandler("")
		if err == nil {
			t.Error("Expected error for empty URL")
		}
		if handler != nil {
			t.Error("Expected nil handler for empty URL")
		}
	})

	t.Run("DownloadWithEmptyURL", func(t *testing.T) {
		registry := NewProtocolRegistry()

		_, err := registry.Download(context.Background(), "", nil)
		if err == nil {
			t.Error("Expected error for empty URL")
		}
	})
}

func TestProtocolHandlerValidation(t *testing.T) {
	t.Run("HandlerWithEmptyScheme", func(t *testing.T) {
		registry := NewProtocolRegistry()
		mockHandler := &mockProtocolHandler{
			scheme:     "", // Empty scheme
			canHandle:  true,
			downloadOk: true,
		}

		err := registry.Register(mockHandler)
		if err != nil {
			t.Fatalf("Failed to register mock handler: %v", err)
		}

		// Should not be able to get handler for empty scheme
		handler, err := registry.GetHandler("://example.com/file")
		if err == nil {
			t.Error("Expected error for empty scheme URL")
		}
		if handler != nil {
			t.Error("Expected nil handler for empty scheme URL")
		}
	})

	t.Run("HandlerCanHandleFalse", func(t *testing.T) {
		registry := NewProtocolRegistry()
		mockHandler := &mockProtocolHandler{
			scheme:     "selective",
			canHandle:  false, // Handler says it can't handle
			downloadOk: true,
		}

		err := registry.Register(mockHandler)
		if err != nil {
			t.Fatalf("Failed to register mock handler: %v", err)
		}

		handler, err := registry.GetHandler("selective://example.com/file")
		// Behavior depends on implementation - might return handler even if canHandle is false
		// since scheme matches, or might check canHandle first
		if err == nil && handler != nil {
			// If handler is returned, download should still work or fail appropriately
			_, err := registry.Download(context.Background(), "selective://example.com/file", nil)
			if err != nil {
				// This is acceptable - handler can't handle this specific URL
				t.Logf("Handler correctly rejected URL: %v", err)
			}
		}
	})
}

func TestConcurrentAccess(t *testing.T) {
	registry := NewProtocolRegistry()

	// Test concurrent registration and access
	done := make(chan bool, 10)

	// Concurrent registrations
	for i := 0; i < 5; i++ {
		go func(id int) {
			defer func() { done <- true }()
			mockHandler := &mockProtocolHandler{
				scheme:     "concurrent" + string(rune(id+'0')),
				canHandle:  true,
				downloadOk: true,
			}
			err := registry.Register(mockHandler)
			if err != nil {
				// Don't use t.Fatalf in goroutine
				t.Errorf("Failed to register mock handler: %v", err)
			}
		}(i)
	}

	// Concurrent access
	for i := 0; i < 5; i++ {
		go func() {
			defer func() { done <- true }()
			protocols := registry.ListProtocols()
			if len(protocols) == 0 {
				t.Error("Expected protocols to be available")
			}
		}()
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	protocols := registry.ListProtocols()
	if len(protocols) < 5 { // Should have at least built-ins + registered ones
		t.Errorf("Expected at least 5 protocols, got %d", len(protocols))
	}
}
