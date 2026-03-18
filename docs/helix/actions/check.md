# HELIX Action: Check

You are performing a bounded HELIX execution-state check.

Your goal is to inspect the current HELIX planning and execution state, decide
whether there is more actionable work for the given scope, and recommend the
next HELIX action without inventing work or drifting from the authority stack.

This action is read-only by default. Do not modify product code. Do not claim
execution beads. Only recommend what should happen next.

## Action Input

You may receive:

- no argument
- `repo`
- a scope selector such as `US-042`, `FEAT-003`, `area:auth`, or `phase:build`

Examples:

- `helix check`
- `helix check repo`
- `helix check FEAT-003`
- `helix check area:auth`

If no scope is given, default to the repository.

## Decision Codes

Your first output line must be exactly one of:

- `NEXT_ACTION: IMPLEMENT`
- `NEXT_ACTION: ALIGN`
- `NEXT_ACTION: BACKFILL`
- `NEXT_ACTION: WAIT`
- `NEXT_ACTION: GUIDANCE`
- `NEXT_ACTION: STOP`

Use them precisely:

- `IMPLEMENT`: one or more ready HELIX execution beads exist and should be worked now
- `ALIGN`: no safe ready execution bead exists, but reconciliation should create or refine the next work set
- `BACKFILL`: the canonical HELIX stack is too incomplete to continue safely
- `WAIT`: there is open work, but it is claimed by another agent or blocked by a truly external dependency that code changes cannot resolve
- `GUIDANCE`: user or stakeholder input is required before safe work can continue
- `STOP`: there is no actionable work remaining for the scope right now

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
- Do not treat open implementation work as proof that the plan is complete.
- Prefer a real `bd ready` view over status-only heuristics such as `bd list --ready`.

## PHASE 0 - Bootstrap

1. Verify upstream Beads is available.
   - If live `bd` access is missing or unhealthy, stop immediately.
   - Do not run `bd init` or infer queue state from alternate tracker sources.
2. Determine the scope.
3. Detect whether canonical HELIX docs exist for the scope.
   - check `docs/helix/`
   - check for alignment or backfill reports relevant to the scope when useful

## PHASE 1 - Queue Health

Inspect the current execution queue using blocker-aware Beads commands.

At minimum, inspect:

- `bd status --json` for global queue health
- `bd ready --json` filtered to HELIX execution beads
- `bd list --status in_progress --label helix --json` for active claimed work
- `bd blocked --json` and open HELIX beads for blocked work when relevant

Preferred ready-work filter:

- label `helix`
- any of `phase:build`, `phase:deploy`, or `phase:iterate`

Do not use review beads as evidence that implementation should continue.

## PHASE 2 - Artifact Health

Assess whether the planning stack is sufficient for continued execution.

Check for:

- missing or obviously incomplete `docs/helix/` coverage
- stale or contradictory upstream artifacts
- recent implementation changes without corresponding planning or test support
- open execution beads whose governing artifacts are too weak to execute safely
- queue starvation caused by missing review, decision, or doc work

## PHASE 3 - Decision Logic

Apply these rules in order:

1. Recommend `IMPLEMENT` when:
   - one or more safe ready HELIX execution beads exist
2. Recommend `BACKFILL` when:
   - the canonical HELIX stack is missing or too incomplete to continue safely
3. Recommend `ALIGN` when:
   - the planning stack exists, but no safe ready execution bead exists and a reconciliation pass is likely to expose or refine the next work
4. Recommend `IMPLEMENT` (not `WAIT`) when:
   - work is blocked, but the blocking beads themselves are actionable
     (e.g., config changes, migrations, infrastructure-as-code fixes)
   - in this case, recommend implementing the blocker bead directly
5. Recommend `WAIT` when:
   - work exists, but is claimed by another agent or blocked on a truly
     external dependency that cannot be resolved by code changes (e.g.,
     waiting for a third-party service, hardware provisioning, or human
     approval)
6. Recommend `GUIDANCE` when:
   - a user or stakeholder decision is the real blocker
7. Recommend `STOP` when:
   - there are no ready execution beads
   - no missing planning work is indicated
   - no blocked or in-progress scope requires action

Do not recommend `ALIGN` just because the queue is empty. Distinguish true work
exhaustion from planning gaps.

## PHASE 4 - Suggested Command

Provide the exact next command for the recommended action where possible:

- `IMPLEMENT`:
  - `helix implement`
- `ALIGN`:
  - `helix align <scope>`
- `BACKFILL`:
  - `helix backfill <scope>`

For `WAIT`, `GUIDANCE`, or `STOP`, provide the exact reason and the condition
that would change the result.

## Output Format

Output these sections in order:

1. `NEXT_ACTION: ...`
2. Scope
3. Queue Health
4. Artifact Health
5. Remaining Work Assessment
6. Recommended Command
7. Stop Or Escalation Condition

Be concise, explicit, and operational.
