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
	"fmt"
	"strconv"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/tools/types"
)

// TypedEvidenceTaskDetailsTool provides evidence task details with typed requests/responses
type TypedEvidenceTaskDetailsTool struct {
	*EvidenceTaskDetailsTool // Embed the original tool for compatibility
}

// NewTypedEvidenceTaskDetailsTool creates a new TypedEvidenceTaskDetailsTool
func NewTypedEvidenceTaskDetailsTool(cfg *config.Config, log logger.Logger) *TypedEvidenceTaskDetailsTool {
	originalTool := NewEvidenceTaskDetailsTool(cfg, log)
	if originalTool == nil {
		return nil
	}

	return &TypedEvidenceTaskDetailsTool{
		EvidenceTaskDetailsTool: originalTool.(*EvidenceTaskDetailsTool),
	}
}

// ExecuteTyped runs the evidence task details tool with a typed request
func (ett *TypedEvidenceTaskDetailsTool) ExecuteTyped(ctx context.Context, req types.Request) (types.Response, error) {
	// Type assert to EvidenceTaskRequest
	taskReq, ok := req.(*types.EvidenceTaskRequest)
	if !ok {
		return types.NewErrorResponse(ett.Name(), "invalid request type, expected EvidenceTaskRequest", nil), nil
	}

	ett.logger.Debug("Executing evidence task details with typed request",
		logger.Field{Key: "task_ref", Value: taskReq.TaskRef})

	// Parse task reference to get numeric ID
	taskID, err := ett.parseTaskReference(taskReq.TaskRef)
	if err != nil {
		return types.NewErrorResponse(ett.Name(), fmt.Sprintf("invalid task reference '%s': %v", taskReq.TaskRef, err), nil), nil
	}

	// Get task details from local data store
	task, err := ett.dataStore.GetEvidenceTask(strconv.Itoa(taskID))
	if err != nil {
		return types.NewErrorResponse(ett.Name(), fmt.Sprintf("failed to retrieve task %d: %v", taskID, err), nil), nil
	}

	if task == nil {
		return types.NewErrorResponse(ett.Name(), fmt.Sprintf("task %d not found", taskID), nil), nil
	}

	// Get related controls and policies
	controls, err := ett.getRelatedControlsTyped(ctx, task)
	if err != nil {
		ett.logger.Warn("Failed to get related controls",
			logger.Field{Key: "task_id", Value: taskID},
			logger.Field{Key: "error", Value: err})
	}

	policies, err := ett.getRelatedPoliciesTyped(ctx, task)
	if err != nil {
		ett.logger.Warn("Failed to get related policies",
			logger.Field{Key: "task_id", Value: taskID},
			logger.Field{Key: "error", Value: err})
	}

	// Generate formatted output
	content, err := ett.formatTaskDetailsTyped(task, controls, policies)
	if err != nil {
		return types.NewErrorResponse(ett.Name(), fmt.Sprintf("failed to format task details: %v", err), nil), nil
	}

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "evidence_task",
		Resource:    fmt.Sprintf("Task %s", taskReq.TaskRef),
		Content:     content,
		Relevance:   1.0, // Task details are always fully relevant
		ExtractedAt: time.Now(),
		Metadata: map[string]interface{}{
			"task_id":   taskID,
			"task_ref":  taskReq.TaskRef,
			"task_name": task.Name,
			"controls":  len(controls),
			"policies":  len(policies),
		},
	}

	// Create typed response
	response := &types.EvidenceTaskResponse{
		ToolResponse: types.NewSuccessResponse(ett.Name(), content, source, map[string]interface{}{
			"request_type": "EvidenceTaskRequest",
		}),
		Task:     task,
		Controls: controls,
		Policies: policies,
		TaskRef:  taskReq.TaskRef,
	}

	return response, nil
}

// getRelatedControlsTyped retrieves controls related to the evidence task
func (ett *TypedEvidenceTaskDetailsTool) getRelatedControlsTyped(ctx context.Context, task *domain.EvidenceTask) ([]domain.Control, error) {
	// This would typically fetch controls from the data store
	// For now, return empty slice - implement based on your specific needs
	return []domain.Control{}, nil
}

// getRelatedPoliciesTyped retrieves policies related to the evidence task
func (ett *TypedEvidenceTaskDetailsTool) getRelatedPoliciesTyped(ctx context.Context, task *domain.EvidenceTask) ([]domain.Policy, error) {
	// This would typically fetch policies from the data store
	// For now, return empty slice - implement based on your specific needs
	return []domain.Policy{}, nil
}

// formatTaskDetailsTyped formats task details into a readable string
func (ett *TypedEvidenceTaskDetailsTool) formatTaskDetailsTyped(task *domain.EvidenceTask, controls []domain.Control, policies []domain.Policy) (string, error) {
	// Use the existing formatting logic from the embedded tool
	// This is a simplified approach - in practice you might want to refactor the formatting logic

	// Create legacy params to reuse existing formatting
	params := map[string]interface{}{
		"task_ref": task.ReferenceID,
	}

	// Call the legacy Execute method to get formatted output
	content, _, err := ett.Execute(context.Background(), params)
	if err != nil {
		return "", err
	}

	return content, nil
}

// Ensure TypedEvidenceTaskDetailsTool implements both Tool and types.Tool interfaces
var _ Tool = (*TypedEvidenceTaskDetailsTool)(nil)
var _ types.TypedTool = (*TypedEvidenceTaskDetailsTool)(nil)
