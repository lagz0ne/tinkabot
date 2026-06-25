---
layer: task
topic: visual-decision-loop
references:
  - ../../../tasks/tinkabot-objective-okr.md
  - ./scoped-multiplayer-mission-assembly.md
  - ./browser-participant-action-bridge.md
  - ./tinkalet-watch-cursors.md
---

# Visual Decision Loop

## Scope

CKR-VIS proves the LLM visualization mission as one release-shaped loop:
rendered visual bundle, sandboxed user selection, durable NATS-backed result,
scoped LLM watcher readback, and transform-chain update. Dynamic arbitrary LLM
bundle publication stays out of this slice; the proof uses included bundles and
generic substrate mechanisms.

## Contract

```text
packaged Tinkabot + visual bundle
  -> Tailscale shell serves rendered artifact
  -> generated UI receives only a trusted-shell lease
  -> generated UI posts item_submit for artifacts.<artifact>.results.*
  -> browser command service creates or guarded-updates tb_items material
  -> scoped LLM watcher profile observes only that result through Tinkalet watch
  -> transform chain updates a projection/artifact without UI receiving NATS
```

The proof must fail if rendering alone is counted, if generated UI receives raw
NATS authority, if the result is not in `tb_items`, if duplicate/stale/malformed
submits do not deny, if the watcher can broaden scope, or if owner credentials
are used as the LLM watch path.

## RED Boundary

- `item_submit` on `tb.app.browser.command` is currently denied before
  materializing an item.
- Tinkabot has no product `item-watcher` profile descriptor that Tinkalet can
  import and scope-check.
- No release-shaped `demo:visual` ties V1-V5 together.

## GREEN Boundary

- V1: a release-shaped bundle artifact renders through the Tailscale shell.
- V2: sandboxed generated UI submits `item_submit` through the trusted shell;
  malformed, duplicate, stale, out-of-scope, and raw-authority payloads deny.
- V3: accepted choice is durable `tb_items` material with value, revision,
  status, provenance, and restart readback.
- V4: scoped LLM watcher sees the result via `tinkalet watch`; broad watch,
  direct get, neighbor/off-scope target, and revoked watcher paths deny.
- V5: source material drives a bundle transform to updated projection/artifact.

## RED Evidence

- Expected initial RED:
  `go test ./tinkabot -run 'TestBrowserItemSubmitBridge|TestTinkaletScopedWatcherProfile' -count=1`
  fails because `item_submit` and scoped watcher profiles are not implemented.
- Actual RED:
  `go test ./tinkabot -run 'TestBrowserItemSubmitBridge|TestTinkaletScopedWatcherProfile' -count=1`
  failed at compile time on missing `App.AdmitWatcher` and
  `App.RevokeWatcher`; `go test ./cmd/tinkabot -run
  TestRunAdmitsWatchersFromStartupFlag -count=1` failed with
  `flag provided but not defined: -watcher`.

## GREEN Evidence

- `TestBrowserItemSubmitBridge` proves `item_submit` creates
  `artifacts.<artifact>.results.*` items through the browser command service,
  denies duplicate create, accepts guarded update, denies stale update, denies
  out-of-artifact scope, and denies raw authority payloads.
- `TestTinkaletScopedWatcherProfile` proves an `item-watcher` profile imports
  into Tinkalet, cannot direct `item get`, cannot broaden to a prefix watch,
  can watch its exact result item, and is denied after revocation.
- `TestRunAdmitsWatchersFromStartupFlag` proves packaged startup can admit a
  watcher profile using `--watcher <name>:<item|prefix>:<target>`.
- `bash scripts/demo-visual-decision.sh /tmp/tinkabot-visual-green3` passed as
  a release-shaped proof over Tailscale URL
  `http://forge.tail6c789a.ts.net:35995`. Proof file:
  `/tmp/tinkabot-visual-green3/visual-decision-proof.json`.
- Proof results: `acceptedIntents=1`, `deniedIntents=0`,
  `acceptedSubmits=1`, `deniedDispatches=0`, `authorityLeakCount=0`,
  `submitLatencyMs=73`, `artifactRendered=true`,
  `watcherIsolated=true`, `watcherHasOwnerProfile=false`,
  `transformChanged=true`, and `restartDurable=true`.
- The rendered artifact proof opened
  `http://forge.tail6c789a.ts.net:35995/artifacts/bundle/clock/index.html`
  in Playwright and saw title `tinkabot sequence`, nonblank text, a
  Mermaid sequence, and the live projection panel.
- The LLM watcher path used a separate Tinkalet config/data/home from the
  owner path; its profile list contained only default profile `llm` with
  role `watcher`, trust `item-watcher`, scope `item`, and target
  `artifacts.artifact-browser.results.choice`.
- The watcher event saw
  `artifacts.artifact-browser.results.choice` with value
  `{"choice":"diagram-a"}`; owner readback and post-restart owner readback saw
  the same durable item with browser-command provenance.
- Verification also passed: focused Go tests for the visual bridge and watcher
  startup, `go test ./... -count=1`, frontend isolation/observe tests, native
  frontend typecheck, frontend build, script syntax checks, and `git diff
  --check`.
- C3 closure passed: `scripts/c3-line-coverage-harness.sh` reported
  `owned_files=494`, `lookup_errors=0`, `uncovered=0`; `c3x eval c3-302`,
  `c3x eval c3-501`, `c3x eval c3-502`, and `c3x check --include-adr` all
  passed.
- Independent review passed: Codex final review reported `VERDICT: PASS` with
  no findings and explicitly closed the watcher-isolation and artifact-render
  blockers; Claude final review reported `VERDICT: PASS` with only LOW
  residuals.

## Non-Goals

- No Mermaid-specific platform API.
- No direct browser NATS credential.
- No owner profile as the LLM watcher happy path.
- No dynamic arbitrary LLM bundle publication claim.
- The watcher demo observes the submitted item from retained KV replay; a
  live-before-submit watcher is supported by the same watch path but is not the
  extra claim this proof counts.
- The `token` raw-authority leak term is intentionally conservative and may
  false-positive on benign content; the proof only counts the decision UI's
  command/DOM surface.
