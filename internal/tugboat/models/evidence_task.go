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

// EvidenceTask represents an evidence collection task from Tugboat Logic API
type EvidenceTask struct {
	ID                 int         `json:"id"`
	Name               string      `json:"name"`
	Description        string      `json:"description"`
	CollectionInterval string      `json:"collection_interval"` // year, quarter, month, etc.
	Completed          bool        `json:"completed"`
	LastCollected      *string     `json:"last_collected,omitempty"` // Can be null or ISO date string
	NextDue            *string     `json:"next_due,omitempty"`       // Can be null or ISO date string
	Priority           string      `json:"priority"`
	Framework          string      `json:"framework"`
	Controls           []string    `json:"controls"`  // List of control IDs
	Assignees          interface{} `json:"assignees"` // Can be []string (IDs) or []EvidenceAssignee (with embeds)
	Tags               interface{} `json:"tags"`      // Can be []string (names) or []EvidenceTag (with embeds)
	Status             string      `json:"status"`    // pending, in_progress, completed, etc.
	CreatedAt          string      `json:"created_at"`
	UpdatedAt          string      `json:"updated_at"`
	AecStatus          interface{} `json:"aec_status,omitempty"` // Can be null, string, or AecStatus object (with embeds)
}

// EvidenceTaskDetails represents detailed evidence task information with embeds from Tugboat Logic API
type EvidenceTaskDetails struct {
	EvidenceTask                                 // Embed the basic evidence task
	MasterContent         *MasterContent         `json:"master_content,omitempty"`
	OrgScope              *OrgScope              `json:"org_scope,omitempty"`
	Tags                  []EvidenceTag          `json:"tags,omitempty"`
	OpenIncidentCount     int                    `json:"open_incident_count,omitempty"`
	JiraIssues            []JiraIssue            `json:"jira_issues,omitempty"`
	Assignees             []EvidenceAssignee     `json:"assignees,omitempty"`
	SupportedIntegrations []SupportedIntegration `json:"supported_integrations,omitempty"`
	AecStatus             interface{}            `json:"aec_status,omitempty"` // Can be string or AecStatus object
	SubtaskMetadata       *SubtaskMetadata       `json:"subtask_metadata,omitempty"`
	LastRemindedAt        *FlexibleTime          `json:"last_reminded_at,omitempty"`
	SupportedInternalAec  []InternalAec          `json:"supported_internal_aec,omitempty"`
	Controls              interface{}            `json:"controls,omitempty"`            // Can be null, array of objects
	AuditProjects         interface{}            `json:"audit_projects,omitempty"`      // Can be null, array of objects
	Usage                 interface{}            `json:"usage,omitempty"`               // Can be null or object
	FrameworkCodes        interface{}            `json:"framework_codes,omitempty"`     // Can be null, array of objects
	Associations          interface{}            `json:"associations,omitempty"`        // Can be null, array, or object
	ParentOrgControls     []Control              `json:"parent_org_controls,omitempty"` // Related controls from embeds=org_controls
	// API metadata fields
	EntityRole  interface{} `json:"__entity_role__"`
	EntityType  string      `json:"__entity_type__"`
	Permissions []string    `json:"__permissions__"`
}

// EvidenceAssignee represents a user assigned to an evidence task
type EvidenceAssignee struct {
	ID         interface{}  `json:"id"` // Can be string or int
	Name       string       `json:"name"`
	Email      string       `json:"email"`
	Role       string       `json:"role"`
	AssignedAt FlexibleTime `json:"assigned_at"`
}

// EvidenceTag represents a tag associated with an evidence task
type EvidenceTag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// EvidenceTaskSummary represents a summary view of evidence tasks
type EvidenceTaskSummary struct {
	Total      int            `json:"total"`
	ByStatus   map[string]int `json:"by_status"`
	ByPriority map[string]int `json:"by_priority"`
	Overdue    int            `json:"overdue"`
	DueSoon    int            `json:"due_soon"`
	LastSync   FlexibleTime   `json:"last_sync"`
}

// Additional API support structures that are referenced in the detailed response
type MasterContent struct {
	ID          interface{} `json:"id"` // Can be string or int
	Name        string      `json:"name,omitempty"`
	Description string      `json:"description,omitempty"`
	Content     string      `json:"content,omitempty"`
	Guidance    string      `json:"guidance,omitempty"` // Critical field for evidence collection
	Help        string      `json:"help,omitempty"`     // Critical field for controls
	Version     int         `json:"version,omitempty"`
}

type SupportedIntegration struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

type AecStatus struct {
	ID                string        `json:"id"`
	Status            string        `json:"status"`
	LastExecuted      *FlexibleTime `json:"last_executed,omitempty"`
	NextScheduled     *FlexibleTime `json:"next_scheduled,omitempty"`
	ErrorMessage      string        `json:"error_message,omitempty"`
	SuccessfulRuns    int           `json:"successful_runs"`
	FailedRuns        int           `json:"failed_runs"`
	LastSuccessfulRun *FlexibleTime `json:"last_successful_run,omitempty"`
}

type SubtaskMetadata struct {
	TotalSubtasks     int `json:"total_subtasks"`
	CompletedSubtasks int `json:"completed_subtasks"`
	PendingSubtasks   int `json:"pending_subtasks"`
	OverdueSubtasks   int `json:"overdue_subtasks"`
}

type InternalAec struct {
	ID            string                 `json:"id"`
	Name          string                 `json:"name"`
	Type          string                 `json:"type"`
	Configuration map[string]interface{} `json:"configuration,omitempty"`
	Enabled       bool                   `json:"enabled"`
	LastExecuted  *FlexibleTime          `json:"last_executed,omitempty"`
	Schedule      string                 `json:"schedule,omitempty"`
}

// EvidenceAttachment represents a file attachment for an evidence task
// This structure is based on OneTrust GRC Security Assurance API
type EvidenceAttachment struct {
	ID          interface{}  `json:"id"` // Can be string or int
	Name        string       `json:"name"`
	Filename    string       `json:"filename,omitempty"`
	ContentType string       `json:"content_type,omitempty"`
	Size        int64        `json:"size,omitempty"`
	UploadedAt  FlexibleTime `json:"uploaded_at,omitempty"`
	UploadedBy  interface{}  `json:"uploaded_by,omitempty"` // Can be user ID or user object
	Description string       `json:"description,omitempty"`
	URL         string       `json:"url,omitempty"`             // Download URL
	EntityType  string       `json:"__entity_type__,omitempty"` // Usually "attachment"
	Permissions []string     `json:"__permissions__,omitempty"`
}

// EvidenceImplementation represents an evidence task implementation
// This is the container for submitted evidence in Tugboat/OneTrust
type EvidenceImplementation struct {
	ID             interface{}          `json:"id"` // Can be string or int
	EvidenceTaskID interface{}          `json:"evidence_task_id,omitempty"`
	CollectedDate  string               `json:"collected_date,omitempty"` // ISO8601 date
	Status         string               `json:"status,omitempty"`
	Notes          string               `json:"notes,omitempty"`
	Attachments    []EvidenceAttachment `json:"attachments,omitempty"`
	Links          []EvidenceLink       `json:"links,omitempty"`
	CreatedAt      FlexibleTime         `json:"created_at,omitempty"`
	UpdatedAt      FlexibleTime         `json:"updated_at,omitempty"`
	CreatedBy      interface{}          `json:"created_by,omitempty"`
	AuditProjectID interface{}          `json:"audit_project_id,omitempty"`
	EntityType     string               `json:"__entity_type__,omitempty"`
	Permissions    []string             `json:"__permissions__,omitempty"`
}

// EvidenceLink represents a URL link associated with evidence
type EvidenceLink struct {
	ID          interface{}  `json:"id"`
	URL         string       `json:"url"`
	Description string       `json:"description,omitempty"`
	CreatedAt   FlexibleTime `json:"created_at,omitempty"`
}

// EvidenceSubmissionHistory represents the history of evidence submissions for a task
type EvidenceSubmissionHistory struct {
	TaskID          interface{}              `json:"task_id"`
	TaskRef         string                   `json:"task_ref,omitempty"` // ET-0001 format
	Implementations []EvidenceImplementation `json:"implementations"`
	TotalCount      int                      `json:"total_count"`
	LastSubmitted   *FlexibleTime            `json:"last_submitted,omitempty"`
}
