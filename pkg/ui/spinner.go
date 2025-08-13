package ui

import (
	"fmt"
	"sync"
	"time"
)

// Spinner represents a console spinner.
type Spinner struct {
	frames   []string
	message  string
	interval time.Duration
	stop     chan bool
	mu       sync.RWMutex
	running  bool
}

// SpinnerStyle represents different spinner styles.
type SpinnerStyle int

const (
	SpinnerStyleDots SpinnerStyle = iota
	SpinnerStyleLine
	SpinnerStyleCircle
	SpinnerStyleSquare
	SpinnerStyleArrow
	SpinnerStyleBounce
)

// getSpinnerFrames returns frames for the specified style.
func getSpinnerFrames(style SpinnerStyle) []string {
	switch style {
	case SpinnerStyleDots:
		return []string{"⣾", "⣽", "⣻", "⢿", "⡿", "⣟", "⣯", "⣷"}
	case SpinnerStyleLine:
		return []string{"-", "\\", "|", "/"}
	case SpinnerStyleCircle:
		return []string{"◐", "◓", "◑", "◒"}
	case SpinnerStyleSquare:
		return []string{"◰", "◳", "◲", "◱"}
	case SpinnerStyleArrow:
		return []string{"←", "↖", "↑", "↗", "→", "↘", "↓", "↙"}
	case SpinnerStyleBounce:
		return []string{"⠁", "⠂", "⠄", "⡀", "⢀", "⠠", "⠐", "⠈"}
	default:
		return []string{"-", "\\", "|", "/"}
	}
}

// NewSpinner creates a new spinner with the specified style.
func NewSpinner(style SpinnerStyle, message string) *Spinner {
	return &Spinner{
		frames:   getSpinnerFrames(style),
		message:  message,
		interval: 100 * time.Millisecond,
		stop:     make(chan bool),
	}
}

// SetMessage updates the spinner message.
func (s *Spinner) SetMessage(message string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.message = message
}

// SetInterval sets the spinner interval.
func (s *Spinner) SetInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.interval = interval
}

// Start starts the spinner.
func (s *Spinner) Start() {
	s.mu.Lock()

	if s.running {
		s.mu.Unlock()
		return
	}

	s.running = true
	s.mu.Unlock()

	go func() {
		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		frameIndex := 0

		// Hide cursor
		fmt.Print("\033[?25l")

		for {
			select {
			case <-s.stop:
				// Clear line and show cursor
				fmt.Print("\r\033[K")
				fmt.Print("\033[?25h")

				return
			case <-ticker.C:
				s.mu.RLock()
				frame := s.frames[frameIndex]
				message := s.message
				s.mu.RUnlock()

				// Clear line and print spinner
				fmt.Printf("\r\033[K%s %s", frame, message)

				frameIndex = (frameIndex + 1) % len(s.frames)
			}
		}
	}()
}

// Stop stops the spinner.
func (s *Spinner) Stop() {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.running {
		s.stop <- true

		s.running = false
	}
}

// StopWithMessage stops the spinner and displays a final message.
func (s *Spinner) StopWithMessage(message string) {
	s.Stop()
	fmt.Printf("\r\033[K%s\n", message)
}

// StopWithSuccess stops the spinner with a success message.
func (s *Spinner) StopWithSuccess(message string) {
	s.Stop()
	fmt.Printf("\r\033[K✅ %s\n", message)
}

// StopWithError stops the spinner with an error message.
func (s *Spinner) StopWithError(message string) {
	s.Stop()
	fmt.Printf("\r\033[K❌ %s\n", message)
}

// StopWithWarning stops the spinner with a warning message.
func (s *Spinner) StopWithWarning(message string) {
	s.Stop()
	fmt.Printf("\r\033[K⚠️  %s\n", message)
}
