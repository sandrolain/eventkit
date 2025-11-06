package main

import (
	"fmt"
	"os"
	"time"

	"github.com/nats-io/nats.go"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
)

func sendCommand() *cobra.Command {
	var (
		sendAddr     string
		sendSubject  string
		sendPayload  string
		sendMIME     string
		sendInterval string
		sendStream   string
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Publish periodic messages to a NATS subject",
		RunE: func(cmd *cobra.Command, args []string) error {
			nc, err := nats.Connect(sendAddr)
			if err != nil {
				return fmt.Errorf("error connecting to NATS: %w", err)
			}
			defer nc.Close()

			var js nats.JetStreamContext
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

			dur, err := time.ParseDuration(sendInterval)
			if err != nil {
				return fmt.Errorf("invalid interval: %w", err)
			}
			ticker := time.NewTicker(dur)
			defer ticker.Stop()

			for range ticker.C {
				body, _, err := toolutil.BuildPayload(sendPayload, sendMIME)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					continue
				}
				if sendStream != "" {
					ack, err := js.Publish(sendSubject, body)
					if err != nil {
						fmt.Fprintf(os.Stderr, "JetStream publish error: %v\n", err)
					} else {
						toolutil.PrintInfo("Published to JetStream, sequence: %d", ack.Sequence)
					}
				} else {
					if err := nc.Publish(sendSubject, body); err != nil {
						fmt.Fprintf(os.Stderr, "Publish error: %v\n", err)
					} else {
						toolutil.PrintInfo("Published %d bytes", len(body))
					}
				}
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&sendAddr, "address", nats.DefaultURL, "NATS server URL")
	cmd.Flags().StringVar(&sendSubject, "subject", "test.subject", "NATS subject")
	toolutil.AddPayloadFlags(cmd, &sendPayload, "{nowtime}", &sendMIME, toolutil.CTText)
	toolutil.AddIntervalFlag(cmd, &sendInterval, "5s")
	cmd.Flags().StringVar(&sendStream, "stream", "", "JetStream stream name (if set, uses JetStream)")

	return cmd
}
