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

package tools_test

import (
	"context"
	"testing"

	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTool implements the Tool interface for testing
type MockTool struct {
	name        string
	description string
	executeFunc func(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error)
}

func (m *MockTool) Name() string {
	return m.name
}

func (m *MockTool) Description() string {
	return m.description
}

func (m *MockTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        m.name,
		Description: m.description,
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"test": map[string]interface{}{
					"type":        "string",
					"description": "Test parameter",
				},
			},
			"required": []string{},
		},
	}
}

func (m *MockTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	if m.executeFunc != nil {
		return m.executeFunc(ctx, params)
	}
	return "mock result", &models.EvidenceSource{
		Type:     m.name,
		Resource: "mock-resource",
		Metadata: params,
	}, nil
}

// MockExtendedTool implements ExtendedTool for testing
type MockExtendedTool struct {
	MockTool
	version  string
	category string
}

func (m *MockExtendedTool) Version() string {
	return m.version
}

func (m *MockExtendedTool) Category() string {
	return m.category
}

// MockConfigurableTool implements ConfigurableTool for testing
type MockConfigurableTool struct {
	MockTool
	config map[string]interface{}
}

func (m *MockConfigurableTool) Configure(config map[string]interface{}) error {
	m.config = config
	return nil
}

func (m *MockConfigurableTool) GetConfiguration() map[string]interface{} {
	return m.config
}

func TestRegistry_Basic(t *testing.T) {
	registry := tools.NewRegistry()

	t.Run("Empty Registry", func(t *testing.T) {
		assert.Equal(t, 0, registry.Count())
		assert.False(t, registry.Exists("nonexistent"))
		assert.Empty(t, registry.List())
		assert.Empty(t, registry.ListNames())
	})

	t.Run("Register Tool", func(t *testing.T) {
		tool := &MockTool{
			name:        "test-tool",
			description: "Test tool description",
		}

		err := registry.Register(tool)
		require.NoError(t, err)

		assert.Equal(t, 1, registry.Count())
		assert.True(t, registry.Exists("test-tool"))

		retrievedTool, err := registry.Get("test-tool")
		require.NoError(t, err)
		assert.Equal(t, tool, retrievedTool)
	})

	t.Run("Register Duplicate Tool", func(t *testing.T) {
		tool1 := &MockTool{name: "duplicate", description: "First tool"}
		tool2 := &MockTool{name: "duplicate", description: "Second tool"}

		err := registry.Register(tool1)
		require.NoError(t, err)

		err = registry.Register(tool2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already registered")
	})

	t.Run("Register Tool with Empty Name", func(t *testing.T) {
		tool := &MockTool{name: "", description: "Tool with empty name"}

		err := registry.Register(tool)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "name cannot be empty")
	})
}

func TestRegistry_GetTool(t *testing.T) {
	registry := tools.NewRegistry()

	tool := &MockTool{
		name:        "existing-tool",
		description: "Existing tool",
	}
	err := registry.Register(tool)
	require.NoError(t, err)

	t.Run("Get Existing Tool", func(t *testing.T) {
		retrievedTool, err := registry.Get("existing-tool")
		require.NoError(t, err)
		assert.Equal(t, tool, retrievedTool)
	})

	t.Run("Get Nonexistent Tool", func(t *testing.T) {
		_, err := registry.Get("nonexistent-tool")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})
}

func TestRegistry_Unregister(t *testing.T) {
	registry := tools.NewRegistry()

	tool := &MockTool{
		name:        "tool-to-remove",
		description: "Tool to be removed",
	}
	err := registry.Register(tool)
	require.NoError(t, err)

	t.Run("Unregister Existing Tool", func(t *testing.T) {
		assert.True(t, registry.Exists("tool-to-remove"))

		err := registry.Unregister("tool-to-remove")
		require.NoError(t, err)

		assert.False(t, registry.Exists("tool-to-remove"))
		assert.Equal(t, 0, registry.Count())
	})

	t.Run("Unregister Nonexistent Tool", func(t *testing.T) {
		err := registry.Unregister("nonexistent-tool")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})
}

func TestRegistry_List(t *testing.T) {
	registry := tools.NewRegistry()

	tools := []*MockTool{
		{name: "tool-a", description: "Tool A"},
		{name: "tool-b", description: "Tool B"},
		{name: "tool-c", description: "Tool C"},
	}

	for _, tool := range tools {
		err := registry.Register(tool)
		require.NoError(t, err)
	}

	t.Run("List Tools", func(t *testing.T) {
		list := registry.List()
		assert.Len(t, list, 3)

		// Check that tools are sorted by name
		assert.Equal(t, "tool-a", list[0].Name)
		assert.Equal(t, "tool-b", list[1].Name)
		assert.Equal(t, "tool-c", list[2].Name)

		// Check tool info content
		for i, info := range list {
			assert.Equal(t, tools[i].name, info.Name)
			assert.Equal(t, tools[i].description, info.Description)
			assert.True(t, info.Enabled)
		}
	})

	t.Run("List Names", func(t *testing.T) {
		names := registry.ListNames()
		assert.Len(t, names, 3)
		assert.Equal(t, []string{"tool-a", "tool-b", "tool-c"}, names)
	})
}

func TestRegistry_ExtendedTool(t *testing.T) {
	registry := tools.NewRegistry()

	extendedTool := &MockExtendedTool{
		MockTool: MockTool{
			name:        "extended-tool",
			description: "Extended tool",
		},
		version:  "1.0.0",
		category: "testing",
	}

	err := registry.Register(extendedTool)
	require.NoError(t, err)

	t.Run("Extended Tool Metadata", func(t *testing.T) {
		list := registry.List()
		assert.Len(t, list, 1)

		info := list[0]
		assert.Equal(t, "extended-tool", info.Name)
		assert.Equal(t, "Extended tool", info.Description)
		assert.Equal(t, "1.0.0", info.Version)
		assert.Equal(t, "testing", info.Category)
		assert.True(t, info.Enabled)
	})
}

func TestRegistry_Execute(t *testing.T) {
	registry := tools.NewRegistry()

	var executedParams map[string]interface{}
	tool := &MockTool{
		name:        "executable-tool",
		description: "Tool that can be executed",
		executeFunc: func(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
			executedParams = params
			return "execution result", &models.EvidenceSource{
				Type:     "executable-tool",
				Resource: "execution-resource",
				Metadata: params,
			}, nil
		},
	}

	err := registry.Register(tool)
	require.NoError(t, err)

	t.Run("Execute Existing Tool", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{
			"param1": "value1",
			"param2": 42,
		}

		result, source, err := registry.Execute(ctx, "executable-tool", params)
		require.NoError(t, err)

		assert.Equal(t, "execution result", result)
		assert.NotNil(t, source)
		assert.Equal(t, "executable-tool", source.Type)
		assert.Equal(t, params, executedParams)
	})

	t.Run("Execute Nonexistent Tool", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{}

		_, _, err := registry.Execute(ctx, "nonexistent-tool", params)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not registered")
	})
}

func TestRegistry_ClaudeToolDefinitions(t *testing.T) {
	registry := tools.NewRegistry()

	tools := []*MockTool{
		{name: "tool-z", description: "Tool Z"},
		{name: "tool-a", description: "Tool A"},
		{name: "tool-m", description: "Tool M"},
	}

	for _, tool := range tools {
		err := registry.Register(tool)
		require.NoError(t, err)
	}

	t.Run("Get Claude Tool Definitions", func(t *testing.T) {
		definitions := registry.GetClaudeToolDefinitions()
		assert.Len(t, definitions, 3)

		// Check that definitions are sorted by name
		assert.Equal(t, "tool-a", definitions[0].Name)
		assert.Equal(t, "tool-m", definitions[1].Name)
		assert.Equal(t, "tool-z", definitions[2].Name)

		// Check content - definitions should be sorted by name
		expectedNames := []string{"tool-a", "tool-m", "tool-z"}
		expectedDescriptions := []string{"Tool A", "Tool M", "Tool Z"}

		for i, def := range definitions {
			assert.Equal(t, expectedNames[i], def.Name)
			assert.Equal(t, expectedDescriptions[i], def.Description)
			assert.NotNil(t, def.InputSchema)
		}
	})
}

func TestRegistry_GlobalFunctions(t *testing.T) {
	// Reset global registry for testing
	tools.GlobalRegistry = tools.NewRegistry()

	tool := &MockTool{
		name:        "global-tool",
		description: "Tool in global registry",
	}

	t.Run("Register and Get Global Tool", func(t *testing.T) {
		err := tools.RegisterTool(tool)
		require.NoError(t, err)

		retrievedTool, err := tools.GetTool("global-tool")
		require.NoError(t, err)
		assert.Equal(t, tool, retrievedTool)
	})

	t.Run("List Global Tools", func(t *testing.T) {
		list := tools.ListTools()
		assert.Len(t, list, 1)
		assert.Equal(t, "global-tool", list[0].Name)
	})

	t.Run("Execute Global Tool", func(t *testing.T) {
		ctx := context.Background()
		params := map[string]interface{}{"test": "value"}

		result, source, err := tools.ExecuteTool(ctx, "global-tool", params)
		require.NoError(t, err)
		assert.Equal(t, "mock result", result)
		assert.NotNil(t, source)
	})

	t.Run("Get All Claude Tool Definitions", func(t *testing.T) {
		definitions := tools.GetAllClaudeToolDefinitions()
		assert.Len(t, definitions, 1)
		assert.Equal(t, "global-tool", definitions[0].Name)
	})
}

func TestRegistry_Stats(t *testing.T) {
	// Reset global registry for testing
	tools.GlobalRegistry = tools.NewRegistry()

	// Register tools with categories
	extendedTool1 := &MockExtendedTool{
		MockTool: MockTool{name: "tool1", description: "Tool 1"},
		category: "security",
	}
	extendedTool2 := &MockExtendedTool{
		MockTool: MockTool{name: "tool2", description: "Tool 2"},
		category: "security",
	}
	extendedTool3 := &MockExtendedTool{
		MockTool: MockTool{name: "tool3", description: "Tool 3"},
		category: "infrastructure",
	}
	plainTool := &MockTool{name: "tool4", description: "Tool 4"}

	err := tools.RegisterTool(extendedTool1)
	require.NoError(t, err)
	err = tools.RegisterTool(extendedTool2)
	require.NoError(t, err)
	err = tools.RegisterTool(extendedTool3)
	require.NoError(t, err)
	err = tools.RegisterTool(plainTool)
	require.NoError(t, err)

	t.Run("Get Registry Stats", func(t *testing.T) {
		stats := tools.GetRegistryStats()

		assert.Equal(t, 4, stats.TotalTools)
		assert.Len(t, stats.ToolNames, 4)
		assert.Contains(t, stats.ToolNames, "tool1")
		assert.Contains(t, stats.ToolNames, "tool2")
		assert.Contains(t, stats.ToolNames, "tool3")
		assert.Contains(t, stats.ToolNames, "tool4")

		// Check category counts
		assert.Equal(t, 2, stats.ToolsByCategory["security"])
		assert.Equal(t, 1, stats.ToolsByCategory["infrastructure"])
		assert.Equal(t, 1, stats.ToolsByCategory["uncategorized"])
	})
}

func TestConfigurableTool(t *testing.T) {
	configurableTool := &MockConfigurableTool{
		MockTool: MockTool{
			name:        "configurable-tool",
			description: "Tool that can be configured",
		},
	}

	config := map[string]interface{}{
		"setting1": "value1",
		"setting2": 42,
	}

	t.Run("Configure Tool", func(t *testing.T) {
		err := configurableTool.Configure(config)
		require.NoError(t, err)

		retrievedConfig := configurableTool.GetConfiguration()
		assert.Equal(t, config, retrievedConfig)
	})
}
