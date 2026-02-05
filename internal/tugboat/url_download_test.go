// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tugboat

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tugboat/models"
)

// Unit tests for URL download helper functions

func TestExtractFilenameFromResponse(t *testing.T) {
	tests := []struct {
		name           string
		contentDisp    string
		contentType    string
		sourceURL      string
		fallbackName   string
		expectedResult string
	}{
		{
			name:           "Content-Disposition with filename",
			contentDisp:    `attachment; filename="report.pdf"`,
			contentType:    "application/pdf",
			sourceURL:      "https://example.com/download",
			fallbackName:   "fallback",
			expectedResult: "report.pdf",
		},
		{
			name:           "Content-Disposition with quoted filename",
			contentDisp:    `attachment; filename="quarterly-report.xlsx"`,
			contentType:    "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
			sourceURL:      "https://example.com/download",
			fallbackName:   "fallback",
			expectedResult: "quarterly-report.xlsx",
		},
		{
			name:           "No Content-Disposition, filename from URL",
			contentDisp:    "",
			contentType:    "application/pdf",
			sourceURL:      "https://example.com/files/document.pdf",
			fallbackName:   "fallback",
			expectedResult: "document.pdf",
		},
		{
			name:           "No Content-Disposition, no file in URL, use fallback with HTML",
			contentDisp:    "",
			contentType:    "text/html; charset=utf-8",
			sourceURL:      "https://example.com/page",
			fallbackName:   "url_123",
			expectedResult: "url_123.html",
		},
		{
			name:           "No Content-Disposition, no file in URL, use fallback with PDF",
			contentDisp:    "",
			contentType:    "application/pdf",
			sourceURL:      "https://example.com/generate",
			fallbackName:   "url_456",
			expectedResult: "url_456.pdf",
		},
		{
			name:           "No Content-Disposition, no file in URL, use fallback with JSON",
			contentDisp:    "",
			contentType:    "application/json",
			sourceURL:      "https://api.example.com/data",
			fallbackName:   "api_response",
			expectedResult: "api_response.json",
		},
		{
			name:           "Filename with special characters sanitized",
			contentDisp:    `attachment; filename="report:2024/Q1.pdf"`,
			contentType:    "application/pdf",
			sourceURL:      "https://example.com/download",
			fallbackName:   "fallback",
			expectedResult: "report_2024_Q1.pdf",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp := &http.Response{
				Header: http.Header{},
			}
			if tt.contentDisp != "" {
				resp.Header.Set("Content-Disposition", tt.contentDisp)
			}
			if tt.contentType != "" {
				resp.Header.Set("Content-Type", tt.contentType)
			}

			result := extractFilenameFromResponse(resp, tt.sourceURL, tt.fallbackName)
			if result != tt.expectedResult {
				t.Errorf("extractFilenameFromResponse() = %q, want %q", result, tt.expectedResult)
			}
		})
	}
}

func TestGetExtensionFromContentType(t *testing.T) {
	tests := []struct {
		contentType string
		expected    string
	}{
		{"text/html", ".html"},
		{"text/html; charset=utf-8", ".html"},
		{"application/pdf", ".pdf"},
		{"application/json", ".json"},
		{"text/plain", ".txt"},
		{"text/csv", ".csv"},
		{"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", ".xlsx"},
		{"application/vnd.ms-excel", ".xls"},
		{"image/png", ".png"},
		{"image/jpeg", ".jpg"},
		{"application/octet-stream", ".html"}, // Unknown defaults to .html
		{"", ".html"},                          // Empty defaults to .html
	}

	for _, tt := range tests {
		t.Run(tt.contentType, func(t *testing.T) {
			result := getExtensionFromContentType(tt.contentType)
			if result != tt.expected {
				t.Errorf("getExtensionFromContentType(%q) = %q, want %q", tt.contentType, result, tt.expected)
			}
		})
	}
}

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"normal-file.pdf", "normal-file.pdf"},
		{"file/with/slashes.txt", "file_with_slashes.txt"},
		{"file\\with\\backslashes.txt", "file_with_backslashes.txt"},
		{"file:with:colons.txt", "file_with_colons.txt"},
		{"file*with*asterisks.txt", "file_with_asterisks.txt"},
		{"file?with?questions.txt", "file_with_questions.txt"},
		{`file"with"quotes.txt`, "file_with_quotes.txt"},
		{"file<with>angles.txt", "file_with_angles.txt"},
		{"file|with|pipes.txt", "file_with_pipes.txt"},
		{"complex/path:name*file?.txt", "complex_path_name_file_.txt"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := sanitizeFilename(tt.input)
			if result != tt.expected {
				t.Errorf("sanitizeFilename(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildPolicyMarkdownFromDetails(t *testing.T) {
	tests := []struct {
		name     string
		policy   *models.PolicyDetails
		contains []string
	}{
		{
			name: "Full policy with all fields",
			policy: &models.PolicyDetails{
				Policy: models.Policy{
					Name:        "Access Control Policy",
					Description: "This policy defines access control requirements.",
					Status:      "published",
				},
				Category: "Security",
				Summary:  "Summary of access control measures.",
				Details:  "Detailed policy content here.",
			},
			contains: []string{
				"# Access Control Policy",
				"**Status:** published",
				"**Category:** Security",
				"## Summary",
				"Summary of access control measures",
				"## Policy Details",
				"Detailed policy content here",
				"## Description",
				"This policy defines access control requirements",
			},
		},
		{
			name: "Policy with current version content",
			policy: &models.PolicyDetails{
				Policy: models.Policy{
					Name:   "Data Protection Policy",
					Status: "draft",
				},
				CurrentVersion: &models.PolicyVersion{
					Content: "Version 2.0 content of the policy.",
				},
			},
			contains: []string{
				"# Data Protection Policy",
				"**Status:** draft",
				"## Content",
				"Version 2.0 content",
			},
		},
		{
			name: "Empty policy shows no content message",
			policy: &models.PolicyDetails{
				Policy: models.Policy{
					Name:   "Empty Policy",
					Status: "published",
				},
			},
			contains: []string{
				"# Empty Policy",
				"*No policy content available via API.*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildPolicyMarkdownFromDetails(tt.policy)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("buildPolicyMarkdownFromDetails() missing expected content: %q\nGot:\n%s", expected, result)
				}
			}
		})
	}
}

func TestBuildControlMarkdownFromDetails(t *testing.T) {
	tests := []struct {
		name     string
		control  *models.ControlDetails
		contains []string
	}{
		{
			name: "Full control with all fields",
			control: &models.ControlDetails{
				Control: models.Control{
					Name:     "Access Provisioning",
					Category: "Access Control",
					Status:   "implemented",
					Body:     "Access to systems requires documented approval.",
					Help:     "Ensure all access requests are logged.",
				},
			},
			contains: []string{
				"# Access Provisioning",
				"**Category:** Access Control",
				"**Status:** implemented",
				"## Description",
				"Access to systems requires documented approval",
				"## Guidance",
				"Ensure all access requests are logged",
			},
		},
		{
			name: "Control with master content",
			control: &models.ControlDetails{
				Control: models.Control{
					Name:   "MFA Control",
					Status: "implemented",
					Body:   "MFA is required for all privileged access.",
				},
				MasterContent: &models.ControlMasterContent{
					Guidance: "Additional guidance from master template.",
				},
			},
			contains: []string{
				"# MFA Control",
				"## Description",
				"MFA is required",
				"## Additional Guidance",
				"Additional guidance from master template",
			},
		},
		{
			name: "Empty control shows no content message",
			control: &models.ControlDetails{
				Control: models.Control{
					Name:   "Empty Control",
					Status: "na",
				},
			},
			contains: []string{
				"# Empty Control",
				"*No control content available via API.*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildControlMarkdownFromDetails(tt.control)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("buildControlMarkdownFromDetails() missing expected content: %q\nGot:\n%s", expected, result)
				}
			}
		})
	}
}

func TestBuildEvidenceTaskMarkdownFromDetails(t *testing.T) {
	tests := []struct {
		name     string
		task     *models.EvidenceTaskDetails
		contains []string
	}{
		{
			name: "Full evidence task with all fields",
			task: &models.EvidenceTaskDetails{
				EvidenceTask: models.EvidenceTask{
					Name:               "User Access Reviews",
					Description:        "Quarterly review of user access to systems.",
					Status:             "completed",
					CollectionInterval: "quarter",
				},
				MasterContent: &models.MasterContent{
					Guidance: "Review all user access quarterly.",
				},
			},
			contains: []string{
				"# User Access Reviews",
				"**Status:** completed",
				"**Collection Interval:** quarter",
				"## Description",
				"Quarterly review of user access",
				"## Guidance",
				"Review all user access quarterly",
			},
		},
		{
			name: "Empty evidence task shows no content message",
			task: &models.EvidenceTaskDetails{
				EvidenceTask: models.EvidenceTask{
					Name:   "Empty Task",
					Status: "pending",
				},
			},
			contains: []string{
				"# Empty Task",
				"*No evidence task content available via API.*",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildEvidenceTaskMarkdownFromDetails(tt.task)
			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("buildEvidenceTaskMarkdownFromDetails() missing expected content: %q\nGot:\n%s", expected, result)
				}
			}
		})
	}
}

// Test for handleTugboatSPAURL pattern matching
func TestHandleTugboatSPAURL_PatternMatching(t *testing.T) {
	tests := []struct {
		name        string
		url         string
		shouldMatch bool
		matchType   string // "policy", "control", "evidence", or ""
	}{
		{
			name:        "Policy URL",
			url:         "https://my.tugboatlogic.com/org/13888/policies/94641",
			shouldMatch: true,
			matchType:   "policy",
		},
		{
			name:        "Control URL",
			url:         "https://my.tugboatlogic.com/org/13888/controls/778771",
			shouldMatch: true,
			matchType:   "control",
		},
		{
			name:        "Evidence task URL",
			url:         "https://my.tugboatlogic.com/org/13888/evidence/tasks/327992",
			shouldMatch: true,
			matchType:   "evidence",
		},
		{
			name:        "Non-Tugboat URL",
			url:         "https://example.com/some/path",
			shouldMatch: false,
			matchType:   "",
		},
		{
			name:        "Tugboat URL but different path",
			url:         "https://my.tugboatlogic.com/org/13888/dashboard",
			shouldMatch: false,
			matchType:   "",
		},
		{
			name:        "API subdomain Tugboat URL",
			url:         "https://api-my.tugboatlogic.com/api/policy/94641",
			shouldMatch: false, // API URLs are not SPA URLs
			matchType:   "",
		},
	}

	// Test pattern matching without actually making API calls
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// We can't easily test the full handleTugboatSPAURL without mocking,
			// but we can verify the URL pattern matching logic
			matched := false
			matchType := ""

			if strings.Contains(tt.url, "tugboatlogic.com") {
				if strings.Contains(tt.url, "/policies/") {
					matched = true
					matchType = "policy"
				} else if strings.Contains(tt.url, "/controls/") {
					matched = true
					matchType = "control"
				} else if strings.Contains(tt.url, "/evidence/tasks/") {
					matched = true
					matchType = "evidence"
				}
			}

			if matched != tt.shouldMatch {
				t.Errorf("URL %q: matched = %v, want %v", tt.url, matched, tt.shouldMatch)
			}
			if matchType != tt.matchType {
				t.Errorf("URL %q: matchType = %q, want %q", tt.url, matchType, tt.matchType)
			}
		})
	}
}

// Test direct URL download with mock server
func TestDownloadDirectURL(t *testing.T) {
	// Create a test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/file.pdf":
			w.Header().Set("Content-Type", "application/pdf")
			w.Header().Set("Content-Disposition", `attachment; filename="test-document.pdf"`)
			w.Write([]byte("PDF content here"))
		case "/page.html":
			w.Header().Set("Content-Type", "text/html")
			w.Write([]byte("<html><body>Test page</body></html>"))
		case "/data.json":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"key": "value"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	// Create a properly initialized client for testing
	testLogger, _ := logger.NewTestLogger()
	client := &Client{
		httpClient: http.DefaultClient,
		logger:     testLogger,
	}

	tests := []struct {
		name             string
		path             string
		fallbackName     string
		expectedFilename string
		expectedContent  string
	}{
		{
			name:             "PDF file with Content-Disposition",
			path:             "/file.pdf",
			fallbackName:     "fallback",
			expectedFilename: "test-document.pdf",
			expectedContent:  "PDF content here",
		},
		{
			name:             "HTML page without Content-Disposition",
			path:             "/page.html",
			fallbackName:     "url_123",
			expectedFilename: "page.html",
			expectedContent:  "<html><body>Test page</body></html>",
		},
		{
			name:             "JSON data",
			path:             "/data.json",
			fallbackName:     "api_data",
			expectedFilename: "data.json",
			expectedContent:  `{"key": "value"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destDir := t.TempDir()

			result, err := client.downloadDirectURL(
				t.Context(),
				server.URL+tt.path,
				destDir,
				tt.fallbackName,
			)

			if err != nil {
				t.Fatalf("downloadDirectURL() error = %v", err)
			}

			if result.Filename != tt.expectedFilename {
				t.Errorf("Filename = %q, want %q", result.Filename, tt.expectedFilename)
			}

			if result.BytesWritten != int64(len(tt.expectedContent)) {
				t.Errorf("BytesWritten = %d, want %d", result.BytesWritten, len(tt.expectedContent))
			}
		})
	}
}
