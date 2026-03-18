package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPolicy_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	policy := Policy{
		ID:          IntOrString("12345"),
		Name:        "Access Control Policy",
		Description: "Defines access controls",
		Framework:   "SOC2",
		Status:      "active",
		Controls: []Control{
			{
				ID:       1001,
				Name:     "Logical Access",
				Category: "Common Criteria",
				Status:   "implemented",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(policy)
	require.NoError(t, err)

	var decoded Policy
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, IntOrString("12345"), decoded.ID)
	assert.Equal(t, policy.Name, decoded.Name)
	assert.Equal(t, policy.Framework, decoded.Framework)
	assert.Len(t, decoded.Controls, 1)
	assert.Equal(t, 1001, decoded.Controls[0].ID)
}

func TestPolicy_IDAsIntOrString(t *testing.T) {
	t.Parallel()

	t.Run("numeric ID", func(t *testing.T) {
		t.Parallel()
		p := Policy{ID: IntOrString("42")}
		assert.Equal(t, "42", p.ID.String())
		assert.Equal(t, 42, p.ID.ToInt())
	})

	t.Run("string ID", func(t *testing.T) {
		t.Parallel()
		p := Policy{ID: IntOrString("POL-0001")}
		assert.Equal(t, "POL-0001", p.ID.String())
		assert.Equal(t, 0, p.ID.ToInt())
	})
}

func TestControl_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	control := Control{
		ID:                1001,
		Name:              "Logical Access Security",
		Body:              "Implements logical access controls",
		Category:          "Common Criteria",
		Status:            "implemented",
		Risk:              "medium",
		IsAutoImplemented: true,
		Codes:             "CC6.1",
		MasterVersionNum:  1,
		MasterControlID:   500,
		OrgID:             100,
		OrgScopeID:        10,
		Framework:         "SOC2",
		FrameworkCodes: []FrameworkCode{
			{MasterControlID: 500, FrameworkName: "SOC 2", FrameworkID: 1, Code: "CC6.1"},
		},
		OrgScope: &OrgScope{
			ID:        10,
			OrgID:     100,
			Name:      "Global",
			ScopeType: "global",
		},
		Tags: []ControlTag{
			{ID: "t1", Name: "security"},
		},
	}

	data, err := json.Marshal(control)
	require.NoError(t, err)

	var decoded Control
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, control.ID, decoded.ID)
	assert.Equal(t, control.Name, decoded.Name)
	assert.Equal(t, control.Body, decoded.Body)
	assert.True(t, decoded.IsAutoImplemented)
	assert.Len(t, decoded.FrameworkCodes, 1)
	assert.Equal(t, "CC6.1", decoded.FrameworkCodes[0].Code)
	assert.NotNil(t, decoded.OrgScope)
	assert.Equal(t, "Global", decoded.OrgScope.Name)
	assert.Len(t, decoded.Tags, 1)
}

func TestPolicyDetails_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	pd := PolicyDetails{
		Policy: Policy{
			ID:        IntOrString("1"),
			Name:      "Test Policy",
			Framework: "SOC2",
			Status:    "active",
		},
		CurrentVersion: &PolicyVersion{
			ID:        "v1",
			Version:   "1.0",
			Content:   "Policy content",
			Status:    "published",
			CreatedAt: now,
			CreatedBy: "admin",
		},
		Tags: []PolicyTag{
			{ID: "t1", Name: "compliance", Color: "#0000ff"},
		},
		AssociationCounts: &AssociationCounts{
			Controls:   5,
			Procedures: 3,
			Evidence:   10,
			Risks:      2,
		},
		Assignees: []PolicyAssignee{
			{ID: "a1", Name: "Alice", Email: "alice@test.com", Role: "owner"},
		},
		Reviewers: []PolicyReviewer{
			{ID: "r1", Name: "Bob", Email: "bob@test.com", ReviewedAt: now},
		},
	}

	data, err := json.Marshal(pd)
	require.NoError(t, err)

	var decoded PolicyDetails
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, "Test Policy", decoded.Name)
	assert.NotNil(t, decoded.CurrentVersion)
	assert.Equal(t, "1.0", decoded.CurrentVersion.Version)
	assert.NotNil(t, decoded.AssociationCounts)
	assert.Equal(t, 5, decoded.AssociationCounts.Controls)
	assert.Len(t, decoded.Tags, 1)
	assert.Len(t, decoded.Assignees, 1)
	assert.Len(t, decoded.Reviewers, 1)
}

func TestControlDetails_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	cd := ControlDetails{
		Control: Control{
			ID:       1001,
			Name:     "Test Control",
			Status:   "implemented",
			Category: "CC",
		},
		DeprecationNotes: "Will be removed in v2",
	}

	data, err := json.Marshal(cd)
	require.NoError(t, err)

	var decoded ControlDetails
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 1001, decoded.ID)
	assert.Equal(t, "Will be removed in v2", decoded.DeprecationNotes)
}

func TestPolicySummary_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	ps := PolicySummary{
		Total:       10,
		ByFramework: map[string]int{"SOC2": 7, "ISO27001": 3},
		ByStatus:    map[string]int{"active": 8, "draft": 2},
		LastSync:    now,
	}

	data, err := json.Marshal(ps)
	require.NoError(t, err)

	var decoded PolicySummary
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 10, decoded.Total)
	assert.Equal(t, 7, decoded.ByFramework["SOC2"])
}

func TestControlSummary_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	cs := ControlSummary{
		Total:       20,
		ByFramework: map[string]int{"SOC2": 15},
		ByStatus:    map[string]int{"implemented": 18, "na": 2},
		ByCategory:  map[string]int{"CC": 10, "AC": 10},
		LastSync:    now,
	}

	data, err := json.Marshal(cs)
	require.NoError(t, err)

	var decoded ControlSummary
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 20, decoded.Total)
	assert.Equal(t, 15, decoded.ByFramework["SOC2"])
}

func TestControlAssociation_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	ca := ControlAssociation{
		ID:           "assoc-1",
		Type:         "implements",
		RelatedID:    "POL-0001",
		RelatedName:  "Access Control Policy",
		RelatedType:  "policy",
		Relationship: "implements",
	}

	data, err := json.Marshal(ca)
	require.NoError(t, err)

	var decoded ControlAssociation
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, ca.ID, decoded.ID)
	assert.Equal(t, ca.RelatedName, decoded.RelatedName)
}

func TestControlFilter_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	cf := ControlFilter{
		Status:    []string{"implemented"},
		Category:  []string{"CC"},
		Framework: "SOC2",
	}

	data, err := json.Marshal(cf)
	require.NoError(t, err)

	var decoded ControlFilter
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, cf.Status, decoded.Status)
	assert.Equal(t, cf.Framework, decoded.Framework)
}

func TestPolicyFilter_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	pf := PolicyFilter{
		Status:    []string{"active"},
		Type:      []string{"governance"},
		Framework: "SOC2",
	}

	data, err := json.Marshal(pf)
	require.NoError(t, err)

	var decoded PolicyFilter
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, pf.Status, decoded.Status)
	assert.Equal(t, pf.Type, decoded.Type)
}
