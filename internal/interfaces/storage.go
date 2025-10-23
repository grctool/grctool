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

// StorageService provides storage operations for the application data
type StorageService interface {
	// Policy operations
	SavePolicy(policy *domain.Policy) error
	GetPolicy(id string) (*domain.Policy, error)
	GetPolicyByReferenceAndID(referenceID, numericID string) (*domain.Policy, error)
	GetAllPolicies() ([]domain.Policy, error)
	GetPolicySummary() (*domain.PolicySummary, error)

	// Control operations
	SaveControl(control *domain.Control) error
	GetControl(id string) (*domain.Control, error)
	GetControlByReferenceAndID(referenceID, numericID string) (*domain.Control, error)
	GetAllControls() ([]domain.Control, error)
	GetControlSummary() (*domain.ControlSummary, error)

	// Evidence task operations
	SaveEvidenceTask(task *domain.EvidenceTask) error
	GetEvidenceTask(id string) (*domain.EvidenceTask, error)
	GetEvidenceTaskByReferenceAndID(referenceID, numericID string) (*domain.EvidenceTask, error)
	GetAllEvidenceTasks() ([]domain.EvidenceTask, error)
	GetEvidenceTaskSummary() (*domain.EvidenceTaskSummary, error)

	// Evidence record operations
	SaveEvidenceRecord(record *domain.EvidenceRecord) error
	GetEvidenceRecord(id string) (*domain.EvidenceRecord, error)
	GetEvidenceRecordsByTaskID(taskID int) ([]domain.EvidenceRecord, error)

	// Statistics and metadata
	GetStats() (map[string]interface{}, error)
	SetSyncTime(syncType string, syncTime time.Time) error
	GetSyncTime(syncType string) (time.Time, error)

	// Utility operations
	Clear() error
}

// CacheService provides caching capabilities for performance optimization
type CacheService interface {
	// Generic cache operations
	Set(key string, value interface{}, expiration time.Duration) error
	Get(key string, target interface{}) error
	Delete(key string) error
	Clear() error
	Exists(key string) bool

	// Specialized cache operations
	SetToolResult(toolName string, params map[string]interface{}, result interface{}, expiration time.Duration) error
	GetToolResult(toolName string, params map[string]interface{}, target interface{}) error

	SetSummary(summaryType, id string, summary interface{}, expiration time.Duration) error
	GetSummary(summaryType, id string, target interface{}) error

	// Cache management
	GetSize() int64
	GetStats() map[string]interface{}
}

// FileService provides low-level file operations
type FileService interface {
	// Basic file operations
	Save(category, id string, data interface{}) error
	Load(category, id string, target interface{}) error
	Exists(category, id string) bool
	Delete(category, id string) error
	List(category string) ([]string, error)

	// Directory operations
	CreateDirectory(path string) error
	DeleteDirectory(path string) error
	ListDirectories(basePath string) ([]string, error)

	// Utility operations
	GetFullPath(category, id string) string
	GetSize() (int64, error)
}

// LocalDataStore provides offline-first data access without external dependencies
type LocalDataStore interface {
	StorageService

	// Offline-specific operations
	IsDataAvailable() bool
	GetDataSources() []string
	ValidateDataIntegrity() error

	// Fallback chain operations
	SetFallbackEnabled(enabled bool)
	IsFallbackEnabled() bool

	// Local data management
	ImportData(sourcePath string) error
	ExportData(targetPath string) error
	GetLastDataUpdate() (time.Time, error)
}
