---
layer: task
topic: bundle-dir-app
status: complete
references:
  - ../approach/bundle-v1.md
  - ../plan/bundle-v1.md
---

# Bundle Dir App Task

## Brief

Bundle-v1 slice 1: the directory form of the bundle, end to end, per Plan
slice 1. `--bundle <dir>` loads `bundle.json`, validates disjoint authority,
lands records in a memory-storage script bucket, wires per-entry
router/loop/policy slots, fires boot entries through request/reply, and the
shell serves `GET /artifacts/<name>` (recorded media type, sandbox headers)
and `GET /projections/<id>` (JSON) read-only. Ships `examples/clock` as the
runnable proof app.

## Acceptance Contract

- `go test ./tinkabot -run TestBundle -count=1` passes: a Start with
  BundleDir loads the manifest, boot emits the frontend artifact and initial
  projection through the materializer, the artifact route serves the body
  with its media type and sandbox CSP, the projection route serves JSON, and
  a caller-creds trigger round trip updates the projection (allowed family);
  manifests colliding with the durable script key, trigger subject,
  projection id, or artifact prefix fail Start with a typed bundle error, as
  do intra-bundle duplicates (denied family); unknown manifest fields and
  missing required fields fail typed (malformed family); a restart of the
  same store without BundleDir serves the prior surface with no bundle bucket
  and no bundle records (ephemerality family); failures name the bundle layer
  (attributed family).
- The full standing battery stays green.
- The example bundle is proven live in a browser: page renders projection
  state served by the binary started with `--bundle examples/clock`.

## RED Artifact

Executed 2026-06-12: `cd substrate/go && go test ./tinkabot -run TestBundle
-count=1` -> compile failure: `cfg.BundleDir undefined (type Config has no
field or method BundleDir)`, `undefined: BundleRejected` — no bundle surface
exists.

## Verification Evidence

GREEN executed 2026-06-12.

`cd substrate/go && go test ./tinkabot -run TestBundle -count=1 -v` -> `ok`,
13/13 subtests — AppServes: Start with BundleDir over `examples/clock` boots
the entry through request/reply, the artifact route serves the emitted page
(`text/html`, `Content-Security-Policy` containing `sandbox`), the
projection route serves the stored record JSON, the bundle record is absent
from the durable script bucket (author-creds KV read), and a caller-creds
trigger (`accepted` reply) advances the projection's unix sequence.
EphemeralAcrossRestart: after a bundle-less restart of the same store the
bundle artifact is 404, the bundle trigger gets no reply, and the manual
trigger route still answers. ManifestCollision (6 typed BundleRejected
denials): durable script key, durable trigger subject, projection `main`,
artifact prefix `artifact/`, prefix overlap `art`, reserved subject
`tb.session.`, plus DuplicateInBundle. MalformedManifest (4 typed
BundleRejected denials): unknown field, missing trigger, wrong kind, missing
bundle.json. Denial oracles are typed-error asserts via `assertKind`, never
exit-code.

Full battery 2026-06-12: test, typecheck, build
release:evidence, gate:fakes, gate:parallel, gate:coverage, gate:scenarios,
gate:manual, `go test ./... -count=1` (9/9 packages), `git diff --check` —
all PASS.

Live browser proof 2026-06-12: binary started with `--bundle
examples/clock`; `GET /artifacts/bundle/clock/index.html` 200 `text/html`;
agent-browser rendered the page and its fetch of `/projections/bundle.clock`
displayed the boot projection (`renderedAt 08:14:38Z`); `nats request
--creds caller.creds -H Tinkabot-Request-Id:req-live-1 tb.bundle.clock.tick`
replied `accepted` and the page advanced itself to `renderedAt 08:15:14Z`
within its 2s poll — the one-stroke app loop, end to end.

## Addendum (2026-06-12, derive-by-construction)

After the first GREEN the user resolved the naming contract upward (Plan
Escalation Log): authority is derived, never declared. Manifest entries
shrank to `name`/`file`/`command`/`projections` (short ids)/`boot`; the
loader derives `scripts.bundle.<name>.<entry>`, `tb.bundle.<name>.<entry>`,
`bundle.<name>.<id>`, `bundle/<name>/`, and the perms growth collapsed to
the single wildcard `tb.bundle.<name>.>`. The entire collision-check family
(durable script key, trigger subject, projection `main`, artifact prefix
overlap, reserved subjects, durable-bucket probe) was deleted — a manifest
cannot spell those collisions; free-form naming fields are unknown-field
rejections.

Re-driven RED-GREEN: RED — `TestBundle/AppServes` failed against the old
contract (`/projections/bundle.clock.state` 404, old free-form manifest);
GREEN — `go test ./tinkabot -run TestBundle -count=1` -> `ok`, 13/13
(AppServes, EphemeralAcrossRestart, InvalidNames: BadEntryName /
DuplicateEntryName / DuplicateProjection, MalformedManifest: UnknownField /
FreeFormTrigger / MissingCommand / WrongKind / MissingManifest); full Go
suite 9/9 ok; live browser re-proof recorded below.

## Residual Risk

- NATS subject permissions cannot scope base64url-encoded artifact-manifest
  (`a.<base64url>`) and script-record KV keys, so artifact grants are
  enforced by the in-process script policy, not auth; acceptable under the
  operator-trusts-the-bundle posture, revisit if bundles ever load from
  untrusted sources (per-bundle buckets are the auth-true evolution).
- `/artifacts/` and `/projections/` serve all materialized state read-only
  on the loopback shell — observer-level reach, acceptable at the declared
  posture, revisit with any external exposure.
- The boot reply oracle is substring-based (`accepted`/`duplicate`) like the
  manual's own oracle; a structured status decode is a cheap later upgrade.

## Scope

Owns:

- `substrate/go/tinkabot/bundle.go` — manifest decode, disjoint validation,
  ephemeral bucket, record derivation, slot wiring, boot firing; typed
  `Bundle*` error kinds.
- `Config.BundleDir` + `--bundle` flag in `cmd/tinkabot`.
- Shell route additions for `/artifacts/` and `/projections/` plus read-only
  material getters in `substrate/go/embednats/script_materializer.go`.
- Permission growth for bundle trigger subjects and the ephemeral bucket.
- `substrate/go/tinkabot/bundle_test.go`, test fixtures, `examples/clock`.

Does not own:

- Zip handling (Plan slice 2), manual (`docs/manual/v1.md`) integration,
  release manifest growth, any change to the non-bundle startup path's
  behavior, session subsystem surfaces.
