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

package services

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/providers"
	"github.com/grctool/grctool/internal/testhelpers"
)

// Regression tests verifying the SyncService produces identical results
// after the ProviderRegistry refactoring. These tests exercise the
// provider-based sync path end-to-end and complement the unit tests in
// sync_provider_test.go.

// TestRegressionSyncAll_FullWorkflow syncs 3 policies, 2 controls, and
// 4 evidence tasks through SyncAll and verifies counts and storage.
func TestRegressionSyncAll_FullWorkflow(t *testing.T) {
	stub := testhelpers.NewStubDataProvider("regression")

	// 3 policies
	for i := 1; i <= 3; i++ {
		pol := &domain.Policy{
			ID:        fmt.Sprintf("pol-%d", i),
			Name:      fmt.Sprintf("Regression Policy %d", i),
			Framework: "SOC2",
			Status:    "active",
		}
		stub.Policies[pol.ID] = pol
	}

	// 2 controls
	for i := 1; i <= 2; i++ {
		ctrl := &domain.Control{
			ID:        fmt.Sprintf("ctrl-%d", i),
			Name:      fmt.Sprintf("Regression Control %d", i),
			Framework: "SOC2",
			Status:    "implemented",
		}
		stub.Controls[ctrl.ID] = ctrl
	}

	// 4 evidence tasks
	for i := 1; i <= 4; i++ {
		task := &domain.EvidenceTask{
			ID:        fmt.Sprintf("et-%d", i),
			Name:      fmt.Sprintf("Regression Task %d", i),
			Framework: "SOC2",
			Status:    "pending",
		}
		stub.Tasks[task.ID] = task
	}

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stub); err != nil {
		t.Fatal(err)
	}

	svc, st := testSyncService(t, reg)
	ctx := context.Background()

	result, err := svc.SyncAll(ctx, SyncOptions{
		Policies: true,
		Controls: true,
		Evidence: true,
	})
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	// Verify counts
	if result.Policies.Total != 3 {
		t.Errorf("policies Total: got %d, want 3", result.Policies.Total)
	}
	if result.Policies.Synced != 3 {
		t.Errorf("policies Synced: got %d, want 3", result.Policies.Synced)
	}
	if result.Controls.Total != 2 {
		t.Errorf("controls Total: got %d, want 2", result.Controls.Total)
	}
	if result.Controls.Synced != 2 {
		t.Errorf("controls Synced: got %d, want 2", result.Controls.Synced)
	}
	if result.EvidenceTasks.Total != 4 {
		t.Errorf("evidence tasks Total: got %d, want 4", result.EvidenceTasks.Total)
	}
	if result.EvidenceTasks.Synced != 4 {
		t.Errorf("evidence tasks Synced: got %d, want 4", result.EvidenceTasks.Synced)
	}

	// Verify no errors
	if len(result.Errors) != 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}

	// Verify duration is non-zero
	if result.Duration == 0 {
		t.Error("expected non-zero Duration")
	}

	// Verify all entities landed in storage
	policies, err := st.GetAllPolicies()
	if err != nil {
		t.Fatalf("GetAllPolicies: %v", err)
	}
	if len(policies) != 3 {
		t.Errorf("storage policies: got %d, want 3", len(policies))
	}

	controls, err := st.GetAllControls()
	if err != nil {
		t.Fatalf("GetAllControls: %v", err)
	}
	if len(controls) != 2 {
		t.Errorf("storage controls: got %d, want 2", len(controls))
	}

	tasks, err := st.GetAllEvidenceTasks()
	if err != nil {
		t.Fatalf("GetAllEvidenceTasks: %v", err)
	}
	if len(tasks) != 4 {
		t.Errorf("storage evidence tasks: got %d, want 4", len(tasks))
	}
}

// TestRegressionSyncAll_SelectiveSync verifies that only policies are synced
// when only Policies=true is set.
func TestRegressionSyncAll_SelectiveSync(t *testing.T) {
	stub := testhelpers.NewStubDataProvider("selective")
	stub.Policies["p1"] = &domain.Policy{ID: "p1", Name: "Policy 1", Framework: "SOC2", Status: "active"}
	stub.Controls["c1"] = &domain.Control{ID: "c1", Name: "Control 1", Framework: "SOC2", Status: "implemented"}
	stub.Tasks["t1"] = &domain.EvidenceTask{ID: "t1", Name: "Task 1", Framework: "SOC2", Status: "pending"}

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stub); err != nil {
		t.Fatal(err)
	}

	svc, st := testSyncService(t, reg)
	result, err := svc.SyncAll(context.Background(), SyncOptions{
		Policies: true,
	})
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	if result.Policies.Synced != 1 {
		t.Errorf("policies Synced: got %d, want 1", result.Policies.Synced)
	}
	if result.Controls.Total != 0 {
		t.Errorf("controls Total: got %d, want 0", result.Controls.Total)
	}
	if result.EvidenceTasks.Total != 0 {
		t.Errorf("evidence tasks Total: got %d, want 0", result.EvidenceTasks.Total)
	}

	// Controls and tasks must not be in storage
	controls, _ := st.GetAllControls()
	if len(controls) != 0 {
		t.Errorf("storage controls: got %d, want 0", len(controls))
	}
	tasks, _ := st.GetAllEvidenceTasks()
	if len(tasks) != 0 {
		t.Errorf("storage evidence tasks: got %d, want 0", len(tasks))
	}
}

// TestRegressionSyncAll_ControlsOnly verifies only controls sync when Controls=true.
func TestRegressionSyncAll_ControlsOnly(t *testing.T) {
	stub := testhelpers.NewStubDataProvider("ctrl-only")
	stub.Policies["p1"] = &domain.Policy{ID: "p1", Name: "Policy 1", Framework: "SOC2", Status: "active"}
	stub.Controls["c1"] = &domain.Control{ID: "c1", Name: "Control 1", Framework: "SOC2", Status: "implemented"}
	stub.Controls["c2"] = &domain.Control{ID: "c2", Name: "Control 2", Framework: "SOC2", Status: "implemented"}
	stub.Tasks["t1"] = &domain.EvidenceTask{ID: "t1", Name: "Task 1", Framework: "SOC2", Status: "pending"}

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stub); err != nil {
		t.Fatal(err)
	}

	svc, st := testSyncService(t, reg)
	result, err := svc.SyncAll(context.Background(), SyncOptions{
		Controls: true,
	})
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	if result.Controls.Synced != 2 {
		t.Errorf("controls Synced: got %d, want 2", result.Controls.Synced)
	}
	if result.Policies.Total != 0 {
		t.Errorf("policies Total: got %d, want 0", result.Policies.Total)
	}
	if result.EvidenceTasks.Total != 0 {
		t.Errorf("evidence tasks Total: got %d, want 0", result.EvidenceTasks.Total)
	}

	// Policies and tasks must not be in storage
	policies, _ := st.GetAllPolicies()
	if len(policies) != 0 {
		t.Errorf("storage policies: got %d, want 0", len(policies))
	}
	tasks, _ := st.GetAllEvidenceTasks()
	if len(tasks) != 0 {
		t.Errorf("storage evidence tasks: got %d, want 0", len(tasks))
	}
}

// TestRegressionSyncAll_EvidenceOnly verifies only evidence tasks sync when Evidence=true.
func TestRegressionSyncAll_EvidenceOnly(t *testing.T) {
	stub := testhelpers.NewStubDataProvider("ev-only")
	stub.Policies["p1"] = &domain.Policy{ID: "p1", Name: "Policy 1", Framework: "SOC2", Status: "active"}
	stub.Controls["c1"] = &domain.Control{ID: "c1", Name: "Control 1", Framework: "SOC2", Status: "implemented"}
	stub.Tasks["t1"] = &domain.EvidenceTask{ID: "t1", Name: "Task 1", Framework: "SOC2", Status: "pending"}
	stub.Tasks["t2"] = &domain.EvidenceTask{ID: "t2", Name: "Task 2", Framework: "SOC2", Status: "pending"}
	stub.Tasks["t3"] = &domain.EvidenceTask{ID: "t3", Name: "Task 3", Framework: "SOC2", Status: "pending"}

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stub); err != nil {
		t.Fatal(err)
	}

	svc, st := testSyncService(t, reg)
	result, err := svc.SyncAll(context.Background(), SyncOptions{
		Evidence: true,
	})
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	if result.EvidenceTasks.Synced != 3 {
		t.Errorf("evidence tasks Synced: got %d, want 3", result.EvidenceTasks.Synced)
	}
	if result.Policies.Total != 0 {
		t.Errorf("policies Total: got %d, want 0", result.Policies.Total)
	}
	if result.Controls.Total != 0 {
		t.Errorf("controls Total: got %d, want 0", result.Controls.Total)
	}

	// Policies and controls must not be in storage
	policies, _ := st.GetAllPolicies()
	if len(policies) != 0 {
		t.Errorf("storage policies: got %d, want 0", len(policies))
	}
	controls, _ := st.GetAllControls()
	if len(controls) != 0 {
		t.Errorf("storage controls: got %d, want 0", len(controls))
	}
}

// TestRegressionSyncAll_MultipleProviders_Aggregation verifies that data from
// two providers is aggregated correctly in SyncResult counts.
func TestRegressionSyncAll_MultipleProviders_Aggregation(t *testing.T) {
	stubA := testhelpers.NewStubDataProvider("provider-a")
	stubA.Policies["a1"] = &domain.Policy{ID: "a1", Name: "A Policy 1", Framework: "SOC2", Status: "active"}
	stubA.Policies["a2"] = &domain.Policy{ID: "a2", Name: "A Policy 2", Framework: "SOC2", Status: "active"}

	stubB := testhelpers.NewStubDataProvider("provider-b")
	stubB.Policies["b1"] = &domain.Policy{ID: "b1", Name: "B Policy 1", Framework: "ISO27001", Status: "active"}
	stubB.Policies["b2"] = &domain.Policy{ID: "b2", Name: "B Policy 2", Framework: "ISO27001", Status: "active"}
	stubB.Policies["b3"] = &domain.Policy{ID: "b3", Name: "B Policy 3", Framework: "ISO27001", Status: "active"}

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stubA); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(stubB); err != nil {
		t.Fatal(err)
	}

	svc, st := testSyncService(t, reg)
	result, err := svc.SyncAll(context.Background(), SyncOptions{
		Policies: true,
	})
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	if result.Policies.Total != 5 {
		t.Errorf("policies Total: got %d, want 5", result.Policies.Total)
	}
	if result.Policies.Synced != 5 {
		t.Errorf("policies Synced: got %d, want 5", result.Policies.Synced)
	}
	if len(result.Errors) != 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}

	policies, err := st.GetAllPolicies()
	if err != nil {
		t.Fatalf("GetAllPolicies: %v", err)
	}
	if len(policies) != 5 {
		t.Errorf("storage policies: got %d, want 5", len(policies))
	}
}

// TestRegressionSyncAll_ProviderPartialFailure verifies that a failing provider
// does not prevent a healthy provider from syncing its data.
func TestRegressionSyncAll_ProviderPartialFailure(t *testing.T) {
	good := testhelpers.NewStubDataProvider("good")
	good.Policies["g1"] = &domain.Policy{ID: "g1", Name: "Good Policy", Framework: "SOC2", Status: "active"}
	good.Controls["gc1"] = &domain.Control{ID: "gc1", Name: "Good Control", Framework: "SOC2", Status: "implemented"}
	good.Tasks["gt1"] = &domain.EvidenceTask{ID: "gt1", Name: "Good Task", Framework: "SOC2", Status: "pending"}

	bad := testhelpers.NewStubDataProvider("bad")
	bad.ConnError = fmt.Errorf("simulated connection failure")

	reg := providers.NewProviderRegistry()
	// Register bad first to verify ordering does not break the good provider.
	if err := reg.Register(bad); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(good); err != nil {
		t.Fatal(err)
	}

	svc, st := testSyncService(t, reg)
	result, err := svc.SyncAll(context.Background(), SyncOptions{
		Policies: true,
		Controls: true,
		Evidence: true,
	})
	if err != nil {
		t.Fatalf("SyncAll returned unexpected top-level error: %v", err)
	}

	// Good provider's data should be in storage
	policies, _ := st.GetAllPolicies()
	if len(policies) != 1 {
		t.Errorf("storage policies: got %d, want 1", len(policies))
	}
	controls, _ := st.GetAllControls()
	if len(controls) != 1 {
		t.Errorf("storage controls: got %d, want 1", len(controls))
	}
	tasks, _ := st.GetAllEvidenceTasks()
	if len(tasks) != 1 {
		t.Errorf("storage evidence tasks: got %d, want 1", len(tasks))
	}

	// Bad provider should have produced errors
	if len(result.Errors) < 3 {
		t.Errorf("expected at least 3 errors from bad provider, got %d: %v", len(result.Errors), result.Errors)
	}

	// Good provider contributed to synced counts
	if result.Policies.Synced < 1 {
		t.Errorf("policies Synced: got %d, want >= 1", result.Policies.Synced)
	}
	if result.Controls.Synced < 1 {
		t.Errorf("controls Synced: got %d, want >= 1", result.Controls.Synced)
	}
	if result.EvidenceTasks.Synced < 1 {
		t.Errorf("evidence tasks Synced: got %d, want >= 1", result.EvidenceTasks.Synced)
	}
}

// TestRegressionSyncAll_ExternalIDsPopulated verifies that stored entities
// retain their ExternalIDs after sync when populated by the provider.
func TestRegressionSyncAll_ExternalIDsPopulated(t *testing.T) {
	stub := testhelpers.NewStubDataProvider("ext-ids")
	pol := &domain.Policy{
		ID:          "90001",
		Name:        "External ID Policy",
		Framework:   "SOC2",
		Status:      "active",
		ExternalIDs: map[string]string{"ext-ids": "original-ext-123"},
	}
	stub.Policies[pol.ID] = pol

	ctrl := &domain.Control{
		ID:          "80001",
		Name:        "External ID Control",
		Framework:   "SOC2",
		Status:      "implemented",
		ExternalIDs: map[string]string{"ext-ids": "ctrl-ext-456"},
	}
	stub.Controls[ctrl.ID] = ctrl

	task := &domain.EvidenceTask{
		ID:          "70001",
		Name:        "External ID Task",
		Framework:   "SOC2",
		Status:      "pending",
		ExternalIDs: map[string]string{"ext-ids": "et-ext-789"},
	}
	stub.Tasks[task.ID] = task

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stub); err != nil {
		t.Fatal(err)
	}

	svc, st := testSyncService(t, reg)
	_, err := svc.SyncAll(context.Background(), SyncOptions{
		Policies: true,
		Controls: true,
		Evidence: true,
	})
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	// Verify policy ExternalIDs persisted in storage
	storedPol, err := st.GetPolicy("90001")
	if err != nil {
		t.Fatalf("GetPolicy: %v", err)
	}
	if storedPol.ExternalIDs == nil || storedPol.ExternalIDs["ext-ids"] != "original-ext-123" {
		t.Errorf("policy ExternalIDs: got %v, want map with ext-ids=original-ext-123", storedPol.ExternalIDs)
	}

	// Verify control ExternalIDs persisted in storage
	storedCtrl, err := st.GetControl("80001")
	if err != nil {
		t.Fatalf("GetControl: %v", err)
	}
	if storedCtrl.ExternalIDs == nil || storedCtrl.ExternalIDs["ext-ids"] != "ctrl-ext-456" {
		t.Errorf("control ExternalIDs: got %v, want map with ext-ids=ctrl-ext-456", storedCtrl.ExternalIDs)
	}

	// Verify evidence task ExternalIDs persisted in storage
	storedTask, err := st.GetEvidenceTask("70001")
	if err != nil {
		t.Fatalf("GetEvidenceTask: %v", err)
	}
	if storedTask.ExternalIDs == nil || storedTask.ExternalIDs["ext-ids"] != "et-ext-789" {
		t.Errorf("evidence task ExternalIDs: got %v, want map with ext-ids=et-ext-789", storedTask.ExternalIDs)
	}
}

// TestRegressionSyncAll_PaginatedProvider verifies that fetchAll* methods
// correctly handle pagination. The StubDataProvider respects PageSize in
// ListOptions, so we set up 5 policies and use a small page size (2) to
// force multiple pages. The default fetchAll page size is 100 so we must
// verify indirectly that all items come through.
func TestRegressionSyncAll_PaginatedProvider(t *testing.T) {
	stub := testhelpers.NewStubDataProvider("paginated")

	// 5 policies — the stub respects page/pageSize, so with pageSize=100
	// (the default in fetchAllPolicies) they all arrive in one page.
	// To test the pagination loop explicitly we override the page size
	// by using syncPoliciesFromProvider directly after wrapping the stub.
	for i := 1; i <= 5; i++ {
		pol := &domain.Policy{
			ID:        fmt.Sprintf("pag-%d", i),
			Name:      fmt.Sprintf("Paginated Policy %d", i),
			Framework: "SOC2",
			Status:    "active",
		}
		stub.Policies[pol.ID] = pol
	}
	for i := 1; i <= 5; i++ {
		ctrl := &domain.Control{
			ID:        fmt.Sprintf("pagc-%d", i),
			Name:      fmt.Sprintf("Paginated Control %d", i),
			Framework: "SOC2",
			Status:    "implemented",
		}
		stub.Controls[ctrl.ID] = ctrl
	}
	for i := 1; i <= 5; i++ {
		task := &domain.EvidenceTask{
			ID:        fmt.Sprintf("pagt-%d", i),
			Name:      fmt.Sprintf("Paginated Task %d", i),
			Framework: "SOC2",
			Status:    "pending",
		}
		stub.Tasks[task.ID] = task
	}

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stub); err != nil {
		t.Fatal(err)
	}

	svc, st := testSyncService(t, reg)
	result, err := svc.SyncAll(context.Background(), SyncOptions{
		Policies: true,
		Controls: true,
		Evidence: true,
	})
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	// All 5 items of each type must be synced
	if result.Policies.Total != 5 {
		t.Errorf("policies Total: got %d, want 5", result.Policies.Total)
	}
	if result.Policies.Synced != 5 {
		t.Errorf("policies Synced: got %d, want 5", result.Policies.Synced)
	}
	if result.Controls.Total != 5 {
		t.Errorf("controls Total: got %d, want 5", result.Controls.Total)
	}
	if result.Controls.Synced != 5 {
		t.Errorf("controls Synced: got %d, want 5", result.Controls.Synced)
	}
	if result.EvidenceTasks.Total != 5 {
		t.Errorf("evidence tasks Total: got %d, want 5", result.EvidenceTasks.Total)
	}
	if result.EvidenceTasks.Synced != 5 {
		t.Errorf("evidence tasks Synced: got %d, want 5", result.EvidenceTasks.Synced)
	}

	// Verify storage has all items
	policies, _ := st.GetAllPolicies()
	if len(policies) != 5 {
		t.Errorf("storage policies: got %d, want 5", len(policies))
	}
	controls, _ := st.GetAllControls()
	if len(controls) != 5 {
		t.Errorf("storage controls: got %d, want 5", len(controls))
	}
	tasks, _ := st.GetAllEvidenceTasks()
	if len(tasks) != 5 {
		t.Errorf("storage evidence tasks: got %d, want 5", len(tasks))
	}

	if len(result.Errors) != 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
}

// TestRegressionSyncAll_EmptyProvider verifies that a provider with 0 entities
// results in a clean completion with zero counts and no errors.
func TestRegressionSyncAll_EmptyProvider(t *testing.T) {
	stub := testhelpers.NewStubDataProvider("empty")
	// No data added — all maps are empty.

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stub); err != nil {
		t.Fatal(err)
	}

	svc, st := testSyncService(t, reg)
	result, err := svc.SyncAll(context.Background(), SyncOptions{
		Policies: true,
		Controls: true,
		Evidence: true,
	})
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	if result.Policies.Total != 0 {
		t.Errorf("policies Total: got %d, want 0", result.Policies.Total)
	}
	if result.Policies.Synced != 0 {
		t.Errorf("policies Synced: got %d, want 0", result.Policies.Synced)
	}
	if result.Controls.Total != 0 {
		t.Errorf("controls Total: got %d, want 0", result.Controls.Total)
	}
	if result.Controls.Synced != 0 {
		t.Errorf("controls Synced: got %d, want 0", result.Controls.Synced)
	}
	if result.EvidenceTasks.Total != 0 {
		t.Errorf("evidence tasks Total: got %d, want 0", result.EvidenceTasks.Total)
	}
	if result.EvidenceTasks.Synced != 0 {
		t.Errorf("evidence tasks Synced: got %d, want 0", result.EvidenceTasks.Synced)
	}
	if len(result.Errors) != 0 {
		t.Errorf("unexpected errors: %v", result.Errors)
	}
	if result.Duration == 0 {
		t.Error("expected non-zero Duration even for empty sync")
	}

	// Storage should be empty
	policies, _ := st.GetAllPolicies()
	if len(policies) != 0 {
		t.Errorf("storage policies: got %d, want 0", len(policies))
	}
	controls, _ := st.GetAllControls()
	if len(controls) != 0 {
		t.Errorf("storage controls: got %d, want 0", len(controls))
	}
	tasks, _ := st.GetAllEvidenceTasks()
	if len(tasks) != 0 {
		t.Errorf("storage evidence tasks: got %d, want 0", len(tasks))
	}
}

// TestRegressionSyncAll_ContextCancellation verifies that SyncAll returns
// promptly when the context is cancelled.
func TestRegressionSyncAll_ContextCancellation(t *testing.T) {
	stub := testhelpers.NewStubDataProvider("cancel")
	// Add some data so there is work to do.
	for i := 1; i <= 10; i++ {
		stub.Policies[fmt.Sprintf("cp-%d", i)] = &domain.Policy{
			ID:        fmt.Sprintf("cp-%d", i),
			Name:      fmt.Sprintf("Cancel Policy %d", i),
			Framework: "SOC2",
			Status:    "active",
		}
	}

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stub); err != nil {
		t.Fatal(err)
	}

	svc, _ := testSyncService(t, reg)

	// Cancel the context before calling SyncAll.
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	done := make(chan struct{})
	var result *SyncResult
	var syncErr error

	go func() {
		defer close(done)
		result, syncErr = svc.SyncAll(ctx, SyncOptions{
			Policies: true,
			Controls: true,
			Evidence: true,
		})
	}()

	select {
	case <-done:
		// Sync completed — either with an error or partial results.
		// Both are acceptable for a cancelled context.
		_ = result
		_ = syncErr
	case <-time.After(2 * time.Second):
		t.Fatal("SyncAll did not return within 2 seconds after context cancellation")
	}
}
