---
layer: plan
topic: endgame-app
references:
  - ../approach/endgame-app.md
  - ./platform-structure.md
  - ./code-structure.md
  - ./nats-script-runtime.md
  - ./script-nats-cli-proof.md
  - ./nats-script-runtime-traced-tdd.md
  - ./browser-frontend-mediator.md
  - ./browser-isolation.md
---

# Endgame App Plan

Diagram: https://diashort.apps.quickable.co/d/c826d60e

## Consumed Approach

This Plan consumes `docs/matched-abstraction/approach/endgame-app.md` as authority. Tinkabot is a complete managed NATS application platform with a trusted embedded browser substrate, managed backend script substrate, durable NATS material, schema-backed contracts, managed NATS auth, centralized operations, and verification through both inside-out ownership tests and outside-in real-NATS proof.

Peer Plans remain lane evidence. `platform-structure` carries Go substrate, Vite frontend, and schema/SDK ownership. `code-structure` records the current repository baseline. `nats-script-runtime` carries script mediation and activation decisions. `nats-script-runtime-traced-tdd` carries prior typed-error discipline for script runtime. `browser-frontend-mediator` carries the dedicated-worker mediation proof. `browser-isolation` carries the v1 opaque iframe, leased channel, gateway mutation, and service-worker proof model. None of those peer Plans supersede this endgame coordination Plan.

## Decomposition

The endgame splits into ten lanes:

| Lane | Owns |
| --- | --- |
| Contract authority | Neutral schema source, provenance envelope, shared fixtures, schema parity semantics, generated or checked Zod, TypeScript, and Go validation targets |
| Managed auth | Domain identity, ownership, session, revision, capability provenance, NATS accounts/principals, permissions, imports, exports, leases, rotation, revocation, and deny-neighbor behavior |
| Subject taxonomy | Authority prefixes, control-plane and app-plane namespaces, reserved surfaces, wildcard admissibility, import/export subject mapping, and denied-neighbor cases |
| Go substrate | NATS lifecycle, stores, auth rendering, connection policy, process boundary, gateway boundary, cookie-backed browser sessions, scoped service-worker bootstrap, activation ledger storage, artifact gateway substrate, attribution, and future sandbox boundary |
| Browser edge | Session bootstrap, service-worker registration/setup contract, browser credential lease, worker principal, credential mint/revoke, artifact serving, cache/CSP/frame sandbox policy, opaque iframe policy, leased channel policy, and browser-to-NATS handoff |
| Frontend substrate | Trusted shell, dedicated worker, scoped service-worker consumer surface, generated-content opaque sandbox, sub-app principals, leased typed intent emission, materialized rendering, and brokered cross-sub-app communication |
| Command acceptance | Durable browser/script/caller intent acceptance, rejection, idempotency, stale-revision checks, status materialization, and handoff into activation |
| Activation | Request/reply, subject, KV, Object Store, stream, and schedule sources normalized into activation intents with dedupe, cursor, lease, chain, and loop controls |
| Script runtime | Script records, metadata, framed process protocol, runtime facade authority, execution principal lease, advanced capability mediation, attribution, denial, cleanup, and sandbox-compatible process contracts |
| Materializer/artifact | Materializer-owned durable projections and product truth; artifact-owned bundle/blob manifests, revisions, digests, isolation, serving policy, build results, invalidations, and stale artifact behavior |
| Centralized ops and release | One operational entry surface, local services, code generation, tests, builds, packaging, evidence manifests, repeatable NATS scenarios, and release gate orchestration |

## Dependency Ordering

Contract authority is first. Every other lane depends on the same provenance envelope, revision vocabulary, schema ids, validation semantics, and fixture model.

Managed auth and subject taxonomy follow the first contract slice. NATS permissions must preserve domain identity, ownership, revision, session, capability provenance, namespace separation, and denied-neighbor behavior before Go, browser, script, or activation code can safely rely on them.

Centralized ops starts early after contract/auth shape exists. It may define stable operation names and evidence manifests before all lanes are implemented, but it cannot present missing gates as passing work.

Go substrate and Browser Edge start after contract, auth, and subject taxonomy are consumable. They may proceed in parallel only through the same principal, session, lease, artifact revision, command intent, acceptance status, chain context, and denial envelope.

Browser service-worker setup sits across Go substrate and Browser Edge. Go owns the cookie session, service-worker script serving, scope headers, revocation, and validation endpoints. Browser Edge owns the sanitized registration contract consumed by the trusted shell. Generated content only consumes the scoped app surface and never receives tokens or NATS credentials.

Browser isolation sits across Browser Edge and Frontend substrate. Browser Edge owns effective sandbox, CSP, frame, artifact serving, cookie, CORS, and service-worker policy. Frontend substrate owns the trusted shell, generated iframe fixture, leased message channel, context stamping, and material/status delivery. Generated content is opaque, credentialless, and mutation-inert until the trusted shell and gateway acceptance path validate the request.

Frontend substrate, Script Runtime, Activation, Command Acceptance, and Materializer/Artifact can proceed in parallel only after their shared contract and authority packet is fixed. They cannot invent lane-local DTOs, permission concepts, subject patterns, or revision semantics.

Command Acceptance sits before activation composition for browser and caller intents. A frontend command intent is inert until durable backend acceptance decides idempotency, revision compatibility, authorization, status, and activation handoff.

Activation and Materializer/Artifact can run in parallel once command acceptance exists. Activation owns causality and loop safety; materialization owns durable observed truth.

Script output is not product truth. Script events, outputs, and proposals become product-visible material only after the owning backend acceptance/materializer authority accepts them into a compatible durable projection or artifact revision.

Live outside-in proof comes last for a slice. It composes already-proven inside-out contracts rather than replacing them.

The NATS seam is the release seam. A slice that crosses actors or lanes must prove its observable result through real NATS-mediated behavior before it can count as endgame-ready, even when its inside-out contract tests pass.

Schedule activation is not an early source. It waits for durable schedule state, lease/leadership, fake-clock tests, catch-up behavior, dedupe, restart recovery, and loop-safety proof.

## Parallelization Rules

Do not parallelize competing schema authority, auth vocabulary, subject taxonomy, provenance envelope, or revision semantics.

Parallelize Go and browser work only when both consume the same cross-lane contract packet and emit the same denial and attribution envelopes.

Frontend and script work may run in parallel when both preserve mediation: generated content emits typed intents only, and scripts use process protocol plus runtime facade only.

Activation and materializer work may run in parallel when activation never treats materialized updates as product authority and materializer never treats activation events as durable truth without projection acceptance.

Inside-out test design may run ahead of implementation per lane, but expected behavior must trace to the endgame Approach and this Plan's error ownership.

Fakes are admissible only when they preserve the declared lower-layer semantics: typed errors, denial behavior, revision handling, attribution fields, timeout/cancel behavior, revocation behavior, and the later live NATS proof that validates the fake.

## Handoff Contracts

Every Task receives:

- Consumed Approach and owning Plan lane.
- Contract version, schema ids, and fixture set.
- Authority owner, principal/session/lease context, and capability policy context.
- Browser session, service-worker scope, cookie path/origin, CSRF/origin, and revocation context when the Task crosses browser substrate.
- Frame sandbox, frame lease, nonce, source window or port, schema revision, artifact revision, capability context, CSP, frame policy, and generated-content egress policy when the Task crosses browser isolation.
- Subject prefix, namespace, wildcard, import/export, and denied-neighbor expectations.
- Revision context for app, schema, artifact, script, snapshot, command, and chain.
- Required typed errors and Resolve/Transform/Propagate policy at its boundary.
- Required proof surface: allowed path, denied neighbor, malformed input, duplicate, stale or mismatched revision, revoked lease, loop suppression when applicable, and attributed failure.

Every Task rejects raw NATS by default, placeholder subjects, broad wildcards without authoritative prefix, language-local schema truth, browser-local durable truth, untyped catch-all failures, silent provenance loss, and happy-path-only proof.

Task output must include a RED artifact, implementation evidence, inside-out contract proof, outside-in NATS proof when the slice crosses an app boundary, denied-neighbor proof, no-slop/simplify/review pass notes, and a wrap-up announcement.

## Verification Strategy

The endgame layer graph is the test ownership graph:

Contract Authority -> Managed Auth and Subject Taxonomy -> Go Substrate -> Browser Edge, Frontend Substrate, Command Acceptance, Activation, Script Runtime, and Materializer/Artifact -> Centralized Release Proof.

Each layer owns typed failures:

| Layer | Declared failure families |
| --- | --- |
| Contract authority | Invalid shape, schema drift, missing provenance, fixture mismatch, validation semantic mismatch |
| Managed auth | Unknown identity, denied capability, permission compile failure, revoked lease, stale revision, provenance loss |
| Subject taxonomy | Reserved surface violation, wildcard overreach, namespace collision, import/export mismatch, denied-neighbor violation |
| Go substrate | NATS unavailable, store unavailable, auth render failure, gateway failure, process boundary failure, substrate critical |
| Browser edge | Invalid session, credential mint failure, credential revoke failure, artifact integrity failure, CSP/frame policy failure, sandbox policy failure, cookie/CORS policy failure |
| Frontend substrate | Denied IPC, raw-NATS vocabulary attempt, stale frame revision, bad message lease, unsafe sandbox token, service-worker scope mismatch, sub-app principal mismatch, broker denial |
| Command acceptance | Intent invalid, duplicate command, stale command revision, acceptance denied, status materialization failed |
| Activation | Unauthorized source, dedupe conflict, cursor failure, lease failure, loop suppressed, schedule state failure |
| Script runtime | Record invalid, facade denied, protocol failure, process failure, advanced capability denied, cleanup failure |
| Materializer/artifact | Stale snapshot, digest mismatch, projection rejected, artifact build failed, manifest mismatch |
| Release proof | Unresolved lower failure becomes an attributed response or event; untyped success is never allowed |

Inside-out tests live where their declared error lives. A higher layer may mock a lower layer only through the lower layer's typed contract and must not inspect lower-layer internals.

Outside-in tests prove composed behavior only: accepted path, denied neighbor, malformed input, duplicate, stale revision, revoked credential, loop suppression, and attributed failure over NATS-mediated behavior.

Outside-in tests must use real NATS whenever the slice reaches the system seam. Valid observation surfaces include request/reply, publish/subscribe, streams, KV, Object Store, status subjects, and materialized projections. Fakes may remain inside lower layers only to localize forced branches after the real-NATS path exists.

For script-side slices, outside-in means real `nats` CLI commands against embedded NATS. The test should send requests or publishes through NATS and observe replies, statuses, streams, KV/Object Store changes, or materialized projections through NATS. In-memory substitutes and mocks are acceptable only for narrow impossible-to-force branches after the real NATS path is proven.

## Release Verification Spine

Every release-shaped slice maps one outside-in scenario to the inside-out contracts that make it diagnosable:

| Spine step | Evidence |
| --- | --- |
| Contract | Same schema revision, generated or checked validators, positive and negative fixtures, and accept/reject parity |
| Auth and subjects | Same identity/capability input compiles into NATS-shaped permissions, imports, exports, leases, and denied-neighbor behavior |
| Browser edge | Worker principal is scoped, generated content cannot see credentials or subjects, artifact revision and CSP/frame/sandbox policy are enforced |
| Browser isolation | Generated content runs in an opaque sandboxed iframe, unsafe same-origin sandbox tokens are denied, leased message channel checks pass, and raw authority messages fail closed |
| Service worker setup | Server-owned cookie session is established, worker script is served under the intended scope, generated content receives no token material, and scope/session/revision mismatch is denied |
| Command acceptance | Intent is durably accepted or denied with idempotency, revision, status, chain, and attribution fields |
| Activation | Accepted intent becomes normalized activation with dedupe, source identity, chain bounds, and loop policy |
| Script runtime | Script runs through process protocol and facade, not raw NATS by default, with explicit host authority and cleanup |
| Materializer/artifact | Accepted effects become compatible projections or artifacts with manifest, digest, snapshot revision, and invalidation event |
| Frontend rendering | Browser observes materialized truth and renders the compatible revision without local durable state |
| Release ops | One centralized operation produces the evidence manifest for the full slice |

No release gate passes on generated file existence alone. It passes when the same app revision proves schema parity, auth provenance, denial behavior, typed errors, live NATS behavior, and artifact/materializer compatibility.

No release gate passes on inside-out proof alone. It passes when inside-out ownership and outside-in real-NATS behavior agree, so a failure can be both reproduced from the system seam and attributed to one owning layer.

No script-side release gate passes on process output alone. It passes when a `nats` CLI caller can trigger the behavior and inspect the resulting platform reaction through NATS-visible surfaces.

## Capability Proof Matrix

Every capability surface carries the same proof shape:

| Case | Required outcome |
| --- | --- |
| Allowed | Effect succeeds and attribution names identity, capability, revision, subject or store, and chain |
| Denied neighbor | Adjacent subject, store key, frame, artifact, or command is rejected before effect |
| Malformed | Schema or protocol layer owns the failure |
| Duplicate | Command, activation, or materializer owner resolves or rejects through idempotency and dedupe policy |
| Stale revision | Owning lane fails closed with exact stale or mismatch error |
| Revoked lease | Authority terminates and later effect is denied with revocation attribution |
| Session scope mismatch | Service-worker or browser session setup is denied before generated content receives a usable substrate surface |
| Frame lease mismatch | Generated-content IPC is denied before any trusted shell or gateway effect |
| Loop-suppressed | Activation or materializer owner records suppression without recursive execution |
| Attributed failure | Caller/browser/script-visible result preserves origin and typed error family |

## First Executable Task Slice

Topic: `endgame-contract-authority`.

Purpose: establish the smallest neutral contract packet every later lane must consume: provenance envelope, principal/session/revision/capability references, NATS-auth-shaped permission/exposure/import shape, subject taxonomy fixtures, browser command intent, command acceptance status, activation intent, artifact manifest, materialized projection, and attributed event/error envelope.

RED artifact: failing parity and contract tests that prove those contracts are not yet authoritative across canonical schema, TypeScript/Zod target, Go validation target, and shared fixtures.

GREEN boundary: minimal canonical contracts plus generated or checked validators needed for parity, denied-neighbor, malformed, stale, revoked, and no-raw-authority fixture checks to pass.

Verification boundary: schema parity, SDK contract tests, TypeScript check, layer validation, denied-neighbor fixtures, malformed fixtures, stale/revoked fixtures, and no raw subject or credential fields at generated-content or script-facing boundaries.

Non-goals: no Go substrate implementation, no Vite shell implementation, no browser WebSocket, no sandbox enforcement, no full script runtime migration, no release packaging.

## Escalation Log

Escalate to Approach if a Task needs generated frontend raw NATS, default script raw NATS, provider-shaped auth as domain truth, language-local schema authority, placeholder subjects, local browser truth, sandbox behavior that rewrites script contracts, or release confidence based only on happy paths.

Escalate within Plan if the first contract slice cannot model provenance and denial, if auth backend choice changes portability, if the centralized ops runner changes release semantics, if Go/Vite lanes need different revision envelopes, or if Browser Edge cannot own both artifact serving and browser credential lifecycle.

Completed structure moves are baseline. Do not inherit old sequencing that treats the TypeScript package move as active work.
