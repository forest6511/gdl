package network

import (
	"context"
	"crypto/tls"
	"net"
	"net/http"
	"sync"
	"time"
)

// DNSCache caches DNS resolutions to reduce lookup overhead
type DNSCache struct {
	mu    sync.RWMutex
	cache map[string]*dnsEntry
}

type dnsEntry struct {
	ips       []net.IP
	timestamp time.Time
}

// NewDNSCache creates a new DNS cache
func NewDNSCache() *DNSCache {
	return &DNSCache{
		cache: make(map[string]*dnsEntry),
	}
}

// Resolve performs a cached DNS lookup
func (dc *DNSCache) Resolve(host string) ([]net.IP, error) {
	dc.mu.RLock()
	entry, exists := dc.cache[host]
	dc.mu.RUnlock()

	// Check if cache entry exists and is fresh (less than 5 minutes old)
	if exists && time.Since(entry.timestamp) < 5*time.Minute {
		return entry.ips, nil
	}

	// Perform DNS lookup
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	// Update cache
	dc.mu.Lock()
	dc.cache[host] = &dnsEntry{
		ips:       ips,
		timestamp: time.Now(),
	}
	dc.mu.Unlock()

	return ips, nil
}

// Clear removes all entries from the DNS cache
func (dc *DNSCache) Clear() {
	dc.mu.Lock()
	dc.cache = make(map[string]*dnsEntry)
	dc.mu.Unlock()
}

// AdvancedConnectionPool extends the basic connection pool with advanced features
type AdvancedConnectionPool struct {
	*ConnectionPool
	dnsCache      *DNSCache
	tlsConfigPool map[string]*tls.Config
	tlsMu         sync.RWMutex
	prewarmList   []string
	prewarmMu     sync.RWMutex
}

// NewAdvancedConnectionPool creates an advanced connection pool
func NewAdvancedConnectionPool(maxIdle, maxConns int) *AdvancedConnectionPool {
	return &AdvancedConnectionPool{
		ConnectionPool: NewConnectionPool(maxIdle, maxConns),
		dnsCache:       NewDNSCache(),
		tlsConfigPool:  make(map[string]*tls.Config),
		prewarmList:    make([]string, 0),
	}
}

// GetClientWithDNSCache returns an HTTP client with DNS caching
func (acp *AdvancedConnectionPool) GetClientWithDNSCache(host string, timeout time.Duration) *http.Client {
	// First resolve DNS through cache
	_, _ = acp.dnsCache.Resolve(host)

	// Get or create client with custom dialer that uses cached DNS
	acp.mu.RLock()
	client, exists := acp.clients[host]
	acp.mu.RUnlock()

	if exists {
		return client
	}

	// Create new client with DNS cache-aware dialer
	acp.mu.Lock()
	defer acp.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists = acp.clients[host]; exists {
		return client
	}

	// Create custom dialer with DNS cache
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
		Resolver: &net.Resolver{
			PreferGo: true,
		},
	}

	// Create transport with custom dialer
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
			// Use cached DNS if available
			host, port, err := net.SplitHostPort(addr)
			if err != nil {
				return dialer.DialContext(ctx, network, addr)
			}

			ips, err := acp.dnsCache.Resolve(host)
			if err != nil || len(ips) == 0 {
				return dialer.DialContext(ctx, network, addr)
			}

			// Use first IP from cache
			cachedAddr := net.JoinHostPort(ips[0].String(), port)
			return dialer.DialContext(ctx, network, cachedAddr)
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          acp.maxIdle,
		MaxIdleConnsPerHost:   acp.maxIdle,
		MaxConnsPerHost:       acp.maxConns,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
	}

	client = &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	acp.clients[host] = client
	return client
}

// PrewarmConnection establishes a connection to a host in advance
func (acp *AdvancedConnectionPool) PrewarmConnection(host string) error {
	// Resolve DNS
	_, err := acp.dnsCache.Resolve(host)
	if err != nil {
		return err
	}

	// Create client for host
	_ = acp.GetClientWithDNSCache(host, 30*time.Second)

	// Add to prewarm list
	acp.prewarmMu.Lock()
	acp.prewarmList = append(acp.prewarmList, host)
	acp.prewarmMu.Unlock()

	// Perform a HEAD request to establish TLS connection
	go func() {
		client := acp.GetClientWithDNSCache(host, 5*time.Second)
		req, _ := http.NewRequest(http.MethodHead, "https://"+host, nil)
		resp, err := client.Do(req)
		if err == nil && resp != nil {
			_ = resp.Body.Close()
		}
	}()

	return nil
}

// PrewarmCDNs pre-warms connections to common CDNs
func (acp *AdvancedConnectionPool) PrewarmCDNs() {
	cdns := []string{
		"cdnjs.cloudflare.com",
		"cdn.jsdelivr.net",
		"unpkg.com",
		"cdn.skypack.dev",
		"cdn.esm.sh",
		"registry.npmjs.org",
		"github.com",
		"raw.githubusercontent.com",
	}

	for _, cdn := range cdns {
		_ = acp.PrewarmConnection(cdn)
	}
}

// GetTLSConfig returns a cached TLS configuration for a host
func (acp *AdvancedConnectionPool) GetTLSConfig(host string) *tls.Config {
	acp.tlsMu.RLock()
	config, exists := acp.tlsConfigPool[host]
	acp.tlsMu.RUnlock()

	if exists {
		return config
	}

	// Create new TLS config
	acp.tlsMu.Lock()
	defer acp.tlsMu.Unlock()

	// Double-check
	if config, exists = acp.tlsConfigPool[host]; exists {
		return config
	}

	config = &tls.Config{
		ServerName:         host,
		MinVersion:         tls.VersionTLS12,
		MaxVersion:         tls.VersionTLS13,
		InsecureSkipVerify: false,
		// Enable session resumption
		ClientSessionCache: tls.NewLRUClientSessionCache(32),
	}

	acp.tlsConfigPool[host] = config
	return config
}

// OptimizeForCDN configures the pool specifically for CDN downloads
func (acp *AdvancedConnectionPool) OptimizeForCDN(cdnHost string) {
	// Pre-warm connection
	_ = acp.PrewarmConnection(cdnHost)

	// Create optimized client for CDN
	acp.mu.Lock()
	defer acp.mu.Unlock()

	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second, // Faster timeout for CDNs
			KeepAlive: 60 * time.Second, // Longer keep-alive for CDNs
		}).DialContext,
		ForceAttemptHTTP2:     true, // CDNs typically support HTTP/2
		MaxIdleConns:          50,   // More idle connections for CDNs
		MaxIdleConnsPerHost:   50,
		MaxConnsPerHost:       50,
		IdleConnTimeout:       120 * time.Second, // Longer idle timeout
		TLSHandshakeTimeout:   5 * time.Second,   // CDNs should be fast
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    true, // CDN content usually pre-compressed
		TLSClientConfig:       acp.GetTLSConfig(cdnHost),
	}

	acp.clients[cdnHost] = &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
}

// GlobalAdvancedPool is the global advanced connection pool
var GlobalAdvancedPool = NewAdvancedConnectionPool(100, 100)
