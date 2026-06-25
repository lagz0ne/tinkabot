---
target: c3-401
scope: whole
type: component
parent: c3-4
title: Trusted Frontend Shell
category: foundation
---
## Goal

Isolate generated browser content.

## Parent Fit

| Field | Value |
| --- | --- |
| Parent | c3-4 Browser Shell |
| Role | Browser isolation helpers, trusted shell entry, CSS, fixture, and Vite build |
| Owns | `apps/frontend/**` except generated embedded Go copy |
| Does not own | Go shell HTTP routes or session credential minting |

## Purpose

Trusted Frontend Shell provides the browser-side isolation and mediation rules. It creates sandboxed frames, checks lease identity, rejects raw authority vocabulary in generated-content messages, and turns valid content intents into browser command intents.

## Governance

| Reference | Type | Governs | Precedence | Notes |
| --- | --- | --- | --- | --- |
| ref-shadow-authority-boundaries | ref | Browser frame leases and generated-content denials | Shell mediation outranks generated UI convenience | Generated UI may render material but cannot receive raw NATS handles. |
| rule-generated-code-no-raw-authority | rule | Browser message validation | Deny raw authority fields | `denyRaw` backs this behavior. |

## Contract

| Surface | Direction | Contract | Boundary | Evidence |
| --- | --- | --- | --- | --- |
| Frame isolation | IN/OUT | Only `allow-scripts` sandbox, leased source, nonce, revisions, and command allow-list pass. | Generated frame to trusted shell | `apps/frontend/src/isolation.ts` |
| Frontend build | OUT | Vite shell output can be embedded into Go runtime. | Frontend to Go static FS | `apps/frontend/vite.config.ts`, `substrate/go/frontend/site/**` |

## Derived Materials

| Material | Must derive from | Allowed variance | Evidence |
| --- | --- | --- | --- |
| Go embedded frontend assets | Goal, Parent Fit, Contract, and Derived Materials | Hash filenames and minification | `bun run build:frontend`, `substrate/go/frontend/site/**` |
