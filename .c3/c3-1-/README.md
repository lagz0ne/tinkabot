---
id: c3-1
c3-seal: b4442c3d2c8aa5ec5559186a9bc5d5ec9a1e99a4e0258024083be7ab97efef0b
title: Contracts and SDK
type: container
parent: c3-0
goal: Own the neutral contract surface that Go, TypeScript, browser, script, and release proofs derive from.
---

## Goal

Own the neutral contract surface that Go, TypeScript, browser, script, and release proofs derive from.

## Components

| ID | Name | Category | Status | Goal Contribution |
| --- | --- | --- | --- | --- |
| c3-101 | Schema Registry |  | active | Own the neutral wire contract. |
| c3-102 | TypeScript SDK |  | active | Expose checked TypeScript contract helpers. |

## Responsibilities

This container owns JSON Schema, contract fixtures, TypeScript SDK validators/types, SDK package build outputs, and parity tests that prevent runtime lanes from inventing local protocol truth.

## Complexity Assessment

Schema and SDK code are allowed to be more explicit at public wire boundaries. Generated or built outputs must stay traceable to their sources, but they are not separate architecture authority.
