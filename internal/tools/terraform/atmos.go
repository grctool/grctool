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
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"gopkg.in/yaml.v3"
)

// AtmosAnalyzer provides Atmos stack configuration analysis capabilities
type AtmosAnalyzer struct {
	config      *config.TerraformToolConfig
	logger      logger.Logger
	baseScanner *Analyzer
}

// NewAtmosAnalyzer creates a new Atmos stack analyzer
func NewAtmosAnalyzer(cfg *config.Config, log logger.Logger) *AtmosAnalyzer {
	return &AtmosAnalyzer{
		config:      &cfg.Evidence.Tools.Terraform,
		logger:      log,
		baseScanner: NewAnalyzer(cfg, log),
	}
}

// Name returns the tool name
func (aa *AtmosAnalyzer) Name() string {
	return "atmos-stack-analyzer"
}

// Description returns the tool description
func (aa *AtmosAnalyzer) Description() string {
	return "Analyzes Atmos stack configurations and multi-environment Terraform deployments for security compliance"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (aa *AtmosAnalyzer) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        aa.Name(),
		Description: aa.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"environments": map[string]interface{}{
					"type":        "array",
					"description": "Specific environments to analyze (e.g., ['dev', 'staging', 'prod']). Leave empty for all environments.",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"stack_names": map[string]interface{}{
					"type":        "array",
					"description": "Specific stack names to analyze (e.g., ['vpc', 'app', 'db']). Leave empty for all stacks.",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"security_focus": map[string]interface{}{
					"type":        "string",
					"description": "Security domain to focus analysis on",
					"enum":        []string{"encryption", "network", "iam", "monitoring", "backup", "all"},
					"default":     "all",
				},
				"compliance_frameworks": map[string]interface{}{
					"type":        "array",
					"description": "Compliance frameworks to check against (e.g., ['SOC2', 'ISO27001', 'PCI'])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"include_drift_analysis": map[string]interface{}{
					"type":        "boolean",
					"description": "Include configuration drift analysis between environments",
					"default":     true,
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "Output format for the analysis results",
					"enum":        []string{"detailed_json", "summary_markdown", "compliance_csv", "drift_report"},
					"default":     "detailed_json",
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs the Atmos stack analyzer with the given parameters
func (aa *AtmosAnalyzer) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	aa.logger.Debug("Executing Atmos stack analyzer", logger.Field{Key: "params", Value: params})

	// Extract parameters
	var environments []string
	if env, ok := params["environments"].([]interface{}); ok {
		for _, e := range env {
			if str, ok := e.(string); ok {
				environments = append(environments, str)
			}
		}
	}

	var stackNames []string
	if stacks, ok := params["stack_names"].([]interface{}); ok {
		for _, s := range stacks {
			if str, ok := s.(string); ok {
				stackNames = append(stackNames, str)
			}
		}
	}

	securityFocus := "all"
	if sf, ok := params["security_focus"].(string); ok {
		securityFocus = sf
	}

	var complianceFrameworks []string
	if cf, ok := params["compliance_frameworks"].([]interface{}); ok {
		for _, f := range cf {
			if str, ok := f.(string); ok {
				complianceFrameworks = append(complianceFrameworks, str)
			}
		}
	}

	includeDriftAnalysis := true
	if ida, ok := params["include_drift_analysis"].(bool); ok {
		includeDriftAnalysis = ida
	}

	outputFormat := "detailed_json"
	if of, ok := params["output_format"].(string); ok {
		outputFormat = of
	}

	// Perform Atmos stack analysis
	analysis, err := aa.performAtmosStackAnalysis(ctx, environments, stackNames, securityFocus, complianceFrameworks, includeDriftAnalysis)
	if err != nil {
		return "", nil, fmt.Errorf("failed to perform Atmos stack analysis: %w", err)
	}

	// Generate report based on format
	report, err := aa.generateAtmosReport(analysis, outputFormat)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate Atmos report: %w", err)
	}

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "atmos-stack-analysis",
		Resource:    fmt.Sprintf("Atmos analysis of %d stacks across %d environments", len(analysis.Stacks), len(analysis.Environments)),
		Content:     report,
		Relevance:   aa.calculateAtmosRelevance(analysis),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"environments_analyzed":     len(analysis.Environments),
			"stacks_analyzed":           len(analysis.Stacks),
			"security_focus":            securityFocus,
			"compliance_frameworks":     complianceFrameworks,
			"include_drift_analysis":    includeDriftAnalysis,
			"drift_issues_found":        len(analysis.DriftIssues),
			"security_findings_total":   aa.countSecurityFindings(analysis.Stacks),
			"compliance_coverage_score": analysis.ComplianceCoverage,
		},
	}

	return report, source, nil
}

// performAtmosStackAnalysis performs comprehensive Atmos stack configuration analysis
func (aa *AtmosAnalyzer) performAtmosStackAnalysis(ctx context.Context, environments, stackNames []string, securityFocus string, complianceFrameworks []string, includeDriftAnalysis bool) (*AtmosStackAnalysis, error) {
	analysis := &AtmosStackAnalysis{
		AnalysisTimestamp:  time.Now(),
		Environments:       []string{},
		Stacks:             []AtmosStack{},
		DriftIssues:        []ConfigurationDrift{},
		SecurityFindings:   []StackSecurityFinding{},
		ComplianceMapping:  make(map[string][]AtmosStack),
		ComplianceCoverage: 0.0,
		EnvironmentMatrix:  make(map[string]map[string]AtmosStack),
	}

	// Discover Atmos stack configurations
	stacks, err := aa.discoverAtmosStacks(ctx, environments, stackNames)
	if err != nil {
		return nil, fmt.Errorf("failed to discover Atmos stacks: %w", err)
	}

	analysis.Stacks = stacks

	// Extract unique environments
	envSet := make(map[string]bool)
	for _, stack := range stacks {
		envSet[stack.Environment] = true
	}
	for env := range envSet {
		analysis.Environments = append(analysis.Environments, env)
	}
	sort.Strings(analysis.Environments)

	// Build environment matrix
	for _, stack := range stacks {
		if analysis.EnvironmentMatrix[stack.Environment] == nil {
			analysis.EnvironmentMatrix[stack.Environment] = make(map[string]AtmosStack)
		}
		analysis.EnvironmentMatrix[stack.Environment][stack.StackName] = stack
	}

	// Perform security analysis on each stack
	for i, stack := range analysis.Stacks {
		securityFindings := aa.analyzeStackSecurity(stack, securityFocus)
		analysis.Stacks[i].SecurityFindings = securityFindings

		// Add to global security findings
		for _, finding := range securityFindings {
			stackFinding := StackSecurityFinding{
				StackName:       stack.StackName,
				Environment:     stack.Environment,
				SecurityFinding: finding,
			}
			analysis.SecurityFindings = append(analysis.SecurityFindings, stackFinding)
		}
	}

	// Perform compliance mapping
	aa.mapStacksToCompliance(analysis.Stacks, complianceFrameworks, analysis.ComplianceMapping)
	analysis.ComplianceCoverage = aa.calculateComplianceCoverage(analysis.Stacks, complianceFrameworks)

	// Perform drift analysis if requested
	if includeDriftAnalysis && len(analysis.Environments) > 1 {
		analysis.DriftIssues = aa.analyzeDriftBetweenEnvironments(analysis.EnvironmentMatrix)
	}

	return analysis, nil
}

// discoverAtmosStacks discovers and parses Atmos stack configurations
func (aa *AtmosAnalyzer) discoverAtmosStacks(ctx context.Context, environments, stackNames []string) ([]AtmosStack, error) {
	var stacks []AtmosStack

	// Use Atmos path from config or default
	atmosPath := aa.getAtmosPath()

	// Walk through Atmos directory structure
	err := filepath.Walk(atmosPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Look for Atmos stack configuration files
		if aa.isAtmosStackFile(path) {
			stack, err := aa.parseAtmosStackFile(path)
			if err != nil {
				aa.logger.Warn("Failed to parse Atmos stack file",
					logger.Field{Key: "file", Value: path},
					logger.Field{Key: "error", Value: err})
				return nil // Continue with other files
			}

			// Filter by requested environments
			if len(environments) > 0 && !aa.stringInSlice(stack.Environment, environments) {
				return nil
			}

			// Filter by requested stack names
			if len(stackNames) > 0 && !aa.stringInSlice(stack.StackName, stackNames) {
				return nil
			}

			stacks = append(stacks, *stack)
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to walk Atmos directory: %w", err)
	}

	return stacks, nil
}

// isAtmosStackFile checks if a file is an Atmos stack configuration file
func (aa *AtmosAnalyzer) isAtmosStackFile(path string) bool {
	// Look for typical Atmos patterns
	patterns := []string{
		`/stacks/.*\.yaml$`,
		`/stacks/.*\.yml$`,
		`/environments/.*\.yaml$`,
		`/environments/.*\.yml$`,
		`.*-stack\.yaml$`,
		`.*-stack\.yml$`,
	}

	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, path)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// parseAtmosStackFile parses an Atmos stack configuration file
func (aa *AtmosAnalyzer) parseAtmosStackFile(filePath string) (*AtmosStack, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var stackConfig map[string]interface{}
	err = yaml.Unmarshal(content, &stackConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to parse YAML: %w", err)
	}

	stack := &AtmosStack{
		FilePath:           filePath,
		StackName:          aa.extractStackNameFromPath(filePath),
		Environment:        aa.extractEnvironmentFromPath(filePath),
		TerraformResources: []models.TerraformScanResult{},
		Variables:          make(map[string]interface{}),
		SecurityFindings:   []SecurityFinding{},
	}

	// Extract variables and configuration
	if vars, ok := stackConfig["vars"].(map[string]interface{}); ok {
		stack.Variables = vars
	}

	// Extract component/terraform reference
	if component, ok := stackConfig["component"].(string); ok {
		stack.Component = component
	} else if terraform, ok := stackConfig["terraform"].(string); ok {
		stack.Component = terraform
	}

	// Extract backend configuration
	if backend, ok := stackConfig["backend"].(map[string]interface{}); ok {
		stack.Backend = backend
	}

	// Extract workspace
	if workspace, ok := stackConfig["workspace"].(string); ok {
		stack.Workspace = workspace
	}

	// Try to find and parse associated Terraform files
	terraformPath := aa.findTerraformPathForStack(stack)
	if terraformPath != "" {
		terraformResources, err := aa.scanTerraformForStack(terraformPath)
		if err != nil {
			aa.logger.Debug("Failed to scan Terraform for stack",
				logger.Field{Key: "stack", Value: stack.StackName},
				logger.Field{Key: "terraform_path", Value: terraformPath},
				logger.Field{Key: "error", Value: err})
		} else {
			stack.TerraformResources = terraformResources
		}
	}

	return stack, nil
}

// extractStackNameFromPath extracts the stack name from the file path
func (aa *AtmosAnalyzer) extractStackNameFromPath(filePath string) string {
	base := filepath.Base(filePath)
	name := strings.TrimSuffix(base, filepath.Ext(base))

	// Remove common suffixes
	suffixes := []string{"-stack", "_stack", ".stack"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(name, suffix) {
			name = strings.TrimSuffix(name, suffix)
			break
		}
	}

	return name
}

// extractEnvironmentFromPath extracts the environment from the file path
func (aa *AtmosAnalyzer) extractEnvironmentFromPath(filePath string) string {
	// Common patterns for environment extraction
	patterns := []map[string]*regexp.Regexp{
		{"env": regexp.MustCompile(`/environments/([^/]+)/`)},
		{"env": regexp.MustCompile(`/([^/]+)/stacks/`)},
		{"env": regexp.MustCompile(`([^/]+)-stack\.ya?ml$`)},
		{"env": regexp.MustCompile(`/stacks/([^/]+)\.ya?ml$`)},
	}

	for _, patternMap := range patterns {
		for _, pattern := range patternMap {
			matches := pattern.FindStringSubmatch(filePath)
			if len(matches) > 1 {
				env := matches[1]
				// Filter out common non-environment names
				if !aa.isNonEnvironmentName(env) {
					return env
				}
			}
		}
	}

	// Default to "unknown" if can't determine
	return "unknown"
}

// isNonEnvironmentName checks if a name is likely not an environment
func (aa *AtmosAnalyzer) isNonEnvironmentName(name string) bool {
	nonEnvNames := []string{
		"stacks", "components", "terraform", "modules", "configs",
		"common", "shared", "global", "base",
	}

	for _, nonEnv := range nonEnvNames {
		if strings.EqualFold(name, nonEnv) {
			return true
		}
	}

	return false
}

// findTerraformPathForStack finds the Terraform component path for a stack
func (aa *AtmosAnalyzer) findTerraformPathForStack(stack *AtmosStack) string {
	if stack.Component == "" {
		return ""
	}

	// Common paths to check for Terraform components
	basePath := aa.getAtmosPath()
	possiblePaths := []string{
		filepath.Join(basePath, "components", "terraform", stack.Component),
		filepath.Join(basePath, "terraform", stack.Component),
		filepath.Join(basePath, "..", "components", "terraform", stack.Component),
		filepath.Join(basePath, "..", "terraform", stack.Component),
	}

	for _, path := range possiblePaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

// scanTerraformForStack scans Terraform files for a specific stack
func (aa *AtmosAnalyzer) scanTerraformForStack(terraformPath string) ([]models.TerraformScanResult, error) {
	// Use the base scanner to scan the Terraform component directory
	return aa.baseScanner.ScanPath(context.Background(), terraformPath, []string{})
}

// analyzeStackSecurity performs security analysis on a single stack
func (aa *AtmosAnalyzer) analyzeStackSecurity(stack AtmosStack, securityFocus string) []SecurityFinding {
	var findings []SecurityFinding

	// Analyze variables for security configurations
	findings = append(findings, aa.analyzeStackVariables(stack)...)

	// Analyze Terraform resources if available
	for _, resource := range stack.TerraformResources {
		resourceFindings := aa.analyzeResourceSecurity(resource, securityFocus)
		findings = append(findings, resourceFindings...)
	}

	// Analyze backend configuration for security
	findings = append(findings, aa.analyzeBackendSecurity(stack)...)

	return findings
}

// analyzeStackVariables analyzes stack variables for security configurations
func (aa *AtmosAnalyzer) analyzeStackVariables(stack AtmosStack) []SecurityFinding {
	var findings []SecurityFinding

	// Check for sensitive variables that might not be properly managed
	sensitiveKeys := []string{
		"password", "secret", "key", "token", "credential", "private",
	}

	for varName, varValue := range stack.Variables {
		varNameLower := strings.ToLower(varName)

		for _, sensitive := range sensitiveKeys {
			if strings.Contains(varNameLower, sensitive) {
				// Check if the value looks like a hardcoded secret
				if valueStr, ok := varValue.(string); ok && valueStr != "" {
					if !aa.isVariableReference(valueStr) {
						findings = append(findings, SecurityFinding{
							Type:           "secrets_management",
							Severity:       "high",
							Description:    fmt.Sprintf("Potentially hardcoded sensitive value in variable: %s", varName),
							Recommendation: "Use Atmos variables or external secret management instead of hardcoded values",
							SOC2Controls:   []string{"CC6.1", "CC6.8"},
						})
					}
				}
				break
			}
		}
	}

	return findings
}

// isVariableReference checks if a value is a variable reference (e.g., ${var.something})
func (aa *AtmosAnalyzer) isVariableReference(value string) bool {
	patterns := []string{
		`\$\{.*\}`,   // ${var.something}
		`var\..*`,    // var.something
		`local\..*`,  // local.something
		`data\..*`,   // data.something
		`module\..*`, // module.something
	}

	for _, pattern := range patterns {
		matched, err := regexp.MatchString(pattern, value)
		if err == nil && matched {
			return true
		}
	}

	return false
}

// analyzeResourceSecurity analyzes Terraform resources for security issues
func (aa *AtmosAnalyzer) analyzeResourceSecurity(resource models.TerraformScanResult, securityFocus string) []SecurityFinding {
	var findings []SecurityFinding

	// Security analysis based on focus area
	if securityFocus == "all" || securityFocus == "encryption" {
		findings = append(findings, aa.analyzeEncryptionCompliance(resource)...)
	}

	if securityFocus == "all" || securityFocus == "network" {
		findings = append(findings, aa.analyzeNetworkSecurity(resource)...)
	}

	if securityFocus == "all" || securityFocus == "iam" {
		findings = append(findings, aa.analyzeIAMSecurity(resource)...)
	}

	if securityFocus == "all" || securityFocus == "monitoring" {
		findings = append(findings, aa.analyzeMonitoringSecurity(resource)...)
	}

	return findings
}

// analyzeEncryptionCompliance analyzes encryption-related security
func (aa *AtmosAnalyzer) analyzeEncryptionCompliance(resource models.TerraformScanResult) []SecurityFinding {
	var findings []SecurityFinding
	resourceType := strings.ToLower(resource.ResourceType)

	// S3 encryption checks
	if strings.Contains(resourceType, "s3_bucket") && !strings.Contains(resourceType, "encryption") {
		findings = append(findings, SecurityFinding{
			Type:           "encryption",
			Severity:       "high",
			Description:    "S3 bucket may not have encryption configured",
			Recommendation: "Ensure S3 bucket has server-side encryption enabled",
			SOC2Controls:   []string{"CC6.8"},
		})
	}

	// RDS encryption checks
	if strings.Contains(resourceType, "rds") || strings.Contains(resourceType, "db_instance") {
		if !aa.hasConfigValue(resource.Configuration, "storage_encrypted") {
			findings = append(findings, SecurityFinding{
				Type:           "encryption",
				Severity:       "high",
				Description:    "RDS instance does not have encryption at rest enabled",
				Recommendation: "Enable storage_encrypted = true for RDS instances",
				SOC2Controls:   []string{"CC6.8"},
			})
		}
	}

	return findings
}

// analyzeNetworkSecurity analyzes network security configurations
func (aa *AtmosAnalyzer) analyzeNetworkSecurity(resource models.TerraformScanResult) []SecurityFinding {
	var findings []SecurityFinding
	resourceType := strings.ToLower(resource.ResourceType)

	// Security group checks
	if strings.Contains(resourceType, "security_group") {
		if aa.hasOpenIngress(resource.Configuration) {
			findings = append(findings, SecurityFinding{
				Type:           "network",
				Severity:       "high",
				Description:    "Security group allows unrestricted ingress (0.0.0.0/0)",
				Recommendation: "Restrict ingress to specific IP ranges or security groups",
				SOC2Controls:   []string{"CC6.6", "CC7.1"},
			})
		}
	}

	return findings
}

// analyzeIAMSecurity analyzes IAM security configurations
func (aa *AtmosAnalyzer) analyzeIAMSecurity(resource models.TerraformScanResult) []SecurityFinding {
	var findings []SecurityFinding
	resourceType := strings.ToLower(resource.ResourceType)

	// IAM policy checks
	if strings.Contains(resourceType, "iam_policy") {
		if aa.hasWildcardPermissions(resource.Configuration) {
			findings = append(findings, SecurityFinding{
				Type:           "iam",
				Severity:       "medium",
				Description:    "IAM policy contains wildcard permissions",
				Recommendation: "Use principle of least privilege with specific permissions",
				SOC2Controls:   []string{"CC6.1", "CC6.3"},
			})
		}
	}

	return findings
}

// analyzeMonitoringSecurity analyzes monitoring and logging configurations
func (aa *AtmosAnalyzer) analyzeMonitoringSecurity(resource models.TerraformScanResult) []SecurityFinding {
	var findings []SecurityFinding
	resourceType := strings.ToLower(resource.ResourceType)

	// CloudTrail checks
	if strings.Contains(resourceType, "cloudtrail") {
		if !aa.hasConfigValue(resource.Configuration, "enable_logging") {
			findings = append(findings, SecurityFinding{
				Type:           "monitoring",
				Severity:       "medium",
				Description:    "CloudTrail may not have logging enabled",
				Recommendation: "Ensure CloudTrail logging is enabled for audit compliance",
				SOC2Controls:   []string{"CC7.2", "CC7.4"},
			})
		}
	}

	return findings
}

// analyzeBackendSecurity analyzes backend configuration security
func (aa *AtmosAnalyzer) analyzeBackendSecurity(stack AtmosStack) []SecurityFinding {
	var findings []SecurityFinding

	if stack.Backend == nil {
		findings = append(findings, SecurityFinding{
			Type:           "backend",
			Severity:       "medium",
			Description:    "No backend configuration found - state may not be securely stored",
			Recommendation: "Configure remote backend with encryption for Terraform state",
			SOC2Controls:   []string{"CC6.8", "CC7.2"},
		})
		return findings
	}

	// Check for S3 backend encryption
	if backendType, ok := stack.Backend["type"].(string); ok && backendType == "s3" {
		if encrypt, ok := stack.Backend["encrypt"].(bool); !ok || !encrypt {
			findings = append(findings, SecurityFinding{
				Type:           "backend",
				Severity:       "high",
				Description:    "S3 backend does not have encryption enabled",
				Recommendation: "Enable encrypt = true in S3 backend configuration",
				SOC2Controls:   []string{"CC6.8"},
			})
		}
	}

	return findings
}

// Helper functions (reused from security analyzer)
func (aa *AtmosAnalyzer) hasConfigValue(config map[string]interface{}, key string) bool {
	value, exists := config[key]
	if !exists {
		return false
	}

	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != "" && v != "false"
	default:
		return value != nil
	}
}

func (aa *AtmosAnalyzer) hasWildcardPermissions(config map[string]interface{}) bool {
	for key, value := range config {
		if strings.Contains(strings.ToLower(key), "policy") {
			if valueStr, ok := value.(string); ok {
				if strings.Contains(valueStr, "*") {
					return true
				}
			}
		}
	}
	return false
}

func (aa *AtmosAnalyzer) hasOpenIngress(config map[string]interface{}) bool {
	for key, value := range config {
		if strings.Contains(strings.ToLower(key), "cidr") || strings.Contains(strings.ToLower(key), "ingress") {
			if valueStr, ok := value.(string); ok {
				if strings.Contains(valueStr, "0.0.0.0/0") {
					return true
				}
			}
		}
	}
	return false
}

// mapStacksToCompliance maps stacks to compliance frameworks
func (aa *AtmosAnalyzer) mapStacksToCompliance(stacks []AtmosStack, frameworks []string, mapping map[string][]AtmosStack) {
	for _, stack := range stacks {
		for _, framework := range frameworks {
			// Map based on security findings and resource types
			isRelevant := false

			for _, resource := range stack.TerraformResources {
				if aa.isResourceRelevantToFramework(resource, framework) {
					isRelevant = true
					break
				}
			}

			if isRelevant {
				mapping[framework] = append(mapping[framework], stack)
			}
		}
	}
}

// isResourceRelevantToFramework checks if a resource is relevant to a compliance framework
func (aa *AtmosAnalyzer) isResourceRelevantToFramework(resource models.TerraformScanResult, framework string) bool {
	frameworkLower := strings.ToLower(framework)
	resourceType := strings.ToLower(resource.ResourceType)

	switch frameworkLower {
	case "soc2":
		return strings.Contains(resourceType, "encrypt") ||
			strings.Contains(resourceType, "iam") ||
			strings.Contains(resourceType, "security_group") ||
			strings.Contains(resourceType, "cloudtrail") ||
			strings.Contains(resourceType, "backup")
	case "iso27001":
		return strings.Contains(resourceType, "encrypt") ||
			strings.Contains(resourceType, "iam") ||
			strings.Contains(resourceType, "monitor") ||
			strings.Contains(resourceType, "log")
	case "pci":
		return strings.Contains(resourceType, "encrypt") ||
			strings.Contains(resourceType, "security_group") ||
			strings.Contains(resourceType, "waf") ||
			strings.Contains(resourceType, "lb")
	default:
		return true // Include all resources for unknown frameworks
	}
}

// calculateComplianceCoverage calculates compliance coverage score
func (aa *AtmosAnalyzer) calculateComplianceCoverage(stacks []AtmosStack, frameworks []string) float64 {
	if len(frameworks) == 0 {
		return 1.0 // Full coverage if no specific frameworks requested
	}

	totalRequiredControls := len(frameworks) * 10 // Assume 10 key controls per framework
	coveredControls := 0

	for _, stack := range stacks {
		for _, resource := range stack.TerraformResources {
			for _, framework := range frameworks {
				if aa.isResourceRelevantToFramework(resource, framework) {
					coveredControls++
					break // Count once per stack per framework
				}
			}
		}
	}

	if totalRequiredControls == 0 {
		return 1.0
	}

	coverage := float64(coveredControls) / float64(totalRequiredControls)
	if coverage > 1.0 {
		coverage = 1.0
	}

	return coverage
}

// analyzeDriftBetweenEnvironments analyzes configuration drift between environments
func (aa *AtmosAnalyzer) analyzeDriftBetweenEnvironments(envMatrix map[string]map[string]AtmosStack) []ConfigurationDrift {
	var driftIssues []ConfigurationDrift

	// Get list of environments
	var environments []string
	for env := range envMatrix {
		environments = append(environments, env)
	}
	sort.Strings(environments)

	// Compare stacks across environments
	if len(environments) < 2 {
		return driftIssues // Need at least 2 environments for drift analysis
	}

	// Get all unique stack names
	stackNames := make(map[string]bool)
	for _, envStacks := range envMatrix {
		for stackName := range envStacks {
			stackNames[stackName] = true
		}
	}

	// Analyze each stack across environments
	for stackName := range stackNames {
		baslineEnv := environments[0]
		baselineStack, hasBaseline := envMatrix[baslineEnv][stackName]

		if !hasBaseline {
			continue
		}

		for i := 1; i < len(environments); i++ {
			compareEnv := environments[i]
			compareStack, hasCompare := envMatrix[compareEnv][stackName]

			if !hasCompare {
				driftIssues = append(driftIssues, ConfigurationDrift{
					StackName:           stackName,
					BaselineEnvironment: baslineEnv,
					CompareEnvironment:  compareEnv,
					DriftType:           "missing_stack",
					Description:         fmt.Sprintf("Stack %s exists in %s but not in %s", stackName, baslineEnv, compareEnv),
					Severity:            "medium",
					Recommendation:      "Ensure consistent stack deployment across environments",
				})
				continue
			}

			// Compare configurations
			drifts := aa.compareStackConfigurations(baselineStack, compareStack)
			driftIssues = append(driftIssues, drifts...)
		}
	}

	return driftIssues
}

// compareStackConfigurations compares two stack configurations for drift
func (aa *AtmosAnalyzer) compareStackConfigurations(baseline, compare AtmosStack) []ConfigurationDrift {
	var drifts []ConfigurationDrift

	// Compare components
	if baseline.Component != compare.Component {
		drifts = append(drifts, ConfigurationDrift{
			StackName:           baseline.StackName,
			BaselineEnvironment: baseline.Environment,
			CompareEnvironment:  compare.Environment,
			DriftType:           "component_mismatch",
			Description:         fmt.Sprintf("Component differs: %s vs %s", baseline.Component, compare.Component),
			Severity:            "high",
			Recommendation:      "Ensure consistent component usage across environments",
		})
	}

	// Compare variables (simplified - check for missing/extra keys)
	baselineKeys := make(map[string]bool)
	for key := range baseline.Variables {
		baselineKeys[key] = true
	}

	compareKeys := make(map[string]bool)
	for key := range compare.Variables {
		compareKeys[key] = true
	}

	// Check for missing variables
	for key := range baselineKeys {
		if !compareKeys[key] {
			drifts = append(drifts, ConfigurationDrift{
				StackName:           baseline.StackName,
				BaselineEnvironment: baseline.Environment,
				CompareEnvironment:  compare.Environment,
				DriftType:           "missing_variable",
				Description:         fmt.Sprintf("Variable %s exists in %s but not in %s", key, baseline.Environment, compare.Environment),
				Severity:            "medium",
				Recommendation:      "Ensure consistent variable configuration across environments",
			})
		}
	}

	// Check for extra variables
	for key := range compareKeys {
		if !baselineKeys[key] {
			drifts = append(drifts, ConfigurationDrift{
				StackName:           baseline.StackName,
				BaselineEnvironment: baseline.Environment,
				CompareEnvironment:  compare.Environment,
				DriftType:           "extra_variable",
				Description:         fmt.Sprintf("Variable %s exists in %s but not in %s", key, compare.Environment, baseline.Environment),
				Severity:            "medium",
				Recommendation:      "Ensure consistent variable configuration across environments",
			})
		}
	}

	return drifts
}

// Utility functions

func (aa *AtmosAnalyzer) getAtmosPath() string {
	// Use configured Atmos path or default
	if aa.config != nil && len(aa.config.ScanPaths) > 0 {
		return aa.config.ScanPaths[0]
	}
	return "deploy/atmos" // Default Atmos path
}

func (aa *AtmosAnalyzer) stringInSlice(target string, slice []string) bool {
	for _, item := range slice {
		if item == target {
			return true
		}
	}
	return false
}

func (aa *AtmosAnalyzer) countSecurityFindings(stacks []AtmosStack) int {
	count := 0
	for _, stack := range stacks {
		count += len(stack.SecurityFindings)
	}
	return count
}

func (aa *AtmosAnalyzer) calculateAtmosRelevance(analysis *AtmosStackAnalysis) float64 {
	relevance := 0.5 // Base score

	// Boost for number of stacks analyzed
	if len(analysis.Stacks) >= 10 {
		relevance += 0.2
	} else if len(analysis.Stacks) >= 5 {
		relevance += 0.1
	}

	// Boost for multi-environment coverage
	if len(analysis.Environments) >= 3 {
		relevance += 0.2
	} else if len(analysis.Environments) >= 2 {
		relevance += 0.1
	}

	// Reduce for drift issues
	if len(analysis.DriftIssues) > 0 {
		relevance -= float64(len(analysis.DriftIssues)) * 0.02
	}

	// Boost for compliance coverage
	relevance += analysis.ComplianceCoverage * 0.2

	// Cap at 1.0
	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}

// generateAtmosReport generates the analysis report in the requested format
func (aa *AtmosAnalyzer) generateAtmosReport(analysis *AtmosStackAnalysis, format string) (string, error) {
	switch format {
	case "detailed_json":
		return aa.generateDetailedJSONReport(analysis)
	case "summary_markdown":
		return aa.generateSummaryMarkdownReport(analysis)
	case "compliance_csv":
		return aa.generateComplianceCSVReport(analysis)
	case "drift_report":
		return aa.generateDriftReport(analysis)
	default:
		return "", fmt.Errorf("unsupported output format: %s", format)
	}
}

// Report generation methods (implement basic versions for now)
func (aa *AtmosAnalyzer) generateDetailedJSONReport(analysis *AtmosStackAnalysis) (string, error) {
	// Marshal the entire analysis to JSON
	data, err := yaml.Marshal(analysis) // Using YAML for better readability
	if err != nil {
		return "", fmt.Errorf("failed to marshal analysis: %w", err)
	}
	return string(data), nil
}

func (aa *AtmosAnalyzer) generateSummaryMarkdownReport(analysis *AtmosStackAnalysis) (string, error) {
	var report strings.Builder

	report.WriteString("# Atmos Stack Security Analysis\n\n")
	report.WriteString(fmt.Sprintf("**Analysis Date:** %s\n", analysis.AnalysisTimestamp.Format(time.RFC3339)))
	report.WriteString(fmt.Sprintf("**Environments:** %d (%s)\n", len(analysis.Environments), strings.Join(analysis.Environments, ", ")))
	report.WriteString(fmt.Sprintf("**Stacks:** %d\n", len(analysis.Stacks)))
	report.WriteString(fmt.Sprintf("**Compliance Coverage:** %.1f%%\n\n", analysis.ComplianceCoverage*100))

	// Security findings summary
	if len(analysis.SecurityFindings) > 0 {
		report.WriteString("## Security Findings\n\n")
		for _, finding := range analysis.SecurityFindings {
			report.WriteString(fmt.Sprintf("### %s - %s (%s)\n", finding.StackName, finding.Environment, finding.SecurityFinding.Severity))
			report.WriteString(fmt.Sprintf("**Type:** %s\n", finding.SecurityFinding.Type))
			report.WriteString(fmt.Sprintf("**Description:** %s\n", finding.SecurityFinding.Description))
			report.WriteString(fmt.Sprintf("**Recommendation:** %s\n\n", finding.SecurityFinding.Recommendation))
		}
	}

	// Drift issues
	if len(analysis.DriftIssues) > 0 {
		report.WriteString("## Configuration Drift Issues\n\n")
		for _, drift := range analysis.DriftIssues {
			report.WriteString(fmt.Sprintf("### %s (%s)\n", drift.StackName, drift.Severity))
			report.WriteString(fmt.Sprintf("**Environments:** %s → %s\n", drift.BaselineEnvironment, drift.CompareEnvironment))
			report.WriteString(fmt.Sprintf("**Description:** %s\n", drift.Description))
			report.WriteString(fmt.Sprintf("**Recommendation:** %s\n\n", drift.Recommendation))
		}
	}

	return report.String(), nil
}

func (aa *AtmosAnalyzer) generateComplianceCSVReport(analysis *AtmosStackAnalysis) (string, error) {
	var report strings.Builder

	// CSV Header
	report.WriteString("Stack Name,Environment,Component,Security Findings,Compliance Score,Drift Issues\n")

	for _, stack := range analysis.Stacks {
		findingsCount := len(stack.SecurityFindings)

		// Calculate compliance score for this stack
		complianceScore := 1.0
		if findingsCount > 0 {
			highSeverityCount := 0
			for _, finding := range stack.SecurityFindings {
				if finding.Severity == "high" {
					highSeverityCount++
				}
			}
			complianceScore = 1.0 - (float64(highSeverityCount) * 0.2)
			if complianceScore < 0 {
				complianceScore = 0
			}
		}

		// Count drift issues for this stack
		driftCount := 0
		for _, drift := range analysis.DriftIssues {
			if drift.StackName == stack.StackName {
				driftCount++
			}
		}

		report.WriteString(fmt.Sprintf("%s,%s,%s,%d,%.2f,%d\n",
			aa.escapeCSV(stack.StackName),
			aa.escapeCSV(stack.Environment),
			aa.escapeCSV(stack.Component),
			findingsCount,
			complianceScore,
			driftCount))
	}

	return report.String(), nil
}

func (aa *AtmosAnalyzer) generateDriftReport(analysis *AtmosStackAnalysis) (string, error) {
	var report strings.Builder

	report.WriteString("# Configuration Drift Analysis\n\n")
	report.WriteString(fmt.Sprintf("**Analysis Date:** %s\n", analysis.AnalysisTimestamp.Format(time.RFC3339)))
	report.WriteString(fmt.Sprintf("**Environments Compared:** %s\n", strings.Join(analysis.Environments, ", ")))
	report.WriteString(fmt.Sprintf("**Total Drift Issues:** %d\n\n", len(analysis.DriftIssues)))

	if len(analysis.DriftIssues) == 0 {
		report.WriteString("No configuration drift detected between environments.\n")
		return report.String(), nil
	}

	// Group drift issues by stack
	driftByStack := make(map[string][]ConfigurationDrift)
	for _, drift := range analysis.DriftIssues {
		driftByStack[drift.StackName] = append(driftByStack[drift.StackName], drift)
	}

	for stackName, drifts := range driftByStack {
		report.WriteString(fmt.Sprintf("## %s\n\n", stackName))
		for _, drift := range drifts {
			report.WriteString(fmt.Sprintf("### %s (%s severity)\n", drift.DriftType, drift.Severity))
			report.WriteString(fmt.Sprintf("**Environments:** %s → %s\n", drift.BaselineEnvironment, drift.CompareEnvironment))
			report.WriteString(fmt.Sprintf("**Description:** %s\n", drift.Description))
			report.WriteString(fmt.Sprintf("**Recommendation:** %s\n\n", drift.Recommendation))
		}
	}

	return report.String(), nil
}

func (aa *AtmosAnalyzer) escapeCSV(value string) string {
	if strings.Contains(value, ",") || strings.Contains(value, "\n") || strings.Contains(value, "\"") {
		value = strings.ReplaceAll(value, "\"", "\"\"")
		return "\"" + value + "\""
	}
	return value
}
