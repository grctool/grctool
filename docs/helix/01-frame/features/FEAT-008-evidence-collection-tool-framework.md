---
title: "FEAT-008: Evidence Collection Tool Framework"
phase: "01-frame"
category: "features"
tags: ["tools", "terraform", "github", "google-workspace", "evidence-collection", "backfill"]
related: ["FEAT-005", "FEAT-007", "FEAT-009", "prd"]
status: "Implemented"
priority: "P0"
created: 2026-04-01
updated: 2026-04-01
backfill: true
---

# FEAT-008: Evidence Collection Tool Framework

## Status

| Field | Value |
|-------|-------|
| Status | Implemented |
| Priority | P0 |
| Owner | Erik LaBianca |
| Implementation | `internal/tools/`, `cmd/tool*.go` |

---

## Overview

GRCTool provides a registry of 29 specialized evidence collection tools that extract compliance-relevant data from infrastructure systems. Each tool encodes domain knowledge about a specific data source (Terraform, GitHub, Google Workspace) or compliance analysis task (relationship mapping, summary generation, prompt assembly). Tools are invocable individually via the CLI or orchestrated by the AI evidence generation pipeline.

## Problem Statement

Compliance evidence collection requires extracting data from diverse infrastructure systems — Terraform configurations, GitHub repositories, Google Workspace documents — and interpreting that data through a compliance lens. Each system has its own API, authentication, data format, and relevant security attributes. Without specialized tools, compliance teams either write one-off scripts (unmaintainable, inconsistent) or manually collect screenshots and exports (slow, error-prone, non-repeatable). A standardized tool framework enables consistent, automated, auditable evidence collection across all infrastructure systems.

---

## Requirements

### Functional Requirements

- FR-001: Tool registry with dynamic registration and lookup by name
- FR-002: Standardized tool interface: `Execute(ctx, request) → (output, source, error)`
- FR-003: CLI invocation via `grctool tool <tool-name> [flags]`
- FR-004: Tool discovery via `grctool tool --help` listing all registered tools
- FR-005: Per-tool help text with usage examples and supported flags
- FR-006: Tool isolation — failed tools do not halt other operations
- FR-007: Structured output from tools suitable for AI consumption
- FR-008: Evidence source metadata attached to tool outputs (tool name, execution time, parameters)

#### Terraform Tools (7)
- FR-T01: `terraform-security-analyzer` — comprehensive security configuration analysis with SOC2 control mapping
- FR-T02: `terraform-security-indexer` — fast resource indexing and querying by security attributes
- FR-T03: `terraform-hcl-parser` — deep HCL structural analysis
- FR-T04: `terraform-evidence-query` — Claude-powered infrastructure evidence queries
- FR-T05: `terraform_analyzer` — configuration analysis for security, modules, data sources
- FR-T06: `terraform-snippets` — pattern-based configuration snippet suggestions
- FR-T07: `atmos-stack-analyzer` — multi-environment Atmos stack analysis

#### GitHub Tools (6)
- FR-G01: `github-permissions` — repository access controls, collaborators, teams
- FR-G02: `github-security-features` — Dependabot, SAST, secret scanning configuration
- FR-G03: `github-workflow-analyzer` — CI/CD workflow security, deployment controls, approval processes
- FR-G04: `github-review-analyzer` — pull request review compliance, approval patterns
- FR-G05: `github-deployment-access` — environment protection rules and deployment controls
- FR-G06: `github-enhanced` — advanced repository search with date filtering and caching

#### Google Workspace Tool (1)
- FR-W01: `google-workspace` — evidence extraction from Drive, Docs, Sheets, Forms

#### Evidence Analysis Tools (6)
- FR-A01: `evidence-task-details` — retrieve task requirements, guidance, and metadata
- FR-A02: `evidence-task-list` — list and filter evidence tasks programmatically
- FR-A03: `evidence-relationships` — map relationships between tasks, controls, and policies
- FR-A04: `control-summary-generator` — template-based control summaries (prompt-as-data pattern)
- FR-A05: `policy-summary-generator` — template-based policy summaries
- FR-A06: `prompt-assembler` — comprehensive prompt generation with context and examples

#### Evidence Management Tools (5)
- FR-M01: `evidence-generator` — AI-coordinated evidence generation with sub-tool orchestration
- FR-M02: `evidence-validator` — completeness scoring and quality assessment
- FR-M03: `evidence-writer` — evidence file writing with window management
- FR-M04: `storage-read` — path-safe file reading with format auto-detection
- FR-M05: `storage-write` — path-safe file writing with directory management

#### Utility Tools (3)
- FR-U01: `name-generator` — filesystem-friendly naming for entities
- FR-U02: `grctool-run` — execute allowlisted grctool commands with structured output
- FR-U03: `tugboat-sync-wrapper` — wrapped sync with structured output and selective resources
- FR-U04: `docs-reader` — documentation search with keyword relevance scoring

### Non-Functional Requirements

- **Performance**: Tool queries against indexed data return in < 100ms; full analysis tools complete within 30 seconds
- **Security**: GitHub tokens and Google credentials are never included in tool output; tool execution respects API rate limits
- **Reliability**: Individual tool failures are contained — one tool's error does not prevent other tools from executing
- **Extensibility**: New tools can be added by implementing the tool interface and registering in the registry

---

## User Stories

### US-081: Discover Available Tools [FEAT-008]

**As a** compliance engineer,
**I want to** see all available evidence collection tools and their purposes
**so that** I know what automated evidence collection is available.

**Acceptance Criteria:**

- [x] `grctool tool --help` lists all 29 registered tools with descriptions
- [x] Each tool has `--help` flag with usage, supported flags, and examples
- [x] Tools are grouped by category in help output

### US-082: Collect Terraform Security Evidence [FEAT-008]

**As a** security engineer,
**I want to** analyze Terraform configurations for security evidence
**so that** I can prove infrastructure compliance controls are implemented.

**Acceptance Criteria:**

- [x] `terraform-security-analyzer` maps Terraform resources to SOC2 controls
- [x] `terraform-security-indexer` supports fast queries by security domain (encryption, access, network, logging)
- [x] `terraform-hcl-parser` provides deep structural analysis of HCL configurations
- [x] Configurable scan paths and include/exclude patterns
- [x] Output includes resource names, security attributes, and control mappings

### US-083: Collect GitHub Security Evidence [FEAT-008]

**As a** compliance manager,
**I want to** extract repository access controls, workflow security, and review compliance from GitHub
**so that** I can demonstrate code change management and access control evidence.

**Acceptance Criteria:**

- [x] `github-permissions` extracts collaborators, teams, and branch protection rules
- [x] `github-security-features` reports on Dependabot, SAST, and secret scanning configuration
- [x] `github-workflow-analyzer` analyzes CI/CD workflows for security controls and approval processes
- [x] `github-review-analyzer` reports on pull request review patterns and compliance
- [x] `github-deployment-access` extracts environment protection rules
- [x] Rate limiting respects GitHub API limits (30 req/min for search)

### US-084: Collect Google Workspace Evidence [FEAT-008]

**As a** compliance manager,
**I want to** extract evidence from Google Drive, Docs, Sheets, and Forms
**so that** I can demonstrate document access controls and policy distribution.

**Acceptance Criteria:**

- [x] `google-workspace` extracts document content and metadata from Drive
- [x] Supports Google Docs, Sheets, and Forms content extraction
- [x] Service account authentication via `GOOGLE_APPLICATION_CREDENTIALS`
- [x] Shared Drive and folder-level access extraction

### US-085: Map Evidence Relationships [FEAT-008]

**As a** compliance manager,
**I want to** see how evidence tasks, controls, and policies relate to each other
**so that** I understand evidence coverage and control gaps.

**Acceptance Criteria:**

- [x] `evidence-relationships` maps tasks → controls → policies with configurable depth
- [x] `evidence-task-list` supports filtering by status, framework, priority, assignee, category, complexity
- [x] `evidence-task-details` provides complete task requirements, guidance, and metadata
- [x] Relationship data includes reference IDs for cross-navigation

### US-086: Generate Evidence Collection Prompts [FEAT-008]

**As a** the AI evidence generation pipeline,
**I want** assembled prompts that combine task requirements, control context, policy references, and tool outputs
**so that** I can generate comprehensive, contextualized evidence.

**Acceptance Criteria:**

- [x] `prompt-assembler` generates prompts combining task description, guidance, related controls, and related policies
- [x] `control-summary-generator` and `policy-summary-generator` produce focused summaries using template-based approach (no AI API calls)
- [x] Prompts include template variable substitutions for organization-specific content
- [x] Output is structured JSON suitable for AI model consumption

---

## Edge Cases and Error Handling

- Missing Terraform files at configured paths: Tool returns empty result with warning, not error
- GitHub API rate limit exceeded: Exponential backoff; tool retries or fails gracefully with partial results
- Google service account lacks permissions: Clear error message identifying the missing scope
- Tool produces empty output: Logged as warning; does not block evidence generation pipeline
- Concurrent tool execution: Tools are stateless and safe for parallel execution

---

## Success Metrics

| Metric | Target | Status |
|--------|--------|--------|
| Tool count | 29 registered tools | Met |
| Infrastructure coverage | Terraform, GitHub, Google Workspace | Met |
| Tool isolation | Zero cross-tool failure propagation | Met |
| Query performance | < 100ms for indexed queries | Met |
| API rate compliance | Zero rate limit violations in normal operation | Met |

---

## Dependencies

- **FEAT-005**: Configuration (tool-specific settings, scan paths, API tokens)
- **FEAT-006**: Authentication (GitHub and Google Workspace credentials)
- **FEAT-007**: Synced data (evidence tasks, controls, policies for analysis tools)
- **Terraform**: HCL configuration files in configured scan paths
- **GitHub API**: REST API for repository data (optional)
- **Google Workspace API**: Drive, Docs, Sheets, Forms APIs (optional)

---

## Out of Scope

- AWS Config or CloudTrail evidence collection
- Azure compliance evidence collection
- Custom tool development SDK or plugin API
- Tool execution scheduling or automation (see FEAT-003)
- Real-time infrastructure monitoring

---

## Traceability

### Related Artifacts
- **Parent PRD Section**: Core Capabilities — Technology Integration; Tool Discovery
- **Implementation**: `internal/tools/`, `cmd/tool*.go`, `cmd/tool_run.go`

### Feature Dependencies
- **Depends On**: FEAT-005 (configuration), FEAT-006 (authentication), FEAT-007 (synced data)
- **Depended By**: FEAT-009 (evidence generation orchestrates tools)

---
*Backfill spec: documents functionality that is already implemented in the codebase.*
