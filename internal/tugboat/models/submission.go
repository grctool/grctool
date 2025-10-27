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

// EvidenceAttachment represents a submitted evidence attachment from Tugboat Logic API
// This maps to the org_evidence_attachment entity in Tugboat's API
type EvidenceAttachment struct {
	ID                    int                `json:"id"`
	EntityType            string             `json:"__entity_type__"`
	EntityRole            *string            `json:"__entity_role__"`
	Permissions           []string           `json:"__permissions__"`
	Created               string             `json:"created"`    // ISO 8601 timestamp
	Collected             string             `json:"collected"`  // Date when evidence was collected (YYYY-MM-DD)
	Notes                 string             `json:"notes"`      // User-provided notes about this evidence
	URL                   string             `json:"url"`        // URL for type="url" attachments
	Type                  string             `json:"type"`       // "url" or "file"
	Deleted               bool               `json:"deleted"`
	OrgID                 int                `json:"org_id"`
	OrgEvidenceID         int                `json:"org_evidence_id"`          // Evidence task ID this belongs to
	OrgEvidenceSubtaskID  *int               `json:"org_evidence_subtask_id"`  // Subtask ID (null if not a subtask)
	OwnerID               int                `json:"owner_id"`
	AttachmentID          *int               `json:"attachment_id"`          // File attachment ID (null for URLs)
	AutomatedRef          *string            `json:"automated_ref"`          // Reference if automated submission
	AutomatedMetadata     interface{}        `json:"automated_metadata"`     // Metadata for automated submissions
	IntegrationType       string             `json:"integration_type"`       // Integration source (e.g., "github", "terraform")
	IntegrationSubtype    string             `json:"integration_subtype"`    // Sub-type within integration
	Attachment            *AttachmentFile    `json:"attachment"`             // File details (embedded when type="file")
	Owner                 *OrganizationMember `json:"owner"`                 // User who submitted (embedded)
	CollectedInWindow     bool               `json:"collected_in_window"`    // Whether collected in current window
	OrgEvidenceSubtask    interface{}        `json:"org_evidence_subtask"`   // Subtask details (embedded)
}

// AttachmentFile represents file details for a submitted evidence attachment
type AttachmentFile struct {
	ID               int      `json:"id"`
	EntityType       string   `json:"__entity_type__"`
	EntityRole       *string  `json:"__entity_role__"`
	Permissions      []string `json:"__permissions__"`
	Created          string   `json:"created"`           // ISO 8601 timestamp
	Updated          string   `json:"updated"`           // ISO 8601 timestamp
	MimeType         string   `json:"mime_type"`         // e.g., "text/csv", "application/pdf"
	AttachmentType   string   `json:"attachment_type"`   // e.g., "org_evidence"
	OriginalFilename string   `json:"original_filename"` // Original file name
	OrgID            int      `json:"org_id"`
	Deleted          bool     `json:"deleted"`
	URL              string   `json:"url"` // Download URL (may be empty, need to construct)
}

// OrganizationMember represents a user/member in the organization
type OrganizationMember struct {
	ID                   int     `json:"id"`
	EntityType           string  `json:"__entity_type__"`
	FirstName            string  `json:"first_name"`
	LastName             string  `json:"last_name"`
	DisplayName          string  `json:"display_name"`       // Full display name
	ShortDisplayName     string  `json:"short_display_name"` // Initials
	Email                string  `json:"email,omitempty"`    // May not always be present
	HasReviewedName      bool    `json:"has_reviewed_name"`
	IsOwner              bool    `json:"is_owner"`
	Created              string  `json:"created"`
	Updated              string  `json:"updated"`
	Suspended            bool    `json:"suspended"`
	TrainingCompleted    *string `json:"training_completed"`   // ISO 8601 timestamp (nullable)
	TrainingExpiryDate   *string `json:"training_expiry_date"` // Date (nullable)
	PendoEnabled         bool    `json:"pendo_enabled"`
	OrgID                int     `json:"org_id"`
	Org                  int     `json:"org"`
	UserID               int     `json:"user_id"`
	AttachmentID         *int    `json:"attachment_id"` // Profile picture (nullable)
	OrgRoleID            int     `json:"org_role_id"`
	Attachment           *AttachmentFile `json:"attachment"` // Profile picture details
}

// EvidenceAttachmentListResponse represents the paginated response for evidence attachments
type EvidenceAttachmentListResponse struct {
	MaxPageSize int                   `json:"max_page_size"`
	PageSize    int                   `json:"page_size"`
	NumPages    int                   `json:"num_pages"`
	PageNumber  int                   `json:"page_number"`
	Count       int                   `json:"count"`
	Next        *int                  `json:"next"`     // Next page number (null if last page)
	Previous    *int                  `json:"previous"` // Previous page number (null if first page)
	Results     []EvidenceAttachment  `json:"results"`  // The actual attachments
}

// EvidenceAttachmentListOptions represents options for listing evidence attachments
type EvidenceAttachmentListOptions struct {
	OrgEvidence       int      `json:"org_evidence,omitempty"`        // Filter by evidence task ID
	ObservationPeriod string   `json:"observation_period,omitempty"`  // Date range filter (e.g., "2013-01-01,2025-12-31")
	Ordering          string   `json:"ordering,omitempty"`            // Sort order (e.g., "-collected,-created")
	Page              int      `json:"page,omitempty"`                // Page number
	PageSize          int      `json:"page_size,omitempty"`           // Items per page (max 200)
	Embeds            []string `json:"embeds,omitempty"`              // Include related data (attachment, org_members, etc.)
	Type              string   `json:"type,omitempty"`                // Filter by type (url, file)
	Deleted           *bool    `json:"deleted,omitempty"`             // Filter by deleted status
}

// AttachmentDownloadResponse represents the response from the attachment download endpoint
// This endpoint returns a signed S3 URL for downloading the actual file
type AttachmentDownloadResponse struct {
	URL              string `json:"url"`               // Signed S3 URL (temporary, with expiration)
	OriginalFilename string `json:"original_filename"` // Original filename of the attachment
}
