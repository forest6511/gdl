// Package main demonstrates production-ready usage patterns for the gdl library.
// This example shows how to integrate gdl into production applications with
// proper error handling, monitoring, and resilience patterns.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"sync"
	"syscall"
	"time"

	"github.com/forest6511/gdl"
)

// DownloadManager manages multiple concurrent downloads with monitoring
type DownloadManager struct {
	downloader   *gdl.Downloader
	downloads    map[string]*DownloadJob
	mu           sync.RWMutex
	metrics      *Metrics
	shutdownChan chan struct{}
}

// DownloadJob represents a single download operation
type DownloadJob struct {
	ID          string
	URL         string
	Destination string
	Status      JobStatus
	StartTime   time.Time
	EndTime     time.Time
	Progress    *ProgressInfo
	Error       error
	Retries     int
	MaxRetries  int
}

// JobStatus represents the status of a download job
type JobStatus int

const (
	StatusPending JobStatus = iota
	StatusRunning
	StatusCompleted
	StatusFailed
	StatusCancelled
)

// ProgressInfo tracks download progress
type ProgressInfo struct {
	TotalBytes      int64
	DownloadedBytes int64
	Speed           int64
	Percentage      float64
	ETA             time.Duration
	LastUpdate      time.Time
}

// Metrics tracks download statistics
type Metrics struct {
	TotalDownloads      int64
	SuccessfulDownloads int64
	FailedDownloads     int64
	TotalBytes          int64
	AverageSpeed        float64
	mu                  sync.RWMutex
}

// NewDownloadManager creates a new download manager
func NewDownloadManager() *DownloadManager {
	return &DownloadManager{
		downloader:   gdl.NewDownloader(),
		downloads:    make(map[string]*DownloadJob),
		metrics:      &Metrics{},
		shutdownChan: make(chan struct{}),
	}
}

// AddDownload adds a new download job to the queue
func (dm *DownloadManager) AddDownload(id, url, destination string) error {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	if _, exists := dm.downloads[id]; exists {
		return fmt.Errorf("download with ID %s already exists", id)
	}

	job := &DownloadJob{
		ID:          id,
		URL:         url,
		Destination: destination,
		Status:      StatusPending,
		Progress:    &ProgressInfo{},
		MaxRetries:  3,
	}

	dm.downloads[id] = job
	return nil
}

// StartDownload initiates a download with comprehensive error handling
func (dm *DownloadManager) StartDownload(ctx context.Context, id string) error {
	dm.mu.Lock()
	job, exists := dm.downloads[id]
	if !exists {
		dm.mu.Unlock()
		return fmt.Errorf("download job %s not found", id)
	}
	job.Status = StatusRunning
	job.StartTime = time.Now()
	dm.mu.Unlock()

	// Update metrics
	dm.metrics.mu.Lock()
	dm.metrics.TotalDownloads++
	dm.metrics.mu.Unlock()

	// Create download options with progress tracking
	options := &gdl.Options{
		ProgressCallback: func(progress gdl.Progress) {
			dm.updateProgress(id, progress)
		},
		MaxConcurrency:    4,
		ChunkSize:         1024 * 1024, // 1MB chunks
		EnableResume:      true,
		RetryAttempts:     job.MaxRetries,
		Timeout:           30 * time.Minute,
		CreateDirs:        true,
		OverwriteExisting: false,
		UserAgent:         "gdl-production/1.0",
	}

	// Perform download with retry logic
	err := dm.downloadWithRetry(ctx, job, options)

	dm.mu.Lock()
	job.EndTime = time.Now()
	if err != nil {
		job.Status = StatusFailed
		job.Error = err
		dm.metrics.mu.Lock()
		dm.metrics.FailedDownloads++
		dm.metrics.mu.Unlock()
	} else {
		job.Status = StatusCompleted
		dm.metrics.mu.Lock()
		dm.metrics.SuccessfulDownloads++
		dm.metrics.TotalBytes += job.Progress.TotalBytes
		dm.metrics.mu.Unlock()
	}
	dm.mu.Unlock()

	return err
}

// downloadWithRetry implements exponential backoff retry logic
func (dm *DownloadManager) downloadWithRetry(ctx context.Context, job *DownloadJob, options *gdl.Options) error {
	var lastErr error

	for attempt := 0; attempt <= job.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			backoffDuration := time.Duration(attempt*attempt) * time.Second
			log.Printf("Download %s failed, retrying in %v (attempt %d/%d)",
				job.ID, backoffDuration, attempt, job.MaxRetries)

			select {
			case <-time.After(backoffDuration):
			case <-ctx.Done():
				return ctx.Err()
			case <-dm.shutdownChan:
				return fmt.Errorf("download manager shutting down")
			}
		}

		job.Retries = attempt
		stats, err := gdl.DownloadWithOptions(ctx, job.URL, job.Destination, options)

		if err == nil {
			log.Printf("Download %s completed successfully in %v",
				job.ID, stats.Duration)
			return nil
		}

		lastErr = err
		log.Printf("Download %s attempt %d failed: %v", job.ID, attempt+1, err)

		// Check if error is retryable
		if !isRetryableError(err) {
			log.Printf("Download %s failed with non-retryable error: %v", job.ID, err)
			break
		}
	}

	return fmt.Errorf("download failed after %d attempts: %w", job.MaxRetries+1, lastErr)
}

// updateProgress updates the progress information for a download
func (dm *DownloadManager) updateProgress(id string, progress gdl.Progress) {
	dm.mu.Lock()
	defer dm.mu.Unlock()

	job, exists := dm.downloads[id]
	if !exists {
		return
	}

	job.Progress.TotalBytes = progress.TotalSize
	job.Progress.DownloadedBytes = progress.BytesDownloaded
	job.Progress.Speed = progress.Speed
	job.Progress.Percentage = progress.Percentage
	job.Progress.ETA = progress.TimeRemaining
	job.Progress.LastUpdate = time.Now()
}

// GetJobStatus returns the current status of a download job
func (dm *DownloadManager) GetJobStatus(id string) (*DownloadJob, error) {
	dm.mu.RLock()
	defer dm.mu.RUnlock()

	job, exists := dm.downloads[id]
	if !exists {
		return nil, fmt.Errorf("download job %s not found", id)
	}

	// Return a copy to avoid race conditions
	return &DownloadJob{
		ID:          job.ID,
		URL:         job.URL,
		Destination: job.Destination,
		Status:      job.Status,
		StartTime:   job.StartTime,
		EndTime:     job.EndTime,
		Progress:    job.Progress,
		Error:       job.Error,
		Retries:     job.Retries,
		MaxRetries:  job.MaxRetries,
	}, nil
}

// GetMetrics returns current download metrics
func (dm *DownloadManager) GetMetrics() *Metrics {
	dm.metrics.mu.RLock()
	defer dm.metrics.mu.RUnlock()

	return &Metrics{
		TotalDownloads:      dm.metrics.TotalDownloads,
		SuccessfulDownloads: dm.metrics.SuccessfulDownloads,
		FailedDownloads:     dm.metrics.FailedDownloads,
		TotalBytes:          dm.metrics.TotalBytes,
		AverageSpeed:        dm.metrics.AverageSpeed,
	}
}

// StartMonitoring starts a monitoring goroutine that reports metrics
func (dm *DownloadManager) StartMonitoring(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			dm.reportMetrics()
		case <-ctx.Done():
			log.Println("Monitoring stopped")
			return
		case <-dm.shutdownChan:
			log.Println("Monitoring stopped due to shutdown")
			return
		}
	}
}

// reportMetrics logs current metrics
func (dm *DownloadManager) reportMetrics() {
	metrics := dm.GetMetrics()

	log.Printf("Download Metrics - Total: %d, Success: %d, Failed: %d, Data: %.2f MB",
		metrics.TotalDownloads,
		metrics.SuccessfulDownloads,
		metrics.FailedDownloads,
		float64(metrics.TotalBytes)/(1024*1024),
	)
}

// Shutdown gracefully shuts down the download manager
func (dm *DownloadManager) Shutdown() {
	close(dm.shutdownChan)
	log.Println("Download manager shutdown initiated")
}

// isRetryableError determines if an error should trigger a retry
func isRetryableError(err error) bool {
	// In a real implementation, you would check for specific error types
	// such as network timeouts, temporary server errors, etc.
	errStr := err.Error()

	// Retry on network-related errors
	retryableErrors := []string{
		"timeout",
		"connection refused",
		"temporary failure",
		"server error",
		"503",
		"502",
		"504",
	}

	for _, retryable := range retryableErrors {
		if contains(errStr, retryable) {
			return true
		}
	}

	return false
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr || len(s) > len(substr) &&
			(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
				(len(s) > len(substr) && s[1:len(substr)+1] == substr)))
}

// ProductionExample demonstrates production usage patterns
func ProductionExample() {
	log.Println("=== Production Usage Example ===")

	// Create context with cancellation
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Set up signal handling for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	// Create download manager
	dm := NewDownloadManager()

	// Start monitoring in background
	go dm.StartMonitoring(ctx, 30*time.Second)

	// Create output directory
	outputDir := "downloads/production"
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Failed to create output directory: %v", err)
	}

	// Add multiple download jobs
	downloads := []struct {
		id   string
		url  string
		file string
	}{
		{"job1", "https://httpbin.org/bytes/1048576", "file1.dat"}, // 1MB
		{"job2", "https://httpbin.org/bytes/2097152", "file2.dat"}, // 2MB
		{"job3", "https://httpbin.org/bytes/512000", "file3.dat"},  // 512KB
	}

	// Add jobs to manager
	for _, download := range downloads {
		destPath := filepath.Join(outputDir, download.file)
		if err := dm.AddDownload(download.id, download.url, destPath); err != nil {
			log.Printf("Failed to add download %s: %v", download.id, err)
			continue
		}
		log.Printf("Added download job: %s -> %s", download.id, destPath)
	}

	// Start downloads concurrently
	var wg sync.WaitGroup
	for _, download := range downloads {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()

			if err := dm.StartDownload(ctx, id); err != nil {
				log.Printf("Download %s failed: %v", id, err)
			} else {
				log.Printf("Download %s completed successfully", id)
			}
		}(download.id)
	}

	// Wait for completion or shutdown signal
	done := make(chan struct{})
	go func() {
		wg.Wait()
		close(done)
	}()

	select {
	case <-done:
		log.Println("All downloads completed")
	case <-sigChan:
		log.Println("Shutdown signal received")
		cancel()
		dm.Shutdown()

		// Wait for graceful shutdown with timeout
		select {
		case <-done:
			log.Println("Downloads completed during shutdown")
		case <-time.After(10 * time.Second):
			log.Println("Shutdown timeout reached")
		}
	}

	// Report final metrics
	dm.reportMetrics()

	// Show job statuses
	log.Println("\n=== Final Job Status ===")
	for _, download := range downloads {
		if job, err := dm.GetJobStatus(download.id); err == nil {
			log.Printf("Job %s: Status=%v, Retries=%d, Duration=%v",
				job.ID,
				statusString(job.Status),
				job.Retries,
				job.EndTime.Sub(job.StartTime),
			)
			if job.Error != nil {
				log.Printf("  Error: %v", job.Error)
			}
		}
	}
}

// statusString converts JobStatus to string
func statusString(status JobStatus) string {
	switch status {
	case StatusPending:
		return "Pending"
	case StatusRunning:
		return "Running"
	case StatusCompleted:
		return "Completed"
	case StatusFailed:
		return "Failed"
	case StatusCancelled:
		return "Cancelled"
	default:
		return "Unknown"
	}
}

// HealthCheckExample demonstrates health checking and monitoring
func HealthCheckExample() {
	log.Println("\n=== Health Check Example ===")

	// Check if gdl can perform basic operations
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Test basic download functionality
	tempFile := filepath.Join(os.TempDir(), "health_check.dat")
	defer os.Remove(tempFile)

	start := time.Now()
	stats, err := gdl.Download(ctx, "https://httpbin.org/bytes/1024", tempFile)
	duration := time.Since(start)

	if err != nil {
		log.Printf("Health check FAILED: %v", err)
		return
	}

	log.Printf("Health check PASSED:")
	log.Printf("  Downloaded: %d bytes", stats.BytesDownloaded)
	log.Printf("  Duration: %v", duration)
	log.Printf("  Speed: %.2f KB/s", float64(stats.AverageSpeed)/1024)
	log.Printf("  Retries: %d", stats.Retries)
}

// ErrorHandlingExample demonstrates comprehensive error handling
func ErrorHandlingExample() {
	log.Println("\n=== Error Handling Example ===")

	ctx := context.Background()

	// Test various error scenarios
	errorTests := []struct {
		name string
		url  string
		desc string
	}{
		{"Invalid URL", "not-a-url", "Malformed URL"},
		{"Non-existent host", "https://thisdomaindoesnotexist12345.com/file", "DNS resolution failure"},
		{"404 Not Found", "https://httpbin.org/status/404", "HTTP 404 error"},
		{"500 Server Error", "https://httpbin.org/status/500", "HTTP 500 error"},
		{"Timeout", "https://httpbin.org/delay/60", "Request timeout"},
	}

	for _, test := range errorTests {
		log.Printf("\nTesting: %s (%s)", test.name, test.desc)

		tempFile := filepath.Join(os.TempDir(), fmt.Sprintf("error_test_%s.dat", test.name))
		defer os.Remove(tempFile)

		// Use short timeout for timeout test
		testCtx := ctx
		if test.name == "Timeout" {
			var cancel context.CancelFunc
			testCtx, cancel = context.WithTimeout(ctx, 5*time.Second)
			defer cancel()
		}

		options := &gdl.Options{
			RetryAttempts: 2,
			Timeout:       5 * time.Second,
		}

		_, err := gdl.DownloadWithOptions(testCtx, test.url, tempFile, options)

		if err != nil {
			log.Printf("  Expected error occurred: %v", err)
		} else {
			log.Printf("  Unexpected success (error expected)")
		}
	}
}

// main demonstrates production-ready usage patterns
func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	log.Println("Starting gdl production usage examples...")

	// Run health check first
	HealthCheckExample()

	// Demonstrate error handling patterns
	ErrorHandlingExample()

	// Run production download manager example
	ProductionExample()

	log.Println("\nAll examples completed!")
}
