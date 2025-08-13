package progress

import (
	"sync"
	"time"
)

// Manager provides advanced progress tracking with moving averages and chunk tracking.
type Manager struct {
	mu              sync.RWMutex
	downloadedBytes int64
	totalBytes      int64
	speed           int64
	timeRemaining   time.Duration
	chunksComplete  int
	totalChunks     int
	startTime       time.Time
	lastUpdateTime  time.Time
	updateChan      chan ProgressInfo
	rateLimiter     *time.Ticker
	movingAvgWindow []int64
}

// ProgressInfo represents detailed progress information.
type ProgressInfo struct {
	TotalBytes      int64
	DownloadedBytes int64
	Speed           int64
	TimeRemaining   time.Duration
	ChunksComplete  int
	TotalChunks     int
	StartTime       time.Time
	LastUpdateTime  time.Time
}

// NewManager creates a new progress manager.
func NewManager() *Manager {
	return &Manager{
		updateChan:      make(chan ProgressInfo, 100),
		movingAvgWindow: make([]int64, 0, 10), // Keep last 10 speed samples
	}
}

// Update updates the download progress.
func (m *Manager) Update(downloaded, total int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()

	// Initialize start time if first update
	if m.startTime.IsZero() {
		m.startTime = now
		m.lastUpdateTime = now
	}

	// Calculate speed if time has passed
	timeDiff := now.Sub(m.lastUpdateTime).Seconds()
	if timeDiff > 0 {
		bytesDownloaded := downloaded - m.downloadedBytes
		currentSpeed := int64(float64(bytesDownloaded) / timeDiff)

		// Add to moving average window
		m.movingAvgWindow = append(m.movingAvgWindow, currentSpeed)
		if len(m.movingAvgWindow) > 10 {
			m.movingAvgWindow = m.movingAvgWindow[1:]
		}

		m.speed = m.calculateSpeed()
	}

	m.downloadedBytes = downloaded
	m.totalBytes = total
	m.lastUpdateTime = now
	m.timeRemaining = m.calculateETA()

	// Send update through channel (non-blocking)
	select {
	case m.updateChan <- m.getProgressInfo():
	default:
	}
}

// GetProgress returns the current progress.
func (m *Manager) GetProgress() ProgressInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.getProgressInfo()
}

// getProgressInfo returns the current progress info (must be called with lock held).
func (m *Manager) getProgressInfo() ProgressInfo {
	return ProgressInfo{
		TotalBytes:      m.totalBytes,
		DownloadedBytes: m.downloadedBytes,
		Speed:           m.speed,
		TimeRemaining:   m.timeRemaining,
		ChunksComplete:  m.chunksComplete,
		TotalChunks:     m.totalChunks,
		StartTime:       m.startTime,
		LastUpdateTime:  m.lastUpdateTime,
	}
}

// Start starts the progress manager with rate limiting.
func (m *Manager) Start() {
	// Create rate limiter to update UI at most 10 times per second
	m.rateLimiter = time.NewTicker(100 * time.Millisecond)
}

// Stop stops the progress manager.
func (m *Manager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.rateLimiter != nil {
		m.rateLimiter.Stop()
		m.rateLimiter = nil
	}

	close(m.updateChan)
}

// UpdateChunks updates chunk completion status.
func (m *Manager) UpdateChunks(complete, total int) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.chunksComplete = complete
	m.totalChunks = total
}

// calculateSpeed calculates the moving average speed.
func (m *Manager) calculateSpeed() int64 {
	if len(m.movingAvgWindow) == 0 {
		return 0
	}

	var sum int64
	for _, speed := range m.movingAvgWindow {
		sum += speed
	}

	return sum / int64(len(m.movingAvgWindow))
}

// calculateETA calculates estimated time remaining.
func (m *Manager) calculateETA() time.Duration {
	if m.speed <= 0 || m.totalBytes <= 0 {
		return 0
	}

	remainingBytes := m.totalBytes - m.downloadedBytes
	if remainingBytes <= 0 {
		return 0
	}

	secondsRemaining := float64(remainingBytes) / float64(m.speed)

	return time.Duration(secondsRemaining) * time.Second
}
