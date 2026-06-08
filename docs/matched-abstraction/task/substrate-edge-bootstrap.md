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

Implementation gate:

- `bun run schema:parity`
- Go substrate-edge targeted tests.
- Browser Edge targeted tests.
- `bun run test`
- `bun run typecheck`
- `bun run build`
- `bun run pack:dry`
- `bun run validate:layers`
- `bun run test:layers`
- no-slop scan over substrate-edge docs, fixtures, and code.

## Wrap-Up Announcement

The `substrate-edge-bootstrap` milestone is complete when Go can derive scoped substrate-edge bootstrap descriptors from canonical contracts, Browser Edge can consume the bootstrap without leaking raw authority to generated content, and canonical browser commands can cross into Command Acceptance through verified contract shapes.
