package main

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/sandrolain/eventkit/pkg/common"
	"github.com/sandrolain/eventkit/pkg/testpayload"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
)

func sendCommand() *cobra.Command {
	var (
		sendAddr       string
		sendSubject    string
		sendPayload    string
		sendMIME       string
		sendInterval   string
		sendStream     string
		headers        []string
		openDelim      string
		closeDelim     string
		seed           int64
		allowFileReads bool
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Publish periodic messages to a NATS subject",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

			nc, err := nats.Connect(sendAddr)
			if err != nil {
				return fmt.Errorf("error connecting to NATS: %w", err)
			}
			defer nc.Close()

			var js nats.JetStreamContext
			if seed != 0 {
				testpayload.SeedRandom(seed)
			}
			testpayload.SetAllowFileReads(allowFileReads)
			headerMap, err := toolutil.ParseHeadersWithDelimiters(headers, openDelim, closeDelim)
			if err != nil {
				return fmt.Errorf("invalid headers: %w", err)
			}

			if sendStream != "" {
				if js, err = nc.JetStream(); err != nil {
					return fmt.Errorf("JetStream context error: %w", err)
				}
				toolutil.PrintSuccess("Connected to NATS with JetStream")
				toolutil.PrintKeyValue("Address", sendAddr)
				toolutil.PrintKeyValue("Subject", sendSubject)
				toolutil.PrintKeyValue("Stream", sendStream)
			} else {
				toolutil.PrintSuccess("Connected to NATS")
				toolutil.PrintKeyValue("Address", sendAddr)
				toolutil.PrintKeyValue("Subject", sendSubject)
			}

			publish := func() error {
				body, _, err := toolutil.BuildPayloadWithDelimiters(sendPayload, sendMIME, openDelim, closeDelim)
				if err != nil {
					toolutil.PrintError("Payload build error: %v", err)
					return err
				}

				// Build NATS message with headers
				msg := nats.NewMsg(sendSubject)
				msg.Data = body
				for k, v := range headerMap {
					msg.Header.Add(k, v)
				}

				if sendStream != "" {
					ack, err := js.PublishMsg(msg)
					if err != nil {
						toolutil.PrintError("JetStream publish error: %v", err)
						return err
					}
					toolutil.PrintInfo("Published to JetStream, sequence: %d", ack.Sequence)
				} else {
					if err := nc.PublishMsg(msg); err != nil {
						toolutil.PrintError("Publish error: %v", err)
						return err
					}
					toolutil.PrintInfo("Published %d bytes", len(body))
				}
				return nil
			}

			return common.StartPeriodicTask(ctx, sendInterval, publish)
		},
	}

	cmd.Flags().StringVar(&sendAddr, "address", nats.DefaultURL, "NATS server URL")
	cmd.Flags().StringVar(&sendSubject, "subject", "test.subject", "NATS subject")
	toolutil.AddPayloadFlags(cmd, &sendPayload, "{nowtime}", &sendMIME, toolutil.CTText)
	toolutil.AddIntervalFlag(cmd, &sendInterval, "5s")
	cmd.Flags().StringVar(&sendStream, "stream", "", "JetStream stream name (if set, uses JetStream)")
	toolutil.AddHeadersFlag(cmd, &headers)
	toolutil.AddTemplateDelimiterFlags(cmd, &openDelim, &closeDelim)
	toolutil.AddSeedFlag(cmd, &seed)
	toolutil.AddAllowFileReadsFlag(cmd, &allowFileReads)

	return cmd
}
