// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package conversion

import (
	"bytes"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/alecthomas/chroma/v2/styles"
)

// ColoredToken represents a syntax-highlighted token
type ColoredToken struct {
	Text  string
	Color RGB
	Bold  bool
}

// RGB represents an RGB color
type RGB struct {
	R, G, B uint8
}

// SyntaxHighlighter highlights code using chroma
type SyntaxHighlighter struct {
	theme string
	style *chroma.Style
}

// NewSyntaxHighlighter creates a new syntax highlighter
func NewSyntaxHighlighter(themeName string) *SyntaxHighlighter {
	// Get style by name, fallback to github
	style := styles.Get(themeName)
	if style == nil {
		style = styles.GitHub
	}

	return &SyntaxHighlighter{
		theme: themeName,
		style: style,
	}
}

// HighlightCode highlights source code and returns colored tokens
func (sh *SyntaxHighlighter) HighlightCode(code, language string) []ColoredToken {
	// Get lexer for the language
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	// Tokenize the code
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		// If tokenization fails, return plain text
		return []ColoredToken{{Text: code, Color: RGB{60, 60, 60}}}
	}

	// Convert chroma tokens to our colored tokens
	var tokens []ColoredToken
	for _, token := range iterator.Tokens() {
		// Get style for this token type
		entry := sh.style.Get(token.Type)

		// Extract color
		color := sh.colorFromEntry(entry)

		// Determine if bold
		bold := entry.Bold == chroma.Yes

		tokens = append(tokens, ColoredToken{
			Text:  token.Value,
			Color: color,
			Bold:  bold,
		})
	}

	return tokens
}

// colorFromEntry extracts RGB color from chroma style entry
func (sh *SyntaxHighlighter) colorFromEntry(entry chroma.StyleEntry) RGB {
	// Default to dark gray
	defaultColor := RGB{R: 60, G: 60, B: 60}

	if entry.Colour.IsSet() {
		// Extract RGB from chroma.Colour
		// Colour is stored as uint32: 0x00RRGGBB
		color := uint32(entry.Colour)
		r := uint8((color >> 16) & 0xFF)
		g := uint8((color >> 8) & 0xFF)
		b := uint8(color & 0xFF)
		return RGB{R: r, G: g, B: b}
	}

	return defaultColor
}

// GetAvailableThemes returns list of available chroma themes
func GetAvailableThemes() []string {
	return styles.Names()
}

// RenderCodeWithHighlighting renders highlighted code to string (for testing)
func RenderCodeWithHighlighting(code, language, themeName string) (string, error) {
	// Get lexer
	lexer := lexers.Get(language)
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	// Get style
	style := styles.Get(themeName)
	if style == nil {
		style = styles.GitHub
	}

	// Create formatter
	formatter := formatters.Get("terminal256")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	// Tokenize
	iterator, err := lexer.Tokenise(nil, code)
	if err != nil {
		return "", err
	}

	// Format
	var buf bytes.Buffer
	err = formatter.Format(&buf, style, iterator)
	if err != nil {
		return "", err
	}

	return buf.String(), nil
}

// DetectLanguage attempts to detect language from code
func DetectLanguage(code string) string {
	// Try to analyze the code
	lexer := lexers.Analyse(code)
	if lexer != nil {
		config := lexer.Config()
		if config != nil && len(config.Aliases) > 0 {
			return config.Aliases[0]
		}
	}

	return ""
}

// GetLanguageAliases returns common aliases for a language
func GetLanguageAliases(language string) []string {
	lexer := lexers.Get(language)
	if lexer != nil {
		config := lexer.Config()
		if config != nil {
			return config.Aliases
		}
	}
	return nil
}

// NormalizeLanguageName converts common language names to chroma-recognized names
func NormalizeLanguageName(lang string) string {
	// Common mappings
	normalizations := map[string]string{
		"golang":     "go",
		"js":         "javascript",
		"ts":         "typescript",
		"py":         "python",
		"rb":         "ruby",
		"yml":        "yaml",
		"md":         "markdown",
		"sh":         "bash",
		"shell":      "bash",
		"dockerfile": "docker",
	}

	lang = strings.ToLower(strings.TrimSpace(lang))
	if normalized, ok := normalizations[lang]; ok {
		return normalized
	}

	return lang
}
