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
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/auth"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

// GitHubSecurityFeaturesTool extracts repository security feature configuration for SOC2 audit evidence
type GitHubSecurityFeaturesTool struct {
	config       *config.GitHubToolConfig
	logger       logger.Logger
	cacheDir     string
	authProvider auth.AuthProvider
	apiClient    *GitHubAPIClient
}

// NewGitHubSecurityFeaturesTool creates a new GitHub security features extraction tool
func NewGitHubSecurityFeaturesTool(cfg *config.Config, log logger.Logger) Tool {
	// Set up cache directory
	cacheDir := filepath.Join(cfg.Storage.DataDir, ".cache", "github_security")

	// Create auth provider
	githubToken := cfg.Auth.GitHub.Token
	if githubToken == "" {
		githubToken = cfg.Evidence.Tools.GitHub.APIToken
	}

	var authProvider auth.AuthProvider
	if githubToken != "" {
		authProvider = auth.NewGitHubAuthProvider(githubToken, cfg.Auth.CacheDir, log)
	} else {
		authProvider = auth.NewGitHubAuthProvider("", cfg.Auth.CacheDir, log)
	}

	return &GitHubSecurityFeaturesTool{
		config:       &cfg.Evidence.Tools.GitHub,
		logger:       log,
		cacheDir:     cacheDir,
		authProvider: authProvider,
		apiClient:    NewGitHubAPIClient(cfg, log),
	}
}

// Name returns the tool name
func (gsft *GitHubSecurityFeaturesTool) Name() string {
	return "github-security-features"
}

// Description returns the tool description
func (gsft *GitHubSecurityFeaturesTool) Description() string {
	return "Extract repository security feature configuration for SOC2 audit evidence"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (gsft *GitHubSecurityFeaturesTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        gsft.Name(),
		Description: gsft.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repository": map[string]interface{}{
					"type":        "string",
					"description": "Repository in format 'owner/repo' (e.g., 'octocat/Hello-World')",
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "Output format for security features report",
					"enum":        []string{"detailed", "matrix", "summary"},
					"default":     "detailed",
				},
				"include_policy_analysis": map[string]interface{}{
					"type":        "boolean",
					"description": "Include security policy analysis",
					"default":     true,
				},
				"include_compliance_mapping": map[string]interface{}{
					"type":        "boolean",
					"description": "Include SOC2/compliance framework mapping",
					"default":     false,
				},
			},
			"required": []string{"repository"},
		},
	}
}

// Execute runs the GitHub security features extraction tool
func (gsft *GitHubSecurityFeaturesTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	startTime := time.Now()
	correlationID := GenerateCorrelationID()

	gsft.logger.Debug("Executing GitHub security features extraction",
		logger.String("correlation_id", correlationID),
		logger.Field{Key: "params", Value: params})

	// Check authentication
	authStatus := gsft.authProvider.GetStatus(ctx)
	if gsft.authProvider.IsAuthRequired() && !authStatus.Authenticated {
		if err := gsft.authProvider.Authenticate(ctx); err != nil {
			return "", nil, fmt.Errorf("GitHub authentication failed: %w", err)
		}
		authStatus = gsft.authProvider.GetStatus(ctx)
	}

	if !authStatus.Authenticated {
		return "", nil, fmt.Errorf("GitHub authentication required for security features API")
	}

	// Extract parameters
	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		return "", nil, fmt.Errorf("repository parameter is required")
	}

	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	outputFormat := "detailed"
	if of, ok := params["output_format"].(string); ok {
		outputFormat = of
	}

	includePolicyAnalysis := true
	if ipa, ok := params["include_policy_analysis"].(bool); ok {
		includePolicyAnalysis = ipa
	}

	includeComplianceMapping := false
	if icm, ok := params["include_compliance_mapping"].(bool); ok {
		includeComplianceMapping = icm
	}

	// Extract security features information
	securityInfo, err := gsft.extractSecurityFeatures(ctx, owner, repo, includePolicyAnalysis, includeComplianceMapping)
	if err != nil {
		return "", nil, fmt.Errorf("failed to extract security features: %w", err)
	}

	// Generate report based on output format
	var report string
	switch outputFormat {
	case "matrix":
		report = gsft.generateSecurityMatrix(securityInfo)
	case "summary":
		report = gsft.generateSecuritySummary(securityInfo)
	default: // detailed
		report = gsft.generateDetailedSecurityReport(securityInfo)
	}

	duration := time.Since(startTime)
	finalAuthStatus := gsft.authProvider.GetStatus(ctx)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "github-security-features",
		Resource:    fmt.Sprintf("GitHub repository security: %s", repository),
		Content:     report,
		Relevance:   gsft.calculateSecurityRelevance(securityInfo),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"repository":                 repository,
			"output_format":              outputFormat,
			"include_policy_analysis":    includePolicyAnalysis,
			"include_compliance_mapping": includeComplianceMapping,
			"security_score":             securityInfo.SecurityScore,
			"enabled_features":           len(securityInfo.EnabledFeatures),
			"total_features":             len(securityInfo.AllFeatures),
			"correlation_id":             correlationID,
			"duration_ms":                duration.Milliseconds(),
			"auth_status": map[string]interface{}{
				"authenticated": finalAuthStatus.Authenticated,
				"provider":      finalAuthStatus.Provider,
			},
		},
	}

	return report, source, nil
}

// SecurityFeaturesInfo contains comprehensive security features information
type SecurityFeaturesInfo struct {
	Repository              string                           `json:"repository"`
	SecuritySettings        models.GitHubSecuritySettings    `json:"security_settings"`
	BranchProtections       []models.GitHubBranch            `json:"branch_protections"`
	SecurityPolicies        []SecurityPolicyInfo             `json:"security_policies,omitempty"`
	ComplianceMapping       map[string][]string              `json:"compliance_mapping,omitempty"`
	AllFeatures             map[string]SecurityFeatureDetail `json:"all_features"`
	EnabledFeatures         []string                         `json:"enabled_features"`
	DisabledFeatures        []string                         `json:"disabled_features"`
	SecurityScore           float64                          `json:"security_score"`
	SecurityRecommendations []SecurityRecommendation         `json:"security_recommendations"`
	ExtractedAt             time.Time                        `json:"extracted_at"`
}

// SecurityFeatureDetail provides detailed information about a security feature
type SecurityFeatureDetail struct {
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	Description  string   `json:"description"`
	SOC2Controls []string `json:"soc2_controls,omitempty"`
	RiskLevel    string   `json:"risk_level"` // high, medium, low
	Category     string   `json:"category"`   // vulnerability, secrets, code_quality, access_control
}

// SecurityPolicyInfo contains information about security policies
type SecurityPolicyInfo struct {
	PolicyType     string    `json:"policy_type"`
	PolicyFile     string    `json:"policy_file"`
	LastUpdated    time.Time `json:"last_updated"`
	ContentSummary string    `json:"content_summary"`
}

// SecurityRecommendation provides actionable security recommendations
type SecurityRecommendation struct {
	Priority      string   `json:"priority"` // high, medium, low
	Category      string   `json:"category"` // feature, policy, access_control
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	ActionItems   []string `json:"action_items"`
	SOC2Relevance string   `json:"soc2_relevance,omitempty"`
}

// extractSecurityFeatures extracts comprehensive security features information
func (gsft *GitHubSecurityFeaturesTool) extractSecurityFeatures(ctx context.Context, owner, repo string, includePolicyAnalysis, includeComplianceMapping bool) (*SecurityFeaturesInfo, error) {
	info := &SecurityFeaturesInfo{
		Repository:  fmt.Sprintf("%s/%s", owner, repo),
		ExtractedAt: time.Now(),
		AllFeatures: make(map[string]SecurityFeatureDetail),
	}

	// Get security settings
	securitySettings, err := gsft.apiClient.GetRepositorySecurity(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get security settings: %w", err)
	}
	info.SecuritySettings = *securitySettings

	// Get branch protections
	branches, err := gsft.apiClient.GetRepositoryBranches(ctx, owner, repo)
	if err != nil {
		gsft.logger.Warn("Failed to get branch protections", logger.Field{Key: "error", Value: err})
	} else {
		// Only include protected branches
		var protectedBranches []models.GitHubBranch
		for _, branch := range branches {
			if branch.Protected {
				protectedBranches = append(protectedBranches, branch)
			}
		}
		info.BranchProtections = protectedBranches
	}

	// Build comprehensive feature analysis
	gsft.buildFeatureAnalysis(info)

	// Include policy analysis if requested
	if includePolicyAnalysis {
		policies, err := gsft.analyzeSecurityPolicies(ctx, owner, repo)
		if err != nil {
			gsft.logger.Warn("Failed to analyze security policies", logger.Field{Key: "error", Value: err})
		} else {
			info.SecurityPolicies = policies
		}
	}

	// Include compliance mapping if requested
	if includeComplianceMapping {
		info.ComplianceMapping = gsft.buildComplianceMapping(info)
	}

	// Calculate security score and recommendations
	info.SecurityScore = gsft.calculateSecurityScore(info)
	info.SecurityRecommendations = gsft.generateSecurityRecommendations(info)

	return info, nil
}

// buildFeatureAnalysis builds comprehensive feature analysis
func (gsft *GitHubSecurityFeaturesTool) buildFeatureAnalysis(info *SecurityFeaturesInfo) {
	// Define all security features with detailed information
	features := map[string]SecurityFeatureDetail{
		"vulnerability_alerts": {
			Name:         "Vulnerability Alerts",
			Enabled:      info.SecuritySettings.VulnerabilityAlertsEnabled,
			Description:  "Automated notifications when security vulnerabilities are found in dependencies",
			SOC2Controls: []string{"CC6.1", "CC6.2", "CC6.3"},
			RiskLevel:    "high",
			Category:     "vulnerability",
		},
		"automated_security_fixes": {
			Name:         "Automated Security Fixes (Dependabot)",
			Enabled:      info.SecuritySettings.AutomatedSecurityFixesEnabled,
			Description:  "Automatic pull requests to fix security vulnerabilities in dependencies",
			SOC2Controls: []string{"CC6.1", "CC6.3", "CC8.1"},
			RiskLevel:    "high",
			Category:     "vulnerability",
		},
		"secret_scanning": {
			Name:         "Secret Scanning",
			Enabled:      info.SecuritySettings.SecretScanningEnabled,
			Description:  "Automatic detection of secrets, tokens, and credentials in code",
			SOC2Controls: []string{"CC6.1", "CC6.7", "CC6.8"},
			RiskLevel:    "high",
			Category:     "secrets",
		},
		"code_scanning": {
			Name:         "Code Scanning",
			Enabled:      info.SecuritySettings.CodeScanningEnabled,
			Description:  "Static analysis to find security vulnerabilities in code",
			SOC2Controls: []string{"CC6.1", "CC6.2", "CC8.1"},
			RiskLevel:    "medium",
			Category:     "code_quality",
		},
		"dependency_graph": {
			Name:         "Dependency Graph",
			Enabled:      info.SecuritySettings.DependencyGraphEnabled,
			Description:  "Visualization and tracking of project dependencies",
			SOC2Controls: []string{"CC6.3", "CC8.1"},
			RiskLevel:    "medium",
			Category:     "vulnerability",
		},
		"security_advisories": {
			Name:         "Security Advisories",
			Enabled:      info.SecuritySettings.SecurityAdvisoryEnabled,
			Description:  "Ability to create and manage security advisories for vulnerabilities",
			SOC2Controls: []string{"CC6.1", "CC6.3"},
			RiskLevel:    "medium",
			Category:     "vulnerability",
		},
	}

	// Add branch protection as a security feature
	branchProtectionEnabled := len(info.BranchProtections) > 0
	features["branch_protection"] = SecurityFeatureDetail{
		Name:         "Branch Protection",
		Enabled:      branchProtectionEnabled,
		Description:  "Rules that protect important branches from unauthorized changes",
		SOC2Controls: []string{"CC6.1", "CC6.2", "CC6.3", "CC6.8"},
		RiskLevel:    "high",
		Category:     "access_control",
	}

	info.AllFeatures = features

	// Categorize enabled and disabled features
	for _, feature := range features {
		if feature.Enabled {
			info.EnabledFeatures = append(info.EnabledFeatures, feature.Name)
		} else {
			info.DisabledFeatures = append(info.DisabledFeatures, feature.Name)
		}
	}
}

// analyzeSecurityPolicies analyzes security-related policy files
func (gsft *GitHubSecurityFeaturesTool) analyzeSecurityPolicies(ctx context.Context, owner, repo string) ([]SecurityPolicyInfo, error) {
	// This would involve checking for common security policy files
	// For now, we'll return a placeholder implementation
	// In a full implementation, this would check for:
	// - SECURITY.md
	// - .github/SECURITY.md
	// - CODE_OF_CONDUCT.md
	// - CONTRIBUTING.md (security sections)

	var policies []SecurityPolicyInfo

	// Check for SECURITY.md file (common security policy location)
	// This would require additional API calls to check file existence and content
	// For brevity, we'll indicate that this feature is available but not implemented in this example

	policies = append(policies, SecurityPolicyInfo{
		PolicyType:     "security_disclosure",
		PolicyFile:     "SECURITY.md",
		LastUpdated:    time.Now().AddDate(0, -3, 0), // Example: 3 months ago
		ContentSummary: "Security policy analysis requires additional API implementation",
	})

	return policies, nil
}

// buildComplianceMapping builds SOC2 compliance mapping
func (gsft *GitHubSecurityFeaturesTool) buildComplianceMapping(info *SecurityFeaturesInfo) map[string][]string {
	mapping := make(map[string][]string)

	// Map enabled features to SOC2 controls
	for _, feature := range info.AllFeatures {
		if feature.Enabled && len(feature.SOC2Controls) > 0 {
			for _, control := range feature.SOC2Controls {
				mapping[control] = append(mapping[control], feature.Name)
			}
		}
	}

	return mapping
}

// calculateSecurityScore calculates overall security score
func (gsft *GitHubSecurityFeaturesTool) calculateSecurityScore(info *SecurityFeaturesInfo) float64 {
	if len(info.AllFeatures) == 0 {
		return 0.0
	}

	totalWeight := 0.0
	enabledWeight := 0.0

	for _, feature := range info.AllFeatures {
		// Weight features by risk level
		weight := 1.0
		switch feature.RiskLevel {
		case "high":
			weight = 3.0
		case "medium":
			weight = 2.0
		case "low":
			weight = 1.0
		}

		totalWeight += weight
		if feature.Enabled {
			enabledWeight += weight
		}
	}

	if totalWeight == 0 {
		return 0.0
	}

	return enabledWeight / totalWeight
}

// generateSecurityRecommendations generates actionable security recommendations
func (gsft *GitHubSecurityFeaturesTool) generateSecurityRecommendations(info *SecurityFeaturesInfo) []SecurityRecommendation {
	var recommendations []SecurityRecommendation

	// Check for high-priority disabled features
	for _, feature := range info.AllFeatures {
		if !feature.Enabled && feature.RiskLevel == "high" {
			rec := SecurityRecommendation{
				Priority:    "high",
				Category:    "feature",
				Title:       fmt.Sprintf("Enable %s", feature.Name),
				Description: fmt.Sprintf("%s is currently disabled, which poses a security risk", feature.Name),
				ActionItems: []string{
					"Navigate to repository Settings > Security & Analysis",
					fmt.Sprintf("Enable %s", feature.Name),
					"Configure appropriate notifications and workflows",
				},
			}
			if len(feature.SOC2Controls) > 0 {
				rec.SOC2Relevance = fmt.Sprintf("Required for SOC2 controls: %s", strings.Join(feature.SOC2Controls, ", "))
			}
			recommendations = append(recommendations, rec)
		}
	}

	// Check for missing branch protection
	if len(info.BranchProtections) == 0 {
		recommendations = append(recommendations, SecurityRecommendation{
			Priority:    "high",
			Category:    "access_control",
			Title:       "Implement Branch Protection Rules",
			Description: "No branch protection rules are configured, allowing unrestricted access to important branches",
			ActionItems: []string{
				"Navigate to repository Settings > Branches",
				"Add protection rules for main/master branch",
				"Enable required reviews and status checks",
				"Consider restricting push access to specific users/teams",
			},
			SOC2Relevance: "Required for SOC2 controls: CC6.1, CC6.2, CC6.3, CC6.8",
		})
	}

	// Check security score for overall recommendations
	if info.SecurityScore < 0.5 {
		recommendations = append(recommendations, SecurityRecommendation{
			Priority:    "medium",
			Category:    "policy",
			Title:       "Improve Overall Security Posture",
			Description: fmt.Sprintf("Current security score (%.1f%%) indicates room for improvement", info.SecurityScore*100),
			ActionItems: []string{
				"Review and enable disabled security features",
				"Implement comprehensive branch protection rules",
				"Create and maintain security policies",
				"Regular security audits and reviews",
			},
			SOC2Relevance: "Supports multiple SOC2 controls across all categories",
		})
	}

	return recommendations
}

// generateDetailedSecurityReport creates a comprehensive security features report
func (gsft *GitHubSecurityFeaturesTool) generateDetailedSecurityReport(info *SecurityFeaturesInfo) string {
	var report strings.Builder

	report.WriteString("# GitHub Security Features Analysis\n\n")
	report.WriteString(fmt.Sprintf("**Repository:** %s\n", info.Repository))
	report.WriteString(fmt.Sprintf("**Generated:** %s\n\n", info.ExtractedAt.Format("2006-01-02 15:04:05")))

	// Executive Summary
	report.WriteString("## Executive Summary\n\n")
	report.WriteString(fmt.Sprintf("- **Overall Security Score:** %.1f%% (%d/%d features enabled)\n",
		info.SecurityScore*100, len(info.EnabledFeatures), len(info.AllFeatures)))
	report.WriteString(fmt.Sprintf("- **Enabled Security Features:** %d\n", len(info.EnabledFeatures)))
	report.WriteString(fmt.Sprintf("- **Disabled Security Features:** %d\n", len(info.DisabledFeatures)))
	report.WriteString(fmt.Sprintf("- **Protected Branches:** %d\n", len(info.BranchProtections)))
	report.WriteString(fmt.Sprintf("- **Security Recommendations:** %d\n\n", len(info.SecurityRecommendations)))

	// Security Features Status
	report.WriteString("## Security Features Status\n\n")
	report.WriteString("| Feature | Status | Risk Level | Category | SOC2 Controls |\n")
	report.WriteString("|---------|--------|------------|----------|---------------|\n")

	for _, feature := range info.AllFeatures {
		status := "❌ Disabled"
		if feature.Enabled {
			status = "✅ Enabled"
		}

		controls := strings.Join(feature.SOC2Controls, ", ")
		if controls == "" {
			controls = "None"
		}

		report.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			feature.Name, status, cases.Title(language.English).String(feature.RiskLevel),
			cases.Title(language.English).String(feature.Category), controls))
	}
	report.WriteString("\n")

	// Detailed Feature Analysis
	report.WriteString("## Detailed Feature Analysis\n\n")

	// Group by category
	categories := map[string][]SecurityFeatureDetail{
		"vulnerability":  {},
		"secrets":        {},
		"code_quality":   {},
		"access_control": {},
	}

	for _, feature := range info.AllFeatures {
		categories[feature.Category] = append(categories[feature.Category], feature)
	}

	for category, features := range categories {
		if len(features) == 0 {
			continue
		}

		report.WriteString(fmt.Sprintf("### %s\n\n", cases.Title(language.English).String(strings.ReplaceAll(category, "_", " "))))

		for _, feature := range features {
			status := "❌ **Disabled**"
			if feature.Enabled {
				status = "✅ **Enabled**"
			}

			report.WriteString(fmt.Sprintf("#### %s\n", feature.Name))
			report.WriteString(fmt.Sprintf("- **Status:** %s\n", status))
			report.WriteString(fmt.Sprintf("- **Description:** %s\n", feature.Description))
			report.WriteString(fmt.Sprintf("- **Risk Level:** %s\n", cases.Title(language.English).String(feature.RiskLevel)))
			if len(feature.SOC2Controls) > 0 {
				report.WriteString(fmt.Sprintf("- **SOC2 Controls:** %s\n", strings.Join(feature.SOC2Controls, ", ")))
			}
			report.WriteString("\n")
		}
	}

	// Branch Protection Analysis
	if len(info.BranchProtections) > 0 {
		report.WriteString("## Branch Protection Analysis\n\n")

		for _, branch := range info.BranchProtections {
			report.WriteString(fmt.Sprintf("### Branch: %s\n", branch.Name))

			if branch.Protection != nil {
				report.WriteString("- **Protection Status:** ✅ Protected\n")

				if branch.Protection.RequiredPullRequestReviews != nil {
					report.WriteString(fmt.Sprintf("- **Required Reviews:** %d\n",
						branch.Protection.RequiredPullRequestReviews.RequiredApprovingReviewCount))
					report.WriteString(fmt.Sprintf("- **Code Owner Reviews:** %v\n",
						branch.Protection.RequiredPullRequestReviews.RequireCodeOwnerReviews))
					report.WriteString(fmt.Sprintf("- **Dismiss Stale Reviews:** %v\n",
						branch.Protection.RequiredPullRequestReviews.DismissStaleReviews))
				}

				if branch.Protection.RequiredStatusChecks != nil {
					report.WriteString(fmt.Sprintf("- **Required Status Checks:** %v\n",
						branch.Protection.RequiredStatusChecks.Contexts))
					report.WriteString(fmt.Sprintf("- **Strict Checks:** %v\n",
						branch.Protection.RequiredStatusChecks.Strict))
				}

				report.WriteString(fmt.Sprintf("- **Enforce Admins:** %v\n", branch.Protection.EnforceAdmins.Enabled))
				report.WriteString(fmt.Sprintf("- **Allow Force Pushes:** %v\n", branch.Protection.AllowForcePushes.Enabled))
				report.WriteString(fmt.Sprintf("- **Allow Deletions:** %v\n", branch.Protection.AllowDeletions.Enabled))
			} else {
				report.WriteString("- **Protection Status:** ❌ Unprotected\n")
			}
			report.WriteString("\n")
		}
	} else {
		report.WriteString("## Branch Protection Analysis\n\n")
		report.WriteString("⚠️ **No branch protection rules configured**\n\n")
		report.WriteString("This poses a significant security risk as anyone with repository access can:\n")
		report.WriteString("- Push directly to important branches\n")
		report.WriteString("- Bypass code reviews\n")
		report.WriteString("- Delete branches\n")
		report.WriteString("- Force push and rewrite history\n\n")
	}

	// Compliance Mapping
	if len(info.ComplianceMapping) > 0 {
		report.WriteString("## SOC2 Compliance Mapping\n\n")
		report.WriteString("| SOC2 Control | Covered by Features |\n")
		report.WriteString("|--------------|---------------------|\n")

		for control, features := range info.ComplianceMapping {
			report.WriteString(fmt.Sprintf("| %s | %s |\n", control, strings.Join(features, ", ")))
		}
		report.WriteString("\n")
	}

	// Security Recommendations
	if len(info.SecurityRecommendations) > 0 {
		report.WriteString("## Security Recommendations\n\n")

		// Group by priority
		priorities := []string{"high", "medium", "low"}
		for _, priority := range priorities {
			var priorityRecs []SecurityRecommendation
			for _, rec := range info.SecurityRecommendations {
				if rec.Priority == priority {
					priorityRecs = append(priorityRecs, rec)
				}
			}

			if len(priorityRecs) == 0 {
				continue
			}

			priorityEmoji := map[string]string{
				"high":   "🔴",
				"medium": "🟡",
				"low":    "🟢",
			}

			report.WriteString(fmt.Sprintf("### %s %s Priority\n\n", priorityEmoji[priority], cases.Title(language.English).String(priority)))

			for _, rec := range priorityRecs {
				report.WriteString(fmt.Sprintf("#### %s\n", rec.Title))
				report.WriteString(fmt.Sprintf("**Description:** %s\n\n", rec.Description))
				report.WriteString("**Action Items:**\n")
				for _, item := range rec.ActionItems {
					report.WriteString(fmt.Sprintf("1. %s\n", item))
				}
				if rec.SOC2Relevance != "" {
					report.WriteString(fmt.Sprintf("\n**SOC2 Relevance:** %s\n", rec.SOC2Relevance))
				}
				report.WriteString("\n")
			}
		}
	}

	return report.String()
}

// generateSecurityMatrix creates a matrix view of security features
func (gsft *GitHubSecurityFeaturesTool) generateSecurityMatrix(info *SecurityFeaturesInfo) string {
	var report strings.Builder

	report.WriteString("# GitHub Security Features Matrix\n\n")
	report.WriteString(fmt.Sprintf("**Repository:** %s\n", info.Repository))
	report.WriteString(fmt.Sprintf("**Security Score:** %.1f%%\n", info.SecurityScore*100))
	report.WriteString(fmt.Sprintf("**Generated:** %s\n\n", info.ExtractedAt.Format("2006-01-02 15:04:05")))

	// Security Features Matrix
	report.WriteString("## Security Features Matrix\n\n")
	report.WriteString("| Category | Feature | Status | Risk Level | SOC2 Controls |\n")
	report.WriteString("|----------|---------|--------|------------|---------------|\n")

	// Group and sort by category
	categories := []string{"vulnerability", "secrets", "code_quality", "access_control"}
	for _, category := range categories {
		for _, feature := range info.AllFeatures {
			if feature.Category != category {
				continue
			}

			status := "❌"
			if feature.Enabled {
				status = "✅"
			}

			controls := strings.Join(feature.SOC2Controls, ", ")
			if controls == "" {
				controls = "None"
			}

			report.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
				cases.Title(language.English).String(strings.ReplaceAll(category, "_", " ")),
				feature.Name, status, cases.Title(language.English).String(feature.RiskLevel), controls))
		}
	}
	report.WriteString("\n")

	return report.String()
}

// generateSecuritySummary creates a summary of security features
func (gsft *GitHubSecurityFeaturesTool) generateSecuritySummary(info *SecurityFeaturesInfo) string {
	var report strings.Builder

	report.WriteString("# GitHub Security Features Summary\n\n")
	report.WriteString(fmt.Sprintf("**Repository:** %s\n", info.Repository))
	report.WriteString(fmt.Sprintf("**Generated:** %s\n\n", info.ExtractedAt.Format("2006-01-02 15:04:05")))

	// Security Score
	report.WriteString("## Security Score\n\n")
	scoreEmoji := "🔴"
	scoreStatus := "Poor"
	if info.SecurityScore >= 0.8 {
		scoreEmoji = "🟢"
		scoreStatus = "Excellent"
	} else if info.SecurityScore >= 0.6 {
		scoreEmoji = "🟡"
		scoreStatus = "Good"
	} else if info.SecurityScore >= 0.4 {
		scoreEmoji = "🟠"
		scoreStatus = "Fair"
	}

	report.WriteString(fmt.Sprintf("%s **Overall Score:** %.1f%% (%s)\n\n",
		scoreEmoji, info.SecurityScore*100, scoreStatus))

	// Feature Summary
	report.WriteString("## Feature Summary\n\n")
	report.WriteString(fmt.Sprintf("- **Total Features:** %d\n", len(info.AllFeatures)))
	report.WriteString(fmt.Sprintf("- **Enabled:** %d (%.1f%%)\n",
		len(info.EnabledFeatures),
		float64(len(info.EnabledFeatures))/float64(len(info.AllFeatures))*100))
	report.WriteString(fmt.Sprintf("- **Disabled:** %d (%.1f%%)\n",
		len(info.DisabledFeatures),
		float64(len(info.DisabledFeatures))/float64(len(info.AllFeatures))*100))
	report.WriteString("\n")

	// Quick Status
	report.WriteString("## Quick Status\n\n")
	report.WriteString("**Enabled Features:**\n")
	for _, feature := range info.EnabledFeatures {
		report.WriteString(fmt.Sprintf("- ✅ %s\n", feature))
	}

	if len(info.DisabledFeatures) > 0 {
		report.WriteString("\n**Disabled Features:**\n")
		for _, feature := range info.DisabledFeatures {
			report.WriteString(fmt.Sprintf("- ❌ %s\n", feature))
		}
	}

	report.WriteString("\n")

	// Top Recommendations
	if len(info.SecurityRecommendations) > 0 {
		report.WriteString("## Top Security Recommendations\n\n")
		count := 0
		for _, rec := range info.SecurityRecommendations {
			if rec.Priority == "high" && count < 3 {
				report.WriteString(fmt.Sprintf("🔴 **%s:** %s\n", rec.Title, rec.Description))
				count++
			}
		}
		if count == 0 {
			report.WriteString("🟢 No high-priority security issues identified.\n")
		}
		report.WriteString("\n")
	}

	return report.String()
}

// Helper methods

func (gsft *GitHubSecurityFeaturesTool) calculateSecurityRelevance(info *SecurityFeaturesInfo) float64 {
	// Base relevance
	relevance := 0.5

	// Higher relevance for repositories with security features enabled
	if info.SecurityScore > 0.5 {
		relevance += 0.2
	}

	// Higher relevance if there are security recommendations
	if len(info.SecurityRecommendations) > 0 {
		relevance += 0.1
	}

	// Higher relevance if branch protections are configured
	if len(info.BranchProtections) > 0 {
		relevance += 0.1
	}

	// Higher relevance for compliance mapping
	if len(info.ComplianceMapping) > 0 {
		relevance += 0.1
	}

	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}
