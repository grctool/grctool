// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/signintech/gopdf"
)

// FontStyle represents a font style
type FontStyle int

const (
	FontStyleRegular FontStyle = iota
	FontStyleBold
	FontStyleItalic
	FontStyleMono
)

// FontManager manages PDF fonts
type FontManager struct {
	pdf            *gopdf.GoPdf
	regularLoaded  bool
	boldLoaded     bool
	italicLoaded   bool
	monoLoaded     bool
	currentStyle   FontStyle
	fontSize       float64
	codeSize       float64
}

// NewFontManager creates a new font manager
func NewFontManager(pdf *gopdf.GoPdf, fontSize, codeSize float64) *FontManager {
	return &FontManager{
		pdf:          pdf,
		currentStyle: FontStyleRegular,
		fontSize:     fontSize,
		codeSize:     codeSize,
	}
}

// LoadFonts loads all required fonts
func (fm *FontManager) LoadFonts(fontFamily, monoFamily string) error {
	// Try to load system fonts
	systemFonts := []struct {
		name  string
		path  string
		style FontStyle
	}{
		{"regular", fm.findSystemFont(fontFamily, "Regular"), FontStyleRegular},
		{"bold", fm.findSystemFont(fontFamily, "Bold"), FontStyleBold},
		{"italic", fm.findSystemFont(fontFamily, "Italic"), FontStyleItalic},
		{"mono", fm.findSystemFont(monoFamily, "Regular"), FontStyleMono},
	}

	for _, font := range systemFonts {
		if font.path != "" {
			if err := fm.loadFont(font.name, font.path, font.style); err != nil {
				// Log warning but continue - will fall back to built-in fonts
				fmt.Printf("Warning: could not load %s font from %s: %v\n", font.name, font.path, err)
			}
		}
	}

	// If no fonts were loaded, use gopdf built-in fonts
	if !fm.regularLoaded && !fm.boldLoaded && !fm.italicLoaded && !fm.monoLoaded {
		return fm.loadBuiltInFonts()
	}

	return nil
}

// loadFont loads a TrueType font from a file
func (fm *FontManager) loadFont(name, path string, style FontStyle) error {
	if path == "" {
		return fmt.Errorf("font path is empty")
	}

	// Check if file exists
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return fmt.Errorf("font file not found: %s", path)
	}

	// Add TTF font to PDF
	if err := fm.pdf.AddTTFFont(name, path); err != nil {
		return fmt.Errorf("failed to add TTF font: %w", err)
	}

	// Mark as loaded
	switch style {
	case FontStyleRegular:
		fm.regularLoaded = true
	case FontStyleBold:
		fm.boldLoaded = true
	case FontStyleItalic:
		fm.italicLoaded = true
	case FontStyleMono:
		fm.monoLoaded = true
	}

	return nil
}

// loadBuiltInFonts uses gopdf's built-in fonts as fallback
func (fm *FontManager) loadBuiltInFonts() error {
	// gopdf has built-in support for standard PDF fonts
	// These don't need to be loaded explicitly
	fm.regularLoaded = true
	fm.boldLoaded = true
	fm.italicLoaded = true
	fm.monoLoaded = true
	return nil
}

// findSystemFont attempts to locate a system font file
func (fm *FontManager) findSystemFont(family, variant string) string {
	// Common font paths on macOS
	macPaths := []string{
		"/System/Library/Fonts",
		"/Library/Fonts",
		"~/Library/Fonts",
	}

	// Common font names
	fontNames := map[string]map[string]string{
		"Helvetica": {
			"Regular": "Helvetica.ttc",
			"Bold":    "Helvetica.ttc", // TTC contains multiple variants
			"Italic":  "Helvetica.ttc",
		},
		"Courier": {
			"Regular": "Courier New.ttf",
			"Bold":    "Courier New Bold.ttf",
			"Italic":  "Courier New Italic.ttf",
		},
	}

	// Get font filename for this family/variant
	if families, ok := fontNames[family]; ok {
		if filename, ok := families[variant]; ok {
			// Search in common locations
			for _, basePath := range macPaths {
				fullPath := filepath.Join(basePath, filename)
				if _, err := os.Stat(fullPath); err == nil {
					return fullPath
				}
			}
		}
	}

	return ""
}

// SetStyle changes the current font style
func (fm *FontManager) SetStyle(style FontStyle) error {
	if fm.currentStyle == style {
		return nil // Already set
	}

	var fontName string
	var fontSize float64

	switch style {
	case FontStyleRegular:
		if fm.regularLoaded {
			fontName = "regular"
		}
		fontSize = fm.fontSize

	case FontStyleBold:
		if fm.boldLoaded {
			fontName = "bold"
		}
		fontSize = fm.fontSize

	case FontStyleItalic:
		if fm.italicLoaded {
			fontName = "italic"
		}
		fontSize = fm.fontSize

	case FontStyleMono:
		if fm.monoLoaded {
			fontName = "mono"
		}
		fontSize = fm.codeSize
	}

	// Fallback to regular if specific style not loaded
	if fontName == "" {
		fontName = "regular"
	}

	// Set font in PDF
	if err := fm.pdf.SetFont(fontName, "", fontSize); err != nil {
		return fmt.Errorf("failed to set font: %w", err)
	}

	fm.currentStyle = style
	return nil
}

// SetFontSize changes the font size for current style
func (fm *FontManager) SetFontSize(size float64) {
	fm.pdf.SetFontSize(size)
}

// GetFontSize returns the current base font size
func (fm *FontManager) GetFontSize() float64 {
	return fm.fontSize
}

// GetCodeFontSize returns the code font size
func (fm *FontManager) GetCodeFontSize() float64 {
	return fm.codeSize
}
