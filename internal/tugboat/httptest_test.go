// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tugboat

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/grctool/grctool/internal/tugboat/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// newTestClient creates a Client pointing at the given test server.
func newTestClient(t *testing.T, serverURL string) *Client {
	t.Helper()
	return &Client{
		baseURL:    serverURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		logger:     testhelpers.NewStubLogger(),
	}
}

// newTestClientWithCreds creates a Client with evidence submission credentials.
func newTestClientWithCreds(t *testing.T, serverURL string) *Client {
	t.Helper()
	return &Client{
		baseURL:    serverURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		logger:     testhelpers.NewStubLogger(),
		username:   "testuser",
		password:   "testpass",
		apiKey:     "test-api-key",
	}
}

// ---------------------------------------------------------------------------
// NewClient tests
// ---------------------------------------------------------------------------

func TestNewClient_Basic(t *testing.T) {
	// Cannot use t.Parallel() with t.Setenv
	origKey := os.Getenv("TUGBOAT_API_KEY")
	os.Setenv("TUGBOAT_API_KEY", "test-key")
	defer func() {
		if origKey == "" {
			os.Unsetenv("TUGBOAT_API_KEY")
		} else {
			os.Setenv("TUGBOAT_API_KEY", origKey)
		}
	}()

	cfg := &config.TugboatConfig{
		BaseURL:      "https://api-my.tugboatlogic.com",
		CookieHeader: "token=abc",
		Timeout:      10 * time.Second,
		RateLimit:    5,
	}

	client := NewClient(cfg, nil)
	require.NotNil(t, client)
	assert.Equal(t, "https://api-my.tugboatlogic.com", client.baseURL)
	assert.Equal(t, "token=abc", client.cookieHeader)
	assert.NotNil(t, client.httpClient)
	assert.NotNil(t, client.rateLimiter)
	assert.Equal(t, "test-key", client.apiKey)

	client.Close()
}

func TestNewClient_NoRateLimit(t *testing.T) {
	// Cannot use t.Parallel() with env var manipulation
	origKey := os.Getenv("TUGBOAT_API_KEY")
	os.Setenv("TUGBOAT_API_KEY", "k")
	defer func() {
		if origKey == "" {
			os.Unsetenv("TUGBOAT_API_KEY")
		} else {
			os.Setenv("TUGBOAT_API_KEY", origKey)
		}
	}()

	cfg := &config.TugboatConfig{
		BaseURL: "https://example.com",
		Timeout: 5 * time.Second,
	}

	client := NewClient(cfg, nil)
	require.NotNil(t, client)
	assert.Nil(t, client.rateLimiter)
	client.Close()
}

func TestClient_Close_NilRateLimiter(t *testing.T) {
	t.Parallel()

	// Ensure Close doesn't panic with nil rateLimiter
	c := &Client{logger: testhelpers.NewStubLogger()}
	c.Close() // should not panic
}

// ---------------------------------------------------------------------------
// extractBearerToken tests
// ---------------------------------------------------------------------------

func TestExtractBearerToken(t *testing.T) {
	t.Parallel()

	t.Run("base64 encoded JWT cookie", func(t *testing.T) {
		t.Parallel()
		tokenJSON := `{"access_token":"my-jwt-token-123"}`
		encoded := base64.StdEncoding.EncodeToString([]byte(tokenJSON))

		c := &Client{cookieHeader: "token=" + encoded}
		token, err := c.extractBearerToken()
		require.NoError(t, err)
		assert.Equal(t, "my-jwt-token-123", token)
	})

	t.Run("URL-encoded cookie", func(t *testing.T) {
		t.Parallel()
		tokenJSON := `{"access_token":"url-encoded-token"}`
		// URL-encode the JSON (simulate browser encoding)
		encoded := strings.ReplaceAll(tokenJSON, `"`, "%22")
		encoded = strings.ReplaceAll(encoded, `{`, "%7B")
		encoded = strings.ReplaceAll(encoded, `}`, "%7D")
		encoded = strings.ReplaceAll(encoded, `:`, "%3A")

		c := &Client{cookieHeader: "token=" + encoded}
		token, err := c.extractBearerToken()
		require.NoError(t, err)
		assert.Equal(t, "url-encoded-token", token)
	})

	t.Run("empty cookie header", func(t *testing.T) {
		t.Parallel()
		c := &Client{cookieHeader: ""}
		_, err := c.extractBearerToken()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no cookie header")
	})

	t.Run("no token cookie", func(t *testing.T) {
		t.Parallel()
		c := &Client{cookieHeader: "session=abc; other=xyz"}
		_, err := c.extractBearerToken()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token cookie not found")
	})

	t.Run("invalid base64 and invalid URL encoding", func(t *testing.T) {
		t.Parallel()
		c := &Client{cookieHeader: "token=not-valid-anything%%%"}
		_, err := c.extractBearerToken()
		assert.Error(t, err)
	})

	t.Run("valid base64 but no access_token field", func(t *testing.T) {
		t.Parallel()
		noTokenJSON := `{"refresh_token":"only-refresh"}`
		encoded := base64.StdEncoding.EncodeToString([]byte(noTokenJSON))
		c := &Client{cookieHeader: "token=" + encoded}
		_, err := c.extractBearerToken()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "access_token not found")
	})

	t.Run("multiple cookies with token present", func(t *testing.T) {
		t.Parallel()
		tokenJSON := `{"access_token":"found-it"}`
		encoded := base64.StdEncoding.EncodeToString([]byte(tokenJSON))
		c := &Client{cookieHeader: "session=xyz; token=" + encoded + "; other=abc"}
		token, err := c.extractBearerToken()
		require.NoError(t, err)
		assert.Equal(t, "found-it", token)
	})
}

// ---------------------------------------------------------------------------
// makeRequest tests
// ---------------------------------------------------------------------------

func TestMakeRequest_SetsHeaders(t *testing.T) {
	t.Parallel()

	var captured http.Header
	var capturedMethod string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = r.Header.Clone()
		capturedMethod = r.Method
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	resp, err := c.makeRequest(context.Background(), "GET", "/api/test", nil)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "GET", capturedMethod)
	assert.Equal(t, "application/json", captured.Get("Content-Type"))
	assert.Equal(t, "application/json", captured.Get("Accept"))
	assert.Equal(t, "grctool/1.0.0", captured.Get("User-Agent"))
}

func TestMakeRequest_WithBody(t *testing.T) {
	t.Parallel()

	var receivedBody map[string]string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		data, _ := io.ReadAll(r.Body)
		json.Unmarshal(data, &receivedBody)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	body := map[string]string{"name": "Test Policy"}
	resp, err := c.makeRequest(context.Background(), "POST", "/api/test", body)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, "Test Policy", receivedBody["name"])
}

func TestMakeRequest_ContextCancelled(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(5 * time.Second)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	_, err := c.makeRequest(ctx, "GET", "/api/test", nil)
	assert.Error(t, err)
}

func TestMakeRequest_WithRateLimiter(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	c.rateLimiter = time.NewTicker(50 * time.Millisecond)
	defer c.rateLimiter.Stop()

	start := time.Now()
	for i := 0; i < 3; i++ {
		resp, err := c.makeRequest(context.Background(), "GET", "/test", nil)
		require.NoError(t, err)
		resp.Body.Close()
	}
	elapsed := time.Since(start)
	// 3 requests at 50ms interval = at least ~100ms
	assert.True(t, elapsed >= 80*time.Millisecond, "Rate limiting should slow requests, got %v", elapsed)
}

func TestMakeRequest_CookieAuth(t *testing.T) {
	t.Parallel()

	var authHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authHeader = r.Header.Get("Authorization")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	// Use a valid base64-encoded token
	tokenJSON := `{"access_token":"my-token"}`
	encoded := base64.StdEncoding.EncodeToString([]byte(tokenJSON))

	c := newTestClient(t, server.URL)
	c.cookieHeader = "token=" + encoded

	resp, err := c.makeRequest(context.Background(), "GET", "/api/test", nil)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "Bearer my-token", authHeader)
}

func TestMakeRequest_FallbackCookieAuth(t *testing.T) {
	t.Parallel()

	var cookieHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookieHeader = r.Header.Get("Cookie")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	c.cookieHeader = "session=plain-cookie"

	resp, err := c.makeRequest(context.Background(), "GET", "/api/test", nil)
	require.NoError(t, err)
	resp.Body.Close()

	assert.Equal(t, "session=plain-cookie", cookieHeader)
}

// ---------------------------------------------------------------------------
// handleResponse tests
// ---------------------------------------------------------------------------

func TestHandleResponse_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"name": "Test"})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	var result map[string]string
	resp, err := c.makeRequest(context.Background(), "GET", "/test", nil)
	require.NoError(t, err)

	err = c.handleResponse(resp, &result)
	require.NoError(t, err)
	assert.Equal(t, "Test", result["name"])
}

func TestHandleResponse_HTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`{"error": {"code": "NOT_FOUND", "message": "Resource not found"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	resp, err := c.makeRequest(context.Background(), "GET", "/test", nil)
	require.NoError(t, err)

	var result map[string]interface{}
	err = c.handleResponse(resp, &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NOT_FOUND")
}

func TestHandleResponse_HTTPErrorPlainText(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`Internal Server Error`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	resp, err := c.makeRequest(context.Background(), "GET", "/test", nil)
	require.NoError(t, err)

	var result map[string]interface{}
	err = c.handleResponse(resp, &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 500")
}

func TestHandleResponse_WrappedAPIResponse(t *testing.T) {
	t.Parallel()

	// Some endpoints wrap data in {"data": ...}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"data": {"name": "Wrapped"}}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	resp, err := c.makeRequest(context.Background(), "GET", "/test", nil)
	require.NoError(t, err)

	var result map[string]string
	err = c.handleResponse(resp, &result)
	require.NoError(t, err)
	// The handler first tries direct unmarshal which would put "data" as key
	// If the shape doesn't match, it tries wrapped response
	assert.NotEmpty(t, result)
}

func TestHandleResponse_MalformedJSON(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{not valid json`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	resp, err := c.makeRequest(context.Background(), "GET", "/test", nil)
	require.NoError(t, err)

	var result map[string]interface{}
	err = c.handleResponse(resp, &result)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse response")
}

func TestHandleResponse_NilResult(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	resp, err := c.makeRequest(context.Background(), "DELETE", "/test", nil)
	require.NoError(t, err)

	err = c.handleResponse(resp, nil)
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// get/post/put/patch/delete helper tests
// ---------------------------------------------------------------------------

func TestClient_GetHelper(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	var result map[string]string
	err := c.get(context.Background(), "/test", &result)
	require.NoError(t, err)
	assert.Equal(t, "ok", result["status"])
}

func TestClient_PostHelper(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(map[string]int{"id": 1})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	var result map[string]int
	err := c.post(context.Background(), "/test", map[string]string{"name": "new"}, &result)
	require.NoError(t, err)
	assert.Equal(t, 1, result["id"])
}

func TestClient_PutHelper(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PUT", r.Method)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"updated": "true"})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	var result map[string]string
	err := c.put(context.Background(), "/test", nil, &result)
	require.NoError(t, err)
}

func TestClient_PatchHelper(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "PATCH", r.Method)
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"patched": "true"})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	var result map[string]string
	err := c.patch(context.Background(), "/test", nil, &result)
	require.NoError(t, err)
}

func TestClient_DeleteHelper(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "DELETE", r.Method)
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	err := c.delete(context.Background(), "/test")
	require.NoError(t, err)
}

// ---------------------------------------------------------------------------
// TestConnection tests
// ---------------------------------------------------------------------------

func TestTestConnection_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/org_evidence/")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"results":[]}`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	err := c.TestConnection(context.Background())
	require.NoError(t, err)
}

func TestTestConnection_AuthFailure(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		statusCode int
		wantErr    string
	}{
		{"401 unauthorized", http.StatusUnauthorized, "authentication failed"},
		{"403 forbidden", http.StatusForbidden, "access denied"},
		{"500 server error", http.StatusInternalServerError, "failed with status: 500"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.statusCode)
			}))
			defer server.Close()

			c := newTestClient(t, server.URL)
			err := c.TestConnection(context.Background())
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}

// ---------------------------------------------------------------------------
// makeEvidenceRequest tests
// ---------------------------------------------------------------------------

func TestMakeEvidenceRequest_SetsAuthHeaders(t *testing.T) {
	t.Parallel()

	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedHeaders = r.Header.Clone()
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c := newTestClientWithCreds(t, server.URL)

	resp, err := c.makeEvidenceRequest(context.Background(), "POST", server.URL+"/collector/123/", nil, "multipart/form-data")
	require.NoError(t, err)
	resp.Body.Close()

	// Check Basic Auth
	auth := receivedHeaders.Get("Authorization")
	assert.True(t, strings.HasPrefix(auth, "Basic "), "Expected Basic auth header")

	// Check API key
	assert.Equal(t, "test-api-key", receivedHeaders.Get("X-API-KEY"))

	// Check content type
	assert.Equal(t, "multipart/form-data", receivedHeaders.Get("Content-Type"))
}

func TestMakeEvidenceRequest_MissingCredentials(t *testing.T) {
	t.Parallel()

	t.Run("missing username", func(t *testing.T) {
		t.Parallel()
		c := &Client{
			httpClient: &http.Client{},
			logger:     testhelpers.NewStubLogger(),
			password:   "pass",
			apiKey:     "key",
		}
		_, err := c.makeEvidenceRequest(context.Background(), "POST", "http://example.com", nil, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "credentials not configured")
	})

	t.Run("missing password", func(t *testing.T) {
		t.Parallel()
		c := &Client{
			httpClient: &http.Client{},
			logger:     testhelpers.NewStubLogger(),
			username:   "user",
			apiKey:     "key",
		}
		_, err := c.makeEvidenceRequest(context.Background(), "POST", "http://example.com", nil, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "credentials not configured")
	})

	t.Run("missing API key", func(t *testing.T) {
		t.Parallel()
		c := &Client{
			httpClient: &http.Client{},
			logger:     testhelpers.NewStubLogger(),
			username:   "user",
			password:   "pass",
		}
		_, err := c.makeEvidenceRequest(context.Background(), "POST", "http://example.com", nil, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "TUGBOAT_API_KEY")
	})
}

// ---------------------------------------------------------------------------
// Policy endpoint tests
// ---------------------------------------------------------------------------

func TestGetPolicies(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/policy/")
		assert.Contains(t, r.URL.RawQuery, "page=1")
		assert.Contains(t, r.URL.RawQuery, "page_size=10")
		assert.Contains(t, r.URL.RawQuery, "org=13888")

		resp := PolicyListResponse{
			MaxPageSize: 200,
			PageSize:    10,
			NumPages:    2,
			PageNumber:  1,
			Count:       15,
			Results: []models.Policy{
				{
					ID:   models.IntOrString("94641"),
					Name: "Access Control Policy",
				},
				{
					ID:   models.IntOrString("94642"),
					Name: "Data Protection Policy",
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	policies, err := c.GetPolicies(context.Background(), &PolicyListOptions{
		Org:      "13888",
		Page:     1,
		PageSize: 10,
	})
	require.NoError(t, err)
	require.Len(t, policies, 2)
	assert.Equal(t, "Access Control Policy", policies[0].Name)
	assert.Equal(t, "94641", policies[0].ID.String())
}

func TestGetPolicies_WithFilters(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "framework=SOC2")
		assert.Contains(t, r.URL.RawQuery, "status=active")
		assert.Contains(t, r.URL.RawQuery, "embeds=tags")

		json.NewEncoder(w).Encode(PolicyListResponse{Results: []models.Policy{}})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	policies, err := c.GetPolicies(context.Background(), &PolicyListOptions{
		Org:       "13888",
		Framework: "SOC2",
		Status:    "active",
	})
	require.NoError(t, err)
	assert.Empty(t, policies)
}

func TestGetPolicies_NilOpts(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// With nil opts, endpoint should be plain /api/policy/
		assert.Equal(t, "/api/policy/", r.URL.Path)
		assert.Empty(t, r.URL.RawQuery)
		json.NewEncoder(w).Encode(PolicyListResponse{Results: []models.Policy{}})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	policies, err := c.GetPolicies(context.Background(), nil)
	require.NoError(t, err)
	assert.Empty(t, policies)
}

func TestGetAllPolicies_Pagination(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		var resp PolicyListResponse
		if callCount == 1 {
			// First page: return 25 items (full page)
			policies := make([]models.Policy, 25)
			for i := range policies {
				policies[i] = models.Policy{
					ID:   models.IntOrString(fmt.Sprintf("%d", i+1)),
					Name: fmt.Sprintf("Policy %d", i+1),
				}
			}
			resp.Results = policies
		} else {
			// Second page: return 5 items (partial page = last page)
			policies := make([]models.Policy, 5)
			for i := range policies {
				policies[i] = models.Policy{
					ID:   models.IntOrString(fmt.Sprintf("%d", 25+i+1)),
					Name: fmt.Sprintf("Policy %d", 25+i+1),
				}
			}
			resp.Results = policies
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)

	policies, err := c.GetAllPolicies(context.Background(), "13888", "")
	require.NoError(t, err)
	assert.Len(t, policies, 30)
	assert.Equal(t, 2, callCount)
}

func TestGetAllPolicies_EmptyResult(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(PolicyListResponse{Results: []models.Policy{}})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	policies, err := c.GetAllPolicies(context.Background(), "13888", "")
	require.NoError(t, err)
	assert.Empty(t, policies)
}

func TestGetPolicyDetails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/policy/94641/")
		assert.Contains(t, r.URL.RawQuery, "embeds=current_version")

		details := models.PolicyDetails{
			Policy: models.Policy{
				ID:   models.IntOrString("94641"),
				Name: "Access Control Policy",
			},
			Summary:  "A comprehensive policy",
			Category: "Access Control",
		}
		json.NewEncoder(w).Encode(details)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	details, err := c.GetPolicyDetails(context.Background(), "94641")
	require.NoError(t, err)
	assert.Equal(t, "94641", details.ID.String())
	assert.Equal(t, "A comprehensive policy", details.Summary)
}

// ---------------------------------------------------------------------------
// Control endpoint tests
// ---------------------------------------------------------------------------

func TestGetControls(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/org_control/")
		assert.Contains(t, r.URL.RawQuery, "embeds=framework_codes")
		assert.Contains(t, r.URL.RawQuery, "org=13888")

		resp := ControlListResponse{
			Count: 2,
			Results: []models.Control{
				{ID: 778805, Name: "Logical Access Security", Status: "implemented"},
				{ID: 778806, Name: "Physical Access", Status: "implemented"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	controls, err := c.GetControls(context.Background(), &ControlListOptions{
		Org:      "13888",
		Page:     1,
		PageSize: 25,
	})
	require.NoError(t, err)
	require.Len(t, controls, 2)
	assert.Equal(t, 778805, controls[0].ID)
}

func TestGetAllControls_Pagination(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			controls := make([]models.Control, 25)
			for i := range controls {
				controls[i] = models.Control{ID: i + 1, Name: fmt.Sprintf("Control %d", i+1)}
			}
			json.NewEncoder(w).Encode(ControlListResponse{Results: controls})
		} else {
			controls := make([]models.Control, 3)
			for i := range controls {
				controls[i] = models.Control{ID: 25 + i + 1, Name: fmt.Sprintf("Control %d", 25+i+1)}
			}
			json.NewEncoder(w).Encode(ControlListResponse{Results: controls})
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	controls, err := c.GetAllControls(context.Background(), "13888", "")
	require.NoError(t, err)
	assert.Len(t, controls, 28)
}

func TestGetControlDetails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/org_control/778805/")

		details := models.ControlDetails{
			Control: models.Control{
				ID: 778805,
				Name: "Logical Access Security",
			},
		}
		json.NewEncoder(w).Encode(details)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	details, err := c.GetControlDetails(context.Background(), "778805")
	require.NoError(t, err)
	assert.Equal(t, 778805, details.ID)
}

func TestGetControlDetails_ArrayResponse(t *testing.T) {
	t.Parallel()

	// Some APIs return single objects wrapped in an array
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		details := []models.ControlDetails{
			{Control: models.Control{ID: 778805, Name: "Control 1"}},
		}
		json.NewEncoder(w).Encode(details)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	details, err := c.GetControlDetails(context.Background(), "778805")
	require.NoError(t, err)
	assert.Equal(t, 778805, details.ID)
}

func TestGetControlDetails_EmptyArray(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]models.ControlDetails{})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetControlDetails(context.Background(), "999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty response")
}

func TestGetControlDetails_HTTPError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`Not Found`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetControlDetails(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "HTTP 404")
}

func TestGetControlDetailsWithEvidenceEmbeds(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "embeds=evidence_tasks")
		details := models.ControlDetails{
			Control: models.Control{ID: 778805, Name: "Test Control"},
		}
		json.NewEncoder(w).Encode(details)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	details, err := c.GetControlDetailsWithEvidenceEmbeds(context.Background(), "778805")
	require.NoError(t, err)
	assert.Equal(t, 778805, details.ID)
}

// ---------------------------------------------------------------------------
// Evidence task endpoint tests
// ---------------------------------------------------------------------------

func TestGetEvidenceTasks(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/org_evidence/")
		assert.Contains(t, r.URL.RawQuery, "org=13888")

		resp := EvidenceTaskListResponse{
			Count: 2,
			Results: []models.EvidenceTask{
				{ID: 327992, Name: "GitHub Access Controls"},
				{ID: 327993, Name: "Terraform Security"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	tasks, err := c.GetEvidenceTasks(context.Background(), &EvidenceTaskListOptions{
		Org:      "13888",
		Page:     1,
		PageSize: 25,
	})
	require.NoError(t, err)
	require.Len(t, tasks, 2)
	assert.Equal(t, 327992, tasks[0].ID)
}

func TestGetEvidenceTasks_WithEmbeds(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "embeds=assignees")
		assert.Contains(t, r.URL.RawQuery, "embeds=tags")
		json.NewEncoder(w).Encode(EvidenceTaskListResponse{Results: []models.EvidenceTask{}})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetEvidenceTasks(context.Background(), &EvidenceTaskListOptions{
		Org:    "13888",
		Embeds: []string{"assignees", "tags"},
	})
	require.NoError(t, err)
}

func TestGetAllEvidenceTasks_Pagination(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			tasks := make([]models.EvidenceTask, 25)
			for i := range tasks {
				tasks[i] = models.EvidenceTask{ID: i + 1}
			}
			json.NewEncoder(w).Encode(EvidenceTaskListResponse{Results: tasks})
		} else {
			json.NewEncoder(w).Encode(EvidenceTaskListResponse{Results: []models.EvidenceTask{{ID: 26}}})
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	tasks, err := c.GetAllEvidenceTasks(context.Background(), "13888", "")
	require.NoError(t, err)
	assert.Len(t, tasks, 26)
}

func TestGetEvidenceTaskDetails(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/org_evidence/327992/")
		assert.Contains(t, r.URL.RawQuery, "embeds=master_content")
		assert.Contains(t, r.URL.RawQuery, "embeds=org_controls")

		details := models.EvidenceTaskDetails{
			EvidenceTask: models.EvidenceTask{
				ID: 327992,
				Name: "GitHub Repository Access Controls",
			},
			MasterContent: &models.MasterContent{
				Guidance: "Use GitHub API",
			},
			EntityType: "org_evidence",
		}
		json.NewEncoder(w).Encode(details)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	details, err := c.GetEvidenceTaskDetails(context.Background(), "327992")
	require.NoError(t, err)
	assert.Equal(t, 327992, details.ID)
	assert.Equal(t, "Use GitHub API", details.MasterContent.Guidance)
}

func TestGetEvidenceTasksByControl(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "control_id=778805")
		assert.Contains(t, r.URL.RawQuery, "org=13888")

		json.NewEncoder(w).Encode(EvidenceTaskListResponse{
			Results: []models.EvidenceTask{
				{ID: 327992, Name: "GitHub Access Controls"},
			},
		})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	tasks, err := c.GetEvidenceTasksByControl(context.Background(), "778805", "13888")
	require.NoError(t, err)
	require.Len(t, tasks, 1)
	assert.Equal(t, 327992, tasks[0].ID)
}

// ---------------------------------------------------------------------------
// Submissions / EvidenceAttachments endpoint tests
// ---------------------------------------------------------------------------

func TestGetEvidenceAttachments(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/org_evidence_attachment/")
		assert.Contains(t, r.URL.RawQuery, "org_evidence=327992")
		assert.Contains(t, r.URL.RawQuery, "ordering=-collected,-created")

		resp := models.EvidenceAttachmentListResponse{
			Count: 1,
			Results: []models.EvidenceAttachment{
				{ID: 50001, Type: "file", OrgEvidenceID: 327992},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	attachments, err := c.GetEvidenceAttachments(context.Background(), &models.EvidenceAttachmentListOptions{
		OrgEvidence: 327992,
		Page:        1,
		PageSize:    25,
	})
	require.NoError(t, err)
	require.Len(t, attachments, 1)
	assert.Equal(t, 50001, attachments[0].ID)
}

func TestGetEvidenceAttachments_Defaults(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check defaults are applied
		assert.Contains(t, r.URL.RawQuery, "ordering=-collected,-created")
		assert.Contains(t, r.URL.RawQuery, "page=1")
		assert.Contains(t, r.URL.RawQuery, "page_size=25")
		assert.Contains(t, r.URL.RawQuery, "embeds=attachment")
		assert.Contains(t, r.URL.RawQuery, "embeds=org_members")

		json.NewEncoder(w).Encode(models.EvidenceAttachmentListResponse{Results: []models.EvidenceAttachment{}})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetEvidenceAttachments(context.Background(), &models.EvidenceAttachmentListOptions{
		OrgEvidence: 1,
	})
	require.NoError(t, err)
}

func TestGetEvidenceAttachments_WithDeletedFilter(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "deleted=false")
		json.NewEncoder(w).Encode(models.EvidenceAttachmentListResponse{Results: []models.EvidenceAttachment{}})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	falseVal := false
	_, err := c.GetEvidenceAttachments(context.Background(), &models.EvidenceAttachmentListOptions{
		OrgEvidence: 1,
		Deleted:     &falseVal,
	})
	require.NoError(t, err)
}

func TestGetAllEvidenceAttachments_Pagination(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			items := make([]models.EvidenceAttachment, 100)
			for i := range items {
				items[i] = models.EvidenceAttachment{ID: i + 1}
			}
			json.NewEncoder(w).Encode(models.EvidenceAttachmentListResponse{Results: items})
		} else {
			json.NewEncoder(w).Encode(models.EvidenceAttachmentListResponse{
				Results: []models.EvidenceAttachment{{ID: 101}},
			})
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	all, err := c.GetAllEvidenceAttachments(context.Background(), 327992, "2013-01-01,2025-12-31")
	require.NoError(t, err)
	assert.Len(t, all, 101)
}

func TestGetEvidenceAttachmentsByTask(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "observation_period=2013-01-01,2025-12-31")
		json.NewEncoder(w).Encode(models.EvidenceAttachmentListResponse{Results: []models.EvidenceAttachment{}})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetEvidenceAttachmentsByTask(context.Background(), 327992)
	require.NoError(t, err)
}

func TestGetEvidenceAttachmentsByTaskAndWindow(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "observation_period=2025-01-01,2025-03-31")
		json.NewEncoder(w).Encode(models.EvidenceAttachmentListResponse{Results: []models.EvidenceAttachment{}})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetEvidenceAttachmentsByTaskAndWindow(context.Background(), 327992, "2025-01-01", "2025-03-31")
	require.NoError(t, err)
}

func TestGetAttachmentDownloadURL(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/org_evidence_attachment/9001/download/")
		json.NewEncoder(w).Encode(models.AttachmentDownloadResponse{
			URL:              "https://s3.amazonaws.com/bucket/file.csv?token=abc",
			OriginalFilename: "evidence.csv",
		})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	resp, err := c.GetAttachmentDownloadURL(context.Background(), 9001)
	require.NoError(t, err)
	assert.Contains(t, resp.URL, "s3.amazonaws.com")
	assert.Equal(t, "evidence.csv", resp.OriginalFilename)
}

// ---------------------------------------------------------------------------
// SubmitEvidence tests (using httptest)
// ---------------------------------------------------------------------------

func TestSubmitEvidence_SuccessWithHTTPTest(t *testing.T) {
	t.Parallel()

	var receivedContentType string
	var receivedAuth string
	var receivedAPIKey string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		receivedAuth = r.Header.Get("Authorization")
		receivedAPIKey = r.Header.Get("X-API-KEY")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(SubmitEvidenceResponse{Success: true, Message: "Received"})
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "evidence.csv")
	require.NoError(t, os.WriteFile(testFile, []byte("col1,col2\nval1,val2"), 0644))

	c := newTestClientWithCreds(t, server.URL)

	resp, err := c.SubmitEvidence(context.Background(), &SubmitEvidenceRequest{
		CollectorURL:  server.URL + "/collector/123/",
		FilePath:      testFile,
		CollectedDate: time.Date(2025, 3, 15, 0, 0, 0, 0, time.UTC),
	})
	require.NoError(t, err)
	assert.True(t, resp.Success)
	assert.True(t, strings.HasPrefix(receivedContentType, "multipart/form-data"))
	assert.True(t, strings.HasPrefix(receivedAuth, "Basic "))
	assert.Equal(t, "test-api-key", receivedAPIKey)
}

func TestSubmitEvidence_MissingCollectorURL(t *testing.T) {
	t.Parallel()

	c := newTestClientWithCreds(t, "http://unused")
	_, err := c.SubmitEvidence(context.Background(), &SubmitEvidenceRequest{
		FilePath:      "/some/file.csv",
		CollectedDate: time.Now(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "collector URL is required")
}

func TestSubmitEvidence_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(`{"error": "bad request"}`))
	}))
	defer server.Close()

	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "evidence.csv")
	require.NoError(t, os.WriteFile(testFile, []byte("data"), 0644))

	c := newTestClientWithCreds(t, server.URL)

	_, err := c.SubmitEvidence(context.Background(), &SubmitEvidenceRequest{
		CollectorURL:  server.URL + "/collector/123/",
		FilePath:      testFile,
		CollectedDate: time.Now(),
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "submission failed with status 400")
}

// ---------------------------------------------------------------------------
// min helper
// ---------------------------------------------------------------------------

func TestMin(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 3, min(3, 5))
	assert.Equal(t, 3, min(5, 3))
	assert.Equal(t, 0, min(0, 0))
	assert.Equal(t, -1, min(-1, 0))
}

// ---------------------------------------------------------------------------
// Error scenarios across endpoints
// ---------------------------------------------------------------------------

func TestGetPolicies_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`Internal Server Error`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetPolicies(context.Background(), &PolicyListOptions{Org: "13888"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get policies")
}

func TestGetControls_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`error`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetControls(context.Background(), &ControlListOptions{Org: "13888"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get controls")
}

func TestGetEvidenceTasks_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`error`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetEvidenceTasks(context.Background(), &EvidenceTaskListOptions{Org: "13888"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get evidence tasks")
}

func TestGetAllPolicies_ErrorOnPage(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			policies := make([]models.Policy, 25)
			for i := range policies {
				policies[i] = models.Policy{ID: models.IntOrString(fmt.Sprintf("%d", i))}
			}
			json.NewEncoder(w).Encode(PolicyListResponse{Results: policies})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
			w.Write([]byte(`error`))
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetAllPolicies(context.Background(), "13888", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get policies page 2")
}

func TestGetAllControls_ErrorOnPage(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			controls := make([]models.Control, 25)
			json.NewEncoder(w).Encode(ControlListResponse{Results: controls})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetAllControls(context.Background(), "13888", "")
	assert.Error(t, err)
}

func TestGetAllEvidenceTasks_ErrorOnPage(t *testing.T) {
	t.Parallel()

	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		if callCount == 1 {
			tasks := make([]models.EvidenceTask, 25)
			json.NewEncoder(w).Encode(EvidenceTaskListResponse{Results: tasks})
		} else {
			w.WriteHeader(http.StatusInternalServerError)
		}
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetAllEvidenceTasks(context.Background(), "13888", "")
	assert.Error(t, err)
}

func TestGetEvidenceTaskDetails_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte(`not found`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetEvidenceTaskDetails(context.Background(), "999")
	assert.Error(t, err)
}

func TestGetControlDetailsWithEvidenceEmbeds_EmptyArray(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]models.ControlDetails{})
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetControlDetailsWithEvidenceEmbeds(context.Background(), "999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "empty response")
}

func TestGetControlDetailsWithEvidenceEmbeds_MalformedResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`not json at all`))
	}))
	defer server.Close()

	c := newTestClient(t, server.URL)
	_, err := c.GetControlDetailsWithEvidenceEmbeds(context.Background(), "999")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to parse")
}
