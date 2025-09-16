// Package monitoring provides performance monitoring and metrics collection for gdl downloads.
package monitoring

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/forest6511/gdl/pkg/types"
)

// MetricsCollector collects and aggregates download performance metrics
type MetricsCollector struct {
	mu                sync.RWMutex
	downloads         map[string]*DownloadMetrics
	aggregated        *AggregatedMetrics
	enabled           bool
	retentionDuration time.Duration
}

// DownloadMetrics contains metrics for a single download operation
type DownloadMetrics struct {
	ID              string
	URL             string
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	TotalBytes      int64
	BytesDownloaded int64
	AverageSpeed    int64
	MaxSpeed        int64
	MinSpeed        int64
	RetryCount      int
	ChunksUsed      int
	Success         bool
	ErrorType       string
	ErrorMessage    string
	Resumed         bool
	ConcurrencyUsed int
	Protocol        string
	StatusCode      int
}

// AggregatedMetrics contains aggregated metrics across all downloads
type AggregatedMetrics struct {
	TotalDownloads      int64
	SuccessfulDownloads int64
	FailedDownloads     int64
	TotalBytes          int64
	TotalDuration       time.Duration
	AverageSpeed        float64
	ThroughputMBps      float64
	SuccessRate         float64
	AverageRetries      float64
	AverageConcurrency  float64
	ProtocolBreakdown   map[string]int64
	ErrorBreakdown      map[string]int64
	LastUpdated         time.Time
}

// PerformanceSnapshot represents a point-in-time performance measurement
type PerformanceSnapshot struct {
	Timestamp             time.Time
	ActiveDownloads       int
	TotalThroughput       int64
	MemoryUsage           int64
	ConcurrentConnections int
	QueuedDownloads       int
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		downloads: make(map[string]*DownloadMetrics),
		aggregated: &AggregatedMetrics{
			ProtocolBreakdown: make(map[string]int64),
			ErrorBreakdown:    make(map[string]int64),
		},
		enabled:           true,
		retentionDuration: 24 * time.Hour, // Keep metrics for 24 hours
	}
}

// Enable enables metrics collection
func (mc *MetricsCollector) Enable() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.enabled = true
}

// Disable disables metrics collection
func (mc *MetricsCollector) Disable() {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.enabled = false
}

// SetRetentionDuration sets how long to keep individual download metrics
func (mc *MetricsCollector) SetRetentionDuration(duration time.Duration) {
	mc.mu.Lock()
	defer mc.mu.Unlock()
	mc.retentionDuration = duration
}

// RecordDownloadStart records the start of a download operation
func (mc *MetricsCollector) RecordDownloadStart(id, url string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.enabled {
		return
	}

	protocol := extractProtocol(url)

	metrics := &DownloadMetrics{
		ID:        id,
		URL:       url,
		StartTime: time.Now(),
		Protocol:  protocol,
		MinSpeed:  int64(^uint64(0) >> 1), // Max int64 as initial min
	}

	mc.downloads[id] = metrics
}

// RecordDownloadProgress updates progress metrics for an ongoing download
func (mc *MetricsCollector) RecordDownloadProgress(id string, bytesDownloaded, totalBytes, currentSpeed int64) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.enabled {
		return
	}

	metrics, exists := mc.downloads[id]
	if !exists {
		return
	}

	metrics.BytesDownloaded = bytesDownloaded
	metrics.TotalBytes = totalBytes

	// Update speed metrics
	if currentSpeed > metrics.MaxSpeed {
		metrics.MaxSpeed = currentSpeed
	}
	if currentSpeed < metrics.MinSpeed && currentSpeed > 0 {
		metrics.MinSpeed = currentSpeed
	}

	// Calculate average speed
	elapsed := time.Since(metrics.StartTime)
	if elapsed > 0 {
		metrics.AverageSpeed = int64(float64(bytesDownloaded) / elapsed.Seconds())
	}
}

// RecordDownloadComplete records the completion of a download operation
func (mc *MetricsCollector) RecordDownloadComplete(id string, stats *types.DownloadStats) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if !mc.enabled {
		return
	}

	metrics, exists := mc.downloads[id]
	if !exists {
		return
	}

	metrics.EndTime = time.Now()
	metrics.Duration = metrics.EndTime.Sub(metrics.StartTime)
	metrics.Success = stats.Success
	metrics.RetryCount = stats.Retries
	metrics.ChunksUsed = stats.ChunksUsed
	metrics.Resumed = stats.Resumed
	metrics.TotalBytes = stats.TotalSize
	metrics.BytesDownloaded = stats.BytesDownloaded
	metrics.AverageSpeed = stats.AverageSpeed

	if stats.Error != nil {
		metrics.ErrorType = classifyError(stats.Error)
		metrics.ErrorMessage = stats.Error.Error()
	}

	// Update aggregated metrics
	mc.updateAggregatedMetrics()
}

// updateAggregatedMetrics recalculates aggregated metrics
func (mc *MetricsCollector) updateAggregatedMetrics() {
	// Reset aggregated metrics
	mc.aggregated = &AggregatedMetrics{
		ProtocolBreakdown: make(map[string]int64),
		ErrorBreakdown:    make(map[string]int64),
		LastUpdated:       time.Now(),
	}

	var totalSpeed float64
	var totalConcurrency int64
	var validSpeedCount int64

	for _, metrics := range mc.downloads {
		mc.aggregated.TotalDownloads++
		mc.aggregated.TotalBytes += metrics.TotalBytes
		mc.aggregated.TotalDuration += metrics.Duration

		// Protocol breakdown
		mc.aggregated.ProtocolBreakdown[metrics.Protocol]++

		if metrics.Success {
			mc.aggregated.SuccessfulDownloads++
			if metrics.AverageSpeed > 0 {
				totalSpeed += float64(metrics.AverageSpeed)
				validSpeedCount++
			}
		} else {
			mc.aggregated.FailedDownloads++
			if metrics.ErrorType != "" {
				mc.aggregated.ErrorBreakdown[metrics.ErrorType]++
			}
		}

		mc.aggregated.AverageRetries += float64(metrics.RetryCount)
		totalConcurrency += int64(metrics.ConcurrencyUsed)
	}

	// Calculate derived metrics
	if mc.aggregated.TotalDownloads > 0 {
		mc.aggregated.SuccessRate = float64(mc.aggregated.SuccessfulDownloads) / float64(mc.aggregated.TotalDownloads)
		mc.aggregated.AverageRetries /= float64(mc.aggregated.TotalDownloads)
		mc.aggregated.AverageConcurrency = float64(totalConcurrency) / float64(mc.aggregated.TotalDownloads)
	}

	if validSpeedCount > 0 {
		mc.aggregated.AverageSpeed = totalSpeed / float64(validSpeedCount)
		mc.aggregated.ThroughputMBps = mc.aggregated.AverageSpeed / (1024 * 1024)
	}
}

// GetDownloadMetrics returns metrics for a specific download
func (mc *MetricsCollector) GetDownloadMetrics(id string) (*DownloadMetrics, error) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics, exists := mc.downloads[id]
	if !exists {
		return nil, fmt.Errorf("metrics not found for download ID: %s", id)
	}

	// Return a copy to avoid race conditions
	metricsCopy := *metrics
	return &metricsCopy, nil
}

// GetAggregatedMetrics returns current aggregated metrics
func (mc *MetricsCollector) GetAggregatedMetrics() *AggregatedMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	// Return a copy to avoid race conditions
	aggregatedCopy := *mc.aggregated
	aggregatedCopy.ProtocolBreakdown = make(map[string]int64)
	aggregatedCopy.ErrorBreakdown = make(map[string]int64)

	for k, v := range mc.aggregated.ProtocolBreakdown {
		aggregatedCopy.ProtocolBreakdown[k] = v
	}
	for k, v := range mc.aggregated.ErrorBreakdown {
		aggregatedCopy.ErrorBreakdown[k] = v
	}

	return &aggregatedCopy
}

// GetAllDownloadMetrics returns metrics for all downloads
func (mc *MetricsCollector) GetAllDownloadMetrics() []*DownloadMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics := make([]*DownloadMetrics, 0, len(mc.downloads))
	for _, m := range mc.downloads {
		metricsCopy := *m
		metrics = append(metrics, &metricsCopy)
	}

	return metrics
}

// CleanupOldMetrics removes metrics older than the retention duration
func (mc *MetricsCollector) CleanupOldMetrics() {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	cutoff := time.Now().Add(-mc.retentionDuration)

	for id, metrics := range mc.downloads {
		if metrics.EndTime.Before(cutoff) && !metrics.EndTime.IsZero() {
			delete(mc.downloads, id)
		}
	}

	// Recalculate aggregated metrics after cleanup
	mc.updateAggregatedMetrics()
}

// GetPerformanceSnapshot returns a current performance snapshot
func (mc *MetricsCollector) GetPerformanceSnapshot() *PerformanceSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	var activeDownloads int
	var totalThroughput int64
	var concurrentConnections int

	for _, metrics := range mc.downloads {
		if metrics.EndTime.IsZero() { // Still active
			activeDownloads++
			totalThroughput += metrics.AverageSpeed
			concurrentConnections += metrics.ConcurrencyUsed
		}
	}

	return &PerformanceSnapshot{
		Timestamp:             time.Now(),
		ActiveDownloads:       activeDownloads,
		TotalThroughput:       totalThroughput,
		ConcurrentConnections: concurrentConnections,
	}
}

// StartPeriodicCleanup starts a goroutine that periodically cleans up old metrics
func (mc *MetricsCollector) StartPeriodicCleanup(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.CleanupOldMetrics()
		case <-ctx.Done():
			return
		}
	}
}

// ExportMetrics exports metrics in a structured format
func (mc *MetricsCollector) ExportMetrics() map[string]interface{} {
	aggregated := mc.GetAggregatedMetrics()
	allDownloads := mc.GetAllDownloadMetrics()

	export := map[string]interface{}{
		"aggregated": map[string]interface{}{
			"total_downloads":      aggregated.TotalDownloads,
			"successful_downloads": aggregated.SuccessfulDownloads,
			"failed_downloads":     aggregated.FailedDownloads,
			"total_bytes":          aggregated.TotalBytes,
			"average_speed_bps":    aggregated.AverageSpeed,
			"throughput_mbps":      aggregated.ThroughputMBps,
			"success_rate":         aggregated.SuccessRate,
			"average_retries":      aggregated.AverageRetries,
			"average_concurrency":  aggregated.AverageConcurrency,
			"protocol_breakdown":   aggregated.ProtocolBreakdown,
			"error_breakdown":      aggregated.ErrorBreakdown,
			"last_updated":         aggregated.LastUpdated,
		},
		"downloads": make([]map[string]interface{}, len(allDownloads)),
	}

	for i, download := range allDownloads {
		export["downloads"].([]map[string]interface{})[i] = map[string]interface{}{
			"id":                download.ID,
			"url":               download.URL,
			"start_time":        download.StartTime,
			"end_time":          download.EndTime,
			"duration_ms":       download.Duration.Milliseconds(),
			"total_bytes":       download.TotalBytes,
			"bytes_downloaded":  download.BytesDownloaded,
			"average_speed_bps": download.AverageSpeed,
			"max_speed_bps":     download.MaxSpeed,
			"min_speed_bps":     download.MinSpeed,
			"retry_count":       download.RetryCount,
			"chunks_used":       download.ChunksUsed,
			"success":           download.Success,
			"error_type":        download.ErrorType,
			"resumed":           download.Resumed,
			"concurrency_used":  download.ConcurrencyUsed,
			"protocol":          download.Protocol,
		}
	}

	return export
}

// Helper functions

// extractProtocol extracts the protocol from a URL
func extractProtocol(url string) string {
	if len(url) < 4 {
		return "unknown"
	}

	if url[:4] == "http" {
		if len(url) > 4 && url[4] == 's' {
			return "https"
		}
		return "http"
	}

	if len(url) > 3 && url[:3] == "ftp" {
		return "ftp"
	}

	if len(url) > 2 && url[:2] == "s3" {
		return "s3"
	}

	return "unknown"
}

// classifyError classifies errors into categories for metrics
func classifyError(err error) string {
	if err == nil {
		return ""
	}

	errStr := err.Error()

	// Network errors
	if contains(errStr, "timeout") || contains(errStr, "deadline exceeded") {
		return "timeout"
	}
	if contains(errStr, "connection refused") || contains(errStr, "no route to host") {
		return "network"
	}
	if contains(errStr, "dns") || contains(errStr, "no such host") {
		return "dns"
	}

	// HTTP errors
	if contains(errStr, "404") || contains(errStr, "not found") {
		return "not_found"
	}
	if contains(errStr, "403") || contains(errStr, "forbidden") {
		return "forbidden"
	}
	if contains(errStr, "401") || contains(errStr, "unauthorized") {
		return "unauthorized"
	}
	if contains(errStr, "500") || contains(errStr, "502") || contains(errStr, "503") || contains(errStr, "504") {
		return "server_error"
	}

	// File system errors
	if contains(errStr, "permission denied") || contains(errStr, "access denied") {
		return "permission"
	}
	if contains(errStr, "no space") || contains(errStr, "disk full") {
		return "disk_space"
	}

	// General categories
	if contains(errStr, "cancelled") || contains(errStr, "context canceled") {
		return "cancelled"
	}

	return "unknown"
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		(s == substr ||
			(len(s) > len(substr) &&
				(s[:len(substr)] == substr ||
					s[len(s)-len(substr):] == substr ||
					indexOfSubstring(s, substr) >= 0)))
}

// indexOfSubstring finds the index of a substring in a string
func indexOfSubstring(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}
