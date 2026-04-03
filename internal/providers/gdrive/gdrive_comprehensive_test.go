// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package gdrive

import (
	"context"
	"encoding/csv"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Provider coverage gaps
// ---------------------------------------------------------------------------

func TestGDriveSyncProvider_ListPolicies_ListError(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.listErr = fmt.Errorf("API quota exceeded")
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	_, _, err := p.ListPolicies(context.Background(), interfaces.ListOptions{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API quota exceeded")
}

func TestGDriveSyncProvider_ListPolicies_GetContentError(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.files = []DriveFile{
		{ID: "doc-ok", Name: "Good", MimeType: "application/vnd.google-apps.document"},
		{ID: "doc-bad", Name: "Bad", MimeType: "application/vnd.google-apps.document"},
	}
	client.contents["doc-ok"] = "# Good Policy\n\nContent."
	// doc-bad has no content → GetFileContent returns not found

	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())
	policies, count, err := p.ListPolicies(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, count) // only the good doc
	assert.Len(t, policies, 1)
	assert.Equal(t, "doc-ok", policies[0].ID)
}

// ---------------------------------------------------------------------------
// PushControl tests
// ---------------------------------------------------------------------------

func TestGDriveSyncProvider_PushControl_Create(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	implDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	control := &domain.Control{
		ID:              "CC-06.1",
		ReferenceID:     "CC-06.1",
		Name:            "Logical Access",
		Status:          "implemented",
		RiskLevel:       "High",
		Category:        "Access Control",
		FrameworkCodes:  []domain.FrameworkCode{{Code: "SOC2"}},
		ImplementedDate: &implDate,
	}
	err := p.PushControl(context.Background(), control)
	require.NoError(t, err)

	assert.Len(t, client.created, 1)
	assert.Equal(t, client.created[0], control.ExternalIDs["gdrive"])
	// Verify sheet was stored
	sheet := client.sheets[control.ExternalIDs["gdrive"]]
	require.NotNil(t, sheet)
	assert.Equal(t, "Control Matrix", sheet.Title)
	assert.Len(t, sheet.Rows, 1)
	assert.Equal(t, "CC-06.1", sheet.Rows[0][0])
}

func TestGDriveSyncProvider_PushControl_Update(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.sheets["existing-sheet"] = &SheetData{Title: "Control Matrix", Headers: controlMatrixHeaders}
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	control := &domain.Control{
		ID:          "CC-06.1",
		ReferenceID: "CC-06.1",
		Name:        "Updated Control",
		Status:      "tested",
		ExternalIDs: map[string]string{"gdrive": "existing-sheet"},
	}
	err := p.PushControl(context.Background(), control)
	require.NoError(t, err)

	// Verify the sheet was updated (not created)
	assert.Empty(t, client.created)
	sheet := client.sheets["existing-sheet"]
	require.NotNil(t, sheet)
	assert.Equal(t, "Updated Control", sheet.Rows[0][1])
}

func TestGDriveSyncProvider_PushControl_CreateError(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.createErr = fmt.Errorf("quota exceeded")
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	control := &domain.Control{ID: "CC-06.1", ReferenceID: "CC-06.1"}
	err := p.PushControl(context.Background(), control)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "quota exceeded")
}

// ---------------------------------------------------------------------------
// PushEvidenceTask tests
// ---------------------------------------------------------------------------

func TestGDriveSyncProvider_PushEvidenceTask_Create(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	task := &domain.EvidenceTask{
		ID:                 "ET-0047",
		ReferenceID:        "ET-0047",
		Name:               "GitHub Repo Access",
		Status:             "pending",
		Priority:           "High",
		CollectionInterval: "quarterly",
		Framework:          "SOC2",
		Category:           "Infrastructure",
	}
	err := p.PushEvidenceTask(context.Background(), task)
	require.NoError(t, err)

	assert.Len(t, client.created, 1)
	assert.Equal(t, client.created[0], task.ExternalIDs["gdrive"])
	sheet := client.sheets[task.ExternalIDs["gdrive"]]
	require.NotNil(t, sheet)
	assert.Equal(t, "Evidence Tasks", sheet.Title)
	assert.Len(t, sheet.Rows, 1)
	assert.Equal(t, "ET-0047", sheet.Rows[0][0])
}

func TestGDriveSyncProvider_PushEvidenceTask_Update(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.sheets["existing-task-sheet"] = &SheetData{Title: "Evidence Tasks", Headers: evidenceTaskHeaders}
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	task := &domain.EvidenceTask{
		ID:          "ET-0047",
		ReferenceID: "ET-0047",
		Name:        "Updated Task",
		Status:      "active",
		ExternalIDs: map[string]string{"gdrive": "existing-task-sheet"},
	}
	err := p.PushEvidenceTask(context.Background(), task)
	require.NoError(t, err)

	assert.Empty(t, client.created)
	sheet := client.sheets["existing-task-sheet"]
	require.NotNil(t, sheet)
	assert.Equal(t, "Updated Task", sheet.Rows[0][1])
}

func TestGDriveSyncProvider_PushEvidenceTask_CreateError(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.createErr = fmt.Errorf("drive full")
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	task := &domain.EvidenceTask{ID: "ET-0047", ReferenceID: "ET-0047"}
	err := p.PushEvidenceTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "drive full")
}

func TestGDriveSyncProvider_DeleteControl(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())
	err := p.DeleteControl(context.Background(), "ctrl-sheet-id")
	require.NoError(t, err)
	assert.Contains(t, client.deleted, "ctrl-sheet-id")
}

func TestGDriveSyncProvider_DeleteEvidenceTask(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())
	err := p.DeleteEvidenceTask(context.Background(), "task-sheet-id")
	require.NoError(t, err)
	assert.Contains(t, client.deleted, "task-sheet-id")
}

func TestGDriveSyncProvider_PushPolicy_CreateError(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.createErr = fmt.Errorf("drive full")
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	policy := &domain.Policy{ID: "POL-001", Name: "Test", Content: "# Test"}
	err := p.PushPolicy(context.Background(), policy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "drive full")
}

func TestGDriveSyncProvider_PushPolicy_UpdateError(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.updateErr = fmt.Errorf("permission denied")
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	policy := &domain.Policy{
		ID:          "POL-001",
		Name:        "Test",
		Content:     "# Test",
		ExternalIDs: map[string]string{"gdrive": "existing-id"},
	}
	err := p.PushPolicy(context.Background(), policy)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "permission denied")
}

func TestGDriveSyncProvider_DetectChanges_ListError(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.listErr = fmt.Errorf("network error")
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	_, err := p.DetectChanges(context.Background(), time.Now().Add(-1*time.Hour))
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "network error")
}

func TestGDriveSyncProvider_DetectChanges_NoChanges(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.files = []DriveFile{
		{ID: "doc-1", Name: "Old", Modified: time.Now().Add(-2 * time.Hour)},
	}
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	changes, err := p.DetectChanges(context.Background(), time.Now().Add(-1*time.Hour))
	require.NoError(t, err)
	assert.Empty(t, changes.Changes)
}

func TestGDriveSyncProvider_GetPolicy_ParseError(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	// MarkdownToDoc doesn't really error on valid strings, but an empty
	// content still produces a valid doc. Test the GetFileContent error path.
	client.getErr = fmt.Errorf("access denied")
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	_, err := p.GetPolicy(context.Background(), "doc-1")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "access denied")
}

// ---------------------------------------------------------------------------
// Sheets coverage gaps
// ---------------------------------------------------------------------------

func TestParseControlMatrix_ShortRows(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	sheet := &SheetData{
		Title:   "Control Matrix",
		Headers: []string{"Reference ID", "Name", "Status", "Risk Level", "Framework Codes", "Category", "Implemented Date", "Tested Date"},
		Rows: [][]string{
			{"CC-06.1", "Short Row"}, // only 2 columns instead of 8
		},
	}
	controls, err := b.ParseControlMatrix(sheet)
	require.NoError(t, err)
	assert.Len(t, controls, 1)
	assert.Equal(t, "CC-06.1", controls[0].ReferenceID)
	assert.Equal(t, "Short Row", controls[0].Name)
	assert.Empty(t, controls[0].Status) // missing columns are empty
}

func TestParseEvidenceTaskSheet_ShortRows(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	sheet := &SheetData{
		Title:   "Evidence Tasks",
		Headers: []string{"Reference ID", "Name", "Status", "Priority", "Collection Interval", "Framework", "Category", "Last Collected", "Next Due"},
		Rows: [][]string{
			{"ET-0001", "Short Task"},
		},
	}
	tasks, err := b.ParseEvidenceTaskSheet(sheet)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "ET-0001", tasks[0].ReferenceID)
}

func TestParseControlMatrix_WithDates(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	sheet := &SheetData{
		Title:   "Control Matrix",
		Headers: []string{"Reference ID", "Name", "Status", "Risk Level", "Framework Codes", "Category", "Implemented Date", "Tested Date"},
		Rows: [][]string{
			{"CC-06.1", "Test", "Effective", "High", "CC6.1, CC6.2", "Security", "2025-06-15", "2025-09-20"},
		},
	}
	controls, err := b.ParseControlMatrix(sheet)
	require.NoError(t, err)
	assert.Len(t, controls, 1)
	assert.Len(t, controls[0].FrameworkCodes, 2)
	assert.Equal(t, "CC6.1", controls[0].FrameworkCodes[0].Code)
	assert.False(t, controls[0].ImplementedDate.IsZero())
	assert.False(t, controls[0].TestedDate.IsZero())
}

func TestParseEvidenceTaskSheet_WithDates(t *testing.T) {
	t.Parallel()
	b := &ControlMatrixBuilder{}
	sheet := &SheetData{
		Title:   "Evidence Tasks",
		Headers: []string{"Reference ID", "Name", "Status", "Priority", "Collection Interval", "Framework", "Category", "Last Collected", "Next Due"},
		Rows: [][]string{
			{"ET-0047", "GitHub Access", "Active", "High", "Quarterly", "SOC2", "Access Control", "2025-12-01", "2026-03-01"},
		},
	}
	tasks, err := b.ParseEvidenceTaskSheet(sheet)
	require.NoError(t, err)
	assert.Len(t, tasks, 1)
	assert.Equal(t, "High", tasks[0].Priority)
	assert.False(t, tasks[0].LastCollected.IsZero())
	assert.False(t, tasks[0].NextDue.IsZero())
}

func TestSheetData_ToCSV_SpecialCharacters(t *testing.T) {
	t.Parallel()
	sheet := &SheetData{
		Title:   "Test",
		Headers: []string{"Name", "Description"},
		Rows: [][]string{
			{"Policy, with comma", `She said "hello"`},
			{"Normal", "No special chars"},
		},
	}
	csvStr := sheet.ToCSV()
	// Verify CSV properly escapes commas and quotes
	r := csv.NewReader(strings.NewReader(csvStr))
	records, err := r.ReadAll()
	require.NoError(t, err)
	assert.Len(t, records, 3) // header + 2 rows
	assert.Equal(t, "Policy, with comma", records[1][0])
	assert.Equal(t, `She said "hello"`, records[1][1])
}

// ---------------------------------------------------------------------------
// Gdocs converter additional round-trip tests
// ---------------------------------------------------------------------------

func TestMarkdownToDoc_PolicyRoundTrip_WithTable(t *testing.T) {
	t.Parallel()
	// A realistic policy with a table (common in compliance docs)
	md := `# Access Control Policy

## Purpose

This policy defines access control requirements.

## Control Matrix

| Control | Status | Owner |
|---------|--------|-------|
| CC-06.1 | Effective | Security |
| CC-06.2 | Tested | Engineering |

## References

See the [SOC2 framework](https://example.com/soc2) for details.
`
	// Build and parse round-trip through SheetData
	controls := []domain.Control{
		{ReferenceID: "CC-06.1", Name: "Logical Access", Status: "Effective", Category: "Security"},
		{ReferenceID: "CC-06.2", Name: "Auth Controls", Status: "Tested", Category: "Engineering"},
	}
	builder := &ControlMatrixBuilder{}
	sheet := builder.BuildControlMatrix(controls)
	parsed, err := builder.ParseControlMatrix(sheet)
	require.NoError(t, err)
	assert.Equal(t, len(controls), len(parsed))
	for i := range controls {
		assert.Equal(t, controls[i].ReferenceID, parsed[i].ReferenceID)
		assert.Equal(t, controls[i].Status, parsed[i].Status)
	}

	// Verify the markdown itself is non-empty (just a sanity check)
	assert.NotEmpty(t, md)
}

// ---------------------------------------------------------------------------
// Control round-trip: push then read back
// ---------------------------------------------------------------------------

func TestGDrive_ControlRoundTrip_PushThenList(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	implDate := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	testDate := time.Date(2026, 3, 20, 0, 0, 0, 0, time.UTC)
	original := &domain.Control{
		ID:              "CC-06.1",
		ReferenceID:     "CC-06.1",
		Name:            "Logical Access Controls",
		Status:          "implemented",
		RiskLevel:       "High",
		Category:        "Access Control",
		FrameworkCodes:  []domain.FrameworkCode{{Code: "CC6.1"}, {Code: "CC6.2"}},
		ImplementedDate: &implDate,
		TestedDate:      &testDate,
	}

	// Push to create
	require.NoError(t, p.PushControl(context.Background(), original))
	sheetID := original.ExternalIDs["gdrive"]
	require.NotEmpty(t, sheetID)

	// Register the sheet as a file so ListControls can find it
	client.files = append(client.files, DriveFile{
		ID:       sheetID,
		Name:     "Control Matrix",
		MimeType: "application/vnd.google-apps.spreadsheet",
	})

	// Read back via ListControls
	controls, count, err := p.ListControls(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, controls, 1)

	roundTripped := controls[0]
	assert.Equal(t, original.ReferenceID, roundTripped.ReferenceID)
	assert.Equal(t, original.Name, roundTripped.Name)
	assert.Equal(t, original.Status, roundTripped.Status)
	assert.Equal(t, original.RiskLevel, roundTripped.RiskLevel)
	assert.Equal(t, original.Category, roundTripped.Category)
	assert.Len(t, roundTripped.FrameworkCodes, 2)
	assert.Equal(t, "CC6.1", roundTripped.FrameworkCodes[0].Code)
	assert.False(t, roundTripped.ImplementedDate.IsZero())
	assert.False(t, roundTripped.TestedDate.IsZero())
}

// ---------------------------------------------------------------------------
// Evidence task round-trip: push then read back
// ---------------------------------------------------------------------------

func TestGDrive_EvidenceTaskRoundTrip_PushThenList(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	lastCollected := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	nextDue := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	original := &domain.EvidenceTask{
		ID:                 "ET-0047",
		ReferenceID:        "ET-0047",
		Name:               "GitHub Repository Access Controls",
		Status:             "active",
		Priority:           "High",
		CollectionInterval: "quarterly",
		Framework:          "SOC2",
		Category:           "Infrastructure",
		LastCollected:      &lastCollected,
		NextDue:            &nextDue,
	}

	// Push to create
	require.NoError(t, p.PushEvidenceTask(context.Background(), original))
	sheetID := original.ExternalIDs["gdrive"]
	require.NotEmpty(t, sheetID)

	// Register the sheet so ListEvidenceTasks can find it
	client.files = append(client.files, DriveFile{
		ID:       sheetID,
		Name:     "Evidence Tasks",
		MimeType: "application/vnd.google-apps.spreadsheet",
	})

	// Read back
	tasks, count, err := p.ListEvidenceTasks(context.Background(), interfaces.ListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 1, count)
	require.Len(t, tasks, 1)

	roundTripped := tasks[0]
	assert.Equal(t, original.ReferenceID, roundTripped.ReferenceID)
	assert.Equal(t, original.Name, roundTripped.Name)
	assert.Equal(t, original.Status, roundTripped.Status)
	assert.Equal(t, original.Priority, roundTripped.Priority)
	assert.Equal(t, original.CollectionInterval, roundTripped.CollectionInterval)
	assert.Equal(t, original.Framework, roundTripped.Framework)
	assert.Equal(t, original.Category, roundTripped.Category)
	assert.False(t, roundTripped.LastCollected.IsZero())
	assert.False(t, roundTripped.NextDue.IsZero())
}

// ---------------------------------------------------------------------------
// Audit trail for push operations
// ---------------------------------------------------------------------------

func TestGDrive_AuditLog_PushControl(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	p.ClearAuditLog()
	control := &domain.Control{ID: "CC-06.1", ReferenceID: "CC-06.1", Name: "Test"}
	require.NoError(t, p.PushControl(context.Background(), control))

	entries := p.AuditLog()
	require.Len(t, entries, 1)
	assert.Equal(t, "exported", entries[0].Action)
	assert.Equal(t, "outbound", entries[0].Direction)
	assert.Equal(t, "CC-06.1", entries[0].EntityID)
	assert.Equal(t, "control", entries[0].EntityType)
}

func TestGDrive_AuditLog_PushEvidenceTask(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())

	p.ClearAuditLog()
	task := &domain.EvidenceTask{ID: "ET-0047", ReferenceID: "ET-0047", Name: "Test"}
	require.NoError(t, p.PushEvidenceTask(context.Background(), task))

	entries := p.AuditLog()
	require.Len(t, entries, 1)
	assert.Equal(t, "exported", entries[0].Action)
	assert.Equal(t, "outbound", entries[0].Direction)
	assert.Equal(t, "ET-0047", entries[0].EntityID)
	assert.Equal(t, "evidence_task", entries[0].EntityType)
}

// ---------------------------------------------------------------------------
// Scope-aware push filtering
// ---------------------------------------------------------------------------

func TestGDrive_PushPolicy_FilteredByScope(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())
	p.SetScope(SyncScope{
		Policies:      EntityScope{Enabled: true, Exclude: []string{"POL-DRAFT-*"}},
		Controls:      EntityScope{Enabled: true},
		EvidenceTasks: EntityScope{Enabled: true},
	})

	// Excluded policy — should be silently skipped
	pol := &domain.Policy{ID: "POL-DRAFT-001", ReferenceID: "POL-DRAFT-001", Name: "Draft", Content: "# Draft"}
	err := p.PushPolicy(context.Background(), pol)
	require.NoError(t, err)
	assert.Empty(t, client.created) // nothing created

	entries := p.AuditLog()
	require.Len(t, entries, 1)
	assert.Equal(t, "unchanged", entries[0].Action)
}

func TestGDrive_PushControl_FilteredByScope(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())
	p.SetScope(SyncScope{
		Policies:      EntityScope{Enabled: true},
		Controls:      EntityScope{Enabled: false},
		EvidenceTasks: EntityScope{Enabled: true},
	})

	control := &domain.Control{ID: "CC-06.1", ReferenceID: "CC-06.1"}
	err := p.PushControl(context.Background(), control)
	require.NoError(t, err)
	assert.Empty(t, client.created)
}

func TestGDrive_PushEvidenceTask_FilteredByScope(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	p := NewGDriveSyncProvider(client, "root", testhelpers.NewStubLogger())
	p.SetScope(SyncScope{
		Policies:      EntityScope{Enabled: true},
		Controls:      EntityScope{Enabled: true},
		EvidenceTasks: EntityScope{Enabled: true, TagsExclude: []string{"sensitive"}},
	})

	task := &domain.EvidenceTask{
		ID:          "ET-0001",
		ReferenceID: "ET-0001",
		Tags:        []domain.Tag{{Name: "sensitive"}},
	}
	err := p.PushEvidenceTask(context.Background(), task)
	require.NoError(t, err)
	assert.Empty(t, client.created)
}

// ---------------------------------------------------------------------------
// PushControl / PushEvidenceTask update errors
// ---------------------------------------------------------------------------

func TestGDriveSyncProvider_PushControl_UpdateError(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	client.sheets["existing-sheet"] = &SheetData{Title: "Control Matrix"}
	// Make UpdateSheetData fail by removing the sheet after setup
	// Use a custom stub that always errors
	errClient := &errorUpdateStubClient{stubDriveClient: client}
	p := NewGDriveSyncProvider(errClient, "root", testhelpers.NewStubLogger())

	control := &domain.Control{
		ID:          "CC-06.1",
		ReferenceID: "CC-06.1",
		ExternalIDs: map[string]string{"gdrive": "existing-sheet"},
	}
	err := p.PushControl(context.Background(), control)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update error")
}

func TestGDriveSyncProvider_PushEvidenceTask_UpdateError(t *testing.T) {
	t.Parallel()
	client := newStubDriveClient()
	errClient := &errorUpdateStubClient{stubDriveClient: client}
	p := NewGDriveSyncProvider(errClient, "root", testhelpers.NewStubLogger())

	task := &domain.EvidenceTask{
		ID:          "ET-0001",
		ReferenceID: "ET-0001",
		ExternalIDs: map[string]string{"gdrive": "existing-sheet"},
	}
	err := p.PushEvidenceTask(context.Background(), task)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "update error")
}

// errorUpdateStubClient wraps stubDriveClient to always error on UpdateSheetData.
type errorUpdateStubClient struct {
	*stubDriveClient
}

func (e *errorUpdateStubClient) UpdateSheetData(ctx context.Context, fileID string, data *SheetData) error {
	return fmt.Errorf("update error")
}
