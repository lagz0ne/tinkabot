---
target: c3-302
scope: whole
type: component
parent: c3-3
title: Tinkalet Reactions and Coordination
category: feature
---
## Goal

Bridge durable items to local action.

## Parent Fit

| Field | Value |
| --- | --- |
| Parent | c3-3 Tinkalet Edge |
| Role | Item records, watches, cursors, server-owned schedules, and local reaction execution |
| Owns | Tinkalet item/watch/reaction/schedule behavior and corresponding Go tests |
| Does not own | Bundle manifest filter graph or runtime-owned NATS accounts |

## Purpose

This component implements the edge side of chain reaction outside bundles: Tinkabot records durable item truth, Tinkalet watches or waits through a profile, and an explicit local reaction can run a local command and write back a product item only when profile authority permits it.

## Governance

| Reference | Type | Governs | Precedence | Notes |
| --- | --- | --- | --- | --- |
| ref-nats-native-chain-reaction | ref | Item, watch, schedule, and reaction flows | Durable item truth outranks local edge assumptions | Watches and schedules are NATS-backed product flows. |
| ref-shadow-authority-boundaries | ref | Local reaction execution and writeback | Server grants and profile scope outrank local action | Reaction failure must not advance the cursor. |

## Contract

| Surface | Direction | Contract | Boundary | Evidence |
| --- | --- | --- | --- | --- |
| Item and watch commands | IN/OUT | Item revisions, waits, live watch, retained cursor catch-up, and stale cursor denial. | Tinkalet to Tinkabot item KV | `substrate/go/tinkabot/tinkalet_item_test.go`, `tinkalet_watch_test.go` |
| Local reactions and schedules | IN/OUT | Reactions run explicit local argv; schedules are Tinkabot-owned timing records writing items. | Local host to durable item truth | `substrate/go/tinkabot/tinkalet_reaction_test.go`, `tinkalet_schedule_test.go` |

## Derived Materials

| Material | Must derive from | Allowed variance | Evidence |
| --- | --- | --- | --- |
| Tinkalet edge task evidence | Goal, Parent Fit, Contract, and Derived Materials | Task transcript wording | `docs/matched-abstraction/task/tinkalet-*.md` |
