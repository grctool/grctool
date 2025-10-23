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

package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/storage"
)

// EvidenceTaskDetailsTool provides task details retrieval capabilities
type EvidenceTaskDetailsTool struct {
	config    *config.Config
	logger    logger.Logger
	dataStore interfaces.LocalDataStore
}

// NewEvidenceTaskDetailsTool creates a new EvidenceTaskDetailsTool
func NewEvidenceTaskDetailsTool(cfg *config.Config, log logger.Logger) Tool {
	// Initialize local data store for offline-first operation
	// Apply defaults and resolve paths
	paths := cfg.Storage.Paths.WithDefaults().ResolveRelativeTo(cfg.Storage.DataDir)
	localDataStore, err := storage.NewLocalDataStore(cfg.Storage.DataDir, paths)
	if err != nil {
		log.Error("Failed to initialize local data store for evidence task details tool",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	return &EvidenceTaskDetailsTool{
		config:    cfg,
		logger:    log,
		dataStore: localDataStore,
	}
}

// Name returns the tool name
func (ett *EvidenceTaskDetailsTool) Name() string {
	return "evidence-task-details"
}

// Description returns the tool description
func (ett *EvidenceTaskDetailsTool) Description() string {
	return "Retrieves detailed information about evidence tasks including requirements, status, and metadata"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (ett *EvidenceTaskDetailsTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        ett.Name(),
		Description: ett.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_ref": map[string]interface{}{
					"type":        "string",
					"description": "Task reference in format ET-101, ET101, or numeric ID (e.g., 328001)",
					"examples":    []string{"ET-101", "ET101", "328001"},
				},
			},
			"required": []string{"task_ref"},
		},
	}
}

// Execute retrieves evidence task details
func (ett *EvidenceTaskDetailsTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	ett.logger.Debug("Executing evidence task details retrieval", logger.Field{Key: "params", Value: params})

	// Extract task reference
	taskRefRaw, ok := params["task_ref"]
	if !ok {
		return "", nil, fmt.Errorf("task_ref parameter is required")
	}

	taskRef, ok := taskRefRaw.(string)
	if !ok {
		return "", nil, fmt.Errorf("task_ref must be a string")
	}

	// Parse task reference to get numeric ID
	taskID, err := ett.parseTaskReference(taskRef)
	if err != nil {
		return "", nil, fmt.Errorf("invalid task reference '%s': %w", taskRef, err)
	}

	// Get task details from local data store
	task, err := ett.dataStore.GetEvidenceTask(strconv.Itoa(taskID))
	if err != nil {
		return "", nil, fmt.Errorf("failed to retrieve task %d: %w", taskID, err)
	}

	if task == nil {
		return "", nil, fmt.Errorf("task %d not found", taskID)
	}

	// Create detailed response
	response := map[string]interface{}{
		"task": map[string]interface{}{
			"id":                  task.ID,
			"reference_id":        task.ReferenceID,
			"name":                task.Name,
			"description":         task.Description,
			"guidance":            task.Guidance,
			"framework":           task.Framework,
			"status":              task.Status,
			"priority":            task.Priority,
			"collection_interval": task.CollectionInterval,
			"completed":           task.Completed,
			"last_collected":      task.LastCollected,
			"next_due":            task.NextDue,
			"due_days_before":     task.DueDaysBefore,
			"ad_hoc":              task.AdHoc,
			"sensitive":           task.Sensitive,
			"created_at":          task.CreatedAt,
			"updated_at":          task.UpdatedAt,
		},
		"requirements": map[string]interface{}{
			"collection_type":  task.GetCollectionType(),
			"complexity_level": task.GetComplexityLevel(),
			"category":         task.GetCategory(),
			"aec_status":       task.GetAecStatusDisplay(),
		},
		"relationships": map[string]interface{}{
			"controls":      task.Controls,
			"control_count": len(task.Controls),
		},
		"metadata": map[string]interface{}{
			"org_scope": nil,
			"assignees": []map[string]interface{}{},
		},
	}

	// Add org scope if available
	if task.OrgScope != nil {
		response["metadata"].(map[string]interface{})["org_scope"] = map[string]interface{}{
			"id":          task.OrgScope.ID,
			"name":        task.OrgScope.Name,
			"type":        task.OrgScope.Type,
			"description": task.OrgScope.Description,
		}
	}

	// Add assignees if available
	if len(task.Assignees) > 0 {
		assignees := make([]map[string]interface{}, len(task.Assignees))
		for i, assignee := range task.Assignees {
			assignees[i] = map[string]interface{}{
				"id":    assignee.ID,
				"name":  assignee.Name,
				"email": assignee.Email,
				"role":  assignee.Role,
			}
		}
		response["metadata"].(map[string]interface{})["assignees"] = assignees
	}

	// Add AEC status details if available
	if task.AecStatus != nil {
		aecDetails := map[string]interface{}{
			"id":     task.AecStatus.ID,
			"status": task.AecStatus.Status,
		}

		if task.AecStatus.LastExecuted != nil {
			aecDetails["last_executed"] = task.AecStatus.LastExecuted
		}

		if task.AecStatus.NextScheduled != nil {
			aecDetails["next_scheduled"] = task.AecStatus.NextScheduled
		}

		if task.AecStatus.ErrorMessage != "" {
			aecDetails["error_message"] = task.AecStatus.ErrorMessage
		}

		response["metadata"].(map[string]interface{})["aec_details"] = aecDetails
	}

	// Convert response to JSON
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "evidence_task",
		Resource:    fmt.Sprintf("Task %s (%d)", task.ReferenceID, task.ID),
		Content:     string(jsonData),
		Relevance:   1.0, // Always highly relevant for task-specific queries
		ExtractedAt: task.UpdatedAt,
		Metadata: map[string]interface{}{
			"task_id":       task.ID,
			"reference_id":  task.ReferenceID,
			"framework":     task.Framework,
			"status":        task.Status,
			"control_count": len(task.Controls),
		},
	}

	ett.logger.Info("Successfully retrieved evidence task details",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "reference_id", Value: task.ReferenceID},
		logger.Field{Key: "framework", Value: task.Framework})

	return string(jsonData), source, nil
}

// parseTaskReference converts various task reference formats to numeric ID
func (ett *EvidenceTaskDetailsTool) parseTaskReference(taskRef string) (int, error) {
	// Create validator for task reference normalization
	validator := NewValidator(ett.config.Storage.DataDir)

	result, err := validator.ValidateTaskReference(taskRef)
	if err != nil {
		return 0, err
	}

	if !result.Valid {
		return 0, fmt.Errorf("invalid task reference format")
	}

	// Get normalized value
	normalizedRef := result.Normalized["task_ref"]
	if normalizedRef == "" {
		normalizedRef = taskRef
	}

	// Convert to integer
	taskID, err := strconv.Atoi(normalizedRef)
	if err != nil {
		return 0, fmt.Errorf("failed to convert task reference to numeric ID: %w", err)
	}

	return taskID, nil
}

// Category returns the tool category
func (ett *EvidenceTaskDetailsTool) Category() string {
	return "evidence"
}

// Version returns the tool version
func (ett *EvidenceTaskDetailsTool) Version() string {
	return "1.0.0"
}
