---
target: c3-2
scope: whole
type: container
parent: c3-0
title: Go Product Runtime
boundary: single Go binary, embedded NATS, auth, materialization, bundles, sessions, and server shell
---
## Goal

Run Tinkabot as the server authority that turns NATS-native activations, scripts, materials, bundles, and shell serving into one product posture.

## Components

| ID | Name | Category | Status | Goal Contribution |
| --- | --- | --- | --- | --- |

## Responsibilities

This container owns the `tinkabot` binary, embedded NATS lifecycle, operator/JWT auth, activation ledger, source router, script runtime, materializer, bundle loading, sandboxing, artifact/projection routes, session runtime, and Go-side manual proof.

## Complexity Assessment

This is the highest-risk container because it crosses auth, process execution, durable NATS state, browser serving, and generated code. Changes here need real embedded NATS verification, denial oracles, and release evidence.
