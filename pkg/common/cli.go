package common

import (
	"context"
	"fmt"
	"os"
	"time"
)

// ParseInterval parses a duration string and returns a time.Duration.
// Returns an error if the interval is invalid.
func ParseInterval(interval string) (time.Duration, error) {
	dur, err := time.ParseDuration(interval)
	if err != nil {
		return 0, fmt.Errorf("invalid interval: %w", err)
	}
	if dur <= 0 {
		return 0, fmt.Errorf("interval must be positive")
	}
	return dur, nil
}

// StartPeriodicTask executes the given task function periodically at the specified interval.
// The task runs in a goroutine on each tick. The function blocks until the context is cancelled.
// If the context is cancelled, the ticker is stopped and the function returns nil.
func StartPeriodicTask(ctx context.Context, interval string, task func() error) error {
	dur, err := ParseInterval(interval)
	if err != nil {
		return err
	}

	ticker := time.NewTicker(dur)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-ticker.C:
			go func() {
				if err := task(); err != nil {
					fmt.Fprintf(os.Stderr, "Task error: %v\n", err)
				}
			}()
		}
	}
}

// RunOnce executes the task function once immediately.
// Returns an error if the task fails.
func RunOnce(task func() error) error {
	return task()
}
