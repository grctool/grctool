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
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/storage"
)

// EvidenceRelationshipsTool provides evidence task relationship mapping capabilities
type EvidenceRelationshipsTool struct {
	config    *config.Config
	logger    logger.Logger
	dataStore interfaces.LocalDataStore
}

// NewEvidenceRelationshipsTool creates a new EvidenceRelationshipsTool
func NewEvidenceRelationshipsTool(cfg *config.Config, log logger.Logger) Tool {
	// Initialize local data store for offline-first operation
	// Apply defaults and resolve paths
	paths := cfg.Storage.Paths.WithDefaults().ResolveRelativeTo(cfg.Storage.DataDir)
	localDataStore, err := storage.NewLocalDataStore(cfg.Storage.DataDir, paths)
	if err != nil {
		log.Error("Failed to initialize local data store for evidence relationships tool",
			logger.Field{Key: "error", Value: err})
		return nil
	}

	return &EvidenceRelationshipsTool{
		config:    cfg,
		logger:    log,
		dataStore: localDataStore,
	}
}

// Name returns the tool name
func (ert *EvidenceRelationshipsTool) Name() string {
	return "evidence-relationships"
}

// Description returns the tool description
func (ert *EvidenceRelationshipsTool) Description() string {
	return "Maps relationships between evidence tasks, controls, and policies with configurable depth analysis"
}

// GetClaudeToolDefinition returns the tool definition for Claude
func (ert *EvidenceRelationshipsTool) GetClaudeToolDefinition() models.ClaudeTool {
	return models.ClaudeTool{
		Name:        ert.Name(),
		Description: ert.Description(),
		InputSchema: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"task_ref": map[string]interface{}{
					"type":        "string",
					"description": "Task reference in format ET-101, ET101, or numeric ID (e.g., 328001)",
					"examples":    []string{"ET-101", "ET101", "328001"},
				},
				"depth": map[string]interface{}{
					"type":        "integer",
					"description": "Analysis depth: 1=direct relationships, 2=include related controls/policies, 3=full dependency graph",
					"minimum":     1,
					"maximum":     3,
					"default":     2,
				},
				"include_policies": map[string]interface{}{
					"type":        "boolean",
					"description": "Include policy relationships in the analysis",
					"default":     true,
				},
				"include_controls": map[string]interface{}{
					"type":        "boolean",
					"description": "Include control relationships in the analysis",
					"default":     true,
				},
			},
			"required": []string{"task_ref"},
		},
	}
}

// Execute maps evidence task relationships
func (ert *EvidenceRelationshipsTool) Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error) {
	ert.logger.Debug("Executing evidence relationships mapping", logger.Field{Key: "params", Value: params})

	// Extract and validate parameters
	taskRefRaw, ok := params["task_ref"]
	if !ok {
		return "", nil, fmt.Errorf("task_ref parameter is required")
	}

	taskRef, ok := taskRefRaw.(string)
	if !ok {
		return "", nil, fmt.Errorf("task_ref must be a string")
	}

	// Parse depth parameter
	depth := 2 // default
	if depthRaw, exists := params["depth"]; exists {
		if depthFloat, ok := depthRaw.(float64); ok {
			depth = int(depthFloat)
		} else if depthInt, ok := depthRaw.(int); ok {
			depth = depthInt
		}
	}
	if depth < 1 || depth > 3 {
		depth = 2
	}

	// Parse boolean flags
	includePolicies := true
	if policyFlag, exists := params["include_policies"]; exists {
		if flag, ok := policyFlag.(bool); ok {
			includePolicies = flag
		}
	}

	includeControls := true
	if controlFlag, exists := params["include_controls"]; exists {
		if flag, ok := controlFlag.(bool); ok {
			includeControls = flag
		}
	}

	// Parse task reference to get numeric ID
	taskID, err := ert.parseTaskReference(taskRef)
	if err != nil {
		return "", nil, fmt.Errorf("invalid task reference '%s': %w", taskRef, err)
	}

	// Get task details from local data store
	task, err := ert.dataStore.GetEvidenceTask(strconv.Itoa(taskID))
	if err != nil {
		return "", nil, fmt.Errorf("failed to retrieve task %d: %w", taskID, err)
	}

	if task == nil {
		return "", nil, fmt.Errorf("task %d not found", taskID)
	}

	// Build relationship graph based on task and parameters
	graph, err := ert.buildRelationshipGraph(ctx, task, depth, includeControls, includePolicies)
	if err != nil {
		return "", nil, fmt.Errorf("failed to build relationship graph: %w", err)
	}

	// Convert response to JSON
	jsonData, err := json.MarshalIndent(graph, "", "  ")
	if err != nil {
		return "", nil, fmt.Errorf("failed to marshal response: %w", err)
	}

	// Create evidence source
	source := &models.EvidenceSource{
		Type:        "relationship_mapping",
		Resource:    fmt.Sprintf("Task %s relationship graph", task.ReferenceID),
		Content:     string(jsonData),
		Relevance:   ert.calculateRelevance(task, depth),
		ExtractedAt: task.UpdatedAt,
		Metadata: map[string]interface{}{
			"task_id":          task.ID,
			"reference_id":     task.ReferenceID,
			"framework":        task.Framework,
			"depth":            depth,
			"control_count":    len(task.Controls),
			"include_controls": includeControls,
			"include_policies": includePolicies,
		},
	}

	ert.logger.Info("Successfully mapped evidence task relationships",
		logger.Field{Key: "task_id", Value: taskID},
		logger.Field{Key: "reference_id", Value: task.ReferenceID},
		logger.Field{Key: "depth", Value: depth},
		logger.Field{Key: "control_count", Value: len(task.Controls)})

	return string(jsonData), source, nil
}

// buildRelationshipGraph builds the relationship graph based on task and parameters
func (ert *EvidenceRelationshipsTool) buildRelationshipGraph(ctx context.Context, task *domain.EvidenceTask, depth int, includeControls, includePolicies bool) (map[string]interface{}, error) {
	graph := map[string]interface{}{
		"task": map[string]interface{}{
			"id":           task.ID,
			"reference_id": task.ReferenceID,
			"name":         task.Name,
			"framework":    task.Framework,
			"status":       task.Status,
			"priority":     task.Priority,
		},
		"relationships": map[string]interface{}{
			"direct": map[string]interface{}{
				"controls": []map[string]interface{}{},
				"policies": []map[string]interface{}{},
			},
		},
		"summary": map[string]interface{}{
			"depth":          depth,
			"total_controls": len(task.Controls),
			"total_policies": 0, // We'll update this as we find related policies
		},
	}

	// Add direct control relationships from task
	if includeControls && len(task.Controls) > 0 {
		controls := make([]map[string]interface{}, 0, len(task.Controls))
		for _, controlRef := range task.Controls {
			// Try to get detailed control information from data store
			control, err := ert.dataStore.GetControl(controlRef)
			if err != nil {
				// If we can't get detailed info, use basic reference
				controlData := map[string]interface{}{
					"id":                controlRef,
					"relationship_type": "verifies",
				}
				controls = append(controls, controlData)
				continue
			}

			controlData := map[string]interface{}{
				"id":                control.ID,
				"reference_id":      control.ReferenceID,
				"name":              control.Name,
				"description":       control.Description,
				"category":          control.Category,
				"status":            control.Status,
				"framework":         control.Framework,
				"relationship_type": "verifies",
			}
			controls = append(controls, controlData)
		}
		graph["relationships"].(map[string]interface{})["direct"].(map[string]interface{})["controls"] = controls
	}

	// Add direct policy relationships from task
	if includePolicies && len(task.Policies) > 0 {
		policies := make([]map[string]interface{}, 0, len(task.Policies))
		policyCount := 0
		for _, policyRef := range task.Policies {
			// Try to get detailed policy information from data store
			policy, err := ert.dataStore.GetPolicy(policyRef)
			if err != nil {
				// If we can't get detailed info, use basic reference
				policyData := map[string]interface{}{
					"id":                policyRef,
					"relationship_type": "governs",
				}
				policies = append(policies, policyData)
				continue
			}

			policyData := map[string]interface{}{
				"id":                policy.ID,
				"reference_id":      policy.ReferenceID,
				"name":              policy.Name,
				"description":       policy.Description,
				"version":           policy.Version,
				"status":            policy.Status,
				"relationship_type": "governs",
			}
			policies = append(policies, policyData)
			policyCount++
		}
		graph["relationships"].(map[string]interface{})["direct"].(map[string]interface{})["policies"] = policies
		graph["summary"].(map[string]interface{})["total_policies"] = policyCount
	} else if includePolicies {
		graph["relationships"].(map[string]interface{})["direct"].(map[string]interface{})["policies"] = []map[string]interface{}{}
	}

	// Add extended relationships for depth > 1
	if depth > 1 {
		extended := ert.buildExtendedRelationships(ctx, task, depth, includeControls, includePolicies)
		graph["relationships"].(map[string]interface{})["extended"] = extended
	}

	// Add compliance context based on task framework
	graph["compliance_context"] = map[string]interface{}{
		"framework":   task.Framework,
		"status":      task.Status,
		"priority":    task.Priority,
		"description": task.Description,
	}

	// Add basic recommendations
	graph["recommendations"] = []string{
		"Review control relationships",
		"Validate evidence collection procedures",
		"Document evidence trail",
	}

	// Add suggested tools based on task requirements
	graph["suggested_tools"] = []string{
		"evidence-task-details",
		"prompt-assembler",
	}

	return graph, nil
}

// buildExtendedRelationships builds extended relationship mappings for deeper analysis
func (ert *EvidenceRelationshipsTool) buildExtendedRelationships(ctx context.Context, task *domain.EvidenceTask, depth int, includeControls, includePolicies bool) map[string]interface{} {
	extended := map[string]interface{}{
		"framework_mapping": map[string]interface{}{},
		"cross_references":  []map[string]interface{}{},
	}

	// Create basic framework mapping from task information
	frameworkMapping := make(map[string]map[string]interface{})

	framework := task.Framework
	if framework == "" {
		framework = "unspecified"
	}

	frameworkMapping[framework] = map[string]interface{}{
		"controls": []map[string]interface{}{},
		"policies": []map[string]interface{}{},
	}

	// Add basic control information if available
	if includeControls && len(task.Controls) > 0 {
		for _, controlRef := range task.Controls {
			controlData := map[string]interface{}{
				"id":       controlRef,
				"category": "verification",
			}

			controls := frameworkMapping[framework]["controls"].([]map[string]interface{})
			frameworkMapping[framework]["controls"] = append(controls, controlData)
		}
	}

	extended["framework_mapping"] = frameworkMapping

	// For depth 3, add basic dependency graph
	if depth >= 3 {
		extended["dependency_graph"] = ert.buildBasicDependencyGraph(task)
	}

	return extended
}

// buildBasicDependencyGraph builds a simplified dependency graph based on task data
func (ert *EvidenceRelationshipsTool) buildBasicDependencyGraph(task *domain.EvidenceTask) map[string]interface{} {
	graph := map[string]interface{}{
		"nodes": []map[string]interface{}{},
		"edges": []map[string]interface{}{},
	}

	// Add task node
	taskNode := map[string]interface{}{
		"id":    fmt.Sprintf("task_%d", task.ID),
		"label": task.Name,
		"type":  "evidence_task",
		"metadata": map[string]interface{}{
			"framework": task.Framework,
			"status":    task.Status,
		},
	}
	graph["nodes"] = append(graph["nodes"].([]map[string]interface{}), taskNode)

	// Add control nodes and edges for each control reference
	for _, controlRef := range task.Controls {
		controlNode := map[string]interface{}{
			"id":    fmt.Sprintf("control_%s", controlRef),
			"label": controlRef,
			"type":  "control",
			"metadata": map[string]interface{}{
				"framework": task.Framework,
			},
		}
		graph["nodes"] = append(graph["nodes"].([]map[string]interface{}), controlNode)

		// Add edge from task to control
		edge := map[string]interface{}{
			"from":  fmt.Sprintf("task_%d", task.ID),
			"to":    fmt.Sprintf("control_%s", controlRef),
			"type":  "verifies",
			"label": "verifies",
		}
		graph["edges"] = append(graph["edges"].([]map[string]interface{}), edge)
	}

	return graph
}

// calculateRelevance calculates the relevance score based on task depth and results
func (ert *EvidenceRelationshipsTool) calculateRelevance(task *domain.EvidenceTask, depth int) float64 {
	base := 0.7 // Base relevance

	// Increase relevance based on number of control relationships
	relationshipCount := len(task.Controls)
	if relationshipCount >= 10 {
		base += 0.2
	} else if relationshipCount >= 5 {
		base += 0.15
	} else if relationshipCount >= 1 {
		base += 0.1
	}

	// Increase relevance based on depth
	base += float64(depth-1) * 0.05

	// Cap at 1.0
	if base > 1.0 {
		base = 1.0
	}

	return base
}

// parseTaskReference converts various task reference formats to numeric ID
func (ert *EvidenceRelationshipsTool) parseTaskReference(taskRef string) (int, error) {
	// Create validator for task reference normalization
	validator := NewValidator(ert.config.Storage.DataDir)

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
func (ert *EvidenceRelationshipsTool) Category() string {
	return "evidence"
}

// Version returns the tool version
func (ert *EvidenceRelationshipsTool) Version() string {
	return "1.0.0"
}
