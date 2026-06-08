---
layer: approach
topic: nats-script-runtime
references:
  - ./charter.md
  - ./platform-structure.md
---

# NATS Script Runtime Approach

## Platform Reset Supersession

`docs/matched-abstraction/approach/platform-structure.md` supersedes the earlier Bun-local substrate direction. This document still owns script mediation, metadata, activation, and no-raw-NATS invariants. It no longer owns substrate runtime authority.

Go substrate now owns NATS infrastructure authority, auth, Docker/sandboxing direction, process lifecycle, connection policy, and execution attribution. Existing Bun and `@lagz0ne/nats-embedded` proofs remain evidence for prior slices, not current platform authority.

## Scope

Tinkabot will support a NATS-native TypeScript script runtime. Scripts are source plus metadata stored as versioned logical records in JetStream KV. Tinkabot owns CRUD, activation, execution orchestration, metadata validation, runtime context assembly, and attribution. Scripts run as trusted code until sandboxing is added.

The system is NATS-centered but not unrestricted NATS exposure. Scripts are NATS-agnostic process contracts by default. They receive input through runtime-owned process channels and ask the runtime facade to publish progress/output/events. Direct NATS client or CLI access is an explicit advanced capability, not the base path.

The first implementation slice may be small, but every included behavior must be complete at its boundary. Happy-path execution alone is not success.

## Core Thesis

The script contract should preserve NATS advantages without requiring scripts to know NATS. Metadata records what a script is for, what it imports, what outside-in exposure may activate it, what the runtime facade may publish or subscribe to on its behalf, what schemas govern its IO and IPC events, and what security posture applies.

## Layer Contract

Approach owns the safety posture, process/facade boundary, activation boundary, NATS vocabulary, subject authority rules, and distinction between script-facing imports, outside-in exposure, and underlying NATS permissions.

Plan owns decomposition into contracts for runtime substrate, script record, imports/permissions, schemas, activation, execution exchange, orchestration, and verification.

Task owns executable proofs for one bounded slice, such as starting local NATS through Bun or proving store/load/execute/event flow.

## Decision Hierarchy

Trusted execution is explicit. Metadata declaring `sandbox: "none"` is accountability, not enforcement.

NATS auth vocabulary is authoritative for access: `permissions.publish`, `permissions.subscribe`, `allow`, `deny`, and `allow_responses`. LLM-facing fields such as `desc` and `reason` explain selection and intent but do not grant authority.

Subject declarations are concrete subjects or concrete wildcard patterns. They do not use placeholder subject strings. Left-side subject tokens carry authority and scope; right-side tokens carry operation/detail. Wildcards are allowed only after a concrete authoritative prefix.

`imports` is the script-facing inside-out abstraction. `exposure` is the outside-in activation abstraction. `permissions` is the underlying NATS security contract. The runtime maps imports into scoped process context and mediated facade operations, and maps exposure into authorized activation sources.

Activation is a first-class runtime layer between NATS/event sources and execution. Its purpose is to convert authorized outside-in stimuli into a normalized execution request without exposing raw NATS access to scripts.

Request/reply is one activation source. It is not the core execution abstraction. Every execution starts from an `ActivationIntent`.

The canonical process protocol is framed stdio RPC: stdin and stdout carry structured protocol messages, while stderr remains diagnostics only. Script-to-runtime progress and publish requests are protocol notifications or requests handled by the runtime facade. The runtime validates, enriches, and forwards allowed events to NATS.

This protocol choice is intentionally long-run and cross-platform. POSIX file descriptors, shell helpers, or one-shot stdin/stdout adapters may exist below the runtime boundary, but they do not define the domain contract.

Deny semantics are meaningful. When `allow` and `deny` overlap, deny wins. `allow_responses` is bounded to the invocation/reply contract and is not a broad publish escape hatch.

Every execution outcome must be attributable to the exact source, metadata, permissions, caller, activation subject, runtime context, chain context, and event trail used. This includes success, denial, invalid metadata, missing record, revision mismatch, script failure, and cleanup behavior.

Activation owns outside-in exposure: request/reply subjects, ordinary subject subscriptions, JetStream durable consumers, KV watches, and schedule providers that may start script execution.

Activation owns durable activation state: source cursors, dedupe records, activation ledger entries, restart recovery, and source-specific delivery position.

Activation owns chain attribution: `chainId`, `rootId`, `parentId`, `triggerId`, source kind, source address, source sequence or revision, dedupe key, observed time, hop, and max hop policy.

Activation owns loop safety. Script-produced events must not recursively trigger unbounded execution through wildcard exposure, overlapping subjects, or repeated source delivery.

Metadata distinguishes `imports` from `exposure`. Imports are inside-out capabilities a script may use while running. Exposure is the outside-in source surface allowed to activate the script. Both use NATS auth vocabulary where applicable: `publish`, `subscribe`, `allow`, `deny`, concrete subject patterns, and deny precedence.

Time-based activation is not an in-memory timer. Schedule activation requires durable schedule state, lease or leadership behavior, catch-up semantics, and idempotency before implementation.

## Reference Policy

This Approach references official NATS concepts as constraints:

- NATS authorization permissions use `publish`, `subscribe`, `allow`, `deny`, and `allow_responses`.
- NATS wildcards use `*` for one token and `>` for trailing tokens, with `>` only at the end.
- NATS messages have subject, payload, headers, and optional reply.
- JetStream streams capture subjects, including wildcard subject sets.

## Plan-Readiness Gate

Plan can proceed when it preserves these invariants:

- Superseded evidence: Bun previously owned local runtime orchestration for early proof slices; Go substrate now owns platform runtime orchestration.
- Go substrate owns local and release runtime orchestration. Existing Bun and `@lagz0ne/nats-embedded` checks remain evidence only.
- Script records live in JetStream KV as source plus metadata.
- Execution starts from `ActivationIntent`; request/reply is one activation source with optional reply context.
- Default scripts do not need NATS client code; NATS interaction goes through the runtime facade and declared permissions.
- Schemas describe script-facing input, output, and event surfaces, not the entire NATS system.
- Exposure and imports stay separate metadata concepts.
- Subscribe authority is enforceable before an activation source is bound or consumed.
- Durable activation state has an owner.
- Chain attribution and loop safety are mandatory.
- The first slice states what is allowed, what is denied, who owns each decision, and how every execution is attributed under success and failure.
- Any move from trusted-only declaration to sandbox enforcement returns to Approach.

Plan must return to Approach if it needs raw NATS exposure, a new authority concept outside mediated imports/exposure and NATS-shaped permissions, ambiguous subject templates, unresolved deny/response semantics, request/reply as the privileged execution core, schedule activation as a local timer, or if edge cases are dropped to make the slice easier.
