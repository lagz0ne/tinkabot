---
target: c3-101
scope: whole
type: component
parent: c3-1
title: Schema Registry
category: foundation
---
## Goal

Own the neutral wire contract.

## Parent Fit

| Field | Value |
| --- | --- |
| Parent | c3-1 Contracts and SDK |
| Role | Neutral schema and fixture source for runtime lanes |
| Owns | `schemas/base/v1/**` and schema README |
| Does not own | Runtime-specific authorization or transport implementation |

## Purpose

The Schema Registry defines the base v1 contract vocabulary for auth policies, browser commands, activation intents, script records, script effects, artifact manifests, material projections, events, and session records/frames. It is the contract source that Go and TypeScript must check against instead of inventing per-language truth.

## Governance

| Reference | Type | Governs | Precedence | Notes |
| --- | --- | --- | --- | --- |
| ref-nats-native-chain-reaction | ref | Activation and material contract vocabulary | Schema precedes SDK/runtime types | NATS source kinds and durable material appear in contract fixtures. |

## Contract

| Surface | Direction | Contract | Boundary | Evidence |
| --- | --- | --- | --- | --- |
| JSON Schema | OUT | `contract.schema.json` defines accepted and rejected protocol shapes. | Schema to SDK and Go validators | `schemas/base/v1/contract.schema.json` |
| Fixtures | OUT | Valid and invalid examples pin parity and denial cases. | Schema to tests | `schemas/base/v1/fixtures/**` |

## Derived Materials

| Material | Must derive from | Allowed variance | Evidence |
| --- | --- | --- | --- |
| SDK and Go contract tests | Goal, Parent Fit, Contract, and Derived Materials | Language-specific parser code only | `bun run test:contracts`, `cd substrate/go && go test ./contract` |
