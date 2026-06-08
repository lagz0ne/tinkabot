---
layer: approach
topic: go-substrate
references:
  - ./endgame-app.md
---

# Go Substrate Approach

Diagram: https://diashort.apps.quickable.co/d/4a99eb1d

## Purpose

The Go substrate is the server-side authority layer for Tinkabot's managed NATS platform. It embeds and manages NATS as the default substrate, owns live NATS lifecycle, NATS-auth-shaped enforcement, durable NATS stores, activation ledgers, artifact gateway substrate, process boundaries, execution attribution, and the later Docker/sandbox enforcement path.

Go is not a schema sidecar and not only a policy mirror. TypeScript may provide SDK contracts and browser-facing helpers, but Go owns the substrate surfaces that become operationally real.

## Scope

Go substrate owns embedded NATS lifecycle and connection policy: start, stop, health, JetStream availability, store access, account/user surfaces, route or listener policy, and WebSocket-facing support where the browser worker needs it.

Go substrate owns the HA and scale mission through NATS-native mechanisms: server clustering, JetStream replication and quorum, route/gateway/leaf topology where needed, queue or consumer distribution where needed, and observable readiness for degraded or recovering topology. It does not invent a parallel consensus, routing, or storage-replication layer when NATS already provides the mechanism.

Go substrate owns auth rendering from canonical managed-auth policy into NATS-shaped accounts, users or principals, permissions, imports, exports, bounded responses, credential leases, rotation, revocation, and denial attribution.

Go substrate owns durable NATS stores for activation, scripts, manifests, projections, artifacts, execution records, leases, and cleanup records. Store names, keys, revisions, and stream positions are substrate concerns once they become live infrastructure.

Go substrate owns execution boundaries: script process command, path, env, process protocol attachment, lifecycle, timeout, cancellation, resource envelope, audit context, and future Docker/sandbox placement.

Go substrate owns server-side gateway boundaries for artifact retrieval and browser bootstrap support. Browser Edge can consume sanitized bootstrap shape and render-time policy, but Go owns the server substrate that mints or denies credentials and serves or denies artifact material.

## Layer Contract

Approach owns Go substrate purpose, authority boundaries, non-goals, invariants, and Plan-readiness gates. It may decide what Go must own and what Go must reject, but it does not choose task sequencing or file-level implementation.

Plan owns Go substrate decomposition, sequencing, handoff contracts, and verification strategy under this Approach and the Endgame App Approach.

Task owns one executable Go substrate proof. Task may implement packages, tests, and evidence for one bounded substrate unit. Task may not redefine NATS auth vocabulary, schema authority, browser trust, script mediation, or materialized product truth.

## Non-Goals

- No TypeScript ownership of live substrate behavior.
- No provider-shaped auth as domain authority.
- No script raw NATS access by default.
- No generated browser content credentials, subjects, tokens, publish APIs, or subscribe APIs.
- No external-only NATS deployment assumption unless it preserves the same embedded-substrate lifecycle, auth, store, HA, scale, and attribution contract.
- No process sandbox promise without explicit path, env, IO, resource, identity, and cleanup contracts.
- No live NATS happy path as release confidence without denied, stale, duplicate, revoked, malformed, and attributed-failure proof.
- No materialized product truth inside Go substrate unless the materializer layer accepts it.

## Invariants

Go consumes canonical contracts; it does not invent lane-local DTO authority. Schema validation remains neutral, and capability policy remains separate from shape validity.

Go treats embedded NATS as a managed platform component, not a best-effort development helper. Single-node mode is a topology choice; HA/scale mode must map to NATS-provided clustering, JetStream replicas, quorum, leaf/gateway, and operational readiness semantics.

Go renders NATS auth vocabulary. Grants and denials speak in accounts, users or principals, permissions, imports, exports, allow, deny, bounded responses, and lease status.

Go is least-authority by construction. Browser, script, activation, materializer, artifact gateway, and internal control surfaces receive separate authority envelopes.

Go preserves provenance. Every live credential, store write, activation record, process run, gateway decision, denial, and cleanup record carries principal, session, lease, app revision, schema revision, artifact or script revision when present, and chain context.

Go owns revocation as enforcement, not decoration. Revoked or expired leases cannot mint new credentials, run processes, write effects, or keep gateway authority.

Go process execution remains sandbox-compatible before Docker exists. The non-sandboxed path must already declare execution target, environment, IO, lifecycle, resource, identity, cleanup, and audit behavior.

Go substrate failures are typed at the substrate boundary. Unknowns become substrate critical errors with origin, operation, and cause. Lower storage, auth, gateway, process, and NATS errors resolve or transform at their owning Go layer.

## Decision Hierarchy

NATS auth vocabulary outranks Go-local access concepts.

Canonical schemas outrank Go structs.

Explicit leases outrank ambient process or browser authority.

NATS-native HA/scale outranks bespoke substrate replication or routing.

Durable NATS store positions outrank local memory for substrate truth.

Typed attribution outranks opaque logs.

Sandbox-ready process contracts outrank quick execution shortcuts.

## Plan-Readiness Gate

Plan work may proceed only when it preserves these gates:

- Go is the owner for embedded NATS lifecycle, NATS-native HA/scale posture, auth rendering, credential leases, store access, process boundaries, gateway substrate, activation ledger, and execution attribution.
- Contract Authority, Managed Auth, Subject Taxonomy, Command Acceptance, and Browser Edge outputs remain consumed inputs, not redefined shapes.
- Every Go-owned authority path has allowed and denied cases, including revoked, expired, stale, malformed, duplicate, and denied-neighbor behavior where applicable.
- HA/scale proof remains based on NATS-provided clustering, JetStream replica/quorum, route/gateway/leaf, queue/consumer, and observability behavior.
- Script execution remains mediated through a facade and process protocol, not ambient NATS.
- Browser Edge receives only scoped worker authority and content-safe bootstrap output.
- Future Docker/sandbox enforcement can be added without changing the script behavior contract.
- Release proof remains able to connect these inside-out Go proofs to live NATS-mediated outside-in scenarios.
