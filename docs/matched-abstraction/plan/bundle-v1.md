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

Three slices, strictly sequenced; slice 1 delivers the user-testable app.

### Slice 1: bundle-dir-app

Owns the directory form end to end. `--bundle <dir>` on the binary loads a
strictly-decoded `bundle.json` manifest; each entry names one script source
file plus its short projection ids and an optional boot flag, and the loader
derives all authority under the bundle namespace per Approach invariant 2
(amended) — entries cannot spell collisions, so load validation is name
hygiene only. The loader creates a memory-storage script bucket, lands the
derived script records there, wires one source-router route plus script loop
per entry with a per-entry script policy, and fires each `boot` entry once
through the normal request/reply activation path with a per-run request id.
Two read-only shell routes give the frontend its reach: `GET
/artifacts/<name>` serving artifact bodies with their recorded media type
under sandbox headers, and `GET /projections/<id>` serving projection JSON.
Ships a runnable example bundle under `examples/`.

### Slice 2: bundle-account-isolation

(Added 2026-06-12, user decision — see Escalation Log.) The bundle plane
moves into its own runtime-minted NATS account: scripts, materials,
artifacts, and ledger live in the bundle account's JetStream plane, bundle
principals are minted there, and the only crossing into the app account is a
service export/import per trigger under the importer's local name. New
embednats primitives: `MintAccount`, `ExportService`, `ImportService`. The
user-visible surface from slice 1 does not move.

### Slice 3: bundle-zip

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

- 2026-06-12: after slice 1's first GREEN, the user resolved the bundle's
  naming contract upward: authority is derived by construction instead of
  declared-and-checked ("proxy the call... always append prefix — but I
  think that's the auth setup as well"; trust posture: the operator loading
  a bundle knows what is inside). Approach invariant 2 amended; slice 1
  re-driven RED-GREEN under the derived contract, deleting the collision
  validation family. Recorded constraint for later slices: NATS subject
  permissions cannot scope base64url-encoded artifact-manifest and
  script-record KV keys, so artifact-grant enforcement stays with the
  in-process script policy unless per-bundle buckets are introduced.
- 2026-06-12 (second escalation, same day): the user resolved the isolation
  mechanism upward again — "we play within the boundary of nats auth...
  same subject doesn't mean a lot as long as the imports and exports are
  correctly set". Slice 2 `bundle-account-isolation` inserted (zip slides to
  slice 3): the account boundary supersedes both the prefix-law and the
  base64url constraint above (bundle buckets are unreachable from the app
  account regardless of key encoding). Derived names survive as the
  import-remap convention on the app-facing surface.
