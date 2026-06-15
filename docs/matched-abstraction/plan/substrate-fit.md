---
layer: plan
topic: substrate-fit
references:
  - ../approach/substrate-fit.md
  - ../approach/bundle-v1.md
---

# Substrate Fit Plan

## Consumed Approach

`../approach/substrate-fit.md` is the sole thinking authority: right
primitive per job, scripts stay NATS-blind, the materializer stays the only
gate, isolation is fail-closed. User decisions (2026-06-15): deps install at
load before the jail is sealed; sandbox bundles only; all three slices to be
completed (order free).

## Decomposition

### Slice A: file-referenced artifacts

The artifact effect may carry a filesystem path (under the script's declared
output dir) instead of an inline body. The materializer reads that file and
streams it into Object Store, validating name/prefix/revision and recording
the manifest exactly as today — bytes never enter a stdout frame, so the
256 KiB frame ceiling no longer bounds artifact size. Inline-body artifacts
keep working (small assets). The output dir is a known, materializer-readable
location per run.

### Slice B: sandboxed bundle processes (bwrap)

Bundle script and filter processes spawn inside bubblewrap: read-only bind of
the bundle dir + toolchain, private tmpfs, the single writable output mount
that feeds Slice A, no network. Fail-closed: if the host cannot sandbox, the
bundle does not start. Bundle load runs `bun install` (or equivalent) for a
bundle that declares deps *before* sealing the jail, so the offline runtime
has its vendored deps. Non-bundle/wired-slot scripts are unchanged.

### Slice D: reference-resolution

Emit side (done): `ScriptPolicy.ProjectionPrefix`; the gate prefixes short
projection ids and relative artifact names to the derived form
(`bundle.<name>.<id>`, `bundle/<name>/<path>`) before the policy check and
materialization, backward-compatible with already-full emits and a no-op for
the wired slot (empty prefix). Serve side (pending a manifest-config
decision): the frontend uses relative fetch paths and Vite a relative base
(`./`); the server resolves them within the bundle's scope per a
manifest-declared mapping, so the page hardcodes no derived name either.

### Slice C: content-addressed serving

The artifact HTTP route serves `ETag: "<digest>"` (the sha256 the Object
Store already computes) and answers `If-None-Match` with `304`; `no-store` is
dropped. Stable artifact names plus cheap revalidation.

## Sequencing and Dependencies

Slice A and Slice C are independent of each other; Slice B's writable output
mount is where Slice A's producer writes, so B assumes A's output-dir
contract (build A first, or define the contract in A and wire the mount in
B). C depends only on the existing digest. All three land before the goal is
met; order within that is free.

## Handoff Contracts

- Slice A owns the artifact effect shape (`path` vs `body`), the output-dir
  convention, and the materializer's file→Object Store stream; it must not
  change the projection path or the gate semantics.
- Slice B owns only the process spawn (`LocalScriptRunner`/`FilterLoop` →
  bwrap wrapper) and the load-time install step; it consumes A's output-dir
  contract and bundle-v1's account/perms unchanged.
- Slice C owns only the artifact HTTP handler headers; it consumes the
  digest already on the stored manifest.

## Verification Strategy

Each slice is RED-first over real seams: A — a script emitting an
over-256 KiB artifact by path materializes and serves (inline-frame path
would fail); B — a bundle process is provably jailed (no network, no FS
beyond the binds) and the run is fail-closed when sandboxing is unavailable;
C — a conditional GET returns 304 on an unchanged digest, 200 on change.
bundle-v1's examples (clock, builder) keep passing; the builder serves a
chunk that would have exceeded the old ceiling as the headline proof.

## Escalation Log

- 2026-06-15: opened from the Vite-filter discussion. The bundle-v1
  frame-ceiling "Non-Goal" is converted here to Slice A (route bytes through
  Object Store via the filesystem, not a bigger frame). v1's deferred
  "docker-sandboxing" is retired in favor of bwrap (Slice B). Both recorded
  back in bundle-v1 when these land.
