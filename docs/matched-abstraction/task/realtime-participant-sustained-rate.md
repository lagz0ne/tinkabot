---
layer: task
topic: realtime-participant-sustained-rate
references:
  - ../../../tasks/tinkabot-objective-okr.md
  - ./realtime-participant-watch.md
  - ./realtime-participant-action-gap.md
---

# Realtime Participant Sustained Rate

## Scope

CKR-REALTIME needs a longer participant-rate proof before a realtime-heavy
reference app can claim a sustained sync envelope. This slice extends the
bounded action-gap harness into a 60 second run: two scoped participants submit
through the product action subject while each participant's filtered KV watch
must observe every expected action revision.

This is still not a max-rate freeze, reconnect/restart proof, browser UI proof,
or terminal scoring claim. It only admits the current sustained baseline.

## Contract

```text
alice scoped participant credential
  -> 600 action submits at 100ms cadence
  -> existing Tinkabot app-action service
  -> alice own action KV subtree
  -> alice filtered watch observes 600/600 unique action ids

bob scoped participant credential
  -> same path in parallel
  -> bob filtered watch observes 600/600 unique action ids
```

The harness must fail on any missing expected action id, duplicate action id,
out-of-scope observed key, watcher error, or non-increasing per-participant KV
revision.

## GREEN Boundary

- Use scoped participant credentials and NATS request/reply.
- Use existing action subjects, action records, and filtered KV watches.
- Add no game-specific subject, scoring rule, custom realtime channel, or
  browser credential claim.
- Keep the long proof opt-in so the ordinary suite remains fast, but run the
  opt-in proof before claiming this slice green.

## GREEN Evidence

`TestParticipantRealtimeSustainedActionGapHarness` proves:

- Alice and Bob each submit 600 unique action ids at a 100ms cadence.
- The run covers 1200 scoped action submissions over about 60 seconds.
- Each participant's own filtered watch observes every expected action id.
- Duplicate ids, out-of-scope keys, watcher errors, missing ids, and
  non-increasing per-participant KV revisions fail the harness.
- The watcher cleanup does not require broad consumer-delete authority for
  participant credentials.

Verification:

```bash
go test ./tinkabot -run TestParticipantRealtimeActionGapHarness -count=1 -v
TINKABOT_REALTIME_SUSTAINED=1 go test ./tinkabot -run TestParticipantRealtimeSustainedActionGapHarness -count=1 -timeout 2m -v
go test ./tinkabot -run 'TestParticipantRealtimeSustainedActionGapHarness|TestParticipantRealtimeActionGapHarness|TestParticipantRealtimeWatchEnvelope|TestParticipantAppActions|TestParticipantAppReducer|TestTurnBasedReferenceMission|TestParticipantAuthority' -count=1
go test ./... -count=1
git diff --check
scripts/c3-line-coverage-harness.sh
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-302
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh check --include-adr
```

Latest sustained proof: `1200` actions across `2` participants in
`59.906036488s`, with every expected action id observed by the scoped filtered
watches.

C3 evidence: `c3-302` holds and includes this task doc; line coverage reports
`owned_files=483`, `lookup_errors=0`, `uncovered=0`; C3 check reports
`total: 28`, `ok: true`.

Independent review:

- Codex noninteractive: `VERDICT: PASS`, findings none. Codex independently
  reran the short action-gap proof, the 60s sustained proof, the focused suite,
  the full Go suite, `git diff --check`, the C3 line harness, `c3-302` eval,
  and `c3 check --include-adr`.
- Claude noninteractive first pass: `VERDICT: FAIL` only because this sustained
  proof had not yet recorded independent-review evidence in the task doc and
  OKR/handoff rows. Claude accepted the runtime, authority, anti-overclaim, and
  C3 binding claims.
- Claude noninteractive follow-up: `VERDICT: PASS`; the review-evidence gap is
  closed and no new overclaim was introduced.

## Non-Goals

- No max participant or max rate freeze.
- No reconnect/restart proof.
- No terminal result or scoring proof.
- No browser direct-NATS or generated UI credential claim.
