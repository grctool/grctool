---
title: "Comprehensive Testing Strategy"
phase: "03-test"
category: "testing"
tags: ["testing", "quality", "vcr", "functional", "e2e", "security"]
related: ["security-tests", "test-procedures"]
created: 2025-01-10
updated: 2026-04-01
helix_mapping: "Backfilled from AGENTS.md, Makefile, docs/04-Development/testing-structure.md, and test layout"
---

# Comprehensive Testing Strategy

## Overview

GRCTool uses a 4-tier testing strategy designed to separate fast no-auth
feedback from slower binary and authenticated coverage. The repository guidance
also adopts a classical, state-based testing style: prefer pure functions,
consumer-defined interfaces, simple stubs, testdata fixtures, and VCR
recordings before reaching for mocks.

## Testing Philosophy

- Prefer state-based tests over interaction-heavy mocks
- Extract pure functions where business logic can be isolated from I/O
- Define small interfaces in the consumer package and use simple stubs in tests
- Use VCR playback for external HTTP behavior instead of live API dependence in
  CI
- Keep auth-dependent testing behind explicit `e2e` build tags

## Test Tiers

### 1. Unit Tests

Fast, deterministic tests for `internal/...` and `cmd/...` with no external
authentication requirement.

**Primary command**

```bash
make test-unit
```

**Current execution**

```bash
go test -timeout=30s -v ./internal/... ./cmd/... -count=1
```

**Purpose**

- Pure logic validation
- Consumer-interface and stub-based package testing
- Fast regression feedback during development

### 2. Integration Tests

Integration coverage validates cross-package behavior and local orchestration
with VCR playback and local fixtures, without live API dependence.

**Primary command**

```bash
make test-integration
```

**Current execution**

```bash
VCR_MODE=playback go test -tags=integration -timeout=2m -v ./internal/... ./test/integration/... -count=1
```

**Purpose**

- VCR-backed Tugboat and related HTTP flows
- Tool orchestration and provider pipeline checks
- Local data and fixture-driven workflow validation

### 3. Functional Tests

Functional tests require a built binary and exercise the CLI end to end against
the filesystem and test fixtures.

**Primary command**

```bash
make test-functional
```

**Current execution**

```bash
go test -tags="functional" -timeout=5m -v ./test/functional/... -count=1
```

**Purpose**

- Built-binary CLI validation
- Config loading and path behavior
- Real command routing and output behavior

### 4. End-to-End Tests

E2E tests use real authentication and live external services. They are not the
default CI path and should skip cleanly when required auth is unavailable.

**Primary command**

```bash
make test-e2e
```

**Current execution**

```bash
go test -tags="e2e" -timeout=10m -v ./test/e2e/... -count=1
```

**Purpose**

- Real GitHub and Tugboat authentication paths
- Full workflow validation against live systems
- Authenticated behavior that cannot be proven with VCR alone

## Recommended Quality Commands

- `make test-no-auth`: unit + integration, the default fast feedback path
- `make test-all`: unit + integration + functional + e2e
- `make ci`: deps, fmt, vet, lint, test-no-auth, security-scan
- `make test-coverage`: unit-oriented coverage report
- `make test-coverage-all`: broader coverage pass

## Test Organization

### Primary Locations

- `cmd/*_test.go`, `internal/*/*_test.go`: unit-heavy package-local tests
- `test/integration/`: VCR-backed integration coverage
- `test/functional/`: functional CLI scenarios
- `test/e2e/`: authenticated live-service coverage
- `test/helpers/`, `internal/testhelpers/`: shared builders, stubs, and support
- `test/testdata/`, `test/fixtures/`, `test/sample_data/`, `test/vcr_cassettes/`:
  reusable fixtures and recordings
- `test-integration/`: additional local integration support assets

### Build Tags

- Default package tests cover unit-style behavior
- `//go:build functional` marks functional CLI tests
- `//go:build e2e` marks auth-dependent live-service tests
- Integration coverage is invoked via `-tags=integration`

## VCR Strategy

GRCTool uses VCR to record and replay HTTP interactions so CI and local
no-auth testing remain deterministic.

### Supported Modes

- `off`
- `record`
- `playback`
- `record_once`

### Current Expectations

- CI defaults to `VCR_MODE=playback`
- New endpoint coverage should add cassettes with redacted sensitive headers
- Redaction must cover authorization, cookie, api key, token, and password data

## Style Guide For New Tests

- Prefer table-driven tests, using maps where practical
- Use builders or fixture helpers for complex inputs
- Store substantial fixtures under `testdata/`
- Document why a mock is necessary if a mock is used
- Test behavior and outcomes rather than internal implementation sequencing

## Known Drift Addressed By This Backfill

- Earlier HELIX test docs mixed in a mock-heavy philosophy that no longer
  matches repository guidance.
- The current repo explicitly documents four tiers plus `make test-no-auth` and
  CI-friendly no-auth flows.
- Functional and e2e separation is now a first-class part of the project
  contract, not an optional convention.
