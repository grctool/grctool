// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// CollectionPlanManager — UpdateStrategy, AddGap
// ---------------------------------------------------------------------------

func newCollectionPlanManager(t *testing.T) *CollectionPlanManager {
	t.Helper()
	return NewCollectionPlanManager(newTestConfig(t.TempDir()), testhelpers.NewStubLogger())
}

func TestCollectionPlanManager_UpdateStrategy(t *testing.T) {
	t.Parallel()

	cpm := newCollectionPlanManager(t)
	plan := &CollectionPlan{TaskRef: "ET-0001"}

	cpm.UpdateStrategy(plan, "automated", "GitHub API provides all data")

	assert.Equal(t, "automated", plan.Strategy)
	assert.Equal(t, "GitHub API provides all data", plan.Reasoning)
	assert.False(t, plan.LastUpdated.IsZero())
}

func TestCollectionPlanManager_AddGap(t *testing.T) {
	t.Parallel()

	cpm := newCollectionPlanManager(t)
	plan := &CollectionPlan{TaskRef: "ET-0001", Gaps: []EvidenceGap{}}

	gap := EvidenceGap{
		Description: "Missing branch protection evidence",
		Priority:    "high",
		Remediation: "Enable branch protection and collect evidence",
	}
	cpm.AddGap(plan, gap)

	require.Len(t, plan.Gaps, 1)
	assert.Equal(t, "high", plan.Gaps[0].Priority)
	assert.False(t, plan.LastUpdated.IsZero())
}

// ---------------------------------------------------------------------------
// SavePlan + loadPlanFromFile round-trip
// ---------------------------------------------------------------------------

func TestCollectionPlanManager_SaveAndLoadPlan(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cpm := NewCollectionPlanManager(newTestConfig(dir), testhelpers.NewStubLogger())

	plan := &CollectionPlan{
		TaskRef:            "ET-0001",
		TaskName:           "GitHub Access Controls",
		TaskDescription:    "Collect access control evidence",
		Window:             "2025-Q1",
		CollectionInterval: "quarter",
		Strategy:           "automated",
		Reasoning:          "GitHub API available",
		Controls: []ControlRef{
			{ID: "1001", ReferenceID: "CC-06.1", Name: "Logical Access", Category: "Common Criteria"},
		},
		Entries: []EvidenceEntry{
			{Filename: "permissions.json", Title: "Permissions", Source: "github-api", Status: "complete"},
		},
		Gaps: []EvidenceGap{},
	}

	planPath := filepath.Join(dir, "evidence", "plan", "collection_plan.md")

	// Save
	err := cpm.SavePlan(plan, planPath)
	require.NoError(t, err)

	// Verify markdown file exists
	_, err = os.Stat(planPath)
	require.NoError(t, err, "markdown file should exist")

	// Verify metadata file exists
	metaPath := filepath.Join(dir, "evidence", "plan", "collection_plan_metadata.yaml")
	_, err = os.Stat(metaPath)
	require.NoError(t, err, "metadata file should exist")

	// Load back
	loaded, err := cpm.loadPlanFromFile(planPath)
	require.NoError(t, err)
	require.NotNil(t, loaded)

	assert.Equal(t, "ET-0001", loaded.TaskRef)
	assert.Equal(t, "GitHub Access Controls", loaded.TaskName)
	assert.Equal(t, "automated", loaded.Strategy)
	require.Len(t, loaded.Controls, 1)
	assert.Equal(t, "CC-06.1", loaded.Controls[0].ReferenceID)
	require.Len(t, loaded.Entries, 1)
	assert.Equal(t, "permissions.json", loaded.Entries[0].Filename)
}

// ---------------------------------------------------------------------------
// LoadOrCreatePlan
// ---------------------------------------------------------------------------

func TestCollectionPlanManager_LoadOrCreatePlan_NewPlan(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cpm := NewCollectionPlanManager(newTestConfig(dir), testhelpers.NewStubLogger())

	task := &domain.EvidenceTask{
		ReferenceID:        "ET-0047",
		Name:               "GitHub Repo Access",
		Description:        "Access control evidence",
		CollectionInterval: "quarter",
	}

	planPath := filepath.Join(dir, "nonexistent_plan.md")

	plan, err := cpm.LoadOrCreatePlan(task, "2025-Q1", planPath)
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, "ET-0047", plan.TaskRef)
	assert.Equal(t, "2025-Q1", plan.Window)
	assert.Equal(t, "Not Started", plan.Status)
	assert.Equal(t, 0.0, plan.Completeness)
}

func TestCollectionPlanManager_LoadOrCreatePlan_ExistingPlan(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	cpm := NewCollectionPlanManager(newTestConfig(dir), testhelpers.NewStubLogger())

	// First create and save a plan
	planPath := filepath.Join(dir, "plan.md")
	existingPlan := &CollectionPlan{
		TaskRef:  "ET-0001",
		TaskName: "Existing Plan",
		Window:   "2025-Q1",
		Strategy: "manual",
	}
	err := cpm.SavePlan(existingPlan, planPath)
	require.NoError(t, err)

	// Now load it
	task := &domain.EvidenceTask{ReferenceID: "ET-0001"}
	plan, err := cpm.LoadOrCreatePlan(task, "2025-Q1", planPath)
	require.NoError(t, err)
	require.NotNil(t, plan)

	assert.Equal(t, "ET-0001", plan.TaskRef)
	assert.Equal(t, "manual", plan.Strategy)
}

// ---------------------------------------------------------------------------
// generateMarkdownPlan
// ---------------------------------------------------------------------------

func TestCollectionPlanManager_GenerateMarkdownPlan(t *testing.T) {
	t.Parallel()

	cpm := newCollectionPlanManager(t)

	t.Run("with entries and gaps", func(t *testing.T) {
		t.Parallel()
		plan := &CollectionPlan{
			TaskRef:            "ET-0001",
			TaskName:           "GitHub Access",
			TaskDescription:    "Provide access control evidence",
			Window:             "2025-Q1",
			CollectionInterval: "quarter",
			Status:             "In Progress",
			Completeness:       0.5,
			Strategy:           "automated collection",
			Reasoning:          "GitHub API available",
			Controls: []ControlRef{
				{ReferenceID: "CC-06.1", Name: "Logical Access"},
			},
			Entries: []EvidenceEntry{
				{Filename: "perms.json", Source: "github", ControlsSatisfied: []string{"CC-06.1"}, Status: "complete"},
			},
			Gaps: []EvidenceGap{
				{Description: "Need branch protection data", Priority: "high", Remediation: "Collect branch protection rules"},
			},
		}

		md, err := cpm.generateMarkdownPlan(plan)
		require.NoError(t, err)
		assert.Contains(t, md, "ET-0001")
		assert.Contains(t, md, "GitHub Access")
		assert.Contains(t, md, "automated collection")
		assert.Contains(t, md, "CC-06.1")
		assert.Contains(t, md, "perms.json")
		assert.Contains(t, md, "branch protection data")
	})

	t.Run("empty plan", func(t *testing.T) {
		t.Parallel()
		plan := &CollectionPlan{
			TaskRef:            "ET-0002",
			TaskName:           "Empty Task",
			TaskDescription:    "No evidence yet",
			Window:             "2025",
			CollectionInterval: "year",
			Status:             "Not Started",
			Controls:           []ControlRef{},
			Entries:            []EvidenceEntry{},
			Gaps:               []EvidenceGap{},
		}

		md, err := cpm.generateMarkdownPlan(plan)
		require.NoError(t, err)
		assert.Contains(t, md, "No evidence collected yet")
		assert.Contains(t, md, "ET-0002")
	})
}
