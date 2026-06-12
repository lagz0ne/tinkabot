---
layer: task
topic: agent-wrapper-proof
status: complete
references:
  - ../approach/session-v2.md
  - ../plan/session-v2.md
---

# Agent Wrapper Proof Task

## Brief

Session-v2 slice 6/7. Owns the real-runner outside-in proof: a Go wrapper
process under `substrate/go/apps/wrapper` (placement/language escalated and
decided 2026-06-12, recorded in the Plan Escalation Log) drives the real
`claude` CLI over structured stdio, connects to the session subsystem as a
trusted-wrapper principal (minted leaf-scoped credential from
`MintTrustedWrapper`), streams output through the frame-mediation path as
canonical `session.frame` envelopes, and accepts mid-session steers through
the steer subject translated onto the agent's stdin.

The locally verified flag set is `--print --verbose --input-format
stream-json --output-format stream-json --include-partial-messages`. The
Plan's original flag set omitted `--verbose`; the real CLI (claude 2.1.173)
rejects `--print` with stream-json output without it. Recorded here as a
live-CLI finding, not a Plan change.

Proof is local and via manual verbatim pairs — this slice does not enter the
CI gate suite for the live agent (no-live-agent-in-CI carried decision) and
needs no scenario-matrix entry (Plan §Verification Strategy exemption).

`StreamJsonParseFailure` is owned by a CI-runnable decode test over recorded
real-CLI fixtures (no live agent). `WrapperLaneUnproven` and
`ManualDivergence` are owned by the manual-pair surface in `docs/manual/v1.md`
and are exempt from the per-layer real-NATS rule. The manual-runner coupling
is named in the manual: `gate:manual` executes only `# ->` pairs against the
running binary, and sessions join the binary surface in Slice 7; until then
the wrapper pairs use `# Expected:` annotations validated by the scripted
equivalents (`TestAgentWrapperMediated`, `TestAgentWrapperLocalE2E`).

## Acceptance Contract

- `go test ./apps/... -count=1` passes: decode over recorded real-CLI
  stream-json lines with typed `*ParseFailure` on malformed input, canonical
  token/chunk frame build validated against the slice-1 contract registry,
  never-status over all recorded lines, pump loop survival of a malformed
  line, steer translate/skip/reject, `cmd/run` arg validation and creds
  failure.
- `go test ./embednats -run TestAgentWrapperMediated -count=1` passes: the
  real wrapper loop under a `MintTrustedWrapper` credential over real
  embedded NATS in operator/JWT mode, canonical frames observed on the
  mediated output stream, malformed-line survival, steer intent delivered to
  the subprocess stdin in claude user-message shape.
- `TB_E2E_CLAUDE=1 go test ./embednats -run TestAgentWrapperLocalE2E` passes
  locally: a live claude session opened by a steer, observed as canonical
  frames on the mediated stream, and steered mid-session.
- `go run ./apps/wrapper/cmd/run` builds and validates its arguments.
- Manual section "Agent wrapper (local proof only)" in `docs/manual/v1.md`
  records the pairs and names the manual-runner coupling.
- `bun run validate:layers` and the five standing gates pass with the new
  `apps` layer staged.

## Drift Resolution (2026-06-12)

A first implementation pass landed a Go wrapper while the Plan pinned Bun at
repo-root `apps/wrapper`. Escalated to the user instead of silently accepted;
decision: keep Go at `substrate/go/apps/wrapper` (the project consolidated on
Go for everything embedded-NATS-facing). Plan slice 6 text and the Approach's
Purpose framing were amended; the eight invariants are untouched.

Three defects in that first pass were fixed under new RED tests:

1. Dead-code owner: `ParseStreamJsonFrame` lived in `embednats` and was
   called by nothing — the wrapper inlined a duplicate decode and silently
   dropped malformed lines. Parsing moved into the wrapper package and the
   executed loop (`pump`) now uses it.
2. Substrate leak + contract bypass: claude stream-json vocabulary sat in the
   substrate adapter, and the wrapper hand-built non-canonical envelopes
   (missing `kind`/`text`/`body`, extra `event`/`payload`) that pass the
   mediator's frame/origin check but violate the slice-1 schema. The wrapper
   now emits canonical frames validated against the contract registry.
3. Unexecutable manual pair: the manual referenced `go run
   ./apps/wrapper/cmd/run`, which did not exist, and steered with a
   non-canonical `{"text":...}` payload. `cmd/run` now exists; the manual
   uses the canonical steer intent.

A fourth defect surfaced by the new RED: `StartWrapper` ignored its context,
so the subprocess was never killed and `Wait` blocked forever (the old code
hung in `publishLine` until the 600s go-test timeout). Cancelling ctx now
kills the subprocess.

## Design Decisions

- Frame mapping: `stream_event` carrying a `text_delta` becomes a token frame
  (`text`); every other agent event becomes a chunk frame whose `body` is the
  verbatim event line as a string value. Claude event keys
  (`usage.input_tokens` et al.) collide with the reserved-vocab `safeValue`
  facade, which scans property names, never values — so verbatim persistence
  (Approach stance) rides as a string body.
- Steer translation: canonical `session.steer_intent` becomes the claude
  stream-json user message `{"type":"user","message":{"role":"user",
  "content":[{"type":"text","text":...}]}}`. Non-steer kinds are skipped;
  malformed payloads are rejected typed and never reach the agent stdin.
- The wrapper never emits status frames (runner-originated by contract); the
  mediator additionally rejects wrapper-origin status (slice 3/4 proof).
- Recorded fixtures are real: captured 2026-06-12 from claude 2.1.173 with
  the named flags (`apps/wrapper/testdata/recorded.jsonl`). The earlier
  invented fixtures (bare `message_start` top-level types) did not match the
  real CLI, which wraps Anthropic events as `{"type":"stream_event",...}`.

## RED Artifact

Executed 2026-06-12, three RED legs:

1. `go test ./apps/... -count=1` — compile failure: `undefined:
   ParseStreamJsonFrame`, `undefined: SessionFrame` (and `SteerToStdin`) in
   the new `apps/wrapper/wrapper_test.go` unit tests over recorded fixtures.
2. `go test ./embednats -run TestAgentWrapperMediated -count=1` — the
   rewritten outside-in test against the first-pass wrapper: no canonical
   token frame on the mediated output stream, steer not translated; the run
   hung in the old `publishLine` until the 600s suite timeout (ctx defect).
3. `go run ./apps/wrapper/cmd/run --help` — package did not exist.

## Verification Evidence

GREEN executed 2026-06-12:

`cd substrate/go && go test ./apps/... -count=1 -cover` -> `ok wrapper
coverage: 40.5%, ok cmd/run coverage: 72.2%` — `TestAgentWrapperStreamJsonDecode`
(5 subtests over recorded real-CLI lines, typed `*ParseFailure` on
malformed), `TestAgentWrapperSessionFrame` (token/chunk validated against the
contract registry via `contract.Open`/`Validate`, never_status over all
recorded lines), `TestAgentWrapperPump` (malformed line skipped, loop
continues, canonical output), `TestAgentWrapperSteerTranslate`
(translate/skip/reject), `TestRunArgValidation`/`TestRunBadCreds` (cmd/run).

`cd substrate/go && go test ./embednats -run TestAgentWrapperMediated
-count=1` -> `ok 1.08s` — operator/JWT mode over real embedded NATS,
`MintTrustedWrapper` credential, real subprocess replaying recorded lines
(malformed line first) then echoing stdin, canonical token/chunk frames
observed on the mediated output stream `tb-session-out-<id>`, steer intent
delivered to subprocess stdin in claude user-message shape and observed back
through mediation.

`cd substrate/go && TB_E2E_CLAUDE=1 go test ./embednats -run
TestAgentWrapperLocalE2E -count=1` -> `PASS (13.34s)` — the live real-runner
proof: a claude session opened by a first steer ("pong" tokens observed on
the mediated stream), then steered mid-session ("maple", streamed as deltas
"ma"+"ple" and matched on the joined token transcript). The first run failed
on a wrong oracle (single-frame contains instead of joined transcript); the
frame dump showed the steered response present as split deltas, and the
oracle was fixed to the joined-transcript form.

`bun run validate:layers` -> `Layer validation passed` with the amended
Approach/Plan/task docs.

Full battery 2026-06-12 (slice files staged): `bun run test`, `test:e2e`,
`typecheck`, `build`, `pack:dry`, `schema:parity`, `release:evidence`,
`test:layers`, `gate:fakes`, `gate:parallel`, `gate:coverage`,
`gate:scenarios`, `gate:manual`, `go test ./... -count=1`, `git diff --check`
-> all PASS.

`coverage-thresholds.json` gains `"apps": 40`. The floor reflects in-package
coverage (pure decode/frame/steer/pump logic plus cmd/run arg paths);
`StartWrapper`'s behavior owner is the cross-package
`TestAgentWrapperMediated` over real NATS, which plain per-layer `-cover`
cannot attribute.

## Scope

Owns:

- `substrate/go/apps/wrapper` — `ParseStreamJsonFrame`/`StreamJsonFrame`/
  `ParseFailure` (stream_json.go), `SessionFrame`/`SteerToStdin` (frame.go),
  `StartWrapper`/`WrapperConfig`/`WrapperHandle`/`pump` (wrapper.go),
  `cmd/run` entry point, recorded fixtures under `testdata/`.
- `substrate/go/embednats/agent_wrapper_proof_test.go` —
  `TestAgentWrapperMediated` (CI-runnable outside-in proof) and
  `TestAgentWrapperLocalE2E` (env-guarded live agent proof; establishes the
  local-only guard precedent `TB_E2E_CLAUDE=1`).
- Manual section "Agent wrapper (local proof only)" in `docs/manual/v1.md`
  with the named manual-runner coupling.
- `"apps"` entry in `substrate/go/coverage-thresholds.json`.

Does not own:

- Scenario-matrix entry (exempt local/manual slice).
- Live-agent CI gate (carried decision).
- Release manifest closure, gate-list extension, browser viewer credential,
  cookie-gated WebSocket, frame-lease scope (all Slice 7).
- Raw terminal/PTY session mode (deferred scope).

## Residual Risk

- `cmd/run`'s happy path (real connect + wait) is thin assembly of
  `StartWrapper`, which is live-proven; the entry point itself is proven for
  arg validation and creds failure only.
- The live e2e depends on a locally authenticated claude CLI; flag drift in
  future CLI versions surfaces only on the next local run (by design —
  no-live-agent-in-CI).
