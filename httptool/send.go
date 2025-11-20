package main

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/sandrolain/eventkit/pkg/common"
	"github.com/sandrolain/eventkit/pkg/testpayload"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
	"github.com/spf13/cobra"
	"github.com/valyala/fasthttp"
)

func sendCommand() *cobra.Command {
	var (
		address        string
		method         string
		path           string
		payload        string
		interval       string
		mime           string
		headers        []string
		openDelim      string
		closeDelim     string
		seed           int64
		allowFileReads bool
		templateVars   []string
		fileRoot       string
		cacheFiles     bool
		files          []string
		formFields     []string
		once           bool
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Send periodic HTTP requests",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := common.SetupGracefulShutdown()
			defer cancel()

			url := address + path
			toolutil.PrintSuccess("Starting HTTP client")
			toolutil.PrintKeyValue("Method", method)
			toolutil.PrintKeyValue("URL", url)
			toolutil.PrintKeyValue("Interval", interval)

			if seed != 0 {
				testpayload.SeedRandom(seed)
			}
			testpayload.SetAllowFileReads(allowFileReads)
			testpayload.SetFileRoot(fileRoot)
			// set cache enable
			testpayload.SetFileCacheEnabled(cacheFiles)
			// parse template vars
			varsMap, errVars := toolutil.ParseTemplateVars(templateVars)
			if errVars != nil {
				return fmt.Errorf("invalid template-var: %w", errVars)
			}
			testpayload.SetTemplateVars(varsMap)

			headerMap, err := toolutil.ParseHeadersWithDelimiters(headers, openDelim, closeDelim)
			if err != nil {
				return fmt.Errorf("invalid headers: %w", err)
			}

			sendRequest := func() {
				var reqBody []byte
				var contentType string
				var err error

				// Check if we need to use multipart/form-data
				if len(files) > 0 || len(formFields) > 0 {
					reqBody, contentType, err = buildMultipartRequest(files, formFields, openDelim, closeDelim)
					if err != nil {
						fmt.Fprintf(os.Stderr, "Multipart request error: %v\n", err)
						return
					}
				} else {
					reqBody, contentType, err = toolutil.BuildPayloadWithDelimiters(payload, mime, openDelim, closeDelim)
					if err != nil {
						fmt.Fprintln(os.Stderr, err)
						return
					}
				}

				r := fasthttp.AcquireRequest()
				w := fasthttp.AcquireResponse()
				defer func() {
					fasthttp.ReleaseRequest(r)
					fasthttp.ReleaseResponse(w)
				}()

				r.Header.SetMethod(method)
				r.SetRequestURI(url)
				if contentType != "" {
					r.Header.Set("Content-Type", contentType)
				}
				for k, v := range headerMap {
					r.Header.Set(k, v)
				}
				if len(reqBody) > 0 {
					r.SetBody(reqBody)
				}

				var client fasthttp.Client
				if err := client.Do(r, w); err != nil {
					fmt.Fprintf(os.Stderr, "Request error: %v\n", err)
					return
				}

				printHTTPResponse(method, url, w)
			}

			return common.RunOnceOrPeriodic(ctx, once, interval, func() error {
				sendRequest()
				return nil
			})
		},
	}

	cmd.Flags().StringVar(&address, "address", "http://localhost:8080", "HTTP server base address, e.g. http://localhost:8080")
	toolutil.AddMethodFlag(cmd, &method, "POST", "HTTP method (POST, PUT, PATCH)")
	toolutil.AddPathFlag(cmd, &path, "/event", "HTTP request path")
	toolutil.AddPayloadFlags(cmd, &payload, "{}", &mime, toolutil.CTJSON)
	toolutil.AddIntervalFlag(cmd, &interval, "5s")
	toolutil.AddOnceFlag(cmd, &once)
	toolutil.AddHeadersFlag(cmd, &headers)
	toolutil.AddTemplateDelimiterFlags(cmd, &openDelim, &closeDelim)
	toolutil.AddSeedFlag(cmd, &seed)
	toolutil.AddAllowFileReadsFlag(cmd, &allowFileReads)
	toolutil.AddTemplateVarFlag(cmd, &templateVars)
	toolutil.AddFileRootFlag(cmd, &fileRoot)
	toolutil.AddFileCacheFlag(cmd, &cacheFiles)
	cmd.Flags().StringArrayVarP(&files, "file", "f", []string{}, "File to upload in multipart/form-data format. Use name=path syntax (can be repeated)")
	cmd.Flags().StringArrayVar(&formFields, "form-field", []string{}, "Form field in name=value format for multipart/form-data (can be repeated)")

	return cmd
}

func printHTTPResponse(method, url string, resp *fasthttp.Response) {
	var headerItems []toolutil.KV
	for key, value := range resp.Header.All() {
		headerItems = append(headerItems, toolutil.KV{Key: string(key), Value: string(value)})
	}

	statusText := fasthttp.StatusMessage(resp.StatusCode())
	sections := []toolutil.MessageSection{
		{Title: "Request", Items: []toolutil.KV{{Key: "Method", Value: method}, {Key: "URL", Value: url}}},
		{Title: "Response", Items: []toolutil.KV{{Key: "Status", Value: fmt.Sprintf("%d %s", resp.StatusCode(), statusText)}}},
		{Title: "Headers", Items: headerItems},
	}

	mimeType := string(resp.Header.ContentType())
	if mimeType == "" {
		mimeType = toolutil.GuessMIME(resp.Body())
	}

	toolutil.PrintColoredMessage("HTTP Response", sections, resp.Body(), mimeType)
}

// buildMultipartRequest creates a multipart/form-data request body with files and form fields.
// Files should be in the format "fieldname=filepath".
// Form fields should be in the format "fieldname=value".
// Values support template interpolation using the specified delimiters.
func buildMultipartRequest(files []string, formFields []string, openDelim string, closeDelim string) ([]byte, string, error) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Add form fields
	for _, field := range formFields {
		parts := splitOnce(field, "=")
		if len(parts) != 2 {
			return nil, "", fmt.Errorf("invalid form field format '%s', expected name=value", field)
		}
		fieldName := parts[0]
		fieldValue := parts[1]

		// Interpolate template variables in field value
		interpolatedValue, err := testpayload.InterpolateWithDelimiters(fieldValue, openDelim, closeDelim)
		if err != nil {
			return nil, "", fmt.Errorf("failed to interpolate form field '%s': %w", fieldName, err)
		}

		if err := writer.WriteField(fieldName, string(interpolatedValue)); err != nil {
			return nil, "", fmt.Errorf("failed to write form field '%s': %w", fieldName, err)
		}
	}

	// Add files
	for _, file := range files {
		parts := splitOnce(file, "=")
		if len(parts) != 2 {
			return nil, "", fmt.Errorf("invalid file format '%s', expected name=path", file)
		}
		fieldName := parts[0]
		filePath := parts[1]

		// Open the file
		// #nosec G304 - File path is intentionally provided by user via CLI flag
		f, err := os.Open(filePath)
		if err != nil {
			return nil, "", fmt.Errorf("failed to open file '%s': %w", filePath, err)
		}
		defer func() {
			if closeErr := f.Close(); closeErr != nil {
				fmt.Fprintf(os.Stderr, "Warning: failed to close file '%s': %v\n", filePath, closeErr)
			}
		}()

		// Create form file part
		fileName := filepath.Base(filePath)
		part, err := writer.CreateFormFile(fieldName, fileName)
		if err != nil {
			return nil, "", fmt.Errorf("failed to create form file for '%s': %w", fieldName, err)
		}

		// Copy file content to part
		if _, err := io.Copy(part, f); err != nil {
			return nil, "", fmt.Errorf("failed to copy file content for '%s': %w", fieldName, err)
		}
	}

	// Close the multipart writer
	if err := writer.Close(); err != nil {
		return nil, "", fmt.Errorf("failed to close multipart writer: %w", err)
	}

	return buf.Bytes(), writer.FormDataContentType(), nil
}

// splitOnce splits a string on the first occurrence of separator.
// Returns a slice with at most 2 elements.
func splitOnce(s, sep string) []string {
	idx := bytes.IndexByte([]byte(s), sep[0])
	if idx == -1 {
		return []string{s}
	}
	return []string{s[:idx], s[idx+1:]}
}
