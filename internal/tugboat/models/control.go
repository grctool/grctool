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

// Control represents a security control from Tugboat Logic API
type Control struct {
	ID                int     `json:"id"`
	Name              string  `json:"name"`
	Body              string  `json:"body"` // Changed from Description to Body
	Category          string  `json:"category"`
	Status            string  `json:"status"` // implemented, na, etc.
	Risk              string  `json:"risk,omitempty"`
	RiskLevel         *string `json:"risk_level,omitempty"` // Can be null or string
	Help              string  `json:"help,omitempty"`
	IsAutoImplemented bool    `json:"is_auto_implemented"`
	ImplementedDate   *string `json:"implemented_date,omitempty"` // Can be null or date string
	TestedDate        *string `json:"tested_date,omitempty"`      // Can be null or date string
	Codes             string  `json:"codes,omitempty"`
	MasterVersionNum  int     `json:"master_version_num"`
	MasterControlID   int     `json:"master_control_id"`
	OrgID             int     `json:"org_id"`
	OrgScopeID        int     `json:"org_scope_id"`
	Framework         string  `json:"framework,omitempty"` // Framework this control belongs to
	// Embedded objects that come with the API response
	FrameworkCodes []FrameworkCode   `json:"framework_codes,omitempty"`
	OrgScope       *OrgScope         `json:"org_scope,omitempty"`
	Assignees      []ControlAssignee `json:"assignees,omitempty"`
	Tags           []ControlTag      `json:"tags,omitempty"`
	AuditProjects  []AuditProject    `json:"audit_projects,omitempty"`
	JiraIssues     []JiraIssue       `json:"jira_issues,omitempty"`
	// API metadata fields
	EntityRole  *string  `json:"__entity_role__"`
	EntityType  string   `json:"__entity_type__"`
	Permissions []string `json:"__permissions__"`
}

// ControlDetails represents detailed control information with embeds from Tugboat Logic API
type ControlDetails struct {
	Control                                          // Embed the basic control
	OrgScope                 *OrgScope               `json:"org_scope,omitempty"`
	Associations             *ControlAssociations    `json:"associations,omitempty"`
	AuditProjects            []AuditProject          `json:"audit_projects,omitempty"`
	JiraIssues               []JiraIssue             `json:"jira_issues,omitempty"`
	Usage                    *ControlUsage           `json:"usage,omitempty"`
	MasterContent            *ControlMasterContent   `json:"master_content,omitempty"`
	Assignees                []ControlAssignee       `json:"assignees,omitempty"`
	Tags                     []ControlTag            `json:"tags,omitempty"`
	DeprecationNotes         string                  `json:"deprecation_notes,omitempty"`
	RecommendedEvidenceCount *int                    `json:"recommended_evidence_count,omitempty"`
	FrameworkCodes           []FrameworkCode         `json:"framework_codes,omitempty"`
	OrgEvidenceMetrics       *ControlEvidenceMetrics `json:"org_evidence_metrics,omitempty"`
	OpenIncidentCount        *int                    `json:"open_incident_count,omitempty"`
	// Evidence relationship fields discovered in API response
	OrgEvidenceCount          *int        `json:"org_evidence_count,omitempty"`
	OrgEvidenceCollectedCount *int        `json:"org_evidence_collected_count,omitempty"`
	OrgEvidenceLastCollected  *string     `json:"org_evidence_last_collected,omitempty"`
	EvidenceTasks             interface{} `json:"evidence_tasks,omitempty"` // Embedded evidence tasks when available
	OrgEvidence               interface{} `json:"org_evidence,omitempty"`   // Alternative field for evidence tasks
}

// FrameworkCode represents a framework code associated with a control
type FrameworkCode struct {
	ID        IntOrString `json:"framework_id"`   // API returns framework_id (int), IntOrString handles conversion
	Code      string      `json:"code"`           // Direct mapping to API's code field
	Framework string      `json:"framework_name"` // API returns framework_name (e.g., "SOC 2")
	Name      string      `json:"name,omitempty"` // Optional, may not be in API response
}

// OrgScope represents the organizational scope for a control
type OrgScope struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	Description string `json:"description"`
	Type        string `json:"type"`
}

// ControlAssignee represents a user assigned to a control
type ControlAssignee struct {
	ID         IntOrString  `json:"id"`
	Name       string       `json:"name"`
	Email      string       `json:"email"`
	Role       string       `json:"role"`
	AssignedAt FlexibleTime `json:"assigned_at"`
}

// ControlTag represents a tag associated with a control
type ControlTag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// AuditProject represents an audit project associated with a control
type AuditProject struct {
	ID          IntOrString  `json:"id"`
	Name        string       `json:"name"`
	Status      string       `json:"status"`
	StartDate   FlexibleTime `json:"start_date"`
	EndDate     FlexibleTime `json:"end_date"`
	Description string       `json:"description"`
}

// JiraIssue represents a Jira issue associated with a control
type JiraIssue struct {
	ID        string       `json:"id"`
	Key       string       `json:"key"`
	Summary   string       `json:"summary"`
	Status    string       `json:"status"`
	Priority  string       `json:"priority"`
	IssueType string       `json:"issue_type"`
	CreatedAt FlexibleTime `json:"created_at"`
	UpdatedAt FlexibleTime `json:"updated_at"`
	Assignee  string       `json:"assignee"`
	Reporter  string       `json:"reporter"`
}

// ControlUsage represents usage statistics for a control
type ControlUsage struct {
	ViewCount      int          `json:"view_count"`
	LastViewedAt   FlexibleTime `json:"last_viewed_at"`
	DownloadCount  int          `json:"download_count"`
	LastDownloaded FlexibleTime `json:"last_downloaded"`
	ReferenceCount int          `json:"reference_count"`
	LastReferenced FlexibleTime `json:"last_referenced"`
}

// ControlMasterContent represents the master content for a control
type ControlMasterContent struct {
	Help        string `json:"help"`
	Guidance    string `json:"guidance"`
	Description string `json:"description"`
}

// ControlAssociations represents various association counts for a control
type ControlAssociations struct {
	Policies   int `json:"policies"`
	Procedures int `json:"procedures"`
	Evidence   int `json:"evidence"`
	Risks      int `json:"risks"`
}

// ControlEvidenceMetrics represents evidence-related metrics for a control
type ControlEvidenceMetrics struct {
	TotalCount    int `json:"total_count"`
	CompleteCount int `json:"complete_count"`
	OverdueCount  int `json:"overdue_count"`
}

// ControlSummary represents a summary view of controls
type ControlSummary struct {
	Total       int            `json:"total"`
	ByFramework map[string]int `json:"by_framework"`
	ByStatus    map[string]int `json:"by_status"`
	ByCategory  map[string]int `json:"by_category"`
	LastSync    FlexibleTime   `json:"last_sync"`
}
