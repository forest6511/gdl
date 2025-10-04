package testing

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestNewErrorSimulator(t *testing.T) {
	config := DefaultSimulationConfig()
	sim := NewErrorSimulator(config)

	if sim == nil {
		t.Fatal("NewErrorSimulator should not return nil")
	}

	if sim.httpServer == nil {
		t.Error("ErrorSimulator should have HTTP server")
	}

	defer sim.Close()
}

func TestNewErrorSimulator_NilConfig(t *testing.T) {
	sim := NewErrorSimulator(nil)

	if sim == nil {
		t.Fatal("NewErrorSimulator should not return nil even with nil config")
	}

	defer sim.Close()
}

func TestDefaultSimulationConfig(t *testing.T) {
	config := DefaultSimulationConfig()

	if config == nil {
		t.Fatal("DefaultSimulationConfig should not return nil")
	}

	if config.MaxResponseSize != 1024*1024 {
		t.Errorf("Expected MaxResponseSize 1MB, got %d", config.MaxResponseSize)
	}

	if config.HTTPStatusCode != 200 {
		t.Errorf("Expected HTTPStatusCode 200, got %d", config.HTTPStatusCode)
	}

	if config.FailAfterBytes != -1 {
		t.Errorf("Expected FailAfterBytes -1, got %d", config.FailAfterBytes)
	}
}

func TestErrorSimulator_GetURL(t *testing.T) {
	sim := NewErrorSimulator(nil)
	defer sim.Close()

	url := sim.GetURL()
	if url == "" {
		t.Error("GetURL should return non-empty URL")
	}

	if !strings.HasPrefix(url, "http://") {
		t.Error("GetURL should return HTTP URL")
	}
}

func TestErrorSimulator_GetEndpoint(t *testing.T) {
	sim := NewErrorSimulator(nil)
	defer sim.Close()

	endpoint := sim.GetEndpoint("/test")
	baseURL := sim.GetURL()
	expected := baseURL + "/test"

	if endpoint != expected {
		t.Errorf("Expected endpoint %s, got %s", expected, endpoint)
	}
}

func TestErrorSimulator_NormalResponse(t *testing.T) {
	config := &SimulationConfig{
		MaxResponseSize: 1024,
		HTTPStatusCode:  200,
		FailureRate:     0.0, // No failures
	}

	sim := NewErrorSimulator(config)
	defer sim.Close()

	resp, err := http.Get(sim.GetURL())
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		t.Errorf("Expected status code 200, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	if len(body) != 1024 {
		t.Errorf("Expected body length 1024, got %d", len(body))
	}
}

func TestErrorSimulator_HTTPStatusHandler(t *testing.T) {
	sim := NewErrorSimulator(nil)
	defer sim.Close()

	testCases := []int{400, 401, 403, 404, 429, 500, 502, 503, 504}

	for _, statusCode := range testCases {
		endpoint := sim.GetEndpoint("/status/" + string(rune(statusCode+'0')))
		if statusCode >= 100 { // Fix the conversion for multi-digit numbers
			endpoint = sim.GetEndpoint(
				"/status/" + string(
					rune(statusCode/100+'0'),
				) + string(
					rune((statusCode%100)/10+'0'),
				) + string(
					rune(statusCode%10+'0'),
				),
			)
		}

		// Use correct endpoint format
		switch statusCode {
		case 400:
			endpoint = sim.GetEndpoint("/status/400")
		case 401:
			endpoint = sim.GetEndpoint("/status/401")
		case 403:
			endpoint = sim.GetEndpoint("/status/403")
		case 404:
			endpoint = sim.GetEndpoint("/status/404")
		case 429:
			endpoint = sim.GetEndpoint("/status/429")
		case 500:
			endpoint = sim.GetEndpoint("/status/500")
		case 502:
			endpoint = sim.GetEndpoint("/status/502")
		case 503:
			endpoint = sim.GetEndpoint("/status/503")
		case 504:
			endpoint = sim.GetEndpoint("/status/504")
		}

		resp, err := http.Get(endpoint)
		if err != nil {
			t.Fatalf("Request to %s failed: %v", endpoint, err)
		}

		_ = resp.Body.Close()

		if resp.StatusCode != statusCode {
			t.Errorf("Expected status code %d, got %d", statusCode, resp.StatusCode)
		}
	}
}

func TestErrorSimulator_NetworkErrorHandler(t *testing.T) {
	sim := NewErrorSimulator(nil)
	defer sim.Close()

	// Test network error endpoint
	resp, err := http.Get(sim.GetEndpoint("/network-error"))

	// This should either fail with connection error or return bad gateway
	if err == nil {
		defer func() { _ = resp.Body.Close() }()

		if resp.StatusCode != http.StatusBadGateway {
			t.Errorf("Expected status code %d, got %d", http.StatusBadGateway, resp.StatusCode)
		}
	}
	// If error occurs, that's also expected behavior for network error simulation
}

func TestErrorSimulator_SlowResponseHandler(t *testing.T) {
	t.Parallel()

	config := &SimulationConfig{
		SlowResponseRate: 1024, // 1KB/s
	}

	sim := NewErrorSimulator(config)
	defer sim.Close()

	start := time.Now()

	resp, err := http.Get(sim.GetEndpoint("/slow"))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	elapsed := time.Since(start)

	// Should take some time due to slow response rate
	if elapsed < 100*time.Millisecond {
		t.Error("Slow response should take longer than 100ms")
	}

	if len(body) == 0 {
		t.Error("Should receive response body")
	}
}

func TestErrorSimulator_PartialFailureHandler(t *testing.T) {
	config := &SimulationConfig{
		FailAfterBytes: 5000, // Fail after 5KB
	}

	sim := NewErrorSimulator(config)
	defer sim.Close()

	resp, err := http.Get(sim.GetEndpoint("/partial-failure"))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Try to read the response - should get partial data
	body, err := io.ReadAll(resp.Body)

	// Either we get an error (connection dropped) or partial data
	if err == nil {
		// If no error, we should have received some data but not the full amount
		if len(body) == 0 {
			t.Error("Should receive some partial data")
		}
	}
	// If error occurs, that's expected for partial failure simulation
}

func TestNewNetworkFailureSimulator(t *testing.T) {
	sim := NewNetworkFailureSimulator()

	if sim == nil {
		t.Fatal("NewNetworkFailureSimulator should not return nil")
	}

	if sim.originalDialer == nil {
		t.Error("NetworkFailureSimulator should have original dialer")
	}
}

func TestNetworkFailureSimulator_CreateFailingDialer(t *testing.T) {
	t.Parallel()

	sim := NewNetworkFailureSimulator()

	testCases := []string{
		"connection_refused",
		"timeout",
		"network_unreachable",
		"host_unreachable",
	}

	for _, failureType := range testCases {
		dialer := sim.CreateFailingDialer(failureType)

		if dialer == nil {
			t.Errorf("CreateFailingDialer should not return nil for %s", failureType)
		}

		// Test that dialer returns expected error
		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		conn, err := dialer(ctx, "tcp", "127.0.0.1:80")

		cancel()

		if err == nil {
			if conn != nil {
				_ = conn.Close()
			}

			t.Errorf("Expected error for failure type %s", failureType)
		}
	}
}

func TestNewDiskSpaceSimulator(t *testing.T) {
	spaceLimit := int64(1024)
	sim := NewDiskSpaceSimulator(spaceLimit)

	if sim == nil {
		t.Fatal("NewDiskSpaceSimulator should not return nil")
	}

	if sim.spaceLimit != spaceLimit {
		t.Errorf("Expected space limit %d, got %d", spaceLimit, sim.spaceLimit)
	}
}

func TestDiskSpaceSimulator_CreateLimitedWriter(t *testing.T) {
	spaceLimit := int64(100)
	sim := NewDiskSpaceSimulator(spaceLimit)

	var buffer strings.Builder

	writer := sim.CreateLimitedWriter(&buffer)

	if writer == nil {
		t.Fatal("CreateLimitedWriter should not return nil")
	}

	// Write data within limit
	data := make([]byte, 50)

	n, err := writer.Write(data)
	if err != nil {
		t.Errorf("Write within limit should succeed: %v", err)
	}

	if n != 50 {
		t.Errorf("Expected to write 50 bytes, wrote %d", n)
	}

	// Write data that exceeds limit
	largeData := make([]byte, 100)

	n, err = writer.Write(largeData)
	if err == nil {
		t.Error("Write exceeding limit should fail")
	}

	// Should have written partial data up to limit
	if n != 50 { // Remaining space after first write
		t.Errorf("Expected to write 50 bytes (remaining space), wrote %d", n)
	}
}

func TestDiskSpaceSimulator_Reset(t *testing.T) {
	sim := NewDiskSpaceSimulator(1024)

	var buffer strings.Builder

	writer := sim.CreateLimitedWriter(&buffer)

	// Write some data
	data := make([]byte, 100)
	_, _ = writer.Write(data)

	if sim.GetBytesWritten() != 100 {
		t.Errorf("Expected 100 bytes written, got %d", sim.GetBytesWritten())
	}

	// Reset
	sim.Reset()

	if sim.GetBytesWritten() != 0 {
		t.Errorf("Expected 0 bytes after reset, got %d", sim.GetBytesWritten())
	}
}

func TestNewHTTPErrorSimulator(t *testing.T) {
	sim := NewHTTPErrorSimulator()

	if sim == nil {
		t.Fatal("NewHTTPErrorSimulator should not return nil")
	}

	if sim.server == nil {
		t.Error("HTTPErrorSimulator should have server")
	}

	defer sim.Close()
}

func TestCreateErrorServer(t *testing.T) {
	server := CreateErrorServer(404, "Not Found")
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 404 {
		t.Errorf("Expected status code 404, got %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	if string(body) != "Not Found" {
		t.Errorf("Expected body 'Not Found', got %s", string(body))
	}
}

func TestCreateTimeoutServer(t *testing.T) {
	server := CreateTimeoutServer(50 * time.Millisecond)
	defer server.Close()

	client := &http.Client{
		Timeout: 20 * time.Millisecond, // Shorter than server delay
	}

	start := time.Now()
	resp, err := client.Get(server.URL)
	if resp != nil {
		_ = resp.Body.Close()
	}
	elapsed := time.Since(start)

	if err == nil {
		t.Error("Expected timeout error")
	}

	// Should timeout before server delay
	if elapsed >= 50*time.Millisecond {
		t.Error("Request should have timed out before server delay")
	}
}

func TestCreatePartialResponseServer(t *testing.T) {
	totalSize := int64(1000)
	sendSize := int64(500)

	server := CreatePartialResponseServer(totalSize, sendSize)
	defer server.Close()

	resp, err := http.Get(server.URL)
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Check content length header
	if resp.ContentLength != totalSize {
		t.Errorf("Expected content length %d, got %d", totalSize, resp.ContentLength)
	}

	// Read response - should get partial data or error
	body, err := io.ReadAll(resp.Body)
	if err == nil {
		// If no error, should have received partial data
		if int64(len(body)) > sendSize {
			t.Errorf("Should not receive more than %d bytes, got %d", sendSize, len(body))
		}
	}
	// If error occurs, that's expected for partial response
}

func TestGetCommonTestScenarios(t *testing.T) {
	scenarios := GetCommonTestScenarios()

	if len(scenarios) == 0 {
		t.Error("GetCommonTestScenarios should return scenarios")
	}

	// Check for expected scenarios
	scenarioNames := make(map[string]bool)
	for _, scenario := range scenarios {
		scenarioNames[scenario.Name] = true

		if scenario.Name == "" {
			t.Error("Scenario should have name")
		}

		if scenario.ExpectedErrorCode == 0 {
			t.Error("Scenario should have expected error code")
		}
	}

	expectedScenarios := []string{
		"Network Connection Refused",
		"Network Timeout",
		"HTTP 404 Not Found",
		"HTTP 500 Server Error",
		"Disk Space Exhaustion",
	}

	for _, expected := range expectedScenarios {
		if !scenarioNames[expected] {
			t.Errorf("Missing expected scenario: %s", expected)
		}
	}
}

func TestNewSimulatorSuite(t *testing.T) {
	suite := NewSimulatorSuite()

	if suite == nil {
		t.Fatal("NewSimulatorSuite should not return nil")
	}

	if suite.ErrorSim == nil {
		t.Error("SimulatorSuite should have ErrorSim")
	}

	if suite.NetworkSim == nil {
		t.Error("SimulatorSuite should have NetworkSim")
	}

	if suite.DiskSpaceSim == nil {
		t.Error("SimulatorSuite should have DiskSpaceSim")
	}

	if suite.HTTPSim == nil {
		t.Error("SimulatorSuite should have HTTPSim")
	}

	defer suite.Close()
}

func TestSimulatorSuite_Close(t *testing.T) {
	suite := NewSimulatorSuite()

	// Should not panic
	suite.Close()

	// Should be safe to call multiple times
	suite.Close()
}

func TestErrorSimulator_LargeFileHandler(t *testing.T) {
	config := &SimulationConfig{
		MaxResponseSize: 10 * 1024 * 1024, // 10MB
	}

	sim := NewErrorSimulator(config)
	defer sim.Close()

	resp, err := http.Get(sim.GetEndpoint("/large-file"))
	if err != nil {
		t.Fatalf("Request failed: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("Failed to read body: %v", err)
	}

	// Should receive large file
	if len(body) == 0 {
		t.Error("Should receive large file data")
	}
}

func TestErrorSimulator_RandomError(t *testing.T) {
	config := &SimulationConfig{
		FailureRate: 1.0, // 100% failure rate
	}

	sim := NewErrorSimulator(config)
	defer sim.Close()

	// Make multiple requests - with 100% failure rate, all should fail
	failureCount := 0
	for i := 0; i < 10; i++ {
		resp, err := http.Get(sim.GetURL())
		if err != nil {
			failureCount++
			continue
		}
		if resp.StatusCode >= 400 {
			failureCount++
		}
		_ = resp.Body.Close()
	}

	// With 100% failure rate, should have failures
	if failureCount == 0 {
		t.Error("Expected at least some failures with 100% failure rate")
	}
}

func TestTimeoutError_Methods(t *testing.T) {
	err := &timeoutError{}

	if err.Error() != "i/o timeout" {
		t.Errorf("Expected error message 'i/o timeout', got %s", err.Error())
	}

	if !err.Timeout() {
		t.Error("Expected Timeout() to return true")
	}

	if err.Temporary() {
		t.Error("Expected Temporary() to return false")
	}
}
