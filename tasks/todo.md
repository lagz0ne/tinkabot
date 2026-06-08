# Tinkabot Handoff

## Current Goal

Execute the platform-structure reset through code layout: Go substrate owns NATS/auth/process/Docker-facing authority, Vite owns frontend shell delivery, SDK/schema owns shared contract shape, and root remains workspace orchestration.

## RED-GREEN-TDD Plan For Code Structure Reorganization

- [x] RED: Prove the existing root-owned `src`, TypeScript tests, build config, package exports, and `dist` conflicted with platform ownership.
- [x] RED: Preserve baseline behavior with `bun test`, `bun run typecheck`, `bun run build`, and layer validation before moving files.
- [x] GREEN: Move the existing TypeScript package into `packages/sdk`.
- [x] GREEN: Make root `package.json` workspace orchestration with delegated SDK commands.
- [x] GREEN: Add ownership notes for `apps/frontend`, `substrate/go`, `schemas`, and `packages/sdk`.
- [x] VERIFY: Run SDK tests, e2e, typecheck, build, dry pack, structure checks, layer validation, and layer unit tests.
- [x] REFACTOR: Fix false-green Bun `--cwd` script shape and remove generated install backup.

## Code Structure Reorganization Evidence

- Current root role: orchestration, docs, task handoff, layer validation, workspace scripts, and lockfile.
- Current SDK package role: `packages/sdk` owns existing TypeScript source, TypeScript tests, package exports, build config, dependencies, and `dist`.
- Current future lanes:
  - `apps/frontend` for the Vite trusted shell.
  - `substrate/go` for Go substrate authority.
  - `schemas` for canonical JSON Schema and generated validation/type artifacts.
- Added matched-abstraction docs:
  - `docs/matched-abstraction/approach/code-structure.md`
  - `docs/matched-abstraction/plan/code-structure.md`
  - `docs/matched-abstraction/task/code-structure-reorganization.md`
- Structure diagram: https://diashort.apps.quickable.co/d/b98307da
- `bun test` -> `27 pass`, `0 fail`.
- `bun run test:e2e` -> `1 pass`, `0 fail`.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit`.
- `bun run build` -> built `packages/sdk/dist`.
- `bun run pack:dry` -> `Total files: 6`.
- root `src`, root `dist`, root `tsconfig.json`, and root `tsdown.config.ts` structure probes -> no output.
- Added `.gitignore` for `node_modules`, `dist`, `coverage`, dry-pack tarballs, Bun install backups, Python bytecode, local env files, and editor noise.
- `git rev-parse --is-inside-work-tree` -> not a git repository, so `git check-ignore` cannot be used in this workspace.
- `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`.
- `bun run test:layers` -> `Ran 10 tests ... OK`.
- `find packages/sdk -maxdepth 1 -type f -name '*.tgz' -print` -> no output.
- `find . -type d -name __pycache__ -print` -> no output.

## RED-GREEN-TDD Plan For Platform Structure Reset

- [x] RED: Capture stale active Bun substrate authority in docs and handoff.
- [x] RED: Use Approach, Plan, and Task subagents to protect the platform reset layers.
- [x] GREEN: Add platform-structure Approach, Plan, and Task docs.
- [x] GREEN: Mark earlier Bun substrate authority as superseded evidence in NATS runtime docs.
- [x] VERIFY: Run layer validation, stale-authority review, no-slop scan, and layer unit tests.
- [x] REFACTOR: Do simplify and review passes for the docs-only reset.

## Platform Structure Reset Evidence

- Added matched-abstraction docs:
  - `docs/matched-abstraction/approach/platform-structure.md`
  - `docs/matched-abstraction/plan/platform-structure.md`
  - `docs/matched-abstraction/task/platform-structure-reset.md`
- Updated earlier NATS runtime docs so Bun and `@lagz0ne/nats-embedded` are preserved as historical or superseded evidence, not current platform authority.
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.
- stale-authority review -> remaining Bun and `@lagz0ne/nats-embedded` matches are explicitly evidence, superseded, historical, or not current authority.
- focused unresolved-marker scan over the new platform docs -> no matches.
- `find . -type d -name __pycache__ -print` -> no output.

## Platform Structure Direction

- Current substrate direction: Go owns NATS infrastructure, auth, Docker/sandboxing direction, process lifecycle, connection policy, command/activation ledger, artifact gateway, and execution attribution.
- Current frontend direction: Vite owns trusted shell delivery; generated content remains a receiver and intent emitter.
- Current SDK/schema direction: JSON Schema is the first neutral source; Zod validators, TypeScript SDK types, Go types/validators, fixtures, and parity tests are generated or checked from that source.
- Current validation direction: TypeScript runtime-facing boundaries use Zod validation, not type-only contracts.
- Current enforcement direction: code generation plus parity tests prevent Go, frontend, SDK, and tests from drifting.
- Existing Bun and `@lagz0ne/nats-embedded` work remains evidence and regression material, not platform authority.

## Session Note - 2026-06-08 Browser/NATS Kill Test

- Ad hoc triage-three Run 2 Pusher pass requested for browser-visible NATS credentials, capability lifecycle, and artifact HTTP auth parity.
- Scope is analysis only: no implementation or test files changed.
- Output should be numbered kill findings with required design constraints and future RED-GREEN-TDD proofs.

## RED-GREEN-TDD Plan For Managed Frontend Dedicated Worker

- [x] RED: Add Approach, Plan, and Task docs for the managed frontend mediator boundary.
- [x] RED: Add failing frontend mediator tests for typed content intents, raw NATS denial, materializer projection state, and dedicated-worker message routing.
- [x] GREEN: Implement the smallest frontend mediator contract, materializer store, and dedicated-worker bridge.
- [x] VERIFY: Run targeted tests, full `bun test`, typecheck, build, pack dry-run, and layer validation.
- [x] REFACTOR: Do no-slop, simplify, and review passes.

## Managed Frontend Dedicated Worker Evidence

- Runtime code added:
  - `src/browser-frontend/index.ts`
  - exports in `src/index.ts`
  - frontend mediator error kinds/layer in `src/nats-script-runtime/errors.ts`
- Tests added:
  - `tests/browser-frontend/dedicated-worker-mediator.test.ts`
- Layer docs added:
  - `docs/matched-abstraction/approach/browser-frontend-mediator.md`
  - `docs/matched-abstraction/plan/browser-frontend-mediator.md`
  - `docs/matched-abstraction/task/browser-frontend-dedicated-worker.md`
- RED evidence: targeted test failed before implementation with `Export named 'createFrontendMediator' not found in module '/home/lagz0ne/dev/tinkabot/src/index.ts'`.
- `bun test tests/browser-frontend/dedicated-worker-mediator.test.ts` -> `4 pass, 0 fail`.
- `bun test` -> `27 pass, 0 fail`.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit`.
- `bun run build` -> emitted ESM, CommonJS, and declaration artifacts.
- `bun pm pack --dry-run` -> `Total files: 5`.
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.
- Review pass found and fixed top-level raw NATS vocabulary bypass: generated content messages are scanned before parsing so fields such as `subject` or `token` cannot be silently dropped.
- Focused unresolved-marker scan over the new browser frontend slice -> no matches.
- `find . -type d -name __pycache__ -print` -> no output.

## Active Feature Inputs

- NATS-based system.
- Scripts are TypeScript and live in NATS-managed storage/subjects.
- Runtime can execute scripts.
- Approach decision: execution is a NATS request/reply command that also creates an attributed execution event trail.
- No sandbox for now, but metadata must record sandbox/security/runtime details.
- Approach decision: scripts are trusted-only until sandboxing exists; metadata is declaration/accountability, not enforcement.
- Tinkabot owns glue for CRUDing scripts and executing scripts.
- Approach decision: script source and metadata live together as versioned logical records in JetStream KV.
- Superseded approach: scripts do not get the TypeScript NATS client as the primary path.
- Current approach: default scripts use the process facade; direct TypeScript NATS client or CLI access is an explicit advanced capability.
- Approach correction: default scripts should not need a NATS client. They are process contracts using stdin/stdout/stderr plus runtime-owned IPC for publish/progress requests.
- Approach decision: NATS interaction by scripts goes through the Tinkabot/runtime facade by default; direct TS NATS client or CLI access is an explicit advanced capability, not the base path.
- Approach correction: choose a battle-tested long-run IPC contract, not a temporary convenience channel. Canonical process IPC is framed stdio RPC; fd 3 or shell helpers are adapters, not the core protocol.
- Scripts can consume provided content/data and publish output/events back to NATS, forming a closed loop.
- Approach decision: script metadata should use succinct, NATS-focused names and allow nested settings objects for future sub-configuration.
- Approach decision: metadata types should carry LLM-oriented `desc`/reasoning notes, not only runtime config.
- Approach decision: NATS capability metadata must model both access and exposure: inside-out publishing and outside-in invocation/consumption.
- Approach decision: NATS wildcard subject patterns are first-class metadata so scripts can declare broad interests and output families.
- Approach decision: metadata must not use placeholder subject strings; declared subjects/patterns are concrete values, and authority is encoded left-to-right in subject tokens.
- Resolved decision: NATS `permissions.publish`, `permissions.subscribe`, `allow`/`deny`, and `allow_responses` are the authoritative metadata vocabulary.
- Approach correction: scripts should not receive the whole NATS surface. They get a mediated mechanism with NATS-shaped security, controlled imports, and Tinkabot/runtime in the middle.
- Approach decision accepted for now: `imports` can be the script-facing abstraction, with `permissions` as the underlying NATS security contract. Revisit after the first plan pass if it becomes awkward.
- Resolved direction: script-facing schema and capability metadata cover input, output, IPC progress/publish requests, execution events, and imports without exposing unrestricted NATS flexibility.
- Superseded evidence: Bun previously owned package management, TypeScript execution, test harness, local process lifecycle, env assembly, and NATS startup/shutdown for early proofs.
- Superseded evidence: the previous local NATS provider was `@lagz0ne/nats-embedded` from `/home/lagz0ne/dev/nats-embedded`.
- Superseded evidence: the previous v1 local proof defaulted to `@lagz0ne/nats-embedded`; current platform authority moved to Go substrate.
- Local proof fact: Bun is installed, NATS CLI is installed, global `nats-server` is not installed.
- Superseded local library fact: `@lagz0ne/nats-embedded` exposes `NatsServer.start({ port, host, jetstream, storeDir, websocket, config, args })`, `server.url`, `server.port`, `server.exited`, and `server.stop()`; it remains prior proof material, not current substrate authority.

## RED-GREEN-TDD Plan For NATS Script Runtime Brainstorm

- [x] RED: Capture the current design gap and highest-risk branch decisions.
- [x] RED: Use a dedicated Approach subagent to protect purpose, invariants, non-goals, and readiness.
- [x] GREEN: Ask one branch-resolving question at a time with a recommended answer.
- [x] GREEN: Present Plan approaches and get approval for the design direction.
- [x] GREEN: After approval, create matched layer docs for the feature.
- [x] VERIFY: Validate layer docs and update this handoff with evidence.

## Candidate Plan Direction

- Recommended approach: contract-first fanout with one vertical lifecycle proof.
- Superseded recommendation: contract-first fanout with a Bun runtime substrate lane first, then one vertical lifecycle proof.
- Alternative: vertical lifecycle first, useful for integration discovery but can overfit the first script.
- Alternative: runtime-boundary first, useful for the hardest no-whole-NATS boundary but can delay storage/schema/execution contracts.
- Candidate visual: https://diashort.apps.quickable.co/d/cba2160a

## NATS Script Runtime Design Evidence

- Feature docs written:
  - `docs/matched-abstraction/approach/nats-script-runtime.md`
  - `docs/matched-abstraction/plan/nats-script-runtime.md`
  - `docs/matched-abstraction/task/nats-script-runtime-design.md`
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.
- `python3 -B /home/lagz0ne/.codex/skills/.system/skill-creator/scripts/quick_validate.py .codex/skills/matched-abstraction-thinking` -> `Skill is valid!`.
- placeholder and uncertainty wording scan -> no matches.
- generated bytecode directory scan -> no matches.
- Edge-case hardening added after user correction:
  - Approach: deny precedence, bounded response authority, exact attribution, and escalation when edge cases are dropped.
  - Plan: edge-case matrix for substrate, record, metadata, imports, mediation, execution, events, and cleanup.
  - Task: complete vertical proof criteria for success, validation failure, record failure, mediation failure, runtime failure, and recovery.

## Active Design Correction

- Do not let the first implementation become a loose MVP. The first slice can be small, but it must be precise, complete, and edge-case strict.
- The vertical proof must prove the real contract boundary, including failures and denials, not just the happy path.
- Edge-case pressure completed across Approach, Plan, and Task layers.
- IPC hardening: prefer cross-platform framed stdio RPC over fd-specific IPC as the canonical long-run contract.

## Active Test Planning Goal

- [x] Draw the runtime layer graph.
- [x] Define typed error sets per layer.
- [x] Define Resolve / Transform / Propagate ownership.
- [x] List tests by owning layer, not by end-to-end convenience.
- [x] Make the vertical proof a final trust-compounding check, not the only test.

## NATS Script Runtime Traced TDD Evidence

- Traced-TDD docs written:
  - `docs/matched-abstraction/plan/nats-script-runtime-traced-tdd.md`
  - `docs/matched-abstraction/task/nats-script-runtime-traced-tdd.md`
- Diagrams:
  - Primary layer graph: https://diashort.apps.quickable.co/d/d29e5453
  - Error ownership graph: https://diashort.apps.quickable.co/d/90f4566b
  - Protocol graph: https://diashort.apps.quickable.co/d/0da56487
  - Vertical proof graph: https://diashort.apps.quickable.co/d/12a339dc
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.
- no-slop wording scan over docs, tasks, and tests -> no matches.
- generated bytecode directory scan -> no matches.

## Next Implementation Order

1. DONE: RED/GREEN substrate and record-store tests.
2. DONE: Final-form distribution build plus BDD end-to-end proof for the current slice.
3. DONE: RED/GREEN metadata/schema and imports/permissions tests.
4. NEXT: promote activation/trigger into Approach, Plan, Traced TDD, and RED tests before framed stdio RPC and process runtime.
5. RED framed stdio RPC and process-runtime tests.
6. RED event-trail and execution-exchange tests.
7. RED vertical proof using embedded NATS and KV history.
8. GREEN only the minimum runtime code needed to satisfy declared contracts.
9. REFACTOR with no-slop, simplify, and review passes.

## NATS Script Runtime Substrate And Record Store Evidence

- Runtime code written:
  - `src/nats-script-runtime/errors.ts`
  - `src/nats-script-runtime/runtime-substrate.ts`
  - `src/nats-script-runtime/script-record-store.ts`
  - `src/nats-script-runtime/index.ts`
- Tests written:
  - `tests/nats-script-runtime/runtime-substrate.test.ts`
  - `tests/nats-script-runtime/script-record-store.test.ts`
- Project setup added:
  - `package.json`
  - `bun.lock`
  - `tsconfig.json`
- Task evidence doc added:
  - `docs/matched-abstraction/task/nats-script-runtime-substrate-record-store.md`
- RED evidence: `bun test` failed before implementation because `src/nats-script-runtime/index` did not exist.
- `bun test` -> `7 pass, 0 fail`.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit` completed successfully.
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.
- weak-word scan over source, tests, docs, tasks, and TS config -> no matches.
- generated bytecode directory scan -> no matches.

## NATS Script Runtime Distribution BDD Evidence

- Distribution setup written:
  - `src/index.ts`
  - `tsdown.config.ts`
  - package `version`, `main`, `module`, `types`, `exports`, `files`, `build`, and `test:e2e` fields
- BDD scenario written:
  - `tests/e2e/nats-script-runtime-distribution.feature.md`
  - `tests/e2e/nats-script-runtime-distribution.bdd.test.ts`
- Generated distribution:
  - `dist/index.mjs`
  - `dist/index.cjs`
  - `dist/index.d.mts`
  - `dist/index.d.cts`
- Task evidence doc added:
  - `docs/matched-abstraction/task/nats-script-runtime-distribution-bdd.md`
- RED evidence: `bun test tests/e2e/nats-script-runtime-distribution.bdd.test.ts` failed with `Script not found "build"` before build metadata existed.
- `bun run build` -> emitted ESM, CommonJS, and declaration artifacts.
- `bun test tests/e2e/nats-script-runtime-distribution.bdd.test.ts` -> `1 pass, 0 fail`.
- `bun test` -> `8 pass, 0 fail`.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit` completed successfully.
- `bun pm pack --dry-run` -> package contains 5 files: `package.json` plus four `dist` artifacts.
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.
- generated bytecode directory scan -> no matches.

## NATS Script Runtime Metadata And Permissions Evidence

- Runtime code written:
  - `src/nats-script-runtime/metadata-validator.ts`
  - `src/nats-script-runtime/permission-resolver.ts`
  - `src/nats-script-runtime/subjects.ts`
  - metadata and permission error kinds in `src/nats-script-runtime/errors.ts`
  - exports in `src/nats-script-runtime/index.ts`
- Tests written:
  - `tests/nats-script-runtime/metadata-validator.test.ts`
  - `tests/nats-script-runtime/permission-resolver.test.ts`
- Task evidence doc added:
  - `docs/matched-abstraction/task/nats-script-runtime-metadata-permissions.md`
- RED evidence: targeted new slice tests failed before implementation because `MetadataValidator` and `PermissionResolver` were not exported.
- `bun test tests/nats-script-runtime/metadata-validator.test.ts tests/nats-script-runtime/permission-resolver.test.ts` -> `8 pass, 0 fail`.
- `bun test` -> `16 pass, 0 fail`.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit` completed successfully.
- `bun run build` -> emitted ESM, CommonJS, and declaration artifacts.
- `bun pm pack --dry-run` -> package contains 5 files.
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.
- generated bytecode directory scan -> no matches.

## Active Activation Layer Design Branch

- Current request/reply execution is not enough to create chains.
- Add a middle activation layer rather than expanding substrate. Substrate keeps owning NATS lifecycle; activation owns event sources and converts them into execution intents.
- Candidate trigger sources: request/reply command, core subject subscription, JetStream durable consumer, KV watch, and schedule/timer provider.
- Time-based triggers require extra care: schedule state, leadership or leases, idempotency keys, missed-fire handling, and replay after restart.
- The activation layer must preserve the same constraints: no raw NATS for scripts by default, concrete subjects, NATS permission vocabulary, chain attribution, loop controls, and cleanup.
- Triage-three conclusion on 2026-06-05: stop review and move to contract drafting plus RED tests. Run 1 proposed 31 findings, Challengers confirmed 31, Arbiter collapsed them to 9 unique verified findings, and Investor recommended no second run.
- Verified activation findings: activation is a first-class layer, request/reply becomes an `ActivationIntent` adapter, durable activation state owns ledger/dedupe/cursors/ack policy, chain attribution and loop control are safety invariants, metadata needs activation/exposure declarations, subscribe authority must be enforced, substrate stays narrow through adapters, time activation is not first, and distribution BDD expands after activation is contracted.
- Next activation order: update Approach/Plan, update Traced TDD with activation errors and R/T/P rows, write RED tests for activation metadata and subscribe enforcement, normalize request/reply through `ActivationIntent`, then add one durable source before schedule work.

## RED-GREEN-TDD Plan For Activation Contract

- [x] RED: Use Approach, Plan, and Task layer subagents to define activation deltas without layer mixing.
- [x] RED: Update matched-abstraction docs so activation is authoritative before code changes.
- [x] RED: Write failing activation tests for metadata exposure, subscribe enforcement, and `ActivationIntent`.
- [x] GREEN: Add typed activation errors and the minimal activation/permission/metadata code needed for the RED tests.
- [x] VERIFY: Run targeted tests, full `bun test`, typecheck, build, distribution pack dry-run, and layer validation.
- [x] REFACTOR: Do no-slop, simplify, and review passes.

## NATS Script Runtime Activation Contract Evidence

- Runtime code written:
  - `src/nats-script-runtime/activation-intent.ts`
  - activation layer and error kinds in `src/nats-script-runtime/errors.ts`
  - `nats.activations` metadata validation in `src/nats-script-runtime/metadata-validator.ts`
  - subscribe and activation-source checks in `src/nats-script-runtime/permission-resolver.ts`
  - exports in `src/nats-script-runtime/index.ts`
- Tests written:
  - `tests/nats-script-runtime/activation-intent.test.ts`
  - activation cases in `tests/nats-script-runtime/metadata-validator.test.ts`
  - activation permission cases in `tests/nats-script-runtime/permission-resolver.test.ts`
- Docs updated:
  - `docs/matched-abstraction/approach/nats-script-runtime.md`
  - `docs/matched-abstraction/plan/nats-script-runtime.md`
  - `docs/matched-abstraction/plan/nats-script-runtime-traced-tdd.md`
  - `docs/matched-abstraction/task/nats-script-runtime-activation-contract.md`
- Diagram added: https://diashort.apps.quickable.co/d/407896c2
- RED evidence: targeted activation tests failed with `8 pass, 3 fail, 1 error` before `ActivationIntent`, subscribe resolver methods, and activation metadata validation existed.
- `bun test tests/nats-script-runtime/metadata-validator.test.ts tests/nats-script-runtime/permission-resolver.test.ts tests/nats-script-runtime/activation-intent.test.ts` -> `12 pass, 0 fail`.
- `bun test` -> `20 pass, 0 fail`.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit`.
- `bun run build` -> emitted ESM, CommonJS, and declaration artifacts.
- `bun pm pack --dry-run` -> `Total files: 5`.
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.
- Review pass found and fixed activation exposure mismatch: `assertActivationSource` now rejects observed subjects that do not match the declared activation subject before checking subscribe permission.
- unresolved-marker scan over docs, tasks, source, and tests -> no matches.
- `find . -type d -name __pycache__ -print` -> no matches.

## RED-GREEN-TDD Plan For Request/Reply Activation Adapter

- [x] RED: Use Approach, Plan, and Task layer subagents to constrain the adapter slice.
- [x] RED: Add request/reply activation adapter task doc before code.
- [x] RED: Write failing adapter tests for authorization, intent preservation, and R/T/P behavior.
- [x] GREEN: Add the minimal request/reply adapter module and exports.
- [x] VERIFY: Run targeted tests, full `bun test`, typecheck, build, pack dry-run, and layer validation.
- [x] REFACTOR: Do no-slop, simplify, and review passes.

## NATS Script Runtime Request/Reply Activation Adapter Evidence

- Runtime code written:
  - `src/nats-script-runtime/request-reply-activation-adapter.ts`
  - exports in `src/nats-script-runtime/index.ts`
- Tests written:
  - `tests/nats-script-runtime/request-reply-activation-adapter.test.ts`
- Task doc added:
  - `docs/matched-abstraction/task/nats-script-runtime-request-reply-activation-adapter.md`
- RED evidence: targeted adapter test failed with `0 pass, 1 fail, 1 error` before `activateRequestReply` existed.
- `bun test tests/nats-script-runtime/request-reply-activation-adapter.test.ts` -> `3 pass, 0 fail`.
- `bun test` -> `23 pass, 0 fail`.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit`.
- `bun run build` -> emitted ESM, CommonJS, and declaration artifacts.
- `bun pm pack --dry-run` -> `Total files: 5`.
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.
- Review/simplify pass narrowed the adapter test double to `ActivateRequestReplyOptions["resolver"]` instead of casting to full `PermissionResolver`.
- unresolved-marker scan over docs, tasks, source, and tests -> no matches.
- `find . -type d -name __pycache__ -print` -> no matches.

## Completed Interrupt

Add Karpathy-inspired agent guidelines to root agent instruction files.

## RED-GREEN-TDD Plan For Interrupt

- [x] RED: Confirm root `AGENTS.md` and `CLAUDE.md` do not already exist.
- [x] RED: Choose one source of truth: `AGENTS.md` real file, `CLAUDE.md` symlink.
- [x] GREEN: Add compact Karpathy/Codex-native guardrails to `AGENTS.md`.
- [x] GREEN: Create `CLAUDE.md -> AGENTS.md` symlink.
- [x] VERIFY: Prove the symlink, content, and test status.
- [x] REFACTOR: Do no-slop, simplify, and review passes for the instruction edit.

## Interrupt Verification Evidence

- `ls -la AGENTS.md CLAUDE.md` -> `CLAUDE.md -> AGENTS.md`.
- `cmp -s AGENTS.md CLAUDE.md` -> `cmp_exit=0`.
- `python3 -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.
- `python3 .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- Note: running the layer validator on `.` fails by design because the validator expects the layer root containing immediate `approach/`, `plan/`, and `task/` directories.

## Assumptions Approved By "go"

- Project-local skill path: `.codex/skills/matched-abstraction-thinking`.
- Project abstraction documents path: `docs/matched-abstraction/`.
- Layer directories: `approach/`, `plan/`, `task/`.
- Each layer is represented by a dedicated subagent when the skill is used.
- The main agent acts as orchestration, synthesis, verification, and change announcer.

## RED-GREEN-TDD Plan

- [x] RED: Define pressure cases for Approach, Plan, and Task layer behavior.
- [x] RED: Ask separate subagents to pressure-test each layer contract.
- [x] GREEN: Create the skill, reference material, project layer docs, and validation script.
- [x] GREEN: Run skill validation and structural validation.
- [x] REFACTOR: Do no-slop, simplify, and review passes.
- [x] DONE: Summarize verified evidence and remaining caveats.

## Notes

- The workspace started empty and is not currently a git repository.
- No `.c3/` directory was present, so the C3 codemap workflow does not apply yet.
- Final verification evidence is recorded in `docs/matched-abstraction/task/baseline-skill-setup.md`.
