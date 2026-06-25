---
layer: task
topic: embedded-nats-adapter
references:
  - ../approach/go-substrate.md
  - ../plan/go-substrate.md
  - ./go-substrate-core.md
---

# Embedded NATS Adapter Task

Diagram: https://diashort.apps.quickable.co/d/9a4270ef

## Objective

Attach the Go substrate core contracts to a real embedded NATS runtime without changing the caller-facing substrate contract. This task proves the live single-node embedded path with JetStream, NATS auth load shape, WebSocket posture, topology probes, drain/shutdown behavior, and adapter-owned error mapping.

## Scope

This task owns one executable unit: the embedded NATS adapter that consumes `go-substrate-core`.

In scope:

- start and stop a managed embedded `nats-server` process in Go.
- enable and verify JetStream readiness for the live adapter path.
- consume core topology, auth, store, lifecycle, and attribution contracts instead of redefining them.
- expose adapter posture for client URL, readiness, JetStream, WebSocket listener configuration, store directory, and topology probe result.
- map runtime failures from NATS, client connection, JetStream, auth loading, WebSocket, probes, drain, shutdown, and unknown panics into adapter-owned typed errors.
- propagate invalid core contracts unchanged when the failure belongs to `go-substrate-core`.

Out of scope:

- activation-source routing.
- script execution.
- materializer writes.
- browser shell or generated UI content.
- Docker sandboxing.
- full HA cluster proof.
- bespoke replication, routing, or auth vocabulary outside NATS-provided mechanisms.

## Acceptance Contract

- The adapter starts a live embedded NATS server from core-approved topology and store inputs.
- JetStream is enabled and observable before the adapter reports ready.
- Adapter readiness is not a log-only condition; it is proven through a live NATS client connection and JetStream check.
- Auth loading is represented through NATS auth vocabulary and fails before readiness when required auth material is absent or invalid.
- WebSocket posture reports whether the listener is enabled and which configured host, port, and TLS posture the browser edge can later consume.
- Topology probes run after the server is reachable and before the runtime is returned to callers.
- Shutdown drains owned client connections, requests server shutdown, waits for shutdown completion, and reports timeout or shutdown failure as adapter errors.
- Core contract invalidity is propagated unchanged; runtime attachment failures are transformed into adapter errors.
- The adapter keeps the HA/scale contract surface from core intact even though this task proves only the live single-node path.

## RED Artifact

Expected failing tests before implementation:

- `T-GO-ADAPTER-LIFECYCLE`: a live single-node embedded NATS runtime starts, reports ready posture, exposes a client URL, enables JetStream, uses a configured store directory, accepts a live NATS client connection, and stops cleanly.
- `T-GO-ADAPTER-CONFIG`: invalid core topology or store input returns the original core error unchanged rather than wrapping it as an adapter error.
- `T-GO-ADAPTER-AUTH-LOAD`: missing, expired, revoked, or malformed auth load material fails before readiness and maps to the adapter auth-load failure family.
- `T-GO-ADAPTER-WEBSOCKET`: enabled WebSocket configuration is reflected in runtime posture, and listener setup failure maps to the adapter WebSocket failure family.
- `T-GO-ADAPTER-TOPOLOGY`: a failed topology probe prevents runtime return and maps to the adapter topology-probe failure family with operation context.
- `T-GO-ADAPTER-DRAIN`: owned client drain, server shutdown, and shutdown wait failures map to drain-timeout or shutdown-failure families without leaking raw NATS errors.
- `T-GO-ADAPTER-CRITICAL`: unexpected adapter panics or nil runtime internals are recovered into adapter-critical errors with origin and operation context.

## Execution Notes

Implement this as a Go adapter package under `substrate/go` that imports `core` and `github.com/nats-io/nats-server/v2/server`. Keep names short at local scope, but keep public config, posture, and error fields explicit because they are safety boundaries.

Use `server.NewServer`, `Server.Start`, `ReadyForConnections`, `ClientURL`, `Shutdown`, and `WaitForShutdown` for the embedded server lifecycle. Use a live `nats.Conn` for readiness and a JetStream call for the JetStream proof. The adapter owns connection draining for connections it creates.

Keep the tests inside-out and traced: every RED id above owns exactly one adapter behavior family. Runtime failures transform at the adapter boundary; core validation failures propagate.

Do not introduce activation subjects, script IPC, Docker behavior, materializer projections, or browser command flow in this task. Those belong to later task layers.

## Verification Evidence

Task prep evidence:

- `test -d .c3 && printf 'C3=yes\n' || printf 'C3=no\n'` -> `C3=no`
- `curl -s -X POST https://diashort.apps.quickable.co/render ...` -> `https://diashort.apps.quickable.co/d/9a4270ef`

RED evidence:

- `go test ./embednats` from `substrate/go` first exposed missing `go.sum` data for the existing JSON schema dependency; `go mod tidy` repaired module sums after NATS dependencies were added.
- `go test ./embednats` from `substrate/go` then failed with missing adapter symbols: `Start`, `Config`, `Kind`, and `Runtime`.

GREEN evidence:

- `go test ./embednats` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/embednats`.
- `go test ./...` from `substrate/go` -> `ok` for `contract`, `core`, `edge`, and `embednats`.
- `bun run schema:parity` -> endgame contract tests `21 pass`, `0 fail`; Go packages `contract`, `core`, `edge`, and `embednats` passed.
- `bun run test` -> `52 pass`, `0 fail`, `334 expect() calls`.
- `bun run test:e2e` -> distribution BDD `1 pass`, `0 fail`, `16 expect() calls`.
- `bun run typecheck` -> SDK and orchestrator typecheck passed.
- `bun run build` -> SDK CJS, ESM, and declaration bundles emitted.
- `bun run pack:dry` -> `tinkabot-0.1.0.tgz`, 6 files.
- `bun run orchestrate:codex -- --dry-run --allow-dirty` -> next topic `activation-source-router`.
- `git diff --check` -> clean.
- Focused placeholder scan over this task doc, `substrate/go/embednats`, and `tasks/todo.md` -> no matches.

Review evidence:

- Subagent layer/security review found an unrestricted internal probe user and missing router-safe client boundary. Fixed by generating a random probe credential, limiting probe permissions to JetStream readiness subjects plus reply inboxes, adding a probe denial test, and adding `Runtime.Connect` as the adapter-owned client boundary.
- Subagent traced-test review found under-owned WebSocket runtime setup mapping, auth-load branches, and stop panic wrapping. Fixed with runtime WebSocket start mapping, auth missing/expired/mismatch/malformed tests, and `Stop` panic recovery for drain, shutdown, and wait paths.

## Wrap-Up Announcement

Shipped: `embedded-nats-adapter` attaches Go substrate core contracts to a verified live embedded NATS runtime with JetStream readiness, NATS-auth-shaped load failure handling, WebSocket posture, topology probe behavior, drain/shutdown semantics, and adapter-owned error mapping. Activation routing, scripts, materialization, Docker, and full HA cluster proof remain later layers.
