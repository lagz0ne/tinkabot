---
layer: task
topic: tinkalet-watch-cursors
status: complete
references:
  - ../approach/tinkalet-edge.md
  - ../plan/tinkalet-edge.md
  - ./tinkalet-item-records.md
---

# Tinkalet Watch Cursors Task

## Objective

Implement Slice 3 `watch-cursors`: Tinkalet can watch item records by key or
prefix, persist cursor position under its data dir, resume without duplicate
events, and keep watching after Tinkabot restart without exposing raw KV names.

## Artifact Pack

Watch attach/reconnect:

```text
tinkalet watch item deploy/123 --cursor deploy123
    |
    | selected profile caller creds
    v
JetStream KV watch over Tinkabot item records
    |
    | replay history, skip <= cursor revision
    | then stay attached for live updates
    v
newline item events
```

Cursor data-dir map:

```text
TINKALET_DATA_DIR/
  profiles/<profile>/caller.creds
  cursors/<name>.json

cursor record:
  kind: tinkalet.cursor.item.v1
  profile/source: selected profile identity
  scope: item | prefix
  target: product key or prefix
  revision: last emitted KV revision
```

Duplicate and stale-event handling:

```text
KV event rev 10 ---- cursor rev 12 ----> skip duplicate/stale
KV event rev 13 ---- cursor rev 12 ----> emit, persist cursor rev 13
process restart ---- cursor rev 13 ----> replay history, skip <= 13
```

Failure matrix:

| Family           | Product reason       | Proof owner |
| ---              | ---                  | ---         |
| lost connection  | connection-lost      | watch path  |
| stale cursor     | stale-cursor         | cursor unit |
| permission loss  | revoked-credentials  | profile auth path |
| malformed event  | malformed-event      | outside-in  |
| duplicate event  | skipped by cursor    | outside-in  |
| daemon restart   | cursor resumes       | outside-in  |
| timeout          | watch-timeout        | outside-in  |

Test intent map:

- Tinkalet unit: malformed cursor names and missing profile are product-shaped.
- Real embedded NATS: watch attaches before an item update and emits the update.
- Cursor: second watch run skips the already emitted revision and waits.
- Restart: same cursor file resumes after Tinkabot restart and emits only the
  later revision.
- Privacy: watch output excludes raw `tb_items`, `$KV`, and credential data.

## Command Contract

```text
tinkalet watch item <key> [--cursor <name>] [--limit <n>] [--timeout <duration>] [--json]
tinkalet watch prefix <prefix> [--cursor <name>] [--limit <n>] [--timeout <duration>] [--json]
tinkalet daemon watch item <key> --cursor <name> [--limit <n>] [--timeout <duration>] [--json]
tinkalet daemon watch prefix <prefix> --cursor <name> [--limit <n>] [--timeout <duration>] [--json]
```

JSON output is newline-delimited item events. Human output is one line per
event: `item <key> <status> rev <n>`.

## Acceptance Contract

- `go test ./tinkalet -run TestWatchCommandDenials -count=1` passes.
- `go test ./tinkabot -run TestTinkaletWatchCursors -count=1` passes.
- `go test ./... -count=1` passes from `substrate/go`.
  `bun run release:evidence` stay green.

## RED Artifact

Added 2026-06-17:

- `substrate/go/tinkalet/tinkalet_test.go` `TestWatchCommandDenials`.
- `substrate/go/tinkabot/tinkalet_watch_test.go` `TestTinkaletWatchCursors`.

Expected RED before implementation:

- `go test ./tinkalet -run TestWatchCommandDenials -count=1` fails because
  `tinkalet watch` is not recognized.
- `go test ./tinkabot -run TestTinkaletWatchCursors -count=1` fails because
  watch commands and cursor files do not exist.

## Verification Evidence

GREEN on 2026-06-17:

- `go test ./tinkalet -run TestWatchCommandDenials -count=1` -> pass.
- `go test ./tinkabot -run 'TestTinkaletItemWatchUsesWatchStream|TestTinkaletDaemonWatchCursorRestartCatchesRetainedEvents' -count=1` -> pass.
- `go test ./tinkabot -run 'TestTinkaletItemWatchUsesWatchStream|TestTinkaletDaemonWatchCursorRestartCatchesRetainedEvents|TestTinkaletItemRecords|TestTinkaletTriggerClock|TestTinkaletAuthorityDenials' -count=1` -> pass.
- `go test ./cmd/tinkalet ./tinkalet -count=1` -> pass.
- `go test ./... -count=1` from `substrate/go` -> pass.
- `bun run gate:scenarios` -> `gate:scenarios passed`.
- `bun run release:evidence` -> `release evidence check passed: 17 milestones over 12 spine steps, 6 gate results`.
- `bun test tests/gate-checkers.test.ts` -> 12 pass.
- `git diff --check` -> pass.

The test plan was strengthened after a focused subagent review: watch tests now
use a watch-only profile that lacks direct item get/read authority, prove two
ordered same-key live revisions, configure retained item history, and prove
daemon restart catch-up emits both retained offline revisions without replaying
the cursor seed. A future cursor is denied as `stale-cursor`.

## Residual Risk

This Task emits watch events and persists cursor position only. Local command
execution, retries, and writeback reactions belong to Slice 4 `local-reactions`.
