---
layer: task
topic: browser-turn-based-board
references:
  - ../../../tasks/tinkabot-objective-okr.md
  - ./turn-based-reference.md
  - ./browser-participant-action-bridge.md
  - ./realtime-browser-participant-ui.md
---

# Browser Turn-Based Board

## Objective

Make the tic-tac-toe-shaped turn reference playable from the browser while
preserving the platform boundary: generated UI submits participant actions
through Tinkabot's trusted shell, and a demo-owned reducer applies or rejects
those actions through Tinkalet. Tinkabot stays generic and must not learn board,
cell, turn, score, or winner semantics.

## Scope

This Task combines two already-green prerequisites:

- `demo:turn` proves a release-shaped reducer sequence with wrong-turn,
  occupied-cell, stale-revision, duplicate-action, and final-winner evidence.
- `demo:realtime-browser` proves two leased browser participants can submit
  participant actions and read their own material through the trusted shell.

The missing behavior is one browser-playable board demo that keeps the running
browser state current as the reducer resolves actions.

## Non-Goals

- No raw NATS credential or subject in generated browser content.
- No board/game-specific Tinkabot API.
- No non-NATS game server.
- No max-rate or max-participant claim.
- No SaaS/control-plane claim.
- No dead local-only demo URL when showing the result; use the Tailscale route
  for user-visible output.

## Execution Contract

```text
packaged Tinkabot
  -> admits demo:alice and demo:bob participant profiles
  -> serves trusted shell over a Tailscale-reachable URL
  -> browser page for Alice reads apps.demo.state.board
  -> browser page for Bob reads apps.demo.state.board
  -> UI click or scripted browser event posts participant_action
  -> Tinkabot action service materializes pending action item
  -> demo reducer loop reads pending action material
  -> reducer uses tinkalet action apply or action reject
  -> shared board item changes or denial receipt appears
  -> both browser pages read or observe the new board state and update DOM
```

The reducer may live inside the demo script or a small demo helper. It must use
owner/reducer authority through Tinkalet product commands, not raw NATS access,
unless the command is explicitly isolated as operator diagnostics and excluded
from the user integration path.

## Acceptance Contract

The slice is GREEN only when `bun run demo:turn-browser` or an equivalently
named package script proves all rows below.

| Requirement | Evidence |
| --- | --- |
| Release-shaped package | Demo builds the release archive, unpacks it, and uses packaged `tinkabot` and `tinkalet`. |
| Remote show URL | Demo prints a Tailscale-reachable shell URL; if the demo is held open, the URL remains reachable after the command prints it. |
| Two participant browsers | Proof opens Alice and Bob as distinct browser pages or equivalent distinct frame leases. |
| Browser-origin moves | At least five legal moves are submitted from browser participant actions, not direct Tinkalet action submit calls. |
| Reducer-owned rules | Wrong-turn and occupied-cell actions are rejected by the demo reducer with durable receipts and no state mutation. |
| Substrate denials | Duplicate action id and stale base revision are denied by the existing action substrate. |
| Board sync | Both browser pages render the same final board and winner after reducer application. |
| Authority boundary | Proof records zero raw-authority leaks and denies any generated-frame attempt to choose another app or participant. |
| Generic platform | The implementation adds no Tinkabot board/cell/winner API and no direct browser KV writer. |

The proof JSON should include:

- `kind: "tinkabot.browserTurnBoardProof.v1"`
- shell URL, local shell URL, and route kind.
- participants and their DOM board snapshots.
- move log with action key, participant, cell, before revision, after revision,
  and receipt outcome.
- denial log for wrong turn, occupied cell, duplicate action, stale revision,
  and participant/app escape.
- final board with `winner: "alice"`.
- `authorityLeakCount: 0`.

## RED Artifact

Current RED: `bun run demo:turn` can finish a tic-tac-toe-shaped sequence, but
the browser UI cannot play that board. `apps/frontend/src/fixture.ts` only has
the realtime counter participant flow; it does not render a board, let the user
choose cells, or refresh both participants after reducer resolution.

Expected first RED check:

```bash
bun run demo:turn-browser
```

The command should fail before implementation because no script exists.

## Suggested File Scope

Preferred implementation ownership for the worker:

- Add `scripts/demo-turn-browser.sh`.
- Add `demo:turn-browser` to `package.json`.
- Extend the existing frontend fixture/shell path only as much as needed to
  support board mode. Keep the realtime-browser flow working.
- Update this Task doc with GREEN evidence after the demo passes.
- Add or adjust focused frontend tests if shell/fixture behavior changes.

Only touch Go code if the existing browser command route cannot express the
contract above. If Go changes are needed, keep them generic and update the
browser participant bridge evidence.

## Verification

Run the narrowest checks that prove the slice:

```bash
bash -n scripts/demo-turn-browser.sh
bun test apps/frontend/tests/isolation.test.ts apps/frontend/tests/observe.test.ts
bunx @typescript/native-preview --noEmit -p apps/frontend/tsconfig.json
bun run demo:turn-browser
git diff --check
```

If Go is touched:

```bash
cd substrate/go && go test ./tinkabot -run 'TestBrowserParticipantActionBridge|TestTurnBasedReferenceMission|TestParticipantAppReducer' -count=1
```

Before claiming the slice complete, run the C3 lookup/eval path for changed
files and record any residual non-claims in this doc.

## GREEN Evidence

Implemented on 2026-06-25 as the browser-playable turn board reference demo.
The implementation keeps the board rules in `scripts/demo-turn-browser.sh`; the
trusted shell and generated frame still use only generic `participant_read` and
`participant_action` commands.

| Requirement | GREEN proof |
| --- | --- |
| Release-shaped package | `bun run demo:turn-browser` built `/tmp/tinkabot-turn-browser-demo.5jDpfs/tinkabot-v0.1.1-linux-amd64` from the release archive and used packaged `tinkabot` / `tinkalet`. |
| Remote show URL | The demo verified Tailscale route `http://forge.tail6c789a.ts.net:33975` and local shell `http://127.0.0.1:33975`; `TINKABOT_DEMO_HOLD=1` keeps that route open for manual play. |
| Two participant browsers | Proof opened Alice and Bob pages with distinct leases and board DOM snapshots. |
| Browser-origin moves | Five accepted moves originated from generated browser content: Alice `a1`, Bob `b1`, Alice `a2`, Bob `b2`, Alice `a3`. |
| Reducer-owned rules | The demo reducer rejected Bob wrong-turn and occupied-cell actions with durable `action reject` receipts and unchanged state revisions. |
| Substrate denials | Browser-origin duplicate id `a1` returned `duplicate-action`; stale base revision `b-stale` returned `stale-revision` and no action item materialized. |
| Board sync | Alice and Bob DOM snapshots both rendered `winner: "alice"` and the same final cells. |
| Authority boundary | Generated-frame participant escape was denied as `FrameScopeEscape`; proof recorded `authorityLeakCount: 0`. |
| Generic platform | No Go changes and no board/cell/winner platform API were added. |

Focused verification:

```bash
bash -n scripts/demo-turn-browser.sh
bun test apps/frontend/tests/isolation.test.ts apps/frontend/tests/observe.test.ts
bunx @typescript/native-preview --noEmit -p apps/frontend/tsconfig.json
bun run demo:turn-browser
git diff --check
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh lookup scripts/demo-turn-browser.sh
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh lookup apps/frontend/src/fixture.ts
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh lookup apps/frontend/src/main.ts
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh lookup docs/matched-abstraction/task/browser-turn-based-board.md
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-302
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-401
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-501
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-502
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh check --include-adr
```

Latest proof:

```text
/tmp/tinkabot-turn-browser-demo.5jDpfs/browser-turn-board-proof.json
kind=tinkabot.browserTurnBoardProof.v1
route=tailscale
moves=5
denials=participant-escape,wrong-turn,duplicate-action,stale-revision,occupied-cell
finalBoard.winner=alice
authorityLeakCount=0
```

Residual non-claims:

- This is a two-browser reference demo, not a max-participant or max-rate claim.
- Generated browser content can submit and read through the trusted shell; it
  does not receive raw NATS credentials or direct KV writer authority.
- The demo reducer is intentionally outside Tinkabot. Tinkabot still has no
  board, cell, turn, or winner API.
