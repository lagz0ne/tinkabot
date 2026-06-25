---
id: c3-2
c3-seal: b801abde2263c5c7cd3bd245e77779765b8a4c06cae64f3e27d43d0637c7b1d1
title: Go Product Runtime
type: container
parent: c3-0
goal: Run Tinkabot as the server authority that turns NATS-native activations, scripts, materials, bundles, and shell serving into one product posture.
---

## Goal

Run Tinkabot as the server authority that turns NATS-native activations, scripts, materials, bundles, and shell serving into one product posture.

## Components

| ID | Name | Category | Status | Goal Contribution |
| --- | --- | --- | --- | --- |
| c3-201 | Daemon Assembly |  | active | Start the product binary posture. |
| c3-202 | Embedded NATS Authority |  | active | Own NATS lifecycle and authority. |
| c3-203 | Bundle Chain Runtime |  | active | Run one folder as one app. |
| c3-204 | Session and Wrapper Runtime |  | active | Mediate session streams and wrapper IO. |

## Responsibilities

This container owns the `tinkabot` binary, embedded NATS lifecycle, operator/JWT auth, activation ledger, source router, script runtime, materializer, bundle loading, sandboxing, artifact/projection routes, session runtime, and Go-side manual proof.

## Complexity Assessment

This is the highest-risk container because it crosses auth, process execution, durable NATS state, browser serving, and generated code. Changes here need real embedded NATS verification, denial oracles, and release evidence.
