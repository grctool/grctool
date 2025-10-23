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

package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/storage"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

func TestPolicyViewCommand(t *testing.T) {
	// Create temporary directory for test data
	tempDir := t.TempDir()

	// Create test storage
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}

	// Create test policy
	testPolicy := &domain.Policy{
		ID:            "POL-TEST-001",
		Name:          "Test Security Policy",
		Description:   "A test policy for unit testing",
		Framework:     "SOC2",
		Status:        "active",
		CreatedAt:     time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:     time.Date(2024, time.March, 15, 12, 0, 0, 0, time.UTC),
		Summary:       "Test policy summary",
		Content:       "# Test Policy\n\nThis is a test policy content with markdown formatting.",
		Version:       "1.0",
		ControlCount:  5,
		EvidenceCount: 3,
		ViewCount:     42,
		Assignees: []domain.Person{
			{
				ID:    "USER-001",
				Name:  "Test User",
				Email: "test@example.com",
				Role:  "Policy Owner",
			},
		},
		Tags: []domain.Tag{
			{ID: "TAG-001", Name: "security", Color: "#ff0000"},
		},
	}

	// Save test policy
	if err := storage.SavePolicy(testPolicy); err != nil {
		t.Fatalf("Failed to save test policy: %v", err)
	}

	tests := []struct {
		name     string
		args     []string
		flags    map[string]string
		validate func(t *testing.T, output string, err error)
	}{
		{
			name: "view existing policy",
			args: []string{"POL-TEST-001"},
			validate: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !strings.Contains(output, "# Policy POL-TEST-001") {
					t.Error("Output should contain policy header")
				}
				if !strings.Contains(output, "Test Security Policy") {
					t.Error("Output should contain policy name")
				}
				if !strings.Contains(output, "This is a test policy content") {
					t.Error("Output should contain policy content")
				}
				if !strings.Contains(output, "## Policy Metadata") {
					t.Error("Output should contain metadata section")
				}
			},
		},
		{
			name:  "view with summary flag",
			args:  []string{"POL-TEST-001"},
			flags: map[string]string{"summary": "true"},
			validate: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !strings.Contains(output, "## Test Security Policy") {
					t.Error("Summary should contain policy title")
				}
				if !strings.Contains(output, "Test policy summary") {
					t.Error("Summary should contain policy summary")
				}
				if strings.Contains(output, "## Policy Metadata") {
					t.Error("Summary should not contain full metadata section")
				}
			},
		},
		{
			name:  "view with output file",
			args:  []string{"POL-TEST-001"},
			flags: map[string]string{"output": filepath.Join(tempDir, "test-policy.md")},
			validate: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}

				// Check that file was created
				outputPath := filepath.Join(tempDir, "test-policy.md")
				if _, err := os.Stat(outputPath); os.IsNotExist(err) {
					t.Error("Output file was not created")
					return
				}

				// Read and validate file content
				content, err := os.ReadFile(outputPath)
				if err != nil {
					t.Errorf("Failed to read output file: %v", err)
					return
				}

				contentStr := string(content)
				if !strings.Contains(contentStr, "# Policy POL-TEST-001") {
					t.Error("File should contain policy header")
				}
				if !strings.Contains(contentStr, "Test Security Policy") {
					t.Error("File should contain policy name")
				}
			},
		},
		{
			name: "view non-existent policy",
			args: []string{"POL-NONEXISTENT"},
			validate: func(t *testing.T, output string, err error) {
				if err == nil {
					t.Error("Expected error for non-existent policy")
				}
				if !strings.Contains(err.Error(), "policy not found") {
					t.Errorf("Error should mention policy not found, got: %v", err)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create command with test environment
			cmd := &cobra.Command{
				Use: "policy view",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Create a minimal config file for the test in the temp directory
					configContent := fmt.Sprintf(`storage:
  data_dir: %s
tugboat:
  base_url: "https://app.tugboatlogic.com"
  org_id: "test-org"
  auth_mode: "browser"
  timeout: 30s
  rate_limit: 10
vcr:
  enabled: false
  mode: "off"
logging:
  level: "warn"
interpolation:
  enabled: true
  variables:
    organization:
      name: "Test Organization"
    "Organization Name": "Test Organization"
`, tempDir)
					configPath := filepath.Join(tempDir, ".grctool.yaml")
					_ = os.WriteFile(configPath, []byte(configContent), 0644)

					// Set working directory to temp dir so config is found
					oldWd, _ := os.Getwd()
					defer func() { _ = os.Chdir(oldWd) }()
					_ = os.Chdir(tempDir)

					// Reinitialize viper to read the test config file
					viper.Reset()
					viper.AddConfigPath(".")
					viper.SetConfigType("yaml")
					viper.SetConfigName(".grctool")
					_ = viper.ReadInConfig()

					return runPolicyView(cmd, args)
				},
			}

			// Add flags
			cmd.Flags().StringP("output", "o", "", "Output file path")
			cmd.Flags().Bool("summary", false, "Show summary format")
			cmd.Flags().Bool("metadata-only", false, "Show metadata only")

			// Set flag values if provided
			for flag, value := range tt.flags {
				_ = cmd.Flags().Set(flag, value)
			}

			// Capture output
			var output strings.Builder
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			// Execute command
			cmd.SetArgs(tt.args)
			err := cmd.Execute()

			// Validate results
			tt.validate(t, output.String(), err)
		})
	}
}

func TestPolicyListCommand(t *testing.T) {
	// Create temporary directory for test data
	tempDir := t.TempDir()

	// Create test storage
	storage, err := storage.NewStorage(config.StorageConfig{DataDir: tempDir})
	if err != nil {
		t.Fatalf("Failed to create test storage: %v", err)
	}

	// Create test policies
	testPolicies := []*domain.Policy{
		{
			ID:          "POL-001",
			Name:        "SOC2 Security Policy",
			Description: "SOC2 compliance policy",
			Framework:   "SOC2",
			Status:      "active",
			CreatedAt:   time.Date(2024, time.January, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2024, time.March, 15, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:          "POL-002",
			Name:        "ISO27001 Risk Management",
			Description: "ISO27001 risk management policy",
			Framework:   "ISO27001",
			Status:      "active",
			CreatedAt:   time.Date(2024, time.January, 15, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2024, time.March, 10, 12, 0, 0, 0, time.UTC),
		},
		{
			ID:          "POL-003",
			Name:        "Draft Policy",
			Description: "A policy in draft status",
			Framework:   "SOC2",
			Status:      "draft",
			CreatedAt:   time.Date(2024, time.February, 1, 0, 0, 0, 0, time.UTC),
			UpdatedAt:   time.Date(2024, time.February, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	// Save test policies
	for _, policy := range testPolicies {
		if err := storage.SavePolicy(policy); err != nil {
			t.Fatalf("Failed to save test policy %s: %v", policy.ID, err)
		}
	}

	tests := []struct {
		name     string
		flags    map[string]string
		validate func(t *testing.T, output string, err error)
	}{
		{
			name: "list all policies",
			validate: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !strings.Contains(output, "Found 3 policies") {
					t.Error("Should show correct policy count")
				}
				if !strings.Contains(output, "POL-001") {
					t.Error("Should contain POL-001")
				}
				if !strings.Contains(output, "POL-002") {
					t.Error("Should contain POL-002")
				}
				if !strings.Contains(output, "POL-003") {
					t.Error("Should contain POL-003")
				}
			},
		},
		{
			name:  "filter by framework",
			flags: map[string]string{"framework": "SOC2"},
			validate: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !strings.Contains(output, "Found 2 policies") {
					t.Error("Should show correct filtered count")
				}
				if !strings.Contains(output, "POL-001") {
					t.Error("Should contain SOC2 policy POL-001")
				}
				if !strings.Contains(output, "POL-003") {
					t.Error("Should contain SOC2 policy POL-003")
				}
				if strings.Contains(output, "POL-002") {
					t.Error("Should not contain ISO27001 policy POL-002")
				}
			},
		},
		{
			name:  "filter by status",
			flags: map[string]string{"status": "active"},
			validate: func(t *testing.T, output string, err error) {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if !strings.Contains(output, "Found 2 policies") {
					t.Error("Should show correct filtered count")
				}
				if !strings.Contains(output, "POL-001") {
					t.Error("Should contain active policy POL-001")
				}
				if !strings.Contains(output, "POL-002") {
					t.Error("Should contain active policy POL-002")
				}
				if strings.Contains(output, "POL-003") {
					t.Error("Should not contain draft policy POL-003")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create command with test environment
			cmd := &cobra.Command{
				Use: "policy list",
				RunE: func(cmd *cobra.Command, args []string) error {
					// Create a minimal config file for the test in the temp directory
					configContent := fmt.Sprintf(`storage:
  data_dir: %s
tugboat:
  base_url: "https://app.tugboatlogic.com"
  org_id: "test-org"
  auth_mode: "browser"
  timeout: 30s
  rate_limit: 10
vcr:
  enabled: false
  mode: "off"
logging:
  level: "warn"
interpolation:
  enabled: true
  variables:
    organization:
      name: "Test Organization"
    "Organization Name": "Test Organization"
`, tempDir)
					configPath := filepath.Join(tempDir, ".grctool.yaml")
					_ = os.WriteFile(configPath, []byte(configContent), 0644)

					// Set working directory to temp dir so config is found
					oldWd, _ := os.Getwd()
					defer func() { _ = os.Chdir(oldWd) }()
					_ = os.Chdir(tempDir)

					// Reinitialize viper to read the test config file
					viper.Reset()
					viper.AddConfigPath(".")
					viper.SetConfigType("yaml")
					viper.SetConfigName(".grctool")
					_ = viper.ReadInConfig()

					return runPolicyList(cmd, args)
				},
			}

			// Add flags
			cmd.Flags().String("framework", "", "Filter by framework")
			cmd.Flags().String("status", "", "Filter by status")
			cmd.Flags().Bool("details", false, "Show details")

			// Set flag values if provided
			for flag, value := range tt.flags {
				_ = cmd.Flags().Set(flag, value)
			}

			// Capture output
			var output strings.Builder
			cmd.SetOut(&output)
			cmd.SetErr(&output)

			// Execute command
			err := cmd.Execute()

			// Validate results
			tt.validate(t, output.String(), err)
		})
	}
}

func TestPolicyCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
	}{
		{
			name: "policy command help",
			args: []string{"policy", "--help"},
		},
		{
			name: "policy view help",
			args: []string{"policy", "view", "--help"},
		},
		{
			name: "policy list help",
			args: []string{"policy", "list", "--help"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output strings.Builder
			rootCmd.SetOut(&output)
			rootCmd.SetErr(&output)
			rootCmd.SetArgs(tt.args)

			// Help commands should not return error
			err := rootCmd.Execute()
			if err != nil {
				t.Errorf("Help command failed: %v", err)
			}

			// Should contain help text
			outputStr := output.String()
			if !strings.Contains(outputStr, "Usage:") {
				t.Error("Help output should contain usage information")
			}
		})
	}
}
