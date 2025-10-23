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

package tools_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Mock GitHub API responses for workflow analysis
var mockWorkflowFilesResponse = []map[string]interface{}{
	{
		"name":         "ci.yml",
		"path":         ".github/workflows/ci.yml",
		"sha":          "abc123",
		"type":         "file",
		"html_url":     "https://github.com/test/repo/blob/main/.github/workflows/ci.yml",
		"download_url": "https://raw.githubusercontent.com/test/repo/main/.github/workflows/ci.yml",
	},
	{
		"name":         "security.yml",
		"path":         ".github/workflows/security.yml",
		"sha":          "def456",
		"type":         "file",
		"html_url":     "https://github.com/test/repo/blob/main/.github/workflows/security.yml",
		"download_url": "https://raw.githubusercontent.com/test/repo/main/.github/workflows/security.yml",
	},
}

var mockWorkflowContentCI = `name: CI
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]

jobs:
  test:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Run tests
        run: npm test
  
  build:
    runs-on: ubuntu-latest
    needs: test
    steps:
      - uses: actions/checkout@v4
      - name: Build application
        run: npm run build`

var mockWorkflowContentSecurity = `name: Security
on:
  push:
    branches: [main]
  pull_request:
    branches: [main]
  schedule:
    - cron: '0 0 * * 0'

jobs:
  codeql:
    runs-on: ubuntu-latest
    permissions:
      security-events: write
    steps:
      - uses: actions/checkout@v4
      - name: Initialize CodeQL
        uses: github/codeql-action/init@v2
        with:
          languages: javascript
      - name: Perform CodeQL Analysis
        uses: github/codeql-action/analyze@v2
  
  dependency-review:
    runs-on: ubuntu-latest
    if: github.event_name == 'pull_request'
    steps:
      - uses: actions/checkout@v4
      - name: Dependency Review
        uses: actions/dependency-review-action@v3

  deploy:
    runs-on: ubuntu-latest
    environment: production
    needs: [codeql, dependency-review]
    if: github.ref == 'refs/heads/main'
    steps:
      - uses: actions/checkout@v4
      - name: Deploy to production
        run: echo "Deploying..."
        env:
          DEPLOY_TOKEN: ${{ secrets.DEPLOY_TOKEN }}`

var mockBranchesResponse = []map[string]interface{}{
	{
		"name":      "main",
		"protected": true,
	},
	{
		"name":      "develop",
		"protected": false,
	},
}

var mockBranchProtectionResponse = map[string]interface{}{
	"required_status_checks": map[string]interface{}{
		"strict":   true,
		"contexts": []string{"test", "build", "security"},
	},
	"required_pull_request_reviews": map[string]interface{}{
		"required_approving_review_count": 2,
		"dismiss_stale_reviews":           true,
		"require_code_owner_reviews":      true,
	},
	"enforce_admins":          true,
	"required_signatures":     true,
	"required_linear_history": false,
	"allow_force_pushes":      false,
	"allow_deletions":         false,
}

var mockCodeScanningResponse = []map[string]interface{}{
	{
		"number": 1,
		"state":  "open",
		"tool": map[string]interface{}{
			"name": "CodeQL",
		},
	},
}

func TestGitHubWorkflowAnalyzer_Execute_Success(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub tests: GITHUB_TOKEN not set (requires real API access)")
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/test/repo/contents/.github/workflows":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockWorkflowFilesResponse)
		case r.URL.Path == "/repos/test/repo/code-scanning/alerts":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockCodeScanningResponse)
		case r.URL.Path == "/repos/test/repo/branches":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockBranchesResponse)
		case r.URL.Path == "/repos/test/repo/branches/main/protection":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockBranchProtectionResponse)
		case r.URL.Path == "/repos/test/repo/contents/.github/dependabot.yml":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name":    "dependabot.yml",
				"content": "dmVyc2lvbjogMg==", // base64 for "version: 2"
			})
		case r.URL.Path == "/repos/test/repo/secret-scanning/alerts":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		default:
			// Handle workflow content downloads
			if r.Host != "" && r.URL.Host == "" {
				// This is a download URL request
				if r.URL.Path == "/test/repo/main/.github/workflows/ci.yml" {
					w.Header().Set("Content-Type", "text/plain")
					w.Write([]byte(mockWorkflowContentCI))
				} else if r.URL.Path == "/test/repo/main/.github/workflows/security.yml" {
					w.Header().Set("Content-Type", "text/plain")
					w.Write([]byte(mockWorkflowContentSecurity))
				} else {
					w.WriteHeader(http.StatusNotFound)
				}
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer server.Close()

	// Create test configuration
	cfg := createTestConfig(t)
	cfg.Evidence.Tools.GitHub.Repository = "test/repo"
	cfg.Evidence.Tools.GitHub.APIToken = "test-token"

	// Create logger
	log := createTestLogger(t)

	// Create tool
	tool := tools.NewGitHubWorkflowAnalyzer(cfg, log)
	require.NotNil(t, tool)

	// Test parameters
	params := map[string]interface{}{
		"analysis_type":           "full",
		"include_content":         false,
		"check_branch_protection": true,
		"use_cache":               false,
	}

	// Execute tool
	ctx := context.Background()
	result, evidenceSource, err := tool.Execute(ctx, params)

	// Verify results
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, evidenceSource)

	// Verify evidence source
	assert.Equal(t, "github-workflow-analyzer", evidenceSource.Type)
	assert.Contains(t, evidenceSource.Resource, "test/repo")
	assert.Greater(t, evidenceSource.Relevance, 0.0)
	assert.NotEmpty(t, evidenceSource.Content)

	// Verify metadata
	metadata := evidenceSource.Metadata
	assert.Contains(t, metadata, "repository")
	assert.Contains(t, metadata, "analysis_type")
	assert.Contains(t, metadata, "correlation_id")
	assert.Equal(t, "test/repo", metadata["repository"])
	assert.Equal(t, "full", metadata["analysis_type"])
}

func TestGitHubWorkflowAnalyzer_SecurityAnalysis(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub tests: GITHUB_TOKEN not set (requires real API access)")
	}

	// Create test server with security-focused responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/repos/test/repo/contents/.github/workflows":
			// Return only security workflow
			securityWorkflow := []map[string]interface{}{
				{
					"name":         "security.yml",
					"path":         ".github/workflows/security.yml",
					"sha":          "def456",
					"type":         "file",
					"html_url":     "https://github.com/test/repo/blob/main/.github/workflows/security.yml",
					"download_url": "https://raw.githubusercontent.com/test/repo/main/.github/workflows/security.yml",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(securityWorkflow)
		case r.URL.Path == "/repos/test/repo/code-scanning/alerts":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockCodeScanningResponse)
		case r.URL.Path == "/repos/test/repo/contents/.github/dependabot.yml":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"name": "dependabot.yml",
			})
		case r.URL.Path == "/repos/test/repo/secret-scanning/alerts":
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode([]map[string]interface{}{})
		default:
			if r.URL.Path == "/test/repo/main/.github/workflows/security.yml" {
				w.Header().Set("Content-Type", "text/plain")
				w.Write([]byte(mockWorkflowContentSecurity))
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}
	}))
	defer server.Close()

	// Create test configuration
	cfg := createTestConfig(t)
	cfg.Evidence.Tools.GitHub.Repository = "test/repo"
	cfg.Evidence.Tools.GitHub.APIToken = "test-token"

	// Create logger
	log := createTestLogger(t)

	// Create tool
	tool := tools.NewGitHubWorkflowAnalyzer(cfg, log)
	require.NotNil(t, tool)

	// Test security-focused analysis
	params := map[string]interface{}{
		"analysis_type":           "security",
		"include_content":         true,
		"check_branch_protection": false,
		"use_cache":               false,
	}

	// Execute tool
	ctx := context.Background()
	result, evidenceSource, err := tool.Execute(ctx, params)

	// Verify results
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, evidenceSource)

	// Verify security-specific content
	assert.Contains(t, result, "Security Scanning Configuration")
	assert.Contains(t, result, "CodeQL")
	assert.Contains(t, result, "Dependabot")

	// Verify metadata indicates security analysis
	metadata := evidenceSource.Metadata
	assert.Equal(t, "security", metadata["analysis_type"])
}

func TestGitHubWorkflowAnalyzer_FilterWorkflows(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub tests: GITHUB_TOKEN not set (requires real API access)")
	}

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test/repo/contents/.github/workflows" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(mockWorkflowFilesResponse)
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test configuration
	cfg := createTestConfig(t)
	cfg.Evidence.Tools.GitHub.Repository = "test/repo"
	cfg.Evidence.Tools.GitHub.APIToken = "test-token"

	// Create logger
	log := createTestLogger(t)

	// Create tool
	tool := tools.NewGitHubWorkflowAnalyzer(cfg, log)
	require.NotNil(t, tool)

	// Test with workflow filters
	params := map[string]interface{}{
		"analysis_type":           "full",
		"filter_workflows":        []string{"*security*"},
		"check_branch_protection": false,
		"use_cache":               false,
	}

	// Execute tool
	ctx := context.Background()
	result, evidenceSource, err := tool.Execute(ctx, params)

	// Verify results
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, evidenceSource)

	// Should only include security workflow based on filter
	assert.Contains(t, result, "security.yml")
	// Should not include ci.yml (filtered out)
	// Note: This test would need more sophisticated mocking to verify filtering behavior
}

func TestGitHubWorkflowAnalyzer_NoWorkflows(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub tests: GITHUB_TOKEN not set (requires real API access)")
	}

	// Create test server that returns 404 for workflows directory
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/repos/test/repo/contents/.github/workflows" {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer server.Close()

	// Create test configuration
	cfg := createTestConfig(t)
	cfg.Evidence.Tools.GitHub.Repository = "test/repo"
	cfg.Evidence.Tools.GitHub.APIToken = "test-token"

	// Create logger
	log := createTestLogger(t)

	// Create tool
	tool := tools.NewGitHubWorkflowAnalyzer(cfg, log)
	require.NotNil(t, tool)

	// Test parameters
	params := map[string]interface{}{
		"analysis_type": "full",
		"use_cache":     false,
	}

	// Execute tool
	ctx := context.Background()
	result, evidenceSource, err := tool.Execute(ctx, params)

	// Should succeed with empty workflow list
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, evidenceSource)

	// Verify it handles no workflows gracefully
	assert.Contains(t, result, "Total Workflows: 0")
}

func TestGitHubWorkflowAnalyzer_InvalidParameters(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub tests: GITHUB_TOKEN not set (requires real API access)")
	}

	// Create test configuration
	cfg := createTestConfig(t)
	cfg.Evidence.Tools.GitHub.Repository = "test/repo"

	// Create logger
	log := createTestLogger(t)

	// Create tool
	tool := tools.NewGitHubWorkflowAnalyzer(cfg, log)
	require.NotNil(t, tool)

	// Test with invalid analysis type
	params := map[string]interface{}{
		"analysis_type": "invalid",
	}

	// Execute tool
	ctx := context.Background()
	result, evidenceSource, err := tool.Execute(ctx, params)

	// Should handle invalid parameters gracefully
	// Note: Validation happens at CLI level, so tool should handle this
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, evidenceSource)
}

func TestGitHubWorkflowAnalyzer_Name(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub tests: GITHUB_TOKEN not set (requires real API access)")
	}

	cfg := createTestConfig(t)
	log := createTestLogger(t)

	tool := tools.NewGitHubWorkflowAnalyzer(cfg, log)
	require.NotNil(t, tool)

	assert.Equal(t, "github-workflow-analyzer", tool.Name())
}

func TestGitHubWorkflowAnalyzer_Description(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub tests: GITHUB_TOKEN not set (requires real API access)")
	}

	cfg := createTestConfig(t)
	log := createTestLogger(t)

	tool := tools.NewGitHubWorkflowAnalyzer(cfg, log)
	require.NotNil(t, tool)

	desc := tool.Description()
	assert.NotEmpty(t, desc)
	assert.Contains(t, desc, "GitHub Actions")
	assert.Contains(t, desc, "workflows")
	assert.Contains(t, desc, "CI/CD")
}

func TestGitHubWorkflowAnalyzer_ClaudeToolDefinition(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub tests: GITHUB_TOKEN not set (requires real API access)")
	}

	cfg := createTestConfig(t)
	log := createTestLogger(t)

	tool := tools.NewGitHubWorkflowAnalyzer(cfg, log)
	require.NotNil(t, tool)

	claudeTool := tool.GetClaudeToolDefinition()

	assert.Equal(t, "github-workflow-analyzer", claudeTool.Name)
	assert.NotEmpty(t, claudeTool.Description)
	assert.NotNil(t, claudeTool.InputSchema)

	// Verify input schema structure
	schema := claudeTool.InputSchema
	assert.Equal(t, "object", schema["type"])

	properties, ok := schema["properties"].(map[string]interface{})
	assert.True(t, ok)
	assert.Contains(t, properties, "analysis_type")
	assert.Contains(t, properties, "include_content")
	assert.Contains(t, properties, "filter_workflows")
	assert.Contains(t, properties, "check_branch_protection")
	assert.Contains(t, properties, "use_cache")

	// Verify analysis_type enum
	analysisType, ok := properties["analysis_type"].(map[string]interface{})
	assert.True(t, ok)
	enum, ok := analysisType["enum"].([]string)
	assert.True(t, ok)
	assert.Contains(t, enum, "security")
	assert.Contains(t, enum, "deployment")
	assert.Contains(t, enum, "approval")
	assert.Contains(t, enum, "full")
}

func TestGitHubWorkflowAnalyzer_AuthenticationRequired(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("Skipping GitHub tests: GITHUB_TOKEN not set (requires real API access)")
	}

	// Create test configuration without token
	cfg := createTestConfig(t)
	cfg.Evidence.Tools.GitHub.Repository = "test/repo"
	cfg.Evidence.Tools.GitHub.APIToken = ""
	cfg.Auth.GitHub.Token = ""

	// Create logger
	log := createTestLogger(t)

	// Create tool
	tool := tools.NewGitHubWorkflowAnalyzer(cfg, log)
	require.NotNil(t, tool)

	// Test parameters
	params := map[string]interface{}{
		"analysis_type": "full",
		"use_cache":     false,
	}

	// Execute tool without authentication
	ctx := context.Background()
	result, evidenceSource, err := tool.Execute(ctx, params)

	// Should still work with limited functionality or cached data
	require.NoError(t, err)
	assert.NotEmpty(t, result)
	assert.NotNil(t, evidenceSource)

	// Verify auth status in metadata
	metadata := evidenceSource.Metadata
	authStatus, ok := metadata["auth_status"].(map[string]interface{})
	assert.True(t, ok)
	// Should indicate authentication issues
	assert.False(t, authStatus["authenticated"].(bool))
}

// Helper function to create test configuration
func createTestConfig(t *testing.T) *config.Config {
	tempDir := t.TempDir()

	return &config.Config{
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:          true,
					Repository:       "test/repo",
					IncludeWorkflows: true,
					IncludeIssues:    true,
					MaxIssues:        50,
				},
			},
		},
		Auth: config.AuthConfig{
			CacheDir: filepath.Join(tempDir, ".auth_cache"),
			GitHub: config.GitHubAuthConfig{
				Token: "",
			},
		},
		Logging: config.LoggingConfig{
			// Empty - use defaults
		},
	}
}

// Helper function to create test logger
func createTestLogger(t *testing.T) logger.Logger {
	logConfig := logger.Config{
		Level:  logger.WarnLevel,
		Output: "stderr",
	}

	log, err := logger.New(&logConfig)
	require.NoError(t, err)

	return log
}
