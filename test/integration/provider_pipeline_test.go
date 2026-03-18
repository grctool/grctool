// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/providers"
	grcsync "github.com/grctool/grctool/internal/sync"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestProviderPipeline_EndToEnd exercises the full provider framework:
// registry → provider → sync orchestrator → conflict resolution → audit trail
func TestProviderPipeline_EndToEnd(t *testing.T) {
	t.Parallel()
	log := testhelpers.NewStubLogger()

	// 1. Create registry and register a stub provider
	reg := providers.NewProviderRegistry()
	stub := testhelpers.NewStubDataProvider("test-provider")

	pol := testhelpers.SamplePolicy()
	pol.ExternalIDs = map[string]string{"test-provider": "ext-pol-1"}
	stub.Policies[pol.ID] = pol

	ctrl := testhelpers.SampleControl()
	ctrl.ExternalIDs = map[string]string{"test-provider": "ext-ctrl-1"}
	stub.Controls[ctrl.ID] = ctrl

	task := testhelpers.SampleEvidenceTask()
	task.ExternalIDs = map[string]string{"test-provider": "ext-task-1"}
	stub.Tasks[task.ID] = task

	require.NoError(t, reg.Register(stub))

	// 2. Verify provider via registry
	provider, err := reg.Get("test-provider")
	require.NoError(t, err)
	assert.Equal(t, "test-provider", provider.Name())

	// 3. List entities through DataProvider interface
	policies, count, err := provider.ListPolicies(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, pol.Name, policies[0].Name)

	controls, count, err := provider.ListControls(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, ctrl.Name, controls[0].Name)

	tasks, count, err := provider.ListEvidenceTasks(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Equal(t, task.Name, tasks[0].Name)

	// 4. Test conflict detection
	detector := &grcsync.ConflictDetector{}
	meta := &domain.SyncMetadata{
		ContentHash: map[string]string{"test-provider": "hash-old"},
		LastSyncTime: map[string]time.Time{"test-provider": time.Now().Add(-1 * time.Hour)},
	}
	changes := []interfaces.ChangeEntry{
		{EntityType: "policy", EntityID: pol.ID, ChangeType: "updated", Hash: "hash-new"},
	}
	conflicts := detector.DetectConflicts(meta, changes, "test-provider")
	assert.Len(t, conflicts, 1)
	assert.Equal(t, grcsync.ClassificationRemoteChanged, conflicts[0].Classification)

	// 5. Resolve with remote-wins policy
	resolver := &grcsync.ConflictResolver{DefaultPolicy: grcsync.ConflictPolicyRemoteWins}
	resolutions := resolver.ResolveAll(conflicts)
	assert.Len(t, resolutions, 1)
	assert.Equal(t, "remote", resolutions[0].Winner)

	// 6. Audit trail
	trail := grcsync.NewAuditTrail(t.TempDir(), log)
	auditLog := trail.NewAuditLog("test-user", []string{"test-provider"})
	auditLog.RecordPull("test-provider", "policy", pol.ID, "hash-old", "hash-new")
	auditLog.RecordConflictResolved("test-provider", "policy", pol.ID, "remote_wins", "hash-old", "hash-new")
	auditLog.Finalize()

	assert.Equal(t, 1, auditLog.Summary.Pulls)
	assert.Equal(t, 1, auditLog.Summary.ConflictsResolved)
	assert.Equal(t, 2, auditLog.Summary.TotalOperations)

	// 7. Save and load audit log
	require.NoError(t, trail.Save(auditLog))
	files, err := trail.List()
	require.NoError(t, err)
	assert.Len(t, files, 1)

	loaded, err := trail.Load(auditLog.RunID)
	require.NoError(t, err)
	assert.Equal(t, auditLog.RunID, loaded.RunID)
	assert.Equal(t, 2, len(loaded.Entries))

	// 8. Health check
	health := reg.HealthCheck(context.Background())
	assert.NoError(t, health["test-provider"])
}

// TestProviderPipeline_MultiProvider exercises multi-provider scenarios.
func TestProviderPipeline_MultiProvider(t *testing.T) {
	t.Parallel()

	reg := providers.NewProviderRegistry()

	// Provider A: 2 policies
	provA := testhelpers.NewStubDataProvider("provider-a")
	p1 := &domain.Policy{ID: "POL-001", Name: "Policy A1"}
	p2 := &domain.Policy{ID: "POL-002", Name: "Policy A2"}
	provA.Policies["POL-001"] = p1
	provA.Policies["POL-002"] = p2

	// Provider B: 1 policy
	provB := testhelpers.NewStubDataProvider("provider-b")
	p3 := &domain.Policy{ID: "POL-003", Name: "Policy B1"}
	provB.Policies["POL-003"] = p3

	require.NoError(t, reg.Register(provA))
	require.NoError(t, reg.Register(provB))

	// List returns sorted names
	names := reg.List()
	assert.Equal(t, []string{"provider-a", "provider-b"}, names)

	// Each provider has its own data
	pa, _ := reg.Get("provider-a")
	policiesA, countA, _ := pa.ListPolicies(context.Background(), interfaces.ListOptions{})
	assert.Equal(t, 2, countA)
	assert.Len(t, policiesA, 2)

	pb, _ := reg.Get("provider-b")
	policiesB, countB, _ := pb.ListPolicies(context.Background(), interfaces.ListOptions{})
	assert.Equal(t, 1, countB)
	assert.Len(t, policiesB, 1)
}

// TestProviderPipeline_ExternalIDLookup exercises the ExternalID round-trip.
func TestProviderPipeline_ExternalIDLookup(t *testing.T) {
	t.Parallel()

	stub := testhelpers.NewStubDataProvider("tugboat")

	// Policy with ExternalIDs
	pol := &domain.Policy{
		ID:          "POL-001",
		Name:        "Access Control Policy",
		ExternalIDs: map[string]string{"tugboat": "12345", "accountablehq": "pol-abc"},
		SyncMetadata: &domain.SyncMetadata{
			LastSyncTime: map[string]time.Time{"tugboat": time.Now()},
			ContentHash:  map[string]string{"tugboat": "sha256:abc123"},
		},
	}
	stub.Policies["POL-001"] = pol

	// Retrieve and verify ExternalIDs preserved
	retrieved, err := stub.GetPolicy(context.Background(), "POL-001")
	require.NoError(t, err)
	assert.Equal(t, "12345", retrieved.ExternalIDs["tugboat"])
	assert.Equal(t, "pol-abc", retrieved.ExternalIDs["accountablehq"])
	assert.Equal(t, "sha256:abc123", retrieved.SyncMetadata.ContentHash["tugboat"])
}
