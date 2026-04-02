---
title: "FEAT-001: AccountableHQ Bidirectional Policy Sync"
phase: "01-frame"
category: "features"
tags: ["accountablehq", "policy-sync", "bidirectional", "integration", "system-of-record"]
related: ["adr-010", "data-design", "contracts", "SD-001"]
status: "Proposed"
priority: "P1"
created: 2026-03-17
updated: 2026-03-17
---

# FEAT-001: AccountableHQ Bidirectional Policy Sync

## Status

| Field | Value |
|-------|-------|
| Status | Proposed |
| Priority | P1 |
| Owner | TBD |
| Target Phase | Phase 2 (Dual-Write) / Phase 3 (GRCTool-as-Source) |

---

## Problem Statement

Organizations using AccountableHQ for policy management need their policies synchronized with GRCTool's master index without manual import/export. Today, compliance teams maintain policies in AccountableHQ and separately manage compliance artifacts in GRCTool, leading to drift between systems, duplicated effort, and ambiguous source-of-truth ownership. As GRCTool transitions to system of record (ADR-010), AccountableHQ must become a bidirectional integration target -- policies flow in both directions with conflict detection, audit trails, and configurable automation.

---

## User Stories

### US-001: Import Policies from AccountableHQ

**As a** compliance manager,
**I want to** import all policies from AccountableHQ into GRCTool's master index
**so that** I have a single source of truth for all compliance policies.

**Acceptance Criteria**:

- Import retrieves all policies from the AccountableHQ policy repository via API
- Each imported policy is assigned a GRCTool canonical ID (POL-NNNN) and mapped to its AccountableHQ external ID in the master index
- Import is idempotent: re-running import updates existing entries rather than creating duplicates
- Policy metadata (title, version, owner, approval status, last modified) is preserved during import
- Import produces a summary report: new policies added, existing policies updated, policies unchanged
- Imported policies are stored in the local filesystem in human-readable format (JSON/Markdown) consistent with existing policy storage conventions

### US-002: Sync Changes Back to AccountableHQ

**As a** compliance manager,
**I want to** have changes made in GRCTool sync back to AccountableHQ
**so that** both systems stay current without manual re-entry.

**Acceptance Criteria**:

- Policies edited locally in GRCTool are pushed to AccountableHQ via the export adapter
- Export updates only the fields that changed, not the entire policy document
- Export respects AccountableHQ's API rate limits and handles throttling gracefully
- Export records the resulting AccountableHQ version/timestamp in the master index sync state
- A dry-run mode shows what would be pushed without making changes
- Export failures for individual policies do not block the remaining policies in the batch

### US-003: Conflict Detection on Simultaneous Edits

**As a** security engineer,
**I want** conflict detection when policies are edited in both systems simultaneously
**so that** no changes are silently lost.

**Acceptance Criteria**:

- Sync compares content hashes and timestamps between GRCTool and AccountableHQ for each policy
- When both sides have changed since the last sync, the policy is flagged as a conflict
- Conflict resolution policy is configurable per-policy or globally: `local_wins`, `remote_wins`, `manual`, `newest_wins` (consistent with ADR-010 conflict policies)
- `manual` conflicts are surfaced in CLI output with a diff showing both versions
- Conflict resolution decisions are recorded in the audit trail
- No data is overwritten without explicit policy configuration or user confirmation

### US-004: Audit Trail for Sync Operations

**As a** CISO,
**I want** an audit trail of all sync operations showing what changed, when, and in which direction
**so that** I can demonstrate compliance data integrity to auditors.

**Acceptance Criteria**:

- Every sync operation (import, export, bidirectional) produces a structured log entry
- Log entries include: timestamp, direction (inbound/outbound/bidirectional), policies affected, fields changed, conflict resolution decisions, operator identity
- Audit trail is stored locally in a git-friendly format (YAML or JSON) under the data directory
- Audit trail entries are append-only; previous entries are never modified
- `grctool sync audit-log` (or equivalent) can query and display sync history with filtering by date range, direction, and policy ID
- Audit trail is sufficient for SOC2 evidence of change management controls

### US-005: Scheduled Automatic Sync

**As a** DevOps engineer,
**I want to** schedule automatic sync on a configurable cadence
**so that** policies stay synchronized without manual intervention.

**Acceptance Criteria**:

- Sync cadence is configurable in `.grctool.yaml` (e.g., every 6 hours, daily, weekly)
- Scheduled sync integrates with the GRCTool task scheduler (FEAT-003)
- Scheduled sync uses the configured conflict resolution policy (no interactive prompts)
- Sync failures produce alerts via configured notification channels (log, webhook)
- A lock mechanism prevents concurrent sync operations from overlapping
- `grctool sync status` shows the last sync time, next scheduled sync, and any pending conflicts

---

## Dependencies

| Dependency | Type | Status | Notes |
|------------|------|--------|-------|
| ADR-010: System of Record Architecture | Architecture | Accepted | Defines the master index, adapter interfaces, and conflict resolution model |
| Master Index Implementation | Technical | Not started | FEAT-001 requires the master index and GRCTool-native ID assignment. **NOTE: Per SD-004 and ADR-011, the master index is implemented as the existing `StorageService` enriched with `ExternalIDs` and `SyncMetadata` fields. No separate `.index/` directory is needed.** |
| ADR-006: Hexagonal Architecture | Architecture | Accepted | AccountableHQ adapter implements the standard port/adapter pattern |
| FEAT-003: Task Scheduler | Feature | Proposed | Required for US-005 (scheduled sync) |
| AccountableHQ API Access | External | TBD | API documentation, authentication credentials, and rate limit details needed |

---

## Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| AccountableHQ API changes or deprecation | Medium | High | Version-pin API calls; adapter abstraction isolates changes; VCR tests capture API behavior |
| Data model mismatches between AccountableHQ and GRCTool | Medium | Medium | Map AccountableHQ fields to GRCTool's policy model with explicit transformation layer; unmappable fields stored as extension metadata |
| Conflict resolution complexity overwhelms users | Medium | Medium | Default to `newest_wins` for automated sync; surface conflicts clearly in CLI; provide `--dry-run` for all sync operations |
| AccountableHQ rate limits block bulk import | Low | Medium | Implement exponential backoff and batch chunking; respect rate limit headers; allow configurable concurrency |
| Policy content drift during migration period | Medium | Medium | Phased rollout: import-only first, then enable bidirectional; shadow mode validates sync before going live |
| Authentication token expiry during long sync | Low | Low | Token refresh logic in adapter; clear error message on auth failure with re-auth instructions |

---

## Out of Scope

- AccountableHQ control or evidence task sync (policy sync only in FEAT-001)
- AccountableHQ user/role synchronization
- Real-time push notifications from AccountableHQ (webhook-based sync)
- Multi-tenant AccountableHQ configurations

---

## Success Metrics

| Metric | Target | Measurement |
|--------|--------|-------------|
| Policy import completeness | 100% of AccountableHQ policies present in master index after import | Compare AccountableHQ policy count with master index count |
| Round-trip fidelity | Zero data loss on import-then-export cycle | Content hash comparison after round-trip |
| Conflict detection accuracy | All concurrent edits detected; zero silent overwrites | Integration test suite with concurrent edit scenarios |
| Sync duration | < 60 seconds for 100 policies | End-to-end timing of bidirectional sync |
| Audit trail completeness | Every sync operation has a corresponding audit entry | Audit log entry count matches sync operation count |

---

## References

- [ADR-010: System of Record Architecture](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-010-system-of-record-architecture)
- [Data Design: System of Record Architecture](/home/erik/Projects/grctool/docs/helix/02-design/data-design/data-design.md#system-of-record-architecture)
- [SD-001: AccountableHQ Adapter Solution Design](/home/erik/Projects/grctool/docs/helix/02-design/solution-designs/SD-001-accountablehq-adapter.md)
- [AccountableHQ API Discovery Reference](/home/erik/Projects/grctool/docs/helix/02-design/research/accountablehq-api-discovery.md)
- [AccountableHQ Scraping Evaluation Reference](/home/erik/Projects/grctool/docs/helix/02-design/research/accountablehq-scraping-evaluation.md)
- [Interface Contracts](/home/erik/Projects/grctool/docs/helix/02-design/contracts/contracts.md)
