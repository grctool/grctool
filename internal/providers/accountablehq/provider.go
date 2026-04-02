// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

// Package accountablehq implements a SyncProvider for AccountableHQ
// bidirectional policy synchronization.
package accountablehq

import (
	"context"
	"fmt"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
)

// Compile-time interface assertion.
var _ interfaces.SyncProvider = (*AccountableHQSyncProvider)(nil)

// AccountableHQClient abstracts the AccountableHQ REST API.
// Implementations: real HTTP client, or stub for testing.
type AccountableHQClient interface {
	// TestConnection verifies API access.
	TestConnection(ctx context.Context) error
	// ListPolicies returns all policies, optionally filtered.
	ListPolicies(ctx context.Context, page, pageSize int) ([]AHQPolicy, int, error)
	// GetPolicy returns a single policy by ID.
	GetPolicy(ctx context.Context, id string) (*AHQPolicy, error)
	// CreatePolicy creates a new policy.
	CreatePolicy(ctx context.Context, policy *AHQPolicy) (string, error)
	// UpdatePolicy updates an existing policy.
	UpdatePolicy(ctx context.Context, id string, policy *AHQPolicy) error
	// DeletePolicy deletes a policy.
	DeletePolicy(ctx context.Context, id string) error
}

// AHQPolicy represents a policy in AccountableHQ's data model.
type AHQPolicy struct {
	ID         string    `json:"id"`
	Title      string    `json:"title"`
	Content    string    `json:"content"`
	Status     string    `json:"status"`
	Version    int       `json:"version"`
	UpdatedAt  time.Time `json:"updated_at"`
	CreatedAt  time.Time `json:"created_at"`
	Category   string    `json:"category,omitempty"`
	Owner      string    `json:"owner,omitempty"`
	ReviewDate string    `json:"review_date,omitempty"`
}

// SyncAuditEntry records a single sync operation for audit trail purposes.
type SyncAuditEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	Provider   string    `json:"provider"`
	Direction  string    `json:"direction"` // "inbound", "outbound", "bidirectional"
	PolicyID   string    `json:"policy_id"`
	ExternalID string    `json:"external_id,omitempty"`
	Action     string    `json:"action"` // "imported", "exported", "conflict_detected", "conflict_resolved", "unchanged", "error"
	Resolution string    `json:"resolution,omitempty"`
	Winner     string    `json:"winner,omitempty"` // "local", "remote", ""
	Error      string    `json:"error,omitempty"`
}

// AccountableHQSyncProvider implements interfaces.SyncProvider.
type AccountableHQSyncProvider struct {
	client   AccountableHQClient
	logger   logger.Logger
	auditLog []SyncAuditEntry // In-memory audit entries for the current sync cycle.
}

// NewAccountableHQSyncProvider creates a new provider.
func NewAccountableHQSyncProvider(client AccountableHQClient, log logger.Logger) *AccountableHQSyncProvider {
	return &AccountableHQSyncProvider{client: client, logger: log}
}

// RegisterWith registers this provider with a ProviderRegistry.
func (p *AccountableHQSyncProvider) RegisterWith(reg interfaces.ProviderRegistry) error {
	return reg.Register(p)
}

func (p *AccountableHQSyncProvider) Name() string { return "accountablehq" }

// Capabilities reports that AccountableHQ supports policies only, with write and change detection.
func (p *AccountableHQSyncProvider) Capabilities() interfaces.ProviderCapabilities {
	return interfaces.ProviderCapabilities{
		SupportsPolicies:      true,
		SupportsControls:      false,
		SupportsEvidenceTasks: false,
		SupportsWrite:         true,
		SupportsChangeDetect:  true,
	}
}

func (p *AccountableHQSyncProvider) TestConnection(ctx context.Context) error {
	return p.client.TestConnection(ctx)
}

func (p *AccountableHQSyncProvider) ListPolicies(ctx context.Context, opts interfaces.ListOptions) ([]domain.Policy, int, error) {
	page := opts.Page
	pageSize := opts.PageSize
	if pageSize == 0 {
		pageSize = 100
	}

	ahqPolicies, total, err := p.client.ListPolicies(ctx, page, pageSize)
	if err != nil {
		return nil, 0, fmt.Errorf("list AccountableHQ policies: %w", err)
	}

	policies := make([]domain.Policy, 0, len(ahqPolicies))
	for _, ap := range ahqPolicies {
		pol := convertToDomain(ap)
		setContentHash(&pol, p.logger)
		policies = append(policies, pol)
	}
	return policies, total, nil
}

func (p *AccountableHQSyncProvider) GetPolicy(ctx context.Context, id string) (*domain.Policy, error) {
	ap, err := p.client.GetPolicy(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get AccountableHQ policy %s: %w", id, err)
	}
	pol := convertToDomain(*ap)
	setContentHash(&pol, p.logger)
	return &pol, nil
}

// Controls and EvidenceTasks are not managed in AccountableHQ.
func (p *AccountableHQSyncProvider) ListControls(ctx context.Context, opts interfaces.ListOptions) ([]domain.Control, int, error) {
	return nil, 0, nil
}

func (p *AccountableHQSyncProvider) GetControl(ctx context.Context, id string) (*domain.Control, error) {
	return nil, fmt.Errorf("controls not managed in AccountableHQ")
}

func (p *AccountableHQSyncProvider) ListEvidenceTasks(ctx context.Context, opts interfaces.ListOptions) ([]domain.EvidenceTask, int, error) {
	return nil, 0, nil
}

func (p *AccountableHQSyncProvider) GetEvidenceTask(ctx context.Context, id string) (*domain.EvidenceTask, error) {
	return nil, fmt.Errorf("evidence tasks not managed in AccountableHQ")
}

func (p *AccountableHQSyncProvider) PushPolicy(ctx context.Context, policy *domain.Policy) error {
	ahqID := ""
	if policy.ExternalIDs != nil {
		ahqID = policy.ExternalIDs["accountablehq"]
	}

	ahqPolicy := convertFromDomain(policy)

	if ahqID != "" {
		if err := p.client.UpdatePolicy(ctx, ahqID, ahqPolicy); err != nil {
			p.recordAudit(policy.ID, ahqID, "exported", "error", "", err.Error())
			return err
		}
		p.recordAudit(policy.ID, ahqID, "exported", "", "", "")
		return nil
	}

	newID, err := p.client.CreatePolicy(ctx, ahqPolicy)
	if err != nil {
		p.recordAudit(policy.ID, "", "exported", "error", "", err.Error())
		return err
	}
	if policy.ExternalIDs == nil {
		policy.ExternalIDs = make(map[string]string)
	}
	policy.ExternalIDs["accountablehq"] = newID
	p.recordAudit(policy.ID, newID, "exported", "", "", "")
	return nil
}

func (p *AccountableHQSyncProvider) PushControl(ctx context.Context, control *domain.Control) error {
	return fmt.Errorf("controls not managed in AccountableHQ")
}

func (p *AccountableHQSyncProvider) PushEvidenceTask(ctx context.Context, task *domain.EvidenceTask) error {
	return fmt.Errorf("evidence tasks not managed in AccountableHQ")
}

func (p *AccountableHQSyncProvider) DeletePolicy(ctx context.Context, id string) error {
	return p.client.DeletePolicy(ctx, id)
}

func (p *AccountableHQSyncProvider) DeleteControl(ctx context.Context, id string) error {
	return fmt.Errorf("controls not managed in AccountableHQ")
}

func (p *AccountableHQSyncProvider) DeleteEvidenceTask(ctx context.Context, id string) error {
	return fmt.Errorf("evidence tasks not managed in AccountableHQ")
}

// ResolveConflict applies a conflict resolution strategy for a detected conflict.
// Per FEAT-001 US-003 and SD-001, the strategies are:
//   - local_wins:  GRCTool version is authoritative; push to AccountableHQ
//   - remote_wins: AccountableHQ version is authoritative; pull to GRCTool
//   - newest_wins: Compare timestamps; most recently modified version wins
//   - manual:      Flag conflict as unresolved; skip until user resolves
func (p *AccountableHQSyncProvider) ResolveConflict(ctx context.Context, conflict interfaces.Conflict, resolution interfaces.ConflictResolution) error {
	if conflict.EntityType != "policy" {
		return fmt.Errorf("accountablehq: only policy conflicts are supported, got %s", conflict.EntityType)
	}

	switch resolution {
	case interfaces.ConflictResolutionLocalWins:
		p.logger.Info("resolving conflict: local wins",
			logger.String("entity", conflict.EntityID),
			logger.String("provider", conflict.Provider))
		p.recordAudit(conflict.EntityID, "", "conflict_resolved", string(resolution), "local", "")
		return nil

	case interfaces.ConflictResolutionRemoteWins:
		p.logger.Info("resolving conflict: remote wins",
			logger.String("entity", conflict.EntityID),
			logger.String("provider", conflict.Provider))
		p.recordAudit(conflict.EntityID, "", "conflict_resolved", string(resolution), "remote", "")
		return nil

	case interfaces.ConflictResolutionNewestWins:
		p.logger.Info("resolving conflict: newest wins",
			logger.String("entity", conflict.EntityID),
			logger.String("provider", conflict.Provider))
		// The orchestrator determines the winner based on timestamps before
		// calling ResolveConflict. We record the decision for audit.
		p.recordAudit(conflict.EntityID, "", "conflict_resolved", string(resolution), "", "")
		return nil

	case interfaces.ConflictResolutionManual:
		p.logger.Info("conflict flagged for manual resolution",
			logger.String("entity", conflict.EntityID),
			logger.String("provider", conflict.Provider))
		p.recordAudit(conflict.EntityID, "", "conflict_detected", string(resolution), "", "")
		return nil

	default:
		return fmt.Errorf("accountablehq: unknown conflict resolution strategy: %s", resolution)
	}
}

func (p *AccountableHQSyncProvider) DetectChanges(ctx context.Context, since time.Time) (*interfaces.ChangeSet, error) {
	// Fetch all policies and filter by UpdatedAt > since
	ahqPolicies, _, err := p.client.ListPolicies(ctx, 0, 1000)
	if err != nil {
		return nil, fmt.Errorf("detect changes: %w", err)
	}

	var changes []interfaces.ChangeEntry
	for _, ap := range ahqPolicies {
		if ap.UpdatedAt.After(since) {
			pol := convertToDomain(ap)
			setContentHash(&pol, p.logger)
			hash := ""
			if pol.SyncMetadata != nil {
				hash = pol.SyncMetadata.ContentHash["accountablehq"]
			}
			changes = append(changes, interfaces.ChangeEntry{
				EntityType: "policy",
				EntityID:   ap.ID,
				ChangeType: "updated",
				Hash:       hash,
				ModifiedAt: ap.UpdatedAt,
			})
		}
	}

	return &interfaces.ChangeSet{
		Provider:   "accountablehq",
		Since:      since,
		DetectedAt: time.Now(),
		Changes:    changes,
	}, nil
}

// AuditLog returns the accumulated audit entries for the current sync cycle.
// Callers should retrieve and persist these after sync completes.
func (p *AccountableHQSyncProvider) AuditLog() []SyncAuditEntry {
	return p.auditLog
}

// ClearAuditLog resets the audit log for the next sync cycle.
func (p *AccountableHQSyncProvider) ClearAuditLog() {
	p.auditLog = nil
}

// recordAudit appends an audit trail entry.
func (p *AccountableHQSyncProvider) recordAudit(policyID, externalID, action, resolution, winner, errMsg string) {
	p.auditLog = append(p.auditLog, SyncAuditEntry{
		Timestamp:  time.Now(),
		Provider:   "accountablehq",
		Direction:  directionForAction(action),
		PolicyID:   policyID,
		ExternalID: externalID,
		Action:     action,
		Resolution: resolution,
		Winner:     winner,
		Error:      errMsg,
	})
}

// directionForAction infers sync direction from the action type.
func directionForAction(action string) string {
	switch action {
	case "exported":
		return "outbound"
	case "imported":
		return "inbound"
	case "conflict_detected", "conflict_resolved":
		return "bidirectional"
	default:
		return "unknown"
	}
}

// setContentHash computes and stores the SHA-256 content hash on a domain policy.
func setContentHash(policy *domain.Policy, log logger.Logger) {
	if policy.SyncMetadata == nil {
		policy.SyncMetadata = &domain.SyncMetadata{}
	}
	if policy.SyncMetadata.ContentHash == nil {
		policy.SyncMetadata.ContentHash = make(map[string]string)
	}
	if policy.SyncMetadata.LastSyncTime == nil {
		policy.SyncMetadata.LastSyncTime = make(map[string]time.Time)
	}

	hash, err := domain.ComputeContentHash(policy)
	if err != nil {
		log.Warn("failed to compute content hash for AccountableHQ policy",
			logger.String("id", policy.ID), logger.String("error", err.Error()))
		return
	}
	policy.SyncMetadata.ContentHash["accountablehq"] = hash
	policy.SyncMetadata.LastSyncTime["accountablehq"] = time.Now()
}

// convertToDomain converts an AccountableHQ policy to a domain policy.
func convertToDomain(ap AHQPolicy) domain.Policy {
	return domain.Policy{
		ID:          ap.ID,
		Name:        ap.Title,
		Content:     ap.Content,
		Status:      ap.Status,
		Category:    ap.Category,
		CreatedAt:   ap.CreatedAt,
		UpdatedAt:   ap.UpdatedAt,
		ExternalIDs: map[string]string{"accountablehq": ap.ID},
		SyncMetadata: &domain.SyncMetadata{
			LastSyncTime: map[string]time.Time{"accountablehq": time.Now()},
		},
	}
}

// convertFromDomain converts a domain policy to an AccountableHQ policy.
func convertFromDomain(p *domain.Policy) *AHQPolicy {
	return &AHQPolicy{
		Title:    p.Name,
		Content:  p.Content,
		Status:   p.Status,
		Category: p.Category,
	}
}
