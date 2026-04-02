---
title: "Test Procedures"
phase: "03-test"
category: "testing"
tags: ["testing", "procedures", "unit", "integration", "functional", "e2e", "vcr", "coverage", "mutation", "benchmarks"]
related: ["testing-strategy", "development-practices", "build-procedures"]
created: 2026-03-17
updated: 2026-03-17
helix_mapping: "03-Test / Test Procedures"
---

# Test Procedures

## Overview

This document provides step-by-step procedures for running, writing, and maintaining tests in GRCTool. It covers each testing tier, the VCR cassette system, coverage reporting, mutation testing, and performance benchmarking. All commands reference the actual Makefile targets and CI pipeline configuration.

## Prerequisites

### Required Tools

| Tool | Purpose | Installation |
|------|---------|-------------|
| Go 1.24+ | Compiler and test runner | `go.dev/dl` |
| golangci-lint | Linting and static analysis | `go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest` |
| gosec | Security scanning | `go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest` |
| govulncheck | Dependency vulnerability checking | `go install golang.org/x/vuln/cmd/govulncheck@latest` |
| gremlins | Mutation testing | `go install github.com/go-gremlins/gremlins/cmd/gremlins@latest` |
| goimports | Import organization | `go install golang.org/x/tools/cmd/goimports@latest` |
| lefthook | Git hooks manager | `go install github.com/evilmartians/lefthook@latest` |
| gh CLI | GitHub authentication for VCR recording | `brew install gh` or system package manager |

### Environment Setup

```bash
# Clone the repository
git clone https://github.com/grctool/grctool.git
cd grctool

# Download dependencies
make deps

# Install lefthook hooks
lefthook install

# Verify the build compiles
make build
```

### Environment Variables

| Variable | Purpose | Default | Required For |
|----------|---------|---------|-------------|
| `VCR_MODE` | VCR recording mode | `playback` | Integration tests |
| `GITHUB_TOKEN` | GitHub API authentication | (none) | E2E tests, VCR recording |
| `TUGBOAT_BASE_URL` | Tugboat API endpoint | (none) | E2E tests |
| `UPDATE_GOLDEN` | Update golden files | `false` | Golden file refresh |
| `GITHUB_RATE_LIMIT` | Requests per minute during recording | (unlimited) | VCR recording |

## Test Tier Procedures

### Tier 1: Unit Tests

**Characteristics**: Fast (2-3 seconds), no external dependencies, no network calls, no file system side effects.

**Build Tags**: `//go:build !e2e && !integration`

**Run Command**:
```bash
make test-unit
# Equivalent to: go test -timeout=30s -v ./internal/... ./cmd/... -count=1
```

**Writing Unit Tests**:

1. Place test files alongside the code they test: `internal/auth/auth.go` -> `internal/auth/auth_test.go`
2. Use the table-driven test pattern with map keys for descriptive names:

```go
func TestIsHigherPermission(t *testing.T) {
    tests := map[string]struct {
        perm1    string
        perm2    string
        expected bool
    }{
        "admin higher than push": {
            perm1: "admin", perm2: "push", expected: true,
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

3. Use `t.TempDir()` for any file system operations (automatically cleaned up).
4. Use `t.Parallel()` when the test has no shared mutable state.
5. Mock external dependencies using interfaces and test doubles (see `internal/auth/provider_test.go` for `mockLogger` pattern).

**Naming Convention**: `*_test.go` in the same package.

### Tier 2: Integration Tests

**Characteristics**: Uses VCR cassette recordings, tests cross-package interactions, runs in ~2 minutes.

**Build Tags**: `//go:build !e2e`

**Run Command**:
```bash
make test-integration
# Equivalent to: VCR_MODE=playback go test -tags=integration -timeout=2m -v ./internal/... ./test/integration/... -count=1
```

**Writing Integration Tests**:

1. Place tests in `test/integration/` or use `_integration_test.go` suffix.
2. Set up VCR for API interactions:

```go
func TestGitHubAPI(t *testing.T) {
    vcr := helpers.SetupVCR(t, "github_api_test")
    defer vcr.Stop()

    client := github.NewClient(vcr.GetHTTPClient())
    repos, err := client.ListRepositories("testorg")
    assert.NoError(t, err)
    assert.NotEmpty(t, repos)
}
```

3. Use test data builders from `test/helpers/`:
   - `helpers.NewEvidenceTaskBuilder()`
   - `helpers.NewPolicyBuilder()`
   - `helpers.NewControlBuilder()`

**Naming Convention**: `*_integration_test.go` or files under `test/integration/`.

### Tier 3: Functional Tests

**Characteristics**: Tests the compiled CLI binary end-to-end, includes file system operations, runs in ~5 minutes.

**Build Tags**: `//go:build functional`

**Run Command**:
```bash
make test-functional
# Builds binary first (make build-test), then:
# go test -tags="functional" -timeout=5m -v ./test/functional/... -count=1
```

**Specialized Functional Test Targets**:
```bash
make test-functional-evidence     # Evidence collection tests
make test-functional-workflows    # CLI workflow tests (TestCompleteAuditWorkflow)
make test-functional-errors       # Error handling tests (TestCLI_.*Error)
make test-functional-performance  # Performance tests
```

**Writing Functional Tests**:

1. Place tests in `test/functional/`.
2. Tests should invoke the built binary at `bin/grctool`.
3. Use `t.TempDir()` for working directories.
4. Assert on CLI exit codes, stdout/stderr output, and generated files.

**Naming Convention**: `*_functional_test.go` under `test/functional/`.

### Tier 4: End-to-End Tests

**Characteristics**: Real API interactions, requires authentication, runs in 10+ minutes.

**Build Tags**: `//go:build e2e`

**Run Command**:
```bash
make test-e2e
# Equivalent to: go test -tags="e2e" -timeout=10m -v ./test/e2e/... -count=1
```

**Specialized E2E Targets**:
```bash
make test-e2e-github         # GitHub API tests only
make test-e2e-tugboat        # Tugboat API tests only
make test-e2e-audit          # Complete audit scenario
make test-e2e-performance    # Performance E2E tests
make test-e2e-config         # Environment configuration validation
make test-e2e-quick          # Subset for fast validation
make test-e2e-comprehensive  # All categories with env flags enabled
```

**Prerequisites**:
- `gh auth login` completed (for `GITHUB_TOKEN`)
- Tugboat Logic credentials configured
- Network access to external APIs

**Naming Convention**: `*_e2e_test.go` under `test/e2e/`.

## VCR Cassette Management

### What is VCR?

VCR (Video Cassette Recorder) records HTTP interactions to JSON cassette files and replays them during subsequent test runs. This enables deterministic integration tests without live API access.

### VCR Modes

| Mode | Behavior | Use Case |
|------|----------|----------|
| `off` | No recording or playback; direct HTTP | Debugging |
| `record` | Record all interactions, overwrite existing cassettes | Fresh recording |
| `record_once` | Record only if cassette does not exist | Incremental recording |
| `playback` | Replay from cassettes; fail if no match | CI, local development |

### Recording New Cassettes

```bash
# Record all cassettes (overwrites existing)
make test-record
# Uses: GITHUB_TOKEN=$(gh auth token) VCR_MODE=record go test ./test/integration/...

# Record only missing cassettes (preserves existing)
make test-record-missing
# Uses: VCR_MODE=record_once with 5 req/min rate limit
# Takes ~10 minutes; only needed once per new test
```

**Rate Limiting During Recording**: The `test-record-missing` target sets `GITHUB_RATE_LIMIT=5` to avoid triggering GitHub's anti-scraping detection. This is intentionally slow.

### Playback (Default)

```bash
make test-playback
# Equivalent to: VCR_MODE=playback go test ./test/integration/...
```

### Cassette Sanitization

The VCR system automatically redacts sensitive headers during recording:

- `Authorization` headers are replaced with `[REDACTED]`
- Configurable via `RedactHeaders` in VCR `Config`
- Cassettes are stored as JSON in the test fixtures directory

**Verification**: Run `TestVCR_RecordAndPlayback` which explicitly asserts that `secret-token` does not appear in the cassette file and `[REDACTED]` does.

### Troubleshooting Cassettes

| Problem | Solution |
|---------|----------|
| Cassette not found during playback | Run `make test-record-missing` to record it |
| Cassette contains stale data | Delete the cassette file and re-record with `make test-record` |
| API response format changed | Delete affected cassettes and re-record |
| Validate cassette structure | Run `go test ./test/integration/vcr_validation_test.go` |

## Coverage Reporting

### Coverage Targets

| Level | Target | Current |
|-------|--------|---------|
| Overall (CI, `ci.yml`) | 15%+ | 22.4% |
| Overall (local, `make coverage-check`) | 80%+ | 22.4% |
| Critical Packages | 90%+ | Varies |
| Test Execution Time | <5s | ~3s |

Note: The CI pipeline (`ci.yml`) enforces a 15% threshold, set realistically for the current state. The local `make coverage-check` target enforces 80%, which is the aspirational target. The `testing.yml` quality-gates job also checks against 80%, but `testing.yml` is currently disabled.

### Running Coverage

```bash
# Generate HTML coverage report
make coverage-report
# Output: coverage.html

# Check if coverage meets 80% threshold (local target)
make coverage-check

# Check critical packages only
make coverage-critical

# Generate coverage badge
make coverage-badge

# Full coverage for all test types
make test-coverage      # Unit tests only
make test-coverage-all  # All tests
```

### Viewing Coverage

```bash
# Open the HTML report
open coverage.html    # macOS
xdg-open coverage.html  # Linux

# View function-level coverage in terminal
go tool cover -func=coverage.out

# View specific package coverage
go test -coverprofile=temp.out ./internal/auth/...
go tool cover -func=temp.out
```

### CI Coverage

The CI pipeline (`ci.yml`) runs coverage analysis in the `coverage` job:

1. Runs `go test -coverprofile=coverage.out -covermode=atomic ./...`
2. Checks against a **15% threshold** (set realistically for current state; `make coverage-check` uses 80% locally)
3. Uploads results to Codecov
4. Generates HTML report as a build artifact

## Mutation Testing

### Overview

Mutation testing verifies test quality by introducing small code changes (mutations) and checking whether tests detect them.

**Tool**: Gremlins (`go-gremlins/gremlins`)

### Running Mutation Tests

```bash
# Full mutation testing (all packages)
make mutation-test

# Quick mutation testing (critical packages only)
make mutation-quick

# Dry run (analysis without execution -- fast)
make mutation-dry-run

# Generate HTML report from existing results
make mutation-report

# Establish baseline scores
make mutation-baseline
```

### Mutation Score Targets

| Package Category | Efficacy Target | Coverage Target |
|-----------------|----------------|----------------|
| Critical Packages | 80%+ | 85%+ |
| Standard Packages | 70%+ | 80%+ |
| Utility Packages | 60%+ | 70%+ |

### Interpreting Results

- **Killed Mutant**: Test detected the mutation (good).
- **Surviving Mutant**: Test did not detect the mutation (indicates a gap).
- **Mutation Score**: `killed / total * 100` -- higher is better.

Current mutation score: ~59.7% (target: 70%+).

### CI Integration

The `testing.yml` workflow runs mutation testing weekly (Monday 2 AM UTC) or on manual dispatch. Results are uploaded as artifacts.

## Performance Benchmarking

### Running Benchmarks

```bash
# Run all benchmarks
make bench
# Output saved to: benchmarks/current.txt

# Compare with baseline
make bench-compare

# Generate CPU and memory profiles
make bench-profile
# Profiles saved to: benchmarks/profiles/cpu.prof, benchmarks/profiles/mem.prof

# Save current results as new baseline
make bench-save

# Memory-focused benchmarks
make bench-memory

# Comprehensive benchmark report
make bench-report
```

### Viewing Profiles

```bash
# Interactive CPU profile viewer
go tool pprof benchmarks/profiles/cpu.prof

# Web-based profile viewer
go tool pprof -http=:8080 benchmarks/profiles/cpu.prof
```

### Performance Targets

| Operation | Target | Current |
|-----------|--------|---------|
| Tugboat Sync | <200ms | 145ms |
| Evidence Generation | <100ms | 25ms |
| Auth Validation | <5ms | 1.2ms |
| Large Dataset Processing | <1s | 850ms |

### CI Integration

The `testing.yml` workflow runs benchmarks weekly and uses `benchmark-action/github-action-benchmark` to track regressions. Alert threshold is set to 110% of baseline.

## CI/CD Test Execution Flow

### On Push / Pull Request (`ci.yml`)

```
ci-checks (lefthook CI)
    |
    +---> test-matrix (unit + integration tests)
    +---> build-matrix (linux/amd64 build)
    +---> coverage (coverage analysis + Codecov upload)
    |
    +---> license-check (license headers)
    +---> security-scan (gosec + secret detection)
    |
    v
quality-gates (aggregates all results)
```

### Weekly / Manual (`testing.yml`)

```
unit-tests (Go 1.24.12 baseline)
integration-tests (VCR playback)
functional-tests (CLI binary tests)
coverage (deep analysis + Codecov)
benchmarks (performance tracking)
mutation (mutation score analysis)
    |
    v
quality-gates (coverage threshold + quality report on PR)
```

### On Tag Push (`release.yml`)

```
pre-release-checks (lefthook CI + tag validation)
release (unit tests + GoReleaser)
```

## Composite Test Targets

| Target | Tiers Included | Auth Required |
|--------|---------------|---------------|
| `make test` | Unit + Integration | No |
| `make test-no-auth` | Unit + Integration | No |
| `make test-all` | Unit + Integration + Functional + E2E | Yes |
| `make test-all-comprehensive` | All tiers + all E2E categories | Yes |
| `make ci` | Deps + Fmt + Vet + Lint + Tests + Security | No |
| `make ci-with-auth` | CI + Functional + E2E | Yes |

Note: `make ci` and `make ci-with-auth` exist in the Makefile. `make ci` runs `deps fmt vet lint test-no-auth security-scan`. `make ci-with-auth` runs `deps fmt vet lint test-all security-scan` (includes functional and E2E tests, which require authentication).

## Golden File Testing

Golden file tests compare complex outputs against saved reference files.

### Running Golden File Tests

```bash
# Normal run (asserts output matches golden file)
go test ./... -run TestEvidenceGeneration

# Update golden files to match current output
UPDATE_GOLDEN=true go test ./...
```

### When to Update Golden Files

- After intentional output format changes
- After adding new fields to evidence output
- Never update blindly -- review the diff first

## Test Quality Checklist

Before submitting a PR with new tests:

- [ ] Test has a descriptive name explaining the behavior being tested
- [ ] Test follows the table-driven pattern with map keys
- [ ] Test uses `t.TempDir()` instead of manual temp directory management
- [ ] Test cleans up resources via `t.Cleanup()` or deferred calls
- [ ] Test does not depend on execution order
- [ ] Test uses appropriate build tags for its tier
- [ ] VCR cassettes do not contain secrets (check for `[REDACTED]`)
- [ ] Coverage for changed packages has not decreased
- [ ] `make test-no-auth` passes locally

## Troubleshooting

### Common Issues

| Issue | Diagnosis | Resolution |
|-------|-----------|------------|
| VCR cassette not found | Missing recording for new test | Run `make test-record-missing` |
| Test passes locally, fails in CI | Environment differences | Check `VCR_MODE`, build tags, Go version |
| Flaky test | Timing dependency or shared state | Add `t.Parallel()` guards, use `t.TempDir()` |
| Coverage below threshold | Untested code paths | Run `make coverage-monitor` to identify gaps |
| Benchmark regression | Code change impacted performance | Run `make bench-compare` and profile with `make bench-profile` |
| Mutation test timeout | Gremlins running on large package | Use `make mutation-quick` for critical packages only |

## References

- Testing Strategy: `docs/helix/03-test/test-plan/testing-strategy.md`
- Makefile: `Makefile` (all test targets)
- CI Pipeline: `.github/workflows/ci.yml`
- Advanced Testing: `.github/workflows/testing.yml`
- VCR Implementation: `internal/vcr/vcr_test.go`
- Test Helpers: `test/helpers/`

---

*These procedures are derived from GRCTool's actual Makefile targets, CI configuration, and test infrastructure. All commands are copy-pasteable and reflect the current project setup.*
