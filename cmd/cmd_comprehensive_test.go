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
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/models"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ============================================================
// status.go — pure helper function tests
// ============================================================

func TestFormatStateLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		state    models.LocalEvidenceState
		expected string
	}{
		{models.StateNoEvidence, "No Evidence"},
		{models.StateGenerated, "Generated"},
		{models.StateValidated, "Validated"},
		{models.StateSubmitted, "Submitted"},
		{models.StateAccepted, "Accepted"},
		{models.StateRejected, "Rejected"},
		{models.LocalEvidenceState("custom"), "custom"},
	}

	for _, tt := range tests {
		t.Run(string(tt.state), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, formatStateLabel(tt.state))
		})
	}
}

func TestFormatAutomationLabel(t *testing.T) {
	t.Parallel()

	tests := []struct {
		level    models.AutomationCapability
		expected string
	}{
		{models.AutomationFully, "Fully Automated"},
		{models.AutomationPartially, "Partially Automated"},
		{models.AutomationManual, "Manual Only"},
		{models.AutomationUnknown, "Unknown"},
		{models.AutomationCapability("other"), "other"},
	}

	for _, tt := range tests {
		t.Run(string(tt.level), func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, formatAutomationLabel(tt.level))
		})
	}
}

func TestFormatBytes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		bytes    int64
		expected string
	}{
		{"zero bytes", 0, "0B"},
		{"small bytes", 512, "512B"},
		{"one KB", 1024, "1.0KB"},
		{"several KB", 5120, "5.0KB"},
		{"one MB", 1024 * 1024, "1.0MB"},
		{"fractional MB", 1536 * 1024, "1.5MB"},
		{"one GB", 1024 * 1024 * 1024, "1.0GB"},
		{"large GB", 3 * 1024 * 1024 * 1024, "3.0GB"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, formatBytes(tt.bytes))
		})
	}
}

func TestFormatTimestamp(t *testing.T) {
	t.Parallel()

	t.Run("nil timestamp", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "N/A", formatTimestamp(nil))
	})

	t.Run("valid timestamp", func(t *testing.T) {
		t.Parallel()
		ts := time.Date(2025, 3, 15, 14, 30, 45, 0, time.UTC)
		assert.Equal(t, "2025-03-15 14:30:45", formatTimestamp(&ts))
	})
}

func TestFormatTimestampRelative(t *testing.T) {
	t.Parallel()

	t.Run("nil timestamp", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, "N/A", formatTimestampRelative(nil))
	})

	t.Run("yesterday", func(t *testing.T) {
		t.Parallel()
		ts := time.Now().Add(-30 * time.Hour)
		assert.Equal(t, "yesterday", formatTimestampRelative(&ts))
	})

	t.Run("days ago", func(t *testing.T) {
		t.Parallel()
		ts := time.Now().Add(-4 * 24 * time.Hour)
		assert.Contains(t, formatTimestampRelative(&ts), "days ago")
	})

	t.Run("weeks ago", func(t *testing.T) {
		t.Parallel()
		ts := time.Now().Add(-14 * 24 * time.Hour)
		assert.Contains(t, formatTimestampRelative(&ts), "weeks ago")
	})

	t.Run("older date uses full format", func(t *testing.T) {
		t.Parallel()
		ts := time.Now().Add(-60 * 24 * time.Hour)
		result := formatTimestampRelative(&ts)
		// Should be in YYYY-MM-DD format
		assert.Regexp(t, `^\d{4}-\d{2}-\d{2}$`, result)
	})
}

func TestTruncateChecksum(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"short checksum", "abc", "abc"},
		{"exactly 8", "abcdefgh", "abcdefgh"},
		{"long checksum", "abcdefghij123456", "abcdefgh..."},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, truncateChecksum(tt.input))
		})
	}
}

func TestNormalizeTaskRef(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"already normalized", "ET-0001", "ET-0001"},
		{"short ref", "ET-1", "ET-0001"},
		{"medium ref", "ET-47", "ET-0047"},
		{"three digit ref", "ET-101", "ET-0101"},
		{"non-ET ref", "CC-01", "CC-01"},
		{"unparseable", "RANDOM", "RANDOM"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, normalizeTaskRef(tt.input))
		})
	}
}

func TestGetLatestTimestamp(t *testing.T) {
	t.Parallel()

	now := time.Now()
	earlier := now.Add(-24 * time.Hour)

	t.Run("no timestamps", func(t *testing.T) {
		t.Parallel()
		task := &models.EvidenceTaskState{}
		assert.Nil(t, getLatestTimestamp(task))
	})

	t.Run("only submitted", func(t *testing.T) {
		t.Parallel()
		task := &models.EvidenceTaskState{LastSubmittedAt: &now}
		assert.Equal(t, &now, getLatestTimestamp(task))
	})

	t.Run("only generated", func(t *testing.T) {
		t.Parallel()
		task := &models.EvidenceTaskState{LastGeneratedAt: &now}
		assert.Equal(t, &now, getLatestTimestamp(task))
	})

	t.Run("generated is more recent", func(t *testing.T) {
		t.Parallel()
		task := &models.EvidenceTaskState{
			LastSubmittedAt: &earlier,
			LastGeneratedAt: &now,
		}
		assert.Equal(t, &now, getLatestTimestamp(task))
	})

	t.Run("submitted is more recent", func(t *testing.T) {
		t.Parallel()
		task := &models.EvidenceTaskState{
			LastSubmittedAt: &now,
			LastGeneratedAt: &earlier,
		}
		assert.Equal(t, &now, getLatestTimestamp(task))
	})
}

func TestGetRecentActivity(t *testing.T) {
	t.Parallel()

	now := time.Now()
	hour := time.Hour
	tasks := []*models.EvidenceTaskState{
		{TaskRef: "ET-0001", LastGeneratedAt: timePtr(now.Add(-3 * hour))},
		{TaskRef: "ET-0002", LastGeneratedAt: timePtr(now.Add(-1 * hour))},
		{TaskRef: "ET-0003"}, // no timestamp
		{TaskRef: "ET-0004", LastSubmittedAt: timePtr(now.Add(-2 * hour))},
	}

	t.Run("returns sorted by most recent", func(t *testing.T) {
		t.Parallel()
		result := getRecentActivity(tasks, 10)
		require.Len(t, result, 4)
		// Most recent first
		assert.Equal(t, "ET-0002", result[0].TaskRef)
		assert.Equal(t, "ET-0004", result[1].TaskRef)
		assert.Equal(t, "ET-0001", result[2].TaskRef)
		// No timestamp goes to end
		assert.Equal(t, "ET-0003", result[3].TaskRef)
	})

	t.Run("respects limit", func(t *testing.T) {
		t.Parallel()
		result := getRecentActivity(tasks, 2)
		assert.Len(t, result, 2)
	})

	t.Run("empty list", func(t *testing.T) {
		t.Parallel()
		result := getRecentActivity(nil, 10)
		assert.Empty(t, result)
	})
}

func TestSortWindowsByDate(t *testing.T) {
	t.Parallel()

	windows := map[string]models.WindowState{
		"2025-Q1": {Window: "2025-Q1", FileCount: 1},
		"2025-Q4": {Window: "2025-Q4", FileCount: 4},
		"2025-Q2": {Window: "2025-Q2", FileCount: 2},
	}

	result := sortWindowsByDate(windows)
	require.Len(t, result, 3)
	// Descending order
	assert.Equal(t, "2025-Q4", result[0].Window)
	assert.Equal(t, "2025-Q2", result[1].Window)
	assert.Equal(t, "2025-Q1", result[2].Window)
}

// ============================================================
// control.go — pure helper function tests
// ============================================================

func TestTruncateString(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    string
		maxLen   int
		expected string
	}{
		{"short string", "hello", 10, "hello"},
		{"exact length", "hello", 5, "hello"},
		{"needs truncation", "hello world", 8, "hello..."},
		{"empty string", "", 5, ""},
		{"min length 4", "abcdefgh", 4, "a..."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, truncateString(tt.input, tt.maxLen))
		})
	}
}

func TestFilterControls(t *testing.T) {
	t.Parallel()

	controls := []*domain.Control{
		{ID: "1", Name: "SOC2 Access", Framework: "SOC2", Status: "implemented", Category: "Access Control"},
		{ID: "2", Name: "ISO Risk", Framework: "ISO27001", Status: "implemented", Category: "Risk Management"},
		{ID: "3", Name: "SOC2 Monitor", Framework: "SOC2", Status: "not_applicable", Category: "Monitoring"},
		{ID: "4", Name: "ISO Access", Framework: "ISO27001", Status: "planned", Category: "Access Control"},
	}

	t.Run("no filters returns all", func(t *testing.T) {
		t.Parallel()
		result := filterControls(controls, "", "", "")
		assert.Len(t, result, 4)
	})

	t.Run("filter by framework", func(t *testing.T) {
		t.Parallel()
		result := filterControls(controls, "SOC2", "", "")
		assert.Len(t, result, 2)
		for _, c := range result {
			assert.Equal(t, "SOC2", c.Framework)
		}
	})

	t.Run("filter by status", func(t *testing.T) {
		t.Parallel()
		result := filterControls(controls, "", "implemented", "")
		assert.Len(t, result, 2)
	})

	t.Run("filter by category (substring match)", func(t *testing.T) {
		t.Parallel()
		result := filterControls(controls, "", "", "Access")
		assert.Len(t, result, 2)
	})

	t.Run("combined filters", func(t *testing.T) {
		t.Parallel()
		result := filterControls(controls, "SOC2", "implemented", "")
		assert.Len(t, result, 1)
		assert.Equal(t, "1", result[0].ID)
	})

	t.Run("case insensitive framework", func(t *testing.T) {
		t.Parallel()
		result := filterControls(controls, "soc2", "", "")
		assert.Len(t, result, 2)
	})

	t.Run("no matches", func(t *testing.T) {
		t.Parallel()
		result := filterControls(controls, "NIST", "", "")
		assert.Empty(t, result)
	})
}

// ============================================================
// evidence_evaluate.go — pure helper function tests
// ============================================================

func TestGetStatusEmoji(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "✓", getStatusEmoji(models.EvaluationPass))
	assert.Equal(t, "⚠", getStatusEmoji(models.EvaluationWarning))
	assert.Equal(t, "✗", getStatusEmoji(models.EvaluationFail))
	assert.Equal(t, "?", getStatusEmoji(models.EvaluationStatus("unknown")))
}

func TestGetDimensionStatusEmoji(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "✓", getDimensionStatusEmoji("pass"))
	assert.Equal(t, "⚠", getDimensionStatusEmoji("warning"))
	assert.Equal(t, "✗", getDimensionStatusEmoji("fail"))
	assert.Equal(t, "?", getDimensionStatusEmoji("other"))
}

func TestGetSeverityEmoji(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "🔴", getSeverityEmoji(models.IssueCritical))
	assert.Equal(t, "🟠", getSeverityEmoji(models.IssueHigh))
	assert.Equal(t, "🟡", getSeverityEmoji(models.IssueMedium))
	assert.Equal(t, "🟢", getSeverityEmoji(models.IssueLow))
	assert.Equal(t, "ℹ️", getSeverityEmoji(models.IssueInfo))
	assert.Equal(t, "•", getSeverityEmoji(models.IssueSeverity("unknown")))
}

func TestGetSeverityFromStatus(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "error", getSeverityFromStatus("fail"))
	assert.Equal(t, "warning", getSeverityFromStatus("warning"))
	assert.Equal(t, "info", getSeverityFromStatus("pass"))
	assert.Equal(t, "info", getSeverityFromStatus("other"))
}

func TestCountPassedDimensions(t *testing.T) {
	t.Parallel()

	result := &models.EvaluationResult{
		Completeness:      models.DimensionScore{Status: "pass"},
		RequirementsMatch: models.DimensionScore{Status: "fail"},
		QualityScore:      models.DimensionScore{Status: "pass"},
		ControlAlignment:  models.DimensionScore{Status: "warning"},
	}

	assert.Equal(t, 2, countPassedDimensions(result))
}

func TestCountFailedDimensions(t *testing.T) {
	t.Parallel()

	result := &models.EvaluationResult{
		Completeness:      models.DimensionScore{Status: "fail"},
		RequirementsMatch: models.DimensionScore{Status: "fail"},
		QualityScore:      models.DimensionScore{Status: "pass"},
		ControlAlignment:  models.DimensionScore{Status: "warning"},
	}

	assert.Equal(t, 2, countFailedDimensions(result))
}

func TestCountWarnings(t *testing.T) {
	t.Parallel()

	result := &models.EvaluationResult{
		Issues: []models.EvaluationIssue{
			{Severity: models.IssueCritical},
			{Severity: models.IssueHigh},
			{Severity: models.IssueMedium},
			{Severity: models.IssueLow},
			{Severity: models.IssueInfo},
		},
	}

	// Only high and medium count as warnings
	assert.Equal(t, 2, countWarnings(result))
}

func TestConvertIssuesToValidationErrors(t *testing.T) {
	t.Parallel()

	issues := []models.EvaluationIssue{
		{Severity: models.IssueCritical, Category: "completeness", Message: "Missing file", Location: "root"},
		{Severity: models.IssueHigh, Category: "quality", Message: "Bad format", Location: "file.json"},
		{Severity: models.IssueMedium, Category: "requirements", Message: "Minor gap"},
		{Severity: models.IssueLow, Category: "other", Message: "Info only"},
	}

	errors := convertIssuesToValidationErrors(issues)

	// Only critical and high are converted
	assert.Len(t, errors, 2)
	assert.Equal(t, "completeness_error", errors[0].Code)
	assert.Equal(t, "Missing file", errors[0].Message)
	assert.Equal(t, "quality_error", errors[1].Code)
}

func TestBuildValidationChecks(t *testing.T) {
	t.Parallel()

	result := &models.EvaluationResult{
		Completeness:      models.DimensionScore{Status: "pass", Details: "All files present"},
		RequirementsMatch: models.DimensionScore{Status: "warning", Details: "Partial match"},
		QualityScore:      models.DimensionScore{Status: "fail", Details: "Poor format"},
		ControlAlignment:  models.DimensionScore{Status: "pass", Details: "Well aligned"},
	}

	checks := buildValidationChecks(result)

	require.Len(t, checks, 4)
	assert.Equal(t, "completeness", checks[0].Code)
	assert.Equal(t, "info", checks[0].Severity)
	assert.Equal(t, "requirements_match", checks[1].Code)
	assert.Equal(t, "warning", checks[1].Severity)
	assert.Equal(t, "quality", checks[2].Code)
	assert.Equal(t, "error", checks[2].Severity)
	assert.Equal(t, "control_alignment", checks[3].Code)
}

func TestSaveEvaluationResultToFile(t *testing.T) {
	t.Parallel()

	result := &models.EvaluationResult{
		TaskRef:      "ET-0001",
		OverallScore: 85.5,
		FileCount:    3,
	}

	t.Run("save as JSON", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "result.json")

		err := saveEvaluationResultToFile(result, path)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var loaded models.EvaluationResult
		require.NoError(t, json.Unmarshal(data, &loaded))
		assert.Equal(t, "ET-0001", loaded.TaskRef)
		assert.InDelta(t, 85.5, loaded.OverallScore, 0.01)
	})

	t.Run("save as YAML", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "result.yaml")

		err := saveEvaluationResultToFile(result, path)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var loaded models.EvaluationResult
		require.NoError(t, yaml.Unmarshal(data, &loaded))
		assert.Equal(t, "ET-0001", loaded.TaskRef)
	})

	t.Run("default format is JSON", func(t *testing.T) {
		t.Parallel()
		tmpDir := t.TempDir()
		path := filepath.Join(tmpDir, "result.txt")

		err := saveEvaluationResultToFile(result, path)
		require.NoError(t, err)

		data, err := os.ReadFile(path)
		require.NoError(t, err)

		var loaded models.EvaluationResult
		require.NoError(t, json.Unmarshal(data, &loaded))
	})
}

func TestSaveAllEvaluationResults(t *testing.T) {
	t.Parallel()

	results := []*models.EvaluationResult{
		{TaskRef: "ET-0001", OverallScore: 85.0},
		{TaskRef: "ET-0002", OverallScore: 72.0},
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "all-results.json")

	err := saveAllEvaluationResults(results, path)
	require.NoError(t, err)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var loaded []*models.EvaluationResult
	require.NoError(t, json.Unmarshal(data, &loaded))
	assert.Len(t, loaded, 2)
}

// ============================================================
// evidence_setup.go — updateConfigCollectorURL
// ============================================================

func TestUpdateConfigCollectorURL(t *testing.T) {
	t.Run("adds collector URL to existing config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".grctool.yaml")

		initialConfig := map[string]interface{}{
			"tugboat": map[string]interface{}{
				"base_url": "https://test.com",
			},
		}
		data, err := yaml.Marshal(initialConfig)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configPath, data, 0644))

		err = updateConfigCollectorURL(configPath, "ET-0001", "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/")
		require.NoError(t, err)

		// Verify the URL was added
		updatedData, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var updated map[string]interface{}
		require.NoError(t, yaml.Unmarshal(updatedData, &updated))

		tugboat := updated["tugboat"].(map[string]interface{})
		urls := tugboat["collector_urls"].(map[string]interface{})
		assert.Equal(t, "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/", urls["ET-0001"])
	})

	t.Run("overwrites existing collector URL", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".grctool.yaml")

		initialConfig := map[string]interface{}{
			"tugboat": map[string]interface{}{
				"base_url": "https://test.com",
				"collector_urls": map[string]interface{}{
					"ET-0001": "https://old-url/",
				},
			},
		}
		data, err := yaml.Marshal(initialConfig)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configPath, data, 0644))

		newURL := "https://openapi.tugboatlogic.com/api/v0/evidence/collector/999/"
		err = updateConfigCollectorURL(configPath, "ET-0001", newURL)
		require.NoError(t, err)

		updatedData, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var updated map[string]interface{}
		require.NoError(t, yaml.Unmarshal(updatedData, &updated))

		tugboat := updated["tugboat"].(map[string]interface{})
		urls := tugboat["collector_urls"].(map[string]interface{})
		assert.Equal(t, newURL, urls["ET-0001"])
	})

	t.Run("creates tugboat section if missing", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".grctool.yaml")

		initialConfig := map[string]interface{}{
			"storage": map[string]interface{}{"data_dir": "./"},
		}
		data, err := yaml.Marshal(initialConfig)
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configPath, data, 0644))

		err = updateConfigCollectorURL(configPath, "ET-0002", "https://openapi.tugboatlogic.com/api/v0/evidence/collector/100/")
		require.NoError(t, err)

		updatedData, err := os.ReadFile(configPath)
		require.NoError(t, err)

		var updated map[string]interface{}
		require.NoError(t, yaml.Unmarshal(updatedData, &updated))

		tugboat := updated["tugboat"].(map[string]interface{})
		urls := tugboat["collector_urls"].(map[string]interface{})
		assert.Equal(t, "https://openapi.tugboatlogic.com/api/v0/evidence/collector/100/", urls["ET-0002"])
	})

	t.Run("backup is removed on success", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, ".grctool.yaml")

		data, err := yaml.Marshal(map[string]interface{}{"tugboat": map[string]interface{}{}})
		require.NoError(t, err)
		require.NoError(t, os.WriteFile(configPath, data, 0644))

		err = updateConfigCollectorURL(configPath, "ET-0001", "https://openapi.tugboatlogic.com/api/v0/evidence/collector/1/")
		require.NoError(t, err)

		_, statErr := os.Stat(configPath + ".backup")
		assert.True(t, os.IsNotExist(statErr), "backup should be removed on success")
	})
}

// ============================================================
// evidence_migrate.go — oldFormatRegex tests
// ============================================================

func TestOldFormatRegex(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		matches bool
		taskRef string
	}{
		{"old format", "ET-0001_access_control_evidence", true, "ET-0001"},
		{"old format short", "ET-0047_some_task", true, "ET-0047"},
		{"new format", "access_control_ET-0001_327992", false, ""},
		{"not a task dir", "random_directory", false, ""},
		{"no underscore", "ET-0001", false, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			matches := oldFormatRegex.FindStringSubmatch(tt.input)
			if tt.matches {
				require.True(t, len(matches) >= 3)
				assert.Equal(t, tt.taskRef, matches[1])
			} else {
				assert.True(t, len(matches) < 3)
			}
		})
	}
}

// ============================================================
// completion.go — completeToolNames
// ============================================================

func TestCompleteToolNames(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}

	t.Run("all tools with empty prefix", func(t *testing.T) {
		t.Parallel()
		completions, directive := completeToolNames(cmd, nil, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		assert.True(t, len(completions) > 10, "should return many tools")
	})

	t.Run("filter by prefix", func(t *testing.T) {
		t.Parallel()
		completions, _ := completeToolNames(cmd, nil, "terraform")
		for _, c := range completions {
			assert.Contains(t, c, "terraform")
		}
	})

	t.Run("no matches for bogus prefix", func(t *testing.T) {
		t.Parallel()
		completions, _ := completeToolNames(cmd, nil, "zzz-nonexistent-tool")
		assert.Empty(t, completions)
	})
}

// ============================================================
// Command structure tests — verify command tree, flags, help
// ============================================================

func TestStatusCommandStructure(t *testing.T) {
	// Cannot use t.Parallel() — sub-tests use global rootCmd

	t.Run("status command exists on root", func(t *testing.T) {
		t.Parallel()
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "status" {
				found = true
				break
			}
		}
		assert.True(t, found, "root command should have 'status' subcommand")
	})

	t.Run("status has task and scan subcommands", func(t *testing.T) {
		t.Parallel()
		var names []string
		for _, cmd := range evidenceStatusCmd.Commands() {
			names = append(names, cmd.Name())
		}
		assert.Contains(t, names, "task")
		assert.Contains(t, names, "scan")
	})

	t.Run("status flags", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, evidenceStatusCmd.Flags().Lookup("filter"))
		assert.NotNil(t, evidenceStatusCmd.Flags().Lookup("automation"))
		assert.NotNil(t, evidenceStatusCmd.Flags().Lookup("verbose"))
	})

	t.Run("status help output", func(t *testing.T) {
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"status", "--help"})
		_ = rootCmd.Execute()
		output := buf.String()
		assert.Contains(t, output, "status")
	})
}

func TestToolCommandStructure(t *testing.T) {
	// Cannot use t.Parallel() — sub-tests use global rootCmd

	t.Run("tool command exists on root", func(t *testing.T) {
		t.Parallel()
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "tool" {
				found = true
				break
			}
		}
		assert.True(t, found, "root command should have 'tool' subcommand")
	})

	t.Run("tool has list and stats subcommands", func(t *testing.T) {
		t.Parallel()
		var names []string
		for _, cmd := range toolCmd.Commands() {
			names = append(names, cmd.Name())
		}
		assert.Contains(t, names, "list")
		assert.Contains(t, names, "stats")
	})

	t.Run("tool persistent flags", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, toolCmd.PersistentFlags().Lookup("output"))
		assert.NotNil(t, toolCmd.PersistentFlags().Lookup("task-ref"))
		assert.NotNil(t, toolCmd.PersistentFlags().Lookup("quiet"))
	})

	t.Run("tool help output", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		rootCmd.SetOut(buf)
		rootCmd.SetErr(buf)
		rootCmd.SetArgs([]string{"tool", "--help"})
		_ = rootCmd.Execute()
		output := buf.String()
		assert.Contains(t, output, "tool")
	})
}

func TestControlCommandStructure(t *testing.T) {
	// Cannot use t.Parallel() — sub-tests use global rootCmd

	t.Run("control command exists on root", func(t *testing.T) {
		t.Parallel()
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "control" {
				found = true
				break
			}
		}
		assert.True(t, found, "root command should have 'control' subcommand")
	})

	t.Run("control has view and list subcommands", func(t *testing.T) {
		t.Parallel()
		var names []string
		for _, cmd := range controlCmd.Commands() {
			names = append(names, cmd.Name())
		}
		assert.Contains(t, names, "view")
		assert.Contains(t, names, "list")
	})

	t.Run("control view flags", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, controlViewCmd.Flags().Lookup("output"))
		assert.NotNil(t, controlViewCmd.Flags().Lookup("summary"))
		assert.NotNil(t, controlViewCmd.Flags().Lookup("metadata-only"))
		assert.NotNil(t, controlViewCmd.Flags().Lookup("details"))
	})

	t.Run("control list flags", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, controlListCmd.Flags().Lookup("output"))
		assert.NotNil(t, controlListCmd.Flags().Lookup("summary"))
		assert.NotNil(t, controlListCmd.Flags().Lookup("framework"))
		assert.NotNil(t, controlListCmd.Flags().Lookup("status"))
		assert.NotNil(t, controlListCmd.Flags().Lookup("category"))
	})

	// Note: control help test moved to TestControlHelp to avoid rootCmd parallel race
}

func TestControlHelp(t *testing.T) {
	// Cannot use t.Parallel() - uses global rootCmd
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"control", "--help"})
	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "control")
}

func TestEvidenceCommandStructure(t *testing.T) {
	t.Parallel()

	t.Run("evidence command exists on root", func(t *testing.T) {
		t.Parallel()
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "evidence" {
				found = true
				break
			}
		}
		assert.True(t, found, "root command should have 'evidence' subcommand")
	})

	t.Run("evidence has expected subcommands", func(t *testing.T) {
		t.Parallel()
		var names []string
		for _, cmd := range evidenceCmd.Commands() {
			names = append(names, cmd.Name())
		}
		assert.Contains(t, names, "list")
		assert.Contains(t, names, "view")
		assert.Contains(t, names, "generate")
		assert.Contains(t, names, "review")
		assert.Contains(t, names, "submit")
		assert.Contains(t, names, "evaluate")
		assert.Contains(t, names, "setup")
		assert.Contains(t, names, "migrate")
	})

	t.Run("evidence list flags", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, evidenceListCmd.Flags().Lookup("status"))
		assert.NotNil(t, evidenceListCmd.Flags().Lookup("framework"))
		assert.NotNil(t, evidenceListCmd.Flags().Lookup("priority"))
		assert.NotNil(t, evidenceListCmd.Flags().Lookup("overdue"))
		assert.NotNil(t, evidenceListCmd.Flags().Lookup("due-soon"))
		assert.NotNil(t, evidenceListCmd.Flags().Lookup("category"))
		assert.NotNil(t, evidenceListCmd.Flags().Lookup("aec-status"))
		assert.NotNil(t, evidenceListCmd.Flags().Lookup("collection-type"))
		assert.NotNil(t, evidenceListCmd.Flags().Lookup("sensitive"))
		assert.NotNil(t, evidenceListCmd.Flags().Lookup("complexity"))
	})

	t.Run("evidence generate flags", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, evidenceGenerateCmd.Flags().Lookup("all"))
		assert.NotNil(t, evidenceGenerateCmd.Flags().Lookup("tools"))
		assert.NotNil(t, evidenceGenerateCmd.Flags().Lookup("format"))
		assert.NotNil(t, evidenceGenerateCmd.Flags().Lookup("output-dir"))
		assert.NotNil(t, evidenceGenerateCmd.Flags().Lookup("window"))
		assert.NotNil(t, evidenceGenerateCmd.Flags().Lookup("context-only"))
		assert.NotNil(t, evidenceGenerateCmd.Flags().Lookup("with-tool-data"))
	})

	t.Run("evidence submit flags", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, evidenceSubmitCmd.Flags().Lookup("window"))
		assert.NotNil(t, evidenceSubmitCmd.Flags().Lookup("notes"))
		assert.NotNil(t, evidenceSubmitCmd.Flags().Lookup("skip-validation"))
		assert.NotNil(t, evidenceSubmitCmd.Flags().Lookup("dry-run"))
	})

	t.Run("evidence evaluate flags", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, evidenceEvaluateCmd.Flags().Lookup("window"))
		assert.NotNil(t, evidenceEvaluateCmd.Flags().Lookup("subfolder"))
		assert.NotNil(t, evidenceEvaluateCmd.Flags().Lookup("all"))
		assert.NotNil(t, evidenceEvaluateCmd.Flags().Lookup("output"))
		assert.NotNil(t, evidenceEvaluateCmd.Flags().Lookup("save-validation"))
		assert.NotNil(t, evidenceEvaluateCmd.Flags().Lookup("verbose"))
	})

	t.Run("evidence setup flags", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, evidenceSetupCmd.Flags().Lookup("collector-url"))
		assert.NotNil(t, evidenceSetupCmd.Flags().Lookup("dry-run"))
		assert.NotNil(t, evidenceSetupCmd.Flags().Lookup("force"))
	})
}

func TestEvidenceEvaluateValidation(t *testing.T) {
	// Cannot use t.Parallel() — runEvidenceEvaluate uses global config

	t.Run("missing task ref and no --all flag", func(t *testing.T) {
		cmd := &cobra.Command{
			Use:  "evaluate",
			Args: cobra.MaximumNArgs(1),
			RunE: runEvidenceEvaluate,
		}
		cmd.Flags().String("window", "", "window")
		cmd.Flags().String("subfolder", "", "subfolder")
		cmd.Flags().Bool("all", false, "all")
		cmd.Flags().StringP("output", "o", "", "output")
		cmd.Flags().Bool("save-validation", true, "save")
		cmd.Flags().Bool("verbose", false, "verbose")

		cmd.SetArgs([]string{})
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "task reference required")
	})

	t.Run("task ref without window", func(t *testing.T) {
		cmd := &cobra.Command{
			Use:  "evaluate",
			Args: cobra.MaximumNArgs(1),
			RunE: runEvidenceEvaluate,
		}
		cmd.Flags().String("window", "", "window")
		cmd.Flags().String("subfolder", "", "subfolder")
		cmd.Flags().Bool("all", false, "all")
		cmd.Flags().StringP("output", "o", "", "output")
		cmd.Flags().Bool("save-validation", true, "save")
		cmd.Flags().Bool("verbose", false, "verbose")

		cmd.SetArgs([]string{"ET-0001"})
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "--window flag required")
	})

	t.Run("task ref with --all flag conflicts", func(t *testing.T) {
		cmd := &cobra.Command{
			Use:  "evaluate",
			Args: cobra.MaximumNArgs(1),
			RunE: runEvidenceEvaluate,
		}
		cmd.Flags().String("window", "", "window")
		cmd.Flags().String("subfolder", "", "subfolder")
		cmd.Flags().Bool("all", false, "all")
		cmd.Flags().StringP("output", "o", "", "output")
		cmd.Flags().Bool("save-validation", true, "save")
		cmd.Flags().Bool("verbose", false, "verbose")

		cmd.SetArgs([]string{"ET-0001", "--all"})
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot specify task reference with --all")
	})

	t.Run("invalid subfolder", func(t *testing.T) {
		cmd := &cobra.Command{
			Use:  "evaluate",
			Args: cobra.MaximumNArgs(1),
			RunE: runEvidenceEvaluate,
		}
		cmd.Flags().String("window", "", "window")
		cmd.Flags().String("subfolder", "", "subfolder")
		cmd.Flags().Bool("all", false, "all")
		cmd.Flags().StringP("output", "o", "", "output")
		cmd.Flags().Bool("save-validation", true, "save")
		cmd.Flags().Bool("verbose", false, "verbose")

		cmd.SetArgs([]string{"ET-0001", "--window", "2025-Q4", "--subfolder", "invalid"})
		buf := new(bytes.Buffer)
		cmd.SetOut(buf)
		cmd.SetErr(buf)

		err := cmd.Execute()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid subfolder")
	})
}

func TestSyncCommandStructure(t *testing.T) {
	t.Parallel()

	t.Run("sync command exists on root", func(t *testing.T) {
		t.Parallel()
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == "sync" {
				found = true
				break
			}
		}
		assert.True(t, found, "root command should have 'sync' subcommand")
	})
}

func TestPolicyCommandStructure(t *testing.T) {
	t.Parallel()

	t.Run("policy has view and list subcommands", func(t *testing.T) {
		t.Parallel()
		var names []string
		for _, cmd := range policyCmd.Commands() {
			names = append(names, cmd.Name())
		}
		assert.Contains(t, names, "view")
		assert.Contains(t, names, "list")
	})

	t.Run("policy view flags", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, policyViewCmd.Flags().Lookup("output"))
		assert.NotNil(t, policyViewCmd.Flags().Lookup("summary"))
		assert.NotNil(t, policyViewCmd.Flags().Lookup("metadata-only"))
	})

	t.Run("policy list flags", func(t *testing.T) {
		t.Parallel()
		assert.NotNil(t, policyListCmd.Flags().Lookup("framework"))
		assert.NotNil(t, policyListCmd.Flags().Lookup("status"))
		assert.NotNil(t, policyListCmd.Flags().Lookup("details"))
	})
}

func TestRootCommandSubcommands(t *testing.T) {
	t.Parallel()

	expectedSubcommands := []string{
		"auth",
		"config",
		"control",
		"evidence",
		"policy",
		"status",
		"sync",
		"tool",
		"update",
		"version",
	}

	var names []string
	for _, cmd := range rootCmd.Commands() {
		names = append(names, cmd.Name())
	}

	for _, expected := range expectedSubcommands {
		assert.Contains(t, names, expected, "root command should have '%s' subcommand", expected)
	}
}

func TestAgentContextTopics(t *testing.T) {
	t.Parallel()

	// Verify topic map has expected keys
	expectedTopics := []string{
		"directory-structure",
		"evidence-workflow",
		"tool-capabilities",
		"status-commands",
		"submission-process",
		"bulk-operations",
	}

	for _, topic := range expectedTopics {
		t.Run(topic, func(t *testing.T) {
			t.Parallel()
			info, ok := agentContextTopics[topic]
			assert.True(t, ok, "topic %s should exist", topic)
			assert.NotEmpty(t, info.filename)
			assert.NotEmpty(t, info.description)
		})
	}
}

func TestListAgentContextTopics(t *testing.T) {
	t.Parallel()

	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)

	err := listAgentContextTopics(cmd)
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Available agent-context topics")
	assert.Contains(t, output, "directory-structure")
	assert.Contains(t, output, "evidence-workflow")
	assert.Contains(t, output, "bulk-operations")
	assert.Contains(t, output, "Usage:")
}

func TestCompleteAgentContextTopics(t *testing.T) {
	t.Parallel()

	cmd := &cobra.Command{}

	t.Run("all topics when no args", func(t *testing.T) {
		t.Parallel()
		completions, directive := completeAgentContextTopics(cmd, nil, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		assert.GreaterOrEqual(t, len(completions), 6)
	})

	t.Run("no completions when args already provided", func(t *testing.T) {
		t.Parallel()
		completions, directive := completeAgentContextTopics(cmd, []string{"evidence-workflow"}, "")
		assert.Equal(t, cobra.ShellCompDirectiveNoFileComp, directive)
		assert.Nil(t, completions)
	})
}

func TestDisplayTaskSummary(t *testing.T) {
	t.Parallel()

	t.Run("task with no windows", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		task := &models.EvidenceTaskState{TaskRef: "ET-0001"}
		displayTaskSummary(cmd, task, false)
		assert.Empty(t, buf.String())
	})

	t.Run("task with submitted window", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		now := time.Now()
		task := &models.EvidenceTaskState{
			TaskRef:  "ET-0001",
			TaskName: "Access Control Evidence",
			Windows: map[string]models.WindowState{
				"2025-Q4": {
					Window:      "2025-Q4",
					FileCount:   3,
					SubmittedAt: &now,
					NewestFile:  &now,
				},
			},
		}
		displayTaskSummary(cmd, task, true)
		output := buf.String()
		assert.Contains(t, output, "ET-0001")
		assert.Contains(t, output, "submitted")
		assert.Contains(t, output, "Access Control Evidence")
	})

	t.Run("task with generated window", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		now := time.Now()
		task := &models.EvidenceTaskState{
			TaskRef: "ET-0002",
			Windows: map[string]models.WindowState{
				"2025-Q3": {
					Window:      "2025-Q3",
					FileCount:   5,
					GeneratedAt: &now,
					NewestFile:  &now,
				},
			},
		}
		displayTaskSummary(cmd, task, false)
		output := buf.String()
		assert.Contains(t, output, "ET-0002")
		assert.Contains(t, output, "generated")
	})
}

func TestDisplayWindowDetail(t *testing.T) {
	t.Parallel()

	t.Run("window with generation meta", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		now := time.Now()
		window := models.WindowState{
			Window:            "2025-Q4",
			FileCount:         3,
			TotalBytes:        5120,
			HasGenerationMeta: true,
			GeneratedAt:       &now,
			GeneratedBy:       "grctool-cli",
			ToolsUsed:         []string{"terraform", "github"},
		}

		displayWindowDetail(cmd, window)
		output := buf.String()
		assert.Contains(t, output, "2025-Q4")
		assert.Contains(t, output, "5.0KB")
		assert.Contains(t, output, "grctool-cli")
		assert.Contains(t, output, "terraform")
	})

	t.Run("window with submission meta", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		now := time.Now()
		window := models.WindowState{
			Window:            "2025-Q3",
			FileCount:         2,
			TotalBytes:        1024,
			HasSubmissionMeta: true,
			SubmittedAt:       &now,
			SubmissionID:      "SUB-123",
			SubmissionStatus:  "accepted",
		}

		displayWindowDetail(cmd, window)
		output := buf.String()
		assert.Contains(t, output, "SUB-123")
		assert.Contains(t, output, "accepted")
	})

	t.Run("window with files", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		window := models.WindowState{
			Window:     "2025-Q4",
			FileCount:  1,
			TotalBytes: 2048,
			Files: []models.FileState{
				{
					Filename:  "evidence.json",
					SizeBytes: 2048,
					Checksum:  "abcdefgh12345678",
				},
			},
		}

		displayWindowDetail(cmd, window)
		output := buf.String()
		assert.Contains(t, output, "evidence.json")
		assert.Contains(t, output, "2.0KB")
		assert.Contains(t, output, "abcdefgh...")
	})
}

func TestDisplayNextSteps(t *testing.T) {
	t.Parallel()

	t.Run("no windows", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		task := &models.EvidenceTaskState{
			Windows: map[string]models.WindowState{},
		}

		displayNextSteps(cmd, "ET-0001", task)
		output := buf.String()
		assert.Contains(t, output, "grctool evidence generate ET-0001")
	})

	t.Run("accepted submission", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		task := &models.EvidenceTaskState{
			Windows: map[string]models.WindowState{
				"2025-Q4": {
					HasSubmissionMeta: true,
					SubmissionStatus:  "accepted",
				},
			},
		}

		displayNextSteps(cmd, "ET-0001", task)
		output := buf.String()
		assert.Contains(t, output, "accepted")
	})

	t.Run("generated evidence", func(t *testing.T) {
		t.Parallel()
		buf := new(bytes.Buffer)
		cmd := &cobra.Command{}
		cmd.SetOut(buf)

		now := time.Now()
		task := &models.EvidenceTaskState{
			Windows: map[string]models.WindowState{
				"2025-Q4": {
					HasGenerationMeta: true,
					GeneratedAt:       &now,
				},
			},
		}

		displayNextSteps(cmd, "ET-0001", task)
		output := buf.String()
		assert.Contains(t, output, "evidence validate")
	})
}

func TestDisplayEvaluationSummary(t *testing.T) {
	t.Parallel()

	t.Run("empty results", func(t *testing.T) {
		// Just verify no panic
		displayEvaluationSummary(nil, nil)
	})

	t.Run("with results", func(t *testing.T) {
		results := []*models.EvaluationResult{
			{OverallStatus: models.EvaluationPass, OverallScore: 90.0},
			{OverallStatus: models.EvaluationWarning, OverallScore: 75.0},
			{OverallStatus: models.EvaluationFail, OverallScore: 40.0},
		}
		// Just verify no panic
		displayEvaluationSummary(results, []string{"ET-0001/Q4: error"})
	})
}

// ============================================================
// Validation rule constants
// ============================================================

func TestValidationRuleConstants(t *testing.T) {
	t.Parallel()

	t.Run("TaskRefRule", func(t *testing.T) {
		t.Parallel()
		assert.False(t, TaskRefRule.Required)
		assert.Equal(t, "string", TaskRefRule.Type)
		assert.NotEmpty(t, TaskRefRule.Pattern)
	})

	t.Run("PathRule", func(t *testing.T) {
		t.Parallel()
		assert.True(t, PathRule.Required)
		assert.Equal(t, "path", PathRule.Type)
		assert.True(t, PathRule.PathSafety)
	})

	t.Run("OptionalPathRule", func(t *testing.T) {
		t.Parallel()
		assert.False(t, OptionalPathRule.Required)
		assert.True(t, OptionalPathRule.PathSafety)
	})

	t.Run("StringRule", func(t *testing.T) {
		t.Parallel()
		assert.True(t, StringRule.Required)
		assert.Equal(t, "string", StringRule.Type)
		assert.Equal(t, 1, StringRule.MinLength)
	})

	t.Run("OptionalStringRule", func(t *testing.T) {
		t.Parallel()
		assert.False(t, OptionalStringRule.Required)
	})

	t.Run("IntRule", func(t *testing.T) {
		t.Parallel()
		assert.True(t, IntRule.Required)
		assert.Equal(t, "int", IntRule.Type)
	})

	t.Run("BoolRule", func(t *testing.T) {
		t.Parallel()
		assert.False(t, BoolRule.Required)
		assert.Equal(t, "bool", BoolRule.Type)
	})
}

// ============================================================
// Help output tests for commands without dedicated test files
// ============================================================

func TestToolManagementCommandHelp(t *testing.T) {
	t.Parallel()

	// Find tool-management command
	var tmCmd *cobra.Command
	for _, cmd := range toolCmd.Commands() {
		if cmd.Name() == "tool-management" {
			tmCmd = cmd
			break
		}
	}

	if tmCmd == nil {
		t.Skip("tool-management not registered as subcommand of tool")
	}

	buf := new(bytes.Buffer)
	tmCmd.SetOut(buf)
	tmCmd.SetErr(buf)
	tmCmd.SetArgs([]string{"--help"})
	_ = tmCmd.Execute()
	assert.Contains(t, buf.String(), "tool-management")
}

func TestUpdateCommandHelp(t *testing.T) {
	// Cannot use t.Parallel() - uses global rootCmd
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"update", "--help"})
	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "update")
}

func TestAgentContextCommandHelp(t *testing.T) {
	// Cannot use t.Parallel() - uses global rootCmd
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"agent-context", "--help"})
	err := rootCmd.Execute()
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "agent-context")
}

// ============================================================
// Helpers
// ============================================================

func timePtr(t time.Time) *time.Time {
	return &t
}
