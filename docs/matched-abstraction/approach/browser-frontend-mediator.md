---
layer: approach
topic: browser-frontend-mediator
references:
  - ./charter.md
  - ./nats-script-runtime.md
  - ./browser-isolation.md
---

# Browser Frontend Mediator Approach

## Browser Isolation Supersession

`docs/matched-abstraction/approach/browser-isolation.md` is the current v1 browser authority. The proven v1 mutation path is gateway-owned Command Acceptance over cookie-backed substrate endpoints; direct browser NATS WebSocket, including the dedicated worker holding a live NATS connection, is deferred until live credential reload, post-connection revocation, denied-neighbor, stale-access, and confidentiality proofs exist.

This document still owns the generated-content mediation vocabulary, the trusted shell/worker authority split, and the dedicated-worker proof history. It no longer decides the v1 browser model. Where this document names the dedicated worker as the default v1 NATS mediator shape, read that as the deferred WebSocket direction, not current authority.

## Scope

Tinkabot's browser frontend mirrors the backend script runtime boundary. Generated browser content runs in an opaque sandboxed frame and does not receive raw NATS access, NATS subjects, reply subjects, credentials, substrate cookies, service-worker registration authority, or permission material. A trusted managed frontend owns the glue: a shell, a dedicated worker mediator, server-owned service-worker setup, and materializer state for rendering observed artifacts.

The generated artifact is a receiver and intent emitter. It may render projections and ask for typed commands, but it cannot publish to NATS directly and cannot choose NATS subjects.

Diagram: https://diashort.apps.quickable.co/d/f396f133

## Core Thesis

The frontend mediator preserves realtime NATS value without making generated content a control-plane participant. The trusted shell owns frame identity, leased message channels, and rendering. The dedicated worker owns NATS-facing command mediation. The service worker, when installed, is registered through a server-owned cookie session and scoped app surface; it is a bootstrap, cache, and material facade rather than an authority holder. The materializer owns observed state presented to generated content.

## Layer Contract

Approach owns the browser trust boundary, mediator vocabulary, authority split between trusted shell/worker/service-worker setup and generated content, service-worker session boundary, and materializer truth invariant.

Plan owns decomposition into message, mediator, materializer, worker bridge, and later vertical proof contracts.

Task owns one executable proof slice, such as validating generated-content messages and routing them through a fakeable dedicated-worker bridge.

## Decision Hierarchy

The trusted shell and dedicated worker are the only browser components allowed to hold or use browser NATS authority.

Generated content communicates through typed browser IPC. It sends command intents and receives sanitized projection/status messages. It never receives a publish API, subject string, NATS client, credential, auth headers, or permission object.

Generated content uses an iframe sandbox that allows scripts and denies same-origin access. The trusted shell rejects unsafe same-origin sandbox tokens for untrusted generated content and treats opaque-origin messages as untrusted until source window or port, nonce, frame lease, schema revision, artifact revision, capability context, and message shape all pass.

The dedicated worker is the default NATS mediator shape for v1. It can keep NATS off the UI thread, own an in-memory lease, and be terminated on revocation. Service workers are not default NATS credential holders because they are origin/scope persistent and lifecycle-managed by the browser.

Service-worker setup is still a first-class browser substrate concern. The trusted shell may register a server-served worker under a server-approved scope after the server establishes an HttpOnly, Secure, SameSite cookie session. The worker uses same-origin, cookie-backed substrate endpoints for bootstrap, cache, and material flow. Mutation flow still requires trusted-shell context and backend Command Acceptance. The worker does not receive bearer tokens, raw NATS credentials, subject lists, permission material, or independent mutation authority.

All frontend command effects remain backend-owned. The frontend mediator publishes command intents only to the approved backend command path. Durable acceptance, idempotency, status, chain attribution, and side effects remain backend activation concerns.

Materializer state is observed truth, not optimistic local truth. Realtime events are invalidations or status updates. Durable snapshots and artifact manifests remain authoritative.

## Non-Goals

- No generated-content NATS client.
- No raw subject bridge such as "publish this subject".
- No service-worker NATS ownership or token storage in v1.
- No service-worker scope chosen by generated content.
- No generated-content `allow-same-origin` for untrusted same-origin artifacts.
- No durable browser-only state as the source of truth.
- No artifact gateway, Vite builder, browser credential issuer, or live NATS WebSocket implementation in this slice.

## Plan-Readiness Gate

Plan can proceed when it preserves these invariants:

- The dedicated worker is a mediator, not a general browser plugin surface.
- Generated content runs in an opaque sandboxed frame unless a stronger separate-origin model is proven by Plan.
- The shell binds generated-content IPC by source window or port, nonce, frame lease, schema revision, artifact revision, and capability context.
- Generated content can only submit schema-shaped command intents.
- The mediator stamps shell-owned context: session, capability, artifact, revision, chain, and frame identity.
- Service-worker registration is server-owned, cookie-session-backed, scope-limited, and revocable.
- Raw NATS vocabulary is denied at the generated-content boundary.
- Materializer state accepts monotonic observed projection/status messages and ignores stale updates.
- Worker tests use fake ports/transports. Real browser and live NATS WebSocket proof belongs to a later vertical slice.
