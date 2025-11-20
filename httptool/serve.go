package main

import (
	"bytes"
	"fmt"
	"log/slog"
	"mime"
	"mime/multipart"
	"strings"

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
				body := ctx.Request.Body()

				// Check if this is a multipart request
				if isMultipartRequest(ct) {
					multipartSections, multipartBody := parseMultipartRequest(ct, body)
					if multipartSections != nil {
						sections = append(sections, multipartSections...)
						toolutil.PrintColoredMessage("HTTP", sections, []byte(multipartBody), "text/plain")
						return
					}
				}

				// Standard request handling
				toolutil.PrintColoredMessage("HTTP", sections, body, ct)
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

// isMultipartRequest checks if the Content-Type indicates a multipart request.
func isMultipartRequest(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return strings.HasPrefix(mediaType, "multipart/")
}

// parseMultipartRequest parses a multipart request and returns sections with file info and form fields.
// Returns nil if parsing fails.
func parseMultipartRequest(contentType string, body []byte) ([]toolutil.MessageSection, string) {
	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		return nil, ""
	}

	boundary, ok := params["boundary"]
	if !ok {
		return nil, ""
	}

	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	var formFields []toolutil.KV
	var files []toolutil.KV
	var bodyParts []string

	for {
		part, err := reader.NextPart()
		if err != nil {
			break
		}

		formName := part.FormName()
		fileName := part.FileName()

		// Read part content
		buf := new(bytes.Buffer)
		size, err := buf.ReadFrom(part)
		if err != nil {
			// Log error but continue processing other parts
			continue
		}

		if fileName != "" {
			// This is a file upload
			files = append(files, toolutil.KV{
				Key:   formName,
				Value: fmt.Sprintf("%s (%d bytes)", fileName, size),
			})
			bodyParts = append(bodyParts, fmt.Sprintf("[File: %s = %s (%d bytes)]", formName, fileName, size))
		} else {
			// This is a form field
			value := buf.String()
			formFields = append(formFields, toolutil.KV{
				Key:   formName,
				Value: value,
			})
			bodyParts = append(bodyParts, fmt.Sprintf("%s = %s", formName, value))
		}
	}

	sections := []toolutil.MessageSection{}
	if len(formFields) > 0 {
		sections = append(sections, toolutil.MessageSection{
			Title: "Form Fields",
			Items: formFields,
		})
	}
	if len(files) > 0 {
		sections = append(sections, toolutil.MessageSection{
			Title: "Files",
			Items: files,
		})
	}

	return sections, strings.Join(bodyParts, "\n")
}
