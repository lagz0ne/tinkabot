---
target: c3-102
scope: whole
type: component
parent: c3-1
title: TypeScript SDK
category: foundation
---
## Goal

Expose checked TypeScript contract helpers.

## Parent Fit

| Field | Value |
| --- | --- |
| Parent | c3-1 Contracts and SDK |
| Role | TypeScript package surface derived from schema and browser/runtime contracts |
| Owns | `packages/sdk/**` |
| Does not own | Go embedded runtime or release packaging |

## Purpose

The TypeScript SDK contains base-contract helpers, browser-frontend helpers, and nats-script-runtime helpers that preserve schema and runtime vocabulary for TypeScript consumers. It also carries tests that keep SDK behavior aligned with schema fixtures and documented contract expectations.

## Governance

| Reference | Type | Governs | Precedence | Notes |
| --- | --- | --- | --- | --- |
| ref-matched-abstraction-layers | ref | Which SDK docs are authority versus derived proof | Schema and Approach outrank package convenience | SDK exports should not become a competing source of product decisions. |

## Contract

| Surface | Direction | Contract | Boundary | Evidence |
| --- | --- | --- | --- | --- |
| Package exports | OUT | `src/index.ts` re-exports contract, browser, and script runtime modules. | SDK to consumers | `packages/sdk/src/index.ts` |
| Runtime substrate helper | IN/OUT | Starts embedded NATS through injected factories and maps runtime errors. | SDK to NATS embedding | `packages/sdk/src/nats-script-runtime/runtime-substrate.ts` |

## Derived Materials

| Material | Must derive from | Allowed variance | Evidence |
| --- | --- | --- | --- |
| Built package output | Goal, Parent Fit, Contract, and Derived Materials | Generated JS/declaration formatting | `packages/sdk/dist/**`, `bun run --cwd packages/sdk build` |
