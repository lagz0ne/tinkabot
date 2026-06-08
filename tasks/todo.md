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
3. NEXT: `command-acceptance`: durable intent acceptance, idempotency, stale-revision denial, status materialization, activation handoff.
4. `substrate-edge-bootstrap`: Go substrate boundary plus Browser Edge credential/artifact bootstrap over shared contracts.
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

Task layer next: `command-acceptance`.

RED:
- Use `triage-three` to pressure-test command acceptance before writing code.
- Write failing tests for schema-valid browser command intent acceptance, duplicate command idempotency, stale revision denial, revoked/expired capability denial, raw-authority rejection, status materialization, and activation handoff shape.

GREEN:
- Add the smallest command acceptance contract consumer that validates command intent, checks capability/revision context, records a durable acceptance status shape, and emits an activation handoff packet without executing scripts.
- Keep browser intent schema validity separate from backend acceptance authority.

VERIFY:
- `bun run schema:parity`
- command-acceptance targeted tests once created
- `bun run test`
- `bun run typecheck`
- `bun run validate:layers`
- `bun run test:layers`
- no-slop scan over command-acceptance docs, fixtures, and code

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
- Codex endgame orchestration plan: `docs/matched-abstraction/plan/codex-endgame-orchestration.md`.
- Codex endgame orchestrator task: `docs/matched-abstraction/task/codex-endgame-orchestrator.md`.

## Recent Git

- `99cc3c1 chore: establish tinkabot workspace baseline`.
- `42d44fe chore: record git baseline`.
- `5c30a1f chore: add terse coding skill`.

## Cleanup Note

This file was reduced from a completed-evidence log to a current handoff. Completed details belong in layer docs, tests, and git commits.
