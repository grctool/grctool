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

package vcr

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestVCR_RecordAndPlayback(t *testing.T) {
	// Create temporary directory for test cassettes
	tmpDir, err := os.MkdirTemp("", "vcr_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Test server that returns predictable responses
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/test" {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			_, _ = w.Write([]byte(`{"message": "test response", "timestamp": "2024-01-01T12:00:00Z"}`))
		} else {
			w.WriteHeader(404)
			_, _ = w.Write([]byte("Not Found"))
		}
	}))
	defer server.Close()

	// VCR configuration for recording
	config := &Config{
		Enabled:         true,
		Mode:            ModeRecord,
		CassetteDir:     tmpDir,
		SanitizeHeaders: true,
		RedactHeaders:   []string{"authorization"},
		MatchMethod:     true,
		MatchURI:        true,
	}

	// Create VCR transport
	vcr := New(config)

	// Create HTTP client with VCR transport
	client := &http.Client{
		Transport: vcr,
		Timeout:   5 * time.Second,
	}

	// Make a request (this should be recorded)
	req, err := http.NewRequest("GET", server.URL+"/test?param=value", nil)
	if err != nil {
		t.Fatalf("failed to create request: %v", err)
	}
	req.Header.Set("Authorization", "Bearer secret-token")
	req.Header.Set("User-Agent", "test-client")

	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatalf("failed to read response: %v", err)
	}

	// Verify response
	if resp.StatusCode != 200 {
		t.Errorf("expected status 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "test response") {
		t.Errorf("expected body to contain 'test response', got: %s", string(body))
	}

	// Check that cassette was created
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read cassette dir: %v", err)
	}
	if len(files) != 1 {
		t.Errorf("expected 1 cassette file, got %d", len(files))
	}

	// Read cassette file to verify content
	cassetteFile := filepath.Join(tmpDir, files[0].Name())
	cassetteData, err := os.ReadFile(cassetteFile)
	if err != nil {
		t.Fatalf("failed to read cassette file: %v", err)
	}

	// Verify that sensitive headers are redacted
	if strings.Contains(string(cassetteData), "secret-token") {
		t.Error("cassette should not contain sensitive token")
	}
	if !strings.Contains(string(cassetteData), "[REDACTED]") {
		t.Error("cassette should contain [REDACTED] for sensitive headers")
	}

	// Now test playback mode
	config.Mode = ModePlayback
	vcrPlayback := New(config)
	clientPlayback := &http.Client{
		Transport: vcrPlayback,
		Timeout:   5 * time.Second,
	}

	// Stop the test server to ensure we're getting data from cassette
	server.Close()

	// Make the same request again (should come from cassette)
	req2, err := http.NewRequest("GET", server.URL+"/test?param=value", nil)
	if err != nil {
		t.Fatalf("failed to create playback request: %v", err)
	}
	req2.Header.Set("Authorization", "Bearer secret-token")
	req2.Header.Set("User-Agent", "test-client")

	resp2, err := clientPlayback.Do(req2)
	if err != nil {
		t.Fatalf("failed to make playback request: %v", err)
	}
	defer resp2.Body.Close()

	// Read playback response
	body2, err := io.ReadAll(resp2.Body)
	if err != nil {
		t.Fatalf("failed to read playback response: %v", err)
	}

	// Verify playback response matches original
	if resp2.StatusCode != 200 {
		t.Errorf("expected playback status 200, got %d", resp2.StatusCode)
	}
	if !bytes.Equal(body, body2) {
		t.Errorf("playback response doesn't match original. Original: %s, Playback: %s", string(body), string(body2))
	}
}

func TestVCR_RecordOnceMode(t *testing.T) {
	// Create temporary directory for test cassettes
	tmpDir, err := os.MkdirTemp("", "vcr_test_record_once")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"message": "response", "count": ` + string(rune(requestCount+48)) + `}`))
	}))
	defer server.Close()

	config := &Config{
		Enabled:     true,
		Mode:        ModeRecordOnce,
		CassetteDir: tmpDir,
		MatchMethod: true,
		MatchURI:    true,
	}

	vcr := New(config)
	client := &http.Client{Transport: vcr}

	// First request - should record
	req1, _ := http.NewRequest("GET", server.URL+"/api/test", nil)
	resp1, err := client.Do(req1)
	if err != nil {
		t.Fatalf("failed to make first request: %v", err)
	}
	resp1.Body.Close()

	// Second request - should playback from cassette
	req2, _ := http.NewRequest("GET", server.URL+"/api/test", nil)
	resp2, err := client.Do(req2)
	if err != nil {
		t.Fatalf("failed to make second request: %v", err)
	}
	defer resp2.Body.Close()

	body2, _ := io.ReadAll(resp2.Body)

	// Verify server was only called once
	if requestCount != 1 {
		t.Errorf("expected server to be called once, but was called %d times", requestCount)
	}

	// Verify we got the original response (count should be 1, not 2)
	if !strings.Contains(string(body2), `"count": 1`) {
		t.Errorf("expected response to contain count 1, got: %s", string(body2))
	}
}

func TestVCR_CassetteNaming(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vcr_test_naming")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		Enabled:     true,
		Mode:        ModeRecord,
		CassetteDir: tmpDir,
		MatchMethod: true,
		MatchURI:    true,
	}

	vcr := New(config)

	// Test different requests generate different cassette names
	tests := []struct {
		method string
		path   string
		query  string
	}{
		{"GET", "/api/policies", "page=1&size=20"},
		{"GET", "/api/policies", "page=2&size=20"},
		{"GET", "/api/controls", "framework=soc2"},
		{"POST", "/api/evidence", ""},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"result": "ok"}`))
	}))
	defer server.Close()

	client := &http.Client{Transport: vcr}

	for _, test := range tests {
		url := server.URL + test.path
		if test.query != "" {
			url += "?" + test.query
		}

		req, _ := http.NewRequest(test.method, url, nil)
		resp, err := client.Do(req)
		if err != nil {
			t.Fatalf("failed to make request %s %s: %v", test.method, test.path, err)
		}
		resp.Body.Close()
	}

	// Check that multiple cassette files were created
	files, err := os.ReadDir(tmpDir)
	if err != nil {
		t.Fatalf("failed to read cassette dir: %v", err)
	}

	if len(files) != len(tests) {
		t.Errorf("expected %d cassette files, got %d", len(tests), len(files))
	}

	// Verify cassette names contain method and path info
	for _, file := range files {
		name := file.Name()
		if !strings.HasSuffix(name, ".json") {
			t.Errorf("cassette file should have .json extension: %s", name)
		}
		if !strings.Contains(name, "_") {
			t.Errorf("cassette name should contain underscores: %s", name)
		}
	}
}

func TestVCR_ModeOff(t *testing.T) {
	config := &Config{
		Enabled: false,
		Mode:    ModeOff,
	}

	vcr := New(config)

	// Should pass through to default transport
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
		_, _ = w.Write([]byte("pass through"))
	}))
	defer server.Close()

	client := &http.Client{Transport: vcr}
	resp, err := client.Get(server.URL)
	if err != nil {
		t.Fatalf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "pass through" {
		t.Errorf("expected 'pass through', got: %s", string(body))
	}
}

func TestVCR_Stats(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "vcr_test_stats")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	config := &Config{
		Enabled:     true,
		Mode:        ModeRecord,
		CassetteDir: tmpDir,
	}

	vcr := New(config)

	// Check initial stats
	stats := vcr.Stats()
	if stats["enabled"] != true {
		t.Error("expected VCR to be enabled in stats")
	}
	if stats["mode"] != string(ModeRecord) {
		t.Errorf("expected mode to be %s, got %s", ModeRecord, stats["mode"])
	}
	if stats["loaded_cassettes"] != 0 {
		t.Errorf("expected 0 loaded cassettes, got %v", stats["loaded_cassettes"])
	}
}
