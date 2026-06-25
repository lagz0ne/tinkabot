---
layer: task
topic: tinkabot-binary
status: complete
references:
  - ../approach/endgame-app.md
  - ../approach/go-substrate.md
  - ../plan/quality-v1.md
  - ./operator-jwt-authority.md
  - ./typed-exposure-posture.md
  - ./script-materializer-loop.md
---

# Tinkabot Binary Task

## Objective

Assemble the v1 product entry surface per `../plan/quality-v1.md` slice 4: a single Go binary (`substrate/go/tinkabot` assembly package plus a thin `substrate/go/cmd/tinkabot` main) that embeds NATS in operator/JWT mode, serves the embedded frontend shell under the proven scope and policy headers, and runs the script materializer loop through the declared exposure posture. It starts from an empty store directory and materializes the operator key, store state, and manual-role creds at first start; restarts from existing state without regeneration; runs the manual's script, trigger, observation, and denial flows through the real `nats` CLI in creds mode — closing the carried KV/Object/publish behavior-commands creds-mode sweep from `./operator-jwt-authority.md` (carried remainder) — drains cleanly on shutdown; and adds the manual's new "starting the binary" section to `docs/manual/v1.md`.

## Scope

- Assembly only: consume `embednats.Exposure`/`Posture` (typed exposure), `Config.Operator`/`MintUser`/`ConnectCreds`/`Revoke`/`ControlAccount`/`AppAccount`/`UserCreds` (operator/JWT authority), `frontend.Files`/`frontend.Index` (embedded shell), and `NewSourceRouter`/`NewScriptLoop`/KV script-material stores (materializer loop). Any capability those slices did not prove returns to the owning slice.
- Lifecycle: `Start(Config)` over store dir + declared exposure + operator mode + shell addr; `App.Posture()` reporting NATS, shell, and wiring surfaces; idempotent `Stop(ctx)` that drains live connections and stops the shell.
- First-start materialization (operator key, JetStream store, role creds files for caller/observer/author) versus reload without regeneration.
- Embedded shell served on loopback HTTP with the proven policy headers (`Service-Worker-Allowed` bound to a narrow `/__tinkabot_session/` scope, `Cache-Control: no-store`, `X-Tinkabot-Worker-Rev`) per the browser-isolation proof vocabulary (`edge.CheckServiceWorkerSetup`).
- Manual flows over the real `nats` CLI in creds mode against the running binary: script define, request/reply trigger, projection/artifact/manifest observation, denied caller ledger write, denied observer material/Object writes, duplicate no-rerun, the KV/Object/publish behavior-commands sweep, and revoked-lease denial — all denial oracles output-parsed, never exit-code.
- Five typed failure families owned here: `StartupMaterializationFailed`, `FrontendServeFailed`, `WiringMismatch` (declared posture versus served surface), `ManualDivergence` (`CheckManual` against `docs/manual/v1.md`), `ShutdownFailed`.

## Non-Goals

- No `release:evidence` extension, `gate:manual` implementation, or synthetic gate-overclaim checks — `quality-release` (slice 5) owns those; this slice only produces the creds-mode evidence that feeds them.
- No exposure posture API changes or new tiers; no operator/JWT surface changes beyond consumption; no materializer loop behavior changes — gaps return to `typed-exposure-posture`, `operator-jwt-authority`, and `script-materializer-loop` respectively.
- No manual behavior-command rewrites: only the new "starting the binary" section lands; existing commands run verbatim (the connection preamble was revised by slice 3).
- No multi-node/HA runtime claims (contract shape only), no live wall-clock scheduler tick source, no product UI rendering beyond the embedded shell, no direct browser NATS WebSocket, no Docker sandboxing, no package publication (pack shape may land; publication stays named deferred) — `../plan/endgame-app.md` deferred list and scope guards.
- No new fakes: all proofs over the real embedded NATS runtime with per-test isolated servers and store dirs and `t.Parallel()`; any forced branch needs a fakes-allowlist entry with written justification.

## Acceptance Contract

- `go test ./tinkabot -count=1 -v` from `substrate/go` passes: empty-dir first start materializes operator key and role creds; restart reloads byte-identical without regeneration; embedded shell served with the proven scope and policy headers and live embedded assets; declared loopback posture is the served NATS/shell surface with full wiring posture; clean idempotent drain shutdown; all five failure families fail typed.
- `go test ./tinkabot -run TestBinaryManual -count=1 -v` from `substrate/go` passes: the manual flows over the real `nats` CLI in creds mode, including the carried KV/Object/publish behavior-commands sweep, denied caller and observer writes (output-parsed oracle), duplicate no-rerun, and revoked-lease denial.
- `go build ./cmd/tinkabot` from `substrate/go` compiles the binary entry point.
- `go test ./... -count=1` from `substrate/go` green with the new package; all four standing gates (`gate:fakes`, `gate:parallel`, `gate:coverage`, `gate:scenarios`) green.
- `docs/manual/v1.md` gains the "starting the binary" section and `CheckManual` validates it against the live posture (`TestBinaryManualStartingSection`).

## RED Artifact

RED is the test-only package `substrate/go/tinkabot` (`binary_test.go`, `manual_test.go`): eight parallel-safe tests against the real embedded runtime through one `boot(t, cfg)` harness seam, referencing the not-yet-existing assembly surface (`Config`, `Start`, `App`, `Posture`/`ShellPosture`/`Wiring`, `RoleCaller`/`RoleObserver`/`RoleAuthor`, `Creds`/`CredsFile`, `Runtime`, `Stop`, `CheckManual`) and the five typed failure kinds (`StartupMaterializationFailed`, `FrontendServeFailed`, `WiringMismatch`, `ManualDivergence`, `ShutdownFailed`). No `cmd/tinkabot` package exists. Executed 2026-06-10 from `substrate/go`:

- `go test ./tinkabot -count=1 -v` -> exit 1, build failure on exactly the missing assembly symbols, e.g.:
  - `tinkabot/binary_test.go:32:29: undefined: Config`
  - `tinkabot/binary_test.go:32:39: undefined: App`
  - `tinkabot/binary_test.go:34:14: undefined: Start`
  - `tinkabot/binary_test.go:54:47: undefined: Kind` / `:56:9: undefined: Error`
  - `tinkabot/binary_test.go:84:32: undefined: RoleCaller` / `:84:44: undefined: RoleObserver`
  - `FAIL github.com/lagz0ne/tinkabot/substrate/go/tinkabot [build failed]`
- `go test ./tinkabot -count=1 -gcflags=-e` (full error list) -> 37 errors, every one `undefined:` on the assembly surface — `Config`, `App`, `Start`, `Kind`, `Error`, `CheckManual`, `RoleCaller`/`RoleObserver`/`RoleAuthor`, `StartupMaterializationFailed`, `FrontendServeFailed`, `WiringMismatch`, `ManualDivergence`, `ShutdownFailed` — zero syntax errors, zero misuse of consumed packages.
- `go test ./tinkabot -run TestBinaryManual -count=1 -v` -> exit 1, same build failure: the manual's creds-mode flow has no binary to run against.
- `go build ./cmd/tinkabot` -> exit 1: `stat substrate/go/cmd/tinkabot: directory not found` — no product entry point exists.
- `go test ./... -count=1` -> exit 1 overall with `ok` for `contract`, `core`, `edge`, `embednats`, `frontend` — the corpus is green everywhere except the RED artifact's build failure.

The failure proves the gap is real: operator/JWT (`embednats/operator.go`), typed exposure (`embednats/exposure.go`), the embedded shell (`frontend/`), and the materializer loop (`embednats/script_materializer.go`) each exist as separately proven packages, but nothing in `substrate/go` assembles them into a startable, restartable, CLI-operable binary, and the manual's KV/Object/publish behavior commands have only ever run under static auth, never creds mode (`./operator-jwt-authority.md` carried remainder).

## GREEN Execution Notes

Implemented 2026-06-10 as `substrate/go/tinkabot/tinkabot.go` (assembly) plus `substrate/go/cmd/tinkabot/main.go` (thin entry: flags, posture print, signal-driven stop, split `run()` for testability):

- **Assembly only.** `Start(Config)` consumes `embednats.Start` (operator mode, declared `Exposure`), `MintUser`/`ConnectCreds`/`Revoke` for role and internal-service principals, `NewSourceRouter` + `NewScriptLoop` + `LocalScriptRunner` + KV stores for the loop, `frontend.Files`/`Index` for the shell, and `edge.CheckServiceWorkerSetup` as the policy-header authority (the binary invents no header, auth, or loop vocabulary).
- **Consumed-surface plumbing, not behavior:** the loop's store constructors were hardwired to the static-auth `rt.Connect`/`ConnectAs` path, which cannot exist in operator mode. Added conn-injection variants `OpenKVLedgerStore`/`OpenKVScriptStore`/`OpenKVMaterialStore` in `embednats` with the existing `New*` constructors delegating to them — identical code paths, zero behavior change, embednats suite untouched and green.
- **Materialization:** first start grows `operator.nk` (embednats-owned), `caller.creds`/`observer.creds`/`author.creds` (0600, minted in `TB_APP` with enumerated NATS-shaped permissions mirroring the proven static-auth sets), the config KV bucket, upload Object bucket, events stream, and the ledger/script/material stores. Restart reloads the operator key byte-identical and mints fresh creds (account identity is ephemeral by the operator slice's design).
- **Drain:** `Stop` stops the route and loop, then revokes the process's minted role creds — minted authority dies with the process; the kicked credentialed clients observe denial on reconnect and abort — with a 150ms bounded grace before server shutdown so the second auth round trip lands while the listener is up. Clean stop is idempotent; a failed stop (expired context) returns typed `ShutdownFailed` and stays retryable.
- **Manual divergence found and owned:** the manual's subject-message trigger said `nats publish --hdr ...`, but nats CLI v0.3.0 only accepts `-H/--header` (`unknown long flag '--hdr'` — the command had never been CLI-executed; the release proof published via Go). Corrected to `-H` in `docs/manual/v1.md` and the test transcription — a ManualDivergence fix this slice owns, not a behavior rewrite.
- **Manual section:** `docs/manual/v1.md` gained `## Starting the binary` (binary invocation, first-start vs restart materialization, posture print, creds-mode connection, shell scope/headers, drain semantics). `CheckManual` validates the document against the live posture: section heading, operator key basename, all three role creds names, and the served service-worker scope.
- **Known wart:** `go build ./cmd/tinkabot` from `substrate/go` compiles but cannot write its output artifact — the default output name `tinkabot` collides with the `./tinkabot` package directory. `go build -o /tmp/tinkabot-bin ./cmd/tinkabot` proves the entry point builds and produces the executable; `go vet ./cmd/...` and the corpus run compile it too.
- **Gates:** `coverage-thresholds.json` gained `tinkabot: 75` (measured 81.2%) and `cmd: 65` (measured 70.8%). No new fakes. The `boot(t, cfg)` seam stays the package's one construction point; `cmd` tests go through `run()`.

## Capability Proof Matrix

Over the running binary's NATS surface (real `nats` CLI in creds mode against the assembled binary via `boot(t, cfg)`):

- **allowed** -> `TestBinaryManual`: author script define over the CLI (`kv put` into the script bucket), request/reply trigger answered `accepted`, observer projection/manifest/artifact reads, and the carried KV/Object/publish behavior-commands creds-mode sweep (subject-message `-H`, KV change, Object change, stream publish).
- **denied-neighbor** -> `TestBinaryManual`: caller denied on the ledger KV write, observer denied on material-KV and object-chunk writes, caller denied on `tb.internal.escape` — all through the output-parsed oracle `wantDenied` (nats CLI v0.3.0 exits 0 on permission errors, so denial evidence is output text, never exit code).
- **malformed** -> N/A for this slice: malformed trigger frames and script-record payloads are owned by the source router and materializer loop (`./script-materializer-loop.md`: strict script-record decode, framed-stdio protocol errors), and malformed credentials are denied at the connection by `operator-jwt-authority` (`TestOperatorConnDeniedJWTs/malformed`). The assembly invents no decoding surface to malform.
- **stale revision** -> N/A for this slice: script-revision matching is owned by the materializer loop (`./script-materializer-loop.md` revision-mismatch case) and expired-JWT staleness at the connection by `operator-jwt-authority` (`TestOperatorConnDeniedJWTs/expired`); the binary only wires the proven revision through `Wiring.ScriptRevision`.
- **duplicate** -> `TestBinaryManual`: the same request id over the CLI answers `duplicate` and the materialized projection stays byte-identical (no rerun).
- **revoked lease** -> `TestBinaryManual`: `Revoke` on the caller principal denies the next CLI request (output-parsed); `TestBinaryDrainShutdown` proves drain-by-revocation closes live credentialed connections at stop.
- **attributed failure** -> `TestBinaryFailureFamiliesTyped`: each of the five owned families asserts its typed kind via `assertKind` (`StartupMaterializationFailed`, `FrontendServeFailed`, `WiringMismatch`, `ManualDivergence`, `ShutdownFailed`), never a bare error; below the assembly, NATS-visible denials stay attributed by the slices that own them (`operator-jwt-authority` lease-carrying denials, materializer-loop typed run outcomes).
- **loop suppression** -> N/A for this slice: activation hop/loop lifecycle is owned by the activation ledger (`activation-ledger-durability.md`, `LoopSuppressed`); the binary consumes `core.NewDurableLedger` as-is and adds no hop semantics.

## Verification Evidence

RED (executed first):

- `go test ./tinkabot -count=1 -v` (from `substrate/go`) -> exit 1, `[build failed]` on exactly the missing assembly symbols (37 `undefined:` errors under `-gcflags=-e`, zero syntax errors). The contracted RED failure.
- `go test ./tinkabot -run TestBinaryManual -count=1 -v` -> exit 1, same build failure.
- `go build ./cmd/tinkabot` -> exit 1, `directory not found`: no binary entry point exists yet.
- `go test ./... -count=1` -> `ok` for `contract`, `core`, `edge`, `embednats`, `frontend`; only the RED package fails. The existing corpus is untouched.

GREEN (executed 2026-06-10 from `substrate/go` unless noted):

- `go test ./tinkabot -count=1 -v` -> PASS: all 8 tests (`FirstStartMaterializes`, `RestartReloadsWithoutRegeneration`, `ServesEmbeddedShell`, `PostureMatchesServedSurface`, `DrainShutdown`, `FailureFamiliesTyped` with all five typed kinds, `ManualStartingSection`, `Manual`), `ok ... 4.534s`.
- `go test ./tinkabot -run TestBinaryManual -count=1 -v` -> PASS (4.46s): real `nats` CLI in creds mode — author script define, `accepted` trigger reply, denied caller ledger write, projection/manifest/artifact observation, denied observer material-KV and object-chunk writes, `duplicate` reply with byte-identical projection, the carried KV/Object/publish behavior-commands creds-mode sweep (subject-message `-H`, KV change, Object change, stream publish, denied `tb.internal.escape`), and revoked-lease denial — all denial oracles output-parsed.
- `go build -o /tmp/tinkabot-bin ./cmd/tinkabot` -> exit 0 (see Known wart for the verbatim form's artifact-name collision).
- `go test ./... -count=1` -> `ok` for all seven packages including `cmd/tinkabot` and `tinkabot`.
- `go test ./tinkabot ./cmd/... -count=2 -shuffle=on` -> ok twice (drain/restart timing stable under shuffle).
- Gates: `gate:fakes` passed; `gate:coverage` passed (cmd 70.8%>=65, tinkabot 81.2%>=75, all prior layers unchanged-green); `gate:parallel` passed (full shuffled corpus); `gate:scenarios` passed.

## Wrap-Up Verification (2026-06-10, full battery from repo root; Go from `substrate/go`)

- `bun run test` -> PASS: 85 pass, 0 fail, 427 expect() calls across 17 files.
- `bun run test:e2e` -> PASS: 1 pass, 0 fail, 16 expect() calls.
- `bun run typecheck` -> PASS: frontend, sdk, and orchestrator all clean via `bunx @typescript/native-preview`.
- `bun run build` -> PASS: frontend vite build + sdk tsdown (CJS+ESM, dist 4 files).
- `bun run pack:dry` -> PASS: `tinkabot-0.1.0.tgz`, 6 files, 194.45KB unpacked.
- `bun run schema:parity` -> PASS: contracts 21 pass / 0 fail; `go test ./...` all 7 packages ok.
- `go test ./... -count=1` -> PASS uncached: `cmd/tinkabot`, `contract`, `core`, `edge`, `embednats`, `frontend`, `tinkabot` — 7/7 ok.
- `bun run release:evidence` -> PASS: 16 milestones over 11 spine steps.
- `bun run gate:fakes` -> PASS. `bun run gate:parallel` -> PASS: all 7 Go packages ok under the shuffled parallel gate. `bun run gate:coverage` -> PASS: cmd 70.8%>=65, contract 73.9%>=70, core 81.7%>=78, edge 82.8%>=78, embednats 78.6%>=72, frontend 100%>=95, tinkabot 82.3%>=75. `bun run gate:scenarios` -> PASS.
- `git diff --check` -> PASS: exit 0, no whitespace or conflict-marker errors.

Gate results: real-nats PASS, parallel-safety PASS, coverage PASS, be-lazy PASS, security PASS, no-slop PASS.

## Wrap-Up

`tinkabot-binary` is complete on its assembly surface. The v1 product entry surface exists and is proven: `substrate/go/tinkabot` assembles the four separately proven slices — operator/JWT authority, typed exposure posture, the embedded frontend shell, and the script materializer loop — into one startable, restartable, CLI-operable binary with a thin `substrate/go/cmd/tinkabot` entry point that compiles and produces the executable (`go build -o /tmp/tinkabot-bin ./cmd/tinkabot` exits 0; the verbatim form's output-name collision with the package directory is the named Known wart). First start from an empty store directory materializes the operator key and the caller/observer/author role creds; restart reloads the operator key byte-identical without regeneration. The embedded shell is served on loopback with the proven policy headers (`Service-Worker-Allowed` bound to `/__tinkabot_session/`, `Cache-Control: no-store`, `X-Tinkabot-Worker-Rev`) and live embedded assets, and the declared loopback posture is the served NATS/shell surface with full wiring. Shutdown drains by revoking the process's minted role creds — live credentialed connections close — and a second `Stop` is idempotent; all five owned failure families (`StartupMaterializationFailed`, `FrontendServeFailed`, `WiringMismatch`, `ManualDivergence`, `ShutdownFailed`) fail typed.

The carried remainder from `./operator-jwt-authority.md` is closed: `TestBinaryManual` runs the manual's flows over the real `nats` CLI in creds mode against the running binary — script define, `accepted` trigger reply, projection/manifest/artifact observation, the full KV/Object/publish behavior-commands creds-mode sweep, denied caller and observer writes, denied `tb.internal.escape`, `duplicate` no-rerun with a byte-identical projection, and revoked-lease denial — every denial oracle output-parsed, never exit-code. `docs/manual/v1.md` carries the new "Starting the binary" section and `CheckManual` validates it against the live posture; the one manual divergence found (`--hdr` vs `-H`) is corrected and owned here. All eight package tests pass, the whole 7-package corpus is green and shuffle-stable, all four standing gates and all six slice gates pass, and the full release battery is green on the final tree. `quality-release` (slice 5) is the next resume point and consumes this slice's creds-mode evidence for `gate:manual` and the `release:evidence` extension.
