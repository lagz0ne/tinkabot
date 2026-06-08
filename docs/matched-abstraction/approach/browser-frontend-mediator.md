---
layer: approach
topic: browser-frontend-mediator
references:
  - ./charter.md
  - ./nats-script-runtime.md
---

# Browser Frontend Mediator Approach

## Scope

Tinkabot's browser frontend mirrors the backend script runtime boundary. Generated browser content does not receive raw NATS access, NATS subjects, reply subjects, credentials, or permission material. A trusted managed frontend owns the glue: a shell, a dedicated worker mediator, and materializer state for rendering observed artifacts.

The generated artifact is a receiver and intent emitter. It may render projections and ask for typed commands, but it cannot publish to NATS directly and cannot choose NATS subjects.

Diagram: https://diashort.apps.quickable.co/d/f396f133

## Core Thesis

The frontend mediator preserves realtime NATS value without making generated content a control-plane participant. The trusted shell owns frame identity and rendering. The dedicated worker owns NATS-facing command mediation. The materializer owns observed state presented to generated content.

## Layer Contract

Approach owns the browser trust boundary, mediator vocabulary, authority split between trusted shell/worker and generated content, service-worker non-default decision, and materializer truth invariant.

Plan owns decomposition into message, mediator, materializer, worker bridge, and later vertical proof contracts.

Task owns one executable proof slice, such as validating generated-content messages and routing them through a fakeable dedicated-worker bridge.

## Decision Hierarchy

The trusted shell and dedicated worker are the only browser components allowed to hold or use browser NATS authority.

Generated content communicates through typed browser IPC. It sends command intents and receives sanitized projection/status messages. It never receives a publish API, subject string, NATS client, credential, auth headers, or permission object.

The dedicated worker is the default mediator shape for v1. It can keep NATS off the UI thread, own an in-memory lease, and be terminated on revocation. Service workers are not the default NATS mediator because they are origin-wide, persistent, and lifecycle-managed by the browser.

All frontend command effects remain backend-owned. The frontend mediator publishes command intents only to the approved backend command path. Durable acceptance, idempotency, status, chain attribution, and side effects remain backend activation concerns.

Materializer state is observed truth, not optimistic local truth. Realtime events are invalidations or status updates. Durable snapshots and artifact manifests remain authoritative.

## Non-Goals

- No generated-content NATS client.
- No raw subject bridge such as "publish this subject".
- No service-worker NATS ownership in v1.
- No durable browser-only state as the source of truth.
- No artifact gateway, Vite builder, browser credential issuer, or live NATS WebSocket implementation in this slice.

## Plan-Readiness Gate

Plan can proceed when it preserves these invariants:

- The dedicated worker is a mediator, not a general browser plugin surface.
- Generated content can only submit schema-shaped command intents.
- The mediator stamps shell-owned context: session, capability, artifact, revision, chain, and frame identity.
- Raw NATS vocabulary is denied at the generated-content boundary.
- Materializer state accepts monotonic observed projection/status messages and ignores stale updates.
- Worker tests use fake ports/transports. Real browser and live NATS WebSocket proof belongs to a later vertical slice.
