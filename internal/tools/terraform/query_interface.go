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

package terraform

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// ClaudeQueryInterface provides a specialized interface for Claude to find compliance evidence
type ClaudeQueryInterface struct {
	config          *config.TerraformToolConfig
	logger          logger.Logger
	baseScanner     *Analyzer
	securityIndexer *SecurityAttributeIndexer
	atmosAnalyzer   *AtmosAnalyzer
}

// NewClaudeQueryInterface creates a new Claude query interface
func NewClaudeQueryInterface(cfg *config.Config, log logger.Logger) *ClaudeQueryInterface {
	return &ClaudeQueryInterface{
		config:          &cfg.Evidence.Tools.Terraform,
		logger:          log,
		baseScanner:     NewAnalyzer(cfg, log),
		securityIndexer: NewSecurityAttributeIndexer(cfg, log),
		atmosAnalyzer:   NewAtmosAnalyzer(cfg, log),
	}
}

// Name returns the tool name
func (cqi *ClaudeQueryInterface) Name() string {
	return "terraform-evidence-query"
}

// Description returns the tool description
func (cqi *ClaudeQueryInterface) Description() string {
	return "Intelligent query interface for Claude to find and retrieve Terraform infrastructure evidence for compliance frameworks"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (cqi *ClaudeQueryInterface) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        cqi.Name(),
		Description: cqi.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"evidence_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of evidence to find",
					"enum":        []string{"control_evidence", "framework_compliance", "security_configuration", "risk_analysis", "environment_comparison", "resource_inventory"},
					"default":     "control_evidence",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Natural language query describing what evidence is needed (e.g., 'Find encryption evidence for SOC2 CC6.8')",
				},
				"control_codes": map[string]interface{}{
					"type":        "array",
					"description": "Specific control codes to find evidence for (e.g., ['CC6.1', 'CC6.8', 'SO2'])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"evidence_tasks": map[string]interface{}{
					"type":        "array",
					"description": "Evidence task IDs to address (e.g., ['ET21', 'ET23', 'ET47'])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"frameworks": map[string]interface{}{
					"type":        "array",
					"description": "Compliance frameworks to check (e.g., ['SOC2', 'ISO27001', 'PCI'])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"environments": map[string]interface{}{
					"type":        "array",
					"description": "Environments to include (e.g., ['prod', 'staging'])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"risk_level": map[string]interface{}{
					"type":        "string",
					"description": "Filter by risk level",
					"enum":        []string{"high", "medium", "low", "all"},
					"default":     "all",
				},
				"detail_level": map[string]interface{}{
					"type":        "string",
					"description": "Level of detail in the response",
					"enum":        []string{"summary", "detailed", "comprehensive"},
					"default":     "detailed",
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "Format for evidence presentation",
					"enum":        []string{"narrative", "structured", "evidence_package", "audit_ready"},
					"default":     "narrative",
				},
				"include_remediation": map[string]interface{}{
					"type":        "boolean",
					"description": "Include remediation recommendations for any gaps found",
					"default":     true,
				},
			},
			"required": []string{"evidence_type"},
		},
	}
}

// Execute runs the Claude query interface with the given parameters
func (cqi *ClaudeQueryInterface) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	cqi.logger.Debug("Executing Claude evidence query interface", logger.Field{Key: "params", Value: params})

	// Extract parameters
	evidenceType := "control_evidence"
	if et, ok := params["evidence_type"].(string); ok {
		evidenceType = et
	}

	query := ""
	if q, ok := params["query"].(string); ok {
		query = q
	}

	var controlCodes []string
	if cc, ok := params["control_codes"].([]interface{}); ok {
		for _, c := range cc {
			if str, ok := c.(string); ok {
				controlCodes = append(controlCodes, str)
			}
		}
	}

	var evidenceTasks []string
	if et, ok := params["evidence_tasks"].([]interface{}); ok {
		for _, t := range et {
			if str, ok := t.(string); ok {
				evidenceTasks = append(evidenceTasks, str)
			}
		}
	}

	var frameworks []string
	if f, ok := params["frameworks"].([]interface{}); ok {
		for _, fr := range f {
			if str, ok := fr.(string); ok {
				frameworks = append(frameworks, str)
			}
		}
	}

	var environments []string
	if env, ok := params["environments"].([]interface{}); ok {
		for _, e := range env {
			if str, ok := e.(string); ok {
				environments = append(environments, str)
			}
		}
	}

	riskLevel := "all"
	if rl, ok := params["risk_level"].(string); ok {
		riskLevel = rl
	}

	detailLevel := "detailed"
	if dl, ok := params["detail_level"].(string); ok {
		detailLevel = dl
	}

	outputFormat := "narrative"
	if of, ok := params["output_format"].(string); ok {
		outputFormat = of
	}

	includeRemediation := true
	if ir, ok := params["include_remediation"].(bool); ok {
		includeRemediation = ir
	}

	// Process the query and gather evidence
	evidenceResult, err := cqi.processEvidenceQuery(ctx, EvidenceQuery{
		EvidenceType:         evidenceType,
		NaturalLanguageQuery: query,
		ControlCodes:         controlCodes,
		EvidenceTasks:        evidenceTasks,
		Frameworks:           frameworks,
		Environments:         environments,
		RiskLevel:            riskLevel,
		DetailLevel:          detailLevel,
		IncludeRemediation:   includeRemediation,
	})
	if err != nil {
		return "", nil, fmt.Errorf("failed to process evidence query: %w", err)
	}

	// Generate response in the requested format
	response, err := cqi.generateEvidenceResponse(evidenceResult, outputFormat)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate evidence response: %w", err)
	}

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "terraform-evidence-query",
		Resource:    fmt.Sprintf("Evidence query: %s (%d resources found)", evidenceType, len(evidenceResult.Resources)),
		Content:     response,
		Relevance:   cqi.calculateQueryRelevance(evidenceResult),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"evidence_type":    evidenceType,
			"natural_query":    query,
			"control_codes":    controlCodes,
			"evidence_tasks":   evidenceTasks,
			"frameworks":       frameworks,
			"environments":     environments,
			"risk_level":       riskLevel,
			"detail_level":     detailLevel,
			"resources_found":  len(evidenceResult.Resources),
			"gaps_identified":  len(evidenceResult.Gaps),
			"confidence_score": evidenceResult.ConfidenceScore,
		},
	}

	return response, source, nil
}

// processEvidenceQuery processes the evidence query and gathers relevant evidence
func (cqi *ClaudeQueryInterface) processEvidenceQuery(ctx context.Context, query EvidenceQuery) (*EvidenceQueryResult, error) {
	// Parse natural language query for additional context
	if query.NaturalLanguageQuery != "" {
		queryContext := cqi.parseNaturalLanguageQuery(query.NaturalLanguageQuery)
		// Merge parsed context with explicit parameters
		query = cqi.mergeQueryContext(query, queryContext)
	}

	// Process based on evidence type
	switch query.EvidenceType {
	case "control_evidence":
		return cqi.findControlEvidence(ctx, query)
	case "framework_compliance":
		return cqi.findFrameworkCompliance(ctx, query)
	case "security_configuration":
		return cqi.findSecurityConfiguration(ctx, query)
	case "risk_analysis":
		return cqi.performRiskAnalysis(ctx, query)
	case "environment_comparison":
		return cqi.compareEnvironments(ctx, query)
	case "resource_inventory":
		return cqi.generateResourceInventory(ctx, query)
	default:
		return nil, fmt.Errorf("unsupported evidence type: %s", query.EvidenceType)
	}
}

// parseNaturalLanguageQuery extracts information from natural language queries
func (cqi *ClaudeQueryInterface) parseNaturalLanguageQuery(query string) QueryContext {
	context := QueryContext{
		ExtractedControls:     []string{},
		ExtractedFrameworks:   []string{},
		ExtractedAttributes:   []string{},
		ExtractedEnvironments: []string{},
		Intent:                "",
	}

	queryLower := strings.ToLower(query)

	// Extract control codes (CC6.1, CC6.8, etc.)
	// Simplified extraction - in a real implementation, would use regexp
	if strings.Contains(queryLower, "cc6.1") {
		context.ExtractedControls = append(context.ExtractedControls, "CC6.1")
	}
	if strings.Contains(queryLower, "cc6.8") {
		context.ExtractedControls = append(context.ExtractedControls, "CC6.8")
	}
	if strings.Contains(queryLower, "cc7.2") {
		context.ExtractedControls = append(context.ExtractedControls, "CC7.2")
	}
	if strings.Contains(queryLower, "so2") {
		context.ExtractedControls = append(context.ExtractedControls, "SO2")
	}

	// Extract frameworks
	if strings.Contains(queryLower, "soc2") || strings.Contains(queryLower, "soc 2") {
		context.ExtractedFrameworks = append(context.ExtractedFrameworks, "SOC2")
	}
	if strings.Contains(queryLower, "iso27001") || strings.Contains(queryLower, "iso 27001") {
		context.ExtractedFrameworks = append(context.ExtractedFrameworks, "ISO27001")
	}
	if strings.Contains(queryLower, "pci") {
		context.ExtractedFrameworks = append(context.ExtractedFrameworks, "PCI")
	}

	// Extract security attributes
	attributeKeywords := map[string]string{
		"encrypt":  "encryption",
		"access":   "access_control",
		"iam":      "access_control",
		"monitor":  "monitoring",
		"log":      "monitoring",
		"backup":   "backup",
		"network":  "network_security",
		"firewall": "network_security",
	}

	for keyword, attribute := range attributeKeywords {
		if strings.Contains(queryLower, keyword) {
			context.ExtractedAttributes = append(context.ExtractedAttributes, attribute)
		}
	}

	// Extract environments
	envKeywords := []string{"prod", "production", "staging", "stage", "dev", "development", "test"}
	for _, env := range envKeywords {
		if strings.Contains(queryLower, env) {
			context.ExtractedEnvironments = append(context.ExtractedEnvironments, env)
		}
	}

	// Determine intent
	if strings.Contains(queryLower, "find") || strings.Contains(queryLower, "show") {
		context.Intent = "find_evidence"
	} else if strings.Contains(queryLower, "compare") {
		context.Intent = "compare"
	} else if strings.Contains(queryLower, "risk") || strings.Contains(queryLower, "vulnerab") {
		context.Intent = "risk_analysis"
	} else if strings.Contains(queryLower, "comply") || strings.Contains(queryLower, "compliance") {
		context.Intent = "compliance_check"
	}

	return context
}

// mergeQueryContext merges parsed natural language context with explicit parameters
func (cqi *ClaudeQueryInterface) mergeQueryContext(query EvidenceQuery, context QueryContext) EvidenceQuery {
	// Add extracted controls if not already specified
	if len(query.ControlCodes) == 0 {
		query.ControlCodes = context.ExtractedControls
	}

	// Add extracted frameworks if not already specified
	if len(query.Frameworks) == 0 {
		query.Frameworks = context.ExtractedFrameworks
	}

	// Add extracted environments if not already specified
	if len(query.Environments) == 0 {
		query.Environments = context.ExtractedEnvironments
	}

	// Adjust evidence type based on intent
	if query.EvidenceType == "control_evidence" && context.Intent == "risk_analysis" {
		query.EvidenceType = "risk_analysis"
	} else if query.EvidenceType == "control_evidence" && context.Intent == "compare" {
		query.EvidenceType = "environment_comparison"
	}

	return query
}

// findControlEvidence finds evidence for specific controls
func (cqi *ClaudeQueryInterface) findControlEvidence(ctx context.Context, query EvidenceQuery) (*EvidenceQueryResult, error) {
	result := &EvidenceQueryResult{
		QueryTimestamp: time.Now(),
		EvidenceType:   query.EvidenceType,
		Resources:      []EvidenceResource{},
		Gaps:           []EvidenceGap{},
	}

	// Try to use index for fast queries (index-first approach)
	allResources, err := cqi.getResourcesFromIndex(ctx, query.ControlCodes, query.Environments, query.RiskLevel)
	if err != nil {
		// Fallback to live scan if index fails
		cqi.logger.Warn("Failed to use index, falling back to live scan", logger.Field{Key: "error", Value: err})
		allResources, err = cqi.baseScanner.ScanForResources(ctx, []string{})
		if err != nil {
			return nil, fmt.Errorf("failed to scan terraform resources: %w", err)
		}
	}

	// Filter resources by control relevance
	for _, resource := range allResources {
		if cqi.isResourceRelevantToControls(resource, query.ControlCodes) {
			evidenceResource := cqi.convertToEvidenceResource(resource, query)
			result.Resources = append(result.Resources, evidenceResource)
		}
	}

	// Identify gaps for requested controls
	result.Gaps = cqi.identifyControlGaps(query.ControlCodes, result.Resources)

	// Calculate confidence score
	result.ConfidenceScore = cqi.calculateControlConfidence(query.ControlCodes, result.Resources)

	// Generate summary
	result.Summary = cqi.generateControlSummary(query.ControlCodes, result.Resources, result.Gaps)

	// Add recommendations if requested
	if query.IncludeRemediation {
		result.Recommendations = cqi.generateControlRecommendations(result.Gaps)
	}

	return result, nil
}

// findFrameworkCompliance finds compliance evidence for frameworks
func (cqi *ClaudeQueryInterface) findFrameworkCompliance(ctx context.Context, query EvidenceQuery) (*EvidenceQueryResult, error) {
	// For each framework, map to relevant controls and find evidence
	allControls := []string{}
	for _, framework := range query.Frameworks {
		controls := cqi.getFrameworkControls(framework)
		allControls = append(allControls, controls...)
	}

	// Remove duplicates
	uniqueControls := cqi.removeDuplicateStrings(allControls)

	// Create a control-based query
	controlQuery := query
	controlQuery.ControlCodes = uniqueControls
	controlQuery.EvidenceType = "control_evidence"

	return cqi.findControlEvidence(ctx, controlQuery)
}

// findSecurityConfiguration finds security configuration evidence
func (cqi *ClaudeQueryInterface) findSecurityConfiguration(ctx context.Context, query EvidenceQuery) (*EvidenceQueryResult, error) {
	result := &EvidenceQueryResult{
		QueryTimestamp: time.Now(),
		EvidenceType:   query.EvidenceType,
		Resources:      []EvidenceResource{},
		Gaps:           []EvidenceGap{},
	}

	// Use security indexer to find security configurations
	indexQuery := SecurityIndexQuery{
		ControlCodes:         query.ControlCodes,
		ComplianceFrameworks: query.Frameworks,
		Environments:         query.Environments,
		IncludeMetadata:      true,
	}

	securityIndex, err := cqi.securityIndexer.buildSecurityIndex(ctx, "by_attribute", indexQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to build security index: %w", err)
	}

	// Convert indexed resources to evidence resources
	for _, indexedResource := range securityIndex.IndexedResources {
		evidenceResource := EvidenceResource{
			ResourceID:       indexedResource.ResourceID,
			ResourceType:     indexedResource.ResourceType,
			ResourceName:     indexedResource.ResourceName,
			FilePath:         indexedResource.FilePath,
			LineRange:        indexedResource.LineRange,
			Environment:      indexedResource.Environment,
			ControlRelevance: indexedResource.ControlRelevance,
			SecurityConfig:   indexedResource.Configuration,
			RiskLevel:        indexedResource.RiskLevel,
			ComplianceStatus: indexedResource.ComplianceStatus,
			EvidenceQuality:  cqi.assessEvidenceQuality(indexedResource),
		}
		result.Resources = append(result.Resources, evidenceResource)
	}

	// Calculate confidence and generate summary
	result.ConfidenceScore = float64(len(result.Resources)) / 10.0 // Simple calculation
	if result.ConfidenceScore > 1.0 {
		result.ConfidenceScore = 1.0
	}

	result.Summary = fmt.Sprintf("Found %d security configuration resources across %d environments",
		len(result.Resources), len(query.Environments))

	return result, nil
}

// performRiskAnalysis performs risk analysis on infrastructure
func (cqi *ClaudeQueryInterface) performRiskAnalysis(ctx context.Context, query EvidenceQuery) (*EvidenceQueryResult, error) {
	result := &EvidenceQueryResult{
		QueryTimestamp: time.Now(),
		EvidenceType:   query.EvidenceType,
		Resources:      []EvidenceResource{},
		Gaps:           []EvidenceGap{},
	}

	// Get all resources and analyze risk
	allResources, err := cqi.baseScanner.ScanForResources(ctx, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to scan terraform resources: %w", err)
	}

	riskDistribution := make(map[string]int)

	for _, resource := range allResources {
		evidenceResource := cqi.convertToEvidenceResource(resource, query)

		// Filter by risk level if specified
		if query.RiskLevel == "all" || evidenceResource.RiskLevel == query.RiskLevel {
			result.Resources = append(result.Resources, evidenceResource)
		}

		riskDistribution[evidenceResource.RiskLevel]++
	}

	// Generate risk-focused summary
	result.Summary = fmt.Sprintf("Risk Analysis: High=%d, Medium=%d, Low=%d resources",
		riskDistribution["high"], riskDistribution["medium"], riskDistribution["low"])

	result.ConfidenceScore = 0.9 // High confidence for risk analysis

	return result, nil
}

// compareEnvironments compares configurations between environments
func (cqi *ClaudeQueryInterface) compareEnvironments(ctx context.Context, query EvidenceQuery) (*EvidenceQueryResult, error) {
	if len(query.Environments) < 2 {
		return nil, fmt.Errorf("environment comparison requires at least 2 environments")
	}

	// Use Atmos analyzer for environment comparison
	atmosParams := map[string]interface{}{
		"environments":           convertToInterfaceSlice(query.Environments),
		"include_drift_analysis": true,
		"output_format":          "drift_report",
	}

	_, _, err := cqi.atmosAnalyzer.Execute(ctx, atmosParams)
	if err != nil {
		return nil, fmt.Errorf("failed to perform environment comparison: %w", err)
	}

	result := &EvidenceQueryResult{
		QueryTimestamp:  time.Now(),
		EvidenceType:    query.EvidenceType,
		Resources:       []EvidenceResource{},
		Gaps:            []EvidenceGap{},
		Summary:         fmt.Sprintf("Compared %d environments for configuration drift", len(query.Environments)),
		ConfidenceScore: 0.8,
	}

	return result, nil
}

// generateResourceInventory generates a comprehensive resource inventory
func (cqi *ClaudeQueryInterface) generateResourceInventory(ctx context.Context, query EvidenceQuery) (*EvidenceQueryResult, error) {
	// Get all resources
	allResources, err := cqi.baseScanner.ScanForResources(ctx, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to scan terraform resources: %w", err)
	}

	result := &EvidenceQueryResult{
		QueryTimestamp:  time.Now(),
		EvidenceType:    query.EvidenceType,
		Resources:       []EvidenceResource{},
		Gaps:            []EvidenceGap{},
		Summary:         fmt.Sprintf("Resource inventory: %d total resources across all environments", len(allResources)),
		ConfidenceScore: 1.0,
	}

	// Convert all resources to evidence resources
	for _, resource := range allResources {
		evidenceResource := cqi.convertToEvidenceResource(resource, query)
		result.Resources = append(result.Resources, evidenceResource)
	}

	return result, nil
}

// Helper functions

func (cqi *ClaudeQueryInterface) isResourceRelevantToControls(resource models.TerraformScanResult, controlCodes []string) bool {
	if len(controlCodes) == 0 {
		return true
	}

	for _, control := range controlCodes {
		for _, resourceControl := range resource.SecurityRelevance {
			if resourceControl == control {
				return true
			}
		}
	}
	return false
}

func (cqi *ClaudeQueryInterface) convertToEvidenceResource(resource models.TerraformScanResult, query EvidenceQuery) EvidenceResource {
	return EvidenceResource{
		ResourceID:       fmt.Sprintf("%s.%s", resource.ResourceType, resource.ResourceName),
		ResourceType:     resource.ResourceType,
		ResourceName:     resource.ResourceName,
		FilePath:         resource.FilePath,
		LineRange:        fmt.Sprintf("%d-%d", resource.LineStart, resource.LineEnd),
		Environment:      cqi.extractEnvironmentFromPath(resource.FilePath),
		ControlRelevance: resource.SecurityRelevance,
		SecurityConfig:   resource.Configuration,
		RiskLevel:        cqi.calculateResourceRisk(resource),
		ComplianceStatus: cqi.calculateResourceCompliance(resource),
		EvidenceQuality:  cqi.assessResourceEvidenceQuality(resource),
	}
}

func (cqi *ClaudeQueryInterface) calculateResourceRisk(resource models.TerraformScanResult) string {
	// Simple risk calculation based on resource type and configuration
	resourceType := strings.ToLower(resource.ResourceType)

	highRiskTypes := []string{"iam_user", "s3_bucket", "security_group", "db_instance"}
	for _, riskType := range highRiskTypes {
		if strings.Contains(resourceType, riskType) {
			return "high"
		}
	}

	mediumRiskTypes := []string{"kms_key", "cloudtrail", "vpc"}
	for _, riskType := range mediumRiskTypes {
		if strings.Contains(resourceType, riskType) {
			return "medium"
		}
	}

	return "low"
}

func (cqi *ClaudeQueryInterface) calculateResourceCompliance(resource models.TerraformScanResult) string {
	// Simple compliance calculation
	if len(resource.SecurityRelevance) > 0 {
		return "compliant"
	}
	return "not_applicable"
}

func (cqi *ClaudeQueryInterface) assessResourceEvidenceQuality(resource models.TerraformScanResult) string {
	score := 0

	// Check for configuration completeness
	if len(resource.Configuration) > 5 {
		score += 2
	} else if len(resource.Configuration) > 2 {
		score += 1
	}

	// Check for security relevance
	if len(resource.SecurityRelevance) > 2 {
		score += 2
	} else if len(resource.SecurityRelevance) > 0 {
		score += 1
	}

	if score >= 3 {
		return "high"
	} else if score >= 2 {
		return "medium"
	} else {
		return "low"
	}
}

func (cqi *ClaudeQueryInterface) assessEvidenceQuality(resource IndexedResource) string {
	score := 0

	if len(resource.SecurityAttributes) > 3 {
		score += 2
	} else if len(resource.SecurityAttributes) > 1 {
		score += 1
	}

	switch resource.ComplianceStatus {
	case "compliant":
		score += 2
	case "partially_compliant":
		score += 1
	}

	if score >= 3 {
		return "high"
	} else if score >= 2 {
		return "medium"
	} else {
		return "low"
	}
}

func (cqi *ClaudeQueryInterface) extractEnvironmentFromPath(filePath string) string {
	fileLower := strings.ToLower(filePath)
	environments := []string{"prod", "production", "staging", "stage", "dev", "development", "test"}

	for _, env := range environments {
		if strings.Contains(fileLower, env) {
			return env
		}
	}

	return "unknown"
}

func (cqi *ClaudeQueryInterface) identifyControlGaps(controlCodes []string, resources []EvidenceResource) []EvidenceGap {
	var gaps []EvidenceGap

	for _, control := range controlCodes {
		hasEvidence := false
		for _, resource := range resources {
			for _, resourceControl := range resource.ControlRelevance {
				if resourceControl == control {
					hasEvidence = true
					break
				}
			}
			if hasEvidence {
				break
			}
		}

		if !hasEvidence {
			gaps = append(gaps, EvidenceGap{
				ControlCode:    control,
				GapType:        "missing_evidence",
				Description:    fmt.Sprintf("No Terraform resources found for control %s", control),
				Severity:       "medium",
				Recommendation: fmt.Sprintf("Implement Terraform resources that demonstrate compliance with %s", control),
			})
		}
	}

	return gaps
}

func (cqi *ClaudeQueryInterface) calculateControlConfidence(controlCodes []string, resources []EvidenceResource) float64 {
	if len(controlCodes) == 0 {
		return 1.0
	}

	coveredControls := 0
	for _, control := range controlCodes {
		for _, resource := range resources {
			for _, resourceControl := range resource.ControlRelevance {
				if resourceControl == control {
					coveredControls++
					goto nextControl
				}
			}
		}
	nextControl:
	}

	return float64(coveredControls) / float64(len(controlCodes))
}

func (cqi *ClaudeQueryInterface) generateControlSummary(controlCodes []string, resources []EvidenceResource, gaps []EvidenceGap) string {
	return fmt.Sprintf("Found evidence for %d controls across %d resources. Identified %d gaps.",
		len(controlCodes)-len(gaps), len(resources), len(gaps))
}

func (cqi *ClaudeQueryInterface) generateControlRecommendations(gaps []EvidenceGap) []string {
	var recommendations []string

	for _, gap := range gaps {
		recommendations = append(recommendations, gap.Recommendation)
	}

	return recommendations
}

func (cqi *ClaudeQueryInterface) getFrameworkControls(framework string) []string {
	frameworkControls := map[string][]string{
		"SOC2":     {"CC6.1", "CC6.2", "CC6.3", "CC6.6", "CC6.7", "CC6.8", "CC7.1", "CC7.2", "CC7.4", "CC8.1"},
		"ISO27001": {"A.9.1.1", "A.9.1.2", "A.10.1.1", "A.12.1.1", "A.12.6.1"},
		"PCI":      {"PCI-1", "PCI-2", "PCI-3", "PCI-4"},
	}

	if controls, exists := frameworkControls[framework]; exists {
		return controls
	}

	return []string{}
}

func (cqi *ClaudeQueryInterface) removeDuplicateStrings(slice []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, item := range slice {
		if !keys[item] {
			keys[item] = true
			result = append(result, item)
		}
	}

	return result
}

func (cqi *ClaudeQueryInterface) calculateQueryRelevance(result *EvidenceQueryResult) float64 {
	relevance := 0.5 // Base score

	// Boost for number of resources found
	if len(result.Resources) >= 20 {
		relevance += 0.2
	} else if len(result.Resources) >= 10 {
		relevance += 0.1
	}

	// Boost for confidence score
	relevance += result.ConfidenceScore * 0.3

	// Reduce for gaps found
	if len(result.Gaps) > 0 {
		relevance -= float64(len(result.Gaps)) * 0.02
	}

	// Cap at 1.0
	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}

func (cqi *ClaudeQueryInterface) generateEvidenceResponse(result *EvidenceQueryResult, outputFormat string) (string, error) {
	switch outputFormat {
	case "narrative":
		return cqi.generateNarrativeResponse(result)
	case "structured":
		return cqi.generateStructuredResponse(result)
	case "evidence_package":
		return cqi.generateEvidencePackage(result)
	case "audit_ready":
		return cqi.generateAuditReadyResponse(result)
	default:
		return "", fmt.Errorf("unsupported output format: %s", outputFormat)
	}
}

func (cqi *ClaudeQueryInterface) generateNarrativeResponse(result *EvidenceQueryResult) (string, error) {
	var response strings.Builder

	response.WriteString("# Evidence Query Results\n\n")
	response.WriteString(fmt.Sprintf("**Query Type:** %s\n", result.EvidenceType))
	response.WriteString(fmt.Sprintf("**Confidence Score:** %.1f%%\n", result.ConfidenceScore*100))
	response.WriteString(fmt.Sprintf("**Summary:** %s\n\n", result.Summary))

	if len(result.Resources) > 0 {
		response.WriteString("## Evidence Found\n\n")
		for i, resource := range result.Resources {
			if i >= 10 { // Limit for narrative format
				response.WriteString(fmt.Sprintf("... and %d more resources\n", len(result.Resources)-10))
				break
			}
			response.WriteString(fmt.Sprintf("### %s\n", resource.ResourceID))
			response.WriteString(fmt.Sprintf("- **Type:** %s\n", resource.ResourceType))
			response.WriteString(fmt.Sprintf("- **Environment:** %s\n", resource.Environment))
			response.WriteString(fmt.Sprintf("- **Risk Level:** %s\n", resource.RiskLevel))
			response.WriteString(fmt.Sprintf("- **Compliance:** %s\n", resource.ComplianceStatus))
			response.WriteString(fmt.Sprintf("- **Location:** %s (%s)\n\n", resource.FilePath, resource.LineRange))
		}
	}

	if len(result.Gaps) > 0 {
		response.WriteString("## Gaps Identified\n\n")
		for _, gap := range result.Gaps {
			response.WriteString(fmt.Sprintf("- **%s:** %s (%s severity)\n", gap.ControlCode, gap.Description, gap.Severity))
		}
		response.WriteString("\n")
	}

	if len(result.Recommendations) > 0 {
		response.WriteString("## Recommendations\n\n")
		for i, rec := range result.Recommendations {
			response.WriteString(fmt.Sprintf("%d. %s\n", i+1, rec))
		}
	}

	return response.String(), nil
}

func (cqi *ClaudeQueryInterface) generateStructuredResponse(result *EvidenceQueryResult) (string, error) {
	var response strings.Builder

	response.WriteString("Resource Type,Resource Name,Environment,Risk Level,Compliance Status,Control Relevance,File Path\n")

	for _, resource := range result.Resources {
		controls := strings.Join(resource.ControlRelevance, ";")
		response.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s\n",
			cqi.escapeCSV(resource.ResourceType),
			cqi.escapeCSV(resource.ResourceName),
			cqi.escapeCSV(resource.Environment),
			cqi.escapeCSV(resource.RiskLevel),
			cqi.escapeCSV(resource.ComplianceStatus),
			cqi.escapeCSV(controls),
			cqi.escapeCSV(resource.FilePath)))
	}

	return response.String(), nil
}

func (cqi *ClaudeQueryInterface) generateEvidencePackage(result *EvidenceQueryResult) (string, error) {
	var response strings.Builder

	response.WriteString("# Evidence Package\n\n")
	response.WriteString(fmt.Sprintf("**Generated:** %s\n", result.QueryTimestamp.Format(time.RFC3339)))
	response.WriteString(fmt.Sprintf("**Evidence Type:** %s\n", result.EvidenceType))
	response.WriteString(fmt.Sprintf("**Confidence Score:** %.1f%%\n\n", result.ConfidenceScore*100))

	// Group by environment
	envGroups := make(map[string][]EvidenceResource)
	for _, resource := range result.Resources {
		envGroups[resource.Environment] = append(envGroups[resource.Environment], resource)
	}

	for env, resources := range envGroups {
		response.WriteString(fmt.Sprintf("## %s Environment\n\n", cases.Title(language.English).String(env)))
		for _, resource := range resources {
			response.WriteString(fmt.Sprintf("- **%s** (%s) - %s\n", resource.ResourceName, resource.ResourceType, resource.ComplianceStatus))
		}
		response.WriteString("\n")
	}

	return response.String(), nil
}

func (cqi *ClaudeQueryInterface) generateAuditReadyResponse(result *EvidenceQueryResult) (string, error) {
	var response strings.Builder

	response.WriteString("# Audit Evidence Report\n\n")
	response.WriteString(fmt.Sprintf("**Report Date:** %s\n", time.Now().Format("2006-01-02")))
	response.WriteString(fmt.Sprintf("**Evidence Collection Type:** %s\n", result.EvidenceType))
	response.WriteString(fmt.Sprintf("**Total Evidence Items:** %d\n", len(result.Resources)))
	response.WriteString(fmt.Sprintf("**Evidence Confidence:** %.1f%%\n\n", result.ConfidenceScore*100))

	response.WriteString("## Executive Summary\n\n")
	response.WriteString(result.Summary)
	response.WriteString("\n\n")

	response.WriteString("## Evidence Details\n\n")

	// Sort by compliance status and risk level for audit presentation
	sort.Slice(result.Resources, func(i, j int) bool {
		if result.Resources[i].ComplianceStatus != result.Resources[j].ComplianceStatus {
			return result.Resources[i].ComplianceStatus > result.Resources[j].ComplianceStatus
		}
		return result.Resources[i].RiskLevel > result.Resources[j].RiskLevel
	})

	for _, resource := range result.Resources {
		response.WriteString(fmt.Sprintf("### Evidence Item: %s\n", resource.ResourceID))
		response.WriteString(fmt.Sprintf("- **Resource Type:** %s\n", resource.ResourceType))
		response.WriteString(fmt.Sprintf("- **Environment:** %s\n", resource.Environment))
		response.WriteString(fmt.Sprintf("- **Compliance Status:** %s\n", resource.ComplianceStatus))
		response.WriteString(fmt.Sprintf("- **Risk Assessment:** %s\n", resource.RiskLevel))
		response.WriteString(fmt.Sprintf("- **Evidence Quality:** %s\n", resource.EvidenceQuality))
		response.WriteString(fmt.Sprintf("- **Source Location:** %s (lines %s)\n", resource.FilePath, resource.LineRange))

		if len(resource.ControlRelevance) > 0 {
			response.WriteString(fmt.Sprintf("- **Applicable Controls:** %s\n", strings.Join(resource.ControlRelevance, ", ")))
		}
		response.WriteString("\n")
	}

	return response.String(), nil
}

// getResourcesFromIndex attempts to load resources from the persistent index with optional filtering
func (cqi *ClaudeQueryInterface) getResourcesFromIndex(ctx context.Context, controlCodes, environments []string, riskLevel string) ([]models.TerraformScanResult, error) {
	// Load the persistent index
	persistedIndex, err := cqi.securityIndexer.LoadOrBuildIndex(ctx, false)
	if err != nil {
		return nil, fmt.Errorf("failed to load index: %w", err)
	}

	// Create query interface
	query := NewIndexQuery(persistedIndex)

	var queryResult *QueryResult

	// Build compound query based on filters
	if len(controlCodes) > 0 {
		// Query by controls
		queryResult = query.ByControl(controlCodes...)
		cqi.logger.Debug("Queried index by controls",
			logger.Int("results", queryResult.Count),
			logger.Duration("query_time", queryResult.QueryTime))
	} else if len(environments) > 0 {
		// Query by environments
		var results []*QueryResult
		for _, env := range environments {
			results = append(results, query.ByEnvironment(env))
		}
		queryResult = Union(results...)
		cqi.logger.Debug("Queried index by environments",
			logger.Int("results", queryResult.Count),
			logger.Duration("query_time", queryResult.QueryTime))
	} else {
		// Get all resources
		queryResult = &QueryResult{
			Resources: persistedIndex.Index.IndexedResources,
			Count:     len(persistedIndex.Index.IndexedResources),
			QueryTime: 0,
		}
	}

	// Apply risk level filter if specified
	if riskLevel != "" && riskLevel != "all" {
		queryResult = query.ByRiskLevel(riskLevel).Filter(func(res IndexedResource) bool {
			// Check if resource is in the current result set
			for _, r := range queryResult.Resources {
				if r.ResourceID == res.ResourceID {
					return true
				}
			}
			return false
		})
	}

	// Apply environment filter if specified after control filtering
	if len(controlCodes) > 0 && len(environments) > 0 {
		envResult := queryResult.Filter(func(res IndexedResource) bool {
			for _, env := range environments {
				if res.Environment == env {
					return true
				}
			}
			return false
		})
		queryResult = envResult
	}

	// Convert IndexedResources to TerraformScanResults
	scanResults := make([]models.TerraformScanResult, 0, len(queryResult.Resources))
	for _, indexed := range queryResult.Resources {
		scanResult := models.TerraformScanResult{
			ResourceType:      indexed.ResourceType,
			ResourceName:      indexed.ResourceID,
			FilePath:          indexed.FilePath,
			Configuration:     indexed.Configuration,
			SecurityRelevance: indexed.ControlRelevance,
		}
		scanResults = append(scanResults, scanResult)
	}

	cqi.logger.Info("Using indexed resources",
		logger.Int("count", len(scanResults)),
		logger.Duration("query_time", queryResult.QueryTime))

	return scanResults, nil
}

func convertToInterfaceSlice(slice []string) []interface{} {
	result := make([]interface{}, len(slice))
	for i, v := range slice {
		result[i] = v
	}
	return result
}

func (cqi *ClaudeQueryInterface) escapeCSV(value string) string {
	if strings.Contains(value, ",") || strings.Contains(value, "\n") || strings.Contains(value, "\"") {
		value = strings.ReplaceAll(value, "\"", "\"\"")
		return "\"" + value + "\""
	}
	return value
}
