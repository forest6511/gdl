package network

import (
	"net/http"
	"testing"
	"time"
)

func TestCreateOptimizedTransport(t *testing.T) {
	transport := CreateOptimizedTransport()

	// Verify connection pooling settings
	if transport.MaxIdleConns != 100 {
		t.Errorf("MaxIdleConns = %d, want 100", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 20 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 20", transport.MaxIdleConnsPerHost)
	}

	if transport.MaxConnsPerHost != 30 {
		t.Errorf("MaxConnsPerHost = %d, want 30", transport.MaxConnsPerHost)
	}

	// Verify timeout settings
	if transport.IdleConnTimeout != 90*time.Second {
		t.Errorf("IdleConnTimeout = %v, want 90s", transport.IdleConnTimeout)
	}

	if transport.TLSHandshakeTimeout != 10*time.Second {
		t.Errorf("TLSHandshakeTimeout = %v, want 10s", transport.TLSHandshakeTimeout)
	}

	// Verify optimization settings
	if transport.DisableKeepAlives != false {
		t.Error("DisableKeepAlives should be false for connection reuse")
	}

	if transport.ForceAttemptHTTP2 != true {
		t.Error("ForceAttemptHTTP2 should be true for better performance")
	}

	// Verify buffer sizes
	if transport.WriteBufferSize != 64*1024 {
		t.Errorf("WriteBufferSize = %d, want %d", transport.WriteBufferSize, 64*1024)
	}

	if transport.ReadBufferSize != 64*1024 {
		t.Errorf("ReadBufferSize = %d, want %d", transport.ReadBufferSize, 64*1024)
	}
}

func TestCreateOptimizedClient(t *testing.T) {
	tests := []struct {
		name            string
		timeout         time.Duration
		expectedTimeout time.Duration
	}{
		{
			name:            "default timeout",
			timeout:         0,
			expectedTimeout: 30 * time.Minute,
		},
		{
			name:            "custom timeout",
			timeout:         10 * time.Minute,
			expectedTimeout: 10 * time.Minute,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := CreateOptimizedClient(tt.timeout)

			if client.Timeout != tt.expectedTimeout {
				t.Errorf("Client timeout = %v, want %v", client.Timeout, tt.expectedTimeout)
			}

			// Verify transport is set
			if client.Transport == nil {
				t.Error("Client transport should not be nil")
			}

			// Verify it's our optimized transport
			transport, ok := client.Transport.(*http.Transport)
			if !ok {
				t.Error("Client transport should be *http.Transport")
			}

			if transport.MaxIdleConnsPerHost != 20 {
				t.Error("Client should use optimized transport settings")
			}
		})
	}
}

func TestCreateLightweightClient(t *testing.T) {
	client := CreateLightweightClient()

	// Verify timeout
	if client.Timeout != 5*time.Minute {
		t.Errorf("Lightweight client timeout = %v, want 5 minutes", client.Timeout)
	}

	// Verify transport settings
	transport, ok := client.Transport.(*http.Transport)
	if !ok {
		t.Fatal("Client transport should be *http.Transport")
	}

	// Verify reduced connection pool
	if transport.MaxIdleConns != 10 {
		t.Errorf("MaxIdleConns = %d, want 10", transport.MaxIdleConns)
	}

	if transport.MaxIdleConnsPerHost != 2 {
		t.Errorf("MaxIdleConnsPerHost = %d, want 2", transport.MaxIdleConnsPerHost)
	}

	// Verify smaller buffers
	if transport.WriteBufferSize != 16*1024 {
		t.Errorf("WriteBufferSize = %d, want %d", transport.WriteBufferSize, 16*1024)
	}

	if transport.ReadBufferSize != 16*1024 {
		t.Errorf("ReadBufferSize = %d, want %d", transport.ReadBufferSize, 16*1024)
	}
}

// Benchmark tests to verify performance impact
func BenchmarkOptimizedTransport(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = CreateOptimizedTransport()
	}
}

func BenchmarkLightweightTransport(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = CreateLightweightClient()
	}
}
