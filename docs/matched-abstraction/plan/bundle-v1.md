---
layer: plan
topic: bundle-v1
references:
  - ../approach/bundle-v1.md
---

# Bundle v1 Plan

## Consumed Approach

`../approach/bundle-v1.md` is the sole thinking authority: ephemeral run
posture, disjoint authority, loader-as-author, unchanged trust posture,
strict decode with typed attributed failure. This Plan only decomposes and
sequences within those invariants.

## Decomposition

Two slices, strictly sequenced; slice 1 delivers the user-testable app.

### Slice 1: bundle-dir-app

Owns the directory form end to end. `--bundle <dir>` on the binary loads a
strictly-decoded `bundle.json` manifest; each entry binds one script source
file to one wired slot (script key, revision, trigger subject, grants:
projection ids + artifact prefix, optional boot). The loader validates
disjoint authority against the durable claims (wired script key and trigger
subject, events subject, projection `main`, artifact prefix `artifact/`,
existing durable script-bucket keys, intra-bundle duplicates), creates a
memory-storage script bucket, lands the derived script records there, wires
one source-router route plus script loop per entry with a per-entry script
policy, and fires each `boot` entry once through the normal request/reply
activation path with a per-run request id. Two read-only shell routes give
the frontend its reach: `GET /artifacts/<name>` serving artifact bodies with
their recorded media type under sandbox headers, and `GET /projections/<id>`
serving projection JSON. Ships a runnable example bundle under `examples/`.

### Slice 2: bundle-zip

Pure front-end to slice 1: `--bundle <file.zip>` extracts to a per-run
directory under the store dir, records the archive's content hash into load
provenance, then enters the identical directory path. No new authority
surface.

## Sequencing and Dependencies

Slice 1 has no dependency beyond the existing v1 binary assembly (wiring,
roles, source router, script loop, material store) and must not modify their
behavior for the non-bundle path. Slice 2 depends only on slice 1's loader
entry point.

## Handoff Contracts

- The loader consumes the existing seams as-is: `OpenKVScriptStore` over a
  pre-created memory bucket, `NewSourceRouter`/`RequestReply`,
  `NewScriptLoop`, `core.NewScriptRuntime` with per-entry policy, the durable
  ledger for dedupe, caller-lease authority for triggers.
- Material read surfaces needed by the shell routes are added to the
  embednats material store as read-only getters; no write surface is added.
- Role/router/service permissions grow only by the bundle's declared trigger
  subjects and the ephemeral bucket's API subjects, derived after manifest
  validation and before minting.

## Verification Strategy

Each slice is RED-first with committed Go tests over real embedded NATS.
Slice 1 families: allowed (load + boot + artifact/projection routes + caller
trigger round trip), denied collision (each disjointness dimension, typed),
malformed manifest (unknown field, missing fields, typed), ephemerality
(restart without the bundle leaves no bundle state; durable surface
unchanged), attributed failure (load failures name the bundle layer).
Frontend reach is additionally proven live in a browser against the example
bundle before the slice is called done. Denial oracles are output-parsed,
never exit-code.

## Escalation Log

- (none yet)
