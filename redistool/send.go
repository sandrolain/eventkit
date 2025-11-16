package main

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sandrolain/eventkit/pkg/testpayload"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
)

func sendCommand() *cobra.Command {
	var (
		sendAddr       string
		sendChannel    string
		sendStream     string
		sendPayload    string
		sendMIME       string
		seed           int64
		allowFileReads bool
		templateVars   []string
		fileRoot       string
		cacheFiles     bool
		sendInterval   string
		sendDataKey    string
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Publish periodic messages to a Redis channel or stream",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := context.Background()
			rdb := redis.NewClient(&redis.Options{Addr: sendAddr})
			defer func() {
				if err := rdb.Close(); err != nil {
					slog.Error("Failed to close Redis client", "error", err)
				}
			}()

			dur, err := time.ParseDuration(sendInterval)
			if err != nil {
				return fmt.Errorf("invalid interval: %w", err)
			}
			ticker := time.NewTicker(dur)
			defer ticker.Stop()

			mode := "channel"
			if sendStream != "" {
				mode = "stream"
			}

			logger := toolutil.Logger()
			if seed != 0 {
				testpayload.SeedRandom(seed)
			}
			testpayload.SetAllowFileReads(allowFileReads)
			testpayload.SetFileRoot(fileRoot)
			testpayload.SetFileCacheEnabled(cacheFiles)
			varsMap, errVars := toolutil.ParseTemplateVars(templateVars)
			if errVars != nil {
				return fmt.Errorf("invalid template-var: %w", errVars)
			}
			testpayload.SetTemplateVars(varsMap)
			logger.Info("Sending to Redis", "address", sendAddr, "mode", mode, "interval", sendInterval)

			for range ticker.C {
				body, _, err := toolutil.BuildPayload(sendPayload, sendMIME)
				if err != nil {
					logger.Error("Failed to build payload", "error", err)
					continue
				}
				switch mode {
				case "stream":
					fields := map[string]interface{}{sendDataKey: body}
					res := rdb.XAdd(ctx, &redis.XAddArgs{Stream: sendStream, Values: fields})
					if err := res.Err(); err != nil {
						logger.Error("XAdd error", "error", err)
					} else {
						logger.Info("Message sent to stream", "stream", sendStream, "id", res.Val())
					}
				default: // channel
					if err := rdb.Publish(ctx, sendChannel, body).Err(); err != nil {
						logger.Error("Publish error", "error", err)
					} else {
						logger.Info("Message sent to channel", "channel", sendChannel, "bytes", len(body))
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&sendAddr, "address", "localhost:6379", "Redis address")
	cmd.Flags().StringVar(&sendChannel, "channel", "test", "Redis channel (for pub-sub mode)")
	cmd.Flags().StringVar(&sendStream, "stream", "", "Redis stream (if set, sends to stream)")
	cmd.Flags().StringVar(&sendDataKey, "dataKey", "data", "Field name holding data in stream messages")
	toolutil.AddPayloadFlags(cmd, &sendPayload, "Hello, Redis!", &sendMIME, toolutil.CTText)
	toolutil.AddIntervalFlag(cmd, &sendInterval, "5s")
	toolutil.AddSeedFlag(cmd, &seed)
	toolutil.AddAllowFileReadsFlag(cmd, &allowFileReads)
	toolutil.AddFileCacheFlag(cmd, &cacheFiles)
	toolutil.AddTemplateVarFlag(cmd, &templateVars)
	toolutil.AddFileRootFlag(cmd, &fileRoot)

	return cmd
}
