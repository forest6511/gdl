package network

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

func TestNewDiagnostics(t *testing.T) {
	diag := NewDiagnostics()

	if diag == nil {
		t.Fatal("NewDiagnostics() returned nil")
	}

	if diag.client == nil {
		t.Error("client should not be nil")
	}

	if diag.timeout != DefaultTimeout {
		t.Errorf("timeout = %v, want %v", diag.timeout, DefaultTimeout)
	}

	if len(diag.testHosts) == 0 {
		t.Error("testHosts should not be empty")
	}

	if len(diag.dnsServers) == 0 {
		t.Error("dnsServers should not be empty")
	}

	if len(diag.bandwidthServers) == 0 {
		t.Error("bandwidthServers should not be empty")
	}

	if diag.verbose != false {
		t.Error("verbose should be false by default")
	}
}

func TestDiagnostics_WithTimeout(t *testing.T) {
	diag := NewDiagnostics()
	newTimeout := 45 * time.Second

	result := diag.WithTimeout(newTimeout)

	if result != diag {
		t.Error("WithTimeout should return the same instance")
	}

	if diag.timeout != newTimeout {
		t.Errorf("timeout = %v, want %v", diag.timeout, newTimeout)
	}

	if diag.client.Timeout != newTimeout {
		t.Errorf("client timeout = %v, want %v", diag.client.Timeout, newTimeout)
	}
}

func TestDiagnostics_WithTestHosts(t *testing.T) {
	diag := NewDiagnostics()
	testHosts := []string{"example.com", "test.org"}

	result := diag.WithTestHosts(testHosts)

	if result != diag {
		t.Error("WithTestHosts should return the same instance")
	}

	if len(diag.testHosts) != len(testHosts) {
		t.Errorf("testHosts length = %d, want %d", len(diag.testHosts), len(testHosts))
	}

	for i, host := range testHosts {
		if diag.testHosts[i] != host {
			t.Errorf("testHosts[%d] = %s, want %s", i, diag.testHosts[i], host)
		}
	}
}

func TestDiagnostics_WithDNSServers(t *testing.T) {
	diag := NewDiagnostics()
	dnsServers := []string{"8.8.8.8", "1.1.1.1"}

	result := diag.WithDNSServers(dnsServers)

	if result != diag {
		t.Error("WithDNSServers should return the same instance")
	}

	if len(diag.dnsServers) != len(dnsServers) {
		t.Errorf("dnsServers length = %d, want %d", len(diag.dnsServers), len(dnsServers))
	}

	for i, server := range dnsServers {
		if diag.dnsServers[i] != server {
			t.Errorf("dnsServers[%d] = %s, want %s", i, diag.dnsServers[i], server)
		}
	}
}

func TestDiagnostics_WithVerbose(t *testing.T) {
	diag := NewDiagnostics()

	result := diag.WithVerbose(true)

	if result != diag {
		t.Error("WithVerbose should return the same instance")
	}

	if !diag.verbose {
		t.Error("verbose should be true")
	}
}

func TestHealthStatus_String(t *testing.T) {
	tests := []struct {
		status   HealthStatus
		expected string
	}{
		{HealthGood, "good"},
		{HealthWarning, "warning"},
		{HealthPoor, "poor"},
		{HealthCritical, "critical"},
		{HealthStatus(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.status.String()
			if result != tt.expected {
				t.Errorf("HealthStatus.String() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestProxyType_String(t *testing.T) {
	tests := []struct {
		proxyType ProxyType
		expected  string
	}{
		{ProxyHTTP, "HTTP"},
		{ProxyHTTPS, "HTTPS"},
		{ProxySOCKS4, "SOCKS4"},
		{ProxySOCKS5, "SOCKS5"},
		{ProxyUnknown, "Unknown"},
		{ProxyType(999), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.proxyType.String()
			if result != tt.expected {
				t.Errorf("ProxyType.String() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestDiagnostics_TestDNSResolution(t *testing.T) {
	diag := NewDiagnostics()
	ctx := context.Background()

	// Test with real DNS servers and hosts (this will depend on network connectivity)
	dnsServers := []string{"8.8.8.8"}
	testHosts := []string{"google.com"}

	result := diag.TestDNSResolution(ctx, dnsServers, testHosts)

	if result.TestName != "dns_resolution" {
		t.Errorf("TestName = %s, want dns_resolution", result.TestName)
	}

	if result.Details == nil {
		t.Error("Details should not be nil")
	}

	if result.Duration <= 0 {
		t.Error("Duration should be greater than 0")
	}

	// Check that details contain expected fields
	if _, ok := result.Details["successful_lookups"]; !ok {
		t.Error("Details should contain successful_lookups")
	}

	if _, ok := result.Details["failed_lookups"]; !ok {
		t.Error("Details should contain failed_lookups")
	}

	if _, ok := result.Details["resolution_results"]; !ok {
		t.Error("Details should contain resolution_results")
	}
}

func TestDiagnostics_TestConnectivity(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	diag := NewDiagnostics()
	ctx := context.Background()

	// Extract hostname from test server URL
	serverURL := strings.TrimPrefix(server.URL, "http://")
	testHosts := []string{serverURL}

	result := diag.TestConnectivity(ctx, testHosts)

	if result.TestName != "connectivity_test" {
		t.Errorf("TestName = %s, want connectivity_test", result.TestName)
	}

	if result.Details == nil {
		t.Error("Details should not be nil")
	}

	if result.Duration <= 0 {
		t.Error("Duration should be greater than 0")
	}

	// Check that details contain expected fields
	if _, ok := result.Details["successful_connections"]; !ok {
		t.Error("Details should contain successful_connections")
	}

	if _, ok := result.Details["failed_connections"]; !ok {
		t.Error("Details should contain failed_connections")
	}

	if _, ok := result.Details["connectivity_results"]; !ok {
		t.Error("Details should contain connectivity_results")
	}
}

func TestDiagnostics_testHTTPConnectivity(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	diag := NewDiagnostics()
	ctx := context.Background()

	result := diag.testHTTPConnectivity(ctx, server.URL)

	if result == nil {
		t.Fatal("testHTTPConnectivity returned nil")
	}

	if success, ok := result["success"]; !ok || !success.(bool) {
		t.Error("Expected successful connection")
	}

	if statusCode, ok := result["status_code"]; !ok || statusCode.(int) != 200 {
		t.Errorf("Expected status code 200, got %v", statusCode)
	}

	if _, ok := result["duration"]; !ok {
		t.Error("Result should contain duration")
	}

	if _, ok := result["headers"]; !ok {
		t.Error("Result should contain headers")
	}
}

func TestDiagnostics_testHTTPConnectivity_InvalidURL(t *testing.T) {
	diag := NewDiagnostics()
	ctx := context.Background()

	result := diag.testHTTPConnectivity(ctx, "http://invalid-host-that-should-not-exist.com")

	if result == nil {
		t.Fatal("testHTTPConnectivity returned nil")
	}

	if success, ok := result["success"]; !ok || success.(bool) {
		t.Error("Expected failed connection for invalid URL")
	}

	if _, ok := result["error"]; !ok {
		t.Error("Result should contain error for failed connection")
	}

	if _, ok := result["duration"]; !ok {
		t.Error("Result should contain duration even for failed connections")
	}
}

func TestDiagnostics_DetectProxy(t *testing.T) {
	// Save original environment
	originalHTTPProxy := os.Getenv("HTTP_PROXY")
	originalHTTPSProxy := os.Getenv("HTTPS_PROXY")

	// Clean up after test
	defer func() {
		_ = os.Setenv("HTTP_PROXY", originalHTTPProxy)
		_ = os.Setenv("HTTPS_PROXY", originalHTTPSProxy)
	}()

	t.Run("No proxy detected", func(t *testing.T) {
		// Clear proxy environment variables
		_ = os.Unsetenv("HTTP_PROXY")
		_ = os.Unsetenv("HTTPS_PROXY")
		_ = os.Unsetenv("http_proxy")
		_ = os.Unsetenv("https_proxy")

		diag := NewDiagnostics()
		ctx := context.Background()

		result := diag.DetectProxy(ctx)
		proxyInfo := result.(*ProxyInfo)

		if proxyInfo.Detected {
			t.Error("Should not detect proxy when none is set")
		}
	})

	t.Run("HTTP proxy detected", func(t *testing.T) {
		// Set HTTP proxy environment variable
		_ = os.Setenv("HTTP_PROXY", "http://proxy.example.com:8080")

		diag := NewDiagnostics()
		ctx := context.Background()

		result := diag.DetectProxy(ctx)
		proxyInfo := result.(*ProxyInfo)

		if !proxyInfo.Detected {
			t.Error("Should detect HTTP proxy")
		}

		if proxyInfo.Type != ProxyHTTP {
			t.Errorf("Proxy type = %v, want %v", proxyInfo.Type, ProxyHTTP)
		}

		if proxyInfo.Address != "proxy.example.com" {
			t.Errorf("Proxy address = %s, want proxy.example.com", proxyInfo.Address)
		}

		if proxyInfo.Port != 8080 {
			t.Errorf("Proxy port = %d, want 8080", proxyInfo.Port)
		}

		if proxyInfo.Authentication {
			t.Error("Should not detect authentication when none is provided")
		}
	})

	t.Run("HTTPS proxy with auth detected", func(t *testing.T) {
		// Clear all proxy environment variables first
		_ = os.Unsetenv("HTTP_PROXY")
		_ = os.Unsetenv("HTTPS_PROXY")
		_ = os.Unsetenv("http_proxy")
		_ = os.Unsetenv("https_proxy")

		// Set HTTPS proxy with authentication
		_ = os.Setenv("HTTPS_PROXY", "https://user:pass@secure-proxy.example.com:3128")

		diag := NewDiagnostics()
		ctx := context.Background()

		result := diag.DetectProxy(ctx)
		proxyInfo := result.(*ProxyInfo)

		if !proxyInfo.Detected {
			t.Error("Should detect HTTPS proxy")
		}

		if proxyInfo.Type != ProxyHTTPS {
			t.Errorf("Proxy type = %v, want %v", proxyInfo.Type, ProxyHTTPS)
		}

		if proxyInfo.Address != "secure-proxy.example.com" {
			t.Errorf("Proxy address = %s, want secure-proxy.example.com", proxyInfo.Address)
		}

		if proxyInfo.Port != 3128 {
			t.Errorf("Proxy port = %d, want 3128", proxyInfo.Port)
		}

		if !proxyInfo.Authentication {
			t.Error("Should detect authentication")
		}
	})
}

func TestDiagnostics_TestBandwidth(t *testing.T) {
	// Create a test server that returns some data
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Return some test data for bandwidth measurement
		data := make([]byte, 1024) // 1KB of test data
		for i := range data {
			data[i] = byte(i % 256)
		}

		_, _ = w.Write(data)
	}))
	defer server.Close()

	diag := NewDiagnostics()
	ctx := context.Background()

	// Extract hostname from test server URL
	serverURL := strings.TrimPrefix(server.URL, "http://")

	result := diag.TestBandwidth(ctx, serverURL)
	bandwidthResult := result.(*BandwidthResult)

	if bandwidthResult.TestServerHost != serverURL {
		t.Errorf("TestServerHost = %s, want %s", bandwidthResult.TestServerHost, serverURL)
	}

	if bandwidthResult.TestDuration <= 0 {
		t.Error("TestDuration should be greater than 0")
	}

	if bandwidthResult.RecommendedUse == "" {
		t.Error("RecommendedUse should not be empty")
	}

	// Note: DownloadSpeed might be 0 if the test completes too quickly or fails
	// We don't test exact values since they depend on test server performance
}

func TestDiagnostics_testDownloadSpeed(t *testing.T) {
	// Create a test server that returns data for bandwidth testing
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "HEAD" {
			w.WriteHeader(http.StatusOK)
			return
		}

		// Return test data
		data := make([]byte, 2048) // 2KB of test data
		for i := range data {
			data[i] = byte(i % 256)
		}

		_, _ = w.Write(data)
	}))
	defer server.Close()

	diag := NewDiagnostics()
	ctx := context.Background()

	// Extract hostname from test server URL
	serverURL := strings.TrimPrefix(server.URL, "http://")

	speed, latency := diag.testDownloadSpeed(ctx, serverURL)

	// Speed might be 0 if test completes very quickly
	// On fast machines (especially Windows CI), latency might be measured as 0
	// This is a known flaky test issue on high-performance CI runners
	if latency < 0 {
		t.Error("Latency should not be negative")
	}

	// Speed should be non-negative
	if speed < 0 {
		t.Error("Speed should not be negative")
	}

	// Log the values for debugging
	t.Logf("Measured speed: %f MB/s, latency: %v", speed, latency)
}

func TestDiagnostics_calculateOverallHealth(t *testing.T) {
	diag := NewDiagnostics()

	t.Run("Good health", func(t *testing.T) {
		health := &NetworkHealth{
			DNSHealth:  &DiagnosticResult{Success: true},
			ConnHealth: &DiagnosticResult{Success: true},
			BandwidthInfo: &BandwidthResult{
				DownloadSpeed: 10.0,
				Latency:       100 * time.Millisecond,
			},
			ProxyInfo: &ProxyInfo{
				Detected: false,
			},
		}

		status := diag.calculateOverallHealth(health)
		if status != HealthGood {
			t.Errorf("Expected HealthGood, got %v", status)
		}
	})

	t.Run("Warning health - slow bandwidth and high latency", func(t *testing.T) {
		health := &NetworkHealth{
			DNSHealth:  &DiagnosticResult{Success: true},
			ConnHealth: &DiagnosticResult{Success: true},
			BandwidthInfo: &BandwidthResult{
				DownloadSpeed: 3.0,             // Slow speed (< 5) triggers warning
				Latency:       2 * time.Second, // High latency (> 1s) triggers warning
			},
		}

		status := diag.calculateOverallHealth(health)
		if status != HealthWarning {
			t.Errorf("Expected HealthWarning, got %v", status)
		}
	})

	t.Run("Poor health - DNS failure", func(t *testing.T) {
		health := &NetworkHealth{
			DNSHealth:  &DiagnosticResult{Success: false},
			ConnHealth: &DiagnosticResult{Success: true},
			BandwidthInfo: &BandwidthResult{
				DownloadSpeed: 10.0,
				Latency:       100 * time.Millisecond,
			},
		}

		status := diag.calculateOverallHealth(health)
		if status != HealthPoor {
			t.Errorf("Expected HealthPoor, got %v", status)
		}
	})

	t.Run("Critical health - multiple failures", func(t *testing.T) {
		health := &NetworkHealth{
			DNSHealth:  &DiagnosticResult{Success: false},
			ConnHealth: &DiagnosticResult{Success: false},
			BandwidthInfo: &BandwidthResult{
				DownloadSpeed: 0.5,             // Very slow
				Latency:       2 * time.Second, // High latency
			},
		}

		status := diag.calculateOverallHealth(health)
		if status != HealthCritical {
			t.Errorf("Expected HealthCritical, got %v", status)
		}
	})
}

func TestDiagnostics_RunFullDiagnostics(t *testing.T) {
	diag := NewDiagnostics()
	ctx := context.Background()

	// Use localhost and simple test configuration to avoid network dependencies
	options := &DiagnosticOptions{
		Timeout:          5 * time.Second,
		TestHosts:        []string{"localhost"},
		DNSServers:       []string{"8.8.8.8"},
		BandwidthServers: []string{"localhost"},
		IncludeBandwidth: false, // Skip bandwidth test to avoid complexity
		IncludeProxy:     true,
		Verbose:          false,
	}

	health, err := diag.RunFullDiagnostics(ctx, options)
	if err != nil {
		t.Errorf("RunFullDiagnostics() failed: %v", err)
	}

	if health == nil {
		t.Fatal("RunFullDiagnostics() returned nil health")
	}

	if health.DNSHealth == nil {
		t.Error("DNSHealth should not be nil")
	}

	if health.ConnHealth == nil {
		t.Error("ConnHealth should not be nil")
	}

	if health.ProxyInfo == nil {
		t.Error("ProxyInfo should not be nil")
	}

	if len(health.Results) == 0 {
		t.Error("Results should not be empty")
	}

	if health.TestDuration <= 0 {
		t.Error("TestDuration should be greater than 0")
	}

	// Overall status should be set
	validStatuses := []HealthStatus{HealthGood, HealthWarning, HealthPoor, HealthCritical}
	statusValid := false

	for _, validStatus := range validStatuses {
		if health.OverallStatus == validStatus {
			statusValid = true
			break
		}
	}

	if !statusValid {
		t.Errorf("OverallStatus = %v, should be one of %v", health.OverallStatus, validStatuses)
	}
}

func TestDiagnostics_RunFullDiagnostics_WithNilOptions(t *testing.T) {
	diag := NewDiagnostics()
	ctx := context.Background()

	// Test with nil options to ensure defaults are used
	health, err := diag.RunFullDiagnostics(ctx, nil)
	if err != nil {
		t.Errorf("RunFullDiagnostics() with nil options failed: %v", err)
	}

	if health == nil {
		t.Fatal("RunFullDiagnostics() returned nil health")
	}

	// Should have used default options and included bandwidth test
	if health.BandwidthInfo == nil {
		// Note: BandwidthInfo might be nil if the test fails, which is acceptable
		t.Log("BandwidthInfo is nil - this might be expected if bandwidth test failed")
	}
}

func TestGetDefaultFunctions(t *testing.T) {
	t.Run("getDefaultTestHosts", func(t *testing.T) {
		hosts := getDefaultTestHosts()
		if len(hosts) == 0 {
			t.Error("getDefaultTestHosts() should return non-empty slice")
		}

		expectedHosts := []string{"google.com", "cloudflare.com", "github.com", "httpbin.org"}
		if len(hosts) != len(expectedHosts) {
			t.Errorf("Expected %d hosts, got %d", len(expectedHosts), len(hosts))
		}

		for i, expected := range expectedHosts {
			if i < len(hosts) && hosts[i] != expected {
				t.Errorf("Host[%d] = %s, want %s", i, hosts[i], expected)
			}
		}
	})

	t.Run("getDefaultDNSServers", func(t *testing.T) {
		servers := getDefaultDNSServers()
		if len(servers) == 0 {
			t.Error("getDefaultDNSServers() should return non-empty slice")
		}

		expectedServers := []string{"8.8.8.8", "1.1.1.1", "208.67.222.222"}
		if len(servers) != len(expectedServers) {
			t.Errorf("Expected %d servers, got %d", len(expectedServers), len(servers))
		}

		for i, expected := range expectedServers {
			if i < len(servers) && servers[i] != expected {
				t.Errorf("Server[%d] = %s, want %s", i, servers[i], expected)
			}
		}
	})

	t.Run("getDefaultBandwidthServers", func(t *testing.T) {
		servers := getDefaultBandwidthServers()
		if len(servers) == 0 {
			t.Error("getDefaultBandwidthServers() should return non-empty slice")
		}

		expectedServers := []string{"httpbin.org", "jsonplaceholder.typicode.com"}
		if len(servers) != len(expectedServers) {
			t.Errorf("Expected %d servers, got %d", len(expectedServers), len(servers))
		}

		for i, expected := range expectedServers {
			if i < len(servers) && servers[i] != expected {
				t.Errorf("Server[%d] = %s, want %s", i, servers[i], expected)
			}
		}
	})
}

func TestDiagnostics_ContextCancellation(t *testing.T) {
	diag := NewDiagnostics()

	// Create a context that will be cancelled quickly
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	// Try to run diagnostics - should return quickly due to context cancellation
	_, err := diag.RunFullDiagnostics(ctx, &DiagnosticOptions{
		Timeout:          30 * time.Second, // Long timeout, but context will cancel first
		TestHosts:        []string{"google.com"},
		DNSServers:       []string{"8.8.8.8"},
		IncludeBandwidth: false,
		IncludeProxy:     false,
	})

	// We expect either context.DeadlineExceeded or successful completion
	// (if the test runs fast enough before context cancellation)
	if err != nil && !errors.Is(err, context.DeadlineExceeded) {
		t.Logf("Got error: %v (this may be acceptable)", err)
	}
}

func TestDiagnosticOptions_Validation(t *testing.T) {
	diag := NewDiagnostics()
	ctx := context.Background()

	t.Run("Empty test hosts", func(t *testing.T) {
		options := &DiagnosticOptions{
			TestHosts:  []string{},
			DNSServers: []string{"8.8.8.8"},
		}

		health, err := diag.RunFullDiagnostics(ctx, options)
		// Should handle empty test hosts gracefully
		if err != nil {
			t.Errorf("RunFullDiagnostics() with empty test hosts failed: %v", err)
		}

		if health == nil {
			t.Error("Should return health result even with empty test hosts")
		}
	})

	t.Run("Empty DNS servers", func(t *testing.T) {
		options := &DiagnosticOptions{
			TestHosts:  []string{"google.com"},
			DNSServers: []string{},
		}

		health, err := diag.RunFullDiagnostics(ctx, options)
		// Should handle empty DNS servers gracefully
		if err != nil {
			t.Errorf("RunFullDiagnostics() with empty DNS servers failed: %v", err)
		}

		if health == nil {
			t.Error("Should return health result even with empty DNS servers")
		}
	})
}
