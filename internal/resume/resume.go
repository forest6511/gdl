// Package resume provides functionality for resuming interrupted downloads.
package resume

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	gdlerrors "github.com/forest6511/gdl/pkg/errors"
)

// ResumeInfo contains metadata about a partial download that can be resumed.
type ResumeInfo struct {
	// URL is the source URL of the download.
	URL string `json:"url"`

	// FilePath is the local path where the file is being downloaded.
	FilePath string `json:"file_path"`

	// DownloadedBytes is the number of bytes already downloaded.
	DownloadedBytes int64 `json:"downloaded_bytes"`

	// TotalBytes is the total size of the file (-1 if unknown).
	TotalBytes int64 `json:"total_bytes"`

	// ETag is the ETag header from the server for cache validation.
	ETag string `json:"etag,omitempty"`

	// LastModified is the Last-Modified header from the server.
	LastModified time.Time `json:"last_modified,omitempty"`

	// ContentLength is the original Content-Length header value.
	ContentLength int64 `json:"content_length"`

	// Checksum is an MD5 hash of the partial file for integrity verification.
	Checksum string `json:"checksum,omitempty"`

	// CreatedAt is when this resume info was first created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when this resume info was last updated.
	UpdatedAt time.Time `json:"updated_at"`

	// UserAgent is the User-Agent used for the original request.
	UserAgent string `json:"user_agent,omitempty"`

	// AcceptRanges indicates if the server supports range requests.
	AcceptRanges bool `json:"accept_ranges"`
}

// Manager handles saving and loading resume information.
type Manager struct {
	// resumeDir is the directory where resume files are stored.
	resumeDir string
}

// NewManager creates a new resume manager with the specified directory.
// If resumeDir is empty, it uses the current directory.
func NewManager(resumeDir string) *Manager {
	if resumeDir == "" {
		resumeDir = "."
	}

	return &Manager{
		resumeDir: resumeDir,
	}
}

// getResumeFilePath returns the path to the resume file for a given download file.
func (m *Manager) getResumeFilePath(filePath string) string {
	// Create a unique resume file name based on the download file
	basename := filepath.Base(filePath)
	resumeFileName := fmt.Sprintf(".%s.gdl.json", basename)

	return filepath.Join(m.resumeDir, resumeFileName)
}

// Save saves resume information to a .gdl.json file.
func (m *Manager) Save(info *ResumeInfo) error {
	info.UpdatedAt = time.Now()
	if info.CreatedAt.IsZero() {
		info.CreatedAt = info.UpdatedAt
	}

	resumeFilePath := m.getResumeFilePath(info.FilePath)

	// Ensure the resume directory exists
	if err := os.MkdirAll(m.resumeDir, 0o750); err != nil {
		return gdlerrors.NewStorageError("create resume directory", err, m.resumeDir)
	}

	// Marshal the resume info to JSON
	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return gdlerrors.WrapError(err, gdlerrors.CodeStorageError, "marshal resume info")
	}

	// Write to the resume file
	if err := os.WriteFile(resumeFilePath, data, 0o600); err != nil {
		return gdlerrors.NewStorageError("write resume file", err, resumeFilePath)
	}

	return nil
}

// Load loads resume information from a .gdl.json file.
func (m *Manager) Load(filePath string) (*ResumeInfo, error) {
	resumeFilePath := m.getResumeFilePath(filePath)

	// Check if resume file exists
	if _, err := os.Stat(resumeFilePath); os.IsNotExist(err) {
		return nil, nil // No resume file exists
	}

	// Read the resume file
	// #nosec G304 - Resume file path is constructed internally, not from user input
	data, err := os.ReadFile(resumeFilePath)
	if err != nil {
		return nil, gdlerrors.NewStorageError("read resume file", err, resumeFilePath)
	}

	// Unmarshal the JSON data
	var info ResumeInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, gdlerrors.WrapError(err, gdlerrors.CodeStorageError, "unmarshal resume info")
	}

	return &info, nil
}

// Delete removes the resume file for a given download file.
func (m *Manager) Delete(filePath string) error {
	resumeFilePath := m.getResumeFilePath(filePath)

	if err := os.Remove(resumeFilePath); err != nil && !os.IsNotExist(err) {
		return gdlerrors.NewStorageError("delete resume file", err, resumeFilePath)
	}

	return nil
}

// Exists checks if a resume file exists for the given file path.
func (m *Manager) Exists(filePath string) bool {
	resumeFilePath := m.getResumeFilePath(filePath)
	_, err := os.Stat(resumeFilePath)

	return err == nil
}

// ValidatePartialFile validates that a partial file matches the resume information.
func (m *Manager) ValidatePartialFile(info *ResumeInfo) error {
	// Check if the partial file exists
	fileInfo, err := os.Stat(info.FilePath)
	if os.IsNotExist(err) {
		return gdlerrors.NewInvalidPathError(info.FilePath, err)
	}

	if err != nil {
		return gdlerrors.NewStorageError("stat partial file", err, info.FilePath)
	}

	// Check if the file size matches the resume info
	if fileInfo.Size() != info.DownloadedBytes {
		return gdlerrors.NewValidationError(
			"partial file size",
			fmt.Sprintf("size %d does not match resume info %d", fileInfo.Size(), info.DownloadedBytes),
		)
	}

	// Validate checksum if available
	if info.Checksum != "" {
		calculatedChecksum, err := m.calculateFileChecksum(info.FilePath)
		if err != nil {
			return gdlerrors.WrapError(err, gdlerrors.CodeStorageError, "calculate file checksum")
		}

		if calculatedChecksum != info.Checksum {
			return gdlerrors.NewValidationError(
				"file checksum",
				fmt.Sprintf("expected %s, got %s", info.Checksum, calculatedChecksum),
			)
		}
	}

	return nil
}

// CalculateAndSetChecksum calculates the MD5 checksum of the partial file and sets it in the resume info.
func (m *Manager) CalculateAndSetChecksum(info *ResumeInfo) error {
	checksum, err := m.calculateFileChecksum(info.FilePath)
	if err != nil {
		return err
	}

	info.Checksum = checksum

	return nil
}

// calculateFileChecksum calculates the MD5 checksum of a file.
func (m *Manager) calculateFileChecksum(filePath string) (string, error) {
	// #nosec G304 -- filePath is constructed internally from validated download destination
	file, err := os.Open(filePath)
	if err != nil {
		return "", gdlerrors.NewStorageError("open file", err, filePath)
	}
	defer func() { _ = file.Close() }()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", gdlerrors.NewStorageError("calculate checksum", err, filePath)
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// CanResume checks if a download can be resumed based on the resume info and server capabilities.
func (m *Manager) CanResume(info *ResumeInfo) bool {
	// Basic checks
	if info == nil || !info.AcceptRanges || info.DownloadedBytes <= 0 {
		return false
	}

	// Check if total bytes is known and we haven't already downloaded everything
	if info.TotalBytes > 0 && info.DownloadedBytes >= info.TotalBytes {
		return false
	}

	// Validate the partial file
	if err := m.ValidatePartialFile(info); err != nil {
		return false
	}

	return true
}

// CleanupOldResumeFiles removes resume files that are older than the specified duration.
func (m *Manager) CleanupOldResumeFiles(maxAge time.Duration) error {
	pattern := filepath.Join(m.resumeDir, ".*.gdl.json")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return gdlerrors.NewStorageError("find resume files", err, pattern)
	}

	cutoff := time.Now().Add(-maxAge)

	for _, resumeFile := range matches {
		fileInfo, err := os.Stat(resumeFile)
		if err != nil {
			continue // Skip files we can't stat
		}

		if fileInfo.ModTime().Before(cutoff) {
			if err := os.Remove(resumeFile); err != nil {
				// Log but don't fail the entire cleanup
				continue
			}
		}
	}

	return nil
}

// ListResumeFiles returns a list of all resume files in the resume directory.
func (m *Manager) ListResumeFiles() ([]string, error) {
	pattern := filepath.Join(m.resumeDir, ".*.gdl.json")

	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, gdlerrors.NewStorageError("find resume files", err, pattern)
	}

	var resumeFiles []string

	for _, match := range matches {
		// Load each resume file to get the original file path
		// #nosec G304 -- match is from filepath.Glob with controlled pattern, not user input
		data, err := os.ReadFile(match)
		if err != nil {
			continue // Skip unreadable files
		}

		var info ResumeInfo
		if err := json.Unmarshal(data, &info); err != nil {
			continue // Skip invalid files
		}

		resumeFiles = append(resumeFiles, info.FilePath)
	}

	return resumeFiles, nil
}

// GetResumeStats returns statistics about resumable downloads.
func (m *Manager) GetResumeStats() (map[string]interface{}, error) {
	resumeFiles, err := m.ListResumeFiles()
	if err != nil {
		return nil, err
	}

	stats := map[string]interface{}{
		"total_resume_files":  len(resumeFiles),
		"resumable_files":     0,
		"total_partial_bytes": int64(0),
	}

	for _, filePath := range resumeFiles {
		info, err := m.Load(filePath)
		if err != nil {
			continue
		}

		if m.CanResume(info) {
			stats["resumable_files"] = stats["resumable_files"].(int) + 1
		}

		stats["total_partial_bytes"] = stats["total_partial_bytes"].(int64) + info.DownloadedBytes
	}

	return stats, nil
}
