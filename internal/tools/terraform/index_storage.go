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
	"compress/gzip"
	"crypto/md5"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/grctool/grctool/internal/logger"
)

const (
	// IndexVersion is the current version of the index format
	IndexVersion = "1.0.0"

	// IndexFileName is the default name for the index file
	IndexFileName = "index.json.gz"
)

// PersistedIndex wraps SecurityIndex with versioning and file tracking
type PersistedIndex struct {
	Version           string                     `json:"version"`
	IndexedAt         time.Time                  `json:"indexed_at"`
	TerraformVersion  string                     `json:"terraform_version,omitempty"`
	ToolVersion       string                     `json:"tool_version"`
	Metadata          IndexMetadata              `json:"metadata"`
	SourceFiles       map[string]*SourceFileInfo `json:"source_files"`
	Index             *SecurityIndex             `json:"index"`
	Statistics        *IndexStatistics           `json:"statistics"`
	ConfigFingerprint string                     `json:"config_fingerprint"`
}

// IndexMetadata contains metadata about the index
type IndexMetadata struct {
	TotalFiles        int      `json:"total_files"`
	TotalResources    int      `json:"total_resources"`
	ScanDurationMs    int64    `json:"scan_duration_ms"`
	SourceDirectories []string `json:"source_directories"`
	IncludePatterns   []string `json:"include_patterns"`
	ExcludePatterns   []string `json:"exclude_patterns"`
}

// SourceFileInfo tracks information about a source Terraform file
type SourceFileInfo struct {
	Path      string    `json:"path"`
	ModTime   time.Time `json:"mtime"`
	SizeBytes int64     `json:"size_bytes"`
	Checksum  string    `json:"checksum"`
}

// IndexStatistics contains aggregate statistics from the index
type IndexStatistics struct {
	ComplianceCoverage      float64                         `json:"compliance_coverage"`
	SecurityFindings        map[string]int                  `json:"security_findings"`
	ControlCoverage         map[string]ControlCoverageStats `json:"control_coverage"`
	AttributeDistribution   map[string]int                  `json:"attribute_distribution"`
	EnvironmentDistribution map[string]int                  `json:"environment_distribution"`
}

// ControlCoverageStats tracks coverage for a specific control
type ControlCoverageStats struct {
	TotalResources     int     `json:"total_resources"`
	CompliantResources int     `json:"compliant_resources"`
	ComplianceRate     float64 `json:"compliance_rate"`
}

// IndexStorage handles persistence of Terraform security indices
type IndexStorage struct {
	indexPath string
	logger    logger.Logger
}

// NewIndexStorage creates a new index storage handler
func NewIndexStorage(indexPath string, log logger.Logger) *IndexStorage {
	return &IndexStorage{
		indexPath: indexPath,
		logger:    log,
	}
}

// SaveIndex saves the index to disk with compression and atomic writes
func (is *IndexStorage) SaveIndex(index *PersistedIndex) error {
	is.logger.Debug("Saving Terraform index", logger.String("path", is.indexPath))

	// Ensure directory exists
	dir := filepath.Dir(is.indexPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create index directory: %w", err)
	}

	// Set version and timestamp
	index.Version = IndexVersion
	index.IndexedAt = time.Now()
	index.ToolVersion = "grctool-" + IndexVersion

	// Use temporary file for atomic write
	tempPath := is.indexPath + ".tmp"
	defer os.Remove(tempPath) // Clean up temp file if we error out

	// Create temp file
	tempFile, err := os.Create(tempPath)
	if err != nil {
		return fmt.Errorf("failed to create temp index file: %w", err)
	}
	defer tempFile.Close()

	// Create gzip writer for compression
	gzipWriter := gzip.NewWriter(tempFile)
	defer gzipWriter.Close()

	// Encode index as JSON
	encoder := json.NewEncoder(gzipWriter)
	encoder.SetIndent("", "  ") // Pretty print for debuggability
	if err := encoder.Encode(index); err != nil {
		return fmt.Errorf("failed to encode index: %w", err)
	}

	// Close gzip writer to flush
	if err := gzipWriter.Close(); err != nil {
		return fmt.Errorf("failed to close gzip writer: %w", err)
	}

	// Close temp file
	if err := tempFile.Close(); err != nil {
		return fmt.Errorf("failed to close temp file: %w", err)
	}

	// Atomic rename (on most filesystems this is atomic)
	if err := os.Rename(tempPath, is.indexPath); err != nil {
		return fmt.Errorf("failed to rename temp index to final path: %w", err)
	}

	is.logger.Info("Successfully saved Terraform index",
		logger.String("path", is.indexPath),
		logger.Int("resources", index.Metadata.TotalResources),
		logger.Int("files", index.Metadata.TotalFiles))

	return nil
}

// LoadIndex loads the index from disk
func (is *IndexStorage) LoadIndex() (*PersistedIndex, error) {
	is.logger.Debug("Loading Terraform index", logger.String("path", is.indexPath))

	// Check if index exists
	if !is.IndexExists() {
		return nil, fmt.Errorf("index does not exist at %s", is.indexPath)
	}

	// Open index file
	file, err := os.Open(is.indexPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open index file: %w", err)
	}
	defer file.Close()

	// Create gzip reader for decompression
	gzipReader, err := gzip.NewReader(file)
	if err != nil {
		return nil, fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzipReader.Close()

	// Decode JSON
	var index PersistedIndex
	decoder := json.NewDecoder(gzipReader)
	if err := decoder.Decode(&index); err != nil {
		return nil, fmt.Errorf("failed to decode index: %w", err)
	}

	// Validate index
	if err := is.validateIndex(&index); err != nil {
		return nil, fmt.Errorf("index validation failed: %w", err)
	}

	is.logger.Info("Successfully loaded Terraform index",
		logger.String("path", is.indexPath),
		logger.Field{Key: "indexed_at", Value: index.IndexedAt},
		logger.Int("resources", index.Metadata.TotalResources))

	return &index, nil
}

// IndexExists checks if the index file exists
func (is *IndexStorage) IndexExists() bool {
	_, err := os.Stat(is.indexPath)
	return err == nil
}

// GetIndexAge returns how long ago the index was created
func (is *IndexStorage) GetIndexAge() (time.Duration, error) {
	if !is.IndexExists() {
		return 0, fmt.Errorf("index does not exist")
	}

	info, err := os.Stat(is.indexPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat index file: %w", err)
	}

	return time.Since(info.ModTime()), nil
}

// GetIndexSize returns the size of the index file in bytes
func (is *IndexStorage) GetIndexSize() (int64, error) {
	if !is.IndexExists() {
		return 0, fmt.Errorf("index does not exist")
	}

	info, err := os.Stat(is.indexPath)
	if err != nil {
		return 0, fmt.Errorf("failed to stat index file: %w", err)
	}

	return info.Size(), nil
}

// DeleteIndex removes the index file
func (is *IndexStorage) DeleteIndex() error {
	if !is.IndexExists() {
		return nil // Already deleted
	}

	if err := os.Remove(is.indexPath); err != nil {
		return fmt.Errorf("failed to delete index: %w", err)
	}

	is.logger.Info("Deleted Terraform index", logger.String("path", is.indexPath))
	return nil
}

// validateIndex performs validation checks on a loaded index
func (is *IndexStorage) validateIndex(index *PersistedIndex) error {
	// Check version compatibility
	if index.Version == "" {
		return fmt.Errorf("index has no version")
	}

	// For now, we only support exact version match
	// In the future, could implement migration logic
	if index.Version != IndexVersion {
		return fmt.Errorf("incompatible index version: %s (expected %s)", index.Version, IndexVersion)
	}

	// Check that index has data
	if index.Index == nil {
		return fmt.Errorf("index has no data")
	}

	// Check metadata consistency
	if index.Metadata.TotalResources != len(index.Index.IndexedResources) {
		is.logger.Warn("Index metadata mismatch",
			logger.Int("metadata_count", index.Metadata.TotalResources),
			logger.Int("actual_count", len(index.Index.IndexedResources)))
	}

	return nil
}

// CalculateFileChecksum calculates MD5 checksum of a file
func CalculateFileChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// BuildSourceFileInfo builds SourceFileInfo for a given file
func BuildSourceFileInfo(filePath string) (*SourceFileInfo, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}

	checksum, err := CalculateFileChecksum(filePath)
	if err != nil {
		return nil, err
	}

	return &SourceFileInfo{
		Path:      filePath,
		ModTime:   info.ModTime(),
		SizeBytes: info.Size(),
		Checksum:  checksum,
	}, nil
}

// GetIndexInfo returns summary information about the index
func (is *IndexStorage) GetIndexInfo() (map[string]interface{}, error) {
	if !is.IndexExists() {
		return map[string]interface{}{
			"exists": false,
		}, nil
	}

	info, err := os.Stat(is.indexPath)
	if err != nil {
		return nil, err
	}

	// Try to load and get details
	index, err := is.LoadIndex()
	if err != nil {
		// Return basic info even if load fails
		return map[string]interface{}{
			"exists":    true,
			"path":      is.indexPath,
			"size":      info.Size(),
			"mod_time":  info.ModTime(),
			"age":       time.Since(info.ModTime()).String(),
			"corrupted": true,
			"error":     err.Error(),
		}, nil
	}

	return map[string]interface{}{
		"exists":              true,
		"path":                is.indexPath,
		"size":                info.Size(),
		"mod_time":            info.ModTime(),
		"age":                 time.Since(info.ModTime()).String(),
		"version":             index.Version,
		"indexed_at":          index.IndexedAt,
		"total_resources":     index.Metadata.TotalResources,
		"total_files":         index.Metadata.TotalFiles,
		"scan_duration_ms":    index.Metadata.ScanDurationMs,
		"compliance_coverage": index.Statistics.ComplianceCoverage,
		"corrupted":           false,
	}, nil
}
