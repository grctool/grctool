---
dun:
  id: helix.workflow.reference
  depends_on:
    - helix.workflow
---
# HELIX Quick Reference Card

## Canonical Docs

- [README.md](README.md): high-level model and authority order
- [EXECUTION.md](EXECUTION.md): queue control and operator loop
- [BEADS.md](BEADS.md): upstream `bd` mapping and labels
- [implementation.md](actions/implementation.md): one bounded execution pass
- [check.md](actions/check.md): queue-drain decision
- [reconcile-alignment.md](actions/reconcile-alignment.md): top-down review
- [backfill-helix-docs.md](actions/backfill-helix-docs.md): conservative reconstruction

## Phase Summary

| Phase | Primary Output | Main Location |
|---|---|---|
| Optional `00-discover` | vision and opportunity framing | `docs/helix/00-discover/` |
| `01-frame` | requirements and stories | `docs/helix/01-frame/` |
| `02-design` | architecture and design contracts | `docs/helix/02-design/` |
| `03-test` | test plans and failing tests | `docs/helix/03-test/`, `tests/` |
| `04-build` | project build guidance + execution beads | `docs/helix/04-build/`, `.beads/` |
| `05-deploy` | rollout docs + deploy beads | `docs/helix/05-deploy/`, `.beads/` |
| `06-iterate` | backlog, reports, follow-up planning | `docs/helix/06-iterate/` |

## Authority Order

1. Product Vision
2. Product Requirements
3. Feature Specs / User Stories
4. Architecture / ADRs
5. Solution / Technical Designs
6. Test Plans / Tests
7. Implementation Plans
8. Source Code / Build Artifacts

## Core Commands

### Bootstrap

```bash
bd init
```

### Commands

```bash
helix run
helix implement
helix implement bd-abc123
helix check repo
helix align repo
helix backfill repo
```

### Beads

```bash
bd ready --label helix --json
bd update <id> --claim
bd show <id>
bd dep tree <id>
bd blocked --json
bd close <id>
bd doctor
(cd .beads/dolt && dolt status && dolt fsck && dolt remote -v)
```

`.beads/dolt` is the authoritative working database. A Dolt remote is
optional; if one exists, it should be a proper shared remote, not a
machine-local or CIFS/SMB-backed `file://` coordination path.

## Beads Labeling

- Always add `helix`.
- Add exactly one phase label:
  `phase:build`, `phase:deploy`, `phase:iterate`, or `phase:review`.
- Add kind labels when useful:
  `kind:build`, `kind:deploy`, `kind:backlog`, `kind:review`.
- Add traceability labels when useful:
  `story:US-XXX`, `feature:FEAT-XXX`, `area:<name>`, `source:metrics`.

## Decision Guide

- Ready execution beads exist:
  run `implementation` or `helix run`.
- No ready execution bead, but the planning stack exists and next work is
  unclear:
  run alignment.
- Canonical docs are missing or too incomplete to execute safely:
  run backfill.
- Work exists but is blocked or already in progress:
  stop and wait.
- The queue drains:
  run `check`, not a blind loop and not `bd list --ready`.

## Artifact Inputs

Use the prompts and templates from the DDX library or local copies under:

- `docs/helix/templates/`

Those templates support the canonical docs under `docs/helix/`; they
do not replace the bounded execution contract.

## Output Reports

- Alignment reviews:
  `docs/helix/06-iterate/alignment-reviews/AR-YYYY-MM-DD[-scope].md`
- Backfill reports:
  `docs/helix/06-iterate/backfill-reports/BF-YYYY-MM-DD[-scope].md`

## Validation

When changing HELIX wrapper behavior or the execution contract:

```bash
go test ./...
git diff --check
```
