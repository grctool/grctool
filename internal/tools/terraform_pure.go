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
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/models"
)

// Pure functions for Terraform analysis
// These functions contain no external dependencies and are easily testable

// ParseTerraformContent parses Terraform content from a reader
func ParseTerraformContent(content io.Reader, filePath string, resourceTypes []string) ([]models.TerraformScanResult, error) {
	var results []models.TerraformScanResult
	scanner := bufio.NewScanner(content)
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
				if IsResourceTypeOfInterest(resourceType, resourceTypes) {
					inResourceBlock = true
					braceDepth = 1
					currentResource = &models.TerraformScanResult{
						FilePath:          filePath,
						ResourceType:      resourceType,
						ResourceName:      resourceName,
						LineStart:         lineNum,
						Configuration:     make(map[string]interface{}),
						SecurityRelevance: GetSecurityRelevance(resourceType),
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
			ParseResourceConfiguration(trimmedLine, currentResource, variablePattern)
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
		return nil, fmt.Errorf("error scanning content: %w", err)
	}

	return results, nil
}

// ParseTerraformHCLBlocks parses specific HCL block types (module, data, locals)
func ParseTerraformHCLBlocks(content io.Reader, filePath, blockType string) ([]models.TerraformScanResult, error) {
	var results []models.TerraformScanResult
	scanner := bufio.NewScanner(content)
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
				ParseResourceConfiguration(trimmedLine, currentBlock, variablePattern)
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
		return nil, fmt.Errorf("error scanning content: %w", err)
	}

	return results, nil
}

// ParseResourceConfiguration parses configuration lines within a resource block
func ParseResourceConfiguration(line string, resource *models.TerraformScanResult, variablePattern *regexp.Regexp) {
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

// AnalyzeEncryptionSettings analyzes encryption configurations in resources
func AnalyzeEncryptionSettings(resources []models.TerraformScanResult) EncryptionAnalysis {
	analysis := EncryptionAnalysis{
		EncryptedResources:   []string{},
		UnencryptedResources: []string{},
		EncryptionMethods:    make(map[string][]string),
	}

	for _, resource := range resources {
		resourceRef := fmt.Sprintf("%s.%s", resource.ResourceType, resource.ResourceName)

		// Check if resource has encryption configuration
		encrypted := false

		// Check common encryption patterns
		for key, value := range resource.Configuration {
			keyLower := strings.ToLower(key)
			valueLower := strings.ToLower(fmt.Sprintf("%v", value))

			if (strings.Contains(keyLower, "encrypt") &&
				(valueLower == "true" || valueLower == "enabled")) ||
				strings.Contains(keyLower, "kms_key") ||
				strings.Contains(keyLower, "encryption_key") {
				encrypted = true

				// Determine encryption method
				var method string
				if strings.Contains(keyLower, "kms") {
					method = "KMS"
				} else if strings.Contains(keyLower, "aes") {
					method = "AES"
				} else {
					method = "Unknown"
				}

				analysis.EncryptionMethods[method] = append(analysis.EncryptionMethods[method], resourceRef)
				break
			}
		}

		if encrypted {
			analysis.EncryptedResources = append(analysis.EncryptedResources, resourceRef)
		} else {
			// Only consider certain resource types as requiring encryption
			if RequiresEncryption(resource.ResourceType) {
				analysis.UnencryptedResources = append(analysis.UnencryptedResources, resourceRef)
			}
		}
	}

	analysis.TotalResources = len(resources)
	analysis.EncryptionScore = calculateEncryptionScore(analysis)

	return analysis
}

// AnalyzeIAMConfiguration analyzes IAM settings in resources
func AnalyzeIAMConfiguration(resources []models.TerraformScanResult) IAMAnalysis {
	analysis := IAMAnalysis{
		IAMRoles:         []string{},
		IAMPolicies:      []string{},
		OverlyPermissive: []string{},
		SecurityGaps:     []string{},
	}

	for _, resource := range resources {
		resourceRef := fmt.Sprintf("%s.%s", resource.ResourceType, resource.ResourceName)
		resourceType := strings.ToLower(resource.ResourceType)

		// Categorize IAM resources
		if strings.Contains(resourceType, "iam_role") {
			analysis.IAMRoles = append(analysis.IAMRoles, resourceRef)
		} else if strings.Contains(resourceType, "iam_policy") {
			analysis.IAMPolicies = append(analysis.IAMPolicies, resourceRef)
		}

		// Check for overly permissive configurations
		for key, value := range resource.Configuration {
			keyLower := strings.ToLower(key)
			valueLower := strings.ToLower(fmt.Sprintf("%v", value))

			if (strings.Contains(keyLower, "action") && valueLower == "*") ||
				(strings.Contains(keyLower, "resource") && valueLower == "*") ||
				(strings.Contains(keyLower, "principal") && valueLower == "*") {
				analysis.OverlyPermissive = append(analysis.OverlyPermissive, resourceRef)
				break
			}
		}
	}

	analysis.TotalIAMResources = len(analysis.IAMRoles) + len(analysis.IAMPolicies)
	analysis.SecurityScore = calculateIAMSecurityScore(analysis)

	return analysis
}

// AnalyzeNetworkSecurity analyzes network security configurations
func AnalyzeNetworkSecurity(resources []models.TerraformScanResult) NetworkSecurityAnalysis {
	analysis := NetworkSecurityAnalysis{
		SecurityGroups:   []string{},
		OpenToInternet:   []string{},
		RestrictedAccess: []string{},
	}

	for _, resource := range resources {
		resourceRef := fmt.Sprintf("%s.%s", resource.ResourceType, resource.ResourceName)
		resourceType := strings.ToLower(resource.ResourceType)

		// Identify network security resources
		if strings.Contains(resourceType, "security_group") ||
			strings.Contains(resourceType, "firewall") ||
			strings.Contains(resourceType, "nacl") {
			analysis.SecurityGroups = append(analysis.SecurityGroups, resourceRef)

			// Check for open access
			for key, value := range resource.Configuration {
				keyLower := strings.ToLower(key)
				valueLower := strings.ToLower(fmt.Sprintf("%v", value))

				if strings.Contains(keyLower, "cidr") &&
					(strings.Contains(valueLower, "0.0.0.0/0") || strings.Contains(valueLower, "::/0")) {
					analysis.OpenToInternet = append(analysis.OpenToInternet, resourceRef)
				} else if strings.Contains(keyLower, "cidr") &&
					!strings.Contains(valueLower, "0.0.0.0/0") &&
					!strings.Contains(valueLower, "::/0") {
					analysis.RestrictedAccess = append(analysis.RestrictedAccess, resourceRef)
				}
			}
		}
	}

	analysis.TotalNetworkResources = len(analysis.SecurityGroups)
	analysis.SecurityScore = calculateNetworkSecurityScore(analysis)

	return analysis
}

// CalculateComplianceScore calculates overall compliance score from analyses
func CalculateComplianceScore(analyses ...SecurityAnalysis) ComplianceScore {
	if len(analyses) == 0 {
		return ComplianceScore{
			OverallScore: 0.0,
			Categories:   make(map[string]float64),
		}
	}

	score := ComplianceScore{
		Categories: make(map[string]float64),
	}

	var totalScore float64
	for _, analysis := range analyses {
		switch a := analysis.(type) {
		case EncryptionAnalysis:
			score.Categories["encryption"] = a.EncryptionScore
			totalScore += a.EncryptionScore
		case IAMAnalysis:
			score.Categories["iam"] = a.SecurityScore
			totalScore += a.SecurityScore
		case NetworkSecurityAnalysis:
			score.Categories["network"] = a.SecurityScore
			totalScore += a.SecurityScore
		}
	}

	score.OverallScore = totalScore / float64(len(analyses))
	return score
}

// IdentifySecurityGaps identifies gaps between current and required security state
func IdentifySecurityGaps(current SecurityState, required SecurityRequirements) []SecurityGap {
	var gaps []SecurityGap

	// Check encryption gaps
	if required.RequireEncryption {
		for _, resource := range current.UnencryptedResources {
			gaps = append(gaps, SecurityGap{
				Type:        "encryption",
				Resource:    resource,
				Description: "Resource should be encrypted but encryption is not configured",
				Severity:    "high",
			})
		}
	}

	// Check IAM gaps
	for _, resource := range current.OverlyPermissiveResources {
		gaps = append(gaps, SecurityGap{
			Type:        "iam",
			Resource:    resource,
			Description: "Resource has overly permissive IAM configuration",
			Severity:    "medium",
		})
	}

	// Check network security gaps
	for _, resource := range current.OpenNetworkResources {
		gaps = append(gaps, SecurityGap{
			Type:        "network",
			Resource:    resource,
			Description: "Resource allows unrestricted network access (0.0.0.0/0)",
			Severity:    "high",
		})
	}

	return gaps
}

// GenerateTerraformSecurityReport generates a security report from analysis results
func GenerateTerraformSecurityReport(analysis SecurityAnalysis, format string) (string, error) {
	switch format {
	case "csv":
		return generateCSVSecurityReport(analysis), nil
	case "markdown":
		return generateMarkdownSecurityReport(analysis), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// GenerateTerraformEvidenceReport generates a structured evidence report from scan results
func GenerateTerraformEvidenceReport(results []models.TerraformScanResult, format string) (string, error) {
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
		return generateCSVReport(results), nil
	case "markdown":
		return generateMarkdownReport(results), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// FilterResultsBySecurityControls returns only scan results that align with specific control filters
func FilterResultsBySecurityControls(results []models.TerraformScanResult, controls []string) []models.TerraformScanResult {
	if len(results) == 0 {
		return nil
	}

	var filters []string
	for _, control := range controls {
		if normalized := strings.ToLower(strings.TrimSpace(control)); normalized != "" {
			filters = append(filters, normalized)
		}
	}

	if len(filters) == 0 {
		copied := make([]models.TerraformScanResult, len(results))
		copy(copied, results)
		return copied
	}

	filtered := make([]models.TerraformScanResult, 0, len(results))
	for _, result := range results {
		if matchesSecurityFilters(result, filters) {
			filtered = append(filtered, result)
		}
	}

	return filtered
}

func matchesSecurityFilters(result models.TerraformScanResult, filters []string) bool {
	resourceTypeLower := strings.ToLower(result.ResourceType)
	resourceNameLower := strings.ToLower(result.ResourceName)

	relevance := make(map[string]struct{}, len(result.SecurityRelevance))
	for _, code := range result.SecurityRelevance {
		relevance[strings.ToLower(code)] = struct{}{}
	}

	for _, filter := range filters {
		if _, ok := relevance[filter]; ok {
			return true
		}

		if strings.Contains(resourceTypeLower, filter) || strings.Contains(resourceNameLower, filter) {
			return true
		}

		if filterMatchesConfiguration(result.Configuration, filter) {
			return true
		}

		switch filter {
		case "encryption", "encrypt":
			if RequiresEncryption(result.ResourceType) ||
				strings.Contains(resourceTypeLower, "kms") ||
				strings.Contains(resourceTypeLower, "encrypt") {
				return true
			}
		case "iam", "identity", "access":
			if strings.Contains(resourceTypeLower, "iam") ||
				strings.Contains(resourceTypeLower, "role") ||
				strings.Contains(resourceTypeLower, "policy") {
				return true
			}
		case "network", "security", "firewall":
			if strings.Contains(resourceTypeLower, "security_group") ||
				strings.Contains(resourceTypeLower, "firewall") ||
				strings.Contains(resourceTypeLower, "nacl") ||
				strings.Contains(resourceTypeLower, "network") {
				return true
			}
		}
	}

	return false
}

func filterMatchesConfiguration(configuration map[string]interface{}, filter string) bool {
	if len(configuration) == 0 {
		return false
	}

	for key, value := range configuration {
		keyLower := strings.ToLower(key)
		if keyLower == "_content" {
			continue
		}

		if strings.Contains(keyLower, filter) {
			return true
		}

		if strings.Contains(strings.ToLower(fmt.Sprint(value)), filter) {
			return true
		}
	}

	return false
}

// Helper functions

// IsResourceTypeOfInterest checks if a resource type matches the types we're looking for
func IsResourceTypeOfInterest(resourceType string, resourceTypes []string) bool {
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

// GetSecurityRelevance returns the security controls that a resource type relates to
func GetSecurityRelevance(resourceType string) []string {
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

// RequiresEncryption checks if a resource type should be encrypted
func RequiresEncryption(resourceType string) bool {
	encryptionRequiredTypes := map[string]bool{
		"aws_s3_bucket":                true,
		"aws_rds_cluster":              true,
		"aws_rds_instance":             true,
		"aws_ebs_volume":               true,
		"aws_efs_file_system":          true,
		"aws_redshift_cluster":         true,
		"azurerm_storage_account":      true,
		"azurerm_sql_database":         true,
		"google_storage_bucket":        true,
		"google_sql_database_instance": true,
	}

	return encryptionRequiredTypes[resourceType]
}

// CalculateTerraformRelevance calculates the relevance score based on scan results
func CalculateTerraformRelevance(results []models.TerraformScanResult) float64 {
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

// Private helper functions for calculations
func calculateEncryptionScore(analysis EncryptionAnalysis) float64 {
	if analysis.TotalResources == 0 {
		return 1.0
	}

	encryptionRequiredResources := len(analysis.EncryptedResources) + len(analysis.UnencryptedResources)
	if encryptionRequiredResources == 0 {
		return 1.0
	}

	return float64(len(analysis.EncryptedResources)) / float64(encryptionRequiredResources)
}

func calculateIAMSecurityScore(analysis IAMAnalysis) float64 {
	if analysis.TotalIAMResources == 0 {
		return 1.0
	}

	secureResources := analysis.TotalIAMResources - len(analysis.OverlyPermissive)
	return float64(secureResources) / float64(analysis.TotalIAMResources)
}

func calculateNetworkSecurityScore(analysis NetworkSecurityAnalysis) float64 {
	if analysis.TotalNetworkResources == 0 {
		return 1.0
	}

	secureResources := len(analysis.RestrictedAccess)
	totalConfiguredResources := len(analysis.OpenToInternet) + len(analysis.RestrictedAccess)

	if totalConfiguredResources == 0 {
		return 1.0
	}

	return float64(secureResources) / float64(totalConfiguredResources)
}

// Report generation functions
func generateCSVReport(results []models.TerraformScanResult) string {
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
		resourceType := escapeCSV(result.ResourceType)
		resourceName := escapeCSV(result.ResourceName)
		filePath := escapeCSV(result.FilePath)
		securityControlsCSV := escapeCSV(securityControls)
		keyConfigCSV := escapeCSV(keyConfigStr)

		report.WriteString(fmt.Sprintf("%s,%s,%s,%s,%s,%s\n",
			resourceType, resourceName, filePath, lineRange, securityControlsCSV, keyConfigCSV))
	}

	return report.String()
}

func generateMarkdownReport(results []models.TerraformScanResult) string {
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

func generateCSVSecurityReport(analysis SecurityAnalysis) string {
	var report strings.Builder
	report.WriteString("category,metric,value\n")

	switch a := analysis.(type) {
	case EncryptionAnalysis:
		report.WriteString(fmt.Sprintf("encryption,security_score,%.2f\n", a.EncryptionScore))
		report.WriteString(fmt.Sprintf("encryption,total_resources,%d\n", a.TotalResources))
		if len(a.EncryptedResources) > 0 {
			report.WriteString(fmt.Sprintf("encryption,encrypted_resources,%s\n", escapeCSV(strings.Join(a.EncryptedResources, "; "))))
		}
		if len(a.UnencryptedResources) > 0 {
			report.WriteString(fmt.Sprintf("encryption,unencrypted_resources,%s\n", escapeCSV(strings.Join(a.UnencryptedResources, "; "))))
		}
		if len(a.EncryptionMethods) > 0 {
			var methods []string
			for method, resources := range a.EncryptionMethods {
				methods = append(methods, fmt.Sprintf("%s (%d)", method, len(resources)))
			}
			sort.Strings(methods)
			report.WriteString(fmt.Sprintf("encryption,encryption_methods,%s\n", escapeCSV(strings.Join(methods, "; "))))
		}
	case IAMAnalysis:
		report.WriteString(fmt.Sprintf("iam,security_score,%.2f\n", a.SecurityScore))
		report.WriteString(fmt.Sprintf("iam,total_resources,%d\n", a.TotalIAMResources))
		if len(a.IAMRoles) > 0 {
			report.WriteString(fmt.Sprintf("iam,roles,%s\n", escapeCSV(strings.Join(a.IAMRoles, "; "))))
		}
		if len(a.IAMPolicies) > 0 {
			report.WriteString(fmt.Sprintf("iam,policies,%s\n", escapeCSV(strings.Join(a.IAMPolicies, "; "))))
		}
		if len(a.OverlyPermissive) > 0 {
			report.WriteString(fmt.Sprintf("iam,overly_permissive,%s\n", escapeCSV(strings.Join(a.OverlyPermissive, "; "))))
		}
	case NetworkSecurityAnalysis:
		report.WriteString(fmt.Sprintf("network,security_score,%.2f\n", a.SecurityScore))
		report.WriteString(fmt.Sprintf("network,total_resources,%d\n", a.TotalNetworkResources))
		if len(a.SecurityGroups) > 0 {
			report.WriteString(fmt.Sprintf("network,security_groups,%s\n", escapeCSV(strings.Join(a.SecurityGroups, "; "))))
		}
		if len(a.OpenToInternet) > 0 {
			report.WriteString(fmt.Sprintf("network,open_to_internet,%s\n", escapeCSV(strings.Join(a.OpenToInternet, "; "))))
		}
		if len(a.RestrictedAccess) > 0 {
			report.WriteString(fmt.Sprintf("network,restricted_access,%s\n", escapeCSV(strings.Join(a.RestrictedAccess, "; "))))
		}
	default:
		report.WriteString(fmt.Sprintf("unknown,type,%T\n", analysis))
	}

	return report.String()
}

func generateMarkdownSecurityReport(analysis SecurityAnalysis) string {
	var report strings.Builder
	report.WriteString("# Terraform Security Analysis\n\n")
	report.WriteString(fmt.Sprintf("_Generated at %s_\n\n", time.Now().Format(time.RFC3339)))

	switch a := analysis.(type) {
	case EncryptionAnalysis:
		report.WriteString("## Encryption Controls\n\n")
		report.WriteString(fmt.Sprintf("- Security score: %.2f\n", a.EncryptionScore))
		report.WriteString(fmt.Sprintf("- Total resources evaluated: %d\n", a.TotalResources))

		if len(a.EncryptedResources) > 0 {
			report.WriteString("\n**Encrypted Resources**\n\n")
			for _, resource := range a.EncryptedResources {
				report.WriteString(fmt.Sprintf("- %s\n", resource))
			}
		}

		if len(a.UnencryptedResources) > 0 {
			report.WriteString("\n**Unencrypted Resources (Action Needed)**\n\n")
			for _, resource := range a.UnencryptedResources {
				report.WriteString(fmt.Sprintf("- %s\n", resource))
			}
		}

		if len(a.EncryptionMethods) > 0 {
			report.WriteString("\n**Encryption Methods**\n\n")
			var methods []string
			for method, resources := range a.EncryptionMethods {
				methods = append(methods, fmt.Sprintf("%s (%d resources)", method, len(resources)))
			}
			sort.Strings(methods)
			for _, methodSummary := range methods {
				report.WriteString(fmt.Sprintf("- %s\n", methodSummary))
			}
		}
	case IAMAnalysis:
		report.WriteString("## Identity and Access Management\n\n")
		report.WriteString(fmt.Sprintf("- Security score: %.2f\n", a.SecurityScore))
		report.WriteString(fmt.Sprintf("- IAM roles discovered: %d\n", len(a.IAMRoles)))
		report.WriteString(fmt.Sprintf("- IAM policies discovered: %d\n", len(a.IAMPolicies)))

		if len(a.OverlyPermissive) > 0 {
			report.WriteString("\n**Overly Permissive Resources (Review Required)**\n\n")
			for _, resource := range a.OverlyPermissive {
				report.WriteString(fmt.Sprintf("- %s\n", resource))
			}
		}
	case NetworkSecurityAnalysis:
		report.WriteString("## Network Security\n\n")
		report.WriteString(fmt.Sprintf("- Security score: %.2f\n", a.SecurityScore))
		report.WriteString(fmt.Sprintf("- Security groups analysed: %d\n", len(a.SecurityGroups)))

		if len(a.OpenToInternet) > 0 {
			report.WriteString("\n**Open to Internet (0.0.0.0/0 or ::/0)**\n\n")
			for _, resource := range a.OpenToInternet {
				report.WriteString(fmt.Sprintf("- %s\n", resource))
			}
		}

		if len(a.RestrictedAccess) > 0 {
			report.WriteString("\n**Restricted Access Resources**\n\n")
			for _, resource := range a.RestrictedAccess {
				report.WriteString(fmt.Sprintf("- %s\n", resource))
			}
		}
	default:
		report.WriteString(fmt.Sprintf("No renderer for analysis type %T\n", analysis))
	}

	return report.String()
}

func escapeCSV(value string) string {
	// If value contains comma, newline, or quote, wrap in quotes and escape internal quotes
	if strings.Contains(value, ",") || strings.Contains(value, "\n") || strings.Contains(value, "\"") {
		value = strings.ReplaceAll(value, "\"", "\"\"")
		return "\"" + value + "\""
	}
	return value
}

// Define types for security analysis (these should be moved to models package eventually)

type SecurityAnalysis interface {
	GetSecurityScore() float64
}

type EncryptionAnalysis struct {
	EncryptedResources   []string
	UnencryptedResources []string
	EncryptionMethods    map[string][]string
	TotalResources       int
	EncryptionScore      float64
}

func (e EncryptionAnalysis) GetSecurityScore() float64 {
	return e.EncryptionScore
}

type IAMAnalysis struct {
	IAMRoles          []string
	IAMPolicies       []string
	OverlyPermissive  []string
	SecurityGaps      []string
	TotalIAMResources int
	SecurityScore     float64
}

func (i IAMAnalysis) GetSecurityScore() float64 {
	return i.SecurityScore
}

type NetworkSecurityAnalysis struct {
	SecurityGroups        []string
	OpenToInternet        []string
	RestrictedAccess      []string
	TotalNetworkResources int
	SecurityScore         float64
}

func (n NetworkSecurityAnalysis) GetSecurityScore() float64 {
	return n.SecurityScore
}

type ComplianceScore struct {
	OverallScore float64
	Categories   map[string]float64
}

type SecurityState struct {
	UnencryptedResources      []string
	OverlyPermissiveResources []string
	OpenNetworkResources      []string
}

type SecurityRequirements struct {
	RequireEncryption         bool
	RequireIAMPolicy          bool
	RequireNetworkRestriction bool
}

type SecurityGap struct {
	Type        string
	Resource    string
	Description string
	Severity    string
}
