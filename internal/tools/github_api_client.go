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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/transport"
)

// GitHubAPIClient provides comprehensive GitHub API access for SOC2 audit evidence
type GitHubAPIClient struct {
	config     *config.GitHubToolConfig
	httpClient *http.Client
	logger     logger.Logger
	baseURL    string
	graphqlURL string
}

// NewGitHubAPIClient creates a new comprehensive GitHub API client
func NewGitHubAPIClient(cfg *config.Config, log logger.Logger) *GitHubAPIClient {
	// Create HTTP transport with logging
	httpTransport := http.DefaultTransport
	httpTransport = transport.NewLoggingTransport(httpTransport, log.WithComponent("github-api-client"))

	return &GitHubAPIClient{
		config: &cfg.Evidence.Tools.GitHub,
		httpClient: &http.Client{
			Timeout:   60 * time.Second, // Longer timeout for comprehensive API calls
			Transport: httpTransport,
		},
		logger:     log,
		baseURL:    "https://api.github.com",
		graphqlURL: "https://api.github.com/graphql",
	}
}

// makeRESTRequest makes a REST API request with proper authentication and error handling
func (client *GitHubAPIClient) makeRESTRequest(ctx context.Context, method, endpoint string, body interface{}) (*http.Response, error) {
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

// makeGraphQLRequest makes a GraphQL API request
func (client *GitHubAPIClient) makeGraphQLRequest(ctx context.Context, query string, variables map[string]interface{}) (*GraphQLResponse, error) {
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
	if err := json.NewDecoder(resp.Body).Decode(&gqlResponse); err != nil {
		return nil, fmt.Errorf("failed to decode GraphQL response: %w", err)
	}

	if len(gqlResponse.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL errors: %v", gqlResponse.Errors)
	}

	return &gqlResponse, nil
}

// GetRepositoryCollaborators gets all repository collaborators with their permissions
func (client *GitHubAPIClient) GetRepositoryCollaborators(ctx context.Context, owner, repo string) ([]models.GitHubCollaborator, error) {
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
func (client *GitHubAPIClient) GetUserRepositoryPermissions(ctx context.Context, owner, repo, username string) (*models.GitHubPermissions, error) {
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
func (client *GitHubAPIClient) GetRepositoryTeams(ctx context.Context, owner, repo string) ([]models.GitHubTeam, error) {
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
func (client *GitHubAPIClient) GetTeamMembers(ctx context.Context, teamID int) ([]models.GitHubTeamMember, error) {
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

// GetBranchProtection gets branch protection rules for a specific branch
func (client *GitHubAPIClient) GetBranchProtection(ctx context.Context, owner, repo, branch string) (*models.GitHubBranchProtection, error) {
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

// GetRepositoryBranches gets all branches in the repository
func (client *GitHubAPIClient) GetRepositoryBranches(ctx context.Context, owner, repo string) ([]models.GitHubBranch, error) {
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

// GetDeploymentEnvironments gets all deployment environments using GraphQL
func (client *GitHubAPIClient) GetDeploymentEnvironments(ctx context.Context, owner, repo string) ([]models.GitHubEnvironment, error) {
	query := `
	query($owner: String!, $repo: String!) {
		repository(owner: $owner, name: $repo) {
			environments(first: 100) {
				nodes {
					name
					id
					protectionRules(first: 100) {
						nodes {
							type
							... on RequiredReviewers {
								requiredReviewers {
									... on User {
										login
										name
										email
									}
									... on Team {
										name
										slug
									}
								}
							}
							... on WaitTimer {
								waitTimer
							}
						}
					}
				}
			}
		}
	}`

	variables := map[string]interface{}{
		"owner": owner,
		"repo":  repo,
	}

	resp, err := client.makeGraphQLRequest(ctx, query, variables)
	if err != nil {
		return nil, fmt.Errorf("failed to get deployment environments: %w", err)
	}

	var result struct {
		Repository struct {
			Environments struct {
				Nodes []models.GitHubEnvironment `json:"nodes"`
			} `json:"environments"`
		} `json:"repository"`
	}

	if err := json.Unmarshal(resp.Data, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal environments response: %w", err)
	}

	return result.Repository.Environments.Nodes, nil
}

// GetRepositorySecurity gets repository security settings
func (client *GitHubAPIClient) GetRepositorySecurity(ctx context.Context, owner, repo string) (*models.GitHubSecuritySettings, error) {
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

// Helper methods for security settings

func (client *GitHubAPIClient) getVulnerabilityAlerts(ctx context.Context, owner, repo string) (bool, error) {
	endpoint := fmt.Sprintf("/repos/%s/%s/vulnerability-alerts", owner, repo)

	resp, err := client.makeRESTRequest(ctx, "GET", endpoint, nil)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusNoContent, nil
}

func (client *GitHubAPIClient) getAutomatedSecurityFixes(ctx context.Context, owner, repo string) (bool, error) {
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

func (client *GitHubAPIClient) getSecretScanning(ctx context.Context, owner, repo string) (bool, error) {
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

func (client *GitHubAPIClient) getCodeScanning(ctx context.Context, owner, repo string) (bool, error) {
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

// GetOrganizationMembers gets all organization members (if applicable)
func (client *GitHubAPIClient) GetOrganizationMembers(ctx context.Context, org string) ([]models.GitHubOrgMember, error) {
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

// GraphQLResponse represents a GraphQL API response
type GraphQLResponse struct {
	Data   json.RawMessage `json:"data"`
	Errors []struct {
		Message string        `json:"message"`
		Path    []interface{} `json:"path"`
	} `json:"errors"`
}
