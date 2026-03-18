package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRelationships_JSONRoundTrip(t *testing.T) {
	t.Parallel()
	rel := Relationships{
		PolicyToProcedures: map[string][]string{
			"POL-0001": {"PROC-0001", "PROC-0002"},
		},
		PolicyToControls: map[string][]string{
			"POL-0001": {"CC-06.1", "CC-06.3"},
		},
		ProcedureToTasks: map[string][]string{
			"PROC-0001": {"ET-0001", "ET-0002"},
		},
		ControlToTasks: map[string][]string{
			"CC-06.1": {"ET-0047"},
		},
		TaskToEvidence: map[string][]string{
			"ET-0047": {"ev-001", "ev-002"},
		},
	}

	data, err := json.Marshal(rel)
	require.NoError(t, err)

	var decoded Relationships
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Len(t, decoded.PolicyToProcedures["POL-0001"], 2)
	assert.Len(t, decoded.PolicyToControls["POL-0001"], 2)
	assert.Len(t, decoded.ControlToTasks["CC-06.1"], 1)
	assert.Len(t, decoded.TaskToEvidence["ET-0047"], 2)
}

func TestRelationships_EmptyMaps(t *testing.T) {
	t.Parallel()
	rel := Relationships{}

	data, err := json.Marshal(rel)
	require.NoError(t, err)

	var decoded Relationships
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Nil(t, decoded.PolicyToProcedures)
	assert.Nil(t, decoded.PolicyToControls)
}
