---
target: c3-301
scope: whole
type: component
parent: c3-3
title: Tinkalet CLI
category: foundation
---
## Goal

Provide profile-aware product commands.

## Parent Fit

| Field | Value |
| --- | --- |
| Parent | c3-3 Tinkalet Edge |
| Role | Human/script/CI CLI surface over profiles, triggers, items, watches, schedules, and reactions |
| Owns | `substrate/go/cmd/tinkalet/**` and `substrate/go/tinkalet/**` |
| Does not own | Server-side durable item truth or scheduler authority |

## Purpose

Tinkalet CLI gives local users and agents a product vocabulary over Tinkabot: profile import/use/list, trigger, item create/get/resolve/wait, watch, daemon watch, schedule set/off, reaction add, and daemon react. It stores profile-local data under explicit config/data roots and avoids leaking raw credentials, subjects, or private NATS errors.

## Governance

| Reference | Type | Governs | Precedence | Notes |
| --- | --- | --- | --- | --- |
| ref-shadow-authority-boundaries | ref | Edge commands must use scoped profile authority and privacy-preserving denials | Tinkabot leases outrank CLI convenience | Tinkalet cannot bypass server authority. |

## Contract

| Surface | Direction | Contract | Boundary | Evidence |
| --- | --- | --- | --- | --- |
| CLI grammar | IN | Commands and flags in `help()` are the user-facing product grammar. | User/tool to Tinkalet | `substrate/go/tinkalet/tinkalet.go` |
| Profile store | IN/OUT | Profiles copy managed caller creds and select a default target. | Local disk to NATS server | `substrate/go/tinkalet/tinkalet_test.go` |

## Derived Materials

| Material | Must derive from | Allowed variance | Evidence |
| --- | --- | --- | --- |
| README/manual Tinkalet tour | Goal, Parent Fit, Contract, and Derived Materials | Paths and request ids only | `README.md`, `docs/manual/v1.md`, `scripts/smoke-tinkalet-package.sh` |
