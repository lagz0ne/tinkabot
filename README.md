# Tinkabot

Tinkabot v1 is a local platform proof for running generated work through a trusted substrate instead of handing raw authority to generated code. The current checkout proves the platform contracts, Go-owned embedded NATS substrate, browser isolation boundary, activation sources, script materializer loop, and release evidence gate.

This is not a published package or finished product UI. It is a verified source checkout with a release evidence manifest at `release/v1.json`.

User manual: `docs/manual/v1.md` — how to operate v1 (scripts, triggers, materials, authority, browser path), with every command quoted from executed proofs.

## Current Runnable Surface

The v1 target is a single Go-owned platform entry surface: embedded NATS, embedded frontend shell, activation, and the script materializer loop operated through the `nats` CLI. This checkout does not yet contain a `package main` entrypoint or installable binary, so the runnable surface today is the verified source-level evidence commands below.

Flow diagram: https://diashort.apps.quickable.co/d/bb63b165

## What v1 Proves

- Shared contracts: JSON Schema is the neutral source for the base v1 contracts, with TypeScript/Zod and Go validation parity.
- Substrate: Go owns embedded NATS lifecycle, JetStream-backed stores, auth rendering, credentials, process boundaries, activation ledger, schedule store, and script materialization.
- Browser edge: the trusted Vite shell is embedded into Go build output; generated browser content stays in an opaque sandboxed iframe and can only emit leased intents.
- Activation: request/reply, subject, KV, Object Store, stream, command acceptance, and schedule sources normalize into durable activation records.
- Script runtime: scripts run through a mediated framed process contract; accepted effects materialize projections and artifacts through substrate-owned stores.
- Release gate: `bun run release:evidence` validates the sixteen v1 milestones over the eleven release-spine steps and rejects incomplete, stale, unresolved, or overclaimed evidence.

## Deferred Scope

The v1 manifest names these as not proven:

- direct browser NATS WebSocket
- Docker sandboxing
- product UI rendering
- live auth reload
- wall-clock scheduler loops
- broad script CRUD UI
- live multi-node HA/scale
- package publication

Do not treat dry package output, frontend shell proof, or HA/scale contract shape as those deferred features.

## Prerequisites

- Bun
- Go
- `nats` CLI for the real-NATS CLI proof tests
- `agent-browser` for browser smoke checks when working on frontend behavior

The repository already records local evidence with `nats` CLI v0.3.0. Permission-denial CLI checks parse command output because that CLI can report permission errors while returning success.

## Verify v1

Install dependencies, then run the release evidence gate:

```bash
bun install
bun run release:evidence
```

Expected shape:

```text
release evidence check passed: 16 milestones over 11 spine steps
```

For a fuller local closeout, run:

```bash
bun run schema:parity
bun run test
bun run test:e2e
bun run typecheck
bun run build
bun run pack:dry
bun run validate:layers
bun run test:layers
```

For Go-only substrate checks:

```bash
cd substrate/go
go test ./...
```

## Useful Focused Proofs

Run these from `substrate/go`:

```bash
go test ./embednats -run TestActivationReleaseProof -count=1
go test ./embednats -run 'TestScriptMaterializerLoopFromNATSCLI|TestLocalScriptRunner|TestKVScriptStoreRejectsUnknownRecordField|TestScriptLoopDurableRunClaimRejectsAcceptedReplay|TestScriptLoopAttributesStatusWriteFailure' -count=1 -v
go test ./embednats -run TestBrowserGatewayCommandAcceptanceOverRealNATS -count=1
go test ./edge -run 'TestGatewayMutation|TestServiceWorker' -count=1
```

These prove the main outside-in surfaces over real embedded NATS or browser-edge policy. They are proof commands, not a user-facing product CLI.

## Project Map

- `release/v1.json`: machine-checkable v1 evidence manifest.
- `scripts/release-evidence.ts`: release evidence checker behind `bun run release:evidence`.
- `schemas/base/v1`: canonical v1 JSON Schema contracts and fixtures.
- `packages/sdk`: TypeScript SDK and contract validation lane.
- `substrate/go`: Go substrate packages for contracts, core behavior, embedded NATS, browser edge, and frontend embed.
- `apps/frontend`: trusted Vite shell used by the browser isolation proof.
- `docs/matched-abstraction`: Approach, Plan, and Task evidence docs.
- `tasks/todo.md`: current handoff and next-session state.

## Authority Docs

The current release authority is `release/v1.json`, backed by these matched-abstraction docs:

- Endgame app: `docs/matched-abstraction/approach/endgame-app.md`, `docs/matched-abstraction/plan/endgame-app.md`
- Go substrate: `docs/matched-abstraction/approach/go-substrate.md`, `docs/matched-abstraction/plan/go-substrate.md`
- Browser isolation: `docs/matched-abstraction/approach/browser-isolation.md`, `docs/matched-abstraction/plan/browser-isolation.md`
- Activation foundation: `docs/matched-abstraction/plan/activation-foundation.md`
- Script CLI proof: `docs/matched-abstraction/plan/script-nats-cli-proof.md`

Historical docs may still use "endgame" wording because the closeout renamed live surfaces to base/v1 while preserving executed evidence names.
