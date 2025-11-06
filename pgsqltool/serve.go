package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/lib/pq"
	"github.com/sandrolain/eventkit/pkg/common"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
)

func serveCommand() *cobra.Command {
	var (
		connStr string
		channel string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "LISTEN to PostgreSQL channel and log notifications",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

			reportProblem := func(ev pq.ListenerEventType, err error) {
				if err != nil {
					slog.Error("Listener problem", "event", ev, "error", err)
				}
			}

			listener := pq.NewListener(connStr, 10*time.Second, time.Minute, reportProblem)
			defer func() {
				if err := listener.Close(); err != nil {
					slog.Error("Failed to close listener", "error", err)
				}
			}()

			if err := listener.Listen(channel); err != nil {
				return fmt.Errorf("LISTEN error: %w", err)
			}

			logger := toolutil.Logger()
			logger.Info("Listening to PostgreSQL", "channel", channel)

			for {
				select {
				case <-ctx.Done():
					logger.Info("Shutting down gracefully")
					return nil
				case n := <-listener.Notify:
					if n == nil {
						continue
					}
					sections := []toolutil.MessageSection{
						{Title: "Channel", Items: []toolutil.KV{{Key: "Name", Value: n.Channel}}},
						{Title: "Meta", Items: []toolutil.KV{
							{Key: "PID", Value: fmt.Sprintf("%d", n.BePid)},
						}},
					}
					ct := toolutil.GuessMIME([]byte(n.Extra))
					toolutil.PrintColoredMessage("PostgreSQL NOTIFY", sections, []byte(n.Extra), ct)
				case <-time.After(90 * time.Second):
					// Ping to keep connection alive
					if err := listener.Ping(); err != nil {
						logger.Error("Ping failed", "error", err)
						return fmt.Errorf("connection lost: %w", err)
					}
				}
			}
		},
	}

	cmd.Flags().StringVar(&connStr, "conn", "postgres://user:pass@localhost:5432/postgres?sslmode=disable", "PostgreSQL connection string")
	cmd.Flags().StringVar(&channel, "channel", "test_channel", "LISTEN channel name")

	return cmd
}
