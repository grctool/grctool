---
title: "Feature: Document & Audit Lifecycle Scheduler"
feature_id: "FEAT-003"
phase: "01-frame"
category: "features"
status: "In Progress"
priority: "P0"
tags: ["lifecycle", "scheduler", "evidence-collection", "audit", "orchestration"]
related: ["adr-010", "data-design", "evidence-state", "contracts"]
created: 2026-03-17
updated: 2026-04-02
---

# FEAT-003: Document & Audit Lifecycle Scheduler

## Status

| Field | Value |
|-------|-------|
| Status | In Progress |
| Priority | P0 (foundational for system-of-record operations) |
| Owner | TBD |
| Target | TBD |

**Implementation status (2026-04-02):** Lifecycle state machines for policies,
controls, and evidence tasks are implemented in `internal/lifecycle/`. A
scheduler package exists in `internal/scheduler/` with cron expression support
and an orchestrator. CLI commands exist under `grctool lifecycle` and
`grctool schedule`. Remaining work: persistent lifecycle state storage,
schedule execution end-to-end wiring, and audit period management. Tracked in
hx-4585e374.

## Problem Statement

Evidence collection in GRCTool was originally ad-hoc, with no formal lifecycle
model for policies, controls, or evidence tasks, and no scheduler for
evidence collection cadences.

**Current state (2026-04-02):** Lifecycle state machines (policy, control,
evidence task) are implemented with validated transitions. A scheduler package
with cron expression support and an orchestrator exists. CLI surfaces for
`grctool lifecycle` and `grctool schedule` are registered. However, lifecycle
state persistence and end-to-end schedule execution are not yet complete —
the scheduler code exists but cannot durably store state across runs or
execute a full evidence collection cycle autonomously.

Organizations operating under SOC 2 Type II and ISO 27001 need:

1. **Clear lifecycle phases** for every compliance artifact -- policies must be reviewed on schedule, controls must be tested periodically, and evidence must be collected within defined audit windows.
2. **Automated scheduling** so evidence collection fires at the right time without human intervention.
3. **Visibility into what is due, overdue, and complete** so compliance managers can manage by exception rather than by checklist.

Without this, evidence gaps are discovered late in audit fieldwork, policy reviews lapse, and control testing falls behind -- all of which create audit findings.

## Lifecycle Models

### Policy Lifecycle

Policies are governance documents that define organizational commitments. They follow a review-and-approval cycle.

| Phase | Description | Transitions To |
|-------|-------------|----------------|
| **Draft** | Policy is being authored or revised | Review |
| **Review** | Policy is under stakeholder review | Approved, Draft (if rejected) |
| **Approved** | Policy has been approved by governance body | Published |
| **Published** | Policy is active and in effect | Review (on schedule), Retired |
| **Retired** | Policy is no longer in effect | -- (terminal) |

Key behaviors:
- Published policies have a `review_due_date` derived from their approval date plus the configured review interval (typically annual).
- When `review_due_date` is approaching (configurable threshold, default 30 days), the scheduler emits a review reminder.
- Version tracking: each transition from Draft to Approved increments the policy version.

### Control Lifecycle

Controls are specific security requirements that implement policies. They are tested on a defined cadence.

| Phase | Description | Transitions To |
|-------|-------------|----------------|
| **Defined** | Control requirement has been documented | Implemented |
| **Implemented** | Control has been put into operation | Tested |
| **Tested** | Control has been validated as operating effectively | Effective, Implemented (if test fails) |
| **Effective** | Control is operating and has passed testing | Tested (on schedule), Deprecated |
| **Deprecated** | Control is no longer applicable | -- (terminal) |

Key behaviors:
- Effective controls have a `next_test_date` based on the testing cadence (quarterly, semi-annual, annual).
- When a control has not been tested within its testing period, it transitions back to Implemented and is flagged as overdue.
- Control testing may be automated (via evidence collection tools) or manual.

### Evidence Task Lifecycle

Evidence tasks are the atomic units of evidence collection. This extends the existing `LocalEvidenceState` enum in `internal/models/evidence_state.go`.

| Phase | Description | Transitions To |
|-------|-------------|----------------|
| **Scheduled** | Task is on the collection schedule but not yet due | Collecting (when schedule fires) |
| **Collecting** | Evidence collection is in progress (tools running) | Collected, Scheduled (on failure with retry) |
| **Collected** | Evidence files have been gathered | Validated |
| **Validated** | Evidence has passed quality checks | Submitted |
| **Submitted** | Evidence has been sent to Tugboat Logic | Accepted, Rejected |
| **Accepted** | Evidence was accepted by the auditor | Scheduled (next period) |
| **Rejected** | Evidence was rejected and needs rework | Collecting |

Key behaviors:
- The Scheduled and Collecting phases are new; the existing states (`no_evidence`, `generated`, `validated`, `submitted`, `accepted`, `rejected`) map to Collected through Rejected.
- Scheduled tasks carry a `next_collection_date` computed from their cron expression and audit window.
- The transition from Scheduled to Collecting is triggered by the scheduler; all other transitions are triggered by explicit commands.

### Audit Lifecycle

An audit period is the top-level container that groups evidence collection into a coherent review cycle.

| Phase | Description | Transitions To |
|-------|-------------|----------------|
| **Planning** | Audit scope, timeline, and ownership are being defined | Fieldwork |
| **Fieldwork** | Auditors are actively requesting and reviewing evidence | Evidence Collection |
| **Evidence Collection** | Organization is gathering and submitting evidence | Review |
| **Review** | Auditor is reviewing submitted evidence | Report, Evidence Collection (if gaps found) |
| **Report** | Audit report is being drafted | Remediation, Close |
| **Remediation** | Organization is addressing findings | Close |
| **Close** | Audit is complete | -- (terminal) |

Key behaviors:
- An audit period defines the time window (e.g., Q1-2026, Annual-2026) and the set of evidence tasks in scope.
- The scheduler uses the audit period to determine which tasks need evidence and by when.
- Multiple audit periods can be active simultaneously (e.g., SOC 2 annual + ISO 27001 surveillance).

## User Stories

### US-001: Define Evidence Collection Schedules

**As a** compliance manager,
**I want to** define evidence collection schedules (quarterly, annually, continuous) per task or task group,
**so that** evidence is collected automatically at the right time without manual tracking.

**Acceptance Criteria:**
- Schedules are defined in `.grctool.yaml` using cron expressions or named intervals (quarterly, monthly, annual).
- Each evidence task can be assigned to one or more schedules.
- Task groups (by control family, framework, or custom tag) can share a schedule.
- `grctool schedule list` shows all schedules with their next run time.

### US-002: Evidence Collection Deadline Dashboard

**As a** compliance manager,
**I want to** see a dashboard of upcoming evidence collection deadlines,
**so that** nothing is missed and I can manage by exception.

**Acceptance Criteria:**
- `grctool schedule status` shows tasks grouped by: overdue, due this week, due this month, upcoming.
- Output includes task ref, task name, assigned schedule, last collected date, and next due date.
- JSON output format is available for integration with external dashboards.

### US-003: Automatic Evidence Collection on Schedule

**As a** security engineer,
**I want** evidence tasks to trigger automatically when their schedule fires, collecting from all configured sources,
**so that** evidence is always current without manual intervention.

**Acceptance Criteria:**
- `grctool schedule run [schedule-name]` executes all tasks assigned to that schedule.
- Each task runs its mapped tools in the configured order with parallel execution where safe.
- Partial failures do not block other tasks; each task reports its own status.
- The command is designed to be called from external cron, systemd timers, or CI pipelines.

### US-004: Audit Lifecycle Timeline

**As an** auditor,
**I want to** see the audit lifecycle timeline showing what evidence has been collected vs. what is still pending,
**so that** I can track audit progress and identify gaps early.

**Acceptance Criteria:**
- `grctool lifecycle status` shows all entities (policies, controls, evidence tasks) with their current lifecycle phase.
- Filtering by audit period, framework, and state is supported.
- Summary statistics show percentage complete per control family and overall.

### US-005: Automatic Policy Review Reminders

**As a** CISO,
**I want** policy review reminders triggered automatically when policies approach their review date,
**so that** governance obligations are met without manual calendar tracking.

**Acceptance Criteria:**
- Policies have a `review_interval` (default: 12 months) and a `last_approved_date`.
- `grctool schedule status` includes policies due for review within the configured reminder window.
- The reminder threshold is configurable (default: 30 days before due date).

### US-006: Scheduled Control Testing

**As a** compliance manager,
**I want** control testing to run on a defined schedule and flag controls that have not been validated within their testing period,
**so that** control effectiveness is continuously monitored.

**Acceptance Criteria:**
- Controls have a `testing_cadence` (quarterly, semi-annual, annual).
- `grctool schedule status` flags controls whose last test date exceeds their cadence.
- `grctool schedule run` can trigger control-level testing by running all evidence tasks mapped to that control.

### US-007: YAML-Configurable and CLI-Overridable Scheduler

**As a** DevOps engineer,
**I want** the scheduler to be configurable via YAML and overridable from the CLI,
**so that** I can integrate it into our existing automation infrastructure.

**Acceptance Criteria:**
- All schedule configuration lives in `.grctool.yaml` under a `schedules` key.
- CLI flags (`--schedule`, `--dry-run`, `--parallel`) override YAML defaults.
- `grctool schedule run --dry-run` previews what would execute without running anything.
- Exit codes follow GRCTool conventions (0 = success, 1 = partial failure, 2 = config error).

## Acceptance Criteria (Feature-Level)

1. All four lifecycle models (policy, control, evidence task, audit) are implemented as state machines with validated transitions.
2. Lifecycle state is persisted in `.state/` and queryable via CLI.
3. At least three schedule types are supported: periodic (cron), event-driven (on sync/change), and manual (CLI trigger).
4. `grctool schedule run` can execute a full evidence collection cycle for an audit period.
5. Schedule configuration in `.grctool.yaml` is documented and validated at startup.
6. The scheduler produces structured output (JSON) suitable for CI/CD integration.
7. Existing `LocalEvidenceState` in `evidence_state.go` is extended, not replaced, maintaining backward compatibility.

## Dependencies

| Dependency | Description | Status |
|------------|-------------|--------|
| ADR-010 | Master index / system-of-record architecture | Accepted |
| `evidence_state.go` | Existing evidence state machine and `StateCache` | Implemented |
| Tool interface (API-002) | Tool registry for task-to-tool mapping | Implemented |
| Storage interface (API-004) | File-based storage for lifecycle state | Implemented (lifecycle persistence incomplete) |
| Configuration interface (API-005) | `.grctool.yaml` schema extension | Implemented |
| FEAT-004 provider registry | Provider routing for scheduled sync operations | In Progress |

## Risks and Mitigations

| Risk | Impact | Likelihood | Mitigation |
|------|--------|------------|------------|
| Cron expression complexity | Users misconfigure schedules, tasks run at wrong times | Medium | Provide named presets (quarterly, monthly, annual) alongside raw cron; validate expressions at config load |
| Timezone handling | Evidence collected in wrong audit window due to TZ mismatch | Medium | Store all times in UTC internally; allow `timezone` config per schedule; display in local time |
| Missed schedules (CLI is not a daemon) | Evidence not collected if no external trigger is configured | High | Document integration patterns for cron/systemd/CI; `grctool schedule status` flags overdue tasks; catch-up mode runs missed schedules |
| State file corruption | Lifecycle state lost or inconsistent | Low | Atomic writes with temp-file-rename; state is reconstructable from filesystem evidence |
| Backward compatibility | Extending `LocalEvidenceState` breaks existing state files | Low | New states are additive; existing state values remain valid; migration logic in state loader |

## References

- [Data Design: Evidence Lifecycle States](/home/erik/Projects/grctool/docs/helix/02-design/data-design/data-design.md)
- [Interface Contracts: Tool Interface](/home/erik/Projects/grctool/docs/helix/02-design/contracts/contracts.md)
- [Existing Evidence State Machine](/home/erik/Projects/grctool/internal/models/evidence_state.go)
- [PRD: Agentic Workflow Overview](/home/erik/Projects/grctool/docs/helix/01-frame/prd/requirements.md)
- [User Stories](/home/erik/Projects/grctool/docs/helix/01-frame/user-stories/stories.md)
