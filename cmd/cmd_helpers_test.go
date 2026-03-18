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
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/models"
	configService "github.com/grctool/grctool/internal/services/config"
	"github.com/grctool/grctool/internal/services/evidence"
	"github.com/grctool/grctool/internal/services/validation"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ============================================================
// evidence.go — formatFileSize, getStatusIcon, getScoreStatus,
//               truncateFileName, extractRequirements,
//               calculatePeriod, parseJSONResult,
//               isTugboatManagedTask, displayTugboatManagedMessage,
//               selectEvidenceTemplate, identifyApplicableToolsForAssembly,
//               generateXxxTemplate, applyTemplateVariables,
//               display* functions
// ============================================================

func TestFormatFileSize(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero", 0, "0 B"},
		{"small", 500, "500 B"},
		{"kilobytes", 2048, "2.0 KB"},
		{"megabytes", 5 * 1024 * 1024, "5.0 MB"},
		{"gigabytes", 3 * 1024 * 1024 * 1024, "3.0 GB"},
		{"just under 1KB", 1023, "1023 B"},
		{"exactly 1KB", 1024, "1.0 KB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, formatFileSize(tt.bytes))
		})
	}
}

func TestGetStatusIcon(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status   string
		expected string
	}{
		{"passed", "✓"},
		{"pass", "✓"},
		{"PASS", "✓"},
		{"failed", "✗"},
		{"fail", "✗"},
		{"FAIL", "✗"},
		{"warning", "⚠️"},
		{"WARNING", "⚠️"},
		{"unknown", "○"},
		{"", "○"},
	}

	for _, tt := range tests {
		t.Run(tt.status, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, getStatusIcon(tt.status))
		})
	}
}

func TestGetScoreStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		percentage float64
		contains   string
	}{
		{"excellent", 95.0, "pass"},
		{"good", 75.0, "pass"},
		{"warning", 55.0, "warning"},
		{"fail", 40.0, "fail"},
		{"zero", 0.0, "fail"},
		{"boundary 90", 90.0, "pass"},
		{"boundary 70", 70.0, "pass"},
		{"boundary 50", 50.0, "warning"},
		{"boundary 49", 49.0, "fail"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := getScoreStatus(tt.percentage)
			assert.Contains(t, result, tt.contains)
		})
	}
}

func TestTruncateFileName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short", "file.json", 20, "file.json"},
		{"long", "very-long-evidence-filename-that-is-too-long.json", 20, "very-long-evidenc..."},
		{"exact", "12345678901234567890", 20, "12345678901234567890"},
		{"empty", "", 10, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, truncateFileName(tt.input, tt.maxLen))
		})
	}
}

func TestCalculatePeriod(t *testing.T) {
	t.Parallel()

	tests := []struct {
		window   string
		expected string
	}{
		{"2025-Q4", "Quarterly period 2025-Q4"},
		{"2025-Q1", "Quarterly period 2025-Q1"},
		{"2025", "2025"},
		{"2025-03", "2025-03"},
	}

	for _, tt := range tests {
		t.Run(tt.window, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, calculatePeriod(tt.window))
		})
	}
}

func TestParseJSONResult(t *testing.T) {
	t.Parallel()

	t.Run("valid JSON", func(t *testing.T) {
		t.Parallel()
		var result map[string]string
		err := parseJSONResult(`{"key":"value"}`, &result)
		require.NoError(t, err)
		assert.Equal(t, "value", result["key"])
	})

	t.Run("invalid JSON", func(t *testing.T) {
		t.Parallel()
		var result map[string]string
		err := parseJSONResult(`not json`, &result)
		assert.Error(t, err)
	})

	t.Run("complex JSON", func(t *testing.T) {
		t.Parallel()
		type nested struct {
			Name  string `json:"name"`
			Count int    `json:"count"`
		}
		var result nested
		err := parseJSONResult(`{"name":"test","count":42}`, &result)
		require.NoError(t, err)
		assert.Equal(t, "test", result.Name)
		assert.Equal(t, 42, result.Count)
	})
}

func TestExtractRequirements(t *testing.T) {
	t.Parallel()

	t.Run("access control task", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			Description: "Demonstrate access control procedures and approval workflow",
			Guidance:    "Show configuration and review process with policy documentation",
		}
		reqs := extractRequirements(task)
		assert.Contains(t, reqs, "Access control documentation")
		assert.Contains(t, reqs, "Approval workflow evidence")
		assert.Contains(t, reqs, "System configuration details")
		assert.Contains(t, reqs, "Procedural documentation")
		assert.Contains(t, reqs, "Policy documentation")
		assert.Contains(t, reqs, "Review documentation")
	})

	t.Run("log monitoring task", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			Description: "Demonstrate log collection and monitoring procedures",
		}
		reqs := extractRequirements(task)
		assert.Contains(t, reqs, "Audit log examples")
		assert.Contains(t, reqs, "Procedural documentation")
	})

	t.Run("training task", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			Description: "Provide training records for security awareness",
		}
		reqs := extractRequirements(task)
		assert.Contains(t, reqs, "Training records")
	})

	t.Run("empty task", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{}
		reqs := extractRequirements(task)
		assert.Empty(t, reqs)
	})
}

func TestIsTugboatManagedTask(t *testing.T) {
	t.Parallel()

	t.Run("AEC enabled with Hybrid collection", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			AecStatus:      &domain.AecStatus{Status: "enabled"},
			CollectionType: "Hybrid",
		}
		assert.True(t, isTugboatManagedTask(task))
	})

	t.Run("AEC disabled", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			AecStatus: &domain.AecStatus{Status: "disabled"},
		}
		assert.False(t, isTugboatManagedTask(task))
	})

	t.Run("AEC nil", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{}
		assert.False(t, isTugboatManagedTask(task))
	})

	t.Run("AEC enabled but Manual collection", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			AecStatus:      &domain.AecStatus{Status: "enabled"},
			CollectionType: "Manual",
		}
		assert.False(t, isTugboatManagedTask(task))
	})
}

func TestDisplayTugboatManagedMessage(t *testing.T) {
	t.Parallel()

	t.Run("with description and guidance", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		task := &domain.EvidenceTask{
			ReferenceID: "ET-0001",
			Name:        "Test Task",
			Description: "Test description",
			Guidance:    "Test guidance",
		}

		displayTugboatManagedMessage(cmd, task)
		output := buf.String()
		assert.Contains(t, output, "ET-0001")
		assert.Contains(t, output, "Test Task")
		assert.Contains(t, output, "Test description")
		assert.Contains(t, output, "Test guidance")
		assert.Contains(t, output, "Tugboat Logic")
	})

	t.Run("without description", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		task := &domain.EvidenceTask{
			ReferenceID: "ET-0002",
			Name:        "Empty Task",
		}

		displayTugboatManagedMessage(cmd, task)
		output := buf.String()
		assert.Contains(t, output, "ET-0002")
		assert.NotContains(t, output, "What's needed")
	})
}

func TestIdentifyApplicableToolsForAssembly(t *testing.T) {
	t.Parallel()

	t.Run("explicit tools override", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{Name: "anything"}
		result := identifyApplicableToolsForAssembly(task, []string{"custom-tool"})
		assert.Equal(t, []string{"custom-tool"}, result)
	})

	t.Run("terraform task", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			Name:        "Terraform Infrastructure Security",
			Description: "Show infrastructure configuration",
		}
		result := identifyApplicableToolsForAssembly(task, nil)
		assert.Contains(t, result, "terraform-security-indexer")
		assert.Contains(t, result, "terraform-security-analyzer")
	})

	t.Run("github task", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			Name:        "GitHub Repository Access",
			Description: "Show repository permissions",
		}
		result := identifyApplicableToolsForAssembly(task, nil)
		assert.Contains(t, result, "github-permissions")
		assert.Contains(t, result, "github-security-features")
	})

	t.Run("google workspace task", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			Name:        "Google Workspace Access",
			Description: "Drive sharing settings",
		}
		result := identifyApplicableToolsForAssembly(task, nil)
		assert.Contains(t, result, "google-workspace")
	})

	t.Run("documentation task", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			Name:        "Security Policy Documentation",
			Description: "Policy review process",
		}
		result := identifyApplicableToolsForAssembly(task, nil)
		assert.Contains(t, result, "docs-reader")
	})

	t.Run("task with no tool hints", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			Name:        "Manual Process",
			Description: "Something completely unrelated",
		}
		result := identifyApplicableToolsForAssembly(task, nil)
		assert.Empty(t, result)
	})

	t.Run("code review task matches github", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{
			Name:        "Code Review Process",
			Description: "Demonstrate code review practices",
		}
		result := identifyApplicableToolsForAssembly(task, nil)
		assert.Contains(t, result, "github-workflow-analyzer")
	})
}

func TestSelectEvidenceTemplate(t *testing.T) {
	t.Parallel()

	categories := []struct {
		category string
		contains string
	}{
		{"Infrastructure", "Infrastructure"},
		{"Personnel", "Personnel"},
		{"Process", "Process"},
		{"Compliance", "Compliance"},
		{"Monitoring", "Monitoring"},
		{"Data", "Data"},
		{"Unknown", "Template"}, // falls back to generic
	}

	for _, tt := range categories {
		t.Run(tt.category, func(t *testing.T) {
			t.Parallel()
			task := &domain.EvidenceTask{
				Category: tt.category,
			}
			template := selectEvidenceTemplate(task)
			assert.NotEmpty(t, template)
			assert.Contains(t, template, tt.contains)
		})
	}

	t.Run("nil MasterContent", func(t *testing.T) {
		t.Parallel()
		task := &domain.EvidenceTask{}
		template := selectEvidenceTemplate(task)
		assert.NotEmpty(t, template, "should return generic template")
	})
}

func TestGenerateTemplates(t *testing.T) {
	t.Parallel()

	t.Run("generic template", func(t *testing.T) {
		t.Parallel()
		template := generateGenericTemplate()
		assert.Contains(t, template, "Evidence Report")
		assert.Contains(t, template, "Executive Summary")
		assert.Contains(t, template, "Technical Evidence")
	})

	t.Run("infrastructure template", func(t *testing.T) {
		t.Parallel()
		template := generateInfrastructureTemplate()
		assert.Contains(t, template, "Infrastructure")
	})

	t.Run("personnel template", func(t *testing.T) {
		t.Parallel()
		template := generatePersonnelTemplate()
		assert.Contains(t, template, "Personnel")
	})

	t.Run("process template", func(t *testing.T) {
		t.Parallel()
		template := generateProcessTemplate()
		assert.Contains(t, template, "Process")
	})

	t.Run("compliance template", func(t *testing.T) {
		t.Parallel()
		template := generateComplianceTemplate()
		assert.Contains(t, template, "Compliance")
	})

	t.Run("monitoring template", func(t *testing.T) {
		t.Parallel()
		template := generateMonitoringTemplate()
		assert.Contains(t, template, "Monitoring")
	})

	t.Run("data template", func(t *testing.T) {
		t.Parallel()
		template := generateDataTemplate()
		assert.Contains(t, template, "Data")
	})
}

func TestApplyTemplateVariables(t *testing.T) {
	t.Parallel()

	task := &domain.EvidenceTask{
		ID:          327992,
		ReferenceID: "ET-0001",
		Name:        "Access Control Evidence",
	}

	template := "# {{TASK_REF}} - {{TASK_NAME}}\nTugboat ID: {{TUGBOAT_ID}}\nWindow: {{WINDOW}}\nPeriod: {{PERIOD}}"
	result := applyTemplateVariables(template, task, "2025-Q4")

	assert.Contains(t, result, "ET-0001")
	assert.Contains(t, result, "Access Control Evidence")
	assert.Contains(t, result, "327992")
	assert.Contains(t, result, "2025-Q4")
	assert.Contains(t, result, "Quarterly period 2025-Q4")
	assert.NotContains(t, result, "{{")
}

func TestGetDefaultTemplate(t *testing.T) {
	t.Parallel()

	template := getDefaultTemplate()
	assert.Contains(t, template, "{{TASK_REF}}")
	assert.Contains(t, template, "{{TASK_NAME}}")
	assert.Contains(t, template, "{{WINDOW}}")
	assert.Contains(t, template, "{{DATE}}")
	assert.Contains(t, template, "{{PERIOD}}")
}

func TestDisplayEvidenceTasksEmpty(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	// Empty tasks list
	err := displayEvidenceTasks(cmd, nil, nil, nil)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No evidence tasks found")
}

// ============================================================
// Help-based tests for subcommands to cover init/flag registration
// ============================================================

func TestToolSubcommandHelp(t *testing.T) {
	// Cannot use t.Parallel() - uses global rootCmd
	for _, sub := range toolCmd.Commands() {
		t.Run("tool_"+sub.Name()+"_help", func(t *testing.T) {
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs([]string{"tool", sub.Name(), "--help"})
			err := rootCmd.Execute()
			assert.NoError(t, err, "help for tool %s should succeed", sub.Name())
		})
	}
}

func TestEvidenceSubcommandHelp(t *testing.T) {
	// Cannot use t.Parallel() - uses global rootCmd
	for _, sub := range evidenceCmd.Commands() {
		t.Run("evidence_"+sub.Name()+"_help", func(t *testing.T) {
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs([]string{"evidence", sub.Name(), "--help"})
			err := rootCmd.Execute()
			assert.NoError(t, err, "help for evidence %s should succeed", sub.Name())
		})
	}
}

func TestStatusSubcommandHelp(t *testing.T) {
	// Cannot use t.Parallel() - uses global rootCmd
	for _, sub := range evidenceStatusCmd.Commands() {
		t.Run("status_"+sub.Name()+"_help", func(t *testing.T) {
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs([]string{"status", sub.Name(), "--help"})
			err := rootCmd.Execute()
			assert.NoError(t, err, "help for status %s should succeed", sub.Name())
		})
	}
}

func TestControlSubcommandHelp(t *testing.T) {
	// Cannot use t.Parallel() - uses global rootCmd
	for _, sub := range controlCmd.Commands() {
		t.Run("control_"+sub.Name()+"_help", func(t *testing.T) {
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs([]string{"control", sub.Name(), "--help"})
			err := rootCmd.Execute()
			assert.NoError(t, err, "help for control %s should succeed", sub.Name())
		})
	}
}

func TestPolicySubcommandHelp(t *testing.T) {
	// Cannot use t.Parallel() - uses global rootCmd
	for _, sub := range policyCmd.Commands() {
		t.Run("policy_"+sub.Name()+"_help", func(t *testing.T) {
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			rootCmd.SetErr(buf)
			rootCmd.SetArgs([]string{"policy", sub.Name(), "--help"})
			err := rootCmd.Execute()
			assert.NoError(t, err, "help for policy %s should succeed", sub.Name())
		})
	}
}

// ============================================================
// evidence.go — formatContextAsMarkdown (boost from 72.3%)
// ============================================================

func TestFormatContextAsMarkdownFull(t *testing.T) {
	t.Parallel()

	now := time.Now()
	task := &domain.EvidenceTask{
		ReferenceID:        "ET-0001",
		Name:               "Access Control Evidence",
		Framework:          "SOC2",
		Priority:           "high",
		CollectionInterval: "quarterly",
		NextDue:            &now,
		Description:        "Demonstrate access controls",
		Controls:           []string{"CC-6.1", "CC-6.3"},
		TugboatURL:         "https://app.tugboatlogic.com/task/1",
	}

	ctx := &EvidenceGenerationContext{
		Task: task,
		RelatedControls: []domain.Control{
			{ReferenceID: "CC-6.1", Name: "Logical Access", Description: "Controls logical access"},
			{ReferenceID: "CC-6.3", Name: "Physical Access"},
		},
		ApplicableTools:  []string{"terraform-security-indexer", "github-permissions"},
		ExistingEvidence: []string{"previous-evidence.csv", "old-report.md"},
		SourceLocations: map[string]string{
			"Terraform": "/infrastructure/terraform",
			"GitHub":    "https://github.com/org/repo",
		},
		PreviousWindows: []string{"2025-Q3", "2025-Q2"},
	}

	result := formatContextAsMarkdown(ctx, task, "2025-Q4")

	assert.Contains(t, result, "ET-0001")
	assert.Contains(t, result, "Access Control Evidence")
	assert.Contains(t, result, "SOC2")
	assert.Contains(t, result, "quarterly")
	assert.Contains(t, result, "Demonstrate access controls")
	assert.Contains(t, result, "CC-6.1")
	assert.Contains(t, result, "Logical Access")
	assert.Contains(t, result, "Controls logical access")
	assert.Contains(t, result, "terraform-security-indexer")
	assert.Contains(t, result, "previous-evidence.csv")
	assert.Contains(t, result, "2 window(s)")
	assert.Contains(t, result, "Terraform")
	assert.Contains(t, result, "Tugboat Task")
}

func TestFormatContextAsMarkdownMinimal(t *testing.T) {
	t.Parallel()

	task := &domain.EvidenceTask{
		ReferenceID: "ET-0002",
		Name:        "Simple Task",
	}

	ctx := &EvidenceGenerationContext{
		Task: task,
	}

	result := formatContextAsMarkdown(ctx, task, "2025-Q4")
	assert.Contains(t, result, "ET-0002")
	assert.Contains(t, result, "No automated tools identified")
	assert.Contains(t, result, "Manually collect")
}

func TestFormatContextAsMarkdownControlIDsOnly(t *testing.T) {
	t.Parallel()

	task := &domain.EvidenceTask{
		ReferenceID: "ET-0003",
		Name:        "Task With Control IDs",
		Controls:    []string{"CC-6.1", "CC-7.2"},
	}

	ctx := &EvidenceGenerationContext{
		Task: task,
		// No RelatedControls, just task.Controls
	}

	result := formatContextAsMarkdown(ctx, task, "2025-Q4")
	assert.Contains(t, result, "CC-6.1")
	assert.Contains(t, result, "CC-7.2")
}

// ============================================================
// evidence.go — createToolRequestForEvidence
// ============================================================

func TestCreateToolRequestForEvidence(t *testing.T) {
	t.Parallel()

	task := &domain.EvidenceTask{
		ReferenceID: "ET-0001",
		Name:        "Access Control Evidence",
		Description: "Demonstrate access controls",
	}

	cfg := &config.Config{}

	t.Run("terraform-security-indexer", func(t *testing.T) {
		t.Parallel()
		req := createToolRequestForEvidence(task, "terraform-security-indexer", cfg)
		assert.Equal(t, "ET-0001", req["task_ref"])
		assert.Equal(t, "control_mapping", req["query_type"])
	})

	t.Run("terraform-security-analyzer", func(t *testing.T) {
		t.Parallel()
		req := createToolRequestForEvidence(task, "terraform-security-analyzer", cfg)
		assert.Equal(t, "all", req["security_domain"])
	})

	t.Run("github-permissions with repo", func(t *testing.T) {
		t.Parallel()
		cfgWithGH := &config.Config{}
		cfgWithGH.Evidence.Tools.GitHub.Repository = "org/repo"
		req := createToolRequestForEvidence(task, "github-permissions", cfgWithGH)
		assert.Equal(t, "org/repo", req["repository"])
	})

	t.Run("github-permissions without repo", func(t *testing.T) {
		t.Parallel()
		req := createToolRequestForEvidence(task, "github-permissions", cfg)
		assert.Nil(t, req["repository"])
	})

	t.Run("unknown tool", func(t *testing.T) {
		t.Parallel()
		req := createToolRequestForEvidence(task, "unknown-tool", cfg)
		assert.Equal(t, "ET-0001", req["task_ref"])
		assert.Equal(t, "Access Control Evidence", req["task_name"])
	})
}

// ============================================================
// evidence.go — displayTugboatManagedMessage (boost branches)
// ============================================================

func TestDisplayTugboatManagedMessageAllBranches(t *testing.T) {
	t.Parallel()

	t.Run("with aec status and controls and guidance", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		task := &domain.EvidenceTask{
			ReferenceID: "ET-0001",
			Name:        "Full Task",
			Description: "Full description with details",
			Guidance:    "Detailed guidance for collectors",
			AecStatus:   &domain.AecStatus{Status: "enabled"},
			Controls:    []string{"CC-6.1", "CC-7.2"},
		}

		displayTugboatManagedMessage(cmd, task)
		output := buf.String()
		assert.Contains(t, output, "Full description")
		assert.Contains(t, output, "Detailed guidance")
		assert.Contains(t, output, "Tugboat Logic")
	})
}

// ============================================================
// evidence.go — displayValidationStatus (boost from 73.7%)
// ============================================================

func TestDisplayValidationStatusWithChecksAndResult(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	result := &models.ValidationResult{
		Status:            "pass",
		CompletenessScore: 92.0,
		TotalChecks:       6,
		PassedChecks:      5,
		FailedChecks:      1,
		Warnings:          2,
		Checks: []models.ValidationCheck{
			{Code: "completeness", Name: "Completeness", Status: "passed", Message: "All present"},
			{Code: "quality", Name: "Quality", Status: "passed", Message: "Good format"},
			{Code: "requirement", Name: "Requirements", Status: "passed", Message: "Met"},
			{Code: "control", Name: "Control Alignment", Status: "failed", Message: "Gap found"},
		},
		Errors: []models.ValidationError{
			{Code: "missing_file", Message: "Missing required file"},
		},
		WarningsList: []models.ValidationError{
			{Code: "naming", Message: "Non-standard filename"},
		},
	}

	displayValidationStatus(cmd, result, nil)
	output := buf.String()
	assert.Contains(t, output, "EVALUATION STATUS")
	assert.Contains(t, output, "92.0")
}

// Note: TestDisplayEvidenceTasksWithTasks would need a real service
// because it calls evidenceService.GetEvidenceTaskSummary which panics on nil.
// The empty tasks path is tested via TestDisplayEvidenceTasksEmpty.

func TestGetComprehensiveTemplates(t *testing.T) {
	t.Parallel()

	t.Run("infrastructure template", func(t *testing.T) {
		t.Parallel()
		template := getInfrastructureTemplate()
		assert.Contains(t, template, "{{TASK_REF}}")
		assert.Contains(t, template, "Infrastructure")
	})

	t.Run("personnel template", func(t *testing.T) {
		t.Parallel()
		template := getPersonnelTemplate()
		assert.Contains(t, template, "{{TASK_REF}}")
		assert.Contains(t, template, "Personnel")
	})

	t.Run("process template", func(t *testing.T) {
		t.Parallel()
		template := getProcessTemplate()
		assert.Contains(t, template, "{{TASK_REF}}")
		assert.Contains(t, template, "Process")
	})
}

// ============================================================
// evidence.go — display functions
// ============================================================

func TestDisplayReviewHeader(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	task := &domain.EvidenceTask{
		ReferenceID: "ET-0001",
		Name:        "Access Control Evidence",
	}

	displayReviewHeader(cmd, task, "2025-Q4")
	output := buf.String()
	assert.Contains(t, output, "ET-0001")
	assert.Contains(t, output, "2025-Q4")
	assert.Contains(t, output, "Access Control Evidence")
	assert.Contains(t, output, "===")
}

func TestDisplayEvidenceFiles(t *testing.T) {
	t.Parallel()

	t.Run("with files", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		files := []models.EvidenceFileRef{
			{Filename: "evidence.json", SizeBytes: 2048},
			{Filename: "summary.md", SizeBytes: 512},
		}

		displayEvidenceFiles(cmd, files, nil)
		output := buf.String()
		assert.Contains(t, output, "evidence.json")
		assert.Contains(t, output, "summary.md")
		assert.Contains(t, output, "Total: 2 files")
	})

	t.Run("no files", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		displayEvidenceFiles(cmd, nil, nil)
		output := buf.String()
		assert.Contains(t, output, "No evidence files found")
	})

	t.Run("with error", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		displayEvidenceFiles(cmd, nil, assert.AnError)
		output := buf.String()
		assert.Contains(t, output, "No evidence files found")
	})
}

func TestDisplayValidationStatus(t *testing.T) {
	t.Parallel()

	t.Run("with result", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		result := &models.ValidationResult{
			Status:            "pass",
			CompletenessScore: 85.0,
			TotalChecks:       4,
			PassedChecks:      3,
			FailedChecks:      1,
		}

		displayValidationStatus(cmd, result, nil)
		output := buf.String()
		assert.Contains(t, output, "EVALUATION STATUS")
	})

	t.Run("nil result", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		displayValidationStatus(cmd, nil, nil)
		output := buf.String()
		assert.Contains(t, output, "No evaluation results found")
	})

	t.Run("with error", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		displayValidationStatus(cmd, nil, assert.AnError)
		output := buf.String()
		assert.Contains(t, output, "No evaluation results found")
	})
}

// ============================================================
// evidence_evaluate.go — displayEvaluationResult
// ============================================================

func TestDisplayEvaluationResult(t *testing.T) {
	// displayEvaluationResult writes to os.Stdout directly via tabwriter,
	// so we can only test that it doesn't panic

	t.Run("pass result", func(t *testing.T) {
		result := &models.EvaluationResult{
			OverallStatus: models.EvaluationPass,
			OverallScore:  90.0,
			Completeness:  models.DimensionScore{Score: 95, MaxScore: 100, Weight: 0.3, Status: "pass"},
			RequirementsMatch: models.DimensionScore{Score: 85, MaxScore: 100, Weight: 0.3, Status: "pass"},
			QualityScore:  models.DimensionScore{Score: 90, MaxScore: 100, Weight: 0.2, Status: "pass"},
			ControlAlignment: models.DimensionScore{Score: 88, MaxScore: 100, Weight: 0.2, Status: "pass"},
			FileCount:     3,
			TotalBytes:    5120,
			Issues: []models.EvaluationIssue{
				{
					Severity:   models.IssueLow,
					Message:    "Minor issue",
					Location:   "file.json",
					Suggestion: "Fix it",
				},
			},
			Recommendations: []string{"Consider adding more detail"},
		}

		assert.NotPanics(t, func() {
			displayEvaluationResult(result, false)
		})
	})

	t.Run("verbose result", func(t *testing.T) {
		result := &models.EvaluationResult{
			OverallStatus:     models.EvaluationWarning,
			OverallScore:      65.0,
			Completeness:      models.DimensionScore{Score: 70, MaxScore: 100, Weight: 0.3, Status: "pass", Details: "OK"},
			RequirementsMatch: models.DimensionScore{Score: 50, MaxScore: 100, Weight: 0.3, Status: "warning", Details: "Partial"},
			QualityScore:      models.DimensionScore{Score: 80, MaxScore: 100, Weight: 0.2, Status: "pass", Details: "Good"},
			ControlAlignment:  models.DimensionScore{Score: 60, MaxScore: 100, Weight: 0.2, Status: "warning", Details: "Needs work"},
		}

		assert.NotPanics(t, func() {
			displayEvaluationResult(result, true)
		})
	})

	t.Run("no issues", func(t *testing.T) {
		result := &models.EvaluationResult{
			OverallStatus: models.EvaluationPass,
			OverallScore:  100.0,
			Completeness:  models.DimensionScore{Score: 100, MaxScore: 100, Weight: 0.3, Status: "pass"},
			RequirementsMatch: models.DimensionScore{Score: 100, MaxScore: 100, Weight: 0.3, Status: "pass"},
			QualityScore:  models.DimensionScore{Score: 100, MaxScore: 100, Weight: 0.2, Status: "pass"},
			ControlAlignment: models.DimensionScore{Score: 100, MaxScore: 100, Weight: 0.2, Status: "pass"},
		}

		assert.NotPanics(t, func() {
			displayEvaluationResult(result, false)
		})
	})
}

// ============================================================
// control.go — formatControlsList, formatControlsListSummary,
//              createControlMetadataView, createControlDetailsSection
// ============================================================

func TestFormatControlsList(t *testing.T) {
	t.Parallel()

	t.Run("with controls", func(t *testing.T) {
		t.Parallel()
		now := time.Now()
		controls := []*domain.Control{
			{ID: 1, Name: "Access Control", Category: "Security", Framework: "SOC2", Status: "implemented", ImplementedDate: &now},
			{ID: 2, Name: "Risk Assessment", Category: "Risk", Framework: "ISO27001", Status: "planned"},
		}

		// Use nil interpolator — formatter can handle it
		result := formatControlsList(controls, nil)
		assert.Contains(t, result, "Controls List")
		assert.Contains(t, result, "Found 2 controls")
		assert.Contains(t, result, "Access Control")
		assert.Contains(t, result, "Risk Assessment")
	})

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()
		result := formatControlsList(nil, nil)
		assert.Contains(t, result, "Found 0 controls")
		assert.Contains(t, result, "No controls found")
	})
}

func TestFormatControlsListSummary(t *testing.T) {
	t.Parallel()

	t.Run("with controls", func(t *testing.T) {
		t.Parallel()
		controls := []*domain.Control{
			{ID: 1, Name: "Access Control", Category: "Security", Status: "implemented"},
			{ID: 2, Name: "Risk Assessment", Category: "Risk", Status: "implemented"},
			{ID: 3, Name: "Draft Policy", Category: "Compliance", Status: "draft"},
		}

		result := formatControlsListSummary(controls, nil)
		assert.Contains(t, result, "Controls Summary")
		assert.Contains(t, result, "Found 3 controls")
		assert.Contains(t, result, "Access Control")
	})

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()
		result := formatControlsListSummary(nil, nil)
		assert.Contains(t, result, "Found 0 controls")
	})
}

func TestCreateControlMetadataView(t *testing.T) {
	t.Parallel()

	now := time.Now()
	control := &domain.Control{
		ID:              1,
		Name:            "Access Control",
		Category:        "Security",
		Framework:       "SOC2",
		Status:          "implemented",
		RiskLevel:       "high",
		ImplementedDate: &now,
		TestedDate:      &now,
		Associations: &domain.ControlAssociations{
			Policies:   3,
			Procedures: 2,
			Evidence:   5,
		},
	}

	result := createControlMetadataView(control, nil)
	assert.Contains(t, result, "Control 1 Metadata")
	assert.Contains(t, result, "Access Control")
	assert.Contains(t, result, "SOC2")
	assert.Contains(t, result, "high")
	assert.Contains(t, result, "Associations")
}

func TestCreateControlDetailsSection(t *testing.T) {
	t.Parallel()

	control := &domain.Control{
		ID:                       1,
		MasterControlID:         100,
		MasterVersionNum:        2,
		OrgID:                   999,
		OrgScopeID:              10,
		OpenIncidentCount:       3,
		RecommendedEvidenceCount: 5,
	}

	result := createControlDetailsSection(control)
	assert.Contains(t, result, "Additional Details")
	assert.Contains(t, result, "100")  // MasterControlID
	assert.Contains(t, result, "999")  // OrgID
	assert.Contains(t, result, "3")    // OpenIncidentCount
	assert.Contains(t, result, "5")    // RecommendedEvidenceCount
}

// ============================================================
// update.go — compareVersions (already 95.5% covered, fill gaps)
// ============================================================

func TestGitHubReleaseStruct(t *testing.T) {
	t.Parallel()

	// Verify GitHubRelease can be unmarshaled from JSON
	jsonStr := `{
		"tag_name": "v1.2.3",
		"name": "Release 1.2.3",
		"published_at": "2025-01-15T10:30:00Z",
		"html_url": "https://github.com/grctool/grctool/releases/tag/v1.2.3",
		"body": "Release notes here"
	}`

	var release GitHubRelease
	err := json.Unmarshal([]byte(jsonStr), &release)
	require.NoError(t, err)
	assert.Equal(t, "v1.2.3", release.TagName)
	assert.Equal(t, "Release 1.2.3", release.Name)
	assert.Equal(t, "Release notes here", release.Body)
}

// ============================================================
// evidence.go — EvidenceGenerateOptions
// ============================================================

func TestEvidenceGenerateOptionsStruct(t *testing.T) {
	t.Parallel()

	opts := EvidenceGenerateOptions{
		Format:    "csv",
		Tools:     []string{"terraform", "github"},
		OutputDir: "/tmp/evidence",
	}

	data, err := json.Marshal(opts)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"format":"csv"`)
	assert.Contains(t, string(data), `"tools":["terraform","github"]`)
}

// ============================================================
// evidence.go — displayDimensionScores, displayRequirementsChecklist,
//               displayControlAlignment, displaySubmissionRecommendation
// ============================================================

func TestDisplayDimensionScores(t *testing.T) {
	t.Parallel()

	t.Run("with checks", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		result := &models.ValidationResult{
			CompletenessScore: 85.0,
			Checks: []models.ValidationCheck{
				{Name: "Completeness Check", Status: "passed"},
				{Name: "Quality Format Check", Status: "passed"},
				{Name: "Requirement Check", Status: "passed"},
				{Name: "Control Alignment Check", Status: "failed"},
			},
		}

		displayDimensionScores(cmd, result)
		output := buf.String()
		assert.Contains(t, output, "Completeness")
		assert.Contains(t, output, "Quality")
	})

	t.Run("without checks uses overall score", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		result := &models.ValidationResult{
			CompletenessScore: 80.0,
		}

		displayDimensionScores(cmd, result)
		output := buf.String()
		assert.Contains(t, output, "Completeness")
	})
}

func TestDisplayRequirementsChecklist(t *testing.T) {
	t.Parallel()

	t.Run("with requirements and files", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		task := &domain.EvidenceTask{
			Description: "Show access control and approval workflow procedures",
			Guidance:    "Include configuration details and log examples",
		}

		displayRequirementsChecklist(cmd, task, true)
		output := buf.String()
		assert.Contains(t, output, "REQUIREMENTS")
		assert.Contains(t, output, "Access control")
	})

	t.Run("without files", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		task := &domain.EvidenceTask{
			Description: "Show access control",
		}

		displayRequirementsChecklist(cmd, task, false)
		output := buf.String()
		assert.Contains(t, output, "REQUIREMENTS")
	})

	t.Run("no requirements extracted", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		task := &domain.EvidenceTask{
			Description: "Something generic",
		}

		displayRequirementsChecklist(cmd, task, true)
		output := buf.String()
		assert.Contains(t, output, "Compliance with related controls")
	})
}

func TestDisplayControlAlignment(t *testing.T) {
	t.Parallel()

	t.Run("no controls", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		task := &domain.EvidenceTask{
			Name: "Access Control Evidence",
		}

		displayControlAlignment(cmd, task, nil)
		output := buf.String()
		assert.Contains(t, output, "CONTROL ALIGNMENT")
		assert.Contains(t, output, "No controls mapped")
	})
}

func TestDisplaySubmissionRecommendation(t *testing.T) {
	t.Parallel()

	t.Run("ready for submission", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		result := &models.ValidationResult{
			ReadyForSubmission: true,
			Status:             "pass",
		}
		task := &domain.EvidenceTask{ReferenceID: "ET-0001"}

		displaySubmissionRecommendation(cmd, task, "2025-Q4", true, true, result, false)
		output := buf.String()
		assert.Contains(t, output, "ET-0001")
	})

	t.Run("already submitted", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		task := &domain.EvidenceTask{ReferenceID: "ET-0002"}

		displaySubmissionRecommendation(cmd, task, "2025-Q4", true, false, nil, true)
		output := buf.String()
		assert.Contains(t, output, "ALREADY SUBMITTED")
	})

	t.Run("not ready no files", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		task := &domain.EvidenceTask{ReferenceID: "ET-0003"}

		displaySubmissionRecommendation(cmd, task, "2025-Q4", false, false, nil, false)
		output := buf.String()
		assert.NotEmpty(t, output)
	})
}

// ============================================================
// evidence.go — generateClaudeInstructions
// ============================================================

func TestGenerateClaudeInstructions(t *testing.T) {
	t.Parallel()

	task := &domain.EvidenceTask{
		ReferenceID: "ET-0001",
		Name:        "Access Control Evidence",
	}

	instructions := generateClaudeInstructions(task, "2025-Q4")
	assert.Contains(t, instructions, "ET-0001")
	assert.Contains(t, instructions, "Access Control Evidence")
	assert.NotEmpty(t, instructions)
}

// ============================================================
// completion.go — completePolicyRefs, completeControlRefs edge cases
// ============================================================

func TestCompletePolicyRefsNoConfig(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	// This will fail to load config and return empty
	completions, directive := completePolicyRefs(cmd, nil, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	// May return nil or empty depending on whether config can be loaded
	_ = completions
}

func TestCompleteControlRefsNoConfig(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}
	completions, directive := completeControlRefs(cmd, nil, "")
	assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
	_ = completions
}

// ============================================================
// Sync command flags verification
// ============================================================

func TestSyncCommandFlags(t *testing.T) {
	t.Parallel()

	// Find sync command
	var syncCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "sync" {
			syncCmd = cmd
			break
		}
	}
	require.NotNil(t, syncCmd)

	t.Run("has expected flags", func(t *testing.T) {
		t.Parallel()
		expectedFlags := []string{"policies", "controls", "evidence", "procedures", "dry-run", "force", "incremental"}
		for _, flag := range expectedFlags {
			assert.NotNil(t, syncCmd.Flags().Lookup(flag), "sync should have --%s flag", flag)
		}
	})
}

// ============================================================
// Additional auth command tests
// ============================================================

func TestAuthCommandSubcommands(t *testing.T) {
	t.Parallel()

	// Find auth command
	var authCmd *cobra.Command
	for _, cmd := range rootCmd.Commands() {
		if cmd.Name() == "auth" {
			authCmd = cmd
			break
		}
	}
	require.NotNil(t, authCmd)

	var names []string
	for _, cmd := range authCmd.Commands() {
		names = append(names, cmd.Name())
	}

	assert.Contains(t, names, "login")
	assert.Contains(t, names, "logout")
	assert.Contains(t, names, "status")
}

// ============================================================
// evidence.go — getCurrentQuarter (already 100% but verify)
// ============================================================

func TestGetCurrentQuarterFormat(t *testing.T) {
	t.Parallel()

	quarter := getCurrentQuarter()
	assert.NotEmpty(t, quarter)
	// Should be in format YYYY-QN
	assert.Regexp(t, `^\d{4}-Q[1-4]$`, quarter)
	assert.True(t, strings.HasPrefix(quarter, "202"))
}

// ============================================================
// edge cases for evidence_evaluate functions
// ============================================================

func TestCountPassedDimensionsAllPass(t *testing.T) {
	t.Parallel()

	result := &models.EvaluationResult{
		Completeness:      models.DimensionScore{Status: "pass"},
		RequirementsMatch: models.DimensionScore{Status: "pass"},
		QualityScore:      models.DimensionScore{Status: "pass"},
		ControlAlignment:  models.DimensionScore{Status: "pass"},
	}
	assert.Equal(t, 4, countPassedDimensions(result))
}

func TestCountFailedDimensionsNoneFail(t *testing.T) {
	t.Parallel()

	result := &models.EvaluationResult{
		Completeness:      models.DimensionScore{Status: "pass"},
		RequirementsMatch: models.DimensionScore{Status: "warning"},
		QualityScore:      models.DimensionScore{Status: "pass"},
		ControlAlignment:  models.DimensionScore{Status: "warning"},
	}
	assert.Equal(t, 0, countFailedDimensions(result))
}

func TestCountWarningsNoIssues(t *testing.T) {
	t.Parallel()

	result := &models.EvaluationResult{
		Issues: []models.EvaluationIssue{},
	}
	assert.Equal(t, 0, countWarnings(result))
}

func TestConvertIssuesToValidationErrorsNoHighSeverity(t *testing.T) {
	t.Parallel()

	issues := []models.EvaluationIssue{
		{Severity: models.IssueLow, Message: "Minor"},
		{Severity: models.IssueInfo, Message: "FYI"},
	}
	errors := convertIssuesToValidationErrors(issues)
	assert.Empty(t, errors)
}

// ============================================================
// config.go — getStatusSymbol, getCheckStatusSymbol, printValidationResults, fileExists
// ============================================================

func TestGetStatusSymbol(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "PASS", getStatusSymbol(true))
	assert.Equal(t, "FAIL", getStatusSymbol(false))
}

func TestGetCheckStatusSymbol(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "✓", getCheckStatusSymbol("pass"))
	assert.Equal(t, "✓", getCheckStatusSymbol("PASS"))
	assert.Equal(t, "✗", getCheckStatusSymbol("fail"))
	assert.Equal(t, "⚠", getCheckStatusSymbol("warning"))
	assert.Equal(t, "?", getCheckStatusSymbol("unknown"))
}

func TestFileExists(t *testing.T) {
	t.Parallel()

	t.Run("existing file", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "test.txt")
		require.NoError(t, os.WriteFile(path, []byte("test"), 0644))
		assert.True(t, fileExists(path))
	})

	t.Run("non-existing file", func(t *testing.T) {
		t.Parallel()
		assert.False(t, fileExists("/nonexistent/path/to/file.txt"))
	})
}

func TestPrintValidationResults(t *testing.T) {
	t.Parallel()

	t.Run("valid result with checks", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		result := &configService.ValidationResult{
			Valid:    true,
			Duration: 100 * time.Millisecond,
			Checks: map[string]configService.ValidationCheck{
				"config_file": {Name: "Config File", Status: "pass", Message: "Found", Duration: 10 * time.Millisecond},
				"storage":     {Name: "Storage", Status: "pass", Message: "OK", Duration: 20 * time.Millisecond},
			},
		}

		printValidationResults(cmd, result)
		output := buf.String()
		assert.Contains(t, output, "Configuration Validation Report")
		assert.Contains(t, output, "PASS")
	})

	t.Run("failed result with errors", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		result := &configService.ValidationResult{
			Valid:    false,
			Duration: 50 * time.Millisecond,
			Checks: map[string]configService.ValidationCheck{
				"config_file": {Name: "Config File", Status: "fail", Message: "Not found", Duration: 5 * time.Millisecond},
			},
			Errors: []string{"Config file not found", "Storage dir not accessible"},
		}

		printValidationResults(cmd, result)
		output := buf.String()
		assert.Contains(t, output, "FAIL")
		assert.Contains(t, output, "Errors")
		assert.Contains(t, output, "Config file not found")
	})
}

// ============================================================
// update.go — compareVersions
// ============================================================

func TestCompareVersionsAdditional(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		v1       string
		v2       string
		expected int
	}{
		{"equal", "1.0.0", "1.0.0", 0},
		{"v1 less", "1.0.0", "1.0.1", -1},
		{"v1 greater", "1.1.0", "1.0.0", 1},
		{"major diff", "2.0.0", "1.0.0", 1},
		{"with v prefix", "v1.0.0", "v1.0.1", -1},
		{"different lengths", "1.0", "1.0.0", 0},
		{"pre-release stripped", "1.0.0-beta", "1.0.0", 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, compareVersions(tt.v1, tt.v2))
		})
	}
}

// ============================================================
// update.go — command structure
// ============================================================

func TestUpdateCommandStructure(t *testing.T) {
	t.Parallel()

	t.Run("update has check and install subcommands", func(t *testing.T) {
		t.Parallel()
		var names []string
		for _, cmd := range updateCmd.Commands() {
			names = append(names, cmd.Name())
		}
		assert.Contains(t, names, "check")
		assert.Contains(t, names, "install")
	})

	t.Run("install command flags", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, installCmd.Flags().Lookup("yes"))
		assert.NotNil(t, installCmd.Flags().Lookup("system"))
	})
}

// ============================================================
// evidence.go — getComplianceTemplate, getMonitoringTemplate, getDataTemplate
// ============================================================

func TestGetComplianceTemplate(t *testing.T) {
	t.Parallel()
	template := getComplianceTemplate()
	assert.NotEmpty(t, template)
	// Currently returns default template
	assert.Contains(t, template, "{{TASK_REF}}")
}

func TestGetMonitoringTemplate(t *testing.T) {
	t.Parallel()
	template := getMonitoringTemplate()
	assert.NotEmpty(t, template)
	// Currently returns default template
	assert.Contains(t, template, "{{TASK_REF}}")
}

func TestGetDataTemplate(t *testing.T) {
	t.Parallel()
	template := getDataTemplate()
	assert.NotEmpty(t, template)
	// Currently returns default template
	assert.Contains(t, template, "{{TASK_REF}}")
}

// ============================================================
// evidence.go — saveAssemblyContext
// ============================================================

func TestSaveAssemblyContext(t *testing.T) {
	tmpDir := t.TempDir()

	task := &domain.EvidenceTask{
		ID:          327992,
		ReferenceID: "ET-0001",
		Name:        "Access Control Evidence",
	}

	ctx := &AssemblyContext{
		ComprehensivePrompt: "Test prompt content",
		ClaudeInstructions:  "Test instructions",
		EvidenceTemplate:    "# {{TASK_REF}} - {{TASK_NAME}}",
	}

	paths, err := saveAssemblyContext(task, "2025-Q4", ctx, tmpDir)
	require.NoError(t, err)
	require.NotNil(t, paths)

	// Verify files were created
	assert.FileExists(t, paths.PromptFile)
	assert.FileExists(t, paths.InstructionsFile)
	assert.FileExists(t, paths.TemplateFile)
	assert.DirExists(t, paths.ToolDataDir)

	// Verify prompt content
	promptData, err := os.ReadFile(paths.PromptFile)
	require.NoError(t, err)
	assert.Equal(t, "Test prompt content", string(promptData))

	// Verify template variables were applied
	templateData, err := os.ReadFile(paths.TemplateFile)
	require.NoError(t, err)
	assert.Contains(t, string(templateData), "ET-0001")
	assert.Contains(t, string(templateData), "Access Control Evidence")
	assert.NotContains(t, string(templateData), "{{TASK_REF}}")
}

// ============================================================
// evidence.go — saveEvidenceContext (saves markdown context)
// ============================================================

func TestSaveEvidenceContext(t *testing.T) {
	tmpDir := t.TempDir()

	task := &domain.EvidenceTask{
		ID:          327992,
		ReferenceID: "ET-0001",
		Name:        "Access Control Evidence",
	}

	contextContent := "# Evidence Context\n\nThis is test context content."

	contextPath, err := saveEvidenceContext(task, "2025-Q4", contextContent, tmpDir)
	require.NoError(t, err)
	assert.NotEmpty(t, contextPath)
	assert.FileExists(t, contextPath)

	// Verify content
	data, err := os.ReadFile(contextPath)
	require.NoError(t, err)
	assert.Equal(t, contextContent, string(data))
}

// ============================================================
// validate_data.go — displayValidationSummary
// ============================================================

func TestDisplayValidationSummary(t *testing.T) {
	t.Parallel()

	t.Run("basic report", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		report := &validation.ValidationReport{
			Summary: validation.ValidationSummary{
				Status:             "pass",
				OverallScore:       85.0,
				TotalPolicies:      10,
				TotalControls:      50,
				TotalEvidenceTasks: 30,
				CriticalIssues:     0,
				Warnings:           2,
			},
			Policies: validation.PolicyValidation{
				ContentCompleteness: 90.0,
				WithContent:         9,
				MissingContent:      1,
			},
			Controls: validation.ControlValidation{
				LinkageCompleteness: 80.0,
				WithPolicyLinks:     40,
				MissingPolicyLinks:  10,
			},
			Evidence: validation.EvidenceValidation{
				ContentCompleteness: 75.0,
				WithGuidance:        22,
				MissingGuidance:     8,
			},
		}

		displayValidationSummary(cmd, report, false)
		output := buf.String()
		assert.Contains(t, output, "DATA VALIDATION REPORT")
		assert.Contains(t, output, "pass")
		assert.Contains(t, output, "85.0%")
		assert.Contains(t, output, "10 policies")
	})

	t.Run("detailed report with issues", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		report := &validation.ValidationReport{
			Summary: validation.ValidationSummary{
				Status:       "warning",
				OverallScore: 60.0,
			},
			Issues: []validation.ValidationIssue{
				{Type: "critical", Category: "policy", Description: "Missing content"},
				{Type: "warning", Category: "control", Description: "Missing link"},
			},
		}

		displayValidationSummary(cmd, report, true)
		output := buf.String()
		assert.Contains(t, output, "Detailed Issues")
		assert.Contains(t, output, "Missing content")
	})

	t.Run("detailed report with many issues truncates", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		issues := make([]validation.ValidationIssue, 25)
		for i := range issues {
			issues[i] = validation.ValidationIssue{
				Type:        "warning",
				Category:    "control",
				Description: "Issue",
			}
		}

		report := &validation.ValidationReport{
			Summary: validation.ValidationSummary{OverallScore: 50.0},
			Issues:  issues,
		}

		displayValidationSummary(cmd, report, true)
		output := buf.String()
		assert.Contains(t, output, "and 5 more issues")
	})
}

// ============================================================
// evidence.go — displayEvidenceMap (with empty/nil data)
// ============================================================

func TestDisplayEvidenceMapEmpty(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	mapResult := &evidence.EvidenceMapResult{
		Summary: &evidence.EvidenceMapSummary{},
	}

	// Call with empty tasks - should not panic
	err := displayEvidenceMap(cmd, mapResult)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No evidence tasks found")
}

func TestDisplayEvidenceMapWithData(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	mapResult := &evidence.EvidenceMapResult{
		Tasks: []domain.EvidenceTask{
			{ID: 1, ReferenceID: "ET-0001", Name: "Test Task"},
		},
		Summary: &evidence.EvidenceMapSummary{
			TotalTasks:      1,
			TotalControls:   2,
			TotalPolicies:   3,
			FrameworkCounts: map[string]int{"SOC2": 1},
		},
		FrameworkGroups: map[string][]domain.EvidenceTask{
			"SOC2": {{ID: 1, ReferenceID: "ET-0001", Name: "Test Task"}},
		},
	}

	err := displayEvidenceMap(cmd, mapResult)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "1 tasks")
	assert.Contains(t, output, "SOC2")
}

// ============================================================
// evidence_evaluate.go — saveEvaluationResultToFile edge cases
// ============================================================

func TestSaveEvaluationResultToFileYML(t *testing.T) {
	t.Parallel()

	result := &models.EvaluationResult{
		TaskRef:      "ET-0001",
		OverallScore: 85.5,
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "result.yml")

	err := saveEvaluationResultToFile(result, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(data), "ET-0001")
}
