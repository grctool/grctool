// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package providers_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grctool/grctool/internal/adapters"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/testhelpers"
	tugboatclient "github.com/grctool/grctool/internal/tugboat"
	tugboatprovider "github.com/grctool/grctool/internal/providers/tugboat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// RelationshipQuerier Contract Test Suite
// ---------------------------------------------------------------------------

// RelationshipQuerierContractSuite runs standard contract tests against any
// provider that implements RelationshipQuerier. Call from each provider's
// test file to verify compliance.
//
// The setup function must return a DataProvider that also implements
// RelationshipQuerier, pre-loaded with:
//   - At least 1 control with ID knownControlID
//   - At least 1 evidence task linked to that control
//   - At least 1 policy with ID knownPolicyID
//   - At least 1 control linked to that policy
func RelationshipQuerierContractSuite(t *testing.T, knownControlID, knownPolicyID string, setup func(t *testing.T) interfaces.DataProvider) {
	t.Helper()

	t.Run("ImplementsRelationshipQuerier", func(t *testing.T) {
		p := setup(t)
		_, ok := p.(interfaces.RelationshipQuerier)
		assert.True(t, ok, "provider must implement RelationshipQuerier")
	})

	t.Run("GetEvidenceTasksByControl_Found", func(t *testing.T) {
		p := setup(t)
		rq := p.(interfaces.RelationshipQuerier)
		tasks, err := rq.GetEvidenceTasksByControl(context.Background(), knownControlID)
		require.NoError(t, err)
		assert.NotEmpty(t, tasks, "known control should have linked tasks")
	})

	t.Run("GetEvidenceTasksByControl_UnknownControl", func(t *testing.T) {
		p := setup(t)
		rq := p.(interfaces.RelationshipQuerier)
		tasks, err := rq.GetEvidenceTasksByControl(context.Background(), "NONEXISTENT-CTRL-99999")
		require.NoError(t, err)
		assert.Empty(t, tasks)
	})

	t.Run("GetControlsByPolicy_NoError", func(t *testing.T) {
		p := setup(t)
		rq := p.(interfaces.RelationshipQuerier)
		_, err := rq.GetControlsByPolicy(context.Background(), knownPolicyID)
		// Some providers may not support this — NoError is required, empty is OK
		require.NoError(t, err)
	})

	t.Run("GetPoliciesByControl_NoError", func(t *testing.T) {
		p := setup(t)
		rq := p.(interfaces.RelationshipQuerier)
		_, err := rq.GetPoliciesByControl(context.Background(), knownControlID)
		require.NoError(t, err)
	})

	t.Run("TypeAssertionNegative", func(t *testing.T) {
		// A plain StubDataProvider should NOT implement RelationshipQuerier
		plain := testhelpers.NewStubDataProvider("plain-provider")
		_, ok := interfaces.DataProvider(plain).(interfaces.RelationshipQuerier)
		assert.False(t, ok)
	})
}

// ---------------------------------------------------------------------------
// Run suite against StubFullProvider
// ---------------------------------------------------------------------------

func TestStubFullProvider_RelationshipContract(t *testing.T) {
	t.Parallel()
	RelationshipQuerierContractSuite(t, "CC-06.1", "POL-001", func(t *testing.T) interfaces.DataProvider {
		fp := testhelpers.NewStubFullProvider("stub-rel")

		pol := &domain.Policy{ID: "POL-001", Name: "Access Control Policy"}
		ctrl := &domain.Control{ID: "CC-06.1", Name: "Logical Access", ReferenceID: "CC-06.1"}
		task := &domain.EvidenceTask{ID: "ET-0047", Name: "GitHub Access Controls"}

		fp.Policies["POL-001"] = pol
		fp.Controls["CC-06.1"] = ctrl
		fp.Tasks["ET-0047"] = task
		fp.ControlToTasks["CC-06.1"] = []string{"ET-0047"}
		fp.PolicyToControls["POL-001"] = []string{"CC-06.1"}
		fp.ControlToPolicies["CC-06.1"] = []string{"POL-001"}

		return fp
	})
}

// ---------------------------------------------------------------------------
// Run suite against TugboatDataProvider (httptest-backed)
// ---------------------------------------------------------------------------

func TestTugboatDataProvider_RelationshipContract(t *testing.T) {
	t.Parallel()

	RelationshipQuerierContractSuite(t, "778805", "POL-001", func(t *testing.T) interfaces.DataProvider {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/api/org_evidence/" && r.URL.Query().Get("control_id") == "778805":
				json.NewEncoder(w).Encode(map[string]interface{}{
					"results": []map[string]interface{}{
						{"id": 327992, "name": "GitHub Access Controls", "completed": false},
					},
					"count": 1, "num_pages": 1,
				})
			default:
				// Policy/control relationship queries return empty (Tugboat limitation)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"results": []interface{}{}, "count": 0, "num_pages": 0,
				})
			}
		}))
		t.Cleanup(server.Close)

		client := tugboatclient.NewClient(&config.TugboatConfig{BaseURL: server.URL}, nil)
		adapter := adapters.NewTugboatToDomain()
		return tugboatprovider.NewTugboatDataProvider(client, adapter, "13888", testhelpers.NewStubLogger())
	})
}

// ---------------------------------------------------------------------------
// Additional contract validation
// ---------------------------------------------------------------------------

func TestRelationshipQuerier_StubFullProvider_Bidirectional(t *testing.T) {
	t.Parallel()
	fp := testhelpers.NewStubFullProvider("bidir")

	pol := &domain.Policy{ID: "POL-001", Name: "Access Policy"}
	ctrl := &domain.Control{ID: "CC-06.1", Name: "Logical Access"}
	task := &domain.EvidenceTask{ID: "ET-0047", Name: "Access Evidence"}

	fp.Policies["POL-001"] = pol
	fp.Controls["CC-06.1"] = ctrl
	fp.Tasks["ET-0047"] = task
	fp.PolicyToControls["POL-001"] = []string{"CC-06.1"}
	fp.ControlToPolicies["CC-06.1"] = []string{"POL-001"}
	fp.ControlToTasks["CC-06.1"] = []string{"ET-0047"}

	// Forward: policy → controls → tasks
	controls, err := fp.GetControlsByPolicy(context.Background(), "POL-001")
	require.NoError(t, err)
	require.Len(t, controls, 1)

	tasks, err := fp.GetEvidenceTasksByControl(context.Background(), controls[0].ID)
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, "ET-0047", tasks[0].ID)

	// Reverse: control → policies
	policies, err := fp.GetPoliciesByControl(context.Background(), "CC-06.1")
	require.NoError(t, err)
	require.Len(t, policies, 1)
	assert.Equal(t, "POL-001", policies[0].ID)
}
