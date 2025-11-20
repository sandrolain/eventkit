package main

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/sandrolain/eventkit/pkg/common"
	"github.com/sandrolain/eventkit/pkg/testpayload"
	"github.com/segmentio/kafka-go"
	"github.com/spf13/cobra"

	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
)

func sendCommand() *cobra.Command {
	var (
		sendBrokers    string
		sendTopic      string
		sendPayload    string
		sendMIME       string
		sendInterval   string
		headers        []string
		openDelim      string
		closeDelim     string
		seed           int64
		allowFileReads bool
		templateVars   []string
		fileRoot       string
		cacheFiles     bool
		once           bool
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Produce periodic Kafka messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

			w := kafka.NewWriter(kafka.WriterConfig{
				Brokers: strings.Split(sendBrokers, ","),
				Topic:   sendTopic,
			})
			defer func() {
				if err := w.Close(); err != nil {
					slog.Error("Failed to close Kafka writer", "error", err)
				}
			}()

			if seed != 0 {
				testpayload.SeedRandom(seed)
			}
			testpayload.SetAllowFileReads(allowFileReads)
			testpayload.SetFileRoot(fileRoot)
			testpayload.SetFileCacheEnabled(cacheFiles)
			if varsMap, errVars := toolutil.ParseTemplateVars(templateVars); errVars != nil {
				return fmt.Errorf("invalid template-var: %w", errVars)
			} else {
				testpayload.SetTemplateVars(varsMap)
			}
			headerMap, err := toolutil.ParseHeadersWithDelimiters(headers, openDelim, closeDelim)
			if err != nil {
				return fmt.Errorf("invalid headers: %w", err)
			}

			logger := toolutil.Logger()
			logger.Info("Producing to Kafka", "brokers", sendBrokers, "topic", sendTopic, "interval", sendInterval)

			produce := func() error {
				body, _, err := toolutil.BuildPayloadWithDelimiters(sendPayload, sendMIME, openDelim, closeDelim)
				if err != nil {
					logger.Error("Failed to build payload", "error", err)
					return err
				}
				msg := kafka.Message{Value: body}
				for k, v := range headerMap {
					msg.Headers = append(msg.Headers, kafka.Header{Key: k, Value: []byte(v)})
				}

				ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer cancel()
				err = w.WriteMessages(ctx, msg)
				if err != nil {
					logger.Error("Failed to send message", "error", err)
					return err
				}
				logger.Info("Message sent", "bytes", len(body))
				return nil
			}

			return common.RunOnceOrPeriodic(ctx, once, sendInterval, produce)
		},
	}

	cmd.Flags().StringVar(&sendBrokers, "brokers", "localhost:9092", "Kafka brokers (comma-separated)")
	cmd.Flags().StringVar(&sendTopic, "topic", "test", "Kafka topic")
	toolutil.AddPayloadFlags(cmd, &sendPayload, "Hello, Kafka!", &sendMIME, toolutil.CTText)
	toolutil.AddIntervalFlag(cmd, &sendInterval, "5s")
	toolutil.AddOnceFlag(cmd, &once)
	toolutil.AddHeadersFlag(cmd, &headers)
	toolutil.AddTemplateDelimiterFlags(cmd, &openDelim, &closeDelim)
	toolutil.AddSeedFlag(cmd, &seed)
	toolutil.AddAllowFileReadsFlag(cmd, &allowFileReads)
	toolutil.AddTemplateVarFlag(cmd, &templateVars)
	toolutil.AddFileRootFlag(cmd, &fileRoot)
	toolutil.AddFileCacheFlag(cmd, &cacheFiles)

	return cmd
}
