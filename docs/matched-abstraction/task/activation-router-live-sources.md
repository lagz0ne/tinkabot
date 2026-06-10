---
layer: task
topic: activation-router-live-sources
status: complete
references:
  - ../approach/endgame-app.md
  - ../plan/activation-foundation.md
  - ./activation-contract-authority.md
  - ./activation-ledger-durability.md
  - ./activation-source-authority.md
---

# Activation Router Live Sources

Diagram: https://diashort.apps.quickable.co/d/0ab25edc

## Objective

Prove that live NATS source observations can become accepted activation records without redefining source authority or durable ledger policy.

The router owns observation normalization for request/reply, ordinary subjects, KV watches, Object Store metadata changes, and JetStream stream consumers. It consumes source-scoped authority and the durable activation ledger. It does not execute scripts, materialize projections, schedule ticks, serve artifacts, or grant browser/script raw NATS access.

## Scope

In scope:

- request/reply and ordinary subject subscriptions over embedded NATS.
- KV watch observations from real JetStream KV.
- Object Store metadata observations from the real Object Store meta stream, preserving JetStream metadata sequence.
- JetStream stream consumption with `Msg.Metadata`.
- explicit source-frame request/message identity headers.
- source-authority check before durable ledger acceptance.
- router-owned typed failures for malformed live frames and listen/watch/consume setup.

Out of scope:

- schedule activation.
- script execution.
- materialization or artifact serving.
- product UI.
- direct browser NATS WebSocket.
- sandboxing.

## Acceptance Contract

- A live request/reply message with a request id becomes an accepted `request_reply` activation and replies with the ledger status.
- A live subject message records the concrete observed subject and message id while preserving the wildcard aperture.
- A KV watch records bucket, key, operation, revision, and a durable resume cursor.
- An Object Store change records bucket, object name, digest, and Object Store meta-stream sequence as the source position.
- A stream message records stream, consumer, subject, stream sequence, consumer sequence, and delivery attempt from JetStream metadata.
- Missing source-frame identity or metadata fails as router-owned `SourceMalformed`.
- Router setup failures are attributed to `LiveSourceRouter` with typed listen/watch/consume kinds.
- Source-authority denials, revoked leases, duplicates, stale cursors, and durable ledger outcomes propagate with their original owning layer.
- A real `nats` CLI request can trigger the request/reply router path through the embedded NATS seam.

## RED Artifact

Write failing embedded-NATS tests that require a live source router to:

- listen to request/reply and subject sources;
- watch KV entries and Object Store metadata changes;
- consume JetStream stream messages with metadata;
- normalize each live observation into the canonical activation source position;
- preserve source principal, source lease, chain, concrete observed coordinate, dedupe, cursor, and attribution;
- surface router-owned malformed/config/listen/watch/consume failures while propagating source-authority and ledger failures.

This RED proves that contract authority, source authority, and ledger durability are necessary but insufficient without a live router boundary.

Initial RED evidence:

- `go test ./embednats -run 'TestSourceRouter' -count=1` from `substrate/go` failed before implementation with missing `HeaderRequestID`, `HeaderMessageID`, `NewSourceRouter`, `RequestReplyListenFailed`, `SubjectSubscribeFailed`, `KVWatchFailed`, `ObjectWatchFailed`, `SourceRouter`, `RouterResult`, and `Route`.

## Execution Notes

Implement a small Go router in `substrate/go/embednats` that converts battle-tested NATS Go observations into `core.Activation` attempts:

- request/reply and plain subject messages use explicit source headers for request/message identity;
- KV entries use bucket, key, operation, revision, and a resume cursor;
- Object Store changes use the Object Store metadata stream message so the adapter preserves the meta-stream sequence;
- stream messages use JetStream `Msg.Metadata`;
- `core.AuthorizeSource` runs before `core.DurableLedger.Accept`;
- duplicate, stale cursor, loop suppression, and durable write behavior remain ledger-owned.

The Object Store path intentionally subscribes to `$O.<bucket>.M.>` and binds to `OBJ_<bucket>` instead of using `ObjectStore.Watch()`, because `ObjectStore.Watch()` returns `ObjectInfo` but does not expose the JetStream metadata needed for the source position.

## Verification Evidence

GREEN evidence:

- Added `SourceRouter`, `Route`, and `RouterResult` in `substrate/go/embednats`.
- Added router-owned typed failures: `RouterConfigInvalid`, `RequestReplyListenFailed`, `SubjectSubscribeFailed`, `KVWatchFailed`, `ObjectWatchFailed`, `StreamConsumeFailed`, `SourceMalformed`, and `RouterCritical`.
- Added live request/reply, subject, KV, Object Store meta-stream, and stream router paths over embedded NATS.
- Added normalization helpers for source identity, concrete source coordinates, dedupe keys, activation ids, cursors, and source positions.
- Added request/reply proof through the real `nats` CLI with source headers.
- Added router tests for malformed frame, denied neighbor, duplicate, stale stream cursor, revoked lease, and router-layer attribution.

Final verification:

- `go test ./embednats -run 'TestSourceRouter' -count=1` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/embednats`.
- `go test ./embednats -count=1` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/embednats`.
- `go test ./... -count=1` from `substrate/go` -> `ok` for `contract`, `core`, `edge`, `embednats`, and `frontend`.
- `git diff --check` -> clean.
- `bun run schema:parity` -> endgame contract tests `21 pass`, `0 fail`; Go packages passed.
- `bun run test` -> `56 pass`, `0 fail`, `393 expect() calls`.
- `bun run typecheck` -> frontend, SDK, and orchestrator typecheck passed.
- `bun run test:e2e` -> `1 pass`, `0 fail`, `16 expect() calls`.
- `bun run build` -> frontend Vite build and SDK tsdown build passed.
- `bun run pack:dry` -> produced `tinkabot-0.1.0.tgz` dry-run package.
- `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`.
- `bun run test:layers` -> `Ran 10 tests ... OK`.

## Wrap-Up

Shipped: live NATS request/reply, subject, KV, Object Store, and stream observations normalize into source-authorized durable activation records, with request/reply also proven through the real `nats` CLI. Schedule activation, script execution, materialization, and release proof remain later tasks.
