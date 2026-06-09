---
layer: approach
topic: endgame-app
references:
  - ./charter.md
  - ./platform-structure.md
  - ./nats-script-runtime.md
  - ./browser-frontend-mediator.md
  - ./browser-isolation.md
---

# Endgame App Approach

Diagram: https://diashort.apps.quickable.co/d/c3d3cbfd

Service-worker session flow: https://diashort.apps.quickable.co/d/f202fd1b

## Purpose

Tinkabot is a complete managed NATS application platform. It combines a trusted embedded browser substrate, server-owned service-worker setup, a managed backend script substrate, NATS-native storage and activation, schema-backed contracts, managed NATS auth, centralized operational entrypoints, and verification that proves behavior both inside-out and outside-in through real NATS-mediated boundaries.

The endgame is not a demo runtime. The system must be releasable as a coherent app whose browser, backend scripts, auth, activation, storage, build, and test surfaces all share one authority model.

The system seam is NATS. Local function calls, fakes, generated files, and browser fixtures may prove ownership inside a layer, but they do not satisfy endgame confidence until the same behavior is observable through real NATS subjects, requests, streams, KV/Object Store records, or materialized projections.

## Core Thesis

The product loop is the unit of design: source and artifacts become durable materialized projections, the browser renders those projections and emits typed intents, the backend accepts or denies those intents, activation runs scripts through mediated authority, and script effects return as attributed events or projections. Realtime NATS accelerates the loop, but durable material and explicit authority define truth.

## Scope

The app owns a managed NATS control plane: identities, ownership, permissions, imports, exports, exposure, revocation, and attribution are expressed in NATS auth vocabulary and mediated by the substrate. Domain identity is a policy input; NATS auth is the compiled enforcement shape.

The browser side is an embedded frontend substrate. A trusted shell, dedicated worker, scoped service worker, and browser edge mediate browser-to-server participation for sub-apps. Generated or supplied sub-app content renders materialized state and emits typed command intents from an opaque sandboxed frame; it does not hold raw NATS authority.

The browser edge owns session bootstrap, service-worker setup, browser credential minting and revocation, artifact serving, cache policy, content security policy, frame sandboxing, and any control-plane behavior that NATS does not provide directly to the browser.

The backend side is a managed script substrate. Scripts are stored, described, activated, executed, audited, and constrained by metadata, schema, and runtime facade policy. Current execution may be trusted, but the contract must preserve a later sandbox path without redefining script behavior.

The storage side uses NATS-native material. Durable manifests, projections, artifacts, activation records, and script records live behind NATS-facing stores and subjects. Browser and script views observe materialized truth rather than inventing local truth.

The operations side has one centralized entry surface for development, test, build, release, and local service orchestration. The exact runner is subordinate to the principle that operational behavior is discoverable, repeatable, and not scattered across ad hoc commands.

The verification side treats NATS as the integration boundary. Outside-in tests prove app behavior from caller/browser/script-facing entrances through real NATS-mediated surfaces. Inside-out tests prove schema, auth, activation, materializer, runtime facade, and substrate contracts before they compose.

## Layer Contract

Approach owns the endgame purpose, authority boundaries, non-goals, invariants, decision hierarchy, reference policy, and Plan-readiness gate for the complete app.

Plan owns decomposition across substrate, auth, frontend, scripts, schemas, materialization, centralized operations, and verification. Plan may decide coordination shape, dependency direction, handoff contracts, and escalation gates only within this Approach.

Task owns one executable proof at a time. Task may implement, test, and report evidence for a bounded Plan unit, but it may not redefine auth authority, browser trust boundaries, script mediation, schema authority, or release confidence rules.

## Non-Goals

- No MVP framing. A narrow slice may be accepted only when its boundary is complete, denied paths included.
- No raw NATS access for generated browser content.
- No token-bearing service worker that becomes an ambient browser credential holder.
- No raw NATS access for backend scripts by default.
- No invented permission vocabulary where NATS auth vocabulary already exists.
- No local browser state as durable product truth.
- No optimistic frontend side effects as authoritative completion.
- No backend script sandbox implementation decision in this Approach; only a contract that keeps script sandboxing possible.
- No scattered build, test, or service scripts.
- No language-local schema authority in Go or TypeScript.
- No treating request/reply as the whole activation model.

## Invariants

NATS is the substrate and auth vocabulary, not a blanket escape hatch. Every actor receives the least NATS-shaped authority needed for its role.

Control plane and app plane are separate authority domains. Scripts, generated browser content, and sub-app sessions cannot import, expose, publish, subscribe, or watch auth, script, schema, build, credential, or materializer-control surfaces unless they are acting through a named control-plane service role.

Mediation is mandatory at trust boundaries. Generated browser content runs in an opaque sandboxed iframe and talks to the trusted shell and worker through leased typed IPC. Backend scripts talk to the runtime through a process protocol and facade. Neither side chooses raw subjects or credentials by default.

Service-worker setup is substrate-owned. The server issues an HttpOnly, Secure, SameSite cookie session, serves the service-worker script under a controlled scope, and sets the allowed worker scope. Browser registration identity is app and scope based; session, lease, capability, and revision authority are server-side checks on every substrate endpoint and mediated message path. The app uses the scoped substrate surface; generated content does not receive bearer tokens, NATS credentials, raw subjects, permission material, substrate cookies, or service-worker registration authority.

Managed auth is lifecycle-aware and provenance-preserving. Issuance, scope, import/export shape, rotation, revocation, denial, and attribution are substrate concerns, and every granted authority must trace from app identity and ownership to a declared exposure, import, or service role.

Browser and script credentials are scoped leases. A lease cannot outlive its session, execution, app revision, artifact revision, or declared capability scope. Revocation must terminate authority rather than merely hide UI.

NATS auth vocabulary is authoritative: `permissions.publish`, `permissions.subscribe`, `allow`, `deny`, `allow_responses`, imports, and exports define access. Deny wins over allow. Response authority is bounded to its invocation context.

Subjects are concrete values or concrete wildcard patterns. Left-side subject tokens carry authority. Wildcards must sit behind a concrete authoritative prefix and may not become placeholders.

Server-side NATS namespace separation is enforced by account or authoritative prefix, not naming convention alone. Browser, script facade, activation, materializer, system, and internal control surfaces have distinct authority envelopes with explicit deny-neighbor behavior.

Schema is the neutral contract authority. Runtime validation is required at every trust boundary. TypeScript, Go, Zod, fixtures, and generated validators derive from or are checked against the same schema source and must agree on accept and reject semantics.

Schema validity does not grant authority. Every trust-boundary message carries both schema context and capability context, and policy decides whether a valid message may produce an effect.

Materialized state is observed truth. Browser views consume durable snapshots, manifests, and monotonic status/projection updates. Realtime events may invalidate, notify, or advance state; they do not replace durable authority.

Materialized content is untrusted data until it crosses a named validation and projection boundary. Trusted shell state, generated content state, artifact manifests, and product truth remain separate.

Generated frontend artifacts are untrusted receiver code with isolated execution identity. Artifact origin, frame identity, revision, digest, integrity, and sandbox policy are part of the authority boundary. The v1 browser isolation model uses an iframe sandbox that allows scripts but denies same-origin access; unsafe same-origin `allow-scripts` plus `allow-same-origin` is rejected for untrusted generated content.

Frame IPC is leased authority, not origin-string trust. The trusted shell validates source window or port identity, nonce, frame lease, schema revision, artifact revision, capability context, and message shape before forwarding generated-content intents.

Service-worker isolation depends on server-controlled origin or path scope. Multiple sub-apps sharing an origin must still receive distinct service-worker scopes, cookie paths or equivalent session partitioning, CSP/frame policy, and command validation. Cookie-backed convenience does not weaken typed intent, CSRF/origin, fetch-metadata, revision, revocation, or capability checks.

Every embedded artifact, frame, or sub-app is an independent principal. Cross-sub-app communication is brokered, schema-shaped, scoped, and attributed; privileged peer-to-peer authority is not a default capability.

Every accepted effect resolves an exact identity, session, artifact revision, script revision, schema version, materialized snapshot revision, and chain context. Stale or mismatched revisions fail closed.

All effects are attributable. Every command, activation, script run, artifact build, materializer update, denial, failure, and cleanup outcome must carry enough identity, chain, source, revision, permission, and timing context to explain why it happened.

Activation sits between event sources and execution. Request/reply, ordinary subjects, KV watches, object changes, streams, and schedules are activation inputs; execution starts from normalized activation intent.

Loop safety is part of activation authority. Wildcards, generated events, materializer notifications, and script outputs must not create unbounded recursive execution.

Chain identity, hop limits, dedupe keys, and loop suppression are cross-plane product invariants spanning browser intent, backend acceptance, activation, script execution, materialization, and embedded rendering.

Trusted script execution is not product authority. Script outputs are mediated proposals, events, or projections until a named backend authority accepts them into durable product truth.

Sandboxing is a future enforcement upgrade, not a future contract rewrite. Path, env, process IO, resource envelope, identity, network/NATS authority, and audit boundaries must already be explicit enough to move execution behind Docker or another sandbox later.

Centralized operations are product infrastructure. Build, test, local service startup, generated contract checks, and release packaging must be available through one managed entry surface with stable names and verifiable results.

Verification confidence requires both directions. Outside-in tests prove that real app entrances produce correct NATS-mediated outcomes. Inside-out tests prove the smaller contracts that make failures diagnosable.

Inside-out tests are not a substitute for the NATS seam. They are the diagnostic map that explains a real-NATS outside-in pass or failure.

A release is not ready unless schema provenance, generated Go artifacts, generated TypeScript/Zod artifacts, fixtures, live browser-worker-substrate proof, centralized operation entries, and cross-lane contract results all agree from the same schema and app revision.

Every capability proof includes allowed, denied-neighbor, malformed, duplicate, stale-revision, revoked-credential, and attributed-failure cases. A happy-path NATS test proves connectivity only.

New activation or import kinds must fit the same taxonomy: authority, schema, attribution, durable position or dedupe behavior, denial behavior, restart behavior, and loop-safety behavior.

## Decision Hierarchy

Least authority outranks convenience.

NATS auth vocabulary outranks invented access concepts.

Mediated contracts outrank direct NATS access.

Policy provenance outranks opaque compiled permissions.

Schema authority outranks language-local types.

Durable materialized truth outranks local browser truth.

Backend-owned command effects outrank frontend optimism.

Attribution outranks silent success.

Centralized operational entrypoints outrank scattered local convenience.

Outside-in NATS proof outranks isolated confidence at release gates, while inside-out contract proof outranks broad end-to-end tests for fault localization.

## Reference Policy

This Approach references peer Approach documents for matched abstraction, platform structure, backend script mediation, and browser mediation. It may reference official NATS auth, account, import/export, permission, wildcard, WebSocket, JetStream, KV, Object Store, and service concepts as external constraints.

Plan documents may reference this Approach as authority and may reference peer Plan documents for coordination. Task documents may reference this Approach only through their owning Plan or for explicit inherited constraints.

Lower layers may use existing TypeScript/Bun work as regression evidence. They may not treat it as substrate authority.

## Plan-Readiness Gate

Plan work may proceed only when it preserves these gates:

- Auth is NATS-auth-shaped and lifecycle-managed by the substrate.
- Control-plane and app-plane authority are separated by account, authoritative prefix, or equivalent deny-enforced boundary.
- Identity, ownership, session, revision, and capability provenance survive compilation into NATS auth.
- Browser sub-apps remain receivers and typed intent emitters, not NATS clients.
- Browser sub-apps run as opaque sandboxed generated content unless a stronger separate-origin model is proven by Plan.
- Browser edge ownership is explicit for credentials, artifact serving, cache/CSP/sandbox policy, and missing browser control-plane behavior.
- Service-worker setup is server-owned, cookie-session-backed, scope-isolated, and token-free for generated content.
- Backend scripts remain NATS-agnostic by default and use the runtime facade for allowed effects.
- Script process contracts are sandbox-compatible from the first release boundary.
- Artifact and materializer authority is durable and observable through NATS-facing storage or subjects.
- Artifact, script, schema, snapshot, command, and chain revisions are compatible before effects are accepted.
- Activation covers more than request/reply and owns dedupe, chain attribution, and loop safety.
- Schema remains the neutral source for cross-language contracts and runtime validation.
- Schema validation and capability authorization stay separate and both are tested.
- Centralized operations are treated as a product boundary, not a convenience detail.
- Tests prove behavior outside-in through app entrances and inside-out through contract boundaries, including deny, duplicate, stale, malformed, and revoked cases.
- Release readiness includes live trusted-shell, dedicated-worker, browser-edge, substrate, script, and materializer proof over real NATS-mediated behavior.
- Any Plan that needs raw authority, scattered scripts, placeholder subjects, schema drift, local browser truth, silent policy provenance, or happy-path-only testing must return to Approach.

## Unresolved Branch Questions

**Which NATS auth backend shape should be authoritative?**

Recommended answer: make the domain model NATS-auth-shaped rather than provider-shaped. The substrate may render static config, account/user credentials, JWT-backed auth, or another NATS-supported mechanism, but the app contract speaks in NATS accounts, users or principals, permissions, imports, exports, and revocation semantics.

**Should the browser connect to NATS WebSocket directly or only through a server gateway?**

Recommended answer: the trusted dedicated worker may hold a scoped browser principal and connect to NATS WebSocket when useful, while the server gateway owns credential issuance, revocation, artifact serving, validation support, and any missing control-plane behavior. Generated sub-app content never receives credentials, subjects, or a publish API.

**Should frontend artifacts live in KV or Object Store?**

Recommended answer: use Object Store for bundle/blob artifacts and KV for manifests, revisions, indexes, projections, and small materialized records. NATS subjects announce changes and activation/materializer layers decide what those changes mean.

**How strict should script safety be before sandboxing exists?**

Recommended answer: strict at the contract boundary even if process isolation is not yet enforced. Identity, env, path, IO, resource envelope, facade permissions, attribution, denial, and cleanup must be explicit now so sandboxing can become an enforcement change later.

**Should centralized operations choose Just or Taskfile now?**

Recommended answer: Approach should require one centralized operational surface, stable names, repeatability, and verifiable results. The exact runner is a Plan concern unless it changes portability or release semantics.

**How much testing must go through real NATS?**

Recommended answer: release-level outside-in tests should use real NATS-mediated behavior. Inside-out tests may use fakes at lower seams only when schema, auth, activation, and facade contracts are independently proven and the fake preserves the same denial and attribution semantics.

**Can sub-apps command the backend or only observe?**

Recommended answer: sub-apps may emit typed command intents through the trusted shell and worker. Durable acceptance, idempotency, side effects, chain attribution, and completion remain backend-owned and observable as materialized state.
