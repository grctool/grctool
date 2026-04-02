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

package domain

import (
	"encoding/json"
	"fmt"
	"strconv"
	"time"
)

// Control represents a comprehensive security control in the domain model
type Control struct {
	// Basic fields
	ID                string     `json:"id"`
	ReferenceID       string     `json:"reference_id"`
	Name              string     `json:"name"`
	Description       string     `json:"description"`
	Category          string     `json:"category"`
	Framework         string     `json:"framework"`
	Status            string     `json:"status"` // implemented, not_applicable, etc.
	Risk              string     `json:"risk,omitempty"`
	RiskLevel         string     `json:"risk_level,omitempty"`
	Help              string     `json:"help,omitempty"`
	IsAutoImplemented bool       `json:"is_auto_implemented"`
	ImplementedDate   *time.Time `json:"implemented_date,omitempty"`
	TestedDate        *time.Time `json:"tested_date,omitempty"`
	Codes             string     `json:"codes,omitempty"`
	// API metadata
	MasterVersionNum int    `json:"master_version_num"`
	MasterControlID  int    `json:"master_control_id"`
	OrgID            int    `json:"org_id"`
	OrgScopeID       int    `json:"org_scope_id"`
	DeprecationNotes string `json:"deprecation_notes,omitempty"`
	// Relationships
	RelatedEvidenceTasks []EvidenceTask  `json:"related_evidence_tasks,omitempty"` // Evidence tasks that support this control
	Tags                 []Tag           `json:"tags,omitempty"`
	Assignees            []Person        `json:"assignees,omitempty"`
	AuditProjects        []AuditProject  `json:"audit_projects,omitempty"`
	JiraIssues           []JiraIssue     `json:"jira_issues,omitempty"`
	FrameworkCodes       []FrameworkCode `json:"framework_codes,omitempty"`
	OrgScope             *OrgScope       `json:"org_scope,omitempty"`
	// Counts and metrics
	RecommendedEvidenceCount int `json:"recommended_evidence_count"`
	OpenIncidentCount        int `json:"open_incident_count"`
	// Usage statistics (following policy pattern)
	ViewCount        int        `json:"view_count"`
	LastViewedAt     *time.Time `json:"last_viewed_at,omitempty"`
	DownloadCount    int        `json:"download_count"`
	LastDownloadedAt *time.Time `json:"last_downloaded_at,omitempty"`
	ReferenceCount   int        `json:"reference_count"`
	LastReferencedAt *time.Time `json:"last_referenced_at,omitempty"`
	// Master content and associations
	MasterContent      *ControlMasterContent   `json:"master_content,omitempty"`
	Associations       *ControlAssociations    `json:"associations,omitempty"`
	OrgEvidenceMetrics *ControlEvidenceMetrics `json:"org_evidence_metrics,omitempty"`
	// Lifecycle state (managed by lifecycle state machine)
	LifecycleState string `json:"lifecycle_state,omitempty"`
	// Multi-provider fields
	// ExternalIDs maps provider names to their external ID for this entity.
	// e.g., {"tugboat": "12345", "accountablehq": "ctrl-abc-123"}
	ExternalIDs  map[string]string `json:"external_ids,omitempty"`
	// SyncMetadata tracks multi-provider sync state.
	SyncMetadata *SyncMetadata     `json:"sync_metadata,omitempty"`
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

// UnmarshalJSON implements custom unmarshaling for Control to handle
// both integer and string ID values for backward compatibility.
func (c *Control) UnmarshalJSON(data []byte) error {
	// Use an alias to avoid infinite recursion
	type ControlAlias Control
	var raw struct {
		ControlAlias
		RawID json.RawMessage `json:"id"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}
	*c = Control(raw.ControlAlias)

	// Parse the ID field: accept both int and string
	if raw.RawID != nil && string(raw.RawID) != "null" {
		var strID string
		if err := json.Unmarshal(raw.RawID, &strID); err == nil {
			c.ID = strID
		} else {
			var intID int
			if err := json.Unmarshal(raw.RawID, &intID); err == nil {
				c.ID = strconv.Itoa(intID)
			} else {
				var floatID float64
				if err := json.Unmarshal(raw.RawID, &floatID); err == nil {
					c.ID = fmt.Sprintf("%.0f", floatID)
				} else {
					return fmt.Errorf("cannot unmarshal control ID: %s", string(raw.RawID))
				}
			}
		}
	}
	return nil
}

// ControlDetails is an alias for Control since we unified the model
// This maintains backward compatibility for existing code
type ControlDetails = Control

// ControlSummary represents an aggregated view of controls
type ControlSummary struct {
	Total       int            `json:"total"`
	ByFramework map[string]int `json:"by_framework"`
	ByStatus    map[string]int `json:"by_status"`
	ByCategory  map[string]int `json:"by_category"`
	LastSync    time.Time      `json:"last_sync"`
}
