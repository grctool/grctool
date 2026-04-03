// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package gdrive

import (
	"testing"

	"github.com/grctool/grctool/internal/domain"
	"github.com/stretchr/testify/assert"
)

func TestSyncScope_PolicyInScope_Enabled(t *testing.T) {
	t.Parallel()
	scope := DefaultSyncScope()
	pol := &domain.Policy{ReferenceID: "POL-0001"}
	assert.True(t, scope.PolicyInScope(pol))
}

func TestSyncScope_PolicyInScope_Disabled(t *testing.T) {
	t.Parallel()
	scope := SyncScope{Policies: EntityScope{Enabled: false}}
	pol := &domain.Policy{ReferenceID: "POL-0001"}
	assert.False(t, scope.PolicyInScope(pol))
}

func TestSyncScope_PolicyInScope_IncludePattern(t *testing.T) {
	t.Parallel()
	scope := SyncScope{
		Policies: EntityScope{
			Enabled: true,
			Include: []string{"POL-000*"},
		},
	}
	assert.True(t, scope.PolicyInScope(&domain.Policy{ReferenceID: "POL-0001"}))
	assert.True(t, scope.PolicyInScope(&domain.Policy{ReferenceID: "POL-0009"}))
	assert.False(t, scope.PolicyInScope(&domain.Policy{ReferenceID: "POL-0010"}))
}

func TestSyncScope_PolicyInScope_ExcludePattern(t *testing.T) {
	t.Parallel()
	scope := SyncScope{
		Policies: EntityScope{
			Enabled: true,
			Exclude: []string{"POL-DRAFT-*"},
		},
	}
	assert.True(t, scope.PolicyInScope(&domain.Policy{ReferenceID: "POL-0001"}))
	assert.False(t, scope.PolicyInScope(&domain.Policy{ReferenceID: "POL-DRAFT-001"}))
}

func TestSyncScope_PolicyInScope_TagsInclude(t *testing.T) {
	t.Parallel()
	scope := SyncScope{
		Policies: EntityScope{
			Enabled:     true,
			TagsInclude: []string{"published"},
		},
	}
	assert.True(t, scope.PolicyInScope(&domain.Policy{
		ReferenceID: "POL-0001",
		Tags:        []domain.Tag{{Name: "published"}},
	}))
	assert.False(t, scope.PolicyInScope(&domain.Policy{
		ReferenceID: "POL-0002",
		Tags:        []domain.Tag{{Name: "draft"}},
	}))
}

func TestSyncScope_PolicyInScope_TagsExclude(t *testing.T) {
	t.Parallel()
	scope := SyncScope{
		Policies: EntityScope{
			Enabled:     true,
			TagsExclude: []string{"draft", "internal"},
		},
	}
	assert.True(t, scope.PolicyInScope(&domain.Policy{
		ReferenceID: "POL-0001",
		Tags:        []domain.Tag{{Name: "published"}},
	}))
	assert.False(t, scope.PolicyInScope(&domain.Policy{
		ReferenceID: "POL-0002",
		Tags:        []domain.Tag{{Name: "draft"}},
	}))
	assert.False(t, scope.PolicyInScope(&domain.Policy{
		ReferenceID: "POL-0003",
		Tags:        []domain.Tag{{Name: "internal"}},
	}))
}

func TestSyncScope_PolicyInScope_CombinedFilters(t *testing.T) {
	t.Parallel()
	scope := SyncScope{
		Policies: EntityScope{
			Enabled:     true,
			Include:     []string{"POL-*"},
			Exclude:     []string{"POL-DRAFT-*"},
			TagsExclude: []string{"sensitive"},
		},
	}
	// Matches include, not excluded
	assert.True(t, scope.PolicyInScope(&domain.Policy{ReferenceID: "POL-0001"}))
	// Matches exclude pattern
	assert.False(t, scope.PolicyInScope(&domain.Policy{ReferenceID: "POL-DRAFT-001"}))
	// Has excluded tag
	assert.False(t, scope.PolicyInScope(&domain.Policy{
		ReferenceID: "POL-0002",
		Tags:        []domain.Tag{{Name: "sensitive"}},
	}))
}

func TestSyncScope_ControlInScope(t *testing.T) {
	t.Parallel()
	scope := SyncScope{
		Controls: EntityScope{
			Enabled: true,
			Include: []string{"CC-*"},
		},
	}
	assert.True(t, scope.ControlInScope(&domain.Control{ReferenceID: "CC-06.1"}))
	assert.False(t, scope.ControlInScope(&domain.Control{ReferenceID: "SO-19"}))
}

func TestSyncScope_ControlInScope_Disabled(t *testing.T) {
	t.Parallel()
	scope := SyncScope{Controls: EntityScope{Enabled: false}}
	assert.False(t, scope.ControlInScope(&domain.Control{ReferenceID: "CC-06.1"}))
}

func TestSyncScope_EvidenceTaskInScope(t *testing.T) {
	t.Parallel()
	scope := SyncScope{
		EvidenceTasks: EntityScope{
			Enabled:     true,
			TagsExclude: []string{"deprecated"},
		},
	}
	assert.True(t, scope.EvidenceTaskInScope(&domain.EvidenceTask{ReferenceID: "ET-0047"}))
	assert.False(t, scope.EvidenceTaskInScope(&domain.EvidenceTask{
		ReferenceID: "ET-0001",
		Tags:        []domain.Tag{{Name: "deprecated"}},
	}))
}

func TestSyncScope_TagsCaseInsensitive(t *testing.T) {
	t.Parallel()
	scope := SyncScope{
		Policies: EntityScope{
			Enabled:     true,
			TagsExclude: []string{"Draft"},
		},
	}
	assert.False(t, scope.PolicyInScope(&domain.Policy{
		ReferenceID: "POL-0001",
		Tags:        []domain.Tag{{Name: "draft"}},
	}))
}

func TestSyncScope_EmptyRefID(t *testing.T) {
	t.Parallel()
	scope := SyncScope{
		Policies: EntityScope{
			Enabled: true,
			Include: []string{"POL-*"},
		},
	}
	// Empty reference ID doesn't match include pattern
	assert.False(t, scope.PolicyInScope(&domain.Policy{ReferenceID: ""}))
}

func TestDefaultSyncScope(t *testing.T) {
	t.Parallel()
	scope := DefaultSyncScope()
	assert.True(t, scope.Policies.Enabled)
	assert.True(t, scope.Controls.Enabled)
	assert.True(t, scope.EvidenceTasks.Enabled)
	assert.Empty(t, scope.Policies.Include)
	assert.Empty(t, scope.Policies.Exclude)
}
