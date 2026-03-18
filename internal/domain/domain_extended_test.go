// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package domain

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Tag UnmarshalJSON tests ---

func TestTag_UnmarshalJSON_String(t *testing.T) {
	t.Parallel()
	var tag Tag
	err := json.Unmarshal([]byte(`"compliance"`), &tag)
	require.NoError(t, err)
	assert.Equal(t, "compliance", tag.Name)
	assert.Empty(t, tag.ID)
	assert.Empty(t, tag.Color)
}

func TestTag_UnmarshalJSON_Object(t *testing.T) {
	t.Parallel()
	var tag Tag
	err := json.Unmarshal([]byte(`{"id":"tag-1","name":"SOC2","color":"blue"}`), &tag)
	require.NoError(t, err)
	assert.Equal(t, "tag-1", tag.ID)
	assert.Equal(t, "SOC2", tag.Name)
	assert.Equal(t, "blue", tag.Color)
}

func TestTag_UnmarshalJSON_InvalidJSON(t *testing.T) {
	t.Parallel()
	var tag Tag
	err := json.Unmarshal([]byte(`invalid`), &tag)
	assert.Error(t, err)
}

func TestTag_UnmarshalJSON_EmptyString(t *testing.T) {
	t.Parallel()
	var tag Tag
	err := json.Unmarshal([]byte(`""`), &tag)
	require.NoError(t, err)
	assert.Empty(t, tag.Name)
}

func TestTag_JSON_Roundtrip(t *testing.T) {
	t.Parallel()
	tag := Tag{ID: "t1", Name: "audit", Color: "red"}
	data, err := json.Marshal(tag)
	require.NoError(t, err)

	var decoded Tag
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, tag, decoded)
}

// --- Person tests ---

func TestPerson_JSON_Roundtrip(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)
	person := Person{
		ID:         "p1",
		Name:       "Alice",
		Email:      "alice@example.com",
		Role:       "admin",
		AssignedAt: &now,
	}
	data, err := json.Marshal(person)
	require.NoError(t, err)

	var decoded Person
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, person.ID, decoded.ID)
	assert.Equal(t, person.Name, decoded.Name)
	assert.Equal(t, person.Email, decoded.Email)
}

// --- FrameworkCode tests ---

func TestFrameworkCode_JSON_Roundtrip(t *testing.T) {
	t.Parallel()
	fc := FrameworkCode{
		ID:        "fc-1",
		Code:      "CC6.8",
		Framework: "SOC 2",
		Name:      "Logical Access Controls",
	}
	data, err := json.Marshal(fc)
	require.NoError(t, err)

	var decoded FrameworkCode
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, fc, decoded)
}

// --- Relationship tests ---

func TestRelationship_JSON_Roundtrip(t *testing.T) {
	t.Parallel()
	rel := Relationship{
		SourceType: "control",
		SourceID:   "AC1",
		TargetType: "evidence_task",
		TargetID:   "ET-0001",
		Type:       "verifies",
	}
	data, err := json.Marshal(rel)
	require.NoError(t, err)

	var decoded Relationship
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, rel, decoded)
}

// --- Policy tests ---

func TestPolicy_JSON_Roundtrip(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)
	policy := Policy{
		ID:          "pol-1",
		Name:        "Access Control Policy",
		Description: "Governs access control",
		Framework:   "SOC2",
		Status:      "published",
		ReferenceID: "P1",
		Category:    "Access Control",
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	data, err := json.Marshal(policy)
	require.NoError(t, err)

	var decoded Policy
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, policy.ID, decoded.ID)
	assert.Equal(t, policy.Name, decoded.Name)
	assert.Equal(t, policy.ReferenceID, decoded.ReferenceID)
}

// --- Control tests ---

func TestControl_JSON_Roundtrip(t *testing.T) {
	t.Parallel()
	control := Control{
	ID: "123",
		ReferenceID: "AC1",
		Name:        "Access Provisioning",
		Description: "Controls access",
		Category:    "Access",
		Framework:   "SOC2",
		Status:      "implemented",
	}
	data, err := json.Marshal(control)
	require.NoError(t, err)

	var decoded Control
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, control.ID, decoded.ID)
	assert.Equal(t, control.ReferenceID, decoded.ReferenceID)
	assert.Equal(t, control.Name, decoded.Name)
}

// --- EvidenceTask tests ---

func TestEvidenceTask_GetCategory(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		task     EvidenceTask
		expected string
	}{
		{
			name:     "returns existing category",
			task:     EvidenceTask{Category: "Infrastructure"},
			expected: "Infrastructure",
		},
		{
			name:     "assigns Infrastructure for firewall",
			task:     EvidenceTask{Name: "Firewall Configuration Evidence"},
			expected: "Infrastructure",
		},
		{
			name:     "assigns Personnel for employee",
			task:     EvidenceTask{Name: "Employee Background Check Records"},
			expected: "Personnel",
		},
		{
			name:     "assigns Process for policy",
			task:     EvidenceTask{Name: "Risk Management Policy Acknowledgment"},
			expected: "Process",
		},
		{
			name:     "assigns Compliance for audit",
			task:     EvidenceTask{Name: "Annual Audit Report"},
			expected: "Compliance",
		},
		{
			name:     "assigns Monitoring for release notes",
			task:     EvidenceTask{Name: "Release Notes Evidence"},
			expected: "Monitoring",
		},
		{
			name:     "assigns Data for data retention",
			task:     EvidenceTask{Name: "Data Retention Schedule"},
			expected: "Data",
		},
		{
			name:     "defaults to Process for unmatched",
			task:     EvidenceTask{Name: "XYZ Unknown Task"},
			expected: "Process",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.task.GetCategory()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvidenceTask_AssignCategory_SetsField(t *testing.T) {
	t.Parallel()
	task := EvidenceTask{Name: "Server Patch Management"}
	result := task.AssignCategory()
	assert.Equal(t, "Infrastructure", result)
	assert.Equal(t, "Infrastructure", task.Category, "field should be set on the struct")
}

func TestEvidenceTask_AssignCategory_DescriptionMatches(t *testing.T) {
	t.Parallel()
	task := EvidenceTask{Name: "Evidence Task", Description: "Show vulnerability scan results"}
	result := task.AssignCategory()
	assert.Equal(t, "Infrastructure", result)
}

func TestEvidenceTask_GetComplexityLevel(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		task     EvidenceTask
		expected string
	}{
		{
			name:     "returns existing complexity",
			task:     EvidenceTask{ComplexityLevel: "Complex"},
			expected: "Complex",
		},
		{
			name:     "simple task - short guidance, few controls",
			task:     EvidenceTask{Name: "Simple Evidence", Guidance: "Short guidance", CollectionInterval: "quarterly"},
			expected: "Simple",
		},
		{
			name: "complex task - long guidance, many controls, annual, sensitive",
			task: EvidenceTask{
				Name:               "Complex Compliance Review",
				Guidance:           string(make([]byte, 1500)),
				CollectionInterval: "annually",
				Sensitive:          true,
				RelatedControls: []Control{
					{ID: "1"}, {ID: "2"}, {ID: "3"}, {ID: "4"},
				},
			},
			expected: "Complex",
		},
		{
			name: "moderate task - medium guidance",
			task: EvidenceTask{
				Name:               "Access Review",
				Guidance:           string(make([]byte, 600)),
				CollectionInterval: "monthly",
				RelatedControls:    []Control{{ID: "1"}, {ID: "2"}},
			},
			expected: "Moderate",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.task.GetComplexityLevel()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvidenceTask_AssignComplexityLevel_SetsField(t *testing.T) {
	t.Parallel()
	task := EvidenceTask{Name: "Simple task"}
	result := task.AssignComplexityLevel()
	assert.NotEmpty(t, result)
	assert.Equal(t, result, task.ComplexityLevel)
}

func TestEvidenceTask_GetCollectionType(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		task     EvidenceTask
		expected string
	}{
		{
			name:     "returns existing collection type",
			task:     EvidenceTask{CollectionType: "Automated"},
			expected: "Automated",
		},
		{
			name: "AEC enabled, no manual needed",
			task: EvidenceTask{
				Name:      "Firewall Config Scan",
				AecStatus: &AecStatus{Status: "enabled"},
			},
			expected: "Automated",
		},
		{
			name: "AEC enabled, manual also needed",
			task: EvidenceTask{
				Name:      "Policy Approval and Review",
				AecStatus: &AecStatus{Status: "enabled"},
			},
			expected: "Hybrid",
		},
		{
			name: "AEC disabled",
			task: EvidenceTask{
				Name:      "Something",
				AecStatus: &AecStatus{Status: "disabled"},
			},
			expected: "Manual",
		},
		{
			name:     "no AEC, can be automated",
			task:     EvidenceTask{Name: "System Configuration Log"},
			expected: "Hybrid",
		},
		{
			name:     "no AEC, cannot be automated",
			task:     EvidenceTask{Name: "XYZ Unknown Evidence"},
			expected: "Manual",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			result := tt.task.GetCollectionType()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestEvidenceTask_RequiresManualEvidence(t *testing.T) {
	t.Parallel()
	assert.True(t, (&EvidenceTask{Name: "Policy Approval Process"}).RequiresManualEvidence())
	assert.True(t, (&EvidenceTask{Name: "Signed Agreement"}).RequiresManualEvidence())
	assert.True(t, (&EvidenceTask{Name: "Training Records"}).RequiresManualEvidence())
	assert.True(t, (&EvidenceTask{Description: "Background check results"}).RequiresManualEvidence())
	assert.False(t, (&EvidenceTask{Name: "Firewall Configuration"}).RequiresManualEvidence())
}

func TestEvidenceTask_CanBeAutomated(t *testing.T) {
	t.Parallel()
	assert.True(t, (&EvidenceTask{Name: "Firewall Configuration"}).CanBeAutomated())
	assert.True(t, (&EvidenceTask{Name: "System Log Review"}).CanBeAutomated())
	assert.True(t, (&EvidenceTask{Name: "User Access List"}).CanBeAutomated())
	assert.True(t, (&EvidenceTask{Description: "Backup verification report"}).CanBeAutomated())
	assert.False(t, (&EvidenceTask{Name: "XYZ Unknown"}).CanBeAutomated())
}

func TestEvidenceTask_GetStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		task     EvidenceTask
		expected string
	}{
		{"returns existing status", EvidenceTask{Status: "in_progress"}, "in_progress"},
		{"completed task", EvidenceTask{Completed: true}, "completed"},
		{"pending task", EvidenceTask{Completed: false}, "pending"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.task.GetStatus())
		})
	}
}

func TestEvidenceTask_AssignStatus(t *testing.T) {
	t.Parallel()
	task := EvidenceTask{Completed: true}
	result := task.AssignStatus()
	assert.Equal(t, "completed", result)
	assert.Equal(t, "completed", task.Status)

	task2 := EvidenceTask{Completed: false}
	result2 := task2.AssignStatus()
	assert.Equal(t, "pending", result2)
	assert.Equal(t, "pending", task2.Status)
}

func TestEvidenceTask_GetPriority(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		task     EvidenceTask
		expected string
	}{
		{"returns existing priority", EvidenceTask{Priority: "critical"}, "critical"},
		{"year interval", EvidenceTask{CollectionInterval: "year"}, "low"},
		{"quarter interval", EvidenceTask{CollectionInterval: "quarter"}, "medium"},
		{"month interval", EvidenceTask{CollectionInterval: "month"}, "high"},
		{"week interval", EvidenceTask{CollectionInterval: "week"}, "high"},
		{"unknown interval", EvidenceTask{CollectionInterval: "biweekly"}, "medium"},
		{"empty interval", EvidenceTask{CollectionInterval: ""}, "medium"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.task.GetPriority())
		})
	}
}

func TestEvidenceTask_AssignPriority(t *testing.T) {
	t.Parallel()
	task := EvidenceTask{CollectionInterval: "year"}
	result := task.AssignPriority()
	assert.Equal(t, "low", result)
	assert.Equal(t, "low", task.Priority)
}

func TestEvidenceTask_GetFramework(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		task     EvidenceTask
		expected string
	}{
		{"returns existing framework", EvidenceTask{Framework: "ISO27001"}, "ISO27001"},
		{"derives from controls", EvidenceTask{
			RelatedControls: []Control{
				{Framework: "SOC2"},
				{Framework: "SOC2"},
				{Framework: "ISO27001"},
			},
		}, "SOC2"},
		{"no controls", EvidenceTask{}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.task.GetFramework())
		})
	}
}

func TestEvidenceTask_AssignFramework_FromFrameworkCodes(t *testing.T) {
	t.Parallel()
	task := EvidenceTask{
		RelatedControls: []Control{
			{
				FrameworkCodes: []FrameworkCode{
					{Framework: "SOC 2"},
					{Framework: "SOC 2"},
				},
			},
		},
	}
	result := task.AssignFramework()
	assert.Equal(t, "SOC 2", result)
	assert.Equal(t, "SOC 2", task.Framework)
}

func TestEvidenceTask_AssignFramework_NoControls(t *testing.T) {
	t.Parallel()
	task := EvidenceTask{}
	result := task.AssignFramework()
	assert.Empty(t, result)
}

func TestEvidenceTask_GetAecStatusDisplay(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		task     EvidenceTask
		expected string
	}{
		{"nil AEC status", EvidenceTask{}, "N/A"},
		{"enabled", EvidenceTask{AecStatus: &AecStatus{Status: "enabled"}}, "Enabled"},
		{"disabled", EvidenceTask{AecStatus: &AecStatus{Status: "disabled"}}, "Disabled"},
		{"na", EvidenceTask{AecStatus: &AecStatus{Status: "na"}}, "Not Available"},
		{"other status", EvidenceTask{AecStatus: &AecStatus{Status: "custom"}}, "CUSTOM"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.expected, tt.task.GetAecStatusDisplay())
		})
	}
}

// --- containsAny tests ---

func TestContainsAny(t *testing.T) {
	t.Parallel()
	assert.True(t, containsAny("firewall configuration review", []string{"firewall", "config"}))
	assert.True(t, containsAny("employee training records", []string{"training"}))
	assert.False(t, containsAny("unknown task", []string{"firewall", "config"}))
	assert.False(t, containsAny("", []string{"anything"}))
	assert.False(t, containsAny("some text", []string{}))
}

// --- EvidenceTask JSON roundtrip ---

func TestEvidenceTask_JSON_Roundtrip(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)
	task := EvidenceTask{
	ID: "42",
		ReferenceID:        "ET-0042",
		Name:               "Test Evidence Task",
		Description:        "A test task",
		CollectionInterval: "quarter",
		Priority:           "medium",
		Status:             "pending",
		Completed:          false,
		Category:           "Infrastructure",
		ComplexityLevel:    "Moderate",
		CollectionType:     "Hybrid",
		CreatedAt:          now,
		UpdatedAt:          now,
		Tags:               []Tag{{Name: "SOC2"}},
	}

	data, err := json.Marshal(task)
	require.NoError(t, err)

	var decoded EvidenceTask
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, task.ID, decoded.ID)
	assert.Equal(t, task.ReferenceID, decoded.ReferenceID)
	assert.Equal(t, task.Name, decoded.Name)
	assert.Equal(t, task.Category, decoded.Category)
}

// --- EvidenceRecord tests ---

func TestEvidenceRecord_JSON_Roundtrip(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)
	record := EvidenceRecord{
		ID:          "ev-1",
		TaskID: "42",
		Title:       "Firewall Rules",
		Description: "Current firewall configuration",
		Content:     "rule1...",
		Format:      "csv",
		Source:      "terraform",
		CollectedAt: now,
		CollectedBy: "grctool",
		Metadata:    map[string]interface{}{"version": "1.0"},
		Attachments: []string{"firewall.csv"},
	}
	data, err := json.Marshal(record)
	require.NoError(t, err)

	var decoded EvidenceRecord
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, record.ID, decoded.ID)
	assert.Equal(t, record.Title, decoded.Title)
}

// --- PolicyReferenceProcessor additional tests ---

func TestPolicyReferenceProcessor_FindNextAvailablePNumber(t *testing.T) {
	t.Parallel()
	processor := NewPolicyReferenceProcessor()

	// No conflicts
	usedRefs := map[string]bool{"P1": true, "P36": true}
	next := processor.findNextAvailablePNumber(usedRefs)
	assert.Equal(t, 1000, next)

	// P1000 taken
	usedRefs["P1000"] = true
	next = processor.findNextAvailablePNumber(usedRefs)
	assert.Equal(t, 1001, next)
}

// --- ControlReferenceProcessor additional tests ---

func TestControlReferenceProcessor_EmptyInput(t *testing.T) {
	t.Parallel()
	processor := NewControlReferenceProcessor()
	result := processor.ProcessControlReferences([]Control{})
	assert.Empty(t, result)
}

func TestControlReferenceProcessor_AllWithPrefixes(t *testing.T) {
	t.Parallel()
	processor := NewControlReferenceProcessor()
	controls := []Control{
		{ID: "1", Name: "AC1 - Access Control"},
		{ID: "2", Name: "OM2 - Operations"},
	}
	result := processor.ProcessControlReferences(controls)
	assert.Len(t, result, 2)
	assert.Equal(t, "AC1", result[0].ReferenceID)
	assert.Equal(t, "OM2", result[1].ReferenceID)
}

func TestControlReferenceProcessor_AllWithoutPrefixes(t *testing.T) {
	t.Parallel()
	processor := NewControlReferenceProcessor()
	controls := []Control{
		{ID: "1", Name: "Zebra Control"},
		{ID: "2", Name: "Alpha Control"},
	}
	result := processor.ProcessControlReferences(controls)
	assert.Len(t, result, 2)
	// Alpha sorts first
	for _, c := range result {
		if c.Name == "Alpha Control" {
			assert.Equal(t, "C1", c.ReferenceID)
		}
		if c.Name == "Zebra Control" {
			assert.Equal(t, "C2", c.ReferenceID)
		}
	}
}

// --- PolicySummary / ControlSummary / EvidenceTaskSummary tests ---

func TestPolicySummary_JSON(t *testing.T) {
	t.Parallel()
	summary := PolicySummary{
		Total:       10,
		ByFramework: map[string]int{"SOC2": 5, "ISO27001": 5},
		ByStatus:    map[string]int{"published": 8, "draft": 2},
	}
	data, err := json.Marshal(summary)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"total":10`)
}

func TestControlSummary_JSON(t *testing.T) {
	t.Parallel()
	summary := ControlSummary{
		Total:       20,
		ByFramework: map[string]int{"SOC2": 15},
		ByStatus:    map[string]int{"implemented": 18},
		ByCategory:  map[string]int{"Access": 10},
	}
	data, err := json.Marshal(summary)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"total":20`)
}

func TestEvidenceTaskSummary_JSON(t *testing.T) {
	t.Parallel()
	summary := EvidenceTaskSummary{
		Total:      50,
		ByStatus:   map[string]int{"pending": 30, "completed": 20},
		ByPriority: map[string]int{"high": 10, "medium": 30, "low": 10},
		Overdue:    5,
		DueSoon:    8,
	}
	data, err := json.Marshal(summary)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"total":50`)
}

// --- Category derivation edge cases ---

func TestEvidenceTask_AssignCategory_PersonnelByDescription(t *testing.T) {
	t.Parallel()
	task := EvidenceTask{Name: "Evidence", Description: "show user access registration process"}
	task.AssignCategory()
	assert.Equal(t, "Personnel", task.Category)
}

func TestEvidenceTask_AssignCategory_MonitoringByDescription(t *testing.T) {
	t.Parallel()
	// Use "release notes" which is unique to Monitoring category
	task := EvidenceTask{Name: "Evidence", Description: "release notes for latest version"}
	task.AssignCategory()
	assert.Equal(t, "Monitoring", task.Category)
}

func TestEvidenceTask_AssignCategory_DataByDescription(t *testing.T) {
	t.Parallel()
	// Use "production data" which uniquely matches Data category
	task := EvidenceTask{Name: "Evidence", Description: "production data in testing environments"}
	task.AssignCategory()
	assert.Equal(t, "Data", task.Category)
}

func TestEvidenceTask_AssignCategory_ComplianceByDescription(t *testing.T) {
	t.Parallel()
	task := EvidenceTask{Name: "Evidence", Description: "annual compliance assessment report"}
	task.AssignCategory()
	assert.Equal(t, "Compliance", task.Category)
}

// --- OrgScope tests ---

func TestOrgScope_JSON_Roundtrip(t *testing.T) {
	t.Parallel()
	scope := OrgScope{ID: 1, Name: "Engineering", Description: "Eng team", Type: "department"}
	data, err := json.Marshal(scope)
	require.NoError(t, err)

	var decoded OrgScope
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, scope, decoded)
}

// --- AuditProject tests ---

func TestAuditProject_JSON_Roundtrip(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)
	project := AuditProject{
		ID:          "ap-1",
		Name:        "SOC2 Audit 2024",
		Status:      "in_progress",
		StartDate:   now,
		EndDate:     now.AddDate(0, 3, 0),
		Description: "Annual audit",
	}
	data, err := json.Marshal(project)
	require.NoError(t, err)

	var decoded AuditProject
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, project.ID, decoded.ID)
	assert.Equal(t, project.Name, decoded.Name)
}

// --- JiraIssue tests ---

func TestJiraIssue_JSON_Roundtrip(t *testing.T) {
	t.Parallel()
	now := time.Now().Truncate(time.Second)
	issue := JiraIssue{
		ID:        "jira-1",
		Key:       "SEC-123",
		Summary:   "Fix access control",
		Status:    "open",
		Priority:  "high",
		IssueType: "task",
		CreatedAt: now,
		UpdatedAt: now,
		Assignee:  "alice",
		Reporter:  "bob",
	}
	data, err := json.Marshal(issue)
	require.NoError(t, err)

	var decoded JiraIssue
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, issue.Key, decoded.Key)
}

// --- EvidenceFilter tests ---

func TestEvidenceFilter_JSON(t *testing.T) {
	t.Parallel()
	filter := EvidenceFilter{
		Status:    []string{"pending"},
		Priority:  []string{"high", "medium"},
		Framework: "SOC2",
	}
	data, err := json.Marshal(filter)
	require.NoError(t, err)

	var decoded EvidenceFilter
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, filter.Status, decoded.Status)
	assert.Equal(t, filter.Framework, decoded.Framework)
}

// --- SubtaskMetadata tests ---

func TestSubtaskMetadata_JSON(t *testing.T) {
	t.Parallel()
	meta := SubtaskMetadata{
		TotalSubtasks:     10,
		CompletedSubtasks: 7,
		PendingSubtasks:   2,
		OverdueSubtasks:   1,
	}
	data, err := json.Marshal(meta)
	require.NoError(t, err)

	var decoded SubtaskMetadata
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, meta, decoded)
}

// --- Integration tests ---

func TestIntegration_JSON(t *testing.T) {
	t.Parallel()
	integration := Integration{
		ID:          "int-1",
		Name:        "GitHub",
		Type:        "scm",
		Description: "GitHub integration",
		Enabled:     true,
		Config:      map[string]interface{}{"repo": "org/repo"},
	}
	data, err := json.Marshal(integration)
	require.NoError(t, err)

	var decoded Integration
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, integration.ID, decoded.ID)
	assert.Equal(t, integration.Name, decoded.Name)
}
