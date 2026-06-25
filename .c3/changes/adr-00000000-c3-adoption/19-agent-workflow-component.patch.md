---
target: c3-504
scope: whole
type: component
parent: c3-5
title: Agent Workflow
category: foundation
---
## Goal

Govern future agent work.

## Parent Fit

| Field | Value |
| --- | --- |
| Parent | c3-5 Proof Docs and Workflow |
| Role | Repo-local agent instructions, installed skills, Claude workflow, and Codex orchestration script |
| Owns | `AGENTS.md`, `.agents/**`, `.claude/**`, `.codex/**`, `scripts/codex-orchestrate.ts` |
| Does not own | Product runtime behavior except by documented workflow gates |

## Purpose

Agent Workflow stores the operating contract for future coding agents: read `tasks/todo.md`, use C3, matched-abstraction, reverse-tornado-okr when relevant, prefer proactive execution, use the TypeScript native preview checker, and verify before done. It also includes installed skill packages that are part of the repo's local work surface.

## Governance

| Reference | Type | Governs | Precedence | Notes |
| --- | --- | --- | --- | --- |
| ref-matched-abstraction-layers | ref | Architecture/planning/layer-doc behavior | Repo instructions outrank ad hoc chat habit | Agents must preserve layer authority. |
| ref-coverage-ratchet | ref | C3/OKR coverage workflow | Measured evidence over self-report | Reviews should check for uncovered code. |

## Contract

| Surface | Direction | Contract | Boundary | Evidence |
| --- | --- | --- | --- | --- |
| Agent instructions | OUT | Shared workspace flow, skill usage, TDD, and verification expectations. | Repo to coding agent | `AGENTS.md` |
| Agent tooling | IN/OUT | Codex orchestrator and local skills provide review/planning/execution aids. | Agent to repo workflow | `.agents/**`, `.claude/**`, `.codex/**`, `scripts/codex-orchestrate.ts` |

## Derived Materials

| Material | Must derive from | Allowed variance | Evidence |
| --- | --- | --- | --- |
| Future handoffs and plans | Goal, Parent Fit, Contract, and Derived Materials | Current-session specifics | `tasks/todo.md` |
