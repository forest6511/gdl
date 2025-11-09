package protocols

import (
	"context"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/forest6511/gdl/internal/core"
	ftpProtocol "github.com/forest6511/gdl/internal/protocols/ftp"
	s3Protocol "github.com/forest6511/gdl/internal/protocols/s3"
	gdlerrors "github.com/forest6511/gdl/pkg/errors"
	"github.com/forest6511/gdl/pkg/types"
)

// ProtocolRegistry manages and dispatches to different protocol handlers
type ProtocolRegistry struct {
	protocols map[string]ProtocolHandler
	mu        sync.RWMutex
}

// ProtocolHandler defines the interface for protocol-specific download handlers
type ProtocolHandler interface {
	Scheme() string
	CanHandle(url string) bool
	Download(ctx context.Context, url string, options *types.DownloadOptions) (*types.DownloadStats, error)
}

// NewProtocolRegistry creates a new protocol registry
func NewProtocolRegistry() *ProtocolRegistry {
	registry := &ProtocolRegistry{
		protocols: make(map[string]ProtocolHandler),
	}

	// Register built-in protocol handlers
	registry.registerBuiltinHandlers()

	return registry
}

// Register adds a protocol handler to the registry
func (pr *ProtocolRegistry) Register(handler ProtocolHandler) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	scheme := handler.Scheme()
	if _, exists := pr.protocols[scheme]; exists {
		return gdlerrors.NewValidationError("protocol_handler", "protocol handler for scheme "+scheme+" already registered")
	}

	pr.protocols[scheme] = handler
	return nil
}

// Unregister removes a protocol handler from the registry
func (pr *ProtocolRegistry) Unregister(scheme string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if _, exists := pr.protocols[scheme]; !exists {
		return gdlerrors.NewValidationError("protocol_handler", "protocol handler for scheme "+scheme+" not found")
	}

	delete(pr.protocols, scheme)
	return nil
}

// GetHandler returns the appropriate handler for a given URL
func (pr *ProtocolRegistry) GetHandler(urlStr string) (ProtocolHandler, error) {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeInvalidURL, "invalid URL", urlStr)
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	handler, exists := pr.protocols[scheme]
	if !exists {
		return nil, gdlerrors.NewValidationError("protocol_handler", "no handler registered for scheme: "+scheme)
	}

	if !handler.CanHandle(urlStr) {
		return nil, gdlerrors.NewValidationError("protocol_handler", "handler for scheme "+scheme+" cannot handle URL")
	}

	return handler, nil
}

// Download performs a download using the appropriate protocol handler
func (pr *ProtocolRegistry) Download(ctx context.Context, urlStr string, options *types.DownloadOptions) (*types.DownloadStats, error) {
	handler, err := pr.GetHandler(urlStr)
	if err != nil {
		return nil, err
	}

	return handler.Download(ctx, urlStr, options)
}

// ListProtocols returns a list of all registered protocol schemes
func (pr *ProtocolRegistry) ListProtocols() []string {
	pr.mu.RLock()
	defer pr.mu.RUnlock()

	schemes := make([]string, 0, len(pr.protocols))
	for scheme := range pr.protocols {
		schemes = append(schemes, scheme)
	}

	return schemes
}

// registerBuiltinHandlers registers the built-in protocol handlers
func (pr *ProtocolRegistry) registerBuiltinHandlers() {
	pr.protocols["http"] = &HTTPHandler{}
	pr.protocols["https"] = &HTTPHandler{}
	pr.protocols["ftp"] = &FTPHandler{}
	pr.protocols["ftps"] = &FTPHandler{}
	pr.protocols["s3"] = &S3Handler{}
	pr.protocols["torrent"] = &TorrentHandler{}
}

// Built-in protocol handlers

// HTTPHandler handles HTTP and HTTPS protocols
type HTTPHandler struct {
	downloader *core.Downloader
}

func (h *HTTPHandler) Scheme() string {
	return "http"
}

func (h *HTTPHandler) CanHandle(url string) bool {
	return strings.HasPrefix(strings.ToLower(url), "http://") ||
		strings.HasPrefix(strings.ToLower(url), "https://")
}

func (h *HTTPHandler) Download(ctx context.Context, url string, options *types.DownloadOptions) (*types.DownloadStats, error) {
	if h.downloader == nil {
		h.downloader = core.NewDownloader()
	}

	// Determine destination from options
	destination := options.Destination
	if destination == "" {
		// Extract filename from URL
		parsedURL, err := parseURL(url)
		if err != nil {
			return nil, gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeInvalidURL, "failed to parse URL", url)
		}
		destination = extractFilenameFromURL(parsedURL)
	}

	return h.downloader.Download(ctx, url, destination, options)
}

// parseURL parses a URL string
func parseURL(urlStr string) (*url.URL, error) {
	return url.Parse(urlStr)
}

// extractFilenameFromURL extracts filename from URL
func extractFilenameFromURL(u *url.URL) string {
	path := u.Path
	if path == "" || path == "/" {
		return "download"
	}
	segments := strings.Split(strings.Trim(path, "/"), "/")
	if len(segments) > 0 {
		return segments[len(segments)-1]
	}
	return "download"
}

// FTPHandler handles FTP and FTPS protocols.
// Note: Full success path coverage requires FTP server integration tests.
// Unit tests cover error paths and basic functionality.
type FTPHandler struct {
	downloader *ftpProtocol.FTPDownloader
}

func (f *FTPHandler) Scheme() string {
	return "ftp"
}

func (f *FTPHandler) CanHandle(url string) bool {
	return strings.HasPrefix(strings.ToLower(url), "ftp://") ||
		strings.HasPrefix(strings.ToLower(url), "ftps://")
}

func (f *FTPHandler) Download(ctx context.Context, url string, options *types.DownloadOptions) (*types.DownloadStats, error) {
	startTime := time.Now()

	// Initialize FTP downloader if needed
	if f.downloader == nil {
		f.downloader = ftpProtocol.NewFTPDownloader(nil) // Use default config
	}

	// Determine destination from options
	destination := options.Destination
	if destination == "" {
		parsedURL, err := parseURL(url)
		if err != nil {
			return nil, gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeInvalidURL, "failed to parse URL", url)
		}
		destination = extractFilenameFromURL(parsedURL)
	}

	// Create destination file
	// #nosec G304 -- destination is provided by user as download target, which is expected behavior
	file, err := os.Create(destination)
	if err != nil {
		return nil, gdlerrors.WrapError(err, gdlerrors.CodeInvalidPath, "failed to create destination file")
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = gdlerrors.WrapError(cerr, gdlerrors.CodeStorageError, "failed to close file")
		}
	}()

	// Download the file
	err = f.downloader.Download(ctx, url, file)

	stats := &types.DownloadStats{
		URL:       url,
		Filename:  destination,
		StartTime: startTime,
		EndTime:   time.Now(),
	}
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	if err != nil {
		stats.Success = false
		stats.Error = err
		return stats, gdlerrors.WrapError(err, gdlerrors.CodeNetworkError, "FTP download failed")
	}

	// Get file size
	fileInfo, err := file.Stat()
	if err == nil {
		stats.BytesDownloaded = fileInfo.Size()
		stats.TotalSize = fileInfo.Size()
	}

	stats.Success = true
	if stats.Duration > 0 {
		stats.AverageSpeed = int64(float64(stats.BytesDownloaded) / stats.Duration.Seconds())
	}

	return stats, nil
}

// S3Handler handles Amazon S3 protocol.
// Note: Full success path coverage requires S3 integration tests or mocking AWS SDK.
// Unit tests cover error paths and basic functionality.
type S3Handler struct {
	downloader *s3Protocol.S3Downloader
}

func (s *S3Handler) Scheme() string {
	return "s3"
}

func (s *S3Handler) CanHandle(url string) bool {
	return strings.HasPrefix(strings.ToLower(url), "s3://")
}

func (s *S3Handler) Download(ctx context.Context, url string, options *types.DownloadOptions) (*types.DownloadStats, error) {
	startTime := time.Now()

	// Initialize S3 downloader if needed
	if s.downloader == nil {
		var err error
		s.downloader, err = s3Protocol.NewS3Downloader(nil) // Use default config
		if err != nil {
			return nil, gdlerrors.WrapError(err, gdlerrors.CodeConfigError, "failed to initialize S3 downloader")
		}
	}

	// Determine destination from options
	destination := options.Destination
	if destination == "" {
		parsedURL, err := parseURL(url)
		if err != nil {
			return nil, gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeInvalidURL, "failed to parse URL", url)
		}
		destination = extractFilenameFromURL(parsedURL)
	}

	// Create destination file
	// #nosec G304 -- destination is provided by user as download target, which is expected behavior
	file, err := os.Create(destination)
	if err != nil {
		return nil, gdlerrors.WrapError(err, gdlerrors.CodeInvalidPath, "failed to create destination file")
	}
	defer func() {
		if cerr := file.Close(); cerr != nil && err == nil {
			err = gdlerrors.WrapError(cerr, gdlerrors.CodeStorageError, "failed to close file")
		}
	}()

	// Download the file
	err = s.downloader.Download(ctx, url, file)

	stats := &types.DownloadStats{
		URL:       url,
		Filename:  destination,
		StartTime: startTime,
		EndTime:   time.Now(),
	}
	stats.Duration = stats.EndTime.Sub(stats.StartTime)

	if err != nil {
		stats.Success = false
		stats.Error = err
		return stats, gdlerrors.WrapError(err, gdlerrors.CodeNetworkError, "S3 download failed")
	}

	// Get file size
	fileInfo, err := file.Stat()
	if err == nil {
		stats.BytesDownloaded = fileInfo.Size()
		stats.TotalSize = fileInfo.Size()
	}

	stats.Success = true
	if stats.Duration > 0 {
		stats.AverageSpeed = int64(float64(stats.BytesDownloaded) / stats.Duration.Seconds())
	}

	return stats, nil
}

// TorrentHandler handles torrent protocol
type TorrentHandler struct{}

func (t *TorrentHandler) Scheme() string {
	return "torrent"
}

func (t *TorrentHandler) CanHandle(url string) bool {
	return strings.HasPrefix(strings.ToLower(url), "magnet:") ||
		strings.HasSuffix(strings.ToLower(url), ".torrent")
}

func (t *TorrentHandler) Download(ctx context.Context, url string, options *types.DownloadOptions) (*types.DownloadStats, error) {
	// TODO: Implement torrent download logic
	return nil, gdlerrors.NewDownloadError(gdlerrors.CodeUnknown, "torrent download not yet implemented")
}
