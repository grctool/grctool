package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubWorkflowAnalysis_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	analysis := GitHubWorkflowAnalysis{
		Repository:   "org/test-repo",
		AnalysisDate: now,
		WorkflowFiles: []GitHubWorkflowFile{
			{
				Name: "ci.yml",
				Path: ".github/workflows/ci.yml",
				Triggers: []GitHubWorkflowTrigger{
					{Event: "push", Branches: []string{"main"}},
					{Event: "pull_request"},
				},
				Jobs: []GitHubWorkflowJob{
					{
						ID:     "test",
						Name:   "Run Tests",
						RunsOn: "ubuntu-latest",
						Steps: []GitHubWorkflowStep{
							{Name: "Checkout", Uses: "actions/checkout@v4"},
							{Name: "Test", Run: "go test ./..."},
						},
					},
				},
				SecuritySteps: []GitHubWorkflowSecurityStep{
					{
						StepName: "CodeQL",
						Action:   "github/codeql-action/analyze@v2",
						Purpose:  "code_scanning",
						Tool:     "CodeQL",
					},
				},
				Relevance: 0.9,
			},
		},
		SecurityScans: []GitHubSecurityScan{
			{
				Type:    "codeql",
				Enabled: true,
				Status:  "active",
			},
		},
		ApprovalRules: []GitHubApprovalRule{
			{
				Branch:          "main",
				RequiredReviews: 2,
				RestrictPushes:  true,
			},
		},
		Statistics: GitHubWorkflowStatistics{
			TotalWorkflows:    3,
			SecurityWorkflows: 1,
			ToolsUsed:         map[string]int{"CodeQL": 1, "Trivy": 1},
			TriggerTypes:      map[string]int{"push": 2, "pull_request": 1},
		},
		ComplianceScore: 0.85,
	}

	data, err := json.Marshal(analysis)
	require.NoError(t, err)

	var decoded GitHubWorkflowAnalysis
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "org/test-repo", decoded.Repository)
	assert.Len(t, decoded.WorkflowFiles, 1)
	assert.Len(t, decoded.WorkflowFiles[0].Triggers, 2)
	assert.Len(t, decoded.WorkflowFiles[0].Jobs, 1)
	assert.Len(t, decoded.WorkflowFiles[0].Jobs[0].Steps, 2)
	assert.InDelta(t, 0.85, decoded.ComplianceScore, 0.001)
}

func TestGitHubPullRequestAnalysis_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	mergedAt := now.Add(-time.Hour)

	analysis := GitHubPullRequestAnalysis{
		Repository:   "org/test-repo",
		AnalysisDate: now,
		DateRange: GitHubDateRange{
			StartDate: now.Add(-30 * 24 * time.Hour),
			EndDate:   now,
			Period:    "30d",
		},
		PullRequests: []GitHubPullRequestDetail{
			{
				Number:   42,
				Title:    "Fix security vulnerability",
				State:    "closed",
				Author:   GitHubUser{Login: "alice", ID: 1},
				MergedAt: &mergedAt,
				Reviews: []GitHubPRReview{
					{
						ID: 1,
						User:  GitHubUser{Login: "bob"},
						State: "APPROVED",
					},
				},
				ChangedFiles: 5,
				Additions:    100,
				Deletions:    20,
				ComplianceStatus: GitHubPRComplianceStatus{
					RequiredReviewsMet: true,
					StatusChecksPassed: true,
					ComplianceScore:    1.0,
				},
				SecurityRelevant: true,
				Relevance:        0.95,
			},
		},
		ReviewMetrics: GitHubReviewMetrics{
			TotalPRs:            50,
			MergedPRs:           45,
			AverageReviewsPerPR: 1.5,
			ApprovalRate:        0.9,
			ComplianceRate:      0.95,
		},
		ComplianceScore: 0.92,
		Recommendations: []GitHubRecommendation{
			{
				Type:        "security",
				Priority:    "high",
				Title:       "Enable CODEOWNERS",
				Description: "Add CODEOWNERS file for critical paths",
			},
		},
	}

	data, err := json.Marshal(analysis)
	require.NoError(t, err)

	var decoded GitHubPullRequestAnalysis
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "org/test-repo", decoded.Repository)
	assert.Len(t, decoded.PullRequests, 1)
	assert.Equal(t, 42, decoded.PullRequests[0].Number)
	assert.NotNil(t, decoded.PullRequests[0].MergedAt)
	assert.True(t, decoded.PullRequests[0].SecurityRelevant)
	assert.Equal(t, 50, decoded.ReviewMetrics.TotalPRs)
	assert.Len(t, decoded.Recommendations, 1)
}

func TestGitHubComplianceRule_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	rule := GitHubComplianceRule{
		RuleID:      "CR-001",
		Name:        "Required Code Reviews",
		Description: "All PRs must have code review",
		Status:      "pass",
		Evidence:    "Branch protection requires 2 reviews",
		Framework:   "SOC2",
		ControlID:   "CC8.1",
	}

	data, err := json.Marshal(rule)
	require.NoError(t, err)

	var decoded GitHubComplianceRule
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, rule.RuleID, decoded.RuleID)
	assert.Equal(t, rule.Status, decoded.Status)
	assert.Equal(t, rule.ControlID, decoded.ControlID)
}

func TestGitHubComplianceBreakdown_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	bd := GitHubComplianceBreakdown{
		Category:     "code_review",
		Passed:       8,
		Failed:       1,
		Warnings:     2,
		NotTested:    0,
		Score:        0.73,
		Requirements: []string{"Reviews required", "CODEOWNERS enforced"},
	}

	data, err := json.Marshal(bd)
	require.NoError(t, err)

	var decoded GitHubComplianceBreakdown
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "code_review", decoded.Category)
	assert.Equal(t, 8, decoded.Passed)
	assert.InDelta(t, 0.73, decoded.Score, 0.001)
}

func TestGitHubWorkflowJob_WithStrategy(t *testing.T) {
	t.Parallel()
	failFast := true
	job := GitHubWorkflowJob{
		ID:          "build",
		Name:        "Build",
		RunsOn:      "ubuntu-latest",
		Environment: "staging",
		Needs:       []string{"test"},
		Strategy: &GitHubWorkflowJobStrategy{
			Matrix: map[string]interface{}{
				"go-version": []string{"1.21", "1.22"},
			},
			FailFast:    &failFast,
			MaxParallel: 2,
		},
		TimeoutMinutes: 30,
		Permissions:    map[string]string{"contents": "read"},
	}

	data, err := json.Marshal(job)
	require.NoError(t, err)

	var decoded GitHubWorkflowJob
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "build", decoded.ID)
	assert.Equal(t, "staging", decoded.Environment)
	assert.NotNil(t, decoded.Strategy)
	assert.NotNil(t, decoded.Strategy.FailFast)
	assert.True(t, *decoded.Strategy.FailFast)
	assert.Equal(t, 2, decoded.Strategy.MaxParallel)
}

func TestGitHubApprovalPatterns_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	patterns := GitHubApprovalPatterns{
		CommonApprovers: []GitHubUser{
			{Login: "alice", ID: 1},
		},
		ApprovalChains: []GitHubApprovalChain{
			{
				Pattern:    []string{"developer", "lead", "security"},
				Frequency:  10,
				Percentage: 0.5,
			},
		},
		BotUsage:              map[string]int{"dependabot": 5},
		AutoMergeUsage:        3,
		RequiredCheckPatterns: map[string]int{"ci/test": 20},
	}

	data, err := json.Marshal(patterns)
	require.NoError(t, err)

	var decoded GitHubApprovalPatterns
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.CommonApprovers, 1)
	assert.Len(t, decoded.ApprovalChains, 1)
	assert.Equal(t, 3, decoded.AutoMergeUsage)
}
