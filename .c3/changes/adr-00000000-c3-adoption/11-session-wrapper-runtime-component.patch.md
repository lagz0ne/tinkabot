---
target: c3-204
scope: whole
type: component
parent: c3-2
title: Session and Wrapper Runtime
category: feature
---
## Goal

Mediate session streams and wrapper IO.

## Parent Fit

| Field | Value |
| --- | --- |
| Parent | c3-2 Go Product Runtime |
| Role | Session frame mediation, trusted wrapper authority, WebSocket observation, and demo session proof |
| Owns | `substrate/go/apps/wrapper/**`, session-related `embednats` and `tinkabot` files |
| Does not own | Browser UI rendering details or Tinkalet product commands |

## Purpose

Session and Wrapper Runtime adapts local process IO into schema-shaped session frames, mediates steering, mints browser observation authority, and proves that trusted wrappers and demo sessions remain inside the same authority model as the rest of the runtime.

## Governance

| Reference | Type | Governs | Precedence | Notes |
| --- | --- | --- | --- | --- |
| ref-shadow-authority-boundaries | ref | Wrapper and browser session authority | Mediated frame authority outranks raw process/browser access | Wrapper output is framed and validated before product use. |

## Contract

| Surface | Direction | Contract | Boundary | Evidence |
| --- | --- | --- | --- | --- |
| Wrapper stream | IN/OUT | Stream JSON frames map to session frames and steering maps to stdin lines. | Local process to NATS session material | `substrate/go/apps/wrapper/**` |
| Web session | OUT | Browser observer gets scoped bearer/ticket authority to its deliver subject only. | Server to trusted shell | `substrate/go/tinkabot/web_session.go`, `substrate/go/embednats/web_session_surface_test.go` |

## Derived Materials

| Material | Must derive from | Allowed variance | Evidence |
| --- | --- | --- | --- |
| Session v2 task docs and tests | Goal, Parent Fit, Contract, and Derived Materials | Test-only fixture details | `docs/matched-abstraction/task/session-*.md`, `go test ./apps/wrapper ./embednats ./tinkabot` |
