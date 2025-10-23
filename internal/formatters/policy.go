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
	"html"
	"regexp"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interpolation"
	"github.com/grctool/grctool/internal/utils"
)

// PolicyFormatter handles formatting policies into various output formats
type PolicyFormatter struct {
	baseFormatter *BaseFormatter
}

// NewPolicyFormatter creates a new policy formatter with default (disabled) interpolation
func NewPolicyFormatter() *PolicyFormatter {
	// Create a disabled interpolator as default to maintain backward compatibility
	config := interpolation.InterpolatorConfig{
		Variables:         make(map[string]string),
		Enabled:           false,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(config)
	return &PolicyFormatter{
		baseFormatter: NewBaseFormatter(interpolator),
	}
}

// NewPolicyFormatterWithInterpolation creates a new policy formatter with the given interpolator
func NewPolicyFormatterWithInterpolation(interpolator interpolation.Interpolator) *PolicyFormatter {
	return &PolicyFormatter{
		baseFormatter: NewBaseFormatter(interpolator),
	}
}

// ToMarkdown converts a policy to markdown format
func (pf *PolicyFormatter) ToMarkdown(policy *domain.Policy) string {
	var md strings.Builder

	// Header section with reference ID, ID, version, and last updated
	if policy.ReferenceID != "" {
		md.WriteString(fmt.Sprintf("# Policy %s - %s\n\n", policy.ReferenceID, policy.Name))
	} else {
		md.WriteString(fmt.Sprintf("# Policy %s\n\n", policy.ID))
	}

	// Metadata header box
	md.WriteString("```\n")
	if policy.ReferenceID != "" {
		md.WriteString(fmt.Sprintf("Reference ID: %s\n", policy.ReferenceID))
	}
	md.WriteString(fmt.Sprintf("Policy ID: %s\n", policy.ID))
	if policy.Category != "" {
		md.WriteString(fmt.Sprintf("Category: %s\n", policy.Category))
	}
	if policy.MasterPolicyID != "" {
		md.WriteString(fmt.Sprintf("Master Policy ID: %s\n", policy.MasterPolicyID))
	}
	if policy.Version != "" {
		md.WriteString(fmt.Sprintf("Version: %s\n", policy.Version))
	}
	if policy.CurrentVersion != nil && policy.CurrentVersion.Version != "" {
		md.WriteString(fmt.Sprintf("Current Version: %s\n", policy.CurrentVersion.Version))
	}
	md.WriteString(fmt.Sprintf("Framework: %s\n", policy.Framework))
	md.WriteString(fmt.Sprintf("Status: %s\n", policy.Status))
	md.WriteString(fmt.Sprintf("Last Updated: %s\n", policy.UpdatedAt.Format("2006-01-02 15:04:05 MST")))
	md.WriteString("```\n\n")

	// Policy title
	if policy.Name != "" {
		name := pf.baseFormatter.InterpolateText(policy.Name)
		md.WriteString(fmt.Sprintf("## %s\n\n", name))
	}

	// Policy summary (if different from description)
	if policy.Summary != "" && policy.Summary != policy.Description {
		md.WriteString("### Summary\n\n")
		summary := pf.baseFormatter.InterpolateText(policy.Summary)
		md.WriteString(fmt.Sprintf("%s\n\n", summary))
	}

	// Main policy content/document
	if policy.Content != "" {
		md.WriteString("### Policy Document\n\n")
		// Clean up the content and ensure proper markdown formatting
		content := pf.baseFormatter.CleanContent(policy.Content)
		md.WriteString(fmt.Sprintf("%s\n\n", content))
	} else if policy.Description != "" {
		md.WriteString("### Policy Document\n\n")
		description := pf.baseFormatter.InterpolateText(policy.Description)
		md.WriteString(fmt.Sprintf("%s\n\n", description))
	}

	// Metadata section
	md.WriteString("---\n\n")
	md.WriteString("## Policy Metadata\n\n")

	// Basic information
	md.WriteString("### Basic Information\n\n")
	md.WriteString("| Field | Value |\n")
	md.WriteString("|-------|-------|\n")
	if policy.ReferenceID != "" {
		md.WriteString(fmt.Sprintf("| **Reference ID** | %s |\n", policy.ReferenceID))
	}
	md.WriteString(fmt.Sprintf("| **Policy ID** | %s |\n", policy.ID))
	md.WriteString(fmt.Sprintf("| **Name** | %s |\n", pf.baseFormatter.InterpolateText(policy.Name)))
	if policy.Category != "" {
		md.WriteString(fmt.Sprintf("| **Category** | %s |\n", policy.Category))
	}
	if policy.MasterPolicyID != "" {
		md.WriteString(fmt.Sprintf("| **Master Policy ID** | %s |\n", policy.MasterPolicyID))
	}
	md.WriteString(fmt.Sprintf("| **Framework** | %s |\n", policy.Framework))
	md.WriteString(fmt.Sprintf("| **Status** | %s |\n", policy.Status))
	md.WriteString(fmt.Sprintf("| **Created** | %s |\n", policy.CreatedAt.Format("2006-01-02")))
	md.WriteString(fmt.Sprintf("| **Last Updated** | %s |\n", policy.UpdatedAt.Format("2006-01-02")))

	// Version information
	if policy.CurrentVersion != nil || policy.LatestVersion != nil {
		md.WriteString("\n### Version Information\n\n")
		md.WriteString("| Field | Value |\n")
		md.WriteString("|-------|-------|\n")

		if policy.CurrentVersion != nil {
			md.WriteString(fmt.Sprintf("| **Current Version** | %s |\n", policy.CurrentVersion.Version))
			md.WriteString(fmt.Sprintf("| **Version Status** | %s |\n", policy.CurrentVersion.Status))
			md.WriteString(fmt.Sprintf("| **Version Created** | %s |\n", policy.CurrentVersion.CreatedAt.Format("2006-01-02")))
			if policy.CurrentVersion.CreatedBy != "" {
				md.WriteString(fmt.Sprintf("| **Version Author** | %s |\n", policy.CurrentVersion.CreatedBy))
			}
		}

		if policy.LatestVersion != nil && (policy.CurrentVersion == nil || policy.LatestVersion.Version != policy.CurrentVersion.Version) {
			md.WriteString(fmt.Sprintf("| **Latest Version** | %s |\n", policy.LatestVersion.Version))
		}
	}

	// Association counts
	if policy.ControlCount > 0 || policy.ProcedureCount > 0 || policy.EvidenceCount > 0 {
		md.WriteString("\n### Associated Items\n\n")
		md.WriteString("| Type | Count |\n")
		md.WriteString("|------|-------|\n")
		if policy.ControlCount > 0 {
			md.WriteString(fmt.Sprintf("| **Controls** | %d |\n", policy.ControlCount))
		}
		if policy.ProcedureCount > 0 {
			md.WriteString(fmt.Sprintf("| **Procedures** | %d |\n", policy.ProcedureCount))
		}
		if policy.EvidenceCount > 0 {
			md.WriteString(fmt.Sprintf("| **Evidence Tasks** | %d |\n", policy.EvidenceCount))
		}
		if policy.RiskCount > 0 {
			md.WriteString(fmt.Sprintf("| **Risk Items** | %d |\n", policy.RiskCount))
		}
	}

	// Usage statistics
	if policy.ViewCount > 0 || policy.DownloadCount > 0 || policy.ReferenceCount > 0 {
		md.WriteString("\n### Usage Statistics\n\n")
		md.WriteString("| Metric | Value |\n")
		md.WriteString("|--------|-------|\n")
		if policy.ViewCount > 0 {
			md.WriteString(fmt.Sprintf("| **Views** | %d |\n", policy.ViewCount))
			if policy.LastViewedAt != nil {
				md.WriteString(fmt.Sprintf("| **Last Viewed** | %s |\n", policy.LastViewedAt.Format("2006-01-02 15:04")))
			}
		}
		if policy.DownloadCount > 0 {
			md.WriteString(fmt.Sprintf("| **Downloads** | %d |\n", policy.DownloadCount))
			if policy.LastDownloadedAt != nil {
				md.WriteString(fmt.Sprintf("| **Last Downloaded** | %s |\n", policy.LastDownloadedAt.Format("2006-01-02 15:04")))
			}
		}
		if policy.ReferenceCount > 0 {
			md.WriteString(fmt.Sprintf("| **References** | %d |\n", policy.ReferenceCount))
			if policy.LastReferencedAt != nil {
				md.WriteString(fmt.Sprintf("| **Last Referenced** | %s |\n", policy.LastReferencedAt.Format("2006-01-02 15:04")))
			}
		}
	}

	// Assignees and reviewers
	if len(policy.Assignees) > 0 {
		md.WriteString("\n### Assignees\n\n")
		for _, assignee := range policy.Assignees {
			md.WriteString(fmt.Sprintf("- **%s** <%s>", assignee.Name, assignee.Email))
			if assignee.Role != "" {
				md.WriteString(fmt.Sprintf(" (%s)", assignee.Role))
			}
			if assignee.AssignedAt != nil {
				md.WriteString(fmt.Sprintf(" - Assigned: %s", assignee.AssignedAt.Format("2006-01-02")))
			}
			md.WriteString("\n")
		}
	}

	if len(policy.Reviewers) > 0 {
		md.WriteString("\n### Reviewers\n\n")
		for _, reviewer := range policy.Reviewers {
			md.WriteString(fmt.Sprintf("- **%s** <%s>", reviewer.Name, reviewer.Email))
			if reviewer.Role != "" {
				md.WriteString(fmt.Sprintf(" (%s)", reviewer.Role))
			}
			md.WriteString("\n")
		}
	}

	// Tags
	if len(policy.Tags) > 0 {
		md.WriteString("\n### Tags\n\n")
		for _, tag := range policy.Tags {
			if tag.Color != "" {
				md.WriteString(fmt.Sprintf("- `%s` <span style=\"color: %s\">●</span>\n", tag.Name, tag.Color))
			} else {
				md.WriteString(fmt.Sprintf("- `%s`\n", tag.Name))
			}
		}
	}

	// Deprecation notes
	if policy.DeprecationNotes != "" {
		md.WriteString("\n### ⚠️ Deprecation Notice\n\n")
		md.WriteString(fmt.Sprintf("> %s\n", policy.DeprecationNotes))
	}

	// Footer
	md.WriteString(fmt.Sprintf("\n---\n\n*Generated on %s*\n", time.Now().Format("2006-01-02 15:04:05 MST")))

	return md.String()
}

// cleanPolicyContent cleans up policy content for better markdown rendering
func (pf *PolicyFormatter) cleanPolicyContent(content string) string {
	// Apply interpolation first to any content that might not go through convertHTMLToMarkdown
	content = pf.baseFormatter.InterpolateText(content)

	// Convert HTML to markdown
	markdown := pf.convertHTMLToMarkdown(content)

	// Remove excessive whitespace
	lines := strings.Split(markdown, "\n")
	var cleanLines []string

	for _, line := range lines {
		// Trim whitespace but preserve intentional indentation
		trimmed := strings.TrimSpace(line)
		if trimmed != "" {
			cleanLines = append(cleanLines, trimmed)
		} else if len(cleanLines) > 0 && cleanLines[len(cleanLines)-1] != "" {
			// Preserve single empty lines for paragraph breaks
			cleanLines = append(cleanLines, "")
		}
	}

	// Join lines back together
	cleaned := strings.Join(cleanLines, "\n")

	// Ensure there's not excessive blank lines
	cleaned = strings.ReplaceAll(cleaned, "\n\n\n", "\n\n")

	// Remove trailing newlines but preserve content structure
	cleaned = strings.TrimRight(cleaned, "\n")

	return cleaned
}

// convertHTMLToMarkdown converts HTML content to markdown format
func (pf *PolicyFormatter) convertHTMLToMarkdown(content string) string {
	// Apply variable interpolation first (this replaces the hardcoded substitution)
	content = pf.baseFormatter.InterpolateText(content)
	// If interpolation fails, we continue with the original content

	// Convert common HTML elements to markdown
	conversions := []struct {
		pattern string
		replace string
	}{
		// Headers
		{`(?i)<h1[^>]*>(.*?)</h1>`, "# $1"},
		{`(?i)<h2[^>]*>(.*?)</h2>`, "## $1"},
		{`(?i)<h3[^>]*>(.*?)</h3>`, "### $1"},
		{`(?i)<h4[^>]*>(.*?)</h4>`, "#### $1"},
		{`(?i)<h5[^>]*>(.*?)</h5>`, "##### $1"},
		{`(?i)<h6[^>]*>(.*?)</h6>`, "###### $1"},

		// Formatting
		{`(?i)<strong[^>]*>(.*?)</strong>`, "**$1**"},
		{`(?i)<b[^>]*>(.*?)</b>`, "**$1**"},
		{`(?i)<em[^>]*>(.*?)</em>`, "*$1*"},
		{`(?i)<i[^>]*>(.*?)</i>`, "*$1*"},

		// Links
		{`(?i)<a[^>]*href=["']([^"']*)["'][^>]*>(.*?)</a>`, "[$2]($1)"},

		// Lists
		{`(?i)<ul[^>]*>`, ""},
		{`(?i)</ul>`, ""},
		{`(?i)<ol[^>]*>`, ""},
		{`(?i)</ol>`, ""},
		{`(?i)<li[^>]*>`, "- "},
		{`(?i)</li>`, ""},

		// Paragraphs and line breaks
		{`(?i)<p[^>]*>`, ""},
		{`(?i)</p>`, "\n\n"},
		{`(?i)<br[^>]*>`, "\n"},
		{`(?i)<br[^>]*/>`, "\n"},

		// Divs
		{`(?i)<div[^>]*>`, ""},
		{`(?i)</div>`, "\n"},

		// Code and preformatted text
		{`(?i)<code[^>]*>(.*?)</code>`, "`$1`"},
		{`(?i)<pre[^>]*>(.*?)</pre>`, "```\n$1\n```"},

		// Remove spans with style attributes but preserve content
		{`(?i)<span[^>]*>(.*?)</span>`, "$1"},
	}

	// Apply conversions
	for _, conv := range conversions {
		re := regexp.MustCompile(conv.pattern)
		content = re.ReplaceAllString(content, conv.replace)
	}

	// Convert tables - inline implementation since we removed the separate method
	// Match table structure
	tableRegex := regexp.MustCompile(`(?i)<table[^>]*>(.*?)</table>`)
	tables := tableRegex.FindAllStringSubmatch(content, -1)

	for _, table := range tables {
		tableContent := table[1]
		markdownTable := pf.convertSingleHTMLTable(tableContent)
		content = strings.Replace(content, table[0], markdownTable, 1)
	}

	// Clean up HTML entities
	content = html.UnescapeString(content)

	// Remove any remaining HTML tags
	re := regexp.MustCompile(`<[^>]*>`)
	content = re.ReplaceAllString(content, "")

	// Clean up excessive whitespace
	content = regexp.MustCompile(`\n{3,}`).ReplaceAllString(content, "\n\n")
	content = regexp.MustCompile(`[ \t]+`).ReplaceAllString(content, " ")

	// Clean up &nbsp; and other common entities
	content = strings.ReplaceAll(content, "&nbsp;", " ")
	content = strings.ReplaceAll(content, "&rsquo;", "'")
	content = strings.ReplaceAll(content, "&ndash;", "–")
	content = strings.ReplaceAll(content, "&mdash;", "—")
	content = strings.ReplaceAll(content, "&ldquo;", "\"")
	content = strings.ReplaceAll(content, "&rdquo;", "\"")

	return strings.TrimSpace(content)
}

// convertSingleHTMLTable converts a single HTML table to markdown
func (pf *PolicyFormatter) convertSingleHTMLTable(tableHTML string) string {
	var result strings.Builder

	// Extract table rows
	rowRegex := regexp.MustCompile(`(?i)<tr[^>]*>(.*?)</tr>`)
	rows := rowRegex.FindAllStringSubmatch(tableHTML, -1)

	for i, row := range rows {
		rowContent := row[1]

		// Extract cells (both td and th)
		cellRegex := regexp.MustCompile(`(?i)<t[hd][^>]*>(.*?)</t[hd]>`)
		cells := cellRegex.FindAllStringSubmatch(rowContent, -1)

		if len(cells) > 0 {
			// Build markdown row
			result.WriteString("|")
			for _, cell := range cells {
				cellContent := strings.TrimSpace(cell[1])
				// Clean up HTML from cell content
				cellContent = regexp.MustCompile(`<[^>]*>`).ReplaceAllString(cellContent, "")
				cellContent = html.UnescapeString(cellContent)
				cellContent = strings.ReplaceAll(cellContent, "\n", " ")
				cellContent = strings.ReplaceAll(cellContent, "|", "\\|")
				result.WriteString(fmt.Sprintf(" %s |", cellContent))
			}
			result.WriteString("\n")

			// Add header separator for first row
			if i == 0 {
				result.WriteString("|")
				for range cells {
					result.WriteString("-------|")
				}
				result.WriteString("\n")
			}
		}
	}

	return result.String()
}

// ToSummaryMarkdown creates a brief markdown summary of the policy
func (pf *PolicyFormatter) ToSummaryMarkdown(policy *domain.Policy) string {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("## %s\n\n", policy.Name))
	if policy.ReferenceID != "" {
		md.WriteString(fmt.Sprintf("**ID:** %s (%s) | **Framework:** %s | **Status:** %s\n\n",
			policy.ReferenceID, policy.ID, policy.Framework, policy.Status))
	} else {
		md.WriteString(fmt.Sprintf("**ID:** %s | **Framework:** %s | **Status:** %s\n\n",
			policy.ID, policy.Framework, policy.Status))
	}

	if policy.Summary != "" {
		md.WriteString(fmt.Sprintf("%s\n\n", policy.Summary))
	} else if policy.Description != "" {
		// Use first paragraph of description
		desc := strings.Split(policy.Description, "\n")[0]
		if len(desc) > 200 {
			desc = desc[:197] + "..."
		}
		md.WriteString(fmt.Sprintf("%s\n\n", desc))
	}

	// Quick stats
	if policy.ControlCount > 0 || policy.EvidenceCount > 0 {
		md.WriteString("**Associated:** ")
		var stats []string
		if policy.ControlCount > 0 {
			stats = append(stats, fmt.Sprintf("%d controls", policy.ControlCount))
		}
		if policy.EvidenceCount > 0 {
			stats = append(stats, fmt.Sprintf("%d evidence tasks", policy.EvidenceCount))
		}
		md.WriteString(strings.Join(stats, ", "))
		md.WriteString("\n\n")
	}

	md.WriteString(fmt.Sprintf("*Last updated: %s*\n", policy.UpdatedAt.Format("2006-01-02")))

	return md.String()
}

// ToDocumentMarkdown creates a comprehensive policy document in markdown format
// This is the main method for generating policy documents that will be saved to files
func (pf *PolicyFormatter) ToDocumentMarkdown(policy *domain.Policy) string {
	var md strings.Builder

	// Use assigned reference ID or generate one from policy name
	var reference string
	if policy.ReferenceID != "" {
		reference = policy.ReferenceID
	} else {
		reference = pf.generateReference(policy.Name)
	}

	// Document title with reference
	md.WriteString(fmt.Sprintf("# %s - %s\n\n", reference, policy.Name))

	// Policy summary/description
	if policy.Summary != "" {
		md.WriteString("## Summary\n\n")
		md.WriteString(fmt.Sprintf("%s\n\n", pf.baseFormatter.CleanContent(policy.Summary)))
	} else if policy.Description != "" {
		md.WriteString("## Summary\n\n")
		md.WriteString(fmt.Sprintf("%s\n\n", pf.baseFormatter.CleanContent(policy.Description)))
	}

	// Main policy content
	if policy.Content != "" {
		md.WriteString("## Policy Document\n\n")
		content := pf.baseFormatter.CleanContent(policy.Content)
		md.WriteString(fmt.Sprintf("%s\n\n", content))
	}

	// Document metadata footer
	md.WriteString("---\n\n")
	md.WriteString("## Document Information\n\n")

	// Basic metadata table
	md.WriteString("| Field | Value |\n")
	md.WriteString("|-------| -------|\n")
	md.WriteString(fmt.Sprintf("| **Document Reference** | %s |\n", reference))
	md.WriteString(fmt.Sprintf("| **Policy ID** | %s |\n", policy.ID))
	md.WriteString(fmt.Sprintf("| **Policy Name** | %s |\n", policy.Name))
	if policy.Category != "" {
		md.WriteString(fmt.Sprintf("| **Category** | %s |\n", policy.Category))
	}
	if policy.MasterPolicyID != "" {
		md.WriteString(fmt.Sprintf("| **Master Policy ID** | %s |\n", policy.MasterPolicyID))
	}
	md.WriteString(fmt.Sprintf("| **Framework** | %s |\n", policy.Framework))
	md.WriteString(fmt.Sprintf("| **Status** | %s |\n", policy.Status))
	md.WriteString(fmt.Sprintf("| **Created Date** | %s |\n", policy.CreatedAt.Format("2006-01-02")))
	md.WriteString(fmt.Sprintf("| **Last Updated** | %s |\n", policy.UpdatedAt.Format("2006-01-02")))

	// Version information
	if policy.Version != "" {
		md.WriteString(fmt.Sprintf("| **Version** | %s |\n", policy.Version))
	}
	if policy.CurrentVersion != nil && policy.CurrentVersion.Version != "" {
		md.WriteString(fmt.Sprintf("| **Current Version** | %s |\n", policy.CurrentVersion.Version))
	}

	// Association information
	if policy.ControlCount > 0 {
		md.WriteString(fmt.Sprintf("| **Associated Controls** | %d |\n", policy.ControlCount))
	}
	if policy.ProcedureCount > 0 {
		md.WriteString(fmt.Sprintf("| **Associated Procedures** | %d |\n", policy.ProcedureCount))
	}
	if policy.EvidenceCount > 0 {
		md.WriteString(fmt.Sprintf("| **Associated Evidence Tasks** | %d |\n", policy.EvidenceCount))
	}
	if policy.RiskCount > 0 {
		md.WriteString(fmt.Sprintf("| **Associated Risk Items** | %d |\n", policy.RiskCount))
	}

	// Assignees and reviewers
	if len(policy.Assignees) > 0 {
		md.WriteString("\n### Assignees\n\n")
		for _, assignee := range policy.Assignees {
			if assignee.Name != "" && assignee.Email != "" {
				md.WriteString(fmt.Sprintf("- **%s** <%s>", assignee.Name, assignee.Email))
				if assignee.Role != "" {
					md.WriteString(fmt.Sprintf(" (%s)", assignee.Role))
				}
				if assignee.AssignedAt != nil && !assignee.AssignedAt.IsZero() {
					md.WriteString(fmt.Sprintf(" - Assigned: %s", assignee.AssignedAt.Format("2006-01-02")))
				}
				md.WriteString("\n")
			}
		}
	}

	if len(policy.Reviewers) > 0 {
		md.WriteString("\n### Reviewers\n\n")
		for _, reviewer := range policy.Reviewers {
			if reviewer.Name != "" && reviewer.Email != "" {
				md.WriteString(fmt.Sprintf("- **%s** <%s>", reviewer.Name, reviewer.Email))
				if reviewer.Role != "" {
					md.WriteString(fmt.Sprintf(" (%s)", reviewer.Role))
				}
				md.WriteString("\n")
			}
		}
	}

	// Tags
	if len(policy.Tags) > 0 {
		md.WriteString("\n### Tags\n\n")
		for _, tag := range policy.Tags {
			if tag.Name != "" {
				md.WriteString(fmt.Sprintf("- `%s`\n", tag.Name))
			}
		}
	}

	// Usage statistics (if available)
	if policy.ViewCount > 0 || policy.DownloadCount > 0 || policy.ReferenceCount > 0 {
		md.WriteString("\n### Usage Statistics\n\n")
		md.WriteString("| Metric | Value |\n")
		md.WriteString("|--------|-------|\n")
		if policy.ViewCount > 0 {
			md.WriteString(fmt.Sprintf("| **Views** | %d |\n", policy.ViewCount))
		}
		if policy.DownloadCount > 0 {
			md.WriteString(fmt.Sprintf("| **Downloads** | %d |\n", policy.DownloadCount))
		}
		if policy.ReferenceCount > 0 {
			md.WriteString(fmt.Sprintf("| **References** | %d |\n", policy.ReferenceCount))
		}
	}

	// Deprecation notice
	if policy.DeprecationNotes != "" {
		md.WriteString("\n### ⚠️ Deprecation Notice\n\n")
		md.WriteString(fmt.Sprintf("> %s\n", policy.DeprecationNotes))
	}

	// Document footer
	md.WriteString(pf.baseFormatter.GenerateDocumentFooter())

	return md.String()
}

// generateReference creates a filename-friendly reference from policy name
func (pf *PolicyFormatter) generateReference(name string) string {
	// Convert common policy names to standard references
	referenceMap := map[string]string{
		"access control":              "A1",
		"information security":        "A2",
		"data retention":              "A3",
		"data retention and disposal": "A3",
		"incident response":           "A4",
		"backup":                      "A5",
		"backup and recovery":         "A5",
		"business continuity":         "A6",
		"vendor management":           "A7",
		"third party":                 "A7",
		"risk management":             "A8",
		"change management":           "A9",
		"asset management":            "A10",
		"network security":            "A11",
		"encryption":                  "A12",
		"monitoring":                  "A13",
		"logging":                     "A13",
		"vulnerability management":    "A14",
		"security training":           "A15",
		"awareness":                   "A15",
		"acceptable use":              "A16",
		"privacy":                     "A17",
		"data privacy":                "A17",
		"physical security":           "A18",
		"hr security":                 "A19",
		"human resources":             "A19",
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
		// Use first letter of each word, up to 3 characters
		var initials strings.Builder
		for i, word := range words {
			if i >= 3 {
				break
			}
			if len(word) > 0 {
				initials.WriteString(strings.ToUpper(string(word[0])))
			}
		}
		return initials.String()
	}

	return "POL"
}

// GetDocumentFilename generates a filename for a policy document using unified pattern
func (pf *PolicyFormatter) GetDocumentFilename(policy *domain.Policy) string {
	fg := utils.NewFilenameGenerator()
	return fg.GenerateFilename(policy.ReferenceID, policy.ID, policy.Name, "md")
}
