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

package models

import "time"

// GenerationMetadata tracks how evidence files were generated
// This enables the state tracking system to understand evidence provenance
type GenerationMetadata struct {
	GeneratedAt      time.Time      `yaml:"generated_at"`
	GeneratedBy      string         `yaml:"generated_by"`      // "claude-code-assisted", "grctool-cli", "manual"
	GenerationMethod string         `yaml:"generation_method"` // "tool_coordination", "manual_upload"
	TaskID           int            `yaml:"task_id"`
	TaskRef          string         `yaml:"task_ref"`
	Window           string         `yaml:"window"`
	ToolsUsed        []string       `yaml:"tools_used,omitempty"`
	FilesGenerated   []FileMetadata `yaml:"files_generated"`
	Status           string         `yaml:"status"` // "generated", "validated", "submitted"
}

// FileMetadata represents metadata about a single evidence file
type FileMetadata struct {
	Path        string    `yaml:"path"`     // Relative path from window directory
	Checksum    string    `yaml:"checksum"` // "sha256:abc123..."
	SizeBytes   int64     `yaml:"size_bytes"`
	GeneratedAt time.Time `yaml:"generated_at"`
}
