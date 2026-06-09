---
layer: plan
topic: go-substrate
status: active
approach_seal: go-substrate@2026-06-08
references:
  - ../approach/go-substrate.md
  - ./endgame-app.md
  - ./activation-foundation.md
---

# Go Substrate Plan

Diagram: https://diashort.apps.quickable.co/d/5edab343

## Consumed Approach

This Plan consumes the sealed `go-substrate` Approach as authority and inherits the Endgame App Plan. Go owns live substrate behavior: embedded NATS lifecycle, NATS-native HA/scale posture, auth rendering, stores, activation ledger, gateway substrate, cookie-backed browser sessions, scoped service-worker bootstrap, process boundary, credential leases, revocation enforcement, attribution, and sandbox-ready execution contracts.

Plan work may refine decomposition, handoff, sequencing, and verification. It may not redefine embedded NATS ownership, NATS-native HA/scale, NATS auth vocabulary, authority envelopes, mediated scripts, generated-content denial, or typed substrate failures.

Existing edge-bootstrap evidence proves a related browser/credential edge, but it does not define Go substrate core authority.

## Decomposition

Go substrate decomposes into nine units:

| Unit | Owns |
| --- | --- |
| Core lifecycle | embedded NATS start/stop/health, topology mode, clustering posture, JetStream readiness, replica/quorum posture, connection policy, shutdown, and critical substrate errors |
| Embedded NATS adapter | live embedded server lifecycle, JetStream enablement, account/auth load path, WebSocket listener posture, topology probes, drain/shutdown behavior, and adapter error mapping |
| Auth render | NATS-shaped account/user/principal output, permissions, imports, exports, bounded responses, credential lease mint/revoke, and provenance |
| Store substrate | KV/Object/Stream access, keys, buckets, revision checks, stream positions, and durable error mapping |
| Activation ledger | accepted activation records, dedupe keys, chain state, loop-suppression records, leases, cursors, and replay/catch-up support |
| Activation foundation | activation source contracts, source-scoped authority, durable ledger/cursor behavior, request/reply, subject, KV/Object/Stream, schedule sources, and activation normalization |
| Process boundary | command, cwd, env, framed stdio RPC attachment, timeout, cancellation, kill, cleanup, attribution, and future Docker envelope |
| Gateway substrate | server-side artifact manifest fetch, digest check, object namespace, CSP/frame/sandbox/cache enforcement, browser isolation policy, browser bootstrap support, cookie-backed session setup, scoped service-worker script serving, and Browser Edge policy handoff |
| Attribution trail | execution, denial, cleanup, gateway, auth, process, and store events with provenance and typed origin |

## Sequencing

First executable slice: `go-substrate-core`.

`go-substrate-core` establishes the substrate kernel: embedded NATS lifecycle facade, HA/scale topology envelope, auth render facade, credential lease facade, activation ledger shape, store facade, process boundary config, gateway substrate config, and typed errors. It may use fakes or in-memory stores, but the interfaces and tests must match live NATS and process semantics.

After `go-substrate-core`, `embedded-nats-adapter` attaches those contracts to a real embedded NATS runtime without changing caller contracts. It proves the single-node live path and preserves topology hooks for later HA/scale proof.

After `embedded-nats-adapter`, `activation-foundation` first expands canonical activation contracts for every source kind, then proves durable ledger/cursor behavior, source-scoped NATS authority, live source routing, schedule behavior, and release proof. The live router is a unit inside that foundation, not the first authority boundary. After the activation foundation is verified, `script-materializer-loop` can consume Go substrate contracts instead of inventing substrate behavior. Later slices attach process execution, artifact serving, HA/scale proof, and outside-in release proof.

All Go substrate units coordinate through the same principal, session, lease, revision, chain, and typed-error envelope; later live attachments may vary implementation but not caller contracts.

## Core Handoff

`go-substrate-core` hands these contracts to `embedded-nats-adapter`:

| Surface | Handoff |
| --- | --- |
| Lifecycle | topology mode, readiness state, drain/shutdown contract, degraded-state vocabulary, and critical error shape |
| Auth render | NATS-shaped accounts/users/principals, permissions, imports/exports, bounded responses, lease identity, revocation state, and wildcard denial result |
| Store substrate | bucket/key/stream names, revision expectations, object digest expectations, cursor positions, and durable error mapping |
| Activation ledger | activation id, dedupe key, source lease, chain state, loop-suppression state, replay position, and stale-state denial |
| Process boundary | executable target, cwd, env projection, framed RPC mode, timeout/resource envelope, cancel/kill/cleanup contract, and run attribution |
| Gateway substrate | object namespace, digest, object-read authority, MIME/CSP/frame/sandbox/cache policy, browser isolation policy, Browser Edge handoff, cookie/session/scope context, and lease binding |
| Attribution | event envelope, provenance fields, source layer, operation, cause, and unknown-to-critical transform |

`go-substrate-core` rejects live server startup, activation-source watching, script execution, materialized projection writes, Docker enforcement, and release-level proof.

## Handoff Contracts

Every Go substrate Task receives:

- canonical contract schema id and fixture set.
- managed auth policy and subject taxonomy expectations.
- sealed Approach id and any peer Plan/Task references consumed as inputs.
- embedded NATS topology mode, route/gateway/leaf posture, JetStream replica/quorum expectations, and degraded-readiness expectations.
- principal, session, lease, app revision, schema revision, chain, and artifact or script revision context.
- NATS auth vocabulary output shape.
- typed error set and origin layer.
- allowed, denied-neighbor, malformed, duplicate, stale, revoked, expired, cleanup, and attributed-failure expectations.

Every Go substrate Task rejects:

- TypeScript-only authority.
- raw script NATS authority by default.
- generated browser content authority.
- provider-specific auth as domain truth.
- activation-source behavior inside core lifecycle, auth, store, process, or gateway units.
- untyped errors at Go boundaries.
- local memory as durable substrate proof except inside a named fake.

## Verification Strategy

Inside-out Go tests own Go substrate errors:

| Layer | Failure families |
| --- | --- |
| Core lifecycle | embedded config invalid, NATS unavailable, cluster route unavailable, JetStream unavailable, replica policy invalid, quorum unavailable, drain or shutdown failure, substrate critical |
| Embedded NATS adapter | server start failed, client connect failed, JetStream enable failed, auth load failed, WebSocket unavailable, topology probe failed, drain timed out, adapter critical |
| Auth render | auth render invalid, wildcard overreach, lease mint denied, lease revoked, lease expired, permission compile failure |
| Store substrate | bucket missing, key missing, revision mismatch, write conflict, deleted record, stream cursor failure |
| Activation ledger | duplicate activation, stale activation, loop suppressed, lease acquisition failure, replay cursor failure |
| Activation foundation | source kind invalid, source lease missing, source auth denied, watch cursor invalid, schedule lease missing, source duplicate, source loop suppressed |
| Process boundary | process config invalid, start failed, protocol unavailable, resource denied, timeout, cancel failed, kill failed, cleanup failed |
| Gateway substrate | artifact missing, digest mismatch, namespace denied, MIME denied, CSP/frame/sandbox missing, unsafe sandbox token, cache policy invalid, CORS policy invalid, lease denied |
| Browser session bootstrap | session invalid, cookie policy invalid, service-worker scope denied, service-worker script denied, origin check failed, fetch metadata denied |
| Attribution trail | attribution missing, event write failed, unknown transformed to critical |

Cross-language parity remains required where Go consumes schema fixtures. Go tests must prove not only schema validity, but policy decisions and rendered substrate outputs.

Outside-in NATS proof is not required for `go-substrate-core`, but the Task must keep interfaces close enough that `embedded-nats-adapter` can replace fakes without changing caller contracts. Fake topology must preserve NATS-provided HA/scale semantics: cluster membership, JetStream replica/quorum readiness, stream position behavior, WebSocket credential posture, and degraded readiness.

## Traced Test Ownership

Each unit owns its declared errors and either resolves, transforms, or propagates consumed errors at its boundary:

| Consumer | Consumes | Required acknowledgment |
| --- | --- | --- |
| Embedded NATS adapter | core lifecycle contracts and NATS runtime failures | Transform runtime failures into adapter errors; propagate core contract invalidity unchanged |
| Auth render | canonical schemas, managed auth, subject taxonomy | Transform invalid authority into auth render errors; resolve denied-neighbor and wildcard overreach as denial outputs |
| Store substrate | embedded adapter connection and NATS KV/Object/Stream failures | Transform NATS storage failures into store substrate errors; propagate adapter unavailability |
| Activation ledger | store substrate, auth lease, chain context | Resolve duplicate and loop-suppressed activations; transform stale, lease, cursor, and write failures into ledger errors |
| Activation foundation | canonical contracts, source authority, adapter subscriptions, store watches, schedule leases, activation ledger | Transform contract and source failures at their owning unit; propagate adapter unavailability; resolve duplicate and loop outcomes through the ledger |
| Process boundary | activation ledger, auth lease, process runtime | Transform process runtime failures into process errors; resolve cancellation and cleanup outcomes explicitly |
| Gateway substrate | store substrate, auth lease, Browser Edge policy, browser isolation policy, browser session policy | Transform object/auth/session/isolation/policy failures into gateway errors; resolve cache, frame policy, CORS policy, and service-worker scope eligibility before serving |
| Attribution trail | all unit events | Transform event-write failure into attribution error; wrap unknowns as substrate critical |

Plan verification is complete only when every declared error in the table above has one owning Task test and every cross-unit consumed error has an explicit Resolve, Transform, or Propagate decision.

## Escalation

Escalate to Approach if a Go Task needs raw script NATS access, browser content credentials, provider-shaped auth authority, schema drift, bespoke HA/scale behavior outside NATS-provided mechanisms, process execution without explicit env/path/IO/cleanup, or live NATS happy paths without denial proof.

Escalate within Plan if `go-substrate-core` cannot define embedded topology, auth, store, ledger, process, and gateway contracts without implementing all live infrastructure at once, if `embedded-nats-adapter` cannot attach live NATS without changing caller contracts, or if `activation-foundation` cannot keep source contracts, source authority, ledger durability, live routing, and scheduling separated before script execution.
