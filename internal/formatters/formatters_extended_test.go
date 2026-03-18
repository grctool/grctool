// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package formatters

import (
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interpolation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---- ControlFormatter tests (all 0% coverage) ----

func newTestControlFormatter() *ControlFormatter {
	return NewControlFormatter()
}

func newTestControlFormatterWithInterpolation() *ControlFormatter {
	config := interpolation.InterpolatorConfig{
		Variables: map[string]string{
			"organization.name": "TestCorp",
		},
		Enabled:           true,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	return NewControlFormatterWithInterpolation(interpolation.NewStandardInterpolator(config))
}

func sampleControl() *domain.Control {
	now := time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC)
	tested := time.Date(2025, 2, 1, 0, 0, 0, 0, time.UTC)
	viewed := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)
	return &domain.Control{
		ID:                "CC6.1",
		ReferenceID:       "CC6.1",
		Name:              "Logical Access Controls",
		Description:       "Controls that restrict logical access to systems.",
		Category:          "Access Control",
		Framework:         "SOC2",
		Status:            "implemented",
		Risk:              "Unauthorized access to systems",
		RiskLevel:         "High",
		IsAutoImplemented: true,
		ImplementedDate:   &now,
		TestedDate:        &tested,
		Codes:             "SOC2-CC6.1",
		DeprecationNotes:  "Will be updated in next framework revision",
		MasterContent: &domain.ControlMasterContent{
			Help:     "Implement role-based access controls.",
			Guidance: "Review access permissions quarterly.",
		},
		Associations: &domain.ControlAssociations{
			Policies:   3,
			Procedures: 2,
			Evidence:   5,
			Risks:      1,
		},
		OrgEvidenceMetrics: &domain.ControlEvidenceMetrics{
			TotalCount:    10,
			CompleteCount: 8,
			OverdueCount:  2,
		},
		RecommendedEvidenceCount: 5,
		OpenIncidentCount:        1,
		Assignees: []domain.Person{
			{Name: "Alice Smith", Email: "alice@example.com"},
		},
		AuditProjects: []domain.AuditProject{
			{Name: "SOC2 2025", Status: "in_progress", Description: "Annual audit"},
		},
		FrameworkCodes: []domain.FrameworkCode{
			{Framework: "SOC2", Code: "CC6.1", Name: "Logical Access"},
		},
		Tags: []domain.Tag{
			{Name: "security", Color: "#ff0000"},
		},
		ViewCount:      42,
		LastViewedAt:   &viewed,
		DownloadCount:  5,
		ReferenceCount: 10,
	}
}

func TestControlFormatter_NewControlFormatter(t *testing.T) {
	t.Parallel()
	cf := newTestControlFormatter()
	assert.NotNil(t, cf)
	assert.NotNil(t, cf.baseFormatter)
}

func TestControlFormatter_NewControlFormatterWithInterpolation(t *testing.T) {
	t.Parallel()
	cf := newTestControlFormatterWithInterpolation()
	assert.NotNil(t, cf)
}

func TestControlFormatter_ToMarkdown(t *testing.T) {
	t.Parallel()
	cf := newTestControlFormatter()
	ctrl := sampleControl()

	md := cf.ToMarkdown(ctrl)

	assert.Contains(t, md, "# Control CC6.1")
	assert.Contains(t, md, "Control ID: CC6.1")
	assert.Contains(t, md, "Framework: SOC2")
	assert.Contains(t, md, "Status: implemented")
	assert.Contains(t, md, "Implemented: 2025-01-15")
	assert.Contains(t, md, "Last Tested: 2025-02-01")
	assert.Contains(t, md, "## Logical Access Controls")
	assert.Contains(t, md, "### Description")
	assert.Contains(t, md, "Controls that restrict logical access")
	assert.Contains(t, md, "### Implementation Help")
	assert.Contains(t, md, "### Implementation Guidance")
	assert.Contains(t, md, "### Risk Information")
	assert.Contains(t, md, "**Risk Level**: High")
	assert.Contains(t, md, "Auto-Implemented")
	assert.Contains(t, md, "**Policies** | 3")
	assert.Contains(t, md, "**Evidence Tasks** | 5")
	assert.Contains(t, md, "**Total Evidence** | 10")
	assert.Contains(t, md, "**Recommended Evidence** | 5")
	assert.Contains(t, md, "**Open Incidents** | 1")
	assert.Contains(t, md, "Alice Smith")
	assert.Contains(t, md, "SOC2 2025")
	assert.Contains(t, md, "SOC2-CC6.1")
	assert.Contains(t, md, "Deprecation Notice")
	assert.Contains(t, md, "security")
}

func TestControlFormatter_ToMarkdown_Minimal(t *testing.T) {
	t.Parallel()
	cf := newTestControlFormatter()
	ctrl := &domain.Control{
		ID:     "CC9.9",
		Name:   "",
		Status: "not_applicable",
	}

	md := cf.ToMarkdown(ctrl)
	assert.Contains(t, md, "# Control CC9.9")
	assert.NotContains(t, md, "### Description")
	assert.NotContains(t, md, "### Implementation Help")
}

func TestControlFormatter_ToSummaryMarkdown(t *testing.T) {
	t.Parallel()
	cf := newTestControlFormatter()
	ctrl := sampleControl()

	md := cf.ToSummaryMarkdown(ctrl)

	assert.Contains(t, md, "## Logical Access Controls")
	assert.Contains(t, md, "**ID:** CC6.1")
	assert.Contains(t, md, "**Category:** Access Control")
	assert.Contains(t, md, "**Associated:** 3 policies, 5 evidence tasks")
	assert.Contains(t, md, "*Implemented: 2025-01-15*")
	assert.Contains(t, md, "*Last tested: 2025-02-01*")
}

func TestControlFormatter_ToSummaryMarkdown_LongDescription(t *testing.T) {
	t.Parallel()
	cf := newTestControlFormatter()
	ctrl := &domain.Control{
		ID:          "CC1.1",
		Name:        "Test",
		Category:    "General",
		Status:      "active",
		Description: strings.Repeat("A", 250),
	}

	md := cf.ToSummaryMarkdown(ctrl)
	// Description should be truncated to 200 chars
	assert.Contains(t, md, "...")
}

func TestControlFormatter_ToDocumentMarkdown(t *testing.T) {
	t.Parallel()
	cf := newTestControlFormatter()
	ctrl := sampleControl()

	md := cf.ToDocumentMarkdown(ctrl)

	assert.Contains(t, md, "# CC6.1 - Logical Access Controls")
	assert.Contains(t, md, "## Description")
	assert.Contains(t, md, "## Implementation Help")
	assert.Contains(t, md, "## Implementation Guidance")
	assert.Contains(t, md, "## Risk Information")
	assert.Contains(t, md, "## Document Information")
	assert.Contains(t, md, "Document Reference")
	assert.Contains(t, md, "| **Associated Policies** | 3 |")
}

func TestControlFormatter_ToDocumentMarkdown_NoReferenceID(t *testing.T) {
	t.Parallel()
	cf := newTestControlFormatter()
	ctrl := &domain.Control{
		ID:   "123",
		Name: "Access Provisioning",
	}

	md := cf.ToDocumentMarkdown(ctrl)
	// Should fall back to generateReference
	assert.Contains(t, md, "# C1 - Access Provisioning")
}

func TestControlFormatter_generateReference(t *testing.T) {
	t.Parallel()
	cf := newTestControlFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"exact match", "access provisioning", "C1"},
		{"exact match password", "password policy", "C4"},
		{"partial match", "multi-factor authentication system", "C5"},
		{"network match", "network access control rules", "C7"},
		{"encryption match", "data encryption at rest", "C16"},
		{"no match generates initials", "custom unknown thing", "CCU"},
		{"empty string", "", "CTRL"},
		{"single word", "testing", "CT"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := cf.generateReference(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestControlFormatter_GetDocumentFilename(t *testing.T) {
	t.Parallel()
	cf := newTestControlFormatter()
	ctrl := sampleControl()

	filename := cf.GetDocumentFilename(ctrl)
	assert.NotEmpty(t, filename)
	assert.True(t, strings.HasSuffix(filename, ".md"))
}

// ---- BaseFormatter convertSingleTable tests ----

func TestBaseFormatter_ConvertSingleTable(t *testing.T) {
	t.Parallel()
	config := interpolation.InterpolatorConfig{
		Variables: make(map[string]string),
		Enabled:   false,
	}
	bf := NewBaseFormatter(interpolation.NewStandardInterpolator(config))

	tableHTML := `<tr><th>Header 1</th><th>Header 2</th></tr><tr><td>Cell 1</td><td>Cell 2</td></tr>`
	result := bf.convertSingleTable(tableHTML)

	assert.Contains(t, result, "| Header 1 |")
	assert.Contains(t, result, "|-------|")
	assert.Contains(t, result, "| Cell 1 |")
}

func TestBaseFormatter_ConvertHTMLTables(t *testing.T) {
	t.Parallel()
	config := interpolation.InterpolatorConfig{
		Variables: make(map[string]string),
		Enabled:   false,
	}
	bf := NewBaseFormatter(interpolation.NewStandardInterpolator(config))

	input := `<table><tr><th>A</th></tr><tr><td>B</td></tr></table>`
	result := bf.convertHTMLTables(input)

	assert.Contains(t, result, "| A |")
	assert.Contains(t, result, "| B |")
	assert.NotContains(t, result, "<table>")
}

// ---- FormatPersonList partial coverage (empty email case) ----

func TestBaseFormatter_FormatPersonList_EmptyEmails(t *testing.T) {
	t.Parallel()
	config := interpolation.InterpolatorConfig{
		Variables: make(map[string]string),
		Enabled:   false,
	}
	bf := NewBaseFormatter(interpolation.NewStandardInterpolator(config))

	people := []PersonInfo{
		{Name: "Bob", Email: ""},
		{Name: "Alice", Email: "alice@example.com", Role: "Admin"},
	}

	result := bf.FormatPersonList(people, "Team Members")
	assert.Contains(t, result, "Bob")
	assert.Contains(t, result, "Alice")
	assert.Contains(t, result, "Admin")
}

// ---- FormatUsageStatistics partial coverage ----

func TestBaseFormatter_FormatUsageStatistics_AllZeros(t *testing.T) {
	t.Parallel()
	icfg := interpolation.InterpolatorConfig{
		Variables: make(map[string]string),
		Enabled:   false,
	}
	bf := NewBaseFormatter(interpolation.NewStandardInterpolator(icfg))

	stats := UsageStats{}
	result := bf.FormatUsageStatistics(stats)
	// With all zeros, it should not contain usage stats section
	assert.NotContains(t, result, "Usage Statistics")
}

func TestBaseFormatter_FormatUsageStatistics_WithDates(t *testing.T) {
	t.Parallel()
	icfg := interpolation.InterpolatorConfig{
		Variables: make(map[string]string),
		Enabled:   false,
	}
	bf := NewBaseFormatter(interpolation.NewStandardInterpolator(icfg))

	now := time.Now()
	stats := UsageStats{
		ViewCount:        10,
		LastViewedAt:     now,
		DownloadCount:    3,
		LastDownloadedAt: now,
		ReferenceCount:   7,
		LastReferencedAt: now,
	}
	result := bf.FormatUsageStatistics(stats)
	assert.Contains(t, result, "Usage Statistics")
	assert.Contains(t, result, "10")
}

// ---- FormatTagList partial coverage (empty tags) ----

func TestBaseFormatter_FormatTagList_EmptyColor(t *testing.T) {
	t.Parallel()
	icfg := interpolation.InterpolatorConfig{
		Variables: make(map[string]string),
		Enabled:   false,
	}
	bf := NewBaseFormatter(interpolation.NewStandardInterpolator(icfg))

	tags := []TagInfo{
		{Name: "compliance", Color: ""},
		{Name: "security", Color: "#00ff00"},
	}
	result := bf.FormatTagList(tags)
	assert.Contains(t, result, "compliance")
	assert.Contains(t, result, "security")
}

// ---- InterpolateText partial coverage ----

func TestBaseFormatter_InterpolateText_Enabled(t *testing.T) {
	t.Parallel()
	icfg := interpolation.InterpolatorConfig{
		Variables: map[string]string{
			"organization.name": "MyCorp",
		},
		Enabled:           true,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	bf := NewBaseFormatter(interpolation.NewStandardInterpolator(icfg))

	result := bf.InterpolateText("Welcome to {{organization.name}}")
	require.Contains(t, result, "MyCorp")
}

// ---- EvidenceTaskFormatter SetRegistry test ----

func TestEvidenceTaskFormatter_SetRegistry_Nil(t *testing.T) {
	t.Parallel()
	etf := NewEvidenceTaskFormatter()
	etf.SetRegistry(nil)
	// Should not panic, registry is just nil
}
