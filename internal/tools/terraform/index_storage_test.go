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
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/logger"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestIndex creates a minimal PersistedIndex for testing
func createTestIndex() *PersistedIndex {
	return &PersistedIndex{
		Version:     IndexVersion,
		IndexedAt:   time.Now(),
		ToolVersion: "grctool-" + IndexVersion,
		Metadata: IndexMetadata{
			TotalFiles:        5,
			TotalResources:    10,
			ScanDurationMs:    1000,
			SourceDirectories: []string{"/test/terraform"},
		},
		SourceFiles: map[string]*SourceFileInfo{
			"/test/main.tf": {
				Path:      "/test/main.tf",
				ModTime:   time.Now(),
				SizeBytes: 1024,
				Checksum:  "abc123",
			},
		},
		Index: &SecurityIndex{
			IndexedResources: []IndexedResource{
				{
					ResourceType:       "aws_s3_bucket",
					ResourceID:         "test-bucket",
					FilePath:           "/test/main.tf",
					LineRange:          "10-20",
					ControlRelevance:   []string{"CC-06.1"},
					SecurityAttributes: []string{"encryption"},
					ComplianceStatus:   "compliant",
					RiskLevel:          "low",
					Configuration: map[string]interface{}{
						"encryption": "AES256",
					},
				},
			},
			ControlMapping: map[string][]IndexedResource{
				"CC-06.1": {
					{
						ResourceType:       "aws_s3_bucket",
						ResourceID:         "test-bucket",
						FilePath:           "/test/main.tf",
						LineRange:          "10-20",
						ControlRelevance:   []string{"CC-06.1"},
						SecurityAttributes: []string{"encryption"},
					},
				},
			},
			SecurityAttributes: map[string]SecurityAttributeDetails{
				"encryption": {
					AttributeName: "encryption",
					ResourceCount: 1,
				},
			},
		},
		Statistics: &IndexStatistics{
			ComplianceCoverage: 0.85,
			SecurityFindings: map[string]int{
				"encryption_enabled": 1,
			},
			ControlCoverage: map[string]ControlCoverageStats{
				"CC-06.1": {
					TotalResources:     1,
					CompliantResources: 1,
					ComplianceRate:     1.0,
				},
			},
			AttributeDistribution: map[string]int{
				"encryption": 1,
			},
		},
		ConfigFingerprint: "test-fingerprint",
	}
}

// TestSaveIndex tests saving an index with compression and atomic writes
func TestSaveIndex(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) (string, *PersistedIndex)
		wantErr   bool
	}{
		{
			name: "save valid index",
			setupFunc: func(t *testing.T) (string, *PersistedIndex) {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "test_index.json.gz")
				return indexPath, createTestIndex()
			},
			wantErr: false,
		},
		{
			name: "create directory if not exists",
			setupFunc: func(t *testing.T) (string, *PersistedIndex) {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "subdir", "nested", "index.json.gz")
				return indexPath, createTestIndex()
			},
			wantErr: false,
		},
		{
			name: "overwrite existing index",
			setupFunc: func(t *testing.T) (string, *PersistedIndex) {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "index.json.gz")

				// Create an existing file
				err := os.WriteFile(indexPath, []byte("old data"), 0644)
				require.NoError(t, err)

				return indexPath, createTestIndex()
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexPath, index := tt.setupFunc(t)
			log, _ := logger.NewTestLogger()
			storage := NewIndexStorage(indexPath, log)

			// Save the index
			err := storage.SaveIndex(index)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)

			// Verify file exists
			assert.FileExists(t, indexPath)

			// Verify file is gzipped
			file, err := os.Open(indexPath)
			require.NoError(t, err)
			defer file.Close()

			gzReader, err := gzip.NewReader(file)
			require.NoError(t, err, "file should be gzip compressed")
			defer gzReader.Close()

			// Verify JSON can be decoded
			var loaded PersistedIndex
			decoder := json.NewDecoder(gzReader)
			err = decoder.Decode(&loaded)
			require.NoError(t, err, "should decode JSON successfully")

			// Verify version was set
			assert.Equal(t, IndexVersion, loaded.Version)

			// Verify tool version was set
			assert.Equal(t, "grctool-"+IndexVersion, loaded.ToolVersion)

			// Verify timestamp was set (should be recent)
			assert.WithinDuration(t, time.Now(), loaded.IndexedAt, 5*time.Second)

			// Verify data integrity
			assert.Equal(t, index.Metadata.TotalResources, loaded.Metadata.TotalResources)
			assert.Equal(t, index.Metadata.TotalFiles, loaded.Metadata.TotalFiles)
			assert.Len(t, loaded.Index.IndexedResources, len(index.Index.IndexedResources))

			// Verify temp file was cleaned up
			tempPath := indexPath + ".tmp"
			assert.NoFileExists(t, tempPath, "temp file should be cleaned up")
		})
	}
}

// TestLoadIndex tests loading and decompressing an index
func TestLoadIndex(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string
		wantErr   bool
		errMsg    string
	}{
		{
			name: "load valid index",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "index.json.gz")
				log, _ := logger.NewTestLogger()
				storage := NewIndexStorage(indexPath, log)

				index := createTestIndex()
				err := storage.SaveIndex(index)
				require.NoError(t, err)

				return indexPath
			},
			wantErr: false,
		},
		{
			name: "error on missing index",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "nonexistent.json.gz")
			},
			wantErr: true,
			errMsg:  "index does not exist",
		},
		{
			name: "error on corrupted gzip",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "corrupted.json.gz")

				// Write invalid gzip data
				err := os.WriteFile(indexPath, []byte("not gzipped data"), 0644)
				require.NoError(t, err)

				return indexPath
			},
			wantErr: true,
			errMsg:  "failed to create gzip reader",
		},
		{
			name: "error on invalid JSON",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "invalid.json.gz")

				// Write valid gzip but invalid JSON
				file, err := os.Create(indexPath)
				require.NoError(t, err)
				defer file.Close()

				gzWriter := gzip.NewWriter(file)
				_, err = gzWriter.Write([]byte("{invalid json"))
				require.NoError(t, err)
				gzWriter.Close()

				return indexPath
			},
			wantErr: true,
			errMsg:  "failed to decode index",
		},
		{
			name: "error on version mismatch",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "old_version.json.gz")

				index := createTestIndex()
				index.Version = "0.5.0" // Old version

				file, err := os.Create(indexPath)
				require.NoError(t, err)
				defer file.Close()

				gzWriter := gzip.NewWriter(file)
				encoder := json.NewEncoder(gzWriter)
				err = encoder.Encode(index)
				require.NoError(t, err)
				gzWriter.Close()

				return indexPath
			},
			wantErr: true,
			errMsg:  "incompatible index version",
		},
		{
			name: "error on missing index data",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "no_data.json.gz")

				index := createTestIndex()
				index.Index = nil // Missing index data

				file, err := os.Create(indexPath)
				require.NoError(t, err)
				defer file.Close()

				gzWriter := gzip.NewWriter(file)
				encoder := json.NewEncoder(gzWriter)
				err = encoder.Encode(index)
				require.NoError(t, err)
				gzWriter.Close()

				return indexPath
			},
			wantErr: true,
			errMsg:  "index has no data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexPath := tt.setupFunc(t)
			log, _ := logger.NewTestLogger()
			storage := NewIndexStorage(indexPath, log)

			index, err := storage.LoadIndex()

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, index)
			assert.Equal(t, IndexVersion, index.Version)
			assert.NotNil(t, index.Index)
		})
	}
}

// TestIndexExists tests checking if an index file exists
func TestIndexExists(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string
		want      bool
	}{
		{
			name: "index exists",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "exists.json.gz")

				err := os.WriteFile(indexPath, []byte("test"), 0644)
				require.NoError(t, err)

				return indexPath
			},
			want: true,
		},
		{
			name: "index does not exist",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "missing.json.gz")
			},
			want: false,
		},
		{
			name: "directory does not exist",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "missing", "dir", "index.json.gz")
			},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexPath := tt.setupFunc(t)
			log, _ := logger.NewTestLogger()
			storage := NewIndexStorage(indexPath, log)

			exists := storage.IndexExists()
			assert.Equal(t, tt.want, exists)
		})
	}
}

// TestGetIndexAge tests retrieving the age of the index
func TestGetIndexAge(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) (string, time.Time)
		wantErr   bool
		checkAge  func(t *testing.T, age time.Duration, created time.Time)
	}{
		{
			name: "get age of recent index",
			setupFunc: func(t *testing.T) (string, time.Time) {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "index.json.gz")

				now := time.Now()
				err := os.WriteFile(indexPath, []byte("test"), 0644)
				require.NoError(t, err)

				return indexPath, now
			},
			wantErr: false,
			checkAge: func(t *testing.T, age time.Duration, created time.Time) {
				assert.Greater(t, age, time.Duration(0))
				assert.Less(t, age, 5*time.Second, "age should be less than 5 seconds")
			},
		},
		{
			name: "error on missing index",
			setupFunc: func(t *testing.T) (string, time.Time) {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "missing.json.gz"), time.Now()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexPath, created := tt.setupFunc(t)
			log, _ := logger.NewTestLogger()
			storage := NewIndexStorage(indexPath, log)

			age, err := storage.GetIndexAge()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			if tt.checkAge != nil {
				tt.checkAge(t, age, created)
			}
		})
	}
}

// TestGetIndexSize tests retrieving the size of the index file
func TestGetIndexSize(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) (string, int64)
		wantErr   bool
	}{
		{
			name: "get size of index",
			setupFunc: func(t *testing.T) (string, int64) {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "index.json.gz")

				data := []byte("test data with some content")
				err := os.WriteFile(indexPath, data, 0644)
				require.NoError(t, err)

				return indexPath, int64(len(data))
			},
			wantErr: false,
		},
		{
			name: "error on missing index",
			setupFunc: func(t *testing.T) (string, int64) {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "missing.json.gz"), 0
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexPath, expectedSize := tt.setupFunc(t)
			log, _ := logger.NewTestLogger()
			storage := NewIndexStorage(indexPath, log)

			size, err := storage.GetIndexSize()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, expectedSize, size)
		})
	}
}

// TestDeleteIndex tests deleting an index file
func TestDeleteIndex(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string
		wantErr   bool
	}{
		{
			name: "delete existing index",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "index.json.gz")

				err := os.WriteFile(indexPath, []byte("test"), 0644)
				require.NoError(t, err)

				return indexPath
			},
			wantErr: false,
		},
		{
			name: "delete non-existent index (no error)",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "missing.json.gz")
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexPath := tt.setupFunc(t)
			log, _ := logger.NewTestLogger()
			storage := NewIndexStorage(indexPath, log)

			err := storage.DeleteIndex()

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.False(t, storage.IndexExists(), "index should not exist after deletion")
		})
	}
}

// TestValidateIndex tests index validation logic
func TestValidateIndex(t *testing.T) {
	tests := []struct {
		name    string
		index   *PersistedIndex
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid index",
			index:   createTestIndex(),
			wantErr: false,
		},
		{
			name: "missing version",
			index: func() *PersistedIndex {
				idx := createTestIndex()
				idx.Version = ""
				return idx
			}(),
			wantErr: true,
			errMsg:  "index has no version",
		},
		{
			name: "incompatible version",
			index: func() *PersistedIndex {
				idx := createTestIndex()
				idx.Version = "0.5.0"
				return idx
			}(),
			wantErr: true,
			errMsg:  "incompatible index version",
		},
		{
			name: "missing index data",
			index: func() *PersistedIndex {
				idx := createTestIndex()
				idx.Index = nil
				return idx
			}(),
			wantErr: true,
			errMsg:  "index has no data",
		},
		{
			name: "metadata mismatch (warning, not error)",
			index: func() *PersistedIndex {
				idx := createTestIndex()
				idx.Metadata.TotalResources = 999 // Doesn't match actual count
				return idx
			}(),
			wantErr: false, // Just a warning
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			log, _ := logger.NewTestLogger()
			storage := NewIndexStorage("/tmp/test.json.gz", log)

			err := storage.validateIndex(tt.index)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.errMsg != "" {
					assert.Contains(t, err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// TestCalculateFileChecksum tests MD5 checksum calculation
func TestCalculateFileChecksum(t *testing.T) {
	tests := []struct {
		name         string
		setupFunc    func(t *testing.T) string
		wantErr      bool
		wantChecksum string
	}{
		{
			name: "calculate checksum of file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "test.txt")

				err := os.WriteFile(filePath, []byte("hello world"), 0644)
				require.NoError(t, err)

				return filePath
			},
			wantErr:      false,
			wantChecksum: "5eb63bbbe01eeed093cb22bb8f5acdc3", // MD5 of "hello world"
		},
		{
			name: "error on missing file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "missing.txt")
			},
			wantErr: true,
		},
		{
			name: "checksum of empty file",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "empty.txt")

				err := os.WriteFile(filePath, []byte(""), 0644)
				require.NoError(t, err)

				return filePath
			},
			wantErr:      false,
			wantChecksum: "d41d8cd98f00b204e9800998ecf8427e", // MD5 of empty string
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := tt.setupFunc(t)

			checksum, err := CalculateFileChecksum(filePath)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.wantChecksum, checksum)
		})
	}
}

// TestBuildSourceFileInfo tests building source file metadata
func TestBuildSourceFileInfo(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) (string, int64, string)
		wantErr   bool
	}{
		{
			name: "build info for valid file",
			setupFunc: func(t *testing.T) (string, int64, string) {
				tmpDir := t.TempDir()
				filePath := filepath.Join(tmpDir, "main.tf")

				content := []byte("resource \"aws_s3_bucket\" \"test\" {}")
				err := os.WriteFile(filePath, content, 0644)
				require.NoError(t, err)

				checksum := "5eb63bbbe01eeed093cb22bb8f5acdc3" // Will vary
				return filePath, int64(len(content)), checksum
			},
			wantErr: false,
		},
		{
			name: "error on missing file",
			setupFunc: func(t *testing.T) (string, int64, string) {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "missing.tf"), 0, ""
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath, expectedSize, _ := tt.setupFunc(t)

			info, err := BuildSourceFileInfo(filePath)

			if tt.wantErr {
				assert.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.NotNil(t, info)
			assert.Equal(t, filePath, info.Path)
			assert.Equal(t, expectedSize, info.SizeBytes)
			assert.NotEmpty(t, info.Checksum)
			assert.WithinDuration(t, time.Now(), info.ModTime, 5*time.Second)
		})
	}
}

// TestGetIndexInfo tests retrieving index summary information
func TestGetIndexInfo(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T) string
		validate  func(t *testing.T, info map[string]interface{})
	}{
		{
			name: "info for valid index",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "index.json.gz")
				log, _ := logger.NewTestLogger()
				storage := NewIndexStorage(indexPath, log)

				index := createTestIndex()
				err := storage.SaveIndex(index)
				require.NoError(t, err)

				return indexPath
			},
			validate: func(t *testing.T, info map[string]interface{}) {
				assert.True(t, info["exists"].(bool))
				assert.False(t, info["corrupted"].(bool))
				assert.Equal(t, IndexVersion, info["version"])
				assert.NotNil(t, info["indexed_at"])
				assert.Equal(t, 10, info["total_resources"])
				assert.Equal(t, 5, info["total_files"])
			},
		},
		{
			name: "info for missing index",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				return filepath.Join(tmpDir, "missing.json.gz")
			},
			validate: func(t *testing.T, info map[string]interface{}) {
				assert.False(t, info["exists"].(bool))
			},
		},
		{
			name: "info for corrupted index",
			setupFunc: func(t *testing.T) string {
				tmpDir := t.TempDir()
				indexPath := filepath.Join(tmpDir, "corrupted.json.gz")

				// Write invalid data
				err := os.WriteFile(indexPath, []byte("corrupted"), 0644)
				require.NoError(t, err)

				return indexPath
			},
			validate: func(t *testing.T, info map[string]interface{}) {
				assert.True(t, info["exists"].(bool))
				assert.True(t, info["corrupted"].(bool))
				assert.NotEmpty(t, info["error"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			indexPath := tt.setupFunc(t)
			log, _ := logger.NewTestLogger()
			storage := NewIndexStorage(indexPath, log)

			info, err := storage.GetIndexInfo()
			require.NoError(t, err)
			require.NotNil(t, info)

			if tt.validate != nil {
				tt.validate(t, info)
			}
		})
	}
}

// TestAtomicWrite verifies that save operations are atomic
func TestAtomicWrite(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "index.json.gz")
	log, _ := logger.NewTestLogger()
	storage := NewIndexStorage(indexPath, log)

	// Save initial index
	index1 := createTestIndex()
	index1.Metadata.TotalResources = 10
	err := storage.SaveIndex(index1)
	require.NoError(t, err)

	// Save updated index
	index2 := createTestIndex()
	index2.Metadata.TotalResources = 20
	err = storage.SaveIndex(index2)
	require.NoError(t, err)

	// Load and verify we got the latest version
	loaded, err := storage.LoadIndex()
	require.NoError(t, err)
	assert.Equal(t, 20, loaded.Metadata.TotalResources, "should have latest version")

	// Verify temp file doesn't exist
	tempPath := indexPath + ".tmp"
	assert.NoFileExists(t, tempPath, "temp file should be cleaned up")
}

// TestIndexStorageRoundTrip tests full save/load cycle
func TestIndexStorageRoundTrip(t *testing.T) {
	tmpDir := t.TempDir()
	indexPath := filepath.Join(tmpDir, "roundtrip.json.gz")
	log, _ := logger.NewTestLogger()
	storage := NewIndexStorage(indexPath, log)

	// Create original index
	original := createTestIndex()
	original.Metadata.TotalResources = 42
	original.Metadata.TotalFiles = 13
	original.Metadata.ScanDurationMs = 5000
	original.Statistics.ComplianceCoverage = 0.95
	original.ConfigFingerprint = "unique-fingerprint"

	// Save it
	err := storage.SaveIndex(original)
	require.NoError(t, err)

	// Load it back
	loaded, err := storage.LoadIndex()
	require.NoError(t, err)

	// Verify all fields match
	assert.Equal(t, original.Metadata.TotalResources, loaded.Metadata.TotalResources)
	assert.Equal(t, original.Metadata.TotalFiles, loaded.Metadata.TotalFiles)
	assert.Equal(t, original.Metadata.ScanDurationMs, loaded.Metadata.ScanDurationMs)
	assert.Equal(t, original.Statistics.ComplianceCoverage, loaded.Statistics.ComplianceCoverage)
	assert.Equal(t, original.ConfigFingerprint, loaded.ConfigFingerprint)
	assert.Len(t, loaded.Index.IndexedResources, len(original.Index.IndexedResources))
	assert.Len(t, loaded.SourceFiles, len(original.SourceFiles))
}
