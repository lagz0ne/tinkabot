---
layer: task
topic: typed-exposure-posture
status: complete
references:
  - ../approach/endgame-app.md
  - ../approach/go-substrate.md
  - ../plan/quality-v1.md
---

# Typed Exposure Posture Task

## Objective

Make exposure a typed posture declared through the harness factory seam (`substrate/go/embednats/harness_test.go` `start(t, cfg)`), not a port number (`../plan/quality-v1.md`, slice 2): in-process default via `server.Options.DontListen` + `Server.InProcessConn()` + `nats.InProcessServer` (nats.go v1.52.0, pinned decision), explicit loopback opt-in carrying the existing `nats` CLI outside-in proofs unchanged, and a typed denied-by-default external tier per surface (NATS port, WebSocket, HTTP gateway) where missing auth tier or missing TLS beyond loopback is a typed error, never a warning.

## Scope

This task owns the four failure families that today have no owner:

- **Undeclared exposure**: raw socket-requesting config (`Port`, `WebSocket`) without a declared posture fails typed (`ExposureUndeclared`).
- **Posture mismatch**: the declared posture must provably be the posture the server has — no-socket check for in-process, bound-address check for loopback; divergence fails typed (`ExposureMismatch`).
- **In-process connection failure**: failures on the in-process transport are typed (`InProcessConnFailed`).
- **External tier policy violation**: external opt-in without a matching auth tier or without TLS fails typed (`ExposureDenied`).

At GREEN: the typed posture API on the embednats `Config`/`Posture`, migration of every embednats proof to a declared posture through the single seam with assertions unchanged, new tests `t.Parallel()` over isolated servers/stores, and all four standing gates (`gate:fakes`, `gate:parallel`, `gate:coverage`, `gate:scenarios`) green again.

## Non-Goals

- No operator/JWT auth migration — static auth stays; `operator-jwt-authority` is slice 3 and must not run concurrently (`../plan/quality-v1.md`, ordering).
- No binary assembly, lifecycle, or key materialization (`tinkabot-binary`); no `release:evidence` extension, `gate:manual`, or `docs/manual/v1.md` edits (`quality-release`).
- No live TLS provisioning or real external-network serving: the external tier lands as a typed, denied-by-default option with typed denial paths only; "matching auth tier" is a typed policy check, not a working auth backend (`../plan/endgame-app.md`, scope guards).
- No claim of HA/multi-node exposure, schedule liveness, direct browser WebSocket, or any behavior/assertion change in migrated tests — posture declaration is the only test-side change.
- `nats` CLI denial evidence stays an output-parsed oracle, never exit-code.

## Acceptance Contract

- `go test ./embednats -run 'TestExposure' -count=1 -v` from `substrate/go` passes: in-process default serves a real JetStream KV round trip with no TCP endpoint advertised; declared loopback yields a bound, dialable `127.0.0.1` socket carrying outside-in TCP traffic; each external surface without auth tier or TLS is denied typed; a forced listener behind a declared in-process posture is refused typed.
- `go test ./embednats -count=1` and `go test ./core ./embednats -count=1` green with the whole corpus declaring postures through the one seam, assertions unchanged.
- All four standing gates green; any fake forced for an impossible branch is allowlisted with a code-comment justification (the posture-mismatch test uses the in-package construction hook on the real nats-server — no fake type).

## RED Artifact

RED is `substrate/go/embednats/exposure_test.go`: six parallel-safe tests against the real embedded runtime, referencing the not-yet-existing typed posture API (`Exposure`, `InProcess`, `Loopback`, `External`, `ExternalSurfaces`, `TierExternal`, `TLSFiles`, `ExposeInProcess`, `ExposeLoopback`, `Config.Exposure`, `Posture.Exposure`) and typed denial kinds (`ExposureUndeclared`, `ExposureDenied`, `ExposureMismatch`, `InProcessConnFailed`). Executed 2026-06-10 from `substrate/go`:

- `go test ./embednats -run 'TestExposure' -count=1 -v` -> exit 1, build failure on the missing symbols (full list via `-gcflags=-e`, 45 errors), e.g.:
  - `embednats/exposure_test.go:33:6: cfg.Exposure undefined (type Config has no field or method Exposure)`
  - `embednats/exposure_test.go:86:9: p.Exposure undefined (type Posture has no field or method Exposure)`
  - `embednats/exposure_test.go:169:26: undefined: ExposureUndeclared` / `200:26: undefined: ExposureDenied` / `225:24: undefined: ExposureMismatch` / `246:24: undefined: InProcessConnFailed`
  - `FAIL github.com/lagz0ne/tinkabot/substrate/go/embednats [build failed]`
- `go test ./embednats -count=1` -> exit 1, same build failure (the corpus cannot run until the seam grows the posture API).
- `go test ./core -count=1` -> exit 0, `ok` — core vocabulary untouched by RED.
- `bun run gate:parallel` -> exit 1, `1 findings (isolation-violation=1)`: `go test ./... -count=1 -shuffle=on exited 1`, caused solely by the RED artifact's build failure; zero structural findings against `exposure_test.go` (every new test calls `t.Parallel()` and obtains its server through `start(t, cfg)`). GREEN must restore this gate.

The failure proves the gap is real: `Config` (`substrate/go/embednats/embednats.go:60-84`) takes raw `Host`/`Port`/`WebSocket`, `Start` always binds a loopback TCP socket (harness `valid(t)` uses `Port: -1`, so every test holds an undeclared loopback socket), no in-process connection path exists (`grep -rn DontListen|InProcessConn|InProcessServer substrate/go` -> no hits), and nothing denies undeclared or mismatched exposure.

## Verification Evidence

RED phase executed 2026-06-10 (GREEN and full wrap-up evidence follow under the Capability Proof Matrix and Wrap-Up Verification sections):

- `go test ./embednats -run 'TestExposure' -count=1 -v` (from `substrate/go`) -> exit 1: `FAIL github.com/lagz0ne/tinkabot/substrate/go/embednats [build failed]` on the missing typed posture/denial symbols.
- `go test ./embednats -count=1` -> exit 1, same build failure.
- `go test ./core -count=1` -> exit 0: `ok github.com/lagz0ne/tinkabot/substrate/go/core 0.045s`.
- `bun run gate:parallel` -> exit 1: `gate:parallel FAILED: 1 findings (isolation-violation=1)` — only the RED build failure, no structural findings against the new tests.

## Execution Notes (GREEN, 2026-06-10)

Changed (smallest surface that turns RED green at its boundary):

- `substrate/go/embednats/exposure.go` (new): the typed posture vocabulary — `Exposure`/`ExposureMode` (`ExposeInProcess`, `ExposeLoopback`, `ExposeExternal`), constructors `InProcess()`/`Loopback()`/`External(ExternalSurfaces)`, `AuthTier`/`TierExternal`, `TLSFiles`, `ExternalSurfaces`, `ExposurePosture{Mode, Addr}`, plus `Config.exposure()` resolution: zero-value defaults to in-process; raw `Host`/`Port`/`WebSocket`/`Core.Topology.WebSocket` fields under an in-process posture -> `ExposureUndeclared`; external surfaces checked per surface for `TierExternal` and full `TLSFiles` -> `ExposureDenied`; a fully declared external tier is still `ExposureDenied` ("denied by default; live external serving is not provided") — the tier is typed and policy-checked only, per the scope guard against claiming a live external surface.
- `substrate/go/embednats/embednats.go`: four new `Kind`s (`ExposureUndeclared`, `ExposureDenied`, `ExposureMismatch`, `InProcessConnFailed`); `Config.Exposure` and `Posture.Exposure` fields; `Start` resolves the posture before defaults, sets `DontListen` (no `Host`/`Port`) for in-process, and after readiness checks declared-vs-actual via `srv.Addr()` (in-process must have no listener; loopback must be bound on `127.0.0.1`) -> `ExposureMismatch`; in-process posture publishes `ClientURL: ""` and the owned probe client dials `nats.InProcessServer(srv)`; `Connect`/`ConnectAs` collapse into one `dial` helper that uses the in-process transport when the posture says so and maps its connect failures to `InProcessConnFailed` (loopback keeps `ClientConnectFailed`).
- `substrate/go/embednats/embednats_test.go`: corpus migration is one line — `valid(t)` declares `Exposure: Loopback()`. Every existing proof flows through `valid(t)` + `start(t, cfg)`, so assertions are unchanged and the loopback posture carries the `nats` CLI outside-in proofs as before.

Found and bounded during GREEN (upstream, not ours): rejecting auth over `Server.InProcessConn()` deadlocks the synchronous `net.Pipe` handshake (server writes `-ERR` while the client is still blocked writing; TCP rejects in <1ms with `nats: Authorization Violation`), and only the server `WriteDeadline` (default 10s) breaks it, surfacing as `io: read/write on closed pipe` after 10.000s. Mitigation: in-process runtimes set `opts.WriteDeadline = cfg.ReadyTimeout`, so the typed denial resolves in ~2s; measured 10.000s -> 2.001s. Loopback server options are untouched.

Gate-blocker remediation (2026-06-10, same GREEN):

- Coverage: `TestExposureExternalDeniedByDefault` gains `fully declared nats surface still denied` (surface on + `TierExternal` + both TLS files -> still `ExposureDenied`), proving denied-by-default is terminal, not just a precondition check; `TestExposureUndeclaredSocketDenied` gains `unknown exposure mode` (`ExposureMode("public")` -> `ExposureUndeclared`), the malformed-posture case. `exposure.go` now has zero uncovered blocks in the full-corpus profile.
- Security: the loopback branch of `Config.exposure()` rejects any non-loopback socket host with `ExposureDenied` BEFORE any server is constructed — both `WebSocket.Host` (previously a runtime declaring `Loopback()` could bind the WebSocket listener on `0.0.0.0` while reporting a loopback posture) and the main `Host` (previously caught only post-start via `srv.Addr()` as `ExposureMismatch`, which briefly held a widened listener before the typed shutdown). Owning tests: `TestExposureLoopbackWebSocketBeyondLoopbackDenied` and `TestExposureLoopbackHostBeyondLoopbackDeniedBeforeBind`, the latter proving via the in-package construction hook that no server object exists when the denial fires.
- No-slop: the stale RED narration header was removed from `exposure_test.go`; the RED record lives in this doc only.

## Capability Proof Matrix

Over the exposure surface (real embedded NATS; JetStream KV round trips and outside-in TCP/CLI traffic):

- **allowed** -> `TestExposureInProcessDefaultServesWithoutSocket` (in-process KV round trip, no TCP endpoint) + `TestExposureLoopbackDeclaredBindsLoopback` (declared loopback bound, dialable, outside-in round trip).
- **denied-neighbor** -> carried `source_authority_cli_test.go` proofs (nats CLI denial under the declared `Loopback()` posture, output-parsed oracle) + `TestExposureInProcessConnectFailureTyped` (wrong lease over the in-process transport).
- **malformed** -> `TestExposureUndeclaredSocketDenied` (`unknown exposure mode` -> `ExposureUndeclared`; raw socket fields without a posture -> `ExposureUndeclared`) + `TestExposureLoopbackWebSocketBeyondLoopbackDenied` (non-loopback websocket host under loopback -> `ExposureDenied`) + `TestExposureLoopbackHostBeyondLoopbackDeniedBeforeBind` (non-loopback main host under loopback -> `ExposureDenied` before any server is built).
- **attributed failure** -> every denial asserts a typed adapter kind via `assertAdapter` (`ExposureUndeclared`, `ExposureDenied`, `ExposureMismatch`, `InProcessConnFailed`), never a bare error.
- **duplicate** / **stale revision** / **revoked lease** / **loop suppression** -> N/A for this slice: exposure is a declaration-time policy check with no lease/revision/idempotency lifecycle; that lifecycle is owned by `operator-jwt-authority` (`../plan/quality-v1.md`, slice 3).

GREEN evidence (all from `substrate/go` unless noted; final post-remediation state, re-executed 2026-06-10):

- `go test ./embednats -run 'TestExposure' -count=1 -v` -> exit 0: all 8 exposure tests PASS (12 subtests), `ok ... 2.049s` — no-socket in-process KV round trip, declared loopback bound+dialable on `127.0.0.1` with outside-in TCP traffic, 7/7 external denials typed `ExposureDenied` (including the fully declared surface), unknown mode + raw socket fields refused `ExposureUndeclared`, non-loopback websocket host and pre-bind non-loopback main host under loopback refused `ExposureDenied`, forced listener behind in-process posture refused `ExposureMismatch`, rejected in-process lease -> `InProcessConnFailed`.
- `go test ./embednats -count=1` -> exit 0: `ok ... 4.537s` (whole corpus on declared postures, assertions unchanged).
- `go test ./core ./embednats -count=1` -> exit 0: `ok core 0.093s`, `ok embednats 4.537s` — no core vocabulary touched.
- `bun run gate:parallel` -> exit 0: shuffled `go test ./... -count=1` all `ok` (contract, core, edge, embednats, frontend), `gate:parallel passed`.
- `bun run gate:fakes` -> exit 0: `gate:fakes passed` (mismatch test uses the in-package construction hook on the real nats-server; no fake type added).
- `bun run gate:coverage` -> exit 0: `contract 73.9%>=70%`, `core 81.7%>=78%`, `edge 82.8%>=78%`, `embednats 78.4%>=72%`, `frontend 100%>=95%`, `gate:coverage passed`.
- `bun run gate:scenarios` -> exit 0: `gate:scenarios passed`.

## Wrap-Up Verification (2026-06-10, full battery from repo root; Go from `substrate/go`)

- `bun run test` -> PASS: 85 pass, 0 fail, 427 expect() calls across 17 files.
- `bun run test:e2e` -> PASS: 1 pass, 0 fail, 16 expect() calls.
- `bun run typecheck` -> PASS: frontend, sdk, and orchestrator all clean via `bunx @typescript/native-preview` (exit 0).
- `bun run build` -> PASS: frontend vite build into `substrate/go/frontend/site` + sdk tsdown build (dist CJS/ESM ~64kB, .d.ts 32.48kB).
- `bun run pack:dry` -> PASS: `tinkabot-0.1.0.tgz`, 6 files, unpacked 194.45KB.
- `bun run schema:parity` -> PASS: contract tests pass; Go `contract`, `core`, `edge`, `embednats`, `frontend` all ok.
- `go test ./... -count=1` -> PASS uncached: `contract 0.051s`, `core 0.093s`, `edge 0.054s`, `embednats 4.537s`, `frontend 0.004s`.
- `bun run release:evidence` -> PASS: `release evidence check passed: 16 milestones over 11 spine steps`.
- `bun run gate:fakes` / `gate:parallel` / `gate:coverage` / `gate:scenarios` -> all PASS (parallel gate ran the full shuffled Go suite, all 5 packages ok; coverage floors met on all five layers).
- `git diff --check` -> PASS: no whitespace or conflict-marker errors.

Gate results: real-nats PASS, parallel-safety PASS, be-lazy PASS, coverage PASS, no-slop PASS, security PASS.

## Wrap-Up

`typed-exposure-posture` is complete. Exposure is a typed posture declared through the one harness seam (`start(t, cfg)`), not a port number: the in-process default serves real JetStream traffic with no TCP endpoint, loopback is an explicit declaration that carries the existing `nats` CLI outside-in proofs unchanged, and the external tier exists only as a typed, denied-by-default policy surface. All four standing failure families have a typed owner (`ExposureUndeclared`, `ExposureDenied`, `ExposureMismatch`, `InProcessConnFailed`), every denial is asserted by kind, the whole embednats corpus runs on declared postures with assertions unchanged, and all four standing gates plus the full release battery are green on the final tree. The next slice is `operator-jwt-authority` (quality-v1 slice 3).
