# Lessons

Project-specific corrections:

- For NATS script metadata, keep names succinct and NATS-focused. Prefer nested settings objects when a field will need sub-configuration later.
- NATS script metadata should use `desc`, not `meaning`, for LLM-readable descriptions. Model both access and exposure: inside-out publishing and outside-in invocation/consumption.
- Do not use placeholder subject strings such as angle-bracket IDs in NATS metadata examples. Use concrete subject values/patterns and preserve left-to-right authority in subject token design.
- Prefer NATS auth vocabulary for authority: `permissions.publish`, `permissions.subscribe`, `allow`, `deny`, and `allow_responses`. Keep LLM-facing fields descriptive, not authoritative.
- For script runtime design, do not expose the whole NATS surface to scripts. Preserve NATS advantages through a mediated mechanism: security vocabulary, imports, and Tinkabot/runtime as the middle layer.
- Avoid the MVP trap on this project. A first slice may be small, but it must be precise and complete at its chosen boundary, with edge cases and denial paths designed up front.
- Default scripts should be NATS-agnostic process contracts. Use stdin for input, stdout for final result, stderr for diagnostics, and a runtime-owned IPC channel for progress/publish requests that Tinkabot validates and forwards to NATS.
- For long-run process IPC, prefer battle-tested framed stdio RPC over fd-specific channels. Extra fds can be adapters, but the canonical contract should be cross-platform and language-agnostic.
- Before implementing this runtime, spend upfront effort on layer-owned tests. Code starts only after typed errors, Resolve / Transform / Propagate ownership, protocol tests, and vertical proof fixtures are explicit.
- For local `@lagz0ne/nats-embedded` usage in this repo, include the local platform binary package explicitly or set `NATS_EMBEDDED_BINARY`; tests must clean their own JetStream `storeDir`.
- For package distribution work, verify the pack shape as well as the build. `bun pm pack --dry-run` caught the missing package version before final handoff.
- Request/reply execution alone cannot create script chains. Model reactive and automated starts as an activation layer above substrate, not as substrate itself.
- When triage converges across architecture, reliability, and test-contract angles, stop reviewing and convert the findings into matched-abstraction docs plus RED tests. More review is lower value than executable contract pressure.
- Before presenting concepts or strategy, use `triage-three` as decision support more often: stress-test the idea, collapse weak branches, and present the user with sharper tradeoffs plus a recommended path.
- Add and use `be-lazy` for coding posture: short clear names, compiler-backed inference, direct code, and no redundant ceremony, while keeping public contracts and safety boundaries explicit.
- For the Go substrate, embed and manage NATS as the default platform component. HA/scale posture must use NATS-provided clustering, JetStream replica/quorum, route/gateway/leaf, WebSocket, queue/consumer, and observability semantics rather than bespoke replication or treating NATS as an external-only dependency.
