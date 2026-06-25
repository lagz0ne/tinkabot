---
layer: plan
topic: platform-structure
references:
  - ../approach/platform-structure.md
  - ../approach/nats-script-runtime.md
  - ../approach/browser-frontend-mediator.md
---

# Platform Structure Plan

## Consumed Approach

This plan consumes `docs/matched-abstraction/approach/platform-structure.md` as current platform authority. Go substrate owns infrastructure authority. Vite owns frontend shell delivery. SDK/schema owns shared shape. JSON Schema is the first neutral schema source, with generated or checked Zod, TypeScript SDK, Go validation/types, fixtures, and parity tests.

Existing NATS script runtime and browser mediator docs remain authority for mediation invariants, but not for substrate ownership.

## Decomposition

Platform reset has five lanes:

- Canonical schema lane: JSON Schema source, generated Zod validators, TypeScript SDK types, Go types/validators, and contract fixtures.
- Go substrate lane: NATS services, auth/capability policy, Docker/sandboxing direction, process lifecycle, connection policy, command/activation ledger, artifact gateway, and execution attribution.
- Vite frontend lane: trusted shell, dedicated-worker mediator, generated-content receiver boundary, materializer display, and browser-side validation.
- Mediation integration lane: backend scripts and frontend content both prove no raw NATS by default while using typed intent/state contracts.
- Enforcement lane: parity tests, generated artifact manifest, stale-authority scans, and outside-in BDD proof.

## Dependency Ordering

The schema lane comes first because every other lane consumes shared shape.

Go substrate and Vite frontend can proceed in parallel after the initial schema contracts exist.

Mediation integration follows the substrate and shell contracts.

Outside-in BDD comes after one substrate path, one frontend path, and one generated schema path are all working.

## Parallelization Rules

Do not parallelize work that can redefine canonical schema shape.

Parallelize Go substrate and Vite shell only after their shared contracts are generated or fixture-backed.

Keep generated frontend content as receiver/intent emitter in every frontend task.

Keep backend scripts mediated in every substrate task.

Tests may be designed in parallel, but expected behavior must trace back to Approach invariants.

## Handoff Contract

Task units receive:

- Owning lane and inherited platform Approach.
- Canonical schema version and generated artifact expectations when schema is involved.
- Non-goals: no raw NATS default, no TypeScript-only contract truth, no Go-only contract truth, no unverified sandbox claims.
- Required evidence: RED artifact, generated or validated contract output, denial behavior when crossing trust boundaries, and layer validation.

## Verification Strategy

Verification is contract-first:

- Schema parity: JSON Schema source generates or validates Zod, TypeScript SDK, Go types/validators, and fixtures.
- Mediation: scripts and generated frontend content cannot access raw NATS by default.
- Runtime validation: TypeScript boundaries parse with Zod before trusted use.
- Substrate authority: Go owns NATS/auth/process/Docker-facing concerns.
- Frontend shell: Vite shell and dedicated worker preserve receiver/intent content boundary.
- Outside-in: one full app loop proves frontend intent, backend acceptance, script execution, material update, and frontend rendering.

## Escalation Log

Escalate to Approach if a task needs raw NATS by default, TypeScript-first schema truth, Go-first schema truth, service-worker NATS ownership, Bun substrate authority, or Docker sandbox enforcement claims before a sandbox proof exists.
