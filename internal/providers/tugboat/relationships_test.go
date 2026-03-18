// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tugboat

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/grctool/grctool/internal/adapters"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/config"
	tugboatclient "github.com/grctool/grctool/internal/tugboat"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTugboatDataProvider_ImplementsRelationshipQuerier(t *testing.T) {
	t.Parallel()
	var p interfaces.DataProvider = &TugboatDataProvider{}
	_, ok := p.(interfaces.RelationshipQuerier)
	assert.True(t, ok, "TugboatDataProvider must implement RelationshipQuerier")
}

func TestTugboatDataProvider_GetEvidenceTasksByControl(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the control filter is passed
		assert.Contains(t, r.URL.RawQuery, "control_id=778805")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{"id": 327992, "name": "GitHub Access Controls", "completed": false},
				{"id": 327993, "name": "GitHub Workflow Security", "completed": false},
			},
			"count":     2,
			"num_pages": 1,
		})
	}))
	defer server.Close()

	client := tugboatclient.NewClient(&config.TugboatConfig{BaseURL: server.URL}, nil)
	adapter := adapters.NewTugboatToDomain()
	log := testhelpers.NewStubLogger()
	provider := NewTugboatDataProvider(client, adapter, "13888", log)

	tasks, err := provider.GetEvidenceTasksByControl(context.Background(), "778805")
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
	assert.Equal(t, "GitHub Access Controls", tasks[0].Name)
}

func TestTugboatDataProvider_GetEvidenceTasksByControl_Empty(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results":   []interface{}{},
			"count":     0,
			"num_pages": 0,
		})
	}))
	defer server.Close()

	client := tugboatclient.NewClient(&config.TugboatConfig{BaseURL: server.URL}, nil)
	adapter := adapters.NewTugboatToDomain()
	provider := NewTugboatDataProvider(client, adapter, "13888", testhelpers.NewStubLogger())

	tasks, err := provider.GetEvidenceTasksByControl(context.Background(), "999999")
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

func TestTugboatDataProvider_GetEvidenceTasksByControl_Error(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := tugboatclient.NewClient(&config.TugboatConfig{BaseURL: server.URL}, nil)
	adapter := adapters.NewTugboatToDomain()
	provider := NewTugboatDataProvider(client, adapter, "13888", testhelpers.NewStubLogger())

	_, err := provider.GetEvidenceTasksByControl(context.Background(), "778805")
	assert.Error(t, err)
}

func TestTugboatDataProvider_GetControlsByPolicy_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	// Tugboat API doesn't support this query — returns empty with log message.
	provider := NewTugboatDataProvider(nil, nil, "13888", testhelpers.NewStubLogger())
	controls, err := provider.GetControlsByPolicy(context.Background(), "POL-001")
	require.NoError(t, err)
	assert.Empty(t, controls)
}

func TestTugboatDataProvider_GetPoliciesByControl_ReturnsEmpty(t *testing.T) {
	t.Parallel()
	// Same — not supported by Tugboat API.
	provider := NewTugboatDataProvider(nil, nil, "13888", testhelpers.NewStubLogger())
	policies, err := provider.GetPoliciesByControl(context.Background(), "CC-06.1")
	require.NoError(t, err)
	assert.Empty(t, policies)
}

func TestTugboatDataProvider_TypeAssertionPattern(t *testing.T) {
	t.Parallel()
	provider := NewTugboatDataProvider(nil, nil, "13888", testhelpers.NewStubLogger())

	// Cast to DataProvider, then type-assert RelationshipQuerier
	var dp interfaces.DataProvider = provider
	rq, ok := dp.(interfaces.RelationshipQuerier)
	assert.True(t, ok)
	assert.NotNil(t, rq)
}
