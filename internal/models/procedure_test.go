package models

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcedure_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	proc := Procedure{
		ID:          "proc-001",
		PolicyID:    "POL-0001",
		Name:        "Quarterly Access Review",
		Description: "Review all access quarterly",
		Frequency:   "quarterly",
		Status:      "active",
		Steps: []Step{
			{
				ID:          "step-1",
				Name:        "Pull access list",
				Description: "Export current access list",
				Order:       1,
				Required:    true,
				Status:      "completed",
			},
			{
				ID:          "step-2",
				Name:        "Review with manager",
				Description: "Manager reviews access list",
				Order:       2,
				Required:    true,
				Status:      "pending",
			},
			{
				ID:          "step-3",
				Name:        "Archive results",
				Description: "Save review results",
				Order:       3,
				Required:    false,
				Status:      "pending",
			},
		},
		CreatedAt: now,
		UpdatedAt: now,
	}

	data, err := json.Marshal(proc)
	require.NoError(t, err)

	var decoded Procedure
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, proc.ID, decoded.ID)
	assert.Equal(t, proc.PolicyID, decoded.PolicyID)
	assert.Equal(t, proc.Frequency, decoded.Frequency)
	assert.Len(t, decoded.Steps, 3)
	assert.Equal(t, 1, decoded.Steps[0].Order)
	assert.True(t, decoded.Steps[0].Required)
	assert.False(t, decoded.Steps[2].Required)
}

func TestProcedureSummary_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	now := time.Now().UTC().Truncate(time.Second)
	ps := ProcedureSummary{
		Total:       5,
		ByFrequency: map[string]int{"quarterly": 3, "monthly": 2},
		ByStatus:    map[string]int{"active": 4, "draft": 1},
		LastSync:    now,
	}

	data, err := json.Marshal(ps)
	require.NoError(t, err)

	var decoded ProcedureSummary
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, 5, decoded.Total)
	assert.Equal(t, 3, decoded.ByFrequency["quarterly"])
}
