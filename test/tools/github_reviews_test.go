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

//go:build !e2e

package tools_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock GitHub API responses for PR review analysis
var mockPullRequestsResponse = []map[string]interface{}{
	{
		"number":     123,
		"title":      "Add security encryption feature",
		"state":      "closed",
		"merged_at":  "2024-01-15T10:30:00Z",
		"created_at": "2024-01-10T09:00:00Z",
		"updated_at": "2024-01-15T10:30:00Z",
		"user": map[string]interface{}{
			"login":    "developer1",
			"id":       12345,
			"type":     "User",
			"html_url": "https://github.com/developer1",
		},
		"head": map[string]interface{}{
			"ref": "feature/security-encryption",
			"sha": "abc123def",
		},
		"base": map[string]interface{}{
			"ref": "main",
			"sha": "def456ghi",
		},
		"html_url": "https://github.com/test/repo/pull/123",
		"labels": []map[string]interface{}{
			{"name": "security"},
			{"name": "enhancement"},
		},
		"requested_reviewers": []map[string]interface{}{
			{
				"login":    "reviewer1",
				"id":       67890,
				"type":     "User",
				"html_url": "https://github.com/reviewer1",
			},
		},
		"requested_teams": []map[string]interface{}{},
		"changed_files":   5,
		"additions":       150,
		"deletions":       25,
	},
	{
		"number":     124,
		"title":      "Update documentation",
		"state":      "open",
		"created_at": "2024-01-12T14:20:00Z",
		"updated_at": "2024-01-14T16:45:00Z",
		"user": map[string]interface{}{
			"login":    "developer2",
			"id":       23456,
			"type":     "User",
			"html_url": "https://github.com/developer2",
		},
		"head": map[string]interface{}{
			"ref": "docs/update-readme",
			"sha": "ghi789jkl",
		},
		"base": map[string]interface{}{
			"ref": "main",
			"sha": "jkl012mno",
		},
		"html_url": "https://github.com/test/repo/pull/124",
		"labels": []map[string]interface{}{
			{"name": "documentation"},
		},
		"requested_reviewers": []map[string]interface{}{},
		"requested_teams":     []map[string]interface{}{},
		"changed_files":       2,
		"additions":           45,
		"deletions":           10,
	},
}

var mockReviewsResponse = []map[string]interface{}{
	{
		"id":    456789,
		"state": "APPROVED",
		"user": map[string]interface{}{
			"login":    "reviewer1",
			"id":       67890,
			"type":     "User",
			"html_url": "https://github.com/reviewer1",
		},
		"body":               "LGTM! Good security implementation.",
		"submitted_at":       "2024-01-14T15:30:00Z",
		"commit_id":          "abc123def",
		"author_association": "MEMBER",
	},
	{
		"id":    456790,
		"state": "CHANGES_REQUESTED",
		"user": map[string]interface{}{
			"login":    "reviewer2",
			"id":       78901,
			"type":     "User",
			"html_url": "https://github.com/reviewer2",
		},
		"body":               "Please add more test coverage for the encryption module.",
		"submitted_at":       "2024-01-13T11:15:00Z",
		"commit_id":          "abc123def",
		"author_association": "COLLABORATOR",
	},
	{
		"id":    456791,
		"state": "APPROVED",
		"user": map[string]interface{}{
			"login":    "reviewer2",
			"id":       78901,
			"type":     "User",
			"html_url": "https://github.com/reviewer2",
		},
		"body":               "Tests added, looks good now!",
		"submitted_at":       "2024-01-15T09:45:00Z",
		"commit_id":          "abc123def",
		"author_association": "COLLABORATOR",
	},
}

var mockCheckRunsResponse = map[string]interface{}{
	"check_runs": []map[string]interface{}{
		{
			"name":         "Test Suite",
			"status":       "completed",
			"conclusion":   "success",
			"html_url":     "https://github.com/test/repo/runs/123",
			"started_at":   "2024-01-15T10:00:00Z",
			"completed_at": "2024-01-15T10:15:00Z",
		},
		{
			"name":         "Security Scan",
			"status":       "completed",
			"conclusion":   "success",
			"html_url":     "https://github.com/test/repo/runs/124",
			"started_at":   "2024-01-15T10:10:00Z",
			"completed_at": "2024-01-15T10:20:00Z",
		},
	},
}

func TestGitHubReviewAnalyzer_Execute_Success(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/test/repo/pulls":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockPullRequestsResponse)
		case r.URL.Path == "/repos/test/repo/pulls/123/reviews":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockReviewsResponse)
		case r.URL.Path == "/repos/test/repo/pulls/124/reviews":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{}) // No reviews for PR 124
		case r.URL.Path == "/repos/test/repo/commits/abc123def/check-runs":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockCheckRunsResponse)
		case r.URL.Path == "/repos/test/repo/commits/ghi789jkl/check-runs":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"check_runs": []map[string]interface{}{}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test configuration
	cfg := createTestConfigReviews(t)
	cfg.Evidence.Tools.GitHub.Repository = "test/repo"
	cfg.Evidence.Tools.GitHub.APIToken = "test-token"

	// Create logger
	log := createTestLoggerReviews(t)

	// Create tool
	tool := tools.NewGitHubReviewAnalyzer(cfg, log)
	require.NotNil(t, tool)

	// Test parameters
	params := map[string]interface{}{
		"analysis_period":      "90d",
		"state":                "all",
		"include_security_prs": true,
		"detailed_metrics":     true,
		"check_compliance":     true,
		"max_prs":              200,
		"use_cache":            false,
	}

	// Execute tool
	ctx := context.Background()
	result, evidenceSource, err := tool.Execute(ctx, params)

	// Verify results
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, evidenceSource)

	// Verify evidence source
	assert.Equal(t, "github-review-analyzer", evidenceSource.Type)
	assert.Contains(t, evidenceSource.Resource, "test/repo")
	assert.Greater(t, evidenceSource.Relevance, 0.0)
	assert.NotEmpty(t, evidenceSource.Content)

	// Verify metadata
	metadata := evidenceSource.Metadata
	assert.Contains(t, metadata, "repository")
	assert.Contains(t, metadata, "analysis_period")
	assert.Contains(t, metadata, "pr_count")
	assert.Contains(t, metadata, "compliance_score")
	assert.Contains(t, metadata, "correlation_id")
	assert.Equal(t, "test/repo", metadata["repository"])
	assert.Equal(t, "90d", metadata["analysis_period"])
}

func TestGitHubReviewAnalyzer_SecurityPRsOnly(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/test/repo/pulls":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockPullRequestsResponse)
		case r.URL.Path == "/repos/test/repo/pulls/123/reviews":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockReviewsResponse)
		case r.URL.Path == "/repos/test/repo/pulls/124/reviews":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		case r.URL.Path == "/repos/test/repo/commits/abc123def/check-runs":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockCheckRunsResponse)
		case r.URL.Path == "/repos/test/repo/commits/ghi789jkl/check-runs":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"check_runs": []map[string]interface{}{}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test configuration
	cfg := createTestConfigReviews(t)
	cfg.Evidence.Tools.GitHub.Repository = "test/repo"
	cfg.Evidence.Tools.GitHub.APIToken = "test-token"

	// Create logger
	log := createTestLoggerReviews(t)

	// Create tool
	tool := tools.NewGitHubReviewAnalyzer(cfg, log)
	require.NotNil(t, tool)

	// Test with security PRs focus
	params := map[string]interface{}{
		"analysis_period":      "30d",
		"include_security_prs": true,
		"detailed_metrics":     true,
		"use_cache":            false,
	}

	// Execute tool
	ctx := context.Background()
	result, evidenceSource, err := tool.Execute(ctx, params)

	// Verify results
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, evidenceSource)

	// Should include security analysis
	assert.Contains(t, result, "Security Pull Requests")
	assert.Contains(t, result, "security")

	// Verify metadata shows security focus
	metadata := evidenceSource.Metadata
	assert.Equal(t, "30d", metadata["analysis_period"])
}

func TestGitHubReviewAnalyzer_ComplianceAnalysis(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/test/repo/pulls":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockPullRequestsResponse)
		case r.URL.Path == "/repos/test/repo/pulls/123/reviews":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockReviewsResponse)
		case r.URL.Path == "/repos/test/repo/pulls/124/reviews":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		case r.URL.Path == "/repos/test/repo/commits/abc123def/check-runs":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockCheckRunsResponse)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test configuration
	cfg := createTestConfigReviews(t)
	cfg.Evidence.Tools.GitHub.Repository = "test/repo"
	cfg.Evidence.Tools.GitHub.APIToken = "test-token"

	// Create logger
	log := createTestLoggerReviews(t)

	// Create tool
	tool := tools.NewGitHubReviewAnalyzer(cfg, log)
	require.NotNil(t, tool)

	// Test with compliance checking
	params := map[string]interface{}{
		"analysis_period":  "180d",
		"check_compliance": true,
		"detailed_metrics": true,
		"use_cache":        false,
	}

	// Execute tool
	ctx := context.Background()
	result, evidenceSource, err := tool.Execute(ctx, params)

	// Verify results
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, evidenceSource)

	// Should include compliance analysis
	assert.Contains(t, result, "Compliance Analysis")
	assert.Contains(t, result, "Compliance Score")
	assert.Contains(t, result, "Compliance Rate")

	// Verify compliance score in metadata
	metadata := evidenceSource.Metadata
	complianceScore, ok := metadata["compliance_score"].(float64)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, complianceScore, 0.0)
	assert.LessOrEqual(t, complianceScore, 1.0)
}

func TestGitHubReviewAnalyzer_LimitedPRs(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/test/repo/pulls":
			// Return only first PR based on max_prs parameter
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockPullRequestsResponse[:1])
		case r.URL.Path == "/repos/test/repo/pulls/123/reviews":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockReviewsResponse)
		case r.URL.Path == "/repos/test/repo/commits/abc123def/check-runs":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockCheckRunsResponse)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test configuration
	cfg := createTestConfigReviews(t)
	cfg.Evidence.Tools.GitHub.Repository = "test/repo"
	cfg.Evidence.Tools.GitHub.APIToken = "test-token"

	// Create logger
	log := createTestLoggerReviews(t)

	// Create tool
	tool := tools.NewGitHubReviewAnalyzer(cfg, log)
	require.NotNil(t, tool)

	// Test with limited PRs
	params := map[string]interface{}{
		"analysis_period": "90d",
		"max_prs":         1,
		"use_cache":       false,
	}

	// Execute tool
	ctx := context.Background()
	result, evidenceSource, err := tool.Execute(ctx, params)

	// Verify results
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, evidenceSource)

	// Verify PR count in metadata
	metadata := evidenceSource.Metadata
	prCount, ok := metadata["pr_count"].(int)
	assert.True(t, ok)
	assert.Equal(t, 1, prCount)
}

func TestGitHubReviewAnalyzer_StateFiltering(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/test/repo/pulls":
			// Filter response based on state parameter
			stateParam := r.URL.Query().Get("state")
			var response []map[string]interface{}

			for _, pr := range mockPullRequestsResponse {
				if stateParam == "all" || pr["state"] == stateParam {
					response = append(response, pr)
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(response)
		case r.URL.Path == "/repos/test/repo/pulls/123/reviews":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockReviewsResponse)
		case r.URL.Path == "/repos/test/repo/pulls/124/reviews":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		case r.URL.Path == "/repos/test/repo/commits/abc123def/check-runs":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockCheckRunsResponse)
		case r.URL.Path == "/repos/test/repo/commits/ghi789jkl/check-runs":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{"check_runs": []map[string]interface{}{}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Test different states
	testCases := []struct {
		state    string
		expected string
	}{
		{"open", "Open PRs"},
		{"closed", "Closed PRs"},
		{"merged", "Merged PRs"},
		{"all", "Total Pull Requests"},
	}

	for _, tc := range testCases {
		t.Run("state_"+tc.state, func(t *testing.T) {
			// Create test configuration
			cfg := createTestConfig(t)
			cfg.Evidence.Tools.GitHub.Repository = "test/repo"
			cfg.Evidence.Tools.GitHub.APIToken = "test-token"

			// Create logger
			log := createTestLogger(t)

			// Create tool
			tool := tools.NewGitHubReviewAnalyzer(cfg, log)
			require.NotNil(t, tool)

			// Test with specific state
			params := map[string]interface{}{
				"analysis_period": "90d",
				"state":           tc.state,
				"use_cache":       false,
			}

			// Execute tool
			ctx := context.Background()
			result, evidenceSource, err := tool.Execute(ctx, params)

			// Verify results
			require.NoError(t, err)
			assert.NotEmpty(t, result)
			assert.NotNil(t, evidenceSource)

			// Verify state-specific content
			assert.Contains(t, result, tc.expected)
		})
	}
}

func TestGitHubReviewAnalyzer_Name(t *testing.T) {
	cfg := createTestConfigReviews(t)
	log := createTestLoggerReviews(t)

	tool := tools.NewGitHubReviewAnalyzer(cfg, log)
	require.NotNil(t, tool)

	assert.Equal(t, "github-review-analyzer", tool.Name())
}

func TestGitHubReviewAnalyzer_Description(t *testing.T) {
	cfg := createTestConfigReviews(t)
	log := createTestLoggerReviews(t)

	tool := tools.NewGitHubReviewAnalyzer(cfg, log)
	require.NotNil(t, tool)

	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "pull request")
	assert.Contains(t, desc, "review")
	assert.Contains(t, desc, "approval")
	assert.Contains(t, desc, "SOC2")
}

func TestGitHubReviewAnalyzer_ClaudeToolDefinition(t *testing.T) {
	cfg := createTestConfigReviews(t)
	log := createTestLoggerReviews(t)

	tool := tools.NewGitHubReviewAnalyzer(cfg, log)
	require.NotNil(t, tool)

	claudeTool := tool.GetClaudeToolDefinition()

	assert.Equal(t, "github-review-analyzer", claudeTool.Name)
	assert.NotEmpty(t, claudeTool.Description)
	assert.NotNil(t, claudeTool.InputSchema)

	// Verify input schema structure
	schema := claudeTool.InputSchema
	assert.Equal(t, "object", schema["type"])

	properties, ok := schema["properties"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, properties, "analysis_period")
	assert.Contains(t, properties, "state")
	assert.Contains(t, properties, "include_security_prs")
	assert.Contains(t, properties, "detailed_metrics")
	assert.Contains(t, properties, "check_compliance")
	assert.Contains(t, properties, "max_prs")
	assert.Contains(t, properties, "use_cache")

	// Verify analysis_period enum
	analysisPeriod, ok := properties["analysis_period"].(map[string]interface{})
	assert.True(t, ok)
	enum, ok := analysisPeriod["enum"].([]string)
	assert.True(t, ok)
	assert.Contains(t, enum, "30d")
	assert.Contains(t, enum, "90d")
	assert.Contains(t, enum, "180d")
	assert.Contains(t, enum, "1y")

	// Verify state enum
	state, ok := properties["state"].(map[string]interface{})
	assert.True(t, ok)
	stateEnum, ok := state["enum"].([]string)
	assert.True(t, ok)
	assert.Contains(t, stateEnum, "open")
	assert.Contains(t, stateEnum, "closed")
	assert.Contains(t, stateEnum, "merged")
	assert.Contains(t, stateEnum, "all")

	// Verify max_prs constraints
	maxPRs, ok := properties["max_prs"].(map[string]interface{})
	assert.True(t, ok)
	assert.Equal(t, 10, maxPRs["minimum"])
	assert.Equal(t, 1000, maxPRs["maximum"])
}

func TestGitHubReviewAnalyzer_AuthenticationRequired(t *testing.T) {
	// Create test configuration without token
	cfg := createTestConfig(t)
	cfg.Evidence.Tools.GitHub.Repository = "test/repo"
	cfg.Evidence.Tools.GitHub.APIToken = ""
	cfg.Auth.GitHub.Token = ""

	// Create logger
	log := createTestLogger(t)

	// Create tool
	tool := tools.NewGitHubReviewAnalyzer(cfg, log)
	require.NotNil(t, tool)

	// Test parameters
	params := map[string]interface{}{
		"analysis_period": "90d",
		"use_cache":       false,
	}

	// Execute tool without authentication
	ctx := context.Background()
	result, evidenceSource, err := tool.Execute(ctx, params)

	// Should still work with limited functionality or cached data
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, evidenceSource)

	// Verify auth status in metadata
	metadata := evidenceSource.Metadata
	authStatus, ok := metadata["auth_status"].(map[string]interface{})
	assert.True(t, ok)
	// Should indicate authentication issues
	assert.False(t, authStatus["authenticated"].(bool))
}

func TestGitHubReviewAnalyzer_EmptyResponse(t *testing.T) {
	// Create test server that returns empty PR list
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test/repo/pulls" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test configuration
	cfg := createTestConfigReviews(t)
	cfg.Evidence.Tools.GitHub.Repository = "test/repo"
	cfg.Evidence.Tools.GitHub.APIToken = "test-token"

	// Create logger
	log := createTestLoggerReviews(t)

	// Create tool
	tool := tools.NewGitHubReviewAnalyzer(cfg, log)
	require.NotNil(t, tool)

	// Test parameters
	params := map[string]interface{}{
		"analysis_period": "90d",
		"use_cache":       false,
	}

	// Execute tool
	ctx := context.Background()
	result, evidenceSource, err := tool.Execute(ctx, params)

	// Should succeed with empty PR list
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, evidenceSource)

	// Verify it handles no PRs gracefully
	assert.Contains(t, result, "Total Pull Requests: 0")

	// Verify metadata shows zero PRs
	metadata := evidenceSource.Metadata
	assert.Equal(t, 0, metadata["pr_count"])
}

func TestGitHubReviewAnalyzer_InvalidPeriod(t *testing.T) {
	// Create test configuration
	cfg := createTestConfig(t)
	cfg.Evidence.Tools.GitHub.Repository = "test/repo"

	// Create logger
	log := createTestLogger(t)

	// Create tool
	tool := tools.NewGitHubReviewAnalyzer(cfg, log)
	require.NotNil(t, tool)

	// Test with invalid period
	params := map[string]interface{}{
		"analysis_period": "invalid",
	}

	// Execute tool
	ctx := context.Background()
	result, evidenceSource, err := tool.Execute(ctx, params)
	require.NoError(t, err)

	// Should handle invalid parameters gracefully
	// Note: Validation happens at CLI level, so tool should handle this
	assert.NotEmpty(t, result)
	assert.NotNil(t, evidenceSource)
}

// Helper function to create test configuration
func createTestConfigReviews(t *testing.T) *config.Config {
	tempDir := t.TempDir()

	return &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:          true,
					Repository:       "test/repo",
					IncludeWorkflows: true,
					IncludeIssues:    true,
					MaxIssues:        50,
				},
			},
		},
		Auth: config.AuthConfig{
			CacheDir: filepath.Join(tempDir, ".auth_cache"),
			GitHub: config.GitHubAuthConfig{
				Token: "",
			},
		},
		Logging: config.LoggingConfig{
			// Empty - use defaults
		},
	}
}

// Helper function to create test logger
func createTestLoggerReviews(t *testing.T) logger.Logger {
	logConfig := logger.Config{
		Level:  logger.WarnLevel,
		Output: "stderr",
	}

	log, err := logger.New(&logConfig)
	require.NoError(t, err)

	return log
}
