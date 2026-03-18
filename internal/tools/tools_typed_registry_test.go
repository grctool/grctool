// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"testing"

	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Global registry wrapper functions (RegisterTool, GetTool, ExecuteTool, etc.)
// ---------------------------------------------------------------------------

func TestGlobalRegistryFunctions(t *testing.T) {
	// These tests swap the GlobalRegistry to avoid contaminating other tests.

	t.Run("RegisterTool and GetTool", func(t *testing.T) {
		original := GlobalRegistry
		GlobalRegistry = NewRegistry()
		defer func() { GlobalRegistry = original }()

		tool := &stubTool{name: "global-test-tool", description: "test"}
		err := RegisterTool(tool)
		require.NoError(t, err)

		got, err := GetTool("global-test-tool")
		require.NoError(t, err)
		assert.Equal(t, "global-test-tool", got.Name())
	})

	t.Run("GetTool not found", func(t *testing.T) {
		original := GlobalRegistry
		GlobalRegistry = NewRegistry()
		defer func() { GlobalRegistry = original }()

		_, err := GetTool("nonexistent")
		assert.Error(t, err)
	})

	t.Run("ExecuteTool", func(t *testing.T) {
		original := GlobalRegistry
		GlobalRegistry = NewRegistry()
		defer func() { GlobalRegistry = original }()

		tool := &stubTool{name: "exec-test", description: "test"}
		require.NoError(t, RegisterTool(tool))

		result, source, err := ExecuteTool(context.Background(), "exec-test", nil)
		require.NoError(t, err)
		assert.Equal(t, "stub result", result)
		assert.Nil(t, source)
	})

	t.Run("ExecuteTool not found", func(t *testing.T) {
		original := GlobalRegistry
		GlobalRegistry = NewRegistry()
		defer func() { GlobalRegistry = original }()

		_, _, err := ExecuteTool(context.Background(), "nonexistent", nil)
		assert.Error(t, err)
	})

	t.Run("GetAllClaudeToolDefinitions", func(t *testing.T) {
		original := GlobalRegistry
		GlobalRegistry = NewRegistry()
		defer func() { GlobalRegistry = original }()

		require.NoError(t, RegisterTool(&stubTool{name: "tool-a", description: "A"}))
		require.NoError(t, RegisterTool(&stubTool{name: "tool-b", description: "B"}))

		defs := GetAllClaudeToolDefinitions()
		require.Len(t, defs, 2)
		assert.Equal(t, "tool-a", defs[0].Name)
		assert.Equal(t, "tool-b", defs[1].Name)
	})

	t.Run("GetRegistryStats with categorized tool", func(t *testing.T) {
		original := GlobalRegistry
		GlobalRegistry = NewRegistry()
		defer func() { GlobalRegistry = original }()

		require.NoError(t, RegisterTool(&stubExtendedTool{
			stubTool: stubTool{name: "cat-tool", description: "A"},
			version:  "1.0",
			category: "security",
		}))

		stats := GetRegistryStats()
		assert.Equal(t, 1, stats.TotalTools)
		assert.Contains(t, stats.ToolNames, "cat-tool")
		assert.Equal(t, 1, stats.ToolsByCategory["security"])
	})

	t.Run("GetRegistryStats uncategorized tool", func(t *testing.T) {
		original := GlobalRegistry
		GlobalRegistry = NewRegistry()
		defer func() { GlobalRegistry = original }()

		require.NoError(t, RegisterTool(&stubTool{name: "plain", description: "plain"}))

		stats := GetRegistryStats()
		assert.Equal(t, 1, stats.TotalTools)
		assert.Equal(t, 1, stats.ToolsByCategory["uncategorized"])
	})
}

// ---------------------------------------------------------------------------
// TypedToolRegistry — operations with proper types
// ---------------------------------------------------------------------------

func TestTypedToolRegistry_LegacyOperations(t *testing.T) {
	t.Parallel()

	t.Run("RegisterLegacyTool and GetLegacyTool", func(t *testing.T) {
		t.Parallel()
		reg := &TypedToolRegistry{
			legacyTools: make(map[string]Tool),
			typedTools:  make(map[string]types.TypedTool),
			adapters:    make(map[string]*types.ToolAdapter),
		}
		tool := &stubTool{name: "legacy-tool", description: "legacy"}
		reg.legacyTools["legacy-tool"] = tool

		got, exists := reg.GetLegacyTool("legacy-tool")
		assert.True(t, exists)
		assert.Equal(t, "legacy-tool", got.Name())

		_, exists = reg.GetLegacyTool("nonexistent")
		assert.False(t, exists)
	})

	t.Run("ListLegacyTools", func(t *testing.T) {
		t.Parallel()
		reg := &TypedToolRegistry{
			legacyTools: make(map[string]Tool),
			typedTools:  make(map[string]types.TypedTool),
			adapters:    make(map[string]*types.ToolAdapter),
		}
		reg.legacyTools["a"] = &stubTool{name: "a"}
		reg.legacyTools["b"] = &stubTool{name: "b"}

		list := reg.ListLegacyTools()
		assert.Len(t, list, 2)
		assert.Contains(t, list, "a")
		assert.Contains(t, list, "b")
	})

	t.Run("ListTypedTools empty", func(t *testing.T) {
		t.Parallel()
		reg := &TypedToolRegistry{
			legacyTools: make(map[string]Tool),
			typedTools:  make(map[string]types.TypedTool),
			adapters:    make(map[string]*types.ToolAdapter),
		}

		list := reg.ListTypedTools()
		assert.Empty(t, list)
	})

	t.Run("HasTypedVersion false for legacy only", func(t *testing.T) {
		t.Parallel()
		reg := &TypedToolRegistry{
			legacyTools: make(map[string]Tool),
			typedTools:  make(map[string]types.TypedTool),
			adapters:    make(map[string]*types.ToolAdapter),
		}
		reg.legacyTools["x"] = &stubTool{name: "x"}

		assert.False(t, reg.HasTypedVersion("x"))
		assert.False(t, reg.HasTypedVersion("nonexistent"))
	})

	t.Run("GetTypedTool not found", func(t *testing.T) {
		t.Parallel()
		reg := &TypedToolRegistry{
			legacyTools: make(map[string]Tool),
			typedTools:  make(map[string]types.TypedTool),
			adapters:    make(map[string]*types.ToolAdapter),
		}

		_, exists := reg.GetTypedTool("nonexistent")
		assert.False(t, exists)
	})
}

// ---------------------------------------------------------------------------
// stubToolWithExec returns configurable results from Execute
// ---------------------------------------------------------------------------

type stubToolWithExec struct {
	name   string
	result string
	err    error
	source *models.EvidenceSource
}

func (s *stubToolWithExec) Name() string        { return s.name }
func (s *stubToolWithExec) Description() string  { return "stub with exec" }
func (s *stubToolWithExec) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{Name: s.name, Description: "stub"}
}
func (s *stubToolWithExec) Execute(_ context.Context, _ map[string]interface{}) (string, *models.EvidenceSource, error) {
	return s.result, s.source, s.err
}

func TestRegistry_Execute_WithSource(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	source := &models.EvidenceSource{Type: "test", Resource: "test-resource"}
	require.NoError(t, r.Register(&stubToolWithExec{
		name:   "sourced-tool",
		result: "result with source",
		source: source,
	}))

	result, gotSource, err := r.Execute(context.Background(), "sourced-tool", nil)
	require.NoError(t, err)
	assert.Equal(t, "result with source", result)
	require.NotNil(t, gotSource)
	assert.Equal(t, "test", gotSource.Type)
}
