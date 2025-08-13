package backends

import (
	"context"
	"fmt"
	"io"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"

	"github.com/forest6511/godl/pkg/storage"
)

// S3Backend implements storage using AWS S3 or S3-compatible services
type S3Backend struct {
	client *s3.Client
	bucket string
	prefix string
}

// NewS3Backend creates a new S3 storage backend
func NewS3Backend() *S3Backend {
	return &S3Backend{}
}

// Init initializes the S3 backend with configuration
func (s3b *S3Backend) Init(config map[string]interface{}) error {
	bucket, ok := config["bucket"].(string)
	if !ok || bucket == "" {
		return fmt.Errorf("bucket is required for S3 backend")
	}
	s3b.bucket = bucket

	// Optional prefix for all keys
	if prefix, ok := config["prefix"].(string); ok {
		s3b.prefix = strings.TrimSuffix(prefix, "/")
	}

	// Initialize S3 client
	if err := s3b.initClient(config); err != nil {
		return fmt.Errorf("failed to initialize S3 client: %w", err)
	}

	return nil
}

// initClient initializes the AWS S3 client
func (s3b *S3Backend) initClient(config map[string]interface{}) error {
	ctx := context.Background()

	// Get region (required)
	region, ok := config["region"].(string)
	if !ok || region == "" {
		region = "us-east-1" // default region
	}

	var awsConfig aws.Config
	var err error

	// Check for profile-based authentication
	if profile, ok := config["profile"].(string); ok && profile != "" {
		awsConfig, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
			awsconfig.WithSharedConfigProfile(profile),
		)
	} else if accessKey, hasAccessKey := config["accessKeyId"].(string); hasAccessKey && accessKey != "" {
		// Use provided credentials
		secretKey, _ := config["secretAccessKey"].(string)
		sessionToken, _ := config["sessionToken"].(string)

		creds := credentials.NewStaticCredentialsProvider(accessKey, secretKey, sessionToken)
		awsConfig, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
			awsconfig.WithCredentialsProvider(creds),
		)
	} else {
		// Use default credential chain
		awsConfig, err = awsconfig.LoadDefaultConfig(ctx,
			awsconfig.WithRegion(region),
		)
	}

	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client with optional custom endpoint
	clientOptions := func(o *s3.Options) {
		if endpoint, ok := config["endpoint"].(string); ok && endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
		if usePathStyle, ok := config["usePathStyle"].(bool); ok {
			o.UsePathStyle = usePathStyle
		}
	}

	s3b.client = s3.NewFromConfig(awsConfig, clientOptions)

	return nil
}

// Save stores data to S3 at the specified key
func (s3b *S3Backend) Save(ctx context.Context, key string, data io.Reader) error {
	fullKey := s3b.buildKey(key)

	input := &s3.PutObjectInput{
		Bucket: aws.String(s3b.bucket),
		Key:    aws.String(fullKey),
		Body:   data,
	}

	_, err := s3b.client.PutObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to save object to S3 s3://%s/%s: %w", s3b.bucket, fullKey, err)
	}

	return nil
}

// Load retrieves data from S3 for the given key
func (s3b *S3Backend) Load(ctx context.Context, key string) (io.ReadCloser, error) {
	fullKey := s3b.buildKey(key)

	input := &s3.GetObjectInput{
		Bucket: aws.String(s3b.bucket),
		Key:    aws.String(fullKey),
	}

	result, err := s3b.client.GetObject(ctx, input)
	if err != nil {
		if strings.Contains(err.Error(), "NoSuchKey") {
			return nil, storage.ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get object from S3 s3://%s/%s: %w", s3b.bucket, fullKey, err)
	}

	return result.Body, nil
}

// Delete removes data from S3 for the given key
func (s3b *S3Backend) Delete(ctx context.Context, key string) error {
	fullKey := s3b.buildKey(key)

	// First check if the object exists
	exists, err := s3b.Exists(ctx, key)
	if err != nil {
		return err
	}
	if !exists {
		return storage.ErrKeyNotFound
	}

	input := &s3.DeleteObjectInput{
		Bucket: aws.String(s3b.bucket),
		Key:    aws.String(fullKey),
	}

	_, err = s3b.client.DeleteObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to delete object from S3 s3://%s/%s: %w", s3b.bucket, fullKey, err)
	}

	return nil
}

// Exists checks if data exists at the given key in S3
func (s3b *S3Backend) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := s3b.buildKey(key)

	input := &s3.HeadObjectInput{
		Bucket: aws.String(s3b.bucket),
		Key:    aws.String(fullKey),
	}

	_, err := s3b.client.HeadObject(ctx, input)
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence in S3 s3://%s/%s: %w", s3b.bucket, fullKey, err)
	}

	return true, nil
}

// List returns a list of keys with the given prefix
func (s3b *S3Backend) List(ctx context.Context, prefix string) ([]string, error) {
	fullPrefix := s3b.buildKey(prefix)

	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(s3b.bucket),
		Prefix: aws.String(fullPrefix),
	}

	var keys []string
	paginator := s3.NewListObjectsV2Paginator(s3b.client, input)

	for paginator.HasMorePages() {
		result, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to list objects in S3 bucket %s: %w", s3b.bucket, err)
		}

		for _, obj := range result.Contents {
			if obj.Key != nil {
				// Remove the prefix to get the original key
				key := s3b.stripPrefix(*obj.Key)
				keys = append(keys, key)
			}
		}
	}

	return keys, nil
}

// Close cleans up resources (no-op for S3)
func (s3b *S3Backend) Close() error {
	return nil
}

// buildKey constructs the full S3 key including any configured prefix
func (s3b *S3Backend) buildKey(key string) string {
	if s3b.prefix == "" {
		return key
	}
	return s3b.prefix + "/" + strings.TrimPrefix(key, "/")
}

// stripPrefix removes the configured prefix from an S3 key to get the original key
func (s3b *S3Backend) stripPrefix(s3Key string) string {
	if s3b.prefix == "" {
		return s3Key
	}

	prefixWithSlash := s3b.prefix + "/"
	if strings.HasPrefix(s3Key, prefixWithSlash) {
		return strings.TrimPrefix(s3Key, prefixWithSlash)
	}

	return s3Key
}

// GetObjectSize returns the size of an object in S3
func (s3b *S3Backend) GetObjectSize(ctx context.Context, key string) (int64, error) {
	fullKey := s3b.buildKey(key)

	input := &s3.HeadObjectInput{
		Bucket: aws.String(s3b.bucket),
		Key:    aws.String(fullKey),
	}

	result, err := s3b.client.HeadObject(ctx, input)
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchKey") {
			return 0, storage.ErrKeyNotFound
		}
		return 0, fmt.Errorf("failed to get object metadata from S3 s3://%s/%s: %w", s3b.bucket, fullKey, err)
	}

	if result.ContentLength == nil {
		return 0, fmt.Errorf("content length not available for object s3://%s/%s", s3b.bucket, fullKey)
	}

	return *result.ContentLength, nil
}

// GetObjectMetadata returns metadata for an S3 object
func (s3b *S3Backend) GetObjectMetadata(ctx context.Context, key string) (map[string]interface{}, error) {
	fullKey := s3b.buildKey(key)

	input := &s3.HeadObjectInput{
		Bucket: aws.String(s3b.bucket),
		Key:    aws.String(fullKey),
	}

	result, err := s3b.client.HeadObject(ctx, input)
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchKey") {
			return nil, storage.ErrKeyNotFound
		}
		return nil, fmt.Errorf("failed to get object metadata from S3 s3://%s/%s: %w", s3b.bucket, fullKey, err)
	}

	metadata := make(map[string]interface{})

	if result.ContentLength != nil {
		metadata["ContentLength"] = *result.ContentLength
	}
	if result.ContentType != nil {
		metadata["ContentType"] = *result.ContentType
	}
	if result.ETag != nil {
		metadata["ETag"] = *result.ETag
	}
	if result.LastModified != nil {
		metadata["LastModified"] = *result.LastModified
	}
	if result.StorageClass != "" {
		metadata["StorageClass"] = string(result.StorageClass)
	}

	// Add custom metadata
	if result.Metadata != nil {
		metadata["UserMetadata"] = result.Metadata
	}

	return metadata, nil
}
