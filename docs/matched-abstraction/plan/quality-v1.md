---
layer: plan
topic: quality-v1
references:
  - ../approach/endgame-app.md
  - ../approach/go-substrate.md
  - ./endgame-app.md
---

# Quality V1 Plan

## Consumed Approach

This Plan consumes `docs/matched-abstraction/approach/endgame-app.md` as top-level authority and `docs/matched-abstraction/approach/go-substrate.md` as sealed substrate authority. The Endgame App Plan (`./endgame-app.md`) remains the completed v1 decomposition record; this Plan starts where it ended, after `release-spine` closed all sixteen v1 milestones behind `bun run release:evidence`.

The Go Substrate Approach is sealed. This Plan refines decomposition and verification only. It does not reopen embedded NATS ownership, NATS auth vocabulary (`permissions.publish`, `permissions.subscribe`, `allow`, `deny`, `allow_responses`), separated authority envelopes, mediated scripts, generated-content denial, or typed substrate failures.

This Plan carries the following pinned decisions as consumed authority. They are inputs, not open questions:

- The program goal is a usable, high-quality v1 — functional, performant, secure — where a user can start using the tool. The v1 entry surface is a single Go binary: embedded NATS plus embedded frontend shell plus the script materializer loop, operated through the `nats` CLI.
- Four standing gates apply to every slice: all tests run over real embedded NATS with an explicit fakes allowlist; tests execute in parallel against isolated servers and stores; coverage is dual — inside-out per-layer measurement plus outside-in scenario-matrix completeness; and `be-lazy` style is enforced by a diff-scoped reviewer gate. The reviewer gate is a process gate carried in every handoff, not a slice of its own.
- The auth backbone is NATS operator/JWT mode, verified against pinned nats-server v2.14.2 (`TrustedOperators`, `AccountResolver`/`MemAccResolver`, live `UpdateAccountClaims`). The master operator key is substrate-held and generated at first start; accounts split along control-plane/app-plane authority domains; principals become short-lived user JWTs carrying the existing lease fields; rolling permission and account updates apply to live connections; revocation disconnects. Proving this closes the deferred live-auth-reload item. The substrate-callback alternative was considered and rejected.
- Exposure is a typed posture, not a port number: in-process default (`DontListen` plus `InProcessConn` plus `nats.InProcessServer`, verified in nats.go v1.52.0), loopback opt-in (what `nats` CLI usage and outside-in proofs construct explicitly), and external opt-in per surface (NATS port, WebSocket, HTTP gateway) with matching auth tier and TLS beyond loopback. Tests keep loopback through the same declared API.
- `docs/manual/v1.md` is the usage contract. The binary must satisfy the manual unchanged, plus a new section for starting the binary, and a "manual commands run verbatim" gate belongs in this program.
- This program takes on two previously deferred items: live auth reload (through the operator/JWT backbone) and the product entry surface (through the binary). Direct browser NATS WebSocket, Docker sandboxing, product UI rendering beyond the shell, broad script CRUD UI, multi-node HA, and package publication stay deferred. If the binary slice naturally makes pack and publish shaped, the shape may land, but publication itself stays out of scope and the manifest must keep naming it deferred.

## Decomposition

The program splits into five slices:

| Slice | Owns |
| --- | --- |
| `quality-gate-infrastructure` | One shared embedded-NATS test harness with per-test isolated servers and stores, parallel execution discipline, the explicit fakes allowlist and its checker, inside-out per-layer coverage measurement, and the outside-in scenario-matrix completeness check |
| `typed-exposure-posture` | The typed exposure API: in-process default, loopback opt-in, external opt-in per surface with matching auth tier and TLS requirement; migration of all proofs to declared loopback through that API; denial of undeclared or mismatched exposure |
| `operator-jwt-authority` | Operator/JWT mode: substrate-held master operator key generated at first start, control-plane and app-plane account split, principal-to-user-JWT minting carrying existing lease fields, live rolling account/permission updates, revocation that disconnects, and the live auth reload proof |
| `tinkabot-binary` | The single Go binary: startup and shutdown lifecycle, first-start key and store materialization, embedded frontend shell serving, the script materializer loop wired through declared exposure and operator/JWT auth, and the manual's new "starting the binary" section |
| `quality-release` | Extending `bun run release:evidence` with the four gate results and the manual-verbatim check, so one centralized operation proves the quality program the same way it proves the v1 milestones |

Each slice is small but complete at its boundary, denied and failure paths included. The four standing gates apply inside every slice from `quality-gate-infrastructure` onward; the first slice exists precisely so the later slices never run without them.

## Dependency Ordering

`quality-gate-infrastructure` is first. Both the parallel-test refactor and the operator/JWT migration touch every embednats test's server setup, which creates a double-touch hazard: migrating auth first would edit each test's construction individually, and the parallel refactor would then edit the same call sites again. Ordering the harness first inverts that cost. The harness slice converts per-test server construction into one declared factory seam once; after that, the exposure migration and the operator/JWT migration are each a change to one seam plus per-test declarations, not a second sweep over every test file. Total churn is one structural pass over the tests plus two factory-level changes, instead of three structural passes.

`typed-exposure-posture` is second. It completes the harness construction surface: every proof declares its posture (in-process or loopback) through the same API the binary will consume, and the external tier exists as a typed, denied-by-default option. Landing exposure before auth keeps the heavier migration honest — operator/JWT must work identically across in-process and loopback postures, and the posture API is the seam that proves it.

`operator-jwt-authority` is third. It is the largest authority migration in the program and it rides the now-stable harness: the static-auth-to-JWT switch lands in the factory and the principal vocabulary, while tests keep their declared postures and case matrices. It closes live auth reload, which the binary depends on for revocation and rolling updates against live connections.

`tinkabot-binary` is fourth. It is deliberately an assembly slice: it consumes the exposure API, the operator/JWT authority, the existing embedded frontend package, and the existing materializer loop. Nothing in the binary may invent auth, exposure, or loop behavior the earlier slices did not prove; if it needs to, the work returns to the owning slice.

`quality-release` is last. It packages the gate results and the manual-verbatim proof into the existing centralized evidence operation. Like `release-spine`, it adds no runtime features; an unsupported claim it finds routes back to the owning slice.

## Parallelization Rules

Do not parallelize `typed-exposure-posture` and `operator-jwt-authority`. Both modify harness construction and principal wiring; running them concurrently recreates the double-touch hazard the ordering exists to avoid.

Within `quality-gate-infrastructure`, the fakes-allowlist checker, the per-layer coverage measurement, and the scenario-matrix check may proceed in parallel once the harness factory seam is fixed, because they read test structure rather than mutate it.

Frontend-embed and binary-lifecycle work inside `tinkabot-binary` may run in parallel with each other but not ahead of `operator-jwt-authority`, because binary startup materializes the operator key and account split at first start.

`quality-release` checker work may begin once the gate operation names exist, but it cannot present a gate as passing before the owning slice has landed its proof — the same rule the endgame Plan applied to centralized ops.

The `be-lazy` diff-scoped reviewer gate runs per slice as a standing process step and is never a parallelizable work item.

## Handoff Contracts

Every Task in this program receives the consumed Approach documents, this Plan, its slice row, the four standing gates, and the pinned decisions above. Every Task output includes a RED artifact, implementation evidence, inside-out proof, outside-in real-NATS proof where the slice crosses an actor boundary, gate results, a `be-lazy` reviewer pass, and a wrap-up announcement. Every Task rejects un-allowlisted fakes, shared mutable test servers, undeclared exposure, invented auth vocabulary, and happy-path-only proof.

Per-slice contracts:

`quality-gate-infrastructure` receives the current embednats test corpus and the existing fake usages (memory ledger and schedule stores and any narrow branch-forcing fakes) as its RED input. Its proof surface is: the full Go suite passes in parallel with isolated servers and stores; an injected isolation violation is detected; an un-allowlisted fake fails the fakes gate while each allowlisted fake carries a written justification and the real-NATS proof that validates it; per-layer coverage reports a number per substrate layer; and the scenario matrix reports completeness against the pinned case families (allowed, denied neighbor, malformed, duplicate, stale, revoked, attributed failure). It owns the failure families: fakes violation, isolation violation, coverage gap, scenario-matrix hole, and measurement stale.

`typed-exposure-posture` receives the harness factory seam and the verified in-process mechanism. Its proof surface is: in-process default opens no socket and serves all in-process proofs; loopback opt-in is explicitly declared and carries the `nats` CLI proofs unchanged in behavior; external opt-in without a matching auth tier or without TLS beyond loopback is denied as a typed failure, not a warning; and a posture declared by a test is the posture the server actually has. It owns the failure families: undeclared exposure, posture mismatch, in-process connection failure, and external tier policy violation.

`operator-jwt-authority` receives the posture API, the existing managed-auth policy vocabulary, and the existing lease fields as the provenance it must preserve. Its proof surface covers the full pinned case matrix over real embedded NATS in operator mode: allowed publish/subscribe/request per account, denied neighbor across the control-plane/app-plane account split, malformed and expired JWT denied at the connection, duplicate principal handling, stale account claims superseded by a live `UpdateAccountClaims` push, revoked principal disconnected from a live connection and denied on reconnect, and every grant and denial attributed with the lease fields carried in the JWT. First-start operator key generation and reload from the store directory are part of the surface. It owns the failure families: operator key material failure, account compile failure, JWT mint failure, live update failure, revocation enforcement failure, and provenance loss.

`tinkabot-binary` receives the exposure API, operator/JWT authority, the embedded frontend package, and the materializer loop as consumed inputs. Its proof surface is: the binary starts from an empty store directory and materializes key and store state; it starts again from existing state without regeneration; the embedded shell is served under the proven scope and policy headers; the manual's script, trigger, observation, and denial flows run through the `nats` CLI against the running binary, including denied caller and observer writes, duplicate no-rerun, and revoked-lease denial; and shutdown drains cleanly. It owns the failure families: startup materialization failure, frontend serve failure, wiring mismatch between declared posture and served surface, manual divergence, and shutdown failure.

`quality-release` receives the four gate operations, the manual, and `release/v1.json` conventions. Its proof surface is: the extended evidence check fails on a synthetic missing or overclaimed gate result, fails on a manual command whose live output diverges, and passes on the real corpus; the deferred list still names everything that stays deferred. It owns the failure families: gate result missing, gate overclaim, manual divergence, and evidence stale.

## Verification Strategy

Existing commands remain authoritative: `bun run test`, `bun run test:e2e`, `bun run typecheck`, `bun run build`, `bun run pack:dry`, `bun run schema:parity`, `go test ./... -count=1` from `substrate/go`, and `bun run release:evidence`.

This program introduces stable operation names now; their implementations belong to the owning slices:

| Operation | Owner slice | Proves |
| --- | --- | --- |
| `bun run gate:parallel` | `quality-gate-infrastructure` | Full Go suite over real embedded NATS with `-shuffle=on`, parallel execution, and isolated servers/stores |
| `bun run gate:fakes` | `quality-gate-infrastructure` | Every fake in the test corpus appears in the explicit allowlist with justification; anything else fails |
| `bun run gate:coverage` | `quality-gate-infrastructure` | Inside-out per-layer coverage measurement against declared thresholds |
| `bun run gate:scenarios` | `quality-gate-infrastructure` | Outside-in scenario-matrix completeness over the pinned case families |
| `bun run gate:manual` | `quality-release` | Manual commands run verbatim against the binary and produce the documented outcomes |
| `bun run release:evidence` (extended) | `quality-release` | The v1 manifest plus the four gate results plus the manual-verbatim result as one centralized release gate |

The `be-lazy` reviewer gate is a process gate executed per slice over the slice diff; it produces review evidence in the Task wrap-up rather than a script name.

The test ownership graph from the endgame Plan stays in force: inside-out tests live where their declared failure family lives; outside-in proofs compose already-proven contracts over real NATS-mediated surfaces; fakes localize forced branches only after the real path exists and only when allowlisted.

## First Executable Task Slice

Topic: `quality-gate-infrastructure`.

Purpose: establish the harness and the four standing gates so every later slice in this program runs under them, and so the exposure and auth migrations become single-seam changes instead of test-corpus sweeps.

RED artifact: gate operations failing on the current corpus — `gate:fakes` reporting fakes that exist without an allowlist, `gate:parallel` reporting either serialized execution or isolation collisions under shuffled parallel runs, `gate:coverage` reporting absent per-layer measurement, and `gate:scenarios` reporting matrix holes or absent matrix definition. The findings must be concrete and attributable to current files, the same way the release-spine RED enumerated its 27 findings.

GREEN boundary: one harness factory that every embednats test uses to obtain an isolated embedded server and stores; the full Go suite green under shuffled parallel execution; an explicit fakes allowlist with per-entry justification and the real-NATS proof that validates each fake; per-layer coverage measurement emitting numbers per substrate layer; and a scenario matrix declaring the pinned case families per outside-in surface with completeness checked. Injected violations — an un-allowlisted fake, a deliberately shared server — must fail their gates to prove the gates detect, not just report.

Verification boundary: `bun run gate:parallel`, `bun run gate:fakes`, `bun run gate:coverage`, `bun run gate:scenarios`, `go test ./... -count=1` from `substrate/go`, `bun run test`, `bun run typecheck`, and the `be-lazy` reviewer pass over the slice diff.

Non-goals: no exposure API change, no auth backend change, no binary work, no manual edits, no release-evidence extension. Test behavior must remain unchanged through the refactor; the slice proves the same assertions under new discipline.

## Escalation Log

Escalate to Approach, explicitly unsealing where required, if any slice needs to redefine embedded NATS ownership, NATS auth vocabulary, separated authority envelopes, mediated scripts, generated-content denial, or typed substrate failures; if operator/JWT mode cannot carry the existing lease, session, revision, and capability provenance into user JWTs; if the exposure posture would give generated browser content a NATS surface; or if release confidence would rest on happy paths or unmeasured coverage.

Escalate within this Plan if the harness factory cannot serve both static-auth and operator-mode construction during the migration window; if operator/JWT auth cannot keep the manual's behavior commands verbatim beyond the connection preamble; if the binary needs a capability the materializer loop, exposure API, or auth backbone did not prove; or if gate thresholds turn out to gate noise instead of quality.

Return to the owning slice, without escalation, when a later slice finds a missing proof: exposure gaps return to `typed-exposure-posture`, auth gaps to `operator-jwt-authority`, gate-detection gaps to `quality-gate-infrastructure`, and manual divergence found by `quality-release` returns to `tinkabot-binary`.

One reconciliation is pre-decided at Plan level: the manual's "Connecting with the nats CLI" section authenticates with `--user` and `--password` where the password is a lease id. Operator/JWT mode authenticates connections with JWT credentials. The manual-verbatim contract therefore binds to the manual's behavior commands — script records, triggers, statuses, observations, denials, and their outcomes — which must run unchanged. The connection preamble is the one manual surface the `operator-jwt-authority` slice owns revising, in the same slice that proves live JWT auth, with every behavior command re-verified verbatim under the new preamble. If the migration forces changes beyond the connection preamble, that exceeds this Plan's authority and escalates.
