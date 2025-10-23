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

//go:build !e2e && !functional

package helpers

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEvidenceTaskBuilder(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup    func() *EvidenceTaskBuilder
		validate func(t *testing.T, result interface{})
	}{
		"default builder creates valid task": {
			setup: func() *EvidenceTaskBuilder {
				return NewEvidenceTaskBuilder()
			},
			validate: func(t *testing.T, result interface{}) {
				task := result.(*EvidenceTaskBuilder).Build()
				assert.Equal(t, 1001, task.ID)
				assert.Equal(t, "Test Evidence Task", task.Name)
				assert.Equal(t, "quarterly", task.CollectionInterval)
				assert.False(t, task.AdHoc)
				assert.False(t, task.Sensitive)
			},
		},
		"builder with custom values": {
			setup: func() *EvidenceTaskBuilder {
				return NewEvidenceTaskBuilder().
					WithID(2001).
					WithName("Custom Task").
					WithCollectionInterval("monthly").
					WithSensitive(true)
			},
			validate: func(t *testing.T, result interface{}) {
				task := result.(*EvidenceTaskBuilder).Build()
				assert.Equal(t, 2001, task.ID)
				assert.Equal(t, "Custom Task", task.Name)
				assert.Equal(t, "monthly", task.CollectionInterval)
				assert.True(t, task.Sensitive)
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			builder := tt.setup()
			tt.validate(t, builder)
		})
	}
}

func TestPolicyBuilder(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		setup    func() *PolicyBuilder
		validate func(t *testing.T, result interface{})
	}{
		"default builder creates valid policy": {
			setup: func() *PolicyBuilder {
				return NewPolicyBuilder()
			},
			validate: func(t *testing.T, result interface{}) {
				policy := result.(*PolicyBuilder).Build()
				assert.Equal(t, "2001", policy.ID.String())
				assert.Equal(t, "Test Security Policy", policy.Name)
				assert.Equal(t, "SOC2", policy.Framework)
				assert.Equal(t, "active", policy.Status)
			},
		},
		"builder with string ID": {
			setup: func() *PolicyBuilder {
				return NewPolicyBuilder().
					WithStringID("P001").
					WithName("Custom Policy").
					WithFramework("ISO27001")
			},
			validate: func(t *testing.T, result interface{}) {
				policy := result.(*PolicyBuilder).Build()
				assert.Equal(t, "P001", policy.ID.String())
				assert.Equal(t, "Custom Policy", policy.Name)
				assert.Equal(t, "ISO27001", policy.Framework)
			},
		},
	}

	for name, tt := range tests {
		tt := tt
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			builder := tt.setup()
			tt.validate(t, builder)
		})
	}
}

func TestControlBuilder(t *testing.T) {
	t.Parallel()

	builder := NewControlBuilder().
		WithID(3001).
		WithName("Access Control").
		WithCategory("Security").
		WithAutoImplemented(true)

	control := builder.Build()

	assert.Equal(t, 3001, control.ID)
	assert.Equal(t, "Access Control", control.Name)
	assert.Equal(t, "Security", control.Category)
	assert.True(t, control.IsAutoImplemented)
}

func TestEvidenceAssigneeBuilder(t *testing.T) {
	t.Parallel()

	assignee := NewEvidenceAssigneeBuilder().
		WithName("John Doe").
		WithEmail("john@example.com").
		WithRole("analyst").
		Build()

	assert.Equal(t, "John Doe", assignee.Name)
	assert.Equal(t, "john@example.com", assignee.Email)
	assert.Equal(t, "analyst", assignee.Role)
}

func TestEvidenceTagBuilder(t *testing.T) {
	t.Parallel()

	tag := NewEvidenceTagBuilder().
		WithName("security").
		WithColor("#ff0000").
		Build()

	assert.Equal(t, "security", tag.Name)
	assert.Equal(t, "#ff0000", tag.Color)
}
