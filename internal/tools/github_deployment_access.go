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

// GitHubDeploymentAccessTool extracts deployment environment access controls for SOC2 audit evidence
type GitHubDeploymentAccessTool struct {
	config       *config.GitHubToolConfig
	logger       logger.Logger
	cacheDir     string
	authProvider auth.AuthProvider
	apiClient    *GitHubAPIClient
}

// NewGitHubDeploymentAccessTool creates a new GitHub deployment access extraction tool
func NewGitHubDeploymentAccessTool(cfg *config.Config, log logger.Logger) Tool {
	// Set up cache directory
	cacheDir := filepath.Join(cfg.Storage.DataDir, ".cache", "github_deployments")

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

	return &GitHubDeploymentAccessTool{
		config:       &cfg.Evidence.Tools.GitHub,
		logger:       log,
		cacheDir:     cacheDir,
		authProvider: authProvider,
		apiClient:    NewGitHubAPIClient(cfg, log),
	}
}

// Name returns the tool name
func (gdat *GitHubDeploymentAccessTool) Name() string {
	return "github-deployment-access"
}

// Description returns the tool description
func (gdat *GitHubDeploymentAccessTool) Description() string {
	return "Extract deployment environment access controls and protection rules for SOC2 audit evidence"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (gdat *GitHubDeploymentAccessTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        gdat.Name(),
		Description: gdat.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"repository": map[string]interface{}{
					"type":        "string",
					"description": "Repository in format 'owner/repo' (e.g., 'octocat/Hello-World')",
				},
				"environment": map[string]interface{}{
					"type":        "string",
					"description": "Specific environment name to analyze (optional - if not provided, all environments will be analyzed)",
				},
				"output_format": map[string]interface{}{
					"type":        "string",
					"description": "Output format for deployment access report",
					"enum":        []string{"detailed", "matrix", "summary"},
					"default":     "detailed",
				},
				"include_branch_rules": map[string]interface{}{
					"type":        "boolean",
					"description": "Include branch protection rules that affect deployments",
					"default":     true,
				},
			},
			"required": []string{"repository"},
		},
	}
}

// Execute runs the GitHub deployment access extraction tool
func (gdat *GitHubDeploymentAccessTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	startTime := time.Now()
	correlationID := GenerateCorrelationID()

	gdat.logger.Debug("Executing GitHub deployment access extraction",
		logger.String("correlation_id", correlationID),
		logger.Field{Key: "params", Value: params})

	// Check authentication
	authStatus := gdat.authProvider.GetStatus(ctx)
	if gdat.authProvider.IsAuthRequired() && !authStatus.Authenticated {
		if err := gdat.authProvider.Authenticate(ctx); err != nil {
			return "", nil, fmt.Errorf("GitHub authentication failed: %w", err)
		}
		authStatus = gdat.authProvider.GetStatus(ctx)
	}

	if !authStatus.Authenticated {
		return "", nil, fmt.Errorf("GitHub authentication required for deployment access API")
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

	environment := ""
	if env, ok := params["environment"].(string); ok {
		environment = env
	}

	outputFormat := "detailed"
	if of, ok := params["output_format"].(string); ok {
		outputFormat = of
	}

	includeBranchRules := true
	if ibr, ok := params["include_branch_rules"].(bool); ok {
		includeBranchRules = ibr
	}

	// Extract deployment access information
	deploymentAccess, err := gdat.extractDeploymentAccess(ctx, owner, repo, environment, includeBranchRules)
	if err != nil {
		return "", nil, fmt.Errorf("failed to extract deployment access: %w", err)
	}

	// Generate report based on output format
	var report string
	switch outputFormat {
	case "matrix":
		report = gdat.generateDeploymentMatrix(deploymentAccess)
	case "summary":
		report = gdat.generateDeploymentSummary(deploymentAccess)
	default: // detailed
		report = gdat.generateDetailedDeploymentReport(deploymentAccess)
	}

	duration := time.Since(startTime)
	finalAuthStatus := gdat.authProvider.GetStatus(ctx)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "github-deployment-access",
		Resource:    fmt.Sprintf("GitHub repository deployments: %s", repository),
		Content:     report,
		Relevance:   gdat.calculateDeploymentRelevance(deploymentAccess),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"repository":             repository,
			"environment_filter":     environment,
			"output_format":          outputFormat,
			"include_branch_rules":   includeBranchRules,
			"total_environments":     len(deploymentAccess.Environments),
			"protected_environments": gdat.countProtectedEnvironments(deploymentAccess.Environments),
			"correlation_id":         correlationID,
			"duration_ms":            duration.Milliseconds(),
			"auth_status": map[string]interface{}{
				"authenticated": finalAuthStatus.Authenticated,
				"provider":      finalAuthStatus.Provider,
			},
		},
	}

	return report, source, nil
}

// DeploymentAccessInfo contains comprehensive deployment access information
type DeploymentAccessInfo struct {
	Repository   string                          `json:"repository"`
	Environments []models.GitHubEnvironment      `json:"environments"`
	BranchRules  []models.GitHubBranch           `json:"branch_rules,omitempty"`
	AccessMatrix []models.GitHubDeploymentAccess `json:"access_matrix"`
	ExtractedAt  time.Time                       `json:"extracted_at"`
}

// extractDeploymentAccess extracts comprehensive deployment access information
func (gdat *GitHubDeploymentAccessTool) extractDeploymentAccess(ctx context.Context, owner, repo, environment string, includeBranchRules bool) (*DeploymentAccessInfo, error) {
	info := &DeploymentAccessInfo{
		Repository:  fmt.Sprintf("%s/%s", owner, repo),
		ExtractedAt: time.Now(),
	}

	// Get deployment environments
	environments, err := gdat.apiClient.GetDeploymentEnvironments(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment environments: %w", err)
	}

	// Filter to specific environment if requested
	if environment != "" {
		var filteredEnvs []models.GitHubEnvironment
		for _, env := range environments {
			if env.Name == environment {
				filteredEnvs = append(filteredEnvs, env)
				break
			}
		}
		if len(filteredEnvs) == 0 {
			return nil, fmt.Errorf("environment '%s' not found", environment)
		}
		environments = filteredEnvs
	}

	info.Environments = environments

	// Get branch rules if requested (these can affect deployments)
	if includeBranchRules {
		branches, err := gdat.apiClient.GetRepositoryBranches(ctx, owner, repo)
		if err != nil {
			gdat.logger.Warn("Failed to get branch rules", logger.Field{Key: "error", Value: err})
		} else {
			// Only include protected branches
			var protectedBranches []models.GitHubBranch
			for _, branch := range branches {
				if branch.Protected {
					protectedBranches = append(protectedBranches, branch)
				}
			}
			info.BranchRules = protectedBranches
		}
	}

	// Build access matrix
	info.AccessMatrix = gdat.buildDeploymentAccessMatrix(info.Environments)

	return info, nil
}

// buildDeploymentAccessMatrix creates a deployment access matrix
func (gdat *GitHubDeploymentAccessTool) buildDeploymentAccessMatrix(environments []models.GitHubEnvironment) []models.GitHubDeploymentAccess {
	var accessMatrix []models.GitHubDeploymentAccess

	for _, env := range environments {
		access := models.GitHubDeploymentAccess{
			Environment: env.Name,
		}

		for _, rule := range env.ProtectionRules {
			switch rule.Type {
			case "required_reviewers":
				access.RequiredReviewers = rule.RequiredReviewers
				access.RequiredApprovals = len(rule.RequiredReviewers)
			case "wait_timer":
				access.WaitTimer = rule.WaitTimer
			}
		}

		// Determine if pushes are restricted (this would be from branch protection)
		access.RestrictPushes = len(env.ProtectionRules) > 0

		accessMatrix = append(accessMatrix, access)
	}

	return accessMatrix
}

// generateDetailedDeploymentReport creates a detailed deployment access report
func (gdat *GitHubDeploymentAccessTool) generateDetailedDeploymentReport(info *DeploymentAccessInfo) string {
	var report strings.Builder

	report.WriteString("# GitHub Deployment Access Control Report\n\n")
	report.WriteString(fmt.Sprintf("**Repository:** %s\n", info.Repository))
	report.WriteString(fmt.Sprintf("**Generated:** %s\n\n", info.ExtractedAt.Format("2006-01-02 15:04:05")))

	// Executive Summary
	report.WriteString("## Executive Summary\n\n")
	report.WriteString(fmt.Sprintf("- **Total Environments:** %d\n", len(info.Environments)))
	protectedCount := gdat.countProtectedEnvironments(info.Environments)
	report.WriteString(fmt.Sprintf("- **Protected Environments:** %d\n", protectedCount))
	report.WriteString(fmt.Sprintf("- **Unprotected Environments:** %d\n", len(info.Environments)-protectedCount))
	if len(info.BranchRules) > 0 {
		report.WriteString(fmt.Sprintf("- **Protected Branches:** %d\n", len(info.BranchRules)))
	}
	report.WriteString("\n")

	// Environment Details
	report.WriteString("## Environment Protection Details\n\n")
	if len(info.Environments) == 0 {
		report.WriteString("No deployment environments configured.\n\n")
	} else {
		for _, env := range info.Environments {
			report.WriteString(fmt.Sprintf("### Environment: %s\n", env.Name))

			if len(env.ProtectionRules) == 0 {
				report.WriteString("- **Protection Level:** ⚠️ Unprotected - Anyone with repository access can deploy\n\n")
			} else {
				report.WriteString("- **Protection Level:** ✅ Protected\n")
				report.WriteString("- **Protection Rules:**\n")

				for _, rule := range env.ProtectionRules {
					switch rule.Type {
					case "required_reviewers":
						report.WriteString(fmt.Sprintf("  - **Required Reviewers:** %d\n", len(rule.RequiredReviewers)))
						if len(rule.RequiredReviewers) > 0 {
							report.WriteString("    - **Authorized Reviewers:**\n")
							for _, reviewer := range rule.RequiredReviewers {
								switch reviewer.Type {
								case "User":
									report.WriteString(fmt.Sprintf("      - User: %s", reviewer.Login))
									if reviewer.Name != "" {
										report.WriteString(fmt.Sprintf(" (%s)", reviewer.Name))
									}
									report.WriteString("\n")
								case "Team":
									report.WriteString(fmt.Sprintf("      - Team: %s\n", reviewer.Slug))
								}
							}
						}
					case "wait_timer":
						report.WriteString(fmt.Sprintf("  - **Wait Timer:** %d minutes\n", rule.WaitTimer))
					default:
						report.WriteString(fmt.Sprintf("  - **Rule Type:** %s\n", rule.Type))
					}
				}
				report.WriteString("\n")
			}
		}
	}

	// Branch Protection Impact
	if len(info.BranchRules) > 0 {
		report.WriteString("## Branch Protection Rules Affecting Deployments\n\n")
		for _, branch := range info.BranchRules {
			report.WriteString(fmt.Sprintf("### Branch: %s\n", branch.Name))

			if branch.Protection != nil {
				if branch.Protection.RequiredPullRequestReviews != nil {
					report.WriteString(fmt.Sprintf("- **Required PR Reviews:** %d\n",
						branch.Protection.RequiredPullRequestReviews.RequiredApprovingReviewCount))
					report.WriteString(fmt.Sprintf("- **Code Owner Reviews Required:** %v\n",
						branch.Protection.RequiredPullRequestReviews.RequireCodeOwnerReviews))
				}

				if branch.Protection.RequiredStatusChecks != nil && len(branch.Protection.RequiredStatusChecks.Contexts) > 0 {
					report.WriteString(fmt.Sprintf("- **Required Status Checks:** %s\n",
						strings.Join(branch.Protection.RequiredStatusChecks.Contexts, ", ")))
				}

				if branch.Protection.Restrictions != nil {
					report.WriteString("- **Push Restrictions:** Yes\n")
					if len(branch.Protection.Restrictions.Users) > 0 {
						var userLogins []string
						for _, user := range branch.Protection.Restrictions.Users {
							userLogins = append(userLogins, user.Login)
						}
						report.WriteString(fmt.Sprintf("  - **Allowed Users:** %s\n", strings.Join(userLogins, ", ")))
					}
					if len(branch.Protection.Restrictions.Teams) > 0 {
						var teamNames []string
						for _, team := range branch.Protection.Restrictions.Teams {
							teamNames = append(teamNames, team.Name)
						}
						report.WriteString(fmt.Sprintf("  - **Allowed Teams:** %s\n", strings.Join(teamNames, ", ")))
					}
				} else {
					report.WriteString("- **Push Restrictions:** No (all users with write access can push)\n")
				}
			}
			report.WriteString("\n")
		}
	}

	// Access Matrix Summary
	report.WriteString("## Deployment Access Matrix\n\n")
	report.WriteString("| Environment | Protection | Required Reviewers | Wait Timer | Deploy Access |\n")
	report.WriteString("|-------------|------------|-------------------|------------|---------------|\n")

	for _, access := range info.AccessMatrix {
		protection := "❌ No"
		if len(access.RequiredReviewers) > 0 || access.WaitTimer > 0 {
			protection = "✅ Yes"
		}

		reviewers := "None"
		if access.RequiredApprovals > 0 {
			reviewers = fmt.Sprintf("%d required", access.RequiredApprovals)
		}

		waitTimer := "None"
		if access.WaitTimer > 0 {
			waitTimer = fmt.Sprintf("%d min", access.WaitTimer)
		}

		deployAccess := "All repo contributors"
		if len(access.RequiredReviewers) > 0 {
			var reviewerNames []string
			for _, reviewer := range access.RequiredReviewers {
				if reviewer.Type == "User" {
					reviewerNames = append(reviewerNames, reviewer.Login)
				} else {
					reviewerNames = append(reviewerNames, fmt.Sprintf("@%s", reviewer.Slug))
				}
			}
			deployAccess = strings.Join(reviewerNames, ", ")
		}

		report.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			access.Environment, protection, reviewers, waitTimer, deployAccess))
	}
	report.WriteString("\n")

	// Security Recommendations
	report.WriteString("## Security Recommendations\n\n")
	unprotectedEnvs := len(info.Environments) - gdat.countProtectedEnvironments(info.Environments)
	if unprotectedEnvs > 0 {
		report.WriteString("⚠️ **High Priority:**\n")
		report.WriteString(fmt.Sprintf("- %d environment(s) lack deployment protection rules\n", unprotectedEnvs))
		report.WriteString("- Consider implementing required reviewers for production environments\n")
		report.WriteString("- Consider implementing wait timers for critical deployments\n\n")
	}

	if len(info.Environments) > 0 && gdat.countProtectedEnvironments(info.Environments) == len(info.Environments) {
		report.WriteString("✅ **Good Security Posture:**\n")
		report.WriteString("- All environments have protection rules configured\n")
		report.WriteString("- Deployment access is appropriately restricted\n\n")
	}

	return report.String()
}

// generateDeploymentMatrix creates a matrix view of deployment access
func (gdat *GitHubDeploymentAccessTool) generateDeploymentMatrix(info *DeploymentAccessInfo) string {
	var report strings.Builder

	report.WriteString("# GitHub Deployment Access Matrix\n\n")
	report.WriteString(fmt.Sprintf("**Repository:** %s\n", info.Repository))
	report.WriteString(fmt.Sprintf("**Generated:** %s\n\n", info.ExtractedAt.Format("2006-01-02 15:04:05")))

	// Environment Access Matrix
	report.WriteString("## Environment Access Matrix\n\n")
	report.WriteString("| Environment | Protected | Required Reviewers | Authorized Users/Teams | Wait Timer |\n")
	report.WriteString("|-------------|-----------|-------------------|------------------------|------------|\n")

	for _, access := range info.AccessMatrix {
		protected := "No"
		if len(access.RequiredReviewers) > 0 || access.WaitTimer > 0 {
			protected = "Yes"
		}

		requiredReviewers := "0"
		if access.RequiredApprovals > 0 {
			requiredReviewers = fmt.Sprintf("%d", access.RequiredApprovals)
		}

		authorizedList := "All with repo access"
		if len(access.RequiredReviewers) > 0 {
			var authorized []string
			for _, reviewer := range access.RequiredReviewers {
				if reviewer.Type == "User" {
					authorized = append(authorized, reviewer.Login)
				} else {
					authorized = append(authorized, fmt.Sprintf("@%s", reviewer.Slug))
				}
			}
			authorizedList = strings.Join(authorized, ", ")
		}

		waitTimer := "None"
		if access.WaitTimer > 0 {
			waitTimer = fmt.Sprintf("%d min", access.WaitTimer)
		}

		report.WriteString(fmt.Sprintf("| %s | %s | %s | %s | %s |\n",
			access.Environment, protected, requiredReviewers, authorizedList, waitTimer))
	}
	report.WriteString("\n")

	return report.String()
}

// generateDeploymentSummary creates a summary of deployment access
func (gdat *GitHubDeploymentAccessTool) generateDeploymentSummary(info *DeploymentAccessInfo) string {
	var report strings.Builder

	report.WriteString("# GitHub Deployment Access Summary\n\n")
	report.WriteString(fmt.Sprintf("**Repository:** %s\n", info.Repository))
	report.WriteString(fmt.Sprintf("**Generated:** %s\n\n", info.ExtractedAt.Format("2006-01-02 15:04:05")))

	// Summary Statistics
	report.WriteString("## Summary Statistics\n\n")
	protectedCount := gdat.countProtectedEnvironments(info.Environments)
	unprotectedCount := len(info.Environments) - protectedCount

	report.WriteString(fmt.Sprintf("- **Total Environments:** %d\n", len(info.Environments)))
	report.WriteString(fmt.Sprintf("- **Protected Environments:** %d (%.1f%%)\n",
		protectedCount, float64(protectedCount)/float64(len(info.Environments))*100))
	report.WriteString(fmt.Sprintf("- **Unprotected Environments:** %d (%.1f%%)\n",
		unprotectedCount, float64(unprotectedCount)/float64(len(info.Environments))*100))

	// Protection coverage
	var protectionScore float64
	if len(info.Environments) > 0 {
		protectionScore = float64(protectedCount) / float64(len(info.Environments)) * 100
	}
	report.WriteString(fmt.Sprintf("- **Protection Coverage:** %.1f%%\n\n", protectionScore))

	// Environment List
	if len(info.Environments) > 0 {
		report.WriteString("## Environment Protection Status\n\n")
		for _, env := range info.Environments {
			status := "❌ Unprotected"
			if len(env.ProtectionRules) > 0 {
				status = "✅ Protected"
			}
			report.WriteString(fmt.Sprintf("- **%s:** %s\n", env.Name, status))
		}
		report.WriteString("\n")
	}

	// Security Assessment
	report.WriteString("## Security Assessment\n\n")
	if protectionScore >= 100 {
		report.WriteString("🟢 **Excellent:** All environments are protected\n")
	} else if protectionScore >= 80 {
		report.WriteString("🟡 **Good:** Most environments are protected\n")
	} else if protectionScore >= 50 {
		report.WriteString("🟠 **Moderate:** Some environments lack protection\n")
	} else {
		report.WriteString("🔴 **Poor:** Most environments are unprotected\n")
	}
	report.WriteString("\n")

	return report.String()
}

// Helper methods

func (gdat *GitHubDeploymentAccessTool) countProtectedEnvironments(environments []models.GitHubEnvironment) int {
	count := 0
	for _, env := range environments {
		if len(env.ProtectionRules) > 0 {
			count++
		}
	}
	return count
}

func (gdat *GitHubDeploymentAccessTool) calculateDeploymentRelevance(info *DeploymentAccessInfo) float64 {
	if len(info.Environments) == 0 {
		return 0.3 // Low relevance if no environments
	}

	relevance := 0.5 // Base relevance

	protectedCount := gdat.countProtectedEnvironments(info.Environments)
	if protectedCount > 0 {
		relevance += 0.3 // Higher relevance for protected environments
	}

	if len(info.BranchRules) > 0 {
		relevance += 0.1 // Additional relevance for branch protection
	}

	if len(info.Environments) >= 3 {
		relevance += 0.1 // More environments = more complex deployment setup
	}

	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}
