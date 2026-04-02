---
title: "FEAT-007: Tugboat Logic Data Synchronization"
phase: "01-frame"
category: "features"
tags: ["sync", "tugboat", "policies", "controls", "evidence-tasks", "backfill"]
related: ["FEAT-005", "FEAT-006", "FEAT-008", "prd"]
status: "Implemented"
priority: "P0"
created: 2026-04-01
updated: 2026-04-01
backfill: true
---

# FEAT-007: Tugboat Logic Data Synchronization

## Status

| Field | Value |
|-------|-------|
| Status | Implemented |
| Priority | P0 |
| Implementation | `internal/services/sync.go`, `internal/tugboat/`, `internal/storage/` |

---

## Overview

GRCTool synchronizes compliance data — policies, controls, and evidence tasks — from Tugboat Logic's REST API into a local filesystem-based data store. Sync provides the foundational data layer that all evidence generation, analysis, and submission operations depend on. Data is stored as human-readable JSON files with a deterministic naming convention, cached for performance, and designed for version control via git.

## Problem Statement

Compliance data lives in Tugboat Logic's web application, but compliance engineers need this data locally for automated evidence generation, infrastructure mapping, and offline analysis. Without automated sync, teams manually export data or work from stale copies, leading to evidence generated against outdated controls, missed evidence tasks, and inconsistent policy references. Sync must be reliable, incremental, and produce deterministic output suitable for git tracking.

---

## Requirements

### Functional Requirements

- FR-001: Full sync of policies, controls, and evidence tasks from Tugboat Logic API
- FR-002: Incremental sync detecting changes since last successful sync
- FR-003: Selective sync by resource type (`--policies`, `--controls`, `--evidence`)
- FR-004: Force sync ignoring cache (`--force`)
- FR-005: Deterministic file naming: `{ReferenceID}_{NumericID}_{SanitizedName}.json`
- FR-006: Sync metadata tracking (last sync time, entity hashes, provider source)
- FR-007: Sync summary with statistics (new, updated, unchanged, errors)
- FR-008: Sync validation checking data integrity post-sync
- FR-009: Rate limiting with configurable requests-per-second
- FR-010: Relationship preservation between policies, controls, and evidence tasks
- FR-011: VCR (HTTP recording/playback) support for deterministic testing

### Non-Functional Requirements

- **Performance**: Full sync of 200+ policies, 1000+ controls, 500+ evidence tasks completes within 5 minutes; incremental sync within 30 seconds
- **Reliability**: >99% success rate with automatic retries on transient failures; partial sync failures do not corrupt previously synced data
- **Storage**: ~50MB for typical organization; JSON files optimized for git diff readability
- **Security**: Bearer token transmitted only over HTTPS; credentials never included in synced data files

---

## User Stories

### US-071: Full Data Sync [FEAT-007]

**As a** compliance manager,
**I want to** sync all compliance data from Tugboat Logic with a single command
**so that** I have a complete local copy of policies, controls, and evidence tasks.

**Acceptance Criteria:**

- [x] `grctool sync` downloads all policies, controls, and evidence tasks
- [x] Each entity is stored as a separate JSON file in the data directory
- [x] Files use deterministic naming: `{ReferenceID}_{NumericID}_{SanitizedName}.json`
- [x] Sync reports total entities synced, time elapsed, and any errors
- [x] Synced data includes all metadata: assignments, tags, frameworks, relationships

### US-072: Incremental Sync [FEAT-007]

**As a** security engineer running daily syncs,
**I want** incremental sync that only fetches changed data
**so that** routine syncs are fast and generate minimal git diffs.

**Acceptance Criteria:**

- [x] `grctool sync --incremental` fetches only entities changed since last sync
- [x] Change detection uses content hashing to identify modifications
- [x] Unchanged files are not rewritten (preserving git status)
- [x] Incremental sync completes in seconds for typical change volumes

### US-073: Selective Resource Sync [FEAT-007]

**As a** DevOps engineer,
**I want to** sync only specific resource types
**so that** I can update just the evidence tasks without re-syncing all policies.

**Acceptance Criteria:**

- [x] `grctool sync --policies` syncs only policies
- [x] `grctool sync --controls` syncs only controls
- [x] `grctool sync --evidence` syncs only evidence tasks
- [x] Flags can be combined for multi-type selective sync

### US-074: Sync Validation [FEAT-007]

**As a** compliance manager,
**I want to** validate the integrity of synced data
**so that** I know the local data accurately reflects Tugboat Logic.

**Acceptance Criteria:**

- [x] `grctool sync validate` checks synced file integrity
- [x] Validates entity counts against API totals
- [x] Reports missing, extra, or corrupted files
- [x] `grctool sync summary` shows sync statistics and timing

### US-075: View Policies and Controls [FEAT-007]

**As a** compliance engineer,
**I want to** browse and view synced policies and controls from the CLI
**so that** I can understand my compliance posture without opening the Tugboat web UI.

**Acceptance Criteria:**

- [x] `grctool policy list` lists all synced policies with reference IDs and names
- [x] `grctool policy view <policy-id>` displays full policy details in markdown
- [x] `grctool control list` lists all synced controls with categories and frameworks
- [x] `grctool control view <control-id>` displays control details with related evidence tasks
- [x] `grctool evidence list` lists evidence tasks with filtering (status, framework, assignee, etc.)
- [x] `grctool evidence view <task-id>` displays task details with requirements and guidance

---

## Edge Cases and Error Handling

- Authentication expired during sync: Fail with clear re-auth message; do not partially overwrite files
- Rate limit exceeded: Respect Tugboat API rate limit headers; exponential backoff on 429 responses
- Network interruption: Resume capability for large syncs; partial results preserved
- Duplicate reference IDs from API: Deterministic deduplication based on numeric ID
- Special characters in entity names: Sanitized to filesystem-safe characters in filenames
- Empty API responses: Treated as "no entities" rather than an error; logged as warning

---

## Success Metrics

| Metric | Target | Status |
|--------|--------|--------|
| Full sync reliability | >99% success rate | Met |
| Incremental sync time | < 30 seconds for typical delta | Met |
| Data completeness | 100% of Tugboat entities present locally after sync | Met |
| File naming determinism | Identical sync runs produce identical filenames | Met |
| VCR test coverage | All API interactions recorded for deterministic testing | Met |

---

## Dependencies

- **FEAT-005**: Configuration system (data directory, Tugboat base URL, rate limits)
- **FEAT-006**: Authentication (bearer token for API access)
- **Tugboat Logic API**: REST endpoints for policies, controls, evidence tasks

---

## Out of Scope

- Bidirectional sync (pushing data to Tugboat) — see FEAT-001, FEAT-004
- Real-time streaming or webhook-based sync
- Sync with non-Tugboat compliance platforms — see FEAT-004
- Conflict resolution for concurrent modifications — see FEAT-004
- Evidence file synchronization (only metadata is synced)

---

## Traceability

### Related Artifacts
- **Parent PRD Section**: Core Capabilities — Data Synchronization
- **Implementation**: `internal/services/sync.go`, `internal/tugboat/`, `internal/storage/`, `cmd/sync.go`

### Feature Dependencies
- **Depends On**: FEAT-005 (configuration), FEAT-006 (authentication)
- **Depended By**: FEAT-008 (tools need synced data), FEAT-009 (evidence generation uses synced tasks/controls/policies), FEAT-010 (submission references synced tasks)

---
*Backfill spec: documents functionality that is already implemented in the codebase.*
