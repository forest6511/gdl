// Package testing provides error simulation tools for testing download scenarios.
package testing

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/forest6511/godl/pkg/errors"
)

// ErrorSimulator provides comprehensive error simulation capabilities for testing
// download scenarios including network failures, disk space issues, HTTP errors,
// and timeout conditions.
type ErrorSimulator struct {
	httpServer     *httptest.Server
	networkLatency time.Duration
	failureRate    float64 // 0.0 to 1.0
	mu             sync.RWMutex
	requestCount   int
}

// SimulationConfig configures the error simulation behavior.
type SimulationConfig struct {
	// NetworkLatency adds artificial latency to responses
	NetworkLatency time.Duration

	// FailureRate determines the probability of failures (0.0 to 1.0)
	FailureRate float64

	// MaxResponseSize limits response body size
	MaxResponseSize int64

	// ChunkDelay adds delay between response chunks
	ChunkDelay time.Duration

	// TimeoutAfter causes requests to timeout after this duration
	TimeoutAfter time.Duration

	// HTTPStatusCode overrides the HTTP status code to return
	HTTPStatusCode int

	// FailAfterBytes causes failure after reading a certain number of bytes
	FailAfterBytes int64

	// DiskSpaceLimit simulates insufficient disk space
	DiskSpaceLimit uint64

	// EnableConnectionReset simulates connection resets
	EnableConnectionReset bool

	// EnableDNSFailure simulates DNS resolution failures
	EnableDNSFailure bool

	// SlowResponseRate limits response byte rate (bytes per second)
	SlowResponseRate int64
}

// DefaultSimulationConfig returns a basic simulation configuration.
func DefaultSimulationConfig() *SimulationConfig {
	return &SimulationConfig{
		NetworkLatency:  0,
		FailureRate:     0.0,
		MaxResponseSize: 1024 * 1024, // 1MB
		ChunkDelay:      0,
		HTTPStatusCode:  200,
		FailAfterBytes:  -1, // No failure
	}
}

// NewErrorSimulator creates a new error simulator with the given configuration.
func NewErrorSimulator(config *SimulationConfig) *ErrorSimulator {
	if config == nil {
		config = DefaultSimulationConfig()
	}

	sim := &ErrorSimulator{
		networkLatency: config.NetworkLatency,
		failureRate:    config.FailureRate,
	}

	// Create HTTP server with error simulation handlers
	mux := http.NewServeMux()
	mux.HandleFunc("/", sim.createErrorHandler(config))
	mux.HandleFunc("/network-error", sim.networkErrorHandler)
	mux.HandleFunc("/timeout", sim.timeoutHandler(config))
	mux.HandleFunc("/slow", sim.slowResponseHandler(config))
	mux.HandleFunc("/status/", sim.httpStatusHandler)
	mux.HandleFunc("/partial-failure", sim.partialFailureHandler(config))
	mux.HandleFunc("/large-file", sim.largeFileHandler(config))

	sim.httpServer = httptest.NewServer(mux)

	return sim
}

// Close shuts down the error simulator.
func (es *ErrorSimulator) Close() {
	if es.httpServer != nil {
		es.httpServer.Close()
	}
}

// GetURL returns the base URL of the test server.
func (es *ErrorSimulator) GetURL() string {
	return es.httpServer.URL
}

// GetEndpoint returns a specific endpoint URL.
func (es *ErrorSimulator) GetEndpoint(path string) string {
	return es.httpServer.URL + path
}

// createErrorHandler creates a handler that simulates various error conditions.
func (es *ErrorSimulator) createErrorHandler(config *SimulationConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		es.mu.Lock()
		es.requestCount++
		currentCount := es.requestCount
		es.mu.Unlock()

		// Add artificial latency
		if config.NetworkLatency > 0 {
			time.Sleep(config.NetworkLatency)
		}

		// Simulate random failures based on failure rate
		if es.shouldFail(currentCount, config.FailureRate) {
			es.simulateRandomError(w, r, config)
			return
		}

		// Simulate timeout
		if config.TimeoutAfter > 0 {
			time.Sleep(config.TimeoutAfter)
		}

		// Return configured HTTP status
		if config.HTTPStatusCode != 200 {
			w.WriteHeader(config.HTTPStatusCode)
			_, _ = fmt.Fprintf(w, "Simulated HTTP %d response", config.HTTPStatusCode)

			return
		}

		// Return normal response
		es.normalResponseHandler(w, r, config)
	}
}

// shouldFail determines if a request should fail based on failure rate.
func (es *ErrorSimulator) shouldFail(requestCount int, failureRate float64) bool {
	if failureRate <= 0 {
		return false
	}

	if failureRate >= 1.0 {
		return true
	}

	// Use request count as seed for deterministic behavior in tests
	return float64(requestCount%100)/100.0 < failureRate
}

// simulateRandomError simulates various types of random errors.
func (es *ErrorSimulator) simulateRandomError(
	w http.ResponseWriter,
	r *http.Request,
	config *SimulationConfig,
) {
	errorType := es.requestCount % 4

	switch errorType {
	case 0:
		// Connection reset
		if hj, ok := w.(http.Hijacker); ok {
			if conn, _, err := hj.Hijack(); err == nil {
				_ = conn.Close()
				return
			}
		}

		w.WriteHeader(http.StatusInternalServerError)
	case 1:
		// Timeout simulation
		time.Sleep(100 * time.Millisecond)
	case 2:
		// HTTP error status
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = fmt.Fprint(w, "Internal server error")
	case 3:
		// Partial response then disconnect
		w.Header().Set("Content-Length", "1000000")
		_, _ = fmt.Fprint(w, "Partial data")

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
		// Simulate connection drop by panicking
		panic("simulated connection drop")
	}
}

// normalResponseHandler serves a normal response.
func (es *ErrorSimulator) normalResponseHandler(
	w http.ResponseWriter,
	r *http.Request,
	config *SimulationConfig,
) {
	// Determine response size
	responseSize := int64(1024) // Default 1KB
	if config.MaxResponseSize > 0 {
		responseSize = config.MaxResponseSize
	}

	// Set headers
	w.Header().Set("Content-Length", fmt.Sprintf("%d", responseSize))
	w.Header().Set("Accept-Ranges", "bytes")
	w.WriteHeader(200)

	// Write response data in chunks
	chunkSize := int64(8192) // 8KB chunks
	written := int64(0)

	for written < responseSize {
		remaining := responseSize - written
		if remaining < chunkSize {
			chunkSize = remaining
		}

		// Create chunk data
		chunk := make([]byte, chunkSize)
		for i := range chunk {
			chunk[i] = byte('A' + (written+int64(i))%26)
		}

		// Check if we should fail after certain bytes
		if config.FailAfterBytes > 0 && written >= config.FailAfterBytes {
			// Simulate connection drop
			return
		}

		// Write chunk
		n, err := w.Write(chunk)
		if err != nil {
			return
		}

		written += int64(n)

		// Add chunk delay if configured
		if config.ChunkDelay > 0 {
			time.Sleep(config.ChunkDelay)
		}

		// Simulate slow response rate
		if config.SlowResponseRate > 0 {
			expectedDuration := time.Duration(
				chunkSize*1000/config.SlowResponseRate,
			) * time.Millisecond
			time.Sleep(expectedDuration)
		}

		// Flush data
		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}
	}
}

// networkErrorHandler simulates network-level errors.
func (es *ErrorSimulator) networkErrorHandler(w http.ResponseWriter, r *http.Request) {
	// Simulate connection reset by hijacking and closing connection
	if hj, ok := w.(http.Hijacker); ok {
		if conn, _, err := hj.Hijack(); err == nil {
			_ = conn.Close()
			return
		}
	}

	// Fallback to HTTP error
	w.WriteHeader(http.StatusBadGateway)
	_, _ = fmt.Fprint(w, "Network error simulation")
}

// timeoutHandler simulates timeout scenarios.
func (es *ErrorSimulator) timeoutHandler(config *SimulationConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		timeout := 30 * time.Second
		if config.TimeoutAfter > 0 {
			timeout = config.TimeoutAfter
		}

		// Start writing response but then timeout
		w.Header().Set("Content-Length", "1000000")
		w.WriteHeader(200)
		_, _ = fmt.Fprint(w, "Starting response...")

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		// Wait for timeout
		time.Sleep(timeout)

		// Connection should be dropped by client timeout
	}
}

// slowResponseHandler simulates slow network responses.
func (es *ErrorSimulator) slowResponseHandler(config *SimulationConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		responseSize := int64(10240) // 10KB
		chunkSize := int64(1024)     // 1KB chunks
		delay := 1 * time.Second     // 1 second between chunks

		if config.SlowResponseRate > 0 {
			// Calculate delay based on desired rate
			delay = time.Duration(chunkSize*1000/config.SlowResponseRate) * time.Millisecond
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", responseSize))
		w.WriteHeader(200)

		written := int64(0)
		for written < responseSize {
			remaining := responseSize - written
			if remaining < chunkSize {
				chunkSize = remaining
			}

			// Write chunk
			chunk := make([]byte, chunkSize)
			for i := range chunk {
				chunk[i] = byte('X')
			}

			_, _ = w.Write(chunk)
			written += chunkSize

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

			// Add delay between chunks
			if written < responseSize {
				time.Sleep(delay)
			}
		}
	}
}

// httpStatusHandler returns specific HTTP status codes.
func (es *ErrorSimulator) httpStatusHandler(w http.ResponseWriter, r *http.Request) {
	// Extract status code from path like /status/404
	path := strings.TrimPrefix(r.URL.Path, "/status/")
	statusCode := 500 // default

	switch path {
	case "400":
		statusCode = 400
	case "401":
		statusCode = 401
	case "403":
		statusCode = 403
	case "404":
		statusCode = 404
	case "429":
		statusCode = 429
	case "500":
		statusCode = 500
	case "502":
		statusCode = 502
	case "503":
		statusCode = 503
	case "504":
		statusCode = 504
	}

	w.WriteHeader(statusCode)
	_, _ = fmt.Fprintf(w, "HTTP %d response for testing", statusCode)
}

// partialFailureHandler simulates partial download failures.
func (es *ErrorSimulator) partialFailureHandler(config *SimulationConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		totalSize := int64(100000) // 100KB
		failAt := totalSize / 2    // Fail at 50%

		if config.FailAfterBytes > 0 {
			failAt = config.FailAfterBytes
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", totalSize))
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(200)

		written := int64(0)
		chunkSize := int64(1024)

		for written < failAt && written < totalSize {
			remaining := min(chunkSize, failAt-written)

			chunk := make([]byte, remaining)
			for i := range chunk {
				chunk[i] = byte('P') // P for partial
			}

			_, _ = w.Write(chunk)
			written += remaining

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
		}

		// Simulate failure by closing connection
		if hj, ok := w.(http.Hijacker); ok {
			if conn, _, err := hj.Hijack(); err == nil {
				_ = conn.Close()
			}
		}
	}
}

// largeFileHandler simulates downloading large files.
func (es *ErrorSimulator) largeFileHandler(config *SimulationConfig) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		fileSize := int64(10 * 1024 * 1024) // 10MB
		if config.MaxResponseSize > 0 {
			fileSize = config.MaxResponseSize
		}

		w.Header().Set("Content-Length", fmt.Sprintf("%d", fileSize))
		w.Header().Set("Accept-Ranges", "bytes")
		w.WriteHeader(200)

		// Stream large file content
		chunkSize := int64(32768) // 32KB chunks
		written := int64(0)

		for written < fileSize {
			remaining := fileSize - written
			if remaining < chunkSize {
				chunkSize = remaining
			}

			chunk := make([]byte, chunkSize)
			// Fill with pattern data
			for i := range chunk {
				chunk[i] = byte((written + int64(i)) % 256)
			}

			n, err := w.Write(chunk)
			if err != nil {
				return
			}

			written += int64(n)

			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}

			// Add small delay for large files to simulate real network conditions
			if config.ChunkDelay > 0 {
				time.Sleep(config.ChunkDelay)
			}
		}
	}
}

// min returns the minimum of two int64 values.
func min(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}

// NetworkFailureSimulator simulates various network failure scenarios.
type NetworkFailureSimulator struct {
	originalDialer *net.Dialer
}

// NewNetworkFailureSimulator creates a network failure simulator.
func NewNetworkFailureSimulator() *NetworkFailureSimulator {
	return &NetworkFailureSimulator{
		originalDialer: &net.Dialer{Timeout: 30 * time.Second},
	}
}

// CreateFailingDialer returns a dialer that simulates network failures.
func (nfs *NetworkFailureSimulator) CreateFailingDialer(
	failureType string,
) func(context.Context, string, string) (net.Conn, error) {
	return func(ctx context.Context, network, addr string) (net.Conn, error) {
		switch failureType {
		case "connection_refused":
			return nil, &net.OpError{
				Op:     "dial",
				Net:    network,
				Source: nil,
				Addr:   &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 80},
				Err:    syscall.ECONNREFUSED,
			}
		case "timeout":
			time.Sleep(100 * time.Millisecond) // Simulate timeout quickly

			return nil, &net.OpError{
				Op:     "dial",
				Net:    network,
				Source: nil,
				Addr:   &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 80},
				Err:    &timeoutError{},
			}
		case "network_unreachable":
			return nil, &net.OpError{
				Op:     "dial",
				Net:    network,
				Source: nil,
				Addr:   &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 80},
				Err:    syscall.ENETUNREACH,
			}
		case "host_unreachable":
			return nil, &net.OpError{
				Op:     "dial",
				Net:    network,
				Source: nil,
				Addr:   &net.TCPAddr{IP: net.ParseIP("192.168.1.1"), Port: 80},
				Err:    syscall.EHOSTUNREACH,
			}
		default:
			// Normal connection
			return nfs.originalDialer.DialContext(ctx, network, addr)
		}
	}
}

// timeoutError implements net.Error for timeout simulation.
type timeoutError struct{}

func (e *timeoutError) Error() string   { return "i/o timeout" }
func (e *timeoutError) Timeout() bool   { return true }
func (e *timeoutError) Temporary() bool { return false }

// DiskSpaceSimulator simulates disk space exhaustion scenarios.
type DiskSpaceSimulator struct {
	originalWriter io.Writer
	bytesWritten   int64
	spaceLimit     int64
	mu             sync.Mutex
}

// NewDiskSpaceSimulator creates a disk space simulator.
func NewDiskSpaceSimulator(spaceLimit int64) *DiskSpaceSimulator {
	return &DiskSpaceSimulator{
		spaceLimit: spaceLimit,
	}
}

// CreateLimitedWriter returns a writer that simulates disk space exhaustion.
func (dss *DiskSpaceSimulator) CreateLimitedWriter(w io.Writer) io.Writer {
	dss.originalWriter = w

	return &limitedWriter{
		simulator: dss,
		writer:    w,
	}
}

type limitedWriter struct {
	simulator *DiskSpaceSimulator
	writer    io.Writer
}

func (lw *limitedWriter) Write(p []byte) (int, error) {
	lw.simulator.mu.Lock()
	defer lw.simulator.mu.Unlock()

	// Check if writing would exceed space limit
	if lw.simulator.bytesWritten+int64(len(p)) > lw.simulator.spaceLimit {
		// Calculate how much we can write before hitting limit
		canWrite := lw.simulator.spaceLimit - lw.simulator.bytesWritten
		if canWrite <= 0 {
			return 0, &os.PathError{
				Op:   "write",
				Path: "simulated",
				Err:  syscall.ENOSPC, // No space left on device
			}
		}

		// Write what we can, then fail
		n, err := lw.writer.Write(p[:canWrite])

		lw.simulator.bytesWritten += int64(n)
		if err != nil {
			return n, err
		}

		// Return disk space error
		return n, &os.PathError{
			Op:   "write",
			Path: "simulated",
			Err:  syscall.ENOSPC,
		}
	}

	// Normal write
	n, err := lw.writer.Write(p)
	lw.simulator.bytesWritten += int64(n)

	return n, err
}

// Reset resets the bytes written counter.
func (dss *DiskSpaceSimulator) Reset() {
	dss.mu.Lock()
	defer dss.mu.Unlock()

	dss.bytesWritten = 0
}

// GetBytesWritten returns the total bytes written.
func (dss *DiskSpaceSimulator) GetBytesWritten() int64 {
	dss.mu.Lock()
	defer dss.mu.Unlock()

	return dss.bytesWritten
}

// HTTPErrorSimulator provides utilities for testing HTTP error responses.
type HTTPErrorSimulator struct {
	server *httptest.Server
}

// NewHTTPErrorSimulator creates an HTTP error simulator.
func NewHTTPErrorSimulator() *HTTPErrorSimulator {
	sim := &HTTPErrorSimulator{}

	mux := http.NewServeMux()
	mux.HandleFunc("/", sim.handleRequest)
	sim.server = httptest.NewServer(mux)

	return sim
}

// Close shuts down the HTTP error simulator.
func (hes *HTTPErrorSimulator) Close() {
	if hes.server != nil {
		hes.server.Close()
	}
}

// GetURL returns the server URL.
func (hes *HTTPErrorSimulator) GetURL() string {
	return hes.server.URL
}

// handleRequest handles all requests and can return various HTTP errors.
func (hes *HTTPErrorSimulator) handleRequest(w http.ResponseWriter, r *http.Request) {
	// This is a basic handler - in tests, you would override this behavior
	// by stopping this server and creating new ones with specific handlers
	w.WriteHeader(200)
	_, _ = fmt.Fprint(w, "Default response")
}

// CreateErrorServer creates a server that returns a specific HTTP error.
func CreateErrorServer(statusCode int, responseBody string) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(statusCode)

		if responseBody != "" {
			_, _ = fmt.Fprint(w, responseBody)
		}
	}))
}

// CreateTimeoutServer creates a server that times out.
func CreateTimeoutServer(delay time.Duration) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(delay)
		w.WriteHeader(200)
		_, _ = fmt.Fprint(w, "Delayed response")
	}))
}

// CreatePartialResponseServer creates a server that sends partial responses.
func CreatePartialResponseServer(totalSize, sendSize int64) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", fmt.Sprintf("%d", totalSize))
		w.WriteHeader(200)

		// Send only partial data
		data := make([]byte, sendSize)
		for i := range data {
			data[i] = byte('X')
		}

		_, _ = w.Write(data)

		if f, ok := w.(http.Flusher); ok {
			f.Flush()
		}

		// Simulate connection drop by hijacking and closing
		if hj, ok := w.(http.Hijacker); ok {
			if conn, _, err := hj.Hijack(); err == nil {
				_ = conn.Close()
			}
		}
	}))
}

// TestScenario represents a complete test scenario with multiple error conditions.
type TestScenario struct {
	Name              string
	NetworkFailure    string           // Type of network failure to simulate
	HTTPStatusCode    int              // HTTP status code to return
	DiskSpaceLimit    int64            // Disk space limit in bytes
	TimeoutAfter      time.Duration    // Timeout duration
	PartialFailureAt  int64            // Bytes after which to fail
	ExpectedErrorCode errors.ErrorCode // Expected error code
	ExpectedRetryable bool             // Whether error should be retryable
}

// GetCommonTestScenarios returns a set of common error scenarios for testing.
func GetCommonTestScenarios() []TestScenario {
	return []TestScenario{
		{
			Name:              "Network Connection Refused",
			NetworkFailure:    "connection_refused",
			ExpectedErrorCode: errors.CodeNetworkError,
			ExpectedRetryable: true,
		},
		{
			Name:              "Network Timeout",
			NetworkFailure:    "timeout",
			ExpectedErrorCode: errors.CodeTimeout,
			ExpectedRetryable: true,
		},
		{
			Name:              "HTTP 404 Not Found",
			HTTPStatusCode:    404,
			ExpectedErrorCode: errors.CodeFileNotFound,
			ExpectedRetryable: false,
		},
		{
			Name:              "HTTP 500 Server Error",
			HTTPStatusCode:    500,
			ExpectedErrorCode: errors.CodeServerError,
			ExpectedRetryable: true,
		},
		{
			Name:              "HTTP 403 Forbidden",
			HTTPStatusCode:    403,
			ExpectedErrorCode: errors.CodeAuthenticationFailed,
			ExpectedRetryable: false,
		},
		{
			Name:              "Disk Space Exhaustion",
			DiskSpaceLimit:    1024, // 1KB limit
			ExpectedErrorCode: errors.CodeInsufficientSpace,
			ExpectedRetryable: false,
		},
		{
			Name:              "Request Timeout",
			TimeoutAfter:      5 * time.Second,
			ExpectedErrorCode: errors.CodeTimeout,
			ExpectedRetryable: true,
		},
		{
			Name:              "Partial Download Failure",
			PartialFailureAt:  5000, // Fail after 5KB
			ExpectedErrorCode: errors.CodeNetworkError,
			ExpectedRetryable: true,
		},
	}
}

// SimulatorSuite combines all simulators for comprehensive testing.
type SimulatorSuite struct {
	ErrorSim     *ErrorSimulator
	NetworkSim   *NetworkFailureSimulator
	DiskSpaceSim *DiskSpaceSimulator
	HTTPSim      *HTTPErrorSimulator
}

// NewSimulatorSuite creates a complete simulator suite.
func NewSimulatorSuite() *SimulatorSuite {
	return &SimulatorSuite{
		ErrorSim:     NewErrorSimulator(DefaultSimulationConfig()),
		NetworkSim:   NewNetworkFailureSimulator(),
		DiskSpaceSim: NewDiskSpaceSimulator(1024 * 1024), // 1MB default limit
		HTTPSim:      NewHTTPErrorSimulator(),
	}
}

// Close shuts down all simulators in the suite.
func (ss *SimulatorSuite) Close() {
	if ss.ErrorSim != nil {
		ss.ErrorSim.Close()
	}

	if ss.HTTPSim != nil {
		ss.HTTPSim.Close()
	}
}
