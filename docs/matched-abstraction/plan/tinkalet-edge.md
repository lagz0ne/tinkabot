---
layer: plan
topic: tinkalet-edge
references:
  - ../approach/tinkalet-edge.md
  - ../approach/product-success.md
  - ../approach/go-substrate.md
  - ../approach/bundle-v1.md
---

# Tinkalet Edge Plan

## Consumed Approach

This Plan consumes `../approach/tinkalet-edge.md` as the owning authority for
the Tinkabot/Tinkalet role split. Tinkabot is the server daemon and shared
authority. Tinkalet is the profile-aware edge CLI/daemon for humans, local
scripts, CI jobs, and local agents.

This Plan also consumes the product bar from `../approach/product-success.md`,
the sealed Go substrate authority from `../approach/go-substrate.md`, and the
packaged-app boundary from `../approach/bundle-v1.md`. It does not reopen NATS
as the substrate, server-side lease enforcement, bundle sandbox posture, or the
rule that raw NATS remains proof/diagnostic surface rather than the primary
user path.

The carried product decisions are:

- Tinkabot owns durable authority, item truth, schedule truth, and timing.
- Tinkalet owns profiles, local edge state, waits/watches on behalf of a user,
  and explicit local reactions.
- A Tinkalet daemon is allowed and expected for long-running watches and local
  reactions.
- Profiles are first-class authority contexts, not only URLs.
- Items, waits, watches, reactions, schedules, profiles, and bundles are the
  first product vocabulary.
- Bundle work remains valuable as a packaged-app form, but it is not the only
  workflow shape.

## Abstraction Descent Contract

No implementation Task should be created directly from the Approach. Each layer
must leave enough artifacts for the next layer to act without guessing.

```text
Approach
  owns: role split, vocabulary, invariants, non-goals
  must leave: concept diagrams, role boundaries, readiness gate
        |
        v
Plan
  owns: decomposition, dependency order, command shape, failure families
  must leave: sequence diagrams, state maps, test matrix, task handoff contracts
        |
        v
Task
  owns: one executable proof
  must leave: acceptance contract, RED artifact, implementation evidence
        |
        v
Code and tests
  own: concrete behavior
  must leave: passing checks, release evidence, updated docs
```

The descent rule is strict: a lower layer may cite higher-layer artifacts, but
it may not invent missing higher-layer decisions. If a Task needs a new role,
vocabulary term, authority boundary, or decomposition axis, it escalates upward
instead of coding through the gap.

## Artifact Ladder

Each layer needs a different artifact pack.

```text
+----------+---------------------+---------------------+---------------------+
| Layer    | Design artifact     | Implementation aid  | Test artifact       |
+----------+---------------------+---------------------+---------------------+
| Approach | role map            | vocabulary boundary | denial principles   |
| Plan     | sequence/state maps | slice handoffs      | failure matrix      |
| Task     | scoped proof brief  | exact acceptance    | RED test/evidence   |
| Code     | local comments only | concrete changes    | passing commands    |
+----------+---------------------+---------------------+---------------------+
```

Required before creating any implementation Task from this Plan:

- a user-flow diagram for the slice;
- an authority diagram showing which role owns each state transition;
- a state/data map naming Tinkabot durable state and Tinkalet local state;
- a command sketch that is user-facing and avoids raw NATS as the happy path;
- a failure-family matrix with denied, stale, revoked, malformed, duplicate, and
  reconnect cases where relevant;
- a test-intent map that names which checks are unit, real embedded NATS,
  daemon/restart, manual, or release-evidence checks.

## Product Surface Map

The first Tinkalet shape should make the current release tour usable without
teaching NATS first.

```text
human / script / CI
        |
        v
    tinkalet
        |
        +-- profile: resolves server, shell, creds, trust context
        |
        +-- item: create/get/resolve/wait/watch product records
        |
        +-- trigger: asks Tinkabot to run an existing intent
        |
        +-- reaction: local daemon watches and runs explicit local action
        |
        +-- schedule: edits server-side timing intent
        v
    Tinkabot daemon
        |
        v
    embedded NATS / JetStream / auth / bundle runtime
```

The CLI grammar below is a Plan-level sketch. Task may refine spelling only
when it preserves the same product concepts:

```text
tinkalet profile list
tinkalet profile use <name>
tinkalet profile import local --store <dir>

tinkalet trigger <bundle-or-intent> [--json]

tinkalet item create <key> [--status pending] [--value <json>]
tinkalet item get <key>
tinkalet item resolve <key> [--value <json>]
tinkalet item wait <key> --for resolved
tinkalet item watch <key-or-prefix>

tinkalet reaction add <name> --watch <key-or-prefix> --run <cmd>
tinkalet reaction list
tinkalet daemon

tinkalet schedule set <key> --every <duration>
tinkalet schedule off <key>
```

Raw NATS commands remain allowed in docs as proof, diagnostics, and advanced
escape hatch. They are not the first tour path.

## State and Authority Map

```text
Tinkabot store
  durable items
  item revisions and provenance
  schedule records
  lease and credential authority
  embedded NATS state
  bundle runtime state

Tinkalet config
  profile names
  default profile
  profile aliases
  profile source metadata

Tinkalet data
  daemon pid/lock
  watch cursors
  reaction registry
  local reaction logs
  retry queue
  cached server metadata
```

Authority rule:

```text
Tinkabot may say: this item is pending, due, accepted, resolved, failed.
Tinkalet may say: on this local machine, run this explicit reaction now.

Tinkabot must not imply it owns the local shell.
Tinkalet must not imply it can bypass Tinkabot leases or durable truth.
```

## Decomposition

Six slices. The first slice is the release-tour bridge; later slices add the
full coordination vocabulary.

### Slice 1: profile-trigger-tour

Owns the first product-shaped user path. Tinkalet can import or select a local
profile, resolve packaged credentials/endpoint data, and trigger an existing
Tinkabot intent such as the clock bundle without requiring the user to run the
NATS CLI. README/examples may add or switch only clearly labeled Slice 1
clock-tour blocks that are executed by the Slice 1 package smoke. This is a local
release-shaped tour proof, not a public release claim. The root README, manual,
release evidence, and package metadata may not present Tinkalet as the primary
public operating surface until Slice 6 promotes the claim across all release
docs and gates.

Required artifact pack before Task:

- profile import/use sequence diagram;
- trigger request authority diagram;
- command sketch for the README tour;
- local profile source contract naming how endpoint, shell URL, credential
  reference, trust, and source are discovered without asking the user to copy
  raw NATS details from daemon stdout;
- failure matrix for missing profile, stale creds, revoked creds,
  denied-neighbor, denied trigger, stronger-credential fallback, unreachable
  daemon, malformed response, duplicate request, profile override, and raw
  substrate leakage;
- test-intent map covering CLI parsing, profile file behavior, real embedded
  NATS trigger proof, visible app-state change, packaged archive behavior,
  no-global-CLI behavior, and executable docs smoke.
- package-tour transcript showing `tinkabot` start, `tinkalet profile import`,
  `profile use`, `trigger`, and visible clock state change from an unpacked
  release-shaped package.

### Slice 2: item-records

Owns the durable item vocabulary. Tinkabot exposes item records with status,
value, reason, revision, provenance, and timestamps. Tinkalet can create, get,
resolve, and wait for items through product commands.

Required artifact pack before Task:

- item lifecycle state diagram;
- item storage and revision authority map;
- wait-until-resolved sequence diagram;
- failure matrix for duplicate create, stale revision, denied resolve, malformed
  value, expired lease, and restart recovery;
- test-intent map covering schema/unit checks, real embedded NATS storage,
  denied-neighbor behavior, restart proof, and command output shape.

### Slice 3: watch-cursors

Owns watch behavior for CLI one-shot streams and Tinkalet daemon cursors.
Tinkalet can watch an item or prefix, survive reconnect where cursor state
allows it, and surface changes without exposing raw KV vocabulary.

Required artifact pack before Task:

- watch attach/reconnect sequence diagram;
- cursor data-dir map;
- duplicate and stale-event handling diagram;
- failure matrix for lost connection, stale cursor, permission loss, malformed
  event, duplicate event, and daemon restart;
- test-intent map covering unit cursor logic, real embedded NATS watch flow,
  daemon restart, and manual terminal behavior.

### Slice 4: local-reactions

Owns local reaction registration and execution. Tinkalet daemon watches a
condition and runs an explicit local command, script, or agent hook. The result
is written back to Tinkabot as an item update or product event.

Required artifact pack before Task:

- reaction trust-boundary diagram;
- local command lifecycle diagram;
- reaction registry map in Tinkalet data;
- failure matrix for command failure, denied writeback, duplicate event, daemon
  crash, retry exhaustion, and removed profile;
- test-intent map covering reaction matching, process execution boundary, real
  embedded NATS writeback, daemon restart, and manual safety documentation.

### Slice 5: schedules

Owns server-side timing as product vocabulary. Tinkalet edits schedule records,
but Tinkabot owns durable timing and attribution. Edge-local timers are not
used to claim server schedule reliability.

Required artifact pack before Task:

- schedule ownership diagram;
- schedule-to-item/write sequence diagram;
- schedule record state map;
- failure matrix for malformed duration, denied schedule edit, missed tick,
  restart catch-up, cancellation, and edge offline behavior;
- test-intent map covering schedule validation, real embedded NATS timing,
  restart/catch-up, and docs examples.

### Slice 6: release-docs-and-proof

Owns the public-facing consolidation after the first useful Tinkalet surface.
Slice 1 proves a runnable local/package-shaped Tinkalet tour exists; Slice 6
proves public docs, manual, release metadata, and centralized evidence now
claim Tinkalet as the primary operating surface. NATS proof remains present and
honest.

Required artifact pack before Task:

- before/after tour diagram;
- command transcript sketch;
- release package contents map;
- failure matrix for stale docs, package missing Tinkalet, sidecar mismatch,
  manual divergence, and overclaiming NATS abstraction;
- test-intent map covering packaging, README smoke, manual gate, release
  evidence, and layer validation.

## Dependency Ordering

Slice 1 precedes release-doc consolidation because it creates the first
Tinkalet tour. Slice 2 precedes slices 3, 4, and 5 because watches, reactions,
and schedules need product records to observe or write. Slice 3 precedes slice
4 because reactions consume watch/cursor behavior. Slice 5 can proceed after
slice 2 once schedule ownership is accepted. Slice 6 follows the first slice
that produces a user-testable path, and expands as later slices land.

```text
profile-trigger-tour
        |
        v
  item-records
   /    |    \
  v     v     v
watch  schedules  release-docs-and-proof
  |
  v
local-reactions
```

## Handoff Contract

Each Task created from this Plan must receive:

- the consumed Approach invariants it must preserve;
- the slice artifact pack listed above, filled with concrete diagrams or maps;
- the user-visible command contract for that slice;
- the state and authority boundary it may touch;
- the failure families it owns;
- the RED artifact it must produce before implementation;
- the checks that count as GREEN;
- the docs or release surfaces that must change if user behavior changes.

A Task is not ready if its command contract is only "wrap NATS", if its tests
only prove a helper in isolation, or if it cannot explain which state is
Tinkabot durable truth versus Tinkalet local convenience.

## Verification Strategy

Testing follows the same abstraction ladder:

```text
Approach test meaning:
  Prove the denied behaviors that would violate product roles.

Plan test meaning:
  Name failure families and choose the surface that proves each family.

Task test meaning:
  Write the RED test or failing evidence before code.

Implementation test meaning:
  Make the narrowest meaningful checks pass, then run the release-facing gates
  touched by the slice.
```

Inside-out checks cover profile parsing, item state validation, cursor logic,
reaction registry behavior, and schedule record validation.

Outside-in checks use real embedded NATS for any behavior that crosses
Tinkalet/Tinkabot, credential, JetStream, watch, or request/reply boundaries.
No slice may claim product behavior solely from mocked NATS seams.

Daemon checks prove startup, lock behavior, reconnect, restart cursor recovery,
and explicit shutdown for the slices that introduce long-running Tinkalet
behavior.

Docs/release checks prove that README, examples, manual commands, package
contents, and release evidence remain honest. If the happy path changes from
NATS CLI to Tinkalet, the docs must change in the same Task.

For Slice 1 specifically, README/example/package smoke must be executable
before implementation is accepted: the archive must contain runnable `tinkabot`
and `tinkalet` binaries, `release.json` must declare both packaged executable
commands in non-promotional contents metadata, and the Tinkalet tour must run
with a scrubbed `PATH` so no global `nats` or
ambient `tinkalet` can satisfy the proof. Manual and centralized
`release:evidence` consolidation remain owned by Slice 6 before any release
claim says Tinkalet is the primary public operating surface.

Clock schedule controls remain diagnostic NATS commands until Slice 5 owns the
schedule vocabulary. Slice 1 must not claim schedule control as NATS-free.

## Escalation Log

- Open: exact CLI spelling remains Plan-level until Slice 1 creates the
  concrete command contract. The vocabulary is fixed by the Approach; spelling
  may still adjust for ergonomics.
- Open: remote pairing flow is not yet chosen. Slice 1 may support local profile
  import first and defer remote pairing if the artifact pack records the
  limitation honestly.
- Open: item storage shape is not chosen. Slice 2 must select the concrete
  store/API shape under Go substrate authority rather than embedding that choice
  in Slice 1.
