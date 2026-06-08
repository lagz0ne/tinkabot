---
layer: task
topic: nats-script-runtime-traced-tdd
references:
  - ../approach/platform-structure.md
  - ../plan/nats-script-runtime-traced-tdd.md
  - ../plan/platform-structure.md
  - ../plan/nats-script-runtime.md
  - ../approach/nats-script-runtime.md
---

# NATS Script Runtime Traced TDD Task

## Platform Reset Supersession

`docs/matched-abstraction/plan/platform-structure.md` supersedes this task wherever substrate ownership, local runtime orchestration, or release platform structure is concerned. This task remains historical planning evidence for layer-owned test design, typed error contracts, and the earlier Bun vertical-proof suite.

## Task Brief

Create the test-first implementation planning artifact for the NATS script runtime. This task does not implement runtime code. It defines the layer graph, typed error sets, Resolve / Transform / Propagate policy, protocol test contract, vertical proof suite, and concrete fixtures required before coding.

Scope includes `docs/matched-abstraction/plan/nats-script-runtime-traced-tdd.md`, this Task doc, and `tasks/todo.md`.

## Acceptance Contract

The task is accepted when the test plan maps every runtime layer to typed errors, every consumed error has a Resolve / Transform / Propagate policy, every test has one owning layer, the protocol contract is concrete enough to write RED tests, and the vertical proof is explicitly last rather than the only test.

The plan must preserve the design constraints: no default raw NATS for scripts, framed stdio RPC as canonical process protocol, NATS auth vocabulary as authority, exact KV revision attribution, strict negative cases, and cleanup/rerun safety.

## RED Artifact

- `sed -n '1,260p' docs/matched-abstraction/plan/nats-script-runtime.md` -> showed Plan contracts and edge-case matrix but no typed error ownership table.
- `sed -n '1,220p' docs/matched-abstraction/task/nats-script-runtime-design.md` -> showed vertical proof acceptance but no dependency-ordered test plan.
- `find /home/lagz0ne/.agents/skills -maxdepth 2 -name 'SKILL.md' -print` -> showed `traced-tdd` is available while `test-driven-development` is not installed in this environment.
- Test ownership graph rendered: https://diashort.apps.quickable.co/d/d29e5453.
- Error ownership graph rendered by the layer-contract subagent: https://diashort.apps.quickable.co/d/90f4566b.
- Protocol graph rendered by the protocol subagent: https://diashort.apps.quickable.co/d/0da56487.
- Vertical proof graph rendered by the embedded-NATS integration subagent: https://diashort.apps.quickable.co/d/12a339dc.

## Execution Notes

This planning pass uses `traced-tdd`: dependency graph first, typed errors second, acknowledgment table third, tests last. It intentionally stops before implementation.

Layer subagents protected the abstraction split:

- Layer-contract subagent supplied declared errors and Resolve / Transform / Propagate ownership.
- Protocol subagent supplied the JSON-RPC 2.0 over `Content-Length` stdio contract and protocol-only tests.
- Historical vertical-proof subagent supplied the real `@lagz0ne/nats-embedded` JetStream suite and fixtures.

The resulting implementation plan should be written from `docs/matched-abstraction/plan/nats-script-runtime-traced-tdd.md` and should not start with the vertical proof. Lower-layer tests must fail and pass first.

Implementation order for the next Task pass:

1. RED substrate and record-store tests.
2. RED metadata/schema and imports/permissions tests.
3. RED framed stdio RPC and process-runtime tests.
4. RED event-trail and execution-exchange tests.
5. RED historical vertical proof using embedded NATS and KV history; current platform proof must be re-owned by the Go substrate lane.
6. GREEN only the minimum runtime code needed to satisfy the declared contracts.
7. REFACTOR with no-slop, simplify, and review passes.

## Verification Evidence

- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.
- `find . -type d -name __pycache__ -print` -> no matches.

## Wrap-Up Announcement

The final response must state whether implementation tasks are now ready, which test layers must be written first, and which decisions remain open but not blocking.
