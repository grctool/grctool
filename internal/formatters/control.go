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
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interpolation"
	"github.com/grctool/grctool/internal/utils"
)

// ControlFormatter handles formatting controls into various output formats
type ControlFormatter struct {
	baseFormatter *BaseFormatter
}

// NewControlFormatter creates a new control formatter with default (disabled) interpolation
func NewControlFormatter() *ControlFormatter {
	// Create a disabled interpolator as default to maintain backward compatibility
	config := interpolation.InterpolatorConfig{
		Variables:         make(map[string]string),
		Enabled:           false,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(config)
	return &ControlFormatter{
		baseFormatter: NewBaseFormatter(interpolator),
	}
}

// NewControlFormatterWithInterpolation creates a new control formatter with the given interpolator
func NewControlFormatterWithInterpolation(interpolator interpolation.Interpolator) *ControlFormatter {
	return &ControlFormatter{
		baseFormatter: NewBaseFormatter(interpolator),
	}
}

// ToMarkdown converts a control to markdown format
func (cf *ControlFormatter) ToMarkdown(control *domain.Control) string {
	var md strings.Builder

	// Header section with ID and basic info
	md.WriteString(fmt.Sprintf("# Control %d\n\n", control.ID))

	// Metadata header box
	md.WriteString("```\n")
	md.WriteString(fmt.Sprintf("Control ID: %d\n", control.ID))
	md.WriteString(fmt.Sprintf("Framework: %s\n", control.Framework))
	md.WriteString(fmt.Sprintf("Category: %s\n", control.Category))
	md.WriteString(fmt.Sprintf("Status: %s\n", control.Status))
	if control.ImplementedDate != nil && !control.ImplementedDate.IsZero() {
		md.WriteString(fmt.Sprintf("Implemented: %s\n", control.ImplementedDate.Format("2006-01-02")))
	}
	if control.TestedDate != nil && !control.TestedDate.IsZero() {
		md.WriteString(fmt.Sprintf("Last Tested: %s\n", control.TestedDate.Format("2006-01-02")))
	}
	md.WriteString("```\n\n")

	// Control title
	if control.Name != "" {
		name := cf.baseFormatter.InterpolateText(control.Name)
		md.WriteString(fmt.Sprintf("## %s\n\n", name))
	}

	// Control description
	if control.Description != "" {
		md.WriteString("### Description\n\n")
		description := cf.baseFormatter.CleanContent(control.Description)
		md.WriteString(fmt.Sprintf("%s\n\n", description))
	}

	// Control help content (from master content)
	if control.MasterContent != nil && control.MasterContent.Help != "" {
		md.WriteString("### Implementation Help\n\n")
		help := cf.baseFormatter.CleanContent(control.MasterContent.Help)
		md.WriteString(fmt.Sprintf("%s\n\n", help))
	}

	// Control guidance content (from master content)
	if control.MasterContent != nil && control.MasterContent.Guidance != "" {
		md.WriteString("### Implementation Guidance\n\n")
		guidance := cf.baseFormatter.CleanContent(control.MasterContent.Guidance)
		md.WriteString(fmt.Sprintf("%s\n\n", guidance))
	}

	// Risk information
	if control.Risk != "" || control.RiskLevel != "" {
		md.WriteString("### Risk Information\n\n")
		if control.Risk != "" {
			md.WriteString(fmt.Sprintf("**Risk**: %s\n\n", cf.baseFormatter.InterpolateText(control.Risk)))
		}
		if control.RiskLevel != "" {
			md.WriteString(fmt.Sprintf("**Risk Level**: %s\n\n", control.RiskLevel))
		}
	}

	// Metadata section
	md.WriteString("---\n\n")
	md.WriteString("## Control Metadata\n\n")

	// Basic information
	md.WriteString("### Basic Information\n\n")
	basicRows := []MetadataRow{
		{"Control ID", strconv.Itoa(control.ID)},
		{"Reference ID", control.ReferenceID},
		{"Name", cf.baseFormatter.InterpolateText(control.Name)},
		{"Category", control.Category},
		{"Framework", control.Framework},
		{"Status", control.Status},
	}
	if control.RiskLevel != "" {
		basicRows = append(basicRows, MetadataRow{"Risk Level", control.RiskLevel})
	}
	if control.IsAutoImplemented {
		basicRows = append(basicRows, MetadataRow{"Auto-Implemented", "Yes"})
	}
	md.WriteString(cf.baseFormatter.FormatMetadataTable(basicRows))

	// Implementation dates
	if (control.ImplementedDate != nil && !control.ImplementedDate.IsZero()) ||
		(control.TestedDate != nil && !control.TestedDate.IsZero()) {
		md.WriteString("\n### Implementation Timeline\n\n")
		dateRows := []MetadataRow{}
		if control.ImplementedDate != nil && !control.ImplementedDate.IsZero() {
			dateRows = append(dateRows, MetadataRow{"Implemented Date", control.ImplementedDate.Format("2006-01-02")})
		}
		if control.TestedDate != nil && !control.TestedDate.IsZero() {
			dateRows = append(dateRows, MetadataRow{"Last Tested", control.TestedDate.Format("2006-01-02")})
		}
		md.WriteString(cf.baseFormatter.FormatMetadataTable(dateRows))
	}

	// Association counts
	if control.Associations != nil && (control.Associations.Policies > 0 ||
		control.Associations.Procedures > 0 || control.Associations.Evidence > 0 ||
		control.Associations.Risks > 0) {
		md.WriteString("\n### Associated Items\n\n")
		md.WriteString("| Type | Count |\n")
		md.WriteString("|------|-------|\n")
		if control.Associations.Policies > 0 {
			md.WriteString(fmt.Sprintf("| **Policies** | %d |\n", control.Associations.Policies))
		}
		if control.Associations.Procedures > 0 {
			md.WriteString(fmt.Sprintf("| **Procedures** | %d |\n", control.Associations.Procedures))
		}
		if control.Associations.Evidence > 0 {
			md.WriteString(fmt.Sprintf("| **Evidence Tasks** | %d |\n", control.Associations.Evidence))
		}
		if control.Associations.Risks > 0 {
			md.WriteString(fmt.Sprintf("| **Risk Items** | %d |\n", control.Associations.Risks))
		}
	}

	// Evidence metrics
	if control.OrgEvidenceMetrics != nil && (control.OrgEvidenceMetrics.TotalCount > 0 ||
		control.OrgEvidenceMetrics.CompleteCount > 0 || control.OrgEvidenceMetrics.OverdueCount > 0) {
		md.WriteString("\n### Evidence Metrics\n\n")
		md.WriteString("| Metric | Value |\n")
		md.WriteString("|--------|-------|\n")
		if control.OrgEvidenceMetrics.TotalCount > 0 {
			md.WriteString(fmt.Sprintf("| **Total Evidence** | %d |\n", control.OrgEvidenceMetrics.TotalCount))
		}
		if control.OrgEvidenceMetrics.CompleteCount > 0 {
			md.WriteString(fmt.Sprintf("| **Complete Evidence** | %d |\n", control.OrgEvidenceMetrics.CompleteCount))
		}
		if control.OrgEvidenceMetrics.OverdueCount > 0 {
			md.WriteString(fmt.Sprintf("| **Overdue Evidence** | %d |\n", control.OrgEvidenceMetrics.OverdueCount))
		}
	}

	// Other metrics
	if control.RecommendedEvidenceCount > 0 || control.OpenIncidentCount > 0 {
		md.WriteString("\n### Additional Metrics\n\n")
		md.WriteString("| Metric | Value |\n")
		md.WriteString("|--------|-------|\n")
		if control.RecommendedEvidenceCount > 0 {
			md.WriteString(fmt.Sprintf("| **Recommended Evidence** | %d |\n", control.RecommendedEvidenceCount))
		}
		if control.OpenIncidentCount > 0 {
			md.WriteString(fmt.Sprintf("| **Open Incidents** | %d |\n", control.OpenIncidentCount))
		}
	}

	// Usage statistics
	usageStats := UsageStats{
		ViewCount:        control.ViewCount,
		LastViewedAt:     getTimeValue(control.LastViewedAt),
		DownloadCount:    control.DownloadCount,
		LastDownloadedAt: getTimeValue(control.LastDownloadedAt),
		ReferenceCount:   control.ReferenceCount,
		LastReferencedAt: getTimeValue(control.LastReferencedAt),
	}
	md.WriteString(cf.baseFormatter.FormatUsageStatistics(usageStats))

	// Assignees
	if len(control.Assignees) > 0 {
		people := make([]PersonInfo, 0, len(control.Assignees))
		for _, assignee := range control.Assignees {
			people = append(people, PersonInfo{
				Name:       assignee.Name,
				Email:      assignee.Email,
				Role:       "", // Controls don't have assignee roles in the same way as policies
				AssignedAt: getTimeValue(assignee.AssignedAt),
			})
		}
		md.WriteString(cf.baseFormatter.FormatPersonList(people, "Assignees"))
	}

	// Audit projects
	if len(control.AuditProjects) > 0 {
		md.WriteString("\n### Audit Projects\n\n")
		for _, project := range control.AuditProjects {
			md.WriteString(fmt.Sprintf("- **%s** (Status: %s)", project.Name, project.Status))
			if project.Description != "" {
				md.WriteString(fmt.Sprintf(" - %s", project.Description))
			}
			md.WriteString("\n")
		}
	}

	// Framework codes
	if len(control.FrameworkCodes) > 0 {
		md.WriteString("\n### Framework Codes\n\n")
		for _, code := range control.FrameworkCodes {
			md.WriteString(fmt.Sprintf("- **%s**: %s", code.Framework, code.Code))
			if code.Name != "" {
				md.WriteString(fmt.Sprintf(" - %s", code.Name))
			}
			md.WriteString("\n")
		}
	}

	// Tags
	if len(control.Tags) > 0 {
		tags := make([]TagInfo, 0, len(control.Tags))
		for _, tag := range control.Tags {
			tags = append(tags, TagInfo{
				Name:  tag.Name,
				Color: tag.Color,
			})
		}
		md.WriteString(cf.baseFormatter.FormatTagList(tags))
	}

	// Codes field (if present)
	if control.Codes != "" {
		md.WriteString("\n### Control Codes\n\n")
		md.WriteString(fmt.Sprintf("%s\n", control.Codes))
	}

	// Deprecation notes
	if control.DeprecationNotes != "" {
		md.WriteString("\n### ⚠️ Deprecation Notice\n\n")
		md.WriteString(fmt.Sprintf("> %s\n", control.DeprecationNotes))
	}

	// Footer
	md.WriteString(fmt.Sprintf("\n---\n\n*Generated on %s*\n", time.Now().Format("2006-01-02 15:04:05 MST")))

	return md.String()
}

// ToSummaryMarkdown creates a brief markdown summary of the control
func (cf *ControlFormatter) ToSummaryMarkdown(control *domain.Control) string {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("## %s\n\n", control.Name))
	md.WriteString(fmt.Sprintf("**ID:** %d | **Category:** %s | **Status:** %s\n\n",
		control.ID, control.Category, control.Status))

	if control.Description != "" {
		// Use first paragraph of description
		desc := strings.Split(control.Description, "\n")[0]
		if len(desc) > 200 {
			desc = desc[:197] + "..."
		}
		md.WriteString(fmt.Sprintf("%s\n\n", desc))
	}

	// Quick stats
	var stats []string
	if control.Associations != nil {
		if control.Associations.Policies > 0 {
			stats = append(stats, fmt.Sprintf("%d policies", control.Associations.Policies))
		}
		if control.Associations.Evidence > 0 {
			stats = append(stats, fmt.Sprintf("%d evidence tasks", control.Associations.Evidence))
		}
	}
	if len(stats) > 0 {
		md.WriteString("**Associated:** ")
		md.WriteString(strings.Join(stats, ", "))
		md.WriteString("\n\n")
	}

	// Implementation status
	if control.ImplementedDate != nil && !control.ImplementedDate.IsZero() {
		md.WriteString(fmt.Sprintf("*Implemented: %s*", control.ImplementedDate.Format("2006-01-02")))
		if control.TestedDate != nil && !control.TestedDate.IsZero() {
			md.WriteString(fmt.Sprintf(" | *Last tested: %s*", control.TestedDate.Format("2006-01-02")))
		}
		md.WriteString("\n")
	}

	return md.String()
}

// ToDocumentMarkdown creates a comprehensive control document in markdown format
// This is the main method for generating control documents that will be saved to files
func (cf *ControlFormatter) ToDocumentMarkdown(control *domain.Control) string {
	var md strings.Builder

	// Use actual reference ID or generate one as fallback
	var reference string
	if control.ReferenceID != "" {
		reference = control.ReferenceID
	} else {
		// Fallback to generated reference if ReferenceID is not set
		reference = cf.generateReference(control.Name)
	}

	// Document title with reference
	md.WriteString(fmt.Sprintf("# %s - %s\n\n", reference, control.Name))

	// Control description
	if control.Description != "" {
		md.WriteString("## Description\n\n")
		md.WriteString(fmt.Sprintf("%s\n\n", cf.baseFormatter.CleanContent(control.Description)))
	}

	// Implementation help
	if control.MasterContent != nil && control.MasterContent.Help != "" {
		md.WriteString("## Implementation Help\n\n")
		help := cf.baseFormatter.CleanContent(control.MasterContent.Help)
		md.WriteString(fmt.Sprintf("%s\n\n", help))
	}

	// Implementation guidance
	if control.MasterContent != nil && control.MasterContent.Guidance != "" {
		md.WriteString("## Implementation Guidance\n\n")
		guidance := cf.baseFormatter.CleanContent(control.MasterContent.Guidance)
		md.WriteString(fmt.Sprintf("%s\n\n", guidance))
	}

	// Risk information
	if control.Risk != "" || control.RiskLevel != "" {
		md.WriteString("## Risk Information\n\n")
		if control.Risk != "" {
			md.WriteString(fmt.Sprintf("**Risk**: %s\n\n", cf.baseFormatter.InterpolateText(control.Risk)))
		}
		if control.RiskLevel != "" {
			md.WriteString(fmt.Sprintf("**Risk Level**: %s\n\n", control.RiskLevel))
		}
	}

	// Document metadata footer
	md.WriteString("---\n\n")
	md.WriteString("## Document Information\n\n")

	// Basic metadata table
	metadataRows := []MetadataRow{
		{"Document Reference", reference},
		{"Control ID", strconv.Itoa(control.ID)},
		{"Reference ID", control.ReferenceID},
		{"Control Name", control.Name},
		{"Category", control.Category},
		{"Framework", control.Framework},
		{"Status", control.Status},
	}
	if control.RiskLevel != "" {
		metadataRows = append(metadataRows, MetadataRow{"Risk Level", control.RiskLevel})
	}
	if control.IsAutoImplemented {
		metadataRows = append(metadataRows, MetadataRow{"Auto-Implemented", "Yes"})
	}
	if control.ImplementedDate != nil && !control.ImplementedDate.IsZero() {
		metadataRows = append(metadataRows, MetadataRow{"Implemented Date", control.ImplementedDate.Format("2006-01-02")})
	}
	if control.TestedDate != nil && !control.TestedDate.IsZero() {
		metadataRows = append(metadataRows, MetadataRow{"Last Tested", control.TestedDate.Format("2006-01-02")})
	}

	md.WriteString(cf.baseFormatter.FormatMetadataTable(metadataRows))

	// Association information
	if control.Associations != nil && (control.Associations.Policies > 0 ||
		control.Associations.Procedures > 0 || control.Associations.Evidence > 0 ||
		control.Associations.Risks > 0) {
		md.WriteString("\n### Associated Items\n\n")
		md.WriteString("| Type | Count |\n")
		md.WriteString("|------|-------|\n")
		if control.Associations.Policies > 0 {
			md.WriteString(fmt.Sprintf("| **Associated Policies** | %d |\n", control.Associations.Policies))
		}
		if control.Associations.Procedures > 0 {
			md.WriteString(fmt.Sprintf("| **Associated Procedures** | %d |\n", control.Associations.Procedures))
		}
		if control.Associations.Evidence > 0 {
			md.WriteString(fmt.Sprintf("| **Associated Evidence Tasks** | %d |\n", control.Associations.Evidence))
		}
		if control.Associations.Risks > 0 {
			md.WriteString(fmt.Sprintf("| **Associated Risk Items** | %d |\n", control.Associations.Risks))
		}
	}

	// Evidence metrics
	if control.OrgEvidenceMetrics != nil && (control.OrgEvidenceMetrics.TotalCount > 0 ||
		control.OrgEvidenceMetrics.CompleteCount > 0 || control.OrgEvidenceMetrics.OverdueCount > 0) {
		md.WriteString("\n### Evidence Metrics\n\n")
		md.WriteString("| Metric | Value |\n")
		md.WriteString("|--------|-------|\n")
		if control.OrgEvidenceMetrics.TotalCount > 0 {
			md.WriteString(fmt.Sprintf("| **Total Evidence** | %d |\n", control.OrgEvidenceMetrics.TotalCount))
		}
		if control.OrgEvidenceMetrics.CompleteCount > 0 {
			md.WriteString(fmt.Sprintf("| **Complete Evidence** | %d |\n", control.OrgEvidenceMetrics.CompleteCount))
		}
		if control.OrgEvidenceMetrics.OverdueCount > 0 {
			md.WriteString(fmt.Sprintf("| **Overdue Evidence** | %d |\n", control.OrgEvidenceMetrics.OverdueCount))
		}
	}

	// Assignees
	if len(control.Assignees) > 0 {
		people := make([]PersonInfo, 0, len(control.Assignees))
		for _, assignee := range control.Assignees {
			if assignee.Name != "" || assignee.Email != "" {
				people = append(people, PersonInfo{
					Name:       assignee.Name,
					Email:      assignee.Email,
					AssignedAt: getTimeValue(assignee.AssignedAt),
				})
			}
		}
		if len(people) > 0 {
			md.WriteString(cf.baseFormatter.FormatPersonList(people, "Assignees"))
		}
	}

	// Audit projects
	if len(control.AuditProjects) > 0 {
		md.WriteString("\n### Audit Projects\n\n")
		for _, project := range control.AuditProjects {
			if project.Name != "" {
				md.WriteString(fmt.Sprintf("- **%s** (Status: %s)", project.Name, project.Status))
				if project.Description != "" {
					md.WriteString(fmt.Sprintf(" - %s", project.Description))
				}
				md.WriteString("\n")
			}
		}
	}

	// Framework codes
	if len(control.FrameworkCodes) > 0 {
		md.WriteString("\n### Framework Codes\n\n")
		for _, code := range control.FrameworkCodes {
			if code.Framework != "" || code.Code != "" {
				md.WriteString(fmt.Sprintf("- **%s**: %s", code.Framework, code.Code))
				if code.Name != "" {
					md.WriteString(fmt.Sprintf(" - %s", code.Name))
				}
				md.WriteString("\n")
			}
		}
	}

	// Tags
	if len(control.Tags) > 0 {
		tags := make([]TagInfo, 0, len(control.Tags))
		for _, tag := range control.Tags {
			if tag.Name != "" {
				tags = append(tags, TagInfo{
					Name:  tag.Name,
					Color: tag.Color,
				})
			}
		}
		if len(tags) > 0 {
			md.WriteString(cf.baseFormatter.FormatTagList(tags))
		}
	}

	// Usage statistics (if available)
	usageStats := UsageStats{
		ViewCount:        control.ViewCount,
		LastViewedAt:     getTimeValue(control.LastViewedAt),
		DownloadCount:    control.DownloadCount,
		LastDownloadedAt: getTimeValue(control.LastDownloadedAt),
		ReferenceCount:   control.ReferenceCount,
		LastReferencedAt: getTimeValue(control.LastReferencedAt),
	}
	md.WriteString(cf.baseFormatter.FormatUsageStatistics(usageStats))

	// Deprecation notice
	if control.DeprecationNotes != "" {
		md.WriteString("\n### ⚠️ Deprecation Notice\n\n")
		md.WriteString(fmt.Sprintf("> %s\n", control.DeprecationNotes))
	}

	// Document footer
	md.WriteString(cf.baseFormatter.GenerateDocumentFooter())

	return md.String()
}

// generateReference creates a filename-friendly reference from control name
func (cf *ControlFormatter) generateReference(name string) string {
	// Convert common control names to standard references
	referenceMap := map[string]string{
		"access provisioning":         "C1",
		"access approval":             "C1",
		"user access management":      "C2",
		"privileged access":           "C3",
		"password management":         "C4",
		"password policy":             "C4",
		"authentication":              "C5",
		"multi-factor authentication": "C5",
		"mfa":                         "C5",
		"session management":          "C6",
		"network access control":      "C7",
		"network segmentation":        "C8",
		"firewall":                    "C9",
		"intrusion detection":         "C10",
		"intrusion prevention":        "C10",
		"vulnerability management":    "C11",
		"vulnerability scanning":      "C11",
		"patch management":            "C12",
		"system hardening":            "C13",
		"configuration management":    "C14",
		"change management":           "C15",
		"data encryption":             "C16",
		"encryption":                  "C16",
		"data backup":                 "C17",
		"backup":                      "C17",
		"data retention":              "C18",
		"data disposal":               "C19",
		"incident response":           "C20",
		"security incident":           "C20",
		"logging":                     "C21",
		"log management":              "C21",
		"monitoring":                  "C22",
		"security monitoring":         "C22",
		"audit":                       "C23",
		"audit logging":               "C23",
		"risk assessment":             "C24",
		"risk management":             "C24",
		"security training":           "C25",
		"awareness training":          "C25",
		"security awareness":          "C25",
		"physical security":           "C26",
		"asset management":            "C27",
		"asset inventory":             "C27",
		"vendor management":           "C28",
		"third party":                 "C28",
		"business continuity":         "C29",
		"disaster recovery":           "C30",
	}

	// Normalize the name for lookup
	normalized := strings.ToLower(strings.TrimSpace(name))

	// Check for exact matches first
	if ref, exists := referenceMap[normalized]; exists {
		return ref
	}

	// Check for partial matches
	for key, ref := range referenceMap {
		if strings.Contains(normalized, key) {
			return ref
		}
	}

	// If no match found, generate a generic reference
	words := strings.Fields(normalized)
	if len(words) > 0 {
		// Use first letter of each word, up to 3 characters, with C prefix
		var initials strings.Builder
		initials.WriteString("C")
		for i, word := range words {
			if i >= 2 { // Limit to 2 additional characters after C
				break
			}
			if len(word) > 0 {
				initials.WriteString(strings.ToUpper(string(word[0])))
			}
		}
		return initials.String()
	}

	return "CTRL"
}

// GetDocumentFilename generates a filename for a control document using unified pattern
func (cf *ControlFormatter) GetDocumentFilename(control *domain.Control) string {
	fg := utils.NewFilenameGenerator()
	return fg.GenerateFilename(control.ReferenceID, strconv.Itoa(control.ID), control.Name, "md")
}

// getTimeValue safely extracts a time value, handling nil pointers
func getTimeValue(t *time.Time) time.Time {
	if t == nil {
		return time.Time{}
	}
	return *t
}
