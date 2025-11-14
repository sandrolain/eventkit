package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/segmentio/kafka-go"
	"github.com/spf13/cobra"

	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
)

func sendCommand() *cobra.Command {
	var (
		sendBrokers  string
		sendTopic    string
		sendPayload  string
		sendMIME     string
		sendInterval string
		headers      []string
		openDelim    string
		closeDelim   string
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Produce periodic Kafka messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			dur, err := time.ParseDuration(sendInterval)
			if err != nil {
				return fmt.Errorf("invalid interval: %w", err)
			}

			w := kafka.NewWriter(kafka.WriterConfig{
				Brokers: strings.Split(sendBrokers, ","),
				Topic:   sendTopic,
			})
			defer func() {
				if err := w.Close(); err != nil {
					slog.Error("Failed to close Kafka writer", "error", err)
				}
			}()

			ticker := time.NewTicker(dur)
			defer ticker.Stop()

			headerMap, err := toolutil.ParseHeadersWithDelimiters(headers, openDelim, closeDelim)
			if err != nil {
				return fmt.Errorf("invalid headers: %w", err)
			}

			logger := toolutil.Logger()
			logger.Info("Producing to Kafka", "brokers", sendBrokers, "topic", sendTopic, "interval", sendInterval)

			for range ticker.C {
				body, _, err := toolutil.BuildPayloadWithDelimiters(sendPayload, sendMIME, openDelim, closeDelim)
				if err != nil {
					logger.Error("Failed to build payload", "error", err)
					continue
				}
				msg := kafka.Message{Value: body}
				for k, v := range headerMap {
					msg.Headers = append(msg.Headers, kafka.Header{Key: k, Value: []byte(v)})
				}

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				err = w.WriteMessages(ctx, msg)
				cancel()
				if err != nil {
					logger.Error("Failed to send message", "error", err)
				} else {
					logger.Info("Message sent", "bytes", len(body))
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&sendBrokers, "brokers", "localhost:9092", "Kafka brokers (comma-separated)")
	cmd.Flags().StringVar(&sendTopic, "topic", "test", "Kafka topic")
	toolutil.AddPayloadFlags(cmd, &sendPayload, "Hello, Kafka!", &sendMIME, toolutil.CTText)
	toolutil.AddIntervalFlag(cmd, &sendInterval, "5s")
	toolutil.AddHeadersFlag(cmd, &headers)
	toolutil.AddTemplateDelimiterFlags(cmd, &openDelim, &closeDelim)

	return cmd
}
