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
	"time"

	"github.com/grctool/grctool/internal/models"
)

// Shared types for GitHub tools consolidation

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

// DeploymentAccessInfo contains comprehensive deployment access information
type DeploymentAccessInfo struct {
	Repository   string                          `json:"repository"`
	Environments []models.GitHubEnvironment      `json:"environments"`
	BranchRules  []models.GitHubBranch           `json:"branch_rules,omitempty"`
	AccessMatrix []models.GitHubDeploymentAccess `json:"access_matrix"`
	ExtractedAt  time.Time                       `json:"extracted_at"`
}

// SecurityFeaturesInfo contains comprehensive security features information
type SecurityFeaturesInfo struct {
	Repository              string                           `json:"repository"`
	SecuritySettings        models.GitHubSecuritySettings    `json:"security_settings"`
	BranchProtections       []models.GitHubBranch            `json:"branch_protections"`
	SecurityPolicies        []SecurityPolicyInfo             `json:"security_policies,omitempty"`
	ComplianceMapping       map[string][]string              `json:"compliance_mapping,omitempty"`
	AllFeatures             map[string]SecurityFeatureDetail `json:"all_features"`
	EnabledFeatures         []string                         `json:"enabled_features"`
	DisabledFeatures        []string                         `json:"disabled_features"`
	SecurityScore           float64                          `json:"security_score"`
	SecurityRecommendations []SecurityRecommendation         `json:"security_recommendations"`
	ExtractedAt             time.Time                        `json:"extracted_at"`
}

// SecurityFeatureDetail provides detailed information about a security feature
type SecurityFeatureDetail struct {
	Name         string   `json:"name"`
	Enabled      bool     `json:"enabled"`
	Description  string   `json:"description"`
	SOC2Controls []string `json:"soc2_controls,omitempty"`
	RiskLevel    string   `json:"risk_level"` // high, medium, low
	Category     string   `json:"category"`   // vulnerability, secrets, code_quality, access_control
}

// SecurityPolicyInfo contains information about security policies
type SecurityPolicyInfo struct {
	PolicyType     string    `json:"policy_type"`
	PolicyFile     string    `json:"policy_file"`
	LastUpdated    time.Time `json:"last_updated"`
	ContentSummary string    `json:"content_summary"`
}

// SecurityRecommendation provides actionable security recommendations
type SecurityRecommendation struct {
	Priority      string   `json:"priority"` // high, medium, low
	Category      string   `json:"category"` // feature, policy, access_control
	Title         string   `json:"title"`
	Description   string   `json:"description"`
	ActionItems   []string `json:"action_items"`
	SOC2Relevance string   `json:"soc2_relevance,omitempty"`
}

// GraphQLResponse represents a GraphQL API response
type GraphQLResponse struct {
	Data   []byte `json:"data"`
	Errors []struct {
		Message string        `json:"message"`
		Path    []interface{} `json:"path"`
	} `json:"errors"`
}
