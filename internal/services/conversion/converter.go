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

// ConversionOptions configures PDF generation
type ConversionOptions struct {
	// Document metadata
	Title    string            // Document title
	Author   string            // Document author
	Subject  string            // Document subject
	Metadata map[string]string // Additional metadata

	// Page layout
	PageSize string  // "A4" or "Letter"
	Margins  Margins // Page margins in points (1/72 inch)

	// Professional features
	HeaderImage       string  // Path to header image (PNG/JPEG)
	HeaderImageHeight float64 // Image height in points (default: 100pt)
	GenerateTOC       bool    // Generate table of contents
	TOCDepth          int     // Max heading level in TOC (default: 3)
	SyntaxTheme       string  // Chroma theme name (default: "github")
	ShowPageNumbers   bool    // Show page numbers in footer
	TaskRef           string  // Task reference for header display
	Window            string  // Collection window for header display
	Confidential      bool    // Show "Confidential" in footer

	// Font customization
	FontFamily     string  // Default font family (default: "Helvetica")
	MonoFontFamily string  // Monospace font family (default: "Courier")
	FontSize       float64 // Base font size in points (default: 11pt)
	CodeFontSize   float64 // Code block font size in points (default: 9pt)
}

// Margins defines page margins
type Margins struct {
	Top    float64
	Bottom float64
	Left   float64
	Right  float64
}

// Converter converts markdown files to PDF
type Converter interface {
	// ConvertMarkdownToPDF converts a markdown file to PDF
	ConvertMarkdownToPDF(inputPath, outputPath string, opts *ConversionOptions) error
}

// NewConverter creates a new markdown to PDF converter
// Uses goldmark (parser) + gopdf (renderer)
func NewConverter() Converter {
	return NewGoldmarkGoPDFConverter()
}

// DefaultOptions returns default conversion options
func DefaultOptions() *ConversionOptions {
	return &ConversionOptions{
		// Page layout
		PageSize: "A4",
		Margins: Margins{
			Top:    72, // 1 inch
			Bottom: 72, // 1 inch
			Left:   72, // 1 inch
			Right:  72, // 1 inch
		},

		// Professional features
		HeaderImageHeight: 100,      // 100 points (~1.4 inches)
		GenerateTOC:       true,     // Generate TOC by default
		TOCDepth:          3,        // H1, H2, H3 in TOC
		SyntaxTheme:       "github", // GitHub-style syntax highlighting
		ShowPageNumbers:   true,     // Show page numbers
		Confidential:      true,     // Mark as confidential

		// Fonts
		FontFamily:     "Helvetica", // System font
		MonoFontFamily: "Courier",   // System monospace font
		FontSize:       11,          // 11pt body text
		CodeFontSize:   9,           // 9pt code blocks
	}
}
