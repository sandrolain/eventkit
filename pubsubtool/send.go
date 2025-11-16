package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	pubsub "cloud.google.com/go/pubsub/v2"
	"github.com/sandrolain/eventkit/pkg/testpayload"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
)

func sendCommand() *cobra.Command {
	var (
		sendProject    string
		sendTopic      string
		sendPayload    string
		sendMIME       string
		seed           int64
		allowFileReads bool
		templateVars   []string
		fileRoot       string
		cacheFiles     bool
		sendInterval   string
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Publish periodic Pub/Sub messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			client, err := pubsub.NewClient(ctx, sendProject)
			if err != nil {
				return fmt.Errorf("Pub/Sub client error: %w", err)
			}
			defer func() {
				if err := client.Close(); err != nil {
					slog.Error("Failed to close Pub/Sub client", "error", err)
				}
			}()

			publisher := client.Publisher(sendTopic)
			defer publisher.Stop()

			dur, err := time.ParseDuration(sendInterval)
			if err != nil {
				return fmt.Errorf("invalid interval: %w", err)
			}
			ticker := time.NewTicker(dur)
			defer ticker.Stop()

			logger := toolutil.Logger()
			if seed != 0 {
				testpayload.SeedRandom(seed)
			}
			testpayload.SetAllowFileReads(allowFileReads)
			// file cache
			testpayload.SetFileCacheEnabled(cacheFiles)
			testpayload.SetFileRoot(fileRoot)
			varsMap, errVars := toolutil.ParseTemplateVars(templateVars)
			if errVars != nil {
				return fmt.Errorf("invalid template-var: %w", errVars)
			}
			testpayload.SetTemplateVars(varsMap)
			logger.Info("Publishing to Pub/Sub", "project", sendProject, "topic", sendTopic, "interval", sendInterval)

			for range ticker.C {
				body, _, err := toolutil.BuildPayload(sendPayload, sendMIME)
				if err != nil {
					logger.Error("Failed to build payload", "error", err)
					continue
				}

				result := publisher.Publish(ctx, &pubsub.Message{Data: body})
				id, err := result.Get(ctx)
				if err != nil {
					logger.Error("Failed to send message", "error", err)
				} else {
					logger.Info("Message sent", "id", id, "bytes", len(body))
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&sendProject, "project", "test-project", "Google Cloud Project ID")
	cmd.Flags().StringVar(&sendTopic, "topic", "test-topic", "Pub/Sub topic ID")
	toolutil.AddPayloadFlags(cmd, &sendPayload, "Hello, PubSub!", &sendMIME, toolutil.CTText)
	toolutil.AddIntervalFlag(cmd, &sendInterval, "5s")
	toolutil.AddSeedFlag(cmd, &seed)
	toolutil.AddAllowFileReadsFlag(cmd, &allowFileReads)
	toolutil.AddTemplateVarFlag(cmd, &templateVars)
	toolutil.AddFileRootFlag(cmd, &fileRoot)
	toolutil.AddFileCacheFlag(cmd, &cacheFiles)

	return cmd
}
