package common

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestParseInterval(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    time.Duration
		wantErr bool
	}{
		{"Valid seconds", "5s", 5 * time.Second, false},
		{"Valid milliseconds", "500ms", 500 * time.Millisecond, false},
		{"Valid minutes", "2m", 2 * time.Minute, false},
		{"Valid hours", "1h", 1 * time.Hour, false},
		{"Complex duration", "1h30m", 90 * time.Minute, false},
		{"Invalid format", "invalid", 0, true},
		{"Zero duration", "0s", 0, true},
		{"Negative duration", "-5s", 0, true},
		{"Empty string", "", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseInterval(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseInterval() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ParseInterval() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStartPeriodicTask(t *testing.T) {
	t.Run("Task executes periodically", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
		defer cancel()

		callCount := 0
		task := func() error {
			callCount++
			return nil
		}

		err := StartPeriodicTask(ctx, "100ms", task)
		if err != nil {
			t.Fatalf("StartPeriodicTask() error = %v", err)
		}

		if callCount < 2 {
			t.Errorf("Task should execute at least 2 times, got %d", callCount)
		}
	})

	t.Run("Invalid interval returns error", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		task := func() error { return nil }

		err := StartPeriodicTask(ctx, "invalid", task)
		if err == nil {
			t.Error("StartPeriodicTask() expected error for invalid interval")
		}
	})

	t.Run("Context cancellation stops task", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		callCount := 0
		task := func() error {
			callCount++
			if callCount >= 2 {
				cancel()
			}
			return nil
		}

		err := StartPeriodicTask(ctx, "50ms", task)
		if err != nil {
			t.Fatalf("StartPeriodicTask() error = %v", err)
		}

		if callCount < 2 {
			t.Errorf("Task should execute at least 2 times before cancellation, got %d", callCount)
		}
	})
}

func TestRunOnce(t *testing.T) {
	t.Run("Successful task execution", func(t *testing.T) {
		executed := false
		task := func() error {
			executed = true
			return nil
		}

		err := RunOnce(task)
		if err != nil {
			t.Errorf("RunOnce() error = %v", err)
		}
		if !executed {
			t.Error("Task was not executed")
		}
	})

	t.Run("Task error is propagated", func(t *testing.T) {
		expectedErr := context.DeadlineExceeded
		task := func() error {
			return expectedErr
		}

		err := RunOnce(task)
		if err != expectedErr {
			t.Errorf("RunOnce() error = %v, want %v", err, expectedErr)
		}
	})
}

func TestRunOnceOrPeriodic(t *testing.T) {
	t.Run("once mode", func(t *testing.T) {
		callCount := 0
		task := func() error {
			callCount++
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		defer cancel()

		err := RunOnceOrPeriodic(ctx, true, "10ms", task)
		if err != nil {
			t.Fatalf("RunOnceOrPeriodic() error = %v", err)
		}

		if callCount != 1 {
			t.Errorf("RunOnceOrPeriodic(once=true) called task %d times, want 1", callCount)
		}
	})

	t.Run("periodic mode", func(t *testing.T) {
		callCount := 0
		task := func() error {
			callCount++
			return nil
		}

		ctx, cancel := context.WithTimeout(context.Background(), 150*time.Millisecond)
		defer cancel()

		err := RunOnceOrPeriodic(ctx, false, "50ms", task)
		if err != nil {
			t.Fatalf("RunOnceOrPeriodic() error = %v", err)
		}

		// Should be called at least 2 times in 150ms with 50ms interval
		if callCount < 2 {
			t.Errorf("RunOnceOrPeriodic(once=false) called task %d times, want at least 2", callCount)
		}
	})

	t.Run("once mode with task error", func(t *testing.T) {
		expectedErr := errors.New("task failed")
		task := func() error {
			return expectedErr
		}

		ctx := context.Background()
		err := RunOnceOrPeriodic(ctx, true, "10ms", task)
		if err != expectedErr {
			t.Errorf("RunOnceOrPeriodic(once=true) error = %v, want %v", err, expectedErr)
		}
	})

	t.Run("periodic mode with invalid interval", func(t *testing.T) {
		task := func() error {
			return nil
		}

		ctx := context.Background()
		err := RunOnceOrPeriodic(ctx, false, "invalid", task)
		if err == nil {
			t.Error("RunOnceOrPeriodic() expected error for invalid interval")
		}
	})
}
