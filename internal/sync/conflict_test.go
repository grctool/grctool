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
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
)

// --- Detection Tests ---

func TestDetectConflicts_InSync(t *testing.T) {
	detector := NewConflictDetector()
	meta := &domain.SyncMetadata{
		ContentHash: map[string]string{
			"tugboat": "abc123",
		},
	}
	changes := []interfaces.ChangeEntry{
		{EntityType: "policy", EntityID: "POL-0001", Hash: "abc123", ModifiedAt: time.Now()},
	}

	conflicts := detector.DetectConflicts(meta, changes, "tugboat")

	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts for in-sync entities, got %d", len(conflicts))
	}
}

func TestDetectConflicts_RemoteOnly(t *testing.T) {
	detector := NewConflictDetector()
	meta := &domain.SyncMetadata{
		ContentHash: map[string]string{
			// No entry for "tugboat" provider
		},
	}
	changes := []interfaces.ChangeEntry{
		{EntityType: "policy", EntityID: "POL-0099", Hash: "new-hash", ModifiedAt: time.Now()},
	}

	conflicts := detector.DetectConflicts(meta, changes, "tugboat")

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Classification != ClassificationRemoteOnly {
		t.Errorf("expected remote_only, got %s", conflicts[0].Classification)
	}
	if conflicts[0].EntityID != "POL-0099" {
		t.Errorf("expected entity ID POL-0099, got %s", conflicts[0].EntityID)
	}
}

func TestDetectConflicts_RemoteChanged(t *testing.T) {
	detector := NewConflictDetector()
	meta := &domain.SyncMetadata{
		ContentHash: map[string]string{
			"tugboat": "old-hash",
			// No "local" key — local is unchanged from stored
		},
	}
	changes := []interfaces.ChangeEntry{
		{EntityType: "control", EntityID: "CC-01", Hash: "new-remote-hash", ModifiedAt: time.Now()},
	}

	conflicts := detector.DetectConflicts(meta, changes, "tugboat")

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Classification != ClassificationRemoteChanged {
		t.Errorf("expected remote_changed, got %s", conflicts[0].Classification)
	}
}

func TestDetectConflicts_BothChanged(t *testing.T) {
	detector := NewConflictDetector()
	meta := &domain.SyncMetadata{
		ContentHash: map[string]string{
			"tugboat": "original-hash",
			"local":   "local-modified-hash", // Local changed from original
		},
	}
	changes := []interfaces.ChangeEntry{
		{EntityType: "policy", EntityID: "POL-0001", Hash: "remote-modified-hash", ModifiedAt: time.Now()},
	}

	conflicts := detector.DetectConflicts(meta, changes, "tugboat")

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Classification != ClassificationBothChanged {
		t.Errorf("expected both_changed, got %s", conflicts[0].Classification)
	}
	if conflicts[0].LocalHash != "local-modified-hash" {
		t.Errorf("expected local hash 'local-modified-hash', got %s", conflicts[0].LocalHash)
	}
	if conflicts[0].RemoteHash != "remote-modified-hash" {
		t.Errorf("expected remote hash 'remote-modified-hash', got %s", conflicts[0].RemoteHash)
	}
}

func TestDetectConflicts_LocalChanged(t *testing.T) {
	detector := NewConflictDetector()
	meta := &domain.SyncMetadata{
		ContentHash: map[string]string{
			"tugboat": "original-hash",
			"local":   "local-modified-hash", // Local changed
		},
	}
	changes := []interfaces.ChangeEntry{
		// Remote hash matches stored — remote did NOT change
		{EntityType: "policy", EntityID: "POL-0001", Hash: "original-hash", ModifiedAt: time.Now()},
	}

	conflicts := detector.DetectConflicts(meta, changes, "tugboat")

	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict, got %d", len(conflicts))
	}
	if conflicts[0].Classification != ClassificationLocalChanged {
		t.Errorf("expected local_changed, got %s", conflicts[0].Classification)
	}
}

func TestDetectConflicts_NilSyncMetadata(t *testing.T) {
	detector := NewConflictDetector()
	changes := []interfaces.ChangeEntry{
		{EntityType: "policy", EntityID: "POL-0001", Hash: "hash1", ModifiedAt: time.Now()},
		{EntityType: "control", EntityID: "CC-01", Hash: "hash2", ModifiedAt: time.Now()},
	}

	conflicts := detector.DetectConflicts(nil, changes, "tugboat")

	if len(conflicts) != 2 {
		t.Fatalf("expected 2 conflicts, got %d", len(conflicts))
	}
	for _, c := range conflicts {
		if c.Classification != ClassificationRemoteOnly {
			t.Errorf("expected remote_only for nil metadata, got %s for %s", c.Classification, c.EntityID)
		}
		if c.Provider != "tugboat" {
			t.Errorf("expected provider 'tugboat', got %s", c.Provider)
		}
	}
}

func TestDetectConflicts_MultipleChanges(t *testing.T) {
	detector := NewConflictDetector()
	meta := &domain.SyncMetadata{
		ContentHash: map[string]string{
			"tugboat": "stored-hash",
			"local":   "stored-hash", // Local unchanged
		},
		LastSyncTime: map[string]time.Time{
			"tugboat": time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	}
	changes := []interfaces.ChangeEntry{
		// In sync — remote hash matches stored, local matches stored
		{EntityType: "policy", EntityID: "POL-0001", Hash: "stored-hash", ModifiedAt: time.Now()},
		// Remote changed — different remote hash
		{EntityType: "control", EntityID: "CC-01", Hash: "new-remote-hash", ModifiedAt: time.Now()},
	}

	conflicts := detector.DetectConflicts(meta, changes, "tugboat")

	// Only the remote-changed entity should appear (in-sync is filtered out)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 conflict (in-sync filtered), got %d", len(conflicts))
	}
	if conflicts[0].Classification != ClassificationRemoteChanged {
		t.Errorf("expected remote_changed, got %s", conflicts[0].Classification)
	}
	if conflicts[0].EntityID != "CC-01" {
		t.Errorf("expected entity CC-01, got %s", conflicts[0].EntityID)
	}
	// Should have local modified time from sync metadata
	expected := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if !conflicts[0].LocalModified.Equal(expected) {
		t.Errorf("expected local modified %v, got %v", expected, conflicts[0].LocalModified)
	}
}

func TestDetectConflicts_EmptyChanges(t *testing.T) {
	detector := NewConflictDetector()
	meta := &domain.SyncMetadata{
		ContentHash: map[string]string{
			"tugboat": "hash",
		},
	}

	conflicts := detector.DetectConflicts(meta, []interfaces.ChangeEntry{}, "tugboat")

	if len(conflicts) != 0 {
		t.Errorf("expected 0 conflicts for empty changes, got %d", len(conflicts))
	}
}

// --- Resolution Tests ---

func TestResolve_LocalWins(t *testing.T) {
	resolver := NewConflictResolver(ConflictPolicyLocalWins)
	conflict := Conflict{
		EntityType:     "policy",
		EntityID:       "POL-0001",
		Provider:       "tugboat",
		Classification: ClassificationBothChanged,
		LocalHash:      "local-hash",
		RemoteHash:     "remote-hash",
	}

	res := resolver.Resolve(conflict)

	if res.Winner != "local" {
		t.Errorf("expected winner 'local', got '%s'", res.Winner)
	}
	if res.Policy != ConflictPolicyLocalWins {
		t.Errorf("expected policy local_wins, got %s", res.Policy)
	}
	if res.ResolvedAt.IsZero() {
		t.Error("expected non-zero ResolvedAt")
	}
}

func TestResolve_RemoteWins(t *testing.T) {
	resolver := NewConflictResolver(ConflictPolicyRemoteWins)
	conflict := Conflict{
		EntityType:     "control",
		EntityID:       "CC-01",
		Provider:       "tugboat",
		Classification: ClassificationBothChanged,
	}

	res := resolver.Resolve(conflict)

	if res.Winner != "remote" {
		t.Errorf("expected winner 'remote', got '%s'", res.Winner)
	}
	if res.Policy != ConflictPolicyRemoteWins {
		t.Errorf("expected policy remote_wins, got %s", res.Policy)
	}
}

func TestResolve_NewestWins_LocalNewer(t *testing.T) {
	resolver := NewConflictResolver(ConflictPolicyNewestWins)
	conflict := Conflict{
		EntityType:     "policy",
		EntityID:       "POL-0001",
		Classification: ClassificationBothChanged,
		LocalModified:  time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC),
		RemoteModified: time.Date(2026, 3, 17, 12, 0, 0, 0, time.UTC),
	}

	res := resolver.Resolve(conflict)

	if res.Winner != "local" {
		t.Errorf("expected winner 'local' (newer), got '%s'", res.Winner)
	}
}

func TestResolve_NewestWins_RemoteNewer(t *testing.T) {
	resolver := NewConflictResolver(ConflictPolicyNewestWins)
	conflict := Conflict{
		EntityType:     "policy",
		EntityID:       "POL-0001",
		Classification: ClassificationBothChanged,
		LocalModified:  time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC),
		RemoteModified: time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC),
	}

	res := resolver.Resolve(conflict)

	if res.Winner != "remote" {
		t.Errorf("expected winner 'remote' (newer), got '%s'", res.Winner)
	}
}

func TestResolve_NewestWins_SameTime(t *testing.T) {
	resolver := NewConflictResolver(ConflictPolicyNewestWins)
	sameTime := time.Date(2026, 3, 18, 12, 0, 0, 0, time.UTC)
	conflict := Conflict{
		EntityType:     "policy",
		EntityID:       "POL-0001",
		Classification: ClassificationBothChanged,
		LocalModified:  sameTime,
		RemoteModified: sameTime,
	}

	res := resolver.Resolve(conflict)

	if res.Winner != "local" {
		t.Errorf("expected winner 'local' (tiebreak), got '%s'", res.Winner)
	}
}

func TestResolve_Manual(t *testing.T) {
	resolver := NewConflictResolver(ConflictPolicyManual)
	conflict := Conflict{
		EntityType:     "evidence_task",
		EntityID:       "ET-0047",
		Provider:       "tugboat",
		Classification: ClassificationBothChanged,
	}

	res := resolver.Resolve(conflict)

	if res.Winner != "" {
		t.Errorf("expected empty winner for manual policy, got '%s'", res.Winner)
	}
	if res.Policy != ConflictPolicyManual {
		t.Errorf("expected policy manual, got %s", res.Policy)
	}
}

func TestResolveAll(t *testing.T) {
	resolver := NewConflictResolver(ConflictPolicyRemoteWins)
	conflicts := []Conflict{
		{EntityType: "policy", EntityID: "POL-0001", Classification: ClassificationRemoteChanged},
		{EntityType: "control", EntityID: "CC-01", Classification: ClassificationBothChanged},
		{EntityType: "evidence_task", EntityID: "ET-0001", Classification: ClassificationLocalChanged},
	}

	resolutions := resolver.ResolveAll(conflicts)

	if len(resolutions) != 3 {
		t.Fatalf("expected 3 resolutions, got %d", len(resolutions))
	}
	for i, res := range resolutions {
		if res.Winner != "remote" {
			t.Errorf("resolution[%d]: expected winner 'remote', got '%s'", i, res.Winner)
		}
		if res.Conflict.EntityID != conflicts[i].EntityID {
			t.Errorf("resolution[%d]: expected entity %s, got %s", i, conflicts[i].EntityID, res.Conflict.EntityID)
		}
	}
}

func TestResolveAll_Empty(t *testing.T) {
	resolver := NewConflictResolver(ConflictPolicyLocalWins)

	resolutions := resolver.ResolveAll([]Conflict{})

	if len(resolutions) != 0 {
		t.Errorf("expected 0 resolutions for empty input, got %d", len(resolutions))
	}
}
