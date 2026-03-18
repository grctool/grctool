// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// IntOrString tests
// ---------------------------------------------------------------------------

func TestIntOrString_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		want    IntOrString
		wantErr bool
	}{
		{
			name:  "string value",
			input: `"12345"`,
			want:  IntOrString("12345"),
		},
		{
			name:  "integer value",
			input: `12345`,
			want:  IntOrString("12345"),
		},
		{
			name:  "float value",
			input: `12345.0`,
			want:  IntOrString("12345"),
		},
		{
			name:  "zero integer",
			input: `0`,
			want:  IntOrString("0"),
		},
		{
			name:  "empty string",
			input: `""`,
			want:  IntOrString(""),
		},
		{
			name:  "negative integer",
			input: `-42`,
			want:  IntOrString("-42"),
		},
		{
			name:  "null value unmarshals via string path",
			input: `null`,
			want:  IntOrString(""),
		},
		{
			name:    "boolean value",
			input:   `true`,
			wantErr: true,
		},
		{
			name:    "object value",
			input:   `{"id": 1}`,
			wantErr: true,
		},
		{
			name:    "array value",
			input:   `[1, 2]`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var got IntOrString
			err := json.Unmarshal([]byte(tt.input), &got)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestIntOrString_MarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value IntOrString
		want  string
	}{
		{
			name:  "numeric string",
			value: IntOrString("12345"),
			want:  `"12345"`,
		},
		{
			name:  "empty string",
			value: IntOrString(""),
			want:  `""`,
		},
		{
			name:  "text string",
			value: IntOrString("abc"),
			want:  `"abc"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := json.Marshal(tt.value)
			require.NoError(t, err)
			assert.Equal(t, tt.want, string(got))
		})
	}
}

func TestIntOrString_String(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "12345", IntOrString("12345").String())
	assert.Equal(t, "", IntOrString("").String())
}

func TestIntOrString_ToInt(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		value IntOrString
		want  int
	}{
		{"valid integer", IntOrString("42"), 42},
		{"zero", IntOrString("0"), 0},
		{"negative", IntOrString("-5"), -5},
		{"non-numeric returns 0", IntOrString("abc"), 0},
		{"empty returns 0", IntOrString(""), 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.want, tt.value.ToInt())
		})
	}
}

func TestIntOrString_RoundTrip(t *testing.T) {
	t.Parallel()

	// Test roundtrip: start with int JSON, unmarshal, marshal, compare
	type wrapper struct {
		ID IntOrString `json:"id"`
	}

	// Integer input
	input := `{"id":42}`
	var w wrapper
	require.NoError(t, json.Unmarshal([]byte(input), &w))
	assert.Equal(t, "42", w.ID.String())

	out, err := json.Marshal(w)
	require.NoError(t, err)
	assert.Equal(t, `{"id":"42"}`, string(out))

	// String input
	input2 := `{"id":"99"}`
	var w2 wrapper
	require.NoError(t, json.Unmarshal([]byte(input2), &w2))
	assert.Equal(t, "99", w2.ID.String())
}

// ---------------------------------------------------------------------------
// FlexibleTime tests
// ---------------------------------------------------------------------------

func TestFlexibleTime_UnmarshalJSON(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		input   string
		wantErr bool
		check   func(t *testing.T, ft FlexibleTime)
	}{
		{
			name:  "RFC3339 timestamp",
			input: `"2025-01-15T10:30:00Z"`,
			check: func(t *testing.T, ft FlexibleTime) {
				assert.Equal(t, 2025, ft.Year())
				assert.Equal(t, time.January, ft.Month())
				assert.Equal(t, 15, ft.Day())
			},
		},
		{
			name:  "date only",
			input: `"2025-01-15"`,
			check: func(t *testing.T, ft FlexibleTime) {
				assert.Equal(t, 2025, ft.Year())
				assert.Equal(t, time.January, ft.Month())
				assert.Equal(t, 15, ft.Day())
			},
		},
		{
			name:  "Tugboat API format with microseconds",
			input: `"2021-12-17T14:42:01.016197Z"`,
			check: func(t *testing.T, ft FlexibleTime) {
				assert.Equal(t, 2021, ft.Year())
				assert.Equal(t, time.December, ft.Month())
				assert.Equal(t, 17, ft.Day())
			},
		},
		{
			name:  "timestamp without nanoseconds",
			input: `"2023-06-01T08:00:00Z"`,
			check: func(t *testing.T, ft FlexibleTime) {
				assert.Equal(t, 2023, ft.Year())
				assert.Equal(t, time.June, ft.Month())
			},
		},
		{
			name:  "empty string sets zero time",
			input: `""`,
			check: func(t *testing.T, ft FlexibleTime) {
				assert.True(t, ft.IsZero())
			},
		},
		{
			name:    "invalid format",
			input:   `"not-a-date"`,
			wantErr: true,
		},
		{
			name:    "non-string JSON",
			input:   `12345`,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			var ft FlexibleTime
			err := json.Unmarshal([]byte(tt.input), &ft)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				if tt.check != nil {
					tt.check(t, ft)
				}
			}
		})
	}
}

func TestFlexibleTime_MarshalJSON(t *testing.T) {
	t.Parallel()

	t.Run("zero time marshals to null", func(t *testing.T) {
		t.Parallel()
		ft := FlexibleTime{}
		data, err := json.Marshal(ft)
		require.NoError(t, err)
		assert.Equal(t, "null", string(data))
	})

	t.Run("non-zero time marshals to RFC3339", func(t *testing.T) {
		t.Parallel()
		ft := FlexibleTime{Time: time.Date(2025, 3, 15, 12, 0, 0, 0, time.UTC)}
		data, err := json.Marshal(ft)
		require.NoError(t, err)
		assert.Equal(t, `"2025-03-15T12:00:00Z"`, string(data))
	})
}

func TestFlexibleTime_RoundTrip(t *testing.T) {
	t.Parallel()

	type wrapper struct {
		Created FlexibleTime `json:"created"`
	}

	// date-only -> RFC3339
	input := `{"created":"2025-06-01"}`
	var w wrapper
	require.NoError(t, json.Unmarshal([]byte(input), &w))
	assert.Equal(t, 2025, w.Created.Year())

	out, err := json.Marshal(w)
	require.NoError(t, err)
	assert.Contains(t, string(out), "2025-06-01")

	// null handling via pointer
	type ptrWrapper struct {
		Created *FlexibleTime `json:"created,omitempty"`
	}
	input2 := `{}`
	var pw ptrWrapper
	require.NoError(t, json.Unmarshal([]byte(input2), &pw))
	assert.Nil(t, pw.Created)
}

// ---------------------------------------------------------------------------
// Policy model tests
// ---------------------------------------------------------------------------

func TestPolicy_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	input := `{
		"id": 94641,
		"name": "Access Control Policy",
		"description": "Defines access controls",
		"framework": "SOC2",
		"status": "active",
		"created_at": "2021-12-17T14:42:01.016197Z",
		"updated_at": "2025-01-15"
	}`

	var p Policy
	require.NoError(t, json.Unmarshal([]byte(input), &p))

	assert.Equal(t, "94641", p.ID.String())
	assert.Equal(t, "Access Control Policy", p.Name)
	assert.Equal(t, "SOC2", p.Framework)
	assert.Equal(t, "active", p.Status)
	assert.Equal(t, 2021, p.CreatedAt.Year())
	assert.Equal(t, 2025, p.UpdatedAt.Year())

	// Marshal back
	out, err := json.Marshal(p)
	require.NoError(t, err)
	assert.Contains(t, string(out), `"name":"Access Control Policy"`)
	assert.Contains(t, string(out), `"id":"94641"`)
}

func TestPolicy_IDAsString(t *testing.T) {
	t.Parallel()

	input := `{"id": "94641", "name": "Test Policy"}`
	var p Policy
	require.NoError(t, json.Unmarshal([]byte(input), &p))
	assert.Equal(t, "94641", p.ID.String())
}

func TestPolicyDetails_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	input := `{
		"id": 94641,
		"name": "Access Control Policy",
		"description": "Defines access controls",
		"framework": "SOC2",
		"status": "active",
		"summary": "A comprehensive access control policy",
		"category": "Access Control",
		"master_policy_id": 100,
		"version_num": 3,
		"org_id": "13888",
		"created_at": "2021-12-17T14:42:01.016197Z",
		"updated_at": "2025-01-15",
		"tags": [{"id": "t1", "name": "security", "color": "#ff0000"}],
		"assignees": [{"id": "u1", "name": "John", "email": "john@example.com", "role": "owner", "assigned_at": "2025-01-01"}],
		"association_counts": {"controls": 5, "procedures": 2, "evidence": 10}
	}`

	var pd PolicyDetails
	require.NoError(t, json.Unmarshal([]byte(input), &pd))

	assert.Equal(t, "94641", pd.ID.String())
	assert.Equal(t, "A comprehensive access control policy", pd.Summary)
	assert.Equal(t, "Access Control", pd.Category)
	assert.Equal(t, "100", pd.MasterPolicyID.String())
	assert.Equal(t, 3, pd.VersionNum)
	assert.Equal(t, "13888", pd.OrgID.String())
	require.Len(t, pd.Tags, 1)
	assert.Equal(t, "security", pd.Tags[0].Name)
	require.Len(t, pd.Assignees, 1)
	assert.Equal(t, "John", pd.Assignees[0].Name)
	require.NotNil(t, pd.AssociationCounts)
	assert.Equal(t, 5, pd.AssociationCounts.Controls)
}

func TestPolicyDetails_NullableFields(t *testing.T) {
	t.Parallel()

	input := `{
		"id": 1,
		"name": "Minimal Policy",
		"description": "",
		"framework": "",
		"status": "draft",
		"published_date": null,
		"review_date": null,
		"current_version": null,
		"latest_version": null,
		"association_counts": null,
		"usage": null,
		"created_at": "",
		"updated_at": ""
	}`

	var pd PolicyDetails
	require.NoError(t, json.Unmarshal([]byte(input), &pd))

	assert.Nil(t, pd.PublishedDate)
	assert.Nil(t, pd.ReviewDate)
	assert.Nil(t, pd.CurrentVersion)
	assert.Nil(t, pd.LatestVersion)
	assert.Nil(t, pd.AssociationCounts)
	assert.Nil(t, pd.Usage)
	assert.True(t, pd.CreatedAt.IsZero())
}

// ---------------------------------------------------------------------------
// Control model tests
// ---------------------------------------------------------------------------

func TestControl_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	input := `{
		"id": 778805,
		"name": "Logical Access Security",
		"body": "The entity implements logical access security",
		"category": "Common Criteria",
		"status": "implemented",
		"risk": "medium",
		"risk_level": "high",
		"is_auto_implemented": false,
		"implemented_date": "2024-06-01",
		"codes": "CC6.1",
		"master_version_num": 2,
		"master_control_id": 100,
		"org_id": 13888,
		"org_scope_id": 5,
		"framework_codes": [
			{"framework_id": 1, "code": "CC6.1", "framework_name": "SOC 2"}
		],
		"__entity_role__": null,
		"__entity_type__": "org_control",
		"__permissions__": ["view", "edit"]
	}`

	var c Control
	require.NoError(t, json.Unmarshal([]byte(input), &c))

	assert.Equal(t, 778805, c.ID)
	assert.Equal(t, "Logical Access Security", c.Name)
	assert.Equal(t, "The entity implements logical access security", c.Body)
	assert.Equal(t, "implemented", c.Status)
	require.NotNil(t, c.RiskLevel)
	assert.Equal(t, "high", *c.RiskLevel)
	assert.False(t, c.IsAutoImplemented)
	require.NotNil(t, c.ImplementedDate)
	assert.Equal(t, "2024-06-01", *c.ImplementedDate)
	assert.Equal(t, 13888, c.OrgID)
	require.Len(t, c.FrameworkCodes, 1)
	assert.Equal(t, "CC6.1", c.FrameworkCodes[0].Code)
	assert.Equal(t, "SOC 2", c.FrameworkCodes[0].Framework)
	assert.Nil(t, c.EntityRole)
	assert.Equal(t, "org_control", c.EntityType)
	assert.Contains(t, c.Permissions, "view")

	// Marshal
	out, err := json.Marshal(c)
	require.NoError(t, err)
	assert.Contains(t, string(out), `"id":778805`)
	assert.Contains(t, string(out), `"name":"Logical Access Security"`)
}

func TestControl_NullableFields(t *testing.T) {
	t.Parallel()

	input := `{
		"id": 1,
		"name": "Minimal Control",
		"body": "",
		"category": "",
		"status": "na",
		"risk_level": null,
		"implemented_date": null,
		"tested_date": null,
		"org_scope": null,
		"__entity_role__": null,
		"__entity_type__": "org_control",
		"__permissions__": []
	}`

	var c Control
	require.NoError(t, json.Unmarshal([]byte(input), &c))

	assert.Nil(t, c.RiskLevel)
	assert.Nil(t, c.ImplementedDate)
	assert.Nil(t, c.TestedDate)
	assert.Nil(t, c.OrgScope)
}

func TestControlDetails_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	input := `{
		"id": 778805,
		"name": "Logical Access Security",
		"body": "Implementation details",
		"category": "Common Criteria",
		"status": "implemented",
		"master_version_num": 2,
		"master_control_id": 100,
		"org_id": 13888,
		"org_scope_id": 5,
		"__entity_role__": null,
		"__entity_type__": "org_control",
		"__permissions__": [],
		"associations": {"policies": 3, "procedures": 1, "evidence": 5, "risks": 2},
		"recommended_evidence_count": 4,
		"org_evidence_metrics": {"total_count": 10, "complete_count": 8, "overdue_count": 2},
		"open_incident_count": 1,
		"org_evidence_count": 10,
		"org_evidence_collected_count": 8,
		"org_evidence_last_collected": "2025-01-15"
	}`

	var cd ControlDetails
	require.NoError(t, json.Unmarshal([]byte(input), &cd))

	assert.Equal(t, 778805, cd.ID)
	require.NotNil(t, cd.Associations)
	assert.Equal(t, 3, cd.Associations.Policies)
	assert.Equal(t, 5, cd.Associations.Evidence)
	require.NotNil(t, cd.RecommendedEvidenceCount)
	assert.Equal(t, 4, *cd.RecommendedEvidenceCount)
	require.NotNil(t, cd.OrgEvidenceMetrics)
	assert.Equal(t, 10, cd.OrgEvidenceMetrics.TotalCount)
	require.NotNil(t, cd.OpenIncidentCount)
	assert.Equal(t, 1, *cd.OpenIncidentCount)
	require.NotNil(t, cd.OrgEvidenceCount)
	assert.Equal(t, 10, *cd.OrgEvidenceCount)
}

func TestFrameworkCode_IntOrStringID(t *testing.T) {
	t.Parallel()

	// Integer framework_id
	input := `{"framework_id": 42, "code": "CC6.1", "framework_name": "SOC 2"}`
	var fc FrameworkCode
	require.NoError(t, json.Unmarshal([]byte(input), &fc))
	assert.Equal(t, "42", fc.ID.String())
	assert.Equal(t, "CC6.1", fc.Code)

	// String framework_id
	input2 := `{"framework_id": "42", "code": "CC6.1", "framework_name": "SOC 2"}`
	var fc2 FrameworkCode
	require.NoError(t, json.Unmarshal([]byte(input2), &fc2))
	assert.Equal(t, "42", fc2.ID.String())
}

// ---------------------------------------------------------------------------
// EvidenceTask model tests
// ---------------------------------------------------------------------------

func TestEvidenceTask_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	input := `{
		"id": 327992,
		"name": "GitHub Repository Access Controls",
		"description": "Show team permissions",
		"collection_interval": "quarter",
		"completed": false,
		"last_collected": "2025-01-15",
		"next_due": "2025-04-15",
		"priority": "high",
		"framework": "SOC2",
		"controls": ["CC6.1", "CC6.3"],
		"assignees": ["user1", "user2"],
		"tags": ["infrastructure", "github"],
		"status": "pending",
		"created_at": "2021-12-17T14:42:01Z",
		"updated_at": "2025-01-15T10:00:00Z",
		"aec_status": null
	}`

	var et EvidenceTask
	require.NoError(t, json.Unmarshal([]byte(input), &et))

	assert.Equal(t, 327992, et.ID)
	assert.Equal(t, "GitHub Repository Access Controls", et.Name)
	assert.Equal(t, "quarter", et.CollectionInterval)
	assert.False(t, et.Completed)
	require.NotNil(t, et.LastCollected)
	assert.Equal(t, "2025-01-15", *et.LastCollected)
	assert.Equal(t, "high", et.Priority)
	assert.Equal(t, "SOC2", et.Framework)
	require.Len(t, et.Controls, 2)
	assert.Equal(t, "CC6.1", et.Controls[0])
	assert.Equal(t, "pending", et.Status)

	// Marshal back
	out, err := json.Marshal(et)
	require.NoError(t, err)
	assert.Contains(t, string(out), `"id":327992`)
}

func TestEvidenceTask_AssigneesAsObjects(t *testing.T) {
	t.Parallel()

	// When embeds are used, assignees come as objects
	input := `{
		"id": 1,
		"name": "Test Task",
		"description": "",
		"collection_interval": "year",
		"completed": false,
		"priority": "",
		"framework": "",
		"controls": [],
		"assignees": [{"id": 123, "name": "John Doe", "email": "john@example.com", "role": "assignee", "assigned_at": "2025-01-01"}],
		"tags": [{"id": "t1", "name": "infrastructure", "color": "#00ff00"}],
		"status": "pending",
		"created_at": "",
		"updated_at": "",
		"aec_status": {"id": "aec1", "status": "active"}
	}`

	var et EvidenceTask
	require.NoError(t, json.Unmarshal([]byte(input), &et))
	assert.Equal(t, 1, et.ID)

	// Assignees stored as interface{} - should be parseable
	assigneesJSON, err := json.Marshal(et.Assignees)
	require.NoError(t, err)
	assert.Contains(t, string(assigneesJSON), "John Doe")

	// Tags stored as interface{} - should be parseable
	tagsJSON, err := json.Marshal(et.Tags)
	require.NoError(t, err)
	assert.Contains(t, string(tagsJSON), "infrastructure")

	// aec_status stored as interface{} - should be parseable
	aecJSON, err := json.Marshal(et.AecStatus)
	require.NoError(t, err)
	assert.Contains(t, string(aecJSON), "active")
}

func TestEvidenceTask_NullableFields(t *testing.T) {
	t.Parallel()

	input := `{
		"id": 1,
		"name": "Minimal Task",
		"description": "",
		"collection_interval": "",
		"completed": false,
		"last_collected": null,
		"next_due": null,
		"priority": "",
		"framework": "",
		"controls": null,
		"assignees": null,
		"tags": null,
		"status": "",
		"created_at": "",
		"updated_at": "",
		"aec_status": null
	}`

	var et EvidenceTask
	require.NoError(t, json.Unmarshal([]byte(input), &et))

	assert.Nil(t, et.LastCollected)
	assert.Nil(t, et.NextDue)
	assert.Nil(t, et.Controls)
	assert.Nil(t, et.Assignees)
	assert.Nil(t, et.Tags)
	assert.Nil(t, et.AecStatus)
}

func TestEvidenceTaskDetails_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	input := `{
		"id": 327992,
		"name": "GitHub Repository Access Controls",
		"description": "Show team permissions",
		"collection_interval": "quarter",
		"completed": false,
		"priority": "high",
		"framework": "SOC2",
		"controls": ["CC6.1"],
		"status": "pending",
		"created_at": "2021-12-17T14:42:01Z",
		"updated_at": "2025-01-15T10:00:00Z",
		"master_content": {
			"id": 500,
			"name": "GitHub Access",
			"description": "Collect GitHub access data",
			"guidance": "Use GitHub API to extract permissions",
			"version": 1
		},
		"org_scope": {"id": 5, "name": "Production", "description": "Prod env", "type": "environment"},
		"tags": [{"id": "t1", "name": "github"}],
		"assignees": [{"id": 10, "name": "Jane", "email": "jane@example.com", "role": "owner", "assigned_at": "2025-01-01"}],
		"subtask_metadata": {"total_subtasks": 3, "completed_subtasks": 1, "pending_subtasks": 2, "overdue_subtasks": 0},
		"open_incident_count": 0,
		"parent_org_controls": [{"id": 778805, "name": "Logical Access", "body": "", "category": "", "status": "implemented", "master_version_num": 0, "master_control_id": 0, "org_id": 0, "org_scope_id": 0, "__entity_type__": "org_control", "__permissions__": []}],
		"__entity_role__": null,
		"__entity_type__": "org_evidence",
		"__permissions__": ["view"]
	}`

	var etd EvidenceTaskDetails
	require.NoError(t, json.Unmarshal([]byte(input), &etd))

	assert.Equal(t, 327992, etd.ID)
	assert.Equal(t, "GitHub Repository Access Controls", etd.Name)
	require.NotNil(t, etd.MasterContent)
	assert.Equal(t, "Use GitHub API to extract permissions", etd.MasterContent.Guidance)
	require.NotNil(t, etd.OrgScope)
	assert.Equal(t, "Production", etd.OrgScope.Name)
	require.Len(t, etd.Tags, 1)
	assert.Equal(t, "github", etd.Tags[0].Name)
	require.Len(t, etd.Assignees, 1)
	assert.Equal(t, "Jane", etd.Assignees[0].Name)
	require.NotNil(t, etd.SubtaskMetadata)
	assert.Equal(t, 3, etd.SubtaskMetadata.TotalSubtasks)
	require.Len(t, etd.ParentOrgControls, 1)
	assert.Equal(t, 778805, etd.ParentOrgControls[0].ID)
	assert.Equal(t, "org_evidence", etd.EntityType)
}

// ---------------------------------------------------------------------------
// EvidenceAttachment / Submission model tests
// ---------------------------------------------------------------------------

func TestEvidenceAttachment_JSONRoundTrip(t *testing.T) {
	t.Parallel()

	input := `{
		"id": 50001,
		"__entity_type__": "org_evidence_attachment",
		"__entity_role__": null,
		"__permissions__": ["view", "delete"],
		"created": "2025-01-20T10:00:00Z",
		"collected": "2025-01-20",
		"notes": "Quarterly review of GitHub access",
		"url": "",
		"type": "file",
		"deleted": false,
		"org_id": 13888,
		"org_evidence_id": 327992,
		"org_evidence_subtask_id": null,
		"owner_id": 100,
		"attachment_id": 9001,
		"automated_ref": null,
		"automated_metadata": null,
		"integration_type": "github",
		"integration_subtype": "permissions",
		"attachment": {
			"id": 9001,
			"__entity_type__": "attachment",
			"__entity_role__": null,
			"__permissions__": ["view"],
			"created": "2025-01-20T10:00:00Z",
			"updated": "2025-01-20T10:00:00Z",
			"mime_type": "text/csv",
			"attachment_type": "org_evidence",
			"original_filename": "github_access.csv",
			"org_id": 13888,
			"deleted": false,
			"url": ""
		},
		"owner": {
			"id": 100,
			"__entity_type__": "org_member",
			"first_name": "Jane",
			"last_name": "Smith",
			"display_name": "Jane Smith",
			"short_display_name": "JS",
			"email": "jane@example.com",
			"has_reviewed_name": true,
			"is_owner": false,
			"created": "2020-01-01T00:00:00Z",
			"updated": "2025-01-01T00:00:00Z",
			"suspended": false,
			"training_completed": null,
			"training_expiry_date": null,
			"pendo_enabled": true,
			"org_id": 13888,
			"org": 13888,
			"user_id": 200,
			"attachment_id": null,
			"org_role_id": 1,
			"attachment": null
		},
		"collected_in_window": true,
		"org_evidence_subtask": null
	}`

	var ea EvidenceAttachment
	require.NoError(t, json.Unmarshal([]byte(input), &ea))

	assert.Equal(t, 50001, ea.ID)
	assert.Equal(t, "org_evidence_attachment", ea.EntityType)
	assert.Equal(t, "file", ea.Type)
	assert.False(t, ea.Deleted)
	assert.Equal(t, 327992, ea.OrgEvidenceID)
	assert.Nil(t, ea.OrgEvidenceSubtaskID)
	assert.Equal(t, "github", ea.IntegrationType)
	require.NotNil(t, ea.Attachment)
	assert.Equal(t, "github_access.csv", ea.Attachment.OriginalFilename)
	assert.Equal(t, "text/csv", ea.Attachment.MimeType)
	require.NotNil(t, ea.Owner)
	assert.Equal(t, "Jane Smith", ea.Owner.DisplayName)
	assert.True(t, ea.CollectedInWindow)

	// Marshal back
	out, err := json.Marshal(ea)
	require.NoError(t, err)
	assert.Contains(t, string(out), `"id":50001`)
}

func TestEvidenceAttachmentListResponse_JSONParse(t *testing.T) {
	t.Parallel()

	input := `{
		"max_page_size": 200,
		"page_size": 25,
		"num_pages": 3,
		"page_number": 1,
		"count": 55,
		"next": 2,
		"previous": null,
		"results": [
			{
				"id": 1,
				"__entity_type__": "org_evidence_attachment",
				"created": "2025-01-01T00:00:00Z",
				"collected": "2025-01-01",
				"notes": "",
				"url": "",
				"type": "file",
				"deleted": false,
				"org_id": 1,
				"org_evidence_id": 1,
				"owner_id": 1,
				"integration_type": "",
				"integration_subtype": "",
				"collected_in_window": true
			}
		]
	}`

	var resp EvidenceAttachmentListResponse
	require.NoError(t, json.Unmarshal([]byte(input), &resp))

	assert.Equal(t, 200, resp.MaxPageSize)
	assert.Equal(t, 25, resp.PageSize)
	assert.Equal(t, 3, resp.NumPages)
	assert.Equal(t, 1, resp.PageNumber)
	assert.Equal(t, 55, resp.Count)
	require.NotNil(t, resp.Next)
	assert.Equal(t, 2, *resp.Next)
	assert.Nil(t, resp.Previous)
	require.Len(t, resp.Results, 1)
	assert.Equal(t, 1, resp.Results[0].ID)
}

func TestAttachmentDownloadResponse_JSON(t *testing.T) {
	t.Parallel()

	input := `{
		"url": "https://s3.amazonaws.com/bucket/file.csv?token=xyz",
		"original_filename": "evidence_report.csv"
	}`

	var adr AttachmentDownloadResponse
	require.NoError(t, json.Unmarshal([]byte(input), &adr))

	assert.Equal(t, "https://s3.amazonaws.com/bucket/file.csv?token=xyz", adr.URL)
	assert.Equal(t, "evidence_report.csv", adr.OriginalFilename)
}

// ---------------------------------------------------------------------------
// Additional supporting types
// ---------------------------------------------------------------------------

func TestMasterContent_JSON(t *testing.T) {
	t.Parallel()

	input := `{
		"id": 500,
		"name": "GitHub Access",
		"description": "Collect GitHub access data",
		"content": "Detailed content here",
		"guidance": "Use GitHub API to extract permissions",
		"help": "Contact security team for access",
		"version": 2
	}`

	var mc MasterContent
	require.NoError(t, json.Unmarshal([]byte(input), &mc))
	assert.Equal(t, "Use GitHub API to extract permissions", mc.Guidance)
	assert.Equal(t, "Contact security team for access", mc.Help)
	assert.Equal(t, 2, mc.Version)
}

func TestAecStatus_JSON(t *testing.T) {
	t.Parallel()

	input := `{
		"id": "aec-001",
		"status": "active",
		"last_executed": "2025-01-10T08:00:00Z",
		"next_scheduled": "2025-02-10T08:00:00Z",
		"error_message": "",
		"successful_runs": 10,
		"failed_runs": 1,
		"last_successful_run": "2025-01-10T08:00:00Z"
	}`

	var as AecStatus
	require.NoError(t, json.Unmarshal([]byte(input), &as))

	assert.Equal(t, "active", as.Status)
	assert.Equal(t, 10, as.SuccessfulRuns)
	assert.Equal(t, 1, as.FailedRuns)
	require.NotNil(t, as.LastExecuted)
	assert.Equal(t, 2025, as.LastExecuted.Year())
}

func TestSubtaskMetadata_JSON(t *testing.T) {
	t.Parallel()

	input := `{
		"total_subtasks": 5,
		"completed_subtasks": 3,
		"pending_subtasks": 1,
		"overdue_subtasks": 1
	}`

	var sm SubtaskMetadata
	require.NoError(t, json.Unmarshal([]byte(input), &sm))
	assert.Equal(t, 5, sm.TotalSubtasks)
	assert.Equal(t, 3, sm.CompletedSubtasks)
	assert.Equal(t, 1, sm.OverdueSubtasks)
}

func TestPolicySummary_JSON(t *testing.T) {
	t.Parallel()

	input := `{
		"total": 40,
		"by_framework": {"SOC2": 25, "ISO27001": 15},
		"by_status": {"active": 35, "draft": 5},
		"last_sync": "2025-01-15T10:00:00Z"
	}`

	var ps PolicySummary
	require.NoError(t, json.Unmarshal([]byte(input), &ps))
	assert.Equal(t, 40, ps.Total)
	assert.Equal(t, 25, ps.ByFramework["SOC2"])
	assert.Equal(t, 35, ps.ByStatus["active"])
}

func TestControlSummary_JSON(t *testing.T) {
	t.Parallel()

	input := `{
		"total": 100,
		"by_framework": {"SOC2": 60, "ISO27001": 40},
		"by_status": {"implemented": 80, "na": 20},
		"by_category": {"Common Criteria": 50, "Trust Services": 50},
		"last_sync": "2025-01-15"
	}`

	var cs ControlSummary
	require.NoError(t, json.Unmarshal([]byte(input), &cs))
	assert.Equal(t, 100, cs.Total)
	assert.Equal(t, 80, cs.ByStatus["implemented"])
}

func TestEvidenceTaskSummary_JSON(t *testing.T) {
	t.Parallel()

	input := `{
		"total": 150,
		"by_status": {"pending": 50, "completed": 80, "in_progress": 20},
		"by_priority": {"high": 30, "medium": 80, "low": 40},
		"overdue": 5,
		"due_soon": 15,
		"last_sync": "2025-01-15T10:00:00Z"
	}`

	var es EvidenceTaskSummary
	require.NoError(t, json.Unmarshal([]byte(input), &es))
	assert.Equal(t, 150, es.Total)
	assert.Equal(t, 5, es.Overdue)
	assert.Equal(t, 15, es.DueSoon)
	assert.Equal(t, 30, es.ByPriority["high"])
}

func TestEvidenceAssignee_FlexibleID(t *testing.T) {
	t.Parallel()

	// Integer ID
	input := `{"id": 123, "name": "John", "email": "john@example.com", "role": "owner", "assigned_at": "2025-01-01"}`
	var ea EvidenceAssignee
	require.NoError(t, json.Unmarshal([]byte(input), &ea))
	assert.Equal(t, "John", ea.Name)

	// String ID
	input2 := `{"id": "abc-123", "name": "Jane", "email": "jane@example.com", "role": "assignee", "assigned_at": "2025-01-01"}`
	var ea2 EvidenceAssignee
	require.NoError(t, json.Unmarshal([]byte(input2), &ea2))
	assert.Equal(t, "Jane", ea2.Name)
}

func TestInternalAec_JSON(t *testing.T) {
	t.Parallel()

	input := `{
		"id": "iaec-001",
		"name": "GitHub Scanner",
		"type": "scanner",
		"configuration": {"repo": "org/main"},
		"enabled": true,
		"last_executed": "2025-01-10T08:00:00Z",
		"schedule": "0 0 * * 1"
	}`

	var ia InternalAec
	require.NoError(t, json.Unmarshal([]byte(input), &ia))
	assert.Equal(t, "GitHub Scanner", ia.Name)
	assert.True(t, ia.Enabled)
	assert.Equal(t, "0 0 * * 1", ia.Schedule)
}

func TestSupportedIntegration_JSON(t *testing.T) {
	t.Parallel()

	input := `{
		"id": "int-001",
		"name": "GitHub",
		"type": "vcs",
		"description": "Version control integration",
		"enabled": true,
		"config": {"org": "myorg"}
	}`

	var si SupportedIntegration
	require.NoError(t, json.Unmarshal([]byte(input), &si))
	assert.Equal(t, "GitHub", si.Name)
	assert.True(t, si.Enabled)
	assert.Equal(t, "myorg", si.Config["org"])
}

func TestEvidenceAttachmentListOptions_JSON(t *testing.T) {
	t.Parallel()

	opts := EvidenceAttachmentListOptions{
		OrgEvidence:       327992,
		ObservationPeriod: "2013-01-01,2025-12-31",
		Ordering:          "-collected,-created",
		Page:              1,
		PageSize:          25,
		Embeds:            []string{"attachment", "org_members"},
		Type:              "file",
	}

	data, err := json.Marshal(opts)
	require.NoError(t, err)
	assert.Contains(t, string(data), `"org_evidence":327992`)
	assert.Contains(t, string(data), `"ordering":"-collected,-created"`)
}
