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

package services

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// idFields are the JSON field names that should be string-typed IDs.
var idFields = map[string]bool{
	"id":         true,
	"task_id":    true,
	"control_id": true,
	"policy_id":  true,
}

// MigrateResult holds the outcome of a JSON ID migration run.
type MigrateResult struct {
	FilesScanned  int
	FilesModified int
	FilesFailed   int
	Changes       []MigrateChange
	Errors        []string
}

// MigrateChange describes a single field that was (or would be) converted.
type MigrateChange struct {
	FilePath string
	Field    string
	OldValue interface{}
	NewValue string
}

// MigrateJSONFiles walks dataDir looking for .json files, and converts any
// numeric "id", "task_id", "control_id", or "policy_id" values to strings.
// When dryRun is true the files are not modified.
func MigrateJSONFiles(dataDir string, dryRun bool) (*MigrateResult, error) {
	result := &MigrateResult{}

	// Collect all JSON files under dataDir.
	var jsonFiles []string
	err := filepath.Walk(dataDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip inaccessible entries
		}
		if !info.IsDir() && strings.HasSuffix(strings.ToLower(info.Name()), ".json") {
			jsonFiles = append(jsonFiles, path)
		}
		return nil
	})
	if err != nil {
		return result, fmt.Errorf("walking data directory: %w", err)
	}

	for _, path := range jsonFiles {
		result.FilesScanned++

		changes, err := migrateFile(path, dryRun)
		if err != nil {
			result.FilesFailed++
			result.Errors = append(result.Errors, fmt.Sprintf("%s: %v", path, err))
			continue
		}
		if len(changes) > 0 {
			result.FilesModified++
			result.Changes = append(result.Changes, changes...)
		}
	}

	return result, nil
}

// migrateFile processes a single JSON file.  It returns the changes found (and
// applied, unless dryRun was true for the caller – the write decision is made
// here based on whether changes exist).
func migrateFile(path string, dryRun bool) ([]MigrateChange, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading file: %w", err)
	}

	var root interface{}
	if err := json.Unmarshal(data, &root); err != nil {
		return nil, fmt.Errorf("parsing JSON: %w", err)
	}

	var changes []MigrateChange
	converted := convertValue(root, path, &changes)

	if len(changes) == 0 {
		return nil, nil
	}

	if !dryRun {
		out, err := json.MarshalIndent(converted, "", "  ")
		if err != nil {
			return changes, fmt.Errorf("marshaling JSON: %w", err)
		}
		// Append a trailing newline to match typical file conventions.
		out = append(out, '\n')
		if err := os.WriteFile(path, out, 0644); err != nil {
			return changes, fmt.Errorf("writing file: %w", err)
		}
	}

	return changes, nil
}

// convertValue recursively walks a parsed JSON value, converting numeric ID
// fields to strings.  Discovered changes are appended to *changes.
func convertValue(v interface{}, filePath string, changes *[]MigrateChange) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		out := make(map[string]interface{}, len(val))
		for k, child := range val {
			if idFields[k] {
				if num, ok := child.(float64); ok {
					strVal := formatFloat(num)
					*changes = append(*changes, MigrateChange{
						FilePath: filePath,
						Field:    k,
						OldValue: num,
						NewValue: strVal,
					})
					out[k] = strVal
					continue
				}
			}
			out[k] = convertValue(child, filePath, changes)
		}
		return out
	case []interface{}:
		out := make([]interface{}, len(val))
		for i, child := range val {
			out[i] = convertValue(child, filePath, changes)
		}
		return out
	default:
		return v
	}
}

// formatFloat converts a float64 (from JSON number) to a clean string.
// Integer-valued floats are rendered without a decimal point.
func formatFloat(f float64) string {
	if f == float64(int64(f)) {
		return fmt.Sprintf("%d", int64(f))
	}
	return fmt.Sprintf("%g", f)
}
