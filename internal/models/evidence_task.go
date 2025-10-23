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

// EvidenceTask represents an evidence collection task from Tugboat Logic
type EvidenceTask struct {
	ID                 int       `json:"id"`
	Name               string    `json:"name"`
	Description        string    `json:"description"`
	Guidance           string    `json:"guidance"`
	AdHoc              bool      `json:"ad_hoc"`
	LastCollected      *string   `json:"last_collected,omitempty"`
	CollectionInterval string    `json:"collection_interval"`
	DueDaysBefore      int       `json:"due_days_before_end"`
	MasterVersionNum   int       `json:"master_version_num"`
	Sensitive          bool      `json:"sensitive"`
	OrgID              int       `json:"org_id"`
	Completed          bool      `json:"completed"`
	MasterEvidenceID   int       `json:"master_evidence_id"`
	Created            time.Time `json:"created"`
	EntityType         string    `json:"__entity_type__"`
	EntityRole         *string   `json:"__entity_role__"`
	Permissions        []string  `json:"__permissions__"`

	// Additional fields for compatibility
	PolicyID     string            `json:"policy_id,omitempty"`
	ProcedureID  string            `json:"procedure_id,omitempty"`
	ControlID    string            `json:"control_id,omitempty"`
	Status       string            `json:"status,omitempty"` // Derived from Completed field
	Priority     string            `json:"priority,omitempty"`
	Framework    string            `json:"framework,omitempty"`
	ReferenceID  string            `json:"reference_id,omitempty"` // ET1, ET2, etc.
	DueDate      *time.Time        `json:"due_date,omitempty"`
	AssignedTo   string            `json:"assigned_to,omitempty"`
	Requirements map[string]string `json:"requirements,omitempty"`
	Evidence     []Evidence        `json:"evidence,omitempty"`
	CreatedAt    time.Time         `json:"created_at,omitempty"`
	UpdatedAt    time.Time         `json:"updated_at,omitempty"`
}

// EvidenceTaskSummary represents a summary view of evidence tasks
type EvidenceTaskSummary struct {
	Total      int            `json:"total"`
	ByStatus   map[string]int `json:"by_status"`
	ByPriority map[string]int `json:"by_priority"`
	Overdue    int            `json:"overdue"`
	DueSoon    int            `json:"due_soon"` // Due within 7 days
	LastSync   time.Time      `json:"last_sync"`
}

// EvidenceFilter represents filters for evidence task queries
type EvidenceFilter struct {
	Status     []string   `json:"status,omitempty"`
	Priority   []string   `json:"priority,omitempty"`
	Framework  string     `json:"framework,omitempty"`
	AssignedTo string     `json:"assigned_to,omitempty"`
	DueBefore  *time.Time `json:"due_before,omitempty"`
	DueAfter   *time.Time `json:"due_after,omitempty"`
}

// EvidenceTaskDetails represents detailed evidence task information with embeds
type EvidenceTaskDetails struct {
	EvidenceTask                                 // Embed the basic evidence task
	MasterContent         *MasterContent         `json:"master_content,omitempty"`
	OrgScope              *OrgScope              `json:"org_scope,omitempty"`
	Tags                  []EvidenceTag          `json:"tags,omitempty"`
	OpenIncidentCount     int                    `json:"open_incident_count,omitempty"`
	JiraIssues            []JiraIssue            `json:"jira_issues,omitempty"`
	Assignees             []EvidenceAssignee     `json:"assignees,omitempty"`
	SupportedIntegrations []SupportedIntegration `json:"supported_integrations,omitempty"`
	AecStatus             *AecStatus             `json:"aec_status,omitempty"`
	SubtaskMetadata       *SubtaskMetadata       `json:"subtask_metadata,omitempty"`
	LastRemindedAt        *time.Time             `json:"last_reminded_at,omitempty"`
	SupportedInternalAec  []InternalAec          `json:"supported_internal_aec,omitempty"`
}

// EvidenceTag represents a tag associated with an evidence task
type EvidenceTag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// EvidenceAssignee represents someone assigned to an evidence task
type EvidenceAssignee struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role,omitempty"`
}

// SupportedIntegration represents an integration supported for evidence collection
type SupportedIntegration struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// AecStatus represents the Automated Evidence Collection status
type AecStatus struct {
	ID                string     `json:"id"`
	Status            string     `json:"status"`
	LastExecuted      *time.Time `json:"last_executed,omitempty"`
	NextScheduled     *time.Time `json:"next_scheduled,omitempty"`
	ErrorMessage      string     `json:"error_message,omitempty"`
	SuccessfulRuns    int        `json:"successful_runs"`
	FailedRuns        int        `json:"failed_runs"`
	LastSuccessfulRun *time.Time `json:"last_successful_run,omitempty"`
}

// SubtaskMetadata represents metadata about subtasks
type SubtaskMetadata struct {
	TotalSubtasks     int `json:"total_subtasks"`
	CompletedSubtasks int `json:"completed_subtasks"`
	PendingSubtasks   int `json:"pending_subtasks"`
	OverdueSubtasks   int `json:"overdue_subtasks"`
}

// InternalAec represents internal automated evidence collection configurations
type InternalAec struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Enabled       bool                   `json:"enabled"`
	LastExecuted  *time.Time             `json:"last_executed,omitempty"`
	Schedule      string                 `json:"schedule,omitempty"`
}

// GetStatus returns the status, deriving from completed field if not set
func (et *EvidenceTask) GetStatus() string {
	if et.Status != "" {
		return et.Status
	}
	if et.Completed {
		return "completed"
	}
	return "pending"
}

// GetPriority returns the priority, deriving from collection_interval if not set
func (et *EvidenceTask) GetPriority() string {
	if et.Priority != "" {
		return et.Priority
	}
	switch et.CollectionInterval {
	case "year":
		return "low"
	case "quarter":
		return "medium"
	case "month", "week":
		return "high"
	default:
		return "medium"
	}
}

// GetFramework returns the framework if set
func (et *EvidenceTask) GetFramework() string {
	return et.Framework
}
