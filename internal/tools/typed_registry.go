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
	"sync"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/tools/types"
)

// TypedToolRegistry manages both legacy and typed tools
type TypedToolRegistry struct {
	mu          sync.RWMutex
	legacyTools map[string]Tool
	typedTools  map[string]types.TypedTool
	adapters    map[string]*types.ToolAdapter
}

// Global typed registry instance
var typedRegistry *TypedToolRegistry
var typedRegistryOnce sync.Once

// GetTypedRegistry returns the global typed tool registry
func GetTypedRegistry() *TypedToolRegistry {
	typedRegistryOnce.Do(func() {
		typedRegistry = &TypedToolRegistry{
			legacyTools: make(map[string]Tool),
			typedTools:  make(map[string]types.TypedTool),
			adapters:    make(map[string]*types.ToolAdapter),
		}
	})
	return typedRegistry
}

// RegisterTypedTool registers a typed tool
func (r *TypedToolRegistry) RegisterTypedTool(tool types.TypedTool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()
	if _, exists := r.typedTools[name]; exists {
		return fmt.Errorf("typed tool with name '%s' already registered", name)
	}

	r.typedTools[name] = tool

	// If this tool also implements the legacy interface, register it there too
	if legacyTool, ok := tool.(Tool); ok {
		r.legacyTools[name] = legacyTool
	}

	return nil
}

// RegisterLegacyTool registers a legacy tool and creates an adapter
func (r *TypedToolRegistry) RegisterLegacyTool(tool Tool) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := tool.Name()

	// Check if we already have this as a typed tool
	if _, exists := r.typedTools[name]; exists {
		// Already have a typed version, don't override
		return nil
	}

	r.legacyTools[name] = tool

	// Create adapter for legacy tool
	if legacyOnly, ok := tool.(types.LegacyTool); ok {
		adapter := types.NewToolAdapter(legacyOnly)
		r.adapters[name] = adapter
		r.typedTools[name] = adapter
	}

	return nil
}

// GetTypedTool returns a typed tool by name
func (r *TypedToolRegistry) GetTypedTool(name string) (types.TypedTool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.typedTools[name]
	return tool, exists
}

// GetLegacyTool returns a legacy tool by name
func (r *TypedToolRegistry) GetLegacyTool(name string) (Tool, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	tool, exists := r.legacyTools[name]
	return tool, exists
}

// ListTypedTools returns all registered typed tools
func (r *TypedToolRegistry) ListTypedTools() map[string]types.TypedTool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]types.TypedTool)
	for name, tool := range r.typedTools {
		result[name] = tool
	}
	return result
}

// ListLegacyTools returns all registered legacy tools
func (r *TypedToolRegistry) ListLegacyTools() map[string]Tool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]Tool)
	for name, tool := range r.legacyTools {
		result[name] = tool
	}
	return result
}

// HasTypedVersion returns true if a tool has a typed implementation
func (r *TypedToolRegistry) HasTypedVersion(name string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, hasTyped := r.typedTools[name]
	_, isAdapter := r.adapters[name]

	return hasTyped && !isAdapter
}

// InitializeTypedToolRegistry initializes both legacy and typed tools
func InitializeTypedToolRegistry(cfg *config.Config, log logger.Logger) error {
	registry := GetTypedRegistry()

	// Initialize typed versions of core tools first
	if err := registerTypedCoreTools(registry, cfg, log); err != nil {
		return fmt.Errorf("failed to register typed core tools: %w", err)
	}

	// Initialize legacy tools (this also calls the original InitializeToolRegistry)
	if err := InitializeToolRegistry(cfg, log); err != nil {
		return fmt.Errorf("failed to initialize legacy tools: %w", err)
	}

	// Register all legacy tools with the typed registry
	if err := registerLegacyToolsWithTypedRegistry(registry, cfg, log); err != nil {
		return fmt.Errorf("failed to register legacy tools with typed registry: %w", err)
	}

	log.Info("Successfully initialized typed tool registry",
		logger.Field{Key: "typed_tools", Value: len(registry.typedTools)},
		logger.Field{Key: "legacy_tools", Value: len(registry.legacyTools)})

	return nil
}

// registerTypedCoreTools registers the core typed tool implementations
func registerTypedCoreTools(registry *TypedToolRegistry, cfg *config.Config, log logger.Logger) error {
	// Register typed Terraform tool
	if typedTerraformTool := NewTypedTerraformTool(cfg, log); typedTerraformTool != nil {
		if err := registry.RegisterTypedTool(typedTerraformTool); err != nil {
			log.Error("Failed to register typed terraform tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered typed terraform tool")
		}
	}

	// Register typed GitHub tool
	if typedGitHubTool := NewTypedGitHubTool(cfg, log); typedGitHubTool != nil {
		if err := registry.RegisterTypedTool(typedGitHubTool); err != nil {
			log.Error("Failed to register typed github tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered typed github tool")
		}
	}

	// Register typed evidence task details tool
	if typedEvidenceTaskTool := NewTypedEvidenceTaskDetailsTool(cfg, log); typedEvidenceTaskTool != nil {
		if err := registry.RegisterTypedTool(typedEvidenceTaskTool); err != nil {
			log.Error("Failed to register typed evidence task details tool", logger.Field{Key: "error", Value: err})
		} else {
			log.Debug("Registered typed evidence task details tool")
		}
	}

	return nil
}

// registerLegacyToolsWithTypedRegistry registers existing legacy tools with the typed registry
func registerLegacyToolsWithTypedRegistry(registry *TypedToolRegistry, cfg *config.Config, log logger.Logger) error {
	// Get all tools from the existing registry
	existingTools := ListTools()

	for _, toolInfo := range existingTools {
		// Only register if we don't already have a typed version
		if !registry.HasTypedVersion(toolInfo.Name) {
			// Get the actual tool instance from the global registry
			if tool, err := GlobalRegistry.Get(toolInfo.Name); err == nil {
				if err := registry.RegisterLegacyTool(tool); err != nil {
					log.Warn("Failed to register legacy tool with typed registry",
						logger.Field{Key: "tool", Value: toolInfo.Name},
						logger.Field{Key: "error", Value: err})
				}
			}
		}
	}

	return nil
}

// ValidateAndExecuteTypedTool is a utility function for command handlers
func ValidateAndExecuteTypedTool(toolName string, params map[string]interface{}) (types.Response, error) {
	registry := GetTypedRegistry()

	// Get the typed tool
	tool, exists := registry.GetTypedTool(toolName)
	if !exists {
		return nil, fmt.Errorf("tool '%s' not found", toolName)
	}

	// Create and validate the request
	req, err := types.ValidateAndConvertParams(toolName, params)
	if err != nil {
		return nil, fmt.Errorf("invalid request parameters: %w", err)
	}

	// Execute the tool
	return tool.ExecuteTyped(context.TODO(), req)
}
