---
layer: plan
topic: matched-abstraction-thinking
references:
  - ../approach/charter.md
---

# Matched Abstraction Thinking Plan

## Consumed Approach

This plan consumes `docs/matched-abstraction/approach/charter.md` as authority. The key carried decisions are top-down decision flow, strict layer ownership, layer-matched document storage, and subagent protection for each layer.

## Decomposition

The orchestrator coordinates four work areas:

- Approach agent protects the design-space contract.
- Plan agent protects decomposition and handoff contracts.
- Task agent protects one executable unit and verification evidence.
- Main orchestrator synthesizes outputs, updates docs, announces ready-to-change state, and verifies the full setup.

Layer agents can work in parallel when their scopes are independent. Work serializes when a lower layer depends on a not-yet-approved higher-layer decision.

## Handoff Contract

Every layer handoff includes purpose, owned artifacts, allowed references, rejected concerns, readiness gate, and one branch-resolving question with a recommended answer when uncertainty remains.

Task handoffs also include acceptance criteria, RED artifact, verification method, touched surfaces, and wrap-up requirements.

## Verification Strategy

The baseline setup is verified by three checks: skill metadata validation, validator unit tests, and layer document validation. Task-level work must record exact commands and results before completion is claimed.

## Escalation Log

No unresolved Approach decision is blocking this baseline. If future Plan work needs to choose between incompatible abstraction models, it must escalate to Approach before creating task units.
