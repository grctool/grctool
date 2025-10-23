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
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/auth"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/transport"
)

// GitHubEnhancedTool provides enhanced GitHub API searching capabilities
type GitHubEnhancedTool struct {
	config       *config.GitHubToolConfig
	httpClient   *http.Client
	logger       logger.Logger
	cacheDir     string
	authProvider auth.AuthProvider
}

// NewGitHubEnhancedTool creates a new enhanced GitHub searcher tool
func NewGitHubEnhancedTool(cfg *config.Config, log logger.Logger) Tool {
	// Create HTTP transport with logging if enabled
	httpTransport := http.DefaultTransport
	httpTransport = transport.NewLoggingTransport(httpTransport, log.WithComponent("github-enhanced-api"))

	// Set up cache directory
	cacheDir := filepath.Join(cfg.Storage.DataDir, "github_cache")

	// Create auth provider - use GitHub token from auth config, fallback to tool config
	githubToken := cfg.Auth.GitHub.Token
	if githubToken == "" {
		githubToken = cfg.Evidence.Tools.GitHub.APIToken
	}

	var authProvider auth.AuthProvider
	if githubToken != "" {
		authProvider = auth.NewGitHubAuthProvider(githubToken, cfg.Auth.CacheDir, log)
	} else {
		// Use GitHub provider with empty token to show proper unauthenticated state
		authProvider = auth.NewGitHubAuthProvider("", cfg.Auth.CacheDir, log)
	}

	return &GitHubEnhancedTool{
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
func (get *GitHubEnhancedTool) Name() string {
	return "github-enhanced"
}

// Description returns the tool description
func (get *GitHubEnhancedTool) Description() string {
	return "Enhanced GitHub repository searcher with multiple search types, date filtering, and caching"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (get *GitHubEnhancedTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        get.Name(),
		Description: get.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query for GitHub content (e.g., 'security vulnerability encryption')",
				},
				"search_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of search: commit, workflow, issue, pr, or all",
					"enum":        []string{"commit", "workflow", "issue", "pr", "all"},
					"default":     "all",
				},
				"since": map[string]interface{}{
					"type":        "string",
					"description": "Date filter - only results since this date (YYYY-MM-DD format)",
					"pattern":     "^\\d{4}-\\d{2}-\\d{2}$",
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return",
					"minimum":     1,
					"maximum":     500,
					"default":     50,
				},
				"use_cache": map[string]interface{}{
					"type":        "boolean",
					"description": "Use cached API results when available",
					"default":     true,
				},
				"labels": map[string]interface{}{
					"type":        "array",
					"description": "Filter by GitHub labels (for issues and PRs)",
					"items": map[string]interface{}{
						"type": "string",
					},
				},
			},
			"required": []string{"query"},
		},
	}
}

// Execute runs the enhanced GitHub search tool
func (get *GitHubEnhancedTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	startTime := time.Now()
	correlationID := GenerateCorrelationID()

	get.logger.Debug("Executing enhanced GitHub searcher",
		logger.String("correlation_id", correlationID),
		logger.Field{Key: "params", Value: params})

	// Check authentication status
	authStatus := get.authProvider.GetStatus(ctx)

	// Attempt authentication if auth is required and not already authenticated
	if get.authProvider.IsAuthRequired() && !authStatus.Authenticated {
		if err := get.authProvider.Authenticate(ctx); err != nil {
			authStatus.Error = err.Error()

			// For GitHub, we can still work with cached data if token is not available
			get.logger.Warn("GitHub authentication failed, will use cached data only",
				logger.Field{Key: "error", Value: err})
		} else {
			// Refresh auth status after successful authentication
			authStatus = get.authProvider.GetStatus(ctx)
		}
	}

	// Extract parameters with defaults
	query, ok := params["query"].(string)
	if !ok || query == "" {
		return "", nil, fmt.Errorf("query parameter is required")
	}

	searchType := "all"
	if st, ok := params["search_type"].(string); ok {
		searchType = st
	}

	since := ""
	if s, ok := params["since"].(string); ok {
		since = s
	}

	limit := 50
	if l, ok := params["limit"].(int); ok {
		limit = l
	}

	useCache := true
	if uc, ok := params["use_cache"].(bool); ok {
		useCache = uc
	}

	var labels []string
	if l, ok := params["labels"].([]interface{}); ok {
		for _, label := range l {
			if str, ok := label.(string); ok {
				labels = append(labels, str)
			}
		}
	}

	// Check cache first if enabled
	var results *GitHubSearchResults
	var err error
	cacheKey := get.generateCacheKey(query, searchType, since, limit, labels)

	if useCache {
		if cachedResults := get.loadFromCache(cacheKey); cachedResults != nil {
			get.logger.Debug("Using cached GitHub search results", logger.String("cache_key", cacheKey))
			results = cachedResults
		}
	}

	// If no cached results, perform new search (only if authenticated or auth not required)
	if results == nil {
		// Check if we can make API calls
		if get.authProvider.IsAuthRequired() && !authStatus.Authenticated {
			// Can't make API calls without authentication, return empty results
			get.logger.Warn("Cannot perform GitHub API search without authentication, returning empty results")
			results = &GitHubSearchResults{}
		} else {
			results, err = get.performEnhancedSearch(ctx, query, searchType, since, limit, labels)
			if err != nil {
				return "", nil, fmt.Errorf("failed to search GitHub: %w", err)
			}

			// Cache results if caching is enabled
			if useCache {
				get.saveToCache(cacheKey, results)
			}
		}
	}

	// Generate report
	report := get.generateEnhancedReport(results, query, searchType)

	// Get final auth status
	finalAuthStatus := get.authProvider.GetStatus(ctx)
	duration := time.Since(startTime)

	// Determine data source
	dataSource := "cache"
	if finalAuthStatus.Authenticated && !finalAuthStatus.CacheUsed {
		dataSource = "api"
	}

	// Create evidence source with auth metadata
	source := &models.EvidenceSource{
		Type:        "github-enhanced",
		Resource:    fmt.Sprintf("GitHub repository: %s", get.config.Repository),
		Content:     report,
		Relevance:   get.calculateEnhancedRelevance(results, query),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"repository":     get.config.Repository,
			"search_type":    searchType,
			"total_results":  results.TotalCount(),
			"query":          query,
			"since":          since,
			"labels":         labels,
			"cache_used":     useCache && results != nil,
			"correlation_id": correlationID,
			"duration_ms":    duration.Milliseconds(),
			"auth_status": map[string]interface{}{
				"authenticated": finalAuthStatus.Authenticated,
				"provider":      finalAuthStatus.Provider,
				"cache_used":    finalAuthStatus.CacheUsed,
				"token_present": finalAuthStatus.TokenPresent,
			},
			"data_source": dataSource,
		},
	}

	return report, source, nil
}

// performEnhancedSearch performs the actual GitHub API searches
func (get *GitHubEnhancedTool) performEnhancedSearch(ctx context.Context, query, searchType, since string, limit int, labels []string) (*GitHubSearchResults, error) {
	results := &GitHubSearchResults{}

	switch searchType {
	case "commit":
		commits, err := get.searchCommits(ctx, query, since, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to search commits: %w", err)
		}
		results.Commits = commits

	case "workflow":
		workflows, err := get.searchWorkflows(ctx, query, since, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to search workflows: %w", err)
		}
		results.Workflows = workflows

	case "issue":
		issues, err := get.searchIssues(ctx, query, since, limit, labels, "issue")
		if err != nil {
			return nil, fmt.Errorf("failed to search issues: %w", err)
		}
		results.Issues = issues

	case "pr":
		prs, err := get.searchIssues(ctx, query, since, limit, labels, "pr")
		if err != nil {
			return nil, fmt.Errorf("failed to search pull requests: %w", err)
		}
		results.PullRequests = prs

	case "all":
		// Search all types with reduced limits per type
		perTypeLimit := limit / 4
		if perTypeLimit < 5 {
			perTypeLimit = 5
		}

		if commits, err := get.searchCommits(ctx, query, since, perTypeLimit); err == nil {
			results.Commits = commits
		}

		if workflows, err := get.searchWorkflows(ctx, query, since, perTypeLimit); err == nil {
			results.Workflows = workflows
		}

		if issues, err := get.searchIssues(ctx, query, since, perTypeLimit, labels, "issue"); err == nil {
			results.Issues = issues
		}

		if prs, err := get.searchIssues(ctx, query, since, perTypeLimit, labels, "pr"); err == nil {
			results.PullRequests = prs
		}

	default:
		return nil, fmt.Errorf("unsupported search type: %s", searchType)
	}

	return results, nil
}

// searchCommits searches for commits
func (get *GitHubEnhancedTool) searchCommits(ctx context.Context, query, since string, limit int) ([]GitHubCommitResult, error) {
	searchQuery := fmt.Sprintf("repo:%s %s", get.config.Repository, query)

	if since != "" {
		searchQuery += fmt.Sprintf(" committer-date:>%s", since)
	}

	url := fmt.Sprintf("https://api.github.com/search/commits?q=%s&sort=committer-date&order=desc&per_page=%d",
		url.QueryEscape(searchQuery), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.cloak-preview")
	if get.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+get.config.APIToken)
	}

	resp, err := get.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var searchResponse struct {
		Items []GitHubCommitResult `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Calculate relevance scores
	for i := range searchResponse.Items {
		searchResponse.Items[i].Relevance = get.calculateCommitRelevance(searchResponse.Items[i], query)
	}

	return searchResponse.Items, nil
}

// searchWorkflows searches for workflow files
func (get *GitHubEnhancedTool) searchWorkflows(ctx context.Context, query, since string, limit int) ([]GitHubWorkflowResult, error) {
	searchQuery := fmt.Sprintf("repo:%s %s path:.github/workflows", get.config.Repository, query)

	url := fmt.Sprintf("https://api.github.com/search/code?q=%s&sort=indexed&order=desc&per_page=%d",
		url.QueryEscape(searchQuery), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if get.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+get.config.APIToken)
	}

	resp, err := get.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var searchResponse struct {
		Items []GitHubWorkflowResult `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Calculate relevance scores
	for i := range searchResponse.Items {
		searchResponse.Items[i].Relevance = get.calculateWorkflowRelevance(searchResponse.Items[i], query)
	}

	return searchResponse.Items, nil
}

// searchIssues searches for issues or pull requests
func (get *GitHubEnhancedTool) searchIssues(ctx context.Context, query, since string, limit int, labels []string, itemType string) ([]models.GitHubIssueResult, error) {
	searchQuery := fmt.Sprintf("repo:%s %s", get.config.Repository, query)

	if itemType == "pr" {
		searchQuery += " is:pr"
	} else {
		searchQuery += " is:issue"
	}

	if since != "" {
		searchQuery += fmt.Sprintf(" created:>%s", since)
	}

	for _, label := range labels {
		searchQuery += fmt.Sprintf(" label:\"%s\"", label)
	}

	url := fmt.Sprintf("https://api.github.com/search/issues?q=%s&sort=updated&order=desc&per_page=%d",
		url.QueryEscape(searchQuery), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if get.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+get.config.APIToken)
	}

	resp, err := get.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var searchResponse struct {
		Items []models.GitHubIssueResult `json:"items"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&searchResponse); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Calculate relevance scores
	for i := range searchResponse.Items {
		searchResponse.Items[i].Relevance = get.calculateIssueRelevance(searchResponse.Items[i], query, labels)
	}

	return searchResponse.Items, nil
}

// generateEnhancedReport creates a formatted report from GitHub search results
func (get *GitHubEnhancedTool) generateEnhancedReport(results *GitHubSearchResults, query, searchType string) string {
	var report strings.Builder

	report.WriteString("# Enhanced GitHub Security Evidence\n\n")
	report.WriteString(fmt.Sprintf("Repository: %s\n", get.config.Repository))
	report.WriteString(fmt.Sprintf("Search Query: %s\n", query))
	report.WriteString(fmt.Sprintf("Search Type: %s\n", searchType))
	report.WriteString(fmt.Sprintf("Total Results: %d\n\n", results.TotalCount()))

	// Report commits
	if len(results.Commits) > 0 {
		report.WriteString(fmt.Sprintf("## Commits (%d)\n\n", len(results.Commits)))
		for _, commit := range results.Commits {
			report.WriteString(fmt.Sprintf("### %s\n", commit.SHA[:8]))
			report.WriteString(fmt.Sprintf("- **Message**: %s\n", commit.Commit.Message))
			report.WriteString(fmt.Sprintf("- **Author**: %s\n", commit.Commit.Author.Name))
			report.WriteString(fmt.Sprintf("- **Date**: %s\n", commit.Commit.Author.Date.Format("2006-01-02")))
			report.WriteString(fmt.Sprintf("- **Relevance**: %.2f\n", commit.Relevance))
			report.WriteString(fmt.Sprintf("- **URL**: %s\n\n", commit.HTMLURL))
		}
	}

	// Report workflows
	if len(results.Workflows) > 0 {
		report.WriteString(fmt.Sprintf("## Workflows (%d)\n\n", len(results.Workflows)))
		for _, workflow := range results.Workflows {
			report.WriteString(fmt.Sprintf("### %s\n", workflow.Name))
			report.WriteString(fmt.Sprintf("- **Path**: %s\n", workflow.Path))
			report.WriteString(fmt.Sprintf("- **Relevance**: %.2f\n", workflow.Relevance))
			report.WriteString(fmt.Sprintf("- **URL**: %s\n\n", workflow.HTMLURL))
		}
	}

	// Report issues
	if len(results.Issues) > 0 {
		report.WriteString(fmt.Sprintf("## Issues (%d)\n\n", len(results.Issues)))
		for _, issue := range results.Issues {
			report.WriteString(fmt.Sprintf("### Issue #%d: %s\n", issue.Number, issue.Title))
			report.WriteString(fmt.Sprintf("- **State**: %s\n", issue.State))
			report.WriteString(fmt.Sprintf("- **Created**: %s\n", issue.CreatedAt.Format("2006-01-02")))
			if len(issue.Labels) > 0 {
				report.WriteString(fmt.Sprintf("- **Labels**: %s\n", strings.Join(func() []string {
					names := make([]string, len(issue.Labels))
					for i, l := range issue.Labels {
						names[i] = l.Name
					}
					return names
				}(), ", ")))
			}
			report.WriteString(fmt.Sprintf("- **Relevance**: %.2f\n", issue.Relevance))
			report.WriteString(fmt.Sprintf("- **URL**: %s\n\n", issue.URL))
		}
	}

	// Report pull requests
	if len(results.PullRequests) > 0 {
		report.WriteString(fmt.Sprintf("## Pull Requests (%d)\n\n", len(results.PullRequests)))
		for _, pr := range results.PullRequests {
			report.WriteString(fmt.Sprintf("### PR #%d: %s\n", pr.Number, pr.Title))
			report.WriteString(fmt.Sprintf("- **State**: %s\n", pr.State))
			report.WriteString(fmt.Sprintf("- **Created**: %s\n", pr.CreatedAt.Format("2006-01-02")))
			if len(pr.Labels) > 0 {
				report.WriteString(fmt.Sprintf("- **Labels**: %s\n", strings.Join(func() []string {
					names := make([]string, len(pr.Labels))
					for i, l := range pr.Labels {
						names[i] = l.Name
					}
					return names
				}(), ", ")))
			}
			report.WriteString(fmt.Sprintf("- **Relevance**: %.2f\n", pr.Relevance))
			report.WriteString(fmt.Sprintf("- **URL**: %s\n\n", pr.URL))
		}
	}

	if results.TotalCount() == 0 {
		report.WriteString("No relevant results found for the given search criteria.\n")
	}

	return report.String()
}

// calculateEnhancedRelevance calculates the overall relevance of the search results
func (get *GitHubEnhancedTool) calculateEnhancedRelevance(results *GitHubSearchResults, query string) float64 {
	totalCount := results.TotalCount()
	if totalCount == 0 {
		return 0.0
	}

	totalRelevance := 0.0

	// Sum relevance from all result types
	for _, commit := range results.Commits {
		totalRelevance += commit.Relevance
	}
	for _, workflow := range results.Workflows {
		totalRelevance += workflow.Relevance
	}
	for _, issue := range results.Issues {
		totalRelevance += issue.Relevance
	}
	for _, pr := range results.PullRequests {
		totalRelevance += pr.Relevance
	}

	// Average relevance with bonus for multiple result types
	avgRelevance := totalRelevance / float64(totalCount)

	// Count distinct result types found
	resultTypes := 0
	if len(results.Commits) > 0 {
		resultTypes++
	}
	if len(results.Workflows) > 0 {
		resultTypes++
	}
	if len(results.Issues) > 0 {
		resultTypes++
	}
	if len(results.PullRequests) > 0 {
		resultTypes++
	}

	// Bonus for diversity in result types
	if resultTypes >= 3 {
		avgRelevance += 0.2
	} else if resultTypes >= 2 {
		avgRelevance += 0.1
	}

	// Cap at 1.0
	if avgRelevance > 1.0 {
		avgRelevance = 1.0
	}

	return avgRelevance
}

// calculateCommitRelevance calculates relevance for commit results
func (get *GitHubEnhancedTool) calculateCommitRelevance(commit GitHubCommitResult, query string) float64 {
	score := 0.0
	queryLower := strings.ToLower(query)

	// Message relevance
	messageLower := strings.ToLower(commit.Commit.Message)
	if strings.Contains(messageLower, queryLower) {
		score += 0.6
	}

	// Recent commits get bonus
	if time.Since(commit.Commit.Author.Date) < 30*24*time.Hour {
		score += 0.2
	}

	// Security-related keywords get bonus
	securityKeywords := []string{"security", "encrypt", "auth", "vulner", "cve", "fix"}
	for _, keyword := range securityKeywords {
		if strings.Contains(messageLower, keyword) {
			score += 0.2
			break
		}
	}

	return score
}

// calculateWorkflowRelevance calculates relevance for workflow results
func (get *GitHubEnhancedTool) calculateWorkflowRelevance(workflow GitHubWorkflowResult, query string) float64 {
	score := 0.0
	queryLower := strings.ToLower(query)

	// Name relevance
	nameLower := strings.ToLower(workflow.Name)
	if strings.Contains(nameLower, queryLower) {
		score += 0.5
	}

	// Path relevance
	pathLower := strings.ToLower(workflow.Path)
	if strings.Contains(pathLower, queryLower) {
		score += 0.3
	}

	// Security workflow patterns
	securityPatterns := []string{"security", "scan", "audit", "compliance", "test"}
	for _, pattern := range securityPatterns {
		if strings.Contains(nameLower, pattern) {
			score += 0.3
			break
		}
	}

	return score
}

// calculateIssueRelevance calculates relevance for issue/PR results
func (get *GitHubEnhancedTool) calculateIssueRelevance(issue models.GitHubIssueResult, query string, labels []string) float64 {
	score := 0.0
	queryLower := strings.ToLower(query)

	// Title relevance
	titleLower := strings.ToLower(issue.Title)
	if strings.Contains(titleLower, queryLower) {
		score += 0.5
	}

	// Body relevance
	bodyLower := strings.ToLower(issue.Body)
	if strings.Contains(bodyLower, queryLower) {
		score += 0.3
	}

	// Label matching
	if len(labels) > 0 {
		labelMatch := 0
		for _, searchLabel := range labels {
			for _, issueLabel := range issue.Labels {
				if strings.EqualFold(issueLabel.Name, searchLabel) {
					labelMatch++
					break
				}
			}
		}
		score += float64(labelMatch) / float64(len(labels)) * 0.4
	}

	// State bonus (open issues/PRs are more relevant)
	if issue.State == "open" {
		score += 0.2
	}

	// Recent activity bonus
	if time.Since(issue.UpdatedAt) < 30*24*time.Hour {
		score += 0.1
	}

	return score
}

// Cache management functions

func (get *GitHubEnhancedTool) generateCacheKey(query, searchType, since string, limit int, labels []string) string {
	keyData := fmt.Sprintf("q:%s|st:%s|s:%s|l:%d|labels:%v", query, searchType, since, limit, labels)
	return fmt.Sprintf("%x", md5.Sum([]byte(keyData)))
}

func (get *GitHubEnhancedTool) loadFromCache(cacheKey string) *GitHubSearchResults {
	searchesCacheDir := filepath.Join(get.cacheDir, "searches")
	cacheFile := filepath.Join(searchesCacheDir, fmt.Sprintf("%s.json", cacheKey))

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
		Results   GitHubSearchResults `json:"results"`
		Timestamp time.Time           `json:"timestamp"`
	}

	if err := json.Unmarshal(data, &cacheData); err != nil {
		return nil
	}

	// Check cache age (30 minutes expiry for API calls)
	if time.Since(cacheData.Timestamp) > 30*time.Minute {
		return nil
	}

	return &cacheData.Results
}

func (get *GitHubEnhancedTool) saveToCache(cacheKey string, results *GitHubSearchResults) {
	cacheData := struct {
		Results   GitHubSearchResults `json:"results"`
		Timestamp time.Time           `json:"timestamp"`
	}{
		Results:   *results,
		Timestamp: time.Now(),
	}

	// Create cache directory
	searchesCacheDir := filepath.Join(get.cacheDir, "searches")
	if err := os.MkdirAll(searchesCacheDir, 0755); err != nil {
		get.logger.Warn("Failed to create cache directory",
			logger.String("cache_dir", searchesCacheDir),
			logger.Field{Key: "error", Value: err})
		return
	}

	// Save to cache file
	cacheFile := filepath.Join(searchesCacheDir, fmt.Sprintf("%s.json", cacheKey))
	data, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		get.logger.Warn("Failed to marshal cache data",
			logger.String("cache_key", cacheKey),
			logger.Field{Key: "error", Value: err})
		return
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		get.logger.Warn("Failed to save GitHub search results to cache",
			logger.String("cache_key", cacheKey),
			logger.String("cache_file", cacheFile),
			logger.Field{Key: "error", Value: err})
	}
}

// GitHubSearchResults aggregates all search result types
type GitHubSearchResults struct {
	Commits      []GitHubCommitResult       `json:"commits"`
	Workflows    []GitHubWorkflowResult     `json:"workflows"`
	Issues       []models.GitHubIssueResult `json:"issues"`
	PullRequests []models.GitHubIssueResult `json:"pull_requests"`
}

func (gsr *GitHubSearchResults) TotalCount() int {
	return len(gsr.Commits) + len(gsr.Workflows) + len(gsr.Issues) + len(gsr.PullRequests)
}

// GitHubCommitResult represents a commit search result
type GitHubCommitResult struct {
	SHA       string     `json:"sha"`
	HTMLURL   string     `json:"html_url"`
	Commit    CommitData `json:"commit"`
	Relevance float64    `json:"relevance"`
}

type CommitData struct {
	Message string     `json:"message"`
	Author  AuthorData `json:"author"`
}

type AuthorData struct {
	Name string    `json:"name"`
	Date time.Time `json:"date"`
}

// GitHubWorkflowResult represents a workflow file search result
type GitHubWorkflowResult struct {
	Name      string  `json:"name"`
	Path      string  `json:"path"`
	HTMLURL   string  `json:"html_url"`
	Relevance float64 `json:"relevance"`
}
