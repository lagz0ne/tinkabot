---
layer: approach
topic: tinkalet-edge
references:
  - ./product-success.md
  - ./go-substrate.md
  - ./bundle-v1.md
  - ./session-v2.md
---

# Tinkalet Edge Approach

## Purpose

This document refines the product direction where Tinkabot stops being only a
bundle runner and becomes a coordination substrate with two named product
roles:

- **Tinkabot** is the server daemon: durable coordination, authority, timing,
  embedded NATS, shell serving, and substrate-owned state.
- **Tinkalet** is the edge participant: profile-aware client, local edge daemon,
  human/script interface, and optional local reaction runner.

The purpose is to make practical use cases crisp before implementation starts.
The bundle work remains valuable, but it becomes one packaged-app form on top of
coordination primitives. It is not the center of the product.

## Core Thesis

Tinkabot should not try to own every process, environment, dependency graph, or
local tool invocation. Tinkabot wins when it owns the hard shared parts:
authority, durable state, causal records, waits, watches, timing, and proof.
Tinkalet wins when it lets humans, scripts, CI jobs, and local agents participate
in that substrate without speaking NATS directly.

The design therefore separates **shared coordination authority** from **local
edge action**. The server records and authorizes what should happen. The edge
decides how local work actually happens.

## Scope

This Approach covers the product role split between Tinkabot and Tinkalet, the
profile model needed for multiple daemon targets, the local edge data boundary,
the user-facing vocabulary for items, waits, watches, reactions, schedules, and
profiles, and the practical use cases that should guide the first Tinkalet
Plan.

This Approach does not define the exact CLI grammar, storage schema, command
package layout, daemon supervision model, network listener shape, or release
sequence. It also does not redefine the Go substrate, bundle runtime, session
runtime, or product-success bars. Those remain governed by their existing
Approach documents.

## Layer Contract

Approach owns the Tinkabot/Tinkalet product roles, authority boundary, profile
concept, edge-state boundary, vocabulary, invariants, non-goals, and
Plan-readiness gate. It may decide which responsibilities belong to server
authority versus edge participation, but it does not choose implementation
slices or file-level design.

Plan owns Tinkalet decomposition, command grammar, storage layout, daemon mode,
failure families, migration from NATS-first README flows, and verification
strategy under this Approach.

Task owns one executable proof at a time. Task may add packages, commands,
tests, docs, and release evidence for one bounded Tinkalet slice. Task may not
turn Tinkalet into server authority, make raw NATS the required user path, or
make bundles the only supported workflow shape.

## Research Anchors

The model intentionally borrows from established tools without copying their
domain:

- Kubernetes kubeconfig and Docker contexts show the right profile shape: one
  client can hold multiple target contexts, switch a default, and override the
  target for one command. Tinkalet should use this pattern for local, staging,
  production, and shared daemons.
- Temporal's split between durable workflow state, client messages, and worker
  execution validates the separation between durable coordination and local
  workers. Tinkabot should own durable item state; Tinkalet should be a worker
  edge when local reactions are needed.
- NATS JetStream KV watch/history is a strong substrate primitive. It should
  remain below the product vocabulary: users should talk about items,
  reactions, waits, and profiles rather than buckets, subjects, or credentials.

## System Picture

```text
                         shared authority boundary
              +------------------------------------------+
              |              TINKABOT DAEMON             |
              |                                          |
              |  embedded NATS / JetStream / auth        |
              |  durable items / revisions / history     |
              |  timers / schedules / leases             |
              |  shell serving / artifact gateway        |
              |  bundle packaged-app runtime             |
              +--------------------+---------------------+
                                   ^
                                   | scoped creds + profile target
                                   v
              +--------------------+---------------------+
              |              TINKALET EDGE               |
              |                                          |
              |  profiles / local data dir               |
              |  human CLI vocabulary                    |
              |  local watches / reconnect cursors       |
              |  optional local reaction daemon          |
              |  explicit script / agent / CI hooks      |
              +--------------------+---------------------+
                                   ^
                                   |
              +--------------------+---------------------+
              | humans, local scripts, CI jobs, agents   |
              +------------------------------------------+
```

## Role Boundary

Tinkabot owns:

- durable coordination state;
- authorization and lease enforcement;
- item history and causal attribution;
- server-side timers and schedule records;
- the embedded NATS lifecycle;
- bundle serving and sandboxing when a packaged app is used;
- browser shell and artifact surfaces.

Tinkalet owns:

- connection profiles and default target selection;
- local edge state such as cursors, reaction registrations, and local logs;
- human-friendly commands;
- waits and watches on behalf of a user or local tool;
- optional local reaction execution, explicitly configured by the user;
- reconnect behavior for long-running edge participation.

Neither role owns everything. That is the design point.

## Data Directories

```text
~/.local/share/tinkabot/
  server store
  operator key material
  embedded NATS state
  daemon metadata
  role credentials / pairing grants

~/.config/tinkalet/
  profiles
  default profile
  profile aliases
  user-facing preferences

~/.local/share/tinkalet/
  edge daemon state
  watch cursors
  reaction registry
  local logs
  retry state
  cached connection metadata
```

The separate Tinkalet data dir is required because one user may connect one
Tinkalet installation to more than one Tinkabot daemon. A profile is not only a
URL; it is a named authority context.

## Profile Model

```text
profile: local
  server: nats://127.0.0.1:<port>
  shell:  http://127.0.0.1:<port>
  creds:  caller-or-edge creds
  trust:  local-owner
  source: imported from tinkabot store / paired / manually added

profile: staging
  server: tls://...
  shell:  https://...
  creds:  edge-scoped creds
  trust:  remote-edge
  source: paired
```

The active profile behaves like a Docker context or kubectl context: normal
commands use it by default, and a one-command override can target another
profile. The Plan must decide the exact CLI grammar; this Approach owns the
profile concept and authority boundary.

## Product Vocabulary

The user-facing vocabulary should stay small:

- **item**: a durable coordination record with status, value, reason,
  revision, provenance, and timestamps.
- **wait**: block until an item or projection reaches a named condition.
- **watch**: stream changes from an item, projection, or Tinkabot-readable KV
  surface.
- **reaction**: a Tinkalet-owned local rule that observes a condition and runs
  an explicit local action.
- **schedule**: a Tinkabot-owned timing rule that writes intent into durable
  state.
- **profile**: a named connection and authority context.
- **bundle**: a packaged app/runtime shape that consumes the same primitives,
  not a separate product universe.

Raw NATS concepts remain available for diagnostics and proofs, but they should
not be the first tool a user needs.

## Item Lifecycle

```text
                 +---------+
                 | pending |
                 +----+----+
                      |
             work accepted / picked up
                      |
                      v
                 +---------+
                 | working |
                 +----+----+
                      |
          +-----------+-----------+
          |                       |
          v                       v
     +----------+           +----------+
     | resolved |           |  failed  |
     +----------+           +----------+
          ^
          |
     human or local tool may resolve after inspection

optional side states:
  ready       visible to human / local actor
  cancelled   explicit stop, not failure
  expired     timing boundary reached
```

An item is not just a KV value. It is a product-level record with a lifecycle
and audit meaning. This is what makes "wait until resolved" understandable
without teaching KV buckets.

## Timing Boundary

Tinkabot owns server-side timing. Tinkalet may own local convenience timers only
when they are explicitly edge-local.

```text
server-side timing:
  "every 15m, mark inbox/sync requested"
  survives Tinkalet exit
  visible to all profiles with permission
  attributed to a server schedule record

edge-side reaction:
  "when inbox/sync is requested, run ./sync.sh here"
  requires Tinkalet daemon to be alive
  owns local retries and logs
  never implies Tinkabot can run ./sync.sh
```

This distinction prevents Tinkabot from silently becoming a process manager
while still making useful automation practical.

## Use Case 1: Release Smoke Without NATS Literacy

```text
current release path:

  human
    |
    | learns server URL, creds path, subject, request id
    v
  nats CLI request tb.bundle.clock.tick
    |
    v
  Tinkabot daemon

target path:

  human
    |
    | "trigger clock tick"
    v
  Tinkalet CLI
    |
    | profile resolves URL + creds + subject derivation
    v
  Tinkabot daemon
```

This is the first practical win. It replaces the current README's NATS command
with a product command while preserving the NATS proof path underneath.

## Use Case 2: Wait Until Resolved

```text
CI job / local script
    |
    | create item: deploy/123 pending
    v
Tinkabot durable item
    |
    | human sees ready/pending item
    v
human or agent
    |
    | resolve deploy/123 with value
    v
waiting CI job resumes
```

The value is not that Tinkabot runs the deployment. The value is that every
participant can agree on the coordination point and its resolution without
sharing a terminal, a database password, or a NATS subject vocabulary.

## Use Case 3: Human-Machine Middle

```text
agent proposes       human judges        local tool continues
     |                    |                       |
     v                    v                       v
 item: plan/7  --->  item resolved  --->  waits unblock / next step starts
 status=ready       value=approved
```

Tinkabot's job is durable handoff and attribution. Tinkalet's job is making the
handoff usable from a terminal, editor integration, notification, or local
agent. The human remains in the loop without Tinkabot owning the human's tools.

## Use Case 4: KV Reaction Chain With Local Execution

```text
Tinkabot KV/projection changes
          |
          v
Tinkalet daemon watch cursor
          |
          v
explicit local reaction rule
          |
          v
local script / CI command / agent runner
          |
          v
Tinkalet writes item result back to Tinkabot
```

This is intentionally not a black-box bundle. The local environment remains
local. Tinkabot provides the durable clock, state, and authority surface.
Tinkalet provides the edge execution and reconnectable watch.

## Use Case 5: Bundle As Packaged App

```text
bundle-v1 today:
  manifest + scripts + frontend
      |
      v
  Tinkabot sandbox runtime
      |
      v
  projections / artifacts

future positioning:
  bundle = packaged app form
      |
      v
  uses items / watches / schedules / reactions as product primitives
      |
      v
  can still use sandboxed server-side process execution when that is the right job
```

The bundle sandbox work remains a proof of server-side governed execution. It
should not force every workflow into server-side process ownership. For many
operator workflows, Tinkalet local reactions are the better primitive.

## Trust Boundary Diagram

```text
                 can store durable truth
                 can grant/revoke leases
                 can own schedules
                         |
                         v
                    TINKABOT
                         ^
                         |
               scoped profile credential
                         |
                         v
                    TINKALET
                         |
          can run local tools only by explicit local rule
                         |
                         v
              scripts / agents / humans / CI
```

The critical denial rule: a Tinkalet reaction is not a server-side grant to run
code. It is a local edge rule using the user's local authority. Tinkabot can
record and authorize the reaction's output, but it does not own the local
process environment.

## Non-Goals

- Replacing NATS. NATS remains the substrate for auth, state, watch, and message
  transport.
- Making Tinkabot a general-purpose process supervisor. Server-side execution
  exists for bundles and governed runtime cases, not as the default automation
  answer.
- Hiding all authority. Tinkalet should hide raw NATS mechanics, not remove the
  need to understand which profile and lease are active.
- Building a full workflow language in the first pass. The first pass needs
  durable items, waits, watches, profiles, and simple reactions.
- Making Tinkalet reactions reliable when the Tinkalet daemon is not running.
  Edge reactions are edge-local; server schedules are the durable timing layer.

## Invariants

1. **Tinkabot is the shared authority.** Durable coordination, profile grants,
   lease enforcement, and schedule records live server-side.
2. **Tinkalet is an edge actor.** It may be long-running, but its local
   reactions are not server authority and do not imply server ownership of local
   processes.
3. **Profiles are first-class.** Every Tinkalet operation targets an explicit or
   active profile; profile state lives outside the Tinkabot server store.
4. **Items are product records, not raw KV values.** A waitable item has status,
   value, reason, revision, provenance, and lifecycle semantics.
5. **NATS is hidden, not bypassed.** The product vocabulary compiles down to
   NATS-backed state, subjects, requests, and watches; raw NATS remains a
   diagnostic layer.
6. **Timing and execution are separate.** Tinkabot can say when work is due;
   Tinkalet or another actor decides how local work runs unless a server-side
   bundle explicitly owns execution.
7. **Human-in-the-loop is a primary path.** A workflow that pauses for a human
   to resolve an item is not a fallback; it is one of the core product shapes.
8. **Bundles consume primitives.** Bundles are packaged apps over the same
   item/watch/schedule/reaction model, not a competing abstraction.

## Decision Hierarchy

1. Go Substrate Approach: embedded NATS, operator/JWT auth, authority envelopes,
   durable substrate mechanics.
2. Product Success Approach: serious operators must repeatedly choose the tool
   for real automation with proof and control.
3. This Approach: Tinkabot/Tinkalet product role split, edge-state model,
   vocabulary, and use-case boundaries.
4. Bundle V1 Approach: packaged server-side app/runtime form.
5. Session V2 Approach: long-lived agent/session execution shape.
6. Future Tinkalet Plan: decomposition, command grammar, storage layout,
   failure families, and verification.

## Plan-Readiness Gate

A Plan may begin when these decisions are accepted:

- The two product names map to roles: Tinkabot is server authority; Tinkalet is
  edge participant.
- Tinkalet may be both one-shot CLI and long-running local daemon.
- Tinkalet owns profile config and local edge data in directories separate from
  the Tinkabot store.
- Product vocabulary centers on items, waits, watches, schedules, reactions,
  and profiles.
- Local reaction execution belongs to Tinkalet, not Tinkabot, unless a bundle
  explicitly uses the server-side governed execution path.
- Bundle work remains useful as a packaged-app layer but no longer defines the
  whole product.

All readiness conditions are met for a Plan whose first release-shaped outcome
is practical: replace the README's NATS hand-holding with Tinkalet profile,
trigger, wait, watch, and schedule commands while preserving NATS as the
substrate proof layer.
