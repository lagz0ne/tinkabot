---
layer: task
topic: bundle-schedule
status: complete
references:
  - ../approach/bundle-v1.md
  - ../plan/bundle-v1.md
---

# Bundle Schedule Task

## Brief

Bundle-v1 slice 3, per the Plan's amended decomposition: a bundle entry can
drive its own updates. The manifest declares the cadence as intent
(`"every": "5s"`, floored at 100ms), and the binary fires that entry's
trigger on a ticker through the same caller request/reply path as boot —
every tick an ordinary attributed, per-run-deduped activation; the script
stays one-shot and the materializer stays the only truth gate. Runtime
control is NATS settings: the app config bucket key
`bundle.<bundle>.<entry>.every`, writable with plain caller authority — a
duration retunes live, `off` pauses, deleting the key falls back to the
manifest. This is bundle-plane automation over the proven request/reply
path, not the deferred HA `schedule` trigger source (no leader epoch, no
fencing token); that deferral stands.

## Acceptance Contract

- `go test ./tinkabot -run TestBundle -count=1` passes with the new
  families: a scheduled fixture's projection advances with no manual trigger
  anywhere (ScheduledTicks); `off` written to the settings key with caller
  creds stops advancement, and a new duration resumes it (same subtest);
  malformed and sub-floor cadences are typed BundleRejected
  (MalformedManifest/MalformedEvery, OverEagerEvery).
- The full standing battery stays green; `examples/clock` (now
  `"every": "5s"`) proven self-ticking live in a browser, paused and retuned
  through `nats kv put`.

## RED Artifact

Executed 2026-06-12: `go test ./tinkabot -run TestBundle -count=1` ->
`TestBundle/ScheduledTicks` failed: `BundleRejected: bundle manifest could
not be decoded: json: unknown field "every"` — the schedule contract did not
exist.

## Verification Evidence

GREEN executed 2026-06-12.

`go test ./tinkabot -run TestBundle -count=1` -> `ok` — ScheduledTicks: a
300ms fixture advanced its projection with no manual trigger; caller-creds
`Put("bundle.t.run.every", "off")` on `config_bucket` froze it (two reads
1.2s apart identical); `Put(..., "200ms")` resumed advancement. Malformed
`"soon"` and over-eager `"5ms"` cadences fail typed. Full Go suite 9/9 ok.

Live browser 2026-06-12: `--bundle examples/clock` with `"every": "5s"` —
page advanced by itself (`renderedAt 09:22:24Z -> 09:22:29Z`, no triggers);
`nats kv put config_bucket bundle.clock.tick.every off` froze it at
`09:22:49Z` across two 7s-apart reads; `... 1s` resumed it (`09:23:05Z`).

## Scope

Owns:

- `substrate/go/tinkabot/bundle.go` — `every` manifest field + validation
  (`minEvery` floor), `scheduleBundle`/`tickBundle` (per-entry ticker, stop
  channels through the existing teardown path, settings consult per cycle,
  per-tick request ids `tick-<nonce>-<entry>-<n>`), boot refactored to share
  the caller connection.
- ScheduledTicks/MalformedEvery/OverEagerEvery tests; `examples/clock`
  cadence + README control section.

Does not own:

- The platform `schedule` trigger source (stays deferred: HA leader epoch,
  fencing, durable cursors); zip (Plan slice 4); per-tick backoff tuning.

## Residual Risk

- Tick outcomes are fire-and-observe: a persistently failing tick (e.g. a
  script whose sequence stops growing) records failed activations in the
  ledger but nothing surfaces it to the operator beyond observer reads.
- The settings key is read once per cycle, so a retune takes effect after
  the current sleep completes — bounded by the previous cadence, worst case
  the manifest default while paused.
