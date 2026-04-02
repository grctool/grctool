// Copyright 2024 GRCTool Authors
// SPDX-License-Identifier: Apache-2.0

package services

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newTestLogger(t *testing.T) logger.Logger {
	t.Helper()
	cfg := logger.DefaultConfig()
	cfg.Level = logger.ErrorLevel // quiet during tests
	log, err := logger.New(cfg)
	require.NoError(t, err)
	return log
}

func newTestStorage(t *testing.T, dir string) *storage.Storage {
	t.Helper()
	s, err := storage.NewStorage(config.StorageConfig{DataDir: dir})
	require.NoError(t, err)
	return s
}

func newTestConfig(dir string) *config.Config {
	return &config.Config{
		Storage: config.StorageConfig{DataDir: dir},
	}
}

// stubDataService is a simple DataService stub that stores data in maps.
type stubDataService struct {
	tasks    map[string]*domain.EvidenceTask
	controls map[string]*domain.Control
	policies map[string]*domain.Policy
	records  map[string][]domain.EvidenceRecord
}

func newStubDataService() *stubDataService {
	return &stubDataService{
		tasks:    make(map[string]*domain.EvidenceTask),
		controls: make(map[string]*domain.Control),
		policies: make(map[string]*domain.Policy),
		records:  make(map[string][]domain.EvidenceRecord),
	}
}

func (s *stubDataService) GetEvidenceTask(_ context.Context, taskID string) (*domain.EvidenceTask, error) {
	if t, ok := s.tasks[taskID]; ok {
		return t, nil
	}
	return nil, os.ErrNotExist
}

func (s *stubDataService) GetAllEvidenceTasks(_ context.Context) ([]domain.EvidenceTask, error) {
	out := make([]domain.EvidenceTask, 0, len(s.tasks))
	for _, t := range s.tasks {
		out = append(out, *t)
	}
	return out, nil
}

func (s *stubDataService) FilterEvidenceTasks(_ context.Context, filter domain.EvidenceFilter) ([]domain.EvidenceTask, error) {
	var out []domain.EvidenceTask
	for _, t := range s.tasks {
		if len(filter.Status) > 0 {
			match := false
			for _, st := range filter.Status {
				if t.Status == st {
					match = true
				}
			}
			if !match {
				continue
			}
		}
		out = append(out, *t)
	}
	return out, nil
}

func (s *stubDataService) GetControl(_ context.Context, controlID string) (*domain.Control, error) {
	if c, ok := s.controls[controlID]; ok {
		return c, nil
	}
	return nil, os.ErrNotExist
}

func (s *stubDataService) GetAllControls(_ context.Context) ([]domain.Control, error) {
	out := make([]domain.Control, 0, len(s.controls))
	for _, c := range s.controls {
		out = append(out, *c)
	}
	return out, nil
}

func (s *stubDataService) GetPolicy(_ context.Context, policyID string) (*domain.Policy, error) {
	if p, ok := s.policies[policyID]; ok {
		return p, nil
	}
	return nil, os.ErrNotExist
}

func (s *stubDataService) GetAllPolicies(_ context.Context) ([]domain.Policy, error) {
	out := make([]domain.Policy, 0, len(s.policies))
	for _, p := range s.policies {
		out = append(out, *p)
	}
	return out, nil
}

func (s *stubDataService) GetRelationships(_ context.Context, _, _ string) ([]domain.Relationship, error) {
	return nil, nil
}

func (s *stubDataService) SaveEvidenceRecord(_ context.Context, record *domain.EvidenceRecord) error {
	s.records[record.TaskID] = append(s.records[record.TaskID], *record)
	return nil
}

func (s *stubDataService) GetEvidenceRecords(_ context.Context, taskID string) ([]domain.EvidenceRecord, error) {
	return s.records[taskID], nil
}

// stubEvidenceScanner implements the EvidenceScanner interface for testing
type stubEvidenceScanner struct {
	scanAllResult   map[string]*models.EvidenceTaskState
	scanTaskResult  *models.EvidenceTaskState
	scanWindowState *models.WindowState
	scanErr         error
}

func (s *stubEvidenceScanner) ScanAll(_ context.Context) (map[string]*models.EvidenceTaskState, error) {
	return s.scanAllResult, s.scanErr
}

func (s *stubEvidenceScanner) ScanTask(_ context.Context, _ string) (*models.EvidenceTaskState, error) {
	return s.scanTaskResult, s.scanErr
}

func (s *stubEvidenceScanner) ScanWindow(_ context.Context, _, _ string) (*models.WindowState, error) {
	return s.scanWindowState, s.scanErr
}

// ---------------------------------------------------------------------------
// data.go – DataService additional tests
// ---------------------------------------------------------------------------

func TestDataServiceImpl_GetRelationships_EvidenceTask(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := newTestStorage(t, dir)
	svc := NewDataService(store)

	// Save a task with controls
	task := &domain.EvidenceTask{
	ID: "100",
		ReferenceID: "ET-0001",
		Name:        "GitHub Access Controls",
		Framework:   "SOC2",
		Controls:    []string{"AC-01", "AC-02"},
	}
	require.NoError(t, store.SaveEvidenceTask(task))

	// Save a matching policy
	pol := &domain.Policy{
		ID:        "5",
		Name:      "Access Policy",
		Framework: "SOC2",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	require.NoError(t, store.SavePolicy(pol))

	rels, err := svc.GetRelationships(context.Background(), "evidence_task", "100")
	require.NoError(t, err)

	// Should have 2 control relationships + 1 policy relationship
	controlRels := 0
	policyRels := 0
	for _, r := range rels {
		if r.TargetType == "control" {
			controlRels++
			assert.Equal(t, "verifies", r.Type)
		}
		if r.TargetType == "policy" {
			policyRels++
			assert.Equal(t, "implements", r.Type)
		}
	}
	assert.Equal(t, 2, controlRels)
	assert.Equal(t, 1, policyRels)
}

func TestDataServiceImpl_GetRelationships_InvalidID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := newTestStorage(t, dir)
	svc := NewDataService(store)

	_, err := svc.GetRelationships(context.Background(), "evidence_task", "not-a-number")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task ID format")
}

func TestDataServiceImpl_GetEvidenceRecords(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := newTestStorage(t, dir)
	svc := NewDataService(store)

	// Save some records
	record := &domain.EvidenceRecord{
		ID:          "rec-1",
		TaskID: "42",
		Title:       "Test Record",
		Content:     "content",
		Format:      "markdown",
		Source:      "manual",
		CollectedAt: time.Now(),
		CollectedBy: "tester",
	}
	require.NoError(t, svc.SaveEvidenceRecord(context.Background(), record))

	records, err := svc.GetEvidenceRecords(context.Background(), "42")
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(records), 1)
}

func TestMatchesEvidenceFilter(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	store := newTestStorage(t, dir)
	impl := NewDataService(store).(*DataServiceImpl)

	now := time.Now()
	past := now.Add(-48 * time.Hour)
	future := now.Add(48 * time.Hour)
	boolTrue := true
	boolFalse := false

	tests := []struct {
		name     string
		task     domain.EvidenceTask
		filter   domain.EvidenceFilter
		expected bool
	}{
		{
			name:     "empty filter matches everything",
			task:     domain.EvidenceTask{Status: "pending", Priority: "high"},
			filter:   domain.EvidenceFilter{},
			expected: true,
		},
		{
			name:     "status match",
			task:     domain.EvidenceTask{Status: "pending"},
			filter:   domain.EvidenceFilter{Status: []string{"pending", "in_progress"}},
			expected: true,
		},
		{
			name:     "status no match",
			task:     domain.EvidenceTask{Status: "completed"},
			filter:   domain.EvidenceFilter{Status: []string{"pending"}},
			expected: false,
		},
		{
			name:     "priority match",
			task:     domain.EvidenceTask{Priority: "high"},
			filter:   domain.EvidenceFilter{Priority: []string{"high"}},
			expected: true,
		},
		{
			name:     "priority no match",
			task:     domain.EvidenceTask{Priority: "low"},
			filter:   domain.EvidenceFilter{Priority: []string{"high", "medium"}},
			expected: false,
		},
		{
			name:     "framework match",
			task:     domain.EvidenceTask{Framework: "SOC2"},
			filter:   domain.EvidenceFilter{Framework: "SOC2"},
			expected: true,
		},
		{
			name:     "framework no match",
			task:     domain.EvidenceTask{Framework: "ISO27001"},
			filter:   domain.EvidenceFilter{Framework: "SOC2"},
			expected: false,
		},
		{
			name:     "due before - within range",
			task:     domain.EvidenceTask{NextDue: &past},
			filter:   domain.EvidenceFilter{DueBefore: &now},
			expected: true,
		},
		{
			name:     "due before - out of range",
			task:     domain.EvidenceTask{NextDue: &future},
			filter:   domain.EvidenceFilter{DueBefore: &now},
			expected: false,
		},
		{
			name:     "due after - within range",
			task:     domain.EvidenceTask{NextDue: &future},
			filter:   domain.EvidenceFilter{DueAfter: &now},
			expected: true,
		},
		{
			name:     "due after - out of range",
			task:     domain.EvidenceTask{NextDue: &past},
			filter:   domain.EvidenceFilter{DueAfter: &now},
			expected: false,
		},
		{
			name:     "sensitive true match",
			task:     domain.EvidenceTask{Sensitive: true},
			filter:   domain.EvidenceFilter{Sensitive: &boolTrue},
			expected: true,
		},
		{
			name:     "sensitive false no match",
			task:     domain.EvidenceTask{Sensitive: true},
			filter:   domain.EvidenceFilter{Sensitive: &boolFalse},
			expected: false,
		},
		{
			name: "aec status filter match with nil AecStatus",
			task: domain.EvidenceTask{},
			filter: domain.EvidenceFilter{
				AecStatus: []string{"na"},
			},
			expected: true,
		},
		{
			name: "aec status filter match with set status",
			task: domain.EvidenceTask{
				AecStatus: &domain.AecStatus{Status: "enabled"},
			},
			filter:   domain.EvidenceFilter{AecStatus: []string{"enabled"}},
			expected: true,
		},
		{
			name: "aec status filter no match",
			task: domain.EvidenceTask{
				AecStatus: &domain.AecStatus{Status: "disabled"},
			},
			filter:   domain.EvidenceFilter{AecStatus: []string{"enabled"}},
			expected: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := impl.matchesEvidenceFilter(tc.task, tc.filter)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// document.go – DocumentService tests
// ---------------------------------------------------------------------------

func TestDocumentService_GenerateDocument(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := newTestConfig(dir)
	ds := NewDocumentService(cfg)

	err := ds.GenerateDocument(PolicyDocument, "test_policy.md", "# Test Policy\n\nContent here.")
	require.NoError(t, err)

	path := ds.GetDocumentPath(PolicyDocument, "test_policy.md")
	assert.FileExists(t, path)

	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), "Test Policy")
}

func TestDocumentService_GenerateDocument_NonMarkdown(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := newTestConfig(dir)
	ds := NewDocumentService(cfg)

	err := ds.GenerateDocument(PolicyDocument, "data.json", `{"key": "value"}`)
	require.NoError(t, err)

	path := ds.GetDocumentPath(PolicyDocument, "data.json")
	content, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Contains(t, string(content), `"key"`)
}

func TestDocumentService_EnsureDocumentDirectory(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := newTestConfig(dir)
	ds := NewDocumentService(cfg)

	err := ds.EnsureDocumentDirectory(ControlDocument)
	require.NoError(t, err)

	docDir := ds.getDocumentPath(ControlDocument)
	info, err := os.Stat(docDir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestDocumentService_GetDocumentPath(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := newTestConfig(dir)
	ds := NewDocumentService(cfg)

	// PolicyDocument path is resolved correctly via ResolveRelativeTo
	path := ds.GetDocumentPath(PolicyDocument, "policy.md")
	assert.Contains(t, path, "policy.md")
	assert.Contains(t, path, dir)
}

func TestDocumentService_AllDocumentTypes(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := newTestConfig(dir)
	ds := NewDocumentService(cfg)

	types := []DocumentType{PolicyDocument, ControlDocument, EvidenceTaskDocument, EvidencePromptDocument}
	for _, dt := range types {
		t.Run(string(dt), func(t *testing.T) {
			err := ds.GenerateDocument(dt, "test.md", "# Test\n")
			require.NoError(t, err)
			assert.FileExists(t, ds.GetDocumentPath(dt, "test.md"))
		})
	}
}

func TestDocumentService_UnknownDocumentType(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	cfg := newTestConfig(dir)
	ds := NewDocumentService(cfg)

	// Unknown type should fall back to baseDir + string(type)
	err := ds.GenerateDocument(DocumentType("custom_type"), "test.md", "# Custom")
	require.NoError(t, err)

	expected := filepath.Join(dir, "custom_type", "test.md")
	assert.FileExists(t, expected)
}

// ---------------------------------------------------------------------------
// doc_generator.go – GenerateAgentDocs tests
// ---------------------------------------------------------------------------

func TestGenerateAgentDocs(t *testing.T) {
	// NOT parallel: uses os.Chdir which is process-global.

	// Save current dir and change to temp dir to control .grctool/ location
	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(origDir) })

	cfg := &config.Config{
		Storage: config.StorageConfig{DataDir: "./data"},
	}

	err = GenerateAgentDocs(cfg, "1.0.0-test")
	require.NoError(t, err)

	// Check that expected docs were generated
	expectedFiles := []string{
		"directory-structure.md",
		"evidence-workflow.md",
		"tool-capabilities.md",
		"status-commands.md",
		"submission-process.md",
		"bulk-operations.md",
	}

	docsDir := filepath.Join(tmpDir, ".grctool", "docs")
	for _, fname := range expectedFiles {
		fpath := filepath.Join(docsDir, fname)
		assert.FileExists(t, fpath, "expected %s to exist", fname)

		content, err := os.ReadFile(fpath)
		require.NoError(t, err)
		assert.NotEmpty(t, content)
		// Check templates were rendered (should not contain {{.Timestamp}})
		assert.NotContains(t, string(content), "{{.Timestamp}}")
	}
}

// ---------------------------------------------------------------------------
// sync.go – SyncService helper method tests
// ---------------------------------------------------------------------------

func TestSyncService_GetWindowFromDate(t *testing.T) {
	t.Parallel()

	// We need to construct a minimal SyncService to call the method.
	// Since getWindowFromDate only uses time parsing, we just need the struct.
	s := &SyncService{}

	tests := []struct {
		name     string
		date     string
		expected string
	}{
		{"Q1 Jan", "2025-01-15", "2025-Q1"},
		{"Q1 Mar", "2025-03-31", "2025-Q1"},
		{"Q2 Apr", "2025-04-01", "2025-Q2"},
		{"Q2 Jun", "2025-06-30", "2025-Q2"},
		{"Q3 Jul", "2025-07-01", "2025-Q3"},
		{"Q3 Sep", "2025-09-15", "2025-Q3"},
		{"Q4 Oct", "2025-10-01", "2025-Q4"},
		{"Q4 Dec", "2025-12-31", "2025-Q4"},
		{"invalid date defaults to current quarter", "not-a-date", ""}, // will produce current quarter
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			result := s.getWindowFromDate(tc.date)
			if tc.expected == "" {
				// Invalid date - just verify format is YYYY-QN
				assert.Regexp(t, `^\d{4}-Q[1-4]$`, result)
			} else {
				assert.Equal(t, tc.expected, result)
			}
		})
	}
}

func TestSyncService_ParseTime(t *testing.T) {
	t.Parallel()
	s := &SyncService{}

	t.Run("valid RFC3339", func(t *testing.T) {
		t.Parallel()
		result := s.parseTime("2025-06-15T10:30:00Z")
		assert.Equal(t, 2025, result.Year())
		assert.Equal(t, time.June, result.Month())
		assert.Equal(t, 15, result.Day())
	})

	t.Run("invalid time returns now", func(t *testing.T) {
		t.Parallel()
		before := time.Now().Add(-time.Second)
		result := s.parseTime("invalid-time")
		after := time.Now().Add(time.Second)
		assert.True(t, result.After(before))
		assert.True(t, result.Before(after))
	})
}

func TestSyncService_GetDisplayName(t *testing.T) {
	t.Parallel()
	s := &SyncService{}

	// Import tugboatModels type inline since we reference it
	// We test via the method directly on the SyncService struct

	t.Run("nil member", func(t *testing.T) {
		result := s.getDisplayName(nil)
		assert.Equal(t, "Unknown", result)
	})
}

func TestSyncOptions_Fields(t *testing.T) {
	t.Parallel()

	opts := SyncOptions{
		OrgID:       "test-org",
		Framework:   "SOC2",
		Policies:    true,
		Controls:    true,
		Evidence:    false,
		Submissions: false,
	}

	assert.Equal(t, "test-org", opts.OrgID)
	assert.Equal(t, "SOC2", opts.Framework)
	assert.True(t, opts.Policies)
	assert.True(t, opts.Controls)
	assert.False(t, opts.Evidence)
	assert.False(t, opts.Submissions)
}

func TestSyncResult_Fields(t *testing.T) {
	t.Parallel()

	now := time.Now()
	result := SyncResult{
		StartTime: now,
		EndTime:   now.Add(5 * time.Second),
		Duration:  5 * time.Second,
		Errors:    []string{"error1", "error2"},
		Policies:  SyncStats{Total: 10, Synced: 8, Errors: 2},
		Controls:  SyncStats{Total: 20, Synced: 20, Errors: 0},
	}

	assert.Equal(t, 10, result.Policies.Total)
	assert.Equal(t, 8, result.Policies.Synced)
	assert.Equal(t, 2, result.Policies.Errors)
	assert.Equal(t, 20, result.Controls.Synced)
	assert.Len(t, result.Errors, 2)
	assert.Equal(t, 5*time.Second, result.Duration)
}

func TestSyncStats_Fields(t *testing.T) {
	t.Parallel()

	stats := SyncStats{
		Total:      100,
		Synced:     90,
		Detailed:   85,
		Errors:     5,
		Skipped:    5,
		Downloaded: 42,
	}

	assert.Equal(t, 100, stats.Total)
	assert.Equal(t, 90, stats.Synced)
	assert.Equal(t, 85, stats.Detailed)
	assert.Equal(t, 5, stats.Errors)
	assert.Equal(t, 5, stats.Skipped)
	assert.Equal(t, 42, stats.Downloaded)
}

// ---------------------------------------------------------------------------
// evidence_evaluator.go – EvidenceEvaluatorService tests
// ---------------------------------------------------------------------------

func newTestEvaluator(t *testing.T) (*EvidenceEvaluatorService, *storage.Storage) {
	t.Helper()
	dir := t.TempDir()
	store := newTestStorage(t, dir)
	log := newTestLogger(t)
	scanner := &stubEvidenceScanner{}
	return NewEvidenceEvaluatorService(filepath.Join(dir, "evidence"), store, scanner, log), store
}

func TestEvidenceEvaluator_EvaluateCompleteness_NoFiles(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	task := &domain.EvidenceTask{
	ID: "1",
		Name: "Test Task",
	}
	window := &models.WindowState{FileCount: 0}
	result := models.NewEvaluationResult("ET-0001", "1", "2025-Q1", "all")

	evaluator.evaluateCompleteness(task, window, result)

	assert.Equal(t, float64(0), result.Completeness.Score)
	assert.Equal(t, "fail", result.Completeness.Status)
	assert.Contains(t, result.Completeness.Details, "No evidence files")
	assert.GreaterOrEqual(t, len(result.Issues), 1)
	assert.Equal(t, models.IssueCritical, result.Issues[0].Severity)
}

func TestEvidenceEvaluator_EvaluateCompleteness_WithFiles(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	task := &domain.EvidenceTask{
	ID: "1",
		Name:        "Test Task",
		Description: "A short description",
	}
	now := time.Now()
	window := &models.WindowState{
		FileCount:  3,
		TotalBytes: 5000,
		NewestFile: &now,
		Files: []models.FileState{
			{Filename: "access_report.csv", SizeBytes: 2000},
			{Filename: "permissions.json", SizeBytes: 1500},
			{Filename: "summary.md", SizeBytes: 1500},
		},
	}
	result := models.NewEvaluationResult("ET-0001", "1", "2025-Q1", "all")

	evaluator.evaluateCompleteness(task, window, result)

	assert.Greater(t, result.Completeness.Score, float64(40))
	assert.Equal(t, float64(100), result.Completeness.MaxScore)
}

func TestEvidenceEvaluator_EvaluateCompleteness_WithGenerationMeta(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	task := &domain.EvidenceTask{
	ID: "1",
		Name: "Test Task",
	}
	now := time.Now()
	window := &models.WindowState{
		FileCount:         2,
		TotalBytes:        3000,
		HasGenerationMeta: true,
		NewestFile:        &now,
		Files: []models.FileState{
			{Filename: "evidence.csv", SizeBytes: 2000},
			{Filename: "readme.md", SizeBytes: 1000},
		},
	}
	result := models.NewEvaluationResult("ET-0001", "1", "2025-Q1", "all")

	evaluator.evaluateCompleteness(task, window, result)

	// With generation meta, score should be higher
	assert.Greater(t, result.Completeness.Score, float64(55))
	assert.Contains(t, result.Completeness.Details, "Generation metadata present")
}

func TestEvidenceEvaluator_EvaluateQuality_NoFiles(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	task := &domain.EvidenceTask{ID: "1", Name: "Test"}
	window := &models.WindowState{FileCount: 0}
	result := models.NewEvaluationResult("ET-0001", "1", "2025-Q1", "all")

	evaluator.evaluateQuality(task, window, result)

	assert.Equal(t, float64(0), result.QualityScore.Score)
	assert.Equal(t, "fail", result.QualityScore.Status)
}

func TestEvidenceEvaluator_EvaluateQuality_HighQuality(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	task := &domain.EvidenceTask{ID: "1", Name: "Test"}
	window := &models.WindowState{
		FileCount: 3,
		Files: []models.FileState{
			{Filename: "access_controls_report.csv", SizeBytes: 5000},
			{Filename: "security_configuration.json", SizeBytes: 3000},
			{Filename: "evidence_summary.md", SizeBytes: 2000},
		},
	}
	result := models.NewEvaluationResult("ET-0001", "1", "2025-Q1", "all")

	evaluator.evaluateQuality(task, window, result)

	// All files follow naming conventions, have reasonable sizes,
	// include structured and documentation formats
	assert.Greater(t, result.QualityScore.Score, float64(70))
	assert.Contains(t, result.QualityScore.Status, "pass")
}

func TestEvidenceEvaluator_EvaluateQuality_PoorNaming(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	task := &domain.EvidenceTask{ID: "1", Name: "Test"}
	window := &models.WindowState{
		FileCount: 2,
		Files: []models.FileState{
			{Filename: "A.txt", SizeBytes: 500},
			{Filename: "B C.pdf", SizeBytes: 500},
		},
	}
	result := models.NewEvaluationResult("ET-0001", "1", "2025-Q1", "all")

	evaluator.evaluateQuality(task, window, result)

	// Poor naming should reduce score
	assert.Less(t, result.QualityScore.Score, float64(80))
}

func TestEvidenceEvaluator_EvaluateRequirementsMatch(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	task := &domain.EvidenceTask{
	ID: "1",
		Name:        "GitHub Repository Access Controls",
		Description: "Show team permissions and repository access controls for GitHub",
		Guidance:    "Collect access control configuration from GitHub",
	}
	window := &models.WindowState{
		FileCount: 2,
		Files: []models.FileState{
			{Filename: "github_permissions.csv", SizeBytes: 2000},
			{Filename: "access_control_matrix.json", SizeBytes: 3000},
		},
	}
	result := models.NewEvaluationResult("ET-0001", "1", "2025-Q1", "all")

	evaluator.evaluateRequirementsMatch(task, window, result)

	assert.Greater(t, result.RequirementsMatch.Score, float64(40))
	assert.NotEmpty(t, result.RequirementsMatch.Details)
}

func TestEvidenceEvaluator_EvaluateRequirementsMatch_NoKeywords(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	task := &domain.EvidenceTask{
	ID: "1",
		Name:        "Simple task",
		Description: "Very basic task with no special keywords",
	}
	window := &models.WindowState{
		FileCount: 1,
		Files: []models.FileState{
			{Filename: "evidence.json", SizeBytes: 1000},
		},
	}
	result := models.NewEvaluationResult("ET-0001", "1", "2025-Q1", "all")

	evaluator.evaluateRequirementsMatch(task, window, result)

	// No specific keywords should give benefit of doubt
	assert.Greater(t, result.RequirementsMatch.Score, float64(20))
}

func TestEvidenceEvaluator_EvaluateControlAlignment_NoControls(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	task := &domain.EvidenceTask{
	ID: "1",
		Name:            "Test",
		RelatedControls: nil,
	}
	window := &models.WindowState{FileCount: 1}
	result := models.NewEvaluationResult("ET-0001", "1", "2025-Q1", "all")

	evaluator.evaluateControlAlignment(task, window, result)

	assert.Equal(t, float64(70), result.ControlAlignment.Score)
	assert.Equal(t, "pass", result.ControlAlignment.Status)
}

func TestEvidenceEvaluator_EvaluateControlAlignment_WithControls(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	task := &domain.EvidenceTask{
	ID: "1",
		Name: "Test",
		RelatedControls: []domain.Control{
			{Name: "Access Control Policies"},
			{Name: "Security Monitoring Configuration"},
		},
	}
	window := &models.WindowState{
		FileCount: 2,
		Files: []models.FileState{
			{Filename: "access_report.csv", SizeBytes: 1000},
			{Filename: "security_scan.json", SizeBytes: 2000},
		},
	}
	result := models.NewEvaluationResult("ET-0001", "1", "2025-Q1", "all")

	evaluator.evaluateControlAlignment(task, window, result)

	// Should have some score since files exist and match some keywords
	assert.Greater(t, result.ControlAlignment.Score, float64(30))
}

func TestEvidenceEvaluator_GenerateRecommendations(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	task := &domain.EvidenceTask{
	ID: "1",
		Name:            "Test",
		RelatedControls: []domain.Control{{Name: "C1"}, {Name: "C2"}},
	}

	result := models.NewEvaluationResult("ET-0001", "1", "2025-Q1", "all")
	result.Completeness.Score = 50
	result.RequirementsMatch.Score = 50
	result.QualityScore.Score = 50
	result.ControlAlignment.Score = 50
	result.FileCount = 1

	evaluator.generateRecommendations(task, result)

	assert.NotEmpty(t, result.Recommendations)
	// All dimensions below 70 should generate recommendations
	assert.GreaterOrEqual(t, len(result.Recommendations), 4)
}

func TestEvidenceEvaluator_GenerateRecommendations_CriticalIssues(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	task := &domain.EvidenceTask{ID: "1", Name: "Test"}
	result := models.NewEvaluationResult("ET-0001", "1", "2025-Q1", "all")
	result.Completeness.Score = 80
	result.RequirementsMatch.Score = 80
	result.QualityScore.Score = 80
	result.ControlAlignment.Score = 80
	result.FileCount = 5
	result.AddIssue(models.IssueCritical, "completeness", "No evidence files", "", "Upload files")

	evaluator.generateRecommendations(task, result)

	found := false
	for _, rec := range result.Recommendations {
		if rec == "Address all critical issues before submission" {
			found = true
		}
	}
	assert.True(t, found, "should recommend addressing critical issues")
}

func TestEvidenceEvaluator_HasProperNaming(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	tests := []struct {
		name     string
		filename string
		expected bool
	}{
		{"valid lowercase underscored", "access_control_report.csv", true},
		{"too short name", "a.csv", false},
		{"uppercase letters", "AccessControl.csv", false},
		{"has spaces", "my report.csv", false},
		{"valid simple name", "github_permissions.json", true},
		{"dashes allowed", "some-file-name.md", true},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, evaluator.hasProperNaming(tc.filename))
		})
	}
}

func TestEvidenceEvaluator_ExtractRequiredKeywords(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	task := &domain.EvidenceTask{
		Name:        "GitHub Access Controls",
		Description: "Show repository permissions and security audit log settings",
	}

	keywords := evaluator.extractRequiredKeywords(task)
	assert.Contains(t, keywords, "github")
	assert.Contains(t, keywords, "permissions")
	assert.Contains(t, keywords, "security")
	assert.Contains(t, keywords, "audit")
}

func TestEvidenceEvaluator_ExtractExpectedFormats(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	t.Run("default formats", func(t *testing.T) {
		task := &domain.EvidenceTask{Description: "Basic evidence"}
		formats := evaluator.extractExpectedFormats(task)
		assert.Contains(t, formats, "csv")
		assert.Contains(t, formats, "json")
		assert.Contains(t, formats, "md")
	})

	t.Run("screenshot mentioned", func(t *testing.T) {
		task := &domain.EvidenceTask{Description: "Provide screenshot of configuration"}
		formats := evaluator.extractExpectedFormats(task)
		assert.Contains(t, formats, "png")
		assert.Contains(t, formats, "jpg")
	})

	t.Run("spreadsheet mentioned", func(t *testing.T) {
		task := &domain.EvidenceTask{Description: "Provide a spreadsheet with user data"}
		formats := evaluator.extractExpectedFormats(task)
		assert.Contains(t, formats, "xlsx")
	})
}

func TestEvidenceEvaluator_CheckFileFormats(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	t.Run("all match", func(t *testing.T) {
		files := []models.FileState{
			{Filename: "data.csv"},
			{Filename: "config.json"},
		}
		score := evaluator.checkFileFormats(files, []string{"csv", "json", "md"})
		assert.Equal(t, 1.0, score)
	})

	t.Run("partial match", func(t *testing.T) {
		files := []models.FileState{
			{Filename: "data.csv"},
			{Filename: "image.png"},
		}
		score := evaluator.checkFileFormats(files, []string{"csv", "json"})
		assert.Equal(t, 0.5, score)
	})

	t.Run("no files", func(t *testing.T) {
		score := evaluator.checkFileFormats(nil, []string{"csv"})
		assert.Equal(t, 0.0, score)
	})
}

func TestEvidenceEvaluator_CalculateKeywordCoverage(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	t.Run("no keywords", func(t *testing.T) {
		t.Parallel()
		files := []models.FileState{{Filename: "test.csv"}}
		assert.Equal(t, 1.0, evaluator.calculateKeywordCoverage(files, nil))
	})

	t.Run("full coverage", func(t *testing.T) {
		t.Parallel()
		files := []models.FileState{
			{Filename: "access_control.csv"},
			{Filename: "security_report.json"},
		}
		coverage := evaluator.calculateKeywordCoverage(files, []string{"access", "security"})
		assert.Equal(t, 1.0, coverage)
	})

	t.Run("partial coverage", func(t *testing.T) {
		t.Parallel()
		files := []models.FileState{{Filename: "access_report.csv"}}
		coverage := evaluator.calculateKeywordCoverage(files, []string{"access", "security"})
		assert.Equal(t, 0.5, coverage)
	})
}

func TestEvidenceEvaluator_ExtractControlKeywords(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	controls := []domain.Control{
		{Name: "Access Control Enforcement"},
		{Name: "Data Encryption Standards"},
	}

	keywords := evaluator.extractControlKeywords(controls)
	assert.NotEmpty(t, keywords)
	// Should skip short words (<=4 chars)
	for _, kw := range keywords {
		assert.Greater(t, len(kw), 4)
	}
}

func TestEvidenceEvaluator_EstimateExpectedFileCount(t *testing.T) {
	t.Parallel()
	evaluator, _ := newTestEvaluator(t)

	t.Run("basic task", func(t *testing.T) {
		task := &domain.EvidenceTask{Description: "Simple task"}
		assert.Equal(t, 1, evaluator.estimateExpectedFileCount(task))
	})

	t.Run("multiple controls", func(t *testing.T) {
		task := &domain.EvidenceTask{
			RelatedControls: []domain.Control{{}, {}, {}, {}},
		}
		assert.Equal(t, 3, evaluator.estimateExpectedFileCount(task))
	})

	t.Run("long description adds a file", func(t *testing.T) {
		longDesc := ""
		for i := 0; i < 60; i++ {
			longDesc += "word word word word word "
		}
		task := &domain.EvidenceTask{Description: longDesc}
		assert.GreaterOrEqual(t, evaluator.estimateExpectedFileCount(task), 2)
	})
}

func TestEvidenceEvaluator_GetTaskDetails(t *testing.T) {
	t.Parallel()
	evaluator, store := newTestEvaluator(t)

	// Save a task
	task := &domain.EvidenceTask{
	ID: "42",
		ReferenceID: "ET-0042",
		Name:        "Test Task",
	}
	require.NoError(t, store.SaveEvidenceTask(task))

	t.Run("valid task ref", func(t *testing.T) {
		got, err := evaluator.getTaskDetails("ET-0042")
		require.NoError(t, err)
		assert.Equal(t, "Test Task", got.Name)
	})

	t.Run("invalid task ref format", func(t *testing.T) {
		_, err := evaluator.getTaskDetails("INVALID")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid task reference format")
	})
}

func TestEvidenceEvaluator_EvaluateWindow(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := newTestStorage(t, dir)
	log := newTestLogger(t)

	// Save a task for the evaluator to find
	task := &domain.EvidenceTask{
	ID: "1",
		ReferenceID: "ET-0001",
		Name:        "GitHub Access Controls",
		Description: "Collect GitHub access control evidence",
	}
	require.NoError(t, store.SaveEvidenceTask(task))

	// Create a scanner that returns good evidence
	now := time.Now()
	scanner := &stubEvidenceScanner{
		scanWindowState: &models.WindowState{
			Window:    "2025-Q1",
			FileCount: 3,
			TotalBytes: 10000,
			NewestFile: &now,
			Files: []models.FileState{
				{Filename: "github_permissions.csv", SizeBytes: 4000},
				{Filename: "access_controls.json", SizeBytes: 3000},
				{Filename: "evidence_summary.md", SizeBytes: 3000},
			},
		},
	}

	evaluator := NewEvidenceEvaluatorService(filepath.Join(dir, "evidence"), store, scanner, log)

	result, err := evaluator.EvaluateWindow(context.Background(), "ET-0001", "2025-Q1")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ET-0001", result.TaskRef)
	assert.Equal(t, "2025-Q1", result.Window)
	assert.Greater(t, result.OverallScore, float64(0))
	assert.Equal(t, 3, result.FileCount)
}

// ---------------------------------------------------------------------------
// evidence_scanner.go – Scanner tests
// ---------------------------------------------------------------------------

func TestEvidenceScanner_ScanAll_EmptyDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "evidence")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))

	log := newTestLogger(t)
	scanner := NewEvidenceScanner(evidenceDir, nil, log)

	result, err := scanner.ScanAll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestEvidenceScanner_ScanAll_NonexistentDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "nonexistent")

	log := newTestLogger(t)
	scanner := NewEvidenceScanner(evidenceDir, nil, log)

	result, err := scanner.ScanAll(context.Background())
	require.NoError(t, err)
	assert.Empty(t, result)
}

func TestEvidenceScanner_ScanAll_WithTaskDirs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "evidence")

	// Create task directory with window and evidence files
	taskDir := filepath.Join(evidenceDir, "GitHub_Access_ET-0001_100")
	windowDir := filepath.Join(taskDir, "2025-Q1")
	require.NoError(t, os.MkdirAll(windowDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(windowDir, "evidence.csv"), []byte("col1,col2\nval1,val2"), 0644))
	require.NoError(t, os.WriteFile(filepath.Join(windowDir, "report.json"), []byte(`{"key":"value"}`), 0644))

	log := newTestLogger(t)
	scanner := NewEvidenceScanner(evidenceDir, nil, log)

	result, err := scanner.ScanAll(context.Background())
	require.NoError(t, err)
	assert.Len(t, result, 1)

	taskState, ok := result["ET-0001"]
	assert.True(t, ok)
	assert.Equal(t, "ET-0001", taskState.TaskRef)
	assert.Contains(t, taskState.Windows, "2025-Q1")
	assert.Equal(t, 2, taskState.Windows["2025-Q1"].FileCount)
}

func TestEvidenceScanner_ScanTask_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "evidence")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))

	log := newTestLogger(t)
	scanner := NewEvidenceScanner(evidenceDir, nil, log)

	taskState, err := scanner.ScanTask(context.Background(), "ET-9999")
	require.NoError(t, err)
	assert.Equal(t, models.StateNoEvidence, taskState.LocalState)
}

func TestEvidenceScanner_ScanWindow_Flat(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "evidence")

	taskDir := filepath.Join(evidenceDir, "Test_Task_ET-0005_500")
	windowDir := filepath.Join(taskDir, "2025-Q4")
	require.NoError(t, os.MkdirAll(windowDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(windowDir, "data.csv"), []byte("a,b\n1,2\n"), 0644))

	log := newTestLogger(t)
	scanner := NewEvidenceScanner(evidenceDir, nil, log)

	ws, err := scanner.ScanWindow(context.Background(), "ET-0005", "2025-Q4")
	require.NoError(t, err)
	assert.Equal(t, 1, ws.FileCount)
	assert.Equal(t, "2025-Q4", ws.Window)
	assert.Greater(t, ws.TotalBytes, int64(0))
}

func TestEvidenceScanner_ScanWindow_WithGenerationMetadata(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "evidence")

	taskDir := filepath.Join(evidenceDir, "Test_Task_ET-0006_600")
	windowDir := filepath.Join(taskDir, "2025-Q2")
	genDir := filepath.Join(windowDir, ".generation")
	require.NoError(t, os.MkdirAll(genDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(windowDir, "evidence.json"), []byte(`{"test":true}`), 0644))

	// Write generation metadata
	genMeta := models.GenerationMetadata{
		GeneratedAt:      time.Now(),
		GeneratedBy:      "grctool-cli",
		GenerationMethod: "tool_coordination",
		ToolsUsed:        []string{"github-permissions"},
	}
	data, err := yaml.Marshal(genMeta)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(genDir, "metadata.yaml"), data, 0644))

	log := newTestLogger(t)
	scanner := NewEvidenceScanner(evidenceDir, nil, log)

	ws, err := scanner.ScanWindow(context.Background(), "ET-0006", "2025-Q2")
	require.NoError(t, err)
	assert.True(t, ws.HasGenerationMeta)
	assert.Equal(t, "tool_coordination", ws.GenerationMethod)
	assert.Equal(t, "grctool-cli", ws.GeneratedBy)
	assert.Contains(t, ws.ToolsUsed, "github-permissions")
}

func TestEvidenceScanner_ScanWindow_WithSubmissionMetadata(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "evidence")

	taskDir := filepath.Join(evidenceDir, "Test_Task_ET-0007_700")
	windowDir := filepath.Join(taskDir, "2025-Q3")
	subDir := filepath.Join(windowDir, ".submission")
	require.NoError(t, os.MkdirAll(subDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(windowDir, "evidence.csv"), []byte("a,b\n"), 0644))

	// Write submission metadata
	subMeta := models.EvidenceSubmission{
		Status: "submitted",
	}
	data, err := yaml.Marshal(subMeta)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(subDir, "submission.yaml"), data, 0644))

	log := newTestLogger(t)
	scanner := NewEvidenceScanner(evidenceDir, nil, log)

	ws, err := scanner.ScanWindow(context.Background(), "ET-0007", "2025-Q3")
	require.NoError(t, err)
	assert.True(t, ws.HasSubmissionMeta)
	assert.Equal(t, "submitted", ws.SubmissionStatus)
}

func TestEvidenceScanner_ScanWindow_HybridStructure(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "evidence")

	taskDir := filepath.Join(evidenceDir, "Hybrid_Task_ET-0008_800")
	windowDir := filepath.Join(taskDir, "2025-Q1")
	submittedDir := filepath.Join(windowDir, ".submitted")
	archiveDir := filepath.Join(windowDir, "archive")

	require.NoError(t, os.MkdirAll(submittedDir, 0755))
	require.NoError(t, os.MkdirAll(archiveDir, 0755))

	// Files in root
	require.NoError(t, os.WriteFile(filepath.Join(windowDir, "working.csv"), []byte("data"), 0644))
	// Files in .submitted
	require.NoError(t, os.WriteFile(filepath.Join(submittedDir, "submitted.json"), []byte("{}"), 0644))
	// Files in archive
	require.NoError(t, os.WriteFile(filepath.Join(archiveDir, "archived.pdf"), []byte("pdf"), 0644))

	log := newTestLogger(t)
	scanner := NewEvidenceScanner(evidenceDir, nil, log)

	ws, err := scanner.ScanWindow(context.Background(), "ET-0008", "2025-Q1")
	require.NoError(t, err)
	assert.Equal(t, 3, ws.FileCount)
	assert.Greater(t, ws.TotalBytes, int64(0))
}

func TestEvidenceScanner_ScanWindow_NotFound(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "evidence")

	taskDir := filepath.Join(evidenceDir, "Test_ET-0009_900")
	require.NoError(t, os.MkdirAll(taskDir, 0755))

	log := newTestLogger(t)
	scanner := NewEvidenceScanner(evidenceDir, nil, log)

	_, err := scanner.ScanWindow(context.Background(), "ET-0009", "2025-Q4")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "window directory not found")
}

func TestEvidenceScanner_ScanTask_MultipleWindows(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "evidence")

	taskDir := filepath.Join(evidenceDir, "Multi_Window_ET-0010_1000")
	for _, w := range []string{"2024-Q4", "2025-Q1", "2025-Q2"} {
		wDir := filepath.Join(taskDir, w)
		require.NoError(t, os.MkdirAll(wDir, 0755))
		require.NoError(t, os.WriteFile(filepath.Join(wDir, "evidence.csv"), []byte("test"), 0644))
	}

	log := newTestLogger(t)
	scanner := NewEvidenceScanner(evidenceDir, nil, log)

	taskState, err := scanner.ScanTask(context.Background(), "ET-0010")
	require.NoError(t, err)
	assert.Len(t, taskState.Windows, 3)
	assert.Contains(t, taskState.Windows, "2024-Q4")
	assert.Contains(t, taskState.Windows, "2025-Q1")
	assert.Contains(t, taskState.Windows, "2025-Q2")
}

func TestEvidenceScanner_ScanTask_IgnoresHiddenAndNonWindowDirs(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "evidence")

	taskDir := filepath.Join(evidenceDir, "Test_ET-0011_1100")
	// Valid window
	require.NoError(t, os.MkdirAll(filepath.Join(taskDir, "2025-Q1"), 0755))
	// Hidden dir - should be skipped
	require.NoError(t, os.MkdirAll(filepath.Join(taskDir, ".context"), 0755))
	// Non-window dir - should be skipped
	require.NoError(t, os.MkdirAll(filepath.Join(taskDir, "notes"), 0755))

	log := newTestLogger(t)
	scanner := NewEvidenceScanner(evidenceDir, nil, log)

	taskState, err := scanner.ScanTask(context.Background(), "ET-0011")
	require.NoError(t, err)
	assert.Len(t, taskState.Windows, 1)
	assert.Contains(t, taskState.Windows, "2025-Q1")
}

func TestEvidenceScanner_ContextCancellation(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "evidence")

	// Create many task directories
	for i := 0; i < 10; i++ {
		taskDir := filepath.Join(evidenceDir, filepath.Join(evidenceDir, "Task_ET-"+string(rune('0'+i))+"_"+string(rune('0'+i))))
		wDir := filepath.Join(taskDir, "2025-Q1")
		os.MkdirAll(wDir, 0755)
		os.WriteFile(filepath.Join(wDir, "data.csv"), []byte("test"), 0644)
	}

	log := newTestLogger(t)
	scanner := NewEvidenceScanner(evidenceDir, nil, log)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	_, err := scanner.ScanAll(ctx)
	assert.Error(t, err)
}

func TestUpdateTimestamps(t *testing.T) {
	t.Parallel()

	t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	t3 := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)

	t.Run("nil to non-nil", func(t *testing.T) {
		var oldest, newest *time.Time
		updateTimestamps(&oldest, &newest, &t1, &t2)
		assert.Equal(t, t1, *oldest)
		assert.Equal(t, t2, *newest)
	})

	t.Run("updates oldest", func(t *testing.T) {
		oldest := &t2
		newest := &t3
		updateTimestamps(&oldest, &newest, &t1, &t2)
		assert.Equal(t, t1, *oldest)
		assert.Equal(t, t3, *newest)
	})

	t.Run("updates newest", func(t *testing.T) {
		oldest := &t1
		newest := &t2
		updateTimestamps(&oldest, &newest, &t2, &t3)
		assert.Equal(t, t1, *oldest)
		assert.Equal(t, t3, *newest)
	})

	t.Run("nil input ignored", func(t *testing.T) {
		oldest := &t1
		newest := &t2
		updateTimestamps(&oldest, &newest, nil, nil)
		assert.Equal(t, t1, *oldest)
		assert.Equal(t, t2, *newest)
	})
}

// ---------------------------------------------------------------------------
// evidence.go – EvidenceServiceSimple tests
// ---------------------------------------------------------------------------

func TestEvidenceServiceSimple_NewEvidenceService(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	cfg := &config.Config{}
	log := newTestLogger(t)

	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)
	assert.NotNil(t, svc)
}

func TestEvidenceServiceSimple_GenerateEvidence_Markdown(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	stub.tasks["42"] = &domain.EvidenceTask{
	ID: "42",
		ReferenceID: "ET-0042",
		Name:        "Test Task",
		Description: "Task description",
		Framework:   "SOC2",
		Priority:    "high",
		Guidance:    "Collect screenshots",
	}

	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	req := &EvidenceGenerationRequest{
		TaskID: "42",
		Title:       "Test Evidence",
		Description: "Generated evidence",
		Format:      "markdown",
		Tools:       []string{"github-permissions"},
	}

	result, err := svc.GenerateEvidence(context.Background(), req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotNil(t, result.Record)
	assert.Equal(t, "42", result.Record.TaskID)
	assert.Contains(t, result.Record.Content, "# Evidence for Task: Test Task")
	assert.Contains(t, result.Record.Content, "Collection Guidance")
}

func TestEvidenceServiceSimple_GenerateEvidence_CSV(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	stub.tasks["1"] = &domain.EvidenceTask{
	ID: "1",
		Name: "CSV Task",
	}

	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	req := &EvidenceGenerationRequest{
		TaskID: "1",
		Title:  "CSV Evidence",
		Format: "csv",
	}

	result, err := svc.GenerateEvidence(context.Background(), req)
	require.NoError(t, err)
	assert.Contains(t, result.Record.Content, "Control/Requirement")
	assert.Contains(t, result.Record.Content, "CSV Task")
}

func TestEvidenceServiceSimple_GenerateEvidence_JSON(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	stub.tasks["2"] = &domain.EvidenceTask{
	ID: "2",
		Name:      "JSON Task",
		Framework: "SOC2",
		Priority:  "medium",
	}

	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	req := &EvidenceGenerationRequest{
		TaskID: "2",
		Title:  "JSON Evidence",
		Format: "json",
	}

	result, err := svc.GenerateEvidence(context.Background(), req)
	require.NoError(t, err)
	assert.Contains(t, result.Record.Content, `"task_name"`)
	assert.Contains(t, result.Record.Content, "JSON Task")
}

func TestEvidenceServiceSimple_GenerateEvidence_TaskNotFound(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	req := &EvidenceGenerationRequest{TaskID: "999"}
	_, err = svc.GenerateEvidence(context.Background(), req)
	assert.Error(t, err)
}

func TestEvidenceServiceSimple_ListEvidenceTasks(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	stub.tasks["1"] = &domain.EvidenceTask{ID: "1", Status: "pending"}
	stub.tasks["2"] = &domain.EvidenceTask{ID: "2", Status: "completed"}
	stub.tasks["3"] = &domain.EvidenceTask{ID: "3", Status: "pending"}

	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	tasks, err := svc.ListEvidenceTasks(context.Background(), domain.EvidenceFilter{Status: []string{"pending"}})
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestEvidenceServiceSimple_GetEvidenceTaskSummary(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()

	now := time.Now()
	overdue := now.Add(-24 * time.Hour)
	dueSoon := now.Add(3 * 24 * time.Hour)
	far := now.Add(30 * 24 * time.Hour)

	stub.tasks["1"] = &domain.EvidenceTask{ID: "1", Status: "pending", Priority: "high", NextDue: &overdue}
	stub.tasks["2"] = &domain.EvidenceTask{ID: "2", Status: "completed", Priority: "low", NextDue: &far}
	stub.tasks["3"] = &domain.EvidenceTask{ID: "3", Status: "pending", Priority: "high", NextDue: &dueSoon}

	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	summary, err := svc.GetEvidenceTaskSummary(context.Background())
	require.NoError(t, err)
	assert.Equal(t, 3, summary.Total)
	assert.Equal(t, 2, summary.ByStatus["pending"])
	assert.Equal(t, 1, summary.ByStatus["completed"])
	assert.Equal(t, 2, summary.ByPriority["high"])
	assert.Equal(t, 1, summary.Overdue)
	assert.Equal(t, 1, summary.DueSoon)
}

func TestEvidenceServiceSimple_AnalyzeEvidenceTask(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	stub.tasks["10"] = &domain.EvidenceTask{
	ID: "10",
		Name:        "GitHub Access Controls",
		Description: "Show GitHub repository access",
		Framework:   "SOC2",
		Priority:    "high",
		Controls:    []string{"AC-01"},
		Policies:    []string{"P1"},
	}
	stub.controls["AC-01"] = &domain.Control{
	ID: "1",
		ReferenceID: "AC-01",
		Name:        "Access Control",
		FrameworkCodes: []domain.FrameworkCode{
			{Code: "CC6.1", Framework: "SOC2"},
		},
	}
	stub.policies["P1"] = &domain.Policy{
		ID:   "P1",
		Name: "Access Policy",
	}

	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	analysis, err := svc.AnalyzeEvidenceTask(context.Background(), "10")
	require.NoError(t, err)
	assert.Equal(t, "10", analysis.TaskID)
	assert.Len(t, analysis.RelatedControls, 1)
	assert.Len(t, analysis.RelatedPolicies, 1)
	assert.NotEmpty(t, analysis.SuggestedTools)
	assert.NotEmpty(t, analysis.Recommendations)
	assert.NotEmpty(t, analysis.RequiredEvidence)
	assert.NotNil(t, analysis.ComplianceContext)
	assert.Equal(t, "SOC2", analysis.ComplianceContext["framework"])
}

func TestEvidenceServiceSimple_AnalyzeEvidenceTask_FallbackPolicies(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	stub.tasks["20"] = &domain.EvidenceTask{
	ID: "20",
		Name:      "Fallback Task",
		Framework: "SOC2",
		// No direct policy references
	}
	stub.policies["P1"] = &domain.Policy{ID: "P1", Name: "SOC2 Policy", Framework: "SOC2"}
	stub.policies["P2"] = &domain.Policy{ID: "P2", Name: "ISO Policy", Framework: "ISO27001"}

	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	analysis, err := svc.AnalyzeEvidenceTask(context.Background(), "20")
	require.NoError(t, err)
	// Should find policies by framework matching
	assert.Len(t, analysis.RelatedPolicies, 1)
	assert.Equal(t, "SOC2 Policy", analysis.RelatedPolicies[0].Name)
}

func TestEvidenceServiceSimple_ReviewEvidence(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	t.Run("without reasoning", func(t *testing.T) {
		review, err := svc.ReviewEvidence(context.Background(), "rec-1", false)
		require.NoError(t, err)
		assert.Equal(t, "moderate", review["completeness"])
		assert.Nil(t, review["reasoning"])
	})

	t.Run("with reasoning", func(t *testing.T) {
		review, err := svc.ReviewEvidence(context.Background(), "rec-1", true)
		require.NoError(t, err)
		assert.NotNil(t, review["reasoning"])
		reasoning := review["reasoning"].(map[string]interface{})
		assert.Equal(t, "template-based", reasoning["analysis_method"])
	})
}

func TestEvidenceServiceSimple_BuildSuggestedTools(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	t.Run("infrastructure category", func(t *testing.T) {
		task := &domain.EvidenceTask{
			Name:        "Firewall Configuration",
			Description: "Evidence of firewall configuration and monitoring",
			Category:    "Infrastructure",
		}
		tools := svc.buildSuggestedTools(task, nil)
		assert.Contains(t, tools, "terraform-security-analyzer")
	})

	t.Run("CC6 controls add github-permissions", func(t *testing.T) {
		task := &domain.EvidenceTask{Name: "Access", Policies: []string{"P1"}}
		controls := []domain.Control{
			{FrameworkCodes: []domain.FrameworkCode{{Code: "CC6.1"}}},
		}
		tools := svc.buildSuggestedTools(task, controls)
		assert.Contains(t, tools, "github-permissions")
		assert.Contains(t, tools, "terraform-security-analyzer")
	})

	t.Run("CC8 controls add workflow analyzer", func(t *testing.T) {
		task := &domain.EvidenceTask{Name: "Change Mgmt"}
		controls := []domain.Control{
			{FrameworkCodes: []domain.FrameworkCode{{Code: "CC8.1"}}},
		}
		tools := svc.buildSuggestedTools(task, controls)
		assert.Contains(t, tools, "github-workflow-analyzer")
	})
}

func TestEvidenceServiceSimple_ExtractRequiredEvidence(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	t.Run("with guidance keywords", func(t *testing.T) {
		task := &domain.EvidenceTask{
			Guidance: "Provide screenshot of audit log configuration",
		}
		evidence := svc.extractRequiredEvidence(task)
		assert.Contains(t, evidence, "Screenshots of relevant configurations")
		assert.Contains(t, evidence, "Audit logs and access records")
	})

	t.Run("without guidance", func(t *testing.T) {
		task := &domain.EvidenceTask{}
		evidence := svc.extractRequiredEvidence(task)
		assert.GreaterOrEqual(t, len(evidence), 3)
		assert.Contains(t, evidence, "Implementation documentation")
	})
}

func TestEvidenceServiceSimple_DeriveTrustServiceCategory(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	t.Run("CC6 controls", func(t *testing.T) {
		controls := []domain.Control{
			{FrameworkCodes: []domain.FrameworkCode{{Code: "CC6.1"}, {Code: "CC6.2"}}},
		}
		cat := svc.deriveTrustServiceCategory(controls)
		assert.Equal(t, "Logical and Physical Access Controls", cat)
	})

	t.Run("CC7 controls", func(t *testing.T) {
		controls := []domain.Control{
			{FrameworkCodes: []domain.FrameworkCode{{Code: "CC7.1"}}},
		}
		cat := svc.deriveTrustServiceCategory(controls)
		assert.Equal(t, "System Operations", cat)
	})

	t.Run("no CC codes", func(t *testing.T) {
		controls := []domain.Control{
			{FrameworkCodes: []domain.FrameworkCode{{Code: "A.9.1"}}},
		}
		cat := svc.deriveTrustServiceCategory(controls)
		assert.Equal(t, "Common Criteria", cat)
	})
}

func TestEvidenceServiceSimple_SaveGeneratedEvidence(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	stub := newStubDataService()
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				OutputDir:        dir,
				IncludeReasoning: true,
			},
		},
	}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	evidence := &models.GeneratedEvidence{
		TaskID: "55",
		GeneratedAt:     time.Now(),
		GeneratedBy:     "test",
		EvidenceFormat:  "markdown",
		EvidenceContent: "# Test Evidence\n\nThis is test content.",
		Reasoning:       "Test reasoning",
	}

	err = svc.saveGeneratedEvidence(evidence)
	require.NoError(t, err)

	taskDir := filepath.Join(dir, "ET55")
	assert.DirExists(t, taskDir)

	// Check evidence file exists
	entries, err := os.ReadDir(taskDir)
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(entries), 1)

	// Reasoning file should exist
	foundReasoning := false
	for _, e := range entries {
		if filepath.Ext(e.Name()) == ".md" {
			foundReasoning = true
		}
	}
	assert.True(t, foundReasoning)
}

func TestEvidenceServiceSimple_SaveGeneratedEvidence_NoOutputDir(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	cfg := &config.Config{
		Evidence: config.EvidenceConfig{
			Generation: config.GenerationConfig{
				OutputDir: "", // empty
			},
		},
	}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	evidence := &models.GeneratedEvidence{
		TaskID: "1",
		GeneratedAt:     time.Now(),
		EvidenceContent: "content",
	}

	err = svc.saveGeneratedEvidence(evidence)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "output directory not configured")
}

func TestEvidenceServiceSimple_ContainsString(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	assert.True(t, svc.containsString([]string{"a", "b", "c"}, "b"))
	assert.False(t, svc.containsString([]string{"a", "b", "c"}, "d"))
	assert.False(t, svc.containsString(nil, "a"))
}

// ---------------------------------------------------------------------------
// evidence_scanner.go – exported helper function tests
// ---------------------------------------------------------------------------

func TestExtractTaskIDFromRef_Extended(t *testing.T) {
	t.Parallel()
	tests := []struct {
		ref      string
		expected string
	}{
		{"ET-0001", "1"},
		{"ET-0100", "100"},
		{"ET-9999", "9999"},
		{"ET-0000", "0"},
		{"ET-", ""},
		{"", ""},
		{"INVALID", ""},
		{"et-0001", ""}, // case sensitive
	}

	for _, tc := range tests {
		t.Run(tc.ref, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, extractTaskIDFromRef(tc.ref))
		})
	}
}

func TestIsWindowDirectory_Extended(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		expected bool
	}{
		{"2025", true},
		{"2025-Q1", true},
		{"2025-Q4", true},
		{"2025-01", true},
		{"2025-12", true},
		{"2025-H1", true},
		{"2025-H2", true},
		{"2025-Q0", false},
		{"2025-Q5", false},
		{"2025-00", false},
		{"2025-13", false},
		{"2025-H0", false},
		{"2025-H3", false},
		{"random", false},
		{"ET-0001", false},
		{".hidden", false},
		{"", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tc.expected, isWindowDirectory(tc.name))
		})
	}
}

// ---------------------------------------------------------------------------
// evidence.go – EvidenceAnalysisResult / EvidenceGenerationRequest type tests
// ---------------------------------------------------------------------------

func TestEvidenceGenerationRequest_Fields(t *testing.T) {
	t.Parallel()
	req := EvidenceGenerationRequest{
		TaskID: "42",
		Title:       "Test",
		Description: "Desc",
		Format:      "csv",
		Tools:       []string{"terraform-security-analyzer"},
		Context:     map[string]interface{}{"key": "val"},
	}
	assert.Equal(t, "42", req.TaskID)
	assert.Equal(t, "csv", req.Format)
	assert.Len(t, req.Tools, 1)
	assert.Equal(t, "val", req.Context["key"])
}

// ---------------------------------------------------------------------------
// document.go – DocumentType constants test
// ---------------------------------------------------------------------------

func TestDocumentTypeConstants(t *testing.T) {
	t.Parallel()
	assert.Equal(t, DocumentType("policy_documents"), PolicyDocument)
	assert.Equal(t, DocumentType("control_documents"), ControlDocument)
	assert.Equal(t, DocumentType("evidence_task_documents"), EvidenceTaskDocument)
	assert.Equal(t, DocumentType("evidence_prompts"), EvidencePromptDocument)
}

// ---------------------------------------------------------------------------
// Additional matchesEvidenceFilter tests for category/complexity/collection
// ---------------------------------------------------------------------------

func TestMatchesEvidenceFilter_CategoryAndComplexity(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := newTestStorage(t, dir)
	impl := NewDataService(store).(*DataServiceImpl)

	t.Run("category filter match", func(t *testing.T) {
		t.Parallel()
		task := domain.EvidenceTask{
			Name:        "Firewall Configuration Evidence",
			Description: "Collect firewall config",
			Category:    "Infrastructure",
		}
		filter := domain.EvidenceFilter{Category: []string{"Infrastructure"}}
		assert.True(t, impl.matchesEvidenceFilter(task, filter))
	})

	t.Run("category filter no match", func(t *testing.T) {
		t.Parallel()
		task := domain.EvidenceTask{
			Category: "Personnel",
		}
		filter := domain.EvidenceFilter{Category: []string{"Infrastructure"}}
		assert.False(t, impl.matchesEvidenceFilter(task, filter))
	})

	t.Run("complexity filter match", func(t *testing.T) {
		t.Parallel()
		task := domain.EvidenceTask{
			ComplexityLevel: "Complex",
		}
		filter := domain.EvidenceFilter{ComplexityLevel: []string{"Complex", "Moderate"}}
		assert.True(t, impl.matchesEvidenceFilter(task, filter))
	})

	t.Run("complexity filter no match", func(t *testing.T) {
		t.Parallel()
		task := domain.EvidenceTask{
			ComplexityLevel: "Simple",
		}
		filter := domain.EvidenceFilter{ComplexityLevel: []string{"Complex"}}
		assert.False(t, impl.matchesEvidenceFilter(task, filter))
	})

	t.Run("collection type filter match", func(t *testing.T) {
		t.Parallel()
		task := domain.EvidenceTask{
			CollectionType: "Automated",
		}
		filter := domain.EvidenceFilter{CollectionType: []string{"Automated", "Hybrid"}}
		assert.True(t, impl.matchesEvidenceFilter(task, filter))
	})

	t.Run("collection type filter no match", func(t *testing.T) {
		t.Parallel()
		task := domain.EvidenceTask{
			CollectionType: "Manual",
		}
		filter := domain.EvidenceFilter{CollectionType: []string{"Automated"}}
		assert.False(t, impl.matchesEvidenceFilter(task, filter))
	})

	t.Run("combined filters", func(t *testing.T) {
		t.Parallel()
		task := domain.EvidenceTask{
			Status:          "pending",
			Priority:        "high",
			Framework:       "SOC2",
			Sensitive:       true,
			Category:        "Infrastructure",
			ComplexityLevel: "Complex",
			CollectionType:  "Automated",
		}
		boolTrue := true
		filter := domain.EvidenceFilter{
			Status:          []string{"pending"},
			Priority:        []string{"high"},
			Framework:       "SOC2",
			Sensitive:       &boolTrue,
			Category:        []string{"Infrastructure"},
			ComplexityLevel: []string{"Complex"},
			CollectionType:  []string{"Automated"},
		}
		assert.True(t, impl.matchesEvidenceFilter(task, filter))
	})
}

// ---------------------------------------------------------------------------
// sync.go – getDisplayName with real tugboat members
// ---------------------------------------------------------------------------

func TestSyncService_GetDisplayName_WithFields(t *testing.T) {
	t.Parallel()
	s := &SyncService{}

	// We need to import the tugboat models type
	// Test with different member configurations
	// Since getDisplayName takes *tugboatModels.OrganizationMember, we test via the struct
	// Unfortunately, the method takes a specific type from tugboat/models
	// We can still test nil
	result := s.getDisplayName(nil)
	assert.Equal(t, "Unknown", result)
}

// ---------------------------------------------------------------------------
// sync.go – convertAttachmentsToSubmission
// ---------------------------------------------------------------------------

func TestSyncService_ConvertAttachmentsToSubmission(t *testing.T) {
	t.Parallel()
	s := &SyncService{}

	// Empty attachments
	result := s.convertAttachmentsToSubmission("ET-0001", "task_dir", "100", "2025-Q1", nil)
	assert.Equal(t, "ET-0001", result.TaskRef)
	assert.Equal(t, "100", result.TaskID)
	assert.Equal(t, "2025-Q1", result.Window)
	assert.Equal(t, "accepted", result.Status)
	assert.Equal(t, 0, result.TotalFileCount)
	assert.Equal(t, 1.0, result.CompletenessScore)
}

// ---------------------------------------------------------------------------
// evidence_scanner.go – findLatestTimestamps
// ---------------------------------------------------------------------------

func TestFindLatestTimestamps(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	log := newTestLogger(t)
	scanner := NewEvidenceScanner(filepath.Join(dir, "evidence"), nil, log).(*evidenceScannerImpl)

	t.Run("empty windows", func(t *testing.T) {
		gen, sub := scanner.findLatestTimestamps(nil)
		assert.Nil(t, gen)
		assert.Nil(t, sub)
	})

	t.Run("with timestamps", func(t *testing.T) {
		t1 := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
		t2 := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
		t3 := time.Date(2025, 3, 1, 0, 0, 0, 0, time.UTC)

		windows := map[string]models.WindowState{
			"2025-Q1": {GeneratedAt: &t1, SubmittedAt: &t3},
			"2025-Q2": {GeneratedAt: &t2},
		}

		gen, sub := scanner.findLatestTimestamps(windows)
		assert.NotNil(t, gen)
		assert.Equal(t, t2, *gen)
		assert.NotNil(t, sub)
		assert.Equal(t, t3, *sub)
	})
}

// ---------------------------------------------------------------------------
// evidence_scanner.go – determineAutomationLevel
// ---------------------------------------------------------------------------

func TestDetermineAutomationLevel(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	log := newTestLogger(t)
	scanner := NewEvidenceScanner(filepath.Join(dir, "evidence"), nil, log).(*evidenceScannerImpl)

	t.Run("no task no tools", func(t *testing.T) {
		windows := map[string]models.WindowState{
			"2025-Q1": {},
		}
		level := scanner.determineAutomationLevel(nil, windows)
		assert.Equal(t, models.AutomationUnknown, level)
	})

	t.Run("with tools in windows", func(t *testing.T) {
		windows := map[string]models.WindowState{
			"2025-Q1": {ToolsUsed: []string{"github-permissions", "terraform-analyzer"}},
		}
		level := scanner.determineAutomationLevel(nil, windows)
		assert.Equal(t, models.AutomationFully, level)
	})

	t.Run("with task description", func(t *testing.T) {
		task := &domain.EvidenceTask{
			Description: "Collect infrastructure monitoring and access log data",
		}
		windows := map[string]models.WindowState{
			"2025-Q1": {ToolsUsed: []string{"terraform-analyzer"}},
		}
		level := scanner.determineAutomationLevel(task, windows)
		assert.NotEqual(t, models.AutomationUnknown, level)
	})
}

// ---------------------------------------------------------------------------
// evidence_scanner.go – detectApplicableTools
// ---------------------------------------------------------------------------

func TestDetectApplicableTools(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	log := newTestLogger(t)
	scanner := NewEvidenceScanner(filepath.Join(dir, "evidence"), nil, log).(*evidenceScannerImpl)

	t.Run("github task", func(t *testing.T) {
		task := &domain.EvidenceTask{
			Name:        "GitHub Repository Access",
			Description: "Show permissions for GitHub repositories",
		}
		tools := scanner.detectApplicableTools(task, nil)
		assert.Contains(t, tools, "github-permissions")
	})

	t.Run("terraform task", func(t *testing.T) {
		task := &domain.EvidenceTask{
			Name:        "Infrastructure Security",
			Description: "Terraform IAM security configuration",
		}
		tools := scanner.detectApplicableTools(task, nil)
		assert.Contains(t, tools, "terraform-security-analyzer")
	})

	t.Run("google workspace task", func(t *testing.T) {
		task := &domain.EvidenceTask{
			Name:        "Google Workspace Access",
			Description: "Show Google Drive docs sharing",
		}
		tools := scanner.detectApplicableTools(task, nil)
		assert.Contains(t, tools, "google-workspace")
	})

	t.Run("atmos task", func(t *testing.T) {
		task := &domain.EvidenceTask{
			Name:        "Multi-Environment Stack",
			Description: "Atmos stack analysis for multi-environment deployment",
		}
		tools := scanner.detectApplicableTools(task, nil)
		assert.Contains(t, tools, "atmos-stack-analyzer")
	})

	t.Run("no task info uses existing tools", func(t *testing.T) {
		windows := map[string]models.WindowState{
			"2025-Q1": {ToolsUsed: []string{"existing-tool"}},
		}
		tools := scanner.detectApplicableTools(nil, windows)
		assert.Contains(t, tools, "existing-tool")
	})
}

// ---------------------------------------------------------------------------
// evidence_scanner.go – readValidationMetadata (via ScanWindow with validation dir)
// ---------------------------------------------------------------------------

func TestEvidenceScanner_ScanWindow_WithValidationMetadata(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	evidenceDir := filepath.Join(dir, "evidence")

	taskDir := filepath.Join(evidenceDir, "Test_ET-0012_1200")
	windowDir := filepath.Join(taskDir, "2025-Q2")
	valDir := filepath.Join(windowDir, ".validation")
	// Also create .submitted to trigger hybrid scanning
	submittedDir := filepath.Join(windowDir, ".submitted")
	require.NoError(t, os.MkdirAll(valDir, 0755))
	require.NoError(t, os.MkdirAll(submittedDir, 0755))
	require.NoError(t, os.WriteFile(filepath.Join(windowDir, "evidence.csv"), []byte("a,b"), 0644))

	// Write validation metadata
	valMeta := models.ValidationResult{
		TaskRef: "ET-0012",
		Status:  "passed",
	}
	data, err := yaml.Marshal(valMeta)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(filepath.Join(valDir, "validation.yaml"), data, 0644))

	log := newTestLogger(t)
	scanner := NewEvidenceScanner(evidenceDir, nil, log)

	ws, err := scanner.ScanWindow(context.Background(), "ET-0012", "2025-Q2")
	require.NoError(t, err)
	// When validation metadata exists in hybrid mode, SubmissionStatus gets set
	assert.Equal(t, "validated", ws.SubmissionStatus)
}

// ---------------------------------------------------------------------------
// evidence_evaluator.go – EvaluateSubfolder
// ---------------------------------------------------------------------------

func TestEvidenceEvaluator_EvaluateSubfolder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	store := newTestStorage(t, dir)
	log := newTestLogger(t)

	// Save a task
	task := &domain.EvidenceTask{
	ID: "50",
		ReferenceID: "ET-0050",
		Name:        "Subfolder Task",
		Description: "Test subfolder evaluation",
	}
	require.NoError(t, store.SaveEvidenceTask(task))

	// The EvaluateSubfolder method calls storage.GetEvidenceFilesFromSubfolder
	// which looks for files in a specific subdirectory structure.
	// For this test we just need the storage and task to exist.
	scanner := &stubEvidenceScanner{}
	evaluator := NewEvidenceEvaluatorService(filepath.Join(dir, "evidence"), store, scanner, log)

	// This will get an empty file list from storage (no files in subfolder)
	result, err := evaluator.EvaluateSubfolder(context.Background(), "ET-0050", "2025-Q1", ".submitted")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, "ET-0050", result.TaskRef)
	assert.Equal(t, ".submitted", result.Subfolder)
	// With no files, completeness should be 0
	assert.Equal(t, float64(0), result.Completeness.Score)
}

// ---------------------------------------------------------------------------
// evidence.go – buildComplianceContext SOC2 branch
// ---------------------------------------------------------------------------

func TestEvidenceServiceSimple_BuildComplianceContext_SOC2(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	task := &domain.EvidenceTask{
		Framework: "SOC2",
		Priority:  "high",
		Status:    "pending",
	}
	controls := []domain.Control{
		{FrameworkCodes: []domain.FrameworkCode{{Code: "CC6.1"}, {Code: "CC6.2"}}},
	}
	policies := []domain.Policy{{Name: "Policy 1"}}

	ctx := svc.buildComplianceContext(task, controls, policies)
	assert.Equal(t, "SOC2", ctx["framework"])
	assert.NotNil(t, ctx["trust_service_category"])
	assert.Equal(t, 1, ctx["control_count"])
	assert.Equal(t, 1, ctx["policy_count"])
}

// ---------------------------------------------------------------------------
// evidence.go – more deriveTrustServiceCategory branches
// ---------------------------------------------------------------------------

func TestDeriveTrustServiceCategory_AllCategories(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	tests := []struct {
		code     string
		expected string
	}{
		{"CC1.1", "Control Environment"},
		{"CC2.1", "Communication and Information"},
		{"CC3.1", "Risk Assessment"},
		{"CC4.1", "Monitoring Activities"},
		{"CC5.1", "Control Activities"},
		{"CC6.1", "Logical and Physical Access Controls"},
		{"CC7.1", "System Operations"},
		{"CC8.1", "Change Management"},
		{"CC9.1", "Risk Mitigation"},
	}

	for _, tc := range tests {
		t.Run(tc.code, func(t *testing.T) {
			t.Parallel()
			controls := []domain.Control{
				{FrameworkCodes: []domain.FrameworkCode{{Code: tc.code}}},
			}
			result := svc.deriveTrustServiceCategory(controls)
			assert.Equal(t, tc.expected, result)
		})
	}
}

// ---------------------------------------------------------------------------
// evidence.go – buildRecommendations with different categories
// ---------------------------------------------------------------------------

func TestBuildRecommendations_Categories(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	t.Run("infrastructure category", func(t *testing.T) {
		task := &domain.EvidenceTask{
			Name:     "Firewall Config",
			Category: "Infrastructure",
		}
		recs := svc.buildRecommendations(task, nil, nil)
		found := false
		for _, r := range recs {
			if r == "Scan Terraform configurations for security controls" {
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("personnel category", func(t *testing.T) {
		task := &domain.EvidenceTask{
			Name:     "User Access",
			Category: "Personnel",
		}
		recs := svc.buildRecommendations(task, nil, nil)
		found := false
		for _, r := range recs {
			if r == "Document access controls and user permissions" {
				found = true
			}
		}
		assert.True(t, found)
	})

	t.Run("monitoring category", func(t *testing.T) {
		task := &domain.EvidenceTask{
			Name:     "Log Monitoring",
			Category: "Monitoring",
		}
		recs := svc.buildRecommendations(task, nil, nil)
		found := false
		for _, r := range recs {
			if r == "Provide evidence of continuous monitoring and alerting" {
				found = true
			}
		}
		assert.True(t, found)
	})
}

// ---------------------------------------------------------------------------
// evidence.go – extractRequiredEvidence with more guidance keywords
// ---------------------------------------------------------------------------

func TestExtractRequiredEvidence_AllKeywords(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	task := &domain.EvidenceTask{
		Guidance: "Provide screenshot of audit log configuration with policy procedure documentation and inventory list",
	}
	evidence := svc.extractRequiredEvidence(task)
	assert.Contains(t, evidence, "Screenshots of relevant configurations")
	assert.Contains(t, evidence, "Audit logs and access records")
	assert.Contains(t, evidence, "Policy and procedure documentation")
	assert.Contains(t, evidence, "Configuration exports and settings")
	assert.Contains(t, evidence, "Inventory lists and asset records")
}

// ---------------------------------------------------------------------------
// evidence.go – buildSuggestedTools with personnel category
// ---------------------------------------------------------------------------

func TestBuildSuggestedTools_PersonnelCategory(t *testing.T) {
	t.Parallel()
	stub := newStubDataService()
	cfg := &config.Config{}
	log := newTestLogger(t)
	svc, err := NewEvidenceService(stub, cfg, log)
	require.NoError(t, err)

	task := &domain.EvidenceTask{
		Name:     "User Access Review",
		Category: "Personnel",
	}
	tools := svc.buildSuggestedTools(task, nil)
	assert.Contains(t, tools, "github-permissions")
}

// ---------------------------------------------------------------------------
// doc_generator.go – GenerateAgentDocs with empty data dir
// ---------------------------------------------------------------------------

func TestGenerateAgentDocs_EmptyDataDir(t *testing.T) {
	// NOT parallel: uses os.Chdir which is process-global.

	origDir, err := os.Getwd()
	require.NoError(t, err)
	tmpDir := t.TempDir()
	require.NoError(t, os.Chdir(tmpDir))
	t.Cleanup(func() { os.Chdir(origDir) })

	cfg := &config.Config{
		Storage: config.StorageConfig{DataDir: ""},
	}

	err = GenerateAgentDocs(cfg, "test")
	require.NoError(t, err)

	// Check template was rendered with default data dir
	content, err := os.ReadFile(filepath.Join(tmpDir, ".grctool", "docs", "directory-structure.md"))
	require.NoError(t, err)
	assert.Contains(t, string(content), "./data")
}
