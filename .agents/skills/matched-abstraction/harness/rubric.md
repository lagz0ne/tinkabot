# Matched-Abstraction Harness Rubric

Score each category from 0 to 3.

- 0: absent, contradictory, or unusable.
- 1: present but shallow, generic, or not connected to the topic.
- 2: adequate and topic-specific, with minor gaps.
- 3: strong, reviewable, and directly useful for downstream task derivation and matching.

## Categories

| Category | What To Reward |
| --- | --- |
| Layer traversal | Current layer, target depth, Approach-to-Plan movement, and Plan-only stopping boundary are explicit. |
| Approach sufficiency | Purpose, scope, non-goals, invariants, decision hierarchy, reference policy, and Plan-readiness are concrete for the topic. |
| Artifact set fitness | The output chooses a small, non-overlapping artifact set and classifies use-now, deferred, upstream, or merged artifacts. |
| Derive/match continuity | Artifacts provide derivation hints for downstream work and matching hints for detecting drift back against owning intent. |
| Decomposition quality | Plan slices are coherent, bounded, ordered, and expose dependencies and parallelism. |
| Handoff contracts | Task-bound units have inputs, outputs, acceptance expectations, and escalation triggers. |
| Verification strategy | Testing and acceptance plans are concrete enough for future RED-GREEN-TDD Task work. |
| Domain fit | The system design reflects real workflow, state, users, integrations, and operational exceptions for the topic. |
| Upward traversal | The output identifies when a Plan discovery would require returning to Approach rather than redefining it inside Plan. |
| Reviewability | A reviewer can see what is decided, what is assumed, what remains open, and why Plan is sufficient. |

Maximum score: 30.

## Score Caps

- Cap at 12 if there is no recognizable Plan.
- Cap at 16 if there is no recognizable Approach.
- Cap at 18 if the output creates implementation Tasks, code, or command execution instead of stopping at Plan.
- Cap at 20 if the Plan is mostly a generic checklist rather than topic-specific decomposition.
- Cap at 21 if artifacts are named only as generic supporting material without depth, timing, or derive/match value.
- Cap at 23 if multiple artifacts are classified use-now but only named, with no compact rendering or sufficiency explanation.
- Cap at 22 if future Tasks require major scope invention because important Plan commitments are missing.
- Cap at 24 if the artifact set is broad but overlapping, creating rabbit holes instead of clearer downstream elaboration and matching.

## Review Notes

Apply caps after category scoring. A capped output can still receive useful category feedback, but the final score cannot exceed the cap.

Do not use keyword matching as evidence of quality. Reward only content that is specific enough to guide or constrain downstream Task work.
