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
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Status      string    `json:"status"`
	Version     int       `json:"version"`
	UpdatedAt   time.Time `json:"updated_at"`
	CreatedAt   time.Time `json:"created_at"`
	Category    string    `json:"category,omitempty"`
	Owner       string    `json:"owner,omitempty"`
	ReviewDate  string    `json:"review_date,omitempty"`
}

// AccountableHQSyncProvider implements interfaces.SyncProvider.
type AccountableHQSyncProvider struct {
	client AccountableHQClient
	logger logger.Logger
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
		policies = append(policies, convertToDomain(ap))
	}
	return policies, total, nil
}

func (p *AccountableHQSyncProvider) GetPolicy(ctx context.Context, id string) (*domain.Policy, error) {
	ap, err := p.client.GetPolicy(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get AccountableHQ policy %s: %w", id, err)
	}
	pol := convertToDomain(*ap)
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
		return p.client.UpdatePolicy(ctx, ahqID, ahqPolicy)
	}

	newID, err := p.client.CreatePolicy(ctx, ahqPolicy)
	if err != nil {
		return err
	}
	if policy.ExternalIDs == nil {
		policy.ExternalIDs = make(map[string]string)
	}
	policy.ExternalIDs["accountablehq"] = newID
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

func (p *AccountableHQSyncProvider) ResolveConflict(ctx context.Context, conflict interfaces.Conflict, resolution interfaces.ConflictResolution) error {
	// Conflict resolution workflow is not yet implemented.
	// Detection is supported via DetectChanges; resolution requires the
	// user-facing workflow design (SD-004 Open Question #3).
	return fmt.Errorf("accountablehq: conflict resolution not yet implemented for %s %s", conflict.EntityType, conflict.EntityID)
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
			changes = append(changes, interfaces.ChangeEntry{
				EntityType: "policy",
				EntityID:   ap.ID,
				ChangeType: "updated",
				Hash:       fmt.Sprintf("%d", ap.Version),
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
			ContentHash:  map[string]string{"accountablehq": fmt.Sprintf("v%d", ap.Version)},
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
