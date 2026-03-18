package submission

import (
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/interfaces"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/providers"
	"github.com/grctool/grctool/internal/storage"
	"github.com/grctool/grctool/internal/testhelpers"
	"github.com/grctool/grctool/internal/tugboat"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// filterPreferPDF
// ---------------------------------------------------------------------------

func TestFilterPreferPDF_NoFiles(t *testing.T) {
	t.Parallel()
	result := filterPreferPDF(nil)
	assert.Empty(t, result)
}

func TestFilterPreferPDF_NoPDFs(t *testing.T) {
	t.Parallel()
	files := []models.EvidenceFileRef{
		{Filename: "report.md", SizeBytes: 100},
		{Filename: "data.csv", SizeBytes: 200},
	}

	result := filterPreferPDF(files)
	require.Len(t, result, 2)
	assert.Equal(t, "report.md", result[0].Filename)
	assert.Equal(t, "data.csv", result[1].Filename)
}

func TestFilterPreferPDF_PDFAndMDSameBasename(t *testing.T) {
	t.Parallel()
	files := []models.EvidenceFileRef{
		{Filename: "report.md", SizeBytes: 100},
		{Filename: "report.pdf", SizeBytes: 500},
		{Filename: "data.csv", SizeBytes: 200},
	}

	result := filterPreferPDF(files)
	require.Len(t, result, 2)

	filenames := make(map[string]bool)
	for _, f := range result {
		filenames[f.Filename] = true
	}

	assert.True(t, filenames["report.pdf"], "PDF should be kept")
	assert.False(t, filenames["report.md"], "MD should be removed when PDF exists")
	assert.True(t, filenames["data.csv"], "CSV should remain")
}

func TestFilterPreferPDF_DifferentBasenames(t *testing.T) {
	t.Parallel()
	files := []models.EvidenceFileRef{
		{Filename: "report1.md", SizeBytes: 100},
		{Filename: "report2.pdf", SizeBytes: 500},
	}

	result := filterPreferPDF(files)
	require.Len(t, result, 2)
	// Both should remain since basenames differ
}

func TestFilterPreferPDF_MultiplePDFMDPairs(t *testing.T) {
	t.Parallel()
	files := []models.EvidenceFileRef{
		{Filename: "evidence_a.md", SizeBytes: 100},
		{Filename: "evidence_a.pdf", SizeBytes: 500},
		{Filename: "evidence_b.md", SizeBytes: 150},
		{Filename: "evidence_b.pdf", SizeBytes: 600},
		{Filename: "standalone.md", SizeBytes: 80},
	}

	result := filterPreferPDF(files)
	require.Len(t, result, 3)

	filenames := make(map[string]bool)
	for _, f := range result {
		filenames[f.Filename] = true
	}

	assert.True(t, filenames["evidence_a.pdf"])
	assert.True(t, filenames["evidence_b.pdf"])
	assert.True(t, filenames["standalone.md"])
	assert.False(t, filenames["evidence_a.md"])
	assert.False(t, filenames["evidence_b.md"])
}

func TestFilterPreferPDF_OnlyPDFs(t *testing.T) {
	t.Parallel()
	files := []models.EvidenceFileRef{
		{Filename: "report.pdf", SizeBytes: 500},
		{Filename: "summary.pdf", SizeBytes: 300},
	}

	result := filterPreferPDF(files)
	assert.Len(t, result, 2)
}

// ---------------------------------------------------------------------------
// getContentType
// ---------------------------------------------------------------------------

func TestGetContentType(t *testing.T) {
	t.Parallel()
	svc := &SubmissionService{}

	tests := []struct {
		filename string
		expected string
	}{
		{"report.txt", "text/plain"},
		{"data.csv", "text/csv"},
		{"config.json", "application/json"},
		{"evidence.pdf", "application/pdf"},
		{"screenshot.png", "image/png"},
		{"photo.gif", "image/gif"},
		{"photo.jpg", "image/jpeg"},
		{"photo.jpeg", "image/jpeg"},
		{"doc.md", "text/markdown"},
		{"report.doc", "application/msword"},
		{"report.docx", "application/vnd.openxmlformats-officedocument.wordprocessingml.document"},
		{"sheet.xls", "application/vnd.ms-excel"},
		{"sheet.xlsx", "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"},
		{"doc.odt", "application/vnd.oasis.opendocument.text"},
		{"sheet.ods", "application/vnd.oasis.opendocument.spreadsheet"},
		{"unknown.xyz", "application/octet-stream"},
		{"noextension", "application/octet-stream"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			t.Parallel()
			ct := svc.getContentType(tt.filename)
			assert.Equal(t, tt.expected, ct)
		})
	}
}

// ---------------------------------------------------------------------------
// buildEvidenceContent
// ---------------------------------------------------------------------------

func TestBuildEvidenceContent(t *testing.T) {
	t.Parallel()
	svc := &SubmissionService{}

	submission := &models.EvidenceSubmission{
		TaskRef: "ET-0001",
		Window:  "2025-Q4",
		Notes:   "Quarterly evidence collection",
		EvidenceFiles: []models.EvidenceFileRef{
			{Filename: "terraform.md", SizeBytes: 1024},
			{Filename: "github.pdf", SizeBytes: 2048},
		},
	}

	content := svc.buildEvidenceContent(submission)

	assert.Contains(t, content, "# Evidence Submission: ET-0001")
	assert.Contains(t, content, "**Collection Window**: 2025-Q4")
	assert.Contains(t, content, "**Files Submitted**: 2")
	assert.Contains(t, content, "terraform.md (1024 bytes)")
	assert.Contains(t, content, "github.pdf (2048 bytes)")
	assert.Contains(t, content, "## Notes")
	assert.Contains(t, content, "Quarterly evidence collection")
}

func TestBuildEvidenceContent_NoNotes(t *testing.T) {
	t.Parallel()
	svc := &SubmissionService{}

	submission := &models.EvidenceSubmission{
		TaskRef: "ET-0002",
		Window:  "2025-Q1",
		Notes:   "",
		EvidenceFiles: []models.EvidenceFileRef{
			{Filename: "data.csv", SizeBytes: 512},
		},
	}

	content := svc.buildEvidenceContent(submission)

	assert.Contains(t, content, "ET-0002")
	assert.NotContains(t, content, "## Notes")
}

func TestBuildEvidenceContent_EmptyFiles(t *testing.T) {
	t.Parallel()
	svc := &SubmissionService{}

	submission := &models.EvidenceSubmission{
		TaskRef:       "ET-0003",
		Window:        "2025-Q2",
		EvidenceFiles: []models.EvidenceFileRef{},
	}

	content := svc.buildEvidenceContent(submission)
	assert.Contains(t, content, "**Files Submitted**: 0")
}

// ---------------------------------------------------------------------------
// buildEvidenceSources
// ---------------------------------------------------------------------------

func TestBuildEvidenceSources(t *testing.T) {
	t.Parallel()
	svc := &SubmissionService{}

	submission := &models.EvidenceSubmission{
		EvidenceFiles: []models.EvidenceFileRef{
			{Filename: "terraform.md", Source: "terraform-scanner"},
			{Filename: "github.pdf", Source: "github-permissions"},
			{Filename: "manual.md", Source: ""},
		},
	}

	sources := svc.buildEvidenceSources(submission)

	// Only files with non-empty Source should be included
	require.Len(t, sources, 2)
	assert.Equal(t, "tool", sources[0].Type)
	assert.Equal(t, "terraform-scanner", sources[0].Tool)
	assert.Equal(t, "github-permissions", sources[1].Tool)
	assert.NotEmpty(t, sources[0].Timestamp)
}

func TestBuildEvidenceSources_NoSources(t *testing.T) {
	t.Parallel()
	svc := &SubmissionService{}

	submission := &models.EvidenceSubmission{
		EvidenceFiles: []models.EvidenceFileRef{
			{Filename: "manual.md", Source: ""},
		},
	}

	sources := svc.buildEvidenceSources(submission)
	assert.Empty(t, sources)
}

// ---------------------------------------------------------------------------
// extractControlsCovered
// ---------------------------------------------------------------------------

func TestExtractControlsCovered(t *testing.T) {
	t.Parallel()
	svc := &SubmissionService{}

	submission := &models.EvidenceSubmission{
		EvidenceFiles: []models.EvidenceFileRef{
			{
				Filename:          "terraform.md",
				ControlsSatisfied: []string{"CC6.1", "CC6.3"},
			},
			{
				Filename:          "github.pdf",
				ControlsSatisfied: []string{"CC6.3", "CC7.1"},
			},
		},
	}

	controls := svc.extractControlsCovered(submission)

	// Should deduplicate
	assert.Len(t, controls, 3)

	controlSet := make(map[string]bool)
	for _, c := range controls {
		controlSet[c] = true
	}
	assert.True(t, controlSet["CC6.1"])
	assert.True(t, controlSet["CC6.3"])
	assert.True(t, controlSet["CC7.1"])
}

func TestExtractControlsCovered_Empty(t *testing.T) {
	t.Parallel()
	svc := &SubmissionService{}

	submission := &models.EvidenceSubmission{
		EvidenceFiles: []models.EvidenceFileRef{
			{Filename: "manual.md"},
		},
	}

	controls := svc.extractControlsCovered(submission)
	assert.Empty(t, controls)
}

// ---------------------------------------------------------------------------
// NewSubmissionService
// ---------------------------------------------------------------------------

func TestNewSubmissionService(t *testing.T) {
	t.Parallel()

	collectorURLs := map[string]string{
		"ET-0001": "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/",
	}

	svc := NewSubmissionService(nil, nil, "test-org", collectorURLs)
	require.NotNil(t, svc)
	assert.Equal(t, "test-org", svc.orgID)
	assert.Equal(t, collectorURLs, svc.collectorURLs)
}

func TestNewSubmissionServiceWithRegistry_Success(t *testing.T) {
	t.Parallel()

	reg := newStubRegistry(t, "test-submitter", true)
	collectorURLs := map[string]string{"ET-0001": "https://example.com/collector/1"}

	svc, err := NewSubmissionServiceWithRegistry(nil, reg, "test-submitter", collectorURLs)
	require.NoError(t, err)
	require.NotNil(t, svc)
	assert.NotNil(t, svc.submitter, "submitter should be resolved from registry")
	assert.Nil(t, svc.tugboatClient, "legacy client should be nil")
}

func TestNewSubmissionServiceWithRegistry_ProviderNotFound(t *testing.T) {
	t.Parallel()

	reg := newStubRegistry(t, "other", true)
	_, err := NewSubmissionServiceWithRegistry(nil, reg, "missing", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not registered")
}

func TestNewSubmissionServiceWithRegistry_NotEvidenceSubmitter(t *testing.T) {
	t.Parallel()

	reg := newStubRegistry(t, "read-only", false)
	_, err := NewSubmissionServiceWithRegistry(nil, reg, "read-only", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not support evidence submission")
}

// ---------------------------------------------------------------------------
// SubmitRequest / SubmitResponse types
// ---------------------------------------------------------------------------

func TestSubmitRequest_Fields(t *testing.T) {
	t.Parallel()
	req := SubmitRequest{
		TaskRef:        "ET-0001",
		Window:         "2025-Q4",
		Notes:          "test notes",
		SkipValidation: true,
		SubmittedBy:    "user@example.com",
	}
	assert.Equal(t, "ET-0001", req.TaskRef)
	assert.True(t, req.SkipValidation)
}

func TestSubmit_MissingCollectorURL(t *testing.T) {
	t.Parallel()
	st, tmpDir := setupTestStorage(t)

	// Create evidence files
	evidenceDir := filepath.Join(tmpDir, "evidence", "ET-0047", "2025-Q4")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(evidenceDir, "evidence.md"),
		[]byte("# Evidence"), 0644))

	// Create a tugboat client config
	tugboatCfg := &config.TugboatConfig{
		BaseURL:  "https://api-my.tugboatlogic.com",
		Timeout:  5000000000, // 5s
		Username: "test",
		Password: "test",
	}
	tugboatClient := tugboat.NewClient(tugboatCfg, nil)

	// Create service with empty collector URLs (will cause submitToTugboat to fail)
	svc := NewSubmissionService(st, tugboatClient, "42", map[string]string{})

	_, err := svc.Submit(context.Background(), &SubmitRequest{
		TaskRef:        "ET-0047",
		Window:         "2025-Q4",
		SkipValidation: true,
	})
	// Should fail because collector URL is not configured
	require.Error(t, err)
	assert.Contains(t, err.Error(), "collector URL not configured")
}

func TestSubmitResponse_Fields(t *testing.T) {
	t.Parallel()
	resp := SubmitResponse{
		Success:      true,
		SubmissionID: "sub-123",
		Status:       "submitted",
		Message:      "Evidence submitted successfully",
	}
	assert.True(t, resp.Success)
	assert.Equal(t, "sub-123", resp.SubmissionID)
}

// ---------------------------------------------------------------------------
// Integration tests with real storage
// ---------------------------------------------------------------------------

// setupTestStorage creates a Storage backed by t.TempDir() and populates it
// with a minimal evidence task fixture.
func setupTestStorage(t *testing.T) (*storage.Storage, string) {
	t.Helper()
	tmpDir := t.TempDir()

	cfg := config.StorageConfig{
		DataDir: tmpDir,
	}

	st, err := storage.NewStorage(cfg)
	require.NoError(t, err)

	// Create a minimal evidence task JSON file
	taskDir := filepath.Join(tmpDir, "docs", "evidence_tasks", "json")
	require.NoError(t, os.MkdirAll(taskDir, 0755))

	task := map[string]interface{}{
		"id":                  327992,
		"name":                "GitHub Repository Access Controls",
		"description":         "Provide evidence of repository access controls.",
		"collection_interval": "quarter",
		"reference_id":        "ET-0047",
	}
	taskData, _ := json.Marshal(task)
	require.NoError(t, os.WriteFile(
		filepath.Join(taskDir, "ET-0047-327992-github_repository_access_controls.json"),
		taskData, 0644))

	return st, tmpDir
}

func TestSubmit_NoTugboatClient_Draft(t *testing.T) {
	t.Parallel()
	st, tmpDir := setupTestStorage(t)

	// Create evidence files for the task/window
	evidenceDir := filepath.Join(tmpDir, "evidence", "ET-0047", "2025-Q4")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(evidenceDir, "github_access.md"),
		[]byte("# Evidence\n\nAccess controls documented."), 0644))

	collectorURLs := map[string]string{
		"ET-0047": "https://openapi.tugboatlogic.com/api/v0/evidence/collector/805/",
	}

	svc := NewSubmissionService(st, nil, "42", collectorURLs)

	resp, err := svc.Submit(context.Background(), &SubmitRequest{
		TaskRef:        "ET-0047",
		Window:         "2025-Q4",
		Notes:          "Q4 evidence",
		SkipValidation: true,
		SubmittedBy:    "test@example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, "draft", resp.Status) // No tugboat client = draft
	assert.NotEmpty(t, resp.SubmissionID)
	assert.Contains(t, resp.SubmissionID, "local-")
}

func TestSubmit_TaskNotFound(t *testing.T) {
	t.Parallel()
	st, _ := setupTestStorage(t)

	svc := NewSubmissionService(st, nil, "42", nil)

	_, err := svc.Submit(context.Background(), &SubmitRequest{
		TaskRef:        "ET-9999",
		Window:         "2025-Q4",
		SkipValidation: true,
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to get evidence task")
}

func TestGetSubmissionStatus_NotFound(t *testing.T) {
	t.Parallel()
	st, _ := setupTestStorage(t)

	svc := NewSubmissionService(st, nil, "42", nil)

	_, err := svc.GetSubmissionStatus(context.Background(), "ET-0047", "2025-Q4")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "submission not found")
}

func TestGetSubmissionHistory_NotFound(t *testing.T) {
	t.Parallel()
	st, _ := setupTestStorage(t)

	svc := NewSubmissionService(st, nil, "42", nil)

	_, err := svc.GetSubmissionHistory("ET-0047", "2025-Q4")
	require.Error(t, err)
}

func TestSubmit_ThenGetStatus(t *testing.T) {
	t.Parallel()
	st, tmpDir := setupTestStorage(t)

	// Create evidence files
	evidenceDir := filepath.Join(tmpDir, "evidence", "ET-0047", "2025-Q4")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(evidenceDir, "evidence.md"),
		[]byte("# Evidence"), 0644))

	svc := NewSubmissionService(st, nil, "42", map[string]string{})

	// Submit
	resp, err := svc.Submit(context.Background(), &SubmitRequest{
		TaskRef:        "ET-0047",
		Window:         "2025-Q4",
		SkipValidation: true,
		SubmittedBy:    "test@example.com",
	})
	require.NoError(t, err)
	require.True(t, resp.Success)

	// Get status
	submission, err := svc.GetSubmissionStatus(context.Background(), "ET-0047", "2025-Q4")
	require.NoError(t, err)
	assert.Equal(t, "draft", submission.Status)
}

func TestSubmit_WithValidation(t *testing.T) {
	t.Parallel()
	st, tmpDir := setupTestStorage(t)

	// Create evidence files
	evidenceDir := filepath.Join(tmpDir, "evidence", "ET-0047", "2025-Q4")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(evidenceDir, "evidence.md"),
		[]byte("# Evidence\n\nSome content."), 0644))

	svc := NewSubmissionService(st, nil, "42", map[string]string{})

	// Submit with validation enabled - validation may fail since we don't have
	// all the required evidence, but we test the validation code path
	resp, err := svc.Submit(context.Background(), &SubmitRequest{
		TaskRef:        "ET-0047",
		Window:         "2025-Q4",
		SkipValidation: false, // Run validation
		SubmittedBy:    "test@example.com",
	})
	// Either succeeds or returns validation failure response (not error)
	if err != nil {
		// Validation service may error if configuration isn't complete
		assert.Contains(t, err.Error(), "validation")
	} else {
		require.NotNil(t, resp)
		// Response may be success or validation failure
		if !resp.Success {
			assert.Equal(t, "validation_failed", resp.Status)
			assert.NotNil(t, resp.ValidationResult)
		}
	}
}

func TestSubmit_WithPDFAndMDFiles(t *testing.T) {
	t.Parallel()
	st, tmpDir := setupTestStorage(t)

	// Create evidence files with both PDF and MD (PDF should be preferred)
	evidenceDir := filepath.Join(tmpDir, "evidence", "ET-0047", "2025-Q4")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(evidenceDir, "report.md"),
		[]byte("# Report in markdown"), 0644))
	require.NoError(t, os.WriteFile(
		filepath.Join(evidenceDir, "report.pdf"),
		[]byte("%PDF-1.4 fake pdf content"), 0644))
	require.NoError(t, os.WriteFile(
		filepath.Join(evidenceDir, "extra.csv"),
		[]byte("col1,col2\nval1,val2"), 0644))

	svc := NewSubmissionService(st, nil, "42", map[string]string{})

	resp, err := svc.Submit(context.Background(), &SubmitRequest{
		TaskRef:        "ET-0047",
		Window:         "2025-Q4",
		SkipValidation: true,
		SubmittedBy:    "test@example.com",
	})
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)

	// The submission should have filtered out report.md in favor of report.pdf
	if resp.Submission != nil {
		filenames := make(map[string]bool)
		for _, f := range resp.Submission.EvidenceFiles {
			filenames[f.Filename] = true
		}
		// report.md should have been filtered out
		assert.True(t, filenames["report.pdf"], "PDF should be included")
		assert.True(t, filenames["extra.csv"], "CSV should be included")
		assert.False(t, filenames["report.md"], "MD should be filtered when PDF exists")
	}
}

func TestSubmit_NoEvidenceFiles(t *testing.T) {
	t.Parallel()
	st, tmpDir := setupTestStorage(t)

	// Create empty evidence directory
	evidenceDir := filepath.Join(tmpDir, "evidence", "ET-0047", "2025-Q4")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))

	svc := NewSubmissionService(st, nil, "42", map[string]string{})

	resp, err := svc.Submit(context.Background(), &SubmitRequest{
		TaskRef:        "ET-0047",
		Window:         "2025-Q4",
		SkipValidation: true,
	})
	// Should succeed with 0 files in draft mode
	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestSubmit_ThenGetHistory(t *testing.T) {
	t.Parallel()
	st, tmpDir := setupTestStorage(t)

	// Create evidence files
	evidenceDir := filepath.Join(tmpDir, "evidence", "ET-0047", "2025-Q4")
	require.NoError(t, os.MkdirAll(evidenceDir, 0755))
	require.NoError(t, os.WriteFile(
		filepath.Join(evidenceDir, "evidence.csv"),
		[]byte("col1,col2\nval1,val2"), 0644))

	svc := NewSubmissionService(st, nil, "42", map[string]string{})

	// Submit
	_, err := svc.Submit(context.Background(), &SubmitRequest{
		TaskRef:        "ET-0047",
		Window:         "2025-Q4",
		SkipValidation: true,
		SubmittedBy:    "test@example.com",
		Notes:          "Test submission",
	})
	require.NoError(t, err)

	// Get history
	history, err := svc.GetSubmissionHistory("ET-0047", "2025-Q4")
	require.NoError(t, err)
	require.NotNil(t, history)
	assert.GreaterOrEqual(t, len(history.Entries), 1)
}

// ---------------------------------------------------------------------------
// Registry-based submission helpers
// ---------------------------------------------------------------------------

// stubSubmitterProvider wraps StubDataProvider with EvidenceSubmitter support.
type stubSubmitterProvider struct {
	*testhelpers.StubDataProvider
	submissions []interfaces.SubmissionMetadata
	submitErr   error
}

var _ interfaces.EvidenceSubmitter = (*stubSubmitterProvider)(nil)

func (s *stubSubmitterProvider) SubmitEvidence(_ context.Context, taskID string, file io.Reader, meta interfaces.SubmissionMetadata) error {
	if s.submitErr != nil {
		return s.submitErr
	}
	s.submissions = append(s.submissions, meta)
	// Drain the reader
	_, _ = io.ReadAll(file)
	return nil
}

func (s *stubSubmitterProvider) ListAttachments(_ context.Context, _ string, _ interfaces.ListOptions) ([]interfaces.Attachment, int, error) {
	return nil, 0, nil
}

func (s *stubSubmitterProvider) DownloadAttachment(_ context.Context, _ string) (io.ReadCloser, string, error) {
	return nil, "", nil
}

// newStubRegistry creates a registry with a single provider.
// If withSubmitter is true, the provider implements EvidenceSubmitter.
func newStubRegistry(t *testing.T, name string, withSubmitter bool) interfaces.ProviderRegistry {
	t.Helper()
	reg := providers.NewProviderRegistry()
	if withSubmitter {
		p := &stubSubmitterProvider{StubDataProvider: testhelpers.NewStubDataProvider(name)}
		require.NoError(t, reg.Register(p))
	} else {
		require.NoError(t, reg.Register(testhelpers.NewStubDataProvider(name)))
	}
	return reg
}
