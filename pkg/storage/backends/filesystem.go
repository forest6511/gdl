package backends

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/forest6511/gdl/pkg/storage"
)

// isPathUnder checks if childPath is under parentPath in a cross-platform way
func isPathUnder(parentPath, childPath string) bool {
	// Get absolute paths
	absParent, err := filepath.Abs(parentPath)
	if err != nil {
		return false
	}
	absChild, err := filepath.Abs(childPath)
	if err != nil {
		return false
	}

	// Clean the paths
	absParent = filepath.Clean(absParent)
	absChild = filepath.Clean(absChild)

	// On Windows, paths are case-insensitive
	if runtime.GOOS == "windows" {
		absParent = strings.ToLower(absParent)
		absChild = strings.ToLower(absChild)
	}

	// Check if child is under parent
	rel, err := filepath.Rel(absParent, absChild)
	if err != nil {
		return false
	}

	// If the relative path starts with ".." or is ".", then child is not under parent
	return !strings.HasPrefix(rel, "..") && rel != "."
}

// FileSystemBackend implements storage using the local file system
type FileSystemBackend struct {
	basePath string
}

// NewFileSystemBackend creates a new file system storage backend
func NewFileSystemBackend() *FileSystemBackend {
	return &FileSystemBackend{}
}

// Init initializes the file system backend with configuration
func (fs *FileSystemBackend) Init(config map[string]interface{}) error {
	basePath, ok := config["basePath"].(string)
	if !ok || basePath == "" {
		return fmt.Errorf("basePath is required for filesystem backend")
	}

	// Expand tilde to home directory (Unix-style only)
	if strings.HasPrefix(basePath, "~/") && runtime.GOOS != "windows" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get user home directory: %w", err)
		}
		basePath = filepath.Join(homeDir, basePath[2:])
	}

	// Convert to absolute path
	absPath, err := filepath.Abs(basePath)
	if err != nil {
		return fmt.Errorf("failed to get absolute path: %w", err)
	}

	fs.basePath = absPath

	// Create base directory if it doesn't exist
	if err := os.MkdirAll(fs.basePath, 0750); err != nil {
		return fmt.Errorf("failed to create base directory %s: %w", fs.basePath, err)
	}

	return nil
}

// Save stores data to the file system at the specified key/path
func (fs *FileSystemBackend) Save(ctx context.Context, key string, data io.Reader) error {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}

	if fs.basePath == "" {
		return storage.ErrBackendNotReady
	}

	filePath := fs.keyToPath(key)

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dir, err)
	}

	// Validate and sanitize the file path to prevent directory traversal
	cleanPath := filepath.Clean(filePath)
	if !isPathUnder(fs.basePath, cleanPath) {
		return fmt.Errorf("path outside base directory not allowed: %s", filePath)
	}

	// Create or truncate the file
	file, err := os.Create(cleanPath) // #nosec G304 - path is validated and sanitized above
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", cleanPath, err)
	}
	defer func() {
		if err := file.Close(); err != nil {
			fmt.Printf("Warning: failed to close file: %v\n", err)
		}
	}()

	// Copy data to file with context cancellation support
	done := make(chan error, 1)
	go func() {
		_, err := io.Copy(file, data)
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil {
			// Clean up partial file on error
			if removeErr := os.Remove(cleanPath); removeErr != nil {
				fmt.Printf("Warning: failed to cleanup partial file %s: %v\n", cleanPath, removeErr)
			}
			return fmt.Errorf("failed to save data to %s: %w", cleanPath, err)
		}
		return nil
	case <-ctx.Done():
		// Clean up partial file on cancellation
		if removeErr := os.Remove(cleanPath); removeErr != nil {
			fmt.Printf("Warning: failed to cleanup partial file %s: %v\n", cleanPath, removeErr)
		}
		return fmt.Errorf("save operation cancelled: %w", ctx.Err())
	}
}

// Load retrieves data from the file system for the given key/path
func (fs *FileSystemBackend) Load(ctx context.Context, key string) (io.ReadCloser, error) {
	// Check for context cancellation
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	if fs.basePath == "" {
		return nil, storage.ErrBackendNotReady
	}

	filePath := fs.keyToPath(key)

	// Validate and sanitize the file path to prevent directory traversal
	cleanPath := filepath.Clean(filePath)
	if !isPathUnder(fs.basePath, cleanPath) {
		return nil, fmt.Errorf("path outside base directory not allowed: %s", filePath)
	}

	// Check if file exists
	if _, err := os.Stat(cleanPath); os.IsNotExist(err) {
		return nil, storage.ErrKeyNotFound
	}

	// Open the file
	file, err := os.Open(cleanPath) // #nosec G304 - path is validated and sanitized above
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", cleanPath, err)
	}

	// Return a reader that respects context cancellation
	return &contextAwareReader{
		ReadCloser: file,
		ctx:        ctx,
	}, nil
}

// Delete removes data from the file system for the given key/path
func (fs *FileSystemBackend) Delete(ctx context.Context, key string) error {
	if fs.basePath == "" {
		return storage.ErrBackendNotReady
	}

	filePath := fs.keyToPath(key)

	// Check if file exists
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return storage.ErrKeyNotFound
	}

	// Remove the file
	if err := os.Remove(filePath); err != nil {
		return fmt.Errorf("failed to delete file %s: %w", filePath, err)
	}

	// Try to remove empty parent directories
	fs.cleanupEmptyDirs(filepath.Dir(filePath))

	return nil
}

// Exists checks if data exists at the given key/path
func (fs *FileSystemBackend) Exists(ctx context.Context, key string) (bool, error) {
	if fs.basePath == "" {
		return false, storage.ErrBackendNotReady
	}

	filePath := fs.keyToPath(key)

	_, err := os.Stat(filePath)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("failed to check file existence %s: %w", filePath, err)
}

// List returns a list of keys with the given prefix
func (fs *FileSystemBackend) List(ctx context.Context, prefix string) ([]string, error) {
	if fs.basePath == "" {
		return nil, storage.ErrBackendNotReady
	}

	var keys []string

	// Walk through the file system starting from the base path
	err := filepath.Walk(fs.basePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Convert file path back to key
		key := fs.pathToKey(path)

		// Check if key matches prefix
		if prefix == "" || strings.HasPrefix(key, prefix) {
			keys = append(keys, key)
		}

		// Check for context cancellation
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to list files: %w", err)
	}

	return keys, nil
}

// Close cleans up resources (no-op for file system)
func (fs *FileSystemBackend) Close() error {
	return nil
}

// keyToPath converts a storage key to a file system path
func (fs *FileSystemBackend) keyToPath(key string) string {
	// Sanitize the key to prevent directory traversal
	cleanKey := filepath.Clean(key)
	cleanKey = strings.TrimPrefix(cleanKey, "/")

	return filepath.Join(fs.basePath, cleanKey)
}

// pathToKey converts a file system path back to a storage key
func (fs *FileSystemBackend) pathToKey(path string) string {
	relPath, err := filepath.Rel(fs.basePath, path)
	if err != nil {
		return path // fallback to full path
	}

	// Convert to forward slashes for consistency
	return filepath.ToSlash(relPath)
}

// cleanupEmptyDirs removes empty parent directories up to the base path
func (fs *FileSystemBackend) cleanupEmptyDirs(dir string) {
	// Don't remove the base path itself
	if dir == fs.basePath || !isPathUnder(fs.basePath, dir) {
		return
	}

	// Try to remove the directory (will fail if not empty)
	if err := os.Remove(dir); err == nil {
		// If successful, try the parent directory
		fs.cleanupEmptyDirs(filepath.Dir(dir))
	}
}

// contextAwareReader wraps a ReadCloser to respect context cancellation
type contextAwareReader struct {
	io.ReadCloser
	ctx context.Context
}

func (r *contextAwareReader) Read(p []byte) (n int, err error) {
	// Check if context is cancelled before reading
	select {
	case <-r.ctx.Done():
		return 0, r.ctx.Err()
	default:
	}

	return r.ReadCloser.Read(p)
}
