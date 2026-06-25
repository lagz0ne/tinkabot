---
target: c3-1
scope: whole
type: container
parent: c3-0
title: Contracts and SDK
boundary: schemas, generated fixtures, and TypeScript SDK package
---
## Goal

Own the neutral contract surface that Go, TypeScript, browser, script, and release proofs derive from.

## Components

| ID | Name | Category | Status | Goal Contribution |
| --- | --- | --- | --- | --- |

## Responsibilities

This container owns JSON Schema, contract fixtures, TypeScript SDK validators/types, SDK package build outputs, and parity tests that prevent runtime lanes from inventing local protocol truth.

## Complexity Assessment

Schema and SDK code are allowed to be more explicit at public wire boundaries. Generated or built outputs must stay traceable to their sources, but they are not separate architecture authority.
