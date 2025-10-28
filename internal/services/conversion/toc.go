// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"fmt"
	"strings"

	"github.com/signintech/gopdf"
	"github.com/yuin/goldmark/ast"
)

// TOCEntry represents a table of contents entry
type TOCEntry struct {
	Level    int     // Heading level (1-6)
	Title    string  // Heading text
	Page     int     // Page number where heading appears
	Y        float64 // Y position on page
	LinkName string  // Internal link name
}

// TOCGenerator generates table of contents
type TOCGenerator struct {
	entries  []TOCEntry
	maxDepth int
	source   []byte
}

// NewTOCGenerator creates a new TOC generator
func NewTOCGenerator(maxDepth int, source []byte) *TOCGenerator {
	if maxDepth < 1 {
		maxDepth = 3
	}
	if maxDepth > 6 {
		maxDepth = 6
	}

	return &TOCGenerator{
		entries:  []TOCEntry{},
		maxDepth: maxDepth,
		source:   source,
	}
}

// AddEntry adds a heading to the TOC
func (toc *TOCGenerator) AddEntry(level int, title string, page int, y float64) {
	// Only include headings up to maxDepth
	if level > toc.maxDepth {
		return
	}

	// Generate unique link name
	linkName := fmt.Sprintf("heading-%d-%s", len(toc.entries), sanitizeForLink(title))

	toc.entries = append(toc.entries, TOCEntry{
		Level:    level,
		Title:    title,
		Page:     page,
		Y:        y,
		LinkName: linkName,
	})
}

// GetEntries returns all TOC entries
func (toc *TOCGenerator) GetEntries() []TOCEntry {
	return toc.entries
}

// HasEntries returns true if there are any TOC entries
func (toc *TOCGenerator) HasEntries() bool {
	return len(toc.entries) > 0
}

// Render renders the table of contents to PDF
func (toc *TOCGenerator) Render(pdf *gopdf.GoPdf, fm *FontManager, opts *ConversionOptions, startY float64) (float64, error) {
	if !toc.HasEntries() {
		return startY, nil
	}

	currentY := startY

	// TOC title
	if err := fm.SetStyle(FontStyleBold); err != nil {
		return currentY, err
	}
	pdf.SetFontSize(18)
	pdf.SetXY(opts.Margins.Left, currentY)
	pdf.Cell(nil, "Table of Contents")
	currentY += 30

	// Reset to regular font
	if err := fm.SetStyle(FontStyleRegular); err != nil {
		return currentY, err
	}
	pdf.SetFontSize(fm.GetFontSize())

	// Render each entry
	for _, entry := range toc.entries {
		// Check if we need a new page
		if currentY > opts.Margins.Top+getPageHeight(opts)-opts.Margins.Bottom {
			pdf.AddPage()
			currentY = opts.Margins.Top
		}

		// Indent based on level
		indent := opts.Margins.Left + float64((entry.Level-1)*20)
		pdf.SetX(indent)
		pdf.SetY(currentY)

		// Format entry text with dots leading to page number
		entryText := fmt.Sprintf("%s %s %d",
			strings.Repeat("  ", entry.Level-1),
			entry.Title,
			entry.Page)

		// TODO: Add clickable link here when gopdf supports it better
		// For now, just render the text
		pdf.Cell(nil, entryText)

		currentY += 16
	}

	return currentY + 20, nil // Add spacing after TOC
}

// CollectFromAST walks the AST and collects all headings
func (toc *TOCGenerator) CollectFromAST(doc ast.Node) {
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		if heading, ok := n.(*ast.Heading); ok {
			// Extract heading text
			title := extractHeadingText(heading, toc.source)

			// Add to TOC (page and Y will be filled in during rendering)
			if heading.Level <= toc.maxDepth {
				toc.AddEntry(heading.Level, title, 0, 0)
			}
		}

		return ast.WalkContinue, nil
	})
}

// extractHeadingText extracts text from a heading node
func extractHeadingText(heading *ast.Heading, source []byte) string {
	var buf strings.Builder

	// Walk children to get text
	for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
		if text, ok := child.(*ast.Text); ok {
			buf.Write(text.Segment.Value(source))
		} else if str, ok := child.(*ast.String); ok {
			buf.Write(str.Value)
		}
	}

	return strings.TrimSpace(buf.String())
}

// sanitizeForLink creates a URL-safe link name
func sanitizeForLink(title string) string {
	// Convert to lowercase
	s := strings.ToLower(title)

	// Replace spaces and special chars with hyphens
	s = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, s)

	// Remove multiple consecutive hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	// Trim hyphens from start and end
	s = strings.Trim(s, "-")

	// Limit length
	if len(s) > 50 {
		s = s[:50]
	}

	return s
}

// getPageHeight returns the page height based on page size
func getPageHeight(opts *ConversionOptions) float64 {
	if opts.PageSize == "Letter" {
		return 792 // 11 inches * 72 points/inch
	}
	return 842 // A4: 297mm ≈ 11.69 inches
}
