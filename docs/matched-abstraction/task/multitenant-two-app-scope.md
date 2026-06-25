---
layer: task
topic: multitenant-two-app-scope
references:
  - ../../../tasks/tinkabot-multitenant-isolation-okr.md
  - ./participant-authority.md
  - ./app-action-revision-contract.md
  - ./realtime-participant-watch.md
---

# Multitenant Two-App Scope

## Scope

CKR-ISO-AUTH proves the first multitenant authority slice: one Tinkabot daemon
can host two app scopes with separate participant profiles and NATS-backed
material, while Tinkalet users can only act inside their own app/participant
scope.

This is not a SaaS control plane, not multi-bundle routing, not owner isolation,
and not max-capacity proof. It uses the existing generic app-action, item, watch,
participant profile, and NATS permission surfaces.

## Contract

```text
one Tinkabot daemon
  -> admits demo:alice, demo:bob, other:alice, other:bob
  -> owner creates apps.demo.state.* and apps.other.state.*
  -> each participant imports only its local Tinkalet profile
  -> own-app item read, action submit, and filtered watch pass
  -> cross-app item read, action submit, watch, trigger, raw action subject, raw KV, and raw bundle trigger fail
  -> revoking demo:alice leaves demo:bob, other:alice, and other:bob live
```

The proof must fail if participants share a broad credential, if an app scope
can read or mutate another app's material, if Tinkalet silently falls back to
owner authority, if direct NATS/KV subjects work for restricted participants, or
if revocation kills more than the targeted participant.

## RED Boundary

Before this slice, no named acceptance proof showed two positive app scopes
active concurrently in one daemon. Existing proofs covered same-app Alice/Bob
isolation and wrong-app denials, but not two app scopes both succeeding while
also denying cross-app paths.

## GREEN Boundary

The slice is green only when `TestMultitenantTwoAppScopeIsolation` proves:

- one daemon admits `demo:alice`, `demo:bob`, `other:alice`, and `other:bob`;
- each participant profile imports through a separate Tinkalet environment;
- `apps.demo.state.*` and `apps.other.state.*` are both readable by their own
  app participants;
- own-app action submit materializes the expected app/participant action item;
- own-app filtered watches return only the scoped app state;
- cross-app item reads, watches, action submits, Tinkalet bundle triggers,
  raw action-subject publishes, raw KV publishes, raw bundle-trigger publishes,
  and direct participant-record reads are denied;
- revoking `demo:alice` denies her reconnect/action path while `demo:bob`,
  `other:alice`, and `other:bob` remain live.

## Evidence

RED:

```text
go test ./tinkabot -run TestMultitenantTwoAppScopeIsolation -count=1 -v
```

First executable RED failed because participant trigger denial returned
`connection-failed` instead of the product-scope denial:

```text
profile demo-alice denied bundle.clock.tick: connection-failed
```

This exposed a real boundary gap. Tinkalet now rejects restricted profiles
before opening the trigger request path: revoked restricted profiles report
`revoked-credentials`, and active `app-participant` / `item-watcher` profiles
report `denied-scope`.

GREEN:

- `TestMultitenantTwoAppScopeIsolation` admits `demo:alice`, `demo:bob`,
  `other:alice`, and `other:bob` in one daemon.
- Owner materializes `apps.demo.state.board` and `apps.other.state.board`.
- Demo participants can read/watch/act only under `apps.demo.*`; other
  participants can read/watch/act only under `apps.other.*`.
- Cross-app item reads, watch prefixes, action submissions, direct
  participant-record reads, raw action-subject publishes, raw `$KV` publishes,
  raw `tb.bundle.clock.tick` publishes, and bundle triggers from restricted
  participant profiles are denied for all four participants.
- Revoking `demo:alice` denies her action path while `demo:bob`, `other:alice`,
  and `other:bob` remain active.

Verification:

```bash
cd substrate/go && go test ./tinkabot -run TestMultitenantTwoAppScopeIsolation -count=1 -v
cd substrate/go && go test ./tinkalet -run 'TestProfileUseAndTriggerDenials|TestParticipantWatchScopeDenialPrecedesNetwork|TestParticipantWatchFiltersDenyMalformedTargets|TestActionCommandDenials|TestWatchCommandDenials' -count=1
```

Current result: both commands pass on 2026-06-25. The final focused package
run also passed: `cd substrate/go && go test ./tinkabot ./tinkalet -count=1`.

Review evidence:

- Initial Codex review returned `VERDICT: FAIL` because `other:bob` and raw
  bundle-trigger denial coverage were asymmetric. The test was strengthened
  instead of weakening the claim.
- Final Codex and Claude noninteractive follow-up reviews both returned
  `VERDICT: PASS`, confirming all four profiles have own-scope read/action/watch,
  cross-app item/watch/action/direct-record/raw-action/raw-KV denial, Tinkalet
  `bundle.clock.tick` denial, and raw `tb.bundle.clock.tick` denial.

## Non-Goals

- No SaaS tenant registry or billing/control-plane route.
- No multi-bundle-in-one-daemon support.
- No owner/caller isolation claim.
- No browser UI or Tailscale release-shaped demo claim.
- No max users, max apps, or max throughput claim.
