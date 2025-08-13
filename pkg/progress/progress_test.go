package progress

import (
	"strings"
	"testing"
	"time"
)

func TestNewProgress(t *testing.T) {
	callback := func(downloaded, total int64, speed int64) {}
	progress := NewProgress(1024, callback)

	if progress.TotalBytes != 1024 {
		t.Errorf("Expected TotalBytes 1024, got %d", progress.TotalBytes)
	}

	if progress.DownloadedBytes != 0 {
		t.Errorf("Expected initial DownloadedBytes 0, got %d", progress.DownloadedBytes)
	}

	if progress.callback == nil {
		t.Error("Expected callback to be set")
	}

	if progress.rateLimiter == nil {
		t.Error("Expected rate limiter to be initialized")
	}
}

func TestProgressUpdate(t *testing.T) {
	callbackCalls := 0

	var lastDownloaded, lastTotal, lastSpeed int64

	callback := func(downloaded, total int64, speed int64) {
		callbackCalls++
		lastDownloaded = downloaded
		lastTotal = total
		lastSpeed = speed
	}

	progress := NewProgress(1024, callback)
	// Set a shorter rate limit for testing
	progress.rateLimiter = NewRateLimiter(10 * time.Millisecond)

	// First update should be allowed
	progress.Update(512)

	if progress.DownloadedBytes != 512 {
		t.Errorf("Expected DownloadedBytes 512, got %d", progress.DownloadedBytes)
	}

	// Wait for rate limiter
	time.Sleep(15 * time.Millisecond)

	// Second update should be allowed
	progress.Update(1024)

	if progress.DownloadedBytes != 1024 {
		t.Errorf("Expected DownloadedBytes 1024, got %d", progress.DownloadedBytes)
	}

	if lastDownloaded != 1024 {
		t.Errorf("Expected callback lastDownloaded 1024, got %d", lastDownloaded)
	}

	if lastTotal != 1024 {
		t.Errorf("Expected callback lastTotal 1024, got %d", lastTotal)
	}

	if progress.Speed <= 0 {
		t.Error("Expected speed to be calculated and positive")
	}

	// Use lastSpeed to avoid unused variable warning
	_ = lastSpeed
}

func TestProgressForceUpdate(t *testing.T) {
	callbackCalls := 0
	callback := func(downloaded, total int64, speed int64) {
		callbackCalls++
	}

	progress := NewProgress(1024, callback)

	// Force update should always call callback
	progress.ForceUpdate(256)

	expectedCalls := 1

	if callbackCalls != expectedCalls {
		t.Errorf("Expected %d callback calls, got %d", expectedCalls, callbackCalls)
	}

	// Another force update should also call callback
	progress.ForceUpdate(512)

	expectedCalls++

	if callbackCalls != expectedCalls {
		t.Errorf("Expected %d callback calls, got %d", expectedCalls, callbackCalls)
	}
}

func TestProgressSnapshot(t *testing.T) {
	progress := NewProgress(1024, nil)
	progress.Update(512)

	snapshot := progress.GetSnapshot()

	if snapshot.TotalBytes != 1024 {
		t.Errorf("Expected snapshot TotalBytes 1024, got %d", snapshot.TotalBytes)
	}

	if snapshot.DownloadedBytes != 512 {
		t.Errorf("Expected snapshot DownloadedBytes 512, got %d", snapshot.DownloadedBytes)
	}

	// Test percentage calculation
	percentage := snapshot.Percentage()
	if percentage != 50.0 {
		t.Errorf("Expected percentage 50.0, got %f", percentage)
	}
}

func TestProgressSnapshotUnknownSize(t *testing.T) {
	progress := NewProgress(-1, nil)
	progress.Update(512)

	snapshot := progress.GetSnapshot()
	percentage := snapshot.Percentage()

	if percentage != -1 {
		t.Errorf("Expected percentage -1 for unknown size, got %f", percentage)
	}
}

func TestProgressSnapshotETA(t *testing.T) {
	progress := NewProgress(1024, nil)

	// Simulate some progress over time
	time.Sleep(10 * time.Millisecond)
	progress.Update(256) // 25% complete

	snapshot := progress.GetSnapshot()
	eta := snapshot.ETA()

	// ETA should be positive for partial downloads with speed
	if snapshot.Speed > 0 && eta < 0 {
		t.Errorf("Expected positive ETA for partial download with speed, got %v", eta)
	}
}

func TestProgressSnapshotString(t *testing.T) {
	progress := NewProgress(1024, nil)

	time.Sleep(10 * time.Millisecond)
	progress.Update(512)

	snapshot := progress.GetSnapshot()
	str := snapshot.String()

	// Should contain percentage
	if !strings.Contains(str, "50.0%") {
		t.Errorf("Expected string to contain '50.0%%', got: %s", str)
	}

	// Should contain sizes
	if !strings.Contains(str, "512 B") {
		t.Errorf("Expected string to contain '512 B', got: %s", str)
	}

	if !strings.Contains(str, "1.0 KB") {
		t.Errorf("Expected string to contain '1.0 KB', got: %s", str)
	}
}

func TestRateLimiter(t *testing.T) {
	limiter := NewRateLimiter(50 * time.Millisecond)

	// First call should be allowed
	if !limiter.Allow() {
		t.Error("First call should be allowed")
	}

	// Immediate second call should be denied
	if limiter.Allow() {
		t.Error("Immediate second call should be denied")
	}

	// After waiting, call should be allowed
	time.Sleep(60 * time.Millisecond)

	if !limiter.Allow() {
		t.Error("Call after waiting should be allowed")
	}
}

func TestProgressReader(t *testing.T) {
	data := "Hello, World! This is test data for progress tracking."
	reader := strings.NewReader(data)

	var (
		callbackCalls  int
		lastDownloaded int64
	)

	callback := func(downloaded, total int64, speed int64) {
		callbackCalls++
		lastDownloaded = downloaded
	}

	progressReader := NewProgressReader(reader, int64(len(data)), callback)

	// Read first chunk
	buffer := make([]byte, 10)

	n, err := progressReader.Read(buffer)
	if err != nil {
		t.Fatalf("Unexpected error reading: %v", err)
	}

	if n != 10 {
		t.Errorf("Expected to read 10 bytes, got %d", n)
	}

	if lastDownloaded != 10 {
		t.Errorf("Expected lastDownloaded 10, got %d", lastDownloaded)
	}

	// Wait for rate limiter to allow update
	time.Sleep(600 * time.Millisecond)

	// Read remaining data
	remaining := make([]byte, len(data)-10)

	n, err = progressReader.Read(remaining)
	if err != nil {
		t.Fatalf("Unexpected error reading remaining: %v", err)
	}

	expectedRemaining := len(data) - 10
	if n != expectedRemaining {
		t.Errorf("Expected to read %d bytes, got %d", expectedRemaining, n)
	}

	// Wait for another rate limiter cycle
	time.Sleep(600 * time.Millisecond)

	// Force an update to ensure callback is called
	progressReader.GetProgress().ForceUpdate(int64(len(data)))

	if lastDownloaded != int64(len(data)) {
		t.Errorf("Expected lastDownloaded %d, got %d", len(data), lastDownloaded)
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{500, "500 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1024 * 1024, "1.0 MB"},
		{1024 * 1024 * 1024, "1.0 GB"},
		{1024 * 1024 * 1024 * 1024, "1.0 TB"},
	}

	for _, test := range tests {
		result := formatBytes(test.bytes)
		if result != test.expected {
			t.Errorf("formatBytes(%d) = %s, expected %s", test.bytes, result, test.expected)
		}
	}
}

func TestFormatDuration(t *testing.T) {
	tests := []struct {
		duration time.Duration
		expected string
	}{
		{-1 * time.Second, "unknown"},
		{30 * time.Second, "30s"},
		{90 * time.Second, "1m30s"},
		{3661 * time.Second, "1h1m"},
		{0, "0s"},
	}

	for _, test := range tests {
		result := formatDuration(test.duration)
		if result != test.expected {
			t.Errorf("formatDuration(%v) = %s, expected %s", test.duration, result, test.expected)
		}
	}
}

func TestProgressConcurrency(t *testing.T) {
	// Test that Progress is safe for concurrent use
	progress := NewProgress(1000, nil)

	done := make(chan bool, 10)

	// Start 10 goroutines that update progress concurrently
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				progress.Update(int64(id*10 + j))
				time.Sleep(time.Millisecond)
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Get final snapshot
	snapshot := progress.GetSnapshot()
	if snapshot.DownloadedBytes < 0 {
		t.Error("Downloaded bytes should not be negative after concurrent updates")
	}
}

func TestProgressElapsed(t *testing.T) {
	progress := NewProgress(1024, nil)

	// Wait a bit and update
	time.Sleep(50 * time.Millisecond)
	progress.Update(512)

	snapshot := progress.GetSnapshot()
	elapsed := snapshot.Elapsed()

	if elapsed < 50*time.Millisecond {
		t.Errorf("Expected elapsed time >= 50ms, got %v", elapsed)
	}

	if elapsed > 200*time.Millisecond {
		t.Errorf("Expected elapsed time <= 200ms, got %v", elapsed)
	}
}
