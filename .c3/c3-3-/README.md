---
id: c3-3
c3-seal: aa5bb987d8b0aad52f3bbb77f3358e3dccd79412ea95e6ffc98568bf2408e3f5
title: Tinkalet Edge
type: container
parent: c3-0
goal: Let humans, scripts, CI jobs, and local agents participate in Tinkabot without speaking raw NATS by default.
---

## Goal

Let humans, scripts, CI jobs, and local agents participate in Tinkabot without speaking raw NATS by default.

## Components

| ID | Name | Category | Status | Goal Contribution |
| --- | --- | --- | --- | --- |
| c3-301 | Tinkalet CLI |  | active | Provide profile-aware product commands. |
| c3-302 | Tinkalet Reactions and Coordination |  | active | Bridge durable items to local action. |

## Responsibilities

This container owns Tinkalet profiles, product commands, item/wait/watch flows, server-owned schedule editing, local reaction registration, local reaction execution, cursor files, and privacy-preserving CLI output.

## Complexity Assessment

Tinkalet straddles local host authority and server authority. It must preserve the boundary: Tinkabot owns durable truth and leases; Tinkalet owns local edge behavior only inside explicit profile and command scope.
