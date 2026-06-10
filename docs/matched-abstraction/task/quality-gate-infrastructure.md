---
layer: task
topic: quality-gate-infrastructure
status: complete
references:
  - ../approach/endgame-app.md
  - ../approach/go-substrate.md
  - ../plan/quality-v1.md
---

# Quality Gate Infrastructure Task

## Objective

Establish the four standing quality gates and the shared embednats harness factory seam so every later `quality-v1` slice runs under them (`../plan/quality-v1.md`, First Executable Task Slice). The stable gate operations are `bun run gate:parallel`, `bun run gate:fakes`, `bun run gate:coverage`, and `bun run gate:scenarios`; their checkers live in `scripts/gate-{parallel,fakes,coverage,scenarios}.ts` over shared corpus scanning in `scripts/gate-lib.ts`.

## Scope

This task owns:

- `scripts/gate-parallel.ts`: full Go suite green with `-shuffle=on` and parallel execution; every top-level Test func calls `t.Parallel()` or carries a `gate:serial` justification; direct `embednats.Start` construction in test files confined to one harness factory seam file.
- `scripts/gate-fakes.ts`: every detected fake type (Memory/InMemory/Fake/Mock/Stub prefixed) must appear in `substrate/go/fakes-allowlist.json` with a written impossible-to-force-branch justification and a resolving real-NATS proof test; anything un-allowlisted fails.
- `scripts/gate-coverage.ts`: inside-out coverage measured per substrate layer (`contract`, `core`, `edge`, `embednats`, `frontend`) against declared thresholds in `substrate/go/coverage-thresholds.json`.
- `scripts/gate-scenarios.ts`: outside-in scenario-matrix completeness in `substrate/go/scenario-matrix.json` over the seven pinned case families (allowed, denied-neighbor, malformed, duplicate, stale, revoked, attributed-failure) per outside-in surface, with citations resolving to committed Go tests. The family list is pinned in the checker so the matrix cannot weaken its own gate.
- At GREEN: the one harness factory seam every embednats test uses for isolated servers/stores, the parallel-safe corpus migration with unchanged assertions, the allowlist/threshold/matrix declarations, and the injected-violation detection proof (an un-allowlisted fake and a deliberately shared server must each FAIL their gate).

## Non-Goals

- No exposure API change (`typed-exposure-posture`), no auth backend change (`operator-jwt-authority`), no binary work (`tinkabot-binary`), no manual edits and no `release:evidence` extension (`quality-release`).
- No new product features, runtime behavior, or test assertions — structural harness and gate work only; the slice proves the same assertions under new discipline.
- No promotion of Plan-deferred scope as proven (direct browser WebSocket, Docker sandboxing, product UI, multi-node HA, package publication).

## Acceptance Contract

- All four gate operations exist under stable names and pass on the migrated corpus; each fails with concrete, file-attributable findings in the slice's owned failure families: fakes violation, isolation violation, coverage gap, scenario-matrix hole, measurement stale (`../plan/quality-v1.md`, Handoff Contracts).
- `go test ./... -count=1 -shuffle=on` from `substrate/go` is green under parallel execution with per-test isolated embedded-NATS servers and stores obtained through one factory seam.
- Each allowlisted fake carries a written narrow-branch justification plus the real-NATS proof that validates it; denial oracles stay output-parsed, never exit-code.
- Detection proof, not reporting: an injected un-allowlisted fake and a deliberately shared server each fail their gates before GREEN closes.
- No gate may be presented as passing without that injected-violation proof.

## RED Artifact

RED is the four gate operations failing on the current, unmodified corpus (no Go file changed; no allowlist, thresholds, or matrix exists). Executed 2026-06-10:

- `bun run gate:fakes` -> exit 1, `4 findings (fakes-violation=4)`:
  - `no fakes allowlist at substrate/go/fakes-allowlist.json; every fake below is un-allowlisted`
  - `un-allowlisted fake MemoryLedgerStore defined at core/core.go:918` — 17 test usage sites across `core/ledger_durability_test.go`, `core/schedule_engine_test.go`, `embednats/source_router_test.go`
  - `un-allowlisted fake MemoryScheduleStore defined at core/schedule.go:45` — 5 usage sites in `core/schedule_engine_test.go`
  - `un-allowlisted fake MemoryMaterialStore defined at core/script_materializer.go:169` — 4 usage sites in `core/script_materializer_test.go`, `embednats/script_materializer_test.go` (a third fake beyond the two the slice contract enumerated; detection found it, which is the point)
- `bun run gate:parallel` -> exit 1, `55 findings (serialized-execution=54, isolation-violation=1)`: all 53 top-level Test funcs under `substrate/go` never call `t.Parallel()` and carry no `gate:serial` justification (e.g. `embednats/source_router_test.go:14 TestSourceRouterAcceptsLiveSourcesOverEmbeddedNATS`); plus `no single harness factory seam: embednats.Start constructed directly in 8 test files (embednats/browser_gateway_test.go:20; embednats/embednats_test.go:16,70,182,193,262; embednats/ledger_test.go:98; embednats/release_proof_test.go:285,314; embednats/schedule_test.go:19; embednats/script_materializer_test.go:348; embednats/source_authority_cli_test.go:29; embednats/source_router_test.go:311)`. The shuffled suite run is skipped when structure is serialized, because a green serial run proves nothing about isolation.
- `bun run gate:coverage` -> exit 1, `6 findings (coverage-gap=6)`: `absent per-layer measurement: no declared thresholds at substrate/go/coverage-thresholds.json`, plus one finding per undeclared layer (`contract`, `core`, `edge`, `embednats`, `frontend`).
- `bun run gate:scenarios` -> exit 1, `1 findings (scenario-matrix-hole=1)`: `absent matrix definition: no substrate/go/scenario-matrix.json declaring the pinned case families (allowed, denied-neighbor, malformed, duplicate, stale, revoked, attributed-failure) per outside-in surface`.

Baseline proving the corpus itself is green pre-refactor (the deficiencies are structural, not behavioral): `go test ./... -count=1 -shuffle=on` from `substrate/go` -> all 5 packages `ok`; `go test ./embednats -count=1` -> `ok`, 7.45s. `bun run typecheck:orchestrator` -> exit 0, so the gate failures are contracted findings, not script errors.

A second detection RED is owed at GREEN time: an injected un-allowlisted fake and a deliberately shared server must each fail their gate, proving the gates detect rather than report.

## Verification Evidence

RED phase executed 2026-06-10 against the unmodified corpus (only the gate checkers, `package.json` script names, and this doc added):

- `bun run gate:fakes` -> exit 1, `gate:fakes FAILED: 4 findings (fakes-violation=4)`; findings name `MemoryLedgerStore` (`core/core.go:918`), `MemoryScheduleStore` (`core/schedule.go:45`), `MemoryMaterialStore` (`core/script_materializer.go:169`) with file:line usage sites, and the missing `substrate/go/fakes-allowlist.json`.
- `bun run gate:parallel` -> exit 1, `gate:parallel FAILED: 55 findings (serialized-execution=54, isolation-violation=1)`; 53 Test funcs without `t.Parallel()`, plus `embednats.Start` constructed directly in 8 test files.
- `bun run gate:coverage` -> exit 1, `gate:coverage FAILED: 6 findings (coverage-gap=6)`; no `substrate/go/coverage-thresholds.json`, no threshold for any of the five layers.
- `bun run gate:scenarios` -> exit 1, `gate:scenarios FAILED: 1 findings (scenario-matrix-hole=1)`; no `substrate/go/scenario-matrix.json`.
- Baseline: `go test ./... -count=1 -shuffle=on` (from `substrate/go`) -> `ok` for all 5 packages (`contract` 0.041s, `core` 0.071s, `edge` 0.045s, `embednats` 7.126s, `frontend` 0.003s); `go test ./embednats -count=1` -> `ok` 7.450s; `bun run typecheck:orchestrator` -> exit 0.

GREEN executed 2026-06-10. Structural change only — no assertion changed; the diff is `t.Parallel()` insertions, `Start(cfg)`-to-seam migrations, fake-justification comments, and the three declaration files:

- Harness factory seam: `substrate/go/embednats/harness_test.go` `start(t, cfg)` is now the only `Start(` construction site in test files; all 13 direct call sites across 8 embednats test files migrated, per-call `t.Cleanup(stop)` moved into the seam. Per-test isolation is what `valid(t)` already provided (random port `-1`, `t.TempDir()` store dir, per-test buckets); the seam pins it.
- Parallel migration: all 53 top-level Test funcs across `contract`, `core`, `edge`, `embednats`, `frontend` now call `t.Parallel()` (no `gate:serial` exception needed; no test touches env, cwd, fixed ports, or shared servers).
- Declarations: `substrate/go/fakes-allowlist.json` (3 entries — MemoryLedgerStore, MemoryScheduleStore, MemoryMaterialStore — each with a narrow impossible-to-force-branch justification and a resolving real-NATS proof test; matching justification comments added at the type definitions in `core/core.go`, `core/schedule.go`, `core/script_materializer.go`); `substrate/go/coverage-thresholds.json` (contract 70, core 78, edge 78, embednats 72, frontend 95 — floors under measured 73.9/81.7/82.8/76.8/100); `substrate/go/scenario-matrix.json` (two genuinely complete outside-in surfaces, `activation-release-proof` and `live-source-router`, each citing committed tests for all seven pinned families).

Gate results on the migrated corpus:

- `bun run gate:fakes` -> exit 0, `gate:fakes passed`.
- `bun run gate:parallel` -> exit 0: structure clean, then `go test ./... -count=1 -shuffle=on` all 5 packages `ok` (embednats 4.6-6.3s, down from 7.1s serial).
- `bun run gate:coverage` -> exit 0: contract 73.9%>=70, core 81.7%>=78, edge 82.8%>=78, embednats 76.8%>=72, frontend 100%>=95.
- `bun run gate:scenarios` -> exit 0, `gate:scenarios passed`.
- `go test ./... -count=1 -shuffle=on` (from `substrate/go`) -> all 5 packages `ok`, repeated across 4 independent shuffled runs (2 direct, 2 via gate:parallel), different shuffle seeds each.
- `go test ./embednats -count=1` -> `ok` 4.57s.
- `bun run typecheck:orchestrator` -> exit 0; `bun run validate:layers` -> passed; `bun run test:layers` -> 10 tests OK.

Injected-violation detection proof (executed, then reverted):

- Injected `type FakeInjectedStore struct{}` in `core/injected_fake_test.go` (tracked) -> `bun run gate:fakes` exit 1, `fakes-violation=1: un-allowlisted fake FakeInjectedStore defined at core/injected_fake_test.go:4`. Removed; gate back to exit 0.
- Injected `embednats/injected_shared_test.go` constructing `Start(valid(t))` outside the seam (tracked) -> `bun run gate:parallel` exit 1, `isolation-violation=1: embednats.Start constructed directly in 2 test files (embednats/harness_test.go:12; embednats/injected_shared_test.go:8)` plus the suite-run-skipped finding. Removed; gate back to exit 0 with the full shuffled suite green.

Both gates detect, not merely report. Scope guards held: no exposure/auth/binary/manual/release-evidence change; no new assertions; denial oracles remain output-parsed (`wantDenied`, CLI output checks untouched).

Full verification suite (slice closeout, all executed 2026-06-10):

- `bun run test` -> `85 pass`, `0 fail`, `427 expect() calls` across 17 files (7.67s; includes `tests/gate-checkers.test.ts` over the four checkers).
- `bun run test:e2e` -> `1 pass`, `0 fail`, `16 expect() calls` (2.98s).
- `bun run typecheck` -> frontend, SDK, and orchestrator all clean via `bunx @typescript/native-preview` (exit 0, no errors).
- `bun run build` -> frontend build plus SDK tsdown: CJS 64.78kB, ESM 63.51kB, d.ts 32.48kB, build complete in ~2s.
- `bun run pack:dry` -> `tinkabot-0.1.0.tgz`, 6 files, unpacked 194.45KB.
- `bun run orchestrate:codex -- --dry-run --allow-dirty` -> `Endgame already DONE; verification passed.` (exit 0).
- `bun run release:evidence` -> `release evidence check passed: 16 milestones over 11 spine steps`.
- `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`.
- `bun run test:layers` -> `Ran 10 tests` in 0.602s, `OK`.
- `git diff --check` -> clean; no whitespace or conflict-marker issues (exit 0, empty output).

Gate results:

- real-nats: pass — every embednats test runs over a real embedded NATS server obtained through the harness seam; the three allowlisted fakes each carry a written narrow-branch justification and a resolving real-NATS proof test, enforced by `gate:fakes`.
- parallel-safety: pass — all 53 Test funcs call `t.Parallel()` with per-test isolated servers/stores; `go test ./... -count=1 -shuffle=on` green across 4 independent shuffled runs with different seeds; `go vet ./...` clean.
- coverage: pass — `gate:coverage` exit 0 (contract 73.9%>=70, core 81.7%>=78, edge 82.8%>=78, embednats 76.8%>=72, frontend 100%>=95) and `gate:scenarios` exit 0 (both outside-in surfaces cite committed tests for all seven pinned families).
- be-lazy: pass — gate checkers use inference-first style with explicit types only at the finding contract and the three declaration wire formats; the Go diff is structural insertions/migrations with no new ceremony.
- security: pass — no authority surface changed; least-authority test credentials and output-parsed denial oracles untouched; injected-violation proofs were reverted and confirmed absent.
- no-slop: pass — diff-scoped scan clean: no slop vocabulary, narrating comments, or placeholder markers in the gate scripts, declarations, or migrated tests.

## Wrap-Up

Quality-gate-infrastructure is done. The four standing gates — `bun run gate:fakes`, `bun run gate:parallel`, `bun run gate:coverage`, `bun run gate:scenarios` — exist under their stable names, pass on the migrated corpus, and are proven detectors: an injected un-allowlisted fake and a deliberately out-of-seam server each failed their gate before GREEN closed. The test corpus runs fully parallel with `-shuffle=on` over per-test isolated embedded NATS servers obtained through the single harness factory seam at `substrate/go/embednats/harness_test.go`; the three Memory fakes are allowlisted with narrow-branch justifications and real-NATS proofs; per-layer coverage floors and the seven-family scenario matrix are declared and enforced. No assertion changed — the same behavior is now proven under the new discipline. Later `quality-v1` slices (`typed-exposure-posture` next) change only the harness factory seam and run under these gates.
