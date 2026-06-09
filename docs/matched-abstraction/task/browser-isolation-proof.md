---
layer: task
topic: browser-isolation-proof
references:
  - ../approach/browser-isolation.md
  - ../plan/browser-isolation.md
  - ./browser-frontend-dedicated-worker.md
  - ./substrate-edge-bootstrap.md
---

# Browser Isolation Proof Task

Diagram: https://diashort.apps.quickable.co/d/2a2abd49

## Objective

Create the first release-shaped browser isolation proof: a trusted shell embeds generated content in an opaque sandboxed iframe, leases a message channel, rejects raw authority, forwards accepted typed intents to the gateway/Command Acceptance path, and proves effective browser headers plus service-worker scope denial behavior.

## Scope

This task owns:

- real shell and generated iframe fixture.
- iframe sandbox policy with scripts allowed and same-origin denied.
- leased `postMessage` or `MessagePort` channel with source, nonce, frame lease, schema revision, artifact revision, and capability validation.
- trusted shell mediation from generated intent to canonical browser command intent.
- gateway mutation smoke path through Command Acceptance.
- browser-observed denial for raw NATS vocabulary, bad nonce, wrong frame source, stale revision, revoked lease, and unsafe sandbox tokens.
- service-worker script/scope/header proof sufficient to deny wrong scope and prevent generated content from receiving token, subject, credential, permission, or registration authority.

## Non-Goals

- No direct browser NATS WebSocket.
- No generated-content service-worker registration.
- No service-worker NATS credential holder.
- No per-app origin fleet.
- No credentialless iframe as the required isolation control.
- No script execution or materializer implementation beyond the smallest status/projection needed for the browser proof.

## Acceptance Contract

- Generated content is loaded in an iframe whose effective sandbox denies same-origin access while allowing script execution.
- Generated content cannot read substrate cookies, local storage, parent state, NATS credentials, subjects, tokens, permissions, or publish/subscribe APIs.
- The trusted shell accepts messages only from the leased source window or port with the expected nonce, frame lease, schema revision, artifact revision, and capability context.
- Raw NATS-shaped vocabulary and disallowed command names are rejected before gateway transport.
- Accepted typed intents are stamped with trusted shell context and reach Command Acceptance as canonical browser command intents.
- Gateway mutation checks reject bad CSRF, bad origin, bad fetch metadata, stale revision, revoked lease, and generated-origin credentialed CORS.
- Service-worker setup proves exact script URL, exact scope, safe `Service-Worker-Allowed`, non-overlap with generated artifact scope, no token material to generated content, and denial for scope/session/revision mismatch.
- Browser-visible denials preserve typed error family and attribution.

## RED Artifact

Expected failing proof before implementation:

- `T-BROWSER-IFRAME-SANDBOX`: real browser fixture shows generated iframe is not yet loaded with effective opaque sandbox policy.
- `T-BROWSER-MESSAGE-LEASE`: bad nonce, wrong source window, stale frame lease, and raw authority vocabulary are not yet denied by a real shell channel.
- `T-BROWSER-COMMAND-ACCEPTANCE`: accepted generated intent is not yet observed through the canonical Command Acceptance path.
- `T-BROWSER-GATEWAY-CSRF`: bad CSRF, bad Origin, bad Fetch-Metadata, revoked lease, and stale revision are not yet denied at the gateway boundary.
- `T-BROWSER-SW-SCOPE`: wrong service-worker scope, broad `Service-Worker-Allowed`, stale worker revision, and generated-content worker registration are not yet denied in a real browser proof.
- `T-BROWSER-ATTRIBUTION`: browser-visible denials do not yet preserve typed error family, frame id, artifact revision, session, lease, and cause.

## Execution Notes

Keep the first slice narrow but real. Use a generated iframe fixture and a trusted shell fixture before adding product UI. Use browser automation to prove effective sandbox, headers, storage/cookie denial, message binding, and service-worker behavior. Use Go and contract tests for gateway policy branches that are difficult to force from browser automation alone.

The frontend-owned isolation layer is tracked in `docs/matched-abstraction/task/frontend-isolation-layer.md`. Gateway CSRF/origin/fetch-metadata and service-worker lifecycle proof remain under this broader browser isolation proof.

Do not attach direct browser NATS WebSocket until live credential reload and post-connection revocation are proven.

## Verification Evidence

Task prep evidence:

- `triage-three browser isolation model` -> `3 pusher angles converged on opaque sandbox plus trusted shell mediation; 3 challenger passes confirmed; arbiter score 34`.
- `curl -s -X POST https://diashort.apps.quickable.co/render ...` -> `https://diashort.apps.quickable.co/d/2a2abd49`.
- `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`.
- `bun run test:layers` -> `Ran 10 tests ... OK`.
- `git diff --check` -> `clean`.

## Wrap-Up Announcement

The browser isolation proof is complete when a real browser proves generated content is opaque, leased, credentialless, and unable to bypass the trusted shell, while accepted typed intents still reach backend Command Acceptance and service-worker setup remains server-owned, scoped, token-free, and denial-tested.
