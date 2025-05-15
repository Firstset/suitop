package util

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
)

// SetupSignalHandler sets up a signal handler that cancels the context when an interrupt is received.
// Returns a cleanup function that should be deferred by the caller.
func SetupSignalHandler(cancel context.CancelFunc) func() {
	// Handle OS signals for graceful shutdown
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)
	stopChan := make(chan struct{})

	go func() {
		sig := <-sigs
		log.Printf("Received signal: %s, shutting down gracefully...", sig)
		cancel()
		close(stopChan)
	}()

	return func() {
		signal.Stop(sigs)
		close(sigs)
		<-stopChan // Wait for the signal handler to complete if possible
	}
}
