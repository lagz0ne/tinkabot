---
layer: task
topic: script-materializer-loop
status: complete
references:
  - ../approach/endgame-app.md
  - ../plan/endgame-app.md
  - ../plan/script-nats-cli-proof.md
  - ./activation-release-proof.md
  - ./activation-router-live-sources.md
  - ./activation-ledger-durability.md
---

# Script Materializer Loop

Diagram: https://diashort.apps.quickable.co/d/8e738818

## Objective

Connect accepted activation to managed script execution, mediated facade effects, and durable materialized outputs over NATS-visible stores.

## Scope

In scope:

- managed script records loaded from NATS KV.
- sandbox-compatible process config validation and local trusted process execution.
- framed stdout facade effects from scripts.
- projection materialization to NATS KV.
- artifact materialization to NATS Object Store.
- outside-in `nats` CLI trigger plus NATS-visible projection observation.
- typed failure attribution for script, protocol, facade, materializer, projection conflict, and cleanup.

Out of scope:

- Docker sandbox enforcement.
- raw NATS credentials, CLI handles, or subjects inside default scripts.
- product UI rendering.
- direct browser NATS WebSocket.
- wall-clock scheduling.
- broad script CRUD UI.

## Acceptance Contract

- A real `nats request` can trigger request/reply activation, accept it through the live router, execute a managed script, and materialize a projection visible with `nats kv get --raw`.
- The same run can write an artifact through Object Store.
- Script process env excludes raw NATS credential variables by default.
- Scripts emit facade frames; raw publish/subscribe/NATS vocabulary is denied before materialization.
- Duplicate or stale projection effects do not overwrite newer materialized truth.
- Process failure, malformed frame, denied facade effect, projection conflict, artifact write failure, and cleanup failure return typed attributed outcomes.

## RED Artifact

Expected failing proof before implementation:

- `T-SCRIPT-MAT-CORE`: core script runtime and materializer contracts are missing.
- `T-SCRIPT-MAT-FAILURES`: facade denial, script failure, projection conflict, and cleanup attribution are missing.
- `T-SCRIPT-MAT-NATS`: embedded-NATS script store, material store, local runner, and script loop are missing.

## Execution Notes

Keep the first implementation direct. Scripts are trusted local processes for now, but the contract must already be sandbox-ready: explicit path, env, timeout, resources, kill, cleanup, identity, framed protocol, and no raw NATS authority.

The materializer accepts only durable facade effects. Process stdout is not product truth. Product truth appears in KV/Object Store after materializer acceptance.

## Verification Evidence

Initial RED evidence:

- `go test ./core -run TestScriptRuntime -count=1` failed before implementation because core script runtime/materializer contracts were missing (`NewMemoryMaterialStore`, `ScriptRunnerFunc`, `ScriptInvocation`, `ScriptRun`, `ScriptEffect`, and related symbols).
- `go test ./embednats -run TestScriptMaterializerLoopFromNATSCLI -count=1` failed before implementation because embedded-NATS script store, material store, local runner, script loop, and NATS-visible projection/artifact surfaces were missing.

GREEN evidence:

- `go test ./core -run TestScriptRuntime -count=1 -v` -> `PASS`; proves accepted activation gating, script record revision/kind checks, raw NATS env filtering, facade denial, process failure, cleanup failure, projection conflict, artifact write failure, and materializer success.
- `go test ./embednats -run 'TestScriptMaterializerLoopFromNATSCLI|TestLocalScriptRunnerRejectsMalformedFrame|TestScriptLoopAttributesStatusWriteFailure' -count=1 -v` -> `PASS`; proves real `nats request` activation, mediated framed stdout effects, KV projection observation via `nats kv get --raw`, Object Store artifact observation via `nats object get`, duplicate no-rerun, malformed frame attribution, and status write attribution.
- Subagent NO-GO review found overgranted source/observer auth, accepted replay rerun risk, permissive Go decoding, non-canonical materialized projection shape, static event id overwrite, raw vocabulary gaps, and unbounded stdout/frame reads.
- Hardened proof: `go test ./embednats -run 'TestScriptMaterializerLoopFromNATSCLI|TestLocalScriptRunner|TestKVScriptStoreRejectsUnknownRecordField|TestScriptLoopDurableRunClaimRejectsAcceptedReplay|TestScriptLoopAttributesStatusWriteFailure' -count=1 -v` -> `PASS`; proves split caller/router/runtime/observer principals, no global `$JS.API.>` script-loop grants, scoped JetStream API subjects including explicit `$JS.API.INFO`, real denied caller ledger write, real denied observer material/Object Store writes, strict script record/effect decoding, `script.record.desc` acceptance, bounded framed stdout, durable accepted replay no-rerun, unique status event ids, canonical material projection KV shape, and artifact manifest materialization.
- Final security re-review: subagent returned GO on prior blockers for scoped NATS auth, required durable run claim, env filtering parity, strict frame/record decode, and `script.record.desc` support.
- `bun run schema:parity` -> `21 pass`, `0 fail`, plus Go packages `ok`; proves `script.record` and `script.effect` schema/SDK parity alongside the existing endgame contract authority.
- `go test ./... -count=1` from `substrate/go` -> `ok` for `contract`, `core`, `edge`, `embednats`, and `frontend`.
- `bun run test` -> `56 pass`, `0 fail`, `396 expect() calls`.
- `bun run typecheck` -> frontend, SDK, and orchestrator typecheck passed.
- `bun run build` -> frontend and SDK distribution build passed.
- `bun run test:e2e` -> `1 pass`, `0 fail`, `16 expect() calls`.
- `bun run pack:dry` -> `Total files: 6`, including `dist/index.cjs`, `dist/index.mjs`, `dist/index.d.cts`, and `dist/index.d.mts`.
- `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`.
- `bun run test:layers` -> `Ran 10 tests ... OK`.
- `git diff --check` -> passed.

## Wrap-Up

Accepted activation now drives managed script execution and materialized projection/artifact writes through NATS-visible stores. The proof covers real `nats` CLI trigger/observation, scoped caller/router/runtime/observer authority, strict script record and frame decoding, durable run claims, canonical projection/artifact materialization, and attributed failures.

This Task is closed in `f93b705 feat: add script materializer loop`. Docker sandboxing, product UI rendering, direct browser NATS WebSocket, wall-clock scheduler loops beyond the schedule engine proof, and broad script CRUD UI remain later work.
