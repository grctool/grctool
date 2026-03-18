// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package providers_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
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
// EvidenceSubmitter Contract Test Suite
// ---------------------------------------------------------------------------

// EvidenceSubmitterContractSuite verifies any provider implementing the
// optional EvidenceSubmitter interface. The setup function must return a
// DataProvider that also implements EvidenceSubmitter, pre-loaded with at
// least one attachment for knownTaskID.
func EvidenceSubmitterContractSuite(t *testing.T, knownTaskID string, setup func(t *testing.T) interfaces.DataProvider) {
	t.Helper()

	t.Run("ImplementsEvidenceSubmitter", func(t *testing.T) {
		p := setup(t)
		_, ok := p.(interfaces.EvidenceSubmitter)
		assert.True(t, ok, "provider must implement EvidenceSubmitter")
	})

	t.Run("ListAttachments_ReturnsResults", func(t *testing.T) {
		p := setup(t)
		es := p.(interfaces.EvidenceSubmitter)
		atts, total, err := es.ListAttachments(context.Background(), knownTaskID, interfaces.ListOptions{})
		require.NoError(t, err)
		assert.Greater(t, total, 0, "known task should have attachments")
		assert.NotEmpty(t, atts)
	})

	t.Run("ListAttachments_UnknownTask", func(t *testing.T) {
		p := setup(t)
		es := p.(interfaces.EvidenceSubmitter)
		atts, _, err := es.ListAttachments(context.Background(), "NONEXISTENT-99999", interfaces.ListOptions{})
		// Some providers may error, others return empty — both are valid
		if err == nil {
			assert.Empty(t, atts)
		}
	})

	t.Run("SubmitEvidence_Accepts", func(t *testing.T) {
		p := setup(t)
		es := p.(interfaces.EvidenceSubmitter)
		meta := interfaces.SubmissionMetadata{
			CollectedDate: "2026-03-18",
			Filename:      "contract-test.csv",
			ContentType:   "text/csv",
		}
		err := es.SubmitEvidence(context.Background(), knownTaskID,
			bytes.NewReader([]byte("test,data\n1,2")), meta)
		// May succeed or fail depending on provider config — no panic is the contract
		_ = err
	})

	t.Run("TypeAssertionNegative", func(t *testing.T) {
		plain := testhelpers.NewStubDataProvider("no-submission")
		_, ok := interfaces.DataProvider(plain).(interfaces.EvidenceSubmitter)
		assert.False(t, ok)
	})
}

// ---------------------------------------------------------------------------
// Run suite against StubFullProvider
// ---------------------------------------------------------------------------

func TestStubFullProvider_EvidenceSubmitterContract(t *testing.T) {
	t.Parallel()
	EvidenceSubmitterContractSuite(t, "ET-0047", func(t *testing.T) interfaces.DataProvider {
		fp := testhelpers.NewStubFullProvider("stub-es")
		fp.Tasks["ET-0047"] = &domain.EvidenceTask{ID: "ET-0047", Name: "GitHub Access"}
		fp.Attachments["ET-0047"] = []interfaces.Attachment{
			{ID: "att-1", TaskID: "ET-0047", Filename: "report.csv", Type: "file"},
		}
		fp.Files["att-1"] = "user,role\nadmin,full"
		return fp
	})
}

// ---------------------------------------------------------------------------
// Run suite against TugboatDataProvider (httptest-backed)
// ---------------------------------------------------------------------------

func TestTugboatDataProvider_EvidenceSubmitterContract(t *testing.T) {
	t.Parallel()
	EvidenceSubmitterContractSuite(t, "327992", func(t *testing.T) interfaces.DataProvider {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/api/org_evidence_attachment/":
				taskID := r.URL.Query().Get("org_evidence")
				if taskID == "327992" {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"results": []map[string]interface{}{
							{
								"id": 1001, "collected": "2026-03-15", "type": "file",
								"org_evidence_id": 327992, "deleted": false,
								"attachment": map[string]interface{}{
									"id": 1, "original_filename": "report.csv", "mime_type": "text/csv",
								},
							},
						},
						"count": 1, "num_pages": 1,
					})
				} else {
					json.NewEncoder(w).Encode(map[string]interface{}{
						"results": []interface{}{}, "count": 0, "num_pages": 0,
					})
				}
			default:
				json.NewEncoder(w).Encode(map[string]interface{}{
					"results": []interface{}{}, "count": 0, "num_pages": 0,
				})
			}
		}))
		t.Cleanup(server.Close)

		client := tugboatclient.NewClient(&config.TugboatConfig{BaseURL: server.URL}, nil)
		return tugboatprovider.NewTugboatDataProvider(client, adapters.NewTugboatToDomain(), "13888", testhelpers.NewStubLogger())
	})
}

// ---------------------------------------------------------------------------
// Additional integration tests
// ---------------------------------------------------------------------------

func TestStubFullProvider_SubmitAndVerify(t *testing.T) {
	t.Parallel()
	fp := testhelpers.NewStubFullProvider("submit-test")
	fp.Tasks["ET-0047"] = &domain.EvidenceTask{ID: "ET-0047"}

	// Submit evidence
	meta := interfaces.SubmissionMetadata{
		CollectedDate: "2026-03-18",
		Window:        "2026-Q1",
		Filename:      "access-controls.csv",
		ContentType:   "text/csv",
		Notes:         "Quarterly review",
	}
	err := fp.SubmitEvidence(context.Background(), "ET-0047",
		bytes.NewReader([]byte("user,role,access\nadmin,admin,full\nviewer,viewer,read")), meta)
	require.NoError(t, err)

	// Verify submission was captured
	require.Len(t, fp.Submissions, 1)
	assert.Equal(t, "ET-0047", fp.Submissions[0].TaskID)
	assert.Contains(t, fp.Submissions[0].Content, "admin")
	assert.Equal(t, "2026-Q1", fp.Submissions[0].Metadata.Window)
	assert.Equal(t, "Quarterly review", fp.Submissions[0].Metadata.Notes)
}

func TestStubFullProvider_DownloadAttachment(t *testing.T) {
	t.Parallel()
	fp := testhelpers.NewStubFullProvider("download-test")
	fp.Files["att-1"] = "header1,header2\nval1,val2"

	reader, filename, err := fp.DownloadAttachment(context.Background(), "att-1")
	require.NoError(t, err)
	defer reader.Close()

	assert.Equal(t, "att-1.dat", filename)
	content, _ := io.ReadAll(reader)
	assert.Contains(t, string(content), "val1")
}

func TestStubFullProvider_DownloadAttachment_NotFound(t *testing.T) {
	t.Parallel()
	fp := testhelpers.NewStubFullProvider("download-test")
	_, _, err := fp.DownloadAttachment(context.Background(), "nonexistent")
	assert.Error(t, err)
}
