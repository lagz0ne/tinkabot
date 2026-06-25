---
layer: task
topic: tinkalet-schedules
status: complete
references:
  - ../approach/tinkalet-edge.md
  - ../plan/tinkalet-edge.md
  - ./tinkalet-item-records.md
---

# Tinkalet Schedules Task

## Objective

Implement Slice 5 `schedules`: Tinkalet edits durable schedule records while
Tinkabot owns wall-clock timing, restart catch-up, attribution, and item
writeback. Edge-local timers are not used to claim server schedule reliability.

## Artifact Pack

Schedule ownership:

```text
human / CI
  |
  | tinkalet schedule set/off
  v
Tinkabot schedule records
  |
  | server-owned wall clock loop
  v
Tinkabot item records
```

Schedule-to-item write sequence:

```text
tinkalet schedule set deploytick --every 200ms --write deploy/789/tick
    |
    | selected profile writes schedule intent
    v
tb_schedules/deploytick active every=200ms
    |
    | Tinkabot loop observes due record
    v
tb_items/deploy/789/tick resolved {schedule, sequence, value}
    |
    | Tinkalet item get/watch/wait sees product item truth
    v
operator script / human / local reaction continues the chain
```

Schedule record state map:

```text
tb_schedules/<name>
  kind: tinkabot.schedule.v1
  name: product schedule name
  status: active | off
  everyMs: interval floor-enforced by server
  writeItem: item key the server updates
  value: user JSON carried into each tick
  sequence: last emitted tick number
  lastTickAt: server timestamp of last tick
  updatedAt: last edit or server tick timestamp
  provenance: profile/source/writer
```

Failure matrix:

| Family                 | Product reason        | Proof owner |
| ---                    | ---                   | ---         |
| malformed duration     | malformed-duration    | Tinkalet unit |
| denied schedule edit   | profile-not-found or auth reason | Tinkalet unit/outside-in |
| missed tick            | due record emits once | outside-in |
| restart catch-up       | sequence advances after restart | outside-in |
| cancellation           | off record stops ticks | outside-in |
| edge offline behavior  | Tinkabot continues timing without Tinkalet daemon | outside-in |

Test intent map:

- Tinkalet unit: malformed duration and no selected profile use product-shaped
  denials.
- Real embedded NATS: `schedule set` writes intent and Tinkabot writes an item
  tick without requiring a Tinkalet daemon.
- Restart/catch-up: after Tinkabot restarts over the same store, a due schedule
  advances its item sequence.
- Cancellation: `schedule off` stops further item sequence changes.
- Privacy: schedule command and item output exclude raw KV subject names and
  credential contents.

## Command Contract

```text
tinkalet schedule set <name> --every <duration> --write <item-key> [--value <json>] [--json]
tinkalet schedule off <name> [--json]
```

JSON output is the schedule record. Human output is one line:
`schedule <name> active every <duration> -> <item-key>` or
`schedule <name> off`.

## Acceptance Contract

- `go test ./tinkalet -run TestScheduleCommandDenials -count=1` passes.
- `go test ./tinkabot -run TestTinkaletSchedules -count=1` passes.
- `go test ./cmd/tinkalet ./tinkalet -count=1` passes.
- `go test ./... -count=1` passes from `substrate/go`.
  `bun run release:evidence` stay green.

## RED Artifact

Added 2026-06-17:

- `substrate/go/tinkalet/tinkalet_test.go` `TestScheduleCommandDenials`.
- `substrate/go/tinkabot/tinkalet_schedule_test.go`
  `TestTinkaletSchedules`.

Expected RED before implementation:

- `go test ./tinkalet -run TestScheduleCommandDenials -count=1` fails because
  `schedule` is not recognized.
- `go test ./tinkabot -run TestTinkaletSchedules -count=1` fails because the
  schedule bucket, command surface, and Tinkabot timing loop do not exist.

## Verification Evidence

GREEN on 2026-06-17:

- `go test ./tinkalet -run TestScheduleCommandDenials -count=1` -> pass.
- `go test ./tinkabot -run TestTinkaletSchedules -count=1` -> pass.
- `go test ./cmd/tinkalet ./tinkalet -count=1` -> pass.
- `go test ./tinkabot -run 'TestTinkaletSchedules|TestTinkaletItemRecords|TestTinkaletItemWatchUsesWatchStream|TestTinkaletDaemonWatchCursorRestartCatchesRetainedEvents|TestTinkaletLocalReaction|TestTinkaletTriggerClock|TestBinaryRestartReloadsWithoutRegeneration|TestBinaryFirstStartMaterializes' -count=1` -> pass.
- `go test ./... -count=1` from `substrate/go` -> pass.
- `bun run gate:scenarios` -> `gate:scenarios passed`.
- `bun run gate:manual` -> `gate:manual passed`.
- `bun run gate:tinkalet-package` -> packaged `tinkabot` started, packaged
  `tinkalet` imported/selected the local profile, packaged NATS sidecar was
  removed, trigger advanced the clock projection, `schedule set` wrote a
  scheduled item tick, `schedule off` cancelled it, and
  `gate:tinkalet-package passed`.
- `bun run release:evidence` -> `release evidence check passed: 17 milestones over 12 spine steps, 6 gate results`.

A focused subagent review tightened the proof before implementation: the
outside-in test uses a restricted schedule profile that can edit schedules and
read the result item but cannot write `tb_items`; the produced item must carry
`tinkabot-schedule` provenance; restart catch-up must advance sequence and item
revision without resetting; `schedule off` must stay stable across another
restart; and schedule outputs are checked for raw KV and credential leakage.

## Residual Risk

This Task implements interval schedules that update one item key. Calendar
syntax, time zones, multi-item fan-out, retry policies, rich schedule logs, and
lease leadership for multi-node timing stay deferred.
