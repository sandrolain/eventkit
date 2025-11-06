package main

import (
	"fmt"
	"log/slog"
	"os"

	coap "github.com/plgd-dev/go-coap/v3"
	coapmux "github.com/plgd-dev/go-coap/v3/mux"
	"github.com/sandrolain/eventkit/pkg/common"
	"github.com/spf13/cobra"
)

func serveCommand() *cobra.Command {
	var (
		serveAddr  string
		serveProto string
	)

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run a CoAP server that logs requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

			logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
			logger.Info("Starting CoAP server", "proto", serveProto, "addr", serveAddr)

			router := coapmux.NewRouter()
			if err := router.Handle("/", SimpleOKHandler(serveProto)); err != nil {
				return err
			}

			// Start server in goroutine
			errChan := make(chan error, 1)
			go func() {
				errChan <- Serve(serveProto, serveAddr, router)
			}()

			// Wait for shutdown or error
			select {
			case <-ctx.Done():
				logger.Info("Shutting down gracefully")
				return nil
			case err := <-errChan:
				return err
			}
		},
	}

	cmd.Flags().StringVar(&serveAddr, "address", ":5683", "Listen address (e.g.: :5683)")
	cmd.Flags().StringVar(&serveProto, "proto", "udp", "CoAP transport protocol: udp or tcp")

	return cmd
}

// Serve runs a mux router on chosen proto (udp or tcp).
func Serve(proto, addr string, router *coapmux.Router) error {
	switch proto {
	case "udp", "tcp":
		return coap.ListenAndServe(proto, addr, router)
	default:
		return fmt.Errorf("unknown mode: %s (use udp or tcp)", proto)
	}
}
