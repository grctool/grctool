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

package tools

import (
	"bufio"
	"context"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
)

// DocsReaderTool provides documentation file searching and analysis capabilities
type DocsReaderTool struct {
	config   *config.Config
	logger   logger.Logger
	cacheDir string
}

// NewDocsReaderTool creates a new documentation reader tool
func NewDocsReaderTool(cfg *config.Config, log logger.Logger) Tool {
	// Set up cache directory
	cacheDir := filepath.Join(cfg.Storage.DataDir, "docs_cache")

	return &DocsReaderTool{
		config:   cfg,
		logger:   log,
		cacheDir: cacheDir,
	}
}

// Name returns the tool name
func (drt *DocsReaderTool) Name() string {
	return "docs-reader"
}

// Description returns the tool description
func (drt *DocsReaderTool) Description() string {
	return "Search and analyze documentation files with keyword relevance scoring and section extraction"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (drt *DocsReaderTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        drt.Name(),
		Description: drt.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"pattern": map[string]interface{}{
					"type":        "string",
					"description": "Glob pattern for files to search (e.g., '*.md', '**/*.txt')",
					"default":     "*.md",
				},
				"query": map[string]interface{}{
					"type":        "string",
					"description": "Search query/keywords to find in documentation",
				},
				"docs_path": map[string]interface{}{
					"type":        "string",
					"description": "Path to documentation directory to search",
					"default":     "docs/",
				},
				"min_relevance": map[string]interface{}{
					"type":        "number",
					"description": "Minimum relevance score to include results (0.0-1.0)",
					"minimum":     0.0,
					"maximum":     1.0,
					"default":     0.1,
				},
				"max_results": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of results to return",
					"minimum":     1,
					"maximum":     100,
					"default":     20,
				},
				"extract_sections": map[string]interface{}{
					"type":        "boolean",
					"description": "Extract relevant sections from markdown files",
					"default":     true,
				},
			},
			"required": []string{"query"},
		},
	}
}

// Execute runs the docs reader tool with the given parameters
func (drt *DocsReaderTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	drt.logger.Debug("Executing docs reader", logger.Field{Key: "params", Value: params})

	// Extract parameters with defaults
	pattern := "*.md"
	if p, ok := params["pattern"].(string); ok && p != "" {
		pattern = p
	}

	query, ok := params["query"].(string)
	if !ok || query == "" {
		return "", nil, fmt.Errorf("query parameter is required")
	}

	docsPath := "docs/"
	if dp, ok := params["docs_path"].(string); ok && dp != "" {
		docsPath = dp
	}

	// Resolve docs path relative to DataDir if not absolute
	if !filepath.IsAbs(docsPath) {
		docsPath = filepath.Join(drt.config.Storage.DataDir, docsPath)
	}

	minRelevance := 0.1
	if mr, ok := params["min_relevance"].(float64); ok {
		minRelevance = mr
	}

	maxResults := 20
	if mr, ok := params["max_results"].(int); ok {
		maxResults = mr
	}

	extractSections := true
	if es, ok := params["extract_sections"].(bool); ok {
		extractSections = es
	}

	// Search for documents
	results, err := drt.searchDocuments(ctx, docsPath, pattern, query, minRelevance, maxResults, extractSections)
	if err != nil {
		return "", nil, fmt.Errorf("failed to search documents: %w", err)
	}

	// Generate report
	report := drt.generateReport(results, query, pattern, docsPath)

	// Calculate overall relevance
	overallRelevance := drt.calculateOverallRelevance(results)

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "docs-reader",
		Resource:    fmt.Sprintf("Documentation in %s matching %s", docsPath, pattern),
		Content:     report,
		Relevance:   overallRelevance,
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"docs_path":        docsPath,
			"pattern":          pattern,
			"query":            query,
			"min_relevance":    minRelevance,
			"extract_sections": extractSections,
			"results_count":    len(results),
			"avg_relevance":    overallRelevance,
		},
	}

	return report, source, nil
}

// searchDocuments performs the actual document search
func (drt *DocsReaderTool) searchDocuments(ctx context.Context, docsPath, pattern, query string, minRelevance float64, maxResults int, extractSections bool) ([]DocSearchResult, error) {
	var results []DocSearchResult

	// Find matching files
	files, err := drt.findMatchingFiles(docsPath, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to find matching files: %w", err)
	}

	drt.logger.Debug("Found matching files", logger.Int("count", len(files)))

	// Process each file
	for _, filePath := range files {
		// Check context cancellation
		select {
		case <-ctx.Done():
			return results, ctx.Err()
		default:
		}

		result, err := drt.analyzeDocument(filePath, query, extractSections)
		if err != nil {
			drt.logger.Warn("Failed to analyze document",
				logger.String("file", filePath),
				logger.Field{Key: "error", Value: err})
			continue
		}

		if result.Relevance >= minRelevance {
			results = append(results, *result)
		}
	}

	// Sort by relevance (highest first)
	sort.Slice(results, func(i, j int) bool {
		return results[i].Relevance > results[j].Relevance
	})

	// Limit results
	if len(results) > maxResults {
		results = results[:maxResults]
	}

	return results, nil
}

// findMatchingFiles finds files matching the given pattern
func (drt *DocsReaderTool) findMatchingFiles(docsPath, pattern string) ([]string, error) {
	var files []string

	// Handle glob patterns
	if strings.Contains(pattern, "*") {
		searchPattern := filepath.Join(docsPath, pattern)
		matches, err := filepath.Glob(searchPattern)
		if err != nil {
			return nil, fmt.Errorf("glob pattern error: %w", err)
		}

		// For recursive patterns like **/*.md, we need to walk the directory
		if strings.Contains(pattern, "**") {
			basePath := docsPath
			err := filepath.Walk(basePath, func(path string, info os.FileInfo, err error) error {
				if err != nil {
					return err
				}

				if info.IsDir() {
					return nil
				}

				// Check if file matches pattern
				_, _ = filepath.Rel(basePath, path) // relPath not used but kept for future reference
				if matched, _ := filepath.Match(strings.TrimPrefix(pattern, "**"+string(filepath.Separator)), filepath.Base(path)); matched {
					files = append(files, path)
				}

				return nil
			})
			if err != nil {
				return nil, fmt.Errorf("walk error: %w", err)
			}
		} else {
			files = matches
		}
	} else {
		// Direct file or simple pattern
		fullPath := filepath.Join(docsPath, pattern)
		if _, err := os.Stat(fullPath); err == nil {
			files = append(files, fullPath)
		}
	}

	// Filter out directories and ensure files exist
	var validFiles []string
	for _, file := range files {
		if info, err := os.Stat(file); err == nil && !info.IsDir() {
			validFiles = append(validFiles, file)
		}
	}

	return validFiles, nil
}

// analyzeDocument analyzes a single document for relevance to the query
func (drt *DocsReaderTool) analyzeDocument(filePath, query string, extractSections bool) (*DocSearchResult, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	result := &DocSearchResult{
		FilePath:      filePath,
		FileName:      filepath.Base(filePath),
		FileType:      strings.ToLower(filepath.Ext(filePath)),
		AnalyzedAt:    time.Now(),
		MatchingLines: []LineMatch{},
		Sections:      []DocumentSection{},
	}

	// Prepare query terms for matching
	queryTerms := drt.prepareQueryTerms(query)

	scanner := bufio.NewScanner(file)
	lineNum := 0
	totalLines := 0
	matchingLinesCount := 0
	var currentSection *DocumentSection

	for scanner.Scan() {
		lineNum++
		totalLines++
		line := scanner.Text()
		trimmedLine := strings.TrimSpace(line)

		// Handle markdown sections if extract_sections is enabled
		if extractSections && result.FileType == ".md" {
			if strings.HasPrefix(trimmedLine, "#") {
				// Save previous section if it exists
				if currentSection != nil && len(currentSection.Content) > 0 {
					currentSection.Relevance = drt.calculateSectionRelevance(currentSection.Content, queryTerms)
					if currentSection.Relevance > 0 {
						result.Sections = append(result.Sections, *currentSection)
					}
				}

				// Start new section
				level := 0
				for _, char := range trimmedLine {
					if char == '#' {
						level++
					} else {
						break
					}
				}

				title := strings.TrimSpace(strings.TrimLeft(trimmedLine, "#"))
				currentSection = &DocumentSection{
					Title:     title,
					Level:     level,
					StartLine: lineNum,
					Content:   []string{},
				}
			} else if currentSection != nil {
				currentSection.Content = append(currentSection.Content, line)
				currentSection.EndLine = lineNum
			}
		}

		// Check line for query matches
		lineRelevance := drt.calculateLineRelevance(line, queryTerms)
		if lineRelevance > 0 {
			matchingLinesCount++
			match := LineMatch{
				LineNumber: lineNum,
				Content:    line,
				Relevance:  lineRelevance,
				Context:    drt.getLineContext(scanner, lineNum),
			}
			result.MatchingLines = append(result.MatchingLines, match)
		}
	}

	// Handle last section if extracting sections
	if extractSections && currentSection != nil && len(currentSection.Content) > 0 {
		currentSection.Relevance = drt.calculateSectionRelevance(currentSection.Content, queryTerms)
		if currentSection.Relevance > 0 {
			result.Sections = append(result.Sections, *currentSection)
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error scanning file: %w", err)
	}

	// Calculate overall document relevance
	result.TotalLines = totalLines
	result.MatchingLinesCount = matchingLinesCount
	result.Relevance = drt.calculateDocumentRelevance(result, queryTerms)

	// Add file metadata
	if info, err := os.Stat(filePath); err == nil {
		result.FileSize = info.Size()
		result.ModifiedAt = info.ModTime()
	}

	return result, nil
}

// prepareQueryTerms processes the query into searchable terms
func (drt *DocsReaderTool) prepareQueryTerms(query string) []string {
	// Split query into terms and clean them
	terms := strings.Fields(strings.ToLower(query))
	var cleanedTerms []string

	for _, term := range terms {
		// Remove punctuation and keep alphanumeric characters
		cleaned := regexp.MustCompile(`[^a-z0-9]+`).ReplaceAllString(term, "")
		if len(cleaned) >= 2 { // Ignore very short terms
			cleanedTerms = append(cleanedTerms, cleaned)
		}
	}

	return cleanedTerms
}

// calculateLineRelevance calculates how relevant a line is to the query
func (drt *DocsReaderTool) calculateLineRelevance(line string, queryTerms []string) float64 {
	if len(queryTerms) == 0 {
		return 0.0
	}

	lineLower := strings.ToLower(line)
	matchCount := 0
	totalMatches := 0

	for _, term := range queryTerms {
		matches := strings.Count(lineLower, term)
		if matches > 0 {
			matchCount++
			totalMatches += matches
		}
	}

	if matchCount == 0 {
		return 0.0
	}

	// Base relevance on term coverage
	termCoverage := float64(matchCount) / float64(len(queryTerms))

	// Bonus for multiple occurrences
	densityBonus := math.Min(float64(totalMatches)/float64(len(queryTerms)), 1.0) * 0.3

	// Bonus for exact phrase matches
	phraseBonus := 0.0
	if strings.Contains(lineLower, strings.Join(queryTerms, " ")) {
		phraseBonus = 0.3
	}

	relevance := (termCoverage * 0.7) + densityBonus + phraseBonus
	return math.Min(relevance, 1.0)
}

// calculateSectionRelevance calculates relevance for a document section
func (drt *DocsReaderTool) calculateSectionRelevance(content []string, queryTerms []string) float64 {
	if len(queryTerms) == 0 || len(content) == 0 {
		return 0.0
	}

	totalRelevance := 0.0
	lineCount := 0

	for _, line := range content {
		lineRelevance := drt.calculateLineRelevance(line, queryTerms)
		if lineRelevance > 0 {
			totalRelevance += lineRelevance
			lineCount++
		}
	}

	if lineCount == 0 {
		return 0.0
	}

	// Average relevance with bonus for multiple matching lines
	avgRelevance := totalRelevance / float64(lineCount)
	coverageBonus := math.Min(float64(lineCount)/float64(len(content)), 0.5) * 0.2

	return math.Min(avgRelevance+coverageBonus, 1.0)
}

// calculateDocumentRelevance calculates overall document relevance
func (drt *DocsReaderTool) calculateDocumentRelevance(result *DocSearchResult, queryTerms []string) float64 {
	if result.TotalLines == 0 || len(queryTerms) == 0 {
		return 0.0
	}

	// Base relevance on matching line density
	lineDensity := float64(result.MatchingLinesCount) / float64(result.TotalLines)
	densityScore := math.Min(lineDensity*10, 1.0) // Scale up density

	// Average line relevance
	lineRelevanceTotal := 0.0
	for _, match := range result.MatchingLines {
		lineRelevanceTotal += match.Relevance
	}

	avgLineRelevance := 0.0
	if len(result.MatchingLines) > 0 {
		avgLineRelevance = lineRelevanceTotal / float64(len(result.MatchingLines))
	}

	// Section relevance (for markdown files)
	sectionRelevance := 0.0
	if len(result.Sections) > 0 {
		sectionTotal := 0.0
		for _, section := range result.Sections {
			sectionTotal += section.Relevance
		}
		sectionRelevance = sectionTotal / float64(len(result.Sections))
	}

	// File name relevance
	fileNameRelevance := drt.calculateLineRelevance(result.FileName, queryTerms) * 0.3

	// Combine all factors
	combinedRelevance := (densityScore * 0.3) + (avgLineRelevance * 0.4) + (sectionRelevance * 0.2) + (fileNameRelevance * 0.1)

	return math.Min(combinedRelevance, 1.0)
}

// getLineContext provides context around a matching line
func (drt *DocsReaderTool) getLineContext(scanner *bufio.Scanner, lineNum int) string {
	// This is a simplified context extraction
	// In a full implementation, we'd need to re-read the file or buffer content
	return fmt.Sprintf("Line %d context", lineNum)
}

// calculateOverallRelevance calculates the overall relevance of all results
func (drt *DocsReaderTool) calculateOverallRelevance(results []DocSearchResult) float64 {
	if len(results) == 0 {
		return 0.0
	}

	totalRelevance := 0.0
	for _, result := range results {
		totalRelevance += result.Relevance
	}

	avgRelevance := totalRelevance / float64(len(results))

	// Bonus for finding multiple relevant documents
	if len(results) >= 5 {
		avgRelevance += 0.1
	} else if len(results) >= 2 {
		avgRelevance += 0.05
	}

	return math.Min(avgRelevance, 1.0)
}

// generateReport creates a formatted report from document search results
func (drt *DocsReaderTool) generateReport(results []DocSearchResult, query, pattern, docsPath string) string {
	var report strings.Builder

	report.WriteString("# Documentation Search Results\n\n")
	report.WriteString(fmt.Sprintf("**Search Query**: %s\n", query))
	report.WriteString(fmt.Sprintf("**Search Path**: %s\n", docsPath))
	report.WriteString(fmt.Sprintf("**File Pattern**: %s\n", pattern))
	report.WriteString(fmt.Sprintf("**Results Found**: %d\n", len(results)))
	report.WriteString(fmt.Sprintf("**Generated**: %s\n\n", time.Now().Format(time.RFC3339)))

	if len(results) == 0 {
		report.WriteString("No documents found matching the search criteria.\n")
		return report.String()
	}

	for i, result := range results {
		report.WriteString(fmt.Sprintf("## Result %d: %s\n\n", i+1, result.FileName))
		report.WriteString(fmt.Sprintf("- **File Path**: %s\n", result.FilePath))
		report.WriteString(fmt.Sprintf("- **File Type**: %s\n", result.FileType))
		report.WriteString(fmt.Sprintf("- **Relevance Score**: %.3f\n", result.Relevance))
		report.WriteString(fmt.Sprintf("- **File Size**: %d bytes\n", result.FileSize))
		report.WriteString(fmt.Sprintf("- **Modified**: %s\n", result.ModifiedAt.Format("2006-01-02 15:04:05")))
		report.WriteString(fmt.Sprintf("- **Matching Lines**: %d/%d\n\n", result.MatchingLinesCount, result.TotalLines))

		// Show top matching lines
		if len(result.MatchingLines) > 0 {
			report.WriteString("**Top Matching Lines:**\n\n")
			maxLines := 5
			if len(result.MatchingLines) < maxLines {
				maxLines = len(result.MatchingLines)
			}

			for j := 0; j < maxLines; j++ {
				match := result.MatchingLines[j]
				report.WriteString(fmt.Sprintf("- Line %d (%.2f): %s\n", match.LineNumber, match.Relevance, strings.TrimSpace(match.Content)))
			}
			report.WriteString("\n")
		}

		// Show relevant sections for markdown files
		if len(result.Sections) > 0 {
			report.WriteString("**Relevant Sections:**\n\n")
			for _, section := range result.Sections {
				if section.Relevance > 0.1 { // Only show reasonably relevant sections
					report.WriteString(fmt.Sprintf("- **%s** (Level %d, Lines %d-%d, Relevance: %.2f)\n",
						section.Title, section.Level, section.StartLine, section.EndLine, section.Relevance))
				}
			}
			report.WriteString("\n")
		}

		report.WriteString("---\n\n")
	}

	return report.String()
}

// DocSearchResult represents a document search result
type DocSearchResult struct {
	FilePath           string            `json:"file_path"`
	FileName           string            `json:"file_name"`
	FileType           string            `json:"file_type"`
	FileSize           int64             `json:"file_size"`
	ModifiedAt         time.Time         `json:"modified_at"`
	AnalyzedAt         time.Time         `json:"analyzed_at"`
	Relevance          float64           `json:"relevance"`
	TotalLines         int               `json:"total_lines"`
	MatchingLinesCount int               `json:"matching_lines_count"`
	MatchingLines      []LineMatch       `json:"matching_lines"`
	Sections           []DocumentSection `json:"sections,omitempty"`
}

// LineMatch represents a matching line in a document
type LineMatch struct {
	LineNumber int     `json:"line_number"`
	Content    string  `json:"content"`
	Relevance  float64 `json:"relevance"`
	Context    string  `json:"context,omitempty"`
}

// DocumentSection represents a section in a markdown document
type DocumentSection struct {
	Title     string   `json:"title"`
	Level     int      `json:"level"`
	StartLine int      `json:"start_line"`
	EndLine   int      `json:"end_line"`
	Content   []string `json:"content"`
	Relevance float64  `json:"relevance"`
}
