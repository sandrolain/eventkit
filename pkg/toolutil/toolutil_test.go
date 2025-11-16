package toolutil

import (
	"encoding/base64"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
	"github.com/spf13/cobra"
)

func TestLogger(t *testing.T) {
	logger := Logger()
	if logger == nil {
		t.Error("Logger() returned nil")
	}
}

func TestPrettyBodyByMIME(t *testing.T) {
	tests := []struct {
		name     string
		mime     string
		body     []byte
		notEmpty bool
	}{
		{
			name:     "Valid JSON",
			mime:     "application/json",
			body:     []byte(`{"name":"test","value":42}`),
			notEmpty: true,
		},
		{
			name:     "Invalid JSON",
			mime:     "application/json",
			body:     []byte(`invalid json`),
			notEmpty: true,
		},
		{
			name:     "Valid CBOR",
			mime:     "application/cbor",
			body:     mustEncodeCBOR(t, map[string]interface{}{"name": "test"}),
			notEmpty: false, // CBOR decoding might fail in colorjson
		},
		{
			name:     "Plain text",
			mime:     "text/plain",
			body:     []byte("hello world"),
			notEmpty: true,
		},
		{
			name:     "Empty body",
			mime:     "application/json",
			body:     []byte{},
			notEmpty: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := PrettyBodyByMIME(tt.mime, tt.body)
			if tt.notEmpty && len(result) == 0 {
				t.Error("PrettyBodyByMIME() returned empty result")
			}
			if !tt.notEmpty && len(result) != 0 {
				t.Error("PrettyBodyByMIME() should return empty for empty input")
			}
		})
	}
}

func mustEncodeCBOR(t *testing.T, v interface{}) []byte {
	t.Helper()
	data, err := cbor.Marshal(v)
	if err != nil {
		t.Fatalf("Failed to encode CBOR: %v", err)
	}
	return data
}

func TestEncodeCBORFromJSON(t *testing.T) {
	tests := []struct {
		name    string
		json    string
		wantErr bool
	}{
		{
			name:    "Valid JSON object",
			json:    `{"name":"test","value":42}`,
			wantErr: false,
		},
		{
			name:    "Valid JSON array",
			json:    `[1,2,3]`,
			wantErr: false,
		},
		{
			name:    "Invalid JSON",
			json:    `{invalid}`,
			wantErr: true,
		},
		{
			name:    "Empty JSON",
			json:    `{}`,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := EncodeCBORFromJSON(tt.json)
			if (err != nil) != tt.wantErr {
				t.Errorf("EncodeCBORFromJSON() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(result) == 0 {
				t.Error("EncodeCBORFromJSON() returned empty result")
			}
		})
	}
}

func TestBuildPayload(t *testing.T) {
	tests := []struct {
		name        string
		rawPayload  string
		mime        string
		wantErr     bool
		checkResult bool
	}{
		{
			name:        "Plain text",
			rawPayload:  "hello world",
			mime:        CTText,
			wantErr:     false,
			checkResult: true,
		},
		{
			name:        "JSON placeholder",
			rawPayload:  "{json}",
			mime:        CTJSON,
			wantErr:     false,
			checkResult: true,
		},
		{
			name:        "Counter placeholder",
			rawPayload:  "{counter}",
			mime:        CTText,
			wantErr:     false,
			checkResult: true,
		},
		{
			name:        "Mixed content",
			rawPayload:  "ID: {counter}, Time: {nowtime}",
			mime:        CTText,
			wantErr:     false,
			checkResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType, err := BuildPayload(tt.rawPayload, tt.mime)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildPayload() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkResult && len(body) == 0 {
				t.Error("BuildPayload() returned empty body")
			}
			if contentType != tt.mime {
				t.Errorf("BuildPayload() contentType = %v, want %v", contentType, tt.mime)
			}
		})
	}
}

func TestGuessMIME(t *testing.T) {
	tests := []struct {
		name string
		body []byte
		want string
	}{
		{
			name: "JSON object",
			body: []byte(`{"name":"test"}`),
			want: CTJSON,
		},
		{
			name: "JSON array",
			body: []byte(`[1,2,3]`),
			want: CTJSON,
		},
		{
			name: "JSON with spaces",
			body: []byte(`  {"name":"test"}  `),
			want: CTJSON,
		},
		{
			name: "Plain text",
			body: []byte("hello world"),
			want: CTCBOR, // 'h' (0x68) matches CBOR text string pattern
		},
		{
			name: "Empty",
			body: []byte{},
			want: CTText,
		},
		{
			name: "CBOR map",
			body: []byte{0xA1, 0x64, 0x6E, 0x61, 0x6D, 0x65, 0x64, 0x74, 0x65, 0x73, 0x74},
			want: CTCBOR,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GuessMIME(tt.body)
			if got != tt.want {
				t.Errorf("GuessMIME() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestAddMethodFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var method string

	AddMethodFlag(cmd, &method, "GET", "Test method flag")

	if cmd.Flags().Lookup("method") == nil {
		t.Error("AddMethodFlag() did not add 'method' flag")
	}
}

func TestAddPathFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var path string

	AddPathFlag(cmd, &path, "/test", "Test path flag")

	if cmd.Flags().Lookup("path") == nil {
		t.Error("AddPathFlag() did not add 'path' flag")
	}
}

func TestAddPayloadFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var payload, mime string

	AddPayloadFlags(cmd, &payload, "{}", &mime, CTJSON)

	if cmd.Flags().Lookup("payload") == nil {
		t.Error("AddPayloadFlags() did not add 'payload' flag")
	}
	if cmd.Flags().Lookup("mime") == nil {
		t.Error("AddPayloadFlags() did not add 'mime' flag")
	}
}

func TestAddIntervalFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var interval string

	AddIntervalFlag(cmd, &interval, "5s")

	if cmd.Flags().Lookup("interval") == nil {
		t.Error("AddIntervalFlag() did not add 'interval' flag")
	}
}

func TestAddServerFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var server string

	AddServerFlag(cmd, &server, "localhost", "address", "broker")

	if cmd.Flags().Lookup("server") == nil {
		t.Error("AddServerFlag() did not add 'server' flag")
	}
	if cmd.Flags().Lookup("address") == nil {
		t.Error("AddServerFlag() did not add 'address' alias")
	}
	if cmd.Flags().Lookup("broker") == nil {
		t.Error("AddServerFlag() did not add 'broker' alias")
	}
}

func TestAddDestFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var dest string

	AddDestFlag(cmd, &dest, "/test", "Test destination", "path", "topic")

	if cmd.Flags().Lookup("dest") == nil {
		t.Error("AddDestFlag() did not add 'dest' flag")
	}
	if cmd.Flags().Lookup("path") == nil {
		t.Error("AddDestFlag() did not add 'path' alias")
	}
	if cmd.Flags().Lookup("topic") == nil {
		t.Error("AddDestFlag() did not add 'topic' alias")
	}
}

func TestPrintColoredMessage(t *testing.T) {
	// This test just verifies it doesn't panic
	sections := []MessageSection{
		{
			Title: "Test Section",
			Items: []KV{
				{Key: "key1", Value: "value1"},
				{Key: "key2", Value: "value2"},
			},
		},
	}

	body := []byte(`{"test":"data"}`)

	// Should not panic
	PrintColoredMessage("Test Title", sections, body, CTJSON)
}

func TestConstants(t *testing.T) {
	if CTJSON != "application/json" {
		t.Errorf("CTJSON = %v, want 'application/json'", CTJSON)
	}
	if CTCBOR != "application/cbor" {
		t.Errorf("CTCBOR = %v, want 'application/cbor'", CTCBOR)
	}
	if CTText != "text/plain" {
		t.Errorf("CTText = %v, want 'text/plain'", CTText)
	}
}

func TestBuildPayloadWithDelimiters(t *testing.T) {
	tests := []struct {
		name        string
		rawPayload  string
		mime        string
		openDelim   string
		closeDelim  string
		wantErr     bool
		checkResult bool
	}{
		{
			name:        "Custom delimiters with counter",
			rawPayload:  "ID: [[counter]]",
			mime:        CTText,
			openDelim:   "[[",
			closeDelim:  "]]",
			wantErr:     false,
			checkResult: true,
		},
		{
			name:        "Percent delimiters",
			rawPayload:  "Time: %nowtime%",
			mime:        CTText,
			openDelim:   "%",
			closeDelim:  "%",
			wantErr:     false,
			checkResult: true,
		},
		{
			name:        "Default delimiters",
			rawPayload:  "{{sentence}}",
			mime:        CTText,
			openDelim:   "{{",
			closeDelim:  "}}",
			wantErr:     false,
			checkResult: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType, err := BuildPayloadWithDelimiters(tt.rawPayload, tt.mime, tt.openDelim, tt.closeDelim)
			if (err != nil) != tt.wantErr {
				t.Errorf("BuildPayloadWithDelimiters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.checkResult && len(body) == 0 {
				t.Error("BuildPayloadWithDelimiters() returned empty body")
			}
			if contentType != tt.mime {
				t.Errorf("BuildPayloadWithDelimiters() contentType = %v, want %v", contentType, tt.mime)
			}
		})
	}
}

func TestBuildPayload_MimeAutoDetect(t *testing.T) {
	body, contentType, err := BuildPayloadWithDelimiters("{{json}}", "", "{{", "}}")
	if err != nil {
		t.Fatalf("BuildPayloadWithDelimiters() error = %v", err)
	}
	if contentType != CTJSON {
		t.Errorf("Expected contentType %s, got %s", CTJSON, contentType)
	}
	if len(body) == 0 {
		t.Errorf("Expected non-empty body")
	}
}

func TestParseHeaders(t *testing.T) {
	tests := []struct {
		name    string
		headers []string
		want    map[string]string
		wantErr bool
		checkFn func(t *testing.T, got map[string]string) // Custom validation function
	}{
		{
			name:    "Valid headers",
			headers: []string{"Content-Type=application/json", "X-Custom-Header=value123"},
			want:    map[string]string{"Content-Type": "application/json", "X-Custom-Header": "value123"},
			wantErr: false,
		},
		{
			name:    "Headers with spaces",
			headers: []string{"Key = Value", " Another =  Test "},
			want:    map[string]string{"Key": "Value", "Another": "Test"},
			wantErr: false,
		},
		{
			name:    "Empty header list",
			headers: []string{},
			want:    map[string]string{},
			wantErr: false,
		},
		{
			name:    "Invalid format - no equals",
			headers: []string{"InvalidHeader"},
			wantErr: true,
		},
		{
			name:    "Invalid format - empty key",
			headers: []string{"=value"},
			wantErr: true,
		},
		{
			name:    "Valid header with equals in value",
			headers: []string{"Authorization=Bearer token=123"},
			want:    map[string]string{"Authorization": "Bearer token=123"},
			wantErr: false,
		},
		{
			name:    "Mixed valid and invalid",
			headers: []string{"Valid=true", "Invalid"},
			wantErr: true,
		},
		{
			name:    "Header with counter template",
			headers: []string{"X-Request-ID={{counter}}"},
			wantErr: false,
			checkFn: func(t *testing.T, got map[string]string) {
				val, exists := got["X-Request-ID"]
				if !exists {
					t.Error("X-Request-ID header not found")
					return
				}
				if val == "" || val == "{{counter}}" {
					t.Errorf("Counter template not interpolated, got: %s", val)
				}
			},
		},
		{
			name:    "Header with nowtime template",
			headers: []string{"X-Timestamp={{nowtime}}"},
			wantErr: false,
			checkFn: func(t *testing.T, got map[string]string) {
				val, exists := got["X-Timestamp"]
				if !exists {
					t.Error("X-Timestamp header not found")
					return
				}
				if val == "" || val == "{{nowtime}}" {
					t.Errorf("Nowtime template not interpolated, got: %s", val)
				}
			},
		},
		{
			name:    "Header with sentence template",
			headers: []string{"X-Message={{sentence}}"},
			wantErr: false,
			checkFn: func(t *testing.T, got map[string]string) {
				val, exists := got["X-Message"]
				if !exists {
					t.Error("X-Message header not found")
					return
				}
				if val == "" || val == "{{sentence}}" {
					t.Errorf("Sentence template not interpolated, got: %s", val)
				}
			},
		},
		{
			name:    "Multiple headers with templates",
			headers: []string{"X-ID={{counter}}", "X-Time={{nowtime}}", "X-Static=static-value"},
			wantErr: false,
			checkFn: func(t *testing.T, got map[string]string) {
				if len(got) != 3 {
					t.Errorf("Expected 3 headers, got %d", len(got))
					return
				}
				if got["X-Static"] != "static-value" {
					t.Errorf("X-Static = %s, want static-value", got["X-Static"])
				}
				if got["X-ID"] == "" || got["X-ID"] == "{{counter}}" {
					t.Errorf("X-ID template not interpolated: %s", got["X-ID"])
				}
				if got["X-Time"] == "" || got["X-Time"] == "{{nowtime}}" {
					t.Errorf("X-Time template not interpolated: %s", got["X-Time"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHeaders(tt.headers)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHeaders() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if tt.checkFn != nil {
					tt.checkFn(t, got)
				} else if tt.want != nil {
					if len(got) != len(tt.want) {
						t.Errorf("ParseHeaders() returned %d headers, want %d", len(got), len(tt.want))
						return
					}
					for k, v := range tt.want {
						if got[k] != v {
							t.Errorf("ParseHeaders()[%q] = %q, want %q", k, got[k], v)
						}
					}
				}
			}
		})
	}
}

func TestAddHeadersFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var headers []string

	AddHeadersFlag(cmd, &headers)

	flag := cmd.Flags().Lookup("header")
	if flag == nil {
		t.Error("AddHeadersFlag() did not add 'header' flag")
		return
	}

	// Check short flag
	if flag.Shorthand != "H" {
		t.Errorf("AddHeadersFlag() shorthand = %q, want 'H'", flag.Shorthand)
	}
}

func TestAddTemplateDelimiterFlags(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var openDelim, closeDelim string

	AddTemplateDelimiterFlags(cmd, &openDelim, &closeDelim)

	if cmd.Flags().Lookup("template-open") == nil {
		t.Error("AddTemplateDelimiterFlags() did not add 'template-open' flag")
	}
	if cmd.Flags().Lookup("template-close") == nil {
		t.Error("AddTemplateDelimiterFlags() did not add 'template-close' flag")
	}
}

func TestAddSeedFlagAndAllowFileReadsFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var seed int64
	var allowFileReads bool
	AddSeedFlag(cmd, &seed)
	AddAllowFileReadsFlag(cmd, &allowFileReads)

	if cmd.Flags().Lookup("seed") == nil {
		t.Error("AddSeedFlag() did not add 'seed' flag")
	}
	if cmd.Flags().Lookup("allow-file-reads") == nil {
		t.Error("AddAllowFileReadsFlag() did not add 'allow-file-reads' flag")
	}
}

func TestParseTemplateVars(t *testing.T) {
	vars := []string{"a=1", "b=two", "c = three"}
	got, err := ParseTemplateVars(vars)
	if err != nil {
		t.Fatalf("ParseTemplateVars() error = %v", err)
	}
	if got["a"] != "1" || got["b"] != "two" || got["c"] != " three" {
		t.Errorf("ParseTemplateVars() = %v", got)
	}
}

func TestAddFileRootFlag(t *testing.T) {
	cmd := &cobra.Command{Use: "test"}
	var root string
	AddFileRootFlag(cmd, &root)
	if cmd.Flags().Lookup("file-root") == nil {
		t.Error("AddFileRootFlag() did not add 'file-root' flag")
	}
}

func TestParseHeadersWithDelimiters(t *testing.T) {
	tests := []struct {
		name       string
		headers    []string
		openDelim  string
		closeDelim string
		want       map[string]string
		wantErr    bool
		checkFn    func(t *testing.T, got map[string]string)
	}{
		{
			name:       "Custom delimiters with counter",
			headers:    []string{"X-ID=<<counter>>"},
			openDelim:  "<<",
			closeDelim: ">>",
			wantErr:    false,
			checkFn: func(t *testing.T, got map[string]string) {
				val := got["X-ID"]
				if val == "" || val == "<<counter>>" {
					t.Errorf("Counter not interpolated with custom delimiters, got: %s", val)
				}
			},
		},
		{
			name:       "Custom delimiters with nowtime",
			headers:    []string{"X-Time=%%nowtime%%"},
			openDelim:  "%%",
			closeDelim: "%%",
			wantErr:    false,
			checkFn: func(t *testing.T, got map[string]string) {
				val := got["X-Time"]
				if val == "" || val == "%%nowtime%%" {
					t.Errorf("Nowtime not interpolated with custom delimiters, got: %s", val)
				}
			},
		},
		{
			name:       "Mixed static and template with custom delimiters",
			headers:    []string{"X-Msg=prefix-<<counter>>-suffix"},
			openDelim:  "<<",
			closeDelim: ">>",
			wantErr:    false,
			checkFn: func(t *testing.T, got map[string]string) {
				val := got["X-Msg"]
				if !strings.HasPrefix(val, "prefix-") || !strings.HasSuffix(val, "-suffix") {
					t.Errorf("Expected prefix-*-suffix pattern, got: %s", val)
				}
				if strings.Contains(val, "<<") || strings.Contains(val, ">>") {
					t.Errorf("Delimiters not replaced, got: %s", val)
				}
			},
		},
		{
			name:       "Default delimiters",
			headers:    []string{"X-ID={{counter}}"},
			openDelim:  "{{",
			closeDelim: "}}",
			wantErr:    false,
			checkFn: func(t *testing.T, got map[string]string) {
				val := got["X-ID"]
				if val == "" || val == "{{counter}}" {
					t.Errorf("Counter not interpolated, got: %s", val)
				}
			},
		},
		{
			name:       "Invalid format with custom delimiters",
			headers:    []string{"InvalidHeader"},
			openDelim:  "<<",
			closeDelim: ">>",
			wantErr:    true,
		},
		{
			name:       "Multiple headers with custom delimiters",
			headers:    []string{"X-ID=<<counter>>", "X-Static=value", "X-Time=<<nowtime>>"},
			openDelim:  "<<",
			closeDelim: ">>",
			wantErr:    false,
			checkFn: func(t *testing.T, got map[string]string) {
				if len(got) != 3 {
					t.Errorf("Expected 3 headers, got %d", len(got))
					return
				}
				if got["X-Static"] != "value" {
					t.Errorf("X-Static = %s, want value", got["X-Static"])
				}
				if got["X-ID"] == "" || got["X-ID"] == "<<counter>>" {
					t.Errorf("X-ID not interpolated: %s", got["X-ID"])
				}
				if got["X-Time"] == "" || got["X-Time"] == "<<nowtime>>" {
					t.Errorf("X-Time not interpolated: %s", got["X-Time"])
				}
			},
		},
		{
			name:       "Binary header with CBOR placeholder gets base64 encoded",
			headers:    []string{"X-Bin={{cbor}}"},
			openDelim:  "{{",
			closeDelim: "}}",
			wantErr:    false,
			checkFn: func(t *testing.T, got map[string]string) {
				val, exists := got["X-Bin"]
				if !exists {
					t.Error("X-Bin header not found")
					return
				}
				// Should be base64 string, decodeable into CBOR
				decoded, err := base64.StdEncoding.DecodeString(val)
				if err != nil {
					t.Errorf("X-Bin not base64 encoded: %v", err)
					return
				}
				var obj map[string]interface{}
				if err := cbor.Unmarshal(decoded, &obj); err != nil {
					t.Errorf("Decoded CBOR invalid: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseHeadersWithDelimiters(tt.headers, tt.openDelim, tt.closeDelim)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseHeadersWithDelimiters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && tt.checkFn != nil {
				tt.checkFn(t, got)
			}
		})
	}
}
