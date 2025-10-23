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
	"fmt"
	"os"
)

// TerraformRequest represents a request to the Terraform scanner tool
type TerraformRequest struct {
	ResourceTypes []string `json:"resource_types,omitempty" validate:"dive,min=1"`
	ControlCodes  []string `json:"control_codes,omitempty" validate:"dive,min=1"`
	OutputFormat  string   `json:"output_format,omitempty" validate:"omitempty,oneof=csv markdown"`
	ScanPaths     []string `json:"scan_paths,omitempty" validate:"omitempty,dive,required"`
	MaxDepth      int      `json:"max_depth,omitempty" validate:"omitempty,min=1,max=10"`
}

// Validate validates the TerraformRequest
func (r *TerraformRequest) Validate() error {
	// Set default output format if not provided
	if r.OutputFormat == "" {
		r.OutputFormat = "csv"
	}

	// Validate using struct tags
	if err := ValidateStruct(r); err != nil {
		return fmt.Errorf("terraform request validation failed: %w", err)
	}

	// Custom validation for scan paths if provided
	for _, path := range r.ScanPaths {
		if info, err := os.Stat(path); err != nil {
			return fmt.Errorf("scan path '%s' does not exist: %w", path, err)
		} else if !info.IsDir() {
			return fmt.Errorf("scan path '%s' is not a directory", path)
		}
	}

	return nil
}

// GitHubRequest represents a request to the GitHub search tool
type GitHubRequest struct {
	Query         string   `json:"query" validate:"required,min=1"`
	Labels        []string `json:"labels,omitempty" validate:"dive,min=1"`
	IncludeClosed bool     `json:"include_closed,omitempty"`
	Repository    string   `json:"repository,omitempty" validate:"omitempty,min=1"`
	SearchType    string   `json:"search_type,omitempty" validate:"omitempty,oneof=issues prs commits"`
	MaxResults    int      `json:"max_results,omitempty" validate:"omitempty,min=1,max=100"`
}

// Validate validates the GitHubRequest
func (r *GitHubRequest) Validate() error {
	// Set default values
	if r.SearchType == "" {
		r.SearchType = "issues"
	}
	if r.MaxResults <= 0 {
		r.MaxResults = 50
	}

	// Validate using struct tags
	if err := ValidateStruct(r); err != nil {
		return fmt.Errorf("github request validation failed: %w", err)
	}

	return nil
}

// EvidenceTaskRequest represents a request to get evidence task details
type EvidenceTaskRequest struct {
	TaskRef string `json:"task_ref" validate:"required,min=1"`
}

// Validate validates the EvidenceTaskRequest
func (r *EvidenceTaskRequest) Validate() error {
	if err := ValidateStruct(r); err != nil {
		return fmt.Errorf("evidence task request validation failed: %w", err)
	}
	return nil
}

// PromptAssemblerRequest represents a request to the prompt assembler tool
type PromptAssemblerRequest struct {
	TaskRef        string `json:"task_ref" validate:"required,min=1"`
	IncludeContext bool   `json:"include_context,omitempty"`
	Format         string `json:"format,omitempty" validate:"omitempty,oneof=markdown text"`
}

// Validate validates the PromptAssemblerRequest
func (r *PromptAssemblerRequest) Validate() error {
	// Set default format if not provided
	if r.Format == "" {
		r.Format = "markdown"
	}

	if err := ValidateStruct(r); err != nil {
		return fmt.Errorf("prompt assembler request validation failed: %w", err)
	}
	return nil
}

// DocsReaderRequest represents a request to the docs reader tool
type DocsReaderRequest struct {
	QueryTerms   []string `json:"query_terms" validate:"required,min=1,dive,min=1"`
	DocumentType string   `json:"document_type,omitempty" validate:"omitempty,oneof=policy control evidence_task"`
	MaxResults   int      `json:"max_results,omitempty" validate:"omitempty,min=1,max=50"`
}

// Validate validates the DocsReaderRequest
func (r *DocsReaderRequest) Validate() error {
	// Set default values
	if r.MaxResults <= 0 {
		r.MaxResults = 10
	}

	if err := ValidateStruct(r); err != nil {
		return fmt.Errorf("docs reader request validation failed: %w", err)
	}
	return nil
}

// GoogleWorkspaceRequest represents a request to the Google Workspace tool
type GoogleWorkspaceRequest struct {
	Query      string   `json:"query" validate:"required,min=1"`
	Services   []string `json:"services,omitempty" validate:"dive,oneof=admin drive gmail calendar meet chat"`
	MaxResults int      `json:"max_results,omitempty" validate:"omitempty,min=1,max=100"`
}

// Validate validates the GoogleWorkspaceRequest
func (r *GoogleWorkspaceRequest) Validate() error {
	// Set default values
	if r.MaxResults <= 0 {
		r.MaxResults = 25
	}

	if err := ValidateStruct(r); err != nil {
		return fmt.Errorf("google workspace request validation failed: %w", err)
	}
	return nil
}

// ControlSummaryRequest represents a request to generate control summaries
type ControlSummaryRequest struct {
	TaskRef     string `json:"task_ref" validate:"required,min=1"`
	ControlCode string `json:"control_code,omitempty" validate:"omitempty,min=1"`
}

// Validate validates the ControlSummaryRequest
func (r *ControlSummaryRequest) Validate() error {
	if err := ValidateStruct(r); err != nil {
		return fmt.Errorf("control summary request validation failed: %w", err)
	}
	return nil
}

// PolicySummaryRequest represents a request to generate policy summaries
type PolicySummaryRequest struct {
	TaskRef   string `json:"task_ref" validate:"required,min=1"`
	PolicyID  string `json:"policy_id,omitempty" validate:"omitempty,min=1"`
	MaxLength int    `json:"max_length,omitempty" validate:"omitempty,min=100,max=5000"`
}

// Validate validates the PolicySummaryRequest
func (r *PolicySummaryRequest) Validate() error {
	// Set default max length
	if r.MaxLength <= 0 {
		r.MaxLength = 1000
	}

	if err := ValidateStruct(r); err != nil {
		return fmt.Errorf("policy summary request validation failed: %w", err)
	}
	return nil
}
