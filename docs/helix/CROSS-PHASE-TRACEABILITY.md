---
title: "Cross-Phase Traceability"
phase: "cross-phase"
category: "governance"
tags: ["traceability", "helix", "specs", "alignment"]
created: 2026-04-02
updated: 2026-04-02
---

# Cross-Phase Traceability

This document maps GRCTool's specifications across the HELIX phases so that any
requirement can be traced from intent to implementation, and any code change can
be traced back to its governing decision.

## HELIX Phase Flow

Specifications flow forward through the HELIX phases. Each phase refines the
prior one and is governed by it:

`Vision -> PRD -> Features -> ADRs / Solution Designs -> Tests -> Build -> Iterate`

- **Vision** — long-term product direction.
- **PRD** (`01-frame/prd/`) — product requirements derived from the vision.
- **Features** (`01-frame/features/`) — FEAT-NNN specs with user stories and
  acceptance criteria.
- **Architecture / ADRs** (`02-design/adr/`) — durable architectural decisions.
- **Solution Designs** (`02-design/solution-designs/`) — SD-NNN designs for
  specific features.
- **Tests / Build** — implementation and its test coverage under `internal/`,
  `cmd/`, and the test suites.
- **Iterate** (`06-iterate/`) — alignment reviews that record drift, gaps, and
  corrections.

## Authority Order

When two artifacts disagree, the higher-authority artifact wins. Resolve the
lower one (or escalate to amend the higher). Order, highest first:

1. Vision
2. PRD
3. Feature Specs / User Stories
4. Architecture / ADRs
5. Solution Designs
6. Tests
7. Implementation Plans
8. Code

## Traceability Matrix

Values below are extracted from the feature frontmatter, `adr-index.md`, and the
solution-design frontmatter. "Governing ADR(s)" lists the architectural
decisions each feature implements or depends on; entries marked *(implied)* are
not declared in the feature's `related` field but follow directly from its
documented decision domain.

| Feature | Status | Governing ADR(s) | Solution Design | Notes |
|---------|--------|------------------|-----------------|-------|
| FEAT-001 AccountableHQ Bidirectional Policy Sync | Proposed (P1) | ADR-010 | SD-001 | Depends on FEAT-004 provider framework; provider stub only, no API integration yet. |
| FEAT-002 Google Drive Bidirectional Sync | Proposed (P1) | ADR-010, ADR-006 | SD-002 | Depends on FEAT-004; GDrive provider stub only; existing read-only `google_workspace.go` unaffected. |
| FEAT-003 Document & Audit Lifecycle Scheduler | In Progress (P0) | ADR-010 | SD-003 | Lifecycle state machines and scheduler implemented; persistent state and end-to-end wiring outstanding. |
| FEAT-004 Universal Document Provider Framework | In Progress (P0) | ADR-010, ADR-006, ADR-011 | SD-004 | Foundation for FEAT-001/002; 22 of 33 acceptance criteria met (see AR-2026-04-01-repo). |
| FEAT-005 CLI Framework & Configuration | Implemented (P0) | ADR-002, ADR-001 *(implied)* | — | Backfilled. Implemented in `cmd/`, `internal/config/`. |
| FEAT-006 Browser-Based Authentication | Implemented (P0) | ADR-003 *(implied)* | — | Backfilled. Implemented in `internal/auth/`, `cmd/auth.go`. |
| FEAT-007 Tugboat Data Synchronization | Implemented (P0) | ADR-004, ADR-006 *(implied)* | — | Backfilled. Implemented in `internal/services/sync.go`, `internal/tugboat/`, `internal/storage/`. |
| FEAT-008 Evidence Collection Tool Framework | Implemented (P0) | ADR-006 *(implied)* | — | Backfilled. Tool registry in `internal/tools/`, `cmd/tool*.go`. |
| FEAT-009 AI-Powered Evidence Generation | Implemented (P0) | ADR-009 *(implied)* | — | Backfilled. Implemented in `internal/evidence/`, `internal/services/evidence/`. |
| FEAT-010 Evidence Submission & Lifecycle | Implemented (P0) | ADR-009, ADR-004 *(implied)* | — | Backfilled. Implemented in `internal/services/submission/`, `internal/tugboat/`. |

Solution designs exist only for the four in-flight system-of-record features
(FEAT-001 through FEAT-004). The backfilled features (FEAT-005 through FEAT-010)
document already-shipped behavior and trace directly to ADRs without an
intermediate solution design.

## Live Gap & Alignment Analysis

Point-in-time traceability and gap analysis — drift between specs and code,
acceptance-criteria status, and corrective actions — is recorded in:

`docs/helix/06-iterate/alignment-reviews/`

Current reviews: `AR-2026-03-18-post-implementation.md`,
`AR-2026-03-18-repo.md`, `AR-2026-04-01-repo.md`. The matrix above is a static
index; the alignment reviews are the authoritative record of current
spec-to-implementation alignment.
