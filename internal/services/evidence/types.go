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

package evidence

import (
	"context"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/services"
)

// Service provides evidence-related operations
type Service interface {
	// Core evidence operations
	ListEvidenceTasks(ctx context.Context, filter domain.EvidenceFilter) ([]domain.EvidenceTask, error)
	GetEvidenceTaskSummary(ctx context.Context) (*domain.EvidenceTaskSummary, error)
	AnalyzeEvidenceTask(ctx context.Context, taskID int) (*services.EvidenceAnalysisResult, error)
	GenerateEvidence(ctx context.Context, req *services.EvidenceGenerationRequest) (*services.EvidenceGenerationResult, error)
	ReviewEvidence(ctx context.Context, recordID string, showReasoning bool) (map[string]interface{}, error)

	// Task resolution and mapping operations
	ResolveTaskID(ctx context.Context, identifier string) (int, error)
	MapEvidenceRelationships(ctx context.Context) (*EvidenceMapResult, error)
	GenerateTemplateBasedPrompt(context *models.EvidenceContext, outputFormat string) string
	ProcessAnalysisForTask(ctx context.Context, taskID int, outputFormat string) (string, string, error)
	ProcessBulkAnalysis(ctx context.Context, outputFormat string) error

	// File and output operations
	SaveAnalysisToFile(filename, content string) error
	SaveEvidenceToFile(outputDir string, record *domain.EvidenceRecord) error
}

// EvidenceMapResult represents the result of evidence relationship mapping
type EvidenceMapResult struct {
	Tasks              []domain.EvidenceTask            `json:"tasks"`
	Controls           []domain.Control                 `json:"controls"`
	Policies           []domain.Policy                  `json:"policies"`
	FrameworkGroups    map[string][]domain.EvidenceTask `json:"framework_groups"`
	TotalRelationships int                              `json:"total_relationships"`
	Summary            *EvidenceMapSummary              `json:"summary"`
}

// EvidenceMapSummary provides summary statistics for evidence mapping
type EvidenceMapSummary struct {
	TotalTasks           int            `json:"total_tasks"`
	TotalControls        int            `json:"total_controls"`
	TotalPolicies        int            `json:"total_policies"`
	FrameworkCounts      map[string]int `json:"framework_counts"`
	StatusCounts         map[string]int `json:"status_counts"`
	PriorityCounts       map[string]int `json:"priority_counts"`
	OverdueCount         int            `json:"overdue_count"`
	AverageRelationships float64        `json:"average_relationships"`
}

// PromptGenerationOptions controls prompt generation behavior
type PromptGenerationOptions struct {
	AllTasks         bool   `json:"all_tasks"`
	OutputFile       string `json:"output_file"`
	IncludeTemplates bool   `json:"include_templates"`
	IncludeChecklist bool   `json:"include_checklist"`
	OutputFormat     string `json:"output_format"`
	MarkdownFormat   bool   `json:"markdown_format"`
}

// BulkGenerationOptions controls bulk evidence generation
type BulkGenerationOptions struct {
	All       bool     `json:"all"`
	Tools     []string `json:"tools"`
	Format    string   `json:"format"`
	OutputDir string   `json:"output_dir"`
}

// SubmissionOptions controls evidence submission behavior
type SubmissionOptions struct {
	DryRun         bool `json:"dry_run"`
	ValidateOnly   bool `json:"validate_only"`
	SkipValidation bool `json:"skip_validation"`
}

// ReviewOptions controls evidence review behavior
type ReviewOptions struct {
	ShowReasoning bool `json:"show_reasoning"`
	ShowSources   bool `json:"show_sources"`
	DetailedMode  bool `json:"detailed_mode"`
}
