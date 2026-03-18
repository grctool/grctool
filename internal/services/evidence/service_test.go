package evidence

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/grctool/grctool/internal/config"
	"github.com/grctool/grctool/internal/domain"
	"github.com/grctool/grctool/internal/logger"
	"github.com/grctool/grctool/internal/models"
	"github.com/grctool/grctool/internal/services"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubDataService implements services.DataService for testing
type stubDataService struct {
	tasks    []domain.EvidenceTask
	controls []domain.Control
	policies []domain.Policy
}

func (s *stubDataService) GetEvidenceTask(_ context.Context, taskID string) (*domain.EvidenceTask, error) {
	for _, t := range s.tasks {
		if t.ID == taskID {
			return &t, nil
		}
	}
	return nil, fmt.Errorf("task not found: %s", taskID)
}

func (s *stubDataService) GetAllEvidenceTasks(_ context.Context) ([]domain.EvidenceTask, error) {
	return s.tasks, nil
}

func (s *stubDataService) FilterEvidenceTasks(_ context.Context, _ domain.EvidenceFilter) ([]domain.EvidenceTask, error) {
	return s.tasks, nil
}

func (s *stubDataService) GetControl(_ context.Context, controlID string) (*domain.Control, error) {
	for _, c := range s.controls {
		if fmt.Sprintf("%s", c.ID) == controlID {
			return &c, nil
		}
	}
	return nil, fmt.Errorf("control not found: %s", controlID)
}

func (s *stubDataService) GetAllControls(_ context.Context) ([]domain.Control, error) {
	return s.controls, nil
}

func (s *stubDataService) GetPolicy(_ context.Context, policyID string) (*domain.Policy, error) {
	for _, p := range s.policies {
		if p.ID == policyID {
			return &p, nil
		}
	}
	return nil, fmt.Errorf("policy not found: %s", policyID)
}

func (s *stubDataService) GetAllPolicies(_ context.Context) ([]domain.Policy, error) {
	return s.policies, nil
}

func (s *stubDataService) GetRelationships(_ context.Context, _, _ string) ([]domain.Relationship, error) {
	return nil, nil
}

func (s *stubDataService) SaveEvidenceRecord(_ context.Context, _ *domain.EvidenceRecord) error {
	return nil
}

func (s *stubDataService) GetEvidenceRecords(_ context.Context, _ string) ([]domain.EvidenceRecord, error) {
	return nil, nil
}

// compile-time check
var _ services.DataService = (*stubDataService)(nil)

// newTestService creates a ServiceImpl suitable for testing methods that
// delegate to evidenceService. Uses t.TempDir() for storage paths.
func newTestService(t *testing.T, ds *stubDataService) *ServiceImpl {
	t.Helper()
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Tugboat: config.TugboatConfig{
			BaseURL: "https://api-my.tugboatlogic.com",
			OrgID:   "42",
		},
		Storage: config.StorageConfig{
			DataDir: tmpDir,
		},
	}

	log := &silentLogger{}

	evidenceSvc, err := services.NewEvidenceService(ds, cfg, log)
	require.NoError(t, err)

	docSvc := services.NewDocumentService(cfg)

	return &ServiceImpl{
		dataService:     ds,
		evidenceService: evidenceSvc,
		documentService: docSvc,
		config:          cfg,
		logger:          log,
	}
}

// silentLogger implements logger.Logger as no-op
type silentLogger struct{}

func (l *silentLogger) Trace(_ string, _ ...logger.Field) {}
func (l *silentLogger) Debug(_ string, _ ...logger.Field) {}
func (l *silentLogger) Info(_ string, _ ...logger.Field)  {}
func (l *silentLogger) Warn(_ string, _ ...logger.Field)  {}
func (l *silentLogger) Error(_ string, _ ...logger.Field) {}

func (l *silentLogger) WithFields(_ ...logger.Field) logger.Logger  { return l }
func (l *silentLogger) WithContext(_ context.Context) logger.Logger  { return l }
func (l *silentLogger) WithComponent(_ string) logger.Logger         { return l }
func (l *silentLogger) TraceOperation(_ string) logger.Tracer        { return &silentTracer{} }
func (l *silentLogger) RequestLogger(_ string) logger.Logger         { return l }
func (l *silentLogger) DumpJSON(_ interface{}, _ string)             {}
func (l *silentLogger) Timing(_ string) logger.TimingLogger          { return &silentTimingLogger{} }

type silentTracer struct{ start time.Time }

func (t *silentTracer) Step(_ string, _ ...logger.Field) {}
func (t *silentTracer) Success(_ ...logger.Field)        {}
func (t *silentTracer) Error(_ error, _ ...logger.Field) {}
func (t *silentTracer) Duration() time.Duration          { return time.Since(t.start) }

type silentTimingLogger struct{}

func (t *silentTimingLogger) Mark(_ string, _ ...logger.Field) {}
func (t *silentTimingLogger) Complete(_ ...logger.Field)       {}
func (t *silentTimingLogger) Abandon(_ string)                 {}

// ---------------------------------------------------------------------------
// ListEvidenceTasks (via full service)
// ---------------------------------------------------------------------------

func TestListEvidenceTasks(t *testing.T) {
	t.Parallel()
	ds := &stubDataService{
		tasks: []domain.EvidenceTask{
			{ID: "1", ReferenceID: "ET0001", Name: "Task A", Framework: "SOC2", Status: "pending"},
			{ID: "2", ReferenceID: "ET0002", Name: "Task B", Framework: "SOC2", Status: "completed"},
		},
	}
	svc := newTestService(t, ds)

	tasks, err := svc.ListEvidenceTasks(context.Background(), domain.EvidenceFilter{})
	require.NoError(t, err)
	assert.Len(t, tasks, 2)
}

func TestListEvidenceTasks_Empty(t *testing.T) {
	t.Parallel()
	ds := &stubDataService{}
	svc := newTestService(t, ds)

	tasks, err := svc.ListEvidenceTasks(context.Background(), domain.EvidenceFilter{})
	require.NoError(t, err)
	assert.Empty(t, tasks)
}

// ---------------------------------------------------------------------------
// GetEvidenceTaskSummary (via full service)
// ---------------------------------------------------------------------------

func TestGetEvidenceTaskSummary(t *testing.T) {
	t.Parallel()
	ds := &stubDataService{
		tasks: []domain.EvidenceTask{
			{ID: "1", Status: "pending", Priority: "high"},
			{ID: "2", Status: "completed", Priority: "medium"},
		},
	}
	svc := newTestService(t, ds)

	summary, err := svc.GetEvidenceTaskSummary(context.Background())
	require.NoError(t, err)
	require.NotNil(t, summary)
}

// ---------------------------------------------------------------------------
// MapEvidenceRelationships (via full service)
// ---------------------------------------------------------------------------

func TestMapEvidenceRelationships_Empty(t *testing.T) {
	t.Parallel()
	ds := &stubDataService{}
	svc := newTestService(t, ds)

	result, err := svc.MapEvidenceRelationships(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Empty(t, result.Tasks)
	assert.NotNil(t, result.Summary)
}

func TestMapEvidenceRelationships_WithData(t *testing.T) {
	t.Parallel()
	ds := &stubDataService{
		tasks: []domain.EvidenceTask{
			{ID: "1", Framework: "SOC2", Status: "pending", Priority: "high"},
			{ID: "2", Framework: "ISO27001", Status: "completed", Priority: "medium"},
		},
		controls: []domain.Control{
			{ID: "1001", Name: "Control A"},
		},
		policies: []domain.Policy{
			{ID: "1", Name: "Policy A"},
		},
	}
	svc := newTestService(t, ds)

	result, err := svc.MapEvidenceRelationships(context.Background())
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Len(t, result.Tasks, 2)
	assert.Len(t, result.Controls, 1)
	assert.Len(t, result.Policies, 1)
	assert.Equal(t, 2, len(result.FrameworkGroups))
	assert.NotNil(t, result.Summary)
	assert.Equal(t, 2, result.Summary.TotalTasks)
}

// ---------------------------------------------------------------------------
// GenerateTemplateBasedPrompt
// ---------------------------------------------------------------------------

func sampleEvidenceContext() *models.EvidenceContext {
	return &models.EvidenceContext{
		Task: models.EvidenceTaskDetails{
			EvidenceTask: models.EvidenceTask{
	ID: "327992",
				ReferenceID:        "ET-0047",
				Name:               "GitHub Repository Access Controls",
				Description:        "Provide evidence of repository access controls.",
				Guidance:           "Extract team membership and permissions from GitHub.",
				CollectionInterval: "quarter",
				Priority:           "high",
				Framework:          "SOC2",
				Status:             "pending",
			},
		},
		Controls: []models.Control{
			{
	ID: "1001",
				Name:     "Logical Access Security",
				Category: "Common Criteria",
				Status:   "implemented",
				Body:     "The entity implements logical access security.",
				FrameworkCodes: []models.FrameworkCode{
					{Code: "CC6.1", FrameworkName: "SOC2"},
				},
			},
		},
		Policies: []models.Policy{
			{
				ID:          "12345",
				Name:        "Access Control Policy",
				Description: "Controls access to information assets.",
				Status:      "active",
				Framework:   "SOC2",
			},
		},
		FrameworkReqs: []string{"CC6.1", "CC6.3"},
		AvailableTools: []models.ToolInfo{
			{Name: "github-permissions", Enabled: true},
			{Name: "terraform-security-analyzer", Enabled: true},
		},
		ControlSummaries: make(map[string]models.AIControlSummary),
		PolicySummaries:  make(map[string]models.AIPolicySummary),
	}
}

func TestGenerateTemplateBasedPrompt_Markdown(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}
	ctx := sampleEvidenceContext()

	prompt := svc.GenerateTemplateBasedPrompt(ctx, "markdown")

	// Header section
	assert.Contains(t, prompt, "# Evidence Collection Task: GitHub Repository Access Controls")
	assert.Contains(t, prompt, "**Task ID:** ET-0047")
	assert.Contains(t, prompt, "**Framework:** SOC2")
	assert.Contains(t, prompt, "**Priority:** high")
	assert.Contains(t, prompt, "**Status:** pending")
	assert.Contains(t, prompt, "**Output Format:** markdown")

	// Task description
	assert.Contains(t, prompt, "## Task Description")
	assert.Contains(t, prompt, "Provide evidence of repository access controls.")

	// Guidance
	assert.Contains(t, prompt, "## Collection Guidance")
	assert.Contains(t, prompt, "Extract team membership")

	// Framework requirements
	assert.Contains(t, prompt, "## Framework Requirements")
	assert.Contains(t, prompt, "CC6.1")

	// Controls
	assert.Contains(t, prompt, "## Related Controls")
	assert.Contains(t, prompt, "Logical Access Security")
	assert.Contains(t, prompt, "**Category:** Common Criteria")

	// Policies
	assert.Contains(t, prompt, "## Related Policies")
	assert.Contains(t, prompt, "Access Control Policy")

	// Tools
	assert.Contains(t, prompt, "## Suggested Evidence Collection Tools")
	assert.Contains(t, prompt, "github-permissions")
	assert.Contains(t, prompt, "terraform-security-analyzer")

	// Markdown format instructions
	assert.Contains(t, prompt, "### Markdown Format Requirements")
	assert.Contains(t, prompt, "Executive Summary")
}

func TestGenerateTemplateBasedPrompt_CSV(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}
	ctx := sampleEvidenceContext()

	prompt := svc.GenerateTemplateBasedPrompt(ctx, "csv")

	assert.Contains(t, prompt, "### CSV Format Requirements")
	assert.Contains(t, prompt, "Control/Requirement")
	assert.Contains(t, prompt, "Verification Method")
}

func TestGenerateTemplateBasedPrompt_JSON(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}
	ctx := sampleEvidenceContext()

	prompt := svc.GenerateTemplateBasedPrompt(ctx, "json")

	assert.Contains(t, prompt, "### JSON Format Requirements")
	assert.Contains(t, prompt, "JSON object")
}

func TestGenerateTemplateBasedPrompt_NoGuidance(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}
	ctx := sampleEvidenceContext()
	ctx.Task.Guidance = ""

	prompt := svc.GenerateTemplateBasedPrompt(ctx, "markdown")
	assert.NotContains(t, prompt, "## Collection Guidance")
}

func TestGenerateTemplateBasedPrompt_NoFrameworkReqs(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}
	ctx := sampleEvidenceContext()
	ctx.FrameworkReqs = nil

	prompt := svc.GenerateTemplateBasedPrompt(ctx, "markdown")
	assert.NotContains(t, prompt, "## Framework Requirements")
}

func TestGenerateTemplateBasedPrompt_NoControls(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}
	ctx := sampleEvidenceContext()
	ctx.Controls = nil

	prompt := svc.GenerateTemplateBasedPrompt(ctx, "markdown")
	assert.NotContains(t, prompt, "## Related Controls")
}

func TestGenerateTemplateBasedPrompt_NoPolicies(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}
	ctx := sampleEvidenceContext()
	ctx.Policies = nil

	prompt := svc.GenerateTemplateBasedPrompt(ctx, "markdown")
	assert.NotContains(t, prompt, "## Related Policies")
}

func TestGenerateTemplateBasedPrompt_NoTools(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}
	ctx := sampleEvidenceContext()
	ctx.AvailableTools = nil

	prompt := svc.GenerateTemplateBasedPrompt(ctx, "markdown")
	assert.NotContains(t, prompt, "## Suggested Evidence Collection Tools")
}

func TestGenerateTemplateBasedPrompt_ToolExamples(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}
	ctx := sampleEvidenceContext()
	// Add specific tools that have example blocks
	ctx.AvailableTools = []models.ToolInfo{
		{Name: "control-summary-generator", Enabled: true},
		{Name: "policy-summary-generator", Enabled: true},
		{Name: "terraform-security-analyzer", Enabled: true},
		{Name: "github-permissions", Enabled: true},
	}

	prompt := svc.GenerateTemplateBasedPrompt(ctx, "markdown")

	assert.Contains(t, prompt, "control-summary-generator")
	assert.Contains(t, prompt, "policy-summary-generator")
	assert.Contains(t, prompt, "terraform-security-analyzer")
	assert.Contains(t, prompt, "github-permissions")
	assert.Contains(t, prompt, "### Tool Usage Examples")
}

func TestGenerateTemplateBasedPrompt_AlwaysHasGuidelines(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}
	ctx := sampleEvidenceContext()

	prompt := svc.GenerateTemplateBasedPrompt(ctx, "markdown")

	assert.Contains(t, prompt, "## Evidence Collection Guidelines")
	assert.Contains(t, prompt, "Include specific configuration details")
	assert.Contains(t, prompt, "template-based assembly")
}

// ---------------------------------------------------------------------------
// SaveAnalysisToFile
// ---------------------------------------------------------------------------

func TestSaveAnalysisToFile(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "analysis")
	content := "# Analysis\n\nSome evidence content."

	err := svc.SaveAnalysisToFile(filename, content)
	require.NoError(t, err)

	// Should have .md extension added
	data, err := os.ReadFile(filename + ".md")
	require.NoError(t, err)
	assert.Contains(t, string(data), "Analysis")
}

func TestSaveAnalysisToFile_WithExtension(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "analysis.md")
	content := "# Test Content"

	err := svc.SaveAnalysisToFile(filename, content)
	require.NoError(t, err)

	data, err := os.ReadFile(filename)
	require.NoError(t, err)
	assert.Contains(t, string(data), "Test Content")
}

func TestSaveAnalysisToFile_CreatesDirectory(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "subdir", "analysis.md")
	content := "# Nested"

	err := svc.SaveAnalysisToFile(filename, content)
	require.NoError(t, err)

	_, err = os.Stat(filename)
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// SaveEvidenceToFile
// ---------------------------------------------------------------------------

func TestSaveEvidenceToFile_CSV(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	tmpDir := t.TempDir()
	record := &domain.EvidenceRecord{
		ID:      "rec-001",
		Content: "col1,col2\nval1,val2",
		Format:  "csv",
	}

	err := svc.SaveEvidenceToFile(tmpDir, record)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmpDir, "evidence_rec-001.csv"))
	require.NoError(t, err)
	assert.Equal(t, "col1,col2\nval1,val2", string(data))
}

func TestSaveEvidenceToFile_Markdown(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	tmpDir := t.TempDir()
	record := &domain.EvidenceRecord{
		ID:      "rec-002",
		Content: "# Evidence\n\nDetails here.",
		Format:  "markdown",
	}

	err := svc.SaveEvidenceToFile(tmpDir, record)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmpDir, "evidence_rec-002.md"))
	require.NoError(t, err)
	assert.Contains(t, string(data), "Evidence")
}

func TestSaveEvidenceToFile_OtherFormat(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	tmpDir := t.TempDir()
	record := &domain.EvidenceRecord{
		ID:      "rec-003",
		Content: "raw content",
		Format:  "txt",
	}

	err := svc.SaveEvidenceToFile(tmpDir, record)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(tmpDir, "evidence_rec-003.txt"))
	require.NoError(t, err)
	assert.Equal(t, "raw content", string(data))
}

func TestSaveEvidenceToFile_CreatesOutputDir(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	tmpDir := t.TempDir()
	outputDir := filepath.Join(tmpDir, "new", "output")
	record := &domain.EvidenceRecord{
		ID:      "rec-004",
		Content: "test",
		Format:  "csv",
	}

	err := svc.SaveEvidenceToFile(outputDir, record)
	require.NoError(t, err)

	_, err = os.Stat(filepath.Join(outputDir, "evidence_rec-004.csv"))
	assert.NoError(t, err)
}

// ---------------------------------------------------------------------------
// ResolveTaskID
// ---------------------------------------------------------------------------

func TestResolveTaskID_NumericID(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{
		dataService: &stubDataService{},
	}

	id, err := svc.ResolveTaskID(context.Background(), "327992")
	require.NoError(t, err)
	assert.Equal(t, "327992", id)
}

func TestResolveTaskID_ReferenceID(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{
		dataService: &stubDataService{
			tasks: []domain.EvidenceTask{
				{ID: "100", ReferenceID: "ET0001"},
				{ID: "200", ReferenceID: "ET0002"},
			},
		},
	}

	id, err := svc.ResolveTaskID(context.Background(), "ET0002")
	require.NoError(t, err)
	assert.Equal(t, "200", id)
}

func TestResolveTaskID_ReferenceID_NotFound(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{
		dataService: &stubDataService{
			tasks: []domain.EvidenceTask{
				{ID: "100", ReferenceID: "ET0001"},
			},
		},
	}

	_, err := svc.ResolveTaskID(context.Background(), "ET9999")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not found")
}

func TestResolveTaskID_InvalidFormat(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{
		dataService: &stubDataService{},
	}

	_, err := svc.ResolveTaskID(context.Background(), "invalid-id")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "invalid task identifier")
}

func TestResolveTaskID_LowerCaseET(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{
		dataService: &stubDataService{
			tasks: []domain.EvidenceTask{
				{ID: "100", ReferenceID: "ET0001"},
			},
		},
	}

	id, err := svc.ResolveTaskID(context.Background(), "et0001")
	require.NoError(t, err)
	assert.Equal(t, "100", id)
}

// ---------------------------------------------------------------------------
// convertToModelsTask / convertToModelsControls / convertToModelsPolicies
// ---------------------------------------------------------------------------

func TestConvertToModelsTask(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}
	now := time.Now()

	task := &domain.EvidenceTask{
	ID: "327992",
		ReferenceID:        "ET-0047",
		Name:               "GitHub Repository Access Controls",
		Description:        "Provide evidence.",
		Guidance:           "Extract data.",
		CollectionInterval: "quarter",
		Priority:           "high",
		Status:             "pending",
		Framework:          "SOC2",
		LastCollected:      &now,
	}

	result := svc.convertToModelsTask(task)
	assert.Equal(t, "327992", result.ID)
	assert.Equal(t, "ET-0047", result.ReferenceID)
	assert.Equal(t, "GitHub Repository Access Controls", result.Name)
	assert.NotNil(t, result.LastCollected)
}

func TestConvertToModelsTask_NilLastCollected(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	task := &domain.EvidenceTask{
	ID: "1",
		LastCollected: nil,
	}

	result := svc.convertToModelsTask(task)
	assert.Nil(t, result.LastCollected)
}

func TestConvertToModelsControls(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	controls := []domain.Control{
		{
	ID: "1001",
			Name:        "Logical Access",
			Description: "Access security.",
			Category:    "Common Criteria",
			Status:      "implemented",
			FrameworkCodes: []domain.FrameworkCode{
				{Code: "CC6.1", Framework: "SOC2"},
			},
		},
	}

	result := svc.convertToModelsControls(controls)
	require.Len(t, result, 1)
	assert.Equal(t, "1001", result[0].ID)
	assert.Equal(t, "Access security.", result[0].Body)
	assert.Len(t, result[0].FrameworkCodes, 1)
}

func TestConvertToModelsControls_WithOrgScope(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	controls := []domain.Control{
		{
	ID: "1001",
			Name: "Test Control",
			OrgScope: &domain.OrgScope{
	ID: 42,
				Name:        "Production",
				Type:        "environment",
				Description: "Prod scope",
			},
		},
	}

	result := svc.convertToModelsControls(controls)
	require.Len(t, result, 1)
	require.NotNil(t, result[0].OrgScope)
	assert.Equal(t, 42, result[0].OrgScope.ID)
	assert.Equal(t, "Production", result[0].OrgScope.Name)
}

func TestConvertToModelsPolicies(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	policies := []domain.Policy{
		{
			ID:          "12345",
			Name:        "Access Control Policy",
			Description: "Controls access.",
			Status:      "active",
			Framework:   "SOC2",
		},
	}

	result := svc.convertToModelsPolicies(policies)
	require.Len(t, result, 1)
	assert.Equal(t, "Access Control Policy", result[0].Name)
	assert.Equal(t, "active", result[0].Status)
	assert.NotNil(t, result[0].Controls)
}

func TestConvertToModelsPolicies_Empty(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	result := svc.convertToModelsPolicies(nil)
	assert.Nil(t, result)
}

// ---------------------------------------------------------------------------
// calculateMapSummary
// ---------------------------------------------------------------------------

func TestCalculateMapSummary(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	now := time.Now()
	pastDue := now.Add(-24 * time.Hour)
	futureDue := now.Add(24 * time.Hour)

	tasks := []domain.EvidenceTask{
		{
	ID: "1",
			Framework: "SOC2",
			Status:    "pending",
			Priority:  "high",
			NextDue:   &pastDue,
		},
		{
	ID: "2",
			Framework: "SOC2",
			Status:    "completed",
			Priority:  "medium",
			NextDue:   &futureDue,
		},
		{
	ID: "3",
			Framework: "ISO27001",
			Status:    "pending",
			Priority:  "high",
			NextDue:   nil,
		},
	}

	controls := []domain.Control{
		{ID: "1001", Name: "Control A"},
		{ID: "1002", Name: "Control B"},
	}

	policies := []domain.Policy{
		{ID: "1", Name: "Policy A"},
	}

	frameworkGroups := map[string][]domain.EvidenceTask{
		"SOC2":     tasks[:2],
		"ISO27001": tasks[2:],
	}

	summary := svc.calculateMapSummary(tasks, controls, policies, frameworkGroups)

	assert.Equal(t, 3, summary.TotalTasks)
	assert.Equal(t, 2, summary.TotalControls)
	assert.Equal(t, 1, summary.TotalPolicies)
	assert.Equal(t, 2, summary.FrameworkCounts["SOC2"])
	assert.Equal(t, 1, summary.FrameworkCounts["ISO27001"])
	assert.Equal(t, 2, summary.StatusCounts["pending"])
	assert.Equal(t, 1, summary.StatusCounts["completed"])
	assert.Equal(t, 2, summary.PriorityCounts["high"])
	assert.Equal(t, 1, summary.PriorityCounts["medium"])
	assert.Equal(t, 1, summary.OverdueCount) // Only the past-due task
}

func TestCalculateMapSummary_Empty(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{}

	summary := svc.calculateMapSummary(nil, nil, nil, nil)

	assert.Equal(t, 0, summary.TotalTasks)
	assert.Equal(t, 0, summary.TotalControls)
	assert.Equal(t, 0, summary.TotalPolicies)
	assert.Equal(t, 0, summary.OverdueCount)
}

// ---------------------------------------------------------------------------
// Type tests
// ---------------------------------------------------------------------------

func TestEvidenceMapResult_Fields(t *testing.T) {
	t.Parallel()
	r := EvidenceMapResult{
		TotalRelationships: 42,
		Summary: &EvidenceMapSummary{
			TotalTasks: 10,
		},
	}
	assert.Equal(t, 42, r.TotalRelationships)
	assert.Equal(t, 10, r.Summary.TotalTasks)
}

func TestPromptGenerationOptions_Fields(t *testing.T) {
	t.Parallel()
	opts := PromptGenerationOptions{
		AllTasks:     true,
		OutputFormat: "markdown",
	}
	assert.True(t, opts.AllTasks)
	assert.Equal(t, "markdown", opts.OutputFormat)
}

func TestBulkGenerationOptions_Fields(t *testing.T) {
	t.Parallel()
	opts := BulkGenerationOptions{
		All:    true,
		Format: "csv",
	}
	assert.True(t, opts.All)
}

func TestSubmissionOptions_Fields(t *testing.T) {
	t.Parallel()
	opts := SubmissionOptions{
		DryRun:       true,
		ValidateOnly: false,
	}
	assert.True(t, opts.DryRun)
}

func TestReviewOptions_Fields(t *testing.T) {
	t.Parallel()
	opts := ReviewOptions{
		ShowReasoning: true,
		DetailedMode:  true,
	}
	assert.True(t, opts.ShowReasoning)
}

// ---------------------------------------------------------------------------
// enrichTasksWithURLs
// ---------------------------------------------------------------------------

func TestEnrichTasksWithURLs_WithBaseURL(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{
		config: &config.Config{
			Tugboat: config.TugboatConfig{
				BaseURL: "https://api-my.tugboatlogic.com",
				OrgID:   "42",
			},
		},
	}

	tasks := []domain.EvidenceTask{
		{ID: "100", OrgID: 42},
		{ID: "200", OrgID: 0}, // Will use config OrgID
	}

	svc.enrichTasksWithURLs(tasks)

	assert.NotEmpty(t, tasks[0].TugboatURL)
	assert.Contains(t, tasks[0].TugboatURL, "100")
	assert.NotEmpty(t, tasks[1].TugboatURL)
	assert.Contains(t, tasks[1].TugboatURL, "200")
}

func TestEnrichTasksWithURLs_EmptyBaseURL(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{
		config: &config.Config{
			Tugboat: config.TugboatConfig{
				BaseURL: "",
			},
		},
	}

	tasks := []domain.EvidenceTask{
		{ID: "100", OrgID: 42},
	}

	svc.enrichTasksWithURLs(tasks)
	assert.Empty(t, tasks[0].TugboatURL)
}

func TestEnrichTasksWithURLs_ZeroID(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{
		config: &config.Config{
			Tugboat: config.TugboatConfig{
				BaseURL: "https://api-my.tugboatlogic.com",
				OrgID:   "42",
			},
		},
	}

	tasks := []domain.EvidenceTask{
		{ID: "", OrgID: 42}, // Zero ID, should not generate URL
	}

	svc.enrichTasksWithURLs(tasks)
	assert.Empty(t, tasks[0].TugboatURL)
}

func TestEnrichTasksWithURLs_ZeroOrgID(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{
		config: &config.Config{
			Tugboat: config.TugboatConfig{
				BaseURL: "https://api-my.tugboatlogic.com",
				OrgID:   "not-a-number", // Non-numeric org ID
			},
		},
	}

	tasks := []domain.EvidenceTask{
		{ID: "100", OrgID: 0}, // Zero org on task, non-parseable config org
	}

	svc.enrichTasksWithURLs(tasks)
	// OrgID will be 0 because "not-a-number" doesn't parse
	assert.Empty(t, tasks[0].TugboatURL)
}

func TestEnrichTasksWithURLs_EmptySlice(t *testing.T) {
	t.Parallel()
	svc := &ServiceImpl{
		config: &config.Config{
			Tugboat: config.TugboatConfig{
				BaseURL: "https://api-my.tugboatlogic.com",
			},
		},
	}

	tasks := []domain.EvidenceTask{}
	svc.enrichTasksWithURLs(tasks) // Should not panic
}
