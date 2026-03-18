package registry_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/registry"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestRegistry(t *testing.T) *registry.EvidenceTaskRegistry {
	t.Helper()
	return registry.NewEvidenceTaskRegistry(t.TempDir())
}

// ---------------------------------------------------------------------------
// Basic creation and registration
// ---------------------------------------------------------------------------

func TestNewEvidenceTaskRegistry(t *testing.T) {
	t.Parallel()
	r := newTestRegistry(t)
	assert.NotNil(t, r)
	assert.Empty(t, r.GetAllEntries())
}

func TestRegisterTask(t *testing.T) {
	t.Parallel()

	r := newTestRegistry(t)
	task := &domain.EvidenceTask{ID: 100, Name: "Test Task", Framework: "SOC2", Status: "pending"}

	ref := r.RegisterTask(task)
	assert.Equal(t, "ET-0001", ref)

	// Register another
	task2 := &domain.EvidenceTask{ID: 200, Name: "Second Task", Framework: "ISO27001", Status: "active"}
	ref2 := r.RegisterTask(task2)
	assert.Equal(t, "ET-0002", ref2)
}

func TestRegisterTask_ExistingUpdatesInfo(t *testing.T) {
	t.Parallel()

	r := newTestRegistry(t)
	task := &domain.EvidenceTask{ID: 100, Name: "Original", Framework: "SOC2", Status: "pending"}
	ref := r.RegisterTask(task)

	// Re-register same ID with updated info
	task.Name = "Updated"
	task.Status = "complete"
	ref2 := r.RegisterTask(task)

	assert.Equal(t, ref, ref2)
	entry, ok := r.GetEntry(100)
	require.True(t, ok)
	assert.Equal(t, "Updated", entry.Name)
	assert.Equal(t, "complete", entry.Status)
}

// ---------------------------------------------------------------------------
// Lookup operations
// ---------------------------------------------------------------------------

func TestGetReference(t *testing.T) {
	t.Parallel()

	r := newTestRegistry(t)
	r.RegisterTask(&domain.EvidenceTask{ID: 42, Name: "Task 42"})

	ref, ok := r.GetReference(42)
	assert.True(t, ok)
	assert.Equal(t, "ET-0001", ref)

	_, ok = r.GetReference(999)
	assert.False(t, ok)
}

func TestGetTaskID(t *testing.T) {
	t.Parallel()

	r := newTestRegistry(t)
	r.RegisterTask(&domain.EvidenceTask{ID: 42})

	id, ok := r.GetTaskID("ET-0001")
	assert.True(t, ok)
	assert.Equal(t, 42, id)

	_, ok = r.GetTaskID("ET-9999")
	assert.False(t, ok)
}

func TestGetEntry(t *testing.T) {
	t.Parallel()

	r := newTestRegistry(t)
	r.RegisterTask(&domain.EvidenceTask{ID: 42, Name: "My Task", Framework: "SOC2", Status: "pending"})

	entry, ok := r.GetEntry(42)
	require.True(t, ok)
	assert.Equal(t, 42, entry.TaskID)
	assert.Equal(t, "My Task", entry.Name)
	assert.Equal(t, "SOC2", entry.Framework)

	_, ok = r.GetEntry(999)
	assert.False(t, ok)
}

// ---------------------------------------------------------------------------
// GetAllEntries ordering
// ---------------------------------------------------------------------------

func TestGetAllEntries_Sorted(t *testing.T) {
	t.Parallel()

	r := newTestRegistry(t)
	// Register in reverse order
	r.RegisterTask(&domain.EvidenceTask{ID: 300})
	r.RegisterTask(&domain.EvidenceTask{ID: 100})
	r.RegisterTask(&domain.EvidenceTask{ID: 200})

	entries := r.GetAllEntries()
	require.Len(t, entries, 3)
	assert.Equal(t, "ET-0001", entries[0].Reference)
	assert.Equal(t, "ET-0002", entries[1].Reference)
	assert.Equal(t, "ET-0003", entries[2].Reference)
}

// ---------------------------------------------------------------------------
// Update and Remove
// ---------------------------------------------------------------------------

func TestUpdateTaskInfo(t *testing.T) {
	t.Parallel()

	r := newTestRegistry(t)
	task := &domain.EvidenceTask{ID: 1, Name: "Original", Framework: "SOC2", Status: "pending"}
	r.RegisterTask(task)

	task.Name = "Updated"
	task.Status = "complete"
	r.UpdateTaskInfo(task)

	entry, ok := r.GetEntry(1)
	require.True(t, ok)
	assert.Equal(t, "Updated", entry.Name)
	assert.Equal(t, "complete", entry.Status)
}

func TestUpdateTaskInfo_NonExistent(t *testing.T) {
	t.Parallel()
	r := newTestRegistry(t)
	// Should not panic
	r.UpdateTaskInfo(&domain.EvidenceTask{ID: 999, Name: "ghost"})
	_, ok := r.GetEntry(999)
	assert.False(t, ok)
}

func TestRemoveTask(t *testing.T) {
	t.Parallel()

	r := newTestRegistry(t)
	r.RegisterTask(&domain.EvidenceTask{ID: 1})
	r.RegisterTask(&domain.EvidenceTask{ID: 2})

	removed := r.RemoveTask(1)
	assert.True(t, removed)
	assert.Len(t, r.GetAllEntries(), 1)

	_, ok := r.GetReference(1)
	assert.False(t, ok)

	// Remove again
	removed = r.RemoveTask(1)
	assert.False(t, removed)
}

// ---------------------------------------------------------------------------
// GetStats
// ---------------------------------------------------------------------------

func TestGetStats(t *testing.T) {
	t.Parallel()

	r := newTestRegistry(t)
	r.RegisterTask(&domain.EvidenceTask{ID: 50})
	r.RegisterTask(&domain.EvidenceTask{ID: 100})

	stats := r.GetStats()
	assert.Equal(t, 2, stats["total_entries"])
	assert.Equal(t, 3, stats["next_reference_num"])
	assert.Equal(t, 100, stats["highest_task_id"])
}

// ---------------------------------------------------------------------------
// Save / Load round-trip
// ---------------------------------------------------------------------------

func TestSaveAndLoadRegistry(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	r := registry.NewEvidenceTaskRegistry(dir)
	r.RegisterTask(&domain.EvidenceTask{ID: 10, Name: "Task A", Framework: "SOC2", Status: "pending"})
	r.RegisterTask(&domain.EvidenceTask{ID: 20, Name: "Task B", Framework: "ISO27001", Status: "active"})

	require.NoError(t, r.SaveRegistry())

	// Load into a fresh registry
	r2 := registry.NewEvidenceTaskRegistry(dir)
	require.NoError(t, r2.LoadRegistry())

	entries := r2.GetAllEntries()
	require.Len(t, entries, 2)
	assert.Equal(t, "Task A", entries[0].Name)
	assert.Equal(t, "Task B", entries[1].Name)

	// Verify next reference number is correct after load
	ref := r2.RegisterTask(&domain.EvidenceTask{ID: 30, Name: "Task C"})
	assert.Equal(t, "ET-0003", ref)
}

func TestLoadRegistry_FileNotExists(t *testing.T) {
	t.Parallel()
	r := newTestRegistry(t)
	assert.NoError(t, r.LoadRegistry()) // should not error
	assert.Empty(t, r.GetAllEntries())
}

func TestLoadRegistry_EmptyFile(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	csvPath := filepath.Join(dir, "docs", "evidence_task_registry.csv")
	require.NoError(t, os.MkdirAll(filepath.Dir(csvPath), 0755))
	require.NoError(t, os.WriteFile(csvPath, []byte{}, 0644))

	r := registry.NewEvidenceTaskRegistry(dir)
	assert.NoError(t, r.LoadRegistry())
}

func TestLoadRegistry_InvalidHeader(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	csvPath := filepath.Join(dir, "docs", "evidence_task_registry.csv")
	require.NoError(t, os.MkdirAll(filepath.Dir(csvPath), 0755))
	require.NoError(t, os.WriteFile(csvPath, []byte("bad,header\n"), 0644))

	r := registry.NewEvidenceTaskRegistry(dir)
	err := r.LoadRegistry()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid header")
}

func TestLoadRegistry_MalformedRows(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	csvPath := filepath.Join(dir, "docs", "evidence_task_registry.csv")
	require.NoError(t, os.MkdirAll(filepath.Dir(csvPath), 0755))
	content := "task_id,reference,name,framework,status\n" +
		"notanumber,ET-0001,Bad Row,SOC2,pending\n" +
		"10,ET-0001,Good Row,SOC2,pending\n"
	require.NoError(t, os.WriteFile(csvPath, []byte(content), 0644))

	r := registry.NewEvidenceTaskRegistry(dir)
	require.NoError(t, r.LoadRegistry())
	// Only the valid row should be loaded
	entries := r.GetAllEntries()
	assert.Len(t, entries, 1)
	assert.Equal(t, "Good Row", entries[0].Name)
}

// ---------------------------------------------------------------------------
// InitializeFromTasks
// ---------------------------------------------------------------------------

func TestInitializeFromTasks(t *testing.T) {
	t.Parallel()

	r := newTestRegistry(t)
	tasks := []domain.EvidenceTask{
		{ID: 30, Name: "C"},
		{ID: 10, Name: "A"},
		{ID: 20, Name: "B"},
	}

	r.InitializeFromTasks(tasks)

	entries := r.GetAllEntries()
	require.Len(t, entries, 3)
	// Should be assigned in ID order: 10->ET-0001, 20->ET-0002, 30->ET-0003
	assert.Equal(t, "ET-0001", entries[0].Reference)
	assert.Equal(t, 10, entries[0].TaskID)
	assert.Equal(t, "ET-0002", entries[1].Reference)
	assert.Equal(t, 20, entries[1].TaskID)
	assert.Equal(t, "ET-0003", entries[2].Reference)
	assert.Equal(t, 30, entries[2].TaskID)
}
