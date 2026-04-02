---
title: "FEAT-009: AI-Powered Evidence Generation"
phase: "01-frame"
category: "features"
tags: ["evidence", "ai", "claude", "generation", "prompt-as-data", "backfill"]
related: ["FEAT-008", "FEAT-010", "FEAT-007", "prd"]
status: "Implemented"
priority: "P0"
created: 2026-04-01
updated: 2026-04-01
backfill: true
---

# FEAT-009: AI-Powered Evidence Generation

## Status

| Field | Value |
|-------|-------|
| Status | Implemented |
| Priority | P0 |
| Implementation | `internal/evidence/`, `internal/services/evidence.go`, `internal/services/evidence/`, `cmd/evidence.go` |

---

## Overview

GRCTool uses Claude AI to generate compliance evidence from tool outputs, synced compliance data, and organizational context. The generation pipeline follows a multi-phase approach: context assembly from synced data, data collection from infrastructure tools, prompt assembly using a "prompt-as-data" pattern, AI-powered evidence synthesis, and quality validation. Generated evidence includes reasoning and source attribution for auditability.

## Problem Statement

Even with automated tool data, converting raw infrastructure outputs into auditor-ready evidence narratives requires domain expertise and significant writing effort. Compliance engineers spend hours per evidence task translating Terraform configs, GitHub permission matrices, and workflow analyses into structured evidence that addresses specific control requirements. AI can automate this synthesis — but only if given rich, structured context about the task requirements, related controls, applicable policies, and available tool outputs. The evidence generation pipeline must be transparent (reasoning visible), traceable (sources cited), and reviewable (human approval before submission).

---

## Requirements

### Functional Requirements

- FR-001: Generate evidence for a single task via `grctool evidence generate <task-id>`
- FR-002: Bulk generation via `grctool evidence generate --all`
- FR-003: Multi-phase pipeline: context → data collection → prompt assembly → AI generation → output
- FR-004: Context includes: task description, guidance, related controls, related policies, template variables
- FR-005: Optional tool data collection (`--with-tool-data`) executing relevant tools for the task
- FR-006: Prompt-as-data pattern — tools produce structured JSON; prompt assembler creates AI input
- FR-007: Claude AI model configurable (model, max_tokens, temperature) via `.grctool.yaml`
- FR-008: Multiple output formats: CSV (tabular evidence), Markdown (narrative evidence)
- FR-009: Evidence includes reasoning and source attribution sections
- FR-010: Window-based evidence management (e.g., "2026-Q1") for rolling collection
- FR-011: Evidence stored locally in `evidence/<task-id>/<window>/` directory structure
- FR-012: Collection plan generation for multi-source evidence tasks

### Non-Functional Requirements

- **Performance**: Single evidence generation completes within 60 seconds (AI response time dependent)
- **Quality**: ≥80% of generated evidence accepted after single human review
- **Security**: Claude API key stored via environment variable (`${CLAUDE_API_KEY}`); never logged or included in output
- **Reliability**: Tool failures during data collection do not block evidence generation; partial data produces partial evidence with warnings
- **Auditability**: All generated evidence includes reasoning chain and source references

---

## User Stories

### US-091: Generate Evidence for a Task [FEAT-009]

**As a** compliance manager,
**I want to** generate evidence for a specific evidence task using AI
**so that** I get audit-ready evidence without manually writing narratives.

**Acceptance Criteria:**

- [x] `grctool evidence generate ET-0001` generates evidence for the specified task
- [x] Generated evidence addresses the task's requirements and guidance
- [x] Evidence references related controls and policies by name and ID
- [x] Output includes a reasoning section explaining how evidence supports the control
- [x] Evidence is saved to `evidence/ET-0001/<window>/` in the configured format

### US-092: Bulk Evidence Generation [FEAT-009]

**As a** compliance manager preparing for an audit,
**I want to** generate evidence for all pending tasks at once
**so that** I can prepare for the audit in a single operation.

**Acceptance Criteria:**

- [x] `grctool evidence generate --all` processes all evidence tasks
- [x] Progress reporting shows which tasks are being processed
- [x] Individual task failures do not block remaining tasks
- [x] Summary report shows success/failure counts and any errors

### US-093: Evidence with Tool Data [FEAT-009]

**As a** security engineer,
**I want** evidence generation to incorporate live infrastructure data
**so that** evidence reflects current infrastructure state, not just policy descriptions.

**Acceptance Criteria:**

- [x] `--with-tool-data` flag triggers relevant tool execution before AI generation
- [x] Tool outputs are included in the prompt context
- [x] Evidence cites specific tool findings (e.g., "Terraform security analyzer found...")
- [x] Multiple tool outputs are aggregated into a coherent evidence package

### US-094: Window-Based Evidence Management [FEAT-009]

**As a** compliance manager collecting quarterly evidence,
**I want** evidence organized by collection window (e.g., "2026-Q1")
**so that** I maintain distinct evidence sets for each audit period.

**Acceptance Criteria:**

- [x] Evidence is organized in `evidence/<task-id>/<window>/` directories
- [x] Multiple windows can coexist for the same task
- [x] Each window tracks: collection date, format, validation status
- [x] Previous windows are preserved when generating new evidence

### US-095: Configure AI Model Parameters [FEAT-009]

**As a** an advanced user,
**I want to** configure the AI model, token limits, and temperature
**so that** I can tune evidence generation for quality vs. cost.

**Acceptance Criteria:**

- [x] Model configurable via `evidence.claude.model` (default: claude-3-5-sonnet-20241022)
- [x] Max tokens configurable via `evidence.claude.max_tokens` (default: 8192)
- [x] Temperature configurable via `evidence.claude.temperature` (default: 0.1)
- [x] API key via `evidence.claude.api_key` supporting `${CLAUDE_API_KEY}` substitution
- [x] Max tool calls configurable via `evidence.generation.max_tool_calls` (default: 50)

---

## Edge Cases and Error Handling

- Claude API unavailable: Fail with clear error; do not produce partial evidence
- API key missing or invalid: Fail before prompt assembly with configuration guidance
- Evidence task has no related controls or policies: Generate evidence from task description and guidance alone; warn about missing context
- Tool data collection partially fails: Include available tool outputs; note missing sources in evidence
- Generated evidence exceeds format limits: Truncation with "continued in next section" markers
- Duplicate generation for same window: Overwrite with warning; previous version not preserved (git tracks history)

---

## Success Metrics

| Metric | Target | Status |
|--------|--------|--------|
| Generation success rate | ≥80% without manual intervention | Met |
| Evidence acceptance rate | ≥90% after single review | Met |
| Generation time (single task) | < 60 seconds | Met |
| Source attribution | 100% of evidence includes sources | Met |
| Reasoning transparency | 100% of evidence includes reasoning | Met |

---

## Dependencies

- **FEAT-005**: Configuration (Claude API key, model parameters, output settings)
- **FEAT-007**: Synced data (evidence tasks, controls, policies for context)
- **FEAT-008**: Tool framework (infrastructure data collection)
- **Claude AI API**: Anthropic API for evidence synthesis

---

## Out of Scope

- Multi-model support (only Claude AI; no OpenAI/Gemini integration)
- Evidence version control beyond git (no built-in versioning system)
- Real-time evidence generation triggered by infrastructure changes
- Collaborative evidence editing or co-authoring
- Evidence generation without AI (fully manual evidence is out of scope for this feature)

---

## Traceability

### Related Artifacts
- **Parent PRD Section**: Core Capabilities — AI-Powered Evidence Generation; Agentic Workflow
- **Implementation**: `internal/evidence/`, `internal/services/evidence.go`, `internal/services/evidence/`, `cmd/evidence.go`

### Feature Dependencies
- **Depends On**: FEAT-005 (config), FEAT-007 (synced data), FEAT-008 (tools)
- **Depended By**: FEAT-010 (submission of generated evidence)

---
*Backfill spec: documents functionality that is already implemented in the codebase.*
