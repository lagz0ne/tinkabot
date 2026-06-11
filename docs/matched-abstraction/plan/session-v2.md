---
layer: plan
topic: session-v2
references:
  - ../approach/session-v2.md
  - ../approach/endgame-app.md
  - ../approach/go-substrate.md
  - ./quality-v1.md
---

# Session V2 Plan

Diagram: https://diashort.apps.quickable.co/d/13b0196d

## Consumed Approach

This Plan consumes `docs/matched-abstraction/approach/session-v2.md` as top-level authority for the session shape, and the sealed `docs/matched-abstraction/approach/go-substrate.md` plus `docs/matched-abstraction/approach/endgame-app.md` as substrate and product-loop authority. It does not reopen embedded NATS ownership, operator/JWT auth vocabulary, plane separation, generated-content denial, or typed substrate failures.

The following are carried decisions, not open questions. They come from the Approach Plan-Readiness Gate:

- The canonical session is a stream of structured stream-json messages (typed token / chunk / status frames). Raw terminal / PTY is a deferred non-canonical mode behind the same frame contract.
- Trust is control-plane attribution bound into a credential at mint. No app-plane-writable record grants authority. The unconditional raw-NATS / CLI import denial is never relaxed by trust.
- Exactly one principal (the session runner) writes a session's steering subject, and re-checks the steerer's lease at apply time. One validating republisher is the only writer of a session's output subject.
- Long-lived sessions are a distinct execution subsystem that never starves activation processing; they do not run on the run-to-completion script runner.
- The durable session record + transcript is recovery authority; liveness is a heartbeat-bound lease; restart behavior is reconciliation derived from lease state, with a terminal record on every path including crash.
- Browser observation is direct NATS over the embedded WebSocket: the HttpOnly cookie session (never page-script-readable) is exchanged at a substrate mint endpoint for an ephemeral viewer credential — bearer-mode, short-TTL, loopback-source-pinned, live-revocable, leaf-scoped to a substrate-bound deliver subject plus command-acceptance publish, with no JetStream API authority — and the WebSocket upgrade is cookie-gated. NATS is the only authorization engine; no parallel streaming-gateway enforcement surface is built. The direct-browser-WebSocket deferral is retired by its named conditions (reload, revocation, denied-neighbor, stale-access proven; confidentiality satisfied at loopback; TLS required before external exposure, which stays deferred). Untrusted app content never holds a credential.
- The substrate persists session output verbatim and does not redact. Confidentiality equals transcript sensitivity; secret handling is a future hook feature, out of scope.
- The real agent runner is proven locally and by manual verbatim pairs; CI proves the subsystem with a deterministic stand-in. No live-agent CI gate is added.

The four standing quality gates from `./quality-v1.md` apply to every slice unchanged: all tests over real embedded NATS with the fakes allowlist, parallel isolated execution, dual coverage (per-layer plus scenario-matrix), and the diff-scoped `be-lazy` reviewer gate. Subject names in this Plan are illustrative coordinates for decomposition; each owning Task fixes concrete subjects under the existing `tb.` taxonomy and records them, consistent with the pinned rule that subjects are concrete values or concrete wildcard patterns.

## Decomposition

Seven slices. Each is complete at its boundary with denial and failure paths, and each that crosses an actor boundary carries its own outside-in real-NATS proof — no slice borrows another's proof.

### Slice 1 — session-contract-authority

Owns the neutral contract: session record shape, session frame vocabulary (out: token / chunk / status; in: steer / stop), steering intent, and the trust-tier vocabulary as a typed value with an owner-layer tag distinguishing it from the existing connection-exposure `AuthTier` and from ledger run-claims. JSON Schema first; generated or checked TS, Zod, Go validators, and fixtures follow it. The reserved-vocabulary collision is resolved here: the output frame field names must not collide with the script facade `rawWords` set (notably `token`), or the contract must explicitly carve session frames out of the facade scan path. Fixtures are tagged by owning layer per the activation-foundation precedent: schema validity proves shape only; authority decisions belong to the named owner layer.

Boundary completeness: parity test plus malformed-frame and unknown-frame-kind denial fixtures.

Failure families: `SchemaParityMismatch`, `UnknownFrameKind`, `MissingProvenance`, `ReservedVocabCollision`.

### Slice 2 — session-runtime-subsystem (untrusted tier)

Owns the long-lived execution subsystem: a per-session supervised process with a stdin input path and an incremental output-frame decode loop, standing apart from the run-to-completion script runner so a live session never starves activation processing. Owns the session lifecycle class (start, attach, idle/explicit stop — not a wall-clock SIGKILL), the heartbeat-bound liveness lease (the run-claim analog, with a real liveness primitive rather than a permanent create), orphan reconciliation on restart, and the terminal completion record written on every termination path including crash. The untrusted tier holds least-authority session credentials — not a session-subtree wildcard.

Boundary completeness and self-contained outside-in proof: a deterministic stand-in agent is supervised through the subsystem, its frames published to the session output subject and observed over real NATS, and a denied-neighbor proof shows the session's own credential cannot observe or write another session's subjects — proven before any session watcher is treated as live. Restart reconciliation is proven: an orphaned stand-in plus an expired liveness lease resolves to a terminal record rather than a poisoned, unrecoverable claim.

Failure families: `SessionStarvation`, `OrphanedChild`, `LivenessLeaseExpired`, `TerminalRecordMissing`.

### Slice 3 — session-frame-mediation

Owns the single validating republisher that is the only writer of a session's output subject. Enforces the frame contract on the way to durable storage (the *what*, which subject permissions cannot gate) and a per-session resource quota that replaces the bounded-output defense the batch runner provided. Decides the stream shape (per-session isolation vs shared stream with per-session caps) so one session cannot evict another's transcript, and the snapshot-plus-tail attach shape so attach does not require replaying unbounded history. No redaction — persistence is verbatim per the Approach non-goal.

Boundary completeness and outside-in proof: an over-quota flood is bounded rather than exhausting storage; a frame violating the contract is rejected at the republisher; a second session's transcript is unaffected by a first session's volume.

Failure families: `SchemaViolationOnOut`, `QuotaExceeded`, `BridgeBypassAttempt`, `CrossSessionEviction`.

### Slice 4 — trusted-wrapper-authority (trusted tier)

Owns trust as control-plane attribution. The mint enumerates exact leaf subjects (wrapper: publish output, subscribe steering; nothing else) and the mint path gains a subject-breadth check that denies a session-subtree wildcard. The runner remains the sole writer of steering and status. Owns the credential lifetime story consistent with recovery: a re-mint / renew handshake bound to the liveness lease, and revoke-on-end. Trust never relaxes the raw-import denial.

Boundary completeness and outside-in proof: an over-broad mint request is denied; a wrapper attempting to publish its own steering subject is denied; a wrapper forging a status frame is denied; a steer published before revocation is denied at apply time after revocation (closing the steer-after-revoke window).

Failure families: `OverbroadMint`, `SelfDeclaredTrust`, `SteerAfterRevoke`, `FakeStatusImpersonation`.

### Slice 5 — steering-acceptance

Owns the single mediated steering path: external steer to command acceptance to activation to the runner to wrapper input, with server-assigned ordering, an idempotency identity the client cannot forge into a second steer, and a delivery path that does not silently drop and does not cost work proportional to total activation history. This slice resolves the two scaling defects the steering path inherits: the full-history ledger scan on every acceptance, and the drop-on-full router delivery.

Boundary completeness and outside-in proof: a steer is delivered, attributed, and ordered end-to-end over real NATS; a duplicate client retry is deduplicated rather than double-applied; a steer is never silently lost under delivery pressure; a stop is ordered against pending steers.

Failure families: `SteerDropped`, `SteerOutOfOrder`, `LedgerScanUnbounded`, `NonIdempotentReplay`.

### Slice 6 — agent-wrapper-proof (real runner, local)

Owns the real-runner outside-in proof: a real Bun wrapper driving the real agent over structured stdio (stream-json, no PTY), streaming to session subjects, and steered through the mediated path, proven locally and as manual verbatim pairs. Confirms the canonical structured-message shape holds against the real producer, and that the deferred raw-terminal mode is genuinely unnecessary for the use case. This slice does not enter the CI gate suite; it extends the manual proof surface.

Boundary completeness and outside-in proof: a real session is observed and steered end-to-end locally; the manual records the exact command/outcome pairs. Where the manual-verbatim runner cannot execute an interactive session pair, the coupling to that runner is named and the runner extension or a scripted-pair alternative is owned here, not left implicit.

Failure families: `StreamJsonParseFailure`, `WrapperLaneUnproven`, `ManualDivergence`.

### Slice 7 — web-session-surface and release closure

Owns the direct-NATS browser observation surface and its composition with the browser isolation model, in two hops. Hop one: the trusted shell exchanges its HttpOnly cookie session at a substrate mint endpoint for an ephemeral viewer credential (bearer-mode, short-TTL, loopback-source-pinned, live-revocable; leaf scope: subscribe on its own deliver subject fed by a substrate-bound consumer over the session stream, publish to command acceptance; no JetStream API authority), then connects to the embedded NATS WebSocket through a cookie-gated same-origin upgrade path served by the binary — the credential is unusable without the cookie and the cookie is unusable without the credential. Viewer credential renewal reuses Slice 4's renew handshake; per-session authorization is NATS denied-neighbor enforcement, not new endpoint logic. Hop two: the trusted shell forwards session frames to untrusted app content only over the existing leased frame channel, and the frame lease is extended with a session observation scope alongside its command allowlist — the lease is the app-scoped token whose imports (observable sessions) and exports (steering commands) are minted, enforced, and revoked through the existing lease lifecycle. App-piece steering rides the proven path unchanged: typed content intent, shell acceptance, command acceptance into Slice 5's mediated steering. Untrusted app content never holds any credential. Also owns program closure: registering the new gates and outside-in surfaces, adding scenario-matrix entries for every new session surface, and extending the centralized release-evidence manifest so the program can prove itself and cannot silently weaken its own gates — including recording the retirement of the direct-browser-WebSocket deferral with its condition citations.

Boundary completeness and outside-in proof: a viewer credential for one session is denied another session's subjects (denied-neighbor over real NATS), proven before viewers are served; a WebSocket upgrade without a valid cookie session is denied; an expired or revoked viewer credential is disconnected and denied reconnect, and renewal through the cookie succeeds; an app frame whose lease lacks a session's observation scope receives no frames for it, and its steer intent for that session is denied at shell acceptance; the shell observes and steers a session end-to-end through the minted credential, the leased frame channel, and command acceptance; `release:evidence` covers the session program over the extended manifest.

Failure families: `CrossSessionLeak`, `UngatedUpgrade`, `StaleViewerCred`, `FrameScopeEscape`, `GateMatrixVacuous`, `UngrownManifest`.

## Dependency Ordering

- Slice 1 precedes all others: every slice consumes the session contract.
- Slice 2 precedes 3, 4, 5, 6, 7: the subsystem and its lifecycle/recovery exist before mediation, trust, steering, real-runner, or surface ride on it.
- Slice 3 precedes 7: the output mediation and stream/attach shape exist before the browser observes them.
- Slice 4 precedes 5 and 6: the mint and single-writer authority exist before steering enforcement and before the real trusted wrapper runs.
- Slice 5 precedes 6 and 7: the mediated steering path exists before the real runner is steered and before the browser sends steering chat.
- Slice 6 does not block Slice 7. Slice 7's browser observation consumes Slice 2's output (via Slice 3) and the already-complete command-acceptance milestone; it depends on Slice 4 only for the viewer/steerer authorization it reuses, and on Slice 5 for the steering path. It does not depend on the real-runner proof. Slices 6 and 7 may proceed in parallel once 3, 4, and 5 are complete.

## Handoff Contract

Each slice is handed to its owning Task with: the consumed Approach invariants it must satisfy, its boundary-completeness obligation (denial and failure paths named above), its self-contained outside-in proof obligation (no borrowed proof), its enumerated failure families with one owning test per family per traced-TDD, and its concrete-subject obligation (the Task fixes and records the actual `tb.` subjects). A slice is not done until its outside-in proof runs over real embedded NATS and its denied-neighbor or per-endpoint denial is proven before the corresponding watcher or viewer is treated as live.

## Verification Strategy

Inside-out: each slice proves ownership of its contract and failure families with per-layer tests over real embedded NATS, one owning test per declared failure family.

Outside-in: each actor-boundary-crossing slice proves behavior through real NATS subjects, streams, KV/Object records, or the browser WebSocket path — never through fakes standing in for the seam. Slice 2 proves the subsystem and denied-neighbor; Slice 3 proves quota and isolation; Slice 4 proves the four authority denials; Slice 5 proves attributed, ordered, lossless, idempotent steering; Slice 6 proves the real runner locally and by manual pairs; Slice 7 proves viewer denied-neighbor over real NATS, the cookie-gated upgrade denial, stale-credential disconnect and renewal, and the end-to-end shell observe-and-steer path.

Scenario-matrix: every new outside-in session surface is added to the scenario matrix so the scenarios gate sees it; a surface that is never added must not pass the gate vacuously. This obligation is a post-check in Slices 2, 5, and 7.

Closure: Slice 7 extends the centralized release-evidence manifest and gate list to cover the session program. The closure check must include the pinned negative-case families (denied-neighbor, malformed, duplicate, revoked, attributed failure) over session surfaces, with denial oracles output-parsed rather than exit-code, consistent with the existing release gate.

## Deferred Scope

Named here so an operator agent does not wander into them and so the release manifest keeps naming them deferred:

- Raw terminal / PTY session mode.
- Built-in redaction, encryption, or content filtering of transcripts (future hook feature).
- Multi-viewer fanout tuning and transcript compaction tuning beyond the snapshot-plus-tail attach shape.
- Session resume UX beyond reconciliation to a terminal-or-readopted state.
- Real agent runner inside CI gates.
- Session list / CRUD UI.
- External (non-loopback, TLS) exposure. Direct browser NATS WebSocket is no longer deferred at the loopback posture — its deferral is retired by Slice 7 with condition citations — but any external/TLS form of it stays deferred with external exposure.

## Escalation Log

- Open: none. The Approach Plan-Readiness Gate is fully met; all five forking decisions are resolved as carried decisions above.
- If a Task discovers that a carried decision cannot be satisfied as stated (for example, that the mediated steering path cannot meet both losslessness and the no-history-scan obligation without an Approach-level change), it escalates upward to the Approach layer rather than redefining the invariant inside the slice.
