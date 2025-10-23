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

package tools

import (
	"context"
	"fmt"
	"sort"
	"sync"

	"github.com/grctool/grctool/internal/models"
)

// Registry manages the registration and discovery of evidence collection tools
type Registry struct {
	tools map[string]Tool
	mutex sync.RWMutex
}

// ToolInfo is imported from models package to avoid circular dependencies
type ToolInfo = models.ToolInfo

// GlobalRegistry is the singleton instance of the tool registry
var GlobalRegistry *Registry

// init initializes the global registry
func init() {
	GlobalRegistry = NewRegistry()
}

// NewRegistry creates a new tool registry
func NewRegistry() *Registry {
	return &Registry{
		tools: make(map[string]Tool),
	}
}

// Register adds a tool to the registry
func (r *Registry) Register(tool Tool) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	name := tool.Name()
	if name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}

	// Check for duplicate registration
	if _, exists := r.tools[name]; exists {
		return fmt.Errorf("tool '%s' is already registered", name)
	}

	r.tools[name] = tool
	return nil
}

// Unregister removes a tool from the registry
func (r *Registry) Unregister(name string) error {
	r.mutex.Lock()
	defer r.mutex.Unlock()

	if _, exists := r.tools[name]; !exists {
		return fmt.Errorf("tool '%s' is not registered", name)
	}

	delete(r.tools, name)
	return nil
}

// Get retrieves a tool by name
func (r *Registry) Get(name string) (Tool, error) {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	tool, exists := r.tools[name]
	if !exists {
		return nil, fmt.Errorf("tool '%s' is not registered", name)
	}

	return tool, nil
}

// List returns information about all registered tools
func (r *Registry) List() []ToolInfo {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	infos := make([]ToolInfo, 0, len(r.tools))
	for name, tool := range r.tools {
		info := ToolInfo{
			Name:        name,
			Description: tool.Description(),
			Enabled:     true, // For now, all registered tools are considered enabled
		}

		// Try to get additional metadata if the tool implements extended interfaces
		if extended, ok := tool.(ExtendedTool); ok {
			info.Version = extended.Version()
			info.Category = extended.Category()
		}

		infos = append(infos, info)
	}

	// Sort by name for consistent output
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].Name < infos[j].Name
	})

	return infos
}

// ListNames returns the names of all registered tools
func (r *Registry) ListNames() []string {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	names := make([]string, 0, len(r.tools))
	for name := range r.tools {
		names = append(names, name)
	}

	sort.Strings(names)
	return names
}

// Exists checks if a tool is registered
func (r *Registry) Exists(name string) bool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	_, exists := r.tools[name]
	return exists
}

// Count returns the number of registered tools
func (r *Registry) Count() int {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	return len(r.tools)
}

// Execute runs a tool with the given parameters
func (r *Registry) Execute(ctx context.Context, toolName string, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	tool, err := r.Get(toolName)
	if err != nil {
		return "", nil, err
	}

	return tool.Execute(ctx, params)
}

// GetClaudeToolDefinitions returns Claude tool definitions for all registered tools
func (r *Registry) GetClaudeToolDefinitions() []models.ClaudeTool {
	r.mutex.RLock()
	defer r.mutex.RUnlock()

	definitions := make([]models.ClaudeTool, 0, len(r.tools))
	for _, tool := range r.tools {
		definitions = append(definitions, tool.GetClaudeToolDefinition())
	}

	// Sort by name for consistent output
	sort.Slice(definitions, func(i, j int) bool {
		return definitions[i].Name < definitions[j].Name
	})

	return definitions
}

// ExtendedTool is an optional interface for tools that provide additional metadata
type ExtendedTool interface {
	Tool
	Version() string
	Category() string
}

// ConfigurableTool is an optional interface for tools that can be configured
type ConfigurableTool interface {
	Tool
	Configure(config map[string]interface{}) error
	GetConfiguration() map[string]interface{}
}

// RegisterTool registers a tool in the global registry
func RegisterTool(tool Tool) error {
	return GlobalRegistry.Register(tool)
}

// GetTool retrieves a tool from the global registry
func GetTool(name string) (Tool, error) {
	return GlobalRegistry.Get(name)
}

// ListTools returns information about all tools in the global registry
func ListTools() []ToolInfo {
	return GlobalRegistry.List()
}

// ExecuteTool runs a tool from the global registry
func ExecuteTool(ctx context.Context, toolName string, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return GlobalRegistry.Execute(ctx, toolName, params)
}

// GetAllClaudeToolDefinitions returns Claude tool definitions for all registered tools
func GetAllClaudeToolDefinitions() []models.ClaudeTool {
	return GlobalRegistry.GetClaudeToolDefinitions()
}

// RegistryStats provides statistics about the registry
type RegistryStats struct {
	SchemaVersion   string         `json:"schema_version"`
	TotalTools      int            `json:"total_tools"`
	ToolsByCategory map[string]int `json:"tools_by_category"`
	ToolNames       []string       `json:"tool_names"`
}

// GetRegistryStats returns statistics about the current registry state
func GetRegistryStats() RegistryStats {
	tools := ListTools()
	stats := RegistryStats{
		SchemaVersion:   "1",
		TotalTools:      len(tools),
		ToolsByCategory: make(map[string]int),
		ToolNames:       make([]string, 0, len(tools)),
	}

	for _, tool := range tools {
		stats.ToolNames = append(stats.ToolNames, tool.Name)

		category := tool.Category
		if category == "" {
			category = "uncategorized"
		}
		stats.ToolsByCategory[category]++
	}

	return stats
}
