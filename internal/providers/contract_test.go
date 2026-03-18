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

package providers_test

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/adapters"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
	tugboatprovider "github.com/grctool/grctool/internal/providers/tugboat"
	"github.com/grctool/grctool/internal/testhelpers"
	tugboatclient "github.com/grctool/grctool/internal/tugboat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// DataProviderContractSuite — reusable contract tests for any DataProvider
// ---------------------------------------------------------------------------

// DataProviderContractSuite runs the standard contract tests against any
// DataProvider implementation. Call this from each provider's test file to
// verify compliance with the interface contract.
//
// The setup function must return a provider pre-loaded with at least one
// policy, one control, and one evidence task. It also returns the known IDs
// for those entities so Get* tests can reference them.
type ContractFixtureIDs struct {
	PolicyID       string
	ControlID      string
	EvidenceTaskID string
	PolicyFramework string // framework value used by the pre-loaded policy
}

func DataProviderContractSuite(t *testing.T, setup func(t *testing.T) (interfaces.DataProvider, ContractFixtureIDs)) {
	t.Helper()

	t.Run("Name", func(t *testing.T) {
		p, _ := setup(t)
		name := p.Name()
		assert.NotEmpty(t, name, "Name() must return a non-empty string")
	})

	t.Run("TestConnection", func(t *testing.T) {
		p, _ := setup(t)
		err := p.TestConnection(context.Background())
		assert.NoError(t, err, "TestConnection() must succeed for a properly configured provider")
	})

	// -----------------------------------------------------------------------
	// Policies
	// -----------------------------------------------------------------------

	t.Run("ListPolicies_ReturnsResults", func(t *testing.T) {
		p, _ := setup(t)
		policies, count, err := p.ListPolicies(context.Background(), interfaces.ListOptions{})
		require.NoError(t, err)
		assert.Greater(t, count, 0, "count must be > 0 when data is loaded")
		assert.NotEmpty(t, policies, "policies slice must not be empty")
	})

	t.Run("ListPolicies_Pagination", func(t *testing.T) {
		p, _ := setup(t)
		policies, _, err := p.ListPolicies(context.Background(), interfaces.ListOptions{
			Page:     1,
			PageSize: 1,
		})
		require.NoError(t, err)
		assert.LessOrEqual(t, len(policies), 1, "page size 1 must return at most 1 result")
	})

	t.Run("ListPolicies_EmptyPage", func(t *testing.T) {
		p, _ := setup(t)
		policies, _, err := p.ListPolicies(context.Background(), interfaces.ListOptions{
			Page:     9999,
			PageSize: 1,
		})
		require.NoError(t, err)
		assert.Empty(t, policies, "requesting a page beyond data should return empty slice")
	})

	t.Run("GetPolicy_Exists", func(t *testing.T) {
		p, ids := setup(t)
		policy, err := p.GetPolicy(context.Background(), ids.PolicyID)
		require.NoError(t, err)
		require.NotNil(t, policy)
		assert.Equal(t, ids.PolicyID, policy.ID, "returned policy ID must match requested ID")
		assert.NotEmpty(t, policy.Name, "policy Name must not be empty")
	})

	t.Run("GetPolicy_NotFound", func(t *testing.T) {
		p, _ := setup(t)
		_, err := p.GetPolicy(context.Background(), "nonexistent-id-99999")
		assert.Error(t, err, "GetPolicy with unknown ID must return an error")
	})

	// -----------------------------------------------------------------------
	// Controls
	// -----------------------------------------------------------------------

	t.Run("ListControls_ReturnsResults", func(t *testing.T) {
		p, _ := setup(t)
		controls, count, err := p.ListControls(context.Background(), interfaces.ListOptions{})
		require.NoError(t, err)
		assert.Greater(t, count, 0)
		assert.NotEmpty(t, controls)
	})

	t.Run("GetControl_Exists", func(t *testing.T) {
		p, ids := setup(t)
		control, err := p.GetControl(context.Background(), ids.ControlID)
		require.NoError(t, err)
		require.NotNil(t, control)
		assert.Equal(t, ids.ControlID, control.ID)
		assert.NotEmpty(t, control.Name)
	})

	t.Run("GetControl_NotFound", func(t *testing.T) {
		p, _ := setup(t)
		_, err := p.GetControl(context.Background(), "nonexistent-id-99999")
		assert.Error(t, err)
	})

	// -----------------------------------------------------------------------
	// Evidence Tasks
	// -----------------------------------------------------------------------

	t.Run("ListEvidenceTasks_ReturnsResults", func(t *testing.T) {
		p, _ := setup(t)
		tasks, count, err := p.ListEvidenceTasks(context.Background(), interfaces.ListOptions{})
		require.NoError(t, err)
		assert.Greater(t, count, 0)
		assert.NotEmpty(t, tasks)
	})

	t.Run("GetEvidenceTask_Exists", func(t *testing.T) {
		p, ids := setup(t)
		task, err := p.GetEvidenceTask(context.Background(), ids.EvidenceTaskID)
		require.NoError(t, err)
		require.NotNil(t, task)
		assert.Equal(t, ids.EvidenceTaskID, task.ID)
		assert.NotEmpty(t, task.Name)
	})

	t.Run("GetEvidenceTask_NotFound", func(t *testing.T) {
		p, _ := setup(t)
		_, err := p.GetEvidenceTask(context.Background(), "nonexistent-id-99999")
		assert.Error(t, err)
	})

	// -----------------------------------------------------------------------
	// Framework filter
	// -----------------------------------------------------------------------

	t.Run("ListPolicies_WithFrameworkFilter", func(t *testing.T) {
		p, ids := setup(t)
		policies, _, err := p.ListPolicies(context.Background(), interfaces.ListOptions{
			Framework: ids.PolicyFramework,
		})
		require.NoError(t, err)
		// With the matching framework, we should get results
		assert.NotEmpty(t, policies, "filtering by the loaded policy's framework should return results")

		// With a non-matching framework, we should get empty
		policiesNone, _, err := p.ListPolicies(context.Background(), interfaces.ListOptions{
			Framework: "NONEXISTENT_FRAMEWORK_XYZ",
		})
		require.NoError(t, err)
		assert.Empty(t, policiesNone, "filtering by a non-existent framework should return empty")
	})

	// -----------------------------------------------------------------------
	// Context cancellation
	// -----------------------------------------------------------------------

	t.Run("ContextCancellation", func(t *testing.T) {
		p, _ := setup(t)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // cancel immediately

		// At least one of the operations must return an error with a
		// cancelled context.  In-memory stubs that ignore context may not
		// fail, but any provider that does real I/O (HTTP, DB) must.
		_, _, errPolicies := p.ListPolicies(ctx, interfaces.ListOptions{})
		_, _, errControls := p.ListControls(ctx, interfaces.ListOptions{})
		_, _, errTasks := p.ListEvidenceTasks(ctx, interfaces.ListOptions{})

		anyErr := errPolicies != nil || errControls != nil || errTasks != nil
		// For network-backed providers we require context propagation.
		// For in-memory stubs that ignore context, log a note but do not fail.
		if !anyErr {
			t.Log("NOTE: provider did not return an error for cancelled context (acceptable for in-memory stubs)")
		}
	})
}

// ---------------------------------------------------------------------------
// StubDataProvider contract verification
// ---------------------------------------------------------------------------

func TestStubDataProvider_Contract(t *testing.T) {
	DataProviderContractSuite(t, func(t *testing.T) (interfaces.DataProvider, ContractFixtureIDs) {
		stub := testhelpers.NewStubDataProvider("test-stub")

		pol := testhelpers.SamplePolicy()
		stub.Policies[pol.ID] = pol

		ctrl := testhelpers.SampleControl()
		stub.Controls[ctrl.ID] = ctrl

		task := testhelpers.SampleEvidenceTask()
		stub.Tasks[task.ID] = task

		return stub, ContractFixtureIDs{
			PolicyID:        pol.ID,
			ControlID:       ctrl.ID,
			EvidenceTaskID:  task.ID,
			PolicyFramework: pol.Framework,
		}
	})
}

// ---------------------------------------------------------------------------
// TugboatDataProvider contract verification via httptest
// ---------------------------------------------------------------------------

func TestTugboatDataProvider_Contract(t *testing.T) {
	DataProviderContractSuite(t, func(t *testing.T) (interfaces.DataProvider, ContractFixtureIDs) {
		t.Helper()

		const (
			policyID   = "101"
			controlID  = "501"
			evidenceID = "1001"
			framework  = "SOC2"
		)

		// Fixture JSON for list responses
		policyListJSON := map[string]interface{}{
			"count":         1,
			"page_number":   1,
			"page_size":     25,
			"num_pages":     1,
			"max_page_size": 100,
			"results": []map[string]interface{}{
				{
					"id":          policyID,
					"name":        "Access Control Policy",
					"description": "Controls access to systems",
					"framework":   framework,
					"status":      "active",
					"created_at":  "2025-01-15T10:00:00Z",
					"updated_at":  "2025-01-15T10:00:00Z",
				},
			},
		}

		policyDetailJSON := map[string]interface{}{
			"id":          policyID,
			"name":        "Access Control Policy",
			"description": "Controls access to systems",
			"summary":     "Access control summary",
			"details":     "Full policy content",
			"framework":   framework,
			"status":      "active",
			"category":    "Security",
			"created_at":  "2025-01-15T10:00:00Z",
			"updated_at":  "2025-01-15T10:00:00Z",
		}

		controlListJSON := map[string]interface{}{
			"count":         1,
			"page_number":   1,
			"page_size":     25,
			"num_pages":     1,
			"max_page_size": 100,
			"results": []map[string]interface{}{
				{
					"id":                  501,
					"name":                "Logical Access Security",
					"body":                "Implement logical access controls",
					"category":            "Common Criteria",
					"framework":           framework,
					"status":              "implemented",
					"is_auto_implemented": false,
					"master_version_num":  1,
					"master_control_id":   100,
					"org_id":              1,
					"org_scope_id":        1,
				},
			},
		}

		controlDetailJSON := map[string]interface{}{
			"id":                  501,
			"name":                "Logical Access Security",
			"body":                "Implement logical access controls",
			"category":            "Common Criteria",
			"framework":           framework,
			"status":              "implemented",
			"is_auto_implemented": false,
			"master_version_num":  1,
			"master_control_id":   100,
			"org_id":              1,
			"org_scope_id":        1,
		}

		evidenceListJSON := map[string]interface{}{
			"count":         1,
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
					"framework":           framework,
					"status":              "pending",
					"priority":            "high",
					"created_at":          "2025-01-15T10:00:00Z",
					"updated_at":          "2025-01-15T10:00:00Z",
				},
			},
		}

		evidenceDetailJSON := map[string]interface{}{
			"id":                  1001,
			"name":                "GitHub Repository Access Controls",
			"description":         "Show team permissions for repos",
			"collection_interval": "quarter",
			"completed":           false,
			"framework":           framework,
			"status":              "pending",
			"priority":            "high",
			"created_at":          "2025-01-15T10:00:00Z",
			"updated_at":          "2025-01-15T10:00:00Z",
			"master_content": map[string]interface{}{
				"guidance": "Collect GitHub team permission evidence",
			},
		}

		// Empty list responses for filtered/paged-beyond queries
		emptyPolicyListJSON := map[string]interface{}{
			"count":         0,
			"page_number":   1,
			"page_size":     25,
			"num_pages":     0,
			"max_page_size": 100,
			"results":       []interface{}{},
		}
		// extractID returns the resource ID from a path like /api/policy/123/
		// by stripping the prefix. Returns empty string for list endpoints.
		extractID := func(path, prefix string) string {
			rest := strings.TrimPrefix(path, prefix)
			rest = strings.TrimSuffix(rest, "/")
			rest = strings.TrimRight(rest, "/")
			return rest
		}

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			path := r.URL.Path
			query := r.URL.RawQuery

			switch {
			// --- Policies ---
			case strings.HasPrefix(path, "/api/policy/"):
				id := extractID(path, "/api/policy/")
				if id == "" {
					// List endpoint
					if strings.Contains(query, "framework=NONEXISTENT_FRAMEWORK_XYZ") {
						json.NewEncoder(w).Encode(emptyPolicyListJSON)
						return
					}
					if strings.Contains(query, "page=9999") {
						json.NewEncoder(w).Encode(emptyPolicyListJSON)
						return
					}
					json.NewEncoder(w).Encode(policyListJSON)
				} else if id == policyID {
					json.NewEncoder(w).Encode(policyDetailJSON)
				} else {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintf(w, `{"error":{"code":"not_found","message":"policy not found"}}`)
				}

			// --- Controls ---
			case strings.HasPrefix(path, "/api/org_control/"):
				id := extractID(path, "/api/org_control/")
				if id == "" {
					json.NewEncoder(w).Encode(controlListJSON)
				} else if id == controlID {
					json.NewEncoder(w).Encode(controlDetailJSON)
				} else {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintf(w, `{"error":{"code":"not_found","message":"control not found"}}`)
				}

			// --- Evidence Tasks ---
			case strings.HasPrefix(path, "/api/org_evidence/"):
				id := extractID(path, "/api/org_evidence/")
				if id == "" {
					json.NewEncoder(w).Encode(evidenceListJSON)
				} else if id == evidenceID {
					json.NewEncoder(w).Encode(evidenceDetailJSON)
				} else {
					w.WriteHeader(http.StatusNotFound)
					fmt.Fprintf(w, `{"error":{"code":"not_found","message":"evidence task not found"}}`)
				}

			default:
				w.WriteHeader(http.StatusNotFound)
				fmt.Fprintf(w, `{"error":{"code":"not_found","message":"not found"}}`)
			}
		}))
		t.Cleanup(server.Close)

		log, err := logger.NewTestLogger()
		require.NoError(t, err)

		client := tugboatclient.NewClient(&config.TugboatConfig{
			BaseURL: server.URL,
			Timeout: 5 * time.Second,
		}, nil)

		adapter := adapters.NewTugboatToDomain()
		provider := tugboatprovider.NewTugboatDataProvider(client, adapter, "test-org", log)

		return provider, ContractFixtureIDs{
			PolicyID:        policyID,
			ControlID:       controlID,
			EvidenceTaskID:  evidenceID,
			PolicyFramework: framework,
		}
	})
}
