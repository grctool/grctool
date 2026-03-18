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

package testhelpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPolicyFixture(t *testing.T) {
	t.Parallel()

	p, err := LoadPolicyFixture("sample_policy")
	require.NoError(t, err)
	assert.Equal(t, "12345", p.ID)
	assert.Equal(t, "POL-0001", p.ReferenceID)
	assert.Equal(t, "Access Control Policy", p.Name)
	assert.Equal(t, "SOC2", p.Framework)
	assert.Equal(t, "active", p.Status)
}

func TestLoadControlFixture(t *testing.T) {
	t.Parallel()

	c, err := LoadControlFixture("sample_control")
	require.NoError(t, err)
	assert.Equal(t, "1001", c.ID)
	assert.Equal(t, "CC-06.1", c.ReferenceID)
	assert.Equal(t, "Logical Access Security", c.Name)
	assert.Equal(t, "SOC2", c.Framework)
	assert.Equal(t, "implemented", c.Status)
}

func TestLoadEvidenceTaskFixture(t *testing.T) {
	t.Parallel()

	et, err := LoadEvidenceTaskFixture("sample_task")
	require.NoError(t, err)
	assert.Equal(t, "327992", et.ID)
	assert.Equal(t, "ET-0047", et.ReferenceID)
	assert.Equal(t, "GitHub Repository Access Controls", et.Name)
	assert.Equal(t, "SOC2", et.Framework)
	assert.Equal(t, "pending", et.Status)
	assert.Len(t, et.Controls, 2)
}

func TestLoadFixture_NotFound(t *testing.T) {
	t.Parallel()

	_, err := LoadFixture("does_not_exist.json")
	assert.Error(t, err)

	_, err = LoadPolicyFixture("nonexistent")
	assert.Error(t, err)
}

func TestSamplePolicy(t *testing.T) {
	t.Parallel()

	p := SamplePolicy()
	assert.NotEmpty(t, p.ID)
	assert.NotEmpty(t, p.ReferenceID)
	assert.NotEmpty(t, p.Name)
	assert.NotEmpty(t, p.Framework)
	assert.NotEmpty(t, p.Status)
	assert.False(t, p.CreatedAt.IsZero())
}

func TestSampleControl(t *testing.T) {
	t.Parallel()

	c := SampleControl()
	assert.NotZero(t, c.ID)
	assert.NotEmpty(t, c.ReferenceID)
	assert.NotEmpty(t, c.Name)
	assert.NotEmpty(t, c.Framework)
	assert.NotEmpty(t, c.Status)
}

func TestSampleEvidenceTask(t *testing.T) {
	t.Parallel()

	et := SampleEvidenceTask()
	assert.NotZero(t, et.ID)
	assert.NotEmpty(t, et.ReferenceID)
	assert.NotEmpty(t, et.Name)
	assert.NotEmpty(t, et.Framework)
	assert.NotEmpty(t, et.Status)
	assert.NotEmpty(t, et.Controls)
}

func TestSampleEvidenceRecord(t *testing.T) {
	t.Parallel()

	r := SampleEvidenceRecord()
	assert.NotEmpty(t, r.ID)
	assert.NotEmpty(t, r.TaskID)
	assert.NotEmpty(t, r.Title)
	assert.NotEmpty(t, r.Source)
	assert.False(t, r.CollectedAt.IsZero())
}
