---
layer: task
topic: quality-release
status: complete
references:
  - ../approach/endgame-app.md
  - ../plan/quality-v1.md
  - ../plan/endgame-app.md
  - ./tinkabot-binary.md
  - ./release-spine.md
  - ./quality-gate-infrastructure.md
---

# Quality Release Task

## Objective

Close the `quality-v1` program per `../plan/quality-v1.md` slice 5: one centralized operation proves the quality program the same way `bun run release:evidence` proves the sixteen v1 milestones. `bun run gate:manual` runs the manual's commands verbatim against the running binary and checks the documented outcomes; the extended `bun run release:evidence` over `release/v1.json` conventions validates the four standing gate results (`gate:fakes`, `gate:parallel`, `gate:coverage`, `gate:scenarios`) plus the manual-verbatim result. This slice consumes — never re-proves — the slice-4 creds-mode evidence: `TestBinaryManual`, `CheckManual`, and the "Starting the binary" section of `docs/manual/v1.md` (`./tinkabot-binary.md` closeout).

## Scope

- `bun run gate:manual` (`scripts/gate-manual.ts`): the manual-verbatim gate named by `../plan/quality-v1.md` (Verification Strategy operation table) — manual commands run verbatim against the binary, denial oracles output-parsed, never exit-code (nats CLI v0.3.0 exits 0 on permission errors).
- The `release:evidence` extension: `release/v1.json` conventions gain gate-result entries; the checker hardcodes the required gate list (the four standing gates plus `gate:manual`) the same way it hardcodes the Plan milestone list, so the manifest cannot weaken its own gates.
- New owned failure families, each genuinely detected by unit tests over synthetic corpora: `gate-result-missing`, `gate-overclaim`, `manual-divergence`, plus the existing `evidence-stale` discipline extended to gate results.
- The deferred list stays named and unproven: direct browser NATS WebSocket, Docker sandboxing, product UI rendering beyond the shell, broad script CRUD UI, multi-node HA, package publication. Live auth reload and the product entry surface are the only two formerly deferred items now closed (`operator-jwt-authority`, `tinkabot-binary`).

## Non-Goals

- No runtime features: like `release-spine`, this slice only packages and checks evidence (`../plan/quality-v1.md` ordering: `quality-release` is last and adds no runtime behavior).
- No reopening of completed slices; any unsupported claim routes to the owning slice — manual divergence to `tinkabot-binary`, gate-detection gaps to `quality-gate-infrastructure`, exposure gaps to `typed-exposure-posture`, auth gaps to `operator-jwt-authority`.
- No re-running `TestBinaryManual`/`CheckManual` proofs as new work; slice 4's creds-mode evidence is consumed input.
- No weakening of the release-spine checker: the hardcoded gate list (sixteen milestones, eleven spine steps, eight deferred items, four scope guards, seven pinned case families) stays intact; the extension adds families, it does not relax them.
- No edits to `docs/manual/v1.md` content; divergence found by `gate:manual` routes back to `tinkabot-binary`. The Known wart stands: verbatim `go build ./cmd/tinkabot` collides with the package directory name; `go build -o /tmp/tinkabot-bin ./cmd/tinkabot` is the working form.
- No package publication and no un-deferring of any deferred-scope item.

## Acceptance Contract

- `bun run gate:manual` exists and runs the manual's commands verbatim against the running binary, producing the documented outcomes; every denial oracle is output-parsed.
- The extended `bun run release:evidence` fails on a synthetic missing gate result (`gate-result-missing`), fails on a synthetic overclaimed gate result — presented as passing with no landed proof or with a recorded failure (`gate-overclaim`), fails on manual-verbatim evidence whose cited command diverges from the manual (`manual-divergence`), and passes on the real corpus with the deferred list intact.
- `tests/release-evidence.test.ts` proves each new failure family is genuinely detected on synthetic corpora, so the extended checker cannot rubber-stamp — the same discipline it already enforces for the four release-spine families.
- `tests/gate-checkers.test.ts` proves the `gate:manual` checker detects a live outcome diverging from the documented one and passes verbatim-matching transcripts.
- Scope guards stay encoded: HA/scale contract-shape only, managed-auth policy-compile level, schedule engine without a live tick source, nats CLI denial output-parsed; no gate is presented as passing before its owning slice landed the proof.

## RED Artifact

RED is the extended checker's detection gap made concrete on executable artifacts, mirroring the release-spine RED pattern: real findings, attributable, before implementation.

1. The Plan-named operation does not exist: `package.json` has no `gate:manual` script (only `gate:parallel`/`gate:fakes`/`gate:coverage`/`gate:scenarios`), so `bun run gate:manual` fails to run at all, and `scripts/gate-manual.ts` does not exist.
2. New unit tests over synthetic corpora fail because the current checker cannot detect the new failure families: `tests/release-evidence.test.ts` gained six tests (a corpus with no gate-results block, a corpus missing one required gate result, a gate presented as passing with no landed result line in the cited doc, a gate presented as passing whose recorded result is a failure, manual-verbatim evidence citing a command absent from the manual, and a manual-verbatim citation against a missing manual doc) — every one fails with `Received: []` because `check()` emits nothing for gate results. `tests/gate-checkers.test.ts` gained two `gate:manual` tests that fail on `Cannot find module '../scripts/gate-manual'`.
3. The current `bun run release:evidence` passes on the real corpus while saying nothing about gate results or manual verbatimness — the quality-program claims are unverifiable by the centralized gate today.

The failure proves detection is genuinely absent now, so GREEN is detection capability, not rubber-stamping.

## GREEN Execution Notes

Implemented 2026-06-10 as `scripts/gate-manual.ts` plus the `release:evidence` extension in `scripts/release-evidence.ts`, with `release/v1.json` gaining a `gateResults` block:

- **`gate:manual`** (`bun run gate:manual`): parses the manual's bash blocks into command/outcome pairs (a command line directly followed by `# -> outcome`, comment-continued until a blank line), builds the binary with `go build -o <tmp>/tinkabot-bin ./cmd/tinkabot` (the Known wart's working form), starts it on an isolated store with `--shell 127.0.0.1:0`, parses the printed posture for the client URL and role creds paths, lands the script record behind the manual's documented observations through the author flow, then runs each pair verbatim under the manual's connection preamble (`--creds`, creds mode) — observer creds for `$MATERIAL_BUCKET`/`$ARTIFACT_BUCKET` reads, caller creds otherwise. Documented outcomes elide volatile values with `...`; the stable anchors between elisions must all appear in the live output, whitespace-insensitively. All oracles are output text, never exit codes. Commands without a documented outcome carry no verbatim claim here; their creds-mode proof stays `TestBinaryManual` (consumed slice-4 input, never re-proved).
- **Checker extension** (`scripts/release-evidence.ts`): `REQUIRED_GATES` hardcodes the five Plan gates (`gate:fakes`, `gate:parallel`, `gate:coverage`, `gate:scenarios`, `gate:manual`) next to `MILESTONES`, so the manifest cannot weaken its own gates. Each required gate needs a landed `gateResults` entry whose cited doc exists and carries the recorded result line; a missing entry is `gate-result-missing`; a result line absent from the cited doc or recording a non-pass is `gate-overclaim`; a result line sitting outside the cited doc's executed verification evidence is `evidence-stale` (the milestone discipline extended to gate results); `gate:manual` additionally cites the manual doc and its verbatim commands — a command absent from the manual is `manual-divergence`, a missing manual doc is `citation-unresolved`. The four release-spine families and their gate lists are untouched.
- **Manual divergences found, routed to `tinkabot-binary`, fixed in its owned surfaces** (same precedent as slice 4's `--hdr` -> `-H`): (1) the binary's served wiring used invented names (`tb.app.trigger.main`, `tb.app.events.main`, `tb_config`, `tb_upload`) while the manual — the usage contract the binary must satisfy unchanged (`../plan/quality-v1.md` pinned decision) — documents `tb.proof.runtime.execute`, `tb.proof.events.main`, `config_bucket`, `artifacts`; `substrate/go/tinkabot/tinkabot.go` `wiring()` now serves the manual's literals (assembly constants only, every test derives from `Wiring`). (2) the manual documented the trigger reply as `Accepted`/`Duplicate`, but the wire status is lowercase (`core.Accepted = "accepted"`); the `# -> Accepted` outcome line and the two status mentions in "Triggering work" now carry the executed lowercase forms, while the PascalCase typed kinds (`LeaseRevoked`, `StaleCursor`, `DeniedNeighbor`, `LoopSuppressed`, `SourceMalformed`) are wire-accurate and stand.
- **Known boundary:** the manual's "Defining a script" example JSON uses `"key"`/`"revision"` where the strict decoder reads `"scriptKey"`/`"scriptRevision"` (`core.ScriptRecord` tags). That example carries no documented outcome, so it is outside the pair-bound verbatim surface; noted for `tinkabot-binary`/`script-materializer-loop` as manual-example drift, not fixed here.
- **`release/v1.json`**: gains `gateResults` — one entry per required gate (`gate`, `command`, `result`, `doc` citing this Task) plus `manual`/`verbatim` for `gate:manual` naming the three documented command/outcome pairs held verbatim against the running binary. Milestones, spine coverage, deferred scope (all eight items), doc authority, and scope guards are unchanged.

Landed gate results (executed 2026-06-10, quoted by `release/v1.json` `gateResults`):

- `bun run gate:fakes` -> `gate:fakes passed`.
- `bun run gate:parallel` -> `gate:parallel passed` (all 7 Go packages ok under the shuffled parallel gate).
- `bun run gate:coverage` -> `gate:coverage passed` (tinkabot 82.3%>=75, frontend 100%>=95, all layers at or above their declared thresholds).
- `bun run gate:scenarios` -> `gate:scenarios passed`.
- `bun run gate:manual` -> `gate:manual passed` (three documented command/outcome pairs verbatim against the running binary: trigger reply `accepted`, projection `p.main`, artifact manifest `a.YXJ0aWZhY3QvbWFpbi5qcw`).

## Verification Evidence

RED (executed 2026-06-10 from the repo root):

- `bun run gate:manual` -> exit 1, `error: Script not found "gate:manual"` — the Plan-named operation does not exist.
- `bun run release:evidence` -> exit 0, `release evidence check passed: 16 milestones over 11 spine steps` — the centralized gate passes while validating no gate result and no manual-verbatim claim: the detection gap.
- `bun test tests/release-evidence.test.ts` -> exit 1, `21 pass, 6 fail`: all six new failure-family tests fail with `Expected to contain: "gate-result-missing" / "gate-overclaim" / "manual-divergence" / "citation-unresolved"` against `Received: []`; all 21 release-spine family tests still pass.
- `bun test tests/gate-checkers.test.ts` -> exit 1, `8 pass, 2 fail`: both `gate:manual` tests fail with `Cannot find module '../scripts/gate-manual'`; all 8 existing gate tests still pass.

GREEN (executed 2026-06-10 from the repo root; Go from `substrate/go`):

- `bun run gate:manual` -> exit 0, `gate:manual passed`: the binary is built (`go build -o`, Known wart form), started on an isolated store, and the manual's three documented command/outcome pairs run verbatim in creds mode — trigger reply `accepted`, projection `p.main`, manifest `a.YXJ0aWZhY3QvbWFpbi5qcw` — all anchors matched in live output.
- `bun test tests/release-evidence.test.ts` -> 29 pass, 0 fail: all 21 release-spine family tests plus the eight gate-result/manual-verbatim tests (`gate-result-missing` x3 including a `gate:manual` entry gutted of manual/verbatim, `gate-overclaim` x2, gate-result `evidence-stale`, `manual-divergence`, missing-manual-doc `citation-unresolved`).
- `bun test tests/gate-checkers.test.ts` -> 11 pass, 0 fail: the 8 existing gate tests plus three `gate:manual` checker tests (measurement-stale on a manual with no command/outcome pairs, live divergence detected, verbatim match passes).
- `bun run release:evidence` -> exit 0, `release evidence check passed: 16 milestones over 11 spine steps, 5 gate results`.
- Synthetic negatives over the real corpus (mutated `release/v1.json`, then restored): dropping the `gate:coverage` entry -> exit 1, `gate-result-missing (1): required gate gate:coverage has no landed result entry`; recording `gate:coverage FAILED: 2 findings` as the result -> exit 1, `gate-overclaim (1): gate gate:coverage is presented as passing with no landed result line`; adding verbatim command `nats request tb.phantom.subject ping` -> exit 1, `manual-divergence (1): manual-verbatim command not found in docs/manual/v1.md`. Restored corpus passes.
- Live manual divergence (tampered `# -> accepted` to `# -> LeaseRevoked`, then restored): `bun run gate:manual` -> exit 1, `manual-divergence (1): ... missing "LeaseRevoked" (live: accepted)` — the gate reads the running binary, not the document's claim.
- `go test ./... -count=1` -> ok uncached for all 7 packages (`cmd/tinkabot`, `contract`, `core`, `edge`, `embednats`, `frontend`, `tinkabot`) with the routed wiring fix in place.
- Gates on the final tree: `gate:fakes passed`, `gate:parallel passed` (full shuffled corpus), `gate:coverage passed` (tinkabot 82.3%>=75, all layers at threshold), `gate:scenarios passed`, `gate:manual passed`.
Full battery on the final tree (executed 2026-06-10 from the repo root; Go from `substrate/go`):

- `bun run test` -> 96 pass, 0 fail (438 expect() calls, 17 files).
- `bun run test:e2e` -> 1 pass, 0 fail (16 expect() calls).
- `bun run typecheck` -> pass: frontend, sdk, orchestrator all clean via `bunx @typescript/native-preview`.
- `bun run build` -> pass: vite frontend build into `substrate/go/frontend/site` plus sdk tsdown (CJS+ESM, dist 4 files).
- `bun run pack:dry` -> pass: `tinkabot-0.1.0.tgz`, 6 files, 194.45KB unpacked, exit 0.
- `bun run schema:parity` -> pass: contract tests 21 pass / 0 fail; Go contract package ok.
- `go test ./... -count=1` -> pass: all 7 packages ok uncached (`embednats` 4.32s, `tinkabot` 4.63s).
- `bun run release:evidence` -> pass: `release evidence check passed: 16 milestones over 11 spine steps, 5 gate results`.
- `bun run gate:fakes` -> `gate:fakes passed`.
- `bun run gate:parallel` -> `gate:parallel passed`: all 7 Go packages ok under the shuffled parallel gate.
- `bun run gate:coverage` -> `gate:coverage passed`: cmd 70.8%>=65, contract 73.9%>=70, core 81.7%>=78, edge 82.8%>=78, embednats 78.6%>=72, frontend 100%>=95, tinkabot 82.3%>=75.
- `bun run gate:scenarios` -> `gate:scenarios passed`.
- `bun run gate:manual` -> `gate:manual passed`.
- `git diff --check` -> clean, exit 0.

Gate results: real-nats PASS, parallel-safety PASS, no-slop PASS, security PASS, coverage PASS, be-lazy PASS.

## Wrap-Up

`quality-release` is complete and closes the `quality-v1` program — all five slices are DONE. The Plan-named manual-verbatim gate exists: `bun run gate:manual` builds the binary (`go build -o`, the Known wart's working form), starts it on an isolated store, and runs the manual's three documented command/outcome pairs verbatim in creds mode against the running binary — trigger reply `accepted`, projection `p.main`, artifact manifest `a.YXJ0aWZhY3QvbWFpbi5qcw` — with every oracle output-parsed and divergence proven live (a tampered documented outcome fails `manual-divergence` against the running binary's actual output). The extended `bun run release:evidence` is the single centralized release gate for both programs: it hardcodes the five required gates next to the sixteen milestones, validates 5 landed gate results plus the manual-verbatim claim over `release/v1.json`, and its three new failure families (`gate-result-missing`, `gate-overclaim`, `manual-divergence`) plus the gate-result `evidence-stale` extension are each genuinely detected on synthetic corpora (29-test checker self-proof) and on synthetic negatives over the real corpus. The four release-spine families, the eight-item deferred list, and the four scope guards are intact and unweakened. The two manual divergences found were routed to and fixed in `tinkabot-binary`'s owned surfaces (served wiring literals, lowercase wire statuses); the manual-example JSON drift is noted as owned by `tinkabot-binary`/`script-materializer-loop`, not fixed here. All five gates, all six slice gates, and the full release battery are green on the final tree. The v1 platform target is reached: sixteen v1 milestones plus five quality-v1 slices, all proven through the centralized evidence gate.
