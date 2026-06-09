---
layer: task
topic: activation-release-proof
status: complete
references:
  - ../approach/endgame-app.md
  - ../plan/activation-foundation.md
  - ./activation-contract-authority.md
  - ./activation-ledger-durability.md
  - ./activation-source-authority.md
  - ./activation-router-live-sources.md
  - ./activation-schedule-engine.md
  - ./command-acceptance.md
  - ./browser-isolation-proof.md
---

# Activation Release Proof

Diagram: https://diashort.apps.quickable.co/d/2e24d446

## Objective

Compose the activation foundation through real embedded NATS surfaces for live router sources, plus NATS-backed durable schedule stores, and prove that outcomes agree with the inside-out ownership map.

## Scope

In scope:

- request/reply proof through real `nats` CLI.
- subject, KV, Object Store, and stream proof through real embedded NATS.
- schedule proof through direct engine acceptance over real embedded NATS KV schedule state and ledger stores.
- `command_acceptance` cited as peer outside-in evidence from Browser Isolation Proof, not reimplemented as a live source-router source.
- allowed, malformed, denied-neighbor, duplicate, stale cursor, revoked lease, and loop-suppressed outcomes.
- attribution that names the owning layer for each outcome.

Out of scope:

- script execution.
- materialization.
- product UI.
- direct browser NATS WebSocket.
- wall-clock scheduler loops.
- NATS tick adapter or facade.
- sandboxing.

## Acceptance Contract

- Every live-router source kind (`request_reply`, `subject`, `kv`, `object`, `stream`) can produce an accepted durable activation record through a real embedded NATS observation.
- `command_acceptance` release confidence remains peer-owned by Browser Isolation Proof, where real embedded NATS request/reply returns canonical accepted and rejected `command.acceptance` statuses.
- Schedule can produce an accepted durable activation record through real embedded NATS KV state and ledger stores.
- Request/reply can be triggered by the installed `nats` CLI.
- Malformed live frames are owned by `LiveSourceRouter`.
- Denied-neighbor and revoked-lease cases are owned by `SourceAuthority`.
- Duplicate is an `ActivationLedger` status outcome; stale cursor is an `ActivationLedger` typed failure.
- Loop suppression is owned by `ActivationLedger`, while schedule records terminal tick state.
- Every release-proof result includes scenario name, owner, kind/status, and durable record where one exists.

## RED Artifact

Expected failing proof before implementation:

- `T-REL-ACT-ALLOW`: request/reply CLI, subject, KV, Object Store, stream, and NATS-backed schedule accepted outcomes all carry durable activation records.
- `T-REL-ACT-FAILURES`: malformed, denied-neighbor, duplicate, stale cursor, revoked lease, and loop-suppressed scenarios preserve owner attribution.
- `T-REL-ACT-ATTR`: release proof cannot pass on raw errors or happy-path records without a normalized outcome shape.

## Execution Notes

Do not add new activation semantics in this task. The release proof is a composition harness and test suite over existing authority, router, schedule, and ledger behavior.

Stale cursor setup may seed a higher source cursor from real JetStream metadata, but the asserted stale failure must enter through the live stream router. Schedule proof does not claim a NATS tick source until a tick adapter or facade exists.

## Verification Evidence

Initial RED evidence:

- `go test ./embednats -run TestActivationReleaseProof -count=1` from `substrate/go` -> RED failed before GREEN with missing `ReleaseOutcome` and `ProofOutcome` symbols.

Targeted GREEN evidence:

- `go test ./embednats -run TestActivationReleaseProof -count=1` from `substrate/go` -> passed after release proof helper and tests were hardened.

Final verification:

- `go test ./embednats -run TestActivationReleaseProof -count=1` from `substrate/go` -> `ok`.
- `go test ./... -count=1` from `substrate/go` -> `ok` for `contract`, `core`, `edge`, `embednats`, and `frontend`.
- `bun run schema:parity` -> contract tests and Go parity passed.
- `bun run test` -> `56 pass`, `0 fail`, `393 expect() calls`.
- `bun run typecheck` -> frontend, SDK, and orchestrator typecheck passed.
- `bun run test:e2e` -> `1 pass`, `0 fail`, `16 expect() calls`.
- `bun run build` -> frontend and SDK builds passed.
- `bun run pack:dry` -> `tinkabot-0.1.0.tgz`, 6 files, `188.70KB`.
- `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`.
- `bun run test:layers` -> `Ran 10 tests ... OK`.
- `git diff --check` -> passed.
- Focused no-slop scan over release proof symbols, placeholder evidence, overclaiming phrases, direct-only paths, and memory-store fallback found only expected RED evidence text and existing peer inside-out tests.

## Wrap-Up

When complete, announce that activation foundation release proof composes contract, source authority, router, schedule, command-acceptance peer evidence, and ledger behavior through real NATS-mediated surfaces or NATS-backed durable stores as scoped above. Also state that script execution, materialization, product UI, direct browser NATS WebSocket, NATS tick adapter, and sandboxing remain later tasks.
