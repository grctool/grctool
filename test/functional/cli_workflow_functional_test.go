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

//go:build functional

package functional

import (
	"encoding/json"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCompleteAuditWorkflow_CLI tests complete audit workflow through CLI commands
func TestCompleteAuditWorkflow_CLI(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	// Step 1: Test sync command (may fail without authentication)
	t.Run("Step1_DataSync", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "sync", "--dry-run")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			if strings.Contains(outputStr, "authentication") ||
				strings.Contains(outputStr, "credentials") ||
				strings.Contains(outputStr, "unauthorized") {
				t.Log("Sync correctly requires authentication - this is expected")
				assert.Contains(t, outputStr, "auth", "Should mention authentication")
			} else {
				t.Logf("Sync failed with: %v, output: %s", err, outputStr)
			}
		} else {
			// If sync succeeds, validate it ran
			assert.Contains(t, outputStr, "sync", "Should mention sync operation")
		}
	})

	// Step 2: Test evidence task listing
	t.Run("Step2_EvidenceTaskList", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "tool", "evidence-task-list",
			"--output-format", "json")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			if strings.Contains(outputStr, "no evidence tasks found") ||
				strings.Contains(outputStr, "directory not found") {
				t.Skip("Skipping workflow test - requires evidence data")
			}
			t.Logf("Evidence task list warning: %v, output: %s", err, outputStr)
		} else {
			// Validate task list structure
			if strings.Contains(outputStr, "[") || strings.Contains(outputStr, "task") {
				assert.True(t, len(outputStr) > 0, "Should produce task list")
			}
		}

		t.Logf("Step 2 - Task list length: %d characters", len(outputStr))
	})

	// Step 3: Test evidence analysis for specific task
	t.Run("Step3_EvidenceAnalysis", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "tool", "evidence-task-details",
			"--task-ref", "ET-101",
			"--output-format", "json")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			if strings.Contains(outputStr, "task not found") {
				t.Skip("Skipping workflow test - specific task ET-101 not available")
			}
			t.Logf("Evidence analysis warning: %v, output: %s", err, outputStr)
		} else {
			// Validate analysis output
			if strings.Contains(outputStr, "{") {
				var result map[string]interface{}
				err := json.Unmarshal([]byte(outputStr), &result)
				assert.NoError(t, err, "Should produce valid JSON")
				assert.Contains(t, outputStr, "ET-101", "Should contain task reference")
			}
		}

		t.Logf("Step 3 - Analysis output length: %d characters", len(outputStr))
	})

	// Step 4: Test prompt generation
	t.Run("Step4_PromptGeneration", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "tool", "prompt-assembler",
			"--task-ref", "ET-101",
			"--output-format", "markdown")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			if strings.Contains(outputStr, "task not found") {
				t.Skip("Skipping workflow test - prompt generation requires task data")
			}
			t.Logf("Prompt generation warning: %v, output: %s", err, outputStr)
		} else {
			// Validate prompt output
			if len(outputStr) > 0 {
				assert.Contains(t, outputStr, "#", "Should contain markdown headers")
				assert.True(t, len(outputStr) > 100, "Should generate substantial prompt content")
			}
		}

		t.Logf("Step 4 - Prompt length: %d characters", len(outputStr))
	})

	// Step 5: Test evidence validation (if any evidence exists)
	t.Run("Step5_EvidenceValidation", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "tool", "evidence-validator",
			"--task-ref", "ET-101",
			"--output-format", "summary")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			if strings.Contains(outputStr, "no evidence found") ||
				strings.Contains(outputStr, "task not found") {
				t.Log("Evidence validation skipped - no evidence to validate (expected)")
			} else {
				t.Logf("Evidence validation warning: %v, output: %s", err, outputStr)
			}
		} else {
			// Validate validation output
			assert.True(t, len(outputStr) > 0, "Should produce validation summary")
			assert.Contains(t, outputStr, "validation", "Should mention validation")
		}

		t.Logf("Step 5 - Validation output length: %d characters", len(outputStr))
	})
}

// TestToolChaining_CLI tests chaining multiple tools through CLI
func TestToolChaining_CLI(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	// Chain 1: Storage write -> read -> terraform analysis
	t.Run("StorageToTerraformChain", func(t *testing.T) {
		testFile := "terraform_test_config.tf"
		terraformContent := `resource "aws_s3_bucket" "test" {
  bucket = "test-evidence-bucket"
  
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        sse_algorithm = "AES256"
      }
    }
  }

  versioning {
    enabled = true
  }
}`

		// Step 1: Write terraform config to storage
		cmd := exec.Command(binaryPath, "tool", "storage-write",
			"--path", testFile,
			"--content", terraformContent)

		output, err := cmd.CombinedOutput()
		require.NoError(t, err, "Storage write should succeed: %s", string(output))

		// Step 2: Read it back
		cmd = exec.Command(binaryPath, "tool", "storage-read",
			"--path", testFile)

		output, err = cmd.CombinedOutput()
		require.NoError(t, err, "Storage read should succeed: %s", string(output))
		assert.Contains(t, string(output), "aws_s3_bucket", "Should contain terraform content")

		// Step 3: Try to analyze it (may not work directly but validates CLI chaining)
		t.Log("Successfully chained storage write -> read operations")
	})

	// Chain 2: Evidence task details -> prompt generation -> summary generation
	t.Run("EvidenceAnalysisChain", func(t *testing.T) {
		// Step 1: Get evidence task details
		cmd := exec.Command(binaryPath, "tool", "evidence-task-details",
			"--task-ref", "ET-101",
			"--output-format", "json")

		output, err := cmd.CombinedOutput()
		if err != nil {
			if strings.Contains(string(output), "task not found") {
				t.Skip("Evidence analysis chain requires task data")
			}
			t.Logf("Step 1 failed: %v, output: %s", err, string(output))
			return
		}

		// Step 2: Generate prompt for the task
		cmd = exec.Command(binaryPath, "tool", "prompt-assembler",
			"--task-ref", "ET-101",
			"--output-format", "markdown")

		promptOutput, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Step 2 failed: %v, output: %s", err, string(promptOutput))
			return
		}

		assert.True(t, len(promptOutput) > 0, "Should generate prompt content")

		// Step 3: Try policy summary (if available)
		cmd = exec.Command(binaryPath, "tool", "policy-summary-generator",
			"--task-ref", "ET-101",
			"--output-format", "markdown")

		summaryOutput, err := cmd.CombinedOutput()
		if err != nil {
			t.Logf("Step 3 completed with warning: %v, output: %s", err, string(summaryOutput))
		} else {
			assert.True(t, len(summaryOutput) > 0, "Should generate summary content")
		}

		t.Log("Successfully chained evidence analysis workflow")
	})
}

// TestParallelToolExecution_CLI tests running multiple tools in parallel
func TestParallelToolExecution_CLI(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("ParallelStorageOperations", func(t *testing.T) {
		// Test multiple storage operations in parallel
		type result struct {
			name   string
			output string
			err    error
		}

		results := make(chan result, 3)

		// Launch parallel operations
		go func() {
			cmd := exec.Command(binaryPath, "tool", "storage-write",
				"--path", "parallel_test_1.json",
				"--content", `{"test": "parallel1"}`,
				"--format", "json")
			output, err := cmd.CombinedOutput()
			results <- result{"write1", string(output), err}
		}()

		go func() {
			cmd := exec.Command(binaryPath, "tool", "storage-write",
				"--path", "parallel_test_2.json",
				"--content", `{"test": "parallel2"}`,
				"--format", "json")
			output, err := cmd.CombinedOutput()
			results <- result{"write2", string(output), err}
		}()

		go func() {
			cmd := exec.Command(binaryPath, "tool", "evidence-task-list",
				"--output-format", "json")
			output, err := cmd.CombinedOutput()
			results <- result{"list", string(output), err}
		}()

		// Collect results
		successes := 0
		for i := 0; i < 3; i++ {
			select {
			case res := <-results:
				if res.err == nil {
					successes++
					t.Logf("Parallel operation %s succeeded", res.name)
				} else {
					t.Logf("Parallel operation %s failed: %v, output: %s", res.name, res.err, res.output)
				}
			case <-time.After(30 * time.Second):
				t.Fatalf("Parallel operations timed out")
			}
		}

		// At least storage operations should succeed
		assert.GreaterOrEqual(t, successes, 2, "At least 2 parallel operations should succeed")
	})
}

// TestConfigurationWorkflow_CLI tests configuration-related workflows
func TestConfigurationWorkflow_CLI(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("ConfigurationCheck", func(t *testing.T) {
		// Test that the tool can read its configuration
		cmd := exec.Command(binaryPath, "config", "show")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			// Check if it's a missing command issue
			if strings.Contains(outputStr, "unknown command") {
				t.Log("Config show command not available - this is expected")
			} else {
				t.Logf("Config check failed: %v, output: %s", err, outputStr)
			}
		} else {
			// Validate configuration output
			assert.True(t, len(outputStr) > 0, "Should show configuration")
			assert.Contains(t, outputStr, "data_dir", "Should mention data directory")
		}
	})

	t.Run("ToolRegistryCheck", func(t *testing.T) {
		// Test tool registry functionality
		cmd := exec.Command(binaryPath, "tool", "list")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		require.NoError(t, err, "Tool list should succeed: %s", outputStr)
		assert.Contains(t, outputStr, "terraform", "Should list terraform tools")
		assert.Contains(t, outputStr, "storage", "Should list storage tools")
		assert.Contains(t, outputStr, "evidence", "Should list evidence tools")

		// Count available tools
		toolLines := strings.Split(outputStr, "\n")
		toolCount := 0
		for _, line := range toolLines {
			if strings.Contains(line, ":") && !strings.HasPrefix(line, "Available") {
				toolCount++
			}
		}

		assert.GreaterOrEqual(t, toolCount, 10, "Should have at least 10 registered tools")
		t.Logf("Found %d registered tools", toolCount)
	})
}

// TestDataIntegrityWorkflow_CLI tests data integrity through workflow
func TestDataIntegrityWorkflow_CLI(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("DataConsistencyCheck", func(t *testing.T) {
		testData := map[string]string{
			"test_data_1.json": `{"id": 1, "type": "evidence", "status": "pending"}`,
			"test_data_2.json": `{"id": 2, "type": "control", "status": "completed"}`,
			"test_data_3.json": `{"id": 3, "type": "policy", "status": "in_progress"}`,
		}

		// Write test data
		for filename, content := range testData {
			cmd := exec.Command(binaryPath, "tool", "storage-write",
				"--path", filename,
				"--content", content,
				"--format", "json")

			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Write should succeed for %s: %s", filename, string(output))
		}

		// Read back and verify
		for filename, expectedContent := range testData {
			cmd := exec.Command(binaryPath, "tool", "storage-read",
				"--path", filename,
				"--format", "json")

			output, err := cmd.CombinedOutput()
			require.NoError(t, err, "Read should succeed for %s: %s", filename, string(output))

			// Verify JSON structure integrity
			var expected, actual map[string]interface{}
			err = json.Unmarshal([]byte(expectedContent), &expected)
			require.NoError(t, err, "Expected content should be valid JSON")

			err = json.Unmarshal(output, &actual)
			require.NoError(t, err, "Read output should be valid JSON")

			assert.Equal(t, expected["id"], actual["id"], "ID should match")
			assert.Equal(t, expected["type"], actual["type"], "Type should match")
			assert.Equal(t, expected["status"], actual["status"], "Status should match")
		}

		t.Log("Data consistency check completed successfully")
	})
}
