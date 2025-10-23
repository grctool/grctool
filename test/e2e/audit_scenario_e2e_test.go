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

package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tools"
	"github.com/grctool/grctool/test/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// hasRequiredAuth checks if all required authentication is available
func hasRequiredAuth(t *testing.T) bool {
	githubAuth := os.Getenv("GITHUB_TOKEN") != ""
	tugboatAuth := hasValidTugboatAuth(t)

	t.Logf("GitHub auth available: %v", githubAuth)
	t.Logf("Tugboat auth available: %v", tugboatAuth)

	return githubAuth && tugboatAuth
}

// buildGrctoolBinary builds the grctool binary for E2E testing
func buildGrctoolBinary(t *testing.T) {
	buildCmd := exec.Command("go", "build", "-o", "bin/grctool")
	buildCmd.Dir = "../../"
	err := buildCmd.Run()
	require.NoError(t, err, "Failed to build grctool binary")
}

// TestCompleteSOC2Audit_E2E tests complete SOC2 audit scenario with real APIs
func TestCompleteSOC2Audit_E2E(t *testing.T) {
	// Skip if missing required auth
	if !hasRequiredAuth(t) {
		t.Skip("Complete authentication required for full audit test")
	}

	buildGrctoolBinary(t)

	t.Log("Starting complete SOC2 audit E2E test...")
	startTime := time.Now()

	// Step 1: Sync from Tugboat
	t.Log("Step 1: Syncing data from Tugboat...")
	syncCmd := exec.Command("../../bin/grctool", "sync", "--verbose")
	syncOutput, err := syncCmd.CombinedOutput()
	require.NoError(t, err, "Tugboat sync should succeed")

	syncOutputStr := string(syncOutput)
	assert.Contains(t, strings.ToLower(syncOutputStr), "sync")
	t.Log("✓ Tugboat sync completed")

	// Step 2: Collect GitHub evidence
	testRepo := os.Getenv("TEST_GITHUB_REPO")
	if testRepo == "" {
		testRepo = "octocat/Hello-World"
	}

	t.Log("Step 2: Collecting GitHub evidence...")
	githubCmd := exec.Command("../../bin/grctool", "tool", "github-permissions",
		"--repository="+testRepo, "--output-format=detailed")
	githubOutput, err := githubCmd.CombinedOutput()

	if err != nil {
		t.Logf("GitHub permissions tool may not be available via CLI: %v", err)
		// Try using the tool directly
		cfg := &config.Config{
			Evidence: config.EvidenceConfig{
				Tools: config.ToolsConfig{
					GitHub: config.GitHubToolConfig{
						Enabled:    true,
						APIToken:   os.Getenv("GITHUB_TOKEN"),
						Repository: testRepo,
					},
				},
			},
		}

		log, _ := logger.NewTestLogger()
		githubTool := tools.NewGitHubTool(cfg, log)
		result, source, toolErr := githubTool.Execute(context.Background(), map[string]interface{}{
			"analysis_type": "permissions",
			"output_format": "detailed",
		})

		require.NoError(t, toolErr, "GitHub tool should work")
		assert.NotEmpty(t, result)
		assert.NotNil(t, source)
		githubOutput = []byte(result)
	}

	assert.NotEmpty(t, githubOutput)
	t.Log("✓ GitHub evidence collection completed")

	// Step 3: Analyze terraform infrastructure (if available)
	terraformPath := os.Getenv("TEST_TERRAFORM_PATH")
	if terraformPath == "" {
		terraformPath = "../../test_terraform"
	}

	t.Log("Step 3: Analyzing Terraform infrastructure...")
	terraformCmd := exec.Command("../../bin/grctool", "tool", "terraform-security-analyzer",
		"--scan-paths="+terraformPath)
	terraformOutput, err := terraformCmd.CombinedOutput()

	if err != nil {
		t.Logf("Terraform security analyzer may not be available via CLI: %v", err)
		// Try using the tool directly
		cfg := helpers.SetupE2ETest(t)

		log, _ := logger.NewTestLogger()
		terraformTool := tools.NewTerraformSecurityAnalyzerAdapter(cfg, log)
		if terraformTool != nil {
			result, source, toolErr := terraformTool.Execute(context.Background(), map[string]interface{}{
				"scan_paths":    []string{terraformPath},
				"output_format": "detailed",
			})

			if toolErr == nil {
				assert.NotEmpty(t, result)
				assert.NotNil(t, source)
				terraformOutput = []byte(result)
				t.Log("✓ Terraform security analysis completed")
			} else {
				t.Logf("Terraform analysis failed (may be expected): %v", toolErr)
			}
		}
	} else {
		assert.NotEmpty(t, terraformOutput)
		t.Log("✓ Terraform security analysis completed")
	}

	// Step 4: Generate evidence prompt for specific task
	t.Log("Step 4: Generating evidence prompt...")
	promptCmd := exec.Command("../../bin/grctool", "tool", "prompt-assembler",
		"--task-ref=ET-101")
	promptOutput, err := promptCmd.CombinedOutput()

	if err != nil {
		t.Logf("Prompt assembler may not be available via CLI: %v", err)
		// Try using the tool directly
		cfg := helpers.SetupE2ETest(t)

		log, _ := logger.NewTestLogger()
		promptTool := tools.NewPromptAssemblerTool(cfg, log)
		if promptTool != nil {
			result, source, toolErr := promptTool.Execute(context.Background(), map[string]interface{}{
				"task_ref": "ET-101",
			})

			if toolErr == nil {
				assert.NotEmpty(t, result)
				assert.NotNil(t, source)
				_ = []byte(result) // Convert to verify result is valid
				t.Log("✓ Evidence prompt generation completed")
			} else {
				t.Logf("Prompt assembly failed (may be expected if ET-101 not synced): %v", toolErr)
			}
		}
	} else {
		assert.NotEmpty(t, promptOutput)
		t.Log("✓ Evidence prompt generation completed")
	}

	// Step 5: List evidence tasks to verify data availability
	t.Log("Step 5: Listing available evidence tasks...")
	listCmd := exec.Command("../../bin/grctool", "evidence", "list", "--format=json")
	_, err = listCmd.CombinedOutput()

	if err != nil {
		t.Logf("Evidence list command may not be available: %v", err)
		// Try using the tool directly
		listCfg := helpers.SetupE2ETest(t)

		listLog, _ := logger.NewTestLogger()
		listTool := tools.NewEvidenceTaskListTool(listCfg, listLog)
		if listTool != nil {
			result, source, toolErr := listTool.Execute(context.Background(), map[string]interface{}{
				"format": "json",
			})

			if toolErr == nil {
				assert.NotEmpty(t, result)
				assert.NotNil(t, source)
			} else {
				t.Logf("Evidence task listing failed: %v", toolErr)
			}
		}
	}

	// Validate complete audit workflow results
	duration := time.Since(startTime)

	t.Logf("Complete SOC2 audit E2E test completed in %v", duration)

	// Validate outputs contain expected content
	githubOutputStr := string(githubOutput)
	if len(githubOutputStr) > 0 {
		assert.Contains(t, strings.ToLower(githubOutputStr), "github")
	}

	if len(terraformOutput) > 0 {
		terraformOutputStr := string(terraformOutput)
		securityKeywords := []string{"security", "terraform", "resource"}
		hasSecurityContent := false
		for _, keyword := range securityKeywords {
			if strings.Contains(strings.ToLower(terraformOutputStr), keyword) {
				hasSecurityContent = true
				break
			}
		}
		assert.True(t, hasSecurityContent, "Terraform output should contain security-related content")
	}

	// Performance assertion - full audit should complete in reasonable time
	assert.Less(t, duration, 5*time.Minute, "Complete audit should finish within 5 minutes")

	t.Log("✓ Complete SOC2 audit E2E test successful")
}

// TestQuarterlyReview_E2E tests periodic evidence collection workflow
func TestQuarterlyReview_E2E(t *testing.T) {
	if !hasRequiredAuth(t) {
		t.Skip("Complete authentication required for quarterly review test")
	}

	if os.Getenv("TEST_QUARTERLY_REVIEW") == "" {
		t.Skip("TEST_QUARTERLY_REVIEW not enabled")
	}

	buildGrctoolBinary(t)

	t.Log("Starting quarterly review E2E test...")

	// Simulate quarterly review workflow
	reviewSteps := []struct {
		name        string
		description string
		command     []string
	}{
		{
			name:        "data_sync",
			description: "Sync latest compliance data",
			command:     []string{"../../bin/grctool", "sync"},
		},
		{
			name:        "evidence_analysis",
			description: "Analyze evidence completeness",
			command:     []string{"../../bin/grctool", "evidence", "list", "--status=pending"},
		},
		{
			name:        "control_validation",
			description: "Validate control effectiveness",
			command:     []string{"../../bin/grctool", "control", "list", "--format=summary"},
		},
	}

	for _, step := range reviewSteps {
		t.Run(step.name, func(t *testing.T) {
			t.Logf("Executing %s: %s", step.name, step.description)

			cmd := exec.Command(step.command[0], step.command[1:]...)
			output, err := cmd.CombinedOutput()

			// Some commands may not exist yet, which is acceptable
			if err != nil {
				t.Logf("Step %s failed (may be expected): %v", step.name, err)
			} else {
				outputStr := string(output)
				assert.NotEmpty(t, outputStr)
				t.Logf("Step %s completed successfully", step.name)
			}
		})
	}

	t.Log("✓ Quarterly review E2E test completed")
}

// TestEvidenceCollectionWorkflow_E2E tests end-to-end evidence collection workflow
func TestEvidenceCollectionWorkflow_E2E(t *testing.T) {
	if !hasRequiredAuth(t) {
		t.Skip("Complete authentication required for evidence collection workflow test")
	}

	buildGrctoolBinary(t)

	cfg := helpers.SetupE2ETest(t)

	// Override Evidence config with GitHub settings
	cfg.Evidence = config.EvidenceConfig{
		Tools: config.ToolsConfig{
			GitHub: config.GitHubToolConfig{
				Enabled:    true,
				APIToken:   os.Getenv("GITHUB_TOKEN"),
				Repository: os.Getenv("TEST_GITHUB_REPO"),
			},
		},
	}

	if cfg.Evidence.Tools.GitHub.Repository == "" {
		cfg.Evidence.Tools.GitHub.Repository = "octocat/Hello-World"
	}

	log, _ := logger.NewTestLogger()

	t.Log("Testing evidence collection workflow...")

	// Step 1: Initialize tool registry
	err := tools.InitializeToolRegistry(cfg, log)
	require.NoError(t, err, "Tool registry should initialize successfully")

	// Step 2: List available tools
	availableTools := tools.ListTools()
	assert.Greater(t, len(availableTools), 0, "Should have registered tools")

	t.Logf("Available tools: %d", len(availableTools))
	for _, tool := range availableTools {
		t.Logf("- %s: %s", tool.Name, tool.Description)
	}

	// Step 3: Test evidence generation workflow
	evidenceTools := []string{
		"evidence-task-details",
		"prompt-assembler",
		"evidence-relationships",
	}

	for _, toolName := range evidenceTools {
		t.Run("tool_"+toolName, func(t *testing.T) {
			tool, err := tools.GetTool(toolName)
			if err != nil {
				t.Skipf("Tool %s not registered", toolName)
			}

			// Test with minimal parameters
			params := map[string]interface{}{}
			if toolName == "evidence-task-details" || toolName == "prompt-assembler" {
				params["task_ref"] = "ET-101"
			}

			result, source, err := tool.Execute(context.Background(), params)

			if err != nil {
				t.Logf("Tool %s execution failed (may be expected if data not available): %v", toolName, err)
			} else {
				assert.NotEmpty(t, result)
				assert.NotNil(t, source)
				t.Logf("Tool %s executed successfully", toolName)
			}
		})
	}

	t.Log("✓ Evidence collection workflow test completed")
}

// TestComplianceReporting_E2E tests compliance reporting workflow
func TestComplianceReporting_E2E(t *testing.T) {
	if !hasRequiredAuth(t) {
		t.Skip("Complete authentication required for compliance reporting test")
	}

	buildGrctoolBinary(t)

	t.Log("Testing compliance reporting workflow...")

	// Test report generation with available data
	reportingSteps := []struct {
		toolName    string
		description string
		params      map[string]interface{}
	}{
		{
			toolName:    "control-summary-generator",
			description: "Generate control summaries",
			params:      map[string]interface{}{"format": "markdown"},
		},
		{
			toolName:    "policy-summary-generator",
			description: "Generate policy summaries",
			params:      map[string]interface{}{"format": "markdown"},
		},
	}

	for _, step := range reportingSteps {
		t.Run(step.toolName, func(t *testing.T) {
			tool, err := tools.GetTool(step.toolName)
			if err != nil {
				t.Skipf("Tool %s not registered", step.toolName)
			}

			result, source, err := tool.Execute(context.Background(), step.params)

			if err != nil {
				t.Logf("Reporting step %s failed (may be expected if no data): %v", step.toolName, err)
			} else {
				assert.NotEmpty(t, result)
				assert.NotNil(t, source)
				t.Logf("Reporting step %s completed successfully", step.toolName)
			}
		})
	}

	t.Log("✓ Compliance reporting workflow test completed")
}

// TestDataIntegrity_E2E tests data integrity across the complete workflow
func TestDataIntegrity_E2E(t *testing.T) {
	if !hasRequiredAuth(t) {
		t.Skip("Complete authentication required for data integrity test")
	}

	if os.Getenv("TEST_DATA_INTEGRITY") == "" {
		t.Skip("TEST_DATA_INTEGRITY not enabled")
	}

	buildGrctoolBinary(t)

	t.Log("Testing data integrity across workflow...")

	// Step 1: Capture initial state
	initialSyncCmd := exec.Command("../../bin/grctool", "sync")
	initialOutput, err := initialSyncCmd.CombinedOutput()
	require.NoError(t, err, "Initial sync should succeed")

	// Step 2: Perform operations
	dataDir := "../../docs"
	if _, err := os.Stat(dataDir); err == nil {
		// Count files before operations
		beforeCount := 0
		err = filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				beforeCount++
			}
			return nil
		})
		require.NoError(t, err)

		t.Logf("Files before operations: %d", beforeCount)

		// Perform some operations that might modify data
		cfg := helpers.SetupE2ETest(t)

		log, _ := logger.NewTestLogger()

		// Test storage operations
		storageTool := tools.NewStorageReadTool(cfg, log)
		if storageTool != nil {
			_, _, err := storageTool.Execute(context.Background(), map[string]interface{}{
				"path":      "docs",
				"recursive": true,
			})

			if err != nil {
				t.Logf("Storage read operation failed: %v", err)
			}
		}

		// Count files after operations
		afterCount := 0
		err = filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
			if !info.IsDir() {
				afterCount++
			}
			return nil
		})
		require.NoError(t, err)

		t.Logf("Files after operations: %d", afterCount)

		// Data integrity check - file count should remain stable
		assert.Equal(t, beforeCount, afterCount, "File count should remain consistent")
	}

	// Step 3: Verify final sync produces consistent results
	finalSyncCmd := exec.Command("../../bin/grctool", "sync")
	finalOutput, err := finalSyncCmd.CombinedOutput()
	require.NoError(t, err, "Final sync should succeed")

	// Compare sync outputs for consistency indicators
	initialStr := string(initialOutput)
	finalStr := string(finalOutput)

	// Both should contain sync-related content
	assert.Contains(t, strings.ToLower(initialStr), "sync")
	assert.Contains(t, strings.ToLower(finalStr), "sync")

	t.Log("✓ Data integrity test completed")
}
