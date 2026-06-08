---
layer: approach
topic: platform-structure
references:
  - ./charter.md
  - ./nats-script-runtime.md
  - ./browser-frontend-mediator.md
---

# Platform Structure Approach

## Scope

Tinkabot resets from a package-shaped TypeScript runtime toward a mediated NATS platform. Go owns the substrate. Vite owns frontend shell delivery. A shared SDK/schema layer owns contract shape. TypeScript runtime-facing boundaries use Zod validation generated from, or paired with, neutral schema contracts.

The reset preserves the existing mediation invariants: backend scripts do not receive raw NATS by default, generated frontend content does not receive raw NATS by default, and both sides communicate through typed mediated boundaries.

Diagram: https://diashort.apps.quickable.co/d/d6335d7c

## Layer Contract

Approach owns platform authority boundaries, canonical contract ownership, validation policy, and supersession of earlier Bun substrate authority.

Plan owns lane decomposition, dependency ordering, handoff contracts, and verification strategy.

Task owns one executable unit, such as persisting the reset, scaffolding schema contracts, or proving one generated artifact path.

## Core Thesis

Go is the substrate because NATS authority, auth, Docker/sandboxing, process lifecycle, connection policy, and execution attribution belong in a long-running infrastructure layer. Vite is the frontend because the product needs a real browser shell. The SDK/schema layer is the shared contract authority because Go and TypeScript must not drift.

## Decision Hierarchy

Mediation and least authority outrank convenience.

Shared schema truth outranks local Go structs or TypeScript types.

Runtime validation outranks type-only TypeScript contracts at trust boundaries.

Go owns substrate authority: NATS infrastructure, auth, Docker/sandboxing direction, process lifecycle, connection policy, and execution attribution.

Vite owns frontend delivery. Generated content stays untrusted relative to the shell and dedicated worker.

JSON Schema is the initial neutral contract source. Zod validators, TypeScript SDK types, Go structs/validators, contract fixtures, and test data are generated or checked against that source.

Existing Bun and `@lagz0ne/nats-embedded` work is evidence only. It may preserve behavior, fixtures, and regression expectations, but it no longer defines platform authority.

## Reference Policy

This Approach may reference peer Approach docs for script mediation and browser mediation invariants. Lower layers may use existing TypeScript/Bun tests as evidence, but may not treat current folder shape or Bun runtime mechanics as platform authority.

## Non-Goals

- No raw NATS access by default for scripts or generated frontend content.
- No TypeScript-only schema truth.
- No Go-only schema truth.
- No service-worker ownership of browser NATS authority in v1.
- No Docker sandbox implementation in this decision slice.
- No implementation scaffold until the reset docs and stale-authority cleanup are verified.

## Plan-Readiness Gate

Plan can proceed when it preserves these invariants:

- Go substrate owns infrastructure authority.
- Vite owns frontend shell delivery.
- SDK/schema owns shared shape.
- JSON Schema is the first neutral schema source.
- Zod validates TypeScript runtime-facing boundaries.
- Code generation and parity tests enforce long-run alignment.
- Raw NATS remains unavailable to scripts and generated frontend content by default.
- Prior Bun substrate work is evidence only.
