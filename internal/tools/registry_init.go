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

package tools

import (
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/templates"
	"github.com/grctool/grctool/internal/tools/github"
)

// InitializeToolRegistry registers all available tools with the global registry
func InitializeToolRegistry(cfg *config.Config, log logger.Logger) error {
	// Initialize singleton template manager
	if _, err := templates.GetSingleton(log); err != nil {
		log.Error("Failed to initialize singleton template manager", logger.Field{Key: "error", Value: err})
		return err
	}

	// Register existing tools (original implementations)
	if terraformTool := NewTerraformTool(cfg, log); terraformTool != nil {
		if err := RegisterTool(terraformTool); err != nil {
			log.Error("Failed to register terraform tool", logger.Field{Key: "error", Value: err})
		}
	}

	// TODO: Register refactored tools with pure functions and dependency injection
	// These constructors need to be implemented
	/*
		if terraformRefactoredTool := NewTerraformToolRefactored(cfg, log); terraformRefactoredTool != nil {
			if err := RegisterTool(terraformRefactoredTool); err != nil {
				log.Error("Failed to register terraform refactored tool", logger.Field{Key: "error", Value: err})
			} else {
				log.Debug("Registered terraform refactored tool")
			}
		}

		// Register refactored GitHub permissions tool
		if githubPermissionsRefactoredTool := NewGitHubPermissionsToolRefactored(cfg, log); githubPermissionsRefactoredTool != nil {
			if err := RegisterTool(githubPermissionsRefactoredTool); err != nil {
				log.Error("Failed to register github permissions refactored tool", logger.Field{Key: "error", Value: err})
			} else {
				log.Debug("Registered github permissions refactored tool")
			}
		}
	*/

	// Register consolidated GitHub tools
	if githubTool := github.NewGitHubAdapter(cfg, log); githubTool != nil {
		if err := RegisterTool(githubTool); err != nil {
			log.Error("Failed to register github tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered consolidated github tool")
		}
	}

	// Register evidence analysis tools
	if evidenceTaskDetailsTool := NewEvidenceTaskDetailsTool(cfg, log); evidenceTaskDetailsTool != nil {
		if err := RegisterTool(evidenceTaskDetailsTool); err != nil {
			log.Error("Failed to register evidence task details tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered evidence task details tool")
		}
	}

	if evidenceTaskListTool := NewEvidenceTaskListTool(cfg, log); evidenceTaskListTool != nil {
		if err := RegisterTool(evidenceTaskListTool); err != nil {
			log.Error("Failed to register evidence task list tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered evidence task list tool")
		}
	}

	if evidenceRelationshipsTool := NewEvidenceRelationshipsTool(cfg, log); evidenceRelationshipsTool != nil {
		if err := RegisterTool(evidenceRelationshipsTool); err != nil {
			log.Error("Failed to register evidence relationships tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered evidence relationships tool")
		}
	}

	if promptAssemblerTool := NewPromptAssemblerTool(cfg, log); promptAssemblerTool != nil {
		if err := RegisterTool(promptAssemblerTool); err != nil {
			log.Error("Failed to register prompt assembler tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered prompt assembler tool")
		}
	}

	if policySummaryTool := NewPolicySummaryGeneratorTool(cfg, log); policySummaryTool != nil {
		if err := RegisterTool(policySummaryTool); err != nil {
			log.Error("Failed to register policy summary generator tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered policy summary generator tool")
		}
	}

	if controlSummaryTool := NewControlSummaryGeneratorTool(cfg, log); controlSummaryTool != nil {
		if err := RegisterTool(controlSummaryTool); err != nil {
			log.Error("Failed to register control summary generator tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered control summary generator tool")
		}
	}

	// Register enhanced data source tools
	if terraformSecurityTool := NewTerraformSecurityAnalyzerAdapter(cfg, log); terraformSecurityTool != nil {
		if err := RegisterTool(terraformSecurityTool); err != nil {
			log.Error("Failed to register terraform security analyzer tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered terraform security analyzer tool")
		}
	}

	if terraformHCLTool := NewTerraformHCLParserAdapter(cfg, log); terraformHCLTool != nil {
		if err := RegisterTool(terraformHCLTool); err != nil {
			log.Error("Failed to register terraform HCL parser tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered terraform HCL parser tool")
		}
	}

	if terraformSnippetsTool := NewTerraformSnippetsAdapter(cfg, log); terraformSnippetsTool != nil {
		if err := RegisterTool(terraformSnippetsTool); err != nil {
			log.Error("Failed to register terraform snippets tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered terraform snippets tool")
		}
	}

	if terraformAtmosTool := NewTerraformAtmosAdapter(cfg, log); terraformAtmosTool != nil {
		if err := RegisterTool(terraformAtmosTool); err != nil {
			log.Error("Failed to register terraform atmos analyzer tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered terraform atmos analyzer tool")
		}
	}

	// Register Infrastructure Compliance Monitoring tools
	if terraformSecurityIndexerTool := NewTerraformSecurityIndexerAdapter(cfg, log); terraformSecurityIndexerTool != nil {
		if err := RegisterTool(terraformSecurityIndexerTool); err != nil {
			log.Error("Failed to register terraform security indexer tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered terraform security indexer tool")
		}
	}

	if terraformQueryInterfaceTool := NewTerraformQueryInterfaceAdapter(cfg, log); terraformQueryInterfaceTool != nil {
		if err := RegisterTool(terraformQueryInterfaceTool); err != nil {
			log.Error("Failed to register terraform query interface tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered terraform query interface tool")
		}
	}

	if githubEnhancedTool := github.NewGitHubEnhancedAdapter(cfg, log); githubEnhancedTool != nil {
		if err := RegisterTool(githubEnhancedTool); err != nil {
			log.Error("Failed to register github enhanced tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered consolidated github enhanced tool")
		}
	}

	// Register comprehensive GitHub API tools for SOC2 audit evidence
	if githubPermissionsTool := github.NewGitHubPermissionsAdapter(cfg, log); githubPermissionsTool != nil {
		if err := RegisterTool(githubPermissionsTool); err != nil {
			log.Error("Failed to register github permissions tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered consolidated github permissions tool")
		}
	}

	if githubDeploymentAccessTool := github.NewGitHubDeploymentAccessAdapter(cfg, log); githubDeploymentAccessTool != nil {
		if err := RegisterTool(githubDeploymentAccessTool); err != nil {
			log.Error("Failed to register github deployment access tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered consolidated github deployment access tool")
		}
	}

	if githubSecurityFeaturesTool := github.NewGitHubSecurityFeaturesAdapter(cfg, log); githubSecurityFeaturesTool != nil {
		if err := RegisterTool(githubSecurityFeaturesTool); err != nil {
			log.Error("Failed to register github security features tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered consolidated github security features tool")
		}
	}

	if docsReaderTool := NewDocsReaderTool(cfg, log); docsReaderTool != nil {
		if err := RegisterTool(docsReaderTool); err != nil {
			log.Error("Failed to register docs reader tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered docs reader tool")
		}
	}

	// Register consolidated GitHub analysis tools
	if githubWorkflowTool := github.NewGitHubWorkflowAnalyzerAdapter(cfg, log); githubWorkflowTool != nil {
		if err := RegisterTool(githubWorkflowTool); err != nil {
			log.Error("Failed to register github workflow analyzer tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered consolidated github workflow analyzer tool")
		}
	}

	if githubReviewTool := github.NewGitHubReviewAnalyzerAdapter(cfg, log); githubReviewTool != nil {
		if err := RegisterTool(githubReviewTool); err != nil {
			log.Error("Failed to register github review analyzer tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered consolidated github review analyzer tool")
		}
	}

	// Register evidence generation and management tools
	if evidenceGeneratorTool, err := NewEvidenceGeneratorTool(cfg, log); err != nil {
		log.Error("Failed to create evidence generator tool", logger.Field{Key: "error", Value: err})
	} else if err := RegisterTool(evidenceGeneratorTool); err != nil {
		log.Error("Failed to register evidence generator tool", logger.Field{Key: "error", Value: err})
	} else {
		log.Debug("Registered evidence generator tool")
	}

	if evidenceValidatorTool, err := NewEvidenceValidatorTool(cfg, log); err != nil {
		log.Error("Failed to create evidence validator tool", logger.Field{Key: "error", Value: err})
	} else if err := RegisterTool(evidenceValidatorTool); err != nil {
		log.Error("Failed to register evidence validator tool", logger.Field{Key: "error", Value: err})
	} else {
		log.Debug("Registered evidence validator tool")
	}

	if tugboatSyncTool, err := NewTugboatSyncWrapperTool(cfg, log); err != nil {
		log.Error("Failed to create tugboat sync wrapper tool", logger.Field{Key: "error", Value: err})
	} else if err := RegisterTool(tugboatSyncTool); err != nil {
		log.Error("Failed to register tugboat sync wrapper tool", logger.Field{Key: "error", Value: err})
	} else {
		log.Debug("Registered tugboat sync wrapper tool")
	}

	if storageReadTool := NewStorageReadTool(cfg, log); storageReadTool != nil {
		if err := RegisterTool(storageReadTool); err != nil {
			log.Error("Failed to register storage read tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered storage read tool")
		}
	}

	if storageWriteTool := NewStorageWriteTool(cfg, log); storageWriteTool != nil {
		if err := RegisterTool(storageWriteTool); err != nil {
			log.Error("Failed to register storage write tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered storage write tool")
		}
	}

	if grctoolRunTool := NewGrctoolRunTool(cfg, log); grctoolRunTool != nil {
		if err := RegisterTool(grctoolRunTool); err != nil {
			log.Error("Failed to register grctool run tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered grctool run tool")
		}
	}

	// Register Google Workspace evidence collection tool
	if googleWorkspaceTool := NewGoogleWorkspaceTool(cfg, log); googleWorkspaceTool != nil {
		if err := RegisterTool(googleWorkspaceTool); err != nil {
			log.Error("Failed to register google workspace tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered google workspace tool")
		}
	}

	// Register name generator tool
	if nameGeneratorTool := NewNameGeneratorTool(cfg, log); nameGeneratorTool != nil {
		if err := RegisterTool(nameGeneratorTool); err != nil {
			log.Error("Failed to register name generator tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered name generator tool")
		}
	}

	// Register evidence writer tool
	if evidenceWriterTool := NewEvidenceWriterTool(cfg, log); evidenceWriterTool != nil {
		if err := RegisterTool(evidenceWriterTool); err != nil {
			log.Error("Failed to register evidence writer tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered evidence writer tool")
		}
	}

	log.Info("Tool registry initialization completed",
		logger.Field{Key: "total_tools", Value: GlobalRegistry.Count()})

	return nil
}

// GetAllToolsForFramework returns tools that are relevant for a specific framework
func GetAllToolsForFramework(framework string) []ToolInfo {
	allTools := ListTools()

	// For now, all tools are considered relevant to all frameworks
	// In the future, this could be enhanced to filter based on framework-specific capabilities
	return append([]ToolInfo{}, allTools...)
}

// GetEvidenceAnalysisTools returns the evidence analysis tools specifically
func GetEvidenceAnalysisTools() []ToolInfo {
	allTools := ListTools()
	var evidenceTools []ToolInfo

	evidenceToolNames := map[string]bool{
		"evidence-task-details":     true,
		"evidence-task-list":        true,
		"evidence-relationships":    true,
		"prompt-assembler":          true,
		"policy-summary-generator":  true,
		"control-summary-generator": true,
		"evidence-generator":        true,
		"evidence-validator":        true,
		"compliance-reporter":       true,
		"soc2-mapper":               true,
		"infrastructure-visualizer": true,
		"evidence-correlator":       true,
		"report-templates":          true,
	}

	for _, tool := range allTools {
		if evidenceToolNames[tool.Name] {
			evidenceTools = append(evidenceTools, tool)
		}
	}

	return evidenceTools
}

// GetDataSourceTools returns the enhanced data source tools specifically
func GetDataSourceTools() []ToolInfo {
	allTools := ListTools()
	var dataSourceTools []ToolInfo

	dataSourceToolNames := map[string]bool{
		"terraform_analyzer":          true,
		"terraform-hcl-parser":        true,
		"terraform-security-analyzer": true,
		// "terraform_analyzer_refactored": true, // TODO: implement
		"github-enhanced":          true,
		"github-workflow-analyzer": true,
		"github-review-analyzer":   true,
		// "github-permissions-refactored": true, // TODO: implement
		"docs-reader": true,
	}

	for _, tool := range allTools {
		if dataSourceToolNames[tool.Name] {
			dataSourceTools = append(dataSourceTools, tool)
		}
	}

	return dataSourceTools
}

// GetManagementTools returns the evidence management tools specifically
func GetManagementTools() []ToolInfo {
	allTools := ListTools()
	var managementTools []ToolInfo

	managementToolNames := map[string]bool{
		"evidence-generator":   true,
		"evidence-validator":   true,
		"tugboat-sync-wrapper": true,
		"storage-read":         true,
		"storage-write":        true,
		"grctool-run":          true,
	}

	for _, tool := range allTools {
		if managementToolNames[tool.Name] {
			managementTools = append(managementTools, tool)
		}
	}

	return managementTools
}

// GetStorageTools returns the storage-related tools specifically
func GetStorageTools() []ToolInfo {
	allTools := ListTools()
	var storageTools []ToolInfo

	storageToolNames := map[string]bool{
		"storage-read":  true,
		"storage-write": true,
	}

	for _, tool := range allTools {
		if storageToolNames[tool.Name] {
			storageTools = append(storageTools, tool)
		}
	}

	return storageTools
}
