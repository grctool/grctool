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

package terraform

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
)

// IndexValidator handles cache invalidation logic for Terraform indices
type IndexValidator struct {
	config *config.TerraformToolConfig
	logger logger.Logger
}

// NewIndexValidator creates a new index validator
func NewIndexValidator(cfg *config.TerraformToolConfig, log logger.Logger) *IndexValidator {
	return &IndexValidator{
		config: cfg,
		logger: log,
	}
}

// InvalidationReason describes why an index needs rebuilding
type InvalidationReason string

const (
	ReasonNone             InvalidationReason = "none"
	ReasonIndexMissing     InvalidationReason = "index_missing"
	ReasonVersionMismatch  InvalidationReason = "version_mismatch"
	ReasonFileModified     InvalidationReason = "file_modified"
	ReasonFileAdded        InvalidationReason = "file_added"
	ReasonFileDeleted      InvalidationReason = "file_deleted"
	ReasonChecksumMismatch InvalidationReason = "checksum_mismatch"
	ReasonConfigChanged    InvalidationReason = "config_changed"
	ReasonIndexCorrupted   InvalidationReason = "index_corrupted"
	ReasonIndexTooOld      InvalidationReason = "index_too_old"
)

// ValidationResult contains the result of index validation
type ValidationResult struct {
	NeedsRebuild  bool               `json:"needs_rebuild"`
	Reason        InvalidationReason `json:"reason"`
	Details       string             `json:"details"`
	ChangedFiles  []string           `json:"changed_files,omitempty"`
	AddedFiles    []string           `json:"added_files,omitempty"`
	DeletedFiles  []string           `json:"deleted_files,omitempty"`
	LastIndexedAt time.Time          `json:"last_indexed_at,omitempty"`
	CurrentFiles  int                `json:"current_files"`
	IndexedFiles  int                `json:"indexed_files"`
}

// NeedsRebuild checks if the index needs to be rebuilt
func (iv *IndexValidator) NeedsRebuild(index *PersistedIndex) ValidationResult {
	// Check 1: Version compatibility
	if index.Version != IndexVersion {
		return ValidationResult{
			NeedsRebuild: true,
			Reason:       ReasonVersionMismatch,
			Details:      fmt.Sprintf("Index version %s incompatible with current version %s", index.Version, IndexVersion),
		}
	}

	// Check 2: Configuration fingerprint (scan paths, patterns changed)
	currentFingerprint := iv.calculateConfigFingerprint()
	if index.ConfigFingerprint != currentFingerprint {
		return ValidationResult{
			NeedsRebuild: true,
			Reason:       ReasonConfigChanged,
			Details:      "Terraform scan configuration has changed (scan paths or include/exclude patterns)",
		}
	}

	// Check 3: Age check (if index is very old, suggest rebuild)
	maxAge := 7 * 24 * time.Hour // 7 days
	age := time.Since(index.IndexedAt)
	if age > maxAge {
		iv.logger.Warn("Index is old",
			logger.Duration("age", age),
			logger.Duration("max_age", maxAge))
		// Note: This is a soft warning, not a hard requirement
		// We'll continue checking files
	}

	// Check 4: File changes (mtime, additions, deletions)
	currentFiles, err := iv.getCurrentTerraformFiles()
	if err != nil {
		return ValidationResult{
			NeedsRebuild: true,
			Reason:       ReasonIndexCorrupted,
			Details:      fmt.Sprintf("Failed to scan current Terraform files: %v", err),
		}
	}

	// Compare files
	result := iv.compareFiles(index.SourceFiles, currentFiles)
	result.LastIndexedAt = index.IndexedAt
	result.IndexedFiles = len(index.SourceFiles)
	result.CurrentFiles = len(currentFiles)

	// If age is excessive, suggest rebuild even if no file changes
	if !result.NeedsRebuild && age > maxAge {
		return ValidationResult{
			NeedsRebuild:  true,
			Reason:        ReasonIndexTooOld,
			Details:       fmt.Sprintf("Index is %s old (older than %s)", age, maxAge),
			LastIndexedAt: index.IndexedAt,
			CurrentFiles:  len(currentFiles),
			IndexedFiles:  len(index.SourceFiles),
		}
	}

	return result
}

// HasFileChanges performs a quick check for file changes without full checksum validation
func (iv *IndexValidator) HasFileChanges(index *PersistedIndex) (bool, []string, error) {
	currentFiles, err := iv.getCurrentTerraformFiles()
	if err != nil {
		return false, nil, err
	}

	var changedFiles []string

	// Check for file count mismatch (quick check)
	if len(currentFiles) != len(index.SourceFiles) {
		// Some files added or deleted
		return true, []string{"file count mismatch"}, nil
	}

	// Check mtimes (quick check, no checksums)
	for path, currentInfo := range currentFiles {
		indexedInfo, exists := index.SourceFiles[path]
		if !exists {
			changedFiles = append(changedFiles, path+" (new)")
			continue
		}

		// Compare modification times
		if !currentInfo.ModTime.Equal(indexedInfo.ModTime) {
			changedFiles = append(changedFiles, path+" (modified)")
		}

		// Compare file sizes (quick indicator of change)
		if currentInfo.SizeBytes != indexedInfo.SizeBytes {
			changedFiles = append(changedFiles, path+" (size changed)")
		}
	}

	// Check for deleted files
	for path := range index.SourceFiles {
		if _, exists := currentFiles[path]; !exists {
			changedFiles = append(changedFiles, path+" (deleted)")
		}
	}

	return len(changedFiles) > 0, changedFiles, nil
}

// compareFiles performs detailed comparison of indexed vs current files
func (iv *IndexValidator) compareFiles(indexed, current map[string]*SourceFileInfo) ValidationResult {
	var changedFiles []string
	var addedFiles []string
	var deletedFiles []string

	// Check for modified or added files
	for path, currentInfo := range current {
		indexedInfo, exists := indexed[path]

		if !exists {
			// New file
			addedFiles = append(addedFiles, path)
			continue
		}

		// Check if file was modified
		if iv.isFileModified(indexedInfo, currentInfo) {
			changedFiles = append(changedFiles, path)
		}
	}

	// Check for deleted files
	for path := range indexed {
		if _, exists := current[path]; !exists {
			deletedFiles = append(deletedFiles, path)
		}
	}

	// Determine if rebuild is needed
	needsRebuild := len(changedFiles) > 0 || len(addedFiles) > 0 || len(deletedFiles) > 0

	if !needsRebuild {
		return ValidationResult{
			NeedsRebuild: false,
			Reason:       ReasonNone,
			Details:      "Index is up to date",
		}
	}

	// Determine primary reason
	reason := ReasonFileModified
	details := fmt.Sprintf("Changed: %d, Added: %d, Deleted: %d",
		len(changedFiles), len(addedFiles), len(deletedFiles))

	if len(addedFiles) > len(changedFiles) && len(addedFiles) > len(deletedFiles) {
		reason = ReasonFileAdded
	} else if len(deletedFiles) > len(changedFiles) && len(deletedFiles) > len(addedFiles) {
		reason = ReasonFileDeleted
	}

	return ValidationResult{
		NeedsRebuild: true,
		Reason:       reason,
		Details:      details,
		ChangedFiles: changedFiles,
		AddedFiles:   addedFiles,
		DeletedFiles: deletedFiles,
	}
}

// isFileModified checks if a file has been modified based on mtime and size
func (iv *IndexValidator) isFileModified(indexed, current *SourceFileInfo) bool {
	// Check modification time
	if !current.ModTime.Equal(indexed.ModTime) {
		return true
	}

	// Check file size (quick indicator)
	if current.SizeBytes != indexed.SizeBytes {
		return true
	}

	// For paranoid mode, could also check checksum here
	// But mtime + size is usually sufficient
	return false
}

// isFileModifiedWithChecksum performs deep comparison including checksum
func (iv *IndexValidator) isFileModifiedWithChecksum(indexed, current *SourceFileInfo) bool {
	// First do quick checks
	if iv.isFileModified(indexed, current) {
		return true
	}

	// If mtime and size match, but we want to be sure, check checksum
	// Note: This requires reading the file, so it's slower
	if current.Checksum != indexed.Checksum {
		iv.logger.Warn("File has same mtime/size but different checksum",
			logger.String("file", current.Path))
		return true
	}

	return false
}

// getCurrentTerraformFiles scans for current Terraform files
func (iv *IndexValidator) getCurrentTerraformFiles() (map[string]*SourceFileInfo, error) {
	files := make(map[string]*SourceFileInfo)

	for _, scanPath := range iv.config.ScanPaths {
		err := filepath.Walk(scanPath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				// Skip files we can't access
				return nil
			}

			if info.IsDir() {
				return nil
			}

			// Check if file matches patterns
			if !iv.matchesIncludePatterns(path) {
				return nil
			}

			if iv.matchesExcludePatterns(path) {
				return nil
			}

			// Build file info
			absPath, err := filepath.Abs(path)
			if err != nil {
				absPath = path
			}

			files[absPath] = &SourceFileInfo{
				Path:      absPath,
				ModTime:   info.ModTime(),
				SizeBytes: info.Size(),
				// Note: We don't calculate checksum here for performance
				// Checksum is calculated during index build
			}

			return nil
		})

		if err != nil {
			return nil, fmt.Errorf("failed to walk path %s: %w", scanPath, err)
		}
	}

	return files, nil
}

// matchesIncludePatterns checks if a file matches include patterns
func (iv *IndexValidator) matchesIncludePatterns(filePath string) bool {
	if len(iv.config.IncludePatterns) == 0 {
		// No patterns means include all
		return true
	}

	for _, pattern := range iv.config.IncludePatterns {
		matched, err := filepath.Match(pattern, filepath.Base(filePath))
		if err != nil {
			continue
		}
		if matched {
			return true
		}

		// Also check full path for patterns with directories
		if strings.Contains(pattern, "/") {
			matched, err = filepath.Match(pattern, filePath)
			if err == nil && matched {
				return true
			}
		}
	}

	return false
}

// matchesExcludePatterns checks if a file matches exclude patterns
func (iv *IndexValidator) matchesExcludePatterns(filePath string) bool {
	for _, pattern := range iv.config.ExcludePatterns {
		matched, err := filepath.Match(pattern, filepath.Base(filePath))
		if err != nil {
			continue
		}
		if matched {
			return true
		}

		// Also check full path
		if strings.Contains(pattern, "/") {
			matched, err = filepath.Match(pattern, filePath)
			if err == nil && matched {
				return true
			}
		}
	}

	return false
}

// calculateConfigFingerprint calculates a fingerprint of the current configuration
func (iv *IndexValidator) calculateConfigFingerprint() string {
	// Create a string representation of the config that matters for indexing
	var parts []string

	// Scan paths
	parts = append(parts, fmt.Sprintf("scan_paths:%v", iv.config.ScanPaths))

	// Include patterns
	if len(iv.config.IncludePatterns) > 0 {
		parts = append(parts, fmt.Sprintf("include:%v", iv.config.IncludePatterns))
	}

	// Exclude patterns
	if len(iv.config.ExcludePatterns) > 0 {
		parts = append(parts, fmt.Sprintf("exclude:%v", iv.config.ExcludePatterns))
	}

	configStr := strings.Join(parts, "|")

	// Simple hash
	return fmt.Sprintf("%x", configStr)
}

// GetFileChecksums calculates checksums for a set of files (slow operation)
func (iv *IndexValidator) GetFileChecksums(paths []string) (map[string]string, error) {
	checksums := make(map[string]string)

	for _, path := range paths {
		checksum, err := CalculateFileChecksum(path)
		if err != nil {
			iv.logger.Warn("Failed to calculate checksum",
				logger.String("file", path),
				logger.Field{Key: "error", Value: err})
			continue
		}
		checksums[path] = checksum
	}

	return checksums, nil
}

// QuickValidation performs a fast validation check (mtime/size only)
func (iv *IndexValidator) QuickValidation(index *PersistedIndex) bool {
	hasChanges, _, err := iv.HasFileChanges(index)
	if err != nil {
		return true // Assume rebuild needed on error
	}
	return hasChanges
}

// FullValidation performs complete validation including checksums
func (iv *IndexValidator) FullValidation(index *PersistedIndex) ValidationResult {
	result := iv.NeedsRebuild(index)

	// If quick checks say rebuild needed, return immediately
	if result.NeedsRebuild {
		return result
	}

	// Perform checksum validation on all files
	currentFiles, err := iv.getCurrentTerraformFiles()
	if err != nil {
		return ValidationResult{
			NeedsRebuild: true,
			Reason:       ReasonIndexCorrupted,
			Details:      fmt.Sprintf("Failed to get current files: %v", err),
		}
	}

	var checksumMismatches []string
	for path, currentInfo := range currentFiles {
		indexedInfo, exists := index.SourceFiles[path]
		if !exists {
			continue // Already caught by file comparison
		}

		// Calculate current checksum
		checksum, err := CalculateFileChecksum(path)
		if err != nil {
			iv.logger.Warn("Failed to calculate checksum", logger.String("file", path))
			continue
		}

		currentInfo.Checksum = checksum

		// Compare with indexed checksum
		if checksum != indexedInfo.Checksum {
			checksumMismatches = append(checksumMismatches, path)
		}
	}

	if len(checksumMismatches) > 0 {
		return ValidationResult{
			NeedsRebuild: true,
			Reason:       ReasonChecksumMismatch,
			Details:      fmt.Sprintf("%d files have different checksums despite matching mtime", len(checksumMismatches)),
			ChangedFiles: checksumMismatches,
		}
	}

	return ValidationResult{
		NeedsRebuild: false,
		Reason:       ReasonNone,
		Details:      "Index is up to date (full validation passed)",
	}
}
