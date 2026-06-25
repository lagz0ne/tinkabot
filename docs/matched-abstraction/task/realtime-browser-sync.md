---
layer: task
topic: realtime-browser-sync
references:
  - ../../../tasks/tinkabot-objective-okr.md
  - ./turn-based-reference.md
---

# Realtime Browser Sync

## Scope

CKR-REALTIME needs measurement before a realtime-heavy game can be admitted.
This slice measures the existing NATS-native clock chain in a real browser. It
does not count the scoped multiplayer mission complete, does not add a
typeracing API, and does not claim participant throughput.

## Contract

```text
bundle.clock.tick every 100ms
  -> NATS request/reply trigger
  -> bundle.clock.state projection
  -> KV watch feeds present filter
  -> bundle.clock.view projection
  -> trusted shell serves generated clock UI
  -> browser DOM reports age/fetch stats from _p/view
```

The browser proof samples the generated UI, not only curl against the projection
route. Latest-state polling can coalesce updates, so this proof is only the DOM
sync-age branch of CKR-REALTIME. Participant-rate and watch revision-gap proof
remain separate.

## RED Acceptance

```bash
dist=$(mktemp -d /tmp/tinkabot-rt-red.XXXXXX)
TINKABOT_DEMO_BROWSER_SYNC=1 \
TINKABOT_DEMO_FAST_EVERY=100ms \
  bash scripts/demo-chain-reaction.sh "$dist"
test -f "$dist/realtime-browser-sync.json"
```

RED before implementation: the demo ignores `TINKABOT_DEMO_BROWSER_SYNC`, exits
without a browser proof artifact, and the final file check fails.

## GREEN Boundary

- Reuse the existing clock bundle and `demo:chain` package flow.
- Keep the public user-facing URL Tailscale/MagicDNS when available.
- Use Playwright only as the measurement harness.
- Fail the demo if browser p95/p99, filter p95, or source interval p95 exceed
  the provisional CKR-REALTIME target.
- Do not introduce a custom realtime channel or game-specific mechanism.

## GREEN Evidence

Implemented on 2026-06-24 as a CKR-REALTIME measurement prerequisite.

Focused verification:

```bash
TINKABOT_DEMO_BROWSER_SYNC=1 TINKABOT_DEMO_FAST_EVERY=100ms bash scripts/demo-chain-reaction.sh /tmp/tinkabot-rt-green
bash -n scripts/demo-chain-reaction.sh scripts/demo-live-patch.sh scripts/demo-turn-based.sh scripts/release-package.sh scripts/package-tinkabot.sh
git diff --check
scripts/c3-line-coverage-harness.sh
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh check --include-adr
```

Latest GREEN run:

```text
Tailscale URL: http://forge.tail6c789a.ts.net:40327/artifacts/bundle/clock/index.html
Proof: /tmp/tinkabot-rt-green.32iCRS/realtime-browser-sync.json
Samples: 112 over 6000ms at 50ms sampling
Unique seqs: 58
Browser age: p50 75ms, p95 149ms, p99 173ms, max 176ms
Fetch: p95 9ms, max 12ms
Filter latency: p95 19ms, max 21ms
Source interval: p95 140ms, max 143ms
Thresholds: browser p95 <= 250ms, browser p99 <= 500ms,
filter p95 <= 50ms, source interval p95 <= 150ms
Pass: true
```

The proof records `largeSeqGapsOver175Ms` for later analysis, but it does not
fail on sequence coalescing. This task proves browser-visible freshness of the
latest projection only; revision-gap and terminal-loss proof remain separate.

Independent review:

- Codex noninteractive `VERDICT: PASS`, findings none.
- Claude noninteractive `VERDICT: PASS`, findings none. Claude listed only
  non-blocking residuals: warmup uses the p99 age gate before final p95
  enforcement, percentile selection is conservative, and the informational
  sequence-gap constant is not documented as a threshold.

## Non-Goals

- No participant-rate target.
- No typeracing-specific API.
- No event-loss claim from latest-projection polling alone.
- No mission-family completion claim.
