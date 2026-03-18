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

// Package gdocs provides bidirectional conversion between Markdown and an
// intermediate Google Docs structural representation. The intermediate
// representation (DocsElement) maps to the Google Docs API structural content
// model without importing the actual API types.
package gdocs

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gmext "github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

// ElementType identifies the kind of structural element.
type ElementType string

const (
	ElementParagraph     ElementType = "paragraph"
	ElementHeading       ElementType = "heading"
	ElementList          ElementType = "list"
	ElementListItem      ElementType = "list_item"
	ElementCodeBlock     ElementType = "code_block"
	ElementTable         ElementType = "table"
	ElementThematicBreak ElementType = "thematic_break"
	ElementBlockquote    ElementType = "blockquote"
	ElementLink          ElementType = "link"
	ElementBold          ElementType = "bold"
	ElementItalic        ElementType = "italic"
	ElementCode          ElementType = "code"
	ElementImage         ElementType = "image"
)

// TextStyle carries inline formatting attributes for a text run.
type TextStyle struct {
	Bold   bool   `json:"bold,omitempty"`
	Italic bool   `json:"italic,omitempty"`
	Code   bool   `json:"code,omitempty"`
	Link   string `json:"link,omitempty"`
}

// DocsElement represents a structural element in a Google Doc.
// It maps to the Google Docs API StructuralElement type.
type DocsElement struct {
	Type     ElementType   `json:"type"`
	Content  string        `json:"content,omitempty"`
	Level    int           `json:"level,omitempty"`      // heading level 1-6
	Items    []DocsElement `json:"items,omitempty"`       // list items, table cells, inline runs
	Style    *TextStyle    `json:"style,omitempty"`       // inline formatting
	Rows     [][]string    `json:"rows,omitempty"`        // table rows (simplified: cells as strings)
	Language string        `json:"language,omitempty"`    // code block language
	URL      string        `json:"url,omitempty"`         // link URL
	ListType string        `json:"list_type,omitempty"`   // "bullet" or "ordered"
}

// Document represents a complete Google Doc structure.
type Document struct {
	Title    string        `json:"title"`
	Elements []DocsElement `json:"elements"`
}

// MarkdownToDoc converts Markdown text to a Document structure using goldmark.
func MarkdownToDoc(markdown string) (*Document, error) {
	if markdown == "" {
		return &Document{Elements: []DocsElement{}}, nil
	}

	md := goldmark.New(
		goldmark.WithExtensions(gmext.NewTable()),
	)

	source := []byte(markdown)
	reader := text.NewReader(source)
	tree := md.Parser().Parse(reader)

	doc := &Document{}
	err := walkNode(tree, source, doc)
	if err != nil {
		return nil, fmt.Errorf("converting markdown to doc: %w", err)
	}

	return doc, nil
}

// walkNode recursively walks the goldmark AST and populates Document elements.
func walkNode(n ast.Node, source []byte, doc *Document) error {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		elem, err := convertNode(child, source)
		if err != nil {
			return err
		}
		if elem != nil {
			doc.Elements = append(doc.Elements, *elem)
		}
	}
	return nil
}

// convertNode converts a single AST node to a DocsElement.
func convertNode(n ast.Node, source []byte) (*DocsElement, error) {
	switch node := n.(type) {
	case *ast.Heading:
		runs := extractInlineRuns(node, source)
		return &DocsElement{
			Type:  ElementHeading,
			Level: node.Level,
			Items: runs,
		}, nil

	case *ast.Paragraph:
		runs := extractInlineRuns(node, source)
		return &DocsElement{
			Type:  ElementParagraph,
			Items: runs,
		}, nil

	case *ast.TextBlock:
		runs := extractInlineRuns(node, source)
		return &DocsElement{
			Type:  ElementParagraph,
			Items: runs,
		}, nil

	case *ast.List:
		listType := "bullet"
		if node.IsOrdered() {
			listType = "ordered"
		}
		var items []DocsElement
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			if li, ok := child.(*ast.ListItem); ok {
				item, err := convertListItem(li, source)
				if err != nil {
					return nil, err
				}
				items = append(items, *item)
			}
		}
		return &DocsElement{
			Type:     ElementList,
			ListType: listType,
			Items:    items,
		}, nil

	case *ast.FencedCodeBlock:
		lang := ""
		if node.Language(source) != nil {
			lang = string(node.Language(source))
		}
		var buf bytes.Buffer
		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			buf.Write(seg.Value(source))
		}
		content := buf.String()
		// Remove trailing newline from code content
		content = strings.TrimRight(content, "\n")
		return &DocsElement{
			Type:     ElementCodeBlock,
			Content:  content,
			Language: lang,
		}, nil

	case *ast.CodeBlock:
		var buf bytes.Buffer
		lines := node.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			buf.Write(seg.Value(source))
		}
		content := strings.TrimRight(buf.String(), "\n")
		return &DocsElement{
			Type:    ElementCodeBlock,
			Content: content,
		}, nil

	case *ast.ThematicBreak:
		return &DocsElement{
			Type: ElementThematicBreak,
		}, nil

	case *ast.Blockquote:
		var items []DocsElement
		for child := node.FirstChild(); child != nil; child = child.NextSibling() {
			elem, err := convertNode(child, source)
			if err != nil {
				return nil, err
			}
			if elem != nil {
				items = append(items, *elem)
			}
		}
		return &DocsElement{
			Type:  ElementBlockquote,
			Items: items,
		}, nil

	default:
		// Handle table extension nodes via kind check
		kind := n.Kind()
		if kind.String() == "Table" {
			return convertTable(n, source)
		}
		// Skip unknown node types
		return nil, nil
	}
}

// convertListItem converts a list item node to a DocsElement.
func convertListItem(li *ast.ListItem, source []byte) (*DocsElement, error) {
	var items []DocsElement
	for child := li.FirstChild(); child != nil; child = child.NextSibling() {
		elem, err := convertNode(child, source)
		if err != nil {
			return nil, err
		}
		if elem != nil {
			items = append(items, *elem)
		}
	}
	return &DocsElement{
		Type:  ElementListItem,
		Items: items,
	}, nil
}

// convertTable converts a goldmark extension table node to a DocsElement.
func convertTable(n ast.Node, source []byte) (*DocsElement, error) {
	var rows [][]string
	for row := n.FirstChild(); row != nil; row = row.NextSibling() {
		var cells []string
		for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
			cellText := extractPlainText(cell, source)
			cells = append(cells, cellText)
		}
		rows = append(rows, cells)
	}
	return &DocsElement{
		Type: ElementTable,
		Rows: rows,
	}, nil
}

// extractInlineRuns walks inline children and produces a flat list of styled runs.
func extractInlineRuns(n ast.Node, source []byte) []DocsElement {
	var runs []DocsElement
	collectInlines(n, source, &TextStyle{}, &runs)
	return runs
}

// collectInlines recursively collects inline elements, tracking accumulated style.
func collectInlines(n ast.Node, source []byte, style *TextStyle, runs *[]DocsElement) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {
		case *ast.Text:
			content := string(node.Segment.Value(source))
			if content == "" {
				continue
			}
			elem := DocsElement{
				Type:    ElementParagraph,
				Content: content,
			}
			if style.Bold || style.Italic || style.Code || style.Link != "" {
				elem.Style = &TextStyle{
					Bold:   style.Bold,
					Italic: style.Italic,
					Code:   style.Code,
					Link:   style.Link,
				}
			}
			*runs = append(*runs, elem)
			// If the text has a soft line break after it, add a space
			if node.SoftLineBreak() {
				*runs = append(*runs, DocsElement{
					Type:    ElementParagraph,
					Content: " ",
				})
			}

		case *ast.String:
			content := string(node.Value)
			if content == "" {
				continue
			}
			elem := DocsElement{
				Type:    ElementParagraph,
				Content: content,
			}
			if style.Bold || style.Italic || style.Code || style.Link != "" {
				elem.Style = &TextStyle{
					Bold:   style.Bold,
					Italic: style.Italic,
					Code:   style.Code,
					Link:   style.Link,
				}
			}
			*runs = append(*runs, elem)

		case *ast.Emphasis:
			newStyle := *style
			if node.Level == 1 {
				newStyle.Italic = true
			} else if node.Level >= 2 {
				newStyle.Bold = true
			}
			collectInlines(child, source, &newStyle, runs)

		case *ast.CodeSpan:
			content := extractPlainText(node, source)
			elem := DocsElement{
				Type:    ElementCode,
				Content: content,
				Style: &TextStyle{
					Code: true,
				},
			}
			*runs = append(*runs, elem)

		case *ast.Link:
			linkStyle := *style
			linkStyle.Link = string(node.Destination)
			linkContent := extractPlainText(node, source)
			elem := DocsElement{
				Type:    ElementLink,
				Content: linkContent,
				URL:     string(node.Destination),
				Style: &TextStyle{
					Bold:   style.Bold,
					Italic: style.Italic,
					Link:   string(node.Destination),
				},
			}
			*runs = append(*runs, elem)

		case *ast.Image:
			altText := extractPlainText(node, source)
			elem := DocsElement{
				Type:    ElementImage,
				Content: altText,
				URL:     string(node.Destination),
			}
			*runs = append(*runs, elem)

		case *ast.AutoLink:
			url := string(node.URL(source))
			elem := DocsElement{
				Type:    ElementLink,
				Content: url,
				URL:     url,
				Style: &TextStyle{
					Link: url,
				},
			}
			*runs = append(*runs, elem)

		default:
			// Recurse for unknown inline containers
			collectInlines(child, source, style, runs)
		}
	}
}

// extractPlainText extracts the plain text content of a node and its children.
func extractPlainText(n ast.Node, source []byte) string {
	var buf bytes.Buffer
	extractPlainTextRecursive(n, source, &buf)
	return buf.String()
}

func extractPlainTextRecursive(n ast.Node, source []byte, buf *bytes.Buffer) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch node := child.(type) {
		case *ast.Text:
			buf.Write(node.Segment.Value(source))
		case *ast.String:
			buf.Write(node.Value)
		case *ast.CodeSpan:
			extractPlainTextRecursive(child, source, buf)
		default:
			extractPlainTextRecursive(child, source, buf)
		}
	}
}

// DocToMarkdown converts a Document structure back to Markdown text.
func DocToMarkdown(doc *Document) (string, error) {
	if doc == nil {
		return "", nil
	}

	var buf bytes.Buffer
	for i, elem := range doc.Elements {
		if i > 0 {
			buf.WriteString("\n")
		}
		writeElement(&buf, &elem, 0)
	}
	result := buf.String()
	// Ensure trailing newline
	if result != "" && !strings.HasSuffix(result, "\n") {
		result += "\n"
	}
	return result, nil
}

// writeElement writes a single DocsElement as Markdown into the buffer.
func writeElement(buf *bytes.Buffer, elem *DocsElement, depth int) {
	switch elem.Type {
	case ElementHeading:
		prefix := strings.Repeat("#", elem.Level)
		buf.WriteString(prefix + " ")
		writeInlineRuns(buf, elem.Items)
		buf.WriteString("\n")

	case ElementParagraph:
		if len(elem.Items) > 0 {
			writeInlineRuns(buf, elem.Items)
		} else if elem.Content != "" {
			buf.WriteString(elem.Content)
		}
		buf.WriteString("\n")

	case ElementList:
		for i, item := range elem.Items {
			writeListItem(buf, &item, elem.ListType, i+1, depth)
		}

	case ElementCodeBlock:
		buf.WriteString("```")
		if elem.Language != "" {
			buf.WriteString(elem.Language)
		}
		buf.WriteString("\n")
		buf.WriteString(elem.Content)
		buf.WriteString("\n```\n")

	case ElementThematicBreak:
		buf.WriteString("---\n")

	case ElementBlockquote:
		for _, item := range elem.Items {
			var inner bytes.Buffer
			writeElement(&inner, &item, depth)
			lines := strings.Split(strings.TrimRight(inner.String(), "\n"), "\n")
			for _, line := range lines {
				if line == "" {
					buf.WriteString(">\n")
				} else {
					buf.WriteString("> " + line + "\n")
				}
			}
		}

	case ElementTable:
		if len(elem.Rows) == 0 {
			return
		}
		// Write header row
		buf.WriteString("| " + strings.Join(elem.Rows[0], " | ") + " |\n")
		// Write separator
		seps := make([]string, len(elem.Rows[0]))
		for i := range seps {
			seps[i] = "---"
		}
		buf.WriteString("| " + strings.Join(seps, " | ") + " |\n")
		// Write data rows
		for _, row := range elem.Rows[1:] {
			buf.WriteString("| " + strings.Join(row, " | ") + " |\n")
		}
	}
}

// writeListItem writes a single list item as Markdown.
func writeListItem(buf *bytes.Buffer, item *DocsElement, listType string, index int, depth int) {
	indent := strings.Repeat("  ", depth)
	prefix := "- "
	if listType == "ordered" {
		prefix = fmt.Sprintf("%d. ", index)
	}

	if len(item.Items) == 0 {
		buf.WriteString(indent + prefix + item.Content + "\n")
		return
	}

	for i, sub := range item.Items {
		if sub.Type == ElementParagraph {
			if i == 0 {
				buf.WriteString(indent + prefix)
			} else {
				buf.WriteString(indent + "  ")
			}
			if len(sub.Items) > 0 {
				writeInlineRuns(buf, sub.Items)
			} else {
				buf.WriteString(sub.Content)
			}
			buf.WriteString("\n")
		} else if sub.Type == ElementList {
			// Nested list
			for j, nestedItem := range sub.Items {
				writeListItem(buf, &nestedItem, sub.ListType, j+1, depth+1)
			}
		}
	}
}

// writeInlineRuns writes a sequence of inline runs as Markdown.
func writeInlineRuns(buf *bytes.Buffer, runs []DocsElement) {
	for _, run := range runs {
		writeInlineRun(buf, &run)
	}
}

// writeInlineRun writes a single inline run as Markdown.
func writeInlineRun(buf *bytes.Buffer, run *DocsElement) {
	switch run.Type {
	case ElementCode:
		buf.WriteString("`" + run.Content + "`")
	case ElementLink:
		buf.WriteString("[" + run.Content + "](" + run.URL + ")")
	case ElementImage:
		buf.WriteString("![" + run.Content + "](" + run.URL + ")")
	default:
		content := run.Content
		if run.Style != nil {
			if run.Style.Link != "" {
				buf.WriteString("[" + content + "](" + run.Style.Link + ")")
				return
			}
			if run.Style.Code {
				buf.WriteString("`" + content + "`")
				return
			}
			if run.Style.Bold && run.Style.Italic {
				buf.WriteString("***" + content + "***")
				return
			}
			if run.Style.Bold {
				buf.WriteString("**" + content + "**")
				return
			}
			if run.Style.Italic {
				buf.WriteString("*" + content + "*")
				return
			}
		}
		buf.WriteString(content)
	}
}

// ContentHash computes a SHA-256 hash of the document content for change detection.
func ContentHash(doc *Document) string {
	data, err := json.Marshal(doc)
	if err != nil {
		// Fallback: hash the title
		h := sha256.Sum256([]byte(doc.Title))
		return fmt.Sprintf("%x", h)
	}
	h := sha256.Sum256(data)
	return fmt.Sprintf("%x", h)
}
