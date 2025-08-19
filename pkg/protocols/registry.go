package protocols

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"sync"

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
		return fmt.Errorf("protocol handler for scheme %s already registered", scheme)
	}

	pr.protocols[scheme] = handler
	return nil
}

// Unregister removes a protocol handler from the registry
func (pr *ProtocolRegistry) Unregister(scheme string) error {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if _, exists := pr.protocols[scheme]; !exists {
		return fmt.Errorf("protocol handler for scheme %s not found", scheme)
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
		return nil, fmt.Errorf("invalid URL: %w", err)
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	handler, exists := pr.protocols[scheme]
	if !exists {
		return nil, fmt.Errorf("no handler registered for scheme: %s", scheme)
	}

	if !handler.CanHandle(urlStr) {
		return nil, fmt.Errorf("handler for scheme %s cannot handle URL: %s", scheme, urlStr)
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
type HTTPHandler struct{}

func (h *HTTPHandler) Scheme() string {
	return "http"
}

func (h *HTTPHandler) CanHandle(url string) bool {
	return strings.HasPrefix(strings.ToLower(url), "http://") ||
		strings.HasPrefix(strings.ToLower(url), "https://")
}

func (h *HTTPHandler) Download(ctx context.Context, url string, options *types.DownloadOptions) (*types.DownloadStats, error) {
	// TODO: Implement HTTP download logic
	return nil, fmt.Errorf("HTTP download not yet implemented")
}

// FTPHandler handles FTP and FTPS protocols
type FTPHandler struct{}

func (f *FTPHandler) Scheme() string {
	return "ftp"
}

func (f *FTPHandler) CanHandle(url string) bool {
	return strings.HasPrefix(strings.ToLower(url), "ftp://") ||
		strings.HasPrefix(strings.ToLower(url), "ftps://")
}

func (f *FTPHandler) Download(ctx context.Context, url string, options *types.DownloadOptions) (*types.DownloadStats, error) {
	// TODO: Implement FTP download logic
	return nil, fmt.Errorf("FTP download not yet implemented")
}

// S3Handler handles Amazon S3 protocol
type S3Handler struct{}

func (s *S3Handler) Scheme() string {
	return "s3"
}

func (s *S3Handler) CanHandle(url string) bool {
	return strings.HasPrefix(strings.ToLower(url), "s3://")
}

func (s *S3Handler) Download(ctx context.Context, url string, options *types.DownloadOptions) (*types.DownloadStats, error) {
	// TODO: Implement S3 download logic
	return nil, fmt.Errorf("S3 download not yet implemented")
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
	return nil, fmt.Errorf("torrent download not yet implemented")
}
