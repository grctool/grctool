---
dun:
  id: helix.workflow.beads
  depends_on:
    - helix.workflow
---
# HELIX Beads Integration

HELIX uses upstream Beads by Steve Yegge as its execution tracker.

- Upstream repo (`@steveyegge/beads` / `steveyegge/beads`): <https://github.com/steveyegge/beads>
- Upstream docs: <https://steveyegge.github.io/beads/>
- Local usage help: `bd quickstart`, `bd human`, `bd --help`

Do not define a HELIX-specific bead file format. Use upstream `bd` issues,
dependencies, labels, parents, and ready queues. HELIX adds conventions on top
of `bd`; it does not replace `bd`.

## Scope

This document owns HELIX's mapping onto upstream Beads.

- Follow this file for labels, issue types, `spec-id`, and queue-query
  conventions.
- Follow [EXECUTION.md](EXECUTION.md) for loop control and action sequencing.
- If another HELIX document shows different `bd` semantics, prefer this file
  and correct the drift there.

## Setup

Initialize Beads once at the project root:

```bash
bd init
bd quickstart
bd human
```

This creates the upstream `.beads/` workspace managed by `bd` and Dolt. HELIX
canonical planning artifacts still live under `docs/helix/`.

HELIX execution assumes a working repo-local Beads workspace already exists.
If live `bd` access is missing or unhealthy, stop immediately. Do not run
`bd init` or inspect alternate tracker sources from execution flows.

## Working Database and Remotes

The repository-local Beads database under `.beads/dolt` is the authoritative
working database for a repo.

- A Dolt remote is optional.
- If a project uses a shared Dolt remote for coordination, it must be a real
  shared remote, not a machine-local `file://` path.
- Do not use machine-local or CIFS/SMB-backed `file://` remotes as a hot
  coordination path.
- A `file://` remote, if a project explicitly allows one at all, must be
  documented as manual backup only, not routine agent sync.
- If no proper shared remote exists, prefer local-only operation over a broken
  `file://` remote.

Push/pull failures, non-fast-forward races, or missing table-file / manifest
errors do not automatically mean the local Beads DB is corrupted. A healthy
local `.beads/dolt` paired with a bad remote topology is a different failure
mode and should be treated differently.

## HELIX Mapping

Use upstream Beads fields directly:

- `type`: use native issue types such as `task`, `epic`, `chore`, or `decision`
- `parent`: group related work under a review epic or other parent bead
- `deps` / `bd dep add`: encode blocking dependencies
- `status`: use native operational states such as `open`, `in_progress`,
  `deferred`, and `closed`
- `spec-id`: point to the nearest governing canonical artifact
- `description`, `design`, `acceptance`, and `notes`: capture the work contract
- `labels`: encode HELIX-specific execution semantics

Blocked work should be modeled with dependencies and surfaced through
`bd blocked` / `bd ready`, not with a custom HELIX status taxonomy.

## Verification Evidence Conventions

If a repository defines canonical verification wrappers or proof lanes, treat
them as the closure surface for the corresponding bead.

- Close execution beads from the repo-owned wrapper command and its retained
  artifacts, not from narrower package or file commands that were only used to
  debug the failure.
- When the canonical lane contradicts historical close evidence, do not leave
  the bead closed just because upstream Beads has no native `failed` status.
  Reopen the bead immediately or create an explicit regression bead linked to
  the contradicted close evidence.
- Record the exact command, run date, exit status, and artifact paths reviewed
  in the bead close comment or regression note.

## HELIX Label Conventions

Every HELIX bead should have:

- `helix`
- one phase label: `phase:build`, `phase:deploy`, `phase:iterate`, or `phase:review`

Add labels as needed for traceability:

- `kind:build`, `kind:deploy`, `kind:backlog`, `kind:review`
- `story:US-XXX`
- `feature:FEAT-XXX`
- `source:metrics`, `source:feedback`, `source:retrospective`, `source:incident`
- area labels such as `area:auth` or `area:cli`

Use `--spec-id` for the closest governing artifact. Put the full authority
chain in the description when more than one canonical artifact governs the
work.

## HELIX Categories in Upstream Beads

| HELIX category | Upstream type | Required labels | Typical governing refs |
|---|---|---|---|
| Story build work | `task` | `helix`, `phase:build`, `kind:build`, `story:US-XXX` | `TP-XXX`, `TD-XXX`, `implementation-plan.md` |
| Story deploy work | `task` | `helix`, `phase:deploy`, `kind:deploy`, `story:US-XXX` | deployment docs, related build bead IDs |
| Improvement backlog item | `task` or `chore` | `helix`, `phase:iterate`, `kind:backlog` | lessons learned, feedback, incidents, metrics |
| Review epic | `epic` | `helix`, `phase:review`, `kind:review` | review scope and governing artifacts |
| Review slice | `task` | `helix`, `phase:review`, `kind:review` | parent review epic plus scoped artifacts |

## Command Patterns

### Build Bead

```bash
bd create "Implement US-036: evidence collection API" \
  --type task \
  --labels helix,phase:build,kind:build,story:US-036,feature:FEAT-001 \
  --spec-id TP-036 \
  --description "Governing artifacts: US-036, TD-036, TP-036, docs/helix/04-build/implementation-plan.md" \
  --design "Implement the CLI slice needed to satisfy the failing TP-036 tests." \
  --acceptance "TP-036 tests pass; no upstream artifact drift is introduced."
```

### Deploy Bead

```bash
bd create "Roll out US-036: evidence collection API" \
  --type task \
  --labels helix,phase:deploy,kind:deploy,story:US-036 \
  --spec-id docs/helix/05-deploy/deployment-checklist.md \
  --description "Governing artifacts: deployment-checklist.md, monitoring-setup.md, runbook.md, build bead grctool-a3f2dd" \
  --acceptance "Smoke checks pass, monitoring is clean, rollback trigger is documented." \
  --deps grctool-a3f2dd
```

### Backlog Bead

```bash
bd create "Reduce evidence sync retry noise in logs" \
  --type chore \
  --labels helix,phase:iterate,kind:backlog,source:metrics,area:sync \
  --spec-id docs/helix/06-iterate/lessons-learned.md \
  --description "Derived from lessons learned and production metrics." \
  --acceptance "Retry logging volume drops without hiding actionable failures."
```

### Review Epic and Review Beads

```bash
review_epic=$(bd create "HELIX alignment review: evidence collection and sync" \
  --type epic \
  --labels helix,phase:review,kind:review \
  --spec-id docs/helix/01-frame/prd.md \
  --silent)

bd create "Review evidence collection traceability and implementation drift" \
  --type task \
  --parent "$review_epic" \
  --labels helix,phase:review,kind:review,area:evidence \
  --description "Review scope: evidence requirements, design, tests, and implementation alignment."
```

## Daily Workflow

For operator-facing execution flow, bounded loop control, and Codex or Claude
automation patterns, see [EXECUTION.md](EXECUTION.md).

Use the direct `bd` commands below when triaging or managing the queue
manually.

Find ready work:

```bash
bd ready --label helix --label phase:build
bd ready --label helix --label phase:iterate --label kind:backlog
```

Claim, inspect, and close work:

```bash
bd update <id> --claim
bd show <id>
bd dep tree <id>
bd close <id>
```

Check blocked work or epic progress:

```bash
bd blocked
bd epic status
```

## Troubleshooting Local vs Remote Health

When Beads looks broken, separate local DB health from remote topology health.

Check the local repository state first:

```bash
bd doctor
(cd .beads/dolt && dolt status && dolt fsck)
```

If those checks pass, the repo-local `.beads/dolt` database is likely healthy.

Then inspect remote configuration separately:

```bash
(cd .beads/dolt && dolt remote -v)
```

Treat the remote as misconfigured or unsafe when it points at a machine-local
`file://` path or a CIFS/SMB-backed filesystem that multiple agents are using
as a hot coordination backend.

If no proper shared remote exists, leave Dolt remote sync unset and keep using
the local repository database.

## Repairing a Stale or Bad Remote

If a project needs a shared remote, repoint `origin` to a proper shared remote:

```bash
bd dolt remote remove origin
bd dolt remote add origin <shared-dolt-remote>
bd dolt pull
```

Do not replace one bad machine-local or CIFS/SMB-backed `file://` remote with
another. If no proper shared remote exists yet, operate locally until one does.

## What Not To Do

- Do not create `.beads/build/*.yaml`, `.beads/deploy/*.yaml`, or other HELIX-specific bead files.
- Do not invent `BEAD-###` identifiers; use native `bd` issue IDs.
- Do not invent a parallel status model when upstream status plus dependencies already models the work.
- Do not tell agents that every session must push or pull an arbitrary local `file://` Dolt remote.
