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

package helpers

import (
	"net/http"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/vcr"
	"github.com/grctool/grctool/test/testdata"
	"github.com/grctool/grctool/test/tools"
)

// Integration test helpers - VCR recordings, tool orchestration
func SetupIntegrationTest(t *testing.T, toolName string) (*config.Config, logger.Logger) {
	cfg := createIntegrationConfig(t)
	log := createIntegrationLogger(t)
	return cfg, log
}

func createIntegrationConfig(t *testing.T) *config.Config {
	tempDir := t.TempDir()
	return &config.Config{
		Evidence: config.EvidenceConfig{
			Tools: config.ToolsConfig{
				GitHub: config.GitHubToolConfig{
					Enabled:    true,
					Repository: "test-org/test-repo",
					APIToken:   "integration-test-token",
				},
				Terraform: config.TerraformToolConfig{
					Enabled:         true,
					ScanPaths:       []string{tempDir},
					IncludePatterns: []string{"*.tf", "*.tfvars"},
					ExcludePatterns: []string{".terraform/"},
				},
			},
		},
		Storage: config.StorageConfig{
			DataDir: tempDir,
		},
	}
}

func createIntegrationLogger(t *testing.T) logger.Logger {
	log, err := logger.New(&logger.Config{
		Level:  logger.InfoLevel, // More verbose for integration tests
		Format: "text",
		Output: "stdout",
	})
	if err != nil {
		t.Fatalf("Failed to create integration test logger: %v", err)
	}
	return log
}

// Setup VCR for GitHub integration tests
func SetupGitHubIntegrationTest(t *testing.T) (*http.Client, *vcr.VCR) {
	return tools.SetupIntegrationVCR(t, "github_integration")
}

// Setup VCR for Terraform integration tests
func SetupTerraformIntegrationTest(t *testing.T) string {
	// Create terraform test files to temp directory
	testData := testdata.NewTestDataManager(t)
	return testData.CreateTempTerraformProject(t, []string{
		"multi_az", "encryption", "iam_roles", "network_security",
	})
}

// Setup VCR for tool integration tests
func SetupToolIntegrationTest(t *testing.T, toolName string) (*http.Client, *vcr.VCR, string) {
	// Create HTTP client with VCR
	client, vcr := tools.SetupIntegrationVCR(t, toolName+"_integration")

	// Create test workspace
	testData := testdata.NewTestDataManager(t)
	workspaceDir := testData.CreateCompleteTestEnv(t)

	return client, vcr, workspaceDir
}

// Helper for evidence task integration tests
func SetupEvidenceTaskIntegrationTest(t *testing.T, taskRef string) (*config.Config, string) {
	cfg := createIntegrationConfig(t)
	testData := testdata.NewTestDataManager(t)

	// Create workspace with evidence task data
	workspaceDir := testData.CreateCompleteTestWorkspace(t, testdata.TestWorkspaceConfig{
		EvidenceFixtures: []string{taskRef + "_sample"},
		ConfigFixture:    "complete",
	})

	cfg.Storage.DataDir = workspaceDir
	return cfg, workspaceDir
}

// Validation helpers for integration tests
func ValidateToolOutput(t *testing.T, output interface{}, expectedFields []string) {
	// Add validation logic for tool outputs
	if output == nil {
		t.Fatal("Tool output should not be nil")
	}

	// Type assertion and field validation would go here
	// This is a placeholder for more specific validation
}

func ValidateVCRRecording(t *testing.T, vcrInstance *vcr.VCR, minInteractions int) {
	stats := vcrInstance.Stats()

	totalInteractions, ok := stats["total_interactions"].(int)
	if !ok {
		t.Fatal("Expected total_interactions to be an integer")
	}

	if totalInteractions < minInteractions {
		t.Errorf("Expected at least %d interactions, got %d", minInteractions, totalInteractions)
	}
}

// Environment helpers for integration tests
func SkipIfVCRDisabled(t *testing.T) {
	// Check if VCR is available for integration tests
	// This could check environment variables or configuration
}
