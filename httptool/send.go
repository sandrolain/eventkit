package main

import (
	"fmt"
	"os"

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
				reqBody, contentType, err := toolutil.BuildPayloadWithDelimiters(payload, mime, openDelim, closeDelim)
				if err != nil {
					fmt.Fprintln(os.Stderr, err)
					return
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

			return common.StartPeriodicTask(ctx, interval, func() error {
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
	toolutil.AddHeadersFlag(cmd, &headers)
	toolutil.AddTemplateDelimiterFlags(cmd, &openDelim, &closeDelim)
	toolutil.AddSeedFlag(cmd, &seed)
	toolutil.AddAllowFileReadsFlag(cmd, &allowFileReads)
	toolutil.AddTemplateVarFlag(cmd, &templateVars)
	toolutil.AddFileRootFlag(cmd, &fileRoot)
	toolutil.AddFileCacheFlag(cmd, &cacheFiles)

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
