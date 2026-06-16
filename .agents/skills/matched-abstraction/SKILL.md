---
name: matched-abstraction
description: Use when design, planning, implementation, or delegation risks mixing Approach, Plan, and Task abstraction layers; triggers include architecture decisions, implementation plans, task handoffs, layered docs, orchestration, decomposition, verification strategy, and strategy-to-execution drift.
---

# Matched Abstraction

Use top-down layer discipline: Approach constrains thinking, Plan decomposes coordination, and Task executes one bounded unit with evidence. Lower layers may cite higher or peer artifacts, but must not redefine them.

## First Move

Announce RED-GREEN-TDD as the verification posture for non-trivial work. Identify the current layer before producing output. Explore locally answerable facts before asking the user.

If a branch decision blocks the current layer, ask one branch-resolving question at a time. Include the recommended answer and the evidence behind it. If the current layer already allows a safe default, state the default and continue.

## Workflow

1. Approach: define purpose, scope, non-goals, invariants, decision hierarchy, reference policy, and Plan-readiness.
2. Plan: consume Approach authority and produce decomposition, dependency ordering, handoff contracts, verification strategy, and escalation rules.
3. Task: execute one bounded unit with scope, acceptance criteria, RED artifact, execution notes, verification evidence, and wrap-up.
4. Orchestrate: synthesize layer outputs, preserve authority direction, then verify.

One layer owner handles one layer. Do not proceed downward while the current layer has unresolved branch decisions.

## Layer Rules

| Layer | Owns | Must reject |
| --- | --- | --- |
| Approach | Intent, principles, invariants, non-goals, decision authority | Sequencing, file work, task lists, commands |
| Plan | Decomposition, dependencies, handoff contracts, verification strategy | New Approach decisions, file-level recipes, task checklists |
| Task | One executable unit, RED proof, implementation evidence, final announcement | Architecture changes, new decomposition, vague verification |

## Output Modes

Inline mode: when file persistence is unavailable, unnecessary, or explicitly forbidden, return document-shaped Approach, Plan, and Task outputs inline. Keep the same layer boundaries and state which persistence or executable validation was skipped because of the request constraint.

Persistence mode: when the caller or repository provides a layer root, store artifacts under that root using `approach/`, `plan/`, and `task/` directories. Each persisted document needs `layer`, `topic`, and `references` frontmatter. Reference direction is top-down: Approach -> Approach; Plan -> Approach/Plan; Task -> Approach/Plan/peer Task.

## Validation Contract

The authoritative check is the layer contract, not a tool. Before claiming readiness, verify:

- Approach has scope, layer contract, and Plan-readiness.
- Plan names consumed Approach authority, decomposition, and verification strategy.
- Task has acceptance contract, RED artifact, and concrete command/result evidence.
- References point only to allowed layers.
- No layer contains placeholders, hidden branch decisions, or work owned by another layer.

If this skill's optional executable mirror is available, run:

```bash
node <skill-dir>/scripts/validate-layers.mjs <layer-root>
```

Treat the result as a convenience check. If tool output and the layer contract disagree, the contract wins and the tool needs correction.

## Resources

Read `references/layer-contract.md` when boundaries are unclear. Read `references/layer-prompts.md` only when delegating layer-owned work.

## Red Flags

- Approach output contains checkboxes, commands, sequencing, or file-edit recipes.
- Plan output reopens Approach decisions or collapses into a task checklist.
- Task output lacks RED evidence or says work is complete without commands/results.
- A lower layer becomes authority for a higher layer.
- The orchestrator edits through uncertainty instead of escalating upward.
