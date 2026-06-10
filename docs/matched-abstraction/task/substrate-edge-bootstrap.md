---
layer: task
topic: substrate-edge-bootstrap
references:
  - ../approach/endgame-app.md
  - ../plan/endgame-app.md
  - ./endgame-contract-authority.md
  - ./managed-auth-subjects.md
  - ./command-acceptance.md
---

# Substrate Edge Bootstrap Task

Diagram: https://diashort.apps.quickable.co/d/8e1c7e86

## Objective

Create the first pure substrate/browser-edge bootstrap boundary over the shared endgame contract packet: Go consumes canonical auth and artifact contracts into a scoped worker credential descriptor plus artifact gateway policy shape, and the trusted Browser Edge consumes that bootstrap without exposing raw authority to generated content.

## Scope

This task owns:

- Go contract consumption for `auth.policy` and `artifact.manifest`.
- lease, revocation, expiration, and provenance denial before credential descriptor creation.
- worker-only browser credential descriptor shape.
- content-safe browser bootstrap context shape.
- artifact gateway manifest policy shape with digest, object namespace, CSP, frame, and sandbox requirements.
- Browser Edge conversion from generated-content intent to canonical `browser.command_intent`.
- proof that generated content output contains no raw NATS, subject, token, credential, permission, publish, or subscribe vocabulary.

## Non-Goals

- No live NATS WebSocket.
- No Vite UI.
- No HTTP artifact serving.
- No script execution.
- No activation worker.
- No materialization loop.
- No Docker sandboxing.
- No release-ready outside-in browser proof.

## Acceptance Contract

- Go bootstrap consumes valid shared fixtures and preserves schema revision, app revision, principal, session, lease, artifact revision, and policy context.
- Revoked, expired, and provenance-mismatched capabilities fail before credential descriptor creation.
- Worker bootstrap output may carry scoped credential descriptors; content bootstrap output must carry only sanitized context.
- Browser Edge canonicalizes content intent into `browser.command_intent` and proves it parses through Contract Authority.
- The old frontend-local `frontend.command_intent` shape does not cross the substrate edge.
- Artifact gateway policy accepts the valid manifest and rejects digest mismatch, object refs outside the allowed artifact namespace, and missing CSP/frame/sandbox policy.
- Revocation invalidates the worker credential path and returns a sanitized attributed denial to generated content.

## RED Artifact

Expected failing tests before implementation:

- `T-SUBSTRATE-CONTRACT-CONSUME`: Go bootstrap consumes canonical auth policy plus artifact manifest fixtures and preserves provenance/context.
- `T-SUBSTRATE-LEASE-DENY`: revoked, expired, and provenance-mismatched capability fixtures deny before any credential descriptor is created.
- `T-EDGE-CREDENTIAL-SPLIT`: Browser Edge bootstrap output separates worker-only credential descriptor from content-safe context and deep-scan denies raw authority vocabulary in content output.
- `T-EDGE-COMMAND-CANONICAL`: Browser Edge converts generated-content intent into canonical `browser.command_intent` accepted by Contract Authority and Command Acceptance.
- `T-GATEWAY-MANIFEST-POLICY`: artifact gateway policy accepts valid manifest and rejects digest mismatch, outside namespace, and missing CSP/frame/sandbox policy.
- `T-EDGE-REVOKE`: revocation blocks later command publish and returns sanitized denial to content.

## Execution Notes

Keep this slice pure and fakeable. Go owns substrate-edge policy shape and denial. TypeScript owns trusted Browser Edge bootstrap consumption and canonical command bridging. Shared schemas remain the contract authority.

Do not introduce live transport, servers, UI rendering, or runtime execution in this task. Later milestones can attach these pure shapes to NATS WebSocket, Vite, object storage, activation, and materialization.

## Verification Evidence

Task prep evidence:

- `curl -s -X POST https://diashort.apps.quickable.co/render ...` -> `https://diashort.apps.quickable.co/d/8e1c7e86`
- `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`
- `bun run test:layers` -> `Ran 10 tests ... OK`

RED:

- `bun test packages/sdk/tests/endgame-contract/substrate-edge-bootstrap.test.ts` -> failed with missing `createBrowserEdgeBootstrap` export.
- `go test ./edge` from `substrate/go` -> failed with missing `BuildBootstrap`, `BootstrapOptions`, `ErrorKind`, and `EdgeError`.

GREEN:

- `bun test packages/sdk/tests/endgame-contract/substrate-edge-bootstrap.test.ts` -> `4 pass`, `0 fail`, `17 expect() calls`.
- `go test ./edge` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/edge`.
- `bun run schema:parity` -> endgame contract tests `21 pass`, `0 fail`, `152 expect() calls`; Go contract and edge packages `ok`.
- `bun run typecheck` -> SDK plus orchestrator typecheck passed.
- `bun run test` -> `52 pass`, `0 fail`, `334 expect() calls`.
- `bun run build` -> SDK ESM, CommonJS, and declarations emitted.
- `bun run pack:dry` -> `tinkabot-0.1.0.tgz`, 6 files, unpacked size `179.68KB`.
- `bun run validate:layers` -> `Layer validation passed: docs/matched-abstraction`.
- `bun run test:layers` -> `Ran 10 tests ... OK`.
- no-slop scan over substrate-edge docs, fixtures, and code -> only intentional handoff vocabulary.

Named negative-case evidence (re-executed 2026-06-10 during the release-spine evidence audit):

- `go test ./edge -run TestSubstrateDeniesLeaseBeforeCredentialDescriptor -v -count=1` from `substrate/go` -> `--- PASS: TestSubstrateDeniesLeaseBeforeCredentialDescriptor` with named subtests `revoked`, `expired`, and `provenance`: the substrate denies the lease before any credential descriptor is returned to a neighbor.

Review passes:

- No-slop pass: no live NATS WebSocket, Vite UI, HTTP serving, script execution, activation worker, materialization loop, or Docker sandboxing was added.
- Simplify pass: Go owns substrate-edge derivation and gateway denial; TypeScript owns Browser Edge bootstrap and canonical command bridging; both stay pure/fakeable.
- Review pass: empty credential refs are denied, revoked leases fail before worker credentials return, and default artifact namespace enforcement is active.

## Wrap-Up Announcement

Shipped: Go derives scoped substrate-edge bootstrap descriptors from canonical contracts, Browser Edge consumes the bootstrap without leaking raw authority to generated content, and canonical browser commands cross into Command Acceptance through verified contract shapes. Evidence recorded above.
