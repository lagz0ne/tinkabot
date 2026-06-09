# Tinkabot Handoff

## Current State

- Repo: `lagz0ne/tinkabot`, private, branch `main`.
- Remote: `origin git@github.com:lagz0ne/tinkabot.git`.
- Latest pushed commit: `5c30a1f chore: add terse coding skill`.
- Worktree baseline before this cleanup: clean against `origin/main`.
- Root role: orchestration only.
- Current implementation lives in `packages/sdk` and `substrate/go`.
- Active/future lanes:
  - `schemas`: canonical JSON Schema and codegen authority.
  - `substrate/go`: Go embedded-NATS/auth/process/Docker-facing substrate.
  - `apps/frontend`: Vite trusted shell.

## Active Goal

Reach the Tinkabot endgame with matched-abstraction docs, inside-out ownership proof, outside-in real-NATS proof, and NATS as the system seam for release confidence.

## Milestone Workflow

1. DONE: `endgame-contract-authority`: neutral schemas, fixtures, TS/Zod target, Go validation target, and parity command.
2. DONE: `managed-auth-subjects`: identity/capability provenance, subject taxonomy, NATS auth compilation fixtures, lease/revocation/expiration proof, advanced capability denial, bounded responses, and export/exposure pairing.
3. DONE: `command-acceptance`: durable intent acceptance, atomic idempotency, required command ids, capability context binding, stale-revision denial, capability lease denial, status materialization, activation handoff.
4. DONE: `substrate-edge-bootstrap`: Go substrate boundary plus Browser Edge credential/artifact bootstrap over shared contracts.
5. DONE: `go-substrate-core`: Go-owned embedded NATS lifecycle, HA/scale topology, auth render, credential leases, store substrate, activation ledger, process boundary, gateway substrate, attribution.
6. DONE: `embedded-nats-adapter`: Go substrate contracts attach to a real embedded NATS runtime with JetStream, auth load path, WebSocket posture, topology probes, and drain/shutdown behavior.
7. DONE: `activation-contract-authority`: canonical activation source contracts, fixtures, SDK validation, and Go validation for all source kinds under the activation foundation plan.
8. DONE: `activation-ledger-durability`: durable activation records, source cursors, duplicate resolution, loop suppression, replay/catch-up, and restart behavior.
9. DONE: `activation-source-authority`: source-scoped NATS auth, permissions, imports, exports, bounded responses, revocation, and denied-neighbor proof.
10. DONE: `frontend-isolation-layer`: Vite shell, opaque generated iframe fixture, leased source-window message path, raw-authority denial, and Go-embedded frontend build.
11. DONE: `browser-isolation-proof`: gateway Command Acceptance smoke proof plus service-worker scope/header denial.
12. DONE: `activation-router-live-sources`: request/reply, subject subscriptions, KV/Object/Stream watches, and accepted activation normalization over live NATS.
13. NEXT: `activation-schedule-engine`: durable schedule state, lease/leadership, fake-clock tests, catch-up, restart recovery, tick dedupe, and loop safety.
14. `activation-release-proof`: outside-in real NATS activation scenarios tied back to inside-out contract, ledger, source authority, router, and schedule proof.
15. `script-materializer-loop`: mediated script execution, accepted effects, materialized projections/artifacts, cleanup.
16. `release-spine`: centralized ops evidence manifest with outside-in real NATS proof and inside-out ownership proof.

## Operating Rules

- Read this file first at session start.
- Use RED-GREEN-TDD for non-trivial changes.
- Use `matched-abstraction-thinking` for architecture, planning, task handoff, and layer docs.
- Use `triage-three` before presenting non-trivial concepts or architecture choices; skip for direct execution/status.
- Use `be-lazy` when coding: short names, compiler-backed inference, direct code, explicit only at public/safety/schema/error boundaries.
- Verify before done. Prefer narrow meaningful checks, then record evidence here only when it changes the current handoff.

## Current Direction

- Endgame Approach is the current top-level app authority: `docs/matched-abstraction/approach/endgame-app.md`.
- Endgame Plan is the current decomposition authority: `docs/matched-abstraction/plan/endgame-app.md`.
- Browser Isolation Approach/Plan are the current generated-content isolation authority: `docs/matched-abstraction/approach/browser-isolation.md`, `docs/matched-abstraction/plan/browser-isolation.md`.
- Go Substrate Approach is sealed for `go-substrate-core` and downstream Go substrate planning: `docs/matched-abstraction/approach/go-substrate.md`.
- Activation Foundation Plan is the current activation decomposition authority: `docs/matched-abstraction/plan/activation-foundation.md`.
- Script NATS CLI Proof Plan is the current script-side outside-in proof authority: `docs/matched-abstraction/plan/script-nats-cli-proof.md`.
- The product loop is source/artifact -> materialized projection -> browser intent -> durable backend acceptance -> activation -> script execution -> attributed event/projection update.
- Go owns substrate authority: embedded NATS lifecycle, NATS-native HA/scale posture, auth, process lifecycle, Docker/sandboxing direction, connection policy, activation ledger, artifact gateway, execution attribution.
- Vite owns the trusted browser shell. Generated browser content remains an opaque sandboxed receiver and intent emitter.
- Schema/SDK owns shared contract shape. JSON Schema is the first neutral source; generated or checked Zod, TS types, Go validators/types, and fixtures follow it.
- Existing Bun/TypeScript runtime and `@lagz0ne/nats-embedded` work is regression evidence and SDK material, not current substrate authority.
- NATS is the system seam. Inside-out tests localize ownership; outside-in tests must prove cross-lane behavior through real NATS-mediated surfaces before a release-shaped slice counts as endgame-ready.
- Default scripts stay NATS-agnostic process contracts. Runtime facade mediates NATS publish/progress/import requests.
- Script-side outside-in proof uses real `nats` CLI commands against embedded NATS to trigger behavior and observe replies, statuses, streams, KV/Object Store records, or materialized projections.
- Activation is a first-class layer above substrate; request/reply is only one activation source.
- Browser edge owns session bootstrap, service-worker setup, browser credential mint/revoke, artifact serving, cache/CSP/sandbox policy, and missing browser control-plane behavior.
- Service-worker setup is substrate-owned: the server issues an HttpOnly/Secure/SameSite cookie session, serves the worker script under a controlled scope, and exposes a scoped substrate surface without handing generated content tokens, NATS credentials, subjects, permissions, cookies, or registration authority.
- Control plane and app plane are separate authority domains.

## Next Slice

Task layer next: `activation-schedule-engine`.

Assumption:
- `browser-isolation` triage-three converged on the v1 model: generated artifacts run in `iframe sandbox="allow-scripts"` without `allow-same-origin`; trusted shell/dedicated worker owns leased IPC; gateway and Command Acceptance own mutation.
- Frontend isolation layer is complete after subagent hardening: Vite shell uses a blob-backed generated artifact fixture, exact script-only sandbox, source-window message lease, nonce/revision/capability/expected-revision checks, structured-clone raw-authority denial, and Go-embedded build output.
- Browser isolation proof is complete: Go Browser Edge denies gateway CSRF/origin/fetch-metadata/CORS/stale/revoked cases; service-worker setup denies unsafe scope/header/revision/generated-registration cases; Chrome proves exact service-worker scope and broad-scope denial; embedded NATS request/reply proves canonical `command.acceptance` accepted and rejected statuses at the NATS seam.
- Current embedded NATS auth users are static at server start. Direct browser NATS WebSocket remains deferred until live credential reload and post-connection revocation are proven.
- Service worker is server-owned scoped bootstrap/cache/material facade only. It must not hold NATS credentials, bearer tokens, raw subjects, permission material, or independent mutation authority.
- Live source router is complete: request/reply, subject, KV, Object Store meta-stream, and stream observations normalize through source authority into the durable activation ledger over real embedded NATS.
- Schedule engine is now the next activation foundation task.

RED:
- Write failing durable schedule engine tests before implementation.
- RED must prove that schedule activation cannot be a best-effort timer: it needs durable schedule state, lease/leadership/fencing, deterministic clock input, catch-up, restart recovery, tick dedupe, loop safety, and attributed failures.

GREEN:
- Add a schedule engine that consumes canonical schedule activation shape and emits accepted activation records only after durable lease/fencing and clock-position checks pass.
- Prove duplicate ticks, stale/rewound clock positions, lost/revoked lease, exhausted chain, catch-up after restart, and attributed failures.
- Do not add script execution, materializer implementation, product UI, direct browser NATS WebSocket, or sandboxing in this schedule slice.

VERIFY:
- targeted schedule engine inside-out tests with fake clock and typed schedule failures
- embedded NATS durable store proof for schedule state, lease/fencing, restart catch-up, and accepted schedule activation records
- duplicate, stale cursor/clock, revoked lease, loop-suppressed, and attributed failure cases
- `bun run schema:parity`
- `go test ./...` from `substrate/go`
- `bun run test`
- `bun run test:e2e`
- `bun run typecheck`
- `bun run build`
- `bun run pack:dry`
- `bun run validate:layers`
- `bun run test:layers`
- no-slop scan over browser isolation docs, frontend shell proof, gateway policy, service-worker setup, schemas, SDK validation, and Go validation

Evidence gathered:
- Orchestrated command-acceptance worker patch was applied to the primary checkout after the first generated worktree failed full verification due dependency path placement.
- Targeted command acceptance: `bun test packages/sdk/tests/endgame-contract/command-acceptance.test.ts` -> `9 pass`, `0 fail`, `53 expect() calls`.
- Contract/schema parity: `bun run schema:parity` -> endgame contract tests `17 pass`, `0 fail`; Go contract package `ok`.
- Full tests: `bun run test` -> `48 pass`, `0 fail`, `317 expect() calls`.
- Typecheck: `bun run typecheck` -> SDK plus orchestrator typecheck passed.
- Build: `bun run build` -> SDK ESM, CommonJS, and declarations emitted.
- Layer docs: `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`; `bun run test:layers` -> `Ran 10 tests ... OK`.
- Orchestrator fix: generated worktrees now live as direct siblings of the repo root so relative local file dependencies resolve like the primary checkout.
- Next-slice triage-three: confirmed risks are schema-only Go proof, frontend-local intent drift, raw authority leakage to generated content, and unproven artifact gateway policy.
- Substrate Edge Bootstrap task doc: `docs/matched-abstraction/task/substrate-edge-bootstrap.md`; diagram `https://diashort.apps.quickable.co/d/8e1c7e86`.
- Substrate Edge targeted Browser Edge: `bun test packages/sdk/tests/endgame-contract/substrate-edge-bootstrap.test.ts` -> `4 pass`, `0 fail`, `17 expect() calls`.
- Substrate Edge targeted Go: `go test ./edge` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/edge`.
- Substrate Edge schema parity: `bun run schema:parity` -> endgame contract tests `21 pass`, `0 fail`; Go contract and edge packages `ok`.
- Substrate Edge full tests: `bun run test` -> `52 pass`, `0 fail`, `334 expect() calls`.
- Substrate Edge typecheck: `bun run typecheck` -> SDK plus orchestrator typecheck passed.
- Substrate Edge build/package: `bun run build` -> SDK bundles emitted; `bun run pack:dry` -> `tinkabot-0.1.0.tgz`, 6 files.
- Go substrate matched-abstraction docs: `docs/matched-abstraction/approach/go-substrate.md`, `docs/matched-abstraction/plan/go-substrate.md`, `docs/matched-abstraction/task/go-substrate-core.md`; diagram `https://diashort.apps.quickable.co/d/4a99eb1d`.
- Go substrate Approach seal: embedded NATS ownership, NATS-native HA/scale, NATS auth vocabulary, separated authority envelopes, mediated scripts, generated-content denial, and typed substrate failures are locked unless Approach is explicitly reopened.
- Go substrate Plan refinement: `embedded-nats-adapter` now sits between `go-substrate-core` and `activation-source-router`; Plan diagram `https://diashort.apps.quickable.co/d/5edab343`; traced test ownership requires one owning Task test per declared error and explicit Resolve/Transform/Propagate for consumed errors.
- Go substrate core RED: `go test ./core` from `substrate/go` -> failed with missing `BuildPlan`, topology, lease, status, and process protocol symbols.
- Go substrate core targeted: `go test ./core` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/core`.
- Go substrate core all-Go: `go test ./...` from `substrate/go` -> `ok` for `contract`, `core`, and `edge`.
- Go substrate core schema parity: `bun run schema:parity` -> endgame contract tests `21 pass`, `0 fail`; Go `contract`, `core`, and `edge` packages `ok`.
- Go substrate core full tests: `bun run test` -> `52 pass`, `0 fail`, `334 expect() calls`.
- Go substrate core typecheck/build/package: `bun run typecheck`, `bun run build`, and `bun run pack:dry` passed.
- Embedded NATS Adapter task doc: `docs/matched-abstraction/task/embedded-nats-adapter.md`; diagram `https://diashort.apps.quickable.co/d/9a4270ef`.
- Embedded NATS Adapter RED: `go test ./embednats` from `substrate/go` -> failed with missing adapter symbols after module sums were repaired.
- Embedded NATS Adapter targeted: `go test ./embednats` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/embednats`.
- Embedded NATS Adapter all-Go: `go test ./...` from `substrate/go` -> `ok` for `contract`, `core`, `edge`, and `embednats`.
- Embedded NATS Adapter schema parity: `bun run schema:parity` -> endgame contract tests `21 pass`, `0 fail`; Go packages `contract`, `core`, `edge`, and `embednats` passed.
- Embedded NATS Adapter full tests/typecheck/build/package/layers: `bun run test`, `bun run test:e2e`, `bun run typecheck`, `bun run build`, `bun run pack:dry`, `bun run validate:layers`, and `bun run test:layers` passed.
- Embedded NATS Adapter handoff: `bun run orchestrate:codex -- --dry-run --allow-dirty` -> next topic `activation-source-router`; `git diff --check` and focused placeholder scan were clean.
- Embedded NATS Adapter review hardening: subagents found an unrestricted probe user, missing router-safe client boundary, under-owned auth/WebSocket error branches, and missing stop panic recovery. Fixed with random least-authority probe credentials, `Runtime.Connect`, expanded auth/WebSocket tests, stop panic recovery for drain/shutdown/wait, and post-review full verification.
- Activation Foundation Plan: `docs/matched-abstraction/plan/activation-foundation.md`; first task doc: `docs/matched-abstraction/task/activation-contract-authority.md`; diagram `https://diashort.apps.quickable.co/d/32dc325b`.
- Activation Foundation docs validation: `bun run validate:layers`, `bun run test:layers`, and `git diff --check` passed.
- Activation Foundation orchestrator check: `bun run orchestrate:codex -- --dry-run --allow-dirty` now selects topic `activation-contract-authority`.
- Activation Foundation subagent reinforcement: layer, traced-TDD, and NATS runtime reviewers found contract-policy ownership blur, missing owner-layer fixture matrix, underspecified source principal/lease envelope, broad wildcard aperture risk, Object Store watcher cursor risk, and thin schedule fencing. Docs were hardened to keep policy denials schema-valid and require owner-layer tags plus source/schedule/cursor fields.
- Activation Foundation reinforcement arbiter: final read-only arbiter returned `BLOCKING: no` and required no further patch.
- Activation Contract Authority RED: `bun test packages/sdk/tests/endgame-contract/contract-authority.test.ts` and `go test ./contract -count=1` failed on new activation source fixtures before schema and SDK implementation.
- Activation Contract Authority GREEN: canonical schema, SDK validation, Go validation, fixtures, and parity now cover all activation source kinds plus source principal, source lease, cursor, wildcard aperture, provenance, and owner-layer tags.
- Activation Contract Authority verification: targeted contract tests, command-acceptance tests, substrate-edge tests, `go test ./... -count=1`, `bun run schema:parity`, `bun run test`, `bun run typecheck`, `bun run build`, `bun run pack:dry`, `bun run validate:layers`, `bun run test:layers`, and `git diff --check` passed.
- Activation Contract Authority handoff: `bun run orchestrate:codex -- --dry-run --allow-dirty` now selects topic `activation-ledger-durability`.
- Activation Ledger Durability task doc: `docs/matched-abstraction/task/activation-ledger-durability.md`; diagram `https://diashort.apps.quickable.co/d/32dc325b`.
- Activation Ledger Durability RED: `go test ./core -count=1` from `substrate/go` failed before implementation with missing durable ledger symbols and durable `StaleCursor` path.
- Activation Ledger Durability GREEN: Go core now has `DurableLedger`, `LedgerStore`, `MemoryLedgerStore`, durable activation records, source cursor extraction for all activation source kinds, encoded collision-safe replay cursors, duplicate resolution, loop suppression records, replay/catch-up, unknown replay cursor failure, restart recovery, mandatory source lease binding, source kind binding, and write-conflict mapping. Embedded NATS now has `KVLedgerStore` backed by real JetStream KV; `MemoryLedgerStore` remains only for narrow unit checks.
- Activation Ledger Durability embedded correction: user corrected fake-first testing. Added `go test ./embednats -run 'TestEmbeddedLedger' -count=1` to prove accept, duplicate, restart, replay, stale cursor behavior, and all canonical source kinds over the embedded NATS runtime and JetStream KV.
- Activation Ledger Durability subagent review hardening: layer reviewer passed; tests reviewer blockers were fixed with all-source cursor tests, restart replay/cursor proof, and no-write denial assertions; risk reviewer blockers were fixed with encoded replay cursors, mandatory lease ids, source kind binding, replay collision proof, and missing-lease proof. Tests and risk re-reviews returned `STATUS: passed`.
- Activation Ledger Durability verification: `go test ./core -count=1`, `go test ./embednats -run 'TestEmbeddedLedger' -count=1`, `go test ./embednats -count=1`, `go test ./... -count=1`, `bun run schema:parity`, `bun run test`, `bun run test:e2e`, `bun run typecheck`, `bun run build`, `bun run pack:dry`, `bun run validate:layers`, `bun run test:layers`, and `git diff --check` passed.
- Script NATS CLI Proof Plan: `docs/matched-abstraction/plan/script-nats-cli-proof.md`; diagram `https://diashort.apps.quickable.co/d/ff5f7a64`. Local CLI evidence: `nats --version` -> `v0.3.0`; command surface includes `request`, `publish`, `subscribe`, `kv`, `object`, `stream`, and auth flags.
- Activation Source Authority task doc: `docs/matched-abstraction/task/activation-source-authority.md`; diagram `https://diashort.apps.quickable.co/d/63fd4830`.
- Activation Source Authority RED: `go test ./core -run TestSourceAuthority -count=1` failed before implementation with missing `AuthorizeSource`, `SourceAuthDenied`, and related symbols.
- Activation Source Authority GREEN: Go core now has `AuthorizeSource`, `SourceGrant`, `SourceAuthDenied`, `DeniedNeighbor`, NATS `*`/`>` subject matching, deny-over-allow precedence, source principal/lease/revision checks, request/reply bounded response authority, import/export/exposure preservation, source coordinate normalization for all canonical source kinds, and grant/denial attribution.
- Activation Source Authority CLI proof: `go test ./embednats -run TestSourceAuthorityCLIAllowedAndDeniedSubject -count=1` uses embedded NATS plus real `nats request` with source credentials to prove allowed request/reply and denied-neighbor permission evidence.
- Activation Source Authority verification: `go test ./core -run TestSourceAuthority -count=1`, `go test ./embednats -run TestSourceAuthorityCLIAllowedAndDeniedSubject -count=1`, `go test ./core ./embednats -count=1`, `go test ./... -count=1`, `bun run schema:parity`, `bun run test`, `bun run test:e2e`, `bun run typecheck`, `bun run build`, `bun run pack:dry`, `bun run validate:layers`, `bun run test:layers`, and `git diff --check` passed.
- Endgame service-worker refinement: Approach/Plan docs now require server-owned, cookie-session-backed, scoped service-worker setup as part of substrate/browser edge. Verification: `bun run validate:layers`, `bun run test:layers`, and `git diff --check` passed.
- Browser Isolation triage: final v1 model is opaque sandboxed generated iframe, leased shell/worker message channel, gateway-owned mutation through Command Acceptance, and service-worker bootstrap/cache/material facade only. Arbiter diagram: `https://diashort.apps.quickable.co/d/2a2abd49`.
- Browser Isolation layer docs: `docs/matched-abstraction/approach/browser-isolation.md`, `docs/matched-abstraction/plan/browser-isolation.md`, and `docs/matched-abstraction/task/browser-isolation-proof.md` now define the v1 model and proof gate. Verification: `bun run validate:layers`, `bun run test:layers`, and `git diff --check` passed.
- Frontend Isolation Layer: Vite app and proof shell under `apps/frontend`, Go embed package under `substrate/go/frontend`, and task doc `docs/matched-abstraction/task/frontend-isolation-layer.md`. Verification: `bun run test:frontend`, `bun run --cwd apps/frontend typecheck`, `bun run build:frontend`, `go test ./frontend -count=1`, `agent-browser` smoke, `bun run test`, `bun run typecheck`, `bun run build`, `bun run schema:parity`, `bun run test:e2e`, `bun run pack:dry`, `bun run validate:layers`, `bun run test:layers`, and `git diff --check` passed.
- Frontend Isolation subagent verification: layer reviewer GO; browser/runtime reviewer found blockers in structured-clone raw-authority denial, `expectedRevision` binding, and cookie-proof overclaim; Go/release reviewer found source-archive readiness NO-GO until frontend and Go embed files are tracked/committed or a clean-checkout regeneration proof exists. Runtime blockers were patched, then source distribution was sealed in commit `c3c3649`; `git ls-files --error-unmatch` and `git archive HEAD apps/frontend substrate/go/frontend package.json` prove the frontend workspace and Go embed site are in committed source.
- Browser Isolation Proof: Go Browser Edge now owns gateway mutation policy and service-worker setup policy; embedded NATS proves browser command acceptance over real request/reply; Chrome proves service-worker exact scope and broad-scope denial; agent-browser proves trusted shell command/denial smoke. Verification: `go test ./edge -run 'TestGatewayMutation|TestServiceWorker' -count=1`, `go test ./embednats -run TestBrowserGatewayCommandAcceptanceOverRealNATS -count=1`, `bun run test:frontend`, agent-browser smoke, `go test ./... -count=1`, `bun run test`, `bun run typecheck`, `bun run build`, `bun run schema:parity`, `bun run test:e2e`, `bun run pack:dry`, `bun run validate:layers`, `bun run test:layers`, and `git diff --check` passed.
- Activation Router Live Sources task doc: `docs/matched-abstraction/task/activation-router-live-sources.md`; diagram `https://diashort.apps.quickable.co/d/0ab25edc`.
- Activation Router Live Sources RED: `go test ./embednats -run 'TestSourceRouter' -count=1` failed before implementation with missing `HeaderRequestID`, `HeaderMessageID`, `NewSourceRouter`, `RequestReplyListenFailed`, `SubjectSubscribeFailed`, `KVWatchFailed`, `ObjectWatchFailed`, `SourceRouter`, `RouterResult`, and `Route`.
- Activation Router Live Sources GREEN: Go embedded-NATS now has `SourceRouter`, `Route`, `RouterResult`, router-owned typed failures, live request/reply, subject, KV, Object Store meta-stream, and stream router paths, explicit source identity headers, Object Store meta-sequence preservation, source-authority-before-ledger acceptance, and request/reply proof through real `nats` CLI.
- Activation Router Live Sources verification: `go test ./embednats -run 'TestSourceRouter' -count=1`, `go test ./embednats -count=1`, `go test ./... -count=1` from `substrate/go`, `bun run schema:parity`, `bun run test`, `bun run typecheck`, `bun run test:e2e`, `bun run build`, `bun run pack:dry`, `bun run validate:layers`, `bun run test:layers`, and `git diff --check` passed.

## Current Verification Commands

- `bun test` or `bun run test` -> SDK tests.
- `bun run test:e2e` -> SDK distribution BDD.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit`.
- `bun run build` -> builds frontend into `substrate/go/frontend/site` and SDK into `packages/sdk/dist`.
- `bun run pack:dry` -> dry package check.
- `bun run orchestrate:codex -- --dry-run --allow-dirty` -> smoke-test the Codex endgame orchestration plan without launching agents.
- `bun run validate:layers` -> matched-abstraction docs.
- `bun run test:layers` -> layer validator unit tests.

## Pinned Decisions

- NATS auth vocabulary is authoritative: `permissions.publish`, `permissions.subscribe`, `allow`, `deny`, `allow_responses`.
- Metadata uses `desc`, not `meaning`.
- Model both access and exposure: inside-out imports/publish and outside-in activation/consumption.
- Subjects must be concrete values or concrete wildcard patterns. No placeholder subject strings.
- Deny wins over allow. `allow_responses` is bounded to invocation/reply.
- Canonical process IPC is framed stdio RPC; fd-specific helpers are adapters only.
- A first slice may be small, but must be complete at its boundary with denial/failure paths.
- NATS auth is the compiled enforcement shape; identity, ownership, session, revision, and capability provenance must survive into it.
- Browser and script credentials are scoped leases, not durable ambient credentials.
- Schema validates shape; capability policy authorizes effects.
- Managed auth compilation denies raw/advanced imports and non-request-reply exposure by default, requires `allow_responses.expiresMs` when response authority is present, distinguishes revoked from expired leases, and requires exported subjects to match declared exposure subjects.
- Command acceptance requires command ids, claims statuses atomically before activation handoff is returned, resolves duplicate command ids without second activation, binds command session/capability context to the active lease, rejects stale revisions, exhausted chain budgets, and revoked/expired capability contexts, and emits `activation.intent` with `source.kind = "command_acceptance"` only for accepted first-seen commands.
- Substrate edge bootstrap stays pure/fakeable: Go derives scoped worker credential descriptors and artifact gateway policy from canonical contracts; Browser Edge splits worker-only credentials from content-safe context and emits only canonical `browser.command_intent` outward.
- Browser service-worker setup belongs to the substrate/browser edge, not generated content. It is cookie-session-backed, scope-isolated, revocable, CSRF/origin checked, and token-free for generated content.
- Browser generated content runs as opaque sandboxed receiver code for v1. The shell must bind IPC by source window or port, nonce, frame lease, schema revision, artifact revision, and capability context; `event.origin` alone is not authority.
- Unsafe same-origin `allow-scripts` plus `allow-same-origin` is denied for untrusted generated content.
- Direct browser NATS WebSocket is deferred until live credential reload, post-connection revocation, denied-neighbor, stale-access, and confidentiality proof exist.
- Go substrate Approach is sealed for `go-substrate-core`; downstream Plan/Task work may refine decomposition and verification, but cannot redefine embedded NATS ownership, HA/scale authority, auth vocabulary, authority envelopes, mediated scripts, generated-content denial, or typed substrate failures.
- Go substrate embeds and manages NATS by default; HA/scale posture uses NATS-provided clustering, JetStream replica/quorum, route/gateway/leaf, WebSocket, queue/consumer, and observability semantics rather than bespoke substrate replication or routing.
- Go substrate core must exist before embedded-NATS adapter, activation, script, or materializer implementation consumes substrate behavior; TypeScript runtime work is regression evidence, not substrate authority.
- Go substrate core is complete as a pure/fakeable contract package under `substrate/go/core`; embedded-NATS adapter consumes it rather than redefining substrate contracts.
- Embedded NATS adapter must sit between Go substrate core and activation foundation so real NATS lifecycle, JetStream, auth load path, WebSocket posture, topology probes, and drain/shutdown semantics are proven before reactive triggers depend on them.
- Embedded NATS adapter is complete under `substrate/go/embednats`; it proves a live single-node embedded server, JetStream `AccountInfo` readiness, NATS auth user loading, least-authority internal probe credential, router-safe `Runtime.Connect`, WebSocket random URL posture, topology probe failure, drain wait, shutdown timeout, stop panic recovery, and adapter-owned error mapping.
- Activation foundation must sit after embedded-NATS adapter and before script-materializer-loop so reactive triggers do not get invented inside script execution.
- Activation contract authority comes before live source routing. The live router consumes canonical source shape, source principal, source lease, cursor, wildcard, provenance, chain, dedupe, and parity fixtures.
- Activation contract authority and durable ledger behavior are complete. Source-scoped authority is now the next activation foundation task.
- Activation foundation is one program with task-owned proofs: contract authority, ledger durability, source authority, live router, schedule engine, and release proof.
- Activation ledger durability stays below source authority and live routing: it records accepted attempts, source cursor state, duplicate/loop/replay outcomes, lease binding, and durable failures, but it does not decide whether a source principal may observe a subject, bucket, object, stream, or schedule. Durable proofs should use embedded NATS/JetStream where available; mocks/fakes are only for narrow branch forcing.
- Activation source authority is complete below live routing: it authorizes source observation with NATS-shaped permissions/imports/exports/exposure, source lease lifecycle/revision checks, bounded request/reply responses, denied-neighbor checks, and typed attribution. Ordinary `subject` sources currently reject `>` apertures as overreach; bounded `*` aperture is allowed.
- Live source router is complete below schedule activation: it turns real embedded-NATS request/reply, subject, KV, Object Store meta-stream, and stream observations into source-authorized durable activation records. Object Store routing uses the meta stream instead of `ObjectStore.Watch()` so the source position preserves JetStream sequence metadata.
- Script-side outside-in proof is driven by real `nats` CLI commands against embedded NATS. CLI proves caller/platform behavior; it does not give default scripts raw NATS authority.
- Release gates must include allowed, denied-neighbor, malformed, duplicate, stale-revision, revoked-credential, and attributed-failure cases over NATS-mediated behavior.

## Milestone Index

Historical details live in matched-abstraction docs and git history. Do not expand this handoff with completed evidence unless it changes current work.

- Baseline skill setup: `docs/matched-abstraction/task/baseline-skill-setup.md`.
- NATS runtime design: `docs/matched-abstraction/{approach,plan}/nats-script-runtime.md`.
- Traced TDD plan: `docs/matched-abstraction/plan/nats-script-runtime-traced-tdd.md`.
- Runtime substrate and record store proof: `docs/matched-abstraction/task/nats-script-runtime-substrate-record-store.md`.
- Distribution BDD proof: `docs/matched-abstraction/task/nats-script-runtime-distribution-bdd.md`.
- Metadata and permissions proof: `docs/matched-abstraction/task/nats-script-runtime-metadata-permissions.md`.
- Activation contract proof: `docs/matched-abstraction/task/nats-script-runtime-activation-contract.md`.
- Request/reply activation adapter proof: `docs/matched-abstraction/task/nats-script-runtime-request-reply-activation-adapter.md`.
- Browser frontend mediator proof: `docs/matched-abstraction/task/browser-frontend-dedicated-worker.md`.
- Browser isolation approach: `docs/matched-abstraction/approach/browser-isolation.md`.
- Browser isolation plan: `docs/matched-abstraction/plan/browser-isolation.md`.
- Browser isolation proof task: `docs/matched-abstraction/task/browser-isolation-proof.md`.
- Frontend isolation layer task: `docs/matched-abstraction/task/frontend-isolation-layer.md`.
- Platform reset: `docs/matched-abstraction/{approach,plan,task}/platform-structure*.md`.
- Code structure reset: `docs/matched-abstraction/{approach,plan}/code-structure.md` and `docs/matched-abstraction/task/code-structure-reorganization.md`.
- Endgame app approach: `docs/matched-abstraction/approach/endgame-app.md`.
- Endgame app plan: `docs/matched-abstraction/plan/endgame-app.md`.
- Go substrate approach: `docs/matched-abstraction/approach/go-substrate.md`.
- Go substrate plan: `docs/matched-abstraction/plan/go-substrate.md`.
- Activation foundation plan: `docs/matched-abstraction/plan/activation-foundation.md`.
- Endgame contract authority task: `docs/matched-abstraction/task/endgame-contract-authority.md`.
- Managed auth subjects task: `docs/matched-abstraction/task/managed-auth-subjects.md`.
- Command acceptance task: `docs/matched-abstraction/task/command-acceptance.md`.
- Substrate edge bootstrap task: `docs/matched-abstraction/task/substrate-edge-bootstrap.md`.
- Go substrate core task: `docs/matched-abstraction/task/go-substrate-core.md`.
- Embedded NATS adapter task: `docs/matched-abstraction/task/embedded-nats-adapter.md`.
- Activation contract authority task: `docs/matched-abstraction/task/activation-contract-authority.md`.
- Activation ledger durability task: `docs/matched-abstraction/task/activation-ledger-durability.md`.
- Activation source authority task: `docs/matched-abstraction/task/activation-source-authority.md`.
- Script NATS CLI proof plan: `docs/matched-abstraction/plan/script-nats-cli-proof.md`.
- Codex endgame orchestration plan: `docs/matched-abstraction/plan/codex-endgame-orchestration.md`.
- Codex endgame orchestrator task: `docs/matched-abstraction/task/codex-endgame-orchestrator.md`.

## Recent Git

- `99cc3c1 chore: establish tinkabot workspace baseline`.
- `42d44fe chore: record git baseline`.
- `5c30a1f chore: add terse coding skill`.

## Cleanup Note

This file was reduced from a completed-evidence log to a current handoff. Completed details belong in layer docs, tests, and git commits.
