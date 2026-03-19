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

package services

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/adapters"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/formatters"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/interpolation"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/naming"
	"github.com/grctool/grctool/internal/providers"
	tugboatProvider "github.com/grctool/grctool/internal/providers/tugboat"
	"github.com/grctool/grctool/internal/registry"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/tugboat"
	tugboatModels "github.com/grctool/grctool/internal/tugboat/models"
)

// SyncService handles synchronization between data providers and local storage.
// It supports iterating over multiple registered DataProviders via a ProviderRegistry,
// while preserving backward compatibility with direct Tugboat client usage.
type SyncService struct {
	tugboatClient         *tugboat.Client
	adapter               *adapters.TugboatToDomain
	registry              interfaces.ProviderRegistry // Provider registry for multi-provider sync
	storage               *storage.Storage            // Single storage interface for all operations
	policyFormatter       *formatters.PolicyFormatter
	controlFormatter      *formatters.ControlFormatter
	evidenceTaskFormatter *formatters.EvidenceTaskFormatter
	evidenceTaskRegistry  *registry.EvidenceTaskRegistry
	documentService       *DocumentService
	baseDir               string
	logger                logger.Logger
}

// NewSyncService creates a new sync service using a direct Tugboat client.
// It creates a ProviderRegistry internally and registers a TugboatDataProvider
// so that the provider-based sync methods work transparently.
// This constructor is backward compatible with existing callers.
func NewSyncService(tugboatClient *tugboat.Client, storage *storage.Storage, cfg *config.Config, log logger.Logger) *SyncService {
	// Create a ProviderRegistry and register the Tugboat provider
	reg := providers.NewProviderRegistry()
	adapter := adapters.NewTugboatToDomain()
	tp := tugboatProvider.NewTugboatDataProvider(tugboatClient, adapter, cfg.Tugboat.OrgID, log)
	if err := reg.Register(tp); err != nil {
		log.Warn("Failed to register tugboat provider in registry", logger.Error(err))
	}

	svc := NewSyncServiceWithRegistry(reg, storage, cfg, log)
	// Preserve the direct tugboat client for submission sync and validation
	svc.tugboatClient = tugboatClient
	svc.adapter = adapter
	return svc
}

// NewSyncServiceWithRegistry creates a new sync service using a ProviderRegistry.
// This allows syncing from any registered DataProvider, not just Tugboat.
// The tugboatClient field is left nil; callers that need submission sync
// should also set it explicitly or use NewSyncService instead.
func NewSyncServiceWithRegistry(reg interfaces.ProviderRegistry, storage *storage.Storage, cfg *config.Config, log logger.Logger) *SyncService {
	// Create interpolator from config
	interpolatorConfig := interpolation.InterpolatorConfig{
		Variables:         cfg.Interpolation.GetFlatVariables(),
		Enabled:           cfg.Interpolation.Enabled,
		OnMissingVariable: interpolation.MissingVariableIgnore,
	}
	interpolator := interpolation.NewStandardInterpolator(interpolatorConfig)

	// Create evidence task registry and load existing entries
	evidenceTaskRegistry := registry.NewEvidenceTaskRegistry(cfg.Storage.DataDir)
	if err := evidenceTaskRegistry.LoadRegistry(); err != nil {
		// Log warning but continue - registry will be created on first use
		log.Warn("Failed to load evidence task registry", logger.Error(err))
	}

	// Create evidence task formatter and set its registry
	evidenceTaskFormatter := formatters.NewEvidenceTaskFormatterWithInterpolation(interpolator)
	evidenceTaskFormatter.SetRegistry(evidenceTaskRegistry)

	return &SyncService{
		registry:              reg,
		storage:               storage,
		policyFormatter:       formatters.NewPolicyFormatterWithInterpolation(interpolator),
		controlFormatter:      formatters.NewControlFormatterWithInterpolation(interpolator),
		evidenceTaskFormatter: evidenceTaskFormatter,
		evidenceTaskRegistry:  evidenceTaskRegistry,
		documentService:       NewDocumentService(cfg),
		baseDir:               cfg.Storage.DataDir,
		logger:                log.WithComponent("sync_service"),
	}
}

// SyncOptions represents options for synchronization
type SyncOptions struct {
	OrgID       string `json:"org_id"`
	Framework   string `json:"framework,omitempty"`
	Policies    bool   `json:"policies"`
	Controls    bool   `json:"controls"`
	Evidence    bool   `json:"evidence"`
	Submissions bool   `json:"submissions"`
}

// SyncResult represents the result of a synchronization operation
type SyncResult struct {
	Policies      SyncStats     `json:"policies"`
	Controls      SyncStats     `json:"controls"`
	EvidenceTasks SyncStats     `json:"evidence_tasks"`
	Submissions   SyncStats     `json:"submissions"`
	Duration      time.Duration `json:"duration"`
	Errors        []string      `json:"errors,omitempty"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
}

// SyncStats represents statistics for a sync operation
type SyncStats struct {
	Total      int `json:"total"`
	Synced     int `json:"synced"`
	Detailed   int `json:"detailed"`
	Errors     int `json:"errors"`
	Skipped    int `json:"skipped"`
	Downloaded int `json:"downloaded"` // For submissions: number of files downloaded
}

// SyncAll performs a complete synchronization of all data types.
// It iterates over all registered providers in the registry for policies,
// controls, and evidence tasks. Submission sync still uses the direct
// Tugboat client when available.
func (s *SyncService) SyncAll(ctx context.Context, opts SyncOptions) (*SyncResult, error) {
	result := &SyncResult{
		StartTime: time.Now(),
		Errors:    []string{},
	}

	// Sync policies, controls, and evidence tasks from all registered providers
	providerNames := s.registry.List()
	for _, name := range providerNames {
		provider, err := s.registry.Get(name)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to get provider %s: %v", name, err))
			continue
		}

		s.logger.Info("Syncing from provider", logger.String("provider", name))

		if opts.Policies {
			stats, err := s.syncPoliciesFromProvider(ctx, provider, opts)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Policy sync failed (provider %s): %v", name, err))
			} else {
				result.Policies.Total += stats.Total
				result.Policies.Synced += stats.Synced
				result.Policies.Detailed += stats.Detailed
				result.Policies.Errors += stats.Errors
			}
		}

		if opts.Controls {
			stats, err := s.syncControlsFromProvider(ctx, provider, opts)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Control sync failed (provider %s): %v", name, err))
			} else {
				result.Controls.Total += stats.Total
				result.Controls.Synced += stats.Synced
				result.Controls.Detailed += stats.Detailed
				result.Controls.Errors += stats.Errors
			}
		}

		if opts.Evidence {
			stats, err := s.syncEvidenceTasksFromProvider(ctx, provider, opts)
			if err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("Evidence task sync failed (provider %s): %v", name, err))
			} else {
				result.EvidenceTasks.Total += stats.Total
				result.EvidenceTasks.Synced += stats.Synced
				result.EvidenceTasks.Detailed += stats.Detailed
				result.EvidenceTasks.Errors += stats.Errors
			}
		}
	}

	// Sync submissions if requested (still uses direct Tugboat client)
	if opts.Submissions {
		if err := s.syncSubmissions(ctx, opts, &result.Submissions); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Submission sync failed: %v", err))
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// Internal sync methods

// syncPolicies is a backward-compatible wrapper that delegates to the first
// registered provider. Kept for callers that use the old SyncOptions pattern.
func (s *SyncService) syncPolicies(ctx context.Context, opts SyncOptions, stats *SyncStats) error {
	providerNames := s.registry.List()
	for _, name := range providerNames {
		provider, err := s.registry.Get(name)
		if err != nil {
			stats.Errors++
			continue
		}
		provStats, err := s.syncPoliciesFromProvider(ctx, provider, opts)
		if err != nil {
			return err
		}
		stats.Total += provStats.Total
		stats.Synced += provStats.Synced
		stats.Detailed += provStats.Detailed
		stats.Errors += provStats.Errors
	}
	return nil
}

// syncControls is a backward-compatible wrapper that delegates to the first
// registered provider. Kept for callers that use the old SyncOptions pattern.
func (s *SyncService) syncControls(ctx context.Context, opts SyncOptions, stats *SyncStats) error {
	providerNames := s.registry.List()
	for _, name := range providerNames {
		provider, err := s.registry.Get(name)
		if err != nil {
			stats.Errors++
			continue
		}
		provStats, err := s.syncControlsFromProvider(ctx, provider, opts)
		if err != nil {
			return err
		}
		stats.Total += provStats.Total
		stats.Synced += provStats.Synced
		stats.Detailed += provStats.Detailed
		stats.Errors += provStats.Errors
	}
	return nil
}

// syncEvidenceTasks is a backward-compatible wrapper that delegates to
// registered providers. Kept for callers that use the old SyncOptions pattern.
func (s *SyncService) syncEvidenceTasks(ctx context.Context, opts SyncOptions, stats *SyncStats) error {
	providerNames := s.registry.List()
	for _, name := range providerNames {
		provider, err := s.registry.Get(name)
		if err != nil {
			stats.Errors++
			continue
		}
		provStats, err := s.syncEvidenceTasksFromProvider(ctx, provider, opts)
		if err != nil {
			return err
		}
		stats.Total += provStats.Total
		stats.Synced += provStats.Synced
		stats.Detailed += provStats.Detailed
		stats.Errors += provStats.Errors
	}
	return nil
}

// syncPoliciesFromProvider fetches all policies from a single DataProvider,
// processes reference IDs, saves to storage, and generates documents.
func (s *SyncService) syncPoliciesFromProvider(ctx context.Context, provider interfaces.DataProvider, opts SyncOptions) (*SyncStats, error) {
	stats := &SyncStats{}

	// Fetch all policies from the provider using pagination
	allPolicies, err := s.fetchAllPolicies(ctx, provider, opts.Framework)
	if err != nil {
		return stats, fmt.Errorf("failed to get policies from provider %s: %w", provider.Name(), err)
	}

	stats.Total = len(allPolicies)
	stats.Detailed = len(allPolicies)

	// Process reference IDs for all policies
	refProcessor := domain.NewPolicyReferenceProcessor()
	processedPolicies := refProcessor.ProcessPolicyReferences(allPolicies)

	// Save all processed policies
	for i := range processedPolicies {
		policy := &processedPolicies[i]

		// Save complete policy info
		if err := s.savePolicyThroughDataService(ctx, policy); err != nil {
			stats.Errors++
			continue
		}

		// Generate policy document
		if err := s.generatePolicyDocument(policy); err != nil {
			s.logger.Warn("Failed to generate policy document",
				logger.String("policy_id", policy.ID),
				logger.String("provider", provider.Name()),
				logger.Error(err))
		}

		stats.Synced++
	}

	return stats, nil
}

// syncControlsFromProvider fetches all controls from a single DataProvider,
// processes reference IDs, saves to storage, and generates documents.
func (s *SyncService) syncControlsFromProvider(ctx context.Context, provider interfaces.DataProvider, opts SyncOptions) (*SyncStats, error) {
	stats := &SyncStats{}

	// Fetch all controls from the provider using pagination
	allControls, err := s.fetchAllControls(ctx, provider, opts.Framework)
	if err != nil {
		return stats, fmt.Errorf("failed to get controls from provider %s: %w", provider.Name(), err)
	}

	stats.Total = len(allControls)
	stats.Detailed = len(allControls)

	// Process reference IDs for all controls
	refProcessor := domain.NewControlReferenceProcessor()
	processedControls := refProcessor.ProcessControlReferences(allControls)

	// Save all processed controls
	for _, domainControl := range processedControls {
		// Save complete control info
		if err := s.saveControlThroughDataService(ctx, &domainControl); err != nil {
			stats.Errors++
			continue
		}

		// Generate control document
		if err := s.generateControlDocument(&domainControl); err != nil {
			s.logger.Warn("Failed to generate control document",
				logger.String("control_id", domainControl.ID),
				logger.String("provider", provider.Name()),
				logger.Error(err))
		}

		stats.Synced++
	}

	return stats, nil
}

// syncEvidenceTasksFromProvider fetches all evidence tasks from a single DataProvider,
// processes reference IDs, preserves existing references, saves to storage,
// and generates documents.
func (s *SyncService) syncEvidenceTasksFromProvider(ctx context.Context, provider interfaces.DataProvider, opts SyncOptions) (*SyncStats, error) {
	stats := &SyncStats{}

	// Fetch all evidence tasks from the provider using pagination
	allTasks, err := s.fetchAllEvidenceTasks(ctx, provider, opts.Framework)
	if err != nil {
		return stats, fmt.Errorf("failed to get evidence tasks from provider %s: %w", provider.Name(), err)
	}

	stats.Total = len(allTasks)

	// First pass: process related controls and preserve existing reference IDs
	var domainTasks []domain.EvidenceTask
	for _, domainTask := range allTasks {
		// Process reference IDs for related controls
		refProcessor := domain.NewControlReferenceProcessor()
		processedControls := refProcessor.ProcessControlReferences(domainTask.RelatedControls)
		domainTask.RelatedControls = processedControls

		// Check if we have an existing task with a reference ID
		existingTask, err := s.storage.GetEvidenceTask(domainTask.ID)
		if err == nil && existingTask.ReferenceID != "" {
			// Preserve the existing reference ID
			domainTask.ReferenceID = existingTask.ReferenceID
		}

		domainTasks = append(domainTasks, domainTask)
		stats.Synced++
		stats.Detailed++
	}

	// Register all tasks in the registry and update their information
	for i := range domainTasks {
		// Register/update the task in the registry (this modifies the task)
		s.evidenceTaskFormatter.RegisterTask(&domainTasks[i])

		// Save the updated task with its reference ID
		if err := s.saveEvidenceTaskThroughDataService(ctx, &domainTasks[i]); err != nil {
			s.logger.Warn("Failed to save updated evidence task",
				logger.String("task_id", domainTasks[i].ID),
				logger.String("provider", provider.Name()),
				logger.Error(err))
		}

		// Generate evidence task document
		if err := s.generateEvidenceTaskDocument(&domainTasks[i]); err != nil {
			s.logger.Warn("Failed to generate evidence task document",
				logger.String("task_id", domainTasks[i].ID),
				logger.String("provider", provider.Name()),
				logger.Error(err))
		}
	}

	// Save the updated registry
	if err := s.evidenceTaskRegistry.SaveRegistry(); err != nil {
		s.logger.Warn("Failed to save evidence task registry", logger.Error(err))
	}

	return stats, nil
}

// fetchAllPolicies retrieves all policies from a DataProvider, handling pagination.
func (s *SyncService) fetchAllPolicies(ctx context.Context, provider interfaces.DataProvider, framework string) ([]domain.Policy, error) {
	var allPolicies []domain.Policy
	page := 1
	pageSize := 100

	for {
		policies, _, err := provider.ListPolicies(ctx, interfaces.ListOptions{
			Page:      page,
			PageSize:  pageSize,
			Framework: framework,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list policies page %d: %w", page, err)
		}

		allPolicies = append(allPolicies, policies...)

		if len(policies) < pageSize {
			break
		}
		page++
	}

	return allPolicies, nil
}

// fetchAllControls retrieves all controls from a DataProvider, handling pagination.
func (s *SyncService) fetchAllControls(ctx context.Context, provider interfaces.DataProvider, framework string) ([]domain.Control, error) {
	var allControls []domain.Control
	page := 1
	pageSize := 100

	for {
		controls, _, err := provider.ListControls(ctx, interfaces.ListOptions{
			Page:      page,
			PageSize:  pageSize,
			Framework: framework,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list controls page %d: %w", page, err)
		}

		allControls = append(allControls, controls...)

		if len(controls) < pageSize {
			break
		}
		page++
	}

	return allControls, nil
}

// fetchAllEvidenceTasks retrieves all evidence tasks from a DataProvider, handling pagination.
func (s *SyncService) fetchAllEvidenceTasks(ctx context.Context, provider interfaces.DataProvider, framework string) ([]domain.EvidenceTask, error) {
	var allTasks []domain.EvidenceTask
	page := 1
	pageSize := 100

	for {
		tasks, _, err := provider.ListEvidenceTasks(ctx, interfaces.ListOptions{
			Page:      page,
			PageSize:  pageSize,
			Framework: framework,
		})
		if err != nil {
			return nil, fmt.Errorf("failed to list evidence tasks page %d: %w", page, err)
		}

		allTasks = append(allTasks, tasks...)

		if len(tasks) < pageSize {
			break
		}
		page++
	}

	return allTasks, nil
}

func (s *SyncService) syncSubmissions(ctx context.Context, opts SyncOptions, stats *SyncStats) error {
	// Try provider-based path: resolve EvidenceSubmitter from registry
	submitter := s.resolveEvidenceSubmitter("tugboat")
	if submitter != nil {
		return s.syncSubmissionsViaProvider(ctx, opts, stats, submitter)
	}

	// Fallback: direct Tugboat client (legacy path)
	if s.tugboatClient == nil {
		return fmt.Errorf("no evidence submitter available: tugboat provider not registered and no direct client configured")
	}
	return s.syncSubmissionsLegacy(ctx, opts, stats)
}

// resolveEvidenceSubmitter checks the registry for a provider that implements EvidenceSubmitter.
func (s *SyncService) resolveEvidenceSubmitter(providerName string) interfaces.EvidenceSubmitter {
	provider, err := s.registry.Get(providerName)
	if err != nil {
		return nil
	}
	if es, ok := provider.(interfaces.EvidenceSubmitter); ok {
		return es
	}
	return nil
}

// syncSubmissionsViaProvider syncs submissions using the EvidenceSubmitter interface.
func (s *SyncService) syncSubmissionsViaProvider(ctx context.Context, opts SyncOptions, stats *SyncStats, submitter interfaces.EvidenceSubmitter) error {
	// Get all evidence tasks via the registry (already provider-based)
	tasks, err := s.getAllEvidenceTasksFromProvider(ctx, opts.Framework)
	if err != nil {
		return fmt.Errorf("failed to get evidence tasks for submission sync: %w", err)
	}

	totalAttachments := 0

	for _, task := range tasks {
		taskID := task.ID
		if extID, ok := task.ExternalIDs["tugboat"]; ok {
			taskID = extID
		}

		atts, _, err := submitter.ListAttachments(ctx, taskID, interfaces.ListOptions{})
		if err != nil {
			s.logger.Warn("Failed to list attachments for evidence task",
				logger.String("task_id", taskID),
				logger.Error(err))
			stats.Errors++
			continue
		}

		totalAttachments += len(atts)

		if err := s.saveInterfaceAttachmentsForTask(ctx, task, atts, submitter, stats); err != nil {
			s.logger.Warn("Failed to save attachments for evidence task",
				logger.String("task_id", taskID),
				logger.Error(err))
			stats.Errors++
			continue
		}

		stats.Synced++
	}

	stats.Total = totalAttachments
	stats.Detailed = totalAttachments - stats.Errors
	return nil
}

// saveInterfaceAttachmentsForTask saves attachments retrieved via the EvidenceSubmitter interface.
func (s *SyncService) saveInterfaceAttachmentsForTask(ctx context.Context, task domain.EvidenceTask, atts []interfaces.Attachment, submitter interfaces.EvidenceSubmitter, stats *SyncStats) error {
	if len(atts) == 0 {
		return nil
	}

	taskRef := task.ReferenceID
	if taskRef == "" {
		s.logger.Warn("Evidence task has no reference ID, skipping attachment save",
			logger.String("task_id", task.ID))
		return nil
	}

	// Group by window
	windowMap := make(map[string][]interfaces.Attachment)
	for _, att := range atts {
		window := s.getWindowFromDate(att.CollectedDate)
		windowMap[window] = append(windowMap[window], att)
	}

	for window, windowAtts := range windowMap {
		taskDirName := naming.GetEvidenceTaskDirName(task.Name, taskRef, task.ID)
		evidenceDir := filepath.Join(s.baseDir, "evidence", taskDirName, window, naming.SubfolderArchive)
		if err := os.MkdirAll(evidenceDir, 0755); err != nil {
			stats.Errors++
			continue
		}

		for _, att := range windowAtts {
			if att.Type == "url" {
				filename := fmt.Sprintf("url_reference_%s.txt", att.ID)
				destPath := filepath.Join(evidenceDir, filename)
				urlContent := fmt.Sprintf("URL: %s\nNotes: %s\nCollected: %s\n", att.URL, att.Notes, att.CollectedDate)
				if err := os.WriteFile(destPath, []byte(urlContent), 0644); err != nil {
					stats.Errors++
				} else {
					stats.Downloaded++
				}
			} else if att.Filename != "" {
				// Download via interface
				reader, _, err := submitter.DownloadAttachment(ctx, att.ID)
				if err != nil {
					s.logger.Warn("Failed to download attachment via provider",
						logger.String("attachment_id", att.ID),
						logger.Error(err))
					stats.Errors++
					continue
				}
				if reader != nil {
					destPath := filepath.Join(evidenceDir, att.Filename)
					data, err := io.ReadAll(reader)
					reader.Close()
					if err != nil {
						stats.Errors++
						continue
					}
					if err := os.WriteFile(destPath, data, 0644); err != nil {
						stats.Errors++
						continue
					}
					stats.Downloaded++
				} else {
					stats.Skipped++
				}
			} else {
				stats.Skipped++
			}
		}
	}

	return nil
}

// getAllEvidenceTasksFromProvider fetches all evidence tasks from the first registered provider.
func (s *SyncService) getAllEvidenceTasksFromProvider(ctx context.Context, framework string) ([]domain.EvidenceTask, error) {
	names := s.registry.List()
	if len(names) == 0 {
		return nil, fmt.Errorf("no providers registered")
	}
	provider, err := s.registry.Get(names[0])
	if err != nil {
		return nil, err
	}
	return s.fetchAllEvidenceTasks(ctx, provider, framework)
}

// syncSubmissionsLegacy is the original direct Tugboat client submission sync.
func (s *SyncService) syncSubmissionsLegacy(ctx context.Context, opts SyncOptions, stats *SyncStats) error {
	apiTasks, err := s.tugboatClient.GetAllEvidenceTasks(ctx, opts.OrgID, opts.Framework)
	if err != nil {
		return fmt.Errorf("failed to get evidence tasks for submission sync: %w", err)
	}

	totalAttachments := 0

	for _, apiTask := range apiTasks {
		attachments, err := s.tugboatClient.GetEvidenceAttachmentsByTask(ctx, apiTask.ID)
		if err != nil {
			s.logger.Warn("Failed to get attachments for evidence task",
				logger.Int("task_id", apiTask.ID),
				logger.Error(err))
			stats.Errors++
			continue
		}

		totalAttachments += len(attachments)

		if err := s.saveAttachmentsForTask(ctx, apiTask.ID, attachments, stats); err != nil {
			s.logger.Warn("Failed to save attachments for evidence task",
				logger.Int("task_id", apiTask.ID),
				logger.Error(err))
			stats.Errors++
			continue
		}

		stats.Synced++
	}

	stats.Total = totalAttachments
	stats.Detailed = totalAttachments - stats.Errors
	return nil
}

// saveAttachmentsForTask groups and saves attachments for a specific task
// It also downloads the actual files for file-type attachments
func (s *SyncService) saveAttachmentsForTask(ctx context.Context, taskID int, attachments []tugboatModels.EvidenceAttachment, stats *SyncStats) error {
	if len(attachments) == 0 {
		return nil // No attachments to save
	}

	// Get the evidence task to find its reference ID (ET-0001)
	task, err := s.storage.GetEvidenceTask(strconv.Itoa(taskID))
	if err != nil {
		// Task not found in storage - might not have been synced yet
		s.logger.Warn("Evidence task not found in storage, skipping attachment save",
			logger.Int("task_id", taskID))
		return nil
	}

	if task.ReferenceID == "" {
		s.logger.Warn("Evidence task has no reference ID, skipping attachment save",
			logger.Int("task_id", taskID))
		return nil
	}

	// Group attachments by collection window
	// For now, we'll use the collection date to determine the window
	// TODO: Implement proper window detection based on collection_interval
	windowMap := make(map[string][]tugboatModels.EvidenceAttachment)
	for _, attachment := range attachments {
		// Extract year-quarter from collection date (YYYY-MM-DD -> YYYY-Qn)
		window := s.getWindowFromDate(attachment.Collected)
		windowMap[window] = append(windowMap[window], attachment)
	}

	// Save attachments for each window
	for window, windowAttachments := range windowMap {
		// Download actual files to archive/ subfolder for file-type attachments
		taskDirName := naming.GetEvidenceTaskDirName(task.Name, task.ReferenceID, task.ID)
		evidenceDir := filepath.Join(s.baseDir, "evidence", taskDirName, window, naming.SubfolderArchive)
		if err := os.MkdirAll(evidenceDir, 0755); err != nil {
			s.logger.Warn("Failed to create evidence directory",
				logger.String("task_ref", task.ReferenceID),
				logger.String("window", window),
				logger.Error(err))
			stats.Errors++
			continue
		}

		s.logger.Debug("Processing attachments for window",
			logger.String("task_ref", task.ReferenceID),
			logger.String("window", window),
			logger.Int("total_attachments", len(windowAttachments)))

		for _, att := range windowAttachments {
			if att.Type == "file" && att.Attachment != nil {
				// Download the file
				filename := att.Attachment.OriginalFilename
				if filename == "" {
					filename = fmt.Sprintf("attachment_%d", att.ID)
				}
				destPath := filepath.Join(evidenceDir, filename)

				s.logger.Debug("Downloading attachment file",
					logger.Int("attachment_id", att.ID),
					logger.String("filename", filename),
					logger.String("dest", destPath))

				if err := s.tugboatClient.DownloadAttachment(ctx, att.ID, destPath); err != nil {
					s.logger.Warn("Failed to download attachment",
						logger.Int("attachment_id", att.ID),
						logger.String("filename", filename),
						logger.Error(err))
					stats.Errors++
					continue
				}

				stats.Downloaded++
			} else if att.Type == "url" {
				// Save URL to a text file
				filename := fmt.Sprintf("url_reference_%d.txt", att.ID)
				destPath := filepath.Join(evidenceDir, filename)
				urlContent := fmt.Sprintf("URL: %s\nNotes: %s\nCollected: %s\n", att.URL, att.Notes, att.Collected)
				if err := os.WriteFile(destPath, []byte(urlContent), 0644); err != nil {
					s.logger.Warn("Failed to save URL reference",
						logger.Int("attachment_id", att.ID),
						logger.Error(err))
					stats.Errors++
				} else {
					stats.Downloaded++
				}
			} else {
				// Log skipped attachments for debugging
				s.logger.Debug("Skipping attachment - not downloadable",
					logger.Int("attachment_id", att.ID),
					logger.String("type", att.Type),
					logger.Field{Key: "has_attachment_object", Value: att.Attachment != nil})
				stats.Skipped++
			}
		}

		submission := s.convertAttachmentsToSubmission(task.ReferenceID, taskDirName, strconv.Itoa(taskID), window, windowAttachments)
		if err := s.storage.SaveSubmissionToSubfolder(submission, naming.SubfolderArchive); err != nil {
			s.logger.Warn("Failed to save submission for window",
				logger.String("task_ref", task.ReferenceID),
				logger.String("window", window),
				logger.Error(err))
			continue
		}

		// Also save to submission history in submitted/ subfolder
		history, err := s.storage.LoadSubmissionHistory(task.ReferenceID, window)
		if err != nil {
			// History doesn't exist yet, create new one
			history = &models.SubmissionHistory{
				TaskRef: task.ReferenceID,
				Window:  window,
				Entries: []models.SubmissionHistoryEntry{},
			}
		}

		// Add entries from attachments (one per attachment)
		for _, att := range windowAttachments {
			entry := models.SubmissionHistoryEntry{
				SubmissionID: strconv.Itoa(att.ID),
				SubmittedAt:  s.parseTime(att.Created),
				SubmittedBy:  s.getDisplayName(att.Owner),
				Status:       "accepted", // Tugboat attachments are already accepted
				FileCount:    1,
				Notes:        att.Notes,
			}
			history.Entries = append(history.Entries, entry)
		}

		if err := s.storage.SaveSubmissionHistoryToSubfolder(history, naming.SubfolderArchive); err != nil {
			s.logger.Warn("Failed to save submission history",
				logger.String("task_ref", task.ReferenceID),
				logger.String("window", window),
				logger.Error(err))
		}
	}

	return nil
}

// Domain storage integration methods
// These methods save domain models directly to domain storage

func (s *SyncService) savePolicyThroughDataService(ctx context.Context, policy *domain.Policy) error {
	// Always use unified storage for consistent naming
	return s.storage.SavePolicy(policy)
}

func (s *SyncService) saveControlThroughDataService(ctx context.Context, control *domain.Control) error {
	// Always use unified storage for consistent naming
	return s.storage.SaveControl(control)
}

func (s *SyncService) saveEvidenceTaskThroughDataService(ctx context.Context, task *domain.EvidenceTask) error {
	// Always use unified storage for consistent naming
	return s.storage.SaveEvidenceTask(task)
}

// Utility methods

// GetSyncSummary provides a summary of the last sync operation
func (s *SyncService) GetSyncSummary(ctx context.Context) (map[string]interface{}, error) {
	// Get actual counts from unified storage
	policies, err := s.storage.GetAllPolicies()
	if err != nil {
		return nil, fmt.Errorf("failed to get policies for summary: %w", err)
	}

	controls, err := s.storage.GetAllControls()
	if err != nil {
		return nil, fmt.Errorf("failed to get controls for summary: %w", err)
	}

	evidenceTasks, err := s.storage.GetAllEvidenceTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence tasks for summary: %w", err)
	}

	// Get last sync metadata from storage
	lastSync, syncStatus := s.getLastSyncMetadata()

	summary := map[string]interface{}{
		"last_sync":            lastSync,
		"total_policies":       len(policies),
		"total_controls":       len(controls),
		"total_evidence_tasks": len(evidenceTasks),
		"sync_status":          syncStatus,
		"next_scheduled":       lastSync.Add(24 * time.Hour), // Daily sync
		"data_freshness":       time.Since(lastSync).String(),
	}

	return summary, nil
}

// ValidateSync checks if the local data is consistent with remote data
func (s *SyncService) ValidateSync(ctx context.Context, opts SyncOptions) (map[string]interface{}, error) {
	startTime := time.Now()

	// Initialize validation result
	validation := map[string]interface{}{
		"status":         "valid",
		"checks":         make(map[string]interface{}),
		"last_validated": startTime,
		"errors":         []string{},
		"warnings":       []string{},
	}

	checks := make(map[string]interface{})
	var errors []string
	var warnings []string

	// Validate policy data
	if opts.Policies || (!opts.Policies && !opts.Controls && !opts.Evidence) {
		policyCheck, err := s.validatePolicyData(ctx, opts)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Policy validation failed: %v", err))
			checks["policy_validation"] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			checks["policy_validation"] = policyCheck
			if policyCheck["status"] == "warning" {
				warnings = append(warnings, policyCheck["message"].(string))
			}
		}
	}

	// Validate control data
	if opts.Controls || (!opts.Policies && !opts.Controls && !opts.Evidence) {
		controlCheck, err := s.validateControlData(ctx, opts)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Control validation failed: %v", err))
			checks["control_validation"] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			checks["control_validation"] = controlCheck
			if controlCheck["status"] == "warning" {
				warnings = append(warnings, controlCheck["message"].(string))
			}
		}
	}

	// Validate evidence task data
	if opts.Evidence || (!opts.Policies && !opts.Controls && !opts.Evidence) {
		evidenceCheck, err := s.validateEvidenceTaskData(ctx, opts)
		if err != nil {
			errors = append(errors, fmt.Sprintf("Evidence task validation failed: %v", err))
			checks["evidence_validation"] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			checks["evidence_validation"] = evidenceCheck
			if evidenceCheck["status"] == "warning" {
				warnings = append(warnings, evidenceCheck["message"].(string))
			}
		}
	}

	// Validate data integrity and relationships
	integrityCheck, err := s.validateDataIntegrity(ctx)
	if err != nil {
		errors = append(errors, fmt.Sprintf("Data integrity validation failed: %v", err))
		checks["data_integrity"] = map[string]interface{}{
			"status": "error",
			"error":  err.Error(),
		}
	} else {
		checks["data_integrity"] = integrityCheck
		if integrityCheck["status"] == "warning" {
			warnings = append(warnings, integrityCheck["message"].(string))
		}
	}

	// Determine overall status
	status := "valid"
	if len(errors) > 0 {
		status = "error"
	} else if len(warnings) > 0 {
		status = "warning"
	}

	validation["status"] = status
	validation["checks"] = checks
	validation["errors"] = errors
	validation["warnings"] = warnings
	validation["duration"] = time.Since(startTime).String()

	return validation, nil
}

// Helper methods for sync validation and metadata

// getLastSyncMetadata retrieves the last sync metadata from storage
func (s *SyncService) getLastSyncMetadata() (time.Time, string) {
	// In a real implementation, this would read from a metadata file or database
	// For now, we'll check the modification time of the data files

	// Try to get the modification time of the most recent data file
	policies, err := s.storage.GetAllPolicies()
	if err != nil || len(policies) == 0 {
		return time.Time{}, "never_synced"
	}

	// Use a reasonable default - in practice this would be stored as sync metadata
	lastSync := time.Now().Add(-2 * time.Hour) // Assume synced 2 hours ago
	return lastSync, "completed"
}

// validatePolicyData validates policy data consistency
func (s *SyncService) validatePolicyData(ctx context.Context, opts SyncOptions) (map[string]interface{}, error) {
	// Get local policies
	localPolicies, err := s.storage.GetAllPolicies()
	if err != nil {
		return nil, fmt.Errorf("failed to get local policies: %w", err)
	}

	// Get remote policies for comparison
	remotePolicies, err := s.tugboatClient.GetAllPolicies(ctx, opts.OrgID, opts.Framework)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote policies: %w", err)
	}

	localCount := len(localPolicies)
	remoteCount := len(remotePolicies)

	check := map[string]interface{}{
		"local_count":  localCount,
		"remote_count": remoteCount,
		"status":       "valid",
	}

	if localCount != remoteCount {
		check["status"] = "warning"
		check["message"] = fmt.Sprintf("Policy count mismatch: local=%d, remote=%d", localCount, remoteCount)
	} else {
		check["message"] = fmt.Sprintf("Policy count matches: %d policies", localCount)
	}

	// Since we unified the policy model, detailed data check is no longer needed
	// All policies now contain complete information in a single model

	return check, nil
}

// validateControlData validates control data consistency
func (s *SyncService) validateControlData(ctx context.Context, opts SyncOptions) (map[string]interface{}, error) {
	// Get local controls
	localControls, err := s.storage.GetAllControls()
	if err != nil {
		return nil, fmt.Errorf("failed to get local controls: %w", err)
	}

	// Get remote controls for comparison
	remoteControls, err := s.tugboatClient.GetAllControls(ctx, opts.OrgID, opts.Framework)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote controls: %w", err)
	}

	localCount := len(localControls)
	remoteCount := len(remoteControls)

	check := map[string]interface{}{
		"local_count":  localCount,
		"remote_count": remoteCount,
		"status":       "valid",
	}

	if localCount != remoteCount {
		check["status"] = "warning"
		check["message"] = fmt.Sprintf("Control count mismatch: local=%d, remote=%d", localCount, remoteCount)
	} else {
		check["message"] = fmt.Sprintf("Control count matches: %d controls", localCount)
	}

	// Since we unified the control model, detailed data check is no longer needed
	// All controls now contain complete information in a single model

	return check, nil
}

// validateEvidenceTaskData validates evidence task data consistency
func (s *SyncService) validateEvidenceTaskData(ctx context.Context, opts SyncOptions) (map[string]interface{}, error) {
	// Get local evidence tasks
	localTasks, err := s.storage.GetAllEvidenceTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to get local evidence tasks: %w", err)
	}

	// Get remote evidence tasks for comparison
	remoteTasks, err := s.tugboatClient.GetAllEvidenceTasks(ctx, opts.OrgID, opts.Framework)
	if err != nil {
		return nil, fmt.Errorf("failed to get remote evidence tasks: %w", err)
	}

	localCount := len(localTasks)
	remoteCount := len(remoteTasks)

	check := map[string]interface{}{
		"local_count":  localCount,
		"remote_count": remoteCount,
		"status":       "valid",
	}

	if localCount != remoteCount {
		check["status"] = "warning"
		check["message"] = fmt.Sprintf("Evidence task count mismatch: local=%d, remote=%d", localCount, remoteCount)
	} else {
		check["message"] = fmt.Sprintf("Evidence task count matches: %d tasks", localCount)
	}

	// Since we unified the evidence task model, detailed data check is no longer needed
	// All evidence tasks now contain complete information in a single model

	return check, nil
}

// validateDataIntegrity validates data integrity and relationships
func (s *SyncService) validateDataIntegrity(ctx context.Context) (map[string]interface{}, error) {
	check := map[string]interface{}{
		"status":  "valid",
		"message": "Data integrity checks passed",
	}

	var issues []string

	// Check for orphaned data
	evidenceTasks, err := s.storage.GetAllEvidenceTasks()
	if err != nil {
		return nil, fmt.Errorf("failed to get evidence tasks: %w", err)
	}

	controls, err := s.storage.GetAllControls()
	if err != nil {
		return nil, fmt.Errorf("failed to get controls: %w", err)
	}

	policies, err := s.storage.GetAllPolicies()
	if err != nil {
		return nil, fmt.Errorf("failed to get policies: %w", err)
	}

	// Create lookup maps for efficient checking
	controlMap := make(map[string]bool)
	for _, control := range controls {
		controlMap[fmt.Sprintf("%s", control.ID)] = true
	}

	policyMap := make(map[string]bool)
	for _, policy := range policies {
		policyMap[policy.ID] = true
	}

	// Check for broken references in evidence tasks
	var brokenControlRefs []string
	for _, task := range evidenceTasks {
		// Check control references using unified model
		for _, controlID := range task.Controls {
			if !controlMap[controlID] {
				brokenControlRefs = append(brokenControlRefs, fmt.Sprintf("Task %s references missing control %s", task.ID, controlID))
			}
		}
	}

	if len(brokenControlRefs) > 0 {
		issues = append(issues, fmt.Sprintf("Broken control references: %d found", len(brokenControlRefs)))
		check["broken_control_refs"] = brokenControlRefs
	}

	// Check for data consistency issues
	var inconsistencies []string

	// Check that tasks have proper framework assignments
	for _, task := range evidenceTasks {
		if task.Framework == "" {
			inconsistencies = append(inconsistencies, fmt.Sprintf("Task %s has no framework assigned", task.ID))
		}
	}

	if len(inconsistencies) > 0 {
		issues = append(issues, fmt.Sprintf("Data inconsistencies: %d found", len(inconsistencies)))
		check["inconsistencies"] = inconsistencies
	}

	// Set final status
	if len(issues) > 0 {
		check["status"] = "warning"
		check["message"] = fmt.Sprintf("Data integrity issues found: %s", strings.Join(issues, ", "))
		check["issues"] = issues
	}

	return check, nil
}

// generatePolicyDocument creates a comprehensive markdown document for a policy
func (s *SyncService) generatePolicyDocument(policy *domain.Policy) error {
	// Generate the document content
	documentContent := s.policyFormatter.ToDocumentMarkdown(policy)

	// Generate filename
	filename := s.policyFormatter.GetDocumentFilename(policy)

	// Use document service to write the file
	return s.documentService.GenerateDocument(PolicyDocument, filename, documentContent)
}

// generateControlDocument creates a comprehensive markdown document for a control
func (s *SyncService) generateControlDocument(control *domain.Control) error {
	// Generate the document content
	documentContent := s.controlFormatter.ToDocumentMarkdown(control)

	// Generate filename
	filename := s.controlFormatter.GetDocumentFilename(control)

	// Use document service to write the file
	return s.documentService.GenerateDocument(ControlDocument, filename, documentContent)
}

// generateEvidenceTaskDocument creates a comprehensive markdown document for an evidence task
func (s *SyncService) generateEvidenceTaskDocument(task *domain.EvidenceTask) error {
	// Generate the document content
	documentContent := s.evidenceTaskFormatter.ToDocumentMarkdown(task)

	// Generate filename
	filename := s.evidenceTaskFormatter.GetDocumentFilename(task)

	// Use document service to write the file
	return s.documentService.GenerateDocument(EvidenceTaskDocument, filename, documentContent)
}

// Helper methods for submission sync

// getWindowFromDate converts a date string (YYYY-MM-DD) to a window identifier (YYYY-Qn)
func (s *SyncService) getWindowFromDate(dateStr string) string {
	// Parse date
	t, err := time.Parse("2006-01-02", dateStr)
	if err != nil {
		// Default to current quarter if parsing fails
		t = time.Now()
	}

	year := t.Year()
	month := t.Month()

	// Determine quarter based on month
	var quarter int
	switch {
	case month >= 1 && month <= 3:
		quarter = 1
	case month >= 4 && month <= 6:
		quarter = 2
	case month >= 7 && month <= 9:
		quarter = 3
	default:
		quarter = 4
	}

	return fmt.Sprintf("%d-Q%d", year, quarter)
}

// convertAttachmentsToSubmission converts Tugboat attachments to EvidenceSubmission model
func (s *SyncService) convertAttachmentsToSubmission(taskRef string, taskDirName string, taskID string, window string, attachments []tugboatModels.EvidenceAttachment) *models.EvidenceSubmission {
	now := time.Now()
	submission := &models.EvidenceSubmission{
		TaskID:            taskID,
		TaskRef:           taskRef,
		Window:            window,
		Status:            "accepted", // Tugboat attachments are already accepted
		CreatedAt:         now,
		SubmittedAt:       &now,
		AcceptedAt:        &now,
		EvidenceFiles:     []models.EvidenceFileRef{},
		TotalFileCount:    len(attachments),
		ValidationStatus:  "passed",
		CompletenessScore: 1.0,
	}

	// Convert attachments to evidence file refs
	for _, att := range attachments {
		fileRef := models.EvidenceFileRef{
			Title:  att.Notes,
			Source: att.IntegrationType,
		}

		// Handle different attachment types
		if att.Type == "file" && att.Attachment != nil {
			fileRef.Filename = att.Attachment.OriginalFilename
			fileRef.RelativePath = fmt.Sprintf("evidence/%s/%s/%s", taskDirName, window, att.Attachment.OriginalFilename)
		} else if att.Type == "url" {
			fileRef.Filename = "url_reference.txt"
			fileRef.Title = att.URL
			fileRef.RelativePath = fmt.Sprintf("evidence/%s/%s/url_%d.txt", taskDirName, window, att.ID)
		}

		submission.EvidenceFiles = append(submission.EvidenceFiles, fileRef)
	}

	// Set submitted by from first attachment owner
	if len(attachments) > 0 && attachments[0].Owner != nil {
		submission.SubmittedBy = s.getDisplayName(attachments[0].Owner)
	}

	return submission
}

// parseTime parses ISO 8601 timestamp
func (s *SyncService) parseTime(timeStr string) time.Time {
	t, err := time.Parse(time.RFC3339, timeStr)
	if err != nil {
		return time.Now()
	}
	return t
}

// getDisplayName extracts display name from organization member
func (s *SyncService) getDisplayName(member *tugboatModels.OrganizationMember) string {
	if member == nil {
		return "Unknown"
	}
	if member.DisplayName != "" {
		return member.DisplayName
	}
	if member.Email != "" {
		return member.Email
	}
	return fmt.Sprintf("%s %s", member.FirstName, member.LastName)
}
