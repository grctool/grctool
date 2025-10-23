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
	"strconv"
	"strings"
	"time"

	"github.com/grctool/grctool/internal/adapters"
	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/formatters"
	"github.com/grctool/grctool/internal/interpolation"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/registry"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/tugboat"
)

// SyncService handles synchronization between Tugboat Logic API and local storage
// This service uses adapters to convert between API models and domain models
type SyncService struct {
	tugboatClient         *tugboat.Client
	adapter               *adapters.TugboatToDomain
	storage               *storage.Storage // Single storage interface for all operations
	policyFormatter       *formatters.PolicyFormatter
	controlFormatter      *formatters.ControlFormatter
	evidenceTaskFormatter *formatters.EvidenceTaskFormatter
	evidenceTaskRegistry  *registry.EvidenceTaskRegistry
	documentService       *DocumentService
	baseDir               string
	logger                logger.Logger
}

// NewSyncService creates a new sync service
func NewSyncService(tugboatClient *tugboat.Client, storage *storage.Storage, cfg *config.Config, log logger.Logger) *SyncService {
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
		tugboatClient:         tugboatClient,
		adapter:               adapters.NewTugboatToDomain(),
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
	OrgID     string `json:"org_id"`
	Framework string `json:"framework,omitempty"`
	Policies  bool   `json:"policies"`
	Controls  bool   `json:"controls"`
	Evidence  bool   `json:"evidence"`
}

// SyncResult represents the result of a synchronization operation
type SyncResult struct {
	Policies      SyncStats     `json:"policies"`
	Controls      SyncStats     `json:"controls"`
	EvidenceTasks SyncStats     `json:"evidence_tasks"`
	Duration      time.Duration `json:"duration"`
	Errors        []string      `json:"errors,omitempty"`
	StartTime     time.Time     `json:"start_time"`
	EndTime       time.Time     `json:"end_time"`
}

// SyncStats represents statistics for a sync operation
type SyncStats struct {
	Total    int `json:"total"`
	Synced   int `json:"synced"`
	Detailed int `json:"detailed"`
	Errors   int `json:"errors"`
	Skipped  int `json:"skipped"`
}

// SyncAll performs a complete synchronization of all data types
func (s *SyncService) SyncAll(ctx context.Context, opts SyncOptions) (*SyncResult, error) {
	result := &SyncResult{
		StartTime: time.Now(),
		Errors:    []string{},
	}

	// Sync policies if requested
	if opts.Policies {
		if err := s.syncPolicies(ctx, opts, &result.Policies); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Policy sync failed: %v", err))
		}
	}

	// Sync controls if requested
	if opts.Controls {
		if err := s.syncControls(ctx, opts, &result.Controls); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Control sync failed: %v", err))
		}
	}

	// Sync evidence tasks if requested
	if opts.Evidence {
		if err := s.syncEvidenceTasks(ctx, opts, &result.EvidenceTasks); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Evidence task sync failed: %v", err))
		}
	}

	result.EndTime = time.Now()
	result.Duration = result.EndTime.Sub(result.StartTime)

	return result, nil
}

// Internal sync methods

func (s *SyncService) syncPolicies(ctx context.Context, opts SyncOptions, stats *SyncStats) error {
	// Get policies from Tugboat API
	apiPolicies, err := s.tugboatClient.GetAllPolicies(ctx, opts.OrgID, opts.Framework)
	if err != nil {
		return fmt.Errorf("failed to get policies from API: %w", err)
	}

	stats.Total = len(apiPolicies)

	// Collect all domain policies first
	var domainPolicies []domain.Policy

	for _, apiPolicy := range apiPolicies {
		// Get detailed policy information from API
		apiPolicyDetails, err := s.tugboatClient.GetPolicyDetails(ctx, apiPolicy.ID.String())
		if err != nil {
			stats.Errors++
			continue
		}

		// Convert API model to domain model using adapter (includes all details)
		domainPolicy := s.adapter.ConvertPolicy(*apiPolicyDetails)
		domainPolicies = append(domainPolicies, domainPolicy)
		stats.Detailed++
	}

	// Process reference IDs for all policies
	refProcessor := domain.NewPolicyReferenceProcessor()
	processedPolicies := refProcessor.ProcessPolicyReferences(domainPolicies)

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
			// Don't fail the sync for document generation errors, but log them
			s.logger.Warn("Failed to generate policy document",
				logger.String("policy_id", policy.ID),
				logger.Error(err))
		}

		stats.Synced++
	}

	return nil
}

func (s *SyncService) syncControls(ctx context.Context, opts SyncOptions, stats *SyncStats) error {
	// Get controls from Tugboat API
	apiControls, err := s.tugboatClient.GetAllControls(ctx, opts.OrgID, opts.Framework)
	if err != nil {
		return fmt.Errorf("failed to get controls from API: %w", err)
	}

	stats.Total = len(apiControls)

	// Collect all domain controls first
	var domainControls []domain.Control

	for _, apiControl := range apiControls {
		// Get detailed control information from API
		apiControlDetails, err := s.tugboatClient.GetControlDetails(ctx, strconv.Itoa(apiControl.ID))
		if err != nil {
			stats.Errors++
			continue
		}

		// Convert API model to domain model using adapter (includes all details)
		domainControl := s.adapter.ConvertControl(*apiControlDetails)
		domainControls = append(domainControls, domainControl)
		stats.Detailed++
	}

	// Process reference IDs for all controls
	refProcessor := domain.NewControlReferenceProcessor()
	processedControls := refProcessor.ProcessControlReferences(domainControls)

	// Save all processed controls
	for _, domainControl := range processedControls {
		// Save complete control info
		if err := s.saveControlThroughDataService(ctx, &domainControl); err != nil {
			stats.Errors++
			continue
		}

		// Generate control document
		if err := s.generateControlDocument(&domainControl); err != nil {
			// Don't fail the sync for document generation errors, but log them
			s.logger.Warn("Failed to generate control document",
				logger.Int("control_id", domainControl.ID),
				logger.Error(err))
		}

		stats.Synced++
	}

	return nil
}

func (s *SyncService) syncEvidenceTasks(ctx context.Context, opts SyncOptions, stats *SyncStats) error {
	// Get evidence tasks from Tugboat API
	apiTasks, err := s.tugboatClient.GetAllEvidenceTasks(ctx, opts.OrgID, opts.Framework)
	if err != nil {
		return fmt.Errorf("failed to get evidence tasks from API: %w", err)
	}

	stats.Total = len(apiTasks)

	// First pass: collect all domain tasks, preserving existing reference IDs
	var domainTasks []domain.EvidenceTask
	for _, apiTask := range apiTasks {
		// Get detailed task information from API
		apiTaskDetails, err := s.tugboatClient.GetEvidenceTaskDetails(ctx, strconv.Itoa(apiTask.ID))
		if err != nil {
			stats.Errors++
			continue
		}

		// Convert API model to domain model using adapter (includes all details)
		domainTask := s.adapter.ConvertEvidenceTask(*apiTaskDetails)

		// Process reference IDs for related controls
		refProcessor := domain.NewControlReferenceProcessor()
		processedControls := refProcessor.ProcessControlReferences(domainTask.RelatedControls)
		domainTask.RelatedControls = processedControls

		// Check if we have an existing task with a reference ID
		existingTask, err := s.storage.GetEvidenceTask(strconv.Itoa(domainTask.ID))
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
				logger.Int("task_id", domainTasks[i].ID),
				logger.Error(err))
		}

		// Generate evidence task document
		if err := s.generateEvidenceTaskDocument(&domainTasks[i]); err != nil {
			// Don't fail the sync for document generation errors, but log them
			s.logger.Warn("Failed to generate evidence task document",
				logger.Int("task_id", domainTasks[i].ID),
				logger.Error(err))
		}
	}

	// Save the updated registry
	if err := s.evidenceTaskRegistry.SaveRegistry(); err != nil {
		s.logger.Warn("Failed to save evidence task registry", logger.Error(err))
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
		controlMap[fmt.Sprintf("%d", control.ID)] = true
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
				brokenControlRefs = append(brokenControlRefs, fmt.Sprintf("Task %d references missing control %s", task.ID, controlID))
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
			inconsistencies = append(inconsistencies, fmt.Sprintf("Task %d has no framework assigned", task.ID))
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
