package main

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/sandrolain/eventkit/pkg/common"
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
			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

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

			publish := func() error {
				body, _, err := toolutil.BuildPayload(sendPayload, sendMIME)
				if err != nil {
					toolutil.PrintError("Payload build error: %v", err)
					return err
				}
				if sendStream != "" {
					ack, err := js.Publish(sendSubject, body)
					if err != nil {
						toolutil.PrintError("JetStream publish error: %v", err)
						return err
					}
					toolutil.PrintInfo("Published to JetStream, sequence: %d", ack.Sequence)
				} else {
					if err := nc.Publish(sendSubject, body); err != nil {
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

	return cmd
}
