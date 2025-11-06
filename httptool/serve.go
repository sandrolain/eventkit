package main

import (
	"log/slog"

	"github.com/sandrolain/eventkit/pkg/common"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
)

func serveCommand() *cobra.Command {
	var serveAddr string

	cmd := &cobra.Command{
		Use:   "serve",
		Short: "Run an HTTP server that logs requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

			slog.Info("Starting HTTP server", "addr", serveAddr)

			handler := func(ctx *fasthttp.RequestCtx) {
				var queryItems []toolutil.KV
				for key, value := range ctx.QueryArgs().All() {
					queryItems = append(queryItems, toolutil.KV{Key: string(key), Value: string(value)})
				}
				var headerItems []toolutil.KV
				for key, value := range ctx.Request.Header.All() {
					headerItems = append(headerItems, toolutil.KV{Key: string(key), Value: string(value)})
				}
				sections := []toolutil.MessageSection{
					{Title: "Request", Items: []toolutil.KV{{Key: "Method", Value: string(ctx.Method())}, {Key: "URI", Value: string(ctx.RequestURI())}}},
					{Title: "Query", Items: queryItems},
					{Title: "Remote", Items: []toolutil.KV{{Key: "Addr", Value: ctx.RemoteAddr().String()}}},
					{Title: "Headers", Items: headerItems},
				}
				ct := string(ctx.Request.Header.ContentType())
				toolutil.PrintColoredMessage("HTTP", sections, ctx.Request.Body(), ct)
			}

			// Start server in goroutine
			errChan := make(chan error, 1)
			go func() {
				if err := fasthttp.ListenAndServe(serveAddr, handler); err != nil {
					slog.Error("error serving HTTP", "err", err)
					errChan <- err
				}
			}()

			// Wait for shutdown or error
			select {
			case <-ctx.Done():
				slog.Info("Shutting down gracefully")
				return nil
			case err := <-errChan:
				return err
			}
		},
	}

	cmd.Flags().StringVar(&serveAddr, "address", "0.0.0.0:9090", "HTTP listen address")
	return cmd
}
