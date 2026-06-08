---
layer: approach
topic: matched-abstraction-thinking
references: []
---

# Matched Abstraction Thinking Approach

## Scope

This project uses matched abstraction thinking as the baseline system-design discipline. Work progresses top-down through Approach, Plan, and Task layers. Each layer owns a distinct level of thought and stores its documents in the matching directory.

Diagram: https://diashort.apps.quickable.co/d/9071cb70

## Core Thesis

The level of abstraction must match the decision being made. Approach defines the design space, Plan decomposes the design space into coordinated work, and Task executes one bounded unit with proof.

## Layer Contract

Approach owns intent, scope, non-goals, invariants, vocabulary, decision hierarchy, and success conditions.

Plan owns decomposition, dependency ordering, parallelization, handoff contracts, verification strategy, and escalation gates.

Task owns one executable unit, acceptance criteria, RED proof, implementation notes, verification evidence, and the wrap-up announcement.

Each active layer is protected by a dedicated subagent. One subagent may not own multiple layers in the same decision flow. The main agent orchestrates, synthesizes, announces ready-to-change state, and verifies.

## Decision Hierarchy

Decisions flow downward. Plan carries Approach decisions and may not reopen them. Task carries Plan and Approach decisions and may not redefine either. A lower layer that finds a missing or contradictory higher-layer decision escalates upward instead of patching around it.

## Reference Policy

Approach documents may reference peer Approach documents and external project constraints. Plan documents may reference Approach and peer Plan documents. Task documents may reference Approach, Plan, and peer Task documents only for dependency or interface facts.

## Plan-Readiness Gate

Plan work may begin when the Approach layer states purpose, non-goals, layer ownership, decision authority, reference direction, and the branch questions that remain open. If a Plan would have to rediscover a catastrophic design decision, Approach is not ready.
