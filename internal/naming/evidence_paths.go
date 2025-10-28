// Package naming provides unified naming conventions for GRCTool directory structures.
package naming

import (
	"fmt"
	"regexp"
	"strings"
)

const (
	// TaskDirNameFormat is the format string for evidence task directory names.
	// Format: {ReferenceID}_{SanitizedTaskName}
	TaskDirNameFormat = "%s_%s"

	// TaskDirNamePattern is the regex pattern for parsing evidence task directory names.
	TaskDirNamePattern = `^(ET-\d{4})_(.+)$`

	// MaxTaskNameLength is the maximum length for sanitized task names.
	MaxTaskNameLength = 100
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
// The format is: {ReferenceID}_{SanitizedTaskName}
// Example: "ET-0001_GitHub_Access_Controls"
func GetEvidenceTaskDirName(taskRef, taskName string) string {
	sanitized := SanitizeTaskName(taskName)
	return fmt.Sprintf(TaskDirNameFormat, taskRef, sanitized)
}

// ParseEvidenceTaskDirName parses a directory name back into its components.
// Returns the task reference ID and the sanitized task name.
// If the directory name doesn't match the expected pattern, returns empty strings.
func ParseEvidenceTaskDirName(dirName string) (ref string, name string) {
	matches := taskDirRegex.FindStringSubmatch(dirName)
	if len(matches) < 3 {
		return "", ""
	}

	ref = matches[1]
	// Convert underscores back to spaces for the name
	name = strings.ReplaceAll(matches[2], "_", " ")

	return ref, name
}

// SanitizeTaskName converts a task name into a filesystem-safe format.
// Rules:
// - Keeps only alphanumeric characters, spaces, hyphens, underscores, parentheses, and brackets
// - Replaces spaces with underscores
// - Removes consecutive underscores
// - Trims leading/trailing underscores
// - Limits length to MaxTaskNameLength characters
func SanitizeTaskName(name string) string {
	// Remove unsafe characters
	safe := unsafeCharsRegex.ReplaceAllString(name, "")

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

// MatchesTaskRef checks if a directory name matches or starts with the given task reference.
// This is useful for finding directories when you only have the reference ID.
// Returns true if:
// - The directory name exactly matches the task reference (e.g., "ET-0001")
// - The directory name starts with the task reference followed by underscore (e.g., "ET-0001_TaskName")
func MatchesTaskRef(dirName, taskRef string) bool {
	if dirName == taskRef {
		return true
	}
	prefix := taskRef + "_"
	return strings.HasPrefix(dirName, prefix)
}

// ExtractTaskRef extracts the task reference ID from a directory name.
// Returns empty string if no valid reference is found.
func ExtractTaskRef(dirName string) string {
	matches := taskDirRegex.FindStringSubmatch(dirName)
	if len(matches) < 2 {
		// Try to match just the reference ID if directory uses old format
		refPattern := regexp.MustCompile(`^(ET-\d{4})$`)
		refMatches := refPattern.FindStringSubmatch(dirName)
		if len(refMatches) >= 2 {
			return refMatches[1]
		}
		return ""
	}
	return matches[1]
}
