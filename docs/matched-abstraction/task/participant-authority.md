---
layer: task
topic: participant-authority
references:
  - ../approach/tinkalet-edge.md
  - ../plan/tinkalet-edge.md
  - ../../../tasks/tinkabot-objective-okr.md
---

# Participant Authority

## Scope

CKR-AUTH proves the generic participant authority surface before any multiplayer
game work starts. The slice owns participant admit/revoke, participant profile
import, scoped NATS authority, and generated-frame mediation. It must not add a
tic-tac-toe, typeracing, or game-specific credential path.

## Authority Matrix

| Actor | Allowed | Forbidden | Downstream proof |
| --- | --- | --- | --- |
| Owner profile | Admit and revoke participants for an app; keep broad local-owner authority. | Hand owner `caller.creds` to participants. | Owner admit writes participant record and descriptor without credential leaks. |
| Participant profile | Connect with derived scoped JWT, read its app state and own action records for sync, and submit only its own app actions through the product action service. | Write/read another participant, another app, schedules, config, upload, bundle triggers, raw NATS subjects, broad KV reads, or direct item writes. | `participant_scope_denial_pass`, denied-neighbor/cross-scope tests. |
| Tinkalet | Import participant profiles, select them, and run product commands through the selected profile. | Treat participant as `local-owner` or silently fall back to stronger credentials. | Profile list/import tests and stale-credential denial. |
| Generated UI | Emit typed participant app intents through the trusted shell lease. | Receive credentials, subjects, tokens, KV names, store handles, or direct publish authority. | Frontend frame mediation tests. |
| Tinkabot substrate | Mint/revoke scoped JWTs, persist participant records, enforce NATS boundaries. | Create game-specific tokens or non-NATS state channels. | Real embedded NATS acceptance tests. |

## RED Acceptance

The RED tests must fail before implementation because no participant admit
surface exists yet.

| Requirement | RED proof |
| --- | --- |
| Owner admits participants | `TestParticipantAuthority` calls `AdmitParticipant` for Alice and Bob and imports both profiles with Tinkalet. |
| Participant scope is narrow | Alice can submit only Alice's app action through the product service; Alice-as-Bob and raw cross-participant publish are denied. |
| Revocation is live | Revoking Alice denies reconnect/action while Bob remains live. |
| Generated UI is mediated | Frontend isolation accepts matching app/participant intents and rejects wrong-app or wrong-participant intents. |

## Expected RED Failures

| Command | Expected failure before GREEN |
| --- | --- |
| `cd substrate/go && go test ./tinkabot -run TestParticipantAuthority -count=1` | Compile fails because `App.AdmitParticipant`, `App.RevokeParticipant`, and `ParticipantProfile` do not exist yet. |
| `cd apps/frontend && bun test tests/isolation.test.ts` | Runtime assertion fails because `accept()` does not yet bind `appId`/`participantId` into the mediated command context or deny mismatched app/participant IDs. |

`cd apps/frontend && bun run typecheck` is not the RED oracle for this slice;
it currently passes because the test fixture feeds participant scope through the
existing structural helper.

## GREEN Boundary

The smallest acceptable GREEN is a generic participant profile mechanism:
server-side admit/revoke, NATS-backed participant records, role/trust vocabulary
for Tinkalet profiles, and frame lease app/participant scope. Game mechanics,
turn rules, realtime rates, and visual submit bridges stay out of this task.

## GREEN Evidence

Implemented on 2026-06-24 as a CKR-AUTH substrate slice, not a complete
multiplayer mission. The mechanism is generic participant authority:

| Requirement | GREEN proof |
| --- | --- |
| Owner admits participants | `App.AdmitParticipant(appID, participantID)` mints derived scoped NATS credentials, writes a participant profile descriptor, and materializes `tinkabot.participant.v1` in `tb_items`. |
| Participant scope is narrow | Participant credentials can request only `tb.app.<app>.participants.<id>.action` for mutation and can read only scoped app state plus their own action records for sync; direct item writes, cross-participant action subjects, broad KV reads, and raw cross-scope `$KV.tb_items...` publishes are denied. |
| Revocation is live | `App.RevokeParticipant` revokes the NATS user, updates the local descriptor and NATS participant record, and Tinkalet reports `revoked-credentials`; repeated admit rotates the participant identity by revoking the previous scoped NATS user; another participant remains live. |
| Generated UI is mediated | Frame leases may bind `appId` and `participantId`; mismatched app or participant content intents fail with `FrameScopeEscape`. |

Verification:

```bash
cd substrate/go && go test ./tinkabot -run TestParticipantAuthority -count=1
cd substrate/go && go test ./tinkabot -run 'TestParticipantAuthority|TestLocalProfileDescriptor|TestTinkaletAuthorityDenials|TestTinkaletItemWatchUsesWatchStream' -count=1
cd substrate/go && go test ./tinkalet -count=1
cd substrate/go && go test ./... -count=1
bun test apps/frontend/tests/isolation.test.ts
bunx @typescript/native-preview --noEmit -p apps/frontend/tsconfig.json
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh lookup substrate/go/tinkabot/participant_authority_test.go
scripts/c3-line-coverage-harness.sh
C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh check --include-adr
```

Current C3 harness result: `owned_files=477`, `lookup_errors=0`,
`uncovered=0`.

Review closure:

| Review gap | Closure |
| --- | --- |
| Duplicate admit could leave an older scoped participant credential valid. | `AdmitParticipant` now revokes the prior recorded participant user before accepting the replacement profile; `TestParticipantAuthority` saves old creds and proves they fail direct NATS reconnect after rotation. |
| Revocation was mostly proven through Tinkalet's local descriptor path. | `TestParticipantAuthority` now also connects with revoked participant creds directly through `Runtime.ConnectCreds` and expects NATS denial. |
| Read/non-item denial was inferred from permissions. | The test now requires a NATS permission error for direct read of another participant record, and raw wrong-app/config/schedule/direct-action writes are denied with participant creds. |

Residual scope:

| Not covered here | Next owner |
| --- | --- |
| Turn rules, stale move semantics, and idempotent game actions | CKR-TURN |
| Browser DOM realtime sync and participant-rate limits | CKR-REALTIME |
| Visual submit-to-item bridge and scoped LLM watcher flow | CKR-VIS |
