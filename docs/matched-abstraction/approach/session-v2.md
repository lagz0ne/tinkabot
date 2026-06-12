---
layer: approach
topic: session-v2
references:
  - ./endgame-app.md
  - ./go-substrate.md
  - ./browser-isolation.md
---

# Session V2 Approach

Diagram: https://diashort.apps.quickable.co/d/13b0196d

## Purpose

Tinkabot v1 mediates scripts as run-to-completion processes whose effects materialize after exit. Session V2 adds a second, co-equal execution shape: a long-lived agent session. A wrapper process (driving an agent runner such as `claude` over structured stdio) stays alive, streams its output continuously, and accepts steering input mid-run. The substrate exposes that session over NATS for observation and steering, and a trusted browser surface streams the session and sends steering chat.

This is not a refinement of the script materializer loop. The script loop is batch: one activation, one run, end-of-run effects. A session is interactive and unbounded in time: it is observed while it runs and influenced while it runs. The two share the activation front door and the authority model, but a session is its own execution subsystem.

## Core Thesis

A session has two lifetimes, and conflating them is the central design error this Approach exists to prevent.

The **session record** — identity, provenance, and the durable transcript of everything the agent has emitted — lives in NATS-native storage and is the authority. It survives substrate restart and is always recoverable by replay.

**Liveness** — the running wrapper process, its leased credential, and its open steering pipe — is ephemeral. It is an external OS process that dies with the binary or on its own fault. Liveness is not authority; it is a leased, heartbeat-bound claim against a session record.

Recovery is therefore reconciliation, not a restart policy. On every start the substrate reconciles live claims against durable records. Whether a session resumes, is fenced, or is terminated with a completion record derives from the state of its liveness lease at reconciliation time. The tactical questions the design must answer — orphaned children, poisoned run claims, credential re-mint, terminal records on the crash path — are all consequences of this single framing, not independent features.

## Scope

Session V2 owns the long-lived session execution shape and the authority that surrounds it: how a wrapper is started and supervised, how its output becomes durable observable truth, how steering reaches it, how it is credentialed, and how the browser observes and steers it. It consumes — and does not redefine — the Go Substrate Approach (embedded NATS ownership, operator/JWT auth vocabulary, separated authority planes, mediated scripts, generated-content denial, typed substrate failures) and the Endgame App Approach (the product loop, materialized truth, scoped leases, control/app plane separation). The Browser Isolation Approach remains authority for the trusted-shell / opaque-content split; this Approach adds an observation-and-steering surface to it without weakening it.

The canonical session is a stream of **structured messages** (stream-json: typed token, chunk, and status frames). Raw terminal byte streams are not the canonical shape; a PTY raw mode, if ever built, is a deferred non-canonical option behind the same frame contract.

The substrate is the boundary, not the redactor. Session output is persisted as it is emitted; its confidentiality is exactly that of the transcript itself. Secret handling and content filtering are out of scope here and belong to a later hook feature. Frame mediation in this Approach is integrity and resource control (schema validity, per-session quota), never privacy.

## Invariants

These are pinned. Plan and Task may decide how to satisfy them, never whether to.

1. **Single authorized writer for steering.** Exactly one principal — the session runner — publishes to a session's steering subject. No other principal, including the trusted wrapper itself, may publish steering or status. The runner re-checks the steerer's lease status at the moment it applies a steer, not only when the steer was accepted, so a revoked lease cannot steer through a durable message accepted before revocation.

2. **Trust is control-plane attribution.** A wrapper's trust tier and the scope of any credential minted for it are decided by the control plane at mint time and bound into the credential. No app-plane-writable record (script record, KV value, frame field) grants or escalates authority. Schema validity never grants effect authority. The unconditional denial of raw NATS and CLI imports is not relaxed by trust; a trusted wrapper publishes through mediation, never through raw NATS authority.

3. **Least-authority leaf scope.** Session credentials enumerate concrete leaf subjects. A wrapper credential carries publish on its session ingest subject (the input to output mediation) and subscribe on its session steering subject, and nothing else. Wildcard-subtree grants over a session's subject space are denied at mint. The mint path itself enforces subject breadth; scope is never assumed safe because it names one session.

4. **One mediated publisher for output.** A single validating republisher is the only writer of a session's output subject. It enforces the frame contract — which NATS subject permissions cannot, because they gate where a principal may publish, never what — and a per-session resource quota that replaces the bounded-output defense the batch runner provided. No raw bridge publishes session content directly to durable storage.

5. **Sessions are a distinct execution subsystem.** Long-lived sessions do not run on the single-goroutine, run-to-completion activation runner, whose contract is a mandatory wall-clock timeout and end-of-run effect validation. A long-lived session never blocks or starves activation processing. Session lifecycle (idle and explicit termination) is its own class, not a wall-clock kill.

6. **The session record is recovery authority; liveness is a reconcilable lease.** The durable session record and transcript are authoritative and always recoverable by replay. Liveness is a heartbeat-bound lease that the substrate reconciles against the record on start. Every termination path, including substrate or wrapper crash, resolves to a terminal record through reconciliation. No session outcome is left implicit.

7. **Steering is attributed and lossless.** Steering that originates outside the substrate flows through command acceptance and activation so every steer carries identity, capability, and ordering provenance. The delivery path from acceptance to the wrapper's input must not silently drop, and must not cost work proportional to total activation history.

8. **Browser observation is direct NATS over a two-step minted viewer credential, and composes with browser isolation.** The trusted shell observes sessions over the embedded NATS WebSocket directly, and NATS itself is the only authorization engine — no parallel enforcement surface is built. Custody is two-step: the durable authority is the HttpOnly cookie session, never readable by page script; the shell exchanges it at a substrate mint endpoint for an ephemeral viewer credential — control-plane-minted, bearer-mode with no signing seed in the browser, short-TTL, source-pinned to loopback, live-revocable, and leaf-scoped to concrete subjects (subscribe on its own deliver subject fed by a substrate-bound consumer, publish to command acceptance; no JetStream API authority). The WebSocket upgrade itself is cookie-gated by the substrate, so the ephemeral credential is unusable without the cookie and the cookie is unusable without the credential. Renewal rides the same renew handshake as wrapper credentials. This retires the direct-browser-NATS-WebSocket deferral by meeting its named conditions — live credential reload, post-connection revocation, denied-neighbor, and stale-access are proven by the operator/JWT authority work; confidentiality is satisfied at the loopback posture, and TLS remains required before any external exposure, which stays deferred. Untrusted app content never holds any credential: it receives session frames only through the trusted shell's leased frame channel, and steers only through typed content intents accepted by the shell and command acceptance. The frame lease is the app-scoped authority token — it carries the app's observation scope (which sessions it may watch) and its steering command allowlist, and is minted, enforced, and revoked through the existing lease lifecycle.

## Non-Goals

- Built-in secret redaction, transcript encryption, or content filtering. Transcript confidentiality equals transcript sensitivity; filtering is a later hook feature.
- Raw terminal / PTY fidelity as a canonical shape. Structured stream-json is canonical; raw terminal is a deferred optional mode.
- Real agent-runner execution inside CI gates. The real wrapper is proven locally and by manual verbatim pairs; CI proves the subsystem with a deterministic stand-in. No new CI lane or live-agent gate is taken on now.
- Multi-viewer fanout tuning, transcript compaction tuning, session resume UX, session list/CRUD UI, multi-node operation, and external (non-loopback, TLS) exposure. These stay deferred and the release manifest must keep naming them deferred.

## Layer Contract

This Approach owns the intent of the session shape, the authority invariants above, the canonical session vocabulary choice, the recovery framing, and the non-goals. It owns the decision authority for what "trusted" grants and who declares it, for whether the browser streams through a gateway or directly, and for whether sessions reuse or stand apart from the activation runner.

This Approach does not own decomposition, slice sequencing, failure-family enumeration, subject naming, file work, or commands. Those belong to the Session V2 Plan, which consumes this Approach. Where this Approach and the Go Substrate or Endgame App Approaches appear to conflict, the sealed Go Substrate Approach wins on substrate mechanics and the Endgame App Approach wins on the product loop; this Approach may only add the session shape within those constraints, never reopen them.

## Decision Hierarchy

1. Go Substrate Approach (sealed): embedded NATS, auth vocabulary, authority planes, typed failures.
2. Endgame App Approach: product loop, materialized truth, scoped leases, plane separation.
3. Browser Isolation Approach: trusted shell vs opaque content authority split.
4. This Approach (Session V2): the session execution shape and its authority invariants.
5. Session V2 Plan: decomposition, sequencing, verification strategy, failure families.
6. Session V2 Tasks: one executable unit each, with RED proof.

A lower layer may cite a higher one; it may never redefine it.

## Plan-Readiness Gate

The Plan may begin when, and only when, the following are true:

- The canonical session shape is fixed as structured stream-json messages, with raw terminal explicitly deferred. (Decided.)
- "Trusted" is defined as control-plane attribution bound at mint, declared by the control plane, never relaxing the raw-import denial. (Decided.)
- Browser observation is fixed as direct NATS over a two-step minted, leaf-scoped, ephemeral viewer credential behind a cookie-gated WebSocket upgrade, retiring the direct-browser-WebSocket deferral by its own named conditions; untrusted app content stays credential-free behind the leased frame channel. (Decided.)
- Long-lived sessions are fixed as a distinct subsystem, not a mode of the activation runner. (Decided.)
- Recovery is fixed as reconciliation between a durable session record and a heartbeat-bound liveness lease, with restart behavior derived from lease state rather than a global policy. (Decided.)
- The eight invariants and the non-goals above are accepted as carried decisions, not open questions.

All readiness conditions are met. The Plan may proceed to decompose, sequence, and assign failure families, and must honor: complete-at-boundary slices with denial and failure paths; denied-neighbor proof before any session subject watcher goes live; a centralized release-evidence closure that the manifest cannot weaken; and scenario-matrix entries for every new outside-in surface.
