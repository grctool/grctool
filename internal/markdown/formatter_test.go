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
	"strings"
	"testing"
)

func TestNewFormatter(t *testing.T) {
	tests := []struct {
		name   string
		config Config
		want   Config
	}{
		{
			name:   "default config",
			config: Config{},
			want: Config{
				LineLength:         72,
				PreserveLineBreaks: false,
				IndentSize:         2,
			},
		},
		{
			name: "custom config",
			config: Config{
				LineLength:         80,
				PreserveLineBreaks: true,
				IndentSize:         4,
			},
			want: Config{
				LineLength:         80,
				PreserveLineBreaks: true,
				IndentSize:         4,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := NewFormatter(tt.config)
			if f.config.LineLength != tt.want.LineLength {
				t.Errorf("LineLength = %v, want %v", f.config.LineLength, tt.want.LineLength)
			}
			if f.config.PreserveLineBreaks != tt.want.PreserveLineBreaks {
				t.Errorf("PreserveLineBreaks = %v, want %v", f.config.PreserveLineBreaks, tt.want.PreserveLineBreaks)
			}
			if f.config.IndentSize != tt.want.IndentSize {
				t.Errorf("IndentSize = %v, want %v", f.config.IndentSize, tt.want.IndentSize)
			}
		})
	}
}

func TestFormatter_WrapText(t *testing.T) {
	f := NewFormatter(DefaultConfig())

	tests := []struct {
		name       string
		input      string
		lineLength int
		want       string
	}{
		{
			name:       "short line",
			input:      "This is a short line.",
			lineLength: 72,
			want:       "This is a short line.",
		},
		{
			name:       "long line wrapping",
			input:      "This is a very long line that should definitely be wrapped because it exceeds the maximum line length of 72 characters.",
			lineLength: 72,
			want:       "This is a very long line that should definitely be wrapped because it\nexceeds the maximum line length of 72 characters.",
		},
		{
			name:       "preserve headers",
			input:      "# This is a very long header that should not be wrapped even though it exceeds 72 characters",
			lineLength: 72,
			want:       "# This is a very long header that should not be wrapped even though it exceeds 72 characters",
		},
		{
			name:       "preserve lists",
			input:      "- This is a very long list item that should not be wrapped even though it exceeds the limit",
			lineLength: 72,
			want:       "- This is a very long list item that should not be wrapped even though it exceeds the limit",
		},
		{
			name:       "preserve code blocks",
			input:      "```\nThis is a very long line inside a code block that should not be wrapped no matter how long it is\n```",
			lineLength: 72,
			want:       "```\nThis is a very long line inside a code block that should not be wrapped no matter how long it is\n```",
		},
		{
			name:       "multiple paragraphs",
			input:      "First paragraph that is quite long and should be wrapped at the appropriate boundary.\n\nSecond paragraph that is also long and should be wrapped independently from the first.",
			lineLength: 72,
			want:       "First paragraph that is quite long and should be wrapped at the\nappropriate boundary.\n\nSecond paragraph that is also long and should be wrapped independently\nfrom the first.",
		},
		{
			name:       "preserve indentation",
			input:      "    This is an indented line that is very long and should be wrapped while preserving the initial indentation.",
			lineLength: 72,
			want:       "    This is an indented line that is very long and should be wrapped\n    while preserving the initial indentation.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.WrapText(tt.input, tt.lineLength)
			if got != tt.want {
				t.Errorf("WrapText() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatter_CleanContent(t *testing.T) {
	f := NewFormatter(DefaultConfig())

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "remove trailing spaces",
			input: "Line with trailing spaces   \nAnother line   ",
			want:  "Line with trailing spaces\nAnother line",
		},
		{
			name:  "reduce multiple blank lines",
			input: "First line\n\n\n\nSecond line",
			want:  "First line\n\nSecond line",
		},
		{
			name:  "preserve code blocks",
			input: "```\nCode with spaces   \n   More code\n```",
			want:  "```\nCode with spaces   \n   More code\n```",
		},
		{
			name:  "remove trailing newlines",
			input: "Content\n\n\n",
			want:  "Content",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := f.CleanContent(tt.input)
			if got != tt.want {
				t.Errorf("CleanContent() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestFormatter_Format(t *testing.T) {
	f := NewFormatter(Config{LineLength: 50}) // Shorter for testing

	input := `# Test Document

This is a paragraph with some trailing spaces   
that should be cleaned up and wrapped properly at the fifty character boundary.

## Section Two

- List item that should not be wrapped
- Another list item

Here's a code block:
` + "```" + `
This code should not be wrapped or modified   
` + "```" + `

Final paragraph that needs wrapping because it's quite long and exceeds our limit.`

	want := `# Test Document

This is a paragraph with some trailing spaces
that should be cleaned up and wrapped properly at
the fifty character boundary.

## Section Two

- List item that should not be wrapped
- Another list item

Here's a code block:
` + "```" + `
This code should not be wrapped or modified   
` + "```" + `

Final paragraph that needs wrapping because it's
quite long and exceeds our limit.`

	got := f.Format(input)
	if got != want {
		t.Errorf("Format() mismatch\nGot:\n%s\n\nWant:\n%s", got, want)
	}
}

func TestFormatter_FormatPrompt(t *testing.T) {
	f := NewFormatter(DefaultConfig())

	prompt := `You are an expert security auditor working on assembling evidence to support a SOC2 Type II audit. Use the following context and the tools at your disposal to assemble evidence showing that the controls in question have been implemented.

**Evidence Task**: Access Control Registration and De-registration Process Document
**Description**: Documented process for granting and revoking access to systems when employees join or leave the organization.`

	result := f.FormatPrompt(prompt)

	// Check that the prompt is wrapped at 72 characters
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		if len(line) > 72 && !strings.HasPrefix(line, "**") {
			t.Errorf("Line %d exceeds 72 characters: %d chars: %s", i+1, len(line), line)
		}
	}

	// Check that markdown formatting is preserved
	if !strings.Contains(result, "**Evidence Task**:") {
		t.Error("Markdown bold formatting was not preserved")
	}
}
