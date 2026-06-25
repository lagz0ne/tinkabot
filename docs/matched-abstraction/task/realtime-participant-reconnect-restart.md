---
layer: task
topic: realtime-participant-reconnect-restart
references:
  - ../../../tasks/tinkabot-objective-okr.md
  - ./realtime-participant-watch.md
  - ./realtime-participant-action-gap.md
  - ./realtime-participant-sustained-rate.md
---

# Realtime Participant Reconnect Restart

## Scope

CKR-REALTIME needs a reconnect/restart proof before the scoped multiplayer
mission can claim that realtime sync survives a dropped watcher or binary
restart. This slice proves the generic participant action path plus Tinkalet
watch cursor catch-up over retained NATS KV history.

This is not a browser UI proof, not terminal scoring, not max participant
capacity, and not a custom replay channel.

## Contract

```text
alice scoped participant profile
  -> submit action rt-alice-0
  -> tinkalet watch prefix apps.demo.participants.alice.actions --cursor alice-reconnect
  -> cursor records latest observed action revision

tinkabot process stopped
  -> tinkabot process restarted on the same store
  -> alice participant profile re-imported from its persisted profile dir
  -> submit actions rt-alice-1..rt-alice-3
  -> same tinkalet cursor replays only the missed retained action revisions
```

The proof must fail on missed action ids, duplicate cursor replay,
non-increasing revisions, out-of-scope action keys, leaked substrate details, or
a requirement for a non-NATS replay path.

## RED Boundary

- Drive action submission through the Tinkalet participant profile path.
- Drive reconnect catch-up through `tinkalet watch --cursor`.
- Use existing `tb_items` KV history and participant-scoped watch filters.
- Do not re-admit Alice merely to hide restart behavior.
- Do not grant broad consumer or KV authority to the participant.
- Do not add browser, game, scoring, or custom-event behavior in this slice.

## GREEN Boundary

The slice is green only when `TestParticipantRealtimeReconnectRestartCatchUp`
proves:

- A scoped participant action watcher records a cursor.
- The Tinkabot process can stop and restart on the same store.
- The persisted participant profile can be re-imported after restart.
- The same Tinkalet cursor catches up retained missed action records in order.
- A subsequent watch with the same cursor does not replay duplicates.
- No raw bucket, `$KV`, credential, or broad authority detail leaks.

## GREEN Evidence

`TestParticipantRealtimeReconnectRestartCatchUp` proves:

- Alice submits `rt-alice-0` through an app-participant Tinkalet profile.
- `tinkalet watch prefix apps.demo.participants.alice.actions --cursor
  alice-reconnect` records a cursor at the seed action revision.
- Tinkabot stops and restarts on the same store.
- The persisted Alice profile is re-imported from its participant profile dir
  and points at the restarted NATS/shell endpoints.
- Alice submits `rt-alice-1..rt-alice-3` through the same scoped product path.
- The same cursor replays only the missed retained action records in strict
  revision order.
- A follow-up watch with the same cursor times out instead of replaying
  duplicates.
- Alice remains revokable after restart; revoked persisted creds fail direct
  reconnect and Tinkalet action submit reports `revoked-credentials`.

The substrate backing proof is `TestOperatorRevocationAfterRestart`: a valid
root-signed user JWT minted before runtime restart reconnects after restart, and
then can still be revoked through persisted account revocations. Malformed user
public keys remain rejected by the existing revocation failure-family test.

RED evidence:

```bash
go test ./tinkabot -run TestParticipantRealtimeReconnectRestartCatchUp -count=1
```

First RED failed at compile time because the reconnect acceptance helpers did
not exist. After helper implementation, runtime RED failed with
`action rt-alice-1 denied submit: connection-failed`, proving the restarted
participant descriptor still pointed at the old endpoint.

GREEN verification:

```bash
go test ./tinkabot -run TestParticipantRealtimeReconnectRestartCatchUp -count=1 -v
go test ./embednats -run TestOperatorRevocationAfterRestart -count=1 -v
go test ./tinkabot -run 'TestParticipantRealtimeReconnectRestartCatchUp|TestParticipantRealtimeActionGapHarness|TestParticipantRealtimeWatchEnvelope|TestParticipantAppActions|TestParticipantAppReducer|TestTurnBasedReferenceMission|TestParticipantAuthority' -count=1
go test ./embednats -run 'TestOperatorRevocationAfterRestart|TestOperatorRevocationDisconnectsLive|TestOperatorFailureFamiliesTyped' -count=1
go test ./tinkalet -run 'TestParticipantWatchScopeDenialPrecedesNetwork|TestParticipantWatchFiltersDenyMalformedTargets' -count=1
go test ./... -count=1
git diff --check
scripts/c3-line-coverage-harness.sh
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-302
```

C3 evidence: `c3-302` holds and includes this task doc; line coverage reports
`owned_files=484`, `lookup_errors=0`, `uncovered=0`; C3 check reports
`total: 28`, `ok: true`.

Independent review:

- Codex noninteractive: `VERDICT: PASS`, findings none. Codex reran focused
  restart/cursor and operator revocation tests plus `c3 eval c3-302` and
  `c3 check --include-adr`; it did not rerun the full Go suite.
- Claude noninteractive: `VERDICT: PASS`; it verified cursor replay,
  descriptor refresh without re-admit, revocation after restart, no broad
  participant authority or non-NATS replay path, and no docs/C3 overclaim.
  Residuals were limited to intentionally open terminal-result/UI proof and a
  harmless test cleanup that has been removed.

## Non-Goals

- No browser direct-NATS or generated UI credential claim.
- No terminal result, scoring, or race-order claim.
- No max-rate or max-participant freeze.
- No custom realtime engine or replay subject.
