# Layer Contract

## Traversal Contract

Abstraction depth is job-dependent. Use Approach, Plan, and Task as default checkpoints, then add named sublayers only when they help the user review the work at the right level, such as architecture, design, testing, data, operations, or migration.

Each active layer must produce an artifact set hypothesis before descending or asking for approval. The hypothesis names the supplementary documents or evidence that would help the user judge sufficiency: diagrams, flows, matrices, examples, contracts, logs, prototypes, or command evidence. Create the artifact when it is cheap and useful; otherwise explain why the layer is still sufficient without it.

Late discoveries may force upward traversal. When lower-layer evidence breaks a higher-layer boundary, stop local expansion, record the discovery, return to the owning higher layer, revise its decision or scope, then update affected lower layers.

## Artifact Set Contract

Artifacts are reasoning instruments. Select them by the downstream elaboration burden and matching burden, not by discipline labels. The same artifact type can appear at multiple layers when its depth changes.

Every artifact set decision should classify candidates:

- Use now: needed to make the current layer compelling and reviewable.
- Defer: useful later, but too deep for the current layer.
- Return upward: evidence that an upstream decision is missing or wrong.
- Drop or merge: overlaps another artifact without distinct reasoning value.

For artifacts used now, render the smallest useful version at the current depth. A compact sketch, table, matrix, or example is enough when it makes decisions reviewable. If an artifact is only named but not rendered, do not classify it as use-now unless the layer explicitly explains why naming alone is sufficient.

Before finalizing the artifact set, check for critical surfaces. If the topic has multiple sources of truth, unreliable external events, stateful sessions, money movement, entitlement rules, manual correction, permissions, or operational recovery, the owning layer should render compact artifacts for the central surfaces instead of deferring them:

- Reconciliation matrix: disagreement case, authority, correction path, audit/matching evidence.
- Exception/recovery flow table: scenario, trigger, owner, allowed action, downstream test.
- External boundary table: actor/system, trust level, event or contract, failure signal, fallback.

Also check central operability surfaces:

- Authority matrix: role or actor, allowed action, forbidden or escalated action, audit evidence, downstream test.
- Lifecycle/truth-state matrix: state or quantity, meaning, owning authority, transition/evidence, mismatch signal.
- Rule/quantity example matrix: competing rule, entitlement, reservation, price, or quantity case; expected precedence or calculation; evidence; downstream test.
- Rollout/acceptance slice: rollout or integration surface, readiness evidence, rollback/escalation trigger, downstream Task family.

Use these when humans have different authority, manual correction changes truth, a central concept changes state over time, correctness depends on precedence or quantity math, or the prompt asks for later testing, integration, or rollout work. If the prompt explicitly asks for downstream rollout work, a thin rollout/acceptance slice belongs at Plan depth. If a critical surface is omitted, explain which rendered artifact covers the same reasoning job.

For rule/quantity surfaces, representative examples are part of the artifact. Do not satisfy this with vocabulary alone. Include at least one normal row and one conflict or exception row when later tests would need precedence, formula, or quantity-bucket meaning.

When one output spans multiple layers, "defer to Plan" from Approach is not enough if the Plan is included in the same output. Either render the artifact in the target layer, merge it into a rendered target-layer artifact, or classify it as deferred below the target layer with a reason.

Before final output, audit the artifact list: every item marked use-now must be rendered, explicitly merged into a rendered artifact, or downgraded. Do not mark plural sequence diagrams, API sketches, test matrices, or rollout plans as use-now unless representative rows are present in the output.

For artifacts used now, capture:

- Depth: orienting, boundary, relational, behavioral, or operational.
- Derivation hints: downstream material this artifact should make easier to elaborate.
- Matching hints: invariants, vocabulary, ownership, states, risks, or acceptance conditions later work must align with.
- Allowed elaboration: freedom the next layer retains.
- Depth stop: details intentionally not decided at this layer.
- Mismatch signals: discoveries that force upward traversal.

Use the smallest artifact set that lets the next layer elaborate without invention and lets reviewers detect drift without guessing.

## Approach

Approach owns the top-level intent, abstraction boundaries, decision hierarchy, invariants, vocabulary, non-goals, and success conditions. It decides what kind of thinking is valid for the system.

Required artifacts:

- Approach charter.
- Layer contract.
- Decision hierarchy.
- Reference policy.
- Artifact set hypothesis for user confirmation.
- Artifact set fitness: use now, defer, return upward, drop or merge.
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
- Artifact set map for the planned work.
- Derive/match and depth notes for the artifact set.
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
- Review evidence that confirms the bounded unit is complete.
- Matching evidence against owning Plan and Approach artifacts.
- Wrap-up announcement.

Allowed references: owning Plan doc, Approach constraints, peer Task docs for dependency/interface facts, source code, tests, logs, and generated evidence.

Forbidden references: Task cannot introduce Approach or Plan decisions, rewrite higher-layer rationale, depend on undefined peer behavior, or expand scope because implementation revealed a better abstraction.
