---
layer: plan
topic: browser-isolation
references:
  - ../approach/browser-isolation.md
  - ./endgame-app.md
  - ./browser-frontend-mediator.md
  - ./go-substrate.md
---

# Browser Isolation Plan

Diagram: https://diashort.apps.quickable.co/d/2a2abd49

## Consumed Approach

This Plan consumes `docs/matched-abstraction/approach/browser-isolation.md` as authority. The carried decision is the v1 opaque sandbox model: generated artifacts run in an iframe sandbox with scripts allowed and same-origin access denied, the trusted shell and dedicated worker mediate a leased message channel, all mutations pass through the server gateway and Command Acceptance, and the service worker remains a server-owned scoped bootstrap, cache, and material facade.

This Plan also consumes the Endgame App, Browser Frontend Mediator, and Go Substrate Plans for lane ownership. Those Plans define command acceptance, gateway substrate, browser edge, and frontend mediator responsibilities; this Plan defines how those responsibilities compose into a browser isolation proof.

## Decomposition

The browser isolation proof decomposes into six units:

| Unit | Owns |
| --- | --- |
| Frame sandbox policy | iframe sandbox tokens, blocked same-origin access, frame headers, artifact revision binding, and generated-content denial surface |
| Leased message channel | `WindowProxy` or `MessagePort` binding, nonce, frame lease, schema revision, artifact revision, capability context, and source validation |
| Trusted shell mediation | material/status delivery, generated intent validation, raw authority denial, context stamping, and dedicated-worker handoff |
| Gateway mutation path | cookie-backed session checks, CSRF, origin, fetch-metadata, no credentialed CORS to generated origins, Command Acceptance handoff, idempotency, stale/revoked denial, and attribution |
| Service-worker substrate | server-owned script serving, exact scope, safe `Service-Worker-Allowed`, update policy, cache policy, revocation, and scope/session/revision mismatch denial |
| Browser evidence | effective browser headers, real iframe behavior, service-worker registration behavior, message-channel denial behavior, and browser-visible attributed errors |

## Sequencing

The first isolation slice should prove the browser boundary before introducing direct browser NATS WebSocket. It starts with a real trusted shell, a generated iframe fixture, a leased message channel, and a gateway command acceptance smoke path.

Service-worker proof may share the same slice only if it stays scoped to bootstrap, cache, material, and mismatch denial. It must not introduce NATS credentials, raw subjects, or mutation authority into the worker.

Per-app origin partitioning is a strengthening option after the v1 proof. It may be added when multitenancy or same-origin browser APIs need stronger blast-radius separation, but it does not replace leased message channels or gateway acceptance.

Trusted worker NATS WebSocket is a transport upgrade after v1. Observe-only subscriptions require scoped subscription, denied-neighbor, stale-access, and confidentiality proof. Command-intake publish requires fixed-subject authority, live post-connection revocation, reload behavior, and backend acceptance proof.

## Handoff Contracts

Every browser isolation Task receives:

- app, session, lease, artifact revision, schema revision, frame id, frame nonce, and capability context.
- sandbox token policy, CSP policy, frame policy, artifact serving policy, and generated-content egress policy.
- service-worker script URL, scope, `Service-Worker-Allowed`, cache policy, update policy, session policy, and revocation context when the Task crosses service-worker setup.
- command intent schema, acceptance status schema, denial envelope, and attribution envelope.
- required denied cases for raw NATS vocabulary, bad nonce, wrong frame source, stale artifact revision, revoked lease, unsafe sandbox token, bad CSRF, bad origin, bad fetch metadata, wrong service-worker scope, and stale worker revision.

Each Task rejects generated-content cookies, generated-content service-worker registration authority, generated-content raw substrate endpoint access, origin-string-only message trust, broad service-worker scope, broad credentialed CORS, and happy-path-only browser proof.

## Verification Strategy

Inside-out tests own typed policy decisions:

| Owner | Failure families |
| --- | --- |
| Frame sandbox policy | unsafe sandbox token, frame policy denied, artifact revision mismatch, generated-content storage/cookie access denied |
| Leased message channel | bad nonce, wrong source window or port, stale frame lease, schema mismatch, capability mismatch |
| Trusted shell mediation | raw authority vocabulary, disallowed command, stale projection, untrusted material event |
| Gateway mutation path | invalid session, CSRF denied, origin denied, fetch metadata denied, credentialed CORS denied, stale revision, revoked lease, acceptance denied |
| Service-worker substrate | script denied, scope denied, broad allowed scope, stale worker revision, cache policy denied, revocation denied |
| Browser evidence | effective header mismatch, iframe escape, worker registration mismatch, attributed browser error missing |

Outside-in browser proof must use a real browser to verify effective sandbox, CSP, frame headers, cookie/storage denial, message channel binding, and service-worker scope behavior. Contract-only tests remain useful, but they do not make browser isolation release-green.

Outside-in NATS proof is required only when an accepted browser intent crosses into Command Acceptance, Activation, or materialized state. The proof should observe the accepted or denied result through NATS-visible status, stream, KV/Object Store, or projection surfaces.

## Escalation

Escalate to Approach if a Task needs generated-content credentials, generated-content raw NATS, service-worker NATS ownership, same-origin sandbox bypass, mutation endpoints callable without trusted shell context, or release confidence based only on contract tests.

Escalate within Plan if the browser proof cannot produce effective header evidence, if service-worker lifecycle behavior cannot be tested without weakening isolation, or if direct NATS WebSocket becomes necessary before live revocation and stale-access proof exist.
