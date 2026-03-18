// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package accountablehq

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewScraperClient(t *testing.T) {
	t.Parallel()
	client, err := NewScraperClient(ScraperConfig{
		BaseURL: "https://app.accountablehq.com",
	})
	require.NoError(t, err)
	assert.NotNil(t, client)
	assert.Equal(t, "https://app.accountablehq.com", client.baseURL)
	assert.False(t, client.IsLoggedIn())
}

func TestNewScraperClient_DefaultTimeout(t *testing.T) {
	t.Parallel()
	client, err := NewScraperClient(ScraperConfig{BaseURL: "https://example.com"})
	require.NoError(t, err)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
}

func TestNewScraperClient_CustomTimeout(t *testing.T) {
	t.Parallel()
	client, err := NewScraperClient(ScraperConfig{
		BaseURL: "https://example.com",
		Timeout: 10 * time.Second,
	})
	require.NoError(t, err)
	assert.Equal(t, 10*time.Second, client.httpClient.Timeout)
}

func TestScraperClient_Login_Success(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "/login", r.URL.Path)
		assert.Equal(t, "application/x-www-form-urlencoded", r.Header.Get("Content-Type"))

		r.ParseForm()
		assert.Equal(t, "testuser", r.FormValue("username"))
		assert.Equal(t, "testpass", r.FormValue("password"))

		// Set session cookie
		http.SetCookie(w, &http.Cookie{
			Name:  "session_id",
			Value: "abc123",
			Path:  "/",
		})
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client, err := NewScraperClient(ScraperConfig{BaseURL: server.URL})
	require.NoError(t, err)

	err = client.Login(context.Background(), "testuser", "testpass")
	require.NoError(t, err)
	assert.True(t, client.IsLoggedIn())
	assert.Equal(t, "testuser", client.session.Username)
}

func TestScraperClient_Login_Unauthorized(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client, err := NewScraperClient(ScraperConfig{BaseURL: server.URL})
	require.NoError(t, err)

	err = client.Login(context.Background(), "bad", "creds")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "login failed")
	assert.False(t, client.IsLoggedIn())
}

func TestScraperClient_Login_Forbidden(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client, err := NewScraperClient(ScraperConfig{BaseURL: server.URL})
	require.NoError(t, err)

	err = client.Login(context.Background(), "user", "pass")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "403")
}

func TestScraperClient_IsLoggedIn_Expired(t *testing.T) {
	t.Parallel()
	client, _ := NewScraperClient(ScraperConfig{BaseURL: "https://example.com"})
	client.session.LoggedIn = true
	client.session.ExpiresAt = time.Now().Add(-1 * time.Hour) // expired
	assert.False(t, client.IsLoggedIn())
}

func TestScraperClient_IsLoggedIn_Valid(t *testing.T) {
	t.Parallel()
	client, _ := NewScraperClient(ScraperConfig{BaseURL: "https://example.com"})
	client.session.LoggedIn = true
	client.session.ExpiresAt = time.Now().Add(1 * time.Hour) // valid
	assert.True(t, client.IsLoggedIn())
}

func TestScraperClient_DoGet_Authenticated(t *testing.T) {
	t.Parallel()

	var receivedCookies []*http.Cookie
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/login" {
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "tok123", Path: "/"})
			w.WriteHeader(http.StatusOK)
			return
		}
		receivedCookies = r.Cookies()
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`<html><body>policies</body></html>`))
	}))
	defer server.Close()

	client, _ := NewScraperClient(ScraperConfig{BaseURL: server.URL})

	// Login first
	err := client.Login(context.Background(), "user", "pass")
	require.NoError(t, err)

	// Authenticated request should carry cookies
	resp, err := client.doGet(context.Background(), "/policies")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	// Cookie jar should forward the session cookie
	found := false
	for _, c := range receivedCookies {
		if c.Name == "session" && c.Value == "tok123" {
			found = true
		}
	}
	assert.True(t, found, "session cookie should be forwarded")
}

func TestScraperClient_TrailingSlashStripped(t *testing.T) {
	t.Parallel()
	client, _ := NewScraperClient(ScraperConfig{BaseURL: "https://example.com/"})
	assert.Equal(t, "https://example.com", client.baseURL)
}
