// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"context"
	"testing"

	"github.com/grctool/grctool/internal/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// stubTool — minimal Tool implementation for registry tests
// ---------------------------------------------------------------------------

type stubTool struct {
	name        string
	description string
}

func (s *stubTool) Name() string        { return s.name }
func (s *stubTool) Description() string { return s.description }
func (s *stubTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        s.name,
		Description: s.description,
		InputSchema: map[string]interface{}{"type": "object"},
	}
}
func (s *stubTool) Execute(_ context.Context, _ map[string]interface{}) (string, *models.EvidenceSource, error) {
	return "stub result", nil, nil
}

// stubExtendedTool also implements ExtendedTool
type stubExtendedTool struct {
	stubTool
	version  string
	category string
}

func (s *stubExtendedTool) Version() string  { return s.version }
func (s *stubExtendedTool) Category() string { return s.category }

// ---------------------------------------------------------------------------
// Registry.Register
// ---------------------------------------------------------------------------

func TestRegistry_Register(t *testing.T) {
	t.Parallel()

	t.Run("register succeeds", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		err := r.Register(&stubTool{name: "alpha", description: "A"})
		assert.NoError(t, err)
		assert.Equal(t, 1, r.Count())
	})

	t.Run("duplicate registration fails", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		require.NoError(t, r.Register(&stubTool{name: "alpha", description: "A"}))

		err := r.Register(&stubTool{name: "alpha", description: "A2"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("empty name fails", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		err := r.Register(&stubTool{name: "", description: "blank"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "empty")
	})
}

// ---------------------------------------------------------------------------
// Registry.Unregister
// ---------------------------------------------------------------------------

func TestRegistry_Unregister(t *testing.T) {
	t.Parallel()

	t.Run("unregister existing tool", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		require.NoError(t, r.Register(&stubTool{name: "alpha"}))
		assert.Equal(t, 1, r.Count())

		err := r.Unregister("alpha")
		assert.NoError(t, err)
		assert.Equal(t, 0, r.Count())
	})

	t.Run("unregister missing tool fails", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		err := r.Unregister("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})
}

// ---------------------------------------------------------------------------
// Registry.Get
// ---------------------------------------------------------------------------

func TestRegistry_Get(t *testing.T) {
	t.Parallel()

	t.Run("get existing tool", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		require.NoError(t, r.Register(&stubTool{name: "alpha", description: "desc-a"}))

		tool, err := r.Get("alpha")
		require.NoError(t, err)
		assert.Equal(t, "alpha", tool.Name())
		assert.Equal(t, "desc-a", tool.Description())
	})

	t.Run("get missing tool returns error", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()

		_, err := r.Get("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})
}

// ---------------------------------------------------------------------------
// Registry.Exists
// ---------------------------------------------------------------------------

func TestRegistry_Exists(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	require.NoError(t, r.Register(&stubTool{name: "alpha"}))

	assert.True(t, r.Exists("alpha"))
	assert.False(t, r.Exists("beta"))
}

// ---------------------------------------------------------------------------
// Registry.Count
// ---------------------------------------------------------------------------

func TestRegistry_Count(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	assert.Equal(t, 0, r.Count())

	require.NoError(t, r.Register(&stubTool{name: "a"}))
	require.NoError(t, r.Register(&stubTool{name: "b"}))
	assert.Equal(t, 2, r.Count())
}

// ---------------------------------------------------------------------------
// Registry.List & ListNames — sorted output
// ---------------------------------------------------------------------------

func TestRegistry_List(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	require.NoError(t, r.Register(&stubTool{name: "beta", description: "B tool"}))
	require.NoError(t, r.Register(&stubTool{name: "alpha", description: "A tool"}))

	infos := r.List()
	require.Len(t, infos, 2)
	// Should be sorted alphabetically
	assert.Equal(t, "alpha", infos[0].Name)
	assert.Equal(t, "beta", infos[1].Name)
	assert.Equal(t, "A tool", infos[0].Description)
	assert.True(t, infos[0].Enabled)
}

func TestRegistry_List_ExtendedTool(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	require.NoError(t, r.Register(&stubExtendedTool{
		stubTool: stubTool{name: "ext-tool", description: "extended"},
		version:  "2.0.0",
		category: "security",
	}))

	infos := r.List()
	require.Len(t, infos, 1)
	assert.Equal(t, "2.0.0", infos[0].Version)
	assert.Equal(t, "security", infos[0].Category)
}

func TestRegistry_ListNames(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	require.NoError(t, r.Register(&stubTool{name: "charlie"}))
	require.NoError(t, r.Register(&stubTool{name: "alpha"}))
	require.NoError(t, r.Register(&stubTool{name: "bravo"}))

	names := r.ListNames()
	assert.Equal(t, []string{"alpha", "bravo", "charlie"}, names)
}

// ---------------------------------------------------------------------------
// Registry.Execute
// ---------------------------------------------------------------------------

func TestRegistry_Execute(t *testing.T) {
	t.Parallel()

	t.Run("execute existing tool", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()
		require.NoError(t, r.Register(&stubTool{name: "alpha"}))

		result, source, err := r.Execute(context.Background(), "alpha", nil)
		require.NoError(t, err)
		assert.Equal(t, "stub result", result)
		assert.Nil(t, source)
	})

	t.Run("execute missing tool returns error", func(t *testing.T) {
		t.Parallel()
		r := NewRegistry()

		_, _, err := r.Execute(context.Background(), "nonexistent", nil)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// Registry.GetClaudeToolDefinitions
// ---------------------------------------------------------------------------

func TestRegistry_GetClaudeToolDefinitions(t *testing.T) {
	t.Parallel()

	r := NewRegistry()
	require.NoError(t, r.Register(&stubTool{name: "beta", description: "B"}))
	require.NoError(t, r.Register(&stubTool{name: "alpha", description: "A"}))

	defs := r.GetClaudeToolDefinitions()
	require.Len(t, defs, 2)
	// Should be sorted by name
	assert.Equal(t, "alpha", defs[0].Name)
	assert.Equal(t, "beta", defs[1].Name)
}

// ---------------------------------------------------------------------------
// GetRegistryStats (uses GlobalRegistry - snapshot based test)
// ---------------------------------------------------------------------------

func TestGetRegistryStats(t *testing.T) {
	// This test uses the global registry, so it may already have tools
	// registered. We just verify the shape of the result.
	stats := GetRegistryStats()
	assert.Equal(t, "1", stats.SchemaVersion)
	assert.GreaterOrEqual(t, stats.TotalTools, 0)
	assert.NotNil(t, stats.ToolsByCategory)
	assert.NotNil(t, stats.ToolNames)
}
