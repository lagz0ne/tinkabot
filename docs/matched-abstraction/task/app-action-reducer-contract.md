---
layer: task
topic: app-action-reducer-contract
references:
  - ./app-action-revision-contract.md
  - ./tinkalet-item-records.md
  - ../../../tasks/tinkabot-objective-okr.md
---

# App Action Reducer Contract

## Scope

CKR-TURN needs a generic reducer step after action ingress. This slice still
does not create a tic-tac-toe board, cell, turn, or game API. It gives app or
bundle logic one product-shaped way to consume a pending action and resolve the
shared state item with the action's recorded base revision.

## Contract

```text
apps.demo.participants.alice.actions.move-1
    | pending action carries stateKey + baseRevision + payload
    v
tinkalet action apply apps.demo.participants.alice.actions.move-1 --value <next-state>
    | reducer profile reads action and state
    | state revision must equal action.baseRevision
    v
tb_items/apps.demo.participants.alice.actions.move-1.receipt
    | pending receipt is created as the apply/reject claim
    v
tb_items/apps.demo.state.board
    | resolved with KV compare-and-set
    v
tb_items/apps.demo.participants.alice.actions.move-1.receipt
    | updated to resolved/applied
```

The reducer is generic: the command accepts a next-state JSON value that app
logic has already derived. It does not know turn order, cells, scores, or
whether a move is legal.

## RED Acceptance

| Requirement | RED proof |
| --- | --- |
| Reducer consumes pending action | `TestParticipantAppReducer` submits an action, then applies it by action key. |
| State update is CAS-bound | If shared state advances after action submission but before apply, apply is denied as `stale-revision` and no receipt appears. |
| Apply is idempotent by receipt | Reapplying an already-applied action returns `duplicate-action` without changing state again. |
| Participant cannot reduce | An `app-participant` profile cannot run `action apply`; it receives `denied-scope`. |
| Receipt is durable material | Successful apply creates `<action-key>.receipt` with action key, state key, and resulting state revision. |

Expected RED before GREEN:

```bash
cd substrate/go && go test ./tinkabot -run TestParticipantAppReducer -count=1
```

It fails before implementation because `tinkalet action apply` is not part of
the CLI grammar.

## GREEN Boundary

- `action apply` reads a pending `tinkabot.appAction.v1` item.
- It reads the action's `stateKey` and compares the state KV revision with
  `baseRevision`.
- It creates a deterministic pending receipt at `<action-key>.receipt` before
  state mutation so apply/reject share one NATS KV serialization point.
- It updates the state item through NATS KV `Update`, not direct in-memory
  state.
- It updates the deterministic receipt to `resolved` / `applied` after the
  state CAS succeeds.
- It rejects participant profiles before opening mutation authority.

## GREEN Evidence

Implemented on 2026-06-24 as a CKR-TURN reducer prerequisite slice, not a
complete turn-based game mission.

| Requirement | GREEN proof |
| --- | --- |
| Reducer consumes pending action | `tinkalet action apply <action-key> --value <next-state>` reads the action item and applies the derived state. |
| State update is CAS-bound | `TestParticipantAppReducer` advances state between submit and apply, then expects `stale-revision` and no receipt. |
| Apply is idempotent by receipt | Reapplying an action with an existing receipt returns `duplicate-action`. |
| Participant cannot reduce | The selected `app-participant` profile receives `denied-scope` before reducer mutation. |
| Receipt is durable material | Successful apply creates `<action-key>.receipt` as `tinkabot.appActionReceipt.v1`, then updates it to `resolved` / `applied` with action key, state key, action revision, and state revision. |
| Action state key remains scoped | A forged action item whose `stateKey` points outside `apps.<app>.state.>` is denied as `malformed-action`. |

Focused verification:

```bash
cd substrate/go && go test ./tinkabot -run 'TestAppActionMalformedSubject|TestParticipantAppActions|TestParticipantAppReducer|TestParticipantAuthority' -count=1
cd substrate/go && go test ./tinkalet -count=1
cd substrate/go && go test ./... -count=1
scripts/c3-line-coverage-harness.sh
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh check --include-adr
```

Current result: all commands pass. Latest C3 coverage is
`owned_files=477`, `lookup_errors=0`, `uncovered=0`; C3 structural check is
`total: 28`, `ok: true`.

Follow-up correction after the turn reference review: apply/reject now serialize
on the deterministic receipt key before state mutation. Crash recovery from a
remaining `pending` receipt is outside this prerequisite slice.

## Non-Goals

- No game-specific legality rules.
- No realtime measurement.
- No client-only state mutation.
- No reducer daemon loop yet; this slice proves the product command.
