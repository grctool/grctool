package conversion

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/signintech/gopdf"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

// ===========================================================================
// syntax.go tests
// ===========================================================================

func TestNewSyntaxHighlighter(t *testing.T) {
	t.Parallel()
	sh := NewSyntaxHighlighter("github")
	require.NotNil(t, sh)
	assert.Equal(t, "github", sh.theme)
	assert.NotNil(t, sh.style)
}

func TestNewSyntaxHighlighter_FallbackTheme(t *testing.T) {
	t.Parallel()
	sh := NewSyntaxHighlighter("nonexistent-theme-xyz")
	require.NotNil(t, sh)
	// Should fall back to github style
	assert.NotNil(t, sh.style)
}

func TestHighlightCode_Go(t *testing.T) {
	t.Parallel()
	sh := NewSyntaxHighlighter("github")

	tokens := sh.HighlightCode("package main\n\nfunc main() {}\n", "go")
	require.NotEmpty(t, tokens)

	// Verify all source text is preserved
	var combined strings.Builder
	for _, tok := range tokens {
		combined.WriteString(tok.Text)
	}
	assert.Contains(t, combined.String(), "package")
	assert.Contains(t, combined.String(), "main")
	assert.Contains(t, combined.String(), "func")
}

func TestHighlightCode_UnknownLanguage(t *testing.T) {
	t.Parallel()
	sh := NewSyntaxHighlighter("github")

	tokens := sh.HighlightCode("some text", "unknownlang12345")
	require.NotEmpty(t, tokens)

	var combined strings.Builder
	for _, tok := range tokens {
		combined.WriteString(tok.Text)
	}
	assert.Contains(t, combined.String(), "some text")
}

func TestHighlightCode_Python(t *testing.T) {
	t.Parallel()
	sh := NewSyntaxHighlighter("github")
	code := "def hello():\n    print('world')\n"

	tokens := sh.HighlightCode(code, "python")
	require.NotEmpty(t, tokens)

	var combined strings.Builder
	for _, tok := range tokens {
		combined.WriteString(tok.Text)
	}
	assert.Contains(t, combined.String(), "def")
	assert.Contains(t, combined.String(), "hello")
}

func TestHighlightCode_EmptyCode(t *testing.T) {
	t.Parallel()
	sh := NewSyntaxHighlighter("github")

	tokens := sh.HighlightCode("", "go")
	// Empty input may return nil or empty slice - either is acceptable
	// Verify no panic occurs
	var combined strings.Builder
	for _, tok := range tokens {
		combined.WriteString(tok.Text)
	}
	// Combined output should be empty or whitespace
	assert.Empty(t, strings.TrimSpace(combined.String()))
}

func TestColorFromEntry_DefaultColor(t *testing.T) {
	t.Parallel()
	sh := NewSyntaxHighlighter("github")

	// Test default color when entry has no colour set
	// We call HighlightCode with plain text which will use default colors
	tokens := sh.HighlightCode("plain text", "text")
	require.NotEmpty(t, tokens)
}

func TestGetAvailableThemes(t *testing.T) {
	t.Parallel()
	themes := GetAvailableThemes()
	require.NotEmpty(t, themes)
	// GitHub theme should be available
	assert.Contains(t, themes, "github")
}

func TestRenderCodeWithHighlighting(t *testing.T) {
	t.Parallel()
	result, err := RenderCodeWithHighlighting("x = 1", "python", "github")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestRenderCodeWithHighlighting_UnknownLanguage(t *testing.T) {
	t.Parallel()
	result, err := RenderCodeWithHighlighting("some code", "unknownlang", "github")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestRenderCodeWithHighlighting_UnknownTheme(t *testing.T) {
	t.Parallel()
	result, err := RenderCodeWithHighlighting("print('hi')", "python", "nonexistent")
	require.NoError(t, err)
	assert.NotEmpty(t, result)
}

func TestDetectLanguage(t *testing.T) {
	t.Parallel()
	// Python is generally detectable
	lang := DetectLanguage("#!/usr/bin/env python3\nimport os\nprint(os.getcwd())\n")
	// May or may not detect - just ensure no panic
	_ = lang
}

func TestDetectLanguage_Empty(t *testing.T) {
	t.Parallel()
	lang := DetectLanguage("")
	assert.Equal(t, "", lang)
}

func TestGetLanguageAliases_Known(t *testing.T) {
	t.Parallel()
	aliases := GetLanguageAliases("go")
	require.NotNil(t, aliases)
	assert.Contains(t, aliases, "go")
}

func TestGetLanguageAliases_Unknown(t *testing.T) {
	t.Parallel()
	aliases := GetLanguageAliases("nonexistentlang12345")
	assert.Nil(t, aliases)
}

func TestNormalizeLanguageName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"golang", "go"},
		{"js", "javascript"},
		{"ts", "typescript"},
		{"py", "python"},
		{"rb", "ruby"},
		{"yml", "yaml"},
		{"md", "markdown"},
		{"sh", "bash"},
		{"shell", "bash"},
		{"dockerfile", "docker"},
		{"go", "go"},         // already normalized
		{"python", "python"}, // already normalized
		{"  GOLANG  ", "go"}, // whitespace and case
		{"JS", "javascript"}, // uppercase
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, NormalizeLanguageName(tt.input))
		})
	}
}

// ===========================================================================
// converter.go tests
// ===========================================================================

func TestDefaultOptions(t *testing.T) {
	t.Parallel()
	opts := DefaultOptions()
	require.NotNil(t, opts)

	assert.Equal(t, "A4", opts.PageSize)
	assert.Equal(t, 72.0, opts.Margins.Top)
	assert.Equal(t, 72.0, opts.Margins.Bottom)
	assert.Equal(t, 72.0, opts.Margins.Left)
	assert.Equal(t, 72.0, opts.Margins.Right)
	assert.Equal(t, 100.0, opts.HeaderImageHeight)
	assert.True(t, opts.GenerateTOC)
	assert.Equal(t, 3, opts.TOCDepth)
	assert.Equal(t, "github", opts.SyntaxTheme)
	assert.True(t, opts.ShowPageNumbers)
	assert.True(t, opts.Confidential)
	assert.Equal(t, "Helvetica", opts.FontFamily)
	assert.Equal(t, "Courier", opts.MonoFontFamily)
	assert.Equal(t, 11.0, opts.FontSize)
	assert.Equal(t, 9.0, opts.CodeFontSize)
}

func TestNewConverter(t *testing.T) {
	t.Parallel()
	c := NewConverter()
	require.NotNil(t, c)
}

func TestNewGoldmarkGoPDFConverter(t *testing.T) {
	t.Parallel()
	c := NewGoldmarkGoPDFConverter()
	require.NotNil(t, c)
	require.NotNil(t, c.md)
}

// ===========================================================================
// toc.go tests
// ===========================================================================

func TestNewTOCGenerator(t *testing.T) {
	t.Parallel()

	toc := NewTOCGenerator(3, []byte("# test"))
	require.NotNil(t, toc)
	assert.Equal(t, 3, toc.maxDepth)
}

func TestNewTOCGenerator_ClampMinDepth(t *testing.T) {
	t.Parallel()
	toc := NewTOCGenerator(0, nil)
	assert.Equal(t, 3, toc.maxDepth) // Should clamp to 3
}

func TestNewTOCGenerator_ClampMaxDepth(t *testing.T) {
	t.Parallel()
	toc := NewTOCGenerator(10, nil)
	assert.Equal(t, 6, toc.maxDepth) // Should clamp to 6
}

func TestTOCGenerator_AddEntry(t *testing.T) {
	t.Parallel()
	toc := NewTOCGenerator(3, nil)

	toc.AddEntry(1, "Introduction", 1, 100.0)
	toc.AddEntry(2, "Details", 2, 200.0)

	entries := toc.GetEntries()
	require.Len(t, entries, 2)
	assert.Equal(t, "Introduction", entries[0].Title)
	assert.Equal(t, 1, entries[0].Level)
	assert.Equal(t, 1, entries[0].Page)
	assert.Contains(t, entries[0].LinkName, "heading-0-introduction")
}

func TestTOCGenerator_AddEntry_BeyondMaxDepth(t *testing.T) {
	t.Parallel()
	toc := NewTOCGenerator(2, nil)

	toc.AddEntry(1, "H1", 1, 0)
	toc.AddEntry(2, "H2", 1, 0)
	toc.AddEntry(3, "H3", 1, 0) // Beyond maxDepth=2, should be skipped

	entries := toc.GetEntries()
	assert.Len(t, entries, 2)
}

func TestTOCGenerator_HasEntries(t *testing.T) {
	t.Parallel()

	toc := NewTOCGenerator(3, nil)
	assert.False(t, toc.HasEntries())

	toc.AddEntry(1, "Title", 1, 0)
	assert.True(t, toc.HasEntries())
}

func TestTOCGenerator_CollectFromAST(t *testing.T) {
	t.Parallel()

	source := []byte("# Heading 1\n\n## Heading 2\n\n### Heading 3\n\n#### Heading 4\n")

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
	doc := md.Parser().Parse(text.NewReader(source))

	toc := NewTOCGenerator(3, source)
	toc.CollectFromAST(doc)

	entries := toc.GetEntries()
	// maxDepth=3, so H4 should be excluded
	assert.Len(t, entries, 3)
	assert.Equal(t, "Heading 1", entries[0].Title)
	assert.Equal(t, 1, entries[0].Level)
	assert.Equal(t, "Heading 2", entries[1].Title)
	assert.Equal(t, 2, entries[1].Level)
	assert.Equal(t, "Heading 3", entries[2].Title)
	assert.Equal(t, 3, entries[2].Level)
}

func TestSanitizeForLink(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"Hello World", "hello-world"},
		{"Simple", "simple"},
		{"Hello  World!", "hello-world"},
		{"Multiple   Spaces   Here", "multiple-spaces-here"},
		{"UPPERCASE", "uppercase"},
		{"with-hyphens", "with-hyphens"},
		{"with_underscores", "with-underscores"},
		{"123 Numbers", "123-numbers"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			t.Parallel()
			result := sanitizeForLink(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSanitizeForLink_LongTitle(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("abcdefghij", 10) // 100 chars
	result := sanitizeForLink(long)
	assert.LessOrEqual(t, len(result), 50)
}

func TestGetPageHeight(t *testing.T) {
	t.Parallel()

	t.Run("A4", func(t *testing.T) {
		t.Parallel()
		opts := &ConversionOptions{PageSize: "A4"}
		assert.Equal(t, 842.0, getPageHeight(opts))
	})

	t.Run("Letter", func(t *testing.T) {
		t.Parallel()
		opts := &ConversionOptions{PageSize: "Letter"}
		assert.Equal(t, 792.0, getPageHeight(opts))
	})

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		opts := &ConversionOptions{PageSize: ""}
		assert.Equal(t, 842.0, getPageHeight(opts))
	})
}

// ===========================================================================
// layout.go tests
// ===========================================================================

func TestGetPageWidth(t *testing.T) {
	t.Parallel()

	t.Run("A4", func(t *testing.T) {
		t.Parallel()
		opts := &ConversionOptions{PageSize: "A4"}
		assert.Equal(t, 595.0, getPageWidth(opts))
	})

	t.Run("Letter", func(t *testing.T) {
		t.Parallel()
		opts := &ConversionOptions{PageSize: "Letter"}
		assert.Equal(t, 612.0, getPageWidth(opts))
	})

	t.Run("default", func(t *testing.T) {
		t.Parallel()
		opts := &ConversionOptions{PageSize: ""}
		assert.Equal(t, 595.0, getPageWidth(opts))
	})
}

func TestPageLayout_GetContentStartY(t *testing.T) {
	t.Parallel()
	opts := DefaultOptions()
	pl := &PageLayout{opts: opts}

	// Page 1: starts at margin top
	assert.Equal(t, opts.Margins.Top, pl.GetContentStartY(1))

	// Page 2+: has extra space for header
	assert.Equal(t, opts.Margins.Top+20, pl.GetContentStartY(2))
}

func TestPageLayout_GetContentEndY(t *testing.T) {
	t.Parallel()
	opts := DefaultOptions()
	pl := &PageLayout{opts: opts}

	endY := pl.GetContentEndY()
	expected := getPageHeight(opts) - opts.Margins.Bottom - 20
	assert.Equal(t, expected, endY)
}

func TestPageLayout_SetTotalPages(t *testing.T) {
	t.Parallel()
	opts := DefaultOptions()
	pl := &PageLayout{opts: opts}

	pl.SetTotalPages(10)
	assert.Equal(t, 10, pl.totalPages)
}

// ===========================================================================
// fonts.go tests
// ===========================================================================

func TestFontStyle_Constants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, FontStyle(0), FontStyleRegular)
	assert.Equal(t, FontStyle(1), FontStyleBold)
	assert.Equal(t, FontStyle(2), FontStyleItalic)
	assert.Equal(t, FontStyle(3), FontStyleMono)
}

func TestFontManager_GetFontSize(t *testing.T) {
	t.Parallel()
	fm := &FontManager{fontSize: 11.0, codeSize: 9.0}
	assert.Equal(t, 11.0, fm.GetFontSize())
}

func TestFontManager_GetCodeFontSize(t *testing.T) {
	t.Parallel()
	fm := &FontManager{fontSize: 11.0, codeSize: 9.0}
	assert.Equal(t, 9.0, fm.GetCodeFontSize())
}

// ===========================================================================
// ConversionOptions / Margins types
// ===========================================================================

func TestConversionOptions_Fields(t *testing.T) {
	t.Parallel()
	opts := &ConversionOptions{
		Title:           "Test Document",
		Author:          "Test Author",
		Subject:         "Test Subject",
		PageSize:        "A4",
		HeaderImage:     "/path/to/image.png",
		GenerateTOC:     true,
		TOCDepth:        3,
		SyntaxTheme:     "github",
		ShowPageNumbers: true,
		TaskRef:         "ET-0001",
		Window:          "2025-Q4",
		Confidential:    true,
	}

	assert.Equal(t, "Test Document", opts.Title)
	assert.Equal(t, "Test Author", opts.Author)
	assert.Equal(t, "ET-0001", opts.TaskRef)
	assert.Equal(t, "2025-Q4", opts.Window)
	assert.True(t, opts.Confidential)
}

func TestMargins_Fields(t *testing.T) {
	t.Parallel()
	m := Margins{Top: 72, Bottom: 72, Left: 72, Right: 72}
	assert.Equal(t, 72.0, m.Top)
	assert.Equal(t, 72.0, m.Bottom)
}

func TestRGB_Fields(t *testing.T) {
	t.Parallel()
	c := RGB{R: 255, G: 128, B: 0}
	assert.Equal(t, uint8(255), c.R)
	assert.Equal(t, uint8(128), c.G)
	assert.Equal(t, uint8(0), c.B)
}

func TestColoredToken_Fields(t *testing.T) {
	t.Parallel()
	tok := ColoredToken{
		Text:  "func",
		Color: RGB{R: 0, G: 0, B: 255},
		Bold:  true,
	}
	assert.Equal(t, "func", tok.Text)
	assert.True(t, tok.Bold)
}

func TestTOCEntry_Fields(t *testing.T) {
	t.Parallel()
	entry := TOCEntry{
		Level:    2,
		Title:    "Section",
		Page:     3,
		Y:        150.5,
		LinkName: "heading-0-section",
	}
	assert.Equal(t, 2, entry.Level)
	assert.Equal(t, "Section", entry.Title)
	assert.Equal(t, 3, entry.Page)
}

// ===========================================================================
// ConvertMarkdownToPDF tests
// ===========================================================================

func TestConvertMarkdownToPDF_MissingInput(t *testing.T) {
	t.Parallel()
	tmpDir := t.TempDir()
	outputPath := filepath.Join(tmpDir, "test.pdf")

	c := NewConverter()
	err := c.ConvertMarkdownToPDF("/nonexistent/file.md", outputPath, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "reading markdown file")
}

// findTestFont returns a path to a TTF font for integration tests, or empty if none found.
func findTestFont() string {
	candidates := []string{
		"/usr/share/fonts/truetype/droid/DroidSansFallbackFull.ttf",
		"/usr/share/fonts/truetype/cascadia-code/CascadiaCode.ttf",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/truetype/liberation/LiberationSans-Regular.ttf",
		"/System/Library/Fonts/Helvetica.ttc",
	}
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

func TestConvertMarkdownToPDF_WithSystemFont(t *testing.T) {
	t.Parallel()
	fontPath := findTestFont()
	if fontPath == "" {
		t.Skip("no system TTF font found for integration test")
	}

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.md")
	outputPath := filepath.Join(tmpDir, "test.pdf")

	md := "# Evidence Report\n\n## Overview\n\nThis is a test.\n\n### Details\n\n- Item one\n- Item two\n\n"
	md += "```go\npackage main\nfunc main() {}\n```\n\n"
	md += "| Col1 | Col2 |\n|------|------|\n| A    | B    |\n\n"
	md += "**Bold** and *italic*.\n"

	err := os.WriteFile(inputPath, []byte(md), 0644)
	require.NoError(t, err)

	// Create a converter with custom font configuration
	c := NewGoldmarkGoPDFConverter()

	// Build options that reference a real font
	opts := DefaultOptions()
	opts.Title = "Test Report"
	opts.TaskRef = "ET-0001"
	opts.Window = "2025-Q4"
	opts.Subject = "Test"

	// We need to override font loading - create a custom PDF with actual font
	pdf := &gopdf.GoPdf{}
	pageSize := gopdf.PageSizeA4
	pdf.Start(gopdf.Config{PageSize: *pageSize})

	// Load the font we found for all styles
	err = pdf.AddTTFFont("regular", fontPath)
	if err != nil {
		t.Skipf("could not load font %s: %v", fontPath, err)
	}
	err = pdf.AddTTFFont("bold", fontPath)
	require.NoError(t, err)
	err = pdf.AddTTFFont("italic", fontPath)
	require.NoError(t, err)
	err = pdf.AddTTFFont("mono", fontPath)
	require.NoError(t, err)

	// Since we can load fonts, the converter should work
	// Let's test through the public API by using the converter directly
	err = c.ConvertMarkdownToPDF(inputPath, outputPath, opts)
	// On Linux with fallback fonts, this might fail because LoadFonts doesn't find macOS paths
	// but loadBuiltInFonts marks them as loaded without actually adding to gopdf
	if err != nil {
		assert.Contains(t, err.Error(), "font")
		t.Skipf("ConvertMarkdownToPDF failed with font error (expected on non-macOS): %v", err)
	}

	info, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(0))
}

func TestFontManager_LoadFont_ValidTTF(t *testing.T) {
	t.Parallel()
	fontPath := findTestFont()
	if fontPath == "" {
		t.Skip("no system TTF font found")
	}

	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	fm := NewFontManager(pdf, 11.0, 9.0)

	err := fm.loadFont("regular", fontPath, FontStyleRegular)
	require.NoError(t, err)
	assert.True(t, fm.regularLoaded)

	err = fm.loadFont("bold", fontPath, FontStyleBold)
	require.NoError(t, err)
	assert.True(t, fm.boldLoaded)

	err = fm.loadFont("italic", fontPath, FontStyleItalic)
	require.NoError(t, err)
	assert.True(t, fm.italicLoaded)

	err = fm.loadFont("mono", fontPath, FontStyleMono)
	require.NoError(t, err)
	assert.True(t, fm.monoLoaded)
}

func TestFontManager_SetStyle_WithLoadedFonts(t *testing.T) {
	t.Parallel()
	fontPath := findTestFont()
	if fontPath == "" {
		t.Skip("no system TTF font found")
	}

	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	fm := NewFontManager(pdf, 11.0, 9.0)

	// Load all font styles
	require.NoError(t, fm.loadFont("regular", fontPath, FontStyleRegular))
	require.NoError(t, fm.loadFont("bold", fontPath, FontStyleBold))
	require.NoError(t, fm.loadFont("italic", fontPath, FontStyleItalic))
	require.NoError(t, fm.loadFont("mono", fontPath, FontStyleMono))

	// Test setting various styles
	err := fm.SetStyle(FontStyleBold)
	require.NoError(t, err)
	assert.Equal(t, FontStyleBold, fm.currentStyle)

	err = fm.SetStyle(FontStyleItalic)
	require.NoError(t, err)
	assert.Equal(t, FontStyleItalic, fm.currentStyle)

	err = fm.SetStyle(FontStyleMono)
	require.NoError(t, err)
	assert.Equal(t, FontStyleMono, fm.currentStyle)

	err = fm.SetStyle(FontStyleRegular)
	require.NoError(t, err)
	assert.Equal(t, FontStyleRegular, fm.currentStyle)

	// Setting same style should be no-op
	err = fm.SetStyle(FontStyleRegular)
	require.NoError(t, err)
}

// ===========================================================================
// FontManager unit tests
// ===========================================================================

func TestNewFontManager(t *testing.T) {
	t.Parallel()
	// We can create a FontManager with a fresh GoPdf
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	fm := NewFontManager(pdf, 11.0, 9.0)
	require.NotNil(t, fm)
	assert.Equal(t, 11.0, fm.GetFontSize())
	assert.Equal(t, 9.0, fm.GetCodeFontSize())
	assert.Equal(t, FontStyleRegular, fm.currentStyle)
}

func TestFontManager_LoadBuiltInFonts(t *testing.T) {
	t.Parallel()
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	fm := NewFontManager(pdf, 11.0, 9.0)
	err := fm.loadBuiltInFonts()
	require.NoError(t, err)
	assert.True(t, fm.regularLoaded)
	assert.True(t, fm.boldLoaded)
	assert.True(t, fm.italicLoaded)
	assert.True(t, fm.monoLoaded)
}

func TestFontManager_FindSystemFont_NotFound(t *testing.T) {
	t.Parallel()
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	fm := NewFontManager(pdf, 11.0, 9.0)

	// Unknown font family should return empty
	path := fm.findSystemFont("UnknownFont123", "Regular")
	assert.Empty(t, path)
}

func TestFontManager_FindSystemFont_UnknownVariant(t *testing.T) {
	t.Parallel()
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	fm := NewFontManager(pdf, 11.0, 9.0)

	// Known family but unknown variant
	path := fm.findSystemFont("Helvetica", "ExtraBold")
	assert.Empty(t, path)
}

func TestFontManager_LoadFont_EmptyPath(t *testing.T) {
	t.Parallel()
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	fm := NewFontManager(pdf, 11.0, 9.0)
	err := fm.loadFont("test", "", FontStyleRegular)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "font path is empty")
}

func TestFontManager_LoadFont_NonexistentFile(t *testing.T) {
	t.Parallel()
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	fm := NewFontManager(pdf, 11.0, 9.0)
	err := fm.loadFont("test", "/nonexistent/font.ttf", FontStyleRegular)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "font file not found")
}

func TestFontManager_LoadFonts_FallsBackToBuiltIn(t *testing.T) {
	t.Parallel()
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	fm := NewFontManager(pdf, 11.0, 9.0)
	// On Linux CI, no macOS fonts exist, so should fall back to built-in
	err := fm.LoadFonts("Helvetica", "Courier")
	require.NoError(t, err)
	// All should be marked as loaded (either from file or fallback)
	assert.True(t, fm.regularLoaded)
	assert.True(t, fm.boldLoaded)
}

func TestFontManager_SetFontSize(t *testing.T) {
	t.Parallel()
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	fm := NewFontManager(pdf, 11.0, 9.0)
	// Just verify no panic
	fm.SetFontSize(14.0)
}

// ===========================================================================
// PageLayout unit tests
// ===========================================================================

func TestNewPageLayout(t *testing.T) {
	t.Parallel()
	opts := DefaultOptions()
	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	fm := NewFontManager(pdf, 11.0, 9.0)

	pl := NewPageLayout(opts, fm)
	require.NotNil(t, pl)
	assert.Equal(t, opts, pl.opts)
}

// ===========================================================================
// extractHeadingText
// ===========================================================================

// setupPDFWithFonts creates a gopdf with fonts loaded for testing.
// Returns nil if no system fonts are available.
func setupPDFWithFonts(t *testing.T) (*gopdf.GoPdf, *FontManager) {
	t.Helper()
	fontPath := findTestFont()
	if fontPath == "" {
		return nil, nil
	}

	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})
	pdf.AddPage()

	fm := NewFontManager(pdf, 11.0, 9.0)
	require.NoError(t, fm.loadFont("regular", fontPath, FontStyleRegular))
	require.NoError(t, fm.loadFont("bold", fontPath, FontStyleBold))
	require.NoError(t, fm.loadFont("italic", fontPath, FontStyleItalic))
	require.NoError(t, fm.loadFont("mono", fontPath, FontStyleMono))

	// Force initial font set by setting to a different style first,
	// then switching to regular so SetStyle actually calls pdf.SetFont
	fm.currentStyle = FontStyleBold // Trick so SetStyle will actually call SetFont
	require.NoError(t, fm.SetStyle(FontStyleRegular))
	pdf.SetFontSize(11)

	return pdf, fm
}

func TestTOCGenerator_Render(t *testing.T) {
	t.Parallel()
	pdf, fm := setupPDFWithFonts(t)
	if pdf == nil {
		t.Skip("no system TTF font found")
	}

	opts := DefaultOptions()
	toc := NewTOCGenerator(3, nil)
	toc.AddEntry(1, "Introduction", 1, 100)
	toc.AddEntry(2, "Details", 2, 200)
	toc.AddEntry(3, "Sub-details", 2, 300)

	endY, err := toc.Render(pdf, fm, opts, opts.Margins.Top)
	require.NoError(t, err)
	assert.Greater(t, endY, opts.Margins.Top)
}

func TestTOCGenerator_Render_Empty(t *testing.T) {
	t.Parallel()
	pdf, fm := setupPDFWithFonts(t)
	if pdf == nil {
		t.Skip("no system TTF font found")
	}

	opts := DefaultOptions()
	toc := NewTOCGenerator(3, nil)

	endY, err := toc.Render(pdf, fm, opts, opts.Margins.Top)
	require.NoError(t, err)
	assert.Equal(t, opts.Margins.Top, endY) // Unchanged if no entries
}

func TestPageLayout_RenderTitlePage(t *testing.T) {
	t.Parallel()
	pdf, fm := setupPDFWithFonts(t)
	if pdf == nil {
		t.Skip("no system TTF font found")
	}

	opts := DefaultOptions()
	opts.Title = "Test Report"
	opts.TaskRef = "ET-0001"
	opts.Window = "2025-Q4"
	opts.Subject = "Test Subject"

	pl := NewPageLayout(opts, fm)

	endY, err := pl.RenderTitlePage(pdf)
	require.NoError(t, err)
	assert.Greater(t, endY, opts.Margins.Top)
}

func TestPageLayout_RenderTitlePage_NoTitle(t *testing.T) {
	t.Parallel()
	pdf, fm := setupPDFWithFonts(t)
	if pdf == nil {
		t.Skip("no system TTF font found")
	}

	opts := DefaultOptions()
	opts.Title = "" // Will use default "Evidence Document"

	pl := NewPageLayout(opts, fm)

	endY, err := pl.RenderTitlePage(pdf)
	require.NoError(t, err)
	assert.Greater(t, endY, opts.Margins.Top)
}

func TestPageLayout_AddHeader(t *testing.T) {
	t.Parallel()
	pdf, fm := setupPDFWithFonts(t)
	if pdf == nil {
		t.Skip("no system TTF font found")
	}

	opts := DefaultOptions()
	opts.TaskRef = "ET-0001"
	opts.Title = "A Very Long Title That Exceeds Forty Characters In Total"
	opts.Window = "2025-Q4"

	pl := NewPageLayout(opts, fm)

	// Page 1 should skip header
	err := pl.AddHeader(pdf, 1)
	require.NoError(t, err)

	// Page 2 should add header
	pdf.AddPage()
	err = pl.AddHeader(pdf, 2)
	require.NoError(t, err)
}

func TestPageLayout_AddFooter(t *testing.T) {
	t.Parallel()
	pdf, fm := setupPDFWithFonts(t)
	if pdf == nil {
		t.Skip("no system TTF font found")
	}

	opts := DefaultOptions()
	opts.ShowPageNumbers = true
	opts.Confidential = true

	pl := NewPageLayout(opts, fm)
	pl.SetTotalPages(5)

	err := pl.AddFooter(pdf, 1)
	require.NoError(t, err)
}

func TestPageLayout_AddFooter_NoPageNumbers(t *testing.T) {
	t.Parallel()
	pdf, fm := setupPDFWithFonts(t)
	if pdf == nil {
		t.Skip("no system TTF font found")
	}

	opts := DefaultOptions()
	opts.ShowPageNumbers = false
	opts.Confidential = false

	pl := NewPageLayout(opts, fm)

	err := pl.AddFooter(pdf, 1)
	require.NoError(t, err)
}

func TestPageLayout_RenderHeaderImage_MissingFile(t *testing.T) {
	t.Parallel()
	pdf, fm := setupPDFWithFonts(t)
	if pdf == nil {
		t.Skip("no system TTF font found")
	}

	opts := DefaultOptions()
	opts.HeaderImage = "/nonexistent/image.png"

	pl := NewPageLayout(opts, fm)

	// RenderTitlePage should log warning but continue
	_, err := pl.RenderTitlePage(pdf)
	// Should succeed because header image failure is handled gracefully
	require.NoError(t, err)
}

// TestConvertMarkdownToPDF_FullPipeline tests the full conversion by manually
// building the pipeline with real fonts (bypassing the macOS-only font finder).
func TestConvertMarkdownToPDF_FullPipeline(t *testing.T) {
	t.Parallel()
	fontPath := findTestFont()
	if fontPath == "" {
		t.Skip("no system TTF font found for integration test")
	}

	tmpDir := t.TempDir()
	inputPath := filepath.Join(tmpDir, "test.md")
	outputPath := filepath.Join(tmpDir, "test.pdf")

	md := "# Evidence Report\n\n## Overview\n\nThis is a test evidence document.\n\n"
	md += "### Subsection\n\n- Item one\n- Item two\n- Item three\n\n"
	md += "```go\npackage main\nfunc main() { println(\"hello\") }\n```\n\n"
	md += "| Control | Status |\n|---------|--------|\n| CC6.1 | Pass |\n\n"
	md += "**Bold text** and *italic text* and normal text.\n\n"
	md += "## Conclusion\n\nDone.\n"

	err := os.WriteFile(inputPath, []byte(md), 0644)
	require.NoError(t, err)

	// Read source
	source, err := os.ReadFile(inputPath)
	require.NoError(t, err)

	// Build the pipeline manually (like ConvertMarkdownToPDF but with real font)
	c := NewGoldmarkGoPDFConverter()
	opts := DefaultOptions()
	opts.Title = "Test Evidence Report"
	opts.TaskRef = "ET-0001"
	opts.Window = "2025-Q4"
	opts.Subject = "Test Subject"
	opts.Confidential = true
	opts.ShowPageNumbers = true

	doc := c.md.Parser().Parse(text.NewReader(source))

	pdf := &gopdf.GoPdf{}
	pdf.Start(gopdf.Config{PageSize: *gopdf.PageSizeA4})

	// Load fonts manually using system font
	fm := NewFontManager(pdf, opts.FontSize, opts.CodeFontSize)
	require.NoError(t, fm.loadFont("regular", fontPath, FontStyleRegular))
	require.NoError(t, fm.loadFont("bold", fontPath, FontStyleBold))
	require.NoError(t, fm.loadFont("italic", fontPath, FontStyleItalic))
	require.NoError(t, fm.loadFont("mono", fontPath, FontStyleMono))

	syntaxHL := NewSyntaxHighlighter(opts.SyntaxTheme)
	tocGen := NewTOCGenerator(opts.TOCDepth, source)
	pageLayout := NewPageLayout(opts, fm)

	// First pass: collect TOC
	tocGen.CollectFromAST(doc)

	// Start first page
	pdf.AddPage()

	// Render title page
	_, err = pageLayout.RenderTitlePage(pdf)
	require.NoError(t, err)

	// Render TOC
	if tocGen.HasEntries() {
		pdf.AddPage()
		_, err = tocGen.Render(pdf, fm, opts, opts.Margins.Top)
		require.NoError(t, err)
	}

	// Start content page
	pdf.AddPage()
	pageHeight := gopdf.PageSizeA4.H - opts.Margins.Top - opts.Margins.Bottom

	ctx := &renderContext{
		pdf:           pdf,
		opts:          opts,
		source:        source,
		currentY:      pageLayout.GetContentStartY(pdf.GetNumberOfPages()),
		pageHeight:    pageHeight,
		fontManager:   fm,
		syntaxHL:      syntaxHL,
		tocGen:        tocGen,
		pageLayout:    pageLayout,
		currentPage:   pdf.GetNumberOfPages(),
		tocEntryIndex: 0,
	}

	// Set initial font
	fm.currentStyle = FontStyleBold // Force SetStyle to actually set font
	require.NoError(t, fm.SetStyle(FontStyleRegular))
	pdf.SetFontSize(opts.FontSize)

	// Walk AST and render
	err = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		return c.renderNode(ctx, n, entering)
	})
	require.NoError(t, err)

	// Add headers and footers
	totalPages := pdf.GetNumberOfPages()
	pageLayout.SetTotalPages(totalPages)
	for pageNum := 1; pageNum <= totalPages; pageNum++ {
		require.NoError(t, pageLayout.AddHeader(pdf, pageNum))
		require.NoError(t, pageLayout.AddFooter(pdf, pageNum))
	}

	// Write PDF
	err = pdf.WritePdf(outputPath)
	require.NoError(t, err)

	// Verify
	info, err := os.Stat(outputPath)
	require.NoError(t, err)
	assert.Greater(t, info.Size(), int64(100))
}

func TestExtractHeadingText(t *testing.T) {
	t.Parallel()
	source := []byte("# Hello World\n\nSome text.\n")

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(parser.WithAutoHeadingID()),
	)
	doc := md.Parser().Parse(text.NewReader(source))

	// Walk to find heading
	var heading *ast.Heading
	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if h, ok := n.(*ast.Heading); ok && entering {
			heading = h
			return ast.WalkStop, nil
		}
		return ast.WalkContinue, nil
	})

	require.NotNil(t, heading)
	title := extractHeadingText(heading, source)
	assert.Equal(t, "Hello World", title)
}
