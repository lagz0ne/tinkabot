# Agent Run Prompt

Use the matched-abstraction instructions supplied with this prompt. If your environment supports skills, use the `matched-abstraction` skill. If it does not, treat the supplied skill files as the governing process.

Produce matched-abstraction documents for the selected topic. Stop at the Plan layer. The Plan must derive future Task work clearly enough that missing Plan content would be visible as missing delivery scope, but do not create Task documents, write code, or execute commands.

Keep the output compact enough for small/free models: prefer dense tables and short bullets, avoid repeated prose, and target roughly 2,500 output tokens. Render representative current-depth artifacts instead of exhaustive inventories.

Start by stating:

- Current layer.
- Target depth: Plan.
- Verification posture: RED-GREEN-TDD for future task work.
- Artifact set hypothesis for reviewing sufficiency before descending.

Return exactly these artifacts:

1. Approach document.
2. Plan document.
3. Brief sufficiency review explaining why the Plan is ready, or what branch question blocks it.

Requirements:

- Keep Approach at intent, scope, non-goals, invariants, decision hierarchy, reference policy, artifact set fitness, and Plan-readiness.
- Keep Plan at decomposition, dependency order, handoff contracts, verification strategy, artifact set fitness, and escalation rules.
- Include architecture, design, and testing artifacts when they help review the current abstraction and downstream derive/match work. Name useful artifacts even when you do not render them.
- For the artifact set, classify use-now, deferred, upstream, and dropped/merged candidates. For use-now artifacts, render the smallest useful version at the current depth or explain why naming alone is sufficient; state depth, derivation hints, matching hints, allowed elaboration, depth stop, and mismatch signals.
- If the topic has conflicting truth sources, unreliable external events, stateful sessions, money movement, entitlement rules, manual correction, permissions, or recovery paths, include compact use-now artifacts for the central reconciliation, exception/recovery, and external-boundary surfaces unless another rendered artifact clearly covers them.
- If the topic has differentiated roles, privileged/manual actions, a central object that changes state over time, correctness depending on precedence or quantity math, or requested downstream testing/integration/rollout work, include compact authority, lifecycle/truth-state, rule/quantity example, or rollout/acceptance artifacts unless another rendered artifact clearly covers the same reasoning job. If rollout work is explicitly requested downstream, include a thin rollout/acceptance slice at Plan depth.
- For rule/quantity surfaces, include at least one normal row and one conflict or exception row when later tests would need precedence, formula, or quantity-bucket meaning. Vocabulary alone is not enough.
- Because this run includes both Approach and Plan, deferring an artifact "to Plan" from Approach only counts if you render or merge it in the Plan. Otherwise classify it as deferred below Plan with a reason.
- Before final output, audit your artifact list: every use-now item must be rendered, explicitly merged into a rendered artifact, or downgraded. Do not mark plural sequence diagrams, API sketches, test matrices, or rollout plans as use-now unless representative rows are present.
- Keep use-now artifacts to the smallest non-overlapping set needed for review; merge or defer anything that would make the output exceed the compact budget.
- Ask at most one branch-resolving question only if the answer cannot be safely inferred. Include your recommended answer.
- Return upward if a Plan discovery invalidates the Approach boundary. Do not patch around it inside Plan.
