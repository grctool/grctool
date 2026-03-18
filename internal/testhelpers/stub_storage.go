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

package testhelpers

import (
	"fmt"
	"sync"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
)

// Compile-time assertion that StubStorageService implements StorageService.
var _ interfaces.StorageService = (*StubStorageService)(nil)

// StubStorageService is an in-memory implementation of the StorageService
// interface for use in tests. It stores all data in maps and requires no
// filesystem or external dependencies.
type StubStorageService struct {
	mu        sync.RWMutex
	policies  map[string]*domain.Policy
	controls  map[string]*domain.Control
	tasks     map[string]*domain.EvidenceTask
	records   map[string]*domain.EvidenceRecord
	syncTimes map[string]time.Time
}

// NewStubStorageService creates a StubStorageService with empty maps.
func NewStubStorageService() *StubStorageService {
	return &StubStorageService{
		policies:  make(map[string]*domain.Policy),
		controls:  make(map[string]*domain.Control),
		tasks:     make(map[string]*domain.EvidenceTask),
		records:   make(map[string]*domain.EvidenceRecord),
		syncTimes: make(map[string]time.Time),
	}
}

// NewStubStorageServiceWithData creates a StubStorageService pre-populated
// with the supplied slices. Policies are keyed by ID, controls by
// strconv.Itoa(ID), and tasks by strconv.Itoa(ID).
func NewStubStorageServiceWithData(
	policies []domain.Policy,
	controls []domain.Control,
	tasks []domain.EvidenceTask,
) *StubStorageService {
	s := NewStubStorageService()
	for i := range policies {
		p := policies[i]
		s.policies[p.ID] = &p
	}
	for i := range controls {
		c := controls[i]
		s.controls[c.ID] = &c
	}
	for i := range tasks {
		t := tasks[i]
		s.tasks[t.ID] = &t
	}
	return s
}

// ---------------------------------------------------------------------------
// Policy operations
// ---------------------------------------------------------------------------

func (s *StubStorageService) SavePolicy(policy *domain.Policy) error {
	if policy == nil {
		return fmt.Errorf("policy is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies[policy.ID] = policy
	return nil
}

func (s *StubStorageService) GetPolicy(id string) (*domain.Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	p, ok := s.policies[id]
	if !ok {
		return nil, fmt.Errorf("policy not found: %s", id)
	}
	return p, nil
}

func (s *StubStorageService) GetPolicyByReferenceAndID(referenceID, numericID string) (*domain.Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.policies {
		if p.ReferenceID == referenceID || p.ID == numericID {
			return p, nil
		}
	}
	return nil, fmt.Errorf("policy not found: ref=%s id=%s", referenceID, numericID)
}

func (s *StubStorageService) GetAllPolicies() ([]domain.Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Policy, 0, len(s.policies))
	for _, p := range s.policies {
		result = append(result, *p)
	}
	return result, nil
}

func (s *StubStorageService) GetPolicySummary() (*domain.PolicySummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	summary := &domain.PolicySummary{
		Total:       len(s.policies),
		ByFramework: make(map[string]int),
		ByStatus:    make(map[string]int),
	}
	for _, p := range s.policies {
		summary.ByFramework[p.Framework]++
		summary.ByStatus[p.Status]++
	}
	return summary, nil
}

// ---------------------------------------------------------------------------
// GetByExternalID lookups
// ---------------------------------------------------------------------------

func (s *StubStorageService) GetPolicyByExternalID(provider, externalID string) (*domain.Policy, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, p := range s.policies {
		if p.ExternalIDs != nil && p.ExternalIDs[provider] == externalID {
			return p, nil
		}
	}
	return nil, fmt.Errorf("policy not found for provider %q with external ID %q", provider, externalID)
}

func (s *StubStorageService) GetControlByExternalID(provider, externalID string) (*domain.Control, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.controls {
		if c.ExternalIDs != nil && c.ExternalIDs[provider] == externalID {
			return c, nil
		}
	}
	return nil, fmt.Errorf("control not found for provider %q with external ID %q", provider, externalID)
}

func (s *StubStorageService) GetEvidenceTaskByExternalID(provider, externalID string) (*domain.EvidenceTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.tasks {
		if t.ExternalIDs != nil && t.ExternalIDs[provider] == externalID {
			return t, nil
		}
	}
	return nil, fmt.Errorf("evidence task not found for provider %q with external ID %q", provider, externalID)
}

// ---------------------------------------------------------------------------
// Control operations
// ---------------------------------------------------------------------------

func (s *StubStorageService) SaveControl(control *domain.Control) error {
	if control == nil {
		return fmt.Errorf("control is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.controls[control.ID] = control
	return nil
}

func (s *StubStorageService) GetControl(id string) (*domain.Control, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	c, ok := s.controls[id]
	if !ok {
		return nil, fmt.Errorf("control not found: %s", id)
	}
	return c, nil
}

func (s *StubStorageService) GetControlByReferenceAndID(referenceID, numericID string) (*domain.Control, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, c := range s.controls {
		if c.ReferenceID == referenceID || c.ID == numericID {
			return c, nil
		}
	}
	return nil, fmt.Errorf("control not found: ref=%s id=%s", referenceID, numericID)
}

func (s *StubStorageService) GetAllControls() ([]domain.Control, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.Control, 0, len(s.controls))
	for _, c := range s.controls {
		result = append(result, *c)
	}
	return result, nil
}

func (s *StubStorageService) GetControlSummary() (*domain.ControlSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	summary := &domain.ControlSummary{
		Total:       len(s.controls),
		ByFramework: make(map[string]int),
		ByStatus:    make(map[string]int),
		ByCategory:  make(map[string]int),
	}
	for _, c := range s.controls {
		summary.ByFramework[c.Framework]++
		summary.ByStatus[c.Status]++
		summary.ByCategory[c.Category]++
	}
	return summary, nil
}

// ---------------------------------------------------------------------------
// Evidence task operations
// ---------------------------------------------------------------------------

func (s *StubStorageService) SaveEvidenceTask(task *domain.EvidenceTask) error {
	if task == nil {
		return fmt.Errorf("evidence task is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.tasks[task.ID] = task
	return nil
}

func (s *StubStorageService) GetEvidenceTask(id string) (*domain.EvidenceTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.tasks[id]
	if !ok {
		return nil, fmt.Errorf("evidence task not found: %s", id)
	}
	return t, nil
}

func (s *StubStorageService) GetEvidenceTaskByReferenceAndID(referenceID, numericID string) (*domain.EvidenceTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, t := range s.tasks {
		if t.ReferenceID == referenceID || t.ID == numericID {
			return t, nil
		}
	}
	return nil, fmt.Errorf("evidence task not found: ref=%s id=%s", referenceID, numericID)
}

func (s *StubStorageService) GetAllEvidenceTasks() ([]domain.EvidenceTask, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]domain.EvidenceTask, 0, len(s.tasks))
	for _, t := range s.tasks {
		result = append(result, *t)
	}
	return result, nil
}

func (s *StubStorageService) GetEvidenceTaskSummary() (*domain.EvidenceTaskSummary, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	summary := &domain.EvidenceTaskSummary{
		Total:      len(s.tasks),
		ByStatus:   make(map[string]int),
		ByPriority: make(map[string]int),
	}
	for _, t := range s.tasks {
		summary.ByStatus[t.Status]++
		summary.ByPriority[t.Priority]++
	}
	return summary, nil
}

// ---------------------------------------------------------------------------
// Evidence record operations
// ---------------------------------------------------------------------------

func (s *StubStorageService) SaveEvidenceRecord(record *domain.EvidenceRecord) error {
	if record == nil {
		return fmt.Errorf("evidence record is nil")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[record.ID] = record
	return nil
}

func (s *StubStorageService) GetEvidenceRecord(id string) (*domain.EvidenceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	r, ok := s.records[id]
	if !ok {
		return nil, fmt.Errorf("evidence record not found: %s", id)
	}
	return r, nil
}

func (s *StubStorageService) GetEvidenceRecordsByTaskID(taskID string) ([]domain.EvidenceRecord, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var result []domain.EvidenceRecord
	for _, r := range s.records {
		if r.TaskID == taskID {
			result = append(result, *r)
		}
	}
	return result, nil
}

// ---------------------------------------------------------------------------
// Statistics and metadata
// ---------------------------------------------------------------------------

func (s *StubStorageService) GetStats() (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return map[string]interface{}{
		"policies":         len(s.policies),
		"controls":         len(s.controls),
		"evidence_tasks":   len(s.tasks),
		"evidence_records": len(s.records),
	}, nil
}

func (s *StubStorageService) SetSyncTime(syncType string, syncTime time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.syncTimes[syncType] = syncTime
	return nil
}

func (s *StubStorageService) GetSyncTime(syncType string) (time.Time, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.syncTimes[syncType]
	if !ok {
		return time.Time{}, fmt.Errorf("sync time not found: %s", syncType)
	}
	return t, nil
}

// ---------------------------------------------------------------------------
// Utility operations
// ---------------------------------------------------------------------------

func (s *StubStorageService) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.policies = make(map[string]*domain.Policy)
	s.controls = make(map[string]*domain.Control)
	s.tasks = make(map[string]*domain.EvidenceTask)
	s.records = make(map[string]*domain.EvidenceRecord)
	s.syncTimes = make(map[string]time.Time)
	return nil
}
