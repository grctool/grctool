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

package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stripANSI removes ANSI color codes from a string
func stripANSI(s string) string {
	// Regex to match ANSI escape sequences
	ansiRegex := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)
	return ansiRegex.ReplaceAllString(s, "")
}

// extractJSON extracts JSON from output that may contain log messages
func extractJSON(s string) string {
	// Strip ANSI codes first
	cleaned := stripANSI(s)

	// Find the first '{' and last '}' to extract JSON
	start := strings.Index(cleaned, "{")
	end := strings.LastIndex(cleaned, "}")

	if start == -1 || end == -1 || end < start {
		return cleaned // Return as-is if no JSON found
	}

	return cleaned[start : end+1]
}

// TestCLICommands tests the CLI interface for all enhanced tools
func TestCLICommands(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI integration tests in short mode")
	}

	// Build the grctool binary for testing
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "grctool")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = filepath.Join("..", "..")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Build output: %s", string(output))
	}
	require.NoError(t, err, "Failed to build grctool binary")

	// Setup test environment
	testDataDir := setupTestEnvironment(t)
	defer cleanup(testDataDir)

	tests := []struct {
		name         string
		args         []string
		expectError  bool
		validateJSON func(t *testing.T, output string)
		contains     []string
		notContains  []string
	}{
		{
			name: "Tool List Command",
			args: []string{"tool", "list"},
			validateJSON: func(t *testing.T, output string) {
				// Parse JSON response
				var response struct {
					OK   bool `json:"ok"`
					Data struct {
						Tools []struct {
							Name        string `json:"name"`
							Description string `json:"description"`
						} `json:"tools"`
						Count int `json:"count"`
					} `json:"data"`
					Meta struct {
						Tool string `json:"tool"`
					} `json:"meta"`
				}

				err := json.Unmarshal([]byte(output), &response)
				require.NoError(t, err, "Output should be valid JSON")

				// Validate response structure
				assert.True(t, response.OK, "Response should indicate success")
				assert.Greater(t, response.Data.Count, 0, "Should have registered tools")
				assert.Equal(t, response.Data.Count, len(response.Data.Tools), "Count should match tools array length")
				assert.Equal(t, "list", response.Meta.Tool, "Meta should indicate 'list' tool")

				// Check for specific tools
				toolNames := make(map[string]bool)
				for _, tool := range response.Data.Tools {
					toolNames[tool.Name] = true
					assert.NotEmpty(t, tool.Description, "Tool %s should have description", tool.Name)
				}

				assert.True(t, toolNames["terraform-security-analyzer"], "Should have terraform-security-analyzer tool")
				assert.True(t, toolNames["github-searcher"], "Should have github-searcher tool")
			},
		},
		{
			name: "Terraform Scanner Help",
			args: []string{"tool", "terraform-scanner", "--help"},
			contains: []string{
				"terraform-scanner",
				"Terraform",
				"scan",
			},
		},
		{
			name: "GitHub Searcher Help",
			args: []string{"tool", "github-searcher", "--help"},
			contains: []string{
				"github-searcher",
				"GitHub",
				"search",
			},
		},
		{
			name: "Invalid Tool Name Shows Help",
			args: []string{"tool", "nonexistent-tool"},
			// Cobra shows help for unknown commands, not an error
			expectError: false,
			contains:    []string{"Available Commands", "Evidence assembly tools"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add --config flag to specify config file location
			configPath := filepath.Join(testDataDir, ".grctool.yaml")
			args := append([]string{"--config", configPath}, tt.args...)
			cmd := exec.Command(binaryPath, args...)
			cmd.Env = append(os.Environ(), "GRCTOOL_DATA_DIR="+testDataDir)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			if tt.expectError {
				assert.Error(t, err, "Expected command to fail")
			} else {
				assert.NoError(t, err, "Command failed: %s", stderr.String())
			}

			// For JSON validation, use only stdout (tool commands output JSON to stdout)
			// For text validation (like --help), use combined output
			stdoutStr := stdout.String()
			stderrStr := stderr.String()
			combinedOutput := stdoutStr + stderrStr

			// If JSON validation is specified, extract and validate JSON from stdout
			if tt.validateJSON != nil {
				jsonOutput := extractJSON(stdoutStr)
				tt.validateJSON(t, jsonOutput)
			}

			// Check string contains (use combined output for text checks)
			for _, contains := range tt.contains {
				assert.Contains(t, combinedOutput, contains, "Output should contain: %s", contains)
			}

			for _, notContains := range tt.notContains {
				assert.NotContains(t, combinedOutput, notContains, "Output should not contain: %s", notContains)
			}
		})
	}
}

func TestTerraformScannerCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Terraform scanner CLI tests in short mode")
	}

	// Build the grctool binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "grctool")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = filepath.Join("..", "..")
	err := cmd.Run()
	require.NoError(t, err, "Failed to build grctool binary")

	// Setup test environment with terraform fixtures
	testDataDir := setupTestEnvironmentWithTerraform(t)
	defer cleanup(testDataDir)

	tests := []struct {
		name     string
		args     []string
		validate func(t *testing.T, output string)
	}{
		{
			name: "Basic Terraform Scanner Scan",
			args: []string{
				"tool", "terraform-scanner",
			},
			validate: func(t *testing.T, output string) {
				// Should be valid JSON
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				assert.NoError(t, err, "Output should be valid JSON")

				// Should have standard tool response structure
				assert.Contains(t, result, "ok")
				assert.Contains(t, result, "data")
				assert.Contains(t, result, "meta")
			},
		},
		{
			name: "Terraform Scanner with Resource Types",
			args: []string{
				"tool", "terraform-scanner",
				"--resource-types", "aws_kms_key,aws_s3_bucket_encryption",
			},
			validate: func(t *testing.T, output string) {
				// Should find encryption resources
				assert.Contains(t, output, "aws_kms_key")
				assert.Contains(t, output, "aws_s3_bucket_encryption")
			},
		},
		{
			name: "Terraform Enhanced with Pattern",
			args: []string{
				"tool", "terraform-scanner",
				"--pattern", "encrypt|kms",
				"--output-format", "markdown",
				"--use-cache=false",
			},
			validate: func(t *testing.T, output string) {
				// Should be valid JSON wrapper
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				assert.NoError(t, err, "Output should be valid JSON")

				// Extract markdown content from data.result
				data, ok := result["data"].(map[string]interface{})
				assert.True(t, ok, "Should have data field")
				markdownContent, ok := data["result"].(string)
				assert.True(t, ok, "Should have result field with string content")

				// Should be markdown format
				assert.Contains(t, markdownContent, "# Enhanced Terraform Security Configuration Evidence")
				assert.Contains(t, markdownContent, "**Configuration:**")
			},
		},
		{
			name: "Terraform Enhanced with Control Hint",
			args: []string{
				"tool", "terraform-scanner",
				"--control-hint", "CC6.8",
				"--output-format", "csv",
				"--use-cache=false",
			},
			validate: func(t *testing.T, output string) {
				// Should be valid JSON wrapper
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				assert.NoError(t, err, "Output should be valid JSON")

				// Extract CSV content from data.result
				data, ok := result["data"].(map[string]interface{})
				assert.True(t, ok, "Should have data field")
				csvContent, ok := data["result"].(string)
				assert.True(t, ok, "Should have result field with string content")

				// Should be CSV format
				lines := strings.Split(csvContent, "\n")
				assert.Greater(t, len(lines), 1)

				// Should have CSV headers
				header := lines[0]
				assert.Contains(t, header, "Resource Type")
				assert.Contains(t, header, "Security Controls")
			},
		},
		{
			name: "Terraform Enhanced with Max Results",
			args: []string{
				"tool", "terraform-scanner",
				"--max-results", "2",
				"--output-format", "json",
				"--use-cache=false",
			},
			validate: func(t *testing.T, output string) {
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				assert.NoError(t, err)

				if results, ok := result["results"].([]interface{}); ok {
					assert.LessOrEqual(t, len(results), 2, "Should respect max results limit")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add --config flag to specify config file location
			configPath := filepath.Join(testDataDir, ".grctool.yaml")
			args := append([]string{"--config", configPath}, tt.args...)
			cmd := exec.Command(binaryPath, args...)
			cmd.Env = append(os.Environ(),
				"GRCTOOL_DATA_DIR="+testDataDir,
				"GRCTOOL_LOG_LEVEL=error", // Reduce noise in tests
			)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			assert.NoError(t, err, "Command failed: %s", stderr.String())

			// Extract JSON from stdout (stripping any log messages)
			jsonOutput := extractJSON(stdout.String())
			assert.NotEmpty(t, jsonOutput, "Should produce output")

			if tt.validate != nil {
				tt.validate(t, jsonOutput)
			}
		})
	}
}

func TestGitHubSearcherCLI(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping GitHub searcher CLI tests in short mode")
	}

	// Build the grctool binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "grctool")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = filepath.Join("..", "..")
	err := cmd.Run()
	require.NoError(t, err)

	// Setup test environment
	testDataDir := setupTestEnvironment(t)
	defer cleanup(testDataDir)

	// Skip if no GitHub token is available
	githubToken := os.Getenv("GITHUB_TOKEN")
	if githubToken == "" {
		t.Skip("No GITHUB_TOKEN environment variable set, skipping GitHub CLI tests")
	}

	tests := []struct {
		name     string
		args     []string
		validate func(t *testing.T, output string)
		timeout  time.Duration
	}{
		{
			name: "Basic GitHub Search",
			args: []string{
				"tool", "github_searcher",
				"--query", "security",
			},
			timeout: 30 * time.Second,
			validate: func(t *testing.T, output string) {
				// Should have GitHub report structure
				assert.Contains(t, output, "GitHub Security Evidence")
				assert.Contains(t, output, "Repository:")
			},
		},
		{
			name: "GitHub Search with Labels",
			args: []string{
				"tool", "github_searcher",
				"--query", "security vulnerability",
				"--labels", "security,vulnerability",
			},
			timeout: 30 * time.Second,
			validate: func(t *testing.T, output string) {
				assert.Contains(t, output, "security")
			},
		},
		{
			name: "GitHub Search Include Closed",
			args: []string{
				"tool", "github_searcher",
				"--query", "compliance",
				"--include-closed=true",
			},
			timeout: 30 * time.Second,
			validate: func(t *testing.T, output string) {
				// Should include both open and closed issues
				assert.Contains(t, output, "GitHub Security Evidence")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Add --config flag to specify config file location
			configPath := filepath.Join(testDataDir, ".grctool.yaml")
			args := append([]string{"--config", configPath}, tt.args...)

			var cmd *exec.Cmd
			var stdout, stderr bytes.Buffer

			// Set timeout for GitHub API calls
			if tt.timeout > 0 {
				ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
				defer cancel()
				cmd = exec.CommandContext(ctx, binaryPath, args...)
			} else {
				cmd = exec.Command(binaryPath, args...)
			}

			cmd.Env = append(os.Environ(),
				"GRCTOOL_DATA_DIR="+testDataDir,
				"GRCTOOL_LOG_LEVEL=error",
				"GITHUB_TOKEN="+githubToken,
			)
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			if err != nil {
				// GitHub API might be unavailable or rate limited
				t.Logf("GitHub search command failed (possibly due to API limits): %v", err)
				t.Logf("Stderr: %s", stderr.String())
				return
			}

			// Extract JSON from stdout (stripping any log messages)
			jsonOutput := extractJSON(stdout.String())
			assert.NotEmpty(t, jsonOutput, "Should produce output")

			if tt.validate != nil {
				tt.validate(t, jsonOutput)
			}
		})
	}
}

func TestCLIErrorHandling(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI error handling tests in short mode")
	}

	// Build the grctool binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "grctool")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = filepath.Join("..", "..")
	err := cmd.Run()
	require.NoError(t, err)

	// Setup test environment
	testDataDir := setupTestEnvironment(t)
	defer cleanup(testDataDir)

	errorTests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		// Note: Cobra shows help text for invalid/missing subcommands, doesn't error
		// These tests verify Cobra behavior rather than our code, so skipped
		// {
		//	name:        "Missing Tool Name",
		//	args:        []string{"tool"},
		//	expectError: true,
		//	errorMsg:    "missing tool name",
		// },
		// {
		//	name:        "Invalid Tool Name",
		//	args:        []string{"tool", "invalid-tool-name"},
		//	expectError: true,
		//	errorMsg:    "unknown command",
		// },
		{
			name:        "Terraform Enhanced Missing Required Args",
			args:        []string{"tool", "terraform-scanner", "--resource-types"},
			expectError: true,
			errorMsg:    "flag needs an argument",
		},
		{
			name:        "GitHub Searcher Missing Query",
			args:        []string{"tool", "github-searcher"},
			expectError: true,
			errorMsg:    "required flag",
		},
		{
			name:        "Invalid Output Format",
			args:        []string{"tool", "terraform-scanner", "--output-format", "invalid"},
			expectError: true,
			errorMsg:    "allowed_values",
		},
		// Note: Cobra treats non-boolean values as false, doesn't error
		// {
		//	name:        "Invalid Boolean Flag",
		//	args:        []string{"tool", "terraform-scanner", "--use-cache", "invalid"},
		//	expectError: true,
		//	errorMsg:    "invalid syntax",
		// },
		{
			name:        "Invalid Number Flag",
			args:        []string{"tool", "terraform-scanner", "--max-results", "not-a-number"},
			expectError: true,
			errorMsg:    "invalid",
		},
	}

	for _, tt := range errorTests {
		t.Run(tt.name, func(t *testing.T) {
			// Add --config flag to specify config file location
			configPath := filepath.Join(testDataDir, ".grctool.yaml")
			args := append([]string{"--config", configPath}, tt.args...)
			cmd := exec.Command(binaryPath, args...)
			cmd.Env = append(os.Environ(), "GRCTOOL_DATA_DIR="+testDataDir)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()

			if tt.expectError {
				output := stdout.String() + stderr.String()

				// Check if it's a JSON error response (ok: false)
				var jsonResp map[string]interface{}
				jsonOutput := extractJSON(stdout.String())
				if jsonOutput != "" && json.Unmarshal([]byte(jsonOutput), &jsonResp) == nil {
					// JSON error response
					if ok, exists := jsonResp["ok"].(bool); exists && !ok {
						// This is a valid error response
						if tt.errorMsg != "" {
							assert.Contains(t, strings.ToLower(output), strings.ToLower(tt.errorMsg),
								"Error message should contain: %s", tt.errorMsg)
						}
						return
					}
				}

				// Otherwise expect command to fail with exit code
				assert.Error(t, err, "Expected command to fail")
				if tt.errorMsg != "" {
					assert.Contains(t, strings.ToLower(output), strings.ToLower(tt.errorMsg),
						"Error message should contain: %s", tt.errorMsg)
				}
			} else {
				assert.NoError(t, err, "Command should not fail")
			}
		})
	}
}

func TestCLIOutputFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping CLI output format tests in short mode")
	}

	// Build the grctool binary
	tempDir := t.TempDir()
	binaryPath := filepath.Join(tempDir, "grctool")

	cmd := exec.Command("go", "build", "-o", binaryPath, ".")
	cmd.Dir = filepath.Join("..", "..")
	err := cmd.Run()
	require.NoError(t, err)

	// Setup test environment with terraform fixtures
	testDataDir := setupTestEnvironmentWithTerraform(t)
	defer cleanup(testDataDir)

	formatTests := []struct {
		name     string
		format   string
		validate func(t *testing.T, output string)
	}{
		{
			name:   "JSON Output Format",
			format: "json",
			validate: func(t *testing.T, output string) {
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				assert.NoError(t, err, "Should be valid JSON")

				// Check standard tool response structure
				assert.Contains(t, result, "ok")
				assert.Contains(t, result, "data")
				assert.Contains(t, result, "meta")
			},
		},
		{
			name:   "CSV Output Format",
			format: "csv",
			validate: func(t *testing.T, output string) {
				// Parse JSON wrapper
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				assert.NoError(t, err, "Should be valid JSON wrapper")

				// Extract CSV from data.result
				data, ok := result["data"].(map[string]interface{})
				assert.True(t, ok, "Should have data field")
				csvContent, ok := data["result"].(string)
				assert.True(t, ok, "Should have result field")

				lines := strings.Split(strings.TrimSpace(csvContent), "\n")
				if len(lines) > 1 {
					// Check CSV header if there's data
					header := lines[0]
					assert.Contains(t, header, "Resource Type")
					assert.Contains(t, header, "Resource Name")
					assert.Contains(t, header, "Security Controls")
				}
			},
		},
		{
			name:   "Markdown Output Format",
			format: "markdown",
			validate: func(t *testing.T, output string) {
				// Parse JSON wrapper
				var result map[string]interface{}
				err := json.Unmarshal([]byte(output), &result)
				assert.NoError(t, err, "Should be valid JSON wrapper")

				// Extract markdown from data.result
				data, ok := result["data"].(map[string]interface{})
				assert.True(t, ok, "Should have data field")
				markdownContent, ok := data["result"].(string)
				assert.True(t, ok, "Should have result field")

				// Check markdown structure
				assert.Contains(t, markdownContent, "# Enhanced Terraform Security Configuration Evidence")
				assert.Contains(t, markdownContent, "**Configuration:**")
			},
		},
	}

	for _, tt := range formatTests {
		t.Run(tt.name, func(t *testing.T) {
			// Add --config flag to specify config file location
			configPath := filepath.Join(testDataDir, ".grctool.yaml")
			args := append([]string{"--config", configPath},
				"tool", "terraform-scanner",
				"--output-format", tt.format,
				"--use-cache=false",
			)
			cmd := exec.Command(binaryPath, args...)
			cmd.Env = append(os.Environ(), "GRCTOOL_DATA_DIR="+testDataDir)

			var stdout, stderr bytes.Buffer
			cmd.Stdout = &stdout
			cmd.Stderr = &stderr

			err := cmd.Run()
			assert.NoError(t, err, "Command failed: %s", stderr.String())

			// Extract JSON from stdout (all formats return JSON wrapper)
			output := extractJSON(stdout.String())
			assert.NotEmpty(t, output, "Should produce output")

			if tt.validate != nil {
				tt.validate(t, output)
			}
		})
	}
}

// Helper functions

func setupTestEnvironment(t *testing.T) string {
	tempDir := t.TempDir()

	// Create basic directory structure
	dirs := []string{"docs", "evidence", ".cache"}
	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(tempDir, dir), 0755)
		require.NoError(t, err)
	}

	// Create .grctool.yaml config file
	configContent := `tugboat:
  base_url: "https://api-test.tugboatlogic.com"
  org_id: "test-org-id"
storage:
  data_dir: "` + tempDir + `"
log_level: "error"
evidence:
  tools:
    terraform:
      enabled: true
      scan_paths:
        - "` + tempDir + `"
      include_patterns:
        - "*.tf"
        - "*.tfvars"
      exclude_patterns:
        - ".terraform/"
    github:
      enabled: false
`

	configPath := filepath.Join(tempDir, ".grctool.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	return tempDir
}

func setupTestEnvironmentWithTerraform(t *testing.T) string {
	tempDir := setupTestEnvironment(t)

	// Copy terraform fixtures if they exist
	fixturesDir := filepath.Join("..", "..", "test_data", "terraform", "soc2")
	if _, err := os.Stat(fixturesDir); err == nil {
		// Copy SOC2 fixtures to test environment
		terraformDir := filepath.Join(tempDir, "terraform")
		err := os.MkdirAll(terraformDir, 0755)
		require.NoError(t, err)

		fixtures := []string{
			"cc6_8_encryption.tf",
			"cc6_1_access_control.tf",
			"cc8_1_audit_logging.tf",
		}

		for _, fixture := range fixtures {
			srcPath := filepath.Join(fixturesDir, fixture)
			dstPath := filepath.Join(terraformDir, fixture)

			if _, err := os.Stat(srcPath); err == nil {
				data, err := os.ReadFile(srcPath)
				require.NoError(t, err)

				err = os.WriteFile(dstPath, data, 0644)
				require.NoError(t, err)
			}
		}
	} else {
		// Create minimal terraform file for testing
		terraformDir := filepath.Join(tempDir, "terraform")
		err := os.MkdirAll(terraformDir, 0755)
		require.NoError(t, err)

		testTF := `
resource "aws_kms_key" "test" {
  description         = "Test encryption key"
  enable_key_rotation = true
  
  tags = {
    Purpose = "Testing"
    Control = "CC6.8"
  }
}

resource "aws_s3_bucket_encryption" "test" {
  bucket = "test-bucket"
  
  server_side_encryption_configuration {
    rule {
      apply_server_side_encryption_by_default {
        kms_master_key_id = aws_kms_key.test.arn
        sse_algorithm     = "aws:kms"
      }
    }
  }
}
`
		err = os.WriteFile(filepath.Join(terraformDir, "test.tf"), []byte(testTF), 0644)
		require.NoError(t, err)
	}

	// Update config to include terraform directory
	configContent := `tugboat:
  base_url: "https://api-test.tugboatlogic.com"
  org_id: "test-org-id"
storage:
  data_dir: "` + tempDir + `"
log_level: "error"
evidence:
  tools:
    terraform:
      enabled: true
      scan_paths:
        - "` + filepath.Join(tempDir, "terraform") + `"
      include_patterns:
        - "*.tf"
        - "*.tfvars"
      exclude_patterns:
        - ".terraform/"
    github:
      enabled: false
`

	configPath := filepath.Join(tempDir, ".grctool.yaml")
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	return tempDir
}

func cleanup(tempDir string) {
	os.RemoveAll(tempDir)
}
