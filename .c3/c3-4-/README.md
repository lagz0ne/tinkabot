---
id: c3-4
c3-seal: b439d6128bfb2680a5eb1bbd6f42c597cbf58addf707aae31c75bb746e84274c
title: Browser Shell
type: container
parent: c3-0
goal: Provide the trusted browser-side substrate that renders generated artifacts, observes materialized state, and mediates browser intents without exposing raw NATS authority to generated content.
---

## Goal

Provide the trusted browser-side substrate that renders generated artifacts, observes materialized state, and mediates browser intents without exposing raw NATS authority to generated content.

## Components

| ID | Name | Category | Status | Goal Contribution |
| --- | --- | --- | --- | --- |
| c3-401 | Trusted Frontend Shell |  | active | Isolate generated browser content. |
| c3-402 | Browser Observation |  | active | Observe sessions through scoped browser grants. |

## Responsibilities

This container owns frontend isolation helpers, trusted shell assets, session observation over browser NATS WebSocket, service-worker/browser tests, and the Go embedded frontend file system.

## Complexity Assessment

The shell is proof infrastructure today, not a polished product UI. Its architecture constraint still matters: generated UI is sandboxed receiver code and shell-owned mediation decides what can observe or command.
