# Testing Structure and Guidelines

**Version:** 1.0
**Date:** 2025-10-10
**Purpose:** Define the complete testing strategy for GRCTool

## Overview

GRCTool uses a **4-tier testing pyramid** to ensure quality at every level:

```
         ┌─────────────┐
         │     E2E     │  Real APIs, Full workflows
         │   (Manual)  │  Requires authentication
         └─────────────┘
              ▲
         ┌─────────────┐
         │ Functional  │  CLI binary testing
         │  (5-10 min) │  No real APIs
         └─────────────┘
              ▲
         ┌─────────────┐
         │ Integration │  VCR recordings
         │  (2-5 min)  │  Mock HTTP calls
         └─────────────┘
              ▲
         ┌─────────────┐
         │    Unit     │  Pure functions
         │  (<30 sec)  │  Fast, isolated
         └─────────────┘
```

---

## Test Tier Breakdown

### Tier 1: Unit Tests (No Build Tags)

**Purpose**: Test individual functions and components in isolation

**Characteristics**:
- No external dependencies
- Use mocks/fakes for all I/O
- Fast execution (< 30 seconds total)
- Run on every commit

**Build Tags**: None (default tests)

**Location**: Alongside source files (`*_test.go`)

**Example**:
```go
// internal/config/config_test.go
func TestConfigValidation(t *testing.T) {
    cfg := &Config{
        Tugboat: TugboatConfig{
            BaseURL: "https://api.tugboat.com",
        },
    }
    err := cfg.Validate()
    assert.NoError(t, err)
}
```

**Run Command**:
```bash
make test
# or
go test ./...
```

**Configuration**: No config file needed, uses in-memory test data

---

### Tier 2: Integration Tests (`//go:build integration`)

**Purpose**: Test component integration with mocked external services

**Characteristics**:
- Uses VCR cassettes to record/replay HTTP interactions
- Tests API client logic without hitting real APIs
- Validates configuration and error handling
- Runs in CI on every PR

**Build Tags**: `//go:build integration`

**Location**: `test/integration/` or alongside source with build tag

**Example**:
```go
//go:build integration

package tugboat

func TestTugboatClientWithVCR(t *testing.T) {
    // VCR will replay recorded HTTP interactions
    client := NewClient(cfg, vcrConfig)
    tasks, err := client.GetEvidenceTasks(ctx, opts)
    assert.NoError(t, err)
    assert.NotEmpty(t, tasks)
}
```

**Run Command**:
```bash
make test-integration
# or
go test -tags=integration ./...
```

**Configuration**: Uses `t.TempDir()` for isolated test directories

**VCR Cassettes**: Stored in `internal/*/testdata/vcr/`

---

### Tier 3: Functional Tests (`//go:build functional`)

**Purpose**: Test CLI binary behavior end-to-end

**Characteristics**:
- Requires built `bin/grctool` binary
- Tests command-line interface
- Uses fixtures and mock servers
- No real API calls
- Runs in CI after build

**Build Tags**: `//go:build functional`

**Location**: `test/functional/`

**Example**:
```go
//go:build functional

func TestCLISyncCommand(t *testing.T) {
    cmd := exec.Command("./bin/grctool", "sync", "--help")
    output, err := cmd.Output()
    assert.NoError(t, err)
    assert.Contains(t, string(output), "synchronize")
}
```

**Run Command**:
```bash
make test-functional
# Requires: go build -o bin/grctool first
```

**Configuration**: Inline config or test fixtures

---

### Tier 4: End-to-End Tests (`//go:build e2e`)

**Purpose**: Test complete workflows with real APIs

**Characteristics**:
- Requires **real authentication** (GITHUB_TOKEN, Tugboat auth)
- Makes **actual API calls** to external services
- Tests full user workflows
- **Manual execution only** (not in CI)
- Slow (minutes to run)

**Build Tags**: `//go:build e2e`

**Location**: `test/e2e/`

**Configuration**: `test/fixtures/e2e/.grctool.yaml`

**Example**:
```go
//go:build e2e

func TestCompleteSOC2Audit_E2E(t *testing.T) {
    if !hasRequiredAuth(t) {
        t.Skip("Requires real authentication")
    }

    // Uses real Tugboat and GitHub APIs
    cfg := helpers.SetupE2ETest(t)
    // ... test complete workflow
}
```

**Run Command**:
```bash
# Requires authentication setup first
./bin/grctool auth login

# Run E2E tests
make test-e2e
# or
go test -tags=e2e ./test/e2e/
```

**Required Environment Variables**:
- `GITHUB_TOKEN` - GitHub personal access token
- `TUGBOAT_BEARER` - Set via `grctool auth login`

**Configuration File**: `test/fixtures/e2e/.grctool.yaml`

---

## Configuration Files and Path Resolution

### How Config Paths Work

**Key Principle**: Paths in config files are **relative to the config file location**, not the current working directory.

```yaml
# In /project/.grctool.yaml
storage:
  data_dir: "../data"  # Resolves to /data
  cache_dir: "./.cache"              # Resolves to /project/.cache
```

This means:
- ✅ Config files are portable
- ✅ Tests work regardless of CWD
- ✅ Same config works from any directory

### Production Config

**Location**: `/pool0/erik/Projects/grctool/.grctool.yaml` (gitignored)

**Purpose**: Local development and production use

**Template**: `configs/example.yaml`

**Setup**:
```bash
cp configs/example.yaml .grctool.yaml
# Edit .grctool.yaml with your settings
```

### Test Config

**Location**: `test/fixtures/e2e/.grctool.yaml`

**Purpose**: E2E test execution

**Paths**: Relative to `test/fixtures/e2e/` directory

**Data Directory**: Points to `test/fixtures/isms/` (sibling directory)

**Used By**: `helpers.SetupE2ETest(t)` in E2E tests

---

## Test Data Fixtures

### Directory Structure

```
test/fixtures/
├── e2e/
│   ├── .grctool.yaml          # E2E test config
│   ├── local_data/            # Generated (gitignored)
│   └── .cache/                # Generated (gitignored)
├── isms/                      # Test ISMS data
│   ├── docs/
│   │   ├── policies/          # Sample policy JSON files
│   │   ├── controls/          # Sample control JSON files
│   │   ├── evidence_tasks/    # Sample evidence task JSON files
│   │   └── sync_times/        # Sync metadata
│   ├── evidence/              # Generated evidence (gitignored)
│   └── prompts/               # Sample prompts
├── terraform/                 # Terraform test fixtures
│   ├── large/                 # Large Terraform projects
│   └── medium/                # Medium-sized projects
└── vcr_cassettes/             # VCR recordings (gitignored if generated)
```

### Creating Test Data

**Only create data if tests actually need it.** Most E2E tests use VCR recordings or skip if data unavailable.

**Minimal sample** (if needed):

```json
// test/fixtures/isms/docs/evidence_tasks/328001.json
{
  "id": 328001,
  "reference_id": "ET-001",
  "name": "Test Evidence Task",
  "collection_interval": "quarterly",
  "framework": "SOC2",
  "status": "pending"
}
```

---

## Running Tests

### Quick Reference

```bash
# Unit tests (fast, always run)
make test

# Integration tests (VCR recordings)
make test-integration

# Functional tests (requires build)
make build
make test-functional

# E2E tests (requires auth, manual only)
./bin/grctool auth login
make test-e2e

# All non-E2E tests
make test test-integration test-functional

# Check coverage
make coverage
```

### CI/CD Pipeline

**GitHub Actions** runs automatically:

1. **Unit Tests** - Every commit
2. **Integration Tests** - Every commit (uses VCR)
3. **Functional Tests** - Every PR (builds binary first)
4. **E2E Tests** - Manual only (requires secrets)

**Configuration**: `.github/workflows/testing.yml`

---

## Test Helpers

### Unit Test Helpers

**Location**: `test/helpers/unit_helpers.go`

**Key Functions**:
- `CreateTestConfig(t)` - In-memory test config
- `CreateTempDir(t)` - Temporary directory with cleanup

**Usage**:
```go
cfg := helpers.CreateTestConfig(t)
```

### Integration Test Helpers

**Location**: `test/helpers/integration_helpers.go`

**Key Functions**:
- `SetupIntegrationTest(t, toolName)` - Config + VCR setup
- `SetupGitHubIntegrationTest(t)` - GitHub VCR client
- `SetupTerraformIntegrationTest(t)` - Terraform fixtures

**Usage**:
```go
cfg, log := helpers.SetupIntegrationTest(t, "terraform")
```

### Functional Test Helpers

**Location**: `test/helpers/functional_helpers.go`

**Key Functions**:
- `RequireBinaryExists(t, binary)` - Ensure grctool built
- `RunCLICommand(t, args...)` - Execute grctool command

**Usage**:
```go
helpers.RequireBinaryExists(t, "./bin/grctool")
output := helpers.RunCLICommand(t, "sync", "--help")
```

### E2E Test Helpers

**Location**: `test/helpers/e2e_helpers.go`

**Key Functions**:
- `SetupE2ETest(t)` - **Loads test config from fixtures**
- `SkipIfNoGitHubAuth(t)` - Skip if no GitHub token
- `SkipIfNoTugboatAuth(t)` - Skip if no Tugboat auth
- `ValidateE2EEnvironment(t)` - Check auth and connectivity

**Usage**:
```go
cfg := helpers.SetupE2ETest(t)  // Uses test/fixtures/e2e/.grctool.yaml
```

---

## Best Practices

### Test Organization

1. **Colocate unit tests** with source code
2. **Use build tags** for integration/functional/e2e
3. **One test file per source file** when possible
4. **Group related tests** with `t.Run()` subtests

### Naming Conventions

```go
// Unit test
func TestFunctionName(t *testing.T) { }

// Integration test
//go:build integration
func TestClientIntegration(t *testing.T) { }

// Functional test
//go:build functional
func TestCLICommand_Functional(t *testing.T) { }

// E2E test
//go:build e2e
func TestWorkflow_E2E(t *testing.T) { }
```

### Configuration in Tests

**DO**:
- ✅ Use `helpers.SetupE2ETest(t)` for E2E tests
- ✅ Use `t.TempDir()` for temporary directories
- ✅ Load config from `test/fixtures/e2e/.grctool.yaml`

**DON'T**:
- ❌ Hardcode paths like `DataDir: "../../"`
- ❌ Create config structs inline for E2E tests
- ❌ Assume specific working directory

### Skipping Tests

```go
// Skip if environment not set up
if os.Getenv("GITHUB_TOKEN") == "" {
    t.Skip("GITHUB_TOKEN required")
}

// Skip optional E2E test
if os.Getenv("TEST_QUARTERLY_REVIEW") == "" {
    t.Skip("TEST_QUARTERLY_REVIEW not enabled")
}
```

### Assertions

Use `testify/assert` and `testify/require`:

```go
// assert - test continues if fails
assert.NotEmpty(t, output)
assert.Contains(t, output, "success")

// require - test stops if fails
require.NoError(t, err)
require.NotNil(t, cfg)
```

---

## Debugging Tests

### Verbose Output

```bash
go test -v ./internal/config
go test -v -tags=integration ./...
```

### Run Single Test

```bash
go test -v ./internal/config -run TestConfigValidation
go test -v -tags=e2e ./test/e2e -run TestCompleteSOC2Audit
```

### Check Test Coverage

```bash
make coverage
# Opens coverage report in browser
```

### VCR Debugging

```bash
# Re-record VCR cassettes
VCR_MODE=record go test -tags=integration ./internal/tugboat

# Use live API (bypass VCR)
VCR_MODE=off go test -tags=integration ./internal/tugboat
```

---

## Common Issues

### "Config file not found"

**Cause**: Running E2E test without test config

**Fix**: Ensure `test/fixtures/e2e/.grctool.yaml` exists

```bash
ls test/fixtures/e2e/.grctool.yaml
```

### "DataDir path incorrect"

**Cause**: Config paths resolved from wrong location

**Fix**: Use `helpers.SetupE2ETest(t)` instead of manual config

### "Authentication required"

**Cause**: E2E test needs real credentials

**Fix**: Set up authentication:

```bash
# GitHub
export GITHUB_TOKEN="ghp_..."

# Tugboat
./bin/grctool auth login
```

### "Binary not found"

**Cause**: Functional test can't find `./bin/grctool`

**Fix**: Build binary first

```bash
make build
make test-functional
```

---

## Adding New Tests

### 1. Determine Test Tier

Ask:
- Pure logic test? → **Unit test**
- Need HTTP mocking? → **Integration test**
- Testing CLI behavior? → **Functional test**
- Need real APIs? → **E2E test**

### 2. Add Build Tag (if needed)

```go
//go:build integration

package mypackage
```

### 3. Use Appropriate Helper

```go
// E2E
cfg := helpers.SetupE2ETest(t)

// Integration
cfg, log := helpers.SetupIntegrationTest(t, "tool-name")

// Unit
cfg := &config.Config{ /* test config */ }
```

### 4. Add to CI (optional)

Update `.github/workflows/testing.yml` if needed

---

## References

- [Testing Best Practices](https://go.dev/doc/effective_go#testing)
- [Build Tags Documentation](https://pkg.go.dev/cmd/go#hdr-Build_constraints)
- [Testify Documentation](https://github.com/stretchr/testify)
- [VCR for Go](https://github.com/dnaeon/go-vcr)

---

## Summary

**GRCTool uses a 4-tier testing strategy:**

| Tier | Build Tag | Speed | Auth | Config | Run In CI |
|------|-----------|-------|------|--------|-----------|
| Unit | None | < 30s | No | In-memory | ✅ Always |
| Integration | `integration` | 2-5 min | No | t.TempDir() | ✅ Always |
| Functional | `functional` | 5-10 min | No | Inline | ✅ On PR |
| E2E | `e2e` | 10+ min | **Yes** | test/fixtures/e2e/.grctool.yaml | ❌ Manual only |

**Key Takeaway**: `.grctool.yaml` is the single source of truth for all configuration. Paths in config files are resolved relative to the config file location, making configs portable and tests reliable.
