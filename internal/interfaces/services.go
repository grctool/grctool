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

package interfaces

import (
	"time"

	"github.com/grctool/grctool/internal/domain"
)

// DataService provides access to policies, controls, and evidence tasks
type DataService interface {
	// Policy operations
	GetPolicy(id string) (*domain.Policy, error)
	GetPolicyByReferenceAndID(referenceID, numericID string) (*domain.Policy, error)
	GetAllPolicies() ([]domain.Policy, error)
	GetPolicySummary() (*domain.PolicySummary, error)

	// Control operations
	GetControl(id string) (*domain.Control, error)
	GetControlByReferenceAndID(referenceID, numericID string) (*domain.Control, error)
	GetAllControls() ([]domain.Control, error)
	GetControlSummary() (*domain.ControlSummary, error)

	// Evidence task operations
	GetEvidenceTask(id string) (*domain.EvidenceTask, error)
	GetEvidenceTaskByReferenceAndID(referenceID, numericID string) (*domain.EvidenceTask, error)
	GetAllEvidenceTasks() ([]domain.EvidenceTask, error)
	GetEvidenceTaskSummary() (*domain.EvidenceTaskSummary, error)

	// Stats and synchronization
	GetStats() (map[string]interface{}, error)
	SetSyncTime(syncType string, syncTime time.Time) error
	GetSyncTime(syncType string) (time.Time, error)
}

// EvidenceService provides evidence-related operations
type EvidenceService interface {
	// Evidence record operations
	SaveEvidenceRecord(record *domain.EvidenceRecord) error
	GetEvidenceRecord(id string) (*domain.EvidenceRecord, error)
	GetEvidenceRecordsByTaskID(taskID int) ([]domain.EvidenceRecord, error)
}

// SyncService provides synchronization operations with external systems
type SyncService interface {
	// Synchronization operations
	SyncPolicies() error
	SyncControls() error
	SyncEvidenceTasks() error
	SyncAll() error

	// Status operations
	GetSyncStatus() (map[string]interface{}, error)
	GetLastSyncTime(syncType string) (time.Time, error)
}

// ToolService provides tool execution capabilities for evidence collection
type ToolService interface {
	// Tool execution
	ExecuteTool(toolName string, params map[string]interface{}) (interface{}, error)
	ListAvailableTools() ([]string, error)
	GetToolDescription(toolName string) (string, error)
}

// ConfigService provides configuration management
type ConfigService interface {
	// Configuration operations
	GetDataDirectory() string
	GetCacheDirectory() string
	GetLocalDataDirectory() string
	IsOfflineModeEnabled() bool
	GetToolConfiguration(toolName string) (map[string]interface{}, error)
}
