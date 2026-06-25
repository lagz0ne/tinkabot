---
layer: task
topic: app-action-revision-contract
references:
  - ../approach/tinkalet-edge.md
  - ../plan/tinkalet-edge.md
  - ./participant-authority.md
  - ./tinkalet-item-records.md
  - ../../../tasks/tinkabot-objective-okr.md
---

# App Action Revision Contract

## Scope

CKR-TURN needs a generic participant app-action path before any turn-based game
example. The platform stores participant actions and shared app state as
NATS-backed item material. It does not know tic-tac-toe, cells, turns, score, or
game rules.

## Contract

Participant app actions enter through a Tinkabot-owned request/reply subject and
are materialized into `tb_items` only after Tinkabot validates the selected
participant, app scope, action id, state item, and expected state revision.

```text
tinkalet action submit move-1 --state apps.demo.state.board --base-revision 7
    |
    | selected app-participant profile
    v
tb.app.demo.participants.alice.action
    |
    | Tinkabot action service checks apps.demo.state.board revision == 7
    v
tb_items/apps.demo.participants.alice.actions.move-1
    |
    | bundle/app transformer watches actions and resolves shared state
    v
tb_items/apps.demo.state.board
```

Action item value:

```json
{
  "kind": "tinkabot.appAction.v1",
  "appId": "demo",
  "participantId": "alice",
  "actionId": "move-1",
  "stateKey": "apps.demo.state.board",
  "baseRevision": 7,
  "payload": {}
}
```

## RED Acceptance

| Requirement | RED proof |
| --- | --- |
| Participant action goes through product service | `TestParticipantAppActions` submits with an `app-participant` profile and expects a durable action item. |
| State revision is authoritative | Submitting with an old `--base-revision` is denied as `stale-revision` before an action item appears. |
| Action id is idempotent per participant | Reusing the same action id is denied as `duplicate-action`. |
| Participants cannot bypass the action contract | Direct `item create` or raw `$KV` writes to the action prefix are denied for participant profiles. |
| Scoped reads feed UI sync | Participant profiles may read `apps.<app>.state.>` and their own `apps.<app>.participants.<id>.actions.>` records, but cannot read another app or participant. |
| Shared state revision is checked by Tinkabot | Participants submit a `baseRevision`; Tinkabot reads `apps.<app>.state.>` and denies stale actions. Participants cannot resolve shared state directly. |

Expected RED before GREEN:

```bash
cd substrate/go && go test ./tinkabot -run TestParticipantAppActions -count=1
cd substrate/go && go test ./tinkalet -run TestActionCommandDenials -count=1
```

The first command fails because `tinkalet action submit` does not exist and
participant profiles can still create their own action KV items directly. The
second command fails because the Tinkalet CLI has no action command grammar.

## GREEN Boundary

The smallest acceptable GREEN is a generic action submit service plus Tinkalet
grammar:

- participant profiles may request only their own
  `tb.app.<app>.participants.<id>.action` subject.
- Tinkabot reads shared state from `apps.<app>.state.>` and compares the KV
  revision to `baseRevision`; participant profiles get only scoped read grants
  for app state and their own action records, not a broad KV read grant.
- Tinkabot creates action items under
  `apps.<app>.participants.<id>.actions.<action-id>` with `kv.Create`, so
  duplicate ids are rejected by NATS KV.
- app or bundle logic remains responsible for legal/illegal turn evaluation and
  resolving shared state with the existing item revision compare-and-set path.

## GREEN Evidence

Implemented on 2026-06-24 as a CKR-TURN prerequisite slice, not a complete
turn-based game mission.

| Requirement | GREEN proof |
| --- | --- |
| Participant action goes through product service | `tinkalet action submit` infers app and participant from the selected `app-participant` profile and requests `tb.app.<app>.participants.<id>.action`. |
| State revision is authoritative | Tinkabot action service reads the shared state item and denies old `baseRevision` values as `stale-revision` before creating an action item. |
| Action id is idempotent per participant | Action materialization uses `kv.Create` at `apps.<app>.participants.<id>.actions.<action-id>` and maps key-exists to `duplicate-action`. |
| Participants cannot bypass the action contract | Participant direct `item create`/`item resolve` writes are denied as `denied-scope`, and raw `$KV` or cross-participant action subjects are denied by NATS permissions. |
| Scoped reads feed UI sync | `TestParticipantAppActions` proves a participant can read the current app state and its own action record while cross-app/cross-participant access remains denied. |

Focused verification:

```bash
cd substrate/go && go test ./tinkalet -run TestActionCommandDenials -count=1
cd substrate/go && go test ./tinkabot -run 'TestAppActionMalformedSubject|TestParticipantAppActions|TestParticipantAuthority' -count=1
cd substrate/go && go test ./... -count=1
scripts/c3-line-coverage-harness.sh
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh check --include-adr
```

Current result: all commands pass. Latest C3 coverage is
`owned_files=477`, `lookup_errors=0`, `uncovered=0`; C3 structural check is
`total: 28`, `ok: true`.

Review closure:

| Review gap | Closure |
| --- | --- |
| Action service looked broader than needed. | `actionServicePerms` is scoped to app-state direct reads and action-subtree direct reads/writes only. |
| Participant scoped reads were ambiguous. | Docs and `TestParticipantAppActions` now prove allowed app-state/own-action reads and denied cross-app reads. |
| Denied participant reads could surface as connection failure. | Tinkalet item reads now pass the selected profile into error mapping so participant permission errors return `denied-scope`. |
| Malformed action subjects should remain covered. | `TestAppActionMalformedSubject` exercises wrong shape, uppercase app, uppercase participant, and wrong tail token before any KV dependency. |

## Non-Goals

- No tic-tac-toe board, cell, turn, or player API in the platform.
- No custom realtime channel.
- No direct credentials, subjects, bucket names, or store handles exposed to
  generated UI.
- No client-only stale or turn enforcement.
