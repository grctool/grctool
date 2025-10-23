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

package formatters

import (
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interpolation"
)

func TestPolicyFormatter_ToMarkdown(t *testing.T) {
	formatter := NewPolicyFormatter()

	tests := []struct {
		name     string
		policy   *domain.PolicyDetails
		validate func(t *testing.T, markdown string)
	}{
		{
			name: "Complete policy with all fields",
			policy: &domain.PolicyDetails{
				ID:          "POL-001",
				Name:        "Information Security Policy",
				Description: "This policy establishes the framework for information security.",
				Framework:   "SOC2",
				Status:      "active",
				CreatedAt:   time.Date(2023, 1, 15, 10, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2024, 3, 10, 14, 30, 0, 0, time.UTC),
				Summary:     "Comprehensive information security policy for the organization.",
				Content:     "# Information Security Policy\n\n## Purpose\n\nThis policy ensures the confidentiality, integrity, and availability of information assets.\n\n## Scope\n\nThis policy applies to all employees, contractors, and third parties.",
				Version:     "2.1",
				CurrentVersion: &domain.PolicyVersion{
					ID:        "VER-001",
					Version:   "2.1",
					Status:    "published",
					CreatedAt: time.Date(2024, 3, 10, 14, 30, 0, 0, time.UTC),
					CreatedBy: "John Doe",
				},
				ControlCount:     5,
				ProcedureCount:   3,
				EvidenceCount:    8,
				RiskCount:        2,
				ViewCount:        42,
				LastViewedAt:     timePtr(time.Date(2024, 3, 15, 9, 0, 0, 0, time.UTC)),
				DownloadCount:    7,
				LastDownloadedAt: timePtr(time.Date(2024, 3, 14, 16, 30, 0, 0, time.UTC)),
				ReferenceCount:   12,
				LastReferencedAt: timePtr(time.Date(2024, 3, 16, 11, 15, 0, 0, time.UTC)),
				Assignees: []domain.Person{
					{
						ID:         "USR-001",
						Name:       "Jane Smith",
						Email:      "jane.smith@example.com",
						Role:       "Policy Owner",
						AssignedAt: timePtr(time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)),
					},
				},
				Reviewers: []domain.Person{
					{
						ID:    "USR-002",
						Name:  "Bob Johnson",
						Email: "bob.johnson@example.com",
						Role:  "Security Manager",
					},
				},
				Tags: []domain.Tag{
					{ID: "TAG-001", Name: "security", Color: "#ff0000"},
					{ID: "TAG-002", Name: "compliance", Color: "#00ff00"},
				},
			},
			validate: func(t *testing.T, md string) {
				// Check header section
				if !strings.Contains(md, "# Policy POL-001") {
					t.Error("Missing policy header")
				}
				if !strings.Contains(md, "Policy ID: POL-001") {
					t.Error("Missing policy ID in metadata box")
				}
				if !strings.Contains(md, "Version: 2.1") {
					t.Error("Missing version in metadata box")
				}
				if !strings.Contains(md, "Framework: SOC2") {
					t.Error("Missing framework")
				}
				if !strings.Contains(md, "Status: active") {
					t.Error("Missing status")
				}

				// Check content sections
				if !strings.Contains(md, "## Information Security Policy") {
					t.Error("Missing policy title")
				}
				if !strings.Contains(md, "### Summary") {
					t.Error("Missing summary section")
				}
				if !strings.Contains(md, "### Policy Document") {
					t.Error("Missing policy document section")
				}
				if !strings.Contains(md, "confidentiality, integrity, and availability") {
					t.Error("Missing policy content")
				}

				// Check metadata sections
				if !strings.Contains(md, "## Policy Metadata") {
					t.Error("Missing metadata section")
				}
				if !strings.Contains(md, "### Basic Information") {
					t.Error("Missing basic information section")
				}
				if !strings.Contains(md, "### Version Information") {
					t.Error("Missing version information section")
				}
				if !strings.Contains(md, "### Associated Items") {
					t.Error("Missing associated items section")
				}
				if !strings.Contains(md, "### Usage Statistics") {
					t.Error("Missing usage statistics section")
				}

				// Check specific counts
				if !strings.Contains(md, "| **Controls** | 5 |") {
					t.Error("Missing or incorrect control count")
				}
				if !strings.Contains(md, "| **Evidence Tasks** | 8 |") {
					t.Error("Missing or incorrect evidence count")
				}
				if !strings.Contains(md, "| **Views** | 42 |") {
					t.Error("Missing or incorrect view count")
				}

				// Check assignees and reviewers
				if !strings.Contains(md, "### Assignees") {
					t.Error("Missing assignees section")
				}
				if !strings.Contains(md, "Jane Smith") {
					t.Error("Missing assignee name")
				}
				if !strings.Contains(md, "jane.smith@example.com") {
					t.Error("Missing assignee email")
				}
				if !strings.Contains(md, "### Reviewers") {
					t.Error("Missing reviewers section")
				}
				if !strings.Contains(md, "Bob Johnson") {
					t.Error("Missing reviewer name")
				}

				// Check tags
				if !strings.Contains(md, "### Tags") {
					t.Error("Missing tags section")
				}
				if !strings.Contains(md, "`security`") {
					t.Error("Missing security tag")
				}
				if !strings.Contains(md, "`compliance`") {
					t.Error("Missing compliance tag")
				}

				// Check footer
				if !strings.Contains(md, "*Generated on") {
					t.Error("Missing generation timestamp")
				}
			},
		},
		{
			name: "Minimal policy with basic fields only",
			policy: &domain.PolicyDetails{
				ID:          "POL-002",
				Name:        "Basic Policy",
				Description: "A simple policy description.",
				Framework:   "ISO27001",
				Status:      "draft",
				CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
			},
			validate: func(t *testing.T, md string) {
				// Check header
				if !strings.Contains(md, "# Policy POL-002") {
					t.Error("Missing policy header")
				}
				if !strings.Contains(md, "## Basic Policy") {
					t.Error("Missing policy name")
				}

				// Should use description as policy document when no content
				if !strings.Contains(md, "### Policy Document") {
					t.Error("Missing policy document section")
				}
				if !strings.Contains(md, "A simple policy description.") {
					t.Error("Missing policy description content")
				}

				// Should not have sections for missing data
				if strings.Contains(md, "### Summary") {
					t.Error("Should not have summary section when no summary provided")
				}
				if strings.Contains(md, "### Version Information") {
					t.Error("Should not have version section when no version info")
				}
				if strings.Contains(md, "### Associated Items") {
					t.Error("Should not have associated items when counts are zero")
				}
				if strings.Contains(md, "### Usage Statistics") {
					t.Error("Should not have usage stats when counts are zero")
				}
				if strings.Contains(md, "### Assignees") {
					t.Error("Should not have assignees section when empty")
				}
				if strings.Contains(md, "### Tags") {
					t.Error("Should not have tags section when empty")
				}
			},
		},
		{
			name: "Policy with deprecation notice",
			policy: &domain.PolicyDetails{
				ID:               "POL-003",
				Name:             "Deprecated Policy",
				Description:      "This policy is being phased out.",
				Framework:        "SOC2",
				Status:           "deprecated",
				CreatedAt:        time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:        time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				DeprecationNotes: "This policy has been superseded by POL-004. Please refer to the new policy for current requirements.",
			},
			validate: func(t *testing.T, md string) {
				if !strings.Contains(md, "### ⚠️ Deprecation Notice") {
					t.Error("Missing deprecation notice section")
				}
				if !strings.Contains(md, "superseded by POL-004") {
					t.Error("Missing deprecation notice content")
				}
			},
		},
		{
			name: "Policy with content that needs cleaning",
			policy: &domain.PolicyDetails{
				ID:          "POL-004",
				Name:        "Policy with Messy Content",
				Description: "A policy with content that needs formatting.",
				Framework:   "SOC2",
				Status:      "active",
				CreatedAt:   time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
				Content:     "   # Title with extra spaces   \n\n\n\nParagraph with multiple blank lines.\n\n\n\nAnother paragraph.   \n   \n   \nFinal paragraph.",
			},
			validate: func(t *testing.T, md string) {
				// Check that content is cleaned up
				content := extractPolicyDocumentSection(md)

				// Should not have excessive blank lines
				if strings.Contains(content, "\n\n\n") {
					t.Error("Content should not have excessive blank lines")
				}

				// Should preserve paragraph breaks
				if !strings.Contains(content, "blank lines.\n\nAnother paragraph.") {
					t.Error("Should preserve single paragraph breaks")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			markdown := formatter.ToMarkdown(tt.policy)

			// Basic validation
			if markdown == "" {
				t.Fatal("Markdown output is empty")
			}

			// Run specific validation
			tt.validate(t, markdown)
		})
	}
}

func TestPolicyFormatter_ToSummaryMarkdown(t *testing.T) {
	formatter := NewPolicyFormatter()

	policy := &domain.PolicyDetails{
		ID:            "POL-001",
		Name:          "Test Policy",
		Description:   "This is a long description that should be truncated when used in summary format. " + strings.Repeat("Extra text ", 20),
		Framework:     "SOC2",
		Status:        "active",
		UpdatedAt:     time.Date(2024, 3, 10, 14, 30, 0, 0, time.UTC),
		Summary:       "A brief summary of the policy.",
		ControlCount:  5,
		EvidenceCount: 3,
	}

	summary := formatter.ToSummaryMarkdown(policy)

	// Check basic structure
	if !strings.Contains(summary, "## Test Policy") {
		t.Error("Missing policy title in summary")
	}

	if !strings.Contains(summary, "**ID:** POL-001") {
		t.Error("Missing policy ID in summary")
	}

	if !strings.Contains(summary, "**Framework:** SOC2") {
		t.Error("Missing framework in summary")
	}

	if !strings.Contains(summary, "**Status:** active") {
		t.Error("Missing status in summary")
	}

	// Should use summary, not description
	if !strings.Contains(summary, "A brief summary of the policy.") {
		t.Error("Missing policy summary text")
	}

	// Should show associated items
	if !strings.Contains(summary, "**Associated:**") {
		t.Error("Missing associated items section")
	}

	if !strings.Contains(summary, "5 controls") {
		t.Error("Missing control count")
	}

	if !strings.Contains(summary, "3 evidence tasks") {
		t.Error("Missing evidence task count")
	}

	// Should show last updated
	if !strings.Contains(summary, "*Last updated: 2024-03-10*") {
		t.Error("Missing last updated date")
	}
}

func TestPolicyFormatter_CleanPolicyContent(t *testing.T) {
	formatter := NewPolicyFormatter()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Remove excessive whitespace",
			input:    "   Line with spaces   \n  Another line  \n\n\n\nLine after many blanks",
			expected: "Line with spaces\nAnother line\n\nLine after many blanks",
		},
		{
			name:     "Preserve single paragraph breaks",
			input:    "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.",
			expected: "First paragraph.\n\nSecond paragraph.\n\nThird paragraph.",
		},
		{
			name:     "Remove trailing whitespace",
			input:    "Line one   \nLine two\t\nLine three\n   \n",
			expected: "Line one\nLine two\nLine three",
		},
		{
			name:     "Handle empty input",
			input:    "",
			expected: "",
		},
		{
			name:     "Handle only whitespace",
			input:    "   \n\n\t\t\n   ",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatter.cleanPolicyContent(tt.input)
			if result != tt.expected {
				t.Errorf("cleanPolicyContent() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestNewPolicyFormatter(t *testing.T) {
	formatter := NewPolicyFormatter()
	if formatter == nil {
		t.Error("NewPolicyFormatter() returned nil")
	}
}

// Helper functions

func timePtr(t time.Time) *time.Time {
	return &t
}

func extractPolicyDocumentSection(markdown string) string {
	lines := strings.Split(markdown, "\n")
	var inPolicyDoc bool
	var policyContent []string

	for _, line := range lines {
		if strings.Contains(line, "### Policy Document") {
			inPolicyDoc = true
			continue
		}
		if inPolicyDoc && strings.HasPrefix(line, "---") {
			break
		}
		if inPolicyDoc {
			policyContent = append(policyContent, line)
		}
	}

	return strings.Join(policyContent, "\n")
}

func TestPolicyFormatterWithInterpolation(t *testing.T) {
	// Create interpolator with test variables
	config := interpolation.InterpolatorConfig{
		Variables: map[string]string{
			"organization.name": "Acme Corporation",
			"Organization Name": "Acme Corporation",
			"support.email":     "support@acme.com",
			"security.email":    "security@acme.com",
		},
		Enabled:           true,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}

	interpolator := interpolation.NewStandardInterpolator(config)
	formatter := NewPolicyFormatterWithInterpolation(interpolator)

	// Create policy with template variables
	policy := &domain.Policy{
		ID:          "POL-001",
		Name:        "{{organization.name}} Security Policy",
		Description: "Security policy for [Organization Name] employees",
		Framework:   "SOC2",
		Status:      "active",
		CreatedAt:   time.Date(2023, 1, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt:   time.Date(2024, 3, 10, 14, 30, 0, 0, time.UTC),
		Summary:     "This policy applies to all {{organization.name}} staff members.",
		Content:     "<h1>{{organization.name}} Workstation Policy</h1><p>All [Organization Name] employees must follow this policy. Contact {{support.email}} for questions.</p>",
	}

	markdown := formatter.ToMarkdown(policy)

	// Check that variables were interpolated in the title
	if !strings.Contains(markdown, "## Acme Corporation Security Policy") {
		t.Error("Policy name should have interpolated organization.name")
	}

	// Check that variables were interpolated in summary
	if !strings.Contains(markdown, "This policy applies to all Acme Corporation staff members.") {
		t.Error("Summary should have interpolated organization.name")
	}

	// Check that variables were interpolated in content
	if !strings.Contains(markdown, "# Acme Corporation Workstation Policy") {
		t.Error("Content should have interpolated organization.name in title")
	}

	if !strings.Contains(markdown, "All Acme Corporation employees must follow") {
		t.Error("Content should have interpolated [Organization Name]")
	}

	if !strings.Contains(markdown, "Contact support@acme.com for questions") {
		t.Error("Content should have interpolated support.email")
	}

	// Check that original template variables are not present
	if strings.Contains(markdown, "{{organization.name}}") {
		t.Error("Template variables should be replaced, not left as-is")
	}

	if strings.Contains(markdown, "[Organization Name]") {
		t.Error("Bracket variables should be replaced, not left as-is")
	}
}

func TestPolicyFormatterWithMissingVariables(t *testing.T) {
	// Create interpolator with limited variables
	config := interpolation.InterpolatorConfig{
		Variables: map[string]string{
			"organization.name": "Test Corp",
		},
		Enabled:           true,
		OnMissingVariable: interpolation.MissingVariableIgnore, // Should leave missing vars unchanged
	}

	interpolator := interpolation.NewStandardInterpolator(config)
	formatter := NewPolicyFormatterWithInterpolation(interpolator)

	// Create policy with both known and unknown variables
	policy := &domain.Policy{
		ID:        "POL-001",
		Name:      "{{organization.name}} Security Policy",
		Content:   "Contact {{organization.name}} at {{unknown.email}} for support.",
		Status:    "active",
		CreatedAt: time.Date(2023, 1, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 3, 10, 14, 30, 0, 0, time.UTC),
	}

	markdown := formatter.ToMarkdown(policy)

	// Check that known variables were interpolated
	if !strings.Contains(markdown, "Test Corp Security Policy") {
		t.Error("Known variables should be interpolated")
	}

	if !strings.Contains(markdown, "Contact Test Corp at") {
		t.Error("Known variables in content should be interpolated")
	}

	// Check that unknown variables were left unchanged
	if !strings.Contains(markdown, "{{unknown.email}}") {
		t.Error("Unknown variables should be left unchanged when using MissingVariableIgnore")
	}
}

func TestPolicyFormatterInterpolationDisabled(t *testing.T) {
	// Create disabled interpolator
	config := interpolation.InterpolatorConfig{
		Variables: map[string]string{
			"organization.name": "Should Not Appear",
		},
		Enabled:           false, // Disabled
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}

	interpolator := interpolation.NewStandardInterpolator(config)
	formatter := NewPolicyFormatterWithInterpolation(interpolator)

	// Create policy with template variables
	policy := &domain.Policy{
		ID:        "POL-001",
		Name:      "{{organization.name}} Security Policy",
		Content:   "Contact {{organization.name}} for support.",
		Status:    "active",
		CreatedAt: time.Date(2023, 1, 15, 10, 0, 0, 0, time.UTC),
		UpdatedAt: time.Date(2024, 3, 10, 14, 30, 0, 0, time.UTC),
	}

	markdown := formatter.ToMarkdown(policy)

	// Check that variables were NOT interpolated when disabled
	if !strings.Contains(markdown, "{{organization.name}} Security Policy") {
		t.Error("Variables should NOT be interpolated when interpolation is disabled")
	}

	if strings.Contains(markdown, "Should Not Appear") {
		t.Error("Disabled interpolation should not replace any variables")
	}
}
