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

package validation

import (
	"context"
)

// Service provides data validation operations
type Service interface {
	// Core validation operations
	ValidateAll(ctx context.Context) (*ValidationReport, error)
	ValidatePolicies(ctx context.Context) (*PolicyValidation, error)
	ValidateControls(ctx context.Context) (*ControlValidation, error)
	ValidateEvidenceTasks(ctx context.Context) (*EvidenceValidation, error)
	ValidateRelationships(ctx context.Context) (*RelationshipValidation, error)

	// Report generation
	GenerateReport(ctx context.Context, options ValidationOptions) (*ValidationReport, error)
	FormatReport(report *ValidationReport, format string) (string, error)
	SaveReport(report *ValidationReport, outputPath string, format string) error
}

// ValidationReport represents the overall validation report
type ValidationReport struct {
	Summary       ValidationSummary      `json:"summary"`
	Policies      PolicyValidation       `json:"policies"`
	Controls      ControlValidation      `json:"controls"`
	Evidence      EvidenceValidation     `json:"evidence"`
	Relationships RelationshipValidation `json:"relationships"`
	Issues        []ValidationIssue      `json:"issues"`
}

// ValidationSummary provides high-level validation results
type ValidationSummary struct {
	TotalPolicies      int     `json:"total_policies"`
	TotalControls      int     `json:"total_controls"`
	TotalEvidenceTasks int     `json:"total_evidence_tasks"`
	OverallScore       float64 `json:"overall_score"`
	Status             string  `json:"status"` // "pass", "warning", "fail"
	CriticalIssues     int     `json:"critical_issues"`
	Warnings           int     `json:"warnings"`
}

// PolicyValidation represents policy validation results
type PolicyValidation struct {
	WithContent            int     `json:"with_content"`
	WithSubstantialContent int     `json:"with_substantial_content"` // 10+ lines
	WithExtensiveContent   int     `json:"with_extensive_content"`   // 100+ lines
	MissingDescription     int     `json:"missing_description"`
	MissingContent         int     `json:"missing_content"`
	AverageContentLength   float64 `json:"average_content_length"`
	ContentCompleteness    float64 `json:"content_completeness"`
}

// ControlValidation represents control validation results
type ControlValidation struct {
	WithDescription     int     `json:"with_description"`
	WithPolicyLinks     int     `json:"with_policy_links"`
	MissingDescription  int     `json:"missing_description"`
	MissingPolicyLinks  int     `json:"missing_policy_links"`
	LinkageCompleteness float64 `json:"linkage_completeness"`
}

// EvidenceValidation represents evidence task validation results
type EvidenceValidation struct {
	WithDescription     int     `json:"with_description"`
	WithGuidance        int     `json:"with_guidance"`
	WithControlLinks    int     `json:"with_control_links"`
	MissingDescription  int     `json:"missing_description"`
	MissingGuidance     int     `json:"missing_guidance"`
	MissingControlLinks int     `json:"missing_control_links"`
	ContentCompleteness float64 `json:"content_completeness"`
}

// RelationshipValidation represents relationship validation results
type RelationshipValidation struct {
	ControlsWithPolicies    int `json:"controls_with_policies"`
	EvidenceWithControls    int `json:"evidence_with_controls"`
	BrokenControlReferences int `json:"broken_control_references"`
	BrokenPolicyReferences  int `json:"broken_policy_references"`
}

// ValidationIssue represents a specific validation issue
type ValidationIssue struct {
	Type        string `json:"type"`     // "critical", "warning", "info"
	Category    string `json:"category"` // "policy", "control", "evidence", "relationship"
	ID          string `json:"id"`
	Description string `json:"description"`
	Details     string `json:"details,omitempty"`
}

// ValidationOptions controls validation behavior
type ValidationOptions struct {
	Detailed    bool   `json:"detailed"`
	JSONOutput  bool   `json:"json_output"`
	OutputFile  string `json:"output_file"`
	Format      string `json:"format"` // "json", "text", "markdown"
	IncludeInfo bool   `json:"include_info"`
	FailOnWarn  bool   `json:"fail_on_warn"`
}

// DataValidator implements the validation logic
type DataValidator interface {
	ValidateAll(ctx context.Context) (*ValidationReport, error)
	ValidatePolicies(ctx context.Context) (*PolicyValidation, []ValidationIssue, error)
	ValidateControls(ctx context.Context) (*ControlValidation, []ValidationIssue, error)
	ValidateEvidenceTasks(ctx context.Context) (*EvidenceValidation, []ValidationIssue, error)
	ValidateRelationships(ctx context.Context) (*RelationshipValidation, []ValidationIssue, error)
}
