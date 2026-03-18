// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// EvidenceValidatorTool — helper functions and pure logic
// We test the methods that don't require full storage initialization.
// ---------------------------------------------------------------------------

// newMinimalEvidenceValidator creates a validator with minimal config for
// testing pure utility methods. We bypass NewEvidenceValidatorTool which
// requires a real storage backend.
func newMinimalEvidenceValidator(t *testing.T, dataDir string) *EvidenceValidatorTool {
	t.Helper()
	return &EvidenceValidatorTool{
		config: newTestConfig(dataDir),
		logger: testhelpers.NewStubLogger(),
	}
}

// ---------------------------------------------------------------------------
// detectFormat
// ---------------------------------------------------------------------------

func TestEvidenceValidator_DetectFormat(t *testing.T) {
	t.Parallel()

	evt := newMinimalEvidenceValidator(t, t.TempDir())

	tests := map[string]struct {
		filePath string
		content  string
		want     string
	}{
		"markdown by extension":    {filePath: "report.md", content: "", want: "markdown"},
		"markdown alt extension":   {filePath: "report.markdown", content: "", want: "markdown"},
		"json by extension":        {filePath: "data.json", content: "", want: "json"},
		"csv by extension":         {filePath: "data.csv", content: "", want: "csv"},
		"txt by extension":         {filePath: "notes.txt", content: "", want: "text"},
		"json by content object":   {filePath: "unknown.dat", content: `{"key": "value"}`, want: "json"},
		"json by content array":    {filePath: "unknown.dat", content: `[1, 2, 3]`, want: "json"},
		"markdown by content":      {filePath: "unknown.dat", content: "# Title\n\n**bold** text", want: "markdown"},
		"csv by content":           {filePath: "unknown.dat", content: "name,age\nAlice,30\nBob,25", want: "csv"},
		"text fallback":            {filePath: "unknown.dat", content: "just plain text", want: "text"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := evt.detectFormat(tc.filePath, tc.content)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// isPathSafe
// ---------------------------------------------------------------------------

func TestEvidenceValidator_IsPathSafe(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	evt := newMinimalEvidenceValidator(t, dataDir)

	t.Run("path traversal blocked", func(t *testing.T) {
		t.Parallel()
		assert.False(t, evt.isPathSafe("../../etc/passwd"))
	})

	t.Run("path outside data dir blocked", func(t *testing.T) {
		t.Parallel()
		assert.False(t, evt.isPathSafe("/tmp/outside"))
	})

	t.Run("path inside data dir allowed", func(t *testing.T) {
		t.Parallel()
		assert.True(t, evt.isPathSafe(dataDir+"/evidence/report.md"))
	})
}

// ---------------------------------------------------------------------------
// extractTaskIDFromContent
// ---------------------------------------------------------------------------

func TestEvidenceValidator_ExtractTaskIDFromContent(t *testing.T) {
	t.Parallel()

	evt := newMinimalEvidenceValidator(t, t.TempDir())

	t.Run("finds ET reference", func(t *testing.T) {
		t.Parallel()
		content := "# Evidence for ET47\nSome evidence content here"
		taskID := evt.extractTaskIDFromContent(content)
		assert.Equal(t, 327991+47, taskID)
	})

	t.Run("no ET reference returns 0", func(t *testing.T) {
		t.Parallel()
		content := "Just some random content\nNo task references"
		taskID := evt.extractTaskIDFromContent(content)
		assert.Equal(t, 0, taskID)
	})

	t.Run("ET in later lines still found if within first 10", func(t *testing.T) {
		t.Parallel()
		content := "line 1\nline 2\nline 3\nline 4\nTask ET5 evidence\n"
		taskID := evt.extractTaskIDFromContent(content)
		assert.Equal(t, 327991+5, taskID)
	})
}

// ---------------------------------------------------------------------------
// validateFormat
// ---------------------------------------------------------------------------

func TestEvidenceValidator_ValidateFormat(t *testing.T) {
	t.Parallel()

	evt := newMinimalEvidenceValidator(t, t.TempDir())

	t.Run("valid json", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateFormat(`{"key": "value"}`, "json", result)
		// No error issues expected
		for _, issue := range result.Issues {
			assert.NotEqual(t, "error", issue.Type, "valid JSON should not produce errors")
		}
	})

	t.Run("invalid json", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateFormat(`{broken json`, "json", result)
		require.NotEmpty(t, result.Issues)
		assert.Equal(t, "error", result.Issues[0].Type)
		assert.Equal(t, "format", result.Issues[0].Category)
	})

	t.Run("markdown without formatting", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateFormat("just plain text no markdown", "markdown", result)
		require.NotEmpty(t, result.Issues)
		assert.Equal(t, "warning", result.Issues[0].Type)
	})

	t.Run("markdown with formatting OK", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateFormat("# Title\n**bold** content", "markdown", result)
		assert.Empty(t, result.Issues)
	})

	t.Run("csv too short", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateFormat("just one line", "csv", result)
		require.NotEmpty(t, result.Issues)
		assert.Equal(t, "warning", result.Issues[0].Type)
	})
}

// ---------------------------------------------------------------------------
// validateCompleteness
// ---------------------------------------------------------------------------

func TestEvidenceValidator_ValidateCompleteness(t *testing.T) {
	t.Parallel()

	evt := newMinimalEvidenceValidator(t, t.TempDir())

	t.Run("short content gets warning", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateCompleteness("short", result)
		hasShortWarning := false
		for _, issue := range result.Issues {
			if issue.Category == "completeness" && issue.Type == "warning" {
				hasShortWarning = true
			}
		}
		assert.True(t, hasShortWarning)
	})

	t.Run("long content without headers has no short content warning", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		longContent := ""
		for i := 0; i < 200; i++ {
			longContent += "x"
		}
		evt.validateCompleteness(longContent, result)
		for _, issue := range result.Issues {
			if issue.Description != "" {
				assert.NotContains(t, issue.Description, "very short")
			}
		}
	})
}

// ---------------------------------------------------------------------------
// validateContent — placeholder and sensitive data detection
// ---------------------------------------------------------------------------

func TestEvidenceValidator_ValidateContent(t *testing.T) {
	t.Parallel()

	evt := newMinimalEvidenceValidator(t, t.TempDir())

	t.Run("detects placeholder text", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateContent("This is a TODO placeholder FIXME later", result)
		// Should find TODO and FIXME
		placeholderIssues := 0
		for _, issue := range result.Issues {
			if issue.Category == "content" && issue.Type == "warning" {
				placeholderIssues++
			}
		}
		assert.GreaterOrEqual(t, placeholderIssues, 2)
	})

	t.Run("detects potential sensitive data", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateContent("password: hunter2\napi_key: abc123xyz", result)
		sensitiveIssues := 0
		for _, issue := range result.Issues {
			if issue.Category == "content" && issue.Type == "error" {
				sensitiveIssues++
			}
		}
		assert.GreaterOrEqual(t, sensitiveIssues, 1)
	})

	t.Run("clean content has no issues", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateContent("This is clean evidence content about access controls.", result)
		assert.Empty(t, result.Issues)
	})
}

// ---------------------------------------------------------------------------
// validateSources
// ---------------------------------------------------------------------------

func TestEvidenceValidator_ValidateSources(t *testing.T) {
	t.Parallel()

	evt := newMinimalEvidenceValidator(t, t.TempDir())

	t.Run("content with source reference OK", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateSources("Source: GitHub API\nData collected from repository", result)
		// Should not have the "lacks sources" warning
		for _, issue := range result.Issues {
			assert.NotContains(t, issue.Description, "lacks clear source")
		}
	})

	t.Run("content without source reference gets warning", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateSources("Some evidence data without attribution", result)
		require.NotEmpty(t, result.Issues)
		assert.Contains(t, result.Issues[0].Description, "lacks clear source")
	})
}

// ---------------------------------------------------------------------------
// validateCompliance
// ---------------------------------------------------------------------------

func TestEvidenceValidator_ValidateCompliance(t *testing.T) {
	t.Parallel()

	evt := newMinimalEvidenceValidator(t, t.TempDir())

	t.Run("content with framework reference", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateCompliance(nil, "This evidence addresses SOC2 CC6.1 requirements", result)
		assert.Empty(t, result.Issues)
	})

	t.Run("content without framework reference", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateCompliance(nil, "Generic evidence without framework context", result)
		require.NotEmpty(t, result.Issues)
		assert.Equal(t, "info", result.Issues[0].Type)
		assert.Equal(t, "compliance", result.Issues[0].Category)
	})
}

// ---------------------------------------------------------------------------
// validateMetadata
// ---------------------------------------------------------------------------

func TestEvidenceValidator_ValidateMetadata(t *testing.T) {
	t.Parallel()

	evt := newMinimalEvidenceValidator(t, t.TempDir())

	t.Run("sufficient metadata", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateMetadata("Generated on Date: 2025-01-15\nVersion: 1.0\nAuthor: grctool", result)
		assert.Empty(t, result.Issues)
	})

	t.Run("insufficient metadata", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.validateMetadata("Just some text with no metadata", result)
		require.NotEmpty(t, result.Issues)
		assert.Equal(t, "info", result.Issues[0].Type)
	})
}

// ---------------------------------------------------------------------------
// calculateScores
// ---------------------------------------------------------------------------

func TestEvidenceValidator_CalculateScores(t *testing.T) {
	t.Parallel()

	evt := newMinimalEvidenceValidator(t, t.TempDir())

	t.Run("no issues = perfect scores", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{Issues: []ValidationIssue{}}
		evt.calculateScores(result)
		assert.Equal(t, 100.0, result.CompletenessScore)
		assert.Equal(t, 100.0, result.QualityScore)
		assert.Equal(t, "passed", result.OverallStatus)
	})

	t.Run("warnings only = passed_with_warnings", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{
			Issues: []ValidationIssue{
				{Type: "warning", Category: "completeness"},
			},
		}
		evt.calculateScores(result)
		assert.Equal(t, "passed_with_warnings", result.OverallStatus)
		assert.Less(t, result.CompletenessScore, 100.0)
	})

	t.Run("many warnings = needs_improvement", func(t *testing.T) {
		t.Parallel()
		issues := make([]ValidationIssue, 4)
		for i := range issues {
			issues[i] = ValidationIssue{Type: "warning", Category: "completeness"}
		}
		result := &EvidenceValidationResult{Issues: issues}
		evt.calculateScores(result)
		assert.Equal(t, "needs_improvement", result.OverallStatus)
	})

	t.Run("errors = failed", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{
			Issues: []ValidationIssue{
				{Type: "error", Category: "format"},
			},
		}
		evt.calculateScores(result)
		assert.Equal(t, "failed", result.OverallStatus)
		assert.Less(t, result.QualityScore, 60.0)
	})

	t.Run("scores do not go below zero", func(t *testing.T) {
		t.Parallel()
		issues := make([]ValidationIssue, 20)
		for i := range issues {
			issues[i] = ValidationIssue{Type: "error", Category: "format"}
		}
		result := &EvidenceValidationResult{Issues: issues}
		evt.calculateScores(result)
		assert.GreaterOrEqual(t, result.CompletenessScore, 0.0)
		assert.GreaterOrEqual(t, result.QualityScore, 0.0)
	})
}

// ---------------------------------------------------------------------------
// generateSummary
// ---------------------------------------------------------------------------

func TestEvidenceValidator_GenerateSummary(t *testing.T) {
	t.Parallel()

	evt := newMinimalEvidenceValidator(t, t.TempDir())

	t.Run("summary with no issues", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{
			OverallStatus:     "passed",
			CompletenessScore: 100.0,
			QualityScore:      100.0,
			Issues:            []ValidationIssue{},
		}
		evt.generateSummary(result)
		assert.Contains(t, result.Summary, "passed")
		assert.Contains(t, result.Summary, "100.0")
		assert.Contains(t, result.Summary, "No issues found")
	})

	t.Run("summary with mixed issues", func(t *testing.T) {
		t.Parallel()
		result := &EvidenceValidationResult{
			OverallStatus:     "failed",
			CompletenessScore: 50.0,
			QualityScore:      30.0,
			Issues: []ValidationIssue{
				{Type: "error", Category: "format"},
				{Type: "warning", Category: "completeness"},
				{Type: "info", Category: "compliance"},
			},
			Recommendations: []string{"Fix format", "Add more detail"},
		}
		evt.generateSummary(result)
		assert.Contains(t, result.Summary, "failed")
		assert.Contains(t, result.Summary, "3 issues")
		assert.Contains(t, result.Summary, "1 errors")
		assert.Contains(t, result.Summary, "1 warnings")
		assert.Contains(t, result.Summary, "1 info")
		assert.Contains(t, result.Summary, "2 recommendations")
	})
}

// ---------------------------------------------------------------------------
// validateEvidenceFile end-to-end (creates actual temp file)
// ---------------------------------------------------------------------------

func TestEvidenceValidator_ValidateEvidenceFile(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	evt := newMinimalEvidenceValidator(t, dataDir)

	t.Run("valid markdown with good content", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		evtLocal := newMinimalEvidenceValidator(t, dir)

		content := `# Evidence Report for SOC2 Compliance

## Description
This evidence document describes access control procedures.

## Evidence
Source: GitHub API
Based on: Repository permissions analysis

Generated on Date: 2025-01-15
Version: 1.0
Author: grctool
`
		filePath := filepath.Join(dir, "evidence.md")
		require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))

		result, err := evtLocal.validateEvidenceFile(nil, filePath, "comprehensive")
		require.NoError(t, err)
		require.NotNil(t, result)
		assert.NotEmpty(t, result.Summary)
		assert.Greater(t, result.CompletenessScore, 0.0)
	})

	t.Run("invalid path blocked", func(t *testing.T) {
		t.Parallel()
		_, err := evt.validateEvidenceFile(nil, "../../etc/passwd", "basic")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsafe path")
	})

	t.Run("nonexistent file", func(t *testing.T) {
		t.Parallel()
		_, err := evt.validateEvidenceFile(nil, filepath.Join(dataDir, "nonexistent.md"), "basic")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("JSON file with errors", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		evtLocal := newMinimalEvidenceValidator(t, dir)

		filePath := filepath.Join(dir, "bad.json")
		require.NoError(t, os.WriteFile(filePath, []byte(`{broken json TODO FIXME}`), 0o644))

		result, err := evtLocal.validateEvidenceFile(nil, filePath, "standard")
		require.NoError(t, err)
		require.NotNil(t, result)
		// Should have format error
		hasFormatError := false
		for _, issue := range result.Issues {
			if issue.Category == "format" && issue.Type == "error" {
				hasFormatError = true
			}
		}
		assert.True(t, hasFormatError, "should detect JSON format error")
	})
}

// ---------------------------------------------------------------------------
// parseIntSafe
// ---------------------------------------------------------------------------

func TestParseIntSafe(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 42, parseIntSafe("42"))
	assert.Equal(t, 0, parseIntSafe("not-a-number"))
	assert.Equal(t, 0, parseIntSafe(""))
	assert.Equal(t, -5, parseIntSafe("-5"))
}
