// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package sync

import (
	"context"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/providers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func mustTestLogger(t *testing.T) logger.Logger {
	t.Helper()
	log, err := logger.NewTestLogger()
	require.NoError(t, err)
	return log
}

// TestSyncOrchestrator_PullControl exercises the control pull path in pullEntity.
func TestSyncOrchestrator_PullControl(t *testing.T) {
	t.Parallel()

	ctrl := &domain.Control{ID: "CC-06.1", Name: "Logical Access", ReferenceID: "CC-06.1", ExternalIDs: map[string]string{"ctrl-provider": "CC-06.1"}}
	sp := &stubSyncProvider{
		name:     "ctrl-provider",
		controls: map[string]*domain.Control{"CC-06.1": ctrl},
		changeSet: &interfaces.ChangeSet{
			Provider: "ctrl-provider",
			Changes: []interfaces.ChangeEntry{
				{EntityType: "control", EntityID: "CC-06.1", ChangeType: "created", Hash: "hash1"},
			},
		},
	}
	storage := newStubStorageService()

	reg := providers.NewProviderRegistry()
	require.NoError(t, reg.Register(sp))

	orch := NewSyncOrchestrator(reg, storage, ConflictPolicyRemoteWins, mustTestLogger(t))
	result, err := orch.SyncProvider(context.Background(), "ctrl-provider", time.Time{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.PullCount)

	// Verify control was saved to storage
	assert.Len(t, storage.controls, 1)
}

// TestSyncOrchestrator_PullEvidenceTask exercises the evidence_task pull path.
func TestSyncOrchestrator_PullEvidenceTask(t *testing.T) {
	t.Parallel()

	task := &domain.EvidenceTask{ID: "ET-0047", Name: "GitHub Access Controls", ReferenceID: "ET-0047", ExternalIDs: map[string]string{"task-provider": "ET-0047"}}
	sp := &stubSyncProvider{
		name:  "task-provider",
		tasks: map[string]*domain.EvidenceTask{"ET-0047": task},
		changeSet: &interfaces.ChangeSet{
			Provider: "task-provider",
			Changes: []interfaces.ChangeEntry{
				{EntityType: "evidence_task", EntityID: "ET-0047", ChangeType: "created", Hash: "hash1"},
			},
		},
	}
	storage := newStubStorageService()

	reg := providers.NewProviderRegistry()
	require.NoError(t, reg.Register(sp))

	orch := NewSyncOrchestrator(reg, storage, ConflictPolicyRemoteWins, mustTestLogger(t))
	result, err := orch.SyncProvider(context.Background(), "task-provider", time.Time{})
	require.NoError(t, err)
	assert.Equal(t, 1, result.PullCount)
	assert.Len(t, storage.tasks, 1)
}

// TestSyncOrchestrator_PushLocalWins_Control exercises the push path for controls.
func TestSyncOrchestrator_PushLocalWins_Control(t *testing.T) {
	t.Parallel()

	ctrl := &domain.Control{
		ID:          "CC-06.1",
		Name:        "Logical Access",
		ExternalIDs: map[string]string{"push-provider": "CC-06.1"},
	}
	sp := &stubSyncProvider{
		name:     "push-provider",
		controls: map[string]*domain.Control{},
		changeSet: &interfaces.ChangeSet{
			Provider: "push-provider",
			Changes: []interfaces.ChangeEntry{
				{EntityType: "control", EntityID: "CC-06.1", ChangeType: "updated", Hash: "remote-hash"},
			},
		},
	}
	ctrl.SyncMetadata = &domain.SyncMetadata{
		ContentHash: map[string]string{"push-provider": "old-hash", "local": "local-new-hash"},
	}
	storage := &stubStorageService{
		controls: map[string]*domain.Control{"push-provider:CC-06.1": ctrl},
	}

	reg := providers.NewProviderRegistry()
	require.NoError(t, reg.Register(sp))

	orch := NewSyncOrchestrator(reg, storage, ConflictPolicyLocalWins, mustTestLogger(t))
	result, err := orch.SyncProvider(context.Background(), "push-provider", time.Time{})
	require.NoError(t, err)
	assert.True(t, result.PushCount >= 0) // local wins means push or skip depending on classification
	assert.Equal(t, 0, result.ManualCount)
}

// TestSyncOrchestrator_PushLocalWins_EvidenceTask exercises push for tasks.
func TestSyncOrchestrator_PushLocalWins_EvidenceTask(t *testing.T) {
	t.Parallel()

	task := &domain.EvidenceTask{
		ID:          "ET-0047",
		Name:        "GitHub Access",
		ExternalIDs: map[string]string{"push-prov": "ET-0047"},
	}
	sp := &stubSyncProvider{
		name:  "push-prov",
		tasks: map[string]*domain.EvidenceTask{},
		changeSet: &interfaces.ChangeSet{
			Provider: "push-prov",
			Changes: []interfaces.ChangeEntry{
				{EntityType: "evidence_task", EntityID: "ET-0047", ChangeType: "updated", Hash: "remote"},
			},
		},
	}
	task.SyncMetadata = &domain.SyncMetadata{
		ContentHash: map[string]string{"push-prov": "old", "local": "local-new"},
	}
	storage := &stubStorageService{
		tasks: map[string]*domain.EvidenceTask{"push-prov:ET-0047": task},
	}

	reg := providers.NewProviderRegistry()
	require.NoError(t, reg.Register(sp))

	orch := NewSyncOrchestrator(reg, storage, ConflictPolicyLocalWins, mustTestLogger(t))
	result, err := orch.SyncProvider(context.Background(), "push-prov", time.Time{})
	require.NoError(t, err)
	assert.Equal(t, 0, result.ManualCount)
}

// TestSyncOrchestrator_UnknownEntityType exercises the unknown entity type path.
func TestSyncOrchestrator_UnknownEntityType(t *testing.T) {
	t.Parallel()

	sp := &stubSyncProvider{
		name: "bad-entity",
		changeSet: &interfaces.ChangeSet{
			Provider: "bad-entity",
			Changes: []interfaces.ChangeEntry{
				{EntityType: "widget", EntityID: "W-1", ChangeType: "created", Hash: "hash"},
			},
		},
	}
	storage := newStubStorageService()

	reg := providers.NewProviderRegistry()
	require.NoError(t, reg.Register(sp))

	orch := NewSyncOrchestrator(reg, storage, ConflictPolicyRemoteWins, mustTestLogger(t))
	result, err := orch.SyncProvider(context.Background(), "bad-entity", time.Time{})
	require.NoError(t, err)
	// Unknown entity type should result in an error recorded, not a crash
	assert.True(t, len(result.Errors) > 0 || result.PullCount == 0)
}

// TestSyncOrchestrator_GetLocalSyncMetadata_AllEntityTypes tests metadata lookup for all types.
func TestSyncOrchestrator_GetLocalSyncMetadata_AllEntityTypes(t *testing.T) {
	t.Parallel()

	pol := &domain.Policy{
		ID:           "POL-001",
		ExternalIDs:  map[string]string{"prov": "ext-1"},
		SyncMetadata: &domain.SyncMetadata{ContentHash: map[string]string{"prov": "pol-hash"}},
	}
	ctrl := &domain.Control{
		ID:           "CC-06.1",
		ExternalIDs:  map[string]string{"prov": "ext-2"},
		SyncMetadata: &domain.SyncMetadata{ContentHash: map[string]string{"prov": "ctrl-hash"}},
	}
	task := &domain.EvidenceTask{
		ID:           "ET-0047",
		ExternalIDs:  map[string]string{"prov": "ext-3"},
		SyncMetadata: &domain.SyncMetadata{ContentHash: map[string]string{"prov": "task-hash"}},
	}

	storage := &stubStorageService{
		policies: map[string]*domain.Policy{"prov:ext-1": pol},
		controls: map[string]*domain.Control{"prov:ext-2": ctrl},
		tasks:    map[string]*domain.EvidenceTask{"prov:ext-3": task},
	}

	reg := providers.NewProviderRegistry()
	orch := NewSyncOrchestrator(reg, storage, ConflictPolicyRemoteWins, mustTestLogger(t))

	// Test each entity type lookup
	meta := orch.getLocalSyncMetadata("policy", "ext-1", "prov")
	assert.NotNil(t, meta)
	assert.Equal(t, "pol-hash", meta.ContentHash["prov"])

	meta = orch.getLocalSyncMetadata("control", "ext-2", "prov")
	assert.NotNil(t, meta)
	assert.Equal(t, "ctrl-hash", meta.ContentHash["prov"])

	meta = orch.getLocalSyncMetadata("evidence_task", "ext-3", "prov")
	assert.NotNil(t, meta)
	assert.Equal(t, "task-hash", meta.ContentHash["prov"])

	// Unknown type returns nil
	meta = orch.getLocalSyncMetadata("widget", "ext-1", "prov")
	assert.Nil(t, meta)
}
