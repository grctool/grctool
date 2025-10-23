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

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/storage"
)

// DataServiceImpl implements the DataService interface using the unified storage layer
// This service operates directly on domain models without conversion
type DataServiceImpl struct {
	storage *storage.Storage
}

// NewDataService creates a new data service implementation using unified storage
func NewDataService(storage *storage.Storage) DataService {
	return &DataServiceImpl{
		storage: storage,
	}
}

// Evidence task operations

func (d *DataServiceImpl) GetEvidenceTask(ctx context.Context, taskID int) (*domain.EvidenceTask, error) {
	// Get directly from unified storage - no conversion needed
	return d.storage.GetEvidenceTask(strconv.Itoa(taskID))
}

func (d *DataServiceImpl) GetAllEvidenceTasks(ctx context.Context) ([]domain.EvidenceTask, error) {
	// Get directly from unified storage - no conversion needed
	return d.storage.GetAllEvidenceTasks()
}

func (d *DataServiceImpl) FilterEvidenceTasks(ctx context.Context, filter domain.EvidenceFilter) ([]domain.EvidenceTask, error) {
	// Get all tasks first (in a real implementation, we'd push filtering to storage)
	allTasks, err := d.GetAllEvidenceTasks(ctx)
	if err != nil {
		return nil, err
	}

	// Apply domain-level filtering
	var filteredTasks []domain.EvidenceTask
	for _, task := range allTasks {
		if d.matchesEvidenceFilter(task, filter) {
			filteredTasks = append(filteredTasks, task)
		}
	}

	return filteredTasks, nil
}

// Control operations

func (d *DataServiceImpl) GetControl(ctx context.Context, controlID string) (*domain.Control, error) {
	// Get directly from unified storage - no conversion needed
	return d.storage.GetControl(controlID)
}

func (d *DataServiceImpl) GetAllControls(ctx context.Context) ([]domain.Control, error) {
	// Get directly from unified storage - no conversion needed
	return d.storage.GetAllControls()
}

// Policy operations

func (d *DataServiceImpl) GetPolicy(ctx context.Context, policyID string) (*domain.Policy, error) {
	// Get directly from unified storage - no conversion needed
	return d.storage.GetPolicy(policyID)
}

func (d *DataServiceImpl) GetAllPolicies(ctx context.Context) ([]domain.Policy, error) {
	// Get directly from unified storage - no conversion needed
	return d.storage.GetAllPolicies()
}

// Relationship operations

func (d *DataServiceImpl) GetRelationships(ctx context.Context, sourceType, sourceID string) ([]domain.Relationship, error) {
	// For now, generate relationships based on stored data
	// In a real implementation, we might store relationships explicitly
	var relationships []domain.Relationship

	switch sourceType {
	case "evidence_task":
		taskID, err := strconv.Atoi(sourceID)
		if err != nil {
			return nil, fmt.Errorf("invalid evidence task ID: %s", sourceID)
		}

		// Get task to find related controls
		task, err := d.GetEvidenceTask(ctx, taskID)
		if err != nil {
			return nil, err
		}

		// Create relationships to controls
		for _, controlID := range task.Controls {
			relationships = append(relationships, domain.Relationship{
				SourceType: "evidence_task",
				SourceID:   sourceID,
				TargetType: "control",
				TargetID:   controlID,
				Type:       "verifies",
			})
		}

		// Find related policies by framework
		policies, err := d.GetAllPolicies(ctx)
		if err != nil {
			return nil, err
		}

		for _, policy := range policies {
			if policy.Framework == task.Framework {
				relationships = append(relationships, domain.Relationship{
					SourceType: "evidence_task",
					SourceID:   sourceID,
					TargetType: "policy",
					TargetID:   policy.ID,
					Type:       "implements",
				})
			}
		}
	}

	return relationships, nil
}

// Evidence record operations

func (d *DataServiceImpl) SaveEvidenceRecord(ctx context.Context, record *domain.EvidenceRecord) error {
	// Save directly to unified storage - no conversion needed
	return d.storage.SaveEvidenceRecord(record)
}

func (d *DataServiceImpl) GetEvidenceRecords(ctx context.Context, taskID int) ([]domain.EvidenceRecord, error) {
	// Get directly from unified storage - no conversion needed
	return d.storage.GetEvidenceRecordsByTaskID(taskID)
}

// Helper methods

func (d *DataServiceImpl) matchesEvidenceFilter(task domain.EvidenceTask, filter domain.EvidenceFilter) bool {
	// Status filter
	if len(filter.Status) > 0 {
		statusMatch := false
		for _, status := range filter.Status {
			if task.Status == status {
				statusMatch = true
				break
			}
		}
		if !statusMatch {
			return false
		}
	}

	// Priority filter
	if len(filter.Priority) > 0 {
		priorityMatch := false
		for _, priority := range filter.Priority {
			if task.Priority == priority {
				priorityMatch = true
				break
			}
		}
		if !priorityMatch {
			return false
		}
	}

	// Framework filter
	if filter.Framework != "" && task.Framework != filter.Framework {
		return false
	}

	// Date filters
	if filter.DueBefore != nil && task.NextDue != nil && task.NextDue.After(*filter.DueBefore) {
		return false
	}

	if filter.DueAfter != nil && task.NextDue != nil && task.NextDue.Before(*filter.DueAfter) {
		return false
	}

	// Category filter
	if len(filter.Category) > 0 {
		taskCategory := task.GetCategory()
		categoryMatch := false
		for _, category := range filter.Category {
			if taskCategory == category {
				categoryMatch = true
				break
			}
		}
		if !categoryMatch {
			return false
		}
	}

	// AEC Status filter
	if len(filter.AecStatus) > 0 {
		aecStatusMatch := false
		taskAecStatus := ""
		if task.AecStatus != nil {
			taskAecStatus = task.AecStatus.Status
		} else {
			taskAecStatus = "na"
		}

		for _, aecStatus := range filter.AecStatus {
			if taskAecStatus == aecStatus {
				aecStatusMatch = true
				break
			}
		}
		if !aecStatusMatch {
			return false
		}
	}

	// Collection Type filter
	if len(filter.CollectionType) > 0 {
		taskCollectionType := task.GetCollectionType()
		collectionTypeMatch := false
		for _, collectionType := range filter.CollectionType {
			if taskCollectionType == collectionType {
				collectionTypeMatch = true
				break
			}
		}
		if !collectionTypeMatch {
			return false
		}
	}

	// Sensitive filter
	if filter.Sensitive != nil && task.Sensitive != *filter.Sensitive {
		return false
	}

	// Complexity Level filter
	if len(filter.ComplexityLevel) > 0 {
		taskComplexity := task.GetComplexityLevel()
		complexityMatch := false
		for _, complexity := range filter.ComplexityLevel {
			if taskComplexity == complexity {
				complexityMatch = true
				break
			}
		}
		if !complexityMatch {
			return false
		}
	}

	return true
}
