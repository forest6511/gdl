package network

import (
	"net"
	"net/http"
	"sync"
	"time"
)

// ConnectionPool manages a pool of HTTP clients for connection reuse
type ConnectionPool struct {
	mu       sync.RWMutex
	clients  map[string]*http.Client
	maxIdle  int
	maxConns int
}

// NewConnectionPool creates a new connection pool
func NewConnectionPool(maxIdle, maxConns int) *ConnectionPool {
	return &ConnectionPool{
		clients:  make(map[string]*http.Client),
		maxIdle:  maxIdle,
		maxConns: maxConns,
	}
}

// GetClient returns an HTTP client for the given host
func (cp *ConnectionPool) GetClient(host string, timeout time.Duration) *http.Client {
	cp.mu.RLock()
	client, exists := cp.clients[host]
	cp.mu.RUnlock()

	if exists {
		return client
	}

	// Create new client with optimized transport
	cp.mu.Lock()
	defer cp.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists = cp.clients[host]; exists {
		return client
	}

	// Create optimized transport for this host
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          cp.maxIdle,
		MaxIdleConnsPerHost:   cp.maxIdle,
		MaxConnsPerHost:       cp.maxConns,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
	}

	client = &http.Client{
		Transport: transport,
		Timeout:   timeout,
	}

	cp.clients[host] = client
	return client
}

// Close closes all clients in the pool
func (cp *ConnectionPool) Close() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	for _, client := range cp.clients {
		if transport, ok := client.Transport.(*http.Transport); ok {
			transport.CloseIdleConnections()
		}
	}

	cp.clients = make(map[string]*http.Client)
}

// Stats returns pool statistics
func (cp *ConnectionPool) Stats() map[string]interface{} {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	stats := make(map[string]interface{})
	stats["num_clients"] = len(cp.clients)
	stats["max_idle"] = cp.maxIdle
	stats["max_conns"] = cp.maxConns

	hosts := make([]string, 0, len(cp.clients))
	for host := range cp.clients {
		hosts = append(hosts, host)
	}
	stats["hosts"] = hosts

	return stats
}

// GlobalPool is a shared connection pool for the application
var GlobalPool = NewConnectionPool(100, 100)
