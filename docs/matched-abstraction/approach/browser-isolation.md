---
layer: approach
topic: browser-isolation
references:
  - ./endgame-app.md
  - ./browser-frontend-mediator.md
  - ./go-substrate.md
---

# Browser Isolation Approach

Diagram: https://diashort.apps.quickable.co/d/2a2abd49

## Scope

Browser isolation owns the trust boundary between generated frontend artifacts and the managed Tinkabot browser substrate.

The v1 browser isolation model is an opaque-origin sandboxed iframe for generated artifacts, a trusted shell and dedicated worker for mediation, a Go-owned gateway for cookie-backed browser substrate endpoints, and a server-owned scoped service worker used only as bootstrap, cache, and material facade.

Generated artifacts render materialized state and emit typed intents. They do not receive NATS credentials, bearer tokens, subjects, permission material, publish APIs, subscribe APIs, substrate cookies, or service-worker registration authority.

## Core Thesis

The browser boundary must isolate untrusted code by browser primitive first, then by typed protocol. Schema validation is necessary but not enough. The generated frame must be unable to read substrate cookies or same-origin storage, unable to register a worker, unable to call raw substrate endpoints, and unable to smuggle authority through message fields.

The trusted shell binds a generated frame through `WindowProxy` or `MessagePort`, a nonce, frame lease, artifact revision, schema revision, and capability context. `event.origin` alone is not an authority signal for opaque sandboxed frames.

All mutation authority stays behind the server gateway and Command Acceptance. The browser substrate may accelerate material observation, but it does not let generated content become a control-plane participant.

## Layer Contract

Approach owns the browser isolation model, authority invariants, non-goals, and Plan-readiness gates.

Plan owns decomposition into frame lease, message channel, shell mediation, gateway mutation path, service-worker lifecycle, artifact serving policy, and browser proof obligations.

Task owns one executable proof slice at a time, including RED browser assertions, implementation evidence, effective header checks, and release evidence.

## Invariants

Generated artifacts run in an iframe sandbox that allows scripts but does not allow same-origin access. Unsafe same-origin `allow-scripts` plus `allow-same-origin` is rejected for untrusted generated content.

The generated frame communicates only over a leased `postMessage` or `MessagePort` channel. The trusted side validates source window or port identity, nonce, frame lease, schema revision, artifact revision, capability context, and message shape before forwarding anything.

The trusted shell and dedicated worker are the only browser components that may hold browser authority. Generated content never receives raw NATS vocabulary, credentials, subjects, permissions, auth headers, cookies, or a publish/subscribe surface.

Gateway mutations require Command Acceptance plus CSRF, origin, fetch-metadata, session, lease, capability, artifact revision, stale-revision, revocation, idempotency, and attribution checks.

The service worker is server-owned and scope-limited. It may support bootstrap, cache, and material reads, but it must not hold NATS credentials, bearer tokens, raw subjects, permission material, or independent mutation authority.

Service-worker registration identity is app and scope based. Session, lease, capability, and revision authority are checked by the server on every substrate endpoint and mediated message path rather than treated as browser registration identity.

Generated-content isolation may be strengthened by separate origins for multitenancy, but separate origin does not replace typed mediation, frame lease checks, CSP, or gateway acceptance.

## Non-Goals

- No service worker as NATS credential, token, subject, permission, or independent mutation holder.
- No generated-content direct NATS WebSocket access.
- No same-origin path or service-worker scope isolation as the primary boundary for untrusted generated JavaScript.
- No credentialless iframe as the primary security control while browser support remains limited.
- No HTTP-only cookie iframe model without browser lifecycle, CSRF, origin, fetch-metadata, and service-worker scope proof.
- No backend script sandbox implementation decision in this Approach.

## Decision Hierarchy

Browser primitive isolation outranks protocol convenience.

Opaque iframe execution plus leased channel identity outranks origin-string trust.

Server gateway and Command Acceptance outrank frontend mutation convenience.

No credential in generated content outranks realtime shortcutting.

Server-owned service-worker setup outranks generated-content cache or network convenience.

Effective browser proof outranks contract-only evidence for release readiness.

## Plan-Readiness Gate

Plan work may proceed only when it preserves these gates:

- Generated artifacts use opaque sandboxed execution by default.
- Frame communication is leased, nonce-bound, revision-bound, schema-bound, and capability-bound.
- Unsafe same-origin sandbox tokens are denied for untrusted content.
- The shell and dedicated worker mediate all generated-content intents.
- Gateway mutation endpoints reject raw generated-content requests without trusted shell context and Command Acceptance.
- Cookie-backed endpoints enforce CSRF, origin, fetch-metadata, no credentialed CORS to generated origins, session, lease, revision, and capability checks.
- Service-worker scope, script URL, `Service-Worker-Allowed`, cache policy, update policy, and revocation behavior are controlled by the server.
- Browser proof validates effective iframe sandbox, CSP, frame headers, service-worker scope, and denial behavior in a real browser.
- Direct browser NATS WebSocket remains deferred until live credential reload, post-connection revocation, denied-neighbor, stale-access, and confidentiality proof exist.
