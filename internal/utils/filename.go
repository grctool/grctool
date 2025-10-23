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
	"regexp"
	"strings"
)

// FilenameGenerator provides unified filename generation across the system
type FilenameGenerator struct {
	maxNameLength int
}

// NewFilenameGenerator creates a new filename generator
func NewFilenameGenerator() *FilenameGenerator {
	return &FilenameGenerator{
		maxNameLength: 100, // Maximum length for the name portion
	}
}

// GenerateFilename creates a unified filename pattern:
// {ReferenceID}_{NumericID}_{SanitizedName}.{extension}
// Examples:
// - P136_94623_Data_Retention_and_Disposal.json
// - AC1_778771_Access_Provisioning_and_Approval.json
// - ET1_327992_Access_Control_Registration_and_Deregistration_Process_Document.json
func (fg *FilenameGenerator) GenerateFilename(referenceID string, numericID string, name string, extension string) string {
	sanitizedName := fg.SanitizeName(name)

	// Ensure extension starts with a dot
	if !strings.HasPrefix(extension, ".") {
		extension = "." + extension
	}

	return fmt.Sprintf("%s_%s_%s%s", referenceID, numericID, sanitizedName, extension)
}

// SanitizeName sanitizes a name for use in filenames
func (fg *FilenameGenerator) SanitizeName(name string) string {
	// Replace various characters with underscores
	sanitized := name

	// Replace forward slashes with underscores
	sanitized = strings.ReplaceAll(sanitized, "/", "_")

	// Replace hyphens with underscores to maintain consistency
	// (but keep hyphens that are part of compound words)
	sanitized = strings.ReplaceAll(sanitized, " - ", "_")

	// Replace spaces with underscores
	sanitized = strings.ReplaceAll(sanitized, " ", "_")

	// Remove parentheses
	sanitized = strings.ReplaceAll(sanitized, "(", "")
	sanitized = strings.ReplaceAll(sanitized, ")", "")

	// Remove commas
	sanitized = strings.ReplaceAll(sanitized, ",", "")

	// Remove quotes
	sanitized = strings.ReplaceAll(sanitized, "'", "")
	sanitized = strings.ReplaceAll(sanitized, "\"", "")

	// Replace ampersands
	sanitized = strings.ReplaceAll(sanitized, "&", "and")

	// Replace percent signs
	sanitized = strings.ReplaceAll(sanitized, "%", "percent")

	// Remove hash symbols
	sanitized = strings.ReplaceAll(sanitized, "#", "")

	// Replace at symbols
	sanitized = strings.ReplaceAll(sanitized, "@", "at")

	// Remove other problematic characters
	sanitized = strings.ReplaceAll(sanitized, ":", "")
	sanitized = strings.ReplaceAll(sanitized, "?", "")
	sanitized = strings.ReplaceAll(sanitized, "*", "")
	sanitized = strings.ReplaceAll(sanitized, "<", "")
	sanitized = strings.ReplaceAll(sanitized, ">", "")
	sanitized = strings.ReplaceAll(sanitized, "|", "")
	sanitized = strings.ReplaceAll(sanitized, "\\", "")

	// Replace multiple consecutive underscores with single underscore
	multipleUnderscores := regexp.MustCompile(`_{2,}`)
	sanitized = multipleUnderscores.ReplaceAllString(sanitized, "_")

	// Trim leading and trailing underscores
	sanitized = strings.Trim(sanitized, "_")

	// Truncate if necessary
	if len(sanitized) > fg.maxNameLength {
		sanitized = sanitized[:fg.maxNameLength]
		// Trim any trailing underscore after truncation
		sanitized = strings.TrimRight(sanitized, "_")
	}

	return sanitized
}

// ParseFilename parses a unified filename pattern back into its components
// Returns referenceID, numericID, name, extension
func (fg *FilenameGenerator) ParseFilename(filename string) (string, string, string, string, error) {
	// Extract extension
	lastDot := strings.LastIndex(filename, ".")
	if lastDot == -1 {
		return "", "", "", "", fmt.Errorf("no extension found in filename: %s", filename)
	}

	extension := filename[lastDot:]
	nameWithoutExt := filename[:lastDot]

	// Split by underscore to get components
	parts := strings.Split(nameWithoutExt, "_")
	if len(parts) < 3 {
		return "", "", "", "", fmt.Errorf("invalid filename format: %s", filename)
	}

	referenceID := parts[0]
	numericID := parts[1]
	// Join the rest as the name (in case name had underscores)
	name := strings.Join(parts[2:], "_")

	return referenceID, numericID, name, extension, nil
}

// IsValidFilename checks if a filename follows the unified pattern
func (fg *FilenameGenerator) IsValidFilename(filename string) bool {
	// Pattern: {ReferenceID}_{NumericID}_{SanitizedName}.{extension}
	// ReferenceID: alphanumeric (e.g., P136, AC1, ET1)
	// NumericID: digits only
	// SanitizedName: alphanumeric with underscores
	// Extension: .json or .md
	pattern := `^[A-Z]+\d*_\d+_[A-Za-z0-9_]+\.(json|md)$`
	matched, _ := regexp.MatchString(pattern, filename)
	return matched
}
