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
	"strconv"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/storage"
)

// DataServiceInterface defines the interface for data access operations
// This avoids circular imports with the services package
type DataServiceInterface interface {
	GetEvidenceTask(ctx context.Context, taskID int) (*domain.EvidenceTaskDetails, error)
	GetAllEvidenceTasks(ctx context.Context) ([]domain.EvidenceTask, error)
	GetEvidenceRecords(ctx context.Context, taskID int) ([]domain.EvidenceRecord, error)
	GetPolicy(ctx context.Context, policyID string) (*domain.Policy, error)
	GetControl(ctx context.Context, controlID string) (*domain.Control, error)
}

// SimpleDataService provides a simple implementation of DataServiceInterface
type SimpleDataService struct {
	storage *storage.Storage
}

func (s *SimpleDataService) GetEvidenceTask(ctx context.Context, taskID int) (*domain.EvidenceTaskDetails, error) {
	// Convert int to string for storage call
	taskIDStr := strconv.Itoa(taskID)
	basicTask, err := s.storage.GetEvidenceTask(taskIDStr)
	if err != nil {
		return nil, err
	}

	// EvidenceTaskDetails is just a type alias for EvidenceTask, so we can cast directly
	return basicTask, nil
}

func (s *SimpleDataService) GetAllEvidenceTasks(ctx context.Context) ([]domain.EvidenceTask, error) {
	return s.storage.GetAllEvidenceTasks()
}

func (s *SimpleDataService) GetEvidenceRecords(ctx context.Context, taskID int) ([]domain.EvidenceRecord, error) {
	// This would be implemented when evidence records are stored
	return []domain.EvidenceRecord{}, nil
}

func (s *SimpleDataService) GetPolicy(ctx context.Context, policyID string) (*domain.Policy, error) {
	// Use policyID directly since it's already a string
	policy, err := s.storage.GetPolicy(policyID)
	if err != nil {
		return nil, err
	}

	return policy, nil
}

func (s *SimpleDataService) GetControl(ctx context.Context, controlID string) (*domain.Control, error) {
	// Use controlID directly since it's already a string
	control, err := s.storage.GetControl(controlID)
	if err != nil {
		return nil, err
	}

	return control, nil
}

// Common utility functions

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// minFloat returns the minimum of two floats
func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// max returns the maximum of two floats
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

// maxFloat returns the maximum of two floats (alias for consistency)
func maxFloat(a, b float64) float64 {
	return max(a, b)
}
