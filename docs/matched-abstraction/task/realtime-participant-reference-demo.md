---
layer: task
topic: realtime-participant-reference-demo
references:
  - ../../../tasks/tinkabot-objective-okr.md
  - ./realtime-participant-action-gap.md
  - ./realtime-participant-sustained-rate.md
  - ./realtime-participant-terminal-result.md
  - ./turn-based-reference.md
---

# Realtime Participant Reference Demo

## Scope

CKR-REALTIME needs a release-shaped proof that intersects the existing package
demo path with scoped participant realtime action accounting and terminal result
materialization. This slice adds `bun run demo:realtime`: it builds the release
archive, starts packaged Tinkabot with `--participant demo:alice` and
`--participant demo:bob`, disables the packaged NATS sidecar before user-level
commands, then drives all actions through packaged Tinkalet profiles.

This is not a browser-originated participant-action bridge, not a direct-NATS
client harness, not a max-rate freeze, and not scoped multiplayer mission
completion.

## Contract

```text
release package
  -> packaged Tinkabot with alice/bob participant profiles
  -> packaged Tinkalet submits rate actions through scoped participant profiles
  -> participant filtered watches account expected action records
  -> packaged Tinkalet submits terminal actions
  -> owner/reducer applies legal terminal actions and rejects late action
  -> proof JSON records action accounting, rate, terminal result, and denials
```

The proof must fail if actions bypass Tinkalet profiles, if any expected action
is missing from the participant's own filtered watch, if terminal state is not
NATS-backed material, if a late action mutates final state, if neighbor action
watching succeeds, or if output leaks raw bucket, `$KV`, credential, or NATS URL
details.

## RED Boundary

- Public command: `bun run demo:realtime`.
- Expected RED before implementation: `error: Script not found "demo:realtime"`.
- Do not add a browser-submit claim in this slice.
- Do not add a typeracing-specific platform subject, score service, or realtime
  channel.
- Do not use the packaged NATS CLI sidecar for the user-level proof after
  profile import.

## GREEN Boundary

The slice is green only when `bun run demo:realtime` writes
`realtime-reference-proof.json` with:

- `participants_started = 2`.
- `expected_actions == observed_actions`.
- `revision_gap_count = 0`, where the count means missing expected action ids
  or non-increasing own-watch revisions rather than contiguous global KV
  sequence gaps.
- `participant_rate_hz_per_participant >= 10`.
- `terminal_event_loss = 0`.
- `authority_violation_count = 0`.
- `raw_authority_leak_count = 0`.
- final winner and final state revision from `apps.demo.state.terminal`.
- late action rejected as `race-finished` without state mutation.

## RED Evidence

`bun run demo:realtime` failed first with:

```text
error: Script not found "demo:realtime"
```

## GREEN Evidence

- RED: `bun run demo:realtime` first failed with
  `error: Script not found "demo:realtime"`.
- Harness fix: the first implementation run exposed an inline Node module-shape
  issue because the repo is ESM and the proof mixed `require()` with top-level
  `await`; switching the inline proof to ESM imports fixed the harness without
  changing product behavior.
- `bun run demo:realtime` -> Tailscale URL
  `http://forge.tail6c789a.ts.net:39893`, proof
  `/tmp/tinkabot-realtime-demo.zJPQCA/realtime-reference-proof.json`.
- Latest proof metrics: `participants_started=2`, `expected_actions=60`,
  `observed_actions=60`, `revision_gap_count=0`,
  `participant_rate_hz_per_participant=50.59`, `terminal_event_loss=0`,
  `authority_violation_count=0`, `raw_authority_leak_count=0`, final winner
  `alice`, final state revision `76`.
  The proof also records the observed action id set per participant.
- `bash -n scripts/demo-realtime-participant.sh scripts/demo-turn-based.sh scripts/demo-chain-reaction.sh scripts/demo-live-patch.sh scripts/release-package.sh scripts/package-tinkabot.sh`
- `go test ./tinkabot -run 'TestParticipantRealtimeTerminalResultMaterialization|TestParticipantRealtimeActionGapHarness|TestParticipantRealtimeWatchEnvelope|TestParticipantAppActions|TestParticipantAppReducer|TestTurnBasedReferenceMission|TestParticipantAuthority' -count=1`
- `go test ./cmd/tinkabot -run 'TestRunStartsPrintsPostureAndStopsOnSignal|TestRunAdmitsParticipantsFromStartupFlag|TestRunRequiresStore|TestRunPrintsVersion' -count=1`
- `go test ./tinkalet -run 'TestParticipantWatchScopeDenialPrecedesNetwork|TestParticipantWatchFiltersDenyMalformedTargets' -count=1`
- `go test ./... -count=1` from `substrate/go`
- `git diff --check`
- `scripts/c3-line-coverage-harness.sh` -> `owned_files=488`,
  `lookup_errors=0`, `uncovered=0`
- `C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-302`
- `C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-501`
- `C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh check --include-adr`

## Independent Review

- Codex initial noninteractive review: `VERDICT: PASS`, no blocking
  findings. The review independently reran `bun run demo:realtime` and
  identified hardening gaps in proof observability and wording: record the
  watched action id set, clarify `revision_gap_count`, and keep the leak claim
  scoped to product JSON/user-level output rather than daemon debug logs.
- Claude initial noninteractive review: `VERDICT: PASS`, no blocking
  findings. It did not rerun the live demo, but it agreed the slice was scoped
  to packaged realtime substrate proof rather than browser-originated
  participant UI.
- Follow-up hardening recorded `observed_action_ids`, clarified
  `revision_gap_count_kind`, and redacted fail-path `tinkabot.log` /
  `tinkabot.err` dumps for NATS URLs, credential paths, bucket names, `$KV`,
  and key markers without changing the happy-path proof.
- Codex follow-up and final redaction-only reviews: `VERDICT: PASS`, no
  confirmed issues. The final redaction-only review did not rerun the full demo
  by scope; it verified `bash -n` and that proof gates still read the JSON proof
  directly.
- Claude follow-up and final redaction-only reviews: `VERDICT: PASS`. Claude
  noted the fail-path redaction is best-effort for abnormal bare key-body dumps,
  which stays outside this slice's product-output leak claim.

## Non-Goals

- No browser-generated participant action submit.
- No direct browser NATS credential.
- No max participant or max-rate threshold.
- No example-specific platform primitive.
- No complete scoped multiplayer mission claim until the browser/UI path and
  final reference mission are proven together.
