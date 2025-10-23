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
	"strings"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/tugboat"
)

// EvidenceTaskListTool provides a list of evidence tasks with filtering capabilities
type EvidenceTaskListTool struct {
	config    *config.Config
	logger    logger.Logger
	dataStore interfaces.LocalDataStore
}

// NewEvidenceTaskListTool creates a new evidence task list tool
func NewEvidenceTaskListTool(cfg *config.Config, log logger.Logger) Tool {
	// Initialize local data store for offline-first operation
	// Apply defaults and resolve paths
	paths := cfg.Storage.Paths.WithDefaults().ResolveRelativeTo(cfg.Storage.DataDir)
	localDataStore, err := storage.NewLocalDataStore(cfg.Storage.DataDir, paths)
	if err != nil {
		log.Error("Failed to initialize local data store for evidence task list tool",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	return &EvidenceTaskListTool{
		config:    cfg,
		logger:    log,
		dataStore: localDataStore,
	}
}

// Name returns the tool name
func (e *EvidenceTaskListTool) Name() string {
	return "evidence-task-list"
}

// Description returns the tool description
func (e *EvidenceTaskListTool) Description() string {
	return "List evidence tasks with filtering capabilities for programmatic access"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (e *EvidenceTaskListTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        e.Name(),
		Description: "List evidence collection tasks with optional filtering by status, category, priority, framework, and other criteria. Returns structured JSON data suitable for automation.",
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"status": map[string]interface{}{
					"type":        "array",
					"description": "Filter by status (pending, completed, overdue)",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{"pending", "completed", "overdue"},
					},
				},
				"framework": map[string]interface{}{
					"type":        "string",
					"description": "Filter by framework (soc2, iso27001, etc)",
				},
				"priority": map[string]interface{}{
					"type":        "array",
					"description": "Filter by priority (high, medium, low)",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{"high", "medium", "low"},
					},
				},
				"category": map[string]interface{}{
					"type":        "array",
					"description": "Filter by category (Infrastructure, Personnel, Process, Compliance, Monitoring, Data)",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{"Infrastructure", "Personnel", "Process", "Compliance", "Monitoring", "Data"},
					},
				},
				"assignee": map[string]interface{}{
					"type":        "string",
					"description": "Filter by assignee name",
				},
				"overdue": map[string]interface{}{
					"type":        "boolean",
					"description": "Show only overdue tasks",
				},
				"due_soon": map[string]interface{}{
					"type":        "boolean",
					"description": "Show tasks due within 7 days",
				},
				"aec_status": map[string]interface{}{
					"type":        "array",
					"description": "Filter by AEC (Automated Evidence Collection) status",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{"enabled", "disabled", "na"},
					},
				},
				"collection_type": map[string]interface{}{
					"type":        "array",
					"description": "Filter by collection type",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{"Manual", "Automated", "Hybrid"},
					},
				},
				"sensitive": map[string]interface{}{
					"type":        "boolean",
					"description": "Show only sensitive data tasks",
				},
				"complexity": map[string]interface{}{
					"type":        "array",
					"description": "Filter by complexity level",
					"items": map[string]interface{}{
						"type": "string",
						"enum": []string{"Simple", "Moderate", "Complex"},
					},
				},
				"limit": map[string]interface{}{
					"type":        "integer",
					"description": "Maximum number of tasks to return (default: no limit)",
					"minimum":     1,
				},
			},
		},
	}
}

// Execute runs the evidence task list tool with the given parameters
func (e *EvidenceTaskListTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	e.logger.Debug("Executing evidence task list tool", logger.Field{Key: "params", Value: params})

	// Build filter from parameters
	filter := e.buildFilterFromParams(params)

	// Get all evidence tasks
	allTasks, err := e.dataStore.GetAllEvidenceTasks()
	if err != nil {
		return "", nil, fmt.Errorf("failed to get evidence tasks: %w", err)
	}

	// Apply filters
	filteredTasks := e.applyFilters(allTasks, filter, params)

	// Apply limit if specified
	if limit, ok := params["limit"].(float64); ok && limit > 0 {
		limitInt := int(limit)
		if len(filteredTasks) > limitInt {
			filteredTasks = filteredTasks[:limitInt]
		}
	}

	// Enrich tasks with Tugboat web URLs
	e.enrichTasksWithURLs(filteredTasks)

	// Prepare response data
	response := map[string]interface{}{
		"total_tasks":    len(allTasks),
		"filtered_tasks": len(filteredTasks),
		"filter_applied": filter,
		"tasks":          filteredTasks,
		"generated_at":   time.Now().Format(time.RFC3339),
	}

	// Convert to JSON
	jsonData, err := json.MarshalIndent(response, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "evidence_task_list",
		Resource:    "Evidence Task Registry",
		Content:     string(jsonData),
		Relevance:   1.0,
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"total_tasks":    len(allTasks),
			"filtered_tasks": len(filteredTasks),
			"filter":         filter,
		},
	}

	return string(jsonData), source, nil
}

// buildFilterFromParams converts tool parameters to domain filter
func (e *EvidenceTaskListTool) buildFilterFromParams(params map[string]interface{}) domain.EvidenceFilter {
	filter := domain.EvidenceFilter{}

	// Status filter
	if statusParam, ok := params["status"].([]interface{}); ok {
		for _, s := range statusParam {
			if statusStr, ok := s.(string); ok {
				filter.Status = append(filter.Status, statusStr)
			}
		}
	}

	// Framework filter
	if framework, ok := params["framework"].(string); ok && framework != "" {
		filter.Framework = framework
	}

	// Priority filter
	if priorityParam, ok := params["priority"].([]interface{}); ok {
		for _, p := range priorityParam {
			if priorityStr, ok := p.(string); ok {
				filter.Priority = append(filter.Priority, priorityStr)
			}
		}
	}

	// Category filter
	if categoryParam, ok := params["category"].([]interface{}); ok {
		for _, c := range categoryParam {
			if categoryStr, ok := c.(string); ok {
				filter.Category = append(filter.Category, categoryStr)
			}
		}
	}

	// Assignee filter
	if assignee, ok := params["assignee"].(string); ok && assignee != "" {
		filter.AssignedTo = assignee
	}

	// Note: Overdue and due_soon are computed filters, handled in matchesFilter function
	// They don't exist in the domain.EvidenceFilter struct

	// AEC status filter
	if aecStatusParam, ok := params["aec_status"].([]interface{}); ok {
		for _, a := range aecStatusParam {
			if aecStr, ok := a.(string); ok {
				filter.AecStatus = append(filter.AecStatus, aecStr)
			}
		}
	}

	// Collection type filter
	if collectionTypeParam, ok := params["collection_type"].([]interface{}); ok {
		for _, c := range collectionTypeParam {
			if collectionStr, ok := c.(string); ok {
				filter.CollectionType = append(filter.CollectionType, collectionStr)
			}
		}
	}

	// Note: Sensitive is computed filter, handled in matchesFilter function

	// Complexity filter
	if complexityParam, ok := params["complexity"].([]interface{}); ok {
		for _, c := range complexityParam {
			if complexityStr, ok := c.(string); ok {
				filter.ComplexityLevel = append(filter.ComplexityLevel, complexityStr)
			}
		}
	}

	return filter
}

// applyFilters applies the domain filter to the task list
func (e *EvidenceTaskListTool) applyFilters(tasks []domain.EvidenceTask, filter domain.EvidenceFilter, params map[string]interface{}) []domain.EvidenceTask {
	var filtered []domain.EvidenceTask

	for _, task := range tasks {
		if e.matchesFilter(task, filter, params) {
			filtered = append(filtered, task)
		}
	}

	return filtered
}

// matchesFilter checks if a task matches the given filter criteria
func (e *EvidenceTaskListTool) matchesFilter(task domain.EvidenceTask, filter domain.EvidenceFilter, params map[string]interface{}) bool {
	// Status filter
	if len(filter.Status) > 0 {
		taskStatus := e.getTaskStatus(task)
		if !e.stringInSlice(taskStatus, filter.Status) {
			return false
		}
	}

	// Framework filter
	if filter.Framework != "" {
		if !strings.EqualFold(task.Framework, filter.Framework) {
			return false
		}
	}

	// Priority filter
	if len(filter.Priority) > 0 {
		if !e.stringInSlice(task.Priority, filter.Priority) {
			return false
		}
	}

	// Category filter
	if len(filter.Category) > 0 {
		if !e.stringInSlice(task.Category, filter.Category) {
			return false
		}
	}

	// Assignee filter
	if filter.AssignedTo != "" {
		assigneeMatch := false
		for _, assignee := range task.Assignees {
			if strings.Contains(strings.ToLower(assignee.Name), strings.ToLower(filter.AssignedTo)) ||
				strings.Contains(strings.ToLower(assignee.Email), strings.ToLower(filter.AssignedTo)) {
				assigneeMatch = true
				break
			}
		}
		if !assigneeMatch {
			return false
		}
	}

	// Overdue filter (computed from params)
	if overdue, ok := params["overdue"].(bool); ok && overdue {
		if !e.isTaskOverdue(task) {
			return false
		}
	}

	// Due soon filter (computed from params)
	if dueSoon, ok := params["due_soon"].(bool); ok && dueSoon {
		if !e.isTaskDueSoon(task) {
			return false
		}
	}

	// AEC status filter
	if len(filter.AecStatus) > 0 {
		aecStatus := "na"
		if task.AecStatus != nil {
			aecStatus = strings.ToLower(task.AecStatus.Status)
		}
		if !e.stringInSlice(aecStatus, filter.AecStatus) {
			return false
		}
	}

	// Collection type filter
	if len(filter.CollectionType) > 0 {
		if !e.stringInSlice(task.CollectionType, filter.CollectionType) {
			return false
		}
	}

	// Sensitive filter (computed from params)
	if sensitive, ok := params["sensitive"].(bool); ok && sensitive {
		if !task.Sensitive {
			return false
		}
	}

	// Complexity filter
	if len(filter.ComplexityLevel) > 0 {
		if !e.stringInSlice(task.ComplexityLevel, filter.ComplexityLevel) {
			return false
		}
	}

	return true
}

// Helper functions

func (e *EvidenceTaskListTool) stringInSlice(str string, slice []string) bool {
	for _, s := range slice {
		if strings.EqualFold(str, s) {
			return true
		}
	}
	return false
}

func (e *EvidenceTaskListTool) getTaskStatus(task domain.EvidenceTask) string {
	if task.Completed {
		return "completed"
	}
	if e.isTaskOverdue(task) {
		return "overdue"
	}
	return "pending"
}

func (e *EvidenceTaskListTool) isTaskOverdue(task domain.EvidenceTask) bool {
	// This is a simplified check - in reality, you'd parse the due date properly
	// For now, we'll use a basic heuristic
	return strings.Contains(task.Status, "overdue") || strings.Contains(task.Status, "⚠️")
}

func (e *EvidenceTaskListTool) isTaskDueSoon(task domain.EvidenceTask) bool {
	// This is a simplified check - in reality, you'd parse the due date and check if it's within 7 days
	// For now, we'll return false since we don't have proper date parsing
	return false
}

// enrichTasksWithURLs adds Tugboat web UI URLs to each task
func (e *EvidenceTaskListTool) enrichTasksWithURLs(tasks []domain.EvidenceTask) {
	if e.config.Tugboat.BaseURL == "" {
		// If no base URL configured, skip URL generation
		return
	}

	for i := range tasks {
		if tasks[i].ID > 0 {
			// Use task's org ID if available, otherwise use config org ID
			orgID := tasks[i].OrgID
			if orgID == 0 {
				// Parse org ID from config string to int
				// Config stores it as string, we need int for URL builder
				fmt.Sscanf(e.config.Tugboat.OrgID, "%d", &orgID)
			}

			if orgID > 0 {
				tasks[i].TugboatURL = tugboat.BuildEvidenceTaskURL(e.config.Tugboat.BaseURL, orgID, tasks[i].ID)
			}
		}
	}
}
