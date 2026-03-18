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

package sync

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/grctool/grctool/internal/logger"
	"gopkg.in/yaml.v3"
)

// AuditEntry records a single sync operation for compliance audit trail.
type AuditEntry struct {
	Timestamp  time.Time `json:"timestamp" yaml:"timestamp"`
	Provider   string    `json:"provider" yaml:"provider"`
	Operation  string    `json:"operation" yaml:"operation"`           // "pull", "push", "conflict_resolved", "conflict_manual", "error"
	EntityType string    `json:"entity_type" yaml:"entity_type"`       // "policy", "control", "evidence_task"
	EntityID   string    `json:"entity_id" yaml:"entity_id"`
	BeforeHash string    `json:"before_hash,omitempty" yaml:"before_hash,omitempty"`
	AfterHash  string    `json:"after_hash,omitempty" yaml:"after_hash,omitempty"`
	Resolution string    `json:"resolution,omitempty" yaml:"resolution,omitempty"` // "local_wins", "remote_wins", etc.
	Actor      string    `json:"actor" yaml:"actor"`
	Details    string    `json:"details,omitempty" yaml:"details,omitempty"`
}

// AuditLog records all entries from a sync run.
type AuditLog struct {
	RunID     string       `json:"run_id" yaml:"run_id"`
	StartTime time.Time    `json:"start_time" yaml:"start_time"`
	EndTime   time.Time    `json:"end_time" yaml:"end_time"`
	Actor     string       `json:"actor" yaml:"actor"`
	Providers []string     `json:"providers" yaml:"providers"`
	Entries   []AuditEntry `json:"entries" yaml:"entries"`
	Summary   AuditSummary `json:"summary" yaml:"summary"`
}

// AuditSummary provides counts for the sync run.
type AuditSummary struct {
	TotalOperations   int `json:"total_operations" yaml:"total_operations"`
	Pulls             int `json:"pulls" yaml:"pulls"`
	Pushes            int `json:"pushes" yaml:"pushes"`
	ConflictsResolved int `json:"conflicts_resolved" yaml:"conflicts_resolved"`
	ConflictsManual   int `json:"conflicts_manual" yaml:"conflicts_manual"`
	Errors            int `json:"errors" yaml:"errors"`
}

// AuditTrail manages append-only audit logs for sync operations.
type AuditTrail struct {
	baseDir string
	logger  logger.Logger
}

// NewAuditTrail creates an audit trail writer.
func NewAuditTrail(baseDir string, log logger.Logger) *AuditTrail {
	return &AuditTrail{
		baseDir: baseDir,
		logger:  log,
	}
}

// NewAuditLog starts a new audit log for a sync run.
func (a *AuditTrail) NewAuditLog(actor string, providers []string) *AuditLog {
	now := time.Now()
	runID := now.Format("20060102-150405")
	return &AuditLog{
		RunID:     runID,
		StartTime: now,
		Actor:     actor,
		Providers: providers,
		Entries:   []AuditEntry{},
	}
}

// RecordPull records a pull operation.
func (l *AuditLog) RecordPull(provider, entityType, entityID, beforeHash, afterHash string) {
	l.Entries = append(l.Entries, AuditEntry{
		Timestamp:  time.Now(),
		Provider:   provider,
		Operation:  "pull",
		EntityType: entityType,
		EntityID:   entityID,
		BeforeHash: beforeHash,
		AfterHash:  afterHash,
		Actor:      l.Actor,
	})
}

// RecordPush records a push operation.
func (l *AuditLog) RecordPush(provider, entityType, entityID, beforeHash, afterHash string) {
	l.Entries = append(l.Entries, AuditEntry{
		Timestamp:  time.Now(),
		Provider:   provider,
		Operation:  "push",
		EntityType: entityType,
		EntityID:   entityID,
		BeforeHash: beforeHash,
		AfterHash:  afterHash,
		Actor:      l.Actor,
	})
}

// RecordConflictResolved records a resolved conflict.
func (l *AuditLog) RecordConflictResolved(provider, entityType, entityID, resolution, beforeHash, afterHash string) {
	l.Entries = append(l.Entries, AuditEntry{
		Timestamp:  time.Now(),
		Provider:   provider,
		Operation:  "conflict_resolved",
		EntityType: entityType,
		EntityID:   entityID,
		BeforeHash: beforeHash,
		AfterHash:  afterHash,
		Resolution: resolution,
		Actor:      l.Actor,
	})
}

// RecordConflictManual records an unresolved conflict requiring manual action.
func (l *AuditLog) RecordConflictManual(provider, entityType, entityID, localHash, remoteHash string) {
	l.Entries = append(l.Entries, AuditEntry{
		Timestamp:  time.Now(),
		Provider:   provider,
		Operation:  "conflict_manual",
		EntityType: entityType,
		EntityID:   entityID,
		BeforeHash: localHash,
		AfterHash:  remoteHash,
		Actor:      l.Actor,
	})
}

// RecordError records a sync error.
func (l *AuditLog) RecordError(provider, entityType, entityID, errorMsg string) {
	l.Entries = append(l.Entries, AuditEntry{
		Timestamp:  time.Now(),
		Provider:   provider,
		Operation:  "error",
		EntityType: entityType,
		EntityID:   entityID,
		Actor:      l.Actor,
		Details:    errorMsg,
	})
}

// Finalize sets the end time and computes summary.
func (l *AuditLog) Finalize() {
	l.EndTime = time.Now()
	l.Summary = AuditSummary{}
	for _, e := range l.Entries {
		l.Summary.TotalOperations++
		switch e.Operation {
		case "pull":
			l.Summary.Pulls++
		case "push":
			l.Summary.Pushes++
		case "conflict_resolved":
			l.Summary.ConflictsResolved++
		case "conflict_manual":
			l.Summary.ConflictsManual++
		case "error":
			l.Summary.Errors++
		}
	}
}

// logFilename returns the filename for a given run ID.
func logFilename(runID string) string {
	return fmt.Sprintf("sync-%s.yaml", runID)
}

// Save writes the audit log to disk as YAML.
// File: {baseDir}/sync-{run_id}.yaml
func (a *AuditTrail) Save(log *AuditLog) error {
	if err := os.MkdirAll(a.baseDir, 0o755); err != nil {
		return fmt.Errorf("create audit dir: %w", err)
	}

	data, err := yaml.Marshal(log)
	if err != nil {
		return fmt.Errorf("marshal audit log: %w", err)
	}

	path := filepath.Join(a.baseDir, logFilename(log.RunID))
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return fmt.Errorf("write audit log: %w", err)
	}

	a.logger.Info("audit log saved",
		logger.Field{Key: "run_id", Value: log.RunID},
		logger.Field{Key: "path", Value: path},
	)

	return nil
}

// List returns all audit log files, newest first.
func (a *AuditTrail) List() ([]string, error) {
	entries, err := os.ReadDir(a.baseDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read audit dir: %w", err)
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if len(name) > 5 && name[:5] == "sync-" && filepath.Ext(name) == ".yaml" {
			names = append(names, name)
		}
	}

	// Sort descending (newest first) — filenames contain timestamps so
	// lexicographic reverse order gives newest first.
	sort.Sort(sort.Reverse(sort.StringSlice(names)))

	return names, nil
}

// Load reads a specific audit log from disk.
func (a *AuditTrail) Load(runID string) (*AuditLog, error) {
	path := filepath.Join(a.baseDir, logFilename(runID))

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("audit log not found: %s", runID)
		}
		return nil, fmt.Errorf("read audit log: %w", err)
	}

	var log AuditLog
	if err := yaml.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("unmarshal audit log: %w", err)
	}

	return &log, nil
}
