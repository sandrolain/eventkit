package common

import (
	"os"
	"syscall"
	"testing"
	"time"
)

func TestSetupGracefulShutdown(t *testing.T) {
	t.Run("Context is created", func(t *testing.T) {
		ctx, cancel := SetupGracefulShutdown()
		defer cancel()

		if ctx == nil {
			t.Error("SetupGracefulShutdown() returned nil context")
		}

		select {
		case <-ctx.Done():
			t.Error("Context should not be cancelled immediately")
		default:
			// Expected: context is not done
		}
	})

	t.Run("Cancel function works", func(t *testing.T) {
		ctx, cancel := SetupGracefulShutdown()

		cancel()

		select {
		case <-ctx.Done():
			// Expected: context is cancelled
		case <-time.After(100 * time.Millisecond):
			t.Error("Context was not cancelled after calling cancel()")
		}
	})

	t.Run("Signal cancels context", func(t *testing.T) {
		// This test is tricky as it involves actual signal handling
		// We'll test that the context is properly set up
		ctx, cancel := SetupGracefulShutdown()
		defer cancel()

		// Send SIGTERM to ourselves in a goroutine
		go func() {
			time.Sleep(50 * time.Millisecond)
			p, _ := os.FindProcess(os.Getpid())
			_ = p.Signal(syscall.SIGTERM)
		}()

		select {
		case <-ctx.Done():
			// Expected: context is cancelled by signal
		case <-time.After(200 * time.Millisecond):
			t.Error("Context was not cancelled by signal")
		}
	})
}
