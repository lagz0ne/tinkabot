---
layer: task
topic: go-substrate-core
references:
  - ../approach/go-substrate.md
  - ../plan/go-substrate.md
  - ./endgame-contract-authority.md
  - ./managed-auth-subjects.md
  - ./command-acceptance.md
  - ./substrate-edge-bootstrap.md
---

# Go Substrate Core Task

Diagram: https://diashort.apps.quickable.co/d/4a99eb1d

## Objective

Establish the first Go-owned substrate core boundary so the embedded-NATS adapter, activation, script, and materializer work consume Go substrate contracts instead of TypeScript runtime assumptions. This task turns Go from schema/edge proof into the owner of embedded NATS lifecycle shape, HA/scale topology envelope, auth render shape, credential lease semantics, activation ledger shape, process boundary config, gateway substrate config, and typed substrate errors.

## Scope

This task owns one executable unit: the fakeable Go substrate core contract. Its acceptance surface includes:

- Go substrate core package shape.
- typed Go substrate error families and origin fields.
- lifecycle facade for embedded NATS start/stop/health, topology mode, clustering posture, JetStream readiness, replica/quorum posture, route/gateway/leaf descriptors, and WebSocket listener posture.
- NATS-auth-shaped render output for principals, permissions, imports, exports, bounded responses, and lease provenance.
- credential lease mint/revoke facade for browser and script principals.
- store substrate facade for KV/Object/Stream access, bucket/key checks, revisions, stream positions, and typed durable error mapping.
- activation ledger facade for activation id, dedupe key, chain, status, and replay position.
- process boundary config for command, cwd, env, framed stdio RPC mode, timeout, kill, cleanup, identity, and later Docker placement.
- gateway substrate config for server-side artifact namespace, digest, CSP/frame/sandbox/cache enforcement, MIME policy, and Browser Edge handoff.
- attribution envelope emitted by auth, ledger, process, gateway, and cleanup decisions.

## Non-Goals

- No live embedded NATS adapter or full live cluster orchestration beyond a fakeable lifecycle and topology contract.
- No real account JWT/operator auth backend.
- No activation-source router for request/reply, subject, KV/Object/Stream, or schedule triggers.
- No script execution.
- No materialized projection writes.
- No Vite/browser UI.
- No Docker sandbox implementation.
- No release-level outside-in proof.

## Acceptance Contract

- Go substrate core can build a substrate plan from canonical auth, artifact, activation, and NATS topology inputs without TypeScript runtime authority.
- Embedded NATS topology output distinguishes single-node and HA/scale modes through NATS-native clustering, JetStream replica/quorum, route/gateway/leaf, WebSocket, and degraded-readiness semantics.
- Revoked, expired, stale, malformed, denied-neighbor, wildcard-overreach, duplicate, topology-invalid, and quorum-unavailable cases fail before leases, ledger acceptance, process config, or gateway authority are returned.
- Auth render output is NATS-auth-shaped and preserves provenance.
- Browser and script credential leases are scoped, revocable, and attributed.
- Store substrate output preserves bucket/key identity, revision checks, stream positions, and durable typed errors.
- Activation ledger accepts first activation, resolves duplicates, rejects stale chain state, and records loop suppression.
- Process boundary config is sandbox-ready: command, cwd, env projection, framed stdio RPC, timeout, resource envelope, cancel/kill, cleanup, identity, and attribution are explicit.
- Gateway substrate config enforces artifact namespace, digest, object-store read authority, CSP, frame, sandbox, MIME, cache policy, lease binding, and Browser Edge handoff.
- Every failure belongs to a declared family and carries layer, operation, and enough provenance for the next layer to transform or resolve.

## Error Families

| Family | Required kinds |
| --- | --- |
| Core lifecycle | config invalid, start failed, NATS unavailable, cluster route unavailable, JetStream unavailable, replica policy invalid, quorum unavailable, drain failed, shutdown failed, substrate critical |
| Auth render | render invalid, wildcard overreach, permission compile failed, lease mint denied, lease revoked, lease expired |
| Store substrate | bucket missing, key missing, revision mismatch, write conflict, deleted record, cursor failure |
| Activation ledger | duplicate activation, stale chain, loop suppressed, lease acquire failed, replay cursor failed |
| Process boundary | config invalid, start failed, protocol unavailable, resource denied, timeout, cancel failed, kill failed, cleanup failed |
| Gateway substrate | artifact missing, digest mismatch, namespace denied, object read denied, MIME denied, CSP/frame/sandbox missing, cache policy invalid, lease denied |
| Attribution trail | attribution missing, event write failed, unknown transformed to critical |

## RED Artifact

Expected failing tests before implementation:

- `T-GO-CORE-LIFECYCLE`: lifecycle facade reports embedded NATS topology, JetStream readiness, replica/quorum posture, degraded readiness, route/gateway/leaf/WebSocket posture, drain/shutdown/rejoin errors, and substrate critical errors through typed Go substrate errors.
- `T-GO-AUTH-RENDER`: managed auth fixture renders NATS-shaped auth output with provenance; invalid render, wildcard overreach, permission compile failure, revoked/expired/stale fixtures deny before credential lease output.
- `T-GO-CRED-LEASE`: browser and script leases mint with scoped authority, revoke idempotently, and deny use after revocation.
- `T-GO-STORE-SUBSTRATE`: KV/Object/Stream facade checks buckets, keys, revisions, write conflicts, deleted records, stream cursors, fake durable positions, and typed substrate errors.
- `T-GO-ACT-LEDGER`: activation ledger accepts first activation, resolves duplicate dedupe key, rejects stale chain, records loop suppression, denies lease acquisition failure, and reports replay cursor failure.
- `T-GO-PROC-BOUNDARY`: process boundary config requires command, cwd, env projection, framed stdio RPC mode, timeout, resource envelope, cancel/kill, cleanup, identity, and attribution without executing scripts.
- `T-GO-GATEWAY-SUBSTRATE`: gateway substrate config consumes artifact manifest and rejects artifact missing, digest mismatch, namespace denial, object read denial, MIME denial, CSP/frame/sandbox absence, cache policy failure, and lease denial.
- `T-GO-ATTRIBUTION`: auth, ledger, process, gateway, and cleanup failures emit typed attribution events.

## Execution Notes

Keep this slice pure and fakeable. Define Go substrate contracts and fakes that preserve live embedded NATS/process semantics, including NATS-native HA/scale posture, then prove them with inside-out Go tests. Do not start the embedded-NATS adapter, activation-source routers, scripts, or materializers from this task.

Use the existing `contract` and `edge` packages as inputs. If a shape belongs to schema or managed auth, consume it; do not recreate authority in Go-local vocabulary.

## Verification Evidence

Task prep evidence:

- `curl -s -X POST https://diashort.apps.quickable.co/render ...` -> `https://diashort.apps.quickable.co/d/4a99eb1d`
- `test -d .c3 && echo C3=yes || echo C3=no` -> `C3=no`

RED evidence:

- `go test ./core` from `substrate/go` -> failed with missing `BuildPlan`, `HAScale`, `BrowserLease`, `ScriptLease`, `Accepted`, and `FramedStdio`.

Implementation evidence:

- `go test ./core` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/core`.
- `go test ./...` from `substrate/go` -> `ok` for `contract`, `core`, and `edge`.
- `bun run schema:parity` -> endgame contract tests `21 pass`, `0 fail`, `152 expect() calls`; Go `contract`, `core`, and `edge` packages `ok`.
- `bun run typecheck` -> SDK plus orchestrator typecheck passed.
- `bun run test` -> `52 pass`, `0 fail`, `334 expect() calls`.
- `bun run build` -> SDK ESM, CommonJS, and declarations emitted.
- `bun run pack:dry` -> `tinkabot-0.1.0.tgz`, 6 files.
- `git diff --check` -> passed.

Named negative-case evidence (re-executed 2026-06-10 during the release-spine evidence audit):

- `go test ./core -run TestBuildPlanDeniesBeforeAuthority -v -count=1` from `substrate/go` -> `--- PASS: TestBuildPlanDeniesBeforeAuthority`: neighbor authority is denied before any substrate plan is built.
- `go test ./core -run TestCredentialLeaseBookRevokesIdempotently -v -count=1` from `substrate/go` -> `--- PASS: TestCredentialLeaseBookRevokesIdempotently`: revoked leases deny credential reuse idempotently.
- `go test ./core -run TestErrorAttribution -v -count=1` from `substrate/go` -> `--- PASS: TestErrorAttribution`: auth, ledger, process, gateway, and cleanup failures are attributed as typed events.

## Wrap-Up Announcement

The `go-substrate-core` milestone is complete. Go now owns a verified, typed, fakeable substrate core contract that `embedded-nats-adapter` and later activation, script, and materializer work can consume for embedded NATS lifecycle, HA/scale topology, auth render, leases, store substrate, ledger, process boundary, gateway substrate, and attribution without relying on TypeScript runtime authority.
