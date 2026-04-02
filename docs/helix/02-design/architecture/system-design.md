---
title: "System Architecture and Design"
phase: "02-design"
category: "architecture"
tags: ["architecture", "design", "go", "cli", "providers", "tools", "storage"]
related: ["contracts", "data-design", "security-architecture"]
created: 2025-01-10
updated: 2026-04-01
helix_mapping: "Backfilled from README.md, docs/ARCHITECTURE.md, cmd/, internal/, and test layout"
---

# System Architecture and Design

## Architecture Overview

GRCTool is a Go 1.24 CLI for compliance automation around Tugboat Logic data,
evidence generation, and evidence collection tooling. The current runtime shape
is a layered CLI application with provider and tool registries rather than a
strict service mesh: Cobra commands initialize configuration and logging, then
dispatch into internal packages for auth, sync, storage, tool execution, and
evidence workflows.

## Current Runtime Shape

```text
main.go
  -> cmd.Execute()
    -> Cobra command tree in cmd/
      -> config + logging bootstrap
      -> provider registry bootstrap
      -> tool registry bootstrap
      -> command handlers
        -> internal/auth
        -> internal/tugboat
        -> internal/storage
        -> internal/services
        -> internal/tools
        -> internal/providers
```

### Architectural Characteristics

- CLI-first application entrypoint via `main.go` and `cmd/root.go`
- Configuration loaded from `.grctool.yaml` with environment overrides via Viper
- Structured logging initialized at command startup, with separate console and
  file behavior
- Provider registry for compliance-system integrations and sync-capable
  backends
- Tool registry for evidence collection commands with standardized JSON output
- Filesystem-backed JSON persistence and cache management under configured data
  and cache directories
- Test surface split into unit, integration, functional, and e2e tiers

## Repository Structure

### Command Surface

- `main.go`: injects build metadata and delegates to Cobra execution
- `cmd/`: root command, auth, sync, evidence, policy, control, schedule,
  tooling, validation, versioning, and update workflows

### Core Internal Domains

- `internal/auth`: browser-based Tugboat authentication, Safari automation,
  token extraction, provider support
- `internal/config`: configuration schema, defaults, loading, validation, path
  resolution
- `internal/providers`: provider factory and registry abstractions
- `internal/tugboat`: Tugboat HTTP client and resource-specific operations
- `internal/storage`: local JSON stores, cache handling, submission history, and
  unified storage
- `internal/services`: higher-level orchestration for sync, evidence, document
  generation, migration, and scanning
- `internal/tools`: evidence tool interfaces, registry, concrete Terraform,
  GitHub, Google Workspace, and evidence-management tools
- `internal/models` and `internal/domain`: data transfer and domain-oriented
  types used across commands and services
- `internal/logger` and `internal/transport`: structured logging and HTTP
  transport concerns

### Supporting Surfaces

- `configs/`: example configuration
- `docs/`: user, reference, development, and HELIX documentation
- `test/`, `test-integration/`: functional, integration, e2e, helpers, and
  fixtures
- `.github/workflows/` and `scripts/`: CI, release, recording, quality, and
  maintenance automation

## Key Runtime Flows

### Authentication

1. `grctool auth login` enters via Cobra command handlers.
2. `internal/auth` opens Safari and captures cookies or bearer state.
3. Credentials are validated against Tugboat before persistence.
4. Configuration storage persists auth state for later sync and submission
   operations.

### Sync

1. Sync commands load config and the provider registry.
2. The provider registry selects a sync-capable provider.
3. Tugboat client calls fetch policies, controls, evidence tasks, and related
   metadata.
4. Storage packages write deterministic JSON artifacts and sync metadata.

### Tool Execution

1. `grctool tool ...` builds a tool context with correlation ID, config,
   validator, logger, and JSON output writer.
2. The global tool registry resolves the requested tool.
3. The tool executes and returns structured output plus evidence source
   metadata.
4. Command wrappers emit JSON envelopes for human or AI consumers.

### Evidence Lifecycle

1. Evidence commands read synced compliance data and task context.
2. Tool orchestration optionally collects supplemental infrastructure evidence.
3. Services assemble prompts and generate or evaluate evidence outputs.
4. Submission-related code validates and sends outputs to Tugboat collector
   endpoints while storing submission history locally.

## Architectural Patterns In Use

### Registry-Based Extensibility

- Providers are registered behind consumer-facing interfaces and discovered by
  name.
- Evidence tools are registered centrally and enumerated for CLI and AI usage.
- This gives GRCTool one bounded place for capability discovery without
  hard-coding every workflow into each command.

### Consumer-Defined Interfaces

The repository guidance favors small, consumer-defined interfaces. Concrete
registries and services generally accept interfaces but return concrete
implementations where practical.

### Thin CLI, Deeper Internal Packages

Command files own flag parsing, help, and human-facing errors. Business logic
and I/O coordination live under `internal/`.

### Local-First Persistence

Compliance entities, prompts, evidence outputs, caches, and submission history
are stored on disk in deterministic formats that are intended to be auditable
and git-friendly.

## Operational Constraints

- Browser-driven Tugboat authentication is Safari-specific on macOS for the
  automated flow.
- Tugboat is the primary live integration today; provider abstractions indicate
  broader ambitions but the current repository is still centered on Tugboat.
- Logging must redact secrets and keep diagnostics out of stdout for internal
  packages.
- All network I/O should use bounded timeouts, retries, and rate limits through
  centralized clients.
- Existing code and tests are evidence of behavior, but higher-order HELIX docs
  govern intended architecture.

## Known Drift Addressed By This Backfill

- Earlier architecture docs described package paths that no longer exist, such
  as nested `cmd/auth/` or `internal/claude/`.
- The current codebase uses registries and broad internal package groupings more
  heavily than the older, simplified layered diagram suggested.
- Testing and delivery architecture now depend on explicit 4-tier testing,
  VCR-backed integration coverage, and richer CI/release automation.
