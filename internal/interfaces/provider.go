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

package interfaces

import (
	"context"
	"io"
	"time"

	"github.com/grctool/grctool/internal/domain"
)

// ListOptions provides pagination and filtering for list operations.
// Fields map to what common compliance APIs support (page-based pagination,
// framework filtering). Providers may ignore unsupported fields.
type ListOptions struct {
	Page      int    `json:"page,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
	Framework string `json:"framework,omitempty"`
	Status    string `json:"status,omitempty"`
	Category  string `json:"category,omitempty"`
}

// ChangeEntry represents a single change detected by a SyncProvider.
type ChangeEntry struct {
	EntityType string    `json:"entity_type"` // "policy", "control", "evidence_task"
	EntityID   string    `json:"entity_id"`
	ChangeType string    `json:"change_type"` // "created", "updated", "deleted"
	Hash       string    `json:"hash,omitempty"`
	ModifiedAt time.Time `json:"modified_at"`
}

// ChangeSet represents all changes detected since a given point.
type ChangeSet struct {
	Provider   string        `json:"provider"`
	Since      time.Time     `json:"since"`
	DetectedAt time.Time     `json:"detected_at"`
	Changes    []ChangeEntry `json:"changes,omitempty"`
}

// ProviderRegistry manages registration and lookup of DataProvider instances.
// Services depend on this interface rather than the concrete struct in
// internal/providers, enabling mock injection in tests.
type ProviderRegistry interface {
	// Register adds a provider. Returns error if name already registered.
	Register(provider DataProvider) error

	// Get returns a provider by name, or error if not found.
	Get(name string) (DataProvider, error)

	// GetSyncProvider returns the provider as a SyncProvider, or error if
	// not found or if the provider doesn't implement SyncProvider.
	GetSyncProvider(name string) (SyncProvider, error)

	// List returns all registered provider names, sorted.
	List() []string

	// ListSyncProviders returns names of providers implementing SyncProvider.
	ListSyncProviders() []string

	// HealthCheck tests connectivity for all registered providers.
	HealthCheck(ctx context.Context) map[string]error
}

// ProviderCapabilities reports which entity types and operations a provider supports.
// Callers should check capabilities before calling entity-specific methods.
type ProviderCapabilities struct {
	SupportsPolicies      bool `json:"supports_policies"`
	SupportsControls      bool `json:"supports_controls"`
	SupportsEvidenceTasks bool `json:"supports_evidence_tasks"`
	SupportsWrite         bool `json:"supports_write"`
	SupportsChangeDetect  bool `json:"supports_change_detect"`
}

// DataProvider is the read-only interface for compliance data sources.
// Any system that can provide policies, controls, or evidence tasks
// implements this interface.
type DataProvider interface {
	// Name returns the unique identifier for this provider (e.g., "tugboat", "accountablehq").
	Name() string

	// Capabilities reports which entity types and operations this provider supports.
	Capabilities() ProviderCapabilities

	// TestConnection verifies that the provider is reachable and authenticated.
	TestConnection(ctx context.Context) error

	// ListPolicies returns a page of policies and the total count.
	ListPolicies(ctx context.Context, opts ListOptions) ([]domain.Policy, int, error)

	// GetPolicy retrieves a single policy by its provider-native ID.
	GetPolicy(ctx context.Context, id string) (*domain.Policy, error)

	// ListControls returns a page of controls and the total count.
	ListControls(ctx context.Context, opts ListOptions) ([]domain.Control, int, error)

	// GetControl retrieves a single control by its provider-native ID.
	GetControl(ctx context.Context, id string) (*domain.Control, error)

	// ListEvidenceTasks returns a page of evidence tasks and the total count.
	ListEvidenceTasks(ctx context.Context, opts ListOptions) ([]domain.EvidenceTask, int, error)

	// GetEvidenceTask retrieves a single evidence task by its provider-native ID.
	GetEvidenceTask(ctx context.Context, id string) (*domain.EvidenceTask, error)
}

// SyncProvider extends DataProvider with write-back and change detection
// for bidirectional synchronization.
type SyncProvider interface {
	DataProvider

	// PushPolicy creates or updates a policy in the remote system.
	PushPolicy(ctx context.Context, policy *domain.Policy) error

	// PushControl creates or updates a control in the remote system.
	PushControl(ctx context.Context, control *domain.Control) error

	// PushEvidenceTask creates or updates an evidence task in the remote system.
	PushEvidenceTask(ctx context.Context, task *domain.EvidenceTask) error

	// DeletePolicy removes a policy from the remote system.
	DeletePolicy(ctx context.Context, id string) error

	// DeleteControl removes a control from the remote system.
	DeleteControl(ctx context.Context, id string) error

	// DeleteEvidenceTask removes an evidence task from the remote system.
	DeleteEvidenceTask(ctx context.Context, id string) error

	// DetectChanges returns entities that changed since the given time.
	DetectChanges(ctx context.Context, since time.Time) (*ChangeSet, error)

	// ResolveConflict applies a conflict resolution decision to an entity.
	// Returns an error if the resolution cannot be applied (e.g., the entity
	// was modified again since the conflict was detected).
	ResolveConflict(ctx context.Context, conflict Conflict, resolution ConflictResolution) error
}

// ConflictResolution enumerates strategies for resolving sync conflicts.
type ConflictResolution string

const (
	ConflictResolutionLocalWins  ConflictResolution = "local_wins"
	ConflictResolutionRemoteWins ConflictResolution = "remote_wins"
	ConflictResolutionNewestWins ConflictResolution = "newest_wins"
	ConflictResolutionManual     ConflictResolution = "manual"
)

// Conflict represents a detected conflict between local and remote state
// for a single entity.
type Conflict struct {
	EntityType string    `json:"entity_type"` // "policy", "control", "evidence_task"
	EntityID   string    `json:"entity_id"`   // GRCTool-native ID
	Provider   string    `json:"provider"`
	LocalHash  string    `json:"local_hash"`
	RemoteHash string    `json:"remote_hash"`
	DetectedAt time.Time `json:"detected_at"`
}

// SubmissionMetadata provides context for an evidence submission.
type SubmissionMetadata struct {
	CollectedDate string            `json:"collected_date"`          // YYYY-MM-DD
	Notes         string            `json:"notes,omitempty"`
	Window        string            `json:"window,omitempty"`        // e.g., "2026-Q1"
	ContentType   string            `json:"content_type,omitempty"`  // MIME type
	Filename      string            `json:"filename,omitempty"`
	Metadata      map[string]string `json:"metadata,omitempty"`
}

// Attachment represents an evidence attachment in an upstream platform.
type Attachment struct {
	ID              string `json:"id"`
	TaskID          string `json:"task_id"`
	Filename        string `json:"filename"`
	MimeType        string `json:"mime_type"`
	Type            string `json:"type"`              // "file" or "url"
	URL             string `json:"url,omitempty"`
	CollectedDate   string `json:"collected_date"`    // YYYY-MM-DD
	Notes           string `json:"notes,omitempty"`
	Owner           string `json:"owner,omitempty"`
	IntegrationType string `json:"integration_type,omitempty"`
	Deleted         bool   `json:"deleted,omitempty"`
}

// EvidenceSubmitter is an optional interface for providers that support
// evidence submission and attachment management. Not all providers have
// submission capability — callers must type-assert before use:
//
//	if es, ok := provider.(EvidenceSubmitter); ok {
//	    err := es.SubmitEvidence(ctx, taskID, file, meta)
//	}
type EvidenceSubmitter interface {
	// SubmitEvidence uploads evidence for a task.
	SubmitEvidence(ctx context.Context, taskID string, file io.Reader, meta SubmissionMetadata) error

	// ListAttachments returns evidence attachments for a task.
	ListAttachments(ctx context.Context, taskID string, opts ListOptions) ([]Attachment, int, error)

	// DownloadAttachment returns a reader for an attachment's content.
	DownloadAttachment(ctx context.Context, attachmentID string) (io.ReadCloser, string, error) // reader, filename, error
}

// RelationshipQuerier is an optional interface for providers that support
// cross-entity relationship queries. Not all providers have relationship
// data — callers must type-assert before use:
//
//	if rq, ok := provider.(RelationshipQuerier); ok {
//	    tasks, err := rq.GetEvidenceTasksByControl(ctx, controlID)
//	}
type RelationshipQuerier interface {
	// GetEvidenceTasksByControl returns evidence tasks linked to a control.
	GetEvidenceTasksByControl(ctx context.Context, controlID string) ([]domain.EvidenceTask, error)

	// GetControlsByPolicy returns controls implementing a policy.
	GetControlsByPolicy(ctx context.Context, policyID string) ([]domain.Control, error)

	// GetPoliciesByControl returns policies that a control implements.
	GetPoliciesByControl(ctx context.Context, controlID string) ([]domain.Policy, error)
}
