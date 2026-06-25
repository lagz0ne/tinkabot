# Tinkabot Real Chess App OKR

## Objective

Deliver a real two-player browser chess app on Tinkabot: a chess-only UI where
player one opens a board-scoped link with a name, player two opens the same
board link with a name, both play legal chess moves in turn, and a checkmate
winner is decided and materialized through the Tinkabot/Tinkalet/NATS reaction
chain.

Target metric: `1 / 1` release-shaped chess acceptance demo passes and leaves a
Tailscale-reachable app open for manual play. Current measured state:
`1 / 1` for the acceptance proof; held manual links are live at the Tailscale
URL recorded below.

## Anti-Goals

| Anti-goal | Metric / tripwire | Why |
| --- | --- | --- |
| No raw browser authority | `authorityLeakCount == 0`; generated UI never receives NATS URL, credential, subject write access, seed, token, or raw KV writer. | Browser players are untrusted participants. |
| No platform chess API | `platformChessAPIAdditions == 0`; no Go/Tinkabot board, piece, move, checkmate, or chess subject primitive. | Chess is an app/bundle/reducer behavior over generic participant actions. |
| No hand-rolled chess legality | reducer uses a proven chess rules engine; no custom implementation of legal move generation, checkmate, castling, en-passant, or draw rules. | Chess rules are edge-case heavy and should not be guessed. |
| No fake app success | acceptance requires two real browser links/pages and manual-click-compatible UI; script-only reducer proofs do not count. | The user must know what to look at and be able to play both sides. |
| No visual clutter | first screen is only the chess/join experience; no generic Tinkabot shell copy, no diagnostic panels as primary UI. | The demo must feel like a real chess app, not a harness. |
| No local-only demo | user-facing URL must be Tailscale-reachable and verified during the run. | The user cannot access localhost. |
| No bypassing Tinkalet/reaction path | terminal state must be produced through the same participant action plus reducer apply/reject chain. | The point is the mechanism, not chess alone. |
| No cross-board/user bleed | board number and player identity scope every read/write/action; wrong-board and wrong-player actions are denied or ignored with proof. | Board number is the app-level isolation key. |

## Action Envelope

Allowed:

- Add a chess app mode to the browser shell/generated content if it remains
  generic from the platform perspective.
- Add a demo script such as `demo:chess` that builds the release archive,
  starts packaged Tinkabot, opens or verifies two browser players, and runs the
  reducer.
- Add a dependency for chess rules if it is scoped to app/demo/frontend code and
  not Tinkabot platform primitives.
- Use CDN-hosted browser dependencies only when the demo remains deterministic
  enough for proof, or prefer package dependency if that gives better local
  verification.

Forbidden:

- Add raw browser NATS credentials or direct browser KV writes.
- Add chess-specific Go substrate APIs or NATS subject primitives.
- Count a page that renders generic shell text instead of the actual board.
- Count a proof that cannot be manually opened through Tailscale.
- Claim arbitrary SaaS/multitenant chess service readiness.

## DKR Queue

| DKR | Question | Budget | Expected output |
| --- | --- | --- | --- |
| DKR-CHESS-1 | What is the smallest app architecture that gives a real chess UI while preserving the generic platform boundary? | One discovery pass | Task contract and implementation file scope. |
| DKR-CHESS-2 | Which chess rules engine should own legality in reducer/browser preview? | One dependency inspection pass | Engine choice with proof it covers legal moves and terminal results. |
| DKR-CHESS-3 | Can existing `participant_read` / `participant_action` express join, move, and terminal state without Go changes? | One focused code inspection | Decision: frontend/script-only app layer vs generic bridge extension. |

## CKRs

| CKR | Metric | Acceptance |
| --- | --- | --- |
| CKR-CHESS-JOIN | `2 / 2` named players join the same board code through browser UI. | Board state materializes player names, colors, and waiting/ready status; duplicate join is denied. |
| CKR-CHESS-PLAY | `1 / 1` legal turn sequence reaches terminal state. | Browser-origin moves are legal chess moves, alternate turns, update board state, and materialize checkmate through reducer apply/reject. |
| CKR-CHESS-DENIALS | `4 / 4` invalid cases are proven. | Wrong turn, illegal move, wrong board, and stale/duplicate move are denied without state mutation. |
| CKR-CHESS-UI | `1 / 1` manual Tailscale chess app is usable. | Two links are provided; first viewport is a real chess board/join flow only; both pages stay synchronized. |
| CKR-CHESS-DOCS | `1 / 1` README/task docs explain setup/use from multiple viewpoints. | User, app author/reducer, and LLM/integration watcher views are documented without raw-NATS as normal path. |

## RED State

Current RED:

- The present held link is not a real product chess app and does not make clear
  what the user should inspect.
- The tic-tac-toe board mode proves the generic participant/reducer mechanism
  but is still a harness, not a real chess game.
- There is no join flow for player names and board code.
- There is no chess rules engine in the app/reducer path.
- There is no terminal chess result materialized by a reaction chain.

## GREEN State

`bun run demo:chess` passed on 2026-06-25. Proof:
`/tmp/tinkabot-chess-live.k143Sf/dist/chess-proof.json`.

Measured result:

- `kind`: `tinkabot.realChessAppProof.v1`
- `route`: `tailscale`
- `shellUrl`: `http://forge.tail6c789a.ts.net:44937`
- players: `Alice` as white, `Bob` as black
- terminal result: checkmate, black wins, `Qh4#`
- denials: duplicate join, wrong-board read, wrong board, wrong turn, illegal move,
  duplicate action, stale revision
- reaction mode: `tinkalet.watch.prefix`
- browser DOM: 64 chess squares, parent generic shell text absent
- name input stability: `nameTypingStable: true`
- focused name-input review: Codex and Claude `VERDICT: PASS`
- `authorityLeakCount`: `0`
- `platformChessAPIAdditions`: `0`
- legality engine: `chess.js@1.4.0`
- `pass`: `true`

Held manual board:

- run: `/tmp/tinkabot-chess-live.k143Sf/dist`
- reducer log: `/tmp/tinkabot-chess-live.k143Sf/dist/chess-manual-reducer.log`
- manual smoke proof: `/tmp/tinkabot-chess-live.k143Sf/dist/chess-manual-smoke.json`
- Alice: `http://forge.tail6c789a.ts.net:44937/?tb_app=demo&tb_participant=alice&tb_state=apps.demo.state.chess.board-1782376267705-manual&tb_session=demo-001&tb_chess=1&tb_board_no=board-1782376267705-manual`
- Bob: `http://forge.tail6c789a.ts.net:44937/?tb_app=demo&tb_participant=bob&tb_state=apps.demo.state.chess.board-1782376267705-manual&tb_session=demo-001&tb_chess=1&tb_board_no=board-1782376267705-manual`

Expected first RED command:

```bash
bun run demo:chess
```

It should fail until a release-shaped chess demo exists.

## Freshness And Evidence

Every GREEN claim must name:

- proof JSON path.
- Tailscale URL(s).
- package root used.
- reducer log path.
- focused test commands.
- anti-goal readings.

Do not count task completion alone as objective progress; count only the direct
acceptance demo and anti-goal readings.
