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

package domain

import (
	"strings"
	"time"
)

// EvidenceTask represents a comprehensive evidence collection task in the domain model
type EvidenceTask struct {
	// Basic fields
	ID                 int        `json:"id"`
	ReferenceID        string     `json:"reference_id,omitempty"` // ET1, ET2, etc.
	Name               string     `json:"name"`
	Description        string     `json:"description"`
	Guidance           string     `json:"guidance"`
	CollectionInterval string     `json:"collection_interval"`
	Priority           string     `json:"priority"`
	Framework          string     `json:"framework"`
	Status             string     `json:"status"`
	Completed          bool       `json:"completed"`
	Category           string     `json:"category,omitempty"`         // Infrastructure, Process, Personnel, etc.
	ComplexityLevel    string     `json:"complexity_level,omitempty"` // Simple, Moderate, Complex
	CollectionType     string     `json:"collection_type,omitempty"`  // Manual, Automated, Hybrid
	LastCollected      *time.Time `json:"last_collected,omitempty"`
	NextDue            *time.Time `json:"next_due,omitempty"`
	DueDaysBefore      int        `json:"due_days_before"`
	AdHoc              bool       `json:"ad_hoc"`
	Sensitive          bool       `json:"sensitive"`
	CreatedAt          time.Time  `json:"created_at"`
	UpdatedAt          time.Time  `json:"updated_at"`
	// Web UI link
	TugboatURL string `json:"tugboat_url,omitempty"` // Web UI URL for this evidence task
	// API metadata
	MasterVersionNum int    `json:"master_version_num"`
	MasterEvidenceID int    `json:"master_evidence_id"`
	OrgID            int    `json:"org_id"`
	OrgScopeID       int    `json:"org_scope_id"`
	DeprecationNotes string `json:"deprecation_notes,omitempty"`
	// Relationships
	Controls              []string         `json:"controls,omitempty"`         // Control IDs this task relates to
	RelatedControls       []Control        `json:"related_controls,omitempty"` // Full control details when available
	Policies              []string         `json:"policies,omitempty"`         // Policy IDs this task relates to
	RelatedPolicies       []Policy         `json:"related_policies,omitempty"` // Full policy details when available
	Assignees             []Person         `json:"assignees,omitempty"`
	Tags                  []Tag            `json:"tags,omitempty"`
	AuditProjects         []AuditProject   `json:"audit_projects,omitempty"`
	JiraIssues            []JiraIssue      `json:"jira_issues,omitempty"`
	FrameworkCodes        []FrameworkCode  `json:"framework_codes,omitempty"`
	OrgScope              *OrgScope        `json:"org_scope,omitempty"`
	LastRemindedAt        *time.Time       `json:"last_reminded_at,omitempty"`
	SupportedIntegrations []Integration    `json:"supported_integrations,omitempty"`
	AecStatus             *AecStatus       `json:"aec_status,omitempty"`
	SubtaskMetadata       *SubtaskMetadata `json:"subtask_metadata,omitempty"`
	OpenIncidentCount     int              `json:"open_incident_count"`
	// Usage statistics (following control/policy pattern)
	ViewCount        int        `json:"view_count"`
	LastViewedAt     *time.Time `json:"last_viewed_at,omitempty"`
	DownloadCount    int        `json:"download_count"`
	LastDownloadedAt *time.Time `json:"last_downloaded_at,omitempty"`
	ReferenceCount   int        `json:"reference_count"`
	LastReferencedAt *time.Time `json:"last_referenced_at,omitempty"`
	// Master content and associations
	MasterContent *EvidenceTaskMasterContent `json:"master_content,omitempty"`
	Associations  *EvidenceTaskAssociations  `json:"associations,omitempty"`
}

// EvidenceTaskSummary represents an aggregated view of evidence tasks
type EvidenceTaskSummary struct {
	Total      int            `json:"total"`
	ByStatus   map[string]int `json:"by_status"`
	ByPriority map[string]int `json:"by_priority"`
	Overdue    int            `json:"overdue"`
	DueSoon    int            `json:"due_soon"`
	LastSync   time.Time      `json:"last_sync"`
}

// EvidenceFilter represents filters for evidence task queries
type EvidenceFilter struct {
	Status          []string   `json:"status,omitempty"`
	Priority        []string   `json:"priority,omitempty"`
	Framework       string     `json:"framework,omitempty"`
	AssignedTo      string     `json:"assigned_to,omitempty"`
	DueBefore       *time.Time `json:"due_before,omitempty"`
	DueAfter        *time.Time `json:"due_after,omitempty"`
	Category        []string   `json:"category,omitempty"`
	AecStatus       []string   `json:"aec_status,omitempty"`
	CollectionType  []string   `json:"collection_type,omitempty"`
	Sensitive       *bool      `json:"sensitive,omitempty"`
	ComplexityLevel []string   `json:"complexity_level,omitempty"`
}

// EvidenceRecord represents an actual piece of evidence
type EvidenceRecord struct {
	ID          string                 `json:"id"`
	TaskID      int                    `json:"task_id"`
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Content     string                 `json:"content"`
	Format      string                 `json:"format"` // csv, markdown, pdf, etc.
	Source      string                 `json:"source"` // terraform, github, manual, etc.
	CollectedAt time.Time              `json:"collected_at"`
	CollectedBy string                 `json:"collected_by"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Attachments []string               `json:"attachments,omitempty"`
}

// EvidenceTaskMasterContent represents the master content for an evidence task
type EvidenceTaskMasterContent struct {
	Guidance    string `json:"guidance"`
	Description string `json:"description"`
	Help        string `json:"help"`
}

// EvidenceTaskAssociations represents various association counts for an evidence task
type EvidenceTaskAssociations struct {
	Controls   int `json:"controls"`
	Policies   int `json:"policies"`
	Procedures int `json:"procedures"`
}

// AecStatus represents Automated Evidence Collection status
type AecStatus struct {
	ID                string     `json:"id"`
	Status            string     `json:"status"`
	LastExecuted      *time.Time `json:"last_executed,omitempty"`
	NextScheduled     *time.Time `json:"next_scheduled,omitempty"`
	ErrorMessage      string     `json:"error_message,omitempty"`
	SuccessfulRuns    int        `json:"successful_runs"`
	FailedRuns        int        `json:"failed_runs"`
	LastSuccessfulRun *time.Time `json:"last_successful_run,omitempty"`
}

// SubtaskMetadata represents metadata about subtasks
type SubtaskMetadata struct {
	TotalSubtasks     int `json:"total_subtasks"`
	CompletedSubtasks int `json:"completed_subtasks"`
	PendingSubtasks   int `json:"pending_subtasks"`
	OverdueSubtasks   int `json:"overdue_subtasks"`
}

// Integration represents an integration available for evidence collection
type Integration struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Description string                 `json:"description,omitempty"`
	Enabled     bool                   `json:"enabled"`
	Config      map[string]interface{} `json:"config,omitempty"`
}

// EvidenceTaskDetails is an alias for EvidenceTask since we unified the model
// This maintains backward compatibility for existing code
type EvidenceTaskDetails = EvidenceTask

// GetCategory returns the category for the evidence task, assigning one if not set
func (et *EvidenceTask) GetCategory() string {
	if et.Category != "" {
		return et.Category
	}
	return et.AssignCategory()
}

// AssignCategory intelligently assigns a category based on task characteristics
func (et *EvidenceTask) AssignCategory() string {
	taskText := strings.ToLower(et.Name + " " + et.Description)

	// Infrastructure category - technical/system configurations
	if containsAny(taskText, []string{
		"firewall", "configuration", "encryption", "antivirus", "vulnerability",
		"penetration", "baseline", "workstation", "server", "network", "infrastructure",
		"patch", "capacity", "monitoring", "logging", "backup", "multi-factor",
		"password", "disk encryption", "zones", "serverless", "functions",
	}) {
		et.Category = "Infrastructure"
		return et.Category
	}

	// Personnel category - people and access management
	if containsAny(taskText, []string{
		"employee", "contractor", "user", "access", "registration", "termination",
		"background check", "training", "awareness", "onboarding", "offboarding",
		"personnel", "job description", "performance", "administrative access",
		"segregation of duties",
	}) {
		et.Category = "Personnel"
		return et.Category
	}

	// Process category - policies, procedures, documentation
	if containsAny(taskText, []string{
		"policy", "procedure", "management", "approval", "documentation", "process",
		"incident", "change", "vendor", "risk assessment", "code of conduct",
		"ethics", "organizational chart", "training plan", "disaster recovery",
		"business continuity",
	}) {
		et.Category = "Process"
		return et.Category
	}

	// Compliance category - audit reports and regulatory requirements
	if containsAny(taskText, []string{
		"audit", "compliance", "report", "assessment", "review", "insurance",
		"privacy policy", "data breach", "notification", "lessons learnt",
		"penetration test", "vendor audit", "control monitoring",
	}) {
		et.Category = "Compliance"
		return et.Category
	}

	// Monitoring category - logs, alerts, system monitoring
	if containsAny(taskText, []string{
		"log", "alert", "notification", "monitoring", "availability", "event",
		"system", "customer support", "release notes", "component documentation",
	}) {
		et.Category = "Monitoring"
		return et.Category
	}

	// Data category - data handling, retention, customer data
	if containsAny(taskText, []string{
		"data", "retention", "disposal", "customer", "contract", "eula",
		"encryption key", "production data", "testing", "development",
	}) {
		et.Category = "Data"
		return et.Category
	}

	// Default to Process if no specific category matches
	et.Category = "Process"
	return et.Category
}

// GetComplexityLevel returns the complexity level, assigning one if not set
func (et *EvidenceTask) GetComplexityLevel() string {
	if et.ComplexityLevel != "" {
		return et.ComplexityLevel
	}
	return et.AssignComplexityLevel()
}

// AssignComplexityLevel determines complexity based on multiple factors
func (et *EvidenceTask) AssignComplexityLevel() string {
	score := 0

	// Factor 1: Guidance length (more detailed guidance = more complex)
	if len(et.Guidance) > 1000 {
		score += 2
	} else if len(et.Guidance) > 500 {
		score += 1
	}

	// Factor 2: Number of related controls (more controls = more complex)
	if len(et.RelatedControls) > 3 {
		score += 2
	} else if len(et.RelatedControls) > 1 {
		score += 1
	}

	// Factor 3: Collection interval (more frequent = simpler)
	switch strings.ToLower(et.CollectionInterval) {
	case "monthly":
		score += 1
	case "quarterly":
		score += 0 // neutral
	case "annually", "year":
		score += 2 // annual tasks tend to be more comprehensive
	}

	// Factor 4: Sensitive data handling adds complexity
	if et.Sensitive {
		score += 1
	}

	// Factor 5: Category-based complexity
	switch et.GetCategory() {
	case "Infrastructure":
		score += 1 // Technical complexity
	case "Compliance":
		score += 2 // Regulatory complexity
	case "Process":
		score += 1 // Documentation complexity
	}

	// Assign complexity level based on score
	if score >= 5 {
		et.ComplexityLevel = "Complex"
	} else if score >= 3 {
		et.ComplexityLevel = "Moderate"
	} else {
		et.ComplexityLevel = "Simple"
	}

	return et.ComplexityLevel
}

// GetCollectionType returns the collection type, assigning one if not set
func (et *EvidenceTask) GetCollectionType() string {
	if et.CollectionType != "" {
		return et.CollectionType
	}
	return et.AssignCollectionType()
}

// AssignCollectionType determines how evidence should be collected
func (et *EvidenceTask) AssignCollectionType() string {
	// Check AEC status first
	if et.AecStatus != nil {
		switch et.AecStatus.Status {
		case "enabled":
			// Check if task also needs manual elements
			if et.RequiresManualEvidence() {
				et.CollectionType = "Hybrid"
			} else {
				et.CollectionType = "Automated"
			}
			return et.CollectionType
		case "disabled":
			et.CollectionType = "Manual" // Could be automated but not currently
			return et.CollectionType
		}
	}

	// Check if task can potentially be automated based on category/content
	if et.CanBeAutomated() {
		et.CollectionType = "Hybrid" // Automation possible but may need manual elements
	} else {
		et.CollectionType = "Manual"
	}

	return et.CollectionType
}

// RequiresManualEvidence checks if task needs manual evidence even with automation
func (et *EvidenceTask) RequiresManualEvidence() bool {
	taskText := strings.ToLower(et.Name + " " + et.Description)

	// These types of evidence typically require manual review/approval
	manualIndicators := []string{
		"approval", "signed", "acknowledgment", "review", "assessment",
		"interview", "observation", "policy", "procedure", "contract",
		"agreement", "training", "background check", "meeting minutes",
	}

	return containsAny(taskText, manualIndicators)
}

// CanBeAutomated checks if task has potential for automation
func (et *EvidenceTask) CanBeAutomated() bool {
	taskText := strings.ToLower(et.Name + " " + et.Description)

	// These types of evidence can potentially be automated
	automationIndicators := []string{
		"configuration", "log", "list", "inventory", "report", "scan",
		"monitoring", "alert", "backup", "user", "access", "firewall",
		"antivirus", "patch", "encryption", "system", "server", "network",
	}

	return containsAny(taskText, automationIndicators)
}

// GetStatus returns the status, deriving from completed field if not set
func (et *EvidenceTask) GetStatus() string {
	if et.Status != "" {
		return et.Status
	}
	return et.AssignStatus()
}

// AssignStatus derives status from the completed field
func (et *EvidenceTask) AssignStatus() string {
	if et.Completed {
		et.Status = "completed"
	} else {
		et.Status = "pending"
	}
	return et.Status
}

// GetPriority returns the priority, deriving from collection interval if not set
func (et *EvidenceTask) GetPriority() string {
	if et.Priority != "" {
		return et.Priority
	}
	return et.AssignPriority()
}

// AssignPriority derives priority from collection interval
func (et *EvidenceTask) AssignPriority() string {
	switch strings.ToLower(et.CollectionInterval) {
	case "year":
		et.Priority = "low"
	case "quarter":
		et.Priority = "medium"
	case "month", "week":
		et.Priority = "high"
	default:
		et.Priority = "medium"
	}
	return et.Priority
}

// GetFramework returns the framework, deriving from related controls if not set
func (et *EvidenceTask) GetFramework() string {
	if et.Framework != "" {
		return et.Framework
	}
	return et.AssignFramework()
}

// AssignFramework derives framework from related controls
func (et *EvidenceTask) AssignFramework() string {
	if len(et.RelatedControls) == 0 {
		return ""
	}

	frameworks := make(map[string]int)
	for _, ctrl := range et.RelatedControls {
		if ctrl.Framework != "" {
			frameworks[ctrl.Framework]++
		}
		// Also check framework_codes for more accurate detection
		for _, fc := range ctrl.FrameworkCodes {
			if fc.Framework != "" {
				frameworks[fc.Framework]++
			}
		}
	}

	// Use the most common framework
	maxCount := 0
	for fw, count := range frameworks {
		if count > maxCount {
			et.Framework = fw
			maxCount = count
		}
	}

	return et.Framework
}

// GetAecStatusDisplay returns a user-friendly AEC status
func (et *EvidenceTask) GetAecStatusDisplay() string {
	if et.AecStatus == nil {
		return "N/A"
	}

	switch et.AecStatus.Status {
	case "enabled":
		return "Enabled"
	case "disabled":
		return "Disabled"
	case "na":
		return "Not Available"
	default:
		return strings.ToTitle(et.AecStatus.Status)
	}
}

// Helper function to check if text contains any of the given keywords
func containsAny(text string, keywords []string) bool {
	for _, keyword := range keywords {
		if strings.Contains(text, keyword) {
			return true
		}
	}
	return false
}
