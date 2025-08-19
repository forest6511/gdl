//go:build !windows

package main

import (
	"context"
	"os"
	"syscall"
	"testing"
	"time"

	"github.com/forest6511/godl/pkg/ui"
)

func TestSignalHandlingUnix(t *testing.T) {
	// Skip all signal handler tests when running with race detector
	// Signal handler tests are inherently racy due to asynchronous signal handling
	if raceEnabled {
		t.Skip("Skipping all signal handler tests with race detector enabled")
	}

	// Save original formatter
	originalFormatter := formatter
	defer func() {
		formatter = originalFormatter
	}()

	t.Run("SIGINT handling", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg := &config{
			quiet:       false,
			interactive: false,
		}

		// Create a mock formatter
		formatter = ui.NewFormatter()

		// Start the signal handler
		handleInterruption(ctx, cancel, cfg)

		// Give the goroutine time to start
		time.Sleep(50 * time.Millisecond)

		// Send SIGINT signal to the current process
		process, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Fatalf("Failed to find process: %v", err)
		}

		err = process.Signal(syscall.SIGINT)
		if err != nil {
			t.Fatalf("Failed to send signal: %v", err)
		}

		// Wait for context to be cancelled
		select {
		case <-ctx.Done():
			// Success - context was cancelled
		case <-time.After(2 * time.Second):
			t.Error("Context was not cancelled after SIGINT")
		}
	})

	t.Run("SIGTERM handling", func(t *testing.T) {
		// Skip this test in CI environment as it can be flaky
		if os.Getenv("CI") != "" {
			t.Skip("Skipping SIGTERM test in CI environment")
		}

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg := &config{
			quiet:       false,
			interactive: false,
		}

		// Create a mock formatter
		formatter = ui.NewFormatter()

		// Start the signal handler
		handleInterruption(ctx, cancel, cfg)

		// Give the goroutine time to start
		time.Sleep(50 * time.Millisecond)

		// Send SIGTERM signal to the current process
		process, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Fatalf("Failed to find process: %v", err)
		}

		err = process.Signal(syscall.SIGTERM)
		if err != nil {
			t.Fatalf("Failed to send signal: %v", err)
		}

		// Wait for context to be cancelled
		select {
		case <-ctx.Done():
			// Success - context was cancelled
		case <-time.After(2 * time.Second):
			t.Error("Context was not cancelled after SIGTERM")
		}
	})

	t.Run("Quiet mode", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		cfg := &config{
			quiet:       true,
			interactive: false,
		}

		// Start the signal handler
		handleInterruption(ctx, cancel, cfg)

		// Give the goroutine time to start
		time.Sleep(50 * time.Millisecond)

		// Send SIGINT signal to the current process
		process, err := os.FindProcess(os.Getpid())
		if err != nil {
			t.Fatalf("Failed to find process: %v", err)
		}

		err = process.Signal(syscall.SIGINT)
		if err != nil {
			t.Fatalf("Failed to send signal: %v", err)
		}

		// Wait for context to be cancelled
		select {
		case <-ctx.Done():
			// Success - context was cancelled silently
		case <-time.After(2 * time.Second):
			t.Error("Context was not cancelled in quiet mode")
		}
	})
}

// Test helper to check if signal handling works with different configurations
func TestHandleInterruptionWithDifferentConfigs(t *testing.T) {
	configs := []struct {
		name string
		cfg  *config
	}{
		{
			name: "Default config",
			cfg: &config{
				quiet:       false,
				interactive: false,
			},
		},
		{
			name: "Quiet mode",
			cfg: &config{
				quiet:       true,
				interactive: false,
			},
		},
		{
			name: "Interactive mode",
			cfg: &config{
				quiet:       false,
				interactive: true,
			},
		},
	}

	for _, tc := range configs {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			// Set formatter if needed
			if formatter == nil {
				formatter = ui.NewFormatter()
			}

			// Start the signal handler
			handleInterruption(ctx, cancel, tc.cfg)

			// Give the goroutine time to start
			time.Sleep(50 * time.Millisecond)

			// Send signal
			process, err := os.FindProcess(os.Getpid())
			if err != nil {
				t.Fatalf("Failed to find process: %v", err)
			}

			err = process.Signal(syscall.SIGINT)
			if err != nil {
				t.Fatalf("Failed to send signal: %v", err)
			}

			// Wait for context to be cancelled
			select {
			case <-ctx.Done():
				// Success
			case <-time.After(2 * time.Second):
				t.Errorf("Context was not cancelled for config %s", tc.name)
			}
		})
	}
}
