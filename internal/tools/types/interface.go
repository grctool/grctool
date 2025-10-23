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

package types

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/grctool/grctool/internal/models"
)

// TypedTool defines the new interface for evidence collection tools with typed requests
type TypedTool interface {
	// GetClaudeToolDefinition returns the tool definition for Claude
	GetClaudeToolDefinition() models.ClaudeTool

	// ExecuteTyped runs the tool with a typed request
	// Returns: typed response, error
	ExecuteTyped(ctx context.Context, req Request) (Response, error)

	// Name returns the tool name
	Name() string

	// Description returns the tool description
	Description() string
}

// LegacyTool is the original interface for backward compatibility
type LegacyTool interface {
	// Execute runs the tool with the given parameters (legacy)
	Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error)

	// GetClaudeToolDefinition returns the tool definition for Claude
	GetClaudeToolDefinition() models.ClaudeTool

	// Name returns the tool name
	Name() string

	// Description returns the tool description
	Description() string
}

// Tool is a unified interface that supports both typed and legacy execution
type Tool interface {
	TypedTool
	LegacyTool
}

// ToolAdapter provides a bridge between legacy tools and the new typed interface
type ToolAdapter struct {
	legacyTool LegacyTool
	toolName   string
}

// NewToolAdapter creates an adapter that wraps a legacy tool
func NewToolAdapter(legacy LegacyTool) *ToolAdapter {
	return &ToolAdapter{
		legacyTool: legacy,
		toolName:   legacy.Name(),
	}
}

// Name implements TypedTool
func (a *ToolAdapter) Name() string {
	return a.legacyTool.Name()
}

// Description implements TypedTool
func (a *ToolAdapter) Description() string {
	return a.legacyTool.Description()
}

// GetClaudeToolDefinition implements TypedTool
func (a *ToolAdapter) GetClaudeToolDefinition() models.ClaudeTool {
	return a.legacyTool.GetClaudeToolDefinition()
}

// ExecuteTyped implements TypedTool by converting typed request to legacy format
func (a *ToolAdapter) ExecuteTyped(ctx context.Context, req Request) (Response, error) {
	// Validate the request
	if err := req.Validate(); err != nil {
		return NewErrorResponse(a.toolName, fmt.Sprintf("request validation failed: %v", err), nil), nil
	}

	// Convert typed request to legacy map format
	params, err := a.requestToLegacyParams(req)
	if err != nil {
		return NewErrorResponse(a.toolName, fmt.Sprintf("failed to convert request: %v", err), nil), nil
	}

	// Call legacy Execute method
	content, source, err := a.legacyTool.Execute(ctx, params)
	if err != nil {
		return NewErrorResponse(a.toolName, err.Error(), nil), nil
	}

	// Convert legacy response to typed response
	metadata := make(map[string]interface{})
	if source != nil {
		metadata = source.Metadata
	}

	return NewSuccessResponse(a.toolName, content, source, metadata), nil
}

// Execute implements LegacyTool by passing through to wrapped tool
func (a *ToolAdapter) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	return a.legacyTool.Execute(ctx, params)
}

// requestToLegacyParams converts a typed request to legacy map format
func (a *ToolAdapter) requestToLegacyParams(req Request) (map[string]interface{}, error) {
	// Use JSON marshaling/unmarshaling to convert
	jsonBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var params map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &params); err != nil {
		return nil, fmt.Errorf("failed to unmarshal to legacy format: %w", err)
	}

	return params, nil
}

// RequestMatcher helps identify what type of request to use for a given tool
type RequestMatcher struct {
	toolRequestTypes map[string]func() Request
}

// NewRequestMatcher creates a new request matcher
func NewRequestMatcher() *RequestMatcher {
	return &RequestMatcher{
		toolRequestTypes: make(map[string]func() Request),
	}
}

// RegisterRequestType registers a request type for a tool name
func (m *RequestMatcher) RegisterRequestType(toolName string, factory func() Request) {
	m.toolRequestTypes[toolName] = factory
}

// CreateRequestForTool creates the appropriate request type for a tool
func (m *RequestMatcher) CreateRequestForTool(toolName string) (Request, error) {
	factory, exists := m.toolRequestTypes[toolName]
	if !exists {
		return nil, fmt.Errorf("no request type registered for tool: %s", toolName)
	}
	return factory(), nil
}

// Initialize the default request matcher with built-in types
func init() {
	DefaultRequestMatcher.RegisterRequestType("terraform_scanner", func() Request { return &TerraformRequest{} })
	DefaultRequestMatcher.RegisterRequestType("github_searcher", func() Request { return &GitHubRequest{} })
	DefaultRequestMatcher.RegisterRequestType("evidence_task_details", func() Request { return &EvidenceTaskRequest{} })
	DefaultRequestMatcher.RegisterRequestType("prompt_assembler", func() Request { return &PromptAssemblerRequest{} })
	DefaultRequestMatcher.RegisterRequestType("docs_reader", func() Request { return &DocsReaderRequest{} })
	DefaultRequestMatcher.RegisterRequestType("google_workspace", func() Request { return &GoogleWorkspaceRequest{} })
	DefaultRequestMatcher.RegisterRequestType("control_summary_generator", func() Request { return &ControlSummaryRequest{} })
	DefaultRequestMatcher.RegisterRequestType("policy_summary_generator", func() Request { return &PolicySummaryRequest{} })
}

// DefaultRequestMatcher is the global request matcher instance
var DefaultRequestMatcher = NewRequestMatcher()
