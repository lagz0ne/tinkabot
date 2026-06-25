---
layer: task
topic: anonymous-typerace-golden
references:
  - ./real-chess-app.md
  - ./browser-participant-action-bridge.md
  - ./nats-native-ui-watch.md
  - ./realtime-participant-reference-demo.md
---

# Anonymous Typerace Golden

## Objective

Add a second app-grade multiplayer golden example beside chess: anonymous users
open scoped race links, join without a named account, type against the same
prompt, see each other's progress in realtime, and get a reducer-decided winner
through the existing Tinkabot/Tinkalet/NATS reaction chain.

## Scope

- Browser UI is typeracing-only when `tb_type=1`.
- Anonymous users are still isolated participants: the demo mints opaque
  `anon-*` participant ids and derived scoped credentials at Tinkabot startup.
- Race state lives under `apps.demo.state.typerace.<race>`.
- Generated UI submits generic `participant_action` payloads for `join` and
  `progress`.
- The app reducer consumes action records from `tinkalet watch prefix` and
  resolves them through `tinkalet action apply` or `tinkalet action reject`.
- Tinkabot platform code remains generic.

## Anti-Goal Checks

| Anti-goal | Task check |
| --- | --- |
| No unauthenticated authority | Anonymous means opaque scoped participant ids, not public browser credentials. |
| No raw browser authority | Proof records `authorityLeakCount: 0`; generated content receives only lease/demo data, command replies, and pushed state. |
| No platform typerace API | Proof records `platformTypeRaceAPIAdditions: 0`; no Go platform primitive is added. |
| No polling fallback | Proof records `stateDelivery: trusted-shell.nats-watch.push` and `generatedIframePollingCount: 0`. |
| No reducer bypass | Winner and late-action denial are materialized by Tinkalet action apply/reject from watched action records. |
| No local-only proof | Demo requires a verified Tailscale route unless explicit local-only development opt-in is set. |

## RED Artifact

Current RED before this slice:

```bash
bun run demo:typerace
```

Expected failure: no package script or golden example existed. The existing
realtime proof was a profile/action accounting proof, not a browser typeracing
app.

## Acceptance Contract

`bun run demo:typerace` is GREEN only when it proves:

| Requirement | Evidence |
| --- | --- |
| Release-shaped package | Build release archive and use packaged `tinkabot` / `tinkalet`. |
| Anonymous scoped users | Start Tinkabot with opaque `anon-*` participants, not hard-coded Alice/Bob identities. |
| Browser app | Open two browser pages with `tb_type=1`, render a typeracing-only app, and avoid the generic proof shell as primary UI. |
| Shared NATS material | Race prompt, runners, progress, and result are material under one app-state item. |
| Reaction reducer | `tinkalet watch prefix` observes browser-origin action records and reducer resolves them through action apply/reject. |
| Realtime UI | Both pages receive pushed state through trusted-shell NATS watch delivery with zero generated-frame state polling. |
| Denials | Participant escape, duplicate join, duplicate action, stale revision, and late progress after finish are denied. |
| Authority | Proof has zero raw authority leaks and no typeracing-specific platform command or subject. |

Expected proof shape:

```json
{
  "kind": "tinkabot.anonymousTypeRaceProof.v1",
  "route": "tailscale",
  "stateDelivery": "trusted-shell.nats-watch.push",
  "generatedIframePollingCount": 0,
  "authorityLeakCount": 0,
  "platformTypeRaceAPIAdditions": 0,
  "pass": true
}
```

## GREEN Evidence

Implemented on 2026-06-25 as `bun run demo:typerace`. The demo builds the
release archive, starts packaged Tinkabot with two opaque participants, exposes
the trusted shell through Tailscale, removes the packaged NATS sidecar before
user-level commands, seeds race state, opens two browser pages, and drives
anonymous join/progress actions through the generated UI. The reducer watches
`apps.demo.participants` via packaged Tinkalet, applies legal progress, rejects
duplicate/stale/late actions, and materializes the final winner under the race
state item.

Latest proof:

```text
/tmp/tinkabot-typerace-demo.kzwmR5/typerace-anon-proof.json
shellUrl=http://forge.tail6c789a.ts.net:42299
kind=tinkabot.anonymousTypeRaceProof.v1
route=tailscale
anonymousUsers=anon-1782386213030972302-a,anon-1782386213030972302-b
winner=anon-1782386213030972302-a
denials=participant-escape,duplicate-join,duplicate-action,stale-revision,late-progress
reactionMode=tinkalet.watch.prefix
stateDelivery=trusted-shell.nats-watch.push
generatedIframePollingCount=0
stateVisibleP95Ms=66
stateVisibleP99Ms=66
authorityLeakCount=0
platformTypeRaceAPIAdditions=0
pass=true
```

Focused verification:

```bash
bash -n scripts/demo-typerace-anon.sh
bun test apps/frontend/tests/isolation.test.ts apps/frontend/tests/observe.test.ts
bunx @typescript/native-preview --noEmit -p apps/frontend/tsconfig.json
bun run --cwd apps/frontend build
bun run demo:typerace
```

## Held Manual Demo

Use:

```bash
TINKABOT_DEMO_HOLD=1 bun run demo:typerace
```

The held run creates a fresh manual race, starts the same Tinkalet reducer, smoke
tests the Tailscale links, then prints two anonymous runner links. Each link is
scoped to one opaque participant id and the same race state key.

## Residual Non-Claims

- Anonymous browser entry still uses startup-minted participant profiles; this
  is not a public signup or arbitrary lobby provisioning service.
- The demo proves a two-runner race, not a max participant or max-rate claim.
- The generated page submits through the trusted shell and receives pushed
  state messages; it still never receives raw NATS credentials or KV handles.
