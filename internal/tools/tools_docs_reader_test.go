// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// helpers for docs reader tests
// ---------------------------------------------------------------------------

func newDocsReaderForTest(t *testing.T, dataDir string) *DocsReaderTool {
	t.Helper()
	cfg := &config.Config{
		Storage: config.StorageConfig{
			DataDir: dataDir,
		},
	}
	log := testhelpers.NewStubLogger()
	tool := NewDocsReaderTool(cfg, log)
	require.NotNil(t, tool)
	return tool.(*DocsReaderTool)
}

func writeTestDoc(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
	return path
}

// ---------------------------------------------------------------------------
// Name / Description / GetClaudeToolDefinition
// ---------------------------------------------------------------------------

func TestDocsReaderTool_Metadata(t *testing.T) {
	t.Parallel()
	drt := newDocsReaderForTest(t, t.TempDir())

	assert.Equal(t, "docs-reader", drt.Name())
	assert.NotEmpty(t, drt.Description())

	def := drt.GetClaudeToolDefinition()
	assert.Equal(t, "docs-reader", def.Name)
	assert.NotNil(t, def.InputSchema)
}

// ---------------------------------------------------------------------------
// prepareQueryTerms
// ---------------------------------------------------------------------------

func TestDocsReaderTool_PrepareQueryTerms(t *testing.T) {
	t.Parallel()
	drt := newDocsReaderForTest(t, t.TempDir())

	tests := map[string]struct {
		query string
		want  []string
	}{
		"simple two-word query": {
			query: "access control",
			want:  []string{"access", "control"},
		},
		"strips punctuation": {
			query: "data-protection, policy!",
			want:  []string{"dataprotection", "policy"},
		},
		"ignores short terms": {
			query: "a is on access",
			want:  []string{"is", "on", "access"},
		},
		"empty query": {
			query: "",
			want:  nil,
		},
		"uppercase normalized": {
			query: "SOC2 Compliance",
			want:  []string{"soc2", "compliance"},
		},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := drt.prepareQueryTerms(tc.query)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// calculateLineRelevance
// ---------------------------------------------------------------------------

func TestDocsReaderTool_CalculateLineRelevance(t *testing.T) {
	t.Parallel()
	drt := newDocsReaderForTest(t, t.TempDir())

	t.Run("no query terms gives zero", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0.0, drt.calculateLineRelevance("anything", nil))
	})

	t.Run("no match gives zero", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0.0, drt.calculateLineRelevance("nothing here", []string{"access", "control"}))
	})

	t.Run("partial match gives positive score", func(t *testing.T) {
		t.Parallel()
		score := drt.calculateLineRelevance("access is granted", []string{"access", "control"})
		assert.Greater(t, score, 0.0)
		assert.LessOrEqual(t, score, 1.0)
	})

	t.Run("full match scores higher than partial", func(t *testing.T) {
		t.Parallel()
		partial := drt.calculateLineRelevance("access is granted", []string{"access", "control"})
		full := drt.calculateLineRelevance("access control policy", []string{"access", "control"})
		assert.Greater(t, full, partial)
	})

	t.Run("exact phrase gets bonus", func(t *testing.T) {
		t.Parallel()
		// Use more terms so scores don't both saturate at 1.0
		noPhrase := drt.calculateLineRelevance("control of access granted", []string{"access", "control", "policy", "review"})
		withPhrase := drt.calculateLineRelevance("access control policy review", []string{"access", "control", "policy", "review"})
		assert.GreaterOrEqual(t, withPhrase, noPhrase)
	})

	t.Run("score capped at 1.0", func(t *testing.T) {
		t.Parallel()
		score := drt.calculateLineRelevance(
			"access access access control control control",
			[]string{"access", "control"},
		)
		assert.LessOrEqual(t, score, 1.0)
	})
}

// ---------------------------------------------------------------------------
// calculateSectionRelevance
// ---------------------------------------------------------------------------

func TestDocsReaderTool_CalculateSectionRelevance(t *testing.T) {
	t.Parallel()
	drt := newDocsReaderForTest(t, t.TempDir())

	t.Run("empty content", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0.0, drt.calculateSectionRelevance(nil, []string{"access"}))
	})

	t.Run("empty query terms", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0.0, drt.calculateSectionRelevance([]string{"some content"}, nil))
	})

	t.Run("no matching lines", func(t *testing.T) {
		t.Parallel()
		content := []string{"nothing relevant", "no matches at all"}
		assert.Equal(t, 0.0, drt.calculateSectionRelevance(content, []string{"access"}))
	})

	t.Run("matching lines give positive score", func(t *testing.T) {
		t.Parallel()
		content := []string{"access control is important", "access is managed", "other stuff"}
		score := drt.calculateSectionRelevance(content, []string{"access"})
		assert.Greater(t, score, 0.0)
		assert.LessOrEqual(t, score, 1.0)
	})
}

// ---------------------------------------------------------------------------
// calculateDocumentRelevance
// ---------------------------------------------------------------------------

func TestDocsReaderTool_CalculateDocumentRelevance(t *testing.T) {
	t.Parallel()
	drt := newDocsReaderForTest(t, t.TempDir())

	t.Run("empty document", func(t *testing.T) {
		t.Parallel()
		result := &DocSearchResult{TotalLines: 0}
		assert.Equal(t, 0.0, drt.calculateDocumentRelevance(result, []string{"access"}))
	})

	t.Run("no query terms", func(t *testing.T) {
		t.Parallel()
		result := &DocSearchResult{TotalLines: 10}
		assert.Equal(t, 0.0, drt.calculateDocumentRelevance(result, nil))
	})

	t.Run("document with matches", func(t *testing.T) {
		t.Parallel()
		result := &DocSearchResult{
			FileName:           "access_control.md",
			TotalLines:         20,
			MatchingLinesCount: 5,
			MatchingLines: []LineMatch{
				{Relevance: 0.8},
				{Relevance: 0.6},
			},
		}
		score := drt.calculateDocumentRelevance(result, []string{"access", "control"})
		assert.Greater(t, score, 0.0)
		assert.LessOrEqual(t, score, 1.0)
	})
}

// ---------------------------------------------------------------------------
// calculateOverallRelevance
// ---------------------------------------------------------------------------

func TestDocsReaderTool_CalculateOverallRelevance(t *testing.T) {
	t.Parallel()
	drt := newDocsReaderForTest(t, t.TempDir())

	t.Run("no results", func(t *testing.T) {
		t.Parallel()
		assert.Equal(t, 0.0, drt.calculateOverallRelevance(nil))
	})

	t.Run("single result", func(t *testing.T) {
		t.Parallel()
		results := []DocSearchResult{{Relevance: 0.5}}
		score := drt.calculateOverallRelevance(results)
		assert.Equal(t, 0.5, score)
	})

	t.Run("multiple results get bonus", func(t *testing.T) {
		t.Parallel()
		results := []DocSearchResult{
			{Relevance: 0.5},
			{Relevance: 0.5},
		}
		score := drt.calculateOverallRelevance(results)
		assert.Greater(t, score, 0.5, "2+ results should get a bonus")
	})

	t.Run("5+ results get higher bonus", func(t *testing.T) {
		t.Parallel()
		results := make([]DocSearchResult, 5)
		for i := range results {
			results[i].Relevance = 0.5
		}
		score := drt.calculateOverallRelevance(results)
		assert.Greater(t, score, 0.55, "5+ results should get larger bonus")
	})

	t.Run("score capped at 1.0", func(t *testing.T) {
		t.Parallel()
		results := make([]DocSearchResult, 10)
		for i := range results {
			results[i].Relevance = 1.0
		}
		score := drt.calculateOverallRelevance(results)
		assert.LessOrEqual(t, score, 1.0)
	})
}

// ---------------------------------------------------------------------------
// generateReport
// ---------------------------------------------------------------------------

func TestDocsReaderTool_GenerateReport(t *testing.T) {
	t.Parallel()
	drt := newDocsReaderForTest(t, t.TempDir())

	t.Run("empty results", func(t *testing.T) {
		t.Parallel()
		report := drt.generateReport(nil, "access control", "*.md", "/docs")
		assert.Contains(t, report, "Documentation Search Results")
		assert.Contains(t, report, "access control")
		assert.Contains(t, report, "Results Found**: 0")
		assert.Contains(t, report, "No documents found")
	})

	t.Run("with results", func(t *testing.T) {
		t.Parallel()
		results := []DocSearchResult{
			{
				FileName:           "policy.md",
				FilePath:           "/docs/policy.md",
				FileType:           ".md",
				Relevance:          0.75,
				TotalLines:         100,
				MatchingLinesCount: 10,
				MatchingLines: []LineMatch{
					{LineNumber: 5, Content: "access control policy", Relevance: 0.9},
				},
				Sections: []DocumentSection{
					{Title: "Overview", Level: 2, StartLine: 1, EndLine: 10, Relevance: 0.5},
				},
			},
		}
		report := drt.generateReport(results, "access control", "*.md", "/docs")
		assert.Contains(t, report, "policy.md")
		assert.Contains(t, report, "0.750")
		assert.Contains(t, report, "Top Matching Lines")
		assert.Contains(t, report, "Relevant Sections")
	})
}

// ---------------------------------------------------------------------------
// findMatchingFiles
// ---------------------------------------------------------------------------

func TestDocsReaderTool_FindMatchingFiles(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	drt := newDocsReaderForTest(t, dir)

	// Create test files
	writeTestDoc(t, dir, "docs/readme.md", "# Readme")
	writeTestDoc(t, dir, "docs/policy.md", "# Policy")
	writeTestDoc(t, dir, "docs/data.json", `{"key": "value"}`)

	t.Run("glob *.md matches markdown only", func(t *testing.T) {
		t.Parallel()
		files, err := drt.findMatchingFiles(filepath.Join(dir, "docs"), "*.md")
		require.NoError(t, err)
		assert.Len(t, files, 2)
	})

	t.Run("specific file", func(t *testing.T) {
		t.Parallel()
		files, err := drt.findMatchingFiles(filepath.Join(dir, "docs"), "data.json")
		require.NoError(t, err)
		assert.Len(t, files, 1)
	})

	t.Run("no matches", func(t *testing.T) {
		t.Parallel()
		files, err := drt.findMatchingFiles(filepath.Join(dir, "docs"), "*.csv")
		require.NoError(t, err)
		assert.Empty(t, files)
	})
}

// ---------------------------------------------------------------------------
// Execute integration test with temp docs
// ---------------------------------------------------------------------------

func TestDocsReaderTool_Execute(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	docsDir := filepath.Join(dir, "docs")

	writeTestDoc(t, docsDir, "access_control.md", `# Access Control Policy

## Overview
This policy defines access control requirements for all systems.

## Requirements
- All access must be authenticated
- Multi-factor authentication is required for production access
- Access reviews conducted quarterly
`)

	writeTestDoc(t, docsDir, "data_protection.md", `# Data Protection Policy

## Overview
This policy covers data protection and encryption standards.

## Requirements
- Data at rest must be encrypted
- Data in transit must use TLS 1.2+
`)

	drt := newDocsReaderForTest(t, dir)

	t.Run("search for access control", func(t *testing.T) {
		t.Parallel()
		result, source, err := drt.Execute(context.Background(), map[string]interface{}{
			"query":    "access control authentication",
			"pattern":  "*.md",
			"docs_path": "docs/",
		})
		require.NoError(t, err)
		assert.NotEmpty(t, result)
		assert.Contains(t, result, "access_control.md")
		require.NotNil(t, source)
		assert.Equal(t, "docs-reader", source.Type)
	})

	t.Run("missing query returns error", func(t *testing.T) {
		t.Parallel()
		_, _, err := drt.Execute(context.Background(), map[string]interface{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "query parameter is required")
	})

	t.Run("no matching docs", func(t *testing.T) {
		t.Parallel()
		result, source, err := drt.Execute(context.Background(), map[string]interface{}{
			"query":         "xyzzyzzy nonexistent term",
			"pattern":       "*.md",
			"docs_path":     "docs/",
			"min_relevance": 0.9,
		})
		require.NoError(t, err)
		assert.Contains(t, result, "No documents found")
		require.NotNil(t, source)
	})

	t.Run("custom parameters", func(t *testing.T) {
		t.Parallel()
		result, _, err := drt.Execute(context.Background(), map[string]interface{}{
			"query":            "encryption data",
			"pattern":          "*.md",
			"docs_path":        "docs/",
			"min_relevance":    0.01,
			"max_results":      1,
			"extract_sections": true,
		})
		require.NoError(t, err)
		assert.NotEmpty(t, result)
	})
}

// ---------------------------------------------------------------------------
// analyzeDocument with section extraction
// ---------------------------------------------------------------------------

func TestDocsReaderTool_AnalyzeDocument(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	drt := newDocsReaderForTest(t, dir)

	docPath := writeTestDoc(t, dir, "test.md", `# Main Title

## Access Control
Users must authenticate before accessing any system.
Multi-factor authentication is required.

## Data Protection
All data must be encrypted at rest.
`)

	result, err := drt.analyzeDocument(docPath, "access authentication", true)
	require.NoError(t, err)
	require.NotNil(t, result)

	assert.Equal(t, "test.md", result.FileName)
	assert.Equal(t, ".md", result.FileType)
	assert.Greater(t, result.TotalLines, 0)
	assert.Greater(t, result.MatchingLinesCount, 0)
	assert.Greater(t, result.Relevance, 0.0)

	// Should have extracted sections
	assert.NotEmpty(t, result.Sections, "should extract markdown sections")
}
