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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestServer(handler http.HandlerFunc) (*httptest.Server, *HTTPClient) {
	server := httptest.NewServer(handler)
	client := NewHTTPClient(HTTPClientConfig{
		BaseURL: server.URL,
		APIKey:  "test-api-key",
		Timeout: 5 * time.Second,
	})
	return server, client
}

func TestHTTPClient_TestConnection_Success(t *testing.T) {
	t.Parallel()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Bearer test-api-key", r.Header.Get("Authorization"))
		assert.Equal(t, "application/json", r.Header.Get("Accept"))
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(apiResponse{Data: json.RawMessage("[]")})
	})
	defer server.Close()

	err := client.TestConnection(context.Background())
	assert.NoError(t, err)
}

func TestHTTPClient_TestConnection_Unauthorized(t *testing.T) {
	t.Parallel()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})
	defer server.Close()

	err := client.TestConnection(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "401")
}

func TestHTTPClient_TestConnection_Forbidden(t *testing.T) {
	t.Parallel()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	})
	defer server.Close()

	err := client.TestConnection(context.Background())
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestHTTPClient_ListPolicies(t *testing.T) {
	t.Parallel()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/policies", r.URL.Path)
		assert.Equal(t, "1", r.URL.Query().Get("page"))   // 0-indexed input → 1-indexed API
		assert.Equal(t, "25", r.URL.Query().Get("per_page"))
		json.NewEncoder(w).Encode(apiResponse{
			Data: json.RawMessage(`[{"id":"p1","title":"Policy One","version":2},{"id":"p2","title":"Policy Two","version":1}]`),
			Meta: &apiMeta{Page: 1, PerPage: 25, Total: 42},
		})
	})
	defer server.Close()

	policies, total, err := client.ListPolicies(context.Background(), 0, 0)
	require.NoError(t, err)
	assert.Equal(t, 42, total)
	assert.Len(t, policies, 2)
	assert.Equal(t, "p1", policies[0].ID)
	assert.Equal(t, "Policy One", policies[0].Title)
}

func TestHTTPClient_ListPolicies_Error(t *testing.T) {
	t.Parallel()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	})
	defer server.Close()

	_, _, err := client.ListPolicies(context.Background(), 0, 25)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

func TestHTTPClient_GetPolicy(t *testing.T) {
	t.Parallel()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/v1/policies/p1", r.URL.Path)
		// Return without envelope (raw body)
		json.NewEncoder(w).Encode(AHQPolicy{ID: "p1", Title: "Test Policy", Content: "# Test", Version: 5})
	})
	defer server.Close()

	policy, err := client.GetPolicy(context.Background(), "p1")
	require.NoError(t, err)
	assert.Equal(t, "p1", policy.ID)
	assert.Equal(t, "Test Policy", policy.Title)
	assert.Equal(t, 5, policy.Version)
}

func TestHTTPClient_GetPolicy_NotFound(t *testing.T) {
	t.Parallel()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	})
	defer server.Close()

	_, err := client.GetPolicy(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "404")
}

func TestHTTPClient_CreatePolicy(t *testing.T) {
	t.Parallel()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/api/v1/policies", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var received AHQPolicy
		json.NewDecoder(r.Body).Decode(&received)
		assert.Equal(t, "New Policy", received.Title)

		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"data": map[string]string{"id": "p-new-1"},
		})
	})
	defer server.Close()

	id, err := client.CreatePolicy(context.Background(), &AHQPolicy{Title: "New Policy", Content: "# New"})
	require.NoError(t, err)
	assert.Equal(t, "p-new-1", id)
}

func TestHTTPClient_CreatePolicy_Error(t *testing.T) {
	t.Parallel()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnprocessableEntity)
		w.Write([]byte("validation failed"))
	})
	defer server.Close()

	_, err := client.CreatePolicy(context.Background(), &AHQPolicy{Title: "Bad"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "422")
}

func TestHTTPClient_UpdatePolicy(t *testing.T) {
	t.Parallel()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		assert.Equal(t, "/api/v1/policies/p1", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	})
	defer server.Close()

	err := client.UpdatePolicy(context.Background(), "p1", &AHQPolicy{Title: "Updated"})
	assert.NoError(t, err)
}

func TestHTTPClient_UpdatePolicy_Error(t *testing.T) {
	t.Parallel()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		w.Write([]byte("forbidden"))
	})
	defer server.Close()

	err := client.UpdatePolicy(context.Background(), "p1", &AHQPolicy{Title: "Fail"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestHTTPClient_DeletePolicy(t *testing.T) {
	t.Parallel()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		assert.Equal(t, "/api/v1/policies/p1", r.URL.Path)
		w.WriteHeader(http.StatusNoContent)
	})
	defer server.Close()

	err := client.DeletePolicy(context.Background(), "p1")
	assert.NoError(t, err)
}

func TestHTTPClient_DeletePolicy_Error(t *testing.T) {
	t.Parallel()
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("not found"))
	})
	defer server.Close()

	err := client.DeletePolicy(context.Background(), "nonexistent")
	assert.Error(t, err)
}

func TestHTTPClient_ListPolicies_NoEnvelope(t *testing.T) {
	t.Parallel()
	// Some APIs return a flat array without the data/meta envelope.
	server, client := newTestServer(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]AHQPolicy{
			{ID: "p1", Title: "Flat Response"},
		})
	})
	defer server.Close()

	policies, total, err := client.ListPolicies(context.Background(), 0, 25)
	require.NoError(t, err)
	assert.Equal(t, 1, total) // no meta, falls back to len(result)
	assert.Len(t, policies, 1)
}

func TestHTTPClient_CompileTimeInterface(t *testing.T) {
	var _ AccountableHQClient = (*HTTPClient)(nil)
}
