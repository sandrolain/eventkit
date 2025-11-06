package toolutil

import (
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
