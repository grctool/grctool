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
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/logger"
	"gopkg.in/yaml.v3"
)

// CollectionPlanManager handles collection plan operations
type CollectionPlanManager struct {
	config *config.Config
	logger logger.Logger
}

// NewCollectionPlanManager creates a new plan manager
func NewCollectionPlanManager(cfg *config.Config, log logger.Logger) *CollectionPlanManager {
	return &CollectionPlanManager{
		config: cfg,
		logger: log,
	}
}

// CollectionPlan represents the plan structure
type CollectionPlan struct {
	TaskRef            string          `json:"task_ref" yaml:"task_ref"`
	TaskName           string          `json:"task_name" yaml:"task_name"`
	TaskDescription    string          `json:"task_description" yaml:"task_description"`
	Window             string          `json:"window" yaml:"window"`
	CollectionInterval string          `json:"collection_interval" yaml:"collection_interval"`
	Status             string          `json:"status" yaml:"status"`
	Completeness       float64         `json:"completeness" yaml:"completeness"`
	Strategy           string          `json:"strategy" yaml:"strategy"`
	Reasoning          string          `json:"reasoning" yaml:"reasoning"`
	Controls           []ControlRef    `json:"controls" yaml:"controls"`
	Entries            []EvidenceEntry `json:"entries" yaml:"entries"`
	Gaps               []EvidenceGap   `json:"gaps" yaml:"gaps"`
	LastUpdated        time.Time       `json:"last_updated" yaml:"last_updated"`
}

// EvidenceEntry represents a single evidence file in the collection plan
type EvidenceEntry struct {
	Filename          string    `json:"filename" yaml:"filename"`
	Title             string    `json:"title" yaml:"title"`
	Source            string    `json:"source" yaml:"source"`
	ControlsSatisfied []string  `json:"controls_satisfied" yaml:"controls_satisfied"`
	Status            string    `json:"status" yaml:"status"` // complete, partial, pending
	CollectedAt       time.Time `json:"collected_at" yaml:"collected_at"`
	Summary           string    `json:"summary" yaml:"summary"`
}

// EvidenceGap represents a missing evidence item
type EvidenceGap struct {
	Description string    `json:"description" yaml:"description"`
	Priority    string    `json:"priority" yaml:"priority"` // high, medium, low
	Remediation string    `json:"remediation" yaml:"remediation"`
	DueDate     time.Time `json:"due_date,omitempty" yaml:"due_date,omitempty"`
}

// Helper functions for completeness calculation and formatting

// CalculateCompleteness scores evidence completeness
// Pure function for completeness calculation
func CalculateCompleteness(entries []EvidenceEntry) float64 {
	if len(entries) == 0 {
		return 0.0
	}

	total := 0.0
	for _, entry := range entries {
		switch entry.Status {
		case "complete":
			total += 1.0
		case "partial":
			total += 0.5
		}
		// "pending" adds 0.0
	}

	return total / float64(len(entries))
}

// FormatCompleteness converts a completeness score to percentage string
// Pure function for display formatting
func FormatCompleteness(score float64) string {
	percentage := int(score * 100)
	return fmt.Sprintf("%d%%", percentage)
}

// GetCompletenessStatus returns status based on completeness score
// Pure function for status determination
func GetCompletenessStatus(score float64) string {
	if score >= 0.9 {
		return "Complete"
	} else if score >= 0.5 {
		return "In Progress"
	} else if score > 0.0 {
		return "Started"
	}
	return "Not Started"
}

// ControlRef represents a control referenced in the plan
type ControlRef struct {
	ID          string `json:"id" yaml:"id"`
	ReferenceID string `json:"reference_id" yaml:"reference_id"`
	Name        string `json:"name" yaml:"name"`
	Category    string `json:"category" yaml:"category"`
}

// LoadOrCreatePlan loads an existing plan or creates a new one
func (cpm *CollectionPlanManager) LoadOrCreatePlan(task *domain.EvidenceTask, window string, planPath string) (*CollectionPlan, error) {
	cpm.logger.Debug("Loading or creating collection plan",
		logger.Field{Key: "task_ref", Value: task.ReferenceID},
		logger.Field{Key: "window", Value: window},
		logger.Field{Key: "plan_path", Value: planPath})

	// Try to load existing plan
	if _, err := os.Stat(planPath); err == nil {
		plan, err := cpm.loadPlanFromFile(planPath)
		if err != nil {
			cpm.logger.Warn("Failed to load existing plan, creating new one",
				logger.Field{Key: "error", Value: err})
		} else {
			return plan, nil
		}
	}

	// Create new plan
	plan := &CollectionPlan{
		TaskRef:            task.ReferenceID,
		TaskName:           task.Name,
		TaskDescription:    task.Description,
		Window:             window,
		CollectionInterval: task.CollectionInterval,
		Status:             "Not Started",
		Completeness:       0.0,
		Strategy:           "",
		Reasoning:          "",
		Controls:           cpm.convertControls(task.RelatedControls),
		Entries:            []EvidenceEntry{},
		Gaps:               []EvidenceGap{},
		LastUpdated:        time.Now(),
	}

	return plan, nil
}

// SavePlan saves the plan to both markdown and metadata
func (cpm *CollectionPlanManager) SavePlan(plan *CollectionPlan, planPath string) error {
	cpm.logger.Debug("Saving collection plan",
		logger.Field{Key: "task_ref", Value: plan.TaskRef},
		logger.Field{Key: "plan_path", Value: planPath})

	// Update plan metadata
	plan.Completeness = CalculateCompleteness(plan.Entries)
	plan.Status = GetCompletenessStatus(plan.Completeness)
	plan.LastUpdated = time.Now()

	// Generate markdown content
	markdownContent, err := cpm.generateMarkdownPlan(plan)
	if err != nil {
		return fmt.Errorf("failed to generate markdown plan: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(planPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create plan directory: %w", err)
	}

	// Write markdown file
	if err := os.WriteFile(planPath, []byte(markdownContent), 0644); err != nil {
		return fmt.Errorf("failed to write plan file: %w", err)
	}

	// Also save metadata as YAML for programmatic access
	metadataPath := strings.Replace(planPath, ".md", "_metadata.yaml", 1)
	metadataContent, err := yaml.Marshal(plan)
	if err != nil {
		return fmt.Errorf("failed to marshal plan metadata: %w", err)
	}

	if err := os.WriteFile(metadataPath, metadataContent, 0644); err != nil {
		return fmt.Errorf("failed to write plan metadata: %w", err)
	}

	cpm.logger.Info("Collection plan saved successfully",
		logger.Field{Key: "task_ref", Value: plan.TaskRef},
		logger.Field{Key: "completeness", Value: FormatCompleteness(plan.Completeness)})

	return nil
}

// AddEvidenceEntry adds a new evidence entry to the plan
func (cpm *CollectionPlanManager) AddEvidenceEntry(plan *CollectionPlan, entry EvidenceEntry) {
	cpm.logger.Debug("Adding evidence entry to plan",
		logger.Field{Key: "filename", Value: entry.Filename},
		logger.Field{Key: "status", Value: entry.Status})

	plan.Entries = append(plan.Entries, entry)
	plan.Completeness = CalculateCompleteness(plan.Entries)
	plan.Status = GetCompletenessStatus(plan.Completeness)
	plan.LastUpdated = time.Now()
}

// UpdateStrategy updates the collection strategy and reasoning
func (cpm *CollectionPlanManager) UpdateStrategy(plan *CollectionPlan, strategy, reasoning string) {
	cpm.logger.Debug("Updating collection strategy",
		logger.Field{Key: "task_ref", Value: plan.TaskRef})

	plan.Strategy = strategy
	plan.Reasoning = reasoning
	plan.LastUpdated = time.Now()
}

// AddGap adds a gap to the plan
func (cpm *CollectionPlanManager) AddGap(plan *CollectionPlan, gap EvidenceGap) {
	cpm.logger.Debug("Adding gap to plan",
		logger.Field{Key: "description", Value: gap.Description})

	plan.Gaps = append(plan.Gaps, gap)
	plan.LastUpdated = time.Now()
}

// loadPlanFromFile loads a plan from the metadata YAML file
func (cpm *CollectionPlanManager) loadPlanFromFile(planPath string) (*CollectionPlan, error) {
	metadataPath := strings.Replace(planPath, ".md", "_metadata.yaml", 1)

	data, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read plan metadata: %w", err)
	}

	var plan CollectionPlan
	if err := yaml.Unmarshal(data, &plan); err != nil {
		return nil, fmt.Errorf("failed to unmarshal plan metadata: %w", err)
	}

	return &plan, nil
}

// convertControls converts domain controls to plan control references
func (cpm *CollectionPlanManager) convertControls(controls []domain.Control) []ControlRef {
	refs := make([]ControlRef, len(controls))
	for i, control := range controls {
		refs[i] = ControlRef{
			ID:          fmt.Sprintf("%d", control.ID),
			ReferenceID: control.ReferenceID,
			Name:        control.Name,
			Category:    control.Category,
		}
	}
	return refs
}

// generateMarkdownPlan generates the markdown representation of the plan
func (cpm *CollectionPlanManager) generateMarkdownPlan(plan *CollectionPlan) (string, error) {
	tmpl := `# Evidence Collection Plan - {{.TaskRef}} {{.TaskName}}
**Window**: {{.Window}}  
**Collection Interval**: {{.CollectionInterval}}  
**Last Updated**: {{.LastUpdated.Format "2006-01-02T15:04:05Z07:00"}}  
**Status**: {{.Status}} ({{printf "%.0f" (mul .Completeness 100)}}% Complete)

## Task Requirements
**Description**: {{.TaskDescription}}

**Controls Addressed**:{{range .Controls}}
- {{.ReferenceID}}: {{.Name}}{{end}}

{{if .Strategy}}## Collection Strategy

{{.Strategy}}

{{end}}{{if .Reasoning}}## Reasoning

{{.Reasoning}}

{{end}}## Evidence Inventory

{{if .Entries}}| File | Source | Controls | Status |
|------|--------|----------|--------|{{range .Entries}}
| {{.Filename}} | {{.Source}} | {{join .ControlsSatisfied ", "}} | {{statusIcon .Status}} {{.Status}} |{{end}}

{{else}}No evidence collected yet.

{{end}}{{if .Gaps}}## Gaps and Next Steps

| Gap | Priority | Remediation | Due Date |
|-----|----------|-------------|----------|{{range .Gaps}}
| {{.Description}} | {{.Priority}} | {{.Remediation}} | {{if not .DueDate.IsZero}}{{.DueDate.Format "2006-01-02"}}{{else}}TBD{{end}} |{{end}}

{{end}}## Completeness Assessment
- Overall Completeness: **{{printf "%.0f" (mul .Completeness 100)}}%**
- Status: **{{.Status}}**
- Evidence Files: {{len .Entries}}
- Identified Gaps: {{len .Gaps}}

{{if eq .CollectionInterval "year"}}## Notes
- Next collection window begins: {{add .LastUpdated.Year 1}}-01-01
{{else if eq .CollectionInterval "quarter"}}## Notes  
- Next collection window: Next quarter
{{else if eq .CollectionInterval "month"}}## Notes
- Next collection window: Next month
{{end}}`

	// Create template with helper functions
	t := template.New("plan").Funcs(template.FuncMap{
		"mul":  func(a, b float64) float64 { return a * b },
		"add":  func(a, b int) int { return a + b },
		"join": func(strs []string, sep string) string { return strings.Join(strs, sep) },
		"statusIcon": func(status string) string {
			switch status {
			case "complete":
				return "✅"
			case "partial":
				return "⚠️"
			case "pending":
				return "❌"
			default:
				return "⏳"
			}
		},
	})

	t, err := t.Parse(tmpl)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	var buf strings.Builder
	if err := t.Execute(&buf, plan); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}
