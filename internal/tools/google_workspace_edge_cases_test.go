// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build e2e

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoogleWorkspaceEdgeCases tests various edge cases and boundary conditions
func TestGoogleWorkspaceEdgeCases(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	cfgLogger := &logger.Config{
		Level:  logger.DebugLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(t, err)

	tool := NewGoogleWorkspaceTool(cfg, log)
	gwt := tool.(*GoogleWorkspaceTool)

	t.Run("Time_Parsing_Edge_Cases", func(t *testing.T) {
		testCases := []struct {
			input    string
			expected bool // true if should parse successfully
		}{
			{"2023-06-01T12:00:00.000Z", true},
			{"2023-06-01T12:00:00Z", true},
			{"2023-06-01T12:00:00.123456Z", true},
			{"", false},
			{"invalid-time", false},
			{"2023-13-01T12:00:00Z", false}, // Invalid month
			{"2023-06-32T12:00:00Z", false}, // Invalid day
			{"2023-06-01T25:00:00Z", false}, // Invalid hour
			{"not-a-timestamp", false},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("Parse_%s", strings.ReplaceAll(tc.input, ":", "_")), func(t *testing.T) {
				result := gwt.parseGoogleTime(tc.input)
				if tc.expected {
					assert.False(t, result.IsZero(), "Expected successful parse for %s", tc.input)
				} else {
					assert.True(t, result.IsZero(), "Expected failed parse for %s", tc.input)
				}
			})
		}
	})

	t.Run("MIME_Type_Edge_Cases", func(t *testing.T) {
		testCases := []struct {
			mimeType string
			expected string
		}{
			{"application/vnd.google-apps.document", "document"},
			{"application/vnd.google-apps.spreadsheet", "spreadsheet"},
			{"application/vnd.google-apps.presentation", "presentation"},
			{"application/vnd.google-apps.form", "form"},
			{"application/vnd.google-apps.folder", "folder"},
			{"application/vnd.google-apps.drawing", "drawing"},
			{"application/pdf", "file"},
			{"image/jpeg", "file"},
			{"text/plain", "file"},
			{"", "file"}, // Empty mime type
			{"invalid/mime/type/with/too/many/parts", "file"},
			{"application/vnd.unknown", "file"},
		}

		for _, tc := range testCases {
			t.Run(fmt.Sprintf("MIME_%s", strings.ReplaceAll(tc.mimeType, "/", "_")), func(t *testing.T) {
				result := gwt.getMimeTypeCategory(tc.mimeType)
				assert.Equal(t, tc.expected, result)
			})
		}
	})

	t.Run("Relevance_Calculation_Edge_Cases", func(t *testing.T) {
		testCases := []struct {
			name     string
			result   *GoogleWorkspaceResult
			expected float64 // Expected relevance range minimum
		}{
			{
				"Empty_Content",
				&GoogleWorkspaceResult{
					Content:      "",
					DocumentType: "docs",
				},
				0.5, // Base relevance
			},
			{
				"Extremely_Long_Content",
				&GoogleWorkspaceResult{
					Content:      strings.Repeat("A", 50000), // 50K characters
					DocumentType: "docs",
				},
				0.8, // Should get content boost
			},
			{
				"Very_Recent_Modification",
				&GoogleWorkspaceResult{
					Content:      "Some content",
					DocumentType: "docs",
					ModifiedAt:   time.Now().Add(-1 * time.Hour), // 1 hour ago
				},
				0.7, // Base + recent modification boost
			},
			{
				"Very_Old_Modification",
				&GoogleWorkspaceResult{
					Content:      "Some content",
					DocumentType: "docs",
					ModifiedAt:   time.Now().AddDate(-1, 0, 0), // 1 year ago
				},
				0.5, // Just base relevance
			},
			{
				"Zero_Time_Modification",
				&GoogleWorkspaceResult{
					Content:      "Some content",
					DocumentType: "docs",
					ModifiedAt:   time.Time{}, // Zero time
				},
				0.5, // Just base relevance
			},
			{
				"Large_Folder",
				&GoogleWorkspaceResult{
					Content:        "Folder content",
					DocumentType:   "folder",
					FolderContents: make([]FolderItem, 50), // 50 items
				},
				0.9, // Should get folder boost
			},
			{
				"Forms_With_Content",
				&GoogleWorkspaceResult{
					Content:      strings.Repeat("Form content ", 100), // Decent content
					DocumentType: "forms",
					ModifiedAt:   time.Now().Add(-24 * time.Hour), // Yesterday
				},
				0.9, // Should get forms boost + content + recent
			},
			{
				"Sheets_With_Content",
				&GoogleWorkspaceResult{
					Content:      strings.Repeat("Sheet data ", 200), // More content
					DocumentType: "sheets",
				},
				0.75, // Should get sheets boost + content (adjusted for floating point precision)
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				relevance := gwt.calculateRelevance(tc.result)
				assert.GreaterOrEqual(t, relevance, tc.expected)
				assert.LessOrEqual(t, relevance, 1.0)    // Should never exceed 1.0
				assert.GreaterOrEqual(t, relevance, 0.0) // Should never be negative
			})
		}
	})

	t.Run("Report_Generation_Edge_Cases", func(t *testing.T) {
		// Empty result
		result := &GoogleWorkspaceResult{
			DocumentID:   "empty-doc",
			DocumentName: "",
			DocumentType: "docs",
		}
		rules := ExtractionRules{}

		report := gwt.generateReport(result, "docs", rules)
		assert.Contains(t, report, "# Google Workspace Evidence Report")
		assert.Contains(t, report, "*No content extracted*")

		// Result with all possible fields populated
		result = &GoogleWorkspaceResult{
			DocumentID:   "full-doc",
			DocumentName: "Complete Document",
			DocumentType: "sheets",
			Owner:        "Test Owner",
			Editors:      []string{"Editor1", "Editor2"},
			CreatedAt:    time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
			ModifiedAt:   time.Date(2023, 6, 1, 15, 30, 0, 0, time.UTC),
			MimeType:     "application/vnd.google-apps.spreadsheet",
			Content:      "Rich content here",
			SheetData: [][]interface{}{
				{"Header1", "Header2"},
				{"Data1", "Data2"},
			},
			FolderContents: []FolderItem{
				{
					ID:       "file1",
					Name:     "File 1",
					Type:     "document",
					Owner:    "Owner 1",
					Size:     1024,
					Modified: time.Date(2023, 5, 1, 9, 0, 0, 0, time.UTC),
				},
			},
			Metadata: map[string]interface{}{
				"custom_field": "custom_value",
				"number_field": 42,
			},
		}

		rules = ExtractionRules{
			IncludeMetadata: true,
		}

		report = gwt.generateReport(result, "sheets", rules)

		// Check all sections are present
		assert.Contains(t, report, "## Document Metadata")
		assert.Contains(t, report, "## Folder Contents")
		assert.Contains(t, report, "## Sheet Data Summary")
		assert.Contains(t, report, "## Document Content")
		assert.Contains(t, report, "## Additional Metadata")
		assert.Contains(t, report, "- **Editors**: Editor1, Editor2")
		assert.Contains(t, report, "- **Total Rows**: 2")
		assert.Contains(t, report, "- **custom_field**: custom_value")
	})
}

// TestGoogleWorkspaceNetworkConditions tests various network conditions and API responses
func TestGoogleWorkspaceNetworkConditions(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	cfgLogger := &logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(t, err)

	t.Run("API_Rate_Limiting", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++

			if strings.Contains(r.URL.Path, "/token") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`))
				return
			}

			// Simulate rate limiting after 3 requests
			if requestCount > 3 {
				w.WriteHeader(http.StatusTooManyRequests)
				w.Header().Set("Retry-After", "60")
				w.Write([]byte(`{"error": {"code": 429, "message": "Rate limit exceeded"}}`))
				return
			}

			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Not found"}`))
		}))
		defer server.Close()

		credPath := createTestCredentials(t, tempDir, server.URL)
		tool := NewGoogleWorkspaceTool(cfg, log)
		ctx := context.Background()

		params := map[string]interface{}{
			"document_id":      "test-doc-id",
			"credentials_path": credPath,
		}

		// First few requests should work (but fail with not found)
		for i := 0; i < 3; i++ {
			_, _, err := tool.Execute(ctx, params)
			assert.Error(t, err) // Will error due to not found, not rate limit
		}

		// Next request should hit rate limit
		_, _, err = tool.Execute(ctx, params)
		assert.Error(t, err)
	})

	t.Run("Network_Timeouts", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/token") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`))
				return
			}

			// Simulate timeout by taking too long
			time.Sleep(5 * time.Second)
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		credPath := createTestCredentials(t, tempDir, server.URL)
		tool := NewGoogleWorkspaceTool(cfg, log)

		// Use a context with timeout
		ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
		defer cancel()

		params := map[string]interface{}{
			"document_id":      "test-doc-id",
			"credentials_path": credPath,
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
		// Note: The actual timeout might be handled by the underlying HTTP client
	})

	t.Run("Malformed_API_Responses", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/token") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`))
				return
			}

			// Return malformed JSON
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"incomplete": "json structure"`)) // Missing required fields
		}))
		defer server.Close()

		credPath := createTestCredentials(t, tempDir, server.URL)
		tool := NewGoogleWorkspaceTool(cfg, log)
		ctx := context.Background()

		params := map[string]interface{}{
			"document_id":      "test-doc-id",
			"credentials_path": credPath,
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
		// The error should indicate API response issues
	})

	t.Run("Empty_API_Responses", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/token") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`))
				return
			}

			// Return empty response
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{}`)) // Valid JSON but empty
		}))
		defer server.Close()

		credPath := createTestCredentials(t, tempDir, server.URL)
		tool := NewGoogleWorkspaceTool(cfg, log)
		ctx := context.Background()

		params := map[string]interface{}{
			"document_id":      "test-doc-id",
			"credentials_path": credPath,
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
	})
}

// TestGoogleWorkspaceDataValidation tests data validation and boundary conditions
func TestGoogleWorkspaceDataValidation(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	cfgLogger := &logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(t, err)

	t.Run("Large_Sheet_Data", func(t *testing.T) {
		// Test handling of large spreadsheet data
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/token") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`))
				return
			}

			if strings.Contains(r.URL.Path, "/sheets/v4/spreadsheets/") && strings.Contains(r.URL.Path, "/values/") {
				// Generate large dataset
				var values [][]interface{}
				values = append(values, []interface{}{"ID", "Name", "Department", "Role", "Status"})

				for i := 0; i < 1000; i++ {
					values = append(values, []interface{}{
						fmt.Sprintf("EMP%04d", i),
						fmt.Sprintf("Employee %d", i),
						"Engineering",
						"Developer",
						"Active",
					})
				}

				response := map[string]interface{}{
					"range":          "Sheet1!A:E",
					"majorDimension": "ROWS",
					"values":         values,
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
				return
			}

			if strings.Contains(r.URL.Path, "/sheets/v4/spreadsheets/") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{
					"spreadsheetId": "large-sheet-id",
					"properties": {"title": "Large Employee Dataset"},
					"sheets": [{"properties": {"title": "Employees"}}]
				}`))
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		credPath := createTestCredentials(t, tempDir, server.URL)
		tool := NewGoogleWorkspaceTool(cfg, log)
		ctx := context.Background()

		params := map[string]interface{}{
			"document_id":      "large-sheet-id",
			"document_type":    "sheets",
			"credentials_path": credPath,
		}

		start := time.Now()
		report, source, err := tool.Execute(ctx, params)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.NotEmpty(t, report)
		assert.NotNil(t, source)
		assert.Contains(t, report, "Rows: 1001") // 1000 data rows + 1 header
		assert.Less(t, duration, 30*time.Second, "Large dataset processing took too long")
	})

	t.Run("Complex_Document_Structure", func(t *testing.T) {
		// Test handling of documents with complex nested structures
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/token") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`))
				return
			}

			if strings.Contains(r.URL.Path, "/docs/v1/documents/") {
				// Create a complex document structure with multiple paragraphs and elements
				var content []map[string]interface{}

				for i := 0; i < 100; i++ {
					content = append(content, map[string]interface{}{
						"paragraph": map[string]interface{}{
							"elements": []map[string]interface{}{
								{
									"textRun": map[string]interface{}{
										"content": fmt.Sprintf("This is paragraph %d with some detailed content about security policies and procedures. ", i),
									},
								},
								{
									"textRun": map[string]interface{}{
										"content": fmt.Sprintf("Additional text in paragraph %d with more information.\n", i),
									},
								},
							},
						},
					})
				}

				response := map[string]interface{}{
					"documentId": "complex-doc-id",
					"title":      "Complex Security Document",
					"revisionId": "complex-revision-123",
					"body": map[string]interface{}{
						"content": content,
					},
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		credPath := createTestCredentials(t, tempDir, server.URL)
		tool := NewGoogleWorkspaceTool(cfg, log)
		ctx := context.Background()

		params := map[string]interface{}{
			"document_id":      "complex-doc-id",
			"document_type":    "docs",
			"credentials_path": credPath,
		}

		report, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, report)
		assert.NotNil(t, source)

		// Check that complex content was parsed
		assert.Contains(t, source.Content, "paragraph 0")
		assert.Contains(t, source.Content, "paragraph 50")
		assert.Contains(t, source.Content, "paragraph 99")
		assert.True(t, len(source.Content) > 10000, "Complex document should have substantial content")
	})

	t.Run("Unicode_And_Special_Characters", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/token") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`))
				return
			}

			if strings.Contains(r.URL.Path, "/docs/v1/documents/") {
				response := map[string]interface{}{
					"documentId": "unicode-doc-id",
					"title":      "Document with Special Characters: 测试文档 🔐 Sécurité",
					"revisionId": "unicode-revision-123",
					"body": map[string]interface{}{
						"content": []map[string]interface{}{
							{
								"paragraph": map[string]interface{}{
									"elements": []map[string]interface{}{
										{
											"textRun": map[string]interface{}{
												"content": "Security Policy 安全政策\n\nThis document contains special characters: ñáéíóú, çüß, αβγδε, 日本語, العربية\n\nEmojis: 🔒 🛡️ ⚠️ ✅ ❌\n\nCode symbols: <>&\"'\n\nMath: ∑∫∂∇\n",
											},
										},
									},
								},
							},
						},
					},
				}

				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				json.NewEncoder(w).Encode(response)
				return
			}

			w.WriteHeader(http.StatusNotFound)
		}))
		defer server.Close()

		credPath := createTestCredentials(t, tempDir, server.URL)
		tool := NewGoogleWorkspaceTool(cfg, log)
		ctx := context.Background()

		params := map[string]interface{}{
			"document_id":      "unicode-doc-id",
			"document_type":    "docs",
			"credentials_path": credPath,
		}

		report, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, report)
		assert.NotNil(t, source)

		// Check that Unicode and special characters are preserved
		assert.Contains(t, source.Content, "安全政策")
		assert.Contains(t, source.Content, "🔒")
		assert.Contains(t, source.Content, "ñáéíóú")
		assert.Contains(t, source.Content, "العربية")
		assert.Contains(t, source.Content, "<>&\"'")
		assert.Contains(t, report, "测试文档")
	})
}

// TestGoogleWorkspaceValidationEdgeCases tests validation with edge cases
func TestGoogleWorkspaceValidationEdgeCases(t *testing.T) {
	tempDir := t.TempDir()

	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	cfgLogger := &logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(t, err)

	enhancedTool := NewGoogleWorkspaceToolWithMappings(cfg, log)

	t.Run("Boundary_Content_Lengths", func(t *testing.T) {
		testCases := []struct {
			name          string
			contentLength int
			minLength     int
			shouldPass    bool
		}{
			{"Exactly_Minimum", 100, 100, true},
			{"One_Below_Minimum", 99, 100, false},
			{"One_Above_Minimum", 101, 100, true},
			{"Zero_Length", 0, 1, false},
			{"Zero_Minimum", 50, 0, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				docConfig := DocumentConfig{
					DocumentID:   "test-doc",
					DocumentName: "Test Document",
					DocumentType: "docs",
					Validation: GoogleValidationRules{
						MinContentLength: tc.minLength,
					},
				}

				content := strings.Repeat("A", tc.contentLength)
				source := &models.EvidenceSource{
					Content: content,
				}

				validation := enhancedTool.validateDocumentContent(docConfig, source)
				if tc.shouldPass {
					assert.True(t, validation["passed"].(bool), "Expected validation to pass for %s", tc.name)
				} else {
					assert.False(t, validation["passed"].(bool), "Expected validation to fail for %s", tc.name)
				}
			})
		}
	})

	t.Run("Case_Insensitive_Keywords", func(t *testing.T) {
		docConfig := DocumentConfig{
			DocumentID:   "test-doc",
			DocumentName: "Test Document",
			DocumentType: "docs",
			Validation: GoogleValidationRules{
				RequiredKeywords: []string{"Security", "ACCESS", "Policy"},
			},
		}

		testCases := []struct {
			content          string
			expectedWarnings int
		}{
			{"This document has security, access, and policy information.", 0},
			{"This document has SECURITY, ACCESS, and POLICY information.", 0},
			{"This document has Security, Access, and Policy information.", 0},
			{"This document has security and access information.", 1}, // Missing "policy"
			{"This document has security information.", 2},            // Missing "access" and "policy"
			{"This document has no relevant keywords.", 3},            // Missing all
			{"", 3}, // Empty content, missing all
		}

		for i, tc := range testCases {
			t.Run(fmt.Sprintf("Case_%d", i), func(t *testing.T) {
				source := &models.EvidenceSource{
					Content: tc.content,
				}

				validation := enhancedTool.validateDocumentContent(docConfig, source)
				warnings := validation["warnings"].([]string)
				assert.Len(t, warnings, tc.expectedWarnings, "Expected %d warnings for content: %s", tc.expectedWarnings, tc.content)
			})
		}
	})

	t.Run("Date_Range_Edge_Cases", func(t *testing.T) {
		docConfig := DocumentConfig{
			DocumentID:   "test-doc",
			DocumentName: "Test Document",
			DocumentType: "docs",
			Validation: GoogleValidationRules{
				DateRange: &DateRange{
					From: "2023-01-01",
					To:   "2023-12-31",
				},
			},
		}

		testCases := []struct {
			name     string
			date     time.Time
			warnings int
		}{
			{"Within_Range", time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC), 0},
			{"Start_Boundary", time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC), 0},
			{"End_Boundary", time.Date(2023, 12, 31, 23, 59, 59, 0, time.UTC), 0},
			{"Before_Range", time.Date(2022, 12, 31, 23, 59, 59, 0, time.UTC), 1},
			{"After_Range", time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC), 1},
			{"Way_Before", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), 1},
			{"Way_After", time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC), 1},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				source := &models.EvidenceSource{
					Content: "Test content",
					Metadata: map[string]interface{}{
						"modified_at": tc.date,
					},
				}

				validation := enhancedTool.validateDocumentContent(docConfig, source)
				warnings := validation["warnings"].([]string)
				assert.Len(t, warnings, tc.warnings, "Expected %d warnings for date: %s", tc.warnings, tc.date.String())
			})
		}
	})

	t.Run("Invalid_Date_Formats", func(t *testing.T) {
		docConfig := DocumentConfig{
			DocumentID:   "test-doc",
			DocumentName: "Test Document",
			DocumentType: "docs",
			Validation: GoogleValidationRules{
				DateRange: &DateRange{
					From: "invalid-date-format",
					To:   "2023-13-45", // Invalid date
				},
			},
		}

		source := &models.EvidenceSource{
			Content: "Test content",
			Metadata: map[string]interface{}{
				"modified_at": time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
			},
		}

		// Should not crash with invalid date formats
		validation := enhancedTool.validateDocumentContent(docConfig, source)
		assert.NotNil(t, validation)
		assert.True(t, validation["passed"].(bool)) // Should pass since date validation fails gracefully
	})

	t.Run("Missing_Metadata_Fields", func(t *testing.T) {
		docConfig := DocumentConfig{
			DocumentID:   "test-doc",
			DocumentName: "Test Document",
			DocumentType: "docs",
			Validation: GoogleValidationRules{
				DateRange: &DateRange{
					From: "2023-01-01",
					To:   "2023-12-31",
				},
			},
		}

		testCases := []struct {
			name     string
			metadata map[string]interface{}
		}{
			{"No_Metadata", nil},
			{"Empty_Metadata", map[string]interface{}{}},
			{"Wrong_Type", map[string]interface{}{"modified_at": "not-a-time"}},
			{"Nil_Value", map[string]interface{}{"modified_at": nil}},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				source := &models.EvidenceSource{
					Content:  "Test content",
					Metadata: tc.metadata,
				}

				// Should not crash with missing or invalid metadata
				validation := enhancedTool.validateDocumentContent(docConfig, source)
				assert.NotNil(t, validation)
				// Should still validate successfully (just skip date validation)
				assert.True(t, validation["passed"].(bool))
			})
		}
	})
}
