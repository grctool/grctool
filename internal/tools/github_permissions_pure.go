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
	"fmt"
	"strings"

	"github.com/grctool/grctool/internal/models"
)

// FormatEnabled formats a boolean as an enabled/disabled string
func FormatEnabled(enabled bool) string {
	if enabled {
		return "✅ Enabled"
	}
	return "❌ Disabled"
}

// IsHigherPermission compares two GitHub permission levels
func IsHigherPermission(perm1, perm2 string) bool {
	permLevels := map[string]int{
		"admin":    5,
		"maintain": 4,
		"push":     3,
		"write":    3,
		"triage":   2,
		"pull":     1,
		"read":     1,
		"none":     0,
	}

	return permLevels[perm1] > permLevels[perm2]
}

// CanPushBasedOnPermission determines if a permission level allows push access
func CanPushBasedOnPermission(permission string) bool {
	pushPermissions := map[string]bool{
		"admin":    true,
		"maintain": true,
		"push":     true,
		"write":    true,
		"triage":   false,
		"pull":     false,
		"read":     false,
		"none":     false,
	}

	return pushPermissions[permission]
}

// CalculateRelevanceScore computes a relevance score for a GitHub access control matrix
func CalculateRelevanceScore(matrix *models.GitHubAccessControlMatrix) float64 {
	relevance := 0.5 // Base relevance

	// Higher relevance for repositories with more sophisticated access controls
	if len(matrix.Collaborators) > 0 {
		relevance += 0.1
	}
	if len(matrix.Teams) > 0 {
		relevance += 0.1
	}
	if len(matrix.AccessSummary.ProtectedBranches) > 0 {
		relevance += 0.1
	}
	if len(matrix.AccessSummary.ProtectedEnvironments) > 0 {
		relevance += 0.1
	}
	if matrix.AccessSummary.SecurityFeatures.SecurityScore > 0.5 {
		relevance += 0.1
	}

	if relevance > 1.0 {
		relevance = 1.0
	}

	return relevance
}

// ParseRepositoryOwnerAndName parses "owner/repo" format into separate parts
func ParseRepositoryOwnerAndName(repository string) (owner, repo string, err error) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("repository must be in format 'owner/repo', got: %s", repository)
	}
	return parts[0], parts[1], nil
}

// CalculateSecurityScore computes a security score based on GitHub security settings
func CalculateSecurityScore(settings models.GitHubSecuritySettings) float64 {
	score := 0.0
	total := 0.0

	// Vulnerability alerts
	total += 0.2
	if settings.VulnerabilityAlertsEnabled {
		score += 0.2
	}

	// Automated security fixes
	total += 0.2
	if settings.AutomatedSecurityFixesEnabled {
		score += 0.2
	}

	// Secret scanning
	total += 0.3
	if settings.SecretScanningEnabled {
		score += 0.3
	}

	// Code scanning
	total += 0.2
	if settings.CodeScanningEnabled {
		score += 0.2
	}

	// Dependency graph
	total += 0.1
	if settings.DependencyGraphEnabled {
		score += 0.1
	}

	if total > 0 {
		return score / total
	}
	return 0.0
}

// BuildSecurityFeatureSummary creates a summary of security features
func BuildSecurityFeatureSummary(settings models.GitHubSecuritySettings) models.GitHubSecurityFeatureSummary {
	enabled := []string{}
	disabled := []string{}

	if settings.VulnerabilityAlertsEnabled {
		enabled = append(enabled, "Vulnerability Alerts")
	} else {
		disabled = append(disabled, "Vulnerability Alerts")
	}

	if settings.AutomatedSecurityFixesEnabled {
		enabled = append(enabled, "Automated Security Fixes")
	} else {
		disabled = append(disabled, "Automated Security Fixes")
	}

	if settings.SecretScanningEnabled {
		enabled = append(enabled, "Secret Scanning")
	} else {
		disabled = append(disabled, "Secret Scanning")
	}

	if settings.CodeScanningEnabled {
		enabled = append(enabled, "Code Scanning")
	} else {
		disabled = append(disabled, "Code Scanning")
	}

	if settings.DependencyGraphEnabled {
		enabled = append(enabled, "Dependency Graph")
	} else {
		disabled = append(disabled, "Dependency Graph")
	}

	return models.GitHubSecurityFeatureSummary{
		SecurityScore:    CalculateSecurityScore(settings),
		EnabledFeatures:  enabled,
		DisabledFeatures: disabled,
		TotalFeatures:    len(enabled) + len(disabled),
		EnabledCount:     len(enabled),
	}
}

// FilterProtectedBranches filters branches to only return protected ones
func FilterProtectedBranches(branches []models.GitHubBranch) []string {
	protected := []string{} // Initialize as empty slice, not nil
	for _, branch := range branches {
		if branch.Protected {
			protected = append(protected, branch.Name)
		}
	}
	return protected
}

// CountCollaboratorsByPermission counts collaborators by permission level
func CountCollaboratorsByPermission(collaborators []models.GitHubCollaborator) map[string]int {
	counts := make(map[string]int)
	for _, collaborator := range collaborators {
		counts[collaborator.Permissions.Permission]++
	}
	return counts
}
