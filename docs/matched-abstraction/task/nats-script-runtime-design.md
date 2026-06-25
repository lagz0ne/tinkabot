---
layer: task
topic: nats-script-runtime
references:
  - ../approach/platform-structure.md
  - ../plan/nats-script-runtime.md
  - ../approach/nats-script-runtime.md
---

# NATS Script Runtime Design Task

## Platform Reset Supersession

`docs/matched-abstraction/approach/platform-structure.md` supersedes the Bun-local substrate decision recorded in this task. This task remains historical design evidence for script metadata, mediated NATS access, framed stdio RPC, and edge-case strictness.

## Task Brief

Capture the approved NATS TypeScript script-runtime design into matched abstraction docs and keep the project handoff current. This task is documentation and design validation only; it does not implement the runtime.

Scope includes the new Approach and Plan docs for the feature, this Task doc, and `tasks/todo.md`.

## Acceptance Contract

The task is accepted when the feature docs preserve the approved abstraction boundaries, the Plan carries Approach decisions without redefining them, edge cases are represented as first-class success criteria, and the Task records concrete verification evidence.

The future vertical proof is accepted only when one Bun-driven proof covers:

- success: store, load exact revision, execute, reply, event, cleanup.
- validation failure: invalid metadata is rejected before execution.
- record failure: missing record and revision mismatch do not execute the script.
- mediation failure: undeclared import and denied publish fail at the mediated boundary.
- runtime failure: script throw or timeout still returns a typed reply and attributed event.
- recovery: a valid execution after denial or failure still succeeds.
- process facade: script uses framed stdio RPC for input/result/progress/publish protocol messages, writes diagnostics to stderr, and relies on Tinkabot to validate and forward allowed publish requests to NATS.

## RED Artifact

- `sed -n '1,140p' tasks/todo.md` -> showed NATS feature inputs and open brainstorm items without persisted feature layer docs.
- `bun --version` -> `1.3.14`.
- `nats-server --version` -> `command not found`.
- `npm view @lagz0ne/nats-embedded version --json` -> `0.3.1`.
- local source inspection -> `NatsServer.start`, `server.url`, `server.port`, `server.exited`, and `server.stop()` are available in `/home/lagz0ne/dev/nats-embedded/packages/nats-embedded/src/index.ts`.

## Execution Notes

Superseded design evidence: the prior design used `@lagz0ne/nats-embedded` as the v1 local NATS provider because it matched the Bun-managed server strategy and avoided global `nats-server` drift. Current platform authority moved substrate ownership to Go.

The base script model is language-agnostic and supports Bash through protocol adapters or helpers: framed stdio RPC for protocol messages and stderr for diagnostics. NATS client/CLI access is not required for default scripts.

Feature diagram: https://diashort.apps.quickable.co/d/cba2160a

Proof flow diagram: https://diashort.apps.quickable.co/d/fd5ae8fd

## Verification Evidence

- Edge-case pressure passes were completed by dedicated Approach, Plan, and Task subagents before this hardening edit.
- Process facade correction added: default scripts are NATS-agnostic and interact with NATS through Tinkabot/runtime IPC forwarding.
- IPC hardening added: canonical IPC is framed stdio RPC; fd-specific channels are adapters, not the domain contract.

## Wrap-Up Announcement

The final response for this design pass must state the approved direction, docs written, validation commands, and remaining open Plan questions.
