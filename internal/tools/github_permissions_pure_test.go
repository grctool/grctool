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

package tools

import (
	"testing"

	"github.com/grctool/grctool/internal/models"
	"github.com/stretchr/testify/assert"
)

func TestFormatEnabled(t *testing.T) {
	tests := map[string]struct {
		enabled  bool
		expected string
	}{
		"enabled": {
			enabled:  true,
			expected: "✅ Enabled",
		},
		"disabled": {
			enabled:  false,
			expected: "❌ Disabled",
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := FormatEnabled(tt.enabled)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestIsHigherPermission(t *testing.T) {
	tests := map[string]struct {
		perm1    string
		perm2    string
		expected bool
	}{
		"admin higher than push": {
			perm1:    "admin",
			perm2:    "push",
			expected: true,
		},
		"push not higher than admin": {
			perm1:    "push",
			perm2:    "admin",
			expected: false,
		},
		"write equals push": {
			perm1:    "write",
			perm2:    "push",
			expected: false,
		},
		"maintain higher than triage": {
			perm1:    "maintain",
			perm2:    "triage",
			expected: true,
		},
		"read not higher than pull": {
			perm1:    "read",
			perm2:    "pull",
			expected: false,
		},
		"unknown permissions default to zero": {
			perm1:    "unknown",
			perm2:    "admin",
			expected: false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := IsHigherPermission(tt.perm1, tt.perm2)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCanPushBasedOnPermission(t *testing.T) {
	tests := map[string]struct {
		permission string
		canPush    bool
	}{
		"admin can push": {
			permission: "admin",
			canPush:    true,
		},
		"maintain can push": {
			permission: "maintain",
			canPush:    true,
		},
		"push can push": {
			permission: "push",
			canPush:    true,
		},
		"write can push": {
			permission: "write",
			canPush:    true,
		},
		"triage cannot push": {
			permission: "triage",
			canPush:    false,
		},
		"pull cannot push": {
			permission: "pull",
			canPush:    false,
		},
		"read cannot push": {
			permission: "read",
			canPush:    false,
		},
		"none cannot push": {
			permission: "none",
			canPush:    false,
		},
		"unknown permission cannot push": {
			permission: "unknown",
			canPush:    false,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := CanPushBasedOnPermission(tt.permission)
			assert.Equal(t, tt.canPush, result)
		})
	}
}

func TestParseRepositoryOwnerAndName(t *testing.T) {
	tests := map[string]struct {
		repository    string
		expectedOwner string
		expectedRepo  string
		expectError   bool
	}{
		"valid repository": {
			repository:    "octocat/Hello-World",
			expectedOwner: "octocat",
			expectedRepo:  "Hello-World",
			expectError:   false,
		},
		"organization repository": {
			repository:    "7thsense/grctool",
			expectedOwner: "7thsense",
			expectedRepo:  "grctool",
			expectError:   false,
		},
		"missing slash": {
			repository:  "invalid-repo",
			expectError: true,
		},
		"too many parts": {
			repository:  "owner/repo/extra",
			expectError: true,
		},
		"empty string": {
			repository:  "",
			expectError: true,
		},
		"only slash": {
			repository:  "/",
			expectError: true,
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			owner, repo, err := ParseRepositoryOwnerAndName(tt.repository)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "repository must be in format 'owner/repo'")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedOwner, owner)
				assert.Equal(t, tt.expectedRepo, repo)
			}
		})
	}
}

func TestCalculateSecurityScore(t *testing.T) {
	tests := map[string]struct {
		settings models.GitHubSecuritySettings
		expected float64
	}{
		"all features enabled": {
			settings: models.GitHubSecuritySettings{
				VulnerabilityAlertsEnabled:    true,
				AutomatedSecurityFixesEnabled: true,
				SecretScanningEnabled:         true,
				CodeScanningEnabled:           true,
				DependencyGraphEnabled:        true,
			},
			expected: 1.0,
		},
		"no features enabled": {
			settings: models.GitHubSecuritySettings{
				VulnerabilityAlertsEnabled:    false,
				AutomatedSecurityFixesEnabled: false,
				SecretScanningEnabled:         false,
				CodeScanningEnabled:           false,
				DependencyGraphEnabled:        false,
			},
			expected: 0.0,
		},
		"only secret scanning enabled": {
			settings: models.GitHubSecuritySettings{
				VulnerabilityAlertsEnabled:    false,
				AutomatedSecurityFixesEnabled: false,
				SecretScanningEnabled:         true,
				CodeScanningEnabled:           false,
				DependencyGraphEnabled:        false,
			},
			expected: 0.3,
		},
		"vulnerability alerts and dependency graph": {
			settings: models.GitHubSecuritySettings{
				VulnerabilityAlertsEnabled:    true,
				AutomatedSecurityFixesEnabled: false,
				SecretScanningEnabled:         false,
				CodeScanningEnabled:           false,
				DependencyGraphEnabled:        true,
			},
			expected: 0.3, // 0.2 + 0.1
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := CalculateSecurityScore(tt.settings)
			assert.InDelta(t, tt.expected, result, 0.001) // Allow small floating point differences
		})
	}
}

func TestBuildSecurityFeatureSummary(t *testing.T) {
	t.Run("builds correct summary", func(t *testing.T) {
		settings := models.GitHubSecuritySettings{
			VulnerabilityAlertsEnabled:    true,
			AutomatedSecurityFixesEnabled: false,
			SecretScanningEnabled:         true,
			CodeScanningEnabled:           false,
			DependencyGraphEnabled:        true,
		}

		result := BuildSecurityFeatureSummary(settings)

		assert.Contains(t, result.EnabledFeatures, "Vulnerability Alerts")
		assert.Contains(t, result.EnabledFeatures, "Secret Scanning")
		assert.Contains(t, result.EnabledFeatures, "Dependency Graph")
		assert.Contains(t, result.DisabledFeatures, "Automated Security Fixes")
		assert.Contains(t, result.DisabledFeatures, "Code Scanning")
		assert.Equal(t, 5, result.TotalFeatures)
		assert.Equal(t, 3, result.EnabledCount)
		assert.InDelta(t, 0.6, result.SecurityScore, 0.001) // 0.2 + 0.3 + 0.1
	})
}

func TestFilterProtectedBranches(t *testing.T) {
	tests := map[string]struct {
		branches []models.GitHubBranch
		expected []string
	}{
		"no branches": {
			branches: []models.GitHubBranch{},
			expected: []string{},
		},
		"no protected branches": {
			branches: []models.GitHubBranch{
				{Name: "main", Protected: false},
				{Name: "develop", Protected: false},
			},
			expected: []string{},
		},
		"some protected branches": {
			branches: []models.GitHubBranch{
				{Name: "main", Protected: true},
				{Name: "develop", Protected: false},
				{Name: "staging", Protected: true},
			},
			expected: []string{"main", "staging"},
		},
		"all protected branches": {
			branches: []models.GitHubBranch{
				{Name: "main", Protected: true},
				{Name: "staging", Protected: true},
			},
			expected: []string{"main", "staging"},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := FilterProtectedBranches(tt.branches)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCountCollaboratorsByPermission(t *testing.T) {
	tests := map[string]struct {
		collaborators []models.GitHubCollaborator
		expected      map[string]int
	}{
		"no collaborators": {
			collaborators: []models.GitHubCollaborator{},
			expected:      map[string]int{},
		},
		"single collaborator": {
			collaborators: []models.GitHubCollaborator{
				{Login: "user1", Permissions: models.GitHubPermissions{Permission: "admin"}},
			},
			expected: map[string]int{
				"admin": 1,
			},
		},
		"multiple collaborators with different permissions": {
			collaborators: []models.GitHubCollaborator{
				{Login: "admin1", Permissions: models.GitHubPermissions{Permission: "admin"}},
				{Login: "admin2", Permissions: models.GitHubPermissions{Permission: "admin"}},
				{Login: "dev1", Permissions: models.GitHubPermissions{Permission: "push"}},
				{Login: "dev2", Permissions: models.GitHubPermissions{Permission: "push"}},
				{Login: "dev3", Permissions: models.GitHubPermissions{Permission: "push"}},
				{Login: "reader1", Permissions: models.GitHubPermissions{Permission: "pull"}},
			},
			expected: map[string]int{
				"admin": 2,
				"push":  3,
				"pull":  1,
			},
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := CountCollaboratorsByPermission(tt.collaborators)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCalculateRelevanceScore(t *testing.T) {
	tests := map[string]struct {
		matrix   *models.GitHubAccessControlMatrix
		expected float64
	}{
		"empty matrix": {
			matrix: &models.GitHubAccessControlMatrix{
				AccessSummary: models.GitHubAccessSummary{
					SecurityFeatures: models.GitHubSecurityFeatureSummary{
						SecurityScore: 0.0,
					},
				},
			},
			expected: 0.5, // Base relevance
		},
		"matrix with collaborators": {
			matrix: &models.GitHubAccessControlMatrix{
				Collaborators: []models.GitHubCollaborator{
					{Login: "user1"},
				},
				AccessSummary: models.GitHubAccessSummary{
					SecurityFeatures: models.GitHubSecurityFeatureSummary{
						SecurityScore: 0.0,
					},
				},
			},
			expected: 0.6, // Base + 0.1 for collaborators
		},
		"matrix with high security score": {
			matrix: &models.GitHubAccessControlMatrix{
				AccessSummary: models.GitHubAccessSummary{
					SecurityFeatures: models.GitHubSecurityFeatureSummary{
						SecurityScore: 0.8,
					},
				},
			},
			expected: 0.6, // Base + 0.1 for high security score
		},
		"full featured matrix": {
			matrix: &models.GitHubAccessControlMatrix{
				Collaborators: []models.GitHubCollaborator{{Login: "user1"}},
				Teams:         []models.GitHubTeam{{Name: "team1"}},
				AccessSummary: models.GitHubAccessSummary{
					ProtectedBranches:     []string{"main"},
					ProtectedEnvironments: []string{"prod"},
					SecurityFeatures: models.GitHubSecurityFeatureSummary{
						SecurityScore: 0.9,
					},
				},
			},
			expected: 1.0, // Capped at 1.0 (0.5 + 0.1 + 0.1 + 0.1 + 0.1 + 0.1)
		},
	}

	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			result := CalculateRelevanceScore(tt.matrix)
			assert.InDelta(t, tt.expected, result, 0.001)
		})
	}
}
