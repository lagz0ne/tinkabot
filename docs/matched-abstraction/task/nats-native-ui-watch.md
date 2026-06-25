---
layer: task
topic: nats-native-ui-watch
references:
  - ../../../tasks/tinkabot-nats-native-max-okr.md
  - ./real-chess-app.md
  - ./browser-participant-action-bridge.md
  - ./realtime-participant-watch.md
---

# NATS-Native UI Watch

## Objective

Move browser app-state sync from generated-frame `participant_read` polling to
NATS-native push: Tinkabot owns a scoped KV watch, republishes item changes to a
per-viewer state subject, the trusted shell subscribes over NATS WebSocket, and
generated UI receives realtime state updates only as leased `tinkabot.state`
messages. User action replies still return as leased `tinkabot.command.result`
messages.

## Boundary

- Generated content does not receive NATS URLs, JWTs, seeds, raw subjects, or
  KV handles.
- The trusted shell may hold the viewer bearer and subscribe to the
  viewer-scoped browser state subject minted with that bearer.
- The backend browser-command service owns KV watch creation and validates
  app/participant/key scope before publishing state events.
- The platform remains generic: no chess, board, or typeracing primitive is
  added.

## RED

Before this task, chess mode called `participant_read` from the generated frame
on a timer and after moves. That was secure but not the product realtime model:
the UI was asking for state instead of reacting to NATS-pushed material.

## GREEN

- `participant_watch` is a shell-only browser command. It validates the same
  participant state scope as `participant_read`, starts a KV `WatchFiltered`
  on `tb_items`, and publishes `tinkabot.browserState.v1` events to
  `tb.app.browser.state.<viewer-nonce>.<app>.<participant>.<hash>`.
- Viewer credentials can subscribe only to their own opaque browser-state
  branch and request/reply inboxes; generated content never sees the delivery
  subject or prefix.
- The trusted shell subscribes, receives pushed state events, runs the raw
  authority filter, and posts a leased `tinkabot.state` message into the
  sandboxed frame.
- Generated board/chess modes removed product refresh loops and wait on pushed
  state before submitting actions.

## Evidence

Latest proof:

```text
/tmp/tinkabot-chess-demo.XbwrdL/chess-proof.json
route=tailscale
shellUrl=http://forge.tail6c789a.ts.net:36681
stateDelivery=trusted-shell.nats-watch.push
generatedIframePollingCount=0
generatedIframePollingProof.source=trusted-shell.dispatched
generatedIframePollingProof.stateReadCommands=[]
stateVisibleP95Ms=71
stateVisibleP99Ms=71
touchMoveProof.pass=true
visualSmoke.pass=true
authorityLeakCount=0
pass=true
```

Independent review:

```text
Claude noninteractive: VERDICT: PASS
Codex noninteractive: VERDICT: PASS
```

Focused checks:

```bash
cd substrate/go && go test ./tinkabot -run TestBrowserParticipantActionBridge -count=1 -v
cd substrate/go && go test ./tinkabot -run 'TestBrowserParticipantActionBridge|TestWebSessionShell' -count=1
cd substrate/go && go test ./embednats -run TestWebSessionSurface -count=1
bun test apps/frontend/tests/isolation.test.ts apps/frontend/tests/observe.test.ts
bunx @typescript/native-preview --noEmit -p apps/frontend/tsconfig.json
bun run --cwd apps/frontend build
bash -n scripts/demo-chess.sh
bun run demo:chess
```

## Residual Non-Claims

- Browser state push is proven for app state items used by participant apps.
  It does not grant generated content direct JetStream/KV API access.
- The clock bundle remains a bundle/Tinkalet tour; the current browser realtime
  product proof is the chess app-state path.
- Load-envelope expansion beyond this proof belongs to `CKR-LOAD-WATCH`.
