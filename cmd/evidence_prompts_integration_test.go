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

//go:build integration
// +build integration

package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvidencePromptsCommandIntegration(t *testing.T) {
	// Test the actual behavior we're seeing in production
	t.Run("evidence analyze --all with no synced data", func(t *testing.T) {
		// Create a temporary directory for test data
		tempDir, err := os.MkdirTemp("", "grctool-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a complete config file
		configContent := `
tugboat:
  base_url: "https://app.tugboatlogic.com"
  cookie_header: "test-cookie"
storage:
  data_dir: "` + tempDir + `"
logging:
  level: info
  file: false
evidence:
  claude:
    api_key: "test-key"
`
		configPath := filepath.Join(tempDir, ".grctool.yaml")
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Execute the command directly using the actual CLI
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"--config", configPath, "evidence", "analyze", "--all"})

		// Reset command for clean state
		evidenceAnalyzeCmd.Flags().Set("all", "false")

		err = rootCmd.Execute()

		// The command should not error when no tasks are found
		assert.NoError(t, err)

		output := buf.String()
		// When properly implemented, it should show 0 tasks found
		assert.Contains(t, output, "Found 0 evidence tasks")
		// And should suggest syncing
		assert.Contains(t, output, "Run 'grctool sync --evidence'")
	})

	t.Run("evidence analyze with specific missing task ID", func(t *testing.T) {
		// Create a temporary directory for test data
		tempDir, err := os.MkdirTemp("", "grctool-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create a complete config file
		configContent := `
tugboat:
  base_url: "https://app.tugboatlogic.com"
  cookie_header: "test-cookie"
storage:
  data_dir: "` + tempDir + `"
logging:
  level: info
  file: false
evidence:
  claude:
    api_key: "test-key"
`
		configPath := filepath.Join(tempDir, ".grctool.yaml")
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Execute the command
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"--config", configPath, "evidence", "analyze", "862839"})

		err = rootCmd.Execute()

		// This should error with task not found
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "evidence task not found: 862839")
	})

	t.Run("evidence analyze with existing task", func(t *testing.T) {
		// Create a temporary directory for test data
		tempDir, err := os.MkdirTemp("", "grctool-test-*")
		require.NoError(t, err)
		defer os.RemoveAll(tempDir)

		// Create evidence_tasks directory
		evidenceTasksDir := filepath.Join(tempDir, "evidence_tasks")
		err = os.MkdirAll(evidenceTasksDir, 0755)
		require.NoError(t, err)

		// Create evidence_prompts directory
		evidencePromptsDir := filepath.Join(tempDir, "evidence_prompts")
		err = os.MkdirAll(evidencePromptsDir, 0755)
		require.NoError(t, err)

		// Create a sample evidence task
		taskData := `{
			"id": 328001,
			"name": "Test Evidence Task",
			"description": "Test task for unit testing",
			"framework": "SOC2",
			"priority": "High",
			"status": "pending",
			"reference_id": "ET10",
			"collection_interval": "Annually",
			"guidance": "Test guidance",
			"controls": [],
			"completed": false,
			"ad_hoc": false,
			"sensitive": false,
			"created_at": "2024-01-01T00:00:00Z",
			"updated_at": "2024-01-01T00:00:00Z"
		}`
		taskPath := filepath.Join(evidenceTasksDir, "ET10_328001_Test_Evidence_Task.json")
		err = os.WriteFile(taskPath, []byte(taskData), 0644)
		require.NoError(t, err)

		// Create a complete config file
		configContent := `
tugboat:
  base_url: "https://app.tugboatlogic.com"
  cookie_header: "test-cookie"
storage:
  data_dir: "` + tempDir + `"
logging:
  level: info
  file: false
evidence:
  claude:
    api_key: "test-key"
`
		configPath := filepath.Join(tempDir, ".grctool.yaml")
		err = os.WriteFile(configPath, []byte(configContent), 0644)
		require.NoError(t, err)

		// Execute the command
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"--config", configPath, "evidence", "analyze", "328001"})

		err = rootCmd.Execute()

		// This should succeed
		assert.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "Analyzing evidence task 328001")
		assert.Contains(t, output, "Test Evidence Task")
		assert.Contains(t, output, "Prompt generated:")
		assert.Contains(t, output, "Related controls: 0, Related policies: 0")

		// Check that prompt file was created
		promptFiles, err := os.ReadDir(evidencePromptsDir)
		require.NoError(t, err)
		assert.Equal(t, 1, len(promptFiles))
		assert.Contains(t, promptFiles[0].Name(), "ET10_328001_Test_Evidence_Task.md")
	})
}
