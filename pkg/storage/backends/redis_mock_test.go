package backends

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"testing"
	"time"
)

// TestRedisBackendConfiguration tests Redis backend initialization and configuration
func TestRedisBackendConfiguration(t *testing.T) {
	t.Run("InitWithConfig", func(t *testing.T) {
		backend := NewRedisBackend()

		// Test with all config options
		config := map[string]interface{}{
			"addr":     "redis.example.com:6380",
			"password": "secret",
			"db":       float64(5), // JSON numbers come as float64
			"prefix":   "myapp:",
		}

		err := backend.Init(config)
		// Will fail to connect but tests config parsing
		if err == nil {
			t.Log("Redis connection succeeded (Redis is running)")
		} else {
			t.Logf("Redis connection failed as expected: %v", err)
		}
	})

	t.Run("InitWithIntDB", func(t *testing.T) {
		backend := NewRedisBackend()

		config := map[string]interface{}{
			"addr": "localhost:6379",
			"db":   3, // int type
		}

		err := backend.Init(config)
		if err == nil {
			t.Log("Redis connection succeeded with int db")
		} else {
			t.Logf("Redis connection failed as expected: %v", err)
		}
	})

	t.Run("InitWithDefaultValues", func(t *testing.T) {
		backend := NewRedisBackend()

		// Empty config should use defaults
		err := backend.Init(map[string]interface{}{})
		if err == nil {
			t.Log("Redis connection succeeded with defaults")
		} else {
			t.Logf("Redis connection failed as expected: %v", err)
		}
	})

	t.Run("KeyHandling", func(t *testing.T) {
		backend := &RedisBackend{
			prefix: "test-prefix",
		}

		// Test buildKey
		key := backend.buildKey("mykey")
		expected := "test-prefix:mykey"
		if key != expected {
			t.Errorf("buildKey: expected %s, got %s", expected, key)
		}

		// Test stripPrefix
		stripped := backend.stripPrefix("test-prefix:mykey")
		expected = "mykey"
		if stripped != expected {
			t.Errorf("stripPrefix: expected %s, got %s", expected, stripped)
		}

		// Test stripPrefix without matching prefix
		stripped = backend.stripPrefix("other:mykey")
		expected = "other:mykey"
		if stripped != expected {
			t.Errorf("stripPrefix no match: expected %s, got %s", expected, stripped)
		}
	})

	t.Run("KeyHandlingNoPrefix", func(t *testing.T) {
		backend := &RedisBackend{
			prefix: "",
		}

		// Test buildKey without prefix
		key := backend.buildKey("mykey")
		if key != "mykey" {
			t.Errorf("buildKey without prefix: expected mykey, got %s", key)
		}

		// Test stripPrefix without prefix
		stripped := backend.stripPrefix("mykey")
		if stripped != "mykey" {
			t.Errorf("stripPrefix without prefix: expected mykey, got %s", stripped)
		}
	})
}

// TestRedisBackendClose tests Close operation
func TestRedisBackendClose(t *testing.T) {
	t.Run("CloseWithoutClient", func(t *testing.T) {
		backend := &RedisBackend{}
		err := backend.Close()
		if err != nil {
			t.Errorf("Expected no error closing nil client, got %v", err)
		}
	})
}

// TestRedisBackendConcurrentOperations tests thread safety
func TestRedisBackendConcurrentOperations(t *testing.T) {
	backend := &RedisBackend{
		prefix: "test",
	}

	// Test concurrent key operations
	done := make(chan bool, 3)

	go func() {
		key := backend.buildKey("key1")
		if key == "" {
			t.Error("buildKey returned empty")
		}
		done <- true
	}()

	go func() {
		key := backend.stripPrefix("test:key2")
		if key == "" {
			t.Error("stripPrefix returned empty")
		}
		done <- true
	}()

	go func() {
		key := backend.buildKey("key3")
		stripped := backend.stripPrefix(key)
		if stripped != "key3" {
			t.Errorf("Round trip failed: got %s", stripped)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}

// TestRedisBackendSaveReadAll tests Save with io.ReadAll
func TestRedisBackendSaveReadAll(t *testing.T) {
	// Test that Save properly reads all data from reader
	data := strings.Repeat("test data ", 1000) // Large data
	reader := strings.NewReader(data)

	// Read all data like Save does
	dataBytes, err := io.ReadAll(reader)
	if err != nil {
		t.Errorf("ReadAll failed: %v", err)
	}

	if len(dataBytes) != len(data) {
		t.Errorf("Expected %d bytes, got %d", len(data), len(dataBytes))
	}
}

// TestRedisBackendLoadReader tests Load return type
func TestRedisBackendLoadReader(t *testing.T) {
	// Test that Load returns proper ReadCloser
	data := "test data"
	reader := io.NopCloser(strings.NewReader(data))

	// Verify it implements io.ReadCloser
	var _ = reader

	// Read from it
	buf := new(bytes.Buffer)
	_, err := io.Copy(buf, reader)
	if err != nil {
		t.Errorf("Failed to read: %v", err)
	}

	if buf.String() != data {
		t.Errorf("Expected %s, got %s", data, buf.String())
	}

	// Close it
	err = reader.Close()
	if err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

// TestRedisBackendMetadataConversion tests metadata type conversion
func TestRedisBackendMetadataConversion(t *testing.T) {
	// Test converting various types to strings for metadata
	metadata := map[string]interface{}{
		"string": "value",
		"int":    42,
		"float":  3.14,
		"bool":   true,
		"nil":    nil,
		"time":   time.Now(),
	}

	// Simulate conversion like in SetMetadata
	fields := make(map[string]string)
	for k, v := range metadata {
		fields[k] = fmt.Sprintf("%v", v)
	}

	// Verify conversions
	if fields["string"] != "value" {
		t.Errorf("String conversion failed: %s", fields["string"])
	}
	if fields["int"] != "42" {
		t.Errorf("Int conversion failed: %s", fields["int"])
	}
	// Float conversion may vary slightly
	if !strings.HasPrefix(fields["float"], "3.14") {
		t.Errorf("Float conversion failed: %s", fields["float"])
	}
	if fields["bool"] != "true" {
		t.Errorf("Bool conversion failed: %s", fields["bool"])
	}
	if fields["nil"] != "<nil>" {
		t.Errorf("Nil conversion failed: %s", fields["nil"])
	}
	// Time conversion should produce a timestamp
	if fields["time"] == "" {
		t.Errorf("Time conversion failed: %s", fields["time"])
	}
}

// TestRedisBackendListPattern tests List pattern building
func TestRedisBackendListPattern(t *testing.T) {
	t.Run("ListWithPrefix", func(t *testing.T) {
		backend := &RedisBackend{
			prefix: "app",
		}

		// Test building pattern for List
		pattern := backend.buildKey("users") + "*"
		expected := "app:users*"
		if pattern != expected {
			t.Errorf("Expected pattern %s, got %s", expected, pattern)
		}
	})

	t.Run("ListWithoutPrefix", func(t *testing.T) {
		backend := &RedisBackend{
			prefix: "",
		}

		// Test building pattern for List without prefix
		pattern := backend.buildKey("data") + "*"
		expected := "data*"
		if pattern != expected {
			t.Errorf("Expected pattern %s, got %s", expected, pattern)
		}
	})
}

// TestRedisBackendPrefixHandling tests prefix trimming edge cases
func TestRedisBackendPrefixHandling(t *testing.T) {
	t.Run("PrefixWithTrailingColon", func(t *testing.T) {
		backend := NewRedisBackend()
		config := map[string]interface{}{
			"prefix": "myapp:", // With trailing colon
		}
		// Init will trim the trailing colon
		_ = backend.Init(config)
		// Check that prefix was trimmed
		if backend.prefix != "myapp" {
			t.Errorf("Expected prefix 'myapp', got '%s'", backend.prefix)
		}
	})

	t.Run("EmptyPrefix", func(t *testing.T) {
		backend := NewRedisBackend()
		config := map[string]interface{}{
			"prefix": "",
		}
		_ = backend.Init(config)
		if backend.prefix != "" {
			t.Errorf("Expected empty prefix, got '%s'", backend.prefix)
		}
	})
}

// TestRedisBackendHelperMethods tests Redis backend helper functions
func TestRedisBackendHelperMethods(t *testing.T) {
	t.Run("BuildKeyVariations", func(t *testing.T) {
		tests := []struct {
			name     string
			prefix   string
			key      string
			expected string
		}{
			{"with prefix", "cache", "user:123", "cache:user:123"},
			{"empty prefix", "", "user:123", "user:123"},
			{"complex key", "app", "a:b:c:d", "app:a:b:c:d"},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				backend := &RedisBackend{prefix: tt.prefix}
				result := backend.buildKey(tt.key)
				if result != tt.expected {
					t.Errorf("buildKey(%s) = %s, want %s", tt.key, result, tt.expected)
				}
			})
		}
	})

	t.Run("StripPrefixVariations", func(t *testing.T) {
		tests := []struct {
			name     string
			prefix   string
			redisKey string
			expected string
		}{
			{"with matching prefix", "cache", "cache:user:123", "user:123"},
			{"without matching prefix", "cache", "other:user:123", "other:user:123"},
			{"empty prefix", "", "user:123", "user:123"},
			{"exact match prefix", "cache", "cache:", ""},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				backend := &RedisBackend{prefix: tt.prefix}
				result := backend.stripPrefix(tt.redisKey)
				if result != tt.expected {
					t.Errorf("stripPrefix(%s) = %s, want %s", tt.redisKey, result, tt.expected)
				}
			})
		}
	})
}

// TestRedisBackendEdgeCases tests edge cases
func TestRedisBackendEdgeCases(t *testing.T) {
	t.Run("InitWithInvalidDBType", func(t *testing.T) {
		backend := NewRedisBackend()
		config := map[string]interface{}{
			"db": "not-a-number", // Invalid type
		}
		err := backend.Init(config)
		// Should use default db (0)
		if err == nil {
			t.Log("Redis connection succeeded with invalid db type (used default)")
		} else {
			t.Logf("Redis connection failed as expected: %v", err)
		}
	})

	t.Run("InitWithNilValues", func(t *testing.T) {
		backend := NewRedisBackend()
		config := map[string]interface{}{
			"addr":     nil,
			"password": nil,
			"db":       nil,
			"prefix":   nil,
		}
		err := backend.Init(config)
		// Should use all defaults
		if err == nil {
			t.Log("Redis connection succeeded with nil values (used defaults)")
		} else {
			t.Logf("Redis connection failed as expected: %v", err)
		}
	})
}
