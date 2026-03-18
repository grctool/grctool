// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

// Package gdrive implements a SyncProvider for Google Drive bidirectional sync.
// Policies are stored as Google Docs, controls as Sheets, evidence tasks as Sheets.
package gdrive

import (
	"context"
	"fmt"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/gdocs"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
)

// Compile-time interface assertion.
var _ interfaces.SyncProvider = (*GDriveSyncProvider)(nil)

// GDriveSyncProvider implements interfaces.SyncProvider for Google Drive.
type GDriveSyncProvider struct {
	rootFolderID string
	logger       logger.Logger

	// driveClient abstracts Google Drive API calls for testability.
	driveClient DriveClient
}

// DriveClient abstracts the Google Drive/Docs/Sheets API operations.
// Implementations: real Google API client, or stub for testing.
type DriveClient interface {
	// ListFiles returns files in a folder, optionally filtered by MIME type.
	ListFiles(ctx context.Context, folderID string, mimeType string) ([]DriveFile, error)
	// GetFileContent returns the text content of a Google Doc.
	GetFileContent(ctx context.Context, fileID string) (string, error)
	// CreateDoc creates a new Google Doc with the given content.
	CreateDoc(ctx context.Context, folderID, title, markdownContent string) (string, error)
	// UpdateDoc updates an existing Google Doc's content.
	UpdateDoc(ctx context.Context, fileID, markdownContent string) error
	// DeleteFile moves a file to trash.
	DeleteFile(ctx context.Context, fileID string) error
	// GetRevisions returns file revisions since a given time.
	GetRevisions(ctx context.Context, fileID string, since time.Time) ([]Revision, error)
	// TestConnection verifies API access.
	TestConnection(ctx context.Context) error
}

// DriveFile represents a file in Google Drive.
type DriveFile struct {
	ID       string    `json:"id"`
	Name     string    `json:"name"`
	MimeType string    `json:"mime_type"`
	Modified time.Time `json:"modified"`
}

// Revision represents a file revision.
type Revision struct {
	ID       string    `json:"id"`
	Modified time.Time `json:"modified"`
}

// NewGDriveSyncProvider creates a GDrive sync provider.
func NewGDriveSyncProvider(client DriveClient, rootFolderID string, log logger.Logger) *GDriveSyncProvider {
	return &GDriveSyncProvider{
		rootFolderID: rootFolderID,
		logger:       log,
		driveClient:  client,
	}
}

func (p *GDriveSyncProvider) Name() string { return "gdrive" }

// Capabilities reports that GDrive currently supports policies only (as Docs).
// Controls and evidence tasks (as Sheets) are planned but not yet implemented.
func (p *GDriveSyncProvider) Capabilities() interfaces.ProviderCapabilities {
	return interfaces.ProviderCapabilities{
		SupportsPolicies:      true,
		SupportsControls:      false,
		SupportsEvidenceTasks: false,
		SupportsWrite:         true,
		SupportsChangeDetect:  true,
	}
}

func (p *GDriveSyncProvider) TestConnection(ctx context.Context) error {
	return p.driveClient.TestConnection(ctx)
}

func (p *GDriveSyncProvider) ListPolicies(ctx context.Context, opts interfaces.ListOptions) ([]domain.Policy, int, error) {
	files, err := p.driveClient.ListFiles(ctx, p.rootFolderID, "application/vnd.google-apps.document")
	if err != nil {
		return nil, 0, fmt.Errorf("list policy docs: %w", err)
	}

	var policies []domain.Policy
	for _, f := range files {
		content, err := p.driveClient.GetFileContent(ctx, f.ID)
		if err != nil {
			p.logger.Warn("skip policy doc", logger.Field{Key: "file_id", Value: f.ID}, logger.Field{Key: "error", Value: err.Error()})
			continue
		}
		doc, err := gdocs.MarkdownToDoc(content)
		if err != nil {
			p.logger.Warn("skip malformed doc", logger.Field{Key: "file_id", Value: f.ID}, logger.Field{Key: "error", Value: err.Error()})
			continue
		}
		policies = append(policies, domain.Policy{
			ID:          f.ID,
			Name:        f.Name,
			Content:     content,
			ExternalIDs: map[string]string{"gdrive": f.ID},
			SyncMetadata: &domain.SyncMetadata{
				ContentHash:  map[string]string{"gdrive": gdocs.ContentHash(doc)},
				LastSyncTime: map[string]time.Time{"gdrive": time.Now()},
			},
		})
	}

	// Pagination
	if opts.PageSize > 0 {
		start := opts.Page * opts.PageSize
		if start >= len(policies) {
			return nil, len(policies), nil
		}
		end := start + opts.PageSize
		if end > len(policies) {
			end = len(policies)
		}
		return policies[start:end], len(policies), nil
	}
	return policies, len(policies), nil
}

func (p *GDriveSyncProvider) GetPolicy(ctx context.Context, id string) (*domain.Policy, error) {
	content, err := p.driveClient.GetFileContent(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get policy doc %s: %w", id, err)
	}
	doc, err := gdocs.MarkdownToDoc(content)
	if err != nil {
		return nil, fmt.Errorf("parse policy doc %s: %w", id, err)
	}
	return &domain.Policy{
		ID:          id,
		Name:        doc.Title,
		Content:     content,
		ExternalIDs: map[string]string{"gdrive": id},
		SyncMetadata: &domain.SyncMetadata{
			ContentHash: map[string]string{"gdrive": gdocs.ContentHash(doc)},
		},
	}, nil
}

// ListControls and GetControl read from Sheets — stub for now.
func (p *GDriveSyncProvider) ListControls(ctx context.Context, opts interfaces.ListOptions) ([]domain.Control, int, error) {
	return nil, 0, nil // Sheets integration in grct-8kp.4
}

func (p *GDriveSyncProvider) GetControl(ctx context.Context, id string) (*domain.Control, error) {
	return nil, fmt.Errorf("control %s: sheets integration not yet implemented", id)
}

func (p *GDriveSyncProvider) ListEvidenceTasks(ctx context.Context, opts interfaces.ListOptions) ([]domain.EvidenceTask, int, error) {
	return nil, 0, nil // Sheets integration in grct-8kp.4
}

func (p *GDriveSyncProvider) GetEvidenceTask(ctx context.Context, id string) (*domain.EvidenceTask, error) {
	return nil, fmt.Errorf("evidence task %s: sheets integration not yet implemented", id)
}

func (p *GDriveSyncProvider) PushPolicy(ctx context.Context, policy *domain.Policy) error {
	gdID := ""
	if policy.ExternalIDs != nil {
		gdID = policy.ExternalIDs["gdrive"]
	}

	if gdID != "" {
		// Update existing doc
		return p.driveClient.UpdateDoc(ctx, gdID, policy.Content)
	}
	// Create new doc
	newID, err := p.driveClient.CreateDoc(ctx, p.rootFolderID, policy.Name, policy.Content)
	if err != nil {
		return err
	}
	if policy.ExternalIDs == nil {
		policy.ExternalIDs = make(map[string]string)
	}
	policy.ExternalIDs["gdrive"] = newID
	return nil
}

func (p *GDriveSyncProvider) PushControl(ctx context.Context, control *domain.Control) error {
	return fmt.Errorf("sheets push not yet implemented")
}

func (p *GDriveSyncProvider) PushEvidenceTask(ctx context.Context, task *domain.EvidenceTask) error {
	return fmt.Errorf("sheets push not yet implemented")
}

func (p *GDriveSyncProvider) DeletePolicy(ctx context.Context, id string) error {
	return p.driveClient.DeleteFile(ctx, id)
}

func (p *GDriveSyncProvider) DeleteControl(ctx context.Context, id string) error {
	return p.driveClient.DeleteFile(ctx, id)
}

func (p *GDriveSyncProvider) DeleteEvidenceTask(ctx context.Context, id string) error {
	return p.driveClient.DeleteFile(ctx, id)
}

func (p *GDriveSyncProvider) ResolveConflict(ctx context.Context, conflict interfaces.Conflict, resolution interfaces.ConflictResolution) error {
	return fmt.Errorf("gdrive: conflict resolution not yet implemented for %s %s", conflict.EntityType, conflict.EntityID)
}

func (p *GDriveSyncProvider) DetectChanges(ctx context.Context, since time.Time) (*interfaces.ChangeSet, error) {
	files, err := p.driveClient.ListFiles(ctx, p.rootFolderID, "")
	if err != nil {
		return nil, fmt.Errorf("list files for change detection: %w", err)
	}

	var changes []interfaces.ChangeEntry
	for _, f := range files {
		if f.Modified.After(since) {
			changes = append(changes, interfaces.ChangeEntry{
				EntityType: "policy", // TODO: determine from MIME type
				EntityID:   f.ID,
				ChangeType: "updated",
				ModifiedAt: f.Modified,
			})
		}
	}

	return &interfaces.ChangeSet{
		Provider:   "gdrive",
		Since:      since,
		DetectedAt: time.Now(),
		Changes:    changes,
	}, nil
}
