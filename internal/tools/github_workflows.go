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
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/auth"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/transport"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
	"gopkg.in/yaml.v3"
)

// GitHubWorkflowAnalyzer provides GitHub Actions workflow analysis capabilities
type GitHubWorkflowAnalyzer struct {
	config       *config.GitHubToolConfig
	httpClient   *http.Client
	logger       logger.Logger
	cacheDir     string
	authProvider auth.AuthProvider
}

// NewGitHubWorkflowAnalyzer creates a new GitHub workflow analyzer
func NewGitHubWorkflowAnalyzer(cfg *config.Config, log logger.Logger) Tool {
	// Create HTTP transport with logging
	httpTransport := http.DefaultTransport
	httpTransport = transport.NewLoggingTransport(httpTransport, log.WithComponent("github-workflow-api"))

	// Set up cache directory
	cacheDir := filepath.Join(cfg.Storage.DataDir, "github_cache", "workflows")

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

	return &GitHubWorkflowAnalyzer{
		config: &cfg.Evidence.Tools.GitHub,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: httpTransport,
		},
		logger:       log,
		cacheDir:     cacheDir,
		authProvider: authProvider,
	}
}

// Name returns the tool name
func (gwa *GitHubWorkflowAnalyzer) Name() string {
	return "github-workflow-analyzer"
}

// Description returns the tool description
func (gwa *GitHubWorkflowAnalyzer) Description() string {
	return "Analyze GitHub Actions workflows for CI/CD security evidence, deployment controls, and approval processes"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (gwa *GitHubWorkflowAnalyzer) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        gwa.Name(),
		Description: gwa.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"analysis_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of workflow analysis: security, deployment, approval, full",
					"enum":        []string{"security", "deployment", "approval", "full"},
					"default":     "full",
				},
				"include_content": map[string]interface{}{
					"type":        "boolean",
					"description": "Include full workflow file content in results",
					"default":     false,
				},
				"filter_workflows": map[string]interface{}{
					"type":        "array",
					"description": "Filter workflows by name patterns (e.g., ['*security*', '*deploy*'])",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
				"check_branch_protection": map[string]interface{}{
					"type":        "boolean",
					"description": "Check branch protection rules and approval requirements",
					"default":     true,
				},
				"use_cache": map[string]interface{}{
					"type":        "boolean",
					"description": "Use cached results when available",
					"default":     true,
				},
			},
			"required": []string{},
		},
	}
}

// Execute runs the GitHub workflow analysis
func (gwa *GitHubWorkflowAnalyzer) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	startTime := time.Now()
	correlationID := GenerateCorrelationID()

	gwa.logger.Debug("Executing GitHub workflow analyzer",
		logger.String("correlation_id", correlationID),
		logger.Field{Key: "params", Value: params})

	// Check authentication
	authStatus := gwa.authProvider.GetStatus(ctx)
	if gwa.authProvider.IsAuthRequired() && !authStatus.Authenticated {
		if err := gwa.authProvider.Authenticate(ctx); err != nil {
			gwa.logger.Warn("GitHub authentication failed, will use cached data only",
				logger.Field{Key: "error", Value: err})
		}
		// Note: authStatus will be refreshed later when needed
	}

	// Extract parameters
	analysisType := "full"
	if at, ok := params["analysis_type"].(string); ok {
		analysisType = at
	}

	includeContent := false
	if ic, ok := params["include_content"].(bool); ok {
		includeContent = ic
	}

	var filterWorkflows []string
	if fw, ok := params["filter_workflows"].([]interface{}); ok {
		for _, filter := range fw {
			if str, ok := filter.(string); ok {
				filterWorkflows = append(filterWorkflows, str)
			}
		}
	}

	checkBranchProtection := true
	if cbp, ok := params["check_branch_protection"].(bool); ok {
		checkBranchProtection = cbp
	}

	useCache := true
	if uc, ok := params["use_cache"].(bool); ok {
		useCache = uc
	}

	// Perform analysis
	analysis, err := gwa.analyzeWorkflows(ctx, analysisType, includeContent, filterWorkflows, checkBranchProtection, useCache)
	if err != nil {
		return "", nil, fmt.Errorf("failed to analyze workflows: %w", err)
	}

	// Generate report
	report := gwa.generateWorkflowReport(analysis, analysisType)

	// Get final auth status
	finalAuthStatus := gwa.authProvider.GetStatus(ctx)
	duration := time.Since(startTime)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "github-workflow-analyzer",
		Resource:    fmt.Sprintf("GitHub workflows: %s", gwa.config.Repository),
		Content:     report,
		Relevance:   gwa.calculateWorkflowRelevance(analysis),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"repository":       gwa.config.Repository,
			"analysis_type":    analysisType,
			"workflow_count":   len(analysis.WorkflowFiles),
			"security_scans":   len(analysis.SecurityScans),
			"approval_rules":   len(analysis.ApprovalRules),
			"compliance_score": analysis.ComplianceScore,
			"correlation_id":   correlationID,
			"duration_ms":      duration.Milliseconds(),
			"auth_status": map[string]interface{}{
				"authenticated": finalAuthStatus.Authenticated,
				"provider":      finalAuthStatus.Provider,
				"cache_used":    finalAuthStatus.CacheUsed,
			},
		},
	}

	return report, source, nil
}

// analyzeWorkflows performs the comprehensive workflow analysis
func (gwa *GitHubWorkflowAnalyzer) analyzeWorkflows(ctx context.Context, analysisType string, includeContent bool, filterWorkflows []string, checkBranchProtection bool, useCache bool) (*models.GitHubWorkflowAnalysis, error) {
	analysis := &models.GitHubWorkflowAnalysis{
		Repository:   gwa.config.Repository,
		AnalysisDate: time.Now(),
	}

	// Get workflow files
	workflows, err := gwa.getWorkflowFiles(ctx, filterWorkflows, includeContent, useCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow files: %w", err)
	}
	analysis.WorkflowFiles = workflows

	// Analyze security scanning configuration
	if analysisType == "security" || analysisType == "full" {
		securityScans, err := gwa.analyzeSecurityScanning(ctx, useCache)
		if err != nil {
			gwa.logger.Warn("Failed to analyze security scanning", logger.Field{Key: "error", Value: err})
		} else {
			analysis.SecurityScans = securityScans
		}
	}

	// Analyze approval rules and branch protection
	if (analysisType == "approval" || analysisType == "full") && checkBranchProtection {
		approvalRules, err := gwa.analyzeBranchProtection(ctx, useCache)
		if err != nil {
			gwa.logger.Warn("Failed to analyze branch protection", logger.Field{Key: "error", Value: err})
		} else {
			analysis.ApprovalRules = approvalRules
		}
	}

	// Generate statistics and compliance score
	analysis.Statistics = gwa.generateWorkflowStatistics(workflows)
	analysis.ComplianceScore = gwa.calculateComplianceScore(analysis)

	return analysis, nil
}

// getWorkflowFiles retrieves and parses GitHub Actions workflow files
func (gwa *GitHubWorkflowAnalyzer) getWorkflowFiles(ctx context.Context, filterWorkflows []string, includeContent bool, useCache bool) ([]models.GitHubWorkflowFile, error) {
	// Get workflow files from GitHub API
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/.github/workflows", gwa.config.Repository)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if gwa.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+gwa.config.APIToken)
	}

	resp, err := gwa.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		// Repository doesn't have workflows directory
		return []models.GitHubWorkflowFile{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var files []struct {
		Name        string `json:"name"`
		Path        string `json:"path"`
		SHA         string `json:"sha"`
		Type        string `json:"type"`
		HTMLURL     string `json:"html_url"`
		DownloadURL string `json:"download_url"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&files); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var workflows []models.GitHubWorkflowFile
	for _, file := range files {
		// Only process YAML workflow files
		if file.Type != "file" || (!strings.HasSuffix(file.Name, ".yml") && !strings.HasSuffix(file.Name, ".yaml")) {
			continue
		}

		// Apply filters if specified
		if len(filterWorkflows) > 0 && !gwa.matchesFilter(file.Name, filterWorkflows) {
			continue
		}

		workflow, err := gwa.parseWorkflowFile(ctx, file.Name, file.Path, file.HTMLURL, file.DownloadURL, includeContent)
		if err != nil {
			gwa.logger.Warn("Failed to parse workflow file",
				logger.String("file", file.Name),
				logger.Field{Key: "error", Value: err})
			continue
		}

		workflows = append(workflows, *workflow)
	}

	return workflows, nil
}

// parseWorkflowFile downloads and parses a workflow file
func (gwa *GitHubWorkflowAnalyzer) parseWorkflowFile(ctx context.Context, name, path, htmlURL, downloadURL string, includeContent bool) (*models.GitHubWorkflowFile, error) {
	// Download workflow content
	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create download request: %w", err)
	}

	resp, err := gwa.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to download workflow: %w", err)
	}
	defer resp.Body.Close()

	content, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read workflow content: %w", err)
	}

	// Parse YAML content
	var workflowYAML map[string]interface{}
	if err := yaml.Unmarshal(content, &workflowYAML); err != nil {
		return nil, fmt.Errorf("failed to parse workflow YAML: %w", err)
	}

	workflow := &models.GitHubWorkflowFile{
		Name:         name,
		Path:         path,
		HTMLURL:      htmlURL,
		LastModified: time.Now(), // We'd need to get this from file info in a real implementation
		Relevance:    1.0,        // Calculate based on content analysis
	}

	if includeContent {
		workflow.Content = string(content)
	}

	// Parse triggers
	if on, ok := workflowYAML["on"]; ok {
		workflow.Triggers = gwa.parseTriggers(on)
	}

	// Parse jobs
	if jobs, ok := workflowYAML["jobs"].(map[string]interface{}); ok {
		workflow.Jobs = gwa.parseJobs(jobs)
	}

	// Analyze security and approval steps
	workflow.SecuritySteps = gwa.extractSecuritySteps(workflow.Jobs)
	workflow.ApprovalSteps = gwa.extractApprovalSteps(workflow.Jobs)

	// Analyze compliance rules
	workflow.ComplianceRules = gwa.analyzeWorkflowCompliance(workflow)

	return workflow, nil
}

// parseTriggers extracts workflow trigger information
func (gwa *GitHubWorkflowAnalyzer) parseTriggers(onData interface{}) []models.GitHubWorkflowTrigger {
	var triggers []models.GitHubWorkflowTrigger

	switch on := onData.(type) {
	case string:
		// Simple string trigger
		triggers = append(triggers, models.GitHubWorkflowTrigger{Event: on})
	case []interface{}:
		// Array of trigger events
		for _, event := range on {
			if eventStr, ok := event.(string); ok {
				triggers = append(triggers, models.GitHubWorkflowTrigger{Event: eventStr})
			}
		}
	case map[string]interface{}:
		// Complex trigger configuration
		for event, config := range on {
			trigger := models.GitHubWorkflowTrigger{Event: event}

			if configMap, ok := config.(map[string]interface{}); ok {
				if branches, ok := configMap["branches"].([]interface{}); ok {
					for _, branch := range branches {
						if branchStr, ok := branch.(string); ok {
							trigger.Branches = append(trigger.Branches, branchStr)
						}
					}
				}
				if paths, ok := configMap["paths"].([]interface{}); ok {
					for _, path := range paths {
						if pathStr, ok := path.(string); ok {
							trigger.Paths = append(trigger.Paths, pathStr)
						}
					}
				}
				if schedule, ok := configMap["schedule"].([]interface{}); ok && len(schedule) > 0 {
					if scheduleMap, ok := schedule[0].(map[string]interface{}); ok {
						if cron, ok := scheduleMap["cron"].(string); ok {
							trigger.Schedule = cron
						}
					}
				}
				trigger.Filters = configMap
			}

			triggers = append(triggers, trigger)
		}
	}

	return triggers
}

// parseJobs extracts job information from workflow
func (gwa *GitHubWorkflowAnalyzer) parseJobs(jobsData map[string]interface{}) []models.GitHubWorkflowJob {
	var jobs []models.GitHubWorkflowJob

	for jobID, jobData := range jobsData {
		if jobMap, ok := jobData.(map[string]interface{}); ok {
			job := models.GitHubWorkflowJob{
				ID: jobID,
			}

			if name, ok := jobMap["name"].(string); ok {
				job.Name = name
			}
			if runsOn, ok := jobMap["runs-on"].(string); ok {
				job.RunsOn = runsOn
			}
			if environment, ok := jobMap["environment"].(string); ok {
				job.Environment = environment
			}

			// Parse steps
			if steps, ok := jobMap["steps"].([]interface{}); ok {
				job.Steps = gwa.parseSteps(steps)
			}

			// Parse other job properties
			if needs, ok := jobMap["needs"].([]interface{}); ok {
				for _, need := range needs {
					if needStr, ok := need.(string); ok {
						job.Needs = append(job.Needs, needStr)
					}
				}
			}

			if ifCondition, ok := jobMap["if"].(string); ok {
				job.If = ifCondition
			}

			if permissions, ok := jobMap["permissions"].(map[string]interface{}); ok {
				job.Permissions = make(map[string]string)
				for key, value := range permissions {
					if valueStr, ok := value.(string); ok {
						job.Permissions[key] = valueStr
					}
				}
			}

			jobs = append(jobs, job)
		}
	}

	return jobs
}

// parseSteps extracts step information from job
func (gwa *GitHubWorkflowAnalyzer) parseSteps(stepsData []interface{}) []models.GitHubWorkflowStep {
	var steps []models.GitHubWorkflowStep

	for _, stepData := range stepsData {
		if stepMap, ok := stepData.(map[string]interface{}); ok {
			step := models.GitHubWorkflowStep{}

			if id, ok := stepMap["id"].(string); ok {
				step.ID = id
			}
			if name, ok := stepMap["name"].(string); ok {
				step.Name = name
			}
			if uses, ok := stepMap["uses"].(string); ok {
				step.Uses = uses
			}
			if run, ok := stepMap["run"].(string); ok {
				step.Run = run
			}

			// Parse 'with' parameters
			if withData, ok := stepMap["with"].(map[string]interface{}); ok {
				step.With = withData
			}

			// Parse environment variables
			if envData, ok := stepMap["env"].(map[string]interface{}); ok {
				step.Env = make(map[string]string)
				for key, value := range envData {
					if valueStr, ok := value.(string); ok {
						step.Env[key] = valueStr
					}
				}
			}

			steps = append(steps, step)
		}
	}

	return steps
}

// extractSecuritySteps identifies security-related steps in workflows
func (gwa *GitHubWorkflowAnalyzer) extractSecuritySteps(jobs []models.GitHubWorkflowJob) []models.GitHubWorkflowSecurityStep {
	var securitySteps []models.GitHubWorkflowSecurityStep

	securityActions := map[string]string{
		"github/codeql-action/init":                 "code_scanning",
		"github/codeql-action/analyze":              "code_scanning",
		"securecodewarrior/github-action-add-sarif": "code_scanning",
		"github/super-linter":                       "code_scanning",
		"aquasecurity/trivy-action":                 "dependency_check",
		"actions/dependency-review-action":          "dependency_check",
		"snyk/actions":                              "dependency_check",
		"ossf/scorecard-action":                     "supply_chain_security",
		"sigstore/cosign-installer":                 "artifact_signing",
		"slsa-framework/slsa-github-generator":      "supply_chain_security",
	}

	for _, job := range jobs {
		for _, step := range job.Steps {
			if step.Uses != "" {
				for actionPattern, purpose := range securityActions {
					if strings.Contains(step.Uses, actionPattern) {
						securityStep := models.GitHubWorkflowSecurityStep{
							StepName:        step.Name,
							Action:          step.Uses,
							Purpose:         purpose,
							Tool:            gwa.extractToolName(step.Uses),
							Configuration:   step.With,
							FailureHandling: gwa.determineFailureHandling(step),
							Frequency:       gwa.determineStepFrequency(job),
						}
						securitySteps = append(securitySteps, securityStep)
						break
					}
				}
			}

			// Check for security-related run commands
			if step.Run != "" && gwa.isSecurityCommand(step.Run) {
				securityStep := models.GitHubWorkflowSecurityStep{
					StepName:        step.Name,
					Action:          "run",
					Purpose:         gwa.determineCommandPurpose(step.Run),
					Tool:            gwa.extractToolFromCommand(step.Run),
					FailureHandling: gwa.determineFailureHandling(step),
					Frequency:       gwa.determineStepFrequency(job),
				}
				securitySteps = append(securitySteps, securityStep)
			}
		}
	}

	return securitySteps
}

// extractApprovalSteps identifies approval and deployment control steps
func (gwa *GitHubWorkflowAnalyzer) extractApprovalSteps(jobs []models.GitHubWorkflowJob) []models.GitHubWorkflowApprovalStep {
	var approvalSteps []models.GitHubWorkflowApprovalStep

	for _, job := range jobs {
		if job.Environment != "" {
			// Environment-based approval
			approvalStep := models.GitHubWorkflowApprovalStep{
				StepName:    fmt.Sprintf("Environment approval: %s", job.Environment),
				Environment: job.Environment,
			}
			approvalSteps = append(approvalSteps, approvalStep)
		}

		// Check for manual approval actions
		for _, step := range job.Steps {
			if strings.Contains(step.Uses, "manual") || strings.Contains(step.Uses, "approval") {
				approvalStep := models.GitHubWorkflowApprovalStep{
					StepName:    step.Name,
					Environment: job.Environment,
				}
				approvalSteps = append(approvalSteps, approvalStep)
			}
		}
	}

	return approvalSteps
}

// analyzeSecurityScanning checks repository security scanning configuration
func (gwa *GitHubWorkflowAnalyzer) analyzeSecurityScanning(ctx context.Context, useCache bool) ([]models.GitHubSecurityScan, error) {
	var scans []models.GitHubSecurityScan

	// Check CodeQL configuration
	codeqlScan, err := gwa.checkCodeQLConfiguration(ctx)
	if err == nil {
		scans = append(scans, *codeqlScan)
	}

	// Check Dependabot configuration
	dependabotScan, err := gwa.checkDependabotConfiguration(ctx)
	if err == nil {
		scans = append(scans, *dependabotScan)
	}

	// Check secret scanning (requires admin access)
	secretScan, err := gwa.checkSecretScanningConfiguration(ctx)
	if err == nil {
		scans = append(scans, *secretScan)
	}

	return scans, nil
}

// checkCodeQLConfiguration checks if CodeQL is enabled and configured
func (gwa *GitHubWorkflowAnalyzer) checkCodeQLConfiguration(ctx context.Context) (*models.GitHubSecurityScan, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/code-scanning/alerts", gwa.config.Repository)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if gwa.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+gwa.config.APIToken)
	}

	resp, err := gwa.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	scan := &models.GitHubSecurityScan{
		Type:          "codeql",
		Enabled:       resp.StatusCode == http.StatusOK,
		Configuration: make(map[string]interface{}),
		Status:        "unknown",
	}

	switch resp.StatusCode {
	case http.StatusOK:
		scan.Status = "active"
		// Could parse actual alerts here for more detail
	case http.StatusNotFound:
		scan.Status = "disabled"
	}

	return scan, nil
}

// checkDependabotConfiguration checks Dependabot configuration
func (gwa *GitHubWorkflowAnalyzer) checkDependabotConfiguration(ctx context.Context) (*models.GitHubSecurityScan, error) {
	// Check for dependabot.yml file
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/.github/dependabot.yml", gwa.config.Repository)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if gwa.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+gwa.config.APIToken)
	}

	resp, err := gwa.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	scan := &models.GitHubSecurityScan{
		Type:          "dependabot",
		Enabled:       resp.StatusCode == http.StatusOK,
		Configuration: make(map[string]interface{}),
		Status:        "unknown",
	}

	if resp.StatusCode == http.StatusOK {
		scan.Status = "active"
		// Could parse dependabot.yml content for more detail
	} else {
		scan.Status = "disabled"
	}

	return scan, nil
}

// checkSecretScanningConfiguration checks secret scanning status
func (gwa *GitHubWorkflowAnalyzer) checkSecretScanningConfiguration(ctx context.Context) (*models.GitHubSecurityScan, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/secret-scanning/alerts", gwa.config.Repository)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if gwa.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+gwa.config.APIToken)
	}

	resp, err := gwa.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	scan := &models.GitHubSecurityScan{
		Type:          "secret_scanning",
		Enabled:       resp.StatusCode == http.StatusOK,
		Configuration: make(map[string]interface{}),
		Status:        "unknown",
	}

	switch resp.StatusCode {
	case http.StatusOK:
		scan.Status = "active"
	case http.StatusNotFound:
		scan.Status = "disabled"
	}

	return scan, nil
}

// analyzeBranchProtection checks branch protection rules
func (gwa *GitHubWorkflowAnalyzer) analyzeBranchProtection(ctx context.Context, useCache bool) ([]models.GitHubApprovalRule, error) {
	// Get protected branches
	url := fmt.Sprintf("https://api.github.com/repos/%s/branches", gwa.config.Repository)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if gwa.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+gwa.config.APIToken)
	}

	resp, err := gwa.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var branches []struct {
		Name      string `json:"name"`
		Protected bool   `json:"protected"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&branches); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var approvalRules []models.GitHubApprovalRule
	for _, branch := range branches {
		if branch.Protected {
			rule, err := gwa.getBranchProtectionDetails(ctx, branch.Name)
			if err != nil {
				gwa.logger.Warn("Failed to get branch protection details",
					logger.String("branch", branch.Name),
					logger.Field{Key: "error", Value: err})
				continue
			}
			approvalRules = append(approvalRules, *rule)
		}
	}

	return approvalRules, nil
}

// getBranchProtectionDetails gets detailed protection rules for a branch
func (gwa *GitHubWorkflowAnalyzer) getBranchProtectionDetails(ctx context.Context, branch string) (*models.GitHubApprovalRule, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/branches/%s/protection", gwa.config.Repository, url.PathEscape(branch))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if gwa.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+gwa.config.APIToken)
	}

	resp, err := gwa.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &models.GitHubApprovalRule{Branch: branch}, nil // Return basic rule if details unavailable
	}

	var protection struct {
		RequiredStatusChecks struct {
			Strict   bool     `json:"strict"`
			Contexts []string `json:"contexts"`
		} `json:"required_status_checks"`
		RequiredPullRequestReviews struct {
			RequiredApprovingReviewCount int  `json:"required_approving_review_count"`
			DismissStaleReviews          bool `json:"dismiss_stale_reviews"`
			RequireCodeOwnerReviews      bool `json:"require_code_owner_reviews"`
		} `json:"required_pull_request_reviews"`
		EnforceAdmins        bool `json:"enforce_admins"`
		RequireSignedCommits bool `json:"required_signatures"`
		RequireLinearHistory bool `json:"required_linear_history"`
		AllowForcePushes     bool `json:"allow_force_pushes"`
		AllowDeletions       bool `json:"allow_deletions"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&protection); err != nil {
		return nil, fmt.Errorf("failed to parse protection response: %w", err)
	}

	rule := &models.GitHubApprovalRule{
		Branch:                       branch,
		RequiredReviews:              protection.RequiredPullRequestReviews.RequiredApprovingReviewCount,
		RequiredApprovingReviewCount: protection.RequiredPullRequestReviews.RequiredApprovingReviewCount,
		RequireCodeOwnerReviews:      protection.RequiredPullRequestReviews.RequireCodeOwnerReviews,
		DismissStaleReviews:          protection.RequiredPullRequestReviews.DismissStaleReviews,
		RequiredStatusChecks:         protection.RequiredStatusChecks.Contexts,
		RequireUpToDateBranches:      protection.RequiredStatusChecks.Strict,
		RequireSignedCommits:         protection.RequireSignedCommits,
		RequireLinearHistory:         protection.RequireLinearHistory,
		AllowForcePushes:             protection.AllowForcePushes,
		AllowDeletions:               protection.AllowDeletions,
	}

	return rule, nil
}

// Helper methods

func (gwa *GitHubWorkflowAnalyzer) matchesFilter(filename string, filters []string) bool {
	for _, filter := range filters {
		matched, _ := filepath.Match(filter, filename)
		if matched {
			return true
		}
	}
	return false
}

func (gwa *GitHubWorkflowAnalyzer) extractToolName(actionName string) string {
	parts := strings.Split(actionName, "/")
	if len(parts) >= 2 {
		return parts[len(parts)-2] + "/" + parts[len(parts)-1]
	}
	return actionName
}

func (gwa *GitHubWorkflowAnalyzer) determineFailureHandling(step models.GitHubWorkflowStep) string {
	if step.ContinueOnError {
		return "ignore"
	}
	return "fail"
}

func (gwa *GitHubWorkflowAnalyzer) determineStepFrequency(job models.GitHubWorkflowJob) string {
	// This would need to analyze the workflow triggers to determine frequency
	return "on_trigger"
}

func (gwa *GitHubWorkflowAnalyzer) isSecurityCommand(command string) bool {
	securityKeywords := []string{"scan", "security", "vulner", "audit", "test", "lint", "bandit", "semgrep"}
	commandLower := strings.ToLower(command)
	for _, keyword := range securityKeywords {
		if strings.Contains(commandLower, keyword) {
			return true
		}
	}
	return false
}

func (gwa *GitHubWorkflowAnalyzer) determineCommandPurpose(command string) string {
	commandLower := strings.ToLower(command)
	if strings.Contains(commandLower, "test") {
		return "testing"
	}
	if strings.Contains(commandLower, "scan") || strings.Contains(commandLower, "security") {
		return "security_scanning"
	}
	if strings.Contains(commandLower, "lint") {
		return "code_quality"
	}
	return "custom_security"
}

func (gwa *GitHubWorkflowAnalyzer) extractToolFromCommand(command string) string {
	// Extract tool name from command (simplified)
	words := strings.Fields(command)
	if len(words) > 0 {
		return words[0]
	}
	return "custom"
}

func (gwa *GitHubWorkflowAnalyzer) analyzeWorkflowCompliance(workflow *models.GitHubWorkflowFile) []models.GitHubComplianceRule {
	var rules []models.GitHubComplianceRule

	// Check for security scanning
	hasSecurityScanning := len(workflow.SecuritySteps) > 0
	rules = append(rules, models.GitHubComplianceRule{
		RuleID:      "SEC-001",
		Name:        "Security Scanning Required",
		Description: "Workflows should include security scanning steps",
		Status:      gwa.boolToStatus(hasSecurityScanning),
		Evidence:    fmt.Sprintf("Found %d security steps", len(workflow.SecuritySteps)),
		Framework:   "SOC2",
		ControlID:   "CC6.1",
	})

	// Check for approval steps
	hasApprovalSteps := len(workflow.ApprovalSteps) > 0
	rules = append(rules, models.GitHubComplianceRule{
		RuleID:      "APP-001",
		Name:        "Approval Controls Required",
		Description: "Deployment workflows should require approval",
		Status:      gwa.boolToStatus(hasApprovalSteps),
		Evidence:    fmt.Sprintf("Found %d approval steps", len(workflow.ApprovalSteps)),
		Framework:   "SOC2",
		ControlID:   "CC6.2",
	})

	return rules
}

func (gwa *GitHubWorkflowAnalyzer) boolToStatus(b bool) string {
	if b {
		return "pass"
	}
	return "fail"
}

func (gwa *GitHubWorkflowAnalyzer) generateWorkflowStatistics(workflows []models.GitHubWorkflowFile) models.GitHubWorkflowStatistics {
	stats := models.GitHubWorkflowStatistics{
		TotalWorkflows:      len(workflows),
		ToolsUsed:           make(map[string]int),
		TriggerTypes:        make(map[string]int),
		ComplianceBreakdown: make(map[string]models.GitHubComplianceBreakdown),
	}

	totalJobs := 0
	for _, workflow := range workflows {
		totalJobs += len(workflow.Jobs)

		// Count security workflows
		if len(workflow.SecuritySteps) > 0 {
			stats.SecurityWorkflows++
		}

		// Count tools used
		for _, secStep := range workflow.SecuritySteps {
			stats.ToolsUsed[secStep.Tool]++
		}

		// Count trigger types
		for _, trigger := range workflow.Triggers {
			stats.TriggerTypes[trigger.Event]++
		}

		// Count security and approval steps
		stats.SecurityStepsCount += len(workflow.SecuritySteps)
		stats.ApprovalStepsCount += len(workflow.ApprovalSteps)
	}

	if len(workflows) > 0 {
		stats.AverageJobsPerFlow = float64(totalJobs) / float64(len(workflows))
	}

	return stats
}

func (gwa *GitHubWorkflowAnalyzer) calculateComplianceScore(analysis *models.GitHubWorkflowAnalysis) float64 {
	totalRules := 0
	passedRules := 0

	for _, workflow := range analysis.WorkflowFiles {
		for _, rule := range workflow.ComplianceRules {
			totalRules++
			if rule.Status == "pass" {
				passedRules++
			}
		}
	}

	if totalRules == 0 {
		return 0.0
	}

	return float64(passedRules) / float64(totalRules)
}

func (gwa *GitHubWorkflowAnalyzer) calculateWorkflowRelevance(analysis *models.GitHubWorkflowAnalysis) float64 {
	if len(analysis.WorkflowFiles) == 0 {
		return 0.0
	}

	// Base relevance on compliance score and security features
	relevance := analysis.ComplianceScore * 0.6

	// Add bonus for security scanning
	if len(analysis.SecurityScans) > 0 {
		relevance += 0.2
	}

	// Add bonus for approval rules
	if len(analysis.ApprovalRules) > 0 {
		relevance += 0.2
	}

	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}

func (gwa *GitHubWorkflowAnalyzer) generateWorkflowReport(analysis *models.GitHubWorkflowAnalysis, analysisType string) string {
	var report strings.Builder

	report.WriteString("# GitHub Actions Workflow Analysis\n\n")
	report.WriteString(fmt.Sprintf("**Repository**: %s\n", analysis.Repository))
	report.WriteString(fmt.Sprintf("**Analysis Date**: %s\n", analysis.AnalysisDate.Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("**Analysis Type**: %s\n", analysisType))
	report.WriteString(fmt.Sprintf("**Compliance Score**: %.2f\n\n", analysis.ComplianceScore))

	// Workflow Summary
	report.WriteString("## Workflow Summary\n\n")
	report.WriteString(fmt.Sprintf("- **Total Workflows**: %d\n", analysis.Statistics.TotalWorkflows))
	report.WriteString(fmt.Sprintf("- **Security Workflows**: %d\n", analysis.Statistics.SecurityWorkflows))
	report.WriteString(fmt.Sprintf("- **Security Steps**: %d\n", analysis.Statistics.SecurityStepsCount))
	report.WriteString(fmt.Sprintf("- **Approval Steps**: %d\n", analysis.Statistics.ApprovalStepsCount))
	report.WriteString(fmt.Sprintf("- **Average Jobs per Workflow**: %.1f\n\n", analysis.Statistics.AverageJobsPerFlow))

	// Security Scanning
	if len(analysis.SecurityScans) > 0 {
		report.WriteString("## Security Scanning Configuration\n\n")
		for _, scan := range analysis.SecurityScans {
			report.WriteString(fmt.Sprintf("### %s\n", cases.Title(language.English).String(strings.ReplaceAll(scan.Type, "_", " "))))
			report.WriteString(fmt.Sprintf("- **Enabled**: %t\n", scan.Enabled))
			report.WriteString(fmt.Sprintf("- **Status**: %s\n", scan.Status))
			if len(scan.Languages) > 0 {
				report.WriteString(fmt.Sprintf("- **Languages**: %s\n", strings.Join(scan.Languages, ", ")))
			}
			report.WriteString("\n")
		}
	}

	// Branch Protection Rules
	if len(analysis.ApprovalRules) > 0 {
		report.WriteString("## Branch Protection Rules\n\n")
		for _, rule := range analysis.ApprovalRules {
			report.WriteString(fmt.Sprintf("### Branch: %s\n", rule.Branch))
			report.WriteString(fmt.Sprintf("- **Required Reviews**: %d\n", rule.RequiredReviews))
			report.WriteString(fmt.Sprintf("- **Code Owner Reviews Required**: %t\n", rule.RequireCodeOwnerReviews))
			report.WriteString(fmt.Sprintf("- **Dismiss Stale Reviews**: %t\n", rule.DismissStaleReviews))
			report.WriteString(fmt.Sprintf("- **Require Signed Commits**: %t\n", rule.RequireSignedCommits))
			if len(rule.RequiredStatusChecks) > 0 {
				report.WriteString(fmt.Sprintf("- **Required Status Checks**: %s\n", strings.Join(rule.RequiredStatusChecks, ", ")))
			}
			report.WriteString("\n")
		}
	}

	// Workflow Details
	if len(analysis.WorkflowFiles) > 0 {
		report.WriteString("## Workflow Files\n\n")
		for _, workflow := range analysis.WorkflowFiles {
			report.WriteString(fmt.Sprintf("### %s\n", workflow.Name))
			report.WriteString(fmt.Sprintf("- **Path**: %s\n", workflow.Path))
			report.WriteString(fmt.Sprintf("- **Jobs**: %d\n", len(workflow.Jobs)))
			report.WriteString(fmt.Sprintf("- **Security Steps**: %d\n", len(workflow.SecuritySteps)))
			report.WriteString(fmt.Sprintf("- **Approval Steps**: %d\n", len(workflow.ApprovalSteps)))

			if len(workflow.Triggers) > 0 {
				var triggerEvents []string
				for _, trigger := range workflow.Triggers {
					triggerEvents = append(triggerEvents, trigger.Event)
				}
				report.WriteString(fmt.Sprintf("- **Triggers**: %s\n", strings.Join(triggerEvents, ", ")))
			}

			// Compliance status
			if len(workflow.ComplianceRules) > 0 {
				passed := 0
				for _, rule := range workflow.ComplianceRules {
					if rule.Status == "pass" {
						passed++
					}
				}
				report.WriteString(fmt.Sprintf("- **Compliance**: %d/%d rules passed\n", passed, len(workflow.ComplianceRules)))
			}

			report.WriteString(fmt.Sprintf("- **URL**: %s\n\n", workflow.HTMLURL))
		}
	}

	// Tools Used
	if len(analysis.Statistics.ToolsUsed) > 0 {
		report.WriteString("## Security Tools Used\n\n")
		for tool, count := range analysis.Statistics.ToolsUsed {
			report.WriteString(fmt.Sprintf("- **%s**: %d workflows\n", tool, count))
		}
		report.WriteString("\n")
	}

	return report.String()
}
