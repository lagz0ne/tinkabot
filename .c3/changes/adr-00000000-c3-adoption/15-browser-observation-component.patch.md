---
target: c3-402
scope: whole
type: component
parent: c3-4
title: Browser Observation
category: feature
---
## Goal

Observe sessions through scoped browser grants.

## Parent Fit

| Field | Value |
| --- | --- |
| Parent | c3-4 Browser Shell |
| Role | Trusted browser observation over cookie-gated viewer mint and WebSocket NATS connection |
| Owns | `apps/frontend/src/observe.ts` and observation tests |
| Does not own | Server-side session runtime or wrapper process IO |

## Purpose

Browser Observation lets the trusted shell stream session token frames by minting a scoped viewer grant and one-use WebSocket ticket from the server. It keeps generated content out of the NATS connection path.

## Governance

| Reference | Type | Governs | Precedence | Notes |
| --- | --- | --- | --- | --- |
| ref-shadow-authority-boundaries | ref | Browser observer grants and generated-content exclusion | Server mint and shell ownership outrank direct generated access | The shell holds the NATS connection. |

## Contract

| Surface | Direction | Contract | Boundary | Evidence |
| --- | --- | --- | --- | --- |
| Viewer mint | IN/OUT | Cookie-gated POST returns bearer JWT, deliver subject, and one-use WebSocket ticket. | Browser shell to server | `apps/frontend/src/observe.ts` |
| Session frame rendering | IN | Only token session frames render text; malformed or chunk frames render nothing. | NATS frame to UI | `apps/frontend/tests/observe.test.ts` |

## Derived Materials

| Material | Must derive from | Allowed variance | Evidence |
| --- | --- | --- | --- |
| Browser observation tests | Goal, Parent Fit, Contract, and Derived Materials | Test doubles for fetch/NATS only | `bun run --cwd apps/frontend test` |
