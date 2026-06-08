---
layer: task
topic: nats-script-runtime-request-reply-activation-adapter
references:
  - ../approach/nats-script-runtime.md
  - ../plan/nats-script-runtime.md
  - ../plan/nats-script-runtime-traced-tdd.md
  - ./nats-script-runtime-activation-contract.md
---

# NATS Script Runtime Request/Reply Activation Adapter Task

## Objective

Implement the bounded request/reply activation adapter that converts an inbound request/reply activation envelope into an authorized `ActivationIntent`.

## Scope

- Add a focused adapter module for request/reply activation.
- Require a declared activation name.
- Require the activation declaration to exist in script metadata.
- Require the inbound request subject to match the declared activation subject.
- Call `PermissionResolver.assertActivationSource` before intent creation.
- Convert authorization failures into activation-layer errors.
- Preserve typed non-authorization runtime errors from lower owners.
- Wrap unknown failures as `ActivationCritical`.
- Preserve reply subject and inbound headers in the resulting `ActivationIntent`.

## Non-Goals

- No NATS server startup.
- No NATS subscription binding.
- No script execution.
- No process runtime work.
- No KV watch.
- No JetStream consumer.
- No schedule provider.
- No activation ledger, cursor, ack, retry, restart, or durable dedupe implementation.

## Acceptance Contract

- Adapter returns a valid `ActivationIntent` for an authorized request/reply envelope.
- Adapter rejects envelopes without `activationName`.
- Adapter rejects unknown activation names.
- Adapter rejects inbound subjects that do not match the declared activation subject.
- Adapter calls `PermissionResolver.assertActivationSource` with the declared activation source and observed subject.
- Adapter returns `ActivationUnauthorized` for `PermissionDenied` and `PermissionDeniedByDenyRule`.
- Adapter preserves typed non-authorization runtime errors from lower owners.
- Adapter wraps unknown thrown values as `ActivationCritical`.
- Adapter preserves `replySubject`, payload, headers, chain fields, and dedupe identity.

## RED Artifact

| Test id | Owning layer | RED assertion |
| --- | --- | --- |
| T11-RR-AUTH-001 | Activation | Valid exposure plus subscribe allow creates an intent |
| T11-RR-AUTH-002 | Activation | Missing exposure becomes `ActivationUnauthorized` |
| T11-RR-AUTH-003 | Activation | Observed subject mismatch becomes `ActivationUnauthorized` |
| T11-RR-AUTH-004 | Activation | Subscribe deny beats allow and becomes `ActivationUnauthorized` |
| T11-RR-AUTH-005 | Activation | No subscribe allow becomes `ActivationUnauthorized` |
| T11-RR-INTENT-001 | Activation | Payload, headers, source identity, chain fields, and dedupe key are preserved |
| T11-RR-INTENT-002 | Activation | Optional reply context is preserved only when present |
| T11-RR-ERR-001 | Activation | Invalid adapter input becomes `ActivationConfigInvalid` |
| T11-RR-ERR-002 | Activation | Hop-limit violation remains `ActivationLoopSuppressed` |
| T11-RR-ERR-003 | Activation | Metadata/schema lower errors propagate unchanged |
| T11-RR-ERR-004 | Activation | Non-authorization permission lower errors propagate unchanged |
| T11-RR-ERR-005 | Activation | Unknown thrown values become `ActivationCritical` |

## Execution Notes

The adapter receives an already-delivered request/reply envelope and script metadata. It does not bind transport. Request/reply dedupe key is produced but not durably enforced in this slice.

## Verification Evidence

- `bun test tests/nats-script-runtime/request-reply-activation-adapter.test.ts` -> RED failed with `0 pass, 1 fail, 1 error` because `activateRequestReply` was not exported.
- `bun test tests/nats-script-runtime/request-reply-activation-adapter.test.ts` -> `3 pass, 0 fail`.
- `bun test` -> `23 pass, 0 fail`.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit`.
- `bun run build` -> emitted ESM, CommonJS, and declaration artifacts.
- `bun pm pack --dry-run` -> `Total files: 5`.
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.

## Wrap-Up Announcement

Request/reply activation adapter is complete when an inbound authorized request/reply envelope produces a valid `ActivationIntent`, authorization failures are translated to `ActivationUnauthorized`, lower-owner typed errors remain visible, and no transport binding or script execution behavior has been added.
