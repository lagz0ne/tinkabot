---
layer: task
topic: browser-participant-action-bridge
references:
  - ../../../tasks/tinkabot-objective-okr.md
  - ./realtime-participant-reference-demo.md
  - ./realtime-participant-watch.md
  - ./app-action-revision-contract.md
---

# Browser Participant Action Bridge

## Scope

CKR-REALTIME needs the missing bridge between generated browser UI and the
generic participant action substrate. Existing frontend isolation can turn a
leased generated-frame message into `browser.command_intent`, and existing
Tinkabot app-action service can materialize participant actions through
`tb.app.<app>.participants.<id>.action`. This slice connects those two
surfaces without giving generated UI raw NATS authority.

This is not a typeracing platform API, not a direct browser NATS credential,
not latest-projection polling as event proof, and not scoped multiplayer
mission completion.

## Contract

```text
generated browser content
  -> content.intent command=participant_action
  -> trusted shell stamps app/participant lease context
  -> browser.command_intent passes canonical schema with app/participant context
  -> Tinkabot-owned browser command handler validates context and payload
  -> existing participant action service materializes the action item
  -> browser path can read back scoped state/action material without raw authority
```

The bridge must fail if app or participant context is missing, if payload tries
to choose another app/participant, if the command is not `participant_action`,
if the payload is malformed, if stale/duplicate app-action checks fail, or if
raw bucket, `$KV`, credential, token, subject, publish, or subscribe details
leak through the product response.

## RED Boundary

- Public command path: request/reply on `tb.app.browser.command` with a
  `browser.command_intent` payload.
- Expected RED before implementation: request fails because Tinkabot has no
  product subscriber for `tb.app.browser.command`.
- Schema RED: canonical `browser.command_intent.context` rejects
  `appId`/`participantId`.
- Do not make the frontend test alone count as GREEN; durable NATS-backed item
  materialization is required.

## GREEN Boundary

The slice is green only when:

- Canonical schema and SDK parse `browser.command_intent.context.appId` and
  `participantId` as optional trusted-shell-stamped fields.
- A browser command with `command=participant_action`,
  `context.appId=demo`, `context.participantId=alice`, and generic payload
  `{actionId,stateKey,baseRevision,value}` creates the same pending action item
  shape as Tinkalet app-action submit.
- Duplicate, stale, malformed, wrong-command, missing-context, wrong-app, and
  wrong-participant browser commands return denials without materializing an
  action.
- Browser readback can fetch the app state and the participant's own action
  item, while neighbor action readback is denied.
- Trusted-shell context must include session, capability, artifact, frame,
  revision, chain, app, and participant fields.
- Product responses do not expose raw NATS authority.

## RED Evidence

- `bun test packages/sdk/tests/base-contract/command-acceptance.test.ts -t T-CMD-PARTICIPANT-CONTEXT` -> RED:
  `TinkabotRuntimeError: Contract input is invalid`, proving canonical
  `browser.command_intent.context` rejected `appId` / `participantId`.
- `cd substrate/go && go test ./tinkabot -run TestBrowserParticipantActionBridge -count=1` -> RED:
  `nats: no responders available for request`, proving Tinkabot had no product
  responder for `tb.app.browser.command`.

## GREEN Evidence

- Schema/SDK GREEN:
  `bun test packages/sdk/tests/base-contract/command-acceptance.test.ts -t T-CMD-PARTICIPANT-CONTEXT`
  -> `1 pass`, proving canonical `browser.command_intent` preserves
  trusted-shell-stamped `context.appId` and `context.participantId`.
- Frontend lease GREEN:
  `bun test apps/frontend/tests/isolation.test.ts -t participant` -> `1 pass`,
  proving generated-frame `participant_action` intents must match the leased
  app and participant.
- Frontend raw-authority follow-up:
  `bun test apps/frontend/tests/isolation.test.ts` -> `7 pass`, proving the
  trusted shell denies raw-authority keys including `password`.
- Product bridge GREEN:
  `cd substrate/go && go test ./tinkabot -run TestBrowserParticipantActionBridge -count=1`
  -> `ok`. The test mints a browser-style bearer publisher with only
  `tb.app.browser.command`, submits `participant_action`, verifies the existing
  app-action service materializes `apps.demo.participants.alice.actions.browser-1`,
  reads action and state material back through `participant_read`, denies
  neighbor action readback, denies duplicate action, denies stale revision
  without materializing the stale action, denies payload participant escape,
  denies raw-authority payload keys, denies unknown commands, and denies missing
  trusted-shell context.
- Broader focused proof:
  `cd substrate/go && go test ./tinkabot -run 'TestBrowserParticipantActionBridge|TestAppActionMalformedSubject|TestParticipantAppActions|TestParticipantAppReducer|TestParticipantRealtimeWatchEnvelope|TestParticipantRealtimeActionGapHarness|TestParticipantRealtimeTerminalResultMaterialization|TestParticipantAuthority' -count=1`
  -> `ok`.
- Package freshness proof:
  `bun run build:frontend` -> Vite build wrote embedded shell assets under
  `substrate/go/frontend/site`; `bun run pack:tinkabot -- /tmp/tinkabot-pack-bridge-rawfix`
  -> package build passed and ran the frontend build before Go binaries.
- Full Go proof:
  `cd substrate/go && go test ./... -count=1` -> all packages passed.
- Narrow JS proof:
  `bun test packages/sdk/tests/base-contract/contract-authority.test.ts packages/sdk/tests/base-contract/command-acceptance.test.ts apps/frontend/tests/isolation.test.ts apps/frontend/tests/observe.test.ts`
  -> `22 pass`, `0 fail`.
- C3 coverage proof:
  `scripts/c3-line-coverage-harness.sh` -> `owned_files=489`,
  `lookup_errors=0`, `uncovered=0`.
- `git diff --check` -> clean.

## Independent Review

- Codex noninteractive review:
  `/tmp/tinkabot-codex-browser-bridge-review.txt` -> `VERDICT: PASS`, no
  blocking findings. It recorded one explicit gap: `tb.app.browser.command`
  trusts the trusted-shell surface and does not independently bind arbitrary
  browser publishers to participant identity.
- Claude noninteractive review:
  `/tmp/tinkabot-claude-browser-bridge-review.txt` -> `VERDICT: PASS`. It
  found raw-authority denylist drift across layers as a non-blocking
  defense-in-depth gap.
- Raw-authority follow-up fixed the drift by aligning the frontend, Go bridge,
  and rebuilt generated shell asset on the credential vocabulary including
  `password`, `bearer`, `cred`, `jwt`, `nkey`, `seed`, and `secret`.
  Codex follow-up `/tmp/tinkabot-codex-browser-bridge-rawfix-review.txt` and
  Claude follow-up `/tmp/tinkabot-claude-browser-bridge-rawfix-review.txt` both
  returned `VERDICT: PASS`.

## Residuals

- The browser command route is a trusted-shell-mediated surface; it still does
  not prove arbitrary browser publishers can safely self-assert app or
  participant identity.
- Frontend and Go raw-authority filters match exact normalized field names; the
  SDK contract filter remains stricter because it catches substring matches.

Blocked checks:

- `bun test apps/frontend/tests` runs the non-browser tests but the
  service-worker browser test fails because `/usr/bin/google-chrome` is absent
  in this environment.
- `bun run --cwd packages/sdk typecheck` is blocked by the local file dependency
  `@lagz0ne/nats-embedded` missing built `dist` types; attempting to build that
  adjacent repo failed because its `tsdown` dev tool is not installed there.

## Non-Goals

- No direct browser NATS credential for generated content.
- No direct browser or handler write to `tb_items`; action creation goes
  through the existing app-action service path.
- No claim that arbitrary browser publishers can self-assert participant
  identity safely; this bridge is for trusted-shell-mediated command intents.
- No max-rate or max-participant freeze.
- No typeracing, tic-tac-toe, board, score, move, or cell platform primitive.
- No scoped multiplayer mission completion claim until the realtime-heavy UI
  reference flow is proven end to end.
