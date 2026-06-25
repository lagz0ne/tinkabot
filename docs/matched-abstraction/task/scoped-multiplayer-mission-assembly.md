---
layer: task
topic: scoped-multiplayer-mission-assembly
references:
  - ../../../tasks/tinkabot-objective-okr.md
  - ./participant-authority.md
  - ./turn-based-reference.md
  - ./realtime-participant-reference-demo.md
  - ./browser-participant-action-bridge.md
  - ./realtime-browser-participant-ui.md
---

# Scoped Multiplayer Mission Assembly

## Scope

This task records the claim audit that lets the scoped multiplayer mission
count as complete. No new platform mechanism is introduced here. The claim is
assembled from already-green CKR-AUTH, CKR-TURN, and CKR-REALTIME proofs.

The counted mission is intentionally bounded: two scoped participants, one
turn-based reference path, one measured realtime-heavy reference path, trusted
shell mediated generated UI, durable NATS-backed material, and zero authority
leaks in product proof output. It does not freeze a max participant count, max
rate, arbitrary browser publisher identity, or a typeracing-specific platform
API.

## Row Audit

| Row | Requirement | Evidence |
| --- | --- | --- |
| M1 | Owner can admit multiple participants to the same app with derived scoped profiles. | CKR-AUTH proves admit/revoke, duplicate-admit rotation, scoped profile import, denied-neighbor access, revoked credential denial, and app/participant frame leases. Packaged turn and realtime demos start Tinkabot with `demo:alice` and `demo:bob`. |
| M2 | Shared app state is NATS-native material, not process/browser-local truth. | CKR-TURN stores shared state and receipts through `tb_items`; CKR-REALTIME materializes action records and terminal state under `apps.demo.state.terminal`; reconnect/restart proof shows cursor catch-up over retained NATS material. |
| M3 | Turn-based example enforces turn order and legal moves. | `demo:turn` and `TestTurnBasedReferenceMission` prove legal completion plus wrong-turn, stale, duplicate, and occupied-cell denial through generic action apply/reject. |
| M4 | Realtime-heavy example uses a measured NATS-native sync path. | `demo:realtime` proves 60/60 observed scoped actions at `50.59Hz` per participant with zero missing-id/order gaps; the browser UI proof adds Tailscale generated-frame action/readback latency p95 under threshold. |
| M5 | Realtime result/scoring is derived from authoritative material/projections. | Terminal-result proof and `demo:realtime` materialize final state under `apps.demo.state.terminal`, reject late action as `race-finished`, record `terminal_event_loss=0`, and keep winner/final revision in proof JSON. |
| M6 | Generated multiplayer UI remains mediated by shell/profile surfaces only. | Browser command bridge and browser UI proof route generated-frame `participant_action` / `participant_read` through the trusted shell and `tb.app.browser.command`, with no raw NATS credential in generated content and zero authority leaks. |

## Claim Boundary

- Counted: scoped multiplayer mission family, `1 / 2` objective progress.
- Counted by assembly: CKR-AUTH plus CKR-TURN plus CKR-REALTIME evidence.
- Not counted: max participants, max rate, arbitrary browser publisher identity,
  direct generated-frame NATS, client-owned scoring, custom realtime channel, or
  example-specific platform primitive.

## Verification

- Read-only DKR audit returned `PASS_TO_COUNT` for rows M1-M6 and identified no
  smallest missing CKR.
- Audit checks included `c3 check --include-adr` -> `total: 28`, `ok: true`,
  and shell syntax for the turn, realtime participant, and realtime browser demo
  scripts.
