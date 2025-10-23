---
title: "Internal API Documentation"
type: "reference"
category: "api"
tags: ["api", "services", "interfaces", "golang", "architecture"]
related: ["[[cli-commands]]", "[[data-formats]]", "[[naming-conventions]]"]
created: 2025-09-10
modified: 2025-09-10
status: "active"
---

# Internal API Documentation

## Overview

This document describes the internal Go APIs and service interfaces used within GRCTool. These interfaces provide the foundation for evidence collection, data management, and compliance automation functionality.

## Service Architecture

GRCTool follows a layered service architecture with clear separation of concerns:

```
CLI Layer (cmd/)
    ↓
Service Layer (internal/services/)
    ↓
Domain Layer (internal/domain/)
    ↓
Infrastructure Layer (internal/storage, internal/tugboat)
```

## Core Service Interfaces

### DataService Interface

**Location**: `internal/interfaces/services.go`

Provides access to policies, controls, and evidence tasks with caching and relationship management.

```go
type DataService interface {
    // Policy operations
    GetPolicy(id string) (*domain.Policy, error)
    GetPolicyByReferenceAndID(referenceID, numericID string) (*domain.Policy, error)
    GetAllPolicies() ([]domain.Policy, error)
    GetPolicySummary() (*domain.PolicySummary, error)

    // Control operations
    GetControl(id string) (*domain.Control, error)
    GetControlByReferenceAndID(referenceID, numericID string) (*domain.Control, error)
    GetAllControls() ([]domain.Control, error)
    GetControlSummary() (*domain.ControlSummary, error)

    // Evidence task operations
    GetEvidenceTask(id string) (*domain.EvidenceTask, error)
    GetEvidenceTaskByReferenceAndID(referenceID, numericID string) (*domain.EvidenceTask, error)
    GetAllEvidenceTasks() ([]domain.EvidenceTask, error)
    GetEvidenceTaskSummary() (*domain.EvidenceTaskSummary, error)

    // Stats and synchronization
    GetStats() (map[string]interface{}, error)
    SetSyncTime(syncType string, syncTime time.Time) error
    GetSyncTime(syncType string) (time.Time, error)
}
```

#### Usage Examples

**Retrieving Evidence Tasks:**
```go
// Get specific evidence task by ID
task, err := dataService.GetEvidenceTask("ET-0001")
if err != nil {
    return fmt.Errorf("failed to get evidence task: %w", err)
}

// Get evidence task by reference and numeric ID
task, err := dataService.GetEvidenceTaskByReferenceAndID("ET", "0001")
if err != nil {
    return fmt.Errorf("failed to get evidence task: %w", err)
}

// Get all evidence tasks with summary
tasks, err := dataService.GetAllEvidenceTasks()
summary, err := dataService.GetEvidenceTaskSummary()
```

**Policy Management:**
```go
// Retrieve policy by reference ID
policy, err := dataService.GetPolicyByReferenceAndID("POL", "0001")
if err != nil {
    return fmt.Errorf("failed to get policy: %w", err)
}

// Get policy summary for dashboard
summary, err := dataService.GetPolicySummary()
fmt.Printf("Total policies: %d, Updated: %d\n", summary.Total, summary.Updated)
```

### EvidenceService Interface

**Location**: `internal/interfaces/services.go`

Manages evidence records, generation, and validation.

```go
type EvidenceService interface {
    // Evidence record operations
    SaveEvidenceRecord(record *domain.EvidenceRecord) error
    GetEvidenceRecord(id string) (*domain.EvidenceRecord, error)
    GetEvidenceRecordsByTaskID(taskID int) ([]domain.EvidenceRecord, error)
}
```

#### Evidence Generation Request

**Location**: `internal/services/evidence.go`

```go
type EvidenceGenerationRequest struct {
    TaskID      int                    `json:"task_id"`
    Title       string                 `json:"title"`
    Description string                 `json:"description"`
    Format      string                 `json:"format"` // csv, markdown, pdf, etc.
    Tools       []string               `json:"tools"`  // terraform, github, manual, etc.
    Context     map[string]interface{} `json:"context,omitempty"`
}
```

**Usage Examples:**
```go
// Generate evidence for specific task
req := &EvidenceGenerationRequest{
    TaskID:      1001,
    Title:       "Access Control Documentation",
    Description: "Comprehensive access control evidence for SOC 2",
    Format:      "markdown",
    Tools:       []string{"terraform", "github-permissions"},
    Context: map[string]interface{}{
        "framework": "soc2",
        "audit_date": time.Now().Format("2006-01-02"),
    },
}

record, err := evidenceService.GenerateEvidence(ctx, req)
if err != nil {
    return fmt.Errorf("evidence generation failed: %w", err)
}

// Save evidence record
err = evidenceService.SaveEvidenceRecord(record)
if err != nil {
    return fmt.Errorf("failed to save evidence: %w", err)
}
```

### SyncService Interface

**Location**: `internal/interfaces/services.go`

Handles synchronization with external systems like Tugboat Logic.

```go
type SyncService interface {
    // Synchronization operations
    SyncPolicies() error
    SyncControls() error
    SyncEvidenceTasks() error
    SyncAll() error

    // Status operations
    GetSyncStatus() (map[string]interface{}, error)
    GetLastSyncTime(syncType string) (time.Time, error)
}
```

**Usage Examples:**
```go
// Incremental sync of all data types
err := syncService.SyncAll()
if err != nil {
    log.Errorf("sync failed: %v", err)
    return err
}

// Check sync status
status, err := syncService.GetSyncStatus()
if err != nil {
    return err
}

fmt.Printf("Last sync: %v\n", status["last_sync"])
fmt.Printf("Policies synced: %v\n", status["policies_count"])
```

### ToolService Interface

**Location**: `internal/interfaces/services.go`

Provides tool execution capabilities for evidence collection.

```go
type ToolService interface {
    // Tool execution
    ExecuteTool(toolName string, params map[string]interface{}) (interface{}, error)
    ListAvailableTools() ([]string, error)
    GetToolDescription(toolName string) (string, error)
}
```

**Usage Examples:**
```go
// Execute Terraform scanner tool
params := map[string]interface{}{
    "path":           "./infrastructure",
    "resource_types": []string{"aws_security_group", "aws_iam_role"},
    "focus":         "security,encryption",
    "task_ref":      "ET-0011",
}

result, err := toolService.ExecuteTool("terraform-scanner", params)
if err != nil {
    return fmt.Errorf("tool execution failed: %w", err)
}

// Execute GitHub permissions analysis
params = map[string]interface{}{
    "repository":           "myorg/myrepo",
    "include_collaborators": true,
    "include_teams":        true,
    "task_ref":            "ET-0001",
}

result, err = toolService.ExecuteTool("github-permissions", params)
```

### ConfigService Interface

**Location**: `internal/interfaces/services.go`

Manages configuration access and tool-specific settings.

```go
type ConfigService interface {
    // Configuration operations
    GetDataDirectory() string
    GetCacheDirectory() string
    GetLocalDataDirectory() string
    IsOfflineModeEnabled() bool
    GetToolConfiguration(toolName string) (map[string]interface{}, error)
}
```

**Usage Examples:**
```go
// Get directories for file operations
dataDir := configService.GetDataDirectory()
cacheDir := configService.GetCacheDirectory()

// Check offline mode
if configService.IsOfflineModeEnabled() {
    log.Info("Running in offline mode")
    return useOfflineData()
}

// Get tool-specific configuration
toolConfig, err := configService.GetToolConfiguration("terraform-scanner")
if err != nil {
    return fmt.Errorf("failed to get tool config: %w", err)
}

maxFiles := toolConfig["max_files"].(int)
timeout := toolConfig["timeout"].(string)
```

## Domain Models

### EvidenceTask Domain Model

**Location**: `internal/domain/evidence_task.go`

```go
type EvidenceTask struct {
    ID              int                 `json:"id"`
    ReferenceID     string             `json:"reference_id"`     // ET-0001
    NumericID       string             `json:"numeric_id"`       // 0001
    TugboatID       int                `json:"tugboat_id"`       // 328001
    Name            string             `json:"name"`
    Description     string             `json:"description"`
    Status          EvidenceTaskStatus `json:"status"`
    Framework       string             `json:"framework"`        // soc2, iso27001
    Priority        Priority           `json:"priority"`
    Category        string             `json:"category"`
    RequiredSources []string           `json:"required_sources"` // terraform, github, manual
    Automation      AutomationLevel    `json:"automation"`
    CreatedAt       time.Time          `json:"created_at"`
    UpdatedAt       time.Time          `json:"updated_at"`
    DueDate         *time.Time         `json:"due_date"`
}

type EvidenceTaskStatus string

const (
    EvidenceTaskStatusPending    EvidenceTaskStatus = "pending"
    EvidenceTaskStatusInProgress EvidenceTaskStatus = "in_progress"
    EvidenceTaskStatusCompleted  EvidenceTaskStatus = "completed"
    EvidenceTaskStatusOverdue    EvidenceTaskStatus = "overdue"
)

type AutomationLevel string

const (
    AutomationLevelFull    AutomationLevel = "full"      // 95%+ automated
    AutomationLevelHigh    AutomationLevel = "high"      // 75-95% automated
    AutomationLevelMedium  AutomationLevel = "medium"    // 50-75% automated
    AutomationLevelLow     AutomationLevel = "low"       // 25-50% automated
    AutomationLevelManual  AutomationLevel = "manual"    // <25% automated
)
```

### EvidenceRecord Domain Model

**Location**: `internal/domain/evidence_record.go`

```go
type EvidenceRecord struct {
    ID               string                 `json:"id"`
    TaskID           int                   `json:"task_id"`
    TaskReferenceID  string                `json:"task_reference_id"`  // ET-0001
    Title            string                `json:"title"`
    Content          string                `json:"content"`
    ContentType      string                `json:"content_type"`       // markdown, json, csv
    Sources          []EvidenceSource      `json:"sources"`
    Metadata         EvidenceMetadata      `json:"metadata"`
    QualityScore     int                   `json:"quality_score"`      // 0-100
    ValidationStatus ValidationStatus     `json:"validation_status"`
    CreatedAt        time.Time            `json:"created_at"`
    CreatedBy        string               `json:"created_by"`          // system, user email
    Version          int                  `json:"version"`
    FilePath         string               `json:"file_path"`
}

type EvidenceSource struct {
    Type        string                 `json:"type"`         // terraform, github, manual
    Tool        string                 `json:"tool"`         // terraform-scanner, github-permissions
    Location    string                 `json:"location"`     // file path, repository, URL
    Timestamp   time.Time             `json:"timestamp"`
    Metadata    map[string]interface{} `json:"metadata"`
    Checksum    string                `json:"checksum"`     // For integrity verification
}

type EvidenceMetadata struct {
    Framework       string            `json:"framework"`        // soc2, iso27001
    Controls        []string          `json:"controls"`         // Control IDs this evidence supports
    Completeness    float64          `json:"completeness"`     // 0.0-1.0
    Accuracy        float64          `json:"accuracy"`         // 0.0-1.0
    CollectionTime  time.Duration    `json:"collection_time"`  // Time taken to collect
    AutomationLevel string           `json:"automation_level"` // full, high, medium, low, manual
    ReviewRequired  bool             `json:"review_required"`
    Tags            []string         `json:"tags"`
}
```

### Policy and Control Models

**Location**: `internal/domain/policy.go`, `internal/domain/control.go`

```go
type Policy struct {
    ID           string    `json:"id"`
    ReferenceID  string    `json:"reference_id"`  // POL-0001
    TugboatID    int       `json:"tugboat_id"`
    Name         string    `json:"name"`
    Description  string    `json:"description"`
    Content      string    `json:"content"`
    Version      string    `json:"version"`
    Status       string    `json:"status"`
    Owner        string    `json:"owner"`
    Reviewers    []string  `json:"reviewers"`
    ApprovedBy   string    `json:"approved_by"`
    ApprovedDate *time.Time `json:"approved_date"`
    ReviewDate   *time.Time `json:"review_date"`
    CreatedAt    time.Time `json:"created_at"`
    UpdatedAt    time.Time `json:"updated_at"`
}

type Control struct {
    ID                string    `json:"id"`
    ReferenceID       string    `json:"reference_id"`       // AC-01, CC-01_1
    TugboatID         int       `json:"tugboat_id"`
    Name              string    `json:"name"`
    Description       string    `json:"description"`
    Framework         string    `json:"framework"`          // soc2, iso27001
    Category          string    `json:"category"`
    ImplementationStatus string `json:"implementation_status"`
    ControlType       string    `json:"control_type"`       // preventive, detective, corrective
    Frequency         string    `json:"frequency"`          // continuous, daily, monthly, etc.
    Owner             string    `json:"owner"`
    EvidenceRequirements []string `json:"evidence_requirements"`
    RelatedControls   []string  `json:"related_controls"`
    CreatedAt         time.Time `json:"created_at"`
    UpdatedAt         time.Time `json:"updated_at"`
}
```

## Tool Execution API

### Tool Interface

**Location**: `internal/tools/interface.go`

```go
type Tool interface {
    Name() string
    Description() string
    Category() string
    Execute(ctx context.Context, params map[string]interface{}) (string, *models.EvidenceSource, error)
}

type ToolRegistry interface {
    Register(tool Tool) error
    Get(name string) (Tool, error)
    List() []Tool
    GetByCategory(category string) []Tool
}
```

### Tool Execution Context

**Location**: `internal/tools/context.go`

```go
type ToolContext struct {
    Logger      logger.Logger
    Config      *config.Config
    TaskRef     string                // ET-0001
    Parameters  map[string]interface{}
    OutputDir   string
    Quiet       bool
    correlationID string
}

type ToolMeta struct {
    CorrelationID    string        `json:"correlation_id"`
    TaskRef          string        `json:"task_ref,omitempty"`
    Tool             string        `json:"tool"`
    Version          string        `json:"version"`
    Timestamp        time.Time     `json:"timestamp"`
    DurationMS       int64         `json:"duration_ms"`
    AuthStatus       string        `json:"auth_status,omitempty"`
    DataSource       string        `json:"data_source,omitempty"`
    SchemaVersion    string        `json:"schema_version"`
}

type ToolEnvelope struct {
    Success bool        `json:"success"`
    Content string      `json:"content"`
    Meta    ToolMeta    `json:"meta"`
    Error   *ToolError  `json:"error,omitempty"`
}
```

### Tool Output Standards

All tools must return consistent JSON envelopes:

**Success Response:**
```json
{
  "success": true,
  "content": "Evidence content or analysis results",
  "meta": {
    "correlation_id": "uuid-v4",
    "task_ref": "ET-0001",
    "tool": "terraform-scanner",
    "version": "1.0.0",
    "timestamp": "2025-09-10T14:30:00Z",
    "duration_ms": 5234,
    "auth_status": "authenticated",
    "data_source": "file_system",
    "schema_version": "1.0"
  }
}
```

**Error Response:**
```json
{
  "success": false,
  "content": "",
  "meta": {
    "correlation_id": "uuid-v4",
    "tool": "terraform-scanner",
    "version": "1.0.0",
    "timestamp": "2025-09-10T14:30:00Z",
    "duration_ms": 1234,
    "schema_version": "1.0"
  },
  "error": {
    "code": "INVALID_PATH",
    "message": "Terraform configuration path not found",
    "details": {
      "path": "./invalid-path",
      "suggestions": ["./infrastructure", "./terraform"]
    }
  }
}
```

## Error Handling Patterns

### Service-Level Errors

**Standard Error Types:**
```go
type ServiceError struct {
    Code    string `json:"code"`
    Message string `json:"message"`
    Context map[string]interface{} `json:"context,omitempty"`
    Cause   error  `json:"-"`
}

func (e *ServiceError) Error() string {
    return fmt.Sprintf("%s: %s", e.Code, e.Message)
}

// Predefined error codes
const (
    ErrCodeNotFound        = "NOT_FOUND"
    ErrCodeInvalidInput    = "INVALID_INPUT" 
    ErrCodeAuthRequired    = "AUTH_REQUIRED"
    ErrCodePermissionDenied = "PERMISSION_DENIED"
    ErrCodeRateLimited     = "RATE_LIMITED"
    ErrCodeServiceUnavailable = "SERVICE_UNAVAILABLE"
    ErrCodeInternalError   = "INTERNAL_ERROR"
)
```

**Error Wrapping:**
```go
// In service implementations
func (s *dataService) GetEvidenceTask(id string) (*domain.EvidenceTask, error) {
    task, err := s.storage.GetEvidenceTask(id)
    if err != nil {
        if errors.Is(err, storage.ErrNotFound) {
            return nil, &ServiceError{
                Code:    ErrCodeNotFound,
                Message: fmt.Sprintf("evidence task %s not found", id),
                Context: map[string]interface{}{"task_id": id},
                Cause:   err,
            }
        }
        return nil, fmt.Errorf("failed to get evidence task: %w", err)
    }
    return task, nil
}
```

## Context and Cancellation

All service methods support context for cancellation and timeout handling:

```go
// Service method with context
func (s *evidenceService) GenerateEvidence(ctx context.Context, req *EvidenceGenerationRequest) (*domain.EvidenceRecord, error) {
    // Check context cancellation
    select {
    case <-ctx.Done():
        return nil, ctx.Err()
    default:
    }

    // Set operation timeout
    ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
    defer cancel()

    // Long-running operation with context
    return s.executeEvidenceGeneration(ctx, req)
}

// Usage with timeout
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
defer cancel()

record, err := evidenceService.GenerateEvidence(ctx, req)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Error("Evidence generation timed out")
    }
    return err
}
```

## Testing Interfaces

### Service Mocks

**Mock Generation:**
```go
//go:generate mockery --name=DataService --output=mocks --outpkg=mocks

// Usage in tests
func TestEvidenceGeneration(t *testing.T) {
    mockDataService := &mocks.DataService{}
    mockDataService.On("GetEvidenceTask", "ET-0001").Return(testTask, nil)
    
    service := NewEvidenceService(mockDataService, logger)
    
    result, err := service.GenerateEvidence(ctx, request)
    assert.NoError(t, err)
    assert.NotNil(t, result)
    
    mockDataService.AssertExpectations(t)
}
```

### Integration Testing

**Test Utilities:**
```go
// Test service setup
func SetupTestServices(t *testing.T) (*TestServices, func()) {
    tempDir := t.TempDir()
    cfg := &config.Config{
        Storage: config.StorageConfig{
            DataDir:  filepath.Join(tempDir, "data"),
            CacheDir: filepath.Join(tempDir, "cache"),
        },
    }
    
    logger := logger.NewTestLogger(t)
    dataService := NewDataService(cfg, logger)
    evidenceService := NewEvidenceService(dataService, logger)
    
    services := &TestServices{
        Data:     dataService,
        Evidence: evidenceService,
        Config:   cfg,
        Logger:   logger,
    }
    
    cleanup := func() {
        os.RemoveAll(tempDir)
    }
    
    return services, cleanup
}
```

## Performance Considerations

### Caching Strategies

**Service-Level Caching:**
```go
type cachedDataService struct {
    underlying DataService
    cache      *cache.Cache
    ttl        time.Duration
}

func (s *cachedDataService) GetEvidenceTask(id string) (*domain.EvidenceTask, error) {
    // Check cache first
    if cached, found := s.cache.Get(fmt.Sprintf("task:%s", id)); found {
        return cached.(*domain.EvidenceTask), nil
    }
    
    // Fetch from underlying service
    task, err := s.underlying.GetEvidenceTask(id)
    if err != nil {
        return nil, err
    }
    
    // Cache result
    s.cache.Set(fmt.Sprintf("task:%s", id), task, s.ttl)
    return task, nil
}
```

### Batch Operations

**Batch Processing Interface:**
```go
type BatchProcessor interface {
    ProcessBatch(ctx context.Context, items []BatchItem) (*BatchResult, error)
}

type BatchItem struct {
    ID      string                 `json:"id"`
    Type    string                 `json:"type"`
    Payload map[string]interface{} `json:"payload"`
}

type BatchResult struct {
    Processed int           `json:"processed"`
    Failed    int           `json:"failed"`
    Errors    []BatchError  `json:"errors,omitempty"`
    Duration  time.Duration `json:"duration"`
}
```

## Migration and Versioning

### API Versioning

**Version-Aware Services:**
```go
type VersionedService struct {
    version string
    service Service
}

func NewVersionedService(version string) Service {
    switch version {
    case "v1":
        return &ServiceV1{}
    case "v2":  
        return &ServiceV2{}
    default:
        return &ServiceV1{} // Default to v1
    }
}
```

### Schema Migration

**Migration Interface:**
```go
type Migrator interface {
    Migrate(ctx context.Context, fromVersion, toVersion string) error
    GetCurrentVersion(ctx context.Context) (string, error)
    GetSupportedVersions() []string
}
```

## References

- [[cli-commands]] - Command-line interface documentation
- [[data-formats]] - JSON schemas and data structures
- [[naming-conventions]] - Naming standards and conventions
- [[helix/04-build/development-practices|Service Implementation Examples]] - Service patterns and examples
- [[helix/03-test/testing-strategy|Testing Guide]] - Testing strategy and procedures
- [[helix/04-build/development-practices|Error Handling Best Practices]] - Error handling guidelines

---

*This API documentation is automatically updated from Go source code analysis. Last updated: 2025-09-10*