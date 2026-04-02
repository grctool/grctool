---
title: "FEAT-010: Evidence Submission & Lifecycle Management"
phase: "01-frame"
category: "features"
tags: ["evidence", "submission", "review", "evaluation", "validation", "tugboat", "backfill"]
related: ["FEAT-009", "FEAT-007", "FEAT-006", "prd"]
status: "Implemented"
priority: "P0"
created: 2026-04-01
updated: 2026-04-01
backfill: true
---

# FEAT-010: Evidence Submission & Lifecycle Management

## Status

| Field | Value |
|-------|-------|
| Status | Implemented |
| Priority | P0 |
| Owner | Erik LaBianca |
| Implementation | `internal/services/submission/`, `internal/services/validation/`, `cmd/evidence.go` |

---

## Overview

GRCTool provides a complete evidence lifecycle — from generation through evaluation, review, and submission to Tugboat Logic via the Custom Evidence Integration API. Each stage adds quality gates: automated validation scores evidence completeness and quality, human review provides audit-ready assessment, and pre-submission checks ensure evidence meets platform requirements. Collector URLs are configured per evidence task, and submission history is tracked for audit trail purposes.

## Problem Statement

Generated evidence is only valuable if it can be validated, reviewed, and submitted to the compliance platform. Without structured lifecycle management, compliance teams generate evidence but then manually upload it to Tugboat Logic, losing traceability between the generated artifact and the submitted version. Quality varies because there's no automated quality scoring or validation before submission. The lifecycle must enforce a quality-gated pipeline: generate → evaluate → review → submit, with clear status at each stage and a complete audit trail.

---

## Requirements

### Functional Requirements

- FR-001: Evaluate evidence quality via `grctool evidence evaluate <task-id>`
- FR-002: Review evidence with human-readable assessment via `grctool evidence review <task-id>`
- FR-003: Submit evidence to Tugboat Logic via `grctool evidence submit <task-id>`
- FR-004: Configure collector URLs per evidence task via `grctool evidence setup <task-id> --collector-url <url>`
- FR-005: Collector URL storage in `.grctool.yaml` under `tugboat.collector_urls`
- FR-006: Support Tugboat task ID or GRCTool reference ID (`ET-NNNN` or numeric ID) for setup
- FR-007: Dry-run mode for setup (`--dry-run`) previewing changes without modifying config
- FR-008: Pre-submission validation checking evidence completeness and format
- FR-009: Submission via Custom Evidence Integration API with HTTP Basic Auth
- FR-010: Submission history tracking (timestamp, status, response) per task per window
- FR-011: Evidence quality scoring with completeness, quality, and source richness dimensions

### Non-Functional Requirements

- **Performance**: Evaluation and review complete within 30 seconds; submission within 10 seconds
- **Security**: Custom Integration API credentials (username/password) stored securely in config; submission over HTTPS only
- **Reliability**: Submission failures produce clear error messages with retry guidance; failed submissions do not mark evidence as submitted
- **Auditability**: Complete submission history with timestamps, statuses, and response data preserved in `submission_history.json`

---

## User Stories

### US-101: Evaluate Evidence Quality [FEAT-010]

**As a** compliance manager,
**I want to** evaluate generated evidence quality before review
**so that** I can identify evidence that needs improvement before human review.

**Acceptance Criteria:**

- [x] `grctool evidence evaluate ET-0001` scores evidence on multiple dimensions
- [x] Scoring dimensions include: completeness (requirements coverage), quality (reasoning, sources, specificity), readiness (submission-ready)
- [x] Scores are numeric with clear pass/fail thresholds
- [x] Specific recommendations for improvement when scores are below threshold
- [x] Evaluation does not modify the evidence — read-only assessment

### US-102: Review Evidence [FEAT-010]

**As a** compliance manager,
**I want to** review evidence with a human-readable assessment
**so that** I can make an informed approve/reject decision.

**Acceptance Criteria:**

- [x] `grctool evidence review ET-0001` displays evidence with review context
- [x] Review shows: evidence content, quality scores, related controls addressed, sources cited
- [x] Assessment includes specific strengths and weaknesses
- [x] Output format is human-readable markdown suitable for sharing with auditors

### US-103: Configure Collector URL [FEAT-010]

**As a** compliance engineer setting up evidence submission,
**I want to** configure the Tugboat collector URL for each evidence task
**so that** evidence is submitted to the correct Tugboat endpoint.

**Acceptance Criteria:**

- [x] `grctool evidence setup ET-0001 --collector-url "https://..."` configures the URL
- [x] Supports both GRCTool reference IDs (ET-0001) and Tugboat numeric IDs (327992)
- [x] `--dry-run` previews changes without modifying configuration
- [x] Interactive mode prompts for URL when `--collector-url` is omitted
- [x] Collector URL is persisted in `.grctool.yaml` under `tugboat.collector_urls`
- [x] Validation checks URL format before saving

### US-104: Submit Evidence to Tugboat [FEAT-010]

**As a** compliance manager,
**I want to** submit approved evidence to Tugboat Logic via the API
**so that** evidence appears in Tugboat for auditor review without manual upload.

**Acceptance Criteria:**

- [x] `grctool evidence submit ET-0001` submits evidence to the configured collector URL
- [x] Pre-submission validation checks evidence exists and collector URL is configured
- [x] Submission uses Custom Evidence Integration API with HTTP Basic Auth
- [x] Success/failure status is clearly reported
- [x] Submission metadata (timestamp, status, response) recorded in `submission_history.json`
- [x] Failed submissions produce actionable error messages

### US-105: Track Submission History [FEAT-010]

**As a** CISO,
**I want** a complete history of all evidence submissions
**so that** I can demonstrate evidence chain of custody to auditors.

**Acceptance Criteria:**

- [x] Each submission attempt is recorded with timestamp, status, and response
- [x] History is stored per-task per-window in `submission_history.json`
- [x] History is append-only — previous entries are never modified
- [x] History includes both successful and failed submission attempts

### US-106: Validate Evidence Before Submission [FEAT-010]

**As a** compliance engineer,
**I want** automated validation before submission
**so that** I don't submit incomplete or low-quality evidence.

**Acceptance Criteria:**

- [x] Pre-submission validation runs automatically before `evidence submit`
- [x] Checks: evidence file exists, format is valid, collector URL is configured, authentication is valid
- [x] Quality checks: no placeholder text, required fields populated, sources cited
- [x] Validation failures block submission with specific error messages
- [x] `--skip-validation` flag available for override when needed

---

## Edge Cases and Error Handling

- Collector URL not configured for task: Submit fails with clear message directing to `evidence setup`
- Tugboat API returns authentication error: Fail with message to check `tugboat.username` and `tugboat.password` in config
- Evidence file missing or empty: Validation catches before submission attempt
- Network error during submission: Submission recorded as failed in history; retryable
- Collector URL format invalid: Validation rejects during `evidence setup`
- Submission succeeds but Tugboat returns unexpected response: Log response, mark as submitted, warn about unexpected format

---

## Success Metrics

| Metric | Target | Status |
|--------|--------|--------|
| Submission success rate | >95% when evidence passes validation | Met |
| Validation accuracy | Zero false positives blocking valid evidence | Met |
| Quality scoring precision | Evaluation scores correlate with auditor acceptance | Met |
| Submission audit trail | 100% of submissions tracked in history | Met |
| Setup friction | < 2 minutes per task to configure collector URL | Met |

---

## Dependencies

- **FEAT-005**: Configuration (collector URLs, API credentials)
- **FEAT-006**: Authentication (Tugboat API access for submission)
- **FEAT-007**: Synced data (evidence task metadata for validation)
- **FEAT-009**: Generated evidence (content to submit)
- **Tugboat Logic Custom Evidence Integration API**: REST endpoint for evidence submission

---

## Out of Scope

- Evidence approval workflows with multiple reviewers
- Automated re-submission on schedule
- Evidence recall/retraction after submission
- Submission to non-Tugboat platforms (see FEAT-004)
- Evidence versioning beyond git history

---

## Traceability

### Related Artifacts
- **Parent PRD Section**: Core Capabilities — Evidence Submission API; Key Workflows — Review and Approval
- **Implementation**: `internal/services/submission/`, `internal/services/validation/`, `cmd/evidence.go`

### Feature Dependencies
- **Depends On**: FEAT-005 (config), FEAT-006 (auth), FEAT-007 (synced data), FEAT-009 (generated evidence)
- **Depended By**: None currently (terminal in the pipeline)

---
*Backfill spec: documents functionality that is already implemented in the codebase.*
