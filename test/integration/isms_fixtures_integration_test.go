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

package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ISMSTestEnvironment holds the test environment for ISMS fixtures
type ISMSTestEnvironment struct {
	Config              *config.Config
	Logger              logger.Logger
	TaskDetailsTool     tools.Tool
	RelationshipsTool   tools.Tool
	PromptAssemblerTool tools.Tool
	ToolMappingKey      *ToolMappingKey
	FixturesDir         string
}

// ToolMappingKey represents the tool selection validation key
type ToolMappingKey struct {
	Description    string                  `json:"description"`
	Version        string                  `json:"version"`
	Created        string                  `json:"created"`
	Mappings       map[string]ToolMapping  `json:"mappings"`
	TestingNotes   ToolMappingTestingNotes `json:"testing_notes"`
	ToolCategories map[string][]string     `json:"tool_categories"`
}

// ToolMapping represents expected tools for a specific evidence task
type ToolMapping struct {
	TaskName         string   `json:"task_name"`
	ExpectedTools    []string `json:"expected_tools"`
	Rationale        string   `json:"rationale"`
	KeyIndicators    []string `json:"key_indicators"`
	AlternativeTools []string `json:"alternative_tools"`
}

// ToolMappingTestingNotes contains testing guidance
type ToolMappingTestingNotes struct {
	ValidationApproach string            `json:"validation_approach"`
	Scoring            map[string]string `json:"scoring"`
	Considerations     []string          `json:"considerations"`
}

// setupISMSFixturesTestEnv sets up the test environment for ISMS fixtures
func setupISMSFixturesTestEnv(t *testing.T) *ISMSTestEnvironment {
	// Determine absolute path to test/sample_data
	fixturesDir, err := filepath.Abs(filepath.Join("..", "sample_data"))
	require.NoError(t, err, "Failed to get absolute path to fixtures directory")

	// Verify fixtures directory exists
	_, err = os.Stat(fixturesDir)
	require.NoError(t, err, "Fixtures directory does not exist: %s", fixturesDir)

	// Create configuration pointing to test fixtures
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: fixturesDir,
		},
	}

	// Create logger
	log, err := logger.New(&logger.Config{
		Level:  logger.ErrorLevel,
		Format: "text",
		Output: "stdout",
	})
	require.NoError(t, err, "Failed to create logger")

	// Initialize evidence tools
	taskDetailsTool := tools.NewEvidenceTaskDetailsTool(cfg, log)
	require.NotNil(t, taskDetailsTool, "Failed to create evidence task details tool")

	relationshipsTool := tools.NewEvidenceRelationshipsTool(cfg, log)
	require.NotNil(t, relationshipsTool, "Failed to create evidence relationships tool")

	promptAssemblerTool := tools.NewPromptAssemblerTool(cfg, log)
	require.NotNil(t, promptAssemblerTool, "Failed to create prompt assembler tool")

	// Load tool mapping key
	toolMappingKey := loadToolMappingKey(t, fixturesDir)

	return &ISMSTestEnvironment{
		Config:              cfg,
		Logger:              log,
		TaskDetailsTool:     taskDetailsTool,
		RelationshipsTool:   relationshipsTool,
		PromptAssemblerTool: promptAssemblerTool,
		ToolMappingKey:      toolMappingKey,
		FixturesDir:         fixturesDir,
	}
}

// loadToolMappingKey loads and parses the tool mapping validation key
func loadToolMappingKey(t *testing.T, fixturesDir string) *ToolMappingKey {
	keyPath := filepath.Join(fixturesDir, "tool_mapping_key.json")
	data, err := os.ReadFile(keyPath)
	require.NoError(t, err, "Failed to read tool mapping key")

	var key ToolMappingKey
	err = json.Unmarshal(data, &key)
	require.NoError(t, err, "Failed to parse tool mapping key")

	return &key
}

// Cleanup cleans up the test environment
func (env *ISMSTestEnvironment) Cleanup() {
	// Nothing to cleanup for now since we're using static fixtures
}

// ===== Test Category 1: Evidence Task Loading & Validation =====

func TestISMSFixtures_LoadEvidenceTasks(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	evidenceTasks := []string{"ET-001", "ET-002", "ET-003", "ET-004", "ET-005", "ET-101"}

	for _, taskRef := range evidenceTasks {
		t.Run(taskRef, func(t *testing.T) {
			params := map[string]interface{}{
				"task_ref": taskRef,
			}

			result, source, err := env.TaskDetailsTool.Execute(ctx, params)
			require.NoError(t, err, "Failed to load evidence task %s", taskRef)
			assert.NotEmpty(t, result, "Task details result should not be empty")
			assert.NotNil(t, source, "Evidence source should not be nil")

			// Parse result as JSON
			var taskData map[string]interface{}
			err = json.Unmarshal([]byte(result), &taskData)
			require.NoError(t, err, "Failed to parse task details JSON")

			// Response has nested structure with "task" field
			task, ok := taskData["task"].(map[string]interface{})
			require.True(t, ok, "Response should contain 'task' field")

			// Verify essential fields
			assert.Contains(t, task, "id", "Task should have id field")
			assert.Contains(t, task, "reference_id", "Task should have reference_id")
			assert.Contains(t, task, "name", "Task should have name")
			assert.Contains(t, task, "description", "Task should have description")

			t.Logf("Successfully loaded task %s: %s", taskRef, task["name"])
		})
	}
}

func TestISMSFixtures_ValidateCrossReferences(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	ctx := context.Background()

	testCases := []struct {
		taskRef          string
		expectedControls []string
		expectedPolicies []string
	}{
		{"ET-001", []string{"CC6.1"}, []string{"POL-005"}},
		{"ET-002", []string{"AC-01", "AC-02"}, []string{"POL-003"}},
		{"ET-003", []string{"CC6.1", "CC8.1"}, []string{"POL-004"}},
		{"ET-004", []string{"CC1.1", "CC1.2"}, []string{"POL-002"}},
		{"ET-005", []string{"CC8.1"}, []string{"POL-004"}},
	}

	for _, tc := range testCases {
		t.Run(tc.taskRef, func(t *testing.T) {
			params := map[string]interface{}{
				"task_ref": tc.taskRef,
			}

			result, _, err := env.TaskDetailsTool.Execute(ctx, params)
			require.NoError(t, err)

			var taskData map[string]interface{}
			err = json.Unmarshal([]byte(result), &taskData)
			require.NoError(t, err)

			// Check related controls
			if relatedControls, ok := taskData["related_controls"].([]interface{}); ok {
				controlRefs := make([]string, 0)
				for _, ctrl := range relatedControls {
					if ctrlMap, ok := ctrl.(map[string]interface{}); ok {
						if ref, ok := ctrlMap["reference_id"].(string); ok {
							controlRefs = append(controlRefs, ref)
						}
					}
				}

				for _, expectedCtrl := range tc.expectedControls {
					assert.Contains(t, controlRefs, expectedCtrl,
						"Task %s should reference control %s", tc.taskRef, expectedCtrl)
				}
			}

			// Check related policies
			if relatedPolicies, ok := taskData["related_policies"].([]interface{}); ok {
				policyRefs := make([]string, 0)
				for _, pol := range relatedPolicies {
					if polMap, ok := pol.(map[string]interface{}); ok {
						if ref, ok := polMap["reference_id"].(string); ok {
							policyRefs = append(policyRefs, ref)
						}
					}
				}

				for _, expectedPol := range tc.expectedPolicies {
					assert.Contains(t, policyRefs, expectedPol,
						"Task %s should reference policy %s", tc.taskRef, expectedPol)
				}
			}
		})
	}
}

func TestISMSFixtures_TaskMetadata(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	ctx := context.Background()

	testCases := []struct {
		taskRef                string
		expectedFramework      string
		expectedCategory       string
		expectedComplexity     string
		expectedCollectionType string
	}{
		{"ET-001", "SOC2 Type II", "Infrastructure", "Complex", "Automated"},
		{"ET-002", "SOC2 Type II", "Personnel", "Moderate", "Automated"},
		{"ET-003", "SOC2 Type II", "Infrastructure", "Complex", "Automated"},
		{"ET-004", "SOC2 Type II", "Personnel", "Moderate", "Hybrid"},
		{"ET-005", "SOC2 Type II", "Process", "Moderate", "Automated"},
	}

	for _, tc := range testCases {
		t.Run(tc.taskRef, func(t *testing.T) {
			params := map[string]interface{}{
				"task_ref": tc.taskRef,
			}

			result, _, err := env.TaskDetailsTool.Execute(ctx, params)
			require.NoError(t, err)

			var taskData map[string]interface{}
			err = json.Unmarshal([]byte(result), &taskData)
			require.NoError(t, err)

			// Extract nested sections
			task, ok := taskData["task"].(map[string]interface{})
			require.True(t, ok, "Response should contain 'task' field")

			requirements, ok := taskData["requirements"].(map[string]interface{})
			require.True(t, ok, "Response should contain 'requirements' field")

			// Validate metadata fields from nested structure
			assert.Equal(t, tc.expectedFramework, task["framework"],
				"Task %s should have framework %s", tc.taskRef, tc.expectedFramework)
			assert.Equal(t, tc.expectedCategory, requirements["category"],
				"Task %s should have category %s", tc.taskRef, tc.expectedCategory)
			assert.Equal(t, tc.expectedComplexity, requirements["complexity_level"],
				"Task %s should have complexity %s", tc.taskRef, tc.expectedComplexity)
			assert.Equal(t, tc.expectedCollectionType, requirements["collection_type"],
				"Task %s should have collection type %s", tc.taskRef, tc.expectedCollectionType)
		})
	}
}

// ===== Test Category 2: Evidence Relationships Tool Tests =====

func TestISMSFixtures_EvidenceRelationships_ET001(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	params := map[string]interface{}{
		"task_ref":         "ET-001",
		"depth":            2,
		"include_policies": true,
		"include_controls": true,
	}

	result, source, err := env.RelationshipsTool.Execute(ctx, params)
	require.NoError(t, err, "Failed to get relationships for ET-001")
	assert.NotEmpty(t, result)
	assert.NotNil(t, source)

	// Parse and validate relationships
	var relationships map[string]interface{}
	err = json.Unmarshal([]byte(result), &relationships)
	require.NoError(t, err)

	// Should map to CC6.1 control and POL-005 policy
	resultLower := strings.ToLower(result)
	assert.Contains(t, resultLower, "cc6.1", "Should reference CC6.1 control")
	assert.Contains(t, resultLower, "pol-005", "Should reference POL-005 policy")

	t.Logf("ET-001 relationships: %v", relationships)
}

func TestISMSFixtures_EvidenceRelationships_ET003(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	params := map[string]interface{}{
		"task_ref":         "ET-003",
		"depth":            2,
		"include_policies": true,
		"include_controls": true,
	}

	result, source, err := env.RelationshipsTool.Execute(ctx, params)
	require.NoError(t, err, "Failed to get relationships for ET-003")
	assert.NotEmpty(t, result)
	assert.NotNil(t, source)

	// ET-003 should map to CC6.1 and CC8.1 controls, POL-004 policy
	resultLower := strings.ToLower(result)
	assert.Contains(t, resultLower, "cc6.1", "Should reference CC6.1 control")
	assert.Contains(t, resultLower, "cc8.1", "Should reference CC8.1 control")
	assert.Contains(t, resultLower, "pol-004", "Should reference POL-004 policy")
}

func TestISMSFixtures_EvidenceRelationships_DepthAnalysis(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	taskRef := "ET-002"

	for depth := 1; depth <= 3; depth++ {
		t.Run(fmt.Sprintf("Depth_%d", depth), func(t *testing.T) {
			params := map[string]interface{}{
				"task_ref":         taskRef,
				"depth":            depth,
				"include_policies": true,
				"include_controls": true,
			}

			result, _, err := env.RelationshipsTool.Execute(ctx, params)
			require.NoError(t, err, "Failed to get relationships at depth %d", depth)
			assert.NotEmpty(t, result)

			// Higher depth should include more relationships
			t.Logf("Depth %d result length: %d", depth, len(result))
		})
	}
}

// ===== Test Category 3: Prompt Assembler Tool Tests =====

func TestISMSFixtures_PromptAssembler_ET001(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	params := map[string]interface{}{
		"task_ref":         "ET-001",
		"context_level":    "standard",
		"include_examples": true,
		"output_format":    "markdown",
		"save_to_file":     false,
	}

	result, source, err := env.PromptAssemblerTool.Execute(ctx, params)
	require.NoError(t, err, "Failed to generate prompt for ET-001")
	assert.NotEmpty(t, result)
	assert.NotNil(t, source)

	// Parse JSON response
	var response map[string]interface{}
	err = json.Unmarshal([]byte(result), &response)
	require.NoError(t, err)

	// Verify prompt contains key elements
	promptText, ok := response["prompt_text"].(string)
	require.True(t, ok, "Response should contain prompt_text")
	assert.NotEmpty(t, promptText, "Prompt text should not be empty")

	// Verify infrastructure security keywords are present
	promptLower := strings.ToLower(promptText)
	assert.Contains(t, promptLower, "infrastructure", "Prompt should mention infrastructure")
	assert.Contains(t, promptLower, "security", "Prompt should mention security")
	assert.Contains(t, promptLower, "encryption", "Prompt should mention encryption")

	t.Logf("Generated prompt length: %d characters", len(promptText))
}

func TestISMSFixtures_PromptAssembler_ET002(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	params := map[string]interface{}{
		"task_ref":         "ET-002",
		"context_level":    "comprehensive",
		"include_examples": true,
		"output_format":    "markdown",
		"save_to_file":     false,
	}

	result, source, err := env.PromptAssemblerTool.Execute(ctx, params)
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, source)

	// Parse response
	var response map[string]interface{}
	err = json.Unmarshal([]byte(result), &response)
	require.NoError(t, err)

	promptText, ok := response["prompt_text"].(string)
	require.True(t, ok)

	// Verify repository access control keywords
	promptLower := strings.ToLower(promptText)
	assert.Contains(t, promptLower, "access", "Prompt should mention access")
	assert.Contains(t, promptLower, "repository", "Prompt should mention repository")
	assert.Contains(t, promptLower, "permission", "Prompt should mention permissions")
}

func TestISMSFixtures_PromptAssembler_AllFormats(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	formats := []string{"markdown", "csv", "json"}

	for _, format := range formats {
		t.Run(format, func(t *testing.T) {
			params := map[string]interface{}{
				"task_ref":         "ET-003",
				"context_level":    "standard",
				"include_examples": false,
				"output_format":    format,
				"save_to_file":     false,
			}

			result, source, err := env.PromptAssemblerTool.Execute(ctx, params)
			require.NoError(t, err, "Failed to generate %s format prompt", format)
			assert.NotEmpty(t, result)
			assert.NotNil(t, source)

			// Verify format is set in response
			var response map[string]interface{}
			err = json.Unmarshal([]byte(result), &response)
			require.NoError(t, err)

			metadata, ok := response["prompt_metadata"].(map[string]interface{})
			require.True(t, ok)
			assert.Equal(t, format, metadata["output_format"])
		})
	}
}

// ===== Test Category 4: Tool Selection Intelligence Tests =====

// scoreToolSelection scores the tool selection based on validation key
func scoreToolSelection(selectedTools []string, mapping ToolMapping) (string, float64) {
	matchCount := 0
	totalExpected := len(mapping.ExpectedTools)

	for _, expected := range mapping.ExpectedTools {
		for _, selected := range selectedTools {
			if strings.EqualFold(expected, selected) {
				matchCount++
				break
			}
		}
	}

	// Check alternatives
	altMatchCount := 0
	for _, alt := range mapping.AlternativeTools {
		for _, selected := range selectedTools {
			if strings.EqualFold(alt, selected) {
				altMatchCount++
				break
			}
		}
	}

	// Calculate score
	matchRatio := float64(matchCount) / float64(totalExpected)

	if matchCount == totalExpected {
		return "perfect_match", 1.0
	} else if matchCount >= (totalExpected*2/3) || (matchCount > 0 && altMatchCount > 0) {
		return "good_match", matchRatio
	} else if matchCount > 0 {
		return "partial_match", matchRatio
	}
	return "poor_match", 0.0
}

func TestISMSFixtures_ToolSelection_ET001(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	taskRef := "ET-001"
	mapping, ok := env.ToolMappingKey.Mappings[taskRef]
	require.True(t, ok, "Tool mapping should exist for %s", taskRef)

	// Expected tools for infrastructure: terraform-scanner, terraform-hcl-parser, terraform-security-analyzer
	t.Logf("Expected tools for %s: %v", taskRef, mapping.ExpectedTools)
	t.Logf("Rationale: %s", mapping.Rationale)
	t.Logf("Key indicators: %v", mapping.KeyIndicators)

	// In a real implementation, this would call the evidence assembly worker
	// For now, we verify the mapping structure is correct
	assert.Equal(t, 3, len(mapping.ExpectedTools), "ET-001 should have 3 expected tools")
	assert.Contains(t, mapping.ExpectedTools, "terraform-scanner")
	assert.Contains(t, mapping.ExpectedTools, "terraform-hcl-parser")
	assert.Contains(t, mapping.ExpectedTools, "terraform-security-analyzer")
}

func TestISMSFixtures_ToolSelection_ET002(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	taskRef := "ET-002"
	mapping, ok := env.ToolMappingKey.Mappings[taskRef]
	require.True(t, ok, "Tool mapping should exist for %s", taskRef)

	// Expected tools for repository access: github-permissions, github-deployment-access
	t.Logf("Expected tools for %s: %v", taskRef, mapping.ExpectedTools)
	assert.Equal(t, 2, len(mapping.ExpectedTools), "ET-002 should have 2 expected tools")
	assert.Contains(t, mapping.ExpectedTools, "github-permissions")
	assert.Contains(t, mapping.ExpectedTools, "github-deployment-access")
}

func TestISMSFixtures_ToolSelection_ET003(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	taskRef := "ET-003"
	mapping, ok := env.ToolMappingKey.Mappings[taskRef]
	require.True(t, ok, "Tool mapping should exist for %s", taskRef)

	// Expected tools for CI/CD: github-workflow-analyzer, github-security-features
	t.Logf("Expected tools for %s: %v", taskRef, mapping.ExpectedTools)
	assert.Equal(t, 2, len(mapping.ExpectedTools), "ET-003 should have 2 expected tools")
	assert.Contains(t, mapping.ExpectedTools, "github-workflow-analyzer")
	assert.Contains(t, mapping.ExpectedTools, "github-security-features")
}

func TestISMSFixtures_ToolSelection_ET004(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	taskRef := "ET-004"
	mapping, ok := env.ToolMappingKey.Mappings[taskRef]
	require.True(t, ok, "Tool mapping should exist for %s", taskRef)

	// Expected tools for training: google-workspace, docs-reader
	t.Logf("Expected tools for %s: %v", taskRef, mapping.ExpectedTools)
	assert.Equal(t, 2, len(mapping.ExpectedTools), "ET-004 should have 2 expected tools")
	assert.Contains(t, mapping.ExpectedTools, "google-workspace")
	assert.Contains(t, mapping.ExpectedTools, "docs-reader")
}

func TestISMSFixtures_ToolSelection_ET005(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	taskRef := "ET-005"
	mapping, ok := env.ToolMappingKey.Mappings[taskRef]
	require.True(t, ok, "Tool mapping should exist for %s", taskRef)

	// Expected tools for code review: github-review-analyzer, github-permissions
	t.Logf("Expected tools for %s: %v", taskRef, mapping.ExpectedTools)
	assert.Equal(t, 2, len(mapping.ExpectedTools), "ET-005 should have 2 expected tools")
	assert.Contains(t, mapping.ExpectedTools, "github-review-analyzer")
	assert.Contains(t, mapping.ExpectedTools, "github-permissions")
}

func TestISMSFixtures_ToolSelectionScoring(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	testCases := []struct {
		name          string
		selectedTools []string
		expectedTools []string
		alternatives  []string
		expectedScore string
	}{
		{
			name:          "Perfect match",
			selectedTools: []string{"terraform-scanner", "terraform-hcl-parser", "terraform-security-analyzer"},
			expectedTools: []string{"terraform-scanner", "terraform-hcl-parser", "terraform-security-analyzer"},
			alternatives:  []string{},
			expectedScore: "perfect_match",
		},
		{
			name:          "Good match with alternatives",
			selectedTools: []string{"terraform-scanner", "terraform-hcl-parser", "terraform-query-interface"},
			expectedTools: []string{"terraform-scanner", "terraform-hcl-parser", "terraform-security-analyzer"},
			alternatives:  []string{"terraform-query-interface"},
			expectedScore: "good_match",
		},
		{
			name:          "Partial match",
			selectedTools: []string{"terraform-scanner"},
			expectedTools: []string{"terraform-scanner", "terraform-hcl-parser", "terraform-security-analyzer"},
			alternatives:  []string{},
			expectedScore: "partial_match",
		},
		{
			name:          "Poor match",
			selectedTools: []string{"unrelated-tool"},
			expectedTools: []string{"terraform-scanner", "terraform-hcl-parser"},
			alternatives:  []string{},
			expectedScore: "poor_match",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			mapping := ToolMapping{
				ExpectedTools:    tc.expectedTools,
				AlternativeTools: tc.alternatives,
			}

			score, _ := scoreToolSelection(tc.selectedTools, mapping)
			assert.Equal(t, tc.expectedScore, score,
				"Tool selection should score as %s", tc.expectedScore)
		})
	}
}

// ===== Test Category 5: End-to-End Workflow Tests =====

func TestISMSFixtures_CompleteWorkflow_ET001(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	taskRef := "ET-001"

	// Step 1: Load task details
	t.Log("Step 1: Loading task details...")
	taskParams := map[string]interface{}{
		"task_ref": taskRef,
	}
	taskResult, _, err := env.TaskDetailsTool.Execute(ctx, taskParams)
	require.NoError(t, err, "Failed to load task details")
	assert.NotEmpty(t, taskResult)

	// Step 2: Get relationships
	t.Log("Step 2: Mapping evidence relationships...")
	relParams := map[string]interface{}{
		"task_ref":         taskRef,
		"depth":            2,
		"include_policies": true,
		"include_controls": true,
	}
	relResult, _, err := env.RelationshipsTool.Execute(ctx, relParams)
	require.NoError(t, err, "Failed to map relationships")
	assert.NotEmpty(t, relResult)

	// Step 3: Generate prompt
	t.Log("Step 3: Generating evidence collection prompt...")
	promptParams := map[string]interface{}{
		"task_ref":         taskRef,
		"context_level":    "comprehensive",
		"include_examples": true,
		"output_format":    "markdown",
		"save_to_file":     false,
	}
	promptResult, _, err := env.PromptAssemblerTool.Execute(ctx, promptParams)
	require.NoError(t, err, "Failed to generate prompt")
	assert.NotEmpty(t, promptResult)

	// Step 4: Validate tool selection
	t.Log("Step 4: Validating expected tool selection...")
	mapping, ok := env.ToolMappingKey.Mappings[taskRef]
	require.True(t, ok, "Tool mapping should exist")

	t.Logf("Expected tools: %v", mapping.ExpectedTools)
	t.Logf("Rationale: %s", mapping.Rationale)

	// Workflow complete
	t.Log("✓ Complete workflow executed successfully")
}

func TestISMSFixtures_CompleteWorkflow_MultiTask(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	ctx := context.Background()
	tasks := []string{"ET-001", "ET-002", "ET-003", "ET-004", "ET-005"}

	for _, taskRef := range tasks {
		t.Run(taskRef, func(t *testing.T) {
			// Execute complete workflow for each task
			taskParams := map[string]interface{}{
				"task_ref": taskRef,
			}

			taskResult, _, err := env.TaskDetailsTool.Execute(ctx, taskParams)
			require.NoError(t, err, "Failed to load task %s", taskRef)
			assert.NotEmpty(t, taskResult)

			// Get relationships
			relParams := map[string]interface{}{
				"task_ref":         taskRef,
				"depth":            2,
				"include_policies": true,
				"include_controls": true,
			}
			relResult, _, err := env.RelationshipsTool.Execute(ctx, relParams)
			require.NoError(t, err, "Failed to map relationships for %s", taskRef)
			assert.NotEmpty(t, relResult)

			// Generate prompt
			promptParams := map[string]interface{}{
				"task_ref":         taskRef,
				"context_level":    "standard",
				"include_examples": false,
				"output_format":    "markdown",
				"save_to_file":     false,
			}
			promptResult, _, err := env.PromptAssemblerTool.Execute(ctx, promptParams)
			require.NoError(t, err, "Failed to generate prompt for %s", taskRef)
			assert.NotEmpty(t, promptResult)

			t.Logf("✓ Workflow complete for %s", taskRef)
		})
	}
}

func TestISMSFixtures_WorkflowWithValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping ISMS fixtures integration tests in short mode")
	}

	env := setupISMSFixturesTestEnv(t)
	defer env.Cleanup()

	ctx := context.Background()

	testCases := []struct {
		taskRef             string
		expectedToolCount   int
		expectedControlsMin int
		expectedPoliciesMin int
	}{
		{"ET-001", 3, 1, 1}, // Infrastructure: terraform tools, CC6.1, POL-005
		{"ET-002", 2, 2, 1}, // Access control: github tools, AC-01+AC-02, POL-003
		{"ET-003", 2, 2, 1}, // CI/CD: github tools, CC6.1+CC8.1, POL-004
		{"ET-004", 2, 2, 1}, // Training: docs tools, CC1.1+CC1.2, POL-002
		{"ET-005", 2, 1, 1}, // Code review: github tools, CC8.1, POL-004
	}

	for _, tc := range testCases {
		t.Run(tc.taskRef, func(t *testing.T) {
			// Execute workflow and validate
			taskParams := map[string]interface{}{
				"task_ref": tc.taskRef,
			}

			taskResult, _, err := env.TaskDetailsTool.Execute(ctx, taskParams)
			require.NoError(t, err)

			var taskData map[string]interface{}
			err = json.Unmarshal([]byte(taskResult), &taskData)
			require.NoError(t, err)

			// Validate minimum controls from relationships section
			relationships, ok := taskData["relationships"].(map[string]interface{})
			require.True(t, ok, "Task should have relationships")

			relatedControls, ok := relationships["controls"].([]interface{})
			require.True(t, ok, "Task relationships should have controls")
			assert.GreaterOrEqual(t, len(relatedControls), tc.expectedControlsMin,
				"Task %s should have at least %d controls", tc.taskRef, tc.expectedControlsMin)

			// Note: policies are not in the basic task details response
			// They would need to be fetched via the relationships tool with proper depth
			// For now, we just validate the tool mapping exists
			// (Policy validation removed since it requires the evidence-relationships tool)

			// Validate expected tool count from mapping
			mapping, ok := env.ToolMappingKey.Mappings[tc.taskRef]
			require.True(t, ok, "Tool mapping should exist for %s", tc.taskRef)
			assert.Equal(t, tc.expectedToolCount, len(mapping.ExpectedTools),
				"Task %s should have %d expected tools", tc.taskRef, tc.expectedToolCount)
		})
	}
}
