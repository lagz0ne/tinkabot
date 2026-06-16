# Layer Prompts

Use these as compact starting points. Add only task-local context and raw artifacts.

## Approach Owner

Protect the Approach layer only. Return purpose, scope, non-goals, invariants, decision hierarchy, reference policy, Plan-readiness gate, and any unresolved branch question with a recommended answer. Reject sequencing, file work, commands, and task lists.

## Plan Owner

Protect the Plan layer only. Consume the named Approach docs. Return decomposition, dependency ordering, parallelization rules, handoff contracts, verification strategy, escalation log, and any unresolved branch question with a recommended answer. Reject new Approach decisions and executable task checklists.

## Task Owner

Protect one Task layer unit only. Consume the owning Plan section plus inherited Approach constraints. Return objective, exact scope, non-goals, acceptance contract, RED artifact, execution notes, verification evidence, and wrap-up announcement. Reject architecture changes, new decomposition, and vague completion claims.
