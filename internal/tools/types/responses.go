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
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/models"
)

// ToolResponse is a standardized response from tool execution
type ToolResponse struct {
	Content     string                 `json:"content"`
	Source      *models.EvidenceSource `json:"source,omitempty"`
	Metadata    map[string]interface{} `json:"metadata"`
	Success     bool                   `json:"success"`
	Error       string                 `json:"error,omitempty"`
	ExecutedAt  time.Time              `json:"executed_at"`
	ExecutionID string                 `json:"execution_id,omitempty"`
	ToolName    string                 `json:"tool_name"`
	RequestType string                 `json:"request_type,omitempty"`
}

// GetContent returns the response content
func (r *ToolResponse) GetContent() string {
	return r.Content
}

// GetMetadata returns the response metadata
func (r *ToolResponse) GetMetadata() map[string]interface{} {
	return r.Metadata
}

// TerraformResponse represents a response from the Terraform scanner
type TerraformResponse struct {
	*ToolResponse
	Results          []models.TerraformScanResult `json:"results"`
	ResourceCount    int                          `json:"resource_count"`
	SecurityFindings int                          `json:"security_findings"`
	ScannedPaths     []string                     `json:"scanned_paths"`
	Format           string                       `json:"format"`
}

// GitHubResponse represents a response from the GitHub search tool
type GitHubResponse struct {
	*ToolResponse
	Issues     []models.GitHubIssueResult `json:"issues,omitempty"`
	TotalCount int                        `json:"total_count"`
	Query      string                     `json:"query"`
	SearchType string                     `json:"search_type"`
	Repository string                     `json:"repository"`
}

// EvidenceTaskResponse represents a response with evidence task details
type EvidenceTaskResponse struct {
	*ToolResponse
	Task     *domain.EvidenceTask `json:"task"`
	Controls []domain.Control     `json:"controls,omitempty"`
	Policies []domain.Policy      `json:"policies,omitempty"`
	TaskRef  string               `json:"task_ref"`
}

// PromptAssemblerResponse represents a response from the prompt assembler
type PromptAssemblerResponse struct {
	*ToolResponse
	Prompt             string   `json:"prompt"`
	TaskRef            string   `json:"task_ref"`
	IncludedControls   []string `json:"included_controls,omitempty"`
	IncludedPolicies   []string `json:"included_policies,omitempty"`
	PromptLength       int      `json:"prompt_length"`
	GeneratedTimestamp string   `json:"generated_timestamp"`
}

// DocsReaderResponse represents a response from the docs reader
type DocsReaderResponse struct {
	*ToolResponse
	Documents    []DocumentResult `json:"documents"`
	QueryTerms   []string         `json:"query_terms"`
	DocumentType string           `json:"document_type,omitempty"`
	TotalFound   int              `json:"total_found"`
}

// DocumentResult represents a single document found by the docs reader
type DocumentResult struct {
	Type      string                 `json:"type"`
	ID        string                 `json:"id"`
	Title     string                 `json:"title"`
	Content   string                 `json:"content"`
	Relevance float64                `json:"relevance"`
	Path      string                 `json:"path"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

// GoogleWorkspaceResponse represents a response from Google Workspace tool
type GoogleWorkspaceResponse struct {
	*ToolResponse
	Results      []GoogleWorkspaceResult `json:"results"`
	Query        string                  `json:"query"`
	Services     []string                `json:"services"`
	TotalResults int                     `json:"total_results"`
	QueryTime    time.Duration           `json:"query_time"`
}

// GoogleWorkspaceResult represents a single result from Google Workspace
type GoogleWorkspaceResult struct {
	Service     string                 `json:"service"`
	Type        string                 `json:"type"`
	ID          string                 `json:"id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	URL         string                 `json:"url,omitempty"`
	Relevance   float64                `json:"relevance"`
	CreatedAt   *time.Time             `json:"created_at,omitempty"`
	UpdatedAt   *time.Time             `json:"updated_at,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ControlSummaryResponse represents a response from control summary generation
type ControlSummaryResponse struct {
	*ToolResponse
	TaskRef       string `json:"task_ref"`
	ControlCode   string `json:"control_code,omitempty"`
	Summary       string `json:"summary"`
	SummaryLength int    `json:"summary_length"`
}

// PolicySummaryResponse represents a response from policy summary generation
type PolicySummaryResponse struct {
	*ToolResponse
	TaskRef       string `json:"task_ref"`
	PolicyID      string `json:"policy_id,omitempty"`
	Summary       string `json:"summary"`
	SummaryLength int    `json:"summary_length"`
	MaxLength     int    `json:"max_length"`
}

// ValidationResponse represents a response from validation tools
type ValidationResponse struct {
	*ToolResponse
	IsValid     bool                `json:"is_valid"`
	Errors      []ValidationError   `json:"errors,omitempty"`
	Warnings    []ValidationWarning `json:"warnings,omitempty"`
	ValidatedAt time.Time           `json:"validated_at"`
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// ValidationWarning represents a validation warning
type ValidationWarning struct {
	Field   string `json:"field"`
	Value   string `json:"value,omitempty"`
	Message string `json:"message"`
	Code    string `json:"code,omitempty"`
}

// NewSuccessResponse creates a new successful tool response
func NewSuccessResponse(toolName, content string, source *models.EvidenceSource, metadata map[string]interface{}) *ToolResponse {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &ToolResponse{
		Content:    content,
		Source:     source,
		Metadata:   metadata,
		Success:    true,
		ExecutedAt: time.Now(),
		ToolName:   toolName,
	}
}

// NewErrorResponse creates a new error tool response
func NewErrorResponse(toolName, errorMsg string, metadata map[string]interface{}) *ToolResponse {
	if metadata == nil {
		metadata = make(map[string]interface{})
	}

	return &ToolResponse{
		Content:    "",
		Metadata:   metadata,
		Success:    false,
		Error:      errorMsg,
		ExecutedAt: time.Now(),
		ToolName:   toolName,
	}
}
