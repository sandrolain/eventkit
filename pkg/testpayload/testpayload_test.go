package testpayload

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/fxamacker/cbor/v2"
)

func TestGenerateRandomJSON(t *testing.T) {
	data, err := GenerateRandomJSON()
	if err != nil {
		t.Fatalf("GenerateRandomJSON() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("GenerateRandomJSON() returned empty data")
	}

	// Verify it's valid JSON
	var payload Payload
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Errorf("GenerateRandomJSON() produced invalid JSON: %v", err)
	}

	// Verify required fields are present
	if payload.ID == "" {
		t.Error("Generated payload missing ID field")
	}
	if payload.Name == "" {
		t.Error("Generated payload missing Name field")
	}
}

func TestGenerateRandomCBOR(t *testing.T) {
	data, err := GenerateRandomCBOR()
	if err != nil {
		t.Fatalf("GenerateRandomCBOR() error = %v", err)
	}

	if len(data) == 0 {
		t.Error("GenerateRandomCBOR() returned empty data")
	}

	// Verify it's valid CBOR
	var payload Payload
	if err := cbor.Unmarshal(data, &payload); err != nil {
		t.Errorf("GenerateRandomCBOR() produced invalid CBOR: %v", err)
	}
}

func TestGenerateSentence(t *testing.T) {
	sentence := GenerateSentence()
	if sentence == "" {
		t.Error("GenerateSentence() returned empty string")
	}
}

func TestGenerateSentimentPhrase(t *testing.T) {
	phrase := GenerateSentimentPhrase()
	if phrase == "" {
		t.Error("GenerateSentimentPhrase() returned empty string")
	}

	// Check it contains expected components
	validStarts := []string{"I love", "I hate", "I think", "I feel", "I wish", "I see"}
	hasValidStart := false
	for _, start := range validStarts {
		if strings.HasPrefix(phrase, start) {
			hasValidStart = true
			break
		}
	}
	if !hasValidStart {
		t.Errorf("GenerateSentimentPhrase() produced phrase with unexpected start: %s", phrase)
	}
}

func TestGenerateRandomDateTime(t *testing.T) {
	dt := GenerateRandomDateTime()
	if dt == "" {
		t.Error("GenerateRandomDateTime() returned empty string")
	}
	// Could add more validation for RFC3339 format
}

func TestGenerateNowDateTime(t *testing.T) {
	dt := GenerateNowDateTime()
	if dt == "" {
		t.Error("GenerateNowDateTime() returned empty string")
	}
}

func TestGenerateCounter(t *testing.T) {
	first := GenerateCounter()
	second := GenerateCounter()
	third := GenerateCounter()

	if second != first+1 {
		t.Errorf("Counter not incrementing correctly: got %d after %d", second, first)
	}
	if third != second+1 {
		t.Errorf("Counter not incrementing correctly: got %d after %d", third, second)
	}
}

func TestInterpolate(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		checkLen bool
		contains string
	}{
		{"Plain text", "hello world", false, ""},
		{"JSON placeholder", "{{json}}", true, ""},
		{"CBOR placeholder", "{{cbor}}", true, ""},
		{"Sentiment placeholder", "{{sentiment}}", false, ""},
		{"Sentence placeholder", "{{sentence}}", false, ""},
		{"DateTime placeholder", "{{datetime}}", false, ""},
		{"NowTime placeholder", "{{nowtime}}", false, ""},
		{"Counter placeholder", "{{counter}}", false, ""},
		{"Mixed text", "Message: {{sentence}}", false, "Message:"},
		{"Multiple placeholders", "ID: {{counter}}, Time: {{nowtime}}", false, "ID:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := Interpolate(tt.input)
			if err != nil {
				t.Errorf("Interpolate() error = %v", err)
				return
			}

			if len(result) == 0 {
				t.Error("Interpolate() returned empty result")
			}

			if tt.checkLen && len(result) < 10 {
				t.Errorf("Interpolate() result too short: %d bytes", len(result))
			}

			if tt.contains != "" && !strings.Contains(string(result), tt.contains) {
				t.Errorf("Interpolate() result should contain '%s', got: %s", tt.contains, string(result))
			}
		})
	}
}

func TestInterpolateWithDelimiters(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		openDelim  string
		closeDelim string
		want       string
		wantErr    bool
	}{
		{
			name:       "Custom delimiters - double brackets",
			input:      "Hello [[sentence]]",
			openDelim:  "[[",
			closeDelim: "]]",
			wantErr:    false,
		},
		{
			name:       "Custom delimiters - percent signs",
			input:      "Count: %counter%",
			openDelim:  "%",
			closeDelim: "%",
			wantErr:    false,
		},
		{
			name:       "Default delimiters",
			input:      "Message: {{sentence}}",
			openDelim:  "{{",
			closeDelim: "}}",
			wantErr:    false,
		},
		{
			name:       "Mixed text with custom delimiters",
			input:      "ID: <<counter>>, Time: <<nowtime>>",
			openDelim:  "<<",
			closeDelim: ">>",
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := InterpolateWithDelimiters(tt.input, tt.openDelim, tt.closeDelim)
			if (err != nil) != tt.wantErr {
				t.Errorf("InterpolateWithDelimiters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(result) == 0 {
				t.Error("InterpolateWithDelimiters() returned empty result")
			}
		})
	}
}

func TestInterpolateWithDelimiters_FilePlaceholder(t *testing.T) {
	// Create a temporary file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test.txt")
	testContent := "Hello from file!"

	// #nosec G306 -- writing file for test payload generation
	if err := os.WriteFile(tmpFile, []byte(testContent), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name       string
		input      string
		openDelim  string
		closeDelim string
		want       string
		wantErr    bool
	}{
		{
			name:       "File placeholder with default delimiters",
			input:      "{{file:" + tmpFile + "}}",
			openDelim:  "{{",
			closeDelim: "}}",
			want:       testContent,
			wantErr:    false,
		},
		{
			name:       "File placeholder with custom delimiters",
			input:      "[[file:" + tmpFile + "]]",
			openDelim:  "[[",
			closeDelim: "]]",
			want:       testContent,
			wantErr:    false,
		},
		{
			name:       "Mixed content with file",
			input:      "Content: {{file:" + tmpFile + "}} - end",
			openDelim:  "{{",
			closeDelim: "}}",
			want:       "Content: " + testContent + " - end",
			wantErr:    false,
		},
		{
			name:       "Non-existent file",
			input:      "{{file:/nonexistent/file.txt}}",
			openDelim:  "{{",
			closeDelim: "}}",
			wantErr:    true,
		},
		{
			name:       "Empty file path",
			input:      "{{file:}}",
			openDelim:  "{{",
			closeDelim: "}}",
			wantErr:    true,
		},
		{
			name:       "Unclosed file placeholder",
			input:      "{{file:" + tmpFile,
			openDelim:  "{{",
			closeDelim: "}}",
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := InterpolateWithDelimiters(tt.input, tt.openDelim, tt.closeDelim)
			if (err != nil) != tt.wantErr {
				t.Errorf("InterpolateWithDelimiters() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if string(result) != tt.want {
					t.Errorf("InterpolateWithDelimiters() = %q, want %q", string(result), tt.want)
				}
			}
		})
	}
}

func TestInterpolateWithDelimiters_MultipleFiles(t *testing.T) {
	// Create temporary files
	tmpDir := t.TempDir()
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "file2.txt")

	// #nosec G306 -- writing file for test payload generation
	if err := os.WriteFile(file1, []byte("Content1"), 0644); err != nil {
		t.Fatalf("Failed to create test file1: %v", err)
	}
	// #nosec G306 -- writing file for test payload generation
	if err := os.WriteFile(file2, []byte("Content2"), 0644); err != nil {
		t.Fatalf("Failed to create test file2: %v", err)
	}

	input := "First: {{file:" + file1 + "}}, Second: {{file:" + file2 + "}}"
	want := "First: Content1, Second: Content2"

	result, err := InterpolateWithDelimiters(input, "{{", "}}")
	if err != nil {
		t.Errorf("InterpolateWithDelimiters() error = %v", err)
		return
	}

	if string(result) != want {
		t.Errorf("InterpolateWithDelimiters() = %q, want %q", string(result), want)
	}
}

func TestTestPayloadType_IsValid(t *testing.T) {
	tests := []struct {
		payloadType TestPayloadType
		want        bool
	}{
		{TestPayloadJSON, true},
		{TestPayloadCBOR, true},
		{TestPayloadSentiment, true},
		{TestPayloadSentence, true},
		{TestPayloadDateTime, true},
		{TestPayloadNowTime, true},
		{"invalid", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(string(tt.payloadType), func(t *testing.T) {
			got := tt.payloadType.IsValid()
			if got != tt.want {
				t.Errorf("IsValid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTestPayloadType_GetContentType(t *testing.T) {
	tests := []struct {
		payloadType TestPayloadType
		want        string
	}{
		{TestPayloadJSON, "application/json"},
		{TestPayloadCBOR, "application/cbor"},
		{TestPayloadSentiment, "text/plain"},
		{TestPayloadSentence, "text/plain"},
		{TestPayloadDateTime, "text/plain"},
		{TestPayloadNowTime, "text/plain"},
		{"invalid", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(string(tt.payloadType), func(t *testing.T) {
			got := tt.payloadType.GetContentType()
			if got != tt.want {
				t.Errorf("GetContentType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTestPayloadType_Generate(t *testing.T) {
	tests := []struct {
		payloadType TestPayloadType
		wantErr     bool
	}{
		{TestPayloadJSON, false},
		{TestPayloadCBOR, false},
		{TestPayloadSentiment, false},
		{TestPayloadSentence, false},
		{TestPayloadDateTime, false},
		{TestPayloadNowTime, false},
		{TestPayloadCounter, false},
		{"invalid", true},
	}

	for _, tt := range tests {
		t.Run(string(tt.payloadType), func(t *testing.T) {
			got, err := tt.payloadType.Generate()
			if (err != nil) != tt.wantErr {
				t.Errorf("Generate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && len(got) == 0 {
				t.Error("Generate() returned empty data")
			}
		})
	}
}
