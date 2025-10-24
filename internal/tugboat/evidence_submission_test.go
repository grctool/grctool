// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tugboat

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
)

func TestValidateFileType(t *testing.T) {
	tests := []struct {
		name     string
		filename string
		wantErr  bool
		errMsg   string
	}{
		// Supported file types
		{
			name:     "CSV file",
			filename: "evidence.csv",
			wantErr:  false,
		},
		{
			name:     "JSON file",
			filename: "data.json",
			wantErr:  false,
		},
		{
			name:     "PDF file",
			filename: "report.pdf",
			wantErr:  false,
		},
		{
			name:     "TXT file",
			filename: "notes.txt",
			wantErr:  false,
		},
		{
			name:     "PNG image",
			filename: "screenshot.png",
			wantErr:  false,
		},
		{
			name:     "JPG image",
			filename: "photo.jpg",
			wantErr:  false,
		},
		{
			name:     "JPEG image",
			filename: "photo.jpeg",
			wantErr:  false,
		},

		// Unsupported file types
		{
			name:     "HTML file - explicitly unsupported",
			filename: "page.html",
			wantErr:  true,
			errMsg:   "not supported by Tugboat",
		},
		{
			name:     "JavaScript file - explicitly unsupported",
			filename: "script.js",
			wantErr:  true,
			errMsg:   "not supported by Tugboat",
		},
		{
			name:     "Executable - explicitly unsupported",
			filename: "program.exe",
			wantErr:  true,
			errMsg:   "not supported by Tugboat",
		},
		{
			name:     "PHP file - explicitly unsupported",
			filename: "script.php5",
			wantErr:  true,
			errMsg:   "not supported by Tugboat",
		},

		// Unknown extensions
		{
			name:     "Unknown extension",
			filename: "file.xyz",
			wantErr:  true,
			errMsg:   "not in the list of supported types",
		},
		{
			name:     "No extension",
			filename: "file",
			wantErr:  true,
			errMsg:   "no extension",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateFileType(tt.filename)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ValidateFileType() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("ValidateFileType() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("ValidateFileType() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestSubmitEvidence_ValidateFilePath(t *testing.T) {
	// Create a mock server that accepts submissions
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true}`))
	}))
	defer server.Close()

	// Create client with test credentials
	cfg := &config.TugboatConfig{
		BaseURL:  server.URL,
		Username: "testuser",
		Password: "testpass",
		Timeout:  5 * time.Second,
	}

	// Set API key for test
	os.Setenv("TUGBOAT_API_KEY", "test-api-key")
	defer os.Unsetenv("TUGBOAT_API_KEY")

	client := NewClient(cfg, nil)

	tests := []struct {
		name     string
		filePath string
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "Non-existent file",
			filePath: "/nonexistent/file.csv",
			wantErr:  true,
			errMsg:   "failed to open file",
		},
		{
			name:     "Empty file path",
			filePath: "",
			wantErr:  true,
			errMsg:   "file path is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := &SubmitEvidenceRequest{
				CollectorURL:  server.URL,
				FilePath:      tt.filePath,
				CollectedDate: time.Now(),
			}

			_, err := client.SubmitEvidence(context.Background(), req)
			if tt.wantErr {
				if err == nil {
					t.Errorf("SubmitEvidence() expected error but got nil")
					return
				}
				if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("SubmitEvidence() error = %v, want error containing %q", err, tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("SubmitEvidence() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestSubmitEvidence_FileSizeValidation(t *testing.T) {
	// Create a temporary file that exceeds the size limit
	tmpDir := t.TempDir()
	largeFile := filepath.Join(tmpDir, "large.csv")

	// Create a 21MB file (exceeds 20MB limit)
	f, err := os.Create(largeFile)
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer f.Close()

	// Write 21MB of data
	data := make([]byte, 21*1024*1024)
	if _, err := f.Write(data); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}
	f.Close()

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Create client
	cfg := &config.TugboatConfig{
		BaseURL:  server.URL,
		Username: "testuser",
		Password: "testpass",
		Timeout:  5 * time.Second,
	}
	os.Setenv("TUGBOAT_API_KEY", "test-api-key")
	defer os.Unsetenv("TUGBOAT_API_KEY")

	client := NewClient(cfg, nil)

	req := &SubmitEvidenceRequest{
		CollectorURL:  server.URL,
		FilePath:      largeFile,
		CollectedDate: time.Now(),
	}

	_, err = client.SubmitEvidence(context.Background(), req)
	if err == nil {
		t.Error("SubmitEvidence() expected error for oversized file but got nil")
		return
	}
	if !strings.Contains(err.Error(), "exceeds maximum") {
		t.Errorf("SubmitEvidence() error = %v, want error containing 'exceeds maximum'", err)
	}
}

func TestSubmitEvidence_Success(t *testing.T) {
	// Create a small test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.csv")
	testData := "header1,header2\nvalue1,value2\n"
	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Track request details
	var receivedAuth string
	var receivedAPIKey string
	var receivedContentType string
	var receivedBody []byte

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture request details
		receivedAuth = r.Header.Get("Authorization")
		receivedAPIKey = r.Header.Get("X-API-KEY")
		receivedContentType = r.Header.Get("Content-Type")

		// Read body
		body, _ := io.ReadAll(r.Body)
		receivedBody = body

		// Verify method
		if r.Method != "POST" {
			t.Errorf("Expected POST method, got %s", r.Method)
		}

		// Return success response
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"success": true, "message": "Evidence received"}`))
	}))
	defer server.Close()

	// Create client
	cfg := &config.TugboatConfig{
		BaseURL:  server.URL,
		Username: "testuser",
		Password: "testpass",
		Timeout:  5 * time.Second,
	}
	os.Setenv("TUGBOAT_API_KEY", "test-api-key-12345")
	defer os.Unsetenv("TUGBOAT_API_KEY")

	client := NewClient(cfg, nil)

	// Submit evidence
	collectedDate := time.Date(2025, 10, 23, 0, 0, 0, 0, time.UTC)
	req := &SubmitEvidenceRequest{
		CollectorURL:  server.URL + "/api/v0/evidence/collector/123/",
		FilePath:      testFile,
		CollectedDate: collectedDate,
		ContentType:   "text/csv",
	}

	resp, err := client.SubmitEvidence(context.Background(), req)
	if err != nil {
		t.Fatalf("SubmitEvidence() unexpected error = %v", err)
	}

	// Verify response
	if !resp.Success {
		t.Error("Expected Success=true in response")
	}
	if resp.Message != "Evidence received" {
		t.Errorf("Expected message 'Evidence received', got %q", resp.Message)
	}

	// Verify authentication headers
	expectedAuth := "Basic " + basicAuth("testuser", "testpass")
	if receivedAuth != expectedAuth {
		t.Errorf("Expected Authorization header %q, got %q", expectedAuth, receivedAuth)
	}
	if receivedAPIKey != "test-api-key-12345" {
		t.Errorf("Expected X-API-KEY header %q, got %q", "test-api-key-12345", receivedAPIKey)
	}

	// Verify content type
	if !strings.HasPrefix(receivedContentType, "multipart/form-data") {
		t.Errorf("Expected Content-Type to start with 'multipart/form-data', got %q", receivedContentType)
	}

	// Verify body contains multipart data
	bodyStr := string(receivedBody)
	if !strings.Contains(bodyStr, "name=\"collected\"") {
		t.Error("Request body should contain 'collected' field")
	}
	if !strings.Contains(bodyStr, "2025-10-23") {
		t.Error("Request body should contain date '2025-10-23'")
	}
	if !strings.Contains(bodyStr, "name=\"file\"") {
		t.Error("Request body should contain 'file' field")
	}
	if !strings.Contains(bodyStr, testData) {
		t.Error("Request body should contain file data")
	}
}

func TestSubmitEvidence_MissingCredentials(t *testing.T) {
	// Create a test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.csv")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	tests := []struct {
		name     string
		username string
		password string
		apiKey   string
		wantErr  string
	}{
		{
			name:     "Missing username",
			username: "",
			password: "pass",
			apiKey:   "key",
			wantErr:  "credentials not configured",
		},
		{
			name:     "Missing password",
			username: "user",
			password: "",
			apiKey:   "key",
			wantErr:  "credentials not configured",
		},
		{
			name:     "Missing API key",
			username: "user",
			password: "pass",
			apiKey:   "",
			wantErr:  "TUGBOAT_API_KEY",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.TugboatConfig{
				BaseURL:  "http://example.com",
				Username: tt.username,
				Password: tt.password,
				Timeout:  5 * time.Second,
			}

			if tt.apiKey != "" {
				os.Setenv("TUGBOAT_API_KEY", tt.apiKey)
			} else {
				os.Unsetenv("TUGBOAT_API_KEY")
			}
			defer os.Unsetenv("TUGBOAT_API_KEY")

			client := NewClient(cfg, nil)

			req := &SubmitEvidenceRequest{
				CollectorURL:  "http://example.com/collector/123/",
				FilePath:      testFile,
				CollectedDate: time.Now(),
			}

			_, err := client.SubmitEvidence(context.Background(), req)
			if err == nil {
				t.Error("Expected error for missing credentials")
				return
			}
			if !strings.Contains(err.Error(), tt.wantErr) {
				t.Errorf("Error = %v, want error containing %q", err, tt.wantErr)
			}
		})
	}
}

// Helper function to create Basic Auth string (base64 encoded)
func basicAuth(username, password string) string {
	// Return base64 encoding of "testuser:testpass"
	// In real code, http.Request.SetBasicAuth() does this encoding automatically
	return "dGVzdHVzZXI6dGVzdHBhc3M="
}
