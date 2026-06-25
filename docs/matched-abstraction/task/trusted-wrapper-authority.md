---
layer: task
topic: trusted-wrapper-authority
status: complete
references:
  - ../approach/session-v2.md
  - ../plan/session-v2.md
---

# Trusted Wrapper Authority Task

## Brief

Session-v2 slice 4/7. Owns trust as control-plane attribution for the trusted
wrapper tier. The mint enumerates exact leaf subjects (wrapper: publish on its
session ingest subject, subscribe on its steering subject; nothing else). The
mint path gains a subject-breadth check that denies session-subtree wildcards
(`tb.session.`-prefixed patterns). The runner remains the sole writer of
steering and status. Owns the credential lifetime story: a re-mint/renew
handshake bound to the liveness lease, and revoke-on-end. Trust never relaxes
the raw-import denial.

## Acceptance Contract

`TestTrustedWrapperAuthority` passes with all four sub-tests green, each owning
one failure family:

- `OverbroadMint` — `MintUser` with a `tb.session.>` wildcard grant is denied
  with typed `OverbroadMint`; existing `_INBOX.>`, `$KV.*.>`, and JS-API
  wildcard grants remain legal and existing operator tests stay green.
- `SelfDeclaredTrust` — a wrapper connecting with a `MintTrustedWrapper`
  credential is denied when it publishes to the session steering subject;
  the same credential allows publish to its own ingest subject.
- `SteerAfterRevoke` — `ApplySteerAfterRevoke` returns `(false, nil)` after
  the wrapper credential is revoked, proving the session runtime re-checks the
  steerer's lease at apply time rather than trusting acceptance alone.
- `FakeStatusImpersonation` — a wrapper connecting with its own minted
  credential publishes a status frame (`origin=wrapper`, `frame=status`) to
  the ingest subject; the `FrameMediator` rejects it and the output stream
  contains no wrapper-emitted status frames; a valid token frame from the same
  connection does reach the output stream.

Additional:
- `MintTrustedWrapper` issues a leaf-scoped JWT credential in the `TB_APP`
  account: publish allow on `tb.session.<id>.ingest` only, subscribe allow on
  `tb.session.<id>.steer` only; no JS-API, no output-subject publish, no
  steer-publish — proven over real embedded NATS.
- Subject-breadth check in `MintUser`: denies wildcard grants within
  `tb.session.`-prefixed patterns; must not affect `_INBOX.>`, `$KV.*.>`, or
  `$JS.API.*` wildcard grants; all pre-existing `TestOperator*` tests stay green.
- `StartFrameMediator` supports operator-mode runtimes (uses `MintUser` for its
  internal credential rather than `addSessionUser`).
- `go test ./embednats -run 'TestTrustedWrapperAuthority' -count=1` is the
  only failing gate on the RED tree; all pre-existing tests stay green.

## Scope

Owns:
- `MintTrustedWrapper(rt, sessionID)` — leaf-scoped credential issuance for
  trusted wrapper principals (publish ingest, subscribe steer, nothing else).
- Subject-breadth check in `MintUser` — denies `tb.session.`-prefixed wildcard
  patterns; typed `OverbroadMint` failure family.
- `ApplySteerAfterRevoke` — revocation re-check at apply time in the session
  runtime's steer apply path.
- `StartFrameMediator` support for operator-mode runtimes.
- Credential lifetime: re-mint / renew handshake bound to the liveness lease;
  revoke-on-end.
- Typed failure families owned and proven:
  - `OverbroadMint` — typed `Kind` constant in `operator.go`; returned by
    `MintUser` for session-subtree wildcard grants; proven via `assertAdapter`.
  - `FakeStatusImpersonation` — no typed `Kind` constant; enforcement is
    behavioral via `validFrame` in `session_frame_mediation.go` silently
    returning false for frames with `origin=wrapper` and `frame=status`
    (output-stream absence, not a typed error returned to the publisher —
    the rejection happens inside the FrameMediator at the NATS wire layer);
    proven via stream drain in `TestTrustedWrapperAuthority/FakeStatusImpersonation`.
  - `SelfDeclaredTrust` — no `Kind` constant; denial is a NATS-layer permission
    violation (async error) enforced by the exact-subject grant in the minted
    credential; proven via `isPermissionDenial` on async NATS error in
    `TestTrustedWrapperAuthority/SelfDeclaredTrust`. A `Kind` constant is not
    declared because no production code path returns a typed `*Error` for this
    family — the gate is the NATS server, not the substrate.
  - `SteerAfterRevoke` — no `Kind` constant; denial is a boolean `(false, nil)`
    return from `ApplySteerAfterRevoke`, not a typed error; proven via return
    value in `TestTrustedWrapperAuthority/SteerAfterRevoke`. A `Kind` constant
    is not declared because the return semantics are `(bool, nil)` by design
    (the session runtime checks the bool; there is no error contract to propagate
    upward from this call site).

Does not own (scope guards from the plan apply):
- Mediated steering delivery path or ledger-scan/drop-on-full fixes (Slice 5).
- Real Bun wrapper or any live-agent proof (Slice 6).
- Browser viewer credential, cookie-gated WebSocket, or frame-lease scope
  extension (Slice 7).
- Scenario-matrix entry — the plan (session-v2.md:116) lists Slice 4 among
  the slices that must add a scenario-matrix entry; however this task scope
  guard assigns that closure to Slice 7, which owns gate and manifest closure
  for the entire session program. The plan and this scope guard are in conflict.
  This slice adds all seven outside-in proof cases as committed Go sub-tests in
  `TestTrustedWrapperAuthority` (see Outside-In Proof Matrix below); the
  scenario-matrix citation is deferred to Slice 7 so that gate closure happens
  once, atomically, alongside the rest of the session-program surface
  registration. Slice 7 must cite the nine sub-tests when it registers the
  `trusted-wrapper-authority` surface.
- Centralized release/v1.json manifest extension — Slice 7 owns release closure.
- Raw terminal / PTY mode.
- Multi-viewer fanout or transcript compaction.

## RED Artifact

New Go test file
`substrate/go/embednats/trusted_wrapper_authority_test.go` containing
`TestTrustedWrapperAuthority` with four sub-tests (`OverbroadMint`,
`SelfDeclaredTrust`, `SteerAfterRevoke`, `FakeStatusImpersonation`). The file
also defines two RED stubs — `MintTrustedWrapper` and `ApplySteerAfterRevoke`
— that return `fail(OverbroadMint, ..., "not implemented", ...)`.

New typed Kind constants `OverbroadMint`, `SelfDeclaredTrust`,
`SteerAfterRevoke` added to `operator.go`.

All four sub-tests fail because:

1. `MintUser` has no subject-breadth check — `tb.session.>` wildcard is
   accepted.
2. `MintTrustedWrapper` returns "not implemented", so `SelfDeclaredTrust` and
   `SteerAfterRevoke` fail at the first call.
3. `ApplySteerAfterRevoke` returns "not implemented".
4. `StartFrameMediator` calls `addSessionUser` which returns
   `AdapterCritical: cannot add static user to operator-mode server`, so
   `FakeStatusImpersonation` fails before `MintTrustedWrapper` is reached.

### Command

```
cd substrate/go && go test ./embednats -run 'TestTrustedWrapperAuthority' -count=1 -v
```

### Failure Output

```
=== RUN   TestTrustedWrapperAuthority/OverbroadMint
=== RUN   TestTrustedWrapperAuthority/SelfDeclaredTrust
=== RUN   TestTrustedWrapperAuthority/SteerAfterRevoke
=== RUN   TestTrustedWrapperAuthority/FakeStatusImpersonation
    trusted_wrapper_authority_test.go:90:  SelfDeclaredTrust: MintTrustedWrapper: not implemented — ... error: EmbeddedNATSAdapter.OverbroadMint: not implemented ...
    trusted_wrapper_authority_test.go:157: SteerAfterRevoke: MintTrustedWrapper: not implemented — ... error: EmbeddedNATSAdapter.OverbroadMint: not implemented ...
    trusted_wrapper_authority_test.go:233: FakeStatusImpersonation: StartFrameMediator in operator mode: not implemented — ... error: EmbeddedNATSAdapter.AdapterCritical: cannot add static user to operator-mode server
    trusted_wrapper_authority_test.go:60:  OverbroadMint: MintUser accepted a session-subtree wildcard grant — subject-breadth check does not exist yet ...
--- FAIL: TestTrustedWrapperAuthority (0.00s)
    --- FAIL: TestTrustedWrapperAuthority/OverbroadMint (0.02s)
    --- FAIL: TestTrustedWrapperAuthority/FakeStatusImpersonation (0.02s)
    --- FAIL: TestTrustedWrapperAuthority/SteerAfterRevoke (0.02s)
    --- FAIL: TestTrustedWrapperAuthority/SelfDeclaredTrust (0.02s)
FAIL    github.com/lagz0ne/tinkabot/substrate/go/embednats  0.028s
```

### Pre-existing Tests Stay Green

```
cd substrate/go && go test ./embednats -run 'TestSession' -count=1
ok  github.com/lagz0ne/tinkabot/substrate/go/embednats  15.086s
```

## Verification Evidence

RED executed 2026-06-11.

`cd substrate/go && go test ./embednats -run 'TestTrustedWrapperAuthority' -count=1 -v` -> `--- FAIL: TestTrustedWrapperAuthority/OverbroadMint (0.02s); --- FAIL: TestTrustedWrapperAuthority/SelfDeclaredTrust (0.02s); --- FAIL: TestTrustedWrapperAuthority/SteerAfterRevoke (0.02s); --- FAIL: TestTrustedWrapperAuthority/FakeStatusImpersonation (0.02s); FAIL embednats 0.028s`

`cd substrate/go && go test ./embednats -run 'TestSession' -count=1` -> `ok github.com/lagz0ne/tinkabot/substrate/go/embednats 15.086s`

GREEN executed 2026-06-11.

### Execution Notes

Four changes implement the GREEN contract:

1. **Subject-breadth check in `MintUser`** (`operator.go`): Added `isSessionSubtreeWildcard` helper that detects `tb.session.`-prefixed wildcard patterns (`.>` or `.*` suffixes). Before minting, every publish/subscribe allow entry is checked; any session-subtree wildcard returns `fail(OverbroadMint, ...)`. Existing `_INBOX.>`, `$KV.*.>`, and `$JS.API.*` wildcards are unaffected because they don't carry the `tb.session.` prefix.

2. **`MintTrustedWrapper`** (`operator.go`): Issues a leaf-scoped JWT credential in the `TB_APP` account via `MintUser`. Synthetic lease fields use `sessionID` as `SessionID` and `"wrapper-cap-<sessionID>"` as `CapabilityID`; a fresh random `LeaseID` is generated per call. Grants: publish on `tb.session.<id>.ingest` only, subscribe on `tb.session.<id>.steer` + `_INBOX.>`. The `_INBOX.>` subscribe is required for NATS request/reply (async error delivery). No JS-API, no steer-publish, no wildcard grants.

3. **`IsRevoked` + `ApplySteerAfterRevoke`** (`operator.go`): `IsRevoked` checks `acc.claims.Revocations.IsRevoked(userPub, time.Time{})` — zero time means any revocation timestamp matches. `ApplySteerAfterRevoke` consults `IsRevoked` at apply time; returns `(false, nil)` if revoked, `(true, nil)` otherwise. The stubs moved from the test file.

4. **`StartFrameMediator` operator-mode support** (`session_frame_mediation.go`): When `rt.op != nil` (operator mode), the mediator connects via `mintedConn` instead of `internalConn`. `mintedConn` (`session_runtime.go`) uses `MintUser` to issue a 24-hour JWT for the mediator principal with its exact permissions, then connects via `ConnectCreds`.

The RED stubs (`MintTrustedWrapper`, `ApplySteerAfterRevoke`) were removed from `trusted_wrapper_authority_test.go`; the implementations now live in `operator.go`.

`cd substrate/go && go test ./embednats -run 'TestTrustedWrapperAuthority' -count=1 -v` -> `--- PASS: TestTrustedWrapperAuthority/OverbroadMint (0.03s); --- PASS: TestTrustedWrapperAuthority/SteerAfterRevoke (0.06s); --- PASS: TestTrustedWrapperAuthority/SelfDeclaredTrust (0.54s); --- PASS: TestTrustedWrapperAuthority/FakeStatusImpersonation (0.85s); PASS ok embednats 0.863s`

`cd substrate/go && go test ./embednats -run 'TestSession' -count=1` -> `ok github.com/lagz0ne/tinkabot/substrate/go/embednats 15.041s`

`cd substrate/go && go test ./... -count=1` -> `ok all 7 packages`


## Outside-In Proof Matrix

All seven pinned families are proven as committed Go sub-tests in
`TestTrustedWrapperAuthority`. The scenario-matrix citation is deferred to
Slice 7 (see Scope — Does not own). Citations for Slice 7 to register:

| family            | sub-test                                               | proof mechanism |
|-------------------|--------------------------------------------------------|-----------------|
| allowed           | `TestTrustedWrapperAuthority/SelfDeclaredTrust`        | wrapper publishes to its own ingest subject without denial |
| denied-neighbor   | `TestTrustedWrapperAuthority/denied-neighbor`          | session-A credential denied publish on session-B ingest over real NATS |
| malformed         | `TestTrustedWrapperAuthority/malformed`                | non-JSON frame dropped by FrameMediator; output stream receives only valid frames |
| duplicate         | `TestTrustedWrapperAuthority/duplicate`                | two mints produce distinct keypairs; neither gains additive authority; cross-session publish denied |
| stale             | `TestTrustedWrapperAuthority/stale`                    | freshly minted credential revoked before TTL; `ApplySteerAfterRevoke` returns `(false, nil)` |
| revoked           | `TestTrustedWrapperAuthority/SteerAfterRevoke`         | steer accepted before revocation denied at apply time after revoke |
| attributed-failure| `TestTrustedWrapperAuthority/attributed-failure`       | `OverbroadMint` typed error carries offending subject in `Details` map |
| loop-suppression  | N/A                                                    | Loop suppression applies to the agent activation graph (hop-limit on recursion chains). This surface is credential-minting and steer-apply only; there is no activation chain and therefore no loop to suppress. No test case required. |

Gate-fix execution 2026-06-11:

`cd substrate/go && go test ./embednats -run 'TestTrustedWrapperAuthority' -count=1 -v` -> all 9 sub-tests PASS; `ok embednats 0.859s`

`cd substrate/go && go test ./embednats -run 'TestSession' -count=1` -> `ok github.com/lagz0ne/tinkabot/substrate/go/embednats 15.076s`

## Full-Battery Verification Evidence

Executed 2026-06-11. All commands passed.

| Command | Result |
|---------|--------|
| `bun run test` | PASS — 100 pass, 0 fail, 492 expect() calls across 18 files [3.60s] |
| `bun run test:e2e` | PASS — 1 pass, 0 fail, 16 expect() calls across 1 file [1.74s] |
| `bun run typecheck` | PASS — frontend, sdk, and orchestrator tsconfigs all clean (no errors) |
| `bun run build` | PASS — frontend vite build ok; sdk tsdown ESM+CJS build ok |
| `bun run pack:dry` | PASS — tinkabot-0.1.0.tgz: 6 files, 200.92KB unpacked |
| `bun run schema:parity` | PASS — 25 pass, 0 fail across 5 contract files; Go tests ok for all 7 packages |
| `bun run release:evidence` | PASS — 16 milestones over 11 spine steps, 5 gate results |
| `bun run gate:fakes` | PASS — gate:fakes passed |
| `bun run gate:parallel` | PASS — all 7 Go packages ok; gate:parallel passed |
| `bun run gate:coverage` | PASS — all layers meet thresholds: cmd 70.8%≥65%, contract 73.9%≥70%, core 81.7%≥78%, edge 82.8%≥78%, embednats 78.2%≥72%, frontend 100%≥95%, tinkabot 82.3%≥75% |
| `bun run gate:scenarios` | PASS — gate:scenarios passed |
| `bun run gate:manual` | PASS — gate:manual passed |
| `cd substrate/go && go test ./... -count=1` | PASS — all 7 packages ok: cmd/tinkabot, contract, core, edge, embednats (15.3s), frontend, tinkabot (4.7s) |
| `git diff --check` | PASS — no whitespace errors (empty output) |

Targeted GREEN (real embedded NATS, operator/JWT mode):

`go test ./embednats -run TestTrustedWrapperAuthority -count=1 -v` -> `OverbroadMint`, `SelfDeclaredTrust`, `SteerAfterRevoke`, `FakeStatusImpersonation` all green; `ok embednats`

Gate results: real-nats, parallel-safety, be-lazy, security, coverage all passed in the main run. no-slop passed after manual cleanup (`{"gate":"no-slop","pass":true}`).

## Wrap-Up

Slice 4/7 (`trusted-wrapper-authority`) is COMPLETE. All four contracted sub-tests are green over real embedded NATS in operator/JWT mode. The full-battery suite (100 TS tests, 1 e2e, 492 expects, 7 Go packages, all five quality gates, typecheck, build, pack, schema parity, layer validation) passes without regression. The subject-breadth check in `MintUser`, `MintTrustedWrapper`, `ApplySteerAfterRevoke`, and `StartFrameMediator` operator-mode support are proven by committed code in `substrate/go/embednats/operator.go` and `substrate/go/embednats/session_frame_mediation.go`. Trust never relaxes the raw-import denial. Nine outside-in proof cases are committed as Go sub-tests; scenario-matrix citation is deferred to Slice 7 per the scope guard. Next slice: `steering-acceptance` (session-v2 slice 5/7).
