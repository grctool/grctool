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

package storage

import (
	"fmt"
	"os"
	"path/filepath"
)

// EvidenceFolderStructure represents the directory structure for a task/window evidence collection
type EvidenceFolderStructure struct {
	RootDir       string // ET-0001_Task_Name/2025-Q4/
	EvidenceDir   string // evidence/ subdirectory for main evidence documents
	SourcesDir    string // sources/ subdirectory for full supporting files
	MetadataDir   string // metadata/ subdirectory for provenance, submission metadata
	UseStructured bool   // Whether to use structured folders or flat structure
}

// NewEvidenceFolderStructure creates a new evidence folder structure
func NewEvidenceFolderStructure(rootDir string, useStructured bool) *EvidenceFolderStructure {
	efs := &EvidenceFolderStructure{
		RootDir:       rootDir,
		UseStructured: useStructured,
	}

	if useStructured {
		// Structured layout
		efs.EvidenceDir = filepath.Join(rootDir, "evidence")
		efs.SourcesDir = filepath.Join(rootDir, "sources")
		efs.MetadataDir = filepath.Join(rootDir, "metadata")
	} else {
		// Flat layout (backward compatible)
		efs.EvidenceDir = rootDir
		efs.SourcesDir = rootDir
		efs.MetadataDir = filepath.Join(rootDir, ".submission")
	}

	return efs
}

// Create creates all necessary directories for the evidence structure
func (efs *EvidenceFolderStructure) Create() error {
	dirs := []string{
		efs.RootDir,
		efs.EvidenceDir,
		efs.MetadataDir,
	}

	// Only create sources directory if using structured folders
	if efs.UseStructured {
		dirs = append(dirs, efs.SourcesDir)
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}

// GetEvidenceFilePath returns the full path for an evidence file
func (efs *EvidenceFolderStructure) GetEvidenceFilePath(filename string) string {
	return filepath.Join(efs.EvidenceDir, filename)
}

// GetSourceFilePath returns the full path for a source file
func (efs *EvidenceFolderStructure) GetSourceFilePath(filename string) string {
	return filepath.Join(efs.SourcesDir, filename)
}

// GetMetadataFilePath returns the full path for a metadata file
func (efs *EvidenceFolderStructure) GetMetadataFilePath(filename string) string {
	return filepath.Join(efs.MetadataDir, filename)
}

// GetCollectionPlanPath returns the path to the collection plan file
func (efs *EvidenceFolderStructure) GetCollectionPlanPath() string {
	if efs.UseStructured {
		return efs.GetMetadataFilePath("collection_plan.yaml")
	}
	// Backward compatible - keep as markdown in root
	return filepath.Join(efs.RootDir, "collection_plan.md")
}

// GetProvenancePath returns the path to the provenance metadata file
func (efs *EvidenceFolderStructure) GetProvenancePath() string {
	return efs.GetMetadataFilePath("provenance.yaml")
}

// GetSubmissionPath returns the path to the submission metadata file
func (efs *EvidenceFolderStructure) GetSubmissionPath() string {
	return efs.GetMetadataFilePath("submission.yaml")
}

// GetValidationPath returns the path to the validation metadata file
func (efs *EvidenceFolderStructure) GetValidationPath() string {
	return efs.GetMetadataFilePath("validation.yaml")
}

// GetHistoryPath returns the path to the submission history file
func (efs *EvidenceFolderStructure) GetHistoryPath() string {
	return efs.GetMetadataFilePath("history.yaml")
}

// CreateReadme creates a README.md file in the evidence folder
func (efs *EvidenceFolderStructure) CreateReadme(taskRef, window string) error {
	if !efs.UseStructured {
		// Only create README for structured folders
		return nil
	}

	readmePath := filepath.Join(efs.RootDir, "README.md")

	content := fmt.Sprintf(`# Evidence Collection: %s - %s

## Directory Structure

- **evidence/** - Main evidence documents (CSV, Markdown, JSON)
- **sources/** - Full supporting source files (Terraform files, scripts, etc.)
- **metadata/** - Collection metadata
  - provenance.yaml - Git commit, branch, and collection context
  - collection_plan.yaml - Inventory of all evidence files
  - submission.yaml - Submission tracking and status
  - validation.yaml - Validation results
  - history.yaml - Submission history

## About This Evidence

This evidence was collected using GRCTool's automated compliance evidence collection system.

### Provenance Information

See metadata/provenance.yaml for:
- Git commit SHA and branch
- Collection timestamp
- Tool version used
- Source repository information

### Source Files

All referenced source files are included in the sources/ directory with:
- Original file checksums for verification
- Relative paths from git repository root
- Last modification timestamps

### Evidence Documents

Main evidence documents in the evidence/ directory include:
- Extracted code snippets with context
- Structured data exports (CSV/JSON)
- Supporting documentation (Markdown)

## Validation

Evidence validation results are available in metadata/validation.yaml

## Submission

Submission tracking and history are maintained in:
- metadata/submission.yaml - Current submission status
- metadata/history.yaml - Historical submissions

---
*Auto-generated by GRCTool*
`, taskRef, window)

	return os.WriteFile(readmePath, []byte(content), 0644)
}

// Exists checks if the evidence folder structure exists
func (efs *EvidenceFolderStructure) Exists() bool {
	_, err := os.Stat(efs.RootDir)
	return err == nil
}
