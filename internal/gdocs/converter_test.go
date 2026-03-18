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

package gdocs

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMarkdownToDoc_Headings(t *testing.T) {
	for level := 1; level <= 6; level++ {
		prefix := strings.Repeat("#", level)
		md := prefix + " Heading Level " + string(rune('0'+level))
		doc, err := MarkdownToDoc(md)
		require.NoError(t, err)
		require.Len(t, doc.Elements, 1)
		assert.Equal(t, ElementHeading, doc.Elements[0].Type)
		assert.Equal(t, level, doc.Elements[0].Level)
	}
}

func TestMarkdownToDoc_Paragraphs(t *testing.T) {
	md := "First paragraph.\n\nSecond paragraph."
	doc, err := MarkdownToDoc(md)
	require.NoError(t, err)
	require.Len(t, doc.Elements, 2)
	assert.Equal(t, ElementParagraph, doc.Elements[0].Type)
	assert.Equal(t, ElementParagraph, doc.Elements[1].Type)
}

func TestMarkdownToDoc_Bold(t *testing.T) {
	md := "This is **bold** text."
	doc, err := MarkdownToDoc(md)
	require.NoError(t, err)
	require.Len(t, doc.Elements, 1)

	runs := doc.Elements[0].Items
	// Find the bold run
	found := false
	for _, r := range runs {
		if r.Style != nil && r.Style.Bold {
			assert.Equal(t, "bold", r.Content)
			found = true
		}
	}
	assert.True(t, found, "expected a bold run")
}

func TestMarkdownToDoc_Italic(t *testing.T) {
	md := "This is *italic* text."
	doc, err := MarkdownToDoc(md)
	require.NoError(t, err)
	require.Len(t, doc.Elements, 1)

	found := false
	for _, r := range doc.Elements[0].Items {
		if r.Style != nil && r.Style.Italic {
			assert.Equal(t, "italic", r.Content)
			found = true
		}
	}
	assert.True(t, found, "expected an italic run")
}

func TestMarkdownToDoc_Code(t *testing.T) {
	md := "Use `grctool sync` command."
	doc, err := MarkdownToDoc(md)
	require.NoError(t, err)
	require.Len(t, doc.Elements, 1)

	found := false
	for _, r := range doc.Elements[0].Items {
		if r.Type == ElementCode || (r.Style != nil && r.Style.Code) {
			assert.Equal(t, "grctool sync", r.Content)
			found = true
		}
	}
	assert.True(t, found, "expected an inline code run")
}

func TestMarkdownToDoc_Links(t *testing.T) {
	md := "Visit [GRCTool](https://grctool.dev) for docs."
	doc, err := MarkdownToDoc(md)
	require.NoError(t, err)
	require.Len(t, doc.Elements, 1)

	found := false
	for _, r := range doc.Elements[0].Items {
		if r.Type == ElementLink {
			assert.Equal(t, "GRCTool", r.Content)
			assert.Equal(t, "https://grctool.dev", r.URL)
			found = true
		}
	}
	assert.True(t, found, "expected a link run")
}

func TestMarkdownToDoc_BulletList(t *testing.T) {
	md := "- Item one\n- Item two\n- Item three"
	doc, err := MarkdownToDoc(md)
	require.NoError(t, err)
	require.Len(t, doc.Elements, 1)
	assert.Equal(t, ElementList, doc.Elements[0].Type)
	assert.Equal(t, "bullet", doc.Elements[0].ListType)
	assert.Len(t, doc.Elements[0].Items, 3)
}

func TestMarkdownToDoc_OrderedList(t *testing.T) {
	md := "1. First\n2. Second\n3. Third"
	doc, err := MarkdownToDoc(md)
	require.NoError(t, err)
	require.Len(t, doc.Elements, 1)
	assert.Equal(t, ElementList, doc.Elements[0].Type)
	assert.Equal(t, "ordered", doc.Elements[0].ListType)
	assert.Len(t, doc.Elements[0].Items, 3)
}

func TestMarkdownToDoc_CodeBlock(t *testing.T) {
	md := "```go\nfunc main() {\n\tfmt.Println(\"hello\")\n}\n```"
	doc, err := MarkdownToDoc(md)
	require.NoError(t, err)
	require.Len(t, doc.Elements, 1)
	assert.Equal(t, ElementCodeBlock, doc.Elements[0].Type)
	assert.Equal(t, "go", doc.Elements[0].Language)
	assert.Contains(t, doc.Elements[0].Content, "func main()")
}

func TestMarkdownToDoc_Blockquote(t *testing.T) {
	md := "> This is a quoted block."
	doc, err := MarkdownToDoc(md)
	require.NoError(t, err)
	require.Len(t, doc.Elements, 1)
	assert.Equal(t, ElementBlockquote, doc.Elements[0].Type)
	require.NotEmpty(t, doc.Elements[0].Items)
}

func TestMarkdownToDoc_Table(t *testing.T) {
	md := "| Name | Role |\n| --- | --- |\n| Alice | Admin |\n| Bob | User |"
	doc, err := MarkdownToDoc(md)
	require.NoError(t, err)
	require.Len(t, doc.Elements, 1)
	assert.Equal(t, ElementTable, doc.Elements[0].Type)
	// Should have header + 2 data rows (separator is not a data row in goldmark)
	require.GreaterOrEqual(t, len(doc.Elements[0].Rows), 2)
}

func TestMarkdownToDoc_ThematicBreak(t *testing.T) {
	md := "Above\n\n---\n\nBelow"
	doc, err := MarkdownToDoc(md)
	require.NoError(t, err)
	// Should have paragraph, thematic break, paragraph
	found := false
	for _, elem := range doc.Elements {
		if elem.Type == ElementThematicBreak {
			found = true
		}
	}
	assert.True(t, found, "expected a thematic break element")
}

func TestDocToMarkdown_Headings(t *testing.T) {
	doc := &Document{
		Elements: []DocsElement{
			{Type: ElementHeading, Level: 1, Items: []DocsElement{{Type: ElementParagraph, Content: "Title"}}},
			{Type: ElementHeading, Level: 2, Items: []DocsElement{{Type: ElementParagraph, Content: "Section"}}},
			{Type: ElementHeading, Level: 3, Items: []DocsElement{{Type: ElementParagraph, Content: "Subsection"}}},
		},
	}

	md, err := DocToMarkdown(doc)
	require.NoError(t, err)
	assert.Contains(t, md, "# Title")
	assert.Contains(t, md, "## Section")
	assert.Contains(t, md, "### Subsection")
}

func TestDocToMarkdown_FullDocument(t *testing.T) {
	doc := &Document{
		Title: "Test Document",
		Elements: []DocsElement{
			{Type: ElementHeading, Level: 1, Items: []DocsElement{{Type: ElementParagraph, Content: "Introduction"}}},
			{Type: ElementParagraph, Items: []DocsElement{
				{Type: ElementParagraph, Content: "This is "},
				{Type: ElementParagraph, Content: "bold", Style: &TextStyle{Bold: true}},
				{Type: ElementParagraph, Content: " text."},
			}},
			{Type: ElementList, ListType: "bullet", Items: []DocsElement{
				{Type: ElementListItem, Items: []DocsElement{
					{Type: ElementParagraph, Items: []DocsElement{{Type: ElementParagraph, Content: "Item one"}}},
				}},
				{Type: ElementListItem, Items: []DocsElement{
					{Type: ElementParagraph, Items: []DocsElement{{Type: ElementParagraph, Content: "Item two"}}},
				}},
			}},
			{Type: ElementCodeBlock, Language: "bash", Content: "grctool sync"},
			{Type: ElementThematicBreak},
		},
	}

	md, err := DocToMarkdown(doc)
	require.NoError(t, err)
	assert.Contains(t, md, "# Introduction")
	assert.Contains(t, md, "**bold**")
	assert.Contains(t, md, "- Item one")
	assert.Contains(t, md, "- Item two")
	assert.Contains(t, md, "```bash")
	assert.Contains(t, md, "grctool sync")
	assert.Contains(t, md, "---")
}

func TestRoundTrip_SimpleDocument(t *testing.T) {
	original := "# Hello World\n\nThis is a simple document.\n\n## Section One\n\nSome text here.\n"

	doc, err := MarkdownToDoc(original)
	require.NoError(t, err)

	result, err := DocToMarkdown(doc)
	require.NoError(t, err)

	// Verify structural equivalence
	assert.Contains(t, result, "# Hello World")
	assert.Contains(t, result, "This is a simple document.")
	assert.Contains(t, result, "## Section One")
	assert.Contains(t, result, "Some text here.")
}

func TestRoundTrip_ComplexDocument(t *testing.T) {
	original := `# Access Control Policy

## Purpose

This policy defines **access control** requirements for *all systems*.

## Requirements

1. All access must be authenticated
2. Use role-based access control
3. Review access quarterly

### Technical Controls

- Enable MFA for all users
- Use SSO where possible
- Log all access events

## Code Example

` + "```yaml\naccess_control:\n  mfa_required: true\n  sso_enabled: true\n```" + `

---

## References

Visit [NIST](https://nist.gov) for more information.
`

	doc, err := MarkdownToDoc(original)
	require.NoError(t, err)

	result, err := DocToMarkdown(doc)
	require.NoError(t, err)

	// Verify key structural elements survived the round-trip
	assert.Contains(t, result, "# Access Control Policy")
	assert.Contains(t, result, "## Purpose")
	assert.Contains(t, result, "**access control**")
	assert.Contains(t, result, "*all systems*")
	assert.Contains(t, result, "## Requirements")
	assert.Contains(t, result, "1. All access must be authenticated")
	assert.Contains(t, result, "### Technical Controls")
	assert.Contains(t, result, "- Enable MFA for all users")
	assert.Contains(t, result, "```yaml")
	assert.Contains(t, result, "mfa_required: true")
	assert.Contains(t, result, "---")
	assert.Contains(t, result, "[NIST](https://nist.gov)")
}

func TestRoundTrip_PolicyDocument(t *testing.T) {
	// A realistic compliance policy document
	original := `# Data Protection Policy

## 1. Overview

This policy establishes the requirements for protecting sensitive data across all systems operated by the organization.

## 2. Scope

This policy applies to:

- All employees and contractors
- All information systems
- All data classifications

## 3. Data Classification

| Classification | Description | Examples |
| --- | --- | --- |
| Public | Non-sensitive data | Marketing materials |
| Internal | Business-sensitive | Financial reports |
| Confidential | Highly sensitive | Customer PII |

## 4. Controls

> All data at rest must be encrypted using AES-256 or equivalent.

### 4.1 Encryption Requirements

All production databases must implement:

1. Encryption at rest
2. Encryption in transit
3. Key rotation every 90 days

### 4.2 Access Controls

Use ` + "`RBAC`" + ` for all data access. See the [Access Control Policy](https://example.com/access) for details.
`

	doc, err := MarkdownToDoc(original)
	require.NoError(t, err)
	require.NotNil(t, doc)

	result, err := DocToMarkdown(doc)
	require.NoError(t, err)

	// Verify structural fidelity
	assert.Contains(t, result, "# Data Protection Policy")
	assert.Contains(t, result, "## 1. Overview")
	assert.Contains(t, result, "- All employees and contractors")
	assert.Contains(t, result, "| Classification")
	assert.Contains(t, result, "| Public")
	assert.Contains(t, result, "> All data at rest must be encrypted")
	assert.Contains(t, result, "### 4.1 Encryption Requirements")
	assert.Contains(t, result, "1. Encryption at rest")
	assert.Contains(t, result, "`RBAC`")
	assert.Contains(t, result, "[Access Control Policy](https://example.com/access)")
}

func TestContentHash_Deterministic(t *testing.T) {
	doc := &Document{
		Title: "Test",
		Elements: []DocsElement{
			{Type: ElementParagraph, Content: "Hello"},
		},
	}

	hash1 := ContentHash(doc)
	hash2 := ContentHash(doc)
	assert.Equal(t, hash1, hash2)
	assert.Len(t, hash1, 64) // SHA-256 hex is 64 chars
}

func TestContentHash_DifferentContent(t *testing.T) {
	doc1 := &Document{
		Title: "Test A",
		Elements: []DocsElement{
			{Type: ElementParagraph, Content: "Hello"},
		},
	}
	doc2 := &Document{
		Title: "Test B",
		Elements: []DocsElement{
			{Type: ElementParagraph, Content: "World"},
		},
	}

	hash1 := ContentHash(doc1)
	hash2 := ContentHash(doc2)
	assert.NotEqual(t, hash1, hash2)
}

func TestMarkdownToDoc_EmptyInput(t *testing.T) {
	doc, err := MarkdownToDoc("")
	require.NoError(t, err)
	require.NotNil(t, doc)
	assert.Empty(t, doc.Elements)
}
