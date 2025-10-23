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
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// SecurityAnalyzer provides comprehensive security configuration analysis for Terraform manifests
type SecurityAnalyzer struct {
	config      *config.TerraformToolConfig
	logger      logger.Logger
	baseScanner *Analyzer                 // Use the new consolidated Analyzer instead of TerraformTool
	indexer     *SecurityAttributeIndexer // Index-first architecture for fast queries
}

// NewSecurityAnalyzer creates a new Terraform Security Analyzer
func NewSecurityAnalyzer(cfg *config.Config, log logger.Logger) *SecurityAnalyzer {
	return &SecurityAnalyzer{
		config:      &cfg.Evidence.Tools.Terraform,
		logger:      log,
		baseScanner: NewAnalyzer(cfg, log),
		indexer:     NewSecurityAttributeIndexer(cfg, log),
	}
}

// Name returns the tool name
func (tsa *SecurityAnalyzer) Name() string {
	return "terraform-security-analyzer"
}

// Description returns the tool description
func (tsa *SecurityAnalyzer) Description() string {
	return "Comprehensive security configuration analyzer for Terraform manifests with SOC2 control mapping"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (tsa *SecurityAnalyzer) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        tsa.Name(),
		Description: tsa.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"security_domain": map[string]interface{}{
					"type":        "string",
					"description": "Security domain to focus on: encryption, iam, network, backup, monitoring, or all",
					"enum":        []string{"encryption", "iam", "network", "backup", "monitoring", "all"},
					"default":     "all",
				},
				"soc2_controls": map[string]interface{}{
					"type":        "array",
					"description": "Specific SOC2 controls to find evidence for (e.g., [\"CC6.1\", \"CC6.8\"])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"evidence_tasks": map[string]interface{}{
					"type":        "array",
					"description": "Evidence task IDs to address (e.g., [\"ET21\", \"ET23\", \"ET47\", \"ET71\", \"ET103\"])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"include_compliance_gaps": map[string]interface{}{
					"type":        "boolean",
					"description": "Include analysis of potential compliance gaps and recommendations",
					"default":     true,
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "Output format: detailed_json, summary_markdown, or compliance_csv",
					"enum":        []string{"detailed_json", "summary_markdown", "compliance_csv"},
					"default":     "detailed_json",
				},
				"extract_sensitive_configs": map[string]interface{}{
					"type":        "boolean",
					"description": "Extract detailed security configurations (excludes actual secrets)",
					"default":     true,
				},
				"skip_cache": map[string]interface{}{
					"type":        "boolean",
					"description": "Skip cached index and force live scan (default: false)",
					"default":     false,
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs the Terraform security analyzer with the given parameters
func (tsa *SecurityAnalyzer) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	tsa.logger.Debug("Executing Terraform security analyzer", logger.Field{Key: "params", Value: params})

	// Extract parameters
	securityDomain := "all"
	if sd, ok := params["security_domain"].(string); ok {
		securityDomain = sd
	}

	var soc2Controls []string
	if sc, ok := params["soc2_controls"].([]interface{}); ok {
		for _, c := range sc {
			if str, ok := c.(string); ok {
				soc2Controls = append(soc2Controls, str)
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

	includeComplianceGaps := true
	if icg, ok := params["include_compliance_gaps"].(bool); ok {
		includeComplianceGaps = icg
	}

	outputFormat := "detailed_json"
	if of, ok := params["output_format"].(string); ok {
		outputFormat = of
	}

	extractSensitiveConfigs := true
	if esc, ok := params["extract_sensitive_configs"].(bool); ok {
		extractSensitiveConfigs = esc
	}

	skipCache := false
	if sc, ok := params["skip_cache"].(bool); ok {
		skipCache = sc
	}

	// Perform security analysis using index
	securityAnalysis, err := tsa.performSecurityAnalysis(ctx, securityDomain, soc2Controls, evidenceTasks, extractSensitiveConfigs, skipCache)
	if err != nil {
		return "", nil, fmt.Errorf("failed to perform security analysis: %w", err)
	}

	// Add compliance gap analysis if requested
	if includeComplianceGaps {
		securityAnalysis.ComplianceGaps = tsa.analyzeComplianceGaps(securityAnalysis)
	}

	// Generate report based on format
	report, err := tsa.generateSecurityReport(securityAnalysis, outputFormat)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate security report: %w", err)
	}

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "terraform-security-analysis",
		Resource:    fmt.Sprintf("Security analysis of %d resources across %d files", len(securityAnalysis.SecurityResources), len(securityAnalysis.FilesAnalyzed)),
		Content:     report,
		Relevance:   tsa.calculateSecurityRelevance(securityAnalysis),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"security_domain":           securityDomain,
			"soc2_controls":             soc2Controls,
			"evidence_tasks":            evidenceTasks,
			"resources_analyzed":        len(securityAnalysis.SecurityResources),
			"files_analyzed":            len(securityAnalysis.FilesAnalyzed),
			"encryption_configs":        len(securityAnalysis.EncryptionConfigs),
			"iam_configs":               len(securityAnalysis.IAMConfigs),
			"network_configs":           len(securityAnalysis.NetworkConfigs),
			"backup_configs":            len(securityAnalysis.BackupConfigs),
			"monitoring_configs":        len(securityAnalysis.MonitoringConfigs),
			"compliance_gaps_found":     len(securityAnalysis.ComplianceGaps),
			"include_compliance_gaps":   includeComplianceGaps,
			"extract_sensitive_configs": extractSensitiveConfigs,
		},
	}

	return report, source, nil
}

// performSecurityAnalysis performs comprehensive security configuration analysis
func (tsa *SecurityAnalyzer) performSecurityAnalysis(ctx context.Context, domain string, soc2Controls, evidenceTasks []string, extractSensitive bool, skipCache bool) (*SecurityAnalysisResult, error) {
	// Load or build index (fast path: use cached index)
	persistedIndex, err := tsa.indexer.LoadOrBuildIndex(ctx, skipCache)
	if err != nil {
		// Fallback to live scan if index fails
		tsa.logger.Warn("Failed to load index, falling back to live scan",
			logger.Field{Key: "error", Value: err})
		return tsa.performLiveScan(ctx, domain, soc2Controls, evidenceTasks, extractSensitive)
	}

	// Use index query layer for fast filtering
	query := NewIndexQuery(persistedIndex)
	var indexedResources []IndexedResource

	// Query based on requested controls or evidence tasks
	if len(soc2Controls) > 0 {
		result := query.ByControl(soc2Controls...)
		indexedResources = result.Resources
		tsa.logger.Debug("Queried index by controls",
			logger.Int("results", result.Count),
			logger.Duration("query_time", result.QueryTime))
	} else if len(evidenceTasks) > 0 {
		// Query by evidence tasks
		var allResults []*QueryResult
		for _, taskRef := range evidenceTasks {
			allResults = append(allResults, query.ByEvidenceTask(taskRef))
		}
		unionResult := Union(allResults...)
		indexedResources = unionResult.Resources
		tsa.logger.Debug("Queried index by evidence tasks",
			logger.Int("results", unionResult.Count),
			logger.Duration("query_time", unionResult.QueryTime))
	} else {
		// Get all resources
		indexedResources = persistedIndex.Index.IndexedResources
	}

	// Convert indexed resources to TerraformScanResult for analysis
	allResults := tsa.convertIndexedToScanResults(indexedResources)

	tsa.logger.Info("Using cached index",
		logger.Int("indexed_resources", len(indexedResources)))

	// Process resources using shared logic
	return tsa.processResources(allResults, domain, soc2Controls, evidenceTasks, extractSensitive)
}

// performLiveScan performs a live scan when index is unavailable (fallback)
func (tsa *SecurityAnalyzer) performLiveScan(ctx context.Context, domain string, soc2Controls, evidenceTasks []string, extractSensitive bool) (*SecurityAnalysisResult, error) {
	// Get all terraform resources via live scan
	allResults, err := tsa.baseScanner.ScanForResources(ctx, []string{})
	if err != nil {
		return nil, fmt.Errorf("failed to scan terraform resources: %w", err)
	}

	tsa.logger.Info("Performing live scan",
		logger.Int("resources_scanned", len(allResults)))

	return tsa.processResources(allResults, domain, soc2Controls, evidenceTasks, extractSensitive)
}

// convertIndexedToScanResults converts indexed resources back to scan results
func (tsa *SecurityAnalyzer) convertIndexedToScanResults(indexed []IndexedResource) []models.TerraformScanResult {
	results := make([]models.TerraformScanResult, len(indexed))

	for i, res := range indexed {
		results[i] = models.TerraformScanResult{
			ResourceType:      res.ResourceType,
			ResourceName:      res.ResourceName,
			FilePath:          res.FilePath,
			LineStart:         0, // Not stored in index
			LineEnd:           0, // Not stored in index
			Configuration:     res.Configuration,
			SecurityRelevance: res.ControlRelevance,
		}
	}

	return results
}

// processResources processes a list of scan results into security analysis
func (tsa *SecurityAnalyzer) processResources(allResults []models.TerraformScanResult, domain string, soc2Controls, evidenceTasks []string, extractSensitive bool) (*SecurityAnalysisResult, error) {
	analysis := &SecurityAnalysisResult{
		AnalysisTimestamp:   time.Now(),
		SecurityDomain:      domain,
		RequestedControls:   soc2Controls,
		RequestedTasks:      evidenceTasks,
		SecurityResources:   []SecurityResource{},
		EncryptionConfigs:   []EncryptionConfig{},
		IAMConfigs:          []IAMConfig{},
		NetworkConfigs:      []NetworkConfig{},
		BackupConfigs:       []BackupConfig{},
		MonitoringConfigs:   []MonitoringConfig{},
		FilesAnalyzed:       []string{},
		SOC2ControlMapping:  make(map[string][]SecurityResource),
		EvidenceTaskMapping: make(map[string][]SecurityResource),
	}

	// Track unique files
	fileSet := make(map[string]bool)

	// Process each terraform resource for security configurations
	for _, result := range allResults {
		fileSet[result.FilePath] = true

		// Check if resource is security-relevant for the requested domain
		if !tsa.isSecurityRelevant(result, domain) {
			continue
		}

		// Extract security configurations based on resource type
		securityResource := tsa.extractSecurityResource(result, extractSensitive)
		analysis.SecurityResources = append(analysis.SecurityResources, securityResource)

		// Extract domain-specific configurations
		if domain == "all" || domain == "encryption" {
			if encConfig := tsa.extractEncryptionConfig(result); encConfig != nil {
				analysis.EncryptionConfigs = append(analysis.EncryptionConfigs, *encConfig)
			}
		}

		if domain == "all" || domain == "iam" {
			if iamConfig := tsa.extractIAMConfig(result); iamConfig != nil {
				analysis.IAMConfigs = append(analysis.IAMConfigs, *iamConfig)
			}
		}

		if domain == "all" || domain == "network" {
			if netConfig := tsa.extractNetworkConfig(result); netConfig != nil {
				analysis.NetworkConfigs = append(analysis.NetworkConfigs, *netConfig)
			}
		}

		if domain == "all" || domain == "backup" {
			if backupConfig := tsa.extractBackupConfig(result); backupConfig != nil {
				analysis.BackupConfigs = append(analysis.BackupConfigs, *backupConfig)
			}
		}

		if domain == "all" || domain == "monitoring" {
			if monConfig := tsa.extractMonitoringConfig(result); monConfig != nil {
				analysis.MonitoringConfigs = append(analysis.MonitoringConfigs, *monConfig)
			}
		}

		// Map to SOC2 controls
		tsa.mapToControls(securityResource, analysis)
	}

	// Populate files analyzed
	for file := range fileSet {
		analysis.FilesAnalyzed = append(analysis.FilesAnalyzed, file)
	}
	sort.Strings(analysis.FilesAnalyzed)

	return analysis, nil
}

func (tsa *SecurityAnalyzer) isSecurityRelevant(result models.TerraformScanResult, domain string) bool {
	resourceType := strings.ToLower(result.ResourceType)

	// Define security-relevant resource patterns by domain
	securityPatterns := map[string][]string{
		"encryption": {
			"aws_kms_", "aws_s3_bucket_encryption", "aws_ebs_encryption", "aws_rds_cluster",
			"azurerm_key_vault", "google_kms_", "aws_acm_certificate",
		},
		"iam": {
			"aws_iam_", "aws_sts_", "azurerm_role_", "azurerm_active_directory",
			"google_project_iam", "google_service_account",
		},
		"network": {
			"aws_vpc", "aws_security_group", "aws_nacl", "aws_subnet", "aws_route",
			"aws_internet_gateway", "aws_nat_gateway", "aws_lb", "aws_cloudfront",
			"azurerm_virtual_network", "azurerm_network_security_group",
			"google_compute_network", "google_compute_firewall",
		},
		"backup": {
			"aws_backup_", "aws_s3_bucket_versioning", "aws_db_snapshot",
			"azurerm_backup_", "google_compute_snapshot",
		},
		"monitoring": {
			"aws_cloudtrail", "aws_cloudwatch", "aws_config_", "aws_guardduty",
			"azurerm_monitor_", "azurerm_log_analytics", "google_logging",
			"google_monitoring",
		},
	}

	if domain == "all" {
		// Check all domains
		for _, patterns := range securityPatterns {
			for _, pattern := range patterns {
				if strings.Contains(resourceType, pattern) {
					return true
				}
			}
		}
	} else {
		// Check specific domain
		if patterns, exists := securityPatterns[domain]; exists {
			for _, pattern := range patterns {
				if strings.Contains(resourceType, pattern) {
					return true
				}
			}
		}
	}

	return false
}

// extractSecurityResource extracts a generic security resource configuration
func (tsa *SecurityAnalyzer) extractSecurityResource(result models.TerraformScanResult, extractSensitive bool) SecurityResource {
	secResource := SecurityResource{
		ResourceType:      result.ResourceType,
		ResourceName:      result.ResourceName,
		FilePath:          result.FilePath,
		LineRange:         fmt.Sprintf("%d-%d", result.LineStart, result.LineEnd),
		SecurityRelevance: result.SecurityRelevance,
		Configuration:     make(map[string]interface{}),
		SecurityFindings:  []SecurityFinding{},
	}

	// Extract security-relevant configuration items
	for key, value := range result.Configuration {
		if key == "_content" {
			continue // Skip raw content
		}

		// Check if configuration item is security-relevant
		if tsa.isSecurityRelevantConfig(key, value) {
			if extractSensitive || !tsa.isSensitiveConfig(key, value) {
				secResource.Configuration[key] = value
			} else {
				secResource.Configuration[key] = "[REDACTED]"
			}
		}
	}

	// Analyze for security findings
	secResource.SecurityFindings = tsa.analyzeSecurityFindings(result)

	return secResource
}

// isSecurityRelevantConfig checks if a configuration key-value pair is security-relevant
func (tsa *SecurityAnalyzer) isSecurityRelevantConfig(key string, value interface{}) bool {
	keyLower := strings.ToLower(key)

	securityKeywords := []string{
		"policy", "permission", "role", "access", "security", "encrypt", "kms",
		"ssl", "tls", "certificate", "auth", "token", "key", "secret", "password",
		"firewall", "acl", "backup", "snapshot", "logging", "monitor", "audit",
		"compliance", "retention", "versioning", "mfa", "2fa",
	}

	for _, keyword := range securityKeywords {
		if strings.Contains(keyLower, keyword) {
			return true
		}
	}

	// Check value for security keywords if it's a string
	if valueStr, ok := value.(string); ok {
		valueLower := strings.ToLower(valueStr)
		for _, keyword := range securityKeywords {
			if strings.Contains(valueLower, keyword) {
				return true
			}
		}
	}

	return false
}

// isSensitiveConfig checks if a configuration contains sensitive information
func (tsa *SecurityAnalyzer) isSensitiveConfig(key string, value interface{}) bool {
	keyLower := strings.ToLower(key)

	sensitiveKeywords := []string{
		"password", "secret", "token", "key", "private", "credential",
	}

	for _, keyword := range sensitiveKeywords {
		if strings.Contains(keyLower, keyword) {
			return true
		}
	}

	return false
}

// analyzeSecurityFindings analyzes a resource for security findings
func (tsa *SecurityAnalyzer) analyzeSecurityFindings(result models.TerraformScanResult) []SecurityFinding {
	var findings []SecurityFinding

	// Define security checks based on resource type
	resourceType := strings.ToLower(result.ResourceType)

	// Encryption findings
	if strings.Contains(resourceType, "s3_bucket") {
		if !tsa.hasConfigValue(result.Configuration, "server_side_encryption") {
			findings = append(findings, SecurityFinding{
				Type:           "encryption",
				Severity:       "high",
				Description:    "S3 bucket does not have server-side encryption configured",
				Recommendation: "Enable server-side encryption for S3 bucket",
				SOC2Controls:   []string{"CC6.8"},
			})
		}
	}

	if strings.Contains(resourceType, "rds") || strings.Contains(resourceType, "db_instance") {
		if !tsa.hasConfigValue(result.Configuration, "storage_encrypted") {
			findings = append(findings, SecurityFinding{
				Type:           "encryption",
				Severity:       "high",
				Description:    "RDS instance does not have encryption at rest enabled",
				Recommendation: "Enable storage encryption for RDS instance",
				SOC2Controls:   []string{"CC6.8"},
			})
		}
	}

	// IAM findings
	if strings.Contains(resourceType, "iam_policy") {
		if tsa.hasWildcardPermissions(result.Configuration) {
			findings = append(findings, SecurityFinding{
				Type:           "iam",
				Severity:       "medium",
				Description:    "IAM policy contains wildcard permissions",
				Recommendation: "Use principle of least privilege with specific permissions",
				SOC2Controls:   []string{"CC6.1", "CC6.3"},
			})
		}
	}

	// Network security findings
	if strings.Contains(resourceType, "security_group") {
		if tsa.hasOpenIngress(result.Configuration) {
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

// Helper functions for security analysis
func (tsa *SecurityAnalyzer) hasConfigValue(config map[string]interface{}, key string) bool {
	value, exists := config[key]
	if !exists {
		return false
	}

	// Check if value is truthy
	switch v := value.(type) {
	case bool:
		return v
	case string:
		return v != "" && v != "false"
	default:
		return value != nil
	}
}

func (tsa *SecurityAnalyzer) hasWildcardPermissions(config map[string]interface{}) bool {
	// Check for wildcard permissions in IAM policies
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

func (tsa *SecurityAnalyzer) hasOpenIngress(config map[string]interface{}) bool {
	// Check for open ingress rules (0.0.0.0/0)
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

// Domain-specific configuration extractors
func (tsa *SecurityAnalyzer) extractEncryptionConfig(result models.TerraformScanResult) *EncryptionConfig {
	resourceType := strings.ToLower(result.ResourceType)

	// Only process encryption-related resources
	if !strings.Contains(resourceType, "kms") &&
		!strings.Contains(resourceType, "encrypt") &&
		!strings.Contains(resourceType, "certificate") &&
		!strings.Contains(resourceType, "s3_bucket") &&
		!strings.Contains(resourceType, "rds") &&
		!strings.Contains(resourceType, "ebs") {
		return nil
	}

	config := &EncryptionConfig{
		ResourceType: result.ResourceType,
		ResourceName: result.ResourceName,
		FilePath:     result.FilePath,
		LineRange:    fmt.Sprintf("%d-%d", result.LineStart, result.LineEnd),
	}

	// Extract encryption-specific configuration
	if kmsKeyId, exists := result.Configuration["kms_key_id"]; exists {
		config.KMSKeyID = fmt.Sprintf("%v", kmsKeyId)
	}

	if encrypted, exists := result.Configuration["encrypted"]; exists {
		if encBool, ok := encrypted.(bool); ok {
			config.EncryptionEnabled = encBool
		} else if encStr, ok := encrypted.(string); ok {
			config.EncryptionEnabled = encStr == "true"
		}
	}

	if encType, exists := result.Configuration["server_side_encryption"]; exists {
		config.EncryptionType = fmt.Sprintf("%v", encType)
	}

	if transitEnc, exists := result.Configuration["encrypt_in_transit"]; exists {
		if transitBool, ok := transitEnc.(bool); ok {
			config.EncryptionInTransit = transitBool
		}
	}

	return config
}

func (tsa *SecurityAnalyzer) extractIAMConfig(result models.TerraformScanResult) *IAMConfig {
	resourceType := strings.ToLower(result.ResourceType)

	// Only process IAM-related resources
	if !strings.Contains(resourceType, "iam") && !strings.Contains(resourceType, "role") {
		return nil
	}

	config := &IAMConfig{
		ResourceType: result.ResourceType,
		ResourceName: result.ResourceName,
		FilePath:     result.FilePath,
		LineRange:    fmt.Sprintf("%d-%d", result.LineStart, result.LineEnd),
		Permissions:  []string{},
		Policies:     []string{},
	}

	// Extract IAM-specific configuration
	if assumeRolePolicy, exists := result.Configuration["assume_role_policy"]; exists {
		config.AssumeRolePolicy = fmt.Sprintf("%v", assumeRolePolicy)
	}

	if policy, exists := result.Configuration["policy"]; exists {
		config.Policies = append(config.Policies, fmt.Sprintf("%v", policy))
	}

	if managedPolicyArns, exists := result.Configuration["managed_policy_arns"]; exists {
		config.ManagedPolicyARNs = fmt.Sprintf("%v", managedPolicyArns)
	}

	// Extract permissions from policy documents
	for key, value := range result.Configuration {
		if strings.Contains(strings.ToLower(key), "policy") {
			if valueStr, ok := value.(string); ok {
				permissions := tsa.extractPermissionsFromPolicy(valueStr)
				config.Permissions = append(config.Permissions, permissions...)
			}
		}
	}

	return config
}

func (tsa *SecurityAnalyzer) extractNetworkConfig(result models.TerraformScanResult) *NetworkConfig {
	resourceType := strings.ToLower(result.ResourceType)

	// Only process network-related resources
	if !strings.Contains(resourceType, "vpc") &&
		!strings.Contains(resourceType, "security_group") &&
		!strings.Contains(resourceType, "subnet") &&
		!strings.Contains(resourceType, "nacl") &&
		!strings.Contains(resourceType, "firewall") {
		return nil
	}

	config := &NetworkConfig{
		ResourceType:      result.ResourceType,
		ResourceName:      result.ResourceName,
		FilePath:          result.FilePath,
		LineRange:         fmt.Sprintf("%d-%d", result.LineStart, result.LineEnd),
		IngressRules:      []NetworkRule{},
		EgressRules:       []NetworkRule{},
		SecurityGroups:    []string{},
		AvailabilityZones: []string{},
	}

	// Extract network-specific configuration
	if vpcId, exists := result.Configuration["vpc_id"]; exists {
		config.VPCID = fmt.Sprintf("%v", vpcId)
	}

	if azs, exists := result.Configuration["availability_zones"]; exists {
		config.AvailabilityZones = append(config.AvailabilityZones, fmt.Sprintf("%v", azs))
	}

	// Extract ingress/egress rules
	config.IngressRules = tsa.extractNetworkRules(result.Configuration, "ingress")
	config.EgressRules = tsa.extractNetworkRules(result.Configuration, "egress")

	return config
}

func (tsa *SecurityAnalyzer) extractBackupConfig(result models.TerraformScanResult) *BackupConfig {
	resourceType := strings.ToLower(result.ResourceType)

	// Only process backup-related resources
	if !strings.Contains(resourceType, "backup") &&
		!strings.Contains(resourceType, "snapshot") &&
		!strings.Contains(resourceType, "versioning") {
		return nil
	}

	config := &BackupConfig{
		ResourceType: result.ResourceType,
		ResourceName: result.ResourceName,
		FilePath:     result.FilePath,
		LineRange:    fmt.Sprintf("%d-%d", result.LineStart, result.LineEnd),
	}

	// Extract backup-specific configuration
	if backupEnabled, exists := result.Configuration["backup_enabled"]; exists {
		if backupBool, ok := backupEnabled.(bool); ok {
			config.BackupEnabled = backupBool
		}
	}

	if retentionPeriod, exists := result.Configuration["backup_retention_period"]; exists {
		config.RetentionPeriod = fmt.Sprintf("%v", retentionPeriod)
	}

	if versioningEnabled, exists := result.Configuration["versioning"]; exists {
		if versioningBool, ok := versioningEnabled.(bool); ok {
			config.VersioningEnabled = versioningBool
		}
	}

	return config
}

func (tsa *SecurityAnalyzer) extractMonitoringConfig(result models.TerraformScanResult) *MonitoringConfig {
	resourceType := strings.ToLower(result.ResourceType)

	// Only process monitoring-related resources
	if !strings.Contains(resourceType, "cloudtrail") &&
		!strings.Contains(resourceType, "cloudwatch") &&
		!strings.Contains(resourceType, "log") &&
		!strings.Contains(resourceType, "monitor") &&
		!strings.Contains(resourceType, "guardduty") {
		return nil
	}

	config := &MonitoringConfig{
		ResourceType:      result.ResourceType,
		ResourceName:      result.ResourceName,
		FilePath:          result.FilePath,
		LineRange:         fmt.Sprintf("%d-%d", result.LineStart, result.LineEnd),
		LoggingEnabled:    false,
		MonitoringEnabled: false,
		AlertsConfigured:  false,
	}

	// Extract monitoring-specific configuration
	if loggingEnabled, exists := result.Configuration["enable_logging"]; exists {
		if loggingBool, ok := loggingEnabled.(bool); ok {
			config.LoggingEnabled = loggingBool
		}
	}

	if retention, exists := result.Configuration["retention_in_days"]; exists {
		config.RetentionPeriod = fmt.Sprintf("%v", retention)
	}

	if eventTypes, exists := result.Configuration["event_types"]; exists {
		config.EventTypes = fmt.Sprintf("%v", eventTypes)
	}

	return config
}

// Helper functions
func (tsa *SecurityAnalyzer) extractPermissionsFromPolicy(policyJSON string) []string {
	// Simple regex-based extraction of permissions from JSON policy
	re := regexp.MustCompile(`"Action":\s*"([^"]+)"`)
	matches := re.FindAllStringSubmatch(policyJSON, -1)

	var permissions []string
	for _, match := range matches {
		if len(match) > 1 {
			permissions = append(permissions, match[1])
		}
	}

	return permissions
}

func (tsa *SecurityAnalyzer) extractNetworkRules(config map[string]interface{}, ruleType string) []NetworkRule {
	var rules []NetworkRule

	// Look for ingress/egress rules in configuration
	for key, value := range config {
		if strings.Contains(strings.ToLower(key), ruleType) {
			// Parse network rule (simplified)
			rule := NetworkRule{
				Protocol: "tcp", // Default
				FromPort: "0",
				ToPort:   "65535",
				CIDR:     []string{"0.0.0.0/0"}, // Default - should be flagged as security issue
			}

			if valueStr, ok := value.(string); ok {
				// Parse rule from string representation
				// TODO: Extract protocol, ports, CIDR blocks if valueStr contains "protocol"
				// This is a simplified parser - in production, would use HCL parser
				_ = valueStr // Currently unused, waiting for parser implementation
			}

			rules = append(rules, rule)
		}
	}

	return rules
}

// mapToControls maps security resources to SOC2 controls and evidence tasks
func (tsa *SecurityAnalyzer) mapToControls(resource SecurityResource, analysis *SecurityAnalysisResult) {
	// Map to SOC2 controls based on resource type and security relevance
	for _, control := range resource.SecurityRelevance {
		analysis.SOC2ControlMapping[control] = append(analysis.SOC2ControlMapping[control], resource)
	}

	// Map to evidence tasks based on resource type
	evidenceTaskMappings := map[string][]string{
		"aws_s3_bucket":       {"ET23"},  // Data Encryption at Rest
		"aws_kms_key":         {"ET23"},  // Data Encryption at Rest
		"aws_lb_listener":     {"ET21"},  // Encryption in Transit
		"aws_acm_certificate": {"ET21"},  // Encryption in Transit
		"aws_iam_role":        {"ET47"},  // Infrastructure Password Configuration
		"aws_iam_policy":      {"ET47"},  // Infrastructure Password Configuration
		"aws_security_group":  {"ET71"},  // Firewall Configurations
		"aws_nacl":            {"ET71"},  // Firewall Configurations
		"aws_vpc":             {"ET103"}, // Multi-Availability Zones
		"aws_subnet":          {"ET103"}, // Multi-Availability Zones
	}

	if tasks, exists := evidenceTaskMappings[resource.ResourceType]; exists {
		for _, task := range tasks {
			analysis.EvidenceTaskMapping[task] = append(analysis.EvidenceTaskMapping[task], resource)
		}
	}
}

// analyzeComplianceGaps analyzes for potential compliance gaps
func (tsa *SecurityAnalyzer) analyzeComplianceGaps(analysis *SecurityAnalysisResult) []ComplianceGap {
	var gaps []ComplianceGap

	// Check for missing encryption configurations
	if len(analysis.EncryptionConfigs) == 0 {
		gaps = append(gaps, ComplianceGap{
			Type:          "encryption",
			Severity:      "high",
			Description:   "No encryption configurations found in Terraform manifests",
			SOC2Controls:  []string{"CC6.8"},
			EvidenceTasks: []string{"ET21", "ET23"},
			Recommendations: []string{
				"Implement KMS key management for encryption at rest",
				"Configure SSL/TLS for encryption in transit",
				"Enable S3 bucket encryption",
				"Enable RDS encryption",
			},
		})
	}

	// Check for insufficient IAM configurations
	if len(analysis.IAMConfigs) < 3 {
		gaps = append(gaps, ComplianceGap{
			Type:          "iam",
			Severity:      "medium",
			Description:   "Limited IAM configurations found - may indicate insufficient access controls",
			SOC2Controls:  []string{"CC6.1", "CC6.3"},
			EvidenceTasks: []string{"ET47"},
			Recommendations: []string{
				"Implement comprehensive IAM role-based access control",
				"Use IAM policies with least privilege principles",
				"Configure multi-factor authentication requirements",
			},
		})
	}

	// Check for network security gaps
	if len(analysis.NetworkConfigs) == 0 {
		gaps = append(gaps, ComplianceGap{
			Type:          "network",
			Severity:      "high",
			Description:   "No network security configurations found",
			SOC2Controls:  []string{"CC6.6", "CC7.1"},
			EvidenceTasks: []string{"ET71", "ET103"},
			Recommendations: []string{
				"Configure VPC with proper network segmentation",
				"Implement security groups with restrictive rules",
				"Set up network ACLs for additional security layers",
				"Configure multi-AZ deployment for high availability",
			},
		})
	}

	// Check for missing monitoring configurations
	if len(analysis.MonitoringConfigs) == 0 {
		gaps = append(gaps, ComplianceGap{
			Type:          "monitoring",
			Severity:      "medium",
			Description:   "No monitoring/logging configurations found",
			SOC2Controls:  []string{"CC7.2", "CC7.4"},
			EvidenceTasks: []string{},
			Recommendations: []string{
				"Enable CloudTrail for API logging",
				"Configure CloudWatch for monitoring and alerting",
				"Set up log retention policies",
				"Implement security event monitoring",
			},
		})
	}

	return gaps
}

// calculateSecurityRelevance calculates the relevance score for the security analysis
func (tsa *SecurityAnalyzer) calculateSecurityRelevance(analysis *SecurityAnalysisResult) float64 {
	relevance := 0.5 // Base score

	// Boost for number of security resources found
	if len(analysis.SecurityResources) >= 20 {
		relevance += 0.2
	} else if len(analysis.SecurityResources) >= 10 {
		relevance += 0.1
	}

	// Boost for coverage of security domains
	domainCount := 0
	if len(analysis.EncryptionConfigs) > 0 {
		domainCount++
	}
	if len(analysis.IAMConfigs) > 0 {
		domainCount++
	}
	if len(analysis.NetworkConfigs) > 0 {
		domainCount++
	}
	if len(analysis.MonitoringConfigs) > 0 {
		domainCount++
	}

	relevance += float64(domainCount) * 0.05

	// Boost for SOC2 control coverage
	if len(analysis.SOC2ControlMapping) >= 5 {
		relevance += 0.1
	}

	// Reduce for compliance gaps
	if len(analysis.ComplianceGaps) > 0 {
		relevance -= float64(len(analysis.ComplianceGaps)) * 0.02
	}

	// Cap at 1.0
	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}

// generateSecurityReport generates the security analysis report
func (tsa *SecurityAnalyzer) generateSecurityReport(analysis *SecurityAnalysisResult, format string) (string, error) {
	switch format {
	case "detailed_json":
		return tsa.generateDetailedJSONReport(analysis)
	case "summary_markdown":
		return tsa.generateSummaryMarkdownReport(analysis)
	case "compliance_csv":
		return tsa.generateComplianceCSVReport(analysis)
	default:
		return "", fmt.Errorf("unsupported output format: %s", format)
	}
}

func (tsa *SecurityAnalyzer) generateDetailedJSONReport(analysis *SecurityAnalysisResult) (string, error) {
	data, err := json.MarshalIndent(analysis, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON report: %w", err)
	}
	return string(data), nil
}

func (tsa *SecurityAnalyzer) generateSummaryMarkdownReport(analysis *SecurityAnalysisResult) (string, error) {
	var report strings.Builder

	report.WriteString("# Terraform Security Configuration Analysis\n\n")
	report.WriteString(fmt.Sprintf("**Analysis Date:** %s\n", analysis.AnalysisTimestamp.Format(time.RFC3339)))
	report.WriteString(fmt.Sprintf("**Security Domain:** %s\n", analysis.SecurityDomain))
	report.WriteString(fmt.Sprintf("**Files Analyzed:** %d\n", len(analysis.FilesAnalyzed)))
	report.WriteString(fmt.Sprintf("**Security Resources Found:** %d\n\n", len(analysis.SecurityResources)))

	// Executive Summary
	report.WriteString("## Executive Summary\n\n")
	report.WriteString(fmt.Sprintf("- **Encryption Configurations:** %d\n", len(analysis.EncryptionConfigs)))
	report.WriteString(fmt.Sprintf("- **IAM Configurations:** %d\n", len(analysis.IAMConfigs)))
	report.WriteString(fmt.Sprintf("- **Network Configurations:** %d\n", len(analysis.NetworkConfigs)))
	report.WriteString(fmt.Sprintf("- **Backup Configurations:** %d\n", len(analysis.BackupConfigs)))
	report.WriteString(fmt.Sprintf("- **Monitoring Configurations:** %d\n", len(analysis.MonitoringConfigs)))
	report.WriteString(fmt.Sprintf("- **Compliance Gaps:** %d\n\n", len(analysis.ComplianceGaps)))

	// SOC2 Control Mapping
	if len(analysis.SOC2ControlMapping) > 0 {
		report.WriteString("## SOC2 Control Mapping\n\n")
		for control, resources := range analysis.SOC2ControlMapping {
			report.WriteString(fmt.Sprintf("### %s (%d resources)\n", control, len(resources)))
			for _, resource := range resources {
				report.WriteString(fmt.Sprintf("- **%s** (%s) - `%s`\n", resource.ResourceName, resource.ResourceType, resource.FilePath))
			}
			report.WriteString("\n")
		}
	}

	// Evidence Task Mapping
	if len(analysis.EvidenceTaskMapping) > 0 {
		report.WriteString("## Evidence Task Mapping\n\n")
		for task, resources := range analysis.EvidenceTaskMapping {
			report.WriteString(fmt.Sprintf("### %s (%d resources)\n", task, len(resources)))
			for _, resource := range resources {
				report.WriteString(fmt.Sprintf("- **%s** (%s) - `%s`\n", resource.ResourceName, resource.ResourceType, resource.FilePath))
			}
			report.WriteString("\n")
		}
	}

	// Compliance Gaps
	if len(analysis.ComplianceGaps) > 0 {
		report.WriteString("## Compliance Gaps\n\n")
		for _, gap := range analysis.ComplianceGaps {
			report.WriteString(fmt.Sprintf("### %s (%s severity)\n", cases.Title(language.English).String(gap.Type), gap.Severity))
			report.WriteString(fmt.Sprintf("**Description:** %s\n\n", gap.Description))
			report.WriteString("**Recommendations:**\n")
			for _, rec := range gap.Recommendations {
				report.WriteString(fmt.Sprintf("- %s\n", rec))
			}
			report.WriteString("\n")
		}
	}

	return report.String(), nil
}

func (tsa *SecurityAnalyzer) generateComplianceCSVReport(analysis *SecurityAnalysisResult) (string, error) {
	var report strings.Builder

	// CSV Header
	report.WriteString("Resource Type,Resource Name,File Path,Line Range,SOC2 Controls,Evidence Tasks,Security Findings,Compliance Status\n")

	for _, resource := range analysis.SecurityResources {
		controls := strings.Join(resource.SecurityRelevance, ";")

		// Find evidence tasks for this resource
		var evidenceTasks []string
		for task, resources := range analysis.EvidenceTaskMapping {
			for _, mappedResource := range resources {
				if mappedResource.ResourceName == resource.ResourceName {
					evidenceTasks = append(evidenceTasks, task)
				}
			}
		}
		tasks := strings.Join(evidenceTasks, ";")

		// Summarize security findings
		var findingSummaries []string
		for _, finding := range resource.SecurityFindings {
			findingSummaries = append(findingSummaries, fmt.Sprintf("%s:%s", finding.Type, finding.Severity))
		}
		findings := strings.Join(findingSummaries, ";")

		// Determine compliance status
		complianceStatus := "COMPLIANT"
		for _, finding := range resource.SecurityFindings {
			if finding.Severity == "high" {
				complianceStatus = "NON_COMPLIANT"
				break
			} else if finding.Severity == "medium" && complianceStatus == "COMPLIANT" {
				complianceStatus = "PARTIALLY_COMPLIANT"
			}
		}

		// Escape CSV values
		resourceType := tsa.escapeCSV(resource.ResourceType)
		resourceName := tsa.escapeCSV(resource.ResourceName)
		filePath := tsa.escapeCSV(resource.FilePath)
		controlsCSV := tsa.escapeCSV(controls)
		tasksCSV := tsa.escapeCSV(tasks)
		findingsCSV := tsa.escapeCSV(findings)

		report.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s,%s\n",
			resourceType, resourceName, filePath, resource.LineRange, controlsCSV, tasksCSV, findingsCSV, complianceStatus))
	}

	return report.String(), nil
}

func (tsa *SecurityAnalyzer) escapeCSV(value string) string {
	if strings.Contains(value, ",") || strings.Contains(value, "\n") || strings.Contains(value, "\"") {
		value = strings.ReplaceAll(value, "\"", "\"\"")
		return "\"" + value + "\""
	}
	return value
}
