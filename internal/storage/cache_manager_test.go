// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestCacheManager(t *testing.T) *CacheManager {
	t.Helper()
	cm, err := NewCacheManager(t.TempDir())
	require.NoError(t, err)
	return cm
}

func TestNewCacheManager(t *testing.T) {
	t.Parallel()

	t.Run("creates subdirectories", func(t *testing.T) {
		t.Parallel()
		cm := newTestCacheManager(t)
		assert.NotNil(t, cm)
		assert.NotEmpty(t, cm.baseDir)
		assert.Equal(t, 24*time.Hour, cm.defaultTTL)
	})
}

func TestCache_SetAndGet(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	// Set with explicit TTL
	err := cm.Set("greeting", "hello world", 1*time.Hour)
	require.NoError(t, err)

	var result string
	err = cm.Get("greeting", &result)
	require.NoError(t, err)
	assert.Equal(t, "hello world", result)
}

func TestCache_SetAndGet_DefaultTTL(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	err := cm.Set("key", map[string]int{"count": 5}, 0) // 0 => default TTL
	require.NoError(t, err)

	var result map[string]int
	err = cm.Get("key", &result)
	require.NoError(t, err)
	assert.Equal(t, 5, result["count"])
}

func TestCache_GetMiss(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	var result string
	err := cm.Get("nonexistent", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache miss")
}

func TestCache_Expiration(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	// Create an entry that is already expired by saving directly with past expiration
	entry := &CacheEntry{
		Key:        "expired-key",
		Value:      "stale",
		Expiration: time.Now().Add(-1 * time.Hour), // expired 1 hour ago
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		AccessedAt: time.Now().Add(-2 * time.Hour),
	}
	cacheKey := cm.generateCacheKey("expired-key")
	err := cm.fileStorage.Save("general", cacheKey, entry)
	require.NoError(t, err)

	var result string
	err = cm.Get("expired-key", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestCache_Delete(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	require.NoError(t, cm.Set("to-delete", "value", 1*time.Hour))
	assert.True(t, cm.Exists("to-delete"))

	err := cm.Delete("to-delete")
	require.NoError(t, err)
	assert.False(t, cm.Exists("to-delete"))
}

func TestCache_Exists(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	assert.False(t, cm.Exists("nope"))

	require.NoError(t, cm.Set("yes", "here", 1*time.Hour))
	assert.True(t, cm.Exists("yes"))
}

func TestCache_Clear(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	require.NoError(t, cm.Set("a", "1", 1*time.Hour))
	require.NoError(t, cm.Set("b", "2", 1*time.Hour))

	err := cm.Clear()
	require.NoError(t, err)

	assert.False(t, cm.Exists("a"))
	assert.False(t, cm.Exists("b"))
}

func TestCache_SetToolResult_GetToolResult(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	params := map[string]interface{}{
		"repository": "org/repo",
		"format":     "matrix",
	}
	toolResult := map[string]interface{}{
		"teams": []string{"engineering", "security"},
		"count": float64(2),
	}

	err := cm.SetToolResult("github-permissions", params, toolResult, 1*time.Hour)
	require.NoError(t, err)

	var loaded map[string]interface{}
	err = cm.GetToolResult("github-permissions", params, &loaded)
	require.NoError(t, err)
	assert.Equal(t, float64(2), loaded["count"])
}

func TestCache_GetToolResult_Miss(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	var result map[string]interface{}
	err := cm.GetToolResult("no-tool", map[string]interface{}{"x": "y"}, &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache miss")
}

func TestCache_GetToolResult_Expired(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	params := map[string]interface{}{"key": "val"}
	key := "tool-expired_" + cm.generateParamKey(params)
	cacheKey := cm.generateCacheKey(key)

	entry := &CacheEntry{
		Key:        key,
		Value:      "stale-data",
		Expiration: time.Now().Add(-1 * time.Hour),
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		AccessedAt: time.Now().Add(-2 * time.Hour),
	}
	require.NoError(t, cm.fileStorage.Save("tool_results", cacheKey, entry))

	var result string
	err := cm.GetToolResult("tool-expired", params, &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestCache_SetSummary_GetSummary(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	summary := map[string]interface{}{
		"total":   float64(10),
		"pending": float64(3),
	}

	err := cm.SetSummary("policy", "all", summary, 1*time.Hour)
	require.NoError(t, err)

	var loaded map[string]interface{}
	err = cm.GetSummary("policy", "all", &loaded)
	require.NoError(t, err)
	assert.Equal(t, float64(10), loaded["total"])
}

func TestCache_GetSummary_Miss(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	var result map[string]interface{}
	err := cm.GetSummary("nonexistent", "id", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cache miss")
}

func TestCache_GetSummary_Expired(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	key := "expired-summary_id1"
	cacheKey := cm.generateCacheKey(key)
	entry := &CacheEntry{
		Key:        key,
		Value:      "old",
		Expiration: time.Now().Add(-1 * time.Hour),
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		AccessedAt: time.Now().Add(-2 * time.Hour),
	}
	require.NoError(t, cm.fileStorage.Save("summaries", cacheKey, entry))

	var result string
	err := cm.GetSummary("expired-summary", "id1", &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestCache_SetToolResult_DefaultTTL(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	params := map[string]interface{}{"a": "b"}
	err := cm.SetToolResult("tool", params, "data", 0) // 0 => default 6h
	require.NoError(t, err)

	var result string
	err = cm.GetToolResult("tool", params, &result)
	require.NoError(t, err)
	assert.Equal(t, "data", result)
}

func TestCache_SetSummary_DefaultTTL(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	err := cm.SetSummary("type", "id", "summary-data", 0) // 0 => default 12h
	require.NoError(t, err)

	var result string
	err = cm.GetSummary("type", "id", &result)
	require.NoError(t, err)
	assert.Equal(t, "summary-data", result)
}

func TestCache_GetSize(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	// Initially empty (only subdirs)
	initialSize := cm.GetSize()

	require.NoError(t, cm.Set("bulk", "some data content here", 1*time.Hour))

	afterSize := cm.GetSize()
	assert.Greater(t, afterSize, initialSize)
}

func TestCache_GetStats(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	require.NoError(t, cm.Set("key1", "val1", 1*time.Hour))
	require.NoError(t, cm.SetToolResult("tool1", map[string]interface{}{"p": "v"}, "result", 1*time.Hour))
	require.NoError(t, cm.SetSummary("stype", "sid", "sval", 1*time.Hour))

	stats := cm.GetStats()
	assert.Contains(t, stats, "total_entries")
	assert.Contains(t, stats, "expired_entries")
	assert.Contains(t, stats, "cache_size_bytes")
	assert.Contains(t, stats, "max_cache_size_bytes")
	assert.Contains(t, stats, "hit_ratio")
	assert.Contains(t, stats, "generated_at")

	totalEntries, ok := stats["total_entries"].(int)
	assert.True(t, ok)
	assert.GreaterOrEqual(t, totalEntries, 3)
}

func TestCache_GenerateCacheKey_Deterministic(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	key1 := cm.generateCacheKey("same-input")
	key2 := cm.generateCacheKey("same-input")
	assert.Equal(t, key1, key2)

	key3 := cm.generateCacheKey("different-input")
	assert.NotEqual(t, key1, key3)
}

func TestCache_GenerateParamKey_Deterministic(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	params := map[string]interface{}{"a": "1", "b": "2"}
	key1 := cm.generateParamKey(params)
	key2 := cm.generateParamKey(params)
	assert.Equal(t, key1, key2)
}

func TestCache_CleanupExpiredEntries(t *testing.T) {
	t.Parallel()
	cm := newTestCacheManager(t)

	// Create an expired entry
	cacheKey := cm.generateCacheKey("cleanup-test")
	entry := &CacheEntry{
		Key:        "cleanup-test",
		Value:      "old",
		Expiration: time.Now().Add(-1 * time.Hour),
		CreatedAt:  time.Now().Add(-2 * time.Hour),
		AccessedAt: time.Now().Add(-2 * time.Hour),
	}
	require.NoError(t, cm.fileStorage.Save("general", cacheKey, entry))

	// Create a valid entry
	require.NoError(t, cm.Set("valid-entry", "still-good", 1*time.Hour))

	// Run cleanup
	cm.cleanupExpiredEntries()

	// Expired entry should be gone
	assert.False(t, cm.fileStorage.Exists("general", cacheKey))

	// Valid entry should remain
	var result string
	err := cm.Get("valid-entry", &result)
	require.NoError(t, err)
	assert.Equal(t, "still-good", result)
}
