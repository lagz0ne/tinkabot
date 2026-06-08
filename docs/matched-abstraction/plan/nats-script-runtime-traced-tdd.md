---
layer: plan
topic: nats-script-runtime-traced-tdd
references:
  - ../approach/platform-structure.md
  - ../approach/nats-script-runtime.md
  - ./platform-structure.md
  - ./nats-script-runtime.md
---

# NATS Script Runtime Traced TDD Plan

## Platform Reset Supersession

`docs/matched-abstraction/plan/platform-structure.md` supersedes this traced-TDD plan wherever substrate ownership, local runtime orchestration, or release platform shape is concerned. This plan remains historical test evidence for the prior Bun proof and current authority for script mediation, activation, permission, record-store, process protocol, and event-trail test ownership until those lanes are re-cut under the Go substrate.

## Consumed Approach

This plan consumes `docs/matched-abstraction/approach/platform-structure.md`, `docs/matched-abstraction/approach/nats-script-runtime.md`, `docs/matched-abstraction/plan/platform-structure.md`, and `docs/matched-abstraction/plan/nats-script-runtime.md` as authority. It carries forward the test bar: the first implementation slice is small but complete, NATS stays behind the Tinkabot/runtime facade, scripts speak framed stdio RPC by default, errors are typed, and the vertical proof compounds trust rather than replacing lower-layer tests.

## Decomposition

Primary layer graph: https://diashort.apps.quickable.co/d/d29e5453
Error ownership graph: https://diashort.apps.quickable.co/d/90f4566b
Protocol graph: https://diashort.apps.quickable.co/d/0da56487
Vertical proof graph: https://diashort.apps.quickable.co/d/12a339dc
Activation graph: https://diashort.apps.quickable.co/d/407896c2

Dependency order:

1. Superseded runtime substrate evidence: Bun started and stopped embedded JetStream NATS through `@lagz0ne/nats-embedded`; current substrate authority moved to the Go lane.
2. Script record store: JetStream KV stores source plus metadata and resolves exact revisions.
3. Metadata/schema validator: concrete subjects, wildcard rules, schema references, trust posture, and NATS auth vocabulary.
4. Imports/permissions: script-facing imports mapped to `permissions.publish`, `permissions.subscribe`, `allow`, `deny`, and bounded `allow_responses`.
5. Activation: outside-in sources become authorized `ActivationIntent` values with attribution, ledger, cursor, dedupe, ack, and loop safety.
6. Framed stdio RPC protocol: JSON-RPC 2.0 over `Content-Length` framing, with stderr diagnostics isolated.
7. Process runtime: Bun process spawn, env/path assembly, stdin/stdout/stderr, timeout, cancellation, and cleanup.
8. Runtime mediation: facade methods that validate script publish/progress/import requests before NATS forwarding.
9. Attribution event trail: success, failure, denial, caller, source, revision, chain, and output/error attribution.
10. Execution exchange: accepts `ActivationIntent` values and composes lower layers into one typed execution result.
11. Vertical proof: a real embedded-NATS closed loop that proves the composed contract without retesting lower-layer internals.

Declared error ownership:

| Layer | Owns | Declared errors |
| --- | --- | --- |
| Runtime substrate | Embedded NATS and JetStream lifecycle | `SubstrateStartupFailed`, `SubstrateUnavailable`, `SubstrateCleanupFailed`, `SubstrateCritical` |
| Script record store | KV record lookup, history, revision, persistence | `RecordNotFound`, `RecordRevisionMismatch`, `RecordDeletedOrStale`, `RecordWriteConflict`, `RecordPersistenceFailed`, `RecordCritical` |
| Metadata/schema validator | Metadata shape, subject patterns, schema ids, trust posture | `MetadataInvalid`, `PlaceholderSubjectRejected`, `WildcardPatternInvalid`, `SchemaReferenceMissing`, `SchemaMismatch`, `SecurityPostureUnsupported`, `MetadataCritical` |
| Imports/permissions | Declared import authority and NATS-shaped permission decisions | `ImportNotDeclared`, `PermissionDenied`, `PermissionDeniedByDenyRule`, `ResponseAuthorityExceeded`, `AdvancedCapabilityDenied`, `PermissionCritical` |
| Activation | Trigger source declarations, activation intent creation, authorization, ledger, cursor, dedupe, ack policy, and chain safety | `ActivationConfigInvalid`, `ActivationUnauthorized`, `ActivationSourceUnavailable`, `ActivationCursorFailed`, `ActivationLedgerFailed`, `ActivationDedupeConflict`, `ActivationAckFailed`, `ActivationLoopSuppressed`, `ActivationScheduleLeaseFailed`, `ActivationCritical` |
| Framed stdio RPC protocol | Message framing, JSON-RPC envelope, id correlation, notification rules | `FrameHeaderMalformed`, `ContentLengthMissing`, `ContentLengthInvalid`, `FrameBodyTruncated`, `FrameBodyTooLarge`, `JsonParseFailed`, `RpcInvalidRequest`, `RpcMethodNotFound`, `RpcInvalidParams`, `RpcResponseWithoutRequest`, `RpcDuplicateResponse`, `RpcRequestTimedOut`, `RpcRequestCancelled`, `RpcWriteFailed`, `ProtocolCritical` |
| Process runtime | Child process lifecycle and stream cleanup | `ScriptSpawnFailed`, `ScriptProcessFailed`, `ScriptTimeout`, `ScriptCancelled`, `ProcessCleanupFailed`, `ProcessCritical` |
| Runtime mediation | Facade method dispatch across protocol, permissions, process, and event trail | `ProtocolFrameInvalid`, `ProtocolMessageInvalid`, `ScriptSpawnFailed`, `ScriptProcessFailed`, `ScriptTimeout`, `ScriptCancelled`, `MediationCritical` |
| Attribution event trail | Event payload attribution and NATS publication of execution events | `AttributionMissingField`, `EventPublishFailed`, `EventTrailCritical` |
| Execution exchange | Incoming execution request, final reply, and lower-error resolution | `ExecutionRequestInvalid`, `ExecutionReplyFailed`, `ExecutionCritical` |

Resolve / Transform / Propagate table:

| Ack | Boundary | Lower condition | Policy | Result |
| --- | --- | --- | --- | --- |
| A01 | Substrate | startup, unavailable, cleanup failure | Transform | Matching substrate error |
| A02 | Substrate | unknown thrown value | Transform | `SubstrateCritical` |
| A03 | Record store | missing, stale, revision mismatch, write conflict | Transform | Matching record error |
| A04 | Record store | substrate failure during KV operation | Transform | `RecordPersistenceFailed` with substrate origin |
| A05 | Metadata/schema | placeholder, wildcard, schema, posture violation | Transform | Matching metadata/schema error |
| A06 | Metadata/schema | unknown thrown value | Transform | `MetadataCritical` |
| A07 | Permissions | metadata/schema error while resolving capability | Propagate | Original metadata/schema error |
| A08 | Permissions | undeclared import, deny rule, response escape, missing advanced capability | Transform | Matching permission error |
| A09 | Permissions | unknown thrown value | Transform | `PermissionCritical` |
| A10-ACT | Activation | invalid activation declaration, source shape, or request/reply source subject | Transform | `ActivationConfigInvalid` |
| A11-ACT | Activation | subscribe/exposure denied before source binding | Transform | `ActivationUnauthorized` |
| A12-ACT | Activation | source binding or source client unavailable | Transform | `ActivationSourceUnavailable` |
| A13-ACT | Activation | ledger, cursor, dedupe, ack, or loop policy failure | Transform | Matching activation error |
| A14-ACT | Activation | metadata/schema or permission error while authorizing source | Propagate | Original lower error |
| A15-ACT | Activation | request/reply input accepted | Resolve | `ActivationIntent` with optional reply context |
| A16-ACT | Activation | unknown thrown value | Transform | `ActivationCritical` |
| A10 | Mediation | permission error from facade check | Propagate | Original permission error |
| A11 | Mediation | bad protocol frame or message | Transform | Matching protocol or mediation error |
| A12 | Mediation | spawn failure, throw, timeout, cancellation | Transform | Matching mediation/process error |
| A13 | Mediation | stderr output | Resolve | Diagnostic event only, no protocol failure |
| A14 | Mediation | event trail failure during progress or publish | Propagate | Original event-trail error |
| A15 | Mediation | unknown thrown value | Transform | `MediationCritical` |
| A16 | Event trail | missing attribution field | Transform | `AttributionMissingField` |
| A17 | Event trail | NATS publish or substrate failure | Transform | `EventPublishFailed` |
| A18 | Event trail | unknown thrown value | Transform | `EventTrailCritical` |
| A19 | Execution exchange | invalid execution request | Transform | `ExecutionRequestInvalid` |
| A20 | Execution exchange | lower declared runtime error | Resolve | Typed NATS error reply plus attributed failure event with origin preserved |
| A21 | Execution exchange | reply publication failure | Transform | `ExecutionReplyFailed` |
| A22 | Execution exchange | unknown thrown value | Transform | `ExecutionCritical` |

Protocol contract:

JSON-RPC 2.0 messages are framed as `Content-Length: <utf8-byte-count>\r\n\r\n<body>`. The protocol does not support JSON-RPC batch messages. Request ids are strings or integers; notifications omit `id` and never receive a response. The parser handles partial headers and bodies incrementally, waits for the complete declared body before dispatch, preserves request id type, and fails closed on malformed framing instead of speculative resync.

Protocol tests are owned by the framed stdio RPC layer:

| Group | Required proof |
| --- | --- |
| Framing | UTF-8 byte counts, single and multiple frames, split header/body input, no early dispatch, missing or invalid length, duplicate length, oversized body, mismatched body, malformed header, EOF mid-body |
| JSON-RPC | request dispatch and same-id response, notification dispatch without response, response resolves pending request, invalid `jsonrpc`, unsupported batch, result/error conflict, invalid JSON, invalid params |
| Stderr isolation | plain stderr becomes diagnostics, JSON-like stderr is not parsed as protocol, `Content-Length` on stderr is ignored, interleaved stderr does not corrupt stdout parsing |
| Unknown method | unknown request returns method-not-found, unknown or invalid notification writes no response and records a warning/error hook |
| Correlation | concurrent requests resolve by id, out-of-order responses work, unknown ids and duplicate responses are rejected, id type is preserved |
| Cancellation and timeout | request timeout rejects with `RpcRequestTimedOut`, late response is ignored and recorded, explicit cancellation rejects with `RpcRequestCancelled`, handler receives an abort signal |

Test ownership:

| Test | Owning layer | Assertion |
| --- | --- | --- |
| T01 | Runtime substrate | startup, unavailable, cleanup failures become declared substrate errors |
| T02 | Runtime substrate | unknown substrate exception becomes `SubstrateCritical` |
| T03 | Script record store | missing, stale, revision mismatch, and write conflict become declared record errors |
| T04 | Script record store | substrate failure during KV access becomes `RecordPersistenceFailed` |
| T05 | Metadata/schema | placeholder subjects, invalid wildcards, schema gaps, and unsupported posture are rejected |
| T06 | Metadata/schema | unknown metadata exception becomes `MetadataCritical` |
| T07 | Imports/permissions | metadata/schema errors propagate unchanged |
| T08 | Imports/permissions | undeclared import, deny rule, response escape, and missing advanced capability are rejected |
| T09 | Imports/permissions | unknown permission exception becomes `PermissionCritical` |
| T10-ACT-META | Metadata/schema | activation declarations validate source kind, subject settings, schemas, concrete patterns, and `desc` |
| T10-ACT-PERM | Imports/permissions | activation source subscribe authority is enforced with deny precedence |
| T10-ACT-INTENT | Activation | request/reply input becomes `ActivationIntent` with source identity, dedupe key, reply context, and chain bounds |
| T10-ACT-ERR | Activation | invalid or over-limit activation input becomes activation-owned errors |
| T10 | Runtime mediation | facade permission failures propagate unchanged |
| T11 | Framed stdio RPC | bad frame or JSON-RPC message becomes a protocol error |
| T12 | Process runtime | spawn failure, nonzero exit, timeout, and cancellation are typed |
| T13 | Runtime mediation | stderr is diagnostics only and does not fail protocol execution |
| T14 | Runtime mediation | event trail failure during script progress/publish propagates unchanged |
| T15 | Runtime mediation | unknown mediation exception becomes `MediationCritical` |
| T16 | Attribution event trail | missing attribution field becomes `AttributionMissingField` |
| T17 | Attribution event trail | NATS event publish failure becomes `EventPublishFailed` |
| T18 | Attribution event trail | unknown event-trail exception becomes `EventTrailCritical` |
| T19 | Execution exchange | invalid execution request becomes `ExecutionRequestInvalid` |
| T20 | Execution exchange | lower declared runtime error resolves to typed NATS error reply and failure event |
| T21 | Execution exchange | reply publish failure becomes `ExecutionReplyFailed` |
| T22 | Execution exchange | unknown execution exception becomes `ExecutionCritical` |

Historical vertical proof suite:

| Case | Required shape |
| --- | --- |
| Success exact revision | Historical Bun proof started embedded JetStream, created KV bucket `TB_SCRIPT_RECORDS_PROOF`, stored `scripts.proof.echo` revision 1 and revision 2, executed revision 1 through `tb.proof.runtime.execute`, and asserted revision 1 behavior, progress, allowed publish, reply, event trail, and cleanup |
| Invalid metadata | Store malformed metadata with invalid wildcard `tb.proof.bad.>.tail`, execute through NATS, reject before spawn, and emit attributed failure |
| Missing record | Execute an absent KV key, return typed missing-record reply, avoid spawn, and emit failure event |
| Revision mismatch | Request a revision that cannot resolve for the key and prove no fallback to latest |
| Denied publish | Script asks the facade to publish `tb.proof.out.denied.exec_denied_publish_001`; runtime denies it, emits typed failure, and no message appears on the denied subject |
| Denied import | Script requests undeclared raw NATS access; runtime denies through mediation |
| Script failure | Script writes stderr then exits nonzero or throws; runtime returns typed script failure and emits event |
| Recovery | After denial or failure, execute a known-good script on the same runtime fixture and assert success |
| Cleanup | Every path ends child processes, drains NATS clients/subscriptions, stops the server, and leaves the temp store isolated or removed |

Vertical proof fixtures:

| Surface | Concrete value |
| --- | --- |
| Request subject | `tb.proof.runtime.execute` |
| Allowed publish subject | `tb.proof.out.allowed.exec_success_001` |
| Denied publish subject | `tb.proof.out.denied.exec_denied_publish_001` |
| Progress subjects | `tb.proof.exec.exec_success_001.progress`, `tb.proof.exec.exec_throwing_001.progress` |
| Event subjects | `tb.proof.exec.exec_success_001.event`, `tb.proof.exec.exec_invalid_metadata_001.event`, `tb.proof.exec.exec_missing_record_001.event`, `tb.proof.exec.exec_revision_mismatch_001.event`, `tb.proof.exec.exec_denied_publish_001.event`, `tb.proof.exec.exec_denied_import_001.event`, `tb.proof.exec.exec_throwing_001.event`, `tb.proof.exec.exec_recovery_001.event` |
| Permission allow patterns | `tb.proof.out.allowed.>`, `tb.proof.exec.*.progress`, `tb.proof.exec.*.event` |
| Permission deny patterns | `tb.proof.out.denied.>` |
| Schema ids | `tb.schema.proof.script_record.v1`, `tb.schema.proof.metadata.v1`, `tb.schema.proof.input.echo.v1`, `tb.schema.proof.output.echo.v1`, `tb.schema.proof.ipc.progress.v1`, `tb.schema.proof.ipc.publish_request.v1`, `tb.schema.proof.event.execution.v1` |

## Verification Strategy

The implementation plan must write tests in dependency order. A higher-layer test may mock only the declared lower-layer contract and typed errors; it may not inspect lower-layer internals.

The historical vertical proof ran last and was allowed to use real Bun, `@lagz0ne/nats-embedded`, JetStream KV, NATS request/reply, and a representative script process because all lower contracts had already been pinned. The current platform vertical proof must be re-owned by the Go substrate lane.

Every RED step must capture:

- command.
- expected failure.
- owning layer.
- typed error or missing contract.

Every GREEN step must capture:

- passing command.
- test names/subcases.
- evidence that no broader NATS access, placeholder subject, or happy-path-only behavior slipped in.

## Escalation Log

Resolved for the first implementation slice:

- Framed stdio RPC is JSON-RPC 2.0 over `Content-Length` framing.
- `allow_responses` is bounded to the invocation/reply contract. Scripts do not publish replies directly in the base path.
- Execution is not successful until the final success or failure event has been attempted. If the event trail fails before the caller reply is produced, the caller receives a typed event-trail failure with origin preserved.

Open but not blocking:

- Whether future direct NATS advanced capability uses NATS dynamic response permissions directly or a stricter runtime-generated publish rule.
- Whether failed post-reply telemetry receives a degraded local diagnostic sink when a caller reply has already succeeded.

Blocking before coding:

- Starting implementation without a RED test for each owned layer.
- Adding runtime breadth before the vertical proof is strict.
- Letting the vertical proof test lower-layer internals instead of their declared contracts.
- Starting process runtime or framed stdio work before activation metadata, permission, and intent contracts are pinned.
- Implementing schedule activation before durable state, lease, catch-up, and fake-clock contracts exist.
