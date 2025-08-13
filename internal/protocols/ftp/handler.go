package ftp

import (
	"context"
	"fmt"
	"io"
	"net/url"
	"time"

	"github.com/jlaffaye/ftp"
)

// FTPDownloader handles FTP protocol downloads
type FTPDownloader struct {
	client *ftp.ServerConn
	config *Config
}

// Config holds FTP connection configuration
type Config struct {
	Timeout     time.Duration
	DialTimeout time.Duration
	Username    string
	Password    string
	TLSConfig   interface{} // For FTPS support
	PassiveMode bool
	DebugMode   bool
}

// DefaultConfig returns a default FTP configuration
func DefaultConfig() *Config {
	return &Config{
		Timeout:     30 * time.Second,
		DialTimeout: 10 * time.Second,
		Username:    "anonymous",
		Password:    "anonymous@example.com",
		PassiveMode: true,
		DebugMode:   false,
	}
}

// NewFTPDownloader creates a new FTP downloader instance
func NewFTPDownloader(config *Config) *FTPDownloader {
	if config == nil {
		config = DefaultConfig()
	}

	return &FTPDownloader{
		config: config,
	}
}

// Connect establishes a connection to the FTP server
func (f *FTPDownloader) Connect(ctx context.Context, serverURL string) error {
	parsedURL, err := url.Parse(serverURL)
	if err != nil {
		return fmt.Errorf("invalid FTP URL: %w", err)
	}

	// Extract server address and port
	host := parsedURL.Hostname()
	port := parsedURL.Port()
	if port == "" {
		port = "21" // Default FTP port
	}

	server := fmt.Sprintf("%s:%s", host, port)

	// Extract credentials from URL if provided
	username := f.config.Username
	password := f.config.Password
	if parsedURL.User != nil {
		username = parsedURL.User.Username()
		if pwd, set := parsedURL.User.Password(); set {
			password = pwd
		}
	}

	// Establish connection with timeout
	conn, err := ftp.Dial(server, ftp.DialWithTimeout(f.config.DialTimeout))
	if err != nil {
		return fmt.Errorf("failed to connect to FTP server %s: %w", server, err)
	}

	// Authenticate
	err = conn.Login(username, password)
	if err != nil {
		if quitErr := conn.Quit(); quitErr != nil {
			// Log quit error but don't override the main error
			fmt.Printf("Warning: failed to quit FTP connection: %v\n", quitErr)
		}
		return fmt.Errorf("FTP authentication failed for user %s: %w", username, err)
	}

	// Set transfer mode to passive (PASV is enabled by default in modern FTP libraries)
	// Modern FTP libraries typically use passive mode by default, so we skip explicit setting

	f.client = conn
	return nil
}

// Download downloads a file from FTP server and writes it to the provided writer
func (f *FTPDownloader) Download(ctx context.Context, urlStr string, writer io.Writer) error {
	if f.client == nil {
		if err := f.Connect(ctx, urlStr); err != nil {
			return err
		}
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid FTP URL: %w", err)
	}

	// Extract file path from URL
	filePath := parsedURL.Path
	if filePath == "" || filePath == "/" {
		return fmt.Errorf("no file path specified in FTP URL")
	}

	// Create a channel to handle cancellation
	done := make(chan error, 1)

	go func() {
		// Retrieve the file
		response, err := f.client.Retr(filePath)
		if err != nil {
			done <- fmt.Errorf("failed to retrieve file %s: %w", filePath, err)
			return
		}
		defer func() {
			if err := response.Close(); err != nil {
				fmt.Printf("Warning: failed to close FTP response: %v\n", err)
			}
		}()

		// Copy data to writer
		_, err = io.Copy(writer, response)
		if err != nil {
			done <- fmt.Errorf("failed to download file %s: %w", filePath, err)
			return
		}

		done <- nil
	}()

	// Wait for completion or cancellation
	select {
	case err := <-done:
		return err
	case <-ctx.Done():
		return fmt.Errorf("FTP download cancelled: %w", ctx.Err())
	}
}

// GetFileSize retrieves the size of a file on the FTP server
func (f *FTPDownloader) GetFileSize(ctx context.Context, urlStr string) (int64, error) {
	if f.client == nil {
		if err := f.Connect(ctx, urlStr); err != nil {
			return 0, err
		}
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return 0, fmt.Errorf("invalid FTP URL: %w", err)
	}

	filePath := parsedURL.Path
	if filePath == "" || filePath == "/" {
		return 0, fmt.Errorf("no file path specified in FTP URL")
	}

	// Get file size using SIZE command
	size, err := f.client.FileSize(filePath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file size for %s: %w", filePath, err)
	}

	return size, nil
}

// ListFiles lists files in a directory on the FTP server
func (f *FTPDownloader) ListFiles(ctx context.Context, urlStr string) ([]string, error) {
	if f.client == nil {
		if err := f.Connect(ctx, urlStr); err != nil {
			return nil, err
		}
	}

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, fmt.Errorf("invalid FTP URL: %w", err)
	}

	dirPath := parsedURL.Path
	if dirPath == "" {
		dirPath = "/"
	}

	// List files in directory
	entries, err := f.client.List(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to list directory %s: %w", dirPath, err)
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.Type == ftp.EntryTypeFile {
			files = append(files, entry.Name)
		}
	}

	return files, nil
}

// Close closes the FTP connection
func (f *FTPDownloader) Close() error {
	if f.client != nil {
		err := f.client.Quit()
		f.client = nil
		return err
	}
	return nil
}

// IsConnected returns true if the FTP client is connected
func (f *FTPDownloader) IsConnected() bool {
	return f.client != nil
}

// ChangeWorkingDirectory changes the current working directory on the FTP server
func (f *FTPDownloader) ChangeWorkingDirectory(path string) error {
	if f.client == nil {
		return fmt.Errorf("FTP client not connected")
	}

	return f.client.ChangeDir(path)
}

// GetCurrentDirectory returns the current working directory on the FTP server
func (f *FTPDownloader) GetCurrentDirectory() (string, error) {
	if f.client == nil {
		return "", fmt.Errorf("FTP client not connected")
	}

	return f.client.CurrentDir()
}
