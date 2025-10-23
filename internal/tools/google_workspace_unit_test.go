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

// TestGoogleWorkspaceUnitFunctions tests individual functions that can be tested without API integration
func TestGoogleWorkspaceUnitFunctions(t *testing.T) {
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
	gwt := tool.(*GoogleWorkspaceTool)

	t.Run("Generate_Report_With_All_Fields", func(t *testing.T) {
		result := &GoogleWorkspaceResult{
			DocumentID:   "test-doc-123",
			DocumentName: "Complete Test Document",
			DocumentType: "sheets",
			Owner:        "Test Owner",
			Editors:      []string{"Editor1", "Editor2", "Editor3"},
			CreatedAt:    time.Date(2023, 1, 15, 9, 30, 0, 0, time.UTC),
			ModifiedAt:   time.Date(2023, 6, 20, 14, 45, 30, 0, time.UTC),
			MimeType:     "application/vnd.google-apps.spreadsheet",
			Content:      "This is comprehensive test content with detailed information about the document structure and data.",
			SheetData: [][]interface{}{
				{"Name", "Email", "Department", "Role"},
				{"John Doe", "john@example.com", "Engineering", "Developer"},
				{"Jane Smith", "jane@example.com", "Marketing", "Manager"},
			},
			FolderContents: []FolderItem{
				{
					ID:       "file1",
					Name:     "Policy Document",
					Type:     "document",
					MimeType: "application/vnd.google-apps.document",
					Owner:    "Policy Team",
					Size:     2048,
					Created:  time.Date(2023, 2, 1, 10, 0, 0, 0, time.UTC),
					Modified: time.Date(2023, 5, 15, 16, 30, 0, 0, time.UTC),
				},
				{
					ID:       "file2",
					Name:     "Training Records",
					Type:     "spreadsheet",
					MimeType: "application/vnd.google-apps.spreadsheet",
					Owner:    "HR Team",
					Size:     4096,
					Created:  time.Date(2023, 3, 10, 8, 15, 0, 0, time.UTC),
					Modified: time.Date(2023, 6, 1, 11, 45, 0, 0, time.UTC),
				},
			},
			Metadata: map[string]interface{}{
				"revision_id":   "rev123456",
				"sheet_count":   2,
				"row_count":     3,
				"custom_field":  "custom_value",
				"numeric_field": 42,
				"boolean_field": true,
			},
		}

		rules := ExtractionRules{
			IncludeMetadata:  true,
			IncludeRevisions: true,
		}

		report := gwt.generateReport(result, "sheets", rules)

		// Verify report structure and content
		assert.Contains(t, report, "# Google Workspace Evidence Report")
		assert.Contains(t, report, "**Document Type**: sheets")
		assert.Contains(t, report, "**Document Name**: Complete Test Document")
		assert.Contains(t, report, "**Document ID**: test-doc-123")

		// Check metadata section
		assert.Contains(t, report, "## Document Metadata")
		assert.Contains(t, report, "- **Owner**: Test Owner")
		assert.Contains(t, report, "- **Created**: 2023-01-15 09:30:00")
		assert.Contains(t, report, "- **Last Modified**: 2023-06-20 14:45:30")
		assert.Contains(t, report, "- **MIME Type**: application/vnd.google-apps.spreadsheet")
		assert.Contains(t, report, "- **Editors**: Editor1, Editor2, Editor3")

		// Check folder contents section
		assert.Contains(t, report, "## Folder Contents")
		assert.Contains(t, report, "### Policy Document")
		assert.Contains(t, report, "### Training Records")
		assert.Contains(t, report, "- **Type**: document")
		assert.Contains(t, report, "- **Type**: spreadsheet")
		assert.Contains(t, report, "- **Size**: 2048 bytes")
		assert.Contains(t, report, "- **Size**: 4096 bytes")

		// Check sheet data section
		assert.Contains(t, report, "## Sheet Data Summary")
		assert.Contains(t, report, "- **Total Rows**: 3")
		assert.Contains(t, report, "- **Columns**: 4")

		// Check document content section
		assert.Contains(t, report, "## Document Content")
		assert.Contains(t, report, "```")
		assert.Contains(t, report, result.Content)

		// Check additional metadata section
		assert.Contains(t, report, "## Additional Metadata")
		assert.Contains(t, report, "- **revision_id**: rev123456")
		assert.Contains(t, report, "- **sheet_count**: 2")
		assert.Contains(t, report, "- **custom_field**: custom_value")
		assert.Contains(t, report, "- **numeric_field**: 42")
		assert.Contains(t, report, "- **boolean_field**: true")
	})

	t.Run("Generate_Report_Without_Optional_Fields", func(t *testing.T) {
		result := &GoogleWorkspaceResult{
			DocumentID:   "minimal-doc",
			DocumentName: "Minimal Document",
			DocumentType: "docs",
			Content:      "", // Empty content
		}

		rules := ExtractionRules{
			IncludeMetadata: false,
		}

		report := gwt.generateReport(result, "docs", rules)

		// Should still have basic structure
		assert.Contains(t, report, "# Google Workspace Evidence Report")
		assert.Contains(t, report, "**Document Name**: Minimal Document")

		// Should not have metadata section since IncludeMetadata is false
		assert.NotContains(t, report, "## Document Metadata")

		// Should show no content message
		assert.Contains(t, report, "*No content extracted*")

		// Should not have optional sections
		assert.NotContains(t, report, "## Folder Contents")
		assert.NotContains(t, report, "## Sheet Data Summary")
		assert.NotContains(t, report, "## Additional Metadata")
	})

	t.Run("Calculate_Relevance_Comprehensive", func(t *testing.T) {
		// Test all relevance factors
		result := &GoogleWorkspaceResult{
			DocumentType:   "forms",                                     // +0.15
			Content:        strings.Repeat("Important content. ", 1000), // Long content: +0.3
			ModifiedAt:     time.Now().Add(-2 * time.Hour),              // Recent: +0.2
			FolderContents: make([]FolderItem, 25),                      // 25 items: +0.25
		}

		relevance := gwt.calculateRelevance(result)

		// Base (0.5) + forms (0.15) + content (0.3) + recent (0.2) + folder (0.25) = 1.4, capped at 1.0
		assert.Equal(t, 1.0, relevance)

		// Test minimum relevance
		minResult := &GoogleWorkspaceResult{
			DocumentType: "docs",
			Content:      "",                           // No content
			ModifiedAt:   time.Now().AddDate(-2, 0, 0), // Very old
		}

		minRelevance := gwt.calculateRelevance(minResult)
		assert.Equal(t, 0.5, minRelevance) // Base relevance only

		// Test specific content length thresholds
		testCases := []struct {
			contentLength int
			expectedBoost float64
		}{
			{50, 0.0},   // Below 100, no boost
			{500, 0.1},  // 100-1000 range
			{2000, 0.2}, // 1000-5000 range
			{8000, 0.3}, // Above 5000
		}

		for _, tc := range testCases {
			result := &GoogleWorkspaceResult{
				DocumentType: "docs",
				Content:      strings.Repeat("A", tc.contentLength),
			}
			relevance := gwt.calculateRelevance(result)
			expected := 0.5 + tc.expectedBoost // Base + content boost
			assert.Equal(t, expected, relevance, "Content length %d should give boost %f", tc.contentLength, tc.expectedBoost)
		}
	})

	t.Run("All_MIME_Type_Categories", func(t *testing.T) {
		testCases := []struct {
			mimeType string
			expected string
		}{
			// Google Workspace types
			{"application/vnd.google-apps.document", "document"},
			{"application/vnd.google-apps.spreadsheet", "spreadsheet"},
			{"application/vnd.google-apps.presentation", "presentation"},
			{"application/vnd.google-apps.form", "form"},
			{"application/vnd.google-apps.folder", "folder"},
			{"application/vnd.google-apps.drawing", "drawing"},

			// Mixed case variations (case-sensitive matching)
			{"Application/VND.Google-Apps.Document", "file"},    // Case sensitive, won't match
			{"APPLICATION/VND.GOOGLE-APPS.SPREADSHEET", "file"}, // Case sensitive, won't match

			// Common file types
			{"application/pdf", "file"},
			{"image/jpeg", "file"},
			{"image/png", "file"},
			{"text/plain", "file"},
			{"text/html", "file"},
			{"application/json", "file"},
			{"application/xml", "file"},

			// Edge cases
			{"", "file"},
			{"invalid", "file"},
			{"application/unknown", "file"},
			{"text/vnd.google-apps.document", "document"}, // Contains "document" keyword
		}

		for _, tc := range testCases {
			result := gwt.getMimeTypeCategory(tc.mimeType)
			assert.Equal(t, tc.expected, result, "MIME type '%s' should be categorized as '%s'", tc.mimeType, tc.expected)
		}
	})

	t.Run("Time_Parsing_Comprehensive", func(t *testing.T) {
		testCases := []struct {
			input       string
			shouldParse bool
			expected    time.Time
		}{
			// Valid formats
			{"2023-06-01T12:00:00.000Z", true, time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)},
			{"2023-06-01T12:00:00Z", true, time.Date(2023, 6, 1, 12, 0, 0, 0, time.UTC)},
			{"2023-12-31T23:59:59.999Z", true, time.Date(2023, 12, 31, 23, 59, 59, 999000000, time.UTC)},
			{"2024-02-29T00:00:00Z", true, time.Date(2024, 2, 29, 0, 0, 0, 0, time.UTC)}, // Leap year

			// Invalid formats that should fail gracefully
			{"", false, time.Time{}},
			{"invalid-timestamp", false, time.Time{}},
			{"2023-13-01T12:00:00Z", false, time.Time{}}, // Invalid month
			{"2023-06-32T12:00:00Z", false, time.Time{}}, // Invalid day
			{"2023-06-01T25:00:00Z", false, time.Time{}}, // Invalid hour
			{"2023-06-01T12:60:00Z", false, time.Time{}}, // Invalid minute
			{"2023-06-01T12:00:60Z", false, time.Time{}}, // Invalid second
			{"2023-02-29T12:00:00Z", false, time.Time{}}, // Not a leap year
			{"not-a-date", false, time.Time{}},
			{"2023/06/01 12:00:00", false, time.Time{}}, // Wrong format
		}

		for _, tc := range testCases {
			result := gwt.parseGoogleTime(tc.input)
			if tc.shouldParse {
				assert.Equal(t, tc.expected, result, "Input '%s' should parse to %v", tc.input, tc.expected)
				assert.False(t, result.IsZero(), "Parsed time should not be zero for '%s'", tc.input)
			} else {
				assert.True(t, result.IsZero(), "Input '%s' should result in zero time", tc.input)
			}
		}
	})
}

// TestGoogleWorkspaceValidationUnit tests validation functions without complex setup
func TestGoogleWorkspaceValidationUnit(t *testing.T) {
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

	t.Run("Validation_All_Scenarios", func(t *testing.T) {
		// Test comprehensive validation scenarios
		docConfig := DocumentConfig{
			DocumentID:   "validation-test-doc",
			DocumentName: "Validation Test Document",
			DocumentType: "sheets",
			Validation: GoogleValidationRules{
				MinContentLength: 100,
				RequiredKeywords: []string{"security", "compliance", "policy"},
				DateRange: &DateRange{
					From: "2023-01-01",
					To:   "2023-12-31",
				},
			},
		}

		testCases := []struct {
			name             string
			content          string
			metadata         map[string]interface{}
			expectedErrors   int
			expectedWarnings int
			shouldPass       bool
		}{
			{
				name:             "Perfect_Match",
				content:          strings.Repeat("This document covers security, compliance, and policy requirements. ", 10),
				metadata:         map[string]interface{}{"modified_at": time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC)},
				expectedErrors:   0,
				expectedWarnings: 0,
				shouldPass:       true,
			},
			{
				name:             "Content_Too_Short",
				content:          "Short content with security, compliance, policy",
				metadata:         map[string]interface{}{"modified_at": time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC)},
				expectedErrors:   1, // Content length error
				expectedWarnings: 0,
				shouldPass:       false,
			},
			{
				name:             "Missing_All_Keywords",
				content:          strings.Repeat("This document contains general business information without specific terms. ", 5),
				metadata:         map[string]interface{}{"modified_at": time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC)},
				expectedErrors:   0,
				expectedWarnings: 3,    // All three keywords missing
				shouldPass:       true, // Warnings don't fail validation
			},
			{
				name:             "Date_Before_Range",
				content:          strings.Repeat("This document covers security, compliance, and policy requirements. ", 10),
				metadata:         map[string]interface{}{"modified_at": time.Date(2022, 12, 31, 23, 59, 59, 0, time.UTC)},
				expectedErrors:   0,
				expectedWarnings: 1, // Date before range
				shouldPass:       true,
			},
			{
				name:             "Date_After_Range",
				content:          strings.Repeat("This document covers security, compliance, and policy requirements. ", 10),
				metadata:         map[string]interface{}{"modified_at": time.Date(2024, 1, 1, 0, 0, 1, 0, time.UTC)},
				expectedErrors:   0,
				expectedWarnings: 1, // Date after range
				shouldPass:       true,
			},
			{
				name:             "Multiple_Issues",
				content:          "Short text",                                                                       // Too short, missing keywords
				metadata:         map[string]interface{}{"modified_at": time.Date(2022, 1, 1, 0, 0, 0, 0, time.UTC)}, // Before range
				expectedErrors:   1,                                                                                  // Content length
				expectedWarnings: 4,                                                                                  // Three missing keywords + date before range
				shouldPass:       false,
			},
			{
				name:             "Case_Insensitive_Keywords",
				content:          strings.Repeat("This document covers SECURITY, Compliance, and POLICY requirements. ", 10),
				metadata:         map[string]interface{}{"modified_at": time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC)},
				expectedErrors:   0,
				expectedWarnings: 0, // Keywords should match case-insensitively
				shouldPass:       true,
			},
			{
				name:             "Partial_Keywords",
				content:          strings.Repeat("This document covers security and compliance requirements. ", 10), // Missing "policy"
				metadata:         map[string]interface{}{"modified_at": time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC)},
				expectedErrors:   0,
				expectedWarnings: 1, // One missing keyword
				shouldPass:       true,
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				source := &models.EvidenceSource{
					Content:  tc.content,
					Metadata: tc.metadata,
				}

				validation := enhancedTool.validateDocumentContent(docConfig, source)

				assert.Equal(t, tc.shouldPass, validation["passed"].(bool), "Validation pass/fail mismatch")

				errors := validation["errors"].([]string)
				assert.Len(t, errors, tc.expectedErrors, "Expected %d errors, got %d: %v", tc.expectedErrors, len(errors), errors)

				warnings := validation["warnings"].([]string)
				assert.Len(t, warnings, tc.expectedWarnings, "Expected %d warnings, got %d: %v", tc.expectedWarnings, len(warnings), warnings)
			})
		}
	})

	t.Run("Validation_Edge_Cases", func(t *testing.T) {
		// Test validation with edge case configurations
		docConfig := DocumentConfig{
			DocumentID:   "edge-case-doc",
			DocumentName: "Edge Case Document",
			DocumentType: "docs",
			Validation: GoogleValidationRules{
				MinContentLength: 0,          // Zero minimum length
				RequiredKeywords: []string{}, // No required keywords
				DateRange: &DateRange{ // Invalid date format
					From: "invalid-date",
					To:   "also-invalid",
				},
			},
		}

		source := &models.EvidenceSource{
			Content: "Any content",
			Metadata: map[string]interface{}{
				"modified_at": time.Date(2023, 6, 15, 10, 0, 0, 0, time.UTC),
			},
		}

		validation := enhancedTool.validateDocumentContent(docConfig, source)

		// Should pass with invalid date formats (graceful handling)
		assert.True(t, validation["passed"].(bool))
		assert.Empty(t, validation["errors"].([]string))
		assert.Empty(t, validation["warnings"].([]string))
	})
}

// TestMappingsFilePathResolution tests the getMappingsFilePath function comprehensively
func TestMappingsFilePathResolution(t *testing.T) {
	cfgLogger := &logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(t, err)

	t.Run("Data_Directory_Priority", func(t *testing.T) {
		tempDir := t.TempDir()
		dataPath := filepath.Join(tempDir, "google_evidence_mappings.yaml")
		err := os.WriteFile(dataPath, []byte("test content"), 0644)
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

	t.Run("Current_Directory_Fallback", func(t *testing.T) {
		// Create a temporary directory and change to it
		tempDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Create mapping file in current directory
		currentPath := filepath.Join(tempDir, "google_evidence_mappings.yaml")
		err = os.WriteFile(currentPath, []byte("test content"), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			Storage: config.StorageConfig{
				DataDir: "/nonexistent/path",
			},
		}

		loader := NewGoogleEvidenceMappingsLoader(cfg, log)
		path := loader.getMappingsFilePath()

		// Should find the file in current directory (returns relative path)
		assert.Equal(t, "google_evidence_mappings.yaml", path)
	})

	t.Run("Configs_Directory_Fallback", func(t *testing.T) {
		tempDir := t.TempDir()
		originalDir, err := os.Getwd()
		require.NoError(t, err)
		defer os.Chdir(originalDir)

		err = os.Chdir(tempDir)
		require.NoError(t, err)

		// Create configs directory and mapping file
		configsDir := filepath.Join(tempDir, "configs")
		err = os.MkdirAll(configsDir, 0755)
		require.NoError(t, err)

		configPath := filepath.Join(configsDir, "google_evidence_mappings.yaml")
		err = os.WriteFile(configPath, []byte("test content"), 0644)
		require.NoError(t, err)

		cfg := &config.Config{
			Storage: config.StorageConfig{
				DataDir: "/nonexistent/path",
			},
		}

		loader := NewGoogleEvidenceMappingsLoader(cfg, log)
		path := loader.getMappingsFilePath()
		// Returns relative path for configs directory
		assert.Equal(t, "configs/google_evidence_mappings.yaml", path)
	})

	t.Run("Default_Fallback", func(t *testing.T) {
		cfg := &config.Config{
			Storage: config.StorageConfig{
				DataDir: "/completely/nonexistent/path",
			},
		}

		loader := NewGoogleEvidenceMappingsLoader(cfg, log)
		path := loader.getMappingsFilePath()
		assert.Equal(t, "google_evidence_mappings.yaml", path)
	})
}

// TestFolderContentsSorting tests the folder contents sorting functionality
func TestFolderContentsSorting(t *testing.T) {
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
	gwt := tool.(*GoogleWorkspaceTool)

	t.Run("Sort_By_Modified_Date", func(t *testing.T) {
		result := &GoogleWorkspaceResult{
			DocumentID:   "folder-sort-test",
			DocumentName: "Sort Test Folder",
			DocumentType: "folder",
			FolderContents: []FolderItem{
				{
					ID:       "file1",
					Name:     "Oldest File",
					Modified: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
				},
				{
					ID:       "file2",
					Name:     "Newest File",
					Modified: time.Date(2023, 6, 1, 10, 0, 0, 0, time.UTC),
				},
				{
					ID:       "file3",
					Name:     "Middle File",
					Modified: time.Date(2023, 3, 1, 10, 0, 0, 0, time.UTC),
				},
			},
		}

		rules := ExtractionRules{IncludeMetadata: true}
		report := gwt.generateReport(result, "drive", rules)

		// The report should list files in order by modification date (newest first)
		assert.Contains(t, report, "## Folder Contents")

		// Find positions of each file in the report
		newestPos := strings.Index(report, "### Newest File")
		middlePos := strings.Index(report, "### Middle File")
		oldestPos := strings.Index(report, "### Oldest File")

		// Verify ordering: newest should appear before middle, middle before oldest
		assert.True(t, newestPos < middlePos && middlePos < oldestPos,
			"Files should be sorted by modification date (newest first). Positions: newest=%d, middle=%d, oldest=%d",
			newestPos, middlePos, oldestPos)
	})
}
