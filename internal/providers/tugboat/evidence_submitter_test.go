// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tugboat

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/grctool/grctool/internal/adapters"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/interfaces"
	tugboatclient "github.com/grctool/grctool/internal/tugboat"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTugboatDataProvider_ImplementsEvidenceSubmitter(t *testing.T) {
	t.Parallel()
	var p interfaces.DataProvider = &TugboatDataProvider{}
	_, ok := p.(interfaces.EvidenceSubmitter)
	assert.True(t, ok, "TugboatDataProvider must implement EvidenceSubmitter")
}

func TestTugboatDataProvider_ListAttachments(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/api/org_evidence_attachment/")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"results": []map[string]interface{}{
				{
					"id":               1001,
					"collected":        "2026-03-15",
					"notes":            "Q1 review",
					"type":             "file",
					"deleted":          false,
					"org_evidence_id":  327992,
					"integration_type": "github",
					"attachment": map[string]interface{}{
						"id":                1,
						"original_filename": "access-report.csv",
						"mime_type":         "text/csv",
					},
					"owner": map[string]interface{}{
						"display_name": "Erik",
					},
				},
				{
					"id":              1002,
					"collected":       "2026-03-10",
					"type":            "url",
					"url":             "https://example.com/evidence",
					"deleted":         false,
					"org_evidence_id": 327992,
				},
			},
			"count":     2,
			"num_pages": 1,
		})
	}))
	defer server.Close()

	client := tugboatclient.NewClient(&config.TugboatConfig{BaseURL: server.URL}, nil)
	adapter := adapters.NewTugboatToDomain()
	provider := NewTugboatDataProvider(client, adapter, "13888", testhelpers.NewStubLogger())

	atts, total, err := provider.ListAttachments(context.Background(), "327992", interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, atts, 2)

	// First attachment: file type
	assert.Equal(t, "1001", atts[0].ID)
	assert.Equal(t, "327992", atts[0].TaskID)
	assert.Equal(t, "file", atts[0].Type)
	assert.Equal(t, "access-report.csv", atts[0].Filename)
	assert.Equal(t, "text/csv", atts[0].MimeType)
	assert.Equal(t, "Q1 review", atts[0].Notes)
	assert.Equal(t, "Erik", atts[0].Owner)
	assert.Equal(t, "github", atts[0].IntegrationType)

	// Second attachment: url type
	assert.Equal(t, "1002", atts[1].ID)
	assert.Equal(t, "url", atts[1].Type)
	assert.Equal(t, "https://example.com/evidence", atts[1].URL)
}

func TestTugboatDataProvider_ListAttachments_InvalidTaskID(t *testing.T) {
	t.Parallel()
	provider := NewTugboatDataProvider(nil, nil, "13888", testhelpers.NewStubLogger())

	_, _, err := provider.ListAttachments(context.Background(), "not-a-number", interfaces.ListOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID")
}

func TestTugboatDataProvider_ListAttachments_APIError(t *testing.T) {
	t.Parallel()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := tugboatclient.NewClient(&config.TugboatConfig{BaseURL: server.URL}, nil)
	provider := NewTugboatDataProvider(client, adapters.NewTugboatToDomain(), "13888", testhelpers.NewStubLogger())

	_, _, err := provider.ListAttachments(context.Background(), "327992", interfaces.ListOptions{})
	assert.Error(t, err)
}

func TestTugboatDataProvider_SubmitEvidence(t *testing.T) {
	t.Parallel()

	var receivedContentType string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedContentType = r.Header.Get("Content-Type")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"success":     true,
			"message":     "Evidence received",
			"received_at": "2026-03-18T12:00:00Z",
		})
	}))
	defer server.Close()

	client := tugboatclient.NewClient(&config.TugboatConfig{
		BaseURL: server.URL,
	}, nil)
	provider := NewTugboatDataProvider(client, adapters.NewTugboatToDomain(), "13888", testhelpers.NewStubLogger())

	meta := interfaces.SubmissionMetadata{
		CollectedDate: "2026-03-18",
		Filename:      "evidence.csv",
		ContentType:   "text/csv",
	}

	// SubmitEvidence will fail because CollectorURL is empty in the request.
	// This is by design — the collector URL comes from config, not the provider.
	err := provider.SubmitEvidence(context.Background(), "ET-0047",
		bytes.NewReader([]byte("user,role\nadmin,full")), meta)
	assert.Error(t, err) // Expected: collector URL is required
	assert.Contains(t, err.Error(), "collector URL")
	_ = receivedContentType
}

func TestTugboatDataProvider_DownloadAttachment(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]interface{}{
			"url":               "https://s3.amazonaws.com/signed-url",
			"original_filename": "report.csv",
		})
	}))
	defer server.Close()

	client := tugboatclient.NewClient(&config.TugboatConfig{BaseURL: server.URL}, nil)
	provider := NewTugboatDataProvider(client, adapters.NewTugboatToDomain(), "13888", testhelpers.NewStubLogger())

	_, filename, err := provider.DownloadAttachment(context.Background(), "1001")
	// Returns error because actual download via signed URL is not yet implemented
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not yet implemented")
	assert.Equal(t, "report.csv", filename)
}

func TestTugboatDataProvider_DownloadAttachment_InvalidID(t *testing.T) {
	t.Parallel()
	provider := NewTugboatDataProvider(nil, nil, "13888", testhelpers.NewStubLogger())
	_, _, err := provider.DownloadAttachment(context.Background(), "not-a-number")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid attachment ID")
}

func TestTugboatDataProvider_EvidenceSubmitter_TypeAssertion(t *testing.T) {
	t.Parallel()
	provider := NewTugboatDataProvider(nil, nil, "13888", testhelpers.NewStubLogger())
	var dp interfaces.DataProvider = provider
	es, ok := dp.(interfaces.EvidenceSubmitter)
	assert.True(t, ok)
	assert.NotNil(t, es)
}

// Verify attachment ID conversion
func TestAttachmentIDConversion(t *testing.T) {
	t.Parallel()
	// Tugboat uses int IDs, our interface uses string
	tugboatID := 1001
	stringID := strconv.Itoa(tugboatID)
	assert.Equal(t, "1001", stringID)

	backToInt, err := strconv.Atoi(stringID)
	require.NoError(t, err)
	assert.Equal(t, tugboatID, backToInt)
}
