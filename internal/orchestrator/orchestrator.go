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
	"fmt"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools"
)

// ToolOrchestrator manages the execution of tools for evidence collection
// in prompt-as-data mode where tools generate structured output for external AI consumption
type ToolOrchestrator struct {
	terraformTool         tools.Tool
	terraformSnippetsTool tools.Tool
	githubTool            tools.Tool
	googleDocsTool        tools.Tool
	logger                logger.Logger
	maxToolCalls          int
	toolCallCounter       int
}

// NewToolOrchestrator creates a new tool orchestrator
func NewToolOrchestrator(cfg *config.Config, log logger.Logger) *ToolOrchestrator {
	orchestrator := &ToolOrchestrator{
		logger:       log,
		maxToolCalls: cfg.Evidence.Generation.MaxToolCalls,
	}

	// Initialize tools based on configuration
	if cfg.Evidence.Tools.Terraform.Enabled {
		orchestrator.terraformTool = tools.NewTerraformTool(cfg, log)
		orchestrator.terraformSnippetsTool = tools.NewTerraformSnippetsAdapter(cfg, log)
	}
	if cfg.Evidence.Tools.GitHub.Enabled {
		orchestrator.githubTool = tools.NewGitHubTool(cfg, log)
	}
	if cfg.Evidence.Tools.GoogleDocs.Enabled {
		// Google Docs tool would be initialized here
		// orchestrator.googleDocsTool = tools.NewGoogleDocsTool(cfg, log)
		log.Debug("Google Docs tool enabled but not yet implemented")
	}

	return orchestrator
}

// GenerateDataPackage coordinates tool execution to generate structured data packages
// for external AI consumption (prompt-as-data pattern)
func (o *ToolOrchestrator) GenerateDataPackage(ctx context.Context, prompt *models.EvidencePrompt, requestedTools []string) (*models.EvidenceDataPackage, error) {
	o.logger.Info("Starting data package generation",
		logger.Field{Key: "task_id", Value: prompt.TaskID},
		logger.Field{Key: "requested_tools", Value: requestedTools},
	)
	o.toolCallCounter = 0

	// Track sources used
	var sourcesUsed []models.EvidenceSource
	var toolOutputs []models.ToolOutput

	// Execute requested tools in parallel
	for _, toolName := range requestedTools {
		if o.toolCallCounter >= o.maxToolCalls {
			o.logger.Warn("Maximum tool calls reached",
				logger.Field{Key: "max", Value: o.maxToolCalls},
				logger.Field{Key: "actual", Value: o.toolCallCounter},
			)
			break
		}

		o.toolCallCounter++
		o.logger.Info("Executing tool for data collection",
			logger.Field{Key: "tool_name", Value: toolName},
			logger.Field{Key: "call_number", Value: o.toolCallCounter},
		)

		result, source, err := o.executeToolForDataCollection(ctx, toolName, prompt)
		if err != nil {
			o.logger.Error("Tool execution failed",
				logger.Field{Key: "tool", Value: toolName},
				logger.Field{Key: "error", Value: err},
			)
			// Add error info to outputs
			toolOutputs = append(toolOutputs, models.ToolOutput{
				ToolName:  toolName,
				Result:    fmt.Sprintf("Error: %v", err),
				Success:   false,
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"error": err.Error(),
				},
			})
		} else {
			o.logger.Info("Tool execution successful",
				logger.Field{Key: "tool", Value: toolName},
				logger.Field{Key: "result_length", Value: len(result)},
				logger.Field{Key: "has_source", Value: source != nil},
			)

			// Add successful result
			toolOutputs = append(toolOutputs, models.ToolOutput{
				ToolName:  toolName,
				Result:    result,
				Success:   true,
				Timestamp: time.Now(),
				Metadata: map[string]interface{}{
					"result_length": len(result),
				},
			})

			// Track source if provided
			if source != nil {
				sourcesUsed = append(sourcesUsed, *source)
				o.logger.Debug("Source added",
					logger.Field{Key: "source_type", Value: source.Type},
					logger.Field{Key: "source_resource", Value: source.Resource},
					logger.Field{Key: "relevance", Value: source.Relevance},
				)
			}
		}
	}

	// Create data package for external AI consumption
	dataPackage := &models.EvidenceDataPackage{
		TaskID:      prompt.TaskID,
		GeneratedAt: time.Now(),
		Prompt:      prompt.PromptText,
		ToolOutputs: toolOutputs,
		Sources:     sourcesUsed,
		Metadata: map[string]interface{}{
			"tool_count":       len(requestedTools),
			"successful_tools": o.countSuccessfulTools(toolOutputs),
			"total_tool_calls": o.toolCallCounter,
			"generation_mode":  "prompt-as-data",
		},
	}

	o.logger.Info("Data package generation completed",
		logger.Field{Key: "task_id", Value: prompt.TaskID},
		logger.Field{Key: "tools_executed", Value: len(toolOutputs)},
		logger.Field{Key: "sources_count", Value: len(sourcesUsed)},
		logger.Field{Key: "total_tool_calls", Value: o.toolCallCounter},
	)

	return dataPackage, nil
}

// getAvailableToolNames returns the list of available tool names
func (o *ToolOrchestrator) GetAvailableToolNames() []string {
	var toolNames []string

	if o.terraformTool != nil {
		toolNames = append(toolNames, o.terraformTool.Name())
	}
	if o.terraformSnippetsTool != nil {
		toolNames = append(toolNames, o.terraformSnippetsTool.Name())
	}
	if o.githubTool != nil {
		toolNames = append(toolNames, o.githubTool.Name())
	}
	if o.googleDocsTool != nil {
		toolNames = append(toolNames, o.googleDocsTool.Name())
	}

	return toolNames
}

// countSuccessfulTools counts successful tool executions
func (o *ToolOrchestrator) countSuccessfulTools(outputs []models.ToolOutput) int {
	count := 0
	for _, output := range outputs {
		if output.Success {
			count++
		}
	}
	return count
}

// executeToolForDataCollection executes a specific tool for data collection
func (o *ToolOrchestrator) executeToolForDataCollection(ctx context.Context, toolName string, prompt *models.EvidencePrompt) (string, *models.EvidenceSource, error) {
	o.logger.Debug("Executing tool for data collection",
		logger.Field{Key: "name", Value: toolName},
		logger.Field{Key: "task_id", Value: prompt.TaskID},
	)

	// Create basic parameters for tool execution
	params := map[string]interface{}{
		"task_id": prompt.TaskID,
		"prompt":  prompt.PromptText,
	}

	switch toolName {
	case "terraform_scanner", "terraform":
		if o.terraformTool == nil {
			return "", nil, fmt.Errorf("terraform tool not enabled")
		}
		return o.terraformTool.Execute(ctx, params)

	case "terraform_snippets":
		if o.terraformSnippetsTool == nil {
			return "", nil, fmt.Errorf("terraform snippets tool not enabled")
		}
		return o.terraformSnippetsTool.Execute(ctx, params)

	case "github_searcher", "github":
		if o.githubTool == nil {
			return "", nil, fmt.Errorf("github tool not enabled")
		}
		return o.githubTool.Execute(ctx, params)

	case "google_docs_reader", "docs":
		if o.googleDocsTool == nil {
			return "", nil, fmt.Errorf("google docs tool not enabled")
		}
		return o.googleDocsTool.Execute(ctx, params)

	default:
		return "", nil, fmt.Errorf("unknown tool: %s", toolName)
	}
}
