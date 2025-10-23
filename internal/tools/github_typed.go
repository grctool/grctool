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
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools/types"
)

// TypedGitHubTool provides GitHub search capabilities with typed requests/responses
type TypedGitHubTool struct {
	*GitHubTool // Embed the original tool for compatibility
}

// NewTypedGitHubTool creates a new TypedGitHubTool
func NewTypedGitHubTool(cfg *config.Config, log logger.Logger) *TypedGitHubTool {
	return &TypedGitHubTool{
		GitHubTool: NewGitHubTool(cfg, log).(*GitHubTool),
	}
}

// ExecuteTyped runs the GitHub search tool with a typed request
func (gt *TypedGitHubTool) ExecuteTyped(ctx context.Context, req types.Request) (types.Response, error) {
	// Type assert to GitHubRequest
	githubReq, ok := req.(*types.GitHubRequest)
	if !ok {
		return types.NewErrorResponse(gt.Name(), "invalid request type, expected GitHubRequest", nil), nil
	}

	gt.logger.Debug("Executing GitHub search with typed request",
		logger.Field{Key: "query", Value: githubReq.Query},
		logger.Field{Key: "labels", Value: githubReq.Labels},
		logger.Field{Key: "include_closed", Value: githubReq.IncludeClosed},
		logger.Field{Key: "repository", Value: githubReq.Repository})

	// Use repository from request if provided, otherwise use config
	repository := githubReq.Repository
	if repository == "" {
		repository = gt.config.Repository
	}

	// Build search query
	query := githubReq.Query
	if !githubReq.IncludeClosed {
		query += " is:open"
	}

	// Search for issues based on the search type
	var issues []models.GitHubIssueResult
	var err error

	switch githubReq.SearchType {
	case "issues", "":
		issues, err = gt.searchIssuesTyped(ctx, repository, query, githubReq.Labels, githubReq.MaxResults)
		if err != nil {
			return types.NewErrorResponse(gt.Name(), fmt.Sprintf("failed to search GitHub issues: %v", err), nil), nil
		}
	default:
		return types.NewErrorResponse(gt.Name(), fmt.Sprintf("unsupported search type: %s", githubReq.SearchType), nil), nil
	}

	// Generate report
	report := gt.generateReport(issues)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "github",
		Resource:    fmt.Sprintf("GitHub repository: %s", repository),
		Content:     report,
		Relevance:   gt.calculateOverallRelevance(issues),
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"repository":  repository,
			"issue_count": len(issues),
			"query":       githubReq.Query,
			"labels":      githubReq.Labels,
			"search_type": githubReq.SearchType,
		},
	}

	// Create typed response
	response := &types.GitHubResponse{
		ToolResponse: types.NewSuccessResponse(gt.Name(), report, source, map[string]interface{}{
			"request_type": "GitHubRequest",
		}),
		Issues:     issues,
		TotalCount: len(issues),
		Query:      githubReq.Query,
		SearchType: githubReq.SearchType,
		Repository: repository,
	}

	return response, nil
}

// searchIssuesTyped searches for GitHub issues with typed parameters
func (gt *TypedGitHubTool) searchIssuesTyped(ctx context.Context, repository, query string, labels []string, maxResults int) ([]models.GitHubIssueResult, error) {
	// Build search query
	searchQuery := fmt.Sprintf("repo:%s %s", repository, query)

	if len(labels) > 0 {
		for _, label := range labels {
			searchQuery += fmt.Sprintf(" label:\"%s\"", label)
		}
	}

	// Make API request with pagination support
	// Note: maxResults is used to limit results after the API call
	// perPage could be used for pagination in future enhancements

	// Use the existing search functionality from the embedded GitHubTool
	// This is a simplified approach - in practice, you might want to refactor the HTTP logic
	issues, err := gt.SearchSecurityIssues(ctx, query, labels)
	if err != nil {
		return nil, err
	}

	// Limit results if maxResults is specified
	if maxResults > 0 && len(issues) > maxResults {
		issues = issues[:maxResults]
	}

	return issues, nil
}

// UpdateClaudeToolDefinition returns an updated tool definition that supports typed parameters
func (gt *TypedGitHubTool) GetClaudeToolDefinition() models.ClaudeTool {
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
				"repository": map[string]interface{}{
					"type":        "string",
					"description": "GitHub repository to search (e.g., 'owner/repo'). Uses configured repository if not provided.",
				},
				"search_type": map[string]interface{}{
					"type":        "string",
					"description": "Type of items to search for",
					"enum":        []string{"issues", "prs", "commits"},
					"default":     "issues",
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return (1-100)",
					"minimum":     1,
					"maximum":     100,
					"default":     50,
				},
			},
			"required": []string{"query"},
		},
	}
}

// Ensure TypedGitHubTool implements both Tool and types.Tool interfaces
var _ Tool = (*TypedGitHubTool)(nil)
var _ types.TypedTool = (*TypedGitHubTool)(nil)
