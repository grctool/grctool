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

package models

import (
	"time"
)

// GitHubAccessControlMatrix represents the complete access control matrix for a repository
type GitHubAccessControlMatrix struct {
	Repository       GitHubRepositoryInfo    `json:"repository"`
	Collaborators    []GitHubCollaborator    `json:"collaborators"`
	Teams            []GitHubTeam            `json:"teams"`
	Branches         []GitHubBranch          `json:"branches"`
	Environments     []GitHubEnvironment     `json:"environments"`
	SecuritySettings GitHubSecuritySettings  `json:"security_settings"`
	OrganizationInfo *GitHubOrganizationInfo `json:"organization_info,omitempty"`
	ExtractedAt      time.Time               `json:"extracted_at"`
	AccessSummary    GitHubAccessSummary     `json:"access_summary"`
}

// GitHubRepositoryInfo contains basic repository information
type GitHubRepositoryInfo struct {
	Name          string    `json:"name"`
	FullName      string    `json:"full_name"`
	Owner         string    `json:"owner"`
	Private       bool      `json:"private"`
	Archived      bool      `json:"archived"`
	Disabled      bool      `json:"disabled"`
	DefaultBranch string    `json:"default_branch"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	HTMLURL       string    `json:"html_url"`
}

// GitHubCollaborator represents a user with direct access to the repository
type GitHubCollaborator struct {
	Login               string             `json:"login"`
	ID                  int                `json:"id"`
	NodeID              string             `json:"node_id"`
	AvatarURL           string             `json:"avatar_url"`
	HTMLURL             string             `json:"html_url"`
	Type                string             `json:"type"` // User, Bot
	SiteAdmin           bool               `json:"site_admin"`
	Permissions         GitHubPermissions  `json:"permissions"`
	DetailedPermissions *GitHubPermissions `json:"detailed_permissions,omitempty"`
	RoleName            string             `json:"role_name,omitempty"`
}

// GitHubPermissions represents detailed permissions for a user or team
type GitHubPermissions struct {
	Permission string `json:"permission"` // admin, maintain, push, triage, pull
	Admin      bool   `json:"admin"`
	Maintain   bool   `json:"maintain"`
	Push       bool   `json:"push"`
	Triage     bool   `json:"triage"`
	Pull       bool   `json:"pull"`
}

// GitHubTeam represents a team with access to the repository
type GitHubTeam struct {
	ID          int                `json:"id"`
	NodeID      string             `json:"node_id"`
	Name        string             `json:"name"`
	Slug        string             `json:"slug"`
	Description string             `json:"description"`
	Privacy     string             `json:"privacy"`
	Permission  string             `json:"permission"`
	HTMLURL     string             `json:"html_url"`
	Members     []GitHubTeamMember `json:"members"`
	MembersURL  string             `json:"members_url"`
	Parent      *GitHubTeam        `json:"parent,omitempty"`
}

// GitHubTeamMember represents a member of a team
type GitHubTeamMember struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	NodeID    string `json:"node_id"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"`
	SiteAdmin bool   `json:"site_admin"`
	Role      string `json:"role,omitempty"` // member, maintainer
}

// GitHubBranch represents a repository branch with protection settings
type GitHubBranch struct {
	Name       string                  `json:"name"`
	Commit     GitHubCommitRef         `json:"commit"`
	Protected  bool                    `json:"protected"`
	Protection *GitHubBranchProtection `json:"protection,omitempty"`
}

// GitHubCommitRef represents a commit reference
type GitHubCommitRef struct {
	SHA string `json:"sha"`
	URL string `json:"url"`
}

// GitHubBranchProtection represents branch protection rules
type GitHubBranchProtection struct {
	URL                            string                               `json:"url"`
	Enabled                        bool                                 `json:"enabled"`
	RequiredStatusChecks           *GitHubRequiredStatusChecks          `json:"required_status_checks"`
	EnforceAdmins                  GitHubEnforceAdmins                  `json:"enforce_admins"`
	RequiredPullRequestReviews     *GitHubRequiredPullRequestReviews    `json:"required_pull_request_reviews"`
	Restrictions                   *GitHubRestrictions                  `json:"restrictions"`
	RequiredLinearHistory          GitHubRequiredLinearHistory          `json:"required_linear_history"`
	AllowForcePushes               GitHubAllowForcePushes               `json:"allow_force_pushes"`
	AllowDeletions                 GitHubAllowDeletions                 `json:"allow_deletions"`
	RequiredConversationResolution GitHubRequiredConversationResolution `json:"required_conversation_resolution"`
}

// GitHubRequiredStatusChecks represents required status checks configuration
type GitHubRequiredStatusChecks struct {
	URL              string   `json:"url"`
	Strict           bool     `json:"strict"`
	Contexts         []string `json:"contexts"`
	ContextsURL      string   `json:"contexts_url"`
	EnforcementLevel string   `json:"enforcement_level"`
}

// GitHubRequiredPullRequestReviews represents required PR review configuration
type GitHubRequiredPullRequestReviews struct {
	URL                          string                       `json:"url"`
	DismissStaleReviews          bool                         `json:"dismiss_stale_reviews"`
	RequireCodeOwnerReviews      bool                         `json:"require_code_owner_reviews"`
	RequiredApprovingReviewCount int                          `json:"required_approving_review_count"`
	DismissalRestrictions        *GitHubDismissalRestrictions `json:"dismissal_restrictions"`
	RequireLastPushApproval      bool                         `json:"require_last_push_approval"`
}

// GitHubDismissalRestrictions represents who can dismiss PR reviews
type GitHubDismissalRestrictions struct {
	URL   string       `json:"url"`
	Users []GitHubUser `json:"users"`
	Teams []GitHubTeam `json:"teams"`
	Apps  []GitHubApp  `json:"apps"`
}

// GitHubRestrictions represents push restrictions
type GitHubRestrictions struct {
	URL   string       `json:"url"`
	Users []GitHubUser `json:"users"`
	Teams []GitHubTeam `json:"teams"`
	Apps  []GitHubApp  `json:"apps"`
}

// GitHubUser represents a GitHub user
type GitHubUser struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	NodeID    string `json:"node_id"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"`
	SiteAdmin bool   `json:"site_admin"`
}

// GitHubApp represents a GitHub app
type GitHubApp struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Slug   string `json:"slug"`
	NodeID string `json:"node_id"`
}

// GitHubEnforceAdmins represents admin enforcement settings
type GitHubEnforceAdmins struct {
	URL     string `json:"url"`
	Enabled bool   `json:"enabled"`
}

// GitHubRequiredLinearHistory represents linear history requirement
type GitHubRequiredLinearHistory struct {
	Enabled bool `json:"enabled"`
}

// GitHubAllowForcePushes represents force push allowance
type GitHubAllowForcePushes struct {
	Enabled bool `json:"enabled"`
}

// GitHubAllowDeletions represents deletion allowance
type GitHubAllowDeletions struct {
	Enabled bool `json:"enabled"`
}

// GitHubRequiredConversationResolution represents conversation resolution requirement
type GitHubRequiredConversationResolution struct {
	Enabled bool `json:"enabled"`
}

// GitHubEnvironment represents a deployment environment
type GitHubEnvironment struct {
	ID              string                        `json:"id"`
	Name            string                        `json:"name"`
	ProtectionRules []GitHubEnvironmentProtection `json:"protection_rules"`
	CreatedAt       time.Time                     `json:"created_at"`
	UpdatedAt       time.Time                     `json:"updated_at"`
}

// GitHubEnvironmentProtection represents environment protection rules
type GitHubEnvironmentProtection struct {
	Type              string                   `json:"type"`
	RequiredReviewers []GitHubRequiredReviewer `json:"required_reviewers,omitempty"`
	WaitTimer         int                      `json:"wait_timer,omitempty"`
}

// GitHubRequiredReviewer represents a required reviewer for environment deployments
type GitHubRequiredReviewer struct {
	Login string `json:"login,omitempty"`
	Name  string `json:"name,omitempty"`
	Email string `json:"email,omitempty"`
	Slug  string `json:"slug,omitempty"` // For teams
	Type  string `json:"type"`           // User or Team
}

// GitHubSecuritySettings represents repository security configuration
type GitHubSecuritySettings struct {
	VulnerabilityAlertsEnabled    bool `json:"vulnerability_alerts_enabled"`
	AutomatedSecurityFixesEnabled bool `json:"automated_security_fixes_enabled"`
	SecretScanningEnabled         bool `json:"secret_scanning_enabled"`
	CodeScanningEnabled           bool `json:"code_scanning_enabled"`
	DependencyGraphEnabled        bool `json:"dependency_graph_enabled"`
	SecurityAdvisoryEnabled       bool `json:"security_advisory_enabled"`
}

// GitHubOrganizationInfo represents organization-level information
type GitHubOrganizationInfo struct {
	Login             string            `json:"login"`
	ID                int               `json:"id"`
	NodeID            string            `json:"node_id"`
	Name              string            `json:"name"`
	Company           string            `json:"company"`
	Blog              string            `json:"blog"`
	Location          string            `json:"location"`
	Email             string            `json:"email"`
	TwitterUsername   string            `json:"twitter_username"`
	Description       string            `json:"description"`
	PublicRepos       int               `json:"public_repos"`
	PrivateRepos      int               `json:"private_repos"`
	OwnedPrivateRepos int               `json:"owned_private_repos"`
	TotalPrivateRepos int               `json:"total_private_repos"`
	PublicGists       int               `json:"public_gists"`
	PrivateGists      int               `json:"private_gists"`
	Followers         int               `json:"followers"`
	Following         int               `json:"following"`
	HTMLURL           string            `json:"html_url"`
	CreatedAt         time.Time         `json:"created_at"`
	UpdatedAt         time.Time         `json:"updated_at"`
	Type              string            `json:"type"`
	Plan              *GitHubPlan       `json:"plan,omitempty"`
	Members           []GitHubOrgMember `json:"members,omitempty"`
}

// GitHubPlan represents the organization's GitHub plan
type GitHubPlan struct {
	Name          string `json:"name"`
	Space         int    `json:"space"`
	PrivateRepos  int    `json:"private_repos"`
	Collaborators int    `json:"collaborators"`
}

// GitHubOrgMember represents an organization member
type GitHubOrgMember struct {
	Login     string `json:"login"`
	ID        int    `json:"id"`
	NodeID    string `json:"node_id"`
	AvatarURL string `json:"avatar_url"`
	HTMLURL   string `json:"html_url"`
	Type      string `json:"type"`
	SiteAdmin bool   `json:"site_admin"`
	Role      string `json:"role,omitempty"` // member, admin, billing_manager, etc.
}

// GitHubAccessSummary provides a high-level summary of repository access
type GitHubAccessSummary struct {
	TotalCollaborators    int                          `json:"total_collaborators"`
	TotalTeams            int                          `json:"total_teams"`
	TotalTeamMembers      int                          `json:"total_team_members"`
	AdminUsers            []string                     `json:"admin_users"`
	MaintainUsers         []string                     `json:"maintain_users"`
	PushUsers             []string                     `json:"push_users"`
	TriageUsers           []string                     `json:"triage_users"`
	ReadOnlyUsers         []string                     `json:"read_only_users"`
	AdminTeams            []string                     `json:"admin_teams"`
	WriteTeams            []string                     `json:"write_teams"`
	ReadTeams             []string                     `json:"read_teams"`
	ProtectedBranches     []string                     `json:"protected_branches"`
	ProtectedEnvironments []string                     `json:"protected_environments"`
	SecurityFeatures      GitHubSecurityFeatureSummary `json:"security_features"`
}

// GitHubSecurityFeatureSummary summarizes enabled security features
type GitHubSecurityFeatureSummary struct {
	EnabledFeatures  []string `json:"enabled_features"`
	DisabledFeatures []string `json:"disabled_features"`
	TotalFeatures    int      `json:"total_features"`
	EnabledCount     int      `json:"enabled_count"`
	SecurityScore    float64  `json:"security_score"` // 0.0 to 1.0
}

// GitHubDeploymentAccess represents users/teams with deployment access
type GitHubDeploymentAccess struct {
	Environment       string                   `json:"environment"`
	RequiredReviewers []GitHubRequiredReviewer `json:"required_reviewers"`
	RequiredApprovals int                      `json:"required_approvals,omitempty"`
	WaitTimer         int                      `json:"wait_timer,omitempty"`
	RestrictPushes    bool                     `json:"restrict_pushes"`
	AllowedPushers    []string                 `json:"allowed_pushers,omitempty"`
}

// GitHubPermissionMatrix represents a flattened view of all permissions
type GitHubPermissionMatrix struct {
	Repository      string                   `json:"repository"`
	UserAccess      []GitHubUserAccess       `json:"user_access"`
	TeamAccess      []GitHubTeamAccess       `json:"team_access"`
	BranchRules     []GitHubBranchAccessRule `json:"branch_rules"`
	DeploymentRules []GitHubDeploymentAccess `json:"deployment_rules"`
	GeneratedAt     time.Time                `json:"generated_at"`
}

// GitHubUserAccess represents consolidated user access information
type GitHubUserAccess struct {
	Username        string   `json:"username"`
	UserID          int      `json:"user_id"`
	DirectAccess    string   `json:"direct_access"`    // admin, maintain, push, triage, pull, none
	TeamMemberships []string `json:"team_memberships"` // List of teams user belongs to
	EffectiveAccess string   `json:"effective_access"` // Highest permission level
	CanDeploy       []string `json:"can_deploy"`       // List of environments user can deploy to
	CanPushTo       []string `json:"can_push_to"`      // List of branches user can push to
	IsOrgAdmin      bool     `json:"is_org_admin"`
	IsSiteAdmin     bool     `json:"is_site_admin"`
}

// GitHubTeamAccess represents team access information
type GitHubTeamAccess struct {
	TeamName    string   `json:"team_name"`
	TeamID      int      `json:"team_id"`
	TeamSlug    string   `json:"team_slug"`
	Permission  string   `json:"permission"`
	MemberCount int      `json:"member_count"`
	Members     []string `json:"members"`
}

// GitHubBranchAccessRule represents branch-level access rules
type GitHubBranchAccessRule struct {
	BranchName                    string   `json:"branch_name"`
	IsProtected                   bool     `json:"is_protected"`
	RequiredReviews               int      `json:"required_reviews"`
	RequireCodeOwnerReviews       bool     `json:"require_code_owner_reviews"`
	DismissStaleReviews           bool     `json:"dismiss_stale_reviews"`
	RequiredStatusChecks          []string `json:"required_status_checks"`
	EnforceAdmins                 bool     `json:"enforce_admins"`
	RestrictPushes                bool     `json:"restrict_pushes"`
	AllowedPushers                []string `json:"allowed_pushers,omitempty"`
	AllowForcePushes              bool     `json:"allow_force_pushes"`
	AllowDeletions                bool     `json:"allow_deletions"`
	RequireLinearHistory          bool     `json:"require_linear_history"`
	RequireConversationResolution bool     `json:"require_conversation_resolution"`
}
