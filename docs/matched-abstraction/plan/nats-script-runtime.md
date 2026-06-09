---
layer: plan
topic: nats-script-runtime
references:
  - ../approach/nats-script-runtime.md
  - ../approach/platform-structure.md
  - ./script-nats-cli-proof.md
  - ./platform-structure.md
  - ./orchestration.md
---

# NATS Script Runtime Plan

## Platform Reset Supersession

`docs/matched-abstraction/plan/platform-structure.md` supersedes this plan wherever substrate ownership, local runtime orchestration, or v1 platform shape is concerned. This plan remains useful as evidence for script metadata, activation, permission, record-store, and distribution contracts.

The earlier Bun and `@lagz0ne/nats-embedded` substrate lane is not current platform authority. Go substrate now owns NATS infrastructure, auth, Docker/sandboxing direction, process lifecycle, connection policy, and execution attribution.

## Consumed Approach

This plan consumes `docs/matched-abstraction/approach/nats-script-runtime.md` as authority for script-level mediation and `docs/matched-abstraction/approach/platform-structure.md` as authority for platform ownership. The carried script decisions are trusted-only execution, JetStream KV script records, process-based default scripts, runtime-facade-mediated NATS access, imports as script-facing inside-out abstraction, exposure as outside-in activation abstraction, NATS auth vocabulary as authority, concrete subject patterns, request/reply as one activation source, and `ActivationIntent` as the execution entry contract.

## Decomposition

Superseded evidence direction: contract-first fanout with a Bun runtime substrate lane first, then one vertical lifecycle proof.

Plan units:

- Runtime substrate evidence contract: prior Bun slices prove package management, TypeScript execution, local process lifecycle, env assembly, test harness, and embedded NATS startup/shutdown behavior. Current substrate authority moved to the Go lane in the platform-structure plan.
- Script record contract: one logical JetStream KV record contains TypeScript source plus metadata and revision identity.
- Metadata/schema contract: succinct NATS-focused fields, `desc` and `reason`, concrete subject declarations, and separate schema references for input, output, IPC events, and NATS-forwarded event surfaces.
- Imports/permissions contract: script-facing imports and activation exposure map to NATS `permissions.publish`, `permissions.subscribe`, `allow`, `deny`, and `allow_responses`.
- Activation contract: outside-in sources become authorized `ActivationIntent` values before execution. Activation owns trigger source binding, activation ledger, cursors, dedupe, ack policy, chain attribution, and loop suppression.
- Runtime mediation contract: scripts speak framed stdio RPC, use stderr for diagnostics, send progress/publish requests through runtime-handled protocol messages, receive mediated imports, and may opt into explicit advanced NATS client/CLI capability. Scripts do not receive unrestricted NATS discovery.
- Execution exchange contract: Tinkabot accepts validated `ActivationIntent` values, resolves the KV script record, runs it with input/context, returns a direct response only when reply context exists, and publishes attributed events.
- CRUD/orchestration contract: Tinkabot owns create, read, update, delete, validation, execution request handling, and attribution.
- Vertical proof evidence contract: the prior Bun proof starts embedded JetStream NATS, stores a script record, loads it, executes it, returns a reply, emits events, and stops cleanly. The current v1 platform proof must be re-owned by the Go substrate lane.
- Edge-case contract: the same bounded proof covers success, denied access, invalid metadata, runtime failure, attribution, and cleanup without expanding into a broad platform.

## Handoff Contract

Future Task units receive:

- The owned Plan unit and inherited Approach decisions.
- Non-goals: no sandbox enforcement, no unrestricted NATS exposure, no default NATS client requirement for scripts, no whole-NATS schema layer, no placeholder subjects.
- Inputs: approved vocabulary, contract boundary, representative concrete subjects, schema references, and expected observable behavior.
- Outputs: concrete artifact, RED proof, verification evidence, and wrap-up notes.
- Rejection rule: Task may not rename NATS auth concepts, broaden NATS access, or make Bun-specific mechanics part of the domain protocol.

## Verification Strategy

Verification is layered:

- Metadata checks reject placeholder subjects and require NATS auth vocabulary.
- Subject checks require concrete authoritative prefixes before wildcard use.
- Schema checks validate input, output, and event payload/header contracts by referenced schema id.
- Runtime boundary checks prove scripts receive process channels and mediated imports rather than unrestricted NATS.
- Activation checks prove outside-in exposure is declared, subscribe authority is enforced, request/reply normalizes into `ActivationIntent`, and chain/loop fields are present before execution.
- Embedded substrate evidence checks prove Bun can start JetStream through `@lagz0ne/nats-embedded` and cleanly stop it. Current platform checks must prove the Go substrate path.
- Vertical proof checks store, load, execute, reply, event attribution, and cleanup in one closed loop.
- Script-side outside-in proof uses the real `nats` CLI against embedded NATS. Commands such as `nats request`, `nats publish`, `nats subscribe`, `nats kv`, and `nats object` are the release-facing trigger and observation tools, not mocks.

Edge-case verification matrix:

| Contract | Required proof |
| --- | --- |
| Runtime substrate | Embedded JetStream readiness, startup failure handling, shutdown after success and failure |
| Script record | Missing record, exact KV revision execution, revision mismatch, deleted or stale record handling |
| Metadata/schema | Placeholder subject rejection, invalid wildcard rejection, missing schema reference, schema mismatch |
| Imports/permissions | Declared import succeeds, undeclared import fails, denied publish fails, deny beats allow |
| Activation | Invalid activation declaration, unauthorized source, dedupe conflict, cursor failure, loop suppression, request/reply intent normalization |
| Runtime mediation | Script can emit progress/publish requests through framed stdio RPC; script cannot obtain whole-NATS access through env, imports, TS client, or injected CLI |
| Execution exchange | Success, invalid input, script throw, timeout or cancellation, reply failure behavior |
| Event trail | Success and failure events include execution id, script id/revision, caller, request subject, status, timestamps, and output/error reference |
| Cleanup | Clients close, subscriptions drain, embedded server stops, temp store is removed or isolated, proof is rerunnable |

CLI proof matrix:

| CLI command | Expected platform reaction |
| --- | --- |
| `nats request <allowed-command-subject> <payload>` | command accepted, activation accepted, script executed, response or typed denial returned |
| `nats publish <allowed-source-subject> <payload>` | source router accepts activation, script status/event appears on observable subject or stream |
| `nats publish <denied-neighbor-subject> <payload>` | no script execution; denial or auth failure is observable |
| `nats subscribe <status-or-event-subject> --count 1` | attributed status/event includes activation, script, source, lease, revision, and chain context |
| `nats kv get <projection-bucket> <key>` or `nats object get <artifact-bucket> <name>` | materialized truth or artifact evidence is visible after accepted execution |

## Escalation Log

Open but not blocking:

- Whether `imports` and runtime injection share one logical contract or split into stored metadata and resolved runtime view.
- Exact strictness of left-to-right subject authority validation before broad wildcard use.
- Minimum event payload fields beyond execution id, script id/revision, caller, request subject, status, timestamps, and output/error references.
- Whether `allow_responses` uses NATS dynamic response permission directly or a stricter explicit publish rule for reply subjects.
- Exact framing choice for stdio RPC, with JSON-RPC 2.0 and Content-Length framing as the leading candidate.
- Whether the first durable activation source is KV watch or JetStream durable consumer after request/reply normalization.

Blocking escalation:

- Any request for sandbox enforcement.
- Any requirement to expose unrestricted NATS to scripts.
- Any attempt to treat prior Bun-managed embedded NATS as current platform authority.
- Any attempt to accept happy-path execution without denial and failure attribution.
- Any attempt to keep request/reply as the core execution contract instead of normalizing through `ActivationIntent`.
- Any attempt to implement schedule activation before durable state, lease, catch-up, and fake-clock contracts exist.
