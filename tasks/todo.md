# Tinkabot Handoff

## Current State

- Repo: `lagz0ne/tinkabot`, private, branch `main`.
- Remote: `origin git@github.com:lagz0ne/tinkabot.git`.
- Latest pushed commit: `5c30a1f chore: add terse coding skill`.
- Worktree baseline before this cleanup: clean against `origin/main`.
- Root role: orchestration only.
- Current implementation lives in `packages/sdk`.
- Future lanes:
  - `schemas`: canonical JSON Schema and codegen authority.
  - `substrate/go`: Go NATS/auth/process/Docker-facing substrate.
  - `apps/frontend`: Vite trusted shell.

## Active Goal

Progress Tinkabot through the Endgame Plan by completing verified milestones.

## Milestone Workflow

1. DONE: `endgame-contract-authority`: neutral schemas, fixtures, TS/Zod target, Go validation target, and parity command.
2. DONE: `managed-auth-subjects`: identity/capability provenance, subject taxonomy, NATS auth compilation fixtures, lease/revocation/expiration proof, advanced capability denial, bounded responses, and export/exposure pairing.
3. DONE: `command-acceptance`: durable intent acceptance, atomic idempotency, required command ids, capability context binding, stale-revision denial, capability lease denial, status materialization, activation handoff.
4. NEXT: `substrate-edge-bootstrap`: Go substrate boundary plus Browser Edge credential/artifact bootstrap over shared contracts.
5. `script-materializer-loop`: mediated script execution, accepted effects, materialized projections/artifacts, cleanup.
6. `release-spine`: centralized ops evidence manifest with outside-in real NATS proof and inside-out ownership proof.

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
- The product loop is source/artifact -> materialized projection -> browser intent -> durable backend acceptance -> activation -> script execution -> attributed event/projection update.
- Go owns substrate authority: NATS infra, auth, process lifecycle, Docker/sandboxing direction, connection policy, activation ledger, artifact gateway, execution attribution.
- Vite owns the trusted browser shell. Generated browser content remains a receiver and intent emitter.
- Schema/SDK owns shared contract shape. JSON Schema is the first neutral source; generated or checked Zod, TS types, Go validators/types, and fixtures follow it.
- Existing Bun/TypeScript runtime and `@lagz0ne/nats-embedded` work is regression evidence and SDK material, not current substrate authority.
- Default scripts stay NATS-agnostic process contracts. Runtime facade mediates NATS publish/progress/import requests.
- Activation is a first-class layer above substrate; request/reply is only one activation source.
- Browser edge owns session bootstrap, browser credential mint/revoke, artifact serving, cache/CSP/sandbox policy, and missing browser control-plane behavior.
- Control plane and app plane are separate authority domains.

## Next Slice

Task layer next: `substrate-edge-bootstrap`.

Assumption:
- Go substrate and Browser Edge consume the existing contract/auth/command packet instead of inventing new identity, credential, artifact, or status shapes.
- This slice bootstraps substrate/browser-edge boundaries only. It must not implement full script execution, materialization, schedule activation, or frontend rendering.

RED:
- Use `triage-three` to pressure-test Go substrate and Browser Edge bootstrap before writing code.
- Write failing tests for Go contract consumption, lease denial before credential descriptor creation, Browser Edge credential/content split, canonical `browser.command_intent` bridging, artifact gateway manifest policy, revocation denial, and no raw NATS credential exposure to generated content.

GREEN:
- Add a pure/fakeable Go substrate-edge boundary that consumes shared contracts, models scoped worker credential descriptors, denies revoked/expired/stale leases, and validates artifact gateway policy shape.
- Add a trusted Browser Edge bootstrap boundary that consumes sanitized bootstrap context, withholds raw authority from generated content, and emits canonical `browser.command_intent`.
- Keep Go substrate authority, Browser Edge credential lifecycle, and generated-content access separate.

VERIFY:
- `bun run schema:parity`
- substrate/browser-edge targeted tests once created
- `bun run test`
- `bun run typecheck`
- `bun run validate:layers`
- `bun run test:layers`
- no-slop scan over substrate/browser-edge docs, fixtures, and code

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

## Current Verification Commands

- `bun test` or `bun run test` -> SDK tests.
- `bun run test:e2e` -> SDK distribution BDD.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit`.
- `bun run build` -> builds `packages/sdk/dist`.
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
- Platform reset: `docs/matched-abstraction/{approach,plan,task}/platform-structure*.md`.
- Code structure reset: `docs/matched-abstraction/{approach,plan}/code-structure.md` and `docs/matched-abstraction/task/code-structure-reorganization.md`.
- Endgame app approach: `docs/matched-abstraction/approach/endgame-app.md`.
- Endgame app plan: `docs/matched-abstraction/plan/endgame-app.md`.
- Endgame contract authority task: `docs/matched-abstraction/task/endgame-contract-authority.md`.
- Managed auth subjects task: `docs/matched-abstraction/task/managed-auth-subjects.md`.
- Command acceptance task: `docs/matched-abstraction/task/command-acceptance.md`.
- Substrate edge bootstrap task: `docs/matched-abstraction/task/substrate-edge-bootstrap.md`.
- Codex endgame orchestration plan: `docs/matched-abstraction/plan/codex-endgame-orchestration.md`.
- Codex endgame orchestrator task: `docs/matched-abstraction/task/codex-endgame-orchestrator.md`.

## Recent Git

- `99cc3c1 chore: establish tinkabot workspace baseline`.
- `42d44fe chore: record git baseline`.
- `5c30a1f chore: add terse coding skill`.

## Cleanup Note

This file was reduced from a completed-evidence log to a current handoff. Completed details belong in layer docs, tests, and git commits.
