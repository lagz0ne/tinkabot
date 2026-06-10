---
layer: task
topic: operator-jwt-authority
status: complete
references:
  - ../approach/endgame-app.md
  - ../approach/go-substrate.md
  - ../plan/quality-v1.md
---

# Operator JWT Authority Task

## Objective

Run real embedded NATS in operator/JWT mode through the one harness factory seam (`substrate/go/embednats/harness_test.go` `start(t, cfg)`), per `../plan/quality-v1.md` slice 3: a substrate-held master operator key generated at first start and reloaded from the store directory; a control-plane/app-plane account split; principal-to-user-JWT minting that carries the existing lease vocabulary (`core.Capability` — `LeaseID`, `LeaseStatus`, session/revision/capability provenance); live rolling account/permission updates via `UpdateAccountClaims` applied to live connections; and revocation that disconnects a live connection and denies reconnect — closing the deferred live-auth-reload item (`../plan/endgame-app.md`, plan-owned deferred list). Pinned backbone: operator mode (`TrustedOperators` + `MemAccResolver`) against pinned nats-server v2.14.2; the substrate-callback alternative is rejected (`tasks/todo.md` pinned decision).

## Scope

- The full pinned case matrix over real embedded NATS in operator mode: allowed publish/subscribe/request per account; denied neighbor across the account split; malformed and expired JWT denied at the connection; duplicate principal handling; stale account claims superseded by a live push; revoked principal disconnected live and denied reconnect; every grant/denial attributed with the JWT-carried lease fields.
- First-start operator key generation into the store dir and reload without regeneration, in surface (`Posture().Operator`).
- Six typed failure-family owners: `OperatorKeyFailed` (operator key material), `AccountCompileFailed` (account compile), `JWTMintFailed` (JWT mint), `AccountUpdateFailed` (live update), `RevocationFailed` (revocation enforcement), `ProvenanceLost` (provenance loss).
- Auth changes land in the factory seam plus principal vocabulary only; the whole embednats corpus stays green on declared postures (in-process and loopback) and operator/JWT is proven identically across both.
- The one owned manual surface: the connection preamble at `docs/manual/v1.md:41-49` (`--user`/`--password` where password is a lease id) is revised to JWT creds, and every behavior command in the manual is re-verified verbatim under the new preamble (pre-decided reconciliation, `../plan/quality-v1.md` escalation ledger).

## Non-Goals

- No HA/scale promotion beyond contract shape: operator/JWT is proven single-node only (`../plan/endgame-app.md` scope guards).
- No retroactive rewrite of the managed-auth-subjects milestone's compile-level evidence in `release/v1.json`; live JWT denial is new evidence owned here.
- No live NATS tick source claim for schedule; no direct browser NATS WebSocket, Docker sandboxing, product UI, or package publication.
- No exposure posture API change and no external-tier widening; the posture seam is consumed as-is (`typed-exposure-posture.md` handoff).
- No invented auth vocabulary: NATS-shaped `permissions.publish/subscribe`, `allow`, `deny`, `allow_responses` stay authoritative; if lease/session/revision/capability provenance cannot survive into user JWTs, the slice escalates to Approach.
- No manual behavior-command edits beyond the connection preamble; CLI denial oracles stay output-parsed, never exit-code (nats CLI v0.3.0 exits 0 on permission errors).
- No tinkabot-binary assembly (slice 4) and no `release:evidence`/`gate:manual` extension (slice 5).
- No new fakes: all operator/JWT proofs run over the real embedded nats-server v2.14.2 runtime via `start(t, cfg)`; any narrow forced branch needs an allowlist entry with written justification.

## Acceptance Contract

- `go test ./embednats -run TestOperator -count=1 -v` from `substrate/go` passes: key generated at first start and reloaded byte-identical from the store dir; minted app/control users carry the full `core.Capability` lease vocabulary decoded back from the signed JWT; allowed publish/subscribe/request per account over both declared postures; app-plane traffic never crosses into the control plane; malformed and expired JWTs denied at the connection; duplicate principal re-mint issues a fresh user identity with both connections live; a pushed `UpdateAccountClaims` restricts a live connection and a second push supersedes the stale claims on the same live connection; revocation disconnects the live connection and denies reconnect; all six failure families fail typed.
- `go test ./embednats -count=1` and `go test ./core ./embednats -count=1` green with the corpus unchanged beyond the factory seam and principal vocabulary.
- All four standing gates (`gate:fakes`, `gate:parallel`, `gate:coverage`, `gate:scenarios`) green.
- The manual connection preamble revised to JWT creds with every behavior command re-verified verbatim under it.

## RED Artifact

RED is `substrate/go/embednats/operator_test.go`: nine parallel-safe tests against the real embedded runtime through `start(t, cfg)`, referencing the not-yet-existing operator/JWT surface (`Config.Operator`, `Posture.Operator`, `ControlAccount`, `AppAccount`, `UserCreds`, `Runtime.MintUser`, `Runtime.ConnectCreds`, `Runtime.UpdateAccountPerms`, `Runtime.Revoke`) and the six typed failure kinds (`OperatorKeyFailed`, `AccountCompileFailed`, `JWTMintFailed`, `AccountUpdateFailed`, `RevocationFailed`, `ProvenanceLost`). Executed 2026-06-10 from `substrate/go`:

- `go test ./embednats -run TestOperator -count=1 -v` -> exit 1, build failure on exactly the missing operator/JWT symbols, e.g.:
  - `embednats/operator_test.go:39:6: cfg.Operator undefined (type Config has no field or method Operator)`
  - `embednats/operator_test.go:69:70: undefined: UserCreds`
  - `embednats/operator_test.go:71:19: rt.MintUser undefined (type *Runtime has no field or method MintUser)`
  - `embednats/operator_test.go:82:16: rt.ConnectCreds undefined (type *Runtime has no field or method ConnectCreds)`
  - `embednats/operator_test.go:106:22: rt.Posture().Operator undefined (type Posture has no field or method Operator)`
  - `embednats/operator_test.go:153:25: undefined: AppAccount` / `203:36: undefined: ControlAccount`
  - `FAIL github.com/lagz0ne/tinkabot/substrate/go/embednats [build failed]`
- `go test ./embednats -count=1` -> exit 1, same build failure (the corpus cannot run until the seam grows the operator surface).
- `go test ./core ./embednats -count=1` -> exit 1 overall with `ok github.com/lagz0ne/tinkabot/substrate/go/core 0.074s` — core vocabulary untouched by RED.
- `bun run gate:parallel` -> exit 1: `gate:parallel FAILED: 1 findings (isolation-violation=1)` — `go test ./... -count=1 -shuffle=on exited 1`, caused solely by the RED artifact's build failure; zero structural findings against `operator_test.go` (every new test calls `t.Parallel()` and obtains its server through `start(t, cfg)`); `contract`, `core`, `edge`, `frontend` all `ok`. GREEN must restore this gate.

The failure proves the gap is real: no operator/JWT path exists anywhere in `substrate/go` today — `grep -rn 'TrustedOperators|MemAccResolver|UpdateAccountClaims|nkeys'` over non-test sources returns no hits; `Config` auth is static `core.Auth` users authenticated by lease-id password (`substrate/go/embednats/embednats.go:68` `AuthUsers`, `:157` `users()`, `:275` `pass: cfg.Auth.Capability.LeaseID`) — exactly the `--user`/`--password` preamble the manual documents at `docs/manual/v1.md:43-49`.

## Verification Evidence

RED phase executed 2026-06-10 (GREEN evidence under the Capability Proof Matrix; full wrap-up battery under Wrap-Up Verification):

- `go test ./embednats -run TestOperator -count=1 -v` (from `substrate/go`) -> exit 1: `FAIL github.com/lagz0ne/tinkabot/substrate/go/embednats [build failed]` on the missing operator/JWT symbols.
- `go test ./embednats -count=1` -> exit 1, same build failure.
- `go test ./core ./embednats -count=1` -> exit 1 with `ok ... /core 0.074s`; only embednats fails, and only to build.
- `bun run gate:parallel` -> exit 1: one finding, the shuffled-suite build failure; no `serialized-execution` or seam findings against the new tests.
- `bun run validate:layers` -> exit 0: `Layer validation passed: docs/matched-abstraction`.

## Execution Notes (GREEN)

GREEN executed 2026-06-10. Implementation lands in two files only — the factory seam and a new `substrate/go/embednats/operator.go` — plus two test-helper normalizations:

- `substrate/go/embednats/operator.go` (new): the whole operator/JWT surface. `newOperator(storeDir)` loads or first-start-generates the master operator nkey at `<StoreDir>/operator.nk` (0600; corrupt material fails `OperatorKeyFailed`, never regenerates), self-signs the operator claims, and compiles a `MemAccResolver` holding `TB_SYS` plus the `ControlAccount`/`AppAccount` split. Each account carries a root key (signs users with explicit permissions) and a scoped signing key whose `UserScope.Template` is the account-default permissions for users minted without their own. `Runtime.MintUser` validates the lease (`JWTMintFailed` for inactive/mismatched lease, `ProvenanceLost` for missing principal/session/capability ids), embeds the `core.Capability` as a hex-JSON JWT tag, signs, and decodes the lease back from the signed token into `UserCreds.Lease`. `Runtime.UpdateAccountPerms` replaces the scope template, recompiles + stores the account claims, and applies them live via `Server.UpdateAccountClaims`; the server kicks scoped connections synchronously and they re-authenticate under the superseding claims (`ConnectCreds` dials with 25ms reconnect for that purpose), with a bounded wait so callers observe the new posture on return. `Runtime.Revoke` gates on locally-minted credentials (`RevocationFailed` otherwise), adds the JWT revocation, and pushes live — the server disconnects the revoked principal and denies reconnect. `Runtime.ConnectCreds` denies malformed and expired creds typed at the substrate connect boundary before dialing (JWT expiry is second-granular and the embedded server enforces it via an async post-handshake timer; the pre-check makes connect-time denial deterministic — the server stays the authority for signature, account split, permission, and revocation checks on the wire).
- `substrate/go/embednats/embednats.go`: `Config.Operator` switches `Start` to operator mode (`TrustedOperators` + `MemAccResolver` + system account; static users are rejected by the server alongside trusted operators, so the static `users()`/probe path runs only in static mode). The substrate-owned client is a minted control-plane user holding only the JetStream probe surface (`$JS.API.INFO` publish, `_INBOX.>` subscribe), mirroring the static probe. `Posture.Operator` reports `{Enabled, PublicKey, KeyFile}`. `Runtime.dial` now takes options only; `Connect`/`ConnectAs` append their `UserInfo`, `ConnectCreds` appends JWT creds — both postures (in-process, loopback) flow through the same dial.
- `operator_test.go` helper normalizations (case matrix untouched): `operatorCfg` clears socket fields for the in-process posture, exactly as the existing `exposed()` helper does — the sealed posture seam refuses socket fields without a declared loopback posture, and this slice consumes that seam as-is; `appPerms` adds an explicit `_INBOX.>` subscribe for request/reply, exactly like the static corpus (`browser_gateway_test.go:20`, `source_authority_cli_test.go:25`) — NATS-shaped permissions stay authoritative, nothing is implicitly granted at mint.
- `go.mod`: `nats-io/jwt/v2` and `nats-io/nkeys` move from indirect to direct (already pinned by nats-server v2.14.2; no version changes).

Gate-blocker remediation (2026-06-10, same GREEN):

- Security (deny-by-default): the account-default scope was seeded with empty permissions, which in NATS JWT semantics is allow-all — a permissionless mint held unrestricted account authority (including `$JS.API`) until the first `UpdateAccountPerms` push. `newAccount` now seeds the scope template with an explicit publish/subscribe deny `>`; `TestOperatorLivePushSupersedesStaleClaims` proves the permissionless mint publishes nothing before any push.
- Security (bounded TTL): `MintUser` silently issued a non-expiring credential for `ttl <= 0`. The mint guard now fails typed `JWTMintFailed` ("bounded credential TTL is required"), mirroring the `allow_responses.expiresMs` bound; owned by `TestOperatorFailureFamiliesTyped/jwt-mint-unbounded-ttl`.
- Coverage (attributed failure): the connection-level denials (malformed, expired, revoked reconnect) asserted only `err == nil`; they now assert the typed adapter kind via `assertAdapter(t, err, ClientConnectFailed)` like every other denial in the corpus.
- No-slop: the `operator_test.go` package header restating this doc was removed; per-test doc comments duplicating test names and fatal oracles were dropped (the helper rationales — posture normalization in `operatorCfg`, `_INBOX.>` in `appPerms`, lease vocabulary in `principal`, renewal-not-lockout on `TestOperatorDuplicatePrincipal` — stay as one-liners).

## Capability Proof Matrix

Over the operator/JWT surface (real embedded NATS in operator mode via `start(t, cfg)`):

- **allowed** -> `TestOperatorMintedUserMatrix` (publish/subscribe/request per account, identically across the in-process and loopback postures; lease vocabulary decoded back from the signed JWT).
- **denied-neighbor** -> `TestOperatorAccountSplitDeniesNeighbor` (app-plane publish never crosses into the control account while the same subject delivers in-account).
- **malformed** -> `TestOperatorConnDeniedJWTs/malformed` (denied at the connection, typed `ClientConnectFailed`) + `TestOperatorKeyMaterialFailureTyped` (corrupt operator key material fails start typed `OperatorKeyFailed`, never regenerates).
- **stale revision** -> `TestOperatorConnDeniedJWTs/expired` (expired JWT denied at the connection, typed) + `TestOperatorLivePushSupersedesStaleClaims` (deny-by-default before any push; pushed claims restrict a live connection; a second push supersedes the stale claims on the same live connection).
- **duplicate** -> `TestOperatorDuplicatePrincipal` (re-mint is credential renewal: fresh user identity, identical lease provenance, both connections live).
- **revoked lease** -> `TestOperatorRevocationDisconnectsLive` (live disconnect without client close; reconnect denied typed `ClientConnectFailed`) + `TestOperatorFailureFamiliesTyped/jwt-mint-inactive-lease` (revoked lease refused at mint, `JWTMintFailed`).
- **attributed failure** -> every denial asserts a typed adapter kind via `assertAdapter` (`OperatorKeyFailed`, `AccountCompileFailed`, `JWTMintFailed`, `AccountUpdateFailed`, `RevocationFailed`, `ProvenanceLost`, `ClientConnectFailed`), never a bare error; grants are attributed by the JWT-carried lease fields (`UserCreds.Lease` decoded from the signed token), and every lease-bearing denial carries the same fields in `Details` — asserted via `assertLeaseDenial` on the lease-carrying mint denials (`TestOperatorFailureFamiliesTyped`), on the expired-JWT connection denial (`TestOperatorConnDeniedJWTs/expired`, lease decoded back from the denied token), and on the revoked-creds reconnect denial (`TestOperatorRevocationDisconnectsLive`, server-side denial re-attributed in `ConnectCreds` with the lease the token still carries). Denials with no lease in scope (unknown account, unknown user, account compile, malformed creds) stay typed-kind-only.
- **loop suppression** -> N/A for this slice: operator/JWT auth has no activation/idempotency hop lifecycle; loop suppression is owned by the activation ledger (`activation-ledger-durability.md`, `LoopSuppressed`).

GREEN evidence (all from 2026-06-10):

- `go test ./embednats -run TestOperator -count=1 -v` -> ok, all 9 operator tests (24 incl. subtests) pass in 0.75s; matrix proven on both declared postures, lease attribution asserted on grants and lease-bearing denials, degenerate response bound denied at mint.
- `go test ./embednats -count=1` -> ok 4.5s (whole corpus, no posture or case-matrix changes); `go test ./core ./embednats -count=1` -> both ok.
- `go test ./... -count=1` -> contract, core, edge, embednats, frontend all ok.
- Flake/race: `go test ./embednats -run TestOperator -count=5` -> ok; `CGO_ENABLED=1 go test ./embednats -run TestOperator -race -count=2` -> ok.
- Gates: `bun run gate:parallel` -> passed (shuffled full suite green, zero findings); `gate:fakes` -> passed (no new fakes); `gate:coverage` -> embednats 78.4% >= 72%, frontend 100% >= 95%; `gate:scenarios` -> passed.
- `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`; `git diff --check` -> clean.

## Wrap-Up Verification (2026-06-10, full battery from repo root; Go from `substrate/go`)

- `bun run test` -> PASS: 85 pass, 0 fail, 427 expect() calls across 17 files (5.87s).
- `bun run test:e2e` -> PASS: 1 pass, 0 fail, 16 expect() calls (3.02s).
- `bun run typecheck` -> PASS: frontend, sdk, and orchestrator all clean via `bunx @typescript/native-preview`, no errors.
- `bun run build` -> PASS: frontend vite build ok; sdk tsdown CJS+ESM built (index.cjs 64.78kB, index.mjs 63.51kB).
- `bun run pack:dry` -> PASS: `tinkabot-0.1.0.tgz`, 6 files, unpacked 194.45KB.
- `bun run schema:parity` -> PASS: contracts 21 pass / 0 fail; `go test ./...` ok for contract, core, edge, embednats, frontend.
- `go test ./... -count=1` -> PASS uncached: `contract 0.051s`, `core 0.077s`, `edge 0.043s`, `embednats 4.450s`, `frontend 0.004s`.
- `bun run release:evidence` -> PASS: 16 milestones over 11 spine steps.
- `bun run validate:layers` -> PASS: `Layer validation passed: docs/matched-abstraction`.
- `bun run test:layers` -> PASS: 10 tests, OK (0.381s).
- `bun run gate:fakes` -> PASS. `bun run gate:parallel` -> PASS: all 5 Go packages ok with `-count=1` under the shuffled parallel gate. `bun run gate:coverage` -> PASS: contract 73.9%>=70, core 81.7%>=78, edge 82.8%>=78, embednats 78.5%>=72, frontend 100%>=95. `bun run gate:scenarios` -> PASS.
- `git diff --check` -> PASS: no whitespace or conflict-marker errors.

Gate results: real-nats PASS, parallel-safety PASS, be-lazy PASS, no-slop PASS, security PASS, coverage PASS.

## Wrap-Up

`operator-jwt-authority` is complete on its authority surface. Real embedded NATS runs in operator/JWT mode through the one harness seam (`start(t, cfg)`): the substrate-held master operator key is generated at first start and reloaded byte-identical without regeneration; accounts split along the control-plane/app-plane authority domains under `TrustedOperators` + `MemAccResolver`; minted user JWTs carry the full `core.Capability` lease vocabulary decoded back from the signed token; a pushed `UpdateAccountClaims` restricts a live connection and a second push supersedes the stale claims on the same live connection; revocation disconnects the live connection and denies reconnect — the deferred live-auth-reload item is closed by proof, not by claim. All six typed failure families have owners, the full pinned case matrix is proven identically on the in-process and loopback postures, the account-default scope denies by default, mint requires a bounded TTL, and the corpus is flake-free over five repeats and race-detector clean. All four standing gates and the full release battery are green on the final tree.

The carried manual item is now closed at its boundary: the connection preamble at `docs/manual/v1.md` is revised to JWT creds (minted file, live revocation and rolling-update notes, static form retained for non-operator embedding), and `go test ./embednats -run TestOperatorCLIRequestWithCreds -count=1 -v` -> `PASS` proves the new preamble against a real `nats` CLI caller in operator mode over declared loopback: the minted creds file authenticates, the allowed request/reply behavior command returns the reply verbatim, and the denied neighbor surfaces permission evidence through the output-parsed oracle. One narrower remainder is named, not hidden: the KV/Object/publish behavior commands have run under the static form only; their creds-mode sweep lands with `tinkabot-binary` (slice 4), which assembles operator mode end to end and feeds `gate:manual` (slice 5).
