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

package evidence

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/utils"
)

// SourceFileCollector handles collection of source files and snippet extraction
type SourceFileCollector struct {
	dataDir        string
	gitRoot        string
	contextLines   int // Number of context lines before/after snippet
	maxSnippetSize int // Maximum snippet size in lines
}

// NewSourceFileCollector creates a new source file collector
func NewSourceFileCollector(dataDir string) *SourceFileCollector {
	return &SourceFileCollector{
		dataDir:        dataDir,
		contextLines:   5,  // Default: 5 lines before/after
		maxSnippetSize: 50, // Default: max 50 lines per snippet
	}
}

// SetGitRoot sets the git repository root for relative path calculation
func (s *SourceFileCollector) SetGitRoot(gitRoot string) {
	s.gitRoot = gitRoot
}

// SetContextLines sets the number of context lines for snippet extraction
func (s *SourceFileCollector) SetContextLines(lines int) {
	s.contextLines = lines
}

// CollectSourceFile collects a source file with optional snippet extraction
// Returns a SourceFileRef with file metadata and optional snippet
func (s *SourceFileCollector) CollectSourceFile(
	sourcePath string,
	lineRange string, // Optional: "45-67" or "123" or ""
	destDir string, // Where to copy the file
) (*models.SourceFileRef, error) {
	if sourcePath == "" {
		return nil, fmt.Errorf("source path is empty")
	}

	// Get absolute path
	absSourcePath, err := filepath.Abs(sourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve source path: %w", err)
	}

	// Check if file exists
	fileInfo, err := os.Stat(absSourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat source file: %w", err)
	}

	// Calculate checksum
	checksum, err := s.calculateChecksum(absSourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	// Determine original path (relative to git root or data dir)
	originalPath, err := s.getRelativePath(absSourcePath)
	if err != nil {
		return nil, fmt.Errorf("failed to get relative path: %w", err)
	}

	sourceRef := &models.SourceFileRef{
		OriginalPath:   originalPath,
		LineRange:      lineRange,
		ChecksumSHA256: checksum,
		SizeBytes:      fileInfo.Size(),
		LastModified:   fileInfo.ModTime(),
	}

	// Extract snippet if line range specified
	if lineRange != "" {
		snippet, err := s.extractSnippet(absSourcePath, lineRange)
		if err != nil {
			return nil, fmt.Errorf("failed to extract snippet: %w", err)
		}
		sourceRef.Snippet = snippet
	}

	// Copy file to destination if specified
	if destDir != "" {
		copiedPath, err := s.copySourceFile(absSourcePath, destDir)
		if err != nil {
			return nil, fmt.Errorf("failed to copy source file: %w", err)
		}
		sourceRef.CopiedPath = copiedPath
	}

	return sourceRef, nil
}

// extractSnippet extracts a code snippet from a file with context lines
func (s *SourceFileCollector) extractSnippet(filePath, lineRange string) (string, error) {
	// Parse line range
	startLine, endLine, err := s.parseLineRange(lineRange)
	if err != nil {
		return "", fmt.Errorf("invalid line range: %w", err)
	}

	// Add context lines
	startLine = max(1, startLine-s.contextLines)
	endLine = endLine + s.contextLines

	// Limit snippet size
	if endLine-startLine+1 > s.maxSnippetSize {
		endLine = startLine + s.maxSnippetSize - 1
	}

	// Read file and extract lines
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	var snippetLines []string
	scanner := bufio.NewScanner(file)
	lineNum := 1

	for scanner.Scan() {
		if lineNum >= startLine && lineNum <= endLine {
			snippetLines = append(snippetLines, scanner.Text())
		}
		if lineNum > endLine {
			break
		}
		lineNum++
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("error reading file: %w", err)
	}

	// Format snippet with line numbers
	var builder strings.Builder
	for i, line := range snippetLines {
		actualLineNum := startLine + i
		builder.WriteString(fmt.Sprintf("%4d | %s\n", actualLineNum, line))
	}

	return builder.String(), nil
}

// parseLineRange parses a line range string like "45-67" or "123"
func (s *SourceFileCollector) parseLineRange(rangeStr string) (int, int, error) {
	if rangeStr == "" {
		return 0, 0, fmt.Errorf("empty line range")
	}

	// Check if it's a range (e.g., "45-67")
	if strings.Contains(rangeStr, "-") {
		parts := strings.SplitN(rangeStr, "-", 2)
		start, err := strconv.Atoi(strings.TrimSpace(parts[0]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid start line: %w", err)
		}
		end, err := strconv.Atoi(strings.TrimSpace(parts[1]))
		if err != nil {
			return 0, 0, fmt.Errorf("invalid end line: %w", err)
		}
		if start > end {
			return 0, 0, fmt.Errorf("start line %d is greater than end line %d", start, end)
		}
		return start, end, nil
	}

	// Single line number
	line, err := strconv.Atoi(strings.TrimSpace(rangeStr))
	if err != nil {
		return 0, 0, fmt.Errorf("invalid line number: %w", err)
	}
	return line, line, nil
}

// copySourceFile copies a source file to the destination directory
// Returns the relative path within the evidence folder
func (s *SourceFileCollector) copySourceFile(sourcePath, destDir string) (string, error) {
	// Ensure destination directory exists
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Get filename
	filename := filepath.Base(sourcePath)

	// Check if file already exists, append counter if needed
	destPath := filepath.Join(destDir, filename)
	counter := 1
	for {
		if _, err := os.Stat(destPath); os.IsNotExist(err) {
			break
		}
		// File exists, try with counter
		ext := filepath.Ext(filename)
		nameWithoutExt := strings.TrimSuffix(filename, ext)
		destPath = filepath.Join(destDir, fmt.Sprintf("%s_%d%s", nameWithoutExt, counter, ext))
		counter++
	}

	// Copy file
	if err := s.copyFile(sourcePath, destPath); err != nil {
		return "", fmt.Errorf("failed to copy file: %w", err)
	}

	// Return path relative to data directory
	relPath, err := utils.RelativizePathFromDataDir(destPath, s.dataDir)
	if err != nil {
		return "", fmt.Errorf("failed to get relative path: %w", err)
	}

	return relPath, nil
}

// copyFile copies a file from src to dst
func (s *SourceFileCollector) copyFile(src, dst string) error {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sourceFile.Close()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	if err != nil {
		return err
	}

	// Preserve file permissions
	sourceInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	return os.Chmod(dst, sourceInfo.Mode())
}

// calculateChecksum calculates SHA256 checksum of a file
func (s *SourceFileCollector) calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// getRelativePath returns a path relative to git root or data dir
func (s *SourceFileCollector) getRelativePath(absPath string) (string, error) {
	// Try git root first
	if s.gitRoot != "" {
		relPath, err := filepath.Rel(s.gitRoot, absPath)
		if err == nil && !strings.HasPrefix(relPath, "..") {
			return relPath, nil
		}
	}

	// Fall back to data directory
	return utils.RelativizePathFromDataDir(absPath, s.dataDir)
}

// CollectMultipleSourceFiles collects multiple source files at once
func (s *SourceFileCollector) CollectMultipleSourceFiles(
	sourceFiles []string,
	destDir string,
) ([]models.SourceFileRef, error) {
	var refs []models.SourceFileRef

	for _, sourcePath := range sourceFiles {
		ref, err := s.CollectSourceFile(sourcePath, "", destDir)
		if err != nil {
			return nil, fmt.Errorf("failed to collect %s: %w", sourcePath, err)
		}
		refs = append(refs, *ref)
	}

	return refs, nil
}

// max returns the maximum of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
