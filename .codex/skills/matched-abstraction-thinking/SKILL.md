---
name: matched-abstraction-thinking
description: Use when design, planning, task execution, or subagent orchestration risks mixing Approach, Plan, and Task abstraction layers; triggers include architecture decisions, implementation plans, task handoffs, layer docs, and strategy/decomposition/execution drift.
---

# Matched Abstraction Thinking

Use top-down layer discipline: Approach constrains thinking, Plan decomposes coordination, Task executes one bounded unit and proves it. Lower layers may cite higher or peer docs, never redefine them.

## Opening Move

Announce RED-GREEN-TDD. Read `tasks/todo.md` first. If `.c3/` exists, use the `c3` skill before exploration or changes. Explore answerable local questions instead of asking.

When a visual would clarify the layer flow, use Diashort and store the shortlink in the relevant doc.

## Workflow

1. Approach: dispatch an Approach subagent to protect purpose, invariants, non-goals, decision hierarchy, and Plan-readiness.
2. Plan: dispatch a Plan subagent to consume Approach docs and produce decomposition, sequencing, handoff, verification, and escalation contracts.
3. Task: dispatch a Task subagent per executable unit to define scope, RED artifact, execution notes, verification evidence, and wrap-up.
4. Orchestrate: synthesize layer outputs, update docs, announce ready-to-change state, then verify.

One subagent owns one layer. Do not proceed downward while the current layer has unresolved branch decisions.

## Layer Rules

| Layer | Owns | Must reject |
| --- | --- | --- |
| Approach | Intent, principles, invariants, non-goals, decision authority | Sequencing, file work, task lists, commands |
| Plan | Decomposition, dependencies, handoff contracts, verification strategy | New Approach decisions, file-level recipes, task checklists |
| Task | One executable unit, RED proof, implementation evidence, final announcement | Architecture changes, new decomposition, vague verification |

## Grill-Style Progression

Do not trigger `grill-me`; reuse its posture. Ask one branch-resolving question at a time with the recommended answer. Discover local answers. Move down only after the current readiness gate passes.

## Documents

Store docs under `docs/matched-abstraction/`:

- `approach/`: charters, layer contracts, decision hierarchy, readiness gates.
- `plan/`: orchestration briefs, decomposition maps, handoff contracts, verification strategy, escalation logs.
- `task/`: task briefs, acceptance contracts, RED artifacts, execution notes, verification evidence, wrap-up announcements.

Each doc needs `layer`, `topic`, and `references` frontmatter. Reference direction is top-down: Approach -> Approach; Plan -> Approach/Plan; Task -> Approach/Plan/peer Task.

If the request forbids file edits or limits inputs, keep the same layer boundaries and return document-shaped Approach, Plan, and Task outputs inline. State that persistence and validation were skipped because of the explicit request constraint.

## Resources

Read `references/layer-contract.md` when boundaries are unclear. Read `references/subagent-prompts.md` before dispatching layer agents.

Run:

```bash
python3 .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction
```

## Red Flags

- Approach doc contains checkboxes, commands, or file-edit recipes.
- Plan doc reopens Approach decisions or collapses into a task checklist.
- Task doc lacks RED evidence or says work is complete without commands/results.
- A lower layer becomes authority for a higher layer.
- The orchestrator edits through uncertainty instead of escalating upward.
