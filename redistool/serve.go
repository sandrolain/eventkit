package main

import (
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sandrolain/eventkit/pkg/common"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
)

func serveCommand() *cobra.Command {
	var (
		subAddr     string
		subChannel  string
		subStream   string
		subGroup    string
		subConsumer string
		subDataKey  string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Subscribe to a channel or consume a stream and log messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

			rdb := redis.NewClient(&redis.Options{Addr: subAddr})
			defer func() {
				if err := rdb.Close(); err != nil {
					slog.Error("Failed to close Redis client", "error", err)
				}
			}()

			logger := toolutil.Logger()

			if subStream != "" {
				logger.Info("Listening to Redis stream", "stream", subStream, "address", subAddr)
				lastID := "$"
				useGroup := subGroup != "" && subConsumer != ""
				if useGroup {
					// Create group idempotently; ignore error if exists
					if err := rdb.XGroupCreateMkStream(ctx, subStream, subGroup, "0").Err(); err != nil {
						logger.Warn("Group creation warning (may already exist)", "error", err)
					}
					lastID = ">"
				}

				for {
					select {
					case <-ctx.Done():
						logger.Info("Shutting down gracefully")
						return nil
					default:
						var res []redis.XStream
						var err error
						if useGroup {
							res, err = rdb.XReadGroup(ctx, &redis.XReadGroupArgs{
								Group:    subGroup,
								Consumer: subConsumer,
								Streams:  []string{subStream, lastID},
								Count:    1,
								Block:    1000,
								NoAck:    false,
							}).Result()
						} else {
							res, err = rdb.XRead(ctx, &redis.XReadArgs{
								Streams: []string{subStream, lastID},
								Count:   1,
								Block:   1000,
							}).Result()
						}
						if err != nil {
							if err == redis.Nil {
								continue
							}
							logger.Error("Error reading from stream", "error", err)
							time.Sleep(2 * time.Second)
							continue
						}

						for _, xstream := range res {
							for _, xmsg := range xstream.Messages {
								// Metadata and fields
								var items []toolutil.KV
								items = append(items, toolutil.KV{Key: "ID", Value: xmsg.ID})
								for k, v := range xmsg.Values {
									items = append(items, toolutil.KV{Key: k, Value: fmt.Sprintf("%v", v)})
								}
								sections := []toolutil.MessageSection{
									{Title: "Stream", Items: []toolutil.KV{{Key: "Name", Value: xstream.Stream}}},
									{Title: "Message", Items: items},
								}

								// Extract body
								var data []byte
								if v, ok := xmsg.Values[subDataKey]; ok {
									switch vv := v.(type) {
									case string:
										data = []byte(vv)
									case []byte:
										data = vv
									default:
										data = []byte(fmt.Sprintf("%v", vv))
									}
								}

								ct := toolutil.GuessMIME(data)
								toolutil.PrintColoredMessage("Redis Stream", sections, data, ct)

								if useGroup {
									if err := rdb.XAck(ctx, subStream, subGroup, xmsg.ID).Err(); err != nil {
										logger.Error("Failed to ack message", "error", err)
									}
								} else {
									lastID = xmsg.ID
								}
							}
						}
					}
				}
			}

			// Channel mode
			logger.Info("Listening to Redis channel", "channel", subChannel, "address", subAddr)
			pubsub := rdb.Subscribe(ctx, subChannel)
			defer func() {
				if err := pubsub.Close(); err != nil {
					logger.Error("Failed to close pubsub", "error", err)
				}
			}()

			ch := pubsub.Channel()
			for {
				select {
				case <-ctx.Done():
					logger.Info("Shutting down gracefully")
					return nil
				case msg := <-ch:
					if msg == nil {
						continue
					}
					sections := []toolutil.MessageSection{
						{Title: "Channel", Items: []toolutil.KV{{Key: "Name", Value: msg.Channel}}},
					}
					ct := toolutil.GuessMIME([]byte(msg.Payload))
					toolutil.PrintColoredMessage("Redis PubSub", sections, []byte(msg.Payload), ct)
				}
			}
		},
	}

	cmd.Flags().StringVar(&subAddr, "address", "localhost:6379", "Redis address")
	cmd.Flags().StringVar(&subChannel, "channel", "test", "Redis channel (for pub-sub mode)")
	cmd.Flags().StringVar(&subStream, "stream", "", "Redis stream (if set, listens to stream)")
	cmd.Flags().StringVar(&subGroup, "group", "", "Redis consumer group (stream mode)")
	cmd.Flags().StringVar(&subConsumer, "consumer", "", "Redis consumer name (stream mode)")
	cmd.Flags().StringVar(&subDataKey, "dataKey", "data", "Field name holding data in stream messages")

	return cmd
}
