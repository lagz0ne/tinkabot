---
layer: task
topic: tinkalet-package-tour
status: complete
references:
  - ../approach/tinkalet-edge.md
  - ../plan/tinkalet-edge.md
  - ./profile-trigger-tour.md
  - ./tinkalet-cli-profile.md
  - ./tinkalet-local-profile-source.md
  - ./tinkalet-trigger-live.md
  - ./tinkalet-authority-denials.md
---

# Tinkalet Package Tour Task

## Objective

Complete Slice 1 sub-Task 5: the release-shaped archive contains both
`tinkabot` and `tinkalet`, declares them in package metadata, and the packaged
Tinkalet tour starts the clock bundle, imports the local profile, triggers
`bundle.clock.tick`, and observes a projection change without depending on a
global or bundled NATS CLI for the product trigger path.

## Scope

Owns:

- Package scripts building `tinkalet` beside `tinkabot`.
- Release metadata declaring both packaged commands.
- A reusable package smoke that runs from the archive root.
- README and clock-example promotion of the Tinkalet profile/trigger tour.

Does not own npm wrapping, item/watch/reaction/schedule product commands,
manual release-evidence promotion, or replacing NATS diagnostic commands for
schedule controls.

## Acceptance Contract

- `bash -n scripts/package-tinkabot.sh scripts/release-package.sh
  scripts/smoke-tinkalet-package.sh` passes.
- `bun run smoke:tinkalet-package` builds a release archive, asserts executable
  `tinkabot`, `tinkalet`, bundled `bwrap`, and bundled `nats`, starts packaged
  `./tinkabot --bundle examples/clock`, imports/selects the local profile with
  packaged `./tinkalet`, removes the packaged NATS sidecar before product
  Tinkalet commands, triggers `bundle.clock.tick`, and observes the clock
  projection change.
- README and example docs show Tinkalet as the Slice 1 product trigger path;
  raw NATS commands remain only for diagnostics and schedule controls.

## RED Artifact

Before this Task, `scripts/package-tinkabot.sh` built only `tinkabot` and
sidecars; `scripts/release-package.sh` did not declare `tinkalet`; and no
release-archive smoke proved the Tinkalet tour.

Executed 2026-06-17:

- `bun run smoke:tinkalet-package` -> expected failure because the script and
  package script did not exist.

## Verification Evidence

- `bash -n scripts/package-tinkabot.sh scripts/release-package.sh
  scripts/smoke-tinkalet-package.sh` -> `ok`.
- `bun run smoke:tinkalet-package` -> `ok`: release archive built, package root
  unpacked, executable `tinkabot`/`tinkalet`/`bwrap`/`nats` asserted,
  packaged `tinkabot` started `examples/clock`, packaged `tinkalet` imported
  and selected the local profile under explicit config/data dirs, the packaged
  NATS sidecar was renamed before Tinkalet commands, and
  `profile local accepted bundle.clock.tick` changed the clock projection.

## Execution Notes

Implemented 2026-06-17. `scripts/package-tinkabot.sh` now builds
`substrate/go/cmd/tinkalet` with the same version ldflags as `tinkabot`.
`scripts/release-package.sh` records both commands in `release.json`.
`scripts/smoke-tinkalet-package.sh` exercises the release archive as a user
would: package root, explicit Tinkalet config/data dirs, local profile import,
selected profile, and product intent trigger. The smoke scrubs Tinkalet's
environment and removes the packaged NATS sidecar before trigger execution to
prove Tinkalet is using the Go client path, not shelling out.

## Residual Risk

The package tour still leaves schedule control as a diagnostic NATS operation.
That is intentional until the schedule slice adds product commands. Manual and
centralized release-evidence promotion remain separate release-doc work.
