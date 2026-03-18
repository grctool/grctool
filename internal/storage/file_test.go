// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package storage

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestFileStorage(t *testing.T) *FileStorage {
	t.Helper()
	fs, err := NewFileStorage(t.TempDir())
	require.NoError(t, err)
	return fs
}

func TestNewFileStorage(t *testing.T) {
	t.Parallel()

	t.Run("creates base directory", func(t *testing.T) {
		t.Parallel()
		dir := filepath.Join(t.TempDir(), "nested", "dir")
		fs, err := NewFileStorage(dir)
		require.NoError(t, err)
		assert.NotNil(t, fs)
		assert.DirExists(t, dir)
	})
}

func TestFileStorage_SaveAndLoad(t *testing.T) {
	t.Parallel()

	type payload struct {
		Name  string `json:"name"`
		Value int    `json:"value"`
	}

	fs := newTestFileStorage(t)

	original := payload{Name: "test-item", Value: 42}
	err := fs.Save("widgets", "w1", &original)
	require.NoError(t, err)

	var loaded payload
	err = fs.Load("widgets", "w1", &loaded)
	require.NoError(t, err)
	assert.Equal(t, original, loaded)
}

func TestFileStorage_SaveCreatesCollectionDir(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	err := fs.Save("new_collection", "item1", map[string]string{"a": "b"})
	require.NoError(t, err)
	assert.DirExists(t, filepath.Join(fs.baseDir, "new_collection"))
}

func TestFileStorage_LoadNonExistent(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	var target map[string]string
	err := fs.Load("nonexistent", "nope", &target)
	assert.Error(t, err)
}

func TestFileStorage_List(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	// Empty collection returns empty slice
	ids, err := fs.List("empty_collection")
	require.NoError(t, err)
	assert.Empty(t, ids)

	// Save some items
	require.NoError(t, fs.Save("items", "alpha", "a"))
	require.NoError(t, fs.Save("items", "beta", "b"))
	require.NoError(t, fs.Save("items", "gamma", "c"))

	ids, err = fs.List("items")
	require.NoError(t, err)
	assert.Len(t, ids, 3)
	assert.Contains(t, ids, "alpha")
	assert.Contains(t, ids, "beta")
	assert.Contains(t, ids, "gamma")
}

func TestFileStorage_ListIgnoresNonJSON(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	// Create collection dir with a non-JSON file
	collDir := filepath.Join(fs.baseDir, "mixed")
	require.NoError(t, os.MkdirAll(collDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(collDir, "readme.txt"), []byte("hi"), 0644))
	require.NoError(t, fs.Save("mixed", "real", "data"))

	ids, err := fs.List("mixed")
	require.NoError(t, err)
	assert.Len(t, ids, 1)
	assert.Equal(t, "real", ids[0])
}

func TestFileStorage_Delete(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	require.NoError(t, fs.Save("coll", "item1", "data"))
	assert.True(t, fs.Exists("coll", "item1"))

	err := fs.Delete("coll", "item1")
	require.NoError(t, err)
	assert.False(t, fs.Exists("coll", "item1"))

	// Delete non-existent should not error
	err = fs.Delete("coll", "item1")
	assert.NoError(t, err)
}

func TestFileStorage_Exists(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	assert.False(t, fs.Exists("coll", "missing"))

	require.NoError(t, fs.Save("coll", "present", "yes"))
	assert.True(t, fs.Exists("coll", "present"))
}

func TestFileStorage_ClearCollection(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	require.NoError(t, fs.Save("clearme", "a", "1"))
	require.NoError(t, fs.Save("clearme", "b", "2"))
	require.NoError(t, fs.Save("keepme", "c", "3"))

	err := fs.Clear("clearme")
	require.NoError(t, err)

	ids, err := fs.List("clearme")
	require.NoError(t, err)
	assert.Empty(t, ids)

	// Other collection unaffected
	ids, err = fs.List("keepme")
	require.NoError(t, err)
	assert.Len(t, ids, 1)
}

func TestFileStorage_ClearNonExistentCollection(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	err := fs.Clear("doesnotexist")
	assert.NoError(t, err)
}

func TestFileStorage_ClearAll(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	require.NoError(t, fs.Save("a", "x", "1"))
	require.NoError(t, fs.Save("b", "y", "2"))

	err := fs.ClearAll()
	require.NoError(t, err)

	// Base dir should be removed
	_, err = os.Stat(fs.baseDir)
	assert.True(t, os.IsNotExist(err))
}

func TestFileStorage_GetStats(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	require.NoError(t, fs.Save("policies", "p1", "data"))
	require.NoError(t, fs.Save("policies", "p2", "data"))
	require.NoError(t, fs.Save("controls", "c1", "data"))

	stats := fs.GetStats()
	assert.Equal(t, fs.baseDir, stats["base_dir"])
	assert.Equal(t, 2, stats["policies_count"])
	assert.Equal(t, 1, stats["controls_count"])
}

func TestFileStorage_CreateDirectory(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	newDir := filepath.Join(fs.baseDir, "sub", "nested")
	err := fs.CreateDirectory(newDir)
	require.NoError(t, err)
	assert.DirExists(t, newDir)
}

func TestFileStorage_DeleteDirectory(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	dir := filepath.Join(fs.baseDir, "todelete")
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "f.txt"), []byte("hi"), 0644))

	err := fs.DeleteDirectory(dir)
	require.NoError(t, err)
	_, statErr := os.Stat(dir)
	assert.True(t, os.IsNotExist(statErr))
}

func TestFileStorage_ListDirectories(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	require.NoError(t, os.MkdirAll(filepath.Join(fs.baseDir, "dir_a"), 0755))
	require.NoError(t, os.MkdirAll(filepath.Join(fs.baseDir, "dir_b"), 0755))
	require.NoError(t, os.WriteFile(filepath.Join(fs.baseDir, "file.txt"), []byte("x"), 0644))

	dirs, err := fs.ListDirectories(fs.baseDir)
	require.NoError(t, err)
	assert.Len(t, dirs, 2)
	assert.Contains(t, dirs, "dir_a")
	assert.Contains(t, dirs, "dir_b")
}

func TestFileStorage_ListDirectories_NonExistentPath(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	_, err := fs.ListDirectories(filepath.Join(fs.baseDir, "nope"))
	assert.Error(t, err)
}

func TestFileStorage_GetFullPath(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	path := fs.GetFullPath("policies", "POL-0001")
	expected := filepath.Join(fs.baseDir, "policies", "POL-0001.json")
	assert.Equal(t, expected, path)
}

func TestFileStorage_GetSize(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	// Empty storage
	size, err := fs.GetSize()
	require.NoError(t, err)
	assert.Equal(t, int64(0), size)

	// Add some data
	require.NoError(t, fs.Save("data", "item", map[string]string{"key": "value"}))

	size, err = fs.GetSize()
	require.NoError(t, err)
	assert.Greater(t, size, int64(0))
}

func TestFileStorage_LoadAll_ReturnsError(t *testing.T) {
	t.Parallel()
	fs := newTestFileStorage(t)

	var target []string
	err := fs.LoadAll("anything", &target)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not implemented")
}
