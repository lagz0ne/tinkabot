---
layer: task
topic: session-contract-authority
status: complete
references:
  - ../approach/session-v2.md
  - ../plan/session-v2.md
---

# Session Contract Authority Task

## Objective

Establish the canonical session contract in the neutral `schemas/base/v1` lane, parity-proven across TS/Zod and Go with zero new wiring: session record shape, session frame vocabulary (out: token / chunk / status; in: steer / stop), steering intent, and the trust-tier typed value with an owner-layer tag distinguishing it from the connection-exposure `AuthTier` and from ledger run-claims. JSON Schema is first; checked TS types, Zod, Go validators, and owner-layer-tagged fixtures follow it.

This task proves contract authority only. Schema validity proves shape; it never grants effect authority (Approach invariant 2).

## Scope

This task owns:

- session record shape with provenance.
- session frame vocabulary: out-frames `token`, `chunk`, `status`; in-frames `steer`, `stop`.
- frame origin as a contract field: `status` frames are runner-originated lifecycle frames; `token`/`chunk` frames are wrapper-originated — making a wrapper-emitted status frame rejectable (the `FakeStatusImpersonation` hook slices 3 and 4 prove live).
- steering intent shape.
- trust-tier typed value, owner-layer-tagged, distinct from connection-exposure `AuthTier` and ledger run-claims; neither existing vocabulary is redefined, merged, or migrated.
- `session.cases.json` driving both validation targets through the existing contract registry so `bun run schema:parity` covers session shapes with no new wiring.
- boundary completeness: parity plus malformed-frame and unknown-frame-kind denial fixtures.
- rawWords reserved-vocabulary resolution: frame field names avoid `rawWords` substrings (the facade scan in `substrate/go/core/script_materializer.go:24-445` matches field names only, never values, so the frame kind value `token` does not itself collide). An explicit facade carve-out is fallback only, and must not weaken the script-effect facade scan for non-session paths; existing facade denial tests stay green unchanged.

## Non-Goals

- No session process, supervisor, liveness lease, republisher, mint path, steering delivery, or browser surface (slices 2-7).
- No effect-authority enforcement: single-writer steering, mint subject-breadth, republisher quota, lease liveness, and viewer custody belong to their named owner layers; fixtures carry owner-layer tags so authority decisions stay out of this slice.
- No live NATS session-subject watcher, so no denied-neighbor proof is claimed here.
- No raw terminal / PTY mode, no redaction or content filtering, no real agent runner.
- No concrete `tb.session.*` subject finalization beyond what contract fields require; no scenario-matrix entry; no gate or `release/v1.json` manifest extension (Slice 7 owns closure).
- No new fakes: contract tests are pure schema/fixture validation with no NATS seam.

## Acceptance Contract

- Canonical session shapes exist in `schemas/base/v1/contract.schema.json` (`sessionRecord`, `sessionFrame`, `steerIntent`, `trustTier`) and TS/Zod plus Go validators agree with the canonical schema on every `session.cases.json` fixture.
- `bun test packages/sdk/tests/base-contract/session-contract-authority.test.ts` passes with one owning test per failure family: `SchemaParityMismatch` (T-SESSION-PARITY), `UnknownFrameKind` (T-SESSION-UNKNOWN-FRAME), `MissingProvenance` (T-SESSION-MISSING-PROVENANCE), `ReservedVocabCollision` (T-SESSION-RESERVED-VOCAB).
- `go test ./contract -count=1` from `substrate/go` validates all session fixtures including the malformed-frame and unknown-frame-kind denial fixtures; the new Go test is parallel-safe (`t.Parallel()`, no shared state).
- `bun run schema:parity` covers the session shapes through the existing registry with no new wiring.
- Every session fixture carries an owner-layer tag; valid out-frame fixtures carry frame origin (`status` -> `runner`; `token`/`chunk` -> `wrapper`).
- No valid session fixture field name contains a `rawWords` substring (normalized, name-only — values such as the frame kind `token` are exempt by the scan's own rule).
- The facade denial regression guard stays green unchanged: `go test ./core -run 'TestScriptRuntimeMaterializesMediatedEffects|TestScriptRuntimeAttributesFailures' -count=1`.

## RED Artifact

A pair of new owning tests failing against the absent contract, captured before any schema or validator was written:

- `packages/sdk/tests/base-contract/session-contract-authority.test.ts` fails because the session schema defs, `session.cases.json`, the session fixtures (valid record / token / chunk / status / steer / stop and the invalid malformed-frame, unknown-frame-kind, missing-provenance denials with owner-layer tags), and SDK validation do not exist in `schemas/base/v1`.
- `substrate/go/contract/session_test.go` (`TestSessionContractParity`, parallel-safe) fails because session fixtures are not covered by the Go contract registry lane.

This proves the gap the slice owns: `bun run schema:parity` previously passed while validating zero session shapes, so frame vocabulary, frame origin, steering intent shape, trust-tier typing, provenance requirements, and the rawWords naming constraint were unpinned for every downstream slice.

## Verification Evidence

RED (executed 2026-06-11, recorded verbatim):

- Command: `bun test packages/sdk/tests/base-contract/session-contract-authority.test.ts`
- Result: `0 pass`, `4 fail`, `Ran 4 tests across 1 file.`
- T-SESSION-PARITY failure: `expect(schema.$defs.sessionRecord).toBeDefined()` -> `error: expect(received).toBeDefined()` / `Received: undefined` — the canonical schema has no session defs.
- T-SESSION-UNKNOWN-FRAME failure: `ENOENT: no such file or directory, open '/home/lagz0ne/dev/tinkabot/schemas/base/v1/fixtures/invalid/session-frame-unknown-kind.json'`.
- T-SESSION-MISSING-PROVENANCE failure: `ENOENT: no such file or directory, open '/home/lagz0ne/dev/tinkabot/schemas/base/v1/fixtures/invalid/session-record-missing-provenance.json'`.
- T-SESSION-RESERVED-VOCAB failure: `ENOENT: no such file or directory, open '/home/lagz0ne/dev/tinkabot/schemas/base/v1/session.cases.json'`.
- Owning layer: Contract authority. Missing contract proven: no neutral session contract exists in `schemas/base/v1`.

- Command: `cd substrate/go && go test ./contract -count=1`
- Result: `--- FAIL: TestSessionContractParity (0.01s)` / `session_test.go:33: session fixtures are not covered by the contract registry lane: open ../../../schemas/base/v1/session.cases.json: no such file or directory` / `FAIL github.com/lagz0ne/tinkabot/substrate/go/contract 0.042s`.
- Owning layer: Contract authority. Missing contract proven: Go contract registry validates zero session fixtures.

- Command: `bun run schema:parity`
- Result: fails at `test:contracts` with `21 pass`, `4 fail` — the 4 failures are exactly the new session owning tests; all pre-existing base-contract tests still pass, proving parity previously covered zero session shapes.

- Regression guard (must stay green, did): `cd substrate/go && go test ./core -run 'TestScriptRuntimeMaterializesMediatedEffects|TestScriptRuntimeAttributesFailures' -count=1` -> `ok github.com/lagz0ne/tinkabot/substrate/go/core 0.007s`.

GREEN (executed 2026-06-11, recorded verbatim):

- Command: `bun test packages/sdk/tests/base-contract/session-contract-authority.test.ts`
- Result: `4 pass`, `0 fail`, `54 expect() calls`, `Ran 4 tests across 1 file.` — one owning test per failure family green: T-SESSION-PARITY, T-SESSION-UNKNOWN-FRAME, T-SESSION-MISSING-PROVENANCE, T-SESSION-RESERVED-VOCAB.

- Command: `cd substrate/go && go test ./contract -count=1`
- Result: `ok github.com/lagz0ne/tinkabot/substrate/go/contract 0.043s` — `TestSessionContractParity` validates all nine session fixtures (six valid, three denials) through the existing registry.

- Command: `bun run schema:parity`
- Result: green end-to-end — `test:contracts`: `25 pass`, `0 fail`, `Ran 25 tests across 5 files.` (21 pre-existing + 4 new session owning tests); `test:go`: `ok` for all packages including `contract`.

- Regression guard (unchanged, still green): `cd substrate/go && go test ./core -run 'TestScriptRuntimeMaterializesMediatedEffects|TestScriptRuntimeAttributesFailures' -count=1` -> `ok github.com/lagz0ne/tinkabot/substrate/go/core 0.019s`.

- `bun run typecheck` (via `@typescript/native-preview`): clean — frontend, sdk, orchestrator.

- `git diff --check` -> clean.

Full battery on the final tree (executed 2026-06-11 from the repo root; Go from `substrate/go`):

- `bun run test` -> PASS: 100 pass, 0 fail, 492 expect() calls across 18 files (7.53s).
- `bun run test:e2e` -> PASS: 1 pass, 0 fail, 16 expect() calls (4.17s).
- `bun run typecheck` -> PASS: frontend, sdk, orchestrator all clean via `bunx @typescript/native-preview` (exit 0, no errors).
- `bun run build` -> PASS: frontend vite build + sdk build complete (ESM 64.85 kB + d.mts 34.32 kB, CJS 66.22 kB).
- `bun run pack:dry` -> PASS: `tinkabot-0.1.0.tgz`, 6 files, unpacked 200.92KB.
- `bun run schema:parity` -> PASS: 25 contract tests across 5 files pass; `go test ./...` all 7 packages ok.
- `bun run release:evidence` -> PASS: 16 milestones over 11 spine steps, 5 gate results.
- `bun run gate:fakes` -> PASS: `gate:fakes passed`.
- `bun run gate:parallel` -> PASS: `gate:parallel passed`; all 7 Go packages ok under parallel run.
- `bun run gate:coverage` -> PASS: all layers above thresholds — cmd 70.8%>=65, contract 73.9%>=70, core 81.7%>=78, edge 82.8%>=78, embednats 78.6%>=72, frontend 100%>=95, tinkabot 82.3%>=75.
- `bun run gate:scenarios` -> PASS: `gate:scenarios passed`.
- `bun run gate:manual` -> PASS: `gate:manual passed`.
- `cd substrate/go && go test ./... -count=1` -> PASS: all 7 packages ok uncached — cmd/tinkabot 0.310s, contract 0.084s, core 0.139s, edge 0.083s, embednats 4.693s, frontend 0.005s, tinkabot 5.089s.
- `git diff --check` -> PASS: exit 0, no whitespace/conflict-marker errors.

Gate results: real-nats PASS, parallel-safety PASS, coverage PASS, security PASS, be-lazy PASS, no-slop PASS.

## Execution Notes

GREEN was the smallest complete artifact set; the RED tests were not modified.

- `schemas/base/v1/contract.schema.json`: added `$defs` `trustTier`, `sessionState`, `sessionRecord`, `sessionFrame` (oneOf `sessionTokenFrame` / `sessionChunkFrame` / `sessionStatusFrame`), and `steerIntent` (oneOf steer / stop discriminated by `intent` const); `sessionRecord`, `sessionFrame`, `steerIntent` joined the top-level `oneOf`.
- Frame origin is pinned by const: `status` -> `origin: "runner"`, `token`/`chunk` -> `origin: "wrapper"`, so a wrapper-emitted status frame is contract-rejectable (FakeStatusImpersonation hook; live rejection owned by slices 3-4).
- `trustTier` is `{ tier: "untrusted" | "trusted", ownerLayer: const "trusted-wrapper-authority" }` — the owner-layer tag distinguishes it from the connection-exposure `AuthTier` (embednats exposure) and from ledger run-claims; neither existing vocabulary was touched.
- rawWords resolution by naming only, no facade carve-out: session field names are `kind`, `frame`, `origin`, `sessionId`, `runnerId`, `state`, `trust`, `tier`, `ownerLayer`, `text`, `body`, `detail`, `intent` plus the existing provenance fields — none contains a `rawWords` substring (normalized, name-only). The frame kind value `token` is exempt by the scan's own name-only rule. The facade scan in `substrate/go/core` is unchanged.
- `packages/sdk/src/base-contract/index.ts`: mirrored Zod schemas added to the existing `Contract` union; `parseContract` denial path (ContractInvalid at ContractAuthority) reused unchanged. Chunk `body` rides the existing `safeValue` scanner.
- Zero new Go wiring: the Go registry compiles the whole canonical schema, so extending the schema covered Go validation; the only Go addition remains the RED-phase `session_test.go` (`t.Parallel()`, no shared state).
- Fixtures: six valid + three denial fixtures under `schemas/base/v1/fixtures/{valid,invalid}/`, all owner-layer-tagged via `session.cases.json` (record -> session-runtime-subsystem, frames -> session-frame-mediation, steer/stop -> steering-acceptance, denials -> session-contract-authority).
- No effect-authority claim, no runtime, no subjects fixed, no scenario-matrix or manifest change, no new fakes — per scope guards.

## Wrap-Up

`session-contract-authority` is complete — session-v2 slice 1/7 is DONE. The canonical session contract exists in the neutral `schemas/base/v1` lane: `sessionRecord`, `sessionFrame` (out: `token`/`chunk`/`status`; in: `steer`/`stop`), `steerIntent`, and the owner-layer-tagged `trustTier`, with frame origin pinned by const so a wrapper-emitted status frame is contract-rejectable. TS/Zod and Go validators agree with the canonical schema on all nine `session.cases.json` fixtures (six valid, three denials) through the existing contract registry — zero new wiring; `bun run schema:parity` now covers session shapes (25 contract tests, up from 21 that validated zero session shapes). Each of the four declared failure families has one green owning test (T-SESSION-PARITY, T-SESSION-UNKNOWN-FRAME, T-SESSION-MISSING-PROVENANCE, T-SESSION-RESERVED-VOCAB), and the parallel-safe `TestSessionContractParity` validates every fixture from Go. The rawWords reserved-vocabulary constraint is resolved by naming alone — no facade carve-out, the `substrate/go/core` facade scan is unchanged, and the facade denial regression guard is green unchanged. Every fixture carries an owner-layer tag, so authority decisions stay with their named downstream owners; neither `AuthTier` nor ledger run-claims was touched. All six gates (real-nats, parallel-safety, coverage, security, be-lazy, no-slop) and the full release battery pass on the final tree. The contract authority that slices 2-7 consume is pinned; the next slice is `session-runtime-subsystem`.
