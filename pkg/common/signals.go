package common

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

// SetupGracefulShutdown creates a context that will be cancelled on SIGINT or SIGTERM.
// Returns the context and a cleanup function that should be deferred.
// This provides consistent signal handling across all tools.
func SetupGracefulShutdown() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)

	go func() {
		sig := <-sigc
		slog.Info("Received signal, shutting down gracefully", "signal", sig)
		cancel()
	}()

	return ctx, cancel
}

// WaitForShutdown blocks until SIGINT or SIGTERM is received.
// Use this in serve commands that don't need a cancellable context.
func WaitForShutdown() {
	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, os.Interrupt, syscall.SIGTERM)
	sig := <-sigc
	slog.Info("Received signal, shutting down", "signal", sig)
}
