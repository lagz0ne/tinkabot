---
target: c3-502
scope: whole
type: component
parent: c3-5
title: Documentation and Evidence
category: foundation
---
## Goal

Keep layer docs reviewable.

## Parent Fit

| Field | Value |
| --- | --- |
| Parent | c3-5 Proof Docs and Workflow |
| Role | README/manual, matched-abstraction Approach/Plan/Task docs, task handoff, and lessons |
| Owns | `README.md`, `docs/**`, `tasks/**` |
| Does not own | Runtime implementation unless a Task doc explicitly scopes it |

## Purpose

Documentation and Evidence owns the durable explanation surfaces. Approach documents hold product/architecture authority, Plan documents decompose coordination, Task documents record acceptance, RED artifacts, verification, and release proof, while `tasks/todo.md` remains the current handoff state.

## Governance

| Reference | Type | Governs | Precedence | Notes |
| --- | --- | --- | --- | --- |
| ref-matched-abstraction-layers | ref | Approach/Plan/Task authority direction | Higher layers constrain lower layers | Lower docs may cite but not redefine higher intent. |
| ref-coverage-ratchet | ref | Code-to-doc coverage objective | Measured coverage outranks apparent doc volume | Line coverage must be reviewed as a direct metric. |

## Contract

| Surface | Direction | Contract | Boundary | Evidence |
| --- | --- | --- | --- | --- |
| Layer docs | OUT | Approach owns intent, Plan owns decomposition, Task owns executable proof. | Docs to implementation | `docs/matched-abstraction/**` |
| Handoff state | IN/OUT | `tasks/todo.md` records current goal, blockers, debts, and next steps. | Agent session to future work | `tasks/todo.md` |

## Derived Materials

| Material | Must derive from | Allowed variance | Evidence |
| --- | --- | --- | --- |
| User-facing manual and README | Goal, Parent Fit, Contract, and Derived Materials | Install paths and release versions | `docs/manual/v1.md`, `README.md` |
