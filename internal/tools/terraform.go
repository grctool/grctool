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
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
)

// TerraformTool provides Terraform configuration scanning capabilities for evidence collection
type TerraformTool struct {
	config *config.TerraformToolConfig
	logger logger.Logger
}

// NewTerraformTool creates a new TerraformTool
func NewTerraformTool(cfg *config.Config, log logger.Logger) Tool {
	return &TerraformTool{
		config: &cfg.Evidence.Tools.Terraform,
		logger: log,
	}
}

// Name returns the tool name
func (tt *TerraformTool) Name() string {
	return "terraform_analyzer"
}

// Description returns the tool description
func (tt *TerraformTool) Description() string {
	return "Analyzes Terraform configuration files for security, modules, data sources, and compliance"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (tt *TerraformTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        tt.Name(),
		Description: tt.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"analysis_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of analysis to perform",
					"enum":        []string{"security_controls", "resource_types", "compliance_check", "security_issues", "modules", "data_sources", "locals"},
					"default":     "security_controls",
				},
				"resource_types": map[string]interface{}{
					"type":        "array",
					"description": "Resource types to scan for (e.g., aws_iam_role, aws_s3_bucket). Leave empty to scan all.",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"security_controls": map[string]interface{}{
					"type":        "array",
					"description": "Security controls to find evidence for (e.g., encryption, access_control, logging)",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"file_patterns": map[string]interface{}{
					"type":        "array",
					"description": "File patterns to filter analysis (e.g., security.tf, main.tf)",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"standards": map[string]interface{}{
					"type":        "array",
					"description": "Compliance standards to check (e.g., SOC2, PCI)",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"control_codes": map[string]interface{}{
					"type":        "array",
					"description": "Security control codes to find evidence for (e.g., CC6.1, CC6.8)",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "Output format: csv, markdown, or json",
					"enum":        []string{"csv", "markdown", "json"},
					"default":     "csv",
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs the Terraform analyzer with the given parameters
func (tt *TerraformTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	tt.logger.Debug("Executing Terraform analyzer", logger.Field{Key: "params", Value: params})

	// Get analysis_type parameter with default
	analysisType, ok := params["analysis_type"].(string)
	if !ok || analysisType == "" {
		analysisType = "security_controls" // Apply default value
	}

	// Validate analysis_type value
	validAnalysisTypes := map[string]bool{
		"security_controls": true,
		"resource_types":    true,
		"compliance_check":  true,
		"security_issues":   true,
		"modules":           true,
		"data_sources":      true,
		"locals":            true,
	}
	if !validAnalysisTypes[analysisType] {
		return "", nil, fmt.Errorf("invalid analysis_type: %s", analysisType)
	}

	// Extract parameters
	var resourceTypes []string
	if rtParam, exists := params["resource_types"]; exists {
		if rt, ok := rtParam.([]interface{}); ok {
			for _, r := range rt {
				if str, ok := r.(string); ok {
					resourceTypes = append(resourceTypes, str)
				}
			}
		} else {
			return "", nil, fmt.Errorf("resource_types must be an array")
		}
	}

	var securityControls []string
	if sc, ok := params["security_controls"].([]interface{}); ok {
		for _, s := range sc {
			if str, ok := s.(string); ok {
				securityControls = append(securityControls, str)
			}
		}
	}

	var filePatterns []string
	if fpParam, exists := params["file_patterns"]; exists {
		if fp, ok := fpParam.([]interface{}); ok {
			for _, f := range fp {
				if str, ok := f.(string); ok {
					filePatterns = append(filePatterns, str)
				}
			}
		} else {
			return "", nil, fmt.Errorf("file_patterns must be an array")
		}
	}

	var standards []string
	if st, ok := params["standards"].([]interface{}); ok {
		for _, s := range st {
			if str, ok := s.(string); ok {
				standards = append(standards, str)
			}
		}
	}

	var controlCodes []string
	if cc, ok := params["control_codes"].([]interface{}); ok {
		for _, c := range cc {
			if str, ok := c.(string); ok {
				controlCodes = append(controlCodes, str)
			}
		}
	}

	outputFormat := "csv"
	if of, ok := params["output_format"].(string); ok {
		outputFormat = of
	}

	// Extract optional evidence metadata parameters
	var controlHint string
	if ch, ok := params["control_hint"].(string); ok {
		controlHint = ch
	}

	var pattern string
	if p, ok := params["pattern"].(string); ok {
		pattern = p
	}

	// Default bounded_snippets to true (good for evidence quality)
	boundedSnippets := true
	if bs, ok := params["bounded_snippets"].(bool); ok {
		boundedSnippets = bs
	}

	// Execute scan based on analysis type
	var results []models.TerraformScanResult
	var err error

	switch analysisType {
	case "security_controls":
		results, err = tt.scanForSecurityControls(ctx, securityControls, filePatterns)
	case "resource_types":
		results, err = tt.scanForResourceTypes(ctx, resourceTypes, filePatterns)
	case "compliance_check":
		results, err = tt.scanForCompliance(ctx, standards, filePatterns)
	case "security_issues":
		results, err = tt.scanForSecurityIssues(ctx, filePatterns)
	case "modules":
		results, err = tt.scanForModules(ctx, filePatterns)
	case "data_sources":
		results, err = tt.scanForDataSources(ctx, filePatterns)
	case "locals":
		results, err = tt.scanForLocals(ctx, filePatterns)
	default:
		return "", nil, fmt.Errorf("unsupported analysis_type: %s", analysisType)
	}

	if err != nil {
		return "", nil, fmt.Errorf("failed to perform %s analysis: %w", analysisType, err)
	}

	// Handle control codes filtering if specified
	if len(controlCodes) > 0 {
		var filteredResults []models.TerraformScanResult
		for _, result := range results {
			for _, relevantControl := range result.SecurityRelevance {
				for _, requestedControl := range controlCodes {
					if relevantControl == requestedControl {
						filteredResults = append(filteredResults, result)
						goto nextResult
					}
				}
			}
		nextResult:
		}
		results = filteredResults
	}

	// Extract git hash from params if provided
	gitHash := ""
	if gh, ok := params["terraform_git_hash"].(string); ok {
		gitHash = gh
	}

	// Generate report
	report, err := tt.GenerateEvidenceReport(results, outputFormat, gitHash)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate report: %w", err)
	}

	// Create evidence source
	metadata := map[string]interface{}{
		"analysis_type":     analysisType,
		"resource_count":    len(results),
		"scan_paths":        tt.config.ScanPaths,
		"format":            outputFormat,
		"security_controls": securityControls,
		"file_patterns":     filePatterns,
		"standards":         standards,
	}

	// Add optional evidence metadata if provided
	if controlHint != "" {
		metadata["control_hint"] = controlHint
	}
	if pattern != "" {
		metadata["pattern"] = pattern
	}
	metadata["bounded_snippets"] = boundedSnippets

	// Add git hash if provided (during evidence generation)
	if gitHash, ok := params["terraform_git_hash"].(string); ok && gitHash != "" {
		metadata["terraform_git_hash"] = gitHash
	}

	source := &models.EvidenceSource{
		Type:        "terraform_analyzer",
		Resource:    fmt.Sprintf("Analyzed %d Terraform resources (%s)", len(results), analysisType),
		Content:     report,
		Relevance:   tt.calculateRelevance(results),
		ExtractedAt: time.Now(),
		Metadata:    metadata,
	}

	return report, source, nil
}

// calculateRelevance calculates the relevance score based on scan results
func (tt *TerraformTool) calculateRelevance(results []models.TerraformScanResult) float64 {
	if len(results) == 0 {
		return 0.0
	}

	// Base relevance on number of results and security relevance
	relevance := 0.5 // Base score

	// More results increase relevance
	if len(results) >= 10 {
		relevance += 0.2
	} else if len(results) >= 5 {
		relevance += 0.1
	}

	// Check for high-value resource types
	highValueTypes := map[string]bool{
		"aws_iam_role":                      true,
		"aws_iam_policy":                    true,
		"aws_s3_bucket":                     true,
		"aws_security_group":                true,
		"aws_kms_key":                       true,
		"aws_cloudtrail":                    true,
		"aws_config_configuration_recorder": true,
	}

	highValueCount := 0
	for _, result := range results {
		if highValueTypes[result.ResourceType] {
			highValueCount++
		}
	}

	if highValueCount >= 5 {
		relevance += 0.3
	} else if highValueCount >= 2 {
		relevance += 0.2
	} else if highValueCount >= 1 {
		relevance += 0.1
	}

	// Cap at 1.0
	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}

// ScanForResources scans Terraform files for specific resource types
func (tt *TerraformTool) ScanForResources(ctx context.Context, resourceTypes []string) ([]models.TerraformScanResult, error) {
	if !tt.config.Enabled {
		return nil, fmt.Errorf("terraform tool is not enabled")
	}

	var allResults []models.TerraformScanResult

	for _, scanPath := range tt.config.ScanPaths {
		results, err := tt.scanPath(ctx, scanPath, resourceTypes)
		if err != nil {
			continue
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// scanPath scans a specific path for Terraform files
func (tt *TerraformTool) scanPath(ctx context.Context, scanPath string, resourceTypes []string) ([]models.TerraformScanResult, error) {
	var results []models.TerraformScanResult

	// Expand glob patterns and walk directories
	err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}

		// Skip .terraform directories entirely (including all subdirectories)
		if info.IsDir() {
			if info.Name() == ".terraform" {
				return filepath.SkipDir
			}
			return nil
		}

		// Skip files inside .terraform directories
		if strings.Contains(path, string(filepath.Separator)+".terraform"+string(filepath.Separator)) {
			return nil
		}

		// Check if file matches include patterns
		if !tt.matchesPatterns(path, tt.config.IncludePatterns) {
			return nil
		}

		// Check if file matches exclude patterns
		if tt.matchesPatterns(path, tt.config.ExcludePatterns) {
			return nil
		}

		// Scan the file for resources
		fileResults, err := tt.scanFile(ctx, path, resourceTypes)
		if err != nil {
			return nil // Continue processing other files
		}

		results = append(results, fileResults...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk path %s: %w", scanPath, err)
	}

	return results, nil
}

// scanFile scans a single Terraform file for resource blocks
func (tt *TerraformTool) scanFile(ctx context.Context, filePath string, resourceTypes []string) ([]models.TerraformScanResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	var results []models.TerraformScanResult
	scanner := bufio.NewScanner(file)
	lineNum := 0
	inResourceBlock := false
	currentResource := &models.TerraformScanResult{}
	braceDepth := 0
	resourceContent := strings.Builder{}

	// Regular expressions for parsing Terraform
	resourcePattern := regexp.MustCompile(`^resource\s+"([^"]+)"\s+"([^"]+)"\s*\{`)
	variablePattern := regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*=`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Check for resource block start
		if !inResourceBlock {
			matches := resourcePattern.FindStringSubmatch(trimmedLine)
			if len(matches) == 3 {
				resourceType := matches[1]
				resourceName := matches[2]

				// Check if this resource type is of interest
				if tt.isResourceTypeOfInterest(resourceType, resourceTypes) {
					inResourceBlock = true
					braceDepth = 1
					currentResource = &models.TerraformScanResult{
						FilePath:          filePath,
						ResourceType:      resourceType,
						ResourceName:      resourceName,
						LineStart:         lineNum,
						Configuration:     make(map[string]interface{}),
						SecurityRelevance: tt.getSecurityRelevance(resourceType),
					}
					resourceContent.Reset()
					resourceContent.WriteString(line + "\n")
				}
			}
			continue
		}

		// If we're in a resource block, track braces and content
		resourceContent.WriteString(line + "\n")

		// Count braces to determine when block ends
		openBraces := strings.Count(line, "{")
		closeBraces := strings.Count(line, "}")
		braceDepth += openBraces - closeBraces

		// Parse configuration within the resource block
		if braceDepth > 0 {
			tt.parseResourceConfiguration(trimmedLine, currentResource, variablePattern)
		}

		// Check if resource block has ended
		if braceDepth <= 0 {
			currentResource.LineEnd = lineNum
			currentResource.Configuration["_content"] = resourceContent.String()
			// Add resource reference format for tests
			currentResource.Configuration["resource_reference"] = fmt.Sprintf("%s.%s", currentResource.ResourceType, currentResource.ResourceName)
			results = append(results, *currentResource)

			// Reset for next resource
			inResourceBlock = false
			currentResource = &models.TerraformScanResult{}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file %s: %w", filePath, err)
	}

	return results, nil
}

// matchesPatterns checks if a file path matches any of the given glob patterns
func (tt *TerraformTool) matchesPatterns(filePath string, patterns []string) bool {
	for _, pattern := range patterns {
		matched, err := filepath.Match(pattern, filepath.Base(filePath))
		if err != nil {
			continue
		}
		if matched {
			return true
		}

		// Also check if the full path matches (for patterns with directories)
		if strings.Contains(pattern, "/") {
			matched, err = filepath.Match(pattern, filePath)
			if err == nil && matched {
				return true
			}
		}
	}
	return false
}

// isResourceTypeOfInterest checks if a resource type matches the types we're looking for
func (tt *TerraformTool) isResourceTypeOfInterest(resourceType string, resourceTypes []string) bool {
	if len(resourceTypes) == 0 {
		return true // If no specific types requested, return all
	}

	for _, targetType := range resourceTypes {
		// Support glob-style matching
		if strings.Contains(targetType, "*") {
			matched, err := filepath.Match(targetType, resourceType)
			if err == nil && matched {
				return true
			}
		} else if resourceType == targetType {
			return true
		}
	}

	return false
}

// parseResourceConfiguration parses configuration lines within a resource block
func (tt *TerraformTool) parseResourceConfiguration(line string, resource *models.TerraformScanResult, variablePattern *regexp.Regexp) {
	// Skip comments and empty lines
	if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "//") || line == "" {
		return
	}

	// Simple key-value parsing
	matches := variablePattern.FindStringSubmatch(line)
	if len(matches) >= 2 {
		key := matches[1]

		// Extract value (simplified parsing)
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			value := strings.TrimSpace(parts[1])
			// Remove trailing comments
			if idx := strings.Index(value, "#"); idx != -1 {
				value = strings.TrimSpace(value[:idx])
			}
			// Remove quotes and other formatting
			value = strings.Trim(value, `"'`)

			resource.Configuration[key] = value
		}
	}

	// Look for security-relevant configurations
	if strings.Contains(strings.ToLower(line), "policy") ||
		strings.Contains(strings.ToLower(line), "security") ||
		strings.Contains(strings.ToLower(line), "encryption") ||
		strings.Contains(strings.ToLower(line), "access") {

		// Add to security relevance if not already present
		securityKey := "security_config"
		if existing, ok := resource.Configuration[securityKey]; ok {
			if existingStr, ok := existing.(string); ok {
				resource.Configuration[securityKey] = existingStr + "\n" + line
			}
		} else {
			resource.Configuration[securityKey] = line
		}
	}
}

// getSecurityRelevance returns the security controls that a resource type relates to
func (tt *TerraformTool) getSecurityRelevance(resourceType string) []string {
	// Default security mappings based on common AWS/Azure/GCP resource types
	securityMappings := map[string][]string{
		// AWS IAM
		"aws_iam_role":                   {"CC6.1", "CC6.3"},
		"aws_iam_policy":                 {"CC6.1", "CC6.3"},
		"aws_iam_user":                   {"CC6.1", "CC6.2"},
		"aws_iam_group":                  {"CC6.1", "CC6.3"},
		"aws_iam_access_key":             {"CC6.1", "CC6.2"},
		"aws_iam_role_policy_attachment": {"CC6.1", "CC6.3"},

		// AWS Network Security
		"aws_vpc":              {"CC6.6", "CC7.1"},
		"aws_security_group":   {"CC6.6", "CC7.1"},
		"aws_nacl":             {"CC6.6", "CC7.1"},
		"aws_subnet":           {"CC6.6", "CC7.1"},
		"aws_route_table":      {"CC6.6", "CC7.1"},
		"aws_internet_gateway": {"CC6.6", "CC7.1"},
		"aws_nat_gateway":      {"CC6.6", "CC7.1"},

		// AWS Load Balancing & SSL
		"aws_lb":                      {"CC6.7", "CC6.6"},
		"aws_lb_listener":             {"CC6.7", "CC6.6"},
		"aws_alb":                     {"CC6.7", "CC6.6"},
		"aws_cloudfront_distribution": {"CC6.7", "CC6.8"},
		"aws_acm_certificate":         {"CC6.7", "CC6.8"},

		// AWS Encryption & Data Protection
		"aws_kms_key":                       {"CC6.8"},
		"aws_kms_alias":                     {"CC6.8"},
		"aws_s3_bucket":                     {"CC6.8", "CC7.2"},
		"aws_s3_bucket_policy":              {"CC6.8", "CC6.3"},
		"aws_s3_bucket_encryption":          {"CC6.8"},
		"aws_s3_bucket_public_access_block": {"CC6.8", "CC6.3"},
		"aws_ebs_encryption_by_default":     {"CC6.8"},
		"aws_rds_cluster":                   {"CC6.8", "CC7.2"},

		// AWS Monitoring & Logging
		"aws_cloudtrail":                    {"CC7.2", "CC7.4"},
		"aws_cloudwatch_log_group":          {"CC7.2"},
		"aws_config_configuration_recorder": {"CC7.2", "CC8.1"},
		"aws_guardduty_detector":            {"CC7.3", "CC7.4"},

		// Azure equivalents
		"azurerm_resource_group":         {"CC6.3"},
		"azurerm_virtual_network":        {"CC6.6", "CC7.1"},
		"azurerm_network_security_group": {"CC6.6", "CC7.1"},
		"azurerm_key_vault":              {"CC6.8"},
		"azurerm_storage_account":        {"CC6.8", "CC7.2"},

		// Google Cloud equivalents
		"google_compute_network":     {"CC6.6", "CC7.1"},
		"google_compute_firewall":    {"CC6.6", "CC7.1"},
		"google_kms_crypto_key":      {"CC6.8"},
		"google_storage_bucket":      {"CC6.8", "CC7.2"},
		"google_project_iam_binding": {"CC6.1", "CC6.3"},

		// Autoscaling resources (for SO2 - System Operations)
		// AWS Autoscaling
		"aws_autoscaling_group":               {"SO2"},
		"aws_autoscaling_policy":              {"SO2"},
		"aws_appautoscaling_target":           {"SO2"},
		"aws_appautoscaling_policy":           {"SO2"},
		"aws_appautoscaling_scheduled_action": {"SO2"},
		"aws_ecs_service":                     {"SO2"}, // When using auto_scaling
		"aws_eks_node_group":                  {"SO2"}, // Has scaling_config

		// Azure Autoscaling
		"azurerm_autoscale_setting":            {"SO2"},
		"azurerm_monitor_autoscale_setting":    {"SO2"},
		"azurerm_virtual_machine_scale_set":    {"SO2"},
		"azurerm_kubernetes_cluster_node_pool": {"SO2"},

		// Google Cloud Autoscaling
		"google_compute_autoscaler":        {"SO2"},
		"google_compute_region_autoscaler": {"SO2"},
		"google_container_node_pool":       {"SO2"},
		"google_cloud_run_service":         {"SO2"}, // Has autoscaling annotations
	}

	if relevance, exists := securityMappings[resourceType]; exists {
		return relevance
	}

	// Generic mappings for unknown resource types
	resourceLower := strings.ToLower(resourceType)
	var relevance []string

	if strings.Contains(resourceLower, "iam") || strings.Contains(resourceLower, "auth") || strings.Contains(resourceLower, "access") {
		relevance = append(relevance, "CC6.1", "CC6.3")
	}
	if strings.Contains(resourceLower, "network") || strings.Contains(resourceLower, "firewall") || strings.Contains(resourceLower, "security_group") {
		relevance = append(relevance, "CC6.6", "CC7.1")
	}
	if strings.Contains(resourceLower, "encrypt") || strings.Contains(resourceLower, "kms") || strings.Contains(resourceLower, "key") {
		relevance = append(relevance, "CC6.8")
	}
	if strings.Contains(resourceLower, "log") || strings.Contains(resourceLower, "monitor") || strings.Contains(resourceLower, "audit") {
		relevance = append(relevance, "CC7.2")
	}
	if strings.Contains(resourceLower, "autoscal") || strings.Contains(resourceLower, "scaling") || strings.Contains(resourceLower, "scale_set") {
		relevance = append(relevance, "SO2")
	}

	return relevance
}

// ScanForSecurityControls scans for resources related to specific security controls
func (tt *TerraformTool) ScanForSecurityControls(ctx context.Context, controlCodes []string, securityMappings map[string]models.SecurityControlMapping) ([]models.TerraformScanResult, error) {

	// Build list of resource types to scan for based on control mappings
	var resourceTypes []string
	resourceTypeSet := make(map[string]bool)

	for _, controlCode := range controlCodes {
		if mapping, exists := securityMappings[controlCode]; exists {
			for _, resourceType := range mapping.TerraformResources {
				if !resourceTypeSet[resourceType] {
					resourceTypes = append(resourceTypes, resourceType)
					resourceTypeSet[resourceType] = true
				}
			}
		}
	}

	if len(resourceTypes) == 0 {
		return []models.TerraformScanResult{}, nil
	}

	// Perform the scan
	results, err := tt.ScanForResources(ctx, resourceTypes)
	if err != nil {
		return nil, err
	}

	// Filter results to only include those relevant to the requested controls
	var filteredResults []models.TerraformScanResult
	for _, result := range results {
		for _, relevantControl := range result.SecurityRelevance {
			for _, requestedControl := range controlCodes {
				if relevantControl == requestedControl {
					filteredResults = append(filteredResults, result)
					goto nextResult
				}
			}
		}
	nextResult:
	}

	return filteredResults, nil
}

// ExtractSecurityConfiguration extracts security-relevant configuration from scan results
func (tt *TerraformTool) ExtractSecurityConfiguration(results []models.TerraformScanResult) map[string]interface{} {
	securityConfig := make(map[string]interface{})

	// Group results by resource type
	byResourceType := make(map[string][]models.TerraformScanResult)
	for _, result := range results {
		byResourceType[result.ResourceType] = append(byResourceType[result.ResourceType], result)
	}

	// Extract key security configurations
	for resourceType, resources := range byResourceType {
		var configs []map[string]interface{}

		for _, resource := range resources {
			config := map[string]interface{}{
				"resource_name":      resource.ResourceName,
				"file_path":          resource.FilePath,
				"line_range":         fmt.Sprintf("%d-%d", resource.LineStart, resource.LineEnd),
				"security_relevance": resource.SecurityRelevance,
				"configuration":      resource.Configuration,
			}
			configs = append(configs, config)
		}

		securityConfig[resourceType] = configs
	}

	// Add summary statistics
	securityConfig["_summary"] = map[string]interface{}{
		"total_resources":      len(results),
		"resource_types_count": len(byResourceType),
		"scanned_at":           time.Now().Format(time.RFC3339),
	}

	return securityConfig
}

// GenerateEvidenceReport generates a structured evidence report from scan results
func (tt *TerraformTool) GenerateEvidenceReport(results []models.TerraformScanResult, format string, gitHash string) (string, error) {
	if len(results) == 0 {
		return "No Terraform resources found matching the criteria.", nil
	}

	// Sort results by resource type and name for consistent output
	sort.Slice(results, func(i, j int) bool {
		if results[i].ResourceType != results[j].ResourceType {
			return results[i].ResourceType < results[j].ResourceType
		}
		return results[i].ResourceName < results[j].ResourceName
	})

	switch format {
	case "csv":
		return tt.generateCSVReport(results, gitHash), nil
	case "markdown":
		return tt.generateMarkdownReport(results, gitHash), nil
	case "json":
		return tt.generateJSONReport(results, gitHash)
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// generateCSVReport generates a CSV format evidence report
func (tt *TerraformTool) generateCSVReport(results []models.TerraformScanResult, gitHash string) string {
	var report strings.Builder

	// Add git hash as a header comment if provided
	if gitHash != "" {
		report.WriteString(fmt.Sprintf("# Terraform Repository Commit: %s\n", gitHash))
	}

	// CSV Header
	report.WriteString("Resource Type,Resource Name,File Path,Line Range,Security Controls,Key Configuration\n")

	for _, result := range results {
		lineRange := fmt.Sprintf("%d-%d", result.LineStart, result.LineEnd)
		securityControls := strings.Join(result.SecurityRelevance, ";")

		// Extract key configuration items
		var keyConfigs []string
		for key, value := range result.Configuration {
			if key != "_content" && value != "" {
				keyConfigs = append(keyConfigs, fmt.Sprintf("%s=%v", key, value))
			}
		}
		keyConfigStr := strings.Join(keyConfigs, "; ")

		// Escape CSV values
		resourceType := tt.escapeCSV(result.ResourceType)
		resourceName := tt.escapeCSV(result.ResourceName)
		filePath := tt.escapeCSV(result.FilePath)
		securityControlsCSV := tt.escapeCSV(securityControls)
		keyConfigCSV := tt.escapeCSV(keyConfigStr)

		report.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			resourceType, resourceName, filePath, lineRange, securityControlsCSV, keyConfigCSV))
	}

	return report.String()
}

// generateMarkdownReport generates a Markdown format evidence report
func (tt *TerraformTool) generateMarkdownReport(results []models.TerraformScanResult, gitHash string) string {
	var report strings.Builder

	report.WriteString("# Enhanced Terraform Security Configuration Evidence\n\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339)))
	if gitHash != "" {
		report.WriteString(fmt.Sprintf("Terraform Repository Commit: `%s`\n", gitHash))
	}
	report.WriteString(fmt.Sprintf("Total Resources: %d\n\n", len(results)))

	// Group by resource type
	byResourceType := make(map[string][]models.TerraformScanResult)
	for _, result := range results {
		byResourceType[result.ResourceType] = append(byResourceType[result.ResourceType], result)
	}

	for resourceType, resources := range byResourceType {
		report.WriteString(fmt.Sprintf("## %s\n\n", resourceType))

		for _, resource := range resources {
			report.WriteString(fmt.Sprintf("### %s\n\n", resource.ResourceName))
			report.WriteString(fmt.Sprintf("**File:** `%s` (lines %d-%d)\n\n",
				resource.FilePath, resource.LineStart, resource.LineEnd))

			if len(resource.SecurityRelevance) > 0 {
				report.WriteString("**Security Controls:** ")
				for i, control := range resource.SecurityRelevance {
					if i > 0 {
						report.WriteString(", ")
					}
					report.WriteString(fmt.Sprintf("`%s`", control))
				}
				report.WriteString("\n\n")
			}

			if len(resource.Configuration) > 0 {
				report.WriteString("**Configuration:**\n\n")
				for key, value := range resource.Configuration {
					if key != "_content" && value != "" {
						report.WriteString(fmt.Sprintf("- **%s:** `%v`\n", key, value))
					}
				}
				report.WriteString("\n")
			}
		}
	}

	return report.String()
}

// generateJSONReport generates a JSON format evidence report
func (tt *TerraformTool) generateJSONReport(results []models.TerraformScanResult, gitHash string) (string, error) {
	// Group by resource type for better organization
	byResourceType := make(map[string][]models.TerraformScanResult)
	for _, result := range results {
		byResourceType[result.ResourceType] = append(byResourceType[result.ResourceType], result)
	}

	// Create scan summary
	scanSummary := map[string]interface{}{
		"total_resources":      len(results),
		"resource_types_count": len(byResourceType),
		"total_files":          tt.countUniqueFiles(results),
		"scanned_at":           time.Now().Format(time.RFC3339),
	}

	// Add git hash to scan summary if provided
	if gitHash != "" {
		scanSummary["terraform_git_hash"] = gitHash
	}

	// Create structured output matching test expectations
	output := map[string]interface{}{
		"results":        results,
		"scan_summary":   scanSummary,
		"resource_types": byResourceType,
	}

	jsonBytes, err := json.MarshalIndent(output, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(jsonBytes), nil
}

// countUniqueFiles counts unique file paths in scan results
func (tt *TerraformTool) countUniqueFiles(results []models.TerraformScanResult) int {
	files := make(map[string]bool)
	for _, result := range results {
		files[result.FilePath] = true
	}
	return len(files)
}

// escapeCSV escapes a string for CSV format
func (tt *TerraformTool) escapeCSV(value string) string {
	// If value contains comma, newline, or quote, wrap in quotes and escape internal quotes
	if strings.Contains(value, ",") || strings.Contains(value, "\n") || strings.Contains(value, "\"") {
		value = strings.ReplaceAll(value, "\"", "\"\"")
		return "\"" + value + "\""
	}
	return value
}

// Analysis type-specific methods

// scanForSecurityControls scans for resources matching security controls
func (tt *TerraformTool) scanForSecurityControls(ctx context.Context, controls, filePatterns []string) ([]models.TerraformScanResult, error) {
	results, err := tt.scanWithFilePatterns(ctx, []string{}, filePatterns)
	if err != nil {
		return nil, err
	}

	// Clean external references when file patterns are used
	if len(filePatterns) > 0 {
		results = tt.cleanExternalReferences(results)
	}

	// Filter by security controls if specified
	if len(controls) > 0 {
		var filteredResults []models.TerraformScanResult
		for _, result := range results {
			matched := false
			resourceTypeLower := strings.ToLower(result.ResourceType)

			for _, control := range controls {
				if matched {
					break
				}
				controlLower := strings.ToLower(control)

				// Check security relevance
				for _, relevance := range result.SecurityRelevance {
					if strings.Contains(strings.ToLower(relevance), controlLower) {
						matched = true
						break
					}
				}

				// Also check resource type for security relevance
				if !matched {
					if (controlLower == "encryption" && (strings.Contains(resourceTypeLower, "encrypt") || strings.Contains(resourceTypeLower, "kms") || strings.Contains(resourceTypeLower, "s3"))) ||
						(controlLower == "access_control" && (strings.Contains(resourceTypeLower, "iam") || strings.Contains(resourceTypeLower, "security_group"))) ||
						(controlLower == "logging" && (strings.Contains(resourceTypeLower, "cloudtrail") || strings.Contains(resourceTypeLower, "log"))) {
						matched = true
					}
				}
			}

			if matched {
				filteredResults = append(filteredResults, result)
			}
		}
		results = filteredResults
	}

	return results, nil
}

// scanForResourceTypes scans for specific resource types
func (tt *TerraformTool) scanForResourceTypes(ctx context.Context, resourceTypes, filePatterns []string) ([]models.TerraformScanResult, error) {
	return tt.scanWithFilePatterns(ctx, resourceTypes, filePatterns)
}

// scanForCompliance scans for compliance-related resources
func (tt *TerraformTool) scanForCompliance(ctx context.Context, standards, filePatterns []string) ([]models.TerraformScanResult, error) {
	results, err := tt.scanWithFilePatterns(ctx, []string{}, filePatterns)
	if err != nil {
		return nil, err
	}

	// Filter results based on compliance standards
	if len(standards) > 0 {
		var filteredResults []models.TerraformScanResult
		for _, result := range results {
			for _, standard := range standards {
				if tt.isComplianceRelevant(result, strings.ToLower(standard)) {
					filteredResults = append(filteredResults, result)
					break
				}
			}
		}
		results = filteredResults
	}

	return results, nil
}

// scanForSecurityIssues scans for potential security issues
func (tt *TerraformTool) scanForSecurityIssues(ctx context.Context, filePatterns []string) ([]models.TerraformScanResult, error) {
	results, err := tt.scanWithFilePatterns(ctx, []string{}, filePatterns)
	if err != nil {
		return nil, err
	}

	// Filter to resources that might have security issues
	var securityResults []models.TerraformScanResult
	for _, result := range results {
		if tt.hasSecurityIssues(result) {
			securityResults = append(securityResults, result)
		}
	}

	return securityResults, nil
}

// scanForModules scans for module declarations and usage
func (tt *TerraformTool) scanForModules(ctx context.Context, filePatterns []string) ([]models.TerraformScanResult, error) {
	return tt.scanForHCLBlocks(ctx, "module", filePatterns)
}

// scanForDataSources scans for data source declarations
func (tt *TerraformTool) scanForDataSources(ctx context.Context, filePatterns []string) ([]models.TerraformScanResult, error) {
	return tt.scanForHCLBlocks(ctx, "data", filePatterns)
}

// scanForLocals scans for locals blocks
func (tt *TerraformTool) scanForLocals(ctx context.Context, filePatterns []string) ([]models.TerraformScanResult, error) {
	return tt.scanForHCLBlocks(ctx, "locals", filePatterns)
}

// Helper methods

// scanWithFilePatterns performs scanning with file pattern filtering
func (tt *TerraformTool) scanWithFilePatterns(ctx context.Context, resourceTypes, filePatterns []string) ([]models.TerraformScanResult, error) {
	if !tt.config.Enabled {
		return nil, fmt.Errorf("terraform tool is not enabled")
	}

	var allResults []models.TerraformScanResult

	for _, scanPath := range tt.config.ScanPaths {
		results, err := tt.scanPathWithPatterns(ctx, scanPath, resourceTypes, filePatterns)
		if err != nil {
			continue
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// scanPathWithPatterns scans a path with additional file pattern filtering
func (tt *TerraformTool) scanPathWithPatterns(ctx context.Context, scanPath string, resourceTypes, filePatterns []string) ([]models.TerraformScanResult, error) {
	var results []models.TerraformScanResult

	err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file matches include patterns
		if !tt.matchesPatterns(path, tt.config.IncludePatterns) {
			return nil
		}

		// Check if file matches exclude patterns
		if tt.matchesPatterns(path, tt.config.ExcludePatterns) {
			return nil
		}

		// Apply additional file pattern filtering if specified
		if len(filePatterns) > 0 && !tt.matchesPatterns(path, filePatterns) {
			return nil
		}

		// Scan the file for resources
		fileResults, err := tt.scanFile(ctx, path, resourceTypes)
		if err != nil {
			return nil // Continue processing other files
		}

		results = append(results, fileResults...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk path %s: %w", scanPath, err)
	}

	return results, nil
}

// scanForHCLBlocks scans for specific HCL block types (module, data, locals)
func (tt *TerraformTool) scanForHCLBlocks(ctx context.Context, blockType string, filePatterns []string) ([]models.TerraformScanResult, error) {
	if !tt.config.Enabled {
		return nil, fmt.Errorf("terraform tool is not enabled")
	}

	var allResults []models.TerraformScanResult

	for _, scanPath := range tt.config.ScanPaths {
		results, err := tt.scanPathForHCLBlocks(ctx, scanPath, blockType, filePatterns)
		if err != nil {
			continue
		}
		allResults = append(allResults, results...)
	}

	return allResults, nil
}

// scanPathForHCLBlocks scans a path for specific HCL block types
func (tt *TerraformTool) scanPathForHCLBlocks(ctx context.Context, scanPath, blockType string, filePatterns []string) ([]models.TerraformScanResult, error) {
	var results []models.TerraformScanResult

	err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}

		if info.IsDir() {
			return nil
		}

		if !tt.matchesPatterns(path, tt.config.IncludePatterns) {
			return nil
		}

		if tt.matchesPatterns(path, tt.config.ExcludePatterns) {
			return nil
		}

		if len(filePatterns) > 0 && !tt.matchesPatterns(path, filePatterns) {
			return nil
		}

		fileResults, err := tt.scanFileForHCLBlocks(path, blockType)
		if err != nil {
			return nil
		}

		results = append(results, fileResults...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk path %s: %w", scanPath, err)
	}

	return results, nil
}

// scanFileForHCLBlocks scans a file for specific HCL block types
func (tt *TerraformTool) scanFileForHCLBlocks(filePath, blockType string) ([]models.TerraformScanResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	var results []models.TerraformScanResult
	scanner := bufio.NewScanner(file)
	lineNum := 0
	inBlock := false
	currentBlock := &models.TerraformScanResult{}
	braceDepth := 0
	blockContent := strings.Builder{}

	// Different regex patterns for different block types
	var blockPattern *regexp.Regexp
	switch blockType {
	case "module":
		blockPattern = regexp.MustCompile(`^module\s+"([^"]+)"\s*\{`)
	case "data":
		blockPattern = regexp.MustCompile(`^data\s+"([^"]+)"\s+"([^"]+)"\s*\{`)
	case "locals":
		blockPattern = regexp.MustCompile(`^locals\s*\{`)
	default:
		return nil, fmt.Errorf("unsupported block type: %s", blockType)
	}

	variablePattern := regexp.MustCompile(`^\s*([a-zA-Z_][a-zA-Z0-9_]*)\s*=`)

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Check for block start
		if !inBlock {
			matches := blockPattern.FindStringSubmatch(trimmedLine)
			if len(matches) >= 1 {
				inBlock = true
				braceDepth = 0

				// Set resource type and name based on block type
				var resourceType, resourceName string
				switch blockType {
				case "module":
					resourceType = "module"
					resourceName = matches[1]
					// Also add the reference format to configuration for searching
					// This will be included in the CSV output
				case "data":
					if len(matches) >= 3 {
						resourceType = matches[1]
						resourceName = matches[2]
					} else {
						resourceType = "data_source"
						resourceName = matches[1]
					}
				case "locals":
					resourceType = "locals"
					resourceName = "local_values"
				}

				currentBlock = &models.TerraformScanResult{
					FilePath:          filePath,
					ResourceType:      resourceType,
					ResourceName:      resourceName,
					LineStart:         lineNum,
					Configuration:     make(map[string]interface{}),
					SecurityRelevance: []string{},
				}

				// Add reference format for modules to make it findable in tests
				if blockType == "module" {
					currentBlock.Configuration["module_reference"] = fmt.Sprintf("module.%s", resourceName)
				}

				blockContent.Reset()
			}
		}

		// If we're in a block, track braces and content
		if inBlock {
			blockContent.WriteString(line + "\n")

			openBraces := strings.Count(line, "{")
			closeBraces := strings.Count(line, "}")
			braceDepth += openBraces - closeBraces

			// Parse configuration within the block
			if braceDepth > 0 {
				tt.parseResourceConfiguration(trimmedLine, currentBlock, variablePattern)
			}

			// Check if block has ended
			if braceDepth <= 0 {
				currentBlock.LineEnd = lineNum
				currentBlock.Configuration["_content"] = blockContent.String()
				results = append(results, *currentBlock)

				// Reset for next block
				inBlock = false
				currentBlock = &models.TerraformScanResult{}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file %s: %w", filePath, err)
	}

	return results, nil
}

// isComplianceRelevant checks if a resource is relevant for compliance standards
func (tt *TerraformTool) isComplianceRelevant(result models.TerraformScanResult, standard string) bool {
	resourceType := strings.ToLower(result.ResourceType)

	switch standard {
	case "soc2":
		return strings.Contains(resourceType, "encrypt") ||
			strings.Contains(resourceType, "iam") ||
			strings.Contains(resourceType, "security_group") ||
			strings.Contains(resourceType, "cloudtrail") ||
			strings.Contains(resourceType, "backup")
	case "pci":
		return strings.Contains(resourceType, "encrypt") ||
			strings.Contains(resourceType, "security_group") ||
			strings.Contains(resourceType, "waf") ||
			strings.Contains(resourceType, "lb")
	default:
		return true // For unknown standards, include all resources
	}
}

// hasSecurityIssues checks if a resource might have security issues
func (tt *TerraformTool) hasSecurityIssues(result models.TerraformScanResult) bool {
	resourceType := strings.ToLower(result.ResourceType)

	// Check for commonly problematic resource types
	problematicTypes := []string{
		"insecure", "public", "open", "permissive", "wildcard",
	}

	for _, problematic := range problematicTypes {
		if strings.Contains(resourceType, problematic) ||
			strings.Contains(strings.ToLower(result.ResourceName), problematic) {
			return true
		}
	}

	// Check configuration for security issues
	for key, value := range result.Configuration {
		keyLower := strings.ToLower(key)
		valueStr := fmt.Sprintf("%v", value)
		valueLower := strings.ToLower(valueStr)

		// Check for overly permissive configurations
		if (strings.Contains(keyLower, "cidr") && strings.Contains(valueLower, "0.0.0.0/0")) ||
			(strings.Contains(keyLower, "action") && valueLower == "*") ||
			(strings.Contains(keyLower, "resource") && valueLower == "*") ||
			(strings.Contains(keyLower, "public") && (valueLower == "true" || valueLower == "1")) {
			return true
		}
	}

	return false
}

// cleanExternalReferences removes or sanitizes references to external resources
func (tt *TerraformTool) cleanExternalReferences(results []models.TerraformScanResult) []models.TerraformScanResult {
	// Build a set of resource references that exist in these results
	resourceSet := make(map[string]bool)
	for _, result := range results {
		resourceRef := fmt.Sprintf("%s.%s", result.ResourceType, result.ResourceName)
		resourceSet[resourceRef] = true
	}

	// Clean configuration values that reference external resources
	for i, result := range results {
		cleanedConfig := make(map[string]interface{})
		for key, value := range result.Configuration {
			if key == "_content" || key == "resource_reference" || key == "module_reference" {
				// Keep these special keys as-is
				cleanedConfig[key] = value
				continue
			}

			valueStr := fmt.Sprintf("%v", value)
			// Check if value contains references to external resources
			if tt.containsExternalReferences(valueStr, resourceSet) {
				// Replace external references with sanitized version
				cleanedConfig[key] = tt.sanitizeExternalReferences(valueStr)
			} else {
				cleanedConfig[key] = value
			}
		}
		results[i].Configuration = cleanedConfig
	}

	return results
}

// containsExternalReferences checks if a value contains references to resources not in the resource set
func (tt *TerraformTool) containsExternalReferences(value string, resourceSet map[string]bool) bool {
	// Look for patterns like aws_vpc.main, module.vpc, etc.
	refPattern := regexp.MustCompile(`\b(aws_\w+|azurerm_\w+|google_\w+|module)\.[a-zA-Z_][a-zA-Z0-9_]*`)
	matches := refPattern.FindAllString(value, -1)

	for _, match := range matches {
		if !resourceSet[match] {
			// Found an external reference
			return true
		}
	}

	return false
}

// sanitizeExternalReferences removes or replaces external references in a value
func (tt *TerraformTool) sanitizeExternalReferences(value string) string {
	// Replace external resource references with [EXTERNAL_REF]
	refPattern := regexp.MustCompile(`\b(aws_\w+|azurerm_\w+|google_\w+|module)\.[a-zA-Z_][a-zA-Z0-9_]*`)
	return refPattern.ReplaceAllString(value, "[EXTERNAL_REF]")
}
