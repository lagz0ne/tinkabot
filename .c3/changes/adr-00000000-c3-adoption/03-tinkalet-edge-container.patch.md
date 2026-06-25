---
target: c3-3
scope: whole
type: container
parent: c3-0
title: Tinkalet Edge
boundary: profile-aware CLI and local edge participant
---
## Goal

Let humans, scripts, CI jobs, and local agents participate in Tinkabot without speaking raw NATS by default.

## Components

| ID | Name | Category | Status | Goal Contribution |
| --- | --- | --- | --- | --- |

## Responsibilities

This container owns Tinkalet profiles, product commands, item/wait/watch flows, server-owned schedule editing, local reaction registration, local reaction execution, cursor files, and privacy-preserving CLI output.

## Complexity Assessment

Tinkalet straddles local host authority and server authority. It must preserve the boundary: Tinkabot owns durable truth and leases; Tinkalet owns local edge behavior only inside explicit profile and command scope.
