---
layer: plan
topic: codex-endgame-orchestration
references:
  - ../approach/endgame-app.md
  - ./endgame-app.md
---

# Codex Endgame Orchestration Plan

Diagram: https://diashort.apps.quickable.co/d/cb13c8db

## Consumed Approach

This Plan consumes `endgame-app` as authority. The orchestrator exists to accelerate the Endgame Plan without weakening its gates: layer discipline, RED-GREEN-TDD, traced failures, least authority, loop safety, and orchestrator-owned verification.

## Decomposition

The orchestrator is not product runtime. It is a development control loop that repeatedly reads `tasks/todo.md`, identifies the next unfinished endgame milestone, and coordinates Codex instances around that one milestone.

Each round has four roles:

| Role | Authority |
| --- | --- |
| Approach scout | Read-only check that the next milestone respects Approach invariants |
| Test scout | Read-only traced-TDD check for RED/GREEN and verification gaps |
| Risk scout | Read-only adversarial check for collisions, safety, and false completion |
| Worker | The only writer for the round, running in an isolated git worktree |

After the worker creates a patch, the orchestrator applies it to an integration worktree, runs verification itself, asks a read-only reviewer for blockers, runs bounded fix loops when needed, commits progress on the integration branch, and advances to the next milestone.

## Safety Contract

- The primary checkout is never the writer surface.
- Writer agents run in generated git worktrees.
- Read-only scouts and reviewers run with Codex `read-only` sandbox.
- The script never uses `--dangerously-bypass-approvals-and-sandbox`.
- The root worktree must be clean unless the operator explicitly passes `--allow-dirty`.
- The loop has hard `maxRounds`, `maxFix`, and timeout caps.
- Agent logs, prompts, final summaries, manifests, patches, and verification results stay under `.codex-runs/`.
- Generated git worktrees live as direct siblings of the repo root so existing relative local dependencies resolve the same way as the primary checkout.
- Subagents may report success, but only the orchestrator can advance after running verification.

## Verification Strategy

Inside-out tests own the script contract:

- milestone parsing.
- endgame completion detection.
- safe noninteractive Codex argument construction.
- read-only scout fanout with one writer worktree.

Operational smoke is a dry run:

- `bun run orchestrate:codex -- --dry-run --allow-dirty`

Release-shaped proof for the orchestrator is the normal repo verification spine after the script is added:

- `bun run test`
- `bun run typecheck`
- `bun run validate:layers`
- `bun run test:layers`

## Escalation

Escalate before enabling broader automation if the script needs multiple simultaneous writers against one checkout, recursive Codex spawning, destructive cleanup, automatic merges into `main`, or completion based on agent text instead of orchestrator-run verification.
