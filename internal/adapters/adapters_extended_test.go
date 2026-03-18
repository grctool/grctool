// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package adapters

import (
	"testing"
	"time"

	tugboatmodels "github.com/grctool/grctool/internal/tugboat/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTugboatToDomain(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	require.NotNil(t, adapter)
}

// --- ConvertPolicy tests ---

func TestConvertPolicy_BasicPolicy(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	now := time.Now().Truncate(time.Second)

	apiPolicy := tugboatmodels.Policy{
		ID:          "42",
		Name:        "Access Control Policy",
		Description: "Governs access controls",
		Framework:   "SOC2",
		Status:      "published",
		CreatedAt:   tugboatmodels.FlexibleTime{Time: now},
		UpdatedAt:   tugboatmodels.FlexibleTime{Time: now},
	}

	result := adapter.ConvertPolicy(apiPolicy)
	assert.Equal(t, "42", result.ID)
	assert.Equal(t, "Access Control Policy", result.Name)
	assert.Equal(t, "Governs access controls", result.Description)
	assert.Equal(t, "SOC2", result.Framework)
	assert.Equal(t, "published", result.Status)
}

func TestConvertPolicy_PolicyDetails(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	now := time.Now().Truncate(time.Second)

	apiPolicy := tugboatmodels.PolicyDetails{
		Policy: tugboatmodels.Policy{
			ID:        "42",
			Name:      "Access Control Policy",
			Framework: "SOC2",
			Status:    "published",
			CreatedAt: tugboatmodels.FlexibleTime{Time: now},
			UpdatedAt: tugboatmodels.FlexibleTime{Time: now},
		},
		Summary:          "Policy summary",
		Details:          "Full policy content here",
		Category:         "Access Control",
		MasterPolicyID:   "36",
		VersionNum:       3,
		MasterVersionNum: 2,
		Tags: []tugboatmodels.PolicyTag{
			{ID: "t1", Name: "SOC2", Color: "blue"},
		},
		Assignees: []tugboatmodels.PolicyAssignee{
			{
				ID:         "u1",
				Name:       "Alice",
				Email:      "alice@example.com",
				Role:       "owner",
				AssignedAt: tugboatmodels.FlexibleTime{Time: now},
			},
		},
		Reviewers: []tugboatmodels.PolicyReviewer{
			{ID: "u2", Name: "Bob", Email: "bob@example.com", Status: "approved"},
		},
		AssociationCounts: &tugboatmodels.AssociationCounts{
			Controls:   5,
			Procedures: 3,
			Evidence:   10,
		},
		Usage: &tugboatmodels.PolicyUsage{
			ViewCount:      100,
			LastViewedAt:   tugboatmodels.FlexibleTime{Time: now},
			DownloadCount:  50,
			LastDownloaded: tugboatmodels.FlexibleTime{Time: now},
			ReferenceCount: 25,
			LastReferenced: tugboatmodels.FlexibleTime{Time: now},
		},
		CurrentVersion: &tugboatmodels.PolicyVersion{
			ID:        "v1",
			Version:   "3.0",
			Content:   "Version 3 content",
			Status:    "published",
			CreatedAt: tugboatmodels.FlexibleTime{Time: now},
			CreatedBy: "admin",
		},
		LatestVersion: &tugboatmodels.PolicyVersion{
			ID:      "v2",
			Version: "3.1",
		},
		DeprecationNotes: "Will be replaced",
	}

	result := adapter.ConvertPolicy(apiPolicy)
	assert.Equal(t, "42", result.ID)
	assert.Equal(t, "Policy summary", result.Summary)
	assert.Equal(t, "Full policy content here", result.Content) // Details takes priority
	assert.Equal(t, "Access Control", result.Category)
	assert.Equal(t, "36", result.MasterPolicyID)
	assert.Equal(t, 3, result.VersionNum)
	assert.Equal(t, 2, result.MasterVersionNum)
	assert.Equal(t, "Will be replaced", result.DeprecationNotes)

	// Tags
	require.Len(t, result.Tags, 1)
	assert.Equal(t, "SOC2", result.Tags[0].Name)

	// Assignees
	require.Len(t, result.Assignees, 1)
	assert.Equal(t, "Alice", result.Assignees[0].Name)

	// Reviewers
	require.Len(t, result.Reviewers, 1)
	assert.Equal(t, "Bob", result.Reviewers[0].Name)
	assert.Equal(t, "approved", result.Reviewers[0].Role)

	// Association counts
	assert.Equal(t, 5, result.ControlCount)
	assert.Equal(t, 3, result.ProcedureCount)
	assert.Equal(t, 10, result.EvidenceCount)

	// Usage
	assert.Equal(t, 100, result.ViewCount)
	assert.Equal(t, 50, result.DownloadCount)
	assert.Equal(t, 25, result.ReferenceCount)

	// Version
	assert.Equal(t, "3.0", result.Version)
	require.NotNil(t, result.CurrentVersion)
	assert.Equal(t, "v1", result.CurrentVersion.ID)
	require.NotNil(t, result.LatestVersion)
	assert.Equal(t, "v2", result.LatestVersion.ID)
}

func TestConvertPolicy_PolicyDetails_ContentFallback(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()

	// When Details is empty, Content should fall back to CurrentVersion.Content
	apiPolicy := tugboatmodels.PolicyDetails{
		Policy: tugboatmodels.Policy{
			ID:   "1",
			Name: "Test Policy",
		},
		Details: "", // Empty details
		CurrentVersion: &tugboatmodels.PolicyVersion{
			Content: "Version content fallback",
		},
	}

	result := adapter.ConvertPolicy(apiPolicy)
	assert.Equal(t, "Version content fallback", result.Content)
}

func TestConvertPolicy_PolicyDetails_NilOptionalFields(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()

	apiPolicy := tugboatmodels.PolicyDetails{
		Policy: tugboatmodels.Policy{
			ID:   "1",
			Name: "Minimal Policy",
		},
		// All optional fields are nil
	}

	result := adapter.ConvertPolicy(apiPolicy)
	assert.Equal(t, "1", result.ID)
	assert.Nil(t, result.CurrentVersion)
	assert.Nil(t, result.LatestVersion)
	assert.Equal(t, 0, result.ControlCount)
	assert.Equal(t, 0, result.ViewCount)
}

func TestConvertPolicy_UnknownType(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.ConvertPolicy("not a policy")
	assert.Empty(t, result.ID)
	assert.Empty(t, result.Name)
}

// --- ConvertControl tests ---

func TestConvertControl_BasicControl(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	riskLevel := "medium"
	implDate := "2024-01-15T00:00:00Z"

	apiControl := tugboatmodels.Control{
		ID:                42,
		Name:              "AC1 - Access Provisioning",
		Body:              "Controls access provisioning",
		Category:          "Access Control",
		Status:            "implemented",
		Risk:              "medium",
		RiskLevel:         &riskLevel,
		Help:              "Implement RBAC",
		IsAutoImplemented: true,
		ImplementedDate:   &implDate,
		Codes:             "CC6.8",
		Framework:         "SOC2",
	}

	result := adapter.ConvertControl(apiControl)
	assert.Equal(t, 42, result.ID)
	assert.Equal(t, "AC1 - Access Provisioning", result.Name)
	assert.Equal(t, "Controls access provisioning", result.Description) // Body -> Description
	assert.Equal(t, "Access Control", result.Category)
	assert.Equal(t, "implemented", result.Status)
	assert.Equal(t, "medium", result.Risk)
	assert.Equal(t, "medium", result.RiskLevel)
	assert.Equal(t, "Implement RBAC", result.Help)
	assert.True(t, result.IsAutoImplemented)
	assert.NotNil(t, result.ImplementedDate)
	assert.Equal(t, "CC6.8", result.Codes)
}

func TestConvertControl_NilOptionalFields(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()

	apiControl := tugboatmodels.Control{
		ID:              42,
		Name:            "Simple Control",
		Body:            "Body",
		RiskLevel:       nil,
		ImplementedDate: nil,
		TestedDate:      nil,
	}

	result := adapter.ConvertControl(apiControl)
	assert.Empty(t, result.RiskLevel)
	assert.Nil(t, result.ImplementedDate)
	assert.Nil(t, result.TestedDate)
}

func TestConvertControl_InvalidDateFormat(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	badDate := "not-a-date"

	apiControl := tugboatmodels.Control{
		ID:              1,
		Name:            "Test",
		ImplementedDate: &badDate,
	}

	result := adapter.ConvertControl(apiControl)
	assert.Nil(t, result.ImplementedDate, "invalid date should result in nil")
}

func TestConvertControl_UnknownType(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.ConvertControl("not a control")
	assert.Equal(t, 0, result.ID)
}

// --- ConvertEvidenceTask tests ---

func TestConvertEvidenceTask_BasicTask(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	lastCollected := "2024-06-01T00:00:00Z"

	apiTask := tugboatmodels.EvidenceTask{
		ID:                 42,
		Name:               "Firewall Configuration",
		Description:        "Show firewall config",
		CollectionInterval: "quarter",
		Completed:          false,
		LastCollected:      &lastCollected,
		CreatedAt:          "2024-01-01T00:00:00Z",
		UpdatedAt:          "2024-06-15T00:00:00Z",
	}

	result := adapter.ConvertEvidenceTask(apiTask)
	assert.Equal(t, 42, result.ID)
	assert.Equal(t, "Firewall Configuration", result.Name)
	assert.Equal(t, "pending", result.Status)          // Derived from Completed=false
	assert.Equal(t, "medium", result.Priority)          // Derived from quarter
	assert.NotNil(t, result.LastCollected)
	assert.NotNil(t, result.NextDue, "NextDue should be computed from LastCollected + quarter")
	assert.NotEmpty(t, result.Category)       // Auto-assigned
	assert.NotEmpty(t, result.ComplexityLevel) // Auto-assigned
	assert.NotEmpty(t, result.CollectionType)  // Auto-assigned
}

func TestConvertEvidenceTask_WithExistingStatus(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()

	apiTask := tugboatmodels.EvidenceTask{
		ID:       1,
		Name:     "Test",
		Status:   "in_progress",
		Priority: "high",
	}

	result := adapter.ConvertEvidenceTask(apiTask)
	assert.Equal(t, "in_progress", result.Status)
	assert.Equal(t, "high", result.Priority)
}

func TestConvertEvidenceTask_CompletedTask(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()

	apiTask := tugboatmodels.EvidenceTask{
		ID:        1,
		Name:      "Completed Task",
		Completed: true,
	}

	result := adapter.ConvertEvidenceTask(apiTask)
	assert.Equal(t, "completed", result.Status)
}

func TestConvertEvidenceTask_PriorityDerivation(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	tests := []struct {
		interval string
		expected string
	}{
		{"year", "low"},
		{"quarter", "medium"},
		{"month", "high"},
		{"week", "high"},
		{"unknown", "medium"},
	}
	for _, tt := range tests {
		apiTask := tugboatmodels.EvidenceTask{
			ID:                 1,
			Name:               "Test",
			CollectionInterval: tt.interval,
		}
		result := adapter.ConvertEvidenceTask(apiTask)
		assert.Equal(t, tt.expected, result.Priority, "for interval: %s", tt.interval)
	}
}

func TestConvertEvidenceTask_NilDates(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()

	apiTask := tugboatmodels.EvidenceTask{
		ID:            1,
		Name:          "No Dates",
		LastCollected: nil,
		NextDue:       nil,
	}

	result := adapter.ConvertEvidenceTask(apiTask)
	assert.Nil(t, result.LastCollected)
	assert.Nil(t, result.NextDue)
}

func TestConvertEvidenceTask_EmptyDates(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	empty := ""

	apiTask := tugboatmodels.EvidenceTask{
		ID:            1,
		Name:          "Empty Dates",
		LastCollected: &empty,
		NextDue:       &empty,
	}

	result := adapter.ConvertEvidenceTask(apiTask)
	assert.Nil(t, result.LastCollected)
	assert.Nil(t, result.NextDue)
}

func TestConvertEvidenceTask_UnknownType(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.ConvertEvidenceTask("not a task")
	assert.Equal(t, 0, result.ID)
}

// --- Flexible interface conversion tests ---

func TestConvertAssigneesInterface_StringSlice(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()

	assignees := []interface{}{"user-1", "user-2"}
	result := adapter.convertAssigneesInterface(assignees)
	require.Len(t, result, 2)
	assert.Equal(t, "user-1", result[0].ID)
	assert.Equal(t, "user-2", result[1].ID)
}

func TestConvertAssigneesInterface_ObjectSlice(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()

	assignees := []interface{}{
		map[string]interface{}{
			"id":    float64(123),
			"name":  "Alice",
			"email": "alice@example.com",
			"role":  "owner",
		},
	}
	result := adapter.convertAssigneesInterface(assignees)
	require.Len(t, result, 1)
	assert.Equal(t, "123", result[0].ID)
	assert.Equal(t, "Alice", result[0].Name)
	assert.Equal(t, "alice@example.com", result[0].Email)
}

func TestConvertAssigneesInterface_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertAssigneesInterface(nil)
	assert.Nil(t, result)
}

func TestConvertAssigneesInterface_NonSlice(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertAssigneesInterface("not-a-slice")
	assert.Nil(t, result)
}

func TestConvertTagsInterface_StringSlice(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()

	tags := []interface{}{"SOC2", "Infrastructure"}
	result := adapter.convertTagsInterface(tags)
	require.Len(t, result, 2)
	assert.Equal(t, "SOC2", result[0].Name)
	assert.Equal(t, "Infrastructure", result[1].Name)
}

func TestConvertTagsInterface_ObjectSlice(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()

	tags := []interface{}{
		map[string]interface{}{
			"id":    "t1",
			"name":  "SOC2",
			"color": "blue",
		},
	}
	result := adapter.convertTagsInterface(tags)
	require.Len(t, result, 1)
	assert.Equal(t, "t1", result[0].ID)
	assert.Equal(t, "SOC2", result[0].Name)
	assert.Equal(t, "blue", result[0].Color)
}

func TestConvertTagsInterface_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertTagsInterface(nil)
	assert.Nil(t, result)
}

// --- AecStatus conversion tests ---

func TestConvertAecStatus_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertAecStatus(nil)
	assert.Nil(t, result)
}

func TestConvertAecStatus_String(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertAecStatus("enabled")
	require.NotNil(t, result)
	assert.Equal(t, "enabled", result.Status)
}

func TestConvertAecStatus_Object(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()

	statusObj := map[string]interface{}{
		"id":               "aec-1",
		"status":           "enabled",
		"error_message":    "test error",
		"successful_runs":  float64(10),
		"failed_runs":      float64(2),
		"last_executed":    "2024-06-01T00:00:00Z",
		"next_scheduled":   "2024-07-01T00:00:00Z",
		"last_successful_run": "2024-05-30T00:00:00Z",
	}

	result := adapter.convertAecStatus(statusObj)
	require.NotNil(t, result)
	assert.Equal(t, "aec-1", result.ID)
	assert.Equal(t, "enabled", result.Status)
	assert.Equal(t, "test error", result.ErrorMessage)
	assert.Equal(t, 10, result.SuccessfulRuns)
	assert.Equal(t, 2, result.FailedRuns)
	assert.NotNil(t, result.LastExecuted)
	assert.NotNil(t, result.NextScheduled)
	assert.NotNil(t, result.LastSuccessfulRun)
}

func TestConvertAecStatus_UnknownType(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertAecStatus(42)
	assert.Nil(t, result)
}

// --- computeNextDue tests ---

func TestComputeNextDue(t *testing.T) {
	t.Parallel()
	base := time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)

	tests := []struct {
		interval string
		expected *time.Time
	}{
		{"year", timePtr(time.Date(2025, 1, 15, 0, 0, 0, 0, time.UTC))},
		{"quarter", timePtr(time.Date(2024, 4, 15, 0, 0, 0, 0, time.UTC))},
		{"month", timePtr(time.Date(2024, 2, 15, 0, 0, 0, 0, time.UTC))},
		{"week", timePtr(time.Date(2024, 1, 22, 0, 0, 0, 0, time.UTC))},
		{"day", timePtr(time.Date(2024, 1, 16, 0, 0, 0, 0, time.UTC))},
		{"unknown", nil},
	}

	for _, tt := range tests {
		t.Run(tt.interval, func(t *testing.T) {
			t.Parallel()
			result := computeNextDue(&base, tt.interval)
			if tt.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, *tt.expected, *result)
			}
		})
	}
}

func TestComputeNextDue_NilLastCollected(t *testing.T) {
	t.Parallel()
	result := computeNextDue(nil, "month")
	assert.Nil(t, result)
}

// --- convertControlTags tests ---

func TestConvertControlTags_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertControlTags(nil)
	assert.Empty(t, result)
}

func TestConvertControlTags_Array(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	tags := []interface{}{
		map[string]interface{}{"id": "t1", "name": "SOC2", "color": "blue"},
		map[string]interface{}{"id": "t2", "name": "ISO"},
	}
	result := adapter.convertControlTags(tags)
	require.Len(t, result, 2)
	assert.Equal(t, "SOC2", result[0].Name)
}

func TestConvertControlTags_NonArray(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertControlTags("not an array")
	assert.Empty(t, result)
}

// --- convertFrameworkCodes tests ---

func TestConvertFrameworkCodes_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertFrameworkCodes(nil)
	assert.Empty(t, result)
}

func TestConvertFrameworkCodes_ValidArray(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	codes := []interface{}{
		map[string]interface{}{
			"framework_id":   float64(1),
			"framework_name": "SOC 2",
			"code":           "CC6.8",
		},
	}
	result := adapter.convertFrameworkCodes(codes)
	require.Len(t, result, 1)
	assert.Equal(t, "CC6.8", result[0].Code)
	assert.Equal(t, "SOC 2", result[0].Framework)
}

// --- convertEvidenceTaskAssociations tests ---

func TestConvertEvidenceTaskAssociations_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertEvidenceTaskAssociations(nil)
	assert.Nil(t, result)
}

func TestConvertEvidenceTaskAssociations_Object(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	obj := map[string]interface{}{
		"controls":   float64(3),
		"policies":   float64(2),
		"procedures": float64(1),
	}
	result := adapter.convertEvidenceTaskAssociations(obj)
	require.NotNil(t, result)
	assert.Equal(t, 3, result.Controls)
	assert.Equal(t, 2, result.Policies)
	assert.Equal(t, 1, result.Procedures)
}

func TestConvertEvidenceTaskAssociations_NonObject(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertEvidenceTaskAssociations("not an object")
	assert.Nil(t, result)
}

// --- convertSubtaskMetadata tests ---

func TestConvertSubtaskMetadata_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertSubtaskMetadata(nil)
	assert.Nil(t, result)
}

func TestConvertSubtaskMetadata_Valid(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	meta := &tugboatmodels.SubtaskMetadata{
		TotalSubtasks:     10,
		CompletedSubtasks: 7,
		PendingSubtasks:   2,
		OverdueSubtasks:   1,
	}
	result := adapter.convertSubtaskMetadata(meta)
	require.NotNil(t, result)
	assert.Equal(t, 10, result.TotalSubtasks)
	assert.Equal(t, 7, result.CompletedSubtasks)
}

// --- convertOrgScope tests ---

func TestConvertOrgScope_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertOrgScope(nil)
	assert.Nil(t, result)
}

func TestConvertOrgScope_Valid(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	scope := &tugboatmodels.OrgScope{
		ID:          1,
		Name:        "Engineering",
		Description: "Eng team",
		Type:        "department",
	}
	result := adapter.convertOrgScope(scope)
	require.NotNil(t, result)
	assert.Equal(t, 1, result.ID)
	assert.Equal(t, "Engineering", result.Name)
}

// --- convertControlMasterContent tests ---

func TestConvertControlMasterContent_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertControlMasterContent(nil)
	assert.Nil(t, result)
}

func TestConvertControlMasterContent_Valid(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	content := &tugboatmodels.ControlMasterContent{
		Help:        "Help text",
		Guidance:    "Guidance text",
		Description: "Description text",
	}
	result := adapter.convertControlMasterContent(content)
	require.NotNil(t, result)
	assert.Equal(t, "Help text", result.Help)
	assert.Equal(t, "Guidance text", result.Guidance)
}

// --- convertControlAssociations tests ---

func TestConvertControlAssociations_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertControlAssociations(nil)
	assert.Nil(t, result)
}

func TestConvertControlAssociations_Valid(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	assoc := &tugboatmodels.ControlAssociations{
		Policies:   5,
		Procedures: 3,
		Evidence:   10,
		Risks:      2,
	}
	result := adapter.convertControlAssociations(assoc)
	require.NotNil(t, result)
	assert.Equal(t, 5, result.Policies)
	assert.Equal(t, 10, result.Evidence)
}

// --- convertControlEvidenceMetrics tests ---

func TestConvertControlEvidenceMetrics_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertControlEvidenceMetrics(nil)
	assert.Nil(t, result)
}

func TestConvertControlEvidenceMetrics_Valid(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	metrics := &tugboatmodels.ControlEvidenceMetrics{
		TotalCount:    20,
		CompleteCount: 15,
		OverdueCount:  3,
	}
	result := adapter.convertControlEvidenceMetrics(metrics)
	require.NotNil(t, result)
	assert.Equal(t, 20, result.TotalCount)
	assert.Equal(t, 15, result.CompleteCount)
}

// --- convertEvidenceTaskMasterContent tests ---

func TestConvertEvidenceTaskMasterContent_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertEvidenceTaskMasterContent(nil)
	assert.Nil(t, result)
}

func TestConvertEvidenceTaskMasterContent_Valid(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	content := &tugboatmodels.MasterContent{
		Guidance:    "Guidance here",
		Description: "Description here",
		Help:        "Help here",
	}
	result := adapter.convertEvidenceTaskMasterContent(content)
	require.NotNil(t, result)
	assert.Equal(t, "Guidance here", result.Guidance)
	assert.Equal(t, "Help here", result.Help)
}

// --- convertPolicyVersion tests ---

func TestConvertPolicyVersion_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertPolicyVersion(nil)
	assert.Nil(t, result)
}

// --- convertAuditProjectsInterface tests ---

func TestConvertAuditProjectsInterface_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertAuditProjectsInterface(nil)
	assert.Empty(t, result)
}

func TestConvertAuditProjectsInterface_ValidArray(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	projects := []interface{}{
		map[string]interface{}{
			"id":     "ap-1",
			"name":   "SOC2 Audit",
			"status": "active",
		},
	}
	result := adapter.convertAuditProjectsInterface(projects)
	require.Len(t, result, 1)
	assert.Equal(t, "SOC2 Audit", result[0].Name)
}

// --- convertJiraIssuesInterface tests ---

func TestConvertJiraIssuesInterface_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertJiraIssuesInterface(nil)
	assert.Empty(t, result)
}

func TestConvertJiraIssuesInterface_ValidArray(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	issues := []interface{}{
		map[string]interface{}{
			"id":       "j-1",
			"key":      "SEC-123",
			"summary":  "Fix auth",
			"status":   "open",
			"priority": "high",
		},
	}
	result := adapter.convertJiraIssuesInterface(issues)
	require.Len(t, result, 1)
	assert.Equal(t, "SEC-123", result[0].Key)
}

// --- convertEmbeddedControl tests ---

func TestConvertEmbeddedControl(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	ctrlMap := map[string]interface{}{
		"id":                  float64(42),
		"name":                "Access Control",
		"body":                "Description text",
		"category":            "Access",
		"status":              "implemented",
		"framework":           "SOC2",
		"risk":                "medium",
		"risk_level":          "low",
		"help":                "Help text",
		"is_auto_implemented": true,
		"codes":               "CC6.8",
		"master_version_num":  float64(2),
		"master_control_id":   float64(100),
		"org_id":              float64(13888),
		"org_scope_id":        float64(1),
	}
	result := adapter.convertEmbeddedControl(ctrlMap)
	assert.Equal(t, 42, result.ID)
	assert.Equal(t, "Access Control", result.Name)
	assert.Equal(t, "Description text", result.Description)
	assert.Equal(t, "SOC2", result.Framework)
	assert.True(t, result.IsAutoImplemented)
	assert.Equal(t, 2, result.MasterVersionNum)
}

func TestConvertEmbeddedControl_WithMasterContent(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	ctrlMap := map[string]interface{}{
		"id":   float64(1),
		"name": "Test",
		"master_content": map[string]interface{}{
			"guidance":    "Guide",
			"help":        "Help",
			"description": "Desc",
		},
	}
	result := adapter.convertEmbeddedControl(ctrlMap)
	require.NotNil(t, result.MasterContent)
	assert.Equal(t, "Guide", result.MasterContent.Guidance)
	assert.Equal(t, "Help", result.MasterContent.Help)
}

func TestConvertEmbeddedControl_WithDates(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	ctrlMap := map[string]interface{}{
		"id":               float64(1),
		"name":             "Test",
		"implemented_date": "2024-01-15T00:00:00Z",
		"tested_date":      "2024-02-15T00:00:00Z",
	}
	result := adapter.convertEmbeddedControl(ctrlMap)
	assert.NotNil(t, result.ImplementedDate)
	assert.NotNil(t, result.TestedDate)
}

// --- convertEmbeddedEvidenceTasks tests ---

func TestConvertEmbeddedEvidenceTasks_Nil(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertEmbeddedEvidenceTasks(nil)
	assert.Empty(t, result)
}

func TestConvertEmbeddedEvidenceTasks_ValidArray(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	tasks := []interface{}{
		map[string]interface{}{
			"id":          float64(100),
			"name":        "Firewall Config",
			"description": "Show config",
			"status":      "pending",
		},
	}
	result := adapter.convertEmbeddedEvidenceTasks(tasks)
	require.Len(t, result, 1)
	assert.Equal(t, 100, result[0].ID)
	assert.Equal(t, "Firewall Config", result[0].Name)
}

func TestConvertEmbeddedEvidenceTasks_SkipsZeroID(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	tasks := []interface{}{
		map[string]interface{}{
			"name": "No ID task",
		},
	}
	result := adapter.convertEmbeddedEvidenceTasks(tasks)
	assert.Empty(t, result, "tasks with zero ID should be skipped")
}

// --- convertEmbeddedEvidenceTask tests ---

func TestConvertEmbeddedEvidenceTask_AllFields(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	taskMap := map[string]interface{}{
		"id":                  float64(42),
		"name":                "Test Task",
		"description":         "Description",
		"collection_interval": "month",
		"priority":            "high",
		"framework":           "SOC2",
		"status":              "pending",
		"completed":           false,
		"last_collected":      "2024-06-01T00:00:00Z",
		"next_due":            "2024-07-01T00:00:00Z",
		"created_at":          "2024-01-01T00:00:00Z",
		"updated_at":          "2024-06-15T00:00:00Z",
		"master_version_num":  float64(2),
		"master_evidence_id":  float64(100),
		"org_id":              float64(13888),
		"org_scope_id":        float64(1),
		"controls":            []interface{}{float64(42), "43"},
		"master_content": map[string]interface{}{
			"guidance":    "Task guidance",
			"description": "Task desc",
			"help":        "Task help",
		},
	}

	result := adapter.convertEmbeddedEvidenceTask(taskMap)
	assert.Equal(t, 42, result.ID)
	assert.Equal(t, "Test Task", result.Name)
	assert.Equal(t, "month", result.CollectionInterval)
	assert.NotNil(t, result.LastCollected)
	assert.NotNil(t, result.NextDue)
	assert.Equal(t, 2, result.MasterVersionNum)
	assert.Equal(t, 100, result.MasterEvidenceID)
	assert.Len(t, result.Controls, 2)
	require.NotNil(t, result.MasterContent)
	assert.Equal(t, "Task guidance", result.MasterContent.Guidance)
	assert.Equal(t, "Task guidance", result.Guidance) // Also set at top level
}

// --- convertEvidenceAssignees tests ---

func TestConvertEvidenceAssignees_IDTypes(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()

	assignees := []tugboatmodels.EvidenceAssignee{
		{ID: "string-id", Name: "Alice"},
		{ID: 42, Name: "Bob"},
		{ID: float64(99), Name: "Charlie"},
		{ID: nil, Name: "Dave"},
	}

	result := adapter.convertEvidenceAssignees(assignees)
	require.Len(t, result, 4)
	assert.Equal(t, "string-id", result[0].ID)
	assert.Equal(t, "42", result[1].ID)
	assert.Equal(t, "99", result[2].ID)
	assert.Equal(t, "<nil>", result[3].ID)
}

// --- convertSupportedIntegrations tests ---

func TestConvertSupportedIntegrations(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()

	integrations := []tugboatmodels.SupportedIntegration{
		{
			ID:          "int-1",
			Name:        "GitHub",
			Type:        "scm",
			Description: "GitHub integration",
			Enabled:     true,
			Config:      map[string]interface{}{"repo": "org/repo"},
		},
	}

	result := adapter.convertSupportedIntegrations(integrations)
	require.Len(t, result, 1)
	assert.Equal(t, "GitHub", result[0].Name)
	assert.True(t, result[0].Enabled)
}

// --- helper ---

func timePtr(t time.Time) *time.Time {
	return &t
}

// --- convertEvidenceAssignees with assigneeMap ID types ---

func TestConvertAssigneesInterface_ObjectWithStringID(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	assignees := []interface{}{
		map[string]interface{}{"id": "abc"},
	}
	result := adapter.convertAssigneesInterface(assignees)
	require.Len(t, result, 1)
	assert.Equal(t, "abc", result[0].ID)
}

func TestConvertAssigneesInterface_ObjectWithIntID(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	assignees := []interface{}{
		map[string]interface{}{"id": 42},
	}
	result := adapter.convertAssigneesInterface(assignees)
	require.Len(t, result, 1)
	assert.Equal(t, "42", result[0].ID)
}

// --- Empty slice conversions ---

func TestConvertPolicyTags_Empty(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertPolicyTags([]tugboatmodels.PolicyTag{})
	assert.Empty(t, result)
}

func TestConvertPolicyAssignees_Empty(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertPolicyAssignees([]tugboatmodels.PolicyAssignee{})
	assert.Empty(t, result)
}

func TestConvertPolicyReviewers_Empty(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertPolicyReviewers([]tugboatmodels.PolicyReviewer{})
	assert.Empty(t, result)
}

func TestConvertControlAssignees_Empty(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertControlAssignees([]tugboatmodels.ControlAssignee{})
	assert.Empty(t, result)
}

func TestConvertAuditProjects_Empty(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertAuditProjects([]tugboatmodels.AuditProject{})
	assert.Empty(t, result)
}

func TestConvertJiraIssues_Empty(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertJiraIssues([]tugboatmodels.JiraIssue{})
	assert.Empty(t, result)
}

func TestConvertEvidenceTags_Empty(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertEvidenceTags([]tugboatmodels.EvidenceTag{})
	assert.Empty(t, result)
}

func TestConvertSupportedIntegrations_Empty(t *testing.T) {
	t.Parallel()
	adapter := NewTugboatToDomain()
	result := adapter.convertSupportedIntegrations([]tugboatmodels.SupportedIntegration{})
	assert.Empty(t, result)
}
