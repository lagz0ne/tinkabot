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
  - ./tinkalet-schedules.md
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

Does not own item records, watch cursors, reactions, schedule implementation,
remote pairing, or publishing an actual GitHub Release. Later slices may extend
the package gate transcript when they add product commands; the owning slice
keeps implementation evidence, and this release proof keeps the gate visible in
centralized evidence.

## Artifact Pack

Depth: operational. This pack exists to make release-doc promotion auditable,
not to reopen Tinkalet product design.

Before/after tour:

```text
before Slice 1:
  manual teaches NATS CLI as the public operating path
  release evidence has no Tinkalet package gate

after Slice 1:
  manual includes package-root Tinkalet profile/import/trigger commands
  gate:tinkalet-package proves packaged tinkalet without the NATS sidecar
  release evidence requires that gate and cites the manual commands

after Slice 5:
  same package gate also proves schedule set -> item get -> schedule off
  release evidence cites the schedule transcript without moving schedule
  implementation authority out of tinkalet-schedules
```

Command transcript sketch:

```text
./tinkabot --store "$STORE" --shell 127.0.0.1:8419 --bundle examples/clock
./tinkalet profile import local --store "$STORE" --name local
./tinkalet profile use local
./tinkalet trigger bundle.clock.tick --request-id req-clock-1
./tinkalet schedule set packagetick --every 200ms --write package/schedule/tick --value '{"kind":"package-schedule"}'
./tinkalet item get package/schedule/tick --json
./tinkalet schedule off packagetick
```

Package contents map:

```text
release archive root
  tinkabot
  tinkalet
  examples/
  release.json
  libexec/tinkabot/bwrap
  libexec/tinkabot/nats  (renamed before Tinkalet commands in the package gate)
```

Failure matrix:

| Family | Expected proof |
| --- | --- |
| stale docs | release evidence fails when cited manual commands disappear |
| package missing Tinkalet | package gate fails before profile import |
| sidecar mismatch | package gate renames bundled `nats`; Tinkalet commands still work |
| manual divergence | release evidence reports `manual-divergence` |
| gate omission | release evidence reports `gate-result-missing` |
| schedule overclaim | schedule implementation evidence remains in `tinkalet-schedules` |

Test intent map:

- Package smoke: `bun run gate:tinkalet-package`.
- Manual presence: `release/v1.json` `verbatim` commands are found in
  `docs/manual/v1.md`.
- Checker negatives: `tests/release-evidence.test.ts` covers missing gate and
  manual divergence.
- Release evidence: `bun run release:evidence` validates six gate results.
- Manual gate: `bun run gate:manual` proves the NATS manual pairs still run.

Artifact fitness: use now. The before/after tour, transcript, contents map,
failure matrix, and test map are the smallest useful release-doc artifacts for
matching future package-gate extensions back to this Task. Defer publication
proof and broader Tinkalet positioning to future release work.

## Acceptance Contract

- `bun run gate:tinkalet-package` passes from the source checkout.
- `bun run release:evidence` fails if the `gate:tinkalet-package` result is
  omitted from `release/v1.json`.
- `bun run release:evidence` fails if the package gate cites a Tinkalet manual
  command not present in `docs/manual/v1.md`.
- `bun run release:evidence` passes on the final corpus with six gate results.
- `bun run gate:manual` still passes, proving the existing NATS manual pairs
  were not broken by adding Tinkalet docs.
- After the schedule slice, `release/v1.json` also cites the package-gate
  schedule transcript from `docs/manual/v1.md`.

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
- Post-schedule cleanup: `release/v1.json` now cites the schedule set, item get,
  and schedule off commands that the package gate executes.

## Execution Notes

Implemented 2026-06-17. The manual now distinguishes NATS as the substrate
seam from Tinkalet as the package product CLI for local profile import and the
clock bundle trigger. The package smoke remains the executable proof: it builds
the release archive, starts packaged `tinkabot`, imports/selects the local
profile with packaged `tinkalet`, renames `libexec/tinkabot/nats`, triggers
`bundle.clock.tick`, and observes the projection change. `release/v1.json`
records that gate and cites the manual commands; the checker validates both the
gate result line and manual command presence.

The schedule slice later extended the same package gate. That implementation
authority remains in `tinkalet-schedules`; this Task records the release
artifact shape and manifest citations so the public proof surface does not lag
behind the package smoke.

## Residual Risk

Package publication is still deferred; the gate builds and tests the archive
locally. Richer Tinkalet positioning, remote pairing, item/watch/reaction
manual tours, and package-publication proof remain future release work.
