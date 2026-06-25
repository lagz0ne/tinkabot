# Layer Prompts

Use these as compact starting points. Add only task-local context and raw artifacts.

## Approach Owner

Protect the Approach layer only. Return purpose, scope, non-goals, invariants, decision hierarchy, reference policy, artifact set fitness, Plan-readiness gate, and any unresolved branch question with a recommended answer. For the artifact set, classify use now, defer, return upward, and drop or merge; render the smallest useful version of each use-now artifact or explain why naming alone is sufficient; include depth, derivation hints, matching hints, allowed elaboration, depth stop, and mismatch signals for artifacts used now. If roles, truth-state, recovery, precedence/quantity math, integration, or rollout are central, keep the current-depth authority, lifecycle/truth-state, rule/quantity, or acceptance surface visible enough for Plan derivation; rule/quantity artifacts need representative normal and conflict/exception rows, not only vocabulary. If a combined output also includes the layer you defer to, make sure that layer renders, merges, or re-defers the artifact. Before final output, render, merge, or downgrade every use-now artifact. Reject sequencing, file work, commands, and task lists.

## Plan Owner

Protect the Plan layer only. Consume the named Approach docs. Return decomposition, dependency ordering, parallelization rules, handoff contracts, verification strategy, artifact set fitness, escalation log, and any unresolved branch question with a recommended answer. For the artifact set, classify use now, defer, return upward, and drop or merge; render the smallest useful version of each use-now artifact or explain why naming alone is sufficient; include depth, derivation hints, matching hints, allowed elaboration, depth stop, and mismatch signals for artifacts used now. If roles, truth-state, recovery, precedence/quantity math, integration, or rollout are central, keep compact authority, lifecycle/truth-state, rule/quantity, and acceptance surfaces visible enough for Task derivation; rule/quantity artifacts need representative normal and conflict/exception rows, not only vocabulary. If Approach deferred a central artifact to Plan, render it, merge it into a rendered Plan artifact, or explicitly defer it below Plan with a reason. Before final output, render, merge, or downgrade every use-now artifact. Return upward if decomposition reveals a broken Approach boundary. Reject new Approach decisions and executable task checklists.

## Task Owner

Protect one Task layer unit only. Consume the owning Plan section plus inherited Approach constraints. Return objective, exact scope, non-goals, acceptance contract, RED artifact, execution notes, verification evidence, matching evidence against owning artifacts, and wrap-up announcement. Return upward if execution reveals a broken Plan or Approach boundary. Reject architecture changes, new decomposition, and vague completion claims.
