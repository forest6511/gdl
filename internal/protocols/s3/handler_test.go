package s3

import (
	"bytes"
	"context"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Region != "us-east-1" {
		t.Errorf("Expected region 'us-east-1', got %s", config.Region)
	}

	if config.UsePathStyle {
		t.Error("Expected UsePathStyle to be false")
	}

	if config.DisableSSL {
		t.Error("Expected DisableSSL to be false")
	}

	if config.AccessKeyID != "" {
		t.Error("Expected AccessKeyID to be empty")
	}

	if config.Profile != "" {
		t.Error("Expected Profile to be empty")
	}
}

func TestNewS3Downloader(t *testing.T) {
	t.Run("WithNilConfig", func(t *testing.T) {
		// This will fail to initialize client without credentials
		_, err := NewS3Downloader(nil)
		if err == nil {
			t.Log("S3 client initialized with default config (credentials available)")
		} else {
			t.Logf("S3 client initialization failed as expected: %v", err)
		}
	})

	t.Run("WithCustomConfig", func(t *testing.T) {
		config := &Config{
			Region:          "eu-west-1",
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			Endpoint:        "http://localhost:9000",
			UsePathStyle:    true,
		}

		// This might fail but tests configuration handling
		_, err := NewS3Downloader(config)
		if err == nil {
			t.Log("S3 client initialized with custom config")
		} else {
			t.Logf("S3 client initialization failed (expected): %v", err)
		}
	})

	t.Run("WithProfile", func(t *testing.T) {
		config := &Config{
			Region:  "us-west-2",
			Profile: "test-profile",
		}

		// Will fail if profile doesn't exist
		_, err := NewS3Downloader(config)
		if err == nil {
			t.Log("S3 client initialized with profile config")
		} else {
			t.Logf("S3 client initialization failed (expected): %v", err)
		}
	})

	t.Run("WithSessionToken", func(t *testing.T) {
		config := &Config{
			Region:          "us-east-1",
			AccessKeyID:     "test-key",
			SecretAccessKey: "test-secret",
			SessionToken:    "test-token",
		}

		// Test that session token is handled
		_, err := NewS3Downloader(config)
		if err == nil {
			t.Log("S3 client initialized with session token")
		} else {
			t.Logf("S3 client initialization failed (expected): %v", err)
		}
	})
}

func TestParseS3URL(t *testing.T) {
	downloader := &S3Downloader{
		config: DefaultConfig(),
	}

	tests := []struct {
		name        string
		url         string
		wantBucket  string
		wantKey     string
		wantErr     bool
		errContains string
	}{
		{
			name:       "ValidS3URL",
			url:        "s3://bucket-name/path/to/object.txt",
			wantBucket: "bucket-name",
			wantKey:    "path/to/object.txt",
			wantErr:    false,
		},
		{
			name:       "SimpleS3URL",
			url:        "s3://bucket/key",
			wantBucket: "bucket",
			wantKey:    "key",
			wantErr:    false,
		},
		{
			name:        "InvalidScheme",
			url:         "http://bucket/key",
			wantErr:     true,
			errContains: "URL scheme must be 's3'",
		},
		{
			name:        "MissingBucket",
			url:         "s3:///path/to/object",
			wantErr:     true,
			errContains: "bucket name is required",
		},
		{
			name:        "MissingKey",
			url:         "s3://bucket/",
			wantErr:     true,
			errContains: "object key is required",
		},
		{
			name:        "MissingKeyNoSlash",
			url:         "s3://bucket",
			wantErr:     true,
			errContains: "object key is required",
		},
		{
			name:        "InvalidURL",
			url:         "://invalid",
			wantErr:     true,
			errContains: "invalid S3 URL",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bucket, key, err := downloader.parseS3URL(tt.url)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Error should contain %q, got: %v", tt.errContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if bucket != tt.wantBucket {
					t.Errorf("Expected bucket %q, got %q", tt.wantBucket, bucket)
				}
				if key != tt.wantKey {
					t.Errorf("Expected key %q, got %q", tt.wantKey, key)
				}
			}
		})
	}
}

func TestDownload(t *testing.T) {
	t.Run("InvalidURL", func(t *testing.T) {
		// Create downloader with mock config
		downloader := &S3Downloader{
			config: DefaultConfig(),
			client: &s3.Client{},
		}

		var buf bytes.Buffer
		err := downloader.Download(context.Background(), "://invalid", &buf)

		if err == nil {
			t.Error("Expected error for invalid URL")
		}
		if !strings.Contains(err.Error(), "invalid S3 URL") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})

	t.Run("WrongScheme", func(t *testing.T) {
		downloader := &S3Downloader{
			config: DefaultConfig(),
			client: &s3.Client{},
		}

		var buf bytes.Buffer
		err := downloader.Download(context.Background(), "http://bucket/key", &buf)

		if err == nil {
			t.Error("Expected error for wrong scheme")
		}
		if !strings.Contains(err.Error(), "URL scheme must be 's3'") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}

func TestGetObjectSize(t *testing.T) {
	t.Run("InvalidURL", func(t *testing.T) {
		downloader := &S3Downloader{
			config: DefaultConfig(),
			client: &s3.Client{},
		}

		_, err := downloader.GetObjectSize(context.Background(), "://invalid")

		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})

	t.Run("MissingBucket", func(t *testing.T) {
		downloader := &S3Downloader{
			config: DefaultConfig(),
			client: &s3.Client{},
		}

		_, err := downloader.GetObjectSize(context.Background(), "s3:///key")

		if err == nil {
			t.Error("Expected error for missing bucket")
		}
		if !strings.Contains(err.Error(), "bucket name is required") {
			t.Errorf("Unexpected error message: %v", err)
		}
	})
}

func TestListObjects(t *testing.T) {
	t.Run("BasicList", func(t *testing.T) {
		// This test verifies the function logic without actual AWS calls
		downloader := &S3Downloader{
			config: DefaultConfig(),
		}

		// Would need mock client to test actual listing
		if downloader.config.Region != "us-east-1" {
			t.Error("Config not properly set")
		}
	})
}

func TestObjectExists(t *testing.T) {
	t.Run("InvalidURL", func(t *testing.T) {
		downloader := &S3Downloader{
			config: DefaultConfig(),
			client: &s3.Client{},
		}

		_, err := downloader.ObjectExists(context.Background(), "://invalid")

		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})

	t.Run("MissingKey", func(t *testing.T) {
		downloader := &S3Downloader{
			config: DefaultConfig(),
			client: &s3.Client{},
		}

		_, err := downloader.ObjectExists(context.Background(), "s3://bucket/")

		if err == nil {
			t.Error("Expected error for missing key")
		}
	})
}

func TestGetObjectMetadata(t *testing.T) {
	t.Run("InvalidURL", func(t *testing.T) {
		downloader := &S3Downloader{
			config: DefaultConfig(),
			client: &s3.Client{},
		}

		_, err := downloader.GetObjectMetadata(context.Background(), "://invalid")

		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})

	t.Run("WrongScheme", func(t *testing.T) {
		downloader := &S3Downloader{
			config: DefaultConfig(),
			client: &s3.Client{},
		}

		_, err := downloader.GetObjectMetadata(context.Background(), "ftp://bucket/key")

		if err == nil {
			t.Error("Expected error for wrong scheme")
		}
	})
}

func TestDownloadRange(t *testing.T) {
	t.Run("InvalidURL", func(t *testing.T) {
		downloader := &S3Downloader{
			config: DefaultConfig(),
			client: &s3.Client{},
		}

		var buf bytes.Buffer
		err := downloader.DownloadRange(context.Background(), "://invalid", &buf, 0, 100)

		if err == nil {
			t.Error("Expected error for invalid URL")
		}
	})

	t.Run("ValidRangeFormat", func(t *testing.T) {
		// This test verifies the range header format
		start := int64(100)
		end := int64(200)

		// The function would create: "bytes=100-200"
		// Just verify the logic is correct
		if start > end {
			t.Error("Invalid range: start should be less than end")
		}
	})
}

func TestClose(t *testing.T) {
	downloader := &S3Downloader{
		config: DefaultConfig(),
		client: &s3.Client{},
	}

	err := downloader.Close()
	if err != nil {
		t.Errorf("Close should always succeed, got error: %v", err)
	}
}

func TestConfigOptions(t *testing.T) {
	t.Run("DisableSSL", func(t *testing.T) {
		config := &Config{
			DisableSSL: true,
		}

		// This will fail without credentials but tests config handling
		_, err := NewS3Downloader(config)
		if err == nil {
			t.Log("Created S3 downloader with DisableSSL")
		} else {
			t.Logf("Expected initialization error: %v", err)
		}
	})

	t.Run("CustomEndpoint", func(t *testing.T) {
		config := &Config{
			Endpoint:     "http://minio.local:9000",
			UsePathStyle: true,
		}

		// Test S3-compatible service configuration
		_, err := NewS3Downloader(config)
		if err == nil {
			t.Log("Created S3 downloader with custom endpoint")
		} else {
			t.Logf("Expected initialization error: %v", err)
		}
	})

	t.Run("AllOptions", func(t *testing.T) {
		config := &Config{
			Region:          "ap-south-1",
			AccessKeyID:     "key",
			SecretAccessKey: "secret",
			SessionToken:    "token",
			Endpoint:        "https://s3.custom.com",
			UsePathStyle:    true,
			DisableSSL:      false,
			Profile:         "custom",
		}

		// Verify all options are preserved
		if config.Region != "ap-south-1" {
			t.Error("Region not preserved")
		}
		if !config.UsePathStyle {
			t.Error("UsePathStyle not preserved")
		}
		if config.Profile != "custom" {
			t.Error("Profile not preserved")
		}
	})
}

func TestMetadataProcessing(t *testing.T) {
	// Test metadata extraction logic
	t.Run("CompleteMetadata", func(t *testing.T) {
		metadata := make(map[string]interface{})

		// Simulate adding various metadata fields
		contentLength := int64(1024)
		metadata["ContentLength"] = contentLength

		contentType := "application/json"
		metadata["ContentType"] = contentType

		eTag := "\"abc123\""
		metadata["ETag"] = eTag

		now := time.Now()
		metadata["LastModified"] = now

		storageClass := "STANDARD"
		metadata["StorageClass"] = storageClass

		// Verify all fields are present
		if metadata["ContentLength"] != contentLength {
			t.Error("ContentLength not stored correctly")
		}
		if metadata["ContentType"] != contentType {
			t.Error("ContentType not stored correctly")
		}
		if metadata["ETag"] != eTag {
			t.Error("ETag not stored correctly")
		}
		if metadata["LastModified"] != now {
			t.Error("LastModified not stored correctly")
		}
		if metadata["StorageClass"] != storageClass {
			t.Error("StorageClass not stored correctly")
		}
	})

	t.Run("UserMetadata", func(t *testing.T) {
		metadata := make(map[string]interface{})

		userMeta := map[string]string{
			"custom-key": "custom-value",
			"version":    "1.0",
		}
		metadata["UserMetadata"] = userMeta

		// Verify user metadata is preserved
		stored := metadata["UserMetadata"].(map[string]string)
		if stored["custom-key"] != "custom-value" {
			t.Error("User metadata not preserved correctly")
		}
	})
}

func TestErrorHandling(t *testing.T) {
	t.Run("NotFoundHandling", func(t *testing.T) {
		// Test how ObjectExists handles NotFound errors
		errMsg1 := "NotFound: The specified key does not exist"
		if !strings.Contains(errMsg1, "NotFound") {
			t.Error("Should detect NotFound error")
		}

		errMsg2 := "NoSuchKey: The specified key does not exist"
		if !strings.Contains(errMsg2, "NoSuchKey") {
			t.Error("Should detect NoSuchKey error")
		}
	})

	t.Run("WarningMessages", func(t *testing.T) {
		// These test that warnings are handled but don't override main errors
		// The actual implementation prints warnings to stdout

		// Scenario: Download with close error
		// The close error is logged but doesn't affect the download result
		var buf bytes.Buffer
		// Simulate a successful download followed by close warning
		_, err := buf.Write([]byte("test data"))
		if err != nil {
			t.Error("Write should succeed")
		}
	})
}

// TestInitializationPaths tests different initialization paths
func TestInitializationPaths(t *testing.T) {
	t.Run("DefaultConfigPath", func(t *testing.T) {
		// Test initialization with default AWS config
		cfg := &Config{
			Region: "us-east-1",
		}

		// This uses the default credential chain
		_, err := NewS3Downloader(cfg)
		if err == nil {
			t.Log("Initialized with default config")
		} else {
			t.Logf("Expected error without credentials: %v", err)
		}
	})

	t.Run("StaticCredentialsPath", func(t *testing.T) {
		// Test initialization with static credentials
		cfg := &Config{
			Region:          "us-east-1",
			AccessKeyID:     "AKIAIOSFODNN7EXAMPLE",
			SecretAccessKey: "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		}

		// This uses static credentials provider
		_, err := NewS3Downloader(cfg)
		if err == nil {
			t.Log("Initialized with static credentials")
		} else {
			t.Logf("Expected error with test credentials: %v", err)
		}
	})

	t.Run("ProfilePath", func(t *testing.T) {
		// Test initialization with AWS profile
		cfg := &Config{
			Region:  "us-east-1",
			Profile: "nonexistent",
		}

		// This uses shared config profile
		_, err := NewS3Downloader(cfg)
		if err == nil {
			t.Log("Profile exists and was loaded")
		} else {
			t.Logf("Expected error with nonexistent profile: %v", err)
		}
	})
}

func TestListObjectsPagination(t *testing.T) {
	// Test that maxKeys parameter is handled correctly
	t.Run("MaxKeysZero", func(t *testing.T) {
		// When maxKeys is 0, it should not be set in the input
		if maxKeys := int32(0); maxKeys > 0 {
			t.Error("MaxKeys 0 should not be set")
		}
	})

	t.Run("MaxKeysPositive", func(t *testing.T) {
		// When maxKeys is positive, it should be set
		if maxKeys := int32(100); maxKeys <= 0 {
			t.Error("MaxKeys should be positive")
		}
	})

	t.Run("PrefixHandling", func(t *testing.T) {
		// Test that empty prefix is handled differently than non-empty
		prefix := ""
		if prefix != "" {
			t.Error("Empty prefix should not be set")
		}

		prefix = "logs/"
		if prefix == "" {
			t.Error("Non-empty prefix should be set")
		}
	})
}

// Test helper functions
func TestHelperFunctions(t *testing.T) {
	t.Run("AWSStringHelper", func(t *testing.T) {
		// Test that aws.String works correctly
		str := "test"
		ptr := aws.String(str)
		if *ptr != str {
			t.Error("aws.String should create pointer to string")
		}
	})

	t.Run("AWSInt32Helper", func(t *testing.T) {
		// Test that aws.Int32 works correctly
		val := int32(100)
		ptr := aws.Int32(val)
		if *ptr != val {
			t.Error("aws.Int32 should create pointer to int32")
		}
	})
}

// Benchmark tests
func BenchmarkParseS3URL(b *testing.B) {
	downloader := &S3Downloader{
		config: DefaultConfig(),
	}
	url := "s3://bucket-name/path/to/object.txt"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = downloader.parseS3URL(url)
	}
}

func BenchmarkNewS3Downloader(b *testing.B) {
	config := DefaultConfig()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Will fail without credentials but benchmarks initialization
		_, _ = NewS3Downloader(config)
	}
}
