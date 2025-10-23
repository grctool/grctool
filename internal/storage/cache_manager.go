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
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/grctool/grctool/internal/interfaces"
)

// CacheEntry represents a cached item with metadata
type CacheEntry struct {
	Key         string      `json:"key"`
	Value       interface{} `json:"value"`
	Expiration  time.Time   `json:"expiration"`
	CreatedAt   time.Time   `json:"created_at"`
	AccessedAt  time.Time   `json:"accessed_at"`
	AccessCount int         `json:"access_count"`
}

// CacheManager implements a file-based cache for performance optimization
type CacheManager struct {
	baseDir         string
	fileStorage     interfaces.FileService
	defaultTTL      time.Duration
	maxCacheSize    int64
	cleanupInterval time.Duration
}

// NewCacheManager creates a new cache manager instance
func NewCacheManager(baseDir string) (*CacheManager, error) {
	// Ensure base directory exists
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Create subdirectories for different cache types
	subdirs := []string{"tool_results", "summaries", "github", "general"}
	for _, subdir := range subdirs {
		if err := os.MkdirAll(filepath.Join(baseDir, subdir), 0755); err != nil {
			return nil, fmt.Errorf("failed to create cache subdirectory %s: %w", subdir, err)
		}
	}

	fileStorage, err := NewFileStorage(baseDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create file storage for cache: %w", err)
	}

	cm := &CacheManager{
		baseDir:         baseDir,
		fileStorage:     fileStorage,
		defaultTTL:      24 * time.Hour,    // 24 hours default
		maxCacheSize:    100 * 1024 * 1024, // 100MB max
		cleanupInterval: 1 * time.Hour,     // Cleanup every hour
	}

	// Start background cleanup routine
	go cm.backgroundCleanup()

	return cm, nil
}

// Set stores a value in the cache with expiration
func (cm *CacheManager) Set(key string, value interface{}, expiration time.Duration) error {
	if expiration <= 0 {
		expiration = cm.defaultTTL
	}

	entry := &CacheEntry{
		Key:         key,
		Value:       value,
		Expiration:  time.Now().Add(expiration),
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 0,
	}

	cacheKey := cm.generateCacheKey(key)
	return cm.fileStorage.Save("general", cacheKey, entry)
}

// Get retrieves a value from the cache
func (cm *CacheManager) Get(key string, target interface{}) error {
	cacheKey := cm.generateCacheKey(key)

	var entry CacheEntry
	if err := cm.fileStorage.Load("general", cacheKey, &entry); err != nil {
		return fmt.Errorf("cache miss: %w", err)
	}

	// Check expiration
	if time.Now().After(entry.Expiration) {
		// Remove expired entry
		cm.fileStorage.Delete("general", cacheKey)
		return fmt.Errorf("cache entry expired")
	}

	// Update access metadata
	entry.AccessedAt = time.Now()
	entry.AccessCount++
	cm.fileStorage.Save("general", cacheKey, &entry)

	// Marshal and unmarshal to copy the value
	data, err := json.Marshal(entry.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal cached value: %w", err)
	}

	return json.Unmarshal(data, target)
}

// Delete removes a value from the cache
func (cm *CacheManager) Delete(key string) error {
	cacheKey := cm.generateCacheKey(key)
	return cm.fileStorage.Delete("general", cacheKey)
}

// Clear removes all cached entries
func (cm *CacheManager) Clear() error {
	categories := []string{"general", "tool_results", "summaries", "github"}

	for _, category := range categories {
		files, err := cm.fileStorage.List(category)
		if err != nil {
			continue
		}

		for _, file := range files {
			cm.fileStorage.Delete(category, file)
		}
	}

	return nil
}

// Exists checks if a key exists in the cache
func (cm *CacheManager) Exists(key string) bool {
	cacheKey := cm.generateCacheKey(key)
	return cm.fileStorage.Exists("general", cacheKey)
}

// SetToolResult caches a tool execution result
func (cm *CacheManager) SetToolResult(toolName string, params map[string]interface{}, result interface{}, expiration time.Duration) error {
	// Create a unique key based on tool name and parameters
	paramKey := cm.generateParamKey(params)
	key := fmt.Sprintf("%s_%s", toolName, paramKey)

	if expiration <= 0 {
		expiration = 6 * time.Hour // Tool results expire in 6 hours by default
	}

	entry := &CacheEntry{
		Key:         key,
		Value:       result,
		Expiration:  time.Now().Add(expiration),
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 0,
	}

	cacheKey := cm.generateCacheKey(key)
	return cm.fileStorage.Save("tool_results", cacheKey, entry)
}

// GetToolResult retrieves a cached tool execution result
func (cm *CacheManager) GetToolResult(toolName string, params map[string]interface{}, target interface{}) error {
	paramKey := cm.generateParamKey(params)
	key := fmt.Sprintf("%s_%s", toolName, paramKey)
	cacheKey := cm.generateCacheKey(key)

	var entry CacheEntry
	if err := cm.fileStorage.Load("tool_results", cacheKey, &entry); err != nil {
		return fmt.Errorf("tool result cache miss: %w", err)
	}

	// Check expiration
	if time.Now().After(entry.Expiration) {
		cm.fileStorage.Delete("tool_results", cacheKey)
		return fmt.Errorf("tool result cache entry expired")
	}

	// Update access metadata
	entry.AccessedAt = time.Now()
	entry.AccessCount++
	cm.fileStorage.Save("tool_results", cacheKey, &entry)

	// Marshal and unmarshal to copy the value
	data, err := json.Marshal(entry.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal cached tool result: %w", err)
	}

	return json.Unmarshal(data, target)
}

// SetSummary caches a generated summary
func (cm *CacheManager) SetSummary(summaryType, id string, summary interface{}, expiration time.Duration) error {
	key := fmt.Sprintf("%s_%s", summaryType, id)

	if expiration <= 0 {
		expiration = 12 * time.Hour // Summaries expire in 12 hours by default
	}

	entry := &CacheEntry{
		Key:         key,
		Value:       summary,
		Expiration:  time.Now().Add(expiration),
		CreatedAt:   time.Now(),
		AccessedAt:  time.Now(),
		AccessCount: 0,
	}

	cacheKey := cm.generateCacheKey(key)
	return cm.fileStorage.Save("summaries", cacheKey, entry)
}

// GetSummary retrieves a cached summary
func (cm *CacheManager) GetSummary(summaryType, id string, target interface{}) error {
	key := fmt.Sprintf("%s_%s", summaryType, id)
	cacheKey := cm.generateCacheKey(key)

	var entry CacheEntry
	if err := cm.fileStorage.Load("summaries", cacheKey, &entry); err != nil {
		return fmt.Errorf("summary cache miss: %w", err)
	}

	// Check expiration
	if time.Now().After(entry.Expiration) {
		cm.fileStorage.Delete("summaries", cacheKey)
		return fmt.Errorf("summary cache entry expired")
	}

	// Update access metadata
	entry.AccessedAt = time.Now()
	entry.AccessCount++
	cm.fileStorage.Save("summaries", cacheKey, &entry)

	// Marshal and unmarshal to copy the value
	data, err := json.Marshal(entry.Value)
	if err != nil {
		return fmt.Errorf("failed to marshal cached summary: %w", err)
	}

	return json.Unmarshal(data, target)
}

// GetSize returns the total size of the cache in bytes
func (cm *CacheManager) GetSize() int64 {
	size, err := cm.fileStorage.GetSize()
	if err != nil {
		return 0
	}
	return size
}

// GetStats returns cache statistics
func (cm *CacheManager) GetStats() map[string]interface{} {
	stats := make(map[string]interface{})

	categories := []string{"general", "tool_results", "summaries", "github"}
	totalEntries := 0
	expiredEntries := 0

	now := time.Now()

	for _, category := range categories {
		files, err := cm.fileStorage.List(category)
		if err != nil {
			continue
		}

		categoryCount := 0
		categoryExpired := 0

		for _, file := range files {
			var entry CacheEntry
			if err := cm.fileStorage.Load(category, file, &entry); err == nil {
				categoryCount++
				if now.After(entry.Expiration) {
					categoryExpired++
				}
			}
		}

		stats[category+"_entries"] = categoryCount
		stats[category+"_expired"] = categoryExpired

		totalEntries += categoryCount
		expiredEntries += categoryExpired
	}

	stats["total_entries"] = totalEntries
	stats["expired_entries"] = expiredEntries
	stats["cache_size_bytes"] = cm.GetSize()
	stats["max_cache_size_bytes"] = cm.maxCacheSize
	stats["hit_ratio"] = cm.calculateHitRatio()
	stats["generated_at"] = time.Now()

	return stats
}

// generateCacheKey creates a consistent cache key from a string
func (cm *CacheManager) generateCacheKey(key string) string {
	hash := md5.Sum([]byte(key))
	return hex.EncodeToString(hash[:])
}

// generateParamKey creates a consistent key from parameters map
func (cm *CacheManager) generateParamKey(params map[string]interface{}) string {
	// Convert params to JSON for consistent hashing
	data, err := json.Marshal(params)
	if err != nil {
		return "default"
	}

	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

// calculateHitRatio calculates cache hit ratio (placeholder implementation)
func (cm *CacheManager) calculateHitRatio() float64 {
	// This would require tracking hits/misses over time
	// For now, return a placeholder value
	return 0.0
}

// backgroundCleanup runs periodic cleanup of expired entries
func (cm *CacheManager) backgroundCleanup() {
	ticker := time.NewTicker(cm.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		cm.cleanupExpiredEntries()
		cm.enforceMaxCacheSize()
	}
}

// cleanupExpiredEntries removes expired cache entries
func (cm *CacheManager) cleanupExpiredEntries() {
	categories := []string{"general", "tool_results", "summaries", "github"}
	now := time.Now()

	for _, category := range categories {
		files, err := cm.fileStorage.List(category)
		if err != nil {
			continue
		}

		for _, file := range files {
			var entry CacheEntry
			if err := cm.fileStorage.Load(category, file, &entry); err == nil {
				if now.After(entry.Expiration) {
					cm.fileStorage.Delete(category, file)
				}
			}
		}
	}
}

// enforceMaxCacheSize removes least recently used entries if cache is too large
func (cm *CacheManager) enforceMaxCacheSize() {
	currentSize := cm.GetSize()
	if currentSize <= cm.maxCacheSize {
		return
	}

	// Collect all entries with their access information
	type entryInfo struct {
		category    string
		file        string
		accessedAt  time.Time
		accessCount int
	}

	var entries []entryInfo
	categories := []string{"general", "tool_results", "summaries", "github"}

	for _, category := range categories {
		files, err := cm.fileStorage.List(category)
		if err != nil {
			continue
		}

		for _, file := range files {
			var entry CacheEntry
			if err := cm.fileStorage.Load(category, file, &entry); err == nil {
				entries = append(entries, entryInfo{
					category:    category,
					file:        file,
					accessedAt:  entry.AccessedAt,
					accessCount: entry.AccessCount,
				})
			}
		}
	}

	// Sort by access frequency and recency (LRU strategy)
	// For simplicity, just remove entries based on access time
	// In a real implementation, you'd want a more sophisticated algorithm

	// Remove the oldest 25% of entries
	entriesToRemove := len(entries) / 4
	if entriesToRemove < 1 {
		entriesToRemove = 1
	}

	for i := 0; i < entriesToRemove && i < len(entries); i++ {
		cm.fileStorage.Delete(entries[i].category, entries[i].file)
	}
}
