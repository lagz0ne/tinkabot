---
layer: task
topic: bundle-transform
status: complete
references:
  - ../approach/bundle-v1.md
  - ../plan/bundle-v1.md
---

# Bundle Transform Task

## Brief

Bundle-v1 slice 4, per the Plan's amended decomposition: chain-reaction
inside the bundle. A manifest entry with `watches: <projection short id>` is
a long-lived filter process — spawned once with the bundle, backend-like —
fed each change of the watched projection as one JSON line on stdin (the
stored projection record; session-steer feed precedent), emitting framed
effects on stdout continuously. Every effect passes the entry's script
policy and the materializer gate into the entry's own granted projections;
the frontend consumes only the derived view. The platform keeps the
watching: `SourceRouter.KV` activations over the bundle account's material
bucket, with revision cursors, dedupe, stale rejection, and the ledger's
LoopSuppressed backstop. Loop safety is load-time structure: the watches
graph must be a DAG, watched projections must be declared by another entry,
and a filter entry is only a filter (no boot, no every, outputs required).

New embednats surface: `RouterResult.Payload` (KV route carries
`entry.Value()`; the schema's `payload` slot grown where it is needed),
`core.ScriptRuntime.Allow` (exported effect gate), and `FilterLoop`
(spawn-once feed/stream sibling of `ScriptLoop`, lazy respawn on exit,
process-group kill on stop, no run claims — single instance per run).

## Acceptance Contract

- `go test ./embednats -run TestFilterLoop -count=1` passes: a real sh
  filter transforms two fed payloads into two materialized projections with
  growing sequence (FilterTransforms); a filter that exits after one line is
  respawned by the next feed (FilterRespawns); stop kills the process group
  (StopKills).
- `go test ./tinkabot -run TestBundle -count=1` passes: TransformPipe — a
  scheduled state entry plus a `watches` filter entry yield a derived
  `bundle.t.view` that appears and follows the watched projection with no
  manual trigger anywhere, and the value is actually transformed
  (doubled == 2×source); TransformRejected — self-watch, two-entry cycle,
  unknown watched id, watches+every, watches+boot, and watches without
  outputs are all typed BundleRejected at load.
- The full standing battery stays green; `examples/clock` grows a `present`
  filter deriving `bundle.clock.view`, and the live browser page consumes
  only the derived projection.

## RED Artifact

Executed 2026-06-12: `go test ./tinkabot -run TestBundle/TransformPipe
-count=1` -> `BundleRejected: bundle manifest could not be decoded: json:
unknown field "watches"` — the transform contract did not exist.

## Verification Evidence

GREEN executed 2026-06-12, delegated build (three subagents: embednats
primitive, tinkabot wiring, example) with the contract and integration vetted
by the orchestrator.

`go test ./embednats -run TestFilterLoop -count=1` -> `ok` — a real sh
filter transformed two fed payloads into two materialized projections with
growing sequence; a filter exiting after one line was respawned by the next
feed; stop killed the process group. `TestSourceRouter` stayed green with
the Payload addition; `go test ./core -count=1` -> `ok` with the exported
`Allow`.

`go test ./tinkabot -run TestBundle -count=1` -> `ok`, all subtest families
— TransformPipe: with a 300ms scheduled state entry and a `watches` filter
entry, the derived `bundle.t.view` appeared and followed the watched
projection with no manual trigger anywhere, and the value was transformed
(doubled == 2×sourceNs); TransformRejected: SelfWatch, Cycle, UnknownWatch,
WatchWithEvery, WatchWithBoot, WatchWithoutOutput all typed BundleRejected
at load. Pre-existing families (AppServes, ScheduledTicks,
EphemeralAcrossRestart, InvalidNames, MalformedManifest) unregressed.

`go test ./... -count=1` -> all 9 packages ok; full standing battery PASS.

Live browser 2026-06-12: `--bundle examples/clock` — the page consumes only
`/projections/bundle.clock.view`; the `present` filter derived
display-shaped values (`"display":"clock at <iso>"`) from raw
`bundle.clock.state` ticks, advancing on the 5s cadence with no trigger.

## Scope

Owns:

- `substrate/go/core/script_materializer.go` — exported `Allow`.
- `substrate/go/embednats/source_router.go` — `RouterResult.Payload` (KV
  route only); `substrate/go/embednats/filter_loop.go` + test.
- `substrate/go/tinkabot/bundle.go` — `watches` manifest field, two-pass
  validation (DAG, declared-watch, filter-only combos), KV-watch route +
  FilterLoop wiring inside the bundle account, router perms growth by
  `readKV(tb_material)`.
- TransformPipe/TransformRejected tests; `examples/clock` present filter.

Does not own:

- Chain inheritance / hop derivation (deferred until cross-bundle or
  dynamic chains exist; the ledger hop check stays the backstop).
- Cross-bundle or app-plane watches; object/stream/schedule-sourced
  filters; zip (Plan slice 5).
