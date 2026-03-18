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

package providers

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/testhelpers"
)

// stubSyncProvider embeds StubDataProvider and adds SyncProvider methods.
type stubSyncProvider struct {
	*testhelpers.StubDataProvider
}

var _ interfaces.SyncProvider = (*stubSyncProvider)(nil)

func newStubSyncProvider(name string) *stubSyncProvider {
	return &stubSyncProvider{
		StubDataProvider: testhelpers.NewStubDataProvider(name),
	}
}

func (s *stubSyncProvider) PushPolicy(_ context.Context, _ *domain.Policy) error {
	return nil
}

func (s *stubSyncProvider) PushControl(_ context.Context, _ *domain.Control) error {
	return nil
}

func (s *stubSyncProvider) PushEvidenceTask(_ context.Context, _ *domain.EvidenceTask) error {
	return nil
}

func (s *stubSyncProvider) DeletePolicy(_ context.Context, _ string) error {
	return nil
}

func (s *stubSyncProvider) DeleteControl(_ context.Context, _ string) error {
	return nil
}

func (s *stubSyncProvider) DeleteEvidenceTask(_ context.Context, _ string) error {
	return nil
}

func (s *stubSyncProvider) DetectChanges(_ context.Context, _ time.Time) (*interfaces.ChangeSet, error) {
	return &interfaces.ChangeSet{Provider: s.Name()}, nil
}

func (s *stubSyncProvider) ResolveConflict(_ context.Context, _ interfaces.Conflict, _ interfaces.ConflictResolution) error {
	return fmt.Errorf("not implemented")
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestNewProviderRegistry(t *testing.T) {
	reg := NewProviderRegistry()
	if reg.Count() != 0 {
		t.Fatalf("expected empty registry, got count=%d", reg.Count())
	}
	if names := reg.List(); len(names) != 0 {
		t.Fatalf("expected empty list, got %v", names)
	}
}

func TestRegister(t *testing.T) {
	reg := NewProviderRegistry()
	stub := testhelpers.NewStubDataProvider("alpha")

	if err := reg.Register(stub); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := reg.Get("alpha")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name() != "alpha" {
		t.Fatalf("expected name=alpha, got %s", got.Name())
	}
}

func TestRegister_Duplicate(t *testing.T) {
	reg := NewProviderRegistry()
	stub := testhelpers.NewStubDataProvider("dup")

	if err := reg.Register(stub); err != nil {
		t.Fatalf("unexpected error on first register: %v", err)
	}
	err := reg.Register(testhelpers.NewStubDataProvider("dup"))
	if err == nil {
		t.Fatal("expected error on duplicate register, got nil")
	}
}

func TestGet_NotFound(t *testing.T) {
	reg := NewProviderRegistry()
	_, err := reg.Get("nonexistent")
	if err == nil {
		t.Fatal("expected error for missing provider, got nil")
	}
}

func TestGetSyncProvider(t *testing.T) {
	reg := NewProviderRegistry()
	sp := newStubSyncProvider("syncer")

	if err := reg.Register(sp); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := reg.GetSyncProvider("syncer")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Name() != "syncer" {
		t.Fatalf("expected name=syncer, got %s", got.Name())
	}
}

func TestGetSyncProvider_NotSyncProvider(t *testing.T) {
	reg := NewProviderRegistry()
	stub := testhelpers.NewStubDataProvider("readonly")

	if err := reg.Register(stub); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_, err := reg.GetSyncProvider("readonly")
	if err == nil {
		t.Fatal("expected error for non-SyncProvider, got nil")
	}
}

func TestList(t *testing.T) {
	reg := NewProviderRegistry()
	for _, name := range []string{"charlie", "alpha", "bravo"} {
		if err := reg.Register(testhelpers.NewStubDataProvider(name)); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	names := reg.List()
	expected := []string{"alpha", "bravo", "charlie"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d names, got %d", len(expected), len(names))
	}
	for i, n := range names {
		if n != expected[i] {
			t.Fatalf("expected names[%d]=%s, got %s", i, expected[i], n)
		}
	}
}

func TestListSyncProviders(t *testing.T) {
	reg := NewProviderRegistry()
	_ = reg.Register(testhelpers.NewStubDataProvider("readonly1"))
	_ = reg.Register(newStubSyncProvider("syncer1"))
	_ = reg.Register(testhelpers.NewStubDataProvider("readonly2"))
	_ = reg.Register(newStubSyncProvider("syncer2"))

	names := reg.ListSyncProviders()
	expected := []string{"syncer1", "syncer2"}
	if len(names) != len(expected) {
		t.Fatalf("expected %d sync providers, got %d: %v", len(expected), len(names), names)
	}
	for i, n := range names {
		if n != expected[i] {
			t.Fatalf("expected names[%d]=%s, got %s", i, expected[i], n)
		}
	}
}

func TestRemove(t *testing.T) {
	reg := NewProviderRegistry()
	_ = reg.Register(testhelpers.NewStubDataProvider("removeme"))

	reg.Remove("removeme")

	if reg.Has("removeme") {
		t.Fatal("expected provider to be removed")
	}
	_, err := reg.Get("removeme")
	if err == nil {
		t.Fatal("expected error after removal, got nil")
	}
}

func TestRemove_NotFound(t *testing.T) {
	reg := NewProviderRegistry()
	// Should not panic or error.
	reg.Remove("ghost")
}

func TestCount(t *testing.T) {
	reg := NewProviderRegistry()
	if reg.Count() != 0 {
		t.Fatalf("expected 0, got %d", reg.Count())
	}

	_ = reg.Register(testhelpers.NewStubDataProvider("a"))
	if reg.Count() != 1 {
		t.Fatalf("expected 1, got %d", reg.Count())
	}

	_ = reg.Register(testhelpers.NewStubDataProvider("b"))
	if reg.Count() != 2 {
		t.Fatalf("expected 2, got %d", reg.Count())
	}

	reg.Remove("a")
	if reg.Count() != 1 {
		t.Fatalf("expected 1 after remove, got %d", reg.Count())
	}
}

func TestHas(t *testing.T) {
	reg := NewProviderRegistry()
	_ = reg.Register(testhelpers.NewStubDataProvider("exists"))

	if !reg.Has("exists") {
		t.Fatal("expected Has(exists) to be true")
	}
	if reg.Has("nope") {
		t.Fatal("expected Has(nope) to be false")
	}
}

func TestHealthCheck(t *testing.T) {
	reg := NewProviderRegistry()
	_ = reg.Register(testhelpers.NewStubDataProvider("healthy1"))
	_ = reg.Register(testhelpers.NewStubDataProvider("healthy2"))

	results := reg.HealthCheck(context.Background())
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	for name, err := range results {
		if err != nil {
			t.Fatalf("expected nil error for %s, got %v", name, err)
		}
	}
}

func TestHealthCheck_FailingProvider(t *testing.T) {
	reg := NewProviderRegistry()

	healthy := testhelpers.NewStubDataProvider("healthy")
	_ = reg.Register(healthy)

	failing := testhelpers.NewStubDataProvider("failing")
	failing.ConnError = fmt.Errorf("connection refused")
	_ = reg.Register(failing)

	results := reg.HealthCheck(context.Background())
	if results["healthy"] != nil {
		t.Fatalf("expected nil error for healthy, got %v", results["healthy"])
	}
	if results["failing"] == nil {
		t.Fatal("expected error for failing provider, got nil")
	}
}

func TestConcurrentAccess(t *testing.T) {
	reg := NewProviderRegistry()
	const goroutines = 50

	var wg sync.WaitGroup
	wg.Add(goroutines * 3)

	// Concurrent registrations (unique names).
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			name := fmt.Sprintf("provider-%03d", idx)
			_ = reg.Register(testhelpers.NewStubDataProvider(name))
		}(i)
	}

	// Concurrent reads.
	for i := 0; i < goroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			name := fmt.Sprintf("provider-%03d", idx)
			_, _ = reg.Get(name)
		}(i)
	}

	// Concurrent listings.
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			_ = reg.List()
			_ = reg.ListSyncProviders()
			_ = reg.Count()
		}()
	}

	wg.Wait()

	// After all goroutines complete, all providers should be registered.
	if reg.Count() != goroutines {
		t.Fatalf("expected %d providers, got %d", goroutines, reg.Count())
	}
}
