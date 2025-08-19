//go:build !windows

package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/forest6511/gdl/pkg/ui"
)

// handleInterruption sets up graceful interruption handling for Unix systems.
func handleInterruption(ctx context.Context, cancel context.CancelFunc, cfg *config) {
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan

		if !cfg.quiet {
			fmt.Println() // New line after progress bar
			formatter.PrintMessage(
				ui.MessageWarning,
				"Received %s signal, initiating graceful shutdown...",
				sig,
			)

			if cfg.interactive && formatter != nil {
				proceed, err := formatter.ConfirmPrompt("Force immediate termination?", false)
				if err == nil && proceed {
					formatter.PrintMessage(ui.MessageError, "Forcing immediate termination")
					os.Exit(130) // Standard exit code for SIGINT
				}
			}
		}

		cancel()
	}()
}
