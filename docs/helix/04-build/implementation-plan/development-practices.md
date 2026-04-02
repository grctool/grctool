---
title: "Development Practices and Standards"
phase: "04-build"
category: "development"
tags: ["coding-standards", "go", "security", "testing", "logging"]
related: ["secure-coding", "build-procedures"]
created: 2025-01-10
updated: 2026-04-01
helix_mapping: "Backfilled from AGENTS.md, Makefile, and current repository conventions"
---

# Development Practices and Standards

## Overview

This document captures the build-phase rules that are currently enforced by the
repository instructions and supporting automation. The dominant implementation
stance is conservative and incremental: extend existing modules, keep changes
small and reversible, add tests alongside behavior changes, and prefer
deterministic local workflows over ad hoc shortcuts.

## Core Principles

- Prefer simple, composable solutions over new architectural patterns
- Design for change with small modules and explicit interfaces
- Wrap errors with context and avoid panics outside `main`
- Treat security, determinism, and auditability as default constraints
- Keep local and CI development paths reproducible

## Go Standards

### Package and Type Design

- Business logic belongs in `internal/*`; Cobra commands under `cmd/` should stay
  thin
- Define small interfaces where they are consumed
- Accept interfaces and return concrete implementations where practical
- Use descriptive type names and `NewXxx` constructors
- Remove obsolete code instead of introducing `Legacy`, `Old`, or parallel
  replacement variants

### Error Handling

- Use `fmt.Errorf("context: %w", err)` for contextual wrapping
- Define sentinel errors for expected conditions
- Return errors instead of panicking in library code
- Fail fast on invalid configuration and missing prerequisites

### Concurrency

- Pass `context.Context` through long-running and I/O-heavy operations
- Respect cancellation
- Avoid goroutine leaks
- Guard shared state explicitly and prefer immutable data where possible

## Logging and Diagnostics

- Internal packages should use `internal/logger`, not `fmt.Println`
- CLI commands may print user-facing output, but diagnostics still belong in the
  logger path
- Redact secrets such as authorization headers, cookies, tokens, passwords, and
  API keys
- Include structured context such as component, operation, and duration where it
  materially aids debugging

## Testing Expectations During Implementation

- Any behavior change should ship with updated tests in the same change
- Prefer pure functions, stubs, fixtures, and VCR coverage before mocks
- Use the 4-tier test structure: unit, integration, functional, e2e
- Default fast feedback command: `make test-no-auth`
- Recommended pre-push quality gate: `make fmt lint vet test-no-auth`

## Configuration and Secrets

- Configuration lives in YAML and supports environment substitution for secrets
- Do not commit credentials
- Prefer environment variables or keychain-backed flows for sensitive data
- Validate configuration on startup with actionable errors

## HTTP and External Integrations

- Centralize HTTP behavior in the relevant client packages, especially
  `internal/tugboat`
- Use retries with backoff for transient failures
- Respect rate limits and add jitter where appropriate
- Never log raw secrets or sensitive request payloads

## Refactoring Stance

- Modify code in place rather than creating `_new`, `_legacy`, or duplicate
  replacement files
- Remove stale branches when product constraints do not require backward
  compatibility
- Keep code, tests, and docs aligned in the same change
- If behavior changes, update the governing HELIX artifacts and tracker state

## Operational Build Rules

- Formatting: `make fmt`
- Linting: `make lint`
- Static analysis: `make vet`
- Fast no-auth quality pass: `make test-no-auth`
- Broader local validation may include `make test-functional`,
  `make test-integration`, and release/security targets as needed

## Security Expectations

- Practice least privilege in all integrations
- Review evidence outputs before submission
- Keep structured logging redaction current
- Run security and vulnerability checks before release work

## Known Drift Addressed By This Backfill

- Earlier HELIX build docs contained generic example code and package names that
  do not match this repository.
- Current repo policy is more explicit about consumer-defined interfaces,
  in-place refactoring, deterministic test tiers, and structured logging than
  the older draft reflected.
