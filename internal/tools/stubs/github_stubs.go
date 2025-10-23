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

package stubs

import (
	"context"
	"fmt"

	"github.com/grctool/grctool/internal/models"
)

// StubGitHubAPIClient provides stub implementation for GitHub API operations
type StubGitHubAPIClient struct {
	Repositories     map[string]*models.GitHubRepositoryInfo
	Collaborators    map[string][]models.GitHubCollaborator
	Teams            map[string][]models.GitHubTeam
	Branches         map[string][]models.GitHubBranch
	Environments     map[string][]models.GitHubEnvironment
	SecuritySettings map[string]*models.GitHubSecuritySettings
	OrgMembers       map[string][]models.GitHubOrgMember
	Errors           map[string]error
}

// NewStubGitHubAPIClient creates a new stub GitHub API client
func NewStubGitHubAPIClient() *StubGitHubAPIClient {
	return &StubGitHubAPIClient{
		Repositories:     make(map[string]*models.GitHubRepositoryInfo),
		Collaborators:    make(map[string][]models.GitHubCollaborator),
		Teams:            make(map[string][]models.GitHubTeam),
		Branches:         make(map[string][]models.GitHubBranch),
		Environments:     make(map[string][]models.GitHubEnvironment),
		SecuritySettings: make(map[string]*models.GitHubSecuritySettings),
		OrgMembers:       make(map[string][]models.GitHubOrgMember),
		Errors:           make(map[string]error),
	}
}

// GetRepositoryCollaborators returns stubbed collaborators for a repository
func (s *StubGitHubAPIClient) GetRepositoryCollaborators(ctx context.Context, owner, repo string) ([]models.GitHubCollaborator, error) {
	key := fmt.Sprintf("%s/%s", owner, repo)
	if err, ok := s.Errors[key]; ok {
		return nil, err
	}
	if collaborators, ok := s.Collaborators[key]; ok {
		return collaborators, nil
	}
	return []models.GitHubCollaborator{}, nil
}

// GetRepositoryTeams returns stubbed teams for a repository
func (s *StubGitHubAPIClient) GetRepositoryTeams(ctx context.Context, owner, repo string) ([]models.GitHubTeam, error) {
	key := fmt.Sprintf("%s/%s", owner, repo)
	if err, ok := s.Errors[key]; ok {
		return nil, err
	}
	if teams, ok := s.Teams[key]; ok {
		return teams, nil
	}
	return []models.GitHubTeam{}, nil
}

// GetRepositoryBranches returns stubbed branches for a repository
func (s *StubGitHubAPIClient) GetRepositoryBranches(ctx context.Context, owner, repo string) ([]models.GitHubBranch, error) {
	key := fmt.Sprintf("%s/%s", owner, repo)
	if err, ok := s.Errors[key]; ok {
		return nil, err
	}
	if branches, ok := s.Branches[key]; ok {
		return branches, nil
	}
	return []models.GitHubBranch{}, nil
}

// GetDeploymentEnvironments returns stubbed deployment environments for a repository
func (s *StubGitHubAPIClient) GetDeploymentEnvironments(ctx context.Context, owner, repo string) ([]models.GitHubEnvironment, error) {
	key := fmt.Sprintf("%s/%s", owner, repo)
	if err, ok := s.Errors[key]; ok {
		return nil, err
	}
	if environments, ok := s.Environments[key]; ok {
		return environments, nil
	}
	return []models.GitHubEnvironment{}, nil
}

// GetRepositorySecurity returns stubbed security settings for a repository
func (s *StubGitHubAPIClient) GetRepositorySecurity(ctx context.Context, owner, repo string) (*models.GitHubSecuritySettings, error) {
	key := fmt.Sprintf("%s/%s", owner, repo)
	if err, ok := s.Errors[key]; ok {
		return nil, err
	}
	if settings, ok := s.SecuritySettings[key]; ok {
		return settings, nil
	}
	return &models.GitHubSecuritySettings{}, nil
}

// GetOrganizationMembers returns stubbed organization members
func (s *StubGitHubAPIClient) GetOrganizationMembers(ctx context.Context, org string) ([]models.GitHubOrgMember, error) {
	if err, ok := s.Errors[org]; ok {
		return nil, err
	}
	if members, ok := s.OrgMembers[org]; ok {
		return members, nil
	}
	return []models.GitHubOrgMember{}, nil
}

// Helper methods to set up test data

// WithRepository adds a repository to the stub
func (s *StubGitHubAPIClient) WithRepository(owner, repo string, repoInfo *models.GitHubRepositoryInfo) *StubGitHubAPIClient {
	key := fmt.Sprintf("%s/%s", owner, repo)
	s.Repositories[key] = repoInfo
	return s
}

// WithCollaborators adds collaborators to the stub
func (s *StubGitHubAPIClient) WithCollaborators(owner, repo string, collaborators []models.GitHubCollaborator) *StubGitHubAPIClient {
	key := fmt.Sprintf("%s/%s", owner, repo)
	s.Collaborators[key] = collaborators
	return s
}

// WithTeams adds teams to the stub
func (s *StubGitHubAPIClient) WithTeams(owner, repo string, teams []models.GitHubTeam) *StubGitHubAPIClient {
	key := fmt.Sprintf("%s/%s", owner, repo)
	s.Teams[key] = teams
	return s
}

// WithBranches adds branches to the stub
func (s *StubGitHubAPIClient) WithBranches(owner, repo string, branches []models.GitHubBranch) *StubGitHubAPIClient {
	key := fmt.Sprintf("%s/%s", owner, repo)
	s.Branches[key] = branches
	return s
}

// WithEnvironments adds environments to the stub
func (s *StubGitHubAPIClient) WithEnvironments(owner, repo string, environments []models.GitHubEnvironment) *StubGitHubAPIClient {
	key := fmt.Sprintf("%s/%s", owner, repo)
	s.Environments[key] = environments
	return s
}

// WithSecuritySettings adds security settings to the stub
func (s *StubGitHubAPIClient) WithSecuritySettings(owner, repo string, settings *models.GitHubSecuritySettings) *StubGitHubAPIClient {
	key := fmt.Sprintf("%s/%s", owner, repo)
	s.SecuritySettings[key] = settings
	return s
}

// WithOrgMembers adds organization members to the stub
func (s *StubGitHubAPIClient) WithOrgMembers(org string, members []models.GitHubOrgMember) *StubGitHubAPIClient {
	s.OrgMembers[org] = members
	return s
}

// WithError adds an error for a specific operation
func (s *StubGitHubAPIClient) WithError(key string, err error) *StubGitHubAPIClient {
	s.Errors[key] = err
	return s
}

// CreateTestRepository creates a test repository with common data
func CreateTestRepository() *models.GitHubRepositoryInfo {
	return &models.GitHubRepositoryInfo{
		Name:     "test-repo",
		FullName: "test-org/test-repo",
		Owner:    "test-org",
		Private:  true,
	}
}

// CreateTestCollaborators creates test collaborators with various permission levels
func CreateTestCollaborators() []models.GitHubCollaborator {
	return []models.GitHubCollaborator{
		{
			Login: "admin-user",
			ID:    1,
			Permissions: models.GitHubPermissions{
				Permission: "admin",
				Admin:      true,
				Maintain:   true,
				Push:       true,
				Triage:     true,
				Pull:       true,
			},
		},
		{
			Login: "dev-user",
			ID:    2,
			Permissions: models.GitHubPermissions{
				Permission: "push",
				Admin:      false,
				Maintain:   false,
				Push:       true,
				Triage:     true,
				Pull:       true,
			},
		},
		{
			Login: "read-user",
			ID:    3,
			Permissions: models.GitHubPermissions{
				Permission: "pull",
				Admin:      false,
				Maintain:   false,
				Push:       false,
				Triage:     false,
				Pull:       true,
			},
		},
	}
}

// CreateTestTeams creates test teams with members
func CreateTestTeams() []models.GitHubTeam {
	return []models.GitHubTeam{
		{
			ID:         1,
			Name:       "admin-team",
			Slug:       "admin-team",
			Permission: "admin",
			Privacy:    "closed",
			Members: []models.GitHubTeamMember{
				{
					Login: "team-admin",
					ID:    4,
					Role:  "maintainer",
				},
			},
		},
		{
			ID:         2,
			Name:       "dev-team",
			Slug:       "dev-team",
			Permission: "push",
			Privacy:    "closed",
			Members: []models.GitHubTeamMember{
				{
					Login: "team-dev1",
					ID:    5,
					Role:  "member",
				},
				{
					Login: "team-dev2",
					ID:    6,
					Role:  "member",
				},
			},
		},
	}
}

// CreateTestBranches creates test branches with protection rules
func CreateTestBranches() []models.GitHubBranch {
	return []models.GitHubBranch{
		{
			Name:      "main",
			Protected: true,
			Protection: &models.GitHubBranchProtection{
				Enabled: true,
				RequiredPullRequestReviews: &models.GitHubRequiredPullRequestReviews{
					RequiredApprovingReviewCount: 2,
					RequireCodeOwnerReviews:      true,
					DismissStaleReviews:          true,
				},
				RequiredStatusChecks: &models.GitHubRequiredStatusChecks{
					Strict:   true,
					Contexts: []string{"ci/tests", "ci/lint"},
				},
				EnforceAdmins: models.GitHubEnforceAdmins{
					Enabled: true,
				},
				AllowForcePushes: models.GitHubAllowForcePushes{
					Enabled: false,
				},
				AllowDeletions: models.GitHubAllowDeletions{
					Enabled: false,
				},
			},
		},
		{
			Name:      "develop",
			Protected: false,
		},
	}
}

// CreateTestEnvironments creates test deployment environments
func CreateTestEnvironments() []models.GitHubEnvironment {
	return []models.GitHubEnvironment{
		{
			ID:   "1",
			Name: "production",
			ProtectionRules: []models.GitHubEnvironmentProtection{
				{
					Type: "required_reviewers",
					RequiredReviewers: []models.GitHubRequiredReviewer{
						{
							Login: "prod-admin",
							Type:  "User",
						},
					},
				},
				{
					Type:      "wait_timer",
					WaitTimer: 30,
				},
			},
		},
		{
			ID:   "2",
			Name: "staging",
			ProtectionRules: []models.GitHubEnvironmentProtection{
				{
					Type: "required_reviewers",
					RequiredReviewers: []models.GitHubRequiredReviewer{
						{
							Slug: "dev-team",
							Type: "Team",
						},
					},
				},
			},
		},
	}
}

// CreateTestSecuritySettings creates test security settings
func CreateTestSecuritySettings() *models.GitHubSecuritySettings {
	return &models.GitHubSecuritySettings{
		VulnerabilityAlertsEnabled:    true,
		AutomatedSecurityFixesEnabled: true,
		SecretScanningEnabled:         true,
		CodeScanningEnabled:           false,
		DependencyGraphEnabled:        true,
		SecurityAdvisoryEnabled:       false,
	}
}
