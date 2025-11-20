package main

import (
	"bytes"
	"mime/multipart"
	"strings"
	"testing"
)

func TestIsMultipartRequest(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		want        bool
	}{
		{
			name:        "multipart/form-data",
			contentType: "multipart/form-data; boundary=----boundary",
			want:        true,
		},
		{
			name:        "multipart/mixed",
			contentType: "multipart/mixed; boundary=----boundary",
			want:        true,
		},
		{
			name:        "application/json",
			contentType: "application/json",
			want:        false,
		},
		{
			name:        "text/plain",
			contentType: "text/plain",
			want:        false,
		},
		{
			name:        "empty",
			contentType: "",
			want:        false,
		},
		{
			name:        "invalid",
			contentType: "invalid content type",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isMultipartRequest(tt.contentType)
			if got != tt.want {
				t.Errorf("isMultipartRequest() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParseMultipartRequest(t *testing.T) {
	tests := []struct {
		name         string
		setupBody    func() (string, []byte)
		wantSections int
		wantFiles    int
		wantFields   int
	}{
		{
			name: "file and form fields",
			setupBody: func() (string, []byte) {
				var buf bytes.Buffer
				writer := multipart.NewWriter(&buf)

				// Add form field
				writer.WriteField("username", "testuser")
				writer.WriteField("email", "test@example.com")

				// Add file
				part, _ := writer.CreateFormFile("document", "test.txt")
				part.Write([]byte("file content"))

				writer.Close()
				return writer.FormDataContentType(), buf.Bytes()
			},
			wantSections: 2, // Form Fields + Files
			wantFiles:    1,
			wantFields:   2,
		},
		{
			name: "only form fields",
			setupBody: func() (string, []byte) {
				var buf bytes.Buffer
				writer := multipart.NewWriter(&buf)

				writer.WriteField("key1", "value1")
				writer.WriteField("key2", "value2")

				writer.Close()
				return writer.FormDataContentType(), buf.Bytes()
			},
			wantSections: 1, // Form Fields only
			wantFiles:    0,
			wantFields:   2,
		},
		{
			name: "only files",
			setupBody: func() (string, []byte) {
				var buf bytes.Buffer
				writer := multipart.NewWriter(&buf)

				part, _ := writer.CreateFormFile("file1", "test1.txt")
				part.Write([]byte("content1"))

				part, _ = writer.CreateFormFile("file2", "test2.txt")
				part.Write([]byte("content2"))

				writer.Close()
				return writer.FormDataContentType(), buf.Bytes()
			},
			wantSections: 1, // Files only
			wantFiles:    2,
			wantFields:   0,
		},
		{
			name: "invalid content type",
			setupBody: func() (string, []byte) {
				return "invalid", []byte("body")
			},
			wantSections: 0,
			wantFiles:    0,
			wantFields:   0,
		},
		{
			name: "missing boundary",
			setupBody: func() (string, []byte) {
				return "multipart/form-data", []byte("body")
			},
			wantSections: 0,
			wantFiles:    0,
			wantFields:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			contentType, body := tt.setupBody()
			sections, bodyStr := parseMultipartRequest(contentType, body)

			if tt.wantSections == 0 {
				if sections != nil {
					t.Errorf("parseMultipartRequest() expected nil sections, got %d sections", len(sections))
				}
				if bodyStr != "" {
					t.Errorf("parseMultipartRequest() expected empty body string, got %q", bodyStr)
				}
				return
			}

			if len(sections) != tt.wantSections {
				t.Errorf("parseMultipartRequest() got %d sections, want %d", len(sections), tt.wantSections)
			}

			var gotFiles, gotFields int
			for _, section := range sections {
				if section.Title == "Files" {
					gotFiles = len(section.Items)
				}
				if section.Title == "Form Fields" {
					gotFields = len(section.Items)
				}
			}

			if gotFiles != tt.wantFiles {
				t.Errorf("parseMultipartRequest() got %d files, want %d", gotFiles, tt.wantFiles)
			}

			if gotFields != tt.wantFields {
				t.Errorf("parseMultipartRequest() got %d fields, want %d", gotFields, tt.wantFields)
			}

			// Check that body string contains expected info
			if tt.wantFiles > 0 && !strings.Contains(bodyStr, "[File:") {
				t.Error("parseMultipartRequest() body string should contain file info")
			}

			if tt.wantFields > 0 {
				hasFieldMarker := false
				for _, section := range sections {
					if section.Title == "Form Fields" && len(section.Items) > 0 {
						hasFieldMarker = true
						break
					}
				}
				if !hasFieldMarker {
					t.Error("parseMultipartRequest() should have form fields section")
				}
			}
		})
	}
}

func TestParseMultipartRequestFileInfo(t *testing.T) {
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	// Create a file with known size
	content := []byte("test file content with specific length")
	part, _ := writer.CreateFormFile("document", "testfile.pdf")
	part.Write(content)

	writer.Close()

	contentType := writer.FormDataContentType()
	sections, _ := parseMultipartRequest(contentType, buf.Bytes())

	if len(sections) == 0 {
		t.Fatal("Expected at least one section")
	}

	var filesSection *struct {
		Title string
		Items []struct{ Key, Value string }
	}

	for i := range sections {
		if sections[i].Title == "Files" {
			// Create a temporary variable that matches the section structure
			temp := struct {
				Title string
				Items []struct{ Key, Value string }
			}{
				Title: sections[i].Title,
			}
			// Convert the KV items
			for _, item := range sections[i].Items {
				temp.Items = append(temp.Items, struct{ Key, Value string }{
					Key:   item.Key,
					Value: item.Value,
				})
			}
			filesSection = &temp
			break
		}
	}

	if filesSection == nil {
		t.Fatal("Expected Files section")
	}

	if len(filesSection.Items) != 1 {
		t.Fatalf("Expected 1 file, got %d", len(filesSection.Items))
	}

	fileInfo := filesSection.Items[0].Value
	if !strings.Contains(fileInfo, "testfile.pdf") {
		t.Errorf("File info should contain filename, got %q", fileInfo)
	}

	if !strings.Contains(fileInfo, "bytes") {
		t.Errorf("File info should contain size in bytes, got %q", fileInfo)
	}
}
