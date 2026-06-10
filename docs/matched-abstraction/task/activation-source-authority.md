---
layer: task
topic: activation-source-authority
references:
  - ../approach/endgame-app.md
  - ../approach/go-substrate.md
  - ../plan/activation-foundation.md
  - ../plan/script-nats-cli-proof.md
  - ./activation-contract-authority.md
  - ./activation-ledger-durability.md
---

# Activation Source Authority Task

Diagram: https://diashort.apps.quickable.co/d/63fd4830

## Objective

Add the source-authority boundary that decides whether a canonical activation source principal may observe its declared subject, KV key, Object Store object, stream subject, or schedule source. The boundary consumes compiled NATS auth vocabulary plus source principal and lease metadata. It returns a typed grant or a typed denial before any live router, watcher, schedule runner, or script execution exists.

## Scope

In scope:

- source principal and source kind binding.
- source lease active/revoked/expired/stale checks.
- subject wildcard aperture checks with deny-over-allow precedence.
- request/reply bounded response authority.
- imports, exports, and exposure preservation in the authority grant.
- typed authority event attribution.
- embedded NATS plus real `nats` CLI proof that the same permission shape allows the intended subject and denies a neighbor.

Out of scope:

- live source router subscriptions, KV/Object watches, stream consumers, or schedule loops.
- durable ledger acceptance, duplicate policy, replay, or loop suppression.
- script execution, materialization, browser gateway, or artifact serving.
- new auth vocabulary outside NATS `permissions`, `allow`, `deny`, `allow_responses`, imports, exports, and exposure.

## Acceptance Contract

- A schema-valid source fixture is denied when its principal or source lease does not match authority.
- An active source lease matching app revision, schema version, and script revision may receive a grant.
- Revoked and expired source leases fail with `LeaseRevoked` and `LeaseExpired`.
- Stale source lease revision fails with `StaleChain`.
- Deny rules win over allow rules, including denied-neighbor subjects.
- Wildcard source patterns must be concrete NATS wildcard apertures under the left-side authority prefix.
- Request/reply grants require bounded `allow_responses`.
- Grants preserve the matched exposure, imports, exports, observed subject, principal, lease, and attribution event.

## RED Artifact

Expected failing tests before implementation:

- `T-ACT-SRC-AUTH-ALLOW`: allowed source observation returns a grant with preserved source identity and exposure.
- `T-ACT-SRC-AUTH-DENY`: denied-neighbor and deny-over-allow subjects return `SourceAuthDenied`.
- `T-ACT-SRC-AUTH-WILDCARD`: wildcard overreach and source pattern mismatch are denied before routing.
- `T-ACT-SRC-AUTH-RESPONSE`: request/reply source requires bounded response authority.
- `T-ACT-SRC-AUTH-LEASE`: revoked, expired, stale, and mismatched source leases are denied.
- `T-ACT-SRC-AUTH-EVENT`: grant and denial carry source-authority event attribution.
- `T-ACT-SRC-AUTH-CLI`: embedded NATS plus real `nats` CLI allows the granted subject and rejects a denied neighbor using the same auth shape.

## Execution Notes

Keep this in `substrate/go/core` as a pure authority boundary. The embedded NATS test may prove the compiled permissions with real server enforcement, but core must not import NATS. Schedule authority is represented as a namespaced authority subject for now; the schedule engine will later own durable time, leadership, and catch-up behavior.

## Verification Evidence

Task prep and RED evidence:

- `go test ./core -run TestSourceAuthority -count=1` from `substrate/go` failed before implementation with missing `AuthorizeSource`, `SourceAuthDenied`, and related source authority symbols.
- `go test ./embednats -run TestSourceAuthorityCLIAllowedAndDeniedSubject -count=1` from `substrate/go` initially proved real CLI/server denial behavior, then was adjusted for `nats` CLI v0.3.0 surfacing permission errors in output while returning success.

GREEN evidence:

- Added `AuthorizeSource` and `SourceGrant` in `substrate/go/core`.
- Added `SourceAuthDenied` and `DeniedNeighbor` typed errors.
- Added Go NATS subject matching for `*` and `>` with deny-over-allow precedence.
- Added source lease revoked, expired, stale app/schema/script revision, principal binding, source-kind binding, authority ref, exposure, export, import, and bounded response checks.
- Added source coordinate normalization for request/reply, command acceptance, subject, KV, Object Store, stream, and schedule sources.
- Added source-authority attribution event on grants and `SourceAuthority.AuthorizeSource` attribution on denials.
- Added `substrate/go/core/source_authority_test.go` covering allowed canonical sources, denied neighbor, deny-over-allow, wildcard overreach, bounded request/reply responses, revoked/expired/stale source leases, missing exposure/export, and advanced import/exposure denial.
- Added `substrate/go/embednats/source_authority_cli_test.go` using embedded NATS plus real `nats request` CLI against source credentials to prove allowed request/reply and denied-neighbor permission evidence.

Final verification:

- `go test ./core -run TestSourceAuthority -count=1` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/core`.
- `go test ./embednats -run TestSourceAuthorityCLIAllowedAndDeniedSubject -count=1` from `substrate/go` -> `ok github.com/lagz0ne/tinkabot/substrate/go/embednats`.
- `go test ./core ./embednats -count=1` from `substrate/go` -> `ok` for `core` and `embednats`.
- `go test ./... -count=1` from `substrate/go` -> `ok` for `contract`, `core`, `edge`, and `embednats`.
- `bun run schema:parity` -> endgame contract tests `21 pass`, `0 fail`; Go packages passed.
- `bun run test` -> `52 pass`, `0 fail`, `374 expect() calls`.
- `bun run test:e2e` -> `1 pass`, `0 fail`, `16 expect() calls`.
- `bun run typecheck` -> passed.
- `bun run build` -> SDK ESM, CommonJS, and declarations emitted.
- `bun run pack:dry` -> produced `tinkabot-0.1.0.tgz` dry-run package.
- `bun run validate:layers` -> layer validation passed.
- `bun run test:layers` -> `10` layer validation tests passed.
- `git diff --check` -> clean.

## Wrap-Up Announcement

Shipped: source authority owns source-scoped NATS permission decisions, source lease lifecycle/revision checks, import/export/exposure preservation, bounded request/reply response authority, denied-neighbor behavior, and real embedded-NATS CLI proof. Live source routing, durable schedules, script execution, materialization, live credential reload/revocation, and release proof remain later tasks.
