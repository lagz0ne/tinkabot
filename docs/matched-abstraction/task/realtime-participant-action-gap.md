---
layer: task
topic: realtime-participant-action-gap
references:
  - ../../../tasks/tinkabot-objective-okr.md
  - ./realtime-participant-watch.md
  - ./app-action-revision-contract.md
---

# Realtime Participant Action Gap

## Scope

CKR-REALTIME needs an expected-vs-observed revision-gap harness before a
realtime-heavy reference app can claim sync capacity. This slice measures the
generic action ingress plus participant-scoped watch path under a bounded rate:
two app participants submit independent action records while their own filtered
KV watches observe every expected action revision.

This is not a typeracing API, not a browser UI proof, not a reconnect/restart
proof, and not a terminal scoring claim. It is the next measurement primitive
after participant watch authority.

## Contract

```text
participant alice scoped credential
  -> NATS request tb.app.demo.participants.alice.action
  -> Tinkabot action service validates shared state revision
  -> tb_items apps.demo.participants.alice.actions.<id>
  -> alice filtered KV watch observes each expected action id

participant bob scoped credential
  -> same flow for bob's own action subtree
  -> bob cannot depend on alice's watch authority
```

The harness counts expected action ids per participant and fails on any missing
or duplicate observed id. It also requires strict KV revision increase inside
each participant watch stream.

## GREEN Boundary

- Use scoped participant credentials and NATS request/reply.
- Use existing action subjects, action records, and filtered KV watches.
- Add no game-specific subject, scoring rule, or custom realtime channel.
- Keep the test duration short enough for the ordinary Go suite; longer 60s and
  breakpoint runs remain a later measurement using the same mechanism.

## GREEN Evidence

`TestParticipantRealtimeActionGapHarness` proves:

- Alice and Bob each submit a bounded sequence of unique action ids against
  app-scoped NATS-backed state.
- Each participant's own filtered watch observes every expected action id.
- Each participant's observed action revisions are strictly increasing.
- The measured path uses scoped participant credentials and the existing
  `tb_items` action material; no custom event channel is introduced.

Verification:

```bash
go test ./tinkabot -run TestParticipantRealtimeActionGapHarness -count=1
go test ./tinkabot -run 'TestParticipantRealtimeActionGapHarness|TestParticipantRealtimeWatchEnvelope|TestParticipantAppActions|TestParticipantAppReducer|TestTurnBasedReferenceMission|TestParticipantAuthority' -count=1
go test ./tinkalet -run 'TestParticipantWatchScopeDenialPrecedesNetwork|TestParticipantWatchFiltersDenyMalformedTargets' -count=1
go test ./... -count=1
git diff --check
scripts/c3-line-coverage-harness.sh
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-302
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh check --include-adr
```

C3 evidence: `c3-302` holds and includes this task doc; line coverage reports
`owned_files=482`, `lookup_errors=0`, `uncovered=0`; C3 check reports
`total: 28`, `ok: true`.

Independent review:

- Codex noninteractive: `VERDICT: PASS`, findings none.
- Claude noninteractive: `VERDICT: PASS`, no claim mismatches.

## Non-Goals

- No max participant or max rate freeze.
- No 60s capacity claim yet.
- No reconnect/restart proof.
- No terminal result or scoring proof.
- No browser direct-NATS or generated UI credential claim.
