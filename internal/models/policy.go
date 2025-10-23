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

// Common models for policy-related data structures
// Most types are now defined in tugboat/models and domain packages

// ControlAssociation represents associations between controls
type ControlAssociation struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	RelatedID    string `json:"related_id"`
	RelatedName  string `json:"related_name"`
	RelatedType  string `json:"related_type"`
	Relationship string `json:"relationship"`
}

// ControlUsage represents usage statistics for a control
type ControlUsage struct {
	Views      int       `json:"views"`
	LastViewed time.Time `json:"last_viewed"`
	Downloads  int       `json:"downloads"`
	References int       `json:"references"`
}

// MasterContent represents the master content for a control
type MasterContent struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Version     string    `json:"version"`
	LastUpdated time.Time `json:"last_updated"`
	Author      string    `json:"author"`
}

// ControlAssigneeUser represents the user information within a control assignee
type ControlAssigneeUser struct {
	ID             int         `json:"id"`
	EntityRole     interface{} `json:"__entity_role__"`
	EntityType     string      `json:"__entity_type__"`
	Permissions    []string    `json:"__permissions__"`
	Email          string      `json:"email"`
	LastLogin      string      `json:"last_login"` // ISO date string
	Status         string      `json:"status"`
	IsLockedOut    interface{} `json:"is_locked_out"`    // Can be null
	LockedOutUntil interface{} `json:"locked_out_until"` // Can be null
}

// OrgEvidenceMetrics represents evidence metrics for an organization
type OrgEvidenceMetrics struct {
	TotalEvidence     int `json:"total_evidence"`
	PendingEvidence   int `json:"pending_evidence"`
	CompletedEvidence int `json:"completed_evidence"`
	OverdueEvidence   int `json:"overdue_evidence"`
}

// ControlFilter represents filters for control queries
type ControlFilter struct {
	Status    []string `json:"status,omitempty"`
	Category  []string `json:"category,omitempty"`
	Framework string   `json:"framework,omitempty"`
}

// PolicyFilter represents filters for policy queries
type PolicyFilter struct {
	Status    []string `json:"status,omitempty"`
	Type      []string `json:"type,omitempty"`
	Framework string   `json:"framework,omitempty"`
}

// Core types referenced throughout the application

// Policy represents a security policy
type Policy struct {
	ID          IntOrString `json:"id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Framework   string      `json:"framework"` // SOC2, ISO27001, etc.
	Controls    []Control   `json:"controls"`
	Status      string      `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	UpdatedAt   time.Time   `json:"updated_at"`
}

// Control represents a security control
type Control struct {
	ID                int         `json:"id"`
	Name              string      `json:"name"`
	Body              string      `json:"body"` // Changed from Description to Body
	Category          string      `json:"category"`
	Status            string      `json:"status"` // implemented, na, etc.
	Risk              string      `json:"risk,omitempty"`
	RiskLevel         interface{} `json:"risk_level,omitempty"` // Can be null or string
	Help              string      `json:"help,omitempty"`
	IsAutoImplemented bool        `json:"is_auto_implemented"`
	ImplementedDate   interface{} `json:"implemented_date,omitempty"` // Can be null or date
	TestedDate        interface{} `json:"tested_date,omitempty"`      // Can be null or date
	Codes             string      `json:"codes,omitempty"`
	MasterVersionNum  int         `json:"master_version_num"`
	MasterControlID   int         `json:"master_control_id"`
	OrgID             int         `json:"org_id"`
	OrgScopeID        int         `json:"org_scope_id"`
	Framework         string      `json:"framework,omitempty"` // Framework this control belongs to
	// Embedded objects that come with the API response
	FrameworkCodes []FrameworkCode   `json:"framework_codes,omitempty"`
	OrgScope       *OrgScope         `json:"org_scope,omitempty"`
	Assignees      []ControlAssignee `json:"assignees,omitempty"`
	Tags           []ControlTag      `json:"tags,omitempty"`
	AuditProjects  []AuditProject    `json:"audit_projects,omitempty"`
	JiraIssues     []JiraIssue       `json:"jira_issues,omitempty"`
	// API metadata fields
	EntityRole  interface{} `json:"__entity_role__"`
	EntityType  string      `json:"__entity_type__"`
	Permissions []string    `json:"__permissions__"`
}

// FrameworkCode represents a framework code
type FrameworkCode struct {
	MasterControlID int    `json:"master_control_id"`
	FrameworkName   string `json:"framework_name"` // "SOC 2"
	FrameworkID     int    `json:"framework_id"`
	Code            string `json:"code"` // "CC7.5"
}

// OrgScope represents organizational scope
type OrgScope struct {
	ID          int      `json:"id"`
	OrgID       int      `json:"org_id"`
	Created     string   `json:"created"` // ISO date string
	Updated     string   `json:"updated"` // ISO date string
	Name        string   `json:"name"`
	Description string   `json:"description"`
	ScopeType   string   `json:"scope_type"` // "global", etc.
	Acronym     string   `json:"acronym"`
	EntityType  string   `json:"__entity_type__"`
	Permissions []string `json:"__permissions__"`
}

// ControlAssignee represents a control assignee
type ControlAssignee struct {
	ID                 int                  `json:"id"`
	FirstName          string               `json:"first_name"`
	LastName           string               `json:"last_name"`
	HasReviewedName    bool                 `json:"has_reviewed_name"`
	IsOwner            bool                 `json:"is_owner"`
	Created            string               `json:"created"`
	Updated            string               `json:"updated"`
	Suspended          bool                 `json:"suspended"`
	TrainingCompleted  interface{}          `json:"training_completed"` // Can be null
	TrainingExpiryDate string               `json:"training_expiry_date"`
	DisplayName        string               `json:"display_name"`
	ShortDisplayName   string               `json:"short_display_name"`
	PendoEnabled       bool                 `json:"pendo_enabled"`
	OrgID              int                  `json:"org_id"`
	Org                int                  `json:"org"`
	UserID             int                  `json:"user_id"`
	AttachmentID       interface{}          `json:"attachment_id"` // Can be null
	OrgRoleID          int                  `json:"org_role_id"`
	EntityType         string               `json:"__entity_type__"`
	Attachment         interface{}          `json:"attachment"` // Can be null
	User               *ControlAssigneeUser `json:"user,omitempty"`
}

// ControlTag represents a control tag
type ControlTag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// AuditProject represents an audit project
type AuditProject struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Type      string    `json:"type"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
	Framework string    `json:"framework"`
}

// JiraIssue represents a Jira issue
type JiraIssue struct {
	ID          string    `json:"id"`
	Key         string    `json:"key"`
	Summary     string    `json:"summary"`
	Status      string    `json:"status"`
	Priority    string    `json:"priority"`
	IssueType   string    `json:"issue_type"`
	CreatedDate time.Time `json:"created_date"`
	UpdatedDate time.Time `json:"updated_date"`
}

// PolicyDetails represents detailed policy information
type PolicyDetails struct {
	Policy                               // Embed the basic policy
	CurrentVersion    *PolicyVersion     `json:"current_version,omitempty"`
	LatestVersion     *PolicyVersion     `json:"latest_version,omitempty"`
	Tags              []PolicyTag        `json:"tags,omitempty"`
	AssociationCounts *AssociationCounts `json:"association_counts,omitempty"`
	Usage             *PolicyUsage       `json:"usage,omitempty"`
	DeprecationNotes  string             `json:"deprecation_notes,omitempty"`
	Assignees         []PolicyAssignee   `json:"assignees,omitempty"`
	Reviewers         []PolicyReviewer   `json:"reviewers,omitempty"`
}

// PolicyVersion represents a version of a policy
type PolicyVersion struct {
	ID        string    `json:"id"`
	Version   string    `json:"version"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

// PolicyTag represents a policy tag
type PolicyTag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// AssociationCounts represents association counts
type AssociationCounts struct {
	Controls   int `json:"controls"`
	Procedures int `json:"procedures"`
	Evidence   int `json:"evidence"`
	Risks      int `json:"risks"`
}

// PolicyUsage represents policy usage statistics
type PolicyUsage struct {
	Views      int       `json:"views"`
	LastViewed time.Time `json:"last_viewed"`
	Downloads  int       `json:"downloads"`
}

// PolicyAssignee represents a policy assignee
type PolicyAssignee struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Role  string `json:"role,omitempty"`
}

// PolicyReviewer represents a policy reviewer
type PolicyReviewer struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Email      string    `json:"email"`
	Role       string    `json:"role,omitempty"`
	ReviewedAt time.Time `json:"reviewed_at,omitempty"`
}

// ControlDetails represents detailed control information
type ControlDetails struct {
	Control                              // Embed the basic control
	OrgScope                 *OrgScope   `json:"org_scope,omitempty"`
	Associations             interface{} `json:"associations,omitempty"`   // Can be null, array, or object
	AuditProjects            interface{} `json:"audit_projects,omitempty"` // Can be null, array, or object
	JiraIssues               interface{} `json:"jira_issues,omitempty"`    // Can be null, array, or object
	Usage                    interface{} `json:"usage,omitempty"`          // Can be null or object
	MasterContent            interface{} `json:"master_content,omitempty"` // Can be null or object
	Assignees                interface{} `json:"assignees,omitempty"`      // Can be null, array, or object
	Tags                     interface{} `json:"tags,omitempty"`           // Can be null, array, or object
	DeprecationNotes         string      `json:"deprecation_notes,omitempty"`
	RecommendedEvidenceCount interface{} `json:"recommended_evidence_count,omitempty"` // Can be null or int
	FrameworkCodes           interface{} `json:"framework_codes,omitempty"`            // Can be null, array, or object
	OrgEvidenceMetrics       interface{} `json:"org_evidence_metrics,omitempty"`       // Can be null or object
	OpenIncidentCount        interface{} `json:"open_incident_count,omitempty"`        // Can be null or int
}

// PolicySummary represents a policy summary
type PolicySummary struct {
	Total       int            `json:"total"`
	ByFramework map[string]int `json:"by_framework"`
	ByStatus    map[string]int `json:"by_status"`
	LastSync    time.Time      `json:"last_sync"`
}

// ControlSummary represents a control summary
type ControlSummary struct {
	Total       int            `json:"total"`
	ByFramework map[string]int `json:"by_framework"`
	ByStatus    map[string]int `json:"by_status"`
	ByCategory  map[string]int `json:"by_category"`
	LastSync    time.Time      `json:"last_sync"`
}
