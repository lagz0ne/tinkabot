---
layer: task
topic: tinkalet-local-reactions
status: complete
references:
  - ../approach/tinkalet-edge.md
  - ../plan/tinkalet-edge.md
  - ./tinkalet-watch-cursors.md
---

# Tinkalet Local Reactions Task

## Objective

Implement Slice 4 `local-reactions`: Tinkalet can register a local reaction,
run it from daemon mode when an item reaches a watched status, and write the
command result back as a product item without exposing raw KV or process
internals.

## Artifact Pack

Reaction trust boundary:

```text
Tinkabot
  durable item records + timing only
      ^
      | product item writeback
      |
Tinkalet daemon
  selected profile creds + cursor + reaction registry
      |
      | explicit local command argv
      v
local process
  no Tinkabot creds, no raw NATS authority
```

Local command lifecycle:

```text
item event resolved
  |
  | match reaction condition
  v
exec explicit command argv
  |
  | capture stdout/stderr/exitCode
  v
write result item as resolved
  |
  v
advance reaction cursor
```

Reaction registry map:

```text
TINKALET_DATA_DIR/
  reactions/<name>.json
  cursors/reaction-<name>.json

reaction record:
  kind: tinkalet.reaction.v1
  watch: {scope: item, target: deploy/123, status: resolved}
  command: {cmd: /path/to/tool, args: [...]}
  write: {item: deploy/123/result}
```

Failure matrix:

| Family           | Product reason       | Cursor policy |
| ---              | ---                  | ---           |
| command failure  | command-failed       | do not advance matching event |
| denied writeback | denied-writeback     | do not advance matching event |
| duplicate event  | skipped by cursor    | no command run |
| daemon crash     | cursor replays event | retry on next run |
| retry exhaustion | deferred             | later retry policy |
| removed profile  | profile-not-found    | no command run |

Test intent map:

- Tinkalet unit: reaction add validates name, command, watch target, and profile.
- Real embedded NATS: daemon reacts only after matching item status.
- Process boundary: local command receives no credentials and returns captured
  stdout/stderr/exitCode.
- Writeback: result is readable as a product item through Tinkalet.
- Duplicate suppression: a second daemon run on the same cursor times out and
  does not run the command again.

## Command Contract

```text
tinkalet reaction add <name> --watch item <key> --for resolved --cmd <path> [--arg <arg>] --write <item-key>
tinkalet daemon react <name> --once [--timeout <duration>] [--json]
```

## Acceptance Contract

- `go test ./tinkalet -run TestReactionCommandDenials -count=1` passes.
- `go test ./tinkabot -run TestTinkaletLocalReaction -count=1` passes.
- `go test ./... -count=1` passes from `substrate/go`.
  `bun run release:evidence` stay green.

## RED Artifact

Planned RED on 2026-06-17:

- `substrate/go/tinkalet/tinkalet_test.go` `TestReactionCommandDenials`.
- `substrate/go/tinkabot/tinkalet_reaction_test.go`
  `TestTinkaletLocalReaction`.

Expected RED before implementation:

- `go test ./tinkalet -run TestReactionCommandDenials -count=1` fails because
  `reaction` is not recognized.
- `go test ./tinkabot -run TestTinkaletLocalReaction -count=1` fails because
  reaction registry and daemon execution commands do not exist.

## Verification Evidence

GREEN on 2026-06-17:

- `go test ./tinkalet -run TestReactionCommandDenials -count=1` -> pass.
- `go test ./tinkabot -run 'TestTinkaletLocalReaction|TestTinkaletReactionFailureKeepsCursor|TestTinkaletReactionDeniedWritebackKeepsCursor|TestTinkaletReactionRemovedProfileDoesNotRun' -count=1` -> pass.
- `go test ./tinkabot -run 'TestTinkaletLocalReaction|TestTinkaletReactionFailureKeepsCursor|TestTinkaletReactionDeniedWritebackKeepsCursor|TestTinkaletReactionRemovedProfileDoesNotRun|TestTinkaletItemWatchUsesWatchStream|TestTinkaletDaemonWatchCursorRestartCatchesRetainedEvents|TestTinkaletItemRecords' -count=1` -> pass.
- `go test ./cmd/tinkalet ./tinkalet -count=1` -> pass.
- `go test ./... -count=1` from `substrate/go` -> pass.
- `bun run gate:scenarios` -> `gate:scenarios passed`.
- `bun run release:evidence` -> `release evidence check passed: 17 milestones over 12 spine steps, 6 gate results`.
- `bun test tests/gate-checkers.test.ts` -> 12 pass.
- `git diff --check` -> pass.

A focused subagent review tightened the tests: the main outside-in reaction
proof uses a reactor-only profile that can watch and write its result item but
cannot direct-read items; command execution proves argv literal handling and a
scrubbed environment; registry and cursor files are mode `0600` and checked
for raw KV/credential leakage; command failure, denied writeback, and removed
profile failures do not advance the matching-event cursor or run unsafely.

## Residual Risk

This Task intentionally implements one-shot explicit command reactions. Retry
budgets, long-running process supervision, script sandboxes, agent-specific
hooks, and rich reaction logs stay deferred.
