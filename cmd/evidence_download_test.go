// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package cmd

import (
	"strings"
	"testing"

	"github.com/grctool/grctool/internal/tugboat/models"
)

func TestBuildSubmissionMarkdown(t *testing.T) {
	tests := []struct {
		name     string
		att      *models.EvidenceAttachment
		contains []string
		notContains []string
	}{
		{
			name: "Full submission with all fields",
			att: &models.EvidenceAttachment{
				ID:        12345,
				Notes:     "This is the evidence note content.",
				Created:   "2025-01-15T10:30:00Z",
				Collected: "2025-01-01",
				Owner: &models.OrganizationMember{
					DisplayName: "John Smith",
				},
				IntegrationType:    "hubspot",
				IntegrationSubtype: "training",
			},
			contains: []string{
				"# Evidence Submission",
				"**Submitted by:** John Smith",
				"**Submitted on:** 2025-01-15",
				"**Collected:** 2025-01-01",
				"**Source:** hubspot (training)",
				"## Notes",
				"This is the evidence note content.",
				"*Submission ID: 12345*",
			},
			notContains: []string{
				"*No notes provided*",
			},
		},
		{
			name: "Submission with URL",
			att: &models.EvidenceAttachment{
				ID:    67890,
				Notes: "Check the linked document.",
				URL:   "https://example.com/document.pdf",
			},
			contains: []string{
				"# Evidence Submission",
				"## Notes",
				"Check the linked document.",
				"**URL:** https://example.com/document.pdf",
				"*Submission ID: 67890*",
			},
		},
		{
			name: "Submission with no notes",
			att: &models.EvidenceAttachment{
				ID:      11111,
				Notes:   "",
				Created: "2025-01-20T08:00:00Z",
			},
			contains: []string{
				"# Evidence Submission",
				"*No notes provided*",
				"*Submission ID: 11111*",
			},
		},
		{
			name: "Submission with integration type only",
			att: &models.EvidenceAttachment{
				ID:              22222,
				Notes:           "Automated evidence collection.",
				IntegrationType: "github",
			},
			contains: []string{
				"**Source:** github",
				"Automated evidence collection.",
			},
			notContains: []string{
				"(", // No subtype parentheses
			},
		},
		{
			name: "Minimal submission",
			att: &models.EvidenceAttachment{
				ID:    33333,
				Notes: "Minimal note.",
			},
			contains: []string{
				"# Evidence Submission",
				"## Notes",
				"Minimal note.",
				"*Submission ID: 33333*",
			},
			notContains: []string{
				"**Submitted by:**",
				"**Submitted on:**",
				"**Collected:**",
				"**Source:**",
				"**URL:**",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildSubmissionMarkdown(tt.att)

			for _, expected := range tt.contains {
				if !strings.Contains(result, expected) {
					t.Errorf("buildSubmissionMarkdown() missing expected content: %q\nGot:\n%s", expected, result)
				}
			}

			for _, notExpected := range tt.notContains {
				if strings.Contains(result, notExpected) {
					t.Errorf("buildSubmissionMarkdown() contains unexpected content: %q\nGot:\n%s", notExpected, result)
				}
			}
		})
	}
}

func TestIsTextOnlySubmission(t *testing.T) {
	tests := []struct {
		name     string
		att      models.EvidenceAttachment
		expected bool
	}{
		{
			name: "File submission with attachment",
			att: models.EvidenceAttachment{
				Type: "file",
				Attachment: &models.AttachmentFile{
					OriginalFilename: "report.pdf",
				},
				Notes: "Some notes",
			},
			expected: false,
		},
		{
			name: "URL submission with valid URL",
			att: models.EvidenceAttachment{
				Type:  "url",
				URL:   "https://example.com/doc",
				Notes: "Some notes",
			},
			expected: false,
		},
		{
			name: "Text-only with notes",
			att: models.EvidenceAttachment{
				Type:  "automated",
				Notes: "This is text-only evidence.",
			},
			expected: true,
		},
		{
			name: "File type but no attachment with notes",
			att: models.EvidenceAttachment{
				Type:       "file",
				Attachment: nil,
				Notes:      "Manual entry without file.",
			},
			expected: true,
		},
		{
			name: "URL type but empty URL with notes",
			att: models.EvidenceAttachment{
				Type:  "url",
				URL:   "",
				Notes: "Link was removed but notes remain.",
			},
			expected: true,
		},
		{
			name: "No type with notes",
			att: models.EvidenceAttachment{
				Type:  "",
				Notes: "Plain text submission.",
			},
			expected: true,
		},
		{
			name: "No file no URL no notes",
			att: models.EvidenceAttachment{
				Type:  "automated",
				Notes: "",
			},
			expected: false, // No content to save
		},
		{
			name: "Empty submission",
			att: models.EvidenceAttachment{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isTextOnlySubmission(&tt.att)
			if result != tt.expected {
				t.Errorf("isTextOnlySubmission() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0B"},
		{100, "100B"},
		{1023, "1023B"},
		{1024, "1.0KB"},
		{1536, "1.5KB"},
		{1048576, "1.0MB"},
		{1572864, "1.5MB"},
		{1073741824, "1.0GB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatBytes(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatBytes(%d) = %q, want %q", tt.bytes, result, tt.expected)
			}
		})
	}
}
