---
layer: task
topic: realtime-participant-terminal-result
references:
  - ../../../tasks/tinkabot-objective-okr.md
  - ./realtime-participant-watch.md
  - ./realtime-participant-action-gap.md
  - ./realtime-participant-reconnect-restart.md
  - ./app-action-reducer-contract.md
---

# Realtime Participant Terminal Result

## Scope

CKR-REALTIME needs terminal-result proof before a realtime-heavy multiplayer
reference app can claim that final outcome materialization is authoritative.
This slice proves the generic app-action reducer path can produce terminal app
state as durable NATS-backed material, and participants can account for their
own terminal receipts through scoped filtered watches.

This is not a browser UI proof, not a typeracing API, not a max-rate freeze, and
not scoped multiplayer mission completion.

## Contract

```text
alice and bob scoped participant profiles
  -> submit app actions against apps.demo.state.terminal
  -> owner/reducer applies legal actions with NATS KV CAS
  -> owner/reducer rejects a late action after terminal state
  -> apps.demo.state.terminal contains the final result
  -> each participant watch observes own action receipts
```

The proof must fail if terminal state is only client-local, if the late action
mutates final state, if any accepted action lacks exactly one final receipt, if
participant watches require broad authority, or if a custom non-NATS replay path
is introduced.

## RED Boundary

- Use `tinkalet action submit`, `action apply`, `action reject`, `item get`, and
  `watch` over existing item/action surfaces.
- Keep result derivation as app/reducer logic over generic payloads.
- Add no game-specific platform subject, no scoring service, no browser claim,
  and no custom realtime event channel.
- Keep participant authority scoped to app state and own action subtree.

## GREEN Boundary

The slice is green only when `TestParticipantRealtimeTerminalResultMaterialization`
proves:

- Terminal result state is a `tinkabot.item.v1` record under `apps.demo.state`.
- The final state revision is observed by a scoped participant state watch.
- Each submitted participant action has a final receipt under that participant's
  own action subtree.
- A late action after terminal state is rejected with a durable denied receipt
  and does not mutate the terminal state.
- No raw bucket, `$KV`, credential, or broad authority detail leaks.

## RED Evidence

`go test ./tinkabot -run TestParticipantRealtimeTerminalResultMaterialization -count=1`
first failed at compile time because the acceptance test named the terminal
materialization helpers before they existed. That preserved the requirement
boundary before helper implementation.

## GREEN Evidence

- `go test ./tinkabot -run TestParticipantRealtimeTerminalResultMaterialization -count=1 -v`
- `go test ./tinkabot -run 'TestParticipantRealtimeTerminalResultMaterialization|TestParticipantRealtimeReconnectRestartCatchUp|TestParticipantRealtimeActionGapHarness|TestParticipantRealtimeWatchEnvelope|TestParticipantAppActions|TestParticipantAppReducer|TestTurnBasedReferenceMission|TestParticipantAuthority' -count=1`
- `go test ./tinkalet -run 'TestParticipantWatchScopeDenialPrecedesNetwork|TestParticipantWatchFiltersDenyMalformedTargets' -count=1`
- `go test ./... -count=1` from `substrate/go`
- `git diff --check`
- `scripts/c3-line-coverage-harness.sh` -> `owned_files=485`, `lookup_errors=0`, `uncovered=0`
- `C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-302`
- `C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh check --include-adr`

The implementation is test-harness-only for this slice: the terminal state and
winner are app payloads passed through the existing generic `action apply` /
`action reject` reducer path. No terminal-specific platform subject, scoring
service, browser credential, or custom realtime channel was added.

## Independent Review

- Codex noninteractive review: `VERDICT: PASS`; findings none; no blocking
  gaps. It independently reran the focused terminal proof, participant
  watch-denial tests, `c3-302` eval/check, and whitespace check.
- Claude noninteractive review: `VERDICT: PASS`; findings none; gaps none. It
  confirmed terminal state durability, scoped state/action watches, late-action
  rejection without mutation, absence of new platform primitive/custom channel,
  and no mission count by itself.

## Non-Goals

- No browser direct-NATS or generated UI credential claim.
- No max-rate or max-participant freeze.
- No app-specific scoring primitive.
- No complete realtime mission claim until realtime-heavy UI and final mission
  flow are proven together.
