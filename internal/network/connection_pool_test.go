package network

import (
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestConnectionPool_GetClient(t *testing.T) {
	pool := NewConnectionPool(10, 20)
	defer pool.Close()

	// Test getting client for same host returns same instance
	host1 := "example.com"
	client1 := pool.GetClient(host1, 30*time.Second)
	client2 := pool.GetClient(host1, 30*time.Second)

	if client1 != client2 {
		t.Error("Expected same client instance for same host")
	}

	// Test getting client for different host returns different instance
	host2 := "another.com"
	client3 := pool.GetClient(host2, 30*time.Second)

	if client1 == client3 {
		t.Error("Expected different client instances for different hosts")
	}

	// Check stats
	stats := pool.Stats()
	if stats["num_clients"].(int) != 2 {
		t.Errorf("Expected 2 clients in pool, got %d", stats["num_clients"])
	}
}

func TestConnectionPool_Concurrent(t *testing.T) {
	pool := NewConnectionPool(10, 20)
	defer pool.Close()

	var wg sync.WaitGroup
	numGoroutines := 100
	host := "concurrent.com"

	// Concurrent access to same host
	for i := 0; i < numGoroutines; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			client := pool.GetClient(host, 30*time.Second)
			if client == nil {
				t.Error("Got nil client")
			}
		}()
	}

	wg.Wait()

	// Should have only one client for the host
	stats := pool.Stats()
	if stats["num_clients"].(int) != 1 {
		t.Errorf("Expected 1 client in pool, got %d", stats["num_clients"])
	}
}

func TestConnectionPool_RealRequests(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OK"))
	}))
	defer server.Close()

	pool := NewConnectionPool(5, 10)
	defer pool.Close()

	// Use pooled client for multiple requests
	client := pool.GetClient(server.URL, 5*time.Second)

	for i := 0; i < 10; i++ {
		resp, err := client.Get(server.URL)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i, err)
		}
		_ = resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Request %d: expected status 200, got %d", i, resp.StatusCode)
		}
	}
}

func BenchmarkConnectionPool(b *testing.B) {
	pool := NewConnectionPool(10, 20)
	defer pool.Close()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		hosts := []string{"host1.com", "host2.com", "host3.com", "host4.com", "host5.com"}
		for pb.Next() {
			host := hosts[i%len(hosts)]
			pool.GetClient(host, 30*time.Second)
			i++
		}
	})
}

func BenchmarkConnectionPoolVsNew(b *testing.B) {
	// Benchmark with connection pool
	b.Run("WithPool", func(b *testing.B) {
		pool := NewConnectionPool(10, 20)
		defer pool.Close()

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("OK"))
		}))
		defer server.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			client := pool.GetClient(server.URL, 30*time.Second)
			resp, _ := client.Get(server.URL)
			if resp != nil {
				_ = resp.Body.Close()
			}
		}
	})

	// Benchmark without connection pool
	b.Run("WithoutPool", func(b *testing.B) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			_, _ = w.Write([]byte("OK"))
		}))
		defer server.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			client := &http.Client{
				Timeout: 30 * time.Second,
			}
			resp, _ := client.Get(server.URL)
			if resp != nil {
				_ = resp.Body.Close()
			}
		}
	})
}
