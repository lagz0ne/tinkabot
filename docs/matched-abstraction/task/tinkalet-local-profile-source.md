---
layer: task
topic: tinkalet-local-profile-source
status: complete
references:
  - ../approach/tinkalet-edge.md
  - ../plan/tinkalet-edge.md
  - ./profile-trigger-tour.md
  - ./tinkalet-cli-profile.md
---

# Tinkalet Local Profile Source Task

## Objective

Complete Slice 1 sub-Task 2: the Tinkabot daemon writes the machine-readable
local profile descriptor that `tinkalet-cli-profile` already imports. A user can
start Tinkabot with an isolated store and then run
`tinkalet profile import local --store <dir> --name local` without copying a
printed NATS URL or raw credential path.

## Scope

Owns:

- `<store>/local-profile.json` emitted by Tinkabot startup.
- Descriptor fields: kind, dynamic NATS client URL, dynamic shell URL,
  relative caller credential path, role `caller`, trust `local-owner`, and
  source `local-store:<absolute-store-dir>`.
- Descriptor file mode `0600`.
- Cross-package proof that `tinkalet` imports the descriptor from a real
  Tinkabot store and redacts credential contents.

Does not own live trigger request/reply, duplicate proof, revocation,
denied-neighbor, package archive smoke, README/example promotion, or release
evidence.

## Acceptance Contract

- `go test ./tinkabot -run TestLocalProfileDescriptor -count=1` from
  `substrate/go` passes over a real Tinkabot assembly.
- The test proves Tinkabot writes `<store>/local-profile.json` with dynamic
  endpoints from `app.Posture()`, a relative `caller.creds` reference, role
  `caller`, trust `local-owner`, and source `local-store:<store>`.
- The test proves `tinkalet profile import local --store <store> --name local`
  succeeds against that real store and `profile list --json` includes endpoints
  but not credential contents.

## RED Artifact

RED tests were added in `substrate/go/tinkabot/local_profile_test.go`.

Executed 2026-06-17 from `substrate/go`:

- `go test ./tinkabot -run TestLocalProfileDescriptor -count=1` -> expected
  failure before implementation because `<store>/local-profile.json` is not yet
  written by Tinkabot.

## Verification Evidence

- `go test ./tinkabot -run TestLocalProfileDescriptor -count=1` -> RED:
  `local profile descriptor missing: open <store>/local-profile.json: no such
  file or directory`.
- `go test ./tinkabot -run TestLocalProfileDescriptor -count=1` -> `ok` after
  Tinkabot startup writes the descriptor and Tinkalet imports it.

## Execution Notes

Implemented 2026-06-17 as a small Tinkabot startup materialization step after
the shell endpoint is known. `local-profile.json` is written with mode `0600`
and includes only product metadata: dynamic NATS client URL, dynamic shell URL,
relative `caller.creds`, role `caller`, trust `local-owner`, and
`local-store:<absolute-store-dir>`.

## Residual Risk

Import still proves profile setup only. The next sub-Task must make
`tinkalet trigger bundle.clock.tick` use the imported endpoint and credential
over real embedded NATS.
