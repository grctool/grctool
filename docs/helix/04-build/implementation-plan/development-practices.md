---
title: "Development Practices and Standards"
phase: "04-build"
category: "development"
tags: ["coding-standards", "practices", "security", "quality", "go"]
related: ["coding-standards", "security-practices", "contribution-guide"]
created: 2025-01-10
updated: 2025-01-10
helix_mapping: "Consolidated from 04-Development/coding-standards.md and contributing.md"
---

# Development Practices and Standards

## Overview

This document outlines the development practices, coding standards, and quality requirements for GRCTool. These standards ensure code consistency, security, maintainability, and alignment with compliance requirements.

## Core Development Principles

### 1. Security-First Development
- **Secure by Default**: All features should be secure by default
- **Threat Modeling**: Consider security implications for all changes
- **Zero Trust**: Verify all inputs, authenticate all requests
- **Fail Secure**: Default to secure state on errors

### 2. Compliance-Aware Development
- **Audit Trail**: All operations must be traceable
- **Data Integrity**: Ensure data accuracy and consistency
- **Documentation**: Comprehensive documentation for audit purposes
- **Reproducibility**: Deterministic and repeatable operations

### 3. Quality-Driven Development
- **Test-Driven Development**: Write tests before or alongside code
- **Code Review**: All changes require peer review
- **Continuous Integration**: Automated testing and quality checks
- **Performance Awareness**: Consider performance implications

## Go Coding Standards

### Code Organization

#### Package Structure
```go
// Package declaration with meaningful name
package evidence

// Imports grouped by: standard library, third-party, internal
import (
    "context"
    "fmt"
    "time"

    "github.com/spf13/cobra"
    "github.com/rs/zerolog/log"

    "github.com/yourorg/grctool/internal/models"
    "github.com/yourorg/grctool/internal/storage"
)
```

#### File Organization
```go
// File structure (top to bottom):
// 1. Package declaration and imports
// 2. Constants and variables
// 3. Types (interfaces first, then structs)
// 4. Constructor functions
// 5. Methods (grouped by receiver)
// 6. Helper functions

const (
    DefaultTimeout = 30 * time.Second
    MaxRetries     = 3
)

type EvidenceService interface {
    Generate(ctx context.Context, taskID string) (*Evidence, error)
    Validate(evidence *Evidence) error
}

type evidenceService struct {
    storage storage.Repository
    client  *http.Client
}

func NewEvidenceService(storage storage.Repository) EvidenceService {
    return &evidenceService{
        storage: storage,
        client:  &http.Client{Timeout: DefaultTimeout},
    }
}
```

### Naming Conventions

#### Variables and Functions
```go
// Use camelCase for unexported names
var configPath string
func generateEvidence() error { }

// Use PascalCase for exported names
var DefaultConfig Config
func GenerateEvidence() error { }

// Use descriptive names
var evidenceTaskCache map[string]*EvidenceTask  // Good
var cache map[string]*EvidenceTask              // Avoid

// Boolean variables should be questions
var isValid bool       // Good
var hasPermission bool // Good
var valid bool         // Avoid
```

#### Types and Interfaces
```go
// Interface names should describe behavior
type Reader interface { }      // Good - describes what it does
type FileReader interface { }  // Good - specific behavior

// Struct names should describe the entity
type EvidenceTask struct { }   // Good - clear entity
type Config struct { }         // Good - clear purpose

// Avoid stuttering
type evidence.Task struct { }  // Good
type evidence.EvidenceTask struct { } // Avoid - stuttering
```

#### Constants
```go
// Use PascalCase for exported constants
const (
    StatusPending   = "pending"
    StatusCompleted = "completed"
    StatusFailed    = "failed"
)

// Group related constants
const (
    // HTTP status codes
    StatusOK                  = 200
    StatusBadRequest         = 400
    StatusUnauthorized       = 401
    StatusInternalServerError = 500
)
```

### Error Handling

#### Error Creation
```go
// Use standard error patterns
func processEvidence(taskID string) error {
    if taskID == "" {
        return errors.New("task ID cannot be empty")
    }

    // Wrap errors with context
    if err := validateTask(taskID); err != nil {
        return fmt.Errorf("failed to validate task %s: %w", taskID, err)
    }

    return nil
}
```

#### Error Types
```go
// Define custom error types for specific cases
type ValidationError struct {
    Field   string
    Value   interface{}
    Message string
}

func (e *ValidationError) Error() string {
    return fmt.Sprintf("validation failed for field %s: %s", e.Field, e.Message)
}

// Use error variables for common errors
var (
    ErrTaskNotFound     = errors.New("evidence task not found")
    ErrInvalidFormat    = errors.New("invalid format")
    ErrPermissionDenied = errors.New("permission denied")
)
```

#### Error Handling Patterns
```go
// Check errors immediately
result, err := processEvidence(taskID)
if err != nil {
    return fmt.Errorf("processing failed: %w", err)
}

// Don't ignore errors unless explicitly intended
_ = file.Close() // OK - documented that we're ignoring

file.Close() // Avoid - unclear if intentional
```

### Function Design

#### Function Signatures
```go
// Keep function signatures simple and clear
func GenerateEvidence(ctx context.Context, taskID string) (*Evidence, error) {
    // Good - clear parameters and return values
}

// Avoid long parameter lists - use structs instead
type GenerateEvidenceRequest struct {
    TaskID       string
    Format       string
    IncludeRaw   bool
    Timeout      time.Duration
}

func GenerateEvidence(ctx context.Context, req GenerateEvidenceRequest) (*Evidence, error) {
    // Better for complex operations
}
```

#### Function Length
```go
// Keep functions focused and small (< 50 lines ideally)
func validateEvidenceTask(task *EvidenceTask) error {
    if err := validateTaskID(task.ID); err != nil {
        return err
    }

    if err := validateTaskName(task.Name); err != nil {
        return err
    }

    return validateTaskControls(task.Controls)
}

// Extract complex logic into separate functions
func validateTaskID(id string) error {
    if id == "" {
        return ErrEmptyTaskID
    }

    if !strings.HasPrefix(id, "ET-") {
        return ErrInvalidTaskIDPrefix
    }

    return nil
}
```

### Struct Design

#### Struct Tags
```go
type EvidenceTask struct {
    ID          string    `json:"id" yaml:"id"`
    Name        string    `json:"name" yaml:"name"`
    Description string    `json:"description,omitempty" yaml:"description,omitempty"`
    CreatedAt   time.Time `json:"created_at" yaml:"created_at"`

    // Use validate tags for input validation
    Email       string    `json:"email" validate:"required,email"`
    Age         int       `json:"age" validate:"min=0,max=150"`
}
```

#### Constructor Patterns
```go
// Provide constructors for complex types
func NewEvidenceTask(id, name string) *EvidenceTask {
    return &EvidenceTask{
        ID:        id,
        Name:      name,
        CreatedAt: time.Now(),
        Status:    StatusPending,
    }
}

// Use functional options for flexible configuration
type EvidenceTaskOption func(*EvidenceTask)

func WithDescription(desc string) EvidenceTaskOption {
    return func(t *EvidenceTask) {
        t.Description = desc
    }
}

func NewEvidenceTaskWithOptions(id, name string, opts ...EvidenceTaskOption) *EvidenceTask {
    task := NewEvidenceTask(id, name)
    for _, opt := range opts {
        opt(task)
    }
    return task
}
```

### Interface Design

#### Interface Principles
```go
// Keep interfaces small and focused
type Reader interface {
    Read([]byte) (int, error)
}

type Writer interface {
    Write([]byte) (int, error)
}

// Compose interfaces when needed
type ReadWriter interface {
    Reader
    Writer
}

// Define interfaces where they're used, not where they're implemented
type UserService struct {
    storage UserRepository // Interface defined in this package
}

type UserRepository interface {
    GetUser(id string) (*User, error)
    SaveUser(user *User) error
}
```

#### Interface Naming
```go
// Single-method interfaces often end in -er
type Validator interface {
    Validate(input interface{}) error
}

type Generator interface {
    Generate(context.Context) ([]byte, error)
}

// Multi-method interfaces describe the concept
type EvidenceService interface {
    Generate(ctx context.Context, taskID string) (*Evidence, error)
    Validate(evidence *Evidence) error
    Store(evidence *Evidence) error
}
```

## Security Coding Practices

### Input Validation
```go
// Validate all inputs at boundaries
func ProcessEvidenceTask(input string) error {
    // Sanitize input
    cleaned := strings.TrimSpace(input)
    if cleaned == "" {
        return ErrEmptyInput
    }

    // Validate format
    if !isValidTaskID(cleaned) {
        return ErrInvalidFormat
    }

    // Check length limits
    if len(cleaned) > MaxTaskIDLength {
        return ErrInputTooLong
    }

    return processValidatedInput(cleaned)
}

func isValidTaskID(id string) bool {
    // Use strict validation patterns
    pattern := `^ET-\d{4}$`
    matched, _ := regexp.MatchString(pattern, id)
    return matched
}
```

### Secure String Handling
```go
// Use byte slices for sensitive data when possible
func hashPassword(password []byte) ([]byte, error) {
    defer func() {
        // Clear sensitive data from memory
        for i := range password {
            password[i] = 0
        }
    }()

    return bcrypt.GenerateFromPassword(password, bcrypt.DefaultCost)
}

// Avoid logging sensitive information
func authenticateUser(username, password string) error {
    log.Info().
        Str("username", username).
        // Never log password or token
        Msg("Authentication attempt")

    return validateCredentials(username, password)
}
```

### Safe Command Execution
```go
// Validate commands against allowlist
var allowedCommands = map[string]bool{
    "git":       true,
    "terraform": true,
    "kubectl":   true,
}

func executeCommand(cmd string, args ...string) (string, error) {
    if !allowedCommands[cmd] {
        return "", ErrCommandNotAllowed
    }

    // Sanitize arguments
    for i, arg := range args {
        if err := validateArgument(arg); err != nil {
            return "", fmt.Errorf("invalid argument %d: %w", i, err)
        }
    }

    // Execute with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    return executeWithContext(ctx, cmd, args...)
}
```

## Documentation Standards

### Code Comments
```go
// Package documentation at the top of the main file
// Package evidence provides tools for automated SOC2 evidence collection.
//
// The evidence package supports multiple collection methods including:
//   - Infrastructure analysis via Terraform
//   - GitHub repository security scanning
//   - Policy document analysis
//
// All evidence collection operations are designed to be deterministic
// and produce auditable outputs suitable for compliance reporting.
package evidence

// Document exported types with their purpose
// EvidenceTask represents a single evidence collection requirement
// from a compliance framework such as SOC2 or ISO27001.
type EvidenceTask struct {
    // ID uniquely identifies the evidence task (format: ET-####)
    ID string `json:"id"`

    // Name provides a human-readable description of the task
    Name string `json:"name"`
}

// Document exported functions with their behavior
// GenerateEvidence creates compliance evidence for the specified task.
//
// The function performs the following steps:
//   1. Validates the task ID format
//   2. Retrieves task details from storage
//   3. Executes appropriate evidence collection tools
//   4. Formats the output according to requirements
//
// Returns an error if the task is not found or evidence generation fails.
func GenerateEvidence(ctx context.Context, taskID string) (*Evidence, error) {
    // Implementation comments for complex logic
    // Use AI-powered analysis to ensure comprehensive coverage
    if shouldUseAI(taskID) {
        return generateWithAI(ctx, taskID)
    }

    return generateStandard(ctx, taskID)
}
```

### Function Documentation
```go
// Document complex functions with examples
// ParseEvidenceConfig parses configuration from YAML or JSON format.
//
// Supported formats:
//   - YAML: .grctool.yaml, .grctool.yml
//   - JSON: .grctool.json
//
// Example YAML configuration:
//   evidence:
//     claude:
//       api_key: "${CLAUDE_API_KEY}"
//       model: "claude-3-sonnet-20240229"
//     tools:
//       terraform:
//         enabled: true
//         scan_paths: ["./terraform"]
//
// Returns ErrInvalidFormat if the configuration cannot be parsed.
func ParseEvidenceConfig(data []byte, format string) (*Config, error) {
    // Implementation
}
```

## Testing Standards

### Test Structure
```go
func TestEvidenceGeneration(t *testing.T) {
    // Use table-driven tests with descriptive names
    tests := map[string]struct {
        taskID      string
        expected    *Evidence
        expectError bool
        errorMsg    string
    }{
        "valid SOC2 access control task": {
            taskID:   "ET-0001",
            expected: &Evidence{ID: "ET-0001", Type: "access_control"},
        },
        "invalid task ID format": {
            taskID:      "INVALID",
            expectError: true,
            errorMsg:    "invalid task ID format",
        },
        "non-existent task": {
            taskID:      "ET-9999",
            expectError: true,
            errorMsg:    "task not found",
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            result, err := GenerateEvidence(context.Background(), tt.taskID)

            if tt.expectError {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errorMsg)
                assert.Nil(t, result)
            } else {
                assert.NoError(t, err)
                assert.Equal(t, tt.expected, result)
            }
        })
    }
}
```

### Test Helpers
```go
// Create test helpers for common operations
func createTestEvidenceTask(t *testing.T, id string) *EvidenceTask {
    t.Helper()

    return &EvidenceTask{
        ID:        id,
        Name:      fmt.Sprintf("Test Task %s", id),
        CreatedAt: time.Now(),
        Status:    StatusPending,
    }
}

// Use builders for complex test data
func TestComplexEvidenceProcessing(t *testing.T) {
    task := test.NewEvidenceTaskBuilder().
        WithID("ET-0001").
        WithName("Access Control Review").
        WithControls("CC6.1", "CC6.2").
        WithSensitive(true).
        Build()

    result, err := ProcessEvidence(task)
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

## Performance Guidelines

### Memory Management
```go
// Avoid memory leaks with proper cleanup
func processLargeFile(filename string) error {
    file, err := os.Open(filename)
    if err != nil {
        return err
    }
    defer file.Close() // Always close resources

    // Use streaming for large files
    scanner := bufio.NewScanner(file)
    for scanner.Scan() {
        if err := processLine(scanner.Text()); err != nil {
            return err
        }
    }

    return scanner.Err()
}

// Use object pools for frequently allocated objects
var evidencePool = sync.Pool{
    New: func() interface{} {
        return &Evidence{}
    },
}

func processEvidence() error {
    evidence := evidencePool.Get().(*Evidence)
    defer evidencePool.Put(evidence)

    // Reset before use
    *evidence = Evidence{}

    // Process evidence
    return nil
}
```

### Concurrency Patterns
```go
// Use context for cancellation and timeouts
func generateEvidenceWithTimeout(ctx context.Context, taskID string) (*Evidence, error) {
    ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
    defer cancel()

    return generateEvidenceWithContext(ctx, taskID)
}

// Use worker pools for parallel processing
func processBatchEvidence(tasks []string) error {
    const maxWorkers = 10
    sem := make(chan struct{}, maxWorkers)
    var wg sync.WaitGroup

    for _, task := range tasks {
        wg.Add(1)
        go func(taskID string) {
            defer wg.Done()
            sem <- struct{}{} // Acquire semaphore
            defer func() { <-sem }() // Release semaphore

            if err := processEvidence(taskID); err != nil {
                log.Error().Err(err).Str("task", taskID).Msg("Failed to process evidence")
            }
        }(task)
    }

    wg.Wait()
    return nil
}
```

## Quality Assurance

### Code Review Checklist

#### Functionality
- [ ] Code accomplishes the intended functionality
- [ ] Edge cases and error conditions are handled
- [ ] Function signatures are appropriate
- [ ] Return values are checked and handled

#### Security
- [ ] Input validation is performed
- [ ] No hardcoded secrets or credentials
- [ ] Proper authentication and authorization
- [ ] Safe handling of sensitive data

#### Performance
- [ ] No obvious performance issues
- [ ] Appropriate use of resources
- [ ] Proper memory management
- [ ] Suitable for expected load

#### Maintainability
- [ ] Code is readable and well-documented
- [ ] Follows established patterns
- [ ] Appropriate test coverage
- [ ] No code duplication

#### Compliance
- [ ] Audit trail preservation
- [ ] Data integrity maintained
- [ ] Proper error handling and logging
- [ ] Documentation for compliance

### Pre-commit Hooks

All code must pass automated quality checks:

```bash
# Code formatting
make fmt

# Linting
make lint

# Security scanning
make security-scan

# Tests
make test

# Coverage check
make coverage-check
```

## Contribution Workflow

### Development Process
1. **Create Feature Branch**: Branch from `main` for new features
2. **Implement with Tests**: Write tests before or alongside code
3. **Local Quality Checks**: Run full quality suite locally
4. **Create Pull Request**: Submit for peer review
5. **Address Feedback**: Iterate based on review comments
6. **Merge**: Squash and merge after approval

### Commit Standards
```bash
# Use conventional commit format
feat: add evidence generation for SOC2 CC6.1 control
fix: resolve authentication token refresh issue
docs: update API documentation for evidence endpoints
test: add integration tests for Terraform scanner
refactor: improve error handling in storage layer
```

### Branch Naming
```bash
# Use descriptive branch names
feature/evidence-generation-api
fix/authentication-token-refresh
docs/api-documentation-update
test/terraform-scanner-integration
refactor/storage-error-handling
```

## Secure GRC Development Examples

### Evidence Collection Tool Implementation

#### Secure Evidence Collector Pattern
```go
// internal/tools/secure_evidence_collector.go
package tools

import (
    "context"
    "crypto/sha256"
    "fmt"
    "time"

    "github.com/rs/zerolog/log"

    "github.com/yourorg/grctool/internal/audit"
    "github.com/yourorg/grctool/internal/crypto"
    "github.com/yourorg/grctool/internal/models"
)

// SecureEvidenceCollector implements evidence collection with built-in security controls
type SecureEvidenceCollector struct {
    auditTrail    audit.Trail
    cryptoService crypto.Service
    validator     InputValidator
    rateLimiter   RateLimiter
    config        CollectorConfig
}

// CollectorConfig defines security and operational parameters
type CollectorConfig struct {
    MaxConcurrentCollections int           `yaml:"max_concurrent_collections"`
    CollectionTimeout        time.Duration `yaml:"collection_timeout"`
    EnableAuditLogging      bool           `yaml:"enable_audit_logging"`
    RequireSignedEvidence   bool           `yaml:"require_signed_evidence"`
    SensitiveDataRedaction  bool           `yaml:"sensitive_data_redaction"`
}

// CollectEvidence performs secure evidence collection with comprehensive audit trail
func (sec *SecureEvidenceCollector) CollectEvidence(ctx context.Context, req *CollectionRequest) (*SecureEvidence, error) {
    // 1. Input validation and sanitization
    if err := sec.validator.ValidateRequest(req); err != nil {
        sec.auditTrail.LogSecurityEvent(audit.SecurityEvent{
            Type:        "input_validation_failure",
            RequestID:   req.ID,
            UserID:      req.UserID,
            Description: "Evidence collection request failed validation",
            Severity:    "high",
            Timestamp:   time.Now(),
        })
        return nil, fmt.Errorf("request validation failed: %w", err)
    }

    // 2. Rate limiting to prevent abuse
    if err := sec.rateLimiter.CheckLimit(ctx, req.UserID); err != nil {
        sec.auditTrail.LogSecurityEvent(audit.SecurityEvent{
            Type:        "rate_limit_exceeded",
            RequestID:   req.ID,
            UserID:      req.UserID,
            Description: "User exceeded evidence collection rate limit",
            Severity:    "medium",
            Timestamp:   time.Now(),
        })
        return nil, fmt.Errorf("rate limit exceeded: %w", err)
    }

    // 3. Create audit trail entry for collection start
    auditID := sec.auditTrail.StartOperation(audit.Operation{
        Type:        "evidence_collection",
        RequestID:   req.ID,
        UserID:      req.UserID,
        TaskID:      req.TaskID,
        StartTime:   time.Now(),
        Parameters:  sanitizeForAudit(req),
    })
    defer sec.auditTrail.EndOperation(auditID)

    // 4. Execute collection with timeout and monitoring
    ctx, cancel := context.WithTimeout(ctx, sec.config.CollectionTimeout)
    defer cancel()

    evidence, err := sec.executeSecureCollection(ctx, req)
    if err != nil {
        sec.auditTrail.LogError(auditID, err)
        return nil, fmt.Errorf("evidence collection failed: %w", err)
    }

    // 5. Apply data protection measures
    protectedEvidence, err := sec.protectEvidence(evidence, req)
    if err != nil {
        return nil, fmt.Errorf("evidence protection failed: %w", err)
    }

    // 6. Generate cryptographic proof of integrity
    if sec.config.RequireSignedEvidence {
        signature, err := sec.cryptoService.SignEvidence(protectedEvidence)
        if err != nil {
            return nil, fmt.Errorf("evidence signing failed: %w", err)
        }
        protectedEvidence.Signature = signature
    }

    // 7. Log successful completion
    sec.auditTrail.LogSuccess(auditID, audit.SuccessEvent{
        EvidenceID:     protectedEvidence.ID,
        CollectionTime: time.Since(req.StartTime),
        DataSize:       len(protectedEvidence.Content),
        ChecksumSHA256: fmt.Sprintf("%x", sha256.Sum256(protectedEvidence.Content)),
    })

    return protectedEvidence, nil
}

// executeSecureCollection performs the actual evidence collection with security monitoring
func (sec *SecureEvidenceCollector) executeSecureCollection(ctx context.Context, req *CollectionRequest) (*Evidence, error) {
    // Monitor for suspicious patterns during collection
    monitor := NewSecurityMonitor(req)
    defer monitor.Finalize()

    // Select appropriate collection strategy based on evidence type
    collector, err := sec.getCollectorForType(req.EvidenceType)
    if err != nil {
        return nil, fmt.Errorf("no collector available for type %s: %w", req.EvidenceType, err)
    }

    // Execute with resource limits and monitoring
    resourceLimiter := NewResourceLimiter(sec.config)
    limitedCtx := resourceLimiter.LimitContext(ctx)

    evidence, err := collector.Collect(limitedCtx, req)
    if err != nil {
        monitor.RecordError(err)
        return nil, err
    }

    // Validate collected evidence meets quality and security standards
    if err := sec.validateCollectedEvidence(evidence); err != nil {
        return nil, fmt.Errorf("evidence validation failed: %w", err)
    }

    return evidence, nil
}

// protectEvidence applies data protection and privacy controls
func (sec *SecureEvidenceCollector) protectEvidence(evidence *Evidence, req *CollectionRequest) (*SecureEvidence, error) {
    protected := &SecureEvidence{
        Evidence:  *evidence,
        CreatedAt: time.Now(),
        RequestID: req.ID,
        UserID:    req.UserID,
    }

    // Apply data classification
    classification, err := sec.classifyEvidence(evidence)
    if err != nil {
        return nil, fmt.Errorf("evidence classification failed: %w", err)
    }
    protected.Classification = classification

    // Redact sensitive information if configured
    if sec.config.SensitiveDataRedaction {
        redactedContent, err := sec.redactSensitiveData(evidence.Content, classification)
        if err != nil {
            return nil, fmt.Errorf("sensitive data redaction failed: %w", err)
        }
        protected.Content = redactedContent
    }

    // Encrypt sensitive fields
    if classification.RequiresEncryption() {
        encryptedFields, err := sec.cryptoService.EncryptSensitiveFields(evidence.SensitiveFields)
        if err != nil {
            return nil, fmt.Errorf("field encryption failed: %w", err)
        }
        protected.EncryptedFields = encryptedFields
    }

    return protected, nil
}
```

#### Terraform Security Scanner Implementation
```go
// internal/tools/terraform/security_scanner.go
package terraform

import (
    "context"
    "fmt"
    "path/filepath"
    "regexp"
    "strings"

    "github.com/hashicorp/hcl/v2"
    "github.com/hashicorp/hcl/v2/hclparse"
    "github.com/rs/zerolog/log"
)

// SecurityScanner analyzes Terraform configurations for security compliance
type SecurityScanner struct {
    parser     *hclparse.Parser
    rules      SecurityRuleSet
    validator  ConfigValidator
    reporter   ViolationReporter
}

// SecurityRuleSet defines security rules for different compliance frameworks
type SecurityRuleSet struct {
    SOC2Rules      []SecurityRule `yaml:"soc2_rules"`
    ISO27001Rules  []SecurityRule `yaml:"iso27001_rules"`
    CustomRules    []SecurityRule `yaml:"custom_rules"`
}

// SecurityRule defines a specific security check
type SecurityRule struct {
    ID          string              `yaml:"id"`
    Name        string              `yaml:"name"`
    Description string              `yaml:"description"`
    Severity    string              `yaml:"severity"`
    Framework   string              `yaml:"framework"`
    Controls    []string            `yaml:"controls"`
    Check       SecurityCheckFunc   `yaml:"-"`
}

// SecurityCheckFunc defines the signature for security check functions
type SecurityCheckFunc func(ctx context.Context, resource *hcl.Block) []SecurityViolation

// ScanInfrastructure performs comprehensive security analysis of Terraform configuration
func (ss *SecurityScanner) ScanInfrastructure(ctx context.Context, terraformPath string) (*SecurityReport, error) {
    log.Info().Str("path", terraformPath).Msg("Starting Terraform security scan")

    // 1. Discover and parse Terraform files
    files, err := ss.discoverTerraformFiles(terraformPath)
    if err != nil {
        return nil, fmt.Errorf("failed to discover Terraform files: %w", err)
    }

    // 2. Parse configuration files
    parsedConfigs := make([]*hcl.File, 0, len(files))
    for _, file := range files {
        config, diags := ss.parser.ParseHCLFile(file)
        if diags.HasErrors() {
            log.Warn().Str("file", file).Errs("diagnostics", diags.Errs()).Msg("Failed to parse Terraform file")
            continue
        }
        parsedConfigs = append(parsedConfigs, config)
    }

    // 3. Extract resources and data sources
    resources, err := ss.extractResources(parsedConfigs)
    if err != nil {
        return nil, fmt.Errorf("failed to extract resources: %w", err)
    }

    // 4. Apply security rules
    violations := make([]SecurityViolation, 0)
    for _, resource := range resources {
        resourceViolations := ss.checkResourceSecurity(ctx, resource)
        violations = append(violations, resourceViolations...)
    }

    // 5. Generate comprehensive security report
    report := &SecurityReport{
        ScanID:       generateScanID(),
        Timestamp:    time.Now(),
        TerraformPath: terraformPath,
        FilesScanned: len(files),
        ResourcesAnalyzed: len(resources),
        Violations:   violations,
        Summary:      ss.generateSummary(violations),
        Recommendations: ss.generateRecommendations(violations),
    }

    log.Info().
        Str("scan_id", report.ScanID).
        Int("violations", len(violations)).
        Int("critical", report.Summary.CriticalCount).
        Int("high", report.Summary.HighCount).
        Msg("Terraform security scan completed")

    return report, nil
}

// checkResourceSecurity applies all relevant security rules to a resource
func (ss *SecurityScanner) checkResourceSecurity(ctx context.Context, resource *hcl.Block) []SecurityViolation {
    violations := make([]SecurityViolation, 0)

    // Get applicable rules for this resource type
    rules := ss.getRulesForResource(resource)

    for _, rule := range rules {
        select {
        case <-ctx.Done():
            return violations
        default:
            ruleViolations := rule.Check(ctx, resource)
            violations = append(violations, ruleViolations...)
        }
    }

    return violations
}

// Example security check for S3 bucket encryption
func checkS3BucketEncryption(ctx context.Context, resource *hcl.Block) []SecurityViolation {
    violations := make([]SecurityViolation, 0)

    if resource.Type != "aws_s3_bucket" {
        return violations
    }

    // Check for server-side encryption configuration
    hasEncryption := false
    if encryptionBlock := findBlock(resource.Body, "server_side_encryption_configuration"); encryptionBlock != nil {
        if ruleBlock := findBlock(encryptionBlock.Body, "rule"); ruleBlock != nil {
            if applyBlock := findBlock(ruleBlock.Body, "apply_server_side_encryption_by_default"); applyBlock != nil {
                hasEncryption = true
            }
        }
    }

    if !hasEncryption {
        violations = append(violations, SecurityViolation{
            RuleID:      "S3-001",
            Severity:    "HIGH",
            Resource:    resource.Labels[1], // Resource name
            Description: "S3 bucket does not have server-side encryption enabled",
            Framework:   "SOC2",
            Controls:    []string{"CC6.1", "CC6.7"},
            Location: ViolationLocation{
                File:     resource.DefRange.Filename,
                StartLine: resource.DefRange.Start.Line,
                EndLine:   resource.DefRange.End.Line,
            },
            Remediation: "Add server_side_encryption_configuration block with appropriate encryption settings",
        })
    }

    return violations
}

// Example security check for IAM policy least privilege
func checkIAMPolicyLeastPrivilege(ctx context.Context, resource *hcl.Block) []SecurityViolation {
    violations := make([]SecurityViolation, 0)

    if resource.Type != "aws_iam_policy" {
        return violations
    }

    // Extract policy document
    policyAttr := findAttribute(resource.Body, "policy")
    if policyAttr == nil {
        return violations
    }

    policyDoc, err := extractPolicyDocument(policyAttr)
    if err != nil {
        return violations
    }

    // Check for overly permissive policies
    for _, statement := range policyDoc.Statement {
        if statement.Effect == "Allow" {
            // Check for wildcard resources
            for _, resourceArn := range statement.Resource {
                if resourceArn == "*" {
                    violations = append(violations, SecurityViolation{
                        RuleID:      "IAM-001",
                        Severity:    "CRITICAL",
                        Resource:    resource.Labels[1],
                        Description: "IAM policy grants access to all resources (*)",
                        Framework:   "SOC2",
                        Controls:    []string{"CC6.1", "CC6.2", "CC6.3"},
                        Location: ViolationLocation{
                            File:     resource.DefRange.Filename,
                            StartLine: resource.DefRange.Start.Line,
                            EndLine:   resource.DefRange.End.Line,
                        },
                        Remediation: "Restrict resource access to specific ARNs instead of using wildcard (*)",
                    })
                }
            }

            // Check for overly permissive actions
            for _, action := range statement.Action {
                if strings.HasSuffix(action, ":*") {
                    violations = append(violations, SecurityViolation{
                        RuleID:      "IAM-002",
                        Severity:    "HIGH",
                        Resource:    resource.Labels[1],
                        Description: fmt.Sprintf("IAM policy grants broad permissions: %s", action),
                        Framework:   "SOC2",
                        Controls:    []string{"CC6.2"},
                        Location: ViolationLocation{
                            File:     resource.DefRange.Filename,
                            StartLine: resource.DefRange.Start.Line,
                            EndLine:   resource.DefRange.End.Line,
                        },
                        Remediation: "Specify only the minimum required permissions instead of wildcard actions",
                    })
                }
            }
        }
    }

    return violations
}
```

#### GitHub Security Analysis Implementation
```go
// internal/tools/github/security_analyzer.go
package github

import (
    "context"
    "fmt"
    "net/http"
    "strings"
    "time"

    "github.com/google/go-github/v50/github"
    "golang.org/x/oauth2"
)

// SecurityAnalyzer analyzes GitHub repositories for security compliance
type SecurityAnalyzer struct {
    client      *github.Client
    rateLimiter RateLimiter
    validator   SecurityValidator
    config      AnalyzerConfig
}

// AnalyzerConfig defines configuration for the GitHub security analyzer
type AnalyzerConfig struct {
    EnableVulnerabilityScanning bool     `yaml:"enable_vulnerability_scanning"`
    EnableSecretScanning        bool     `yaml:"enable_secret_scanning"`
    RequiredProtections         []string `yaml:"required_protections"`
    AllowedPermissions          []string `yaml:"allowed_permissions"`
    MaxAdminUsers              int      `yaml:"max_admin_users"`
}

// AnalyzeRepositorySecurity performs comprehensive security analysis of a GitHub repository
func (sa *SecurityAnalyzer) AnalyzeRepositorySecurity(ctx context.Context, owner, repo string) (*SecurityAnalysis, error) {
    log.Info().Str("repo", fmt.Sprintf("%s/%s", owner, repo)).Msg("Starting GitHub security analysis")

    analysis := &SecurityAnalysis{
        Repository:  fmt.Sprintf("%s/%s", owner, repo),
        Timestamp:   time.Now(),
        Findings:    make([]SecurityFinding, 0),
    }

    // 1. Analyze repository configuration
    repoFindings, err := sa.analyzeRepositoryConfiguration(ctx, owner, repo)
    if err != nil {
        return nil, fmt.Errorf("repository configuration analysis failed: %w", err)
    }
    analysis.Findings = append(analysis.Findings, repoFindings...)

    // 2. Analyze branch protection settings
    branchFindings, err := sa.analyzeBranchProtection(ctx, owner, repo)
    if err != nil {
        return nil, fmt.Errorf("branch protection analysis failed: %w", err)
    }
    analysis.Findings = append(analysis.Findings, branchFindings...)

    // 3. Analyze access permissions
    permissionFindings, err := sa.analyzeAccessPermissions(ctx, owner, repo)
    if err != nil {
        return nil, fmt.Errorf("access permission analysis failed: %w", err)
    }
    analysis.Findings = append(analysis.Findings, permissionFindings...)

    // 4. Analyze security features
    securityFindings, err := sa.analyzeSecurityFeatures(ctx, owner, repo)
    if err != nil {
        return nil, fmt.Errorf("security features analysis failed: %w", err)
    }
    analysis.Findings = append(analysis.Findings, securityFindings...)

    // 5. Generate summary and recommendations
    analysis.Summary = sa.generateAnalysisSummary(analysis.Findings)
    analysis.Recommendations = sa.generateRecommendations(analysis.Findings)

    log.Info().
        Str("repo", fmt.Sprintf("%s/%s", owner, repo)).
        Int("findings", len(analysis.Findings)).
        Int("critical", analysis.Summary.CriticalCount).
        Msg("GitHub security analysis completed")

    return analysis, nil
}

// analyzeRepositoryConfiguration checks basic repository security settings
func (sa *SecurityAnalyzer) analyzeRepositoryConfiguration(ctx context.Context, owner, repo string) ([]SecurityFinding, error) {
    findings := make([]SecurityFinding, 0)

    // Get repository details
    repository, _, err := sa.client.Repositories.Get(ctx, owner, repo)
    if err != nil {
        return nil, fmt.Errorf("failed to get repository: %w", err)
    }

    // Check if repository is private (SOC2 requirement for sensitive code)
    if repository.GetPrivate() == false {
        findings = append(findings, SecurityFinding{
            ID:          "REPO-001",
            Severity:    "HIGH",
            Category:    "Repository Configuration",
            Title:       "Repository is public",
            Description: "Repository containing compliance-related code should be private",
            Framework:   "SOC2",
            Controls:    []string{"CC6.1", "CC6.7"},
            Remediation: "Change repository visibility to private to protect sensitive compliance code",
        })
    }

    // Check if issues are disabled for production repositories
    if repository.GetHasIssues() && strings.Contains(repo, "prod") {
        findings = append(findings, SecurityFinding{
            ID:          "REPO-002",
            Severity:    "MEDIUM",
            Category:    "Repository Configuration",
            Title:       "Issues enabled on production repository",
            Description: "Production repositories should not have public issue tracking",
            Framework:   "SOC2",
            Controls:    []string{"CC6.1"},
            Remediation: "Disable issues for production repositories to prevent information leakage",
        })
    }

    // Check for required topics/labels
    topics := repository.Topics
    hasComplianceTopic := false
    for _, topic := range topics {
        if strings.Contains(topic, "compliance") || strings.Contains(topic, "soc2") {
            hasComplianceTopic = true
            break
        }
    }

    if !hasComplianceTopic {
        findings = append(findings, SecurityFinding{
            ID:          "REPO-003",
            Severity:    "LOW",
            Category:    "Repository Configuration",
            Title:       "Missing compliance topic",
            Description: "Compliance-related repositories should be tagged with appropriate topics",
            Framework:   "Internal",
            Controls:    []string{"Documentation"},
            Remediation: "Add 'compliance' or 'soc2' topic to repository",
        })
    }

    return findings, nil
}

// analyzeBranchProtection checks branch protection rules
func (sa *SecurityAnalyzer) analyzeBranchProtection(ctx context.Context, owner, repo string) ([]SecurityFinding, error) {
    findings := make([]SecurityFinding, 0)

    // Get default branch
    repository, _, err := sa.client.Repositories.Get(ctx, owner, repo)
    if err != nil {
        return nil, fmt.Errorf("failed to get repository: %w", err)
    }

    defaultBranch := repository.GetDefaultBranch()

    // Get branch protection
    protection, resp, err := sa.client.Repositories.GetBranchProtection(ctx, owner, repo, defaultBranch)
    if err != nil {
        if resp.StatusCode == http.StatusNotFound {
            findings = append(findings, SecurityFinding{
                ID:          "BRANCH-001",
                Severity:    "CRITICAL",
                Category:    "Branch Protection",
                Title:       "No branch protection on default branch",
                Description: "Default branch lacks protection rules",
                Framework:   "SOC2",
                Controls:    []string{"CC8.1", "CC3.2"},
                Remediation: "Enable branch protection with required status checks and review requirements",
            })
            return findings, nil
        }
        return nil, fmt.Errorf("failed to get branch protection: %w", err)
    }

    // Check required status checks
    if protection.RequiredStatusChecks == nil || len(protection.RequiredStatusChecks.Contexts) == 0 {
        findings = append(findings, SecurityFinding{
            ID:          "BRANCH-002",
            Severity:    "HIGH",
            Category:    "Branch Protection",
            Title:       "No required status checks",
            Description: "Branch protection should require status checks to pass",
            Framework:   "SOC2",
            Controls:    []string{"CC8.1"},
            Remediation: "Configure required status checks including CI/CD pipeline and security scans",
        })
    }

    // Check pull request reviews
    if protection.RequiredPullRequestReviews == nil {
        findings = append(findings, SecurityFinding{
            ID:          "BRANCH-003",
            Severity:    "HIGH",
            Category:    "Branch Protection",
            Title:       "No required pull request reviews",
            Description: "Branch protection should require pull request reviews",
            Framework:   "SOC2",
            Controls:    []string{"CC8.1", "CC3.2"},
            Remediation: "Enable required pull request reviews with at least 1 reviewer",
        })
    } else {
        // Check number of required reviewers
        if protection.RequiredPullRequestReviews.RequiredApprovingReviewCount < 1 {
            findings = append(findings, SecurityFinding{
                ID:          "BRANCH-004",
                Severity:    "MEDIUM",
                Category:    "Branch Protection",
                Title:       "Insufficient required reviewers",
                Description: "At least one reviewer should be required for pull requests",
                Framework:   "SOC2",
                Controls:    []string{"CC8.1"},
                Remediation: "Set required approving review count to at least 1",
            })
        }

        // Check if admins are included in restrictions
        if !protection.RequiredPullRequestReviews.DismissStaleReviews {
            findings = append(findings, SecurityFinding{
                ID:          "BRANCH-005",
                Severity:    "MEDIUM",
                Category:    "Branch Protection",
                Title:       "Stale reviews not dismissed",
                Description: "Stale reviews should be dismissed when new commits are pushed",
                Framework:   "SOC2",
                Controls:    []string{"CC8.1"},
                Remediation: "Enable dismissal of stale reviews when new commits are pushed",
            })
        }
    }

    return findings, nil
}
```

### Secure Credential Management

#### Credential Manager Implementation
```go
// internal/auth/credential_manager.go
package auth

import (
    "context"
    "crypto/aes"
    "crypto/cipher"
    "crypto/rand"
    "encoding/base64"
    "fmt"
    "io"
    "os"
    "path/filepath"
    "sync"
    "time"

    "github.com/rs/zerolog/log"
)

// SecureCredentialManager handles secure storage and retrieval of credentials
type SecureCredentialManager struct {
    mu           sync.RWMutex
    credentials  map[string]*EncryptedCredential
    masterKey    []byte
    storagePath  string
    rotationSchedule map[string]time.Duration
}

// EncryptedCredential represents a securely stored credential
type EncryptedCredential struct {
    ID           string    `json:"id"`
    Service      string    `json:"service"`
    EncryptedData []byte   `json:"encrypted_data"`
    Nonce        []byte    `json:"nonce"`
    CreatedAt    time.Time `json:"created_at"`
    ExpiresAt    time.Time `json:"expires_at,omitempty"`
    LastUsed     time.Time `json:"last_used"`
    Usage        int       `json:"usage_count"`
}

// NewSecureCredentialManager creates a new credential manager with encryption
func NewSecureCredentialManager(storagePath string) (*SecureCredentialManager, error) {
    // Generate or load master key
    masterKey, err := loadOrGenerateMasterKey(storagePath)
    if err != nil {
        return nil, fmt.Errorf("failed to initialize master key: %w", err)
    }

    manager := &SecureCredentialManager{
        credentials:      make(map[string]*EncryptedCredential),
        masterKey:        masterKey,
        storagePath:      storagePath,
        rotationSchedule: map[string]time.Duration{
            "github":  30 * 24 * time.Hour, // 30 days
            "tugboat": 7 * 24 * time.Hour,   // 7 days
            "claude":  90 * 24 * time.Hour,  // 90 days
        },
    }

    // Load existing credentials
    if err := manager.loadCredentials(); err != nil {
        log.Warn().Err(err).Msg("Failed to load existing credentials")
    }

    return manager, nil
}

// StoreCredential securely stores a credential with encryption
func (scm *SecureCredentialManager) StoreCredential(ctx context.Context, service, credentialData string) error {
    scm.mu.Lock()
    defer scm.mu.Unlock()

    // Encrypt the credential data
    encryptedData, nonce, err := scm.encrypt([]byte(credentialData))
    if err != nil {
        return fmt.Errorf("failed to encrypt credential: %w", err)
    }

    // Create credential record
    credential := &EncryptedCredential{
        ID:           generateCredentialID(),
        Service:      service,
        EncryptedData: encryptedData,
        Nonce:        nonce,
        CreatedAt:    time.Now(),
        LastUsed:     time.Now(),
        Usage:        0,
    }

    // Set expiration based on service policy
    if rotation, exists := scm.rotationSchedule[service]; exists {
        credential.ExpiresAt = time.Now().Add(rotation)
    }

    // Store in memory
    scm.credentials[service] = credential

    // Persist to disk
    if err := scm.persistCredentials(); err != nil {
        return fmt.Errorf("failed to persist credentials: %w", err)
    }

    log.Info().Str("service", service).Msg("Credential stored securely")
    return nil
}

// GetCredential retrieves and decrypts a credential
func (scm *SecureCredentialManager) GetCredential(ctx context.Context, service string) (string, error) {
    scm.mu.RLock()
    credential, exists := scm.credentials[service]
    scm.mu.RUnlock()

    if !exists {
        return "", fmt.Errorf("credential not found for service: %s", service)
    }

    // Check if credential has expired
    if !credential.ExpiresAt.IsZero() && time.Now().After(credential.ExpiresAt) {
        return "", fmt.Errorf("credential for service %s has expired", service)
    }

    // Decrypt the credential
    decryptedData, err := scm.decrypt(credential.EncryptedData, credential.Nonce)
    if err != nil {
        return "", fmt.Errorf("failed to decrypt credential: %w", err)
    }

    // Update usage statistics
    scm.mu.Lock()
    credential.LastUsed = time.Now()
    credential.Usage++
    scm.mu.Unlock()

    // Asynchronously persist usage update
    go func() {
        if err := scm.persistCredentials(); err != nil {
            log.Error().Err(err).Msg("Failed to persist credential usage update")
        }
    }()

    return string(decryptedData), nil
}

// encrypt encrypts data using AES-GCM
func (scm *SecureCredentialManager) encrypt(data []byte) ([]byte, []byte, error) {
    block, err := aes.NewCipher(scm.masterKey)
    if err != nil {
        return nil, nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, nil, err
    }

    nonce := make([]byte, gcm.NonceSize())
    if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
        return nil, nil, err
    }

    ciphertext := gcm.Seal(nil, nonce, data, nil)
    return ciphertext, nonce, nil
}

// decrypt decrypts data using AES-GCM
func (scm *SecureCredentialManager) decrypt(ciphertext, nonce []byte) ([]byte, error) {
    block, err := aes.NewCipher(scm.masterKey)
    if err != nil {
        return nil, err
    }

    gcm, err := cipher.NewGCM(block)
    if err != nil {
        return nil, err
    }

    plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
    if err != nil {
        return nil, err
    }

    return plaintext, nil
}

// RotateCredentials rotates expired credentials
func (scm *SecureCredentialManager) RotateCredentials(ctx context.Context) error {
    scm.mu.Lock()
    defer scm.mu.Unlock()

    now := time.Now()
    rotatedCount := 0

    for service, credential := range scm.credentials {
        if !credential.ExpiresAt.IsZero() && now.After(credential.ExpiresAt) {
            log.Info().Str("service", service).Msg("Credential expired, requiring rotation")

            // Remove expired credential
            delete(scm.credentials, service)
            rotatedCount++

            // Trigger credential refresh (implementation depends on service)
            if err := scm.triggerCredentialRefresh(ctx, service); err != nil {
                log.Error().Err(err).Str("service", service).Msg("Failed to trigger credential refresh")
            }
        }
    }

    if rotatedCount > 0 {
        if err := scm.persistCredentials(); err != nil {
            return fmt.Errorf("failed to persist credential rotation: %w", err)
        }
        log.Info().Int("count", rotatedCount).Msg("Credentials rotated")
    }

    return nil
}
```

### Compliance-Driven Code Examples

#### SOC 2 Evidence Generation Pipeline
```go
// internal/evidence/soc2_pipeline.go
package evidence

import (
    "context"
    "fmt"
    "time"

    "github.com/rs/zerolog/log"
)

// SOC2EvidencePipeline orchestrates evidence generation for SOC 2 compliance
type SOC2EvidencePipeline struct {
    collectors     map[string]EvidenceCollector
    validator      SOC2Validator
    auditTrail     AuditTrail
    qualityChecker QualityChecker
}

// GenerateComplianceEvidence generates comprehensive evidence for SOC 2 audit
func (sep *SOC2EvidencePipeline) GenerateComplianceEvidence(ctx context.Context, auditPeriod AuditPeriod) (*ComplianceEvidencePackage, error) {
    log.Info().
        Str("start_date", auditPeriod.StartDate.Format("2006-01-02")).
        Str("end_date", auditPeriod.EndDate.Format("2006-01-02")).
        Msg("Starting SOC 2 evidence generation")

    // Create audit trail entry
    auditID := sep.auditTrail.StartAuditEvidenceGeneration(auditPeriod)
    defer sep.auditTrail.EndAuditEvidenceGeneration(auditID)

    package := &ComplianceEvidencePackage{
        Framework:    "SOC2",
        AuditPeriod:  auditPeriod,
        GeneratedAt:  time.Now(),
        Evidence:     make(map[string]*Evidence),
        Summary:      &EvidenceSummary{},
    }

    // SOC 2 Trust Services Criteria evidence collection
    trustCriteria := []string{"security", "availability", "processing_integrity", "confidentiality", "privacy"}

    for _, criteria := range trustCriteria {
        log.Info().Str("criteria", criteria).Msg("Collecting evidence for trust criteria")

        criteriaEvidence, err := sep.collectCriteriaEvidence(ctx, criteria, auditPeriod)
        if err != nil {
            sep.auditTrail.LogError(auditID, fmt.Errorf("failed to collect %s evidence: %w", criteria, err))
            continue
        }

        // Quality check collected evidence
        qualityScore, issues := sep.qualityChecker.AssessEvidence(criteriaEvidence)
        if qualityScore < 0.8 {
            log.Warn().
                Str("criteria", criteria).
                Float64("quality_score", qualityScore).
                Int("issues", len(issues)).
                Msg("Evidence quality below threshold")
        }

        package.Evidence[criteria] = criteriaEvidence
        package.Summary.TotalEvidence++
        package.Summary.QualityScores[criteria] = qualityScore
    }

    // Generate executive summary
    package.ExecutiveSummary = sep.generateExecutiveSummary(package)

    // Validate complete package
    if err := sep.validator.ValidateCompliancePackage(package); err != nil {
        return nil, fmt.Errorf("compliance package validation failed: %w", err)
    }

    log.Info().
        Int("evidence_count", package.Summary.TotalEvidence).
        Float64("avg_quality", package.Summary.AverageQuality()).
        Msg("SOC 2 evidence generation completed")

    return package, nil
}

// collectCriteriaEvidence collects evidence for a specific trust criteria
func (sep *SOC2EvidencePipeline) collectCriteriaEvidence(ctx context.Context, criteria string, period AuditPeriod) (*Evidence, error) {
    evidence := &Evidence{
        ID:          generateEvidenceID("SOC2", criteria),
        Framework:   "SOC2",
        Criteria:    criteria,
        Period:      period,
        Sections:    make(map[string]*EvidenceSection),
        Artifacts:   make([]*EvidenceArtifact, 0),
        CollectedAt: time.Now(),
    }

    // Map criteria to specific controls and evidence requirements
    switch criteria {
    case "security":
        return sep.collectSecurityEvidence(ctx, evidence)
    case "availability":
        return sep.collectAvailabilityEvidence(ctx, evidence)
    case "processing_integrity":
        return sep.collectProcessingIntegrityEvidence(ctx, evidence)
    case "confidentiality":
        return sep.collectConfidentialityEvidence(ctx, evidence)
    case "privacy":
        return sep.collectPrivacyEvidence(ctx, evidence)
    default:
        return nil, fmt.Errorf("unknown trust criteria: %s", criteria)
    }
}

// collectSecurityEvidence implements CC (Common Criteria) evidence collection
func (sep *SOC2EvidencePipeline) collectSecurityEvidence(ctx context.Context, evidence *Evidence) (*Evidence, error) {
    // CC6.1 - Logical and Physical Access Controls
    accessControlSection, err := sep.collectAccessControlEvidence(ctx)
    if err != nil {
        return nil, fmt.Errorf("access control evidence collection failed: %w", err)
    }
    evidence.Sections["CC6.1"] = accessControlSection

    // CC6.2 - Logical and Physical Access Controls - Prior to Entity's Approval
    approvalSection, err := sep.collectAccessApprovalEvidence(ctx)
    if err != nil {
        return nil, fmt.Errorf("access approval evidence collection failed: %w", err)
    }
    evidence.Sections["CC6.2"] = approvalSection

    // CC6.3 - Network Security
    networkSection, err := sep.collectNetworkSecurityEvidence(ctx)
    if err != nil {
        return nil, fmt.Errorf("network security evidence collection failed: %w", err)
    }
    evidence.Sections["CC6.3"] = networkSection

    // CC6.6 - Vulnerability Management
    vulnerabilitySection, err := sep.collectVulnerabilityManagementEvidence(ctx)
    if err != nil {
        return nil, fmt.Errorf("vulnerability management evidence collection failed: %w", err)
    }
    evidence.Sections["CC6.6"] = vulnerabilitySection

    // CC6.7 - Data Transmission and Disposal
    dataProtectionSection, err := sep.collectDataProtectionEvidence(ctx)
    if err != nil {
        return nil, fmt.Errorf("data protection evidence collection failed: %w", err)
    }
    evidence.Sections["CC6.7"] = dataProtectionSection

    // CC6.8 - Entity Information Systems
    systemsSection, err := sep.collectSystemsEvidence(ctx)
    if err != nil {
        return nil, fmt.Errorf("systems evidence collection failed: %w", err)
    }
    evidence.Sections["CC6.8"] = systemsSection

    return evidence, nil
}
```

## References

- [[security-practices]] - Detailed security implementation guidelines
- [[testing-strategy]] - Comprehensive testing approaches
- [[contribution-guide]] - Detailed contribution workflow
- [[api-design]] - API design patterns and standards
- [[evidence-collection-patterns]] - Evidence collection implementation examples
- [[secure-coding-standards]] - Security-focused coding practices
- [[compliance-automation-guide]] - Automated compliance implementation patterns

---

*These development practices and secure code examples ensure GRCTool maintains the highest standards for security, quality, and maintainability while providing concrete implementation patterns for compliance automation and evidence collection.*