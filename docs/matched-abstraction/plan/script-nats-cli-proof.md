---
layer: plan
topic: script-nats-cli-proof
references:
  - ../approach/endgame-app.md
  - ../approach/nats-script-runtime.md
  - ./endgame-app.md
  - ./activation-foundation.md
  - ./nats-script-runtime.md
---

# Script NATS CLI Proof Plan

Diagram: https://diashort.apps.quickable.co/d/ff5f7a64

## Consumed Approach

This Plan consumes the Endgame App Approach and NATS Script Runtime Approach. Script execution remains mediated: scripts do not receive raw NATS authority by default. Real NATS proof moves to the platform boundary: callers use the `nats` CLI against embedded NATS, and the platform reaction is observed through replies, subjects, KV/Object Store, streams, ledger records, statuses, and materialized projections.

The CLI is not a convenience wrapper. It is the outside-in test surface for script-side release confidence because it proves the same NATS subjects, credentials, request/reply behavior, auth denials, stream/KV/Object Store behavior, and event reactions a real caller would see.

## Scope

This Plan owns the proof posture for script-side work after activation foundation:

- trigger script-side behavior with real `nats request` or `nats publish`.
- observe reactions with real `nats subscribe`, `nats kv`, `nats object`, or stream inspection.
- run against the Go embedded NATS substrate, not an in-memory NATS substitute.
- use generated or runtime-issued NATS URL and credentials.
- keep scripts NATS-agnostic by default; script effects go through framed stdio and runtime facade.
- preserve typed inside-out tests for contracts, but treat NATS CLI outside-in proof as mandatory before a script-side slice is complete.

Out of scope:

- exposing raw NATS clients or CLI handles inside default scripts.
- replacing inside-out contract tests with broad end-to-end tests.
- using CLI proof to bypass source authority, activation ledger, script facade, materializer, or browser edge ownership.

## Decomposition

Script-side CLI proof decomposes into six proof units:

| Unit | Owns |
| --- | --- |
| CLI harness | spawning the real `nats` binary, applying NATS URL/credential env, capturing stdout/stderr, and enforcing timeouts |
| Trigger proof | `nats request` and `nats publish` commands that enter through authorized NATS subjects |
| Observation proof | `nats subscribe`, `nats kv`, `nats object`, or stream inspection that observes platform reactions |
| Authority proof | allowed, denied-neighbor, revoked/stale credential, and bounded response behavior visible from the caller boundary |
| Script proof | activation-to-execution handoff, framed stdio script process behavior, runtime facade effect mediation, and attribution |
| Material proof | projection, artifact, event, status, or ledger evidence that remains visible through NATS-facing stores or subjects |

## CLI Proof Boundary

Every script-side release-shaped Task must prove at least one observable loop with real CLI commands:

| Step | Required CLI proof |
| --- | --- |
| Trigger | `nats request` or `nats publish` sends a schema-valid command/source event into embedded NATS |
| Deny | a denied-neighbor subject or revoked/stale credential fails before script execution |
| Activation | ledger/status evidence shows accepted, duplicate, stale, or loop-suppressed result |
| Execution | script run emits an attributed status/event over NATS |
| Effect | materializer/projection/artifact store changes are observable through `nats kv`, `nats object`, stream, or subject subscription |
| Reply | request/reply sources return the expected response or typed denial |

The test harness may spawn `nats` with `exec.Command`, set `NATS_URL`, `NATS_USER`, `NATS_PASSWORD`, `NATS_CREDS`, or equivalent env, and assert stdout/stderr plus NATS-observed side effects. A helper wrapper is allowed only to reduce command boilerplate; the proof still uses the real CLI binary.

## Triage-Three Result

Pusher: CLI proof maximizes release value because it proves NATS auth, subject routing, request/reply, and observable reaction together.

Challenger: CLI proof can become flaky or too broad if it replaces layer-owned tests. Keep it outside-in and scenario-shaped; inside-out tests still own schema, source authority, ledger, script protocol, and materializer contracts.

Arbiter: enforce CLI proof for every script-side slice that crosses NATS. Use embedded NATS and real CLI commands for normal behavior. Use fakes only to force otherwise-impossible branch failures.

## Verification Strategy

Verification is layered:

- Inside-out tests still own schema, source authority, ledger, script process protocol, runtime facade, and materializer contracts.
- Outside-in script-side tests own caller-visible NATS behavior and must use real `nats` CLI commands against embedded NATS.
- Fakes and mocks are admissible only after the ordinary embedded NATS path is proven, and only to force narrow failure branches that cannot be reliably produced through NATS.
- CLI proof must check both command output and NATS-observable side effects; stdout alone is not product truth.
- CLI proof must include at least one denial or failure path for every release-shaped script-side slice.

Scenario matrix:

| Scenario | CLI action | Required observation |
| --- | --- | --- |
| Request execution | `nats request` | typed reply or denial plus activation/script status |
| Event execution | `nats publish` | activation accepted and script status/event emitted |
| Denied neighbor | `nats request` or `nats publish` to adjacent subject | no script execution and visible denial/auth failure |
| Duplicate | repeat command or source event with same dedupe key | duplicate status and no second script run |
| Materialized effect | accepted script effect | `nats kv`, `nats object`, stream, or subscription shows durable product reaction |

## Sequencing Impact

Steps 1 through 4 below are complete: source authority, the live source router, the script execution loop, and the materializer loop all carry real embedded NATS plus `nats` CLI evidence in their owning Task docs. Step 5, the release spine, is the remaining unit. The shape stays:

1. Source authority with real embedded NATS auth proof where possible.
2. Live source router with CLI-triggered request/subject/KV/Object/stream events.
3. Script runtime execution loop triggered through NATS, executed through framed stdio, and observed over NATS.
4. Materializer/artifact loop observed through CLI-visible KV/Object/stream changes.
5. Release spine that runs the whole loop with CLI commands and records evidence.

## Acceptance Gate

A script-side Task is not complete when only Go unit tests or in-memory fakes pass. It is complete when:

- its inside-out typed contracts pass.
- its outside-in CLI scenario passes against embedded NATS.
- denied, duplicate, stale, revoked, malformed, or loop-suppressed behavior is observable as a typed platform reaction.
- the same app/schema/script/materialized revisions appear in the CLI-visible response or stored evidence.

## Escalation

Escalate to Approach if a Task needs default raw NATS inside scripts, CLI-only success without typed contract tests, mocks for ordinary NATS behavior, or local process output as product truth without NATS-observable materialization.
