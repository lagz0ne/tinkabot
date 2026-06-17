---
layer: task
topic: tinkalet-scenario-gate
status: complete
references:
  - ../approach/tinkalet-edge.md
  - ../plan/tinkalet-edge.md
  - ./profile-trigger-tour.md
  - ./tinkalet-trigger-live.md
  - ./tinkalet-authority-denials.md
  - ./tinkalet-package-tour.md
---

# Tinkalet Scenario Gate Task

## Objective

Close the Slice 1 gate hole: `gate:scenarios` must fail if the Tinkalet trigger
outside-in surface is absent, and the real matrix must cite committed tests for
all seven pinned case families.

## Scope

Owns:

- Requiring `tinkalet-trigger` in `scripts/gate-scenarios.ts`.
- Detector proof in `tests/gate-checkers.test.ts`.
- `substrate/go/scenario-matrix.json` entries for the Tinkalet trigger surface.
- A real malformed-response test so the matrix does not cite an unrelated parse
  error for the trigger malformed family.

Does not own centralized `release/v1.json` promotion or manual public-support
claims; those remain the later release-docs-and-proof slice.

## Acceptance Contract

- `go test ./tinkalet -run TestTriggerMalformedResponse -count=1` from
  `substrate/go` passes.
- `bun test tests/gate-checkers.test.ts -t "required Tinkalet trigger surface"
  ` passes and proves omission is detected.
- `bun run gate:scenarios` passes on the real corpus with
  `tinkalet-trigger` present.

## RED Artifact

Before this Task, `gate:scenarios` validated only surfaces present in the
matrix. Removing or omitting `tinkalet-trigger` would not fail the gate.

Executed 2026-06-17:

- `bun test tests/gate-checkers.test.ts -t "required Tinkalet trigger surface"`
  -> expected failure before implementation because no detector existed.

## Verification Evidence

- `go test ./tinkalet -run TestTriggerMalformedResponse -count=1` -> `ok`.
- `bun test tests/gate-checkers.test.ts -t "required Tinkalet trigger surface"`
  -> `ok`.
- `bun run gate:scenarios` -> `gate:scenarios passed`.

## Execution Notes

Implemented 2026-06-17. The malformed trigger test starts a small embedded
NATS server, registers a responder that returns `accepted-but-not-really`, and
asserts Tinkalet renders `malformed-response`. The scenario matrix now cites
the new Tinkalet allowed, duplicate, malformed, denied-neighbor, stale,
revoked, and attributed-failure coverage. The checker hardcodes
`tinkalet-trigger` as a required outside-in surface so the matrix cannot drop
the surface silently.

## Residual Risk

The scenario matrix is still a citation gate, not a semantic test runner.
Semantic coverage lives in the cited Go tests and the package smoke.
