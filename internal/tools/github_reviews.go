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
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/auth"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/transport"
)

// GitHubReviewAnalyzer provides GitHub pull request review analysis capabilities
type GitHubReviewAnalyzer struct {
	config       *config.GitHubToolConfig
	httpClient   *http.Client
	logger       logger.Logger
	cacheDir     string
	authProvider auth.AuthProvider
}

// NewGitHubReviewAnalyzer creates a new GitHub PR review analyzer
func NewGitHubReviewAnalyzer(cfg *config.Config, log logger.Logger) Tool {
	// Create HTTP transport with logging
	httpTransport := http.DefaultTransport
	httpTransport = transport.NewLoggingTransport(httpTransport, log.WithComponent("github-review-api"))

	// Set up cache directory
	cacheDir := filepath.Join(cfg.Storage.DataDir, "github_cache", "reviews")

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

	return &GitHubReviewAnalyzer{
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
func (gra *GitHubReviewAnalyzer) Name() string {
	return "github-review-analyzer"
}

// Description returns the tool description
func (gra *GitHubReviewAnalyzer) Description() string {
	return "Analyze GitHub pull request reviews, approval processes, and code review compliance for SOC2 evidence"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (gra *GitHubReviewAnalyzer) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        gra.Name(),
		Description: gra.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"analysis_period": map[string]interface{}{
					"type":        "string",
					"description": "Time period for analysis: 30d, 90d, 180d, 1y",
					"enum":        []string{"30d", "90d", "180d", "1y"},
					"default":     "90d",
				},
				"state": map[string]interface{}{
					"type":        "string",
					"description": "PR state to analyze: open, closed, merged, all",
					"enum":        []string{"open", "closed", "merged", "all"},
					"default":     "all",
				},
				"include_security_prs": map[string]interface{}{
					"type":        "boolean",
					"description": "Focus on security-related pull requests",
					"default":     true,
				},
				"detailed_metrics": map[string]interface{}{
					"type":        "boolean",
					"description": "Include detailed reviewer statistics and patterns",
					"default":     true,
				},
				"check_compliance": map[string]interface{}{
					"type":        "boolean",
					"description": "Check compliance with review policies",
					"default":     true,
				},
				"max_prs": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of PRs to analyze",
					"minimum":     10,
					"maximum":     1000,
					"default":     200,
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

// Execute runs the GitHub PR review analysis
func (gra *GitHubReviewAnalyzer) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	startTime := time.Now()
	correlationID := GenerateCorrelationID()

	gra.logger.Debug("Executing GitHub review analyzer",
		logger.String("correlation_id", correlationID),
		logger.Field{Key: "params", Value: params})

	// Check authentication
	authStatus := gra.authProvider.GetStatus(ctx)
	if gra.authProvider.IsAuthRequired() && !authStatus.Authenticated {
		if err := gra.authProvider.Authenticate(ctx); err != nil {
			gra.logger.Warn("GitHub authentication failed, will use cached data only",
				logger.Field{Key: "error", Value: err})
		}
		// Note: authStatus will be refreshed later when needed
	}

	// Extract parameters
	analysisPeriod := "90d"
	if ap, ok := params["analysis_period"].(string); ok {
		analysisPeriod = ap
	}

	state := "all"
	if s, ok := params["state"].(string); ok {
		state = s
	}

	includeSecurityPRs := true
	if isp, ok := params["include_security_prs"].(bool); ok {
		includeSecurityPRs = isp
	}

	detailedMetrics := true
	if dm, ok := params["detailed_metrics"].(bool); ok {
		detailedMetrics = dm
	}

	checkCompliance := true
	if cc, ok := params["check_compliance"].(bool); ok {
		checkCompliance = cc
	}

	maxPRs := 200
	if mp, ok := params["max_prs"].(int); ok {
		maxPRs = mp
	}

	useCache := true
	if uc, ok := params["use_cache"].(bool); ok {
		useCache = uc
	}

	// Perform analysis
	analysis, err := gra.analyzePullRequests(ctx, analysisPeriod, state, includeSecurityPRs, detailedMetrics, checkCompliance, maxPRs, useCache)
	if err != nil {
		return "", nil, fmt.Errorf("failed to analyze pull requests: %w", err)
	}

	// Generate report
	report := gra.generateReviewReport(analysis, detailedMetrics)

	// Get final auth status
	finalAuthStatus := gra.authProvider.GetStatus(ctx)
	duration := time.Since(startTime)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "github-review-analyzer",
		Resource:    fmt.Sprintf("GitHub PR reviews: %s", gra.config.Repository),
		Content:     report,
		Relevance:   gra.calculateReviewRelevance(analysis),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"repository":       gra.config.Repository,
			"analysis_period":  analysisPeriod,
			"pr_count":         len(analysis.PullRequests),
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

// analyzePullRequests performs comprehensive PR review analysis
func (gra *GitHubReviewAnalyzer) analyzePullRequests(ctx context.Context, period, state string, includeSecurityPRs, detailedMetrics, checkCompliance bool, maxPRs int, useCache bool) (*models.GitHubPullRequestAnalysis, error) {
	// Calculate date range
	endDate := time.Now()
	startDate := gra.calculateStartDate(period, endDate)

	analysis := &models.GitHubPullRequestAnalysis{
		Repository:   gra.config.Repository,
		AnalysisDate: time.Now(),
		DateRange: models.GitHubDateRange{
			StartDate: startDate,
			EndDate:   endDate,
			Period:    period,
		},
	}

	// Get pull requests
	prs, err := gra.getPullRequests(ctx, state, startDate, maxPRs, useCache)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull requests: %w", err)
	}

	// Filter for security PRs if requested
	if includeSecurityPRs {
		prs = gra.filterSecurityPRs(prs)
	}

	// Get detailed review information for each PR
	for i := range prs {
		if err := gra.enrichPRWithReviews(ctx, &prs[i]); err != nil {
			gra.logger.Warn("Failed to enrich PR with review details",
				logger.Int("pr_number", prs[i].Number),
				logger.Field{Key: "error", Value: err})
		}

		// Check compliance for each PR
		if checkCompliance {
			prs[i].ComplianceStatus = gra.assessPRCompliance(prs[i])
		}

		// Calculate security relevance
		prs[i].SecurityRelevant = gra.isSecurityRelevant(prs[i])
		prs[i].Relevance = gra.calculatePRRelevance(prs[i])
	}

	analysis.PullRequests = prs

	// Calculate metrics
	if detailedMetrics {
		analysis.ReviewMetrics = gra.calculateReviewMetrics(prs)
		analysis.ApprovalPatterns = gra.analyzeApprovalPatterns(prs)
	}

	// Generate recommendations
	analysis.Recommendations = gra.generateRecommendations(analysis)

	// Calculate overall compliance score
	analysis.ComplianceScore = gra.calculateOverallComplianceScore(prs)

	return analysis, nil
}

// getPullRequests retrieves pull requests from GitHub API
func (gra *GitHubReviewAnalyzer) getPullRequests(ctx context.Context, state string, since time.Time, maxPRs int, useCache bool) ([]models.GitHubPullRequestDetail, error) {
	var allPRs []models.GitHubPullRequestDetail
	page := 1
	perPage := 100

	for len(allPRs) < maxPRs {
		// Build URL
		url := fmt.Sprintf("https://api.github.com/repos/%s/pulls?state=%s&sort=updated&direction=desc&page=%d&per_page=%d",
			gra.config.Repository, state, page, perPage)

		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Accept", "application/vnd.github.v3+json")
		if gra.config.APIToken != "" {
			req.Header.Set("Authorization", "token "+gra.config.APIToken)
		}

		resp, err := gra.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("failed to make request: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
		}

		var pagePRs []struct {
			Number    int               `json:"number"`
			Title     string            `json:"title"`
			State     string            `json:"state"`
			User      models.GitHubUser `json:"user"`
			CreatedAt time.Time         `json:"created_at"`
			UpdatedAt time.Time         `json:"updated_at"`
			MergedAt  *time.Time        `json:"merged_at"`
			ClosedAt  *time.Time        `json:"closed_at"`
			HTMLURL   string            `json:"html_url"`
			Head      struct {
				Ref string `json:"ref"`
				SHA string `json:"sha"`
			} `json:"head"`
			Base struct {
				Ref string `json:"ref"`
				SHA string `json:"sha"`
			} `json:"base"`
			RequestedReviewers []models.GitHubUser `json:"requested_reviewers"`
			RequestedTeams     []models.GitHubTeam `json:"requested_teams"`
			Labels             []struct {
				Name string `json:"name"`
			} `json:"labels"`
			ChangedFiles int `json:"changed_files"`
			Additions    int `json:"additions"`
			Deletions    int `json:"deletions"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&pagePRs); err != nil {
			return nil, fmt.Errorf("failed to parse response: %w", err)
		}

		if len(pagePRs) == 0 {
			break // No more PRs
		}

		// Convert and filter by date
		for _, pr := range pagePRs {
			if pr.CreatedAt.Before(since) {
				continue // Skip PRs outside date range
			}

			// Extract labels
			var labels []string
			for _, label := range pr.Labels {
				labels = append(labels, label.Name)
			}

			prDetail := models.GitHubPullRequestDetail{
				Number:             pr.Number,
				Title:              pr.Title,
				State:              pr.State,
				Author:             pr.User,
				CreatedAt:          pr.CreatedAt,
				UpdatedAt:          pr.UpdatedAt,
				MergedAt:           pr.MergedAt,
				ClosedAt:           pr.ClosedAt,
				Labels:             labels,
				RequestedReviewers: pr.RequestedReviewers,
				RequestedTeams:     pr.RequestedTeams,
				ChangedFiles:       pr.ChangedFiles,
				Additions:          pr.Additions,
				Deletions:          pr.Deletions,
				Branch: models.GitHubBranchInfo{
					Name: pr.Head.Ref,
					SHA:  pr.Head.SHA,
					Ref:  pr.Head.Ref,
				},
				BaseBranch: models.GitHubBranchInfo{
					Name: pr.Base.Ref,
					SHA:  pr.Base.SHA,
					Ref:  pr.Base.Ref,
				},
				HTMLURL: pr.HTMLURL,
			}

			allPRs = append(allPRs, prDetail)

			if len(allPRs) >= maxPRs {
				break
			}
		}

		page++
		if len(pagePRs) < perPage {
			break // Last page
		}
	}

	return allPRs, nil
}

// enrichPRWithReviews adds detailed review information to a PR
func (gra *GitHubReviewAnalyzer) enrichPRWithReviews(ctx context.Context, pr *models.GitHubPullRequestDetail) error {
	// Get reviews
	reviews, err := gra.getPRReviews(ctx, pr.Number)
	if err != nil {
		return fmt.Errorf("failed to get PR reviews: %w", err)
	}
	pr.Reviews = reviews

	// Get status checks
	statusChecks, err := gra.getPRStatusChecks(ctx, pr.Number, pr.Branch.SHA)
	if err != nil {
		gra.logger.Warn("Failed to get status checks",
			logger.Int("pr_number", pr.Number),
			logger.Field{Key: "error", Value: err})
	} else {
		pr.StatusChecks = statusChecks
	}

	// Build approval timeline
	pr.ApprovalTimeline = gra.buildApprovalTimeline(pr)

	return nil
}

// getPRReviews gets reviews for a specific PR
func (gra *GitHubReviewAnalyzer) getPRReviews(ctx context.Context, prNumber int) ([]models.GitHubPRReview, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/pulls/%d/reviews", gra.config.Repository, prNumber)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if gra.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+gra.config.APIToken)
	}

	resp, err := gra.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API error %d: %s", resp.StatusCode, string(body))
	}

	var reviews []models.GitHubPRReview
	if err := json.NewDecoder(resp.Body).Decode(&reviews); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return reviews, nil
}

// getPRStatusChecks gets status checks for a specific commit
func (gra *GitHubReviewAnalyzer) getPRStatusChecks(ctx context.Context, prNumber int, sha string) ([]models.GitHubStatusCheck, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s/check-runs", gra.config.Repository, sha)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if gra.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+gra.config.APIToken)
	}

	resp, err := gra.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		// Try status API instead
		return gra.getPRCommitStatus(ctx, sha)
	}

	var checkRuns struct {
		CheckRuns []struct {
			Name        string     `json:"name"`
			Status      string     `json:"status"`
			Conclusion  string     `json:"conclusion"`
			HTMLURL     string     `json:"html_url"`
			StartedAt   time.Time  `json:"started_at"`
			CompletedAt *time.Time `json:"completed_at"`
		} `json:"check_runs"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&checkRuns); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var statusChecks []models.GitHubStatusCheck
	for _, check := range checkRuns.CheckRuns {
		statusCheck := models.GitHubStatusCheck{
			Name:        check.Name,
			Status:      check.Status,
			Conclusion:  check.Conclusion,
			URL:         check.HTMLURL,
			CreatedAt:   check.StartedAt,
			UpdatedAt:   check.StartedAt,
			CompletedAt: check.CompletedAt,
		}
		statusChecks = append(statusChecks, statusCheck)
	}

	return statusChecks, nil
}

// getPRCommitStatus gets commit status (fallback for older status API)
func (gra *GitHubReviewAnalyzer) getPRCommitStatus(ctx context.Context, sha string) ([]models.GitHubStatusCheck, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/commits/%s/status", gra.config.Repository, sha)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if gra.config.APIToken != "" {
		req.Header.Set("Authorization", "token "+gra.config.APIToken)
	}

	resp, err := gra.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return []models.GitHubStatusCheck{}, nil // Return empty if status not available
	}

	var status struct {
		State    string `json:"state"`
		Statuses []struct {
			Context     string    `json:"context"`
			State       string    `json:"state"`
			Description string    `json:"description"`
			TargetURL   string    `json:"target_url"`
			CreatedAt   time.Time `json:"created_at"`
			UpdatedAt   time.Time `json:"updated_at"`
		} `json:"statuses"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&status); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	var statusChecks []models.GitHubStatusCheck
	for _, s := range status.Statuses {
		statusCheck := models.GitHubStatusCheck{
			Name:        s.Context,
			Status:      s.State,
			Conclusion:  s.State,
			URL:         s.TargetURL,
			Description: s.Description,
			CreatedAt:   s.CreatedAt,
			UpdatedAt:   s.UpdatedAt,
		}
		statusChecks = append(statusChecks, statusCheck)
	}

	return statusChecks, nil
}

// buildApprovalTimeline creates a timeline of approval events
func (gra *GitHubReviewAnalyzer) buildApprovalTimeline(pr *models.GitHubPullRequestDetail) []models.GitHubApprovalEvent {
	var timeline []models.GitHubApprovalEvent

	// Add review request events
	for _, reviewer := range pr.RequestedReviewers {
		timeline = append(timeline, models.GitHubApprovalEvent{
			Type:      "review_requested",
			Actor:     reviewer,
			Timestamp: pr.CreatedAt, // Approximate - real implementation would get from events API
			Action:    "requested",
		})
	}

	// Add review events
	for _, review := range pr.Reviews {
		timeline = append(timeline, models.GitHubApprovalEvent{
			Type:        "reviewed",
			Actor:       review.User,
			Timestamp:   review.SubmittedAt,
			Action:      "submitted",
			ReviewState: review.State,
		})
	}

	// Sort by timestamp
	sort.Slice(timeline, func(i, j int) bool {
		return timeline[i].Timestamp.Before(timeline[j].Timestamp)
	})

	return timeline
}

// Helper methods

func (gra *GitHubReviewAnalyzer) calculateStartDate(period string, endDate time.Time) time.Time {
	switch period {
	case "30d":
		return endDate.AddDate(0, 0, -30)
	case "90d":
		return endDate.AddDate(0, 0, -90)
	case "180d":
		return endDate.AddDate(0, 0, -180)
	case "1y":
		return endDate.AddDate(-1, 0, 0)
	default:
		return endDate.AddDate(0, 0, -90) // Default to 90 days
	}
}

func (gra *GitHubReviewAnalyzer) filterSecurityPRs(prs []models.GitHubPullRequestDetail) []models.GitHubPullRequestDetail {
	var securityPRs []models.GitHubPullRequestDetail

	securityKeywords := []string{"security", "vulnerability", "cve", "fix", "patch", "auth", "encrypt", "ssl", "tls"}

	for _, pr := range prs {
		isSecurityPR := false

		// Check title and labels
		titleLower := strings.ToLower(pr.Title)
		for _, keyword := range securityKeywords {
			if strings.Contains(titleLower, keyword) {
				isSecurityPR = true
				break
			}
		}

		// Check labels
		if !isSecurityPR {
			for _, label := range pr.Labels {
				labelLower := strings.ToLower(label)
				for _, keyword := range securityKeywords {
					if strings.Contains(labelLower, keyword) {
						isSecurityPR = true
						break
					}
				}
				if isSecurityPR {
					break
				}
			}
		}

		if isSecurityPR {
			securityPRs = append(securityPRs, pr)
		}
	}

	return securityPRs
}

func (gra *GitHubReviewAnalyzer) isSecurityRelevant(pr models.GitHubPullRequestDetail) bool {
	securityKeywords := []string{"security", "vulnerability", "cve", "auth", "encrypt", "ssl", "tls"}

	titleLower := strings.ToLower(pr.Title)
	for _, keyword := range securityKeywords {
		if strings.Contains(titleLower, keyword) {
			return true
		}
	}

	for _, label := range pr.Labels {
		labelLower := strings.ToLower(label)
		for _, keyword := range securityKeywords {
			if strings.Contains(labelLower, keyword) {
				return true
			}
		}
	}

	return false
}

func (gra *GitHubReviewAnalyzer) assessPRCompliance(pr models.GitHubPullRequestDetail) models.GitHubPRComplianceStatus {
	status := models.GitHubPRComplianceStatus{}

	// Count approvals
	approvals := 0
	for _, review := range pr.Reviews {
		if review.State == "APPROVED" {
			approvals++
		}
	}

	// Simple compliance checks
	status.RequiredReviewsMet = approvals >= 1 // Assume minimum 1 approval required
	status.StatusChecksPassed = gra.allStatusChecksPassed(pr.StatusChecks)
	status.BranchUpToDate = true        // Would need to check against base branch
	status.ConversationsResolved = true // Would need to check conversation API

	// Calculate compliance score
	totalChecks := 4
	passedChecks := 0
	if status.RequiredReviewsMet {
		passedChecks++
	}
	if status.StatusChecksPassed {
		passedChecks++
	}
	if status.BranchUpToDate {
		passedChecks++
	}
	if status.ConversationsResolved {
		passedChecks++
	}

	status.ComplianceScore = float64(passedChecks) / float64(totalChecks)

	// Generate rule violations/satisfactions
	if !status.RequiredReviewsMet {
		status.ViolatedRules = append(status.ViolatedRules, "Required reviews not met")
	} else {
		status.SatisfiedRules = append(status.SatisfiedRules, "Required reviews met")
	}

	if !status.StatusChecksPassed {
		status.ViolatedRules = append(status.ViolatedRules, "Status checks failed")
	} else {
		status.SatisfiedRules = append(status.SatisfiedRules, "Status checks passed")
	}

	return status
}

func (gra *GitHubReviewAnalyzer) allStatusChecksPassed(checks []models.GitHubStatusCheck) bool {
	for _, check := range checks {
		if check.Conclusion != "success" && check.Status != "success" {
			return false
		}
	}
	return true
}

func (gra *GitHubReviewAnalyzer) calculatePRRelevance(pr models.GitHubPullRequestDetail) float64 {
	relevance := 0.5 // Base relevance

	// Security PRs get higher relevance
	if pr.SecurityRelevant {
		relevance += 0.3
	}

	// Recently updated PRs get bonus
	if time.Since(pr.UpdatedAt) < 30*24*time.Hour {
		relevance += 0.1
	}

	// PRs with reviews get bonus
	if len(pr.Reviews) > 0 {
		relevance += 0.1
	}

	// High compliance gets bonus
	if pr.ComplianceStatus.ComplianceScore > 0.8 {
		relevance += 0.1
	}

	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}

func (gra *GitHubReviewAnalyzer) calculateReviewMetrics(prs []models.GitHubPullRequestDetail) models.GitHubReviewMetrics {
	metrics := models.GitHubReviewMetrics{
		TotalPRs:            len(prs),
		ReviewParticipation: make(map[string]int),
		ReviewerStats:       make(map[string]models.GitHubReviewerStats),
		TimeBasedMetrics: models.GitHubTimeBasedMetrics{
			PRsByMonth:         make(map[string]int),
			ReviewsByMonth:     make(map[string]int),
			ApprovalsByMonth:   make(map[string]int),
			AverageTimeToMerge: make(map[string]time.Duration),
		},
	}

	totalReviewTime := time.Duration(0)
	totalApprovalTime := time.Duration(0)
	totalReviews := 0
	securityPRs := 0
	compliantPRs := 0

	for _, pr := range prs {
		// Count by state
		switch pr.State {
		case "open":
			metrics.OpenPRs++
		case "closed":
			if pr.MergedAt != nil {
				metrics.MergedPRs++
			} else {
				metrics.ClosedPRs++
			}
		}

		// Count security PRs
		if pr.SecurityRelevant {
			securityPRs++
		}

		// Count compliant PRs
		if pr.ComplianceStatus.ComplianceScore > 0.8 {
			compliantPRs++
		}

		// Count reviews and participation
		for _, review := range pr.Reviews {
			totalReviews++
			metrics.ReviewParticipation[review.User.Login]++

			// Calculate review time (simplified)
			reviewTime := review.SubmittedAt.Sub(pr.CreatedAt)
			totalReviewTime += reviewTime

			if review.State == "APPROVED" {
				totalApprovalTime += reviewTime
			}

			// Update reviewer stats
			stats := metrics.ReviewerStats[review.User.Login]
			stats.ReviewsGiven++
			switch review.State {
			case "APPROVED":
				stats.ApprovalsGiven++
			case "CHANGES_REQUESTED":
				stats.ChangeRequestsGiven++
			}
			metrics.ReviewerStats[review.User.Login] = stats
		}

		// Time-based metrics
		month := pr.CreatedAt.Format("2006-01")
		metrics.TimeBasedMetrics.PRsByMonth[month]++
		metrics.TimeBasedMetrics.ReviewsByMonth[month] += len(pr.Reviews)

		approvals := 0
		for _, review := range pr.Reviews {
			if review.State == "APPROVED" {
				approvals++
			}
		}
		metrics.TimeBasedMetrics.ApprovalsByMonth[month] += approvals
	}

	// Calculate averages
	if totalReviews > 0 {
		metrics.AverageReviewTime = totalReviewTime / time.Duration(totalReviews)
		metrics.AverageReviewsPerPR = float64(totalReviews) / float64(len(prs))
	}

	if metrics.MergedPRs > 0 {
		metrics.AverageApprovalTime = totalApprovalTime / time.Duration(metrics.MergedPRs)
	}

	// Calculate rates
	if len(prs) > 0 {
		metrics.ComplianceRate = float64(compliantPRs) / float64(len(prs))
	}

	metrics.SecurityPRs = securityPRs

	return metrics
}

func (gra *GitHubReviewAnalyzer) analyzeApprovalPatterns(prs []models.GitHubPullRequestDetail) models.GitHubApprovalPatterns {
	patterns := models.GitHubApprovalPatterns{
		BotUsage:              make(map[string]int),
		RequiredCheckPatterns: make(map[string]int),
	}

	approverCounts := make(map[string]int)

	for _, pr := range prs {
		// Count common approvers
		for _, review := range pr.Reviews {
			if review.State == "APPROVED" {
				approverCounts[review.User.Login]++
			}

			// Count bot usage
			if review.User.Type == "Bot" {
				patterns.BotUsage[review.User.Login]++
			}
		}

		// Count required check patterns
		for _, check := range pr.StatusChecks {
			patterns.RequiredCheckPatterns[check.Name]++
		}
	}

	// Convert top approvers
	type approverCount struct {
		User  models.GitHubUser
		Count int
	}

	var approvers []approverCount
	for login, count := range approverCounts {
		approvers = append(approvers, approverCount{
			User:  models.GitHubUser{Login: login},
			Count: count,
		})
	}

	// Sort by count
	sort.Slice(approvers, func(i, j int) bool {
		return approvers[i].Count > approvers[j].Count
	})

	// Take top 10
	for i, approver := range approvers {
		if i >= 10 {
			break
		}
		patterns.CommonApprovers = append(patterns.CommonApprovers, approver.User)
	}

	return patterns
}

func (gra *GitHubReviewAnalyzer) generateRecommendations(analysis *models.GitHubPullRequestAnalysis) []models.GitHubRecommendation {
	var recommendations []models.GitHubRecommendation

	// Low compliance rate recommendation
	if analysis.ComplianceScore < 0.8 {
		recommendations = append(recommendations, models.GitHubRecommendation{
			Type:        "compliance",
			Priority:    "high",
			Title:       "Improve PR Compliance Rate",
			Description: fmt.Sprintf("Current compliance rate is %.1f%%. Consider strengthening review requirements.", analysis.ComplianceScore*100),
			Evidence:    fmt.Sprintf("Only %.1f%% of PRs meet all compliance requirements", analysis.ComplianceScore*100),
			ActionItems: []string{
				"Enforce required reviews before merge",
				"Require status checks to pass",
				"Enable branch protection rules",
			},
			Impact: "Improved code quality and security posture",
		})
	}

	// Security PR review recommendation
	securityPRRate := float64(analysis.ReviewMetrics.SecurityPRs) / float64(analysis.ReviewMetrics.TotalPRs)
	if securityPRRate > 0.3 {
		recommendations = append(recommendations, models.GitHubRecommendation{
			Type:        "security",
			Priority:    "medium",
			Title:       "Enhanced Security PR Review Process",
			Description: fmt.Sprintf("%.1f%% of PRs are security-related. Consider special review requirements.", securityPRRate*100),
			Evidence:    fmt.Sprintf("%d out of %d PRs contain security changes", analysis.ReviewMetrics.SecurityPRs, analysis.ReviewMetrics.TotalPRs),
			ActionItems: []string{
				"Require security team review for security PRs",
				"Implement additional security testing",
				"Create security PR templates",
			},
			Impact: "Reduced security vulnerabilities in production",
		})
	}

	// Review participation recommendation
	if len(analysis.ReviewMetrics.ReviewParticipation) < 3 {
		recommendations = append(recommendations, models.GitHubRecommendation{
			Type:        "efficiency",
			Priority:    "medium",
			Title:       "Increase Review Participation",
			Description: "Limited number of team members participating in code reviews.",
			Evidence:    fmt.Sprintf("Only %d reviewers actively participating", len(analysis.ReviewMetrics.ReviewParticipation)),
			ActionItems: []string{
				"Distribute review load across team",
				"Implement code owner requirements",
				"Provide review training",
			},
			Impact: "Better knowledge sharing and reduced bus factor",
		})
	}

	return recommendations
}

func (gra *GitHubReviewAnalyzer) calculateOverallComplianceScore(prs []models.GitHubPullRequestDetail) float64 {
	if len(prs) == 0 {
		return 0.0
	}

	totalScore := 0.0
	for _, pr := range prs {
		totalScore += pr.ComplianceStatus.ComplianceScore
	}

	return totalScore / float64(len(prs))
}

func (gra *GitHubReviewAnalyzer) calculateReviewRelevance(analysis *models.GitHubPullRequestAnalysis) float64 {
	relevance := 0.5

	// High compliance gets bonus
	if analysis.ComplianceScore > 0.8 {
		relevance += 0.3
	}

	// Security PRs get bonus
	if analysis.ReviewMetrics.SecurityPRs > 0 {
		relevance += 0.2
	}

	// Recent data gets bonus
	if analysis.DateRange.Period == "30d" || analysis.DateRange.Period == "90d" {
		relevance += 0.1
	}

	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}

func (gra *GitHubReviewAnalyzer) generateReviewReport(analysis *models.GitHubPullRequestAnalysis, detailedMetrics bool) string {
	var report strings.Builder

	report.WriteString("# GitHub Pull Request Review Analysis\n\n")
	report.WriteString(fmt.Sprintf("**Repository**: %s\n", analysis.Repository))
	report.WriteString(fmt.Sprintf("**Analysis Date**: %s\n", analysis.AnalysisDate.Format("2006-01-02 15:04:05")))
	report.WriteString(fmt.Sprintf("**Analysis Period**: %s (%s to %s)\n",
		analysis.DateRange.Period,
		analysis.DateRange.StartDate.Format("2006-01-02"),
		analysis.DateRange.EndDate.Format("2006-01-02")))
	report.WriteString(fmt.Sprintf("**Overall Compliance Score**: %.2f\n\n", analysis.ComplianceScore))

	// Executive Summary
	report.WriteString("## Executive Summary\n\n")
	report.WriteString(fmt.Sprintf("- **Total Pull Requests**: %d\n", len(analysis.PullRequests)))
	report.WriteString(fmt.Sprintf("- **Merged PRs**: %d\n", analysis.ReviewMetrics.MergedPRs))
	report.WriteString(fmt.Sprintf("- **Open PRs**: %d\n", analysis.ReviewMetrics.OpenPRs))
	report.WriteString(fmt.Sprintf("- **Security-Related PRs**: %d (%.1f%%)\n",
		analysis.ReviewMetrics.SecurityPRs,
		float64(analysis.ReviewMetrics.SecurityPRs)/float64(len(analysis.PullRequests))*100))
	report.WriteString(fmt.Sprintf("- **Compliance Rate**: %.1f%%\n", analysis.ComplianceScore*100))
	report.WriteString(fmt.Sprintf("- **Average Reviews per PR**: %.1f\n", analysis.ReviewMetrics.AverageReviewsPerPR))
	report.WriteString("\n")

	// Review Metrics
	if detailedMetrics {
		report.WriteString("## Review Metrics\n\n")
		report.WriteString(fmt.Sprintf("- **Average Review Time**: %s\n", analysis.ReviewMetrics.AverageReviewTime.Round(time.Hour)))
		report.WriteString(fmt.Sprintf("- **Average Approval Time**: %s\n", analysis.ReviewMetrics.AverageApprovalTime.Round(time.Hour)))
		report.WriteString(fmt.Sprintf("- **Active Reviewers**: %d\n", len(analysis.ReviewMetrics.ReviewParticipation)))

		if len(analysis.ReviewMetrics.ReviewParticipation) > 0 {
			report.WriteString("\n### Top Reviewers\n")
			type reviewerCount struct {
				login string
				count int
			}
			var reviewers []reviewerCount
			for login, count := range analysis.ReviewMetrics.ReviewParticipation {
				reviewers = append(reviewers, reviewerCount{login, count})
			}
			sort.Slice(reviewers, func(i, j int) bool {
				return reviewers[i].count > reviewers[j].count
			})

			for i, reviewer := range reviewers {
				if i >= 5 { // Top 5
					break
				}
				report.WriteString(fmt.Sprintf("- **%s**: %d reviews\n", reviewer.login, reviewer.count))
			}
		}
		report.WriteString("\n")
	}

	// Compliance Analysis
	report.WriteString("## Compliance Analysis\n\n")
	compliantPRs := 0
	for _, pr := range analysis.PullRequests {
		if pr.ComplianceStatus.ComplianceScore >= 0.8 {
			compliantPRs++
		}
	}
	report.WriteString(fmt.Sprintf("- **Compliant PRs**: %d out of %d (%.1f%%)\n",
		compliantPRs, len(analysis.PullRequests),
		float64(compliantPRs)/float64(len(analysis.PullRequests))*100))

	// Common violations
	violationCounts := make(map[string]int)
	for _, pr := range analysis.PullRequests {
		for _, violation := range pr.ComplianceStatus.ViolatedRules {
			violationCounts[violation]++
		}
	}

	if len(violationCounts) > 0 {
		report.WriteString("\n### Common Compliance Violations\n")
		type violation struct {
			rule  string
			count int
		}
		var violations []violation
		for rule, count := range violationCounts {
			violations = append(violations, violation{rule, count})
		}
		sort.Slice(violations, func(i, j int) bool {
			return violations[i].count > violations[j].count
		})

		for _, v := range violations {
			report.WriteString(fmt.Sprintf("- **%s**: %d PRs (%.1f%%)\n",
				v.rule, v.count, float64(v.count)/float64(len(analysis.PullRequests))*100))
		}
	}
	report.WriteString("\n")

	// Security Analysis
	securityPRs := 0
	for _, pr := range analysis.PullRequests {
		if pr.SecurityRelevant {
			securityPRs++
		}
	}

	if securityPRs > 0 {
		report.WriteString("## Security Pull Requests\n\n")
		report.WriteString(fmt.Sprintf("- **Security PRs**: %d out of %d (%.1f%%)\n",
			securityPRs, len(analysis.PullRequests),
			float64(securityPRs)/float64(len(analysis.PullRequests))*100))

		report.WriteString("\n### Recent Security PRs\n")
		count := 0
		for _, pr := range analysis.PullRequests {
			if pr.SecurityRelevant && count < 5 {
				report.WriteString(fmt.Sprintf("- **PR #%d**: %s (%s)\n", pr.Number, pr.Title, pr.State))
				report.WriteString(fmt.Sprintf("  - Created: %s\n", pr.CreatedAt.Format("2006-01-02")))
				report.WriteString(fmt.Sprintf("  - Reviews: %d\n", len(pr.Reviews)))
				report.WriteString(fmt.Sprintf("  - Compliance: %.1f%%\n", pr.ComplianceStatus.ComplianceScore*100))
				count++
			}
		}
		report.WriteString("\n")
	}

	// Recommendations
	if len(analysis.Recommendations) > 0 {
		report.WriteString("## Recommendations\n\n")
		for _, rec := range analysis.Recommendations {
			report.WriteString(fmt.Sprintf("### %s (%s priority)\n", rec.Title, rec.Priority))
			report.WriteString(fmt.Sprintf("%s\n\n", rec.Description))
			report.WriteString(fmt.Sprintf("**Evidence**: %s\n\n", rec.Evidence))
			if len(rec.ActionItems) > 0 {
				report.WriteString("**Action Items**:\n")
				for _, item := range rec.ActionItems {
					report.WriteString(fmt.Sprintf("- %s\n", item))
				}
				report.WriteString("\n")
			}
			report.WriteString(fmt.Sprintf("**Expected Impact**: %s\n\n", rec.Impact))
			report.WriteString("---\n\n")
		}
	}

	return report.String()
}
