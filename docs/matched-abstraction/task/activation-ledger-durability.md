---
layer: task
topic: activation-ledger-durability
references:
  - ../approach/endgame-app.md
  - ../approach/go-substrate.md
  - ../plan/activation-foundation.md
  - ../plan/go-substrate.md
  - ./activation-contract-authority.md
---

# Activation Ledger Durability Task

Diagram: https://diashort.apps.quickable.co/d/32dc325b

## Objective

Make activation ledger behavior durable and restart-safe before live source routing exists. This task consumes canonical activation fixtures and proves durable acceptance, source cursor persistence, duplicate resolution, loop suppression records, replay/catch-up, source lease binding, restart recovery, and write-conflict mapping through a Go store contract backed by embedded NATS JetStream KV for durable proof.

## Scope

This task owns one executable unit: Go activation ledger durability.

In scope:

- durable activation acceptance records.
- source cursor extraction from canonical activation source kinds.
- duplicate resolution by dedupe key.
- loop suppression records.
- replay/catch-up records after a cursor.
- source lease binding and non-active lease denial.
- restart recovery through embedded NATS JetStream KV.
- a narrow in-memory store for unit-only denial and conflict checks.
- durable write conflict and replay cursor failure mapping.

Out of scope:

- source-scoped NATS auth compilation.
- live NATS subscribers, watchers, consumers, or schedule runners.
- script execution.
- materialized projection writes.
- browser UI or artifact serving.

## Acceptance Contract

- First acceptance persists an activation record with activation id, dedupe key, source id, source kind, source principal, source lease id, source position, replay cursor, chain id, and `accepted` status.
- Reaccepting the same dedupe key resolves as `duplicate` without creating a second accepted record.
- Chain hop exhaustion records loop suppression and returns a ledger-owned `LoopSuppressed` error.
- Source cursor regression is rejected as `StaleCursor`.
- Replay after a stored cursor returns later accepted records in order.
- A new ledger instance using the same store preserves duplicate and cursor state.
- Source lease must be active and must bind to the passed lease id when a lease id is supplied.
- Store write conflicts map to ledger-owned `WriteConflict`; replay cursor failures map to `ReplayCursorFailed`.

## RED Artifact

Expected failing tests before implementation:

- `T-ACT-LEDGER-ACCEPT`: first acceptance stores activation, source identity, lease identity, position, and replay cursor.
- `T-ACT-LEDGER-DUPLICATE`: duplicate dedupe key resolves without second accepted record.
- `T-ACT-LEDGER-LOOP`: hop exhaustion records suppression and returns `LoopSuppressed`.
- `T-ACT-LEDGER-CURSOR`: stale cursor is rejected.
- `T-ACT-LEDGER-REPLAY`: replay after cursor returns later records in order.
- `T-ACT-LEDGER-LEASE`: non-active or mismatched source lease is denied before durable write.
- `T-ACT-LEDGER-RESTART`: a new ledger using the same store preserves duplicate/cursor state.
- `T-ACT-LEDGER-CONFLICT`: write conflict and cursor failure are mapped to ledger-owned typed errors.

## Execution Notes

Keep Go core free of direct NATS imports by depending on a ledger store contract. Use embedded NATS JetStream KV for durable behavior tests; keep `MemoryLedgerStore` only for narrow unit checks such as forced write conflicts and no-write assertions.

Keep source authority out of this layer. The ledger consumes schema-valid owner-layer fixtures but does not decide whether a source principal may observe a subject, KV key, object, stream, or schedule.

## Verification Evidence

Task prep and RED evidence:

- `sed -n '1,260p' tasks/todo.md` -> next task is `activation-ledger-durability`.
- `sed -n '1,620p' substrate/go/core/core.go` -> current `Ledger` is in-memory and records only dedupe, loop, lease status, and a fake cursor-failed flag.
- `go test ./core -count=1` from `substrate/go` -> failed before implementation with missing `NewMemoryLedgerStore`, missing `NewDurableLedger`, and missing durable `StaleCursor` path.

GREEN evidence:

- Added `DurableLedger`, `LedgerStore`, and `MemoryLedgerStore` in `substrate/go/core`.
- Added `KVLedgerStore` in `substrate/go/embednats` backed by real embedded NATS JetStream KV.
- Added durable activation records with source id, source kind, source principal id, source lease id, source position, source cursor, replay cursor, chain id, and status.
- Added source cursor extraction for `request_reply`, `command_acceptance`, `subject`, `kv`, `object`, `stream`, and `schedule`.
- Added duplicate resolution, loop suppression recording, stale cursor rejection, replay after cursor, unknown replay cursor failure, source lease binding, write-conflict mapping, and restart behavior through a shared store.
- Added encoded replay cursor framing so arbitrary source ids and source cursors cannot collide through delimiter text.
- Source lease id is mandatory on both activation and caller lease, and it must match.
- Source principal kind must match the concrete activation source kind before any durable write.
- Added `substrate/go/core/ledger_durability_test.go` to cover accept, every source cursor kind, replay cursor collision safety, duplicate, replay, unknown replay cursor, restart replay/cursor state, loop suppression, stale cursor, lease denial with no write, source kind mismatch with no write, write conflict with no write, and replay cursor failure.
- Added `substrate/go/embednats/ledger_test.go` to prove durable ledger accept, duplicate, restart, replay, stale cursor behavior, and every canonical source kind over an embedded NATS runtime and JetStream KV bucket.

Review hardening:

- Layer reviewer passed; noted the fake durable store must not be treated as final live persistence by later layers.
- Tests reviewer initially blocked on missing all-source cursor tests, restart replay/cursor proof, and no-write denial assertions. Those were fixed and re-review passed.
- Risk reviewer initially blocked on replay cursor text collisions, optional source lease id binding, missing source kind binding, and missing collision/missing-lease proof. Those were fixed and re-review passed.
- User correction replaced fake-first proof with embedded NATS KV proof. `MemoryLedgerStore` remains only as a narrow unit seam; durable behavior is now verified against embedded NATS.

Final verification:

- `go test ./core -count=1` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/core`.
- `go test ./embednats -run 'TestEmbeddedLedger' -count=1` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/embednats`.
- `go test ./embednats -count=1` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/embednats`.
- `go test ./... -count=1` from `substrate/go` -> `ok` for `contract`, `core`, `edge`, and `embednats`.
- `bun run schema:parity` -> endgame contract tests `21 pass`, `0 fail`; Go packages passed.
- `bun run test` -> `52 pass`, `0 fail`, `374 expect() calls`.
- `bun run test:e2e` -> `1 pass`, `0 fail`, `16 expect() calls`.
- `bun run typecheck` -> passed.
- `bun run build` -> passed.
- `bun run pack:dry` -> produced `tinkabot-0.1.0.tgz` dry-run package.
- `bun run validate:layers` -> layer validation passed.
- `bun run test:layers` -> `10` layer validation tests passed.
- `git diff --check` -> clean.

## Wrap-Up Announcement

When complete, announce that activation ledger durability now owns durable acceptance, source cursors, duplicate resolution, loop suppression records, replay/catch-up, restart recovery, lease binding, and write-conflict mapping. Also state that source-scoped auth, live source routing, schedules, scripts, materialization, and release proof remain later activation foundation tasks.
