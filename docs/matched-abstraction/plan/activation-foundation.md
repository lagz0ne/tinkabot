---
layer: plan
topic: activation-foundation
status: active
references:
  - ../approach/endgame-app.md
  - ../approach/go-substrate.md
  - ./endgame-app.md
  - ./go-substrate.md
---

# Activation Foundation Plan

Diagram: https://diashort.apps.quickable.co/d/32dc325b

## Consumed Approach

This Plan consumes the Endgame App Approach and Go Substrate Approach as authority. Activation is normalized causality between sources and script execution. Request/reply is one source, not the model. Subject subscriptions, KV watches, Object Store watches, streams, and schedules must enter through the same activation authority with schema, source principal, source lease, cursor, dedupe, chain, loop safety, and attribution.

This is one activation program, split into task-owned proofs. The Plan rejects a single implementation patch that mixes schema authority, NATS permissions, ledger persistence, source watching, scheduling, script execution, and materialization. That shape would hide failure ownership and make the later release gate weaker.

The Go substrate remains the live authority for embedded NATS, NATS-native HA/scale posture, NATS auth rendering, store access, activation ledger storage, process boundaries, gateway substrate, and attribution. This Plan does not reopen those Approach decisions.

## Decomposition

Activation foundation decomposes into six units:

| Unit | Owns |
| --- | --- |
| Contract authority | canonical activation source kinds, payload envelope, source principal and lease envelope, cursor, observed source, chain, dedupe, provenance, capability, fixtures, TS/Zod parity, and Go validation parity |
| Ledger durability | accepted activation records, duplicate resolution, loop suppression, replay/catch-up cursor, source position, lease binding, restart behavior, and typed ledger errors |
| Source authority | source-scoped NATS principal shape, permissions, imports, exports, bounded responses, deny-neighbor behavior, revocation, and source lease enforcement |
| Live source router | request/reply, ordinary subject, KV, Object Store, and stream source adapters that normalize source events into ledger acceptance attempts |
| Schedule engine | durable schedule state, lease/leadership, fake-clock behavior, tick dedupe, catch-up, restart recovery, and loop safety |
| Release proof | outside-in NATS scenarios that compose source authority, ledger acceptance, router behavior, schedule behavior, denial, duplicate, stale, revoked, malformed, and attributed-failure cases |

## Source Model

Canonical activation source kinds are:

| Kind | Required position |
| --- | --- |
| `request_reply` | request id, concrete subject, optional bounded reply subject |
| `command_acceptance` | command id, accepted command subject, artifact/frame revision context |
| `subject` | observed subject, subscription pattern, message id or dedupe key |
| `kv` | bucket, key, revision, operation, watch revision, resume cursor |
| `object` | bucket, object name, digest or revision, object meta-stream sequence recorded by the adapter, watch position |
| `stream` | stream, consumer or durable name, message metadata stream sequence, consumer sequence, subject, delivery attempt |
| `schedule` | schedule id, tick id, due time, owner principal, leader epoch, fencing token, acquired time, expiry time, clock id, clock position |

Subjects are concrete values or concrete wildcard patterns under an authoritative left-side prefix. Wildcards describe a source aperture; they are not placeholders. A source may observe a wildcard pattern, but every accepted activation records the concrete observed subject or store coordinate that caused it.

Every source kind carries a `sourcePrincipal` and `sourceLease` envelope with principal id, lease id, lease status, source id, source kind, app revision, schema revision, script revision when bound, and authority reference. Every source kind also carries cursor or position, chain, dedupe key, provenance, capability context, observed time, and script revision compatibility.

Schema validity proves shape only. Contract authority may reject malformed source principal or lease shape, but active, revoked, expired, stale, denied-neighbor, and wildcard-overreach decisions belong to source authority or ledger durability as named by the fixture.

## Sequencing

First executable slice: `activation-contract-authority`.

`activation-contract-authority` extends canonical schema, fixtures, SDK validation, and Go validation for every activation source kind and the shared source principal, source lease, cursor, and provenance envelope. It proves the authority packet before Go router code exists.

`activation-ledger-durability` now makes the ledger ready for restart-safe source positions, duplicate resolution, loop suppression records, replay/catch-up, source lease binding, and typed durable write/cursor failures through a core store contract with embedded NATS JetStream KV proof.

After the ledger, `activation-source-authority` compiles source-scoped NATS authority from managed auth. It uses NATS vocabulary: `permissions.publish`, `permissions.subscribe`, `allow`, `deny`, `allow_responses`, imports, and exports. It must prove denied-neighbor subjects, revoked or expired leases, and wildcard overreach before source watchers are live.

After source authority, `activation-router-live-sources` attaches live request/reply, ordinary subject, KV, Object Store, and stream sources to embedded NATS. The router does not execute scripts, materialize projections, serve artifacts, render auth, or start NATS. It normalizes source events and asks the activation ledger to accept them.

`activation-schedule-engine` follows live non-time sources unless durable schedule state, lease/leadership, fake-clock tests, tick dedupe, catch-up, restart recovery, and loop safety are ready earlier. Schedule is part of the activation foundation, but it is not allowed to enter as a best-effort timer.

`activation-release-proof` composes the source path through real NATS-mediated behavior and proves the release spine before script-materializer-loop consumes activation at scale.

## Boundary Rules

Activation owns source normalization, source authorization results, cursor preservation, dedupe attempts, loop-suppression handoff, and accepted activation output.

Activation rejects raw script NATS access, generated browser NATS access, materializer truth decisions, artifact serving, script process execution, Docker sandboxing, and Go substrate startup behavior.

The source router receives a source-scoped NATS session or facade, not ambient platform authority. `embednats` runtime access is an adapter boundary; activation narrows it for each declared source principal.

Duplicates and loop suppression belong to the ledger. The router may report the result, but it must not reinvent duplicate or loop policy. Malformed source frames, source watch failures, and source auth failures belong to activation-source routing or source authority.

## Verification Strategy

Inside-out tests own failures at the layer where they originate:

| Layer | Failure families |
| --- | --- |
| Contract authority | source kind invalid, source field invalid, cursor invalid, source principal or lease missing, provenance missing, schema parity mismatch |
| Ledger durability | duplicate activation, stale cursor, replay cursor failure, loop suppressed, lease binding failed, durable write conflict |
| Source authority | source auth denied, source lease revoked, source lease expired, permission compile failed, wildcard overreach, denied neighbor |
| Live source router | router config invalid, request/reply listen failed, subject subscribe failed, KV watch failed, object watch failed, stream consume failed, source malformed, router critical |
| Schedule engine | schedule lease missing, schedule lease lost, clock invalid, tick duplicate, catch-up failed, restart recovery failed |
| Release proof | unresolved lower failure loses attribution, outside-in denial mismatch, live NATS behavior diverges from inside-out contract |

Consumed errors need explicit boundary decisions:

| Consumer | Consumes | Decision |
| --- | --- | --- |
| Contract authority | current schema and fixtures | Transform invalid source shape and parity mismatch into contract errors; keep schema-valid policy fixtures tagged with their later owner layer |
| Ledger durability | store substrate and auth lease state | Resolve duplicate and loop-suppressed records; transform cursor, lease, and durable write failures |
| Source authority | managed auth and subject taxonomy | Resolve denied-neighbor and wildcard overreach as denial outputs; transform compile failures |
| Live source router | embedded NATS adapter, source authority, ledger | Propagate adapter unavailability, transform source failures, and return ledger acceptance or suppression results |
| Schedule engine | ledger, source authority, clock, durable schedule store | Resolve duplicate ticks, transform lease/clock/catch-up failures, and never emit best-effort ticks |
| Release proof | all activation units | Propagate typed lower failures into attributed outside-in results |

RED starts at contract authority, not router code. The first failing tests must show that the current schema accepts only `request_reply` and `command_acceptance`, then require `subject`, `kv`, `object`, `stream`, and `schedule` with source principal, source lease, cursor, wildcard, owner-layer tags, and parity behavior.

GREEN is complete only when the new activation packet passes contract parity and the next task can consume it without inventing schema, source authority, or cursor fields.

## Escalation

Escalate to Approach if activation needs raw script NATS access, generated browser credentials, placeholder subjects, provider-shaped auth authority, schema drift, local memory as durable source truth, schedule ticks without durable lease/catch-up behavior, or release confidence based only on happy-path NATS connectivity.

Escalate within Plan if source authorization cannot be expressed with NATS auth vocabulary, if the router cannot narrow runtime authority per source, if the ledger cannot own duplicate and loop outcomes, or if schedule behavior cannot be tested with deterministic time and restart recovery.
