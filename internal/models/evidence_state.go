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

package models

import "time"

// EvidenceTaskState represents the complete state for one evidence task
// including both remote (Tugboat) status and local evidence status
type EvidenceTaskState struct {
	// Task identification
	TaskRef  string `json:"task_ref" yaml:"task_ref"`   // ET-0001
	TaskID   int    `json:"task_id" yaml:"task_id"`     // Numeric ID
	TaskName string `json:"task_name" yaml:"task_name"` // Human-readable name

	// Tugboat sync state (remote)
	TugboatStatus    string    `json:"tugboat_status" yaml:"tugboat_status"`         // pending, in_progress, completed
	TugboatCompleted bool      `json:"tugboat_completed" yaml:"tugboat_completed"`   // Completion flag from Tugboat
	LastSyncedAt     time.Time `json:"last_synced_at" yaml:"last_synced_at"`         // Last sync from Tugboat API
	Framework        string    `json:"framework,omitempty" yaml:"framework,omitempty"` // SOC2, ISO27001, etc.

	// Local evidence state
	LocalState LocalEvidenceState    `json:"local_state" yaml:"local_state"` // Overall local state
	Windows    map[string]WindowState `json:"windows" yaml:"windows"`         // Evidence by window (2025-Q4 → state)

	// Automation capability
	AutomationLevel AutomationCapability `json:"automation_level" yaml:"automation_level"`   // Level of automation available
	ApplicableTools []string             `json:"applicable_tools" yaml:"applicable_tools"`   // Tools that can generate this evidence

	// Timestamps
	LastGeneratedAt *time.Time `json:"last_generated_at,omitempty" yaml:"last_generated_at,omitempty"` // Most recent generation
	LastSubmittedAt *time.Time `json:"last_submitted_at,omitempty" yaml:"last_submitted_at,omitempty"` // Most recent submission
	LastScannedAt   time.Time  `json:"last_scanned_at" yaml:"last_scanned_at"`                         // When this state was computed
}

// WindowState represents evidence state for a specific collection window
type WindowState struct {
	// Window identification
	Window string `json:"window" yaml:"window"` // "2025-Q4", "2025", "2025-10", etc.

	// File inventory
	FileCount  int                `json:"file_count" yaml:"file_count"`     // Number of evidence files
	TotalBytes int64              `json:"total_bytes" yaml:"total_bytes"`   // Total size in bytes
	OldestFile *time.Time         `json:"oldest_file,omitempty" yaml:"oldest_file,omitempty"` // Earliest file timestamp
	NewestFile *time.Time         `json:"newest_file,omitempty" yaml:"newest_file,omitempty"` // Latest file timestamp
	Files      []FileState        `json:"files,omitempty" yaml:"files,omitempty"`             // Individual file details

	// Generation metadata (from .generation/metadata.yaml)
	HasGenerationMeta bool       `json:"has_generation_meta" yaml:"has_generation_meta"` // .generation/metadata.yaml exists
	GenerationMethod  string     `json:"generation_method,omitempty" yaml:"generation_method,omitempty"` // tool_coordination, ai_generation, manual
	GeneratedAt       *time.Time `json:"generated_at,omitempty" yaml:"generated_at,omitempty"`           // When evidence was generated
	GeneratedBy       string     `json:"generated_by,omitempty" yaml:"generated_by,omitempty"`           // Who/what generated (claude-code-assisted, grctool-cli, manual)
	ToolsUsed         []string   `json:"tools_used,omitempty" yaml:"tools_used,omitempty"`               // Tools used to generate evidence

	// Submission metadata (from .submission/submission.yaml)
	HasSubmissionMeta bool       `json:"has_submission_meta" yaml:"has_submission_meta"` // .submission/submission.yaml exists
	SubmissionStatus  string     `json:"submission_status,omitempty" yaml:"submission_status,omitempty"` // draft, validated, submitted, accepted, rejected
	SubmittedAt       *time.Time `json:"submitted_at,omitempty" yaml:"submitted_at,omitempty"`           // When evidence was submitted
	SubmissionID      string     `json:"submission_id,omitempty" yaml:"submission_id,omitempty"`         // Tugboat submission ID
}

// FileState represents metadata for a single evidence file
type FileState struct {
	Filename    string    `json:"filename" yaml:"filename"`         // Just the filename, not full path
	SizeBytes   int64     `json:"size_bytes" yaml:"size_bytes"`     // File size
	Checksum    string    `json:"checksum,omitempty" yaml:"checksum,omitempty"`     // SHA256 checksum if available
	ModifiedAt  time.Time `json:"modified_at" yaml:"modified_at"`   // File modification timestamp
	IsGenerated bool      `json:"is_generated" yaml:"is_generated"` // From tool vs manually added
}

// LocalEvidenceState represents the overall state of local evidence for a task
type LocalEvidenceState string

const (
	// StateNoEvidence indicates no evidence files exist locally
	StateNoEvidence LocalEvidenceState = "no_evidence"

	// StateGenerated indicates evidence has been generated but not validated or submitted
	StateGenerated LocalEvidenceState = "generated"

	// StateValidated indicates evidence has been validated and is ready for submission
	StateValidated LocalEvidenceState = "validated"

	// StateSubmitted indicates evidence has been submitted to Tugboat
	StateSubmitted LocalEvidenceState = "submitted"

	// StateAccepted indicates evidence has been accepted by Tugboat/auditors
	StateAccepted LocalEvidenceState = "accepted"

	// StateRejected indicates evidence was rejected and needs rework
	StateRejected LocalEvidenceState = "rejected"
)

// String returns the string representation of LocalEvidenceState
func (s LocalEvidenceState) String() string {
	return string(s)
}

// AutomationCapability represents the level of automation available for an evidence task
type AutomationCapability string

const (
	// AutomationFully indicates the task can be fully automated with available tools
	AutomationFully AutomationCapability = "fully_automated"

	// AutomationPartially indicates some aspects can be automated but manual work is needed
	AutomationPartially AutomationCapability = "partially_automated"

	// AutomationManual indicates the task requires manual evidence collection
	AutomationManual AutomationCapability = "manual_only"

	// AutomationUnknown indicates automation capability has not been determined
	AutomationUnknown AutomationCapability = "unknown"
)

// String returns the string representation of AutomationCapability
func (a AutomationCapability) String() string {
	return string(a)
}

// DetermineLocalState determines the overall local state based on window states
// This is a helper function to aggregate state across multiple windows
func DetermineLocalState(windows map[string]WindowState) LocalEvidenceState {
	if len(windows) == 0 {
		return StateNoEvidence
	}

	// Check for most advanced state across all windows
	hasAccepted := false
	hasSubmitted := false
	hasValidated := false
	hasGenerated := false

	for _, window := range windows {
		if window.SubmissionStatus == "accepted" {
			hasAccepted = true
		} else if window.SubmissionStatus == "submitted" {
			hasSubmitted = true
		} else if window.SubmissionStatus == "validated" {
			hasValidated = true
		} else if window.FileCount > 0 {
			hasGenerated = true
		}
	}

	// Return most advanced state found
	if hasAccepted {
		return StateAccepted
	}
	if hasSubmitted {
		return StateSubmitted
	}
	if hasValidated {
		return StateValidated
	}
	if hasGenerated {
		return StateGenerated
	}

	return StateNoEvidence
}

// DetermineAutomationCapability determines automation level based on applicable tools and task characteristics
func DetermineAutomationCapability(applicableTools []string, taskDescription string) AutomationCapability {
	if len(applicableTools) == 0 {
		return AutomationUnknown
	}

	// If we have multiple tools, likely fully automated
	if len(applicableTools) >= 2 {
		return AutomationFully
	}

	// Single tool might be partial automation
	// This is a simplified heuristic - could be enhanced based on tool capabilities
	return AutomationPartially
}

// StateCache represents the cached state data structure
// This is saved to .state/evidence_state.yaml for performance
type StateCache struct {
	LastScan time.Time                       `json:"last_scan" yaml:"last_scan"` // When the cache was last updated
	Tasks    map[string]*EvidenceTaskState   `json:"tasks" yaml:"tasks"`         // Task ref → state
}

// NewStateCache creates a new empty state cache
func NewStateCache() *StateCache {
	return &StateCache{
		LastScan: time.Now(),
		Tasks:    make(map[string]*EvidenceTaskState),
	}
}

// GetTask retrieves state for a specific task
func (sc *StateCache) GetTask(taskRef string) (*EvidenceTaskState, bool) {
	state, exists := sc.Tasks[taskRef]
	return state, exists
}

// SetTask updates or adds state for a task
func (sc *StateCache) SetTask(taskRef string, state *EvidenceTaskState) {
	sc.Tasks[taskRef] = state
	sc.LastScan = time.Now()
}

// GetTasksByState returns all tasks in a given local state
func (sc *StateCache) GetTasksByState(targetState LocalEvidenceState) []*EvidenceTaskState {
	var tasks []*EvidenceTaskState
	for _, task := range sc.Tasks {
		if task.LocalState == targetState {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// GetTasksByAutomation returns all tasks with a given automation level
func (sc *StateCache) GetTasksByAutomation(level AutomationCapability) []*EvidenceTaskState {
	var tasks []*EvidenceTaskState
	for _, task := range sc.Tasks {
		if task.AutomationLevel == level {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

// GetStateSummary returns counts by state
func (sc *StateCache) GetStateSummary() map[LocalEvidenceState]int {
	summary := make(map[LocalEvidenceState]int)
	for _, task := range sc.Tasks {
		summary[task.LocalState]++
	}
	return summary
}

// GetAutomationSummary returns counts by automation level
func (sc *StateCache) GetAutomationSummary() map[AutomationCapability]int {
	summary := make(map[AutomationCapability]int)
	for _, task := range sc.Tasks {
		summary[task.AutomationLevel]++
	}
	return summary
}
