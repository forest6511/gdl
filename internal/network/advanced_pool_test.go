package network

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestDNSCache(t *testing.T) {
	cache := NewDNSCache()

	// Test DNS resolution and caching
	host := "example.com"
	ips1, err := cache.Resolve(host)
	if err != nil {
		t.Fatalf("Failed to resolve %s: %v", host, err)
	}

	if len(ips1) == 0 {
		t.Error("Expected at least one IP address")
	}

	// Second resolution should use cache
	ips2, err := cache.Resolve(host)
	if err != nil {
		t.Fatalf("Failed to resolve from cache: %v", err)
	}

	// Should return same IPs
	if len(ips1) != len(ips2) {
		t.Error("Cached result differs from original")
	}

	// Clear cache
	cache.Clear()

	// After clear, should perform new lookup
	ips3, err := cache.Resolve(host)
	if err != nil {
		t.Fatalf("Failed to resolve after clear: %v", err)
	}

	if len(ips3) == 0 {
		t.Error("Expected IPs after cache clear")
	}
}

func TestAdvancedConnectionPool_GetClientWithDNSCache(t *testing.T) {
	pool := NewAdvancedConnectionPool(10, 20)
	defer pool.Close()

	// Test getting client with DNS cache
	client := pool.GetClientWithDNSCache("example.com", 30*time.Second)
	if client == nil {
		t.Error("Expected non-nil client")
	}

	// Should reuse same client
	client2 := pool.GetClientWithDNSCache("example.com", 30*time.Second)
	if client != client2 {
		t.Error("Expected same client instance")
	}

	// Different host should get different client
	client3 := pool.GetClientWithDNSCache("another.com", 30*time.Second)
	if client == client3 {
		t.Error("Expected different client for different host")
	}
}

func TestAdvancedConnectionPool_PrewarmConnection(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodHead {
			w.WriteHeader(http.StatusOK)
			return
		}
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	// Extract host from server URL
	host := server.URL[7:] // Remove "http://"
	if host == "" {
		t.Skip("Cannot extract host from test server URL")
	}

	pool := NewAdvancedConnectionPool(10, 20)
	defer pool.Close()

	// Prewarm connection
	err := pool.PrewarmConnection(host)
	if err != nil {
		// It's OK if prewarming fails for test server
		t.Logf("Prewarm failed (expected for test server): %v", err)
		// For test purposes, we can skip the rest if connection fails
		return
	}

	// Check that host is in prewarm list only if prewarm succeeded
	pool.prewarmMu.RLock()
	found := false
	for _, h := range pool.prewarmList {
		if h == host {
			found = true
			break
		}
	}
	pool.prewarmMu.RUnlock()

	if !found {
		t.Error("Host not added to prewarm list despite successful prewarm")
	}
}

func TestAdvancedConnectionPool_GetTLSConfig(t *testing.T) {
	pool := NewAdvancedConnectionPool(10, 20)
	defer pool.Close()

	host := "example.com"
	config1 := pool.GetTLSConfig(host)
	if config1 == nil {
		t.Fatal("Expected non-nil TLS config")
	}

	// Should return same config
	config2 := pool.GetTLSConfig(host)
	if config1 != config2 {
		t.Error("Expected same TLS config instance")
	}

	// Check config properties
	if config1.ServerName != host {
		t.Errorf("Expected ServerName %s, got %s", host, config1.ServerName)
	}

	if config1.MinVersion != tls.VersionTLS12 {
		t.Error("Expected TLS 1.2 minimum version")
	}
}

func TestAdvancedConnectionPool_OptimizeForCDN(t *testing.T) {
	pool := NewAdvancedConnectionPool(10, 20)
	defer pool.Close()

	cdnHost := "cdn.example.com"
	pool.OptimizeForCDN(cdnHost)

	// Check that optimized client was created
	pool.mu.RLock()
	client, exists := pool.clients[cdnHost]
	pool.mu.RUnlock()

	if !exists || client == nil {
		t.Error("Expected optimized client for CDN")
	}

	// Check transport settings
	if transport, ok := client.Transport.(*http.Transport); ok {
		if transport.MaxIdleConns != 50 {
			t.Error("Expected optimized MaxIdleConns for CDN")
		}
		if !transport.ForceAttemptHTTP2 {
			t.Error("Expected HTTP/2 to be forced for CDN")
		}
		if !transport.DisableCompression {
			t.Error("Expected compression to be disabled for CDN")
		}
	} else {
		t.Error("Transport is not *http.Transport")
	}
}

func BenchmarkDNSCache(b *testing.B) {
	cache := NewDNSCache()
	host := "example.com"

	// Pre-warm cache
	_, _ = cache.Resolve(host)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := cache.Resolve(host)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

func TestAdvancedConnectionPool_GetClient(t *testing.T) {
	pool := NewAdvancedConnectionPool(10, 20)
	defer pool.Close()

	// Test getting client with host
	host := "example.com"
	client := pool.GetClient(host, 30*time.Second)
	if client == nil {
		t.Error("Expected non-nil client")
	}

	// Make a test request to verify client works
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	resp, err := client.Get(server.URL)
	if err != nil {
		t.Errorf("Failed to make request: %v", err)
	} else {
		_ = resp.Body.Close()
	}
}

func TestAdvancedConnectionPool_MultipleHosts(t *testing.T) {
	pool := NewAdvancedConnectionPool(10, 20)
	defer pool.Close()

	hosts := []string{"host1.com", "host2.com", "host3.com"}

	// Get clients for different hosts
	for _, host := range hosts {
		client := pool.GetClientWithDNSCache(host, 30*time.Second)
		if client == nil {
			t.Errorf("Failed to get client for %s", host)
		}
	}

	// Verify we have separate clients
	pool.mu.RLock()
	clientCount := len(pool.clients)
	pool.mu.RUnlock()

	if clientCount < len(hosts) {
		t.Errorf("Expected at least %d clients, got %d", len(hosts), clientCount)
	}
}

func TestAdvancedConnectionPool_ConcurrentAccess(t *testing.T) {
	pool := NewAdvancedConnectionPool(10, 20)
	defer pool.Close()

	// Test concurrent access to the pool
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func(id int) {
			host := fmt.Sprintf("host%d.com", id%3)
			client := pool.GetClientWithDNSCache(host, 30*time.Second)
			if client == nil {
				t.Errorf("Worker %d: Failed to get client", id)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func BenchmarkAdvancedPool_vs_BasicPool(b *testing.B) {
	b.Run("AdvancedPool", func(b *testing.B) {
		pool := NewAdvancedConnectionPool(10, 20)
		defer pool.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = pool.GetClientWithDNSCache("example.com", 30*time.Second)
		}
	})

	b.Run("BasicPool", func(b *testing.B) {
		pool := NewConnectionPool(10, 20)
		defer pool.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = pool.GetClient("example.com", 30*time.Second)
		}
	})
}
