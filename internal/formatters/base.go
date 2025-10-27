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

	"github.com/grctool/grctool/internal/interpolation"
	"github.com/grctool/grctool/internal/markdown"
)

// BaseFormatter provides shared formatting utilities for document formatters
type BaseFormatter struct {
	interpolator interpolation.Interpolator
	mdFormatter  *markdown.Formatter
}

// NewBaseFormatter creates a new base formatter with the given interpolator
func NewBaseFormatter(interpolator interpolation.Interpolator) *BaseFormatter {
	return &BaseFormatter{
		interpolator: interpolator,
		mdFormatter:  markdown.NewFormatter(markdown.DefaultConfig()),
	}
}

// InterpolateText applies variable interpolation to simple text fields
func (bf *BaseFormatter) InterpolateText(text string) string {
	if interpolated, err := bf.interpolator.Interpolate(text); err == nil {
		return interpolated
	}
	// If interpolation fails, return original text
	return text
}

// CleanContent cleans up content for better markdown rendering with interpolation
func (bf *BaseFormatter) CleanContent(content string) string {
	// Apply interpolation first to any content that might not go through convertHTMLToMarkdown
	if interpolated, err := bf.interpolator.Interpolate(content); err == nil {
		content = interpolated
	}

	// Convert HTML to markdown
	markdown := bf.ConvertHTMLToMarkdown(content)

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

	// Apply text wrapping using the markdown formatter
	cleaned = bf.mdFormatter.WrapText(cleaned, 72)

	return cleaned
}

// ConvertHTMLToMarkdown converts HTML content to markdown format
func (bf *BaseFormatter) ConvertHTMLToMarkdown(content string) string {
	// Apply variable interpolation first (this replaces the hardcoded substitution)
	if interpolated, err := bf.interpolator.Interpolate(content); err == nil {
		content = interpolated
	}
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

	// Convert tables
	content = bf.convertHTMLTables(content)

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

// convertHTMLTables converts HTML tables to markdown tables
func (bf *BaseFormatter) convertHTMLTables(content string) string {
	// Simple table conversion - this is a basic implementation
	// For complex tables, consider using a proper HTML parser

	// Match table structure
	tableRegex := regexp.MustCompile(`(?i)<table[^>]*>(.*?)</table>`)
	tables := tableRegex.FindAllStringSubmatch(content, -1)

	for _, table := range tables {
		tableContent := table[1]
		markdownTable := bf.convertSingleTable(tableContent)
		content = strings.Replace(content, table[0], markdownTable, 1)
	}

	return content
}

// convertSingleTable converts a single HTML table to markdown
func (bf *BaseFormatter) convertSingleTable(tableHTML string) string {
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

// SanitizeFilename cleans a filename to be safe for filesystem use
func (bf *BaseFormatter) SanitizeFilename(name string) string {
	cleanName := strings.ReplaceAll(name, "/", "-")
	cleanName = strings.ReplaceAll(cleanName, ":", "-")
	cleanName = strings.ReplaceAll(cleanName, "?", "")
	cleanName = strings.ReplaceAll(cleanName, "*", "")
	cleanName = strings.ReplaceAll(cleanName, "<", "")
	cleanName = strings.ReplaceAll(cleanName, ">", "")
	cleanName = strings.ReplaceAll(cleanName, "|", "-")
	cleanName = strings.ReplaceAll(cleanName, "\\", "-")
	cleanName = strings.ReplaceAll(cleanName, "\"", "")
	// Replace spaces with underscores for better readability
	cleanName = strings.ReplaceAll(cleanName, " ", "_")
	// Limit length to prevent overly long filenames
	if len(cleanName) > 50 {
		cleanName = cleanName[:50]
	}
	return cleanName
}

// GenerateDocumentFooter creates a standard document footer
// Returns empty string to avoid git noise from timestamps
func (bf *BaseFormatter) GenerateDocumentFooter() string {
	return ""
}

// FormatMetadataTable creates a markdown table for metadata
func (bf *BaseFormatter) FormatMetadataTable(rows []MetadataRow) string {
	var md strings.Builder

	md.WriteString("| Field | Value |\n")
	md.WriteString("|-------|-------|\n")

	for _, row := range rows {
		if row.Value != "" {
			md.WriteString(fmt.Sprintf("| **%s** | %s |\n", row.Field, row.Value))
		}
	}

	return md.String()
}

// MetadataRow represents a row in a metadata table
type MetadataRow struct {
	Field string
	Value string
}

// FormatPersonList creates a markdown list for people (assignees, reviewers, etc.)
func (bf *BaseFormatter) FormatPersonList(people []PersonInfo, title string) string {
	if len(people) == 0 {
		return ""
	}

	var md strings.Builder
	md.WriteString(fmt.Sprintf("\n### %s\n\n", title))

	for _, person := range people {
		if person.Name != "" || person.Email != "" {
			md.WriteString(fmt.Sprintf("- **%s**", person.Name))
			if person.Email != "" {
				md.WriteString(fmt.Sprintf(" <%s>", person.Email))
			}
			if person.Role != "" {
				md.WriteString(fmt.Sprintf(" (%s)", person.Role))
			}
			if !person.AssignedAt.IsZero() {
				md.WriteString(fmt.Sprintf(" - Assigned: %s", person.AssignedAt.Format("2006-01-02")))
			}
			md.WriteString("\n")
		}
	}

	return md.String()
}

// PersonInfo represents a person (assignee, reviewer, etc.)
type PersonInfo struct {
	Name       string
	Email      string
	Role       string
	AssignedAt time.Time
}

// FormatTagList creates a markdown list for tags
func (bf *BaseFormatter) FormatTagList(tags []TagInfo) string {
	if len(tags) == 0 {
		return ""
	}

	var md strings.Builder
	md.WriteString("\n### Tags\n\n")

	for _, tag := range tags {
		if tag.Name != "" {
			if tag.Color != "" {
				md.WriteString(fmt.Sprintf("- `%s` <span style=\"color: %s\">●</span>\n", tag.Name, tag.Color))
			} else {
				md.WriteString(fmt.Sprintf("- `%s`\n", tag.Name))
			}
		}
	}

	return md.String()
}

// TagInfo represents a tag
type TagInfo struct {
	Name  string
	Color string
}

// FormatUsageStatistics creates a usage statistics table
func (bf *BaseFormatter) FormatUsageStatistics(stats UsageStats) string {
	if stats.ViewCount == 0 && stats.DownloadCount == 0 && stats.ReferenceCount == 0 {
		return ""
	}

	var md strings.Builder
	md.WriteString("\n### Usage Statistics\n\n")
	md.WriteString("| Metric | Value |\n")
	md.WriteString("|--------|-------|\n")

	if stats.ViewCount > 0 {
		md.WriteString(fmt.Sprintf("| **Views** | %d |\n", stats.ViewCount))
		if !stats.LastViewedAt.IsZero() {
			md.WriteString(fmt.Sprintf("| **Last Viewed** | %s |\n", stats.LastViewedAt.Format("2006-01-02 15:04")))
		}
	}
	if stats.DownloadCount > 0 {
		md.WriteString(fmt.Sprintf("| **Downloads** | %d |\n", stats.DownloadCount))
		if !stats.LastDownloadedAt.IsZero() {
			md.WriteString(fmt.Sprintf("| **Last Downloaded** | %s |\n", stats.LastDownloadedAt.Format("2006-01-02 15:04")))
		}
	}
	if stats.ReferenceCount > 0 {
		md.WriteString(fmt.Sprintf("| **References** | %d |\n", stats.ReferenceCount))
		if !stats.LastReferencedAt.IsZero() {
			md.WriteString(fmt.Sprintf("| **Last Referenced** | %s |\n", stats.LastReferencedAt.Format("2006-01-02 15:04")))
		}
	}

	return md.String()
}

// WrapText wraps text at the specified line length on word boundaries
// Now delegates to the markdown formatter for consistency
func (bf *BaseFormatter) WrapText(text string, lineLength int) string {
	return bf.mdFormatter.WrapText(text, lineLength)
}

// UsageStats represents usage statistics
type UsageStats struct {
	ViewCount        int
	LastViewedAt     time.Time
	DownloadCount    int
	LastDownloadedAt time.Time
	ReferenceCount   int
	LastReferencedAt time.Time
}
