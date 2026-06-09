---
layer: plan
topic: browser-frontend-mediator
references:
  - ../approach/browser-frontend-mediator.md
  - ../approach/browser-isolation.md
  - ../approach/nats-script-runtime.md
  - ./browser-isolation.md
  - ./nats-script-runtime.md
---

# Browser Frontend Mediator Plan

## Consumed Approach

This plan consumes `docs/matched-abstraction/approach/browser-frontend-mediator.md` and `docs/matched-abstraction/approach/browser-isolation.md` as authority. The carried decisions are shell-owned browser authority, dedicated-worker mediation, opaque generated iframe execution, leased message channels, server-owned service-worker setup, generated content as receiver/intent emitter, no raw NATS subject bridge, observed materializer state, and backend-owned command effects.

## Decomposition

Plan units:

- Message contract: typed content-to-mediator command intents, mediator-to-content projection/status/error messages, and denial of raw NATS vocabulary.
- Mediator contract: worker-side command intake that validates content messages, checks allowed command names, stamps trusted context, and hands command intents to an injected transport.
- Materializer contract: shell-side state reducer that accepts sanitized projection/status messages and ignores stale projection updates.
- Dedicated-worker bridge contract: a small binding that connects `message` events to the mediator and posts accepted/error/status messages back to the shell.
- Browser isolation contract: real shell, opaque sandboxed generated iframe, leased `postMessage` or `MessagePort` channel, nonce/source/revision checks, and effective CSP/frame/sandbox evidence.
- Service-worker bootstrap contract: a later server-owned proof that registers a scoped worker through cookie-backed session bootstrap without exposing tokens or NATS credentials to generated content.
- Later vertical proof: browser shell, generated iframe, real dedicated worker, service-worker substrate, artifact gateway, and optional browser NATS WebSocket smoke test after revocation proof.

## Handoff Contract

The first completed Task unit received:

- Scope: message contract, mediator, materializer store, and dedicated-worker bridge only.
- Non-goals: no real browser app, no real NATS WebSocket client, no credential minting, no artifact gateway, no service-worker implementation in this first proof, no generated iframe implementation.
- Inputs: trusted frontend context, allowed command names, fake transport, and fake worker scope.
- Outputs: exported TypeScript API, RED tests, verification evidence, and updated layer docs.

## Verification Strategy

The first proof must show:

- A generated content command becomes a mediated command intent with trusted context.
- Raw NATS-shaped fields are rejected before transport.
- Disallowed command names are rejected before transport.
- Materializer projection updates are monotonic.
- The dedicated-worker bridge turns accepted content messages into posted accepted status and invalid messages into posted error messages.

The browser isolation proof must add live browser worker behavior, opaque iframe sandbox proof, leased channel denial, service-worker cookie-session bootstrap, effective CSP/origin/fetch-metadata checks, and gateway Command Acceptance smoke proof.

Real NATS WebSocket server permission checks remain a later transport proof. They do not gate the v1 browser isolation boundary.

## Escalation Log

Return to Approach if generated content needs raw subjects, direct NATS client access, service-worker NATS ownership, optimistic durable browser state, or a command path that bypasses backend durable acceptance.
