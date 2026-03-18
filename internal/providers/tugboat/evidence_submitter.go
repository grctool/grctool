// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tugboat

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
	tugboatclient "github.com/grctool/grctool/internal/tugboat"
)

// Compile-time assertion.
var _ interfaces.EvidenceSubmitter = (*TugboatDataProvider)(nil)

// SubmitEvidence uploads evidence to Tugboat via the Custom Evidence Integration API.
// Requires a collector URL configured per task in .grctool.yaml.
// The file is written to a temp file since Tugboat's client expects a file path.
func (p *TugboatDataProvider) SubmitEvidence(ctx context.Context, taskID string, file io.Reader, meta interfaces.SubmissionMetadata) error {
	// Write to temp file (Tugboat client expects a file path, not a reader)
	tmpDir := os.TempDir()
	filename := meta.Filename
	if filename == "" {
		filename = fmt.Sprintf("evidence-%s-%s.dat", taskID, time.Now().Format("20060102"))
	}
	tmpPath := filepath.Join(tmpDir, filename)

	tmpFile, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(tmpPath)

	if _, err := io.Copy(tmpFile, file); err != nil {
		tmpFile.Close()
		return fmt.Errorf("write temp file: %w", err)
	}
	tmpFile.Close()

	// Parse collected date
	collectedDate := time.Now()
	if meta.CollectedDate != "" {
		if parsed, err := time.Parse("2006-01-02", meta.CollectedDate); err == nil {
			collectedDate = parsed
		}
	}

	// Build request — collector URL must be configured externally
	// (stored in .grctool.yaml tugboat.collector_urls)
	req := &tugboatclient.SubmitEvidenceRequest{
		CollectorURL:  "", // Must be set by caller via config
		FilePath:      tmpPath,
		CollectedDate: collectedDate,
		ContentType:   meta.ContentType,
	}

	p.logger.Info("submitting evidence to Tugboat",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "filename", Value: filename})

	_, err = p.client.SubmitEvidence(ctx, req)
	return err
}

// ListAttachments returns evidence attachments for a task from Tugboat.
func (p *TugboatDataProvider) ListAttachments(ctx context.Context, taskID string, opts interfaces.ListOptions) ([]interfaces.Attachment, int, error) {
	taskIDInt, err := strconv.Atoi(taskID)
	if err != nil {
		// Try ExternalIDs — taskID might be a GRCTool reference
		taskIDInt = 0
	}

	if taskIDInt == 0 {
		return nil, 0, fmt.Errorf("invalid task ID for Tugboat attachment listing: %s", taskID)
	}

	apiAtts, err := p.client.GetEvidenceAttachmentsByTask(ctx, taskIDInt)
	if err != nil {
		return nil, 0, fmt.Errorf("list attachments for task %s: %w", taskID, err)
	}

	atts := make([]interfaces.Attachment, 0, len(apiAtts))
	for _, a := range apiAtts {
		att := interfaces.Attachment{
			ID:              strconv.Itoa(a.ID),
			TaskID:          taskID,
			Type:            a.Type,
			CollectedDate:   a.Collected,
			Notes:           a.Notes,
			URL:             a.URL,
			IntegrationType: a.IntegrationType,
			Deleted:         a.Deleted,
		}
		if a.Attachment != nil {
			att.Filename = a.Attachment.OriginalFilename
			att.MimeType = a.Attachment.MimeType
		}
		if a.Owner != nil {
			att.Owner = a.Owner.DisplayName
		}
		atts = append(atts, att)
	}

	return atts, len(atts), nil
}

// DownloadAttachment gets a signed URL for an attachment and returns a reader.
func (p *TugboatDataProvider) DownloadAttachment(ctx context.Context, attachmentID string) (io.ReadCloser, string, error) {
	attIDInt, err := strconv.Atoi(attachmentID)
	if err != nil {
		return nil, "", fmt.Errorf("invalid attachment ID: %s", attachmentID)
	}

	resp, err := p.client.GetAttachmentDownloadURL(ctx, attIDInt)
	if err != nil {
		return nil, "", fmt.Errorf("get download URL for attachment %s: %w", attachmentID, err)
	}

	return nil, resp.OriginalFilename, fmt.Errorf("download via signed URL not yet implemented (URL: %s)", resp.URL)
}
