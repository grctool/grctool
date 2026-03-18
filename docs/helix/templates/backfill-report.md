# HELIX Backfill Report: [Scope]

**Backfill Date**: [YYYY-MM-DD]
**Scope**: [repo | subsystem | feature | story-set]
**Status**: [draft | awaiting-guidance | complete | superseded]
**Research Epic**: [bd issue ID]
**Research Beads**: [bd issue IDs]
**Primary Evidence Baseline**: [paths and major sources]

## Backfill Metadata

- **Reviewer / Author**: [name]
- **Run Trigger**: [why backfill was initiated]
- **Authority Baseline**: Product Vision -> PRD -> Feature Specs / User Stories -> Architecture / ADRs -> Solution / Technical Designs -> Test Plans / Tests -> Implementation Plans -> Source Code
- **Confidence Scale**: HIGH / MEDIUM / LOW
- **Upstream Beads References**: [epic ID and key child IDs]

## Scope and Evidence Baseline

### Scope Definition

- [Area 1]
- [Area 2]
- [Area 3]

### Evidence Surveyed

- [docs/...]
- [tests/...]
- [src/...]
- [ci/deploy/runbook/...]
- [git history / release notes / changelog]

## Recursive Coverage

### Coverage Ledger

| Scope Node | Node Type | Review Bead | Coverage Status | Files / Paths Covered | Notes |
|------------|-----------|-------------|-----------------|-----------------------|-------|
| [repo/area/folder/file-set] | [root/area/folder/file-set] | [bd ID] | [complete/in-progress/deferred/excluded] | [paths] | [reason or summary] |

### Explicit Exclusions

| Path | Reason Excluded | Impact on Confidence |
|------|------------------|----------------------|
| [path] | [reason] | [HIGH/MEDIUM/LOW impact] |

### Consolidation Chain

| Child Node | Parent Consolidation Node | Status |
|------------|---------------------------|--------|
| [folder or file-set] | [area or root] | [complete/in-progress] |

## Current-State Summary

### Observed Product Behavior

[Summary]

### Observed Architecture and Runtime Shape

[Summary]

### Operational and Delivery Context

[Summary]

### Evidence vs Inference Notes

- [direct evidence]
- [medium-confidence inference]
- [low-confidence area]

## Artifact Inventory and Gaps

| Artifact Slot | Current State | Action | Confidence | Evidence |
|---------------|---------------|--------|------------|----------|
| [PRD / Architecture / Test Plan / etc.] | [exists/missing/stale/partial] | [preserve/create/update/defer] | [HIGH/MEDIUM/LOW] | [path refs] |

## Confidence Ledger

| Area | Statement | Confidence | Evidence | Notes |
|------|-----------|------------|----------|-------|
| [Area] | [Inferred or confirmed statement] | [HIGH/MEDIUM/LOW] | [path refs] | [why] |

## Guidance Gates

### Questions Raised

| Decision Area | Ambiguity | Evidence | Default Interpretation | User Guidance Needed |
|---------------|-----------|----------|------------------------|----------------------|
| [Area] | [What is unclear] | [path refs] | [Default] | [Exact question] |

### Guidance Received

| Decision Area | User Direction | Date | Applied To |
|---------------|----------------|------|------------|
| [Area] | [Answer] | [date] | [artifact paths] |

## Backfilled Artifacts

| Artifact | Status | Confidence | Basis | Notes |
|----------|--------|------------|-------|-------|
| [docs/helix/...path] | [created/updated/deferred] | [HIGH/MEDIUM/LOW] | [evidence + user guidance] | [notes] |

## Assumption Ledger

| Assumption | Confidence | Affected Artifacts | Confirmation Status | Next Action |
|------------|------------|--------------------|---------------------|-------------|
| [Statement] | [HIGH/MEDIUM/LOW] | [paths] | [confirmed/pending/rejected] | [action] |

## Follow-Up Beads

| Bead ID | Type | HELIX Labels | Goal | Dependencies | Why It Exists |
|---------|------|--------------|------|--------------|---------------|
| [bd ID] | [task/chore/decision] | [phase:... kind:...] | [goal] | [deps] | [rationale] |

## Next Recommended Steps

1. [Highest-priority next step]
2. [Next guidance or artifact update]
3. [Next implementation or test alignment step]
