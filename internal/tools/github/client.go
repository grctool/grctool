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

package github

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/auth"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/transport"
	"github.com/grctool/grctool/internal/vcr"
)

// GitHubClient provides comprehensive GitHub API access for SOC2 audit evidence
type GitHubClient struct {
	config       *config.GitHubToolConfig
	httpClient   *http.Client
	logger       logger.Logger
	baseURL      string
	graphqlURL   string
	cacheDir     string
	authProvider auth.AuthProvider
	rateLimiter  *time.Ticker // For GitHub Search API rate limiting (30 req/min)
}

// NewGitHubClient creates a new comprehensive GitHub API client
func NewGitHubClient(cfg *config.Config, log logger.Logger) *GitHubClient {
	// Start with default transport
	var httpTransport http.RoundTripper = http.DefaultTransport

	// Check environment for VCR_MODE (test/dev only)
	vcrConfig := vcr.FromEnvironment()
	if vcrConfig != nil && vcrConfig.Enabled {
		// VCR wraps the transport
		httpTransport = vcr.New(vcrConfig)
		log.Info("VCR enabled for GitHub client",
			logger.String("mode", string(vcrConfig.Mode)),
			logger.String("cassette_dir", vcrConfig.CassetteDir))
	} else {
		// Only add logging if VCR is not enabled (VCR has its own logging)
		httpTransport = transport.NewLoggingTransport(httpTransport, log.WithComponent("github-api-client"))
		log.Info("VCR disabled for GitHub client")
	}

	// Set up cache directory
	cacheDir := filepath.Join(cfg.Storage.DataDir, "github_cache")

	// Create auth provider - token is populated by config.Load() from multiple sources:
	// 1. cfg.Auth.GitHub.Token
	// 2. cfg.Evidence.Tools.GitHub.APIToken
	// 3. gh CLI (automatically via config loading)
	githubToken := cfg.Auth.GitHub.Token
	if githubToken == "" {
		githubToken = cfg.Evidence.Tools.GitHub.APIToken
	}

	var authProvider auth.AuthProvider
	if githubToken != "" {
		authProvider = auth.NewGitHubAuthProvider(githubToken, cfg.Auth.CacheDir, log)
		log.Debug("GitHub auth provider initialized with token")
	} else {
		authProvider = auth.NewGitHubAuthProvider("", cfg.Auth.CacheDir, log)
		log.Debug("GitHub auth provider initialized without token - gh CLI may be needed")
	}

	client := &GitHubClient{
		config: &cfg.Evidence.Tools.GitHub,
		httpClient: &http.Client{
			Timeout:   60 * time.Second,
			Transport: httpTransport,
		},
		logger:       log,
		baseURL:      "https://api.github.com",
		graphqlURL:   "https://api.github.com/graphql",
		cacheDir:     cacheDir,
		authProvider: authProvider,
	}

	// Set up rate limiting for GitHub Search API (30 requests/minute default)
	// This applies to /search/issues, /search/commits, /search/code endpoints
	rateLimit := cfg.Evidence.Tools.GitHub.RateLimit

	// Allow override via environment variable (useful for slow VCR recording)
	if envLimit := os.Getenv("GITHUB_RATE_LIMIT"); envLimit != "" {
		if parsed, err := strconv.Atoi(envLimit); err == nil && parsed > 0 {
			rateLimit = parsed
			log.Info("GitHub rate limit overridden by environment variable",
				logger.Int("requests_per_minute", rateLimit))
		}
	}

	if rateLimit == 0 {
		rateLimit = 30 // GitHub Search API limit
	}
	if rateLimit > 0 {
		// Convert requests/minute to time between requests
		client.rateLimiter = time.NewTicker(time.Minute / time.Duration(rateLimit))
		log.Info("GitHub Search API rate limiting enabled",
			logger.Int("requests_per_minute", rateLimit))
	}

	return client
}

// Close cleans up client resources
func (client *GitHubClient) Close() {
	if client.rateLimiter != nil {
		client.rateLimiter.Stop()
	}
}

// REST API methods

func (client *GitHubClient) makeRESTRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
	var reqBody io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(jsonBody)
	}

	url := client.baseURL + endpoint
	req, err := http.NewRequestWithContext(ctx, method, url, reqBody)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", "grctool/1.0")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	if client.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+client.config.APIToken)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}

	// Check for rate limiting
	if resp.StatusCode == 403 && resp.Header.Get("X-RateLimit-Remaining") == "0" {
		resetTime := resp.Header.Get("X-RateLimit-Reset")
		resp.Body.Close()
		return nil, fmt.Errorf("GitHub API rate limit exceeded, reset time: %s", resetTime)
	}

	return resp, nil
}

func (client *GitHubClient) makeGraphQLRequest(ctx context.Context, query string, variables map[string]interface{}) (*GraphQLResponse, error) {
	requestBody := map[string]interface{}{
		"query":     query,
		"variables": variables,
	}

	jsonBody, err := json.Marshal(requestBody)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal GraphQL request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", client.graphqlURL, bytes.NewReader(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create GraphQL request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "grctool/1.0")
	if client.config.APIToken != "" {
		req.Header.Set("Authorization", "bearer "+client.config.APIToken)
	}

	resp, err := client.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make GraphQL request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GraphQL API error %d: %s", resp.StatusCode, string(body))
	}

	var gqlResponse GraphQLResponse
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read GraphQL response: %w", err)
	}
	gqlResponse.Data = respBody

	if err := json.Unmarshal(respBody, &gqlResponse); err != nil {
		return nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	if len(gqlResponse.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL errors: %v", gqlResponse.Errors)
	}

	return &gqlResponse, nil
}

// Enhanced search capabilities

func (client *GitHubClient) PerformEnhancedSearch(ctx context.Context, query, searchType, since string, limit int, labels []string) (*GitHubSearchResults, error) {
	results := &GitHubSearchResults{}

	switch searchType {
	case "commit":
		commits, err := client.searchCommits(ctx, query, since, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to search commits: %w", err)
		}
		results.Commits = commits

	case "workflow":
		workflows, err := client.searchWorkflows(ctx, query, since, limit)
		if err != nil {
			return nil, fmt.Errorf("failed to search workflows: %w", err)
		}
		results.Workflows = workflows

	case "issue":
		issues, err := client.searchIssues(ctx, query, since, limit, labels, "issue")
		if err != nil {
			return nil, fmt.Errorf("failed to search issues: %w", err)
		}
		results.Issues = issues

	case "pr":
		prs, err := client.searchIssues(ctx, query, since, limit, labels, "pr")
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

		if commits, err := client.searchCommits(ctx, query, since, perTypeLimit); err == nil {
			results.Commits = commits
		}

		if workflows, err := client.searchWorkflows(ctx, query, since, perTypeLimit); err == nil {
			results.Workflows = workflows
		}

		if issues, err := client.searchIssues(ctx, query, since, perTypeLimit, labels, "issue"); err == nil {
			results.Issues = issues
		}

		if prs, err := client.searchIssues(ctx, query, since, perTypeLimit, labels, "pr"); err == nil {
			results.PullRequests = prs
		}

	default:
		return nil, fmt.Errorf("unsupported search type: %s", searchType)
	}

	return results, nil
}

func (client *GitHubClient) searchCommits(ctx context.Context, query, since string, limit int) ([]GitHubCommitResult, error) {
	// Rate limiting for GitHub Search API
	if client.rateLimiter != nil {
		select {
		case <-client.rateLimiter.C:
			// Continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	searchQuery := fmt.Sprintf("repo:%s %s", client.config.Repository, query)

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
	if client.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+client.config.APIToken)
	}

	resp, err := client.httpClient.Do(req)
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
		searchResponse.Items[i].Relevance = client.calculateCommitRelevance(searchResponse.Items[i], query)
	}

	return searchResponse.Items, nil
}

func (client *GitHubClient) searchWorkflows(ctx context.Context, query, since string, limit int) ([]GitHubWorkflowResult, error) {
	// Rate limiting for GitHub Search API
	if client.rateLimiter != nil {
		select {
		case <-client.rateLimiter.C:
			// Continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	searchQuery := fmt.Sprintf("repo:%s %s path:.github/workflows", client.config.Repository, query)

	url := fmt.Sprintf("https://api.github.com/search/code?q=%s&sort=indexed&order=desc&per_page=%d",
		url.QueryEscape(searchQuery), limit)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if client.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+client.config.APIToken)
	}

	resp, err := client.httpClient.Do(req)
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
		searchResponse.Items[i].Relevance = client.calculateWorkflowRelevance(searchResponse.Items[i], query)
	}

	return searchResponse.Items, nil
}

func (client *GitHubClient) searchIssues(ctx context.Context, query, since string, limit int, labels []string, itemType string) ([]models.GitHubIssueResult, error) {
	// Rate limiting for GitHub Search API
	if client.rateLimiter != nil {
		select {
		case <-client.rateLimiter.C:
			// Continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	searchQuery := fmt.Sprintf("repo:%s %s", client.config.Repository, query)

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
	if client.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+client.config.APIToken)
	}

	resp, err := client.httpClient.Do(req)
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
		searchResponse.Items[i].Relevance = client.calculateIssueRelevance(searchResponse.Items[i], query, labels)
	}

	return searchResponse.Items, nil
}

// Cache management

func (client *GitHubClient) generateCacheKey(query, searchType, since string, limit int, labels []string) string {
	keyData := fmt.Sprintf("q:%s|st:%s|s:%s|l:%d|labels:%v", query, searchType, since, limit, labels)
	return fmt.Sprintf("%x", md5.Sum([]byte(keyData)))
}

func (client *GitHubClient) LoadFromCache(cacheKey string) *GitHubSearchResults {
	searchesCacheDir := filepath.Join(client.cacheDir, "searches")
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

func (client *GitHubClient) SaveToCache(cacheKey string, results *GitHubSearchResults) {
	cacheData := struct {
		Results   GitHubSearchResults `json:"results"`
		Timestamp time.Time           `json:"timestamp"`
	}{
		Results:   *results,
		Timestamp: time.Now(),
	}

	// Create cache directory
	searchesCacheDir := filepath.Join(client.cacheDir, "searches")
	if err := os.MkdirAll(searchesCacheDir, 0755); err != nil {
		client.logger.Warn("Failed to create cache directory",
			logger.String("cache_dir", searchesCacheDir),
			logger.Field{Key: "error", Value: err})
		return
	}

	// Save to cache file
	cacheFile := filepath.Join(searchesCacheDir, fmt.Sprintf("%s.json", cacheKey))
	data, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		client.logger.Warn("Failed to marshal cache data",
			logger.String("cache_key", cacheKey),
			logger.Field{Key: "error", Value: err})
		return
	}

	if err := os.WriteFile(cacheFile, data, 0644); err != nil {
		client.logger.Warn("Failed to save GitHub search results to cache",
			logger.String("cache_key", cacheKey),
			logger.String("cache_file", cacheFile),
			logger.Field{Key: "error", Value: err})
	}
}

// Relevance calculation helper methods

func (client *GitHubClient) calculateCommitRelevance(commit GitHubCommitResult, query string) float64 {
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

func (client *GitHubClient) calculateWorkflowRelevance(workflow GitHubWorkflowResult, query string) float64 {
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

func (client *GitHubClient) calculateIssueRelevance(issue models.GitHubIssueResult, query string, labels []string) float64 {
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

// Legacy search method for backward compatibility
func (client *GitHubClient) SearchSecurityIssues(ctx context.Context, query string, labels []string) ([]models.GitHubIssueResult, error) {
	// Rate limiting for GitHub Search API
	if client.rateLimiter != nil {
		select {
		case <-client.rateLimiter.C:
			// Continue
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	// Build search query
	searchQuery := fmt.Sprintf("repo:%s %s", client.config.Repository, query)

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
	if client.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+client.config.APIToken)
	}

	resp, err := client.httpClient.Do(req)
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
		searchResponse.Items[i].Relevance = client.calculateIssueRelevance(searchResponse.Items[i], query, labels)
	}

	return searchResponse.Items, nil
}

// GetRepositoryCollaborators gets all repository collaborators with their permissions
func (client *GitHubClient) GetRepositoryCollaborators(ctx context.Context, owner, repo string) ([]models.GitHubCollaborator, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/collaborators", owner, repo)

	resp, err := client.makeRESTRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var collaborators []models.GitHubCollaborator
	if err := json.NewDecoder(resp.Body).Decode(&collaborators); err != nil {
		return nil, fmt.Errorf("failed to decode collaborators response: %w", err)
	}

	// Get detailed permissions for each collaborator
	for i := range collaborators {
		permissions, err := client.GetUserRepositoryPermissions(ctx, owner, repo, collaborators[i].Login)
		if err != nil {
			client.logger.Warn("Failed to get detailed permissions for user",
				logger.String("user", collaborators[i].Login),
				logger.Field{Key: "error", Value: err})
			continue
		}
		collaborators[i].DetailedPermissions = permissions
	}

	return collaborators, nil
}

// GetUserRepositoryPermissions gets detailed permissions for a specific user
func (client *GitHubClient) GetUserRepositoryPermissions(ctx context.Context, owner, repo, username string) (*models.GitHubPermissions, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/collaborators/%s/permission", owner, repo, username)

	resp, err := client.makeRESTRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var permissions models.GitHubPermissions
	if err := json.NewDecoder(resp.Body).Decode(&permissions); err != nil {
		return nil, fmt.Errorf("failed to decode permissions response: %w", err)
	}

	return &permissions, nil
}

// GetRepositoryTeams gets all teams with access to the repository
func (client *GitHubClient) GetRepositoryTeams(ctx context.Context, owner, repo string) ([]models.GitHubTeam, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/teams", owner, repo)

	resp, err := client.makeRESTRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var teams []models.GitHubTeam
	if err := json.NewDecoder(resp.Body).Decode(&teams); err != nil {
		return nil, fmt.Errorf("failed to decode teams response: %w", err)
	}

	// Get team members for each team
	for i := range teams {
		members, err := client.GetTeamMembers(ctx, teams[i].ID)
		if err != nil {
			client.logger.Warn("Failed to get team members",
				logger.String("team", teams[i].Name),
				logger.Int("team_id", teams[i].ID),
				logger.Field{Key: "error", Value: err})
			continue
		}
		teams[i].Members = members
	}

	return teams, nil
}

// GetTeamMembers gets all members of a specific team
func (client *GitHubClient) GetTeamMembers(ctx context.Context, teamID int) ([]models.GitHubTeamMember, error) {
	endpoint := fmt.Sprintf("/teams/%d/members", teamID)

	resp, err := client.makeRESTRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var members []models.GitHubTeamMember
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, fmt.Errorf("failed to decode team members response: %w", err)
	}

	return members, nil
}

// GetRepositoryBranches gets all branches in the repository
func (client *GitHubClient) GetRepositoryBranches(ctx context.Context, owner, repo string) ([]models.GitHubBranch, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/branches", owner, repo)

	resp, err := client.makeRESTRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var branches []models.GitHubBranch
	if err := json.NewDecoder(resp.Body).Decode(&branches); err != nil {
		return nil, fmt.Errorf("failed to decode branches response: %w", err)
	}

	// Get protection rules for each branch
	for i := range branches {
		protection, err := client.GetBranchProtection(ctx, owner, repo, branches[i].Name)
		if err != nil {
			client.logger.Warn("Failed to get branch protection",
				logger.String("branch", branches[i].Name),
				logger.Field{Key: "error", Value: err})
			continue
		}
		branches[i].Protection = protection
	}

	return branches, nil
}

// GetBranchProtection gets branch protection rules for a specific branch
func (client *GitHubClient) GetBranchProtection(ctx context.Context, owner, repo, branch string) (*models.GitHubBranchProtection, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/branches/%s/protection", owner, repo, branch)

	resp, err := client.makeRESTRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// Branch protection not enabled
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var protection models.GitHubBranchProtection
	if err := json.NewDecoder(resp.Body).Decode(&protection); err != nil {
		return nil, fmt.Errorf("failed to decode branch protection response: %w", err)
	}

	return &protection, nil
}

// GetDeploymentEnvironments gets all deployment environments using REST API
func (client *GitHubClient) GetDeploymentEnvironments(ctx context.Context, owner, repo string) ([]models.GitHubEnvironment, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/environments", owner, repo)

	resp, err := client.makeRESTRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 404 {
		// No environments configured - return empty list
		return []models.GitHubEnvironment{}, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var envResponse struct {
		TotalCount   int `json:"total_count"`
		Environments []struct {
			ID              int    `json:"id"`
			Name            string `json:"name"`
			ProtectionRules []struct {
				ID        int    `json:"id"`
				Type      string `json:"type"`
				WaitTimer int    `json:"wait_timer,omitempty"`
				Reviewers []struct {
					Type     string `json:"type"` // User or Team
					Reviewer struct {
						Login string `json:"login,omitempty"`
						Name  string `json:"name,omitempty"`
						Email string `json:"email,omitempty"`
						Slug  string `json:"slug,omitempty"`
					} `json:"reviewer,omitempty"`
				} `json:"reviewers,omitempty"`
			} `json:"protection_rules"`
			CreatedAt time.Time `json:"created_at"`
			UpdatedAt time.Time `json:"updated_at"`
		} `json:"environments"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&envResponse); err != nil {
		return nil, fmt.Errorf("failed to decode environments response: %w", err)
	}

	// Convert to our model format
	environments := make([]models.GitHubEnvironment, 0, len(envResponse.Environments))
	for _, env := range envResponse.Environments {
		ghEnv := models.GitHubEnvironment{
			ID:              fmt.Sprintf("%d", env.ID),
			Name:            env.Name,
			CreatedAt:       env.CreatedAt,
			UpdatedAt:       env.UpdatedAt,
			ProtectionRules: make([]models.GitHubEnvironmentProtection, 0, len(env.ProtectionRules)),
		}

		// Convert protection rules
		for _, rule := range env.ProtectionRules {
			protection := models.GitHubEnvironmentProtection{
				Type:      rule.Type,
				WaitTimer: rule.WaitTimer,
			}

			// Convert reviewers
			for _, reviewer := range rule.Reviewers {
				ghReviewer := models.GitHubRequiredReviewer{
					Type:  reviewer.Type,
					Login: reviewer.Reviewer.Login,
					Name:  reviewer.Reviewer.Name,
					Email: reviewer.Reviewer.Email,
					Slug:  reviewer.Reviewer.Slug,
				}
				protection.RequiredReviewers = append(protection.RequiredReviewers, ghReviewer)
			}

			ghEnv.ProtectionRules = append(ghEnv.ProtectionRules, protection)
		}

		environments = append(environments, ghEnv)
	}

	return environments, nil
}

// GetRepositorySecurity gets repository security settings
func (client *GitHubClient) GetRepositorySecurity(ctx context.Context, owner, repo string) (*models.GitHubSecuritySettings, error) {
	// Get vulnerability alerts
	alertsEnabled, err := client.getVulnerabilityAlerts(ctx, owner, repo)
	if err != nil {
		client.logger.Warn("Failed to get vulnerability alerts status", logger.Field{Key: "error", Value: err})
	}

	// Get automated security fixes (Dependabot)
	securityFixesEnabled, err := client.getAutomatedSecurityFixes(ctx, owner, repo)
	if err != nil {
		client.logger.Warn("Failed to get automated security fixes status", logger.Field{Key: "error", Value: err})
	}

	// Get secret scanning status
	secretScanningEnabled, err := client.getSecretScanning(ctx, owner, repo)
	if err != nil {
		client.logger.Warn("Failed to get secret scanning status", logger.Field{Key: "error", Value: err})
	}

	// Get code scanning status
	codeScanningEnabled, err := client.getCodeScanning(ctx, owner, repo)
	if err != nil {
		client.logger.Warn("Failed to get code scanning status", logger.Field{Key: "error", Value: err})
	}

	return &models.GitHubSecuritySettings{
		VulnerabilityAlertsEnabled:    alertsEnabled,
		AutomatedSecurityFixesEnabled: securityFixesEnabled,
		SecretScanningEnabled:         secretScanningEnabled,
		CodeScanningEnabled:           codeScanningEnabled,
	}, nil
}

// GetOrganizationMembers gets all organization members (if applicable)
func (client *GitHubClient) GetOrganizationMembers(ctx context.Context, org string) ([]models.GitHubOrgMember, error) {
	endpoint := fmt.Sprintf("/orgs/%s/members", org)

	resp, err := client.makeRESTRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var members []models.GitHubOrgMember
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		return nil, fmt.Errorf("failed to decode organization members response: %w", err)
	}

	return members, nil
}

// Helper methods for security settings

func (client *GitHubClient) getVulnerabilityAlerts(ctx context.Context, owner, repo string) (bool, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/vulnerability-alerts", owner, repo)

	resp, err := client.makeRESTRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusNoContent, nil
}

func (client *GitHubClient) getAutomatedSecurityFixes(ctx context.Context, owner, repo string) (bool, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/automated-security-fixes", owner, repo)

	resp, err := client.makeRESTRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var result struct {
		Enabled bool `json:"enabled"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, err
	}

	return result.Enabled, nil
}

func (client *GitHubClient) getSecretScanning(ctx context.Context, owner, repo string) (bool, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s", owner, repo)

	resp, err := client.makeRESTRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return false, nil
	}

	var repoInfo struct {
		SecurityAndAnalysis struct {
			SecretScanning struct {
				Status string `json:"status"`
			} `json:"secret_scanning"`
		} `json:"security_and_analysis"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&repoInfo); err != nil {
		return false, err
	}

	return repoInfo.SecurityAndAnalysis.SecretScanning.Status == "enabled", nil
}

func (client *GitHubClient) getCodeScanning(ctx context.Context, owner, repo string) (bool, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/code-scanning/alerts", owner, repo)

	resp, err := client.makeRESTRequest(ctx, "GET", endpoint+"?per_page=1", nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	// If we get 200 or 204, code scanning is enabled
	// If we get 404, code scanning is not enabled
	return resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusNoContent, nil
}

// GenerateCorrelationID generates a unique correlation ID for tracking
func GenerateCorrelationID() string {
	// Use timestamp and simple hash for correlation ID
	return fmt.Sprintf("github-%d", time.Now().UnixNano())
}
