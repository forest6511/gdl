package s3

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

// S3Downloader handles AWS S3 protocol downloads
type S3Downloader struct {
	client *s3.Client
	config *Config
}

// Config holds S3 connection configuration
type Config struct {
	Region          string
	AccessKeyID     string
	SecretAccessKey string
	SessionToken    string
	Endpoint        string // For S3-compatible services
	UsePathStyle    bool   // For S3-compatible services
	DisableSSL      bool
	Profile         string // AWS profile name
}

// DefaultConfig returns a default S3 configuration
func DefaultConfig() *Config {
	return &Config{
		Region:       "us-east-1",
		UsePathStyle: false,
		DisableSSL:   false,
	}
}

// NewS3Downloader creates a new S3 downloader instance
func NewS3Downloader(cfg *Config) (*S3Downloader, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	downloader := &S3Downloader{
		config: cfg,
	}

	if err := downloader.initializeClient(); err != nil {
		return nil, fmt.Errorf("failed to initialize S3 client: %w", err)
	}

	return downloader, nil
}

// initializeClient initializes the AWS S3 client
func (s *S3Downloader) initializeClient() error {
	ctx := context.Background()

	var awsConfig aws.Config
	var err error

	// Load AWS configuration
	if s.config.Profile != "" {
		// Load config with specific profile
		awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(s.config.Region),
			config.WithSharedConfigProfile(s.config.Profile),
		)
	} else if s.config.AccessKeyID != "" && s.config.SecretAccessKey != "" {
		// Use provided credentials
		creds := credentials.NewStaticCredentialsProvider(
			s.config.AccessKeyID,
			s.config.SecretAccessKey,
			s.config.SessionToken,
		)
		awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(s.config.Region),
			config.WithCredentialsProvider(creds),
		)
	} else {
		// Load default configuration
		awsConfig, err = config.LoadDefaultConfig(ctx,
			config.WithRegion(s.config.Region),
		)
	}

	if err != nil {
		return fmt.Errorf("failed to load AWS config: %w", err)
	}

	// Create S3 client options
	options := func(o *s3.Options) {
		if s.config.Endpoint != "" {
			o.BaseEndpoint = aws.String(s.config.Endpoint)
		}
		o.UsePathStyle = s.config.UsePathStyle
	}

	s.client = s3.NewFromConfig(awsConfig, options)

	return nil
}

// parseS3URL parses an S3 URL and returns bucket and key
func (s *S3Downloader) parseS3URL(urlStr string) (bucket, key string, err error) {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return "", "", fmt.Errorf("invalid S3 URL: %w", err)
	}

	if parsedURL.Scheme != "s3" {
		return "", "", fmt.Errorf("URL scheme must be 's3', got: %s", parsedURL.Scheme)
	}

	bucket = parsedURL.Host
	key = strings.TrimPrefix(parsedURL.Path, "/")

	if bucket == "" {
		return "", "", fmt.Errorf("bucket name is required in S3 URL")
	}

	if key == "" {
		return "", "", fmt.Errorf("object key is required in S3 URL")
	}

	return bucket, key, nil
}

// Download downloads an object from S3 and writes it to the provided writer
func (s *S3Downloader) Download(ctx context.Context, url string, writer io.Writer) error {
	bucket, key, err := s.parseS3URL(url)
	if err != nil {
		return err
	}

	// Get the object from S3
	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to get object s3://%s/%s: %w", bucket, key, err)
	}
	defer func() {
		if err := result.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close S3 response body: %v\n", err)
		}
	}()

	// Copy the object content to the writer
	_, err = io.Copy(writer, result.Body)
	if err != nil {
		return fmt.Errorf("failed to download object s3://%s/%s: %w", bucket, key, err)
	}

	return nil
}

// GetObjectSize returns the size of an S3 object
func (s *S3Downloader) GetObjectSize(ctx context.Context, url string) (int64, error) {
	bucket, key, err := s.parseS3URL(url)
	if err != nil {
		return 0, err
	}

	// Head the object to get metadata
	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.HeadObject(ctx, input)
	if err != nil {
		return 0, fmt.Errorf("failed to get object metadata s3://%s/%s: %w", bucket, key, err)
	}

	if result.ContentLength == nil {
		return 0, fmt.Errorf("content length not available for object s3://%s/%s", bucket, key)
	}

	return *result.ContentLength, nil
}

// ListObjects lists objects in an S3 bucket with optional prefix
func (s *S3Downloader) ListObjects(ctx context.Context, bucket, prefix string, maxKeys int32) ([]string, error) {
	input := &s3.ListObjectsV2Input{
		Bucket: aws.String(bucket),
	}

	if prefix != "" {
		input.Prefix = aws.String(prefix)
	}

	if maxKeys > 0 {
		input.MaxKeys = aws.Int32(maxKeys)
	}

	result, err := s.client.ListObjectsV2(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to list objects in bucket %s: %w", bucket, err)
	}

	objects := make([]string, 0, len(result.Contents))
	for _, obj := range result.Contents {
		if obj.Key != nil {
			objects = append(objects, *obj.Key)
		}
	}

	return objects, nil
}

// ObjectExists checks if an object exists in S3
func (s *S3Downloader) ObjectExists(ctx context.Context, url string) (bool, error) {
	bucket, key, err := s.parseS3URL(url)
	if err != nil {
		return false, err
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	_, err = s.client.HeadObject(ctx, input)
	if err != nil {
		if strings.Contains(err.Error(), "NotFound") || strings.Contains(err.Error(), "NoSuchKey") {
			return false, nil
		}
		return false, fmt.Errorf("failed to check object existence s3://%s/%s: %w", bucket, key, err)
	}

	return true, nil
}

// GetObjectMetadata returns metadata for an S3 object
func (s *S3Downloader) GetObjectMetadata(ctx context.Context, url string) (map[string]interface{}, error) {
	bucket, key, err := s.parseS3URL(url)
	if err != nil {
		return nil, err
	}

	input := &s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
	}

	result, err := s.client.HeadObject(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get object metadata s3://%s/%s: %w", bucket, key, err)
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

// DownloadRange downloads a specific range of bytes from an S3 object
func (s *S3Downloader) DownloadRange(ctx context.Context, url string, writer io.Writer, start, end int64) error {
	bucket, key, err := s.parseS3URL(url)
	if err != nil {
		return err
	}

	rangeHeader := fmt.Sprintf("bytes=%d-%d", start, end)

	input := &s3.GetObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(key),
		Range:  aws.String(rangeHeader),
	}

	result, err := s.client.GetObject(ctx, input)
	if err != nil {
		return fmt.Errorf("failed to get object range s3://%s/%s [%d-%d]: %w", bucket, key, start, end, err)
	}
	defer func() {
		if err := result.Body.Close(); err != nil {
			fmt.Printf("Warning: failed to close S3 response body: %v\n", err)
		}
	}()

	_, err = io.Copy(writer, result.Body)
	if err != nil {
		return fmt.Errorf("failed to download object range s3://%s/%s [%d-%d]: %w", bucket, key, start, end, err)
	}

	return nil
}

// Close closes the S3 downloader (no-op for S3 client)
func (s *S3Downloader) Close() error {
	// S3 client doesn't need explicit closing
	return nil
}
