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
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/logger"
)

func newTestAuditTrail(t *testing.T) (*AuditTrail, string) {
	t.Helper()
	dir := t.TempDir()
	log, err := logger.NewTestLogger()
	if err != nil {
		t.Fatalf("create test logger: %v", err)
	}
	return NewAuditTrail(dir, log), dir
}

func TestAuditLog_RecordPull(t *testing.T) {
	trail, _ := newTestAuditTrail(t)
	al := trail.NewAuditLog("test-user", []string{"tugboat"})

	al.RecordPull("tugboat", "policy", "POL-0001", "hash-before", "hash-after")

	if len(al.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(al.Entries))
	}
	e := al.Entries[0]
	if e.Operation != "pull" {
		t.Errorf("expected operation 'pull', got %q", e.Operation)
	}
	if e.Provider != "tugboat" {
		t.Errorf("expected provider 'tugboat', got %q", e.Provider)
	}
	if e.EntityType != "policy" {
		t.Errorf("expected entity_type 'policy', got %q", e.EntityType)
	}
	if e.EntityID != "POL-0001" {
		t.Errorf("expected entity_id 'POL-0001', got %q", e.EntityID)
	}
	if e.BeforeHash != "hash-before" {
		t.Errorf("expected before_hash 'hash-before', got %q", e.BeforeHash)
	}
	if e.AfterHash != "hash-after" {
		t.Errorf("expected after_hash 'hash-after', got %q", e.AfterHash)
	}
	if e.Actor != "test-user" {
		t.Errorf("expected actor 'test-user', got %q", e.Actor)
	}
}

func TestAuditLog_RecordPush(t *testing.T) {
	trail, _ := newTestAuditTrail(t)
	al := trail.NewAuditLog("test-user", []string{"tugboat"})

	al.RecordPush("tugboat", "control", "CC-01", "local-hash", "remote-hash")

	if len(al.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(al.Entries))
	}
	e := al.Entries[0]
	if e.Operation != "push" {
		t.Errorf("expected operation 'push', got %q", e.Operation)
	}
	if e.EntityType != "control" {
		t.Errorf("expected entity_type 'control', got %q", e.EntityType)
	}
	if e.EntityID != "CC-01" {
		t.Errorf("expected entity_id 'CC-01', got %q", e.EntityID)
	}
}

func TestAuditLog_RecordConflictResolved(t *testing.T) {
	trail, _ := newTestAuditTrail(t)
	al := trail.NewAuditLog("test-user", []string{"tugboat"})

	al.RecordConflictResolved("tugboat", "policy", "POL-0002", "remote_wins", "old-hash", "new-hash")

	if len(al.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(al.Entries))
	}
	e := al.Entries[0]
	if e.Operation != "conflict_resolved" {
		t.Errorf("expected operation 'conflict_resolved', got %q", e.Operation)
	}
	if e.Resolution != "remote_wins" {
		t.Errorf("expected resolution 'remote_wins', got %q", e.Resolution)
	}
	if e.BeforeHash != "old-hash" {
		t.Errorf("expected before_hash 'old-hash', got %q", e.BeforeHash)
	}
	if e.AfterHash != "new-hash" {
		t.Errorf("expected after_hash 'new-hash', got %q", e.AfterHash)
	}
}

func TestAuditLog_RecordConflictManual(t *testing.T) {
	trail, _ := newTestAuditTrail(t)
	al := trail.NewAuditLog("test-user", []string{"tugboat"})

	al.RecordConflictManual("tugboat", "evidence_task", "ET-0001", "local-hash", "remote-hash")

	if len(al.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(al.Entries))
	}
	e := al.Entries[0]
	if e.Operation != "conflict_manual" {
		t.Errorf("expected operation 'conflict_manual', got %q", e.Operation)
	}
	if e.EntityType != "evidence_task" {
		t.Errorf("expected entity_type 'evidence_task', got %q", e.EntityType)
	}
	if e.BeforeHash != "local-hash" {
		t.Errorf("expected before_hash (local) 'local-hash', got %q", e.BeforeHash)
	}
	if e.AfterHash != "remote-hash" {
		t.Errorf("expected after_hash (remote) 'remote-hash', got %q", e.AfterHash)
	}
}

func TestAuditLog_RecordError(t *testing.T) {
	trail, _ := newTestAuditTrail(t)
	al := trail.NewAuditLog("test-user", []string{"tugboat"})

	al.RecordError("tugboat", "policy", "POL-0003", "connection timeout")

	if len(al.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(al.Entries))
	}
	e := al.Entries[0]
	if e.Operation != "error" {
		t.Errorf("expected operation 'error', got %q", e.Operation)
	}
	if e.Details != "connection timeout" {
		t.Errorf("expected details 'connection timeout', got %q", e.Details)
	}
}

func TestAuditLog_Finalize(t *testing.T) {
	trail, _ := newTestAuditTrail(t)
	al := trail.NewAuditLog("test-user", []string{"tugboat", "github"})

	al.RecordPull("tugboat", "policy", "POL-0001", "", "abc")
	al.RecordPull("tugboat", "policy", "POL-0002", "", "def")
	al.RecordPush("github", "control", "CC-01", "local", "remote")
	al.RecordConflictResolved("tugboat", "control", "CC-02", "local_wins", "a", "b")
	al.RecordConflictManual("tugboat", "policy", "POL-0003", "x", "y")
	al.RecordError("github", "evidence_task", "ET-0001", "not found")

	al.Finalize()

	if al.EndTime.IsZero() {
		t.Error("expected EndTime to be set after Finalize")
	}
	s := al.Summary
	if s.TotalOperations != 6 {
		t.Errorf("expected total_operations=6, got %d", s.TotalOperations)
	}
	if s.Pulls != 2 {
		t.Errorf("expected pulls=2, got %d", s.Pulls)
	}
	if s.Pushes != 1 {
		t.Errorf("expected pushes=1, got %d", s.Pushes)
	}
	if s.ConflictsResolved != 1 {
		t.Errorf("expected conflicts_resolved=1, got %d", s.ConflictsResolved)
	}
	if s.ConflictsManual != 1 {
		t.Errorf("expected conflicts_manual=1, got %d", s.ConflictsManual)
	}
	if s.Errors != 1 {
		t.Errorf("expected errors=1, got %d", s.Errors)
	}
}

func TestAuditTrail_SaveAndLoad(t *testing.T) {
	trail, _ := newTestAuditTrail(t)
	al := trail.NewAuditLog("test-user", []string{"tugboat"})

	al.RecordPull("tugboat", "policy", "POL-0001", "before", "after")
	al.RecordPush("tugboat", "control", "CC-01", "local", "remote")
	al.Finalize()

	if err := trail.Save(al); err != nil {
		t.Fatalf("Save failed: %v", err)
	}

	loaded, err := trail.Load(al.RunID)
	if err != nil {
		t.Fatalf("Load failed: %v", err)
	}

	if loaded.RunID != al.RunID {
		t.Errorf("expected RunID %q, got %q", al.RunID, loaded.RunID)
	}
	if loaded.Actor != al.Actor {
		t.Errorf("expected Actor %q, got %q", al.Actor, loaded.Actor)
	}
	if len(loaded.Providers) != len(al.Providers) {
		t.Errorf("expected %d providers, got %d", len(al.Providers), len(loaded.Providers))
	}
	if len(loaded.Entries) != len(al.Entries) {
		t.Errorf("expected %d entries, got %d", len(al.Entries), len(loaded.Entries))
	}
	if loaded.Summary.TotalOperations != al.Summary.TotalOperations {
		t.Errorf("expected total_operations=%d, got %d", al.Summary.TotalOperations, loaded.Summary.TotalOperations)
	}
	if loaded.Summary.Pulls != al.Summary.Pulls {
		t.Errorf("expected pulls=%d, got %d", al.Summary.Pulls, loaded.Summary.Pulls)
	}
	if loaded.Summary.Pushes != al.Summary.Pushes {
		t.Errorf("expected pushes=%d, got %d", al.Summary.Pushes, loaded.Summary.Pushes)
	}
}

func TestAuditTrail_List(t *testing.T) {
	trail, _ := newTestAuditTrail(t)

	// Create multiple logs with distinct RunIDs.
	al1 := trail.NewAuditLog("user", []string{"tugboat"})
	al1.RunID = "20260101-100000"
	al1.Finalize()

	al2 := trail.NewAuditLog("user", []string{"tugboat"})
	al2.RunID = "20260102-100000"
	al2.Finalize()

	al3 := trail.NewAuditLog("user", []string{"tugboat"})
	al3.RunID = "20260103-100000"
	al3.Finalize()

	for _, al := range []*AuditLog{al1, al2, al3} {
		if err := trail.Save(al); err != nil {
			t.Fatalf("Save failed: %v", err)
		}
	}

	names, err := trail.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(names) != 3 {
		t.Fatalf("expected 3 files, got %d", len(names))
	}
	// Newest first.
	if names[0] != "sync-20260103-100000.yaml" {
		t.Errorf("expected newest first, got %q", names[0])
	}
	if names[2] != "sync-20260101-100000.yaml" {
		t.Errorf("expected oldest last, got %q", names[2])
	}
}

func TestAuditTrail_List_EmptyDir(t *testing.T) {
	trail, _ := newTestAuditTrail(t)

	names, err := trail.List()
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty list, got %d entries", len(names))
	}
}

func TestAuditTrail_Load_NotFound(t *testing.T) {
	trail, _ := newTestAuditTrail(t)

	_, err := trail.Load("99999999-999999")
	if err == nil {
		t.Fatal("expected error for nonexistent run ID, got nil")
	}
}

func TestAuditLog_EmptyLog(t *testing.T) {
	trail, _ := newTestAuditTrail(t)
	al := trail.NewAuditLog("test-user", []string{"tugboat"})

	al.Finalize()

	s := al.Summary
	if s.TotalOperations != 0 {
		t.Errorf("expected total_operations=0, got %d", s.TotalOperations)
	}
	if s.Pulls != 0 {
		t.Errorf("expected pulls=0, got %d", s.Pulls)
	}
	if s.Pushes != 0 {
		t.Errorf("expected pushes=0, got %d", s.Pushes)
	}
	if s.ConflictsResolved != 0 {
		t.Errorf("expected conflicts_resolved=0, got %d", s.ConflictsResolved)
	}
	if s.ConflictsManual != 0 {
		t.Errorf("expected conflicts_manual=0, got %d", s.ConflictsManual)
	}
	if s.Errors != 0 {
		t.Errorf("expected errors=0, got %d", s.Errors)
	}
}

func TestAuditTrail_SaveCreatesDir(t *testing.T) {
	log, err := logger.NewTestLogger()
	if err != nil {
		t.Fatalf("create test logger: %v", err)
	}

	// Use a subdirectory that doesn't exist yet.
	baseDir := filepath.Join(t.TempDir(), "nested", "audit")
	trail := NewAuditTrail(baseDir, log)

	al := trail.NewAuditLog("test-user", []string{"tugboat"})
	al.RecordPull("tugboat", "policy", "POL-0001", "", "abc")
	al.Finalize()

	if err := trail.Save(al); err != nil {
		t.Fatalf("Save failed when dir doesn't exist: %v", err)
	}

	// Verify the file exists.
	loaded, err := trail.Load(al.RunID)
	if err != nil {
		t.Fatalf("Load failed after Save created dir: %v", err)
	}
	if loaded.RunID != al.RunID {
		t.Errorf("expected RunID %q, got %q", al.RunID, loaded.RunID)
	}
}

func TestAuditLog_RunID_Format(t *testing.T) {
	trail, _ := newTestAuditTrail(t)
	al := trail.NewAuditLog("user", []string{"tugboat"})

	// RunID should be in YYYYMMDD-HHMMSS format.
	if len(al.RunID) != 15 { // "20260318-040000" = 15 chars
		t.Errorf("expected RunID length 15, got %d: %q", len(al.RunID), al.RunID)
	}

	// Should be parseable as time.
	_, err := time.Parse("20060102-150405", al.RunID)
	if err != nil {
		t.Errorf("RunID %q is not valid YYYYMMDD-HHMMSS: %v", al.RunID, err)
	}
}
