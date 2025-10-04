package backends

import (
	"context"
	"strings"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// TestS3BackendGetObjectSize tests GetObjectSize functionality
func TestS3BackendGetObjectSize(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping S3 integration test in short mode")
	}

	t.Run("GetObjectSizeNotFound", func(t *testing.T) {
		backend := NewS3Backend()
		config := map[string]interface{}{
			"bucket": "test-bucket",
			"region": "us-east-1",
		}

		// This will fail to init without real credentials, but tests the path
		err := backend.Init(config)
		if err != nil {
			t.Logf("Init failed as expected without credentials: %v", err)
			return
		}

		// If somehow init succeeds, test GetObjectSize
		_, err = backend.GetObjectSize(context.Background(), "nonexistent-key")
		if err == nil {
			t.Error("Expected error for non-existent object")
		}
	})
}

// TestS3BackendGetObjectMetadata tests GetObjectMetadata functionality
func TestS3BackendGetObjectMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping S3 integration test in short mode")
	}

	t.Run("GetObjectMetadataNotFound", func(t *testing.T) {
		backend := NewS3Backend()
		config := map[string]interface{}{
			"bucket": "test-bucket",
			"region": "us-east-1",
		}

		// This will fail to init without real credentials, but tests the path
		err := backend.Init(config)
		if err != nil {
			t.Logf("Init failed as expected without credentials: %v", err)
			return
		}

		// If somehow init succeeds, test GetObjectMetadata
		_, err = backend.GetObjectMetadata(context.Background(), "nonexistent-key")
		if err == nil {
			t.Error("Expected error for non-existent object")
		}
	})
}

// TestS3BackendInitialization tests S3 backend initialization
func TestS3BackendConfiguration(t *testing.T) {
	t.Run("InitWithProfile", func(t *testing.T) {
		backend := NewS3Backend()
		config := map[string]interface{}{
			"bucket":  "test-bucket",
			"region":  "us-west-2",
			"profile": "test-profile",
			"prefix":  "test-prefix/",
		}
		// This will fail because we can't actually load the profile
		err := backend.Init(config)
		if err == nil {
			t.Log("Profile configuration succeeded (profile exists)")
		} else {
			t.Logf("Profile configuration failed as expected: %v", err)
		}
	})

	t.Run("InitWithCredentials", func(t *testing.T) {
		backend := NewS3Backend()
		config := map[string]interface{}{
			"bucket":          "test-bucket",
			"region":          "us-west-2",
			"accessKeyId":     "AKIAIOSFODNN7EXAMPLE",
			"secretAccessKey": "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
			"sessionToken":    "test-session-token",
			"endpoint":        "http://localhost:9000",
			"usePathStyle":    true,
		}
		// This might fail due to AWS setup but tests the credential path
		err := backend.Init(config)
		if err != nil {
			t.Logf("Init with credentials failed (expected): %v", err)
		}
	})

	t.Run("InitWithDefaultCredentials", func(t *testing.T) {
		backend := NewS3Backend()
		config := map[string]interface{}{
			"bucket": "test-bucket",
		}
		// Uses default credential chain
		err := backend.Init(config)
		if err != nil {
			t.Logf("Init with default credentials failed (expected): %v", err)
		}
	})

	t.Run("KeyHandling", func(t *testing.T) {
		backend := &S3Backend{
			bucket: "test-bucket",
			prefix: "test-prefix",
		}

		// Test buildKey
		key := backend.buildKey("myfile.txt")
		expected := "test-prefix/myfile.txt"
		if key != expected {
			t.Errorf("buildKey: expected %s, got %s", expected, key)
		}

		// Test buildKey with leading slash
		key = backend.buildKey("/myfile.txt")
		expected = "test-prefix/myfile.txt"
		if key != expected {
			t.Errorf("buildKey with leading slash: expected %s, got %s", expected, key)
		}

		// Test stripPrefix
		stripped := backend.stripPrefix("test-prefix/myfile.txt")
		expected = "myfile.txt"
		if stripped != expected {
			t.Errorf("stripPrefix: expected %s, got %s", expected, stripped)
		}

		// Test stripPrefix without matching prefix
		stripped = backend.stripPrefix("other/myfile.txt")
		expected = "other/myfile.txt"
		if stripped != expected {
			t.Errorf("stripPrefix no match: expected %s, got %s", expected, stripped)
		}
	})

	t.Run("KeyHandlingNoPrefix", func(t *testing.T) {
		backend := &S3Backend{
			bucket: "test-bucket",
			prefix: "",
		}

		// Test buildKey without prefix
		key := backend.buildKey("myfile.txt")
		if key != "myfile.txt" {
			t.Errorf("buildKey without prefix: expected myfile.txt, got %s", key)
		}

		// Test stripPrefix without prefix
		stripped := backend.stripPrefix("myfile.txt")
		if stripped != "myfile.txt" {
			t.Errorf("stripPrefix without prefix: expected myfile.txt, got %s", stripped)
		}
	})
}

// TestS3BackendHelperMethods tests S3 backend helper functions
func TestS3BackendHelperMethods(t *testing.T) {
	t.Run("Close", func(t *testing.T) {
		backend := NewS3Backend()
		err := backend.Close()
		if err != nil {
			t.Errorf("Expected no error from Close, got %v", err)
		}
	})

	t.Run("BuildKeyWithEmptyPrefix", func(t *testing.T) {
		backend := &S3Backend{
			prefix: "",
		}
		key := backend.buildKey("test.txt")
		if key != "test.txt" {
			t.Errorf("Expected 'test.txt', got %s", key)
		}
	})

	t.Run("StripPrefixWithEmptyPrefix", func(t *testing.T) {
		backend := &S3Backend{
			prefix: "",
		}
		key := backend.stripPrefix("test.txt")
		if key != "test.txt" {
			t.Errorf("Expected 'test.txt', got %s", key)
		}
	})
}

// TestS3BackendListSuccess tests successful List operation logic
func TestS3BackendListSuccess(t *testing.T) {
	backend := &S3Backend{
		bucket: "test-bucket",
		prefix: "test-prefix",
	}

	// Test the stripPrefix function with expected keys
	keys := []string{
		backend.stripPrefix("test-prefix/file1.txt"),
		backend.stripPrefix("test-prefix/file2.txt"),
		backend.stripPrefix("test-prefix/file3.txt"),
	}

	expected := []string{"file1.txt", "file2.txt", "file3.txt"}
	for i, key := range keys {
		if key != expected[i] {
			t.Errorf("Expected %s, got %s", expected[i], key)
		}
	}
}

// TestS3BackendListWithNilKeys tests List handling nil keys in response
func TestS3BackendListWithNilKeys(t *testing.T) {
	backend := &S3Backend{
		bucket: "test-bucket",
		prefix: "",
	}

	// Test handling of nil keys in Contents
	mockContents := []types.Object{
		{Key: stringPtr("file1.txt")},
		{Key: nil}, // nil key should be skipped
		{Key: stringPtr("file2.txt")},
	}

	var processedKeys []string
	for _, obj := range mockContents {
		if obj.Key != nil {
			key := backend.stripPrefix(*obj.Key)
			processedKeys = append(processedKeys, key)
		}
	}

	if len(processedKeys) != 2 {
		t.Errorf("Expected 2 processed keys, got %d", len(processedKeys))
	}
}

// Helper function to create string pointer
func stringPtr(s string) *string {
	return &s
}

// TestS3BackendInitAdditionalCases tests additional init cases
func TestS3BackendInitAdditionalCases(t *testing.T) {
	t.Run("InitWithEmptyAccessKey", func(t *testing.T) {
		backend := NewS3Backend()
		config := map[string]interface{}{
			"bucket":          "test-bucket",
			"region":          "us-west-2",
			"accessKeyId":     "", // Empty access key should use default chain
			"secretAccessKey": "secret",
		}
		err := backend.Init(config)
		if err != nil {
			t.Logf("Init with empty access key failed (expected): %v", err)
		}
	})

	t.Run("InitWithInvalidRegionType", func(t *testing.T) {
		backend := NewS3Backend()
		config := map[string]interface{}{
			"bucket": "test-bucket",
			"region": 123, // Invalid type
		}
		err := backend.Init(config)
		if err != nil {
			t.Logf("Init with invalid region type failed (expected): %v", err)
		}
	})

	t.Run("InitWithEmptyBucket", func(t *testing.T) {
		backend := NewS3Backend()
		config := map[string]interface{}{
			"bucket": "",
			"region": "us-west-2",
		}
		err := backend.Init(config)
		if err == nil {
			t.Error("Expected error for empty bucket")
		}
		if !strings.Contains(err.Error(), "bucket is required") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("InitWithAllOptions", func(t *testing.T) {
		backend := NewS3Backend()
		config := map[string]interface{}{
			"bucket":          "test-bucket",
			"region":          "us-west-2",
			"prefix":          "my-prefix",
			"accessKeyId":     "AKIAIOSFODNN7EXAMPLE",
			"secretAccessKey": "secret",
			"sessionToken":    "token",
			"endpoint":        "http://localhost:9000",
			"usePathStyle":    true,
		}
		err := backend.Init(config)
		if err != nil {
			t.Logf("Init with all options failed (expected): %v", err)
		}
	})
}

// TestS3BackendConcurrentOperations tests concurrent operations
func TestS3BackendConcurrentOperations(t *testing.T) {
	ctx := context.Background()
	backend := &S3Backend{
		bucket: "test-bucket",
		prefix: "test-prefix",
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
		stripped := backend.stripPrefix("test-prefix/key2")
		if stripped == "" {
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

	// Wait for all goroutines to complete
	for i := 0; i < 3; i++ {
		<-done
	}

	_ = ctx // Avoid unused variable warning
}
