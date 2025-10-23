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
	"time"
)

// Policy represents a comprehensive security policy in the domain model
type Policy struct {
	// Basic fields
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Framework   string    `json:"framework"` // SOC2, ISO27001, etc.
	Status      string    `json:"status"`
	Controls    []Control `json:"controls,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// Reference and categorization
	ReferenceID      string `json:"reference_id,omitempty"`       // Generated reference ID (e.g., "P1", "P36")
	Category         string `json:"category,omitempty"`           // Policy category from API
	MasterPolicyID   string `json:"master_policy_id,omitempty"`   // Source master policy template ID
	VersionNum       int    `json:"version_num,omitempty"`        // Current version number
	MasterVersionNum int    `json:"master_version_num,omitempty"` // Master template version
	// Content fields
	Summary          string `json:"summary,omitempty"`
	Content          string `json:"content,omitempty"`
	DeprecationNotes string `json:"deprecation_notes,omitempty"`
	// Version management
	Version        string         `json:"version,omitempty"`
	CurrentVersion *PolicyVersion `json:"current_version,omitempty"`
	LatestVersion  *PolicyVersion `json:"latest_version,omitempty"`
	// Relationships
	Tags      []Tag    `json:"tags,omitempty"`
	Assignees []Person `json:"assignees,omitempty"`
	Reviewers []Person `json:"reviewers,omitempty"`
	// Association counts
	ControlCount   int `json:"control_count"`
	ProcedureCount int `json:"procedure_count"`
	EvidenceCount  int `json:"evidence_count"`
	RiskCount      int `json:"risk_count"`
	// Usage statistics
	ViewCount        int        `json:"view_count"`
	LastViewedAt     *time.Time `json:"last_viewed_at,omitempty"`
	DownloadCount    int        `json:"download_count"`
	LastDownloadedAt *time.Time `json:"last_downloaded_at,omitempty"`
	ReferenceCount   int        `json:"reference_count"`
	LastReferencedAt *time.Time `json:"last_referenced_at,omitempty"`
}

// PolicyVersion represents a version of a policy document
type PolicyVersion struct {
	ID        string    `json:"id"`
	Version   string    `json:"version"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	CreatedBy string    `json:"created_by"`
}

// PolicyDetails is an alias for Policy since we unified the model
// This maintains backward compatibility for existing code
type PolicyDetails = Policy

// PolicySummary represents an aggregated view of policies
type PolicySummary struct {
	Total       int            `json:"total"`
	ByFramework map[string]int `json:"by_framework"`
	ByStatus    map[string]int `json:"by_status"`
	LastSync    time.Time      `json:"last_sync"`
}
