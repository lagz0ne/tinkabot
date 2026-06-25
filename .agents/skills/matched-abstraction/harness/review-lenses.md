# Review Lenses

Use these lenses to make review consistent across agents. They are evaluator aids, not generation prompts.

## Abstraction Lens

Check whether the output spends enough time at Approach before Plan, whether the Plan consumes Approach authority without redefining it, and whether the run stops at Plan without Task execution.

## Delivery Lens

Check whether future tasks are derivable from the Plan without invention and whether later work has clear matching surfaces back to Plan and Approach intent. Missing Plan scope should be treated as missing future delivery, even if the topic feels obvious.

## Artifact Set Lens

Check whether the output chooses a small, non-overlapping artifact set with distinct reasoning jobs. Reward artifacts that make the layer more compelling and downstream elaboration easier. Penalize artifact lists that are decorative, duplicative, or likely to become rabbit holes.

For use-now artifacts, check whether the output renders a compact artifact at the current depth or explains why naming alone is sufficient. Penalize unrendered use-now artifacts because they leave downstream agents guessing.

Check critical surfaces: if the topic has conflicting truth sources, unreliable external events, stateful sessions, money movement, entitlement rules, manual correction, permissions, or recovery paths, reward compact reconciliation, exception/recovery, and external-boundary artifacts. Penalize deferring those surfaces when they are central to Plan derivation and matching.

Check central operability surfaces: if the topic has differentiated roles, privileged/manual actions, a central object that changes state over time, correctness depending on precedence or quantity math, or requested downstream testing/integration/rollout work, reward compact authority, lifecycle/truth-state, rule/quantity example, and rollout/acceptance artifacts. Penalize plans that leave future Tasks to invent role permissions, state meaning, availability math, rule precedence, or rollout readiness.

For rule/quantity surfaces, check for representative examples, not only vocabulary. Reward at least one normal row and one conflict or exception row when later tests need precedence, formula, or quantity-bucket meaning.

When one output spans multiple layers, check cross-layer defer follow-through. If Approach defers a central artifact to Plan and the Plan is included, the Plan should render it, merge it into a rendered artifact, or explicitly defer it below Plan with a reason.

Check final artifact hygiene: every use-now artifact should be rendered, explicitly merged into a rendered artifact, or downgraded. Penalize plural sequence diagrams, API sketches, test matrices, or rollout plans marked use-now without representative rows.

## Depth And Timing Lens

Check whether each artifact speaks at the right depth for the layer. Reward clear use-now, defer, return-upward, and drop-or-merge decisions. Treat "this belongs later" and "this should have been settled upstream" as healthy signals when justified.

## Operations Lens

Check whether the plan understands the real-world workflow, exceptions, state changes, and operational recovery paths for the system.

## Interface Lens

Check whether external systems, users, devices, APIs, and data boundaries are explicit enough for task handoff.

## Verification Lens

Check whether the Plan includes a credible verification strategy, including RED artifacts future Task owners can create, integration checks, state-transition tests, and operational acceptance evidence.

## Reviewability Lens

Check whether the output gives the user enough diagrams, matrices, contracts, or examples to confirm that descending from Approach to Plan was sufficient and later work can be matched back without guessing.
