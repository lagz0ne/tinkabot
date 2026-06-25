---
layer: task
topic: activation-schedule-engine
status: complete
references:
  - ../approach/endgame-app.md
  - ../plan/activation-foundation.md
  - ./activation-contract-authority.md
  - ./activation-ledger-durability.md
  - ./activation-source-authority.md
  - ./activation-router-live-sources.md
---

# Activation Schedule Engine

Diagram: https://diashort.apps.quickable.co/d/e1cb7a6c

## Objective

Add durable schedule activation without introducing best-effort timers. A schedule tick becomes an activation only after deterministic clock input, schedule lease/leader/fence checks, source authority, and durable ledger acceptance all agree.

## Scope

In scope:

- deterministic fake-clock tick input.
- durable schedule state with restart recovery.
- leader epoch and fencing checks.
- clock-position cursoring.
- catch-up after restart.
- duplicate tick suppression.
- source-authority and ledger integration.
- embedded NATS KV proof for schedule state.

Out of scope:

- wall-clock scheduler loops.
- distributed leader election implementation.
- script execution.
- materialization.
- browser UI.
- sandboxing.

## Acceptance Contract

- A due schedule tick with active source lease, valid clock position, and current fence is accepted into the durable activation ledger.
- Schedule source position uses clock position, not leader epoch, so one leader can emit multiple ticks safely.
- Duplicate tick ids are resolved by the schedule engine before a second activation is emitted.
- Rewound or malformed clocks fail as `ClockInvalid`.
- Missing schedule lease fails as `ScheduleLeaseMissing`.
- Lost/stale leader epoch or fencing fails as `ScheduleLeaseLost`.
- Revoked or expired source leases propagate from source authority.
- Exhausted chain hop propagates from the activation ledger as `LoopSuppressed`.
- Restarted engines recover durable schedule state and catch up only missed ticks.
- Embedded NATS KV stores schedule state durably across engine instances.

## RED Artifact

Expected failing proof before implementation:

- `T-SCHED-ACCEPT`: accepted due tick records source principal, lease, schedule id, tick id, fence, clock, and ledger status.
- `T-SCHED-CURSOR`: schedule ledger position advances by clock position, not leader epoch.
- `T-SCHED-DUP`: duplicate ticks fail at the schedule layer.
- `T-SCHED-CLOCK`: malformed or rewound clock fails at the schedule layer.
- `T-SCHED-LEASE`: missing schedule lease fails at the schedule layer while revoked source lease remains source-authority owned.
- `T-SCHED-CATCHUP`: restarted engine emits only missed ticks.
- `T-SCHED-EMBED`: embedded NATS KV persists schedule state and supports restart catch-up.

## Execution Notes

Keep schedule core pure and deterministic. Store wall-clock integration for a later operational loop. The engine consumes explicit ticks from a clock facade, writes durable schedule state, then delegates source authorization and durable activation acceptance. Durable schedule state belongs beside core contracts; the embedded adapter only supplies a JetStream KV-backed store.

## Verification Evidence

Initial RED evidence:

- `go test ./core -run 'TestSchedule|TestDurableLedgerAcceptsAllSourceCursors' -count=1` from `substrate/go` -> failed before implementation with missing `NewScheduleEngine`, `ScheduleTick`, `ScheduleTickDuplicate`, `ClockInvalid`, `ScheduleLeaseMissing`, `ScheduleLeaseLost`, `ScheduleEngine`, and `MemoryScheduleStore`.
- `go test ./embednats -run TestEmbeddedSchedule -count=1` from `substrate/go` -> failed before implementation with missing `NewKVScheduleStore`, `core.NewScheduleEngine`, `core.ScheduleTick`, and schedule source fields including `DueAt`, `AcquiredAt`, `ExpiresAt`, and `ClockID`.

GREEN evidence:

- Added schedule source fields to Go core activation source decoding.
- Changed schedule ledger position from leader epoch to parsed clock position while preserving schedule id, tick id, fencing token, leader epoch, and clock in the cursor.
- Added `ScheduleEngine`, `ScheduleTick`, `ScheduleStore`, `ScheduleState`, and `MemoryScheduleStore`.
- Added schedule-owned typed failures: `ScheduleConfigInvalid`, `ScheduleLeaseMissing`, `ScheduleLeaseLost`, `ClockInvalid`, `ScheduleTickDuplicate`, `CatchUpFailed`, and `RestartRecoveryFailed`.
- Added schedule acceptance, duplicate detection, malformed tick denial, clock rewind/malformed denial, leader/fencing denial, catch-up, restart recovery, and loop-suppression terminal tick handling.
- Added embedded `KVScheduleStore` backed by real JetStream KV.
- Added embedded NATS proof that schedule state and activation ledger records survive restart/catch-up.

Focused verification:

- `go test ./core -run 'TestSchedule|TestDurableLedgerAcceptsAllSourceCursors' -count=1` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/core`.
- `go test ./embednats -run TestEmbeddedSchedule -count=1` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/embednats`.
- `go test ./core -count=1` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/core`.
- `go test ./embednats -count=1` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/embednats`.
- `go test ./... -count=1` from `substrate/go` -> `ok` for `contract`, `core`, `edge`, `embednats`, and `frontend`.
- `git diff --check` -> clean.

Final verification:

- `bun run schema:parity` -> endgame contract tests `21 pass`, `0 fail`; Go packages passed.
- `bun run test` -> `56 pass`, `0 fail`, `393 expect() calls`.
- `bun run typecheck` -> frontend, SDK, and orchestrator typecheck passed.
- `bun run test:e2e` -> `1 pass`, `0 fail`, `16 expect() calls`.
- `bun run build` -> frontend Vite build and SDK tsdown build passed.
- `bun run pack:dry` -> produced `tinkabot-0.1.0.tgz` with `6 files`, unpacked size `188.70KB`.

## Wrap-Up

Shipped: schedule activation is durable, deterministic, restart-safe, and integrated with source authority plus the activation ledger. Wall-clock loops, script execution, materialization, and release proof remain later tasks.
