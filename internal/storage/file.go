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
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/utils"
)

// FileStorage provides simple file-based storage for any JSON-serializable type
// and implements the FileService interface
type FileStorage struct {
	baseDir           string
	mu                sync.RWMutex
	filenameGenerator *utils.FilenameGenerator
}

// Ensure FileStorage implements the FileService interface
var _ interfaces.FileService = (*FileStorage)(nil)

// NewFileStorage creates a new file-based storage instance
func NewFileStorage(baseDir string) (*FileStorage, error) {
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	return &FileStorage{
		baseDir:           baseDir,
		filenameGenerator: utils.NewFilenameGenerator(),
	}, nil
}

// Save stores an object as a JSON file
// The filename will be: {collection}/{id}.json
func (fs *FileStorage) Save(collection, id string, data interface{}) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	// Create collection directory if it doesn't exist
	collectionDir := filepath.Join(fs.baseDir, collection)
	if err := os.MkdirAll(collectionDir, 0755); err != nil {
		return fmt.Errorf("failed to create collection directory: %w", err)
	}

	// Save as JSON file
	filename := fmt.Sprintf("%s.json", id)
	path := filepath.Join(collectionDir, filename)
	return fs.saveJSONFile(path, data)
}

// Load retrieves an object from a JSON file
func (fs *FileStorage) Load(collection, id string, target interface{}) error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	filename := fmt.Sprintf("%s.json", id)
	path := filepath.Join(fs.baseDir, collection, filename)
	return fs.loadJSONFile(path, target)
}

// List returns all IDs in a collection
func (fs *FileStorage) List(collection string) ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	collectionDir := filepath.Join(fs.baseDir, collection)
	files, err := os.ReadDir(collectionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read collection directory: %w", err)
	}

	var ids []string
	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			id := strings.TrimSuffix(file.Name(), ".json")
			ids = append(ids, id)
		}
	}

	return ids, nil
}

// LoadAll loads all objects from a collection
func (fs *FileStorage) LoadAll(collection string, targetSlice interface{}) error {
	ids, err := fs.List(collection)
	if err != nil {
		return err
	}

	// This is a bit tricky with Go's type system, but we'll handle it in the wrapper
	// For now, just return the IDs and let the caller handle loading each one
	_ = ids
	_ = targetSlice
	return fmt.Errorf("LoadAll not implemented - use List() and Load() for each ID")
}

// Delete removes an object file
func (fs *FileStorage) Delete(collection, id string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	filename := fmt.Sprintf("%s.json", id)
	path := filepath.Join(fs.baseDir, collection, filename)

	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to delete %s/%s: %w", collection, id, err)
	}

	return nil
}

// Exists checks if an object file exists
func (fs *FileStorage) Exists(collection, id string) bool {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	filename := fmt.Sprintf("%s.json", id)
	path := filepath.Join(fs.baseDir, collection, filename)

	_, err := os.Stat(path)
	return err == nil
}

// Clear removes all files from a collection
func (fs *FileStorage) Clear(collection string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	collectionDir := filepath.Join(fs.baseDir, collection)
	files, err := os.ReadDir(collectionDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read collection directory: %w", err)
	}

	for _, file := range files {
		if strings.HasSuffix(file.Name(), ".json") {
			path := filepath.Join(collectionDir, file.Name())
			if err := os.Remove(path); err != nil {
				return fmt.Errorf("failed to remove file %s: %w", path, err)
			}
		}
	}

	return nil
}

// ClearAll removes all data from storage
func (fs *FileStorage) ClearAll() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return os.RemoveAll(fs.baseDir)
}

// GetStats returns storage statistics
func (fs *FileStorage) GetStats() map[string]interface{} {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	stats := map[string]interface{}{
		"base_dir": fs.baseDir,
	}

	// Count files in each collection
	collections, err := os.ReadDir(fs.baseDir)
	if err != nil {
		return stats
	}

	for _, collection := range collections {
		if !collection.IsDir() {
			continue
		}

		collectionDir := filepath.Join(fs.baseDir, collection.Name())
		files, err := os.ReadDir(collectionDir)
		if err != nil {
			continue
		}

		count := 0
		for _, file := range files {
			if strings.HasSuffix(file.Name(), ".json") {
				count++
			}
		}
		stats[collection.Name()+"_count"] = count
	}

	return stats
}

// CreateDirectory creates a new directory at the specified path
func (fs *FileStorage) CreateDirectory(path string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return os.MkdirAll(path, 0755)
}

// DeleteDirectory removes a directory and all its contents
func (fs *FileStorage) DeleteDirectory(path string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	return os.RemoveAll(path)
}

// ListDirectories returns all directories in the given base path
func (fs *FileStorage) ListDirectories(basePath string) ([]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	entries, err := os.ReadDir(basePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory %s: %w", basePath, err)
	}

	var directories []string
	for _, entry := range entries {
		if entry.IsDir() {
			directories = append(directories, entry.Name())
		}
	}

	return directories, nil
}

// GetFullPath returns the full filesystem path for a category and ID
func (fs *FileStorage) GetFullPath(category, id string) string {
	filename := fmt.Sprintf("%s.json", id)
	return filepath.Join(fs.baseDir, category, filename)
}

// GetSize returns the total size of the storage in bytes
func (fs *FileStorage) GetSize() (int64, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	var totalSize int64

	err := filepath.Walk(fs.baseDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			totalSize += info.Size()
		}
		return nil
	})

	if err != nil {
		return 0, fmt.Errorf("failed to calculate storage size: %w", err)
	}

	return totalSize, nil
}

// Private helper methods

func (fs *FileStorage) loadJSONFile(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file %s: %w", path, err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON from %s: %w", path, err)
	}

	return nil
}

func (fs *FileStorage) saveJSONFile(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return fmt.Errorf("failed to write file %s: %w", path, err)
	}

	return nil
}
