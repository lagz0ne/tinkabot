# Tinkabot Frontend Autopilot OKR

Frame state: human-ratified by continuation request on 2026-06-25 after
Claude `claude -p` review returned `VERDICT: REVISE`.

## Objective

Make Tinkabot serve the frontend-autopilot case end to end: a user installs
Tinkabot, uses Tinkalet to log in with minimal procedure, registers a common
Vite frontend handler/template, creates a realtime frontend app through
Tinkalet, uses that frontend app to talk to the LLM that creates it, asks the
LLM to create a website and collect options, and the LLM reacts by watching
KV-backed state.

Target metric: `frontend_autopilot_reference_families == 7 / 7`, where each
family counts only from observed RED-to-GREEN behavior from a clean install.
The count is anchored by `clean_install_to_kv_reaction_journey_passing == 1 / 1`:
the terminal proof must show user option material written through the product
path, LLM-side watch reaction, and rendered collected-option site output.

Current measured state: `7 / 7` counted from fresh proof
`/tmp/tinkabot-frontend-autopilot.MZToam/frontend-autopilot-proof.json`.
`clean_install_to_kv_reaction_journey_passing == true`; no anti-goal tripwire
breached.

## Anti-Goals

| Anti-goal | Type | Metric / tripwire | Owner |
| --- | --- | --- | --- |
| No NATS-native regression | tripwire | `non_nats_product_path_count == 0`; auth, request/reply, KV/Object state, watch, reaction, and frontend app behavior must use NATS-backed product mechanisms. | Orchestrator |
| UI authored by Claude Opus, not Codex | tripwire plus floor | `codex_direct_ui_changed_lines == 0`, `non_opus_authored_ui_artifact_count == 0`, and `claude_opus_ui_artifact_count >= 1`. | Orchestrator plus UI worker evidence |
| Must be autopilot | tripwire plus drift gauge | `manual_unblock_count == 0` after ratification and `post_install_user_command_count <= 3`. | Orchestrator |
| High performance | drift gauge | option write to LLM watch p95 `<= 250ms`, p99 `<= 500ms`; browser pushed state p95 `<= 100ms`, p99 `<= 250ms`; warm Vite handler render p95 `<= 750ms`. | Orchestrator |
| No authority leak | tripwire | `authority_leak_count == 0`; generated UI receives no raw JWT, seed, bearer, raw subject, KV writer, or owner profile. | Trusted shell and proof script |
| No single LLM truth | tripwire | `independent_review_passes >= 2`, with Codex and Claude noninteractive reviews before final completion. | Orchestrator |

Anti-goal coverage review:

| Harm considered | Selected guardrail | Rejected guardrail | Owner | Cadence |
| --- | --- | --- | --- | --- |
| A fast demo bypasses NATS and still looks like success. | Count only proof records with `non_nats_product_path_count == 0` and NATS-backed source evidence. | Treat visual success as enough. | Orchestrator | Before every CKR count. |
| Codex silently creates UI while proving the rest. | Require Opus-authored UI evidence and `codex_direct_ui_changed_lines == 0`. | Accept one Opus artifact while Codex edits surrounding UI. | Orchestrator | At UI worker return and final review. |
| Autopilot becomes a scripted manual walkthrough. | Count user commands and manual unblock events in proof JSON. | Explain that manual steps are temporary but still count. | Orchestrator | Every clean-install proof. |
| LLM reaction is faked by local callbacks. | Terminal proof must show KV/item watch reaction and rendered output. | Use local process memory or direct HTTP state. | Orchestrator | First RED and final proof. |
| Generated UI sees raw authority for convenience. | Trusted shell mediation and leak scan remain tripwires. | Put NATS credentials in generated UI. | Frontend/substrate worker | Browser proof changes. |

## Action Envelope

Action envelope: allowed moves are repo-local OKR/task docs, C3 reads/lookups,
matched-abstraction task evidence, clean-install RED proof scripts, Tinkalet
profile/login/create/watch/reaction work, Tinkabot bundle/Vite-handler work,
Claude Opus UI delegation through `claude -p` without `--bare`, release-shaped
demo proof, Tailscale browser proof, and independent Codex/Claude review;
forbidden actions are raw NATS user paths, Codex-authored UI, generated UI raw
authority, non-NATS state/reaction channels, manual unblocks beyond the metric,
public release/publish, destructive git operations, or frame/threshold/action
envelope changes without human ratification.

Reject any attempted frame, guardrail, metric, threshold, or action-envelope
change unless the human ratifies it.

## Direct Acceptance Families

| Family | Target | Count rule | Current read |
| --- | --- | --- | --- |
| Clean install and login | `1 / 1` | Release-shaped package starts from a clean store; user imports/selects a Tinkalet profile without raw NATS as normal path; command count stays within budget. | counted from `/tmp/tinkabot-frontend-autopilot.MZToam/frontend-autopilot-proof.json`: 3 post-install user commands, all Tinkalet |
| Vite handler/template registration | `1 / 1` | A common Vite frontend handler/template is registered through a product path and becomes the template for app creation. | counted: `tinkalet app handler register vite --from ... --json` exit `0` |
| Realtime frontend app creation | `1 / 1` | Tinkalet creates a frontend app instance from the handler/template and exposes a browser URL with NATS-backed state delivery. | counted: created app record drives `generatedPath` and `resultKey`; visual state delivery is `trusted-shell.nats-watch.push` |
| Claude Opus UI authorship | `1 / 1` | UI artifacts are authored by Claude Opus evidence, not Codex, and pass visual/interaction checks. | counted: Opus generation/provenance/review hash `99c166354ed069d60e4a4cb81e61f3616c0c7b44e6be2cdb61f036548abe2cf9`, `codex_direct_ui_changed_lines=0` |
| User option collection | `1 / 1` | User submits options through generated UI; accepted option lands as durable KV/item material with revision/provenance. | counted: browser `item_submit` accepted `artifacts.options-site.results.plan` rev `6` |
| LLM KV watch reaction | `1 / 1` | LLM-side watcher observes the option through Tinkalet/KV/item watch and reacts without raw NATS or owner profile. | counted: isolated watcher profile observed 5 live prefix watch events and Claude Opus returned `status=reacted` |
| Performance and authority proof | `1 / 1` | Proof JSON records p95/p99 thresholds, `manual_unblock_count`, `post_install_user_command_count`, `authority_leak_count`, and `non_nats_product_path_count`. | counted: option-watch p95/p99 `123ms`, browser pushed-state p95/p99 `7ms`, warm render p95 `135ms`, all tripwires `0` |

## DKR Queue

DKRs are scoped discovery-worker probes with budgets, probability/confidence
outputs, a named steering decision to unlock, and explicit risk/anti-goal
uncertainty to reduce.

| DKR | Budget | Steering decision | Risk / anti-goal uncertainty |
| --- | --- | --- | --- |
| DKR-0 family mapping | one focused pass | Confirm whether `7` families exactly cover the story and identify any merge/split. | Avoid cascade count and wrong denominator. |
| DKR-1 Vite handler path | one code-read/proof pass | Decide whether existing builder bundle is enough or a new handler registry/create command is needed. | Avoid creating a new app layer if existing bundle-as-app fits. |
| DKR-2 autopilot envelope | one proof-design pass | Freeze clean-install command count, idempotency keys, and proof counters. | Avoid manual setup hidden in script internals. |
| DKR-3 Claude Opus UI handoff | one delegation/proof pass | Define UI worker evidence and non-Opus exclusion proof. | Avoid Codex or human UI authorship leak. |
| DKR-4 KV reaction latency | one measured pass | Freeze watch/reaction timing method and thresholds. | Avoid fake local callbacks or stale latency reads. |

Candidate CKRs and candidate PKRs are not promoted until the orchestrator
accepts the supporting DKR learning checkpoint.

## CKRs

| CKR | Metric | Acceptance |
| --- | --- | --- |
| CKR-INSTALL-LOGIN | `install_login_autopilot_pass == 1` | Fresh package install/import/use path works with Tinkalet only and command budget held. |
| CKR-VITE-HANDLER | `vite_handler_template_pass == 1` | A Vite handler/template is product-registered and usable by app creation. |
| CKR-APP-CREATE | `frontend_app_create_pass == 1` | Tinkalet creates a realtime frontend app from the template. |
| CKR-OPUS-UI | `opus_ui_authorship_pass == 1` | Claude Opus produces the UI artifact; Codex does not edit UI. |
| CKR-OPTION-MATERIAL | `option_materialization_pass == 1` | User option becomes durable NATS-backed material with revision/provenance. |
| CKR-LLM-WATCH | `llm_kv_watch_reaction_pass == 1` | LLM watcher reacts to option through Tinkalet/KV/item watch. |
| CKR-PERF-AUTH | `perf_authority_budget_pass == 1` | Performance, autopilot, authority, and NATS-native counters pass. |

CKR-level discovery/delivery balance: CKRs are measurable contribution context,
not subagent work; each CKR starts with the DKR needed to make its contribution
safe, then promotes only the delivery path that direct proof can measure.

PKRs are progression-worker execution units and must report progress signals at
check-ins.

## First RED

The first proof is a clean-install autopilot acceptance test that should fail
until the journey exists:

```bash
bun run demo:frontend-autopilot
```

Expected initial failure: missing product path for Vite handler/template
registration or app creation. The proof must ultimately record:

- `clean_install_to_kv_reaction_journey_passing: true`
- `frontend_autopilot_reference_families: 7`
- `post_install_user_command_count <= 3`
- `manual_unblock_count: 0`
- `non_nats_product_path_count: 0`
- `codex_direct_ui_changed_lines: 0`
- `non_opus_authored_ui_artifact_count: 0`
- `claude_opus_ui_artifact_count >= 1`
- `option_write_to_llm_watch_p95_ms <= 250`
- `option_write_to_llm_watch_p99_ms <= 500`
- `browser_pushed_state_p95_ms <= 100`
- `browser_pushed_state_p99_ms <= 250`
- `warm_vite_handler_render_p95_ms <= 750`
- `authority_leak_count: 0`

## Eval Points

- **Admissibility before action**: the orchestrator screens objective moves
  against fresh anti-goal readings or a dry-run before dispatch. For this OKR,
  every move must name its NATS-backed path, UI authorship effect, autopilot
  command-count effect, and authority-leak risk before worker dispatch.
- **Direct read after action**: the loop reads the real objective, CKR, and
  anti-goal metrics from source records after workers return. For this OKR,
  source records are proof JSON, changed-path checks, worker/authorship records,
  package/demo logs, browser proof, and independent review output.
- **Paired goal/anti-goal eval**: the loop checks objective progress and
  anti-goal hold together; success requires both the objective target and every
  anti-goal threshold to hold. `7 / 7` with Codex-authored UI or a non-NATS
  reaction path is failure.

## Flags

| Flag | Trigger | Default action |
| --- | --- | --- |
| cannot | DKR budget ends without a credible Vite handler or KV-watch LLM reaction path. | Stop affected branch and return evidence. |
| breaking | Any anti-goal metric breaches, including non-NATS path, Codex-authored UI, manual unblock, or authority leak. | Pause committing moves. |
| authority drift | Worker or loop tries to alter frame, metrics, thresholds, or action envelope without human ratification. | Stop proposed move. |
| pointless | Pointless opens when work finished or a CKR metric moved, but the objective metric stays flat / does not move after the lag window. Domain example: a Vite builder proof improves, but `clean_install_to_kv_reaction_journey_passing` stays false after the lag window. | Re-aim the branch. |

## Operating Loop

The orchestrator owns objective checks, check-ins, the OKR board, and subagent
steering until the objective metric reaches target or a human/blocking flag
stops the loop; for this OKR, that means until
`frontend_autopilot_reference_families == 7 / 7` and
`clean_install_to_kv_reaction_journey_passing == 1 / 1`.

Heartbeat cadence and next_check_at: long-running workers write file-based
progress reports under `.okra/runs/frontend-autopilot-v1/workers/`; cadence is
10 minutes; next scheduled check after creation is `2026-06-25T14:30:08Z`.

Metric freshness:

| Metric | observed_at | recorded_at | max_age | status |
| --- | --- | --- | --- | --- |
| `frontend_autopilot_reference_families` | `2026-06-25T16:16:25Z` | `2026-06-25T16:16:25Z` | `24h` | fresh GREEN: `7 / 7`, proof `/tmp/tinkabot-frontend-autopilot.MZToam/frontend-autopilot-proof.json` |
| `clean_install_to_kv_reaction_journey_passing` | `2026-06-25T16:16:25Z` | `2026-06-25T16:16:25Z` | `24h` | fresh GREEN: `1 / 1` |
| anti-goal metrics | `2026-06-25T16:16:25Z` | `2026-06-25T16:16:25Z` | `24h` | fresh GREEN: `manual_unblock_count=0`, `non_nats_product_path_count=0`, `authority_leak_count=0`, `post_install_user_command_count=3`, `codex_direct_ui_changed_lines=0` |

Steering check-ins record value evidence: inbound signal, decision delta,
affected CKR/PKR/DKR or allocation, expected/direct metric or risk effect, and
freshness/evidence reference. Append `steering_value_score >= 0.75` and
`no_value_checkin_count == 0` for valuable steering check-ins.

Storage schema if promoted into `.okra`: frame keys `frame_version`,
`frame_hash`, `objective`, `anti_goals`, `metric_contracts`,
`action_envelope`, and human approval/ratification evidence; tree keys
`tree_version`, `frame_version`, `orchestrator`, `dkrs`, `ckrs`, and `pkrs`.
The `orchestrator` entry must say it owns `objective checks` and `subagent
steering`.

## Current Queue

| Step | Status | Notes |
| --- | --- | --- |
| Ratify revised frame | Done | User asked to achieve the OKR after Claude `VERDICT: REVISE`; proceed with revised frame. |
| Persist OKR board | Done | This file is the durable board. |
| Add first RED proof | Done | `scripts/demo-frontend-autopilot.sh`, `bun run demo:frontend-autopilot`, and `docs/matched-abstraction/task/frontend-autopilot-red.md` define the clean-install RED. |
| Run RED | Done | First proof `/tmp/tinkabot-frontend-autopilot.atFySk/frontend-autopilot-proof.json` recorded `vite-handler-registration-missing`; second proof `/tmp/tinkabot-frontend-autopilot.BHR8Y7/frontend-autopilot-proof.json` advanced to `frontend-app-create-missing`; latest proof `/tmp/tinkabot-frontend-autopilot.odmGvW/frontend-autopilot-proof.json` advanced to `kv-reaction-journey-not-implemented`. |
| Promote first CKR | Done | DKR-1 accepted a Tinkalet-local Vite handler registry as the first bounded slice; focused Tinkalet tests pass. |
| Fix app creation gap | Done | `tinkalet app create frontend <name> --handler vite --json` writes `apps.<name>.state.frontend` through item KV with `writer=tinkalet-app-create`; focused Tinkalet tests and package proof pass. |
| Add Opus UI and KV reaction proof | Done | Final proof generated the UI through Claude Opus, collected options, and reacted from a live Tinkalet KV watch. |
| Fix command budget drift | Done | `profile import local --use` holds the post-install user path to 3 commands. |
| Independent final reviews | Done | Claude Opus via `claude -p` and read-only Codex follow-up both returned `VERDICT: PASS` on the fresh proof. |
