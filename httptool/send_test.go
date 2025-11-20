package main

import (
	"bytes"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildMultipartRequest(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("Hello, World!")
	if err := os.WriteFile(testFile, testContent, 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name       string
		files      []string
		formFields []string
		openDelim  string
		closeDelim string
		wantErr    bool
		validate   func(t *testing.T, body []byte, contentType string)
	}{
		{
			name:       "single file upload",
			files:      []string{"document=" + testFile},
			formFields: []string{},
			openDelim:  "{{",
			closeDelim: "}}",
			wantErr:    false,
			validate: func(t *testing.T, body []byte, contentType string) {
				if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
					t.Errorf("Expected multipart/form-data content type, got %s", contentType)
				}
				if !bytes.Contains(body, testContent) {
					t.Error("Expected file content in multipart body")
				}
				if !bytes.Contains(body, []byte(`name="document"`)) {
					t.Error("Expected field name 'document' in multipart body")
				}
			},
		},
		{
			name:       "multiple files upload",
			files:      []string{"file1=" + testFile, "file2=" + testFile},
			formFields: []string{},
			openDelim:  "{{",
			closeDelim: "}}",
			wantErr:    false,
			validate: func(t *testing.T, body []byte, contentType string) {
				if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
					t.Errorf("Expected multipart/form-data content type, got %s", contentType)
				}
				if !bytes.Contains(body, []byte(`name="file1"`)) {
					t.Error("Expected field name 'file1' in multipart body")
				}
				if !bytes.Contains(body, []byte(`name="file2"`)) {
					t.Error("Expected field name 'file2' in multipart body")
				}
			},
		},
		{
			name:       "file and form fields",
			files:      []string{"document=" + testFile},
			formFields: []string{"username=testuser", "email=test@example.com"},
			openDelim:  "{{",
			closeDelim: "}}",
			wantErr:    false,
			validate: func(t *testing.T, body []byte, contentType string) {
				if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
					t.Errorf("Expected multipart/form-data content type, got %s", contentType)
				}
				if !bytes.Contains(body, []byte("testuser")) {
					t.Error("Expected form field 'username' value in multipart body")
				}
				if !bytes.Contains(body, []byte("test@example.com")) {
					t.Error("Expected form field 'email' value in multipart body")
				}
				if !bytes.Contains(body, testContent) {
					t.Error("Expected file content in multipart body")
				}
			},
		},
		{
			name:       "form fields only",
			files:      []string{},
			formFields: []string{"key1=value1", "key2=value2"},
			openDelim:  "{{",
			closeDelim: "}}",
			wantErr:    false,
			validate: func(t *testing.T, body []byte, contentType string) {
				if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
					t.Errorf("Expected multipart/form-data content type, got %s", contentType)
				}
				if !bytes.Contains(body, []byte("value1")) {
					t.Error("Expected form field 'key1' value in multipart body")
				}
				if !bytes.Contains(body, []byte("value2")) {
					t.Error("Expected form field 'key2' value in multipart body")
				}
			},
		},
		{
			name:       "invalid file format",
			files:      []string{"invalidformat"},
			formFields: []string{},
			openDelim:  "{{",
			closeDelim: "}}",
			wantErr:    true,
			validate:   nil,
		},
		{
			name:       "invalid form field format",
			files:      []string{},
			formFields: []string{"invalidformat"},
			openDelim:  "{{",
			closeDelim: "}}",
			wantErr:    true,
			validate:   nil,
		},
		{
			name:       "non-existent file",
			files:      []string{"document=/path/to/nonexistent/file.txt"},
			formFields: []string{},
			openDelim:  "{{",
			closeDelim: "}}",
			wantErr:    true,
			validate:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, contentType, err := buildMultipartRequest(tt.files, tt.formFields, tt.openDelim, tt.closeDelim)
			if (err != nil) != tt.wantErr {
				t.Errorf("buildMultipartRequest() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.validate != nil {
				tt.validate(t, body, contentType)
			}
		})
	}
}

func TestBuildMultipartRequestWithTemplates(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.txt")
	testContent := []byte("Test content")
	if err := os.WriteFile(testFile, testContent, 0600); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test with template in form field
	formFields := []string{"timestamp={{nowtime}}"}
	body, contentType, err := buildMultipartRequest([]string{}, formFields, "{{", "}}")
	if err != nil {
		t.Fatalf("buildMultipartRequest() failed: %v", err)
	}

	if !strings.HasPrefix(contentType, "multipart/form-data; boundary=") {
		t.Errorf("Expected multipart/form-data content type, got %s", contentType)
	}

	// Extract boundary from content type
	boundary := strings.TrimPrefix(contentType, "multipart/form-data; boundary=")

	// Parse the multipart body
	reader := multipart.NewReader(bytes.NewReader(body), boundary)
	part, err := reader.NextPart()
	if err != nil {
		t.Fatalf("Failed to read multipart part: %v", err)
	}

	if part.FormName() != "timestamp" {
		t.Errorf("Expected form name 'timestamp', got %s", part.FormName())
	}

	// Read the value
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(part); err != nil {
		t.Fatalf("Failed to read part content: %v", err)
	}

	// The value should not be empty (it's a timestamp)
	if buf.Len() == 0 {
		t.Error("Expected non-empty timestamp value")
	}
}

func TestSplitOnce(t *testing.T) {
	tests := []struct {
		name string
		s    string
		sep  string
		want []string
	}{
		{
			name: "normal split",
			s:    "key=value",
			sep:  "=",
			want: []string{"key", "value"},
		},
		{
			name: "multiple separators",
			s:    "key=value=extra",
			sep:  "=",
			want: []string{"key", "value=extra"},
		},
		{
			name: "no separator",
			s:    "keyvalue",
			sep:  "=",
			want: []string{"keyvalue"},
		},
		{
			name: "separator at start",
			s:    "=value",
			sep:  "=",
			want: []string{"", "value"},
		},
		{
			name: "separator at end",
			s:    "key=",
			sep:  "=",
			want: []string{"key", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitOnce(tt.s, tt.sep)
			if len(got) != len(tt.want) {
				t.Errorf("splitOnce() returned %d parts, want %d", len(got), len(tt.want))
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("splitOnce()[%d] = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}
