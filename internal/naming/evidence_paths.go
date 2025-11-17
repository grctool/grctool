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

// Package naming provides unified naming conventions for GRCTool directory structures.
package naming

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// TaskDirNameFormat is the format string for evidence task directory names.
	// Format: {SanitizedTaskName}_{ReferenceID}_{TugboatID}
	TaskDirNameFormat = "%s_%s_%s"

	// TaskDirNamePattern is the regex pattern for parsing evidence task directory names.
	// Matches: TaskName_ET-XXXX_TugboatID
	TaskDirNamePattern = `^(.+)_(ET-\d{4})_(\d+)$`

	// MaxTaskNameLength is the maximum length for sanitized task names.
	MaxTaskNameLength = 100

	// SubfolderSubmitted is the hidden folder for submitted files (moved after upload to prevent resubmission)
	SubfolderSubmitted = ".submitted"

	// SubfolderArchive is the folder for evidence synced FROM Tugboat
	SubfolderArchive = "archive"
)

var (
	// taskDirRegex is compiled once for parsing directory names.
	taskDirRegex = regexp.MustCompile(TaskDirNamePattern)

	// unsafeCharsRegex matches characters that are not safe for filesystem names.
	unsafeCharsRegex = regexp.MustCompile(`[^a-zA-Z0-9\s\-_()\[\]]`)

	// multipleUnderscoresRegex matches consecutive underscores.
	multipleUnderscoresRegex = regexp.MustCompile(`_{2,}`)
)

// GetEvidenceTaskDirName generates a standardized directory name for an evidence task.
// The format is: {SanitizedTaskName}_{ReferenceID}_{TugboatID}
// Example: "GitHub Access Controls_ET-0001_328031"
func GetEvidenceTaskDirName(taskName, taskRef, tugboatID string) string {
	sanitized := SanitizeTaskName(taskName)
	return fmt.Sprintf(TaskDirNameFormat, sanitized, taskRef, tugboatID)
}

// ParseEvidenceTaskDirName parses a directory name back into its components.
// Returns the task name, task reference ID, and Tugboat ID.
// If the directory name doesn't match the expected pattern, returns empty strings.
func ParseEvidenceTaskDirName(dirName string) (name string, ref string, tugboatID string) {
	matches := taskDirRegex.FindStringSubmatch(dirName)
	if len(matches) < 4 {
		return "", "", ""
	}

	// Convert underscores back to spaces for the name
	name = strings.ReplaceAll(matches[1], "_", " ")
	ref = matches[2]
	tugboatID = matches[3]

	return name, ref, tugboatID
}

// SanitizeTaskName converts a task name into a filesystem-safe format.
// Rules:
// - Keeps only alphanumeric characters, spaces, hyphens, underscores, parentheses, and brackets
// - Replaces unsafe characters (and spaces) with underscores
// - Removes consecutive underscores
// - Trims leading/trailing underscores
// - Limits length to MaxTaskNameLength characters
func SanitizeTaskName(name string) string {
	// Replace unsafe characters with underscores
	safe := unsafeCharsRegex.ReplaceAllString(name, "_")

	// Replace spaces with underscores
	safe = strings.ReplaceAll(safe, " ", "_")

	// Remove multiple consecutive underscores
	safe = multipleUnderscoresRegex.ReplaceAllString(safe, "_")

	// Trim underscores from start and end
	safe = strings.Trim(safe, "_")

	// Limit length
	if len(safe) > MaxTaskNameLength {
		safe = safe[:MaxTaskNameLength]
		// Ensure we don't end with an underscore after truncation
		safe = strings.TrimSuffix(safe, "_")
	}

	return safe
}

// MatchesTaskRef checks if a directory name contains the given task reference.
// This is useful for finding directories when you only have the reference ID.
// Returns true if:
// - The directory name exactly matches the task reference (e.g., "ET-0001")
// - The directory name contains the task reference in the expected position (e.g., "TaskName_ET-0001_328031")
func MatchesTaskRef(dirName, taskRef string) bool {
	if dirName == taskRef {
		return true
	}
	// Try parsing to see if the reference matches
	_, ref, _ := ParseEvidenceTaskDirName(dirName)
	return ref == taskRef
}

// ExtractTaskRef extracts the task reference ID from a directory name.
// Returns empty string if no valid reference is found.
func ExtractTaskRef(dirName string) string {
	matches := taskDirRegex.FindStringSubmatch(dirName)
	if len(matches) < 3 {
		// Try to match just the reference ID if directory is exactly the ref
		refPattern := regexp.MustCompile(`^(ET-\d{4})$`)
		refMatches := refPattern.FindStringSubmatch(dirName)
		if len(refMatches) >= 2 {
			return refMatches[1]
		}
		return ""
	}
	// Reference is now in position 2 (TaskName_ET-XXXX_TugboatID)
	return matches[2]
}
