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
	"context"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools/terraform"
)

// TerraformSecurityAnalyzerAdapter adapts the new terraform.SecurityAnalyzer to the Tool interface
type TerraformSecurityAnalyzerAdapter struct {
	securityAnalyzer *terraform.SecurityAnalyzer
}

// NewTerraformSecurityAnalyzerAdapter creates a new adapter for the terraform security analyzer
func NewTerraformSecurityAnalyzerAdapter(cfg *config.Config, log logger.Logger) Tool {
	return &TerraformSecurityAnalyzerAdapter{
		securityAnalyzer: terraform.NewSecurityAnalyzer(cfg, log),
	}
}

func (t *TerraformSecurityAnalyzerAdapter) Name() string {
	return t.securityAnalyzer.Name()
}

func (t *TerraformSecurityAnalyzerAdapter) Description() string {
	return t.securityAnalyzer.Description()
}

func (t *TerraformSecurityAnalyzerAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return t.securityAnalyzer.GetClaudeToolDefinition()
}

func (t *TerraformSecurityAnalyzerAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return t.securityAnalyzer.Execute(ctx, params)
}

// TerraformHCLParserAdapter adapts the new terraform.HCLParser to the Tool interface
type TerraformHCLParserAdapter struct {
	hclParser *terraform.HCLParser
	logger    logger.Logger
}

// NewTerraformHCLParserAdapter creates a new adapter for the terraform HCL parser
func NewTerraformHCLParserAdapter(cfg *config.Config, log logger.Logger) Tool {
	return &TerraformHCLParserAdapter{
		hclParser: terraform.NewHCLParser(cfg, log),
		logger:    log,
	}
}

func (t *TerraformHCLParserAdapter) Name() string {
	return "terraform-hcl-parser"
}

func (t *TerraformHCLParserAdapter) Description() string {
	return "Comprehensive HCL parser for Terraform configurations with deep structural analysis"
}

func (t *TerraformHCLParserAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        t.Name(),
		Description: t.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"scan_paths": map[string]interface{}{
					"type":        "array",
					"description": "Paths to scan for Terraform files",
					"items": map[string]interface{}{
						"type": "string",
					},
					"default": []string{"./"},
				},
				"include_modules": map[string]interface{}{
					"type":        "boolean",
					"description": "Include module analysis in parsing",
					"default":     true,
				},
				"analyze_security": map[string]interface{}{
					"type":        "boolean",
					"description": "Perform security analysis during parsing",
					"default":     true,
				},
				"analyze_ha": map[string]interface{}{
					"type":        "boolean",
					"description": "Perform high availability analysis",
					"default":     true,
				},
				"control_mapping": map[string]interface{}{
					"type":        "array",
					"description": "Specific controls to map resources to (e.g., [\"CC6.1\", \"CC6.8\"])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"include_diagnostics": map[string]interface{}{
					"type":        "boolean",
					"description": "Include HCL parsing diagnostics in output",
					"default":     false,
				},
			},
			"required": []string{},
		},
	}
}

func (t *TerraformHCLParserAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	// Extract parameters
	scanPaths := []string{"./"}
	if sp, ok := params["scan_paths"].([]interface{}); ok {
		scanPaths = []string{}
		for _, path := range sp {
			if str, ok := path.(string); ok {
				scanPaths = append(scanPaths, str)
			}
		}
	}

	includeModules := true
	if im, ok := params["include_modules"].(bool); ok {
		includeModules = im
	}

	analyzeSecurity := true
	if as, ok := params["analyze_security"].(bool); ok {
		analyzeSecurity = as
	}

	analyzeHA := true
	if ah, ok := params["analyze_ha"].(bool); ok {
		analyzeHA = ah
	}

	var controlMapping []string
	if cm, ok := params["control_mapping"].([]interface{}); ok {
		for _, c := range cm {
			if str, ok := c.(string); ok {
				controlMapping = append(controlMapping, str)
			}
		}
	}

	includeDiagnostics := false
	if id, ok := params["include_diagnostics"].(bool); ok {
		includeDiagnostics = id
	}

	// Parse using HCL parser
	result, err := t.hclParser.Parse(ctx, scanPaths, includeModules, analyzeSecurity, analyzeHA, controlMapping, includeDiagnostics)
	if err != nil {
		return "", nil, err
	}

	// Generate report
	report, err := t.hclParser.GenerateReport(result, "detailed")
	if err != nil {
		return "", nil, err
	}

	// Calculate relevance
	relevance := t.hclParser.CalculateRelevance(result)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "terraform-hcl-parse",
		Resource:    "Terraform HCL parse results",
		Content:     report,
		Relevance:   relevance,
		ExtractedAt: result.ParsedAt,
		Metadata: map[string]interface{}{
			"files_processed":     result.ParseSummary.FilesProcessed,
			"total_resources":     result.ParseSummary.TotalResources,
			"modules_found":       len(result.Modules),
			"security_findings":   result.ParseSummary.SecurityFindings,
			"include_modules":     includeModules,
			"analyze_security":    analyzeSecurity,
			"analyze_ha":          analyzeHA,
			"include_diagnostics": includeDiagnostics,
		},
	}

	return report, source, nil
}

// TerraformSnippetsAdapter adapts the new terraform.SnippetsTool to the Tool interface
type TerraformSnippetsAdapter struct {
	snippetsTool *terraform.SnippetsTool
}

// NewTerraformSnippetsAdapter creates a new adapter for the terraform snippets tool
func NewTerraformSnippetsAdapter(cfg *config.Config, log logger.Logger) Tool {
	return &TerraformSnippetsAdapter{
		snippetsTool: terraform.NewSnippetsTool(cfg, log),
	}
}

func (t *TerraformSnippetsAdapter) Name() string {
	return t.snippetsTool.Name()
}

func (t *TerraformSnippetsAdapter) Description() string {
	return t.snippetsTool.Description()
}

func (t *TerraformSnippetsAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return t.snippetsTool.GetClaudeToolDefinition()
}

func (t *TerraformSnippetsAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return t.snippetsTool.Execute(ctx, params)
}

// TerraformSecurityIndexerAdapter adapts the new terraform.SecurityAttributeIndexer to the Tool interface
type TerraformSecurityIndexerAdapter struct {
	securityIndexer *terraform.SecurityAttributeIndexer
}

// NewTerraformSecurityIndexerAdapter creates a new adapter for the terraform security indexer
func NewTerraformSecurityIndexerAdapter(cfg *config.Config, log logger.Logger) Tool {
	return &TerraformSecurityIndexerAdapter{
		securityIndexer: terraform.NewSecurityAttributeIndexer(cfg, log),
	}
}

func (t *TerraformSecurityIndexerAdapter) Name() string {
	return t.securityIndexer.Name()
}

func (t *TerraformSecurityIndexerAdapter) Description() string {
	return t.securityIndexer.Description()
}

func (t *TerraformSecurityIndexerAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return t.securityIndexer.GetClaudeToolDefinition()
}

func (t *TerraformSecurityIndexerAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return t.securityIndexer.Execute(ctx, params)
}

// TerraformQueryInterfaceAdapter adapts the new terraform.ClaudeQueryInterface to the Tool interface
type TerraformQueryInterfaceAdapter struct {
	queryInterface *terraform.ClaudeQueryInterface
}

// NewTerraformQueryInterfaceAdapter creates a new adapter for the terraform query interface
func NewTerraformQueryInterfaceAdapter(cfg *config.Config, log logger.Logger) Tool {
	return &TerraformQueryInterfaceAdapter{
		queryInterface: terraform.NewClaudeQueryInterface(cfg, log),
	}
}

func (t *TerraformQueryInterfaceAdapter) Name() string {
	return t.queryInterface.Name()
}

func (t *TerraformQueryInterfaceAdapter) Description() string {
	return t.queryInterface.Description()
}

func (t *TerraformQueryInterfaceAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return t.queryInterface.GetClaudeToolDefinition()
}

func (t *TerraformQueryInterfaceAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return t.queryInterface.Execute(ctx, params)
}

// TerraformAtmosAdapter adapts the new terraform.AtmosAnalyzer to the Tool interface
type TerraformAtmosAdapter struct {
	atmosAnalyzer *terraform.AtmosAnalyzer
}

// NewTerraformAtmosAdapter creates a new adapter for the terraform atmos analyzer
func NewTerraformAtmosAdapter(cfg *config.Config, log logger.Logger) Tool {
	return &TerraformAtmosAdapter{
		atmosAnalyzer: terraform.NewAtmosAnalyzer(cfg, log),
	}
}

func (t *TerraformAtmosAdapter) Name() string {
	return t.atmosAnalyzer.Name()
}

func (t *TerraformAtmosAdapter) Description() string {
	return t.atmosAnalyzer.Description()
}

func (t *TerraformAtmosAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return t.atmosAnalyzer.GetClaudeToolDefinition()
}

func (t *TerraformAtmosAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return t.atmosAnalyzer.Execute(ctx, params)
}
