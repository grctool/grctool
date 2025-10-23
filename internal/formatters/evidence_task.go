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

package formatters

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interpolation"
	"github.com/grctool/grctool/internal/registry"
	"github.com/grctool/grctool/internal/utils"
)

// EvidenceTaskFormatter handles formatting evidence tasks into various output formats
type EvidenceTaskFormatter struct {
	baseFormatter *BaseFormatter
	registry      *registry.EvidenceTaskRegistry
	// Optional control details for enhanced formatting
	relatedControls map[string]domain.Control
	// Reference mapping for tests
	referenceMapping map[int]string
}

// NewEvidenceTaskFormatter creates a new evidence task formatter with default (disabled) interpolation
func NewEvidenceTaskFormatter() *EvidenceTaskFormatter {
	// Create a disabled interpolator as default to maintain backward compatibility
	config := interpolation.InterpolatorConfig{
		Variables:         make(map[string]string),
		Enabled:           false,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(config)
	return &EvidenceTaskFormatter{
		baseFormatter:    NewBaseFormatter(interpolator),
		registry:         nil, // Will be set by SetRegistry
		relatedControls:  make(map[string]domain.Control),
		referenceMapping: make(map[int]string),
	}
}

// NewEvidenceTaskFormatterWithInterpolation creates a new evidence task formatter with the given interpolator
func NewEvidenceTaskFormatterWithInterpolation(interpolator interpolation.Interpolator) *EvidenceTaskFormatter {
	return &EvidenceTaskFormatter{
		baseFormatter:    NewBaseFormatter(interpolator),
		registry:         nil, // Will be set by SetRegistry
		relatedControls:  make(map[string]domain.Control),
		referenceMapping: make(map[int]string),
	}
}

// SetRegistry sets the evidence task registry for this formatter
func (etf *EvidenceTaskFormatter) SetRegistry(registry *registry.EvidenceTaskRegistry) {
	etf.registry = registry
}

// SetRelatedControls sets the control details for enhanced formatting
func (etf *EvidenceTaskFormatter) SetRelatedControls(controls []domain.Control) {
	etf.relatedControls = make(map[string]domain.Control)
	for _, control := range controls {
		// Map by both string and int representations of ID
		controlIDStr := fmt.Sprintf("%d", control.ID)
		etf.relatedControls[controlIDStr] = control
	}
}

// InitializeReferenceMapping initializes reference mapping for evidence tasks (for testing)
func (etf *EvidenceTaskFormatter) InitializeReferenceMapping(tasks []domain.EvidenceTask) {
	etf.referenceMapping = make(map[int]string)

	// Sort tasks by ID to ensure consistent reference assignment
	sortedTasks := make([]domain.EvidenceTask, len(tasks))
	copy(sortedTasks, tasks)

	// Simple bubble sort by ID
	for i := 0; i < len(sortedTasks); i++ {
		for j := i + 1; j < len(sortedTasks); j++ {
			if sortedTasks[i].ID > sortedTasks[j].ID {
				sortedTasks[i], sortedTasks[j] = sortedTasks[j], sortedTasks[i]
			}
		}
	}

	// Assign sequential references
	for i, task := range sortedTasks {
		ref := fmt.Sprintf("ET%d", i+1)
		etf.referenceMapping[task.ID] = ref
		// Also update the task's ReferenceID
		if i < len(tasks) {
			for j := range tasks {
				if tasks[j].ID == task.ID {
					tasks[j].ReferenceID = ref
					break
				}
			}
		}
	}
}

// RegisterTask registers a task in the registry and returns its reference
func (etf *EvidenceTaskFormatter) RegisterTask(task *domain.EvidenceTask) string {
	if etf.registry == nil {
		// Fallback if no registry is set - use task ID with zero-padding
		ref := fmt.Sprintf("ET-%04d", task.ID)
		task.ReferenceID = ref
		return ref
	}

	// Get or create reference and update the task
	ref := etf.registry.RegisterTask(task)
	task.ReferenceID = ref
	return ref
}

// ToMarkdown converts an evidence task to markdown format
func (etf *EvidenceTaskFormatter) ToMarkdown(task *domain.EvidenceTask) string {
	var md strings.Builder

	// Header section with ID and basic info
	md.WriteString(fmt.Sprintf("# Evidence Task %d\n\n", task.ID))

	// Metadata header box
	md.WriteString("```\n")
	md.WriteString(fmt.Sprintf("Task ID: %d\n", task.ID))
	md.WriteString(fmt.Sprintf("Framework: %s\n", task.Framework))
	md.WriteString(fmt.Sprintf("Priority: %s\n", task.Priority))
	md.WriteString(fmt.Sprintf("Status: %s\n", task.Status))
	md.WriteString(fmt.Sprintf("Collection Interval: %s\n", task.CollectionInterval))
	if task.NextDue != nil && !task.NextDue.IsZero() {
		md.WriteString(fmt.Sprintf("Next Due: %s\n", task.NextDue.Format("2006-01-02")))
	}
	if task.LastCollected != nil && !task.LastCollected.IsZero() {
		md.WriteString(fmt.Sprintf("Last Collected: %s\n", task.LastCollected.Format("2006-01-02")))
	}
	md.WriteString("```\n\n")

	// Evidence task title
	if task.Name != "" {
		name := etf.baseFormatter.InterpolateText(task.Name)
		md.WriteString(fmt.Sprintf("## %s\n\n", name))
	}

	// Evidence task description
	if task.Description != "" {
		md.WriteString("### Description\n\n")
		description := etf.baseFormatter.CleanContent(task.Description)
		md.WriteString(fmt.Sprintf("%s\n\n", description))
	}

	// Evidence collection guidance
	if task.Guidance != "" {
		md.WriteString("### Collection Guidance\n\n")
		guidance := etf.baseFormatter.CleanContent(task.Guidance)
		md.WriteString(fmt.Sprintf("%s\n\n", guidance))
	}

	// Master content guidance (if different from main guidance)
	if task.MasterContent != nil && task.MasterContent.Guidance != "" &&
		task.MasterContent.Guidance != task.Guidance {
		md.WriteString("### Master Content Guidance\n\n")
		masterGuidance := etf.baseFormatter.CleanContent(task.MasterContent.Guidance)
		md.WriteString(fmt.Sprintf("%s\n\n", masterGuidance))
	}

	// Master content help
	if task.MasterContent != nil && task.MasterContent.Help != "" {
		md.WriteString("### Collection Help\n\n")
		help := etf.baseFormatter.CleanContent(task.MasterContent.Help)
		md.WriteString(fmt.Sprintf("%s\n\n", help))
	}

	// Collection requirements
	md.WriteString("### Collection Requirements\n\n")
	md.WriteString(fmt.Sprintf("**Collection Interval**: %s\n\n", task.CollectionInterval))
	if task.Priority != "" {
		md.WriteString(fmt.Sprintf("**Priority**: %s\n\n", task.Priority))
	}
	if task.AdHoc {
		md.WriteString("**Type**: Ad-hoc collection\n\n")
	} else {
		md.WriteString("**Type**: Scheduled collection\n\n")
	}
	if task.Sensitive {
		md.WriteString("⚠️ **Sensitive Data**: This evidence task involves sensitive data\n\n")
	}

	// Metadata section
	md.WriteString("---\n\n")
	md.WriteString("## Evidence Task Metadata\n\n")

	// Basic information
	md.WriteString("### Basic Information\n\n")
	basicRows := []MetadataRow{
		{"Task ID", strconv.Itoa(task.ID)},
		{"Name", etf.baseFormatter.InterpolateText(task.Name)},
		{"Framework", task.Framework},
		{"Priority", task.Priority},
		{"Status", task.Status},
		{"Collection Interval", task.CollectionInterval},
	}
	if task.AdHoc {
		basicRows = append(basicRows, MetadataRow{"Ad-hoc Collection", "Yes"})
	}
	if task.Sensitive {
		basicRows = append(basicRows, MetadataRow{"Sensitive Data", "Yes"})
	}
	if task.Completed {
		basicRows = append(basicRows, MetadataRow{"Completed", "Yes"})
	}
	md.WriteString(etf.baseFormatter.FormatMetadataTable(basicRows))

	// Collection timeline
	if (task.LastCollected != nil && !task.LastCollected.IsZero()) ||
		(task.NextDue != nil && !task.NextDue.IsZero()) {
		md.WriteString("\n### Collection Timeline\n\n")
		timelineRows := []MetadataRow{}
		if task.LastCollected != nil && !task.LastCollected.IsZero() {
			timelineRows = append(timelineRows, MetadataRow{"Last Collected", task.LastCollected.Format("2006-01-02")})
		}
		if task.NextDue != nil && !task.NextDue.IsZero() {
			timelineRows = append(timelineRows, MetadataRow{"Next Due", task.NextDue.Format("2006-01-02")})
		}
		if task.DueDaysBefore > 0 {
			timelineRows = append(timelineRows, MetadataRow{"Due Days Before", strconv.Itoa(task.DueDaysBefore)})
		}
		md.WriteString(etf.baseFormatter.FormatMetadataTable(timelineRows))
	}

	// Association counts
	if task.Associations != nil && (task.Associations.Controls > 0 ||
		task.Associations.Policies > 0 || task.Associations.Procedures > 0) {
		md.WriteString("\n### Associated Items\n\n")
		md.WriteString("| Type | Count |\n")
		md.WriteString("|------|-------|\n")
		if task.Associations.Controls > 0 {
			md.WriteString(fmt.Sprintf("| **Controls** | %d |\n", task.Associations.Controls))
		}
		if task.Associations.Policies > 0 {
			md.WriteString(fmt.Sprintf("| **Policies** | %d |\n", task.Associations.Policies))
		}
		if task.Associations.Procedures > 0 {
			md.WriteString(fmt.Sprintf("| **Procedures** | %d |\n", task.Associations.Procedures))
		}
	}

	// Automated Evidence Collection status
	if task.AecStatus != nil && task.AecStatus.Status != "" && task.AecStatus.Status != "na" {
		md.WriteString("\n### Automated Evidence Collection\n\n")
		md.WriteString("| Metric | Value |\n")
		md.WriteString("|--------|-------|\n")
		md.WriteString(fmt.Sprintf("| **Status** | %s |\n", task.AecStatus.Status))
		if task.AecStatus.LastExecuted != nil && !task.AecStatus.LastExecuted.IsZero() {
			md.WriteString(fmt.Sprintf("| **Last Executed** | %s |\n", task.AecStatus.LastExecuted.Format("2006-01-02 15:04")))
		}
		if task.AecStatus.NextScheduled != nil && !task.AecStatus.NextScheduled.IsZero() {
			md.WriteString(fmt.Sprintf("| **Next Scheduled** | %s |\n", task.AecStatus.NextScheduled.Format("2006-01-02 15:04")))
		}
		if task.AecStatus.SuccessfulRuns > 0 {
			md.WriteString(fmt.Sprintf("| **Successful Runs** | %d |\n", task.AecStatus.SuccessfulRuns))
		}
		if task.AecStatus.FailedRuns > 0 {
			md.WriteString(fmt.Sprintf("| **Failed Runs** | %d |\n", task.AecStatus.FailedRuns))
		}
		if task.AecStatus.ErrorMessage != "" {
			md.WriteString(fmt.Sprintf("| **Last Error** | %s |\n", task.AecStatus.ErrorMessage))
		}
	}

	// Subtask information
	if task.SubtaskMetadata != nil && task.SubtaskMetadata.TotalSubtasks > 0 {
		md.WriteString("\n### Subtask Information\n\n")
		md.WriteString("| Metric | Value |\n")
		md.WriteString("|--------|-------|\n")
		md.WriteString(fmt.Sprintf("| **Total Subtasks** | %d |\n", task.SubtaskMetadata.TotalSubtasks))
		md.WriteString(fmt.Sprintf("| **Completed Subtasks** | %d |\n", task.SubtaskMetadata.CompletedSubtasks))
		md.WriteString(fmt.Sprintf("| **Pending Subtasks** | %d |\n", task.SubtaskMetadata.PendingSubtasks))
		if task.SubtaskMetadata.OverdueSubtasks > 0 {
			md.WriteString(fmt.Sprintf("| **Overdue Subtasks** | %d |\n", task.SubtaskMetadata.OverdueSubtasks))
		}
	}

	// Usage statistics
	usageStats := UsageStats{
		ViewCount:        task.ViewCount,
		LastViewedAt:     getTimeValue(task.LastViewedAt),
		DownloadCount:    task.DownloadCount,
		LastDownloadedAt: getTimeValue(task.LastDownloadedAt),
		ReferenceCount:   task.ReferenceCount,
		LastReferencedAt: getTimeValue(task.LastReferencedAt),
	}
	md.WriteString(etf.baseFormatter.FormatUsageStatistics(usageStats))

	// Assignees
	if len(task.Assignees) > 0 {
		people := make([]PersonInfo, 0, len(task.Assignees))
		for _, assignee := range task.Assignees {
			if assignee.Name != "" || assignee.Email != "" {
				people = append(people, PersonInfo{
					Name:       assignee.Name,
					Email:      assignee.Email,
					AssignedAt: getTimeValue(assignee.AssignedAt),
				})
			}
		}
		if len(people) > 0 {
			md.WriteString(etf.baseFormatter.FormatPersonList(people, "Assignees"))
		}
	}

	// Audit projects
	if len(task.AuditProjects) > 0 {
		md.WriteString("\n### Audit Projects\n\n")
		for _, project := range task.AuditProjects {
			if project.Name != "" {
				md.WriteString(fmt.Sprintf("- **%s** (Status: %s)", project.Name, project.Status))
				if project.Description != "" {
					md.WriteString(fmt.Sprintf(" - %s", project.Description))
				}
				md.WriteString("\n")
			}
		}
	}

	// Framework codes
	if len(task.FrameworkCodes) > 0 {
		md.WriteString("\n### Framework Codes\n\n")
		for _, code := range task.FrameworkCodes {
			if code.Framework != "" || code.Code != "" {
				md.WriteString(fmt.Sprintf("- **%s**: %s", code.Framework, code.Code))
				if code.Name != "" {
					md.WriteString(fmt.Sprintf(" - %s", code.Name))
				}
				md.WriteString("\n")
			}
		}
	}

	// Supported integrations
	if len(task.SupportedIntegrations) > 0 {
		md.WriteString("\n### Supported Integrations\n\n")
		for _, integration := range task.SupportedIntegrations {
			if integration.Name != "" {
				md.WriteString(fmt.Sprintf("- **%s** (%s)", integration.Name, integration.Type))
				if integration.Description != "" {
					md.WriteString(fmt.Sprintf(" - %s", integration.Description))
				}
				if integration.Enabled {
					md.WriteString(" ✅")
				} else {
					md.WriteString(" ❌")
				}
				md.WriteString("\n")
			}
		}
	}

	// Tags
	if len(task.Tags) > 0 {
		tags := make([]TagInfo, 0, len(task.Tags))
		for _, tag := range task.Tags {
			if tag.Name != "" {
				tags = append(tags, TagInfo{
					Name:  tag.Name,
					Color: tag.Color,
				})
			}
		}
		if len(tags) > 0 {
			md.WriteString(etf.baseFormatter.FormatTagList(tags))
		}
	}

	// Related controls - enhanced with control details if available
	if len(task.Controls) > 0 || len(task.RelatedControls) > 0 {
		md.WriteString("\n### Related Controls\n\n")
		md.WriteString("This evidence task is associated with the following controls:\n\n")

		// First, use embedded RelatedControls if available (preferred)
		if len(task.RelatedControls) > 0 {
			for _, control := range task.RelatedControls {
				// Show detailed control information from embedded data
				refID := control.ReferenceID
				if refID == "" {
					refID = fmt.Sprintf("C%d", control.ID)
				}
				md.WriteString(fmt.Sprintf("#### %s - %s\n\n", refID, control.Name))
				if control.Description != "" {
					md.WriteString(fmt.Sprintf("**Description**: %s\n\n", etf.baseFormatter.CleanContent(control.Description)))
				}
				if control.Category != "" {
					md.WriteString(fmt.Sprintf("**Category**: %s | ", control.Category))
				}
				if control.Status != "" {
					md.WriteString(fmt.Sprintf("**Status**: %s\n\n", control.Status))
				}
			}
		} else {
			// Fallback to using controlIDs with SetRelatedControls data or basic IDs
			for _, controlID := range task.Controls {
				if control, exists := etf.relatedControls[controlID]; exists {
					// Show detailed control information from SetRelatedControls
					md.WriteString(fmt.Sprintf("#### %s - %s\n\n", control.ReferenceID, control.Name))
					if control.Description != "" {
						md.WriteString(fmt.Sprintf("**Description**: %s\n\n", etf.baseFormatter.CleanContent(control.Description)))
					}
					if control.Category != "" {
						md.WriteString(fmt.Sprintf("**Category**: %s | ", control.Category))
					}
					if control.Status != "" {
						md.WriteString(fmt.Sprintf("**Status**: %s\n\n", control.Status))
					}
				} else {
					// Fallback to just showing the ID
					md.WriteString(fmt.Sprintf("- Control ID: %s\n", controlID))
				}
			}
		}
	}

	// JIRA issues
	if len(task.JiraIssues) > 0 {
		md.WriteString("\n### JIRA Issues\n\n")
		for _, issue := range task.JiraIssues {
			md.WriteString(fmt.Sprintf("- **%s**: %s", issue.Key, issue.Summary))
			if issue.Status != "" {
				md.WriteString(fmt.Sprintf(" (Status: %s)", issue.Status))
			}
			md.WriteString("\n")
		}
	}

	// Other metrics
	if task.OpenIncidentCount > 0 {
		md.WriteString("\n### Additional Metrics\n\n")
		md.WriteString("| Metric | Value |\n")
		md.WriteString("|--------|-------|\n")
		md.WriteString(fmt.Sprintf("| **Open Incidents** | %d |\n", task.OpenIncidentCount))
	}

	// Deprecation notes
	if task.DeprecationNotes != "" {
		md.WriteString("\n### ⚠️ Deprecation Notice\n\n")
		md.WriteString(fmt.Sprintf("> %s\n", task.DeprecationNotes))
	}

	// Footer
	md.WriteString(fmt.Sprintf("\n---\n\n*Generated on %s*\n", time.Now().Format("2006-01-02 15:04:05 MST")))

	return md.String()
}

// ToSummaryMarkdown creates a brief markdown summary of the evidence task
func (etf *EvidenceTaskFormatter) ToSummaryMarkdown(task *domain.EvidenceTask) string {
	var md strings.Builder

	md.WriteString(fmt.Sprintf("## %s\n\n", task.Name))
	md.WriteString(fmt.Sprintf("**ID:** %d | **Priority:** %s | **Status:** %s\n\n",
		task.ID, task.Priority, task.Status))

	if task.Description != "" {
		// Use first paragraph of description
		desc := strings.Split(task.Description, "\n")[0]
		if len(desc) > 200 {
			desc = desc[:197] + "..."
		}
		md.WriteString(fmt.Sprintf("%s\n\n", desc))
	}

	// Collection information
	md.WriteString(fmt.Sprintf("**Collection Interval**: %s", task.CollectionInterval))
	if task.NextDue != nil && !task.NextDue.IsZero() {
		md.WriteString(fmt.Sprintf(" | **Next Due**: %s", task.NextDue.Format("2006-01-02")))
	}
	md.WriteString("\n\n")

	// Quick stats
	var stats []string
	if task.Associations != nil {
		if task.Associations.Controls > 0 {
			stats = append(stats, fmt.Sprintf("%d controls", task.Associations.Controls))
		}
		if task.Associations.Policies > 0 {
			stats = append(stats, fmt.Sprintf("%d policies", task.Associations.Policies))
		}
	}
	if len(stats) > 0 {
		md.WriteString("**Associated:** ")
		md.WriteString(strings.Join(stats, ", "))
		md.WriteString("\n\n")
	}

	// Collection status
	if task.LastCollected != nil && !task.LastCollected.IsZero() {
		md.WriteString(fmt.Sprintf("*Last collected: %s*", task.LastCollected.Format("2006-01-02")))
		if task.Completed {
			md.WriteString(" | *Status: Completed*")
		}
		md.WriteString("\n")
	}

	return md.String()
}

// ToDocumentMarkdown creates a comprehensive evidence task document in markdown format
// This is the main method for generating evidence task documents that will be saved to files
func (etf *EvidenceTaskFormatter) ToDocumentMarkdown(task *domain.EvidenceTask) string {
	var md strings.Builder

	// Generate reference ID for the evidence task
	reference := etf.generateReference(task)

	// Document title with reference
	interpolatedName := etf.baseFormatter.InterpolateText(task.Name)
	md.WriteString(fmt.Sprintf("# %s - %s\n\n", reference, interpolatedName))

	// Evidence task description
	if task.Description != "" {
		md.WriteString("## Description\n\n")
		md.WriteString(fmt.Sprintf("%s\n\n", etf.baseFormatter.CleanContent(task.Description)))
	}

	// Collection guidance
	if task.Guidance != "" {
		md.WriteString("## Collection Guidance\n\n")
		guidance := etf.baseFormatter.CleanContent(task.Guidance)
		md.WriteString(fmt.Sprintf("%s\n\n", guidance))
	}

	// Master content guidance (if different from main guidance)
	if task.MasterContent != nil && task.MasterContent.Guidance != "" &&
		task.MasterContent.Guidance != task.Guidance {
		md.WriteString("## Master Content Guidance\n\n")
		masterGuidance := etf.baseFormatter.CleanContent(task.MasterContent.Guidance)
		md.WriteString(fmt.Sprintf("%s\n\n", masterGuidance))
	}

	// Collection help
	if task.MasterContent != nil && task.MasterContent.Help != "" {
		md.WriteString("## Collection Help\n\n")
		help := etf.baseFormatter.CleanContent(task.MasterContent.Help)
		md.WriteString(fmt.Sprintf("%s\n\n", help))
	}

	// Collection requirements
	md.WriteString("## Collection Requirements\n\n")
	md.WriteString(fmt.Sprintf("**Collection Interval**: %s\n\n", task.CollectionInterval))
	if task.Priority != "" {
		md.WriteString(fmt.Sprintf("**Priority**: %s\n\n", task.Priority))
	}
	if task.AdHoc {
		md.WriteString("**Type**: Ad-hoc collection\n\n")
	} else {
		md.WriteString("**Type**: Scheduled collection\n\n")
	}
	if task.Sensitive {
		md.WriteString("⚠️ **Sensitive Data**: This evidence task involves sensitive data. Handle with appropriate care.\n\n")
	}

	// Collection strategy (if applicable)
	if len(task.SupportedIntegrations) > 0 {
		md.WriteString("## Collection Strategy\n\n")
		md.WriteString("This evidence task supports the following automated collection methods:\n\n")
		for _, integration := range task.SupportedIntegrations {
			if integration.Name != "" {
				md.WriteString(fmt.Sprintf("- **%s** (%s)", integration.Name, integration.Type))
				if integration.Description != "" {
					md.WriteString(fmt.Sprintf(": %s", integration.Description))
				}
				if integration.Enabled {
					md.WriteString(" ✅ Enabled")
				} else {
					md.WriteString(" ❌ Not enabled")
				}
				md.WriteString("\n")
			}
		}
		md.WriteString("\n")
	}

	// Document metadata footer
	md.WriteString("---\n\n")
	md.WriteString("## Document Information\n\n")

	// Basic metadata table
	metadataRows := []MetadataRow{
		{"Document Reference", reference},
		{"Task ID", strconv.Itoa(task.ID)},
		{"Task Name", etf.baseFormatter.InterpolateText(task.Name)},
		{"Framework", task.Framework},
		{"Priority", task.Priority},
		{"Status", task.Status},
		{"Collection Interval", task.CollectionInterval},
	}
	if task.AdHoc {
		metadataRows = append(metadataRows, MetadataRow{"Ad-hoc Collection", "Yes"})
	}
	if task.Sensitive {
		metadataRows = append(metadataRows, MetadataRow{"Sensitive Data", "Yes"})
	}
	if task.LastCollected != nil && !task.LastCollected.IsZero() {
		metadataRows = append(metadataRows, MetadataRow{"Last Collected", task.LastCollected.Format("2006-01-02")})
	}
	if task.NextDue != nil && !task.NextDue.IsZero() {
		metadataRows = append(metadataRows, MetadataRow{"Next Due", task.NextDue.Format("2006-01-02")})
	}
	if task.CreatedAt.Year() > 1 {
		metadataRows = append(metadataRows, MetadataRow{"Created Date", task.CreatedAt.Format("2006-01-02")})
	}
	if task.UpdatedAt.Year() > 1 {
		metadataRows = append(metadataRows, MetadataRow{"Last Updated", task.UpdatedAt.Format("2006-01-02")})
	}

	md.WriteString(etf.baseFormatter.FormatMetadataTable(metadataRows))

	// Association information
	if task.Associations != nil && (task.Associations.Controls > 0 ||
		task.Associations.Policies > 0 || task.Associations.Procedures > 0) {
		md.WriteString("\n### Associated Items\n\n")
		md.WriteString("| Type | Count |\n")
		md.WriteString("|------|-------|\n")
		if task.Associations.Controls > 0 {
			md.WriteString(fmt.Sprintf("| **Associated Controls** | %d |\n", task.Associations.Controls))
		}
		if task.Associations.Policies > 0 {
			md.WriteString(fmt.Sprintf("| **Associated Policies** | %d |\n", task.Associations.Policies))
		}
		if task.Associations.Procedures > 0 {
			md.WriteString(fmt.Sprintf("| **Associated Procedures** | %d |\n", task.Associations.Procedures))
		}
	}

	// Related controls (specific control IDs) - enhanced with control details if available
	if len(task.Controls) > 0 || len(task.RelatedControls) > 0 {
		md.WriteString("\n### Related Controls\n\n")
		md.WriteString("This evidence task supports the implementation and monitoring of the following security controls:\n\n")

		// First, use embedded RelatedControls if available (preferred)
		if len(task.RelatedControls) > 0 {
			for _, control := range task.RelatedControls {
				// Show comprehensive control information from embedded data
				refID := control.ReferenceID
				if refID == "" {
					refID = fmt.Sprintf("C%d", control.ID)
				}
				md.WriteString(fmt.Sprintf("#### %s - %s\n\n", refID, control.Name))

				if control.Description != "" {
					md.WriteString(fmt.Sprintf("**Description**: %s\n\n", etf.baseFormatter.CleanContent(control.Description)))
				}

				// Control metadata
				md.WriteString("**Control Details**:\n")
				if control.Category != "" {
					md.WriteString(fmt.Sprintf("- **Category**: %s\n", control.Category))
				}
				if control.Status != "" {
					md.WriteString(fmt.Sprintf("- **Implementation Status**: %s\n", control.Status))
				}
				if control.Framework != "" {
					md.WriteString(fmt.Sprintf("- **Framework**: %s\n", control.Framework))
				}

				// Control guidance if available
				if control.MasterContent != nil && control.MasterContent.Guidance != "" {
					md.WriteString(fmt.Sprintf("\n**Implementation Guidance**: %s\n", etf.baseFormatter.CleanContent(control.MasterContent.Guidance)))
				}

				md.WriteString("\n")
			}
		} else {
			// Fallback to using controlIDs with SetRelatedControls data or basic IDs
			for _, controlID := range task.Controls {
				if control, exists := etf.relatedControls[controlID]; exists {
					// Show comprehensive control information from SetRelatedControls
					md.WriteString(fmt.Sprintf("#### %s - %s\n\n", control.ReferenceID, control.Name))

					if control.Description != "" {
						md.WriteString(fmt.Sprintf("**Description**: %s\n\n", etf.baseFormatter.CleanContent(control.Description)))
					}

					// Control metadata
					md.WriteString("**Control Details**:\n")
					if control.Category != "" {
						md.WriteString(fmt.Sprintf("- **Category**: %s\n", control.Category))
					}
					if control.Status != "" {
						md.WriteString(fmt.Sprintf("- **Implementation Status**: %s\n", control.Status))
					}
					if control.Framework != "" {
						md.WriteString(fmt.Sprintf("- **Framework**: %s\n", control.Framework))
					}

					// Control guidance if available
					if control.MasterContent != nil && control.MasterContent.Guidance != "" {
						md.WriteString(fmt.Sprintf("\n**Implementation Guidance**: %s\n", etf.baseFormatter.CleanContent(control.MasterContent.Guidance)))
					}

					md.WriteString("\n")
				} else {
					// Fallback to just showing the ID
					md.WriteString(fmt.Sprintf("- Control ID: %s *(Details not available)*\n", controlID))
				}
			}
		}
	}

	// Assignees
	if len(task.Assignees) > 0 {
		people := make([]PersonInfo, 0, len(task.Assignees))
		for _, assignee := range task.Assignees {
			if assignee.Name != "" || assignee.Email != "" {
				people = append(people, PersonInfo{
					Name:       assignee.Name,
					Email:      assignee.Email,
					AssignedAt: getTimeValue(assignee.AssignedAt),
				})
			}
		}
		if len(people) > 0 {
			md.WriteString(etf.baseFormatter.FormatPersonList(people, "Assignees"))
		}
	}

	// Audit projects
	if len(task.AuditProjects) > 0 {
		md.WriteString("\n### Audit Projects\n\n")
		for _, project := range task.AuditProjects {
			if project.Name != "" {
				md.WriteString(fmt.Sprintf("- **%s** (Status: %s)", project.Name, project.Status))
				if project.Description != "" {
					md.WriteString(fmt.Sprintf(" - %s", project.Description))
				}
				md.WriteString("\n")
			}
		}
	}

	// Framework codes
	if len(task.FrameworkCodes) > 0 {
		md.WriteString("\n### Framework Codes\n\n")
		for _, code := range task.FrameworkCodes {
			if code.Framework != "" || code.Code != "" {
				md.WriteString(fmt.Sprintf("- **%s**: %s", code.Framework, code.Code))
				if code.Name != "" {
					md.WriteString(fmt.Sprintf(" - %s", code.Name))
				}
				md.WriteString("\n")
			}
		}
	}

	// Tags
	if len(task.Tags) > 0 {
		tags := make([]TagInfo, 0, len(task.Tags))
		for _, tag := range task.Tags {
			if tag.Name != "" {
				tags = append(tags, TagInfo{
					Name:  tag.Name,
					Color: tag.Color,
				})
			}
		}
		if len(tags) > 0 {
			md.WriteString(etf.baseFormatter.FormatTagList(tags))
		}
	}

	// Automated Evidence Collection status
	if task.AecStatus != nil && task.AecStatus.Status != "" && task.AecStatus.Status != "na" {
		md.WriteString("\n### Automated Evidence Collection\n\n")
		md.WriteString("| Metric | Value |\n")
		md.WriteString("|--------|-------|\n")
		md.WriteString(fmt.Sprintf("| **Status** | %s |\n", task.AecStatus.Status))
		if task.AecStatus.LastExecuted != nil && !task.AecStatus.LastExecuted.IsZero() {
			md.WriteString(fmt.Sprintf("| **Last Executed** | %s |\n", task.AecStatus.LastExecuted.Format("2006-01-02 15:04")))
		}
		if task.AecStatus.NextScheduled != nil && !task.AecStatus.NextScheduled.IsZero() {
			md.WriteString(fmt.Sprintf("| **Next Scheduled** | %s |\n", task.AecStatus.NextScheduled.Format("2006-01-02 15:04")))
		}
		if task.AecStatus.SuccessfulRuns > 0 {
			md.WriteString(fmt.Sprintf("| **Successful Runs** | %d |\n", task.AecStatus.SuccessfulRuns))
		}
		if task.AecStatus.FailedRuns > 0 {
			md.WriteString(fmt.Sprintf("| **Failed Runs** | %d |\n", task.AecStatus.FailedRuns))
		}
		if task.AecStatus.ErrorMessage != "" {
			md.WriteString(fmt.Sprintf("| **Last Error** | %s |\n", task.AecStatus.ErrorMessage))
		}
	}

	// Usage statistics (if available)
	usageStats := UsageStats{
		ViewCount:        task.ViewCount,
		LastViewedAt:     getTimeValue(task.LastViewedAt),
		DownloadCount:    task.DownloadCount,
		LastDownloadedAt: getTimeValue(task.LastDownloadedAt),
		ReferenceCount:   task.ReferenceCount,
		LastReferencedAt: getTimeValue(task.LastReferencedAt),
	}
	md.WriteString(etf.baseFormatter.FormatUsageStatistics(usageStats))

	// Deprecation notice
	if task.DeprecationNotes != "" {
		md.WriteString("\n### ⚠️ Deprecation Notice\n\n")
		md.WriteString(fmt.Sprintf("> %s\n", task.DeprecationNotes))
	}

	// Document footer
	md.WriteString(etf.baseFormatter.GenerateDocumentFooter())

	return md.String()
}

// ToDocumentMarkdownWithContext creates a comprehensive evidence task document with control/policy context
func (etf *EvidenceTaskFormatter) ToDocumentMarkdownWithContext(task *domain.EvidenceTask, controls []domain.Control, policies []domain.Policy) string {
	// Generate base document content
	baseContent := etf.ToDocumentMarkdown(task)

	// If we have control or policy context, enhance the document
	if len(controls) > 0 || len(policies) > 0 {
		return etf.enhanceDocumentWithContext(baseContent, task, controls, policies)
	}

	return baseContent
}

// enhanceDocumentWithContext adds detailed control and policy information to the document
func (etf *EvidenceTaskFormatter) enhanceDocumentWithContext(baseContent string, task *domain.EvidenceTask, controls []domain.Control, policies []domain.Policy) string {
	var enhanced strings.Builder

	// Split base content to insert enhanced sections before the metadata
	parts := strings.Split(baseContent, "---\n\n## Document Information")
	if len(parts) != 2 {
		// If we can't find the expected split point, return original content
		return baseContent
	}

	// Add the first part (everything before metadata)
	enhanced.WriteString(parts[0])

	// Add enhanced control and policy sections
	etf.addEnhancedControlSection(&enhanced, task, controls)
	etf.addEnhancedPolicySection(&enhanced, task, policies)

	// Add the metadata section back
	enhanced.WriteString("---\n\n## Document Information")
	enhanced.WriteString(parts[1])

	return enhanced.String()
}

// addEnhancedControlSection adds detailed control information
func (etf *EvidenceTaskFormatter) addEnhancedControlSection(md *strings.Builder, task *domain.EvidenceTask, controls []domain.Control) {
	if (len(task.Controls) == 0 && len(task.RelatedControls) == 0) || (len(controls) == 0 && len(task.RelatedControls) == 0) {
		return
	}

	md.WriteString("## Related Controls\n\n")
	md.WriteString("This evidence task supports the following security controls:\n\n")

	// First, use embedded RelatedControls if available (preferred)
	if len(task.RelatedControls) > 0 {
		for _, control := range task.RelatedControls {
			refID := control.ReferenceID
			if refID == "" {
				refID = fmt.Sprintf("C%d", control.ID)
			}
			fmt.Fprintf(md, "### %s - %s\n\n", refID, control.Name)
			if control.Description != "" {
				cleanDescription := etf.baseFormatter.CleanContent(control.Description)
				fmt.Fprintf(md, "**Description**: %s\n\n", cleanDescription)
			}
			if control.Category != "" {
				fmt.Fprintf(md, "**Category**: %s\n\n", control.Category)
			}
			if control.Status != "" {
				fmt.Fprintf(md, "**Status**: %s\n\n", control.Status)
			}
		}
	} else if len(controls) > 0 {
		// Fallback to using provided controls parameter with task.Controls IDs
		// Create a map for quick control lookup by ID
		controlMap := make(map[string]domain.Control)
		for _, control := range controls {
			controlMap[fmt.Sprintf("%d", control.ID)] = control
		}

		for _, controlID := range task.Controls {
			if control, exists := controlMap[controlID]; exists {
				fmt.Fprintf(md, "### %s - %s\n\n", control.ReferenceID, control.Name)
				if control.Description != "" {
					cleanDescription := etf.baseFormatter.CleanContent(control.Description)
					fmt.Fprintf(md, "**Description**: %s\n\n", cleanDescription)
				}
				if control.Category != "" {
					fmt.Fprintf(md, "**Category**: %s\n\n", control.Category)
				}
				if control.Status != "" {
					fmt.Fprintf(md, "**Status**: %s\n\n", control.Status)
				}
			} else {
				fmt.Fprintf(md, "### Control ID: %s\n\n", controlID)
				md.WriteString("*Control details not available*\n\n")
			}
		}
	}
}

// addEnhancedPolicySection adds related policy information based on framework and keywords
func (etf *EvidenceTaskFormatter) addEnhancedPolicySection(md *strings.Builder, task *domain.EvidenceTask, policies []domain.Policy) {
	if len(policies) == 0 {
		return
	}

	// Find related policies by framework match and keyword analysis
	var relatedPolicies []domain.Policy

	// First, match by framework
	if task.Framework != "" {
		for _, policy := range policies {
			if policy.Framework == task.Framework {
				relatedPolicies = append(relatedPolicies, policy)
			}
		}
	}

	// If no framework matches, try keyword matching for access control tasks
	if len(relatedPolicies) == 0 {
		taskNameLower := strings.ToLower(task.Name)
		taskDescLower := strings.ToLower(task.Description)

		for _, policy := range policies {
			policyNameLower := strings.ToLower(policy.Name)

			// Match access control related tasks to access control policies
			if (strings.Contains(taskNameLower, "access") || strings.Contains(taskDescLower, "access")) &&
				strings.Contains(policyNameLower, "access") {
				relatedPolicies = append(relatedPolicies, policy)
			}

			// Match by other common keywords
			keywords := []string{"security", "data", "incident", "change", "backup"}
			for _, keyword := range keywords {
				if (strings.Contains(taskNameLower, keyword) || strings.Contains(taskDescLower, keyword)) &&
					strings.Contains(policyNameLower, keyword) {
					// Check if already added
					found := false
					for _, existing := range relatedPolicies {
						if existing.ID == policy.ID {
							found = true
							break
						}
					}
					if !found {
						relatedPolicies = append(relatedPolicies, policy)
					}
				}
			}
		}
	}

	if len(relatedPolicies) > 0 {
		md.WriteString("## Related Policies\n\n")
		md.WriteString("This evidence task relates to the following organizational policies:\n\n")

		for _, policy := range relatedPolicies {
			fmt.Fprintf(md, "### %s\n\n", policy.Name)
			if policy.Description != "" {
				cleanDescription := etf.baseFormatter.CleanContent(policy.Description)
				fmt.Fprintf(md, "**Description**: %s\n\n", cleanDescription)
			}
			if policy.Status != "" {
				fmt.Fprintf(md, "**Status**: %s\n\n", policy.Status)
			}
		}
	}
}

// generateReference creates a reference ID for an evidence task using the registry
func (etf *EvidenceTaskFormatter) generateReference(task *domain.EvidenceTask) string {
	// If task already has a reference ID, use it
	if task.ReferenceID != "" {
		return task.ReferenceID
	}

	// Check if we have a reference mapping (for tests)
	if etf.referenceMapping != nil {
		if ref, exists := etf.referenceMapping[task.ID]; exists {
			task.ReferenceID = ref
			return ref
		}
	}

	if etf.registry == nil {
		// Fallback if no registry is set
		ref := fmt.Sprintf("ET%d", task.ID)
		task.ReferenceID = ref
		return ref
	}

	// Get reference from registry, register if not found
	if ref, exists := etf.registry.GetReference(task.ID); exists {
		// Update the task's reference ID field
		task.ReferenceID = ref
		return ref
	}

	// Register the task and return its new reference
	ref := etf.registry.RegisterTask(task)
	task.ReferenceID = ref
	return ref
}

// GetDocumentFilename generates a filename for an evidence task document using unified pattern
func (etf *EvidenceTaskFormatter) GetDocumentFilename(task *domain.EvidenceTask) string {
	fg := utils.NewFilenameGenerator()
	return fg.GenerateFilename(task.ReferenceID, strconv.Itoa(task.ID), task.Name, "md")
}
