---
title: "Comprehensive Testing Strategy"
phase: "03-test"
category: "testing"
tags: ["testing", "quality", "coverage", "mutation", "benchmarks", "vcr", "security"]
related: ["security-testing", "performance-testing", "test-automation"]
created: 2025-01-10
updated: 2025-01-10
helix_mapping: "Consolidated from 04-Development/testing-guide.md"
---

# Comprehensive Testing Strategy

## Overview

GRCTool employs a comprehensive 4-tier testing strategy with advanced quality assurance tools including coverage analysis, performance benchmarking, mutation testing, and security validation. This strategy ensures reliability, security, and performance for compliance-critical operations.

### Testing Philosophy
- **Test-Driven Development**: Write tests before or alongside code
- **Quality Gates**: Automated quality enforcement before merge
- **Continuous Improvement**: Regular review and enhancement of test suite
- **Realistic Testing**: Use production-like data and scenarios
- **Security-First**: Security testing integrated throughout the development cycle

## Testing Tiers

### 1. Unit Tests (2-3 seconds)
Fast, isolated tests with no external dependencies.

**Characteristics:**
- No network calls
- No file system dependencies
- Mocked external services
- Focus on single functions/methods

**Commands:**
```bash
make test-unit
go test ./internal/... -short
```

**Build Tags:** `//go:build !e2e && !integration`

**Coverage Goals:**
- **Overall Target**: ≥80%
- **Critical Packages**: ≥90%
- **Standard Packages**: ≥70%

### 2. Integration Tests (2 minutes)
Tests with VCR recordings for deterministic API testing.

**Characteristics:**
- Use VCR cassettes for API interactions
- Test component interactions
- Cross-package validation
- Realistic data flows

**Commands:**
```bash
make test-integration
VCR_MODE=playback go test ./test/integration/...
```

**Build Tags:** `//go:build !e2e`

### 3. Functional Tests (5 minutes)
CLI testing with built binary.

**Characteristics:**
- Test complete CLI workflows
- End-to-end command execution
- File system operations
- Configuration loading

**Commands:**
```bash
make test-functional
go test -tags functional ./test/functional/...
```

**Build Tags:** `//go:build functional`

### 4. End-to-End Tests (10+ minutes)
Real API testing (requires authentication).

**Characteristics:**
- Live API interactions
- Full authentication flows
- Network dependencies
- Real service integration

**Commands:**
```bash
make test-e2e
GITHUB_TOKEN=xyz TUGBOAT_BASE_URL=... go test -tags e2e ./test/e2e/...
```

**Build Tags:** `//go:build e2e`

## Test Organization

### Directory Structure
```
test/
├── helpers/              # Shared test utilities
│   ├── builders.go      # Test data builders
│   ├── golden_file.go   # Golden file testing
│   └── vcr_helper.go    # VCR test helpers
├── integration/         # Cross-package integration tests
├── functional/          # CLI functionality tests
├── e2e/                # End-to-end tests
└── testdata/           # Test fixtures and golden files

internal/
├── auth/
│   ├── auth.go
│   └── auth_test.go    # Unit tests alongside code
├── tools/
│   ├── github_test.go
│   └── terraform_integration_test.go
```

### File Naming Conventions
- **Unit tests**: `*_test.go` in same package
- **Integration tests**: `*_integration_test.go` with appropriate build tags
- **Functional tests**: `*_functional_test.go` with `//go:build functional`
- **E2E tests**: `*_e2e_test.go` with `//go:build e2e`

## VCR (Video Cassette Recorder) Testing

VCR enables deterministic testing of external API interactions by recording and replaying HTTP responses.

### Key Features
- **Deterministic Tests**: Identical responses across test runs
- **Offline Testing**: No external dependencies for CI
- **Security**: Automatic credential redaction
- **Fast Execution**: Eliminate network latency

### Usage Example
```go
func TestGitHubAPI(t *testing.T) {
    // Setup VCR with cassette
    vcr := helpers.SetupVCR(t, "github_api_test")
    defer vcr.Stop()

    client := github.NewClient(vcr.GetHTTPClient())

    // Test API interaction - recorded/replayed automatically
    repos, err := client.ListRepositories("testorg")
    assert.NoError(t, err)
    assert.NotEmpty(t, repos)
}
```

### VCR Modes
- `off`: No recording/playback
- `record`: Record new interactions
- `record_once`: Record only if cassette doesn't exist
- `playback`: Use existing recordings (CI default)

### Cassette Management
```bash
# Record new cassettes
VCR_MODE=record go test ./test/integration/...

# Update existing cassettes
VCR_MODE=record_once go test ./test/integration/...

# Use recorded interactions (default)
VCR_MODE=playback go test ./test/integration/...
```

## Testing Infrastructure

### Test Helpers and Builders

#### Golden File Testing
Compare complex outputs against saved "golden" files:

```go
func TestEvidenceGeneration(t *testing.T) {
    golden := helpers.NewGoldenFile(t)

    evidence := GenerateEvidence(testTask)

    golden.Assert("evidence_output.md", []byte(evidence.Content))
}

// Update golden files
UPDATE_GOLDEN=true go test ./...
```

#### Test Data Builders
Create complex test objects with builder pattern:

```go
func TestEvidenceProcessing(t *testing.T) {
    task := helpers.NewEvidenceTaskBuilder().
        WithName("Access Control Review").
        WithCollectionInterval("monthly").
        WithSensitive(true).
        Build()

    assignee := helpers.NewEvidenceAssigneeBuilder().
        WithName("Jane Doe").
        WithRole("security_analyst").
        Build()

    taskDetails := helpers.NewEvidenceTaskDetailsBuilder().
        WithEvidenceTask(*task).
        WithAssignees([]models.EvidenceAssignee{*assignee}).
        Build()

    result := ProcessEvidence(taskDetails)
    assert.NotNil(t, result)
}
```

#### Available Builders
- `helpers.NewEvidenceTaskBuilder()` - Evidence tasks
- `helpers.NewPolicyBuilder()` - Policies
- `helpers.NewControlBuilder()` - Controls
- `helpers.NewEvidenceAssigneeBuilder()` - Assignees
- `helpers.NewEvidenceTagBuilder()` - Tags

## Testing Style Guide

### Table-Driven Tests (Preferred)

Use maps for better readability and easier identification of failing tests:

```go
func TestIsHigherPermission(t *testing.T) {
    tests := map[string]struct {
        perm1    string
        perm2    string
        expected bool
    }{
        "admin higher than push": {
            perm1:    "admin",
            perm2:    "push",
            expected: true,
        },
        "push not higher than admin": {
            perm1:    "push",
            perm2:    "admin",
            expected: false,
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            result := IsHigherPermission(tt.perm1, tt.perm2)
            assert.Equal(t, tt.expected, result)
        })
    }
}
```

### Test Isolation and Cleanup

```go
func TestWithTempFile(t *testing.T) {
    t.Parallel() // Safe when no shared state

    tmpFile, err := os.CreateTemp("", "test-*.txt")
    require.NoError(t, err)

    t.Cleanup(func() {
        os.Remove(tmpFile.Name())
    })

    // Use tmpFile in test
}

func TestWithTempDir(t *testing.T) {
    tmpDir := t.TempDir() // Automatically cleaned up

    // Use tmpDir in test
}
```

### Error Testing Best Practices

```go
func TestErrorConditions(t *testing.T) {
    tests := map[string]struct {
        input       string
        expectError bool
        errorMsg    string
    }{
        "empty input returns error": {
            input:       "",
            expectError: true,
            errorMsg:    "input cannot be empty",
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            result, err := ProcessInput(tt.input)

            if tt.expectError {
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.errorMsg)
                assert.Nil(t, result)
            } else {
                assert.NoError(t, err)
                assert.NotNil(t, result)
            }
        })
    }
}
```

## Code Coverage

### Coverage Goals and Current Status

| Coverage Level | Target | Current | Status |
|----------------|--------|---------|--------|
| Overall | ≥80% | 22.4% | ❌ |
| Critical Packages | ≥90% | Varies | ❌ |
| Test Execution | <5s | 3s | ✅ |

### Coverage Commands
```bash
# Generate HTML coverage report
make coverage-report

# Check if coverage meets 80% threshold
make coverage-check

# Generate coverage badge
make coverage-badge

# Check critical packages only
make coverage-critical

# Monitor all packages
make coverage-monitor
```

### Critical Packages Needing Coverage
- `internal/tugboat` - 0% (Core API client)
- `internal/storage` - 0% (Data persistence)
- `internal/services/evidence` - 0% (Evidence generation)
- `internal/services/config` - 0% (Configuration)
- `internal/services/validation` - 0% (Data validation)

## Performance Benchmarking

### Running Benchmarks
```bash
# Run all benchmarks
make bench

# Compare with baseline
make bench-compare

# Generate CPU/memory profiles
make bench-profile

# Save as new baseline
make bench-save
```

### Key Benchmarks

| Operation | Target | Current | Status |
|-----------|--------|---------|--------|
| Tugboat Sync | <200ms | 145ms | ✅ |
| Evidence Generation | <100ms | 25ms | ✅ |
| Auth Validation | <5ms | 1.2ms | ✅ |
| Large Dataset Processing | <1s | 850ms | ✅ |

### Benchmark Categories

#### 1. Tugboat Client Benchmarks
- **BenchmarkTugboatClient_Sync**: Full synchronization operation
- **BenchmarkTugboatClient_FetchEvidenceTasks**: Evidence task retrieval
- **BenchmarkTugboatClient_FetchPolicies**: Policy data fetching
- **BenchmarkTugboatClient_ConcurrentRequests**: Concurrent operations

#### 2. Evidence Generator Benchmarks
- **BenchmarkEvidenceGenerator_Process**: Complete evidence generation
- **BenchmarkEvidenceGenerator_LargeDataset**: Performance with large inputs
- **BenchmarkEvidenceGenerator_CoordinateSubTools**: Tool coordination
- **BenchmarkEvidenceGenerator_JSONSerialization**: Output formatting

#### 3. Authentication Benchmarks
- **BenchmarkGitHubAuth_Validate**: GitHub token validation
- **BenchmarkTugboatAuth_GetToken**: Tugboat authentication
- **BenchmarkAuth_CacheOperations**: Authentication caching
- **BenchmarkAuth_ConcurrentValidation**: Concurrent auth operations

#### 4. Memory Benchmarks
- **BenchmarkLargeFileProcessing**: Streaming vs in-memory
- **BenchmarkConcurrentOperations**: Concurrent storage operations
- **BenchmarkCachePerformance**: Cache hit/miss performance
- **BenchmarkMemoryAllocation**: Various allocation patterns

## Mutation Testing

### What is Mutation Testing?
Mutation testing verifies test quality by introducing small changes (mutations) to code and checking if tests fail.

### Key Concepts
- **Mutant**: Code with artificial change
- **Killed Mutant**: Mutant that causes test failure (good)
- **Surviving Mutant**: Mutant that passes tests (indicates missing coverage)
- **Mutation Score**: Percentage of mutants killed (killed/total × 100)

### Running Mutation Tests
```bash
# Full mutation testing
make mutation-test

# Quick testing (critical packages)
make mutation-quick

# Dry run (fast analysis)
make mutation-dry-run

# Generate report
make mutation-report
```

### Mutation Score Targets
- **Critical Packages**: ≥80% efficacy, ≥85% coverage
- **Standard Packages**: ≥70% efficacy, ≥80% coverage
- **Utility Packages**: ≥60% efficacy, ≥70% coverage

### Common Mutation Operators

#### Arithmetic Operators
```go
// Original: a + b
// Mutated:  a - b, a * b, a / b, a % b
```

#### Comparison Operators
```go
// Original: a == b
// Mutated:  a != b, a < b, a > b, a <= b, a >= b
```

#### Boolean Logic
```go
// Original: a && b
// Mutated:  a || b

// Original: !condition
// Mutated:  condition
```

#### Boundary Conditions
```go
// Original: i < len(slice)
// Mutated:  i <= len(slice)

// Original: count > 0
// Mutated:  count >= 0
```

## Integration Testing Patterns

### Evidence Collection Integration
```go
func TestEvidenceCollectionWorkflow(t *testing.T) {
    // Setup VCR for API recording/playback
    vcr := helpers.SetupIntegrationVCR(t, "evidence_collection")
    defer vcr.Stop()

    // Test complete workflow
    tests := map[string]struct {
        taskRef     string
        expectedControls []string
        expectedEvidence int
    }{
        "ET96 user access control": {
            taskRef:          "ET-96",
            expectedControls: []string{"CC6.1", "CC6.3"},
            expectedEvidence: 3,
        },
        "ET103 multi-AZ infrastructure": {
            taskRef:          "ET-103",
            expectedControls: []string{"CC6.8", "A1.2"},
            expectedEvidence: 5,
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            // Execute evidence collection
            result, err := CollectEvidence(tt.taskRef)
            require.NoError(t, err)

            // Validate results
            assert.Equal(t, tt.expectedControls, result.Controls)
            assert.Len(t, result.Evidence, tt.expectedEvidence)
        })
    }
}
```

### Cross-Tool Validation
```go
func TestCrossToolEvidenceCorrelation(t *testing.T) {
    // Test evidence consistency between different tools
    taskRef := "ET-101"

    // Collect evidence from multiple tools
    terraformEvidence, err := CollectTerraformEvidence(taskRef)
    require.NoError(t, err)

    githubEvidence, err := CollectGitHubEvidence(taskRef)
    require.NoError(t, err)

    // Validate correlation
    assert.True(t, hasCommonSecurityThemes(terraformEvidence, githubEvidence))
    assert.True(t, evidenceSupportsControl(terraformEvidence, "CC6.1"))
    assert.True(t, evidenceSupportsControl(githubEvidence, "CC3.2"))
}
```

## Quality Metrics Dashboard

### Current Metrics

| Metric | Target | Current | Status |
|--------|--------|---------|--------|
| Code Coverage | ≥80% | 22.4% | ❌ |
| Mutation Score | ≥70% | 59.7% | ⚠️ |
| Test Execution | <5s | 3s | ✅ |
| Benchmarks | >30 | 47 | ✅ |

### Generating Quality Reports
```bash
# Comprehensive quality report
./scripts/quality-report.sh

# Coverage-specific monitoring
make coverage-monitor

# Performance analysis
make bench-compare

# Mutation testing analysis
make mutation-dry-run
```

## Troubleshooting

### Common Issues

#### VCR Cassette Errors
```bash
# Record new cassettes
VCR_MODE=record go test ./test/integration/...

# Validate cassette structure
go test ./test/integration/vcr_validation_test.go
```

#### Coverage Below Threshold
```bash
# Identify packages needing work
make coverage-monitor

# Check specific package coverage
go test -coverprofile=temp.out ./internal/tugboat/...
go tool cover -func=temp.out
```

#### Benchmark Regressions
```bash
# Compare with baseline
make bench-compare

# Analyze specific benchmark
go test -bench=BenchmarkSlowOperation -benchmem ./internal/package
```

#### Mutation Test Failures
```bash
# Analyze without running mutations
make mutation-dry-run

# Focus on specific package
go-mutesting --verbose ./internal/auth/...
```

## Best Practices Summary

### Writing Tests
1. **Use table-driven tests** with descriptive names
2. **Test error conditions** and edge cases
3. **Ensure test isolation** with proper cleanup
4. **Use test helpers** and builders for complex objects
5. **Follow naming conventions** for test files and functions

### Test Organization
1. **Separate by test type** using build tags
2. **Group related tests** in appropriate directories
3. **Use realistic test data** with proper fixtures
4. **Document test requirements** and dependencies

### Quality Assurance
1. **Maintain high coverage** (80%+ overall, 90%+ critical)
2. **Monitor mutation scores** (70%+ target)
3. **Track performance** with regular benchmarking
4. **Automate quality gates** in CI/CD pipeline

### Continuous Improvement
1. **Review and update tests** regularly
2. **Refactor test code** to reduce duplication
3. **Add tests for new features** before implementation
4. **Learn from test failures** and improve test design

## Evidence Collection Test Cases

### SOC 2 Compliance Test Scenarios

#### Test Case: ET-0001 Access Control Configuration

**Scenario**: Validate that infrastructure access controls meet SOC 2 CC6.1 requirements

**Test Implementation**:
```go
func TestET0001_AccessControlConfiguration(t *testing.T) {
    tests := map[string]struct {
        terraformPath   string
        expectedFindings map[string]int
        complianceLevel string
    }{
        "production environment with proper RBAC": {
            terraformPath: "testdata/terraform/production-rbac",
            expectedFindings: map[string]int{
                "iam_policies":           5,
                "role_assignments":       12,
                "mfa_enforcement":        1,
                "privileged_access":      3,
            },
            complianceLevel: "COMPLIANT",
        },
        "development environment with relaxed controls": {
            terraformPath: "testdata/terraform/dev-environment",
            expectedFindings: map[string]int{
                "iam_policies":           3,
                "role_assignments":       8,
                "mfa_enforcement":        0,
                "privileged_access":      1,
            },
            complianceLevel: "NON_COMPLIANT",
        },
        "legacy environment requiring remediation": {
            terraformPath: "testdata/terraform/legacy-system",
            expectedFindings: map[string]int{
                "iam_policies":           1,
                "role_assignments":       15,
                "mfa_enforcement":        0,
                "privileged_access":      8,
            },
            complianceLevel: "REQUIRES_ATTENTION",
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            // Setup VCR for reproducible testing
            vcr := helpers.SetupVCR(t, "et0001_access_control")
            defer vcr.Stop()

            // Execute evidence collection
            collector := terraform.NewAccessControlCollector(vcr.GetHTTPClient())
            evidence, err := collector.Collect(context.Background(), tt.terraformPath)
            require.NoError(t, err)

            // Validate findings
            for category, expectedCount := range tt.expectedFindings {
                actualCount := evidence.GetFindingCount(category)
                assert.Equal(t, expectedCount, actualCount,
                    "Unexpected finding count for %s", category)
            }

            // Validate compliance assessment
            assert.Equal(t, tt.complianceLevel, evidence.ComplianceLevel)

            // Validate evidence structure
            assert.NotEmpty(t, evidence.Summary)
            assert.NotEmpty(t, evidence.Details)
            assert.True(t, evidence.CollectedAt.After(time.Now().Add(-1*time.Minute)))
        })
    }
}
```

**Test Data Structure**:
```yaml
# testdata/terraform/production-rbac/main.tf
resource "aws_iam_role" "application_role" {
  name = "application-execution-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
        Condition = {
          Bool = {
            "aws:MultiFactorAuthPresent" = "true"
          }
        }
      }
    ]
  })
}

resource "aws_iam_policy" "least_privilege_policy" {
  name = "application-least-privilege"

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:GetObject",
          "s3:PutObject"
        ]
        Resource = "arn:aws:s3:::production-bucket/*"
      }
    ]
  })
}
```

**Expected Evidence Output**:
```json
{
  "evidence_id": "ET-0001-20250122-143052",
  "task_reference": "ET-0001",
  "control_mappings": ["CC6.1", "CC6.2", "CC6.3"],
  "compliance_level": "COMPLIANT",
  "summary": "Access control configuration analysis completed",
  "findings": {
    "iam_policies": {
      "count": 5,
      "details": [
        {
          "policy_name": "application-least-privilege",
          "principle": "least_privilege",
          "compliant": true
        }
      ]
    },
    "mfa_enforcement": {
      "count": 1,
      "details": [
        {
          "resource": "aws_iam_role.application_role",
          "mfa_required": true,
          "compliant": true
        }
      ]
    }
  },
  "evidence_metadata": {
    "collection_method": "automated",
    "tool_version": "terraform-scanner-v1.2.0",
    "collected_at": "2025-01-22T14:30:52Z",
    "evidence_hash": "sha256:a1b2c3..."
  }
}
```

#### Test Case: ET-0015 Encryption Implementation

**Scenario**: Verify encryption controls for SOC 2 CC6.1 data protection requirements

**Test Implementation**:
```go
func TestET0015_EncryptionImplementation(t *testing.T) {
    tests := map[string]struct {
        infraConfig     InfraConfig
        expectedResults EncryptionResults
        shouldPass      bool
    }{
        "all encryption controls implemented": {
            infraConfig: InfraConfig{
                Databases: []Database{
                    {Name: "prod-db", EncryptionAtRest: true, EncryptionInTransit: true},
                },
                Storage: []StorageBucket{
                    {Name: "prod-bucket", Encryption: "AES256", KMSManaged: true},
                },
                LoadBalancers: []LoadBalancer{
                    {Name: "prod-lb", TLSVersion: "1.3", CipherSuites: ["ECDHE-RSA-AES256-GCM-SHA384"]},
                },
            },
            expectedResults: EncryptionResults{
                OverallCompliance: 100.0,
                AtRestCompliance:  100.0,
                InTransitCompliance: 100.0,
                KeyManagementCompliance: 100.0,
            },
            shouldPass: true,
        },
        "partial encryption implementation": {
            infraConfig: InfraConfig{
                Databases: []Database{
                    {Name: "dev-db", EncryptionAtRest: false, EncryptionInTransit: true},
                },
                Storage: []StorageBucket{
                    {Name: "dev-bucket", Encryption: "None", KMSManaged: false},
                },
                LoadBalancers: []LoadBalancer{
                    {Name: "dev-lb", TLSVersion: "1.2", CipherSuites: ["TLS_RSA_WITH_AES_128_CBC_SHA"]},
                },
            },
            expectedResults: EncryptionResults{
                OverallCompliance: 33.3,
                AtRestCompliance:  0.0,
                InTransitCompliance: 66.7,
                KeyManagementCompliance: 0.0,
            },
            shouldPass: false,
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            // Setup test environment
            collector := terraform.NewEncryptionCollector()

            // Execute analysis
            results, err := collector.AnalyzeEncryption(context.Background(), tt.infraConfig)
            require.NoError(t, err)

            // Validate compliance percentages
            assert.InDelta(t, tt.expectedResults.OverallCompliance, results.OverallCompliance, 0.1)
            assert.InDelta(t, tt.expectedResults.AtRestCompliance, results.AtRestCompliance, 0.1)
            assert.InDelta(t, tt.expectedResults.InTransitCompliance, results.InTransitCompliance, 0.1)

            // Validate pass/fail status
            assert.Equal(t, tt.shouldPass, results.OverallCompliance >= 80.0)

            // Validate evidence quality
            assert.NotEmpty(t, results.Evidence)
            assert.True(t, len(results.Evidence) > 0)

            // Golden file comparison for detailed output
            golden := helpers.NewGoldenFile(t)
            golden.Assert(fmt.Sprintf("encryption_analysis_%s.json",
                strings.ReplaceAll(name, " ", "_")),
                []byte(results.ToJSON()))
        })
    }
}
```

### ISO 27001 Test Scenarios

#### Test Case: A.9.1.1 Access Control Policy

**Scenario**: Validate access control policy implementation and documentation

**Test Implementation**:
```go
func TestISO27001_A911_AccessControlPolicy(t *testing.T) {
    tests := map[string]struct {
        policyDocs      []PolicyDocument
        implementationEvidence []ImplementationEvidence
        expectedCompliance ControlCompliance
    }{
        "comprehensive access control framework": {
            policyDocs: []PolicyDocument{
                {
                    ID: "POL-0001",
                    Title: "Information Security Access Control Policy",
                    Version: "v2.1",
                    ApprovedBy: "CISO",
                    EffectiveDate: time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
                    ReviewDate: time.Date(2024, 12, 31, 0, 0, 0, 0, time.UTC),
                    Sections: []string{"scope", "roles", "procedures", "enforcement"},
                },
            },
            implementationEvidence: []ImplementationEvidence{
                {
                    Type: "technical_control",
                    Description: "RBAC implementation in production systems",
                    Location: "aws-iam-policies",
                    LastVerified: time.Now().Add(-24 * time.Hour),
                },
                {
                    Type: "administrative_control",
                    Description: "Access review procedures",
                    Location: "quarterly-access-reviews",
                    LastVerified: time.Now().Add(-72 * time.Hour),
                },
            },
            expectedCompliance: ControlCompliance{
                Status: "IMPLEMENTED",
                Coverage: 95.0,
                Maturity: "OPTIMIZED",
                GapCount: 1,
            },
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            // Create evidence collector
            collector := iso27001.NewAccessControlPolicyCollector()

            // Execute compliance assessment
            assessment, err := collector.AssessCompliance(context.Background(),
                tt.policyDocs, tt.implementationEvidence)
            require.NoError(t, err)

            // Validate compliance results
            assert.Equal(t, tt.expectedCompliance.Status, assessment.Status)
            assert.InDelta(t, tt.expectedCompliance.Coverage, assessment.Coverage, 1.0)
            assert.Equal(t, tt.expectedCompliance.Maturity, assessment.Maturity)

            // Validate evidence traceability
            assert.True(t, assessment.HasTraceableEvidence())
            assert.NotEmpty(t, assessment.EvidenceReferences)

            // Validate audit readiness
            auditPackage, err := assessment.GenerateAuditPackage()
            require.NoError(t, err)
            assert.True(t, auditPackage.IsComplete())
        })
    }
}
```

### Performance Test Cases

#### Test Case: Large-Scale Evidence Collection

**Scenario**: Validate system performance under realistic compliance workloads

**Test Implementation**:
```go
func TestLargeScaleEvidenceCollection(t *testing.T) {
    if testing.Short() {
        t.Skip("Skipping large-scale test in short mode")
    }

    tests := map[string]struct {
        taskCount       int
        concurrency     int
        maxDuration     time.Duration
        expectedSuccess float64
    }{
        "moderate load - 100 tasks": {
            taskCount:       100,
            concurrency:     10,
            maxDuration:     5 * time.Minute,
            expectedSuccess: 98.0,
        },
        "high load - 1000 tasks": {
            taskCount:       1000,
            concurrency:     25,
            maxDuration:     15 * time.Minute,
            expectedSuccess: 95.0,
        },
        "stress test - 5000 tasks": {
            taskCount:       5000,
            concurrency:     50,
            maxDuration:     45 * time.Minute,
            expectedSuccess: 90.0,
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            // Setup performance monitoring
            perfMonitor := NewPerformanceMonitor()
            perfMonitor.Start()
            defer perfMonitor.Stop()

            // Generate test tasks
            tasks := generateTestEvidenceTasks(tt.taskCount)

            // Execute collection with concurrency control
            ctx, cancel := context.WithTimeout(context.Background(), tt.maxDuration)
            defer cancel()

            results, err := ExecuteConcurrentCollection(ctx, tasks, tt.concurrency)
            require.NoError(t, err)

            // Validate performance metrics
            successRate := float64(results.SuccessCount) / float64(tt.taskCount) * 100
            assert.GreaterOrEqual(t, successRate, tt.expectedSuccess,
                "Success rate below threshold")

            // Validate timing constraints
            assert.Less(t, results.TotalDuration, tt.maxDuration,
                "Execution exceeded maximum duration")

            // Validate resource usage
            memoryUsage := perfMonitor.GetPeakMemoryUsage()
            assert.Less(t, memoryUsage.HeapInUse, 2*1024*1024*1024, // 2GB limit
                "Memory usage exceeded 2GB")

            // Validate concurrent processing efficiency
            concurrencyEfficiency := results.ConcurrencyEfficiency()
            assert.GreaterOrEqual(t, concurrencyEfficiency, 0.8,
                "Concurrency efficiency below 80%")
        })
    }
}
```

### Security Test Cases

#### Test Case: Credential Security Validation

**Scenario**: Ensure credentials are handled securely throughout evidence collection

**Test Implementation**:
```go
func TestCredentialSecurityValidation(t *testing.T) {
    tests := map[string]struct {
        credentialType string
        testScenario   string
        expectations   SecurityExpectations
    }{
        "GitHub token handling": {
            credentialType: "github_token",
            testScenario:   "token_lifecycle",
            expectations: SecurityExpectations{
                NoPlaintextLogging: true,
                SecureStorage:      true,
                AutomaticRotation:  false,
                EncryptionAtRest:   true,
            },
        },
        "Tugboat session management": {
            credentialType: "tugboat_session",
            testScenario:   "session_security",
            expectations: SecurityExpectations{
                NoPlaintextLogging: true,
                SecureStorage:      true,
                AutomaticRotation:  true,
                EncryptionAtRest:   true,
            },
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            // Setup security monitoring
            securityMonitor := NewSecurityMonitor()
            securityMonitor.Start()
            defer securityMonitor.Stop()

            // Execute credential operations
            credManager := NewCredentialManager(tt.credentialType)

            // Test credential lifecycle
            credential, err := credManager.AcquireCredential()
            require.NoError(t, err)

            // Use credential in evidence collection
            evidence, err := CollectEvidenceWithCredential(credential)
            require.NoError(t, err)

            // Release credential
            err = credManager.ReleaseCredential(credential)
            require.NoError(t, err)

            // Validate security expectations
            securityReport := securityMonitor.GenerateReport()

            if tt.expectations.NoPlaintextLogging {
                assert.False(t, securityReport.HasPlaintextCredentials(),
                    "Plaintext credentials found in logs")
            }

            if tt.expectations.SecureStorage {
                assert.True(t, securityReport.HasSecureStorage(),
                    "Credentials not stored securely")
            }

            if tt.expectations.EncryptionAtRest {
                assert.True(t, securityReport.HasEncryptionAtRest(),
                    "Credentials not encrypted at rest")
            }

            // Validate no credential leakage
            assert.Empty(t, securityReport.GetCredentialLeaks(),
                "Credential leakage detected")
        })
    }
}
```

### Error Handling Test Cases

#### Test Case: API Failure Recovery

**Scenario**: Validate graceful handling of external API failures

**Test Implementation**:
```go
func TestAPIFailureRecovery(t *testing.T) {
    tests := map[string]struct {
        failureType    string
        failureConfig  FailureConfig
        expectedResult RecoveryResult
    }{
        "GitHub API rate limit": {
            failureType: "rate_limit",
            failureConfig: FailureConfig{
                StatusCode:    403,
                RetryAfter:    60,
                MaxRetries:    3,
                BackoffFactor: 2.0,
            },
            expectedResult: RecoveryResult{
                ShouldRecover:    true,
                MaxRetryDuration: 4 * time.Minute,
                FallbackStrategy: "partial_collection",
            },
        },
        "Tugboat temporary outage": {
            failureType: "service_unavailable",
            failureConfig: FailureConfig{
                StatusCode:    503,
                RetryAfter:    30,
                MaxRetries:    5,
                BackoffFactor: 1.5,
            },
            expectedResult: RecoveryResult{
                ShouldRecover:    true,
                MaxRetryDuration: 8 * time.Minute,
                FallbackStrategy: "cached_data",
            },
        },
        "Claude AI quota exceeded": {
            failureType: "quota_exceeded",
            failureConfig: FailureConfig{
                StatusCode:    429,
                RetryAfter:    3600,
                MaxRetries:    1,
                BackoffFactor: 1.0,
            },
            expectedResult: RecoveryResult{
                ShouldRecover:    false,
                MaxRetryDuration: 0,
                FallbackStrategy: "manual_generation",
            },
        },
    }

    for name, tt := range tests {
        t.Run(name, func(t *testing.T) {
            // Setup failure simulation
            failureSimulator := NewAPIFailureSimulator(tt.failureConfig)
            defer failureSimulator.Reset()

            // Configure client with failure simulation
            client := NewResilientAPIClient(WithFailureSimulation(failureSimulator))

            // Execute evidence collection
            startTime := time.Now()
            result, err := client.CollectEvidence(context.Background(), "ET-0001")
            duration := time.Since(startTime)

            if tt.expectedResult.ShouldRecover {
                // Should eventually succeed
                assert.NoError(t, err)
                assert.NotNil(t, result)
                assert.LessOrEqual(t, duration, tt.expectedResult.MaxRetryDuration)
            } else {
                // Should fail gracefully
                assert.Error(t, err)
                assert.Contains(t, err.Error(), tt.failureType)
            }

            // Validate fallback strategy was applied
            assert.Equal(t, tt.expectedResult.FallbackStrategy,
                client.GetLastFallbackStrategy())

            // Validate retry behavior
            retryCount := failureSimulator.GetRetryCount()
            assert.LessOrEqual(t, retryCount, tt.failureConfig.MaxRetries)
        })
    }
}
```

### Integration Test Helpers

#### Evidence Validation Helper
```go
// helpers/evidence_validator.go
type EvidenceValidator struct {
    schemaValidator JSONSchemaValidator
    contentAnalyzer ContentAnalyzer
}

func (ev *EvidenceValidator) ValidateEvidence(evidence *Evidence) (*ValidationResult, error) {
    result := &ValidationResult{
        Valid:   true,
        Errors:  []string{},
        Warnings: []string{},
    }

    // Schema validation
    if err := ev.schemaValidator.Validate(evidence); err != nil {
        result.Valid = false
        result.Errors = append(result.Errors, fmt.Sprintf("Schema validation failed: %v", err))
    }

    // Content analysis
    contentResult := ev.contentAnalyzer.Analyze(evidence.Content)
    if contentResult.QualityScore < 0.8 {
        result.Warnings = append(result.Warnings,
            fmt.Sprintf("Low content quality score: %.2f", contentResult.QualityScore))
    }

    // Compliance mapping validation
    if len(evidence.ControlMappings) == 0 {
        result.Valid = false
        result.Errors = append(result.Errors, "No control mappings found")
    }

    return result, nil
}
```

#### Test Data Builder
```go
// helpers/evidence_builder.go
type EvidenceTaskBuilder struct {
    task *models.EvidenceTask
}

func NewEvidenceTaskBuilder() *EvidenceTaskBuilder {
    return &EvidenceTaskBuilder{
        task: &models.EvidenceTask{
            ID:          uuid.New().String(),
            Name:        "Test Evidence Task",
            Description: "Test evidence task for validation",
            Framework:   "SOC2",
            Status:      "active",
            CreatedAt:   time.Now(),
            UpdatedAt:   time.Now(),
        },
    }
}

func (b *EvidenceTaskBuilder) WithSOC2Controls(controls ...string) *EvidenceTaskBuilder {
    for _, control := range controls {
        b.task.Controls = append(b.task.Controls, models.Control{
            ID:        control,
            Framework: "SOC2",
            Name:      fmt.Sprintf("SOC2 Control %s", control),
        })
    }
    return b
}

func (b *EvidenceTaskBuilder) WithISO27001Controls(controls ...string) *EvidenceTaskBuilder {
    for _, control := range controls {
        b.task.Controls = append(b.task.Controls, models.Control{
            ID:        control,
            Framework: "ISO27001",
            Name:      fmt.Sprintf("ISO 27001 Control %s", control),
        })
    }
    return b
}

func (b *EvidenceTaskBuilder) WithAutomationLevel(level string) *EvidenceTaskBuilder {
    b.task.AutomationLevel = level
    return b
}

func (b *EvidenceTaskBuilder) Build() *models.EvidenceTask {
    return b.task
}
```

## Test Performance Benchmarks

### Evidence Collection Performance Standards

| Evidence Type | Target Time | Current Performance | Status |
|---------------|-------------|-------------------|--------|
| Infrastructure Analysis | <30s | 18s | ✅ |
| GitHub Security Scan | <45s | 32s | ✅ |
| Policy Document Review | <60s | 41s | ✅ |
| Multi-Framework Analysis | <120s | 95s | ✅ |
| Large Dataset Processing | <300s | 240s | ✅ |

### Quality Metrics for Test Cases

| Test Category | Coverage Target | Current Coverage | Mutation Score |
|---------------|-----------------|------------------|----------------|
| Evidence Collection | 95% | 92% | 78% |
| Error Handling | 90% | 88% | 82% |
| Security Validation | 100% | 96% | 85% |
| Performance Testing | 85% | 79% | 71% |
| Integration Testing | 90% | 87% | 76% |

## References

- [[security-testing]] - Security-specific testing approaches
- [[performance-testing]] - Performance testing and optimization
- [[test-automation]] - CI/CD testing automation
- [[system-architecture]] - Architecture testing patterns
- [[evidence-collection-guide]] - Evidence collection implementation patterns
- [[compliance-framework-testing]] - Framework-specific testing strategies
- [[error-recovery-patterns]] - Error handling and recovery testing

---

*This comprehensive testing strategy ensures GRCTool meets the highest standards for reliability, security, and performance while providing concrete test cases that validate compliance requirements and real-world usage scenarios.*