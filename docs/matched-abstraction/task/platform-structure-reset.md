---
layer: task
topic: platform-structure
references:
  - ../approach/platform-structure.md
  - ../plan/platform-structure.md
---

# Platform Structure Reset Task

## Task Brief

Persist the platform-structure reset as matched-abstraction authority and mark prior Bun substrate language as superseded evidence.

## Acceptance Contract

- Platform Approach and Plan docs exist and validate.
- Existing NATS script runtime docs no longer treat Bun substrate ownership as current platform authority.
- `tasks/todo.md` records Go substrate, Vite frontend, JSON Schema source, Zod validation, and codegen parity as current direction.
- No Go, Vite, schema, Zod, Docker, or codegen implementation scaffold is created in this task.
- Verification evidence records the stale-authority RED scan and final validation outputs.

## RED Artifact

`rg -n "Bun owns|Bun runtime substrate|Bun-managed|@lagz0ne/nats-embedded|embedded JetStream|Bun may own|v1 uses" docs/matched-abstraction tasks/todo.md` -> stale active Bun substrate authority appeared in `tasks/todo.md`, `docs/matched-abstraction/approach/nats-script-runtime.md`, and `docs/matched-abstraction/plan/nats-script-runtime.md`.

## Execution Notes

The task persists the user-approved reset only. It does not choose a Go framework, Vite plugin stack, schema generator, or Docker sandbox strategy.

## Verification Evidence

- `rg -n "Bun owns|Bun runtime substrate|Bun-managed|@lagz0ne/nats-embedded|embedded JetStream|Bun may own|v1 uses" docs/matched-abstraction tasks/todo.md` -> remaining matches are explicitly marked as existing evidence, superseded evidence, historical proof material, or not current authority.
- `rg -n "UNRESOLVED|FIXME|XXX" docs/matched-abstraction/approach/platform-structure.md docs/matched-abstraction/plan/platform-structure.md` -> no matches.
- `find . -type d -name __pycache__ -print` -> no output.

## Wrap-Up Announcement

Platform-structure reset docs and handoff updates are persisted. Bun-centric substrate language is no longer active platform authority; it is retained only as superseded evidence.
