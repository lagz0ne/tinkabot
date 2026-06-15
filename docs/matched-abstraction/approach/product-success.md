---
layer: approach
topic: product-success
references:
  - ./endgame-app.md
  - ./session-v2.md
---

# Product Success Approach

Diagram: https://diashort.apps.quickable.co/d/537a1569

## Purpose

This document defines what Tinkabot must achieve to count as a product success. It is a demanding contract, not a roadmap. Its job is to make the final product judgment as hard as it should be — harder than build completion, harder than test passage, harder than demo success. Building the platform as designed and shipping a product that earns repeated use by serious operators are not the same thing; this Approach ensures they cannot be confused.

Every claim here is a bar to clear, not a direction to travel. A working binary that nobody repeatedly chooses for real automation is not a success. A technically sound authority model that operators find too opaque to operate is not a success. A feature that earns one paid engagement and then churns is not a success.

## Core Thesis

Tinkabot wins if and only if a serious technical operator repeatedly chooses it to run unsafe automation as useful software — with durable proof of what happened, meaningful control over what is permitted, and zero transfer of authority to generated code.

"Winning" requires four simultaneous conditions:

1. **Repeated choice**: the operator initiates new automation under Tinkabot, without being asked, because the prior run gave them more confidence and control than the alternative.
2. **Real authority at stake**: the automation touches production systems, credentials, or consequences the operator cannot trivially undo. A sandbox demo is not a win condition.
3. **Useful software**: the automation produces a durable output the operator actually uses. A running process that emits logs is not a win condition.
4. **No authority transfer**: the operator does not hand signing keys, raw NATS subjects, shared credentials, or unchecked shell authority to generated code. Every granted authority is declared, scoped, attributed, and revocable.

If any of these four conditions is missing, the platform is not yet a product.

## Target User

**Beachhead ICP (Ideal Customer Profile)**: a technical operator at a company whose engineers write automation against real systems — CI runners, deployment pipelines, data sync jobs, infra reconcilers, report builders, or content pipelines — and who has been burned at least once by a script that ran with too much authority, left no audit trail, or produced a result nobody could verify. They write Go, TypeScript, Python, or shell. They operate NATS or are willing to operate it. They are not a security team, but they carry operational accountability for what their automation does.

**Non-ICP that must not be served by the v1 claim**: end users running consumer automation, developers who want a no-code agent builder, teams whose "automation" is a webhook or a cron job with no authority footprint. Serving these users with the current platform is deceptive positioning.

**The person who decides to pay**: a technical lead or platform team member who owns both the automation toolchain and the audit/compliance obligation for what it does in production. They will not pay for a platform they cannot explain to their security reviewer in one sentence. The sentence must be: "Tinkabot runs our automation scripts inside a governed runtime that proves what ran, what it was permitted to do, and what it produced — without giving the scripts raw credential access."

## Product Promise

Tinkabot is the governed runtime for generated automation that needs real authority. It lets an operator declare what scripts may do, run those scripts in a mediated execution environment, and prove what happened — without ever handing authority to the generated code itself.

The primary enemy is the status quo: raw shell scripts or local agents that run with shared credentials, leave no durable audit trail, and cannot be safely observed or steered mid-run. Any feature that does not make Tinkabot clearly better than the status quo for a real-authority automation task is not a product feature; it is platform engineering.

## Scope

This Approach covers the product success contract for Tinkabot v1 and the immediate post-v1 expansion tier. It defines what the platform must achieve with real operators running real automation against real systems — not what the implementation must build. Implementation scope and decomposition belong to the technical Approach and Plan documents.

This Approach does not define architecture, substrate design, auth vocabulary, or protocol shape. It does not define which features ship in which slice. It defines the bars those slices must collectively clear before Tinkabot can claim to be a product.

## Minimum Product Bar

The v1 binary must do all of the following before any success claim is made:

- A script runs inside a bwrap jail with an explicit permission set. The operator can read that permission set and understand what it allows.
- The script's effects (projections, artifacts) materialize durably in NATS-backed storage and are observable through the `nats` CLI.
- Every activation carries traceable provenance: what triggered it, what credential ran it, what it produced or failed to produce.
- The operator can deny the script's effects at the materializer gate. Denial is logged with the same provenance as acceptance.
- A bundle (directory of scripts plus a manifest) loads at startup, runs according to its declared manifest, and refuses to start if the manifest is invalid. The operator can inspect the entire authority surface of the bundle from the manifest alone.
- The trusted browser shell renders a bundle's artifacts in a sandboxed iframe. The iframe cannot reach the NATS substrate.
- A running agent session can be observed by a browser viewer with a leaf-scoped credential. The viewer cannot publish to the session's steering subject without going through command acceptance.
- The operator can restart the binary and the durable state survives. A session that was running reconciles to a terminal record rather than disappearing silently.

If any of these behaviors does not work reliably in a production-representative environment, v1 is not at the minimum bar.

## Canonical Paid Jobs

The three jobs an operator will pay for, in order of near-term plausibility:

1. **Governed script execution with audit**: run automation scripts in a declarative permission environment that produces a durable, verifiable execution record. An operator pays because their compliance or security posture requires them to prove what the script did.
2. **Bundle-as-app deployment**: ship a self-contained bundle of scripts and frontend artifacts that runs as managed software — the bundle declares its authority surface, the runtime enforces it, and the operator does not write custom deployment glue. An operator pays because it eliminates the friction of "deploy agent + config + dashboard as three separate things."
3. **Steerable agent session with access control**: run a long-lived agent that a team member can observe and steer from the browser, where the steering authority is revocable and attributed. An operator pays because they need to put a human in the loop without giving that human a terminal or raw NATS credentials.

A feature that serves none of these three jobs is a debt item, not a product priority.

## First Paid-Worthy Win

The earliest moment a pilot customer is likely to pay is when they complete this scenario without outside help:

> The operator writes a manifest, starts the binary against a real NATS server, runs the bundle, observes a materialized projection in the `nats` CLI, triggers a denial by exceeding the permission set, and reads the denial reason in the audit trail — all without reading source code.

If that scenario requires reading source code, the UX is not ready to bill. If it requires more than 30 minutes for an experienced Go/NATS operator, the onboarding friction is too high.

## Trust Decision-Quality Bar

An operator's decision to trust Tinkabot with real authority depends on their ability to answer five questions without help:

1. What is this script permitted to do? (readable from the manifest, not from Go source)
2. What did this run actually do? (readable from the `nats` CLI or audit subject)
3. What was denied and why? (denial reason in the audit trail, output-parseable)
4. Who triggered this run and when? (activation provenance in the ledger)
5. What happens when I restart? (session reconciliation, not silent data loss)

If any of these five questions requires reading source code, opening a debugger, or contacting support, the trust bar is not met.

## Real-Risk Denial Proof

The platform's value proposition rests entirely on its ability to deny an overreaching script at runtime, not retroactively. The proof that denial works must include:

- A script that requests a permission beyond its manifest: the materializer rejects the effect and the denial is logged.
- A bundle that declares a subject collision with a durable claim: the binary refuses to start and the rejection names the colliding claim.
- A credential whose lease has expired: the credential's next use is denied and the denial is attributed to the lease, not to a generic error.
- A script that attempts to write an artifact outside its declared prefix: the write is blocked at the sandbox boundary, not by the script's own cooperation.
- A session viewer who attempts to steer without a current capability lease: the steer is rejected by the runner, not by the trusted shell alone.

Every one of these must work against a real NATS server with real embedded bwrap jails before the denial posture can be claimed to potential customers.

## What Must Feel Excellent

An operator who uses Tinkabot for 30 days and renews should be able to say:

- "I can read the manifest and understand what the bundle is allowed to do before I start it."
- "When something goes wrong, the audit trail tells me exactly what triggered the run, what the script tried to do, and why it was denied or succeeded."
- "Restarting the binary is safe. Nothing is lost, nothing is replayed incorrectly."
- "Adding a new script entry to the bundle and restarting is a five-minute change, not a migration."
- "The browser view of a running agent is read-only by default. Giving someone steering access is an explicit, revocable grant."

If a 30-day operator cannot make all five statements, the product is not excellent at its core job yet.

## Generated-App vs Steerable-Session Success

These are two distinct success shapes that must not be conflated:

**Generated-app success**: an operator ships a bundle whose frontend renders live materialized state and accepts typed commands through the Tinkabot shell. Success means the bundle runs reliably, the permission surface is readable, and the operator can update the bundle without downtime or data loss. The operator never needs to know what NATS subjects the bundle uses internally.

**Steerable-session success**: an operator runs a long-lived agent session and a human observer can read the agent's output and steer it mid-run with attributed, revocable authority. Success means the session survives substrate restarts, the observer's credential is leaf-scoped and short-lived, and the human-in-the-loop decision is recorded with the same provenance as any other activation.

A product that only succeeds at one of these shapes has half the authority model. Both must work at the minimum bar before a market success claim is made.

## Behavioral Pull Metrics

The leading indicators that real product pull exists, before revenue:

- An operator who used the binary last week starts it again this week without prompting. (Repeat use on real automation.)
- An operator adds a second bundle entry without asking for documentation help. (Manifest model internalized.)
- An operator reads the audit trail in the `nats` CLI to debug a denial — rather than adding a `fmt.Println` to the source. (Observability earning its keep.)
- An operator restricts a permission they had previously left wide open after seeing the audit trail. (Policy authoring becoming iterative.)
- An operator asks to give a teammate read-only session access, not a full shell credential. (Least-authority posture becoming intuitive.)

If none of these behaviors emerge within three months of a pilot customer's first real-automation run, the product is not yet pulling.

## Policy and Authority Authoring

A success condition specific to Tinkabot's authority model: operators must be able to author policy without understanding Go internals or NATS auth JWT structure. The manifest is the policy authoring surface. It must:

- Declare permission sets in terms the operator already understands (subjects, prefixes, artifact names), not in terms of internal NATS permission objects.
- Reject invalid policy at load time with an error message that names the problem and the line, not an internal panic or an opaque typed error string.
- Make the authority surface of a bundle deterministic from the manifest alone: the same manifest always produces the same authority surface, regardless of runtime state.

If an operator has to read the Go source to understand why their permission was rejected, the policy authoring surface is not working.

## Recovery and Reconciliation Honesty

The platform must not hide crashes or silent failures behind a "restart and it will be fine" posture. Every session termination path — clean stop, wrapper crash, substrate crash, lease expiry — must resolve to a terminal record that names why the session ended. The operator reads that record and knows whether to restart, investigate, or escalate.

Reconciliation honesty requires:
- A session that was "running" when the binary restarted appears in the `nats` CLI with a terminal status and a reconciliation reason, not as a hanging "in-progress" record.
- An artifact that failed to materialize appears in the audit trail as a failure with a named reason, not as an absence.
- A denial that occurred during a crashed run appears in the ledger, even if the run's own record is incomplete.

If a crash leaves the system in a state where the operator cannot tell what happened without reading Go logs, the recovery posture is not honest.

## Market Success Bar

Tinkabot has not achieved market success at v1 until:

- Three unrelated pilot operators have each run a bundle or session against a real NATS server (not localhost with demo data) and chosen to run a second one.
- At least one pilot operator has used the audit trail to diagnose and fix a permission problem without help.
- At least one pilot operator has used the denial behavior to block a script from taking an action they had not authorized.
- Zero pilot operators have abandoned the platform because the manifest model was too opaque to understand in a day.

These are minimum bars, not aspirational targets. Missing any one of them means the product is not ready to claim market success.

## Retention Bar

An operator is retained if they run at least one bundle or session per month for three consecutive months. Retention below this rate signals one of:

- The platform is solving a problem the operator only has occasionally (and the friction of setup is not worth it for occasional use).
- The platform is solving a problem the operator has replaced with something simpler.
- The manifest model is understood well enough to use once but not well enough to iterate on.

The retention bar must be earned by real-authority automation, not by demo runs or tutorial exercises.

## Harsh Non-Success

These outcomes are product failures regardless of technical quality:

- An operator demos Tinkabot to their team but does not use it for their next real automation task. (The demo is not the product.)
- An operator praises the architecture but keeps their existing raw-shell scripts for production work. (Architecture appreciation is not retention.)
- An operator uses Tinkabot but disables bwrap because the sandbox friction was too high. (Trust model abandoned = no product.)
- An operator cannot explain to their security reviewer what the manifest's permission set means. (Operator cannot own the posture.)
- An operator's first real-authority run fails because the manifest rejection message was not actionable. (Policy authoring friction above the friction budget.)
- A pilot customer bills one month and churns because the second bundle was harder to write than the first. (Learning curve exceeds value curve.)

## Friction Budget

The following friction is acceptable at v1 and must not be used to rationalize product failures:

- Operators must install and operate a NATS server. (Acceptable: this is the target audience.)
- Operators must write a manifest. (Acceptable: manifest authoring is the product's policy surface.)
- Operators must restart the binary to change a manifest. (Acceptable at v1: hot-reload is deferred.)
- The browser shell is a localhost tool; external exposure requires TLS and a reverse proxy. (Acceptable: external exposure is deferred.)

The following friction is not acceptable and, if observed in pilot feedback, is a blocker to retention:

- A manifest rejection message requires reading Go source code to interpret.
- The audit trail requires knowledge of NATS KV internals to read.
- Reconciling a crashed session requires manual KV surgery.
- A permission denial is silent (no record in the audit trail).

## Buyer-Readable Proof

A serious buyer will ask for proof before committing. The proof Tinkabot must be able to show at a first technical review:

- A manifest for a real-looking script bundle with a readable permission set.
- The `nats` CLI output showing the bundle's materialized projection after a run.
- The `nats` CLI output showing a denial record from a script that exceeded its permission set.
- A reconciliation record for a session that ended because the binary restarted.
- A viewer credential's subject list: concrete leaf subjects, no wildcard-subtree grants.

If the platform cannot produce all five of these without a custom demo script, the proof surface is not ready for buyers.

## Paid-Pilot Minimum

A paid pilot is only appropriate when:

- The binary runs reliably for 72 hours against a real NATS server without a manual restart.
- The manifest model has been validated by at least one operator who did not write the code.
- The audit trail is readable by the `nats` CLI without a wrapper script.
- The denial behavior has been exercised by at least one real-authority script (not a test fixture).
- The reconciliation behavior has been exercised by at least one real restart (not a simulated restart).

Charging a pilot customer before these conditions are met is damaging to the trust relationship the platform depends on.

## P1 Expansion Bar

The following expansions become legitimate targets only after the minimum product bar and market success bar are both cleared:

- Multi-node NATS clustering and HA session recovery. (Not before single-node retention is proven.)
- External (non-loopback) NATS WebSocket exposure with TLS. (Not before loopback use proves the authority model.)
- Bundle hot-reload and manifest diff without restart. (Not before manifest authoring is proven stable for cold starts.)
- Multi-viewer session fanout and session list/CRUD UI. (Not before single-viewer sessions prove the steering model.)
- Untrusted bundle isolation beyond bwrap (Docker, gVisor). (Not before bwrap friction is understood and accepted by pilots.)

Expanding to P1 features before the P0 bar is cleared is scope expansion that weakens the product by diluting the core authority model.

## Layer Contract

This Approach owns the product success criteria: target user, product promise, minimum product bar, success metrics, retention bar, trust bar, denial proof requirements, friction budget, and non-success definition. It does not own implementation decomposition, slice sequencing, or technical architecture — those belong to their respective technical Approach and Plan documents.

No Plan or Task document may declare product success by reference to build completion, test passage, or feature checklist alone. Any success claim must be traceable to the bars defined here.

## Decision Hierarchy

1. This Approach (Product Success): what success means.
2. Endgame App Approach: technical authority model and platform loop.
3. Session V2 Approach: steerable session authority invariants.
4. Technical Plan documents: decomposition and verification strategy.
5. Task documents: one executable proof each.

A technical decision that passes all implementation gates but fails a bar defined here is not a success. A feature that clears this Approach's bars but introduces authority regressions in the technical Approaches is not a success either. Both bars must clear simultaneously.

## Plan-Readiness Gate

A plan that targets product success may proceed only when it can answer:

- Which of the three canonical paid jobs does this plan serve?
- Which of the five trust questions does this plan make answerable without reading source code?
- Which denial proof does this plan add or protect?
- Which behavioral pull metric does this plan move?
- Does this plan reduce or increase manifest-authoring friction?

A plan that cannot answer these questions is a technical plan, not a product plan. Technical plans are necessary but not sufficient.
