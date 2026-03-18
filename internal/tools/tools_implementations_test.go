// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// OSFileReader
// ---------------------------------------------------------------------------

func TestOSFileReader_Open(t *testing.T) {
	t.Parallel()

	reader := NewOSFileReader()
	dir := t.TempDir()
	filePath := filepath.Join(dir, "test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("hello"), 0o644))

	t.Run("open existing file", func(t *testing.T) {
		t.Parallel()
		rc, err := reader.Open(filePath)
		require.NoError(t, err)
		defer rc.Close()
	})

	t.Run("open nonexistent file", func(t *testing.T) {
		t.Parallel()
		_, err := reader.Open(filepath.Join(dir, "nonexistent.txt"))
		assert.Error(t, err)
	})
}

func TestOSFileReader_Walk(t *testing.T) {
	t.Parallel()

	reader := NewOSFileReader()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "a.txt"), []byte("a"), 0o644))
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "sub"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "sub", "b.txt"), []byte("b"), 0o644))

	var files []string
	err := reader.Walk(dir, func(path string, info FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if info != nil && !info.IsDir() {
			files = append(files, info.Name())
		}
		return nil
	})
	require.NoError(t, err)
	assert.Len(t, files, 2)
	assert.Contains(t, files, "a.txt")
	assert.Contains(t, files, "b.txt")
}

func TestOSFileReader_Glob(t *testing.T) {
	t.Parallel()

	reader := NewOSFileReader()
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.go"), []byte("package main"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "test.txt"), []byte("text"), 0o644))

	matches, err := reader.Glob(filepath.Join(dir, "*.go"))
	require.NoError(t, err)
	assert.Len(t, matches, 1)
}

// ---------------------------------------------------------------------------
// OSFileInfo
// ---------------------------------------------------------------------------

func TestOSFileInfo(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	filePath := filepath.Join(dir, "info_test.txt")
	require.NoError(t, os.WriteFile(filePath, []byte("data"), 0o644))

	info, err := os.Stat(filePath)
	require.NoError(t, err)

	osInfo := &OSFileInfo{info: info}
	assert.Equal(t, "info_test.txt", osInfo.Name())
	assert.Equal(t, int64(4), osInfo.Size())
	assert.False(t, osInfo.IsDir())
	assert.NotNil(t, osInfo.ModTime())
}

func TestOSFileInfo_Directory(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	info, err := os.Stat(dir)
	require.NoError(t, err)

	osInfo := &OSFileInfo{info: info}
	assert.True(t, osInfo.IsDir())
}

// ---------------------------------------------------------------------------
// DefaultTerraformFileAnalyzer.MatchesPatterns
// ---------------------------------------------------------------------------

func TestTerraformFileAnalyzer_MatchesPatterns(t *testing.T) {
	t.Parallel()

	analyzer := NewTerraformFileAnalyzer()

	t.Run("matches by basename", func(t *testing.T) {
		t.Parallel()
		assert.True(t, analyzer.MatchesPatterns("main.tf", []string{"*.tf"}))
	})

	t.Run("no match", func(t *testing.T) {
		t.Parallel()
		assert.False(t, analyzer.MatchesPatterns("readme.md", []string{"*.tf"}))
	})

	t.Run("multiple patterns", func(t *testing.T) {
		t.Parallel()
		assert.True(t, analyzer.MatchesPatterns("config.yaml", []string{"*.tf", "*.yaml"}))
	})

	t.Run("empty patterns", func(t *testing.T) {
		t.Parallel()
		assert.False(t, analyzer.MatchesPatterns("main.tf", nil))
	})
}

// ---------------------------------------------------------------------------
// contains helper
// ---------------------------------------------------------------------------

func TestContainsHelper(t *testing.T) {
	t.Parallel()

	assert.True(t, contains("hello world", "world"))
	assert.True(t, contains("path/to/file", "/"))
	assert.False(t, contains("hello", "world"))
	assert.True(t, contains("", ""))
}
