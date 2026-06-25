---
layer: task
topic: real-chess-app
references:
  - ../../../tasks/tinkabot-real-chess-okr.md
  - ./browser-participant-action-bridge.md
  - ./realtime-browser-participant-ui.md
  - ./browser-turn-based-board.md
---

# Real Chess App

## Objective

Replace the harness-like board demo with a real two-player chess app surface:
the first player opens a board-scoped link and enters a name, the second player
opens the same board link and enters a name, both play legal chess turns, and a
checkmate terminal result is decided by the app reducer through the existing
Tinkabot/Tinkalet/NATS reaction chain.

## Scope

- Browser UI is chess-only when `tb_chess=1`.
- Join state, player names, board code, color assignment, move history, FEN,
  and result live in one app state item under `apps.demo.state.chess.<board>`.
- Browser players submit generic `participant_action` payloads for `join`,
  `move`, and optional `resign`.
- A demo-owned reducer uses `chess.js` for legality and terminal status, then
  resolves actions through `tinkalet action apply` / `action reject`.
- Tinkabot platform code remains generic.

## Anti-Goal Checks

| Anti-goal | Task check |
| --- | --- |
| No raw browser authority | Proof records `authorityLeakCount: 0`; generated frame receives only lease/demo data, command replies, and pushed state messages. |
| No platform chess API | No Go change should be required; no chess/piece/move subject or Tinkabot command is added. |
| No hand-rolled legality | Reducer imports `chess.js`; it does not implement legal move generation itself. |
| No fake app success | Demo opens two browser pages, verifies chess DOM, and leaves a held Tailscale URL for manual play. |
| No visual clutter | Chess mode renders only join/status/board/move list, not the generic proof panel as primary experience. |
| No local-only demo | Demo fails unless Tailscale route is verified, except explicit local development opt-in. |
| No reaction bypass | Terminal result is written by the reducer applying/rejecting participant action records through Tinkalet. |
| No cross-board/user bleed | Proof includes wrong-board action denial, wrong-board read denial, and scoped board state. |

## RED Artifact

Current RED:

```bash
bun run demo:chess
```

The script does not exist. The current browser demo can render a tic-tac-toe
style board, but it has no chess-only UI, no player-name/board-number join
flow, and no chess legality engine or terminal chess result.

## Acceptance Contract

`bun run demo:chess` is GREEN only when it proves:

| Requirement | Evidence |
| --- | --- |
| Release-shaped package | Build release archive and use packaged `tinkabot` / `tinkalet`. |
| Chess-only links | Print Alice and Bob Tailscale links with `tb_chess=1`, a board-scoped code, and no generic proof surface as the primary page. |
| Join flow | Browser-origin `join` actions materialize two named players on the linked board code with colors. |
| Legal play | Browser-origin legal moves update FEN, turn, move list, and both pages' board DOM. |
| Invalid denials | Wrong turn, illegal move, duplicate/stale move, and third-player or wrong-board join are denied without mutating board state. |
| Terminal result | A legal sequence reaches checkmate; result is materialized in board state and visible on both pages. |
| Authority | Proof has `authorityLeakCount: 0` and no raw NATS credentials/subjects in generated UI output. |
| Platform generic | No platform chess primitive or raw browser KV writer is introduced. |

Expected proof JSON:

```json
{
  "kind": "tinkabot.realChessAppProof.v1",
  "route": "tailscale",
  "board": "demo-001",
  "players": {"white": "Alice", "black": "Bob"},
  "terminal": {"status": "checkmate", "winner": "black"},
  "authorityLeakCount": 0,
  "pass": true
}
```

## Verification

Focused checks:

```bash
bash -n scripts/demo-chess.sh
bun test apps/frontend/tests/isolation.test.ts apps/frontend/tests/observe.test.ts
bunx @typescript/native-preview --noEmit -p apps/frontend/tsconfig.json
bun run demo:chess
git diff --check
```

If platform Go code changes, add focused Go tests and update this task with the
reason. The expected path is frontend/script only.

## GREEN Evidence

Implemented on 2026-06-25 as `bun run demo:chess`. The demo builds the release
archive, starts packaged Tinkabot with `demo:alice` and `demo:bob`, exposes the
shell through Tailscale, seeds a chess app state item, opens two browser pages,
joins Alice and Bob by name and linked board code, and drives Fool's Mate through
browser-origin participant actions. The reducer uses `chess.js@1.4.0` and
consumes every materialized join/move from `tinkalet watch prefix`, then
resolves the action through packaged `tinkalet action apply` or `action reject`.

Latest proof:

```text
/tmp/tinkabot-chess-live.k143Sf/dist/chess-proof.json
shellUrl=http://forge.tail6c789a.ts.net:44937
kind=tinkabot.realChessAppProof.v1
route=tailscale
players.white=Alice
players.black=Bob
terminal.status=checkmate
terminal.winner=black
terminal.san=Qh4#
denials=duplicate-join,wrong-board-read,wrong-board,wrong-turn,illegal-move,duplicate-action,stale-revision
reactionMode=tinkalet.watch.prefix
watchedReducerOk=true
boardSquares=64
parentGenericShellText=false
nameTypingStable=true
authorityLeakCount=0
platformChessAPIAdditions=0
legalityEngine=chess.js@1.4.0
pass=true
```

Focused verification:

```bash
bash -n scripts/demo-chess.sh
bun test apps/frontend/tests/isolation.test.ts apps/frontend/tests/observe.test.ts
bunx @typescript/native-preview --noEmit -p apps/frontend/tsconfig.json
bun run demo:chess
git diff --check -- apps/frontend/src/main.ts apps/frontend/src/style.css apps/frontend/src/fixture.ts apps/frontend/package.json bun.lock package.json scripts/demo-chess.sh tasks/tinkabot-real-chess-okr.md docs/matched-abstraction/task/real-chess-app.md tasks/todo.md
```

Independent review:

```text
Codex follow-up: VERDICT: PASS
Claude follow-up: VERDICT: PASS
Name-input refresh follow-up: Codex VERDICT: PASS; Claude VERDICT: PASS
```

The follow-up reviews specifically rechecked the board-code scope, persisted
manual-smoke proof, and checkmate-only terminal claim after Codex's initial
failure findings. The name-input refresh follow-up rechecked that polling still
reads state while only DOM re-render is skipped for a focused name field.

NATS-native UI watch follow-up on 2026-06-25 supersedes the polling note above:
`docs/matched-abstraction/task/nats-native-ui-watch.md` removes generated-frame
polling and proves the chess state path is `trusted-shell.nats-watch.push` with
`generatedIframePollingCount=0` measured from the trusted shell dispatch log.

GREEN readings:

| Anti-goal | Reading |
| --- | --- |
| No raw browser authority | `authorityLeakCount: 0`. |
| No platform chess API | `platformChessAPIAdditions: 0`; no Go files were touched. |
| No hand-rolled chess legality | Reducer imports `chess.js@1.4.0`; proof records it as `legalityEngine`. |
| No fake app success | Proof opens two browser pages, verifies 64 chess squares, and verifies the parent page has no generic shell text. |
| No local-only demo | Proof route is `tailscale`. |
| No reaction bypass | Action log is consumed from `tinkalet watch prefix`, then resolved through packaged Tinkalet reducer commands. |
| No cross-board/user bleed | Wrong-board action is rejected; wrong-board read is denied by the trusted shell; duplicate join is rejected. |
| No refresh loop breaking input | Proof slow-types both player names while state is delivered by NATS push and records `nameTypingStable: true`. |

Residual non-claims:

- This is a two-player reference app, not a SaaS lobby or arbitrary-board
  provisioning service.
- Board codes are created by the demo harness and carried by the two links for
  this slice; arbitrary user-created boards are a future product extension.
- The browser UI submits moves through the trusted shell, receives action
  replies as `tinkabot.command.result`, and receives realtime state updates as
  `tinkabot.state` messages from the shell-held NATS subscription; it still does
  not receive raw NATS credentials or direct KV write authority.

## Held Manual Demo

Started on 2026-06-25 with:

```bash
TINKABOT_DEMO_HOLD=1 bun run demo:chess /tmp/tinkabot-chess-live.k143Sf/dist
```

The held run passed the proof, then created fresh manual board
`board-1782376267705-manual` and left the reducer running at:

```text
/tmp/tinkabot-chess-live.k143Sf/dist/chess-manual-reducer.log
/tmp/tinkabot-chess-live.k143Sf/dist/chess-manual-smoke.json
```

Manual links:

```text
Alice: http://forge.tail6c789a.ts.net:44937/?tb_app=demo&tb_participant=alice&tb_state=apps.demo.state.chess.board-1782376267705-manual&tb_session=demo-001&tb_chess=1&tb_board_no=board-1782376267705-manual
Bob:   http://forge.tail6c789a.ts.net:44937/?tb_app=demo&tb_participant=bob&tb_state=apps.demo.state.chess.board-1782376267705-manual&tb_session=demo-001&tb_chess=1&tb_board_no=board-1782376267705-manual
```

Browser smoke verified both links render `Chess board
board-1782376267705-manual`, `waiting`, 64 board squares, persisted typed
name values, and no generic shell text.
