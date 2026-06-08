---
layer: task
topic: nats-script-runtime-activation-contract
references:
  - ../approach/nats-script-runtime.md
  - ../plan/nats-script-runtime.md
  - ../plan/nats-script-runtime-traced-tdd.md
---

# NATS Script Runtime Activation Contract Task

## Objective

Add the bounded activation contract needed before reactive NATS triggers: metadata activation exposure validation, subscribe/source permission enforcement, typed activation errors, and request/reply `ActivationIntent` normalization.

## Exact Scope

- Extend metadata validation with `nats.activations`.
- Support `request_reply` activation source now.
- Add outside-in exposure metadata using concrete NATS subjects and wildcard-aware validation where relevant.
- Add subscribe and activation source permission checks using existing NATS auth vocabulary: `allow`, `deny`, and deny precedence.
- Add typed activation error layer/kinds.
- Add `ActivationIntent` type and request/reply factory with source identity, concrete request subject, optional reply context, chain fields, dedupe key, hop bounds, and observed timestamp.

## Non-Goals

- No real NATS subscriptions.
- No JetStream consumer.
- No KV watch.
- No schedule provider.
- No process/script execution runtime.
- No raw NATS client exposure.

## Acceptance Contract

- Metadata accepts a valid `request_reply` activation declaration.
- Metadata rejects malformed activation declarations.
- Request/reply activation subjects must be concrete inbound subjects.
- Permission resolver enforces activation source authority through `permissions.subscribe`, with deny-over-allow.
- `ActivationIntent` normalizes request/reply input and preserves optional reply context.
- Intent validation rejects missing source identity, invalid dedupe key, invalid hop state, and exceeded hop limit.
- Typed errors identify activation failures without collapsing into substrate or execution errors.
- Existing T01-T09 tests remain green.
- Layer validation remains green.

## RED Artifact

| Test id | Owning layer | RED assertion |
| --- | --- | --- |
| T10-ACT-META-001 | Metadata/schema | Valid `nats.activations.request` metadata is accepted |
| T10-ACT-META-002 | Metadata/schema | Activation with missing `kind` is rejected |
| T10-ACT-META-003 | Metadata/schema | Request/reply activation without a concrete subject is rejected |
| T10-ACT-META-004 | Metadata/schema | `desc` and exposure notes are descriptive, not authority |
| T10-ACT-PERM-001 | Imports/permissions | Activation source is allowed when subscribe allow matches |
| T10-ACT-PERM-002 | Imports/permissions | Activation source is rejected when subscribe deny matches |
| T10-ACT-PERM-003 | Imports/permissions | Activation source is rejected when no subscribe allow matches |
| T10-ACT-INTENT-001 | Activation | Request/reply input builds an `ActivationIntent` |
| T10-ACT-INTENT-002 | Activation | Optional reply context is preserved |
| T10-ACT-INTENT-003 | Activation | Dedupe key is stable for the same source identity and request id |
| T10-ACT-INTENT-004 | Activation | Exceeded `maxHops` is rejected |
| T10-ACT-ERR-001 | Activation | Activation failures use activation-owned error kinds |

## Execution Notes

The RED pass added tests before runtime code. The GREEN pass added:

- `src/nats-script-runtime/activation-intent.ts` for request/reply `ActivationIntent` normalization and activation-owned errors.
- `nats.activations` metadata validation for request/reply exposure.
- `PermissionResolver.assertSubscribe` and `PermissionResolver.assertActivationSource`.
- activation layer and error kinds in `src/nats-script-runtime/errors.ts`.
- public exports from `src/nats-script-runtime/index.ts`.

This slice stayed contract-only. It did not expand `RuntimeSubstrate`, create live NATS listeners, implement KV watches, implement JetStream consumers, or add schedule behavior.

## Verification Evidence

- `bun test tests/nats-script-runtime/metadata-validator.test.ts tests/nats-script-runtime/permission-resolver.test.ts tests/nats-script-runtime/activation-intent.test.ts` -> RED failed with `8 pass, 3 fail, 1 error`.
- `bun test tests/nats-script-runtime/metadata-validator.test.ts tests/nats-script-runtime/permission-resolver.test.ts tests/nats-script-runtime/activation-intent.test.ts` -> GREEN `12 pass, 0 fail`.
- `bun test` -> `20 pass, 0 fail`.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit`.
- `bun run build` -> emitted ESM, CommonJS, and declaration artifacts.
- `bun pm pack --dry-run` -> `Total files: 5`.
- `python3 -B .codex/skills/matched-abstraction-thinking/scripts/validate_layers.py docs/matched-abstraction` -> `Layer validation passed: docs/matched-abstraction`.
- `python3 -B -m unittest tests/test_validate_layers.py` -> `Ran 10 tests ... OK`.

## Wrap-Up Announcement

Activation contract is ready for the next runtime slice when metadata, permissions, typed activation errors, and request/reply `ActivationIntent` are implemented and verified.
