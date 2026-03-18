// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// validateAndCleanPath (package-level helper)
// ---------------------------------------------------------------------------

func TestValidateAndCleanPath(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()

	t.Run("relative path joins with data dir", func(t *testing.T) {
		t.Parallel()
		cleanPath, err := validateAndCleanPath("docs/file.json", dataDir)
		require.NoError(t, err)
		absDataDir, _ := filepath.Abs(dataDir)
		assert.True(t, filepath.IsAbs(cleanPath))
		assert.Equal(t, filepath.Join(absDataDir, "docs", "file.json"), cleanPath)
	})

	t.Run("absolute path within data dir is allowed", func(t *testing.T) {
		t.Parallel()
		absDataDir, _ := filepath.Abs(dataDir)
		target := filepath.Join(absDataDir, "sub", "file.txt")
		cleanPath, err := validateAndCleanPath(target, dataDir)
		require.NoError(t, err)
		assert.Equal(t, target, cleanPath)
	})

	t.Run("path traversal blocked", func(t *testing.T) {
		t.Parallel()
		_, err := validateAndCleanPath("../../etc/passwd", dataDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "traversal")
	})

	t.Run("absolute path outside data dir blocked", func(t *testing.T) {
		t.Parallel()
		_, err := validateAndCleanPath("/tmp/outside", dataDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "outside data directory")
	})
}

// ---------------------------------------------------------------------------
// detectFormat (package-level helper)
// ---------------------------------------------------------------------------

func TestDetectFormat(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		filePath string
		content  string
		want     string
	}{
		"json by extension":       {filePath: "data.json", content: "", want: "json"},
		"yaml by extension .yaml": {filePath: "data.yaml", content: "", want: "yaml"},
		"yaml by extension .yml":  {filePath: "data.yml", content: "", want: "yaml"},
		"markdown by extension":   {filePath: "readme.md", content: "", want: "markdown"},
		"markdown alt ext":        {filePath: "readme.markdown", content: "", want: "markdown"},
		"text by extension":       {filePath: "notes.txt", content: "", want: "text"},
		"json by content object":  {filePath: "data.dat", content: `{"key": "value"}`, want: "json"},
		"json by content array":   {filePath: "data.dat", content: `[1, 2, 3]`, want: "json"},
		"yaml by content":         {filePath: "data.dat", content: "---\nkey: value\n- item", want: "yaml"},
		"yaml by content list":    {filePath: "data.dat", content: "items:\n- first\n- second", want: "yaml"},
		"markdown by content #":   {filePath: "data.dat", content: "# Title\nSome text", want: "markdown"},
		"markdown by content **":  {filePath: "data.dat", content: "Some **bold** text", want: "markdown"},
		"text fallback":           {filePath: "data.dat", content: "just plain text", want: "text"},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			got := detectFormat(tc.filePath, tc.content)
			assert.Equal(t, tc.want, got)
		})
	}
}

// ---------------------------------------------------------------------------
// StorageReadTool — metadata and Execute
// ---------------------------------------------------------------------------

func TestStorageReadTool_Metadata(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t.TempDir())
	log := testhelpers.NewStubLogger()
	srt := NewStorageReadTool(cfg, log)

	assert.Equal(t, "storage-read", srt.Name())
	assert.NotEmpty(t, srt.Description())
	assert.Equal(t, "1.0.0", srt.Version())
	assert.Equal(t, "storage", srt.Category())

	def := srt.GetClaudeToolDefinition()
	assert.Equal(t, "storage-read", def.Name)
}

func TestStorageReadTool_Execute(t *testing.T) {
	t.Parallel()

	dataDir := t.TempDir()
	cfg := newTestConfig(dataDir)
	log := testhelpers.NewStubLogger()
	srt := NewStorageReadTool(cfg, log)

	t.Run("read JSON file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageReadTool(cfg, log)

		// Create a JSON file
		filePath := filepath.Join(dir, "test.json")
		require.NoError(t, os.WriteFile(filePath, []byte(`{"name":"test","value":42}`), 0o644))

		result, source, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "test.json",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "test")
		assert.NotNil(t, source)
		assert.Equal(t, "storage-read", source.Type)
	})

	t.Run("read markdown file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageReadTool(cfg, log)

		filePath := filepath.Join(dir, "readme.md")
		require.NoError(t, os.WriteFile(filePath, []byte("# Hello\nWorld"), 0o644))

		result, _, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "readme.md",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "Hello")
	})

	t.Run("read with metadata", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageReadTool(cfg, log)

		filePath := filepath.Join(dir, "file.txt")
		require.NoError(t, os.WriteFile(filePath, []byte("content"), 0o644))

		result, _, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":          "file.txt",
			"with_metadata": true,
		})
		require.NoError(t, err)
		assert.Contains(t, result, "metadata")
	})

	t.Run("missing path parameter", func(t *testing.T) {
		t.Parallel()
		_, _, err := srt.Execute(context.Background(), map[string]interface{}{})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path parameter is required")
	})

	t.Run("nonexistent file", func(t *testing.T) {
		t.Parallel()
		_, _, err := srt.Execute(context.Background(), map[string]interface{}{
			"path": "nonexistent.txt",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not exist")
	})

	t.Run("path traversal", func(t *testing.T) {
		t.Parallel()
		_, _, err := srt.Execute(context.Background(), map[string]interface{}{
			"path": "../../etc/passwd",
		})
		assert.Error(t, err)
	})

	t.Run("forced format override", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageReadTool(cfg, log)

		filePath := filepath.Join(dir, "data.dat")
		require.NoError(t, os.WriteFile(filePath, []byte("plain text"), 0o644))

		result, source, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":   "data.dat",
			"format": "text",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "plain text")
		assert.NotNil(t, source)
	})

	t.Run("read YAML file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageReadTool(cfg, log)

		filePath := filepath.Join(dir, "config.yaml")
		require.NoError(t, os.WriteFile(filePath, []byte("key: value\nlist:\n  - item1"), 0o644))

		result, _, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "config.yaml",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "key")
	})
}

// ---------------------------------------------------------------------------
// StorageWriteTool — metadata and Execute
// ---------------------------------------------------------------------------

func TestStorageWriteTool_Metadata(t *testing.T) {
	t.Parallel()

	cfg := newTestConfig(t.TempDir())
	log := testhelpers.NewStubLogger()
	swt := NewStorageWriteTool(cfg, log)

	assert.Equal(t, "storage-write", swt.Name())
	assert.NotEmpty(t, swt.Description())
	assert.Equal(t, "1.0.0", swt.Version())
	assert.Equal(t, "storage", swt.Category())

	def := swt.GetClaudeToolDefinition()
	assert.Equal(t, "storage-write", def.Name)
}

func TestStorageWriteTool_Execute(t *testing.T) {
	t.Parallel()

	t.Run("write text file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageWriteTool(cfg, log)

		result, source, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":    "output.txt",
			"content": "hello world",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "success")
		assert.NotNil(t, source)

		// Verify file was written
		absPath := filepath.Join(dir, "output.txt")
		data, err := os.ReadFile(absPath)
		require.NoError(t, err)
		assert.Equal(t, "hello world", string(data))
	})

	t.Run("write JSON file with formatting", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageWriteTool(cfg, log)

		result, _, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":    "data.json",
			"content": `{"key":"value"}`,
		})
		require.NoError(t, err)
		assert.Contains(t, result, "success")

		// Verify JSON was pretty-printed
		absPath := filepath.Join(dir, "data.json")
		data, err := os.ReadFile(absPath)
		require.NoError(t, err)
		assert.Contains(t, string(data), "\n") // pretty-printed
	})

	t.Run("write to nested directory with create_dirs", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageWriteTool(cfg, log)

		_, _, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":        "deep/nested/dir/file.txt",
			"content":     "nested content",
			"create_dirs": true,
		})
		require.NoError(t, err)

		absPath := filepath.Join(dir, "deep", "nested", "dir", "file.txt")
		data, err := os.ReadFile(absPath)
		require.NoError(t, err)
		assert.Equal(t, "nested content", string(data))
	})

	t.Run("overwrite existing file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageWriteTool(cfg, log)

		absPath := filepath.Join(dir, "existing.txt")
		require.NoError(t, os.WriteFile(absPath, []byte("original"), 0o644))

		result, _, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":    "existing.txt",
			"content": "updated",
		})
		require.NoError(t, err)
		assert.Contains(t, result, "existed_before")

		data, err := os.ReadFile(absPath)
		require.NoError(t, err)
		assert.Equal(t, "updated", string(data))
	})

	t.Run("missing path parameter", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageWriteTool(cfg, log)

		_, _, err := tool.Execute(context.Background(), map[string]interface{}{
			"content": "data",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "path parameter is required")
	})

	t.Run("missing content parameter", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageWriteTool(cfg, log)

		_, _, err := tool.Execute(context.Background(), map[string]interface{}{
			"path": "file.txt",
		})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "content parameter is required")
	})

	t.Run("path traversal blocked", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageWriteTool(cfg, log)

		_, _, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":    "../../etc/pwned",
			"content": "evil",
		})
		assert.Error(t, err)
	})

	t.Run("write YAML file", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageWriteTool(cfg, log)

		_, _, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":    "config.yaml",
			"content": "key: value\nlist:\n  - item1",
		})
		require.NoError(t, err)
	})

	t.Run("write markdown file trims trailing whitespace", func(t *testing.T) {
		t.Parallel()
		dir := t.TempDir()
		cfg := newTestConfig(dir)
		log := testhelpers.NewStubLogger()
		tool := NewStorageWriteTool(cfg, log)

		_, _, err := tool.Execute(context.Background(), map[string]interface{}{
			"path":    "doc.md",
			"content": "# Title   \nContent  \t",
		})
		require.NoError(t, err)

		absPath := filepath.Join(dir, "doc.md")
		data, err := os.ReadFile(absPath)
		require.NoError(t, err)
		assert.Equal(t, "# Title\nContent", string(data))
	})
}

// ---------------------------------------------------------------------------
// common_types utility functions
// ---------------------------------------------------------------------------

func TestMinFloat(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 1.5, minFloat(1.5, 2.5))
	assert.Equal(t, 1.5, minFloat(2.5, 1.5))
	assert.Equal(t, 0.0, minFloat(0.0, 1.0))
}

func TestMaxFloat(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 2.5, maxFloat(1.5, 2.5))
	assert.Equal(t, 2.5, maxFloat(2.5, 1.5))
	assert.Equal(t, 1.0, maxFloat(0.0, 1.0))
}

func TestMax(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 100.0, max(100.0, 50.0))
	assert.Equal(t, 100.0, max(50.0, 100.0))
	assert.Equal(t, -1.0, max(-1.0, -5.0))
}

func TestMin(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 5, min(5, 10))
	assert.Equal(t, 5, min(10, 5))
	assert.Equal(t, 0, min(0, 1))
}
