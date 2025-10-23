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

package cmd

import (
	"github.com/grctool/grctool/internal/tools"
	"github.com/spf13/cobra"
)

// githubPermissionsCmd handles the github-permissions tool
var githubPermissionsCmd = &cobra.Command{
	Use:   "github-permissions",
	Short: "Extract comprehensive repository access controls and permissions for SOC2 audit evidence",
	Long: `Extract comprehensive repository access controls and permissions including:
- Direct repository collaborators with permission levels
- Team-based access with member lists
- Branch protection rules and restrictions
- Deployment environment protection rules
- Security feature enablement status
- Organization member information (if applicable)
- Consolidated permission matrices for audit review`,
	RunE: runGitHubPermissions,
}

// githubDeploymentAccessCmd handles the github-deployment-access tool
var githubDeploymentAccessCmd = &cobra.Command{
	Use:   "github-deployment-access",
	Short: "Extract deployment environment access controls and protection rules for SOC2 audit evidence",
	Long: `Extract deployment environment access controls including:
- Deployment environment protection rules
- Required reviewers for environment deployments  
- Wait timers and approval workflows
- Branch protection rules affecting deployments
- User and team deployment authorization
- Environment protection coverage analysis`,
	RunE: runGitHubDeploymentAccess,
}

// githubSecurityFeaturesCmd handles the github-security-features tool
var githubSecurityFeaturesCmd = &cobra.Command{
	Use:   "github-security-features",
	Short: "Extract repository security feature configuration for SOC2 audit evidence",
	Long: `Extract repository security feature configuration including:
- Vulnerability alerts and automated security fixes
- Secret scanning and code scanning status
- Dependency graph and security advisories
- Security policy configuration
- Branch protection security requirements
- Security feature enablement matrix`,
	RunE: runGitHubSecurityFeatures,
}

// githubWorkflowAnalyzerCmd handles the github-workflow-analyzer tool
var githubWorkflowAnalyzerCmd = &cobra.Command{
	Use:   "github-workflow-analyzer",
	Short: "Analyze GitHub Actions workflows for CI/CD security evidence and deployment controls",
	Long: `Analyze GitHub Actions workflows for comprehensive SOC2 evidence including:
- Security scanning workflows (CodeQL, Dependabot, secret scanning)
- Deployment approval and environment protection controls
- CI/CD pipeline security configurations
- Workflow compliance with security requirements
- Branch protection rules and required status checks
- Security tool integration and reporting`,
	RunE: runGitHubWorkflowAnalyzer,
}

// githubReviewAnalyzerCmd handles the github-review-analyzer tool
var githubReviewAnalyzerCmd = &cobra.Command{
	Use:   "github-review-analyzer",
	Short: "Analyze GitHub pull request reviews and approval processes for change management evidence",
	Long: `Analyze GitHub pull request reviews for comprehensive SOC2 evidence including:
- Pull request review history and approval metrics
- Code review participation and compliance rates
- Security-related PR identification and review requirements
- Approval timeline analysis and reviewer statistics
- Change management process compliance assessment
- Review quality metrics and improvement recommendations`,
	RunE: runGitHubReviewAnalyzer,
}

func init() {
	// Add GitHub tools to the tool command
	toolCmd.AddCommand(githubPermissionsCmd)
	toolCmd.AddCommand(githubDeploymentAccessCmd)
	toolCmd.AddCommand(githubSecurityFeaturesCmd)
	toolCmd.AddCommand(githubWorkflowAnalyzerCmd)
	toolCmd.AddCommand(githubReviewAnalyzerCmd)

	// GitHub Permissions flags
	githubPermissionsCmd.Flags().String("repository", "", "repository in format 'owner/repo' (e.g., 'octocat/Hello-World')")
	githubPermissionsCmd.Flags().String("output-format", "detailed", "output format (detailed, matrix, summary)")
	githubPermissionsCmd.Flags().Bool("include-org-members", true, "include organization member information if available")
	githubPermissionsCmd.Flags().Bool("use-cache", true, "use cached API results when available")
	githubPermissionsCmd.MarkFlagRequired("repository")

	// GitHub Deployment Access flags
	githubDeploymentAccessCmd.Flags().String("repository", "", "repository in format 'owner/repo' (e.g., 'octocat/Hello-World')")
	githubDeploymentAccessCmd.Flags().String("environment", "", "specific environment name to analyze (optional)")
	githubDeploymentAccessCmd.Flags().String("output-format", "detailed", "output format (detailed, matrix, summary)")
	githubDeploymentAccessCmd.Flags().Bool("include-branch-rules", true, "include branch protection rules that affect deployments")
	githubDeploymentAccessCmd.MarkFlagRequired("repository")

	// GitHub Security Features flags
	githubSecurityFeaturesCmd.Flags().String("repository", "", "repository in format 'owner/repo' (e.g., 'octocat/Hello-World')")
	githubSecurityFeaturesCmd.Flags().String("output-format", "detailed", "output format (detailed, matrix, summary)")
	githubSecurityFeaturesCmd.Flags().Bool("include-policy-analysis", true, "include security policy analysis")
	githubSecurityFeaturesCmd.Flags().Bool("include-compliance-mapping", false, "include SOC2/compliance framework mapping")
	githubSecurityFeaturesCmd.MarkFlagRequired("repository")

	// GitHub Workflow Analyzer flags
	githubWorkflowAnalyzerCmd.Flags().String("analysis-type", "full", "type of workflow analysis (security, deployment, approval, full)")
	githubWorkflowAnalyzerCmd.Flags().Bool("include-content", false, "include full workflow file content in results")
	githubWorkflowAnalyzerCmd.Flags().StringArray("filter-workflows", []string{}, "filter workflows by name patterns (e.g., '*security*', '*deploy*')")
	githubWorkflowAnalyzerCmd.Flags().Bool("check-branch-protection", true, "check branch protection rules and approval requirements")
	githubWorkflowAnalyzerCmd.Flags().Bool("use-cache", true, "use cached results when available")

	// GitHub Review Analyzer flags
	githubReviewAnalyzerCmd.Flags().String("analysis-period", "90d", "time period for analysis (30d, 90d, 180d, 1y)")
	githubReviewAnalyzerCmd.Flags().String("state", "all", "PR state to analyze (open, closed, merged, all)")
	githubReviewAnalyzerCmd.Flags().Bool("include-security-prs", true, "focus on security-related pull requests")
	githubReviewAnalyzerCmd.Flags().Bool("detailed-metrics", true, "include detailed reviewer statistics and patterns")
	githubReviewAnalyzerCmd.Flags().Bool("check-compliance", true, "check compliance with review policies")
	githubReviewAnalyzerCmd.Flags().Int("max-prs", 200, "maximum number of PRs to analyze")
	githubReviewAnalyzerCmd.Flags().Bool("use-cache", true, "use cached results when available")
}

// runGitHubPermissions executes the github-permissions tool
func runGitHubPermissions(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if repository, _ := cmd.Flags().GetString("repository"); repository != "" {
		params["repository"] = repository
	}

	if outputFormat, _ := cmd.Flags().GetString("output-format"); outputFormat != "" {
		params["output_format"] = outputFormat
	}

	if includeOrgMembers, _ := cmd.Flags().GetBool("include-org-members"); cmd.Flags().Changed("include-org-members") {
		params["include_org_members"] = includeOrgMembers
	}

	if useCache, _ := cmd.Flags().GetBool("use-cache"); cmd.Flags().Changed("use-cache") {
		params["use_cache"] = useCache
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"repository": {
			Required:  true,
			Type:      "string",
			Pattern:   `^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`,
			MinLength: 3,
			MaxLength: 100,
		},
		"output_format": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"detailed", "matrix", "summary"},
		},
		"include_org_members": BoolRule,
		"use_cache":           BoolRule,
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "github-permissions", params, validationRules)
}

// runGitHubDeploymentAccess executes the github-deployment-access tool
func runGitHubDeploymentAccess(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if repository, _ := cmd.Flags().GetString("repository"); repository != "" {
		params["repository"] = repository
	}

	if environment, _ := cmd.Flags().GetString("environment"); environment != "" {
		params["environment"] = environment
	}

	if outputFormat, _ := cmd.Flags().GetString("output-format"); outputFormat != "" {
		params["output_format"] = outputFormat
	}

	if includeBranchRules, _ := cmd.Flags().GetBool("include-branch-rules"); cmd.Flags().Changed("include-branch-rules") {
		params["include_branch_rules"] = includeBranchRules
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"repository": {
			Required:  true,
			Type:      "string",
			Pattern:   `^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`,
			MinLength: 3,
			MaxLength: 100,
		},
		"environment": OptionalStringRule,
		"output_format": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"detailed", "matrix", "summary"},
		},
		"include_branch_rules": BoolRule,
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "github-deployment-access", params, validationRules)
}

// runGitHubSecurityFeatures executes the github-security-features tool
func runGitHubSecurityFeatures(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if repository, _ := cmd.Flags().GetString("repository"); repository != "" {
		params["repository"] = repository
	}

	if outputFormat, _ := cmd.Flags().GetString("output-format"); outputFormat != "" {
		params["output_format"] = outputFormat
	}

	if includePolicyAnalysis, _ := cmd.Flags().GetBool("include-policy-analysis"); cmd.Flags().Changed("include-policy-analysis") {
		params["include_policy_analysis"] = includePolicyAnalysis
	}

	if includeComplianceMapping, _ := cmd.Flags().GetBool("include-compliance-mapping"); cmd.Flags().Changed("include-compliance-mapping") {
		params["include_compliance_mapping"] = includeComplianceMapping
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"repository": {
			Required:  true,
			Type:      "string",
			Pattern:   `^[a-zA-Z0-9_.-]+/[a-zA-Z0-9_.-]+$`,
			MinLength: 3,
			MaxLength: 100,
		},
		"output_format": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"detailed", "matrix", "summary"},
		},
		"include_policy_analysis":    BoolRule,
		"include_compliance_mapping": BoolRule,
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "github-security-features", params, validationRules)
}

// runGitHubWorkflowAnalyzer executes the github-workflow-analyzer tool
func runGitHubWorkflowAnalyzer(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if analysisType, _ := cmd.Flags().GetString("analysis-type"); analysisType != "" {
		params["analysis_type"] = analysisType
	}

	if includeContent, _ := cmd.Flags().GetBool("include-content"); cmd.Flags().Changed("include-content") {
		params["include_content"] = includeContent
	}

	if filterWorkflows, _ := cmd.Flags().GetStringArray("filter-workflows"); len(filterWorkflows) > 0 {
		params["filter_workflows"] = filterWorkflows
	}

	if checkBranchProtection, _ := cmd.Flags().GetBool("check-branch-protection"); cmd.Flags().Changed("check-branch-protection") {
		params["check_branch_protection"] = checkBranchProtection
	}

	if useCache, _ := cmd.Flags().GetBool("use-cache"); cmd.Flags().Changed("use-cache") {
		params["use_cache"] = useCache
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"analysis_type": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"security", "deployment", "approval", "full"},
		},
		"include_content":         BoolRule,
		"filter_workflows":        {Required: false, Type: "array"},
		"check_branch_protection": BoolRule,
		"use_cache":               BoolRule,
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "github-workflow-analyzer", params, validationRules)
}

// runGitHubReviewAnalyzer executes the github-review-analyzer tool
func runGitHubReviewAnalyzer(cmd *cobra.Command, args []string) error {
	// Get parameters from flags
	params := make(map[string]interface{})

	if analysisPeriod, _ := cmd.Flags().GetString("analysis-period"); analysisPeriod != "" {
		params["analysis_period"] = analysisPeriod
	}

	if state, _ := cmd.Flags().GetString("state"); state != "" {
		params["state"] = state
	}

	if includeSecurityPRs, _ := cmd.Flags().GetBool("include-security-prs"); cmd.Flags().Changed("include-security-prs") {
		params["include_security_prs"] = includeSecurityPRs
	}

	if detailedMetrics, _ := cmd.Flags().GetBool("detailed-metrics"); cmd.Flags().Changed("detailed-metrics") {
		params["detailed_metrics"] = detailedMetrics
	}

	if checkCompliance, _ := cmd.Flags().GetBool("check-compliance"); cmd.Flags().Changed("check-compliance") {
		params["check_compliance"] = checkCompliance
	}

	if maxPRs, _ := cmd.Flags().GetInt("max-prs"); cmd.Flags().Changed("max-prs") {
		params["max_prs"] = maxPRs
	}

	if useCache, _ := cmd.Flags().GetBool("use-cache"); cmd.Flags().Changed("use-cache") {
		params["use_cache"] = useCache
	}

	// Define validation rules
	validationRules := map[string]tools.ValidationRule{
		"analysis_period": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"30d", "90d", "180d", "1y"},
		},
		"state": {
			Required:      false,
			Type:          "string",
			AllowedValues: []string{"open", "closed", "merged", "all"},
		},
		"include_security_prs": BoolRule,
		"detailed_metrics":     BoolRule,
		"check_compliance":     BoolRule,
		"max_prs": {
			Required: false,
			Type:     "int",
		},
		"use_cache": BoolRule,
	}

	// Execute tool with validation
	return ValidateAndExecuteTool(cmd, "github-review-analyzer", params, validationRules)
}
