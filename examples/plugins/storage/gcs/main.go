package main

import (
	"context"
	"fmt"
	"io"
	"strings"

	"cloud.google.com/go/storage"
	"google.golang.org/api/option"

	"github.com/forest6511/gdl/pkg/plugin"
)

// GCSPlugin implements storage using Google Cloud Storage
type GCSPlugin struct {
	bucket       string
	client       *storage.Client
	prefix       string
	projectID    string
	keyFile      string
	useEmulator  bool
	emulatorHost string
}

// NewGCSPlugin creates a new Google Cloud Storage plugin
func NewGCSPlugin() *GCSPlugin {
	return &GCSPlugin{
		useEmulator:  false,
		emulatorHost: "localhost:8080",
	}
}

// Name returns the plugin name
func (p *GCSPlugin) Name() string {
	return "gcs-storage"
}

// Version returns the plugin version
func (p *GCSPlugin) Version() string {
	return "1.0.0"
}

// Init initializes the GCS plugin with configuration
func (p *GCSPlugin) Init(config map[string]interface{}) error {
	// Required: bucket name
	bucket, ok := config["bucket"].(string)
	if !ok || bucket == "" {
		return fmt.Errorf("bucket name is required")
	}
	p.bucket = bucket

	// Optional: key prefix for all objects
	if prefix, ok := config["prefix"].(string); ok {
		p.prefix = strings.TrimSuffix(prefix, "/")
	}

	// Optional: project ID
	if projectID, ok := config["project_id"].(string); ok {
		p.projectID = projectID
	}

	// Optional: service account key file path
	if keyFile, ok := config["key_file"].(string); ok {
		p.keyFile = keyFile
	}

	// Optional: use emulator for testing
	if useEmulator, ok := config["use_emulator"].(bool); ok {
		p.useEmulator = useEmulator
	}

	// Optional: emulator host
	if emulatorHost, ok := config["emulator_host"].(string); ok {
		p.emulatorHost = emulatorHost
	}

	// Initialize the GCS client
	if err := p.initClient(); err != nil {
		return fmt.Errorf("failed to initialize GCS client: %w", err)
	}

	return nil
}

// Close cleans up the plugin resources
func (p *GCSPlugin) Close() error {
	if p.client != nil {
		return p.client.Close()
	}
	return nil
}

// ValidateAccess validates security access for operations
func (p *GCSPlugin) ValidateAccess(operation, resource string) error {
	// Allow basic storage operations
	allowedOps := []string{"store", "retrieve", "delete", "exists", "list", "upload", "download"}
	for _, op := range allowedOps {
		if operation == op {
			return nil
		}
	}
	return fmt.Errorf("operation %s not allowed for GCS storage plugin", operation)
}

// initClient initializes the Google Cloud Storage client
func (p *GCSPlugin) initClient() error {
	ctx := context.Background()

	var opts []option.ClientOption

	// Use emulator if configured
	if p.useEmulator {
		opts = append(opts, option.WithEndpoint(fmt.Sprintf("http://%s/storage/v1/", p.emulatorHost)))
		opts = append(opts, option.WithoutAuthentication())
	} else {
		// Use service account key file if provided
		if p.keyFile != "" {
			opts = append(opts, option.WithCredentialsFile(p.keyFile))
		}
		// Note: If no key file is provided, the client will use default credentials
		// (e.g., from environment variables, metadata service, etc.)
	}

	client, err := storage.NewClient(ctx, opts...)
	if err != nil {
		return fmt.Errorf("failed to create GCS client: %w", err)
	}

	p.client = client
	return nil
}

// Store saves data to Google Cloud Storage
func (p *GCSPlugin) Store(ctx context.Context, data []byte, key string) error {
	if p.client == nil {
		return fmt.Errorf("GCS client not initialized")
	}

	fullKey := p.buildKey(key)

	// Get object handle
	obj := p.client.Bucket(p.bucket).Object(fullKey)

	// Create a writer
	writer := obj.NewWriter(ctx)
	defer func() {
		if err := writer.Close(); err != nil {
			fmt.Printf("Warning: failed to close GCS writer: %v\n", err)
		}
	}()

	// Set content type if it can be inferred
	if contentType := p.inferContentType(key); contentType != "" {
		writer.ContentType = contentType
	}

	// Write data
	_, err := writer.Write(data)
	if err != nil {
		return fmt.Errorf("failed to write data to GCS object %s: %w", fullKey, err)
	}

	return nil
}

// Retrieve loads data from Google Cloud Storage
func (p *GCSPlugin) Retrieve(ctx context.Context, key string) ([]byte, error) {
	if p.client == nil {
		return nil, fmt.Errorf("GCS client not initialized")
	}

	fullKey := p.buildKey(key)

	// Get object handle
	obj := p.client.Bucket(p.bucket).Object(fullKey)

	// Create a reader
	reader, err := obj.NewReader(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, fmt.Errorf("object not found: %s", fullKey)
		}
		return nil, fmt.Errorf("failed to create reader for GCS object %s: %w", fullKey, err)
	}
	defer func() {
		if err := reader.Close(); err != nil {
			fmt.Printf("Warning: failed to close GCS reader: %v\n", err)
		}
	}()

	// Read all data
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read data from GCS object %s: %w", fullKey, err)
	}

	return data, nil
}

// Delete removes an object from Google Cloud Storage
func (p *GCSPlugin) Delete(ctx context.Context, key string) error {
	if p.client == nil {
		return fmt.Errorf("GCS client not initialized")
	}

	fullKey := p.buildKey(key)

	// Get object handle
	obj := p.client.Bucket(p.bucket).Object(fullKey)

	// Delete the object
	err := obj.Delete(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return fmt.Errorf("object not found: %s", fullKey)
		}
		return fmt.Errorf("failed to delete GCS object %s: %w", fullKey, err)
	}

	return nil
}

// Exists checks if an object exists in Google Cloud Storage
func (p *GCSPlugin) Exists(ctx context.Context, key string) (bool, error) {
	if p.client == nil {
		return false, fmt.Errorf("GCS client not initialized")
	}

	fullKey := p.buildKey(key)

	// Get object handle
	obj := p.client.Bucket(p.bucket).Object(fullKey)

	// Get object attributes (minimal metadata)
	_, err := obj.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return false, nil
		}
		return false, fmt.Errorf("failed to check existence of GCS object %s: %w", fullKey, err)
	}

	return true, nil
}

// List returns objects with the given prefix
func (p *GCSPlugin) List(ctx context.Context, prefix string) ([]string, error) {
	if p.client == nil {
		return nil, fmt.Errorf("GCS client not initialized")
	}

	fullPrefix := p.buildKey(prefix)

	// Create query
	query := &storage.Query{
		Prefix: fullPrefix,
	}

	// List objects
	it := p.client.Bucket(p.bucket).Objects(ctx, query)

	var keys []string
	for {
		attrs, err := it.Next()
		if err == storage.ErrObjectNotExist {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("failed to list GCS objects: %w", err)
		}

		// Remove prefix to get original key
		key := p.stripPrefix(attrs.Name)
		keys = append(keys, key)
	}

	return keys, nil
}

// buildKey constructs the full GCS object key including any configured prefix
func (p *GCSPlugin) buildKey(key string) string {
	if p.prefix == "" {
		return key
	}
	return p.prefix + "/" + strings.TrimPrefix(key, "/")
}

// stripPrefix removes the configured prefix from a GCS object name to get the original key
func (p *GCSPlugin) stripPrefix(objectName string) string {
	if p.prefix == "" {
		return objectName
	}

	prefixWithSlash := p.prefix + "/"
	if strings.HasPrefix(objectName, prefixWithSlash) {
		return strings.TrimPrefix(objectName, prefixWithSlash)
	}

	return objectName
}

// inferContentType tries to infer content type from the key/filename
func (p *GCSPlugin) inferContentType(key string) string {
	ext := strings.ToLower(key)

	if strings.HasSuffix(ext, ".jpg") || strings.HasSuffix(ext, ".jpeg") {
		return "image/jpeg"
	}
	if strings.HasSuffix(ext, ".png") {
		return "image/png"
	}
	if strings.HasSuffix(ext, ".gif") {
		return "image/gif"
	}
	if strings.HasSuffix(ext, ".pdf") {
		return "application/pdf"
	}
	if strings.HasSuffix(ext, ".txt") {
		return "text/plain"
	}
	if strings.HasSuffix(ext, ".json") {
		return "application/json"
	}
	if strings.HasSuffix(ext, ".xml") {
		return "application/xml"
	}
	if strings.HasSuffix(ext, ".html") {
		return "text/html"
	}
	if strings.HasSuffix(ext, ".css") {
		return "text/css"
	}
	if strings.HasSuffix(ext, ".js") {
		return "application/javascript"
	}

	return "" // Let GCS infer
}

// GetObjectInfo returns metadata about an object
func (p *GCSPlugin) GetObjectInfo(ctx context.Context, key string) (map[string]interface{}, error) {
	if p.client == nil {
		return nil, fmt.Errorf("GCS client not initialized")
	}

	fullKey := p.buildKey(key)

	// Get object handle
	obj := p.client.Bucket(p.bucket).Object(fullKey)

	// Get object attributes
	attrs, err := obj.Attrs(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return nil, fmt.Errorf("object not found: %s", fullKey)
		}
		return nil, fmt.Errorf("failed to get attributes for GCS object %s: %w", fullKey, err)
	}

	info := map[string]interface{}{
		"name":         attrs.Name,
		"size":         attrs.Size,
		"content_type": attrs.ContentType,
		"created":      attrs.Created,
		"updated":      attrs.Updated,
		"md5":          fmt.Sprintf("%x", attrs.MD5),
		"crc32c":       attrs.CRC32C,
		"generation":   attrs.Generation,
	}

	if attrs.Metadata != nil {
		info["metadata"] = attrs.Metadata
	}

	return info, nil
}

// SetObjectMetadata sets custom metadata for an object
func (p *GCSPlugin) SetObjectMetadata(ctx context.Context, key string, metadata map[string]string) error {
	if p.client == nil {
		return fmt.Errorf("GCS client not initialized")
	}

	fullKey := p.buildKey(key)

	// Get object handle
	obj := p.client.Bucket(p.bucket).Object(fullKey)

	// Update metadata
	_, err := obj.Update(ctx, storage.ObjectAttrsToUpdate{
		Metadata: metadata,
	})
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return fmt.Errorf("object not found: %s", fullKey)
		}
		return fmt.Errorf("failed to update metadata for GCS object %s: %w", fullKey, err)
	}

	return nil
}

// CopyObject copies an object within GCS
func (p *GCSPlugin) CopyObject(ctx context.Context, srcKey, destKey string) error {
	if p.client == nil {
		return fmt.Errorf("GCS client not initialized")
	}

	srcFullKey := p.buildKey(srcKey)
	destFullKey := p.buildKey(destKey)

	// Get source and destination object handles
	srcObj := p.client.Bucket(p.bucket).Object(srcFullKey)
	destObj := p.client.Bucket(p.bucket).Object(destFullKey)

	// Copy the object
	_, err := destObj.CopierFrom(srcObj).Run(ctx)
	if err != nil {
		if err == storage.ErrObjectNotExist {
			return fmt.Errorf("source object not found: %s", srcFullKey)
		}
		return fmt.Errorf("failed to copy GCS object from %s to %s: %w", srcFullKey, destFullKey, err)
	}

	return nil
}

// Plugin variable to be loaded by the plugin system
var Plugin plugin.StoragePlugin = &GCSPlugin{}

func main() {
	// This is a plugin, so main() is not used when loaded as a shared library
	// But it can be useful for testing the plugin standalone
	fmt.Println("Google Cloud Storage Plugin")
	fmt.Printf("Name: %s\n", Plugin.Name())
	fmt.Printf("Version: %s\n", Plugin.Version())
}
