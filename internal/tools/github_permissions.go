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
)

// GitHubPermissionsTool extracts comprehensive repository access controls and permissions for SOC2 audit evidence
type GitHubPermissionsTool struct {
	config       *config.GitHubToolConfig
	logger       logger.Logger
	cacheDir     string
	authProvider auth.AuthProvider
	apiClient    *GitHubAPIClient
}

// NewGitHubPermissionsTool creates a new GitHub permissions extraction tool
func NewGitHubPermissionsTool(cfg *config.Config, log logger.Logger) Tool {
	// Set up cache directory
	cacheDir := filepath.Join(cfg.Storage.DataDir, ".cache", "github_permissions")

	// Create auth provider - token is populated by config.Load() from multiple sources
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

	return &GitHubPermissionsTool{
		config:       &cfg.Evidence.Tools.GitHub,
		logger:       log,
		cacheDir:     cacheDir,
		authProvider: authProvider,
		apiClient:    NewGitHubAPIClient(cfg, log),
	}
}

// Name returns the tool name
func (gpt *GitHubPermissionsTool) Name() string {
	return "github-permissions"
}

// Description returns the tool description
func (gpt *GitHubPermissionsTool) Description() string {
	return "Extract comprehensive repository access controls and permissions for SOC2 audit evidence"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (gpt *GitHubPermissionsTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        gpt.Name(),
		Description: gpt.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repository": map[string]interface{}{
					"type":        "string",
					"description": "Repository in format 'owner/repo' (e.g., 'octocat/Hello-World')",
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "Output format for the access control matrix",
					"enum":        []string{"detailed", "matrix", "summary"},
					"default":     "detailed",
				},
				"include_org_members": map[string]interface{}{
					"type":        "boolean",
					"description": "Include organization member information if available",
					"default":     true,
				},
				"use_cache": map[string]interface{}{
					"type":        "boolean",
					"description": "Use cached API results when available",
					"default":     true,
				},
			},
			"required": []string{"repository"},
		},
	}
}

// Execute runs the GitHub permissions extraction tool
func (gpt *GitHubPermissionsTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	startTime := time.Now()
	correlationID := GenerateCorrelationID()

	gpt.logger.Debug("Executing GitHub permissions extraction",
		logger.String("correlation_id", correlationID),
		logger.Field{Key: "params", Value: params})

	// Check authentication status
	authStatus := gpt.authProvider.GetStatus(ctx)

	// Attempt authentication if required and not already authenticated
	if gpt.authProvider.IsAuthRequired() && !authStatus.Authenticated {
		if err := gpt.authProvider.Authenticate(ctx); err != nil {
			authStatus.Error = err.Error()
			return "", nil, fmt.Errorf("GitHub authentication failed: %w", err)
		}
		authStatus = gpt.authProvider.GetStatus(ctx)
	}

	// Validate authentication is available for permissions API
	if !authStatus.Authenticated {
		return "", nil, fmt.Errorf("GitHub authentication required for permissions API access")
	}

	// Extract parameters
	repository, ok := params["repository"].(string)
	if !ok || repository == "" {
		return "", nil, fmt.Errorf("repository parameter is required")
	}

	// Parse repository owner/name
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", nil, fmt.Errorf("repository must be in format 'owner/repo'")
	}
	owner, repo := parts[0], parts[1]

	outputFormat := "detailed"
	if of, ok := params["output_format"].(string); ok {
		outputFormat = of
	}

	includeOrgMembers := true
	if iom, ok := params["include_org_members"].(bool); ok {
		includeOrgMembers = iom
	}

	useCache := true
	if uc, ok := params["use_cache"].(bool); ok {
		useCache = uc
	}

	// Extract repository access control matrix
	matrix, err := gpt.extractAccessControlMatrix(ctx, owner, repo, includeOrgMembers, useCache)
	if err != nil {
		return "", nil, fmt.Errorf("failed to extract access control matrix: %w", err)
	}

	// Generate report based on output format
	var report string
	switch outputFormat {
	case "matrix":
		report = gpt.generatePermissionMatrix(matrix)
	case "summary":
		report = gpt.generateAccessSummary(matrix)
	default: // detailed
		report = gpt.generateDetailedReport(matrix)
	}

	duration := time.Since(startTime)
	finalAuthStatus := gpt.authProvider.GetStatus(ctx)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "github-permissions",
		Resource:    fmt.Sprintf("GitHub repository: %s", repository),
		Content:     report,
		Relevance:   CalculateRelevanceScore(matrix),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"repository":          repository,
			"output_format":       outputFormat,
			"include_org_members": includeOrgMembers,
			"total_collaborators": matrix.AccessSummary.TotalCollaborators,
			"total_teams":         matrix.AccessSummary.TotalTeams,
			"protected_branches":  len(matrix.AccessSummary.ProtectedBranches),
			"correlation_id":      correlationID,
			"duration_ms":         duration.Milliseconds(),
			"auth_status": map[string]interface{}{
				"authenticated": finalAuthStatus.Authenticated,
				"provider":      finalAuthStatus.Provider,
				"cache_used":    finalAuthStatus.CacheUsed,
			},
			"data_source": "api",
		},
	}

	return report, source, nil
}

// extractAccessControlMatrix extracts comprehensive access control information
func (gpt *GitHubPermissionsTool) extractAccessControlMatrix(ctx context.Context, owner, repo string, includeOrgMembers, useCache bool) (*models.GitHubAccessControlMatrix, error) {
	matrix := &models.GitHubAccessControlMatrix{
		Repository: models.GitHubRepositoryInfo{
			Name:     repo,
			FullName: fmt.Sprintf("%s/%s", owner, repo),
			Owner:    owner,
		},
		ExtractedAt: time.Now(),
	}

	// Get repository basic info (this is done as part of other calls, so we'll populate it there)

	// Get collaborators
	collaborators, err := gpt.apiClient.GetRepositoryCollaborators(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get collaborators: %w", err)
	}
	matrix.Collaborators = collaborators

	// Get teams
	teams, err := gpt.apiClient.GetRepositoryTeams(ctx, owner, repo)
	if err != nil {
		gpt.logger.Warn("Failed to get repository teams", logger.Field{Key: "error", Value: err})
		matrix.Teams = []models.GitHubTeam{} // Continue without teams if this fails
	} else {
		matrix.Teams = teams
	}

	// Get branches with protection rules
	branches, err := gpt.apiClient.GetRepositoryBranches(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}
	matrix.Branches = branches

	// Get deployment environments
	environments, err := gpt.apiClient.GetDeploymentEnvironments(ctx, owner, repo)
	if err != nil {
		gpt.logger.Warn("Failed to get deployment environments", logger.Field{Key: "error", Value: err})
		matrix.Environments = []models.GitHubEnvironment{} // Continue without environments if this fails
	} else {
		matrix.Environments = environments
	}

	// Get security settings
	securitySettings, err := gpt.apiClient.GetRepositorySecurity(ctx, owner, repo)
	if err != nil {
		gpt.logger.Warn("Failed to get security settings", logger.Field{Key: "error", Value: err})
		matrix.SecuritySettings = models.GitHubSecuritySettings{} // Continue with empty security settings
	} else {
		matrix.SecuritySettings = *securitySettings
	}

	// Get organization information if requested
	if includeOrgMembers {
		orgMembers, err := gpt.apiClient.GetOrganizationMembers(ctx, owner)
		if err != nil {
			gpt.logger.Warn("Failed to get organization members", logger.Field{Key: "error", Value: err})
		} else {
			matrix.OrganizationInfo = &models.GitHubOrganizationInfo{
				Login:   owner,
				Members: orgMembers,
			}
		}
	}

	// Generate access summary
	matrix.AccessSummary = gpt.generateAccessSummaryData(matrix)

	return matrix, nil
}

// generateAccessSummaryData creates a summary of access controls
func (gpt *GitHubPermissionsTool) generateAccessSummaryData(matrix *models.GitHubAccessControlMatrix) models.GitHubAccessSummary {
	summary := models.GitHubAccessSummary{
		TotalCollaborators: len(matrix.Collaborators),
		TotalTeams:         len(matrix.Teams),
	}

	// Categorize collaborators by permission level
	for _, collab := range matrix.Collaborators {
		username := collab.Login
		switch collab.Permissions.Permission {
		case "admin":
			summary.AdminUsers = append(summary.AdminUsers, username)
		case "maintain":
			summary.MaintainUsers = append(summary.MaintainUsers, username)
		case "push", "write":
			summary.PushUsers = append(summary.PushUsers, username)
		case "triage":
			summary.TriageUsers = append(summary.TriageUsers, username)
		case "pull", "read":
			summary.ReadOnlyUsers = append(summary.ReadOnlyUsers, username)
		}
	}

	// Categorize teams by permission level
	for _, team := range matrix.Teams {
		teamName := team.Name
		summary.TotalTeamMembers += len(team.Members)

		switch team.Permission {
		case "admin":
			summary.AdminTeams = append(summary.AdminTeams, teamName)
		case "maintain", "push", "write":
			summary.WriteTeams = append(summary.WriteTeams, teamName)
		case "pull", "read":
			summary.ReadTeams = append(summary.ReadTeams, teamName)
		}
	}

	// Find protected branches
	for _, branch := range matrix.Branches {
		if branch.Protected {
			summary.ProtectedBranches = append(summary.ProtectedBranches, branch.Name)
		}
	}

	// Find protected environments
	for _, env := range matrix.Environments {
		if len(env.ProtectionRules) > 0 {
			summary.ProtectedEnvironments = append(summary.ProtectedEnvironments, env.Name)
		}
	}

	// Analyze security features
	summary.SecurityFeatures = gpt.analyzeSecurityFeatures(matrix.SecuritySettings)

	return summary
}

// analyzeSecurityFeatures analyzes enabled/disabled security features
func (gpt *GitHubPermissionsTool) analyzeSecurityFeatures(settings models.GitHubSecuritySettings) models.GitHubSecurityFeatureSummary {
	allFeatures := map[string]bool{
		"Vulnerability Alerts":     settings.VulnerabilityAlertsEnabled,
		"Automated Security Fixes": settings.AutomatedSecurityFixesEnabled,
		"Secret Scanning":          settings.SecretScanningEnabled,
		"Code Scanning":            settings.CodeScanningEnabled,
		"Dependency Graph":         settings.DependencyGraphEnabled,
		"Security Advisories":      settings.SecurityAdvisoryEnabled,
	}

	summary := models.GitHubSecurityFeatureSummary{
		TotalFeatures: len(allFeatures),
	}

	for feature, enabled := range allFeatures {
		if enabled {
			summary.EnabledFeatures = append(summary.EnabledFeatures, feature)
			summary.EnabledCount++
		} else {
			summary.DisabledFeatures = append(summary.DisabledFeatures, feature)
		}
	}

	// Calculate security score
	if summary.TotalFeatures > 0 {
		summary.SecurityScore = float64(summary.EnabledCount) / float64(summary.TotalFeatures)
	}

	return summary
}

// generateDetailedReport creates a comprehensive detailed report
func (gpt *GitHubPermissionsTool) generateDetailedReport(matrix *models.GitHubAccessControlMatrix) string {
	var report strings.Builder

	report.WriteString("# GitHub Repository Access Control Matrix\n\n")
	report.WriteString(fmt.Sprintf("**Repository:** %s\n", matrix.Repository.FullName))
	report.WriteString(fmt.Sprintf("**Generated:** %s\n\n", matrix.ExtractedAt.Format("2006-01-02 15:04:05")))

	// Executive Summary
	report.WriteString("## Executive Summary\n\n")
	report.WriteString(fmt.Sprintf("- **Total Collaborators:** %d\n", matrix.AccessSummary.TotalCollaborators))
	report.WriteString(fmt.Sprintf("- **Total Teams:** %d\n", matrix.AccessSummary.TotalTeams))
	report.WriteString(fmt.Sprintf("- **Total Team Members:** %d\n", matrix.AccessSummary.TotalTeamMembers))
	report.WriteString(fmt.Sprintf("- **Protected Branches:** %d\n", len(matrix.AccessSummary.ProtectedBranches)))
	report.WriteString(fmt.Sprintf("- **Protected Environments:** %d\n", len(matrix.AccessSummary.ProtectedEnvironments)))
	report.WriteString(fmt.Sprintf("- **Security Score:** %.1f%% (%d/%d features enabled)\n\n",
		matrix.AccessSummary.SecurityFeatures.SecurityScore*100,
		matrix.AccessSummary.SecurityFeatures.EnabledCount,
		matrix.AccessSummary.SecurityFeatures.TotalFeatures))

	// Direct Collaborators
	report.WriteString("## Direct Repository Collaborators\n\n")
	if len(matrix.Collaborators) == 0 {
		report.WriteString("No direct collaborators found.\n\n")
	} else {
		report.WriteString("| Username | Permission Level | Admin | Push | Pull |\n")
		report.WriteString("|----------|------------------|-------|------|------|\n")
		for _, collab := range matrix.Collaborators {
			report.WriteString(fmt.Sprintf("| %s | %s | %v | %v | %v |\n",
				collab.Login,
				collab.Permissions.Permission,
				collab.Permissions.Admin,
				collab.Permissions.Push,
				collab.Permissions.Pull))
		}
		report.WriteString("\n")
	}

	// Team Access
	report.WriteString("## Team Access\n\n")
	if len(matrix.Teams) == 0 {
		report.WriteString("No team access configured.\n\n")
	} else {
		for _, team := range matrix.Teams {
			report.WriteString(fmt.Sprintf("### Team: %s\n", team.Name))
			report.WriteString(fmt.Sprintf("- **Permission:** %s\n", team.Permission))
			report.WriteString(fmt.Sprintf("- **Privacy:** %s\n", team.Privacy))
			report.WriteString(fmt.Sprintf("- **Members:** %d\n", len(team.Members)))

			if len(team.Members) > 0 {
				report.WriteString("- **Member List:**\n")
				for _, member := range team.Members {
					report.WriteString(fmt.Sprintf("  - %s (%s)\n", member.Login, member.Type))
				}
			}
			report.WriteString("\n")
		}
	}

	// Branch Protection Rules
	report.WriteString("## Branch Protection Rules\n\n")
	protectedCount := 0
	for _, branch := range matrix.Branches {
		if branch.Protected && branch.Protection != nil {
			protectedCount++
			report.WriteString(fmt.Sprintf("### Branch: %s\n", branch.Name))
			report.WriteString("- **Protected:** Yes\n")

			if branch.Protection.RequiredPullRequestReviews != nil {
				report.WriteString(fmt.Sprintf("- **Required Reviews:** %d\n",
					branch.Protection.RequiredPullRequestReviews.RequiredApprovingReviewCount))
				report.WriteString(fmt.Sprintf("- **Require Code Owner Reviews:** %v\n",
					branch.Protection.RequiredPullRequestReviews.RequireCodeOwnerReviews))
				report.WriteString(fmt.Sprintf("- **Dismiss Stale Reviews:** %v\n",
					branch.Protection.RequiredPullRequestReviews.DismissStaleReviews))
			}

			if branch.Protection.RequiredStatusChecks != nil {
				report.WriteString(fmt.Sprintf("- **Required Status Checks:** %v\n",
					branch.Protection.RequiredStatusChecks.Contexts))
				report.WriteString(fmt.Sprintf("- **Strict Status Checks:** %v\n",
					branch.Protection.RequiredStatusChecks.Strict))
			}

			report.WriteString(fmt.Sprintf("- **Enforce Admins:** %v\n", branch.Protection.EnforceAdmins.Enabled))
			report.WriteString(fmt.Sprintf("- **Allow Force Pushes:** %v\n", branch.Protection.AllowForcePushes.Enabled))
			report.WriteString(fmt.Sprintf("- **Allow Deletions:** %v\n", branch.Protection.AllowDeletions.Enabled))
			report.WriteString("\n")
		}
	}

	if protectedCount == 0 {
		report.WriteString("No branch protection rules configured.\n\n")
	}

	// Deployment Environments
	report.WriteString("## Deployment Environments\n\n")
	if len(matrix.Environments) == 0 {
		report.WriteString("No deployment environments configured.\n\n")
	} else {
		for _, env := range matrix.Environments {
			report.WriteString(fmt.Sprintf("### Environment: %s\n", env.Name))

			if len(env.ProtectionRules) == 0 {
				report.WriteString("- **Protection Rules:** None\n\n")
			} else {
				report.WriteString("- **Protection Rules:**\n")
				for _, rule := range env.ProtectionRules {
					report.WriteString(fmt.Sprintf("  - **Type:** %s\n", rule.Type))
					if len(rule.RequiredReviewers) > 0 {
						report.WriteString("  - **Required Reviewers:**\n")
						for _, reviewer := range rule.RequiredReviewers {
							switch reviewer.Type {
							case "User":
								report.WriteString(fmt.Sprintf("    - User: %s\n", reviewer.Login))
							case "Team":
								report.WriteString(fmt.Sprintf("    - Team: %s\n", reviewer.Slug))
							}
						}
					}
					if rule.WaitTimer > 0 {
						report.WriteString(fmt.Sprintf("  - **Wait Timer:** %d minutes\n", rule.WaitTimer))
					}
				}
				report.WriteString("\n")
			}
		}
	}

	// Security Settings
	report.WriteString("## Security Settings\n\n")
	report.WriteString("| Feature | Status |\n")
	report.WriteString("|---------|--------|\n")
	report.WriteString(fmt.Sprintf("| Vulnerability Alerts | %s |\n",
		FormatEnabled(matrix.SecuritySettings.VulnerabilityAlertsEnabled)))
	report.WriteString(fmt.Sprintf("| Automated Security Fixes | %s |\n",
		FormatEnabled(matrix.SecuritySettings.AutomatedSecurityFixesEnabled)))
	report.WriteString(fmt.Sprintf("| Secret Scanning | %s |\n",
		FormatEnabled(matrix.SecuritySettings.SecretScanningEnabled)))
	report.WriteString(fmt.Sprintf("| Code Scanning | %s |\n",
		FormatEnabled(matrix.SecuritySettings.CodeScanningEnabled)))
	report.WriteString("\n")

	// Organization Members (if available)
	if matrix.OrganizationInfo != nil && len(matrix.OrganizationInfo.Members) > 0 {
		report.WriteString("## Organization Members\n\n")
		report.WriteString(fmt.Sprintf("**Organization:** %s\n\n", matrix.OrganizationInfo.Login))
		report.WriteString("| Username | Type | Site Admin |\n")
		report.WriteString("|----------|------|------------|\n")
		for _, member := range matrix.OrganizationInfo.Members {
			report.WriteString(fmt.Sprintf("| %s | %s | %v |\n",
				member.Login, member.Type, member.SiteAdmin))
		}
		report.WriteString("\n")
	}

	return report.String()
}

// generatePermissionMatrix creates a flattened permission matrix view
func (gpt *GitHubPermissionsTool) generatePermissionMatrix(matrix *models.GitHubAccessControlMatrix) string {
	var report strings.Builder

	report.WriteString("# GitHub Permission Matrix\n\n")
	report.WriteString(fmt.Sprintf("**Repository:** %s\n", matrix.Repository.FullName))
	report.WriteString(fmt.Sprintf("**Generated:** %s\n\n", matrix.ExtractedAt.Format("2006-01-02 15:04:05")))

	// User Access Matrix
	report.WriteString("## User Access Matrix\n\n")
	report.WriteString("| User | Direct Access | Team Memberships | Effective Access | Can Deploy To | Can Push To |\n")
	report.WriteString("|------|---------------|------------------|------------------|---------------|-------------|\n")

	// Build user access data
	userAccess := gpt.buildUserAccessMatrix(matrix)
	for _, user := range userAccess {
		teamList := strings.Join(user.TeamMemberships, ", ")
		if teamList == "" {
			teamList = "None"
		}
		deployList := strings.Join(user.CanDeploy, ", ")
		if deployList == "" {
			deployList = "None"
		}
		pushList := strings.Join(user.CanPushTo, ", ")
		if pushList == "" {
			pushList = "All branches"
		}

		report.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
			user.Username,
			user.DirectAccess,
			teamList,
			user.EffectiveAccess,
			deployList,
			pushList))
	}

	report.WriteString("\n")

	// Branch Protection Matrix
	report.WriteString("## Branch Protection Matrix\n\n")
	report.WriteString("| Branch | Protected | Required Reviews | Code Owner Reviews | Status Checks | Enforce Admins |\n")
	report.WriteString("|--------|-----------|------------------|-------------------|---------------|----------------|\n")

	for _, branch := range matrix.Branches {
		protected := "No"
		requiredReviews := "0"
		codeOwnerReviews := "No"
		statusChecks := "None"
		enforceAdmins := "No"

		if branch.Protected && branch.Protection != nil {
			protected = "Yes"
			if branch.Protection.RequiredPullRequestReviews != nil {
				requiredReviews = fmt.Sprintf("%d", branch.Protection.RequiredPullRequestReviews.RequiredApprovingReviewCount)
				if branch.Protection.RequiredPullRequestReviews.RequireCodeOwnerReviews {
					codeOwnerReviews = "Yes"
				}
			}
			if branch.Protection.RequiredStatusChecks != nil && len(branch.Protection.RequiredStatusChecks.Contexts) > 0 {
				statusChecks = strings.Join(branch.Protection.RequiredStatusChecks.Contexts, ", ")
			}
			if branch.Protection.EnforceAdmins.Enabled {
				enforceAdmins = "Yes"
			}
		}

		report.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s | %s |\n",
			branch.Name, protected, requiredReviews, codeOwnerReviews, statusChecks, enforceAdmins))
	}

	report.WriteString("\n")

	return report.String()
}

// generateAccessSummary creates a high-level access summary
func (gpt *GitHubPermissionsTool) generateAccessSummary(matrix *models.GitHubAccessControlMatrix) string {
	var report strings.Builder

	report.WriteString("# GitHub Access Control Summary\n\n")
	report.WriteString(fmt.Sprintf("**Repository:** %s\n", matrix.Repository.FullName))
	report.WriteString(fmt.Sprintf("**Generated:** %s\n\n", matrix.ExtractedAt.Format("2006-01-02 15:04:05")))

	summary := matrix.AccessSummary

	// Access Overview
	report.WriteString("## Access Overview\n\n")
	report.WriteString(fmt.Sprintf("- **Total Users with Access:** %d\n", summary.TotalCollaborators))
	report.WriteString(fmt.Sprintf("- **Total Teams with Access:** %d\n", summary.TotalTeams))
	report.WriteString(fmt.Sprintf("- **Total Team Members:** %d\n", summary.TotalTeamMembers))
	report.WriteString("\n")

	// Permission Breakdown
	report.WriteString("## Permission Level Breakdown\n\n")
	report.WriteString(fmt.Sprintf("- **Admin Users:** %d (%s)\n", len(summary.AdminUsers), strings.Join(summary.AdminUsers, ", ")))
	report.WriteString(fmt.Sprintf("- **Maintain Users:** %d (%s)\n", len(summary.MaintainUsers), strings.Join(summary.MaintainUsers, ", ")))
	report.WriteString(fmt.Sprintf("- **Push/Write Users:** %d (%s)\n", len(summary.PushUsers), strings.Join(summary.PushUsers, ", ")))
	report.WriteString(fmt.Sprintf("- **Triage Users:** %d (%s)\n", len(summary.TriageUsers), strings.Join(summary.TriageUsers, ", ")))
	report.WriteString(fmt.Sprintf("- **Read-Only Users:** %d (%s)\n", len(summary.ReadOnlyUsers), strings.Join(summary.ReadOnlyUsers, ", ")))
	report.WriteString("\n")

	// Team Breakdown
	if summary.TotalTeams > 0 {
		report.WriteString("## Team Permission Breakdown\n\n")
		report.WriteString(fmt.Sprintf("- **Admin Teams:** %d (%s)\n", len(summary.AdminTeams), strings.Join(summary.AdminTeams, ", ")))
		report.WriteString(fmt.Sprintf("- **Write Teams:** %d (%s)\n", len(summary.WriteTeams), strings.Join(summary.WriteTeams, ", ")))
		report.WriteString(fmt.Sprintf("- **Read Teams:** %d (%s)\n", len(summary.ReadTeams), strings.Join(summary.ReadTeams, ", ")))
		report.WriteString("\n")
	}

	// Protection Status
	report.WriteString("## Protection Status\n\n")
	report.WriteString(fmt.Sprintf("- **Protected Branches:** %d (%s)\n",
		len(summary.ProtectedBranches), strings.Join(summary.ProtectedBranches, ", ")))
	report.WriteString(fmt.Sprintf("- **Protected Environments:** %d (%s)\n",
		len(summary.ProtectedEnvironments), strings.Join(summary.ProtectedEnvironments, ", ")))
	report.WriteString("\n")

	// Security Features
	report.WriteString("## Security Features\n\n")
	report.WriteString(fmt.Sprintf("- **Security Score:** %.1f%% (%d/%d features enabled)\n",
		summary.SecurityFeatures.SecurityScore*100,
		summary.SecurityFeatures.EnabledCount,
		summary.SecurityFeatures.TotalFeatures))
	report.WriteString(fmt.Sprintf("- **Enabled Features:** %s\n", strings.Join(summary.SecurityFeatures.EnabledFeatures, ", ")))
	if len(summary.SecurityFeatures.DisabledFeatures) > 0 {
		report.WriteString(fmt.Sprintf("- **Disabled Features:** %s\n", strings.Join(summary.SecurityFeatures.DisabledFeatures, ", ")))
	}
	report.WriteString("\n")

	return report.String()
}

// buildUserAccessMatrix builds a consolidated user access matrix
func (gpt *GitHubPermissionsTool) buildUserAccessMatrix(matrix *models.GitHubAccessControlMatrix) []models.GitHubUserAccess {
	userMap := make(map[string]*models.GitHubUserAccess)

	// Add direct collaborators
	for _, collab := range matrix.Collaborators {
		userMap[collab.Login] = &models.GitHubUserAccess{
			Username:        collab.Login,
			UserID:          collab.ID,
			DirectAccess:    collab.Permissions.Permission,
			EffectiveAccess: collab.Permissions.Permission,
			IsSiteAdmin:     collab.SiteAdmin,
			CanPushTo:       []string{}, // Will be populated based on branch rules
			CanDeploy:       []string{}, // Will be populated based on environment rules
		}
	}

	// Add team memberships
	for _, team := range matrix.Teams {
		for _, member := range team.Members {
			if user, exists := userMap[member.Login]; exists {
				user.TeamMemberships = append(user.TeamMemberships, team.Name)
				// Update effective access if team permission is higher
				if IsHigherPermission(team.Permission, user.EffectiveAccess) {
					user.EffectiveAccess = team.Permission
				}
			} else {
				// User has team access but no direct access
				userMap[member.Login] = &models.GitHubUserAccess{
					Username:        member.Login,
					UserID:          member.ID,
					DirectAccess:    "none",
					TeamMemberships: []string{team.Name},
					EffectiveAccess: team.Permission,
					IsSiteAdmin:     member.SiteAdmin,
					CanPushTo:       []string{},
					CanDeploy:       []string{},
				}
			}
		}
	}

	// Determine branch access based on protection rules
	for _, branch := range matrix.Branches {
		if !branch.Protected {
			// All users with push+ access can push to unprotected branches
			for _, user := range userMap {
				if CanPushBasedOnPermission(user.EffectiveAccess) {
					user.CanPushTo = append(user.CanPushTo, branch.Name)
				}
			}
		} else if branch.Protection != nil && branch.Protection.Restrictions != nil {
			// Only specific users/teams can push to restricted branches
			for _, allowedUser := range branch.Protection.Restrictions.Users {
				if user, exists := userMap[allowedUser.Login]; exists {
					user.CanPushTo = append(user.CanPushTo, branch.Name)
				}
			}
			// Add team members from allowed teams
			for _, allowedTeam := range branch.Protection.Restrictions.Teams {
				for _, team := range matrix.Teams {
					if team.Slug == allowedTeam.Slug {
						for _, member := range team.Members {
							if user, exists := userMap[member.Login]; exists {
								user.CanPushTo = append(user.CanPushTo, branch.Name)
							}
						}
					}
				}
			}
		}
	}

	// Determine deployment environment access
	for _, env := range matrix.Environments {
		for _, rule := range env.ProtectionRules {
			for _, reviewer := range rule.RequiredReviewers {
				if reviewer.Type == "User" {
					if user, exists := userMap[reviewer.Login]; exists {
						user.CanDeploy = append(user.CanDeploy, env.Name)
					}
				}
			}
		}
	}

	// Convert map to slice
	var users []models.GitHubUserAccess
	for _, user := range userMap {
		users = append(users, *user)
	}

	return users
}

// Helper methods

func (gpt *GitHubPermissionsTool) formatEnabled(enabled bool) string {
	if enabled {
		return "✅ Enabled"
	}
	return "❌ Disabled"
}

func (gpt *GitHubPermissionsTool) isHigherPermission(perm1, perm2 string) bool {
	permLevels := map[string]int{
		"admin":    5,
		"maintain": 4,
		"push":     3,
		"write":    3,
		"triage":   2,
		"pull":     1,
		"read":     1,
		"none":     0,
	}

	return permLevels[perm1] > permLevels[perm2]
}

func (gpt *GitHubPermissionsTool) canPushBasedOnPermission(permission string) bool {
	pushPermissions := map[string]bool{
		"admin":    true,
		"maintain": true,
		"push":     true,
		"write":    true,
		"triage":   false,
		"pull":     false,
		"read":     false,
		"none":     false,
	}

	return pushPermissions[permission]
}

func (gpt *GitHubPermissionsTool) calculateRelevance(matrix *models.GitHubAccessControlMatrix) float64 {
	relevance := 0.5 // Base relevance

	// Higher relevance for repositories with more sophisticated access controls
	if len(matrix.Collaborators) > 0 {
		relevance += 0.1
	}
	if len(matrix.Teams) > 0 {
		relevance += 0.1
	}
	if len(matrix.AccessSummary.ProtectedBranches) > 0 {
		relevance += 0.1
	}
	if len(matrix.AccessSummary.ProtectedEnvironments) > 0 {
		relevance += 0.1
	}
	if matrix.AccessSummary.SecurityFeatures.SecurityScore > 0.5 {
		relevance += 0.1
	}

	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}
