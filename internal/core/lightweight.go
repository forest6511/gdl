package core

import (
	"context"
	"io"
	"net/http"
	"time"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
)

// LightweightDownloader is an optimized downloader for small files
// with minimal overhead and reduced memory usage
type LightweightDownloader struct {
	client *http.Client
}

// NewLightweightDownloader creates a new lightweight downloader optimized for small files
func NewLightweightDownloader() *LightweightDownloader {
	return &LightweightDownloader{
		client: &http.Client{
			Transport: &http.Transport{
				MaxIdleConns:        1,
				MaxIdleConnsPerHost: 1,
				MaxConnsPerHost:     1,
				IdleConnTimeout:     10 * time.Second,
				DisableCompression:  false,
				DisableKeepAlives:   true, // Disable keep-alive for single small downloads
			},
			Timeout: 30 * time.Second,
		},
	}
}

// Download performs a lightweight download optimized for small files
func (ld *LightweightDownloader) Download(ctx context.Context, url string, writer io.Writer) (int64, error) {
	return ld.DownloadWithOptions(ctx, url, writer, "")
}

// DownloadWithOptions performs a lightweight download with custom user agent
func (ld *LightweightDownloader) DownloadWithOptions(ctx context.Context, url string, writer io.Writer, userAgent string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeInvalidURL,
			"failed to create request", url)
	}

	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	resp, err := ld.client.Do(req)
	if err != nil {
		return 0, gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeNetworkError,
			"failed to execute request", url)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, gdlerrors.FromHTTPStatus(resp.StatusCode, url)
	}

	// Use a small buffer for small files to reduce memory overhead
	buf := make([]byte, 8*1024) // 8KB buffer for small files
	written, err := io.CopyBuffer(writer, resp.Body, buf)
	if err != nil {
		return written, gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeNetworkError,
			"failed to write response", url)
	}

	return written, nil
}

// DownloadWithProgress performs a lightweight download with progress callback
func (ld *LightweightDownloader) DownloadWithProgress(ctx context.Context, url string, writer io.Writer, progressFunc func(downloaded, total int64)) (int64, error) {
	return ld.DownloadWithProgressAndOptions(ctx, url, writer, progressFunc, "")
}

// DownloadWithProgressAndOptions performs a lightweight download with progress callback and user agent
func (ld *LightweightDownloader) DownloadWithProgressAndOptions(ctx context.Context, url string, writer io.Writer, progressFunc func(downloaded, total int64), userAgent string) (int64, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeInvalidURL,
			"failed to create request", url)
	}

	if userAgent != "" {
		req.Header.Set("User-Agent", userAgent)
	}

	resp, err := ld.client.Do(req)
	if err != nil {
		return 0, gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeNetworkError,
			"failed to execute request", url)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return 0, gdlerrors.FromHTTPStatus(resp.StatusCode, url)
	}

	contentLength := resp.ContentLength
	if contentLength < 0 {
		contentLength = 0
	}

	// Progress tracking wrapper
	pr := &progressReader{
		reader:       resp.Body,
		total:        contentLength,
		progressFunc: progressFunc,
	}

	// Use a small buffer for small files
	buf := make([]byte, 8*1024) // 8KB buffer
	written, err := io.CopyBuffer(writer, pr, buf)
	if err != nil {
		return written, gdlerrors.WrapErrorWithURL(err, gdlerrors.CodeNetworkError,
			"failed to write response", url)
	}

	return written, nil
}

// progressReader wraps an io.Reader to track reading progress
type progressReader struct {
	reader       io.Reader
	total        int64
	downloaded   int64
	progressFunc func(downloaded, total int64)
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.reader.Read(p)
	if n > 0 {
		pr.downloaded += int64(n)
		if pr.progressFunc != nil {
			pr.progressFunc(pr.downloaded, pr.total)
		}
	}
	return n, err
}

// shouldUseLightweight determines if lightweight mode should be used based on content length
func shouldUseLightweight(contentLength int64) bool {
	// Use lightweight mode for files smaller than 1MB
	return contentLength > 0 && contentLength < 1024*1024
}
