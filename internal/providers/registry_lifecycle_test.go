// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package providers

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGetSyncProvider_NotFound covers the missing branch in GetSyncProvider.
func TestGetSyncProvider_NotFound(t *testing.T) {
	t.Parallel()
	reg := NewProviderRegistry()
	_, err := reg.GetSyncProvider("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "provider not found")
}

// TestLifecycle_RegisterUseRemoveReRegister verifies the full lifecycle.
func TestLifecycle_RegisterUseRemoveReRegister(t *testing.T) {
	t.Parallel()
	reg := NewProviderRegistry()

	// Register
	stub := testhelpers.NewStubDataProvider("lifecycle-test")
	p := testhelpers.SamplePolicy()
	stub.Policies[p.ID] = p
	require.NoError(t, reg.Register(stub))
	assert.Equal(t, 1, reg.Count())
	assert.True(t, reg.Has("lifecycle-test"))

	// Use — list policies through the provider
	provider, err := reg.Get("lifecycle-test")
	require.NoError(t, err)
	policies, count, err := provider.ListPolicies(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	assert.Len(t, policies, 1)

	// Remove
	reg.Remove("lifecycle-test")
	assert.Equal(t, 0, reg.Count())
	assert.False(t, reg.Has("lifecycle-test"))
	_, err = reg.Get("lifecycle-test")
	assert.Error(t, err)

	// Re-register with different data
	stub2 := testhelpers.NewStubDataProvider("lifecycle-test")
	p2 := testhelpers.SamplePolicy()
	p2.ID = "POL-9999"
	p2.Name = "Re-registered Policy"
	stub2.Policies[p2.ID] = p2
	require.NoError(t, reg.Register(stub2))
	assert.Equal(t, 1, reg.Count())

	provider2, err := reg.Get("lifecycle-test")
	require.NoError(t, err)
	policies2, _, err := provider2.ListPolicies(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, "Re-registered Policy", policies2[0].Name)
}

// TestLifecycle_HealthCheckDegradation verifies health transitions.
func TestLifecycle_HealthCheckDegradation(t *testing.T) {
	t.Parallel()
	reg := NewProviderRegistry()

	healthy := testhelpers.NewStubDataProvider("healthy")
	degraded := testhelpers.NewStubDataProvider("degraded")
	degraded.ConnError = fmt.Errorf("connection timeout")

	require.NoError(t, reg.Register(healthy))
	require.NoError(t, reg.Register(degraded))

	results := reg.HealthCheck(context.Background())
	assert.NoError(t, results["healthy"])
	assert.Error(t, results["degraded"])
	assert.Contains(t, results["degraded"].Error(), "connection timeout")

	// "Fix" the degraded provider
	degraded.ConnError = nil
	results2 := reg.HealthCheck(context.Background())
	assert.NoError(t, results2["healthy"])
	assert.NoError(t, results2["degraded"])
}

// TestLifecycle_SyncProviderUpgrade tests registering a DataProvider then
// upgrading to SyncProvider.
func TestLifecycle_SyncProviderUpgrade(t *testing.T) {
	t.Parallel()
	reg := NewProviderRegistry()

	// Register as DataProvider only
	dataOnly := testhelpers.NewStubDataProvider("data-only")
	require.NoError(t, reg.Register(dataOnly))

	// Cannot get as SyncProvider
	_, err := reg.GetSyncProvider("data-only")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not implement SyncProvider")

	// Remove and re-register as SyncProvider
	reg.Remove("data-only")

	syncProvider := &stubSyncForLifecycle{
		StubDataProvider: testhelpers.NewStubDataProvider("data-only"),
	}
	require.NoError(t, reg.Register(syncProvider))

	sp, err := reg.GetSyncProvider("data-only")
	require.NoError(t, err)
	assert.Equal(t, "data-only", sp.Name())

	// ListSyncProviders includes it
	syncNames := reg.ListSyncProviders()
	assert.Contains(t, syncNames, "data-only")
}

// TestLifecycle_MultiProviderOrdering verifies deterministic listing.
func TestLifecycle_MultiProviderOrdering(t *testing.T) {
	t.Parallel()
	reg := NewProviderRegistry()

	// Register in non-alphabetical order
	for _, name := range []string{"zebra", "alpha", "middle", "beta"} {
		require.NoError(t, reg.Register(testhelpers.NewStubDataProvider(name)))
	}

	names := reg.List()
	assert.Equal(t, []string{"alpha", "beta", "middle", "zebra"}, names)

	// Remove one, verify ordering maintained
	reg.Remove("middle")
	names2 := reg.List()
	assert.Equal(t, []string{"alpha", "beta", "zebra"}, names2)
}

// TestLifecycle_EmptyRegistryOperations verifies all ops work on empty registry.
func TestLifecycle_EmptyRegistryOperations(t *testing.T) {
	t.Parallel()
	reg := NewProviderRegistry()

	assert.Equal(t, 0, reg.Count())
	assert.False(t, reg.Has("anything"))
	assert.Empty(t, reg.List())
	assert.Empty(t, reg.ListSyncProviders())

	_, err := reg.Get("nope")
	assert.Error(t, err)

	_, err = reg.GetSyncProvider("nope")
	assert.Error(t, err)

	// Remove on empty is a no-op
	reg.Remove("nonexistent")
	assert.Equal(t, 0, reg.Count())

	// HealthCheck on empty returns empty map
	results := reg.HealthCheck(context.Background())
	assert.Empty(t, results)
}

// stubSyncForLifecycle is a minimal SyncProvider for lifecycle tests.
type stubSyncForLifecycle struct {
	*testhelpers.StubDataProvider
}

func (s *stubSyncForLifecycle) PushPolicy(ctx context.Context, policy *domain.Policy) error {
	return nil
}
func (s *stubSyncForLifecycle) PushControl(ctx context.Context, control *domain.Control) error {
	return nil
}
func (s *stubSyncForLifecycle) PushEvidenceTask(ctx context.Context, task *domain.EvidenceTask) error {
	return nil
}
func (s *stubSyncForLifecycle) DeletePolicy(ctx context.Context, id string) error { return nil }
func (s *stubSyncForLifecycle) DeleteControl(ctx context.Context, id string) error { return nil }
func (s *stubSyncForLifecycle) DeleteEvidenceTask(ctx context.Context, id string) error {
	return nil
}
func (s *stubSyncForLifecycle) DetectChanges(ctx context.Context, since time.Time) (*interfaces.ChangeSet, error) {
	return &interfaces.ChangeSet{Provider: s.Name()}, nil
}
