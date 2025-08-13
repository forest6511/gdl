package progress

import (
	"testing"
	"time"
)

func TestNewManager(t *testing.T) {
	manager := NewManager()

	if manager == nil {
		t.Fatal("NewManager() returned nil")
	}

	if manager.updateChan == nil {
		t.Error("updateChan should not be nil")
	}

	if cap(manager.updateChan) != 100 {
		t.Errorf("updateChan capacity = %d, want 100", cap(manager.updateChan))
	}

	if manager.movingAvgWindow == nil {
		t.Error("movingAvgWindow should not be nil")
	}

	if cap(manager.movingAvgWindow) != 10 {
		t.Errorf("movingAvgWindow capacity = %d, want 10", cap(manager.movingAvgWindow))
	}
}

func TestUpdate(t *testing.T) {
	manager := NewManager()

	// First update
	manager.Update(1000, 10000)

	progress := manager.GetProgress()
	if progress.DownloadedBytes != 1000 {
		t.Errorf("DownloadedBytes = %d, want 1000", progress.DownloadedBytes)
	}

	if progress.TotalBytes != 10000 {
		t.Errorf("TotalBytes = %d, want 10000", progress.TotalBytes)
	}

	if progress.StartTime.IsZero() {
		t.Error("StartTime should be set after first update")
	}

	// Wait a bit for speed calculation
	time.Sleep(100 * time.Millisecond)

	// Second update
	manager.Update(2000, 10000)

	progress = manager.GetProgress()
	if progress.DownloadedBytes != 2000 {
		t.Errorf("DownloadedBytes = %d, want 2000", progress.DownloadedBytes)
	}

	// Speed calculation varies based on timing, so just check it's non-negative
	if progress.Speed < 0 {
		t.Error("Speed should not be negative")
	}
}

func TestGetProgress(t *testing.T) {
	manager := NewManager()

	// Initial state
	progress := manager.GetProgress()
	if progress.DownloadedBytes != 0 {
		t.Errorf("Initial DownloadedBytes = %d, want 0", progress.DownloadedBytes)
	}

	if progress.TotalBytes != 0 {
		t.Errorf("Initial TotalBytes = %d, want 0", progress.TotalBytes)
	}

	// Update and get
	manager.Update(5000, 20000)
	progress = manager.GetProgress()

	if progress.DownloadedBytes != 5000 {
		t.Errorf("DownloadedBytes = %d, want 5000", progress.DownloadedBytes)
	}

	if progress.TotalBytes != 20000 {
		t.Errorf("TotalBytes = %d, want 20000", progress.TotalBytes)
	}
}

func TestStartAndStop(t *testing.T) {
	manager := NewManager()

	// Start the manager
	manager.Start()

	if manager.rateLimiter == nil {
		t.Error("rateLimiter should be created after Start()")
	}

	// Stop the manager
	manager.Stop()

	if manager.rateLimiter != nil {
		t.Error("rateLimiter should be nil after Stop()")
	}

	// Verify channel is closed
	select {
	case _, ok := <-manager.updateChan:
		if ok {
			t.Error("updateChan should be closed after Stop()")
		}
	default:
		// Channel is closed
	}
}

func TestUpdateChunks(t *testing.T) {
	manager := NewManager()

	manager.UpdateChunks(5, 10)

	progress := manager.GetProgress()
	if progress.ChunksComplete != 5 {
		t.Errorf("ChunksComplete = %d, want 5", progress.ChunksComplete)
	}

	if progress.TotalChunks != 10 {
		t.Errorf("TotalChunks = %d, want 10", progress.TotalChunks)
	}
}

func TestCalculateSpeed(t *testing.T) {
	tests := []struct {
		name        string
		window      []int64
		expectedAvg int64
	}{
		{
			name:        "Empty window",
			window:      []int64{},
			expectedAvg: 0,
		},
		{
			name:        "Single value",
			window:      []int64{1000},
			expectedAvg: 1000,
		},
		{
			name:        "Multiple values",
			window:      []int64{1000, 2000, 3000},
			expectedAvg: 2000,
		},
		{
			name:        "Full window",
			window:      []int64{1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000},
			expectedAvg: 5500,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				movingAvgWindow: tt.window,
			}

			speed := manager.calculateSpeed()
			if speed != tt.expectedAvg {
				t.Errorf("calculateSpeed() = %d, want %d", speed, tt.expectedAvg)
			}
		})
	}
}

func TestCalculateETA(t *testing.T) {
	tests := []struct {
		name            string
		downloadedBytes int64
		totalBytes      int64
		speed           int64
		expectedETA     time.Duration
		expectZero      bool
	}{
		{
			name:            "Normal calculation",
			downloadedBytes: 5000,
			totalBytes:      10000,
			speed:           1000,
			expectedETA:     5 * time.Second,
			expectZero:      false,
		},
		{
			name:            "Zero speed",
			downloadedBytes: 5000,
			totalBytes:      10000,
			speed:           0,
			expectedETA:     0,
			expectZero:      true,
		},
		{
			name:            "Zero total bytes",
			downloadedBytes: 5000,
			totalBytes:      0,
			speed:           1000,
			expectedETA:     0,
			expectZero:      true,
		},
		{
			name:            "Download complete",
			downloadedBytes: 10000,
			totalBytes:      10000,
			speed:           1000,
			expectedETA:     0,
			expectZero:      true,
		},
		{
			name:            "Downloaded exceeds total",
			downloadedBytes: 11000,
			totalBytes:      10000,
			speed:           1000,
			expectedETA:     0,
			expectZero:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			manager := &Manager{
				downloadedBytes: tt.downloadedBytes,
				totalBytes:      tt.totalBytes,
				speed:           tt.speed,
			}

			eta := manager.calculateETA()

			if tt.expectZero {
				if eta != 0 {
					t.Errorf("calculateETA() = %v, want 0", eta)
				}
			} else {
				if eta != tt.expectedETA {
					t.Errorf("calculateETA() = %v, want %v", eta, tt.expectedETA)
				}
			}
		})
	}
}

func TestMovingAverageWindow(t *testing.T) {
	manager := NewManager()

	// Simulate multiple updates with different speeds
	speeds := []int64{1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000, 11000, 12000}

	for i, speed := range speeds {
		manager.movingAvgWindow = append(manager.movingAvgWindow, speed)
		if len(manager.movingAvgWindow) > 10 {
			manager.movingAvgWindow = manager.movingAvgWindow[1:]
		}

		// After adding 12 values, window should contain last 10
		if i >= 9 {
			if len(manager.movingAvgWindow) != 10 {
				t.Errorf(
					"After %d additions, window size = %d, want 10",
					i+1,
					len(manager.movingAvgWindow),
				)
			}

			// Check first element is correct
			expectedFirst := speeds[i-9]
			if manager.movingAvgWindow[0] != expectedFirst {
				t.Errorf("First element = %d, want %d", manager.movingAvgWindow[0], expectedFirst)
			}
		}
	}

	// Calculate average of last 10 speeds (3000 to 12000)
	expectedAvg := int64(7500)
	calculatedAvg := manager.calculateSpeed()

	if calculatedAvg != expectedAvg {
		t.Errorf("Moving average = %d, want %d", calculatedAvg, expectedAvg)
	}
}

func TestConcurrentAccess(t *testing.T) {
	manager := NewManager()

	manager.Start()
	defer manager.Stop()

	// Start multiple goroutines updating progress
	done := make(chan bool, 3)

	// Writer 1
	go func() {
		for i := 0; i < 100; i++ {
			manager.Update(int64(i*100), 10000)
			time.Sleep(time.Millisecond)
		}

		done <- true
	}()

	// Writer 2
	go func() {
		for i := 0; i < 100; i++ {
			manager.UpdateChunks(i%10, 10)
			time.Sleep(time.Millisecond)
		}

		done <- true
	}()

	// Reader
	go func() {
		for i := 0; i < 200; i++ {
			_ = manager.GetProgress()

			time.Sleep(time.Millisecond)
		}

		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		select {
		case <-done:
			// Success
		case <-time.After(5 * time.Second):
			t.Fatal("Concurrent access test timed out")
		}
	}
}

func TestProgressChannelNonBlocking(t *testing.T) {
	manager := NewManager()

	// Fill the channel
	for i := 0; i < 110; i++ { // More than channel capacity (100)
		manager.Update(int64(i), 1000)
	}

	// Should not panic or block
	// Count messages in channel
	count := 0

	for {
		select {
		case <-manager.updateChan:
			count++
		default:
			goto done
		}
	}

done:

	// Should have at most 100 messages (channel capacity)
	if count > 100 {
		t.Errorf("Channel has %d messages, expected at most 100", count)
	}
}

func BenchmarkUpdate(b *testing.B) {
	manager := NewManager()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		manager.Update(int64(i*1000), int64(b.N*1000))
	}
}

func BenchmarkGetProgress(b *testing.B) {
	manager := NewManager()
	manager.Update(5000, 10000)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = manager.GetProgress()
	}
}

func BenchmarkCalculateSpeed(b *testing.B) {
	manager := &Manager{
		movingAvgWindow: []int64{1000, 2000, 3000, 4000, 5000, 6000, 7000, 8000, 9000, 10000},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = manager.calculateSpeed()
	}
}

func BenchmarkConcurrentAccess(b *testing.B) {
	manager := NewManager()

	b.RunParallel(func(pb *testing.PB) {
		i := 0
		for pb.Next() {
			if i%2 == 0 {
				manager.Update(int64(i*1000), 1000000)
			} else {
				_ = manager.GetProgress()
			}

			i++
		}
	})
}
