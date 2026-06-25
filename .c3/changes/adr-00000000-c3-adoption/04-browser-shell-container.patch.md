---
target: c3-4
scope: whole
type: container
parent: c3-0
title: Browser Shell
boundary: Vite trusted shell and embedded browser assets
---
## Goal

Provide the trusted browser-side substrate that renders generated artifacts, observes materialized state, and mediates browser intents without exposing raw NATS authority to generated content.

## Components

| ID | Name | Category | Status | Goal Contribution |
| --- | --- | --- | --- | --- |

## Responsibilities

This container owns frontend isolation helpers, trusted shell assets, session observation over browser NATS WebSocket, service-worker/browser tests, and the Go embedded frontend file system.

## Complexity Assessment

The shell is proof infrastructure today, not a polished product UI. Its architecture constraint still matters: generated UI is sandboxed receiver code and shell-owned mediation decides what can observe or command.
