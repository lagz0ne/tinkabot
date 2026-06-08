---
layer: task
topic: nats-script-runtime-substrate-record-store
references:
  - ../approach/platform-structure.md
  - ../plan/nats-script-runtime-traced-tdd.md
  - ../plan/nats-script-runtime.md
  - ../approach/nats-script-runtime.md
---

# NATS Script Runtime Substrate And Record Store Task

## Platform Reset Supersession

`docs/matched-abstraction/approach/platform-structure.md` supersedes this task wherever substrate ownership or release platform shape is concerned. This task remains historical implementation and regression evidence for the earlier Bun substrate and record-store slice.

## Task Brief

Implement the first traced-TDD runtime slice: runtime substrate and script record store. This task covers T01 through T04 only. Later metadata, permissions, IPC, process runtime, event trail, execution exchange, and vertical proof work stay outside this slice.

Historical scope included the Bun TypeScript package setup, `src/nats-script-runtime/*`, and tests under `tests/nats-script-runtime/`.

## Acceptance Contract

The historical task was accepted when runtime substrate errors and script record-store errors were machine-inspectable through `TinkabotRuntimeError`, runtime substrate started real embedded JetStream NATS with isolated storage, script record store read exact KV revisions, and all T01 through T04 failure branches mapped to the owning layer's declared errors.

Historical required observed behavior:

- Historical T01: embedded JetStream started, exposed KV, stopped rerun-safely, and mapped startup, availability, and cleanup failures.
- T02: unknown substrate lifecycle values become `SubstrateCritical`.
- T03: missing records, deleted or stale records, revision mismatch, write conflict, and exact revision reads are owned by `ScriptRecordStore`.
- T04: lower substrate failures during KV access transform to `RecordPersistenceFailed` with origin preserved.

## RED Artifact

- `bun test` -> failed because `../../src/nats-script-runtime/index` did not exist for `runtime-substrate.test.ts` and `script-record-store.test.ts`.

## Execution Notes

The first GREEN pass added:

- `package.json`, `bun.lock`, and `tsconfig.json` for Bun tests plus native TypeScript checking.
- `src/nats-script-runtime/errors.ts` for typed runtime errors.
- `src/nats-script-runtime/runtime-substrate.ts` for embedded NATS lifecycle and KV bucket access.
- `src/nats-script-runtime/script-record-store.ts` for KV-backed script records and exact revision reads.
- `src/nats-script-runtime/index.ts` as the slice export surface.
- `tests/nats-script-runtime/runtime-substrate.test.ts` for T01 and T02.
- `tests/nats-script-runtime/script-record-store.test.ts` for T03 and T04.

Review kept the slice narrow. It did not add metadata validation, subject permission checks, framed stdio RPC, process execution, event publication, or request/reply orchestration.

## Verification Evidence

- `bun test` -> `7 pass, 0 fail`.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit` completed successfully.
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.
- `find . -type d -name __pycache__ -print` -> no matches.

## Wrap-Up Announcement

The final response must state that T01 through T04 are implemented and verified, and that the next traced-TDD slice starts with RED metadata/schema and imports/permissions tests.
