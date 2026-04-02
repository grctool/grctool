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
	auditLog     []SyncAuditEntry

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
	// GetSheetData returns parsed spreadsheet data.
	GetSheetData(ctx context.Context, fileID string) (*SheetData, error)
	// UpdateSheetData writes spreadsheet data.
	UpdateSheetData(ctx context.Context, fileID string, data *SheetData) error
	// CreateSheet creates a new spreadsheet.
	CreateSheet(ctx context.Context, folderID, title string, data *SheetData) (string, error)
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

// SyncAuditEntry records a single sync operation for audit trail purposes.
type SyncAuditEntry struct {
	Timestamp  time.Time `json:"timestamp"`
	Provider   string    `json:"provider"`
	Direction  string    `json:"direction"` // "inbound", "outbound", "bidirectional"
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	ExternalID string    `json:"external_id,omitempty"`
	Action     string    `json:"action"` // "imported", "exported", "conflict_detected", "conflict_resolved", "unchanged", "error"
	Resolution string    `json:"resolution,omitempty"`
	Winner     string    `json:"winner,omitempty"`
	Error      string    `json:"error,omitempty"`
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

// Capabilities reports that GDrive supports policies (as Docs) and controls
// and evidence tasks (as Sheets), with write and change detection.
func (p *GDriveSyncProvider) Capabilities() interfaces.ProviderCapabilities {
	return interfaces.ProviderCapabilities{
		SupportsPolicies:      true,
		SupportsControls:      true,
		SupportsEvidenceTasks: true,
		SupportsWrite:         true,
		SupportsChangeDetect:  true,
	}
}

func (p *GDriveSyncProvider) TestConnection(ctx context.Context) error {
	return p.driveClient.TestConnection(ctx)
}

// ---------------------------------------------------------------------------
// Policies (Google Docs)
// ---------------------------------------------------------------------------

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
		pol := domain.Policy{
			ID:          f.ID,
			Name:        f.Name,
			Content:     content,
			ExternalIDs: map[string]string{"gdrive": f.ID},
			SyncMetadata: &domain.SyncMetadata{
				ContentHash:  map[string]string{"gdrive": gdocs.ContentHash(doc)},
				LastSyncTime: map[string]time.Time{"gdrive": time.Now()},
			},
		}
		setContentHash(&pol, p.logger)
		policies = append(policies, pol)
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
	pol := &domain.Policy{
		ID:          id,
		Name:        doc.Title,
		Content:     content,
		ExternalIDs: map[string]string{"gdrive": id},
		SyncMetadata: &domain.SyncMetadata{
			ContentHash: map[string]string{"gdrive": gdocs.ContentHash(doc)},
		},
	}
	setContentHash(pol, p.logger)
	return pol, nil
}

// ---------------------------------------------------------------------------
// Controls (Google Sheets — "Control Matrix")
// ---------------------------------------------------------------------------

func (p *GDriveSyncProvider) ListControls(ctx context.Context, opts interfaces.ListOptions) ([]domain.Control, int, error) {
	files, err := p.driveClient.ListFiles(ctx, p.rootFolderID, "application/vnd.google-apps.spreadsheet")
	if err != nil {
		return nil, 0, fmt.Errorf("list control sheets: %w", err)
	}

	builder := &ControlMatrixBuilder{}
	var allControls []domain.Control

	for _, f := range files {
		sheet, err := p.driveClient.GetSheetData(ctx, f.ID)
		if err != nil {
			p.logger.Warn("skip sheet", logger.Field{Key: "file_id", Value: f.ID}, logger.Field{Key: "error", Value: err.Error()})
			continue
		}
		if sheet.Title != "Control Matrix" {
			continue
		}
		controls, err := builder.ParseControlMatrix(sheet)
		if err != nil {
			p.logger.Warn("skip malformed control matrix", logger.Field{Key: "file_id", Value: f.ID}, logger.Field{Key: "error", Value: err.Error()})
			continue
		}
		for i := range controls {
			controls[i].ExternalIDs = map[string]string{"gdrive": f.ID}
			controls[i].SyncMetadata = &domain.SyncMetadata{
				LastSyncTime: map[string]time.Time{"gdrive": time.Now()},
			}
			setControlContentHash(&controls[i], p.logger)
		}
		allControls = append(allControls, controls...)
	}

	return allControls, len(allControls), nil
}

func (p *GDriveSyncProvider) GetControl(ctx context.Context, id string) (*domain.Control, error) {
	sheet, err := p.driveClient.GetSheetData(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get control sheet %s: %w", id, err)
	}
	builder := &ControlMatrixBuilder{}
	controls, err := builder.ParseControlMatrix(sheet)
	if err != nil {
		return nil, fmt.Errorf("parse control sheet %s: %w", id, err)
	}
	if len(controls) == 0 {
		return nil, fmt.Errorf("no controls found in sheet %s", id)
	}
	controls[0].ExternalIDs = map[string]string{"gdrive": id}
	setControlContentHash(&controls[0], p.logger)
	return &controls[0], nil
}

// ---------------------------------------------------------------------------
// Evidence Tasks (Google Sheets — "Evidence Tasks")
// ---------------------------------------------------------------------------

func (p *GDriveSyncProvider) ListEvidenceTasks(ctx context.Context, opts interfaces.ListOptions) ([]domain.EvidenceTask, int, error) {
	files, err := p.driveClient.ListFiles(ctx, p.rootFolderID, "application/vnd.google-apps.spreadsheet")
	if err != nil {
		return nil, 0, fmt.Errorf("list evidence task sheets: %w", err)
	}

	builder := &ControlMatrixBuilder{}
	var allTasks []domain.EvidenceTask

	for _, f := range files {
		sheet, err := p.driveClient.GetSheetData(ctx, f.ID)
		if err != nil {
			p.logger.Warn("skip sheet", logger.Field{Key: "file_id", Value: f.ID}, logger.Field{Key: "error", Value: err.Error()})
			continue
		}
		if sheet.Title != "Evidence Tasks" {
			continue
		}
		tasks, err := builder.ParseEvidenceTaskSheet(sheet)
		if err != nil {
			p.logger.Warn("skip malformed evidence task sheet", logger.Field{Key: "file_id", Value: f.ID}, logger.Field{Key: "error", Value: err.Error()})
			continue
		}
		for i := range tasks {
			tasks[i].ExternalIDs = map[string]string{"gdrive": f.ID}
			tasks[i].SyncMetadata = &domain.SyncMetadata{
				LastSyncTime: map[string]time.Time{"gdrive": time.Now()},
			}
			setTaskContentHash(&tasks[i], p.logger)
		}
		allTasks = append(allTasks, tasks...)
	}

	return allTasks, len(allTasks), nil
}

func (p *GDriveSyncProvider) GetEvidenceTask(ctx context.Context, id string) (*domain.EvidenceTask, error) {
	sheet, err := p.driveClient.GetSheetData(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("get evidence task sheet %s: %w", id, err)
	}
	builder := &ControlMatrixBuilder{}
	tasks, err := builder.ParseEvidenceTaskSheet(sheet)
	if err != nil {
		return nil, fmt.Errorf("parse evidence task sheet %s: %w", id, err)
	}
	if len(tasks) == 0 {
		return nil, fmt.Errorf("no evidence tasks found in sheet %s", id)
	}
	tasks[0].ExternalIDs = map[string]string{"gdrive": id}
	setTaskContentHash(&tasks[0], p.logger)
	return &tasks[0], nil
}

// ---------------------------------------------------------------------------
// Write operations
// ---------------------------------------------------------------------------

func (p *GDriveSyncProvider) PushPolicy(ctx context.Context, policy *domain.Policy) error {
	gdID := ""
	if policy.ExternalIDs != nil {
		gdID = policy.ExternalIDs["gdrive"]
	}

	if gdID != "" {
		if err := p.driveClient.UpdateDoc(ctx, gdID, policy.Content); err != nil {
			p.recordAudit("policy", policy.ID, gdID, "exported", "error", "", err.Error())
			return err
		}
		p.recordAudit("policy", policy.ID, gdID, "exported", "", "", "")
		return nil
	}
	newID, err := p.driveClient.CreateDoc(ctx, p.rootFolderID, policy.Name, policy.Content)
	if err != nil {
		p.recordAudit("policy", policy.ID, "", "exported", "error", "", err.Error())
		return err
	}
	if policy.ExternalIDs == nil {
		policy.ExternalIDs = make(map[string]string)
	}
	policy.ExternalIDs["gdrive"] = newID
	p.recordAudit("policy", policy.ID, newID, "exported", "", "", "")
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

// ---------------------------------------------------------------------------
// Conflict resolution
// ---------------------------------------------------------------------------

// ResolveConflict applies a conflict resolution strategy.
// Per FEAT-002 US-003: configurable per-policy or globally.
func (p *GDriveSyncProvider) ResolveConflict(ctx context.Context, conflict interfaces.Conflict, resolution interfaces.ConflictResolution) error {
	switch resolution {
	case interfaces.ConflictResolutionLocalWins:
		p.logger.Info("resolving conflict: local wins",
			logger.String("entity", conflict.EntityID),
			logger.String("provider", conflict.Provider))
		p.recordAudit(conflict.EntityType, conflict.EntityID, "", "conflict_resolved", string(resolution), "local", "")
		return nil

	case interfaces.ConflictResolutionRemoteWins:
		p.logger.Info("resolving conflict: remote wins",
			logger.String("entity", conflict.EntityID),
			logger.String("provider", conflict.Provider))
		p.recordAudit(conflict.EntityType, conflict.EntityID, "", "conflict_resolved", string(resolution), "remote", "")
		return nil

	case interfaces.ConflictResolutionNewestWins:
		p.logger.Info("resolving conflict: newest wins",
			logger.String("entity", conflict.EntityID),
			logger.String("provider", conflict.Provider))
		p.recordAudit(conflict.EntityType, conflict.EntityID, "", "conflict_resolved", string(resolution), "", "")
		return nil

	case interfaces.ConflictResolutionManual:
		p.logger.Info("conflict flagged for manual resolution",
			logger.String("entity", conflict.EntityID),
			logger.String("provider", conflict.Provider))
		p.recordAudit(conflict.EntityType, conflict.EntityID, "", "conflict_detected", string(resolution), "", "")
		return nil

	default:
		return fmt.Errorf("gdrive: unknown conflict resolution strategy: %s", resolution)
	}
}

// ---------------------------------------------------------------------------
// Change detection
// ---------------------------------------------------------------------------

func (p *GDriveSyncProvider) DetectChanges(ctx context.Context, since time.Time) (*interfaces.ChangeSet, error) {
	files, err := p.driveClient.ListFiles(ctx, p.rootFolderID, "")
	if err != nil {
		return nil, fmt.Errorf("list files for change detection: %w", err)
	}

	var changes []interfaces.ChangeEntry
	for _, f := range files {
		if f.Modified.After(since) {
			entityType := entityTypeFromMIME(f.MimeType)
			changes = append(changes, interfaces.ChangeEntry{
				EntityType: entityType,
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

// ---------------------------------------------------------------------------
// Audit trail
// ---------------------------------------------------------------------------

// AuditLog returns the accumulated audit entries for the current sync cycle.
func (p *GDriveSyncProvider) AuditLog() []SyncAuditEntry {
	return p.auditLog
}

// ClearAuditLog resets the audit log for the next sync cycle.
func (p *GDriveSyncProvider) ClearAuditLog() {
	p.auditLog = nil
}

func (p *GDriveSyncProvider) recordAudit(entityType, entityID, externalID, action, resolution, winner, errMsg string) {
	p.auditLog = append(p.auditLog, SyncAuditEntry{
		Timestamp:  time.Now(),
		Provider:   "gdrive",
		Direction:  directionForAction(action),
		EntityType: entityType,
		EntityID:   entityID,
		ExternalID: externalID,
		Action:     action,
		Resolution: resolution,
		Winner:     winner,
		Error:      errMsg,
	})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func directionForAction(action string) string {
	switch action {
	case "exported":
		return "outbound"
	case "imported":
		return "inbound"
	case "conflict_detected", "conflict_resolved":
		return "bidirectional"
	default:
		return "unknown"
	}
}

func entityTypeFromMIME(mimeType string) string {
	switch mimeType {
	case "application/vnd.google-apps.document":
		return "policy"
	case "application/vnd.google-apps.spreadsheet":
		return "control" // Sheets can be controls or evidence tasks; default to control
	default:
		return "unknown"
	}
}

// setContentHash computes and stores the SHA-256 content hash on a domain policy.
func setContentHash(policy *domain.Policy, log logger.Logger) {
	if policy.SyncMetadata == nil {
		policy.SyncMetadata = &domain.SyncMetadata{}
	}
	if policy.SyncMetadata.ContentHash == nil {
		policy.SyncMetadata.ContentHash = make(map[string]string)
	}
	if policy.SyncMetadata.LastSyncTime == nil {
		policy.SyncMetadata.LastSyncTime = make(map[string]time.Time)
	}
	hash, err := domain.ComputeContentHash(policy)
	if err != nil {
		log.Warn("failed to compute content hash for GDrive policy",
			logger.String("id", policy.ID), logger.String("error", err.Error()))
		return
	}
	policy.SyncMetadata.ContentHash["gdrive"] = hash
	policy.SyncMetadata.LastSyncTime["gdrive"] = time.Now()
}

func setControlContentHash(control *domain.Control, log logger.Logger) {
	if control.SyncMetadata == nil {
		control.SyncMetadata = &domain.SyncMetadata{}
	}
	if control.SyncMetadata.ContentHash == nil {
		control.SyncMetadata.ContentHash = make(map[string]string)
	}
	if control.SyncMetadata.LastSyncTime == nil {
		control.SyncMetadata.LastSyncTime = make(map[string]time.Time)
	}
	hash, err := domain.ComputeContentHash(control)
	if err != nil {
		log.Warn("failed to compute content hash for GDrive control",
			logger.String("id", control.ID), logger.String("error", err.Error()))
		return
	}
	control.SyncMetadata.ContentHash["gdrive"] = hash
	control.SyncMetadata.LastSyncTime["gdrive"] = time.Now()
}

func setTaskContentHash(task *domain.EvidenceTask, log logger.Logger) {
	if task.SyncMetadata == nil {
		task.SyncMetadata = &domain.SyncMetadata{}
	}
	if task.SyncMetadata.ContentHash == nil {
		task.SyncMetadata.ContentHash = make(map[string]string)
	}
	if task.SyncMetadata.LastSyncTime == nil {
		task.SyncMetadata.LastSyncTime = make(map[string]time.Time)
	}
	hash, err := domain.ComputeContentHash(task)
	if err != nil {
		log.Warn("failed to compute content hash for GDrive task",
			logger.String("id", task.ID), logger.String("error", err.Error()))
		return
	}
	task.SyncMetadata.ContentHash["gdrive"] = hash
	task.SyncMetadata.LastSyncTime["gdrive"] = time.Now()
}
