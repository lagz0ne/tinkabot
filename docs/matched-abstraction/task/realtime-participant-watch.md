---
layer: task
topic: realtime-participant-watch
references:
  - ../../../tasks/tinkabot-objective-okr.md
  - ./app-action-revision-contract.md
  - ./realtime-browser-sync.md
---

# Realtime Participant Watch

## Scope

CKR-REALTIME needs revision accounting before a realtime-heavy game can be
admitted. This slice gives scoped app participants a safe watch path for the
NATS-backed material they are already allowed to read: shared app state for
their app and their own action/receipt subtree.

This is not a typeracing API, not a browser direct-NATS claim, and not a full
participant-rate target. It is the missing sync primitive needed before a
high-rate action harness can measure revision gaps and terminal results.

## Contract

```text
participant profile
  -> tinkalet watch prefix apps.<app>.state
  -> filtered JetStream KV consumer over tb_items app-state subjects only
  -> ordered item-event JSON with KV revisions

participant profile
  -> tinkalet watch prefix apps.<app>.participants.<id>.actions
  -> filtered JetStream KV consumer over own action and receipt subjects only
  -> ordered item-event JSON with KV revisions
```

Participants must not receive broad KV watch authority. Tinkalet must use
server-side watch filters for app-participant profiles, and Tinkabot must grant
only the matching filtered consumer-create subjects.

## RED Acceptance

```bash
cd substrate/go
go test ./tinkabot -run TestParticipantRealtimeWatchEnvelope -count=1
```

Expected RED before implementation: Alice can direct-read scoped app state, but
`tinkalet watch prefix apps.demo.state` fails because participant credentials do
not have a filtered watch surface. Alice must also be denied when watching Bob's
action subtree.

## GREEN Boundary

- Reuse `tb_items` and existing Tinkalet watch event/cursor format.
- Add no game-specific subject, verb, or scoring API.
- Grant only filtered watch authority for app state and own action subtree.
- Preserve owner/watch-only behavior for existing non-participant watch tests.
- Keep raw broad consumer creation denied for participant credentials.
- Deny neighbor and malformed app-participant watch targets before opening a
  NATS connection or creating a watch consumer.

## GREEN Evidence

Implemented on 2026-06-24 as a CKR-REALTIME revision-accounting prerequisite.

`TestParticipantRealtimeWatchEnvelope` proves:

- Alice's app-participant profile watches `apps.demo.state` and receives three
  ordered revisions for `apps.demo.state.rate`.
- Alice watches `apps.demo.participants.alice.actions` and receives her pending
  action plus deterministic receipt revisions.
- Alice cannot watch Bob's action subtree.
- Alice cannot publish a broad KV consumer-create request.
- Watch output does not leak credentials, raw `$KV` subjects, or store handles.
- `TestParticipantWatchScopeDenialPrecedesNetwork` proves Alice's neighbor
  watch target returns `denied-scope` even when the profile points at an
  unreachable NATS server, and that wildcard-looking item targets are rejected
  by command grammar before connection.
- `TestParticipantWatchFiltersDenyMalformedTargets` proves the participant
  filter itself denies malformed item and prefix targets before any network
  call path can be reached.

Implementation shape:

- `tinkalet watch` keeps existing broad `WatchAll` behavior for non-participant
  profiles, preserving local reaction/watch tests.
- `tinkalet watch` uses server-side `WatchFiltered` for app-participant
  profiles after validating the target is app state or the participant's own
  action subtree.
- `itemWatch` runs profile policy, participant target filtering, and local
  cursor loading before `itemKVForProfile` opens NATS.
- `participantAuth` grants only filtered consumer-create subjects for
  `apps.<app>.state.>` and
  `apps.<app>.participants.<participant>.actions.>`.

Verification:

```bash
go test ./tinkalet -run 'TestParticipantWatchScopeDenialPrecedesNetwork|TestParticipantWatchFiltersDenyMalformedTargets' -count=1
go test ./tinkabot -run TestParticipantRealtimeWatchEnvelope -count=1
go test ./tinkabot -run 'TestParticipantRealtimeWatchEnvelope|TestParticipantAppActions|TestParticipantAppReducer|TestTurnBasedReferenceMission|TestParticipantAuthority|TestTinkaletItemWatchUsesWatchStream|TestTinkaletDaemonWatchCursorRestartCatchesRetainedEvents' -count=1
go test ./tinkalet -count=1
go test ./cmd/tinkalet -count=1
go test ./... -count=1
git diff --check
scripts/c3-line-coverage-harness.sh
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-302
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh check --include-adr
```

C3 evidence: `c3-302` holds and includes this task doc; line coverage reports
`owned_files=481`, `lookup_errors=0`, `uncovered=0`; C3 check reports
`total: 28`, `ok: true`.

Independent review:

- Codex first returned `FAIL` for post-connect neighbor denial; the fix moved
  participant target filtering before `itemKVForProfile`.
- Codex then returned `FAIL` for malformed wildcard-looking item targets; the
  fix added grammar checks inside `participantWatchFilters` plus the malformed
  target regression.
- Final Codex noninteractive review returned `VERDICT: PASS`, findings none.
- Final Claude noninteractive review returned `VERDICT: PASS`, findings none.

## Non-Goals

- No max participant/rate freeze.
- No event-loss claim until the high-rate harness records expected vs observed
  revisions under load.
- No generated browser direct-NATS credential.
- No mission-family completion claim.
