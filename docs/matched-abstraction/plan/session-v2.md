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
- Account placement: every session principal — runner, republisher, trusted wrapper, viewer — is minted in the app-plane account (`TB_APP`). The control plane only attributes and mints; no session-scoped principal, subject, or bridge authority is placed in `TB_CONTROL`.
- Tier semantics: an untrusted wrapper holds zero NATS authority — it speaks the frame contract over stdio only, and the runner bridges. A trusted wrapper holds its own minted leaf-scoped credential (publish ingest, subscribe steering) and connects directly. Both tiers converge on the same session subjects, so observers cannot tell tiers apart.
- Output pipeline: wrapper frames reach the session ingest subject (published by the runner for the untrusted tier, by the wrapper itself for the trusted tier); the validating republisher is the sole consumer of ingest and the sole writer of the observed output subject and durable stream.

The four standing quality gates from `./quality-v1.md` apply to every slice unchanged: all tests over real embedded NATS with the fakes allowlist, parallel isolated execution, dual coverage (per-layer plus scenario-matrix), and the diff-scoped `be-lazy` reviewer gate. Subject names in this Plan are illustrative coordinates for decomposition; each owning Task fixes concrete subjects under the existing `tb.` taxonomy and records them, consistent with the pinned rule that subjects are concrete values or concrete wildcard patterns.

## Decomposition

Seven slices. Each is complete at its boundary with denial and failure paths, and each that crosses an actor boundary carries its own outside-in real-NATS proof — no slice borrows another's proof.

### Slice 1 — session-contract-authority

Owns the neutral contract: session record shape, session frame vocabulary (out: token / chunk / status; in: steer / stop), steering intent, and the trust-tier vocabulary as a typed value with an owner-layer tag distinguishing it from the existing connection-exposure `AuthTier` and from ledger run-claims. JSON Schema first; generated or checked TS, Zod, Go validators, and fixtures follow it. The reserved-vocabulary collision is resolved here, knowing the facade scan matches field *names* only (substring over `rawWords`; values are never scanned), so the frame kind value `token` does not itself collide: resolve by choosing frame field names that avoid `rawWords` substrings; an explicit facade carve-out is the fallback only if naming cannot avoid it, and any carve-out must not weaken the script-effect facade scan for non-session paths — existing facade denial tests stay green unchanged. Frame origin is part of the contract: status frames are runner-originated lifecycle frames, while token/chunk frames are wrapper-originated, so a wrapper-emitted status frame arriving on ingest is rejectable by the republisher — that rejection is the `FakeStatusImpersonation` case Slices 3 and 4 prove. Session contracts extend the `schemas/base/v1` lane and the existing contract registry so `bun run schema:parity` covers them without new wiring. Fixtures are tagged by owning layer per the activation-foundation precedent: schema validity proves shape only; authority decisions belong to the named owner layer.

Boundary completeness: parity test plus malformed-frame and unknown-frame-kind denial fixtures.

Failure families: `SchemaParityMismatch`, `UnknownFrameKind`, `MissingProvenance`, `ReservedVocabCollision`.

### Slice 2 — session-runtime-subsystem (untrusted tier)

Owns the long-lived execution subsystem: a per-session supervised process with a stdin input path and an incremental output-frame decode loop, standing apart from the run-to-completion script runner so a live session never starves activation processing. Owns the session lifecycle class (start, attach, idle/explicit stop — not a wall-clock SIGKILL), the heartbeat-bound liveness lease (the run-claim analog, with a real liveness primitive rather than a permanent create; per-key TTL is supported by the pinned nats-server v2.14.2 but only through the nats.go `jetstream` package — `jetstream.KeyTTL` on a bucket created with `LimitMarkerTTL` — which the legacy `nc.JetStream()` KV API used elsewhere in embednats cannot express. The liveness store either adopts the new package for its dedicated lease bucket or uses a dedicated bucket with bucket-level TTL refreshed by heartbeat re-put; the slice's first RED proves key expiry over the real embedded server before anything builds on it), orphan reconciliation on restart, and the terminal completion record written on every termination path including crash. Tier posture in this slice per the carried decisions: the untrusted wrapper process holds zero NATS authority and speaks the frame contract over stdio only; the runner holds the per-session least-authority credential — never a session-subtree wildcard, never `tb.session.>` wholesale — and publishes the wrapper's frames to the session ingest subject. The observed output subject belongs to Slice 3's republisher, not to this slice.

Boundary completeness and self-contained outside-in proof: a deterministic stand-in agent — a real subprocess speaking the real frame contract over stdio, not a fake of the NATS seam — is supervised through the subsystem, its frames published to the session ingest subject and observed on the ingest subject over real NATS (no temporary output-subject bridge; the observed output subject arrives with Slice 3), and a denied-neighbor proof shows the runner's session-scoped credential cannot observe or write another session's subjects — proven before any session watcher is treated as live. Restart reconciliation is proven: an orphaned stand-in plus an expired liveness lease resolves to a terminal record rather than a poisoned, unrecoverable claim.

Failure families: `SessionStarvation`, `OrphanedChild`, `LivenessLeaseExpired`, `TerminalRecordMissing`.

### Slice 3 — session-frame-mediation

Owns the single validating republisher: the sole consumer of session ingest subjects and the only writer of a session's observed output subject and durable stream. Enforces the frame contract on the way to durable storage (the *what*, which subject permissions cannot gate) and a per-session resource quota that replaces the bounded-output defense the batch runner provided. Decides the stream shape (per-session isolation vs shared stream with per-session caps) so one session cannot evict another's transcript, and the snapshot-plus-tail attach shape so attach does not require replaying unbounded history. No redaction — persistence is verbatim per the Approach non-goal.

Boundary completeness and outside-in proof: an over-quota flood is bounded rather than exhausting storage; a frame violating the contract is rejected at the republisher; a second session's transcript is unaffected by a first session's volume.

Failure families: `SchemaViolationOnOut`, `QuotaExceeded`, `BridgeBypassAttempt`, `CrossSessionEviction`.

### Slice 4 — trusted-wrapper-authority (trusted tier)

Owns trust as control-plane attribution. The mint enumerates exact leaf subjects (wrapper: publish on its session ingest subject, subscribe on its steering subject; nothing else — never the observed output subject, whose sole writer stays Slice 3's republisher) and the mint path gains a subject-breadth check that denies a session-subtree wildcard. The breadth check lives in `MintUser` (the single mint seam) and denies wildcard grants only within the session subject space (`tb.session.`-prefixed patterns); existing `_INBOX.>`, `$KV.*.>`, and JS-API wildcard grants remain legal and existing mint tests stay green. The runner remains the sole writer of steering and status. Owns the credential lifetime story consistent with recovery: a re-mint / renew handshake bound to the liveness lease, and revoke-on-end. Trust never relaxes the raw-import denial.

Boundary completeness and outside-in proof: an over-broad mint request is denied; a wrapper attempting to publish its own steering subject is denied; a wrapper forging a status frame is denied; a steer published before revocation is denied at apply time after revocation (closing the steer-after-revoke window).

Failure families: `OverbroadMint`, `SelfDeclaredTrust`, `SteerAfterRevoke`, `FakeStatusImpersonation`.

### Slice 5 — steering-acceptance

Owns the single mediated steering path: external steer to command acceptance to activation to the runner to wrapper input, with server-assigned ordering, an idempotency identity the client cannot forge into a second steer, and a delivery path that does not silently drop and does not cost work proportional to total activation history. This slice resolves the two scaling defects the steering path inherits — the full-history ledger scan on every acceptance, and the drop-on-full router delivery — by fixing them at their existing seams (the router send path and the ledger source read), not by building a parallel steering-only delivery path beside the defective one; no existing test asserts the drop-on-full or buffer-size behavior, so the fix is not constrained by current assertions.

Boundary completeness and outside-in proof: a steer is delivered, attributed, and ordered end-to-end over real NATS; a duplicate client retry is deduplicated rather than double-applied; a steer is never silently lost under delivery pressure; a stop is ordered against pending steers.

Failure families: `SteerDropped`, `SteerOutOfOrder`, `LedgerScanUnbounded`, `NonIdempotentReplay`.

### Slice 6 — agent-wrapper-proof (real runner, local)

Owns the real-runner outside-in proof: a real Bun wrapper driving the real agent over structured stdio — the locally verified `claude` flags are `--print --input-format stream-json --output-format stream-json --include-partial-messages` (no PTY) — streaming to session subjects, and steered through the mediated path, proven locally and as manual verbatim pairs. The real wrapper exercises the trusted tier (its own minted leaf-scoped credential from Slice 4); the untrusted tier is already carried by Slice 2's stand-in proof and is not re-proven here. Confirms the canonical structured-message shape holds against the real producer, and that the deferred raw-terminal mode is genuinely unnecessary for the use case. This slice does not enter the CI gate suite; it extends the manual proof surface.

Boundary completeness and outside-in proof: a real session is observed and steered end-to-end locally; the manual records the exact command/outcome pairs. Where the manual-verbatim runner cannot execute an interactive session pair, the coupling to that runner is named and the runner extension or a scripted-pair alternative is owned here, not left implicit. Failure-family ownership respects the no-live-agent-in-CI decision: `StreamJsonParseFailure` is owned by a CI-runnable wrapper decode test over recorded stream-json frames (no live agent); `WrapperLaneUnproven` and `ManualDivergence` are owned by the manual-pair surface and exempt from the per-layer real-NATS rule. Pre-check before RED: the local `claude` CLI and the named flag set execute; if not, the slice stops and escalates rather than substituting a different agent or faking the proof. The wrapper lives under `apps/wrapper`.

Failure families: `StreamJsonParseFailure`, `WrapperLaneUnproven`, `ManualDivergence`.

### Slice 7 — web-session-surface and release closure

Canonical workflow topic and task-doc name: `web-session-surface`.

Owns the direct-NATS browser observation surface and its composition with the browser isolation model, in two hops. Hop one: the trusted shell exchanges its HttpOnly cookie session at a substrate mint endpoint for an ephemeral viewer credential (bearer-mode, short-TTL, loopback-source-pinned, live-revocable; leaf scope: subscribe on its own deliver subject fed by a substrate-bound consumer over the session stream, publish to command acceptance; no JetStream API authority), then connects to the embedded NATS WebSocket through a cookie-gated same-origin upgrade path served by the binary — the binary's shell HTTP server owns the WS route, validates the cookie session, and proxies the upgraded connection to the embednats loopback WebSocket listener; nats-server's `jwt_cookie` websocket option is NOT the mechanism (the viewer credential travels in CONNECT; the cookie gates only the upgrade), so the credential is unusable without the cookie and the cookie is unusable without the credential. Viewer credential renewal reuses Slice 4's renew handshake; per-session authorization is NATS denied-neighbor enforcement, not new endpoint logic. If the binary does not yet establish a real HttpOnly cookie session at the shell, this slice owns establishing it (consistent with the pinned cookie-session-backed service-worker posture) rather than inventing a substitute or stalling. Hop two: the trusted shell forwards session frames to untrusted app content only over the existing leased frame channel, and the frame lease is extended with a session observation scope alongside its command allowlist — the lease is the app-scoped token whose imports (observable sessions) and exports (steering commands) are minted, enforced, and revoked through the existing lease lifecycle. App-piece steering rides the proven path unchanged: typed content intent, shell acceptance, command acceptance into Slice 5's mediated steering. Untrusted app content never holds any credential. Also owns program closure: registering the new gates and outside-in surfaces, adding scenario-matrix entries for every new session surface, and extending the centralized release-evidence manifest so the program can prove itself and cannot silently weaken its own gates — including recording the retirement of the direct-browser-WebSocket deferral with its condition citations.

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

Scenario-matrix: every new outside-in session surface is added to the scenario matrix so the scenarios gate sees it; a surface that is never added must not pass the gate vacuously. This obligation is a post-check in every slice that adds an outside-in surface: Slices 2, 3, 4, 5, and 7 (Slice 6 is local/manual and exempt). gate:scenarios supports no N/A entry and resolves citations only against committed Go tests: each new surface must cite a committed Go test for all seven pinned families, so slices shape their denial proofs to fill all seven per surface, and Slice 7's cited cases must be Go-side tests (WS upgrade gating, cookie denial, viewer denied-neighbor), never TS or agent-browser runs.

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
