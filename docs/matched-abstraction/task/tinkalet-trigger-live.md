---
layer: task
topic: tinkalet-trigger-live
status: complete
references:
  - ../approach/tinkalet-edge.md
  - ../plan/tinkalet-edge.md
  - ./profile-trigger-tour.md
  - ./tinkalet-cli-profile.md
  - ./tinkalet-local-profile-source.md
---

# Tinkalet Trigger Live Task

## Objective

Complete Slice 1 sub-Task 3: after importing a local profile from a real
Tinkabot store, `tinkalet trigger bundle.clock.tick` uses the selected profile's
managed caller credential to request the existing clock bundle trigger over real
embedded NATS. The user sees product-shaped accepted/duplicate output and a
visible clock projection changes.

## Scope

Owns:

- Tinkalet connecting with `nats.go` using the selected profile's managed
  credential copy.
- Product intent mapping for the taught Slice 1 intent:
  `bundle.clock.tick` -> hidden substrate subject `tb.bundle.clock.tick`.
- Request id header forwarding for deterministic duplicate proof.
- Normal output for accepted and duplicate replies.
- JSON output for accepted/duplicate/denied trigger replies.
- Outside-in proof through real Tinkabot + `examples/clock` and the shell
  projection endpoint.

Does not own broad trigger discovery, item/watch/reaction/schedule commands,
revocation/denied-neighbor/no-stronger-fallback matrix expansion, archive smoke,
or docs/release promotion.

## Acceptance Contract

- `go test ./tinkabot -run TestTinkaletTriggerClock -count=1` from
  `substrate/go` passes.
- The test starts Tinkabot with `examples/clock`, imports and selects the local
  profile through Tinkalet, triggers `bundle.clock.tick`, and observes
  `/projections/bundle.clock.state` advance.
- Reusing the same `--request-id` returns duplicate and does not advance the
  projection again.
- JSON trigger output names only product profile/intent/status/reason and does
  not leak raw `tb.` subjects or credential contents.

## RED Artifact

RED test added in `substrate/go/tinkabot/tinkalet_trigger_test.go`.

Executed 2026-06-17 from `substrate/go`:

- `go test ./tinkabot -run TestTinkaletTriggerClock -count=1` -> expected
  failure before implementation because Tinkalet trigger still returns
  `connection-failed`.

## Verification Evidence

- `go test ./tinkabot -run TestTinkaletTriggerClock -count=1` -> RED:
  Tinkalet returned `profile local denied bundle.clock.tick:
  connection-failed`.
- `go test ./tinkabot -run TestTinkaletTriggerClock -count=1` -> `ok` after
  Tinkalet used `nats.go` with the managed caller credential and request-id
  header.

## Execution Notes

Implemented 2026-06-17 in `substrate/go/tinkalet`: the taught product intent
`bundle.clock.tick` maps internally to `tb.bundle.clock.tick`, Tinkalet connects
with `nats.go` and the selected profile's managed credential copy, forwards
`--request-id` as `Tinkabot-Request-Id`, and renders `accepted`/`duplicate`
without exposing the raw subject in normal output. The test pauses the clock's
manifest schedule so duplicate no-rerun is attributable to request-id dedupe,
not a background tick.

## Residual Risk

This Task proves the happy path and duplicate no-rerun for one taught intent.
The next Task must expand denial and privacy coverage for revoked/stale creds,
denied-neighbor, malformed replies, and stronger-credential fallback refusal.
