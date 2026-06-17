---
layer: task
topic: tinkalet-release-docs-and-proof
status: complete
references:
  - ../approach/tinkalet-edge.md
  - ../plan/tinkalet-edge.md
  - ./profile-trigger-tour.md
  - ./tinkalet-package-tour.md
  - ./tinkalet-scenario-gate.md
---

# Tinkalet Release Docs And Proof Task

## Objective

Promote the completed Slice 1 Tinkalet profile-trigger tour into the release
proof surface without overclaiming later Tinkalet slices. The manual should
show the package-root Tinkalet tour, the package smoke should be a release gate,
and centralized release evidence should fail if that gate or its manual
commands disappear.

## Scope

Owns:

- `docs/manual/v1.md` Tinkalet profile-trigger section.
- `package.json` `gate:tinkalet-package` script backed by the package smoke.
- `scripts/smoke-tinkalet-package.sh` stable pass line.
- `scripts/release-evidence.ts` required gate extension and manual-command
  citation check for any gate that cites manual/verbatim evidence.
- `release/v1.json` gate result and Tinkalet doc authority entry.
- Synthetic checker coverage for the new required gate and manual-command
  citation path.

Does not own item records, watch cursors, reactions, schedules, remote pairing,
or publishing an actual GitHub Release.

## Acceptance Contract

- `bun run gate:tinkalet-package` passes from the source checkout.
- `bun run release:evidence` fails if the `gate:tinkalet-package` result is
  omitted from `release/v1.json`.
- `bun run release:evidence` fails if the package gate cites a Tinkalet manual
  command not present in `docs/manual/v1.md`.
- `bun run release:evidence` passes on the final corpus with six gate results.
- `bun run gate:manual` still passes, proving the existing NATS manual pairs
  were not broken by adding Tinkalet docs.

## RED Artifact

Executed 2026-06-17 before GREEN:

- `bun run release:evidence` -> passed while validating only five gate results,
  so the centralized release gate could omit the Tinkalet package tour.
- Synthetic checker intent: adding `gate:tinkalet-package` to required gates
  without a manifest entry must produce `gate-result-missing`; citing a missing
  Tinkalet manual command must produce `manual-divergence`.

## Verification Evidence

- `bun run gate:tinkalet-package` -> `gate:tinkalet-package passed`.
- `bun test tests/release-evidence.test.ts` -> pass, including the new package
  gate manual-command divergence case.
- `bun run release:evidence` -> pass:
  `release evidence check passed: 17 milestones over 12 spine steps, 6 gate
  results`.
- `bun run gate:manual` -> `gate:manual passed`.

## Execution Notes

Implemented 2026-06-17. The manual now distinguishes NATS as the substrate
seam from Tinkalet as the package product CLI for local profile import and the
clock bundle trigger. The package smoke remains the executable proof: it builds
the release archive, starts packaged `tinkabot`, imports/selects the local
profile with packaged `tinkalet`, renames `libexec/tinkabot/nats`, triggers
`bundle.clock.tick`, and observes the projection change. `release/v1.json`
records that gate and cites the manual commands; the checker validates both the
gate result line and manual command presence.

## Residual Risk

This proves only the Slice 1 profile-trigger tour. Schedule control remains a
diagnostic NATS operation until the schedule slice owns product commands.
Package publication is still deferred; the gate builds and tests the archive
locally.
