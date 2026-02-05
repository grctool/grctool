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

package utils

import (
	"fmt"
	"path/filepath"
	"strings"
)

// RelativizePathFromDataDir converts an absolute path to a relative path from the data directory
// This ensures no /home/, /Users/, or other absolute paths leak into evidence
func RelativizePathFromDataDir(absPath, dataDir string) (string, error) {
	if absPath == "" {
		return "", fmt.Errorf("absolute path is empty")
	}

	if dataDir == "" {
		return "", fmt.Errorf("data directory is empty")
	}

	// Make both paths absolute and clean them
	cleanAbsPath, err := filepath.Abs(absPath)
	if err != nil {
		return "", fmt.Errorf("failed to resolve absolute path: %w", err)
	}

	cleanDataDir, err := filepath.Abs(dataDir)
	if err != nil {
		return "", fmt.Errorf("failed to resolve data directory: %w", err)
	}

	// Calculate relative path
	relPath, err := filepath.Rel(cleanDataDir, cleanAbsPath)
	if err != nil {
		return "", fmt.Errorf("failed to compute relative path: %w", err)
	}

	// Check if the path is actually relative (doesn't start with ..)
	// If it does, the file is outside the data directory
	if strings.HasPrefix(relPath, "..") {
		return "", fmt.Errorf("path %s is outside data directory %s", absPath, dataDir)
	}

	return relPath, nil
}

// ValidateRelativePath checks if a path is relative and doesn't contain absolute path markers
func ValidateRelativePath(path string) error {
	if path == "" {
		return fmt.Errorf("path is empty")
	}

	// Check for absolute path markers (Unix and Windows)
	if filepath.IsAbs(path) {
		return fmt.Errorf("path is absolute: %s", path)
	}

	// Check for common absolute path prefixes
	absolutePrefixes := []string{
		"/home/",
		"/Users/",
		"/root/",
		"/pool",
		"C:\\",
		"D:\\",
	}

	for _, prefix := range absolutePrefixes {
		if strings.HasPrefix(path, prefix) {
			return fmt.Errorf("path contains absolute prefix %s: %s", prefix, path)
		}
	}

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return fmt.Errorf("path contains parent directory references: %s", path)
	}

	return nil
}

// SanitizePathForEvidence ensures a path is safe for inclusion in evidence
// This converts to relative paths and validates them
func SanitizePathForEvidence(path, dataDir string) (string, error) {
	// If already relative, validate and return
	if !filepath.IsAbs(path) {
		if err := ValidateRelativePath(path); err != nil {
			return "", fmt.Errorf("invalid relative path: %w", err)
		}
		return path, nil
	}

	// Convert absolute to relative
	relPath, err := RelativizePathFromDataDir(path, dataDir)
	if err != nil {
		return "", fmt.Errorf("failed to relativize path: %w", err)
	}

	return relPath, nil
}

// IsPathWithinDirectory checks if a path is within a given directory
func IsPathWithinDirectory(path, directory string) (bool, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return false, err
	}

	absDir, err := filepath.Abs(directory)
	if err != nil {
		return false, err
	}

	relPath, err := filepath.Rel(absDir, absPath)
	if err != nil {
		return false, err
	}

	// If relative path starts with .., it's outside the directory
	return !strings.HasPrefix(relPath, ".."), nil
}

// NormalizePathSeparators converts path separators to forward slashes
// This ensures consistent path representation across platforms
func NormalizePathSeparators(path string) string {
	return filepath.ToSlash(path)
}

// JoinRelativePaths joins multiple relative path components
// Returns an error if any component is absolute
func JoinRelativePaths(components ...string) (string, error) {
	for i, comp := range components {
		if filepath.IsAbs(comp) {
			return "", fmt.Errorf("component %d is absolute: %s", i, comp)
		}
	}

	joined := filepath.Join(components...)
	return NormalizePathSeparators(joined), nil
}
