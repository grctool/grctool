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

package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Test NewToolOrchestrator initialization
func TestNewToolOrchestrator(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 10,
			},
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled:   true,
					ScanPaths: []string{"/tmp/terraform"},
				},
				GitHub: config.GitHubToolConfig{
					Enabled: true,
				},
				GoogleDocs: config.GoogleDocsToolConfig{
					Enabled: false,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()

	orchestrator := NewToolOrchestrator(cfg, log)

	assert.NotNil(t, orchestrator)
	assert.Equal(t, 10, orchestrator.maxToolCalls)
	assert.Equal(t, 0, orchestrator.toolCallCounter)
	assert.NotNil(t, orchestrator.logger)
	assert.NotNil(t, orchestrator.terraformTool)
	assert.NotNil(t, orchestrator.terraformSnippetsTool)
	assert.NotNil(t, orchestrator.githubTool)
	assert.Nil(t, orchestrator.googleDocsTool) // Not yet implemented
}

// Test NewToolOrchestrator with all tools disabled
func TestNewToolOrchestrator_AllToolsDisabled(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 5,
			},
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled: false,
				},
				GitHub: config.GitHubToolConfig{
					Enabled: false,
				},
				GoogleDocs: config.GoogleDocsToolConfig{
					Enabled: false,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()

	orchestrator := NewToolOrchestrator(cfg, log)

	assert.NotNil(t, orchestrator)
	assert.Nil(t, orchestrator.terraformTool)
	assert.Nil(t, orchestrator.terraformSnippetsTool)
	assert.Nil(t, orchestrator.githubTool)
	assert.Nil(t, orchestrator.googleDocsTool)
}

// Test GetAvailableToolNames
func TestGetAvailableToolNames(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 10,
			},
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled:   true,
					ScanPaths: []string{"/tmp/terraform"},
				},
				GitHub: config.GitHubToolConfig{
					Enabled: true,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()

	orchestrator := NewToolOrchestrator(cfg, log)
	toolNames := orchestrator.GetAvailableToolNames()

	assert.NotEmpty(t, toolNames)
	// Should have terraform, terraform_snippets, and github tools
	assert.GreaterOrEqual(t, len(toolNames), 2)
}

// Test GetAvailableToolNames with no tools
func TestGetAvailableToolNames_NoTools(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 10,
			},
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled: false,
				},
				GitHub: config.GitHubToolConfig{
					Enabled: false,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()

	orchestrator := NewToolOrchestrator(cfg, log)
	toolNames := orchestrator.GetAvailableToolNames()

	assert.Empty(t, toolNames)
}

// Test countSuccessfulTools
func TestCountSuccessfulTools(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 10,
			},
		},
	}

	log, _ := logger.NewTestLogger()
	orchestrator := NewToolOrchestrator(cfg, log)

	outputs := []models.ToolOutput{
		{
			ToolName:  "tool1",
			Success:   true,
			Timestamp: time.Now(),
		},
		{
			ToolName:  "tool2",
			Success:   false,
			Timestamp: time.Now(),
		},
		{
			ToolName:  "tool3",
			Success:   true,
			Timestamp: time.Now(),
		},
	}

	count := orchestrator.countSuccessfulTools(outputs)
	assert.Equal(t, 2, count)
}

// Test countSuccessfulTools with empty slice
func TestCountSuccessfulTools_Empty(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 10,
			},
		},
	}

	log, _ := logger.NewTestLogger()
	orchestrator := NewToolOrchestrator(cfg, log)

	outputs := []models.ToolOutput{}
	count := orchestrator.countSuccessfulTools(outputs)
	assert.Equal(t, 0, count)
}

// Test executeToolForDataCollection with unknown tool
func TestExecuteToolForDataCollection_UnknownTool(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 10,
			},
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled: false,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()
	orchestrator := NewToolOrchestrator(cfg, log)

	ctx := context.Background()
	prompt := &models.EvidencePrompt{
		TaskID:     123,
		PromptText: "Test prompt",
	}

	result, source, err := orchestrator.executeToolForDataCollection(ctx, "unknown_tool", prompt)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown tool")
	assert.Empty(t, result)
	assert.Nil(t, source)
}

// Test executeToolForDataCollection with disabled tool
func TestExecuteToolForDataCollection_DisabledTool(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 10,
			},
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled: false,
				},
				GitHub: config.GitHubToolConfig{
					Enabled: false,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()
	orchestrator := NewToolOrchestrator(cfg, log)

	ctx := context.Background()
	prompt := &models.EvidencePrompt{
		TaskID:     123,
		PromptText: "Test prompt",
	}

	// Test terraform tool when disabled
	result, source, err := orchestrator.executeToolForDataCollection(ctx, "terraform", prompt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
	assert.Empty(t, result)
	assert.Nil(t, source)

	// Test github tool when disabled
	result, source, err = orchestrator.executeToolForDataCollection(ctx, "github", prompt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
	assert.Empty(t, result)
	assert.Nil(t, source)

	// Test google docs tool when disabled
	result, source, err = orchestrator.executeToolForDataCollection(ctx, "google_docs_reader", prompt)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not enabled")
	assert.Empty(t, result)
	assert.Nil(t, source)
}

// Test GenerateDataPackage structure
func TestGenerateDataPackage_Structure(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 10,
			},
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled: false,
				},
				GitHub: config.GitHubToolConfig{
					Enabled: false,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()
	orchestrator := NewToolOrchestrator(cfg, log)

	ctx := context.Background()
	prompt := &models.EvidencePrompt{
		TaskID:     123,
		PromptText: "Generate evidence for SOC2 compliance",
	}

	// Request tools that will fail (disabled)
	requestedTools := []string{"terraform", "github"}

	pkg, err := orchestrator.GenerateDataPackage(ctx, prompt, requestedTools)

	// Should not return error even if tools fail
	assert.NoError(t, err)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	require.NotNil(t, pkg)

	// Verify package structure
	assert.Equal(t, 123, pkg.TaskID)
	assert.Equal(t, prompt.PromptText, pkg.Prompt)
	assert.NotZero(t, pkg.GeneratedAt)
	assert.NotNil(t, pkg.ToolOutputs)
	// Sources can be nil or empty when tools fail
	assert.NotNil(t, pkg.Metadata)

	// All tools should have failed
	assert.Len(t, pkg.ToolOutputs, 2)
	for _, output := range pkg.ToolOutputs {
		assert.False(t, output.Success)
		assert.Contains(t, output.Result, "Error:")
	}

	// Verify metadata
	assert.Contains(t, pkg.Metadata, "tool_count")
	assert.Contains(t, pkg.Metadata, "successful_tools")
	assert.Contains(t, pkg.Metadata, "total_tool_calls")
	assert.Contains(t, pkg.Metadata, "generation_mode")
	assert.Equal(t, "prompt-as-data", pkg.Metadata["generation_mode"])
}

// Test GenerateDataPackage with max tool calls limit
func TestGenerateDataPackage_MaxToolCallsLimit(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 2, // Limit to 2 calls
			},
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled: false,
				},
				GitHub: config.GitHubToolConfig{
					Enabled: false,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()
	orchestrator := NewToolOrchestrator(cfg, log)

	ctx := context.Background()
	prompt := &models.EvidencePrompt{
		TaskID:     123,
		PromptText: "Test prompt",
	}

	// Request 5 tools but only 2 should be executed
	requestedTools := []string{"terraform", "github", "terraform_snippets", "docs", "unknown"}

	pkg, err := orchestrator.GenerateDataPackage(ctx, prompt, requestedTools)

	require.NoError(t, err)
	require.NotNil(t, pkg)

	// Only 2 tools should be executed due to max limit
	assert.LessOrEqual(t, len(pkg.ToolOutputs), 2)
	assert.Equal(t, 2, pkg.Metadata["total_tool_calls"])
}

// Test GenerateDataPackage with empty tool list
func TestGenerateDataPackage_EmptyToolList(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 10,
			},
		},
	}

	log, _ := logger.NewTestLogger()
	orchestrator := NewToolOrchestrator(cfg, log)

	ctx := context.Background()
	prompt := &models.EvidencePrompt{
		TaskID:     123,
		PromptText: "Test prompt",
	}

	requestedTools := []string{}

	pkg, err := orchestrator.GenerateDataPackage(ctx, prompt, requestedTools)

	require.NoError(t, err)
	require.NotNil(t, pkg)

	assert.Empty(t, pkg.ToolOutputs)
	assert.Empty(t, pkg.Sources)
	assert.Equal(t, 0, pkg.Metadata["total_tool_calls"])
}

// Test GenerateDataPackage with context cancellation
func TestGenerateDataPackage_ContextCancellation(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 10,
			},
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled: false,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()
	orchestrator := NewToolOrchestrator(cfg, log)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	prompt := &models.EvidencePrompt{
		TaskID:     123,
		PromptText: "Test prompt",
	}

	requestedTools := []string{"terraform"}

	// Should handle cancelled context gracefully
	pkg, err := orchestrator.GenerateDataPackage(ctx, prompt, requestedTools)

	// May or may not error depending on timing, but should not panic
	if err == nil {
		assert.NotNil(t, pkg)
	}
}

// Test orchestrator with different tool name aliases
func TestExecuteToolForDataCollection_Aliases(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 10,
			},
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled: false,
				},
				GitHub: config.GitHubToolConfig{
					Enabled: false,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()
	orchestrator := NewToolOrchestrator(cfg, log)

	ctx := context.Background()
	prompt := &models.EvidencePrompt{
		TaskID:     123,
		PromptText: "Test prompt",
	}

	testCases := []struct {
		name     string
		toolName string
		errMsg   string
	}{
		{
			name:     "terraform alias",
			toolName: "terraform",
			errMsg:   "not enabled",
		},
		{
			name:     "terraform_scanner alias",
			toolName: "terraform_scanner",
			errMsg:   "not enabled",
		},
		{
			name:     "github alias",
			toolName: "github",
			errMsg:   "not enabled",
		},
		{
			name:     "github_searcher alias",
			toolName: "github_searcher",
			errMsg:   "not enabled",
		},
		{
			name:     "docs alias",
			toolName: "docs",
			errMsg:   "not enabled",
		},
		{
			name:     "google_docs_reader alias",
			toolName: "google_docs_reader",
			errMsg:   "not enabled",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, err := orchestrator.executeToolForDataCollection(ctx, tc.toolName, prompt)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tc.errMsg)
		})
	}
}

// Test tool counter increment
func TestGenerateDataPackage_ToolCounterIncrement(t *testing.T) {
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				MaxToolCalls: 10,
			},
			Tools: config.ToolsConfig{
				Terraform: config.TerraformToolConfig{
					Enabled: false,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()
	orchestrator := NewToolOrchestrator(cfg, log)

	ctx := context.Background()
	prompt := &models.EvidencePrompt{
		TaskID:     123,
		PromptText: "Test prompt",
	}

	// Initial counter should be 0
	assert.Equal(t, 0, orchestrator.toolCallCounter)

	requestedTools := []string{"terraform", "terraform", "terraform"}

	pkg, err := orchestrator.GenerateDataPackage(ctx, prompt, requestedTools)

	require.NoError(t, err)
	require.NotNil(t, pkg)

	// Counter should increment for each tool call
	assert.Equal(t, 3, orchestrator.toolCallCounter)
	assert.Equal(t, 3, pkg.Metadata["total_tool_calls"])
}
