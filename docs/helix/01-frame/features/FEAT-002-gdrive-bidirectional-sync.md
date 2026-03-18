---
title: "FEAT-002: Google Drive Bidirectional Sync"
phase: "01-frame"
category: "feature"
status: "Proposed"
priority: "P1"
tags: ["google-drive", "bidirectional-sync", "integration", "system-of-record"]
related: ["adr-010", "adr-006", "data-design", "SD-002"]
created: 2026-03-17
updated: 2026-03-17
---

# FEAT-002: Google Drive Bidirectional Sync

## Summary

| Field | Value |
|-------|-------|
| Status | Proposed |
| Priority | P1 |
| Owner | TBD |
| Dependencies | ADR-010 (System of Record), Google Workspace OAuth (existing), Master Index |
| Related | SD-002-gdrive-sync-adapter, FEAT-003 (Task Scheduler) |

## Problem Statement

Compliance teams collaborate on policies and controls in Google Docs and Sheets. GRCTool is evolving to become the system of record for compliance data (ADR-010), but the current Google Workspace integration (`google_workspace.go`) is read-only -- it extracts evidence from Docs, Sheets, Drive, and Forms but cannot write back.

This creates a gap: policies and controls managed in GRCTool's master index cannot be published to Google Drive for collaborative review, and edits made by compliance team members in Google Docs are invisible to GRCTool. Organizations need their compliance artifacts accessible in the tools their teams already use, while maintaining GRCTool as the authoritative source.

Without bidirectional sync, teams face:

- **Manual duplication**: Copying policy content from GRCTool into Google Docs for team review, then manually copying edits back.
- **Version drift**: Google Docs and GRCTool diverge silently, creating compliance risk when outdated versions are referenced.
- **Limited collaboration**: Auditors, executives, and non-technical stakeholders cannot access compliance artifacts in their preferred tools.
- **Operational friction**: No automated way to keep external representations current when the master index changes.

## User Stories

### US-1: Publish policies to Google Drive as formatted Docs

**As a** compliance manager,
**I want to** publish all policies from GRCTool to Google Drive as formatted Google Docs,
**so that** my team can review them collaboratively using familiar commenting and suggestion tools.

**Acceptance criteria:**
- Policies from the master index are converted from Markdown to Google Docs structured content via the Docs API.
- Template variables (e.g., `{{organization.name}}`) are interpolated before publishing.
- Each policy creates or updates a single Google Doc; the mapping is stored in the master index.
- Published Docs preserve heading structure, lists, tables, and inline formatting from the Markdown source.

### US-2: Export control matrices as Google Sheets

**As a** compliance manager,
**I want to** export control matrices as Google Sheets,
**so that** stakeholders can view compliance status in a familiar spreadsheet format.

**Acceptance criteria:**
- Controls are exported as rows in a Google Sheet with columns for: control ID, title, description, framework(s), status, mapped policies, mapped evidence tasks.
- Cross-framework control mappings render as additional framework columns (e.g., SOC2, ISO27001).
- The Sheet updates in place on subsequent syncs rather than creating duplicates.
- Conditional formatting highlights controls by status (implemented, partially implemented, not implemented).

### US-3: Sync Google Docs edits back to GRCTool

**As a** policy author,
**I want** edits made in Google Docs to sync back to GRCTool's master index,
**so that** changes made during collaborative review are not lost and the master index stays current.

**Acceptance criteria:**
- GRCTool detects changes in published Google Docs by comparing Drive revision IDs against the last-synced revision.
- Changed Docs content is converted back to Markdown and written to the local policy file.
- Conflicts (both local and remote changed since last sync) are detected and surfaced to the user with diff context.
- Conflict resolution follows the policy configured in `.grctool.yaml` (local_wins, remote_wins, manual, newest_wins).

### US-4: Read-only auditor access via shared Drive folder

**As an** auditor,
**I want** a read-only Drive folder containing current evidence and policies shared with me for review,
**so that** I can conduct my review without requesting access to GRCTool or internal systems.

**Acceptance criteria:**
- A designated "auditor" folder in Google Drive contains the latest published policies and evidence summaries.
- The folder is configurable and can be shared with external email addresses at the Viewer permission level.
- Content in the auditor folder is refreshed on each sync cycle.
- Sensitive or draft policies can be excluded from the auditor folder via configuration.

### US-5: Selective sync scope

**As a** CISO,
**I want to** control which policies are mirrored to Drive and which remain local-only,
**so that** I can keep draft, internal-only, or sensitive policies out of Google Drive.

**Acceptance criteria:**
- `.grctool.yaml` supports include/exclude patterns for Drive sync (by policy ID, tag, or status).
- Default behavior is opt-in: no policies are synced unless explicitly configured.
- A `--dry-run` flag previews what would be synced without making changes.
- Sync scope changes are logged for audit trail purposes.

### US-6: Scheduled and on-demand sync

**As a** DevOps engineer,
**I want** sync to run on a configurable schedule and on-demand,
**so that** Drive content stays current without manual intervention.

**Acceptance criteria:**
- `grctool drive sync` runs a full bidirectional sync cycle on demand.
- `grctool drive sync --direction outbound` and `--direction inbound` support one-directional sync.
- Scheduler integration (FEAT-003) supports cron-style scheduling for Drive sync.
- Sync operations are idempotent; running sync twice with no changes produces no side effects.
- Sync status and last-sync timestamps are recorded in the master index.

## Acceptance Criteria (Feature-Level)

| # | Criterion | Verification |
|---|-----------|--------------|
| AC-1 | Policies published to Google Drive render as properly formatted Google Docs with headings, lists, and tables intact | Manual review of 5+ published policies |
| AC-2 | Control matrices render as Google Sheets with correct column structure and conditional formatting | Manual review of exported Sheet |
| AC-3 | Round-trip fidelity: a policy published to Docs and synced back produces semantically equivalent Markdown | Automated test comparing original and round-tripped content |
| AC-4 | Conflicts are detected when both local and remote changes exist since last sync | Integration test with simultaneous edits |
| AC-5 | Sync scope filtering correctly includes/excludes policies based on configuration | Unit tests for filter logic |
| AC-6 | Google API rate limits are respected; sync does not exceed 300 req/min (Docs) or 12,000 req/day (Drive) | Load test against API quota simulator |
| AC-7 | Sync operations are idempotent with no side effects on repeated runs | Integration test running sync twice |
| AC-8 | Existing read-only Google Workspace evidence collection is unaffected | Regression test suite for `google-workspace` tool |

## Dependencies

| Dependency | Type | Status | Notes |
|------------|------|--------|-------|
| ADR-010: System of Record | Architecture | Accepted | Master index provides canonical data to sync |
| Google Workspace OAuth setup | Infrastructure | Existing | Service account auth pattern in `google_workspace.go`; needs expanded scopes for write access |
| Master index | Data | In progress | Canonical registry of policies, controls, evidence tasks |
| ADR-006: Hexagonal architecture | Architecture | Accepted | Adapter interface pattern for sync integration |
| FEAT-003: Task Scheduler | Feature | Proposed | Scheduled sync execution (not blocking; on-demand works independently) |

## Risks

| Risk | Probability | Impact | Mitigation |
|------|-------------|--------|------------|
| Google API rate limits throttle sync for large policy sets | Medium | Medium | Implement exponential backoff with jitter; batch operations where possible; respect per-minute and per-day quotas; provide progress reporting during throttled syncs |
| Markdown to Google Docs format fidelity loss | High | Medium | Define a supported Markdown subset (headings, lists, tables, bold/italic, code blocks); document known limitations; automated round-trip fidelity tests |
| Large document handling (policies > 100KB) | Low | Medium | Chunk large documents for Docs API; implement progress tracking; set configurable size limits |
| Conflict resolution produces unexpected results | Medium | High | Default to `manual` conflict policy; surface conflicts with full diff context; require explicit user resolution before overwriting |
| OAuth scope expansion requires admin re-approval | Low | Low | Document required scopes upfront; provide scope migration guide for existing installations |
| Google Docs API structural content complexity | Medium | Medium | Use a well-tested Markdown-to-Docs conversion library or build incremental conversion starting with the supported subset |

## Out of Scope

- Real-time collaborative editing (Google Docs native collaboration is preserved; GRCTool syncs periodically, not in real-time).
- Google Forms write-back (Forms remain read-only for evidence extraction).
- Conversion of Google Slides or other Workspace document types.
- Multi-tenant support (single organization per GRCTool instance).

## References

- [ADR-010: System of Record Architecture](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-010-system-of-record-architecture)
- [ADR-006: Hexagonal Architecture](/home/erik/Projects/grctool/docs/helix/02-design/adr/adr-index.md#adr-006-hexagonal-architecture-ports-and-adapters)
- [Data Design: System of Record Architecture](/home/erik/Projects/grctool/docs/helix/02-design/data-design/data-design.md#system-of-record-architecture)
- [Google Workspace Setup Guide](/home/erik/Projects/grctool/docs/01-User-Guide/google-workspace-setup.md)
- [SD-002: Google Drive Sync Adapter](/home/erik/Projects/grctool/docs/helix/02-design/solution-designs/SD-002-gdrive-sync-adapter.md)
