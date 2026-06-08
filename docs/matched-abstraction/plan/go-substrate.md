---
layer: plan
topic: go-substrate
references:
  - ../approach/go-substrate.md
  - ./endgame-app.md
---

# Go Substrate Plan

Diagram: https://diashort.apps.quickable.co/d/4a99eb1d

## Consumed Approach

This Plan consumes `go-substrate` as authority and inherits the Endgame App Plan. Go owns live substrate behavior: embedded NATS lifecycle, NATS-native HA/scale posture, auth rendering, stores, activation ledger, gateway substrate, process boundary, credential leases, revocation enforcement, attribution, and sandbox-ready execution contracts.

Existing edge-bootstrap evidence proves a related browser/credential edge, but it does not define Go substrate core authority.

## Decomposition

Go substrate decomposes into eight units:

| Unit | Owns |
| --- | --- |
| Core lifecycle | embedded NATS start/stop/health, topology mode, clustering posture, JetStream readiness, replica/quorum posture, connection policy, shutdown, and critical substrate errors |
| Auth render | NATS-shaped account/user/principal output, permissions, imports, exports, bounded responses, credential lease mint/revoke, and provenance |
| Store substrate | KV/Object/Stream access, keys, buckets, revision checks, stream positions, and durable error mapping |
| Activation ledger | accepted activation records, dedupe keys, chain state, loop-suppression records, leases, cursors, and replay/catch-up support |
| Activation source router | request/reply, subject subscriptions, KV/Object/Stream watches, schedule sources, source leases, cursor state, and activation normalization |
| Process boundary | command, cwd, env, framed stdio RPC attachment, timeout, cancellation, kill, cleanup, attribution, and future Docker envelope |
| Gateway substrate | server-side artifact manifest fetch, digest check, object namespace, CSP/frame/sandbox/cache enforcement, browser bootstrap support, and Browser Edge policy handoff |
| Attribution trail | execution, denial, cleanup, gateway, auth, process, and store events with provenance and typed origin |

## Sequencing

First executable slice: `go-substrate-core`.

`go-substrate-core` establishes the substrate kernel: embedded NATS lifecycle facade, HA/scale topology envelope, auth render facade, credential lease facade, activation ledger shape, store facade, process boundary config, gateway substrate config, and typed errors. It may use fakes or in-memory stores, but the interfaces and tests must match live NATS and process semantics.

After `go-substrate-core`, `activation-source-router` turns request/reply, ordinary subjects, KV/Object/Stream watches, and schedule sources into accepted activation records with source leases and cursor attribution. After that, `script-materializer-loop` can consume Go substrate contracts instead of inventing substrate behavior. Later slices attach real NATS auth rendering, live activation stores, process execution, artifact serving, HA/scale proof, and outside-in release proof.

All Go substrate units coordinate through the same principal, session, lease, revision, chain, and typed-error envelope; later live attachments may vary implementation but not caller contracts.

## Handoff Contracts

Every Go substrate Task receives:

- canonical contract schema id and fixture set.
- managed auth policy and subject taxonomy expectations.
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
- untyped errors at Go boundaries.
- local memory as durable substrate proof except inside a named fake.

## Verification Strategy

Inside-out Go tests own Go substrate errors:

| Layer | Failure families |
| --- | --- |
| Core lifecycle | embedded config invalid, NATS unavailable, cluster route unavailable, JetStream unavailable, replica policy invalid, quorum unavailable, drain or shutdown failure, substrate critical |
| Auth render | auth render invalid, wildcard overreach, lease mint denied, lease revoked, lease expired, permission compile failure |
| Store substrate | bucket missing, key missing, revision mismatch, write conflict, deleted record, stream cursor failure |
| Activation ledger | duplicate activation, stale activation, loop suppressed, lease acquisition failure, replay cursor failure |
| Activation source router | source subscription denied, watch cursor invalid, schedule lease missing, source duplicate, source loop suppressed |
| Process boundary | process config invalid, start failed, protocol unavailable, resource denied, timeout, cancel failed, kill failed, cleanup failed |
| Gateway substrate | artifact missing, digest mismatch, namespace denied, MIME denied, CSP/frame/sandbox missing, cache policy invalid, lease denied |
| Attribution trail | attribution missing, event write failed, unknown transformed to critical |

Cross-language parity remains required where Go consumes schema fixtures. Go tests must prove not only schema validity, but policy decisions and rendered substrate outputs.

Outside-in NATS proof is not required for `go-substrate-core`, but the Task must keep interfaces close enough that a later real embedded NATS slice can replace fakes without changing caller contracts. Fake topology must preserve NATS-provided HA/scale semantics: cluster membership, JetStream replica/quorum readiness, stream position behavior, WebSocket credential posture, and degraded readiness.

## Escalation

Escalate to Approach if a Go Task needs raw script NATS access, browser content credentials, provider-shaped auth authority, schema drift, bespoke HA/scale behavior outside NATS-provided mechanisms, process execution without explicit env/path/IO/cleanup, or live NATS happy paths without denial proof.

Escalate within Plan if `go-substrate-core` cannot define embedded topology, auth, store, ledger, process, and gateway contracts without implementing all live infrastructure at once, or if `activation-source-router` cannot stay above script execution.
