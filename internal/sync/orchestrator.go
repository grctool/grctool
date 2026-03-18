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

package sync

import (
	"context"
	"fmt"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/providers"
)

// SyncOrchestrator coordinates bidirectional sync across providers.
// It drives a three-phase cycle: detect changes, resolve conflicts,
// and apply updates in both directions.
type SyncOrchestrator struct {
	registry *providers.ProviderRegistry
	storage  interfaces.StorageService
	detector *ConflictDetector
	resolver *ConflictResolver
	logger   logger.Logger
}

// NewSyncOrchestrator creates an orchestrator with the given registry,
// storage service, conflict policy, and logger.
func NewSyncOrchestrator(
	registry *providers.ProviderRegistry,
	storage interfaces.StorageService,
	policy ConflictPolicy,
	log logger.Logger,
) *SyncOrchestrator {
	return &SyncOrchestrator{
		registry: registry,
		storage:  storage,
		detector: NewConflictDetector(),
		resolver: NewConflictResolver(policy),
		logger:   log,
	}
}

// OrchestratorResult captures the outcome of an orchestrated sync.
type OrchestratorResult struct {
	Provider      string        `json:"provider"`
	PullCount     int           `json:"pull_count"`     // entities pulled from remote to local
	PushCount     int           `json:"push_count"`     // entities pushed from local to remote
	ConflictCount int           `json:"conflict_count"` // conflicts detected
	ResolvedCount int           `json:"resolved_count"` // conflicts resolved
	ManualCount   int           `json:"manual_count"`   // conflicts left for manual resolution
	Errors        []string      `json:"errors,omitempty"`
	Duration      time.Duration `json:"duration"`
}

// SyncProvider runs the detect-resolve-apply cycle for one SyncProvider.
func (o *SyncOrchestrator) SyncProvider(ctx context.Context, providerName string, since time.Time) (*OrchestratorResult, error) {
	start := time.Now()
	result := &OrchestratorResult{Provider: providerName}

	// 1. Get the SyncProvider from the registry.
	sp, err := o.registry.GetSyncProvider(providerName)
	if err != nil {
		return nil, err
	}

	o.logger.Info("starting sync", logger.Field{Key: "provider", Value: providerName})

	// 2. DETECT — get remote changes since the given time.
	changeSet, err := sp.DetectChanges(ctx, since)
	if err != nil {
		return nil, fmt.Errorf("detect changes: %w", err)
	}

	// 3. For each change, load local entity and detect conflicts.
	var conflicts []Conflict
	var pullEntries []interfaces.ChangeEntry

	for _, change := range changeSet.Changes {
		localMeta := o.getLocalSyncMetadata(change.EntityType, change.EntityID, providerName)

		detected := o.detector.DetectConflicts(localMeta, []interfaces.ChangeEntry{change}, providerName)
		if len(detected) == 0 {
			// In sync — nothing to do.
			continue
		}

		// Classify: remote-only and remote-changed are not true conflicts,
		// they can be pulled directly. Only both_changed and local_changed
		// require conflict resolution.
		for _, d := range detected {
			switch d.Classification {
			case ClassificationRemoteOnly, ClassificationRemoteChanged:
				pullEntries = append(pullEntries, change)
			default:
				conflicts = append(conflicts, d)
			}
		}
	}

	// 4. RESOLVE conflicts.
	resolutions := o.resolver.ResolveAll(conflicts)
	for _, res := range resolutions {
		result.ConflictCount++
		switch res.Winner {
		case "remote":
			pullEntries = append(pullEntries, interfaces.ChangeEntry{
				EntityType: res.Conflict.EntityType,
				EntityID:   res.Conflict.EntityID,
				ChangeType: "updated",
				Hash:       res.Conflict.RemoteHash,
				ModifiedAt: res.Conflict.RemoteModified,
			})
			result.ResolvedCount++
		case "local":
			// Local wins — push local entity to remote.
			if pushErr := o.pushEntity(ctx, sp, res.Conflict.EntityType, res.Conflict.EntityID); pushErr != nil {
				result.Errors = append(result.Errors, pushErr.Error())
			} else {
				result.PushCount++
			}
			result.ResolvedCount++
		default:
			// Manual — mark entity as conflicted.
			result.ManualCount++
		}
	}

	// 5. APPLY — pull remote changes to local storage.
	for _, entry := range pullEntries {
		if pullErr := o.pullEntity(ctx, sp, entry); pullErr != nil {
			result.Errors = append(result.Errors, pullErr.Error())
			continue
		}
		result.PullCount++
	}

	result.Duration = time.Since(start)

	o.logger.Info("sync complete",
		logger.Field{Key: "provider", Value: providerName},
		logger.Field{Key: "pulled", Value: result.PullCount},
		logger.Field{Key: "pushed", Value: result.PushCount},
		logger.Field{Key: "conflicts", Value: result.ConflictCount},
	)

	return result, nil
}

// SyncAll runs the detect-resolve-apply cycle for all registered SyncProviders.
func (o *SyncOrchestrator) SyncAll(ctx context.Context, since time.Time) ([]*OrchestratorResult, error) {
	names := o.registry.ListSyncProviders()
	results := make([]*OrchestratorResult, 0, len(names))

	for _, name := range names {
		res, err := o.SyncProvider(ctx, name, since)
		if err != nil {
			return results, fmt.Errorf("sync provider %s: %w", name, err)
		}
		results = append(results, res)
	}

	return results, nil
}

// getLocalSyncMetadata loads the entity from storage and returns its SyncMetadata.
// Returns nil if the entity is not found locally.
func (o *SyncOrchestrator) getLocalSyncMetadata(entityType, entityID, provider string) *domain.SyncMetadata {
	switch entityType {
	case "policy":
		p, err := o.storage.GetPolicyByExternalID(provider, entityID)
		if err != nil || p == nil {
			return nil
		}
		return p.SyncMetadata
	case "control":
		c, err := o.storage.GetControlByExternalID(provider, entityID)
		if err != nil || c == nil {
			return nil
		}
		return c.SyncMetadata
	case "evidence_task":
		et, err := o.storage.GetEvidenceTaskByExternalID(provider, entityID)
		if err != nil || et == nil {
			return nil
		}
		return et.SyncMetadata
	default:
		return nil
	}
}

// pullEntity fetches an entity from the remote provider and saves it to local storage
// with updated SyncMetadata.
func (o *SyncOrchestrator) pullEntity(ctx context.Context, sp interfaces.SyncProvider, change interfaces.ChangeEntry) error {
	now := time.Now()
	providerName := sp.Name()

	switch change.EntityType {
	case "policy":
		policy, err := sp.GetPolicy(ctx, change.EntityID)
		if err != nil {
			return fmt.Errorf("pull policy %s: %w", change.EntityID, err)
		}
		if policy.SyncMetadata == nil {
			policy.SyncMetadata = &domain.SyncMetadata{
				LastSyncTime: make(map[string]time.Time),
				ContentHash:  make(map[string]string),
			}
		}
		policy.SyncMetadata.LastSyncTime[providerName] = now
		policy.SyncMetadata.ContentHash[providerName] = change.Hash
		policy.SyncMetadata.ConflictState = ""
		return o.storage.SavePolicy(policy)

	case "control":
		control, err := sp.GetControl(ctx, change.EntityID)
		if err != nil {
			return fmt.Errorf("pull control %s: %w", change.EntityID, err)
		}
		if control.SyncMetadata == nil {
			control.SyncMetadata = &domain.SyncMetadata{
				LastSyncTime: make(map[string]time.Time),
				ContentHash:  make(map[string]string),
			}
		}
		control.SyncMetadata.LastSyncTime[providerName] = now
		control.SyncMetadata.ContentHash[providerName] = change.Hash
		control.SyncMetadata.ConflictState = ""
		return o.storage.SaveControl(control)

	case "evidence_task":
		task, err := sp.GetEvidenceTask(ctx, change.EntityID)
		if err != nil {
			return fmt.Errorf("pull evidence_task %s: %w", change.EntityID, err)
		}
		if task.SyncMetadata == nil {
			task.SyncMetadata = &domain.SyncMetadata{
				LastSyncTime: make(map[string]time.Time),
				ContentHash:  make(map[string]string),
			}
		}
		task.SyncMetadata.LastSyncTime[providerName] = now
		task.SyncMetadata.ContentHash[providerName] = change.Hash
		task.SyncMetadata.ConflictState = ""
		return o.storage.SaveEvidenceTask(task)

	default:
		return fmt.Errorf("unknown entity type: %s", change.EntityType)
	}
}

// pushEntity loads an entity from local storage and pushes it to the remote provider.
func (o *SyncOrchestrator) pushEntity(ctx context.Context, sp interfaces.SyncProvider, entityType, entityID string) error {
	providerName := sp.Name()

	switch entityType {
	case "policy":
		policy, err := o.storage.GetPolicyByExternalID(providerName, entityID)
		if err != nil {
			return fmt.Errorf("load policy %s for push: %w", entityID, err)
		}
		return sp.PushPolicy(ctx, policy)

	case "control":
		control, err := o.storage.GetControlByExternalID(providerName, entityID)
		if err != nil {
			return fmt.Errorf("load control %s for push: %w", entityID, err)
		}
		return sp.PushControl(ctx, control)

	case "evidence_task":
		task, err := o.storage.GetEvidenceTaskByExternalID(providerName, entityID)
		if err != nil {
			return fmt.Errorf("load evidence_task %s for push: %w", entityID, err)
		}
		return sp.PushEvidenceTask(ctx, task)

	default:
		return fmt.Errorf("unknown entity type: %s", entityType)
	}
}
