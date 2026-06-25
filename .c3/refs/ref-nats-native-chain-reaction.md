---
id: ref-nats-native-chain-reaction
c3-seal: 9f531996a42830e16698875453d9e5e2ad8ea432b60799d3df284096be5a5f5f
title: NATS Native Chain Reaction
type: ref
parent: c3-0
goal: Standardize how Tinkabot turns source changes, triggers, item state, projections, and UI updates into an observable chain reaction.
---

## Goal

Standardize how Tinkabot turns source changes, triggers, item state, projections, and UI updates into an observable chain reaction.

## Choice

Use NATS-mediated material as the reaction spine: activation sources normalize events, scripts emit framed effects, materializer writes KV/Object-backed projections and artifacts, and consumers observe derived material rather than private process state.

## Why

The product is for LLM/user-built apps where backend and UI may be generated over time. NATS subjects, KV/Object Store records, streams, schedules, and explicit ledgers provide durable, inspectable, replayable evidence that local callbacks or in-memory UI state cannot provide.

## How

Required pattern from `substrate/go/tinkabot/bundle.go`: bundle entries derive script keys, trigger subjects, projection ids, and artifact prefixes; `watches` filters observe a projection and emit new material through the same gate.

Required proof surfaces: `substrate/go/tinkabot/bundle_test.go` `TransformPipe`, `substrate/go/embednats/filter_loop_test.go`, `examples/clock/scripts/present.sh`, and `docs/matched-abstraction/task/bundle-transform.md`.
