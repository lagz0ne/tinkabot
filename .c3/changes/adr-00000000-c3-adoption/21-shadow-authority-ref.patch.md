---
target: ref-shadow-authority-boundaries
scope: whole
type: ref
parent: c3-0
title: Shadow Authority Boundaries
---
## Goal

Standardize how frontend and backend integration inherit permission through Tinkabot without generated scripts or generated UI handling raw authority.

## Choice

Generated code receives mediated process, frame, profile, or material surfaces; Tinkabot and the trusted shell derive and enforce NATS authority, subjects, credentials, imports, exports, leases, and write paths.

## Why

The user's desired app-building flow lets an LLM progressively register/build an app. If generated code has to declare or hold raw NATS permissions, app construction becomes unsafe, hard to review, and impossible to keep shadow-integrated across frontend and backend.

## How

Required pattern from `substrate/go/tinkabot/bundle.go`: `bundleScript` declares no authority; names are derived under the bundle namespace. Required pattern from `apps/frontend/src/isolation.ts`: generated frame messages are parsed, raw authority words are denied, and accepted intents are wrapped with server-owned lease context.
