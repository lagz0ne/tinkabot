---
layer: task
topic: tinkalet-cli-profile
status: complete
references:
  - ../approach/tinkalet-edge.md
  - ../plan/tinkalet-edge.md
  - ./profile-trigger-tour.md
---

# Tinkalet CLI Profile Task

## Objective

Implement the first bounded Slice 1 sub-Task from
`./profile-trigger-tour.md`: a `tinkalet` command and small Go package that
own profile configuration only. A user can import a local profile descriptor,
list profiles, select a default profile, and exercise profile selection through
a product-shaped trigger stub. This creates the local edge profile model
without claiming the live Tinkabot trigger path, package archive tour, docs
promotion, or daemon behavior.

## Scope

Owns:

- `substrate/go/cmd/tinkalet` help, version, usage, and exit-code behavior.
- `substrate/go/tinkalet` profile config/data-dir behavior.
- `profile import local --store <dir> --name <name>` over an existing
  `<store>/local-profile.json` descriptor.
- Managed credential copy to
  `<TINKALET_DATA_DIR>/profiles/<name>/caller.creds` with mode `0600`.
- `profile list`, `profile list --json`, `profile use <name>`, default lookup,
  `--profile` trigger selection, profile-not-found, and unknown-trigger output.
- No writes to `HOME`, `XDG_CONFIG_HOME`, or `XDG_STATE_HOME` when explicit
  `TINKALET_CONFIG_DIR` and `TINKALET_DATA_DIR` are provided.

Does not own:

- Tinkabot emitting `<store>/local-profile.json`.
- Real embedded-NATS trigger request/reply, accepted/duplicate semantics, or
  projection state change.
- Revocation, denied-neighbor, no-stronger-fallback live authority proofs.
- Release archive smoke, README/example promotion, scenario gate changes, or
  centralized release evidence.
- Tinkalet daemon/watch/reaction behavior.

## Acceptance Contract

- `go test ./cmd/tinkalet ./tinkalet -count=1` from `substrate/go` passes.
- `cmd/tinkalet` prints help and version, returns exit 2 for parse errors, and
  writes usage errors to stderr beginning with `usage: tinkalet `.
- `profile import local --store <dir> --name local` reads a fixture descriptor,
  copies the source credential into the Tinkalet data dir, writes profile config
  with mode `0600`, and prints exactly `profile local imported\n`.
- `profile list` and `profile list --json` match the umbrella Task's exact
  no-profile, imported-no-default, and selected-default oracles without leaking
  credential contents.
- `profile use <name>` writes `default-profile` mode `0600`, prints exactly
  `profile <name> selected\n`, and missing names fail as
  `profile <name> denied profile use: profile-not-found\n`.
- Import denial cases are product-shaped: missing descriptor is
  `import-source-missing`; malformed descriptor, absolute credential path,
  escaping credential path, and missing source credential are
  `import-source-invalid`.
- Trigger parsing resolves selected/default profiles and supports `--profile`
  and `--json` without contacting NATS. Missing default fails as
  `profile default denied <intent>: profile-not-found\n`; unknown intent with a
  selected profile emits denied JSON with reason `unknown-trigger`.

## RED Artifact

RED tests were added in `substrate/go/cmd/tinkalet/main_test.go` and
`substrate/go/tinkalet/tinkalet_test.go`. They reference the not-yet-existing
`run`, `Run`, and `Config` surfaces and assert the profile CLI contract above.

Executed 2026-06-17 from `substrate/go`:

- `go test ./cmd/tinkalet ./tinkalet -count=1` -> `FAIL`: `cmd/tinkalet/main_test.go:12:10: undefined: run`; `tinkalet/tinkalet_test.go:177:10: undefined: Run`; `tinkalet/tinkalet_test.go:177:14: undefined: Config`.

This is the correct RED: the tests compile far enough to prove the intended
missing command/package surface, and no live Tinkabot/NATS behavior is required
for this sub-Task.

## Verification Evidence

- `go test ./cmd/tinkalet ./tinkalet -count=1` -> `ok`: help/version/usage,
  profile import/list/use/default, managed credential copy, no user-home
  mutation, import denials, trigger profile override, and JSON denial output
  passed.
- `go run ./cmd/tinkalet --help` -> `usage: tinkalet <command> [options]`.
- `go run ./cmd/tinkalet --version` -> `tinkalet dev`.
- `go build -o /tmp/tinkalet-cli-profile ./cmd/tinkalet` -> `ok`.
- `go test ./... -count=1` -> `ok`: all eleven Go packages passed, including
  `cmd/tinkalet` and `tinkalet`.

## Execution Notes

Implemented 2026-06-17:

- `substrate/go/cmd/tinkalet` is a thin entry point with `--help`,
  `--version`, stable usage output, and testable `run()`.
- `substrate/go/tinkalet` owns the profile config/data-dir behavior. It reads
  `TINKALET_CONFIG_DIR` and `TINKALET_DATA_DIR` first, writes `profiles.json`,
  `default-profile`, and managed copied credentials with mode `0600`, and does
  not write to `HOME`, `XDG_CONFIG_HOME`, or `XDG_STATE_HOME` when explicit dirs
  are supplied.
- `profile import local` reads an existing `<store>/local-profile.json`, checks
  descriptor shape and credential containment, copies the selected credential to
  `profiles/<name>/caller.creds`, and stores only a relative credential
  reference.
- `profile list` and `profile list --json` are deterministic and sorted by
  profile name. JSON output is a decode-stable contract, not a pretty-print
  promise.
- `trigger` is intentionally a profile-selection stub in this sub-Task. It
  resolves default or `--profile`, emits product-shaped `profile-not-found`,
  `unknown-trigger`, `stale-credentials`, or `connection-failed` denials, and
  exposes endpoint diagnostics only under `--json`.

## Residual Risk

The trigger command remains a profile-selection stub in this sub-Task. The next
sub-Tasks must replace descriptor fixtures with Tinkabot descriptor emission and
replace the `connection-failed` edge with real embedded-NATS trigger proof
before any release tour presents Tinkalet as the primary operating path.
