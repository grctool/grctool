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

//go:build e2e

package e2e

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGitHubPermissions_RealAPI tests GitHub permissions analysis with real API
func TestGitHubPermissions_RealAPI(t *testing.T) {
	// Skip if no auth available
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN required for E2E tests")
	}

	testRepo := os.Getenv("TEST_GITHUB_REPO")
	if testRepo == "" {
		testRepo = "octocat/Hello-World" // Default test repository
	}

	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   os.Getenv("GITHUB_TOKEN"),
					Repository: testRepo,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()

	// Test permissions analysis
	tool := tools.NewGitHubTool(cfg, log)
	result, source, err := tool.Execute(context.Background(), map[string]interface{}{
		"analysis_type": "permissions",
		"output_format": "detailed",
	})

	require.NoError(t, err, "GitHub permissions API should work with valid token")
	assert.NotEmpty(t, result)
	assert.NotNil(t, source)
	assert.Equal(t, "github", source.Type)

	// Validate that response contains permission information
	assert.Contains(t, strings.ToLower(result), "permission")
	t.Logf("GitHub permissions analysis successful for %s", testRepo)
}

// TestGitHubWorkflows_RealAPI tests GitHub Actions workflow analysis with real API
func TestGitHubWorkflows_RealAPI(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN required for E2E tests")
	}

	testRepo := os.Getenv("TEST_GITHUB_REPO")
	if testRepo == "" {
		testRepo = "actions/checkout" // Repository known to have workflows
	}

	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   os.Getenv("GITHUB_TOKEN"),
					Repository: testRepo,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()
	workflowTool := tools.NewGitHubWorkflowAnalyzer(cfg, log)

	if workflowTool == nil {
		t.Skip("GitHub workflow analyzer not available")
	}

	result, source, err := workflowTool.Execute(context.Background(), map[string]interface{}{
		"analysis_type": "security_analysis",
		"output_format": "markdown",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, source)
	assert.Equal(t, "github", source.Type)

	t.Logf("GitHub workflow analysis successful for %s", testRepo)
}

// TestGitHubSecurity_RealAPI tests security feature analysis with real API
func TestGitHubSecurity_RealAPI(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN required for E2E tests")
	}

	testRepo := os.Getenv("TEST_GITHUB_REPO")
	if testRepo == "" {
		testRepo = "github/docs" // Repository known to have security features
	}

	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   os.Getenv("GITHUB_TOKEN"),
					Repository: testRepo,
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()
	tool := tools.NewGitHubTool(cfg, log)

	result, source, err := tool.Execute(context.Background(), map[string]interface{}{
		"analysis_type": "security_features",
		"output_format": "detailed",
	})

	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, source)
	assert.Equal(t, "github", source.Type)

	// Validate that response contains security-related information
	resultLower := strings.ToLower(result)
	securityKeywords := []string{"security", "vulnerability", "dependabot", "code scanning"}
	found := false
	for _, keyword := range securityKeywords {
		if strings.Contains(resultLower, keyword) {
			found = true
			break
		}
	}
	assert.True(t, found, "Result should contain security-related information")

	t.Logf("GitHub security analysis successful for %s", testRepo)
}

// TestGitHubRateLimit_RealAPI tests rate limit handling with real API
func TestGitHubRateLimit_RealAPI(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN required for E2E tests")
	}

	if os.Getenv("TEST_RATE_LIMITS") == "" {
		t.Skip("TEST_RATE_LIMITS not enabled")
	}

	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   os.Getenv("GITHUB_TOKEN"),
					Repository: "octocat/Hello-World",
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()
	tool := tools.NewGitHubTool(cfg, log)

	// Make multiple requests quickly to test rate limit handling
	for i := 0; i < 3; i++ {
		start := time.Now()
		result, source, err := tool.Execute(context.Background(), map[string]interface{}{
			"analysis_type": "basic",
			"output_format": "json",
		})

		duration := time.Since(start)

		// Should handle rate limits gracefully
		if err != nil {
			assert.Contains(t, strings.ToLower(err.Error()), "rate limit")
			t.Logf("Rate limit handled correctly: %v", err)
		} else {
			assert.NotEmpty(t, result)
			assert.NotNil(t, source)
			t.Logf("Request %d completed in %v", i+1, duration)
		}

		// Small delay between requests
		time.Sleep(100 * time.Millisecond)
	}
}

// TestGitHubInvalidRepo_RealAPI tests error handling for invalid repository
func TestGitHubInvalidRepo_RealAPI(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN required for E2E tests")
	}

	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					APIToken:   os.Getenv("GITHUB_TOKEN"),
					Repository: "nonexistent/repository-does-not-exist",
				},
			},
		},
	}

	log, _ := logger.NewTestLogger()
	tool := tools.NewGitHubTool(cfg, log)

	_, _, err := tool.Execute(context.Background(), map[string]interface{}{
		"analysis_type": "basic",
	})

	// Should receive an error for nonexistent repository
	assert.Error(t, err)
	assert.Contains(t, strings.ToLower(err.Error()), "not found")

	t.Logf("Invalid repository error handled correctly: %v", err)
}

// TestGitHubMultipleRepositories_RealAPI tests analysis across multiple repositories
func TestGitHubMultipleRepositories_RealAPI(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN required for E2E tests")
	}

	if os.Getenv("TEST_MULTIPLE_REPOS") == "" {
		t.Skip("TEST_MULTIPLE_REPOS not enabled")
	}

	repositories := []string{
		"octocat/Hello-World",
		"github/docs",
		"actions/checkout",
	}

	log, _ := logger.NewTestLogger()

	for _, repo := range repositories {
		t.Run(repo, func(t *testing.T) {
			cfg := &config.Config{
				Evidence: config.EvidenceConfig{
					Tools: config.ToolsConfig{
						GitHub: config.GitHubToolConfig{
							Enabled:    true,
							APIToken:   os.Getenv("GITHUB_TOKEN"),
							Repository: repo,
						},
					},
				},
			}

			tool := tools.NewGitHubTool(cfg, log)
			result, source, err := tool.Execute(context.Background(), map[string]interface{}{
				"analysis_type": "basic",
				"output_format": "summary",
			})

			require.NoError(t, err)
			assert.NotEmpty(t, result)
			assert.NotNil(t, source)
			assert.Equal(t, "github", source.Type)

			t.Logf("Repository %s analysis successful", repo)
		})
	}
}
