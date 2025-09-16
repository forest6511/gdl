package monitoring

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/forest6511/gdl/pkg/types"
)

func TestNewMetricsCollector(t *testing.T) {
	mc := NewMetricsCollector()

	if mc == nil {
		t.Fatal("NewMetricsCollector returned nil")
	}

	if !mc.enabled {
		t.Error("MetricsCollector should be enabled by default")
	}

	if mc.downloads == nil {
		t.Error("downloads map should be initialized")
	}

	if mc.aggregated == nil {
		t.Error("aggregated metrics should be initialized")
	}

	if mc.retentionDuration != 24*time.Hour {
		t.Errorf("Expected retention duration 24h, got %v", mc.retentionDuration)
	}
}

func TestMetricsCollector_EnableDisable(t *testing.T) {
	mc := NewMetricsCollector()

	// Test disable
	mc.Disable()
	if mc.enabled {
		t.Error("MetricsCollector should be disabled")
	}

	// Test enable
	mc.Enable()
	if !mc.enabled {
		t.Error("MetricsCollector should be enabled")
	}
}

func TestMetricsCollector_SetRetentionDuration(t *testing.T) {
	mc := NewMetricsCollector()

	duration := 12 * time.Hour
	mc.SetRetentionDuration(duration)

	if mc.retentionDuration != duration {
		t.Errorf("Expected retention duration %v, got %v", duration, mc.retentionDuration)
	}
}

func TestMetricsCollector_RecordDownloadStart(t *testing.T) {
	mc := NewMetricsCollector()

	id := "test-download"
	url := "https://example.com/file.zip"

	mc.RecordDownloadStart(id, url)

	metrics, err := mc.GetDownloadMetrics(id)
	if err != nil {
		t.Fatalf("Failed to get download metrics: %v", err)
	}

	if metrics.ID != id {
		t.Errorf("Expected ID %s, got %s", id, metrics.ID)
	}

	if metrics.URL != url {
		t.Errorf("Expected URL %s, got %s", url, metrics.URL)
	}

	if metrics.Protocol != "https" {
		t.Errorf("Expected protocol https, got %s", metrics.Protocol)
	}

	if metrics.StartTime.IsZero() {
		t.Error("Start time should be set")
	}
}

func TestMetricsCollector_RecordDownloadProgress(t *testing.T) {
	mc := NewMetricsCollector()

	id := "test-download"
	mc.RecordDownloadStart(id, "https://example.com/file.zip")

	// Record progress
	bytesDownloaded := int64(1024)
	totalBytes := int64(2048)
	currentSpeed := int64(500)

	mc.RecordDownloadProgress(id, bytesDownloaded, totalBytes, currentSpeed)

	metrics, err := mc.GetDownloadMetrics(id)
	if err != nil {
		t.Fatalf("Failed to get download metrics: %v", err)
	}

	if metrics.BytesDownloaded != bytesDownloaded {
		t.Errorf("Expected bytes downloaded %d, got %d", bytesDownloaded, metrics.BytesDownloaded)
	}

	if metrics.TotalBytes != totalBytes {
		t.Errorf("Expected total bytes %d, got %d", totalBytes, metrics.TotalBytes)
	}

	if metrics.MaxSpeed != currentSpeed {
		t.Errorf("Expected max speed %d, got %d", currentSpeed, metrics.MaxSpeed)
	}

	if metrics.MinSpeed != currentSpeed {
		t.Errorf("Expected min speed %d, got %d", currentSpeed, metrics.MinSpeed)
	}
}

func TestMetricsCollector_RecordDownloadComplete(t *testing.T) {
	mc := NewMetricsCollector()

	id := "test-download"
	mc.RecordDownloadStart(id, "https://example.com/file.zip")

	// Simulate completion
	stats := &types.DownloadStats{
		Success:         true,
		TotalSize:       2048,
		BytesDownloaded: 2048,
		AverageSpeed:    1024,
		Retries:         1,
		ChunksUsed:      4,
		Resumed:         false,
	}

	mc.RecordDownloadComplete(id, stats)

	metrics, err := mc.GetDownloadMetrics(id)
	if err != nil {
		t.Fatalf("Failed to get download metrics: %v", err)
	}

	if !metrics.Success {
		t.Error("Expected success to be true")
	}

	if metrics.RetryCount != stats.Retries {
		t.Errorf("Expected retry count %d, got %d", stats.Retries, metrics.RetryCount)
	}

	if metrics.ChunksUsed != stats.ChunksUsed {
		t.Errorf("Expected chunks used %d, got %d", stats.ChunksUsed, metrics.ChunksUsed)
	}

	if metrics.EndTime.IsZero() {
		t.Error("End time should be set")
	}

	if metrics.Duration == 0 {
		t.Error("Duration should be calculated")
	}
}

func TestMetricsCollector_RecordDownloadCompleteWithError(t *testing.T) {
	mc := NewMetricsCollector()

	id := "test-download"
	mc.RecordDownloadStart(id, "https://example.com/file.zip")

	// Simulate failure
	testError := errors.New("connection timeout")
	stats := &types.DownloadStats{
		Success: false,
		Error:   testError,
		Retries: 3,
	}

	mc.RecordDownloadComplete(id, stats)

	metrics, err := mc.GetDownloadMetrics(id)
	if err != nil {
		t.Fatalf("Failed to get download metrics: %v", err)
	}

	if metrics.Success {
		t.Error("Expected success to be false")
	}

	if metrics.ErrorType != "timeout" {
		t.Errorf("Expected error type timeout, got %s", metrics.ErrorType)
	}

	if metrics.ErrorMessage != testError.Error() {
		t.Errorf("Expected error message %s, got %s", testError.Error(), metrics.ErrorMessage)
	}
}

func TestMetricsCollector_GetAggregatedMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	// Add successful download
	mc.RecordDownloadStart("success", "https://example.com/file1.zip")
	mc.RecordDownloadComplete("success", &types.DownloadStats{
		Success:      true,
		TotalSize:    1024,
		AverageSpeed: 512,
	})

	// Add failed download
	mc.RecordDownloadStart("failure", "https://example.com/file2.zip")
	mc.RecordDownloadComplete("failure", &types.DownloadStats{
		Success: false,
		Error:   errors.New("404 not found"),
	})

	aggregated := mc.GetAggregatedMetrics()

	if aggregated.TotalDownloads != 2 {
		t.Errorf("Expected total downloads 2, got %d", aggregated.TotalDownloads)
	}

	if aggregated.SuccessfulDownloads != 1 {
		t.Errorf("Expected successful downloads 1, got %d", aggregated.SuccessfulDownloads)
	}

	if aggregated.FailedDownloads != 1 {
		t.Errorf("Expected failed downloads 1, got %d", aggregated.FailedDownloads)
	}

	if aggregated.SuccessRate != 0.5 {
		t.Errorf("Expected success rate 0.5, got %f", aggregated.SuccessRate)
	}

	if aggregated.ProtocolBreakdown["https"] != 2 {
		t.Errorf("Expected 2 https downloads, got %d", aggregated.ProtocolBreakdown["https"])
	}

	if aggregated.ErrorBreakdown["not_found"] != 1 {
		t.Errorf("Expected 1 not_found error, got %d", aggregated.ErrorBreakdown["not_found"])
	}
}

func TestMetricsCollector_CleanupOldMetrics(t *testing.T) {
	mc := NewMetricsCollector()
	mc.SetRetentionDuration(1 * time.Millisecond)

	// Add a download and complete it
	mc.RecordDownloadStart("old", "https://example.com/file.zip")
	mc.RecordDownloadComplete("old", &types.DownloadStats{Success: true})

	// Wait for retention period
	time.Sleep(2 * time.Millisecond)

	// Add another download
	mc.RecordDownloadStart("new", "https://example.com/file2.zip")

	// Cleanup old metrics
	mc.CleanupOldMetrics()

	// Old download should be removed
	_, err := mc.GetDownloadMetrics("old")
	if err == nil {
		t.Error("Expected old metrics to be cleaned up")
	}

	// New download should still exist
	_, err = mc.GetDownloadMetrics("new")
	if err != nil {
		t.Error("Expected new metrics to still exist")
	}
}

func TestMetricsCollector_GetPerformanceSnapshot(t *testing.T) {
	mc := NewMetricsCollector()

	// Add active download (not completed)
	mc.RecordDownloadStart("active1", "https://example.com/file1.zip")
	mc.RecordDownloadProgress("active1", 500, 1000, 100)

	// Add completed download
	mc.RecordDownloadStart("completed", "https://example.com/file2.zip")
	mc.RecordDownloadComplete("completed", &types.DownloadStats{Success: true})

	snapshot := mc.GetPerformanceSnapshot()

	if snapshot.ActiveDownloads != 1 {
		t.Errorf("Expected 1 active download, got %d", snapshot.ActiveDownloads)
	}

	if snapshot.Timestamp.IsZero() {
		t.Error("Snapshot timestamp should be set")
	}
}

func TestMetricsCollector_DisabledCollection(t *testing.T) {
	mc := NewMetricsCollector()
	mc.Disable()

	// Try to record metrics while disabled
	mc.RecordDownloadStart("test", "https://example.com/file.zip")

	// Should not have recorded anything
	_, err := mc.GetDownloadMetrics("test")
	if err == nil {
		t.Error("Expected no metrics to be recorded when disabled")
	}
}

func TestMetricsCollector_StartPeriodicCleanup(t *testing.T) {
	mc := NewMetricsCollector()
	mc.SetRetentionDuration(1 * time.Millisecond)

	// Add old download
	mc.RecordDownloadStart("old", "https://example.com/file.zip")
	mc.RecordDownloadComplete("old", &types.DownloadStats{Success: true})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Start periodic cleanup
	go mc.StartPeriodicCleanup(ctx, 10*time.Millisecond)

	// Wait for cleanup to occur
	time.Sleep(50 * time.Millisecond)

	// Old metrics should be cleaned up
	_, err := mc.GetDownloadMetrics("old")
	if err == nil {
		t.Error("Expected old metrics to be cleaned up by periodic cleanup")
	}
}

func TestMetricsCollector_ExportMetrics(t *testing.T) {
	mc := NewMetricsCollector()

	// Add test data
	mc.RecordDownloadStart("test", "https://example.com/file.zip")
	mc.RecordDownloadComplete("test", &types.DownloadStats{
		Success:      true,
		TotalSize:    1024,
		AverageSpeed: 512,
	})

	exported := mc.ExportMetrics()

	// Check structure
	if exported["aggregated"] == nil {
		t.Error("Exported metrics should contain aggregated data")
	}

	if exported["downloads"] == nil {
		t.Error("Exported metrics should contain downloads data")
	}

	// Check aggregated data
	aggregated := exported["aggregated"].(map[string]interface{})
	if aggregated["total_downloads"].(int64) != 1 {
		t.Error("Exported aggregated metrics should show 1 total download")
	}

	// Check downloads data
	downloads := exported["downloads"].([]map[string]interface{})
	if len(downloads) != 1 {
		t.Error("Exported downloads should contain 1 download")
	}

	if downloads[0]["id"].(string) != "test" {
		t.Error("Exported download should have correct ID")
	}
}

// Test helper functions

func TestExtractProtocol(t *testing.T) {
	tests := []struct {
		url      string
		expected string
	}{
		{"https://example.com", "https"},
		{"http://example.com", "http"},
		{"ftp://example.com", "ftp"},
		{"s3://bucket/key", "s3"},
		{"invalid", "unknown"},
		{"", "unknown"},
	}

	for _, test := range tests {
		result := extractProtocol(test.url)
		if result != test.expected {
			t.Errorf("extractProtocol(%s) = %s, expected %s", test.url, result, test.expected)
		}
	}
}

func TestClassifyError(t *testing.T) {
	tests := []struct {
		err      error
		expected string
	}{
		{nil, ""},
		{errors.New("connection timeout"), "timeout"},
		{errors.New("deadline exceeded"), "timeout"},
		{errors.New("connection refused"), "network"},
		{errors.New("no such host"), "dns"},
		{errors.New("404 not found"), "not_found"},
		{errors.New("403 forbidden"), "forbidden"},
		{errors.New("401 unauthorized"), "unauthorized"},
		{errors.New("500 internal server error"), "server_error"},
		{errors.New("permission denied"), "permission"},
		{errors.New("no space left"), "disk_space"},
		{errors.New("context canceled"), "cancelled"},
		{errors.New("unknown error"), "unknown"},
	}

	for _, test := range tests {
		result := classifyError(test.err)
		if result != test.expected {
			t.Errorf("classifyError(%v) = %s, expected %s", test.err, result, test.expected)
		}
	}
}

func TestContains(t *testing.T) {
	tests := []struct {
		s        string
		substr   string
		expected bool
	}{
		{"hello world", "world", true},
		{"hello world", "hello", true},
		{"hello world", "lo wo", true},
		{"hello world", "xyz", false},
		{"hello", "hello world", false},
		{"", "test", false},
		{"test", "", true},
	}

	for _, test := range tests {
		result := contains(test.s, test.substr)
		if result != test.expected {
			t.Errorf("contains(%s, %s) = %v, expected %v", test.s, test.substr, result, test.expected)
		}
	}
}

// Benchmark tests

func BenchmarkMetricsCollector_RecordDownloadStart(b *testing.B) {
	mc := NewMetricsCollector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.RecordDownloadStart(fmt.Sprintf("download-%d", i), "https://example.com/file.zip")
	}
}

func BenchmarkMetricsCollector_RecordDownloadProgress(b *testing.B) {
	mc := NewMetricsCollector()
	mc.RecordDownloadStart("test", "https://example.com/file.zip")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		mc.RecordDownloadProgress("test", int64(i), 1024, 100)
	}
}

func BenchmarkMetricsCollector_GetAggregatedMetrics(b *testing.B) {
	mc := NewMetricsCollector()

	// Add some test data
	for i := 0; i < 100; i++ {
		id := fmt.Sprintf("download-%d", i)
		mc.RecordDownloadStart(id, "https://example.com/file.zip")
		mc.RecordDownloadComplete(id, &types.DownloadStats{
			Success:      i%2 == 0,
			TotalSize:    1024,
			AverageSpeed: 512,
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = mc.GetAggregatedMetrics()
	}
}

func BenchmarkExtractProtocol(b *testing.B) {
	url := "https://example.com/path/to/file.zip"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = extractProtocol(url)
	}
}

func BenchmarkClassifyError(b *testing.B) {
	err := errors.New("connection timeout occurred")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = classifyError(err)
	}
}
