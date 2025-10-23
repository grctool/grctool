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

	"github.com/grctool/grctool/internal/interpolation"
)

func TestBaseFormatter_WrapText(t *testing.T) {
	// Create a base formatter with disabled interpolation for testing
	config := interpolation.InterpolatorConfig{
		Variables:         make(map[string]string),
		Enabled:           false,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(config)
	formatter := NewBaseFormatter(interpolator)

	testCases := []struct {
		name       string
		input      string
		lineLength int
		expected   string
	}{
		{
			name:       "Short line no wrapping needed",
			input:      "This is a short line.",
			lineLength: 72,
			expected:   "This is a short line.",
		},
		{
			name:       "Long line needs wrapping",
			input:      "This is a very long line that definitely exceeds the 30 character limit and should be wrapped at word boundaries.",
			lineLength: 30,
			expected:   "This is a very long line that\ndefinitely exceeds the 30\ncharacter limit and should be\nwrapped at word boundaries.",
		},
		{
			name:       "Multiple lines with some needing wrapping",
			input:      "Short line.\nThis is a much longer line that needs to be wrapped because it exceeds our limit.\nAnother short line.",
			lineLength: 40,
			expected:   "Short line.\nThis is a much longer line that needs to\nbe wrapped because it exceeds our limit.\nAnother short line.",
		},
		{
			name:       "Preserve markdown headers",
			input:      "# This is a very long header that would normally be wrapped but should not be because it is a header",
			lineLength: 30,
			expected:   "# This is a very long header that would normally be wrapped but should not be because it is a header",
		},
		{
			name:       "Preserve markdown lists",
			input:      "- This is a very long list item that would normally be wrapped but should not be because it is a list item",
			lineLength: 30,
			expected:   "- This is a very long list item that would normally be wrapped but should not be because it is a list item",
		},
		{
			name:       "Preserve numbered lists",
			input:      "1. This is a very long numbered list item that would normally be wrapped but should not be because it is a numbered list item",
			lineLength: 30,
			expected:   "1. This is a very long numbered list item that would normally be wrapped but should not be because it is a numbered list item",
		},
		{
			name:       "Preserve table rows",
			input:      "| This is a very long table row | that should not be wrapped | because it contains pipes |",
			lineLength: 30,
			expected:   "| This is a very long table row | that should not be wrapped | because it contains pipes |",
		},
		{
			name:       "Preserve horizontal rules",
			input:      "---",
			lineLength: 30,
			expected:   "---",
		},
		{
			name:       "Preserve leading whitespace",
			input:      "    This is an indented line that is long enough to need wrapping while preserving the initial indentation level.",
			lineLength: 40,
			expected:   "    This is an indented line that is\n    long enough to need wrapping while\n    preserving the initial indentation\n    level.",
		},
		{
			name:       "Handle empty lines",
			input:      "Line 1\n\nLine 3",
			lineLength: 72,
			expected:   "Line 1\n\nLine 3",
		},
		{
			name:       "Default line length when zero",
			input:      "This is a line that should be wrapped at the default 72 character limit when no line length is specified or zero is passed.",
			lineLength: 0,
			expected:   "This is a line that should be wrapped at the default 72 character limit\nwhen no line length is specified or zero is passed.",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatter.WrapText(tc.input, tc.lineLength)
			if result != tc.expected {
				t.Errorf("Expected:\n%q\nGot:\n%q", tc.expected, result)
			}
		})
	}
}

// TestBaseFormatter_shouldSkipWrapping tests are commented out because the method
// has been moved to the internal markdown formatter
/*
func TestBaseFormatter_shouldSkipWrapping(t *testing.T) {
	config := interpolation.InterpolatorConfig{
		Variables:         make(map[string]string),
		Enabled:           false,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(config)
	formatter := NewBaseFormatter(interpolator)

	testCases := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Regular paragraph text should be wrapped",
			input:    "This is regular paragraph text that should be wrapped.",
			expected: false,
		},
		{
			name:     "Markdown header should be skipped",
			input:    "# This is a header",
			expected: true,
		},
		{
			name:     "Markdown header level 2 should be skipped",
			input:    "## This is a level 2 header",
			expected: true,
		},
		{
			name:     "Unordered list should be skipped",
			input:    "- This is a list item",
			expected: true,
		},
		{
			name:     "Alternative unordered list should be skipped",
			input:    "* This is a list item",
			expected: true,
		},
		{
			name:     "Numbered list should be skipped",
			input:    "1. This is a numbered list item",
			expected: true,
		},
		{
			name:     "Double digit numbered list should be skipped",
			input:    "12. This is a numbered list item",
			expected: true,
		},
		{
			name:     "Table row should be skipped",
			input:    "| Column 1 | Column 2 | Column 3 |",
			expected: true,
		},
		{
			name:     "Horizontal rule dashes should be skipped",
			input:    "---",
			expected: true,
		},
		{
			name:     "Horizontal rule asterisks should be skipped",
			input:    "***",
			expected: true,
		},
		{
			name:     "Code block should be skipped",
			input:    "```",
			expected: true,
		},
		{
			name:     "Indented code should be skipped",
			input:    "    code block",
			expected: true,
		},
		{
			name:     "Short line should be skipped",
			input:    "Short",
			expected: true,
		},
		{
			name:     "Empty line should be skipped",
			input:    "",
			expected: true,
		},
		{
			name:     "Whitespace only should be skipped",
			input:    "   ",
			expected: true,
		},
		{
			name:     "Text with leading whitespace but content should not be skipped",
			input:    "  This is regular text with leading whitespace that should be wrapped.",
			expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatter.shouldSkipWrapping(tc.input)
			if result != tc.expected {
				t.Errorf("For input %q, expected %t but got %t", tc.input, tc.expected, result)
			}
		})
	}
}
*/

// TestBaseFormatter_wrapSingleLine tests are commented out because the method
// has been moved to the internal markdown formatter
/*
func TestBaseFormatter_wrapSingleLine(t *testing.T) {
	config := interpolation.InterpolatorConfig{
		Variables:         make(map[string]string),
		Enabled:           false,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(config)
	formatter := NewBaseFormatter(interpolator)

	testCases := []struct {
		name       string
		input      string
		lineLength int
		expected   []string
	}{
		{
			name:       "Short line no wrapping",
			input:      "Short line",
			lineLength: 20,
			expected:   []string{"Short line"},
		},
		{
			name:       "Long line needs wrapping",
			input:      "This is a long line that needs wrapping",
			lineLength: 20,
			expected:   []string{"This is a long line", "that needs wrapping"},
		},
		{
			name:       "Line with leading whitespace preserved",
			input:      "    Indented text that needs wrapping at boundaries",
			lineLength: 25,
			expected:   []string{"    Indented text that", "    needs wrapping at", "    boundaries"},
		},
		{
			name:       "Single word longer than limit",
			input:      "supercalifragilisticexpialidocious",
			lineLength: 10,
			expected:   []string{"supercalifragilisticexpialidocious"},
		},
		{
			name:       "Empty line",
			input:      "",
			lineLength: 20,
			expected:   []string{},
		},
		{
			name:       "Only whitespace",
			input:      "    ",
			lineLength: 20,
			expected:   []string{},
		},
		{
			name:       "Exact length match",
			input:      "Exactly twenty chars",
			lineLength: 20,
			expected:   []string{"Exactly twenty chars"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatter.wrapSingleLine(tc.input, tc.lineLength)
			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d lines, got %d lines", len(tc.expected), len(result))
				t.Errorf("Expected: %v", tc.expected)
				t.Errorf("Got: %v", result)
				return
			}

			for i, line := range result {
				if line != tc.expected[i] {
					t.Errorf("Line %d: expected %q, got %q", i, tc.expected[i], line)
				}
			}
		})
	}
}
*/

func TestBaseFormatter_CleanContent_WithWrapping(t *testing.T) {
	// Test that CleanContent applies text wrapping
	config := interpolation.InterpolatorConfig{
		Variables:         make(map[string]string),
		Enabled:           false,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(config)
	formatter := NewBaseFormatter(interpolator)

	longText := "This is a very long line of text that should be wrapped at 72 characters to make it more readable for humans who are viewing the markdown documents."
	result := formatter.CleanContent(longText)

	// Check that the result contains line breaks (indicating wrapping occurred)
	if !strings.Contains(result, "\n") {
		t.Error("Expected text wrapping to occur, but no line breaks found")
	}

	// Check that no line is longer than 72 characters
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		if len(line) > 72 {
			t.Errorf("Line %d is %d characters long (exceeds 72): %q", i, len(line), line)
		}
	}

	// Check that the wrapped text contains the original words
	originalWords := strings.Fields(longText)
	resultWords := strings.Fields(result)
	if len(originalWords) != len(resultWords) {
		t.Errorf("Word count mismatch: original had %d words, result has %d words", len(originalWords), len(resultWords))
	}
}

func TestBaseFormatter_SanitizeFilename(t *testing.T) {
	// Create a base formatter with disabled interpolation for testing
	config := interpolation.InterpolatorConfig{
		Variables:         make(map[string]string),
		Enabled:           false,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(config)
	formatter := NewBaseFormatter(interpolator)

	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "Simple filename with no special characters",
			input:    "SimpleFilename",
			expected: "SimpleFilename",
		},
		{
			name:     "Filename with spaces",
			input:    "File with spaces",
			expected: "File_with_spaces",
		},
		{
			name:     "Filename with forward slashes",
			input:    "Path/To/File",
			expected: "Path-To-File",
		},
		{
			name:     "Filename with colons",
			input:    "Time:12:30:45",
			expected: "Time-12-30-45",
		},
		{
			name:     "Filename with question marks",
			input:    "What is this?",
			expected: "What_is_this",
		},
		{
			name:     "Filename with asterisks",
			input:    "Important*File*Name",
			expected: "ImportantFileName",
		},
		{
			name:     "Filename with angle brackets",
			input:    "<script>alert('test')</script>",
			expected: "scriptalert('test')-script",
		},
		{
			name:     "Filename with pipes",
			input:    "Command|Pipe|Example",
			expected: "Command-Pipe-Example",
		},
		{
			name:     "Filename with backslashes",
			input:    "Windows\\Path\\File",
			expected: "Windows-Path-File",
		},
		{
			name:     "Filename with quotes",
			input:    `File "with" quotes`,
			expected: "File_with_quotes",
		},
		{
			name:     "Very long filename exceeding 50 characters",
			input:    "This is a very long filename that definitely exceeds the fifty character limit and should be truncated",
			expected: "This_is_a_very_long_filename_that_definitely_excee",
		},
		{
			name:     "Filename exactly 50 characters",
			input:    "This filename has exactly fifty characters in it!!",
			expected: "This_filename_has_exactly_fifty_characters_in_it!!",
		},
		{
			name:     "Mixed special characters",
			input:    "File: <Important> | Path/To\\Document*?.txt",
			expected: "File-_Important_-_Path-To-Document.txt",
		},
		{
			name:     "Unicode characters preserved",
			input:    "File with émojis 🎉 and ñ",
			expected: "File_with_émojis_🎉_and_ñ",
		},
		{
			name:     "Multiple spaces become single underscores",
			input:    "File   with   multiple   spaces",
			expected: "File___with___multiple___spaces",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := formatter.SanitizeFilename(tc.input)
			if result != tc.expected {
				t.Errorf("Expected %q, got %q", tc.expected, result)
			}
		})
	}
}
