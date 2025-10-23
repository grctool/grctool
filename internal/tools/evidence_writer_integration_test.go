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

//go:build !e2e

package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestEvidenceWriterIntegration tests evidence writing with real components
func TestEvidenceWriterIntegration(t *testing.T) {
	// Setup test environment with real components
	tempDir := t.TempDir()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}

	// Create test logger
	log, err := logger.New(&logger.Config{
		Level: logger.DebugLevel,
	})
	require.NoError(t, err)

	// Setup test data using fixtures
	setupTestData(t, tempDir)

	// Create real tool with real validator
	tool := NewEvidenceWriterTool(cfg, log)
	require.NotNil(t, tool)

	ctx := context.Background()

	// Test 1: Basic evidence writing
	params := map[string]interface{}{
		"task_ref": "327992", // Use numeric ID that exists in test data
		"title":    "Test Evidence Document",
		"content":  "# Test Evidence\n\nThis is test evidence content.",
		"format":   "markdown",
		"summary":  "Test summary",
	}

	result, evidenceSource, err := tool.Execute(ctx, params)

	// Test outcomes, not implementation details
	require.NoError(t, err)
	assert.Contains(t, result, "Evidence written successfully")
	assert.NotNil(t, evidenceSource)

	// Verify evidence file was created
	evidenceFiles := findEvidenceFiles(tempDir)
	assert.Len(t, evidenceFiles, 1)

	// Verify collection plan was created
	planFiles := findPlanFiles(tempDir)
	assert.Len(t, planFiles, 1)
}

// Helper functions following AGENTS.md - use real implementations, not mocks

// setupTestData copies test fixtures to the temp directory
func setupTestData(t *testing.T, tempDir string) {
	// Create directory structure with new json subdirectory
	docsDir := filepath.Join(tempDir, "docs", "evidence_tasks", "json")
	require.NoError(t, os.MkdirAll(docsDir, 0755))

	// Copy test evidence task file
	taskData, err := os.ReadFile(filepath.Join("testdata", "evidence_task_sample.json"))
	require.NoError(t, err)

	taskFile := filepath.Join(docsDir, "ET1_327992_Access_Control_Registration_and_De-registration_Process_Document.json")
	require.NoError(t, os.WriteFile(taskFile, taskData, 0644))
}

// findEvidenceFiles finds all evidence files in the temp directory
func findEvidenceFiles(tempDir string) []string {
	var evidenceFiles []string
	evidenceDir := filepath.Join(tempDir, "evidence")

	if _, err := os.Stat(evidenceDir); os.IsNotExist(err) {
		return evidenceFiles
	}

	err := filepath.Walk(evidenceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && (filepath.Ext(path) == ".md" || filepath.Ext(path) == ".csv") {
			// Exclude plan files
			if filepath.Base(path) != "collection_plan.md" {
				evidenceFiles = append(evidenceFiles, path)
			}
		}
		return nil
	})

	if err != nil {
		return evidenceFiles
	}

	return evidenceFiles
}

// findPlanFiles finds all collection plan files
func findPlanFiles(tempDir string) []string {
	var planFiles []string
	evidenceDir := filepath.Join(tempDir, "evidence")

	if _, err := os.Stat(evidenceDir); os.IsNotExist(err) {
		return planFiles
	}

	err := filepath.Walk(evidenceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && filepath.Base(path) == "collection_plan.md" {
			planFiles = append(planFiles, path)
		}
		return nil
	})

	if err != nil {
		return planFiles
	}

	return planFiles
}
