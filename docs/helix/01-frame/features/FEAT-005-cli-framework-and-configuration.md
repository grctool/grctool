---
title: "FEAT-005: CLI Framework & Configuration Management"
phase: "01-frame"
category: "features"
tags: ["cli", "cobra", "configuration", "viper", "yaml", "backfill"]
related: ["prd", "FEAT-006", "FEAT-007"]
status: "Implemented"
priority: "P0"
created: 2026-04-01
updated: 2026-04-01
backfill: true
---

# FEAT-005: CLI Framework & Configuration Management

## Status

| Field | Value |
|-------|-------|
| Status | Implemented |
| Priority | P0 |
| Implementation | `cmd/`, `internal/config/` |

---

## Overview

GRCTool's foundational CLI framework provides the command structure, configuration management, and runtime environment for all compliance operations. Built on Cobra for command routing and Viper for configuration, it delivers a scriptable, composable interface that integrates with developer workflows.

## Problem Statement

Compliance teams need a tool that fits into existing developer and DevOps workflows — not another web portal. The tool must be scriptable for CI/CD integration, configurable per-project, and extensible as new compliance capabilities are added. Configuration must support sensitive values (API keys, tokens) via environment variables while keeping non-sensitive settings in version-controlled YAML.

---

## Requirements

### Functional Requirements

- FR-001: Hierarchical command structure with `grctool <group> <command>` pattern
- FR-002: Persistent configuration via `.grctool.yaml` with initialization command
- FR-003: Environment variable substitution for sensitive configuration values (e.g., `${GITHUB_TOKEN}`)
- FR-004: Configuration validation with connectivity checks
- FR-005: Redacted configuration display (mask tokens, keys, passwords)
- FR-006: Structured logging with configurable levels, outputs, and field redaction
- FR-007: Global flags (verbosity, output format) available to all commands
- FR-008: Help text and usage documentation for every command and subcommand
- FR-009: Version information with build metadata (version, commit, build time)
- FR-010: Template variable interpolation for organization-specific values in policies and controls

### Non-Functional Requirements

- **Performance**: CLI commands start and produce initial output within 1 second
- **Security**: Sensitive fields (password, token, key, secret, api_key, cookie) are redacted in all log output; config files use 0600 permissions
- **Reliability**: Configuration initialization is idempotent — safe to run multiple times
- **Usability**: Commands follow POSIX conventions; help text is discoverable via `--help` on any command

---

## User Stories

### US-051: Initialize Project Configuration [FEAT-005]

**As a** compliance engineer setting up GRCTool for the first time,
**I want to** run a single command to initialize configuration with sensible defaults
**so that** I can start using the tool without manually creating config files.

**Acceptance Criteria:**

- [x] `grctool config init` creates `.grctool.yaml` with documented defaults
- [x] Init is idempotent — running multiple times does not overwrite existing configuration
- [x] Init generates a `CLAUDE.md` guide file for AI assistant context
- [x] Default configuration includes all supported sections with commented examples

### US-052: Configure Evidence Tools [FEAT-005]

**As a** security engineer,
**I want to** configure which evidence tools are enabled and their connection parameters
**so that** evidence generation uses the correct infrastructure sources.

**Acceptance Criteria:**

- [x] Terraform tool paths, include/exclude patterns are configurable
- [x] GitHub token, repository, rate limits are configurable
- [x] Google Workspace credentials and shared drive ID are configurable
- [x] Tools can be individually enabled/disabled
- [x] Sensitive values support `${ENV_VAR}` substitution

### US-053: Validate Configuration [FEAT-005]

**As a** DevOps engineer,
**I want to** validate my configuration before running operations
**so that** I catch misconfigurations early rather than during evidence collection.

**Acceptance Criteria:**

- [x] `grctool config validate` checks YAML syntax
- [x] Validates required fields are present and non-empty
- [x] Reports configuration errors with actionable messages
- [x] `grctool config show` displays current configuration with redacted sensitive values

### US-054: Template Variable Interpolation [FEAT-005]

**As a** compliance manager,
**I want** organization-specific template variables automatically substituted in policy documents
**so that** generated evidence uses our actual organization name and contact information.

**Acceptance Criteria:**

- [x] `{{organization.name}}` and `{{Organization}}` are replaced in policy/control content
- [x] Custom variables are configurable in `.grctool.yaml` under `interpolation.variables`
- [x] Interpolation can be globally disabled via `interpolation.enabled: false`
- [x] Variables are substituted during evidence generation and markdown rendering

### US-055: Structured Logging [FEAT-005]

**As a** developer debugging a failed evidence generation,
**I want** structured, leveled logging with sensitive field redaction
**so that** I can diagnose issues without exposing credentials in logs.

**Acceptance Criteria:**

- [x] Zerolog-based structured logging with trace/debug/info/warn/error levels
- [x] Multiple logger outputs (console, file) with independent level configuration
- [x] URL sanitization removes tokens and credentials from logged URLs
- [x] Configurable field redaction for password, token, key, secret, api_key, cookie
- [x] Component loggers via `logger.WithComponent(name)` for subsystem tracing

---

## Edge Cases and Error Handling

- Missing `.grctool.yaml`: Commands that require configuration fail with a clear message directing the user to run `grctool config init`
- Malformed YAML: Parse errors include line numbers and field context
- Undefined environment variables in `${VAR}` substitutions: Logged as warning, value left as empty string
- Conflicting flag and config values: CLI flags take precedence over config file values

---

## Success Metrics

| Metric | Target | Status |
|--------|--------|--------|
| Command discoverability | Every command has `--help` with examples | Met |
| Config initialization time | < 1 second | Met |
| Sensitive field redaction | Zero credentials in log output | Met |
| Template variable coverage | 408+ substitution points across 40 policies | Met |

---

## Dependencies

- **Cobra** (spf13/cobra): CLI command framework
- **Viper** (spf13/viper): Configuration management
- **Zerolog** (rs/zerolog): Structured logging

---

## Out of Scope

- GUI or web-based configuration interface
- Remote/cloud configuration storage
- Multi-user configuration profiles
- Interactive configuration wizard (beyond `grctool config init`)

---

## Traceability

### Related Artifacts
- **Parent PRD Section**: Core Capabilities — Configuration, Logging
- **Design Artifacts**: `internal/config/`, `internal/logger/`
- **Implementation**: `cmd/root.go`, `cmd/config.go`, `cmd/init.go`, `internal/config/`, `internal/logger/`

### Feature Dependencies
- **Depends On**: None (foundational)
- **Depended By**: FEAT-006, FEAT-007, FEAT-008, FEAT-009, FEAT-010, FEAT-001, FEAT-002, FEAT-003, FEAT-004

---
*Backfill spec: documents functionality that is already implemented in the codebase.*
