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
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoogleWorkspaceComprehensiveCoverage tests all the major functions that currently have 0% coverage
func TestGoogleWorkspaceComprehensiveCoverage(t *testing.T) {
	// Setup test server that mocks Google APIs
	server := setupMockGoogleServer(t)
	defer server.Close()

	// Create test environment
	tempDir := t.TempDir()
	credentialsPath := createTestCredentials(t, tempDir, server.URL)

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

	t.Run("ExtractFromDrive", func(t *testing.T) {
		ctx := context.Background()
		client, err := gwt.initializeGoogleClient(ctx, credentialsPath)
		require.NoError(t, err)

		// Test extracting from a regular file
		rules := ExtractionRules{
			IncludeMetadata: true,
			MaxResults:      10,
		}

		result, err := gwt.extractFromDrive(ctx, client, "test-doc-id", rules)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-doc-id", result.DocumentID)
		assert.Equal(t, "Sample Document", result.DocumentName)
		assert.Equal(t, "document", result.DocumentType)

		// Test extracting from a folder
		result, err = gwt.extractFromDrive(ctx, client, "test-folder-id", rules)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-folder-id", result.DocumentID)
		assert.Contains(t, result.Content, "Folder:")
		assert.Len(t, result.FolderContents, 2) // Based on mock response
	})

	t.Run("ExtractFromDocs", func(t *testing.T) {
		ctx := context.Background()
		client, err := gwt.initializeGoogleClient(ctx, credentialsPath)
		require.NoError(t, err)

		rules := ExtractionRules{
			IncludeMetadata:  true,
			IncludeRevisions: false,
		}

		result, err := gwt.extractFromDocs(ctx, client, "test-doc-id", rules)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-doc-id", result.DocumentID)
		assert.Equal(t, "Security Policy Document", result.DocumentName)
		assert.Contains(t, result.Content, "Security Policy")
		assert.Contains(t, result.Content, "Access Control")

		// Test with metadata
		assert.NotNil(t, result.Metadata)
		assert.Contains(t, result.Metadata, "revision_id")
		assert.Contains(t, result.Metadata, "title")
	})

	t.Run("ExtractFromSheets", func(t *testing.T) {
		ctx := context.Background()
		client, err := gwt.initializeGoogleClient(ctx, credentialsPath)
		require.NoError(t, err)

		// Test with default range
		rules := ExtractionRules{
			IncludeMetadata: true,
		}

		result, err := gwt.extractFromSheets(ctx, client, "test-sheet-id", rules)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-sheet-id", result.DocumentID)
		assert.Equal(t, "Access Review Q2 2023", result.DocumentName)
		assert.Contains(t, result.Content, "Employee")
		assert.Contains(t, result.Content, "John Doe")
		assert.Len(t, result.SheetData, 4) // Header + 3 data rows

		// Test with specific range
		rules.SheetRange = "A1:C10"
		result, err = gwt.extractFromSheets(ctx, client, "test-sheet-id", rules)
		require.NoError(t, err)
		assert.Contains(t, result.Content, "Range: A1:C10")

		// Test with metadata
		assert.NotNil(t, result.Metadata)
		assert.Contains(t, result.Metadata, "sheet_count")
		assert.Contains(t, result.Metadata, "row_count")
	})

	t.Run("ExtractFromForms", func(t *testing.T) {
		ctx := context.Background()
		client, err := gwt.initializeGoogleClient(ctx, credentialsPath)
		require.NoError(t, err)

		rules := ExtractionRules{
			IncludeMetadata: true,
		}

		result, err := gwt.extractFromForms(ctx, client, "test-form-id", rules)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "test-form-id", result.DocumentID)
		assert.Equal(t, "Security Training Completion", result.DocumentName)
		assert.Contains(t, result.Content, "Employee Name")
		assert.Contains(t, result.Content, "Training Module Completed")
		assert.Contains(t, result.Content, "Responses: 2 total")

		// Test metadata
		assert.NotNil(t, result.Metadata)
		assert.Contains(t, result.Metadata, "question_count")
		assert.Contains(t, result.Metadata, "response_count")
	})

	t.Run("ExtractFolderContents", func(t *testing.T) {
		// This test is complex to implement without full Google API mocking
		// For now, we'll test the core functionality through the main Execute method
		ctx := context.Background()
		params := map[string]interface{}{
			"document_id":      "test-folder-id",
			"document_type":    "drive",
			"credentials_path": credentialsPath,
		}

		report, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, report)
		assert.NotNil(t, source)

		// Check that it's recognized as a folder
		assert.Contains(t, report, "Folder:")
	})

	t.Run("ExtractFileContent", func(t *testing.T) {
		// Test through the main Execute method with different document types
		ctx := context.Background()

		testCases := []struct {
			name     string
			docType  string
			expected string
		}{
			{"Document", "docs", "Security Policy Document"},
			{"Spreadsheet", "sheets", "Access Review Q2 2023"},
			{"Form", "forms", "Security Training Completion"},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				params := map[string]interface{}{
					"document_id":      "test-doc-id",
					"document_type":    tc.docType,
					"credentials_path": credentialsPath,
				}

				report, source, err := tool.Execute(ctx, params)
				require.NoError(t, err)
				assert.NotNil(t, source)
				assert.NotEmpty(t, report)

				// Check that content was extracted
				assert.NotEmpty(t, source.Content)
			})
		}
	})

	t.Run("GetQuestionType", func(t *testing.T) {
		// Test question type detection through forms processing
		// This is tested indirectly through the forms extraction which processes questions
		ctx := context.Background()
		params := map[string]interface{}{
			"document_id":      "test-form-id",
			"document_type":    "forms",
			"credentials_path": credentialsPath,
		}

		report, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotNil(t, source)
		assert.NotEmpty(t, report)

		// The forms response contains different question types which are processed
		assert.Contains(t, source.Content, "Employee Name")
		assert.Contains(t, source.Content, "Training Module Completed")
	})
}

// TestGoogleWorkspaceExecutionPaths tests the main Execute method paths that have low coverage
func TestGoogleWorkspaceExecutionPaths(t *testing.T) {
	server := setupMockGoogleServer(t)
	defer server.Close()

	tempDir := t.TempDir()
	credentialsPath := createTestCredentials(t, tempDir, server.URL)

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

	tool := NewGoogleWorkspaceTool(cfg, log)

	t.Run("Execute_AllDocumentTypes", func(t *testing.T) {
		ctx := context.Background()

		testCases := []struct {
			docType  string
			docID    string
			expected string
		}{
			{"drive", "test-doc-id", "Sample Document"},
			{"docs", "test-doc-id", "Security Policy Document"},
			{"sheets", "test-sheet-id", "Access Review Q2 2023"},
			{"forms", "test-form-id", "Security Training Completion"},
		}

		for _, tc := range testCases {
			t.Run(tc.docType, func(t *testing.T) {
				params := map[string]interface{}{
					"document_id":      tc.docID,
					"document_type":    tc.docType,
					"credentials_path": credentialsPath,
					"extraction_rules": map[string]interface{}{
						"include_metadata": true,
						"max_results":      10,
					},
				}

				report, source, err := tool.Execute(ctx, params)
				require.NoError(t, err)
				assert.NotEmpty(t, report)
				assert.NotNil(t, source)
				assert.Contains(t, source.Resource, tc.expected)
				assert.Equal(t, "google-workspace", source.Type)
				assert.True(t, source.Relevance > 0)
			})
		}
	})

	t.Run("Execute_ExtractionRules", func(t *testing.T) {
		ctx := context.Background()

		// Test with various extraction rule combinations
		params := map[string]interface{}{
			"document_id":      "test-sheet-id",
			"document_type":    "sheets",
			"credentials_path": credentialsPath,
			"extraction_rules": map[string]interface{}{
				"include_metadata":  true,
				"include_revisions": true,
				"sheet_range":       "A1:D10",
				"search_query":      "test query",
				"max_results":       50,
			},
		}

		report, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, report)
		assert.NotNil(t, source)

		// Check that extraction rules were applied
		assert.Contains(t, report, "Range: A1:D10")
	})

	t.Run("Execute_CredentialsDiscovery", func(t *testing.T) {
		ctx := context.Background()

		// Test without explicit credentials path - should discover from common locations
		// First, let's place credentials in a discoverable location
		homeCredPath := filepath.Join(tempDir, "application_default_credentials.json")
		err := os.WriteFile(homeCredPath, []byte(createMockCredentialsJSON(server.URL)), 0644)
		require.NoError(t, err)

		// Set environment variable to point to our test credentials
		oldEnv := os.Getenv("GOOGLE_APPLICATION_CREDENTIALS")
		os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", homeCredPath)
		defer os.Setenv("GOOGLE_APPLICATION_CREDENTIALS", oldEnv)

		params := map[string]interface{}{
			"document_id":   "test-doc-id",
			"document_type": "docs",
		}

		report, source, err := tool.Execute(ctx, params)
		require.NoError(t, err)
		assert.NotEmpty(t, report)
		assert.NotNil(t, source)
	})
}

// TestGoogleWorkspaceErrorHandling tests various error scenarios
func TestGoogleWorkspaceErrorHandling(t *testing.T) {
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

	tool := NewGoogleWorkspaceTool(cfg, log)

	t.Run("Authentication_Errors", func(t *testing.T) {
		ctx := context.Background()

		// Invalid credentials file content
		invalidCredPath := filepath.Join(tempDir, "invalid_creds.json")
		err := os.WriteFile(invalidCredPath, []byte(`{"invalid": "json structure"}`), 0644)
		require.NoError(t, err)

		params := map[string]interface{}{
			"document_id":      "test-doc-id",
			"credentials_path": invalidCredPath,
		}

		_, _, err = tool.Execute(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create JWT config")

		// Malformed JSON credentials file
		malformedCredPath := filepath.Join(tempDir, "malformed_creds.json")
		err = os.WriteFile(malformedCredPath, []byte(`{invalid json`), 0644)
		require.NoError(t, err)

		params["credentials_path"] = malformedCredPath
		_, _, err = tool.Execute(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create JWT config")
	})

	t.Run("Network_Errors", func(t *testing.T) {
		// Create server that simulates various network conditions
		errorServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/token":
				// Always return valid token for auth
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`))
			case "/timeout":
				// Simulate timeout
				time.Sleep(10 * time.Second)
				w.WriteHeader(http.StatusOK)
			case "/rate-limit":
				// Simulate rate limiting
				w.WriteHeader(http.StatusTooManyRequests)
				w.Write([]byte(`{"error": "Rate limit exceeded"}`))
			case "/server-error":
				// Simulate server error
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": "Internal server error"}`))
			default:
				// For drive API calls, return 404
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": "File not found"}`))
			}
		}))
		defer errorServer.Close()

		credPath := createTestCredentials(t, tempDir, errorServer.URL)
		ctx := context.Background()

		params := map[string]interface{}{
			"document_id":      "nonexistent-doc-id",
			"credentials_path": credPath,
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to extract from")
	})

	t.Run("Document_Access_Errors", func(t *testing.T) {
		// Test document not found, permission denied, etc.
		permissionServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if strings.Contains(r.URL.Path, "/token") {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"access_token": "test-token", "token_type": "Bearer", "expires_in": 3600}`))
				return
			}

			// Simulate permission denied
			w.WriteHeader(http.StatusForbidden)
			w.Write([]byte(`{"error": "Permission denied"}`))
		}))
		defer permissionServer.Close()

		credPath := createTestCredentials(t, tempDir, permissionServer.URL)
		ctx := context.Background()

		params := map[string]interface{}{
			"document_id":      "restricted-doc-id",
			"credentials_path": credPath,
		}

		_, _, err := tool.Execute(ctx, params)
		assert.Error(t, err)
	})
}

// TestGoogleWorkspaceWithMappingsIntegration tests the evidence mapping integration
func TestGoogleWorkspaceWithMappingsIntegration(t *testing.T) {
	server := setupMockGoogleServer(t)
	defer server.Close()

	tempDir := t.TempDir()
	credentialsPath := createTestCredentials(t, tempDir, server.URL)

	// Create comprehensive test mappings
	mappingsPath := filepath.Join(tempDir, "google_evidence_mappings.yaml")
	mappingsContent := fmt.Sprintf(`
google_workspace:
  default_extraction_rules:
    include_metadata: true
    include_revisions: false
    max_results: 20
  auth:
    credentials_path: "%s"

evidence_mappings:
  ET-101:
    task_ref: "ET-101"
    description: "Access Review Documentation"
    source_type: "google_workspace"
    priority: "high"
    documents:
      - document_id: "test-sheet-id"
        document_name: "Access Review Q2 2023"
        document_type: "sheets"
        extraction_rules:
          include_metadata: true
          sheet_range: "A1:F100"
        validation:
          min_rows: 3
          required_headers: ["Employee", "System", "Access Level"]
          min_content_length: 100
          required_keywords: ["access", "review"]
      - document_id: "test-doc-id"
        document_name: "Security Policy"
        document_type: "docs"
        extraction_rules:
          include_metadata: true
        validation:
          min_content_length: 200
          required_keywords: ["security", "policy"]

  ET-102:
    task_ref: "ET-102"
    description: "Training Documentation"
    source_type: "google_workspace"
    priority: "medium"
    documents:
      - document_id: "test-form-id"
        document_name: "Training Completion Form"
        document_type: "forms"
        validation:
          min_responses: 1

metadata:
  version: "1.0"
  created_date: "2023-01-01"
  updated_date: "2023-06-01"
  created_by: "Security Team"
  refresh_schedule:
    ET-101: "monthly"
    ET-102: "quarterly"

cache_settings:
  enable_content_cache: true
  cache_duration: "24h"
  rate_limits:
    requests_per_minute: 60
    concurrent_requests: 5
`, credentialsPath)

	err := os.WriteFile(mappingsPath, []byte(mappingsContent), 0644)
	require.NoError(t, err)

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

	t.Run("ExecuteForEvidenceTask", func(t *testing.T) {
		enhancedTool := NewGoogleWorkspaceToolWithMappings(cfg, log)
		ctx := context.Background()

		// Test successful execution
		reports, sources, err := enhancedTool.ExecuteForEvidenceTask(ctx, "ET-101")
		require.NoError(t, err)
		assert.Len(t, reports, 2) // Two documents configured
		assert.Len(t, sources, 2)

		// Check that metadata was enhanced
		for _, source := range sources {
			assert.Equal(t, "ET-101", source.Metadata["task_ref"])
			assert.Equal(t, "high", source.Metadata["mapping_priority"])
			assert.Contains(t, source.Metadata, "validation_status")
		}

		// Test task not found
		_, _, err = enhancedTool.ExecuteForEvidenceTask(ctx, "NONEXISTENT")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no mapping found")
	})

	t.Run("DocumentContentValidation", func(t *testing.T) {
		enhancedTool := NewGoogleWorkspaceToolWithMappings(cfg, log)

		// Test validation with content that meets all requirements
		docConfig := DocumentConfig{
			DocumentID:   "test-doc",
			DocumentName: "Test Document",
			DocumentType: "docs",
			Validation: GoogleValidationRules{
				MinContentLength: 50,
				RequiredKeywords: []string{"security", "access"},
				DateRange: &DateRange{
					From: "2023-01-01",
					To:   "2023-12-31",
				},
			},
		}

		source := &models.EvidenceSource{
			Content: "This document contains security policies and access control procedures that are essential for compliance.",
			Metadata: map[string]interface{}{
				"modified_at": time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
			},
		}

		validation := enhancedTool.validateDocumentContent(docConfig, source)
		assert.True(t, validation["passed"].(bool))
		assert.Empty(t, validation["errors"].([]string))
		warnings := validation["warnings"].([]string)
		assert.Empty(t, warnings) // All keywords present

		// Test with failing validation
		source.Content = "Short" // Too short and missing keywords
		validation = enhancedTool.validateDocumentContent(docConfig, source)
		assert.False(t, validation["passed"].(bool))
		errors := validation["errors"].([]string)
		assert.Len(t, errors, 1)
		assert.Contains(t, errors[0], "Content length")

		warnings = validation["warnings"].([]string)
		assert.Len(t, warnings, 2) // Two missing keywords

		// Test date range validation
		source.Content = "This document contains security policies and access control procedures."
		source.Metadata["modified_at"] = time.Date(2022, 12, 31, 10, 0, 0, 0, time.UTC) // Before range
		validation = enhancedTool.validateDocumentContent(docConfig, source)
		warnings = validation["warnings"].([]string)
		found := false
		for _, warning := range warnings {
			if strings.Contains(warning, "before required range start") {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected date range warning not found")
	})
}

// TestGoogleWorkspacePerformance tests performance characteristics
func TestGoogleWorkspacePerformance(t *testing.T) {
	server := setupMockGoogleServer(t)
	defer server.Close()

	tempDir := t.TempDir()
	credentialsPath := createTestCredentials(t, tempDir, server.URL)

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

	tool := NewGoogleWorkspaceTool(cfg, log)

	t.Run("ConcurrentExecution", func(t *testing.T) {
		ctx := context.Background()

		var wg sync.WaitGroup
		errors := make(chan error, 10)

		// Run multiple concurrent executions
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				params := map[string]interface{}{
					"document_id":      fmt.Sprintf("test-doc-id-%d", id),
					"document_type":    "docs",
					"credentials_path": credentialsPath,
				}

				_, _, err := tool.Execute(ctx, params)
				if err != nil {
					errors <- err
				}
			}(i)
		}

		wg.Wait()
		close(errors)

		// Check for any errors
		var errs []error
		for err := range errors {
			errs = append(errs, err)
		}

		if len(errs) > 0 {
			t.Errorf("Found %d errors in concurrent execution: %v", len(errs), errs[0])
		}
	})

	t.Run("LargeDocumentHandling", func(t *testing.T) {
		// This would test how the tool handles large documents
		// For now, we'll test with the existing mock data
		ctx := context.Background()

		params := map[string]interface{}{
			"document_id":      "test-sheet-id",
			"document_type":    "sheets",
			"credentials_path": credentialsPath,
			"extraction_rules": map[string]interface{}{
				"max_results": 100, // Request more results
			},
		}

		start := time.Now()
		_, _, err := tool.Execute(ctx, params)
		duration := time.Since(start)

		require.NoError(t, err)
		assert.Less(t, duration, 10*time.Second, "Execution took too long")
	})
}

// TestGoogleWorkspaceMappingsFilePath tests the getMappingsFilePath function with different scenarios
func TestGoogleWorkspaceMappingsFilePath(t *testing.T) {
	tempDir := t.TempDir()

	cfgLogger := &logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(t, err)

	t.Run("DataDirPath", func(t *testing.T) {
		// Create file in data directory
		dataPath := filepath.Join(tempDir, "google_evidence_mappings.yaml")
		err := os.WriteFile(dataPath, []byte("test"), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			Storage: config.StorageConfig{
				DataDir: tempDir,
			},
		}

		loader := NewGoogleEvidenceMappingsLoader(cfg, log)
		path := loader.getMappingsFilePath()
		assert.Equal(t, dataPath, path)
	})

	t.Run("CurrentDirPath", func(t *testing.T) {
		// Change to temp directory
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(oldWd)

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Create file in current directory
		currentPath := "google_evidence_mappings.yaml"
		err = os.WriteFile(currentPath, []byte("test"), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			Storage: config.StorageConfig{
				DataDir: "/nonexistent", // Should fall back to current dir
			},
		}

		loader := NewGoogleEvidenceMappingsLoader(cfg, log)
		path := loader.getMappingsFilePath()
		assert.Contains(t, path, "google_evidence_mappings.yaml")
	})

	t.Run("ConfigsDirPath", func(t *testing.T) {
		// Create configs directory and file
		configsDir := filepath.Join(tempDir, "configs")
		err := os.MkdirAll(configsDir, 0755)
		require.NoError(t, err)

		configPath := filepath.Join(configsDir, "google_evidence_mappings.yaml")
		err = os.WriteFile(configPath, []byte("test"), 0644)
		require.NoError(t, err)

		// Change to temp directory
		oldWd, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(oldWd)

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		cfg := &config.Config{
			Storage: config.StorageConfig{
				DataDir: "/nonexistent",
			},
		}

		loader := NewGoogleEvidenceMappingsLoader(cfg, log)
		path := loader.getMappingsFilePath()
		assert.Equal(t, configPath, path)
	})

	t.Run("DefaultPath", func(t *testing.T) {
		// No files exist - should return default
		cfg := &config.Config{
			Storage: config.StorageConfig{
				DataDir: "/nonexistent",
			},
		}

		loader := NewGoogleEvidenceMappingsLoader(cfg, log)
		path := loader.getMappingsFilePath()
		assert.Equal(t, "google_evidence_mappings.yaml", path)
	})
}

// Helper functions for mocking

func setupMockGoogleServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case strings.Contains(r.URL.Path, "/token"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"access_token": "mock-access-token",
				"token_type": "Bearer",
				"expires_in": 3600
			}`))

		case strings.Contains(r.URL.Path, "/drive/v3/files/test-folder-id") && !strings.Contains(r.URL.Path, "/values"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "test-folder-id",
				"name": "Security Documents Folder",
				"mimeType": "application/vnd.google-apps.folder",
				"owners": [{"displayName": "Security Team"}],
				"createdTime": "2023-01-01T00:00:00.000Z",
				"modifiedTime": "2023-06-01T12:00:00.000Z"
			}`))

		case strings.Contains(r.URL.Path, "/drive/v3/files/") && !strings.Contains(r.URL.Path, "/values"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "test-doc-id",
				"name": "Sample Document",
				"mimeType": "application/vnd.google-apps.document",
				"owners": [{"displayName": "Test Owner"}],
				"createdTime": "2023-01-01T00:00:00.000Z",
				"modifiedTime": "2023-06-01T12:00:00.000Z",
				"size": "12345"
			}`))

		case strings.Contains(r.URL.Path, "/drive/v3/files") && strings.Contains(r.URL.Query().Get("q"), "parents"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
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
			}`))

		case strings.Contains(r.URL.Path, "/docs/v1/documents/"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"documentId": "test-doc-id",
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
			}`))

		case strings.Contains(r.URL.Path, "/sheets/v4/spreadsheets/") && strings.Contains(r.URL.Path, "/values/"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"range": "User Access!A:Z",
				"majorDimension": "ROWS",
				"values": [
					["Employee", "System", "Access Level", "Review Date", "Reviewer", "Status"],
					["John Doe", "CRM", "Read", "2023-06-01", "Manager", "Approved"],
					["Jane Smith", "Database", "Admin", "2023-06-01", "IT Director", "Approved"],
					["Bob Johnson", "File Server", "Write", "2023-06-01", "Team Lead", "Pending"]
				]
			}`))

		case strings.Contains(r.URL.Path, "/sheets/v4/spreadsheets/"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"spreadsheetId": "test-sheet-id",
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
			}`))

		case strings.Contains(r.URL.Path, "/forms/v1/forms/") && strings.Contains(r.URL.Path, "/responses"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
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
			}`))

		case strings.Contains(r.URL.Path, "/forms/v1/forms/"):
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"formId": "test-form-id",
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
			}`))

		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "Not found"}`))
		}
	}))
}

func createTestCredentials(t *testing.T, tempDir, serverURL string) string {
	credentialsPath := filepath.Join(tempDir, "credentials.json")
	credentialsJSON := createMockCredentialsJSON(serverURL)
	err := os.WriteFile(credentialsPath, []byte(credentialsJSON), 0644)
	require.NoError(t, err)
	return credentialsPath
}

func createMockCredentialsJSON(serverURL string) string {
	// This is a test private key generated specifically for testing
	// It's not used in any production environment
	testPrivateKey := `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQC3UqCis2vKkW1r
so4pRAKOgOGm8ugcxLZkfIJw3t+1z78a3h7k+1ihbRuRU232Rf6RiCxozBCUPaw3
7SwT0dKIUgyR52pnSYHPQ3Gvx02evsLoC18kVBcpQqUoKVHjoV0xpjDQoxqne61D
t1CVqINUnV7mK9PNVN3ZelrZA5n7D/3SCUyicY2KCsYWliQcEHo9dFKktkFHlAvT
APtDEYXXtJbiQhKdMrIwLYbD/b9kQgUCiZve4dMhXrwq2XAovq2xHXmp+d1FrFFD
EU3VulSRU+MCUr0/9NcdtsYwdn2APJwWpSsCts8HCBMiAqJHuyj3X4YsOKz8TG/k
tFETWDhzAgMBAAECggEADkaVcsYVFUu44SOg9W6x4kYADH+q+p6I849MmxyIAEEC
yUVV05ANKVj1Rh5gmEaAGfYoOyr0+Y5J7HsALTTwN6RoDS6ftxZe0PSYFE+paDzD
sc085ffUa+agNN3u3hKRTs35zC1/ZF55siXC1TyvqXWtz6/HCRzcP9TK4U5p/cZx
taGLam17AXTv5B9rjl3ENn6HqeY0gynIx+2eobUwiCVXdnAxp+3H0Egp8kx+VTX9
D5+TEI2JDFhuH6/bc5bT0O8/PQCZUaKFZwUn5ybxLkX95SvseeZRemyF3ARYGvnO
sAbCFLAcYs10bOkuVXkdfLZaRYyNTsH7cM5XHkhygQKBgQDnfeRSXYac0PChcXl7
W6wteU7tjK/qxDPJZmwkqBujwT2T/oYA4QTvTfqsL6/hHeqw7ngZ5ThLpZI9ckN5
lw/yZ6e2fUbqExvrO9ph7AMqrpQoP0W9Y+wWenoIlaVYhSG4aqFPCwNQ5wHpo8rB
HgGfqZjlNP0ZLG5XIp3MGvEpgQKBgQDKuzbB8GaKv2ANEOTLkw33uqdBaEo8dffM
9cDgkjDvvW8uPYQiYfDtl9XK4W5EHiVA4s2HfJRBcFaELsrtSrcDVL5DLLUKaR91
LMhefeaZj1JLHnRTNkKevcXRd34yLrNfYQBEbpMcEDsr6FR1m9HqufUhIjroSzNW
JD1/snlT8wKBgB5SFv3S0jboBxyeSFMoBr1ODlB/BOuzFzVh/PgwLK6eOPqRc+vZ
jVPq2tKCzH6n9H2IPqLlqyH9ZdI2jS/34VbWzNjSP9+Y8Sc2h7wbta55f15mKzRL
SjkHgcRuFWIqzefhz48S2jRWjaGUmpIA5CWNiUE8V4pcj3dKSXDadowBAoGAGR1+
WCJnIbM5vASmw42RQmpuRA0efUUEEPE1Ft0lkN3AA1N9piDKDzUrODobRfcSGGrA
mZNWbpDzNubxHtqNt6zs8Td9qi+BxStqG0KvqcB2qnW4ZYKoWDAcbKnICYF9mUhU
FyY3tVdRbUwYAoXuSI0HEDbEY3jFgFt2/vXmT/8CgYEAyWXi3SvP0mzupmkLt1Yt
hW82nP7bgo65Qn9Q+hjNZEUpEC1z5qGtkQBzk7s0j0R+Oufsbd18azFUm7xHHXgr
C4l3nHNC8rLyKOVujyrsexLaUz6SdwNdAeK9uY/7IHRKjsLx9+yG5h4fq62NpzJf
L4jm86FlKXqGiy8GQ2/9Oko=
-----END PRIVATE KEY-----`

	// Escape the newlines for JSON
	escapedKey := strings.ReplaceAll(testPrivateKey, "\n", "\\n")

	return fmt.Sprintf(`{
		"type": "service_account",
		"project_id": "test-project",
		"private_key_id": "test-key-id",
		"private_key": "%s",
		"client_email": "test@test-project.iam.gserviceaccount.com",
		"client_id": "1234567890",
		"auth_uri": "https://accounts.google.com/o/oauth2/auth",
		"token_uri": "%s/token",
		"auth_provider_x509_cert_url": "https://www.googleapis.com/oauth2/v1/certs",
		"client_x509_cert_url": "https://www.googleapis.com/robot/v1/metadata/x509/test%%40test-project.iam.gserviceaccount.com"
	}`, escapedKey, serverURL)
}
