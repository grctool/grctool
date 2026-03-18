package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGitHubPermissionMatrix_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	matrix := GitHubPermissionMatrix{
		Repository: "org/test-repo",
		UserAccess: []GitHubUserAccess{
			{
				Username:        "alice",
				UserID:          1,
				DirectAccess:    "admin",
				TeamMemberships: []string{"engineering"},
				EffectiveAccess: "admin",
				CanDeploy:       []string{"production"},
				CanPushTo:       []string{"main"},
				IsOrgAdmin:      true,
			},
		},
		TeamAccess: []GitHubTeamAccess{
			{
				TeamName:    "Engineering",
				TeamID:      100,
				TeamSlug:    "engineering",
				Permission:  "push",
				MemberCount: 5,
				Members:     []string{"alice", "bob"},
			},
		},
		BranchRules: []GitHubBranchAccessRule{
			{
				BranchName:                    "main",
				IsProtected:                   true,
				RequiredReviews:               2,
				RequireCodeOwnerReviews:       true,
				DismissStaleReviews:           true,
				EnforceAdmins:                 true,
				RequireConversationResolution: true,
			},
		},
		DeploymentRules: []GitHubDeploymentAccess{
			{
				Environment: "production",
				RequiredReviewers: []GitHubRequiredReviewer{
					{Login: "alice", Type: "User"},
				},
			},
		},
		GeneratedAt: now,
	}

	data, err := json.Marshal(matrix)
	require.NoError(t, err)

	var decoded GitHubPermissionMatrix
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "org/test-repo", decoded.Repository)
	assert.Len(t, decoded.UserAccess, 1)
	assert.True(t, decoded.UserAccess[0].IsOrgAdmin)
	assert.Len(t, decoded.TeamAccess, 1)
	assert.Equal(t, 5, decoded.TeamAccess[0].MemberCount)
	assert.Len(t, decoded.BranchRules, 1)
	assert.True(t, decoded.BranchRules[0].IsProtected)
}

func TestGitHubBranchProtection_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	bp := GitHubBranchProtection{
		URL:     "https://api.github.com/repos/org/repo/branches/main/protection",
		Enabled: true,
		RequiredStatusChecks: &GitHubRequiredStatusChecks{
			Strict:   true,
			Contexts: []string{"ci/test", "ci/lint"},
		},
		EnforceAdmins: GitHubEnforceAdmins{Enabled: true},
		RequiredPullRequestReviews: &GitHubRequiredPullRequestReviews{
			DismissStaleReviews:          true,
			RequireCodeOwnerReviews:      true,
			RequiredApprovingReviewCount: 2,
			RequireLastPushApproval:      true,
		},
		RequiredLinearHistory:          GitHubRequiredLinearHistory{Enabled: false},
		AllowForcePushes:               GitHubAllowForcePushes{Enabled: false},
		AllowDeletions:                 GitHubAllowDeletions{Enabled: false},
		RequiredConversationResolution: GitHubRequiredConversationResolution{Enabled: true},
	}

	data, err := json.Marshal(bp)
	require.NoError(t, err)

	var decoded GitHubBranchProtection
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.True(t, decoded.Enabled)
	assert.NotNil(t, decoded.RequiredStatusChecks)
	assert.True(t, decoded.RequiredStatusChecks.Strict)
	assert.Len(t, decoded.RequiredStatusChecks.Contexts, 2)
	assert.True(t, decoded.EnforceAdmins.Enabled)
	assert.NotNil(t, decoded.RequiredPullRequestReviews)
	assert.Equal(t, 2, decoded.RequiredPullRequestReviews.RequiredApprovingReviewCount)
	assert.False(t, decoded.AllowForcePushes.Enabled)
}

func TestGitHubEnvironment_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	env := GitHubEnvironment{
		ID:   "env-1",
		Name: "production",
		ProtectionRules: []GitHubEnvironmentProtection{
			{
				Type: "required_reviewers",
				RequiredReviewers: []GitHubRequiredReviewer{
					{Login: "alice", Type: "User"},
					{Name: "Engineering", Slug: "engineering", Type: "Team"},
				},
			},
			{
				Type:      "wait_timer",
				WaitTimer: 30,
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(env)
	require.NoError(t, err)

	var decoded GitHubEnvironment
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "production", decoded.Name)
	assert.Len(t, decoded.ProtectionRules, 2)
	assert.Len(t, decoded.ProtectionRules[0].RequiredReviewers, 2)
	assert.Equal(t, 30, decoded.ProtectionRules[1].WaitTimer)
}

func TestGitHubSecuritySettings_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	ss := GitHubSecuritySettings{
		VulnerabilityAlertsEnabled:    true,
		AutomatedSecurityFixesEnabled: true,
		SecretScanningEnabled:         true,
		CodeScanningEnabled:           false,
		DependencyGraphEnabled:        true,
		SecurityAdvisoryEnabled:       false,
	}

	data, err := json.Marshal(ss)
	require.NoError(t, err)

	var decoded GitHubSecuritySettings
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.True(t, decoded.VulnerabilityAlertsEnabled)
	assert.True(t, decoded.SecretScanningEnabled)
	assert.False(t, decoded.CodeScanningEnabled)
}

func TestGitHubOrganizationInfo_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	org := GitHubOrganizationInfo{
		Login:       "test-org",
		ID: 12345,
		Name:        "Test Organization",
		Description: "A test org",
		PublicRepos: 10,
		PrivateRepos: 50,
		CreatedAt:   now,
		UpdatedAt:   now,
		Plan: &GitHubPlan{
			Name:         "enterprise",
			PrivateRepos: 999,
		},
		Members: []GitHubOrgMember{
			{Login: "alice", ID: 1, Role: "admin"},
			{Login: "bob", ID: 2, Role: "member"},
		},
	}

	data, err := json.Marshal(org)
	require.NoError(t, err)

	var decoded GitHubOrganizationInfo
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "test-org", decoded.Login)
	assert.NotNil(t, decoded.Plan)
	assert.Equal(t, "enterprise", decoded.Plan.Name)
	assert.Len(t, decoded.Members, 2)
}

func TestGitHubAccessSummary_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	summary := GitHubAccessSummary{
		TotalCollaborators:    5,
		TotalTeams:            3,
		TotalTeamMembers:      12,
		AdminUsers:            []string{"alice"},
		MaintainUsers:         []string{"bob"},
		PushUsers:             []string{"charlie", "dave"},
		ReadOnlyUsers:         []string{"eve"},
		AdminTeams:            []string{"security"},
		WriteTeams:            []string{"engineering"},
		ProtectedBranches:     []string{"main", "release"},
		ProtectedEnvironments: []string{"production", "staging"},
		SecurityFeatures: GitHubSecurityFeatureSummary{
			EnabledFeatures:  []string{"dependabot", "secret_scanning"},
			DisabledFeatures: []string{"code_scanning"},
			TotalFeatures:    3,
			EnabledCount:     2,
			SecurityScore:    0.67,
		},
	}

	data, err := json.Marshal(summary)
	require.NoError(t, err)

	var decoded GitHubAccessSummary
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 5, decoded.TotalCollaborators)
	assert.Len(t, decoded.AdminUsers, 1)
	assert.Len(t, decoded.ProtectedBranches, 2)
	assert.InDelta(t, 0.67, decoded.SecurityFeatures.SecurityScore, 0.001)
}

func TestGitHubTeam_WithParent(t *testing.T) {
	t.Parallel()
	team := GitHubTeam{
		ID: 100,
		Name: "Frontend",
		Slug: "frontend",
		Parent: &GitHubTeam{
			ID: 50,
			Name: "Engineering",
			Slug: "engineering",
		},
		Members: []GitHubTeamMember{
			{Login: "alice", ID: 1, Role: "maintainer"},
		},
	}

	data, err := json.Marshal(team)
	require.NoError(t, err)

	var decoded GitHubTeam
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.NotNil(t, decoded.Parent)
	assert.Equal(t, "Engineering", decoded.Parent.Name)
	assert.Len(t, decoded.Members, 1)
}
