---
target: c3-203
scope: whole
type: component
parent: c3-2
title: Bundle Chain Runtime
category: feature
---
## Goal

Run one folder as one app.

## Parent Fit

| Field | Value |
| --- | --- |
| Parent | c3-2 Go Product Runtime |
| Role | Bundle manifest loading, account isolation, schedules, transforms, sandboxing, artifact/projection serving |
| Owns | `substrate/go/tinkabot/bundle*`, bundle examples, bundle task evidence |
| Does not own | Durable non-bundle app-plane script installation |

## Purpose

Bundle Chain Runtime implements the user's app shape: a bundle delivers backend scripts plus UI artifacts, entries can boot, tick, or watch projections, filters transform NATS material into frontend-shaped views, and scripts inherit derived authority through the runtime rather than declaring permissions explicitly.

## Governance

| Reference | Type | Governs | Precedence | Notes |
| --- | --- | --- | --- | --- |
| ref-bundle-as-app | ref | Bundle manifest and one-folder app behavior | Bundle Approach outranks individual examples | Examples are proof apps, not separate product truths. |
| ref-nats-native-chain-reaction | ref | `watches` filters and projection/artifact materialization | NATS material outranks direct UI state | Frontend consumes derived projections. |
| ref-shadow-authority-boundaries | ref | Derived script keys, triggers, projections, artifacts, and account isolation | Derived authority over manifest-declared raw permissions | Manifests cannot spell raw NATS authority. |
| rule-bundle-sandbox-default-fail-closed | rule | Bundle process execution | Sandbox default over host convenience | `--bundle-sandbox none` is explicit opt-in only. |

## Contract

| Surface | Direction | Contract | Boundary | Evidence |
| --- | --- | --- | --- | --- |
| Bundle manifest | IN | Strict `bundle.manifest` with script names, command, projections, boot/every/watches; authority derived. | Bundle dir to daemon | `substrate/go/tinkabot/bundle.go`, `examples/*/bundle.json` |
| Chain reaction | IN/OUT | Projection changes feed long-lived filters through JSONL and filters emit mediated effects. | NATS KV material to script process | `docs/matched-abstraction/task/bundle-transform.md`, `substrate/go/tinkabot/bundle_test.go` |

## Derived Materials

| Material | Must derive from | Allowed variance | Evidence |
| --- | --- | --- | --- |
| Clock and builder examples | Goal, Parent Fit, Contract, and Derived Materials | Example-specific scripts and README prose | `examples/clock/**`, `examples/builder/**` |
