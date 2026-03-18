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

package interfaces_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/testhelpers"
)

// ---------------------------------------------------------------------------
// Compile-time interface satisfaction
// ---------------------------------------------------------------------------

var _ interfaces.DataProvider = (*testhelpers.StubDataProvider)(nil)
var _ interfaces.DataProvider = (*stubSyncProvider)(nil)
var _ interfaces.SyncProvider = (*stubSyncProvider)(nil)

// ---------------------------------------------------------------------------
// stubSyncProvider — test-local SyncProvider for contract verification
// ---------------------------------------------------------------------------

type stubSyncProvider struct {
	*testhelpers.StubDataProvider
	pushedPolicies []domain.Policy
	pushedControls []domain.Control
	pushedTasks    []domain.EvidenceTask
	deletedIDs     []string
	changeSet      *interfaces.ChangeSet
	pushErr        error
	deleteErr      error
	detectErr      error
}

func newStubSyncProvider(name string) *stubSyncProvider {
	return &stubSyncProvider{
		StubDataProvider: testhelpers.NewStubDataProvider(name),
	}
}

func (s *stubSyncProvider) PushPolicy(_ context.Context, policy *domain.Policy) error {
	if s.pushErr != nil {
		return s.pushErr
	}
	s.pushedPolicies = append(s.pushedPolicies, *policy)
	s.Policies[policy.ID] = policy
	return nil
}

func (s *stubSyncProvider) PushControl(_ context.Context, control *domain.Control) error {
	if s.pushErr != nil {
		return s.pushErr
	}
	s.pushedControls = append(s.pushedControls, *control)
	s.Controls[control.ID] = control
	return nil
}

func (s *stubSyncProvider) PushEvidenceTask(_ context.Context, task *domain.EvidenceTask) error {
	if s.pushErr != nil {
		return s.pushErr
	}
	s.pushedTasks = append(s.pushedTasks, *task)
	s.Tasks[task.ID] = task
	return nil
}

func (s *stubSyncProvider) DeletePolicy(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedIDs = append(s.deletedIDs, id)
	delete(s.Policies, id)
	return nil
}

func (s *stubSyncProvider) DeleteControl(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedIDs = append(s.deletedIDs, id)
	delete(s.Controls, id)
	return nil
}

func (s *stubSyncProvider) DeleteEvidenceTask(_ context.Context, id string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deletedIDs = append(s.deletedIDs, id)
	delete(s.Tasks, id)
	return nil
}

func (s *stubSyncProvider) DetectChanges(_ context.Context, _ time.Time) (*interfaces.ChangeSet, error) {
	if s.detectErr != nil {
		return nil, s.detectErr
	}
	return s.changeSet, nil
}

// ---------------------------------------------------------------------------
// ListOptions tests
// ---------------------------------------------------------------------------

func TestListOptionsZeroValue(t *testing.T) {
	t.Parallel()
	var opts interfaces.ListOptions
	if opts.Page != 0 {
		t.Errorf("expected Page 0, got %d", opts.Page)
	}
	if opts.PageSize != 0 {
		t.Errorf("expected PageSize 0, got %d", opts.PageSize)
	}
	if opts.Framework != "" {
		t.Errorf("expected empty Framework, got %q", opts.Framework)
	}
	if opts.Status != "" {
		t.Errorf("expected empty Status, got %q", opts.Status)
	}
	if opts.Category != "" {
		t.Errorf("expected empty Category, got %q", opts.Category)
	}
}

func TestListOptionsConstruction(t *testing.T) {
	t.Parallel()
	opts := interfaces.ListOptions{
		Page:      2,
		PageSize:  25,
		Framework: "SOC2",
		Status:    "active",
		Category:  "Infrastructure",
	}
	if opts.Page != 2 {
		t.Errorf("expected Page 2, got %d", opts.Page)
	}
	if opts.PageSize != 25 {
		t.Errorf("expected PageSize 25, got %d", opts.PageSize)
	}
	if opts.Framework != "SOC2" {
		t.Errorf("expected Framework SOC2, got %q", opts.Framework)
	}
}

// ---------------------------------------------------------------------------
// ChangeSet / ChangeEntry tests
// ---------------------------------------------------------------------------

func TestChangeSetConstruction(t *testing.T) {
	t.Parallel()
	now := time.Now()
	cs := interfaces.ChangeSet{
		Provider:   "tugboat",
		Since:      now.Add(-24 * time.Hour),
		DetectedAt: now,
		Changes: []interfaces.ChangeEntry{
			{
				EntityType: "policy",
				EntityID:   "POL-0001",
				ChangeType: "updated",
				Hash:       "abc123",
				ModifiedAt: now.Add(-1 * time.Hour),
			},
		},
	}
	if cs.Provider != "tugboat" {
		t.Errorf("expected provider tugboat, got %q", cs.Provider)
	}
	if len(cs.Changes) != 1 {
		t.Fatalf("expected 1 change, got %d", len(cs.Changes))
	}
	entry := cs.Changes[0]
	if entry.EntityType != "policy" {
		t.Errorf("expected entity type policy, got %q", entry.EntityType)
	}
	if entry.EntityID != "POL-0001" {
		t.Errorf("expected entity ID POL-0001, got %q", entry.EntityID)
	}
	if entry.ChangeType != "updated" {
		t.Errorf("expected change type updated, got %q", entry.ChangeType)
	}
	if entry.Hash != "abc123" {
		t.Errorf("expected hash abc123, got %q", entry.Hash)
	}
}

func TestChangeSetEmptyChanges(t *testing.T) {
	t.Parallel()
	cs := interfaces.ChangeSet{
		Provider: "test",
	}
	if len(cs.Changes) != 0 {
		t.Errorf("expected 0 changes, got %d", len(cs.Changes))
	}
}

// ---------------------------------------------------------------------------
// StubDataProvider tests — list, get, filtering
// ---------------------------------------------------------------------------

func TestStubDataProviderName(t *testing.T) {
	t.Parallel()
	p := testhelpers.NewStubDataProvider("test-provider")
	if p.Name() != "test-provider" {
		t.Errorf("expected name test-provider, got %q", p.Name())
	}
}

func TestStubDataProviderTestConnection(t *testing.T) {
	t.Parallel()
	p := testhelpers.NewStubDataProvider("test")
	if err := p.TestConnection(context.Background()); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestStubDataProviderConnectionFailure(t *testing.T) {
	t.Parallel()
	p := testhelpers.NewStubDataProvider("test")
	p.ConnError = fmt.Errorf("connection refused")
	if err := p.TestConnection(context.Background()); err == nil {
		t.Error("expected connection error, got nil")
	}
}

func TestStubDataProviderConnectionErrorPropagates(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := testhelpers.NewStubDataProvider("test")
	p.ConnError = fmt.Errorf("timeout")

	if _, _, err := p.ListPolicies(ctx, interfaces.ListOptions{}); err == nil {
		t.Error("expected error from ListPolicies")
	}
	if _, _, err := p.ListControls(ctx, interfaces.ListOptions{}); err == nil {
		t.Error("expected error from ListControls")
	}
	if _, _, err := p.ListEvidenceTasks(ctx, interfaces.ListOptions{}); err == nil {
		t.Error("expected error from ListEvidenceTasks")
	}
	if _, err := p.GetPolicy(ctx, "1"); err == nil {
		t.Error("expected error from GetPolicy")
	}
	if _, err := p.GetControl(ctx, "1"); err == nil {
		t.Error("expected error from GetControl")
	}
	if _, err := p.GetEvidenceTask(ctx, "1"); err == nil {
		t.Error("expected error from GetEvidenceTask")
	}
}

func TestStubDataProviderListPolicies(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := testhelpers.NewStubDataProvider("test")
	p.Policies["1"] = &domain.Policy{ID: "1", Name: "Access Control", Framework: "SOC2"}
	p.Policies["2"] = &domain.Policy{ID: "2", Name: "Data Protection", Framework: "ISO27001"}

	policies, total, err := p.ListPolicies(ctx, interfaces.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(policies) != 2 {
		t.Errorf("expected 2 policies, got %d", len(policies))
	}
}

func TestStubDataProviderListPoliciesFrameworkFilter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := testhelpers.NewStubDataProvider("test")
	p.Policies["1"] = &domain.Policy{ID: "1", Framework: "SOC2"}
	p.Policies["2"] = &domain.Policy{ID: "2", Framework: "ISO27001"}

	policies, total, err := p.ListPolicies(ctx, interfaces.ListOptions{Framework: "SOC2"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(policies) != 1 {
		t.Errorf("expected 1 policy, got %d", len(policies))
	}
	if policies[0].Framework != "SOC2" {
		t.Errorf("expected SOC2, got %q", policies[0].Framework)
	}
}

func TestStubDataProviderGetPolicy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := testhelpers.NewStubDataProvider("test")
	p.Policies["POL-1"] = &domain.Policy{ID: "POL-1", Name: "Test Policy"}

	got, err := p.GetPolicy(ctx, "POL-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Test Policy" {
		t.Errorf("expected Test Policy, got %q", got.Name)
	}
}

func TestStubDataProviderGetPolicyNotFound(t *testing.T) {
	t.Parallel()
	p := testhelpers.NewStubDataProvider("test")
	_, err := p.GetPolicy(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected not found error, got nil")
	}
}

func TestStubDataProviderListControls(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := testhelpers.NewStubDataProvider("test")
	p.Controls["1"] = &domain.Control{ID: "1", Name: "CC6.1", Framework: "SOC2"}
	p.Controls["2"] = &domain.Control{ID: "2", Name: "A.9.1", Framework: "ISO27001"}

	controls, total, err := p.ListControls(ctx, interfaces.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(controls) != 2 {
		t.Errorf("expected 2 controls, got %d", len(controls))
	}
}

func TestStubDataProviderGetControlNotFound(t *testing.T) {
	t.Parallel()
	p := testhelpers.NewStubDataProvider("test")
	_, err := p.GetControl(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected not found error, got nil")
	}
}

func TestStubDataProviderListEvidenceTasks(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := testhelpers.NewStubDataProvider("test")
	p.Tasks["1"] = &domain.EvidenceTask{ID: "1", Name: "GitHub Permissions", Status: "pending"}
	p.Tasks["2"] = &domain.EvidenceTask{ID: "2", Name: "Firewall Config", Status: "completed"}

	tasks, total, err := p.ListEvidenceTasks(ctx, interfaces.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}

func TestStubDataProviderListEvidenceTasksStatusFilter(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := testhelpers.NewStubDataProvider("test")
	p.Tasks["1"] = &domain.EvidenceTask{ID: "1", Status: "pending"}
	p.Tasks["2"] = &domain.EvidenceTask{ID: "2", Status: "completed"}

	tasks, total, err := p.ListEvidenceTasks(ctx, interfaces.ListOptions{Status: "pending"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("expected total 1, got %d", total)
	}
	if len(tasks) != 1 || tasks[0].Status != "pending" {
		t.Errorf("expected 1 pending task, got %v", tasks)
	}
}

func TestStubDataProviderGetEvidenceTaskNotFound(t *testing.T) {
	t.Parallel()
	p := testhelpers.NewStubDataProvider("test")
	_, err := p.GetEvidenceTask(context.Background(), "nonexistent")
	if err == nil {
		t.Error("expected not found error, got nil")
	}
}

// ---------------------------------------------------------------------------
// Pagination tests
// ---------------------------------------------------------------------------

func TestStubDataProviderPagination(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := testhelpers.NewStubDataProvider("test")
	for i := 1; i <= 5; i++ {
		id := fmt.Sprintf("%d", i)
		p.Policies[id] = &domain.Policy{ID: id, Name: fmt.Sprintf("Policy %d", i)}
	}

	// Page 1, size 2
	page1, total, err := p.ListPolicies(ctx, interfaces.ListOptions{Page: 1, PageSize: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(page1) != 2 {
		t.Errorf("expected 2 items on page 1, got %d", len(page1))
	}

	// Page 2, size 2
	page2, total, err := p.ListPolicies(ctx, interfaces.ListOptions{Page: 2, PageSize: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(page2) != 2 {
		t.Errorf("expected 2 items on page 2, got %d", len(page2))
	}

	// Page 3, size 2 (last page, partial)
	page3, total, err := p.ListPolicies(ctx, interfaces.ListOptions{Page: 3, PageSize: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(page3) != 1 {
		t.Errorf("expected 1 item on page 3, got %d", len(page3))
	}

	// Page 4, size 2 (beyond data)
	page4, total, err := p.ListPolicies(ctx, interfaces.ListOptions{Page: 4, PageSize: 2})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 5 {
		t.Errorf("expected total 5, got %d", total)
	}
	if len(page4) != 0 {
		t.Errorf("expected 0 items on page 4, got %d", len(page4))
	}
}

func TestStubDataProviderPaginationZeroPageSize(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := testhelpers.NewStubDataProvider("test")
	p.Policies["1"] = &domain.Policy{ID: "1"}
	p.Policies["2"] = &domain.Policy{ID: "2"}

	// PageSize 0 means return all
	all, total, err := p.ListPolicies(ctx, interfaces.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("expected total 2, got %d", total)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 items, got %d", len(all))
	}
}

func TestStubDataProviderEmptyResults(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := testhelpers.NewStubDataProvider("test")

	policies, total, err := p.ListPolicies(ctx, interfaces.ListOptions{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 0 {
		t.Errorf("expected total 0, got %d", total)
	}
	if len(policies) != 0 {
		t.Errorf("expected 0 policies, got %d", len(policies))
	}
}

// ---------------------------------------------------------------------------
// SyncProvider tests — push, delete, detect changes
// ---------------------------------------------------------------------------

func TestSyncProviderEmbeddsDataProvider(t *testing.T) {
	t.Parallel()
	sp := newStubSyncProvider("sync-test")
	// SyncProvider can be used as DataProvider
	var dp interfaces.DataProvider = sp
	if dp.Name() != "sync-test" {
		t.Errorf("expected name sync-test, got %q", dp.Name())
	}
}

func TestSyncProviderPushPolicy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	sp := newStubSyncProvider("sync-test")

	policy := &domain.Policy{ID: "POL-1", Name: "Test Policy"}
	if err := sp.PushPolicy(ctx, policy); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sp.pushedPolicies) != 1 {
		t.Errorf("expected 1 pushed policy, got %d", len(sp.pushedPolicies))
	}

	// Verify it's now retrievable
	got, err := sp.GetPolicy(ctx, "POL-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name != "Test Policy" {
		t.Errorf("expected Test Policy, got %q", got.Name)
	}
}

func TestSyncProviderPushControl(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	sp := newStubSyncProvider("sync-test")

	control := &domain.Control{ID: "CC-01", Name: "Test Control"}
	if err := sp.PushControl(ctx, control); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sp.pushedControls) != 1 {
		t.Errorf("expected 1 pushed control, got %d", len(sp.pushedControls))
	}
}

func TestSyncProviderPushEvidenceTask(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	sp := newStubSyncProvider("sync-test")

	task := &domain.EvidenceTask{ID: "ET-001", Name: "Test Task"}
	if err := sp.PushEvidenceTask(ctx, task); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(sp.pushedTasks) != 1 {
		t.Errorf("expected 1 pushed task, got %d", len(sp.pushedTasks))
	}
}

func TestSyncProviderDeletePolicy(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	sp := newStubSyncProvider("sync-test")
	sp.Policies["POL-1"] = &domain.Policy{ID: "POL-1"}

	if err := sp.DeletePolicy(ctx, "POL-1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := sp.GetPolicy(ctx, "POL-1")
	if err == nil {
		t.Error("expected not found error after delete, got nil")
	}
}

func TestSyncProviderDeleteControl(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	sp := newStubSyncProvider("sync-test")
	sp.Controls["CC-01"] = &domain.Control{ID: "CC-01"}

	if err := sp.DeleteControl(ctx, "CC-01"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := sp.GetControl(ctx, "CC-01")
	if err == nil {
		t.Error("expected not found error after delete, got nil")
	}
}

func TestSyncProviderDeleteEvidenceTask(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	sp := newStubSyncProvider("sync-test")
	sp.Tasks["ET-001"] = &domain.EvidenceTask{ID: "ET-001"}

	if err := sp.DeleteEvidenceTask(ctx, "ET-001"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_, err := sp.GetEvidenceTask(ctx, "ET-001")
	if err == nil {
		t.Error("expected not found error after delete, got nil")
	}
}

func TestSyncProviderDetectChanges(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	now := time.Now()
	sp := newStubSyncProvider("sync-test")
	sp.changeSet = &interfaces.ChangeSet{
		Provider:   "sync-test",
		Since:      now.Add(-1 * time.Hour),
		DetectedAt: now,
		Changes: []interfaces.ChangeEntry{
			{
				EntityType: "policy",
				EntityID:   "POL-1",
				ChangeType: "created",
				Hash:       "hash1",
				ModifiedAt: now.Add(-30 * time.Minute),
			},
			{
				EntityType: "control",
				EntityID:   "CC-01",
				ChangeType: "updated",
				Hash:       "hash2",
				ModifiedAt: now.Add(-15 * time.Minute),
			},
		},
	}

	cs, err := sp.DetectChanges(ctx, now.Add(-1*time.Hour))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cs.Provider != "sync-test" {
		t.Errorf("expected provider sync-test, got %q", cs.Provider)
	}
	if len(cs.Changes) != 2 {
		t.Fatalf("expected 2 changes, got %d", len(cs.Changes))
	}
}

func TestSyncProviderDetectChangesError(t *testing.T) {
	t.Parallel()
	sp := newStubSyncProvider("sync-test")
	sp.detectErr = fmt.Errorf("api timeout")

	_, err := sp.DetectChanges(context.Background(), time.Now())
	if err == nil {
		t.Error("expected error, got nil")
	}
}

func TestSyncProviderPushError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	sp := newStubSyncProvider("sync-test")
	sp.pushErr = fmt.Errorf("write denied")

	if err := sp.PushPolicy(ctx, &domain.Policy{ID: "1"}); err == nil {
		t.Error("expected push error for policy")
	}
	if err := sp.PushControl(ctx, &domain.Control{ID: "1"}); err == nil {
		t.Error("expected push error for control")
	}
	if err := sp.PushEvidenceTask(ctx, &domain.EvidenceTask{ID: "1"}); err == nil {
		t.Error("expected push error for evidence task")
	}
}

func TestSyncProviderDeleteError(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	sp := newStubSyncProvider("sync-test")
	sp.deleteErr = fmt.Errorf("delete denied")

	if err := sp.DeletePolicy(ctx, "1"); err == nil {
		t.Error("expected delete error for policy")
	}
	if err := sp.DeleteControl(ctx, "1"); err == nil {
		t.Error("expected delete error for control")
	}
	if err := sp.DeleteEvidenceTask(ctx, "1"); err == nil {
		t.Error("expected delete error for evidence task")
	}
}
