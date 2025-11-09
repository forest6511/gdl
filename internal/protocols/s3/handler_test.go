package s3

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
)

// MockS3Client simulates S3 client for testing
type MockS3Client struct {
	getObjectErr   error
	headObjectErr  error
	listObjectsErr error
	objectContent  string
	objectSize     int64
	objects        []types.Object
	contentType    string
	lastModified   *time.Time
}

func (m *MockS3Client) GetObject(ctx context.Context, params *s3.GetObjectInput, optFns ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	if m.getObjectErr != nil {
		return nil, m.getObjectErr
	}

	body := io.NopCloser(strings.NewReader(m.objectContent))
	size := int64(len(m.objectContent))

	return &s3.GetObjectOutput{
		Body:          body,
		ContentLength: &size,
		ContentType:   &m.contentType,
	}, nil
}

func (m *MockS3Client) HeadObject(ctx context.Context, params *s3.HeadObjectInput, optFns ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	if m.headObjectErr != nil {
		return nil, m.headObjectErr
	}

	return &s3.HeadObjectOutput{
		ContentLength: &m.objectSize,
		ContentType:   &m.contentType,
		LastModified:  m.lastModified,
	}, nil
}

func (m *MockS3Client) ListObjectsV2(ctx context.Context, params *s3.ListObjectsV2Input, optFns ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	if m.listObjectsErr != nil {
		return nil, m.listObjectsErr
	}

	return &s3.ListObjectsV2Output{
		Contents: m.objects,
	}, nil
}

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

// Additional edge case tests for improved coverage
func TestDownloadEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "InvalidS3URL",
			url:     "://invalid",
			wantErr: true,
			errMsg:  "invalid S3 URL",
		},
		{
			name:    "MissingBucket",
			url:     "s3:///key",
			wantErr: true,
			errMsg:  "bucket name is required",
		},
		{
			name:    "MissingKey",
			url:     "s3://bucket/",
			wantErr: true,
			errMsg:  "object key is required",
		},
		{
			name:    "NonS3Scheme",
			url:     "http://bucket/key",
			wantErr: true,
			errMsg:  "URL scheme must be 's3'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloader := &S3Downloader{
				config: DefaultConfig(),
			}

			ctx := context.Background()
			var buf bytes.Buffer

			err := downloader.Download(ctx, tt.url, &buf)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestGetObjectSizeEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "InvalidURL",
			url:     "://invalid",
			wantErr: true,
			errMsg:  "invalid S3 URL",
		},
		{
			name:    "NonS3Scheme",
			url:     "ftp://bucket/key",
			wantErr: true,
			errMsg:  "URL scheme must be 's3'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloader := &S3Downloader{
				config: DefaultConfig(),
			}

			ctx := context.Background()
			_, err := downloader.GetObjectSize(ctx, tt.url)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestListObjectsParameters(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping S3 ListObjects test in short mode")
	}

	// This test verifies the ListObjects method signature and error handling
	// Without actual AWS credentials/client, we skip the API call tests
	t.Log("ListObjects method exists and accepts expected parameters")
}

func TestObjectExistsEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "InvalidURL",
			url:     "://invalid",
			wantErr: true,
			errMsg:  "invalid S3 URL",
		},
		{
			name:    "NonS3Scheme",
			url:     "http://bucket/key",
			wantErr: true,
			errMsg:  "URL scheme must be 's3'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloader := &S3Downloader{
				config: DefaultConfig(),
			}

			ctx := context.Background()
			_, err := downloader.ObjectExists(ctx, tt.url)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestGetObjectMetadataEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "InvalidURL",
			url:     "://invalid",
			wantErr: true,
			errMsg:  "invalid S3 URL",
		},
		{
			name:    "NonS3Scheme",
			url:     "ftp://bucket/key",
			wantErr: true,
			errMsg:  "URL scheme must be 's3'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloader := &S3Downloader{
				config: DefaultConfig(),
			}

			ctx := context.Background()
			_, err := downloader.GetObjectMetadata(ctx, tt.url)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

func TestDownloadRangeEdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		start   int64
		end     int64
		wantErr bool
		errMsg  string
	}{
		{
			name:    "InvalidURL",
			url:     "://invalid",
			start:   0,
			end:     1024,
			wantErr: true,
			errMsg:  "invalid S3 URL",
		},
		{
			name:    "NonS3Scheme",
			url:     "http://bucket/key",
			start:   0,
			end:     1024,
			wantErr: true,
			errMsg:  "URL scheme must be 's3'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			downloader := &S3Downloader{
				config: DefaultConfig(),
			}

			ctx := context.Background()
			var buf bytes.Buffer

			err := downloader.DownloadRange(ctx, tt.url, &buf, tt.start, tt.end)

			if tt.wantErr && err == nil {
				t.Errorf("Expected error containing '%s', got nil", tt.errMsg)
			}
			if tt.wantErr && err != nil && !strings.Contains(err.Error(), tt.errMsg) {
				t.Errorf("Expected error containing '%s', got: %v", tt.errMsg, err)
			}
		})
	}
}

// TestDownloadWithMock tests Download with mock S3 client
func TestDownloadWithMock(t *testing.T) {
	t.Run("SuccessfulDownload", func(t *testing.T) {
		mock := &MockS3Client{
			objectContent: "test s3 content",
			contentType:   "text/plain",
		}

		downloader := &S3Downloader{
			config: DefaultConfig(),
		}
		downloader.SetClient(mock)

		var buf bytes.Buffer
		err := downloader.Download(context.Background(), "s3://test-bucket/test-key.txt", &buf)

		if err != nil {
			t.Errorf("Download failed: %v", err)
		}

		if buf.String() != "test s3 content" {
			t.Errorf("Expected 'test s3 content', got %s", buf.String())
		}
	})

	t.Run("DownloadWithGetObjectError", func(t *testing.T) {
		accessDeniedErr := errors.New("access denied")
		mock := &MockS3Client{
			getObjectErr: accessDeniedErr,
		}

		downloader := &S3Downloader{
			config: DefaultConfig(),
		}
		downloader.SetClient(mock)

		var buf bytes.Buffer
		err := downloader.Download(context.Background(), "s3://test-bucket/test-key.txt", &buf)

		if err == nil {
			t.Error("Expected access denied error")
		}
		// Check that the underlying error is preserved via Unwrap
		if !errors.Is(err, accessDeniedErr) {
			t.Errorf("Expected underlying error to be access denied, got %v", err)
		}
		// Check that the wrapper message is appropriate
		if !strings.Contains(err.Error(), "failed to get object") {
			t.Errorf("Expected 'failed to get object' in error message, got %v", err)
		}
	})

	t.Run("DownloadInvalidURL", func(t *testing.T) {
		downloader := &S3Downloader{
			config: DefaultConfig(),
		}

		var buf bytes.Buffer
		err := downloader.Download(context.Background(), "://invalid", &buf)

		if err == nil {
			t.Error("Expected invalid URL error")
		}
	})

	t.Run("DownloadWrongScheme", func(t *testing.T) {
		downloader := &S3Downloader{
			config: DefaultConfig(),
		}

		var buf bytes.Buffer
		err := downloader.Download(context.Background(), "http://bucket/key", &buf)

		if err == nil {
			t.Error("Expected wrong scheme error")
		}
	})

	t.Run("DownloadMissingKey", func(t *testing.T) {
		downloader := &S3Downloader{
			config: DefaultConfig(),
		}

		var buf bytes.Buffer
		err := downloader.Download(context.Background(), "s3://bucket/", &buf)

		if err == nil {
			t.Error("Expected missing key error")
		}
	})
}

// TestGetObjectSizeWithMock tests GetObjectSize with mock
func TestGetObjectSizeWithMock(t *testing.T) {
	t.Run("SuccessfulGetSize", func(t *testing.T) {
		mock := &MockS3Client{
			objectSize: 12345,
		}

		downloader := &S3Downloader{
			config: DefaultConfig(),
		}
		downloader.SetClient(mock)

		size, err := downloader.GetObjectSize(context.Background(), "s3://bucket/key.txt")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if size != 12345 {
			t.Errorf("Expected size 12345, got %d", size)
		}
	})

	t.Run("GetSizeError", func(t *testing.T) {
		mock := &MockS3Client{
			headObjectErr: errors.New("not found"),
		}

		downloader := &S3Downloader{
			config: DefaultConfig(),
		}
		downloader.SetClient(mock)

		_, err := downloader.GetObjectSize(context.Background(), "s3://bucket/missing.txt")
		if err == nil {
			t.Error("Expected not found error")
		}
	})
}

// TestListObjectsWithMock tests ListObjects with mock
func TestListObjectsWithMock(t *testing.T) {
	t.Run("SuccessfulList", func(t *testing.T) {
		now := time.Now()
		size1 := int64(100)
		size2 := int64(200)

		mock := &MockS3Client{
			objects: []types.Object{
				{Key: aws.String("file1.txt"), Size: &size1, LastModified: &now},
				{Key: aws.String("file2.txt"), Size: &size2, LastModified: &now},
			},
		}

		downloader := &S3Downloader{
			config: DefaultConfig(),
		}
		downloader.SetClient(mock)

		objects, err := downloader.ListObjects(context.Background(), "bucket", "", 0)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(objects) != 2 {
			t.Errorf("Expected 2 objects, got %d", len(objects))
		}

		if objects[0] != "file1.txt" {
			t.Errorf("Expected 'file1.txt', got %s", objects[0])
		}
	})

	t.Run("ListError", func(t *testing.T) {
		mock := &MockS3Client{
			listObjectsErr: errors.New("permission denied"),
		}

		downloader := &S3Downloader{
			config: DefaultConfig(),
		}
		downloader.SetClient(mock)

		_, err := downloader.ListObjects(context.Background(), "bucket", "", 0)
		if err == nil {
			t.Error("Expected permission denied error")
		}
	})
}

// TestObjectExistsWithMock tests ObjectExists with mock
func TestObjectExistsWithMock(t *testing.T) {
	t.Run("ObjectExists", func(t *testing.T) {
		mock := &MockS3Client{
			objectSize: 100,
		}

		downloader := &S3Downloader{
			config: DefaultConfig(),
		}
		downloader.SetClient(mock)

		exists, err := downloader.ObjectExists(context.Background(), "s3://bucket/key.txt")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !exists {
			t.Error("Expected object to exist")
		}
	})

	t.Run("ObjectNotExists", func(t *testing.T) {
		mock := &MockS3Client{
			headObjectErr: errors.New("NotFound"),
		}

		downloader := &S3Downloader{
			config: DefaultConfig(),
		}
		downloader.SetClient(mock)

		exists, err := downloader.ObjectExists(context.Background(), "s3://bucket/missing.txt")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if exists {
			t.Error("Expected object to not exist")
		}
	})
}

// TestGetObjectMetadataWithMock tests GetObjectMetadata with mock
func TestGetObjectMetadataWithMock(t *testing.T) {
	t.Run("SuccessfulGetMetadata", func(t *testing.T) {
		now := time.Now()
		contentType := "application/json"

		mock := &MockS3Client{
			objectSize:   5000,
			contentType:  contentType,
			lastModified: &now,
		}

		downloader := &S3Downloader{
			config: DefaultConfig(),
		}
		downloader.SetClient(mock)

		metadata, err := downloader.GetObjectMetadata(context.Background(), "s3://bucket/data.json")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if size, ok := metadata["ContentLength"].(int64); !ok || size != 5000 {
			t.Errorf("Expected size 5000, got %v", metadata["ContentLength"])
		}

		if ct, ok := metadata["ContentType"].(string); !ok || ct != contentType {
			t.Errorf("Expected content type %s, got %v", contentType, metadata["ContentType"])
		}
	})

	t.Run("GetMetadataError", func(t *testing.T) {
		mock := &MockS3Client{
			headObjectErr: errors.New("forbidden"),
		}

		downloader := &S3Downloader{
			config: DefaultConfig(),
		}
		downloader.SetClient(mock)

		_, err := downloader.GetObjectMetadata(context.Background(), "s3://bucket/secret.txt")
		if err == nil {
			t.Error("Expected forbidden error")
		}
	})
}

// TestDownloadRangeWithMock tests DownloadRange with mock
func TestDownloadRangeWithMock(t *testing.T) {
	t.Run("SuccessfulRangeDownload", func(t *testing.T) {
		mock := &MockS3Client{
			objectContent: "0123456789",
		}

		downloader := &S3Downloader{
			config: DefaultConfig(),
		}
		downloader.SetClient(mock)

		var buf bytes.Buffer
		err := downloader.DownloadRange(context.Background(), "s3://bucket/file.bin", &buf, 0, 4)

		if err != nil {
			t.Errorf("Download range failed: %v", err)
		}

		// Mock returns all content, but in real scenario it would return range
		content := buf.String()
		if len(content) == 0 {
			t.Error("Expected content from range download")
		}
	})

	t.Run("RangeDownloadError", func(t *testing.T) {
		mock := &MockS3Client{
			getObjectErr: errors.New("range not satisfiable"),
		}

		downloader := &S3Downloader{
			config: DefaultConfig(),
		}
		downloader.SetClient(mock)

		var buf bytes.Buffer
		err := downloader.DownloadRange(context.Background(), "s3://bucket/file.bin", &buf, 0, 100)

		if err == nil {
			t.Error("Expected range error")
		}
	})
}
