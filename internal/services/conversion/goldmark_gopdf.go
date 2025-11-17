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

package conversion

import (
	"fmt"
	"os"
	"strings"

	"github.com/signintech/gopdf"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	extast "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// GoldmarkGoPDFConverter converts markdown to PDF using goldmark and gopdf
type GoldmarkGoPDFConverter struct {
	md goldmark.Markdown
}

// NewGoldmarkGoPDFConverter creates a new converter instance
func NewGoldmarkGoPDFConverter() *GoldmarkGoPDFConverter {
	md := goldmark.New(
		goldmark.WithExtensions(
			extension.GFM,           // GitHub Flavored Markdown
			extension.Table,         // Tables
			extension.Strikethrough, // Strikethrough text
			extension.TaskList,      // Task lists
		),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
	)

	return &GoldmarkGoPDFConverter{md: md}
}

// renderContext holds state during PDF rendering
type renderContext struct {
	pdf           *gopdf.GoPdf
	opts          *ConversionOptions
	source        []byte
	currentY      float64
	pageHeight    float64
	listLevel     int
	inCodeBlock   bool
	inTable       bool
	tableColCount int
	// Professional features
	fontManager   *FontManager
	syntaxHL      *SyntaxHighlighter
	tocGen        *TOCGenerator
	pageLayout    *PageLayout
	currentPage   int
	tocEntryIndex int // Track which TOC entry we're at during rendering
}

// ConvertMarkdownToPDF converts a markdown file to PDF
func (c *GoldmarkGoPDFConverter) ConvertMarkdownToPDF(inputPath, outputPath string, opts *ConversionOptions) error {
	// Use default options if not provided
	if opts == nil {
		opts = DefaultOptions()
	}

	// Read markdown file
	source, err := os.ReadFile(inputPath)
	if err != nil {
		return fmt.Errorf("reading markdown file: %w", err)
	}

	// Parse markdown to AST
	doc := c.md.Parser().Parse(text.NewReader(source))

	// Create PDF
	pdf := &gopdf.GoPdf{}

	// Determine page size
	var pageSize *gopdf.Rect
	if opts.PageSize == "Letter" {
		pageSize = gopdf.PageSizeLetter
	} else {
		pageSize = gopdf.PageSizeA4
	}

	pdf.Start(gopdf.Config{PageSize: *pageSize})

	// Initialize professional features
	fontManager := NewFontManager(pdf, opts.FontSize, opts.CodeFontSize)
	if err := fontManager.LoadFonts(opts.FontFamily, opts.MonoFontFamily); err != nil {
		return fmt.Errorf("loading fonts: %w", err)
	}

	syntaxHL := NewSyntaxHighlighter(opts.SyntaxTheme)
	tocGen := NewTOCGenerator(opts.TOCDepth, source)
	pageLayout := NewPageLayout(opts, fontManager)

	// First pass: collect TOC entries if enabled
	if opts.GenerateTOC {
		tocGen.CollectFromAST(doc)
	}

	// Start first page
	pdf.AddPage()

	// Render title page
	_, err = pageLayout.RenderTitlePage(pdf)
	if err != nil {
		return fmt.Errorf("rendering title page: %w", err)
	}

	// Render TOC if enabled and has entries
	if opts.GenerateTOC && tocGen.HasEntries() {
		pdf.AddPage()
		_, err = tocGen.Render(pdf, fontManager, opts, opts.Margins.Top)
		if err != nil {
			return fmt.Errorf("rendering TOC: %w", err)
		}
	}

	// Start content on new page
	pdf.AddPage()

	// Calculate available page height
	pageHeight := pageSize.H - opts.Margins.Top - opts.Margins.Bottom

	// Create render context
	ctx := &renderContext{
		pdf:           pdf,
		opts:          opts,
		source:        source,
		currentY:      pageLayout.GetContentStartY(pdf.GetNumberOfPages()),
		pageHeight:    pageHeight,
		fontManager:   fontManager,
		syntaxHL:      syntaxHL,
		tocGen:        tocGen,
		pageLayout:    pageLayout,
		currentPage:   pdf.GetNumberOfPages(),
		tocEntryIndex: 0,
	}

	// Set initial font
	if err := fontManager.SetStyle(FontStyleRegular); err != nil {
		return fmt.Errorf("setting initial font: %w", err)
	}
	pdf.SetFontSize(opts.FontSize)

	// Walk AST and render to PDF
	err = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		return c.renderNode(ctx, n, entering)
	})
	if err != nil {
		return fmt.Errorf("rendering PDF: %w", err)
	}

	// Add headers and footers to all pages
	totalPages := pdf.GetNumberOfPages()
	pageLayout.SetTotalPages(totalPages)
	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		if err := pageLayout.AddHeader(pdf, pageNum); err != nil {
			return fmt.Errorf("adding header to page %d: %w", pageNum, err)
		}
		if err := pageLayout.AddFooter(pdf, pageNum); err != nil {
			return fmt.Errorf("adding footer to page %d: %w", pageNum, err)
		}
	}

	// Write PDF to file
	if err := pdf.WritePdf(outputPath); err != nil {
		return fmt.Errorf("writing PDF file: %w", err)
	}

	return nil
}

// renderNode renders a single AST node to PDF
func (c *GoldmarkGoPDFConverter) renderNode(ctx *renderContext, n ast.Node, entering bool) (ast.WalkStatus, error) {
	switch n := n.(type) {
	case *ast.Heading:
		return c.renderHeading(ctx, n, entering)
	case *ast.Paragraph:
		return c.renderParagraph(ctx, n, entering)
	case *ast.Text:
		return c.renderText(ctx, n, entering)
	case *ast.String:
		return c.renderString(ctx, n, entering)
	case *ast.CodeBlock, *ast.FencedCodeBlock:
		return c.renderCodeBlock(ctx, n, entering)
	case *ast.List:
		return c.renderList(ctx, n, entering)
	case *ast.ListItem:
		return c.renderListItem(ctx, n, entering)
	case *ast.Emphasis:
		return c.renderEmphasis(ctx, n, entering)
	case *extast.Table:
		return c.renderTable(ctx, n, entering)
	case *extast.TableRow:
		return c.renderTableRow(ctx, n, entering)
	case *extast.TableCell:
		return c.renderTableCell(ctx, n, entering)
	}

	return ast.WalkContinue, nil
}

// renderHeading renders a heading
func (c *GoldmarkGoPDFConverter) renderHeading(ctx *renderContext, n *ast.Heading, entering bool) (ast.WalkStatus, error) {
	if entering {
		// Add spacing before heading
		ctx.currentY += 12

		// Check for page break
		if err := c.checkPageBreak(ctx, 40); err != nil {
			return ast.WalkStop, err
		}

		// Set bold font for headings
		if err := ctx.fontManager.SetStyle(FontStyleBold); err != nil {
			return ast.WalkStop, err
		}

		// Set font size based on heading level
		fontSize := 18.0 - (float64(n.Level) * 2.0)
		if fontSize < 11 {
			fontSize = 11
		}
		ctx.pdf.SetFontSize(fontSize)

		ctx.pdf.SetX(ctx.opts.Margins.Left)
		ctx.pdf.SetY(ctx.currentY)
		ctx.pdf.SetTextColor(0, 0, 0)

		// Update TOC entry with page number if this is a TOC heading
		if ctx.tocGen != nil && ctx.tocGen.HasEntries() {
			entries := ctx.tocGen.GetEntries()
			if ctx.tocEntryIndex < len(entries) {
				entries[ctx.tocEntryIndex].Page = ctx.currentPage
				entries[ctx.tocEntryIndex].Y = ctx.currentY
				ctx.tocEntryIndex++
			}
		}
	} else {
		// Restore regular font after heading
		if err := ctx.fontManager.SetStyle(FontStyleRegular); err != nil {
			return ast.WalkStop, err
		}
		ctx.pdf.SetFontSize(ctx.opts.FontSize)

		// Add spacing after heading
		ctx.currentY += 8
	}

	return ast.WalkContinue, nil
}

// renderParagraph renders a paragraph
func (c *GoldmarkGoPDFConverter) renderParagraph(ctx *renderContext, n *ast.Paragraph, entering bool) (ast.WalkStatus, error) {
	if entering {
		// Add spacing before paragraph
		ctx.currentY += 6

		// Check for page break
		if err := c.checkPageBreak(ctx, 20); err != nil {
			return ast.WalkStop, err
		}

		// Ensure regular font style
		if err := ctx.fontManager.SetStyle(FontStyleRegular); err != nil {
			return ast.WalkStop, err
		}

		ctx.pdf.SetX(ctx.opts.Margins.Left)
		ctx.pdf.SetY(ctx.currentY)
		ctx.pdf.SetFontSize(ctx.opts.FontSize)
	} else {
		// Add spacing after paragraph
		ctx.currentY += 6
	}

	return ast.WalkContinue, nil
}

// renderText renders text content
func (c *GoldmarkGoPDFConverter) renderText(ctx *renderContext, n *ast.Text, entering bool) (ast.WalkStatus, error) {
	if entering {
		text := string(n.Segment.Value(ctx.source))
		ctx.pdf.Cell(nil, text)
		ctx.currentY = ctx.pdf.GetY()
	}

	return ast.WalkContinue, nil
}

// renderString renders a string node
func (c *GoldmarkGoPDFConverter) renderString(ctx *renderContext, n *ast.String, entering bool) (ast.WalkStatus, error) {
	if entering {
		text := string(n.Value)
		ctx.pdf.Cell(nil, text)
		ctx.currentY = ctx.pdf.GetY()
	}

	return ast.WalkContinue, nil
}

// renderCodeBlock renders a code block with syntax highlighting
func (c *GoldmarkGoPDFConverter) renderCodeBlock(ctx *renderContext, n ast.Node, entering bool) (ast.WalkStatus, error) {
	if entering {
		ctx.currentY += 8

		// Check for page break
		if err := c.checkPageBreak(ctx, 40); err != nil {
			return ast.WalkStop, err
		}

		// Set monospace font
		if err := ctx.fontManager.SetStyle(FontStyleMono); err != nil {
			return ast.WalkStop, err
		}
		ctx.pdf.SetFontSize(ctx.opts.CodeFontSize)

		// Get code content
		var code strings.Builder
		lines := n.Lines()
		for i := 0; i < lines.Len(); i++ {
			line := lines.At(i)
			code.Write(line.Value(ctx.source))
		}
		codeText := code.String()

		// Detect language from fenced code block
		language := ""
		if fenced, ok := n.(*ast.FencedCodeBlock); ok {
			if fenced.Info != nil {
				langBytes := fenced.Info.Segment.Value(ctx.source)
				language = string(langBytes)
				language = NormalizeLanguageName(language)
			}
		}

		// Apply syntax highlighting
		var tokens []ColoredToken
		if language != "" && ctx.syntaxHL != nil {
			tokens = ctx.syntaxHL.HighlightCode(codeText, language)
		} else {
			// Fallback to plain text with default color
			tokens = []ColoredToken{{Text: codeText, Color: RGB{60, 60, 60}}}
		}

		// Render highlighted code
		ctx.pdf.SetX(ctx.opts.Margins.Left + 10)
		ctx.pdf.SetY(ctx.currentY)

		for _, token := range tokens {
			// Split token into lines
			tokenLines := strings.Split(token.Text, "\n")
			for i, line := range tokenLines {
				if i > 0 {
					// New line
					ctx.currentY += 12
					ctx.pdf.SetXY(ctx.opts.Margins.Left+10, ctx.currentY)

					// Check for page break
					if err := c.checkPageBreak(ctx, 12); err != nil {
						return ast.WalkStop, err
					}
				}

				if line != "" {
					// Set color for this token
					ctx.pdf.SetTextColor(token.Color.R, token.Color.G, token.Color.B)

					// Render text
					ctx.pdf.Cell(nil, line)
				}
			}
		}

		// Restore regular font and color
		ctx.pdf.SetTextColor(0, 0, 0)
		if err := ctx.fontManager.SetStyle(FontStyleRegular); err != nil {
			return ast.WalkStop, err
		}
		ctx.pdf.SetFontSize(ctx.opts.FontSize)
		ctx.currentY += 8
	}

	return ast.WalkContinue, nil
}

// renderList renders a list
func (c *GoldmarkGoPDFConverter) renderList(ctx *renderContext, n *ast.List, entering bool) (ast.WalkStatus, error) {
	if entering {
		ctx.listLevel++
		ctx.currentY += 4
	} else {
		ctx.listLevel--
		ctx.currentY += 4
	}

	return ast.WalkContinue, nil
}

// renderListItem renders a list item
func (c *GoldmarkGoPDFConverter) renderListItem(ctx *renderContext, n *ast.ListItem, entering bool) (ast.WalkStatus, error) {
	if entering {
		// Check for page break
		if err := c.checkPageBreak(ctx, 20); err != nil {
			return ast.WalkStop, err
		}

		indent := ctx.opts.Margins.Left + float64(ctx.listLevel*20)
		ctx.pdf.SetX(indent)
		ctx.pdf.SetY(ctx.currentY)

		// Add bullet point
		ctx.pdf.Cell(nil, "• ")
		ctx.currentY += 14
	}

	return ast.WalkContinue, nil
}

// renderEmphasis renders emphasized (italic/bold) text
// Note: ast.Emphasis handles both italic and bold depending on Level
func (c *GoldmarkGoPDFConverter) renderEmphasis(ctx *renderContext, n *ast.Emphasis, entering bool) (ast.WalkStatus, error) {
	if entering {
		// n.Level == 1 is italic (*text* or _text_)
		// n.Level == 2 is bold (**text** or __text__)
		if n.Level == 2 {
			if err := ctx.fontManager.SetStyle(FontStyleBold); err != nil {
				return ast.WalkStop, err
			}
		} else if n.Level == 1 {
			if err := ctx.fontManager.SetStyle(FontStyleItalic); err != nil {
				return ast.WalkStop, err
			}
		}
	} else {
		// Restore regular font
		if err := ctx.fontManager.SetStyle(FontStyleRegular); err != nil {
			return ast.WalkStop, err
		}
	}

	return ast.WalkContinue, nil
}

// renderTable renders a table (simplified)
func (c *GoldmarkGoPDFConverter) renderTable(ctx *renderContext, n *extast.Table, entering bool) (ast.WalkStatus, error) {
	if entering {
		ctx.inTable = true
		ctx.currentY += 8
		ctx.tableColCount = 0
	} else {
		ctx.inTable = false
		ctx.currentY += 8
	}

	return ast.WalkContinue, nil
}

// renderTableRow renders a table row
func (c *GoldmarkGoPDFConverter) renderTableRow(ctx *renderContext, n *extast.TableRow, entering bool) (ast.WalkStatus, error) {
	if entering {
		// Check for page break
		if err := c.checkPageBreak(ctx, 20); err != nil {
			return ast.WalkStop, err
		}

		ctx.pdf.SetX(ctx.opts.Margins.Left)
		ctx.pdf.SetY(ctx.currentY)
	} else {
		ctx.currentY += 16
	}

	return ast.WalkContinue, nil
}

// renderTableCell renders a table cell
func (c *GoldmarkGoPDFConverter) renderTableCell(ctx *renderContext, n *extast.TableCell, entering bool) (ast.WalkStatus, error) {
	if entering {
		// Simple table cell rendering
		cellWidth := 100.0
		x := ctx.opts.Margins.Left + float64(ctx.tableColCount)*cellWidth
		ctx.pdf.SetX(x)
		ctx.tableColCount++
	}

	return ast.WalkContinue, nil
}

// checkPageBreak checks if a new page is needed
func (c *GoldmarkGoPDFConverter) checkPageBreak(ctx *renderContext, heightNeeded float64) error {
	maxY := ctx.pageLayout.GetContentEndY()
	if ctx.currentY+heightNeeded > maxY {
		ctx.pdf.AddPage()
		ctx.currentPage = ctx.pdf.GetNumberOfPages()
		ctx.currentY = ctx.pageLayout.GetContentStartY(ctx.currentPage)
	}

	return nil
}
