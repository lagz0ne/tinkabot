---
layer: task
topic: managed-auth-subjects
references:
  - ../approach/endgame-app.md
  - ../plan/endgame-app.md
  - ./endgame-contract-authority.md
---

# Managed Auth Subjects Task

## Objective

Prove that managed auth and subject taxonomy consume the `endgame-contract-authority` packet and compile domain identity/capability fixtures into NATS-auth-shaped permissions, imports, exports, leases, revocation, and denied-neighbor behavior without losing provenance.

## Scope

This task owns:

- identity, principal, session, lease, revision, and capability provenance checks.
- subject taxonomy for control-plane, app-plane, script facade, activation, materializer, system, and internal surfaces.
- NATS-auth-shaped compile output for `permissions.publish`, `permissions.subscribe`, `allow`, `deny`, imports, paired exports/exposure, and bounded response authority.
- denied-neighbor behavior for adjacent subjects, reserved surfaces, wildcard overreach, and namespace collisions.
- lease, expiration, and revocation proof at the policy compile layer.
- typed auth/subject failures matching the Endgame Plan.
- policy sidecar fixtures layered on the existing contract-authority packet.

## Non-Goals

- No live NATS server auth enforcement.
- No credential minting implementation.
- No Browser Edge session bootstrap.
- No Go substrate runtime or gateway implementation.
- No Vite shell, worker, or sub-app implementation.
- No command ledger, activation worker, script runtime, or materializer implementation.
- No provider-specific auth backend as domain truth.
- No new schema authority beyond consuming the contract-authority packet.

## Acceptance Contract

- Valid policy fixtures compile into NATS-auth-shaped permissions/imports/exports with provenance preserved.
- Schema-valid but unauthorized fixtures are denied by capability policy, proving shape validity is not authority.
- Deny beats allow for overlapping subject permissions.
- Wildcards are accepted only behind concrete authoritative prefixes.
- Placeholder subjects and reserved control-plane surfaces are rejected.
- Browser and script base policies cannot access raw NATS or control-plane surfaces by default.
- Lease revocation causes later policy use to fail with revocation attribution.
- Stale revision capability use fails closed with exact stale/mismatch attribution.
- Compile output carries identity, session, lease status, capability, schema revision, app revision, and subject-taxonomy context.
- Typed failures cover denied capability, advanced capability, unbounded response authority, revoked lease, expired lease, stale revision, provenance loss, reserved surface violation, wildcard overreach, namespace collision, import/export mismatch, missing exposure subject, and denied-neighbor violation.

## RED Artifact

Add failing managed-auth/subject tests proving the current system cannot yet consume the contract packet into an authoritative policy compile result.

Expected RED failure:

- auth compiler/export is missing.
- subject taxonomy resolver/export is missing.
- positive policy fixture cannot compile.
- denied-neighbor fixture is not rejected.
- revoked lease fixture is not rejected.
- stale revision fixture is not rejected.
- provenance fields are absent from compile output or denial output.

## Execution Notes

Keep this slice inside-out and contract-level. It may compile policy fixtures into NATS-auth-shaped data, but it must not claim live NATS enforcement.

Use the existing contract-authority packet as input. Do not invent lane-local identity, revision, subject, or capability shapes.

Treat auth and subject taxonomy as one milestone because NATS permissions are unsafe without namespace and denied-neighbor semantics.

## Verification Evidence

RED:

- Command: `bun test packages/sdk/tests/endgame-contract/managed-auth-subjects.test.ts`
- Expected failure: `Export named 'assertSubscribe' not found in module ... packages/sdk/src/index.ts`.
- Missing contract proven: managed-auth compiler and subject taxonomy exports are absent.

- Command: `bun run schema:parity`
- Expected failure: `expected action to fail` for `denied-browser-raw-nats-cased`.
- Missing contract proven: contract parser/schema raw-authority denial is weaker than the frontend mediator for cased raw subject keys.

- Sidecar review blockers: export/exposure pairing was not checked, advanced exposure kinds compiled by schema alone, present `allow_responses` could omit `expiresMs`, and expired leases mapped to `RevokedLease` without `leaseStatus` detail.

- Command: `bun test packages/sdk/tests/endgame-contract/managed-auth-subjects.test.ts`
- Expected failure after blocker tests: `3 pass`, `3 fail`; lease denial details lacked `leaseStatus`, advanced capability denial details lacked `leaseStatus`, and export/exposure mismatch compiled instead of failing.
- Missing contract proven: schema-valid policy packets were still stronger than capability policy in uncovered auth paths.

GREEN:

- Schema parity: `bun run schema:parity` -> contract tests `8 pass`, `0 fail`; Go contract package `ok`.
- Managed auth targeted tests: `bun test packages/sdk/tests/endgame-contract/managed-auth-subjects.test.ts` -> `6 pass`, `0 fail`, `41 expect() calls`.
- Subject taxonomy targeted tests: `T-SUBJECT-TAXONOMY` classifies app/control subjects and rejects reserved control grants plus overbroad wildcards.
- Denied-neighbor fixtures: `auth-policy.json` allows `tb.proof.out.allowed.>` but denies `tb.proof.out.denied.>`; `assertPublish` returns `PermissionDeniedByDenyRule` for the denied family and `PermissionDenied` for adjacent runtime publish.
- Revoked lease fixtures: `auth-policy-revoked-lease.json` is schema-valid and compile-denied with `RevokedLease`.
- Expired lease fixtures: `auth-policy-expired-lease.json` is schema-valid and compile-denied with `ExpiredLease` plus `leaseStatus`.
- Stale revision fixtures: `auth-policy-provenance-mismatch.json` is schema-valid and compile-denied with `StaleRevision`.
- Advanced capability fixtures: `auth-policy-advanced-import.json` and `auth-policy-advanced-exposure.json` are schema-valid and compile-denied with `AdvancedCapabilityDenied`.
- Response bound fixtures: `auth-policy-unbounded-response.json` is schema-valid and compile-denied with `ResponseAuthorityUnbounded`.
- Export/exposure fixtures: missing export, missing exposure, and missing exposure subject fixtures are schema-valid and compile-denied with `ImportExportMismatch` or `ExposureSubjectMissing`.
- Provenance preservation: `auth-policy.compiled-nats.json` carries provenance, capability, permission, import, export, exposure, and subject groups from the source policy.
- No raw browser/script authority fixtures: `browser-command-raw-nats-cased.json` is rejected by both TypeScript/Zod and Go schema validation.

General checks:

- `bun run typecheck` -> `bunx @typescript/native-preview --noEmit`.
- `bun run test` -> `35 pass`, `0 fail`, `237 expect() calls`.
- `bun run build` -> SDK ESM, CommonJS, and declarations emitted.
- `bun run pack:dry` -> `Total files: 6`, unpacked size `139.91KB`.

## Wrap-Up Announcement

The `managed-auth-subjects` milestone proves that domain identity and capability policy compile into NATS-auth-shaped authority with subject taxonomy, denial, lease, revocation, stale-revision, and provenance behavior preserved. Later substrate, Browser Edge, command acceptance, activation, script runtime, and materializer lanes can consume this policy boundary without inventing their own authority model.
