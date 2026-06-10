---
layer: task
topic: command-acceptance
references:
  - ../approach/endgame-app.md
  - ../plan/endgame-app.md
  - ./endgame-contract-authority.md
  - ./managed-auth-subjects.md
---

# Command Acceptance Task

## Objective

Implement the first backend command-acceptance boundary over the endgame contract packet: schema-shaped browser intent enters, backend authority decides acceptance or denial, durable status is materialized, and a valid activation handoff packet is emitted only for first-seen accepted commands.

## Scope

This task owns:

- browser command intent consumption through the existing contract parser.
- atomic command idempotency by required command id.
- command session/capability context binding.
- stale artifact revision denial.
- revoked and expired capability denial as materialized statuses.
- unknown command denial.
- command acceptance status materialization through a store boundary.
- command-acceptance activation handoff shape.
- typed command-acceptance runtime errors for materialization and critical failures.
- parity updates for the activation source contract needed to represent command-acceptance handoff provenance.

## Non-Goals

- No script execution.
- No live NATS transport.
- No activation worker, ledger, retry, cursor, or dedupe implementation beyond the handoff packet.
- No Browser Edge session bootstrap or credential minting.
- No materialized projection update after activation.
- No Go substrate storage implementation.
- No raw NATS authority for generated browser content.

## Acceptance Contract

- Schema-valid browser command intent can be accepted when capability is active, command route exists, and expected artifact revision matches the current artifact revision.
- `commandId` is required by the shared contract.
- Duplicate command id returns the existing materialized status and emits no second activation handoff, including concurrent duplicate attempts.
- Command context session and capability ids must match the active capability lease before status materialization can accept.
- Stale revision is denied with `StaleRevision` from `CommandAcceptance`.
- Revoked and expired capability contexts are denied with `RevokedLease` or `ExpiredLease` from `ManagedAuth`.
- Unknown commands are denied with `AcceptanceDenied` from `CommandAcceptance`.
- Exhausted command chain hop budget is denied with `ActivationLoopSuppressed` before activation handoff.
- Raw-authority payloads fail at `ContractAuthority` before status materialization.
- Accepted and rejected statuses parse as `command.acceptance` contracts.
- Accepted first-seen commands emit an `activation.intent` contract with `source.kind = "command_acceptance"`.
- Status store claim/materialization failure throws `StatusMaterializationFailed` from `CommandAcceptance`.

## RED Artifact

- Command: `bun test packages/sdk/tests/endgame-contract/command-acceptance.test.ts`
- Initial environment prerequisite: failed before RED assertion because `node_modules` was absent and `zod` could not be resolved.
- Setup command: `bun install --frozen-lockfile --offline` -> installed cached packages but failed to link missing local file deps `@lagz0ne/nats-embedded` and `@lagz0ne/nats-embedded-linux-x64`.
- RED command rerun: `bun test packages/sdk/tests/endgame-contract/command-acceptance.test.ts`
- Expected failure: `Export named 'createCommandAcceptance' not found in module ... packages/sdk/src/endgame-contract/index.ts`.
- Missing contract proven: command-acceptance consumer, status store, idempotency, denial, and activation handoff boundary were absent.

## Execution Notes

Added `packages/sdk/src/endgame-contract/command-acceptance.ts` as the milestone consumer. It accepts unknown input, parses it through `parseContract`, reads an existing status by command id, materializes a new accepted or rejected status through a store boundary, and builds an activation handoff only after route, capability, and revision checks pass.

Extended `activation.intent.source` from request/reply-only to a discriminated source union including `command_acceptance`. This keeps browser command provenance explicit without claiming script execution or live activation.

Endgame contract tests now import the endgame-contract boundary directly instead of the SDK aggregate index, so contract parity does not depend on unrelated NATS runtime substrate imports.

## Verification Evidence

RED:

- `bun test packages/sdk/tests/endgame-contract/command-acceptance.test.ts` -> `0 pass`, `1 fail`, missing `createCommandAcceptance` export after dependency setup.

GREEN:

- `bun test packages/sdk/tests/endgame-contract/command-acceptance.test.ts` -> `9 pass`, `0 fail`, `53 expect() calls`.
- `bun run schema:parity` -> endgame contract tests `17 pass`, `0 fail`, `135 expect() calls`; Go contract package `ok`.
- `bunx @typescript/native-preview --noEmit --ignoreConfig --target ES2022 --module ESNext --moduleResolution Bundler --types bun-types --skipLibCheck --strict src/endgame-contract/index.ts src/endgame-contract/command-acceptance.ts tests/endgame-contract/command-acceptance.test.ts tests/endgame-contract/contract-authority.test.ts tests/endgame-contract/managed-auth-subjects.test.ts` from `packages/sdk` -> exit `0`.
- `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`.
- `bun run test:layers` -> `Ran 10 tests ... OK`.

Named negative-case evidence (re-executed 2026-06-10 during the release-spine evidence audit):

- `bun test packages/sdk/tests/endgame-contract/command-acceptance.test.ts -t T-CMD-IDEMPOTENCY` -> `2 pass`, `0 fail`: duplicate commands resolve without a second activation handoff, atomically under concurrency.
- `bun test packages/sdk/tests/endgame-contract/command-acceptance.test.ts -t T-CMD-DENY` -> `1 pass`, `0 fail`: stale revision and unknown commands are denied as command-acceptance statuses.
- `bun test packages/sdk/tests/endgame-contract/command-acceptance.test.ts -t T-CMD-CONTRACT` -> `1 pass`, `0 fail`: raw-authority intent is rejected before status materialization.

Full-suite proof:

- Initial orchestrator verification failed because the generated integration worktree was nested under `.codex-runs/`, which broke the repo's relative local dependency path to `../../../nats-embedded`.
- The worker patch was applied to the primary checkout, where dependencies resolve correctly, and the orchestrator was fixed to place generated worktrees as direct siblings of the repo root.
- `bun run typecheck` -> SDK typecheck plus orchestrator typecheck passed.
- `bun run test` -> `48 pass`, `0 fail`, `317 expect() calls`.
- `bun run build` -> SDK ESM, CommonJS, and declarations emitted.

Review passes:

- No-slop pass: command acceptance stays in `endgame-contract`, no script execution, NATS transport, Browser Edge, materializer, or Go substrate storage was added.
- No-slop scan over slop markers in command-acceptance code, tests, schema, task doc, and handoff -> only intentional milestone/status vocabulary.
- Simplify pass: one direct consumer module, one store interface, one memory proof store, and one activation source extension; no new framework or generic ledger abstraction.
- Review pass: accepted status is claimed atomically before activation handoff is returned; duplicates reuse existing status and return no activation packet; command context must match the active capability.

## Wrap-Up Announcement

Shipped: schema-shaped browser commands are accepted or denied at the contract-consumer boundary with durable status semantics, atomic idempotency, stale revision denial, capability context and lease denial, raw-authority rejection, and activation handoff materialized as verified contract shapes. Evidence recorded above.
