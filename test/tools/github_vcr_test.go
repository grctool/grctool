// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/vcr"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestVCR_RecordAndPlayback verifies the VCR transport can record HTTP
// interactions to cassettes and replay them, with header sanitization.
func TestVCR_RecordAndPlayback(t *testing.T) {
	// Create a mock GitHub-like API server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/search/issues":
			json.NewEncoder(w).Encode(map[string]interface{}{
				"total_count": 1,
				"items": []map[string]interface{}{
					{"id": 1, "title": "Security audit finding", "state": "open"},
				},
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	cassetteDir := filepath.Join(t.TempDir(), "cassettes")
	require.NoError(t, os.MkdirAll(cassetteDir, 0755))

	// Phase 1: Record against the live mock server
	recordCfg := &vcr.Config{
		Enabled:         true,
		Mode:            vcr.ModeRecord,
		CassetteDir:     cassetteDir,
		SanitizeHeaders: true,
		RedactHeaders:   []string{"authorization"},
		MatchMethod:     true,
		MatchURI:        true,
	}
	recorder := vcr.New(recordCfg)
	recordClient := &http.Client{Transport: recorder}

	req, err := http.NewRequest("GET", server.URL+"/search/issues?q=security", nil)
	require.NoError(t, err)
	req.Header.Set("Authorization", "token ghp_secret123")

	resp, err := recordClient.Do(req)
	require.NoError(t, err)
	resp.Body.Close()
	assert.Equal(t, 200, resp.StatusCode)

	// Verify cassette was created
	cassettes, err := filepath.Glob(filepath.Join(cassetteDir, "*.json"))
	require.NoError(t, err)
	require.Greater(t, len(cassettes), 0, "VCR cassette should be created")

	// Verify Authorization header is redacted in cassette
	for _, cassettePath := range cassettes {
		data, err := os.ReadFile(cassettePath)
		require.NoError(t, err)
		assert.NotContains(t, string(data), "ghp_secret123",
			"Authorization token should be redacted in cassette")
	}

	// Phase 2: Playback from cassette (server no longer needed)
	playbackCfg := &vcr.Config{
		Enabled:     true,
		Mode:        vcr.ModePlayback,
		CassetteDir: cassetteDir,
		MatchMethod: true,
		MatchURI:    true,
	}
	player := vcr.New(playbackCfg)
	playbackClient := &http.Client{Transport: player}

	// Same request URL — VCR should match the recorded cassette
	req2, err := http.NewRequest("GET", server.URL+"/search/issues?q=security", nil)
	require.NoError(t, err)

	resp2, err := playbackClient.Do(req2)
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, 200, resp2.StatusCode)

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&result))
	assert.Equal(t, float64(1), result["total_count"])
}

// TestVCR_EnvironmentConfig verifies VCR configuration from environment variables.
func TestVCR_EnvironmentConfig(t *testing.T) {
	// VCR_MODE not set — disabled
	t.Setenv("VCR_MODE", "")
	cfg := vcr.FromEnvironment()
	assert.Nil(t, cfg)

	// VCR_MODE=off — disabled
	t.Setenv("VCR_MODE", "off")
	cfg = vcr.FromEnvironment()
	assert.Nil(t, cfg)

	// VCR_MODE=playback — enabled
	t.Setenv("VCR_MODE", "playback")
	t.Setenv("VCR_CASSETTE_DIR", "")
	cfg = vcr.FromEnvironment()
	require.NotNil(t, cfg)
	assert.True(t, cfg.Enabled)
	assert.Equal(t, vcr.Mode("playback"), cfg.Mode)

	// VCR_CASSETTE_DIR override
	t.Setenv("VCR_CASSETTE_DIR", "/tmp/test-cassettes")
	cfg = vcr.FromEnvironment()
	require.NotNil(t, cfg)
	assert.Equal(t, "/tmp/test-cassettes", cfg.CassetteDir)
}

// TestVCR_RecordOnce records on first request, replays on second.
func TestVCR_RecordOnce(t *testing.T) {
	callCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{"call": callCount})
	}))
	defer server.Close()

	cassetteDir := filepath.Join(t.TempDir(), "cassettes")
	require.NoError(t, os.MkdirAll(cassetteDir, 0755))

	cfg := &vcr.Config{
		Enabled:     true,
		Mode:        vcr.ModeRecordOnce,
		CassetteDir: cassetteDir,
		MatchMethod: true,
		MatchURI:    true,
	}

	// First request — records
	v1 := vcr.New(cfg)
	c1 := &http.Client{Transport: v1}
	req1, _ := http.NewRequest("GET", server.URL+"/api/data", nil)
	resp1, err := c1.Do(req1)
	require.NoError(t, err)
	resp1.Body.Close()
	assert.Equal(t, 1, callCount, "first request should hit server")

	// Second request with fresh VCR — should replay from cassette
	v2 := vcr.New(cfg)
	c2 := &http.Client{Transport: v2}
	req2, _ := http.NewRequest("GET", server.URL+"/api/data", nil)
	resp2, err := c2.Do(req2)
	require.NoError(t, err)
	defer resp2.Body.Close()
	assert.Equal(t, 1, callCount, "second request should NOT hit server (replay)")

	var result map[string]interface{}
	require.NoError(t, json.NewDecoder(resp2.Body).Decode(&result))
	assert.Equal(t, float64(1), result["call"], "should replay the recorded response")
}
