package main

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"time"

	coaptcp "github.com/plgd-dev/go-coap/v3/tcp"
	coapudp "github.com/plgd-dev/go-coap/v3/udp"
	"github.com/sandrolain/eventkit/pkg/common"
	testpayload "github.com/sandrolain/eventkit/pkg/testpayload"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
)

func sendCommand() *cobra.Command {
	var (
		sendAddress  string
		sendPath     string
		sendPayload  string
		sendInterval string
		sendProto    string
		sendMIME     string
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send periodic CoAP POST requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

			logger := toolutil.Logger()
			logger.Info("Sending CoAP POST periodically", "proto", sendProto, "addr", sendAddress, "path", sendPath, "interval", sendInterval)

			sendOnce := func() {
				var body []byte
				var ct string

				b, err := testpayload.Interpolate(sendPayload)
				if err != nil {
					fmt.Fprintf(os.Stderr, "Failed to interpolate payload: %v\n", err)
					return
				}
				body = b
				ct = sendMIME

				if ct == "" {
					ct = toolutil.CTJSON
				}

				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()

				var code any
				var respBody []byte

				mt := MimeToCoapMediaType(ct)

				switch sendProto {
				case "udp":
					client, err := coapudp.Dial(sendAddress)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to dial CoAP (udp): %v\n", err)
						return
					}
					defer client.Close() //nolint:errcheck
					resp, err := client.Post(ctx, sendPath, mt, bytes.NewReader(body))
					if err != nil {
						fmt.Fprintf(os.Stderr, "POST error: %v\n", err)
						return
					}
					code = resp.Code()
					if resp.Body() != nil {
						b, errRead := io.ReadAll(resp.Body())
						if errRead != nil {
							fmt.Fprintf(os.Stderr, "Failed to read response body: %v\n", errRead)
						} else {
							respBody = b
						}
					}
				case "tcp":
					client, err := coaptcp.Dial(sendAddress)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Failed to dial CoAP (tcp): %v\n", err)
						return
					}
					defer client.Close() //nolint:errcheck
					resp, err := client.Post(ctx, sendPath, mt, bytes.NewReader(body))
					if err != nil {
						fmt.Fprintf(os.Stderr, "POST error: %v\n", err)
						return
					}
					code = resp.Code()
					if resp.Body() != nil {
						b, errRead := io.ReadAll(resp.Body())
						if errRead != nil {
							fmt.Fprintf(os.Stderr, "Failed to read response body: %v\n", errRead)
						} else {
							respBody = b
						}
					}
				default:
					fmt.Fprintf(os.Stderr, "Unknown proto: %s (use udp or tcp)\n", sendProto)
					return
				}

				logger.Info("Response received", "code", code, "len", len(respBody))
				if len(respBody) > 0 {
					logger.Info("Response body", "body", string(respBody))
				}
			}

			return common.StartPeriodicTask(ctx, sendInterval, func() error {
				sendOnce()
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&sendAddress, "address", "localhost:5683", "CoAP server address:port")
	toolutil.AddPathFlag(cmd, &sendPath, "/event", "CoAP resource path")
	toolutil.AddPayloadFlags(cmd, &sendPayload, "{}", &sendMIME, toolutil.CTJSON)
	toolutil.AddIntervalFlag(cmd, &sendInterval, "5s")
	cmd.Flags().StringVar(&sendProto, "proto", "udp", "CoAP transport protocol: udp or tcp")

	return cmd
}
