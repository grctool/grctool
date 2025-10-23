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

package integration_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestVCRCassettes_Structure(t *testing.T) {
	// Test that VCR cassettes are properly structured and contain expected content

	cassetteDir := filepath.Join("..", "..", "test", "vcr_cassettes")

	t.Run("VCR Directory Exists", func(t *testing.T) {
		_, err := os.Stat(cassetteDir)
		assert.NoError(t, err, "VCR cassettes directory should exist")
	})

	expectedCassettes := []string{
		"github_permissions_full.yaml",
		"github_workflows_analysis.yaml",
		"github_reviews_extract.yaml",
		"github_security_comprehensive.yaml",
		"evidence_et96_user_access.yaml",
		"evidence_cross_tool.yaml",
	}

	for _, cassetteName := range expectedCassettes {
		t.Run("Cassette_"+strings.TrimSuffix(cassetteName, ".yaml"), func(t *testing.T) {
			cassettePath := filepath.Join(cassetteDir, cassetteName)

			// Check cassette exists
			info, err := os.Stat(cassettePath)
			require.NoError(t, err, "Cassette %s should exist", cassetteName)

			// Check cassette is not empty
			assert.Greater(t, info.Size(), int64(100), "Cassette %s should not be empty", cassetteName)

			// Read and validate cassette content
			content, err := os.ReadFile(cassettePath)
			require.NoError(t, err, "Should be able to read cassette %s", cassetteName)

			contentStr := string(content)

			// Validate YAML structure
			assert.Contains(t, contentStr, "interactions:", "Cassette should contain interactions")
			assert.Contains(t, contentStr, "request:", "Cassette should contain requests")
			assert.Contains(t, contentStr, "response:", "Cassette should contain responses")
			assert.Contains(t, contentStr, "version:", "Cassette should have version")

			// Validate sensitive data is redacted
			assert.NotContains(t, contentStr, "token:", "Raw tokens should be redacted")
			assert.Contains(t, contentStr, "[REDACTED]", "Should contain redaction markers")

			// Validate GitHub API structure
			if strings.Contains(cassetteName, "github") {
				assert.Contains(t, contentStr, "api.github.com", "Should contain GitHub API URLs")
				assert.Contains(t, contentStr, "application/vnd.github.v3+json", "Should contain GitHub API headers")
			}

			t.Logf("Cassette %s is valid (%d bytes)", cassetteName, info.Size())
		})
	}
}

func TestVCRCassettes_ContentValidation(t *testing.T) {
	// Test specific content expectations for different cassette types

	cassetteDir := filepath.Join("..", "..", "test", "vcr_cassettes")

	testCases := []struct {
		cassette        string
		expectedContent []string
		description     string
	}{
		{
			cassette: "github_permissions_full.yaml",
			expectedContent: []string{
				"admin permissions",
				"branch protection",
				"access control",
			},
			description: "GitHub permissions cassette should contain access control content",
		},
		{
			cassette: "github_workflows_analysis.yaml",
			expectedContent: []string{
				"ci/cd",
				"security scan",
				"automation",
			},
			description: "GitHub workflows cassette should contain CI/CD and security content",
		},
		{
			cassette: "github_reviews_extract.yaml",
			expectedContent: []string{
				"review",
				"approval",
				"code review",
			},
			description: "GitHub reviews cassette should contain review process content",
		},
		{
			cassette: "evidence_et96_user_access.yaml",
			expectedContent: []string{
				"ET96",
				"user access control",
				"RBAC",
				"authentication",
			},
			description: "ET96 evidence cassette should contain user access control content",
		},
		{
			cassette: "evidence_cross_tool.yaml",
			expectedContent: []string{
				"security policy",
				"change management",
				"data protection",
				"compliance",
			},
			description: "Cross-tool evidence cassette should contain multiple control types",
		},
	}

	for _, tc := range testCases {
		t.Run(strings.TrimSuffix(tc.cassette, ".yaml"), func(t *testing.T) {
			cassettePath := filepath.Join(cassetteDir, tc.cassette)

			content, err := os.ReadFile(cassettePath)
			require.NoError(t, err, "Should be able to read %s", tc.cassette)

			contentStr := strings.ToLower(string(content))

			for _, expectedContent := range tc.expectedContent {
				assert.Contains(t, contentStr, strings.ToLower(expectedContent),
					"Cassette %s should contain '%s'", tc.cassette, expectedContent)
			}

			t.Logf("%s: %s", tc.cassette, tc.description)
		})
	}
}

func TestVCRCassettes_SecurityValidation(t *testing.T) {
	// Test that cassettes properly redact sensitive information

	cassetteDir := filepath.Join("..", "..", "test", "vcr_cassettes")

	entries, err := os.ReadDir(cassetteDir)
	require.NoError(t, err, "Should be able to read cassette directory")

	// Patterns that indicate actual secrets (not words in discussion of security)
	secretTokenPatterns := []string{
		"ghp_",    // GitHub personal access tokens
		"gho_",    // GitHub OAuth tokens
		"ghu_",    // GitHub user-to-server tokens
		"ghs_",    // GitHub server-to-server tokens
		"Bearer ", // Bearer tokens (should be redacted)
	}

	for _, entry := range entries {
		if !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}

		t.Run("Security_"+strings.TrimSuffix(entry.Name(), ".yaml"), func(t *testing.T) {
			cassettePath := filepath.Join(cassetteDir, entry.Name())

			content, err := os.ReadFile(cassettePath)
			require.NoError(t, err, "Should be able to read %s", entry.Name())

			contentStr := string(content)

			// Check for actual token patterns (not words in body content)
			for _, pattern := range secretTokenPatterns {
				if strings.Contains(contentStr, pattern) {
					// Check if it's properly redacted
					lines := strings.Split(contentStr, "\n")
					for _, line := range lines {
						if strings.Contains(line, pattern) && !strings.Contains(line, "[REDACTED]") {
							t.Errorf("Cassette %s contains unredacted token pattern: %s in line: %s",
								entry.Name(), pattern, strings.TrimSpace(line))
						}
					}
				}
			}

			// Ensure redaction markers are present (tokens should be redacted)
			assert.Contains(t, contentStr, "[REDACTED]",
				"Cassette %s should contain redaction markers", entry.Name())

			t.Logf("Security validation passed for %s", entry.Name())
		})
	}
}

func TestVCRCassettes_InteractionCounts(t *testing.T) {
	// Test that cassettes contain reasonable numbers of interactions

	cassetteDir := filepath.Join("..", "..", "test", "vcr_cassettes")

	expectedInteractions := map[string]struct {
		min int
		max int
	}{
		"github_permissions_full.yaml":       {min: 1, max: 10},
		"github_workflows_analysis.yaml":     {min: 1, max: 10},
		"github_reviews_extract.yaml":        {min: 1, max: 10},
		"github_security_comprehensive.yaml": {min: 3, max: 15},
		"evidence_et96_user_access.yaml":     {min: 1, max: 10},
		"evidence_cross_tool.yaml":           {min: 2, max: 15},
	}

	for cassetteName, expected := range expectedInteractions {
		t.Run(strings.TrimSuffix(cassetteName, ".yaml"), func(t *testing.T) {
			cassettePath := filepath.Join(cassetteDir, cassetteName)

			content, err := os.ReadFile(cassettePath)
			require.NoError(t, err, "Should be able to read %s", cassetteName)

			contentStr := string(content)

			// Count interactions by counting "- request:" occurrences
			interactionCount := strings.Count(contentStr, "- request:")

			assert.GreaterOrEqual(t, interactionCount, expected.min,
				"Cassette %s should have at least %d interactions", cassetteName, expected.min)
			assert.LessOrEqual(t, interactionCount, expected.max,
				"Cassette %s should have at most %d interactions", cassetteName, expected.max)

			t.Logf("Cassette %s has %d interactions (expected %d-%d)",
				cassetteName, interactionCount, expected.min, expected.max)
		})
	}
}
