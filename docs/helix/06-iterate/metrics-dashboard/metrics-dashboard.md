---
title: "Metrics Dashboard"
phase: "06-iterate"
category: "metrics"
tags: ["metrics", "dashboard", "quality", "performance", "coverage", "security"]
related: ["feedback-analysis", "lessons-learned", "roadmap-feedback"]
created: 2026-03-17
updated: 2026-03-17
helix_mapping: "06-Iterate metrics-dashboard artifact"
---

# Metrics Dashboard

## Executive Summary

This dashboard defines the key metrics for tracking GRCTool's health across code quality, testing, performance, security, and operational effectiveness. Each metric includes its current state, target, and the specific command or process used to measure it.

### Overall Health Score

| Category | Weight | Status | Notes |
|----------|--------|--------|-------|
| Code Quality | 25% | Needs improvement | Coverage below target, lint nearly clean |
| Test Effectiveness | 25% | Progressing | VCR tests reliable, mutation testing established |
| Performance | 15% | Healthy | Benchmarks baselined, no regressions |
| Security | 20% | Healthy | Secret detection, gosec, govulncheck in place |
| Operational | 15% | Healthy | 30 tools, evidence generation functional |

## Code Quality Metrics

### Coverage

| Metric | Current | Target | Trend | How to Measure |
|--------|---------|--------|-------|----------------|
| Overall code coverage | ~22% | >=80% | Improving | `make coverage-report` |
| CI coverage threshold | 15% | 80% | Ratcheting up | `make coverage-check` |
| Critical package coverage | Varies | >=90% | Monitoring | `make coverage-critical` |

**Collection command:**
```bash
# Generate coverage report and check threshold
make coverage-check

# Detailed HTML report
make coverage-report

# Critical packages only
make coverage-critical
```

**CI enforcement**: The `ci.yml` workflow runs coverage analysis and fails if below threshold. The threshold is currently set at 15% and should be ratcheted up as coverage improves.

### Complexity and Maintainability

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Cyclomatic complexity per function | <=15 | `golangci-lint run` (gocyclo linter) |
| Maintainability index | >=20 | `golangci-lint run` (maintidx linter) |
| Cognitive complexity | <=15 | `golangci-lint run` (gocognit linter) |

**Collection command:**
```bash
# Run full linter suite including complexity checks
make lint

# Or directly
golangci-lint run
```

### Linting

| Metric | Current | Target | How to Measure |
|--------|---------|--------|----------------|
| Total lint issues | 4 | 0 | `golangci-lint run --count` |
| Deprecated API usage | 0 | 0 | `golangci-lint run` (staticcheck SA1019) |
| Formatting violations | 0 | 0 | `gofmt -l ./...` |
| Import organization | 0 | 0 | `goimports -l ./...` |

**CI enforcement**: Lefthook pre-commit runs `gofmt`, `goimports`, `go vet`, and `golangci-lint` on staged files.

## Test Metrics

### Test Pass Rates

| Test Tier | Pass Rate | Total Tests | How to Run |
|-----------|-----------|-------------|------------|
| Unit tests | Target: 100% | Growing | `make test-unit` |
| Integration tests (VCR) | 83% (25/30) | 30 | `make test-integration` |
| Functional tests | Requires build | Varies | `make test-functional` |
| E2E tests | Manual (requires auth) | Varies | `make test-e2e` |

**Collection command:**
```bash
# Fast feedback (unit + integration, no auth needed)
make test-no-auth

# Complete suite
make test-all
```

### Mutation Testing Scores

| Metric | Current | Target | How to Measure |
|--------|---------|--------|----------------|
| Overall mutation score | ~59.7% | >=80% | `make mutation-test` |
| Critical package mutation score | Varies | >=85% | `make mutation-quick` |

**Collection commands:**
```bash
# Full mutation testing (slow)
make mutation-test

# Quick mutation test on critical packages
make mutation-quick

# Dry run (fast analysis without running tests)
make mutation-dry-run

# Generate report from existing results
make mutation-report

# Establish baseline
make mutation-baseline
```

### VCR Cassette Coverage

| Metric | How to Measure | Target |
|--------|----------------|--------|
| Cassettes recorded | `ls test/vcr_cassettes/*.yaml | wc -l` | All API-dependent integration tests |
| Cassette freshness | File modification dates | Re-recorded within last 90 days |
| Cassette security validation | Tests: `TestVCRCassettes_SecurityValidation` | No auth tokens in cassettes |

**Recording new cassettes:**
```bash
# Record all cassettes (requires GITHUB_TOKEN)
make test-record

# Record only missing cassettes
make test-record-missing

# Playback existing cassettes
make test-playback
```

## Performance Metrics

### CLI Response Time

| Operation | Target | How to Measure |
|-----------|--------|----------------|
| `grctool version` | < 100ms | `time grctool version` |
| `grctool auth status` | < 500ms | `time grctool auth status` |
| `grctool tool evidence-task-list` | < 2s | `time grctool tool evidence-task-list` |
| `grctool sync` | < 30s | `time grctool sync` |
| `grctool evidence generate ET-XXXX` | < 60s | `time grctool evidence generate ET-XXXX` |

### Benchmark Performance

| Benchmark | Target | How to Measure |
|-----------|--------|----------------|
| Tugboat sync benchmark | < 200ms | `make bench` (BenchmarkTugboatClient_Sync) |
| Evidence generation benchmark | < 100ms | `make bench` |
| Auth validation benchmark | < 5ms | `make bench` |
| Large file processing | < 500ms | `make bench-memory` |

**Collection commands:**
```bash
# Run all benchmarks (3 iterations, single CPU)
make bench
# Results saved to benchmarks/current.txt

# Compare with baseline
make bench-compare

# Save current as baseline
make bench-save

# Generate CPU and memory profiles
make bench-profile

# Memory-focused benchmarks
make bench-memory

# Comprehensive comparison report
make bench-report
```

**CI enforcement**: The `testing.yml` workflow runs benchmarks weekly with 110% regression alert threshold.

### Build Performance

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Local build time | < 10s | `time make build` |
| Full CI pipeline | < 10 min | GitHub Actions workflow duration |
| Cross-platform build | < 5 min | `time make build-all` |

## Security Metrics

### Vulnerability Counts

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Critical vulnerabilities (gosec) | 0 | `make security-scan` |
| High vulnerabilities (gosec) | 0 | CI artifact: `gosec-results.json` |
| Medium vulnerabilities (gosec) | < 5 | CI artifact: `gosec-results.json` |
| Known dependency CVEs | 0 critical, 0 high | `make vulnerability-check` |

**Collection commands:**
```bash
# Static security analysis
make security-scan

# Dependency vulnerability check
make vulnerability-check
```

### Secret Detection

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Secrets detected in pre-commit | 0 (after false-positive filtering) | Lefthook pre-commit hook |
| Secrets in full repository scan | 0 | `./scripts/detect-secrets.sh $(git ls-files)` |
| False positive rate | < 5% | Manual review of blocked commits |

### Dependency Freshness

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Go version | Current stable | `go version` vs latest release |
| Direct dependency age | < 6 months behind latest | `go list -u -m all` |
| Security patch lag | < 1 week for critical | `govulncheck ./...` |

### License Compliance

| Metric | Target | How to Measure |
|--------|--------|----------------|
| License header coverage | 100% of .go files | `./scripts/check-license-headers.sh` |
| Dependency license compatibility | All Apache 2.0 compatible | Manual review of go.sum |

## Operational Metrics

### Tool Coverage

| Metric | Current | Target | How to Measure |
|--------|---------|--------|----------------|
| Total evidence collection tools | 30 | Growing | `grctool tool --help | wc -l` |
| Evidence tasks automated | 90/105 (85%) [Source: Tugboat Logic sync data - verify with current sync] | 100/105 (95%) | `grctool tool evidence-task-list` |
| Tool categories covered | 5 (Terraform, GitHub, Google Workspace, Evidence, Utility) | 8+ | Tool inventory |

### Evidence Generation Success

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Evidence generation success rate | >=80% | Track success/failure across `grctool evidence generate` runs |
| Evidence quality score (evaluation) | >=90/100 | `grctool evidence evaluate ET-XXXX` |
| Auditor acceptance rate | >=90% | Post-audit feedback |

### Submission Success

| Metric | Target | How to Measure |
|--------|--------|----------------|
| Evidence submission success rate | >=95% | Track `grctool evidence submit` outcomes |
| Collector URL configuration rate | 100% of active tasks | Check `.grctool.yaml` `tugboat.collector_urls` |

## Metrics Collection and Reporting

### Automated Collection (CI)

The following metrics are collected automatically on every push to `main` and every PR:

| Metric | CI Job | Artifact |
|--------|--------|----------|
| Code coverage | `coverage` | `coverage.out`, `coverage.html` |
| Security findings | `security-scan` | `gosec-results.json`, `gosec-results.sarif` |
| Build success | `build-matrix` | `grctool-linux-amd64` binary |
| Lint status | `ci-checks` | CI logs |
| License compliance | `license-check` | CI logs |
| Quality gate status | `quality-gates` | CI logs |

### Weekly Collection (Scheduled)

The `testing.yml` advanced testing workflow runs weekly:

| Metric | CI Job | Artifact |
|--------|--------|----------|
| Deep coverage analysis | `coverage` | `advanced-coverage-reports` |
| Performance benchmarks | `benchmarks` | `benchmark-results` |
| Mutation testing | `mutation` | `mutation-report` |
| CI runtime baseline | `unit-tests` | CI logs (Go 1.24.12) |

### Manual Collection

| Metric | Command | When to Run |
|--------|---------|-------------|
| Vulnerability scan | `make vulnerability-check` | Before releases, after dependency updates |
| Comprehensive benchmarks | `make bench-report` | Before releases |
| Mutation baseline | `make mutation-baseline` | Monthly |
| Full E2E validation | `make test-e2e-comprehensive` | Before releases |

### Reporting Dashboard

To generate a comprehensive quality snapshot:

```bash
# Quick quality check
make ci

# Full quality report (requires all tools installed)
echo "=== Coverage ==="
make coverage-check

echo "=== Security ==="
make security-scan

echo "=== Benchmarks ==="
make bench

echo "=== Lint ==="
make lint

echo "=== Tests ==="
make test-no-auth

echo "=== Build ==="
make build-all
```

### Codebase Statistics

```bash
# Quick stats
make ai-stats

# Output:
# Go files: N
# Test files: N
# Packages: N
# Dependencies: N
# LOC: N
```

## System-of-Record Metrics

These metrics track progress toward GRCTool's evolution from a compliance data aggregator to the system of record for GRC data (see Product Vision in `01-frame/prd/requirements.md`). All metrics in this section are planned and will be instrumented as the system-of-record architecture is implemented.

### Master Index Coverage

| Metric | Current | Target | Status | How to Measure |
|--------|---------|--------|--------|----------------|
| Policies indexed locally | -- | 100% | Planned | Count of policies in master index vs. known external sources |
| Controls indexed locally | -- | 100% | Planned | Count of controls in master index vs. known external sources |
| Evidence tasks indexed locally | -- | 100% | Planned | Count of evidence tasks in master index vs. known external sources |
| Control mappings indexed locally | -- | 100% | Planned | Cross-reference completeness between controls and evidence tasks |

### Sync Health

| Metric | Current | Target | Status | How to Measure |
|--------|---------|--------|--------|----------------|
| Bidirectional sync success rate | -- | >=99% | Planned | Successful sync operations / total sync attempts |
| Conflict resolution rate | -- | >=95% automated | Planned | Automatically resolved conflicts / total conflicts detected |
| Sync latency (inbound) | -- | < 30s | Planned | Time from sync initiation to local index update |
| Sync latency (outbound) | -- | < 30s | Planned | Time from local change to external platform confirmation |
| Sync conflict backlog | -- | 0 unresolved | Planned | Count of conflicts awaiting manual resolution |

### Integration Adapter Metrics

| Metric | Current | Target | Status | How to Measure |
|--------|---------|--------|--------|----------------|
| Active integration adapters | 1 (Tugboat) | 3+ | Planned | Count of registered adapters in plugin registry |
| Per-adapter sync success rate | -- | >=95% per adapter | Planned | Success/failure tracking per adapter per sync cycle |
| Adapter health check pass rate | -- | 100% | Planned | Periodic adapter connectivity and auth validation |

### Data Quality

| Metric | Current | Target | Status | How to Measure |
|--------|---------|--------|--------|----------------|
| Schema validation pass rate | -- | 100% | Planned | Artifacts passing schema validation / total artifacts |
| Missing field detection rate | -- | < 1% missing required fields | Planned | Required fields present / total required fields across all artifacts |
| Cross-reference integrity | -- | 100% | Planned | Valid cross-references (e.g., control-to-policy links) / total cross-references |
| Duplicate artifact detection | -- | 0 duplicates | Planned | Deduplication check across master index |

### Data Governance

| Metric | Current | Target | Status | How to Measure |
|--------|---------|--------|--------|----------------|
| Audit trail completeness | -- | 100% of mutations logged | Planned | Master index mutations with audit log entries / total mutations |
| Unauthorized modification attempts | -- | 0 | Planned | Modifications bypassing the sanctioned write path |
| Data export compliance | -- | 100% | Planned | Exports adhering to configured data sovereignty rules / total exports |

## Metric Targets Summary

| Category | Metric | Current | Near-Term Target | Long-Term Target |
|----------|--------|---------|-----------------|-----------------|
| Quality | Code coverage | ~22% | 60% | >=80% |
| Quality | Lint issues | 4 | 0 | 0 |
| Quality | Complexity per function | Varies | <=15 | <=15 |
| Testing | Mutation score | ~59.7% | 70% | >=80% |
| Testing | Integration test pass rate | 83% | 100% | 100% |
| Performance | CLI response time (version) | Fast | < 100ms | < 100ms |
| Performance | Benchmark regression | None detected | < 110% of baseline | < 105% of baseline |
| Security | Critical vulnerabilities | 0 | 0 | 0 |
| Security | Secrets in codebase | 0 | 0 | 0 |
| Operational | Evidence task automation | 85% [Source: Tugboat Logic sync data - verify with current sync] | 90% | 95% |
| Operational | Tool count | 30 | 35 | 40+ |

---

*Dashboard specification last updated: 2026-03-17. Metrics should be reviewed monthly, with targets adjusted quarterly based on progress and priorities.*
