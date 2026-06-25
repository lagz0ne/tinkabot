# Tinkabot Multitenant Isolation OKR

Frame state: human-ratified by user correction on 2026-06-25.

## Objective

Make Tinkabot multitenant-by-design: multiple concurrent users, LLM agents,
and app bundles can be set up in isolation through the explicit Tinkabot daemon,
Tinkalet middleman profiles, and NATS auth boundaries. Isolation is a day-one
product invariant, not a later SaaS hardening layer.

Metric: `isolation_acceptance_scenarios / target_isolation_acceptance_scenarios`.

Current progress: `6 / 6` after DKR-0 mapped existing evidence,
CKR-ISO-AUTH proved the first missing app-scope scenario, the public examples
were reframed so normal work goes through Tinkalet while raw NATS is
owner/operator diagnostic-only, and CKR-ISO-CONCURRENCY added the combined
release-shaped concurrency proof. Counted scenario families: runtime role
separation, tenant/app boundary, user and participant boundary, generated UI
boundary, Tinkalet middleman boundary, and concurrent operations boundary.

Target: `6 / 6` isolation scenario families complete with `0` anti-goal
tripwire breaches.

| Scenario family | Counts only when |
| --- | --- |
| Runtime role separation | Tinkabot daemon owns server authority, embedded NATS/operator/JWT auth, bundle loading, materialization, shell mediation, and revocation; Tinkalet owns local profile import/use/watch/action/reaction; scripts and generated UI inherit derived authority without declaring broad permissions. |
| Tenant and app boundary | Two app or tenant scopes can run concurrently with disjoint NATS subjects, accounts/imports/exports where needed, material buckets, artifact routes, and participant/profile records; cross-app reads, writes, watches, and trigger/action attempts are denied over real NATS. |
| User and participant boundary | Multiple users can be admitted to the same app with separate scoped profiles; each can perform allowed work and cannot impersonate, read, write, watch, or reconnect as another user after rotation or revocation. |
| Generated UI boundary | Browser-visible generated UI remains sandboxed and mediated by the trusted shell; it submits typed intents and receives scoped readbacks, never raw JWTs, seeds, subjects, KV handles, or owner profiles. |
| Tinkalet middleman boundary | User, LLM, and transform flows use Tinkalet product commands/profiles for trigger, item, watch, action, schedule, and reaction work; raw NATS use remains diagnostic-only and cannot be required in the public integration model. |
| Concurrent operations boundary | A release-shaped proof runs concurrent app/user setup and activity through Tailscale-reachable browser or CLI flows, preserves isolation under restart/reconnect, and reports measured sync/latency/rate limits without claiming an unfrozen maximum. |

Deployment interpretation: the product can become SaaS-shaped, but the mechanism
must stay instance-safe. A SaaS control plane may orchestrate app instances,
profiles, and routes later; it must not become the authority that bypasses
Tinkabot daemon auth, Tinkalet profiles, or NATS isolation. Per-app instances are
an acceptable first proof if the same invariants hold when multiple isolated
apps/users are active at once.

## Anti-Goals

| Anti-goal | Metric | Type | Tripwire |
| --- | --- | --- | --- |
| No shared broad authority | `shared_broad_credential_paths` | tripwire | Any participant, watcher, generated UI, script, bundle transform, or user-facing integration receives owner credentials, raw seeds, broad `tb.>` or `$JS.API.>` access, or a shared credential that spans tenants/apps/users. |
| No cross-tenant or cross-app bleed | `cross_scope_access_breaches` | tripwire | Any Alice/Bob, app A/app B, tenant A/tenant B read, write, watch, action, trigger, material, artifact, or profile access succeeds outside its granted scope. |
| No Tinkalet bypass in the integration model | `public_raw_nats_required_paths` | tripwire | README, examples, demos, or tests require users/LLMs to use raw NATS subjects or KV APIs for normal product work instead of Tinkalet/Tinkabot commands. |
| No UI authority leak | `generated_ui_authority_leaks` | tripwire | A generated frame, bundle artifact, or browser demo can see credentials, raw subjects, KV bucket names used as authority, profile files, bearer tokens, seeds, or owner store paths. |
| No late isolation retrofit | `features_claimed_before_isolation_proof` | tripwire | A feature is marked complete before deny-neighbor, revocation/rotation, restart/reconnect, and concurrent-user checks exist for its authority surface. |
| No process-local tenant truth | `non_nats_tenant_truth_paths` | tripwire | Tenant/user/app truth is stored only in process memory, local browser state, ad hoc files, or a custom server channel instead of NATS-backed material, profiles, ledger, or config records. |
| No over-abstraction | `unproven_control_plane_abstractions` | tripwire | A SaaS dashboard, tenancy API, plugin marketplace, or orchestration layer is added before the daemon/profile/NATS isolation proof identifies the minimum needed mechanism. |
| No example-owned security | `example_specific_security_paths` | tripwire | Mermaid, tic-tac-toe, typeracing, or any demo gets a privileged security path unavailable to other bundles/profiles. |

Anti-goal coverage review:

| Harm considered | Selected guardrail | Rejected guardrail | Owner | Cadence |
| --- | --- | --- | --- | --- |
| SaaS pressure creates one global credential. | Every product path maps to scoped NATS auth and profile ownership. | Share one daemon caller credential and rely on app code discipline. | Orchestrator plus security reviewer | Before each CKR admission and after each release-shaped proof. |
| Tinkalet becomes optional in docs and examples. | Public flows use Tinkalet commands/profiles; raw NATS appears only as diagnostics. | Teach users to publish directly to hidden subjects. | Orchestrator | README/manual/example changes. |
| UI convenience leaks credentials. | Trusted shell mediates typed intents/readbacks and filters authority vocabulary. | Put NATS credentials in generated UI. | Frontend/substrate worker, checked by reviewer | Browser or UI proof changes. |
| Concurrent users pass happy-path tests but fail denial cases. | Count a scenario only with allowed path, denied neighbor, revoked/rotated stale credential, and restart/reconnect proof. | Claim isolation from two successful clients alone. | Worker author, checked by orchestrator | Every CKR closeout. |
| Control-plane design outruns runtime proof. | DKR decides topology before CKR implementation. | Build SaaS naming/routes first and patch security later. | Orchestrator | DKR-0 and DKR-1. |

## Action Envelope

Allowed moves: repo-local OKR/task docs, matched-abstraction Approach/Plan/Task
artifacts, C3 queries/evals, RED acceptance tests, focused substrate/frontend
changes that implement admitted CKRs, release-shaped demo scripts, Tailscale
demo proofs, and noninteractive Codex/Claude review for substantial security or
completion claims.

Forbidden moves: destructive git operations, public release/publish actions,
weakening existing security tests, relying on localhost-only user demos, raw
credential delivery to generated UI, custom non-NATS tenant state, raw-NATS
public integration flows, or example-specific security mechanisms.

Approval gates: any objective or anti-goal change, any security posture
expansion, any public network exposure beyond the current local-plus-Tailscale
demo posture, any irreversible external action, and any decision to count an
existing proof without a DKR mapping it to this objective.

## Decomposition

DKR structures CKR. DKR does not count as delivery unless it returns a concrete,
measurable CKR and a proof path.

### DKRs

| DKR | Budget | Learning output |
| --- | --- | --- |
| DKR-0 isolation evidence inventory | One focused pass | Done: existing bundle, participant, Tinkalet, browser, revocation, and realtime proofs are mapped below. Result is `3 / 6` counted, `3 / 6` partial. |
| DKR-1 topology decision | One focused pass | Done: first proof topology is one daemon with multiple app scopes at the participant/action/material layer; per-app instances remain valid but cannot substitute for the shared-daemon isolation proof. |
| DKR-2 threat model and authority matrix | One focused pass | Freeze owner, tenant, app, participant, watcher, LLM, transform, generated UI, Tinkalet, and Tinkabot allowed/denied operations. |
| DKR-3 concurrency envelope | One measured pass | Done: minimum proof is one packaged daemon, two app scopes, four Tinkalet participant profiles, concurrent scoped actions, cross-scope denials, same-store restart/reconnect cursor catch-up, duplicate replay timeout, targeted revocation in the focused regression, and observed-only rate/latency metrics. |

### DKR-0 Isolation Evidence Inventory Return

Discovery workers: `019efc9f-c4bc-71a1-8316-9a3ad0f1f88d` and
`019efc9f-db7c-78c1-b4fd-2999c0b7f2f7`. Scope: read-only, then orchestrator
current-state verification over focused tests.

Interpretation: old app-substrate proofs can count only when they prove the
new multitenant isolation scenario directly. Similar evidence is recorded as
partial, not counted.

| Scenario family | DKR-0 result | Strongest current evidence | Gap before target |
| --- | --- | --- | --- |
| Runtime role separation | Counted | C3 says Tinkabot is the server authority and Tinkalet is the profile-aware edge: `c3-0`, `c3-2`, `c3-3`, `c3-301`, `c3-302`, `ref-shadow-authority-boundaries`, and `rule-generated-code-no-raw-authority`. Focused checks include `TestBundle`, `TestBinaryFirstStartMaterializes`, `TestBundleAccountSeam`, `TestOperatorRevocationDisconnectsLive`, Tinkalet profile/denial tests, and frontend isolation tests. | None for role separation. Multitenant breadth is covered by the rows below. |
| Tenant and app boundary | Partial | Bundle account isolation is strong: `TestBundleAccountSeam` proves account isolation and import-only service crossing; `TestBundle` proves bundle material is invisible from the app account while shell routes still serve it. Participant tests deny wrong-app reads/actions/watches for `apps.other`. | No proof yet shows two positive app or tenant scopes active concurrently in one daemon, with disjoint material/profile records and cross-scope read/write/watch/action/trigger denial. Multi-bundle-in-one-daemon is not the next assumption because the binary currently exposes one `--bundle` posture. |
| User and participant boundary | Counted | `TestParticipantAuthority` admits Alice/Bob, rotates Alice, denies old and revoked credentials, denies cross-user/direct raw writes, and keeps Bob live. Realtime/action/watch/reconnect tests prove scoped participant app actions, filtered watches, cursor catch-up, and revocation after restart. | None for same-app participant isolation. |
| Generated UI boundary | Counted | Frontend isolation denies raw authority vocabulary and enforces lease app/participant scope. `TestBrowserParticipantActionBridge`, `TestBrowserItemSubmitBridge`, `demo:realtime-browser`, and `demo:visual` prove generated UI uses trusted-shell typed intents/readbacks, not raw NATS credentials. | Arbitrary browser publisher identity remains a non-claim, but generated UI mediation is proven for the trusted-shell path. |
| Tinkalet middleman boundary | Counted | Tinkalet profile/trigger/item/action/watch/schedule/reaction commands exist; packaged `demo:chain`, `demo:turn`, `demo:realtime`, and `demo:visual` remove the packaged NATS sidecar before user-level Tinkalet commands; README and public examples teach profile/product flow and frame raw NATS as owner/operator diagnostics only. | None for the public integration model. Low-level operator manuals may still document NATS primitives as mechanism. |
| Concurrent operations boundary | Partial | `demo:realtime` proves two participants through packaged Tinkalet with action accounting, terminal material, and leak checks. `demo:realtime-browser` proves two browser pages through Tailscale with action/readback latency and leak checks. `TestParticipantRealtimeReconnectRestartCatchUp` proves restart/reconnect cursor catch-up. | Evidence is split across single-app proofs. Need one release-shaped concurrent app/user proof that includes restart/reconnect plus measured sync/rate/latency without claiming max capacity. |

Focused current verification:

```bash
cd substrate/go && go test ./tinkabot -run 'TestParticipantAuthority|TestBrowserParticipantActionBridge|TestBrowserItemSubmitBridge|TestParticipantRealtimeWatchEnvelope|TestParticipantRealtimeActionGapHarness|TestParticipantRealtimeReconnectRestartCatchUp|TestParticipantRealtimeTerminalResultMaterialization|TestTurnBasedReferenceMission' -count=1
cd substrate/go && go test ./tinkabot -run 'TestBundle|TestBinaryFirstStartMaterializes|TestLocalProfileDescriptor' -count=1
cd substrate/go && go test ./tinkalet -run 'TestParticipantWatchScopeDenialPrecedesNetwork|TestParticipantWatchFiltersDenyMalformedTargets|TestActionCommandDenials|TestWatchCommandDenials|TestTriggerIntentGrammarDenials|TestTriggerGenericBundleIntent' -count=1
cd substrate/go && go test ./embednats -run 'TestBundleAccountSeam|TestOperatorRevocationAfterRestart|TestOperatorRevocationDisconnectsLive' -count=1
bun test apps/frontend/tests/isolation.test.ts
```

All commands passed on 2026-06-25.

DKR-0 recommendation: admit `CKR-ISO-AUTH` first as a one-daemon two-app-scope
proof. Do not build a SaaS control plane or multi-bundle router yet. Prove one
daemon can admit `demo:alice`, `demo:bob`, `other:alice`, and `other:bob`;
create separate `apps.demo.state.*` and `apps.other.state.*`; allow own-scope
reads/actions/watches; deny cross-app read/watch/action/direct `$KV` attempts;
and prove rotation/revocation kills only the targeted participant while another
app remains live.

### DKR-1 Topology Decision Return

Decision: prove multitenancy first as one Tinkabot daemon with multiple app
scopes at the participant/action/material layer.

Why this topology is first:

| Option | DKR-1 read | Decision |
| --- | --- | --- |
| Per-app instances | Strongly aligned with the current bundle posture: one process can load one bundle, and bundle accounts isolate scripts/material/artifacts behind service import/export. | Keep as a valid deployment model, but do not let it replace the shared-daemon app-scope proof because it does not prove concurrent app/user isolation inside one runtime. |
| One daemon with multiple app scopes | Already partially present: participant ids include app id, action subjects are app-scoped, item keys are app-scoped, and Tinkalet participant profiles carry app/participant metadata. | Use this for `CKR-ISO-AUTH`. It directly tests the day-one multitenancy invariant without adding a SaaS control plane. |
| SaaS control plane | Future orchestration can create instances, route shell URLs, and distribute profiles. | Defer. It must not own runtime authority, bypass Tinkabot daemon auth, or weaken Tinkalet profile isolation. |

`CKR-ISO-AUTH` is frozen as `docs/matched-abstraction/task/multitenant-two-app-scope.md`.

### DKR-3 Concurrency Envelope Return

Decision: close the concurrent operations boundary with both a focused
regression and a release-shaped demo.

Why this shape is first:

| Option | DKR-3 read | Decision |
| --- | --- | --- |
| Extend `demo:realtime` | It already proves fast two-participant action accounting, terminal material, and leak checks, but it is intentionally one app scope. | Do not mutate it; keep it as the single-app realtime reference. |
| Browser multitenant proof | It would add useful UI evidence, but the missing OKR gap is the daemon/profile/NATS isolation mechanism under concurrent app/user load. | Defer until a browser product scenario needs two app scopes. |
| New packaged CLI proof plus focused Go regression | The Go test can pin revocation and restart invariants cheaply; the package demo can prove the public release path with Tailscale shell URLs and packaged Tinkalet profiles. | Use this for `CKR-ISO-CONCURRENCY`. |

`CKR-ISO-CONCURRENCY` is frozen as
`docs/matched-abstraction/task/multitenant-concurrency-reference-demo.md`.

### CKR Candidates

| CKR | Metric | Initial target |
| --- | --- | --- |
| CKR-ISO-AUTH | `tenant_app_user_scope_denial_pass` | GREEN: one daemon admits two app scopes and two users per app; own-scope item/action/watch paths pass; cross-app item/action/watch/trigger/direct `$KV` paths deny; targeted revocation leaves other app/users live. |
| CKR-ISO-PROFILE | `tinkalet_profile_isolation_pass` | Owner, participant, watcher, and LLM profiles are imported/used through Tinkalet; stale/rotated/revoked profiles fail without leaking raw authority. |
| CKR-ISO-UI | `generated_ui_mediation_pass` | Two browser users interact through generated UI with typed intents/readbacks only; authority vocabulary, wrong app/user, stale revision, and neighbor access are denied. |
| CKR-ISO-RUNTIME | `restart_reconnect_revocation_pass` | Isolation survives daemon restart, profile refresh, credential rotation, and revocation; unaffected users/apps remain live. |
| CKR-ISO-CONCURRENCY | `concurrent_setup_activity_pass` | Release-shaped proof runs concurrent setup and activity through Tailscale-reachable browser or CLI flows and reports latency/rate observations. |
| CKR-ISO-DOCS | `integration_model_doc_pass` | README/manual/example docs explain the model from setup-chain, Tinkalet, user/LLM, and operator viewpoints without raw NATS as the normal path. |

## Anti-Goal Eval Points

| Eval point | Question | Required action |
| --- | --- | --- |
| Admissibility check | Does this move preserve daemon-owned authority, Tinkalet middleman profiles, and NATS auth isolation? | If not, reject the CKR or run DKR before implementation. |
| Direct proof check | Does the proof include allowed path, denied neighbor, revoked or stale credential denial, restart/reconnect behavior, and leak scan? | If any piece is missing, do not count the scenario. |
| Paired metric check | Did the scenario metric pass while every anti-goal metric stayed at zero? | Count only paired pass. Any tripwire breach makes progress `0` for that scenario. |

## Open Flags

| Flag | Type | Status | Evidence | Next action |
| --- | --- | --- | --- | --- |
| Missing positive two-app proof | cannot | resolved | `TestMultitenantTwoAppScopeIsolation` proves two active app scopes and two users per app in one daemon, with own-scope read/action/watch pass for all four participants, cross-app item/watch/action/direct-record/raw-action/raw-`$KV` denial, Tinkalet and raw bundle-trigger denial, and targeted revocation. Initial Codex review failed the asymmetric proof; the final Codex and Claude follow-ups passed after the matrix was completed. | Use this as the topology base for the next CKR. |
| Raw NATS still looks like a normal integration path in docs | breaking | resolved | Public examples now say Tinkalet is the normal user/LLM/transform integration surface and raw NATS is owner/operator diagnostics only. | Keep this framing in future README, demo, and example changes. |
| Concurrent proof split across single-app demos | cannot | resolved | `TestMultitenantConcurrentRestartCatchUp` proves four participants across two apps with concurrent Tinkalet action submission, own-cursor replay, product and direct raw-NATS cross-scope denials before and after restart, same-store restart, cursor catch-up, duplicate replay timeout, and targeted revocation with the other three participants still live. `bun run demo:iso-concurrency` proves the packaged public path through Tailscale shell URLs, packaged Tinkalet profiles, 24/24 observed actions, zero revision gaps, zero duplicate replay, 48 product denials, zero authority/leak counters, and observed-only action rate/latency metrics; revocation remains in the focused regression until the packaged CLI has an owner-facing revoke command. | Keep capacity wording observed-only until a separate max/breakpoint CKR exists. |

## Current Queue

| Step | Status | Notes |
| --- | --- | --- |
| Create objective and anti-goal board | Done | This file is the new source for the multitenant isolation objective. |
| Update handoff | Done | `tasks/todo.md` points future agents at this board. |
| Run DKR-0 | Done | Existing evidence maps to `3 / 6`; three scenario families remain partial. |
| Run DKR-1 | Done | First proof topology is one daemon with two app scopes; per-app instances are valid but not a substitute for this proof. |
| Freeze first CKR | Done | `CKR-ISO-AUTH` is frozen in `docs/matched-abstraction/task/multitenant-two-app-scope.md`. |
| Implement CKR-ISO-AUTH | Done | `TestMultitenantTwoAppScopeIsolation` passed after Tinkalet restricted trigger denial was fixed to return `denied-scope` before network and after Codex review forced the two-user/two-app denial matrix to cover all four participants. |
| Close Tinkalet middleman docs boundary | Done | Public examples now keep raw NATS as owner/operator diagnostics only; normal user, LLM, and transform work goes through Tinkalet profiles and commands. |
| Implement CKR-ISO-CONCURRENCY | Done | `TestMultitenantConcurrentRestartCatchUp` and `bun run demo:iso-concurrency` close the combined concurrent operations proof with two app scopes, four profiles, restart/reconnect cursor catch-up, product plus raw bypass denials, regression-level revocation, and observed-only timing metrics. |
