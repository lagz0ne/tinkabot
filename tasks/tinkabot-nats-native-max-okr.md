# Tinkabot NATS-Native Max OKR

Frame state: human-ratified by user direction on 2026-06-25.

## Objective

Maximize Tinkabot as a NATS-native app substrate: realtime UI, auth,
request/reply, KV/Object state, reactions, and demos use NATS primitives
end-to-end, while generated UI remains sandboxed and app performance is good
enough to feel native.

Target metric: `5 / 5` NATS-native product acceptance families pass with
release-shaped proof and at least one Tailscale-reachable live demo. A family
counts only when the direct proof shows NATS as the mechanism, not a local or
HTTP workaround.

Current measured state: `5 / 5` counted. Local proof passed and independent
Codex plus Claude noninteractive reviews returned `VERDICT: PASS`.

## Anti-Goals

| Anti-goal | Type | Metric / tripwire | Owner |
| --- | --- | --- | --- |
| No non-NATS product path | binary tripwire | `nonNatsProductPathCount == 0` for realtime, auth, request/reply, KV/Object state, reactions, and demos. Any user-facing product path that bypasses NATS primitives is breaking. | Orchestrator |
| No browser raw authority | binary tripwire | `authorityLeakCount == 0`; generated UI receives no NATS URL, seed, JWT, bearer, credential, raw subject write access, or raw KV writer. | Trusted shell |
| No polling-as-realtime | binary tripwire | `generatedIframePollingCount == 0` for product realtime state sync. Generated UI may react to pushed lease messages; trusted shell or backend may hold NATS watches. | Browser shell |
| No slow app experience | drift gauge | interactive path p95 <= `75ms`, p99 <= `150ms`; pushed state visible p95 <= `100ms`, p99 <= `250ms`; no revision gaps under the accepted load profile. | Orchestrator |
| No UI slop | binary tripwire | touch targets >= `44px`, chess/app demo screenshots show no mojibake, no overlap, no generic harness as primary surface, and visual smoke passes on mobile and desktop. | Orchestrator |
| No invented mechanism | binary tripwire | no bespoke app bus, local callback truth, direct HTTP state endpoint, game-specific platform API, or hidden in-memory state source counts toward objective progress. | Orchestrator |

## Direct Acceptance Families

| Family | Target | Count rule | Current read |
| --- | --- | --- | --- |
| NATS auth boundary | `1 / 1` | Runtime roles, participants, Tinkalet profiles, revocation, and cross-scope denials enforced by NATS auth with no shadow authority leak. | counted from participant and multitenant isolation proofs |
| Request/reply action path | `1 / 1` | Browser/generator actions enter through NATS request/reply and materialize generic app-action records. | counted from browser participant action bridge and realtime browser UI proof |
| KV/Object durable truth | `1 / 1` | App state, projections, artifacts, and action receipts are KV/Object material, not private process state. | counted from item/action/materializer/bundle proofs |
| Reactive state delivery | `1 / 1` | UI state changes reach generated UI from NATS watch/subscription push, not generated-frame polling. | local proof counted: `stateDelivery=trusted-shell.nats-watch.push`, `generatedIframePollingCount=0` from trusted-shell dispatch log |
| Product-grade demo performance | `1 / 1` | A live Tailscale demo proves realtime UX thresholds, touch usability, and no UI slop while preserving the NATS path. | local proof counted: Tailscale chess proof p95/p99 `71ms`, touch/visual pass |

## DKR Queue

| DKR | Question | Budget | Output |
| --- | --- | --- | --- |
| DKR-NATS-UI-WATCH | What is the smallest generic trusted-shell watch surface that pushes NATS KV/item changes into leased generated frames without raw browser authority? | one code/discovery pass | CKR-UI-WATCH task contract |
| DKR-PERF-BUDGET | Which exact latency counters should become release gates for app actions and pushed state? | one proof pass | thresholds and measurement harness |
| DKR-DEMO-QUALITY | What screenshot/touch checks catch "UI slop" without making the demo game-specific? | one visual proof pass | mobile/desktop smoke criteria |
| DKR-NO-BYPASS-AUDIT | Where do demos/docs still present polling, curl, direct NATS CLI, or HTTP fetch as normal app behavior? | one docs/code scan | cleanup CKRs or explicit owner-diagnostic labels |

## CKRs

| CKR | Metric | Acceptance |
| --- | --- | --- |
| CKR-UI-WATCH | `1 / 1` generated-frame state sync uses shell-held NATS watch push. | Trusted shell watches allowed app state/action material through NATS, posts leased `tinkabot.state` messages to the iframe, and generated content removes product polling loops. |
| CKR-CHESS-QUALITY | `1 / 1` chess demo feels product-grade. | Two Tailscale player links render a clean board on mobile/desktop, touch move selection is reliable, state updates are pushed, p95/p99 meet thresholds, and no raw authority leaks. |
| CKR-DOC-NATS-ONLY | `1 / 1` public docs and README show only NATS-native product mechanisms. | User-facing docs explain auth, request/reply, KV/Object, watches, and reactions as the path; diagnostics are clearly labeled owner/operator-only. |
| CKR-LOAD-WATCH | `1 / 1` accepted load profile has no revision gaps. | Multi-user watch proof records expected vs observed revisions, p95/p99 state visibility, and zero duplicate/lost terminal results. |
| CKR-DOUBLE-REVIEW | `1 / 1` no single LLM truth for final claim. | Codex and Claude noninteractive reviews both return `VERDICT: PASS` on the NATS-only and performance evidence. |

## First RED

The previous chess app violated the new frame because generated content polled
state through repeated `participant_read`. It is secure, but it is not the
maximal NATS-native UI model and it produces bad perceived latency.

Expected RED command family:

```bash
bun run demo:chess
```

The proof must become RED until it records:

- `generatedIframePollingCount: 0`
- `stateDelivery: "trusted-shell.nats-watch.push"`
- `stateVisibleP95Ms <= 100`
- `stateVisibleP99Ms <= 250`
- `touchMoveProof.pass == true`
- `visualSmoke.pass == true`

Latest GREEN proof:

```text
/tmp/tinkabot-chess-demo.XbwrdL/chess-proof.json
shellUrl=http://forge.tail6c789a.ts.net:36681
stateDelivery=trusted-shell.nats-watch.push
generatedIframePollingCount=0
generatedIframePollingProof.source=trusted-shell.dispatched
stateVisibleP95Ms=71
stateVisibleP99Ms=71
authorityLeakCount=0
pass=true
```

Independent review:

```text
Claude noninteractive: VERDICT: PASS
Codex noninteractive: VERDICT: PASS
```

## Action Envelope

Allowed:

- Add generic trusted-shell state watch delivery backed by NATS watch/subscribe
  primitives.
- Extend browser lease messages to carry pushed state snapshots/events.
- Update demos to use Tinkalet profiles, NATS request/reply, KV/Object, and
  NATS watches as the only product path.
- Add performance and screenshot/touch proof gates.

Forbidden:

- Giving generated UI raw NATS credentials or raw KV authority.
- Calling polling "realtime" for product demos.
- Adding chess-, board-, typeracing-, or game-specific platform primitives.
- Using direct HTTP state endpoints, local callback truth, or in-memory app
  state as objective evidence.
- Showing a non-NATS path to the user as the product way.
- Weakening UX acceptance because the substrate proof is technically correct.

## Three Anti-Goal Eval Points

1. Admissibility before dispatch: every proposed CKR/task must name the NATS
   primitive it uses for auth, request/reply, KV/Object truth, and watch/reaction
   delivery. Missing primitive means veto or DKR.
2. Direct read after acting: proof JSON must record `nonNatsProductPathCount`,
   `authorityLeakCount`, `generatedIframePollingCount`, latency p95/p99, and
   visual/touch smoke results from the live Tailscale route.
3. Paired progress read: a CKR counts only when its metric moves and all
   anti-goal readings hold. A faster demo that bypasses NATS is a failure.

## Flags

| Flag | Trigger | Default action |
| --- | --- | --- |
| breaking | any non-NATS product path, raw authority leak, or generated-frame polling survives in counted evidence | pause committing work |
| pointless | task lands but direct NATS-native family count does not move | re-aim the branch |
| cannot | DKR budget ends without a generic NATS primitive path | return to user with options |
| authority drift | worker relaxes the frame, invents a non-NATS mechanism, or expands to product-specific APIs | stop the move |
| slop | performance or visual/touch proof fails | do not count the demo |

## Operating Loop

- Cadence: short autonomous loops; update `tasks/todo.md` after each material
  move.
- Freshness: all current metrics come from proof JSON, C3 evals, and live
  Tailscale demo smoke in the same turn or are marked stale.
- Worker split: discovery workers answer DKR questions; progression workers
  implement scoped CKRs. Workers hand back on unknowns.
- Storage: this file is the frame/tree; proof JSON and task docs are move
  results; `tasks/todo.md` is the resume pointer.
