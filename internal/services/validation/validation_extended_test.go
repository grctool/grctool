// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// helper to create a storage instance with sample data
func setupValidationStorage(t *testing.T) (*storage.Storage, string) {
	t.Helper()
	tmpDir := t.TempDir()
	cfg := config.StorageConfig{
		DataDir: tmpDir,
		Paths:   config.StoragePaths{}.WithDefaults(),
	}
	stor, err := storage.NewStorage(cfg)
	require.NoError(t, err)
	return stor, tmpDir
}

func writeJSON(t *testing.T, path string, v interface{}) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0755))
	data, err := json.Marshal(v)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(path, data, 0644))
}

// ---- DataValidatorImpl tests ----

func TestDataValidatorImpl_ValidateAll_Empty(t *testing.T) {
	t.Parallel()
	stor, tmpDir := setupValidationStorage(t)
	v := NewDataValidator(stor, tmpDir)

	report, err := v.ValidateAll(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, report)
	assert.Equal(t, "pass", report.Summary.Status)
	assert.Equal(t, 0, report.Summary.CriticalIssues)
	assert.Equal(t, 0, report.Summary.Warnings)
}

func TestDataValidatorImpl_ValidatePolicies_WithAndWithoutContent(t *testing.T) {
	t.Parallel()
	stor, tmpDir := setupValidationStorage(t)

	paths := config.StoragePaths{}.WithDefaults()
	policyDir := filepath.Join(tmpDir, paths.PoliciesJSON)
	require.NoError(t, os.MkdirAll(policyDir, 0755))

	// Policy with substantial content (10+ lines)
	longDesc := strings.Repeat("Line of content about security policy.\n", 15)
	writeJSON(t, filepath.Join(policyDir, "pol-001.json"), map[string]interface{}{
		"id":          "1",
		"name":        "Access Control Policy",
		"description": longDesc,
	})

	// Policy with no description
	writeJSON(t, filepath.Join(policyDir, "pol-002.json"), map[string]interface{}{
		"id":          "2",
		"name":        "Empty Policy",
		"description": "",
	})

	v := NewDataValidator(stor, tmpDir)
	pv, issues, err := v.ValidatePolicies(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, pv.WithContent)
	assert.Equal(t, 1, pv.MissingContent)
	assert.Equal(t, 1, pv.WithSubstantialContent)
	assert.Equal(t, 1, pv.MissingDescription)
	assert.Equal(t, 50.0, pv.ContentCompleteness)
	assert.True(t, pv.AverageContentLength > 0)
	assert.Len(t, issues, 1) // one warning for empty description
	assert.Equal(t, "warning", issues[0].Type)
	assert.Equal(t, "policy", issues[0].Category)
}

func TestDataValidatorImpl_ValidateControls_WithAndWithoutDescription(t *testing.T) {
	t.Parallel()
	stor, tmpDir := setupValidationStorage(t)

	paths := config.StoragePaths{}.WithDefaults()
	controlDir := filepath.Join(tmpDir, paths.ControlsJSON)
	require.NoError(t, os.MkdirAll(controlDir, 0755))

	// Control with description and policy links
	writeJSON(t, filepath.Join(controlDir, "ctrl-001.json"), map[string]interface{}{
		"id":          "CC6.1",
		"name":        "Access Control",
		"description": "Controls logical access.",
		"associations": map[string]interface{}{
			"policies": 2,
		},
	})

	// Control with no description and no policy links
	writeJSON(t, filepath.Join(controlDir, "ctrl-002.json"), map[string]interface{}{
		"id":          "CC6.2",
		"name":        "User Registration",
		"description": "",
	})

	v := NewDataValidator(stor, tmpDir)
	cv, issues, err := v.ValidateControls(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, cv.WithDescription)
	assert.Equal(t, 1, cv.MissingDescription)
	assert.Equal(t, 1, cv.WithPolicyLinks)
	assert.Equal(t, 1, cv.MissingPolicyLinks)
	assert.Equal(t, 50.0, cv.LinkageCompleteness)
	// 2 issues: one for missing description, one for missing policy links
	assert.Len(t, issues, 2)
}

func TestDataValidatorImpl_ValidateEvidenceTasks_WithGuidance(t *testing.T) {
	t.Parallel()
	stor, tmpDir := setupValidationStorage(t)

	paths := config.StoragePaths{}.WithDefaults()
	etDir := filepath.Join(tmpDir, paths.EvidenceTasksJSON)
	require.NoError(t, os.MkdirAll(etDir, 0755))

	writeJSON(t, filepath.Join(etDir, "et-001.json"), map[string]interface{}{
		"id":          "1",
		"name":        "GitHub Access",
		"description": "Verify repo access",
		"guidance":    "Check team permissions",
		"controls":    []string{"CC6.1"},
	})

	writeJSON(t, filepath.Join(etDir, "et-002.json"), map[string]interface{}{
		"id":          "2",
		"name":        "Empty Task",
		"description": "",
		"guidance":    "",
	})

	v := NewDataValidator(stor, tmpDir)
	ev, issues, err := v.ValidateEvidenceTasks(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 1, ev.WithDescription)
	assert.Equal(t, 1, ev.MissingDescription)
	assert.Equal(t, 1, ev.WithGuidance)
	assert.Equal(t, 1, ev.MissingGuidance)
	assert.Equal(t, 2, ev.WithControlLinks) // simplified impl always increments
	assert.Equal(t, 50.0, ev.ContentCompleteness)
	assert.Len(t, issues, 1) // one for missing description
}

func TestDataValidatorImpl_ValidateRelationships_Returns_Empty(t *testing.T) {
	t.Parallel()
	stor, tmpDir := setupValidationStorage(t)
	v := NewDataValidator(stor, tmpDir)

	rv, issues, err := v.ValidateRelationships(context.Background())
	require.NoError(t, err)
	assert.NotNil(t, rv)
	assert.Empty(t, issues)
	assert.Equal(t, 0, rv.BrokenControlReferences)
}

func TestDataValidatorImpl_GenerateSummary_StatusVariants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		issues         []ValidationIssue
		expectedStatus string
	}{
		{
			name:           "pass when no issues",
			issues:         nil,
			expectedStatus: "pass",
		},
		{
			name: "warning when only warnings",
			issues: []ValidationIssue{
				{Type: "warning", Category: "policy", Description: "test warning"},
			},
			expectedStatus: "warning",
		},
		{
			name: "fail when critical issues",
			issues: []ValidationIssue{
				{Type: "critical", Category: "control", Description: "test critical"},
				{Type: "warning", Category: "policy", Description: "test warning"},
			},
			expectedStatus: "fail",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			stor, tmpDir := setupValidationStorage(t)
			v := NewDataValidator(stor, tmpDir).(*DataValidatorImpl)

			report := &ValidationReport{
				Issues: tt.issues,
				Policies: PolicyValidation{
					WithContent:    5,
					MissingContent: 0,
				},
				Controls: ControlValidation{
					WithDescription:    3,
					MissingDescription: 1,
				},
				Evidence: EvidenceValidation{
					WithDescription:    4,
					MissingDescription: 2,
				},
			}

			summary := v.generateSummary(report)
			assert.Equal(t, tt.expectedStatus, summary.Status)
		})
	}
}

// ---- ServiceImpl tests ----

func TestServiceImpl_FormatReport_JSON(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	report := &ValidationReport{
		Summary: ValidationSummary{
			Status:       "pass",
			OverallScore: 95.0,
		},
		Issues: []ValidationIssue{},
	}

	result, err := svc.FormatReport(report, "json")
	require.NoError(t, err)
	assert.Contains(t, result, `"status": "pass"`)
	assert.Contains(t, result, `"overall_score": 95`)

	// Verify it's valid JSON
	var parsed ValidationReport
	require.NoError(t, json.Unmarshal([]byte(result), &parsed))
	assert.Equal(t, "pass", parsed.Summary.Status)
}

func TestServiceImpl_FormatReport_Text(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	report := &ValidationReport{
		Summary: ValidationSummary{
			Status:       "warning",
			OverallScore: 80.0,
			Warnings:     2,
		},
		Policies: PolicyValidation{
			ContentCompleteness: 90.0,
			WithContent:         9,
			MissingContent:      1,
		},
		Controls: ControlValidation{
			LinkageCompleteness: 75.0,
			WithPolicyLinks:     6,
			MissingPolicyLinks:  2,
		},
		Evidence: EvidenceValidation{
			ContentCompleteness: 85.0,
			WithGuidance:        7,
			MissingGuidance:     3,
		},
		Issues: []ValidationIssue{
			{Type: "warning", Category: "control", Description: "Missing link"},
		},
	}

	result, err := svc.FormatReport(report, "text")
	require.NoError(t, err)
	assert.Contains(t, result, "=== DATA VALIDATION REPORT ===")
	assert.Contains(t, result, "Overall Status: WARNING")
	assert.Contains(t, result, "Overall Score: 80.0%")
	assert.Contains(t, result, "VALIDATION ISSUES:")
	assert.Contains(t, result, "[WARNING] control: Missing link")
}

func TestServiceImpl_FormatReport_Markdown(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	report := &ValidationReport{
		Summary: ValidationSummary{
			Status:       "pass",
			OverallScore: 100.0,
		},
		Issues: []ValidationIssue{
			{Type: "info", Category: "policy", Description: "All good"},
		},
	}

	result, err := svc.FormatReport(report, "markdown")
	require.NoError(t, err)
	assert.Contains(t, result, "# Data Validation Report")
	assert.Contains(t, result, "## Summary")
	assert.Contains(t, result, "## Validation Issues")
}

func TestServiceImpl_FormatReport_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	_, err := svc.FormatReport(&ValidationReport{}, "xml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}

func TestServiceImpl_FormatReport_EmptyFormat_DefaultsToText(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	result, err := svc.FormatReport(&ValidationReport{Issues: []ValidationIssue{}}, "")
	require.NoError(t, err)
	assert.Contains(t, result, "=== DATA VALIDATION REPORT ===")
}

func TestServiceImpl_SaveReport(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}
	tmpDir := t.TempDir()

	report := &ValidationReport{
		Summary: ValidationSummary{Status: "pass", OverallScore: 100.0},
		Issues:  []ValidationIssue{},
	}

	outPath := filepath.Join(tmpDir, "report.json")
	err := svc.SaveReport(report, outPath, "json")
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"status": "pass"`)
}

func TestServiceImpl_SaveReport_InvalidPath(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	err := svc.SaveReport(&ValidationReport{Issues: []ValidationIssue{}}, "/nonexistent/dir/report.json", "json")
	require.Error(t, err)
}

func TestServiceImpl_SaveReport_UnsupportedFormat(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}
	tmpDir := t.TempDir()

	err := svc.SaveReport(&ValidationReport{}, filepath.Join(tmpDir, "report.xml"), "xml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported format")
}
