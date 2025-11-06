package main

import (
	"bytes"
	"fmt"
	"os"

	coapmessage "github.com/plgd-dev/go-coap/v3/message"
	coapcodes "github.com/plgd-dev/go-coap/v3/message/codes"
	coapmux "github.com/plgd-dev/go-coap/v3/mux"
	toolutil "github.com/sandrolain/eventkit/pkg/toolutil"
)

// PrintCoAPRequest logs details about an incoming CoAP request.
func PrintCoAPRequest(proto, remote string, req *coapmux.Message) {
	// Build sections and delegate to shared formatter
	path, errPath := req.Options().Path()
	if errPath != nil {
		path = "(error reading path)"
	}
	// Build query
	var query string
	for _, opt := range req.Options() {
		if opt.ID == coapmessage.URIQuery {
			if query != "" {
				query += "&"
			}
			query += string(opt.Value)
		}
	}
	// Build options dump
	var optionItems []toolutil.KV
	for _, opt := range req.Options() {
		optionItems = append(optionItems, toolutil.KV{Key: fmt.Sprintf("%v", opt.ID), Value: fmt.Sprintf("%v", opt.Value)})
	}
	sections := []toolutil.MessageSection{
		{Title: "Request", Items: []toolutil.KV{{Key: "From", Value: fmt.Sprintf("%s (%s)", remote, proto)}, {Key: "Code", Value: fmt.Sprintf("%v", req.Code())}, {Key: "Path", Value: path}, {Key: "Query", Value: query}, {Key: "Token", Value: fmt.Sprintf("%v", req.Token())}}},
		{Title: "Options", Items: optionItems},
	}
	var mime string
	if mt, err := req.Options().ContentFormat(); err == nil {
		mime = CoapMediaTypeToMIME(coapmessage.MediaType(mt))
	}
	var bodyBytes []byte
	if req.Body() != nil {
		buf := new(bytes.Buffer)
		if _, err := buf.ReadFrom(req.Body()); err == nil {
			bodyBytes = buf.Bytes()
		}
	}
	toolutil.PrintColoredMessage("CoAP", sections, bodyBytes, mime)
}

// SimpleOKHandler builds a handler that prints and responds with 2.05 Content and text/plain OK.
func SimpleOKHandler(proto string) coapmux.Handler {
	return coapmux.HandlerFunc(func(w coapmux.ResponseWriter, req *coapmux.Message) {
		PrintCoAPRequest(proto, w.Conn().RemoteAddr().String(), req)
		if err := w.SetResponse(coapcodes.Content, coapmessage.TextPlain, bytes.NewReader([]byte("OK"))); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to set response: %v\n", err)
		}
	})
}

// MimeToCoapMediaType maps common MIME types to CoAP media types.
func MimeToCoapMediaType(ct string) coapmessage.MediaType {
	switch ct {
	case toolutil.CTJSON:
		return coapmessage.AppJSON
	case toolutil.CTCBOR:
		return coapmessage.AppCBOR
	case toolutil.CTText:
		return coapmessage.TextPlain
	default:
		return coapmessage.AppOctets
	}
}

// CoapMediaTypeToMIME maps CoAP media types to MIME strings.
func CoapMediaTypeToMIME(mt coapmessage.MediaType) string {
	switch mt {
	case coapmessage.AppJSON:
		return toolutil.CTJSON
	case coapmessage.AppCBOR:
		return toolutil.CTCBOR
	case coapmessage.TextPlain:
		return toolutil.CTText
	default:
		return "application/octet-stream"
	}
}
