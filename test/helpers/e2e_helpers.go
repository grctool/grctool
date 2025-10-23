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

package helpers

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/require"
)

// E2E test helpers - real APIs with auth

// SetupE2ETest loads the E2E test configuration from test fixtures
func SetupE2ETest(t *testing.T) *config.Config {
	t.Helper()

	// Path to E2E test config (relative to project root)
	testConfigPath := "test/fixtures/e2e/.grctool.yaml"

	// Make path absolute from current directory
	absPath, err := filepath.Abs(testConfigPath)
	require.NoError(t, err, "Failed to resolve test config path")

	// Reset viper to clean state
	viper.Reset()

	// Load config from test fixtures
	viper.SetConfigFile(absPath)
	err = viper.ReadInConfig()
	require.NoError(t, err, "Failed to read E2E test config from %s", absPath)

	// Load and resolve paths (paths will be relative to config file location)
	cfg, err := config.Load()
	require.NoError(t, err, "Failed to load E2E test config")

	return cfg
}

func SkipIfNoGitHubAuth(t *testing.T) {
	t.Helper()
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("GITHUB_TOKEN required for E2E tests")
	}
}

func SkipIfNoTugboatAuth(t *testing.T) {
	t.Helper()
	// Check for valid Tugboat authentication
	cmd := exec.Command("./bin/grctool", "auth", "status")
	if err := cmd.Run(); err != nil {
		t.Skip("Valid Tugboat authentication required for E2E tests")
	}
}

func SetupE2EEnvironment(t *testing.T) map[string]string {
	t.Helper()

	env := map[string]string{
		"GITHUB_TOKEN":     os.Getenv("GITHUB_TOKEN"),
		"TEST_GITHUB_REPO": getTestGitHubRepo(),
		"TUGBOAT_API_KEY":  os.Getenv("TUGBOAT_API_KEY"),
		"TUGBOAT_BASE_URL": getTugboatBaseURL(),
	}

	for key, value := range env {
		if value == "" {
			t.Logf("Warning: %s not set", key)
		}
	}

	return env
}

func getTestGitHubRepo() string {
	repo := os.Getenv("TEST_GITHUB_REPO")
	if repo == "" {
		return "7thsense/test-compliance-repo" // Default test repo
	}
	return repo
}

func getTugboatBaseURL() string {
	url := os.Getenv("TUGBOAT_BASE_URL")
	if url == "" {
		return "https://api.tugboat.qa" // Default Tugboat API
	}
	return url
}

// E2E test environment validation
func ValidateE2EEnvironment(t *testing.T) {
	t.Helper()

	// Check required binaries
	RequireBinaryExists(t, "git")
	RequireBinaryExists(t, "./bin/grctool")

	// Check required environment variables
	env := SetupE2EEnvironment(t)
	RequireEnvVar(t, "GITHUB_TOKEN", env["GITHUB_TOKEN"])

	// Validate API connectivity
	ValidateGitHubConnectivity(t, env["GITHUB_TOKEN"])

	if env["TUGBOAT_API_KEY"] != "" {
		ValidateTugboatConnectivity(t, env["TUGBOAT_API_KEY"], env["TUGBOAT_BASE_URL"])
	}
}

func RequireBinaryExists(t *testing.T, binary string) {
	t.Helper()
	_, err := exec.LookPath(binary)
	require.NoError(t, err, "Required binary not found: %s", binary)
}

func RequireEnvVar(t *testing.T, name, value string) {
	t.Helper()
	require.NotEmpty(t, value, "Required environment variable not set: %s", name)
}

func ValidateGitHubConnectivity(t *testing.T, token string) {
	t.Helper()

	if token == "" {
		t.Skip("No GitHub token provided")
		return
	}

	// Test GitHub API connectivity
	cmd := exec.Command("curl", "-s", "-H", "Authorization: token "+token,
		"https://api.github.com/user")
	output, err := cmd.Output()
	require.NoError(t, err, "Failed to connect to GitHub API")
	require.Contains(t, string(output), "login", "GitHub API response should contain user info")
}

func ValidateTugboatConnectivity(t *testing.T, apiKey, baseURL string) {
	t.Helper()

	if apiKey == "" {
		t.Skip("No Tugboat API key provided")
		return
	}

	// Test Tugboat API connectivity using grctool
	cmd := exec.Command("./bin/grctool", "auth", "status")
	cmd.Env = append(os.Environ(),
		"TUGBOAT_API_KEY="+apiKey,
		"TUGBOAT_BASE_URL="+baseURL,
	)

	err := cmd.Run()
	require.NoError(t, err, "Failed to authenticate with Tugboat API")
}

// E2E test workspace setup
func SetupE2EWorkspace(t *testing.T) string {
	t.Helper()

	tempDir := t.TempDir()
	env := SetupE2EEnvironment(t)

	// Create production-like config
	configContent := `data_dir: .
log_level: info
evidence:
  tools:
    github:
      enabled: true
      repository: ` + env["TEST_GITHUB_REPO"] + `
      api_token: ` + env["GITHUB_TOKEN"] + `
    terraform:
      enabled: true
      scan_paths:
        - terraform/
        - infra/
      include_patterns:
        - "*.tf"
        - "*.tfvars"
      exclude_patterns:
        - ".terraform/"
        - "*.backup"
`

	configPath := tempDir + "/.grctool.yaml"
	err := os.WriteFile(configPath, []byte(configContent), 0600) // Secure permissions for config with secrets
	require.NoError(t, err)

	return tempDir
}

// Real API integration helpers
func TestRealGitHubIntegration(t *testing.T, repo string) {
	t.Helper()

	SkipIfNoGitHubAuth(t)

	// Test real GitHub API calls
	cmd := exec.Command("./bin/grctool", "tool", "github-enhanced", "--repo", repo)
	cmd.Env = append(os.Environ(), "GITHUB_TOKEN="+os.Getenv("GITHUB_TOKEN"))

	output, err := cmd.Output()
	require.NoError(t, err, "GitHub tool should succeed with real API")
	require.NotEmpty(t, output, "GitHub tool should return data")
}

func TestRealTugboatIntegration(t *testing.T) {
	t.Helper()

	SkipIfNoTugboatAuth(t)

	// Test real Tugboat sync
	cmd := exec.Command("./bin/grctool", "sync", "--verbose")
	output, err := cmd.Output()
	require.NoError(t, err, "Tugboat sync should succeed")
	require.NotEmpty(t, output, "Sync should return data")
}

// Cleanup helpers for E2E tests
func CleanupE2EArtifacts(t *testing.T, workspaceDir string) {
	t.Helper()

	// Clean up any temporary files created during E2E tests
	// This might include cache files, logs, etc.
	cacheDir := workspaceDir + "/.cache"
	if _, err := os.Stat(cacheDir); err == nil {
		err = os.RemoveAll(cacheDir)
		require.NoError(t, err, "Failed to cleanup cache directory")
	}

	logFile := workspaceDir + "/grctool.log"
	if _, err := os.Stat(logFile); err == nil {
		err = os.Remove(logFile)
		require.NoError(t, err, "Failed to cleanup log file")
	}
}

// Rate limit helpers for E2E tests
func WaitForRateLimit(t *testing.T) {
	t.Helper()
	// Add delays between E2E API calls to avoid rate limiting
	// This would typically check rate limit headers and sleep appropriately
}
