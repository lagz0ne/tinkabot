---
layer: task
topic: bundle-account-isolation
status: complete
references:
  - ../approach/bundle-v1.md
  - ../plan/bundle-v1.md
---

# Bundle Account Isolation Task

## Brief

Bundle-v1 slice 2, per the Plan's amended decomposition: the bundle plane
moves into its own runtime-minted NATS account. The account — not naming —
is the isolation boundary: scripts, materials, artifacts, and ledger live in
the bundle account's JetStream plane (memory storage; the account lifecycle
is what makes the plane ephemeral), bundle principals are minted in that
account, and the only crossing into `TB_APP` is a service export/import per
trigger subject, under the importer's local name. The user-visible surface
(manifest, `examples/clock`, shell routes, caller trigger) does not move.

New embednats primitives: `MintAccount(name)` (runtime account mint into the
live resolver), `ExportService(account, subject)`, and
`ImportService(account, fromAccount, subject, localSubject)` — each
compiling into account claims and riding the existing live `pushAccount`.

## Acceptance Contract

- `go test ./embednats -run TestBundleAccountSeam -count=1` passes: a
  request on the same subject does not cross the account boundary without an
  import (SameSubjectNoCrossing); an exported service imported into `TB_APP`
  answers round trip (ImportedServiceRoundTrip); the same bucket name in the
  minted account is invisible from `TB_APP` (JetStreamIsolated); duplicate
  account mints and exports from unknown accounts fail typed.
- `go test ./tinkabot -run TestBundle -count=1` passes with the AppServes
  isolation oracles: `tb_bundle` does not exist in the app account and the
  bundle projection never lands in the app material bucket, while the shell
  routes still serve both artifact and projection from the bundle store.
- The full standing battery stays green; `examples/clock` proven live in a
  browser across the boundary.

## RED Artifact

Executed 2026-06-12, two hops:

- embednats: `go test ./embednats -run TestBundleAccountSeam -count=1` ->
  compile failure: `rt.MintAccount undefined`, `rt.ExportService undefined`,
  `rt.ImportService undefined` — the primitives did not exist.
- tinkabot: `go test ./tinkabot -run TestBundle/AppServes -count=1` ->
  `bundle bucket visible in the app account: <nil>` — the single-account
  wiring failed the isolation oracle.

## Verification Evidence

GREEN executed 2026-06-12.

`go test ./embednats -run TestBundleAccountSeam -count=1` -> `ok 0.08s` —
no-crossing, imported round trip (poll until the claims push routes),
JetStream isolation (`shadow` bucket in the bundle account,
`ErrBucketNotFound` from `TB_APP`), typed duplicate-mint and
unknown-account denials.

`go test ./tinkabot -run TestBundle -count=1` -> `ok`, 13/13 — AppServes now
additionally proves: a `$JS.API.>` probe in `TB_APP` gets
`ErrBucketNotFound` for `tb_bundle` and `ErrKeyNotFound` for
`p.bundle.clock.state` in the app `tb_material`, while `/artifacts/` and
`/projections/` serve the same content from the bundle account's store, and
the caller trigger round trip works through the export/import.

`go test ./... -count=1` -> all 9 packages ok.

Live browser 2026-06-12: binary with `--bundle examples/clock`; page
rendered the boot projection (`renderedAt 08:45:37Z`); `nats request ...
tb.bundle.clock.tick` with app-account caller creds crossed the import and
the page advanced itself to `renderedAt 08:45:52Z`.

## Scope

Owns:

- `substrate/go/embednats/operator.go` — `MintAccount`, `ExportService`,
  `ImportService`; `substrate/go/embednats/bundle_account_test.go`.
- `substrate/go/tinkabot/bundle.go` — `startBundle` account rework
  (`bundle.account()`, per-bundle service/router principals,
  `bundleServicePerms`/`bundleRouterPerms`, memory buckets in the bundle
  account reusing the app plane's bucket names, export/import per trigger,
  boot retry window), material-store dispatch in the shell routes.
- `substrate/go/tinkabot/tinkabot.go` — `bundleMaterials` field; reverted
  router/service perms growth (the bundle plane no longer touches app
  principals beyond the caller's publish wildcard).

Does not own:

- Zip handling (now Plan slice 3), private exports/activation tokens,
  multi-bundle runs, manual integration, per-bundle revocation at teardown
  (process exit bounds the account).

## Residual Risk

- Service exports are public within the operator (any account could import
  them); fine single-operator at loopback, revisit with activation tokens if
  account tenancy ever diversifies.
- The boot retry window (10s) covers claims-push propagation; a pathological
  propagation stall fails Start typed rather than hanging.
- Orphaned JetStream file state from prior runs' app accounts still
  accumulates in the store dir (pre-existing posture); bundle accounts add
  none (memory storage).
