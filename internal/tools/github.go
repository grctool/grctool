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
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/transport"
	"github.com/grctool/grctool/internal/vcr"
)

// GitHubTool provides access to GitHub API for security evidence collection
type GitHubTool struct {
	config     *config.GitHubToolConfig
	httpClient *http.Client
	logger     logger.Logger
}

// NewGitHubTool creates a new GitHub tool
func NewGitHubTool(cfg *config.Config, log logger.Logger) Tool {
	// Create HTTP transport - start with default
	var httpTransport http.RoundTripper = http.DefaultTransport

	// Check environment for VCR_MODE (test/dev only)
	vcrConfig := vcr.FromEnvironment()
	if vcrConfig != nil && vcrConfig.Enabled {
		httpTransport = vcr.New(vcrConfig)
	} else {
		// Add logging if VCR not enabled
		httpTransport = transport.NewLoggingTransport(httpTransport, log.WithComponent("github-api"))
	}

	return &GitHubTool{
		config: &cfg.Evidence.Tools.GitHub,
		httpClient: &http.Client{
			Timeout:   30 * time.Second,
			Transport: httpTransport,
		},
		logger: log,
	}
}

// Name returns the tool name
func (gt *GitHubTool) Name() string {
	return "github_searcher"
}

// Description returns the tool description
func (gt *GitHubTool) Description() string {
	return "Search GitHub repository for security-related issues, pull requests, and discussions"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (gt *GitHubTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        gt.Name(),
		Description: gt.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query for GitHub issues (e.g., 'security vulnerability encryption')",
				},
				"labels": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "GitHub labels to filter by (e.g., ['security', 'bug', 'compliance'])",
				},
				"include_closed": map[string]interface{}{
					"type":        "boolean",
					"description": "Include closed issues in search results",
					"default":     false,
				},
			},
			"required": []string{"query"},
		},
	}
}

// Execute runs the GitHub search tool
func (gt *GitHubTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	gt.logger.Debug("Executing GitHub search", logger.Field{Key: "params", Value: params})

	// Extract parameters
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return "", nil, fmt.Errorf("query parameter is required")
	}

	var labels []string
	if l, ok := params["labels"].([]interface{}); ok {
		for _, label := range l {
			if str, ok := label.(string); ok {
				labels = append(labels, str)
			}
		}
	}

	includeClosed := false
	if ic, ok := params["include_closed"].(bool); ok {
		includeClosed = ic
	}

	// Modify query based on includeClosed
	if !includeClosed {
		query += " is:open"
	}

	// Search for issues
	issues, err := gt.SearchSecurityIssues(ctx, query, labels)
	if err != nil {
		return "", nil, fmt.Errorf("failed to search GitHub issues: %w", err)
	}

	// Limit results if too many
	if len(issues) > gt.config.MaxIssues {
		issues = issues[:gt.config.MaxIssues]
	}

	// Generate report
	report := gt.generateReport(issues)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "github",
		Resource:    fmt.Sprintf("GitHub repository: %s", gt.config.Repository),
		Content:     report,
		Relevance:   gt.calculateOverallRelevance(issues),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"repository":     gt.config.Repository,
			"issue_count":    len(issues),
			"query":          query,
			"labels":         labels,
			"include_closed": includeClosed,
		},
	}

	return report, source, nil
}

// generateReport creates a formatted report from GitHub issues
func (gt *GitHubTool) generateReport(issues []models.GitHubIssueResult) string {
	if len(issues) == 0 {
		return "No relevant GitHub issues found."
	}

	var report strings.Builder
	report.WriteString("# GitHub Security Evidence\n\n")
	report.WriteString(fmt.Sprintf("Repository: %s\n", gt.config.Repository))
	report.WriteString(fmt.Sprintf("Issues Found: %d\n\n", len(issues)))

	for _, issue := range issues {
		report.WriteString(fmt.Sprintf("## Issue #%d: %s\n", issue.Number, issue.Title))
		report.WriteString(fmt.Sprintf("- **State**: %s\n", issue.State))
		report.WriteString(fmt.Sprintf("- **Created**: %s\n", issue.CreatedAt.Format("2006-01-02")))
		report.WriteString(fmt.Sprintf("- **Updated**: %s\n", issue.UpdatedAt.Format("2006-01-02")))
		if issue.ClosedAt != nil {
			report.WriteString(fmt.Sprintf("- **Closed**: %s\n", issue.ClosedAt.Format("2006-01-02")))
		}
		if len(issue.Labels) > 0 {
			labelNames := make([]string, len(issue.Labels))
			for i, label := range issue.Labels {
				labelNames[i] = label.Name
			}
			report.WriteString(fmt.Sprintf("- **Labels**: %s\n", strings.Join(labelNames, ", ")))
		}
		report.WriteString(fmt.Sprintf("- **Relevance Score**: %.2f\n", issue.Relevance))
		report.WriteString(fmt.Sprintf("- **URL**: %s\n", issue.URL))

		// Include body excerpt
		if issue.Body != "" {
			excerpt := issue.Body
			if len(excerpt) > 500 {
				excerpt = excerpt[:500] + "..."
			}
			report.WriteString(fmt.Sprintf("\n**Description**:\n%s\n", excerpt))
		}
		report.WriteString("\n---\n\n")
	}

	return report.String()
}

// calculateOverallRelevance calculates the overall relevance of the search results
func (gt *GitHubTool) calculateOverallRelevance(issues []models.GitHubIssueResult) float64 {
	if len(issues) == 0 {
		return 0.0
	}

	totalRelevance := 0.0
	for _, issue := range issues {
		totalRelevance += issue.Relevance
	}

	// Average relevance with a bonus for finding multiple issues
	avgRelevance := totalRelevance / float64(len(issues))

	// Bonus for finding multiple relevant issues
	if len(issues) >= 5 {
		avgRelevance += 0.2
	} else if len(issues) >= 2 {
		avgRelevance += 0.1
	}

	// Cap at 1.0
	if avgRelevance > 1.0 {
		avgRelevance = 1.0
	}

	return avgRelevance
}

// SearchSecurityIssues searches for GitHub issues related to security evidence
func (gt *GitHubTool) SearchSecurityIssues(ctx context.Context, query string, labels []string) ([]models.GitHubIssueResult, error) {
	// Build search query
	searchQuery := fmt.Sprintf("repo:%s %s", gt.config.Repository, query)

	if len(labels) > 0 {
		for _, label := range labels {
			searchQuery += fmt.Sprintf(" label:\"%s\"", label)
		}
	}

	// Make API request
	url := fmt.Sprintf("https://api.github.com/search/issues?q=%s&sort=updated&order=desc&per_page=50",
		url.QueryEscape(searchQuery))

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if gt.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+gt.config.APIToken)
	}

	resp, err := gt.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	// Parse response
	var searchResponse struct {
		Items      []models.GitHubIssueResult `json:"items"`
		TotalCount int                        `json:"total_count"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Calculate relevance scores
	for i := range searchResponse.Items {
		searchResponse.Items[i].Relevance = gt.calculateRelevance(searchResponse.Items[i], query, labels)
	}

	return searchResponse.Items, nil
}

// calculateRelevance calculates a relevance score for a GitHub issue
func (gt *GitHubTool) calculateRelevance(issue models.GitHubIssueResult, query string, labels []string) float64 {
	score := 0.0

	// Title relevance
	titleLower := strings.ToLower(issue.Title)
	queryLower := strings.ToLower(query)
	if strings.Contains(titleLower, queryLower) {
		score += 0.5
	}

	// Body relevance
	bodyLower := strings.ToLower(issue.Body)
	if strings.Contains(bodyLower, queryLower) {
		score += 0.3
	}

	// Label matching
	labelMatch := 0
	for _, searchLabel := range labels {
		for _, issueLabel := range issue.Labels {
			if strings.EqualFold(issueLabel.Name, searchLabel) {
				labelMatch++
				break
			}
		}
	}
	if len(labels) > 0 {
		score += float64(labelMatch) / float64(len(labels)) * 0.4
	}

	// State bonus (open issues are more relevant)
	if issue.State == "open" {
		score += 0.2
	}

	// Recent activity bonus
	timeSinceUpdate := time.Since(issue.UpdatedAt)
	if timeSinceUpdate < 30*24*time.Hour { // Within 30 days
		score += 0.1
	}

	return score
}

// GetClaudeToolSchema returns the Claude tool schema for GitHub integration
func (gt *GitHubTool) GetClaudeToolSchema() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        "github_search",
		Description: "Search GitHub repository for security-related issues, pull requests, and discussions",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query for GitHub issues (e.g., 'security vulnerability encryption')",
				},
				"labels": map[string]interface{}{
					"type":        "array",
					"items":       map[string]interface{}{"type": "string"},
					"description": "GitHub labels to filter by (e.g., ['security', 'bug', 'compliance'])",
				},
			},
			"required": []string{"query"},
		},
	}
}
