# HELIX Action: Implementation

You are performing one bounded HELIX execution pass against upstream Beads
(`bd`).

Your goal is to choose one ready execution bead, implement it completely
without drifting from the authoritative planning stack, satisfy all applicable
project quality gates, create any necessary follow-on beads, commit the work
with explicit bead traceability, close the bead, and exit.

This action is intentionally single-run. It must never loop internally or claim
multiple beads in one invocation. External supervisors may invoke it
repeatedly, but each run handles at most one bead. When the ready queue drains,
the external supervisor should run `docs/helix/actions/check.md` instead
of continuing blindly.

## Action Input

You may receive:

- no argument
- an explicit bead ID such as `bd-abc123`
- a scope selector such as `US-042`, `FEAT-003`, `area:auth`, or `phase:deploy`

Examples:

- `helix implement`
- `helix implement bd-abc123`
- `helix implement US-042`
- `helix implement area:auth`

If no argument is given, choose the best ready HELIX execution bead.

## Authority Order

When artifacts disagree, use this order:

1. Product Vision
2. Product Requirements
3. Feature Specs / User Stories
4. Architecture / ADRs
5. Solution Designs / Technical Designs
6. Test Plans / Tests
7. Implementation Plans
8. Source Code / Build Artifacts

Rules:

- Higher layers govern lower layers.
- Tests govern build execution but do not override requirements or design.
- Source code reflects current state but does not redefine the plan.
- If a bead conflicts with its governing artifacts, do not implement the drift.
- Prefer aligning code and docs to plan. Only propose plan changes when the
  evidence is strong and the governing artifacts are stale or incomplete.

## Beads Rules

Use native upstream Beads only. Follow:

- `docs/helix/BEADS.md`
- <https://github.com/steveyegge/beads>
- <https://steveyegge.github.io/beads/>

Do not create custom HELIX bead files.

This action works only on execution beads. Exclude review beads by default.

Eligible beads typically have:

- `helix`
- one of `phase:build`, `phase:deploy`, or `phase:iterate`
- no unresolved blockers in `bd`

Do not claim or implement `phase:review` beads with this action.

## Core Principle

Select the smallest ready bead that unlocks meaningful forward progress and has
enough governing context to execute safely.

Do not pick work just because it is ready. Beads with weak authority, unclear
scope, or missing verification must be refined before implementation.

## PHASE 0 - Bootstrap

1. Verify upstream Beads is available.
   - If live `bd` access is missing or unhealthy, stop immediately.
   - Do not run `bd init` or inspect alternate tracker sources from this action.
2. Inspect the current git worktree.
   - Do not revert unrelated changes.
   - If unrelated changes create commit risk, isolate your bead changes rather
     than cleaning the tree destructively.
3. Load project quality and completeness gates.
   - Read relevant HELIX guidance such as `docs/helix/04-build/enforcer.md`
     and any repo-specific CI, lint, test, security, or release rules.

## PHASE 1 - Candidate Discovery

Determine the candidate set:

1. If the input is an explicit bead ID:
   - inspect only that bead
2. If the input is a scope selector:
   - search ready HELIX execution beads matching the selector
3. If no input is given:
   - inspect ready HELIX execution beads, excluding `phase:review`

Use upstream commands such as:

- `bd ready`
- `bd show <id>`
- `bd dep tree <id>`

## PHASE 2 - Candidate Ranking

Rank candidates deterministically.

Prefer, in order:

1. explicit user-selected bead
2. unblocked bead with the clearest governing artifacts
3. bead on or near the critical path because other beads depend on it
4. bead whose acceptance criteria are specific and locally verifiable
5. smallest coherent slice likely to finish cleanly in one run

De-prioritize or reject beads when:

- governing artifacts are missing or contradictory
- acceptance criteria are vague
- required verification is undefined
- the bead is a hidden planning or decision task in execution clothing
- the bead would require broad speculative refactoring to complete

If no candidate is safe to execute, do not claim one. Report the reason and
open a refinement or decision bead if appropriate. Exit cleanly so the
supervisor can run the queue-health check.

## PHASE 3 - Claim And Context Load

For the selected bead:

1. claim it with `bd update <id> --claim`
2. inspect:
   - bead fields and labels
   - `spec-id`
   - parent epic or parent bead
   - dependency tree
   - acceptance text
   - related story, feature, or area labels
3. load the governing artifacts referenced by:
   - `spec-id`
   - bead description
   - parent bead or epic
   - linked user story, feature, design, or test artifacts
4. determine the work phase from labels:
   - `phase:build`
   - `phase:deploy`
   - `phase:iterate`

## PHASE 4 - Pre-Execution Validation

Before editing code or docs, validate:

- the bead is still ready and unblocked
- the governing artifacts are sufficient to execute
- the acceptance criteria match upstream intent
- the verification method is concrete
- there is no upstream contradiction that should be resolved first

If the bead is underspecified or divergent:

- stop implementation
- document the gap
- create the needed follow-on bead such as `decision`, `doc`, or `design`
- leave the current bead open unless it is genuinely invalid

## PHASE 5 - Phase-Appropriate Execution

### `phase:build`

Follow Build-phase discipline strictly:

- implement only what is needed to satisfy the governing tests and artifacts
- do not change test expectations just to make the bead pass
- do not add unspecified features
- keep changes scoped to the bead
- refactor only after verification is green

### `phase:deploy`

Follow Deploy-phase discipline strictly:

- execute rollout, release, monitoring, and runbook work only within the bead scope
- do not expand product behavior or sneak in implementation changes unrelated to deployment safety
- verify rollback and observability expectations where required

### `phase:iterate`

Follow Iterate-phase discipline strictly:

- limit changes to documented cleanup, lessons, backlog, or metrics work
- do not turn iterate work into hidden feature implementation
- capture concrete follow-on execution beads when new work is discovered

## PHASE 6 - Follow-On Bead Capture

If execution reveals additional work:

- create a new upstream bead immediately
- make it atomic and deterministic
- set `spec-id` to the nearest governing artifact
- add the correct HELIX labels
- encode blockers with `bd dep add`

Create follow-on beads when:

- remaining work is outside the current bead scope
- a new bug or cleanup item is discovered
- governing docs must change before more code should land
- deployment or iterate work is exposed by build completion

Do not silently absorb follow-on work into the current bead.

## PHASE 7 - Verification

Run all verification required by the bead and the project.

At minimum, verify:

- the bead acceptance criteria are satisfied
- relevant tests pass
- no previously passing required checks now fail
- lint, type, format, or static analysis gates pass if defined by the project
- docs/config/runbooks are updated where required
- any build, deploy, or iterate phase exit conditions touched by the work are still valid

If the repository defines canonical verification wrappers or proof lanes, use
those wrappers for closure evidence. Narrower package or file commands are for
debugging after the canonical lane fails; they do not replace the maintained
closure surface.

If verification fails:

- fix the issue within the bead scope, or
- leave the bead open with a precise status note and follow-on beads if needed

Do not commit broken work as a completed bead.

If a canonical verification run contradicts a previously closed bead, do not
leave that bead green. Reopen it immediately or create a linked regression bead
that records the exact contradictory command, date, exit status, and reviewed
artifacts.

## PHASE 8 - Commit And Close

If the bead is complete:

1. review the diff for scope discipline
2. create a comprehensive commit that references the bead ID
3. include in the commit message:
   - bead ID
   - concise summary
   - governing artifact references where helpful
   - verification summary
4. close the bead with `bd close <id>`

If the worktree contains unrelated changes, commit only the files that belong to
the bead. Never revert unrelated work just to simplify the commit.

## PHASE 9 - Output

Report:

1. Selected Bead
2. Why It Was Chosen
3. Governing Artifacts
4. Work Completed
5. Follow-On Beads Created
6. Verification Performed
7. Commit Created
8. Final Bead Status
9. Open Risks Or Decisions

Be precise and deterministic.
