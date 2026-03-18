// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package interfaces_test

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubEvidenceSubmitter implements both DataProvider and EvidenceSubmitter.
type stubEvidenceSubmitter struct {
	*testhelpers.StubDataProvider
	submissions []stubSubmission
	attachments map[string][]interfaces.Attachment
	files       map[string]string // attachmentID → content
	submitErr   error
}

type stubSubmission struct {
	TaskID   string
	Content  string
	Metadata interfaces.SubmissionMetadata
}

func (s *stubEvidenceSubmitter) SubmitEvidence(ctx context.Context, taskID string, file io.Reader, meta interfaces.SubmissionMetadata) error {
	if s.submitErr != nil {
		return s.submitErr
	}
	content, err := io.ReadAll(file)
	if err != nil {
		return err
	}
	s.submissions = append(s.submissions, stubSubmission{
		TaskID:   taskID,
		Content:  string(content),
		Metadata: meta,
	})
	return nil
}

func (s *stubEvidenceSubmitter) ListAttachments(ctx context.Context, taskID string, opts interfaces.ListOptions) ([]interfaces.Attachment, int, error) {
	atts, ok := s.attachments[taskID]
	if !ok {
		return nil, 0, nil
	}
	total := len(atts)
	if opts.PageSize > 0 {
		start := opts.Page * opts.PageSize
		if start >= total {
			return nil, total, nil
		}
		end := start + opts.PageSize
		if end > total {
			end = total
		}
		return atts[start:end], total, nil
	}
	return atts, total, nil
}

func (s *stubEvidenceSubmitter) DownloadAttachment(ctx context.Context, attachmentID string) (io.ReadCloser, string, error) {
	content, ok := s.files[attachmentID]
	if !ok {
		return nil, "", fmt.Errorf("attachment not found: %s", attachmentID)
	}
	return io.NopCloser(strings.NewReader(content)), attachmentID + ".csv", nil
}

// --- Tests ---

func TestEvidenceSubmitter_TypeAssertion_Positive(t *testing.T) {
	t.Parallel()
	var provider interfaces.DataProvider = &stubEvidenceSubmitter{
		StubDataProvider: testhelpers.NewStubDataProvider("sub-test"),
	}
	es, ok := provider.(interfaces.EvidenceSubmitter)
	assert.True(t, ok)
	assert.NotNil(t, es)
}

func TestEvidenceSubmitter_TypeAssertion_Negative(t *testing.T) {
	t.Parallel()
	var provider interfaces.DataProvider = testhelpers.NewStubDataProvider("plain")
	_, ok := provider.(interfaces.EvidenceSubmitter)
	assert.False(t, ok)
}

func TestEvidenceSubmitter_SubmitEvidence(t *testing.T) {
	t.Parallel()
	es := &stubEvidenceSubmitter{
		StubDataProvider: testhelpers.NewStubDataProvider("sub-test"),
	}

	meta := interfaces.SubmissionMetadata{
		CollectedDate: "2026-03-18",
		Notes:         "Quarterly access review",
		Window:        "2026-Q1",
		ContentType:   "text/csv",
		Filename:      "access-controls.csv",
	}

	err := es.SubmitEvidence(context.Background(), "ET-0047",
		bytes.NewReader([]byte("user,role,access\nadmin,admin,full")), meta)
	require.NoError(t, err)

	assert.Len(t, es.submissions, 1)
	assert.Equal(t, "ET-0047", es.submissions[0].TaskID)
	assert.Contains(t, es.submissions[0].Content, "admin")
	assert.Equal(t, "2026-Q1", es.submissions[0].Metadata.Window)
}

func TestEvidenceSubmitter_SubmitEvidence_Error(t *testing.T) {
	t.Parallel()
	es := &stubEvidenceSubmitter{
		StubDataProvider: testhelpers.NewStubDataProvider("sub-test"),
		submitErr:        fmt.Errorf("quota exceeded"),
	}

	err := es.SubmitEvidence(context.Background(), "ET-0001",
		bytes.NewReader([]byte("data")), interfaces.SubmissionMetadata{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}

func TestEvidenceSubmitter_ListAttachments(t *testing.T) {
	t.Parallel()
	es := &stubEvidenceSubmitter{
		StubDataProvider: testhelpers.NewStubDataProvider("sub-test"),
		attachments: map[string][]interfaces.Attachment{
			"ET-0047": {
				{ID: "att-1", TaskID: "ET-0047", Filename: "report.csv", Type: "file"},
				{ID: "att-2", TaskID: "ET-0047", Filename: "screenshot.png", Type: "file"},
			},
		},
	}

	atts, total, err := es.ListAttachments(context.Background(), "ET-0047", interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 2, total)
	assert.Len(t, atts, 2)
	assert.Equal(t, "report.csv", atts[0].Filename)
}

func TestEvidenceSubmitter_ListAttachments_Pagination(t *testing.T) {
	t.Parallel()
	es := &stubEvidenceSubmitter{
		StubDataProvider: testhelpers.NewStubDataProvider("sub-test"),
		attachments: map[string][]interfaces.Attachment{
			"ET-0047": {
				{ID: "att-1"}, {ID: "att-2"}, {ID: "att-3"},
			},
		},
	}

	atts, total, err := es.ListAttachments(context.Background(), "ET-0047",
		interfaces.ListOptions{Page: 0, PageSize: 2})
	require.NoError(t, err)
	assert.Equal(t, 3, total)
	assert.Len(t, atts, 2)
}

func TestEvidenceSubmitter_ListAttachments_Empty(t *testing.T) {
	t.Parallel()
	es := &stubEvidenceSubmitter{
		StubDataProvider: testhelpers.NewStubDataProvider("sub-test"),
		attachments:      map[string][]interfaces.Attachment{},
	}

	atts, total, err := es.ListAttachments(context.Background(), "ET-UNKNOWN", interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, total)
	assert.Empty(t, atts)
}

func TestEvidenceSubmitter_DownloadAttachment(t *testing.T) {
	t.Parallel()
	es := &stubEvidenceSubmitter{
		StubDataProvider: testhelpers.NewStubDataProvider("sub-test"),
		files: map[string]string{
			"att-1": "user,role\nadmin,full\nviewer,read",
		},
	}

	reader, filename, err := es.DownloadAttachment(context.Background(), "att-1")
	require.NoError(t, err)
	defer reader.Close()

	assert.Equal(t, "att-1.csv", filename)
	content, err := io.ReadAll(reader)
	require.NoError(t, err)
	assert.Contains(t, string(content), "admin")
}

func TestEvidenceSubmitter_DownloadAttachment_NotFound(t *testing.T) {
	t.Parallel()
	es := &stubEvidenceSubmitter{
		StubDataProvider: testhelpers.NewStubDataProvider("sub-test"),
		files:            map[string]string{},
	}

	_, _, err := es.DownloadAttachment(context.Background(), "nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestSubmissionMetadata_Fields(t *testing.T) {
	t.Parallel()
	meta := interfaces.SubmissionMetadata{
		CollectedDate: "2026-03-18",
		Notes:         "Automated collection",
		Window:        "2026-Q1",
		ContentType:   "application/pdf",
		Filename:      "evidence.pdf",
		Metadata:      map[string]string{"source": "terraform", "version": "1.5"},
	}
	assert.Equal(t, "2026-03-18", meta.CollectedDate)
	assert.Equal(t, "terraform", meta.Metadata["source"])
}

func TestAttachment_Fields(t *testing.T) {
	t.Parallel()
	att := interfaces.Attachment{
		ID:              "att-123",
		TaskID:          "ET-0047",
		Filename:        "access-review.csv",
		MimeType:        "text/csv",
		Type:            "file",
		CollectedDate:   "2026-03-18",
		Notes:           "Q1 review",
		Owner:           "compliance@example.com",
		IntegrationType: "github",
	}
	assert.Equal(t, "att-123", att.ID)
	assert.Equal(t, "github", att.IntegrationType)
	assert.False(t, att.Deleted)
}

// Verify caller pattern.
func TestEvidenceSubmitter_CallerPattern(t *testing.T) {
	t.Parallel()

	submitEvidence := func(provider interfaces.DataProvider, taskID string, data []byte) error {
		es, ok := provider.(interfaces.EvidenceSubmitter)
		if !ok {
			return fmt.Errorf("provider %s does not support evidence submission", provider.Name())
		}
		return es.SubmitEvidence(context.Background(), taskID,
			bytes.NewReader(data), interfaces.SubmissionMetadata{CollectedDate: "2026-03-18"})
	}

	// Provider WITH submission
	withSub := &stubEvidenceSubmitter{
		StubDataProvider: testhelpers.NewStubDataProvider("with-sub"),
	}
	err := submitEvidence(withSub, "ET-0001", []byte("evidence data"))
	assert.NoError(t, err)
	assert.Len(t, withSub.submissions, 1)

	// Provider WITHOUT submission
	withoutSub := testhelpers.NewStubDataProvider("no-sub")
	err = submitEvidence(withoutSub, "ET-0001", []byte("data"))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support evidence submission")
}
