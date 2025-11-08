package main

import (
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/sandrolain/eventkit/pkg/common"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
)

func serveCommand() *cobra.Command {
	var (
		subAddr    string
		subSubject string
		subStream  string
		subDurable string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Subscribe to a subject and log messages",
		RunE: func(cmd *cobra.Command, args []string) error {
			nc, err := nats.Connect(subAddr)
			if err != nil {
				return fmt.Errorf("error connecting to NATS: %w", err)
			}
			defer nc.Close()

			// Shared handler
			handler := func(msg *nats.Msg) {
				sections := []toolutil.MessageSection{{Title: "Subject", Items: []toolutil.KV{{Key: "Name", Value: msg.Subject}}}}
				if msg.Reply != "" {
					sections = append(sections, toolutil.MessageSection{Title: "Reply", Items: []toolutil.KV{{Key: "To", Value: msg.Reply}}})
				}
				if len(msg.Header) > 0 {
					var headerItems []toolutil.KV
					for k, v := range msg.Header {
						headerItems = append(headerItems, toolutil.KV{Key: k, Value: fmt.Sprintf("%v", v)})
					}
					sections = append(sections, toolutil.MessageSection{Title: "Headers", Items: headerItems})
				}
				ct := toolutil.GuessMIME(msg.Data)
				toolutil.PrintColoredMessage("NATS", sections, msg.Data, ct)
				if msg.Reply != "" {
					if err := nc.Publish(msg.Reply, []byte("OK")); err != nil {
						toolutil.PrintError("Failed to send reply: %v", err)
					}
				}
			}

			var sub *nats.Subscription
			if subStream != "" {
				js, err := nc.JetStream()
				if err != nil {
					return fmt.Errorf("JetStream context error: %w", err)
				}
				fmt.Printf("Listening (JetStream) on %s, subject '%s', stream '%s'\n", subAddr, subSubject, subStream)
				opts := []nats.SubOpt{nats.BindStream(subStream), nats.DeliverNew()}
				if subDurable != "" {
					opts = append(opts, nats.Durable(subDurable))
				}
				sub, err = js.Subscribe(subSubject, handler, opts...)
				if err != nil {
					return fmt.Errorf("error subscribing (JetStream): %w", err)
				}
			} else {
				fmt.Printf("Listening on %s, subject '%s'\n", subAddr, subSubject)
				sub, err = nc.Subscribe(subSubject, handler)
				if err != nil {
					return fmt.Errorf("error subscribing to subject: %w", err)
				}
			}

			if subStream != "" {
				toolutil.PrintSuccess("Subscribed to NATS with JetStream")
				toolutil.PrintKeyValue("Address", subAddr)
				toolutil.PrintKeyValue("Subject", subSubject)
				toolutil.PrintKeyValue("Stream", subStream)
			} else {
				toolutil.PrintSuccess("Subscribed to NATS")
				toolutil.PrintKeyValue("Address", subAddr)
				toolutil.PrintKeyValue("Subject", subSubject)
			}

			common.WaitForShutdown()

			if err := sub.Drain(); err != nil {
				toolutil.PrintError("Failed to drain subscription: %v", err)
			}
			return nil
		},
	}

	cmd.Flags().StringVar(&subAddr, "address", nats.DefaultURL, "NATS server URL")
	cmd.Flags().StringVar(&subSubject, "subject", "test", "NATS subject to listen on")
	cmd.Flags().StringVar(&subStream, "stream", "", "JetStream stream name (if set, uses JetStream consumer)")
	cmd.Flags().StringVar(&subDurable, "durable", "", "JetStream durable consumer name (optional)")

	return cmd
}
