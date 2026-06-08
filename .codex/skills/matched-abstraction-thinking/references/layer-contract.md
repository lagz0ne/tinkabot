# Layer Contract

## Approach

Approach owns the top-level intent, abstraction boundaries, decision hierarchy, invariants, vocabulary, non-goals, and success conditions. It decides what kind of thinking is valid for the system.

Required artifacts:

- Approach charter.
- Layer contract.
- Decision hierarchy.
- Reference policy.
- Plan-readiness gate.

Allowed references: peer Approach docs, external principles, and project-wide constraints.

Forbidden references: Plan or Task docs as authority, execution order, concrete tasks, file-level work, command recipes, and tactical decomposition.

## Plan

Plan owns decomposition. It consumes approved Approach decisions and turns them into execution boundaries: sequencing, dependency shape, agent roles, handoff contracts, verification strategy, and escalation gates.

Required artifacts:

- Plan brief with consumed Approach docs and carried decisions.
- Decomposition map.
- Handoff contract for Task-bound units.
- Verification strategy.
- Escalation log.

Allowed references: Approach docs as authority, peer Plan docs for coordination, code/docs as feasibility evidence, and Task artifacts only as execution feedback after Task work exists.

Forbidden references: Plan cannot redefine Approach, treat Task docs as authority, embed file-by-file edits, or require Task internals to understand Plan.

## Task

Task owns one executable unit. It converts an approved Plan slice into scoped work, acceptance criteria, RED proof, implementation notes, verification evidence, and a wrap-up announcement.

Required artifacts:

- Task brief.
- Acceptance contract.
- RED artifact.
- Execution notes.
- Verification evidence.
- Wrap-up announcement.

Allowed references: owning Plan doc, Approach constraints, peer Task docs for dependency/interface facts, source code, tests, logs, and generated evidence.

Forbidden references: Task cannot introduce Approach or Plan decisions, rewrite higher-layer rationale, depend on undefined peer behavior, or expand scope because implementation revealed a better abstraction.
