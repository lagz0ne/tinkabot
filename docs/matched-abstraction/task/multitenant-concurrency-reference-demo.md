---
layer: task
topic: multitenant-concurrency-reference-demo
references:
  - ../../../tasks/tinkabot-multitenant-isolation-okr.md
  - ./multitenant-two-app-scope.md
  - ./realtime-participant-reference-demo.md
  - ./realtime-participant-reconnect-restart.md
---

# Multitenant Concurrency Reference Demo

## Scope

CKR-ISO-CONCURRENCY closes the remaining multitenant isolation gap with one
release-shaped proof. The proof builds the package archive, starts packaged
Tinkabot as the daemon authority, admits two app scopes with two participants
each, disables the packaged NATS CLI sidecar before user-level commands, and
drives all activity through packaged Tinkalet profiles.

This is not a SaaS control plane, not a max-capacity claim, not a browser UI
proof, and not a new security path for an example app.

## Contract

```text
release package
  -> packaged Tinkabot daemon starts demo:alice, demo:bob, other:alice, other:bob
  -> packaged Tinkalet imports owner and four participant profiles
  -> owner creates app-scoped state under apps.demo.* and apps.other.*
  -> four participant profiles submit concurrent scoped actions
  -> each profile watches only its own action subtree with a durable cursor
  -> cross-app, neighbor, raw-action-item, and bundle-trigger attempts deny
  -> Tinkabot restarts on the same store
  -> profiles re-import refreshed descriptors
  -> four participants submit more actions
  -> same Tinkalet cursors catch up retained post-restart actions
  -> duplicate cursor replay times out
  -> proof JSON records observed rate, action latency, watch latency, and leak counters
```

The proof must fail if any participant needs raw NATS, if any app scope can read
or act in the other app, if neighbor action watching succeeds, if restart loses
retained actions, if cursor replay duplicates already-consumed actions, or if
product output leaks raw bucket, `$KV`, credential, NATS URL, or key material
details.

## RED Boundary

- Public command: `bun run demo:iso-concurrency`.
- Expected RED before implementation: missing package script and missing proof
  artifact; the OKR still recorded the concurrent operations boundary as split
  across single-app demos and restart tests.
- Do not add a SaaS tenancy API.
- Do not require raw NATS subjects or KV APIs for the user-level proof.
- Do not claim a maximum user count, app count, action rate, or latency bound.
- Keep revocation in the focused regression/API proof because the packaged CLI
  has no owner-facing participant revoke command yet.

## GREEN Boundary

The slice is green only when:

- `TestMultitenantConcurrentRestartCatchUp` proves the combined invariant at
  the API/regression layer: four participants across two apps, concurrent
  Tinkalet action submission, own-cursor replay, product and direct raw-NATS
  cross-scope denials before and after restart, same-store restart, cursor
  catch-up, duplicate replay timeout, and targeted revocation with the other
  three participants still live.
- `bun run demo:iso-concurrency` writes `iso-concurrency-proof.json` with:
  - `apps_started = 2`.
  - `participants_started = 4`.
  - `expected_actions == observed_actions`.
  - `revision_gap_count = 0`.
  - `duplicate_replay_count = 0`.
  - `authority_denials = 48`.
  - `authority_violation_count = 0`.
  - `raw_authority_leak_count = 0`.
  - `restart_reconnect_pass = true`.
  - `capacity_claim = "observed-only"`.

## GREEN Evidence

- First demo RED exposed a harness bug: concurrent action revisions can be
  ordered by completion rather than action-id launch order. The proof now
  checks expected action-id sets plus strictly increasing observed revisions,
  matching the realtime reference demo's revision-gap definition.
- Codex follow-up review failed the first green pass because post-restart
  denials covered Tinkalet product commands but not direct raw-NATS bypass
  attempts. `assertISODenials` now includes direct participant-record reads, raw
  action-subject publishes, raw `$KV` action writes, and raw bundle-trigger
  publishes, and the same helper runs before and after restart.
- `go test ./tinkabot -run TestMultitenantConcurrentRestartCatchUp -count=1 -v`
  -> pass in `141.68s`.
- `bun run demo:iso-concurrency` -> Tailscale shell
  `http://forge.tail6c789a.ts.net:44975`, restart shell
  `http://forge.tail6c789a.ts.net:42891`, proof
  `/tmp/tinkabot-iso-concurrency.Nn2iiN/iso-concurrency-proof.json`.
- Latest proof metrics: `apps_started=2`, `participants_started=4`,
  `expected_actions=24`, `observed_actions=24`, `revision_gap_count=0`,
  `duplicate_replay_count=0`, `authority_denials=48`,
  `authority_violation_count=0`, `raw_authority_leak_count=0`,
  observed per-participant action rate `61.86Hz`, action latency p95 `36ms`,
  watch replay latency p95 `5046ms`, and
  `capacity_claim="observed-only"`.
- Final independent review: Codex follow-up `VERDICT: PASS`, findings none,
  after `bash -n`, `TestMultitenantConcurrentRestartCatchUp`,
  `TestMultitenantTwoAppScopeIsolation`, and `c3 eval c3-302`. Claude
  follow-up `VERDICT: PASS`, findings none, after `bash -n` and the combined
  focused Go regression pair.

## Verification

```bash
bash -n scripts/demo-iso-concurrency.sh
cd substrate/go && go test ./tinkabot -run TestMultitenantConcurrentRestartCatchUp -count=1 -v
bun run demo:iso-concurrency
```

C3 eval and lookup are ownership and coverage checks for this task surface; the
behavioral proof is the focused Go test plus the packaged demo above.

Revocation is intentionally proven by the focused Go regression because the
packaged public CLI path has no owner-facing participant revoke command yet.

## Non-Goals

- No SaaS dashboard or control-plane abstraction.
- No browser-originated multitenant UI proof in this slice.
- No max-rate, max-user, or max-app claim.
- No direct raw-NATS integration path for normal users, LLMs, or transforms.
