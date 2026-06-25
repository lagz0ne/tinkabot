---
name: matched-abstraction
description: Use when design, planning, implementation, or delegation risks mixing abstraction layers or skipping from high-level intent to detail too quickly; triggers include architecture decisions, design artifacts, testing strategy, implementation plans, task handoffs, layered docs, orchestration, decomposition, verification strategy, artifact-set refinement, derive/match review surfaces, and strategy-to-execution drift.
---

# Matched Abstraction

Use matched abstraction to move deliberately from high-level intent to detail. Approach constrains thinking, Plan decomposes coordination, and Task executes one bounded unit with evidence. These layers are checkpoints for review, not proof that every job needs exactly three documents. Choose the depth required by the job, spend enough time at each active layer to make its decisions reviewable, and descend only after the current layer is sufficient. Lower layers may cite higher or peer artifacts, but must not redefine them.

## First Move

Announce RED-GREEN-TDD as the verification posture for non-trivial work. Identify the current layer and the likely target depth before producing output. Explore locally answerable facts before asking the user.

Build an artifact set hypothesis before asking for confirmation or moving downward. Name the documents, diagrams, matrices, examples, logs, prototypes, or other evidence that would make the current layer compelling enough to derive downstream work and match later work back to the owning intent.

If a branch decision blocks the current layer, ask one branch-resolving question at a time. Include the recommended answer and the evidence behind it. If the current layer already allows a safe default, state the default and continue.

## Workflow

1. Approach: define purpose, scope, non-goals, invariants, decision hierarchy, reference policy, artifact set fitness, and Plan-readiness.
2. Plan: consume Approach authority and produce decomposition, dependency ordering, handoff contracts, verification strategy, escalation rules, and artifact set fitness.
3. Task: execute one bounded unit with scope, acceptance criteria, RED artifact, execution notes, verification evidence, and wrap-up.
4. Orchestrate: synthesize layer outputs, preserve authority direction, then verify.
5. Revisit: when a late discovery breaks a higher-layer boundary, stop expanding the lower layer, record the discovery, return to the owning layer, revise it, then regenerate affected lower-layer work.

One layer owner handles one layer. Do not proceed downward while the current layer has unresolved branch decisions or missing review artifacts that could materially change the next layer.

## Traversal Discipline

- Treat abstraction as a range from high intent to detailed evidence. Use Approach, Plan, and Task as the default checkpoints, and add named sublayers only when the job needs them, such as architecture, product design, test design, or migration strategy.
- Spend time inside the current abstraction before descending. Inspect purpose, constraints, interfaces, risks, unknowns, success conditions, and evidence needs at the layer's own level.
- Match supplementary artifact sets to the nature and depth of the layer. Architecture may need system, dependency, sequence, or decision diagrams. Design may need flows, wireframes, state models, or content inventories. Testing may need coverage maps, test matrices, fixtures, RED artifacts, and command evidence.
- Use artifacts as reasoning instruments, not decoration. If an artifact is not created inline or persisted, name it, explain its downstream derive/match value, and state whether the layer is still sufficient without it.
- Ask the user to resolve only the decision that cannot be safely inferred after local exploration and artifact set hypothesis. Include the recommended answer.
- Define sufficiency as "bounded enough for the next layer", not "exhaustively complete."

## Artifact Set Fitness

Choose the smallest artifact set that makes the layer's content compelling, non-overlapping, and useful for downstream derive-and-match. Do not assign artifact types to fixed layers. Information architecture, ERDs, sequence diagrams, state models, wireframes, test matrices, prototypes, logs, and examples may appear at multiple layers, but their depth and authority must change with the layer.

For each active layer, decide:

- Use now: artifacts needed here because they make current-layer decisions reviewable and downstream work easier to elaborate.
- Defer: artifacts that may be useful later, but would answer a lower-layer question too early.
- Return upward: artifacts whose sudden need reveals that an upstream intent, boundary, invariant, vocabulary, or ownership decision was missed.
- Drop or merge: artifacts that overlap another artifact without adding a distinct reasoning job.

For artifacts used now, render the smallest useful version at the current depth. A use-now state model may be a five-row table, a use-now sequence may be a short happy/error path, and a use-now matrix may contain representative rows instead of every case. If an artifact is only named but not rendered, do not classify it as use-now unless you explain why naming alone is sufficient for descent.

Before finalizing the artifact set, check for critical surfaces. If the topic has multiple sources of truth, unreliable external events, stateful sessions, money movement, entitlement rules, manual correction, permissions, or operational recovery, render a compact current-depth artifact for each central surface instead of deferring it. Prefer:

- Reconciliation matrix: source A vs source B, expected authority, correction path, audit/matching evidence.
- Exception/recovery flow table: scenario, triggering signal, owner, allowed action, downstream test.
- External boundary table: external actor/system, trust level, consumed/produced events, failure signal, fallback.

Also check for central operability surfaces that often look "too detailed" but are needed for downstream derivation:

- Authority matrix: actor or role, allowed decision/action, forbidden or escalated action, audit evidence, downstream test.
- Lifecycle/truth-state matrix: state or quantity, meaning, owning authority, transition/evidence, mismatch signal.
- Rule/quantity example matrix: competing rule, entitlement, reservation, price, or quantity case; expected precedence or calculation; evidence; downstream test.
- Rollout/acceptance slice: rollout or integration surface, readiness evidence, rollback/escalation trigger, downstream Task family.

Use these when humans have different authority, manual correction changes truth, a central concept changes state over time, correctness depends on precedence or quantity math, or the prompt asks for later testing, integration, or rollout work. If the prompt explicitly asks for downstream rollout work, a thin rollout/acceptance slice belongs at Plan depth. If one of these surfaces is central but omitted, state why another rendered artifact already covers the same reasoning job.

For rule/quantity surfaces, representative examples are part of the artifact. Do not satisfy this with vocabulary alone. Include at least one normal row and one conflict or exception row when later tests would need precedence, formula, or quantity-bucket meaning.

When one output spans multiple layers, "defer to Plan" from Approach is not enough if the Plan is included in the same output. Either render the artifact in the target layer, merge it into a rendered target-layer artifact, or classify it as deferred below the target layer with a reason.

Before final output, audit the artifact list: every item marked use-now must be rendered, explicitly merged into a rendered artifact, or downgraded. Do not mark plural sequence diagrams, API sketches, test matrices, or rollout plans as use-now unless representative rows are present in the output.

For artifacts used now, state:

- Artifact depth: orienting, boundary, relational, behavioral, or operational.
- Derivation hints: what downstream material the artifact should help produce.
- Matching hints: what later material must align with to prevent drift.
- Allowed elaboration: what the next layer may decide freely.
- Depth stop: what this artifact intentionally does not decide yet.
- Mismatch signals: what discovery forces return to the owning layer.

Choose the shallowest depth that lets the next layer elaborate without invention and lets reviewers detect mismatch without guessing. Healthy ping-pong across layers is allowed: "use later, not at this depth" and "this should have been settled upstream" are valid outcomes.

## Layer Rules

| Layer | Owns | Must reject |
| --- | --- | --- |
| Approach | Intent, principles, invariants, non-goals, decision authority | Sequencing, file work, task lists, commands |
| Plan | Decomposition, dependencies, handoff contracts, verification strategy | New Approach decisions, file-level recipes, task checklists |
| Task | One executable unit, RED proof, implementation evidence, final announcement | Architecture changes, new decomposition, vague verification |

## Output Modes

Inline mode: when file persistence is unavailable, unnecessary, or explicitly forbidden, return document-shaped Approach, Plan, and Task outputs inline. Keep the same layer boundaries and state which persistence was skipped because of the request constraint.

Persistence mode: when the caller or repository provides a layer root, store artifacts under that root using `approach/`, `plan/`, and `task/` directories. Each persisted document needs `layer`, `topic`, and `references` frontmatter. Keep supplementary artifacts near the layer that owns them. Reference direction is top-down: Approach -> Approach; Plan -> Approach/Plan; Task -> Approach/Plan/peer Task.

Persisted docs should make review artifacts and review evidence explicit near the owning layer. Use clear headings when helpful, but do not treat heading presence as validation.

## Sufficiency Review

Matched abstraction is not mechanically provable from headings or keywords. Before claiming readiness, review the artifacts and evidence against the layer contract:

- Approach has scope, layer contract, and Plan-readiness.
- Plan names consumed Approach authority, decomposition, and verification strategy.
- Task has acceptance contract, RED artifact, and concrete command/result evidence.
- Each active layer states artifact set fitness or evidence for why descending is sufficient.
- References point only to allowed layers.
- No layer contains placeholders, hidden branch decisions, or work owned by another layer.

Use mechanical checks only for packaging, syntax, links, or executable task evidence where those checks actually test the artifact. Do not present structural lint as proof that the abstraction is matched.

## Resources

Read `references/layer-contract.md` when boundaries are unclear. Read `references/layer-prompts.md` only when delegating layer-owned work.

## Red Flags

- Approach output contains checkboxes, commands, sequencing, or file-edit recipes.
- Plan output reopens Approach decisions or collapses into a task checklist.
- Task output lacks RED evidence or says work is complete without commands/results.
- A lower layer becomes authority for a higher layer.
- The agent asks the user to approve descent before naming the artifact set hypothesis and recommended path.
- The agent patches around a late discovery instead of returning to the layer whose boundary broke.
- The orchestrator edits through uncertainty instead of escalating upward.
