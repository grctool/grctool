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

// Policy represents a security policy from Tugboat Logic API
type Policy struct {
	ID          IntOrString  `json:"id"`
	Name        string       `json:"name"`
	Description string       `json:"description"`
	Framework   string       `json:"framework"` // SOC2, ISO27001, etc.
	Controls    []Control    `json:"controls"`
	Status      string       `json:"status"`
	CreatedAt   FlexibleTime `json:"created_at"`
	UpdatedAt   FlexibleTime `json:"updated_at"`
}

// PolicyDetails represents detailed policy information with embeds from Tugboat Logic API
type PolicyDetails struct {
	Policy                               // Embed the basic policy
	Summary           string             `json:"summary,omitempty"`            // Policy summary from API
	Details           string             `json:"details,omitempty"`            // Policy document content from API
	Category          string             `json:"category,omitempty"`           // Policy category (e.g., "Access Control")
	MasterPolicyID    IntOrString        `json:"master_policy_id,omitempty"`   // Reference to master policy template
	VersionNum        int                `json:"version_num,omitempty"`        // Current version number
	MasterVersionNum  int                `json:"master_version_num,omitempty"` // Master template version
	PublishedByID     IntOrString        `json:"published_by_id,omitempty"`    // User ID who published
	PublishedDate     *FlexibleTime      `json:"published_date,omitempty"`     // Publication date
	ReviewDate        *FlexibleTime      `json:"review_date,omitempty"`        // Next review date
	NoRFPUsage        bool               `json:"no_rfp_usage,omitempty"`       // RFP usage flag
	OrgID             IntOrString        `json:"org_id,omitempty"`             // Organization ID
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
	ID        string       `json:"id"`
	Version   string       `json:"version"`
	Content   string       `json:"content"`
	Status    string       `json:"status"`
	CreatedAt FlexibleTime `json:"created_at"`
	CreatedBy string       `json:"created_by"`
}

// PolicyTag represents a tag associated with a policy
type PolicyTag struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// AssociationCounts represents various association counts for a policy
type AssociationCounts struct {
	Controls   int `json:"controls"`
	Procedures int `json:"procedures"`
	Evidence   int `json:"evidence"`
}

// PolicyUsage represents usage statistics for a policy
type PolicyUsage struct {
	ViewCount      int          `json:"view_count"`
	LastViewedAt   FlexibleTime `json:"last_viewed_at"`
	DownloadCount  int          `json:"download_count"`
	LastDownloaded FlexibleTime `json:"last_downloaded"`
	ReferenceCount int          `json:"reference_count"`
	LastReferenced FlexibleTime `json:"last_referenced"`
}

// PolicyAssignee represents a user assigned to a policy
type PolicyAssignee struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Email      string       `json:"email"`
	Role       string       `json:"role"`
	AssignedAt FlexibleTime `json:"assigned_at"`
}

// PolicyReviewer represents a user who reviews a policy
type PolicyReviewer struct {
	ID         string       `json:"id"`
	Name       string       `json:"name"`
	Email      string       `json:"email"`
	ReviewedAt FlexibleTime `json:"reviewed_at"`
	Status     string       `json:"status"`
	Comments   string       `json:"comments"`
}

// PolicySummary represents a summary view of policies
type PolicySummary struct {
	Total       int            `json:"total"`
	ByFramework map[string]int `json:"by_framework"`
	ByStatus    map[string]int `json:"by_status"`
	LastSync    FlexibleTime   `json:"last_sync"`
}
