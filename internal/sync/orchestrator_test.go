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
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/providers"
)

// --- Stub SyncProvider ---

type stubSyncProvider struct {
	name       string
	changeSet  *interfaces.ChangeSet
	detectErr  error
	policies   map[string]*domain.Policy
	controls   map[string]*domain.Control
	tasks      map[string]*domain.EvidenceTask
	pushed     []interface{} // track pushed entities
	getPolicyErr       error
	getControlErr      error
	getEvidenceTaskErr error
}

func newStubSyncProvider(name string) *stubSyncProvider {
	return &stubSyncProvider{
		name:     name,
		policies: make(map[string]*domain.Policy),
		controls: make(map[string]*domain.Control),
		tasks:    make(map[string]*domain.EvidenceTask),
	}
}

func (s *stubSyncProvider) Name() string { return s.name }

func (s *stubSyncProvider) Capabilities() interfaces.ProviderCapabilities {
	return interfaces.ProviderCapabilities{
		SupportsPolicies:      true,
		SupportsControls:      true,
		SupportsEvidenceTasks: true,
		SupportsWrite:         true,
		SupportsChangeDetect:  true,
	}
}

func (s *stubSyncProvider) TestConnection(_ context.Context) error { return nil }

func (s *stubSyncProvider) ListPolicies(_ context.Context, _ interfaces.ListOptions) ([]domain.Policy, int, error) {
	return nil, 0, nil
}

func (s *stubSyncProvider) GetPolicy(_ context.Context, id string) (*domain.Policy, error) {
	if s.getPolicyErr != nil {
		return nil, s.getPolicyErr
	}
	p, ok := s.policies[id]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", id)
	}
	return p, nil
}

func (s *stubSyncProvider) ListControls(_ context.Context, _ interfaces.ListOptions) ([]domain.Control, int, error) {
	return nil, 0, nil
}

func (s *stubSyncProvider) GetControl(_ context.Context, id string) (*domain.Control, error) {
	if s.getControlErr != nil {
		return nil, s.getControlErr
	}
	c, ok := s.controls[id]
	if !ok {
		return nil, fmt.Errorf("control not found: %s", id)
	}
	return c, nil
}

func (s *stubSyncProvider) ListEvidenceTasks(_ context.Context, _ interfaces.ListOptions) ([]domain.EvidenceTask, int, error) {
	return nil, 0, nil
}

func (s *stubSyncProvider) GetEvidenceTask(_ context.Context, id string) (*domain.EvidenceTask, error) {
	if s.getEvidenceTaskErr != nil {
		return nil, s.getEvidenceTaskErr
	}
	t, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("evidence task not found: %s", id)
	}
	return t, nil
}

func (s *stubSyncProvider) PushPolicy(_ context.Context, policy *domain.Policy) error {
	s.pushed = append(s.pushed, policy)
	return nil
}

func (s *stubSyncProvider) PushControl(_ context.Context, control *domain.Control) error {
	s.pushed = append(s.pushed, control)
	return nil
}

func (s *stubSyncProvider) PushEvidenceTask(_ context.Context, task *domain.EvidenceTask) error {
	s.pushed = append(s.pushed, task)
	return nil
}

func (s *stubSyncProvider) DeletePolicy(_ context.Context, _ string) error   { return nil }
func (s *stubSyncProvider) DeleteControl(_ context.Context, _ string) error  { return nil }
func (s *stubSyncProvider) DeleteEvidenceTask(_ context.Context, _ string) error { return nil }

func (s *stubSyncProvider) DetectChanges(_ context.Context, _ time.Time) (*interfaces.ChangeSet, error) {
	if s.detectErr != nil {
		return nil, s.detectErr
	}
	return s.changeSet, nil
}

func (s *stubSyncProvider) ResolveConflict(_ context.Context, _ interfaces.Conflict, _ interfaces.ConflictResolution) error {
	return fmt.Errorf("not implemented")
}

// --- Stub StorageService ---

type stubStorageService struct {
	policies map[string]*domain.Policy        // keyed by "provider:externalID"
	controls map[string]*domain.Control       // keyed by "provider:externalID"
	tasks    map[string]*domain.EvidenceTask  // keyed by "provider:externalID"
	saved    []interface{}                     // track all saved entities
}

func newStubStorageService() *stubStorageService {
	return &stubStorageService{
		policies: make(map[string]*domain.Policy),
		controls: make(map[string]*domain.Control),
		tasks:    make(map[string]*domain.EvidenceTask),
	}
}

func (s *stubStorageService) SavePolicy(p *domain.Policy) error {
	s.saved = append(s.saved, p)
	// Store by external IDs for lookup.
	for prov, eid := range p.ExternalIDs {
		s.policies[prov+":"+eid] = p
	}
	return nil
}

func (s *stubStorageService) GetPolicy(_ string) (*domain.Policy, error) { return nil, nil }
func (s *stubStorageService) GetPolicyByReferenceAndID(_, _ string) (*domain.Policy, error) {
	return nil, nil
}
func (s *stubStorageService) GetAllPolicies() ([]domain.Policy, error) { return nil, nil }
func (s *stubStorageService) GetPolicySummary() (*domain.PolicySummary, error) { return nil, nil }

func (s *stubStorageService) GetPolicyByExternalID(provider, externalID string) (*domain.Policy, error) {
	p, ok := s.policies[provider+":"+externalID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return p, nil
}

func (s *stubStorageService) SaveControl(c *domain.Control) error {
	s.saved = append(s.saved, c)
	for prov, eid := range c.ExternalIDs {
		s.controls[prov+":"+eid] = c
	}
	return nil
}

func (s *stubStorageService) GetControl(_ string) (*domain.Control, error) { return nil, nil }
func (s *stubStorageService) GetControlByReferenceAndID(_, _ string) (*domain.Control, error) {
	return nil, nil
}
func (s *stubStorageService) GetAllControls() ([]domain.Control, error) { return nil, nil }
func (s *stubStorageService) GetControlSummary() (*domain.ControlSummary, error) { return nil, nil }

func (s *stubStorageService) GetControlByExternalID(provider, externalID string) (*domain.Control, error) {
	c, ok := s.controls[provider+":"+externalID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return c, nil
}

func (s *stubStorageService) SaveEvidenceTask(t *domain.EvidenceTask) error {
	s.saved = append(s.saved, t)
	for prov, eid := range t.ExternalIDs {
		s.tasks[prov+":"+eid] = t
	}
	return nil
}

func (s *stubStorageService) GetEvidenceTask(_ string) (*domain.EvidenceTask, error) {
	return nil, nil
}
func (s *stubStorageService) GetEvidenceTaskByReferenceAndID(_, _ string) (*domain.EvidenceTask, error) {
	return nil, nil
}
func (s *stubStorageService) GetAllEvidenceTasks() ([]domain.EvidenceTask, error) { return nil, nil }
func (s *stubStorageService) GetEvidenceTaskSummary() (*domain.EvidenceTaskSummary, error) {
	return nil, nil
}

func (s *stubStorageService) GetEvidenceTaskByExternalID(provider, externalID string) (*domain.EvidenceTask, error) {
	t, ok := s.tasks[provider+":"+externalID]
	if !ok {
		return nil, fmt.Errorf("not found")
	}
	return t, nil
}

func (s *stubStorageService) SaveEvidenceRecord(_ *domain.EvidenceRecord) error { return nil }
func (s *stubStorageService) GetEvidenceRecord(_ string) (*domain.EvidenceRecord, error) {
	return nil, nil
}
func (s *stubStorageService) GetEvidenceRecordsByTaskID(_ string) ([]domain.EvidenceRecord, error) {
	return nil, nil
}

func (s *stubStorageService) GetStats() (map[string]interface{}, error) { return nil, nil }
func (s *stubStorageService) SetSyncTime(_ string, _ time.Time) error  { return nil }
func (s *stubStorageService) GetSyncTime(_ string) (time.Time, error)  { return time.Time{}, nil }
func (s *stubStorageService) Clear() error                             { return nil }

// --- Helper ---

func newTestOrchestrator(t *testing.T, storage *stubStorageService, policy ConflictPolicy, provs ...interfaces.DataProvider) *SyncOrchestrator {
	t.Helper()
	registry := providers.NewProviderRegistry()
	for _, p := range provs {
		if err := registry.Register(p); err != nil {
			t.Fatalf("register provider: %v", err)
		}
	}
	log, err := logger.NewTestLogger()
	if err != nil {
		t.Fatalf("create test logger: %v", err)
	}
	return NewSyncOrchestrator(registry, storage, policy, log)
}

// --- Tests ---

func TestSyncOrchestrator_PullNewEntities(t *testing.T) {
	t.Parallel()

	sp := newStubSyncProvider("test-provider")
	sp.changeSet = &interfaces.ChangeSet{
		Provider: "test-provider",
		Changes: []interfaces.ChangeEntry{
			{EntityType: "policy", EntityID: "pol-1", ChangeType: "created", Hash: "hash-a", ModifiedAt: time.Now()},
			{EntityType: "policy", EntityID: "pol-2", ChangeType: "created", Hash: "hash-b", ModifiedAt: time.Now()},
		},
	}
	sp.policies["pol-1"] = &domain.Policy{ID: "POL-0001", Name: "Policy A", ExternalIDs: map[string]string{"test-provider": "pol-1"}}
	sp.policies["pol-2"] = &domain.Policy{ID: "POL-0002", Name: "Policy B", ExternalIDs: map[string]string{"test-provider": "pol-2"}}

	storage := newStubStorageService()
	orch := newTestOrchestrator(t, storage, ConflictPolicyRemoteWins, sp)

	result, err := orch.SyncProvider(context.Background(), "test-provider", time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PullCount != 2 {
		t.Errorf("expected PullCount=2, got %d", result.PullCount)
	}
	if result.PushCount != 0 {
		t.Errorf("expected PushCount=0, got %d", result.PushCount)
	}
	if result.ConflictCount != 0 {
		t.Errorf("expected ConflictCount=0, got %d", result.ConflictCount)
	}
	if len(result.Errors) != 0 {
		t.Errorf("expected no errors, got %v", result.Errors)
	}
	if result.Duration <= 0 {
		t.Error("expected positive duration")
	}
	// Verify entities were saved to storage.
	if len(storage.saved) != 2 {
		t.Errorf("expected 2 saved entities, got %d", len(storage.saved))
	}
}

func TestSyncOrchestrator_NoChanges(t *testing.T) {
	t.Parallel()

	sp := newStubSyncProvider("test-provider")
	sp.changeSet = &interfaces.ChangeSet{
		Provider: "test-provider",
		Changes:  []interfaces.ChangeEntry{},
	}

	storage := newStubStorageService()
	orch := newTestOrchestrator(t, storage, ConflictPolicyRemoteWins, sp)

	result, err := orch.SyncProvider(context.Background(), "test-provider", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.PullCount != 0 {
		t.Errorf("expected PullCount=0, got %d", result.PullCount)
	}
	if result.PushCount != 0 {
		t.Errorf("expected PushCount=0, got %d", result.PushCount)
	}
	if result.ConflictCount != 0 {
		t.Errorf("expected ConflictCount=0, got %d", result.ConflictCount)
	}
}

func TestSyncOrchestrator_ConflictLocalWins(t *testing.T) {
	t.Parallel()

	sp := newStubSyncProvider("test-provider")
	sp.changeSet = &interfaces.ChangeSet{
		Provider: "test-provider",
		Changes: []interfaces.ChangeEntry{
			{EntityType: "policy", EntityID: "pol-1", ChangeType: "updated", Hash: "remote-hash", ModifiedAt: time.Now()},
		},
	}
	// The provider must also have the entity for push.
	sp.policies["pol-1"] = &domain.Policy{ID: "POL-0001", Name: "Policy A", ExternalIDs: map[string]string{"test-provider": "pol-1"}}

	// Pre-populate local storage with a policy that has both local and provider hash changed.
	storage := newStubStorageService()
	localPolicy := &domain.Policy{
		ID:          "POL-0001",
		Name:        "Policy A (local edit)",
		ExternalIDs: map[string]string{"test-provider": "pol-1"},
		SyncMetadata: &domain.SyncMetadata{
			ContentHash: map[string]string{
				"test-provider": "original-hash",
				"local":         "local-modified-hash",
			},
			LastSyncTime: map[string]time.Time{
				"test-provider": time.Now().Add(-1 * time.Hour),
			},
		},
	}
	_ = storage.SavePolicy(localPolicy)
	storage.saved = nil // reset saved tracker

	orch := newTestOrchestrator(t, storage, ConflictPolicyLocalWins, sp)

	result, err := orch.SyncProvider(context.Background(), "test-provider", time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ConflictCount != 1 {
		t.Errorf("expected ConflictCount=1, got %d", result.ConflictCount)
	}
	if result.ResolvedCount != 1 {
		t.Errorf("expected ResolvedCount=1, got %d", result.ResolvedCount)
	}
	if result.PullCount != 0 {
		t.Errorf("expected PullCount=0 (local wins), got %d", result.PullCount)
	}
	if result.PushCount != 1 {
		t.Errorf("expected PushCount=1 (local wins pushes), got %d", result.PushCount)
	}
	if result.ManualCount != 0 {
		t.Errorf("expected ManualCount=0, got %d", result.ManualCount)
	}
}

func TestSyncOrchestrator_ConflictRemoteWins(t *testing.T) {
	t.Parallel()

	sp := newStubSyncProvider("test-provider")
	sp.changeSet = &interfaces.ChangeSet{
		Provider: "test-provider",
		Changes: []interfaces.ChangeEntry{
			{EntityType: "control", EntityID: "ctrl-1", ChangeType: "updated", Hash: "remote-hash", ModifiedAt: time.Now()},
		},
	}
	sp.controls["ctrl-1"] = &domain.Control{ID: "CC-01", Name: "Control A", ExternalIDs: map[string]string{"test-provider": "ctrl-1"}}

	// Pre-populate local storage with a control that has both changed.
	storage := newStubStorageService()
	localCtrl := &domain.Control{
		ID:          "CC-01",
		Name:        "Control A (local edit)",
		ExternalIDs: map[string]string{"test-provider": "ctrl-1"},
		SyncMetadata: &domain.SyncMetadata{
			ContentHash: map[string]string{
				"test-provider": "original-hash",
				"local":         "local-modified-hash",
			},
			LastSyncTime: map[string]time.Time{
				"test-provider": time.Now().Add(-1 * time.Hour),
			},
		},
	}
	_ = storage.SaveControl(localCtrl)
	storage.saved = nil // reset

	orch := newTestOrchestrator(t, storage, ConflictPolicyRemoteWins, sp)

	result, err := orch.SyncProvider(context.Background(), "test-provider", time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ConflictCount != 1 {
		t.Errorf("expected ConflictCount=1, got %d", result.ConflictCount)
	}
	if result.ResolvedCount != 1 {
		t.Errorf("expected ResolvedCount=1, got %d", result.ResolvedCount)
	}
	if result.PullCount != 1 {
		t.Errorf("expected PullCount=1 (remote wins pulls), got %d", result.PullCount)
	}
	if result.PushCount != 0 {
		t.Errorf("expected PushCount=0, got %d", result.PushCount)
	}
}

func TestSyncOrchestrator_ConflictManual(t *testing.T) {
	t.Parallel()

	sp := newStubSyncProvider("test-provider")
	sp.changeSet = &interfaces.ChangeSet{
		Provider: "test-provider",
		Changes: []interfaces.ChangeEntry{
			{EntityType: "policy", EntityID: "pol-1", ChangeType: "updated", Hash: "remote-hash", ModifiedAt: time.Now()},
		},
	}
	sp.policies["pol-1"] = &domain.Policy{ID: "POL-0001", Name: "Policy A", ExternalIDs: map[string]string{"test-provider": "pol-1"}}

	storage := newStubStorageService()
	localPolicy := &domain.Policy{
		ID:          "POL-0001",
		Name:        "Policy A (local edit)",
		ExternalIDs: map[string]string{"test-provider": "pol-1"},
		SyncMetadata: &domain.SyncMetadata{
			ContentHash: map[string]string{
				"test-provider": "original-hash",
				"local":         "local-modified-hash",
			},
			LastSyncTime: map[string]time.Time{
				"test-provider": time.Now().Add(-1 * time.Hour),
			},
		},
	}
	_ = storage.SavePolicy(localPolicy)
	storage.saved = nil

	orch := newTestOrchestrator(t, storage, ConflictPolicyManual, sp)

	result, err := orch.SyncProvider(context.Background(), "test-provider", time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.ConflictCount != 1 {
		t.Errorf("expected ConflictCount=1, got %d", result.ConflictCount)
	}
	if result.ManualCount != 1 {
		t.Errorf("expected ManualCount=1, got %d", result.ManualCount)
	}
	if result.ResolvedCount != 0 {
		t.Errorf("expected ResolvedCount=0, got %d", result.ResolvedCount)
	}
	if result.PullCount != 0 {
		t.Errorf("expected PullCount=0, got %d", result.PullCount)
	}
	if result.PushCount != 0 {
		t.Errorf("expected PushCount=0, got %d", result.PushCount)
	}
}

func TestSyncOrchestrator_ProviderError(t *testing.T) {
	t.Parallel()

	sp := newStubSyncProvider("test-provider")
	sp.detectErr = fmt.Errorf("connection refused")

	storage := newStubStorageService()
	orch := newTestOrchestrator(t, storage, ConflictPolicyRemoteWins, sp)

	_, err := orch.SyncProvider(context.Background(), "test-provider", time.Time{})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if got := err.Error(); got != "detect changes: connection refused" {
		t.Errorf("unexpected error message: %s", got)
	}
}

func TestSyncOrchestrator_SyncAll(t *testing.T) {
	t.Parallel()

	sp1 := newStubSyncProvider("provider-a")
	sp1.changeSet = &interfaces.ChangeSet{
		Provider: "provider-a",
		Changes: []interfaces.ChangeEntry{
			{EntityType: "policy", EntityID: "pol-1", ChangeType: "created", Hash: "hash-1", ModifiedAt: time.Now()},
		},
	}
	sp1.policies["pol-1"] = &domain.Policy{ID: "POL-0001", Name: "Policy 1", ExternalIDs: map[string]string{"provider-a": "pol-1"}}

	sp2 := newStubSyncProvider("provider-b")
	sp2.changeSet = &interfaces.ChangeSet{
		Provider: "provider-b",
		Changes: []interfaces.ChangeEntry{
			{EntityType: "control", EntityID: "ctrl-1", ChangeType: "created", Hash: "hash-2", ModifiedAt: time.Now()},
			{EntityType: "control", EntityID: "ctrl-2", ChangeType: "created", Hash: "hash-3", ModifiedAt: time.Now()},
		},
	}
	sp2.controls["ctrl-1"] = &domain.Control{ID: "CC-01", Name: "Control 1", ExternalIDs: map[string]string{"provider-b": "ctrl-1"}}
	sp2.controls["ctrl-2"] = &domain.Control{ID: "CC-02", Name: "Control 2", ExternalIDs: map[string]string{"provider-b": "ctrl-2"}}

	storage := newStubStorageService()
	orch := newTestOrchestrator(t, storage, ConflictPolicyRemoteWins, sp1, sp2)

	results, err := orch.SyncAll(context.Background(), time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}

	// Results are sorted by provider name.
	if results[0].Provider != "provider-a" {
		t.Errorf("expected first result for provider-a, got %s", results[0].Provider)
	}
	if results[0].PullCount != 1 {
		t.Errorf("provider-a: expected PullCount=1, got %d", results[0].PullCount)
	}
	if results[1].Provider != "provider-b" {
		t.Errorf("expected second result for provider-b, got %s", results[1].Provider)
	}
	if results[1].PullCount != 2 {
		t.Errorf("provider-b: expected PullCount=2, got %d", results[1].PullCount)
	}
}

func TestSyncOrchestrator_SyncAll_NoSyncProviders(t *testing.T) {
	t.Parallel()

	// Register a DataProvider-only stub (not a SyncProvider).
	storage := newStubStorageService()
	registry := providers.NewProviderRegistry()
	if err := registry.Register(&dataOnlyProvider{name: "readonly"}); err != nil {
		t.Fatalf("register: %v", err)
	}
	log, err := logger.NewTestLogger()
	if err != nil {
		t.Fatalf("create test logger: %v", err)
	}
	orch := NewSyncOrchestrator(registry, storage, ConflictPolicyRemoteWins, log)

	results, err := orch.SyncAll(context.Background(), time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("expected 0 results for no sync providers, got %d", len(results))
	}
}

func TestSyncOrchestrator_PullEntityError(t *testing.T) {
	t.Parallel()

	sp := newStubSyncProvider("test-provider")
	sp.changeSet = &interfaces.ChangeSet{
		Provider: "test-provider",
		Changes: []interfaces.ChangeEntry{
			{EntityType: "policy", EntityID: "pol-good", ChangeType: "created", Hash: "hash-a", ModifiedAt: time.Now()},
			{EntityType: "policy", EntityID: "pol-bad", ChangeType: "created", Hash: "hash-b", ModifiedAt: time.Now()},
			{EntityType: "control", EntityID: "ctrl-good", ChangeType: "created", Hash: "hash-c", ModifiedAt: time.Now()},
		},
	}
	// Only add some entities — pol-bad is missing, causing a pull error.
	sp.policies["pol-good"] = &domain.Policy{ID: "POL-0001", Name: "Good Policy", ExternalIDs: map[string]string{"test-provider": "pol-good"}}
	sp.controls["ctrl-good"] = &domain.Control{ID: "CC-01", Name: "Good Control", ExternalIDs: map[string]string{"test-provider": "ctrl-good"}}

	storage := newStubStorageService()
	orch := newTestOrchestrator(t, storage, ConflictPolicyRemoteWins, sp)

	result, err := orch.SyncProvider(context.Background(), "test-provider", time.Time{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Two entities should succeed, one should fail.
	if result.PullCount != 2 {
		t.Errorf("expected PullCount=2, got %d", result.PullCount)
	}
	if len(result.Errors) != 1 {
		t.Errorf("expected 1 error, got %d: %v", len(result.Errors), result.Errors)
	}
}

// --- DataProvider-only stub (does not implement SyncProvider) ---

type dataOnlyProvider struct {
	name string
}

func (d *dataOnlyProvider) Name() string                        { return d.name }
func (d *dataOnlyProvider) Capabilities() interfaces.ProviderCapabilities {
	return interfaces.ProviderCapabilities{SupportsPolicies: true, SupportsControls: true, SupportsEvidenceTasks: true}
}
func (d *dataOnlyProvider) TestConnection(_ context.Context) error { return nil }
func (d *dataOnlyProvider) ListPolicies(_ context.Context, _ interfaces.ListOptions) ([]domain.Policy, int, error) {
	return nil, 0, nil
}
func (d *dataOnlyProvider) GetPolicy(_ context.Context, _ string) (*domain.Policy, error) {
	return nil, nil
}
func (d *dataOnlyProvider) ListControls(_ context.Context, _ interfaces.ListOptions) ([]domain.Control, int, error) {
	return nil, 0, nil
}
func (d *dataOnlyProvider) GetControl(_ context.Context, _ string) (*domain.Control, error) {
	return nil, nil
}
func (d *dataOnlyProvider) ListEvidenceTasks(_ context.Context, _ interfaces.ListOptions) ([]domain.EvidenceTask, int, error) {
	return nil, 0, nil
}
func (d *dataOnlyProvider) GetEvidenceTask(_ context.Context, _ string) (*domain.EvidenceTask, error) {
	return nil, nil
}
