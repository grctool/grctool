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

package models

import (
	"time"
)

// ToolInfo contains metadata about a registered tool
type ToolInfo struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Version     string `json:"version,omitempty"`
	Category    string `json:"category,omitempty"`
	Enabled     bool   `json:"enabled"`
}

// Evidence represents a piece of evidence for a task (existing model)
type Evidence struct {
	ID          string            `json:"id"`
	TaskID      string            `json:"task_id"`
	Type        string            `json:"type"` // terraform, document, screenshot, etc.
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Content     string            `json:"content,omitempty"`   // For text-based evidence
	FilePath    string            `json:"file_path,omitempty"` // For file-based evidence
	Metadata    map[string]string `json:"metadata"`
	Status      string            `json:"status"` // draft, ready, submitted, approved, rejected
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

// TerraformEvidence represents Terraform-specific evidence (existing model)
type TerraformEvidence struct {
	EvidenceType string   `json:"evidence_type"` // iam_policy, vpc_config, encryption, etc.
	ResourceType string   `json:"resource_type"` // aws_iam_role, aws_s3_bucket, etc.
	ResourceName string   `json:"resource_name"`
	FilePath     string   `json:"file_path"`
	LineNumbers  string   `json:"line_numbers"`
	Content      string   `json:"content"`
	Controls     []string `json:"controls"` // Mapped security controls
}

// EvidenceRelationship represents the connection between evidence tasks, controls, and policies
type EvidenceRelationship struct {
	EvidenceTaskID    int                      `json:"evidence_task_id"`
	EvidenceTask      *EvidenceTask            `json:"evidence_task"`
	RelatedControls   []EvidenceControlSummary `json:"related_controls"`
	GoverningPolicies []EvidencePolicySummary  `json:"governing_policies"`
	RelationshipType  string                   `json:"relationship_type"` // direct, indirect, framework
	FrameworkCodes    []string                 `json:"framework_codes"`   // e.g., ["CC6.1", "CC6.2"]
	CreatedAt         time.Time                `json:"created_at"`
}

// EvidenceControlSummary represents a simplified control for relationship mapping
type EvidenceControlSummary struct {
	ID             int             `json:"id"`
	Name           string          `json:"name"`
	Category       string          `json:"category"`
	Status         string          `json:"status"`
	FrameworkCodes []FrameworkCode `json:"framework_codes"`
	OrgScope       *OrgScope       `json:"org_scope,omitempty"`
}

// EvidencePolicySummary represents a simplified policy for relationship mapping
type EvidencePolicySummary struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Type     string `json:"type"`
	Status   string `json:"status"`
	Category string `json:"category"`
}

// EvidencePrompt represents the generated prompt for Claude
type EvidencePrompt struct {
	TaskID          int             `json:"task_id"`
	GeneratedAt     time.Time       `json:"generated_at"`
	Context         EvidenceContext `json:"context"`
	PromptText      string          `json:"prompt_text"`
	ToolsAvailable  []string        `json:"tools_available"`
	ExpectedOutputs []string        `json:"expected_outputs"`
	FilePath        string          `json:"file_path"` // Where the prompt is saved
}

// EvidenceContext contains all context needed for evidence generation
type EvidenceContext struct {
	Task             EvidenceTaskDetails        `json:"task"`
	Controls         []Control                  `json:"controls"`
	Policies         []Policy                   `json:"policies"`
	ControlSummaries map[int]AIControlSummary   `json:"control_summaries,omitempty"`
	PolicySummaries  map[string]AIPolicySummary `json:"policy_summaries,omitempty"`
	FrameworkReqs    []string                   `json:"framework_requirements"`
	PreviousEvidence []string                   `json:"previous_evidence"`
	SecurityMappings SecurityMappings           `json:"security_mappings"`
	AvailableTools   []ToolInfo                 `json:"available_tools,omitempty"`
	OutputFormat     string                     `json:"output_format,omitempty"`
}

// SecurityMappings represents the security control to resource mappings
type SecurityMappings struct {
	SOC2 map[string]SecurityControlMapping `json:"soc2"`
}

// SecurityControlMapping represents the mapping of a security control to resources
type SecurityControlMapping struct {
	TerraformResources []string `json:"terraform_resources"`
	Description        string   `json:"description,omitempty"`
	Requirements       []string `json:"requirements,omitempty"`
}

// AIControlSummary represents an AI-generated summary of a control for a specific evidence task
type AIControlSummary struct {
	ControlID          int       `json:"control_id"`
	ControlName        string    `json:"control_name"`
	TaskID             int       `json:"task_id"`
	Summary            string    `json:"summary"`
	KeyRequirements    []string  `json:"key_requirements"`
	VerificationPoints []string  `json:"verification_points"`
	GeneratedAt        time.Time `json:"generated_at"`
	CacheKey           string    `json:"cache_key"`
}

// AIPolicySummary represents an AI-generated summary of a policy for a specific evidence task
type AIPolicySummary struct {
	PolicyID         string    `json:"policy_id"`
	PolicyName       string    `json:"policy_name"`
	TaskID           int       `json:"task_id"`
	Summary          string    `json:"summary"`
	KeyRequirements  []string  `json:"key_requirements"`
	RelevantSections []string  `json:"relevant_sections"`
	GeneratedAt      time.Time `json:"generated_at"`
	CacheKey         string    `json:"cache_key"`
}

// GeneratedEvidence represents the AI-generated evidence package
type GeneratedEvidence struct {
	TaskID          int                 `json:"task_id"`
	GeneratedAt     time.Time           `json:"generated_at"`
	GeneratedBy     string              `json:"generated_by"`     // "claude-3-5-sonnet"
	EvidenceFormat  string              `json:"evidence_format"`  // "csv" or "markdown"
	EvidenceContent string              `json:"evidence_content"` // Main evidence content
	SourcesUsed     []EvidenceSource    `json:"sources_used"`
	Reasoning       string              `json:"reasoning"`        // Claude's thought process
	Completeness    float64             `json:"completeness"`     // 0.0-1.0 score
	QualityScore    string              `json:"quality_score"`    // "low", "medium", "high"
	OutputDirectory string              `json:"output_directory"` // Directory where files are saved
	Status          string              `json:"status"`           // draft, ready, reviewed, submitted
	CreatedAt       time.Time           `json:"created_at"`
	UpdatedAt       time.Time           `json:"updated_at"`
	ToolsUsed       []string            `json:"tools_used"`   // Tools that were used
	Conversation    []ConversationEntry `json:"conversation"` // Full conversation history
}

// EvidenceSource tracks what sources were used to generate evidence
type EvidenceSource struct {
	Type        string                 `json:"type"`      // terraform, github, google_docs
	Resource    string                 `json:"resource"`  // file path, issue number, doc ID
	Content     string                 `json:"content"`   // extracted content
	Relevance   float64                `json:"relevance"` // 0.0-1.0 relevance score
	ExtractedAt time.Time              `json:"extracted_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"` // Additional source-specific data
}

// ClaudeRequest represents a request to Claude for evidence generation
type ClaudeRequest struct {
	Model       string                 `json:"model"`
	MaxTokens   int                    `json:"max_tokens"`
	Temperature float64                `json:"temperature"`
	Messages    []ClaudeMessage        `json:"messages"`
	Tools       []ClaudeTool           `json:"tools,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// ClaudeMessage represents a message in the Claude conversation
type ClaudeMessage struct {
	Role    string `json:"role"` // user, assistant, system
	Content string `json:"content"`
}

// ClaudeResponse represents Claude's response
type ClaudeResponse struct {
	ID           string          `json:"id"`
	Type         string          `json:"type"`
	Role         string          `json:"role"`
	Model        string          `json:"model"`
	Content      []ClaudeContent `json:"content"`
	StopReason   string          `json:"stop_reason"`
	StopSequence *string         `json:"stop_sequence"`
	Usage        ClaudeUsage     `json:"usage"`
}

// ClaudeContent represents content in Claude's response
type ClaudeContent struct {
	Type  string                 `json:"type"` // text, tool_use, tool_result
	Text  string                 `json:"text,omitempty"`
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
}

// ClaudeUsage represents token usage in Claude's response
type ClaudeUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

// ClaudeTool represents a tool available to Claude
type ClaudeTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ToolUse represents a tool call in Claude's response
type ToolUse struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input,omitempty"`
}

// ConversationEntry represents a single message in the evidence generation conversation
type ConversationEntry struct {
	Role      string    `json:"role"` // user, assistant, tool
	Content   string    `json:"content"`
	ToolCalls []ToolUse `json:"tool_calls,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}

// TerraformScanResult represents the result of scanning Terraform files
type TerraformScanResult struct {
	FilePath          string                 `json:"file_path"`
	ResourceType      string                 `json:"resource_type"`
	ResourceName      string                 `json:"resource_name"`
	Configuration     map[string]interface{} `json:"configuration"`
	LineStart         int                    `json:"line_start"`
	LineEnd           int                    `json:"line_end"`
	SecurityRelevance []string               `json:"security_relevance"` // Which controls this relates to
}

// GitHubIssueResult represents a GitHub issue relevant to security evidence
type GitHubLabel struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type GitHubIssueResult struct {
	Number    int           `json:"number"`
	Title     string        `json:"title"`
	Body      string        `json:"body"`
	State     string        `json:"state"`
	Labels    []GitHubLabel `json:"labels"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
	ClosedAt  *time.Time    `json:"closed_at,omitempty"`
	URL       string        `json:"url"`
	Relevance float64       `json:"relevance"` // Calculated relevance score
}

// EvidenceGenerationRequest represents a request to generate evidence
type EvidenceGenerationRequest struct {
	TaskID           int       `json:"task_id"`
	ToolsEnabled     []string  `json:"tools_enabled"` // terraform, github, google_docs
	OutputFormat     string    `json:"output_format"` // csv, markdown
	IncludeReasoning bool      `json:"include_reasoning"`
	MaxToolCalls     int       `json:"max_tool_calls"`
	RequestedBy      string    `json:"requested_by,omitempty"`
	RequestedAt      time.Time `json:"requested_at"`
}

// EvidenceGenerationResponse represents the response from evidence generation
type EvidenceGenerationResponse struct {
	Success         bool               `json:"success"`
	TaskID          int                `json:"task_id"`
	OutputDirectory string             `json:"output_directory"`
	GeneratedFiles  []string           `json:"generated_files"`
	Evidence        *GeneratedEvidence `json:"evidence,omitempty"`
	Error           string             `json:"error,omitempty"`
	Duration        time.Duration      `json:"duration"`
	CompletedAt     time.Time          `json:"completed_at"`
}

// EvidenceAnalysisResult represents the result of analyzing an evidence task
type EvidenceAnalysisResult struct {
	TaskID            int                      `json:"task_id"`
	Relationships     *EvidenceRelationship    `json:"relationships"`
	PromptGenerated   bool                     `json:"prompt_generated"`
	PromptPath        string                   `json:"prompt_path,omitempty"`
	SecurityMappings  []SecurityControlMapping `json:"security_mappings"`
	RecommendedTools  []string                 `json:"recommended_tools"`
	ComplexityScore   float64                  `json:"complexity_score"` // 0.0-1.0
	EstimatedDuration string                   `json:"estimated_duration"`
	AnalyzedAt        time.Time                `json:"analyzed_at"`
}

// EvidenceDataPackage represents a structured data package for external AI consumption
// This is the new prompt-as-data pattern where tools generate data instead of AI doing analysis
type EvidenceDataPackage struct {
	TaskID      int                    `json:"task_id"`
	GeneratedAt time.Time              `json:"generated_at"`
	Prompt      string                 `json:"prompt"`       // Original prompt/requirements
	ToolOutputs []ToolOutput           `json:"tool_outputs"` // Results from each tool
	Sources     []EvidenceSource       `json:"sources"`      // Source data collected
	Metadata    map[string]interface{} `json:"metadata"`     // Package metadata
}

// ToolOutput represents the output from a single tool execution
type ToolOutput struct {
	ToolName  string                 `json:"tool_name"` // Name of the tool executed
	Result    string                 `json:"result"`    // Tool execution result
	Success   bool                   `json:"success"`   // Whether tool executed successfully
	Timestamp time.Time              `json:"timestamp"` // When tool was executed
	Metadata  map[string]interface{} `json:"metadata"`  // Tool-specific metadata
}

// PolicySummaryContext represents the context needed for policy summary generation
type PolicySummaryContext struct {
	Task   EvidenceTaskDetails `json:"task"`   // Evidence task details
	Policy Policy              `json:"policy"` // Policy to summarize
}

// ControlSummaryContext represents the context needed for control summary generation
type ControlSummaryContext struct {
	Task    EvidenceTaskDetails `json:"task"`    // Evidence task details
	Control Control             `json:"control"` // Control to summarize
}
