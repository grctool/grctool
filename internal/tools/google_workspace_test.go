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
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock Google API responses for testing
const (
	mockServiceAccountJSON = `{
		"type": "service_account",
		"project_id": "test-project",
		"private_key_id": "test-key-id",
		"private_key": "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC...\n-----END PRIVATE KEY-----\n",
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"client_id": "1234567890",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "https://oauth2.googleapis.com/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test%40test-project.iam.gserviceaccount.com"
	}`

	mockDriveFileResponse = `{
		"id": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
		"name": "Sample Access Review",
		"mimeType": "application/vnd.google-apps.spreadsheet",
		"owners": [{"displayName": "Test Owner", "emailAddress": "owner@test.com"}],
		"createdTime": "2023-01-01T00:00:00.000Z",
		"modifiedTime": "2023-06-01T12:00:00.000Z",
		"size": "12345"
	}`

	mockDocsResponse = `{
		"documentId": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
		"title": "Security Policy Document",
		"revisionId": "ALm37BVTiPMa9C0q4SQA3pOlUiB9sg",
		"body": {
			"content": [
				{
					"paragraph": {
						"elements": [
							{
								"textRun": {
									"content": "# Security Policy\n\nThis document outlines our security policies and procedures.\n\n## Access Control\n\nAll access to systems must be authorized and reviewed quarterly.\n\n## Data Protection\n\nAll sensitive data must be encrypted at rest and in transit.\n"
								}
							}
						]
					}
				}
			]
		}
	}`

	mockSheetsResponse = `{
		"spreadsheetId": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
		"properties": {
			"title": "Access Review Q2 2023"
		},
		"sheets": [
			{
				"properties": {
					"title": "User Access"
				}
			}
		]
	}`

	mockSheetsValuesResponse = `{
		"range": "User Access!A:Z",
		"majorDimension": "ROWS",
		"values": [
			["Employee", "System", "Access Level", "Review Date", "Reviewer", "Status"],
			["John Doe", "CRM", "Read", "2023-06-01", "Manager", "Approved"],
			["Jane Smith", "Database", "Admin", "2023-06-01", "IT Director", "Approved"],
			["Bob Johnson", "File Server", "Write", "2023-06-01", "Team Lead", "Pending"]
		]
	}`

	mockFormsResponse = `{
		"formId": "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms",
		"info": {
			"title": "Security Training Completion",
			"description": "Please complete this form to confirm your security training."
		},
		"items": [
			{
				"title": "Employee Name",
				"description": "Your full name",
				"questionItem": {
					"question": {
						"textQuestion": {}
					}
				}
			},
			{
				"title": "Training Module Completed",
				"description": "Which training module did you complete?",
				"questionItem": {
					"question": {
						"choiceQuestion": {
							"options": [
								{"value": "Security Awareness"},
								{"value": "Data Protection"},
								{"value": "Incident Response"}
							]
						}
					}
				}
			}
		]
	}`

	mockFormsResponsesResponse = `{
		"responses": [
			{
				"responseId": "resp1",
				"createTime": "2023-06-01T10:00:00Z",
				"lastSubmittedTime": "2023-06-01T10:00:00Z"
			},
			{
				"responseId": "resp2",
				"createTime": "2023-06-02T14:30:00Z",
				"lastSubmittedTime": "2023-06-02T14:30:00Z"
			}
		]
	}`

	mockFolderContentsResponse = `{
		"files": [
			{
				"id": "doc1",
				"name": "Security Policy 2023",
				"mimeType": "application/vnd.google-apps.document",
				"owners": [{"displayName": "Security Team"}],
				"createdTime": "2023-01-15T09:00:00Z",
				"modifiedTime": "2023-03-20T16:30:00Z",
				"size": "8765"
			},
			{
				"id": "sheet1",
				"name": "Access Review Q1",
				"mimeType": "application/vnd.google-apps.spreadsheet",
				"owners": [{"displayName": "HR Manager"}],
				"createdTime": "2023-04-01T08:00:00Z",
				"modifiedTime": "2023-04-15T17:45:00Z",
				"size": "15432"
			}
		]
	}`
)

func TestGoogleWorkspaceTool(t *testing.T) {
	// Create test directory and credentials file
	tempDir := t.TempDir()
	credentialsPath := filepath.Join(tempDir, "credentials.json")
	err := os.WriteFile(credentialsPath, []byte(mockServiceAccountJSON), 0644)
	require.NoError(t, err)

	// Create test config
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GoogleDocs: config.GoogleDocsToolConfig{
					Enabled:         true,
					CredentialsFile: credentialsPath,
				},
			},
		},
	}

	cfgLogger := &logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(t, err)

	tool := NewGoogleWorkspaceTool(cfg, log)

	t.Run("Tool Properties", func(t *testing.T) {
		assert.Equal(t, "google-workspace", tool.Name())
		assert.Contains(t, tool.Description(), "Google Workspace documents")

		definition := tool.GetClaudeToolDefinition()
		assert.Equal(t, "google-workspace", definition.Name)
		assert.NotNil(t, definition.InputSchema)

		// Check required parameters
		if required, ok := definition.InputSchema["required"].([]string); ok {
			assert.Contains(t, required, "document_id")
		}
	})

	t.Run("Parameter Validation", func(t *testing.T) {
		ctx := context.Background()

		// Test missing document_id
		params := map[string]interface{}{}
		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "document_id parameter is required")

		// Test empty document_id
		params = map[string]interface{}{
			"document_id": "",
		}
		_, _, err = tool.Execute(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "document_id parameter is required")
	})

	t.Run("Credentials Discovery", func(t *testing.T) {
		ctx := context.Background()

		// Test with explicit credentials path

		// Mock HTTP server would be needed for full test
		// For now, just test that credentials are loaded correctly
		gwt := tool.(*GoogleWorkspaceTool)

		// Test credentials file reading
		client, err := gwt.initializeGoogleClient(ctx, credentialsPath)
		require.NoError(t, err)
		assert.NotNil(t, client)
	})

	t.Run("Invalid Credentials Path", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"document_id":      "test-doc-id",
			"credentials_path": "/nonexistent/path.json",
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read credentials file")
	})

	t.Run("Unsupported Document Type", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"document_id":      "test-doc-id",
			"document_type":    "invalid-type",
			"credentials_path": credentialsPath,
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported document type")
	})
}

func TestGoogleWorkspaceToolWithMockServer(t *testing.T) {
	// Create mock Google API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.Contains(r.URL.Path, "/drive/v3/files/") && !strings.Contains(r.URL.Path, "/values"):
			if strings.Contains(r.URL.Path, "folder-id") {
				// Return folder metadata for folder requests
				folderResponse := strings.Replace(mockDriveFileResponse, "spreadsheet", "folder", 1)
				folderResponse = strings.Replace(folderResponse, "Sample Access Review", "Security Documents", 1)
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(folderResponse))
			} else {
				// Return file metadata
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(mockDriveFileResponse))
			}
		case strings.Contains(r.URL.Path, "/drive/v3/files"):
			// List files in folder
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockFolderContentsResponse))
		case strings.Contains(r.URL.Path, "/docs/v1/documents/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockDocsResponse))
		case strings.Contains(r.URL.Path, "/sheets/v4/spreadsheets/") && strings.Contains(r.URL.Path, "/values/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockSheetsValuesResponse))
		case strings.Contains(r.URL.Path, "/sheets/v4/spreadsheets/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockSheetsResponse))
		case strings.Contains(r.URL.Path, "/forms/v1/forms/") && strings.Contains(r.URL.Path, "/responses"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockFormsResponsesResponse))
		case strings.Contains(r.URL.Path, "/forms/v1/forms/"):
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(mockFormsResponse))
		case strings.Contains(r.URL.Path, "/token"):
			// OAuth token endpoint
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"access_token": "mock-access-token",
				"token_type": "Bearer",
				"expires_in": 3600
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Not found"))
		}
	}))
	defer server.Close()

	// Create test directory and credentials file
	tempDir := t.TempDir()
	credentialsPath := filepath.Join(tempDir, "credentials.json")

	// Modify the service account JSON to point to our mock server
	mockCredentials := strings.Replace(mockServiceAccountJSON,
		"https://oauth2.googleapis.com/token",
		server.URL+"/token", 1)

	err := os.WriteFile(credentialsPath, []byte(mockCredentials), 0644)
	require.NoError(t, err)

	// Create test config
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

	// Note: Due to the complexity of mocking the entire Google API client stack,
	// these tests focus on the core logic and structure. Full integration tests
	// would require actual Google API access or more sophisticated mocking.

	t.Run("Helper Functions", func(t *testing.T) {
		gwt := tool.(*GoogleWorkspaceTool)

		// Test MIME type categorization
		assert.Equal(t, "document", gwt.getMimeTypeCategory("application/vnd.google-apps.document"))
		assert.Equal(t, "spreadsheet", gwt.getMimeTypeCategory("application/vnd.google-apps.spreadsheet"))
		assert.Equal(t, "form", gwt.getMimeTypeCategory("application/vnd.google-apps.form"))
		assert.Equal(t, "folder", gwt.getMimeTypeCategory("application/vnd.google-apps.folder"))
		assert.Equal(t, "file", gwt.getMimeTypeCategory("application/pdf"))

		// Test time parsing
		parsed := gwt.parseGoogleTime("2023-06-01T12:00:00.000Z")
		assert.False(t, parsed.IsZero())
		assert.Equal(t, 2023, parsed.Year())
		assert.Equal(t, time.June, parsed.Month())

		// Test invalid time parsing
		parsed = gwt.parseGoogleTime("invalid-time")
		assert.True(t, parsed.IsZero())

		// Test empty time parsing
		parsed = gwt.parseGoogleTime("")
		assert.True(t, parsed.IsZero())
	})

	t.Run("Relevance Calculation", func(t *testing.T) {
		gwt := tool.(*GoogleWorkspaceTool)

		// Test base relevance
		result := &GoogleWorkspaceResult{
			DocumentType: "docs",
			Content:      "Short content",
			ModifiedAt:   time.Now().AddDate(0, 0, -5), // 5 days ago
		}
		relevance := gwt.calculateRelevance(result)
		assert.True(t, relevance >= 0.5 && relevance <= 1.0)

		// Test high content relevance
		result.Content = strings.Repeat("A", 6000) // Long content
		relevance = gwt.calculateRelevance(result)
		assert.True(t, relevance > 0.8)

		// Test forms relevance boost
		result.DocumentType = "forms"
		relevance = gwt.calculateRelevance(result)
		assert.True(t, relevance > 0.8)

		// Test folder contents boost
		result.FolderContents = []FolderItem{{}, {}, {}} // 3 items
		relevance = gwt.calculateRelevance(result)
		assert.True(t, relevance > 0.8)

		// Test old document penalty
		result.ModifiedAt = time.Now().AddDate(0, 0, -100) // 100 days ago
		relevance = gwt.calculateRelevance(result)
		// Should still be reasonable but not get the recent modification boost
		assert.True(t, relevance >= 0.5)
	})

	t.Run("Report Generation", func(t *testing.T) {
		gwt := tool.(*GoogleWorkspaceTool)

		result := &GoogleWorkspaceResult{
			DocumentID:   "test-doc-id",
			DocumentName: "Test Document",
			DocumentType: "docs",
			Owner:        "Test Owner",
			CreatedAt:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			ModifiedAt:   time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC),
			Content:      "This is test content for the document.",
			MimeType:     "application/vnd.google-apps.document",
		}

		rules := ExtractionRules{
			IncludeMetadata: true,
		}

		report := gwt.generateReport(result, "docs", rules)

		// Check report structure
		assert.Contains(t, report, "# Google Workspace Evidence Report")
		assert.Contains(t, report, "**Document Type**: docs")
		assert.Contains(t, report, "**Document Name**: Test Document")
		assert.Contains(t, report, "**Document ID**: test-doc-id")
		assert.Contains(t, report, "## Document Metadata")
		assert.Contains(t, report, "- **Owner**: Test Owner")
		assert.Contains(t, report, "## Document Content")
		assert.Contains(t, report, "This is test content for the document.")
	})

	t.Run("Sheet Data Report", func(t *testing.T) {
		gwt := tool.(*GoogleWorkspaceTool)

		result := &GoogleWorkspaceResult{
			DocumentID:   "test-sheet-id",
			DocumentName: "Test Spreadsheet",
			DocumentType: "sheets",
			SheetData: [][]interface{}{
				{"Header1", "Header2", "Header3"},
				{"Data1", "Data2", "Data3"},
				{"Data4", "Data5", "Data6"},
			},
		}

		rules := ExtractionRules{
			IncludeMetadata: false,
		}

		report := gwt.generateReport(result, "sheets", rules)

		// Check sheet-specific content
		assert.Contains(t, report, "## Sheet Data Summary")
		assert.Contains(t, report, "- **Total Rows**: 3")
		assert.Contains(t, report, "- **Columns**: 3")
	})

	t.Run("Folder Contents Report", func(t *testing.T) {
		gwt := tool.(*GoogleWorkspaceTool)

		result := &GoogleWorkspaceResult{
			DocumentID:   "test-folder-id",
			DocumentName: "Test Folder",
			DocumentType: "folder",
			FolderContents: []FolderItem{
				{
					ID:       "file1",
					Name:     "Document 1",
					Type:     "document",
					Owner:    "Owner 1",
					Modified: time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC),
					Size:     1024,
				},
				{
					ID:       "file2",
					Name:     "Spreadsheet 1",
					Type:     "spreadsheet",
					Owner:    "Owner 2",
					Modified: time.Date(2023, 5, 15, 9, 30, 0, 0, time.UTC),
					Size:     2048,
				},
			},
		}

		rules := ExtractionRules{
			IncludeMetadata: true,
		}

		report := gwt.generateReport(result, "drive", rules)

		// Check folder-specific content
		assert.Contains(t, report, "## Folder Contents")
		assert.Contains(t, report, "### Document 1")
		assert.Contains(t, report, "### Spreadsheet 1")
		assert.Contains(t, report, "- **Type**: document")
		assert.Contains(t, report, "- **Type**: spreadsheet")
		assert.Contains(t, report, "- **Size**: 1024 bytes")
		assert.Contains(t, report, "- **Size**: 2048 bytes")
	})
}

func TestGoogleWorkspaceToolWithMappings(t *testing.T) {
	// Create test directory
	tempDir := t.TempDir()
	credentialsPath := filepath.Join(tempDir, "credentials.json")
	err := os.WriteFile(credentialsPath, []byte(mockServiceAccountJSON), 0644)
	require.NoError(t, err)

	// Create test mappings file
	mappingsPath := filepath.Join(tempDir, "google_evidence_mappings.yaml")
	mappingsContent := `
google_workspace:
  default_extraction_rules:
    include_metadata: true
    include_revisions: false
    max_results: 20
  auth:
    credentials_path: "` + credentialsPath + `"

evidence_mappings:
  ET-101:
    task_ref: "ET-101"
    description: "Quarterly access review documentation"
    source_type: "google_workspace"
    priority: "high"
    documents:
      - document_id: "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms"
        document_name: "Q2 2023 Access Review"
        document_type: "sheets"
        extraction_rules:
          include_metadata: true
          sheet_range: "A1:F100"
        validation:
          min_rows: 10
          required_headers: ["Employee", "System", "Access Level"]
          min_content_length: 500

metadata:
  version: "1.0"
  created_date: "2023-01-01"
  updated_date: "2023-06-01"
  created_by: "Security Team"
  refresh_schedule:
    ET-101: "monthly"

cache_settings:
  enable_content_cache: true
  cache_duration: "24h"
  rate_limits:
    requests_per_minute: 60
    concurrent_requests: 5
`
	err = os.WriteFile(mappingsPath, []byte(mappingsContent), 0644)
	require.NoError(t, err)

	// Create test config
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

	t.Run("Mappings Loader", func(t *testing.T) {
		loader := NewGoogleEvidenceMappingsLoader(cfg, log)

		// Test loading mappings
		mappings, err := loader.LoadMappings()
		require.NoError(t, err)
		assert.NotNil(t, mappings)
		assert.Equal(t, "1.0", mappings.Metadata.Version)
		assert.Len(t, mappings.EvidenceMappings, 1)

		// Test getting specific mapping
		mapping, err := loader.GetMappingForTask("ET-101")
		require.NoError(t, err)
		assert.Equal(t, "ET-101", mapping.TaskRef)
		assert.Equal(t, "high", mapping.Priority)
		assert.Len(t, mapping.Documents, 1)

		// Test mapping not found
		_, err = loader.GetMappingForTask("NONEXISTENT")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no mapping found")

		// Test supported task refs
		taskRefs, err := loader.GetSupportedTaskRefs()
		require.NoError(t, err)
		assert.Contains(t, taskRefs, "ET-101")

		// Test document validation
		err = loader.ValidateDocumentAccess(mapping)
		assert.NoError(t, err)

		// Test parameter transformation
		params, err := loader.TransformToGoogleAPIParams(mapping, 0)
		require.NoError(t, err)
		assert.Equal(t, "1BxiMVs0XRA5nFMdKvBdBZjgmUUqptlbs74OgvE2upms", params["document_id"])
		assert.Equal(t, "sheets", params["document_type"])
		assert.NotNil(t, params["extraction_rules"])

		extractionRules := params["extraction_rules"].(map[string]interface{})
		assert.Equal(t, true, extractionRules["include_metadata"])
		assert.Equal(t, "A1:F100", extractionRules["sheet_range"])
	})

	t.Run("Enhanced Tool with Mappings", func(t *testing.T) {
		enhancedTool := NewGoogleWorkspaceToolWithMappings(cfg, log)

		// Test getting supported evidence tasks
		tasks, err := enhancedTool.GetSupportedEvidenceTasks()
		require.NoError(t, err)
		assert.Contains(t, tasks, "ET-101")

		// Test getting task mapping info
		mapping, err := enhancedTool.GetTaskMappingInfo("ET-101")
		require.NoError(t, err)
		assert.Equal(t, "ET-101", mapping.TaskRef)

		// Test refresh mappings
		enhancedTool.RefreshMappings() // Should not error
	})

	t.Run("Document Content Validation", func(t *testing.T) {
		enhancedTool := NewGoogleWorkspaceToolWithMappings(cfg, log)

		// Create test document config
		docConfig := DocumentConfig{
			DocumentID:   "test-doc",
			DocumentName: "Test Document",
			DocumentType: "sheets",
			Validation: GoogleValidationRules{
				MinContentLength: 100,
				RequiredKeywords: []string{"access", "review"},
				MinRows:          5,
			},
		}

		// Create test evidence source with valid content (longer than 100 characters)
		source := &models.EvidenceSource{
			Content: "This document contains access review information with all required keywords. This is a comprehensive document that provides detailed access control review procedures and guidelines for security compliance. The content meets all minimum length requirements.",
			Metadata: map[string]interface{}{
				"modified_at": time.Now(),
			},
		}

		validation := enhancedTool.validateDocumentContent(docConfig, source)
		assert.True(t, validation["passed"].(bool))
		assert.Empty(t, validation["errors"].([]string))

		// Test with invalid content (too short)
		source.Content = "Short"
		validation = enhancedTool.validateDocumentContent(docConfig, source)
		assert.False(t, validation["passed"].(bool))
		errors := validation["errors"].([]string)
		assert.Len(t, errors, 1)
		assert.Contains(t, errors[0], "Content length")

		// Test with missing keywords
		source.Content = strings.Repeat("A", 200) // Long enough but missing keywords
		validation = enhancedTool.validateDocumentContent(docConfig, source)
		assert.True(t, validation["passed"].(bool)) // Passes because missing keywords are warnings, not errors
		warnings := validation["warnings"].([]string)
		assert.Len(t, warnings, 2) // Two missing keywords
	})
}

func TestExtractionRulesDefaults(t *testing.T) {
	tempDir := t.TempDir()

	// Create minimal mappings file to test defaults
	mappingsPath := filepath.Join(tempDir, "google_evidence_mappings.yaml")
	mappingsContent := `
google_workspace:
  default_extraction_rules:
    include_metadata: true
    max_results: 50

evidence_mappings:
  ET-TEST:
    task_ref: "ET-TEST"
    description: "Test mapping"
    source_type: "google_workspace"
    priority: "medium"
    documents:
      - document_id: "test-doc"
        document_name: "Test Document"
        document_type: "docs"
        # No extraction_rules specified - should inherit defaults

metadata:
  version: "1.0"
`
	err := os.WriteFile(mappingsPath, []byte(mappingsContent), 0644)
	require.NoError(t, err)

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

	loader := NewGoogleEvidenceMappingsLoader(cfg, log)

	mappings, err := loader.LoadMappings()
	require.NoError(t, err)

	mapping := mappings.EvidenceMappings["ET-TEST"]
	doc := mapping.Documents[0]

	// Check that defaults were applied
	assert.True(t, doc.ExtractionRules.IncludeMetadata)
	assert.Equal(t, 50, doc.ExtractionRules.MaxResults)
}
