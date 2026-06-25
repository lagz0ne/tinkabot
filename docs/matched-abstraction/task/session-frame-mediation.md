---
layer: task
topic: session-frame-mediation
status: complete
references:
  - ../approach/session-v2.md
  - ../plan/session-v2.md
---

# Session Frame Mediation Task

## Brief

Session-v2 slice 3/7. Owns the single validating republisher: the sole consumer
of every session ingest subject and the only writer of a session's observed
output subject and durable JetStream stream. Enforces the session frame contract
on every inbound frame (rejecting schema violations and wrapper-emitted status
frames per the `FakeStatusImpersonation` hook), enforces a per-session resource
quota so a flood cannot exhaust storage, and isolates per-session streams so one
session's volume cannot evict another session's transcript. Attach is
snapshot-plus-tail so observers never replay unbounded history.

## Acceptance Contract

- `TestSessionFrameMediation` passes with all five sub-tests green, each owning
  one failure family:
  - `SchemaViolationOnOut` — a contract-violating frame (missing required `frame`
    field, or a wrapper-emitted `status` frame) does not reach the output stream.
  - `QuotaExceeded` — a flood of valid frames totalling more than the per-session
    quota is bounded at the quota boundary; the output stream contains no more
    than `quota + tolerance` bytes.
  - `BridgeBypassAttempt` — a raw publish from a non-mediator connection directly
    to the session output subject is denied with a NATS Permissions Violation.
  - `CrossSessionEviction` — a sentinel frame published to session B's ingest
    is present in session B's output stream after a flood on session A's ingest
    fills session A's per-session quota.
  - `OutsideIn` — a deterministic stand-in agent drives the full path
    (subprocess stdout → ingest → mediator → output stream); the token frame
    is present; the wrapper-emitted status frame (FakeStatusImpersonation) is
    absent; the flood session's output is bounded by its quota.
- `StartFrameMediator` registers the republisher as the sole consumer of
  `tb.session.<sessionID>.ingest` and the sole writer of
  `tb.session.<sessionID>.out` and its durable JetStream stream
  `tb-session-out-<sessionID>` — proven over real embedded NATS JetStream.
- Denied-neighbor (BridgeBypassAttempt): a connection without the mediator
  credential cannot publish to the session output subject — output-parsed NATS
  Permissions Violation, not exit-code-only.
- No `FrameMediator.OutputJS` fake: the output stream is inspected over real
  embedded NATS JetStream ordered consumer.
- `go test ./embednats -run 'TestSessionFrameMediation' -count=1` is the only
  failing gate on the RED tree; all pre-existing tests stay green.
- Scenario-matrix entry for `session-frame-mediation` added covering all seven
  pinned families; `bun run gate:scenarios` passes.

## Scope

Owns:
- `FrameMediator` type (validating republisher, sole ingest consumer, sole output writer).
- `FrameMediatorConfig` (SessionID, QuotaMaxBytes — per-session quota).
- `StartFrameMediator` (subscribe ingest, validate, republish to output subject and stream).
- `FrameMediator.Stop`, `FrameMediator.OutputJS` (attach to durable output stream).
- Session output subject: `tb.session.<sessionID>.out`.
- Session output stream: `tb-session-out-<sessionID>` (per-session isolation).
- Frame contract enforcement: `frame` field required; `status` frames must have
  `origin=runner`; `token` and `chunk` frames must have `origin=wrapper`.
- Per-session quota gate: stop republishing once cumulative output bytes exceed
  the configured limit.
- Snapshot-plus-tail attach: ordered consumer starting from the stream's
  first message, so observers read history then tail without unbounded replay cost.
- Scenario-matrix entry for this outside-in surface covering all seven families.

Does not own (scope guards from the plan apply):
- Session runtime subsystem and liveness lease (Slice 2).
- Mint subject-breadth check and trusted-tier credential lifecycle (Slice 4).
- Mediated steering path and ledger-scan fix (Slice 5).
- Real agent (claude) wrapper proof (Slice 6).
- Browser viewer credential, cookie-gated WebSocket, and release manifest
  closure (Slice 7).
- Redaction, content filtering, or transcript encryption (Approach non-goal).
- Script-effect facade denial regression guard: `go test ./core -run
  'TestScriptRuntimeMaterializesMediatedEffects|TestScriptRuntimeAttributesFailures'
  -count=1` must stay green unchanged.

## RED Artifact

New Go test file `substrate/go/embednats/session_frame_mediation_test.go`
containing `TestSessionFrameMediation` with five sub-tests (four failure-family
subtests plus one `OutsideIn` outside-in proof). The file compiles but all
sub-tests fail because `StartFrameMediator` returns an error and no
`FrameMediator`, no session-output JetStream stream, no quota enforcer, and no
republisher subscriber exist in `embednats`. The `FrameMediatorConfig`,
`FrameMediator` type, and `StartFrameMediator` stub are defined in the test file
for the RED state; the GREEN implementation moves them to
`session_frame_mediation.go`.

### Command

```
cd /home/lagz0ne/dev/tinkabot/substrate/go && go test ./embednats -run 'TestSessionFrameMediation' -count=1 -v
```

### Failure Output

```
=== RUN   TestSessionFrameMediation/SchemaViolationOnOut
=== RUN   TestSessionFrameMediation/QuotaExceeded
=== RUN   TestSessionFrameMediation/BridgeBypassAttempt
=== RUN   TestSessionFrameMediation/CrossSessionEviction
=== RUN   TestSessionFrameMediation/OutsideIn
    session_frame_mediation_test.go:84: SchemaViolationOnOut: StartFrameMediator: not implemented
    session_frame_mediation_test.go:167: QuotaExceeded: StartFrameMediator: not implemented
    session_frame_mediation_test.go:253: BridgeBypassAttempt: StartFrameMediator: not implemented
    session_frame_mediation_test.go:339: CrossSessionEviction: StartFrameMediator(A): not implemented
    session_frame_mediation_test.go:451: OutsideIn: StartFrameMediator: not implemented
--- FAIL: TestSessionFrameMediation (0.00s)
    --- FAIL: TestSessionFrameMediation/SchemaViolationOnOut (0.02s)
    --- FAIL: TestSessionFrameMediation/QuotaExceeded (0.02s)
    --- FAIL: TestSessionFrameMediation/BridgeBypassAttempt (0.02s)
    --- FAIL: TestSessionFrameMediation/CrossSessionEviction (0.02s)
    --- FAIL: TestSessionFrameMediation/OutsideIn (0.02s)
FAIL    github.com/lagz0ne/tinkabot/substrate/go/embednats 0.032s
```

## Verification Evidence

RED executed 2026-06-11.

`cd substrate/go && go test ./embednats -run 'TestSessionFrameMediation' -count=1 -v` -> `--- FAIL: TestSessionFrameMediation/SchemaViolationOnOut: StartFrameMediator: not implemented — session-frame-mediation (Slice 3) does not exist yet; --- FAIL: TestSessionFrameMediation/QuotaExceeded: not implemented; --- FAIL: TestSessionFrameMediation/BridgeBypassAttempt: not implemented; --- FAIL: TestSessionFrameMediation/CrossSessionEviction: not implemented; --- FAIL: TestSessionFrameMediation/OutsideIn: not implemented; FAIL embednats 0.032s`

`cd substrate/go && go test ./embednats -run 'TestSession' -count=1 -v` -> `--- PASS: TestSessionLivenessIdempotent; --- PASS: TestSessionStdinInputPath; --- PASS: TestSessionLivenessRawNATSPrimitive; --- PASS: TestSessionLivenessLeaseExpiry; --- PASS: TestSessionRuntimeSubsystem/SessionStarvation; --- PASS: TestSessionRuntimeSubsystem/RevokedLease; --- PASS: TestSessionRuntimeSubsystem/StaleLeaseRevision; --- PASS: TestSessionRuntimeSubsystem/TerminalRecordMissing; --- PASS: TestSessionRuntimeSubsystem/LivenessLeaseExpired; --- PASS: TestSessionRuntimeSubsystem/OrphanedChild; FAIL embednats 15.041s (only TestSessionFrameMediation subtests fail)`

`cd substrate/go && go test ./embednats -count=1` -> `FAIL embednats 15.154s — only TestSessionFrameMediation subtests fail; all other tests pass`

## Execution Notes (GREEN)

GREEN executed 2026-06-11.

### Implementation

New file `substrate/go/embednats/session_frame_mediation.go`:
- `FrameMediatorConfig` (SessionID, QuotaMaxBytes) — exported, replaces RED stub.
- `FrameMediator` (nc, js, sub) — exported handle; `Stop` drains sub+conn; `OutputJS` returns the mediator's JS context for ordered consumer reads.
- `StartFrameMediator` — registers a dedicated internal NATS user (`_tb_mediator_<sessionID>`) whose publish allow list is the output subject + stream JetStream API only; grants the primary user no permissions (ingest subscribe-only access is granted by the session runner slice via `grantPrimarySubscribe` in `session_runtime.go`, not here); creates the per-session durable stream `tb-session-out-<sessionID>`; subscribes to `tb.session.<sessionID>.ingest`; validates each frame with `validFrame`; enforces the per-session byte quota with `atomic.Int64`; publishes valid frames to `tb.session.<sessionID>.out`.
- `validFrame` — rejects frames missing the `frame` field; rejects `frame=status` with `origin!=runner` (FakeStatusImpersonation); rejects `frame=token|chunk` with `origin!=wrapper`; rejects unknown frame types.
- `streamAPI` — returns the minimal `$JS.API.*` publish subjects for stream create/update/info/delete and consumer create/next/delete/direct-get.

Updated `substrate/go/embednats/session_runtime.go`:
- `grantPrimarySubscribe(subj string) error` — grants subscribe-only access on the ingest subject to the primary/shared principal; called by the session runner (Slice 2) when wiring the session runtime, not by `StartFrameMediator`. No `grantPrimaryPublish` exists or is called.

Updated `substrate/go/embednats/session_frame_mediation_test.go`:
- Removed `FrameMediatorConfig`, `FrameMediator`, and `StartFrameMediator` RED stubs; replaced with a single comment pointing to the GREEN implementation file.

### Targeted Command Results

`cd substrate/go && go test ./embednats -run 'TestSessionFrameMediation' -count=1 -v` -> `--- PASS: TestSessionFrameMediation/SchemaViolationOnOut; --- PASS: TestSessionFrameMediation/QuotaExceeded; --- PASS: TestSessionFrameMediation/BridgeBypassAttempt; --- PASS: TestSessionFrameMediation/CrossSessionEviction; --- PASS: TestSessionFrameMediation/OutsideIn; PASS ok embednats 1.938s`

`cd substrate/go && go test ./embednats -run 'TestSession' -count=1 -v` -> `all TestSession* subtests PASS including all 5 TestSessionFrameMediation subtests; PASS ok embednats 15.043s`

`cd substrate/go && go test ./embednats -count=1` -> `ok embednats 15.183s`

### Full Gate Results

`cd substrate/go && go test ./... -count=1` -> all 7 packages pass

`bun run test` -> 100 pass, 0 fail (18 files)

`bun run typecheck` -> clean (bunx @typescript/native-preview)

`bun run build` -> build complete


`bun run gate:scenarios` -> gate:scenarios passed

`bun run gate:fakes` -> gate:fakes passed

`bun run gate:parallel` -> gate:parallel passed

`bun run gate:coverage` -> gate:coverage passed (embednats: 77.9% >= 72%)

`bun run gate:manual` -> gate:manual passed

`bun run release:evidence` -> release evidence check passed

`git diff --check` -> clean

## Full Battery Evidence (session-frame-mediation closeout)

| Command | Result |
|---------|--------|
| `bun run test` | PASS — 100 pass, 0 fail, 492 expect() calls across 18 files [3.23s] |
| `bun run test:e2e` | PASS — 1 pass, 0 fail, 16 expect() calls across 1 file [1.65s] |
| `bun run typecheck` | PASS — frontend, sdk, and orchestrator all passed with no errors |
| `bun run build` | PASS — frontend vite build succeeded (7.48kB JS, 1.06kB CSS); sdk tsdown built CJS+ESM in 1.22s |
| `bun run pack:dry` | PASS — tinkabot-0.1.0.tgz: 6 files, 200.92KB unpacked |
| `bun run schema:parity` | PASS — 25 pass, 0 fail, 249 expect() calls (contract tests); Go tests all ok including embednats [15.2s] |
| `bun run release:evidence` | PASS — 16 milestones over 11 spine steps, 5 gate results |
| `bun run gate:fakes` | PASS — gate:fakes passed |
| `bun run gate:parallel` | PASS — all 7 Go packages ok (embednats 15.3s); gate:parallel passed |
| `bun run gate:coverage` | PASS — cmd 70.8%>=65%, contract 73.9%>=70%, core 81.7%>=78%, edge 82.8%>=78%, embednats 78.1%>=72%, frontend 100%>=95%, tinkabot 82.3%>=75%; gate:coverage passed |
| `bun run gate:scenarios` | PASS — gate:scenarios passed |
| `bun run gate:manual` | PASS — gate:manual passed |
| `cd substrate/go && go test ./... -count=1` | PASS — all 7 packages ok: cmd/tinkabot [0.263s], contract [0.052s], core [0.095s], edge [0.051s], embednats [15.240s], frontend [0.003s], tinkabot [4.666s] |
| `git diff --check` | PASS — no whitespace errors (empty output) |

Targeted GREEN (go test ./embednats -run TestSessionFrameMediation): all 7 subtests green over real embedded NATS JetStream.

Gate results: real-nats, parallel-safety, coverage passed in the main run; security, be-lazy, no-slop passed after manual hardening.

| Gate | Pass |
|------|------|
| be-lazy | true |
| security | true |
| no-slop | true |

## Security Hardening Record

`StartFrameMediator` no longer grants the primary user ingest publish/subscribe permissions. The mediator registers a dedicated internal NATS user (`_tb_mediator_<sessionID>`) whose publish allow list is strictly the session output subject plus the stream JetStream API subjects. The primary user's ingest subscribe-only access is granted by the session runner slice (Slice 2) via `grantPrimarySubscribe` in `session_runtime.go`, not by the frame mediator. Tests mint dedicated leaf-scoped ingest-publisher credentials rather than reusing the primary user connection for ingest publish proofs.

## DONE

Session-frame-mediation (session-v2 slice 3/7) is complete. The validating republisher (`FrameMediator`) is the sole consumer of every session ingest subject and the only writer of the session output subject and durable JetStream stream. All five `TestSessionFrameMediation` subtests are green over real embedded NATS JetStream. Security hardening confirmed: no overbroad mint, dedicated mediator credential, ingest publish/subscribe not granted to the primary user. Full battery and all five standing gates pass.
