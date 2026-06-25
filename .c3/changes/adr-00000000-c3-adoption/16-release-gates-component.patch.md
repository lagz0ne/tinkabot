---
target: c3-501
scope: whole
type: component
parent: c3-5
title: Release Gates and Packaging
category: foundation
---
## Goal

Prove package and release claims.

## Parent Fit

| Field | Value |
| --- | --- |
| Parent | c3-5 Proof Docs and Workflow |
| Role | Root scripts, package/archive builders, release evidence, gate checkers, and top-level tests |
| Owns | `package.json`, `scripts/**`, `release/**`, root `tests/**`, `bun.lock`, `skills-lock.json` |
| Does not own | Individual runtime features beyond checking their published evidence |

## Purpose

Release Gates and Packaging turns implementation into reproducible release-shaped proof. It builds local archives, runs manual/package/evidence/scenario/fake/coverage gates, validates release manifests, and keeps root tests tied to documented commands.

## Governance

| Reference | Type | Governs | Precedence | Notes |
| --- | --- | --- | --- | --- |
| ref-coverage-ratchet | ref | Coverage checks and release evidence | Mechanical evidence outranks prose claims | A release claim must cite commands or artifacts. |
| ref-matched-abstraction-layers | ref | Whether evidence is a Task proof or user-facing manual claim | Task evidence and release manifest must agree | Docs are proof surfaces, not hidden authority. |

## Contract

| Surface | Direction | Contract | Boundary | Evidence |
| --- | --- | --- | --- | --- |
| Package scripts | IN/OUT | Named package scripts run build, test, release, and gate workflows. | Repo operator to proof automation | `package.json`, `scripts/*.ts`, `scripts/*.sh` |
| Release manifest | OUT | `release/v1.json` records milestones, gates, manual evidence, and deferred scope. | Proof automation to release claim | release/v1.json |

## Derived Materials

| Material | Must derive from | Allowed variance | Evidence |
| --- | --- | --- | --- |
| GitHub release archive | Goal, Parent Fit, Contract, and Derived Materials | Version and output directory | `bun run release:package dist/release` |
