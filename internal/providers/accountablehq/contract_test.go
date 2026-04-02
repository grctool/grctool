// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package accountablehq

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Contract tests: verify AccountableHQSyncProvider satisfies DataProvider
// contract with a real HTTP client backed by httptest.
// ---------------------------------------------------------------------------

func newContractSetup(t *testing.T) (*AccountableHQSyncProvider, *httptest.Server) {
	t.Helper()

	mux := http.NewServeMux()

	// List policies
	mux.HandleFunc("/api/v1/policies", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			// POST for create
			if r.Method == "POST" {
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": map[string]string{"id": "ahq-created"},
				})
				return
			}
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		json.NewEncoder(w).Encode(apiResponse{
			Data: json.RawMessage(`[
				{"id":"ahq-1","title":"Contract Policy","content":"# Contract","status":"active","version":1,"created_at":"2025-01-01T00:00:00Z","updated_at":"2026-03-01T00:00:00Z"},
				{"id":"ahq-2","title":"Second Policy","content":"# Second","status":"draft","version":2,"created_at":"2025-06-01T00:00:00Z","updated_at":"2026-02-01T00:00:00Z"}
			]`),
			Meta: &apiMeta{Page: 1, PerPage: 25, Total: 2},
		})
	})

	// Get/Update/Delete single policy
	mux.HandleFunc("/api/v1/policies/ahq-1", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			json.NewEncoder(w).Encode(AHQPolicy{
				ID: "ahq-1", Title: "Contract Policy", Content: "# Contract", Status: "active", Version: 1,
				CreatedAt: time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC),
				UpdatedAt: time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
			})
		case "PUT":
			w.WriteHeader(http.StatusOK)
		case "DELETE":
			w.WriteHeader(http.StatusNoContent)
		}
	})

	mux.HandleFunc("/api/v1/policies/nonexistent", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})

	server := httptest.NewServer(mux)

	client := NewHTTPClient(HTTPClientConfig{BaseURL: server.URL, APIKey: "contract-key"})
	provider := NewAccountableHQSyncProvider(client, testhelpers.NewStubLogger())

	return provider, server
}

// --- DataProvider contract ---

func TestContract_Name(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()
	assert.NotEmpty(t, p.Name())
}

func TestContract_TestConnection(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()
	assert.NoError(t, p.TestConnection(context.Background()))
}

func TestContract_ListPolicies_ReturnsResults(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	policies, count, err := p.ListPolicies(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Greater(t, count, 0)
	assert.NotEmpty(t, policies)
	assert.NotEmpty(t, policies[0].ExternalIDs["accountablehq"])
}

func TestContract_ListPolicies_Pagination(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	policies, _, err := p.ListPolicies(context.Background(), interfaces.ListOptions{PageSize: 1})
	require.NoError(t, err)
	// Server returns all (pagination is server-side), but provider requests per_page=1
	assert.NotEmpty(t, policies)
}

func TestContract_GetPolicy_Exists(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	policy, err := p.GetPolicy(context.Background(), "ahq-1")
	require.NoError(t, err)
	assert.Equal(t, "ahq-1", policy.ID)
	assert.Equal(t, "Contract Policy", policy.Name)
	assert.NotEmpty(t, policy.Content)
	assert.Equal(t, "ahq-1", policy.ExternalIDs["accountablehq"])
}

func TestContract_GetPolicy_NotFound(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	_, err := p.GetPolicy(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestContract_ListControls_Empty(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	controls, count, err := p.ListControls(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, controls)
}

func TestContract_GetControl_Error(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	_, err := p.GetControl(context.Background(), "any")
	assert.Error(t, err)
}

func TestContract_ListEvidenceTasks_Empty(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	tasks, count, err := p.ListEvidenceTasks(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, count)
	assert.Empty(t, tasks)
}

// --- SyncProvider contract ---

func TestContract_PushPolicy_Create(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	p2 := makeDomainPolicy("New Policy", "# New Policy Content")
	pol := &p2
	err := p.PushPolicy(context.Background(), pol)
	require.NoError(t, err)
	assert.Equal(t, "ahq-created", pol.ExternalIDs["accountablehq"])
}

func TestContract_PushPolicy_Update(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	p2 := makeDomainPolicyWithExt("Updated", "# Updated", "ahq-1")
	pol := &p2
	err := p.PushPolicy(context.Background(), pol)
	assert.NoError(t, err)
}

func TestContract_DeletePolicy(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	err := p.DeletePolicy(context.Background(), "ahq-1")
	assert.NoError(t, err)
}

func TestContract_DetectChanges(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	since := time.Date(2026, 2, 15, 0, 0, 0, 0, time.UTC)
	changes, err := p.DetectChanges(context.Background(), since)
	require.NoError(t, err)
	assert.Equal(t, "accountablehq", changes.Provider)
	// ahq-1 updated 2026-03-01 (after since), ahq-2 updated 2026-02-01 (before since)
	assert.Len(t, changes.Changes, 1)
	assert.Equal(t, "ahq-1", changes.Changes[0].EntityID)
}

func TestContract_ResolveConflict_AllStrategies(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	conflict := interfaces.Conflict{
		EntityType: "policy",
		EntityID:   "POL-0001",
		Provider:   "accountablehq",
		LocalHash:  "abc",
		RemoteHash: "def",
		DetectedAt: time.Now(),
	}

	for _, strategy := range []interfaces.ConflictResolution{
		interfaces.ConflictResolutionLocalWins,
		interfaces.ConflictResolutionRemoteWins,
		interfaces.ConflictResolutionNewestWins,
		interfaces.ConflictResolutionManual,
	} {
		err := p.ResolveConflict(context.Background(), conflict, strategy)
		assert.NoError(t, err, "ResolveConflict(%s) should succeed", strategy)
	}
}

func TestContract_ResolveConflict_UnsupportedEntityType(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	conflict := interfaces.Conflict{
		EntityType: "control",
		EntityID:   "CC-06.1",
		Provider:   "accountablehq",
	}
	err := p.ResolveConflict(context.Background(), conflict, interfaces.ConflictResolutionLocalWins)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "only policy conflicts")
}

func TestContract_ResolveConflict_AuditTrail(t *testing.T) {
	t.Parallel()
	p, s := newContractSetup(t)
	defer s.Close()

	conflict := interfaces.Conflict{
		EntityType: "policy",
		EntityID:   "POL-0001",
		Provider:   "accountablehq",
		LocalHash:  "abc",
		RemoteHash: "def",
		DetectedAt: time.Now(),
	}

	p.ClearAuditLog()
	require.NoError(t, p.ResolveConflict(context.Background(), conflict, interfaces.ConflictResolutionLocalWins))

	entries := p.AuditLog()
	require.Len(t, entries, 1)
	assert.Equal(t, "conflict_resolved", entries[0].Action)
	assert.Equal(t, "local_wins", entries[0].Resolution)
	assert.Equal(t, "local", entries[0].Winner)
	assert.Equal(t, "POL-0001", entries[0].PolicyID)
}

// --- helpers ---

func makeDomainPolicy(name, content string) domain.Policy {
	return domain.Policy{Name: name, Content: content}
}

func makeDomainPolicyWithExt(name, content, ahqID string) domain.Policy {
	return domain.Policy{
		Name:        name,
		Content:     content,
		ExternalIDs: map[string]string{"accountablehq": ahqID},
	}
}
