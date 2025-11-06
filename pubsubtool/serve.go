package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pubsub "cloud.google.com/go/pubsub/v2"
	"github.com/sandrolain/eventkit/pkg/common"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
)

func serveCommand() *cobra.Command {
	var (
		subProject string
		subSub     string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Subscribe and log Pub/Sub messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

			client, err := pubsub.NewClient(ctx, subProject)
			if err != nil {
				return fmt.Errorf("Pub/Sub client error: %w", err)
			}
			defer func() {
				if err := client.Close(); err != nil {
					slog.Error("Failed to close Pub/Sub client", "error", err)
				}
			}()

			sub := client.Subscriber(subSub)

			logger := toolutil.Logger()
			logger.Info("Listening to Pub/Sub", "project", subProject, "subscription", subSub)

			err = sub.Receive(ctx, func(ctx context.Context, m *pubsub.Message) {
				var attrItems []toolutil.KV
				for k, v := range m.Attributes {
					attrItems = append(attrItems, toolutil.KV{Key: k, Value: v})
				}

				sections := []toolutil.MessageSection{
					{Title: "Subscription", Items: []toolutil.KV{{Key: "Name", Value: subSub}}},
					{Title: "Meta", Items: []toolutil.KV{{Key: "PublishTime", Value: m.PublishTime.Format(time.RFC3339)}}},
					{Title: "Attributes", Items: attrItems},
				}

				ct := toolutil.GuessMIME(m.Data)
				toolutil.PrintColoredMessage("Pub/Sub", sections, m.Data, ct)

				m.Ack()
			})

			if err != nil {
				return fmt.Errorf("receive error: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&subProject, "project", "test-project", "Google Cloud Project ID")
	cmd.Flags().StringVar(&subSub, "subscription", "test-sub", "Pub/Sub subscription ID")

	return cmd
}
