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
	"fmt"
	"regexp"
	"strings"
	"time"
)

// CalculateEvidenceWindow determines the window for a given interval and date
// Pure function - no side effects, easily testable
func CalculateEvidenceWindow(interval string, date time.Time) string {
	switch strings.ToLower(interval) {
	case "year", "annual", "annually":
		return fmt.Sprintf("%d", date.Year())
	case "quarter", "quarterly":
		quarter := (date.Month()-1)/3 + 1
		return fmt.Sprintf("%d-Q%d", date.Year(), quarter)
	case "month", "monthly":
		return fmt.Sprintf("%d-%02d", date.Year(), date.Month())
	case "six_month", "semi-annual", "semiannual":
		half := 1
		if date.Month() > 6 {
			half = 2
		}
		return fmt.Sprintf("%d-H%d", date.Year(), half)
	default:
		return fmt.Sprintf("%d", date.Year())
	}
}

// GenerateEvidenceFilename creates a numbered filename
// Pure function for deterministic file naming
func GenerateEvidenceFilename(index int, title string) string {
	// Sanitize title for filesystem - keep alphanumeric, spaces, hyphens, underscores
	safe := regexp.MustCompile(`[^a-zA-Z0-9\s\-_]`).ReplaceAllString(title, "")

	// Replace spaces and hyphens with underscores, convert to lowercase
	safe = strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(safe, " ", "_"), "-", "_"))

	// Remove multiple consecutive underscores
	safe = regexp.MustCompile(`_{2,}`).ReplaceAllString(safe, "_")

	// Trim underscores from start and end
	safe = strings.Trim(safe, "_")

	// Ensure we have something
	if safe == "" {
		safe = "evidence"
	}

	return fmt.Sprintf("%02d_%s", index, safe)
}

// SanitizeTaskName creates a filesystem-safe directory name from task name
// Pure function for directory naming
func SanitizeTaskName(name string) string {
	// Keep alphanumeric, spaces, hyphens, underscores, parentheses
	safe := regexp.MustCompile(`[^a-zA-Z0-9\s\-_()\[\]]`).ReplaceAllString(name, "")

	// Replace spaces with underscores
	safe = strings.ReplaceAll(safe, " ", "_")

	// Remove multiple consecutive underscores
	safe = regexp.MustCompile(`_{2,}`).ReplaceAllString(safe, "_")

	// Trim underscores from start and end
	safe = strings.Trim(safe, "_")

	// Ensure we have something and limit length
	if safe == "" {
		safe = "evidence_task"
	}

	// Limit length to 100 characters for filesystem compatibility
	if len(safe) > 100 {
		safe = safe[:100]
		safe = strings.TrimSuffix(safe, "_")
	}

	return safe
}
