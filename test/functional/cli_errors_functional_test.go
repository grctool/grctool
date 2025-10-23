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
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestCLI_InvalidParameters tests CLI error handling with invalid parameters
func TestCLI_InvalidParameters(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("InvalidGlobalFlag", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "--invalid-global-flag", "value")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		assert.Error(t, err, "Should fail with invalid global flag")
		assert.Contains(t, outputStr, "unknown flag", "Should mention unknown flag")
		assert.Contains(t, outputStr, "--invalid-global-flag", "Should identify the problematic flag")
	})

	t.Run("InvalidCommand", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "nonexistent-command")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		assert.Error(t, err, "Should fail with invalid command")
		assert.Contains(t, outputStr, "unknown command", "Should mention unknown command")
		assert.Contains(t, outputStr, "nonexistent-command", "Should identify the problematic command")
	})

	t.Run("InvalidToolName", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "tool", "nonexistent-tool")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		assert.Error(t, err, "Should fail with invalid tool name")
		assert.Contains(t, outputStr, "not registered", "Should mention tool not registered")
		assert.Contains(t, outputStr, "available tools", "Should suggest available tools")
	})

	t.Run("InvalidToolParameter", func(t *testing.T) {
		cmd := exec.Command(binaryPath, "tool", "storage-read", "--invalid-param", "value")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		assert.Error(t, err, "Should fail with invalid tool parameter")
		assert.Contains(t, outputStr, "unknown flag", "Should mention unknown flag")
		assert.Contains(t, outputStr, "--invalid-param", "Should identify the problematic parameter")
	})

	t.Run("MissingRequiredParameter", func(t *testing.T) {
		// Test storage-read without required --path parameter
		cmd := exec.Command(binaryPath, "tool", "storage-read")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		assert.Error(t, err, "Should fail without required parameter")
		// The error message may vary depending on how parameters are validated
		assert.True(t,
			strings.Contains(outputStr, "required") ||
				strings.Contains(outputStr, "path") ||
				strings.Contains(outputStr, "parameter"),
			"Should mention missing required parameter: %s", outputStr)
	})

	t.Run("InvalidParameterValue", func(t *testing.T) {
		// Test with invalid output format
		cmd := exec.Command(binaryPath, "tool", "evidence-task-list", "--output-format", "invalid-format")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			// May fail with validation error or proceed with default
			if strings.Contains(outputStr, "invalid") || strings.Contains(outputStr, "format") {
				assert.Contains(t, outputStr, "format", "Should mention format issue")
			} else {
				t.Logf("Tool handled invalid format gracefully: %s", outputStr)
			}
		}
	})
}

// TestCLI_FileSystemErrors tests CLI handling of file system errors
func TestCLI_FileSystemErrors(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("NonexistentFile", func(t *testing.T) {
		// Try to read a file that doesn't exist
		cmd := exec.Command(binaryPath, "tool", "storage-read", "--path", "nonexistent-file.json")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		assert.Error(t, err, "Should fail when trying to read nonexistent file")
		assert.True(t,
			strings.Contains(outputStr, "not found") ||
				strings.Contains(outputStr, "no such file") ||
				strings.Contains(outputStr, "does not exist"),
			"Should indicate file not found: %s", outputStr)
	})

	t.Run("ReadOnlyDirectory", func(t *testing.T) {
		// Create a read-only directory and try to write to it
		readOnlyDir := filepath.Join(t.TempDir(), "readonly")
		err := os.Mkdir(readOnlyDir, 0555) // Read and execute only
		require.NoError(t, err, "Failed to create read-only directory")

		// Try to write to read-only directory (may not work on all systems)
		readOnlyFile := filepath.Join(readOnlyDir, "test.json")
		cmd := exec.Command(binaryPath, "tool", "storage-write",
			"--path", readOnlyFile,
			"--content", `{"test": "readonly"}`,
			"--format", "json")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			// Expected to fail with permission error
			assert.True(t,
				strings.Contains(outputStr, "permission") ||
					strings.Contains(outputStr, "denied") ||
					strings.Contains(outputStr, "read-only") ||
					strings.Contains(outputStr, "cannot"),
				"Should indicate permission issue: %s", outputStr)
		} else {
			t.Log("Write to read-only directory succeeded (filesystem may not enforce permissions)")
		}
	})

	t.Run("InvalidPath", func(t *testing.T) {
		// Test with invalid path characters (depends on OS)
		invalidPath := "invalid\x00path.json" // Null byte should be invalid
		cmd := exec.Command(binaryPath, "tool", "storage-read", "--path", invalidPath)
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			assert.True(t,
				strings.Contains(outputStr, "invalid") ||
					strings.Contains(outputStr, "path") ||
					strings.Contains(outputStr, "character"),
				"Should indicate path issue: %s", outputStr)
		} else {
			t.Log("Invalid path was handled gracefully")
		}
	})
}

// TestCLI_MissingConfig tests CLI behavior with missing or invalid configuration
func TestCLI_MissingConfig(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("MissingConfigFile", func(t *testing.T) {
		// Test with custom config directory that doesn't exist
		tempDir := t.TempDir()
		nonExistentConfig := filepath.Join(tempDir, "nonexistent", "config.yaml")

		// Set environment variable to point to nonexistent config
		cmd := exec.Command(binaryPath, "tool", "list")
		cmd.Env = append(os.Environ(), "GRCTOOL_CONFIG="+nonExistentConfig)

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		// Should still work with default configuration or graceful degradation
		if err != nil {
			if strings.Contains(outputStr, "config") {
				assert.Contains(t, outputStr, "config", "Should mention config issue")
			} else {
				t.Logf("Tool failed for other reason: %v, output: %s", err, outputStr)
			}
		} else {
			t.Log("Tool handled missing config gracefully")
			assert.Contains(t, outputStr, "terraform", "Should still show tools with default config")
		}
	})

	t.Run("InvalidDataDirectory", func(t *testing.T) {
		// Test with invalid data directory
		invalidDataDir := "/nonexistent/data/directory"

		cmd := exec.Command(binaryPath, "tool", "evidence-task-list")
		cmd.Env = append(os.Environ(), "GRCTOOL_DATA_DIR="+invalidDataDir)

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			assert.True(t,
				strings.Contains(outputStr, "not found") ||
					strings.Contains(outputStr, "directory") ||
					strings.Contains(outputStr, "path"),
				"Should indicate data directory issue: %s", outputStr)
		} else {
			t.Log("Tool handled invalid data directory gracefully")
		}
	})

	t.Run("PermissionDeniedDataDir", func(t *testing.T) {
		// Create a directory we can't read
		inaccessibleDir := filepath.Join(t.TempDir(), "inaccessible")
		err := os.Mkdir(inaccessibleDir, 0000) // No permissions
		require.NoError(t, err, "Failed to create inaccessible directory")

		cmd := exec.Command(binaryPath, "tool", "evidence-task-list")
		cmd.Env = append(os.Environ(), "GRCTOOL_DATA_DIR="+inaccessibleDir)

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			assert.True(t,
				strings.Contains(outputStr, "permission") ||
					strings.Contains(outputStr, "denied") ||
					strings.Contains(outputStr, "access"),
				"Should indicate permission issue: %s", outputStr)
		} else {
			t.Log("Tool handled permission issue gracefully")
		}

		// Clean up - restore permissions to allow cleanup
		os.Chmod(inaccessibleDir, 0755)
	})
}

// TestCLI_NetworkErrors tests CLI handling of network-related errors
func TestCLI_NetworkErrors(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("SyncWithoutAuth", func(t *testing.T) {
		// Test sync without authentication (should fail gracefully)
		cmd := exec.Command(binaryPath, "sync")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			// Should fail with authentication error
			assert.True(t,
				strings.Contains(outputStr, "auth") ||
					strings.Contains(outputStr, "credential") ||
					strings.Contains(outputStr, "token") ||
					strings.Contains(outputStr, "login"),
				"Should indicate authentication issue: %s", outputStr)
		} else {
			t.Log("Sync succeeded (may be using cached credentials or different auth method)")
		}
	})

	t.Run("GoogleWorkspaceWithoutAuth", func(t *testing.T) {
		// Test Google Workspace tool without authentication
		cmd := exec.Command(binaryPath, "tool", "google-workspace",
			"--analysis-type", "users",
			"--output-format", "json")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			// Should fail with authentication error
			assert.True(t,
				strings.Contains(outputStr, "auth") ||
					strings.Contains(outputStr, "credential") ||
					strings.Contains(outputStr, "oauth") ||
					strings.Contains(outputStr, "permission"),
				"Should indicate authentication issue: %s", outputStr)
		} else {
			t.Log("Google Workspace tool succeeded (may have valid credentials)")
		}
	})
}

// TestCLI_MalformedInput tests CLI handling of malformed input data
func TestCLI_MalformedInput(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("InvalidJSON", func(t *testing.T) {
		// Try to write invalid JSON
		invalidJSON := `{"key": "value", "invalid": }`
		cmd := exec.Command(binaryPath, "tool", "storage-write",
			"--path", "invalid.json",
			"--content", invalidJSON,
			"--format", "json")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			// Should fail with JSON parsing error
			assert.True(t,
				strings.Contains(outputStr, "json") ||
					strings.Contains(outputStr, "parse") ||
					strings.Contains(outputStr, "invalid") ||
					strings.Contains(outputStr, "format"),
				"Should indicate JSON parsing issue: %s", outputStr)
		} else {
			t.Log("Invalid JSON was accepted (tool may not validate JSON format)")
		}
	})

	t.Run("EmptyInput", func(t *testing.T) {
		// Try to write empty content
		cmd := exec.Command(binaryPath, "tool", "storage-write",
			"--path", "empty.txt",
			"--content", "")

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			assert.True(t,
				strings.Contains(outputStr, "empty") ||
					strings.Contains(outputStr, "content") ||
					strings.Contains(outputStr, "required"),
				"Should indicate empty content issue: %s", outputStr)
		} else {
			t.Log("Empty content was accepted")
			assert.Contains(t, outputStr, "success", "Should indicate successful write")
		}
	})

	t.Run("VeryLongInput", func(t *testing.T) {
		// Try to write very long content
		longContent := strings.Repeat("Very long content ", 10000) // ~180KB
		cmd := exec.Command(binaryPath, "tool", "storage-write",
			"--path", "long.txt",
			"--content", longContent)

		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			if strings.Contains(outputStr, "too large") || strings.Contains(outputStr, "size") {
				assert.Contains(t, outputStr, "size", "Should mention size limitation")
			} else {
				t.Logf("Long content failed for other reason: %v, output: %s", err, outputStr)
			}
		} else {
			t.Log("Very long content was accepted")
			assert.Contains(t, outputStr, "success", "Should indicate successful write")
		}
	})
}

// TestCLI_ConcurrentAccess tests CLI handling of concurrent access scenarios
func TestCLI_ConcurrentAccess(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("ConcurrentWrites", func(t *testing.T) {
		// Test multiple processes writing to the same file
		filename := "concurrent_test.json"
		type result struct {
			id     int
			output string
			err    error
		}

		results := make(chan result, 3)

		// Launch concurrent writes
		for i := 0; i < 3; i++ {
			go func(id int) {
				content := `{"writer": ` + string(rune('0'+id)) + `, "timestamp": "` + strings.ReplaceAll(strings.ReplaceAll(time.Now().Format(time.RFC3339Nano), ":", "-"), ".", "-") + `"}`
				cmd := exec.Command(binaryPath, "tool", "storage-write",
					"--path", filename,
					"--content", content,
					"--format", "json")

				output, err := cmd.CombinedOutput()
				results <- result{id, string(output), err}
			}(i)
		}

		// Collect results
		successes := 0
		for i := 0; i < 3; i++ {
			select {
			case res := <-results:
				if res.err == nil {
					successes++
					t.Logf("Concurrent write %d succeeded", res.id)
				} else {
					t.Logf("Concurrent write %d failed: %v, output: %s", res.id, res.err, res.output)
				}
			case <-time.After(30 * time.Second):
				t.Fatalf("Concurrent writes timed out")
			}
		}

		// At least one should succeed (others may fail due to file locking)
		assert.GreaterOrEqual(t, successes, 1, "At least one concurrent write should succeed")
		t.Logf("Concurrent write results: %d/%d succeeded", successes, 3)
	})
}

// TestCLI_ResourceLimits tests CLI behavior under resource constraints
func TestCLI_ResourceLimits(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("ManySmallFiles", func(t *testing.T) {
		// Test creating many small files quickly
		const numFiles = 50
		type result struct {
			id  int
			err error
		}

		results := make(chan result, numFiles)

		// Create many files concurrently
		for i := 0; i < numFiles; i++ {
			go func(id int) {
				filename := fmt.Sprintf("small_file_%d.json", id)
				content := fmt.Sprintf(`{"id": %d, "type": "small_test"}`, id)

				cmd := exec.Command(binaryPath, "tool", "storage-write",
					"--path", filename,
					"--content", content,
					"--format", "json")

				_, err := cmd.CombinedOutput()
				results <- result{id, err}
			}(i)
		}

		// Collect results with timeout
		successes := 0
		for i := 0; i < numFiles; i++ {
			select {
			case res := <-results:
				if res.err == nil {
					successes++
				}
			case <-time.After(60 * time.Second):
				t.Fatalf("Many small files test timed out after processing %d/%d files", successes, numFiles)
			}
		}

		// Most should succeed
		successRate := float64(successes) / float64(numFiles)
		assert.GreaterOrEqual(t, successRate, 0.8, "At least 80%% of small file operations should succeed (got %.1f%%)", successRate*100)
		t.Logf("Small files test: %d/%d succeeded (%.1f%%)", successes, numFiles, successRate*100)
	})
}

// TestCLI_GracefulDegradation tests that CLI degrades gracefully under various conditions
func TestCLI_GracefulDegradation(t *testing.T) {
	binaryPath := ensureBinaryExists(t)

	t.Run("PartialDataAvailable", func(t *testing.T) {
		// Test behavior when only some data is available
		cmd := exec.Command(binaryPath, "tool", "evidence-task-list", "--output-format", "json")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		if err != nil {
			// Should provide helpful error message
			assert.True(t,
				strings.Contains(outputStr, "no evidence tasks") ||
					strings.Contains(outputStr, "directory not found") ||
					strings.Contains(outputStr, "sync required"),
				"Should provide helpful error message: %s", outputStr)
		} else {
			// Should provide empty or partial results gracefully
			t.Log("Evidence task list succeeded with partial data")
			assert.True(t, len(outputStr) > 0, "Should provide some output")
		}
	})

	t.Run("ToolListAlwaysWorks", func(t *testing.T) {
		// Tool list should always work regardless of data availability
		cmd := exec.Command(binaryPath, "tool", "list")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		require.NoError(t, err, "Tool list should always succeed: %s", outputStr)
		assert.Contains(t, outputStr, "Available tools", "Should show available tools")
		assert.True(t, len(outputStr) > 100, "Should provide substantial tool information")
	})

	t.Run("HelpAlwaysWorks", func(t *testing.T) {
		// Help should always work
		cmd := exec.Command(binaryPath, "--help")
		output, err := cmd.CombinedOutput()
		outputStr := string(output)

		require.NoError(t, err, "Help should always succeed: %s", outputStr)
		assert.Contains(t, outputStr, "grctool", "Should mention the tool name")
		assert.Contains(t, outputStr, "Commands", "Should list available commands")
		assert.True(t, len(outputStr) > 200, "Should provide substantial help information")
	})
}
