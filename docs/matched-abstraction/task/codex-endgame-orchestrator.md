---
layer: task
topic: codex-endgame-orchestrator
references:
  - ../approach/endgame-app.md
  - ../plan/endgame-app.md
  - ../plan/codex-endgame-orchestration.md
---

# Codex Endgame Orchestrator Task

## Objective

Add a root script that manages Codex instances in a bounded loop until the Endgame Plan milestones are done, while preserving isolated writer worktrees, read-only scouts, orchestrator-owned verification, and persistent evidence.

## Scope

- Parse `tasks/todo.md` milestone status.
- Detect the next unfinished milestone and the all-DONE endgame state.
- Build noninteractive `codex exec` calls with `--ask-for-approval never`, `--json`, `--ephemeral`, and explicit sandbox mode.
- Fan out read-only Approach/Test/Risk scouts.
- Run exactly one writer in an isolated git worktree per round.
- Apply worker patches into an integration worktree.
- Run verification from the orchestrator, then read-only review and bounded fix loops.
- Persist run manifest, prompts, JSONL logs, stderr, summaries, patches, and integration branch state.

## Non-Goals

- No automatic merge into `main`.
- No direct writer edits in the primary checkout.
- No recursive Codex spawning.
- No destructive cleanup of user work.
- No claim that agent-reported success is sufficient.
- No implementation of the next product milestone.

## Acceptance Contract

- The script detects the next Endgame Plan milestone from `tasks/todo.md`.
- The script detects completion only when every milestone in the workflow is `DONE`.
- Noninteractive Codex calls use public `codex exec` JSONL mode, explicit sandbox mode, `--ephemeral`, and `--ask-for-approval never`.
- Read-only scouts cannot write by command construction.
- Exactly one worker writes per round, and that worker runs in an isolated git worktree.
- Worker output becomes a patch that the orchestrator applies to an integration worktree before verification.
- The orchestrator runs verification gates itself before review, fix, commit, or loop advancement.
- The loop has hard round, fix, timeout, and lock controls.
- Dry run reports the next round plan without launching agents.

## RED Artifact

- Command: `bun test tests/codex-orchestrator.test.ts`
- Expected failure: `Cannot find module '../scripts/codex-orchestrate'`.
- Missing contract proven: the repo had no executable Codex endgame orchestrator.

## GREEN Evidence

- Added `scripts/codex-orchestrate.ts`.
- Added `tests/codex-orchestrator.test.ts`.
- Added `orchestrate:codex`, `test:orchestrator`, and `typecheck:orchestrator` scripts.
- Added `.codex-runs/` to `.gitignore`.
- Added `tsconfig.orchestrator.json` so root orchestration code is compiler-checked.

## Verification Evidence

- `bun test tests/codex-orchestrator.test.ts` -> `4 pass`, `0 fail`, `23 expect() calls`.
- `bun run typecheck:orchestrator` -> `bunx @typescript/native-preview --noEmit -p tsconfig.orchestrator.json`.
- `bun run orchestrate:codex -- --dry-run --allow-dirty` -> parsed `command-acceptance` as next milestone, planned read-only scouts, one workspace-write worker worktree, read-only review, and full verification gates.
- `bun run test` -> `39 pass`, `0 fail`, `260 expect() calls`.
- `bun run typecheck` -> SDK typecheck plus orchestrator typecheck passed.
- `bun run build` -> SDK ESM, CommonJS, and declarations emitted.
- `bun run pack:dry` -> `Total files: 6`, unpacked size `139.91KB`.
- `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`.
- `bun run test:layers` -> `Ran 10 tests ... OK`.
- `git diff --check` -> clean.

## Wrap-Up Announcement

The Codex endgame orchestrator is ready as a development control loop. It can coordinate Codex subagents against the current Endgame Plan without letting parallel writers collide in the primary checkout or allowing agent text to replace verified milestone evidence.
