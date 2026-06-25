# Tinkabot Objective OKR

Frame state: human-ratified by user request on 2026-06-24.

## Objective

Make Tinkabot a NATS-native app substrate where an LLM or user can assemble a
complete interactive app as a bundle: UI, Tinkalet integration, scoped
authority, durable material, and reactive updates all flow through NATS
primitives.

Metric: `complete_reference_missions / target_reference_missions`.

Current progress: `2 / 2`. The scoped multiplayer mission is counted from the
assembled CKR-AUTH, CKR-TURN, and CKR-REALTIME proofs; the LLM visualization
mission is counted from the CKR-VIS release-shaped visual decision proof.

Target: `2 / 2` reference mission families complete with `0` anti-goal
tripwire breaches:

| Mission family | Counts only when |
| --- | --- |
| LLM visualization loop | An LLM can publish or build a visual bundle, the user can interact with it through Tinkabot UI, the submitted choice is stored as NATS-backed material, and a scoped LLM watcher can observe that result to steer the next turn. |
| Scoped multiplayer loop | Multiple users can join the same app with derived scoped credentials, share state through NATS-native material, and complete both a turn-based path and a measured realtime-heavy path without shared broad authority. |

The examples prove the substrate; they are not the product boundary. No example
is allowed to introduce a platform mechanism that does not generalize back to
the bundle, Tinkalet profile, authority, material, or NATS reaction spine.

## Anti-Goals

| Anti-goal | Metric | Type | Tripwire |
| --- | --- | --- | --- |
| Do not compromise security | `security_tripwire_breaches` | tripwire | Any generated UI, generated script, or user profile receives raw broad authority, can cross another user or bundle boundary, or can bypass the materializer/command gate. |
| Not an MVP claim | `partial_usecases_claimed_complete` | tripwire | Any mission marked complete without release-shaped outside-in proof, user-visible flow, docs, and failure behavior. |
| No over-abstraction | `example_specific_platform_mechanisms` | tripwire | Any new platform primitive exists only for Mermaid, tic-tac-toe, or typeracing instead of the generic substrate mechanism. |
| Double V only | `code_derived_acceptance_tests` | tripwire | Acceptance tests are written from implementation details instead of requirement rows and user-visible outcomes. |
| Platform stays generic | `usecase_owned_platform_exceptions` | tripwire | A use case gets a privileged path unavailable to other bundles/profiles. |
| NATS primitives first | `non_nats_state_or_event_channels` | tripwire | State, events, identity, watch, schedule, or reaction flow bypasses NATS primitives without a recorded DKR and human-ratified exception. |

Anti-goal coverage review:

| Harm considered | Selected guardrail | Rejected guardrail | Owner | Cadence |
| --- | --- | --- | --- | --- |
| Generated code gains authority because it is convenient. | Shadow authority remains enforced: generated code receives mediated frame/profile/material surfaces only. | Give bundles raw NATS credentials and rely on prompt discipline. | Orchestrator plus security reviewer | Before every CKR implementation and after every proof. |
| Examples become the platform. | Every DKR must map the example back to a shared bundle/Tinkalet/NATS mechanism. | Add one-off example APIs and clean them later. | Orchestrator | Every DKR return and Plan update. |
| Tests rubber-stamp current code. | Requirement rows are written before implementation tasks; code-level tests must cite the requirement row they fulfill. | Derive acceptance coverage from existing methods or files. | Worker author, checked by orchestrator | Before each progression worker starts. |
| Realtime path invents a custom engine. | Realtime work starts as DKR until the NATS primitive envelope and latency target are measured. | Promise typeracing delivery before the substrate limit is known. | Orchestrator | During realtime DKR and before CKR freeze. |
| The loop claims progress from task completion. | Direct mission metrics are read from release-shaped demos/tests, not from completed task count. | Treat docs, code, or test presence as done. | Orchestrator | End of each loop. |

## Action Envelope

Allowed moves: repo-local OKR/task docs, matched-abstraction Approach/Plan/Task
artifacts, C3 queries/checks/eval specs, RED acceptance tests, focused
substrate/frontend/example changes that implement an admitted CKR, release-shaped
demo scripts, Tailscale-reachable demo proofs, and noninteractive Codex/Claude
review for substantial claims.

Forbidden moves: destructive git operations, public releases or package
publishes, relaxing anti-goals, bypassing C3 for architecture claims, exposing
raw broad credentials to generated code, relying on localhost-only demos for
user-facing proof, adding non-NATS side channels for state or event flow, or
turning an execution task into silent discovery.

Approval gates: any goal or anti-goal change, any security posture expansion,
any network/public exposure change beyond the current loopback-plus-Tailscale
demo posture, any irreversible external action, and any decision to drop one of
the named mission families.

## Decomposition

DKR is the tool for structuring CKR. It is not delivery, and it does not count
toward the objective except by returning measurable CKRs or returning empty with
evidence.

### DKRs

| DKR | Budget | Learning output |
| --- | --- | --- |
| DKR-0 mission contract map | One focused worker pass | Done: requirement rows and acceptance metric candidates are recorded below; recommended CKR split is now reflected in the CKR candidates table. |
| DKR-1 visualization primitive map | One focused worker pass | Done: current bundle/material/item/watch surfaces, missing submit bridge, watch-profile gap, and CKR-VIS metrics are recorded below. |
| DKR-2 participant profile authority | One focused worker pass | Done: current owner/profile/JWT/frame surfaces, gaps, acceptance metrics, and CKR-AUTH sequence are recorded below. |
| DKR-3 realtime envelope | One measured probe pass | Done: current clock/projection path measured live; provisional CKR-REALTIME admission target and required pre-typeracing measurements are recorded below. |
| DKR-4 genericity pressure test | One review pass after DKR-1 through DKR-3 | Done: mechanism classification, metric freeze status, tripwire review, and next progression task are recorded below. |

### DKR-0 Requirement Rows

Discovery worker: `019ef7a4-06ae-7981-bb9a-2c6fbc515145`.
Scope: no file edits; structure CKR candidates only.

LLM visualization mission:

| Row | Requirement | Acceptance metric candidates | Tripwires |
| --- | --- | --- | --- |
| V1 | LLM or user can publish or build a visual bundle, with Mermaid only as an example. | `visual_bundle_render_pass=1`; browser renders a nonblank artifact from bundle material; artifact is served through the Tinkabot shell. | `ABS/GEN`: Mermaid-only platform path. `SEC`: generated UI receives raw credentials. |
| V2 | Generated page exposes selectable items and an action submit path. | `visual_action_submit_pass=1`; valid submit accepted; malformed, duplicate, and stale submits denied. | `SEC`: direct KV/NATS write from generated UI. `DV`: tests assert DOM details instead of user action result. |
| V3 | Submitted choice becomes durable NATS-backed material with value, revision, provenance, and status. | `visual_result_materialized_pass=1`; item/projection read shows submitted value plus revision/provenance; restart does not erase truth. | `NATS`: hidden file/in-memory state. `DONE`: no failure behavior for stale or denied writes. |
| V4 | A scoped LLM/agent watcher can observe the result and use it to steer the next turn. | `llm_watch_result_pass=1`; scoped watcher sees result; revoked/neighbor watcher denied; no operator credential in happy path. | `SEC`: LLM gets broad credentials. `NATS`: watcher bypasses product watch/item surface. |
| V5 | Non-directly-consumable content can be transformed through the reaction chain into a consumable artifact/projection. | `visual_transform_chain_pass=1`; source to transformer to materialized result to UI update is visible through NATS-backed material. | `ABS`: special transformer API only for diagrams. `NATS`: local callback is the source of truth. |

Scoped multiplayer mission:

| Row | Requirement | Acceptance metric candidates | Tripwires |
| --- | --- | --- | --- |
| M1 | Owner can admit multiple participants to the same app with derived scoped profiles. | `participant_profile_mint_pass=1`; two user profiles connect; cross-user action denied; revocation takes effect. | `SEC`: shared initial credential. `GEN`: game-specific credential path. |
| M2 | Shared app state is NATS-native material, not process/browser-local truth. | `multiplayer_state_material_pass=1`; two clients observe the same revision; restart/reconnect preserves state. | `NATS`: websocket/game server owns truth outside NATS. |
| M3 | Turn-based example, such as tic-tac-toe, enforces turn order and legal moves. | `turn_based_acceptance_pass=1`; legal game completes; wrong-turn, stale-revision, duplicate, and occupied-cell moves denied. | `SEC`: client-only enforcement. `DONE`: happy path without denial matrix. |
| M4 | Realtime-heavy example, such as typeracing, uses a measured NATS-native sync path. | Candidate after DKR-3: `realtime_sync_p95_ms`, `event_loss_rate`, `max_participants_at_rate`, `authority_violation_count=0`. | `NATS`: custom realtime channel bypass. `DONE`: no measured threshold. |
| M5 | Realtime result/scoring is derived from authoritative material/projections. | `race_result_consistency_pass=1`; participants agree on progress/order; reconnect does not fork result. | `SEC`: trusting client final state. `DV`: tests mirror implementation buffers instead of race outcome. |
| M6 | Generated multiplayer UI remains mediated by shell/profile surfaces only. | `frame_intent_mediation_pass=1`; generated UI submits intents, not raw subjects/stores; denied raw authority words stay denied. | `SEC`: raw authority leaked to generated frame. `ABS`: UI-specific escape hatch. |

### DKR-1 Visualization Primitive Map Return

Discovery worker: `019ef7a8-fc9d-71e1-b217-8434a7e02bb3`.
Scope: read-only; structure CKR-VIS only.

Current usable surfaces:

| Row | Existing support | Boundary |
| --- | --- | --- |
| V1 visual artifact publish/render | Bundle entries derive trigger subjects, projection ids, and artifact prefixes; effects land in NATS-backed material stores; shell HTTP readback serves artifacts, `_p` projections, and projections. | Existing bundles satisfy included visual apps. Dynamic LLM publication of arbitrary new visual bundles is not yet an explicit product surface. |
| V2 user action submit | Trusted shell isolation accepts leased `content.intent` messages and denies raw authority vocabulary; browser command acceptance has real NATS proof; Tinkalet trigger maps product intents to hidden bundle subjects. | Missing generic submit bridge from sandboxed artifact to durable item result. Generated code must submit typed intent through the trusted shell/acceptance path, never raw NATS or `tb_items`. |
| V3 durable result storage | App-plane item storage exists in JetStream KV `tb_items`; Tinkalet item create/get/resolve/wait writes `tinkabot.item.v1` records with revision/provenance; restart durability is tested. | Visual choices should land in `tb_items`; bundle material is suitable for projections/artifacts but is process-ephemeral by design. |
| V4 LLM watch/readback | Tinkalet `watch item/prefix` watches `tb_items` with cursor persistence, replay, restart catch-up, and stale cursor denial. | CKR-VIS needs a release-shaped scoped LLM/watch profile or equivalent readback path; browser session observation is not the right product surface. |
| V5 transform/render chain | Bundle `watches` observe KV material and run filters through the same materializer gate; clock/builder examples and TransformPipe tests prove source-to-projection/artifact chains. | Clear enough. Mermaid remains one artifact format, not a platform primitive. |

Missing or ambiguous before CKR-VIS:

| Gap | Why it matters |
| --- | --- |
| Generic visual submit bridge | Command acceptance and item storage both exist, but no product surface turns sandboxed UI choice into `tb_items` result. |
| Release-shaped LLM watcher profile | Tests prove scoped watch credentials, but the product path for an LLM watching a result is not explicit. |
| Dynamic visual bundle publication | Existing bundle startup/build surfaces are enough for included bundles; arbitrary LLM-published visual code needs a later DKR if it becomes part of CKR-VIS. |

Recommended CKR-VIS acceptance metrics:

| Metric | Target |
| --- | --- |
| `V1_rendered_artifact` | Visual bundle produces NATS-backed projection plus Object Store artifact; browser renders nonblank from `/artifacts/...`. |
| `V2_action_submit` | Sandboxed UI submits typed `visual.choice.submit` through shell command acceptance; malformed, duplicate, stale, and raw-authority payloads are denied. |
| `V3_restart_durable` | Accepted choice survives Tinkabot restart in `tb_items` with status, value, revision, and provenance. |
| `V4_llm_watch` | Scoped watcher sees the result via Tinkalet watch JSON, cannot direct-read unauthorized items, and fails cleanly on revoked/stale credentials. |
| `V5_transform_chain` | Source material changes drive a watched transform into updated projection/artifact without direct UI/NATS authority. |
| `CKR_VIS_green` | V1-V5 pass in one release-shaped scenario with no raw credential, subject, `$KV`, bearer token, or `tb_items` leakage. |

Recommended CKR-VIS sequence:

| Step | Purpose |
| --- | --- |
| Write one outside-in RED scenario | Cover V1-V5 together so rendering alone cannot count as mission completion. |
| Use generic bundle transform pattern | Prove visual artifact transformation without Mermaid-specific platform behavior. |
| Add command-to-item bridge | Turn typed browser intent into durable `tb_items` record through trusted acceptance. |
| Expose scoped LLM watcher/readback | Use Tinkalet watch/profile surfaces, not browser session observation. |
| Prove denial and durability | Restart durability, duplicate/stale denial, raw authority denial, and watcher scope denial. |

### DKR-2 Participant Profile Authority Return

Discovery worker: `019ef7a9-2eb0-75b2-a877-45762290b022`.
Scope: read-only; structure CKR-AUTH only.

Current usable surfaces:

| Surface | Existing support | Notes |
| --- | --- | --- |
| Owner profile | Tinkabot writes `local-profile.json`; Tinkalet imports it, stores profile-local credentials, redacts list output, and uses the selected profile for trigger/item/watch/schedule flows. | Current public descriptor is owner/caller shaped. |
| Scoped credential primitive | `Runtime.MintUser` can mint JWT credentials with capability provenance, TTL, scoped publish/subscribe permissions, and revocation that disconnects live clients and denies reconnect. | This is the correct NATS-first primitive for participants. |
| Account boundary primitive | Runtime account mint plus service import/export already express app/bundle boundaries. | Avoid game-specific token systems. |
| Existing narrow-role tests | Watch, schedule, and reaction tests prove narrower profile shapes in pieces. | They still masquerade as `role=caller` / `trust=local-owner`; participant role vocabulary is missing. |
| Generated UI mediation | Trusted shell isolation denies raw authority words and wraps accepted frame intents with lease/revision context. | Needs participant/app action lease, not raw JWTs in the frame. |
| Browser viewer pattern | Existing session viewer grant uses bearer-only leaf-scoped JWT and revocation without exposing seed material. | Useful precedent for browser participant grants. |

Gaps before CKR-AUTH can start:

| Gap | Why it matters |
| --- | --- |
| Owner admits participant | No product API or profile descriptor exists for "owner admits participant to this app". |
| Participant record store | Need NATS-backed material tying app/bundle id, participant id, JWT user pub, lease id, profile name, status, and revocation audit. |
| Participant role scope | Current `caller` role is too broad for participants because it can touch trigger/config/item/schedule/upload surfaces. |
| Two-participant denied-neighbor proof | Neighbor denial exists in pieces, but not for two participants in the same app. |
| Participant frame lease | Generated UI has generic frame denial, but no app/participant action lease for typed multiplayer intents. |

Recommended CKR-AUTH acceptance metrics:

| Metric | Target |
| --- | --- |
| `participant_profile_mint_pass` | Owner admits `alice` and `bob`; both import profiles and connect over real embedded NATS with derived credentials; no shared `caller.creds`. |
| `participant_scope_denial_pass` | Alice's valid action is accepted; Alice-as-Bob, Bob-private read/subscribe, wrong app/bundle, and raw subject attempts are denied. |
| `participant_revocation_pass` | Revoking Alice disconnects active clients and denies reconnect/action; Bob and owner remain live. |
| `frame_intent_mediation_pass` | Generated UI sends typed content intents only; raw authority words, stale revision, wrong participant/app/session, and unlisted commands are denied. |
| `participant_authority_acceptance_pass` | M1 and M6 are green over real NATS with `authority_violation_count=0`. |

Recommended CKR-AUTH sequence:

| Step | Purpose |
| --- | --- |
| Freeze authority matrix | Owner, participant, generated UI, Tinkalet, shell, and app/bundle account allowed/forbidden actions. |
| Write RED CKR-AUTH tests | Mint, denied-neighbor, revocation, and frame mediation before implementation. |
| Add participant admit/revoke service | Use `Runtime.MintUser`, capability tags, NATS-backed participant records, and import/export where account boundaries matter. |
| Extend Tinkalet profiles | Represent participant roles/trust without leaking raw authority. |
| Extend shell/frame lease context | Generated UI emits typed participant app intents only. |
| Prove generic app action path | Establish participant authority before tic-tac-toe or typeracing delivery. |

### DKR-3 Realtime Envelope Return

Discovery worker: `019ef7b3-ddc7-7510-8f51-7fef3a68a61b`.
Scope: no file edits; live measurement allowed; cleanup required.

Live command:

```bash
TINKABOT_DEMO_HOLD=1 bun run demo:chain
```

Observed: demo passed, retuned `bundle.clock.tick.every` to `100ms`, disabled
the packaged NATS sidecar as `nats.disabled`, exposed
`http://forge.tail6c789a.ts.net:44265/artifacts/bundle/clock/index.html`, then
was stopped. Cleanup was verified by a failed local `_p/view` curl after stop
and no bracket-matched held `tinkabot` or `socat` process.

Original measurement limitation: browser DOM measurement through Playwright was
blocked by `ERR_MODULE_NOT_FOUND: Cannot find package 'playwright'`, so DKR-3
first returned projection fetch measurements only. The browser DOM sync
measurement prerequisite is now green for the existing clock chain through
`apps/frontend/node_modules/playwright`; participant-rate, revision-gap,
reconnect/restart, and terminal-result prerequisites are now green as separate
proofs. Realtime-heavy participant UI measurement remains separate.

Observed projection fetch envelope:

| Route | Samples | Unique seqs | Duration | `ageMs` | `fetchMs` | `sourceIntervalMs` | `filterLatencyMs` |
| --- | ---: | ---: | --- | --- | --- | --- | --- |
| loopback `_p/view` every 50ms | 193 | 98 | 10s, `seqSpanMs=9927` | p50 70, p95 128, p99 142, max 145 | p95 3, max 60 | p95 129, max 135 | p95 24, max 33 |
| Tailscale `_p/view` every 50ms | 96 | 49 | 5s, `seqSpanMs=4926` | p50 70, p95 123, p99 161, max 161 | p95 4, max 56 | p95 125, max 130 | p95 31, max 35 |

Observed browser DOM sync envelope:

| Route | Samples | Unique seqs | Duration | `browserAgeMs` | `fetchMs` | `sourceIntervalMs` | `filterLatencyMs` |
| --- | ---: | ---: | --- | --- | --- | --- | --- |
| Tailscale generated clock UI, DOM sample every 50ms | 112 | 58 | 6s | p50 75, p95 149, p99 173, max 176 | p95 9, max 12 | p95 140, max 143 | p95 19, max 21 |

Latest command:

```bash
TINKABOT_DEMO_BROWSER_SYNC=1 TINKABOT_DEMO_FAST_EVERY=100ms \
  bash scripts/demo-chain-reaction.sh /tmp/tinkabot-rt-green.32iCRS
```

Proof: `/tmp/tinkabot-rt-green.32iCRS/realtime-browser-sync.json`.

Tailscale URL:
`http://forge.tail6c789a.ts.net:40327/artifacts/bundle/clock/index.html`.

Current timing path:

```text
config_bucket schedule retune
-> NATS request/reply on derived tb.bundle.clock.tick
-> tick.sh framed state projection + artifact effect
-> materializer writes bundle KV/Object material
-> KV watch feeds present.sh
-> present.sh emits bundle.clock.view with timing fields
-> shell serves /artifacts/bundle/clock/_p/view from bundle material KV
```

NATS primitives involved: operator/JWT scoped users, account import/export,
request/reply, JetStream KV, Object Store, KV watch/consumer, and the existing
Tinkalet item/schedule flows. No non-NATS realtime channel is part of the proof.

Provisional CKR-REALTIME admission target:

| Metric | Provisional target |
| --- | --- |
| `realtime_ui_sync_age_p95_ms` | `<= 250` in a release-shaped browser probe. |
| `realtime_ui_sync_age_p99_ms` | `<= 500` in a release-shaped browser probe. |
| `filter_latency_p95_ms` | `<= 50` at `100ms` cadence. |
| `source_interval_p95_ms` | `<= 150` at `100ms` cadence. |
| `authority_violation_count` | `0`. |
| `raw_authority_leak_count` | `0`. |
| `terminal_event_loss` | `0` for final authoritative result; do not infer event loss from latest-projection polling. |

Do not freeze max participants/rate yet. Candidate starting bar before admitting
typeracing: prove at least two scoped participants at `10Hz` each for `60s`,
then find the break point for four or more participants before claiming
capacity.

Must measure before CKR-REALTIME freezes:

| Required measurement | Reason |
| --- | --- |
| Browser DOM sync age with Playwright or agent-browser available | Done for the existing clock chain at 100ms cadence and for the generated participant UI action/readback path over Tailscale. |
| KV/watch revision gaps separately from latest-state polling | Latest projection polling can coalesce updates and hide event loss. |
| Two scoped participant profiles after CKR-AUTH | Realtime cannot be admitted before participant authority exists. |
| Reconnect/restart and authoritative scoring/final-result loss | Typeracing correctness depends on final truth, not only live progress. |
| Tailscale route separately from loopback | User-facing demos must be MagicDNS/Tailscale reachable. |

### DKR-4 Genericity Pressure Test Return

Discovery worker: `019ef7bd-9295-7cd3-a028-dba940fff16f`.
Scope: read-only; decide whether the CKR structure remains admissible.

Result: non-empty and admissible. The CKR structure can proceed only where the
mechanism maps back to bundle, Tinkalet profile, authority, material, or the
NATS reaction spine.

Mechanism classification:

| CKR | Mechanism | Classification |
| --- | --- | --- |
| CKR-VIS | Bundle as one-folder app with derived script keys, trigger subjects, projections, and artifact prefixes. | Existing mechanism reuse. |
| CKR-VIS | KV/Object-backed artifact/projection rendering through trusted shell routes. | Existing mechanism reuse. |
| CKR-VIS | Bundle `watches` transform chain through KV watch -> filter -> materializer -> projection/artifact. | Existing mechanism reuse. |
| CKR-VIS | Sandboxed generated UI emits typed content intent through trusted shell/Command Acceptance. | Existing mechanism reuse. |
| CKR-VIS | Generic command-to-item submit bridge from typed UI choice to durable `tb_items` record. | Landed generic mechanism. |
| CKR-VIS | Scoped LLM watcher/readback via Tinkalet profile/watch, not browser observation. | Landed generic mechanism. |
| CKR-VIS | Arbitrary dynamic LLM bundle publication/build admission. | Missing generic mechanism; keep outside frozen CKR-VIS unless separately DKR'd. |
| CKR-VIS | Mermaid renderer/API, direct UI write to `tb_items`, raw subject/token/KV exposure. | Rejected example-specific shortcut. |
| CKR-AUTH | `Runtime.MintUser` scoped JWTs, TTL, provenance, revocation disconnect/reconnect denial. | Existing mechanism reuse. |
| CKR-AUTH | Runtime account mint plus import/export boundaries. | Existing mechanism reuse. |
| CKR-AUTH | Owner admits/revokes participant product service. | Landed generic mechanism. |
| CKR-AUTH | NATS-backed participant record store with app/bundle, participant, lease, status, revocation audit. | Landed generic mechanism. |
| CKR-AUTH | Participant role/trust vocabulary narrower than `caller`. | Landed generic mechanism. |
| CKR-AUTH | Tinkalet participant profiles without leaking raw credentials/subjects. | Landed generic mechanism. |
| CKR-AUTH | Participant/app frame lease for typed generated UI intents. | Landed generic mechanism. |
| CKR-AUTH | Two-participant denied-neighbor proof over real NATS. | Landed proof. |
| CKR-AUTH | Shared `caller.creds`, game token system, raw JWT in frame, broad participant caller role. | Rejected example-specific shortcut. |
| CKR-TURN | Shared app state as NATS-native item/projection material. | Existing mechanism reuse. |
| CKR-TURN | Generic participant app-action path with revision, idempotency, stale/duplicate denial. | Landed generic prerequisite. |
| CKR-TURN | Turn rule evaluation as app/bundle logic over authoritative material. | Reference proof landed through Tinkalet-only packaged demo; browser UI and realtime-heavy mission proof still missing. |
| CKR-TURN | Tic-tac-toe board/cell platform API, client-only enforcement, in-memory game server. | Rejected example-specific shortcut. |
| CKR-REALTIME | NATS-native sync path using request/reply, scoped users, KV/Object, KV watch/consumer, projections. | Existing mechanism reuse. |
| CKR-REALTIME | Participant high-rate action ingress under scoped profiles/leases. | Action ingress, scoped participant watch, bounded two-participant action-gap, and a 60s sustained participant-rate baseline exist; max-rate/breakpoint remains unfrozen. |
| CKR-REALTIME | Browser DOM sync-age and revision-gap measurement harness. | Browser DOM proof landed for the clock chain; participant revision-watch, bounded action-gap, 60s sustained-rate, reconnect/restart cursor-catch-up, terminal-result materialization, packaged realtime participant reference demo, browser participant action/readback bridge, and generated browser participant UI proof landed. |
| CKR-REALTIME | Terminal result/final scoring materialized authoritatively, not client-owned. | Generic terminal materialization proof landed through Tinkalet app-action apply/reject and participant scoped watches; mission-specific realtime scoring still waits for the realtime-heavy reference flow. |
| CKR-REALTIME | Packaged realtime participant reference through scoped Tinkalet profiles and filtered watches. | Packaged `demo:realtime` landed with 60/60 observed scoped actions, `50.59Hz` per participant, zero missing-id/order gaps, zero terminal event loss, no authority leaks, late-action rejection, and Codex/Claude PASS reviews. Browser-originated participant command/readback bridge and generated browser UI proof are now landed as generic prerequisites. |
| CKR-REALTIME | Browser-originated participant action and readback bridge. | Landed generic `participant_action` / `participant_read` command bridge on `tb.app.browser.command`: trusted-shell context carries app/participant, handler forwards action creation to the existing participant action service, scoped reads return app state or own action material, and duplicate/stale/neighbor/payload-escape denials are proven. |
| CKR-REALTIME | Generated participant browser UI over the trusted shell. | Landed release-shaped `demo:realtime-browser`: two Tailscale browser pages leased as Alice/Bob submit 40 total generated-frame actions and 40 own-action readbacks through the trusted shell, with zero denials and zero authority leaks. |
| CKR-REALTIME | Custom realtime/WebSocket game channel, client final-state scoring, latest-projection polling as event-loss proof. | Rejected example-specific shortcut. |
| CKR-TEACH | README/demo chain setup, Tinkalet profile flow, bundle reaction explanation. | Existing mechanism reuse. |
| CKR-TEACH | Release-shaped docs/demo gate tied to CKR rows and Tailscale-visible proof. | Shared substrate mechanism. |
| CKR-TEACH | LLM watch flow and app-user submit flow docs once CKR-VIS/CKR-AUTH land. | Missing doc proof. |

Metric freeze status:

| Metric | Status |
| --- | --- |
| `visualization_acceptance_pass=1` | GREEN: `demo:visual` proves V1-V5 in one release-shaped flow, excluding arbitrary dynamic LLM bundle publication unless a later DKR admits it. |
| `participant_authority_acceptance_pass=1` | GREEN for the generic CKR-AUTH slice: participant mint, duplicate-admit credential rotation, scope/read/raw-write denial, revocation, frame mediation, and `authority_violation_count=0` are proven over real NATS. |
| `docs_and_demo_acceptance_pass=1` | Ready to freeze as a release gate, not a mission family. |
| `turn_based_acceptance_pass=1` | GREEN for the reference proof: packaged binary/Tinkalet flow completes a legal turn sequence and proves wrong-turn, stale, duplicate, and occupied-cell denials on the green action/reducer substrate. |
| `realtime_acceptance_pass=1` | GREEN for the scoped multiplayer mission assembly: CKR-AUTH, CKR-TURN, and CKR-REALTIME together cover M1-M6 with scoped participants, shared NATS material, turn denials, measured realtime action/readback, terminal materialization, and trusted-shell-mediated generated UI. |
| Realtime max participants/rate | Explicitly unfrozen. |

Tripwire review:

| Anti-goal | Tripwire |
| --- | --- |
| Security | Generated UI/scripts/profiles receive raw credentials, subjects, tokens, store handles, or broad `caller` authority. |
| Completeness | Renderer-only CKR-VIS, CKR-AUTH without deny/revoke/frame proof, or CKR-TURN/CKR-REALTIME without release-shaped proof, docs, and failure behavior. |
| No over-abstraction | Any platform primitive named after Mermaid, tic-tac-toe, or typeracing. |
| Double V | RED acceptance cites DOM internals, helper names, or implementation buffers instead of V/M rows and user-visible outcomes. |
| Generic platform | Participant authority or app-action paths are privileged for one use case instead of available to bundles/profiles generally. |
| NATS-first | State, identity, watches, schedules, reactions, realtime sync, or final results leave NATS primitives without a new DKR plus human-ratified exception. |

Recommended next progression task: objective closure review. The scoped
multiplayer mission assembly audit and CKR-VIS visual decision proof now count
`2 / 2`; remaining work is verification/review hygiene, not another mission
family.

### CKR Candidates

These are the current CKR lanes after DKR-1 through DKR-4 returned primitive
maps, authority structure, realtime measurements, and a genericity pressure
test. Mission completeness stays whole: CKR-AUTH can be green without counting
either reference mission family complete.

| CKR | Metric | Target | Source |
| --- | --- | --- | --- |
| CKR-VIS: LLM visual decision loop complete | `visualization_acceptance_pass` | Frozen: `1` with V1-V5 all green in one release-shaped flow; dynamic LLM bundle publication excluded unless separately DKR'd | Release-shaped bundle/Tinkalet/browser/LLM-watch proof; do not split into renderer-only or submit-only delivery. |
| CKR-AUTH: Scoped participant authority complete | `participant_authority_acceptance_pass` | Frozen: `1` with participant mint, scope denial, revocation, frame mediation, and `authority_violation_count=0` | Credential/profile tests and denied-neighbor proof over real NATS authority; must precede multiplayer game delivery. |
| CKR-TURN: Turn-based multiplayer reference complete | `turn_based_acceptance_pass` | Frozen: `1` for a release-shaped packaged Tinkalet flow that proves legal completion plus wrong-turn, stale, duplicate, and occupied denial on top of CKR-AUTH, app-action ingress, reducer/CAS, and rejection receipts | Tic-tac-toe-equivalent rules stay in demo logic; the platform proves generic shared-state and turn-authority mechanisms. |
| CKR-REALTIME: Realtime-heavy multiplayer complete | `realtime_acceptance_pass` | Frozen: `1` for the scoped multiplayer assembly with measured two-participant realtime proof, browser p95 <= 250ms, p99 <= 500ms, filter p95 <= 50ms, source interval p95 <= 150ms at 100ms cadence, 0 authority leaks, and 0 terminal event loss; max participants/rate not frozen | Clock-chain browser sync, participant filtered-watch, bounded action-gap, 60s sustained participant-rate, reconnect/restart cursor-catch-up, terminal-result materialization, packaged realtime participant reference demo, browser participant action/readback bridge, generated browser participant UI proof, and the mission assembly audit are green. |
| CKR-TEACH: User/LLM teaching gate | `docs_and_demo_acceptance_pass` | Frozen: `1` before any mission is counted complete | README and staple demo explain setup chain, Tinkalet setup, LLM watch flow, and app-user flow; this is a gate, not one of the `2 / 2` mission families. |

### PKRs / Tasks

| Task | Done check |
| --- | --- |
| Persist objective board | This file exists and `tasks/todo.md` points to it as the active run control surface. |
| Run DKR-0 | Done: returned requirement rows distinguish mechanism, mission, and example details; CKR split integrated above. |
| Run DKR-1 | Done: returned primitive map names current bundle/material/item/watch surfaces and flags missing visual submit bridge plus LLM watcher profile. |
| Run DKR-2 | Done: returned authority model names current owner/profile/JWT/frame surfaces, participant gaps, and CKR-AUTH metrics. |
| Run DKR-3 | Done: measured current clock/projection path and proposed provisional realtime admission metrics; browser DOM and participant-rate measurements remain required before CKR freeze. |
| Run DKR-4 | Done: pressure-tested genericity, froze admissible CKR-VIS/CKR-AUTH/CKR-TEACH metrics, kept CKR-TURN/CKR-REALTIME provisional, and selected CKR-AUTH as first progression task. |
| Synthesize CKRs | CKR table is updated from DKR evidence without weakening anti-goals. |
| Write first RED acceptance | Done for CKR-AUTH: requirement-level RED covered admit/revoke, participant profile mint/import, denied-neighbor, revocation, and frame mediation. |
| Execute admitted CKR | CKR-AUTH GREEN: code, docs, direct tests, and C3 coverage pass without anti-goal breach; mission-family completion still waits for CKR-VIS/CKR-TURN/CKR-REALTIME/CKR-TEACH proof. |
| Freeze CKR-TURN app-action prerequisite | Done: generic action ingress, scoped participant reads, stale/duplicate/cross-app/direct-write denials, full Go suite, and C3 coverage pass without game-specific API. |
| Freeze CKR-TURN reducer prerequisite | Done: `action apply` consumes pending action records, claims deterministic receipts before state mutation, CAS-resolves shared state, denies duplicate/stale/participant apply, and stays game-agnostic. |
| Build CKR-TURN reference proof | Done: startup participant flags, `action reject`, turn-based reference test, packaged `demo:turn`, README/docs, C3 eval binding, and denial matrix proof are green without board-specific platform APIs. |
| Measure CKR-REALTIME browser DOM sync prerequisite | Done for the existing clock chain: `TINKABOT_DEMO_BROWSER_SYNC=1 TINKABOT_DEMO_FAST_EVERY=100ms bash scripts/demo-chain-reaction.sh /tmp/tinkabot-rt-green.32iCRS` opened the Tailscale clock URL, sampled the generated DOM, wrote `realtime-browser-sync.json`, and passed browser p95/p99, filter p95, and source interval p95 thresholds. |
| Add CKR-REALTIME participant revision-watch prerequisite | Done: app-participant profiles can watch app state and their own action/receipt subtree through filtered KV consumers, are denied neighbor and malformed watch targets before network, and cannot create broad KV consumers. |
| Add CKR-REALTIME participant action-gap harness | Done: two scoped participants submit bounded action sequences through NATS request/reply while filtered watches observe every expected action id with strict per-participant revision increase. |
| Add CKR-REALTIME sustained participant-rate proof | Done: two scoped participants submit 600 action ids each at 100ms cadence over about 60s while filtered watches observe 1200/1200 expected actions. |
| Add CKR-REALTIME reconnect/restart catch-up proof | Done: a scoped participant Tinkalet watch cursor catches up retained action records after Tinkabot restart; persisted participant descriptors refresh to the restarted endpoint, and persisted participant JWTs remain revokable after restart. |
| Add CKR-REALTIME terminal-result materialization proof | Done: scoped participants submit terminal app actions, owner/reducer applies or rejects through Tinkalet, final state materializes in `apps.demo.state.terminal`, scoped watches observe final state and own receipts, and late action rejection does not mutate final state. |
| Add CKR-REALTIME packaged participant reference demo | Done: `bun run demo:realtime` builds the release archive, starts packaged Tinkabot with Alice/Bob participant profiles, disables the packaged NATS sidecar before commands, drives 60 scoped Tinkalet actions, accounts for all own-action watch records, materializes terminal state, rejects a late action, writes `realtime-reference-proof.json`, and passed Codex plus Claude noninteractive reviews including the final redaction-only follow-up. |
| Add CKR-REALTIME browser participant action/readback bridge | Done: canonical `browser.command_intent` admits app/participant context; Tinkabot owns `tb.app.browser.command` for `participant_action` and `participant_read`; action creation forwards to the existing app-action service; readback allows app state and own action material only; duplicate, stale, neighbor, raw-authority, and payload-escape denials are proven without direct browser NATS credentials for generated UI. |
| Count scoped multiplayer mission assembly | Done: read-only DKR audit returned `PASS_TO_COUNT`; rows M1-M6 are covered by CKR-AUTH, CKR-TURN, CKR-REALTIME packaged participant reference, browser command bridge, and generated browser participant UI proof. Objective progress is now `1 / 2`; max participant/rate and arbitrary browser publisher identity remain non-claims. |
| Build CKR-VIS visual decision proof | Done: `item_submit` on `tb.app.browser.command` materializes guarded `artifacts.<artifact>.results.*` items; `item-watcher` profiles are scoped/revokable Tinkalet watch profiles; `demo:visual` proves rendered bundle artifact, sandboxed user submit, durable/restart item readback, scoped watcher readback, and transform-chain update with zero authority leaks. Objective progress is now `2 / 2`; dynamic arbitrary LLM bundle publication remains a non-claim. |

## Current Round Status

| Check | Result |
| --- | --- |
| Goal state | Active `/goal` created on 2026-06-24 for the Tinkabot NATS-native app substrate objective. |
| Current loop posture | Progression-first within the admitted CKRs. DKR structures CKR when the next metric is not yet measurable; implementation proceeds only against requirement rows. |
| DKR-0 result | Complete. Requirement rows V1-V5 and M1-M6 are recorded; recommended split is CKR-VIS, CKR-AUTH, CKR-TURN, CKR-REALTIME behind DKR-3, and CKR-TEACH as a release gate. |
| DKR-1 result | Complete. CKR-VIS has enough primitive structure to start RED later, but two gaps are explicit: generic command-to-item bridge and release-shaped LLM watcher profile. |
| DKR-2 result | Complete. CKR-AUTH must freeze a generic participant authority matrix and prove admit/revoke, denied-neighbor, revocation, and frame mediation before multiplayer game work starts. |
| DKR-3 result | Complete. Current projection fetch, clock-chain browser DOM, participant-rate, revision-gap, reconnect/restart, terminal-result, packaged participant reference, browser action/readback, and generated browser participant UI prerequisites are green. The scoped multiplayer claim audit has passed and counts the multiplayer mission family. |
| DKR-4 result | Complete. CKR structure is admissible; CKR-AUTH is the first progression task because it unlocks TURN/REALTIME and prevents unsafe VIS watcher/submit authority drift. |
| CKR-AUTH result | GREEN. `TestParticipantAuthority` proves participant admit/revoke, duplicate-admit rotation that revokes stale creds, scoped profile import, cross-participant read/write denial, wrong-app/config/schedule raw-write denial, NATS-backed participant record status, direct NATS revoked-credential denial, and unaffected neighbor participant. Frontend isolation proves app/participant frame lease scope. |
| CKR-AUTH evidence | `go test ./tinkabot -run TestParticipantAuthority -count=1`; broader targeted Go tests; `go test ./tinkalet -count=1`; `go test ./... -count=1`; `bun test apps/frontend/tests/isolation.test.ts`; `bunx @typescript/native-preview --noEmit -p apps/frontend/tsconfig.json`; C3 lookup/check/harness with `owned_files=474`, `lookup_errors=0`, `uncovered=0`. |
| CKR-TURN prerequisite result | GREEN. Generic participant app-action/revision ingress exists without game-specific API: participants submit only through `tb.app.<app>.participants.<id>.action`, Tinkabot derives identity from the subject, checks shared state revision, materializes idempotent action records, permits scoped app-state/own-action reads for UI sync, and denies direct KV mutation, stale, duplicate, cross-app, and revoked paths. |
| CKR-TURN prerequisite evidence | `go test ./tinkabot -run 'TestAppActionMalformedSubject|TestParticipantAppActions|TestParticipantAuthority' -count=1`; `go test ./tinkalet -count=1`; `go test ./... -count=1`; `git diff --check`; `scripts/c3-line-coverage-harness.sh` -> `owned_files=477`, `lookup_errors=0`, `uncovered=0`; `C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh check --include-adr` -> `total: 28`, `ok: true`. |
| CKR-TURN reducer result | GREEN. `tinkalet action apply` consumes pending action items, validates the embedded state key/base revision, claims `<action-key>.receipt` before state mutation, updates shared app state with KV CAS, finalizes the receipt, denies stale state without receipt when stale is observed before the claim, denies duplicate receipt, and rejects participant apply as `denied-scope`. |
| CKR-TURN reducer evidence | `go test ./tinkabot -run 'TestAppActionMalformedSubject|TestParticipantAppActions|TestParticipantAppReducer|TestParticipantAuthority' -count=1`; `go test ./tinkalet -count=1`; C3 bindings updated for `docs/matched-abstraction/task/app-action-reducer-contract.md`. |
| CKR-TURN reference result | GREEN. `cmd/tinkabot --participant demo:alice --participant demo:bob` mints app-participant profiles at startup; `tinkalet action reject` writes denied receipts without state mutation; `tinkalet action apply` claims the deterministic receipt key before state mutation so apply/reject serialize through NATS KV `Create`; the reference turn proof completes legal play and materializes wrong-turn, duplicate, stale-revision, and occupied-cell denials through NATS-backed items. This is not a scoped multiplayer mission completion claim because realtime-heavy sync and user-visible multiplayer UI remain. |
| CKR-TURN reference evidence | `go test ./cmd/tinkabot -run 'TestRunStartsPrintsPostureAndStopsOnSignal|TestRunAdmitsParticipantsFromStartupFlag|TestRunRequiresStore|TestRunPrintsVersion' -count=1`; `go test ./tinkalet -run 'TestActionCommandDenials|TestProfileImportListUse|TestProfileImportDenials' -count=1`; `go test ./tinkabot -run 'TestAppActionMalformedSubject|TestParticipantAppActions|TestParticipantAppReducer|TestTurnBasedReferenceMission|TestParticipantAuthority' -count=1`; `go test ./... -count=1`; `bash -n` over demo/package scripts; `git diff --check`; `bun run demo:turn` -> Tailscale URL `http://forge.tail6c789a.ts.net:42781`, proof `/tmp/tinkabot-turn-demo.EIsQmj/turn-proof.json`; C3 coverage `owned_files=479`, `lookup_errors=0`, `uncovered=0`; `c3x eval c3-302` and `c3x check --include-adr` pass; Codex noninteractive follow-up `VERDICT: PASS`, findings none; Claude noninteractive follow-up `VERDICT: PASS`, findings none. |
| CKR-TURN residuals | Explicit non-claims: submit-time stale materialization can race with shared state advancement and is caught by reducer CAS; crash/infrastructure recovery from a `pending` receipt needs a future reclaim path before mission completion. |
| CKR-REALTIME browser sync prerequisite result | GREEN for the existing clock chain only. `TINKABOT_DEMO_BROWSER_SYNC=1` now opens the generated UI through the shown shell URL, samples DOM state from `pre#s`, writes `realtime-browser-sync.json`, and fails if browser p95/p99, filter p95, or source interval p95 exceed the provisional thresholds. Latest proof used the Tailscale URL `http://forge.tail6c789a.ts.net:40327/artifacts/bundle/clock/index.html` and passed with browser p95 `149ms`, p99 `173ms`, filter p95 `19ms`, source interval p95 `140ms`, `112` samples, and `58` unique seqs. |
| CKR-REALTIME browser sync evidence | `TINKABOT_DEMO_BROWSER_SYNC=1 TINKABOT_DEMO_FAST_EVERY=100ms bash scripts/demo-chain-reaction.sh /tmp/tinkabot-rt-green.32iCRS` -> proof `/tmp/tinkabot-rt-green.32iCRS/realtime-browser-sync.json`; `sh -n examples/clock/scripts/present.sh examples/clock/scripts/tick.sh`; `bash -n scripts/demo-chain-reaction.sh scripts/demo-live-patch.sh scripts/demo-turn-based.sh scripts/release-package.sh scripts/package-tinkabot.sh`; `git diff --check`; `scripts/c3-line-coverage-harness.sh` -> `owned_files=480`, `lookup_errors=0`, `uncovered=0`; `C3X_MODE=agent c3x eval c3-502` -> holds; `C3X_MODE=agent c3x check --include-adr` -> `total: 28`, `ok: true`; Codex noninteractive `VERDICT: PASS`, findings none; Claude noninteractive `VERDICT: PASS`, findings none. |
| CKR-REALTIME residuals | Explicit non-claims: latest-projection polling can coalesce revisions, max participants/rate is still unfrozen, and the browser path remains trusted-shell-mediated rather than an arbitrary browser publisher identity proof. |
| CKR-REALTIME participant watch result | GREEN as a revision-accounting prerequisite only. App-participant profiles now locally validate watch target grammar/scope before NATS connection, then use filtered KV watches for `apps.<app>.state` and their own `apps.<app>.participants.<id>.actions` subtree; non-participant watch behavior remains unchanged. `TestParticipantRealtimeWatchEnvelope` proves ordered app-state revisions, own action plus receipt revision visibility, neighbor action-subtree denial, broad consumer-create denial, and no raw authority leakage. |
| CKR-REALTIME participant watch evidence | RED first failed with `watch apps.demo.state denied prefix: denied-scope`; Codex review then found post-connect neighbor denial and malformed-target filter acceptance, both fixed. GREEN proof: `go test ./tinkalet -run 'TestParticipantWatchScopeDenialPrecedesNetwork|TestParticipantWatchFiltersDenyMalformedTargets' -count=1`; `go test ./tinkabot -run TestParticipantRealtimeWatchEnvelope -count=1`; focused app-action/watch/authority test suite; `go test ./tinkalet -count=1`; `go test ./cmd/tinkalet -count=1`; `go test ./... -count=1`; `git diff --check`; `scripts/c3-line-coverage-harness.sh` -> `owned_files=481`, `lookup_errors=0`, `uncovered=0`; `C3X_MODE=agent c3x eval c3-302` -> holds and includes `realtime-participant-watch.md`; `C3X_MODE=agent c3x check --include-adr` -> `total: 28`, `ok: true`; final Codex noninteractive `VERDICT: PASS`, findings none; final Claude noninteractive `VERDICT: PASS`, findings none. |
| CKR-REALTIME action-gap result | GREEN as a bounded harness only. `TestParticipantRealtimeActionGapHarness` admits Alice and Bob, creates independent app state keys, submits 24 unique action ids per participant at a 25ms cadence through scoped NATS request/reply, and verifies each participant's filtered KV watch observes every expected action id with strictly increasing revisions. |
| CKR-REALTIME action-gap evidence | `go test ./tinkabot -run TestParticipantRealtimeActionGapHarness -count=1`; focused action/watch/authority suite; `go test ./tinkalet -run 'TestParticipantWatchScopeDenialPrecedesNetwork|TestParticipantWatchFiltersDenyMalformedTargets' -count=1`; `go test ./... -count=1`; `git diff --check`; `scripts/c3-line-coverage-harness.sh` -> `owned_files=482`, `lookup_errors=0`, `uncovered=0`; `C3X_MODE=agent c3x eval c3-302` -> holds and includes `realtime-participant-action-gap.md`; `C3X_MODE=agent c3x check --include-adr` -> `total: 28`, `ok: true`; Codex noninteractive `VERDICT: PASS`, findings none; Claude noninteractive `VERDICT: PASS`, no claim mismatches. |
| CKR-REALTIME sustained-rate result | GREEN as a 60s baseline only. `TestParticipantRealtimeSustainedActionGapHarness` reuses the scoped action-gap path with Alice and Bob each submitting 600 unique action ids at a 100ms cadence; each participant's filtered KV watch must observe every expected id, reject duplicates/out-of-scope keys, and keep strictly increasing revisions. |
| CKR-REALTIME sustained-rate evidence | `go test ./tinkabot -run TestParticipantRealtimeActionGapHarness -count=1 -v`; `TINKABOT_REALTIME_SUSTAINED=1 go test ./tinkabot -run TestParticipantRealtimeSustainedActionGapHarness -count=1 -timeout 2m -v` -> `1200` actions across `2` participants in `59.906036488s`, with all expected action ids observed; focused action/watch/authority suite; `go test ./... -count=1`; `git diff --check`; `scripts/c3-line-coverage-harness.sh` -> `owned_files=483`, `lookup_errors=0`, `uncovered=0`; `C3X_MODE=agent c3x eval c3-302` -> holds and includes `realtime-participant-sustained-rate.md`; `C3X_MODE=agent c3x check --include-adr` -> `total: 28`, `ok: true`; Codex noninteractive `VERDICT: PASS`, findings none after independently rerunning the sustained proof and gates; Claude noninteractive first pass `VERDICT: FAIL` only because independent-review evidence was not yet recorded, while runtime/authority/anti-overclaim/C3 claims passed; Claude follow-up `VERDICT: PASS`, evidence gap closed with no new overclaim. |
| CKR-REALTIME reconnect/restart result | GREEN as a cursor catch-up prerequisite only. `TestParticipantRealtimeReconnectRestartCatchUp` seeds Alice's Tinkalet cursor on `rt-alice-0`, stops Tinkabot, restarts the same store, re-imports the persisted Alice profile, submits `rt-alice-1..rt-alice-3`, and proves the same cursor replays exactly the missed retained action records in strict revision order. The proof also closes the restart security edge: active participant descriptors refresh to the restarted endpoint, and persisted root-signed participant JWTs remain revokable after process restart. |
| CKR-REALTIME reconnect/restart evidence | RED compile failed on the new acceptance helpers; after helpers, runtime RED failed with `action rt-alice-1 denied submit: connection-failed`, exposing stale participant descriptors after restart. GREEN proof: `go test ./tinkabot -run TestParticipantRealtimeReconnectRestartCatchUp -count=1 -v`; `go test ./embednats -run TestOperatorRevocationAfterRestart -count=1 -v`; focused tinkabot realtime/action/authority suite; focused embednats revocation suite; focused tinkalet participant watch-denial suite; `go test ./... -count=1`; `git diff --check`; `scripts/c3-line-coverage-harness.sh` -> `owned_files=484`, `lookup_errors=0`, `uncovered=0`; `C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-302` -> holds and includes `realtime-participant-reconnect-restart.md`; C3 check from the harness reports `total: 28`, `ok: true`; Codex noninteractive `VERDICT: PASS`, findings none; Claude noninteractive `VERDICT: PASS`, findings none after the harmless test cleanup was removed. |
| CKR-REALTIME terminal-result result | GREEN as a terminal materialization prerequisite only. `TestParticipantRealtimeTerminalResultMaterialization` creates shared terminal state, has Alice and Bob submit actions through scoped participant profiles, applies legal actions through the owner/reducer Tinkalet path, rejects Bob's late action after the final state, and proves the final terminal result is durable `tb_items` material under `apps.demo.state.terminal`. Scoped participant watches observe the final state revision and only their own action/receipt subtrees. |
| CKR-REALTIME terminal-result evidence | RED first failed on missing proof helpers. GREEN proof: `go test ./tinkabot -run TestParticipantRealtimeTerminalResultMaterialization -count=1 -v`; focused realtime/action/authority suite; focused Tinkalet participant watch-denial suite; `go test ./... -count=1`; `git diff --check`; `scripts/c3-line-coverage-harness.sh` -> `owned_files=485`, `lookup_errors=0`, `uncovered=0`; `C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh eval c3-302` -> holds and includes `realtime-participant-terminal-result.md`; `C3X_MODE=agent bash .agents/skills/c3/bin/c3x.sh check --include-adr` -> `total: 28`, `ok: true`; Codex noninteractive `VERDICT: PASS`, findings none, no blocking gaps after rerunning focused terminal/watch/C3 checks; Claude noninteractive `VERDICT: PASS`, findings none, gaps none. |
| CKR-REALTIME packaged reference result | GREEN as a release-shaped packaged participant proof only. `bun run demo:realtime` builds the release archive, starts packaged Tinkabot with `--participant demo:alice` and `--participant demo:bob`, imports owner and participant profiles with packaged Tinkalet, disables the packaged NATS CLI sidecar before user-level commands, submits 60 scoped actions at a measured `50.59Hz` per participant, verifies each participant's own filtered watch observes all expected action ids, materializes terminal state under `apps.demo.state.terminal`, rejects a late action as `race-finished`, and records zero user-level/product-output authority leaks. |
| CKR-REALTIME packaged reference evidence | RED first failed with `error: Script not found "demo:realtime"`; first implementation exposed an ESM inline harness bug and was fixed without product-code changes. GREEN proof: `bun run demo:realtime` -> Tailscale URL `http://forge.tail6c789a.ts.net:39893`, proof `/tmp/tinkabot-realtime-demo.zJPQCA/realtime-reference-proof.json`, `participants_started=2`, `expected_actions=60`, `observed_actions=60`, `revision_gap_count=0` for missing expected action ids or non-increasing own-watch revisions, `participant_rate_hz_per_participant=50.59`, `terminal_event_loss=0`, `authority_violation_count=0`, `raw_authority_leak_count=0`, winner `alice`, final revision `76`; proof records observed action ids per participant; focused realtime/action/authority suite; focused cmd/tinkabot participant startup suite; focused Tinkalet participant watch-denial suite; `go test ./... -count=1`; demo/package `bash -n`; `git diff --check`; `scripts/c3-line-coverage-harness.sh` -> `owned_files=488`, `lookup_errors=0`, `uncovered=0`; `c3 eval c3-302`; `c3 eval c3-501`; serial `c3 check --include-adr` -> `total: 28`, `ok: true`; Codex initial/follow-up/final redaction reviews `VERDICT: PASS`; Claude initial/follow-up/final redaction reviews `VERDICT: PASS`. |
| CKR-REALTIME browser action/readback result | GREEN as a browser-originated participant bridge prerequisite only. Canonical browser command context now carries optional app/participant scope; Tinkabot starts a least-authority browser command service on `tb.app.browser.command`; `participant_action` validates trusted-shell context and payload scope, then forwards to the existing participant action service subject; `participant_read` returns only `apps.<app>.state.>` or the participant's own action material. The packager now rebuilds the embedded frontend before compiling Go binaries. This still is not a realtime-heavy browser UI/reference mission. |
| CKR-REALTIME browser action/readback evidence | RED: SDK schema test failed with `Contract input is invalid`; Go bridge test failed with `nats: no responders available for request`. GREEN: `bun test packages/sdk/tests/base-contract/command-acceptance.test.ts -t T-CMD-PARTICIPANT-CONTEXT`; `bun test apps/frontend/tests/isolation.test.ts`; `cd substrate/go && go test ./tinkabot -run TestBrowserParticipantActionBridge -count=1`; focused realtime/action/authority Go suite; `cd substrate/go && go test ./tinkabot ./cmd/tinkabot ./tinkalet -count=1`; `cd substrate/go && go test ./... -count=1`; `bun test packages/sdk/tests/base-contract`; narrowed JS proof over contract-authority, command-acceptance, frontend isolation, and observe tests -> `22 pass`; `bun run build:frontend`; `bun run pack:tinkabot -- /tmp/tinkabot-pack-bridge-rawfix`; `bash -n` over package/release/realtime demo scripts; `scripts/c3-line-coverage-harness.sh` -> `owned_files=489`, `lookup_errors=0`, `uncovered=0`; `git diff --check`; Codex bridge review `VERDICT: PASS`; Claude bridge review `VERDICT: PASS`; Codex and Claude raw-key follow-ups `VERDICT: PASS`. Blocked checks: full `apps/frontend/tests` needs `/usr/bin/google-chrome`; SDK typecheck is blocked by adjacent local `@lagz0ne/nats-embedded` missing built `dist` types and lacking installed `tsdown`. Residuals: trusted-shell publisher binding is not an arbitrary browser identity proof; frontend/Go raw-key matching is exact normalized while SDK remains stricter. |
| CKR-REALTIME browser participant UI result | GREEN as a release-shaped generated UI prerequisite only. The trusted shell dispatches accepted generated-frame `participant_read` / `participant_action` intents through the cookie-gated browser command NATS connection and returns backend responses to the generated iframe. `bun run demo:realtime-browser` builds the release archive, starts packaged Tinkabot with Alice/Bob scoped participants and `TB_DEMO_SESSION=demo-001`, exposes the shell through Tailscale, seeds state through packaged Tinkalet, then drives two browser pages leased as Alice/Bob. Generated content receives no raw NATS authority. |
| CKR-REALTIME browser participant UI evidence | RED first timed out over Tailscale; focused inspection showed local loopback action/readback worked but Tailscale HTTP crashed on `TypeError: crypto.randomUUID is not a function`. GREEN switched trusted-shell lease nonce generation to a `crypto.getRandomValues` fallback. Codex review then found trusted-shell `tb_session` query interpolation, weaker raw-key matching, and missing direct DOM counter assertions; fixes assign `tb_session` as an input property, align frontend/Go raw-key matching with SDK substring behavior, and assert iframe `actions`/`readbacks`/`denied` counters plus a malicious-session injection guard. Claude review then found rejected command-client caching, accumulating observe connections, narrow leak terms, missing generated-action key guards, and unfiltered outbound command replies; fixes reset failed clients, close stale observe connections, broaden leak terms, guard missing keys, and filter backend responses before iframe delivery. Corrected proof: `TINKABOT_DEMO_BROWSER_ACTIONS=20 TINKABOT_DEMO_BROWSER_INTERVAL_MS=20 bash scripts/demo-realtime-browser.sh /tmp/tinkabot-realtime-browser-green5` passed over Tailscale URL `http://forge.tail6c789a.ts.net:43315`; proof `/tmp/tinkabot-realtime-browser-green5/realtime-browser-proof.json`; `browserPages=2`, `acceptedActions=40`, `readbacks=40`, `deniedDispatches=0`, `authorityLeakCount=0`, action p95 `6ms`, action p99 `9ms`, readback p95 `7ms`, readback p99 `10ms`, shell injection guard passed, Alice/Bob DOM complete with 20 actions, 20 readbacks, and 0 denials each. Focused frontend tests, frontend typecheck, frontend build, focused browser command Go test, focused realtime/authority Go suite, package-level Go suite, full `go test ./...`, script syntax checks, `git diff --check`, C3 coverage `owned_files=491`, `lookup_errors=0`, `uncovered=0`, focused C3 evals, and `c3 check --include-adr` passed. Final Codex review `VERDICT: PASS`, findings none; final Claude review `VERDICT: PASS`, zero new security findings, confirmed all eight prior findings closed. Residuals: trusted-shell-mediated prerequisite only, not arbitrary browser identity proof and not full scoped multiplayer mission completion. |
| CKR-VIS visual decision result | GREEN as the LLM visualization mission proof. `item_submit` on `tb.app.browser.command` lets sandboxed generated UI submit a typed choice through the trusted shell; Tinkabot materializes the choice as guarded `tb_items` material under `artifacts.<artifact>.results.*`; `item-watcher` profiles give an LLM/Tinkalet watcher exact item or prefix watch authority without owner reads or broad scope; and `demo:visual` ties rendered bundle artifact, user submit, watcher readback, transform-chain update, and restart durability into one release-shaped proof. |
| CKR-VIS visual decision evidence | RED first failed because `App.AdmitWatcher`/`App.RevokeWatcher` and `--watcher` startup did not exist. GREEN proof: `bash scripts/demo-visual-decision.sh /tmp/tinkabot-visual-green3` passed over Tailscale URL `http://forge.tail6c789a.ts.net:35995`; proof `/tmp/tinkabot-visual-green3/visual-decision-proof.json` recorded `acceptedIntents=1`, `deniedIntents=0`, `acceptedSubmits=1`, `deniedDispatches=0`, `authorityLeakCount=0`, `submitLatencyMs=73`, `artifactRendered=true`, `watcherIsolated=true`, `watcherHasOwnerProfile=false`, `transformChanged=true`, and `restartDurable=true`. Playwright opened `/artifacts/bundle/clock/index.html` and verified a nonblank rendered bundle artifact with the sequence diagram and live projection panel. The isolated scoped watcher Tinkalet environment contained only default profile `llm`, observed `artifacts.artifact-browser.results.choice` with `choice=diagram-a`, could not direct-read or broaden, and had no owner profile. Owner readback and post-restart owner readback saw the same item and browser-command provenance. Verification: focused Go tests for browser item submit, Tinkalet scoped watcher profile, and watcher startup; frontend isolation/observe tests; native frontend typecheck; frontend build; release-shaped visual demo; script syntax checks; full `go test ./... -count=1`; `git diff --check`; C3 coverage `owned_files=494`, `lookup_errors=0`, `uncovered=0`; `c3x eval c3-302`; `c3x eval c3-501`; `c3x eval c3-502`; `c3x check --include-adr`; Codex noninteractive final review `VERDICT: PASS` with no findings and both prior blockers closed; Claude noninteractive final review `VERDICT: PASS` with LOW residuals only. Residuals: no Mermaid-specific API, no raw browser NATS credential, no owner-profile-as-LLM-watch happy path, no dynamic arbitrary LLM bundle publication claim, watcher event is replay rather than live-before-submit, and raw-authority leak terms are conservative. |
| Scoped multiplayer mission assembly | Counted. Read-only DKR audit returned `PASS_TO_COUNT`: M1 derived scoped profiles, M2 shared NATS material, M3 turn-based denials, M4 measured realtime sync and browser action/readback, M5 authoritative terminal result, and M6 trusted-shell generated UI mediation are covered. |
| Current task | Objective closure complete: C3 coverage/eval/check plus Codex and Claude noninteractive review passed for the `2 / 2` mission count. |
| Open flags | None. |

## Three Anti-Goal Eval Points

| Point | Check | Veto / flag |
| --- | --- | --- |
| Admissibility before acting | Before dispatching a worker, confirm the move preserves shadow authority, NATS-first design, generic platform scope, and Double V test order. | Veto the move if it creates raw authority, non-NATS state flow, example-only platform mechanism, or code-derived acceptance. |
| Direct read after acting | After a worker returns or a task lands, read direct evidence: tests, demo output, C3 lookup/check, review result, or measured sync data. | `breaking` if evidence shows a security, abstraction, NATS, or test-order breach. |
| Paired with objective read | A CKR can move the objective only when its mission metric is green and all anti-goal metrics remain zero. | `pointless` if tasks complete but mission metrics stay flat; `authority_drift` if a worker changes the frame. |

## Flags

| Flag | Opens when | Blocking effect |
| --- | --- | --- |
| cannot | DKR budget is spent and no CKR structure or falsifying evidence returns, or required tooling is unavailable. | Stop affected branch and report only if orchestrator cannot choose another useful branch. |
| breaking | Any anti-goal tripwire breaches or drifts toward breach. | Pause committing work on the affected branch. |
| pointless | Work lands but direct mission metrics do not move after the lag window. | Re-aim decomposition before more execution. |
| authority_drift | Any worker changes the objective, target, anti-goals, action envelope, or approval gates. | Stop the move and return to human frame authority. |

## Operating Loop

Cadence: short autonomous loops with worker check-ins after each DKR/PKR result.
The orchestrator maintains this board and `tasks/todo.md`; workers return
bounded results and do not edit the frame.

Current round: `17`.

Metric freshness:

| Metric | Source | Owner | observed_at | recorded_at | max_age | Lag rule | Missing-data policy |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `complete_reference_missions / target_reference_missions` | Release-shaped acceptance suites, demos, and mission assembly audits | Orchestrator | `2 / 2` | 2026-06-24 | current turn | No lag after acceptance command completes. | Keep branch in DKR or `cannot`; do not infer success from tasks. |
| `security_tripwire_breaches` | Security tests, C3 facts, review, and diff inspection | Orchestrator | `0` for CKR-VIS visual decision proof and scoped multiplayer assembly | 2026-06-24 | current turn | No lag for static/diff evidence; runtime proof lag ends when command finishes. | Pause affected branch. |
| `example_specific_platform_mechanisms` | DKR-4 review and C3/matched-abstraction check | Orchestrator | `0` for CKR-VIS visual decision proof and scoped multiplayer assembly | 2026-06-24 | current turn | No lag. | Return to DKR, not implementation. |
| `non_nats_state_or_event_channels` | DKR primitive maps, code review, and tests | Orchestrator | `0` for CKR-VIS visual decision proof and scoped multiplayer assembly | 2026-06-24 | current turn | No lag. | Veto move unless human ratifies exception. |

Ritual:

| Step | Action |
| --- | --- |
| start-of-turn | Read `tasks/todo.md`, this OKR board, C3 search/read for touched concepts, and current git status. |
| pre-dispatch | Screen the next worker scope against the anti-goal table and action envelope. |
| worker return | Record learning, direct evidence, candidate CKRs, flags, and next admitted move. |
| post-move | Run the narrow meaningful verification and pair the objective read with anti-goal reads. |
| end-of-turn | Update `tasks/todo.md` with current round, evidence, blockers, and next autonomous worker. |

State storage:

| Record | Location |
| --- | --- |
| Frame | This file, write-once except human-ratified revisions. |
| Tree | This file's DKR, CKR, and PKR tables. |
| Results | This file's Current Round Status and `tasks/todo.md` active session entries. |
| Ledger | Append-only status/evidence updates in `tasks/todo.md` plus command/review artifacts when generated. |
| Flags | This file and `tasks/todo.md` until closed. |
