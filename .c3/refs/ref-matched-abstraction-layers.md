---
id: ref-matched-abstraction-layers
c3-seal: f213144a004df58d03a66b957961ff7ec609329850813a9a80d588bc3e1cf4b0
title: Matched Abstraction Layers
type: ref
parent: c3-0
goal: Standardize how architecture intent, decomposition, executable proof, and release evidence stay separated.
---

## Goal

Standardize how architecture intent, decomposition, executable proof, and release evidence stay separated.

## Choice

Use matched-abstraction layers: Approach owns intent/invariants, Plan owns decomposition/verification strategy, Task owns one executable proof with RED and verification evidence, and release/manual docs are proof surfaces rather than hidden design authority.

## Why

This repo already contains deep docs. Without a layer contract, later work can treat a Task transcript, README paragraph, or release manifest as if it overrode product authority. The layer split prevents that drift.

## How

Required pattern from `docs/matched-abstraction/approach/tinkalet-edge.md`, `docs/matched-abstraction/plan/tinkalet-edge.md`, and `docs/matched-abstraction/task/tinkalet-release-docs-and-proof.md`: lower layers cite higher layers and do not redefine product roles.
