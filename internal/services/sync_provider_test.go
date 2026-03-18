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

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/providers"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/testhelpers"
)

// testConfig returns a minimal config suitable for SyncService tests.
func testConfig(dataDir string) *config.Config {
	return &config.Config{
		Storage: config.StorageConfig{
			DataDir: dataDir,
		},
		Interpolation: config.InterpolationConfig{
			Enabled: false,
		},
	}
}

// testSyncService creates a SyncService backed by a ProviderRegistry and
// real storage in the given temp directory.
func testSyncService(t *testing.T, reg *providers.ProviderRegistry) (*SyncService, *storage.Storage) {
	t.Helper()
	dataDir := t.TempDir()
	cfg := testConfig(dataDir)
	st, err := storage.NewStorage(cfg.Storage)
	if err != nil {
		t.Fatalf("failed to create storage: %v", err)
	}
	log, err := logger.NewTestLogger()
	if err != nil {
		t.Fatalf("failed to create test logger: %v", err)
	}
	svc := NewSyncServiceWithRegistry(reg, st, cfg, log)
	return svc, st
}

func TestSyncServiceWithRegistry_SyncPolicies(t *testing.T) {
	stub := testhelpers.NewStubDataProvider("test")
	pol := testhelpers.SamplePolicy()
	stub.Policies[pol.ID] = pol

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stub); err != nil {
		t.Fatal(err)
	}

	svc, st := testSyncService(t, reg)
	ctx := context.Background()

	result, err := svc.SyncAll(ctx, SyncOptions{
		Policies: true,
	})
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	if result.Policies.Total != 1 {
		t.Errorf("expected Total=1, got %d", result.Policies.Total)
	}
	if result.Policies.Synced != 1 {
		t.Errorf("expected Synced=1, got %d", result.Policies.Synced)
	}
	if result.Policies.Errors != 0 {
		t.Errorf("expected Errors=0, got %d", result.Policies.Errors)
	}

	// Verify data landed in storage
	policies, err := st.GetAllPolicies()
	if err != nil {
		t.Fatalf("GetAllPolicies failed: %v", err)
	}
	if len(policies) != 1 {
		t.Fatalf("expected 1 policy in storage, got %d", len(policies))
	}
	if policies[0].Name != pol.Name {
		t.Errorf("expected policy name %q, got %q", pol.Name, policies[0].Name)
	}
}

func TestSyncServiceWithRegistry_SyncControls(t *testing.T) {
	stub := testhelpers.NewStubDataProvider("test")
	ctrl := testhelpers.SampleControl()
	stub.Controls[ctrl.ID] = ctrl

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stub); err != nil {
		t.Fatal(err)
	}

	svc, st := testSyncService(t, reg)
	ctx := context.Background()

	result, err := svc.SyncAll(ctx, SyncOptions{
		Controls: true,
	})
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	if result.Controls.Total != 1 {
		t.Errorf("expected Total=1, got %d", result.Controls.Total)
	}
	if result.Controls.Synced != 1 {
		t.Errorf("expected Synced=1, got %d", result.Controls.Synced)
	}
	if result.Controls.Errors != 0 {
		t.Errorf("expected Errors=0, got %d", result.Controls.Errors)
	}

	// Verify data landed in storage
	controls, err := st.GetAllControls()
	if err != nil {
		t.Fatalf("GetAllControls failed: %v", err)
	}
	if len(controls) != 1 {
		t.Fatalf("expected 1 control in storage, got %d", len(controls))
	}
	if controls[0].Name != ctrl.Name {
		t.Errorf("expected control name %q, got %q", ctrl.Name, controls[0].Name)
	}
}

func TestSyncServiceWithRegistry_SyncEvidenceTasks(t *testing.T) {
	stub := testhelpers.NewStubDataProvider("test")
	task := testhelpers.SampleEvidenceTask()
	stub.Tasks[task.ID] = task

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stub); err != nil {
		t.Fatal(err)
	}

	svc, st := testSyncService(t, reg)
	ctx := context.Background()

	result, err := svc.SyncAll(ctx, SyncOptions{
		Evidence: true,
	})
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	if result.EvidenceTasks.Total != 1 {
		t.Errorf("expected Total=1, got %d", result.EvidenceTasks.Total)
	}
	if result.EvidenceTasks.Synced != 1 {
		t.Errorf("expected Synced=1, got %d", result.EvidenceTasks.Synced)
	}

	// Verify data landed in storage
	tasks, err := st.GetAllEvidenceTasks()
	if err != nil {
		t.Fatalf("GetAllEvidenceTasks failed: %v", err)
	}
	if len(tasks) != 1 {
		t.Fatalf("expected 1 evidence task in storage, got %d", len(tasks))
	}
	if tasks[0].Name != task.Name {
		t.Errorf("expected task name %q, got %q", task.Name, tasks[0].Name)
	}
}

func TestSyncServiceWithRegistry_MultipleProviders(t *testing.T) {
	// Provider A has one policy
	stubA := testhelpers.NewStubDataProvider("provider-a")
	polA := &domain.Policy{
		ID:        "100",
		Name:      "Policy from A",
		Framework: "SOC2",
		Status:    "active",
	}
	stubA.Policies[polA.ID] = polA

	// Provider B has a different policy
	stubB := testhelpers.NewStubDataProvider("provider-b")
	polB := &domain.Policy{
		ID:        "200",
		Name:      "Policy from B",
		Framework: "ISO27001",
		Status:    "active",
	}
	stubB.Policies[polB.ID] = polB

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stubA); err != nil {
		t.Fatal(err)
	}
	if err := reg.Register(stubB); err != nil {
		t.Fatal(err)
	}

	svc, st := testSyncService(t, reg)
	ctx := context.Background()

	result, err := svc.SyncAll(ctx, SyncOptions{
		Policies: true,
	})
	if err != nil {
		t.Fatalf("SyncAll failed: %v", err)
	}

	// Both providers contributed
	if result.Policies.Total != 2 {
		t.Errorf("expected Total=2, got %d", result.Policies.Total)
	}
	if result.Policies.Synced != 2 {
		t.Errorf("expected Synced=2, got %d", result.Policies.Synced)
	}

	// Verify both policies are in storage
	policies, err := st.GetAllPolicies()
	if err != nil {
		t.Fatalf("GetAllPolicies failed: %v", err)
	}
	if len(policies) != 2 {
		t.Fatalf("expected 2 policies in storage, got %d", len(policies))
	}
}

func TestSyncServiceWithRegistry_ProviderError(t *testing.T) {
	stub := testhelpers.NewStubDataProvider("failing")
	stub.ConnError = fmt.Errorf("simulated connection failure")

	reg := providers.NewProviderRegistry()
	if err := reg.Register(stub); err != nil {
		t.Fatal(err)
	}

	svc, _ := testSyncService(t, reg)
	ctx := context.Background()

	result, err := svc.SyncAll(ctx, SyncOptions{
		Policies: true,
		Controls: true,
		Evidence: true,
	})
	// SyncAll itself should not return an error; errors are collected in result
	if err != nil {
		t.Fatalf("SyncAll returned unexpected error: %v", err)
	}

	// Should have recorded errors for all three sync types
	if len(result.Errors) < 3 {
		t.Errorf("expected at least 3 errors, got %d: %v", len(result.Errors), result.Errors)
	}

	// Stats should show zero successful syncs
	if result.Policies.Synced != 0 {
		t.Errorf("expected Synced=0 for policies, got %d", result.Policies.Synced)
	}
	if result.Controls.Synced != 0 {
		t.Errorf("expected Synced=0 for controls, got %d", result.Controls.Synced)
	}
	if result.EvidenceTasks.Synced != 0 {
		t.Errorf("expected Synced=0 for evidence tasks, got %d", result.EvidenceTasks.Synced)
	}
}

func TestSyncServiceWithRegistry_EmptyRegistry(t *testing.T) {
	reg := providers.NewProviderRegistry()

	svc, _ := testSyncService(t, reg)
	ctx := context.Background()

	result, err := svc.SyncAll(ctx, SyncOptions{
		Policies: true,
		Controls: true,
		Evidence: true,
	})
	if err != nil {
		t.Fatalf("SyncAll returned unexpected error: %v", err)
	}

	// No providers means no data synced, but no errors either
	if len(result.Errors) != 0 {
		t.Errorf("expected 0 errors, got %d: %v", len(result.Errors), result.Errors)
	}
	if result.Policies.Total != 0 {
		t.Errorf("expected Total=0 for policies, got %d", result.Policies.Total)
	}
	if result.Controls.Total != 0 {
		t.Errorf("expected Total=0 for controls, got %d", result.Controls.Total)
	}
	if result.EvidenceTasks.Total != 0 {
		t.Errorf("expected Total=0 for evidence tasks, got %d", result.EvidenceTasks.Total)
	}
}
