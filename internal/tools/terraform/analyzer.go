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
	"bufio"
	"context"
	"crypto/md5"
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
	"github.com/grctool/grctool/internal/storage"
)

// Analyzer provides Terraform configuration scanning capabilities with multiple strategies
type Analyzer struct {
	config       *config.TerraformToolConfig
	logger       logger.Logger
	strategy     AnalysisStrategy
	cacheStorage *storage.Storage
	cacheDir     string
}

// NewAnalyzer creates a new Terraform analyzer with the base strategy
func NewAnalyzer(cfg *config.Config, log logger.Logger) *Analyzer {
	// Set up cache directory
	cacheDir := filepath.Join(cfg.Storage.DataDir, ".cache", "terraform")

	// Create cache storage for scan results
	cacheCfg := config.StorageConfig{
		DataDir: cacheDir,
	}
	cacheStorage, err := storage.NewStorage(cacheCfg)
	if err != nil {
		log.Warn("Failed to initialize terraform cache storage", logger.Field{Key: "error", Value: err})
	}

	return &Analyzer{
		config:       &cfg.Evidence.Tools.Terraform,
		logger:       log,
		strategy:     &BaseAnalysisStrategy{logger: log},
		cacheStorage: cacheStorage,
		cacheDir:     cacheDir,
	}
}

// NewEnhancedAnalyzer creates a new Terraform analyzer with the enhanced strategy
func NewEnhancedAnalyzer(cfg *config.Config, log logger.Logger) *Analyzer {
	analyzer := NewAnalyzer(cfg, log)
	analyzer.strategy = &EnhancedAnalysisStrategy{
		BaseAnalysisStrategy: BaseAnalysisStrategy{logger: log},
		cacheDir:             analyzer.cacheDir,
		cacheStorage:         analyzer.cacheStorage,
	}
	return analyzer
}

// SetStrategy allows changing the analysis strategy
func (a *Analyzer) SetStrategy(strategy AnalysisStrategy) {
	a.strategy = strategy
}

// ScanForResources scans Terraform files for specific resource types
func (a *Analyzer) ScanForResources(ctx context.Context, resourceTypes []string) ([]models.TerraformScanResult, error) {
	if !a.config.Enabled {
		return nil, fmt.Errorf("terraform tool is not enabled")
	}

	var allResults []models.TerraformScanResult

	for _, scanPath := range a.config.ScanPaths {
		results, err := a.scanPath(ctx, scanPath, resourceTypes)
		if err != nil {
			continue
		}
		allResults = append(allResults, results...)
	}

	// Apply strategy-specific analysis
	return a.strategy.Analyze(allResults)
}

// ScanPath scans a specific path for Terraform files (public method for external use)
func (a *Analyzer) ScanPath(ctx context.Context, scanPath string, resourceTypes []string) ([]models.TerraformScanResult, error) {
	if !a.config.Enabled {
		return nil, fmt.Errorf("terraform tool is not enabled")
	}

	results, err := a.scanPath(ctx, scanPath, resourceTypes)
	if err != nil {
		return nil, err
	}

	// Apply strategy-specific analysis
	return a.strategy.Analyze(results)
}

// scanPath scans a specific path for Terraform files
func (a *Analyzer) scanPath(ctx context.Context, scanPath string, resourceTypes []string) ([]models.TerraformScanResult, error) {
	var results []models.TerraformScanResult

	// Expand glob patterns and walk directories
	err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue walking
		}

		// Skip directories
		if info.IsDir() {
			return nil
		}

		// Check if file matches include patterns
		if !a.matchesPatterns(path, a.config.IncludePatterns) {
			return nil
		}

		// Check if file matches exclude patterns
		if a.matchesPatterns(path, a.config.ExcludePatterns) {
			return nil
		}

		// Scan the file for resources
		fileResults, err := a.scanFile(ctx, path, resourceTypes)
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
func (a *Analyzer) scanFile(ctx context.Context, filePath string, resourceTypes []string) ([]models.TerraformScanResult, error) {
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
				if a.isResourceTypeOfInterest(resourceType, resourceTypes) {
					inResourceBlock = true
					braceDepth = 1
					currentResource = &models.TerraformScanResult{
						FilePath:          filePath,
						ResourceType:      resourceType,
						ResourceName:      resourceName,
						LineStart:         lineNum,
						Configuration:     make(map[string]interface{}),
						SecurityRelevance: a.getSecurityRelevance(resourceType),
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
			a.parseResourceConfiguration(trimmedLine, currentResource, variablePattern)
		}

		// Check if resource block has ended
		if braceDepth <= 0 {
			currentResource.LineEnd = lineNum
			currentResource.Configuration["_content"] = resourceContent.String()
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
func (a *Analyzer) matchesPatterns(filePath string, patterns []string) bool {
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
func (a *Analyzer) isResourceTypeOfInterest(resourceType string, resourceTypes []string) bool {
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
func (a *Analyzer) parseResourceConfiguration(line string, resource *models.TerraformScanResult, variablePattern *regexp.Regexp) {
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
func (a *Analyzer) getSecurityRelevance(resourceType string) []string {
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

		// Autoscaling resources (for SO2 - System Operations)
		"aws_autoscaling_group":               {"SO2"},
		"aws_autoscaling_policy":              {"SO2"},
		"aws_appautoscaling_target":           {"SO2"},
		"aws_appautoscaling_policy":           {"SO2"},
		"aws_appautoscaling_scheduled_action": {"SO2"},
		"aws_ecs_service":                     {"SO2"}, // When using auto_scaling
		"aws_eks_node_group":                  {"SO2"}, // Has scaling_config

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

// ExtractSecurityConfiguration extracts security-relevant configuration from scan results
func (a *Analyzer) ExtractSecurityConfiguration(results []models.TerraformScanResult) map[string]interface{} {
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
func (a *Analyzer) GenerateEvidenceReport(results []models.TerraformScanResult, format string) (string, error) {
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
		return a.generateCSVReport(results), nil
	case "markdown":
		return a.generateMarkdownReport(results), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// generateCSVReport generates a CSV format evidence report
func (a *Analyzer) generateCSVReport(results []models.TerraformScanResult) string {
	var report strings.Builder

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
		resourceType := a.escapeCSV(result.ResourceType)
		resourceName := a.escapeCSV(result.ResourceName)
		filePath := a.escapeCSV(result.FilePath)
		securityControlsCSV := a.escapeCSV(securityControls)
		keyConfigCSV := a.escapeCSV(keyConfigStr)

		report.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			resourceType, resourceName, filePath, lineRange, securityControlsCSV, keyConfigCSV))
	}

	return report.String()
}

// generateMarkdownReport generates a Markdown format evidence report
func (a *Analyzer) generateMarkdownReport(results []models.TerraformScanResult) string {
	var report strings.Builder

	report.WriteString("# Terraform Security Configuration Evidence\n\n")
	report.WriteString(fmt.Sprintf("Generated: %s\n", time.Now().Format(time.RFC3339)))
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

// escapeCSV escapes a string for CSV format
func (a *Analyzer) escapeCSV(value string) string {
	// If value contains comma, newline, or quote, wrap in quotes and escape internal quotes
	if strings.Contains(value, ",") || strings.Contains(value, "\n") || strings.Contains(value, "\"") {
		value = strings.ReplaceAll(value, "\"", "\"\"")
		return "\"" + value + "\""
	}
	return value
}

// calculateRelevance calculates the relevance score based on scan results
func (a *Analyzer) calculateRelevance(results []models.TerraformScanResult) float64 {
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

// Implementation of AnalysisStrategy interfaces

// Analyze implements BaseAnalysisStrategy
func (bas *BaseAnalysisStrategy) Analyze(results []models.TerraformScanResult) ([]models.TerraformScanResult, error) {
	// Base strategy just returns results as-is
	return results, nil
}

// Analyze implements EnhancedAnalysisStrategy with caching and filtering
func (eas *EnhancedAnalysisStrategy) Analyze(results []models.TerraformScanResult) ([]models.TerraformScanResult, error) {
	// Enhanced analysis would include caching, filtering, and bounded snippets
	// For now, return results as-is but this could be expanded
	return results, nil
}

// Enhanced analyzer methods for caching and filtering

// generateCacheKey generates a cache key for scan parameters
func (a *Analyzer) generateCacheKey(resourceTypes []string, pattern, controlHint string) string {
	keyData := fmt.Sprintf("rt:%v|p:%s|ch:%s", resourceTypes, pattern, controlHint)
	return fmt.Sprintf("%x", md5.Sum([]byte(keyData)))
}

// loadFromCache loads scan results from cache if available
func (a *Analyzer) loadFromCache(cacheKey string) *[]models.TerraformScanResult {
	if a.cacheStorage == nil {
		return nil
	}

	// Try to load cache data
	scansCacheDir := filepath.Join(a.cacheDir, "scans")
	cacheFile := filepath.Join(scansCacheDir, fmt.Sprintf("%s.json", cacheKey))

	// Check if cache file exists
	if _, err := os.Stat(cacheFile); os.IsNotExist(err) {
		return nil
	}

	// Read cache file
	data, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil
	}

	var cacheData struct {
		Results   []models.TerraformScanResult `json:"results"`
		Timestamp time.Time                    `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &cacheData); err != nil {
		return nil
	}

	// Check cache age (1 hour expiry)
	if time.Since(cacheData.Timestamp) > time.Hour {
		return nil
	}

	return &cacheData.Results
}

// saveToCache saves scan results to cache
func (a *Analyzer) saveToCache(cacheKey string, results []models.TerraformScanResult) {
	if a.cacheStorage == nil {
		return
	}

	cacheData := struct {
		Results   []models.TerraformScanResult `json:"results"`
		Timestamp time.Time                    `json:"timestamp"`
	}{
		Results:   results,
		Timestamp: time.Now(),
	}

	// Create cache directory
	scansCacheDir := filepath.Join(a.cacheDir, "scans")
	if err := os.MkdirAll(scansCacheDir, 0755); err != nil {
		a.logger.Warn("Failed to create cache directory",
			logger.String("cache_dir", scansCacheDir),
			logger.Field{Key: "error", Value: err})
		return
	}

	// Save to cache file
	cacheFile := filepath.Join(scansCacheDir, fmt.Sprintf("%s.json", cacheKey))
	data, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		a.logger.Warn("Failed to marshal cache data",
			logger.String("cache_key", cacheKey),
			logger.Field{Key: "error", Value: err})
		return
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		a.logger.Warn("Failed to save terraform scan results to cache",
			logger.String("cache_key", cacheKey),
			logger.String("cache_file", cacheFile),
			logger.Field{Key: "error", Value: err})
	}
}

// extractBoundedSnippet extracts code snippet with context around the resource
func (a *Analyzer) extractBoundedSnippet(result models.TerraformScanResult) string {
	if result.FilePath == "" {
		return ""
	}

	file, err := os.Open(result.FilePath)
	if err != nil {
		return fmt.Sprintf("Error reading file: %v", err)
	}
	defer file.Close()

	// Read the entire file content
	content, err := os.ReadFile(result.FilePath)
	if err != nil {
		return fmt.Sprintf("Error reading file content: %v", err)
	}

	lines := strings.Split(string(content), "\n")

	// Calculate bounds with context
	contextLines := 3
	startLine := result.LineStart - contextLines - 1 // Convert to 0-based index
	if startLine < 0 {
		startLine = 0
	}

	endLine := result.LineEnd + contextLines - 1 // Convert to 0-based index
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	// Extract the snippet
	var snippet strings.Builder
	snippet.WriteString(fmt.Sprintf("File: %s (lines %d-%d)\n", result.FilePath, startLine+1, endLine+1))
	snippet.WriteString("```hcl\n")

	for i := startLine; i <= endLine; i++ {
		lineNum := i + 1
		marker := "  "
		if lineNum >= result.LineStart && lineNum <= result.LineEnd {
			marker = "→ " // Mark the actual resource lines
		}
		snippet.WriteString(fmt.Sprintf("%s%4d: %s\n", marker, lineNum, lines[i]))
	}

	snippet.WriteString("```\n")
	return snippet.String()
}
