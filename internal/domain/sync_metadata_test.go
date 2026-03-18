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

package domain

import (
	"encoding/json"
	"testing"
	"time"
)

func TestSyncMetadata_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second).UTC()
	sm := SyncMetadata{
		LastSyncTime: map[string]time.Time{
			"tugboat":       now,
			"accountablehq": now.Add(-1 * time.Hour),
		},
		ContentHash: map[string]string{
			"tugboat":       "abc123",
			"accountablehq": "def456",
		},
		ConflictState: "both_modified",
	}

	data, err := json.Marshal(sm)
	if err != nil {
		t.Fatalf("Failed to marshal SyncMetadata: %v", err)
	}

	var decoded SyncMetadata
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal SyncMetadata: %v", err)
	}

	if !decoded.LastSyncTime["tugboat"].Equal(now) {
		t.Errorf("LastSyncTime[tugboat] = %v, want %v", decoded.LastSyncTime["tugboat"], now)
	}
	if !decoded.LastSyncTime["accountablehq"].Equal(now.Add(-1 * time.Hour)) {
		t.Errorf("LastSyncTime[accountablehq] mismatch")
	}
	if decoded.ContentHash["tugboat"] != "abc123" {
		t.Errorf("ContentHash[tugboat] = %q, want %q", decoded.ContentHash["tugboat"], "abc123")
	}
	if decoded.ContentHash["accountablehq"] != "def456" {
		t.Errorf("ContentHash[accountablehq] = %q, want %q", decoded.ContentHash["accountablehq"], "def456")
	}
	if decoded.ConflictState != "both_modified" {
		t.Errorf("ConflictState = %q, want %q", decoded.ConflictState, "both_modified")
	}
}

func TestSyncMetadata_Empty(t *testing.T) {
	t.Parallel()

	// A nil *SyncMetadata on a struct should not appear in JSON (omitempty)
	type wrapper struct {
		Name         string        `json:"name"`
		SyncMetadata *SyncMetadata `json:"sync_metadata,omitempty"`
	}

	w := wrapper{Name: "test"}
	data, err := json.Marshal(w)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	jsonStr := string(data)
	if contains(jsonStr, "sync_metadata") {
		t.Errorf("nil SyncMetadata should be omitted from JSON, got: %s", jsonStr)
	}

	// Empty SyncMetadata (zero value) should still serialize (it's not nil pointer)
	w2 := wrapper{Name: "test", SyncMetadata: &SyncMetadata{}}
	data2, err := json.Marshal(w2)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	// Pointer is non-nil so it will appear, but inner fields should be omitted
	var decoded wrapper
	if err := json.Unmarshal(data2, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}
	if decoded.SyncMetadata == nil {
		t.Error("Expected non-nil SyncMetadata after round-trip of empty struct")
	}
}

func TestSyncMetadata_PerProvider(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second).UTC()
	sm := SyncMetadata{
		LastSyncTime: map[string]time.Time{
			"tugboat":       now,
			"accountablehq": now.Add(-2 * time.Hour),
			"gdrive":        now.Add(-5 * time.Minute),
		},
		ContentHash: map[string]string{
			"tugboat":       "hash1",
			"accountablehq": "hash2",
			"gdrive":        "hash3",
		},
	}

	// Each provider should be tracked independently
	if len(sm.LastSyncTime) != 3 {
		t.Errorf("Expected 3 providers in LastSyncTime, got %d", len(sm.LastSyncTime))
	}
	if len(sm.ContentHash) != 3 {
		t.Errorf("Expected 3 providers in ContentHash, got %d", len(sm.ContentHash))
	}

	// Round-trip
	data, err := json.Marshal(sm)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded SyncMetadata
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	for _, provider := range []string{"tugboat", "accountablehq", "gdrive"} {
		if _, ok := decoded.LastSyncTime[provider]; !ok {
			t.Errorf("Missing provider %q in LastSyncTime after round-trip", provider)
		}
		if _, ok := decoded.ContentHash[provider]; !ok {
			t.Errorf("Missing provider %q in ContentHash after round-trip", provider)
		}
	}
}

func TestExternalIDs_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		entity interface{ getExternalIDs() map[string]string }
	}{
		{
			name: "Policy",
			entity: &policyWrapper{Policy{
				ID:          "POL-0001",
				Name:        "Access Control Policy",
				ExternalIDs: map[string]string{"tugboat": "94641"},
			}},
		},
		{
			name: "Control",
			entity: &controlWrapper{Control{
				ID:          "778805",
				Name:        "CC6.8",
				ExternalIDs: map[string]string{"tugboat": "778805"},
			}},
		},
		{
			name: "EvidenceTask",
			entity: &evidenceTaskWrapper{EvidenceTask{
				ID:          "327992",
				Name:        "GitHub Access Controls",
				ExternalIDs: map[string]string{"tugboat": "327992"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.entity)
			if err != nil {
				t.Fatalf("Failed to marshal: %v", err)
			}

			ids := tt.entity.getExternalIDs()
			if ids["tugboat"] == "" {
				t.Error("ExternalIDs[tugboat] should not be empty")
			}

			// Verify JSON contains external_ids
			if !contains(string(data), "external_ids") {
				t.Errorf("JSON should contain external_ids, got: %s", string(data))
			}
		})
	}
}

// wrapper types to provide getExternalIDs for test table
type policyWrapper struct{ Policy }

func (w *policyWrapper) getExternalIDs() map[string]string { return w.ExternalIDs }

type controlWrapper struct{ Control }

func (w *controlWrapper) getExternalIDs() map[string]string { return w.ExternalIDs }

type evidenceTaskWrapper struct{ EvidenceTask }

func (w *evidenceTaskWrapper) getExternalIDs() map[string]string { return w.ExternalIDs }

func TestExternalIDs_OmitEmpty(t *testing.T) {
	t.Parallel()

	// Entities without ExternalIDs should not have external_ids in JSON
	p := Policy{ID: "POL-0001", Name: "Test Policy"}
	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	if contains(string(data), "external_ids") {
		t.Errorf("JSON should not contain external_ids when nil, got: %s", string(data))
	}

	c := Control{ID: "123", Name: "Test Control"}
	data, err = json.Marshal(c)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	if contains(string(data), "external_ids") {
		t.Errorf("JSON should not contain external_ids when nil, got: %s", string(data))
	}

	et := EvidenceTask{ID: "456", Name: "Test Task"}
	data, err = json.Marshal(et)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}
	if contains(string(data), "external_ids") {
		t.Errorf("JSON should not contain external_ids when nil, got: %s", string(data))
	}
}

func TestExternalIDs_MultipleProviders(t *testing.T) {
	t.Parallel()

	p := Policy{
		ID:   "POL-0001",
		Name: "Multi-Provider Policy",
		ExternalIDs: map[string]string{
			"tugboat":       "123",
			"accountablehq": "abc",
		},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	var decoded Policy
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	if decoded.ExternalIDs["tugboat"] != "123" {
		t.Errorf("ExternalIDs[tugboat] = %q, want %q", decoded.ExternalIDs["tugboat"], "123")
	}
	if decoded.ExternalIDs["accountablehq"] != "abc" {
		t.Errorf("ExternalIDs[accountablehq] = %q, want %q", decoded.ExternalIDs["accountablehq"], "abc")
	}
}

func TestPolicy_WithSyncMetadata(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second).UTC()
	p := Policy{
		ID:          "POL-0001",
		Name:        "Access Control Policy",
		Description: "Defines access control requirements",
		Status:      "published",
		CreatedAt:   now,
		UpdatedAt:   now,
		ExternalIDs: map[string]string{"tugboat": "94641"},
		SyncMetadata: &SyncMetadata{
			LastSyncTime:  map[string]time.Time{"tugboat": now},
			ContentHash:   map[string]string{"tugboat": "sha256:abc"},
			ConflictState: "",
		},
	}

	data, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("Failed to marshal Policy with SyncMetadata: %v", err)
	}

	var decoded Policy
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Policy with SyncMetadata: %v", err)
	}

	if decoded.ExternalIDs["tugboat"] != "94641" {
		t.Errorf("ExternalIDs[tugboat] = %q, want %q", decoded.ExternalIDs["tugboat"], "94641")
	}
	if decoded.SyncMetadata == nil {
		t.Fatal("SyncMetadata should not be nil after round-trip")
	}
	if !decoded.SyncMetadata.LastSyncTime["tugboat"].Equal(now) {
		t.Errorf("SyncMetadata.LastSyncTime[tugboat] mismatch")
	}
	if decoded.SyncMetadata.ContentHash["tugboat"] != "sha256:abc" {
		t.Errorf("SyncMetadata.ContentHash[tugboat] = %q, want %q", decoded.SyncMetadata.ContentHash["tugboat"], "sha256:abc")
	}
	if decoded.Name != "Access Control Policy" {
		t.Errorf("Name = %q, want %q", decoded.Name, "Access Control Policy")
	}
}

func TestControl_WithSyncMetadata(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second).UTC()
	c := Control{
		ID:          "778805",
		Name:        "CC6.8",
		Description: "Logical access controls",
		Status:      "implemented",
		ExternalIDs: map[string]string{"tugboat": "778805"},
		SyncMetadata: &SyncMetadata{
			LastSyncTime:  map[string]time.Time{"tugboat": now},
			ContentHash:   map[string]string{"tugboat": "sha256:def"},
			ConflictState: "resolved",
		},
	}

	data, err := json.Marshal(c)
	if err != nil {
		t.Fatalf("Failed to marshal Control with SyncMetadata: %v", err)
	}

	var decoded Control
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Control with SyncMetadata: %v", err)
	}

	if decoded.ExternalIDs["tugboat"] != "778805" {
		t.Errorf("ExternalIDs[tugboat] = %q, want %q", decoded.ExternalIDs["tugboat"], "778805")
	}
	if decoded.SyncMetadata == nil {
		t.Fatal("SyncMetadata should not be nil after round-trip")
	}
	if decoded.SyncMetadata.ConflictState != "resolved" {
		t.Errorf("ConflictState = %q, want %q", decoded.SyncMetadata.ConflictState, "resolved")
	}
}

func TestEvidenceTask_WithSyncMetadata(t *testing.T) {
	t.Parallel()

	now := time.Now().Truncate(time.Second).UTC()
	et := EvidenceTask{
		ID:          "327992",
		Name:        "GitHub Access Controls",
		Description: "Show team permissions",
		Status:      "pending",
		ExternalIDs: map[string]string{"tugboat": "327992"},
		SyncMetadata: &SyncMetadata{
			LastSyncTime:  map[string]time.Time{"tugboat": now},
			ContentHash:   map[string]string{"tugboat": "sha256:ghi"},
			ConflictState: "local_modified",
		},
	}

	data, err := json.Marshal(et)
	if err != nil {
		t.Fatalf("Failed to marshal EvidenceTask with SyncMetadata: %v", err)
	}

	var decoded EvidenceTask
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal EvidenceTask with SyncMetadata: %v", err)
	}

	if decoded.ExternalIDs["tugboat"] != "327992" {
		t.Errorf("ExternalIDs[tugboat] = %q, want %q", decoded.ExternalIDs["tugboat"], "327992")
	}
	if decoded.SyncMetadata == nil {
		t.Fatal("SyncMetadata should not be nil after round-trip")
	}
	if decoded.SyncMetadata.ConflictState != "local_modified" {
		t.Errorf("ConflictState = %q, want %q", decoded.SyncMetadata.ConflictState, "local_modified")
	}
}

func TestBackwardCompatibility(t *testing.T) {
	t.Parallel()

	// JSON without external_ids or sync_metadata should deserialize correctly
	t.Run("Policy_NoNewFields", func(t *testing.T) {
		jsonData := `{"id":"POL-0001","name":"Test Policy","description":"A test","status":"published","created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z","control_count":5,"procedure_count":2,"evidence_count":10,"risk_count":1,"view_count":0,"download_count":0,"reference_count":0}`
		var p Policy
		if err := json.Unmarshal([]byte(jsonData), &p); err != nil {
			t.Fatalf("Failed to unmarshal legacy Policy JSON: %v", err)
		}
		if p.ID != "POL-0001" {
			t.Errorf("ID = %q, want %q", p.ID, "POL-0001")
		}
		if p.ExternalIDs != nil {
			t.Errorf("ExternalIDs should be nil for legacy JSON, got %v", p.ExternalIDs)
		}
		if p.SyncMetadata != nil {
			t.Errorf("SyncMetadata should be nil for legacy JSON, got %v", p.SyncMetadata)
		}
	})

	t.Run("Control_NoNewFields", func(t *testing.T) {
		jsonData := `{"id":"778805","reference_id":"CC6.8","name":"Test Control","description":"desc","category":"Security","framework":"SOC2","status":"implemented","is_auto_implemented":false,"master_version_num":0,"master_control_id":0,"org_id":0,"org_scope_id":0,"recommended_evidence_count":0,"open_incident_count":0,"view_count":0,"download_count":0,"reference_count":0}`
		var c Control
		if err := json.Unmarshal([]byte(jsonData), &c); err != nil {
			t.Fatalf("Failed to unmarshal legacy Control JSON: %v", err)
		}
		if c.ID != "778805" {
			t.Errorf("ID = %q, want %q", c.ID, "778805")
		}
		if c.ExternalIDs != nil {
			t.Errorf("ExternalIDs should be nil for legacy JSON, got %v", c.ExternalIDs)
		}
		if c.SyncMetadata != nil {
			t.Errorf("SyncMetadata should be nil for legacy JSON, got %v", c.SyncMetadata)
		}
	})

	t.Run("EvidenceTask_NoNewFields", func(t *testing.T) {
		jsonData := `{"id":"327992","name":"Test Task","description":"desc","guidance":"","collection_interval":"quarter","priority":"medium","framework":"SOC2","status":"pending","completed":false,"due_days_before":0,"ad_hoc":false,"sensitive":false,"created_at":"2025-01-01T00:00:00Z","updated_at":"2025-01-01T00:00:00Z","master_version_num":0,"master_evidence_id":0,"org_id":0,"org_scope_id":0,"open_incident_count":0,"view_count":0,"download_count":0,"reference_count":0}`
		var et EvidenceTask
		if err := json.Unmarshal([]byte(jsonData), &et); err != nil {
			t.Fatalf("Failed to unmarshal legacy EvidenceTask JSON: %v", err)
		}
		if et.ID != "327992" {
			t.Errorf("ID = %q, want %q", et.ID, "327992")
		}
		if et.ExternalIDs != nil {
			t.Errorf("ExternalIDs should be nil for legacy JSON, got %v", et.ExternalIDs)
		}
		if et.SyncMetadata != nil {
			t.Errorf("SyncMetadata should be nil for legacy JSON, got %v", et.SyncMetadata)
		}
	})
}

// contains checks if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && searchString(s, substr)
}

func searchString(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
