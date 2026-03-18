---
dun:
  id: helix.workflow.execution
  depends_on:
    - helix.workflow
    - helix.workflow.beads
---
# HELIX Execution Guide

This guide covers operator-facing HELIX execution flow: how to run bounded work
passes, how to decide whether more work remains, and how to automate the queue
for Codex or Claude Code.

For upstream Beads integration, labels, `spec-id`, and raw `bd` conventions,
see [BEADS.md](BEADS.md).

## Scope

This document owns HELIX execution behavior.

- Follow this file for queue guards, loop shape, and `NEXT_ACTION` handling.
- Follow the bounded action prompts under `docs/helix/actions/` for
  action-specific behavior.
- Treat examples elsewhere in `docs/helix/` as supportive summaries, not
  alternate execution contracts.

## Core Actions

HELIX uses four top-level execution actions:

- `helix implement`
  Executes one ready execution bead end-to-end, then exits.
- `helix check`
  Determines whether the next step is implementation, alignment, backfill,
  waiting, guidance, or stopping.
- `helix align <scope>`
  Runs a top-down reconciliation review and can emit follow-up execution beads.
- `helix backfill <scope>`
  Reconstructs missing HELIX docs conservatively from current evidence.

## Execution Model

Use a two-stage control loop:

1. Guard on true ready work with `bd ready`, not `bd list --ready`
2. Run the bounded `implementation` action while ready work exists
3. When the queue drains, run the bounded `check` action once
4. Follow `check` to either implement again, align, backfill, wait, ask for
   guidance, or stop

`bd ready` is blocker-aware. `bd list --ready` is not equivalent and should not
control an autonomous execution loop.

## Queue Guard

These examples assume `jq` is available.

```bash
helix_ready_count() {
  bd ready \
    --label helix \
    --label-any phase:build \
    --label-any phase:deploy \
    --label-any phase:iterate \
    --json | jq 'length'
}
```

## Manual Loop

This is the minimal safe operator loop:

```bash
while [ "$(helix_ready_count)" -gt 0 ]; do
  helix implement
done

helix check
```

Interpret `check` as follows:

- `NEXT_ACTION: IMPLEMENT`
  More safe ready work exists; continue.
- `NEXT_ACTION: ALIGN`
  Run `reconcile-alignment` for the indicated scope.
- `NEXT_ACTION: BACKFILL`
  Run `backfill-helix-docs` for the indicated scope.
- `NEXT_ACTION: WAIT`
  Stop and wait for blockers or other in-progress work to resolve.
- `NEXT_ACTION: GUIDANCE`
  Stop and get user or stakeholder input.
- `NEXT_ACTION: STOP`
  No actionable work remains for the current scope.

## Agent Loops

The examples below assume a trusted local repository.

- Codex is intentionally run with `--dangerously-bypass-approvals-and-sandbox`
  and `--progress-cursor`.
- If the agent runtime cannot reach localhost Dolt sockets, force Beads direct
  mode with `BEADS_DOLT_SERVER_MODE=0`.

### Codex

```bash
while [ "$(helix_ready_count)" -gt 0 ]; do
  codex --dangerously-bypass-approvals-and-sandbox exec --progress-cursor -C "$PWD" --ephemeral <<'EOF'
Use the HELIX implementation action at docs/helix/actions/implementation.md.
Execute one ready HELIX execution bead end-to-end.
Follow the action exactly.
EOF
done

codex --dangerously-bypass-approvals-and-sandbox exec --progress-cursor -C "$PWD" --ephemeral <<'EOF'
Use the HELIX check action at docs/helix/actions/check.md.
Return the required NEXT_ACTION line and the exact next command.
Follow the action exactly.
EOF
```

### Claude Code

```bash
while [ "$(helix_ready_count)" -gt 0 ]; do
  claude -p \
    --permission-mode bypassPermissions \
    --dangerously-skip-permissions \
    --no-session-persistence <<'EOF'
Use the HELIX implementation action at docs/helix/actions/implementation.md.
Execute one ready HELIX execution bead end-to-end.
Follow the action exactly.
EOF
done

claude -p \
  --permission-mode bypassPermissions \
  --dangerously-skip-permissions \
  --no-session-persistence <<'EOF'
Use the HELIX check action at docs/helix/actions/check.md.
Return the required NEXT_ACTION line and the exact next command.
Follow the action exactly.
EOF
```

## `helix run`

If this repo provides a small wrapper CLI, expose it on your `PATH` locally.

Main commands:

- `helix run`
- `helix implement`
- `helix check`
- `helix align`
- `helix backfill`

`helix run`:

- loops only while true ready HELIX execution work exists
- runs one bounded implementation pass at a time
- runs `check` when the queue drains
- can trigger `reconcile-alignment` every `N` cycles or when `check` returns
  `ALIGN`
- stops on `WAIT`, `GUIDANCE`, or `STOP`
- uses `codex --dangerously-bypass-approvals-and-sandbox exec
  --progress-cursor` when `--agent codex` is selected
- keeps wrapper and child Beads calls in direct mode when
  `BEADS_DOLT_SERVER_MODE=0` is set

Examples:

```bash
helix run
helix run --agent claude
helix run --review-every 5
helix check repo
helix align auth
```

## Reproducible Testing

The wrapper should be tested with deterministic command stubs rather than live
Codex or Claude sessions.

Run:

```bash
go test ./...
```

## Practical Rules

- Keep execution bounded to one bead per implementation pass.
- Do not use an unconditional `while true` loop.
- Treat `check` as the queue-drain decision point, not `reconcile-alignment`.
- Use alignment to expose or refine the next work set, not as the default work
  picker.
- Do not auto-run backfill unless you are intentionally reconstructing missing
  canonical docs.
