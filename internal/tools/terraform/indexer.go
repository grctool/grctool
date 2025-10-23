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
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// SecurityAttributeIndexer provides infrastructure resource indexing by security attributes
type SecurityAttributeIndexer struct {
	config       *config.TerraformToolConfig
	fullConfig   *config.Config
	logger       logger.Logger
	baseScanner  *Analyzer
	indexStorage *IndexStorage
	validator    *IndexValidator
}

// NewSecurityAttributeIndexer creates a new security attribute indexer
func NewSecurityAttributeIndexer(cfg *config.Config, log logger.Logger) *SecurityAttributeIndexer {
	// Construct index path: {cacheDir}/terraform/index.json.gz
	indexPath := filepath.Join(cfg.Storage.CacheDir, "terraform", IndexFileName)

	indexStorage := NewIndexStorage(indexPath, log)
	validator := NewIndexValidator(&cfg.Evidence.Tools.Terraform, log)

	return &SecurityAttributeIndexer{
		config:       &cfg.Evidence.Tools.Terraform,
		fullConfig:   cfg,
		logger:       log,
		baseScanner:  NewAnalyzer(cfg, log),
		indexStorage: indexStorage,
		validator:    validator,
	}
}

// Name returns the tool name
func (sai *SecurityAttributeIndexer) Name() string {
	return "terraform-security-indexer"
}

// Description returns the tool description
func (sai *SecurityAttributeIndexer) Description() string {
	return "Indexes and queries Terraform infrastructure resources by security attributes for compliance evidence collection"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (sai *SecurityAttributeIndexer) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        sai.Name(),
		Description: sai.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of security query to perform",
					"enum":        []string{"by_control", "by_attribute", "by_framework", "by_risk_level", "by_environment", "by_resource_type"},
					"default":     "by_control",
				},
				"control_codes": map[string]interface{}{
					"type":        "array",
					"description": "SOC2 control codes to find resources for (e.g., ['CC6.1', 'CC6.8', 'CC7.2'])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"security_attributes": map[string]interface{}{
					"type":        "array",
					"description": "Security attributes to filter by (e.g., ['encryption', 'access_control', 'monitoring', 'backup'])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"compliance_frameworks": map[string]interface{}{
					"type":        "array",
					"description": "Compliance frameworks to check (e.g., ['SOC2', 'ISO27001', 'PCI'])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"risk_levels": map[string]interface{}{
					"type":        "array",
					"description": "Risk levels to filter by (e.g., ['high', 'medium', 'low'])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"environments": map[string]interface{}{
					"type":        "array",
					"description": "Environments to include in query (e.g., ['prod', 'staging', 'dev'])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"resource_types": map[string]interface{}{
					"type":        "array",
					"description": "Specific resource types to query (e.g., ['aws_s3_bucket', 'aws_kms_key'])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"include_metadata": map[string]interface{}{
					"type":        "boolean",
					"description": "Include detailed metadata in the index results",
					"default":     true,
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "Output format for the indexed results",
					"enum":        []string{"detailed_json", "summary_table", "security_matrix", "evidence_map"},
					"default":     "detailed_json",
				},
				"skip_cache": map[string]interface{}{
					"type":        "boolean",
					"description": "Skip cached index and force live scan (default: false)",
					"default":     false,
				},
			},
			"required": []string{"query_type"},
		},
	}
}

// Execute runs the security attribute indexer with the given parameters
func (sai *SecurityAttributeIndexer) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	sai.logger.Debug("Executing Terraform security attribute indexer", logger.Field{Key: "params", Value: params})

	// Extract parameters
	queryType := "by_control"
	if qt, ok := params["query_type"].(string); ok {
		queryType = qt
	}

	var controlCodes []string
	if cc, ok := params["control_codes"].([]interface{}); ok {
		for _, c := range cc {
			if str, ok := c.(string); ok {
				controlCodes = append(controlCodes, str)
			}
		}
	}

	var securityAttributes []string
	if sa, ok := params["security_attributes"].([]interface{}); ok {
		for _, a := range sa {
			if str, ok := a.(string); ok {
				securityAttributes = append(securityAttributes, str)
			}
		}
	}

	var complianceFrameworks []string
	if cf, ok := params["compliance_frameworks"].([]interface{}); ok {
		for _, f := range cf {
			if str, ok := f.(string); ok {
				complianceFrameworks = append(complianceFrameworks, str)
			}
		}
	}

	var riskLevels []string
	if rl, ok := params["risk_levels"].([]interface{}); ok {
		for _, r := range rl {
			if str, ok := r.(string); ok {
				riskLevels = append(riskLevels, str)
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

	var resourceTypes []string
	if rt, ok := params["resource_types"].([]interface{}); ok {
		for _, r := range rt {
			if str, ok := r.(string); ok {
				resourceTypes = append(resourceTypes, str)
			}
		}
	}

	includeMetadata := true
	if im, ok := params["include_metadata"].(bool); ok {
		includeMetadata = im
	}

	outputFormat := "detailed_json"
	if of, ok := params["output_format"].(string); ok {
		outputFormat = of
	}

	skipCache := false
	if sc, ok := params["skip_cache"].(bool); ok {
		skipCache = sc
	}

	// Load or build index with caching
	persistedIndex, err := sai.LoadOrBuildIndex(ctx, skipCache)
	if err != nil {
		return "", nil, fmt.Errorf("failed to load or build index: %w", err)
	}

	// Apply query filters to the index
	query := SecurityIndexQuery{
		ControlCodes:         controlCodes,
		SecurityAttributes:   securityAttributes,
		ComplianceFrameworks: complianceFrameworks,
		RiskLevels:           riskLevels,
		Environments:         environments,
		ResourceTypes:        resourceTypes,
		IncludeMetadata:      includeMetadata,
	}

	securityIndex := sai.filterIndex(persistedIndex.Index, queryType, query)
	if err != nil {
		return "", nil, fmt.Errorf("failed to filter security index: %w", err)
	}

	// Generate report based on format
	report, err := sai.generateIndexReport(securityIndex, outputFormat)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate index report: %w", err)
	}

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "terraform-security-index",
		Resource:    fmt.Sprintf("Security index of %d resources across %d attributes", len(securityIndex.IndexedResources), len(securityIndex.SecurityAttributes)),
		Content:     report,
		Relevance:   sai.calculateIndexRelevance(securityIndex),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"query_type":            queryType,
			"control_codes":         controlCodes,
			"security_attributes":   securityAttributes,
			"compliance_frameworks": complianceFrameworks,
			"risk_levels":           riskLevels,
			"environments":          environments,
			"resource_types":        resourceTypes,
			"total_resources":       len(securityIndex.IndexedResources),
			"total_attributes":      len(securityIndex.SecurityAttributes),
			"compliance_coverage":   securityIndex.ComplianceCoverage,
			"risk_distribution":     securityIndex.RiskDistribution,
		},
	}

	return report, source, nil
}

// LoadOrBuildIndex loads a cached index or builds a new one if needed
func (sai *SecurityAttributeIndexer) LoadOrBuildIndex(ctx context.Context, forceRebuild bool) (*PersistedIndex, error) {
	// If force rebuild, skip cache
	if forceRebuild {
		sai.logger.Info("Force rebuild requested, skipping cache")
		return sai.BuildAndPersistIndex(ctx)
	}

	// Try to load existing index
	if sai.indexStorage.IndexExists() {
		index, err := sai.indexStorage.LoadIndex()
		if err != nil {
			sai.logger.Warn("Failed to load index, will rebuild",
				logger.Field{Key: "error", Value: err})
			return sai.BuildAndPersistIndex(ctx)
		}

		// Validate if rebuild is needed
		validationResult := sai.validator.NeedsRebuild(index)
		if !validationResult.NeedsRebuild {
			sai.logger.Info("Using cached index",
				logger.Field{Key: "indexed_at", Value: index.IndexedAt},
				logger.Int("resources", len(index.Index.IndexedResources)))
			return index, nil
		}

		sai.logger.Info("Index needs rebuild",
			logger.String("reason", string(validationResult.Reason)),
			logger.String("details", validationResult.Details))
	} else {
		sai.logger.Info("No cached index found, building new index")
	}

	// Build and persist new index
	return sai.BuildAndPersistIndex(ctx)
}

// BuildAndPersistIndex builds a new index and persists it to disk
func (sai *SecurityAttributeIndexer) BuildAndPersistIndex(ctx context.Context) (*PersistedIndex, error) {
	startTime := time.Now()
	sai.logger.Info("Building Terraform security index")

	// Build the index
	query := SecurityIndexQuery{
		IncludeMetadata: true,
	}
	securityIndex, sourceFiles, err := sai.buildSecurityIndexWithTracking(ctx, "by_control", query)
	if err != nil {
		return nil, fmt.Errorf("failed to build index: %w", err)
	}

	// Calculate statistics
	statistics := sai.calculateIndexStats(securityIndex)

	// Calculate config fingerprint
	configFingerprint := sai.validator.calculateConfigFingerprint()

	// Build persisted index
	persistedIndex := &PersistedIndex{
		Version:     IndexVersion,
		IndexedAt:   time.Now(),
		ToolVersion: "grctool-" + IndexVersion,
		Metadata: IndexMetadata{
			TotalFiles:        len(sourceFiles),
			TotalResources:    len(securityIndex.IndexedResources),
			ScanDurationMs:    time.Since(startTime).Milliseconds(),
			SourceDirectories: sai.config.ScanPaths,
			IncludePatterns:   sai.config.IncludePatterns,
			ExcludePatterns:   sai.config.ExcludePatterns,
		},
		SourceFiles:       sourceFiles,
		Index:             securityIndex,
		Statistics:        statistics,
		ConfigFingerprint: configFingerprint,
	}

	// Persist to disk
	if err := sai.indexStorage.SaveIndex(persistedIndex); err != nil {
		return nil, fmt.Errorf("failed to persist index: %w", err)
	}

	sai.logger.Info("Successfully built and persisted index",
		logger.Int("resources", len(securityIndex.IndexedResources)),
		logger.Int("files", len(sourceFiles)),
		logger.Duration("duration", time.Since(startTime)))

	return persistedIndex, nil
}

// buildSecurityIndex builds a comprehensive security index of infrastructure resources
func (sai *SecurityAttributeIndexer) buildSecurityIndex(ctx context.Context, queryType string, query SecurityIndexQuery) (*SecurityIndex, error) {
	index, _, err := sai.buildSecurityIndexWithTracking(ctx, queryType, query)
	return index, err
}

// buildSecurityIndexWithTracking builds index and returns source file tracking info
func (sai *SecurityAttributeIndexer) buildSecurityIndexWithTracking(ctx context.Context, queryType string, query SecurityIndexQuery) (*SecurityIndex, map[string]*SourceFileInfo, error) {
	// Get all terraform resources
	allResources, err := sai.baseScanner.ScanForResources(ctx, query.ResourceTypes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to scan terraform resources: %w", err)
	}

	index := &SecurityIndex{
		IndexTimestamp:     time.Now(),
		QueryType:          queryType,
		IndexedResources:   []IndexedResource{},
		SecurityAttributes: make(map[string]SecurityAttributeDetails),
		ControlMapping:     make(map[string][]IndexedResource),
		FrameworkMapping:   make(map[string][]IndexedResource),
		RiskDistribution:   make(map[string]int),
		ComplianceCoverage: 0.0,
		EnvironmentStats:   make(map[string]EnvironmentStats),
	}

	// Track source files with checksums
	sourceFiles := make(map[string]*SourceFileInfo)
	processedFiles := make(map[string]bool)

	// Process each resource and build index entries
	for _, resource := range allResources {
		indexedResource := sai.indexResource(resource, query)
		index.IndexedResources = append(index.IndexedResources, indexedResource)

		// Update mappings and statistics
		sai.updateIndexMappings(index, indexedResource)

		// Track source file if not already processed
		if !processedFiles[resource.FilePath] {
			fileInfo, err := BuildSourceFileInfo(resource.FilePath)
			if err != nil {
				sai.logger.Warn("Failed to build source file info",
					logger.String("file", resource.FilePath),
					logger.Field{Key: "error", Value: err})
			} else {
				sourceFiles[resource.FilePath] = fileInfo
				processedFiles[resource.FilePath] = true
			}
		}
	}

	// Calculate aggregate statistics
	sai.calculateIndexStatistics(index, query)

	return index, sourceFiles, nil
}

// filterIndex applies query filters to a loaded index
func (sai *SecurityAttributeIndexer) filterIndex(index *SecurityIndex, queryType string, query SecurityIndexQuery) *SecurityIndex {
	filtered := &SecurityIndex{
		IndexTimestamp:     index.IndexTimestamp,
		QueryType:          queryType,
		IndexedResources:   []IndexedResource{},
		SecurityAttributes: make(map[string]SecurityAttributeDetails),
		ControlMapping:     make(map[string][]IndexedResource),
		FrameworkMapping:   make(map[string][]IndexedResource),
		RiskDistribution:   make(map[string]int),
		ComplianceCoverage: 0.0,
		EnvironmentStats:   make(map[string]EnvironmentStats),
	}

	// Filter resources based on query
	for _, resource := range index.IndexedResources {
		if sai.matchesQuery(resource, queryType, query) {
			filtered.IndexedResources = append(filtered.IndexedResources, resource)
			sai.updateIndexMappings(filtered, resource)
		}
	}

	// Recalculate statistics for filtered set
	sai.calculateIndexStatistics(filtered, query)

	return filtered
}

// calculateIndexStats calculates statistics for the persisted index
func (sai *SecurityAttributeIndexer) calculateIndexStats(index *SecurityIndex) *IndexStatistics {
	stats := &IndexStatistics{
		ComplianceCoverage:      index.ComplianceCoverage,
		SecurityFindings:        make(map[string]int),
		ControlCoverage:         make(map[string]ControlCoverageStats),
		AttributeDistribution:   make(map[string]int),
		EnvironmentDistribution: make(map[string]int),
	}

	// Calculate attribute distribution
	for attr, details := range index.SecurityAttributes {
		stats.AttributeDistribution[attr] = details.ResourceCount
	}

	// Calculate environment distribution
	for env, envStats := range index.EnvironmentStats {
		stats.EnvironmentDistribution[env] = envStats.ResourceCount
	}

	// Calculate control coverage
	for control, resources := range index.ControlMapping {
		compliantCount := 0
		for _, res := range resources {
			if res.ComplianceStatus == "compliant" {
				compliantCount++
			}
		}

		complianceRate := 0.0
		if len(resources) > 0 {
			complianceRate = float64(compliantCount) / float64(len(resources))
		}

		stats.ControlCoverage[control] = ControlCoverageStats{
			TotalResources:     len(resources),
			CompliantResources: compliantCount,
			ComplianceRate:     complianceRate,
		}
	}

	// Calculate security findings (non-compliant resources)
	for _, resource := range index.IndexedResources {
		if resource.ComplianceStatus == "non_compliant" {
			stats.SecurityFindings[resource.RiskLevel]++
		}
	}

	return stats
}

// indexResource creates an indexed representation of a terraform resource
func (sai *SecurityAttributeIndexer) indexResource(resource models.TerraformScanResult, query SecurityIndexQuery) IndexedResource {
	indexed := IndexedResource{
		ResourceID:         fmt.Sprintf("%s.%s", resource.ResourceType, resource.ResourceName),
		ResourceType:       resource.ResourceType,
		ResourceName:       resource.ResourceName,
		FilePath:           resource.FilePath,
		LineRange:          fmt.Sprintf("%d-%d", resource.LineStart, resource.LineEnd),
		Environment:        sai.extractEnvironmentFromPath(resource.FilePath),
		SecurityAttributes: sai.extractSecurityAttributes(resource),
		ControlRelevance:   resource.SecurityRelevance,
		RiskLevel:          sai.calculateResourceRiskLevel(resource),
		ComplianceStatus:   sai.calculateComplianceStatus(resource),
		Configuration:      make(map[string]interface{}),
		LastModified:       time.Now(), // Would be file mtime in real implementation
	}

	// Include metadata if requested
	if query.IncludeMetadata {
		// Filter configuration to include only security-relevant items
		for key, value := range resource.Configuration {
			if sai.isSecurityRelevantConfig(key, value) {
				indexed.Configuration[key] = value
			}
		}
	}

	return indexed
}

// extractSecurityAttributes extracts security attributes from a terraform resource
func (sai *SecurityAttributeIndexer) extractSecurityAttributes(resource models.TerraformScanResult) []string {
	var attributes []string
	attributeSet := make(map[string]bool)

	resourceType := strings.ToLower(resource.ResourceType)

	// Define attribute mappings based on resource type
	attributeMappings := map[string][]string{
		// Encryption attributes
		"aws_kms":                  {"encryption", "key_management"},
		"aws_s3_bucket_encryption": {"encryption", "data_protection"},
		"aws_rds":                  {"encryption", "database_security"},
		"aws_ebs_encryption":       {"encryption", "storage_security"},
		"aws_acm_certificate":      {"encryption", "ssl_tls"},

		// Access control attributes
		"aws_iam":              {"access_control", "identity_management"},
		"aws_security_group":   {"access_control", "network_security"},
		"aws_nacl":             {"access_control", "network_security"},
		"aws_s3_bucket_policy": {"access_control", "data_protection"},

		// Monitoring attributes
		"aws_cloudtrail": {"monitoring", "audit_logging"},
		"aws_cloudwatch": {"monitoring", "observability"},
		"aws_config":     {"monitoring", "compliance_tracking"},
		"aws_guardduty":  {"monitoring", "threat_detection"},

		// Backup attributes
		"aws_backup":               {"backup", "disaster_recovery"},
		"aws_s3_bucket_versioning": {"backup", "data_versioning"},
		"aws_db_snapshot":          {"backup", "database_backup"},

		// Network security attributes
		"aws_vpc":        {"network_security", "network_isolation"},
		"aws_subnet":     {"network_security", "network_segmentation"},
		"aws_lb":         {"network_security", "load_balancing"},
		"aws_cloudfront": {"network_security", "cdn_security"},

		// High availability attributes
		"aws_autoscaling_group": {"high_availability", "auto_scaling"},
		"aws_rds_cluster":       {"high_availability", "database_clustering"},
		"aws_elasticache":       {"high_availability", "caching"},
	}

	// Check for direct matches
	for pattern, attrs := range attributeMappings {
		if strings.Contains(resourceType, pattern) {
			for _, attr := range attrs {
				if !attributeSet[attr] {
					attributes = append(attributes, attr)
					attributeSet[attr] = true
				}
			}
		}
	}

	// Check configuration for additional security attributes
	configAttributes := sai.extractAttributesFromConfiguration(resource.Configuration)
	for _, attr := range configAttributes {
		if !attributeSet[attr] {
			attributes = append(attributes, attr)
			attributeSet[attr] = true
		}
	}

	sort.Strings(attributes)
	return attributes
}

// extractAttributesFromConfiguration extracts security attributes from resource configuration
func (sai *SecurityAttributeIndexer) extractAttributesFromConfiguration(config map[string]interface{}) []string {
	var attributes []string
	attributeSet := make(map[string]bool)

	for key, value := range config {
		keyLower := strings.ToLower(key)
		valueStr := fmt.Sprintf("%v", value)
		valueLower := strings.ToLower(valueStr)

		// Check for encryption indicators
		if strings.Contains(keyLower, "encrypt") || strings.Contains(keyLower, "kms") ||
			strings.Contains(keyLower, "ssl") || strings.Contains(keyLower, "tls") {
			if !attributeSet["encryption"] {
				attributes = append(attributes, "encryption")
				attributeSet["encryption"] = true
			}
		}

		// Check for access control indicators
		if strings.Contains(keyLower, "policy") || strings.Contains(keyLower, "permission") ||
			strings.Contains(keyLower, "role") || strings.Contains(keyLower, "access") {
			if !attributeSet["access_control"] {
				attributes = append(attributes, "access_control")
				attributeSet["access_control"] = true
			}
		}

		// Check for monitoring indicators
		if strings.Contains(keyLower, "log") || strings.Contains(keyLower, "monitor") ||
			strings.Contains(keyLower, "alert") || strings.Contains(keyLower, "audit") {
			if !attributeSet["monitoring"] {
				attributes = append(attributes, "monitoring")
				attributeSet["monitoring"] = true
			}
		}

		// Check for backup indicators
		if strings.Contains(keyLower, "backup") || strings.Contains(keyLower, "snapshot") ||
			strings.Contains(keyLower, "versioning") || strings.Contains(keyLower, "retention") {
			if !attributeSet["backup"] {
				attributes = append(attributes, "backup")
				attributeSet["backup"] = true
			}
		}

		// Check for multi-factor authentication
		if strings.Contains(keyLower, "mfa") || strings.Contains(keyLower, "2fa") ||
			strings.Contains(valueLower, "mfa") || strings.Contains(valueLower, "2fa") {
			if !attributeSet["multi_factor_auth"] {
				attributes = append(attributes, "multi_factor_auth")
				attributeSet["multi_factor_auth"] = true
			}
		}
	}

	return attributes
}

// calculateResourceRiskLevel calculates the risk level of a resource based on its configuration
func (sai *SecurityAttributeIndexer) calculateResourceRiskLevel(resource models.TerraformScanResult) string {
	riskScore := 0
	resourceType := strings.ToLower(resource.ResourceType)

	// High-risk resource types
	highRiskTypes := []string{
		"aws_iam_user", "aws_iam_role", "aws_s3_bucket", "aws_security_group",
		"aws_db_instance", "aws_rds_cluster", "aws_kms_key",
	}
	for _, hrType := range highRiskTypes {
		if strings.Contains(resourceType, hrType) {
			riskScore += 3
			break
		}
	}

	// Check for security misconfigurations
	if sai.hasSecurityMisconfigurations(resource) {
		riskScore += 2
	}

	// Check for missing security controls
	if len(resource.SecurityRelevance) == 0 {
		riskScore += 1
	}

	// Check for public access patterns
	if sai.hasPublicAccess(resource) {
		riskScore += 3
	}

	// Determine risk level
	if riskScore >= 5 {
		return "high"
	} else if riskScore >= 3 {
		return "medium"
	} else {
		return "low"
	}
}

// hasSecurityMisconfigurations checks for common security misconfigurations
func (sai *SecurityAttributeIndexer) hasSecurityMisconfigurations(resource models.TerraformScanResult) bool {
	resourceType := strings.ToLower(resource.ResourceType)

	// S3 bucket misconfigurations
	if strings.Contains(resourceType, "s3_bucket") && !strings.Contains(resourceType, "encryption") {
		return true
	}

	// RDS without encryption
	if strings.Contains(resourceType, "rds") || strings.Contains(resourceType, "db_instance") {
		if encrypted, ok := resource.Configuration["storage_encrypted"]; !ok || encrypted != true {
			return true
		}
	}

	// Security groups with overly permissive rules
	if strings.Contains(resourceType, "security_group") {
		for key, value := range resource.Configuration {
			if strings.Contains(strings.ToLower(key), "cidr") {
				if valueStr, ok := value.(string); ok && strings.Contains(valueStr, "0.0.0.0/0") {
					return true
				}
			}
		}
	}

	return false
}

// hasPublicAccess checks if a resource has public access configurations
func (sai *SecurityAttributeIndexer) hasPublicAccess(resource models.TerraformScanResult) bool {
	for key, value := range resource.Configuration {
		keyLower := strings.ToLower(key)
		valueStr := fmt.Sprintf("%v", value)
		valueLower := strings.ToLower(valueStr)

		// Check for public access indicators
		if strings.Contains(keyLower, "public") && (valueLower == "true" || valueLower == "1") {
			return true
		}

		// Check for open CIDR blocks
		if strings.Contains(keyLower, "cidr") && strings.Contains(valueLower, "0.0.0.0/0") {
			return true
		}

		// Check for wildcard permissions
		if strings.Contains(keyLower, "policy") && strings.Contains(valueLower, "*") {
			return true
		}
	}

	return false
}

// calculateComplianceStatus calculates the compliance status of a resource
func (sai *SecurityAttributeIndexer) calculateComplianceStatus(resource models.TerraformScanResult) string {
	// Check if resource has required security controls
	hasEncryption := sai.hasEncryptionControls(resource)
	hasAccessControl := sai.hasAccessControls(resource)
	hasMonitoring := sai.hasMonitoringControls(resource)

	requiredControls := 0
	implementedControls := 0

	resourceType := strings.ToLower(resource.ResourceType)

	// Determine required controls based on resource type
	if strings.Contains(resourceType, "s3") || strings.Contains(resourceType, "rds") || strings.Contains(resourceType, "kms") {
		requiredControls++
		if hasEncryption {
			implementedControls++
		}
	}

	if strings.Contains(resourceType, "iam") || strings.Contains(resourceType, "security_group") {
		requiredControls++
		if hasAccessControl {
			implementedControls++
		}
	}

	if strings.Contains(resourceType, "cloudtrail") || strings.Contains(resourceType, "cloudwatch") {
		requiredControls++
		if hasMonitoring {
			implementedControls++
		}
	}

	// Calculate compliance percentage
	if requiredControls == 0 {
		return "not_applicable"
	}

	compliancePercent := float64(implementedControls) / float64(requiredControls)

	if compliancePercent >= 1.0 {
		return "compliant"
	} else if compliancePercent >= 0.7 {
		return "partially_compliant"
	} else {
		return "non_compliant"
	}
}

// hasEncryptionControls checks if a resource has encryption controls
func (sai *SecurityAttributeIndexer) hasEncryptionControls(resource models.TerraformScanResult) bool {
	for key, value := range resource.Configuration {
		keyLower := strings.ToLower(key)
		if strings.Contains(keyLower, "encrypt") || strings.Contains(keyLower, "kms") {
			if boolVal, ok := value.(bool); ok && boolVal {
				return true
			}
			if strVal, ok := value.(string); ok && strVal != "" && strVal != "false" {
				return true
			}
		}
	}
	return false
}

// hasAccessControls checks if a resource has access controls
func (sai *SecurityAttributeIndexer) hasAccessControls(resource models.TerraformScanResult) bool {
	for key, value := range resource.Configuration {
		keyLower := strings.ToLower(key)
		if strings.Contains(keyLower, "policy") || strings.Contains(keyLower, "role") || strings.Contains(keyLower, "permission") {
			if strVal, ok := value.(string); ok && strVal != "" {
				return true
			}
		}
	}
	return false
}

// hasMonitoringControls checks if a resource has monitoring controls
func (sai *SecurityAttributeIndexer) hasMonitoringControls(resource models.TerraformScanResult) bool {
	for key, value := range resource.Configuration {
		keyLower := strings.ToLower(key)
		if strings.Contains(keyLower, "log") || strings.Contains(keyLower, "monitor") {
			if boolVal, ok := value.(bool); ok && boolVal {
				return true
			}
			if strVal, ok := value.(string); ok && strVal != "" && strVal != "false" {
				return true
			}
		}
	}
	return false
}

// extractEnvironmentFromPath extracts environment from file path
func (sai *SecurityAttributeIndexer) extractEnvironmentFromPath(filePath string) string {
	// Simple environment extraction based on common patterns
	fileLower := strings.ToLower(filePath)

	environments := []string{"prod", "production", "staging", "stage", "dev", "development", "test", "testing"}
	for _, env := range environments {
		if strings.Contains(fileLower, env) {
			return env
		}
	}

	return "unknown"
}

// matchesQuery checks if an indexed resource matches the query criteria
func (sai *SecurityAttributeIndexer) matchesQuery(resource IndexedResource, queryType string, query SecurityIndexQuery) bool {
	switch queryType {
	case "by_control":
		if len(query.ControlCodes) == 0 {
			return true
		}
		for _, control := range query.ControlCodes {
			for _, resourceControl := range resource.ControlRelevance {
				if resourceControl == control {
					return true
				}
			}
		}
		return false

	case "by_attribute":
		if len(query.SecurityAttributes) == 0 {
			return true
		}
		for _, attr := range query.SecurityAttributes {
			for _, resourceAttr := range resource.SecurityAttributes {
				if resourceAttr == attr {
					return true
				}
			}
		}
		return false

	case "by_framework":
		// Framework matching would require more sophisticated logic
		return true

	case "by_risk_level":
		if len(query.RiskLevels) == 0 {
			return true
		}
		for _, risk := range query.RiskLevels {
			if resource.RiskLevel == risk {
				return true
			}
		}
		return false

	case "by_environment":
		if len(query.Environments) == 0 {
			return true
		}
		for _, env := range query.Environments {
			if resource.Environment == env {
				return true
			}
		}
		return false

	case "by_resource_type":
		if len(query.ResourceTypes) == 0 {
			return true
		}
		for _, rt := range query.ResourceTypes {
			if resource.ResourceType == rt {
				return true
			}
		}
		return false

	default:
		return true
	}
}

// updateIndexMappings updates the index mappings and statistics
func (sai *SecurityAttributeIndexer) updateIndexMappings(index *SecurityIndex, resource IndexedResource) {
	// Update control mapping
	for _, control := range resource.ControlRelevance {
		index.ControlMapping[control] = append(index.ControlMapping[control], resource)
	}

	// Update risk distribution
	index.RiskDistribution[resource.RiskLevel]++

	// Update security attributes
	for _, attr := range resource.SecurityAttributes {
		if details, exists := index.SecurityAttributes[attr]; exists {
			details.ResourceCount++
			details.LastSeen = time.Now()
		} else {
			index.SecurityAttributes[attr] = SecurityAttributeDetails{
				AttributeName: attr,
				ResourceCount: 1,
				FirstSeen:     time.Now(),
				LastSeen:      time.Now(),
			}
		}
	}

	// Update environment stats
	if stats, exists := index.EnvironmentStats[resource.Environment]; exists {
		stats.ResourceCount++
		stats.RiskDistribution[resource.RiskLevel]++
	} else {
		riskDist := make(map[string]int)
		riskDist[resource.RiskLevel] = 1
		index.EnvironmentStats[resource.Environment] = EnvironmentStats{
			Environment:      resource.Environment,
			ResourceCount:    1,
			RiskDistribution: riskDist,
		}
	}
}

// calculateIndexStatistics calculates aggregate statistics for the index
func (sai *SecurityAttributeIndexer) calculateIndexStatistics(index *SecurityIndex, query SecurityIndexQuery) {
	totalResources := len(index.IndexedResources)
	if totalResources == 0 {
		return
	}

	// Calculate compliance coverage
	compliantCount := 0
	for _, resource := range index.IndexedResources {
		if resource.ComplianceStatus == "compliant" {
			compliantCount++
		}
	}
	index.ComplianceCoverage = float64(compliantCount) / float64(totalResources)
}

// isSecurityRelevantConfig checks if a configuration key-value pair is security-relevant
func (sai *SecurityAttributeIndexer) isSecurityRelevantConfig(key string, value interface{}) bool {
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

	return false
}

// calculateIndexRelevance calculates relevance score for the security index
func (sai *SecurityAttributeIndexer) calculateIndexRelevance(index *SecurityIndex) float64 {
	relevance := 0.5 // Base score

	// Boost for number of indexed resources
	if len(index.IndexedResources) >= 50 {
		relevance += 0.2
	} else if len(index.IndexedResources) >= 20 {
		relevance += 0.1
	}

	// Boost for security attribute coverage
	if len(index.SecurityAttributes) >= 10 {
		relevance += 0.1
	}

	// Boost for compliance coverage
	relevance += index.ComplianceCoverage * 0.2

	// Reduce for high risk resources
	highRiskCount := index.RiskDistribution["high"]
	totalResources := len(index.IndexedResources)
	if totalResources > 0 {
		highRiskRatio := float64(highRiskCount) / float64(totalResources)
		if highRiskRatio > 0.3 {
			relevance -= 0.1
		}
	}

	// Cap at 1.0
	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}

// generateIndexReport generates the index report in the requested format
func (sai *SecurityAttributeIndexer) generateIndexReport(index *SecurityIndex, format string) (string, error) {
	switch format {
	case "detailed_json":
		return sai.generateDetailedJSONReport(index)
	case "summary_table":
		return sai.generateSummaryTableReport(index)
	case "security_matrix":
		return sai.generateSecurityMatrixReport(index)
	case "evidence_map":
		return sai.generateEvidenceMapReport(index)
	default:
		return "", fmt.Errorf("unsupported output format: %s", format)
	}
}

// Report generation methods (implement as needed)
func (sai *SecurityAttributeIndexer) generateDetailedJSONReport(index *SecurityIndex) (string, error) {
	// Return the index as formatted JSON-like output
	var report strings.Builder

	report.WriteString("# Security Index Report\n\n")
	report.WriteString(fmt.Sprintf("**Generated:** %s\n", index.IndexTimestamp.Format(time.RFC3339)))
	report.WriteString(fmt.Sprintf("**Query Type:** %s\n", index.QueryType))
	report.WriteString(fmt.Sprintf("**Total Resources:** %d\n", len(index.IndexedResources)))
	report.WriteString(fmt.Sprintf("**Compliance Coverage:** %.1f%%\n\n", index.ComplianceCoverage*100))

	// Risk distribution
	report.WriteString("## Risk Distribution\n\n")
	for risk, count := range index.RiskDistribution {
		report.WriteString(fmt.Sprintf("- **%s:** %d resources\n", cases.Title(language.English).String(risk), count))
	}
	report.WriteString("\n")

	// Security attributes
	report.WriteString("## Security Attributes\n\n")
	for attr, details := range index.SecurityAttributes {
		report.WriteString(fmt.Sprintf("- **%s:** %d resources\n", attr, details.ResourceCount))
	}
	report.WriteString("\n")

	// Top resources by control relevance
	report.WriteString("## Control Mapping\n\n")
	for control, resources := range index.ControlMapping {
		report.WriteString(fmt.Sprintf("### %s (%d resources)\n", control, len(resources)))
		for i, resource := range resources {
			if i >= 5 { // Limit to top 5 per control
				report.WriteString(fmt.Sprintf("... and %d more\n", len(resources)-5))
				break
			}
			report.WriteString(fmt.Sprintf("- %s (%s)\n", resource.ResourceID, resource.RiskLevel))
		}
		report.WriteString("\n")
	}

	return report.String(), nil
}

func (sai *SecurityAttributeIndexer) generateSummaryTableReport(index *SecurityIndex) (string, error) {
	var report strings.Builder

	report.WriteString("Resource ID,Resource Type,Environment,Risk Level,Compliance Status,Security Attributes\n")

	for _, resource := range index.IndexedResources {
		attributes := strings.Join(resource.SecurityAttributes, ";")
		report.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			sai.escapeCSV(resource.ResourceID),
			sai.escapeCSV(resource.ResourceType),
			sai.escapeCSV(resource.Environment),
			sai.escapeCSV(resource.RiskLevel),
			sai.escapeCSV(resource.ComplianceStatus),
			sai.escapeCSV(attributes)))
	}

	return report.String(), nil
}

func (sai *SecurityAttributeIndexer) generateSecurityMatrixReport(index *SecurityIndex) (string, error) {
	var report strings.Builder

	report.WriteString("# Security Matrix Report\n\n")

	// Create matrix of environments vs risk levels
	report.WriteString("## Environment Risk Matrix\n\n")
	report.WriteString("| Environment | High | Medium | Low | Total |\n")
	report.WriteString("|-------------|------|--------|-----|-------|\n")

	for env, stats := range index.EnvironmentStats {
		high := stats.RiskDistribution["high"]
		medium := stats.RiskDistribution["medium"]
		low := stats.RiskDistribution["low"]
		total := stats.ResourceCount

		report.WriteString(fmt.Sprintf("| %s | %d | %d | %d | %d |\n", env, high, medium, low, total))
	}

	return report.String(), nil
}

func (sai *SecurityAttributeIndexer) generateEvidenceMapReport(index *SecurityIndex) (string, error) {
	var report strings.Builder

	report.WriteString("# Evidence Mapping Report\n\n")

	// Group resources by control codes for evidence collection
	for control, resources := range index.ControlMapping {
		report.WriteString(fmt.Sprintf("## Control %s\n\n", control))
		report.WriteString(fmt.Sprintf("**Resources:** %d\n", len(resources)))
		report.WriteString("**Evidence Sources:**\n")

		for _, resource := range resources {
			report.WriteString(fmt.Sprintf("- %s: `%s` (Risk: %s)\n",
				resource.ResourceType, resource.FilePath, resource.RiskLevel))
		}
		report.WriteString("\n")
	}

	return report.String(), nil
}

func (sai *SecurityAttributeIndexer) escapeCSV(value string) string {
	if strings.Contains(value, ",") || strings.Contains(value, "\n") || strings.Contains(value, "\"") {
		value = strings.ReplaceAll(value, "\"", "\"\"")
		return "\"" + value + "\""
	}
	return value
}
