---
layer: task
topic: release-spine
status: complete
references:
  - ../approach/endgame-app.md
  - ../plan/endgame-app.md
---

# Release Spine Task

## Objective

Package one centralized release authority for endgame v1: a machine-checkable evidence manifest at `release/v1.json` plus the centralized operation `bun run release:evidence` that validates it. The manifest maps all sixteen endgame milestones, including this one, onto the eleven Release Verification Spine steps and records the Plan-owned deferred-scope list and the doc authority map. This slice consumes the `Release-Spine Decomposition` handoff in `docs/matched-abstraction/plan/endgame-app.md`; it does not invent gate rules.

## Scope

This task owns:

- `release/v1.json`: the v1 evidence manifest with one entry per milestone naming its owning Task doc, covered spine steps, RED citation, inside-out proof commands, outside-in proof commands or an explicit not-applicable reason, negative-case coverage, and scope guards.
- `scripts/release-evidence.ts`: the checker behind `bun run release:evidence`, owning the four failure families from the Plan: manifest-incomplete, citation-unresolved, scope-overclaim, and evidence-stale.
- `tests/release-evidence.test.ts`: unit proof that each failure family is genuinely detected, so the checker cannot rubber-stamp.
- Routed evidence fixes in owning Task/Plan docs for the concrete gaps the checker surfaces.

## Non-Goals

- No new runtime features: no Go substrate behavior, activation sources, script runtime capability, frontend surface, or schema contract beyond what the manifest cites.
- No implementation of deferred scope: direct browser NATS WebSocket, Docker sandboxing, product UI rendering, live auth reload, wall-clock scheduler loops beyond the engine proof, broad script CRUD UI, live multi-node HA/scale, package publication.
- No re-running or rewriting of completed milestones' proofs; the manifest cites existing executed evidence.
- No Approach reopening unless the authority model itself is wrong.

## Acceptance Contract

- `bun run release:evidence` is the single release gate. It fails on any incomplete, unresolved, overclaiming, or stale manifest entry and passes only on the corrected corpus.
- All sixteen milestones have manifest entries and every one of the eleven spine steps is covered by at least one entry.
- All citations resolve mechanically: cited docs exist, cited command strings appear in the owning Task doc, and cited Go tests resolve as `go test -run` prefix patterns against committed test files.
- Negative-case coverage cites executed verification evidence that names the case; aggregate pass results do not count.
- The manifest names the eight Plan-owned deferred-scope items and never presents them as proven.
- The four Plan scope guards are encoded: HA/scale at contract shape only, managed-auth at policy-compile level, schedule engine proof without a live NATS tick source, and `nats` CLI denial as an output-parsed oracle.
- The doc authority map names current Approach/Plan authority per domain, and superseded docs carry supersession markers.
- The checker itself is proven against all four failure families by unit tests over synthetic corpora.

## RED Artifact

RED is the release-evidence checker failing on the current repository state, in two stages.

Stage 1, structural: before this slice there was no release authority at all — no `release/` directory and no `release:evidence` entry in `package.json`, so the centralized operation could not run.

Stage 2, meaningful: with the checker and the candidate manifest in place, `bun run release:evidence` fails on the current doc corpus by surfacing the known evidence gaps as findings:

- `frontend-isolation-layer` lacks an executed RED command/result citation (its RED section lists expectations only, and its denial evidence is a single counter), reported as manifest-incomplete.
- `command-acceptance`, `substrate-edge-bootstrap`, `go-substrate-core`, and `activation-release-proof` hide negative-case evidence behind aggregate pass results, reported as evidence-stale per case.
- Ten completed task docs, plus this one until GREEN closes it, end with template wrap-ups instead of completion announcements, reported as evidence-stale.
- `docs/matched-abstraction/plan/browser-frontend-mediator.md` is superseded by the browser-isolation Plan but carries no supersession marker, reported as evidence-stale.

Fixes for these findings route to the owning Task/Plan docs; GREEN is the checker passing on the corrected corpus.

## Execution Notes

The checker hardcodes the Plan gate list (sixteen milestones, eleven spine steps, eight deferred-scope items, four guard requirements) from `docs/matched-abstraction/plan/endgame-app.md` so the manifest cannot weaken its own gates. Go test citations resolve against `git ls-files` committed `_test.go` files under `substrate/go`, never against invented test names. The `pack:dry` file count is a measured output, not a constant, so the manifest cites the command, not a frozen number.

GREEN routed every RED finding to its owning doc; no checker rule was weakened and no runtime feature was added:

- `frontend-isolation-layer`: an executed RED was captured live (2026-06-10) by temporarily removing `apps/frontend/src/isolation.ts` and running `bun run test:frontend`, proving the isolation proof depends on the implementation; the failure and the restored `4 pass` result are recorded in the owning doc and cited by the manifest.
- Aggregate-hidden negative cases: the owning docs for `command-acceptance`, `substrate-edge-bootstrap`, `go-substrate-core`, and `activation-release-proof` now record named per-case evidence from re-executed narrow runs of the same committed tests (`-t` filters for bun, `-run` prefixes with `-v` for Go), and the manifest quotes those case-naming lines instead of aggregate pass results.
- Eleven template wrap-ups were rewritten as completion records in the owning task docs, including this one.
- `docs/matched-abstraction/plan/browser-frontend-mediator.md` now carries a Browser Isolation Supersession section naming `plan/browser-isolation.md` as the current v1 browser plan authority.

## Verification Evidence

Stage 1 RED (no release authority exists):

- `ls release` -> `"release": No such file or directory (os error 2)`.
- `grep release:evidence package.json` before this slice -> no matches; the centralized operation did not exist.

Stage 2 RED (checker fails on the current corpus for the contracted reasons):

- `bun run release:evidence` -> `release evidence check FAILED: 27 findings (manifest-incomplete=1, evidence-stale=26)`; findings include `[frontend-isolation-layer] entry lacks an executed RED command/result citation`, fourteen per-case `hidden behind an aggregate pass result` findings for `command-acceptance`, `substrate-edge-bootstrap`, `go-substrate-core`, and `activation-release-proof`, eleven `wrap-up is a template announcement` findings, and `superseded doc docs/matched-abstraction/plan/browser-frontend-mediator.md carries no supersession marker`.
- `bun run typecheck:orchestrator` -> clean; the new `scripts/release-evidence.ts` and `tests/release-evidence.test.ts` typecheck under `@typescript/native-preview`.

Checker self-proof (the failure is meaningful, not a syntax error):

- `bun test tests/release-evidence.test.ts` -> `21 pass`, `0 fail`; rejects malformed manifests and incomplete entries with `manifest-incomplete` findings, rejects commands and quotes that cannot be found where the manifest claims with `citation-unresolved` findings, rejects omitted guards, unnamed deferred scope, and exit-code denial oracles with `scope-overclaim` findings, and flags stale template wrap-ups and aggregate-hidden negative-case evidence as `evidence-stale` findings; every finding is attributed to a failure family and owning milestone.

Routed fix evidence (executed for GREEN, recorded in the owning docs and cited by the manifest):

- `frontend-isolation-layer` executed RED capture (2026-06-10): with `apps/frontend/src/isolation.ts` temporarily moved aside, `bun run test:frontend` -> `error: Cannot find module '../src/isolation'`, `1 fail`, exit `1`; restored -> `4 pass`, `0 fail`, `19 expect() calls`, with `apps/frontend` clean in `git status`.
- Named command-acceptance cases: `bun test packages/sdk/tests/endgame-contract/command-acceptance.test.ts -t T-CMD-IDEMPOTENCY` -> `2 pass`; `-t T-CMD-DENY` -> `1 pass`; `-t T-CMD-CONTRACT` -> `1 pass`; `0 fail` each — duplicate, stale-revision, and raw-authority-rejection evidence named per case.
- Named Go cases from `substrate/go`: `go test ./edge -run TestSubstrateDeniesLeaseBeforeCredentialDescriptor -v`, `go test ./core -run 'TestBuildPlanDeniesBeforeAuthority|TestCredentialLeaseBookRevokesIdempotently|TestErrorAttribution' -v`, and `go test ./embednats -run TestActivationReleaseProofFailureAttribution -v` -> all `PASS` with named subtests (revoked/expired/provenance; malformed/denied_neighbor/duplicate/stale_cursor/revoked_lease/loop_suppressed); the `embednats` run executes over a real embedded NATS server.

GREEN (checker passes on the corrected corpus):

- After routing fixes to the owning docs, intermediate `bun run release:evidence` -> `1 findings (evidence-stale=1)`: only this doc's own template wrap-up remained, proving the routed fixes closed exactly the contracted gaps and nothing else.
- `bun run release:evidence` -> `release evidence check passed: 16 milestones over 11 spine steps`, exit `0`.
- `bun test tests/release-evidence.test.ts` -> `21 pass`, `0 fail` (all four failure families still genuinely detected on synthetic corpora).
- Operational denial path: with `release/v1.json` temporarily absent, `bun run release:evidence` -> `FAILED: 1 findings (manifest-incomplete=1)`, exit `1`; restored, the gate passes again.

Full verification suite (release-shaped closeout, all executed):

- `bun run schema:parity` -> endgame contract tests `21 pass`, `0 fail` (`195 expect() calls`); frontend build ok; `go test ./...` -> `ok` for `contract`, `core`, `edge`, `embednats`, and `frontend`.
- `go test ./...` from `substrate/go` -> `ok` for `contract`, `core`, `edge`, `embednats` (6.877s), and `frontend`; `0` failures.
- `bun run test` -> `77 pass`, `0 fail`, `417 expect() calls` across 16 files.
- `bun run test:e2e` -> `1 pass`, `0 fail`, `16 expect() calls`.
- `bun run typecheck` -> frontend, SDK, and orchestrator all clean via `bunx @typescript/native-preview --noEmit`.
- `bun run build` -> Vite frontend build ok (6 modules); SDK tsdown emits CJS and ESM dist with `.d.cts`/`.d.mts`.
- `bun run pack:dry` -> `tinkabot-0.1.0.tgz`, 6 files, unpacked 194.54KB; the file count is a measured output, cited as the command per the Execution Notes.
- no-slop scan over browser isolation docs, frontend shell proof, gateway policy, service-worker setup, schemas, SDK validation, and Go validation -> clean: no slop vocabulary, emoji, narrating comments, or placeholder markers; the working tree matched `main`, so no diff-introduced slop.
- `git diff --check` -> clean; no whitespace or conflict-marker issues.

Gate results:

- real-nats: pass — outside-in proofs the manifest cites run over the real embedded NATS runtime; the checker itself audits docs and the manifest, recorded as the entry's explicit outside-in not-applicable reason.
- parallel-safety: pass — this slice adds no Go tests; the re-executed narrow runs use existing committed parallel-safe tests with isolated servers.
- coverage: pass — sixteen milestone entries cover all eleven spine steps, and all seven pinned case families are cited by named evidence.
- be-lazy: pass — `scripts/release-evidence.ts` and its tests use inference-first style with explicit types only at the manifest wire format and finding contract.
- security: pass — no authority surface changed; deferred scope stays named-not-proven and supersession markers are verified, not weakened.
- no-slop: pass — focused scan clean across the audited docs and the new checker code.

## Wrap-Up

Release-spine is done. One centralized release authority — `bun run release:evidence` over `release/v1.json` — validates the endgame v1 evidence corpus across all sixteen milestones and eleven spine steps, with deferred scope named, the four Plan scope guards enforced, the doc authority map recorded, and every RED finding routed to and fixed in its owning Task/Plan doc. The gate passes on the corrected corpus, fails on a missing manifest, and is itself proven against all four failure families. This closes the sixteenth and final endgame v1 milestone; the next program is `quality-endgame`.
