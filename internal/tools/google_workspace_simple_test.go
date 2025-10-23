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
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGoogleWorkspaceToolBasics(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: "/tmp",
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
		result.Content = "A very long piece of content that should get a higher relevance score due to its length and the amount of information it contains. This is a substantial document with meaningful content that would be valuable for evidence collection purposes."
		for i := 0; i < 10; i++ {
			result.Content += " Additional content to make it even longer and more substantial."
		}
		relevance = gwt.calculateRelevance(result)
		assert.True(t, relevance > 0.5)

		// Test forms relevance boost
		result.DocumentType = "forms"
		relevance = gwt.calculateRelevance(result)
		assert.True(t, relevance > 0.5)

		// Test folder contents boost
		result.FolderContents = []FolderItem{{}, {}, {}} // 3 items
		relevance = gwt.calculateRelevance(result)
		assert.True(t, relevance > 0.5)
	})
}

func TestGoogleEvidenceMappingsLoaderSimple(t *testing.T) {
	// Create test config
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: "/tmp",
		},
	}

	cfgLogger := &logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	}
	log, err := logger.New(cfgLogger)
	require.NoError(t, err)

	t.Run("Loader Creation", func(t *testing.T) {
		loader := NewGoogleEvidenceMappingsLoader(cfg, log)
		assert.NotNil(t, loader)
	})

	t.Run("Enhanced Tool Creation", func(t *testing.T) {
		enhancedTool := NewGoogleWorkspaceToolWithMappings(cfg, log)
		assert.NotNil(t, enhancedTool)

		// Test refresh mappings (should not error even without mappings file)
		enhancedTool.RefreshMappings()
	})
}
