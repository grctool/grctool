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

package markdown

import (
	"regexp"
	"strings"
)

// Formatter provides consistent markdown formatting across the application
type Formatter struct {
	config Config
}

// Config holds configuration for the markdown formatter
type Config struct {
	// LineLength is the maximum line length for wrapped text (default: 72)
	LineLength int
	// PreserveLineBreaks controls whether to preserve existing line breaks
	PreserveLineBreaks bool
	// IndentSize for nested content (default: 2 spaces)
	IndentSize int
}

// DefaultConfig returns the default formatter configuration
func DefaultConfig() Config {
	return Config{
		LineLength:         72,
		PreserveLineBreaks: true,
		IndentSize:         2,
	}
}

// NewFormatter creates a new markdown formatter with the given configuration
func NewFormatter(config Config) *Formatter {
	// Apply defaults for zero values
	if config.LineLength <= 0 {
		config.LineLength = 72
	}
	if config.IndentSize <= 0 {
		config.IndentSize = 2
	}

	return &Formatter{
		config: config,
	}
}

// Format applies consistent formatting to markdown content
func (f *Formatter) Format(content string) string {
	// Clean up the content first
	cleaned := f.CleanContent(content)

	// Apply text wrapping
	wrapped := f.WrapText(cleaned, f.config.LineLength)

	return wrapped
}

// FormatWithOptions allows custom formatting options for specific content
func (f *Formatter) FormatWithOptions(content string, opts FormatOptions) string {
	// Override config with options if provided
	lineLength := f.config.LineLength
	if opts.LineLength > 0 {
		lineLength = opts.LineLength
	}

	cleaned := content
	if !opts.SkipCleaning {
		cleaned = f.CleanContent(content)
	}

	if opts.NoWrapping {
		return cleaned
	}

	return f.WrapText(cleaned, lineLength)
}

// FormatOptions provides options for customizing formatting behavior
type FormatOptions struct {
	// LineLength overrides the default line length
	LineLength int
	// NoWrapping disables text wrapping entirely
	NoWrapping bool
	// SkipCleaning skips the initial content cleaning step
	SkipCleaning bool
}

// CleanContent cleans up markdown content while preserving structure
func (f *Formatter) CleanContent(content string) string {
	lines := strings.Split(content, "\n")
	var cleanLines []string

	inCodeBlock := false
	for _, line := range lines {
		// Check for code block boundaries
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			cleanLines = append(cleanLines, line)
			continue
		}

		// Preserve code blocks as-is
		if inCodeBlock {
			cleanLines = append(cleanLines, line)
			continue
		}

		// Trim trailing spaces but preserve intentional indentation
		trimmed := strings.TrimRight(line, " \t")
		cleanLines = append(cleanLines, trimmed)
	}

	cleaned := strings.Join(cleanLines, "\n")

	// Ensure there's not excessive blank lines (need to do this repeatedly)
	for strings.Contains(cleaned, "\n\n\n") {
		cleaned = strings.ReplaceAll(cleaned, "\n\n\n", "\n\n")
	}

	// Remove trailing newlines but preserve content structure
	cleaned = strings.TrimRight(cleaned, "\n")

	return cleaned
}

// WrapText wraps text at the specified line length on word boundaries
func (f *Formatter) WrapText(text string, lineLength int) string {
	if lineLength <= 0 {
		lineLength = f.config.LineLength
	}

	lines := strings.Split(text, "\n")
	var wrappedLines []string

	inCodeBlock := false
	for _, line := range lines {
		// Track code blocks
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inCodeBlock = !inCodeBlock
			wrappedLines = append(wrappedLines, line)
			continue
		}

		// Don't wrap content inside code blocks
		if inCodeBlock {
			wrappedLines = append(wrappedLines, line)
			continue
		}

		// Don't wrap markdown headers, lists, or tables
		if f.shouldSkipWrapping(line) {
			wrappedLines = append(wrappedLines, line)
			continue
		}

		// Wrap the line if it's longer than the limit
		if len(line) <= lineLength {
			wrappedLines = append(wrappedLines, line)
		} else {
			wrapped := f.wrapSingleLine(line, lineLength)
			wrappedLines = append(wrappedLines, wrapped...)
		}
	}

	return strings.Join(wrappedLines, "\n")
}

// shouldSkipWrapping determines if a line should be skipped from wrapping
func (f *Formatter) shouldSkipWrapping(line string) bool {
	trimmed := strings.TrimSpace(line)

	// Skip markdown headers
	if strings.HasPrefix(trimmed, "#") {
		return true
	}

	// Skip markdown lists
	if strings.HasPrefix(trimmed, "- ") || strings.HasPrefix(trimmed, "* ") {
		return true
	}

	// Skip numbered lists
	if matched, _ := regexp.MatchString(`^\d+\.\s`, trimmed); matched {
		return true
	}

	// Skip table rows
	if strings.Contains(trimmed, "|") {
		return true
	}

	// Skip markdown horizontal rules
	if strings.HasPrefix(trimmed, "---") || strings.HasPrefix(trimmed, "***") {
		return true
	}

	// Skip blockquotes
	if strings.HasPrefix(trimmed, ">") {
		return true
	}

	// Skip lines that are only whitespace or very short
	if len(trimmed) <= 10 {
		return true
	}

	return false
}

// wrapSingleLine wraps a single line at word boundaries
func (f *Formatter) wrapSingleLine(line string, lineLength int) []string {
	// Preserve leading whitespace
	leadingWhitespace := ""
	trimmed := strings.TrimLeft(line, " \t")
	if len(line) > len(trimmed) {
		leadingWhitespace = line[:len(line)-len(trimmed)]
	}

	words := strings.Fields(trimmed)
	if len(words) == 0 {
		// Return empty slice for lines with no content
		if strings.TrimSpace(line) == "" {
			return []string{}
		}
		return []string{line}
	}

	var wrappedLines []string
	currentLine := leadingWhitespace

	for i, word := range words {
		// If adding this word would exceed the line length, start a new line
		testLine := currentLine
		if i > 0 || len(currentLine) > len(leadingWhitespace) {
			testLine += " "
		}
		testLine += word

		if len(testLine) > lineLength && len(currentLine) > len(leadingWhitespace) {
			// Start a new line with the same leading whitespace
			wrappedLines = append(wrappedLines, currentLine)
			currentLine = leadingWhitespace + word
		} else {
			if i > 0 || len(currentLine) > len(leadingWhitespace) {
				currentLine += " "
			}
			currentLine += word
		}
	}

	// Add the final line
	if len(currentLine) > len(leadingWhitespace) {
		wrappedLines = append(wrappedLines, currentLine)
	}

	return wrappedLines
}

// FormatDocument formats a complete markdown document
func (f *Formatter) FormatDocument(content string) string {
	return f.Format(content)
}

// FormatPrompt formats AI prompts while preserving their structure
func (f *Formatter) FormatPrompt(prompt string) string {
	// Prompts often have specific formatting that should be preserved
	// but we still want consistent line wrapping
	return f.FormatWithOptions(prompt, FormatOptions{
		SkipCleaning: false,
		NoWrapping:   false,
	})
}

// FormatTable formats markdown tables (currently a no-op as tables shouldn't be wrapped)
func (f *Formatter) FormatTable(table string) string {
	return table
}

// FormatCodeBlock formats code blocks (currently a no-op as code shouldn't be wrapped)
func (f *Formatter) FormatCodeBlock(code string) string {
	return code
}
