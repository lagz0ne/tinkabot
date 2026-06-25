---
layer: task
topic: nats-script-runtime-metadata-permissions
references:
  - ../plan/nats-script-runtime-traced-tdd.md
  - ../plan/nats-script-runtime.md
  - ../approach/nats-script-runtime.md
---

# NATS Script Runtime Metadata And Permissions Task

## Task Brief

Implement the traced-TDD metadata/schema and imports/permissions slice. This task covers T05 through T09 only. IPC, process runtime, execution exchange, event publication, and vertical script execution remain outside this slice.

Scope includes metadata validation, subject pattern helpers, permission resolution, typed error expansion, and tests under `tests/nats-script-runtime/`.

## Acceptance Contract

The task is accepted when metadata/schema errors and imports/permissions errors are machine-inspectable through `TinkabotRuntimeError`, NATS subject declarations reject placeholders and invalid wildcards, schema ids are checked against their declared surfaces, untrusted scripts are rejected while sandbox enforcement is absent, metadata/schema failures propagate through permission resolution unchanged, deny rules beat allow rules, response authority is bounded, and raw NATS access requires explicit advanced capability.

Required observed behavior:

- T05: placeholder subjects, invalid wildcards, schema gaps, schema mismatch, malformed metadata, and unsupported security posture are rejected by `MetadataValidator`.
- T06: unknown metadata validation values become `MetadataCritical`.
- T07: metadata/schema errors propagate unchanged through `PermissionResolver`.
- T08: declared imports and NATS wildcard permissions are enforced, including deny-over-allow, response bounds, and advanced raw NATS gating.
- T09: unknown permission failures become `PermissionCritical`.

## RED Artifact

- `bun test tests/nats-script-runtime/metadata-validator.test.ts tests/nats-script-runtime/permission-resolver.test.ts` -> failed because `MetadataValidator` and `PermissionResolver` were not exported.

## Execution Notes

The GREEN pass added:

- `src/nats-script-runtime/metadata-validator.ts` for metadata shape, schema id, security posture, and subject declaration validation.
- `src/nats-script-runtime/permission-resolver.ts` for import authority, publish permission, response authority, and advanced capability checks.
- `src/nats-script-runtime/subjects.ts` for shared NATS subject pattern validation and matching.
- metadata and permission error kinds in `src/nats-script-runtime/errors.ts`.
- exports from `src/nats-script-runtime/index.ts`.
- `tests/nats-script-runtime/metadata-validator.test.ts` for T05 and T06.
- `tests/nats-script-runtime/permission-resolver.test.ts` for T07 through T09.

Review kept NATS effects outside this slice. The permission resolver decides authority only; mediation and vertical proof will verify actual publish side effects later.

## Verification Evidence

- `bun test tests/nats-script-runtime/metadata-validator.test.ts tests/nats-script-runtime/permission-resolver.test.ts` -> `8 pass, 0 fail`.
- `bun test` -> `16 pass, 0 fail`.
- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit` completed successfully.
- `bun run build` -> emitted ESM, CommonJS, and declaration artifacts.
- `bun pm pack --dry-run` -> `Total files: 5`.
- `find . -type d -name __pycache__ -print` -> no matches.

## Wrap-Up Announcement

The final response must state that T05 through T09 are implemented and verified, and that the trigger/activation layer needs a Plan-level decision before it is inserted into the next implementation order.
