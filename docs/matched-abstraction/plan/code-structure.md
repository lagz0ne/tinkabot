---
layer: plan
topic: code-structure
references:
  - ../approach/code-structure.md
  - ../approach/platform-structure.md
  - ./platform-structure.md
---

# Code Structure Plan

Structure diagram: https://diashort.apps.quickable.co/d/b98307da

## Consumed Approach

This plan consumes `docs/matched-abstraction/approach/code-structure.md` and `docs/matched-abstraction/approach/platform-structure.md` as authority. Root becomes orchestration only. Go owns substrate authority. Vite owns trusted browser shell authority. SDK/schema owns shared shape through neutral schema, runtime validation, generated artifacts, and parity fixtures.

## Decomposition

The root orchestration lane owns workspace metadata, cross-lane scripts, docs, task handoff, and layer validation.

The SDK package lane owns the current TypeScript SDK surface, browser mediator contracts, metadata validation helpers, permission helpers, activation intent shape, subject matching behavior, distribution build, and legacy Bun proof material until replacement Go substrate evidence exists.

The schema lane owns canonical JSON Schema sources and the generator contract for Zod validators, Go validators/types, TypeScript SDK types, and fixtures.

The Go substrate lane owns NATS infrastructure, auth, process lifecycle, Docker-facing execution boundaries, activation ledgers, artifact gateways, and execution attribution.

The frontend lane owns the Vite trusted shell, dedicated-worker mediation, materializer runtime, browser validation, and frontend command/state loop.

## Sequencing

First, move the existing TypeScript package out of root into an SDK-owned package without changing behavior.

Second, keep empty future lanes explicit through concise ownership notes so later code lands in the correct owner.

Third, move schema authority into the schema lane with generator and parity tests before Go or Vite add their own schema-shaped code.

Fourth, re-own substrate behavior in Go and retire Bun substrate proof material only after equivalent Go tests pass.

## Verification Strategy

Structure verification proves root no longer has product `src` or `dist` directories and the existing TypeScript code sits under SDK package ownership.

Behavior verification proves the root test, typecheck, build, e2e, and package dry-run commands still work after delegation into the SDK package.

Layer verification proves the matched-abstraction docs still validate.

Escalate back to Approach if the move requires public package renaming, Go implementation, Vite implementation, Docker enforcement, or schema generator semantics.
