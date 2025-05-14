package util

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

// SetupSignalHandler registers for SIGINT and SIGTERM and calls the cancel func when a signal is received.
// It returns a function that can be called to stop the signal handler goroutine (e.g. on clean shutdown).
func SetupSignalHandler(cancel context.CancelFunc) func() {
	sigs := make(chan os.Signal, 1)
	done := make(chan struct{}, 1)

	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		select {
		case sig := <-sigs:
			fmt.Printf("\nReceived signal: %s, shutting down gracefully...\n", sig)
			cancel()
		case <-done:
			// Clean exit for the goroutine
		}
	}()

	return func() {
		signal.Stop(sigs) // Unregister signal notifications
		close(done)       // Signal the goroutine to exit
	}
}
