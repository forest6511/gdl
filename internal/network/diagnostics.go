// Package network provides network diagnostics and health checking capabilities.
package network

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

// DiagnosticResult represents the result of a network diagnostic test.
type DiagnosticResult struct {
	TestName    string                 `json:"test_name"`
	Success     bool                   `json:"success"`
	Duration    time.Duration          `json:"duration"`
	Error       error                  `json:"error,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
	Timestamp   time.Time              `json:"timestamp"`
	Suggestions []string               `json:"suggestions,omitempty"`
}

// NetworkHealth represents overall network health status.
type NetworkHealth struct {
	OverallStatus HealthStatus       `json:"overall_status"`
	DNSHealth     *DiagnosticResult  `json:"dns_health"`
	ConnHealth    *DiagnosticResult  `json:"connectivity_health"`
	BandwidthInfo *BandwidthResult   `json:"bandwidth_info"`
	ProxyInfo     *ProxyInfo         `json:"proxy_info"`
	Results       []DiagnosticResult `json:"results"`
	TestDuration  time.Duration      `json:"test_duration"`
}

// HealthStatus represents the health status.
const (
	unknownValue = "unknown"
)

type HealthStatus int

const (
	// HealthGood indicates good network health.
	HealthGood HealthStatus = iota

	// HealthWarning indicates some network issues.
	HealthWarning

	// HealthPoor indicates significant network problems.
	HealthPoor

	// HealthCritical indicates critical network issues.
	HealthCritical
)

// String returns the string representation of HealthStatus.
func (hs HealthStatus) String() string {
	switch hs {
	case HealthGood:
		return "good"
	case HealthWarning:
		return "warning"
	case HealthPoor:
		return "poor"
	case HealthCritical:
		return "critical"
	default:
		return unknownValue
	}
}

// BandwidthResult represents bandwidth test results.
type BandwidthResult struct {
	DownloadSpeed  float64       `json:"download_speed_mbps"`
	UploadSpeed    float64       `json:"upload_speed_mbps"`
	Latency        time.Duration `json:"latency"`
	PacketLoss     float64       `json:"packet_loss_percent"`
	TestDuration   time.Duration `json:"test_duration"`
	TestServerHost string        `json:"test_server_host"`
	RecommendedUse string        `json:"recommended_use"`
}

// ProxyInfo represents proxy configuration information.
type ProxyInfo struct {
	Detected       bool              `json:"detected"`
	Type           ProxyType         `json:"type"`
	Address        string            `json:"address"`
	Port           int               `json:"port"`
	Authentication bool              `json:"authentication_required"`
	Environment    map[string]string `json:"environment_variables"`
	SystemProxy    bool              `json:"system_proxy"`
	Working        bool              `json:"working"`
	Details        map[string]string `json:"details"`
}

// ProxyType represents the type of proxy.
type ProxyType int

const (
	// ProxyHTTP represents HTTP proxy.
	ProxyHTTP ProxyType = iota

	// ProxyHTTPS represents HTTPS proxy.
	ProxyHTTPS

	// ProxySOCKS4 represents SOCKS4 proxy.
	ProxySOCKS4

	// ProxySOCKS5 represents SOCKS5 proxy.
	ProxySOCKS5

	// ProxyUnknown represents unknown proxy type.
	ProxyUnknown
)

// String returns the string representation of ProxyType.
func (pt ProxyType) String() string {
	switch pt {
	case ProxyHTTP:
		return "HTTP"
	case ProxyHTTPS:
		return "HTTPS"
	case ProxySOCKS4:
		return "SOCKS4"
	case ProxySOCKS5:
		return "SOCKS5"
	default:
		return "Unknown"
	}
}

// Diagnostics provides network diagnostic capabilities.
type Diagnostics struct {
	client           *http.Client
	timeout          time.Duration
	testHosts        []string
	dnsServers       []string
	bandwidthServers []string
	verbose          bool
}

// DiagnosticOptions configures diagnostic behavior.
type DiagnosticOptions struct {
	Timeout          time.Duration
	TestHosts        []string
	DNSServers       []string
	BandwidthServers []string
	IncludeBandwidth bool
	IncludeProxy     bool
	Verbose          bool
}

// Default constants.
const (
	// DefaultTimeout for diagnostic tests.
	DefaultTimeout = 30 * time.Second

	// DefaultDNSTimeout for DNS resolution.
	DefaultDNSTimeout = 5 * time.Second

	// DefaultConnTimeout for connectivity tests.
	DefaultConnTimeout = 10 * time.Second

	// BandwidthTestDuration for bandwidth tests.
	BandwidthTestDuration = 10 * time.Second

	// BandwidthTestSize for bandwidth tests (1MB).
	BandwidthTestSize = 1024 * 1024
)

// NewDiagnostics creates a new network diagnostics instance.
func NewDiagnostics() *Diagnostics {
	return &Diagnostics{
		client: &http.Client{
			Timeout: DefaultTimeout,
			Transport: &http.Transport{
				DialContext: (&net.Dialer{
					Timeout:   DefaultConnTimeout,
					KeepAlive: 30 * time.Second,
				}).DialContext,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		},
		timeout:          DefaultTimeout,
		testHosts:        getDefaultTestHosts(),
		dnsServers:       getDefaultDNSServers(),
		bandwidthServers: getDefaultBandwidthServers(),
		verbose:          false,
	}
}

// WithTimeout sets the diagnostic timeout.
func (d *Diagnostics) WithTimeout(timeout time.Duration) *Diagnostics {
	d.timeout = timeout
	d.client.Timeout = timeout

	return d
}

// WithTestHosts sets the hosts to test connectivity against.
func (d *Diagnostics) WithTestHosts(hosts []string) *Diagnostics {
	d.testHosts = hosts
	return d
}

// WithDNSServers sets the DNS servers to test.
func (d *Diagnostics) WithDNSServers(servers []string) *Diagnostics {
	d.dnsServers = servers
	return d
}

// WithVerbose enables verbose output.
func (d *Diagnostics) WithVerbose(verbose bool) *Diagnostics {
	d.verbose = verbose
	return d
}

// RunFullDiagnostics performs a comprehensive network health check.
func (d *Diagnostics) RunFullDiagnostics(
	ctx context.Context,
	options *DiagnosticOptions,
) (*NetworkHealth, error) {
	startTime := time.Now()

	if options == nil {
		options = &DiagnosticOptions{
			Timeout:          d.timeout,
			TestHosts:        d.testHosts,
			DNSServers:       d.dnsServers,
			BandwidthServers: d.bandwidthServers,
			IncludeBandwidth: true,
			IncludeProxy:     true,
			Verbose:          d.verbose,
		}
	}

	health := &NetworkHealth{
		Results: []DiagnosticResult{},
	}

	// Run DNS diagnostics
	dnsResult := d.TestDNSResolution(ctx, options.DNSServers, options.TestHosts)
	health.DNSHealth = &dnsResult
	health.Results = append(health.Results, dnsResult)

	// Run connectivity tests
	connResult := d.TestConnectivity(ctx, options.TestHosts)
	health.ConnHealth = &connResult
	health.Results = append(health.Results, connResult)

	// Run proxy detection
	if options.IncludeProxy {
		proxyResult := d.DetectProxy(ctx)
		health.ProxyInfo = proxyResult.(*ProxyInfo)
		health.Results = append(health.Results, DiagnosticResult{
			TestName:  "proxy_detection",
			Success:   true,
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
			Details:   map[string]interface{}{"proxy_info": proxyResult},
		})
	}

	// Run bandwidth test
	if options.IncludeBandwidth && len(options.BandwidthServers) > 0 {
		bandwidthResult := d.TestBandwidth(ctx, options.BandwidthServers[0])
		health.BandwidthInfo = bandwidthResult.(*BandwidthResult)
		health.Results = append(health.Results, DiagnosticResult{
			TestName:  "bandwidth_test",
			Success:   true, // TestBandwidth never returns nil
			Duration:  time.Since(startTime),
			Timestamp: time.Now(),
			Details:   map[string]interface{}{"bandwidth_info": bandwidthResult},
		})
	}

	// Calculate overall health status
	health.OverallStatus = d.calculateOverallHealth(health)
	health.TestDuration = time.Since(startTime)

	return health, nil
}

// TestDNSResolution tests DNS resolution for multiple hosts and DNS servers.
func (d *Diagnostics) TestDNSResolution(
	ctx context.Context,
	dnsServers, testHosts []string,
) DiagnosticResult {
	startTime := time.Now()
	result := DiagnosticResult{
		TestName:  "dns_resolution",
		Timestamp: startTime,
		Details:   make(map[string]interface{}),
	}

	var (
		successful int
		failed     int
		totalTime  time.Duration
	)

	resolutionResults := make(map[string]interface{})

	for _, host := range testHosts {
		hostResults := make(map[string]interface{})

		for _, dnsServer := range dnsServers {
			resolver := &net.Resolver{
				PreferGo: true,
				Dial: func(ctx context.Context, network, address string) (net.Conn, error) {
					d := net.Dialer{Timeout: DefaultDNSTimeout}
					return d.DialContext(ctx, network, dnsServer+":53")
				},
			}

			lookupStart := time.Now()
			ips, err := resolver.LookupIPAddr(ctx, host)
			lookupDuration := time.Since(lookupStart)
			totalTime += lookupDuration

			if err != nil {
				failed++
				hostResults[dnsServer] = map[string]interface{}{
					"success":  false,
					"error":    err.Error(),
					"duration": lookupDuration,
				}
			} else {
				successful++

				ipStrings := make([]string, len(ips))
				for i, ip := range ips {
					ipStrings[i] = ip.IP.String()
				}

				hostResults[dnsServer] = map[string]interface{}{
					"success":  true,
					"ips":      ipStrings,
					"duration": lookupDuration,
				}
			}
		}

		resolutionResults[host] = hostResults
	}

	result.Success = successful > failed
	result.Duration = time.Since(startTime)
	result.Details["successful_lookups"] = successful

	result.Details["failed_lookups"] = failed
	if successful+failed > 0 {
		result.Details["average_lookup_time"] = totalTime / time.Duration(successful+failed)
	} else {
		result.Details["average_lookup_time"] = time.Duration(0)
	}

	result.Details["resolution_results"] = resolutionResults

	if !result.Success {
		result.Error = fmt.Errorf(
			"DNS resolution failed for %d out of %d tests",
			failed,
			successful+failed,
		)
		result.Suggestions = []string{
			"Check your DNS server configuration",
			"Try using a different DNS server (8.8.8.8, 1.1.1.1)",
			"Check if your firewall is blocking DNS requests",
			"Verify your network connection",
		}
	}

	return result
}

// TestConnectivity tests basic connectivity to multiple hosts.
func (d *Diagnostics) TestConnectivity(ctx context.Context, testHosts []string) DiagnosticResult {
	startTime := time.Now()
	result := DiagnosticResult{
		TestName:  "connectivity_test",
		Timestamp: startTime,
		Details:   make(map[string]interface{}),
	}

	var (
		successful int
		failed     int
	)

	connectivityResults := make(map[string]interface{})

	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	for _, host := range testHosts {
		wg.Add(1)

		go func(testHost string) {
			defer wg.Done()

			// Test HTTP connectivity
			httpResult := d.testHTTPConnectivity(ctx, "http://"+testHost)

			// Test HTTPS connectivity
			httpsResult := d.testHTTPConnectivity(ctx, "https://"+testHost)

			mu.Lock()
			defer mu.Unlock()

			hostResult := map[string]interface{}{
				"http":  httpResult,
				"https": httpsResult,
			}

			if httpResult["success"].(bool) || httpsResult["success"].(bool) {
				successful++
			} else {
				failed++
			}

			connectivityResults[testHost] = hostResult
		}(host)
	}

	wg.Wait()

	result.Success = successful > 0
	result.Duration = time.Since(startTime)
	result.Details["successful_connections"] = successful
	result.Details["failed_connections"] = failed
	result.Details["connectivity_results"] = connectivityResults

	if !result.Success {
		result.Error = fmt.Errorf("connectivity test failed for all %d hosts", len(testHosts))
		result.Suggestions = []string{
			"Check your internet connection",
			"Verify firewall settings",
			"Check if a proxy is required",
			"Try connecting to different hosts",
		}
	}

	return result
}

// testHTTPConnectivity tests connectivity to a specific HTTP/HTTPS URL.
func (d *Diagnostics) testHTTPConnectivity(
	ctx context.Context,
	testURL string,
) map[string]interface{} {
	startTime := time.Now()

	req, err := http.NewRequestWithContext(ctx, "HEAD", testURL, nil)
	if err != nil {
		return map[string]interface{}{
			"success":  false,
			"error":    err.Error(),
			"duration": time.Since(startTime),
		}
	}

	resp, err := d.client.Do(req)
	duration := time.Since(startTime)

	if err != nil {
		return map[string]interface{}{
			"success":  false,
			"error":    err.Error(),
			"duration": duration,
		}
	}

	defer func() { _ = resp.Body.Close() }()

	return map[string]interface{}{
		"success":     resp.StatusCode < 400,
		"status_code": resp.StatusCode,
		"duration":    duration,
		"headers":     resp.Header,
	}
}

// DetectProxy detects proxy configuration and validates it.
func (d *Diagnostics) DetectProxy(ctx context.Context) interface{} {
	proxyInfo := &ProxyInfo{
		Environment: make(map[string]string),
		Details:     make(map[string]string),
	}

	// Check environment variables
	envVars := []string{
		"HTTP_PROXY",
		"HTTPS_PROXY",
		"http_proxy",
		"https_proxy",
		"ALL_PROXY",
		"all_proxy",
	}
	for _, envVar := range envVars {
		if value := os.Getenv(envVar); value != "" {
			proxyInfo.Environment[envVar] = value
			proxyInfo.Detected = true
		}
	}

	// Parse proxy URL if found
	if proxyInfo.Detected {
		for _, envVar := range envVars {
			if proxyURL := os.Getenv(envVar); proxyURL != "" {
				if u, err := url.Parse(proxyURL); err == nil {
					proxyInfo.Address = u.Hostname()
					if port := u.Port(); port != "" {
						if p, err := strconv.Atoi(port); err == nil {
							proxyInfo.Port = p
						}
					}

					proxyInfo.Authentication = u.User != nil

					switch u.Scheme {
					case "http":
						proxyInfo.Type = ProxyHTTP
					case "https":
						proxyInfo.Type = ProxyHTTPS
					case "socks4":
						proxyInfo.Type = ProxySOCKS4
					case "socks5":
						proxyInfo.Type = ProxySOCKS5
					default:
						proxyInfo.Type = ProxyUnknown
					}

					break
				}
			}
		}
	}

	// Test proxy functionality if detected
	if proxyInfo.Detected {
		proxyInfo.Working = d.testProxyConnectivity(ctx, proxyInfo)
	}

	return proxyInfo
}

// testProxyConnectivity tests if the proxy is working.
func (d *Diagnostics) testProxyConnectivity(ctx context.Context, proxyInfo *ProxyInfo) bool {
	// Create a client configured with the detected proxy
	proxyURL := fmt.Sprintf(
		"%s://%s:%d",
		proxyInfo.Type.String(),
		proxyInfo.Address,
		proxyInfo.Port,
	)

	proxy, err := url.Parse(strings.ToLower(proxyURL))
	if err != nil {
		return false
	}

	client := &http.Client{
		Transport: &http.Transport{
			Proxy: http.ProxyURL(proxy),
		},
		Timeout: 10 * time.Second,
	}

	// Test connectivity through proxy
	req, err := http.NewRequestWithContext(ctx, "HEAD", "http://httpbin.org/status/200", nil)
	if err != nil {
		return false
	}

	resp, err := client.Do(req)
	if err != nil {
		return false
	}
	defer func() { _ = resp.Body.Close() }()

	return resp.StatusCode == 200
}

// TestBandwidth performs a bandwidth test.
func (d *Diagnostics) TestBandwidth(ctx context.Context, testServer string) interface{} {
	startTime := time.Now()

	result := &BandwidthResult{
		TestServerHost: testServer,
		TestDuration:   0,
	}

	// Test download speed
	downloadSpeed, latency := d.testDownloadSpeed(ctx, testServer)
	result.DownloadSpeed = downloadSpeed
	result.Latency = latency

	// For now, we'll skip upload test as it's more complex to implement properly
	result.UploadSpeed = 0
	result.PacketLoss = 0

	result.TestDuration = time.Since(startTime)

	// Provide recommendations based on speed
	if downloadSpeed > 25 {
		result.RecommendedUse = "Excellent for large file downloads"
	} else if downloadSpeed > 10 {
		result.RecommendedUse = "Good for medium file downloads"
	} else if downloadSpeed > 1 {
		result.RecommendedUse = "Suitable for small files only"
	} else {
		result.RecommendedUse = "Very slow connection, consider alternative"
	}

	return result
}

// testDownloadSpeed performs a download speed test.
func (d *Diagnostics) testDownloadSpeed(
	ctx context.Context,
	testServer string,
) (float64, time.Duration) {
	testURL := fmt.Sprintf("http://%s/data", testServer)

	// Ping test for latency
	pingStart := time.Now()

	req, err := http.NewRequestWithContext(ctx, "HEAD", testURL, nil)
	if err != nil {
		return 0, 0
	}

	resp, err := d.client.Do(req)
	if err != nil {
		return 0, 0
	}
	defer func() { _ = resp.Body.Close() }()

	latency := time.Since(pingStart)

	// Download test
	downloadStart := time.Now()

	resp, err = d.client.Get(testURL)
	if err != nil {
		return 0, latency
	}
	defer func() { _ = resp.Body.Close() }()

	// Read response body to measure actual download speed
	bytesRead := int64(0)
	buffer := make([]byte, 32*1024) // 32KB buffer

	for {
		n, err := resp.Body.Read(buffer)
		bytesRead += int64(n)

		if err == io.EOF {
			break
		}

		if err != nil {
			break
		}

		// Stop after a reasonable amount of data or time
		if bytesRead > BandwidthTestSize || time.Since(downloadStart) > BandwidthTestDuration {
			break
		}
	}

	downloadDuration := time.Since(downloadStart)
	if downloadDuration == 0 {
		return 0, latency
	}

	// Calculate speed in Mbps
	speedMbps := float64(bytesRead) * 8 / 1000000 / downloadDuration.Seconds()

	return speedMbps, latency
}

// calculateOverallHealth determines overall network health status.
func (d *Diagnostics) calculateOverallHealth(health *NetworkHealth) HealthStatus {
	criticalIssues := 0
	warnings := 0

	// Check DNS health
	if health.DNSHealth != nil && !health.DNSHealth.Success {
		criticalIssues++
	}

	// Check connectivity
	if health.ConnHealth != nil && !health.ConnHealth.Success {
		criticalIssues++
	}

	// Check bandwidth
	if health.BandwidthInfo != nil {
		if health.BandwidthInfo.DownloadSpeed < 1 {
			criticalIssues++
		} else if health.BandwidthInfo.DownloadSpeed < 5 {
			warnings++
		}

		if health.BandwidthInfo.Latency > 1*time.Second {
			warnings++
		}
	}

	// Check proxy issues
	if health.ProxyInfo != nil && health.ProxyInfo.Detected && !health.ProxyInfo.Working {
		warnings++
	}

	// Determine overall status
	if criticalIssues > 1 {
		return HealthCritical
	} else if criticalIssues == 1 {
		return HealthPoor
	} else if warnings > 1 {
		return HealthWarning
	} else {
		return HealthGood
	}
}

// Helper functions

// getDefaultTestHosts returns default hosts for testing.
func getDefaultTestHosts() []string {
	return []string{
		"google.com",
		"cloudflare.com",
		"github.com",
		"httpbin.org",
	}
}

// getDefaultDNSServers returns default DNS servers for testing.
func getDefaultDNSServers() []string {
	return []string{
		"8.8.8.8",        // Google DNS
		"1.1.1.1",        // Cloudflare DNS
		"208.67.222.222", // OpenDNS
	}
}

// getDefaultBandwidthServers returns default bandwidth test servers.
func getDefaultBandwidthServers() []string {
	return []string{
		"httpbin.org",
		"jsonplaceholder.typicode.com",
	}
}
