package main

import (
	"context"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/sandrolain/eventkit/pkg/common"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/segmentio/kafka-go"
	"github.com/spf13/cobra"
)

func serveCommand() *cobra.Command {
	var (
		subBrokers string
		subTopic   string
		subGroup   string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Consume messages and print them",
		RunE: func(cmd *cobra.Command, args []string) error {
			r := kafka.NewReader(kafka.ReaderConfig{
				Brokers:  strings.Split(subBrokers, ","),
				GroupID:  subGroup,
				Topic:    subTopic,
				MinBytes: 1,
				MaxBytes: 10e6,
			})
			defer func() {
				if err := r.Close(); err != nil {
					slog.Error("Failed to close Kafka reader", "error", err)
				}
			}()

			logger := toolutil.Logger()
			logger.Info("Consuming from Kafka", "brokers", subBrokers, "topic", subTopic, "group", subGroup)

			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

			for {
				select {
				case <-ctx.Done():
					logger.Info("Shutting down gracefully")
					return nil
				default:
					m, err := r.ReadMessage(context.Background())
					if err != nil {
						logger.Error("Error reading message", "error", err)
						return err
					}

					// Build sections with metadata
					var headerItems []toolutil.KV
					for _, h := range m.Headers {
						headerItems = append(headerItems, toolutil.KV{Key: h.Key, Value: string(h.Value)})
					}
					sections := []toolutil.MessageSection{
						{Title: "Topic", Items: []toolutil.KV{{Key: "Name", Value: m.Topic}}},
						{Title: "Meta", Items: []toolutil.KV{
							{Key: "Partition", Value: strconv.Itoa(m.Partition)},
							{Key: "Offset", Value: strconv.FormatInt(m.Offset, 10)},
							{Key: "Time", Value: m.Time.Format(time.RFC3339)},
						}},
						{Title: "Key", Items: []toolutil.KV{{Key: "Value", Value: string(m.Key)}}},
						{Title: "Headers", Items: headerItems},
					}
					ct := toolutil.GuessMIME(m.Value)
					toolutil.PrintColoredMessage("Kafka", sections, m.Value, ct)
				}
			}
		},
	}

	cmd.Flags().StringVar(&subBrokers, "brokers", "localhost:9092", "Kafka brokers (comma-separated)")
	cmd.Flags().StringVar(&subTopic, "topic", "test", "Kafka topic")
	cmd.Flags().StringVar(&subGroup, "group", "", "Kafka consumer group")

	return cmd
}
