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

package tugboat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/grctool/grctool/internal/adapters"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
	tugboatclient "github.com/grctool/grctool/internal/tugboat"
)

// TestTugboatDataProvider_CompileTimeInterface verifies the compile-time
// interface assertion that TugboatDataProvider implements DataProvider.
func TestTugboatDataProvider_CompileTimeInterface(t *testing.T) {
	// This is enforced at compile time via the var _ declaration in provider.go.
	// This test simply documents the assertion.
	var _ interfaces.DataProvider = (*TugboatDataProvider)(nil)
}

// setupTestProvider creates an httptest server, a real tugboat.Client pointing
// at it, a real TugboatToDomain adapter, and wraps them in a TugboatDataProvider.
// The handler function receives the request and returns the response.
func setupTestProvider(t *testing.T, handler http.HandlerFunc) (*TugboatDataProvider, *httptest.Server) {
	t.Helper()

	server := httptest.NewServer(handler)

	log, err := logger.NewTestLogger()
	if err != nil {
		t.Fatalf("failed to create test logger: %v", err)
	}

	client := tugboatclient.NewClient(&config.TugboatConfig{
		BaseURL: server.URL,
	}, nil)

	adapter := adapters.NewTugboatToDomain()
	provider := NewTugboatDataProvider(client, adapter, "test-org", log)

	return provider, server
}

func TestTugboatDataProvider_Name(t *testing.T) {
	provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {})
	defer server.Close()

	if got := provider.Name(); got != "tugboat" {
		t.Errorf("Name() = %q, want %q", got, "tugboat")
	}
}

func TestTugboatDataProvider_TestConnection(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
			if strings.HasPrefix(r.URL.Path, "/api/policy/") {
				resp := map[string]interface{}{
					"count":    1,
					"results":  []interface{}{},
					"num_pages": 1,
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(resp)
				return
			}
			http.NotFound(w, r)
		})
		defer server.Close()

		err := provider.TestConnection(context.Background())
		if err != nil {
			t.Errorf("TestConnection() unexpected error: %v", err)
		}
	})

	t.Run("failure", func(t *testing.T) {
		provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte(`{"error":{"code":"unauthorized","message":"invalid token"}}`))
		})
		defer server.Close()

		err := provider.TestConnection(context.Background())
		if err == nil {
			t.Error("TestConnection() expected error, got nil")
		}
	})
}

func TestTugboatDataProvider_ListPolicies(t *testing.T) {
	provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/policy/") {
			resp := map[string]interface{}{
				"count":         2,
				"page_number":   1,
				"page_size":     25,
				"num_pages":     1,
				"max_page_size": 100,
				"results": []map[string]interface{}{
					{
						"id":          "101",
						"name":        "Access Control Policy",
						"description": "Controls access to systems",
						"framework":   "SOC2",
						"status":      "published",
						"created_at":  "2024-01-01T00:00:00Z",
						"updated_at":  "2024-06-15T00:00:00Z",
					},
					{
						"id":          "102",
						"name":        "Data Protection Policy",
						"description": "Protects sensitive data",
						"framework":   "SOC2",
						"status":      "published",
						"created_at":  "2024-02-01T00:00:00Z",
						"updated_at":  "2024-07-01T00:00:00Z",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	})
	defer server.Close()

	policies, count, err := provider.ListPolicies(context.Background(), interfaces.ListOptions{
		Page:     1,
		PageSize: 25,
	})
	if err != nil {
		t.Fatalf("ListPolicies() unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("ListPolicies() count = %d, want 2", count)
	}
	if len(policies) != 2 {
		t.Fatalf("ListPolicies() returned %d policies, want 2", len(policies))
	}
	if policies[0].Name != "Access Control Policy" {
		t.Errorf("ListPolicies()[0].Name = %q, want %q", policies[0].Name, "Access Control Policy")
	}
	if policies[0].ID != "101" {
		t.Errorf("ListPolicies()[0].ID = %q, want %q", policies[0].ID, "101")
	}
	if policies[1].Name != "Data Protection Policy" {
		t.Errorf("ListPolicies()[1].Name = %q, want %q", policies[1].Name, "Data Protection Policy")
	}
}

func TestTugboatDataProvider_ListPolicies_Pagination(t *testing.T) {
	provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/policy/") {
			// Verify pagination params are forwarded
			query := r.URL.RawQuery
			if !strings.Contains(query, "page=2") {
				t.Errorf("expected page=2 in query, got: %s", query)
			}
			if !strings.Contains(query, "page_size=10") {
				t.Errorf("expected page_size=10 in query, got: %s", query)
			}
			if !strings.Contains(query, "framework=ISO27001") {
				t.Errorf("expected framework=ISO27001 in query, got: %s", query)
			}
			resp := map[string]interface{}{
				"count":         15,
				"page_number":   2,
				"page_size":     10,
				"num_pages":     2,
				"max_page_size": 100,
				"results": []map[string]interface{}{
					{
						"id":          "111",
						"name":        "Page 2 Policy",
						"description": "On page 2",
						"framework":   "ISO27001",
						"status":      "published",
						"created_at":  "2024-01-01T00:00:00Z",
						"updated_at":  "2024-06-15T00:00:00Z",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	})
	defer server.Close()

	policies, count, err := provider.ListPolicies(context.Background(), interfaces.ListOptions{
		Page:      2,
		PageSize:  10,
		Framework: "ISO27001",
	})
	if err != nil {
		t.Fatalf("ListPolicies() unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("ListPolicies() count = %d, want 1", count)
	}
	if len(policies) != 1 {
		t.Fatalf("ListPolicies() returned %d policies, want 1", len(policies))
	}
	if policies[0].Name != "Page 2 Policy" {
		t.Errorf("ListPolicies()[0].Name = %q, want %q", policies[0].Name, "Page 2 Policy")
	}
}

func TestTugboatDataProvider_GetPolicy(t *testing.T) {
	provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/policy/101") {
			resp := map[string]interface{}{
				"id":          "101",
				"name":        "Access Control Policy",
				"description": "Controls access to systems",
				"summary":     "Summary of access control",
				"details":     "Full policy content here",
				"framework":   "SOC2",
				"status":      "published",
				"category":    "Access Control",
				"created_at":  "2024-01-01T00:00:00Z",
				"updated_at":  "2024-06-15T00:00:00Z",
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	})
	defer server.Close()

	policy, err := provider.GetPolicy(context.Background(), "101")
	if err != nil {
		t.Fatalf("GetPolicy() unexpected error: %v", err)
	}
	if policy.ID != "101" {
		t.Errorf("GetPolicy().ID = %q, want %q", policy.ID, "101")
	}
	if policy.Name != "Access Control Policy" {
		t.Errorf("GetPolicy().Name = %q, want %q", policy.Name, "Access Control Policy")
	}
	if policy.Summary != "Summary of access control" {
		t.Errorf("GetPolicy().Summary = %q, want %q", policy.Summary, "Summary of access control")
	}
}

func TestTugboatDataProvider_GetPolicy_NotFound(t *testing.T) {
	provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error":{"code":"not_found","message":"policy not found"}}`))
	})
	defer server.Close()

	_, err := provider.GetPolicy(context.Background(), "999")
	if err == nil {
		t.Error("GetPolicy() expected error for missing policy, got nil")
	}
}

func TestTugboatDataProvider_ListControls(t *testing.T) {
	provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/org_control/") {
			resp := map[string]interface{}{
				"count":         2,
				"page_number":   1,
				"page_size":     25,
				"num_pages":     1,
				"max_page_size": 100,
				"results": []map[string]interface{}{
					{
						"id":                501,
						"name":              "CC6.8 - Logical Access",
						"body":              "Restrict access to authorized users",
						"category":          "Common Criteria",
						"framework":         "SOC2",
						"status":            "implemented",
						"is_auto_implemented": false,
						"master_version_num": 1,
						"master_control_id":  100,
						"org_id":             1,
						"org_scope_id":       1,
					},
					{
						"id":                502,
						"name":              "CC7.1 - Security Monitoring",
						"body":              "Monitor for security events",
						"category":          "Common Criteria",
						"framework":         "SOC2",
						"status":            "implemented",
						"is_auto_implemented": true,
						"master_version_num": 1,
						"master_control_id":  101,
						"org_id":             1,
						"org_scope_id":       1,
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	})
	defer server.Close()

	controls, count, err := provider.ListControls(context.Background(), interfaces.ListOptions{
		Page:     1,
		PageSize: 25,
	})
	if err != nil {
		t.Fatalf("ListControls() unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("ListControls() count = %d, want 2", count)
	}
	if len(controls) != 2 {
		t.Fatalf("ListControls() returned %d controls, want 2", len(controls))
	}
	if controls[0].Name != "CC6.8 - Logical Access" {
		t.Errorf("ListControls()[0].Name = %q, want %q", controls[0].Name, "CC6.8 - Logical Access")
	}
	if controls[0].ID != "501" {
		t.Errorf("ListControls()[0].ID = %q, want %q", controls[0].ID, "501")
	}
}

func TestTugboatDataProvider_GetControl(t *testing.T) {
	provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/org_control/501") {
			resp := map[string]interface{}{
				"id":                  501,
				"name":                "CC6.8 - Logical Access",
				"body":                "Restrict access to authorized users",
				"category":            "Common Criteria",
				"framework":           "SOC2",
				"status":              "implemented",
				"is_auto_implemented": false,
				"master_version_num":  1,
				"master_control_id":   100,
				"org_id":              1,
				"org_scope_id":        1,
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	})
	defer server.Close()

	control, err := provider.GetControl(context.Background(), "501")
	if err != nil {
		t.Fatalf("GetControl() unexpected error: %v", err)
	}
	if control.ID != "501" {
		t.Errorf("GetControl().ID = %q, want %q", control.ID, "501")
	}
	if control.Name != "CC6.8 - Logical Access" {
		t.Errorf("GetControl().Name = %q, want %q", control.Name, "CC6.8 - Logical Access")
	}
	if control.Description != "Restrict access to authorized users" {
		t.Errorf("GetControl().Description = %q, want %q", control.Description, "Restrict access to authorized users")
	}
}

func TestTugboatDataProvider_ListEvidenceTasks(t *testing.T) {
	provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/org_evidence/") {
			resp := map[string]interface{}{
				"count":         2,
				"page_number":   1,
				"page_size":     25,
				"num_pages":     1,
				"max_page_size": 100,
				"results": []map[string]interface{}{
					{
						"id":                  1001,
						"name":                "GitHub Repository Access Controls",
						"description":         "Show team permissions for repos",
						"collection_interval": "quarter",
						"completed":           false,
						"framework":           "SOC2",
						"status":              "pending",
						"created_at":          "2024-01-15T00:00:00Z",
						"updated_at":          "2024-06-01T00:00:00Z",
					},
					{
						"id":                  1002,
						"name":                "Terraform Security Config",
						"description":         "Infrastructure security evidence",
						"collection_interval": "month",
						"completed":           true,
						"framework":           "SOC2",
						"status":              "completed",
						"created_at":          "2024-02-01T00:00:00Z",
						"updated_at":          "2024-07-01T00:00:00Z",
					},
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	})
	defer server.Close()

	tasks, count, err := provider.ListEvidenceTasks(context.Background(), interfaces.ListOptions{
		Page:     1,
		PageSize: 25,
	})
	if err != nil {
		t.Fatalf("ListEvidenceTasks() unexpected error: %v", err)
	}
	if count != 2 {
		t.Errorf("ListEvidenceTasks() count = %d, want 2", count)
	}
	if len(tasks) != 2 {
		t.Fatalf("ListEvidenceTasks() returned %d tasks, want 2", len(tasks))
	}
	if tasks[0].Name != "GitHub Repository Access Controls" {
		t.Errorf("ListEvidenceTasks()[0].Name = %q, want %q", tasks[0].Name, "GitHub Repository Access Controls")
	}
	if tasks[0].ID != "1001" {
		t.Errorf("ListEvidenceTasks()[0].ID = %q, want %q", tasks[0].ID, "1001")
	}
	if tasks[1].Completed != true {
		t.Error("ListEvidenceTasks()[1].Completed = false, want true")
	}
}

func TestTugboatDataProvider_GetEvidenceTask(t *testing.T) {
	provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/org_evidence/1001") {
			resp := map[string]interface{}{
				"id":                  1001,
				"name":                "GitHub Repository Access Controls",
				"description":         "Show team permissions for repos",
				"collection_interval": "quarter",
				"completed":           false,
				"framework":           "SOC2",
				"status":              "pending",
				"created_at":          "2024-01-15T00:00:00Z",
				"updated_at":          "2024-06-01T00:00:00Z",
				"master_content": map[string]interface{}{
					"guidance": "Collect GitHub team permission screenshots",
				},
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}
		http.NotFound(w, r)
	})
	defer server.Close()

	task, err := provider.GetEvidenceTask(context.Background(), "1001")
	if err != nil {
		t.Fatalf("GetEvidenceTask() unexpected error: %v", err)
	}
	if task.ID != "1001" {
		t.Errorf("GetEvidenceTask().ID = %q, want %q", task.ID, "1001")
	}
	if task.Name != "GitHub Repository Access Controls" {
		t.Errorf("GetEvidenceTask().Name = %q, want %q", task.Name, "GitHub Repository Access Controls")
	}
	if task.Guidance != "Collect GitHub team permission screenshots" {
		t.Errorf("GetEvidenceTask().Guidance = %q, want %q", task.Guidance, "Collect GitHub team permission screenshots")
	}
}

// ---------------------------------------------------------------------------
// ContentHash verification
// ---------------------------------------------------------------------------

func TestTugboatDataProvider_ContentHash_Policy(t *testing.T) {
	provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"count": 1,
			"results": []map[string]interface{}{
				{"id": 100, "name": "Test Policy", "body": "Content"},
			},
			"num_pages": 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	policies, _, err := provider.ListPolicies(context.Background(), interfaces.ListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListPolicies() error: %v", err)
	}
	if len(policies) == 0 {
		t.Fatal("expected at least one policy")
	}

	p := policies[0]
	if p.SyncMetadata == nil {
		t.Fatal("SyncMetadata should not be nil")
	}
	hash, ok := p.SyncMetadata.ContentHash["tugboat"]
	if !ok || hash == "" {
		t.Error("ContentHash[tugboat] should be non-empty")
	}
	if len(hash) != 64 {
		t.Errorf("ContentHash should be 64-char SHA-256 hex, got %d chars", len(hash))
	}

	// Verify determinism: calling again should produce the same hash.
	policies2, _, _ := provider.ListPolicies(context.Background(), interfaces.ListOptions{Page: 1, PageSize: 10})
	hash2 := policies2[0].SyncMetadata.ContentHash["tugboat"]
	if hash != hash2 {
		t.Errorf("ContentHash should be deterministic: %q != %q", hash, hash2)
	}
}

func TestTugboatDataProvider_ContentHash_Control(t *testing.T) {
	provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"count": 1,
			"results": []map[string]interface{}{
				{"id": 200, "name": "Access Control", "body": "Controls access"},
			},
			"num_pages": 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	controls, _, err := provider.ListControls(context.Background(), interfaces.ListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListControls() error: %v", err)
	}
	if len(controls) == 0 {
		t.Fatal("expected at least one control")
	}

	c := controls[0]
	if c.SyncMetadata == nil {
		t.Fatal("SyncMetadata should not be nil")
	}
	hash, ok := c.SyncMetadata.ContentHash["tugboat"]
	if !ok || hash == "" {
		t.Error("ContentHash[tugboat] should be non-empty")
	}
	if len(hash) != 64 {
		t.Errorf("ContentHash should be 64-char SHA-256 hex, got %d chars", len(hash))
	}
}

func TestTugboatDataProvider_ContentHash_EvidenceTask(t *testing.T) {
	provider, server := setupTestProvider(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]interface{}{
			"count": 1,
			"results": []map[string]interface{}{
				{"id": 300, "name": "Collect Evidence", "description": "Details"},
			},
			"num_pages": 1,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})
	defer server.Close()

	tasks, _, err := provider.ListEvidenceTasks(context.Background(), interfaces.ListOptions{Page: 1, PageSize: 10})
	if err != nil {
		t.Fatalf("ListEvidenceTasks() error: %v", err)
	}
	if len(tasks) == 0 {
		t.Fatal("expected at least one task")
	}

	task := tasks[0]
	if task.SyncMetadata == nil {
		t.Fatal("SyncMetadata should not be nil")
	}
	hash, ok := task.SyncMetadata.ContentHash["tugboat"]
	if !ok || hash == "" {
		t.Error("ContentHash[tugboat] should be non-empty")
	}
	if len(hash) != 64 {
		t.Errorf("ContentHash should be 64-char SHA-256 hex, got %d chars", len(hash))
	}
}
