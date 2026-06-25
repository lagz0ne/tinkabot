---
layer: task
topic: endgame-contract-authority
references:
  - ../approach/endgame-app.md
  - ../plan/endgame-app.md
---

# Endgame Contract Authority Task

## Objective

Establish the smallest neutral endgame contract packet that every later lane must consume before substrate, browser edge, frontend, activation, script runtime, materializer, or release-proof work expands.

This task proves contract authority, not app behavior.

## Scope

This task owns:

- provenance envelope.
- principal, session, lease, revision, and capability references.
- NATS-auth-shaped permissions, imports, exports, exposure, and deny shape.
- subject taxonomy fixtures for allowed and denied-neighbor surfaces.
- browser command intent.
- command acceptance status.
- activation intent.
- artifact manifest.
- materialized projection.
- attributed event and error envelope.
- shared positive and negative fixtures.
- cross-target accept/reject parity for canonical schema, TypeScript/Zod target, Go validation target, and fixtures.

## Non-Goals

- No Go substrate implementation.
- No Vite shell or browser edge implementation.
- No live NATS WebSocket proof.
- No artifact gateway implementation.
- No script runtime migration.
- No sandbox enforcement.
- No activation worker, command ledger, materializer, or release packaging implementation.
- No lane-local DTOs that bypass the neutral contract packet.
- No provider-specific auth backend decision.

## Acceptance Contract

- A canonical contract packet exists as neutral authority for the scoped shapes.
- TypeScript/Zod and Go validation targets agree with the canonical contract on accepted and rejected fixtures.
- Fixtures cover allowed, denied-neighbor, malformed, duplicate, stale-revision, revoked-lease, no-raw-authority, and attributed-failure cases where applicable to this contract packet.
- Schema validity stays separate from capability authority.
- Browser/generated-content-facing shapes expose no raw NATS subjects, credentials, permission material, or publish API.
- Script-facing base shapes expose no raw NATS authority by default.
- Subject fixtures use concrete authoritative prefixes and reject placeholder subjects or broad wildcard escape.
- Effect-shaped fixtures carry identity, session, revision, capability, schema, and chain context.
- Contract tests fail before the packet exists and pass only when all targets consume the same authority.

## RED Artifact

Add failing parity and contract tests that prove the endgame contract packet is not authoritative yet.

Expected RED failure:

- canonical contract source is missing.
- TypeScript/Zod target is missing or drifting from canonical schema.
- Go validation target is missing or drifting from canonical schema.
- shared fixtures are missing.
- accept/reject parity is missing for denied-neighbor, malformed, stale, revoked, or no-raw-authority cases.

## Execution Notes

Keep the slice contract-only. Prefer the smallest set of shapes that lets later lanes share identity, authority, revision, denial, and attribution semantics.

Do not encode substrate behavior, browser behavior, gateway behavior, script execution behavior, or release orchestration here. Those lanes consume this packet later.

Do not let generated validators or language-local types become authority. They are generated from, or checked against, the neutral source.

## Verification Evidence

RED:

- Command: `bun test packages/sdk/tests/endgame-contract/contract-authority.test.ts`
- Expected failure: `Export named 'parseContract' not found in module ... packages/sdk/src/index.ts`.
- Owning layer: Contract authority.
- Missing contract proven: TypeScript/Zod contract target is absent.

- Command: `go test ./substrate/go/...`
- Expected failure: `directory prefix substrate/go does not contain main module or its selected dependencies`.
- Owning layer: Contract authority.
- Missing contract proven: Go validation target is absent.

GREEN:

- Contract parity command: `bun run schema:parity` -> `2 pass`, `0 fail`; Go contract package `ok`.
- SDK contract tests: `bun run test:contracts` -> `2 pass`, `0 fail`.
- TypeScript check: `bun run typecheck` -> `bunx @typescript/native-preview --noEmit`.
- Go validation/parity check: `bun run test:go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/contract`.
- Fixture parity result: `schemas/endgame/v1/parity.cases.json` drives TypeScript/Zod and Go schema validation over the same fixtures.
- Denied-neighbor fixture result: `fixtures/valid/attributed-denial.json` is schema-valid and carries `PermissionDeniedByDenyRule`; `fixtures/valid/auth-policy.json` carries deny-neighbor permission shape.
- Malformed fixture result: `fixtures/invalid/subject-placeholder.json`, `fixtures/invalid/subject-wildcard.json`, and `fixtures/invalid/missing-provenance.json` are rejected by both targets.
- Stale/revoked fixture result: `fixtures/valid/command-stale-revision.json` and `fixtures/valid/revoked-capability.json` are schema-valid policy-denial fixtures.
- No-raw-authority fixture result: `fixtures/invalid/browser-command-raw-nats.json` is rejected by both targets.



Review passes:

- No-slop pass: focused scan over endgame docs and contract fixtures contains only intentional boundary terms.
- Simplify pass: kept the first slice to one schema packet, one Zod sidecar, and one Go JSON Schema registry; no generator framework or substrate implementation.
- Review pass: confirmed script/browser raw NATS authority remains denied by fixture boundary and no Go/Vite lane code was added.

## Wrap-Up Announcement

The `endgame-contract-authority` milestone establishes the neutral contract packet for provenance, identity, authority, revision, command, activation, artifact, projection, and attributed error semantics. Later lanes can consume one shared contract source instead of inventing lane-local shapes.
