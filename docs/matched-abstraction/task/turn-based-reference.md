---
layer: task
topic: turn-based-reference
references:
  - ./participant-authority.md
  - ./app-action-revision-contract.md
  - ./app-action-reducer-contract.md
  - ../../../tasks/tinkabot-objective-okr.md
---

# Turn-Based Reference

## Scope

CKR-TURN counts only when a release-shaped turn flow proves the generic
participant/action/reducer substrate can complete a legal turn sequence and
materialize denied turns. The example may look like tic-tac-toe, but the
platform must stay unaware of boards, cells, turns, or winners.

## Contract

```text
tinkabot --participant demo:alice --participant demo:bob
    |
    | writes app-participant local profile descriptors
    v
tinkalet profile import local --store <participant-store>
    |
    | participant profile submits action with state revision
    v
tinkalet action submit <id> --state apps.demo.state.board --base-revision <n>
    |
    | app/bundle logic derives either next state or denial reason
    v
tinkalet action apply <action-key> --value <next-state>
    | or
tinkalet action reject <action-key> --reason <reason-token>
    |
    v
tb_items/apps.demo.state.board
tb_items/apps.demo.participants.<id>.actions.<id>.receipt
```

`action reject` is the generic denial mate to `action apply`: it reads the
pending action and state, confirms the action base revision is still current,
and creates a deterministic receipt without mutating shared state. Rule
evaluation remains app/bundle logic. Rejection reasons are lowercase tokens.

## RED Acceptance

| Requirement | RED proof |
| --- | --- |
| Binary admits participants without test-only Go calls | `cmd/tinkabot` accepts repeated `--participant <app>:<id>` flags and writes local profile descriptors for each participant. |
| Legal turn sequence completes | `TestTurnBasedReferenceMission` uses Tinkalet profiles to submit and apply five legal actions until the shared state records a winner. |
| Wrong-turn denial is durable | A participant can submit a syntactically valid action at the current state revision, app logic rejects it with `action reject`, and the state revision/value remain unchanged. |
| Occupied-cell denial is durable | A legal participant action against an occupied logical slot is rejected with a receipt and no state mutation. |
| Stale and duplicate are denied by the substrate | Old base revisions do not materialize an action; reused action ids return `duplicate-action`. |
| No game platform API appears | Tests and demo use only participant profiles, item material, `action submit`, `action apply`, and `action reject`. |

Expected RED before GREEN:

```bash
cd substrate/go && go test ./cmd/tinkabot -run TestRunAdmitsParticipantsFromStartupFlag -count=1
cd substrate/go && go test ./tinkabot -run TestTurnBasedReferenceMission -count=1
cd substrate/go && go test ./tinkalet -run TestActionCommandDenials -count=1
```

The first command fails because the binary cannot admit participants at startup.
The second and third fail because `tinkalet action reject` does not exist.

## GREEN Boundary

- Participant startup admission is a generic binary flag, not a game command.
- Rejection receipts use the same action key and receipt idempotency as apply
  receipts.
- `action reject` denies app-participant profiles before opening mutation
  authority.
- `action reject` validates action shape, scoped state key, and current state
  revision before writing the receipt.
- The turn rules live only in the reference script/test layer.

## GREEN Evidence

Implemented on 2026-06-24 as the CKR-TURN reference proof. This does not count
the scoped multiplayer mission complete yet because realtime-heavy sync and
multiplayer UI proof remain outside this slice.

| Requirement | GREEN proof |
| --- | --- |
| Binary admits participants without test-only Go calls | `TestRunAdmitsParticipantsFromStartupFlag` starts packaged assembly behavior through `cmd/tinkabot` and checks repeated `--participant demo:<id>` profile descriptors and credentials. |
| Legal turn sequence completes | `TestTurnBasedReferenceMission` and `bun run demo:turn` submit and apply five legal actions until the shared state records `winner=alice`. |
| Wrong-turn denial is durable | `action reject --reason wrong-turn` creates a denied receipt while the board state revision/value stay unchanged. |
| Occupied-cell denial is durable | `action reject --reason occupied-cell` creates a denied receipt while the board state revision/value stay unchanged. |
| Stale and duplicate are denied by the substrate | Duplicate action id returns `duplicate-action`; old base revision returns `stale-revision` and no action item materializes. |
| No game platform API appears | The test and demo use startup participant profiles, `item`, `action submit`, `action apply`, and `action reject`; the platform still has no board/cell/winner API. |

Focused verification:

```bash
cd substrate/go && go test ./cmd/tinkabot -run 'TestRunStartsPrintsPostureAndStopsOnSignal|TestRunAdmitsParticipantsFromStartupFlag|TestRunRequiresStore|TestRunPrintsVersion' -count=1
cd substrate/go && go test ./tinkalet -run 'TestActionCommandDenials|TestProfileImportListUse|TestProfileImportDenials' -count=1
cd substrate/go && go test ./tinkabot -run 'TestAppActionMalformedSubject|TestParticipantAppActions|TestParticipantAppReducer|TestTurnBasedReferenceMission|TestParticipantAuthority' -count=1
cd substrate/go && go test ./... -count=1
bash -n scripts/demo-turn-based.sh scripts/demo-live-patch.sh scripts/demo-chain-reaction.sh scripts/release-package.sh scripts/package-tinkabot.sh
bun run demo:turn
scripts/c3-line-coverage-harness.sh
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-302
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh check --include-adr
```

Current result: all commands pass. `bun run demo:turn` produced Tailscale URL
`http://forge.tail6c789a.ts.net:42781` and proof file
`/tmp/tinkabot-turn-demo.EIsQmj/turn-proof.json` with wrong-turn,
duplicate-action, stale-revision, occupied-cell, and final winner evidence.
Latest C3 coverage is `owned_files=479`, `lookup_errors=0`, `uncovered=0`.

Independent follow-up review after the receipt serialization fix: Codex
noninteractive `VERDICT: PASS`, findings none; Claude noninteractive
`VERDICT: PASS`, findings none.

Residual concurrency notes:

- Submit-time stale checks reject an already-old base revision before an action
  item appears. If shared state advances between that check and the action
  create, the pending action may still materialize and must be rejected by the
  reducer CAS path; state safety remains with NATS KV revision checks.
- `action apply` and `action reject` serialize on the deterministic receipt key
  before state mutation, but crash recovery from a `pending` receipt is not yet
  a release mission claim.

## Non-Goals

- No tic-tac-toe, typeracing, or board-specific platform primitive.
- No realtime-rate claim.
- No browser UI completeness claim for the multiplayer mission yet.
- No non-NATS game server or client-owned truth.
