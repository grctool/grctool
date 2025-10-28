// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"fmt"
	"os"
	"time"

	"github.com/signintech/gopdf"
)

// PageLayout manages page headers, footers, and title pages
type PageLayout struct {
	opts       *ConversionOptions
	fm         *FontManager
	totalPages int
}

// NewPageLayout creates a new page layout manager
func NewPageLayout(opts *ConversionOptions, fm *FontManager) *PageLayout {
	return &PageLayout{
		opts: opts,
		fm:   fm,
	}
}

// SetTotalPages sets the total page count (for "Page X of Y")
func (pl *PageLayout) SetTotalPages(total int) {
	pl.totalPages = total
}

// RenderTitlePage renders the title page with optional header image
func (pl *PageLayout) RenderTitlePage(pdf *gopdf.GoPdf) (float64, error) {
	currentY := pl.opts.Margins.Top

	// Render header image if provided
	if pl.opts.HeaderImage != "" {
		imageY, err := pl.renderHeaderImage(pdf)
		if err != nil {
			// Log warning but continue
			fmt.Printf("Warning: failed to render header image: %v\n", err)
		} else {
			currentY = imageY + 20 // Add spacing after image
		}
	}

	// Render document title
	if err := pl.fm.SetStyle(FontStyleBold); err != nil {
		return currentY, err
	}
	pdf.SetFontSize(24)
	pdf.SetXY(pl.opts.Margins.Left, currentY)

	title := pl.opts.Title
	if title == "" {
		title = "Evidence Document"
	}
	pdf.Cell(nil, title)
	currentY += 40

	// Render metadata
	if err := pl.fm.SetStyle(FontStyleRegular); err != nil {
		return currentY, err
	}
	pdf.SetFontSize(11)

	// Task reference
	if pl.opts.TaskRef != "" {
		pdf.SetXY(pl.opts.Margins.Left, currentY)
		pdf.Cell(nil, fmt.Sprintf("Task: %s", pl.opts.TaskRef))
		currentY += 16
	}

	// Collection window
	if pl.opts.Window != "" {
		pdf.SetXY(pl.opts.Margins.Left, currentY)
		pdf.Cell(nil, fmt.Sprintf("Collection Window: %s", pl.opts.Window))
		currentY += 16
	}

	// Generated date
	pdf.SetXY(pl.opts.Margins.Left, currentY)
	pdf.Cell(nil, fmt.Sprintf("Generated: %s", time.Now().Format("January 2, 2006")))
	currentY += 16

	// Subject
	if pl.opts.Subject != "" {
		currentY += 10
		pdf.SetXY(pl.opts.Margins.Left, currentY)
		pdf.Cell(nil, fmt.Sprintf("Subject: %s", pl.opts.Subject))
		currentY += 16
	}

	return currentY + 40, nil // Add spacing before content
}

// renderHeaderImage renders the header image at the top of the page
func (pl *PageLayout) renderHeaderImage(pdf *gopdf.GoPdf) (float64, error) {
	// Check if image file exists
	if _, err := os.Stat(pl.opts.HeaderImage); os.IsNotExist(err) {
		return pl.opts.Margins.Top, fmt.Errorf("header image not found: %s", pl.opts.HeaderImage)
	}

	// Calculate image dimensions
	pageWidth := getPageWidth(pl.opts)
	availableWidth := pageWidth - pl.opts.Margins.Left - pl.opts.Margins.Right
	imageHeight := pl.opts.HeaderImageHeight
	if imageHeight == 0 {
		imageHeight = 100
	}

	// Position at top of page, centered
	x := pl.opts.Margins.Left
	y := pl.opts.Margins.Top

	// Create image holder and add image
	imgH, err := gopdf.ImageHolderByPath(pl.opts.HeaderImage)
	if err != nil {
		return y, fmt.Errorf("failed to load image: %w", err)
	}

	// Calculate scaling to fit width while maintaining aspect ratio
	rect := &gopdf.Rect{
		W: availableWidth,
		H: imageHeight,
	}

	// Add image to PDF
	if err := pdf.ImageByHolder(imgH, x, y, rect); err != nil {
		return y, fmt.Errorf("failed to add image to PDF: %w", err)
	}

	return y + imageHeight, nil
}

// AddHeader adds a header to the current page
func (pl *PageLayout) AddHeader(pdf *gopdf.GoPdf, pageNum int) error {
	if pageNum <= 1 {
		return nil // No header on title page
	}

	// Save current position
	currentFont := pl.fm.currentStyle

	// Set header font
	if err := pl.fm.SetStyle(FontStyleRegular); err != nil {
		return err
	}
	pdf.SetFontSize(9)
	pdf.SetTextColor(128, 128, 128) // Gray text

	headerY := pl.opts.Margins.Top / 2
	pageWidth := getPageWidth(pl.opts)

	// Left: Task reference
	if pl.opts.TaskRef != "" {
		pdf.SetXY(pl.opts.Margins.Left, headerY)
		pdf.Cell(nil, pl.opts.TaskRef)
	}

	// Center: Document title (truncated if needed)
	title := pl.opts.Title
	if len(title) > 40 {
		title = title[:37] + "..."
	}
	centerX := pageWidth / 2
	pdf.SetXY(centerX-50, headerY) // Approximate centering
	pdf.Cell(nil, title)

	// Right: Window
	if pl.opts.Window != "" {
		rightX := pageWidth - pl.opts.Margins.Right - 60
		pdf.SetXY(rightX, headerY)
		pdf.Cell(nil, pl.opts.Window)
	}

	// Draw line under header
	lineY := headerY + 12
	pdf.SetLineWidth(0.5)
	pdf.SetStrokeColor(200, 200, 200)
	pdf.Line(pl.opts.Margins.Left, lineY, pageWidth-pl.opts.Margins.Right, lineY)

	// Restore font and color
	pdf.SetTextColor(0, 0, 0)
	pl.fm.SetStyle(currentFont)
	pdf.SetFontSize(pl.fm.GetFontSize())

	return nil
}

// AddFooter adds a footer to the current page
func (pl *PageLayout) AddFooter(pdf *gopdf.GoPdf, pageNum int) error {
	// Save current position
	currentFont := pl.fm.currentStyle

	// Set footer font
	if err := pl.fm.SetStyle(FontStyleRegular); err != nil {
		return err
	}
	pdf.SetFontSize(9)
	pdf.SetTextColor(128, 128, 128) // Gray text

	pageHeight := getPageHeight(pl.opts)
	footerY := pageHeight - pl.opts.Margins.Bottom/2
	pageWidth := getPageWidth(pl.opts)

	// Draw line above footer
	lineY := footerY - 12
	pdf.SetLineWidth(0.5)
	pdf.SetStrokeColor(200, 200, 200)
	pdf.Line(pl.opts.Margins.Left, lineY, pageWidth-pl.opts.Margins.Right, lineY)

	// Left: Generated date
	pdf.SetXY(pl.opts.Margins.Left, footerY)
	pdf.Cell(nil, time.Now().Format("2006-01-02"))

	// Center: Page numbers (if enabled)
	if pl.opts.ShowPageNumbers {
		centerX := pageWidth / 2
		pageText := fmt.Sprintf("Page %d", pageNum)
		if pl.totalPages > 0 {
			pageText = fmt.Sprintf("Page %d of %d", pageNum, pl.totalPages)
		}
		pdf.SetXY(centerX-30, footerY) // Approximate centering
		pdf.Cell(nil, pageText)
	}

	// Right: Confidential marker (if enabled)
	if pl.opts.Confidential {
		rightX := pageWidth - pl.opts.Margins.Right - 60
		pdf.SetXY(rightX, footerY)
		pdf.Cell(nil, "Confidential")
	}

	// Restore font and color
	pdf.SetTextColor(0, 0, 0)
	pl.fm.SetStyle(currentFont)
	pdf.SetFontSize(pl.fm.GetFontSize())

	return nil
}

// GetContentStartY returns the Y position where content should start
func (pl *PageLayout) GetContentStartY(pageNum int) float64 {
	if pageNum == 1 {
		return pl.opts.Margins.Top
	}
	// Leave space for header
	return pl.opts.Margins.Top + 20
}

// GetContentEndY returns the Y position where content should end
func (pl *PageLayout) GetContentEndY() float64 {
	// Leave space for footer
	return getPageHeight(pl.opts) - pl.opts.Margins.Bottom - 20
}

// getPageWidth returns the page width based on page size
func getPageWidth(opts *ConversionOptions) float64 {
	if opts.PageSize == "Letter" {
		return 612 // 8.5 inches * 72 points/inch
	}
	return 595 // A4: 210mm ≈ 8.27 inches
}
