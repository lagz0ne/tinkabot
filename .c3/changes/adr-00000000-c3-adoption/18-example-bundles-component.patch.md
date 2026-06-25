---
target: c3-503
scope: whole
type: component
parent: c3-5
title: Example Bundles
category: feature
---
## Goal

Demonstrate runnable chain-reaction apps.

## Parent Fit

| Field | Value |
| --- | --- |
| Parent | c3-5 Proof Docs and Workflow |
| Role | Release-shaped bundle examples and example docs |
| Owns | `examples/**` |
| Does not own | The bundle loader implementation itself |

## Purpose

Example Bundles provide concrete, runnable apps that illustrate Tinkabot's model. `clock` demonstrates boot, schedule, transform, and reactive projection serving; `builder` demonstrates a warmer transform where a source projection feeds a Bun/Vite build filter and emits artifacts.

## Governance

| Reference | Type | Governs | Precedence | Notes |
| --- | --- | --- | --- | --- |
| ref-bundle-as-app | ref | Example manifest and README shape | Runtime contract outranks example convenience | Examples must remain valid bundle apps. |
| ref-nats-native-chain-reaction | ref | Transform and projection flow | NATS materialized state is the demo truth | Pages consume derived material, not local hidden state. |

## Contract

| Surface | Direction | Contract | Boundary | Evidence |
| --- | --- | --- | --- | --- |
| Clock example | IN/OUT | `tick` emits state/page; `present` watches state and emits view; page consumes view. | Bundle runtime to browser artifact | `examples/clock/**` |
| Builder example | IN/OUT | Source projection feeds build filter that emits built artifacts/projection. | Bundle runtime to build tool | `examples/builder/**` |

## Derived Materials

| Material | Must derive from | Allowed variance | Evidence |
| --- | --- | --- | --- |
| README and package smoke | Goal, Parent Fit, Contract, and Derived Materials | Local path examples only | `examples/README.md`, `scripts/smoke-tinkalet-package.sh` |
