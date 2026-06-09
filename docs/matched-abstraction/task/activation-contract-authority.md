---
layer: task
topic: activation-contract-authority
references:
  - ../approach/endgame-app.md
  - ../approach/go-substrate.md
  - ../plan/activation-foundation.md
  - ../plan/go-substrate.md
---

# Activation Contract Authority Task

Diagram: https://diashort.apps.quickable.co/d/32dc325b

## Objective

Make activation source shape canonical before live source routing is implemented. This task extends the neutral contract packet so request/reply, command acceptance, ordinary subject messages, KV changes, Object Store changes, stream messages, and schedule ticks all use one source principal, source lease, cursor, chain, dedupe, provenance, capability, and parity model.

## Scope

This task owns one executable unit: contract authority for activation sources.

In scope:

- canonical JSON Schema source updates for activation source kinds.
- positive and negative fixtures for every source kind.
- source principal, source lease, and cursor shape for subject, KV, Object Store, stream, and schedule activation.
- wildcard subject aperture rules with concrete observed subject recording and sibling-denial fixture rows.
- TypeScript/Zod or checked SDK validation parity.
- Go validation parity under `substrate/go`.
- explicit structural reject fixtures for malformed source, missing lease shape, invalid cursor, invalid wildcard aperture shape, missing observed subject, invalid source position, and missing provenance.
- schema-valid policy fixture rows that carry expected owner layer for denied neighbor, wildcard overreach, stale revision, revoked lease, and expired lease.

Out of scope:

- live NATS watchers or subscribers.
- durable ledger implementation beyond contract shape consumed by the later ledger task.
- source principal compilation beyond the contract fields needed by source authority.
- schedule execution.
- script execution.
- materialized projection writes.
- browser UI or artifact serving.

## Acceptance Contract

- `activation.intent.source` accepts exactly these source kinds: `request_reply`, `command_acceptance`, `subject`, `kv`, `object`, `stream`, and `schedule`.
- Every source kind carries enough position information to replay, dedupe, catch up, or reject stale data without inspecting lower-layer internals.
- Every source kind binds to a `sourcePrincipal` and `sourceLease` envelope and preserves provenance, capability context, chain, dedupe key, observed time, and script revision compatibility.
- Wildcard subscription patterns are concrete source apertures under authoritative prefixes, and accepted activations record the concrete observed subject.
- Wildcard fixtures include an allowed aperture and a denied sibling under the same authoritative prefix so broad family watches are not silently accepted.
- KV sources expose revision plus resume cursor in contract shape.
- Object Store sources expose digest or revision plus an adapter-recorded object meta-stream sequence; the contract may not depend on the default object watcher losing NATS message metadata.
- Stream sources expose NATS message metadata fields needed for replay: stream sequence, consumer sequence, subject, and delivery attempt.
- Schedule sources expose deterministic tick identity, due time, owner principal, leader epoch, fencing token, acquired time, expiry time, clock id, and clock position in contract shape.
- Positive fixtures validate across canonical schema, SDK validation, and Go validation.
- Structural negative fixtures fail closed for unknown source kind, malformed cursor, missing lease shape, missing provenance, invalid wildcard aperture shape, missing observed concrete subject, and invalid source-position fields.
- Policy fixture rows for denied neighbor, wildcard overreach, stale revision, revoked lease, and expired lease remain schema-valid and carry `expect.ownerLayer` for later source-authority or ledger tasks.
- The contract does not grant authority. It only carries the data source authority and ledger tasks need to decide.

## RED Artifact

Expected failing tests before implementation:

- `T-ACT-CONTRACT-SOURCES`: schema and parity tests fail because `activation.intent.source` currently accepts only `request_reply` and `command_acceptance`.
- `T-ACT-CONTRACT-CURSOR`: subject, KV, Object Store, stream, and schedule fixtures require source position fields that do not yet exist.
- `T-ACT-CONTRACT-LEASE`: every source fixture requires source principal and lease shape with principal id, lease id, lease status, source id, source kind, app revision, schema revision, script revision when bound, and authority reference.
- `T-ACT-CONTRACT-WILDCARD`: wildcard source apertures must preserve authoritative prefixes, accepted activations must record concrete observed subjects, and fixture rows must include an allowed aperture plus denied sibling under the same prefix.
- `T-ACT-CONTRACT-PARITY`: canonical schema, SDK validation, and Go validation must agree on accept and reject outcomes for all activation source fixtures.
- `T-ACT-CONTRACT-DENIALS`: malformed source, missing lease shape, invalid cursor, missing provenance, invalid wildcard aperture shape, missing observed subject, and invalid source position fail at Contract Authority.
- `T-ACT-POLICY-OWNER-TAGS`: schema-valid denied neighbor, wildcard overreach, stale revision, revoked lease, and expired lease fixtures carry `expect.ownerLayer` so later source-authority and ledger tasks own those denials.

Concrete RED fixture matrix:

| Fixture path | Source kind | Expected validity | Expected owner |
| --- | --- | --- | --- |
| `fixtures/valid/activation-source-subject.json` | `subject` | valid | ContractAuthority |
| `fixtures/valid/activation-source-kv.json` | `kv` | valid | ContractAuthority |
| `fixtures/valid/activation-source-object.json` | `object` | valid | ContractAuthority |
| `fixtures/valid/activation-source-stream.json` | `stream` | valid | ContractAuthority |
| `fixtures/valid/activation-source-schedule.json` | `schedule` | valid | ContractAuthority |
| `fixtures/invalid/activation-source-unknown-kind.json` | unknown | invalid | ContractAuthority |
| `fixtures/invalid/activation-source-missing-lease.json` | `subject` | invalid | ContractAuthority |
| `fixtures/invalid/activation-source-invalid-cursor.json` | `kv` | invalid | ContractAuthority |
| `fixtures/invalid/activation-source-missing-observed-subject.json` | `subject` | invalid | ContractAuthority |
| `fixtures/valid/activation-source-denied-neighbor.json` | `subject` | valid | SourceAuthority |
| `fixtures/valid/activation-source-wildcard-overreach.json` | `subject` | valid | SourceAuthority |
| `fixtures/valid/activation-source-stale-cursor.json` | `stream` | valid | ActivationLedger |
| `fixtures/valid/activation-source-revoked-lease.json` | `kv` | valid | SourceAuthority |
| `fixtures/valid/activation-source-expired-lease.json` | `schedule` | valid | SourceAuthority |

The SDK RED test lives in `packages/sdk/tests/endgame-contract/contract-authority.test.ts` and must assert fixture validity, error kind for invalid rows, and owner layer tags. The Go RED test lives in `substrate/go/contract/registry_test.go` and must assert the same schema-valid and schema-invalid outcomes from `parity.cases.json`.

## Execution Notes

Start with fixtures and failing parity tests. The current schema proves the red condition because `activation.intent.source` has only two variants. Add contract shape before router code so the later Go tasks cannot invent local activation fields.

Keep local names short in code, but keep public schema fields explicit where they carry safety: `sourcePrincipal`, `sourceLease`, `authorityRef`, `cursor`, `observedSubject`, `bucket`, `key`, `revision`, `stream`, `streamSequence`, `consumerSequence`, `deliveryAttempt`, `objectMetaSequence`, `scheduleId`, `tickId`, `dueAt`, `leaderEpoch`, `fencingToken`, `acquiredAt`, `expiresAt`, `clockId`, and `clock`.

Do not add broad source placeholders. Use concrete subject values or concrete wildcard patterns with authoritative left-side prefixes.

## Verification Evidence

Task prep evidence:

- `curl -s -X POST https://diashort.apps.quickable.co/render ...` -> `https://diashort.apps.quickable.co/d/32dc325b`.
- `sed -n '250,330p' schemas/endgame/v1/contract.schema.json` -> `activation.intent.source` currently accepts only `request_reply` and `command_acceptance`.
- `sed -n '1,260p' docs/matched-abstraction/plan/go-substrate.md` -> showed `activation-source-router` as the direct next slice before this activation foundation lift.
- `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`.
- `bun run test:layers` -> `Ran 10 tests ... OK`.
- Activation reinforcement subagents -> all returned `BLOCKING: yes` before doc hardening; confirmed contract-policy ownership blur, missing owner-layer fixture matrix, source principal/lease underspecification, wildcard sibling-denial gap, Object Store watcher cursor risk, and schedule fencing gap.
- Activation reinforcement arbiter -> `BLOCKING: no`; confirmed blocker classes resolved with no remaining patch needed.

RED evidence:

- `bun test packages/sdk/tests/endgame-contract/contract-authority.test.ts` -> failed because new schema-valid activation source fixtures were rejected as `ContractInvalid`.
- `go test ./contract -count=1` from `substrate/go` -> failed on new schema-valid activation source fixtures because canonical schema still admitted only `request_reply` and `command_acceptance`.

GREEN evidence:

- `bun test packages/sdk/tests/endgame-contract/contract-authority.test.ts` -> `2 pass`, `0 fail`, `81 expect() calls`.
- `go test ./contract -count=1` from `substrate/go` -> `ok`.
- `bun test packages/sdk/tests/endgame-contract/command-acceptance.test.ts` -> `9 pass`, `0 fail`, `53 expect() calls`.
- `bun test packages/sdk/tests/endgame-contract/substrate-edge-bootstrap.test.ts` -> `4 pass`, `0 fail`, `17 expect() calls`.
- `go test ./... -count=1` from `substrate/go` -> `ok` for `contract`, `core`, `edge`, and `embednats`.
- `bun run schema:parity` -> endgame contract tests `21 pass`, `0 fail`, `192 expect() calls`; Go packages passed.
- `bun run test` -> `52 pass`, `0 fail`, `374 expect() calls`.
- `bun run typecheck` -> SDK and orchestrator typecheck passed.
- `bun run build` -> SDK CJS, ESM, and declaration bundles emitted.
- `bun run pack:dry` -> `tinkabot-0.1.0.tgz`, 6 files.
- `bun run validate:layers`, `bun run test:layers`, and `git diff --check` passed.

## Wrap-Up Announcement

Activation contract authority is complete. Canonical schema, SDK validation, Go validation, fixtures, and parity cases now cover all activation source kinds with source principal, source lease, cursor, wildcard aperture, provenance, and owner-layer fixture tags. Live source routing, durable ledger behavior, source-scoped auth compilation, schedules, scripts, materialization, and release proof remain later activation foundation tasks.
